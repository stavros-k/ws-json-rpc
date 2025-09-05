package handlers

import (
	"context"
)

type DoubleParams struct {
	Value int `json:"value"`
}

func (h *Handlers) Double(ctx context.Context, params DoubleParams) (AddResult, error) {
	// Direct method call
	return h.Add(ctx, AddParams{A: params.Value, B: params.Value})
}
