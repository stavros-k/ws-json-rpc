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

const MAX_QUEUED_EVENTS = 2048

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

type wsNotification struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type HandlerFunc func(ctx context.Context, params any) (any, error)
type TypedHandlerFunc[TParams any, TResult any] func(ctx context.Context, params TParams) (TResult, error)
type MethodInfo struct {
	// The actual handler function
	handler HandlerFunc
	// Parses the params into the appropriate type
	parser func(json.RawMessage) (any, error)
}

// Event represents an event that can be broadcast to subscribers
type Event struct {
	Name string
	Data any
}

func NewEvent(event string, data any) Event {
	return Event{Name: event, Data: data}
}

func RegisterMethod[TParams any, TResult any](h *Hub, method string, handler TypedHandlerFunc[TParams, TResult], middlewares ...MiddlewareFunc) {
	wrapped := func(ctx context.Context, params any) (any, error) {
		return handler(ctx, params.(TParams))
	}
	parser := func(rawParams json.RawMessage) (any, error) {
		return FromJSON[TParams](rawParams)
	}

	h.registerHandler(method, MethodInfo{
		handler: applyMiddleware(wrapped, middlewares...),
		parser:  parser,
	})
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	logger *slog.Logger

	clientCount int
	clients     map[*Client]struct{}
	clientMutex sync.RWMutex

	register   chan *Client
	unregister chan *Client
	eventChan  chan Event

	methods     map[string]MethodInfo
	methodMutex sync.RWMutex

	knownEvents     map[string]struct{}
	knownEventMutex sync.RWMutex

	subscriptions     map[string]map[*Client]struct{}
	subscriptionMutex sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub(l *slog.Logger) *Hub {
	logger := l.With(slog.String("component", "hub"))

	return &Hub{
		clientCount:   0,
		clients:       make(map[*Client]struct{}),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		methods:       make(map[string]MethodInfo),
		knownEvents:   make(map[string]struct{}),
		subscriptions: make(map[string]map[*Client]struct{}),
		eventChan:     make(chan Event, 100),
		logger:        logger,
	}
}

// RegisterEvent registers an event that clients can subscribe to
func (h *Hub) RegisterEvent(eventName string) {
	h.knownEventMutex.Lock()
	defer h.knownEventMutex.Unlock()
	h.knownEvents[eventName] = struct{}{}
	h.logger.Debug("event registered", slog.String("event", eventName))
}

func (h *Hub) registerHandler(methodName string, handler MethodInfo) {
	h.methodMutex.Lock()
	defer h.methodMutex.Unlock()
	h.methods[methodName] = handler
	h.logger.Debug("method registered", slog.String("method", methodName))
}

// Subscribe adds a client to an event subscription
func (h *Hub) Subscribe(client *Client, event string) error {
	h.knownEventMutex.RLock()
	defer h.knownEventMutex.RUnlock()

	h.subscriptionMutex.Lock()
	defer h.subscriptionMutex.Unlock()

	// Check if event is registered
	if _, ok := h.knownEvents[event]; !ok {
		return fmt.Errorf("unknown event: %s", event)
	}

	if h.subscriptions[event] == nil {
		h.subscriptions[event] = make(map[*Client]struct{})
	}
	h.subscriptions[event][client] = struct{}{}

	client.logger.Info("subscribed to event", slog.String("event", event))
	return nil
}

// Unsubscribe removes a client from an event subscription
func (h *Hub) Unsubscribe(client *Client, event string) {
	h.subscriptionMutex.Lock()
	defer h.subscriptionMutex.Unlock()

	if subscribers, ok := h.subscriptions[event]; ok {
		delete(subscribers, client)
	}

	client.logger.Info("unsubscribed from event", slog.String("event", event))
}

// PublishEvent sends an event to all subscribed clients
func (h *Hub) PublishEvent(event Event) {
	h.eventChan <- event
}

func (h *Hub) handleRegister(client *Client) {
	h.subscriptionMutex.Lock()
	defer h.subscriptionMutex.Unlock()

	h.clients[client] = struct{}{}
	h.clientCount++
	h.logger.Info("client registered", slog.String("client_id", client.id), slog.String("remote_addr", client.remoteAddr))
}

func (h *Hub) handleUnregister(client *Client) {
	h.subscriptionMutex.Lock()
	defer h.subscriptionMutex.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		h.clientCount--
		for _, subscribers := range h.subscriptions {
			delete(subscribers, client)
		}
		close(client.sendChannel)
	}
	h.logger.Info("client disconnected", slog.String("client_id", client.id), slog.String("remote_addr", client.remoteAddr))
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	h.logger.Info("hub started")

	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case event := <-h.eventChan:
			h.broadcastEvent(event)
		}
	}
}

func (h *Hub) broadcastEvent(event Event) {
	h.subscriptionMutex.RLock()
	subscribers := h.subscriptions[event.Name]
	h.subscriptionMutex.RUnlock()

	if len(subscribers) == 0 {
		return
	}

	data, err := ToJSON(event.Data)
	if err != nil {
		h.logger.Error("failed to marshal event data",
			slog.String("event", event.Name),
			slog.String("error", err.Error()))
		return
	}

	notification := wsNotification{Event: event.Name, Data: data}

	msg, err := ToJSON(notification)
	if err != nil {
		h.logger.Error("failed to marshal notification",
			slog.String("event", event.Name),
			slog.String("error", err.Error()))
		return
	}

	count := 0
	for client := range subscribers {
		select {
		case client.sendChannel <- msg:
			count++
		default:
			client.logger.Warn("send channel full, skipping notification",
				slog.String("event", event.Name))
		}
	}

	h.logger.Debug("event broadcast",
		slog.String("event", event.Name),
		slog.Int("recipients", count))
}

// ServeWS handles websocket requests from clients
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
			clientID = fmt.Sprintf("client-%d-%s", h.clientCount, remoteHost)
		}

		client := &Client{
			hub:         h,
			conn:        conn,
			remoteAddr:  remoteHost,
			ctx:         ctx,
			cancel:      cancel,
			sendChannel: make(chan []byte, MAX_QUEUED_EVENTS),
			id:          clientID,
			logger:      h.logger.With(slog.String("client_id", clientID), slog.String("remote_addr", remoteHost)),
		}

		h.register <- client

		go client.writePump()
		go client.readPump()
	}
}
