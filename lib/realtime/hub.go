package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/coder/websocket"
)

var ErrTooManyClients = errors.New("too many websocket clients")

type Client struct {
	network string
	conn    *websocket.Conn
	send    chan []byte
}

type Hub struct {
	maxClients int
	mu         sync.RWMutex
	clients    map[*Client]struct{}
}

func NewHub(maxClients int) *Hub {
	if maxClients <= 0 {
		maxClients = 500
	}
	return &Hub{
		maxClients: maxClients,
		clients:    make(map[*Client]struct{}),
	}
}

func (h *Hub) Register(ctx context.Context, network string, conn *websocket.Conn) (*Client, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.clients) >= h.maxClients {
		return nil, ErrTooManyClients
	}
	c := &Client{
		network: network,
		conn:    conn,
		send:    make(chan []byte, 32),
	}
	h.clients[c] = struct{}{}
	go h.writePump(ctx, c)
	return c, nil
}

func (h *Hub) Unregister(c *Client) {
	if c == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	close(c.send)
}

func (h *Hub) Broadcast(network, msgType string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg, err := EncodeClientMessage(ClientMessage{
		Type:    msgType,
		Network: network,
		Payload: raw,
	})
	if err != nil {
		return err
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.network != network {
			continue
		}
		select {
		case c.send <- msg:
		default:
		}
	}
	return nil
}

func (h *Hub) SendTo(c *Client, network, msgType string, payload any) error {
	if c == nil {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg, err := EncodeClientMessage(ClientMessage{
		Type:    msgType,
		Network: network,
		Payload: raw,
	})
	if err != nil {
		return err
	}
	select {
	case c.send <- msg:
	default:
	}
	return nil
}

func (h *Hub) writePump(ctx context.Context, c *Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.Write(ctx, websocket.MessageText, msg); err != nil {
				return
			}
		}
	}
}

func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		delete(h.clients, c)
		close(c.send)
	}
}
