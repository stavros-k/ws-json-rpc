package rpcapi

import "ws-json-rpc/backend/pkg/rpc"

type Handlers struct {
	hub *rpc.Hub
}

func NewHandlers(hub *rpc.Hub) *Handlers {
	return &Handlers{hub: hub}
}
