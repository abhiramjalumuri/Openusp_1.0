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

	"openusp/pkg/config"
	"openusp/pkg/kafka"
	pb_v1_3 "openusp/pkg/proto/v1_3"
	pb_v1_4 "openusp/pkg/proto/v1_4"

	kafkago "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
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
	connections   map[string]*websocket.Conn // clientID -> websocket connection
	endpointMap   map[string]string          // endpoint ID -> clientID (for routing responses)
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
	kafkaConsumer, err := kafka.NewConsumer(&cfg.Kafka, "openusp-mtp-websocket")
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
		endpointMap:   make(map[string]string),
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

	// Setup Kafka consumer for outbound messages
	if err := svc.setupKafkaConsumer(); err != nil {
		log.Fatalf("❌ Failed to setup Kafka consumer: %v", err)
	}

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── WebSocket Port: %d", wsPort)
	log.Printf("   └── Subprotocol: %s", cfg.MTP.Websocket.Subprotocol)
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
	log.Printf("📥 WebSocket: Received USP TR-369 protobuf message from client %s (%d bytes)", clientID, len(data))

	// Parse the USP Record to extract endpoint ID for routing responses
	endpointID, err := s.extractEndpointID(data)
	if err != nil {
		log.Printf("⚠️ WebSocket: Failed to extract endpoint ID from message: %v", err)
	} else {
		// Map endpoint ID to client ID for response routing
		s.connMutex.Lock()
		s.endpointMap[endpointID] = clientID
		s.connMutex.Unlock()
		log.Printf("🔗 WebSocket: Mapped endpoint %s to client %s", endpointID, clientID)
	}

	// Publish raw TR-369 protobuf bytes directly to Kafka
	// The USP service will parse the protobuf Record
	err = s.kafkaProducer.PublishRaw(
		s.config.Kafka.Topics.USPMessagesInbound,
		"", // key
		data, // raw TR-369 protobuf bytes
	)

	if err != nil {
		return nil, fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	log.Printf("✅ WebSocket: USP TR-369 message published to Kafka topic: %s for client %s", 
		s.config.Kafka.Topics.USPMessagesInbound, clientID)

	// In event-driven architecture, we don't wait for synchronous response
	// The response will be delivered asynchronously through the WebSocket
	// For now, return nil to indicate async processing
	return nil, nil
}

// extractEndpointID extracts the FromId field from USP Record for response routing
func (s *WebSocketMTPService) extractEndpointID(data []byte) (string, error) {
	// Try parsing as v1.3 first
	var record_v13 pb_v1_3.Record
	if err := proto.Unmarshal(data, &record_v13); err == nil && record_v13.FromId != "" {
		return record_v13.FromId, nil
	}

	// Try parsing as v1.4
	var record_v14 pb_v1_4.Record
	if err := proto.Unmarshal(data, &record_v14); err == nil && record_v14.FromId != "" {
		return record_v14.FromId, nil
	}

	return "", fmt.Errorf("failed to parse USP Record or FromId is empty")
}

// extractTargetEndpointID extracts the ToId (target endpoint) from USP Record
func (s *WebSocketMTPService) extractTargetEndpointID(data []byte) (string, error) {
	// Try parsing as v1.3 first
	var record_v13 pb_v1_3.Record
	if err := proto.Unmarshal(data, &record_v13); err == nil && record_v13.ToId != "" {
		return record_v13.ToId, nil
	}

	// Try parsing as v1.4
	var record_v14 pb_v1_4.Record
	if err := proto.Unmarshal(data, &record_v14); err == nil && record_v14.ToId != "" {
		return record_v14.ToId, nil
	}

	return "", fmt.Errorf("failed to parse USP Record or ToId is empty")
}

// setupKafkaConsumer subscribes to outbound messages from Kafka
func (s *WebSocketMTPService) setupKafkaConsumer() error {
	log.Printf("🔌 Setting up Kafka consumer for outbound messages...")

	// Subscribe to USP outbound topic
	topics := []string{s.config.Kafka.Topics.USPMessagesOutbound}

	handler := func(msg *kafkago.Message) error {
		log.Printf("📥 Kafka: Received outbound USP message (%d bytes) from topic: %s", 
			len(msg.Value), *msg.TopicPartition.Topic)

		// Extract target endpoint ID from the USP Record
		targetEndpoint, err := s.extractTargetEndpointID(msg.Value)
		if err != nil {
			log.Printf("⚠️ Failed to extract target endpoint: %v, broadcasting to all clients", err)
			// Broadcast to all connected clients as fallback
			return s.broadcastMessage(msg.Value)
		}

		// Route to specific client based on endpoint mapping
		s.connMutex.RLock()
		clientID, exists := s.endpointMap[targetEndpoint]
		s.connMutex.RUnlock()

		if !exists {
			log.Printf("⚠️ No WebSocket client found for endpoint %s, broadcasting", targetEndpoint)
			return s.broadcastMessage(msg.Value)
		}

		// Send to specific client
		return s.sendToClient(clientID, msg.Value)
	}

	if err := s.kafkaConsumer.Subscribe(topics, handler); err != nil {
		return fmt.Errorf("failed to subscribe to Kafka topics: %w", err)
	}

	log.Printf("✅ Subscribed to Kafka topic: %s", s.config.Kafka.Topics.USPMessagesOutbound)

	// Start consuming in background
	go func() {
		log.Printf("🔄 Starting Kafka consumer for outbound messages...")
		s.kafkaConsumer.Start()
	}()

	return nil
}

// sendToClient sends a message to a specific WebSocket client
func (s *WebSocketMTPService) sendToClient(clientID string, data []byte) error {
	s.connMutex.RLock()
	conn, exists := s.connections[clientID]
	s.connMutex.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not connected", clientID)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		log.Printf("❌ WebSocket: Failed to send message to client %s: %v", clientID, err)
		return err
	}

	log.Printf("📤 WebSocket: Sent outbound message to client %s (%d bytes)", clientID, len(data))
	return nil
}

// broadcastMessage sends a message to all connected WebSocket clients
func (s *WebSocketMTPService) broadcastMessage(data []byte) error {
	s.connMutex.RLock()
	clients := make(map[string]*websocket.Conn)
	for id, conn := range s.connections {
		clients[id] = conn
	}
	s.connMutex.RUnlock()

	if len(clients) == 0 {
		return fmt.Errorf("no connected clients to broadcast to")
	}

	var lastErr error
	successCount := 0

	for clientID, conn := range clients {
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			log.Printf("❌ WebSocket: Failed to broadcast to client %s: %v", clientID, err)
			lastErr = err
		} else {
			successCount++
		}
	}

	log.Printf("📤 WebSocket: Broadcasted message to %d/%d clients (%d bytes)", 
		successCount, len(clients), len(data))

	return lastErr
}
