package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// YAMLAgentConfig represents the YAML structure for agent configuration
type YAMLAgentConfig struct {
	ServiceDiscovery struct {
		ConsulEnabled    bool   `yaml:"consul_enabled"`
		ConsulAddr       string `yaml:"consul_addr"`
		ConsulDatacenter string `yaml:"consul_datacenter"`
	} `yaml:"service_discovery"`

	DeviceInfo struct {
		EndpointID      string `yaml:"endpoint_id"`
		ProductClass    string `yaml:"product_class"`
		Manufacturer    string `yaml:"manufacturer"`
		ModelName       string `yaml:"model_name"`
		SerialNumber    string `yaml:"serial_number"`
		SoftwareVersion string `yaml:"software_version"`
		HardwareVersion string `yaml:"hardware_version"`
		DeviceType      string `yaml:"device_type"`
		OUI             string `yaml:"oui"`
	} `yaml:"device_info"`

	Platform struct {
		URL                  string `yaml:"url"`
		APIVersion           string `yaml:"api_version"`
		RegistrationEndpoint string `yaml:"registration_endpoint"`
		HeartbeatInterval    string `yaml:"heartbeat_interval"`
	} `yaml:"platform"`

	ConsulIntegration struct {
		ServiceDiscoveryEnabled bool     `yaml:"service_discovery_enabled"`
		HealthCheckInterval     string   `yaml:"health_check_interval"`
		ServiceTags             []string `yaml:"service_tags"`
	} `yaml:"consul_integration"`

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

	Development struct {
		Mode            bool `yaml:"mode"`
		MockDataEnabled bool `yaml:"mock_data_enabled"`
		TestMode        bool `yaml:"test_mode"`
		VerboseLogging  bool `yaml:"verbose_logging"`
	} `yaml:"development"`
}

// YAMLTR369Config represents the YAML structure for TR-369 USP agent configuration
type YAMLTR369Config struct {
	YAMLAgentConfig `yaml:",inline"`

	USPProtocol struct {
		Version           string   `yaml:"version"`
		SupportedVersions []string `yaml:"supported_versions"`
		AgentRole         string   `yaml:"agent_role"`
		CommandKey        string   `yaml:"command_key"`
	} `yaml:"usp_protocol"`

	MTP struct {
		DefaultType string `yaml:"default_type"`

		WebSocket struct {
			URL                  string `yaml:"url"`
			Subprotocol          string `yaml:"subprotocol"`
			PingInterval         string `yaml:"ping_interval"`
			PongTimeout          string `yaml:"pong_timeout"`
			ReconnectInterval    string `yaml:"reconnect_interval"`
			MaxReconnectAttempts int    `yaml:"max_reconnect_attempts"`
			MessageEncoding      string `yaml:"message_encoding"`
		} `yaml:"websocket"`

		MQTT struct {
			BrokerURL       string `yaml:"broker_url"`
			ClientID        string `yaml:"client_id"`
			Username        string `yaml:"username"`
			Password        string `yaml:"password"`
			TopicRequest    string `yaml:"topic_request"`
			TopicResponse   string `yaml:"topic_response"`
			QOS             int    `yaml:"qos"`
			Retain          bool   `yaml:"retain"`
			CleanSession    bool   `yaml:"clean_session"`
			KeepAlive       string `yaml:"keep_alive"`
			MessageEncoding string `yaml:"message_encoding"`
		} `yaml:"mqtt"`

		STOMP struct {
			BrokerURL           string `yaml:"broker_url"`
			Username            string `yaml:"username"`
			Password            string `yaml:"password"`
			DestinationRequest  string `yaml:"destination_request"`
			DestinationResponse string `yaml:"destination_response"`
			HeartbeatSend       string `yaml:"heartbeat_send"`
			HeartbeatReceive    string `yaml:"heartbeat_receive"`
			MessageEncoding     string `yaml:"message_encoding"`
		} `yaml:"stomp"`

		UDS struct {
			SocketPath      string `yaml:"socket_path"`
			SocketMode      string `yaml:"socket_mode"`
			MessageEncoding string `yaml:"message_encoding"`
		} `yaml:"uds"`
	} `yaml:"mtp"`

	AgentBehavior struct {
		AutoRegister                bool   `yaml:"auto_register"`
		PeriodicInformEnabled       bool   `yaml:"periodic_inform_enabled"`
		PeriodicInformInterval      string `yaml:"periodic_inform_interval"`
		ParameterUpdateNotification bool   `yaml:"parameter_update_notification"`
		EventNotification           bool   `yaml:"event_notification"`
		DiagnosticMode              bool   `yaml:"diagnostic_mode"`
		MaxRetryCount               int    `yaml:"max_retry_count"`
		RetryInterval               string `yaml:"retry_interval"`
	} `yaml:"agent_behavior"`
}

