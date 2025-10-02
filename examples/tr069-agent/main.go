package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// ConsulService represents a service registered in Consul
type ConsulService struct {
	ServiceName    string            `json:"ServiceName"`
	ServiceID      string            `json:"ServiceID"`
	ServiceTags    []string          `json:"ServiceTags"`
	Address        string            `json:"Address"`        // Node address
	Port           int               `json:"Port"`           // Node port
	ServiceAddress string            `json:"ServiceAddress"` // Service address
	ServicePort    int               `json:"ServicePort"`    // Service port
	Meta           map[string]string `json:"Meta"`
}

// discoverCWMPService discovers the CWMP service via Consul
func discoverCWMPService() (string, error) {
	log.Printf("🔍 Discovering CWMP service via Consul...")

	// Get Consul address from environment or use default
	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		consulAddr = "http://localhost:8500"
	}

	// Query Consul for CWMP service
	url := fmt.Sprintf("%s/v1/catalog/service/openusp-cwmp-service", consulAddr)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("⚠️  Failed to query Consul: %v", err)
		log.Printf("🔄 Falling back to default address: http://localhost:7547")
		return "http://localhost:7547", nil
	}
	defer resp.Body.Close()

	var services []ConsulService
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		log.Printf("⚠️  Failed to decode Consul response: %v", err)
		log.Printf("🔄 Falling back to default address: http://localhost:7547")
		return "http://localhost:7547", nil
	}

	if len(services) == 0 {
		log.Printf("⚠️  No CWMP service found in Consul")
		log.Printf("🔄 Falling back to default address: http://localhost:7547")
		return "http://localhost:7547", nil
	}

	// Use the first available service
	service := services[0]
	// Use ServiceAddress and ServicePort for the actual service endpoint
	address := service.ServiceAddress
	if address == "" {
		address = service.Address
	}
	if address == "127.0.0.1" {
		address = "localhost" // Use localhost for better compatibility
	}

	port := service.ServicePort
	if port == 0 {
		port = service.Port
	}

	cwmpServiceURL := fmt.Sprintf("http://%s:%d", address, port)
	log.Printf("✅ Found CWMP service at: %s", cwmpServiceURL)

	return cwmpServiceURL, nil
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
	Inform         *Inform         `xml:"urn:dslforum-org:cwmp-1-0 Inform,omitempty"`
	InformResponse *InformResponse `xml:"urn:dslforum-org:cwmp-1-0 InformResponse,omitempty"`
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

	// Discover CWMP service dynamically
	cwmpServiceURL, err := discoverCWMPService()
	if err != nil {
		log.Fatalf("Failed to discover CWMP service: %v", err)
	}
	log.Printf("Using CWMP service at: %s", cwmpServiceURL)

	// Create CWMP client with basic auth credentials
	username := os.Getenv("CWMP_USERNAME")
	if username == "" {
		username = "acs" // Default username
	}
	password := os.Getenv("CWMP_PASSWORD")
	if password == "" {
		password = "acs123" // Default password
	}

	client := NewCWMPClient(cwmpServiceURL, username, password)

	// Create a sample Inform message that would come from a TR-069 device
	sampleInform := &Inform{
		DeviceId: DeviceIdStruct{
			Manufacturer: "OpenUSP",
			OUI:          "00D4FE",
			ProductClass: "HomeGateway",
			SerialNumber: "DEMO123456",
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
				Value: "OpenUSP",
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.ProductClass",
				Value: "HomeGateway",
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.SerialNumber",
				Value: "DEMO123456",
				Type:  "xsd:string",
			},
			{
				Name:  "Device.IP.Interface.1.IPv4Address.1.IPAddress",
				Value: "192.168.1.100",
				Type:  "xsd:string",
			},
		},
	}

	fmt.Printf("📱 Sample Device Information:\n")
	fmt.Printf("   Manufacturer: %s\n", sampleInform.DeviceId.Manufacturer)
	fmt.Printf("   Product Class: %s\n", sampleInform.DeviceId.ProductClass)
	fmt.Printf("   Serial Number: %s\n", sampleInform.DeviceId.SerialNumber)
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
		fmt.Println("4. Run this TR-069 agent: go run examples/tr069-agent/main.go")
		fmt.Println()
		fmt.Println("The agent will automatically discover services via Consul.")
		return
	}

	fmt.Println()
	fmt.Println("🎉 TR-069 Device Onboarding Complete!")
	fmt.Println("✅ Device successfully sent Inform message to CWMP service")
	fmt.Println("📊 Check the CWMP service logs for onboarding details")
}
