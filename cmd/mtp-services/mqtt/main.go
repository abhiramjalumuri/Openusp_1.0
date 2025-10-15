package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"openusp/pkg/config"
	"openusp/pkg/kafka"
)

const (
	ServiceName    = "mtp-mqtt"
	ServiceVersion = "1.0.0"
)

// MessageHandler defines the function signature for handling messages
type MessageHandler func(topic string, payload []byte)

// MQTTBroker implements MQTT Message Transfer Protocol for USP
type MQTTBroker struct {
	brokerURL      string
	client         mqtt.Client
	messageHandler MessageHandler
	topics         []string
}

// NewMQTTBroker creates a new MQTT broker connection
func NewMQTTBroker(brokerURL string) (*MQTTBroker, error) {
	// Validate broker URL
	_, err := url.Parse(brokerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid MQTT broker URL: %w", err)
	}

	broker := &MQTTBroker{
		brokerURL: brokerURL,
		topics: []string{
			"/usp/controller/+", // Messages to controllers
			"/usp/agent/+",      // Messages to agents
			"/usp/broadcast",    // Broadcast messages
		},
	}

	return broker, nil
}

// SetMessageHandler sets the message handler function
func (b *MQTTBroker) SetMessageHandler(handler MessageHandler) {
	b.messageHandler = handler
}

// Start connects to MQTT broker and starts listening
func (b *MQTTBroker) Start(ctx context.Context) error {
	log.Printf("🔌 Connecting to MQTT broker: %s", b.brokerURL)

	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(b.brokerURL)
	opts.SetClientID("usp-mtp-service")
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(5 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)

	// Set connection handlers
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("✅ MQTT broker connected: %s", b.brokerURL)

		// Subscribe to USP topics
		for _, topic := range b.topics {
			if token := client.Subscribe(topic, 1, b.onMessage); token.Wait() && token.Error() != nil {
				log.Printf("❌ Failed to subscribe to MQTT topic %s: %v", topic, token.Error())
			} else {
				log.Printf("📡 Subscribed to MQTT topic: %s", topic)
			}
		}
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("⚠️ MQTT connection lost: %v", err)
	})

	// Create and connect client
	b.client = mqtt.NewClient(opts)
	if token := b.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Printf("🛑 MQTT broker shutting down...")

	return nil
}

// onMessage handles incoming MQTT messages
func (b *MQTTBroker) onMessage(client mqtt.Client, msg mqtt.Message) {
	if b.messageHandler != nil {
		go b.messageHandler(msg.Topic(), msg.Payload())
	}
}

// Publish sends a message to an MQTT topic
func (b *MQTTBroker) Publish(topic string, payload []byte) error {
	if b.client == nil || !b.client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	token := b.client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish MQTT message: %w", token.Error())
	}

	return nil
}

// Close disconnects from the MQTT broker
func (b *MQTTBroker) Close() {
	if b.client != nil && b.client.IsConnected() {
		b.client.Disconnect(250)
		log.Printf("✅ MQTT broker disconnected")
	}
}

// MQTTMTPService handles MQTT transport for USP messages
type MQTTMTPService struct {
	config        *config.Config
	mqttBroker    *MQTTBroker
	kafkaClient   *kafka.Client
	kafkaProducer *kafka.Producer
}

