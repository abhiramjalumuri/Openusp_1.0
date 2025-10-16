package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"openusp/pkg/config"
	"openusp/pkg/kafka"
)

const (
	ServiceName    = "mtp-uds"
	ServiceVersion = "1.0.0"
)

// UDSMTPService handles Unix Domain Socket transport for USP messages
type UDSMTPService struct {
	config        *config.Config
	listener      net.Listener
	kafkaClient   *kafka.Client
	kafkaProducer *kafka.Producer
	connections   map[string]net.Conn
	connMutex     sync.RWMutex
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
	svc := &UDSMTPService{
		config:        cfg,
		kafkaClient:   kafkaClient,
		kafkaProducer: kafkaProducer,
		connections:   make(map[string]net.Conn),
	}

	// Initialize UDS listener
	if err := svc.initUDSListener(); err != nil {
		log.Fatalf("❌ Failed to initialize UDS listener: %v", err)
	}
	defer svc.listener.Close()

	// Validate and get ports from configuration - no hardcoded defaults
	healthPort := svc.config.MTPUdsHealthPort
	if healthPort == 0 {
		log.Fatalf("Health port for mtp-uds not configured in openusp.yml (ports.health.mtp_uds)")
	}

	metricsPort := svc.config.MTPUdsMetricsPort
	if metricsPort == 0 {
		log.Fatalf("Metrics port for mtp-uds not configured in openusp.yml (ports.metrics.mtp_uds)")
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
			w.Write([]byte("# HELP mtp_uds_messages_total Total messages processed\n# TYPE mtp_uds_messages_total counter\nmtp_uds_messages_total 0\n"))
		})
		log.Printf("📊 Metrics endpoint listening on port %d", metricsPort)
		http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), metricsMux)
	}()

	// TODO: Start gRPC server on grpcPort
	// This will be implemented after proto definitions are created

	// Start accepting UDS connections
	go svc.acceptConnections()

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── UDS Socket: %s", cfg.MTP.UDS.SocketPath)
	log.Printf("   └── Health Port: %d", healthPort)
	log.Printf("   └── Metrics Port: %d", metricsPort)
	log.Printf("   └── Kafka Brokers: %v", cfg.Kafka.Brokers)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("🛑 Shutting down %s...", ServiceName)

	// Close all connections
	svc.connMutex.Lock()
	for clientID, conn := range svc.connections {
		log.Printf("🔌 Closing UDS connection for client %s", clientID)
		conn.Close()
	}
	svc.connMutex.Unlock()

	// Remove socket file
	os.Remove(cfg.MTP.UDS.SocketPath)
	log.Printf("🗑️  Removed UDS socket: %s", cfg.MTP.UDS.SocketPath)

	log.Printf("✅ %s stopped gracefully", ServiceName)
}

// initUDSListener initializes Unix Domain Socket listener
func (s *UDSMTPService) initUDSListener() error {
	socketPath := s.config.MTP.UDS.SocketPath

	// Remove existing socket file if exists
	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	log.Printf("🔌 Creating UDS listener at %s...", socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on UDS: %w", err)
	}

	// Set socket permissions
	if err := os.Chmod(socketPath, 0660); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.listener = listener
	log.Printf("✅ UDS listener initialized at %s", socketPath)
	return nil
}

// acceptConnections accepts incoming UDS connections
func (s *UDSMTPService) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("❌ UDS accept error: %v", err)
			continue
		}

		clientID := fmt.Sprintf("uds-client-%d", time.Now().UnixNano())
		log.Printf("🔌 New UDS connection (client: %s)", clientID)

		// Register connection
		s.connMutex.Lock()
		s.connections[clientID] = conn
		s.connMutex.Unlock()

		// Handle connection in goroutine
		go s.handleConnection(conn, clientID)
	}
}

// handleConnection handles a single UDS connection
func (s *UDSMTPService) handleConnection(conn net.Conn, clientID string) {
	defer func() {
		s.connMutex.Lock()
		delete(s.connections, clientID)
		s.connMutex.Unlock()
		conn.Close()
		log.Printf("🔌 UDS connection closed for client %s", clientID)
	}()

	buffer := make([]byte, 65536) // 64KB buffer

	for {
		// Read USP message from UDS
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("❌ UDS read error for client %s: %v", clientID, err)
			}
			break
		}

		if n == 0 {
			continue
		}

		payload := buffer[:n]
		log.Printf("📥 UDS: Received USP message from client %s (%d bytes)", clientID, len(payload))

		// Process USP message
		response, err := s.processUSPMessage(payload, clientID)
		if err != nil {
			log.Printf("❌ UDS: Failed to process message from client %s: %v", clientID, err)
			continue
		}

		// Send response back to client
		if len(response) > 0 {
			if _, err := conn.Write(response); err != nil {
				log.Printf("❌ UDS: Failed to send response to client %s: %v", clientID, err)
				break
			}
			log.Printf("📤 UDS: Sent response to client %s (%d bytes)", clientID, len(response))
		}
	}
}

// processUSPMessage publishes USP message to Kafka for processing
func (s *UDSMTPService) processUSPMessage(data []byte, clientID string) ([]byte, error) {
	// Publish USP message to Kafka
	err := s.kafkaProducer.PublishUSPMessage(
		s.config.Kafka.Topics.USPMessagesInbound,
		clientID,                                     // endpointID
		fmt.Sprintf("msg-%d", time.Now().UnixNano()), // messageID
		"Request", // messageType
		data,      // payload
		"uds",     // mtpProtocol
	)

	if err != nil {
		return nil, fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	log.Printf("✅ UDS: USP message published to Kafka for client %s", clientID)

	// In event-driven architecture, we don't wait for synchronous response
	return nil, nil
}
