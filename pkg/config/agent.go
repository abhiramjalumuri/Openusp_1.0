package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AgentConfig represents the base configuration for protocol agents
type AgentConfig struct {
	// Service Discovery

	// Device Information
	EndpointID      string `json:"endpoint_id"`
	ProductClass    string `json:"product_class"`
	Manufacturer    string `json:"manufacturer"`
	ModelName       string `json:"model_name"`
	SerialNumber    string `json:"serial_number"`
	SoftwareVersion string `json:"software_version"`
	HardwareVersion string `json:"hardware_version"`
	DeviceType      string `json:"device_type"`
	OUI             string `json:"oui"`

	// Platform Integration
	PlatformURL          string        `json:"platform_url"`
	APIVersion           string        `json:"api_version"`
	RegistrationEndpoint string        `json:"registration_endpoint"`
	HeartbeatInterval    time.Duration `json:"heartbeat_interval"`
	AutoRegister         bool          `json:"auto_register"`

	// Consul Integration
	ServiceDiscoveryEnabled bool          `json:"service_discovery_enabled"`
	HealthCheckInterval     time.Duration `json:"health_check_interval"`
	ServiceTags             []string      `json:"service_tags"`

	// Security
	TLSEnabled    bool   `json:"tls_enabled"`
	TLSCertFile   string `json:"tls_cert_file"`
	TLSKeyFile    string `json:"tls_key_file"`
	TLSCAFile     string `json:"tls_ca_file"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`

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
	USPVersion           string   `json:"usp_version"`
	USPSupportedVersions []string `json:"usp_supported_versions"`
	USPAgentRole         string   `json:"usp_agent_role"`
	USPCommandKey        string   `json:"usp_command_key"`

	// MTP Configuration
	MTPType string `json:"mtp_type"`

	// WebSocket MTP
	WebSocketURL                  string        `json:"websocket_url"`
	WebSocketSubprotocol          string        `json:"websocket_subprotocol"`
	WebSocketPingInterval         time.Duration `json:"websocket_ping_interval"`
	WebSocketPongTimeout          time.Duration `json:"websocket_pong_timeout"`
	WebSocketReconnectInterval    time.Duration `json:"websocket_reconnect_interval"`
	WebSocketMaxReconnectAttempts int           `json:"websocket_max_reconnect_attempts"`

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
	STOMPBrokerURL             string        `json:"stomp_broker_url"`
	STOMPUsername              string        `json:"stomp_username"`
	STOMPPassword              string        `json:"stomp_password"`
	STOMPDestinationRequest    string        `json:"stomp_destinations_request"`
	STOMPDestinationResponse   string        `json:"stomp_destinations_response"`
	STOMPDestinationOutbound   string        `json:"stomp_destinations_outbound"`
	STOMPDestinationInbound    string        `json:"stomp_destinations_inbound"`
	STOMPDestinationBroadcast  string        `json:"stomp_destinations_broadcast"`
	STOMPHeartbeatSend         time.Duration `json:"stomp_heartbeat_send"`
	STOMPHeartbeatReceive      time.Duration `json:"stomp_heartbeat_receive"`

	// Unix Domain Socket MTP
	UDSSocketPath string `json:"uds_socket_path"`
	UDSSocketMode string `json:"uds_socket_mode"`

	// Agent Behavior
	PeriodicInformEnabled       bool          `json:"periodic_inform_enabled"`
	PeriodicInformInterval      time.Duration `json:"periodic_inform_interval"`
	ParameterUpdateNotification bool          `json:"parameter_update_notification"`
	EventNotification           bool          `json:"event_notification"`
	DiagnosticMode              bool          `json:"diagnostic_mode"`

	// Consul Service Names
}

// Agents holds both USP (TR-369) and CWMP (TR-069) agent configurations parsed from the unified YAML.
type Agents struct {
	USP  *TR369Config
	CWMP *TR069Config
}

