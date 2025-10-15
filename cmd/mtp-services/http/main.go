package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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

// HTTPMTPService manages HTTP/HTTPS transport for USP messages
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

	// Create Kafka producer
	kafkaProducer, err := kafka.NewProducer(kafkaClient)
	if err != nil {
		kafkaClient.Close()
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Create Kafka consumer for outbound messages
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

// initHTTPTransport initializes the HTTP server for receiving TR-069/CWMP messages
func (s *HTTPMTPService) initHTTPTransport() error {
	log.Printf("🔌 Initializing HTTP transport for TR-069/CWMP...")
	log.Printf("   └── Server URL: %s", s.config.MTP.HTTP.ServerURL)
	log.Printf("   └── TLS Enabled: %t", s.config.MTP.HTTP.TLSEnabled)
	log.Printf("   └── Kafka Brokers: %v", s.config.Kafka.Brokers)
	if s.config.MTP.HTTP.Username != "" {
		log.Printf("   └── Authentication: Enabled")
	}

	// Parse the TR-069 port from the ServerURL configuration
	parsedURL, err := url.Parse(s.config.MTP.HTTP.ServerURL)
	if err != nil {
		return fmt.Errorf("failed to parse server_url from config: %w", err)
	}
	
	// Extract the port from the URL, or use 7547 as standard TR-069 port if not specified
	tr069Port := parsedURL.Port()
	if tr069Port == "" {
		tr069Port = "7547" // Standard TR-069 port
	}
	
	// Create HTTP server mux for TR-069/CWMP messages
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleCWMPMessage)
	
	// Parse timeout configurations
	serverTimeout, err := time.ParseDuration(s.config.MTP.HTTP.Timeout)
	if err != nil {
		log.Printf("⚠️ Invalid timeout in config, using default 60s: %v", err)
		serverTimeout = 60 * time.Second
	}
	
	serverIdleTimeout, err := time.ParseDuration(s.config.MTP.HTTP.IdleTimeout)
	if err != nil {
		log.Printf("⚠️ Invalid idle_timeout in config, using default 120s: %v", err)
		serverIdleTimeout = 120 * time.Second
	}
	
	// Create and start HTTP server for TR-069 on configured port
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", tr069Port),
		Handler:      mux,
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
		IdleTimeout:  serverIdleTimeout,
	}
	
	// Start TR-069 server in background
	go func() {
		log.Printf("🌐 TR-069/CWMP HTTP server listening on port %s", tr069Port)
		log.Printf("   └── Forwarding TR-069 messages to Kafka topic: %s", s.config.Kafka.Topics.CWMPMessagesInbound)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ TR-069 HTTP server failed: %v", err)
		}
	}()
	
	log.Printf("✅ HTTP transport initialized - TR-069 endpoint ready")
	
	// Setup Kafka consumer for outbound messages
	if err := s.setupKafkaConsumer(); err != nil {
		log.Printf("⚠️ Failed to setup Kafka consumer: %v", err)
	}
	
	return nil
}

// setupKafkaConsumer subscribes to outbound messages from CWMP service
func (s *HTTPMTPService) setupKafkaConsumer() error {
	log.Printf("🔌 Setting up Kafka consumer for outbound CWMP messages...")

	// Subscribe to CWMP outbound topic
	topics := []string{s.config.Kafka.Topics.CWMPMessagesOutbound}

	handler := func(msg *confluentkafka.Message) error {
		log.Printf("📥 Kafka: Received outbound CWMP message (%d bytes) from topic: %s", len(msg.Value), *msg.TopicPartition.Topic)

		// TODO: Extract device endpoint/session from message metadata
		// TODO: Send message via HTTP to the appropriate CPE device
		// For now, just log that we received it
		log.Printf("✅ Outbound CWMP message ready to send to device")
		
		return nil
	}

	if err := s.kafkaConsumer.Subscribe(topics, handler); err != nil {
		return fmt.Errorf("failed to subscribe to Kafka topics: %w", err)
	}

	log.Printf("✅ Subscribed to Kafka topic: %s", s.config.Kafka.Topics.CWMPMessagesOutbound)

	// Start consuming in background
	go func() {
		log.Printf("🔄 Starting Kafka consumer for outbound messages...")
		s.kafkaConsumer.Start()
	}()

	return nil
}

// Close cleanly shuts down the service
func (s *HTTPMTPService) Close() error {
	if s.httpServer != nil {
		s.httpServer.Close()
	}

	if s.kafkaConsumer != nil {
		s.kafkaConsumer.Stop()
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

// handleCWMPMessage handles incoming TR-069/CWMP HTTP messages and forwards to Kafka
func (s *HTTPMTPService) handleCWMPMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 Received TR-069/CWMP request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	
	// Validate request method - TR-069 uses POST
	if r.Method != http.MethodPost {
		log.Printf("❌ Invalid method for TR-069: %s", r.Method)
		http.Error(w, "TR-069 requires POST method", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication if configured
	if s.config.MTP.HTTP.Username != "" {
		username, password, ok := r.BasicAuth()
		if !ok || username != s.config.MTP.HTTP.Username || password != s.config.MTP.HTTP.Password {
			log.Printf("❌ TR-069 authentication failed from %s", r.RemoteAddr)
			w.Header().Set("WWW-Authenticate", `Basic realm="TR-069 ACS"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("✅ TR-069 authentication successful for user: %s", username)
	}

	// Read TR-069 SOAP message body
	body := make([]byte, 0, r.ContentLength)
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	defer r.Body.Close()

	log.Printf("📦 Received TR-069 message (%d bytes)", len(body))

	// Create Kafka message with TR-069 payload and metadata
	messageKey := fmt.Sprintf("cwmp-%s-%d", r.RemoteAddr, time.Now().UnixNano())
	
	// For now, store headers as JSON in the message (can be enhanced later)
	metadata := map[string]string{
		"content-type":   r.Header.Get("Content-Type"),
		"client-address": r.RemoteAddr,
		"user-agent":     r.Header.Get("User-Agent"),
		"soapaction":     r.Header.Get("SOAPAction"),
		"http-method":    r.Method,
		"http-path":      r.URL.Path,
	}
	
	// Wrap payload with metadata
	type CWMPMessage struct {
		Payload  []byte            `json:"payload"`
		Metadata map[string]string `json:"metadata"`
	}
	
	cwmpMsg := CWMPMessage{
		Payload:  body,
		Metadata: metadata,
	}
	
	msgBytes, err := json.Marshal(cwmpMsg)
	if err != nil {
		log.Printf("❌ Failed to marshal CWMP message: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Forward to CWMP service via Kafka
	topic := s.config.Kafka.Topics.CWMPMessagesInbound
	if topic == "" {
		topic = "cwmp.messages.inbound"
	}

	log.Printf("📤 Forwarding TR-069 message to Kafka topic: %s", topic)
	err = s.kafkaProducer.PublishRaw(topic, messageKey, msgBytes)
	if err != nil {
		log.Printf("❌ Failed to forward TR-069 message to Kafka: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ TR-069 message forwarded to CWMP service via Kafka")

	// Send TR-069 response (empty SOAP response for now - CWMP service will handle async)
	// TR-069 spec requires HTTP 204 No Content or 200 OK with empty body for async processing
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusNoContent)
	
	log.Printf("✅ TR-069 HTTP response sent (204 No Content)")
}

// handleHTTPMessage handles incoming HTTP USP messages
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
			w.Header().Set("WWW-Authenticate", `Basic realm="USP HTTP MTP"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Read message body
	// TODO: Parse USP message and forward to USP service via gRPC
	// For now, just acknowledge receipt
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message received"))
}
