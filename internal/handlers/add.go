package handlers

import (
	"context"
	"time"
	"ws-json-rpc/pkg/ws"
)

type JSONTime struct {
	time.Time
}
type AddParams struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddResult struct {
	Result int      `json:"result"`
	Time   JSONTime `json:"time"`
}

func (h *Handlers) Add(ctx context.Context, hctx *ws.HandlerContext, params AddParams) (AddResult, error) {
	return AddResult{Result: params.A + params.B}, nil
}
