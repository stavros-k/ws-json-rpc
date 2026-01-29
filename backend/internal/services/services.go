package services

import (
	"database/sql"
	"log/slog"
	sqlitegen "ws-json-rpc/backend/internal/database/sqlite/gen"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Services struct {
	l          *slog.Logger
	mqttClient mqtt.Client
	Core       *CoreService
}

func NewServices(l *slog.Logger, db *sql.DB, queries *sqlitegen.Queries) *Services {
	return &Services{
		l:    l.With(slog.String("module", "services")),
		Core: NewCoreService(l, db),
	}
}

// RegisterMQTTClient registers the MQTT client with the services.
func (s *Services) RegisterMQTTClient(client mqtt.Client) {
	s.mqttClient = client
}
