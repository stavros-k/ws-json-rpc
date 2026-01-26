package api

import (
	"net/http"
	"ws-json-rpc/backend/pkg/apitypes"
	"ws-json-rpc/backend/pkg/router"
)

func (s *Server) Ping(w http.ResponseWriter, r *http.Request) error {
	if err := s.db.Ping(); err != nil {
		RespondJSON(w, r, http.StatusInternalServerError, apitypes.PingResponse{
			Message: "Database unreachable", Status: apitypes.PingStatusError,
		})
		return nil
	}

	RespondJSON(w, r, http.StatusOK, apitypes.PingResponse{
		Message: "Pong", Status: apitypes.PingStatusOK,
	})
	return nil
}

func RegisterPing(path string, rb *router.RouteBuilder, s *Server) {
	rb.MustGet(path, router.RouteSpec{
		OperationID: "ping",
		Summary:     "Ping the server",
		Description: "Check if the server is alive",
		Group:       CoreGroup,
		RequestType: nil,
		Handler:     ErrorHandler(s.Ping),
		Responses: GenerateResponses(map[int]router.ResponseSpec{
			200: {
				Description: "Successful ping response",
				Type:        apitypes.PingResponse{},
				Examples: map[string]any{
					"Success": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				},
			},
			500: {
				Description: "Database unreachable",
				Type:        apitypes.PingResponse{},
				Examples: map[string]any{
					"Database unreachable": apitypes.PingResponse{Message: "Database unreachable", Status: apitypes.PingStatusError},
				},
			},
		}),
	})
}
