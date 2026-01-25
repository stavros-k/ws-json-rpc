package httpapi

import (
	"net/http"
	"ws-json-rpc/backend/pkg/utils"
)

func sendJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.WriteHeader(statusCode)
	if err := utils.ToJSONStream(w, data); err != nil {
		return err
	}
	return nil
}

type Handlers struct{}

func (h *Handlers) Ping(w http.ResponseWriter, r *http.Request) {
	resp := PingResponse{
		Message: "Pong",
		Status:  PingStatusOK,
	}
	if err := sendJSON(w, http.StatusOK, resp); err != nil {
		// FIXME: Handler should return error and have middleware handle it
		panic(err)
	}
}
