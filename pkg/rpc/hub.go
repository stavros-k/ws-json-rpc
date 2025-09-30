package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"
	"ws-json-rpc/pkg/rpc/generate"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const (
	MAX_QUEUED_EVENTS_PER_CLIENT = 256
	MAX_REQUEST_TIMEOUT          = 30 * time.Second
	MAX_RESPONSE_TIMEOUT         = 30 * time.Second
	MAX_MESSAGE_SIZE             = 1024 * 1024 // 1 MB
)

// RPCRequest represents an object from the client
type RPCRequest struct {
	Version string          `json:"jsonrpc"`
	ID      uuid.UUID       `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCEvent represents an RPCEvent that can be broadcast to subscribers
type RPCEvent struct {
	EventName string `json:"event"`
	Data      any    `json:"data"`
}

// RPCResponse represents a response from the server
type RPCResponse struct {
	Version string          `json:"jsonrpc"`
	ID      uuid.UUID       `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCErrorObj    `json:"error,omitempty"`
}

// NewRPCResponse creates a new JSON-RPC 2.0 response. Result is marshaled internally.
func NewRPCResponse(id uuid.UUID, result any, err *RPCErrorObj) RPCResponse {
	// Marshal the result
	data, jsonErr := ToJSON(result)
	if jsonErr != nil {
		RPCErrorObj := RPCErrorObj{Code: ErrCodeInternal, Message: "Failed to serialize response"}
		return RPCResponse{Version: "2.0", ID: id, Error: &RPCErrorObj}
	}

	return RPCResponse{Version: "2.0", ID: id, Result: data, Error: err}
}

// RPCErrorObj represents an error on a response
type RPCErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type HandlerError interface {
	Error() string
	Code() int
}

// handlerError is the default implementation of HandlerError
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

// NewHandlerError creates a new HandlerError
func NewHandlerError(code int, message string) HandlerError {
	return handlerError{code: code, message: message}
}

// HandlerFunc is a function that handles a method call
type HandlerFunc func(ctx context.Context, hctx *HandlerContext, params any) (any, error)

// TypedHandlerFunc is a function that handles a method call with typed parameters
type TypedHandlerFunc[TParams any, TResult any] func(ctx context.Context, hctx *HandlerContext, params TParams) (TResult, error)

// MiddlewareFunc is a function that wraps a HandlerFunc with additional behavior
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Method represents a registered method in the hub
type Method struct {
	// The actual handler function
	handler HandlerFunc
	// Parses the params into the appropriate type
	parser func(json.RawMessage) (any, error)
}

// HandlerContext contains data that a handler might need
type HandlerContext struct {
	Logger   *slog.Logger // Logger for this specific request (has method name and request ID)
	WSConn   *WSClient    // WSConn is the WebSocket client (nil for HTTP requests)
	HTTPConn *HTTPClient  // HTTPConn is the HTTP client (nil for WebSocket requests)
}

// NewEvent creates a new event
func NewEvent(eventName string, data any) RPCEvent {
	return RPCEvent{EventName: eventName, Data: data}
}

// RegisterMethod registers a method with the hub
func RegisterMethod[TParams any, TResult any](h *Hub, docs generate.HandlerDocs, method string, handler TypedHandlerFunc[TParams, TResult], middlewares ...MiddlewareFunc) {
	wrapped := func(ctx context.Context, hctx *HandlerContext, params any) (any, error) {
		return handler(ctx, hctx, params.(TParams))
	}

	parser := func(rawParams json.RawMessage) (any, error) {
		return FromJSON[TParams](rawParams)
	}

	// Apply global middlewares first (will be outermost)
	for i := len(h.middlewares) - 1; i >= 0; i-- {
		wrapped = h.middlewares[i](wrapped)
	}

	// Apply method-specific middlewares second (will be innermost)
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}

	var reqZero TParams
	var respZero TResult

	docs.ParamsType = reqZero
	docs.ResultType = respZero
	for _, ex := range docs.Examples {
		if reflect.TypeOf(ex.Params) != reflect.TypeOf(reqZero) {
			panic("example params type does not match handler params type")
		}
		if reflect.TypeOf(ex.Result) != reflect.TypeOf(respZero) {
			panic("example result type does not match handler result type")
		}
	}

	h.generator.AddHandlerType(method, reqZero, respZero, docs)

	h.registerHandler(method, Method{
		handler: wrapped,
		parser:  parser,
	})
}

// RegisterEvent registers an event with the hub
func RegisterEvent[TResult any](h *Hub, docs generate.EventDocs, eventName string) {
	var eventZero TResult
	h.generator.AddEventType(eventName, eventZero, docs)
	h.registerEvent(eventName)
}

// Hub maintains active clients and broadcasts messages
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

// NewHub creates a new Hub instance
func NewHub(l *slog.Logger) *Hub {
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

		generator: generate.NewGenerator(),
	}
}

func (h *Hub) G() {
	h.generator.Run()
	os.Exit(0)
}

// WithMiddleware adds middleware to the hub that will be applied to all registered methods
func (h *Hub) WithMiddleware(middlewares ...MiddlewareFunc) *Hub {
	h.middlewares = append(h.middlewares, middlewares...)
	return h
}

// registerEvent registers an event that clients can subscribe to
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

// Subscribe adds a client to an event subscription
func (h *Hub) Subscribe(client *WSClient, event string) error {
	h.subscriptionsMutex.Lock()
	// Check if event is registered
	if _, ok := h.subscriptions[event]; !ok {
		return fmt.Errorf("unknown event: %s", event)
	}

	h.subscriptions[event][client] = struct{}{}
	h.subscriptionsMutex.Unlock()

	client.logger.Info("subscribed to event", slog.String("event", event))
	return nil
}

