package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"openusp/pkg/config"
	pb_v1_3 "openusp/pkg/proto/v1_3"
	pb_v1_4 "openusp/pkg/proto/v1_4"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

const (
	// USP Agent Configuration - now loaded from configuration
	// Default values - should be overridden via configuration
	defaultAgentEndpointID = ""  // Must be set via configuration
	// controllerID is now loaded from configuration

	// Default Device Information - should be overridden via configuration
	defaultDeviceManufacturer    = ""  // Must be set via configuration
	defaultDeviceModelName       = ""  // Must be set via configuration
	defaultDeviceSerialNumber    = ""  // Must be set via configuration
	defaultDeviceSoftwareVersion = "1.0.0"
	defaultDeviceHardwareVersion = "1.0"
	defaultDeviceProductClass    = ""  // Must be set via configuration

	// Protocol Version Support
	defaultUSPVersion         = "1.3"
	supportedProtocolVersions = "1.3,1.4"
)

// Global configuration variables (loaded in main)
var (
	agentEndpointID       string
	deviceManufacturer    string
	deviceModelName       string
	deviceSerialNumber    string
	deviceSoftwareVersion string
	deviceHardwareVersion string
	deviceProductClass    string
)// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getControllerID retrieves the controller ID from configuration
func getControllerID() string {
	return getEnvOrDefault("OPENUSP_USP_ENDPOINT_ID", "proto::openusp.controller")
}

// printHelp displays command line usage information
func printHelp() {
	fmt.Println("OpenUSP TR-369 Agent")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  usp-agent [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --help, -h       Show this help message")
	fmt.Println("  --info, -i       Show agent information")
	fmt.Println("  --config FILE    Load configuration from YAML file")
	fmt.Println("  --version VER    Specify USP protocol version (1.3 or 1.4)")
	fmt.Println()
}

// printAgentInfo displays agent and device information
func printAgentInfo() {
	// Load configuration to show info
	agentConfig, _ := config.LoadCWMPAgentUnified("configs/openusp.yml")
	
	fmt.Println("OpenUSP TR-369 Demo Agent Information")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println("Agent Configuration:")
	fmt.Printf("  Endpoint ID: %s\n", getConfigValueOrDefault(agentConfig.EndpointID, defaultAgentEndpointID))
	fmt.Printf("  Controller ID: %s\n", getControllerID())
	fmt.Println()
	fmt.Println("Device Information:")
	fmt.Printf("  Manufacturer: %s\n", getConfigValueOrDefault(agentConfig.Manufacturer, defaultDeviceManufacturer))
	fmt.Printf("  Model Name: %s\n", getConfigValueOrDefault(agentConfig.ModelName, defaultDeviceModelName))
	fmt.Printf("  Serial Number: %s\n", getConfigValueOrDefault(agentConfig.SerialNumber, defaultDeviceSerialNumber))
	fmt.Printf("  Software Version: %s\n", getConfigValueOrDefault(agentConfig.SoftwareVersion, defaultDeviceSoftwareVersion))
	fmt.Printf("  Hardware Version: %s\n", getConfigValueOrDefault(agentConfig.HardwareVersion, defaultDeviceHardwareVersion))
	fmt.Printf("  Product Class: %s\n", getConfigValueOrDefault(agentConfig.ProductClass, defaultDeviceProductClass))
	fmt.Println()
	fmt.Println("Protocol Support:")
	fmt.Printf("  Default Version: %s\n", defaultUSPVersion)
	fmt.Printf("  Supported Versions: %s\n", supportedProtocolVersions)
}

// getConfigValueOrDefault returns config value or default if empty
func getConfigValueOrDefault(configValue, defaultValue string) string {
	if configValue != "" {
		return configValue
	}
	return defaultValue
}

// DeviceInfo holds device information
type DeviceInfo struct {
	EndpointID      string
	Manufacturer    string
	ModelName       string
	SerialNumber    string
	SoftwareVersion string
	HardwareVersion string
	ProductClass    string
}

// getDeviceInfo gets device information from configuration with fallbacks
func getDeviceInfo(agentConfig *config.TR369Config) *DeviceInfo {
	endpointID := defaultAgentEndpointID
	manufacturer := defaultDeviceManufacturer
	modelName := defaultDeviceModelName
	serialNumber := defaultDeviceSerialNumber
	softwareVersion := defaultDeviceSoftwareVersion
	hardwareVersion := defaultDeviceHardwareVersion
	productClass := defaultDeviceProductClass

	if agentConfig != nil {
		if agentConfig.EndpointID != "" {
			endpointID = agentConfig.EndpointID
		}
		if agentConfig.Manufacturer != "" {
			manufacturer = agentConfig.Manufacturer
		}
		if agentConfig.ModelName != "" {
			modelName = agentConfig.ModelName
		}
		if agentConfig.SerialNumber != "" {
			serialNumber = agentConfig.SerialNumber
		}
		if agentConfig.SoftwareVersion != "" {
			softwareVersion = agentConfig.SoftwareVersion
		}
		if agentConfig.HardwareVersion != "" {
			hardwareVersion = agentConfig.HardwareVersion
		}
		if agentConfig.ProductClass != "" {
			productClass = agentConfig.ProductClass
		}
	}

	// Environment variable overrides
	if env := os.Getenv("OPENUSP_USP_AGENT_ENDPOINT_ID"); env != "" {
		endpointID = env
	}
	if env := os.Getenv("OPENUSP_USP_AGENT_MANUFACTURER"); env != "" {
		manufacturer = env
	}
	if env := os.Getenv("OPENUSP_USP_AGENT_MODEL_NAME"); env != "" {
		modelName = env
	}
	if env := os.Getenv("OPENUSP_USP_AGENT_SERIAL_NUMBER"); env != "" {
		serialNumber = env
	}
	if env := os.Getenv("OPENUSP_USP_AGENT_SOFTWARE_VERSION"); env != "" {
		softwareVersion = env
	}
	if env := os.Getenv("OPENUSP_USP_AGENT_HARDWARE_VERSION"); env != "" {
		hardwareVersion = env
	}
	if env := os.Getenv("OPENUSP_USP_AGENT_PRODUCT_CLASS"); env != "" {
		productClass = env
	}

	return &DeviceInfo{
		EndpointID:      endpointID,
		Manufacturer:    manufacturer,
		ModelName:       modelName,
		SerialNumber:    serialNumber,
		SoftwareVersion: softwareVersion,
		HardwareVersion: hardwareVersion,
		ProductClass:    productClass,
	}
}

// getMTPServiceURL gets the MTP service WebSocket URL from configuration
func getMTPServiceURL() string {
	// Check if user provided explicit URL via environment
	if url := os.Getenv("USP_WS_URL"); url != "" {
		log.Printf("🔧 Using explicit MTP WebSocket URL from environment: %s", url)
		return url
	}

	// Try to load configuration from YAML
	configPath := "configs/usp-agent.yaml"
	if _, err := os.Stat(configPath); err == nil {
		tr369Config, err := config.LoadYAMLTR369Config(configPath)
		if err == nil && tr369Config.WebSocketURL != "" {
			log.Printf("✅ Using MTP WebSocket URL from YAML config: %s", tr369Config.WebSocketURL)
			return tr369Config.WebSocketURL
		}
	}

	// Fallback to environment-configured port or default
	mtpPort := 8081 // Default MTP WebSocket port
	if portStr := strings.TrimSpace(os.Getenv("OPENUSP_MTP_SERVICE_PORT")); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			mtpPort = p
		}
	}
	fallbackURL := fmt.Sprintf("ws://localhost:%d/ws", mtpPort)
	log.Printf("✅ Using default MTP WebSocket URL: %s", fallbackURL)
	return fallbackURL
}

