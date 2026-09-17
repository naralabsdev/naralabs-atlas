package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/model"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/repository"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/service"
	"github.com/naralabs/naralabs-atlas/lib/realtime"
)

type HomeWebSocketHandler struct {
	cfg           *config.Config
	svc           *service.ExploreService
	hub           *realtime.Hub
	defaultNet    string
	recentLimit   int
	contractLimit int
	log           *slog.Logger

	debounceMu sync.Mutex
	debouncers map[string]*statsDebouncer
}

type statsDebouncer struct {
	timer *time.Timer
}

func NewHomeWebSocketHandler(
	cfg *config.Config,
	svc *service.ExploreService,
	hub *realtime.Hub,
	log *slog.Logger,
) *HomeWebSocketHandler {
	if log == nil {
		log = slog.Default()
	}
	return &HomeWebSocketHandler{
		cfg:           cfg,
		svc:           svc,
		hub:           hub,
		defaultNet:    cfg.Stellar.Network,
		recentLimit:   8,
		contractLimit: 8,
		log:           log,
		debouncers:    make(map[string]*statsDebouncer),
	}
}

func (h *HomeWebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.hub == nil || h.cfg == nil || !h.cfg.Realtime.Enabled {
		http.Error(w, "realtime disabled", http.StatusServiceUnavailable)
		return
	}

	network := strings.TrimSpace(r.URL.Query().Get("network"))
	if network == "" {
		network = h.defaultNet
	}

	opts := &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	}
	if origins := h.cfg.RealtimeOriginList(); len(origins) > 0 {
		opts.InsecureSkipVerify = false
		opts.OriginPatterns = origins
	}

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		h.log.Warn("websocket accept failed", "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "closed")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	client, err := h.hub.Register(ctx, network, conn)
	if err != nil {
		if errors.Is(err, realtime.ErrTooManyClients) {
			conn.Close(websocket.StatusPolicyViolation, "too many clients")
			return
		}
		conn.Close(websocket.StatusInternalError, "register failed")
		return
	}
	defer h.hub.Unregister(client)

	if err := h.sendSnapshot(ctx, client, network); err != nil {
		h.log.Warn("websocket snapshot failed", "error", err, "network", network)
		return
	}

	go h.pingLoop(ctx, client, network)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg realtime.ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type == realtime.MsgPong {
			continue
		}
	}
}

func (h *HomeWebSocketHandler) HandleIngest(ctx context.Context, ingest realtime.IngestMessage) error {
	if h.hub == nil || len(ingest.Events) == 0 {
		return nil
	}

	items := make([]model.EventItem, 0, len(ingest.Events))
	for _, event := range ingest.Events {
		items = append(items, repository.MapStoredEvent(
			event.ID,
			event.ContractID,
			event.TxnHash,
			event.Ledger,
			event.TopicsJSON,
			event.ValueJSON,
			event.SemanticDecoded,
			event.IngestedAt,
		))
	}

	if err := h.hub.Broadcast(ingest.Network, realtime.MsgHomeEventsIngested, items); err != nil {
		return err
	}
	h.scheduleStatsRefresh(ctx, ingest.Network)
	return nil
}

func (h *HomeWebSocketHandler) scheduleStatsRefresh(ctx context.Context, network string) {
	debounce := time.Duration(h.cfg.Realtime.StatsDebounceMs) * time.Millisecond
	if debounce <= 0 {
		debounce = 500 * time.Millisecond
	}

	h.debounceMu.Lock()
	defer h.debounceMu.Unlock()

	d, ok := h.debouncers[network]
	if !ok {
		d = &statsDebouncer{}
		h.debouncers[network] = d
	}
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(debounce, func() {
		if err := h.refreshDerived(context.Background(), network); err != nil {
			h.log.Warn("websocket stats refresh failed", "error", err, "network", network)
		}
	})
}

func (h *HomeWebSocketHandler) refreshDerived(ctx context.Context, network string) error {
	stats, err := h.svc.GetStats(ctx, network)
	if err != nil {
		return err
	}
	if err := h.hub.Broadcast(network, realtime.MsgHomeStatsUpdated, stats); err != nil {
		return err
	}
	contracts, err := h.svc.ListActiveContracts(ctx, network, h.contractLimit)
	if err != nil {
		return err
	}
	return h.hub.Broadcast(network, realtime.MsgHomeContractsUpdated, contracts)
}

func (h *HomeWebSocketHandler) sendSnapshot(ctx context.Context, client *realtime.Client, network string) error {
	payload, err := h.svc.GetHome(ctx, network, h.recentLimit, h.contractLimit)
	if err != nil {
		return err
	}
	return h.hub.SendTo(client, network, realtime.MsgHomeSnapshot, payload)
}

func (h *HomeWebSocketHandler) pingLoop(ctx context.Context, client *realtime.Client, network string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = h.hub.SendTo(client, network, realtime.MsgPing, map[string]string{})
		}
	}
}
