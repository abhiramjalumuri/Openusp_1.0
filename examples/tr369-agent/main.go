package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
	defaultUSPVersion = "1.3"
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
	fmt.Println("  -version string")
	fmt.Printf("        USP protocol version (default \"%s\")\n", defaultUSPVersion)
	fmt.Printf("        Supported versions: %s\n", supportedProtocolVersions)
	fmt.Println("  -help")
	fmt.Println("        Show this help information")
	fmt.Println("  -info")
	fmt.Println("        Show agent information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  tr369-agent                    # Use default USP v1.3")
	fmt.Println("  tr369-agent -version 1.4       # Use USP v1.4")
	fmt.Println("  tr369-agent -info              # Show agent information")
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

// discoverMTPService discovers the MTP service WebSocket URL via Consul
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

type USPClient struct {
	conn         *websocket.Conn
	endpointID   string
	controllerID string
	msgID        string
	version      string  // USP protocol version ("1.3" or "1.4")
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
	version := flag.String("version", defaultUSPVersion, "USP protocol version (1.3 or 1.4)")
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

	// Validate version
	if *version != "1.3" && *version != "1.4" {
		log.Fatalf("Unsupported USP version: %s. Supported versions: %s", *version, supportedProtocolVersions)
	}

	log.Printf("OpenUSP TR-369 Client Example")
	log.Printf("=============================")
	log.Printf("This example demonstrates TR-369 USP protocol communication")
	log.Printf("with the OpenUSP platform via WebSocket MTP.")
	log.Printf("USP Protocol Version: %s", *version)
	log.Printf("")

	// Create USP Client with selected version
	client := NewUSPClient(agentEndpointID, controllerID, *version)
	defer client.Close()

	// Discover MTP Service dynamically
	dynamicURL, err := discoverMTPService()
	if err != nil {
		log.Fatalf("Failed to discover MTP Service: %v", err)
	}
	log.Printf("Discovered MTP Service at: %s", dynamicURL)

	// Connect to MTP Service
	if err := client.Connect(dynamicURL); err != nil {
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
	log.Printf("\nThe agent will automatically discover the MTP Service via Consul.")
}