// getMTPServiceURLWithConfig gets MTP service URL using configuration-driven approach
func getMTPServiceURLWithConfig(agentConfig *config.TR369Config) string {
	if agentConfig == nil {
		return getMTPServiceURL()
	}

	// Use configured WebSocket URL
	if agentConfig.WebSocketURL != "" {
		log.Printf("🔧 Using configured WebSocket URL: %s", agentConfig.WebSocketURL)
		return agentConfig.WebSocketURL
	}

	// Fallback to standard URL discovery
	return getMTPServiceURL()
}

// getMTPTransportType determines the MTP transport type from configuration
func getMTPTransportType() TransportType {
	// Try to load configuration from YAML
	configPath := "configs/usp-agent.yaml"
	if _, err := os.Stat(configPath); err == nil {
		tr369Config, err := config.LoadYAMLTR369Config(configPath)
		if err == nil && tr369Config.MTPType != "" {
			switch strings.ToLower(tr369Config.MTPType) {
			case "stomp":
				log.Printf("✅ Using STOMP transport from YAML config")
				return TransportSTOMP
			case "websocket", "ws":
				log.Printf("✅ Using WebSocket transport from YAML config")
				return TransportWebSocket
			default:
				log.Printf("⚠️ Unknown MTP transport type '%s' in config, defaulting to WebSocket", tr369Config.MTPType)
			}
		}
	}

	// Check environment variable fallback
	mtpType := os.Getenv("MTP_TYPE")
	if mtpType != "" {
		switch strings.ToLower(mtpType) {
		case "stomp":
			log.Printf("✅ Using STOMP transport from environment")
			return TransportSTOMP
		case "websocket", "ws":
			log.Printf("✅ Using WebSocket transport from environment")
			return TransportWebSocket
		default:
			log.Printf("⚠️ Unknown MTP transport type '%s' in environment, defaulting to WebSocket", mtpType)
		}
	}

	// Default to WebSocket
	log.Printf("✅ Using default WebSocket transport")
	return TransportWebSocket
}

