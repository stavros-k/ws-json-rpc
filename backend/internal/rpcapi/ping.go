package rpcapi

import (
	"context"
	rpctypes "ws-json-rpc/backend/internal/rpcapi/types"
	"ws-json-rpc/backend/pkg/rpc"
)

func (h *Handlers) PingHandler(ctx context.Context, hctx *rpc.HandlerContext, params struct{}) (rpctypes.PingResult, error) {
	hctx.Logger.Debug("PingHandler called")

	return rpctypes.PingResult{Message: "pong", Status: rpctypes.PingStatusSuccess}, nil
}
