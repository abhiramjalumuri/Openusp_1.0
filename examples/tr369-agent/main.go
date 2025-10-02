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

	// Protocol Version Support
	defaultUSPVersion         = "1.3"
	supportedProtocolVersions = "1.3,1.4"
)

// ConsulService represents a service discovered from Consul
type ConsulService struct {
	ServiceAddress string `json:"ServiceAddress"`
	ServicePort    int    `json:"ServicePort"`
}

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
	fmt.Println("  tr369-agent [flags]")
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
	fmt.Println("  tr369-agent                                    # Use default configuration")
	fmt.Println("  tr369-agent -version 1.4                      # Override USP version")
	fmt.Println("  tr369-agent -config configs/tr369-agent.yaml  # Use specific config file")
	fmt.Println("  tr369-agent -config my-config.yaml -validate-only  # Validate config only")
	fmt.Println("  tr369-agent -info                             # Show agent information")
	fmt.Println()
	fmt.Println("Configuration Files:")
	fmt.Println("  configs/tr369-agent.yaml                      # Main YAML configuration")
	fmt.Println("  configs/examples/yaml/tr369-consul-enabled.yaml   # Consul enabled example")
	fmt.Println("  configs/examples/yaml/tr369-consul-disabled.yaml  # Standalone example")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  USP_WS_URL      # Override WebSocket URL (default: auto-discover)")
	fmt.Println("  CONSUL_ADDR     # Consul address for service discovery (default: localhost:8500)")
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

// discoverMTPService discovers the MTP service WebSocket URL via Consul or fallback
func discoverMTPService() (string, error) {
	// Check if user provided explicit URL
	if url := os.Getenv("USP_WS_URL"); url != "" {
		log.Printf("🔧 Using explicit MTP WebSocket URL: %s", url)
		return url, nil
	}

	// Try to discover via Consul
	consulAddr := getEnvOrDefault("CONSUL_ADDR", "localhost:8500")
	consulURL := fmt.Sprintf("http://%s/v1/catalog/service/openusp-mtp-service", consulAddr)

	log.Printf("🔍 Discovering MTP service via Consul at %s", consulURL)

	resp, err := http.Get(consulURL)
	if err != nil {
		// Fallback to default URL if Consul is not available
		fallbackURL := "ws://localhost:8081/ws"
		log.Printf("⚠️  Consul not available, using fallback URL: %s", fallbackURL)
		return fallbackURL, nil
	}
	defer resp.Body.Close()

	var services []ConsulService
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return "", fmt.Errorf("failed to decode Consul response: %v", err)
	}

	if len(services) == 0 {
		return "", fmt.Errorf("no MTP service found in Consul")
	}

	// Use the first available service
	service := services[0]
	address := service.ServiceAddress
	if address == "localhost" {
		address = "localhost" // Keep localhost for local development
	}

	wsURL := fmt.Sprintf("ws://%s:%d/ws", address, service.ServicePort)
	log.Printf("✅ Discovered MTP service WebSocket URL: %s", wsURL)
	return wsURL, nil
}

// discoverMTPServiceWithConfig discovers MTP service using configuration-driven approach
func discoverMTPServiceWithConfig(agentConfig *config.TR369Config) (string, error) {
	if agentConfig == nil {
		return discoverMTPService()
	}

	// If Consul is disabled or not configured, use static URL
	if !agentConfig.ConsulEnabled || !agentConfig.ServiceDiscoveryEnabled {
		if agentConfig.WebSocketURL != "" {
			log.Printf("🔧 Using configured WebSocket URL: %s", agentConfig.WebSocketURL)
			return agentConfig.WebSocketURL, nil
		}
		// Fallback to default static URL
		fallbackURL := "ws://localhost:8081/ws"
		log.Printf("🔧 Using default WebSocket URL: %s", fallbackURL)
		return fallbackURL, nil
	}

	// Consul-enabled dynamic discovery
	consulAddr := agentConfig.ConsulAddr
	if consulAddr == "" {
		consulAddr = "localhost:8500"
	}

	// Determine service name from configuration
	serviceName := "openusp-mtp-service"
	if agentConfig.ConsulMTPServiceName != "" {
		serviceName = agentConfig.ConsulMTPServiceName
	}

	consulURL := fmt.Sprintf("http://%s/v1/catalog/service/%s", consulAddr, serviceName)
	log.Printf("🔍 Discovering MTP service '%s' via Consul at %s", serviceName, consulURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(consulURL)
	if err != nil {
		// Fallback to configured static URL if Consul fails
		if agentConfig.WebSocketURL != "" {
			log.Printf("⚠️  Consul discovery failed, using configured URL: %s", agentConfig.WebSocketURL)
			return agentConfig.WebSocketURL, nil
		}
		return "", fmt.Errorf("consul discovery failed and no fallback URL configured: %v", err)
	}
	defer resp.Body.Close()

	var services []ConsulService
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return "", fmt.Errorf("failed to decode Consul response: %v", err)
	}

	if len(services) == 0 {
		// Fallback to configured static URL if no services found
		if agentConfig.WebSocketURL != "" {
			log.Printf("⚠️  No MTP service found in Consul, using configured URL: %s", agentConfig.WebSocketURL)
			return agentConfig.WebSocketURL, nil
		}
		return "", fmt.Errorf("no MTP service found in Consul and no fallback URL configured")
	}

	// Use the first available service
	service := services[0]
	address := service.ServiceAddress
	if address == "" || address == "localhost" {
		address = "localhost" // Keep localhost for local development
	}

	wsURL := fmt.Sprintf("ws://%s:%d/ws", address, service.ServicePort)
	log.Printf("✅ Discovered MTP service WebSocket URL via Consul: %s", wsURL)
	return wsURL, nil
}