// getMTPServiceURLForTransport gets the appropriate MTP service URL based on transport type
func getMTPServiceURLForTransport(transport TransportType, agentConfig *config.TR369Config) string {
	switch transport {
	case TransportSTOMP:
		// For STOMP, use the STOMP broker URL
		if agentConfig != nil && agentConfig.STOMPBrokerURL != "" {
			return agentConfig.STOMPBrokerURL
		}
		// Fallback to environment or default
		stompURL := os.Getenv("STOMP_BROKER_URL")
		if stompURL == "" {
			mtpPort := os.Getenv("OPENUSP_MTP_PORT")
			if mtpPort == "" {
				mtpPort = "61613" // Default STOMP port
			}
			stompURL = fmt.Sprintf("localhost:%s", mtpPort)
		}
		return stompURL
	case TransportWebSocket:
		// For WebSocket, use the existing WebSocket URL logic
		return getMTPServiceURLWithConfig(agentConfig)
	default:
		log.Printf("⚠️ Unknown transport type, falling back to WebSocket URL")
		return getMTPServiceURLWithConfig(agentConfig)
	}
}

// getAPIGatewayURL gets the API Gateway URL from configuration
func getAPIGatewayURL() string {
	// Try to load configuration from YAML
	configPath := "configs/usp-agent.yaml"
	if _, err := os.Stat(configPath); err == nil {
		tr369Config, err := config.LoadYAMLTR369Config(configPath)
		if err == nil && tr369Config.PlatformURL != "" {
			log.Printf("✅ Using API Gateway URL from YAML config: %s", tr369Config.PlatformURL)
			return tr369Config.PlatformURL
		}
	}

	// Fallback to environment or default with dynamic port
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayPort := 6500 // Default API Gateway port
		if portStr := strings.TrimSpace(os.Getenv("OPENUSP_API_GATEWAY_PORT")); portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				apiGatewayPort = p
			}
		}
		apiGatewayURL = fmt.Sprintf("https://localhost:%d", apiGatewayPort)
	}
	log.Printf("✅ Using API Gateway URL from environment/default: %s", apiGatewayURL)
	return apiGatewayURL
}

// TransportType defines the MTP transport protocol
type TransportType string

const (
	TransportWebSocket TransportType = "websocket"
	TransportSTOMP     TransportType = "stomp"
)

type USPClient struct {
	// Connection objects
	wsConn   *websocket.Conn
	stompConn net.Conn
	
	// Configuration
	transport    TransportType
	endpointID   string
	controllerID string
	msgID        string
	version      string // USP protocol version ("1.3" or "1.4")
	agentConfig  *config.TR369Config
	
	// STOMP specific
	stompSessionID string
}

func NewUSPClient(endpointID, controllerID, version string, transport TransportType, agentConfig *config.TR369Config) *USPClient {
	return &USPClient{
		transport:    transport,
		endpointID:   endpointID,
		controllerID: controllerID,
		msgID:        fmt.Sprintf("msg-%d", time.Now().Unix()),
		version:      version,
		agentConfig:  agentConfig,
	}
}

func (c *USPClient) Connect(url string) error {
	return c.ConnectWithSubprotocol(url, "")
}

func (c *USPClient) ConnectWithSubprotocol(url, subprotocol string) error {
	switch c.transport {
	case TransportWebSocket:
		return c.connectWebSocket(url, subprotocol)
	case TransportSTOMP:
		return c.connectSTOMP(url)
	default:
		return fmt.Errorf("unsupported transport type: %s", c.transport)
	}
}

func (c *USPClient) connectWebSocket(url, subprotocol string) error {
	log.Printf("Connecting to MTP Service via WebSocket at: %s", url)

	// Set up WebSocket headers with subprotocol if specified
	var headers http.Header
	if subprotocol != "" {
		headers = http.Header{}
		headers.Set("Sec-WebSocket-Protocol", subprotocol)
		log.Printf("Using WebSocket subprotocol: %s", subprotocol)
	}

	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.wsConn = conn

	// Log the selected subprotocol
	selectedSubprotocol := conn.Subprotocol()
	if selectedSubprotocol != "" {
		log.Printf("Connected to MTP Service successfully (subprotocol: %s)", selectedSubprotocol)
	} else {
		log.Printf("Connected to MTP Service successfully")
	}
	return nil
}

