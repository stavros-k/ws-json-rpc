package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
	"ws-json-rpc/pkg/utils"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// WSClient represents a connected WebSocket client
type WSClient struct {
	conn        *websocket.Conn
	sendChannel chan []byte
	hub         *Hub
	remoteHost  string
	ctx         context.Context
	cancel      context.CancelFunc
	id          string
	logger      *slog.Logger
}

func (c *WSClient) readPump() {
	// When readPump exits, cancel the context and unregister the client
	defer func() {
		c.logger.Info("client read pump exited")
		c.cancel()
		c.hub.unregister <- c
	}()

	for {
		// Read next message
		msgType, message, err := c.conn.Read(c.ctx)
		if err != nil {
			// In case of a ws close error, stop the loop
			var ce websocket.CloseError
			switch {
			case errors.As(err, &ce):
				c.logger.Info("websocket closed normally", slog.Int("code", int(ce.Code)), slog.String("text", ce.Reason))
			case errors.Is(err, io.EOF):
				c.logger.Info("websocket closed abruptly", utils.ErrAttr(err))
			default:
				c.logger.Error("unknown websocket error", utils.ErrAttr(err))
			}
			break
		}
		// Only support text based messages
		if msgType != websocket.MessageText {
			if err := c.sendError(uuid.Nil, ErrCodeInvalid, "Invalid message type. Only text messages are supported."); err != nil {
				c.logger.Error("failed to send error response", utils.ErrAttr(err))
			}
			continue
		}

		// Parse message
		req, err := utils.FromJSON[RPCRequest](message)
		if err != nil {
			c.logger.Warn("parse error", utils.ErrAttr(err))
			if err := c.sendError(uuid.Nil, ErrCodeParse, err.Error()); err != nil {
				c.logger.Error("failed to send error response", utils.ErrAttr(err))
			}
			continue
		}

		// Handle the request
		go c.handleRequest(req)
	}
}

func (c *WSClient) writePump() {
	// When writePump exits, cancel the context and close the send channel
	defer func() {
		c.logger.Info("client write pump exited")
		c.cancel()
		close(c.sendChannel)
	}()

	for {
		select {
		// Exit if context is cancelled
		case <-c.ctx.Done():
			// Close connection on context cancellation
			c.conn.Close(websocket.StatusNormalClosure, "")
			return
		// Exit if channel is closed otherwise send the message
		case message, ok := <-c.sendChannel:
			// If the send channel is closed, close the connection
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			// Write message with a timeout
			ctx, cancel := context.WithTimeout(c.ctx, MAX_RESPONSE_TIMEOUT)
			err := c.conn.Write(ctx, websocket.MessageText, message)
			cancel()

			if err != nil {
				c.logger.Error("write error", utils.ErrAttr(err))
				continue
			}
		}
	}
}

func (c *WSClient) handleRequest(req RPCRequest) {
	// Derive a logger from the original for this request
	reqLogger := c.logger.With(slog.String("method", req.Method))
	reqLogger = reqLogger.With(slog.String("id", req.ID.String()))

	// Get the handler
	c.hub.methodsMutex.RLock()
	method, exists := c.hub.methods[req.Method]
	c.hub.methodsMutex.RUnlock()
	if !exists {
		if err := c.sendError(req.ID, ErrCodeNotFound, fmt.Sprintf("Method %q not found", req.Method)); err != nil {
			reqLogger.Error("failed to send error response", utils.ErrAttr(err))
		}
		return
	}

	// Parse json into the structured params
	typedParams, err := method.parser(req.Params)
	if err != nil {
		reqLogger.Error("unmarshal error", utils.ErrAttr(err))
		if err := c.sendError(req.ID, ErrCodeInvalidParams, fmt.Sprintf("Failed to parse params on method %q: %s", req.Method, err.Error())); err != nil {
			reqLogger.Error("failed to send error response", utils.ErrAttr(err))
		}
		return
	}

	// Set a timeout for the request
	ctx, cancel := context.WithTimeout(c.ctx, MAX_REQUEST_TIMEOUT)
	defer cancel()

	// Create a new HandlerContext
	hctx := &HandlerContext{Logger: reqLogger, WSConn: c}

	// Call the handler
	result, err := method.handler(ctx, hctx, typedParams)
	if err != nil {
		hctx.Logger.Error("handler error", utils.ErrAttr(err))
		// If its a handler error, let handler specify code/message
		if err, ok := err.(HandlerError); ok {
			if err := c.sendError(req.ID, err.Code(), err.Error()); err != nil {
				hctx.Logger.Error("failed to send error response", utils.ErrAttr(err))
			}
			return
		}

		// Unknown errors, send internal error
		if err := c.sendError(req.ID, ErrCodeInternal, fmt.Sprintf("Failed to handle request on method %q: %s", req.Method, err.Error())); err != nil {
			hctx.Logger.Error("failed to send error response", utils.ErrAttr(err))
		}
		return
	}

	if err := c.sendSuccess(req.ID, result); err != nil {
		hctx.Logger.Error("failed to send success response", utils.ErrAttr(err))
	}
}

func (c *WSClient) sendSuccess(id uuid.UUID, result any) error {
	return c.sendData(NewRPCResponse(id, result, nil))
}

func (c *WSClient) sendError(id uuid.UUID, code int, message string) error {
	return c.sendData(NewRPCResponse(id, nil, &RPCErrorObj{Code: code, Message: message}))
}

func (c *WSClient) sendData(r RPCResponse) error {
	msg, err := utils.ToJSON(r)
	if err != nil {
		return err
	}

	// Send the message on the send channel with timeout protection
	select {
	case c.sendChannel <- msg:
		return nil
	case <-time.After(MAX_SEND_CHANNEL_TIMEOUT):
		return fmt.Errorf("send channel full, timeout after %v waiting to queue response", MAX_SEND_CHANNEL_TIMEOUT)
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

// ServeWS handles websocket requests from clients
// This is called for every new connection
func (h *Hub) ServeWS() http.HandlerFunc {
	wsLogger := h.logger.With(slog.String("handler", "ws"))
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			wsLogger.Error("upgrade failed", utils.ErrAttr(err))
			return
		}

		// Limit the size of incoming messages
		conn.SetReadLimit(MAX_MESSAGE_SIZE)

		remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			wsLogger.Error("failed to parse remote address", utils.ErrAttr(err), slog.String("remote_addr", r.RemoteAddr))
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		clientID := r.URL.Query().Get("clientID")
		if clientID == "" {
			wsLogger.Warn("no client ID provided, generating one", slog.String("remote_addr", remoteHost))
			clientID = fmt.Sprintf("ws-%s-%s", remoteHost, uuid.NewString())
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

	result, err := utils.ToJSON(event)
	if err != nil {
		h.logger.Error("failed to marshal event", slog.String("event", event.EventName), utils.ErrAttr(err))
		return
	}

	count := 0
	dropped := 0
	for client := range subscribers {
		select {
		case client.sendChannel <- result:
			count++
		default:
			dropped++
			client.logger.Warn("send channel full, dropping event broadcast", slog.String("event", event.EventName))
		}
	}
	log := h.logger.Debug
	if dropped > 0 {
		log = h.logger.Warn
	}
	log("event broadcast", slog.String("event", event.EventName), slog.Int("recipients", len(subscribers)), slog.Int("delivered", count), slog.Int("dropped", dropped))
}
