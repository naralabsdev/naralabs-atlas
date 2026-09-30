package handler

import (
	"context"
	"testing"

	decoderservice "github.com/naralabs/naralabs-atlas/internal/module/decoder/service"
)

func TestPlaygroundBFFAuthorized(t *testing.T) {
	const token = "test-playground-secret"
	if !playgroundBFFAuthorized("Bearer "+token, token) {
		t.Fatal("expected bearer token to match")
	}
	if playgroundBFFAuthorized("Bearer wrong", token) {
		t.Fatal("expected mismatch")
	}
	if playgroundBFFAuthorized("", token) {
		t.Fatal("expected empty auth to fail")
	}
}

func TestHandlePlaygroundDecodeDisabled(t *testing.T) {
	h := NewDecodeHandler(decoderservice.NewDecodeService(nil), nil, "")
	_, err := h.HandlePlaygroundDecode(context.Background(), &PlaygroundDecodeInput{})
	if err == nil {
		t.Fatal("expected error when playground disabled")
	}
}