func (c *USPClient) connectSTOMP(url string) error {
	log.Printf("Connecting to MTP Service via STOMP at: %s", url)
	
	// Parse STOMP URL (format: stomp://host:port or tcp://host:port)
	var host, port string
	if strings.HasPrefix(url, "stomp://") {
		hostPort := strings.TrimPrefix(url, "stomp://")
		host, port, _ = net.SplitHostPort(hostPort)
	} else if strings.HasPrefix(url, "tcp://") {
		hostPort := strings.TrimPrefix(url, "tcp://")
		host, port, _ = net.SplitHostPort(hostPort)
	} else {
		return fmt.Errorf("invalid STOMP URL format: %s (expected stomp://host:port or tcp://host:port)", url)
	}
	
	if port == "" {
		port = "61613" // Default STOMP port
	}
	
	// Connect to STOMP broker
	conn, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return fmt.Errorf("failed to connect to STOMP broker: %w", err)
	}
	
	c.stompConn = conn
	
	// Send STOMP CONNECT frame with credentials matching RabbitMQ configuration
	// Use default RabbitMQ virtual host "/" instead of hostname
	connectFrame := "CONNECT\naccept-version:1.2\nhost:/\nlogin:openusp\npasscode:openusp123\n\n\x00"
	_, err = c.stompConn.Write([]byte(connectFrame))
	if err != nil {
		c.stompConn.Close()
		return fmt.Errorf("failed to send STOMP CONNECT frame: %w", err)
	}
	
	// Read CONNECTED response
	buffer := make([]byte, 1024)
	n, err := c.stompConn.Read(buffer)
	if err != nil {
		c.stompConn.Close()
		return fmt.Errorf("failed to read STOMP CONNECTED response: %w", err)
	}
	
	response := string(buffer[:n])
	if !strings.HasPrefix(response, "CONNECTED") {
		c.stompConn.Close()
		return fmt.Errorf("expected STOMP CONNECTED response, got: %s", response)
	}
	
	// Extract session ID from CONNECTED response
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "session:") {
			c.stompSessionID = strings.TrimPrefix(line, "session:")
			break
		}
	}
	
	log.Printf("Connected to STOMP broker successfully (session: %s)", c.stompSessionID)
	return nil
}

func (c *USPClient) Close() error {
	switch c.transport {
	case TransportWebSocket:
		if c.wsConn != nil {
			return c.wsConn.Close()
		}
	case TransportSTOMP:
		if c.stompConn != nil {
			// Send STOMP DISCONNECT frame
			disconnectFrame := fmt.Sprintf("DISCONNECT\nsession:%s\n\n\x00", c.stompSessionID)
			c.stompConn.Write([]byte(disconnectFrame))
			return c.stompConn.Close()
		}
	}
	return nil
}

// createOnboardingMessage creates a USP Notify message with device onboarding information
func (c *USPClient) createOnboardingMessage() ([]byte, error) {
	if c.version == "1.4" {
		return c.createOnboardingMessageV14()
	}
	return c.createOnboardingMessageV13()
}

// createOnboardingMessageV13 creates a USP v1.3 Notify message with device onboarding information
func (c *USPClient) createOnboardingMessageV13() ([]byte, error) {
	// Create USP Notify Request for Device Onboarding using v1.3
	notifyReq := &pb_v1_3.Notify{
		SubscriptionId: "device-onboarding",
		SendResp:       true,
		Notification: &pb_v1_3.Notify_OnBoardReq{
			OnBoardReq: &pb_v1_3.Notify_OnBoardRequest{
				Oui:                            "001122", // IEEE OUI for OpenUSP
				ProductClass:                   deviceProductClass,
				SerialNumber:                   deviceSerialNumber,
				AgentSupportedProtocolVersions: supportedProtocolVersions,
			},
		},
	}

	// Create USP Request
	request := &pb_v1_3.Request{
		ReqType: &pb_v1_3.Request_Notify{
			Notify: notifyReq,
		},
	}

	// Create USP Message
	msg := &pb_v1_3.Msg{
		Header: &pb_v1_3.Header{
			MsgId:   fmt.Sprintf("onboard-%d", time.Now().Unix()),
			MsgType: pb_v1_3.Header_NOTIFY,
		},
		Body: &pb_v1_3.Body{
			MsgBody: &pb_v1_3.Body_Request{
				Request: request,
			},
		},
	}

	// Create USP Record
	record := &pb_v1_3.Record{
		Version:         "1.3",
		ToId:            c.controllerID,
		FromId:          c.endpointID,
		PayloadSecurity: pb_v1_3.Record_PLAINTEXT,
		MacSignature:    []byte{},
		SenderCert:      []byte{},
		RecordType: &pb_v1_3.Record_NoSessionContext{
			NoSessionContext: &pb_v1_3.NoSessionContextRecord{
				Payload: func() []byte {
					data, _ := proto.Marshal(msg)
					return data
				}(),
			},
		},
	}

	return proto.Marshal(record)
}

