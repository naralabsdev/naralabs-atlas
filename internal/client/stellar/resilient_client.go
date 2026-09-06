package stellar

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	"github.com/stellar/go-stellar-sdk/clients/rpcclient"
	rpc "github.com/stellar/go-stellar-sdk/protocols/rpc"

	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/lib/rpcchain"
)

type ResilientClient struct {
	pool     *rpcchain.Pool
	throttle *rpcchain.Throttle
	policy   rpcchain.RetryPolicy
	log      *slog.Logger

	mu      sync.Mutex
	clients map[string]*rpcclient.Client
}

func NewResilientClient(cfg *config.Config, log *slog.Logger) (Client, error) {
	urls := cfg.RPCURLs()
	endpoints := make([]rpcchain.Endpoint, 0, len(urls))
	for _, raw := range urls {
		endpoints = append(endpoints, rpcchain.Endpoint{
			URL:   raw,
			Label: endpointLabel(raw),
		})
	}
	pool, err := rpcchain.NewPool(endpoints, log)
	if err != nil {
		return nil, err
	}
	return &ResilientClient{
		pool:     pool,
		throttle: rpcchain.NewThrottle(cfg.RPC.RequestsPerSec),
		policy: rpcchain.RetryPolicy{
			MaxAttempts:   cfg.RPC.MaxAttempts,
			BaseBackoff:   cfg.RPC.BaseBackoff,
			MaxBackoff:    cfg.RPC.MaxBackoff,
			MaxRetryAfter: cfg.RPC.MaxRetryAfter,
			Jitter:        true,
		},
		log:     log,
		clients: make(map[string]*rpcclient.Client),
	}, nil
}

func endpointLabel(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Host
}

func (c *ResilientClient) clientFor(url string) *rpcclient.Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	if client, ok := c.clients[url]; ok {
		return client
	}
	client := rpcclient.NewClient(url, nil)
	c.clients[url] = client
	return client
}

func (c *ResilientClient) LatestLedger(ctx context.Context) (uint32, error) {
	var sequence uint32
	err := c.call(ctx, func(ctx context.Context, endpoint rpcchain.Endpoint) error {
		ledger, err := c.clientFor(endpoint.URL).GetLatestLedger(ctx)
		if err != nil {
			return rpcchain.Retryable(err)
		}
		sequence = ledger.Sequence
		return nil
	})
	return sequence, err
}

func (c *ResilientClient) FetchEvents(
	ctx context.Context,
	network string,
	input FetchEventsInput,
) ([]ContractEvent, uint32, error) {
	startEndpoint := c.pool.ActiveIndex()
	allEvents := make([]ContractEvent, 0, input.Limit)
	cursor := input.Cursor
	startLedger := input.StartLedger

	for {
		pageInput := input
		pageInput.Network = network
		pageInput.StartLedger = startLedger
		pageInput.Cursor = cursor
		if pageInput.Limit == 0 {
			pageInput.Limit = 1000
		}

		var page []ContractEvent
		var nextCursor string
		var latestLedger uint32
		err := c.call(ctx, func(ctx context.Context, endpoint rpcchain.Endpoint) error {
			if startEndpoint >= 0 && cursor != "" && c.pool.ActiveIndex() >= 0 && c.pool.ActiveIndex() != startEndpoint {
				return rpcchain.ErrReanchorCursor
			}
			resp, err := c.fetchPage(ctx, endpoint.URL, pageInput)
			if err != nil {
				return err
			}
			page = resp.events
			nextCursor = resp.cursor
			latestLedger = resp.latestLedger
			return nil
		})
		if rpcchain.IsReanchorCursor(err) {
			return nil, startLedger, err
		}
		if err != nil {
			return nil, startLedger, err
		}

		allEvents = append(allEvents, page...)

		nextLedger := startLedger
		if nextCursor != "" {
			if parsed, parseErr := parseCursorLedger(nextCursor); parseErr == nil && parsed > nextLedger {
				nextLedger = parsed
			}
		}
		if latestLedger > nextLedger {
			nextLedger = latestLedger
		}

		if nextCursor == "" || len(page) == 0 {
			return allEvents, nextLedger, nil
		}
		if input.Limit > 0 && uint32(len(allEvents)) >= input.Limit {
			return allEvents, nextLedger, nil
		}
		if input.EndLedger > 0 && nextLedger >= input.EndLedger {
			return allEvents, nextLedger, nil
		}

		cursor = nextCursor
		startLedger = nextLedger
	}
}

type pageResult struct {
	events       []ContractEvent
	cursor       string
	latestLedger uint32
}

func (c *ResilientClient) fetchPage(ctx context.Context, rpcURL string, input FetchEventsInput) (pageResult, error) {
	pagination := &rpc.PaginationOptions{
		Limit: uint(input.Limit),
	}
	if input.Cursor != "" {
		cursor, err := rpc.ParseCursor(input.Cursor)
		if err != nil {
			return pageResult{}, fmt.Errorf("parse rpc cursor: %w", err)
		}
		pagination.Cursor = &cursor
	}

	request := rpc.GetEventsRequest{
		StartLedger: input.StartLedger,
		Pagination:  pagination,
		Filters:     buildEventFilters(input.ContractIDs),
	}

	response, err := c.clientFor(rpcURL).GetEvents(ctx, request)
	if err != nil {
		return pageResult{}, rpcchain.Retryable(err)
	}

	events := make([]ContractEvent, 0, len(response.Events))
	for _, event := range response.Events {
		events = append(events, mapEvent(input.Network, event))
	}

	return pageResult{
		events:       events,
		cursor:       response.Cursor,
		latestLedger: response.LatestLedger,
	}, nil
}

func (c *ResilientClient) call(ctx context.Context, fn func(context.Context, rpcchain.Endpoint) error) error {
	return rpcchain.Do(ctx, c.policy, func(ctx context.Context) error {
		if err := c.throttle.Wait(ctx); err != nil {
			return err
		}
		return c.pool.Call(ctx, fn)
	})
}

func BuildFilterBatches(contractIDs []string, batchSize int) [][]string {
	if batchSize <= 0 {
		batchSize = 25
	}
	if len(contractIDs) == 0 {
		return [][]string{nil}
	}
	batches := make([][]string, 0, (len(contractIDs)+batchSize-1)/batchSize)
	for i := 0; i < len(contractIDs); i += batchSize {
		end := i + batchSize
		if end > len(contractIDs) {
			end = len(contractIDs)
		}
		batches = append(batches, append([]string(nil), contractIDs[i:end]...))
	}
	return batches
}

func FilterBatchLabel(batch []string) string {
	if len(batch) == 0 {
		return "all-contracts"
	}
	if len(batch) == 1 {
		return batch[0]
	}
	return fmt.Sprintf("%s+%d", batch[0], len(batch)-1)
}

func ParseContractIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