// YAMLTR069Config represents the YAML structure for TR-069 CWMP agent configuration
type YAMLTR069Config struct {
	YAMLAgentConfig `yaml:",inline"`

	CWMPProtocol struct {
		Version               string   `yaml:"version"`
		SupportedVersions     []string `yaml:"supported_versions"`
		AgentRole             string   `yaml:"agent_role"`
		ConnectionRequestAuth string   `yaml:"connection_request_auth"`
	} `yaml:"cwmp_protocol"`

	ACS struct {
		URL                    string `yaml:"url"`
		Username               string `yaml:"username"`
		Password               string `yaml:"password"`
		PeriodicInformEnabled  bool   `yaml:"periodic_inform_enabled"`
		PeriodicInformInterval string `yaml:"periodic_inform_interval"`
		ParameterKey           string `yaml:"parameter_key"`
	} `yaml:"acs"`

	ConnectionRequest struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Path     string `yaml:"path"`
		Port     int    `yaml:"port"`
	} `yaml:"connection_request"`

	SOAPXML struct {
		SOAPVersion      string `yaml:"soap_version"`
		SOAPNamespace    string `yaml:"soap_namespace"`
		XMLStrictParsing bool   `yaml:"xml_strict_parsing"`
		XMLPrettyPrint   bool   `yaml:"xml_pretty_print"`
		EnvelopeNSPrefix string `yaml:"envelope_namespace_prefix"`
	} `yaml:"soap_xml"`

	HTTP struct {
		Timeout               string `yaml:"timeout"`
		KeepAlive             bool   `yaml:"keep_alive"`
		MaxIdleConnections    int    `yaml:"max_idle_connections"`
		IdleTimeout           string `yaml:"idle_timeout"`
		TLSHandshakeTimeout   string `yaml:"tls_handshake_timeout"`
		ExpectContinueTimeout string `yaml:"expect_continue_timeout"`
	} `yaml:"http"`

	AgentBehavior struct {
		AutoRegister              bool   `yaml:"auto_register"`
		InformOnBoot              bool   `yaml:"inform_on_boot"`
		InformOnPeriodic          bool   `yaml:"inform_on_periodic"`
		InformOnValueChange       bool   `yaml:"inform_on_value_change"`
		InformOnConnectionRequest bool   `yaml:"inform_on_connection_request"`
		MaxEnvelopes              int    `yaml:"max_envelopes"`
		RetryMinWaitInterval      string `yaml:"retry_min_wait_interval"`
		RetryIntervalMultiplier   int    `yaml:"retry_interval_multiplier"`
	} `yaml:"agent_behavior"`

	TR181 struct {
		RootObject             string `yaml:"root_object"`
		DeviceType             string `yaml:"device_type"`
		SpecVersion            string `yaml:"spec_version"`
		ParameterNotifications bool   `yaml:"parameter_notifications"`
		ObjectNotifications    bool   `yaml:"object_notifications"`
	} `yaml:"tr181"`

	Session struct {
		Timeout       string `yaml:"timeout"`
		MaxConcurrent int    `yaml:"max_concurrent"`
		KeepAlive     bool   `yaml:"keep_alive"`
		CookieEnabled bool   `yaml:"cookie_enabled"`
	} `yaml:"session"`

	Authentication struct {
		Realm        string `yaml:"realm"`
		Algorithm    string `yaml:"algorithm"`
		QOP          string `yaml:"qop"`
		NonceTimeout string `yaml:"nonce_timeout"`
	} `yaml:"authentication"`

	RPCMethods struct {
		GetRPCMethods          bool `yaml:"get_rpc_methods"`
		SetParameterValues     bool `yaml:"set_parameter_values"`
		GetParameterValues     bool `yaml:"get_parameter_values"`
		GetParameterNames      bool `yaml:"get_parameter_names"`
		SetParameterAttributes bool `yaml:"set_parameter_attributes"`
		GetParameterAttributes bool `yaml:"get_parameter_attributes"`
		AddObject              bool `yaml:"add_object"`
		DeleteObject           bool `yaml:"delete_object"`
		Reboot                 bool `yaml:"reboot"`
		FactoryReset           bool `yaml:"factory_reset"`
		Download               bool `yaml:"download"`
		Upload                 bool `yaml:"upload"`
	} `yaml:"rpc_methods"`

	Events struct {
		Bootstrap                  bool `yaml:"bootstrap"`
		Boot                       bool `yaml:"boot"`
		Periodic                   bool `yaml:"periodic"`
		ValueChange                bool `yaml:"value_change"`
		ConnectionRequest          bool `yaml:"connection_request"`
		TransferComplete           bool `yaml:"transfer_complete"`
		DiagnosticsComplete        bool `yaml:"diagnostics_complete"`
		RequestDownload            bool `yaml:"request_download"`
		AutonomousTransferComplete bool `yaml:"autonomous_transfer_complete"`
	} `yaml:"events"`

	FaultManagement struct {
		ToleranceEnabled  bool   `yaml:"tolerance_enabled"`
		MaxRetryCount     int    `yaml:"max_retry_count"`
		RetryInterval     string `yaml:"retry_interval"`
		DetailedReporting bool   `yaml:"detailed_reporting"`
	} `yaml:"fault_management"`
}

