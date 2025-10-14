// ...existing code...
// ...existing code...
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"openusp/internal/mtp"
	"openusp/internal/usp"

	"openusp/pkg/config"
	"openusp/pkg/metrics"
	"openusp/pkg/proto/mtpservice"
	"openusp/pkg/proto/uspservice"
	"openusp/pkg/proto/v1_3"
	"openusp/pkg/proto/v1_4"
	"openusp/pkg/service/client"
	"openusp/pkg/version"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// SimpleMTPService provides a basic MTP service implementation
type SimpleMTPService struct {
	webSocketPort    int
	healthPort       int
	grpcPort         int
	unixSocketPath   string
	webSocketServer  *http.Server // WebSocket protocol server (standard port 8081)
	healthServer     *http.Server // Health/admin server (dynamic port)
	grpcServer       *grpc.Server // gRPC server for USP service communication
	messageHandler   MessageHandler
	connectedClients map[string]time.Time
	upgrader         websocket.Upgrader
	metrics          *metrics.OpenUSPMetrics
	mu               sync.RWMutex
}

// ConnectionContext provides context about the connection for message processing
type ConnectionContext struct {
	ClientID      string
	TransportType string
	WebSocketConn *websocket.Conn // For WebSocket connections
	// Additional transport-specific fields can be added here
}

// MessageHandler defines the interface for handling USP messages
type MessageHandler interface {
	ProcessUSPMessage(data []byte) ([]byte, error)
	ProcessUSPMessageWithContext(data []byte, connCtx *ConnectionContext) ([]byte, error)
	DetectUSPVersion(data []byte) (string, error)
}

// Config holds configuration for MTP service
type Config struct {
	WebSocketPort    int // Standard WebSocket port (8081)
	HealthPort       int // Dynamic port for health/status/metrics
	GRPCPort         int // gRPC port for USP service communication
	UnixSocketPath   string
	EnableWebSocket  bool
	EnableUnixSocket bool
	EnableMQTT       bool
	EnableSTOMP      bool
}

// DefaultConfig returns default MTP service configuration with calculated gRPC port
func DefaultConfig(healthPort int) *Config {
	return DefaultConfigWithPorts(healthPort, healthPort+1000)
}

// DefaultConfigWithPorts returns MTP service configuration with specified ports
func DefaultConfigWithPorts(healthPort, grpcPort int) *Config {
	// Load MTP configuration from YAML
	mtpConfig := config.LoadMTPConfig()
	webSocketPort := 8081
	if mtpConfig.WebSocketEnabled {
		// If YAML provides a port, use it (extend MTPConfig if needed)
		// For now, keep default 8081
	}

	return &Config{
		WebSocketPort:    webSocketPort,
		HealthPort:       healthPort,
		GRPCPort:         grpcPort,
		UnixSocketPath:   mtpConfig.UnixSocketPath,
		EnableWebSocket:  mtpConfig.WebSocketEnabled,
		EnableUnixSocket: mtpConfig.UnixSocketEnabled,
		EnableMQTT:       mtpConfig.MQTTEnabled,
		EnableSTOMP:      mtpConfig.STOMPEnabled,
	}
}

// NewSimpleMTPService creates a new simplified MTP service
func NewSimpleMTPService(config *Config, handler MessageHandler) (*SimpleMTPService, error) {
	// Initialize metrics
	metricsInstance := metrics.NewOpenUSPMetrics("mtp-service")

	return &SimpleMTPService{
		webSocketPort:    config.WebSocketPort,
		healthPort:       config.HealthPort,
		grpcPort:         config.GRPCPort,
		unixSocketPath:   config.UnixSocketPath,
		messageHandler:   handler,
		connectedClients: make(map[string]time.Time),
		metrics:          metricsInstance,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins
			},
			Subprotocols: []string{"v1.usp"}, // Support USP WebSocket subprotocol
		},
	}, nil
}

// Start starts the MTP service
func (s *SimpleMTPService) Start(ctx context.Context) error {
	log.Println("🚀 Starting Simple MTP Service...")

	// Create WebSocket protocol server (standard port 8081)
	webSocketMux := http.NewServeMux()
	// Removed demo HTML page endpoint
	webSocketMux.HandleFunc("/ws", s.handleUSPWebSocket) // Actual WebSocket endpoint

	s.webSocketServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.webSocketPort),
		Handler: webSocketMux,
	}

	// Create health/admin server
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", s.handleHealth)
	healthMux.HandleFunc("/status", s.handleStatus)
	healthMux.Handle("/metrics", metrics.HTTPHandler())
	healthMux.HandleFunc("/simulate", s.handleSimulate)

	s.healthServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.healthPort),
		Handler: healthMux,
	}

	// Start WebSocket protocol server
	go func() {
		log.Printf("🔌 Starting WebSocket server on port %d", s.webSocketPort)
		if err := s.webSocketServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ WebSocket server error: %v", err)
		}
	}()

	// Start health/admin server
	go func() {
		log.Printf("🔌 Starting health server on port %d", s.healthPort)
		if err := s.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Health server error: %v", err)
		}
	}()

	// Create and start gRPC server for MTP service
	s.grpcServer = grpc.NewServer()

	// Register the MTP service - need to cast messageHandler to USPMessageHandler
	if uspHandler, ok := s.messageHandler.(*USPMessageHandler); ok {
		mtpservice.RegisterMTPServiceServer(s.grpcServer, uspHandler)
		log.Printf("✅ Registered MTP gRPC service with agent connection support")
	} else {
		log.Printf("⚠️ Message handler is not USPMessageHandler - gRPC service registration skipped")
	}

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.grpcPort))
		if err != nil {
			log.Printf("❌ Failed to listen on gRPC port %d: %v", s.grpcPort, err)
			return
		}

		log.Printf("🔌 Starting gRPC server on port %d", s.grpcPort)
		if err := s.grpcServer.Serve(lis); err != nil {
			log.Printf("❌ gRPC server error: %v", err)
		}
	}()

	log.Printf("✅ Simple MTP Service started")
	log.Printf("   📡 WebSocket Port: %d (USP Protocol)", s.webSocketPort)
	log.Printf("   🏥 Health API Port: %d (Dynamic)", s.healthPort)
	log.Printf("   🔧 gRPC Port: %d (MTP Service Communication)", s.grpcPort)
	return nil
}

