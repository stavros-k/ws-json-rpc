package mqtt

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"
	"ws-json-rpc/backend/pkg/generate"
	"ws-json-rpc/backend/pkg/utils"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// TODO: Provide a wrapper around publish, that sets the QoS and retained flags correctly

// MQTTBuilder provides a fluent API for registering MQTT publications and subscriptions.
type MQTTBuilder struct {
	client        mqtt.Client
	collector     generate.MQTTMetadataCollector
	l             *slog.Logger
	operationIDs  map[string]struct{}
	publications  map[string]*PublicationSpec
	subscriptions map[string]*SubscriptionSpec
	connected     bool
}

// MQTTClientOptions contains configuration for creating an MQTT client.
type MQTTClientOptions struct {
	BrokerURL string
	ClientID  string
	Username  string
	Password  string
}

// NewMQTTBuilder creates a new MQTT builder with the given broker configuration.
func NewMQTTBuilder(l *slog.Logger, collector generate.MQTTMetadataCollector, opts MQTTClientOptions) (*MQTTBuilder, error) {
	l = l.With(slog.String("component", "mqtt-builder"))

	if opts.BrokerURL == "" {
		return nil, errors.New("broker URL is required")
	}

	if opts.ClientID == "" {
		return nil, errors.New("client ID is required")
	}

	mb := &MQTTBuilder{
		collector:     collector,
		l:             l,
		operationIDs:  make(map[string]struct{}),
		publications:  make(map[string]*PublicationSpec),
		subscriptions: make(map[string]*SubscriptionSpec),
		connected:     false,
	}

	// Configure MQTT client options
	// TODO: Check this
	clientOpts := mqtt.NewClientOptions()
	clientOpts.AddBroker(opts.BrokerURL)
	clientOpts.SetClientID(opts.ClientID)

	if opts.Username != "" {
		clientOpts.SetUsername(opts.Username)
	}

	if opts.Password != "" {
		clientOpts.SetPassword(opts.Password)
	}

	// Retry every 5 seconds, max interval 15 seconds
	clientOpts.SetAutoReconnect(true)
	clientOpts.SetConnectRetry(true)
	clientOpts.SetConnectTimeout(5 * time.Second)
	clientOpts.SetConnectRetryInterval(5 * time.Second)
	clientOpts.SetMaxReconnectInterval(15 * time.Second)
	clientOpts.SetKeepAlive(30 * time.Second)

	// Set connection callbacks
	clientOpts.SetOnConnectHandler(mb.onConnect)
	clientOpts.SetConnectionLostHandler(mb.onConnectionLost)
	clientOpts.SetReconnectingHandler(mb.onReconnecting)

	// FIXME: Set will message
	// clientOpts.SetWill("", "", 2, true)

	mb.client = mqtt.NewClient(clientOpts)

	l.Info("MQTT builder created", slog.String("broker", opts.BrokerURL), slog.String("clientID", opts.ClientID))

	return mb, nil
}

// Client returns the underlying MQTT client.
func (mb *MQTTBuilder) Client() mqtt.Client {
	return mb.client
}

// Publish registers a publication operation.
func (mb *MQTTBuilder) Publish(topic string, spec PublicationSpec) error {
	// Validate topic
	if err := validateTopicPattern(topic); err != nil {
		return fmt.Errorf("invalid topic pattern: %w", err)
	}

	// Validate spec
	if err := mb.validatePublicationSpec(spec); err != nil {
		return fmt.Errorf("invalid publication spec: %w", err)
	}

	// Check for duplicate operationID
	if _, exists := mb.operationIDs[spec.OperationID]; exists {
		return fmt.Errorf("duplicate operationID: %s", spec.OperationID)
	}

	// Convert topic parameters to documentation format
	topicParams, err := generateParameters(topic, spec.TopicParameters)
	if err != nil {
		return fmt.Errorf("failed to generate topic parameters in operationID %s: %w", spec.OperationID, err)
	}

	// Convert parameterized topic to MQTT wildcard format
	mqttTopic := convertTopicToMQTT(topic)
	spec.TopicMQTT = mqttTopic

	// Register with collector
	if err := mb.collector.RegisterMQTTPublication(&generate.MQTTPublicationInfo{
		OperationID:     spec.OperationID,
		Topic:           topic,
		TopicMQTT:       mqttTopic,
		TopicParameters: topicParams,
		Summary:         spec.Summary,
		Description:     spec.Description,
		Group:           spec.Group,
		Deprecated:      spec.Deprecated,
		QoS:             byte(spec.QoS),
		Retained:        spec.Retained,
		TypeValue:       spec.MessageType,
		Examples:        spec.Examples,
	}); err != nil {
		return fmt.Errorf("failed to register publication with collector: %w", err)
	}

	// Store publication
	mb.operationIDs[spec.OperationID] = struct{}{}
	mb.publications[spec.OperationID] = &spec

	mb.l.Info("Registered MQTT publication", slog.String("operationID", spec.OperationID), slog.String("topic", topic), slog.String("group", spec.Group))

	return nil
}

