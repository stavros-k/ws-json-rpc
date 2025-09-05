package main

import (
	"context"
	"fmt"
	"ws-json-rpc/pkg/ws"
)

type Handlers struct {
	hub *ws.Hub
}

type UnsubscriptionParams struct {
	Event EventKind `json:"event"`
}

func (h *Handlers) Unsubscribe(ctx context.Context, params UnsubscriptionParams) (map[string]bool, error) {
	client, ok := ws.ClientFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no client found")
	}

	h.hub.Unsubscribe(client, params.Event)
	return map[string]bool{"unsubscribed": true}, nil
}

type SubscriptionParams struct {
	Event EventKind `json:"event"`
}

func (h *Handlers) Subscribe(ctx context.Context, params SubscriptionParams) (map[string]bool, error) {
	client, ok := ws.ClientFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no client found")
	}

	// Handler that needs hub access
	if err := h.hub.Subscribe(client, params.Event); err != nil {
		return nil, err
	}

	return map[string]bool{"subscribed": true}, nil
}

type EchoParams struct {
	Message string `json:"message"`
}

type EchoResult struct {
	Echo string `json:"echo"`
}

func (h *Handlers) Echo(ctx context.Context, params EchoParams) (EchoResult, error) {
	return EchoResult{Echo: params.Message}, nil
}

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

type DoubleParams struct {
	Value int `json:"value"`
}

func (h *Handlers) Double(ctx context.Context, params DoubleParams) (AddResult, error) {
	// Direct method call
	return h.Add(ctx, AddParams{A: params.Value, B: params.Value})
}

func (h *Handlers) Ping(ctx context.Context, params struct{}) (string, error) {
	return "pong", nil
}