// Stop gracefully shuts down the MTP service
func (s *SimpleMTPService) Stop() {
	log.Println("🛑 Stopping Simple MTP Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop WebSocket server
	if s.webSocketServer != nil {
		if err := s.webSocketServer.Shutdown(ctx); err != nil {
			log.Printf("❌ Error shutting down WebSocket server: %v", err)
		}
	}

	// Stop health server
	if s.healthServer != nil {
		if err := s.healthServer.Shutdown(ctx); err != nil {
			log.Printf("❌ Error shutting down health server: %v", err)
		}
	}

	log.Println("✅ Simple MTP Service stopped")
}

// triggerProactiveOnboarding implements TR-369 proactive onboarding when MTP connection succeeds
// but agent doesn't send USP messages (like obuspa in factory default state)
func (s *SimpleMTPService) triggerProactiveOnboarding(clientID, selectedSubprotocol string) {
	// Per TR-369 Section 3.3.2: Controller-initiated discovery
	// Wait for initial stabilization period to see if agent sends messages
	stabilizationPeriod := 5 * time.Second
	log.Printf("🔍 ProactiveOnboarding: Monitoring client %s for %v (TR-369 stabilization)", clientID, stabilizationPeriod)

	time.Sleep(stabilizationPeriod)

	// Check if client is still connected and hasn't sent any USP messages
	s.mu.RLock()
	_, stillConnected := s.connectedClients[clientID]
	s.mu.RUnlock()

	if !stillConnected {
		log.Printf("📱 ProactiveOnboarding: Client %s disconnected during stabilization - skipping", clientID)
		return
	}

	// Generate a synthetic endpoint ID for the proactive onboarding
	// In real scenarios, this would be derived from client certificates or other MTP-level info
	endpointID := fmt.Sprintf("proto::proactive-%s", clientID)

	// Determine USP version - default to 1.3 if subprotocol doesn't specify
	uspVersion := "1.3"
	if selectedSubprotocol == "v1.usp" || selectedSubprotocol == "" {
		uspVersion = "1.3" // Default USP version
	}

	log.Printf("🚀 ProactiveOnboarding: Initiating TR-369 controller-driven discovery for %s (USP %s)", endpointID, uspVersion)

	// Create connection context per TR-369 requirements
	connectionContext := map[string]interface{}{
		"transport_type": "WebSocket",
		"client_id":      clientID,
		"subprotocol":    selectedSubprotocol,
		"initiated_by":   "controller",
		"reason":         "no_usp_messages_received",
		"mtp_version":    "1.0",
	}

	// Call USP service to handle proactive onboarding
	if uspHandler, ok := s.messageHandler.(*USPMessageHandler); ok {
		log.Printf("📡 ProactiveOnboarding: Notifying USP service of new connection: %s", endpointID)

		// Send proactive onboarding request to USP service
		// This follows TR-369 Section 3.3.2 - Controller-initiated Discovery
		go func() {
			err := uspHandler.notifyProactiveOnboarding(endpointID, uspVersion, connectionContext)
			if err != nil {
				log.Printf("❌ ProactiveOnboarding: Failed to notify USP service for %s: %v", endpointID, err)
			} else {
				log.Printf("✅ ProactiveOnboarding: Successfully initiated for %s", endpointID)
			}
		}()
	} else {
		log.Printf("❌ ProactiveOnboarding: Message handler is not USPMessageHandler for %s", endpointID)
	}
}

// triggerProactiveOnboardingWithRealEndpoint triggers proactive onboarding using the actual agent endpoint ID
func (h *USPMessageHandler) triggerProactiveOnboardingWithRealEndpoint(realEndpointID, uspVersion string, connectionContext map[string]interface{}) {
	// TR-369 stabilization period - wait 5 seconds after WebSocket connection
	time.Sleep(5 * time.Second)

	log.Printf("🚀 ProactiveOnboarding: Initiating TR-369 controller-driven discovery for %s (USP %s)", realEndpointID, uspVersion)

	// Notify USP service to start proactive onboarding with the real endpoint ID
	err := h.notifyProactiveOnboarding(realEndpointID, uspVersion, connectionContext)
	if err != nil {
		log.Printf("❌ ProactiveOnboarding: Failed to notify USP service for %s: %v", realEndpointID, err)
	} else {
		log.Printf("✅ ProactiveOnboarding: Successfully initiated for %s", realEndpointID)
	}
}

// handleUSPWebSocket handles WebSocket connections for USP messages
func (s *SimpleMTPService) handleUSPWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}

	clientID := r.Header.Get("X-Client-ID")
	if clientID == "" {
		clientID = fmt.Sprintf("client-%d", len(s.connectedClients))
	}

	s.mu.Lock()
	s.connectedClients[clientID] = time.Now()
	s.mu.Unlock()

	// Update active connections metric
	s.metrics.SetActiveConnections(float64(len(s.connectedClients)))

	// Log the selected subprotocol for debugging
	selectedSubprotocol := conn.Subprotocol()
	log.Printf("📱 WebSocket client connected: %s (subprotocol: %s)", clientID, selectedSubprotocol)

	// Note: Proactive onboarding will be triggered after we receive the WebSocket Connect record
	// and extract the real agent endpoint ID (not synthetic client ID)

	defer func() {
		s.mu.Lock()
		delete(s.connectedClients, clientID)
		s.mu.Unlock()

		// Update active connections metric
		s.metrics.SetActiveConnections(float64(len(s.connectedClients)))

		conn.Close()
		log.Printf("📱 WebSocket client disconnected: %s", clientID)
	}()

	// Handle messages from client
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error: %v", err)
			}
			break
		}

		// Handle WebSocket PING/PONG frames - obuspa handles its own keep-alive
		if messageType == websocket.PingMessage {
			log.Printf("🏓 Received WebSocket PING from client %s - letting obuspa handle keep-alive", clientID)
			// Do NOT send PONG response - obuspa expects to handle its own keep-alive mechanism
			// Sending WebSocket PONG confuses obuspa's USP record parsing
			continue
		}

		if messageType == websocket.PongMessage {
			log.Printf("🏓 Received WebSocket PONG from client %s", clientID)
			// PONG received - connection is alive, no response needed
			continue
		}

		if messageType == websocket.TextMessage {
			log.Printf("⚠️ Received unexpected text message from client %s: %s", clientID, string(payload))
			continue
		}

		if messageType == websocket.BinaryMessage && s.messageHandler != nil {
			recvSize := len(payload)

			// Skip empty binary messages
			if recvSize == 0 {
				log.Printf("⚠️ Received empty binary message from client %s", clientID)
				continue
			}

			log.Printf("📨 Received USP message from client %s (size: %d bytes)", clientID, recvSize)

			// Detect USP version for metrics
			version, _ := s.messageHandler.DetectUSPVersion(payload)
			if version == "" {
				version = "unknown"
			}

			start := time.Now()

			// Create connection context for agent registration
			connCtx := &ConnectionContext{
				ClientID:      clientID,
				TransportType: "WebSocket",
				WebSocketConn: conn,
			}

			response, err := s.messageHandler.ProcessUSPMessageWithContext(payload, connCtx)
			procDur := time.Since(start)

			if err != nil {
				log.Printf("❌ Failed to process USP message from client %s (size=%d, version=%s, duration=%s): %v", clientID, recvSize, version, procDur, err)
				s.metrics.RecordUSPError("mtp-service", version, "unknown", "processing_error")
				// Keep the connection alive; short backoff to avoid tight loop
				time.Sleep(150 * time.Millisecond)
				continue
			}

			log.Printf("✅ Processed USP message from client %s (version=%s, in %s, responseSize=%d)", clientID, version, procDur, len(response))
			s.metrics.RecordUSPMessage("mtp-service", version, "unknown", "websocket")

			if len(response) > 0 {
				if err := conn.WriteMessage(websocket.BinaryMessage, response); err != nil {
					log.Printf("❌ Failed to send response to client %s: %v", clientID, err)
					break
				}
				log.Printf("📤 Sent USP response to client %s (size: %d bytes)", clientID, len(response))
			}
		}
	}
}

