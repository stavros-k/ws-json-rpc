package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"
	"ws-json-rpc/internal/consts"
	"ws-json-rpc/internal/handlers"
	"ws-json-rpc/pkg/ws"
	mw "ws-json-rpc/pkg/ws/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	hub := ws.NewHub(logger)
	hub.RegisterEvent(consts.EventKindUserUpdate.String())
	hub.RegisterEvent(consts.EventKindUserLogin.String())

	handlers := handlers.NewHandlers(hub)
	ws.RegisterMethod(hub, consts.MethodKindSubscribe.String(), handlers.Subscribe)
	ws.RegisterMethod(hub, consts.MethodKindUnsubscribe.String(), handlers.Unsubscribe)
	ws.RegisterMethod(hub, consts.MethodKindPing.String(), handlers.Ping)
	ws.RegisterMethod(hub, consts.MethodKindEcho.String(), handlers.Echo, mw.LoggingMiddleware(logger))
	ws.RegisterMethod(hub, consts.MethodKindAdd.String(), handlers.Add)
	ws.RegisterMethod(hub, consts.MethodKindDouble.String(), handlers.Double)
	go hub.Run()
	go simulate(hub)

	http.HandleFunc("/ws", hub.ServeWS())
	logger.Info("WebSocket server starting", slog.String("address", ":8080"))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}

func simulate(h *ws.Hub) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.PublishEvent(ws.NewEvent(consts.EventKindUserLogin.String(), map[string]string{"id": "456", "name": "Alice"}))
	}
}
