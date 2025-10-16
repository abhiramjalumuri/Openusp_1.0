package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	confluentkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gorilla/websocket"

	"openusp/pkg/config"
	"openusp/pkg/kafka"
)

const (
	ServiceName    = "mtp-websocket"
	ServiceVersion = "1.0.0"
)

// WebSocketMTPService handles WebSocket transport for USP messages
type WebSocketMTPService struct {
	config        *config.Config
	kafkaClient   *kafka.Client
	kafkaProducer *kafka.Producer
	kafkaConsumer *kafka.Consumer
	upgrader      websocket.Upgrader
	connections   map[string]*websocket.Conn
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

	// Create Kafka consumer for outbound messages
	kafkaConsumer, err := kafka.NewConsumer(&cfg.Kafka, "mtp-websocket")
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
	svc := &WebSocketMTPService{
		config:        cfg,
		kafkaClient:   kafkaClient,
		kafkaProducer: kafkaProducer,
		kafkaConsumer: kafkaConsumer,
		connections:   make(map[string]*websocket.Conn),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			Subprotocols:    []string{cfg.MTP.Websocket.Subprotocol},
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}

	// Validate and get ports from configuration - no hardcoded defaults
	healthPort := svc.config.MTPWebsocketHealthPort
	if healthPort == 0 {
		log.Fatalf("Health port for mtp-websocket not configured in openusp.yml (ports.health.mtp_websocket)")
	}

	metricsPort := svc.config.MTPWebsocketMetricsPort
	if metricsPort == 0 {
		log.Fatalf("Metrics port for mtp-websocket not configured in openusp.yml (ports.metrics.mtp_websocket)")
	}

	wsPort := svc.config.MTP.Websocket.ServerPort
	if wsPort == 0 {
		log.Fatalf("WebSocket server port for mtp-websocket not configured in openusp.yml (mtp.websocket.server_port)")
	}

	log.Printf("📋 Using ports - Health: %d, Metrics: %d, WebSocket: %d", healthPort, metricsPort, wsPort)

	// Start health endpoint
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy","service":"` + ServiceName + `","version":"` + ServiceVersion + `"}`))
		})
		log.Printf("🏥 Health endpoint listening on port %d", healthPort)
		http.ListenAndServe(fmt.Sprintf(":%d", healthPort), mux)
	}()

	// Start metrics endpoint
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# HELP mtp_websocket_messages_total Total messages processed\n# TYPE mtp_websocket_messages_total counter\nmtp_websocket_messages_total 0\n"))
		})
		log.Printf("📊 Metrics endpoint listening on port %d", metricsPort)
		http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), mux)
	}()

	// TODO: Start gRPC server on grpcPort
	// This will be implemented after proto definitions are created

	// Setup WebSocket HTTP server on main port
	mainMux := http.NewServeMux()
	mainMux.HandleFunc("/usp", svc.handleWebSocket)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", wsPort),
		Handler:      mainMux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// Start WebSocket server
	go func() {
		log.Printf("🌐 WebSocket server listening on port %d", wsPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ WebSocket server error: %v", err)
		}
	}()

	// Setup Kafka consumer for outbound messages (responses from USP service)
	if err := svc.setupKafkaConsumer(); err != nil {
		log.Fatalf("❌ Failed to setup Kafka consumer: %v", err)
	}

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── WebSocket Port: %d", wsPort)
	log.Printf("   └── Subprotocol: %s", cfg.MTP.Websocket.Subprotocol)
	log.Printf("   └── Kafka Inbound Topic: %s", cfg.Kafka.Topics.USPMessagesInbound)
	log.Printf("   └── Kafka Outbound Topic: %s", cfg.Kafka.Topics.USPMessagesOutbound)
	log.Printf("   └── Health Port: %d", healthPort)
	log.Printf("   └── Kafka Brokers: %v", cfg.Kafka.Brokers)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("🛑 Shutting down %s...", ServiceName)

	// Shutdown WebSocket server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ WebSocket server shutdown error: %v", err)
	}

	// Close all WebSocket connections
	svc.connMutex.Lock()
	for id, conn := range svc.connections {
		conn.Close()
		log.Printf("🔌 Closed WebSocket connection: %s", id)
	}
	svc.connMutex.Unlock()

	log.Printf("✅ %s stopped gracefully", ServiceName)
}

