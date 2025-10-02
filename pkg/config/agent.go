package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// AgentConfig represents the base configuration for protocol agents
type AgentConfig struct {
	// Service Discovery
	ConsulEnabled    bool   `json:"consul_enabled"`
	ConsulAddr       string `json:"consul_addr"`
	ConsulDatacenter string `json:"consul_datacenter"`

	// Device Information
	EndpointID       string `json:"endpoint_id"`
	ProductClass     string `json:"product_class"`
	Manufacturer     string `json:"manufacturer"`
	ModelName        string `json:"model_name"`
	SerialNumber     string `json:"serial_number"`
	SoftwareVersion  string `json:"software_version"`
	HardwareVersion  string `json:"hardware_version"`
	DeviceType       string `json:"device_type"`
	OUI              string `json:"oui"`

	// Platform Integration
	PlatformURL            string        `json:"platform_url"`
	APIVersion             string        `json:"api_version"`
	RegistrationEndpoint   string        `json:"registration_endpoint"`
	HeartbeatInterval      time.Duration `json:"heartbeat_interval"`
	AutoRegister           bool          `json:"auto_register"`

	// Consul Integration
	ServiceDiscoveryEnabled bool          `json:"service_discovery_enabled"`
	HealthCheckInterval     time.Duration `json:"health_check_interval"`
	ServiceTags             []string      `json:"service_tags"`

	// Security
	TLSEnabled   bool   `json:"tls_enabled"`
	TLSCertFile  string `json:"tls_cert_file"`
	TLSKeyFile   string `json:"tls_key_file"`
	TLSCAFile    string `json:"tls_ca_file"`
	TLSSkipVerify bool  `json:"tls_skip_verify"`

	// Logging
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`
	LogOutput string `json:"log_output"`
	LogFile   string `json:"log_file"`

	// Performance
	BufferSize        int           `json:"buffer_size"`
	MaxMessageSize    int           `json:"max_message_size"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	ReadTimeout       time.Duration `json:"read_timeout"`
	WriteTimeout      time.Duration `json:"write_timeout"`
	IdleTimeout       time.Duration `json:"idle_timeout"`

	// Development
	DevelopmentMode bool `json:"development_mode"`
	MockDataEnabled bool `json:"mock_data_enabled"`
	TestMode        bool `json:"test_mode"`
	VerboseLogging  bool `json:"verbose_logging"`
}

// TR369Config represents TR-369 USP agent specific configuration
type TR369Config struct {
	AgentConfig

	// USP Protocol
	USPVersion          string   `json:"usp_version"`
	USPSupportedVersions []string `json:"usp_supported_versions"`
	USPAgentRole        string   `json:"usp_agent_role"`
	USPCommandKey       string   `json:"usp_command_key"`

	// MTP Configuration
	MTPType string `json:"mtp_type"`

	// WebSocket MTP
	WebSocketURL               string        `json:"websocket_url"`
	WebSocketSubprotocol       string        `json:"websocket_subprotocol"`
	WebSocketPingInterval      time.Duration `json:"websocket_ping_interval"`
	WebSocketPongTimeout       time.Duration `json:"websocket_pong_timeout"`
	WebSocketReconnectInterval time.Duration `json:"websocket_reconnect_interval"`
	WebSocketMaxReconnectAttempts int        `json:"websocket_max_reconnect_attempts"`

	// MQTT MTP
	MQTTBrokerURL     string        `json:"mqtt_broker_url"`
	MQTTClientID      string        `json:"mqtt_client_id"`
	MQTTUsername      string        `json:"mqtt_username"`
	MQTTPassword      string        `json:"mqtt_password"`
	MQTTTopicRequest  string        `json:"mqtt_topic_request"`
	MQTTTopicResponse string        `json:"mqtt_topic_response"`
	MQTTQOS           int           `json:"mqtt_qos"`
	MQTTRetain        bool          `json:"mqtt_retain"`
	MQTTCleanSession  bool          `json:"mqtt_clean_session"`
	MQTTKeepAlive     time.Duration `json:"mqtt_keep_alive"`

	// STOMP MTP
	STOMPBrokerURL            string        `json:"stomp_broker_url"`
	STOMPUsername             string        `json:"stomp_username"`
	STOMPPassword             string        `json:"stomp_password"`
	STOMPDestinationRequest   string        `json:"stomp_destination_request"`
	STOMPDestinationResponse  string        `json:"stomp_destination_response"`
	STOMPHeartbeatSend        time.Duration `json:"stomp_heartbeat_send"`
	STOMPHeartbeatReceive     time.Duration `json:"stomp_heartbeat_receive"`

	// Unix Domain Socket MTP
	UDSSocketPath string `json:"uds_socket_path"`
	UDSSocketMode string `json:"uds_socket_mode"`

	// Agent Behavior
	PeriodicInformEnabled         bool          `json:"periodic_inform_enabled"`
	PeriodicInformInterval        time.Duration `json:"periodic_inform_interval"`
	ParameterUpdateNotification   bool          `json:"parameter_update_notification"`
	EventNotification             bool          `json:"event_notification"`
	DiagnosticMode                bool          `json:"diagnostic_mode"`

	// Consul Service Names
	ConsulMTPServiceName        string `json:"consul_mtp_service_name"`
	ConsulAPIGatewayServiceName string `json:"consul_api_gateway_service_name"`
}

