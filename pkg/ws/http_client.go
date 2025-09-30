package ws

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type HTTPClient struct {
	hub        *Hub
	remoteHost string
	ctx        context.Context
	cancel     context.CancelFunc
	id         string
	logger     *slog.Logger
}

// sendHTTPSuccess sends a successful JSON-RPC response over HTTP
func (h *Hub) sendHTTPSuccess(w http.ResponseWriter, id uuid.UUID, result any) {
	data, err := ToJSON(result)
	if err != nil {
		h.sendHTTPError(w, id, ErrCodeInternal, "Failed to serialize response")
		return
	}

	resp := RPCResponse{ID: id, Result: data}
	h.sendHTTPResponse(w, resp)
}

// sendHTTPError sends an error JSON-RPC response over HTTP
func (h *Hub) sendHTTPError(w http.ResponseWriter, id uuid.UUID, code int, message string) {
	resp := RPCResponse{
		ID:    id,
		Error: &RPCErrorObj{Code: code, Message: message},
	}
	h.sendHTTPResponse(w, resp)
}

// sendHTTPResponse sends a JSON-RPC response over HTTP using streaming JSON helper
func (h *Hub) sendHTTPResponse(w http.ResponseWriter, resp RPCResponse) {
	w.Header().Set("Content-Type", "application/json")

	if err := ToJSONStream(w, resp); err != nil {
		h.logger.Error("failed to encode HTTP response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
