package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"
	"ws-json-rpc/internal/consts"
	"ws-json-rpc/internal/handlers"
	"ws-json-rpc/pkg/ws"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	hub := ws.NewHub(logger)
	hub.RegisterEvent(consts.EventKindUserUpdate)
	hub.RegisterEvent(consts.EventKindUserLogin)

	handlers := handlers.NewHandlers(hub)

	// Register handlers
	ws.RegisterHandler(hub, consts.MethodKindEcho, handlers.Echo)
	ws.RegisterHandler(hub, consts.MethodKindAdd, handlers.Add)
	ws.RegisterHandler(hub, consts.MethodKindDouble, handlers.Double)
	ws.RegisterHandler(hub, consts.MethodKindSubscribe, handlers.Subscribe)
	ws.RegisterHandler(hub, consts.MethodKindUnsubscribe, handlers.Unsubscribe)
	ws.RegisterHandler(hub, consts.MethodKindPing, handlers.Ping)

	go hub.Run()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			hub.PublishEvent(ws.NewEvent(consts.EventKindUserLogin, map[string]string{"id": "456", "name": "Alice"}))
		}
	}()

	http.HandleFunc("/ws", hub.ServeWS())
	logger.Info("WebSocket server starting", slog.String("address", ":8080"))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}