// handleWebSocket handles WebSocket upgrade and message processing
func (s *WebSocketMTPService) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ Failed to upgrade WebSocket connection: %v", err)
		return
	}

	clientID := fmt.Sprintf("ws-client-%d", time.Now().UnixNano())
	log.Printf("�� New WebSocket connection from %s (client: %s)", r.RemoteAddr, clientID)

	// Register connection
	s.connMutex.Lock()
	s.connections[clientID] = conn
	s.connMutex.Unlock()

	// Cleanup on disconnect
	defer func() {
		s.connMutex.Lock()
		delete(s.connections, clientID)
		s.connMutex.Unlock()
		conn.Close()
		log.Printf("🔌 WebSocket connection closed for client %s", clientID)
	}()

	// Message processing loop
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket read error for client %s: %v", clientID, err)
			}
			break
		}

		if messageType != websocket.BinaryMessage {
			log.Printf("⚠️ WebSocket: Ignoring non-binary message from client %s (type: %d)", clientID, messageType)
			continue
		}

		log.Printf("📥 WebSocket: Received USP message from client %s (%d bytes)", clientID, len(payload))

		// Process USP message via USP service
		response, err := s.processUSPMessage(payload, clientID)
		if err != nil {
			log.Printf("❌ WebSocket: Failed to process message from client %s: %v", clientID, err)
			continue
		}

		// Send response back to client
		if len(response) > 0 {
			if err := conn.WriteMessage(websocket.BinaryMessage, response); err != nil {
				log.Printf("❌ WebSocket: Failed to send response to client %s: %v", clientID, err)
				break
			}
			log.Printf("📤 WebSocket: Sent response to client %s (%d bytes)", clientID, len(response))
		}
	}
}

// processUSPMessage publishes USP message to Kafka for processing
func (s *WebSocketMTPService) processUSPMessage(data []byte, clientID string) ([]byte, error) {
	// Publish USP message to Kafka inbound topic for USP service to process
	err := s.kafkaProducer.PublishUSPMessage(
		s.config.Kafka.Topics.USPMessagesInbound,
		clientID,                                     // endpointID (use clientID for now)
		fmt.Sprintf("msg-%d", time.Now().UnixNano()), // messageID
		"Request",   // messageType
		data,        // payload
		"websocket", // mtpProtocol
	)

	if err != nil {
		return nil, fmt.Errorf("failed to publish to Kafka inbound topic: %w", err)
	}

	log.Printf("✅ WebSocket: USP message published to Kafka inbound topic for client %s", clientID)

	// In event-driven architecture, we don't wait for synchronous response
	// Responses will come through Kafka outbound topic
	return nil, nil
}

// setupKafkaConsumer configures Kafka consumer to receive outbound messages from USP service
func (s *WebSocketMTPService) setupKafkaConsumer() error {
	// Subscribe to outbound topic to receive responses from USP service
	topics := []string{s.config.Kafka.Topics.USPMessagesOutbound}

	handler := func(msg *confluentkafka.Message) error {
		log.Printf("📨 WebSocket: Received outbound message from Kafka (%d bytes)", len(msg.Value))
		
		// TODO: Need to extract endpoint/client ID from Kafka message to route to correct WebSocket connection
		// For now, log that message is ready
		// When implemented:
		// 1. Parse message headers to get target endpoint/client ID
		// 2. Look up WebSocket connection in s.connections map
		// 3. Send message via conn.WriteMessage(websocket.BinaryMessage, msg.Value)
		
		log.Printf("ℹ️  WebSocket: Outbound message ready (client routing implementation pending)")
		
		// Example implementation (when client ID routing is added):
		// s.connMutex.RLock()
		// conn, exists := s.connections[clientID]
		// s.connMutex.RUnlock()
		// if exists {
		//     if err := conn.WriteMessage(websocket.BinaryMessage, msg.Value); err != nil {
		//         log.Printf("❌ WebSocket: Failed to send message to client %s: %v", clientID, err)
		//         return err
		//     }
		//     log.Printf("✅ WebSocket: Sent message to client %s (%d bytes)", clientID, len(msg.Value))
		// } else {
		//     log.Printf("⚠️ WebSocket: Client %s not connected, cannot send message", clientID)
		// }
		
		return nil
	}

	// Subscribe to Kafka outbound topic
	if err := s.kafkaConsumer.Subscribe(topics, handler); err != nil {
		return fmt.Errorf("failed to subscribe to Kafka outbound topic: %w", err)
	}

	log.Printf("✅ WebSocket: Subscribed to Kafka outbound topic: %s", s.config.Kafka.Topics.USPMessagesOutbound)
	return nil
}
