package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"openusp/pkg/config"
)

// getDeviceIPAddress gets the device's primary IP address
func getDeviceIPAddress() string {
	// Try to get the IP address by connecting to a remote address
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Printf("⚠️  Failed to determine IP address, using localhost: %v", err)
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// connectionRequestURL builds a connection request URL from the unified config
func connectionRequestURL(deviceIP string, cfg *config.TR069Config) string {
	// Prefer explicit URL if provided in YAML; otherwise synthesize from path/port
	if cfg.ConnectionRequestURL != "" {
		return cfg.ConnectionRequestURL
	}
	port := cfg.ConnectionRequestPort
	if port == 0 {
		port = 7547
	}
	path := cfg.ConnectionRequestPath
	if path == "" {
		path = "/connection_request"
	}
	return fmt.Sprintf("http://%s:%d%s", deviceIP, port, path)
}

// DeviceInfo represents device information for CWMP agent
type DeviceInfo struct {
	Manufacturer    string
	ProductClass    string
	SerialNumber    string
	OUI             string
	ModelName       string
	SoftwareVersion string
	HardwareVersion string
}

// mapDeviceInfo converts unified TR069Config fields to local DeviceInfo struct
func mapDeviceInfo(agent *config.TR069Config) *DeviceInfo {
	return &DeviceInfo{
		Manufacturer:    agent.Manufacturer,
		ProductClass:    agent.ProductClass,
		SerialNumber:    agent.SerialNumber,
		OUI:             agent.OUI,
		ModelName:       agent.ModelName,
		SoftwareVersion: agent.SoftwareVersion,
		HardwareVersion: agent.HardwareVersion,
	}
}

// CWMP/SOAP structures for TR-069 communication
type Envelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  *Header  `xml:"Header,omitempty"`
	Body    Body     `xml:"Body"`
}

type Header struct {
	ID string `xml:"http://schemas.xmlsoap.org/ws/2004/08/addressing To,omitempty"`
}

type Body struct {
	Inform         *Inform         `xml:"urn:dslforum-org:cwmp-1-2 Inform,omitempty"`
	InformResponse *InformResponse `xml:"urn:dslforum-org:cwmp-1-2 InformResponse,omitempty"`
}

type Inform struct {
	DeviceId      DeviceIdStruct         `xml:"DeviceId"`
	Event         []EventStruct          `xml:"Event>EventStruct"`
	MaxEnvelopes  int                    `xml:"MaxEnvelopes"`
	CurrentTime   string                 `xml:"CurrentTime"`
	RetryCount    int                    `xml:"RetryCount"`
	ParameterList []ParameterValueStruct `xml:"ParameterList>ParameterValueStruct"`
}

type DeviceIdStruct struct {
	Manufacturer string `xml:"Manufacturer"`
	OUI          string `xml:"OUI"`
	ProductClass string `xml:"ProductClass"`
	SerialNumber string `xml:"SerialNumber"`
}

type EventStruct struct {
	EventCode  string `xml:"EventCode"`
	CommandKey string `xml:"CommandKey"`
}

type ParameterValueStruct struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
	Type  string `xml:"Type,attr"`
}

type InformResponse struct {
	MaxEnvelopes int `xml:"MaxEnvelopes"`
}

// CWMPClient represents a TR-069 CWMP client
type CWMPClient struct {
	acsURL   string
	username string
	password string
	client   *http.Client
}

// NewCWMPClient creates a new CWMP client
func NewCWMPClient(acsURL, username, password string) *CWMPClient {
	return &CWMPClient{
		acsURL:   acsURL,
		username: username,
		password: password,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// SendInform sends an Inform message to the CWMP service
func (c *CWMPClient) SendInform(inform *Inform) error {
	// Create SOAP envelope
	envelope := &Envelope{
		Body: Body{
			Inform: inform,
		},
	}

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SOAP envelope: %v", err)
	}

	// Add XML declaration
	xmlRequest := []byte(xml.Header + string(xmlData))

	// Create HTTP request
	req, err := http.NewRequest("POST", c.acsURL, bytes.NewBuffer(xmlRequest))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "")

	// Add basic auth if credentials provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	log.Printf("🌐 Sending CWMP Inform request to: %s", c.acsURL)
	log.Printf("📄 Request XML:\n%s", string(xmlRequest))

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	log.Printf("📨 CWMP Response Status: %s", resp.Status)
	log.Printf("📄 Response XML:\n%s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CWMP service returned error: %s", resp.Status)
	}

	// Parse response
	var responseEnvelope Envelope
	if err := xml.Unmarshal(respBody, &responseEnvelope); err != nil {
		log.Printf("⚠️  Failed to parse response XML: %v", err)
		// Don't return error if we can't parse - the request might still have succeeded
	}

	log.Printf("✅ CWMP Inform sent successfully!")
	return nil
}