// LoadYAMLTR369Config loads TR-369 agent configuration from YAML file
func LoadYAMLTR369Config(configPath string) (*TR369Config, error) {
	yamlConfig := &YAMLTR369Config{}

	// Read YAML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML config file: %v", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %v", err)
	}

	// Convert to TR369Config
	config := &TR369Config{}

	// Service Discovery
	config.ConsulEnabled = yamlConfig.ServiceDiscovery.ConsulEnabled
	config.ConsulAddr = yamlConfig.ServiceDiscovery.ConsulAddr
	config.ConsulDatacenter = yamlConfig.ServiceDiscovery.ConsulDatacenter

	// Device Information
	config.EndpointID = yamlConfig.DeviceInfo.EndpointID
	config.ProductClass = yamlConfig.DeviceInfo.ProductClass
	config.Manufacturer = yamlConfig.DeviceInfo.Manufacturer
	config.ModelName = yamlConfig.DeviceInfo.ModelName
	config.SerialNumber = yamlConfig.DeviceInfo.SerialNumber
	config.SoftwareVersion = yamlConfig.DeviceInfo.SoftwareVersion
	config.HardwareVersion = yamlConfig.DeviceInfo.HardwareVersion
	config.DeviceType = yamlConfig.DeviceInfo.DeviceType
	config.OUI = yamlConfig.DeviceInfo.OUI

	// Platform Integration
	config.PlatformURL = yamlConfig.Platform.URL
	config.APIVersion = yamlConfig.Platform.APIVersion
	config.RegistrationEndpoint = yamlConfig.Platform.RegistrationEndpoint
	if yamlConfig.Platform.HeartbeatInterval != "" {
		if duration, err := time.ParseDuration(yamlConfig.Platform.HeartbeatInterval); err == nil {
			config.HeartbeatInterval = duration
		}
	}

	// USP Protocol
	config.USPVersion = yamlConfig.USPProtocol.Version
	config.USPSupportedVersions = yamlConfig.USPProtocol.SupportedVersions
	config.USPAgentRole = yamlConfig.USPProtocol.AgentRole
	config.USPCommandKey = yamlConfig.USPProtocol.CommandKey

	// MTP Configuration
	config.MTPType = yamlConfig.MTP.DefaultType

	// WebSocket MTP
	config.WebSocketURL = yamlConfig.MTP.WebSocket.URL
	config.WebSocketSubprotocol = yamlConfig.MTP.WebSocket.Subprotocol
	if yamlConfig.MTP.WebSocket.PingInterval != "" {
		if duration, err := time.ParseDuration(yamlConfig.MTP.WebSocket.PingInterval); err == nil {
			config.WebSocketPingInterval = duration
		}
	}
	if yamlConfig.MTP.WebSocket.PongTimeout != "" {
		if duration, err := time.ParseDuration(yamlConfig.MTP.WebSocket.PongTimeout); err == nil {
			config.WebSocketPongTimeout = duration
		}
	}
	if yamlConfig.MTP.WebSocket.ReconnectInterval != "" {
		if duration, err := time.ParseDuration(yamlConfig.MTP.WebSocket.ReconnectInterval); err == nil {
			config.WebSocketReconnectInterval = duration
		}
	}
	config.WebSocketMaxReconnectAttempts = yamlConfig.MTP.WebSocket.MaxReconnectAttempts

	// MQTT MTP
	config.MQTTBrokerURL = yamlConfig.MTP.MQTT.BrokerURL
	config.MQTTClientID = yamlConfig.MTP.MQTT.ClientID
	config.MQTTUsername = yamlConfig.MTP.MQTT.Username
	config.MQTTPassword = yamlConfig.MTP.MQTT.Password
	config.MQTTTopicRequest = yamlConfig.MTP.MQTT.TopicRequest
	config.MQTTTopicResponse = yamlConfig.MTP.MQTT.TopicResponse
	config.MQTTQOS = yamlConfig.MTP.MQTT.QOS
	config.MQTTRetain = yamlConfig.MTP.MQTT.Retain
	config.MQTTCleanSession = yamlConfig.MTP.MQTT.CleanSession
	if yamlConfig.MTP.MQTT.KeepAlive != "" {
		if duration, err := time.ParseDuration(yamlConfig.MTP.MQTT.KeepAlive); err == nil {
			config.MQTTKeepAlive = duration
		}
	}

	// STOMP MTP
	config.STOMPBrokerURL = yamlConfig.MTP.STOMP.BrokerURL
	config.STOMPUsername = yamlConfig.MTP.STOMP.Username
	config.STOMPPassword = yamlConfig.MTP.STOMP.Password
	config.STOMPDestinationRequest = yamlConfig.MTP.STOMP.DestinationRequest
	config.STOMPDestinationResponse = yamlConfig.MTP.STOMP.DestinationResponse
	if yamlConfig.MTP.STOMP.HeartbeatSend != "" {
		if duration, err := time.ParseDuration(yamlConfig.MTP.STOMP.HeartbeatSend); err == nil {
			config.STOMPHeartbeatSend = duration
		}
	}
	if yamlConfig.MTP.STOMP.HeartbeatReceive != "" {
		if duration, err := time.ParseDuration(yamlConfig.MTP.STOMP.HeartbeatReceive); err == nil {
			config.STOMPHeartbeatReceive = duration
		}
	}

	// Unix Domain Socket MTP
	config.UDSSocketPath = yamlConfig.MTP.UDS.SocketPath
	config.UDSSocketMode = yamlConfig.MTP.UDS.SocketMode

	// Agent Behavior
	config.AutoRegister = yamlConfig.AgentBehavior.AutoRegister
	config.PeriodicInformEnabled = yamlConfig.AgentBehavior.PeriodicInformEnabled
	if yamlConfig.AgentBehavior.PeriodicInformInterval != "" {
		if duration, err := time.ParseDuration(yamlConfig.AgentBehavior.PeriodicInformInterval); err == nil {
			config.PeriodicInformInterval = duration
		}
	}
	config.ParameterUpdateNotification = yamlConfig.AgentBehavior.ParameterUpdateNotification
	config.EventNotification = yamlConfig.AgentBehavior.EventNotification
	config.DiagnosticMode = yamlConfig.AgentBehavior.DiagnosticMode

	// Consul Integration
	config.ServiceDiscoveryEnabled = yamlConfig.ConsulIntegration.ServiceDiscoveryEnabled
	if yamlConfig.ConsulIntegration.HealthCheckInterval != "" {
		if duration, err := time.ParseDuration(yamlConfig.ConsulIntegration.HealthCheckInterval); err == nil {
			config.HealthCheckInterval = duration
		}
	}
	config.ServiceTags = yamlConfig.ConsulIntegration.ServiceTags

	// Security
	config.TLSEnabled = yamlConfig.Security.TLSEnabled
	config.TLSCertFile = yamlConfig.Security.TLSCertFile
	config.TLSKeyFile = yamlConfig.Security.TLSKeyFile
	config.TLSCAFile = yamlConfig.Security.TLSCAFile
	config.TLSSkipVerify = yamlConfig.Security.TLSSkipVerify

	// Logging
	config.LogLevel = yamlConfig.Logging.Level
	config.LogFormat = yamlConfig.Logging.Format
	config.LogOutput = yamlConfig.Logging.Output
	config.LogFile = yamlConfig.Logging.File

	// Performance
	config.BufferSize = yamlConfig.Performance.BufferSize
	config.MaxMessageSize = yamlConfig.Performance.MaxMessageSize
	if yamlConfig.Performance.ConnectionTimeout != "" {
		if duration, err := time.ParseDuration(yamlConfig.Performance.ConnectionTimeout); err == nil {
			config.ConnectionTimeout = duration
		}
	}
	if yamlConfig.Performance.ReadTimeout != "" {
		if duration, err := time.ParseDuration(yamlConfig.Performance.ReadTimeout); err == nil {
			config.ReadTimeout = duration
		}
	}
	if yamlConfig.Performance.WriteTimeout != "" {
		if duration, err := time.ParseDuration(yamlConfig.Performance.WriteTimeout); err == nil {
			config.WriteTimeout = duration
		}
	}
	if yamlConfig.Performance.IdleTimeout != "" {
		if duration, err := time.ParseDuration(yamlConfig.Performance.IdleTimeout); err == nil {
			config.IdleTimeout = duration
		}
	}

	// Development
	config.DevelopmentMode = yamlConfig.Development.Mode
	config.MockDataEnabled = yamlConfig.Development.MockDataEnabled
	config.TestMode = yamlConfig.Development.TestMode
	config.VerboseLogging = yamlConfig.Development.VerboseLogging

	return config, nil
}

