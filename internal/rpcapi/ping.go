package rpcapi

import (
	"context"
	"ws-json-rpc/pkg/rpc"
)

func (h *Handlers) PingHandler(ctx context.Context, hctx *rpc.HandlerContext, params struct{}) (PingResult, error) {
	hctx.Logger.Debug("PingHandler called")
	return PingResult{Message: "pong", Status: PingStatusSuccess}, nil
}
