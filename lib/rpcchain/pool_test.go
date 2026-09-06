package rpcchain

import (
	"context"
	"errors"
	"testing"
)

func TestNewPoolRequiresEndpoint(t *testing.T) {
	_, err := NewPool(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPoolCallSuccessSetsActive(t *testing.T) {
	pool, err := NewPool([]Endpoint{{URL: "https://a", Label: "a"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	err = pool.Call(context.Background(), func(ctx context.Context, endpoint Endpoint) error {
		called = true
		if endpoint.Label != "a" {
			t.Fatalf("label=%s", endpoint.Label)
		}
		return nil
	})
	if err != nil || !called {
		t.Fatalf("err=%v called=%v", err, called)
	}
	if pool.ActiveIndex() != 0 {
		t.Fatalf("active=%d", pool.ActiveIndex())
	}
}

func TestPoolFailoverToSecondEndpoint(t *testing.T) {
	pool, err := NewPool([]Endpoint{
		{URL: "https://a", Label: "a"},
		{URL: "https://b", Label: "b"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	attempts := 0
	err = pool.Call(context.Background(), func(ctx context.Context, endpoint Endpoint) error {
		attempts++
		if endpoint.Label == "a" {
			return errors.New("boom")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts < 2 {
		t.Fatalf("attempts=%d", attempts)
	}
	if pool.ActiveIndex() != 1 {
		t.Fatalf("active=%d", pool.ActiveIndex())
	}
}

func TestIsReanchorCursor(t *testing.T) {
	if !IsReanchorCursor(ErrReanchorCursor) {
		t.Fatal("expected reanchor")
	}
	if IsReanchorCursor(errors.New("other")) {
		t.Fatal("unexpected match")
	}
}
