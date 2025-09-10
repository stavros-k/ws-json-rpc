package handlers

import (
	"context"
	"fmt"
	"ws-json-rpc/internal/consts"
	"ws-json-rpc/pkg/ws"
)

type UnsubscribeParams struct {
	Event consts.EventKind `json:"event"`
}
type UnsubscribeResult struct {
	Unsubscribed bool `json:"unsubscribed"`
}

func (h *Handlers) Unsubscribe(ctx context.Context, params UnsubscribeParams) (UnsubscribeResult, error) {
	clientContext, ok := ws.ClientContextFromContext(ctx)
	if !ok {
		return UnsubscribeResult{}, fmt.Errorf("no client found")
	}

	h.hub.Unsubscribe(clientContext.Client, string(params.Event))
	return UnsubscribeResult{Unsubscribed: true}, nil
}

type SubscribeParams struct {
	Event consts.EventKind `json:"event"`
}
type SubscribeResult struct {
	Subscribed bool `json:"subscribed"`
}

func (h *Handlers) Subscribe(ctx context.Context, params SubscribeParams) (SubscribeResult, error) {
	clientContext, ok := ws.ClientContextFromContext(ctx)
	if !ok {
		return SubscribeResult{}, fmt.Errorf("no client found")
	}

	// Handler that needs hub access
	if err := h.hub.Subscribe(clientContext.Client, string(params.Event)); err != nil {
		return SubscribeResult{}, err
	}

	return SubscribeResult{Subscribed: true}, nil
}
