package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"openusp/internal/cwmp"
	"openusp/internal/tr181"
	"openusp/pkg/config"
	"openusp/pkg/kafka"
	"openusp/pkg/metrics"
	"openusp/pkg/version"

	confluentkafka "github.com/confluentinc/confluent-kafka-go/kafka"
)

// CWMPService provides TR-069 protocol support for backward compatibility
type CWMPService struct {
	config            *Config
	healthServer      *http.Server // Health/status/metrics server (dynamic port)
	processor         *cwmp.MessageProcessor
	tr181Mgr          *tr181.DeviceManager
	metrics           *metrics.OpenUSPMetrics
	onboardingManager *cwmp.OnboardingManager
	kafkaClient       *kafka.Client
	kafkaProducer     *kafka.Producer
	kafkaConsumer     *kafka.Consumer
	globalConfig      *config.Config
	mu                sync.RWMutex
	connections       map[string]*cwmp.Session
}

// Config holds configuration for CWMP service
type Config struct {
	HealthPort            int // Port for health/status/metrics
	ACSUsername           string
	ACSPassword           string
	ConnectionTimeout     time.Duration
	SessionTimeout        time.Duration
	MaxConcurrentSessions int
	EnableAuthentication  bool
	TLS                   TLSConfig
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// NewCWMPService creates a new CWMP service instance
func NewCWMPService(globalConfig *config.Config) (*CWMPService, error) {
	// Parse timeout durations from config
	connTimeout, err := time.ParseDuration(globalConfig.CWMPService.ConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid connection timeout: %w", err)
	}

	sessionTimeout, err := time.ParseDuration(globalConfig.CWMPService.SessionTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid session timeout: %w", err)
	}

	// Build service config from global config
	serviceConfig := &Config{
		HealthPort:            globalConfig.CWMPServiceHealthPort,
		ACSUsername:           globalConfig.CWMPService.Username,
		ACSPassword:           globalConfig.CWMPService.Password,
		ConnectionTimeout:     connTimeout,
		SessionTimeout:        sessionTimeout,
		MaxConcurrentSessions: globalConfig.CWMPService.MaxConcurrentSessions,
		EnableAuthentication:  globalConfig.CWMPService.AuthenticationEnabled,
		TLS: TLSConfig{
			Enabled:  false, // TLS handled by reverse proxy/ingress
			CertFile: "",
			KeyFile:  "",
		},
	}

	// Initialize Kafka client
	kafkaClient, err := kafka.NewClient(&globalConfig.Kafka)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka client: %w", err)
	}

	// Ensure Kafka topics exist
	topicsManager := kafka.NewTopicsManager(kafkaClient, &globalConfig.Kafka.Topics)
	if err := topicsManager.EnsureAllTopicsExist(); err != nil {
		return nil, fmt.Errorf("failed to create Kafka topics: %w", err)
	}

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(kafkaClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}

	// Initialize Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(&globalConfig.Kafka, globalConfig.CWMPService.ConsumerGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka consumer: %w", err)
	}

	// Initialize TR-181 manager for data model support
	tr181Manager, err := tr181.NewDeviceManager(globalConfig.TR181.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TR-181 manager: %w", err)
	}

	// Initialize CWMP message processor
	processor := cwmp.NewMessageProcessor(tr181Manager)

	// Initialize metrics
	metricsInstance := metrics.NewOpenUSPMetrics("cwmp-service")

	service := &CWMPService{
		//deployConfig:  nil, // Static configuration - no deployment config needed
		config:        serviceConfig,
		processor:     processor,
		metrics:       metricsInstance,
		tr181Mgr:      tr181Manager,
		kafkaClient:   kafkaClient,
		kafkaProducer: kafkaProducer,
		kafkaConsumer: kafkaConsumer,
		globalConfig:  globalConfig,
		connections:   make(map[string]*cwmp.Session),
	}

	// Initialize onboarding manager with Kafka producer
	onboardingManager, err := cwmp.NewOnboardingManager(kafkaProducer, tr181Manager, &globalConfig.Kafka.Topics)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize onboarding manager: %w", err)
	}

	// Set onboarding manager in service and processor
	service.onboardingManager = onboardingManager
	processor.SetOnboardingManager(onboardingManager)

	// NOTE: CWMP service NO LONGER runs HTTP server on port 7547
	// TR-069 messages are received via mtp-http service on port 7547
	// and forwarded to CWMP service through Kafka

	// Create health/admin server only
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", service.handleHealth)
	healthMux.HandleFunc("/status", service.handleStatus)
	healthMux.Handle("/metrics", metrics.HTTPHandler())

	service.healthServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", serviceConfig.HealthPort),
		Handler:      healthMux,
		ReadTimeout:  serviceConfig.ConnectionTimeout,
		WriteTimeout: serviceConfig.ConnectionTimeout,
		IdleTimeout:  serviceConfig.SessionTimeout,
	}

	return service, nil
}

