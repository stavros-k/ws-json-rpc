package handlers

import (
	"context"
	"ws-json-rpc/pkg/rpc"
)

// Some other comment
type DoubleParams struct {
	Value int `json:"value"`
	Other int `json:"other"`
}

func (h *Handlers) Double(ctx context.Context, hctx *rpc.HandlerContext, params DoubleParams) (AddResult, error) {
	// Direct method call
	return h.Add(ctx, hctx, AddParams{A: params.Value, B: params.Value})
}
