package handlers

import (
	"context"
	"ws-json-rpc/pkg/ws"
)

type Status string

const (
	StatusOK       Status = "ok"
	StatusNotFound Status = "not_found"
	StatusError    Status = "error"
)

type PingResult struct {
	Message string `json:"message"`
	Status  Status `json:"status"`
}

func (h *Handlers) Ping(ctx context.Context, hctx *ws.HandlerContext, params struct{}) (PingResult, error) {
	hctx.Logger.Info("Ping received")
	return PingResult{}, &HandlerError{code: HandlerErrorCodeNotImplemented, message: "not implemented"}
	// return PingResult{Message: "pong", Status: StatusOK}, nil
}