// discoverAPIGatewayWithConfig discovers API Gateway service for device registration
func discoverAPIGatewayWithConfig(agentConfig *config.TR369Config) (string, error) {
	if agentConfig == nil {
		// Default API Gateway URL
		return "http://localhost:6500", nil
	}

	// If Consul is disabled, use static configuration
	if !agentConfig.ConsulEnabled || !agentConfig.ServiceDiscoveryEnabled {
		if agentConfig.PlatformURL != "" {
			log.Printf("🔧 Using configured API Gateway URL: %s", agentConfig.PlatformURL)
			return agentConfig.PlatformURL, nil
		}
		// Fallback to default
		defaultURL := "http://localhost:6500"
		log.Printf("🔧 Using default API Gateway URL: %s", defaultURL)
		return defaultURL, nil
	}

	// Consul-enabled dynamic discovery
	consulAddr := agentConfig.ConsulAddr
	if consulAddr == "" {
		consulAddr = "localhost:8500"
	}

	// Determine service name from configuration
	serviceName := "openusp-api-gateway"
	if agentConfig.ConsulAPIGatewayServiceName != "" {
		serviceName = agentConfig.ConsulAPIGatewayServiceName
	}

	consulURL := fmt.Sprintf("http://%s/v1/catalog/service/%s", consulAddr, serviceName)
	log.Printf("🔍 Discovering API Gateway '%s' via Consul at %s", serviceName, consulURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(consulURL)
	if err != nil {
		// Fallback to configured static URL if Consul fails
		if agentConfig.PlatformURL != "" {
			log.Printf("⚠️  Consul discovery failed, using configured URL: %s", agentConfig.PlatformURL)
			return agentConfig.PlatformURL, nil
		}
		return "", fmt.Errorf("consul discovery failed and no fallback URL configured: %v", err)
	}
	defer resp.Body.Close()

	var services []ConsulService
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return "", fmt.Errorf("failed to decode Consul response: %v", err)
	}

	if len(services) == 0 {
		// Fallback to configured static URL if no services found
		if agentConfig.PlatformURL != "" {
			log.Printf("⚠️  No API Gateway found in Consul, using configured URL: %s", agentConfig.PlatformURL)
			return agentConfig.PlatformURL, nil
		}
		return "", fmt.Errorf("no API Gateway found in Consul and no fallback URL configured")
	}

	// Use the first available service
	service := services[0]
	address := service.ServiceAddress
	if address == "" || address == "localhost" {
		address = "localhost" // Keep localhost for local development
	}

	apiURL := fmt.Sprintf("http://%s:%d", address, service.ServicePort)
	log.Printf("✅ Discovered API Gateway URL via Consul: %s", apiURL)
	return apiURL, nil
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

	// Discover API Gateway
	apiURL, err := discoverAPIGatewayWithConfig(agentConfig)
	if err != nil {
		log.Printf("⚠️  Failed to discover API Gateway, skipping registration: %v", err)
		return nil // Don't fail if registration is not available
	}

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
		configInterface, err := config.LoadConfigFromYAML("tr369", configDir)
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

	// Determine WebSocket URL using configuration-driven service discovery
	wsURL, err := discoverMTPServiceWithConfig(agentConfig)
	if err != nil {
		log.Fatalf("Failed to discover MTP Service: %v", err)
	}

	// Register device with API Gateway (if enabled and available)
	if err := registerDeviceWithAPI(agentConfig, endpointID); err != nil {
		log.Printf("⚠️  Device registration warning: %v", err)
	}

	// Connect to MTP Service
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
	log.Printf("\nThen run this example with:")
	log.Printf("  go run examples/tr369-agent/main.go")
	log.Printf("  go run examples/tr369-agent/main.go --config configs/tr369-agent.yaml")
	log.Printf("\nThe agent will automatically discover the MTP Service via Consul.")
}