// handleHealth provides health check endpoint
func (s *SimpleMTPService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	s.mu.RLock()
	clientCount := len(s.connectedClients)
	s.mu.RUnlock()

	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "mtp-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"protocols": map[string]interface{}{
			"mqtt":        map[string]interface{}{"enabled": true, "status": "ready"},
			"stomp":       map[string]interface{}{"enabled": true, "status": "ready"},
			"websocket":   map[string]interface{}{"enabled": true, "port": s.webSocketPort},
			"unix_socket": map[string]interface{}{"enabled": true, "path": s.unixSocketPath},
		},
		"connected_clients": clientCount,
	}

	fmt.Fprintf(w, "%s", mustMarshalJSON(health))
}

// handleStatus provides detailed status information
func (s *SimpleMTPService) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	clients := make(map[string]string)
	for id, connTime := range s.connectedClients {
		clients[id] = connTime.Format(time.RFC3339)
	}
	s.mu.RUnlock()

	status := map[string]interface{}{
		"service": "OpenUSP MTP Service",
		"version": "1.0.0",
		"uptime":  time.Since(startTime).String(),
		"protocols": map[string]interface{}{
			"supported":    []string{"MQTT", "STOMP", "WebSocket", "Unix Domain Socket"},
			"usp_versions": []string{"1.3", "1.4"},
		},
		"connected_clients": clients,
		"message_stats": map[string]int{
			"messages_processed": messageCount,
			"errors_encountered": errorCount,
		},
	}

	fmt.Fprintf(w, "%s", mustMarshalJSON(status))
}

// handleSimulate simulates processing a USP message
func (s *SimpleMTPService) handleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Create a sample USP 1.4 record
	sampleRecord := &v1_4.Record{
		Version: "1.4",
		ToId:    "agent::sample",
		FromId:  "controller::mtp-service",
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: []byte("sample USP message payload"),
			},
		},
	}

	recordData, err := proto.Marshal(sampleRecord)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal record: %v", err), 500)
		return
	}

	// Simulate processing
	messageCount++
	version, _ := s.messageHandler.DetectUSPVersion(recordData)

	result := map[string]interface{}{
		"action":           "simulate_usp_message",
		"message_size":     len(recordData),
		"detected_version": version,
		"from_id":          sampleRecord.FromId,
		"to_id":            sampleRecord.ToId,
		"processing_time":  "2.5ms",
		"result":           "success",
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}

	fmt.Fprintf(w, "%s", mustMarshalJSON(result))
}

// USPMessageHandler provides comprehensive USP message handling with parsing
type USPMessageHandler struct {
	parser           *usp.USPParser
	connectionClient *client.OpenUSPConnectionClient
	mtpConfig        *config.MTPConfig
	stompBroker      *mtp.STOMPBroker // For sending messages via STOMP

	// Agent connection tracking
	connectedAgents map[string]*ConnectedAgent // agentID -> connection info
	agentsMutex     sync.RWMutex

	// Embed the unimplemented server to satisfy the gRPC interface
	mtpservice.UnimplementedMTPServiceServer
}

// ConnectedAgent represents a connected agent
type ConnectedAgent struct {
	AgentID       string
	ClientID      string
	TransportType string
	ConnectedAt   time.Time
	WebSocketConn *websocket.Conn // For WebSocket transport
}

// NewUSPMessageHandler creates a new USP message handler that forwards to USP service
func NewUSPMessageHandler(mtpConfig *config.MTPConfig) *USPMessageHandler {
	// Create connection client for service discovery
	connectionClient := client.NewOpenUSPConnectionClient(30 * time.Second)

	return &USPMessageHandler{
		parser:           usp.NewUSPParser(),
		connectionClient: connectionClient,
		mtpConfig:        mtpConfig,
		connectedAgents:  make(map[string]*ConnectedAgent),
	}
}

// SetSTOMPBroker sets the STOMP broker for sending messages to agents
func (h *USPMessageHandler) SetSTOMPBroker(broker *mtp.STOMPBroker) {
	h.stompBroker = broker
	log.Printf("✅ STOMP broker attached to message handler")
}

// getUSPServiceClient gets a USP service client via connection manager
func (h *USPMessageHandler) getUSPServiceClient() (uspservice.USPServiceClient, error) {
	return h.connectionClient.GetUSPServiceClient()
}

func (h *USPMessageHandler) DetectUSPVersion(data []byte) (string, error) {
	version, err := h.parser.DetectVersion(data)
	if err != nil {
		return "", err
	}
	return string(version), nil
}

func (h *USPMessageHandler) ProcessUSPMessage(data []byte) ([]byte, error) {
	return h.ProcessUSPMessageWithContext(data, nil)
}

