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
	"ws-json-rpc/pkg/rpc"
	"ws-json-rpc/pkg/rpc/generate"
	mw "ws-json-rpc/pkg/rpc/middleware"

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
	// vm, _ := bindings.New()
	// golang, _ := guts.NewGolangParser()
	// _ = golang.IncludeGenerate("./internal/handlers")
	// _ = golang.IncludeCustom(map[guts.GolangType]guts.GolangType{
	// 	"time.Time": "string",
	// })
	// ts, _ := golang.ToTypescript()
	// ts.ApplyMutations(
	// 	config.ExportTypes,
	// 	config.EnumAsTypes,
	// )

	// keys := []string{}
	// typeMap := map[string]string{}

	// ts.ForEach(func(key string, node bindings.Node) {
	// 	obj, _ := vm.ToTypescriptNode(node)
	// 	text, _ := vm.SerializeToTypescript(obj)
	// 	keys = append(keys, key)
	// 	typeMap[key] = text
	// })
	// sort.Strings(keys)
	// for _, key := range keys {
	// 	fmt.Println(typeMap[key])
	// }

	gen := generator.NewGoParser(&generator.GoParserOptions{
		PrintParsedTypes: false,
	})
	if err := gen.AddDir("./internal/handlers"); err != nil {
		log.Fatal(err)
	}
	if err := gen.AddDir("./internal/consts"); err != nil {
		log.Fatal(err)
	}
	types, err := gen.Run()
	if err != nil {
		log.Fatal(err)
	}

	tsGen := generator.NewTSGenerator(&generator.TSGeneratorOptions{
		GenerateEnumValues: true,
	}, types)
	tsMap := tsGen.GetRenderedTypes()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: slogReplacer,
	}))

	hub := rpc.NewHub(logger, generate.NewGenerator(tsMap))
	rpc.RegisterEvent[handlers.UserUpdateEventResponse](hub, generate.EventDocs{}, consts.EventKindUserUpdate.String())
	rpc.RegisterEvent[handlers.UserLoginEventResponse](hub, generate.EventDocs{}, consts.EventKindUserLogin.String())

	methods := handlers.NewHandlers(hub)
	hub.WithMiddleware(mw.LoggingMiddleware)
	rpc.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindSubscribe.String(), methods.Subscribe)
	rpc.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindUnsubscribe.String(), methods.Unsubscribe)
	rpc.RegisterMethod(hub, generate.HandlerDocs{
		Title:       "Ping",
		Group:       "Utility",
		Description: "Ping the server to check connectivity.",
		Examples: []generate.HandlerExample{
			{
				Title:       "Ping",
				Description: "Ping the server",
				Params:      struct{}{},
				Result:      handlers.PingResult{Message: "pong", Status: handlers.StatusOK},
			},
		},
	}, consts.MethodKindPing.String(), methods.Ping)
	rpc.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindEcho.String(), methods.Echo)
	rpc.RegisterMethod(hub, generate.HandlerDocs{}, consts.MethodKindAdd.String(), methods.Add)
	rpc.RegisterMethod(hub, generate.HandlerDocs{
		Title:       "Double",
		Group:       "Math",
		Description: "Adds and then doubles the result of the addition",
		Examples: []generate.HandlerExample{
			{
				Title:       "Add and double",
				Description: "Adds 5 and 5, then doubles the result",
				Params:      handlers.DoubleParams{Value: 5, Other: 5},
				Result:      handlers.DoubleResult{Result: 20},
			},
		},
	}, consts.MethodKindDouble.String(), methods.Double)
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

func simulate(h *rpc.Hub) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.PublishEvent(rpc.NewEvent(consts.EventKindUserUpdate.String(), handlers.UserUpdateEventResponse{
			ID:   uuid.New().String(),
			Name: "Alice",
		}))
	}
}
