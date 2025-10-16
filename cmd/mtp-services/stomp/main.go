package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"openusp/internal/mtp"
	"openusp/pkg/config"
	"openusp/pkg/kafka"
)

const (
	ServiceName    = "mtp-stomp"
	ServiceVersion = "1.0.0"
)

// STOMPMTPService handles STOMP transport for USP messages
type STOMPMTPService struct {
	config        *config.Config
	stompBroker   *mtp.STOMPBroker
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
	svc := &STOMPMTPService{
		config:        cfg,
		kafkaClient:   kafkaClient,
		kafkaProducer: kafkaProducer,
	}

	// Initialize STOMP broker
	if err := svc.initSTOMPBroker(); err != nil {
		log.Fatalf("❌ Failed to initialize STOMP broker: %v", err)
	}

	// Validate and get ports from configuration - no hardcoded defaults
	healthPort := svc.config.MTPStompHealthPort
	if healthPort == 0 {
		log.Fatalf("Health port for mtp-stomp not configured in openusp.yml (ports.health.mtp_stomp)")
	}

	metricsPort := svc.config.MTPStompMetricsPort
	if metricsPort == 0 {
		log.Fatalf("Metrics port for mtp-stomp not configured in openusp.yml (ports.metrics.mtp_stomp)")
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
			w.Write([]byte("# HELP mtp_stomp_messages_total Total messages processed\n# TYPE mtp_stomp_messages_total counter\nmtp_stomp_messages_total 0\n"))
		})
		log.Printf("📊 Metrics endpoint listening on port %d", metricsPort)
		http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), metricsMux)
	}()

	// TODO: Start gRPC server on grpcPort
	// This will be implemented after proto definitions are created

	// Start STOMP broker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := svc.stompBroker.Start(ctx); err != nil {
			log.Printf("❌ STOMP broker error: %v", err)
		}
	}()

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── STOMP Broker: %s", cfg.MTP.STOMP.BrokerURL)
	log.Printf("   └── STOMP Inbound Queue: %s", cfg.MTP.STOMP.Destinations.Inbound)
	log.Printf("   └── STOMP Outbound Queue: %s", cfg.MTP.STOMP.Destinations.Outbound)
	log.Printf("   └── Health Port: %d", healthPort)
	log.Printf("   └── Kafka Brokers: %v", cfg.Kafka.Brokers)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("🛑 Shutting down %s...", ServiceName)
	cancel()

	// Cleanup
	if svc.stompBroker != nil {
		svc.stompBroker.Close()
	}

	log.Printf("✅ %s stopped gracefully", ServiceName)
}

// initSTOMPBroker initializes STOMP broker connection
func (s *STOMPMTPService) initSTOMPBroker() error {
	// Get STOMP configuration from YAML
	stompConfig := s.config.MTP.STOMP

	// Controller subscribes to the inbound queue to receive messages from agents
	destinations := []string{stompConfig.Destinations.Inbound}

	log.Printf("🔌 Creating STOMP broker connection...")
	log.Printf("   └── Broker URL: %s", stompConfig.BrokerURL)
	log.Printf("   └── Username: %s", stompConfig.Username)
	log.Printf("   └── Subscriptions: %v", destinations)

	broker, err := mtp.NewSTOMPBrokerWithAuth(
		stompConfig.BrokerURL,
		destinations,
		stompConfig.Username,
		stompConfig.Password,
	)
	if err != nil {
		return fmt.Errorf("failed to create STOMP broker: %w", err)
	}

	// Set message handler to process incoming STOMP messages
	broker.SetMessageHandler(s)
	s.stompBroker = broker

	log.Printf("✅ STOMP broker initialized")
	return nil
}

// ProcessUSPMessage implements the MessageProcessor interface
// This is called by the STOMP broker when a message is received
func (s *STOMPMTPService) ProcessUSPMessage(data []byte) ([]byte, error) {
	log.Printf("📥 STOMP: Received USP message (%d bytes)", len(data))

	// Publish USP message to Kafka
	err := s.kafkaProducer.PublishUSPMessage(
		s.config.Kafka.Topics.USPMessagesInbound,
		"stomp-client",                               // endpointID
		fmt.Sprintf("msg-%d", time.Now().UnixNano()), // messageID
		"Request", // messageType
		data,      // payload
		"stomp",   // mtpProtocol
	)

	if err != nil {
		log.Printf("❌ STOMP: Failed to publish to Kafka: %v", err)
		return nil, fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	log.Printf("✅ STOMP: USP message published to Kafka")

	// In event-driven architecture, we don't wait for synchronous response
	return nil, nil
}
