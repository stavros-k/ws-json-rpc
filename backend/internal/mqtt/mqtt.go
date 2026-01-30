package mqtt

import (
	"log/slog"
	"ws-json-rpc/backend/internal/services"
)

// Handler handles MQTT message processing.
type Handler struct {
	l   *slog.Logger
	svc *services.Services
}

// NewMQTTHandler creates a new MQTT handler.
func NewMQTTHandler(l *slog.Logger, svc *services.Services) *Handler {
	return &Handler{
		l:   l.With(slog.String("component", "mqtt-handler")),
		svc: svc,
	}
}
