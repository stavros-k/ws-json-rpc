package handlers

import (
	"context"
	"ws-json-rpc/pkg/ws"
)

// Status represents the status of a ping response
type Status string

const (
	//Sent when the ping is successful
	StatusOK Status = "OK"
	// This is the default status, so it is not necessary to specify it
	StatusNotFound Status = "NotFound"
	// Sent when there is an error processing the ping
	StatusError Status = "Error"
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