// TR069Config represents TR-069 CWMP agent specific configuration
type TR069Config struct {
	AgentConfig

	// CWMP Protocol
	CWMPVersion          string   `json:"cwmp_version"`
	CWMPSupportedVersions []string `json:"cwmp_supported_versions"`
	CWMPAgentRole        string   `json:"cwmp_agent_role"`
	CWMPConnectionRequestAuth string `json:"cwmp_connection_request_auth"`

	// ACS Configuration
	ACSURL                   string        `json:"acs_url"`
	ACSUsername              string        `json:"acs_username"`
	ACSPassword              string        `json:"acs_password"`
	ACSPeriodicInformEnabled bool          `json:"acs_periodic_inform_enabled"`
	ACSPeriodicInformInterval time.Duration `json:"acs_periodic_inform_interval"`
	ACSParameterKey          string        `json:"acs_parameter_key"`

	// Connection Request
	ConnectionRequestURL      string `json:"connection_request_url"`
	ConnectionRequestUsername string `json:"connection_request_username"`
	ConnectionRequestPassword string `json:"connection_request_password"`
	ConnectionRequestPath     string `json:"connection_request_path"`
	ConnectionRequestPort     int    `json:"connection_request_port"`

	// SOAP/XML Configuration
	SOAPVersion             string `json:"soap_version"`
	SOAPNamespace           string `json:"soap_namespace"`
	XMLStrictParsing        bool   `json:"xml_strict_parsing"`
	XMLPrettyPrint          bool   `json:"xml_pretty_print"`
	EnvelopeNamespacePrefix string `json:"envelope_namespace_prefix"`

	// HTTP Configuration
	HTTPTimeout                   time.Duration `json:"http_timeout"`
	HTTPKeepAlive                 bool          `json:"http_keep_alive"`
	HTTPMaxIdleConnections        int           `json:"http_max_idle_connections"`
	HTTPIdleTimeout               time.Duration `json:"http_idle_timeout"`
	HTTPTLSHandshakeTimeout       time.Duration `json:"http_tls_handshake_timeout"`
	HTTPExpectContinueTimeout     time.Duration `json:"http_expect_continue_timeout"`

	// Session Management
	SessionTimeout        time.Duration `json:"session_timeout"`
	SessionMaxConcurrent  int           `json:"session_max_concurrent"`
	SessionKeepAlive      bool          `json:"session_keep_alive"`
	SessionCookieEnabled  bool          `json:"session_cookie_enabled"`

	// TR-181 Data Model
	TR181RootObject           string `json:"tr181_root_object"`
	TR181DeviceType           string `json:"tr181_device_type"`
	TR181SpecVersion          string `json:"tr181_spec_version"`
	TR181ParameterNotifications bool `json:"tr181_parameter_notifications"`
	TR181ObjectNotifications    bool `json:"tr181_object_notifications"`

	// Authentication
	DigestAuthEnabled  bool          `json:"digest_auth_enabled"`
	BasicAuthEnabled   bool          `json:"basic_auth_enabled"`
	AuthRealm         string        `json:"auth_realm"`
	AuthAlgorithm     string        `json:"auth_algorithm"`
	AuthQOP           string        `json:"auth_qop"`
	AuthNonceTimeout  time.Duration `json:"auth_nonce_timeout"`

	// Agent Behavior
	InformOnBoot              bool          `json:"inform_on_boot"`
	InformOnPeriodic          bool          `json:"inform_on_periodic"`
	InformOnValueChange       bool          `json:"inform_on_value_change"`
	InformOnConnectionRequest bool          `json:"inform_on_connection_request"`
	MaxEnvelopes              int           `json:"max_envelopes"`
	RetryMinWaitInterval      time.Duration `json:"retry_min_wait_interval"`
	RetryIntervalMultiplier   int           `json:"retry_interval_multiplier"`

	// RPC Methods
	RPCGetRPCMethods          bool `json:"rpc_get_rpc_methods"`
	RPCSetParameterValues     bool `json:"rpc_set_parameter_values"`
	RPCGetParameterValues     bool `json:"rpc_get_parameter_values"`
	RPCGetParameterNames      bool `json:"rpc_get_parameter_names"`
	RPCSetParameterAttributes bool `json:"rpc_set_parameter_attributes"`
	RPCGetParameterAttributes bool `json:"rpc_get_parameter_attributes"`
	RPCAddObject              bool `json:"rpc_add_object"`
	RPCDeleteObject           bool `json:"rpc_delete_object"`
	RPCReboot                 bool `json:"rpc_reboot"`
	RPCFactoryReset           bool `json:"rpc_factory_reset"`
	RPCDownload               bool `json:"rpc_download"`
	RPCUpload                 bool `json:"rpc_upload"`

	// Events
	EventBootstrap                    bool `json:"event_bootstrap"`
	EventBoot                         bool `json:"event_boot"`
	EventPeriodic                     bool `json:"event_periodic"`
	EventValueChange                  bool `json:"event_value_change"`
	EventConnectionRequest            bool `json:"event_connection_request"`
	EventTransferComplete             bool `json:"event_transfer_complete"`
	EventDiagnosticsComplete          bool `json:"event_diagnostics_complete"`
	EventRequestDownload              bool `json:"event_request_download"`
	EventAutonomousTransferComplete   bool `json:"event_autonomous_transfer_complete"`

	// Fault Management
	FaultToleranceEnabled   bool          `json:"fault_tolerance_enabled"`
	FaultMaxRetryCount      int           `json:"fault_max_retry_count"`
	FaultRetryInterval      time.Duration `json:"fault_retry_interval"`
	FaultDetailedReporting  bool          `json:"fault_detailed_reporting"`

	// Consul Service Names
	ConsulCWMPServiceName       string `json:"consul_cwmp_service_name"`
	ConsulAPIGatewayServiceName string `json:"consul_api_gateway_service_name"`

	// CWMP Service
	CWMPServiceEndpoint string `json:"cwmp_service_endpoint"`

	// Logging
	LogSOAPMessages   bool `json:"log_soap_messages"`
	LogHTTPRequests   bool `json:"log_http_requests"`

	// Simulation
	SimulateDeviceResponses bool `json:"simulate_device_responses"`
}