// createOnboardingMessageV14 creates a USP v1.4 Notify message with device onboarding information
func (c *USPClient) createOnboardingMessageV14() ([]byte, error) {
	// Create USP Notify Request for Device Onboarding
	notifyReq := &pb_v1_4.Notify{
		SubscriptionId: "device-onboarding",
		SendResp:       true,
		Notification: &pb_v1_4.Notify_OnBoardReq{
			OnBoardReq: &pb_v1_4.Notify_OnBoardRequest{
				Oui:                            "001122", // IEEE OUI for OpenUSP
				ProductClass:                   deviceProductClass,
				SerialNumber:                   deviceSerialNumber,
				AgentSupportedProtocolVersions: "1.3,1.4",
			},
		},
	}

	// Create USP Request
	request := &pb_v1_4.Request{
		ReqType: &pb_v1_4.Request_Notify{
			Notify: notifyReq,
		},
	}

	// Create USP Message
	msgID := fmt.Sprintf("onboard-%d", time.Now().Unix())
	msg := &pb_v1_4.Msg{
		Header: &pb_v1_4.Header{
			MsgId:   msgID,
			MsgType: pb_v1_4.Header_NOTIFY,
		},
		Body: &pb_v1_4.Body{
			MsgBody: &pb_v1_4.Body_Request{
				Request: request,
			},
		},
	}

	// Create USP Record with NoSessionContext
	record := &pb_v1_4.Record{
		Version:         "1.4",
		ToId:            c.controllerID,
		FromId:          c.endpointID,
		PayloadSecurity: pb_v1_4.Record_PLAINTEXT,
		MacSignature:    []byte{},
		SenderCert:      []byte{},
		RecordType: &pb_v1_4.Record_NoSessionContext{
			NoSessionContext: &pb_v1_4.NoSessionContextRecord{
				Payload: func() []byte {
					msgBytes, _ := proto.Marshal(msg)
					return msgBytes
				}(),
			},
		},
	}

	return proto.Marshal(record)
}

func (c *USPClient) createGetMessage() ([]byte, error) {
	if c.version == "1.4" {
		return c.createGetMessageV14()
	}
	return c.createGetMessageV13()
}

// createGetMessageV13 creates a USP v1.3 Get request
func (c *USPClient) createGetMessageV13() ([]byte, error) {
	// Create USP Get Request using v1.3
	getReq := &pb_v1_3.Get{
		ParamPaths: []string{
			"Device.DeviceInfo.",
			"Device.Ethernet.",
		},
		MaxDepth: 0, // Get all parameters under the specified paths
	}

	// Create USP Request
	request := &pb_v1_3.Request{
		ReqType: &pb_v1_3.Request_Get{
			Get: getReq,
		},
	}

	// Create USP Message
	msg := &pb_v1_3.Msg{
		Header: &pb_v1_3.Header{
			MsgId:   c.msgID,
			MsgType: pb_v1_3.Header_GET,
		},
		Body: &pb_v1_3.Body{
			MsgBody: &pb_v1_3.Body_Request{
				Request: request,
			},
		},
	}

	// Create USP Record with NoSessionContext
	record := &pb_v1_3.Record{
		Version:         "1.3",
		ToId:            c.controllerID,
		FromId:          c.endpointID,
		PayloadSecurity: pb_v1_3.Record_PLAINTEXT,
		MacSignature:    []byte{},
		SenderCert:      []byte{},
		RecordType: &pb_v1_3.Record_NoSessionContext{
			NoSessionContext: &pb_v1_3.NoSessionContextRecord{
				Payload: func() []byte {
					msgBytes, _ := proto.Marshal(msg)
					return msgBytes
				}(),
			},
		},
	}

	return proto.Marshal(record)
}

// createGetMessageV14 creates a USP v1.4 Get request
func (c *USPClient) createGetMessageV14() ([]byte, error) {
	// Create USP Get Request using v1.4
	getReq := &pb_v1_4.Get{
		ParamPaths: []string{
			"Device.DeviceInfo.",
			"Device.Ethernet.",
		},
		MaxDepth: 0, // Get all parameters under the specified paths
	}

	// Create USP Request
	request := &pb_v1_4.Request{
		ReqType: &pb_v1_4.Request_Get{
			Get: getReq,
		},
	}

	// Create USP Message
	msg := &pb_v1_4.Msg{
		Header: &pb_v1_4.Header{
			MsgId:   c.msgID,
			MsgType: pb_v1_4.Header_GET,
		},
		Body: &pb_v1_4.Body{
			MsgBody: &pb_v1_4.Body_Request{
				Request: request,
			},
		},
	}

	// Create USP Record with NoSessionContext
	record := &pb_v1_4.Record{
		Version:         "1.4",
		ToId:            c.controllerID,
		FromId:          c.endpointID,
		PayloadSecurity: pb_v1_4.Record_PLAINTEXT,
		MacSignature:    []byte{},
		SenderCert:      []byte{},
		RecordType: &pb_v1_4.Record_NoSessionContext{
			NoSessionContext: &pb_v1_4.NoSessionContextRecord{
				Payload: func() []byte {
					msgBytes, _ := proto.Marshal(msg)
					return msgBytes
				}(),
			},
		},
	}

	return proto.Marshal(record)
}

