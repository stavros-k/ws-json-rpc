package rpcapi

import (
	"context"
	rpctypes "ws-json-rpc/internal/rpcapi/types"
	"ws-json-rpc/pkg/rpc"
)

func (h *Handlers) Subscribe(ctx context.Context, hctx *rpc.HandlerContext, params rpctypes.SubscribeParams) (rpctypes.SubscribeResult, error) {
	if hctx.WSConn == nil {
		return rpctypes.SubscribeResult{}, rpc.NewHandlerError(rpc.ErrCodeInvalid, "Subscriptions are only available for WebSocket connections")
	}

	if err := h.hub.Subscribe(hctx.WSConn, string(params.Event)); err != nil {
		return rpctypes.SubscribeResult{}, err
	}

	return rpctypes.SubscribeResult{Success: true}, nil
}

func (h *Handlers) Unsubscribe(ctx context.Context, hctx *rpc.HandlerContext, params rpctypes.UnsubscribeParams) (rpctypes.UnsubscribeResult, error) {
	if hctx.WSConn == nil {
		return rpctypes.UnsubscribeResult{}, rpc.NewHandlerError(rpc.ErrCodeInvalid, "Unsubscriptions are only available for WebSocket connections")
	}

	h.hub.Unsubscribe(hctx.WSConn, string(params.Event))
	return rpctypes.UnsubscribeResult{Success: true}, nil
}
