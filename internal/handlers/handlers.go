package handlers

import (
	"ws-json-rpc/pkg/rpc"
)

// -32768 to -32000	Reserved - Do not use (reserved for pre-defined errors)
// -31999 to -1	Recommended for general application errors
// 1 to 999	Recommended for validation and input errors
// 1000 to 4999	Recommended for business logic errors
// 5000+	Recommended for system or infrastructure errors
type HandlerErrorCode int

const (
	HandlerErrorCodeNotImplemented HandlerErrorCode = -1
	HandlerErrorCodeNotFound       HandlerErrorCode = -2
	HandlerErrorCodeInternal       HandlerErrorCode = -3
)

type HandlerError struct {
	code    HandlerErrorCode
	message string
}

func (e *HandlerError) Error() string {
	return e.message
}

func (e *HandlerError) Code() int {
	return int(e.code)
}

type Handlers struct {
	hub *rpc.Hub
}

func NewHandlers(hub *rpc.Hub) *Handlers {
	return &Handlers{hub: hub}
}

type UserUpdateEventResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserLoginEventResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
