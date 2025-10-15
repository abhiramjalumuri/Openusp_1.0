package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"openusp/pkg/config"
	pb_v1_3 "openusp/pkg/proto/v1_3"
	pb_v1_4 "openusp/pkg/proto/v1_4"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// ============================================================================
// TRANSPORT LAYER
// ============================================================================

// Transport is an abstraction over an MTP transport (WebSocket, STOMP, etc.)
type Transport interface {
	Connect() error
	Send([]byte) error
	Read(timeout time.Duration) ([]byte, error)
	Close() error
	Name() string
}

// ----------------------------------------------------------------------------
// WebSocket Transport
// ----------------------------------------------------------------------------
type WebSocketTransport struct {
	url         string
	subprotocol string
	conn        *websocket.Conn
}

func NewWebSocketTransport(url, subprotocol string) *WebSocketTransport {
	return &WebSocketTransport{url: url, subprotocol: subprotocol}
}

func (w *WebSocketTransport) Name() string { return "websocket" }

func (w *WebSocketTransport) Connect() error {
	log.Printf("Connecting via WebSocket to: %s", w.url)
	var headers http.Header
	if w.subprotocol != "" {
		headers = http.Header{}
		headers.Set("Sec-WebSocket-Protocol", w.subprotocol)
	}
	conn, _, err := websocket.DefaultDialer.Dial(w.url, headers)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	w.conn = conn
	return nil
}

func (w *WebSocketTransport) Send(b []byte) error {
	if w.conn == nil {
		return errors.New("websocket not connected")
	}
	return w.conn.WriteMessage(websocket.BinaryMessage, b)
}

