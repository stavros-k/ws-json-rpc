package ws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/coder/websocket"
)

const (
	ErrCodeParse    = -32700 // "Invalid JSON was received by the server. An error occurred on the server while parsing the JSON text."}
	ErrCodeInvalid  = -32600 // "The JSON sent is not a valid Request object."}
	ErrCodeNotFound = -32601 // "The method does not exist / is not available."}
	ErrCodeInternal = -32603 // "Internal JSON-RPC error."}
)

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

func (c *Client) ID() string {
	return c.id
}

func (c *Client) RemoteAddr() string {
	return c.remoteAddr
}

func (c *Client) readPump() {
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
				c.logger.Info("websocket closed abruptly", slog.String("error", err.Error()))
			default:
				c.logger.Error("unknown websocket error", slog.String("error", err.Error()))
			}
			break
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
				c.logger.Error("write error", slog.String("error", err.Error()))
				continue
			}
		}
	}
}

func (c *Client) handleRequest(req wsRequest) {
	// Derive a logger from the original for this request
	reqLogger := c.logger.With(slog.String("method", req.Method))
	if req.ID != nil {
		reqLogger = reqLogger.With(slog.Int("id", *req.ID))
	}

	reqLogger.Debug("handling request")

	// Get the handler
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

	// Parse json into the structured params
	typedParams, err := method.parser(req.Params)
	if err != nil {
		reqLogger.Error("unmarshal error", slog.String("error", err.Error()))
		if err := c.sendError(req.ID, ErrCodeInternal, fmt.Sprintf("Failed to parse params on method %q: %s", req.Method, err.Error())); err != nil {
			reqLogger.Error("failed to send error response", slog.String("error", err.Error()))
		}
		return
	}

	// Set a timeout for the request
	ctx, cancel := context.WithTimeout(c.ctx, MAX_REQUEST_TIMEOUT)
	defer cancel()

	// Create a new HandlerContext
	hctx := &HandlerContext{Client: c, Logger: reqLogger}

	// Call the handler
	result, err := method.handler(ctx, hctx, typedParams)
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
