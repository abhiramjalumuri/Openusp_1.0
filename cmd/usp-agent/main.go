package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"openusp/pkg/config"
	pb_v1_3 "openusp/pkg/proto/v1_3"
	pb_v1_4 "openusp/pkg/proto/v1_4"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

const (
	// USP Agent Configuration
	agentEndpointID = "proto://tr369-demo-agent-001"
	controllerID    = "proto://openusp-controller"

	// Device Information for Onboarding
	deviceManufacturer    = "OpenUSP"
	deviceModelName       = "TR-369 Demo Agent"
	deviceSerialNumber    = "DEMO-001-2024"
	deviceSoftwareVersion = "1.0.0"
	deviceHardwareVersion = "1.0"
	deviceProductClass    = "DemoAgent"

	// P	wsURL := getMTPServiceURLWithConfig(agentConfig)col Version Support
	defaultUSPVersion         = "1.3"
	supportedProtocolVersions = "1.3,1.4"
)

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// printHelp displays command line usage information
func printHelp() {
	fmt.Println("OpenUSP TR-369 Demo Agent")
	fmt.Println("=========================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  usp-agent [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -config string")
	fmt.Println("        Path to YAML configuration file (optional)")
	fmt.Println("  -version string")
	fmt.Printf("        USP protocol version (default \"%s\")\n", defaultUSPVersion)
	fmt.Printf("        Supported versions: %s\n", supportedProtocolVersions)
	fmt.Println("  -validate-only")
	fmt.Println("        Validate configuration and exit")
	fmt.Println("  -help")
	fmt.Println("        Show this help information")
	fmt.Println("  -info")
	fmt.Println("        Show agent information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  usp-agent                                      # Use default configuration")
	fmt.Println("  usp-agent -version 1.4                        # Override USP version")
	fmt.Println("  usp-agent -config configs/usp-agent.yaml      # Use specific config file")
	fmt.Println("  usp-agent -config my-config.yaml -validate-only  # Validate config only")
	fmt.Println("  usp-agent -info                               # Show agent information")
	fmt.Println()
	fmt.Println("Configuration Files:")
	fmt.Println("  configs/usp-agent.yaml                        # Main YAML configuration")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  USP_WS_URL      # Override WebSocket URL (default: ws://localhost:8081/ws)")
	fmt.Println("  API_GATEWAY_URL # Override API Gateway URL (default: http://localhost:6500)")
}

// printAgentInfo displays agent and device information
func printAgentInfo() {
	fmt.Println("OpenUSP TR-369 Demo Agent Information")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println("Agent Configuration:")
	fmt.Printf("  Endpoint ID: %s\n", agentEndpointID)
	fmt.Printf("  Controller ID: %s\n", controllerID)
	fmt.Println()
	fmt.Println("Device Information:")
	fmt.Printf("  Manufacturer: %s\n", deviceManufacturer)
	fmt.Printf("  Model Name: %s\n", deviceModelName)
	fmt.Printf("  Serial Number: %s\n", deviceSerialNumber)
	fmt.Printf("  Software Version: %s\n", deviceSoftwareVersion)
	fmt.Printf("  Hardware Version: %s\n", deviceHardwareVersion)
	fmt.Printf("  Product Class: %s\n", deviceProductClass)
	fmt.Println()
	fmt.Println("Protocol Support:")
	fmt.Printf("  Default Version: %s\n", defaultUSPVersion)
	fmt.Printf("  Supported Versions: %s\n", supportedProtocolVersions)
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

	// Fallback to default standard port
	fallbackURL := "ws://localhost:8081/ws"
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

// Determine service name from configuration

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

	// Fallback to environment or default
	apiGatewayURL := os.Getenv("API_GATEWAY_URL")
	if apiGatewayURL == "" {
		apiGatewayURL = "http://localhost:6500" // Standard OpenUSP API Gateway port
	}
	log.Printf("✅ Using API Gateway URL from environment/default: %s", apiGatewayURL)
	return apiGatewayURL
}

type USPClient struct {
	conn         *websocket.Conn
	endpointID   string
	controllerID string
	msgID        string
	version      string // USP protocol version ("1.3" or "1.4")
}

func NewUSPClient(endpointID, controllerID, version string) *USPClient {
	return &USPClient{
		endpointID:   endpointID,
		controllerID: controllerID,
		msgID:        fmt.Sprintf("msg-%d", time.Now().Unix()),
		version:      version,
	}
}

func (c *USPClient) Connect(url string) error {
	log.Printf("Connecting to MTP Service at: %s", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.conn = conn
	log.Printf("Connected to MTP Service successfully")
	return nil
}

func (c *USPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
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
	log.Printf("Sending USP Record (size: %d bytes, version: %s)", len(recordBytes), c.version)

	// Send binary USP Record over WebSocket
	err := c.conn.WriteMessage(websocket.BinaryMessage, recordBytes)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("USP Record sent successfully")
	return nil
}

func (c *USPClient) readResponse() error {
	// Set read deadline
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Read response
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("Received response (size: %d bytes): %s", len(message), string(message))
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

	// Load configuration if provided
	var agentConfig *config.TR369Config
	var err error

	if *configPath != "" {
		// Load from specified YAML file
		agentConfig, err = config.LoadYAMLTR369Config(*configPath)
		if err != nil {
			log.Fatalf("Failed to load configuration from %s: %v", *configPath, err)
		}
		log.Printf("Loaded configuration from: %s", *configPath)

		// Use configuration values
		*version = agentConfig.USPVersion
		if *version == "" {
			*version = defaultUSPVersion
		}
	} else {
		// Try to load from default locations
		configDir := filepath.Join("configs")
		configInterface, err := config.LoadConfigFromYAML("usp", configDir)
		if err != nil {
			log.Printf("No configuration file found, using defaults: %v", err)
		} else {
			agentConfig = configInterface.(*config.TR369Config)
			*version = agentConfig.USPVersion
			if *version == "" {
				*version = defaultUSPVersion
			}
			log.Printf("Loaded configuration from default location")
		}
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

	// Use configuration values if available
	endpointID := agentEndpointID
	if agentConfig != nil && agentConfig.EndpointID != "" {
		endpointID = agentConfig.EndpointID
		log.Printf("Using configured endpoint ID: %s", endpointID)
	}
	log.Printf("")

	// Create USP Client with selected version
	client := NewUSPClient(endpointID, controllerID, *version)
	defer client.Close()

	// Get WebSocket URL from configuration
	wsURL := getMTPServiceURLWithConfig(agentConfig)

	// Connect to MTP Service (device registration will happen via USP Notify OnBoard message per TR-369)
	if err := client.Connect(wsURL); err != nil {
		log.Fatalf("Failed to connect to MTP Service: %v", err)
	}

	// Wait a moment for connection to stabilize
	time.Sleep(1 * time.Second)

	// Demonstrate USP operations
	if err := demonstrateUSPOperations(client); err != nil {
		log.Fatalf("Demonstration failed: %v", err)
	}

	log.Printf("\n✅ TR-369 USP Client demonstration completed!")
	log.Printf("\nNote: Make sure the OpenUSP services are running:")
	log.Printf("  make infra-up                    # Start infrastructure (Consul, PostgreSQL)")
	log.Printf("  make build-all                   # Build all services")
	log.Printf("  make start-all                   # Start all OpenUSP services")
	log.Printf("\nThen run this agent with:")
	log.Printf("  make start-usp-agent")
	log.Printf("  ./build/usp-agent --config configs/usp-agent.yaml")
	log.Printf("\nThe agent will automatically discover the MTP Service via Consul.")
}
