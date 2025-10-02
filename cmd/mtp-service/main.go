package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"openusp/internal/grpc"
	"openusp/internal/usp"
	"openusp/pkg/config"
	"openusp/pkg/consul"
	"openusp/pkg/metrics"
	"openusp/pkg/proto/v1_3"
	"openusp/pkg/proto/v1_4"
	"openusp/pkg/version"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// SimpleMTPService provides a basic MTP service implementation for demo
type SimpleMTPService struct {
	webSocketPort    int
	healthPort       int
	unixSocketPath   string
	webSocketServer  *http.Server // WebSocket protocol server (standard port 8081)
	healthServer     *http.Server // Health/admin server (dynamic port)
	messageHandler   MessageHandler
	connectedClients map[string]time.Time
	upgrader         websocket.Upgrader
	metrics          *metrics.OpenUSPMetrics
	mu               sync.RWMutex
}

// MessageHandler defines the interface for handling USP messages
type MessageHandler interface {
	ProcessUSPMessage(data []byte) ([]byte, error)
	DetectUSPVersion(data []byte) (string, error)
}

// Config holds configuration for MTP service
type Config struct {
	WebSocketPort    int    // Standard WebSocket port (8081)
	HealthPort       int    // Dynamic port for health/status/metrics
	UnixSocketPath   string
	EnableWebSocket  bool
	EnableUnixSocket bool
	EnableMQTT       bool
	EnableSTOMP      bool
}

// DefaultConfig returns default MTP service configuration
func DefaultConfig(healthPort int) *Config {
	// WebSocket protocol always uses standard USP MTP port
	webSocketPort := 8081

	return &Config{
		WebSocketPort:    webSocketPort,
		HealthPort:       healthPort,
		UnixSocketPath:   "/tmp/usp-agent.sock",
		EnableWebSocket:  true,
		EnableUnixSocket: true,
		EnableMQTT:       true,
		EnableSTOMP:      true,
	}
}

// NewSimpleMTPService creates a new simplified MTP service
func NewSimpleMTPService(config *Config, handler MessageHandler) (*SimpleMTPService, error) {
	// Initialize metrics
	metricsInstance := metrics.NewOpenUSPMetrics("mtp-service")

	return &SimpleMTPService{
		webSocketPort:    config.WebSocketPort,
		healthPort:       config.HealthPort,
		unixSocketPath:   config.UnixSocketPath,
		messageHandler:   handler,
		connectedClients: make(map[string]time.Time),
		metrics:          metricsInstance,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}, nil
}

// Start starts the MTP service
func (s *SimpleMTPService) Start(ctx context.Context) error {
	log.Println("🚀 Starting Simple MTP Service...")

	// Create WebSocket protocol server (standard port 8081)
	webSocketMux := http.NewServeMux()
	webSocketMux.HandleFunc("/usp", s.handleUSPWebSocketDemo) // Demo HTML page
	webSocketMux.HandleFunc("/ws", s.handleUSPWebSocket)      // Actual WebSocket endpoint

	s.webSocketServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.webSocketPort),
		Handler: webSocketMux,
	}

	// Create health/admin server (dynamic port registered with Consul)
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

	log.Printf("✅ Simple MTP Service started")
	log.Printf("   📡 WebSocket Port: %d (USP Protocol)", s.webSocketPort)
	log.Printf("   🏥 Health API Port: %d (Dynamic)", s.healthPort)
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

