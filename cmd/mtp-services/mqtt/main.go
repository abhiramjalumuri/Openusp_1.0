package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	confluentkafka "github.com/confluentinc/confluent-kafka-go/kafka"

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
	kafkaConsumer *kafka.Consumer
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

	// Create Kafka consumer for outbound messages
	kafkaConsumer, err := kafka.NewConsumer(&cfg.Kafka, "mtp-mqtt")
	if err != nil {
		log.Fatalf("❌ Failed to create Kafka consumer: %v", err)
	}
	defer kafkaConsumer.Close()

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
		kafkaConsumer: kafkaConsumer,
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

	// Setup Kafka consumer for outbound messages (responses from USP service)
	if err := svc.setupKafkaConsumer(); err != nil {
		log.Fatalf("❌ Failed to setup Kafka consumer: %v", err)
	}

	// Start Kafka consumer loop to receive outbound messages
	svc.kafkaConsumer.Start()
	log.Printf("✅ Kafka consumer started for outbound messages")

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── MQTT Broker: %s (TODO: implementation pending)", cfg.MTP.MQTT.BrokerURL)
	log.Printf("   └── Kafka Inbound Topic: %s", cfg.Kafka.Topics.USPMessagesInbound)
	log.Printf("   └── Kafka Outbound Topic: %s", cfg.Kafka.Topics.USPMessagesOutbound)
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
// This is called by the MQTT broker when a message is received from agents
func (s *MQTTMTPService) ProcessUSPMessage(data []byte) ([]byte, error) {
	log.Printf("📥 MQTT: Received USP message from agent (%d bytes)", len(data))

	// MQTT topic for agent responses - this is where we should send responses to
	// In a real implementation, this would be extracted from the MQTT message metadata
	// or agent configuration. For now, use default topic.
	mqttTopic := "usp/agent/response" // Default response topic for agents

	// Publish USP message to Kafka inbound topic for USP service to process
	// Include the MQTT topic in MTPDestination so USP service knows where to route responses
	err := s.kafkaProducer.PublishUSPMessageWithDestination(
		s.config.Kafka.Topics.USPMessagesInbound,
		"mqtt-client",                                // endpointID
		fmt.Sprintf("msg-%d", time.Now().UnixNano()), // messageID
		"Request", // messageType
		data,      // payload
		"mqtt",    // mtpProtocol
		kafka.MTPDestination{
			MQTTTopic: mqttTopic, // Where to send responses for this agent
		},
	)

	if err != nil {
		log.Printf("❌ MQTT: Failed to publish to Kafka inbound topic: %v", err)
		return nil, fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	log.Printf("✅ MQTT: USP message published to Kafka inbound topic with destination topic: %s", mqttTopic)

	// In event-driven architecture, we don't wait for synchronous response
	// Responses will come through Kafka outbound topic
	return nil, nil
}

// setupKafkaConsumer configures Kafka consumer to receive outbound messages from USP service
func (s *MQTTMTPService) setupKafkaConsumer() error {
	// Subscribe to outbound topic to receive responses from USP service
	topics := []string{s.config.Kafka.Topics.USPMessagesOutbound}

	handler := func(msg *confluentkafka.Message) error {
		log.Printf("📨 MQTT: Received outbound message from Kafka (%d bytes)", len(msg.Value))
		
		// Parse the Kafka message envelope to extract MTP destination
		var envelope kafka.USPMessageEvent
		if err := json.Unmarshal(msg.Value, &envelope); err != nil {
			log.Printf("❌ MQTT: Failed to parse Kafka message: %v", err)
			return fmt.Errorf("failed to parse Kafka message: %w", err)
		}

		log.Printf("📨 MQTT: USP Response - EndpointID: %s, MessageType: %s, Payload: %d bytes",
			envelope.EndpointID, envelope.MessageType, len(envelope.Payload))

		// Use the MQTT topic from the MTP destination if provided, otherwise fall back to default
		mqttTopic := envelope.MTPDestination.MQTTTopic
		if mqttTopic == "" {
			mqttTopic = "usp/agent/response" // Default fallback topic
			log.Printf("⚠️ MQTT: No topic in MTPDestination, using fallback: %s", mqttTopic)
		}
		
		log.Printf("📍 MQTT: Routing to topic: %s", mqttTopic)
		
		// TODO: Send message to agent via MQTT
		// When MQTT broker is implemented, publish to agent's topic:
		// if s.mqttBroker != nil {
		//     err := s.mqttBroker.Publish(mqttTopic, envelope.Payload)
		//     if err != nil {
		//         log.Printf("❌ MQTT: Failed to send message to agent %s via %s: %v", envelope.EndpointID, mqttTopic, err)
		//         return err
		//     }
		//     log.Printf("✅ MQTT: Sent response to agent %s via %s (%d bytes)", envelope.EndpointID, mqttTopic, len(envelope.Payload))
		// } else {
		//     log.Printf("⚠️ MQTT: Broker not connected, cannot send message to agent")
		// }
		
		log.Printf("ℹ️  MQTT: Outbound message ready to send to topic %s (MQTT broker implementation pending)", mqttTopic)
		return nil
	}

	// Subscribe to Kafka outbound topic
	if err := s.kafkaConsumer.Subscribe(topics, handler); err != nil {
		return fmt.Errorf("failed to subscribe to Kafka outbound topic: %w", err)
	}

	log.Printf("✅ MQTT: Subscribed to Kafka outbound topic: %s", s.config.Kafka.Topics.USPMessagesOutbound)
	return nil
}
