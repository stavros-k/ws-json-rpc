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
	// Only WebSocket clients can unsubscribe from events
	if hctx.WSConn == nil {
		return UnsubscribeResult{}, ws.NewHandlerError(ws.ErrCodeInvalid, "Subscriptions are only available for WebSocket connections")
	}

	h.hub.Unsubscribe(hctx.WSConn, string(params.Event))
	return UnsubscribeResult{Unsubscribed: true}, nil
}

type SubscribeParams struct {
	Event consts.EventKind `json:"event"`
}
type SubscribeResult struct {
	Subscribed bool `json:"subscribed"`
}

func (h *Handlers) Subscribe(ctx context.Context, hctx *ws.HandlerContext, params SubscribeParams) (SubscribeResult, error) {
	// Only WebSocket clients can subscribe to events
	if hctx.WSConn == nil {
		return SubscribeResult{}, ws.NewHandlerError(ws.ErrCodeInvalid, "Subscriptions are only available for WebSocket connections")
	}

	// Handler that needs hub access
	if err := h.hub.Subscribe(hctx.WSConn, string(params.Event)); err != nil {
		return SubscribeResult{}, err
	}

	return SubscribeResult{Subscribed: true}, nil
}
