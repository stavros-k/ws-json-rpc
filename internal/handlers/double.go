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

type DoubleResult struct {
	Result int `json:"result"`
}

func (h *Handlers) Double(ctx context.Context, hctx *rpc.HandlerContext, params DoubleParams) (DoubleResult, error) {
	res, err := h.Add(ctx, hctx, AddParams{A: params.Value, B: params.Other})
	if err != nil {
		return DoubleResult{}, err
	}

	return DoubleResult{Result: res.Result * 2}, nil
}
