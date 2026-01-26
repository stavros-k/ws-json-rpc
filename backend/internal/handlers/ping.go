package handlers

import (
	"ws-json-rpc/backend/internal/api"
	"ws-json-rpc/backend/pkg/apitypes"
	"ws-json-rpc/backend/pkg/router"
)

func RegisterPing(path string, rb *router.RouteBuilder, server *api.Server) {
	rb.MustGet(path, router.RouteSpec{
		OperationID: "ping",
		Summary:     "Ping the server",
		Description: "Check if the server is alive",
		Group:       "Core",
		RequestType: nil,
		Handler:     api.ErrorHandler(server.Ping),
		Responses: api.GenerateResponses(map[int]router.ResponseSpec{
			200: {
				Description: "Successful ping response",
				Type:        apitypes.PingResponse{},
				Examples: map[string]any{
					"Success": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				},
			},
		}),
	})
}
