package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

// Request represents an object from the client
// Method is required
// Params is optional
// ID is optional for notifications only
type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     *int            `json:"id,omitempty"` // nil for notifications
}

func (r *Request) IsNotification() bool {
	return r.ID == nil
}

// Response represents a response from the server
type Response struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ErrorObj       `json:"error,omitempty"`
	ID     *int            `json:"id"`
}

// ErrorObj represents an error on a response
type ErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newNotification(event string, data json.RawMessage) notification {
	return notification{
		Event: event,
		Data:  data,
	}
}

type notification struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// HandlerFunc with generics
type HandlerFunc[TParams any, TResult any] func(ctx context.Context, params TParams) (TResult, error)

// HandlerWrapper wraps typed handlers for storage
type HandlerWrapper struct {
	handle func(context.Context, json.RawMessage) (any, error)
}

// Event represents an event that can be broadcast to subscribers
type Event struct {
	Name string
	Data any
}

func NewEvent(event EventKinder, data any) Event {
	return Event{
		Name: event.String(),
		Data: data,
	}
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	clients       map[*Client]struct{}
	register      chan *Client
	unregister    chan *Client
	handlers      map[string]HandlerWrapper
	events        map[string]struct{} // Registered events
	subscriptions map[string]map[*Client]struct{}
	mu            sync.RWMutex
	eventChan     chan Event
	logger        *slog.Logger
}

// NewHub creates a new Hub instance
func NewHub(l *slog.Logger) *Hub {
	logger := l.With(slog.String("component", "hub"))

	return &Hub{
		clients:       make(map[*Client]struct{}),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		handlers:      make(map[string]HandlerWrapper),
		events:        make(map[string]struct{}),
		subscriptions: make(map[string]map[*Client]struct{}),
		eventChan:     make(chan Event, 100),
		logger:        logger,
	}
}

// RegisterEvent registers an event that clients can subscribe to
func (h *Hub) RegisterEvent(eventName EventKinder) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events[eventName.String()] = struct{}{}
	h.logger.Debug("event registered", slog.String("event", eventName.String()))
}

// RegisterHandler registers a generic typed handler
func RegisterHandler[TParams any, TResult any](h *Hub, method MethodKinder, handler HandlerFunc[TParams, TResult]) {
	h.mu.Lock()
	defer h.mu.Unlock()

	wrapper := HandlerWrapper{
		handle: func(ctx context.Context, rawParams json.RawMessage) (any, error) {
			params, err := FromJSON[TParams](rawParams)
			if err != nil {
				return nil, fmt.Errorf("invalid params: %w", err)
			}

			result, err := handler(ctx, params)
			if err != nil {
				return nil, err
			}

			return result, nil
		},
	}

	h.handlers[method.String()] = wrapper
	h.logger.Debug("handler registered", slog.String("method", method.String()))
}

// Call allows handlers to call other handlers
func (h *Hub) Call(ctx context.Context, method MethodKinder, params any) (any, error) {
	h.mu.RLock()
	handler, exists := h.handlers[method.String()]
	h.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("method not found: %s", method)
	}

	paramsJSON, err := ToJSON(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	return handler.handle(ctx, paramsJSON)
}

// Subscribe adds a client to an event subscription
func (h *Hub) Subscribe(client *Client, event EventKinder) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if event is registered
	if _, ok := h.events[event.String()]; !ok {
		return fmt.Errorf("unknown event: %s", event)
	}

	if h.subscriptions[event.String()] == nil {
		h.subscriptions[event.String()] = make(map[*Client]struct{})
	}
	h.subscriptions[event.String()][client] = struct{}{}

	client.logger.Info("subscribed to event", slog.String("event", event.String()))
	return nil
}

// Unsubscribe removes a client from an event subscription
func (h *Hub) Unsubscribe(client *Client, event EventKinder) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers, ok := h.subscriptions[event.String()]; ok {
		delete(subscribers, client)
	}

	client.logger.Info("unsubscribed from event", slog.String("event", event.String()))
}

// PublishEvent sends an event to all subscribed clients
func (h *Hub) PublishEvent(event Event) {
	h.eventChan <- event
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	h.logger.Info("hub started")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = struct{}{}
			h.mu.Unlock()
			h.logger.Info("client connected", slog.String("client_id", client.id), slog.String("remote_addr", client.netConn.RemoteAddr().String()))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				h.removeClientSubscriptions(client)
				close(client.sendChannel)
			}
			h.mu.Unlock()
			h.logger.Info("client disconnected", slog.String("client_id", client.id), slog.String("remote_addr", client.netConn.RemoteAddr().String()))

		case event := <-h.eventChan:
			h.broadcastEvent(event)
		}
	}
}

func (h *Hub) removeClientSubscriptions(client *Client) {
	// Already under write lock from unregister
	for _, subscribers := range h.subscriptions {
		delete(subscribers, client)
	}
}

func (h *Hub) broadcastEvent(event Event) {
	h.mu.RLock()
	subscribers := h.subscriptions[event.Name]
	h.mu.RUnlock()

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

	notification := newNotification(event.Name, data)

	msg, err := ToJSON(notification)
	if err != nil {
		h.logger.Error("failed to marshal notification",
			slog.String("event", event.Name),
			slog.String("error", err.Error()))
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

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

		ctx, cancel := context.WithCancel(context.Background())
		netConn := websocket.NetConn(ctx, conn, websocket.MessageText)
		clientID := r.Header.Get("X-Client-ID")
		if clientID == "" {
			clientID = fmt.Sprintf("client-%p", conn)
		}

		client := &Client{
			hub:         h,
			conn:        conn,
			netConn:     netConn,
			ctx:         ctx,
			cancel:      cancel,
			sendChannel: make(chan []byte, 256),
			id:          clientID,
			logger:      h.logger.With(slog.String("client_id", clientID), slog.String("remote_addr", netConn.RemoteAddr().String())),
		}

		h.register <- client

		go client.writePump()
		go client.readPump()
	}
}