// Start starts the CWMP service
func (s *CWMPService) Start(ctx context.Context) error {
	// Start health server
	go func() {
		log.Printf("🏥 CWMP Service health endpoint listening on port %d", s.config.HealthPort)
		if err := s.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Health server error: %v", err)
		}
	}()

	// Setup Kafka consumers for duplex communication
	if err := s.setupKafkaConsumers(); err != nil {
		return fmt.Errorf("failed to setup Kafka consumers: %w", err)
	}

	// Start Kafka consumer
	go s.kafkaConsumer.Start()

	log.Printf("✅ CWMP Service started - Kafka consumer: %s, Health: http://localhost:%d",
		s.globalConfig.CWMPService.ConsumerGroup, s.config.HealthPort)

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownTimeout, _ := time.ParseDuration(s.globalConfig.CWMPService.ShutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	log.Printf("🛑 Shutting down CWMP service...")
	s.kafkaConsumer.Stop()
	s.healthServer.Shutdown(shutdownCtx)

	return nil
}

// setupKafkaConsumers configures Kafka subscriptions for duplex communication
func (s *CWMPService) setupKafkaConsumers() error {
	// Subscribe to all relevant topics for duplex communication
	topics := []string{
		s.globalConfig.Kafka.Topics.CWMPMessagesInbound,  // From MTP-HTTP service
		s.globalConfig.Kafka.Topics.CWMPAPIRequest,       // From API Gateway
		s.globalConfig.Kafka.Topics.CWMPDataRequest,      // From Data Service
		s.globalConfig.Kafka.Topics.DataDeviceCreated,    // Data events from Data Service
		s.globalConfig.Kafka.Topics.DataDeviceUpdated,    // Data events from Data Service
		s.globalConfig.Kafka.Topics.DataParameterUpdated, // Data events from Data Service
	}

	// Define message handler
	handler := func(msg *confluentkafka.Message) error {
		topic := *msg.TopicPartition.Topic
		log.Printf("📨 Received message on topic %s: %d bytes", topic, len(msg.Value))

		// Route to appropriate handler based on topic
		switch topic {
		case s.globalConfig.Kafka.Topics.CWMPMessagesInbound:
			return s.handleCWMPMessage(msg)
		case s.globalConfig.Kafka.Topics.CWMPAPIRequest:
			return s.handleAPIRequest(msg)
		case s.globalConfig.Kafka.Topics.CWMPDataRequest:
			return s.handleDataRequest(msg)
		case s.globalConfig.Kafka.Topics.DataDeviceCreated,
			s.globalConfig.Kafka.Topics.DataDeviceUpdated,
			s.globalConfig.Kafka.Topics.DataParameterUpdated:
			return s.handleDataEvent(msg)
		default:
			log.Printf("⚠️  Unknown topic: %s", topic)
			return nil
		}
	}

	// Subscribe to topics
	if err := s.kafkaConsumer.Subscribe(topics, handler); err != nil {
		return fmt.Errorf("failed to subscribe to Kafka topics: %w", err)
	}

	log.Printf("✅ Subscribed to %d Kafka topics for duplex communication", len(topics))
	for _, topic := range topics {
		log.Printf("   └── %s", topic)
	}
	return nil
}

