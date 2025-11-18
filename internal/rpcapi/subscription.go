package rpcapi

import (
	"context"
	"ws-json-rpc/pkg/rpc"
)

func (h *Handlers) Subscribe(ctx context.Context, hctx *rpc.HandlerContext, params SubscribeParams) (SubscribeResult, error) {
	if hctx.WSConn == nil {
		return SubscribeResult{}, rpc.NewHandlerError(rpc.ErrCodeInvalid, "Subscriptions are only available for WebSocket connections")
	}

	if err := h.hub.Subscribe(hctx.WSConn, string(params.Event)); err != nil {
		return SubscribeResult{}, err
	}

	return SubscribeResult{Success: true}, nil
}

func (h *Handlers) Unsubscribe(ctx context.Context, hctx *rpc.HandlerContext, params UnsubscribeParams) (UnsubscribeResult, error) {
	if hctx.WSConn == nil {
		return UnsubscribeResult{}, rpc.NewHandlerError(rpc.ErrCodeInvalid, "Unsubscriptions are only available for WebSocket connections")
	}

	h.hub.Unsubscribe(hctx.WSConn, string(params.Event))
	return UnsubscribeResult{Success: true}, nil
}
