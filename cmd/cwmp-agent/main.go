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
	"strconv"
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

// getConnectionRequestURL generates the connection request URL for this device
func getConnectionRequestURL() string {
	deviceIP := getDeviceIPAddress()

	// Use standard TR-069 connection request port 7547
	// In a real implementation, this could be configurable
	connectionPort := 7547

	// Check if there's an environment variable override
	if portEnv := os.Getenv("CWMP_CONNECTION_REQUEST_PORT"); portEnv != "" {
		if port, err := strconv.Atoi(portEnv); err == nil {
			connectionPort = port
		}
	}

	return fmt.Sprintf("http://%s:%d/connection_request", deviceIP, connectionPort)
}

// getCWMPServiceURL gets the CWMP service URL from configuration
func getCWMPServiceURL() string {
	// Try to load configuration from YAML
	configPath := "configs/cwmp-agent.yaml"
	if _, err := os.Stat(configPath); err == nil {
		tr069Config, err := config.LoadYAMLTR069Config(configPath)
		if err == nil && tr069Config.ACSURL != "" {
			log.Printf("✅ Using CWMP service URL from YAML config: %s", tr069Config.ACSURL)
			return tr069Config.ACSURL
		}
	}

	// Fallback to environment variables or default standard port
	cwmpHost := os.Getenv("CWMP_ACS_HOST")
	if cwmpHost == "" {
		cwmpHost = "localhost"
	}
	cwmpPort := os.Getenv("CWMP_ACS_PORT")
	if cwmpPort == "" {
		cwmpPort = "7547" // Standard TR-069 ACS port
	}
	cwmpServiceURL := fmt.Sprintf("http://%s:%s", cwmpHost, cwmpPort)
	log.Printf("✅ Using CWMP service URL from environment/default: %s", cwmpServiceURL)
	return cwmpServiceURL
}

// getCWMPCredentials gets the CWMP credentials from configuration
func getCWMPCredentials() (string, string) {
	// Try to load credentials from YAML
	configPath := "configs/cwmp-agent.yaml"
	if _, err := os.Stat(configPath); err == nil {
		tr069Config, err := config.LoadYAMLTR069Config(configPath)
		if err == nil {
			if tr069Config.ACSUsername != "" && tr069Config.ACSPassword != "" {
				log.Printf("✅ Using CWMP credentials from YAML config")
				return tr069Config.ACSUsername, tr069Config.ACSPassword
			}
		}
	}

	// Fallback to environment variables or defaults
	username := os.Getenv("CWMP_USERNAME")
	if username == "" {
		username = "acs" // Default username
	}
	password := os.Getenv("CWMP_PASSWORD")
	if password == "" {
		password = "acs123" // Default password
	}
	log.Printf("✅ Using CWMP credentials from environment/default")
	return username, password
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

// getDeviceInfo gets device information from YAML configuration
func getDeviceInfo() *DeviceInfo {
	// Try to load device info from YAML
	configPath := "configs/cwmp-agent.yaml"
	if _, err := os.Stat(configPath); err == nil {
		tr069Config, err := config.LoadYAMLTR069Config(configPath)
		if err == nil {
			log.Printf("✅ Using device information from YAML config")
			return &DeviceInfo{
				Manufacturer:    tr069Config.Manufacturer,
				ProductClass:    tr069Config.ProductClass,
				SerialNumber:    tr069Config.SerialNumber,
				OUI:             tr069Config.OUI,
				ModelName:       tr069Config.ModelName,
				SoftwareVersion: tr069Config.SoftwareVersion,
				HardwareVersion: tr069Config.HardwareVersion,
			}
		}
	}

	// Fallback to environment variables or defaults
	log.Printf("✅ Using device information from environment/default")
	return &DeviceInfo{
		Manufacturer:    getEnvOrDefault("CWMP_MANUFACTURER", "OpenUSP"),
		ProductClass:    getEnvOrDefault("CWMP_PRODUCT_CLASS", "HomeGateway"),
		SerialNumber:    getEnvOrDefault("CWMP_SERIAL_NUMBER", "DEMO123456"),
		OUI:             getEnvOrDefault("CWMP_OUI", "00D4FE"),
		ModelName:       getEnvOrDefault("CWMP_MODEL_NAME", "CWMP-Gateway-v1"),
		SoftwareVersion: getEnvOrDefault("CWMP_SOFTWARE_VERSION", "1.0.0"),
		HardwareVersion: getEnvOrDefault("CWMP_HARDWARE_VERSION", "1.0"),
	}
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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

// TR069OnboardingDemo demonstrates the TR-069 onboarding functionality
func main() {
	fmt.Println("🚀 TR-069 Device Onboarding Demo")
	fmt.Println("================================")

	// Get CWMP service URL from YAML configuration or environment
	cwmpServiceURL := getCWMPServiceURL()

	// Get CWMP credentials from YAML configuration or environment
	username, password := getCWMPCredentials()

	client := NewCWMPClient(cwmpServiceURL, username, password)

	// Get device information from YAML configuration or environment
	deviceInfo := getDeviceInfo()

	// Get dynamic device information
	deviceIP := getDeviceIPAddress()
	connectionRequestURL := getConnectionRequestURL()

	// Create a sample Inform message that would come from a TR-069 device
	sampleInform := &Inform{
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
				Value: connectionRequestURL,
				Type:  "xsd:string",
			},
		},
	}

	fmt.Printf("📱 Sample Device Information:\n")
	fmt.Printf("   Manufacturer: %s\n", sampleInform.DeviceId.Manufacturer)
	fmt.Printf("   Product Class: %s\n", sampleInform.DeviceId.ProductClass)
	fmt.Printf("   Serial Number: %s\n", sampleInform.DeviceId.SerialNumber)
	fmt.Printf("   Device IP: %s\n", deviceIP)
	fmt.Printf("   Connection Request URL: %s\n", connectionRequestURL)
	fmt.Printf("   Parameters: %d\n", len(sampleInform.ParameterList))
	fmt.Println()

	// Send the Inform message to CWMP service for onboarding
	fmt.Println("🔄 Starting TR-069 onboarding process...")
	if err := client.SendInform(sampleInform); err != nil {
		fmt.Printf("❌ TR-069 onboarding failed: %v\n", err)
		fmt.Println()
		fmt.Println("To see full onboarding in action:")
		fmt.Println("1. Start infrastructure: make infra-up")
		fmt.Println("2. Build services: make build-all")
		fmt.Println("3. Start services: make start-all")
		fmt.Println("4. Run this TR-069 agent: go run cmd/cwmp-agent/main.go")
		fmt.Println()
		fmt.Println("The agent will connect to CWMP service using standard TR-069 port 7547.")
		return
	}

	fmt.Println()
	fmt.Println("🎉 TR-069 Device Onboarding Complete!")
	fmt.Println("✅ Device successfully sent Inform message to CWMP service")
	fmt.Println("📊 Check the CWMP service logs for onboarding details")
}