func (w *WebSocketTransport) Read(timeout time.Duration) ([]byte, error) {
	if w.conn == nil {
		return nil, errors.New("websocket not connected")
	}
	_ = w.conn.SetReadDeadline(time.Now().Add(timeout))
	_, msg, err := w.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (w *WebSocketTransport) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// ----------------------------------------------------------------------------
// STOMP Transport
// ----------------------------------------------------------------------------
type StompTransport struct {
	brokerURL     string
	username      string
	password      string
	destSend      string
	destSubscribe string
	hbSend        time.Duration
	hbRecv        time.Duration
	conn          net.Conn
	sessionID     string
	inbox         chan []byte
	errs          chan error
	closeOnce     sync.Once
	closed        chan struct{}
}

func NewStompTransport(cfg *config.TR369Config) *StompTransport {
	// Use config values for STOMP destinations
	sendDest := cfg.STOMPDestinationOutbound
	subDest := cfg.STOMPDestinationInbound
	log.Println("Stomp queue Tx (agent->controller): ", sendDest)
	log.Println("Stomp queue Rx (controller->agent): ", subDest)

	// Log STOMP credentials (mask password)
	username := cfg.STOMPUsername
	password := cfg.STOMPPassword
	if username == "" {
		username = "openusp"
		log.Println("⚠️  STOMP username not in config, using default: openusp")
	} else {
		log.Printf("🔑 STOMP username from config: %s", username)
	}
	if password == "" {
		password = "openusp123"
		log.Println("⚠️  STOMP password not in config, using default: ****")
	} else {
		log.Println("🔑 STOMP password from config: ****")
	}

	return &StompTransport{
		brokerURL:     cfg.STOMPBrokerURL,
		username:      username,
		password:      password,
		destSend:      sendDest,
		destSubscribe: subDest,
		hbSend:        cfg.STOMPHeartbeatSend,
		hbRecv:        cfg.STOMPHeartbeatReceive,
		inbox:         make(chan []byte, 8),
		errs:          make(chan error, 1),
		closed:        make(chan struct{}),
	}
}

func (s *StompTransport) Name() string { return "stomp" }

func (s *StompTransport) parseHostPort() (string, string, error) {
	url := s.brokerURL
	if strings.HasPrefix(url, "stomp://") {
		url = strings.TrimPrefix(url, "stomp://")
	} else if strings.HasPrefix(url, "tcp://") {
		url = strings.TrimPrefix(url, "tcp://")
	}
	host, port, ok := strings.Cut(url, ":")
	if !ok {
		port = "61613"
	}
	if port == "" {
		port = "61613"
	}
	return host, port, nil
}

func (s *StompTransport) Connect() error {
	host, port, _ := s.parseHostPort()
	log.Printf("🔌 STOMP: Connecting to %s:%s", host, port)
	conn, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return fmt.Errorf("stomp dial: %w", err)
	}
	s.conn = conn

	// Username and password should already be set from config in NewStompTransport
	// These are just safety fallbacks
	if s.username == "" {
		s.username = "openusp"
		log.Printf("⚠️  STOMP username empty, using fallback: openusp")
	}
	if s.password == "" {
		s.password = "openusp123"
		log.Printf("⚠️  STOMP password empty, using fallback")
	}

	log.Printf("🔐 STOMP: Authenticating as user: %s", s.username)
	frame := fmt.Sprintf("CONNECT\naccept-version:1.2\nhost:/\nlogin:%s\npasscode:%s", s.username, s.password)
	if s.hbSend > 0 || s.hbRecv > 0 {
		frame += fmt.Sprintf("\nheart-beat:%d,%d", s.hbSend.Milliseconds(), s.hbRecv.Milliseconds())
	}
	frame += "\n\n\x00"
	if _, err = s.conn.Write([]byte(frame)); err != nil {
		return fmt.Errorf("stomp connect write: %w", err)
	}
	// Read CONNECTED
	_ = s.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 2048)
	n, err := s.conn.Read(buf)
	if err != nil {
		return fmt.Errorf("stomp connect read: %w", err)
	}
	resp := string(buf[:n])
	if !strings.HasPrefix(resp, "CONNECTED") {
		return fmt.Errorf("stomp expected CONNECTED, got: %s", resp)
	}
	for _, line := range strings.Split(resp, "\n") {
		if strings.HasPrefix(line, "session:") {
			s.sessionID = strings.TrimPrefix(line, "session:")
			break
		}
	}
	// Subscribe
	sub := fmt.Sprintf("SUBSCRIBE\nid:sub-1\ndestination:%s\nack:auto\n\n\x00", s.destSubscribe)
	if _, err = s.conn.Write([]byte(sub)); err != nil {
		return fmt.Errorf("stomp subscribe: %w", err)
	}
	log.Printf("STOMP subscribed to %s", s.destSubscribe)
	// Start reader & heartbeat goroutines
	go s.readLoop()
	if s.hbSend > 0 {
		go s.heartbeatLoop()
	}
	return nil
}

func (s *StompTransport) heartbeatLoop() {
	ticker := time.NewTicker(s.hbSend)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if _, err := s.conn.Write([]byte("\n")); err != nil {
				s.errs <- fmt.Errorf("stomp heartbeat write: %w", err)
				return
			}
		case <-s.closed:
			return
		}
	}
}

func (s *StompTransport) readLoop() {
	reader := bufio.NewReader(s.conn)
	for {
		s.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		frame, err := readStompFrame(reader)
		if err != nil {
			select {
			case s.errs <- err:
			default:
			}
			// Use sync.Once to prevent panic from closing already-closed channel
			s.closeOnce.Do(func() {
				close(s.closed)
			})
			return
		}
		if frame.command == "MESSAGE" {
			s.inbox <- frame.body
		}
	}
}

type stompFrame struct {
	command string
	headers map[string]string
	body    []byte
}

