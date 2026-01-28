package mqtt

import (
	"encoding/json"
	"log/slog"
	"time"
	"ws-json-rpc/backend/pkg/apitypes"
	"ws-json-rpc/backend/pkg/mqtt"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

// RegisterDeviceCommandPublish registers the device command publication operation.
func RegisterDeviceCommandPublish(mb *mqtt.MQTTBuilder) {
	mb.MustPublish("devices/{deviceID}/commands", mqtt.PublicationSpec{
		OperationID: "publishDeviceCommand",
		Summary:     "Publish device command",
		Description: "Sends commands to IoT devices.",
		Group:       "Control",
		TopicParameters: []mqtt.TopicParameter{
			{
				Name:        "deviceID",
				Description: "Unique identifier of the target device",
			},
		},
		MessageType: apitypes.DeviceCommand{
			DeviceID: "device-001",
			Command:  "restart",
		},
		QoS:      mqtt.QoSAtLeastOnce,
		Retained: false,
		Examples: map[string]any{
			"restart": apitypes.DeviceCommand{
				DeviceID: "device-001",
				Command:  "restart",
			},
			"updateConfig": apitypes.DeviceCommand{
				DeviceID: "device-001",
				Command:  "update_config",
				Parameters: map[string]string{
					"interval": "60",
					"enabled":  "true",
				},
			},
		},
	})
}

// RegisterDeviceCommandSubscribe registers the device command subscription operation.
func RegisterDeviceCommandSubscribe(mb *mqtt.MQTTBuilder, s *Server) {
	mb.MustSubscribe("devices/{deviceID}/commands", mqtt.SubscriptionSpec{
		OperationID: "subscribeDeviceCommand",
		Summary:     "Subscribe to device commands",
		Description: "Receives commands sent to IoT devices for logging and monitoring.",
		Group:       "Control",
		TopicParameters: []mqtt.TopicParameter{
			{
				Name:        "deviceID",
				Description: "Matches any device ID",
			},
		},
		MessageType: apitypes.DeviceCommand{
			DeviceID: "device-001",
			Command:  "restart",
		},
		Handler: s.handleDeviceCommand,
		QoS:     mqtt.QoSAtLeastOnce,
		Examples: map[string]any{
			"restart": apitypes.DeviceCommand{
				DeviceID: "device-001",
				Command:  "restart",
			},
		},
	})
}

// handleDeviceCommand handles incoming device commands.
func (s *Server) handleDeviceCommand(client pahomqtt.Client, msg pahomqtt.Message) {
	var command apitypes.DeviceCommand
	if err := json.Unmarshal(msg.Payload(), &command); err != nil {
		s.l.Error("Failed to unmarshal device command",
			slog.String("topic", msg.Topic()),
			slog.Any("error", err))

		return
	}

	s.l.Info("Received device command",
		slog.String("deviceID", command.DeviceID),
		slog.String("command", command.Command),
		slog.Any("parameters", command.Parameters))

	// Process the command (e.g., log, validate, forward to device)
	// TODO: Add your business logic here
}

// RegisterDeviceStatusPublish registers the device status publication operation.
func RegisterDeviceStatusPublish(mb *mqtt.MQTTBuilder) {
	mb.MustPublish("devices/{deviceID}/status", mqtt.PublicationSpec{
		OperationID: "publishDeviceStatus",
		Summary:     "Publish device status",
		Description: "Publishes device status updates.",
		Group:       "Control",
		TopicParameters: []mqtt.TopicParameter{
			{
				Name:        "deviceID",
				Description: "Unique identifier of the device",
			},
		},
		MessageType: apitypes.DeviceStatus{
			DeviceID:  "device-001",
			Status:    "online",
			Uptime:    3600,
			Timestamp: time.Time{},
		},
		QoS:      mqtt.QoSAtLeastOnce,
		Retained: true,
		Examples: map[string]any{
			"online": apitypes.DeviceStatus{
				DeviceID:  "device-001",
				Status:    "online",
				Uptime:    3600,
				Timestamp: time.Time{},
			},
			"offline": apitypes.DeviceStatus{
				DeviceID:  "device-001",
				Status:    "offline",
				Uptime:    0,
				Timestamp: time.Time{},
			},
		},
	})
}

// RegisterDeviceStatusSubscribe registers the device status subscription operation.
func RegisterDeviceStatusSubscribe(mb *mqtt.MQTTBuilder, s *Server) {
	mb.MustSubscribe("devices/{deviceID}/status", mqtt.SubscriptionSpec{
		OperationID: "subscribeDeviceStatus",
		Summary:     "Subscribe to device status",
		Description: "Receives device status updates from all IoT devices.",
		Group:       "Control",
		TopicParameters: []mqtt.TopicParameter{
			{
				Name:        "deviceID",
				Description: "Matches any device ID",
			},
		},
		MessageType: apitypes.DeviceStatus{
			DeviceID:  "device-001",
			Status:    "online",
			Uptime:    3600,
			Timestamp: time.Time{},
		},
		Handler: s.handleDeviceStatus,
		QoS:     mqtt.QoSAtLeastOnce,
		Examples: map[string]any{
			"online": apitypes.DeviceStatus{
				DeviceID:  "device-001",
				Status:    "online",
				Uptime:    3600,
				Timestamp: time.Time{},
			},
		},
	})
}

// handleDeviceStatus handles incoming device status updates.
func (s *Server) handleDeviceStatus(client pahomqtt.Client, msg pahomqtt.Message) {
	var status apitypes.DeviceStatus
	if err := json.Unmarshal(msg.Payload(), &status); err != nil {
		s.l.Error("Failed to unmarshal device status",
			slog.String("topic", msg.Topic()),
			slog.Any("error", err))

		return
	}

	s.l.Info("Received device status",
		slog.String("deviceID", status.DeviceID),
		slog.String("status", status.Status),
		slog.Int64("uptime", status.Uptime))

	// Process the status (e.g., update device registry, trigger alerts)
	// TODO: Add your business logic here
}
