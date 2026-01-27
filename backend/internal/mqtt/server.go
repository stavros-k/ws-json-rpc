package mqtt

import (
	"log/slog"
	"ws-json-rpc/backend/internal/services"
)

// Server handles MQTT message processing.
type Server struct {
	l   *slog.Logger
	svc *services.Services
}

// NewMQTTServer creates a new MQTT server.
func NewMQTTServer(l *slog.Logger, svc *services.Services) *Server {
	return &Server{
		l:   l.With(slog.String("component", "mqtt-server")),
		svc: svc,
	}
}