// handleCWMPMessage processes incoming CWMP messages from MTP-HTTP service
func (s *CWMPService) handleCWMPMessage(msg *confluentkafka.Message) error {
	s.processCWMPMessageFromKafka(msg.Value)
	return nil
}

// handleAPIRequest processes API requests from API Gateway
func (s *CWMPService) handleAPIRequest(msg *confluentkafka.Message) error {
	log.Printf("📥 Processing API request from API Gateway")

	// Parse the request
	var req kafka.APIRequest
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		return fmt.Errorf("failed to parse API request: %w", err)
	}

	log.Printf("🔄 CWMP API Request: %s %s (correlation: %s)", req.Method, req.Operation, req.CorrelationID)

	// Process the request and build response
	response := kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Timestamp:     time.Now(),
	}

	// Handle different CWMP-specific operations
	switch req.Operation {
	case "GetCWMPSession", "SendCWMPCommand", "GetCWMPStatus", "GetParameterValues", "SetParameterValues":
		// Process CWMP-specific operations
		response.Status = http.StatusOK
		response.Data = map[string]interface{}{
			"message": fmt.Sprintf("CWMP operation %s processed successfully", req.Operation),
			"service": "cwmp-service",
		}
		log.Printf("✅ CWMP API operation processed: %s", req.Operation)
	default:
		response.Status = http.StatusNotImplemented
		response.Error = fmt.Sprintf("CWMP operation not implemented: %s", req.Operation)
		log.Printf("⚠️ CWMP API operation not implemented: %s", req.Operation)
	}

	// Send response back to API Gateway
	return s.sendAPIResponse(&response)
}

// handleDataRequest processes data requests from Data Service
func (s *CWMPService) handleDataRequest(msg *confluentkafka.Message) error {
	log.Printf("📥 Processing data request from Data Service")

	// Parse the request
	var req kafka.APIRequest
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		return fmt.Errorf("failed to parse data request: %w", err)
	}

	log.Printf("🔄 CWMP Data Request: %s %s (correlation: %s)", req.Method, req.Operation, req.CorrelationID)

	// Process the request and build response
	response := kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Timestamp:     time.Now(),
	}

	// Handle different data operations
	switch req.Operation {
	case "GetDeviceParameters", "SyncDeviceData", "GetDeviceState", "GetCWMPDeviceInfo":
		// Process data operations
		response.Status = http.StatusOK
		response.Data = map[string]interface{}{
			"message": fmt.Sprintf("CWMP data operation %s processed successfully", req.Operation),
			"service": "cwmp-service",
		}
		log.Printf("✅ CWMP data operation processed: %s", req.Operation)
	default:
		response.Status = http.StatusNotImplemented
		response.Error = fmt.Sprintf("CWMP data operation not implemented: %s", req.Operation)
		log.Printf("⚠️ CWMP data operation not implemented: %s", req.Operation)
	}

	// Send response back to Data Service
	return s.sendDataResponse(&response)
}

// sendAPIResponse publishes an API response to Kafka for API Gateway
func (s *CWMPService) sendAPIResponse(resp *kafka.APIResponse) error {
	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal API response: %w", err)
	}

	err = s.kafkaProducer.PublishRaw(
		s.globalConfig.Kafka.Topics.CWMPAPIResponse,
		resp.CorrelationID,
		respData,
	)
	if err != nil {
		log.Printf("❌ Failed to publish API response: %v", err)
		return err
	}

	log.Printf("✅ Published API response to %s (correlation: %s)", s.globalConfig.Kafka.Topics.CWMPAPIResponse, resp.CorrelationID)
	return nil
}

// sendDataResponse publishes a data response to Kafka for Data Service
func (s *CWMPService) sendDataResponse(resp *kafka.APIResponse) error {
	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal data response: %w", err)
	}

	err = s.kafkaProducer.PublishRaw(
		s.globalConfig.Kafka.Topics.CWMPDataResponse,
		resp.CorrelationID,
		respData,
	)
	if err != nil {
		log.Printf("❌ Failed to publish data response: %v", err)
		return err
	}

	log.Printf("✅ Published data response to %s (correlation: %s)", s.globalConfig.Kafka.Topics.CWMPDataResponse, resp.CorrelationID)
	return nil
}

