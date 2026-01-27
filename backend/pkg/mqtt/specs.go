package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// QoS represents MQTT quality of service levels.
type QoS byte

const (
	// QoSAtMostOnce means the message is delivered at most once, or it may not be delivered at all.
	QoSAtMostOnce QoS = 0
	// QoSAtLeastOnce means the message is always delivered at least once.
	QoSAtLeastOnce QoS = 1
	// QoSExactlyOnce means the message is always delivered exactly once.
	QoSExactlyOnce QoS = 2
)

// TopicParameter describes a parameter in an MQTT topic pattern.
type TopicParameter struct {
	// Name is the parameter name (e.g., "deviceID")
	Name string

	// Description explains what this parameter represents
	Description string
}

// PublicationSpec describes an MQTT publication operation.
type PublicationSpec struct {
	// OperationID is a unique identifier for this publication operation (e.g., "publishTemperature").
	OperationID string

	// Summary is a short description of the publication.
	Summary string

	// Description provides detailed information about the publication.
	Description string

	// Group is a logical grouping for the publication (e.g., "Telemetry", "Control").
	Group string

	// Deprecated contains an optional deprecation message.
	Deprecated string

	// TopicParameters describes the parameters in the topic pattern (e.g., {deviceID}).
	TopicParameters []TopicParameter

	// MessageType is the Go type of the message being published.
	MessageType any

	// QoS is the quality of service level for this publication.
	QoS QoS

	// Retained indicates whether the message should be retained by the broker.
	Retained bool

	// Examples contains named examples of messages that can be published.
	Examples map[string]any
}

// SubscriptionSpec describes an MQTT subscription operation.
type SubscriptionSpec struct {
	// OperationID is a unique identifier for this subscription operation (e.g., "subscribeTemperature").
	OperationID string

	// Summary is a short description of the subscription.
	Summary string

	// Description provides detailed information about the subscription.
	Description string

	// Group is a logical grouping for the subscription (e.g., "Telemetry", "Control").
	Group string

	// Deprecated contains an optional deprecation message.
	Deprecated string

	// TopicParameters describes the parameters in the topic pattern (e.g., {deviceID}).
	TopicParameters []TopicParameter

	// MessageType is the expected Go type of messages received on this subscription.
	// This should be a zero value of the message type (e.g., apitypes.TemperatureReading{}).
	MessageType any

	// Handler is the function that will be called when a message is received.
	Handler mqtt.MessageHandler

	// QoS is the quality of service level for this subscription.
	QoS QoS

	// Examples contains named examples of messages that may be received.
	Examples map[string]any
}
