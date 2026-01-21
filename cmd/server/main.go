//go:generate bash -c "cd ../../; ./generate.sh"
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"ws-json-rpc/internal/app"
	"ws-json-rpc/internal/database/sqlite"
	"ws-json-rpc/internal/rpcapi"
	rpctypes "ws-json-rpc/internal/rpcapi/types"
	"ws-json-rpc/pkg/database"
	"ws-json-rpc/pkg/rpc"
	"ws-json-rpc/pkg/rpc/generate"
	"ws-json-rpc/pkg/rpc/middleware"
	"ws-json-rpc/pkg/utils"
	"ws-json-rpc/web"

	"github.com/google/uuid"
)

const (
	shutdownTimeout = 30 * time.Second
)

func main() {
	config, err := app.NewConfig()
	if err != nil {
		fatalIfErr(slog.Default(), fmt.Errorf("failed to create config: %w", err))
	}
	defer config.Close()

	logger := getLogger(config)

	g, err := generator(config, logger)
	if err != nil {
		fatalIfErr(logger, fmt.Errorf("failed to create generator: %w", err))
	}

	hub := rpc.NewHub(logger, g)
	mux := http.NewServeMux()

	methods := rpcapi.NewHandlers(hub)
	hub.WithMiddleware(middleware.LoggingMiddleware)

	// Register events
	rpc.RegisterEvent[rpctypes.DataCreated](hub, string(rpctypes.EventKindDataCreated), rpc.EventOptions{
		Docs: generate.EventDocs{
			Title:       "DataCreated",
			Description: "Event fired when new data is created",
			Group:       "Data",
			Deprecated:  true,
			Examples: []generate.Example{
				{
					Title:       "Basic example",
					Description: "Subscribe to the DataCreated event",
					ResultObj:   rpctypes.DataCreated{ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")},
				},
			},
		},
	})

	// Register methods
	rpc.RegisterMethod(hub, string(rpctypes.MethodKindPing), methods.PingHandler, rpc.RegisterMethodOptions{
		Docs: generate.MethodDocs{
			Title:       "Ping",
			Description: "A simple ping method to check if the server is alive",
			Group:       "Core",
			Tags:        []string{"health", "status"},
			Examples: []generate.Example{
				{
					Title:       "Ping",
					Description: "Ping the server",
					ParamsObj:   nil,
					ResultObj:   rpctypes.PingResult{Message: "pong", Status: rpctypes.PingStatusSuccess},
				},
			},
		},
	})

	rpc.RegisterMethod(hub, string(rpctypes.MethodKindSubscribe), methods.Subscribe, rpc.RegisterMethodOptions{
		Docs: generate.MethodDocs{
			Title:       "Subscribe",
			Description: "Subscribe to a data event",
			Group:       "Utility",
			NoHTTP:      true,
			Examples: []generate.Example{
				{
					Title:       "Subscribe",
					Description: "Subscribe to the DataCreated event",
					ParamsObj:   rpctypes.SubscribeParams{Event: rpctypes.EventKindDataCreated},
					ResultObj:   rpctypes.SubscribeResult{Success: true},
				},
			},
			Errors: []generate.ErrorDoc{
				{
					Title:       "Invalid event",
					Description: "The event topic is invalid",
					Code:        400,
					Message:     "Invalid event topic",
				},
			},
		},
	})

	rpc.RegisterMethod(hub, string(rpctypes.MethodKindUnsubscribe), methods.Unsubscribe, rpc.RegisterMethodOptions{
		Docs: generate.MethodDocs{
			Title:       "Unsubscribe",
			Description: "Unsubscribe from a data event",
			Group:       "Utility",
			NoHTTP:      true,
			Examples: []generate.Example{
				{
					Title:       "Unsubscribe",
					Description: "Unsubscribe from the DataCreated event",
					ParamsObj:   rpctypes.UnsubscribeParams{Event: rpctypes.EventKindDataCreated},
					ResultObj:   rpctypes.UnsubscribeResult{Success: true},
				},
			},
			Errors: []generate.ErrorDoc{
				{
					Title:       "Invalid event",
					Description: "The event topic is invalid",
					Code:        400,
					Message:     "Invalid event topic",
				},
			},
		},
	})

	if err := hub.GenerateDocs(); err != nil {
		fatalIfErr(logger, fmt.Errorf("failed to generate API docs: %w", err))
	}
	if config.Generate {
		logger.Info("Exiting after generating docs")
		return
	}

	migrator, err := database.NewMigrator(logger, sqlite.GetMigrationsFS(), config.Database)
	if err != nil {
		fatalIfErr(logger, fmt.Errorf("failed to create migrator: %w", err))
	}
	if err := migrator.Migrate(); err != nil {
		fatalIfErr(logger, fmt.Errorf("failed to migrate database: %w", err))
	}

	go hub.Run()
	go simulate(hub) // TODO: Remove this

	logger.Info("Registering WS-RPC at /ws")
	mux.HandleFunc("/ws", hub.ServeWS())

	logger.Info("Registering HTTP-RPC at /rpc")
	mux.HandleFunc("/rpc", hub.ServeHTTP())

	web.DocsFS.Register(mux, logger)
	// Redirect root to docs
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
	})

	addr := fmt.Sprintf(":%d", config.Port)
	httpServer := &http.Server{Addr: addr, Handler: mux}

	sigCtx, sigCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer sigCancel()

	// Start HTTP/WS server
	go func() {
		logger.Info("http/ws server listening", slog.String("address", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", utils.ErrAttr(err))
			sigCancel()
		}
	}()

	// Wait for signal (either OS or some failure)
	<-sigCtx.Done()
	logger.Info("received signal, shutting down...")

	// Shutdown / Cleanup
	logger.Info("http/ws server shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http/ws server shutdown failed", utils.ErrAttr(err))
	}

	logger.Info("http/ws server shutdown complete")
}

func generator(config *app.Config, logger *slog.Logger) (generate.Generator, error) {
	if !config.Generate {
		return &generate.MockGenerator{}, nil
	}
	return generate.NewGenerator(logger, generate.GeneratorOptions{
		GoTypesDirPath:               "./internal/rpcapi/types",
		DocsFileOutputPath:           "api_docs.json",
		DatabaseSchemaFileOutputPath: "schema.sql",
		TSTypesOutputPath:            "web/ws-client/generated.ts",
		DocsOptions: generate.DocsOptions{
			Title:       "Local API",
			Description: "A JSON-RPC API over HTTP and Websockets",
		},
	})
}

// TODO: Remove this
func simulate(h *rpc.Hub) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.PublishEvent(rpc.NewEvent(string(rpctypes.EventKindDataCreated), map[string]any{"id": uuid.NewString()}))
	}
}

func getLogger(config *app.Config) *slog.Logger {
	logOptions := slog.HandlerOptions{
		Level:       config.LogLevel,
		ReplaceAttr: utils.SlogReplacer,
	}

	var logHandler slog.Handler = slog.NewJSONHandler(config.LogOutput, &logOptions)
	if config.Generate {
		logHandler = slog.NewTextHandler(config.LogOutput, &logOptions)
	}
	return slog.New(logHandler).With(slog.String("version", utils.GetVersionShort()))
}

func fatalIfErr(l *slog.Logger, err error) {
	if err == nil {
		return
	}

	l.Error("error", utils.ErrAttr(err))
	os.Exit(1)
}
