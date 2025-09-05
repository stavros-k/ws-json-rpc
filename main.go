package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"
	"ws-json-rpc/pkg/ws"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	hub := ws.NewHub(logger)
	hub.RegisterEvent(EventKindUserUpdate)
	hub.RegisterEvent(EventKindUserLogin)

	handlers := &Handlers{
		hub: hub,
	}

	// Register handlers
	ws.RegisterHandler(hub, MethodKindEcho, handlers.Echo)
	ws.RegisterHandler(hub, MethodKindAdd, handlers.Add)
	ws.RegisterHandler(hub, MethodKindDouble, handlers.Double)
	ws.RegisterHandler(hub, MethodKindSubscribe, handlers.Subscribe)
	ws.RegisterHandler(hub, MethodKindUnsubscribe, handlers.Unsubscribe)
	ws.RegisterHandler(hub, MethodKindPing, handlers.Ping)

	go hub.Run()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			hub.PublishEvent(ws.NewEvent(EventKindUserLogin, map[string]string{"id": "456", "name": "Alice"}))
		}
	}()

	http.HandleFunc("/ws", hub.ServeWS())
	logger.Info("WebSocket server starting", slog.String("address", ":8080"))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}
