package wshub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Message types for JSON-RPC-like protocol
type Request struct {
	Method MethodKinder    `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     *int            `json:"id,omitempty"` // nil for notifications
}

type Response struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ErrorObj       `json:"error,omitempty"`
	ID     *int            `json:"id"`
}

type ErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Notification struct {
	Event EventKinder     `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// Client represents a connected WebSocket client
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
	send   chan []byte
	id     string
	logger *slog.Logger
}

// HandlerFunc with generics
type HandlerFunc[TParams any, TResult any] func(ctx context.Context, params TParams) (TResult, error)

// HandlerWrapper wraps typed handlers for storage
type HandlerWrapper struct {
	handler func(context.Context, json.RawMessage) (any, error)
}

// Event represents an event that can be broadcast to subscribers
type Event struct {
	Name EventKinder
	Data any
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	clients       map[*Client]bool
	register      chan *Client
	unregister    chan *Client
	handlers      map[MethodKinder]HandlerWrapper
	events        map[EventKinder]struct{} // Registered events
	subscriptions map[EventKinder]map[*Client]struct{}
	mu            sync.RWMutex
	eventChan     chan Event
	logger        *slog.Logger
}

// New creates a new Hub instance
func New(l *slog.Logger) *Hub {
	logger := l.With(slog.String("component", "hub"))

	return &Hub{
		clients:       make(map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		handlers:      make(map[MethodKinder]HandlerWrapper),
		events:        make(map[EventKinder]struct{}),
		subscriptions: make(map[EventKinder]map[*Client]struct{}),
		eventChan:     make(chan Event, 100),
		logger:        logger,
	}
}

// RegisterEvent registers an event that clients can subscribe to
func (h *Hub) RegisterEvent(eventName EventKinder) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events[eventName] = struct{}{}
	h.logger.Debug("event registered", slog.String("event", eventName.String()))
}

// RegisterHandler registers a generic typed handler
func RegisterHandler[TParams any, TResult any](h *Hub, method MethodKinder, handler HandlerFunc[TParams, TResult]) {
	h.mu.Lock()
	defer h.mu.Unlock()

	wrapper := HandlerWrapper{
		handler: func(ctx context.Context, rawParams json.RawMessage) (interface{}, error) {
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

	h.handlers[method] = wrapper
	h.logger.Debug("handler registered", slog.String("method", method.String()))
}

// Call allows handlers to call other handlers
func (h *Hub) Call(ctx context.Context, method MethodKinder, params interface{}) (interface{}, error) {
	h.mu.RLock()
	wrapper, exists := h.handlers[method]
	h.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("method not found: %s", method)
	}

	paramsJSON, err := ToJSON(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	return wrapper.handler(ctx, paramsJSON)
}

// Subscribe adds a client to an event subscription
func (h *Hub) Subscribe(client *Client, event EventKinder) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if event is registered
	if _, ok := h.events[event]; !ok {
		return fmt.Errorf("unknown event: %s", event)
	}

	if h.subscriptions[event] == nil {
		h.subscriptions[event] = make(map[*Client]struct{})
	}
	h.subscriptions[event][client] = struct{}{}

	client.logger.Info("subscribed to event", slog.String("event", event.String()))
	return nil
}

// Unsubscribe removes a client from an event subscription
func (h *Hub) Unsubscribe(client *Client, event EventKinder) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers, ok := h.subscriptions[event]; ok {
		delete(subscribers, client)
	}

	client.logger.Info("unsubscribed from event", slog.String("event", event.String()))
}

// PublishEvent sends an event to all subscribed clients
func (h *Hub) PublishEvent(event Event) {
	h.eventChan <- event
}