// LoadYAMLTR069Config loads TR-069 agent configuration from YAML file
func LoadYAMLTR069Config(configPath string) (*TR069Config, error) {
	yamlConfig := &YAMLTR069Config{}

	// Read YAML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML config file: %v", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %v", err)
	}

	// Convert to TR069Config
	config := &TR069Config{}

	// Service Discovery
	config.ConsulEnabled = yamlConfig.ServiceDiscovery.ConsulEnabled
	config.ConsulAddr = yamlConfig.ServiceDiscovery.ConsulAddr
	config.ConsulDatacenter = yamlConfig.ServiceDiscovery.ConsulDatacenter

	// Device Information
	config.EndpointID = yamlConfig.DeviceInfo.EndpointID
	config.ProductClass = yamlConfig.DeviceInfo.ProductClass
	config.Manufacturer = yamlConfig.DeviceInfo.Manufacturer
	config.ModelName = yamlConfig.DeviceInfo.ModelName
	config.SerialNumber = yamlConfig.DeviceInfo.SerialNumber
	config.SoftwareVersion = yamlConfig.DeviceInfo.SoftwareVersion
	config.HardwareVersion = yamlConfig.DeviceInfo.HardwareVersion
	config.DeviceType = yamlConfig.DeviceInfo.DeviceType
	config.OUI = yamlConfig.DeviceInfo.OUI

	// Platform Integration
	config.PlatformURL = yamlConfig.Platform.URL
	config.APIVersion = yamlConfig.Platform.APIVersion
	config.RegistrationEndpoint = yamlConfig.Platform.RegistrationEndpoint
	if yamlConfig.Platform.HeartbeatInterval != "" {
		if duration, err := time.ParseDuration(yamlConfig.Platform.HeartbeatInterval); err == nil {
			config.HeartbeatInterval = duration
		}
	}

	// CWMP Protocol
	config.CWMPVersion = yamlConfig.CWMPProtocol.Version
	config.CWMPSupportedVersions = yamlConfig.CWMPProtocol.SupportedVersions
	config.CWMPAgentRole = yamlConfig.CWMPProtocol.AgentRole
	config.CWMPConnectionRequestAuth = yamlConfig.CWMPProtocol.ConnectionRequestAuth

	// ACS Configuration
	config.ACSURL = yamlConfig.ACS.URL
	config.ACSUsername = yamlConfig.ACS.Username
	config.ACSPassword = yamlConfig.ACS.Password
	config.ACSPeriodicInformEnabled = yamlConfig.ACS.PeriodicInformEnabled
	if yamlConfig.ACS.PeriodicInformInterval != "" {
		if duration, err := time.ParseDuration(yamlConfig.ACS.PeriodicInformInterval); err == nil {
			config.ACSPeriodicInformInterval = duration
		}
	}
	config.ACSParameterKey = yamlConfig.ACS.ParameterKey

	// Connection Request
	config.ConnectionRequestURL = yamlConfig.ConnectionRequest.URL
	config.ConnectionRequestUsername = yamlConfig.ConnectionRequest.Username
	config.ConnectionRequestPassword = yamlConfig.ConnectionRequest.Password

	// Consul Integration
	config.ServiceDiscoveryEnabled = yamlConfig.ConsulIntegration.ServiceDiscoveryEnabled
	if yamlConfig.ConsulIntegration.HealthCheckInterval != "" {
		if duration, err := time.ParseDuration(yamlConfig.ConsulIntegration.HealthCheckInterval); err == nil {
			config.HealthCheckInterval = duration
		}
	}
	config.ServiceTags = yamlConfig.ConsulIntegration.ServiceTags

	// Security
	config.TLSEnabled = yamlConfig.Security.TLSEnabled
	config.TLSCertFile = yamlConfig.Security.TLSCertFile
	config.TLSKeyFile = yamlConfig.Security.TLSKeyFile
	config.TLSCAFile = yamlConfig.Security.TLSCAFile
	config.TLSSkipVerify = yamlConfig.Security.TLSSkipVerify

	// Logging
	config.LogLevel = yamlConfig.Logging.Level
	config.LogFormat = yamlConfig.Logging.Format
	config.LogOutput = yamlConfig.Logging.Output
	config.LogFile = yamlConfig.Logging.File

	// Performance
	config.BufferSize = yamlConfig.Performance.BufferSize
	config.MaxMessageSize = yamlConfig.Performance.MaxMessageSize
	if yamlConfig.Performance.ConnectionTimeout != "" {
		if duration, err := time.ParseDuration(yamlConfig.Performance.ConnectionTimeout); err == nil {
			config.ConnectionTimeout = duration
		}
	}
	if yamlConfig.Performance.ReadTimeout != "" {
		if duration, err := time.ParseDuration(yamlConfig.Performance.ReadTimeout); err == nil {
			config.ReadTimeout = duration
		}
	}
	if yamlConfig.Performance.WriteTimeout != "" {
		if duration, err := time.ParseDuration(yamlConfig.Performance.WriteTimeout); err == nil {
			config.WriteTimeout = duration
		}
	}
	if yamlConfig.Performance.IdleTimeout != "" {
		if duration, err := time.ParseDuration(yamlConfig.Performance.IdleTimeout); err == nil {
			config.IdleTimeout = duration
		}
	}

	// Development
	config.DevelopmentMode = yamlConfig.Development.Mode
	config.MockDataEnabled = yamlConfig.Development.MockDataEnabled
	config.TestMode = yamlConfig.Development.TestMode
	config.VerboseLogging = yamlConfig.Development.VerboseLogging

	return config, nil
}

// LoadConfigFromYAML loads configuration from YAML file only (pure YAML-based configuration)
func LoadConfigFromYAML(protocol string, configDir string) (interface{}, error) {
	var yamlPath string

	switch strings.ToLower(protocol) {
	case "usp":
		yamlPath = filepath.Join(configDir, "usp-agent.yaml")
	case "cwmp":
		yamlPath = filepath.Join(configDir, "cwmp-agent.yaml")
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	// Load YAML configuration
	if _, err := os.Stat(yamlPath); err != nil {
		return nil, fmt.Errorf("YAML configuration file not found: %s", yamlPath)
	}

	switch strings.ToLower(protocol) {
	case "usp":
		return LoadYAMLTR369Config(yamlPath)
	case "cwmp":
		return LoadYAMLTR069Config(yamlPath)
	}

	return nil, fmt.Errorf("unsupported protocol: %s", protocol)
}