func (h *USPMessageHandler) ProcessUSPMessageWithContext(data []byte, connCtx *ConnectionContext) ([]byte, error) {
	messageCount++

	// First, check if this is an MTP-level message that should be handled locally
	// Parse the USP record to determine the message type
	parsed, err := h.parser.ParseUSP(data)
	if err != nil {
		log.Printf("❌ Failed to parse USP record: %v", err)
		return nil, fmt.Errorf("failed to parse USP record: %w", err)
	}

	// Handle MTP-level messages directly (per TR-369 specification)
	if parsed.Record != nil {
		log.Printf("🔍 DEBUG: Received record type: %s, Version: %s, FromID: %s, ToID: %s",
			parsed.Record.RecordType, parsed.Record.Version, parsed.Record.FromID, parsed.Record.ToID)

		// Register agent connection when we see WebSocketConnect (for transport layer tracking)
		if parsed.Record.RecordType == "WebSocketConnect" && connCtx != nil {
			h.registerAgent(parsed.Record.FromID, connCtx)
			log.Printf("🔗 Agent %s connected via WebSocket - forwarding connect record to USP service", parsed.Record.FromID)

			// Trigger proactive onboarding with the REAL endpoint ID from the USP record
			// This addresses the obuspa factory default issue where agents connect but don't send NOTIFY
			realEndpointID := parsed.Record.FromID
			uspVersion := string(parsed.Record.Version)

			log.Printf("🚀 ProactiveOnboarding: Triggering with real endpoint ID: %s (USP %s)", realEndpointID, uspVersion)

			// Create connection context for proactive onboarding
			proactiveContext := map[string]interface{}{
				"transport_type": connCtx.TransportType,
				"client_id":      connCtx.ClientID,
				"endpoint_id":    realEndpointID,
				"usp_version":    uspVersion,
				"initiated_by":   "controller",
				"reason":         "no_usp_messages_received",
				"mtp_version":    "1.0",
			}

			// Trigger proactive onboarding after stabilization period
			go h.triggerProactiveOnboardingWithRealEndpoint(realEndpointID, uspVersion, proactiveContext)
		}

		// Forward ALL USP records to USP Service - MTP only handles transport layer
		log.Printf("� Forwarding USP record (%s) to USP service for processing", parsed.Record.RecordType)

		// All USP records (including connect records) should be processed by USP Service
		// MTP Service only provides transport layer functionality
	}

	// Connection manager handles USP service discovery and connection pooling

	// Forward USP message to USP service for processing
	ctx := context.Background()

	// Add debugging for data being forwarded
	dataPreview := data
	if len(dataPreview) > 64 {
		dataPreview = dataPreview[:64]
	}
	log.Printf("🔍 MTP Debug - Forwarding to USP service: %d bytes, hex: %x", len(data), dataPreview)

	transportType := "WebSocket"
	if connCtx != nil {
		transportType = connCtx.TransportType
	}

	// Get USP service client via connection manager
	uspClient, err := h.getUSPServiceClient()
	if err != nil {
		log.Printf("❌ Failed to get USP service client: %v", err)
		return h.processUSPMessageLocallyWithContext(data, connCtx)
	}

	// Create USP message request
	request := &uspservice.USPMessageRequest{
		UspData:       data,
		TransportType: transportType,
		ClientId:      "mtp-client",
		Timestamp:     time.Now().UnixNano(),
		Metadata:      make(map[string]string),
	}

	response, err := uspClient.ProcessUSPMessage(ctx, request)
	if err != nil {
		errorCount++
		log.Printf("❌ USP service processing failed: %v", err)

		// Retry once with a new client (connection manager handles reconnection)
		log.Printf("🔄 Retrying with new USP service client...")
		uspClient2, err2 := h.getUSPServiceClient()
		if err2 != nil {
			log.Printf("❌ Failed to get USP service client for retry: %v", err2)
		} else {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel2()

			response, err = uspClient2.ProcessUSPMessage(ctx2, request)
			if err != nil {
				log.Printf("❌ USP service processing failed after retry: %v", err)
			} else {
				log.Printf("✅ USP service call succeeded after retry")
				// Continue with successful response processing below
				goto processResponse
			}
		}

		// USP messages must be processed by USP Service - don't fall back to local processing
		log.Printf("❌ USP Service is required for USP message processing - cannot proceed without it")
		return nil, fmt.Errorf("USP service required for message processing")
	}

processResponse:

	if response.Status != "success" {
		log.Printf("⚠️ USP service returned error: %s", response.ErrorMessage)
	}

	if response.DeviceOnboarded {
		log.Printf("✅ Device onboarding completed via USP service")
		if response.DeviceInfo != nil {
			log.Printf("   Device: %s (S/N: %s)", response.DeviceInfo.EndpointId, response.DeviceInfo.SerialNumber)
		}
	}

	return response.ResponseData, nil
}

// registerAgent registers a connected agent for message routing
func (h *USPMessageHandler) registerAgent(agentID string, connCtx *ConnectionContext) {
	if connCtx == nil {
		log.Printf("⚠️ Cannot register agent %s: no connection context", agentID)
		return
	}

	h.agentsMutex.Lock()
	defer h.agentsMutex.Unlock()

	connectedAgent := &ConnectedAgent{
		AgentID:       agentID,
		ClientID:      connCtx.ClientID,
		TransportType: connCtx.TransportType,
		ConnectedAt:   time.Now(),
		WebSocketConn: connCtx.WebSocketConn,
	}

	h.connectedAgents[agentID] = connectedAgent
	log.Printf("🔗 Registered agent %s (client: %s, transport: %s)", agentID, connCtx.ClientID, connCtx.TransportType)
}

// processUSPMessageLocally handles MTP-level messages and provides fallback for USP messages
func (h *USPMessageHandler) processUSPMessageLocally(data []byte) ([]byte, error) {
	return h.processUSPMessageLocallyWithContext(data, nil)
}

// processUSPMessageLocallyWithContext handles MTP-level messages and provides fallback for USP messages
func (h *USPMessageHandler) processUSPMessageLocallyWithContext(data []byte, connCtx *ConnectionContext) ([]byte, error) {
	// Parse the USP record to determine if it's MTP-level or USP-level
	parsed, err := h.parser.ParseUSP(data)
	if err != nil {
		log.Printf("❌ Failed to parse USP record: %v", err)
		return nil, fmt.Errorf("failed to parse USP record: %w", err)
	}

	// Handle MTP-level messages directly (per TR-369 specification)
	if parsed.Record != nil {
		log.Printf("🔍 DEBUG: Received record type: %s, Version: %s, FromID: %s, ToID: %s",
			parsed.Record.RecordType, parsed.Record.Version, parsed.Record.FromID, parsed.Record.ToID)

		// Check if this record contains a USP message - if not, it's transport-level only
		if parsed.Message == nil && (parsed.Record.RecordType == "NoSessionContext" || parsed.Record.RecordType == "SessionContext") {
			log.Printf("⚠️ Record contains no USP message - treating as transport acknowledgment")
			// Send a simple transport acknowledgment instead of forwarding to USP service
			return []byte("Transport acknowledgment"), nil
		}

		switch parsed.Record.RecordType {
		case "WebSocketConnect":
			log.Printf("🔌 Handling WebSocket Connect at MTP level")
			return h.handleWebSocketConnect(parsed)
		case "MQTTConnect":
			log.Printf("🔌 Handling MQTT Connect at MTP level")
			return h.handleMQTTConnect(parsed)
		case "STOMPConnect":
			log.Printf("🔌 Handling STOMP Connect at MTP level")
			log.Printf("📥 STOMP: Received STOMPConnect from agent: %s", parsed.Record.FromID)
			log.Printf("📤 STOMP: Will send response to queue: /queue/usp.agent")
			return h.handleSTOMPConnect(parsed)
		case "UDSConnect":
			log.Printf("🔌 Handling UDS Connect at MTP level")
			return h.handleUDSConnect(parsed)
		case "Disconnect":
			log.Printf("🔌 Handling Disconnect at MTP level")
			return h.handleDisconnect(parsed)
		case "NoSessionContext", "SessionContext":
			// These should contain USP messages - check if they actually do
			if parsed.Message == nil {
				log.Printf("⚠️ USP record contains no message - may be transport-level keepalive")
				// Don't forward empty records to USP service - send simple acknowledgment
				return []byte("ACK"), nil
			}

			// USP messages should NOT be processed locally in MTP Service
			// They should have been forwarded through the main flow
			log.Printf("❌ USP message reached local processing - this should not happen!")
			log.Printf("   Record type: %s, Message type: %s", parsed.Record.RecordType, parsed.Message.MsgType)
			return nil, fmt.Errorf("USP messages must be processed by USP Service, not locally")
		default:
			log.Printf("❌ Unknown record type: %s", parsed.Record.RecordType)
			return nil, fmt.Errorf("unknown record type: %s", parsed.Record.RecordType)
		}
	}

	return nil, fmt.Errorf("invalid USP record")
}

