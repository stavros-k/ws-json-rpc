package handlers

import (
	"context"
	"ws-json-rpc/pkg/ws"
)

type EchoParams struct {
	Message string `json:"message"`
}

type EchoResult struct {
	Echo string `json:"echo"`
}

func (h *Handlers) Echo(ctx context.Context, hctx *ws.HandlerContext, params EchoParams) (EchoResult, error) {
	return EchoResult{Echo: params.Message}, nil
}
