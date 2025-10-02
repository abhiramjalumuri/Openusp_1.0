package config

import (
	"time"
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


