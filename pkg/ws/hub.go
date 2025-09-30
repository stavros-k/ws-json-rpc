package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
	"ws-json-rpc/pkg/ws/generate"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const (
	MAX_QUEUED_EVENTS_PER_CLIENT = 256
	MAX_REQUEST_TIMEOUT          = 30 * time.Second
	MAX_RESPONSE_TIMEOUT         = 30 * time.Second
	MAX_MESSAGE_SIZE             = 1024 * 1024 // 1 MB
)

// rpcRequest represents an object from the client
type rpcRequest struct {
	Version string          `json:"jsonrpc"`
	ID      uuid.UUID       `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// wsEvent represents an wsEvent that can be broadcast to subscribers
type wsEvent struct {
	EventName string `json:"event"`
	Data      any    `json:"data"`
}

// rpcResponse represents a response from the server
type rpcResponse struct {
	Version string          `json:"jsonrpc"`
	ID      uuid.UUID       `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcErrorObj    `json:"error,omitempty"`
}

// rpcErrorObj represents an error on a response
type rpcErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type HandlerError interface {
	Error() string
	Code() int
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
	Client *Client
	Logger *slog.Logger
}

// NewEvent creates a new event
func NewEvent(eventName string, data any) wsEvent {
	return wsEvent{EventName: eventName, Data: data}
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

	clients      map[*Client]struct{}
	clientsMutex sync.RWMutex

	methods      map[string]Method
	methodsMutex sync.RWMutex

	subscriptions      map[string]map[*Client]struct{}
	subscriptionsMutex sync.RWMutex

	register   chan *Client
	unregister chan *Client
	eventChan  chan wsEvent

	generator generate.Generator
}

