package handlers

import "context"

type PingResult struct {
	Message string `json:"message"`
}

func (h *Handlers) Ping(ctx context.Context, params struct{}) (PingResult, error) {
	return PingResult{Message: "pong"}, nil
}
