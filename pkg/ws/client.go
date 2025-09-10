package ws

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/coder/websocket"
)

const (
	MAX_REQUEST_TIMEOUT  = 30 * time.Second
	MAX_RESPONSE_TIMEOUT = 30 * time.Second
)

const (
	ErrCodeParse    = -32700 // "Invalid JSON was received by the server. An error occurred on the server while parsing the JSON text."}
	ErrCodeInvalid  = -32600 // "The JSON sent is not a valid Request object."}
	ErrCodeNotFound = -32601 // "The method does not exist / is not available."}
	ErrCodeInternal = -32603 // "Internal JSON-RPC error."}
)

// clientContextKey is a key used for storing the client in the context
type clientContextKey struct{}

type ClientContext struct {
	Logger *slog.Logger
	Client *Client
}

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

func withClientContext(ctx context.Context, client *Client, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, clientContextKey{}, &ClientContext{
		Logger: logger,
		Client: client,
	})
}

func ClientContextFromContext(ctx context.Context) (*ClientContext, bool) {
	clientContext, ok := ctx.Value(clientContextKey{}).(*ClientContext)
	if !ok {
		return nil, false
	}
	return clientContext, true
}

func (c *Client) readPump() {
	// When readPump exits, cancel the context and unregister the client
	defer func() {
		c.logger.Error("client read pump exited")
		c.cancel()
		c.hub.unregister <- c
	}()

	for {
		// Read next message
		msgType, message, err := c.conn.Read(c.ctx)
		if err != nil {
			// In case of a ws close error, stop the loop
			var ce websocket.CloseError
			if errors.As(err, &ce) {
				c.logger.Info("websocket closed", slog.Int("code", int(ce.Code)), slog.String("text", ce.Reason))
				break
			}
			c.logger.Error("websocket error", slog.String("error", err.Error()))
		}
		// Only support text based messages
		if msgType != websocket.MessageText {
			if err := c.sendError(nil, ErrCodeInvalid, "Invalid message type. Only text messages are supported."); err != nil {
				c.logger.Error("failed to send error response", slog.String("error", err.Error()))
			}
			continue
		}

		// Parse message
		req, err := FromJSON[wsRequest](message)
		if err != nil {
			c.logger.Warn("parse error", slog.String("error", err.Error()))
			if err := c.sendError(nil, ErrCodeParse, err.Error()); err != nil {
				c.logger.Error("failed to send error response", slog.String("error", err.Error()))
			}
			continue
		}

		// Handle the request
		go c.handleRequest(req)
	}
}

func (c *Client) writePump() {
	// When writePump exits, cancel the context and close the send channel
	defer func() {
		c.logger.Error("client write pump exited")
		c.cancel()
		close(c.sendChannel)
	}()

	for {
		select {
		case <-c.ctx.Done():
			// Close connection on context cancellation
			c.conn.Close(websocket.StatusNormalClosure, "")
			return
		case message, ok := <-c.sendChannel:
			// If the send channel is closed, close the connection
			// Shouldn't happen, but just in case
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			// Write message with a timeout
			ctx, cancel := context.WithTimeout(c.ctx, MAX_RESPONSE_TIMEOUT)
			err := c.conn.Write(ctx, websocket.MessageText, message)
			cancel()

			if err != nil {
				c.logger.Error("write error", slog.String("error", err.Error()))
				continue
			}
		}
	}
}

func (c *Client) handleRequest(req wsRequest) {
	reqLogger := c.logger.With(slog.String("method", req.Method))
	if req.ID != nil {
		reqLogger = reqLogger.With(slog.Int("id", *req.ID))
	}
	ctx := withClientContext(c.ctx, c, reqLogger)
	reqLogger.Debug("handling request")

	// Fetch the handler
	c.hub.methodsMutex.RLock()
	method, exists := c.hub.methods[req.Method]
	c.hub.methodsMutex.RUnlock()
	if !exists {
		// If its a notification, do nothing
		if req.IsNotification() {
			return
		}
		if err := c.sendError(req.ID, ErrCodeNotFound, fmt.Sprintf("Method %q not found", req.Method)); err != nil {
			reqLogger.Error("failed to send error response", slog.String("error", err.Error()))
		}
		return
	}

	typedParams, err := method.parser(req.Params)
	if err != nil {
		reqLogger.Error("unmarshal error", slog.String("error", err.Error()))
		if err := c.sendError(req.ID, ErrCodeInternal, fmt.Sprintf("Failed to parse params on method %q: %s", req.Method, err.Error())); err != nil {
			reqLogger.Error("failed to send error response", slog.String("error", err.Error()))
		}
		return
	}

	ctx, cancel := context.WithTimeout(ctx, MAX_REQUEST_TIMEOUT)
	defer cancel()
	// Call the handler
	result, err := method.handler(ctx, typedParams)
	if err != nil {
		reqLogger.Error("handler error", slog.String("error", err.Error()))
		if err := c.sendError(req.ID, ErrCodeInternal, fmt.Sprintf("Failed to handle request on method %q: %s", req.Method, err.Error())); err != nil {
			reqLogger.Error("failed to send error response", slog.String("error", err.Error()))
		}
		return
	}

	// If it's notification don't send a response
	if req.IsNotification() {
		return
	}

	if err := c.sendSuccess(req.ID, result); err != nil {
		reqLogger.Error("failed to send success response", slog.String("error", err.Error()))
	}
}

func (c *Client) sendSuccess(id *int, result any) error {
	data, err := ToJSON(result)
	if err != nil {
		return err
	}

	resp := wsResponse{ID: id, Result: data}
	return c.sendData(resp)
}

func (c *Client) sendError(id *int, code int, message string) error {
	resp := wsResponse{
		ID:    id,
		Error: &wsErrorObj{Code: code, Message: message},
	}
	return c.sendData(resp)
}

func (c *Client) sendData(r wsResponse) error {
	msg, err := ToJSON(r)
	if err != nil {
		return err
	}

	// Send the message on the send channel or cancel if the context is done
	select {
	case c.sendChannel <- msg:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}