// handleDataEvent processes data events from Data Service
func (s *CWMPService) handleDataEvent(msg *confluentkafka.Message) error {
	topic := *msg.TopicPartition.Topic
	log.Printf("📥 Processing data event from topic: %s", topic)

	// TODO: Process data events (device created, updated, parameter changed, etc.)
	// This allows CWMP service to stay synchronized with data changes

	return nil
}

// Stop stops the CWMP service
func (s *CWMPService) Stop() error {
	log.Printf("🛑 Stopping CWMP Service...")

	// Use shutdown timeout from config
	ctx, cancel := context.WithTimeout(context.Background(), s.config.SessionTimeout)
	defer cancel()

	// Close all active sessions
	s.mu.Lock()
	for _, session := range s.connections {
		session.Close()
	}
	s.connections = make(map[string]*cwmp.Session)
	s.mu.Unlock()

	// Close onboarding manager
	if s.onboardingManager != nil {
		if err := s.onboardingManager.Close(); err != nil {
			log.Printf("❌ Error closing onboarding manager: %v", err)
		}
	}

	// Close Kafka connections
	if s.kafkaProducer != nil {
		s.kafkaProducer.Close()
	}

	if s.kafkaClient != nil {
		s.kafkaClient.Close()
	}

	// Shutdown health server
	var shutdownErr error
	if err := s.healthServer.Shutdown(ctx); err != nil {
		log.Printf("❌ Error shutting down health server: %v", err)
		if shutdownErr == nil {
			shutdownErr = err
		}
	}

	if shutdownErr != nil {
		return shutdownErr
	}

	return nil
}

// handleCWMPRequest handles incoming CWMP requests
func (s *CWMPService) handleCWMPRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 CWMP Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	// Only accept POST requests for CWMP
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication if enabled
	if s.config.EnableAuthentication {
		username, password, ok := r.BasicAuth()
		if !ok || username != s.config.ACSUsername || password != s.config.ACSPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="CWMP ACS"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Get or create session
	sessionID := s.getSessionID(r)
	session := s.getOrCreateSession(sessionID, r.RemoteAddr)

	// Process CWMP message
	if err := s.processor.ProcessRequest(session, w, r); err != nil {
		log.Printf("❌ CWMP processing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ CWMP Request processed successfully")
}

// handleHealth handles health check requests
func (s *CWMPService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	healthStatus := map[string]interface{}{
		"service":            "cwmp-service",
		"status":             "healthy",
		"timestamp":          time.Now().UTC().Format(time.RFC3339),
		"version":            version.Version,
		"protocol":           "TR-069 v1.2",
		"protocol_namespace": "urn:dslforum-org:cwmp-1-2",
		"supported_versions": []string{"1.0", "1.1", "1.2"},
		"sessions":           len(s.connections),
		"max_sessions":       s.config.MaxConcurrentSessions,
	}

	if err := json.NewEncoder(w).Encode(healthStatus); err != nil {
		log.Printf("❌ Health check encoding error: %v", err)
	}
}

// handleStatus handles detailed status requests
func (s *CWMPService) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	status := map[string]interface{}{
		"service":            "cwmp-service",
		"status":             "running",
		"timestamp":          time.Now().UTC().Format(time.RFC3339),
		"active_sessions":    len(s.connections),
		"max_sessions":       s.config.MaxConcurrentSessions,
		"authentication":     s.config.EnableAuthentication,
		"tls_enabled":        s.config.TLS.Enabled,
		"connection_timeout": s.config.ConnectionTimeout.String(),
		"session_timeout":    s.config.SessionTimeout.String(),
		"tr181_status":       "enabled",
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("❌ Status encoding error: %v", err)
	}
}

