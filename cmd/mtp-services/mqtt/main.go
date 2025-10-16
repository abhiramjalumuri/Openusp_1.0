package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"openusp/pkg/config"
	"openusp/pkg/kafka"
)

const (
	ServiceName    = "mtp-mqtt"
	ServiceVersion = "1.0.0"
)

// MQTTMTPService handles MQTT transport for USP messages
// TODO: Implement MQTT broker connection similar to STOMP
type MQTTMTPService struct {
	config        *config.Config
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

	// TODO: Initialize MQTT broker (implementation pending)
	// if err := svc.initMQTTBroker(); err != nil {
	// 	log.Fatalf("❌ Failed to initialize MQTT broker: %v", err)
	// }

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

	// TODO: Start MQTT broker (implementation pending)

	// go func() {
	// 	if err := svc.mqttBroker.Start(ctx); err != nil {
	// 		log.Printf("❌ MQTT broker error: %v", err)
	// 	}
	// }()

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── MQTT Broker: %s (TODO: implementation pending)", cfg.MTP.MQTT.BrokerURL)
	log.Printf("   └── Health Port: %d", healthPort)
	log.Printf("   └── Kafka Brokers: %v", cfg.Kafka.Brokers)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("🛑 Shutting down %s...", ServiceName)

	// TODO: Cleanup MQTT broker when implemented
	// if svc.mqttBroker != nil {
	// 	svc.mqttBroker.Close()
	// }

	log.Printf("✅ %s stopped gracefully", ServiceName)
}

// TODO: Implement MQTT broker connection
// initMQTTBroker initializes MQTT broker connection
// func (s *MQTTMTPService) initMQTTBroker() error {
// 	// Get MQTT configuration from YAML
// 	mqttConfig := s.config.MTP.MQTT
//
// 	log.Printf("🔌 Creating MQTT broker connection...")
// 	log.Printf("   └── Broker URL: %s", mqttConfig.BrokerURL)
//
// 	broker, err := mtp.NewMQTTBroker(mqttConfig.BrokerURL)
// 	if err != nil {
// 		return fmt.Errorf("failed to create MQTT broker: %w", err)
// 	}
//
// 	// Set message handler to process incoming MQTT messages
// 	broker.SetMessageHandler(s.onMQTTMessage)
// 	s.mqttBroker = broker
//
// 	log.Printf("✅ MQTT broker initialized")
// 	return nil
// }

// TODO: Implement MQTT message handler
// onMQTTMessage handles incoming MQTT messages
// func (s *MQTTMTPService) onMQTTMessage(topic string, payload []byte) {
// 	log.Printf("📥 MQTT: Received message on topic %s (%d bytes)", topic, len(payload))
//
// 	// Process the USP message
// 	response, err := s.ProcessUSPMessage(payload)
// 	if err != nil {
// 		log.Printf("❌ MQTT: Message processing failed: %v", err)
// 		return
// 	}
//
// 	// Response is already sent in ProcessUSPMessage
// 	_ = response
// }

// ProcessUSPMessage implements the MessageProcessor interface
// This is called by the MQTT broker when a message is received
func (s *MQTTMTPService) ProcessUSPMessage(data []byte) ([]byte, error) {
	log.Printf("📥 MQTT: Received USP message (%d bytes)", len(data))

	// Publish USP message to Kafka
	err := s.kafkaProducer.PublishUSPMessage(
		s.config.Kafka.Topics.USPMessagesInbound,
		"mqtt-client",                                // endpointID
		fmt.Sprintf("msg-%d", time.Now().UnixNano()), // messageID
		"Request", // messageType
		data,      // payload
		"mqtt",    // mtpProtocol
	)

	if err != nil {
		log.Printf("❌ MQTT: Failed to publish to Kafka: %v", err)
		return nil, fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	log.Printf("✅ MQTT: USP message published to Kafka")

	// In event-driven architecture, we don't wait for synchronous response
	// The response will be published to MQTT asynchronously via a consumer
	// For now, return nil to indicate async processing
	return nil, nil
}