// LoadTR369Config loads TR-369 agent configuration from environment file
func LoadTR369Config(envFile string) (*TR369Config, error) {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return nil, fmt.Errorf("failed to load environment file %s: %w", envFile, err)
		}
	}

	config := &TR369Config{}
	
	// Load base agent configuration
	loadAgentConfig(&config.AgentConfig)

	// USP Protocol
	config.USPVersion = getEnvStringLocal("USP_VERSION", "1.4")
	config.USPSupportedVersions = strings.Split(getEnvStringLocal("USP_SUPPORTED_VERSIONS", "1.3,1.4"), ",")
	config.USPAgentRole = getEnvStringLocal("USP_AGENT_ROLE", "device")
	config.USPCommandKey = getEnvStringLocal("USP_COMMAND_KEY", "usp-agent-001")

	// MTP Configuration
	config.MTPType = getEnvStringLocal("MTP_TYPE", "websocket")

	// WebSocket MTP
	config.WebSocketURL = getEnvStringLocal("WEBSOCKET_URL", "ws://localhost:8081/ws")
	config.WebSocketSubprotocol = getEnvStringLocal("WEBSOCKET_SUBPROTOCOL", "v1.usp")
	config.WebSocketPingInterval = getEnvDurationLocal("WEBSOCKET_PING_INTERVAL", 30*time.Second)
	config.WebSocketPongTimeout = getEnvDurationLocal("WEBSOCKET_PONG_TIMEOUT", 10*time.Second)
	config.WebSocketReconnectInterval = getEnvDurationLocal("WEBSOCKET_RECONNECT_INTERVAL", 5*time.Second)
	config.WebSocketMaxReconnectAttempts = getEnvIntLocal("WEBSOCKET_MAX_RECONNECT_ATTEMPTS", 10)

	// MQTT MTP
	config.MQTTBrokerURL = getEnvStringLocal("MQTT_BROKER_URL", "tcp://localhost:1883")
	config.MQTTClientID = getEnvStringLocal("MQTT_CLIENT_ID", "usp-agent-001")
	config.MQTTUsername = getEnvStringLocal("MQTT_USERNAME", "usp")
	config.MQTTPassword = getEnvStringLocal("MQTT_PASSWORD", "usp123")
	config.MQTTTopicRequest = getEnvStringLocal("MQTT_TOPIC_REQUEST", "usp/agent/request")
	config.MQTTTopicResponse = getEnvStringLocal("MQTT_TOPIC_RESPONSE", "usp/agent/response")
	config.MQTTQOS = getEnvIntLocal("MQTT_QOS", 1)
	config.MQTTRetain = getEnvBoolLocal("MQTT_RETAIN", false)
	config.MQTTCleanSession = getEnvBoolLocal("MQTT_CLEAN_SESSION", true)
	config.MQTTKeepAlive = getEnvDurationLocal("MQTT_KEEP_ALIVE", 60*time.Second)

	// STOMP MTP
	config.STOMPBrokerURL = getEnvStringLocal("STOMP_BROKER_URL", "tcp://localhost:61613")
	config.STOMPUsername = getEnvStringLocal("STOMP_USERNAME", "usp")
	config.STOMPPassword = getEnvStringLocal("STOMP_PASSWORD", "usp123")
	config.STOMPDestinationRequest = getEnvStringLocal("STOMP_DESTINATION_REQUEST", "/queue/usp.agent.request")
	config.STOMPDestinationResponse = getEnvStringLocal("STOMP_DESTINATION_RESPONSE", "/queue/usp.agent.response")
	config.STOMPHeartbeatSend = getEnvDurationLocal("STOMP_HEARTBEAT_SEND", 30*time.Second)
	config.STOMPHeartbeatReceive = getEnvDurationLocal("STOMP_HEARTBEAT_RECEIVE", 30*time.Second)

	// Unix Domain Socket MTP
	config.UDSSocketPath = getEnvStringLocal("UDS_SOCKET_PATH", "/tmp/usp-agent.sock")
	config.UDSSocketMode = getEnvStringLocal("UDS_SOCKET_MODE", "0666")

	// Agent Behavior
	config.PeriodicInformEnabled = getEnvBoolLocal("AGENT_PERIODIC_INFORM_ENABLED", true)
	config.PeriodicInformInterval = getEnvDurationLocal("AGENT_PERIODIC_INFORM_INTERVAL", 300*time.Second)
	config.ParameterUpdateNotification = getEnvBoolLocal("AGENT_PARAMETER_UPDATE_NOTIFICATION", true)
	config.EventNotification = getEnvBoolLocal("AGENT_EVENT_NOTIFICATION", true)
	config.DiagnosticMode = getEnvBoolLocal("AGENT_DIAGNOSTIC_MODE", false)

	// Consul Service Names
	config.ConsulMTPServiceName = getEnvStringLocal("CONSUL_MTP_SERVICE_NAME", "openusp-mtp-service")
	config.ConsulAPIGatewayServiceName = getEnvStringLocal("CONSUL_API_GATEWAY_SERVICE_NAME", "openusp-api-gateway")

	return config, nil
}