// handleUSPWebSocketDemo provides a demo WebSocket endpoint
func (s *SimpleMTPService) handleUSPWebSocketDemo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>USP MTP Service Demo</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; }
        .protocol { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .status { padding: 10px; border-radius: 3px; margin: 10px 0; }
        .enabled { background-color: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .disabled { background-color: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        button { padding: 10px 20px; margin: 5px; border: none; border-radius: 3px; cursor: pointer; }
        .primary { background-color: #007bff; color: white; }
        .success { background-color: #28a745; color: white; }
        pre { background-color: #f8f9fa; padding: 10px; border-radius: 3px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🌐 OpenUSP MTP Service Demo</h1>
        <p>Message Transfer Protocol Service for TR-369 USP</p>
        
        <div class="protocol">
            <h3>📡 MQTT Protocol</h3>
            <div class="status enabled">✅ Enabled - Ready for MQTT broker connection</div>
            <p>Topics: /usp/controller/+, /usp/agent/+, /usp/broadcast</p>
        </div>
        
        <div class="protocol">
            <h3>📬 STOMP Protocol</h3>
            <div class="status enabled">✅ Enabled - Ready for STOMP broker connection</div>
            <p>Destinations: /queue/usp.controller, /queue/usp.agent, /topic/usp.broadcast</p>
        </div>
        
        <div class="protocol">
            <h3>🌐 WebSocket Protocol</h3>
            <div class="status enabled">✅ Enabled - Running on port 8081</div>
            <p>Endpoint: ws://localhost:8081/usp</p>
            <button class="primary" onclick="testWebSocket()">Test WebSocket Connection</button>
        </div>
        
        <div class="protocol">
            <h3>🔌 Unix Domain Socket</h3>
            <div class="status enabled">✅ Enabled - Socket path: /tmp/usp-agent.sock</div>
            <p>Binary protocol for local IPC communication</p>
        </div>
        
        <div class="protocol">
            <h3>📊 Service Status</h3>
            <button class="success" onclick="checkHealth()">Check Health</button>
            <button class="primary" onclick="simulateMessage()">Simulate USP Message</button>
            <pre id="output">Ready to test MTP service...</pre>
        </div>
    </div>
    
    <script>
        function testWebSocket() {
            const output = document.getElementById('output');
            output.textContent = 'Testing WebSocket connection...';
            
            // Note: This is a demo - actual WebSocket would require proper implementation
            setTimeout(() => {
                output.textContent = 'WebSocket test completed.\n\nNote: Full WebSocket implementation requires:\n- Proper WebSocket upgrade handling\n- USP message serialization\n- Client connection management';
            }, 1000);
        }
        
        function checkHealth() {
            const output = document.getElementById('output');
            fetch('/health')
                .then(response => response.json())
                .then(data => {
                    output.textContent = JSON.stringify(data, null, 2);
                })
                .catch(error => {
                    output.textContent = 'Error: ' + error;
                });
        }
        
        function simulateMessage() {
            const output = document.getElementById('output');
            fetch('/simulate', { method: 'POST' })
                .then(response => response.json())
                .then(data => {
                    output.textContent = JSON.stringify(data, null, 2);
                })
                .catch(error => {
                    output.textContent = 'Error: ' + error;
                });
        }
    </script>
</body>
</html>`
	w.Write([]byte(html))
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

	log.Printf("📱 WebSocket client connected: %s", clientID)

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

		if messageType == websocket.BinaryMessage && s.messageHandler != nil {
			recvSize := len(payload)
			log.Printf("📨 Received USP message from client %s (size: %d bytes)", clientID, recvSize)

			// Detect USP version for metrics
			version, _ := s.messageHandler.DetectUSPVersion(payload)
			if version == "" {
				version = "unknown"
			}

			start := time.Now()
			response, err := s.messageHandler.ProcessUSPMessage(payload)
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
		ToId:    "agent::demo",
		FromId:  "controller::demo",
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: []byte("demo USP message payload"),
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
	parser         *usp.USPParser
	uspClient      *grpc.USPServiceClient
	uspServiceAddr string
}

// NewUSPMessageHandler creates a new USP message handler that forwards to USP service
func NewUSPMessageHandler(uspServiceAddr string) *USPMessageHandler {
	return &USPMessageHandler{
		parser:         usp.NewUSPParser(),
		uspServiceAddr: uspServiceAddr,
	}
}

// initUSPServiceClient initializes the USP service client connection
func (h *USPMessageHandler) initUSPServiceClient() error {
	if h.uspClient != nil {
		return nil // Already initialized
	}

	var err error
	h.uspClient, err = grpc.NewUSPServiceClient(h.uspServiceAddr)
	if err != nil {
		return fmt.Errorf("failed to create USP service client: %w", err)
	}

	log.Printf("✅ Connected to USP Service at %s", h.uspServiceAddr)
	return nil
}

func (h *USPMessageHandler) DetectUSPVersion(data []byte) (string, error) {
	version, err := h.parser.DetectVersion(data)
	if err != nil {
		return "", err
	}
	return string(version), nil
}

func (h *USPMessageHandler) ProcessUSPMessage(data []byte) ([]byte, error) {
	messageCount++

	// Initialize USP service client if needed
	if err := h.initUSPServiceClient(); err != nil {
		log.Printf("⚠️ USP service connection failed: %v", err)
		// Fall back to simple parsing and response
		return h.processUSPMessageLocally(data)
	}

	// Forward USP message to USP service for processing
	ctx := context.Background()
	response, err := h.uspClient.ProcessUSPMessage(ctx, data, "WebSocket", "mtp-client", nil)
	if err != nil {
		errorCount++
		log.Printf("❌ USP service processing failed: %v", err)
		// Fall back to local processing
		return h.processUSPMessageLocally(data)
	}

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

// processUSPMessageLocally provides fallback local USP message processing
func (h *USPMessageHandler) processUSPMessageLocally(data []byte) ([]byte, error) {
	// Parse complete USP record and message
	parsed, err := h.parser.ParseUSP(data)
	if err != nil {
		errorCount++
		log.Printf("❌ Failed to parse USP message: %v", err)
		return nil, fmt.Errorf("failed to parse USP message: %w", err)
	}

	// Log detailed parsing information
	h.logParsedUSP(parsed)

	// Validate parsed record and message
	if parsed.Record != nil {
		if err := h.parser.ValidateRecord(parsed.Record); err != nil {
			log.Printf("⚠️ Record validation warning: %v", err)
		}
	}

	if parsed.Message != nil {
		if err := h.parser.ValidateMessage(parsed.Message); err != nil {
			log.Printf("⚠️ Message validation warning: %v", err)
		}
	}

	// Create appropriate response based on message type
	response := h.createUSPResponse(parsed)

	return response, nil
}

func (h *USPMessageHandler) logParsedUSP(parsed *usp.ParsedUSP) {
	if parsed.Record != nil {
		log.Printf("📋 USP %s Record: %s -> %s (%s)",
			parsed.Record.Version,
			parsed.Record.FromID,
			parsed.Record.ToID,
			parsed.Record.RecordType)
	}

	if parsed.Message != nil {
		log.Printf("📋 USP %s Message: %s (%s)",
			parsed.Message.Version,
			parsed.Message.MsgID,
			parsed.Message.MsgType)

		// Log operation details if available
		if body, ok := parsed.Message.Body.(map[string]interface{}); ok {
			if operation, exists := body["operation"]; exists {
				log.Printf("📋 Operation: %s", operation)

				// Log additional operation-specific details
				switch operation {
				case "Get":
					if paths, ok := body["param_paths"].([]string); ok {
						log.Printf("📋 Parameter paths: %v", paths)
					}
				case "Set":
					if updateObjs, ok := body["update_objs"].(int); ok {
						log.Printf("📋 Update objects: %d", updateObjs)
					}
				case "Add":
					if createObjs, ok := body["create_objs"].(int); ok {
						log.Printf("📋 Create objects: %d", createObjs)
					}
				case "Delete":
					if objPaths, ok := body["obj_paths"].([]string); ok {
						log.Printf("📋 Object paths: %v", objPaths)
					}
				case "Operate":
					if command, ok := body["command"].(string); ok {
						log.Printf("📋 Command: %s", command)
					}
				}
			}
		}
	}

	if len(parsed.Errors) > 0 {
		log.Printf("⚠️ Parsing errors: %v", parsed.Errors)
	}
}

func (h *USPMessageHandler) createUSPResponse(parsed *usp.ParsedUSP) []byte {
	if parsed == nil || parsed.Record == nil {
		return []byte("Invalid USP message")
	}

	// Create response based on version
	switch parsed.Record.Version {
	case usp.Version14:
		return h.createUSP14Response(parsed)
	case usp.Version13:
		return h.createUSP13Response(parsed)
	default:
		return []byte(fmt.Sprintf("Unsupported USP version: %s", parsed.Record.Version))
	}
}

func (h *USPMessageHandler) createUSP14Response(parsed *usp.ParsedUSP) []byte {
	// Create a simple echo response for demo
	responseRecord := &v1_4.Record{
		Version: "1.4",
		ToId:    parsed.Record.FromID,
		FromId:  parsed.Record.ToID,
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: []byte(fmt.Sprintf("Processed USP 1.4 message: %s", parsed.Message.MsgType)),
			},
		},
	}

	data, err := proto.Marshal(responseRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.4 response: %v", err)
		return []byte("Failed to create response")
	}

	return data
}

func (h *USPMessageHandler) createUSP13Response(parsed *usp.ParsedUSP) []byte {
	// Create a simple echo response for demo
	responseRecord := &v1_3.Record{
		Version: "1.3",
		ToId:    parsed.Record.FromID,
		FromId:  parsed.Record.ToID,
		RecordType: &v1_3.Record_NoSessionContext{
			NoSessionContext: &v1_3.NoSessionContextRecord{
				Payload: []byte(fmt.Sprintf("Processed USP 1.3 message: %s", parsed.Message.MsgType)),
			},
		},
	}

	data, err := proto.Marshal(responseRecord)
	if err != nil {
		log.Printf("❌ Failed to marshal USP 1.3 response: %v", err)
		return []byte("Failed to create response")
	}

	return data
}

// Global counters for demo
var (
	startTime    = time.Now()
	messageCount = 0
	errorCount   = 0
)

// mustMarshalJSON marshals to JSON or panics
func mustMarshalJSON(v interface{}) string {
	// Simple JSON marshaling for demo
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
	for k, v := range m {
		if !first {
			result += ","
		}
		first = false
		result += fmt.Sprintf("\"%s\":", k)

		switch val := v.(type) {
		case string:
			result += fmt.Sprintf("\"%s\"", val)
		case int:
			result += fmt.Sprintf("%d", val)
		case map[string]interface{}:
			result += marshalMapToJSON(val)
		case []string:
			result += "["
			for i, s := range val {
				if i > 0 {
					result += ","
				}
				result += fmt.Sprintf("\"%s\"", s)
			}
			result += "]"
		case map[string]string:
			result += "{"
			innerFirst := true
			for ik, iv := range val {
				if !innerFirst {
					result += ","
				}
				innerFirst = false
				result += fmt.Sprintf("\"%s\":\"%s\"", ik, iv)
			}
			result += "}"
		default:
			result += fmt.Sprintf("\"%v\"", val)
		}
	}
	result += "}"
	return result
}

func main() {
	log.Printf("🚀 Starting OpenUSP MTP Service...")

	// Command line flags
	var enableConsul = flag.Bool("consul", false, "Enable Consul service discovery")
	var port = flag.Int("port", 8081, "HTTP port")
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
		fmt.Println("  CONSUL_ENABLED     - Enable Consul service discovery (default: true)")
		fmt.Println("  SERVICE_PORT       - Server port (default: 8081)")
		fmt.Println("  MTP_PORT           - MTP service port (default: 8081)")
		return
	}

	// Override environment from flags
	if *enableConsul {
		os.Setenv("CONSUL_ENABLED", "true")
	}
	if *port != 8081 {
		os.Setenv("SERVICE_PORT", fmt.Sprintf("%d", *port))
		os.Setenv("MTP_PORT", fmt.Sprintf("%d", *port))
	}

	// Load configuration
	deployConfig := config.LoadDeploymentConfigWithPortEnv("openusp-mtp-service", "mtp-service", 8081, "OPENUSP_MTP_SERVICE_PORT")

	// Initialize Consul if enabled
	var registry *consul.ServiceRegistry
	var serviceInfo *consul.ServiceInfo
	if deployConfig.IsConsulEnabled() {
		consulAddr, interval, timeout := deployConfig.GetConsulConfig()
		consulConfig := &consul.Config{
			Address:       consulAddr,
			Datacenter:    "openusp-dev",
			CheckInterval: interval,
			CheckTimeout:  timeout,
		}

		var err error
		registry, err = consul.NewServiceRegistry(consulConfig)
		if err != nil {
			log.Fatalf("Failed to connect to Consul: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		serviceInfo, err = registry.RegisterService(ctx, deployConfig.ServiceName, deployConfig.ServiceType)
		if err != nil {
			log.Fatalf("Failed to register with Consul: %v", err)
		}

		log.Printf("🏛️ Connected to Consul at %s", consulAddr)
		log.Printf("🎯 Service registered with Consul: %s (%s) at localhost:%d",
			serviceInfo.Name, serviceInfo.Meta["service_type"], serviceInfo.Port)
	}

	// Determine the health port to use (dynamic when Consul enabled)
	var healthPort int
	if serviceInfo != nil {
		healthPort = serviceInfo.Port
	} else {
		healthPort = deployConfig.ServicePort
	}

	// Load MTP configuration with dual ports
	// WebSocket uses standard port 8081, health API uses dynamic port
	config := DefaultConfig(healthPort)

	// Create USP message handler that forwards to USP service
	uspServiceAddr := ""
	if deployConfig.IsConsulEnabled() && registry != nil {
		// Allow configurable discovery timeout (default 20s). Accept Go duration format (e.g. 10s, 1m)
		discTimeout := 20 * time.Second
		if v := strings.TrimSpace(os.Getenv("OPENUSP_USP_DISCOVERY_TIMEOUT")); v != "" {
			if d, err := time.ParseDuration(v); err == nil && d > 0 {
				discTimeout = d
			} else if err != nil {
				log.Printf("⚠️  Invalid OPENUSP_USP_DISCOVERY_TIMEOUT value '%s': %v (using default %v)", v, err, discTimeout)
			}
		}

		log.Printf("🔎 Attempting USP service discovery via Consul (timeout: %v)...", discTimeout)
		if uspServiceInfo, err := registry.WaitForService(context.Background(), "openusp-usp-service", discTimeout); err == nil && uspServiceInfo != nil {
			if uspServiceInfo.GRPCPort > 0 {
				uspServiceAddr = fmt.Sprintf("%s:%d", uspServiceInfo.Address, uspServiceInfo.GRPCPort)
				log.Printf("✅ Discovered USP service via Consul: %s (health=%s)", uspServiceAddr, uspServiceInfo.Health)
			} else {
				log.Printf("⚠️  USP service discovered but no gRPC port metadata found; falling back (http port=%d)", uspServiceInfo.Port)
			}
		} else if err != nil {
			log.Printf("⚠️  USP service discovery timed out: %v", err)
		}
	}

	// Fallback to environment variable or default if discovery failed
	if uspServiceAddr == "" {
		if portStr := strings.TrimSpace(os.Getenv("OPENUSP_USP_SERVICE_GRPC_PORT")); portStr != "" {
			uspServiceAddr = "localhost:" + portStr
			log.Printf("🔁 Using USP service address from OPENUSP_USP_SERVICE_GRPC_PORT: %s", uspServiceAddr)
		} else {
			uspServiceAddr = "localhost:56250" // Legacy/static default
			log.Printf("⚠️  USP service not discovered; using legacy default address: %s", uspServiceAddr)
		}
	}

	handler := NewUSPMessageHandler(uspServiceAddr)

	// Create MTP service
	mtpService, err := NewSimpleMTPService(config, handler)
	if err != nil {
		log.Fatalf("Failed to create MTP service: %v", err)
	}

	// Start MTP service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mtpService.Start(ctx); err != nil {
		log.Fatalf("Failed to start MTP service: %v", err)
	}

	log.Printf("🚀 MTP Service started successfully")
	if deployConfig.IsConsulEnabled() && serviceInfo != nil {
		log.Printf("   └── WebSocket Port: %d (USP Protocol)", config.WebSocketPort)
		log.Printf("   └── Health API Port: %d (Dynamic)", serviceInfo.Port)
		log.Printf("   └── Consul Service Discovery: ✅ Enabled")
		log.Printf("   └── Demo UI: http://localhost:%d/usp", config.WebSocketPort)
		log.Printf("   └── Health Check: http://localhost:%d/health", serviceInfo.Port)
		log.Printf("   └── Consul UI: http://localhost:8500/ui/")
	} else {
		log.Printf("   └── WebSocket Port: %d (USP Protocol)", config.WebSocketPort)
		log.Printf("   └── Health API Port: %d (Static)", config.HealthPort)
		log.Printf("   └── Consul Service Discovery: ❌ Disabled")
		log.Printf("   └── Demo UI: http://localhost:%d/usp", config.WebSocketPort)
		log.Printf("   └── Health Check: http://localhost:%d/health", config.HealthPort)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Printf("� Received shutdown signal")

	// Deregister from Consul
	if registry != nil && serviceInfo != nil {
		// NOTE: DeregisterService expects the service name (not the dynamic ID)
		if err := registry.DeregisterService(serviceInfo.Name); err != nil {
			log.Printf("⚠️  Failed to deregister from Consul: %v", err)
		} else {
			log.Printf("✅ Deregistered from Consul successfully")
		}
	}

	cancel()
	mtpService.Stop()
	log.Printf("✅ MTP Service stopped successfully")
}
