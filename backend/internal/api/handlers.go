package api

import (
	"net/http"
	"ws-json-rpc/backend/pkg/apitypes"
)

func (s *Server) Ping(w http.ResponseWriter, r *http.Request) error {
	RespondJSON(w, r, http.StatusOK, apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK})
	return nil
}