func (c *USPClient) sendRecord(recordBytes []byte) error {
	log.Printf("Sending USP Record (size: %d bytes, version: %s) via %s", len(recordBytes), c.version, c.transport)

	switch c.transport {
	case TransportWebSocket:
		return c.sendWebSocketRecord(recordBytes)
	case TransportSTOMP:
		return c.sendSTOMPRecord(recordBytes)
	default:
		return fmt.Errorf("unsupported transport type: %s", c.transport)
	}
}

func (c *USPClient) sendWebSocketRecord(recordBytes []byte) error {
	// Send binary USP Record over WebSocket
	err := c.wsConn.WriteMessage(websocket.BinaryMessage, recordBytes)
	if err != nil {
		return fmt.Errorf("failed to send WebSocket message: %w", err)
	}

	log.Printf("USP Record sent successfully via WebSocket")
	return nil
}

func (c *USPClient) sendSTOMPRecord(recordBytes []byte) error {
	// Send USP Record via STOMP SEND frame - use configurable destination
	destination := "/queue/usp.agent.request" // default fallback
	if c.agentConfig != nil && c.agentConfig.STOMPDestinationRequest != "" {
		destination = c.agentConfig.STOMPDestinationRequest
	}
	contentLength := len(recordBytes)
	
	// Create STOMP SEND frame header
	header := fmt.Sprintf("SEND\ndestination:%s\ncontent-type:application/octet-stream\ncontent-length:%d\n\n", destination, contentLength)
	
	// Create complete STOMP frame: header + binary payload + null terminator
	frame := make([]byte, len(header)+len(recordBytes)+1)
	copy(frame, []byte(header))
	copy(frame[len(header):], recordBytes)
	frame[len(frame)-1] = 0x00 // STOMP null terminator
	
	_, err := c.stompConn.Write(frame)
	if err != nil {
		return fmt.Errorf("failed to send STOMP message: %w", err)
	}

	log.Printf("USP Record sent successfully via STOMP")
	return nil
}

func (c *USPClient) readResponse() error {
	switch c.transport {
	case TransportWebSocket:
		return c.readWebSocketResponse()
	case TransportSTOMP:
		return c.readSTOMPResponse()
	default:
		return fmt.Errorf("unsupported transport type: %s", c.transport)
	}
}

func (c *USPClient) readWebSocketResponse() error {
	// Set read deadline
	c.wsConn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Read response
	_, message, err := c.wsConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read WebSocket response: %w", err)
	}

	log.Printf("Received WebSocket response (size: %d bytes)", len(message))
	return nil
}

func (c *USPClient) readSTOMPResponse() error {
	// Set read deadline
	c.stompConn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Read STOMP response frame
	buffer := make([]byte, 4096)
	n, err := c.stompConn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read STOMP response: %w", err)
	}

	response := string(buffer[:n])
	log.Printf("Received STOMP response (size: %d bytes)", len(response))
	
	// Parse STOMP MESSAGE frame
	if strings.HasPrefix(response, "MESSAGE") {
		// Extract message body (after double newline)
		parts := strings.Split(response, "\n\n")
		if len(parts) >= 2 {
			messageBody := parts[1]
			// Remove null terminator
			messageBody = strings.TrimSuffix(messageBody, "\x00")
			log.Printf("STOMP message body: %s", messageBody)
		}
	}
	
	return nil
}