// LoadAgents loads both USP and CWMP agent configs from agents.yml (single pass)
func LoadAgents(path string) (*Agents, error) {
	if path == "" {
		path = "configs/agents.yml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agents.yml: %w", err)
	}

	// Unmarshal into unified structure capturing both agent sections
	var unified struct {
		USPAgent struct { // usp_agent
			Device struct {
				EndpointID      string `yaml:"endpoint_id"`
				ProductClass    string `yaml:"product_class"`
				Manufacturer    string `yaml:"manufacturer"`
				ModelName       string `yaml:"model_name"`
				SerialNumber    string `yaml:"serial_number"`
				SoftwareVersion string `yaml:"software_version"`
				HardwareVersion string `yaml:"hardware_version"`
				DeviceType      string `yaml:"device_type"`
				OUI             string `yaml:"oui"`
			} `yaml:"device"`
			Protocol struct {
				Version           string `yaml:"version"`
				SupportedVersions string `yaml:"supported_versions"`
				Role              string `yaml:"role"`
				CommandKey        string `yaml:"command_key"`
			} `yaml:"protocol"`
			MTP struct {
				Type      string `yaml:"type"`
				WebSocket struct {
					URL                  string `yaml:"url"`
					Subprotocol          string `yaml:"subprotocol"`
					PingInterval         string `yaml:"ping_interval"`
					PongTimeout          string `yaml:"pong_timeout"`
					ReconnectInterval    string `yaml:"reconnect_interval"`
					MaxReconnectAttempts int    `yaml:"max_reconnect_attempts"`
				} `yaml:"websocket"`
				MQTT struct {
					BrokerURL     string `yaml:"broker_url"`
					ClientID      string `yaml:"client_id"`
					Username      string `yaml:"username"`
					Password      string `yaml:"password"`
					TopicRequest  string `yaml:"topic_request"`
					TopicResponse string `yaml:"topic_response"`
					KeepAlive     string `yaml:"keep_alive"`
					QOS           int    `yaml:"qos"`
					Retain        bool   `yaml:"retain"`
					CleanSession  bool   `yaml:"clean_session"`
				} `yaml:"mqtt"`
				STOMP struct {
					BrokerURL    string `yaml:"broker_url"`
					Username     string `yaml:"username"`
					Password     string `yaml:"password"`
					Destinations struct {
						Request   string `yaml:"request"`
						Response  string `yaml:"response"`
						Outbound  string `yaml:"outbound"`
						Inbound   string `yaml:"inbound"`
						Broadcast string `yaml:"broadcast"`
					} `yaml:"destinations"`
					Heartbeat struct {
						Send    string `yaml:"send"`
						Receive string `yaml:"receive"`
					} `yaml:"heartbeat"`
				} `yaml:"stomp"`
				UDS struct {
					SocketPath string `yaml:"socket_path"`
					SocketMode string `yaml:"socket_mode"`
				} `yaml:"uds"`
			} `yaml:"mtp"`
			Behavior struct {
				AutoRegister                bool   `yaml:"auto_register"`
				PeriodicInformEnabled       bool   `yaml:"periodic_inform_enabled"`
				PeriodicInformInterval      string `yaml:"periodic_inform_interval"`
				ParameterUpdateNotification bool   `yaml:"parameter_update_notification"`
				EventNotification           bool   `yaml:"event_notification"`
				DiagnosticMode              bool   `yaml:"diagnostic_mode"`
			} `yaml:"behavior"`
			Security struct {
				TLSEnabled    bool   `yaml:"tls_enabled"`
				TLSCertFile   string `yaml:"tls_cert_file"`
				TLSKeyFile    string `yaml:"tls_key_file"`
				TLSCAFile     string `yaml:"tls_ca_file"`
				TLSSkipVerify bool   `yaml:"tls_skip_verify"`
			} `yaml:"security"`
			Logging struct {
				Level  string `yaml:"level"`
				Format string `yaml:"format"`
				Output string `yaml:"output"`
				File   string `yaml:"file"`
			} `yaml:"logging"`
			Performance struct {
				BufferSize        int    `yaml:"buffer_size"`
				MaxMessageSize    int    `yaml:"max_message_size"`
				ConnectionTimeout string `yaml:"connection_timeout"`
				ReadTimeout       string `yaml:"read_timeout"`
				WriteTimeout      string `yaml:"write_timeout"`
				IdleTimeout       string `yaml:"idle_timeout"`
			} `yaml:"performance"`
		} `yaml:"usp_agent"`
		CWMpAgent struct { // cwmp_agent
			Device struct {
				EndpointID      string `yaml:"endpoint_id"`
				ProductClass    string `yaml:"product_class"`
				Manufacturer    string `yaml:"manufacturer"`
				ModelName       string `yaml:"model_name"`
				SerialNumber    string `yaml:"serial_number"`
				SoftwareVersion string `yaml:"software_version"`
				HardwareVersion string `yaml:"hardware_version"`
				DeviceType      string `yaml:"device_type"`
				OUI             string `yaml:"oui"`
			} `yaml:"device"`
			Protocol struct {
				Version               string `yaml:"version"`
				SupportedVersions     string `yaml:"supported_versions"`
				Role                  string `yaml:"role"`
				ConnectionRequestAuth string `yaml:"connection_request_auth"`
				ParameterKey          string `yaml:"parameter_key"`
			} `yaml:"protocol"`
			ACS struct {
				URL                    string `yaml:"url"`
				Username               string `yaml:"username"`
				Password               string `yaml:"password"`
				PeriodicInformEnabled  bool   `yaml:"periodic_inform_enabled"`
				PeriodicInformInterval string `yaml:"periodic_inform_interval"`
			} `yaml:"acs"`
			ConnectionRequest struct {
				URL      string `yaml:"url"`
				Path     string `yaml:"path"`
				Port     int    `yaml:"port"`
				Username string `yaml:"username"`
				Password string `yaml:"password"`
			} `yaml:"connection_request"`
			SOAP struct {
				Version string `yaml:"version"`
			} `yaml:"soap"`
			HTTP struct {
				Timeout               string `yaml:"timeout"`
				KeepAlive             bool   `yaml:"keep_alive"`
				MaxIdleConnections    int    `yaml:"max_idle_connections"`
				IdleTimeout           string `yaml:"idle_timeout"`
				TLSHandshakeTimeout   string `yaml:"tls_handshake_timeout"`
				ExpectContinueTimeout string `yaml:"expect_continue_timeout"`
			} `yaml:"http"`
			Behavior struct {
				AutoRegister              bool   `yaml:"auto_register"`
				InformOnBoot              bool   `yaml:"inform_on_boot"`
				InformOnPeriodic          bool   `yaml:"inform_on_periodic"`
				InformOnValueChange       bool   `yaml:"inform_on_value_change"`
				InformOnConnectionRequest bool   `yaml:"inform_on_connection_request"`
				MaxEnvelopes              int    `yaml:"max_envelopes"`
				RetryIntervalMultiplier   int    `yaml:"retry_interval_multiplier"`
				RetryMinWaitInterval      string `yaml:"retry_min_wait_interval"`
			} `yaml:"behavior"`
			Security struct {
				TLSEnabled             bool   `yaml:"tls_enabled"`
				TLSCertFile            string `yaml:"tls_cert_file"`
				TLSKeyFile             string `yaml:"tls_key_file"`
				TLSCAFile              string `yaml:"tls_ca_file"`
				TLSSkipVerify          bool   `yaml:"tls_skip_verify"`
				CertificateAuthEnabled bool   `yaml:"certificate_auth_enabled"`
				DigestAuthEnabled      bool   `yaml:"digest_auth_enabled"`
				BasicAuthEnabled       bool   `yaml:"basic_auth_enabled"`
			} `yaml:"security"`
			Auth struct {
				Realm        string `yaml:"realm"`
				Algorithm    string `yaml:"algorithm"`
				QOP          string `yaml:"qop"`
				NonceTimeout string `yaml:"nonce_timeout"`
			} `yaml:"auth"`
			Logging struct {
				Level        string `yaml:"level"`
				Format       string `yaml:"format"`
				Output       string `yaml:"output"`
				File         string `yaml:"file"`
				SOAPMessages bool   `yaml:"soap_messages"`
				HTTPRequests bool   `yaml:"http_requests"`
			} `yaml:"logging"`
			Performance struct {
				BufferSize        int    `yaml:"buffer_size"`
				MaxMessageSize    int    `yaml:"max_message_size"`
				ConnectionTimeout string `yaml:"connection_timeout"`
				ReadTimeout       string `yaml:"read_timeout"`
				WriteTimeout      string `yaml:"write_timeout"`
				IdleTimeout       string `yaml:"idle_timeout"`
				MaxConnections    int    `yaml:"max_connections"`
			} `yaml:"performance"`
		} `yaml:"cwmp_agent"`
	}
	if err := yaml.Unmarshal(data, &unified); err != nil {
		return nil, fmt.Errorf("parse openusp.yml: %w", err)
	}

	usp := &TR369Config{}
	mapUSP := unified.USPAgent
	usp.EndpointID = mapUSP.Device.EndpointID
	usp.ProductClass = mapUSP.Device.ProductClass
	usp.Manufacturer = mapUSP.Device.Manufacturer
	usp.ModelName = mapUSP.Device.ModelName
	usp.SerialNumber = mapUSP.Device.SerialNumber
	usp.SoftwareVersion = mapUSP.Device.SoftwareVersion
	usp.HardwareVersion = mapUSP.Device.HardwareVersion
	usp.DeviceType = mapUSP.Device.DeviceType
	usp.OUI = mapUSP.Device.OUI
	usp.USPVersion = mapUSP.Protocol.Version
	if mapUSP.Protocol.SupportedVersions != "" {
		usp.USPSupportedVersions = splitAndTrim(mapUSP.Protocol.SupportedVersions)
	}
	usp.USPAgentRole = mapUSP.Protocol.Role
	usp.USPCommandKey = mapUSP.Protocol.CommandKey
	usp.MTPType = mapUSP.MTP.Type
	usp.WebSocketURL = mapUSP.MTP.WebSocket.URL
	usp.WebSocketSubprotocol = mapUSP.MTP.WebSocket.Subprotocol
	usp.WebSocketPingInterval = parseDurationOrZero(mapUSP.MTP.WebSocket.PingInterval)
	usp.WebSocketPongTimeout = parseDurationOrZero(mapUSP.MTP.WebSocket.PongTimeout)
	usp.WebSocketReconnectInterval = parseDurationOrZero(mapUSP.MTP.WebSocket.ReconnectInterval)
	usp.WebSocketMaxReconnectAttempts = mapUSP.MTP.WebSocket.MaxReconnectAttempts
	usp.MQTTBrokerURL = mapUSP.MTP.MQTT.BrokerURL
	usp.MQTTClientID = mapUSP.MTP.MQTT.ClientID
	usp.MQTTUsername = mapUSP.MTP.MQTT.Username
	usp.MQTTPassword = mapUSP.MTP.MQTT.Password
	usp.MQTTTopicRequest = mapUSP.MTP.MQTT.TopicRequest
	usp.MQTTTopicResponse = mapUSP.MTP.MQTT.TopicResponse
	usp.MQTTQOS = mapUSP.MTP.MQTT.QOS
	usp.MQTTRetain = mapUSP.MTP.MQTT.Retain
	usp.MQTTCleanSession = mapUSP.MTP.MQTT.CleanSession
	usp.MQTTKeepAlive = parseDurationOrZero(mapUSP.MTP.MQTT.KeepAlive)
	usp.STOMPBrokerURL = mapUSP.MTP.STOMP.BrokerURL
	usp.STOMPUsername = mapUSP.MTP.STOMP.Username
	usp.STOMPPassword = mapUSP.MTP.STOMP.Password
	usp.STOMPDestinationRequest = mapUSP.MTP.STOMP.Destinations.Request
	usp.STOMPDestinationResponse = mapUSP.MTP.STOMP.Destinations.Response
	usp.STOMPDestinationOutbound = mapUSP.MTP.STOMP.Destinations.Outbound
	usp.STOMPDestinationInbound = mapUSP.MTP.STOMP.Destinations.Inbound
	usp.STOMPDestinationBroadcast = mapUSP.MTP.STOMP.Destinations.Broadcast
	usp.STOMPHeartbeatSend = parseDurationOrZero(mapUSP.MTP.STOMP.Heartbeat.Send)
	usp.STOMPHeartbeatReceive = parseDurationOrZero(mapUSP.MTP.STOMP.Heartbeat.Receive)
	usp.UDSSocketPath = mapUSP.MTP.UDS.SocketPath
	usp.UDSSocketMode = mapUSP.MTP.UDS.SocketMode
	usp.AutoRegister = mapUSP.Behavior.AutoRegister
	usp.PeriodicInformEnabled = mapUSP.Behavior.PeriodicInformEnabled
	usp.PeriodicInformInterval = parseDurationOrZero(mapUSP.Behavior.PeriodicInformInterval)
	usp.ParameterUpdateNotification = mapUSP.Behavior.ParameterUpdateNotification
	usp.EventNotification = mapUSP.Behavior.EventNotification
	usp.DiagnosticMode = mapUSP.Behavior.DiagnosticMode
	usp.TLSEnabled = mapUSP.Security.TLSEnabled
	usp.TLSCertFile = mapUSP.Security.TLSCertFile
	usp.TLSKeyFile = mapUSP.Security.TLSKeyFile
	usp.TLSCAFile = mapUSP.Security.TLSCAFile
	usp.TLSSkipVerify = mapUSP.Security.TLSSkipVerify
	usp.LogLevel = mapUSP.Logging.Level
	usp.LogFormat = mapUSP.Logging.Format
	usp.LogOutput = mapUSP.Logging.Output
	usp.LogFile = mapUSP.Logging.File
	usp.BufferSize = mapUSP.Performance.BufferSize
	usp.MaxMessageSize = mapUSP.Performance.MaxMessageSize
	usp.ConnectionTimeout = parseDurationOrZero(mapUSP.Performance.ConnectionTimeout)
	usp.ReadTimeout = parseDurationOrZero(mapUSP.Performance.ReadTimeout)
	usp.WriteTimeout = parseDurationOrZero(mapUSP.Performance.WriteTimeout)
	usp.IdleTimeout = parseDurationOrZero(mapUSP.Performance.IdleTimeout)

	cwmp := &TR069Config{}
	mapCW := unified.CWMpAgent
	cwmp.EndpointID = mapCW.Device.EndpointID
	cwmp.ProductClass = mapCW.Device.ProductClass
	cwmp.Manufacturer = mapCW.Device.Manufacturer
	cwmp.ModelName = mapCW.Device.ModelName
	cwmp.SerialNumber = mapCW.Device.SerialNumber
	cwmp.SoftwareVersion = mapCW.Device.SoftwareVersion
	cwmp.HardwareVersion = mapCW.Device.HardwareVersion
	cwmp.DeviceType = mapCW.Device.DeviceType
	cwmp.OUI = mapCW.Device.OUI
	cwmp.CWMPVersion = mapCW.Protocol.Version
	if mapCW.Protocol.SupportedVersions != "" {
		cwmp.CWMPSupportedVersions = splitAndTrim(mapCW.Protocol.SupportedVersions)
	}
	cwmp.CWMPAgentRole = mapCW.Protocol.Role
	cwmp.CWMPConnectionRequestAuth = mapCW.Protocol.ConnectionRequestAuth
	cwmp.ACSURL = mapCW.ACS.URL
	cwmp.ACSUsername = mapCW.ACS.Username
	cwmp.ACSPassword = mapCW.ACS.Password
	cwmp.ACSPeriodicInformEnabled = mapCW.ACS.PeriodicInformEnabled
	cwmp.ACSPeriodicInformInterval = parseDurationOrZero(mapCW.ACS.PeriodicInformInterval)
	cwmp.ACSParameterKey = mapCW.Protocol.ParameterKey
	cwmp.ConnectionRequestURL = mapCW.ConnectionRequest.URL
	cwmp.ConnectionRequestPath = mapCW.ConnectionRequest.Path
	cwmp.ConnectionRequestPort = mapCW.ConnectionRequest.Port
	cwmp.ConnectionRequestUsername = mapCW.ConnectionRequest.Username
	cwmp.ConnectionRequestPassword = mapCW.ConnectionRequest.Password
	cwmp.SOAPVersion = mapCW.SOAP.Version
	cwmp.HTTPTimeout = parseDurationOrZero(mapCW.HTTP.Timeout)
	cwmp.HTTPKeepAlive = mapCW.HTTP.KeepAlive
	cwmp.HTTPMaxIdleConnections = mapCW.HTTP.MaxIdleConnections
	cwmp.HTTPIdleTimeout = parseDurationOrZero(mapCW.HTTP.IdleTimeout)
	cwmp.HTTPTLSHandshakeTimeout = parseDurationOrZero(mapCW.HTTP.TLSHandshakeTimeout)
	cwmp.HTTPExpectContinueTimeout = parseDurationOrZero(mapCW.HTTP.ExpectContinueTimeout)
	cwmp.AutoRegister = mapCW.Behavior.AutoRegister
	cwmp.InformOnBoot = mapCW.Behavior.InformOnBoot
	cwmp.InformOnPeriodic = mapCW.Behavior.InformOnPeriodic
	cwmp.InformOnValueChange = mapCW.Behavior.InformOnValueChange
	cwmp.InformOnConnectionRequest = mapCW.Behavior.InformOnConnectionRequest
	cwmp.MaxEnvelopes = mapCW.Behavior.MaxEnvelopes
	cwmp.RetryMinWaitInterval = parseDurationOrZero(mapCW.Behavior.RetryMinWaitInterval)
	cwmp.RetryIntervalMultiplier = mapCW.Behavior.RetryIntervalMultiplier
	cwmp.TLSEnabled = mapCW.Security.TLSEnabled
	cwmp.TLSCertFile = mapCW.Security.TLSCertFile
	cwmp.TLSKeyFile = mapCW.Security.TLSKeyFile
	cwmp.TLSCAFile = mapCW.Security.TLSCAFile
	cwmp.TLSSkipVerify = mapCW.Security.TLSSkipVerify
	cwmp.DigestAuthEnabled = mapCW.Security.DigestAuthEnabled
	cwmp.BasicAuthEnabled = mapCW.Security.BasicAuthEnabled
	cwmp.AuthRealm = mapCW.Auth.Realm
	cwmp.AuthAlgorithm = mapCW.Auth.Algorithm
	cwmp.AuthQOP = mapCW.Auth.QOP
	cwmp.AuthNonceTimeout = parseDurationOrZero(mapCW.Auth.NonceTimeout)
	cwmp.BufferSize = mapCW.Performance.BufferSize
	cwmp.MaxMessageSize = mapCW.Performance.MaxMessageSize
	cwmp.ConnectionTimeout = parseDurationOrZero(mapCW.Performance.ConnectionTimeout)
	cwmp.ReadTimeout = parseDurationOrZero(mapCW.Performance.ReadTimeout)
	cwmp.WriteTimeout = parseDurationOrZero(mapCW.Performance.WriteTimeout)
	cwmp.IdleTimeout = parseDurationOrZero(mapCW.Performance.IdleTimeout)
	return &Agents{USP: usp, CWMP: cwmp}, nil
}

