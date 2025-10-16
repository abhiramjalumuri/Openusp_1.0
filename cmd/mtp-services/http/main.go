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
	ServiceName    = "mtp-http"
	ServiceVersion = "1.0.0"
)

// HTTPMTPService manages HTTP/HTTPS transport for USP messages
type HTTPMTPService struct {
	config        *config.Config
	kafkaClient   *kafka.Client
	kafkaProducer *kafka.Producer
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

	if s.kafkaProducer != nil {
		s.kafkaProducer.Close()
	}

	if s.kafkaClient != nil {
		s.kafkaClient.Close()
	}

	return nil
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