// registerDeviceWithAPI registers the device with the OpenUSP API Gateway
func registerDeviceWithAPI(agentConfig *config.TR369Config, endpointID string) error {
	if agentConfig != nil && !agentConfig.AutoRegister {
		log.Printf("📋 Auto-registration disabled in configuration")
		return nil
	}

	// Get API Gateway URL
	apiURL := getAPIGatewayURL()

	// Prepare device registration data
	deviceData := map[string]interface{}{
		"endpoint_id":      endpointID,
		"manufacturer":     deviceManufacturer,
		"model_name":       deviceModelName,
		"serial_number":    deviceSerialNumber,
		"software_version": deviceSoftwareVersion,
		"hardware_version": deviceHardwareVersion,
		"product_class":    deviceProductClass,
		"device_type":      "agent",
	}

	// Override with configuration values if available
	if agentConfig != nil {
		if agentConfig.Manufacturer != "" {
			deviceData["manufacturer"] = agentConfig.Manufacturer
		}
		if agentConfig.ModelName != "" {
			deviceData["model_name"] = agentConfig.ModelName
		}
		if agentConfig.SerialNumber != "" {
			deviceData["serial_number"] = agentConfig.SerialNumber
		}
		if agentConfig.SoftwareVersion != "" {
			deviceData["software_version"] = agentConfig.SoftwareVersion
		}
		if agentConfig.HardwareVersion != "" {
			deviceData["hardware_version"] = agentConfig.HardwareVersion
		}
		if agentConfig.ProductClass != "" {
			deviceData["product_class"] = agentConfig.ProductClass
		}
		if agentConfig.DeviceType != "" {
			deviceData["device_type"] = agentConfig.DeviceType
		}
	}

	jsonData, err := json.Marshal(deviceData)
	if err != nil {
		return fmt.Errorf("failed to marshal device data: %v", err)
	}

	// Register device with API Gateway
	registrationURL := apiURL + "/api/v1/devices"
	log.Printf("📡 Registering device with API Gateway at: %s", registrationURL)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(registrationURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("⚠️  Device registration failed (will continue): %v", err)
		return nil // Don't fail the agent if registration fails
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		log.Printf("✅ Device registered successfully with API Gateway")
	} else if resp.StatusCode == 409 {
		log.Printf("📋 Device already registered (conflict - normal for re-registration)")
	} else {
		log.Printf("⚠️  Device registration returned status %d (will continue)", resp.StatusCode)
	}

	return nil
}