// GetEventChannel returns the channel for publishing events
func (h *Hub) GetEventChannel() chan<- Event {
	return h.eventChan
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	h.logger.Info("hub started")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info("client connected", slog.String("client_id", client.id))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				h.removeClientSubscriptions(client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Info("client disconnected", slog.String("client_id", client.id))

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
			slog.String("event", event.Name.String()),
			slog.String("error", err.Error()))
		return
	}

	notification := Notification{
		Event: event.Name,
		Data:  data,
	}

	msg, err := ToJSON(notification)
	if err != nil {
		h.logger.Error("failed to marshal notification",
			slog.String("event", event.Name.String()),
			slog.String("error", err.Error()))
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for client := range subscribers {
		select {
		case client.send <- msg:
			count++
		default:
			client.logger.Warn("send channel full, skipping notification",
				slog.String("event", event.Name.String()))
		}
	}

	h.logger.Debug("event broadcast",
		slog.String("event", event.Name.String()),
		slog.Int("recipients", count))
}

func (c *Client) readPump() {
	defer func() {
		c.cancel()
		c.hub.unregister <- c
	}()

	for {
		_, message, err := c.conn.Read(c.ctx)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				c.logger.Error("websocket error", slog.String("error", err.Error()))
			}
			break
		}

		req, err := FromJSON[Request](message)
		if err != nil {
			c.logger.Warn("parse error", slog.String("error", err.Error()))
			c.sendError(nil, -32700, "Parse error")
			continue
		}

		c.logger.Debug("request received",
			slog.String("method", req.Method.String()),
			slog.Any("id", req.ID))

		// Handle the request
		go c.handleRequest(req)
	}
}

func (c *Client) writePump() {
	defer c.cancel()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
			err := c.conn.Write(ctx, websocket.MessageText, message)
			cancel()

			if err != nil {
				c.logger.Error("write error", slog.String("error", err.Error()))
				return
			}
		}
	}
}

func (c *Client) handleRequest(req Request) {
	ctx := context.WithValue(c.ctx, "client", c)

	// Handle regular method calls
	c.hub.mu.RLock()
	wrapper, exists := c.hub.handlers[req.Method]
	c.hub.mu.RUnlock()

	if !exists {
		if req.ID != nil {
			c.sendError(req.ID, -32601, "Method not found")
		}
		return
	}

	// If it's a notification (no ID), we don't send a response
	if req.ID == nil {
		wrapper.handler(ctx, req.Params)
		return
	}

	// Call the handler with context
	result, err := wrapper.handler(ctx, req.Params)
	if err != nil {
		c.logger.Error("handler error",
			slog.String("method", req.Method.String()),
			slog.String("error", err.Error()))
		c.sendError(req.ID, -32603, err.Error())
		return
	}

	c.sendResult(req.ID, result)
}

func (c *Client) sendResult(id *int, result any) {
	data, err := ToJSON(result)
	if err != nil {
		c.sendError(id, -32603, "Internal error")
		return
	}

	resp := Response{
		Result: data,
		ID:     id,
	}

	msg, err := ToJSON(resp)
	if err != nil {
		c.logger.Error("failed to encode response", slog.String("error", err.Error()))
		return
	}

	select {
	case c.send <- msg:
	case <-c.ctx.Done():
	}
}

func (c *Client) sendError(id *int, code int, message string) {
	resp := Response{
		Error: &ErrorObj{
			Code:    code,
			Message: message,
		},
		ID: id,
	}

	msg, err := ToJSON(resp)
	if err != nil {
		c.logger.Error("failed to encode error response", slog.String("error", err.Error()))
		return
	}

	select {
	case c.send <- msg:
	case <-c.ctx.Done():
	}
}

// ServeWS handles websocket requests from clients
func (h *Hub) ServeWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// Configure for production
			// OriginPatterns: []string{"example.com"},
		})
		if err != nil {
			h.logger.Error("upgrade failed", slog.String("error", err.Error()))
			return
		}

		ctx, cancel := context.WithCancel(r.Context())

		clientID := r.Header.Get("X-Client-Id")
		if clientID == "" {
			clientID = fmt.Sprintf("client-%p", conn)
		}

		client := &Client{
			hub:    h,
			conn:   conn,
			ctx:    ctx,
			cancel: cancel,
			send:   make(chan []byte, 256),
			id:     clientID,
			logger: h.logger.With(slog.String("client_id", clientID)),
		}

		h.register <- client

		go client.writePump()
		go client.readPump()
	}
}