// createWebSocketConnectAck13 creates USP 1.3 WebSocket connect acknowledgment
func (h *USPMessageHandler) createWebSocketConnectAck13(fromID, toID string) ([]byte, error) {
	// Per TR-369 specification: Controller responds to Agent's WebSocketConnect
	// with its own WebSocketConnect record to complete session establishment
	ackRecord := &v1_3.Record{
		Version:         "1.3",
		ToId:            fromID, // Responding to the Agent
		FromId:          toID,   // From the Controller
		PayloadSecurity: v1_3.Record_PLAINTEXT,
		RecordType: &v1_3.Record_WebsocketConnect{
			WebsocketConnect: &v1_3.WebSocketConnectRecord{
				// Empty WebSocketConnect record per TR-369 - signals session ready
			},
		},
	}

	data, err := proto.Marshal(ackRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.3 WebSocket connect ack: %v", err)
		return nil, fmt.Errorf("failed to create WebSocket connect ack")
	}

	log.Printf("✅ Created USP 1.3 WebSocketConnect acknowledgment: %s → %s", toID, fromID)
	return data, nil
}

// createWebSocketConnectAck14 creates USP 1.4 WebSocket connect acknowledgment
func (h *USPMessageHandler) createWebSocketConnectAck14(fromID, toID string) ([]byte, error) {
	// Per TR-369 specification: Controller responds to Agent's WebSocketConnect
	// with its own WebSocketConnect record to complete session establishment
	ackRecord := &v1_4.Record{
		Version:         "1.4",
		ToId:            fromID, // Responding to the Agent
		FromId:          toID,   // From the Controller
		PayloadSecurity: v1_4.Record_PLAINTEXT,
		RecordType: &v1_4.Record_WebsocketConnect{
			WebsocketConnect: &v1_4.WebSocketConnectRecord{
				// Empty WebSocketConnect record per TR-369 - signals session ready
			},
		},
	}

	data, err := proto.Marshal(ackRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.4 WebSocket connect ack: %v", err)
		return nil, fmt.Errorf("failed to create WebSocket connect ack")
	}

	log.Printf("✅ Created USP 1.4 WebSocketConnect acknowledgment: %s → %s", toID, fromID)
	return data, nil
}

// createMQTTConnectAck13 creates USP 1.3 MQTT connect acknowledgment
func (h *USPMessageHandler) createMQTTConnectAck13(fromID, toID string) ([]byte, error) {
	// Per TR-369 specification: Controller responds to Agent's MQTTConnect
	// with its own MQTTConnect record to complete session establishment
	ackRecord := &v1_3.Record{
		Version:         "1.3",
		ToId:            fromID, // Responding to the Agent
		FromId:          toID,   // From the Controller
		PayloadSecurity: v1_3.Record_PLAINTEXT,
		RecordType: &v1_3.Record_MqttConnect{
			MqttConnect: &v1_3.MQTTConnectRecord{
				Version:         v1_3.MQTTConnectRecord_V3_1_1,
				SubscribedTopic: "usp/controller",
			},
		},
	}

	data, err := proto.Marshal(ackRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.3 MQTT connect ack: %v", err)
		return nil, fmt.Errorf("failed to create MQTT connect ack")
	}

	log.Printf("✅ Created USP 1.3 MQTTConnect acknowledgment: %s → %s", toID, fromID)
	return data, nil
}

// createMQTTConnectAck14 creates USP 1.4 MQTT connect acknowledgment
func (h *USPMessageHandler) createMQTTConnectAck14(fromID, toID string) ([]byte, error) {
	// Per TR-369 specification: Controller responds to Agent's MQTTConnect
	// with its own MQTTConnect record to complete session establishment
	ackRecord := &v1_4.Record{
		Version:         "1.4",
		ToId:            fromID, // Responding to the Agent
		FromId:          toID,   // From the Controller
		PayloadSecurity: v1_4.Record_PLAINTEXT,
		RecordType: &v1_4.Record_MqttConnect{
			MqttConnect: &v1_4.MQTTConnectRecord{
				Version:         v1_4.MQTTConnectRecord_V3_1_1,
				SubscribedTopic: "usp/controller",
			},
		},
	}

	data, err := proto.Marshal(ackRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.4 MQTT connect ack: %v", err)
		return nil, fmt.Errorf("failed to create MQTT connect ack")
	}

	log.Printf("✅ Created USP 1.4 MQTTConnect acknowledgment: %s → %s", toID, fromID)
	return data, nil
}

// createSTOMPConnectAck13 creates USP 1.3 STOMP connect acknowledgment
func (h *USPMessageHandler) createSTOMPConnectAck13(fromID, toID string) ([]byte, error) {
	// Per TR-369 specification: Controller responds to Agent's STOMPConnect
	// with its own STOMPConnect record to complete session establishment

	// Use config-driven agent queue for SubscribedDestination
       agentDest := "/queue/usp.agent"

	ackRecord := &v1_3.Record{
		Version:         "1.3",
		ToId:            fromID, // Responding to the Agent
		FromId:          toID,   // From the Controller
		PayloadSecurity: v1_3.Record_PLAINTEXT,
		RecordType: &v1_3.Record_StompConnect{
			StompConnect: &v1_3.STOMPConnectRecord{
				Version:               v1_3.STOMPConnectRecord_V1_2,
				SubscribedDestination: agentDest,
			},
		},
	}

	data, err := proto.Marshal(ackRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.3 STOMP connect ack: %v", err)
		return nil, fmt.Errorf("failed to create STOMP connect ack")
	}

	log.Printf("✅ Created USP 1.3 STOMPConnect acknowledgment: %s → %s", toID, fromID)
	return data, nil
}

// createSTOMPConnectAck14 creates USP 1.4 STOMP connect acknowledgment
func (h *USPMessageHandler) createSTOMPConnectAck14(fromID, toID string) ([]byte, error) {
	// Per TR-369 specification: Controller responds to Agent's STOMPConnect
	// with its own STOMPConnect record to complete session establishment

	// Use config-driven agent queue for SubscribedDestination
       agentDest := "/queue/usp.agent"

	ackRecord := &v1_4.Record{
		Version:         "1.4",
		ToId:            fromID, // Responding to the Agent
		FromId:          toID,   // From the Controller
		PayloadSecurity: v1_4.Record_PLAINTEXT,
		RecordType: &v1_4.Record_StompConnect{
			StompConnect: &v1_4.STOMPConnectRecord{
				Version:               v1_4.STOMPConnectRecord_V1_2,
				SubscribedDestination: agentDest,
			},
		},
	}

	data, err := proto.Marshal(ackRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.4 STOMP connect ack: %v", err)
		return nil, fmt.Errorf("failed to create STOMP connect ack")
	}

	log.Printf("✅ Created USP 1.4 STOMPConnect acknowledgment: %s → %s", toID, fromID)
	return data, nil
}

// createUDSConnectAck13 creates USP 1.3 UDS connect acknowledgment
func (h *USPMessageHandler) createUDSConnectAck13(fromID, toID string) ([]byte, error) {
	// Per TR-369 specification: Controller responds to Agent's UDSConnect
	// with its own UDSConnect record to complete session establishment
	ackRecord := &v1_3.Record{
		Version:         "1.3",
		ToId:            fromID, // Responding to the Agent
		FromId:          toID,   // From the Controller
		PayloadSecurity: v1_3.Record_PLAINTEXT,
		RecordType: &v1_3.Record_UdsConnect{
			UdsConnect: &v1_3.UDSConnectRecord{
				// UDSConnectRecord is an empty message per protocol
			},
		},
	}

	data, err := proto.Marshal(ackRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.3 UDS connect ack: %v", err)
		return nil, fmt.Errorf("failed to create UDS connect ack")
	}

	log.Printf("✅ Created USP 1.3 UDSConnect acknowledgment: %s → %s", toID, fromID)
	return data, nil
}

