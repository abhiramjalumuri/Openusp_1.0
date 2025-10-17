package main

import (
	"fmt"
	"io"
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
	ServiceName    = "mtp-http"
	ServiceVersion = "1.0.0"
)

// HTTPMTPService manages HTTP/HTTPS transport for CWMP messages (TR-069)
// Note: HTTP MTP is used for CWMP/TR-069 protocol, not USP/TR-369
type HTTPMTPService struct {
	config        *config.Config
	kafkaClient   *kafka.Client
	kafkaProducer *kafka.Producer
	kafkaConsumer *kafka.Consumer
	httpServer    *http.Server
	httpClient    *http.Client
}

func main() {
	log.Printf("🚀 Starting %s v%s", ServiceName, ServiceVersion)

	// Load configuration
	cfg := config.Load()

	// Create service instance
	svc, err := NewHTTPMTPService(cfg)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}
	defer svc.Close()

	// Initialize HTTP transport
	if err := svc.initHTTPTransport(); err != nil {
		log.Fatalf("❌ Failed to initialize HTTP transport: %v", err)
	}

	// Setup Kafka consumer for outbound CWMP messages (responses from CWMP service)
	if err := svc.setupKafkaConsumer(); err != nil {
		log.Fatalf("❌ Failed to setup Kafka consumer: %v", err)
	}

	// Start Kafka consumer loop to receive outbound messages
	svc.kafkaConsumer.Start()
	log.Printf("✅ Kafka consumer started for outbound messages")

	// Validate and get ports from configuration - no hardcoded defaults
	healthPort := svc.config.MTPHttpHealthPort
	if healthPort == 0 {
		log.Fatalf("Health port for mtp-http not configured in openusp.yml (ports.health.mtp_http)")
	}

	metricsPort := svc.config.MTPHttpMetricsPort
	if metricsPort == 0 {
		log.Fatalf("Metrics port for mtp-http not configured in openusp.yml (ports.metrics.mtp_http)")
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
			w.Write([]byte("# HELP mtp_http_messages_total Total messages processed\n# TYPE mtp_http_messages_total counter\nmtp_http_messages_total 0\n"))
		})
		log.Printf("📊 Metrics endpoint listening on port %d", metricsPort)
		http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), metricsMux)
	}()

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── HTTP Server URL: %s", cfg.MTP.HTTP.ServerURL)
	log.Printf("   └── Protocol: CWMP/TR-069 (not USP/TR-369)")
	log.Printf("   └── Kafka Inbound Topic: %s", cfg.Kafka.Topics.CWMPMessagesInbound)
	log.Printf("   └── Kafka Outbound Topic: %s", cfg.Kafka.Topics.CWMPMessagesOutbound)
	log.Printf("   └── Timeout: %s", cfg.MTP.HTTP.Timeout)
	log.Printf("   └── Max Idle Connections: %d", cfg.MTP.HTTP.MaxIdleConnections)
	log.Printf("   └── Idle Timeout: %s", cfg.MTP.HTTP.IdleTimeout)
	log.Printf("   └── TLS Enabled: %t", cfg.MTP.HTTP.TLSEnabled)
	log.Printf("   └── Health Port: %d", healthPort)
	log.Printf("   └── Metrics Port: %d", metricsPort)
	log.Printf("   └── Kafka Brokers: %v", cfg.Kafka.Brokers)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Printf("🛑 Shutting down %s...", ServiceName)
	log.Printf("✅ %s stopped gracefully", ServiceName)
}