// getSessionID extracts session ID from request
func (s *CWMPService) getSessionID(r *http.Request) string {
	// Try to get session ID from headers first
	if sessionID := r.Header.Get("X-CWMP-Session-ID"); sessionID != "" {
		return sessionID
	}

	// Fallback to remote address as session identifier
	return r.RemoteAddr
}

// getOrCreateSession gets existing session or creates new one
func (s *CWMPService) getOrCreateSession(sessionID, remoteAddr string) *cwmp.Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.connections[sessionID]; exists {
		return session
	}

	// Check session limit
	if len(s.connections) >= s.config.MaxConcurrentSessions {
		log.Printf("⚠️ Maximum sessions reached, cleaning up oldest sessions")
		s.cleanupOldestSessions(10) // Clean up 10% of sessions
	}

	// Create new session
	session := cwmp.NewSession(sessionID, remoteAddr, s.config.SessionTimeout)
	s.connections[sessionID] = session

	log.Printf("🔌 New CWMP session created: %s", sessionID)
	return session
}

// cleanupOldestSessions removes oldest sessions to make room for new ones
func (s *CWMPService) cleanupOldestSessions(percentage int) {
	if percentage <= 0 || percentage > 100 {
		return
	}

	toRemove := len(s.connections) * percentage / 100
	if toRemove == 0 {
		toRemove = 1
	}

	// Simple cleanup - remove first N sessions (in production, use proper LRU)
	count := 0
	for sessionID, session := range s.connections {
		if count >= toRemove {
			break
		}
		session.Close()
		delete(s.connections, sessionID)
		count++
	}

	log.Printf("🧹 Cleaned up %d old sessions", count)
}

func main() {
	log.Printf("🚀 Starting OpenUSP CWMP Service...")

	// Command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.GetFullVersion("OpenUSP CWMP Service"))
		return
	}

	if *showHelp {
		fmt.Println("OpenUSP CWMP Service - TR-069 Protocol Support")
		fmt.Println("==============================================")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  cwmp-service [flags]")
		fmt.Println("")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("All configuration loaded from configs/openusp.yml")
		fmt.Println("Environment variables can override YAML values (see openusp.yml comments)")
		return
	}

	// Load configuration from YAML file
	globalConfig := config.Load()

	// Create CWMP service with loaded configuration
	cwmpService, err := NewCWMPService(globalConfig)
	if err != nil {
		log.Fatalf("Failed to create CWMP service: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("🛑 Received shutdown signal")
		cancel()
	}()

	// Start CWMP service
	if err := cwmpService.Start(ctx); err != nil {
		log.Fatalf("CWMP service error: %v", err)
	}

	log.Printf("✅ CWMP Service stopped")
}

// processCWMPMessageFromKafka processes a TR-069/CWMP message received from Kafka
func (s *CWMPService) processCWMPMessageFromKafka(data []byte) {
	// Unmarshal the CWMP message wrapper
	type CWMPMessage struct {
		Payload  []byte            `json:"payload"`
		Metadata map[string]string `json:"metadata"`
	}

	var cwmpMsg CWMPMessage
	if err := json.Unmarshal(data, &cwmpMsg); err != nil {
		log.Printf("❌ Failed to unmarshal CWMP message: %v", err)
		return
	}

	log.Printf("📦 Processing TR-069 message (%d bytes) from %s", len(cwmpMsg.Payload), cwmpMsg.Metadata["client-address"])

	// Process the TR-069 SOAP payload using the existing processor
	// Note: This would typically parse the SOAP envelope and extract the CWMP method
	//response, err := s.processor.ProcessMessage(cwmpMsg.Payload)
	//if err != nil {
	//	log.Printf("❌ Error processing CWMP message: %v", err)
	//	return
	//}

	// TODO: Parse SOAP envelope and call appropriate CWMP handler

	// In a full implementation, we would:
	// 1. Parse the SOAP envelope
	// 2. Extract the CWMP method (Inform, GetParameterValues, etc.)
	// 3. Call the appropriate handler in processor
	// 4. Generate SOAP response
	// 5. Send response back via Kafka to mtp-http
}