func main() {
	log.Printf("🚀 Starting %s v%s", ServiceName, ServiceVersion)

	// Load configuration from YAML
	cfg := config.Load()

	// Initialize Kafka client
	kafkaClient, err := kafka.NewClient(&cfg.Kafka)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Kafka client: %v", err)
	}
	defer kafkaClient.Close()

	// Create Kafka producer
	kafkaProducer, err := kafka.NewProducer(kafkaClient)
	if err != nil {
		log.Fatalf("❌ Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Ensure topics exist
	topicsManager := kafka.NewTopicsManager(kafkaClient, &cfg.Kafka.Topics)
	if err := topicsManager.EnsureAllTopicsExist(); err != nil {
		log.Printf("⚠️ Failed to ensure topics exist: %v", err)
	}

	// Create service instance
	svc := &MQTTMTPService{
		config:        cfg,
		kafkaClient:   kafkaClient,
		kafkaProducer: kafkaProducer,
	}

	// Initialize MQTT broker
	if err := svc.initMQTTBroker(); err != nil {
		log.Fatalf("❌ Failed to initialize MQTT broker: %v", err)
	}

	// Validate and get ports from configuration - no hardcoded defaults
	healthPort := svc.config.MTPMqttHealthPort
	if healthPort == 0 {
		log.Fatalf("Health port for mtp-mqtt not configured in openusp.yml (ports.health.mtp_mqtt)")
	}

	metricsPort := svc.config.MTPMqttMetricsPort
	if metricsPort == 0 {
		log.Fatalf("Metrics port for mtp-mqtt not configured in openusp.yml (ports.metrics.mtp_mqtt)")
	}

	log.Printf("📋 Using ports - Health: %d, Metrics: %d", healthPort, metricsPort)

	// Start health endpoint
	go func() {
		healthMux := http.NewServeMux()
		healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy","service":"` + ServiceName + `","version":"` + ServiceVersion + `"}`))
		})
		log.Printf("🏥 Health endpoint listening on port %d", healthPort)
		http.ListenAndServe(fmt.Sprintf(":%d", healthPort), healthMux)
	}()

	// Start metrics endpoint
	go func() {
		metricsMux := http.NewServeMux()
		metricsMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# HELP mtp_mqtt_messages_total Total messages processed\n# TYPE mtp_mqtt_messages_total counter\nmtp_mqtt_messages_total 0\n"))
		})
		log.Printf("📊 Metrics endpoint listening on port %d", metricsPort)
		http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), metricsMux)
	}()

	// TODO: Start gRPC server on grpcPort
	// This will be implemented after proto definitions are created
	// Start MQTT broker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := svc.mqttBroker.Start(ctx); err != nil {
			log.Printf("❌ MQTT broker error: %v", err)
		}
	}()

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── MQTT Broker: %s", cfg.MTP.MQTT.BrokerURL)
	log.Printf("   └── Health Port: %d", healthPort)
	log.Printf("   └── Kafka Brokers: %v", cfg.Kafka.Brokers)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("🛑 Shutting down %s...", ServiceName)
	cancel()

	// Cleanup
	if svc.mqttBroker != nil {
		svc.mqttBroker.Close()
	}

	log.Printf("✅ %s stopped gracefully", ServiceName)
}

// initMQTTBroker initializes MQTT broker connection
func (s *MQTTMTPService) initMQTTBroker() error {
	// Get MQTT configuration from YAML
	mqttConfig := s.config.MTP.MQTT

	log.Printf("🔌 Creating MQTT broker connection...")
	log.Printf("   └── Broker URL: %s", mqttConfig.BrokerURL)

	broker, err := NewMQTTBroker(mqttConfig.BrokerURL)
	if err != nil {
		return fmt.Errorf("failed to create MQTT broker: %w", err)
	}

	// Set message handler to process incoming MQTT messages
	broker.SetMessageHandler(s.onMQTTMessage)
	s.mqttBroker = broker

	log.Printf("✅ MQTT broker initialized")
	return nil
}

// onMQTTMessage handles incoming MQTT messages
func (s *MQTTMTPService) onMQTTMessage(topic string, payload []byte) {
	log.Printf("📥 MQTT: Received message on topic %s (%d bytes)", topic, len(payload))

	// Process the USP message
	response, err := s.ProcessUSPMessage(payload)
	if err != nil {
		log.Printf("❌ MQTT: Message processing failed: %v", err)
		return
	}

	// Response is already sent in ProcessUSPMessage
	_ = response
}

// ProcessUSPMessage implements the MessageProcessor interface
// This is called by the MQTT broker when a message is received
func (s *MQTTMTPService) ProcessUSPMessage(data []byte) ([]byte, error) {
	log.Printf("📥 MQTT: Received USP TR-369 protobuf message (%d bytes)", len(data))

	// Publish raw TR-369 protobuf bytes directly to Kafka
	// The USP service will parse the protobuf Record
	err := s.kafkaProducer.PublishRaw(
		s.config.Kafka.Topics.USPMessagesInbound,
		"", // key
		data, // raw TR-369 protobuf bytes
	)

	if err != nil {
		log.Printf("❌ MQTT: Failed to publish to Kafka: %v", err)
		return nil, fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	log.Printf("✅ MQTT: USP TR-369 message published to Kafka topic: %s", s.config.Kafka.Topics.USPMessagesInbound)

	// In event-driven architecture, we don't wait for synchronous response
	// The response will be published to MQTT asynchronously via a consumer
	// For now, return nil to indicate async processing
	return nil, nil
}
