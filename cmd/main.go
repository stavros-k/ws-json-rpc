package main

import (
	"context"
	"log"
	"net/http"
	"ws-json-rpc/pkg/ws"
)

func getUserHandler(ctx context.Context, params GetUserRequest) (any, error) {
	if params.UserID <= 0 {
		return nil, BadRequest("Invalid user ID")
	}

	return User{ID: params.UserID, Name: "John Doe", Email: "john@example.com"}, nil
}

func addHandler(ctx context.Context, params AddRequest) (any, error) {
	if params.A < 0 || params.B < 0 {
		return nil, BadRequest("Numbers must be positive")
	}
	return map[string]int{"result": params.A + params.B}, nil
}

func main() {
	server := ws.NewServer()

	// Type-safe handlers - params are automatically unmarshaled to correct type
	ws.Register(server, "add", addHandler)
	ws.Register(server, "getUser", getUserHandler)

	http.Handle("/ws", server)
	log.Println("WebSocket JSON-RPC server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
