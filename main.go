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

func slogReplacer(groups []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		a.Value = slog.StringValue(time.Now().Format("2006-01-02 15:04:05"))
	}

	return a
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: slogReplacer,
	}))

	hub := ws.NewHub(logger)
	hub.RegisterEvent(consts.EventKindUserUpdate.String())
	hub.RegisterEvent(consts.EventKindUserLogin.String())

	handlers := handlers.NewHandlers(hub)
	hub.WithMiddleware(mw.LoggingMiddleware)
	ws.RegisterMethod(hub, consts.MethodKindSubscribe.String(), handlers.Subscribe)
	ws.RegisterMethod(hub, consts.MethodKindUnsubscribe.String(), handlers.Unsubscribe)
	ws.RegisterMethod(hub, consts.MethodKindPing.String(), handlers.Ping)
	ws.RegisterMethod(hub, consts.MethodKindEcho.String(), handlers.Echo)
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
		h.PublishEvent(ws.NewEvent(consts.EventKindUserUpdate.String(), map[string]string{"id": "456", "name": "Alice"}))
	}
}