func readStompFrame(r *bufio.Reader) (*stompFrame, error) {
	// Read command line
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" { // heartbeat newline
		return &stompFrame{command: "HEARTBEAT", headers: map[string]string{}, body: nil}, nil
	}
	f := &stompFrame{command: line, headers: map[string]string{}}
	// Headers
	for {
		h, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		h = strings.TrimRight(h, "\r\n")
		if h == "" {
			break
		}
		if k, v, ok := strings.Cut(h, ":"); ok {
			f.headers[k] = v
		}
	}
	// Body until null 0
	body, err := r.ReadBytes(0x00)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		body = body[:len(body)-1]
	}
	f.body = body
	return f, nil
}

func (s *StompTransport) Send(b []byte) error {
	if s.conn == nil {
		return errors.New("stomp not connected")
	}
	header := fmt.Sprintf("SEND\ndestination:%s\ncontent-type:application/octet-stream\ncontent-length:%d\n\n", s.destSend, len(b))
	frame := make([]byte, len(header)+len(b)+1)
	copy(frame, header)
	copy(frame[len(header):], b)
	frame[len(frame)-1] = 0
	s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := s.conn.Write(frame)
	return err
}

func (s *StompTransport) Read(timeout time.Duration) ([]byte, error) {
	select {
	case msg := <-s.inbox:
		return msg, nil
	case err := <-s.errs:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("stomp read timeout after %s", timeout)
	}
}

func (s *StompTransport) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closed)
		if s.conn != nil {
			disc := fmt.Sprintf("DISCONNECT\nsession:%s\n\n\x00", s.sessionID)
			s.conn.Write([]byte(disc))
			err = s.conn.Close()
		}
	})
	return err
}

// ============================================================================
// USP CLIENT CONFIGURATION
// ============================================================================

const (
	// USP Agent Configuration - now loaded from configuration
	// Default values - should be overridden via configuration
	defaultAgentEndpointID = "" // Must be set via configuration
	// controllerID is now loaded from configuration

	// Default Device Information - should be overridden via configuration
	defaultDeviceManufacturer    = "" // Must be set via configuration
	defaultDeviceModelName       = "" // Must be set via configuration
	defaultDeviceSerialNumber    = "" // Must be set via configuration
	defaultDeviceSoftwareVersion = "1.0.0"
	defaultDeviceHardwareVersion = "1.0"
	defaultDeviceProductClass    = "" // Must be set via configuration

	// Protocol Version Support
	defaultUSPVersion         = "1.3"
	supportedProtocolVersions = "1.3,1.4"
)

// Global configuration variables (loaded in main)
var (
	agentEndpointID    string
	deviceManufacturer string
	deviceModelName    string
	deviceSerialNumber string
	deviceProductClass string
) // getEnvOrDefault returns environment variable value or default if not set
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
	// Load configuration to show info from agents.yml
	agentConfig, _ := config.LoadCWMPAgentUnified("configs/agents.yml")

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

	// Use fixed port - MTP Service configured via openusp.yml
	mtpPort := 8081 // MTP WebSocket port from YAML config
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

// buildTransport selects and constructs the appropriate Transport implementation.
func buildTransport(agentConfig *config.TR369Config) Transport {
	var mtpType string
	if agentConfig != nil && agentConfig.MTPType != "" {
		mtpType = strings.ToLower(agentConfig.MTPType)
	} else if env := os.Getenv("MTP_TYPE"); env != "" {
		mtpType = strings.ToLower(env)
	}
	if mtpType == "stomp" {
		log.Printf("✅ Using STOMP transport")
		return NewStompTransport(agentConfig)
	}
	// Default to websocket
	wsURL := getMTPServiceURLWithConfig(agentConfig)
	var subprotocol string
	if agentConfig != nil && agentConfig.WebSocketSubprotocol != "" {
		subprotocol = agentConfig.WebSocketSubprotocol
	} else {
		subprotocol = "v1.usp"
	}
	log.Printf("✅ Using WebSocket transport (%s)", wsURL)
	return NewWebSocketTransport(wsURL, subprotocol)
}

// getAPIGatewayURL gets the API Gateway URL from configuration
// getAPIGatewayURL deprecated: unified config or environment should provide any needed URLs.