// LoadTR069Config loads TR-069 agent configuration from environment file
func LoadTR069Config(envFile string) (*TR069Config, error) {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return nil, fmt.Errorf("failed to load environment file %s: %w", envFile, err)
		}
	}

	config := &TR069Config{}
	
	// Load base agent configuration
	loadAgentConfig(&config.AgentConfig)

	// CWMP Protocol
	config.CWMPVersion = getEnvStringLocal("CWMP_VERSION", "1.4")
	config.CWMPSupportedVersions = strings.Split(getEnvStringLocal("CWMP_SUPPORTED_VERSIONS", "1.0,1.1,1.2,1.3,1.4"), ",")
	config.CWMPAgentRole = getEnvStringLocal("CWMP_AGENT_ROLE", "cpe")
	config.CWMPConnectionRequestAuth = getEnvStringLocal("CWMP_CONNECTION_REQUEST_AUTH", "digest")

	// ACS Configuration
	config.ACSURL = getEnvStringLocal("ACS_URL", "http://localhost:7547")
	config.ACSUsername = getEnvStringLocal("ACS_USERNAME", "acs")
	config.ACSPassword = getEnvStringLocal("ACS_PASSWORD", "acs123")
	config.ACSPeriodicInformEnabled = getEnvBoolLocal("ACS_PERIODIC_INFORM_ENABLED", true)
	config.ACSPeriodicInformInterval = getEnvDurationLocal("ACS_PERIODIC_INFORM_INTERVAL", 300*time.Second)
	config.ACSParameterKey = getEnvStringLocal("ACS_PARAMETER_KEY", "cwmp-agent-001")

	// Connection Request
	config.ConnectionRequestURL = getEnvStringLocal("CONNECTION_REQUEST_URL", "http://localhost:9999/cr")
	config.ConnectionRequestUsername = getEnvStringLocal("CONNECTION_REQUEST_USERNAME", "cr")
	config.ConnectionRequestPassword = getEnvStringLocal("CONNECTION_REQUEST_PASSWORD", "cr123")
	config.ConnectionRequestPath = getEnvStringLocal("CONNECTION_REQUEST_PATH", "/cr")
	config.ConnectionRequestPort = getEnvIntLocal("CONNECTION_REQUEST_PORT", 9999)

	// SOAP/XML Configuration
	config.SOAPVersion = getEnvStringLocal("SOAP_VERSION", "1.1")
	config.SOAPNamespace = getEnvStringLocal("SOAP_NAMESPACE", "urn:dslforum-org:cwmp-1-4")
	config.XMLStrictParsing = getEnvBoolLocal("XML_STRICT_PARSING", false)
	config.XMLPrettyPrint = getEnvBoolLocal("XML_PRETTY_PRINT", false)
	config.EnvelopeNamespacePrefix = getEnvStringLocal("ENVELOPE_NAMESPACE_PREFIX", "soap")

	// HTTP Configuration
	config.HTTPTimeout = getEnvDurationLocal("HTTP_TIMEOUT", 60*time.Second)
	config.HTTPKeepAlive = getEnvBoolLocal("HTTP_KEEP_ALIVE", true)
	config.HTTPMaxIdleConnections = getEnvIntLocal("HTTP_MAX_IDLE_CONNECTIONS", 10)
	config.HTTPIdleTimeout = getEnvDurationLocal("HTTP_IDLE_TIMEOUT", 90*time.Second)
	config.HTTPTLSHandshakeTimeout = getEnvDurationLocal("HTTP_TLS_HANDSHAKE_TIMEOUT", 10*time.Second)
	config.HTTPExpectContinueTimeout = getEnvDurationLocal("HTTP_EXPECT_CONTINUE_TIMEOUT", 1*time.Second)

	// Session Management
	config.SessionTimeout = getEnvDurationLocal("SESSION_TIMEOUT", 300*time.Second)
	config.SessionMaxConcurrent = getEnvIntLocal("SESSION_MAX_CONCURRENT", 5)
	config.SessionKeepAlive = getEnvBoolLocal("SESSION_KEEP_ALIVE", true)
	config.SessionCookieEnabled = getEnvBoolLocal("SESSION_COOKIE_ENABLED", false)

	// TR-181 Data Model
	config.TR181RootObject = getEnvStringLocal("TR181_ROOT_OBJECT", "Device.")
	config.TR181DeviceType = getEnvStringLocal("TR181_DEVICE_TYPE", "InternetGatewayDevice")
	config.TR181SpecVersion = getEnvStringLocal("TR181_SPEC_VERSION", "2.19.1")
	config.TR181ParameterNotifications = getEnvBoolLocal("TR181_PARAMETER_NOTIFICATIONS", true)
	config.TR181ObjectNotifications = getEnvBoolLocal("TR181_OBJECT_NOTIFICATIONS", true)

	// Authentication
	config.DigestAuthEnabled = getEnvBoolLocal("DIGEST_AUTH_ENABLED", true)
	config.BasicAuthEnabled = getEnvBoolLocal("BASIC_AUTH_ENABLED", true)
	config.AuthRealm = getEnvStringLocal("AUTH_REALM", "CWMP")
	config.AuthAlgorithm = getEnvStringLocal("AUTH_ALGORITHM", "MD5")
	config.AuthQOP = getEnvStringLocal("AUTH_QOP", "auth")
	config.AuthNonceTimeout = getEnvDurationLocal("AUTH_NONCE_TIMEOUT", 300*time.Second)

	// Agent Behavior
	config.InformOnBoot = getEnvBoolLocal("AGENT_INFORM_ON_BOOT", true)
	config.InformOnPeriodic = getEnvBoolLocal("AGENT_INFORM_ON_PERIODIC", true)
	config.InformOnValueChange = getEnvBoolLocal("AGENT_INFORM_ON_VALUE_CHANGE", true)
	config.InformOnConnectionRequest = getEnvBoolLocal("AGENT_INFORM_ON_CONNECTION_REQUEST", true)
	config.MaxEnvelopes = getEnvIntLocal("AGENT_MAX_ENVELOPES", 1)
	config.RetryMinWaitInterval = getEnvDurationLocal("AGENT_RETRY_MIN_WAIT_INTERVAL", 5*time.Second)
	config.RetryIntervalMultiplier = getEnvIntLocal("AGENT_RETRY_INTERVAL_MULTIPLIER", 2000)

	// RPC Methods
	config.RPCGetRPCMethods = getEnvBoolLocal("RPC_GET_RPC_METHODS", true)
	config.RPCSetParameterValues = getEnvBoolLocal("RPC_SET_PARAMETER_VALUES", true)
	config.RPCGetParameterValues = getEnvBoolLocal("RPC_GET_PARAMETER_VALUES", true)
	config.RPCGetParameterNames = getEnvBoolLocal("RPC_GET_PARAMETER_NAMES", true)
	config.RPCSetParameterAttributes = getEnvBoolLocal("RPC_SET_PARAMETER_ATTRIBUTES", true)
	config.RPCGetParameterAttributes = getEnvBoolLocal("RPC_GET_PARAMETER_ATTRIBUTES", true)
	config.RPCAddObject = getEnvBoolLocal("RPC_ADD_OBJECT", true)
	config.RPCDeleteObject = getEnvBoolLocal("RPC_DELETE_OBJECT", true)
	config.RPCReboot = getEnvBoolLocal("RPC_REBOOT", true)
	config.RPCFactoryReset = getEnvBoolLocal("RPC_FACTORY_RESET", false)
	config.RPCDownload = getEnvBoolLocal("RPC_DOWNLOAD", true)
	config.RPCUpload = getEnvBoolLocal("RPC_UPLOAD", false)

	// Events
	config.EventBootstrap = getEnvBoolLocal("EVENT_BOOTSTRAP", true)
	config.EventBoot = getEnvBoolLocal("EVENT_BOOT", true)
	config.EventPeriodic = getEnvBoolLocal("EVENT_PERIODIC", true)
	config.EventValueChange = getEnvBoolLocal("EVENT_VALUE_CHANGE", true)
	config.EventConnectionRequest = getEnvBoolLocal("EVENT_CONNECTION_REQUEST", true)
	config.EventTransferComplete = getEnvBoolLocal("EVENT_TRANSFER_COMPLETE", true)
	config.EventDiagnosticsComplete = getEnvBoolLocal("EVENT_DIAGNOSTICS_COMPLETE", false)
	config.EventRequestDownload = getEnvBoolLocal("EVENT_REQUEST_DOWNLOAD", false)
	config.EventAutonomousTransferComplete = getEnvBoolLocal("EVENT_AUTONOMOUS_TRANSFER_COMPLETE", false)

	// Fault Management
	config.FaultToleranceEnabled = getEnvBoolLocal("FAULT_TOLERANCE_ENABLED", true)
	config.FaultMaxRetryCount = getEnvIntLocal("FAULT_MAX_RETRY_COUNT", 3)
	config.FaultRetryInterval = getEnvDurationLocal("FAULT_RETRY_INTERVAL", 10*time.Second)
	config.FaultDetailedReporting = getEnvBoolLocal("FAULT_DETAILED_REPORTING", true)

	// Consul Service Names
	config.ConsulCWMPServiceName = getEnvStringLocal("CONSUL_CWMP_SERVICE_NAME", "openusp-cwmp-service")
	config.ConsulAPIGatewayServiceName = getEnvStringLocal("CONSUL_API_GATEWAY_SERVICE_NAME", "openusp-api-gateway")

	// CWMP Service
	config.CWMPServiceEndpoint = getEnvStringLocal("OPENUSP_CWMP_SERVICE_ENDPOINT", "/cwmp")

	// Logging
	config.LogSOAPMessages = getEnvBoolLocal("LOG_SOAP_MESSAGES", false)
	config.LogHTTPRequests = getEnvBoolLocal("LOG_HTTP_REQUESTS", false)

	// Simulation
	config.SimulateDeviceResponses = getEnvBoolLocal("SIMULATE_DEVICE_RESPONSES", false)

	return config, nil
}

