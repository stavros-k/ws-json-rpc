package handlers

import "context"

type EchoParams struct {
	Message string `json:"message"`
}

type EchoResult struct {
	Echo string `json:"echo"`
}

func (h *Handlers) Echo(ctx context.Context, params EchoParams) (EchoResult, error) {
	return EchoResult{Echo: params.Message}, nil
}
