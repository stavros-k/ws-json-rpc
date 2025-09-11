package handlers

import (
	"context"
	"ws-json-rpc/internal/consts"
	"ws-json-rpc/pkg/ws"
)

type UnsubscribeParams struct {
	Event consts.EventKind `json:"event"`
}
type UnsubscribeResult struct {
	Unsubscribed bool `json:"unsubscribed"`
}

func (h *Handlers) Unsubscribe(ctx context.Context, hctx *ws.HandlerContext, params UnsubscribeParams) (UnsubscribeResult, error) {
	h.hub.Unsubscribe(hctx.Client, string(params.Event))
	return UnsubscribeResult{Unsubscribed: true}, nil
}

type SubscribeParams struct {
	Event consts.EventKind `json:"event"`
}
type SubscribeResult struct {
	Subscribed bool `json:"subscribed"`
}

func (h *Handlers) Subscribe(ctx context.Context, hctx *ws.HandlerContext, params SubscribeParams) (SubscribeResult, error) {
	// Handler that needs hub access
	if err := h.hub.Subscribe(hctx.Client, string(params.Event)); err != nil {
		return SubscribeResult{}, err
	}

	return SubscribeResult{Subscribed: true}, nil
}
