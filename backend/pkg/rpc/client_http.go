package rpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"ws-json-rpc/backend/pkg/utils"

	"github.com/google/uuid"
)

type HTTPClient struct {
	w          http.ResponseWriter
	r          *http.Request
	hub        *Hub
	remoteHost string
	id         string
	logger     *slog.Logger
}

func (c *HTTPClient) handleRequest(ctx context.Context, req RPCRequest) {
	reqLogger := c.logger.With(slog.String("method", req.Method))
	reqLogger = reqLogger.With(slog.String("id", req.ID.String()))

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
		reqLogger.Error("unmarshal error", utils.ErrAttr(err))
		c.sendError(req.ID, ErrCodeInvalidParams, fmt.Sprintf("Failed to parse params on method %q: %s", req.Method, err.Error()))

		return
	}

	// Set a timeout for the request
	ctx, cancel := context.WithTimeout(ctx, MAX_REQUEST_TIMEOUT)
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
		hctx.Logger.Error("handler error", utils.ErrAttr(err))
		// If its a handler error, let handler specify code/message
		var he HandlerError
		if errors.As(err, &he) {
			c.sendError(req.ID, he.Code(), he.Error())

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

	if err := utils.ToJSONStream(c.w, resp); err != nil {
		c.logger.Error("failed to encode HTTP response", utils.ErrAttr(err))
	}
}

// ServeHTTP handles HTTP JSON-RPC requests.
func (h *Hub) ServeHTTP() http.HandlerFunc {
	httpLogger := h.logger.With(slog.String("handler", "http"))

	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST and GET requests
		if r.Method != http.MethodPost {
			httpLogger.Warn("http request not allowed", slog.String("method", r.Method))
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

			return
		}

		// Limit the size of the request body
		r.Body = http.MaxBytesReader(w, r.Body, MAX_MESSAGE_SIZE)

		// Parse the request using streaming JSON helper
		req, err := utils.FromJSONStream[RPCRequest](r.Body)
		if err != nil {
			// Create a minimal error response
			resp := NewRPCResponse(uuid.Nil, nil, &RPCErrorObj{Code: ErrCodeParse, Message: "Invalid JSON in request body"})
			w.Header().Set("Content-Type", "application/json")
			if err := utils.ToJSONStream(w, resp); err != nil {
				// Log the error but cannot do much else
				httpLogger.Error("failed to encode HTTP response", utils.ErrAttr(err))
			}

			return
		}

		remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			httpLogger.Error("failed to parse remote address", utils.ErrAttr(err), slog.String("remote_addr", r.RemoteAddr))

			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		clientID := r.Header.Get("X-Client-ID")
		if clientID == "" {
			httpLogger.Warn("no client ID provided, generating one", slog.String("remote_addr", remoteHost))
			clientID = fmt.Sprintf("http-%s-%s", remoteHost, uuid.NewString())
		}

		client := &HTTPClient{
			w:          w,
			r:          r,
			hub:        h,
			remoteHost: remoteHost,
			id:         clientID,
			logger: httpLogger.With(
				slog.String("client_id", clientID),
				slog.String("remote_host", remoteHost),
			),
		}

		// Handle the request
		client.handleRequest(ctx, req)
	}
}