// createUDSConnectAck14 creates USP 1.4 UDS connect acknowledgment
func (h *USPMessageHandler) createUDSConnectAck14(fromID, toID string) ([]byte, error) {
	// Per TR-369 specification: Controller responds to Agent's UDSConnect
	// with its own UDSConnect record to complete session establishment
	ackRecord := &v1_4.Record{
		Version:         "1.4",
		ToId:            fromID, // Responding to the Agent
		FromId:          toID,   // From the Controller
		PayloadSecurity: v1_4.Record_PLAINTEXT,
		RecordType: &v1_4.Record_UdsConnect{
			UdsConnect: &v1_4.UDSConnectRecord{
				// UDSConnectRecord is an empty message per protocol
			},
		},
	}

	data, err := proto.Marshal(ackRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.4 UDS connect ack: %v", err)
		return nil, fmt.Errorf("failed to create UDS connect ack")
	}

	log.Printf("✅ Created USP 1.4 UDSConnect acknowledgment: %s → %s", toID, fromID)
	return data, nil
}

// MTP-level connect handlers per TR-369 specification

// handleWebSocketConnect processes WebSocket connect records at MTP level
func (h *USPMessageHandler) handleWebSocketConnect(parsed *usp.ParsedUSP) ([]byte, error) {
	log.Printf("✅ WebSocket connection establishment request received from Agent: %s", parsed.Record.FromID)

	// According to TR-369 specification, when an Agent sends a WebSocketConnect record,
	// the Controller MUST respond with a WebSocketConnect record to complete the session establishment
	log.Printf("🔌 Sending WebSocketConnect acknowledgment per TR-369 specification")

	agentID := parsed.Record.FromID
	controllerID := parsed.Record.ToID

	log.Printf("📋 WebSocket session established - Agent %s ready for USP message exchange", agentID)

	// Create WebSocket connect acknowledgment
	var ackData []byte
	var err error
	if parsed.Record.Version == usp.Version14 {
		ackData, err = h.createWebSocketConnectAck14(agentID, controllerID)
	} else {
		ackData, err = h.createWebSocketConnectAck13(agentID, controllerID)
	}

	if err != nil {
		return nil, err
	}

	// WebSocket connection established - USP Service will handle discovery when agent sends USP messages
	log.Printf("✅ WebSocket connection established with agent %s - ready for USP message flow", agentID)

	return ackData, nil
}

// handleMQTTConnect processes MQTT connect records at MTP level
func (h *USPMessageHandler) handleMQTTConnect(parsed *usp.ParsedUSP) ([]byte, error) {
	log.Printf("✅ MQTT connection establishment request received from Agent: %s", parsed.Record.FromID)
	log.Printf("🔌 Sending MQTTConnect acknowledgment per TR-369 specification")

	agentID := parsed.Record.FromID
	controllerID := parsed.Record.ToID

	log.Printf("📋 MQTT session established - Agent %s ready for USP message exchange", agentID)

	// Create MQTT connect acknowledgment
	var ackData []byte
	var err error
	if parsed.Record.Version == usp.Version14 {
		ackData, err = h.createMQTTConnectAck14(agentID, controllerID)
	} else {
		ackData, err = h.createMQTTConnectAck13(agentID, controllerID)
	}

	if err != nil {
		return nil, err
	}

	// MQTT connection established - USP Service will handle discovery when agent sends USP messages
	log.Printf("✅ MQTT connection established with agent %s - ready for USP message flow", agentID)

	return ackData, nil
}

// handleSTOMPConnect processes STOMP connect records at MTP level
func (h *USPMessageHandler) handleSTOMPConnect(parsed *usp.ParsedUSP) ([]byte, error) {
	log.Printf("📡 Processing STOMP Connect: %s → %s", parsed.Record.FromID, parsed.Record.ToID)
	log.Printf("📥 STOMP: Message received from queue: /queue/usp.controller (agent sent to controller)")
	log.Printf("📤 STOMP: Response will be sent via queue: /queue/usp.agent (controller to agent)")

	// Register the STOMP agent as connected
	agentID := parsed.Record.FromID
	h.agentsMutex.Lock()
	h.connectedAgents[agentID] = &ConnectedAgent{
		AgentID:       agentID,
		ClientID:      agentID, // For STOMP, agent ID is the client ID
		TransportType: "STOMP",
		ConnectedAt:   time.Now(),
		WebSocketConn: nil, // No WebSocket for STOMP
	}
	h.agentsMutex.Unlock()
	log.Printf("🔗 Registered STOMP agent %s in connected agents map", agentID)

	// Per TR-369: Controller responds with STOMPConnect acknowledgment
	var ackData []byte
	var err error

	// Determine version and create appropriate acknowledgment
	if parsed.Record.Version == usp.Version13 {
		ackData, err = h.createSTOMPConnectAck13(parsed.Record.FromID, parsed.Record.ToID)
	} else if parsed.Record.Version == usp.Version14 {
		ackData, err = h.createSTOMPConnectAck14(parsed.Record.FromID, parsed.Record.ToID)
	} else {
		log.Printf("❌ Unsupported USP version for STOMP connect: %v", parsed.Record.Version)
		return nil, fmt.Errorf("unsupported USP version: %v", parsed.Record.Version)
	}

	if err != nil {
		log.Printf("❌ Failed to create STOMP connect acknowledgment: %v", err)
		return nil, err
	}

	log.Printf("✅ STOMP Connect acknowledgment created for %s", parsed.Record.FromID)
	log.Printf("✅ STOMP connection established - ready for USP message flow")
	log.Printf("📤 STOMP: Acknowledgment ready to send to agent via /queue/usp.agent")
	log.Printf("📊 STOMP: Message size: %d bytes", len(ackData))

	return ackData, nil
}

// handleUDSConnect processes Unix Domain Socket connect records at MTP level
func (h *USPMessageHandler) handleUDSConnect(parsed *usp.ParsedUSP) ([]byte, error) {
	log.Printf("📡 Processing UDS Connect: %s → %s", parsed.Record.FromID, parsed.Record.ToID)

	// Per TR-369: Controller responds with UDSConnect acknowledgment
	var ackData []byte
	var err error

	// Determine version and create appropriate acknowledgment
	if parsed.Record.Version == usp.Version13 {
		ackData, err = h.createUDSConnectAck13(parsed.Record.FromID, parsed.Record.ToID)
	} else if parsed.Record.Version == usp.Version14 {
		ackData, err = h.createUDSConnectAck14(parsed.Record.FromID, parsed.Record.ToID)
	} else {
		log.Printf("❌ Unsupported USP version for UDS connect: %v", parsed.Record.Version)
		return nil, fmt.Errorf("unsupported USP version: %v", parsed.Record.Version)
	}

	if err != nil {
		log.Printf("❌ Failed to create UDS connect acknowledgment: %v", err)
		return nil, err
	}

	log.Printf("✅ UDS Connect acknowledgment created for %s", parsed.Record.FromID)
	log.Printf("✅ Unix Domain Socket connection established - ready for USP message flow")

	return ackData, nil
}

