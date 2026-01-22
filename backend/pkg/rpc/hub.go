package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"ws-json-rpc/backend/pkg/rpc/generate"
	"ws-json-rpc/backend/pkg/utils"

	"github.com/google/uuid"
)

const (
	MAX_QUEUED_EVENTS_PER_CLIENT = 256
	MAX_REQUEST_TIMEOUT          = 30 * time.Second
	MAX_RESPONSE_TIMEOUT         = 30 * time.Second
	MAX_SEND_CHANNEL_TIMEOUT     = 5 * time.Second
	MAX_MESSAGE_SIZE             = 1024 * 1024 // 1 MB
)

const (
	ErrCodeParse         = -32700 // Invalid JSON was received by the server. An error occurred on the server while parsing the JSON text.
	ErrCodeInvalid       = -32600 // The JSON sent is not a valid Request object.
	ErrCodeNotFound      = -32601 // The method does not exist / is not available.
	ErrCodeInvalidParams = -32602 // Invalid method parameter(s).
	ErrCodeInternal      = -32603 // Internal JSON-RPC error.
)

// RPCRequest represents an object from the client.
type RPCRequest struct {
	Version string          `json:"jsonrpc"`
	ID      uuid.UUID       `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCEvent represents an RPCEvent that can be broadcast to subscribers.
type RPCEvent struct {
	EventName string `json:"event"`
	Data      any    `json:"data"`
}

// NewEvent creates a new event.
func NewEvent(eventName string, data any) RPCEvent {
	return RPCEvent{EventName: eventName, Data: data}
}

type EventOptions struct {
	Docs generate.EventDocs
}

// RegisterEvent registers an event with the hub.
func RegisterEvent[TResult any](h *Hub, eventName string, options EventOptions) {
	var eventZero TResult
	h.generator.AddEventType(eventName, eventZero, options.Docs)
	h.registerEvent(eventName)
}

// RPCResponse represents a response from the server.
type RPCResponse struct {
	Version string          `json:"jsonrpc"`
	ID      uuid.UUID       `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCErrorObj    `json:"error,omitempty"`
}

// NewRPCResponse creates a new JSON-RPC 2.0 response. Result is marshaled internally.
func NewRPCResponse(id uuid.UUID, result any, err *RPCErrorObj) RPCResponse {
	// Marshal the result
	data, jsonErr := utils.ToJSON(result)
	if jsonErr != nil {
		RPCErrorObj := RPCErrorObj{Code: ErrCodeInternal, Message: "Failed to serialize response"}
		return RPCResponse{Version: "2.0", ID: id, Error: &RPCErrorObj}
	}

	return RPCResponse{Version: "2.0", ID: id, Result: data, Error: err}
}

// RPCErrorObj represents an error on a response.
type RPCErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// HandlerFunc is a function that handles a method call.
type HandlerFunc func(ctx context.Context, hctx *HandlerContext, params any) (any, error)

// TypedHandlerFunc is a function that handles a method call with typed parameters.
type TypedHandlerFunc[TParams any, TResult any] func(ctx context.Context, hctx *HandlerContext, params TParams) (TResult, error)

// MiddlewareFunc is a function that wraps a HandlerFunc with additional behavior.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Method represents a registered method in the hub.
type Method struct {
	// The actual handler function
	handler HandlerFunc
	// Parses the params into the appropriate type
	parser func(json.RawMessage) (any, error)
}

type RegisterMethodOptions struct {
	Middlewares []MiddlewareFunc
	Docs        generate.MethodDocs
}

// RegisterMethod registers a method with the hub.
func RegisterMethod[TParams any, TResult any](h *Hub, method string, handler TypedHandlerFunc[TParams, TResult], options RegisterMethodOptions) {
	wrapped := func(ctx context.Context, hctx *HandlerContext, params any) (any, error) {
		if params, ok := params.(TParams); ok {
			return handler(ctx, hctx, params)
		}
		return nil, fmt.Errorf("invalid params type: %T", params)
	}

	parser := func(rawParams json.RawMessage) (any, error) {
		return utils.FromJSON[TParams](rawParams)
	}

	// Apply global middlewares first (will be outermost)
	for i := len(h.middlewares) - 1; i >= 0; i-- {
		wrapped = h.middlewares[i](wrapped)
	}

	// Apply method-specific middlewares second (will be innermost)
	for i := len(options.Middlewares) - 1; i >= 0; i-- {
		wrapped = options.Middlewares[i](wrapped)
	}

	var reqZero TParams
	var respZero TResult
	h.generator.AddHandlerType(method, reqZero, respZero, options.Docs)

	h.registerHandler(method, Method{
		handler: wrapped,
		parser:  parser,
	})
}