// Backwards compatibility loaders (prefer LoadAgents in new code)
func LoadUSPAgentUnified(path string) (*TR369Config, error) {
	a, err := LoadAgents(path)
	if err != nil {
		return nil, err
	}
	return a.USP, nil
}
func LoadCWMPAgentUnified(path string) (*TR069Config, error) {
	a, err := LoadAgents(path)
	if err != nil {
		return nil, err
	}
	return a.CWMP, nil
}

// Helper utilities (previously in unified_agents.go)
func parseDurationOrZero(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, _ := time.ParseDuration(s)
	return d
}
func splitAndTrim(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// TR069Config represents TR-069 CWMP agent specific configuration
type TR069Config struct {
	AgentConfig

	// CWMP Protocol
	CWMPVersion               string   `json:"cwmp_version"`
	CWMPSupportedVersions     []string `json:"cwmp_supported_versions"`
	CWMPAgentRole             string   `json:"cwmp_agent_role"`
	CWMPConnectionRequestAuth string   `json:"cwmp_connection_request_auth"`

	// ACS Configuration
	ACSURL                    string        `json:"acs_url"`
	ACSUsername               string        `json:"acs_username"`
	ACSPassword               string        `json:"acs_password"`
	ACSPeriodicInformEnabled  bool          `json:"acs_periodic_inform_enabled"`
	ACSPeriodicInformInterval time.Duration `json:"acs_periodic_inform_interval"`
	ACSParameterKey           string        `json:"acs_parameter_key"`

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
	HTTPTimeout               time.Duration `json:"http_timeout"`
	HTTPKeepAlive             bool          `json:"http_keep_alive"`
	HTTPMaxIdleConnections    int           `json:"http_max_idle_connections"`
	HTTPIdleTimeout           time.Duration `json:"http_idle_timeout"`
	HTTPTLSHandshakeTimeout   time.Duration `json:"http_tls_handshake_timeout"`
	HTTPExpectContinueTimeout time.Duration `json:"http_expect_continue_timeout"`

	// Session Management
	SessionTimeout       time.Duration `json:"session_timeout"`
	SessionMaxConcurrent int           `json:"session_max_concurrent"`
	SessionKeepAlive     bool          `json:"session_keep_alive"`
	SessionCookieEnabled bool          `json:"session_cookie_enabled"`

	// TR-181 Data Model
	TR181RootObject             string `json:"tr181_root_object"`
	TR181DeviceType             string `json:"tr181_device_type"`
	TR181SpecVersion            string `json:"tr181_spec_version"`
	TR181ParameterNotifications bool   `json:"tr181_parameter_notifications"`
	TR181ObjectNotifications    bool   `json:"tr181_object_notifications"`

	// Authentication
	DigestAuthEnabled bool          `json:"digest_auth_enabled"`
	BasicAuthEnabled  bool          `json:"basic_auth_enabled"`
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
	EventBootstrap                  bool `json:"event_bootstrap"`
	EventBoot                       bool `json:"event_boot"`
	EventPeriodic                   bool `json:"event_periodic"`
	EventValueChange                bool `json:"event_value_change"`
	EventConnectionRequest          bool `json:"event_connection_request"`
	EventTransferComplete           bool `json:"event_transfer_complete"`
	EventDiagnosticsComplete        bool `json:"event_diagnostics_complete"`
	EventRequestDownload            bool `json:"event_request_download"`
	EventAutonomousTransferComplete bool `json:"event_autonomous_transfer_complete"`

	// Fault Management
	FaultToleranceEnabled  bool          `json:"fault_tolerance_enabled"`
	FaultMaxRetryCount     int           `json:"fault_max_retry_count"`
	FaultRetryInterval     time.Duration `json:"fault_retry_interval"`
	FaultDetailedReporting bool          `json:"fault_detailed_reporting"`

	// Consul Service Names

	// CWMP Service
	CWMPServiceEndpoint string `json:"cwmp_service_endpoint"`

	// Logging
	LogSOAPMessages bool `json:"log_soap_messages"`
	LogHTTPRequests bool `json:"log_http_requests"`

	// Simulation
	SimulateDeviceResponses bool `json:"simulate_device_responses"`
}