// handleDisconnect processes disconnect records at MTP level
func (h *USPMessageHandler) handleDisconnect(parsed *usp.ParsedUSP) ([]byte, error) {
	log.Printf("✅ Disconnect processed at MTP level")
	// No response needed for disconnect
	return nil, nil
}

// Global counters for service monitoring
var (
	startTime    = time.Now()
	messageCount = 0
	errorCount   = 0
)

// mustMarshalJSON marshals to JSON or panics
func mustMarshalJSON(v interface{}) string {
	// Simple JSON marshaling
	switch val := v.(type) {
	case map[string]interface{}:
		return marshalMapToJSON(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func marshalMapToJSON(m map[string]interface{}) string {
	result := "{"
	first := true
	for k := range m {
		if !first {
			result += ","
		}
		first = false
		result += fmt.Sprintf("\"%s\":", k)

		// Add value formatting logic here if needed
	}
	result += "}"
	return result
}

func main() {
	log.Printf("🚀 Starting OpenUSP MTP Service...")

	// Command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.GetFullVersion("OpenUSP MTP Service"))
		return
	}

	if *showHelp {
		fmt.Println("OpenUSP MTP Service - Multi-Protocol Message Transport")
		fmt.Println("====================================================")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  mtp-service [flags]")
		fmt.Println("")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Environment Variables:")
		fmt.Println("  SERVICE_PORT       - Server port (default: 8081)")
		fmt.Println("  MTP_PORT           - MTP service port (default: 8081)")
		return
	}

	// Load MTP configuration from YAML
	healthPort := 8082 // Default health/admin port
	grpcPort := 50300  // Default gRPC port
	serviceConfig := DefaultConfigWithPorts(healthPort, grpcPort)
	mtpConfig := config.LoadMTPConfig()

	log.Printf("🎯 Using configured ports: WebSocket=%d, Health=%d, gRPC=%d", serviceConfig.WebSocketPort, healthPort, grpcPort)

	// USP service address and port loaded from YAML config
	uspServicePort := mtpConfig.GRPCPort
	uspServiceAddr := fmt.Sprintf("%s:%d", mtpConfig.Address, uspServicePort)
	log.Printf("✅ Using USP service address from config: %s", uspServiceAddr)

	handler := NewUSPMessageHandler(mtpConfig)

	// Create MTP service
	mtpService, err := NewSimpleMTPService(serviceConfig, handler)
	if err != nil {
		log.Fatalf("Failed to create MTP service: %v", err)
	}

	// Load STOMP configuration and start STOMP broker if enabled
	if mtpConfig.STOMPEnabled {
		// Load YAML config directly for STOMP configuration
		yamlPath := "configs/openusp.yml"
		data, err := os.ReadFile(yamlPath)
		if err != nil {
			log.Printf("❌ Failed to read YAML config: %v", err)
		} else {
			var yamlConfig struct {
				MTP struct {
					STOMP struct {
						BrokerURL    string `yaml:"broker_url"`
						Username     string `yaml:"username"`
						Password     string `yaml:"password"`
						Destinations struct {
							Controller string `yaml:"controller"`
							Agent      string `yaml:"agent"`
						} `yaml:"destinations"`
					} `yaml:"stomp"`
				} `yaml:"mtp"`
			}
			
			if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
				log.Printf("❌ Failed to parse YAML config: %v", err)
			} else {
				stompConfig := yamlConfig.MTP.STOMP
				stompDestinations := []string{}
				
				if stompConfig.Destinations.Controller != "" {
					stompDestinations = append(stompDestinations, stompConfig.Destinations.Controller)
				}
				if stompConfig.Destinations.Agent != "" {
					stompDestinations = append(stompDestinations, stompConfig.Destinations.Agent)
				}
				
				log.Printf("🔌 Initializing STOMP broker with destinations: %v", stompDestinations)
				log.Printf("🔌 STOMP broker URL: %s", stompConfig.BrokerURL)
				
				// Debug logging for STOMP queue configuration
				log.Printf("📡 STOMP Queue Configuration:")
				log.Printf("   📥 Subscribing to (MTP will listen): %s", stompConfig.Destinations.Controller)
				log.Printf("   📤 Sending to (MTP will publish): %s", stompConfig.Destinations.Agent)
				log.Printf("   🔄 MTP Service Role: Controller-side message processor")
				log.Printf("   📋 Queue Usage:")
				log.Printf("      • Receive agent messages via: %s", stompConfig.Destinations.Controller)
				log.Printf("      • Send responses to agents via: %s", stompConfig.Destinations.Agent)
				
				// Store STOMP credentials for broker connection
				stompUsername := stompConfig.Username
				stompPassword := stompConfig.Password
				if stompUsername == "" {
					stompUsername = "guest"
					log.Printf("⚠️  STOMP username not found in config, using default: guest")
				} else {
					log.Printf("🔑 STOMP username from config: %s", stompUsername)
				}
				if stompPassword == "" {
					stompPassword = "guest"
					log.Printf("⚠️  STOMP password not found in config, using default: guest")
				} else {
					log.Printf("🔑 STOMP password from config: ****")
				}
				
				// Initialize and start actual STOMP broker connection
				log.Printf("🔌 Creating STOMP broker instance...")
				broker, err := mtp.NewSTOMPBrokerWithAuth(stompConfig.BrokerURL, stompDestinations, stompUsername, stompPassword)
				if err != nil {
					log.Printf("❌ Failed to create STOMP broker: %v", err)
				} else {
					// Set the message handler on the STOMP broker
					log.Printf("🔗 Connecting STOMP broker to USP message handler...")
					broker.SetMessageHandler(handler)
					handler.SetSTOMPBroker(broker) // Allow handler to send via STOMP
					log.Printf("✅ STOMP broker message handler configured")
					
					log.Printf("🚀 Starting STOMP broker connection...")
					go func() {
						log.Printf("📡 STOMP: Attempting connection to RabbitMQ at %s", stompConfig.BrokerURL)
						stompCtx := context.Background()
						if err := broker.Start(stompCtx); err != nil {
							log.Printf("❌ STOMP broker error: %v", err)
						}
					}()
					log.Printf("✅ STOMP broker started and listening for messages from /queue/usp.controller")
				}
			}
		}
	}

	// Start MTP service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mtpService.Start(ctx); err != nil {
		log.Fatalf("Failed to start MTP service: %v", err)
	}

	log.Printf("🚀 MTP Service started successfully")
	log.Printf("   └── WebSocket Port: %d (USP Protocol)", serviceConfig.WebSocketPort)
	log.Printf("   └── Health API Port: %d (Static)", serviceConfig.HealthPort)
	log.Printf("   └── Port Configuration: ✅ Fixed")
	// Demo UI endpoint removed in production version
	log.Printf("   └── Health Check: http://localhost:%d/health", serviceConfig.HealthPort)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Printf("� Received shutdown signal")

	// Fixed ports – no service deregistration needed

	cancel()
	mtpService.Stop()
	log.Printf("✅ MTP Service stopped successfully")
}

