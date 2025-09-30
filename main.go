package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
	"ws-json-rpc/internal/consts"
	"ws-json-rpc/internal/handlers"
	"ws-json-rpc/pkg/generator"
	"ws-json-rpc/pkg/ws"
	"ws-json-rpc/pkg/ws/generate"
	mw "ws-json-rpc/pkg/ws/middleware"

	"github.com/google/uuid"
)

func slogReplacer(groups []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		a.Value = slog.StringValue(time.Now().Format("2006-01-02 15:04:05"))
	}

	return a
}

func main() {
	gen := generator.NewGoParser(&generator.GoParserOptions{
		PrintParsedTypes: true,
	})
	if err := gen.AddDir("./test_data"); err != nil {
		log.Fatal(err)
	}
	types, err := gen.Run()
	if err != nil {
		log.Fatal(err)
	}
	tsGen := generator.NewTSGenerator(&generator.TSGeneratorOptions{
		GenerateEnumValues: true,
	})
	tsGen.Generate(types)
	os.Exit(0)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: slogReplacer,
	}))

	hub := ws.NewHub(logger)
	ws.RegisterEvent[handlers.UserUpdateEventResponse](hub, generate.EventDocs{}, consts.EventKindUserUpdate.String())
	ws.RegisterEvent[handlers.UserLoginEventResponse](hub, generate.EventDocs{}, consts.EventKindUserLogin.String())

	handlers := handlers.NewHandlers(hub)
	hub.WithMiddleware(mw.LoggingMiddleware)
	ws.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindSubscribe.String(), handlers.Subscribe)
	ws.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindUnsubscribe.String(), handlers.Unsubscribe)
	ws.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindPing.String(), handlers.Ping)
	ws.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindEcho.String(), handlers.Echo)
	ws.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindAdd.String(), handlers.Add)
	ws.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindDouble.String(), handlers.Double)
	go hub.Run()
	go simulate(hub)

	hub.G()

	http.HandleFunc("/ws", hub.ServeWS())
	http.HandleFunc("/rpc", hub.ServeHTTP())
	logger.Info("WebSocket server starting", slog.String("address", ":8080"))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}

func simulate(h *ws.Hub) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.PublishEvent(ws.NewEvent(consts.EventKindUserUpdate.String(), handlers.UserUpdateEventResponse{
			ID:   uuid.New().String(),
			Name: "Alice",
		}))
	}
}
