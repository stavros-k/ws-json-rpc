package handlers

import (
	"context"
	"ws-json-rpc/pkg/ws"
)

type DoubleParams struct {
	Value int `json:"value"`
}

func (h *Handlers) Double(ctx context.Context, hctx *ws.HandlerContext, params DoubleParams) (AddResult, error) {
	// Direct method call
	return h.Add(ctx, hctx, AddParams{A: params.Value, B: params.Value})
}