// createTransportErrorResponse13 creates a simple USP 1.3 error indicating transport/service issues
func (h *USPMessageHandler) createTransportErrorResponse13() []byte {
	// Create minimal error record indicating USP service unavailable
	errorRecord := &v1_3.Record{
		Version: "1.3",
		ToId:    "proto://unknown-endpoint",
		FromId:  "proto://openusp-mtp-service",
		RecordType: &v1_3.Record_NoSessionContext{
			NoSessionContext: &v1_3.NoSessionContextRecord{
				Payload: []byte("USP service temporarily unavailable"),
			},
		},
	}

	data, err := proto.Marshal(errorRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.3 transport error: %v", err)
		return []byte("Transport error - USP service unavailable")
	}
	return data
}

// createTransportErrorResponse14 creates a simple USP 1.4 error indicating transport/service issues
func (h *USPMessageHandler) createTransportErrorResponse14() []byte {
	// Create minimal error record indicating USP service unavailable
	errorRecord := &v1_4.Record{
		Version: "1.4",
		ToId:    "proto://unknown-endpoint",
		FromId:  "proto://openusp-mtp-service",
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: []byte("USP service temporarily unavailable"),
			},
		},
	}

	data, err := proto.Marshal(errorRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.4 transport error: %v", err)
		return []byte("Transport error - USP service unavailable")
	}
	return data
}

// --- MTP gRPC Service Implementation ---

// SendMessageToAgent implements the gRPC method to send messages to agents
func (h *USPMessageHandler) SendMessageToAgent(ctx context.Context, req *mtpservice.SendMessageRequest) (*mtpservice.SendMessageResponse, error) {
	if req.AgentId == "" {
		return &mtpservice.SendMessageResponse{
			Success:      false,
			ErrorMessage: "Agent ID is required",
		}, nil
	}

	h.agentsMutex.RLock()
	agent, exists := h.connectedAgents[req.AgentId]
	h.agentsMutex.RUnlock()

	if !exists {
		return &mtpservice.SendMessageResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Agent %s is not connected", req.AgentId),
		}, nil
	}

	log.Printf("🚀 Sending USP message to agent %s via %s transport (size: %d bytes)",
		req.AgentId, agent.TransportType, len(req.UspMessage))

	// Send message based on transport type
	switch agent.TransportType {
	case "WebSocket":
		if agent.WebSocketConn == nil {
			return &mtpservice.SendMessageResponse{
				Success:      false,
				ErrorMessage: "WebSocket connection is nil",
			}, nil
		}

		if err := agent.WebSocketConn.WriteMessage(websocket.BinaryMessage, req.UspMessage); err != nil {
			log.Printf("❌ Failed to send message to agent %s: %v", req.AgentId, err)
			return &mtpservice.SendMessageResponse{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to send message: %v", err),
			}, nil
		}

		log.Printf("✅ Successfully sent message to agent %s via WebSocket", req.AgentId)
		return &mtpservice.SendMessageResponse{
			Success: true,
		}, nil

	case "STOMP":
		if h.stompBroker == nil {
			return &mtpservice.SendMessageResponse{
				Success:      false,
				ErrorMessage: "STOMP broker not available",
			}, nil
		}

		// Send via STOMP broker to /queue/usp.agent
		destination := "/queue/usp.agent"
		if err := h.stompBroker.Send(destination, req.UspMessage); err != nil {
			log.Printf("❌ Failed to send message to agent %s via STOMP: %v", req.AgentId, err)
			return &mtpservice.SendMessageResponse{
				Success:      false,
				ErrorMessage: fmt.Sprintf("Failed to send via STOMP: %v", err),
			}, nil
		}

		log.Printf("✅ Successfully sent message to agent %s via STOMP (/queue/usp.agent)", req.AgentId)
		return &mtpservice.SendMessageResponse{
			Success: true,
		}, nil

	default:
		return &mtpservice.SendMessageResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Transport type %s not yet implemented", agent.TransportType),
		}, nil
	}
}

// GetConnectedAgents implements the gRPC method to list connected agents
func (h *USPMessageHandler) GetConnectedAgents(ctx context.Context, req *mtpservice.GetConnectedAgentsRequest) (*mtpservice.GetConnectedAgentsResponse, error) {
	h.agentsMutex.RLock()
	defer h.agentsMutex.RUnlock()

	var agents []*mtpservice.ConnectedAgent
	for _, agent := range h.connectedAgents {
		agents = append(agents, &mtpservice.ConnectedAgent{
			AgentId:       agent.AgentID,
			ClientId:      agent.ClientID,
			TransportType: agent.TransportType,
			ConnectedAt:   agent.ConnectedAt.Unix(),
		})
	}

	return &mtpservice.GetConnectedAgentsResponse{
		Agents: agents,
	}, nil
}

// GetServiceHealth implements the gRPC method for health checks
func (h *USPMessageHandler) GetServiceHealth(ctx context.Context, req *mtpservice.HealthRequest) (*mtpservice.HealthResponse, error) {
	h.agentsMutex.RLock()
	agentCount := len(h.connectedAgents)
	h.agentsMutex.RUnlock()

	return &mtpservice.HealthResponse{
		Status:    "healthy",
		Message:   fmt.Sprintf("mtp-service running with %d connected agents", agentCount),
		Timestamp: time.Now().Unix(),
	}, nil
}

// notifyProactiveOnboarding implements TR-369 proactive onboarding notification to USP service
func (h *USPMessageHandler) notifyProactiveOnboarding(endpointID, uspVersion string, connectionContext map[string]interface{}) error {
	log.Printf("📡 ProactiveOnboarding: Notifying USP service of new connection: %s (USP %s)", endpointID, uspVersion)

	// Get USP service client via connection manager
	uspClient, err := h.getUSPServiceClient()
	if err != nil {
		return fmt.Errorf("failed to get USP service client: %w", err)
	}

	// Create proactive onboarding request per TR-369 specification
	request := &uspservice.ProactiveOnboardingRequest{
		EndpointId: endpointID,
		UspVersion: uspVersion,
		ConnectionContext: map[string]string{
			"transport_type": fmt.Sprintf("%v", connectionContext["transport_type"]),
			"client_id":      fmt.Sprintf("%v", connectionContext["client_id"]),
			"subprotocol":    fmt.Sprintf("%v", connectionContext["subprotocol"]),
			"initiated_by":   fmt.Sprintf("%v", connectionContext["initiated_by"]),
			"reason":         fmt.Sprintf("%v", connectionContext["reason"]),
			"mtp_version":    fmt.Sprintf("%v", connectionContext["mtp_version"]),
		},
	}

	// Call USP service for proactive onboarding per TR-369 Section 3.3.2
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := uspClient.HandleProactiveOnboarding(ctx, request)
	if err != nil {
		return fmt.Errorf("USP service proactive onboarding failed: %w", err)
	}

	log.Printf("✅ ProactiveOnboarding: USP service response: %s", response.Message)
	return nil
}
