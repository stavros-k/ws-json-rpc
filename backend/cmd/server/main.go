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
	"ws-json-rpc/backend/internal/api"
	"ws-json-rpc/backend/internal/app"
	"ws-json-rpc/backend/pkg/apitypes"
	"ws-json-rpc/backend/pkg/router"
	"ws-json-rpc/backend/pkg/router/generate"
	"ws-json-rpc/backend/pkg/types"
	"ws-json-rpc/backend/pkg/utils"
	"ws-json-rpc/web"
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
			GoTypesDirPath:               "backend/pkg/apitypes",
			DatabaseSchemaFileOutputPath: "schema.sql",
			DocsFileOutputPath:           "api_docs.json",
			OpenAPISpecOutputPath:        "openapi.yaml",
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

	server := api.NewAPIServer(logger, nil)
	mkHandle := func(h api.HandlerFunc) http.HandlerFunc {
		return api.ErrorHandler(h)
	}

	rb, err := router.NewRouteBuilder(logger, collector)
	fatalIfErr(logger, err)

	rb.Route("/api", func(rb *router.RouteBuilder) {
		// Add request logger
		rb.Use(server.LoggerMiddleware)

		rb.Must(rb.Get("/ping", router.RouteSpec{
			OperationID: "ping",
			Summary:     "Ping the server",
			Description: "Check if the server is alive",
			Group:       "Core",
			RequestType: nil,
			Handler:     mkHandle(server.Ping),
			Responses: api.MakeResponses(map[int]router.ResponseSpec{
				200: {
					Description: "Successful ping response",
					Type:        apitypes.PingResponse{},
					Examples: map[string]any{
						"Success": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
					},
				},
			}),
		}))
		rb.MustGet("/team/{teamID}", router.RouteSpec{
			OperationID: "getTeam",
			Summary:     "Get a team",
			Description: "Get a team by its ID",
			Group:       "Team",
			Deprecated:  "Use GetTeamResponseV2 instead.",
			RequestType: &router.RequestBodySpec{
				Type: apitypes.GetTeamRequest{},
				Examples: map[string]any{
					"example-1": apitypes.GetTeamResponse{TeamID: "abxc", Users: []apitypes.User{{UserID: "Asdf"}}},
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
			Responses: api.MakeResponses(map[int]router.ResponseSpec{
				200: {
					Description: "Successful ping response",
					Type:        apitypes.PingResponse{},
					Examples: map[string]any{
						"example-1": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
					},
				},
				201: {
					Description: "Successful ping response",
					Type:        apitypes.GetTeamResponse{},
					Examples: map[string]any{
						"example-1": apitypes.GetTeamResponse{TeamID: "123", Users: []apitypes.User{{UserID: "123", Name: "John"}}},
					},
				},
				400: {
					Description: "Invalid request",
					Type:        apitypes.CreateUserResponse{},
					Examples: map[string]any{
						"example-1": apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
					},
				},
			}),
			Handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		})

		rb.MustPost("/team", router.RouteSpec{
			OperationID: "createTeam",
			Summary:     "Create a team",
			Description: "Create a team by its name",
			Group:       "Team",
			RequestType: &router.RequestBodySpec{
				Type: apitypes.CreateTeamRequest{},
				Examples: map[string]any{
					"example-1": apitypes.CreateTeamRequest{Name: "My Team"},
				},
			},
			Responses: api.MakeResponses(map[int]router.ResponseSpec{
				200: {
					Description: "Successful ping response",
					Type:        apitypes.PingResponse{},
					Examples: map[string]any{
						"example-1": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
					},
				},
				400: {
					Description: "Invalid request",
					Type:        apitypes.CreateUserResponse{},
					Examples: map[string]any{
						"example-1": apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), URL: utils.Ptr(types.MustNewURL("https://localhost:8080/user"))},
					},
				},
			}),
			Handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		})
		rb.MustPut("/team", router.RouteSpec{
			OperationID: "putTeam",
			Summary:     "Create a team",
			Description: "Create a team by its name",
			Group:       "Team",
			RequestType: &router.RequestBodySpec{
				Type: apitypes.CreateTeamRequest{},
				Examples: map[string]any{
					"example-1": apitypes.CreateTeamRequest{Name: "My Team"},
				},
			},
			Responses: api.MakeResponses(map[int]router.ResponseSpec{
				200: {
					Description: "Successful ping response",
					Type:        apitypes.PingResponse{},
					Examples: map[string]any{
						"example-1": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
					},
				},
				400: {
					Description: "Invalid request",
					Type:        apitypes.CreateUserResponse{},
					Examples: map[string]any{
						"example-1": apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), URL: utils.Ptr(types.MustNewURL("https://localhost:8080/user"))},
					},
				},
			}),
			Handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		})
		rb.MustDelete("/team", router.RouteSpec{
			OperationID: "deleteTeam",
			Summary:     "Create a team",
			Description: "Create a team by its name",
			Group:       "Team",
			RequestType: &router.RequestBodySpec{
				Type: apitypes.CreateTeamRequest{},
				Examples: map[string]any{
					"example-1": apitypes.CreateTeamRequest{Name: "My Team"},
				},
			},
			Responses: api.MakeResponses(map[int]router.ResponseSpec{
				200: {
					Description: "Successful ping response",
					Type:        apitypes.PingResponse{},
					Examples: map[string]any{
						"example-1": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
					},
				},
				400: {
					Description: "Invalid request",
					Type:        apitypes.CreateUserResponse{},
					Examples: map[string]any{
						"example-1": apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), URL: utils.Ptr(types.MustNewURL("https://localhost:8080/user"))},
					},
				},
			}),
			Handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		})
	})

	if config.Generate {
		if err := collector.Generate(); err != nil {
			fatalIfErr(logger, fmt.Errorf("failed to generate API documentation: %w", err))
		}

		logger.Info("API documentation generated, exiting")

		return
	}

	router := rb.Router()

	web.DocsApp().Register(router, logger)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
	})

	addr := fmt.Sprintf(":%d", config.Port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           router,
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
