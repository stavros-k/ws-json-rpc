package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

type WSHandlerFunc[T any] func(ctx context.Context, params T) (any, error)

type jsonRPCRequest struct {
	Version string          `json:"jsonrpc"`
	ID      *string         `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"` // json.RawMessage is used to defer JSON unmarshaling for params
}

type jsonRPCResponse struct {
	Version string        `json:"jsonrpc"`
	ID      *string       `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

type methodHandler interface {
	handle(ctx context.Context, params json.RawMessage) (any, error)
}

// Generic typed handler wrapper
type typedHandler[T any] struct {
	handlerFunc WSHandlerFunc[T]
}

func (h *typedHandler[T]) handle(ctx context.Context, params json.RawMessage) (any, error) {
	typedParams, err := fromJSON[T](bytes.NewReader(params))
	if err != nil {
		return nil, newJSONRPCError(codeInvalidParams, "Invalid parameters: "+err.Error())
	}

	return h.handlerFunc(ctx, typedParams)
}

type WSServer struct {
	methods         map[string]methodHandler
	wsAcceptOptions *websocket.AcceptOptions
}

func NewServer() *WSServer {
	return &WSServer{
		methods: make(map[string]methodHandler),
		wsAcceptOptions: &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionContextTakeover,
			OnPingReceived: func(ctx context.Context, p []byte) bool {
				log.Printf("ping received: %s", string(p))
				return true
			},
			OnPongReceived: func(ctx context.Context, p []byte) {
				log.Printf("pong received: %s", string(p))
			},
		},
	}
}

// Generic registration function - type safe handlers, no reflection
func Register[T any](s *WSServer, methodName string, handler WSHandlerFunc[T]) {
	if _, exists := s.methods[methodName]; exists {
		panic(fmt.Sprintf("method %s already registered", methodName))
	}

	s.methods[methodName] = &typedHandler[T]{handlerFunc: handler}
}

// ServeHTTP is called for each new connection
func (s *WSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, s.wsAcceptOptions)
	if err != nil {
		log.Printf("websocket accept failed: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "connection closed")

	s.handleConnection(r.Context(), conn)
}

// handleConnection loops on messages from the connection until it is closed
func (s *WSServer) handleConnection(ctx context.Context, conn *websocket.Conn) {
	for {
		err := s.handleMessage(ctx, conn)
		// No error, go to next message
		if err == nil {
			continue
		}

		// Context canceled, log and return (stop loop)
		if errors.Is(err, context.Canceled) {
			log.Printf("context canceled: %s", err.Error())
			return
		}
		// Connection closed, log and return (stop loop)
		var closeError websocket.CloseError
		if errors.As(err, &closeError) {
			log.Printf("connection close: %s", err.Error())
			return
		}

		// Other error, log and continue
		log.Printf("connection error: %v", err)
	}
}

func (s *WSServer) handleMessage(ctx context.Context, conn *websocket.Conn) error {
	// Read message
	msgType, data, err := conn.Read(ctx)
	if err != nil {
		return fmt.Errorf("read message: %w", err)
	}

	// Discard messages that are not text
	if msgType != websocket.MessageText {
		return fmt.Errorf("unexpected message type: %v", msgType)
	}

	req, err := fromJSON[jsonRPCRequest](bytes.NewReader(data))
	if err != nil {
		return s.sendResponse(ctx, conn, req.ID, nil, &jsonRPCError{Code: codeParseError, Message: "Parse error"})
	}

	// Validate JSON-RPC request
	if req.Version != "2.0" {
		return s.sendResponse(ctx, conn, req.ID, nil, &jsonRPCError{Code: codeInvalidRequest, Message: "Invalid Request"})
	}

	// Find method handler
	handler, exists := s.methods[req.Method]
	if !exists {
		return s.sendResponse(ctx, conn, req.ID, nil, &jsonRPCError{Code: codeMethodNotFound, Message: "Method not found"})
	}

	result, err := handler.handle(ctx, req.Params)
	return s.sendResponse(ctx, conn, req.ID, result, toRPCError(err))
}

// sendResponse sends a JSON-RPC response to the connection
func (s *WSServer) sendResponse(ctx context.Context, conn *websocket.Conn, id *string, res any, rpcErr *jsonRPCError) error {
	// Create the writer
	writer, err := conn.Writer(ctx, websocket.MessageText)
	if err != nil {
		return err
	}
	defer writer.Close()

	rpcResponse := jsonRPCResponse{Version: "2.0", ID: id, Error: rpcErr}
	// If there is an rpcErr, we send it without a result
	if rpcResponse.Error != nil {
		return toJSON(writer, rpcResponse)
	}

	// Attach the result
	rpcResponse.Result = res
	// Send the response
	return toJSON(writer, rpcResponse)
}