// HandlerContext contains data that a handler might need.
type HandlerContext struct {
	Logger   *slog.Logger // Logger for this specific request (has method name and request ID)
	WSConn   *WSClient    // WSConn is the WebSocket client (nil for HTTP requests)
	HTTPConn *HTTPClient  // HTTPConn is the HTTP client (nil for WebSocket requests)
}

type HandlerError interface {
	Error() string
	Code() int
}

// handlerError is the default implementation of HandlerError.
type handlerError struct {
	code    int
	message string
}

func (e handlerError) Error() string {
	return e.message
}

func (e handlerError) Code() int {
	return e.code
}

// NewHandlerError creates a new HandlerError.
func NewHandlerError(code int, message string) HandlerError {
	return handlerError{code: code, message: message}
}

// Hub maintains active clients and broadcasts messages.
type Hub struct {
	logger *slog.Logger

	middlewares []MiddlewareFunc

	clientCount      int
	clientCountMutex sync.RWMutex

	clients      map[*WSClient]struct{}
	clientsMutex sync.RWMutex

	methods      map[string]Method
	methodsMutex sync.RWMutex

	subscriptions      map[string]map[*WSClient]struct{}
	subscriptionsMutex sync.RWMutex

	register   chan *WSClient
	unregister chan *WSClient
	eventChan  chan RPCEvent

	generator generate.Generator
}

// NewHub creates a new Hub instance.
func NewHub(l *slog.Logger, g generate.Generator) *Hub {
	logger := l.With(slog.String("component", "hub"))

	return &Hub{
		logger:     logger,
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		eventChan:  make(chan RPCEvent, 100),

		clientCount:      0,
		clientCountMutex: sync.RWMutex{},

		clients:      make(map[*WSClient]struct{}),
		clientsMutex: sync.RWMutex{},

		methods:      make(map[string]Method),
		methodsMutex: sync.RWMutex{},

		subscriptions:      make(map[string]map[*WSClient]struct{}),
		subscriptionsMutex: sync.RWMutex{},

		generator: g,
	}
}

func (h *Hub) GenerateDocs() error {
	return h.generator.Generate()
}

// PublishEvent sends an event to all subscribed clients.
func (h *Hub) PublishEvent(event RPCEvent) {
	h.eventChan <- event
}

// Subscribe adds a client to an event subscription.
func (h *Hub) Subscribe(client *WSClient, event string) error {
	h.subscriptionsMutex.Lock()
	// Check if event is registered
	if _, ok := h.subscriptions[event]; !ok {
		h.subscriptionsMutex.Unlock()
		return fmt.Errorf("unknown event: %s", event)
	}

	h.subscriptions[event][client] = struct{}{}
	h.subscriptionsMutex.Unlock()

	client.logger.Info("subscribed to event", slog.String("event", event))
	return nil
}

// Unsubscribe removes a client from an event subscription.
func (h *Hub) Unsubscribe(client *WSClient, event string) {
	h.subscriptionsMutex.Lock()
	if subscribers, ok := h.subscriptions[event]; ok {
		delete(subscribers, client)
	}
	h.subscriptionsMutex.Unlock()

	client.logger.Info("unsubscribed from event", slog.String("event", event))
}

// WithMiddleware adds middleware to the hub that will be applied to all registered methods.
func (h *Hub) WithMiddleware(middlewares ...MiddlewareFunc) *Hub {
	h.middlewares = append(h.middlewares, middlewares...)
	return h
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	h.logger.Info("hub started")

	for {
		select {
		case client := <-h.register:
			h.clientRegister(client)

		case client := <-h.unregister:
			h.clientUnregister(client)

		case event := <-h.eventChan:
			h.broadcastEvent(event)
		}
	}
}

// registerEvent registers an event that clients can subscribe to.
func (h *Hub) registerEvent(eventName string) {
	h.subscriptionsMutex.Lock()
	defer h.subscriptionsMutex.Unlock()
	if _, exists := h.subscriptions[eventName]; exists {
		h.logger.Warn("event already registered", slog.String("event", eventName))
		return
	}
	h.subscriptions[eventName] = make(map[*WSClient]struct{})
	h.logger.Debug("event registered", slog.String("event", eventName))
}

// registerHandler registers a method handler.
func (h *Hub) registerHandler(methodName string, handler Method) {
	h.methodsMutex.Lock()
	h.methods[methodName] = handler
	h.methodsMutex.Unlock()
	h.logger.Debug("method registered", slog.String("method", methodName))
}
