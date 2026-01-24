package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"ws-json-rpc/backend/internal/app"
	"ws-json-rpc/backend/internal/httpapi"
	"ws-json-rpc/backend/pkg/router"
	"ws-json-rpc/backend/pkg/router/generate"
	"ws-json-rpc/backend/pkg/utils"
)

const (
	shutdownTimeout   = 30 * time.Second
	readHeaderTimeout = 5 * time.Second
)

func main() {
	config, err := app.NewConfig()
	if err != nil {
		fatalIfErr(slog.Default(), fmt.Errorf("failed to create config: %w", err))
	}

	defer func() {
		if err := config.Close(); err != nil {
			fatalIfErr(slog.Default(), fmt.Errorf("failed to close config: %w", err))
		}
	}()

	logger := getLogger(config)

	// Create collector for OpenAPI generation
	var collector generate.RouteMetadataCollector = &generate.NoopCollector{}
	if config.Generate {
		collector, err = generate.NewOpenAPICollector(logger, generate.OpenAPICollectorOptions{
			GoTypesDirPath:               "backend/internal/httpapi",
			DatabaseSchemaFileOutputPath: "schema.sql",
			DocsFileOutputPath:           "test.json",
			OpenAPISpecOutputPath:        "test.yaml",
			APIInfo: generate.APIInfo{
				Title:       "Local API",
				Version:     utils.GetVersionShort(),
				Description: "Local API Documentation",
				Servers: []generate.ServerInfo{
					{
						URL:         "http://localhost:8080",
						Description: "Local server",
					},
				},
			},
		})
		fatalIfErr(logger, err)
	}

	rb, err := router.NewRouteBuilder(logger, collector)
	fatalIfErr(logger, err)

	rb.Must(rb.Get("/ping", router.RouteSpec{
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
			201: {
				Description: "Successful ping response",
				Type:        httpapi.GetTeamResponse{},
				Examples: map[string]any{
					"example-1": httpapi.GetTeamResponse{TeamID: "123", Users: []httpapi.User{{UserID: "123", Name: "John"}}},
				},
			},
			400: {
				Description: "Invalid request",
				Type:        httpapi.CreateUserResponse{},
				Examples: map[string]any{
					"example-1": httpapi.CreateUserResponse{UserID: "123", CreatedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
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
	}))
	rb.Must(rb.Get("/team/{teamID}", router.RouteSpec{
		OperationID: "getTeam",
		Summary:     "Get a team",
		Description: "Get a team by its ID",
		Tags:        []string{"team"},
		RequestType: &router.RequestBodySpec{
			Type: httpapi.GetTeamRequest{},
			Examples: map[string]any{
				"example-1": httpapi.GetTeamResponse{TeamID: "abxc", Users: []httpapi.User{{UserID: "Asdf"}}},
			},
		},
		Parameters: map[string]router.ParameterSpec{
			"teamID": {
				In:          "path",
				Description: "ID of the team to get",
				Required:    true,
				Type:        new(string),
			},
		},
		Responses: map[int]router.ResponseSpec{
			200: {
				Description: "Successful ping response",
				Type:        httpapi.PingResponse{},
				Examples: map[string]any{
					"example-1": httpapi.PingResponse{Message: "Pong", Status: httpapi.PingStatusOK},
				},
			},
			201: {
				Description: "Successful ping response",
				Type:        httpapi.GetTeamResponse{},
				Examples: map[string]any{
					"example-1": httpapi.GetTeamResponse{TeamID: "123", Users: []httpapi.User{{UserID: "123", Name: "John"}}},
				},
			},
			400: {
				Description: "Invalid request",
				Type:        httpapi.CreateUserResponse{},
				Examples: map[string]any{
					"example-1": httpapi.CreateUserResponse{UserID: "123", CreatedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
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
	}))

	if config.Generate {
		if err := collector.Generate(); err != nil {
			fatalIfErr(logger, fmt.Errorf("failed to generate API documentation: %w", err))
		}

		logger.Info("API documentation generated, exiting")

		return
	}

	addr := fmt.Sprintf(":%d", config.Port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           rb.Router(),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	sigCtx, sigCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer sigCancel()

	// Start HTTP/WS server
	go func() {
		logger.Info("http server listening", slog.String("address", addr))

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", utils.ErrAttr(err))
			sigCancel()
		}
	}()

	// Wait for signal (either OS or some failure)
	<-sigCtx.Done()
	logger.Info("received signal, shutting down...")

	// Shutdown / Cleanup
	logger.Info("http server shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown failed", utils.ErrAttr(err))
	}

	logger.Info("http server shutdown complete")
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
