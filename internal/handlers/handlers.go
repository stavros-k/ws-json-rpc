package handlers

import (
	"ws-json-rpc/pkg/ws"
)

type Handlers struct {
	hub *ws.Hub
}

func NewHandlers(hub *ws.Hub) *Handlers {
	return &Handlers{hub: hub}
}