// Unsubscribe removes a client from an event subscription
func (h *Hub) Unsubscribe(client *WSClient, event string) {
	h.subscriptionsMutex.Lock()
	if subscribers, ok := h.subscriptions[event]; ok {
		delete(subscribers, client)
	}
	h.subscriptionsMutex.Unlock()

	client.logger.Info("unsubscribed from event", slog.String("event", event))
}

// PublishEvent sends an event to all subscribed clients
func (h *Hub) PublishEvent(event RPCEvent) {
	h.eventChan <- event
}

// Run starts the hub's main loop
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

// ServeWS handles websocket requests from clients
// This is called for every new connection
func (h *Hub) ServeWS() http.HandlerFunc {
	wsLogger := h.logger.With(slog.String("handler", "ws"))
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			wsLogger.Error("upgrade failed", slog.String("error", err.Error()))
			return
		}
		conn.SetReadLimit(MAX_MESSAGE_SIZE)

		remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			wsLogger.Error("failed to parse remote address", slog.String("error", err.Error()))
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		clientID := r.URL.Query().Get("clientID")
		if clientID == "" {
			wsLogger.Warn("no client ID provided, generating one", slog.String("remote_addr", remoteHost))
			clientID = fmt.Sprintf("client-%s-%p", remoteHost, conn)
		}

		client := &WSClient{
			hub:         h,
			conn:        conn,
			id:          clientID,
			remoteHost:  remoteHost,
			ctx:         ctx,
			cancel:      cancel,
			sendChannel: make(chan []byte, MAX_QUEUED_EVENTS_PER_CLIENT),
			logger: wsLogger.With(
				slog.String("client_id", clientID),
				slog.String("remote_addr", remoteHost),
			),
		}

		h.register <- client

		go client.writePump()
		go client.readPump()
	}
}

// registerHandler registers a method handler
func (h *Hub) registerHandler(methodName string, handler Method) {
	h.methodsMutex.Lock()
	h.methods[methodName] = handler
	h.methodsMutex.Unlock()
	h.logger.Debug("method registered", slog.String("method", methodName))
}

// clientRegister adds a new client to the hub
func (h *Hub) clientRegister(client *WSClient) {
	h.clientsMutex.Lock()
	h.clients[client] = struct{}{}
	h.clientsMutex.Unlock()

	h.clientCountMutex.Lock()
	h.clientCount++
	h.clientCountMutex.Unlock()

	h.logger.Info("client registered", slog.String("client_id", client.id), slog.String("remote_host", client.remoteHost))
}

// clientUnregister removes a client from the hub
func (h *Hub) clientUnregister(client *WSClient) {
	h.clientsMutex.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)

		h.clientCountMutex.Lock()
		h.clientCount--
		h.clientCountMutex.Unlock()

		h.subscriptionsMutex.Lock()
		for _, subscribers := range h.subscriptions {
			delete(subscribers, client)
		}
		h.subscriptionsMutex.Unlock()
	}
	h.clientsMutex.Unlock()
	h.logger.Info("client disconnected", slog.String("client_id", client.id), slog.String("remote_host", client.remoteHost))
}

// ServeHTTP handles HTTP JSON-RPC requests
func (h *Hub) ServeHTTP() http.HandlerFunc {
	httpLogger := h.logger.With(slog.String("handler", "http"))
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			httpLogger.Warn("http request not allowed", slog.String("method", r.Method))
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the request using streaming JSON helper
		req, err := FromJSONStream[RPCRequest](r.Body)
		if err != nil {
			// Create a minimal error response
			resp := NewRPCResponse(uuid.Nil, nil, &RPCErrorObj{Code: ErrCodeParse, Message: "Invalid JSON in request body"})
			w.Header().Set("Content-Type", "application/json")
			if err := ToJSONStream(w, resp); err != nil {
				// Log the error but cannot do much else
				httpLogger.Error("failed to encode HTTP response", slog.String("error", err.Error()))
			}
			return
		}

		// Create HTTP client for this request
		remoteHost, _, _ := net.SplitHostPort(r.RemoteAddr)
		clientID := fmt.Sprintf("http-%s-%s", remoteHost, req.ID.String())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client := &HTTPClient{
			w:          w,
			r:          r,
			hub:        h,
			remoteHost: remoteHost,
			ctx:        ctx,
			cancel:     cancel,
			id:         clientID,
			logger: httpLogger.With(
				slog.String("client_id", clientID),
				slog.String("remote_host", remoteHost),
			),
		}

		// Handle the request
		client.handleRequest(req)
	}
}

func (h *Hub) broadcastEvent(event RPCEvent) {
	h.subscriptionsMutex.RLock()
	defer h.subscriptionsMutex.RUnlock()
	subscribers, ok := h.subscriptions[event.EventName]
	if !ok {
		h.logger.Warn("attempted to publish to unregistered event", slog.String("event", event.EventName))
		return
	}

	if len(subscribers) == 0 {
		h.logger.Debug("no subscribers for event", slog.String("event", event.EventName))
		return
	}

	result, err := ToJSON(event)
	if err != nil {
		h.logger.Error("failed to marshal event", slog.String("event", event.EventName), slog.String("error", err.Error()))
		return
	}

	count := 0
	for client := range subscribers {
		select {
		case client.sendChannel <- result:
			count++
		default:
			client.logger.Error("send channel full, skipping event broadcast", slog.String("event", event.EventName))
		}
	}

	h.logger.Debug("event broadcast", slog.String("event", event.EventName), slog.Int("recipients", count))
}