// NewHTTPMTPService creates a new HTTP MTP service instance
func NewHTTPMTPService(cfg *config.Config) (*HTTPMTPService, error) {
	// Initialize Kafka client
	kafkaClient, err := kafka.NewClient(&cfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka client: %w", err)
	}

	// Create Kafka producer for publishing CWMP messages to inbound topic
	kafkaProducer, err := kafka.NewProducer(kafkaClient)
	if err != nil {
		kafkaClient.Close()
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Create Kafka consumer for receiving CWMP messages from outbound topic
	kafkaConsumer, err := kafka.NewConsumer(&cfg.Kafka, "mtp-http")
	if err != nil {
		kafkaProducer.Close()
		kafkaClient.Close()
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	// Ensure topics exist
	topicsManager := kafka.NewTopicsManager(kafkaClient, &cfg.Kafka.Topics)
	if err := topicsManager.EnsureAllTopicsExist(); err != nil {
		log.Printf("⚠️ Failed to ensure topics exist: %v", err)
	}

	log.Printf("✅ Kafka client initialized")

	// Create HTTP client with configured timeouts
	timeout, err := time.ParseDuration(cfg.MTP.HTTP.Timeout)
	if err != nil {
		timeout = 60 * time.Second
	}

	idleTimeout, err := time.ParseDuration(cfg.MTP.HTTP.IdleTimeout)
	if err != nil {
		idleTimeout = 90 * time.Second
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.MTP.HTTP.MaxIdleConnections,
			IdleConnTimeout:     idleTimeout,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	return &HTTPMTPService{
		config:        cfg,
		kafkaClient:   kafkaClient,
		kafkaProducer: kafkaProducer,
		kafkaConsumer: kafkaConsumer,
		httpClient:    httpClient,
	}, nil
}

// initHTTPTransport initializes the HTTP server for receiving USP messages
func (s *HTTPMTPService) initHTTPTransport() error {
	log.Printf("🔌 Initializing HTTP transport...")
	log.Printf("   └── Server URL: %s", s.config.MTP.HTTP.ServerURL)
	log.Printf("   └── TLS Enabled: %t", s.config.MTP.HTTP.TLSEnabled)
	log.Printf("   └── Kafka Brokers: %v", s.config.Kafka.Brokers)
	if s.config.MTP.HTTP.Username != "" {
		log.Printf("   └── Authentication: Enabled")
	}

	// Create HTTP server for receiving messages
	// The actual server setup will be done when we implement full HTTP transport
	// For now, just log the configuration
	log.Printf("✅ HTTP transport initialized")
	return nil
}

// Close cleanly shuts down the service
func (s *HTTPMTPService) Close() error {
	if s.httpServer != nil {
		s.httpServer.Close()
	}

	if s.kafkaConsumer != nil {
		s.kafkaConsumer.Close()
	}

	if s.kafkaProducer != nil {
		s.kafkaProducer.Close()
	}

	if s.kafkaClient != nil {
		s.kafkaClient.Close()
	}

	return nil
}

// handleHTTPMessage handles incoming HTTP CWMP messages from TR-069 devices
func (s *HTTPMTPService) handleHTTPMessage(w http.ResponseWriter, r *http.Request) {
	// Validate request method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication if configured
	if s.config.MTP.HTTP.Username != "" {
		username, password, ok := r.BasicAuth()
		if !ok || username != s.config.MTP.HTTP.Username || password != s.config.MTP.HTTP.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="CWMP HTTP MTP"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Read CWMP message body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ HTTP: Failed to read request body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("📥 HTTP: Received CWMP message from device (%d bytes)", len(body))

	// Publish CWMP message to Kafka inbound topic for CWMP service to process
	err = s.kafkaProducer.PublishCWMPMessage(
		s.config.Kafka.Topics.CWMPMessagesInbound,
		r.RemoteAddr,                                 // endpointID
		fmt.Sprintf("msg-%d", time.Now().UnixNano()), // messageID
		"Request", // messageType
		body,      // payload
		"http",    // mtpProtocol
	)

	if err != nil {
		log.Printf("❌ HTTP: Failed to publish CWMP message to Kafka: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ HTTP: CWMP message published to Kafka inbound topic")

	// Send acknowledgment to device
	// In CWMP, the response will come asynchronously via Kafka outbound topic
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("CWMP message received"))
}

// setupKafkaConsumer configures Kafka consumer to receive outbound CWMP messages from CWMP service
func (s *HTTPMTPService) setupKafkaConsumer() error {
	// Subscribe to CWMP outbound topic to receive responses from CWMP service
	topics := []string{s.config.Kafka.Topics.CWMPMessagesOutbound}

	handler := func(msg *confluentkafka.Message) error {
		log.Printf("📨 HTTP: Received outbound CWMP message from Kafka (%d bytes)", len(msg.Value))
		
		// TODO: Send CWMP response back to device via HTTP
		// Implementation notes:
		// 1. Extract device endpoint/URL from Kafka message headers
		// 2. Use s.httpClient to POST the response to the device's URL
		// 3. Handle HTTP connection pooling and timeouts
		// 4. Log success/failure
		
		log.Printf("ℹ️  HTTP: Outbound CWMP message ready to send (device routing implementation pending)")
		
		// Example implementation (when device routing is added):
		// deviceURL := string(msg.Headers[0].Value) // Get device URL from message header
		// resp, err := s.httpClient.Post(deviceURL, "application/soap+xml", bytes.NewReader(msg.Value))
		// if err != nil {
		//     log.Printf("❌ HTTP: Failed to send CWMP message to device %s: %v", deviceURL, err)
		//     return err
		// }
		// defer resp.Body.Close()
		// log.Printf("✅ HTTP: Sent CWMP message to device %s (%d bytes, status: %d)", deviceURL, len(msg.Value), resp.StatusCode)
		
		return nil
	}

	// Subscribe to Kafka outbound topic
	if err := s.kafkaConsumer.Subscribe(topics, handler); err != nil {
		return fmt.Errorf("failed to subscribe to Kafka CWMP outbound topic: %w", err)
	}

	log.Printf("✅ HTTP: Subscribed to Kafka CWMP outbound topic: %s", s.config.Kafka.Topics.CWMPMessagesOutbound)
	return nil
}