// runAgent runs the production CWMP agent
func runAgent() error {
	log.Printf("OpenUSP CWMP Agent starting (YAML-only config)...")

	// Load unified YAML-only TR-069 agent configuration from agents.yml
	agentCfg, err := config.LoadCWMPAgentUnified("")
	if err != nil {
		return fmt.Errorf("failed to load cwmp agent config from agents.yml: %w", err)
	}
	if agentCfg.ACSURL == "" {
		return fmt.Errorf("ACS URL missing in agents.yml (cwmp_agent.acs.url)")
	}

	client := NewCWMPClient(agentCfg.ACSURL, agentCfg.ACSUsername, agentCfg.ACSPassword)

	deviceInfo := mapDeviceInfo(agentCfg)

	// Validate required device information
	if deviceInfo.Manufacturer == "" || deviceInfo.SerialNumber == "" {
		return fmt.Errorf("manufacturer and serial number are required")
	}

	// Get dynamic device information
	deviceIP := getDeviceIPAddress()
	crURL := connectionRequestURL(deviceIP, agentCfg)

	log.Printf("Device Information:")
	log.Printf("  Manufacturer: %s", deviceInfo.Manufacturer)
	log.Printf("  Product Class: %s", deviceInfo.ProductClass)
	log.Printf("  Serial Number: %s", deviceInfo.SerialNumber)
	log.Printf("  Device IP: %s", deviceIP)

	// Create Inform message for device registration
	inform := &Inform{
		DeviceId: DeviceIdStruct{
			Manufacturer: deviceInfo.Manufacturer,
			OUI:          deviceInfo.OUI,
			ProductClass: deviceInfo.ProductClass,
			SerialNumber: deviceInfo.SerialNumber,
		},
		Event: []EventStruct{
			{
				EventCode:  "0 BOOTSTRAP",
				CommandKey: "",
			},
			{
				EventCode:  "1 BOOT",
				CommandKey: "",
			},
		},
		MaxEnvelopes: 1,
		CurrentTime:  time.Now().Format(time.RFC3339),
		RetryCount:   0,
		ParameterList: []ParameterValueStruct{
			{
				Name:  "Device.DeviceInfo.Manufacturer",
				Value: deviceInfo.Manufacturer,
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.ProductClass",
				Value: deviceInfo.ProductClass,
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.SerialNumber",
				Value: deviceInfo.SerialNumber,
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.ModelName",
				Value: deviceInfo.ModelName,
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.SoftwareVersion",
				Value: deviceInfo.SoftwareVersion,
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.HardwareVersion",
				Value: deviceInfo.HardwareVersion,
				Type:  "xsd:string",
			},
			{
				Name:  "Device.IP.Interface.1.IPv4Address.1.IPAddress",
				Value: deviceIP,
				Type:  "xsd:string",
			},
			{
				Name:  "Device.ManagementServer.ConnectionRequestURL",
				Value: crURL,
				Type:  "xsd:string",
			},
		},
	}

	// Send initial Inform message
	log.Printf("Sending initial Inform message to CWMP service...")
	if err := client.SendInform(inform); err != nil {
		return fmt.Errorf("failed to send Inform message: %w", err)
	}

	log.Printf("CWMP Agent registered successfully")

	// In a production implementation, this would:
	// 1. Handle periodic inform messages
	// 2. Process incoming RPC requests from ACS
	// 3. Handle connection requests
	// 4. Maintain TR-069 session state

	// For now, just wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("CWMP Agent shutting down...")
	return nil
}

func main() {
	log.Printf("OpenUSP CWMP Agent")
	log.Printf("==================")

	if err := runAgent(); err != nil {
		log.Fatalf("Agent error: %v", err)
	}

	log.Printf("CWMP Agent terminated")
}