// loadAgentConfig loads the base agent configuration common to both protocols
func loadAgentConfig(config *AgentConfig) {
	// Service Discovery
	config.ConsulEnabled = getEnvBoolLocal("CONSUL_ENABLED", true)
	config.ConsulAddr = getEnvStringLocal("CONSUL_ADDR", "localhost:8500")
	config.ConsulDatacenter = getEnvStringLocal("CONSUL_DATACENTER", "openusp-dev")

	// Device Information (TR369 or TR069 prefix will be used by specific configs)
	config.EndpointID = getEnvStringLocal("ENDPOINT_ID", "")
	config.ProductClass = getEnvStringLocal("PRODUCT_CLASS", "")
	config.Manufacturer = getEnvStringLocal("MANUFACTURER", "")
	config.ModelName = getEnvStringLocal("MODEL_NAME", "")
	config.SerialNumber = getEnvStringLocal("SERIAL_NUMBER", "")
	config.SoftwareVersion = getEnvStringLocal("SOFTWARE_VERSION", "1.1.0")
	config.HardwareVersion = getEnvStringLocal("HARDWARE_VERSION", "1.0")
	config.DeviceType = getEnvStringLocal("DEVICE_TYPE", "gateway")
	config.OUI = getEnvStringLocal("OUI", "00D04F")

	// Platform Integration
	config.PlatformURL = getEnvStringLocal("OPENUSP_PLATFORM_URL", "http://localhost:6500")
	config.APIVersion = getEnvStringLocal("OPENUSP_API_VERSION", "v1")
	config.RegistrationEndpoint = getEnvStringLocal("OPENUSP_REGISTRATION_ENDPOINT", "/api/v1/devices")
	config.HeartbeatInterval = getEnvDurationLocal("OPENUSP_HEARTBEAT_INTERVAL", 60*time.Second)
	config.AutoRegister = getEnvBoolLocal("AGENT_AUTO_REGISTER", true)

	// Consul Integration
	config.ServiceDiscoveryEnabled = getEnvBoolLocal("CONSUL_SERVICE_DISCOVERY_ENABLED", true)
	config.HealthCheckInterval = getEnvDurationLocal("CONSUL_HEALTH_CHECK_INTERVAL", 10*time.Second)
	config.ServiceTags = strings.Split(getEnvStringLocal("CONSUL_SERVICE_TAGS", "agent,v1.1.0"), ",")

	// Security
	config.TLSEnabled = getEnvBoolLocal("TLS_ENABLED", false)
	config.TLSCertFile = getEnvStringLocal("TLS_CERT_FILE", "")
	config.TLSKeyFile = getEnvStringLocal("TLS_KEY_FILE", "")
	config.TLSCAFile = getEnvStringLocal("TLS_CA_FILE", "")
	config.TLSSkipVerify = getEnvBoolLocal("TLS_SKIP_VERIFY", true)

	// Logging
	config.LogLevel = getEnvStringLocal("LOG_LEVEL", "info")
	config.LogFormat = getEnvStringLocal("LOG_FORMAT", "json")
	config.LogOutput = getEnvStringLocal("LOG_OUTPUT", "stdout")
	config.LogFile = getEnvStringLocal("LOG_FILE", "")

	// Performance
	config.BufferSize = getEnvIntLocal("BUFFER_SIZE", 4096)
	config.MaxMessageSize = getEnvIntLocal("MAX_MESSAGE_SIZE", 1048576)
	config.ConnectionTimeout = getEnvDurationLocal("CONNECTION_TIMEOUT", 30*time.Second)
	config.ReadTimeout = getEnvDurationLocal("READ_TIMEOUT", 60*time.Second)
	config.WriteTimeout = getEnvDurationLocal("WRITE_TIMEOUT", 60*time.Second)
	config.IdleTimeout = getEnvDurationLocal("IDLE_TIMEOUT", 300*time.Second)

	// Development
	config.DevelopmentMode = getEnvBoolLocal("DEVELOPMENT_MODE", true)
	config.MockDataEnabled = getEnvBoolLocal("MOCK_DATA_ENABLED", false)
	config.TestMode = getEnvBoolLocal("TEST_MODE", false)
	config.VerboseLogging = getEnvBoolLocal("VERBOSE_LOGGING", false)
}

// Helper functions for environment variable parsing
func getEnvStringLocal(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBoolLocal(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvIntLocal(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDurationLocal(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}