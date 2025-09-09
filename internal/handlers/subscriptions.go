package handlers

import (
	"context"
	"fmt"
	"ws-json-rpc/internal/consts"
	"ws-json-rpc/pkg/ws"
)

type UnsubscriptionParams struct {
	Event consts.EventKind `json:"event"`
}

func (h *Handlers) Unsubscribe(ctx context.Context, params UnsubscriptionParams) (map[string]bool, error) {
	client, ok := ws.ClientFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no client found")
	}

	h.hub.Unsubscribe(client, string(params.Event))
	return map[string]bool{"unsubscribed": true}, nil
}

type SubscriptionParams struct {
	Event consts.EventKind `json:"event"`
}

func (h *Handlers) Subscribe(ctx context.Context, params SubscriptionParams) (map[string]bool, error) {
	client, ok := ws.ClientFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no client found")
	}

	// Handler that needs hub access
	if err := h.hub.Subscribe(client, string(params.Event)); err != nil {
		return nil, err
	}

	return map[string]bool{"subscribed": true}, nil
}
