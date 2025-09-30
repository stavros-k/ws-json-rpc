package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type HTTPClient struct {
	w          http.ResponseWriter
	r          *http.Request
	hub        *Hub
	remoteHost string
	ctx        context.Context
	cancel     context.CancelFunc
	id         string
	logger     *slog.Logger
}

func (c *HTTPClient) handleRequest(req RPCRequest) {
	reqLogger := c.logger.With(slog.String("method", req.Method))
	reqLogger = reqLogger.With(slog.String("id", req.ID.String()))

	reqLogger.Debug("handling HTTP JSON-RPC request")

	// Get the handler
	c.hub.methodsMutex.RLock()
	method, exists := c.hub.methods[req.Method]
	c.hub.methodsMutex.RUnlock()
	if !exists {
		c.sendError(req.ID, ErrCodeNotFound, fmt.Sprintf("Method %q not found", req.Method))
		return
	}

	// Parse json into the structured params
	typedParams, err := method.parser(req.Params)
	if err != nil {
		reqLogger.Error("unmarshal error", slog.String("error", err.Error()))
		c.sendError(req.ID, ErrCodeInternal, fmt.Sprintf("Failed to parse params on method %q: %s", req.Method, err.Error()))
		return
	}

	// Set a timeout for the request
	ctx, cancel := context.WithTimeout(c.r.Context(), MAX_REQUEST_TIMEOUT)
	defer cancel()

	// Create a new HandlerContext
	hctx := &HandlerContext{
		Logger:   reqLogger,
		WSConn:   nil,
		HTTPConn: c,
	}

	// Call the handler
	result, err := method.handler(ctx, hctx, typedParams)
	if err != nil {
		reqLogger.Error("handler error", slog.String("error", err.Error()))
		// If its a handler error, let handler specify code/message
		if err, ok := err.(HandlerError); ok {
			c.sendError(req.ID, err.Code(), err.Error())
			return
		}

		// Unknown errors, send internal error
		c.sendError(req.ID, ErrCodeInternal, fmt.Sprintf("Failed to handle request on method %q: %s", req.Method, err.Error()))
		return
	}

	c.sendSuccess(req.ID, result)
}

func (c *HTTPClient) sendSuccess(id uuid.UUID, result any) {
	c.sendResponse(NewRPCResponse(id, result, nil))
}

func (c *HTTPClient) sendError(id uuid.UUID, code int, message string) {
	c.sendResponse(NewRPCResponse(id, nil, &RPCErrorObj{Code: code, Message: message}))
}

func (c *HTTPClient) sendResponse(resp RPCResponse) {
	c.w.Header().Set("Content-Type", "application/json")

	if err := ToJSONStream(c.w, resp); err != nil {
		c.logger.Error("failed to encode HTTP response", slog.String("error", err.Error()))
		http.Error(c.w, "Internal server error", http.StatusInternalServerError)
	}
}