// MustPublish registers a publication operation and terminates the program if an error occurs.
func (mb *MQTTBuilder) MustPublish(topic string, spec PublicationSpec) {
	if err := mb.Publish(topic, spec); err != nil {
		mb.l.Error("Failed to register publication", slog.String("operationID", spec.OperationID), slog.String("topic", topic), slog.String("group", spec.Group), utils.ErrAttr(err))
		os.Exit(1)
	}
}

// Subscribe registers a subscription operation.
func (mb *MQTTBuilder) Subscribe(topic string, spec SubscriptionSpec) error {
	if topic != generate.SanitizePath(topic) {
		return fmt.Errorf("invalid topic pattern")
	}

	// Validate topic
	if err := validateTopicPattern(topic); err != nil {
		return fmt.Errorf("invalid topic pattern: %w", err)
	}

	// Validate spec
	if err := mb.validateSubscriptionSpec(spec); err != nil {
		return fmt.Errorf("invalid subscription spec: %w", err)
	}

	// Check for duplicate operationID
	if _, exists := mb.operationIDs[spec.OperationID]; exists {
		return fmt.Errorf("duplicate operationID: %s", spec.OperationID)
	}

	// Generate topic parameters
	topicParams, err := generateParameters(topic, spec.TopicParameters)
	if err != nil {
		return fmt.Errorf("failed to generate topic parameters in operationID %s: %w", spec.OperationID, err)
	}

	// Convert parameterized topic to MQTT wildcard format
	mqttTopic := convertTopicToMQTT(topic)
	spec.TopicMQTT = mqttTopic

	// Register with collector
	if err := mb.collector.RegisterMQTTSubscription(&generate.MQTTSubscriptionInfo{
		OperationID:     spec.OperationID,
		Topic:           topic,
		TopicMQTT:       mqttTopic,
		TopicParameters: topicParams,
		Summary:         spec.Summary,
		Description:     spec.Description,
		Group:           spec.Group,
		Deprecated:      spec.Deprecated,
		QoS:             byte(spec.QoS),
		TypeValue:       spec.MessageType,
		Examples:        spec.Examples,
	}); err != nil {
		return fmt.Errorf("failed to register subscription with collector: %w", err)
	}

	// Store subscription with MQTT wildcard topic (for actual subscription)
	mb.operationIDs[spec.OperationID] = struct{}{}
	mb.subscriptions[spec.OperationID] = &spec

	mb.l.Info("Registered MQTT subscription", slog.String("operationID", spec.OperationID), slog.String("topic", topic), slog.String("group", spec.Group))

	return nil
}

// MustSubscribe registers a subscription operation and terminates the program if an error occurs.
func (mb *MQTTBuilder) MustSubscribe(topic string, spec SubscriptionSpec) {
	if err := mb.Subscribe(topic, spec); err != nil {
		mb.l.Error("Failed to register subscription", slog.String("operationID", spec.OperationID), slog.String("topic", topic), slog.String("group", spec.Group), utils.ErrAttr(err))
		os.Exit(1)
	}
}

// Connect connects to the MQTT broker.
func (mb *MQTTBuilder) Connect() error {
	mb.l.Info("Connecting to MQTT broker...")

	token := mb.client.Connect()
	token.Wait()

	if err := token.Error(); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	mb.l.Info("Connected to MQTT broker")

	return nil
}

// Disconnect disconnects from the MQTT broker.
func (mb *MQTTBuilder) Disconnect() {
	if !mb.client.IsConnected() {
		return
	}

	mb.l.Info("Disconnecting from MQTT broker...")
	mb.client.Disconnect(250) // 250ms grace period
	mb.l.Info("Disconnected from MQTT broker")
}

// onConnect is called when the client successfully connects or reconnects to the broker.
func (mb *MQTTBuilder) onConnect(client mqtt.Client) {
	mb.l.Info("Connected to MQTT broker, subscribing to topics", slog.Int("subscriptionCount", len(mb.subscriptions)))
	mb.connected = true

	// Subscribe to all registered subscriptions
	for _, spec := range mb.subscriptions {
		token := client.Subscribe(spec.TopicMQTT, byte(spec.QoS), spec.Handler)
		token.Wait()

		if err := token.Error(); err != nil {
			mb.l.Error("Failed to subscribe", slog.String("topic", spec.TopicMQTT), slog.String("operationID", spec.OperationID), utils.ErrAttr(err))
			return
		}

		mb.l.Info("Subscribed", slog.String("topic", spec.TopicMQTT), slog.String("operationID", spec.OperationID))
	}
}

// onConnectionLost is called when the client loses connection to the broker.
func (mb *MQTTBuilder) onConnectionLost(client mqtt.Client, err error) {
	mb.l.Warn("Connection to MQTT broker lost", utils.ErrAttr(err))
	mb.connected = false
}

// onReconnecting is called when the client is reconnecting to the broker.
func (mb *MQTTBuilder) onReconnecting(client mqtt.Client, opts *mqtt.ClientOptions) {
	mb.l.Info("Reconnecting to MQTT broker", slog.String("broker", opts.Servers[0].String()))
}
