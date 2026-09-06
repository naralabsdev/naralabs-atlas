package rpcchain

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

type Endpoint struct {
	URL   string
	Label string
}

type endpointState struct {
	endpoint Endpoint
	errors   atomic.Int32
	downUntil atomic.Int64
}

// Pool rotates across RPC endpoints with passive health tracking.
type Pool struct {
	endpoints []*endpointState
	active    atomic.Int32
	log       *slog.Logger
}

func NewPool(endpoints []Endpoint, log *slog.Logger) (*Pool, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("rpcchain: at least one endpoint required")
	}
	if log == nil {
		log = slog.Default()
	}
	states := make([]*endpointState, 0, len(endpoints))
	for _, ep := range endpoints {
		states = append(states, &endpointState{endpoint: ep})
	}
	p := &Pool{endpoints: states, log: log}
	p.active.Store(-1)
	return p, nil
}

func (p *Pool) Call(ctx context.Context, fn func(ctx context.Context, endpoint Endpoint) error) error {
	var lastErr error
	for range len(p.endpoints) {
		idx := p.pick()
		state := p.endpoints[idx]
		if until := time.Unix(0, state.downUntil.Load()); !until.IsZero() && time.Now().Before(until) {
			lastErr = ErrAllEndpointsDown
			continue
		}

		err := fn(ctx, state.endpoint)
		if err == nil {
			state.errors.Store(0)
			p.active.Store(int32(idx))
			return nil
		}

		lastErr = err
		count := state.errors.Add(1)
		if count >= 3 {
			state.downUntil.Store(time.Now().Add(30 * time.Second).UnixNano())
			p.log.Warn("rpc endpoint marked down",
				"endpoint", state.endpoint.Label,
				"error", err,
			)
		}
	}
	if lastErr == nil {
		lastErr = ErrAllEndpointsDown
	}
	return lastErr
}

// ActiveIndex returns the index of the last successful endpoint.
func (p *Pool) ActiveIndex() int {
	return int(p.active.Load())
}

func (p *Pool) pick() int {
	now := time.Now()
	active := int(p.active.Load())
	if active >= 0 {
		state := p.endpoints[active]
		until := time.Unix(0, state.downUntil.Load())
		if until.IsZero() || now.After(until) {
			return active
		}
	}
	for i, state := range p.endpoints {
		until := time.Unix(0, state.downUntil.Load())
		if until.IsZero() || now.After(until) {
			return i
		}
	}
	return 0
}
