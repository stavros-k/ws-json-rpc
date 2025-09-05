package handlers

import "context"

type AddParams struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddResult struct {
	Result int `json:"result"`
}

func (h *Handlers) Add(ctx context.Context, params AddParams) (AddResult, error) {
	return AddResult{Result: params.A + params.B}, nil
}