// ----------------------------------------------------------------------------
// USP Client (transport-agnostic)
// ----------------------------------------------------------------------------
type USPClient struct {
	transport    Transport
	endpointID   string
	controllerID string
	msgID        string
	version      string
	agentConfig  *config.TR369Config
}

func NewUSPClient(endpointID, controllerID, version string, tr Transport, agentConfig *config.TR369Config) *USPClient {
	return &USPClient{transport: tr, endpointID: endpointID, controllerID: controllerID, msgID: fmt.Sprintf("msg-%d", time.Now().Unix()), version: version, agentConfig: agentConfig}
}

func (c *USPClient) Connect() error { return c.transport.Connect() }
func (c *USPClient) Close() error   { return c.transport.Close() }

// createOnboardingMessage creates a USP Notify message with Boot! event (TR-369 standard onboarding)
func (c *USPClient) createOnboardingMessage() ([]byte, error) {
	if c.version == "1.4" {
		return c.createBootEventV14()
	}
	return c.createBootEventV13()
}

// createBootEventV13 creates a USP v1.3 Boot! Event Notification (TR-369 standard onboarding)
func (c *USPClient) createBootEventV13() ([]byte, error) {
	// Per TR-369: Agent MUST send Boot! event on startup/reconnect
	// This is the standard onboarding mechanism
	
	// Create Boot! Event with device parameters
	bootEvent := &pb_v1_3.Notify_Event{
		ObjPath:   "Device.Boot!",
		EventName: "Boot!",
		Params: map[string]string{
			"Cause":                 "LocalReboot",
			"CommandKey":            "",
			"FirmwareUpdated":       "false",
			"ParameterMap":          "",
			"SoftwareVersion":       deviceManufacturer + " " + deviceModelName,
			"Manufacturer":          deviceManufacturer,
			"ModelName":             deviceModelName,
			"SerialNumber":          deviceSerialNumber,
			"ProductClass":          deviceProductClass,
		},
	}

	// Create USP Notify Request
	notifyReq := &pb_v1_3.Notify{
		SubscriptionId: "boot-event-subscription",
		SendResp:       true,
		Notification: &pb_v1_3.Notify_Event_{
			Event: bootEvent,
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
			MsgId:   fmt.Sprintf("boot-%d", time.Now().Unix()),
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

// createBootEventV14 creates a USP v1.4 Boot! Event Notification (TR-369 standard onboarding)
func (c *USPClient) createBootEventV14() ([]byte, error) {
	// Per TR-369: Agent MUST send Boot! event on startup/reconnect
	
	// Create Boot! Event with device parameters
	bootEvent := &pb_v1_4.Notify_Event{
		ObjPath:   "Device.Boot!",
		EventName: "Boot!",
		Params: map[string]string{
			"Cause":                 "LocalReboot",
			"CommandKey":            "",
			"FirmwareUpdated":       "false",
			"ParameterMap":          "",
			"SoftwareVersion":       deviceManufacturer + " " + deviceModelName,
			"Manufacturer":          deviceManufacturer,
			"ModelName":             deviceModelName,
			"SerialNumber":          deviceSerialNumber,
			"ProductClass":          deviceProductClass,
		},
	}

	// Create USP Notify Request
	notifyReq := &pb_v1_4.Notify{
		SubscriptionId: "boot-event-subscription",
		SendResp:       true,
		Notification: &pb_v1_4.Notify_Event_{
			Event: bootEvent,
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
	// Parse the record to extract message type for logging
	msgType := extractMessageType(recordBytes, c.version)
	log.Printf("📤 SEND: USP %s message (size: %d bytes, version: %s) via %s", msgType, len(recordBytes), c.version, c.transport.Name())
	
	if err := c.transport.Send(recordBytes); err != nil {
		return fmt.Errorf("send record: %w", err)
	}
	log.Printf("✅ SENT: USP %s message successfully via %s", msgType, c.transport.Name())
	return nil
}

func (c *USPClient) readResponse() error {
	resp, err := c.transport.Read(10 * time.Second)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	
	// Parse the response to extract message type for logging
	msgType := extractMessageType(resp, c.version)
	log.Printf("📥 RECV: USP %s response (%d bytes) via %s", msgType, len(resp), c.transport.Name())
	return nil
}

// extractMessageType parses a USP Record to extract the message type
func extractMessageType(recordBytes []byte, version string) string {
	if version == "1.3" {
		var record pb_v1_3.Record
		if err := proto.Unmarshal(recordBytes, &record); err != nil {
			return "UNKNOWN"
		}
		
		// Extract payload from NoSessionContext
		if noSession := record.GetNoSessionContext(); noSession != nil {
			var msg pb_v1_3.Msg
			if err := proto.Unmarshal(noSession.Payload, &msg); err != nil {
				return "UNKNOWN"
			}
			
			// Get message type from header
			if msg.Header != nil {
				return msg.Header.MsgType.String()
			}
		}
	} else if version == "1.4" {
		var record pb_v1_4.Record
		if err := proto.Unmarshal(recordBytes, &record); err != nil {
			return "UNKNOWN"
		}
		
		// Extract payload from NoSessionContext
		if noSession := record.GetNoSessionContext(); noSession != nil {
			var msg pb_v1_4.Msg
			if err := proto.Unmarshal(noSession.Payload, &msg); err != nil {
				return "UNKNOWN"
			}
			
			// Get message type from header
			if msg.Header != nil {
				return msg.Header.MsgType.String()
			}
		}
	}
	
	return "UNKNOWN"
}

// registerDeviceWithAPI registers the device with the OpenUSP API Gateway

func demonstrateUSPOperations(client *USPClient) error {
	log.Printf("\n🚀 Starting TR-369 USP Client Demonstration")
	log.Printf("=========================================")

	// Step 1: Send Boot! Event (TR-369 Standard Onboarding)
	log.Printf("\n1️⃣  Sending Boot! Event (TR-369 Standard Onboarding)...")
	log.Printf("   Device: %s %s (S/N: %s)", deviceManufacturer, deviceModelName, deviceSerialNumber)
	log.Printf("   Agent Endpoint: %s", client.endpointID)
	log.Printf("   Per TR-369: Agent MUST send Boot! event on startup/reconnect")

	onboardRecord, err := client.createOnboardingMessage()
	if err != nil {
		return fmt.Errorf("failed to create Boot! event: %w", err)
	}

	if err := client.sendRecord(onboardRecord); err != nil {
		return fmt.Errorf("failed to send Boot! event: %w", err)
	}

	log.Printf("Waiting for Boot! event acknowledgment...")
	if err := client.readResponse(); err != nil {
		log.Printf("Error reading Boot! event response: %v", err)
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
	configPath := flag.String("config", "configs/agents.yml", "Path to YAML configuration file (default: configs/agents.yml)")
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
		log.Fatalf("Configuration file is required. Use --config to specify the path to agents.yml")
	}

	// Load from unified YAML file (agents.yml)
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
		deviceProductClass = getConfigValueOrDefault(agentConfig.ProductClass, defaultDeviceProductClass)
	} else {
		agentEndpointID = defaultAgentEndpointID
		deviceManufacturer = defaultDeviceManufacturer
		deviceModelName = defaultDeviceModelName
		deviceSerialNumber = defaultDeviceSerialNumber
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

	// Build transport & client
	transport := buildTransport(agentConfig)
	client := NewUSPClient(endpointID, getControllerID(), *version, transport, agentConfig)
	defer client.Close()
	if err := client.Connect(); err != nil {
		log.Fatalf("❌ Failed to connect via %s: %v", transport.Name(), err)
	}
	log.Printf("✅ Connected via %s transport", transport.Name())

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
