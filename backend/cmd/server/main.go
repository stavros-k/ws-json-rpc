package main

import (
	"context"
	"database/sql"
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
	sqlitegen "ws-json-rpc/backend/internal/database/sqlite/gen"
	mqttapi "ws-json-rpc/backend/internal/mqtt"
	"ws-json-rpc/backend/internal/services"
	"ws-json-rpc/backend/pkg/mqtt"
	"ws-json-rpc/backend/pkg/router"
	"ws-json-rpc/backend/pkg/router/generate"
	"ws-json-rpc/backend/pkg/utils"
	"ws-json-rpc/web"

	_ "github.com/mattn/go-sqlite3"
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
	collector, err := getCollector(config, logger)
	fatalIfErr(logger, err)

	// TODO: Pass DB
	// open sqlite database
	db, err := sql.Open("sqlite3", config.Database)
	fatalIfErr(logger, err)

	defer db.Close()

	queries := sqlitegen.New(db)

	services := services.NewServices(logger, db, queries)

	server := api.NewAPIServer(logger, services)

	rb, err := router.NewRouteBuilder(logger, collector)
	fatalIfErr(logger, err)

	rb.Route("/api", func(rb *router.RouteBuilder) {
		// Add request ID
		rb.Use(server.RequestIDMiddleware)
		// Add request logger
		rb.Use(server.LoggerMiddleware)

		api.RegisterPing("/ping", rb, server)
		rb.Route("/team", func(rb *router.RouteBuilder) {
			api.RegisterGetTeam("/{teamID}", rb, server)
			api.RegisterPutTeam("/", rb, server)
			api.RegisterCreateTeam("/", rb, server)
			api.RegisterDeleteTeam("/", rb, server)
		})
	})

	// Create MQTT builder and register MQTT operations (if enabled or in generate mode)
	mqttServer := mqttapi.NewMQTTServer(logger, services)

	mqttBuilder, err := mqtt.NewMQTTBuilder(logger, collector, mqtt.MQTTBuilderOptions{
		BrokerURL: config.MQTTBroker,
		ClientID:  config.MQTTClientID,
		Username:  config.MQTTUsername,
		Password:  config.MQTTPassword,
	})
	fatalIfErr(logger, err)

	// Register MQTT operations
	logger.Info("Registering MQTT operations...")

	// Telemetry operations
	mqttapi.RegisterTemperaturePublish(mqttBuilder)
	mqttapi.RegisterTemperatureSubscribe(mqttBuilder, mqttServer)
	mqttapi.RegisterSensorTelemetryPublish(mqttBuilder)
	mqttapi.RegisterSensorTelemetrySubscribe(mqttBuilder, mqttServer)

	// Control operations
	mqttapi.RegisterDeviceCommandPublish(mqttBuilder)
	mqttapi.RegisterDeviceCommandSubscribe(mqttBuilder, mqttServer)
	mqttapi.RegisterDeviceStatusPublish(mqttBuilder)
	mqttapi.RegisterDeviceStatusSubscribe(mqttBuilder, mqttServer)

	logger.Info("MQTT operations registered successfully")

	if config.Generate {
		if err := collector.Generate(); err != nil {
			fatalIfErr(logger, fmt.Errorf("failed to generate API documentation: %w", err))
		}

		logger.Info("API documentation generated, exiting")

		return
	}

	if err := mqttBuilder.Connect(); err != nil {
		fatalIfErr(logger, fmt.Errorf("failed to connect to MQTT broker: %w", err))
	}
	defer mqttBuilder.Disconnect()

	web.DocsApp().Register(rb.Router(), logger)
	rb.Router().HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
	})

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

func getCollector(c *app.Config, l *slog.Logger) (generate.MetadataCollector, error) {
	if !c.Generate {
		return &generate.NoopCollector{}, nil
	}

	return generate.NewOpenAPICollector(l, generate.OpenAPICollectorOptions{
		GoTypesDirPath:               "backend/pkg/apitypes",
		DatabaseSchemaFileOutputPath: "schema.sql",
		DocsFileOutputPath:           "api_docs.json",
		OpenAPISpecOutputPath:        "openapi.yaml",
		APIInfo: generate.APIInfo{
			Title:       "Local API",
			Version:     utils.GetVersionShort(),
			Description: "Local API Documentation",
			Servers: []generate.ServerInfo{
				{URL: "http://localhost:8080", Description: "Local server"},
			},
		},
	})
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