// NewHub creates a new Hub instance
func NewHub(l *slog.Logger) *Hub {
	logger := l.With(slog.String("component", "hub"))

	return &Hub{
		logger:     logger,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		eventChan:  make(chan wsEvent, 100),

		clientCount:      0,
		clientCountMutex: sync.RWMutex{},

		clients:      make(map[*Client]struct{}),
		clientsMutex: sync.RWMutex{},

		methods:      make(map[string]Method),
		methodsMutex: sync.RWMutex{},

		subscriptions:      make(map[string]map[*Client]struct{}),
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
	h.subscriptions[eventName] = make(map[*Client]struct{})
	h.logger.Debug("event registered", slog.String("event", eventName))
}

// Subscribe adds a client to an event subscription
func (h *Hub) Subscribe(client *Client, event string) error {
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
func (h *Hub) Unsubscribe(client *Client, event string) {
	h.subscriptionsMutex.Lock()
	if subscribers, ok := h.subscriptions[event]; ok {
		delete(subscribers, client)
	}
	h.subscriptionsMutex.Unlock()

	client.logger.Info("unsubscribed from event", slog.String("event", event))
}

// PublishEvent sends an event to all subscribed clients
func (h *Hub) PublishEvent(event wsEvent) {
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
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			h.logger.Error("upgrade failed", slog.String("error", err.Error()))
			return
		}
		conn.SetReadLimit(MAX_MESSAGE_SIZE)

		remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			h.logger.Error("failed to parse remote address", slog.String("error", err.Error()))
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		clientID := r.URL.Query().Get("clientID")
		if clientID == "" {
			h.logger.Warn("no client ID provided, generating one", slog.String("remote_addr", remoteHost))
			clientID = fmt.Sprintf("client-%s-%p", remoteHost, conn)
		}

		client := &Client{
			hub:         h,
			conn:        conn,
			id:          clientID,
			remoteAddr:  remoteHost,
			ctx:         ctx,
			cancel:      cancel,
			sendChannel: make(chan []byte, MAX_QUEUED_EVENTS_PER_CLIENT),
			logger:      h.logger.With(slog.String("client_id", clientID), slog.String("remote_addr", remoteHost)),
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
func (h *Hub) clientRegister(client *Client) {
	h.clientsMutex.Lock()
	h.clients[client] = struct{}{}
	h.clientsMutex.Unlock()

	h.clientCountMutex.Lock()
	h.clientCount++
	h.clientCountMutex.Unlock()

	h.logger.Info("client registered", slog.String("client_id", client.id), slog.String("remote_addr", client.remoteAddr))
}

// clientUnregister removes a client from the hub
func (h *Hub) clientUnregister(client *Client) {
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
	h.logger.Info("client disconnected", slog.String("client_id", client.id), slog.String("remote_addr", client.remoteAddr))
}

// ServeHTTP handles HTTP JSON-RPC requests
func (h *Hub) ServeHTTP() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the request using streaming JSON helper
		req, err := FromJSONStream[rpcRequest](r.Body)
		if err != nil {
			h.sendHTTPError(w, uuid.Nil, ErrCodeParse, "Invalid JSON in request body")
			return
		}

		// Handle the request
		h.handleHTTPRequest(w, r, req)
	}
}

// handleHTTPRequest processes a single HTTP JSON-RPC request
func (h *Hub) handleHTTPRequest(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	reqLogger := h.logger.With(
		slog.String("method", req.Method),
		slog.String("id", req.ID.String()),
		slog.String("remote_addr", r.RemoteAddr),
	)

	reqLogger.Debug("handling HTTP JSON-RPC request")

	// Get the handler
	h.methodsMutex.RLock()
	method, exists := h.methods[req.Method]
	h.methodsMutex.RUnlock()
	if !exists {
		h.sendHTTPError(w, req.ID, ErrCodeNotFound, fmt.Sprintf("Method %q not found", req.Method))
		return
	}

	// Parse json into the structured params
	typedParams, err := method.parser(req.Params)
	if err != nil {
		reqLogger.Error("unmarshal error", slog.String("error", err.Error()))
		h.sendHTTPError(w, req.ID, ErrCodeInternal, fmt.Sprintf("Failed to parse params on method %q: %s", req.Method, err.Error()))
		return
	}

	// Set a timeout for the request
	ctx, cancel := context.WithTimeout(r.Context(), MAX_REQUEST_TIMEOUT)
	defer cancel()

	// Create a minimal HandlerContext for HTTP (no client)
	hctx := &HandlerContext{Client: nil, Logger: reqLogger}

	// Call the handler
	result, err := method.handler(ctx, hctx, typedParams)
	if err != nil {
		reqLogger.Error("handler error", slog.String("error", err.Error()))
		// If its a handler error, let handler specify code/message
		if err, ok := err.(HandlerError); ok {
			h.sendHTTPError(w, req.ID, err.Code(), err.Error())
			return
		}

		// Unknown errors, send internal error
		h.sendHTTPError(w, req.ID, ErrCodeInternal, fmt.Sprintf("Failed to handle request on method %q: %s", req.Method, err.Error()))
		return
	}

	h.sendHTTPSuccess(w, req.ID, result)
}

// sendHTTPSuccess sends a successful JSON-RPC response over HTTP
func (h *Hub) sendHTTPSuccess(w http.ResponseWriter, id uuid.UUID, result any) {
	data, err := ToJSON(result)
	if err != nil {
		h.sendHTTPError(w, id, ErrCodeInternal, "Failed to serialize response")
		return
	}

	resp := rpcResponse{ID: id, Result: data}
	h.sendHTTPResponse(w, resp)
}

// sendHTTPError sends an error JSON-RPC response over HTTP
func (h *Hub) sendHTTPError(w http.ResponseWriter, id uuid.UUID, code int, message string) {
	resp := rpcResponse{
		ID:    id,
		Error: &rpcErrorObj{Code: code, Message: message},
	}
	h.sendHTTPResponse(w, resp)
}

// sendHTTPResponse sends a JSON-RPC response over HTTP using streaming JSON helper
func (h *Hub) sendHTTPResponse(w http.ResponseWriter, resp rpcResponse) {
	w.Header().Set("Content-Type", "application/json")

	if err := ToJSONStream(w, resp); err != nil {
		h.logger.Error("failed to encode HTTP response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Hub) broadcastEvent(event wsEvent) {
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