func demonstrateUSPOperations(client *USPClient) error {
	log.Printf("\n🚀 Starting TR-369 USP Client Demonstration")
	log.Printf("=========================================")

	// Step 1: Send Device Onboarding Request
	log.Printf("\n1️⃣  Sending Device Onboarding Request...")
	log.Printf("   Device: %s %s (S/N: %s)", deviceManufacturer, deviceModelName, deviceSerialNumber)
	log.Printf("   Agent Endpoint: %s", client.endpointID)

	onboardRecord, err := client.createOnboardingMessage()
	if err != nil {
		return fmt.Errorf("failed to create onboarding message: %w", err)
	}

	if err := client.sendRecord(onboardRecord); err != nil {
		return fmt.Errorf("failed to send onboarding request: %w", err)
	}

	log.Printf("Waiting for onboarding response...")
	if err := client.readResponse(); err != nil {
		log.Printf("Error reading onboarding response: %v", err)
	}

	// Small delay between operations
	time.Sleep(2 * time.Second)

	// Step 2: Send Get Request to retrieve device parameters
	log.Printf("\n2️⃣  Sending USP GET Request...")
	getRecord, err := client.createGetMessage()
	if err != nil {
		return fmt.Errorf("failed to create GET message: %w", err)
	}

	if err := client.sendRecord(getRecord); err != nil {
		return fmt.Errorf("failed to send GET request: %w", err)
	}

	log.Printf("Waiting for GET response...")
	if err := client.readResponse(); err != nil {
		log.Printf("Error reading GET response: %v", err)
	}

	return nil
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to YAML configuration file (optional)")
	version := flag.String("version", defaultUSPVersion, "USP protocol version (1.3 or 1.4)")
	validateOnly := flag.Bool("validate-only", false, "Validate configuration and exit")
	showHelp := flag.Bool("help", false, "Show help information")
	showInfo := flag.Bool("info", false, "Show agent information")
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	if *showInfo {
		printAgentInfo()
		return
	}

	// Load configuration - require config file
	if *configPath == "" {
		log.Fatalf("Configuration file is required. Use --config to specify the path to openusp.yml")
	}

	// Load from unified YAML file
	agentConfig, err := config.LoadUSPAgentUnified(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration from %s: %v", *configPath, err)
	}
	log.Printf("Loaded configuration from: %s", *configPath)

	// Use configuration values
	*version = agentConfig.USPVersion
	if *version == "" {
		*version = defaultUSPVersion
	}

	// Validate configuration if requested
	if *validateOnly {
		if agentConfig != nil {
			log.Printf("✅ Configuration validation passed")
			log.Printf("USP Version: %s", agentConfig.USPVersion)
			log.Printf("Endpoint ID: %s", agentConfig.EndpointID)
			log.Printf("WebSocket URL: %s", agentConfig.WebSocketURL)
		} else {
			log.Printf("✅ Default configuration validation passed")
		}
		return
	}

	// Validate version
	if *version != "1.3" && *version != "1.4" {
		log.Fatalf("Unsupported USP version: %s. Supported versions: %s", *version, supportedProtocolVersions)
	}

	log.Printf("OpenUSP TR-369 Client Example")
	log.Printf("=============================")
	log.Printf("This example demonstrates TR-369 USP protocol communication")
	log.Printf("with the OpenUSP platform via WebSocket MTP.")
	log.Printf("USP Protocol Version: %s", *version)

	// Load configuration values into global variables
	if agentConfig != nil {
		agentEndpointID = getConfigValueOrDefault(agentConfig.EndpointID, defaultAgentEndpointID)
		deviceManufacturer = getConfigValueOrDefault(agentConfig.Manufacturer, defaultDeviceManufacturer)
		deviceModelName = getConfigValueOrDefault(agentConfig.ModelName, defaultDeviceModelName)
		deviceSerialNumber = getConfigValueOrDefault(agentConfig.SerialNumber, defaultDeviceSerialNumber)
		deviceSoftwareVersion = getConfigValueOrDefault(agentConfig.SoftwareVersion, defaultDeviceSoftwareVersion)
		deviceHardwareVersion = getConfigValueOrDefault(agentConfig.HardwareVersion, defaultDeviceHardwareVersion)
		deviceProductClass = getConfigValueOrDefault(agentConfig.ProductClass, defaultDeviceProductClass)
	} else {
		agentEndpointID = defaultAgentEndpointID
		deviceManufacturer = defaultDeviceManufacturer
		deviceModelName = defaultDeviceModelName
		deviceSerialNumber = defaultDeviceSerialNumber
		deviceSoftwareVersion = defaultDeviceSoftwareVersion
		deviceHardwareVersion = defaultDeviceHardwareVersion
		deviceProductClass = defaultDeviceProductClass
	}
	
	// Use fallbacks if still empty
	if agentEndpointID == "" {
		agentEndpointID = "proto::usp.agent.001"
	}
	if deviceManufacturer == "" {
		deviceManufacturer = "Plume Design"
	}
	if deviceModelName == "" {
		deviceModelName = "USP-Agent-Demo"
	}
	if deviceSerialNumber == "" {
		deviceSerialNumber = "USP-DEMO-001"
	}
	if deviceProductClass == "" {
		deviceProductClass = "HomeGateway"
	}

	endpointID := agentEndpointID
	log.Printf("Using endpoint ID: %s", endpointID)
	log.Printf("")

	// Determine transport type from configuration
	transport := getMTPTransportType()

	// Create USP Client with selected version and transport
	client := NewUSPClient(endpointID, getControllerID(), *version, transport, agentConfig)
	defer client.Close()

	// Get service URL based on transport type
	serviceURL := getMTPServiceURLForTransport(transport, agentConfig)

	// Handle transport-specific connection logic
	if transport == TransportSTOMP {
		// STOMP connection logic
		log.Printf("🔌 Connecting to STOMP broker at: %s", serviceURL)
		if err := client.ConnectWithSubprotocol(serviceURL, ""); err != nil {
			log.Fatalf("❌ Failed to connect to STOMP broker: %v", err)
		}
		log.Printf("✅ Connected to STOMP broker successfully")
	} else {
		// WebSocket connection logic
		log.Printf("🔌 Connecting to WebSocket MTP at: %s", serviceURL)
		
		// Get WebSocket subprotocol from configuration
		var subprotocol string
		if agentConfig != nil && agentConfig.WebSocketSubprotocol != "" {
			subprotocol = agentConfig.WebSocketSubprotocol
		} else {
			subprotocol = "v1.usp" // Default USP WebSocket subprotocol
		}

		// Connect to MTP Service (device registration will happen via USP Notify OnBoard message per TR-369)
		if err := client.ConnectWithSubprotocol(serviceURL, subprotocol); err != nil {
			log.Fatalf("❌ Failed to connect to MTP Service: %v", err)
		}
		log.Printf("✅ Connected to WebSocket MTP successfully")
	}

	// Wait a moment for connection to stabilize
	time.Sleep(1 * time.Second)

	// Demonstrate USP operations
	if err := demonstrateUSPOperations(client); err != nil {
		log.Fatalf("Demonstration failed: %v", err)
	}

	log.Printf("\n✅ TR-369 USP Client demonstration completed!")
	log.Printf("\nNote: Make sure the OpenUSP services are running:")
	log.Printf("  make infra-up                    # Start infrastructure (PostgreSQL)")
	log.Printf("  make build-all                   # Build all services")
	log.Printf("  make start-all                   # Start all OpenUSP services")
	log.Printf("\nThen run this agent with:")
	log.Printf("  make start-usp-agent")
	log.Printf("  ./build/usp-agent --config configs/usp-agent.yaml")
	log.Printf("\nThe agent will connect to the MTP Service using static configuration.")
}
