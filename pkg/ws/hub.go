package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

const MAX_QUEUED_EVENTS_PER_CLIENT = 256

// wsRequest represents an object from the client
type wsRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     *int            `json:"id,omitempty"` // nil for notifications
}

func (r *wsRequest) IsNotification() bool {
	return r.ID == nil
}

// wsResponse represents a response from the server
type wsResponse struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  *wsErrorObj     `json:"error,omitempty"`
	ID     *int            `json:"id"`
}

// wsErrorObj represents an error on a response
type wsErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HandlerFunc is a function that handles a method call
type HandlerFunc func(ctx context.Context, params any) (any, error)

// TypedHandlerFunc is a function that handles a method call with typed parameters
type TypedHandlerFunc[TParams any, TResult any] func(ctx context.Context, params TParams) (TResult, error)

// MiddlewareFunc is a function that wraps a HandlerFunc with additional behavior
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Method represents a registered method in the hub
type Method struct {
	// The actual handler function
	handler HandlerFunc
	// Parses the params into the appropriate type
	parser func(json.RawMessage) (any, error)
}

// event represents an event that can be broadcast to subscribers
type event struct {
	Name string
	Data any
}

// NewEvent creates a new event
func NewEvent(eventName string, data any) event {
	return event{Name: eventName, Data: data}
}

// RegisterMethod registers a method with the hub
func RegisterMethod[TParams any, TResult any](h *Hub, method string, handler TypedHandlerFunc[TParams, TResult], middlewares ...MiddlewareFunc) {
	wrapped := func(ctx context.Context, params any) (any, error) {
		return handler(ctx, params.(TParams))
	}
	parser := func(rawParams json.RawMessage) (any, error) {
		return FromJSON[TParams](rawParams)
	}

	// Apply method-specific middlewares first
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}

	// Then apply global middlewares
	for i := len(h.middlewares) - 1; i >= 0; i-- {
		wrapped = h.middlewares[i](wrapped)
	}

	h.registerHandler(method, Method{
		handler: wrapped,
		parser:  parser,
	})
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
	eventChan  chan event
}

// NewHub creates a new Hub instance
func NewHub(l *slog.Logger) *Hub {
	logger := l.With(slog.String("component", "hub"))

	return &Hub{
		logger:     logger,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		eventChan:  make(chan event, 100),

		clientCount:      0,
		clientCountMutex: sync.RWMutex{},

		clients:      make(map[*Client]struct{}),
		clientsMutex: sync.RWMutex{},

		methods:      make(map[string]Method),
		methodsMutex: sync.RWMutex{},

		subscriptions:      make(map[string]map[*Client]struct{}),
		subscriptionsMutex: sync.RWMutex{},
	}
}

// WithMiddleware adds middleware to the hub that will be applied to all registered methods
func (h *Hub) WithMiddleware(middlewares ...MiddlewareFunc) *Hub {
	h.middlewares = append(h.middlewares, middlewares...)
	return h
}

// RegisterEvent registers an event that clients can subscribe to
func (h *Hub) RegisterEvent(eventName string) {
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
func (h *Hub) PublishEvent(event event) {
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

		remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			h.logger.Error("failed to parse remote address", slog.String("error", err.Error()))
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		clientID := r.Header.Get("X-Client-ID")
		if clientID == "" {
			h.logger.Warn("no client ID provided, generating one", slog.String("remote_addr", remoteHost))
			clientID = fmt.Sprintf("client-%s-%p", remoteHost, conn)
		}

		client := &Client{
			hub:         h,
			conn:        conn,
			remoteAddr:  remoteHost,
			ctx:         ctx,
			cancel:      cancel,
			sendChannel: make(chan []byte, MAX_QUEUED_EVENTS_PER_CLIENT),
			id:          clientID,
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

func (h *Hub) broadcastEvent(event event) {
	h.subscriptionsMutex.RLock()
	defer h.subscriptionsMutex.RUnlock()
	subscribers, ok := h.subscriptions[event.Name]
	if !ok {
		return
	}

	if len(subscribers) == 0 {
		return
	}

	result, err := ToJSON(event)
	if err != nil {
		h.logger.Error("failed to marshal event", slog.String("event", event.Name), slog.String("error", err.Error()))
		return
	}

	msg, err := ToJSON(wsResponse{Result: result})
	if err != nil {
		h.logger.Error("failed to marshal notification", slog.String("event", event.Name), slog.String("error", err.Error()))
		return
	}

	count := 0
	for client := range subscribers {
		select {
		case client.sendChannel <- msg:
			count++
		default:
			client.logger.Warn("send channel full, skipping notification", slog.String("event", event.Name))
		}
	}

	h.logger.Debug("event broadcast", slog.String("event", event.Name), slog.Int("recipients", count))
}
