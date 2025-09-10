package ws

import (
	"context"
	"log/slog"
	"time"

	"github.com/coder/websocket"
)

// clientContextKey is a key used for storing the client in the context
type clientContextKey struct{}

// Client represents a connected WebSocket client
type Client struct {
	hub         *Hub
	conn        *websocket.Conn
	remoteAddr  string
	ctx         context.Context
	cancel      context.CancelFunc
	sendChannel chan []byte
	id          string
	logger      *slog.Logger
}

// withClient adds the client to the context
func withClient(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, clientContextKey{}, client)
}

// ClientFromContext retrieves the client from the context
func ClientFromContext(ctx context.Context) (*Client, bool) {
	client, ok := ctx.Value(clientContextKey{}).(*Client)
	return client, ok
}

// ClientID retrieves the client ID from the context
func ClientID(ctx context.Context) string {
	if client, ok := ClientFromContext(ctx); ok {
		return client.id
	}
	return ""
}

func (c *Client) readPump() {
	defer func() {
		c.logger.Error("client read pump exited")
		c.cancel()
		c.hub.unregister <- c
	}()

	for {
		msgType, message, err := c.conn.Read(c.ctx)
		if err != nil {
			c.logger.Error("read error", slog.String("error", err.Error()))
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				c.logger.Error("websocket error", slog.String("error", err.Error()))
			}
			break
		}
		if msgType != websocket.MessageText {
			c.sendError(nil, -32600, "Invalid message type")
			continue
		}

		req, err := FromJSON[wsRequest](message)
		if err != nil {
			c.logger.Warn("parse error", slog.String("error", err.Error()))
			c.sendError(nil, -32700, "Parse error")
			continue
		}

		c.logger.Debug("request received",
			slog.String("method", req.Method),
			slog.Any("id", req.ID))

		// Handle the request
		go c.handleRequest(req)
	}
}

func (c *Client) writePump() {
	defer func() {
		c.cancel()
		close(c.sendChannel)
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.sendChannel:
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

func (c *Client) handleRequest(req wsRequest) {
	// Add client to the context
	ctx := withClient(c.ctx, c)

	// Fetch the handler
	c.hub.methodsMutex.RLock()
	method, exists := c.hub.methods[req.Method]
	c.hub.methodsMutex.RUnlock()
	if !exists {
		// If its a notification, do nothing
		if req.IsNotification() {
			return
		}
		c.sendError(req.ID, -32601, "Method not found")
		return
	}

	typedParams, err := method.parser(req.Params)
	if err != nil {
		c.logger.Error("unmarshal error",
			slog.String("method", req.Method),
			slog.String("error", err.Error()))
		c.sendError(req.ID, -32603, "Internal error")
		return
	}

	// Call the handler
	result, err := method.handler(ctx, typedParams)
	if err != nil {
		c.logger.Error("handler error",
			slog.String("method", req.Method),
			slog.String("error", err.Error()))
		c.sendError(req.ID, -32603, err.Error())
		return
	}

	// If it's notification don't send a response
	if req.IsNotification() {
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

	resp := wsResponse{ID: id, Result: data}
	c.sendData(resp)
}

func (c *Client) sendError(id *int, code int, message string) {
	resp := wsResponse{
		ID:    id,
		Error: &wsErrorObj{Code: code, Message: message},
	}
	c.sendData(resp)
}

func (c *Client) sendData(r wsResponse) {
	msg, err := ToJSON(r)
	if err != nil {
		c.logger.Error("failed to encode response", slog.String("error", err.Error()))
		return
	}

	// Send the message on the send channel or cancel if the context is done
	select {
	case c.sendChannel <- msg:
	case <-c.ctx.Done():
	}
}
