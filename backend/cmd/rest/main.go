package main

import (
	"log/slog"
	"net/http"
	"ws-json-rpc/backend/internal/httpapi"
	"ws-json-rpc/backend/pkg/router"
)

func main() {
	rb, err := router.NewRouteBuilder(slog.Default(), router.RouteBuilderOptions{TypesDirectory: "backend/internal/httpapi"})
	if err != nil {
		panic(err)
	}
	rb.Get("/ping", router.RouteSpec{
		OperationID: "ping",
		Summary:     "Ping the server",
		Description: "Check if the server is alive",
		Tags:        []string{"ping"},
		RequestType: nil,
		Responses: map[int]router.ResponseSpec{
			200: {
				Description: "Successful ping response",
				Type:        httpapi.PingResponse{},
				Examples: map[string]any{
					"example-1": httpapi.PingResponse{Message: "Pong", Status: httpapi.PingStatusOK},
				},
			},
			500: {
				Description: "Internal server error",
				Type:        httpapi.PingResponse{},
			},
		},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	})

	rb.WriteSpecYAML("test.yaml")
}
