package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the structure of the openusp.yaml configuration file
type YAMLConfig struct {
	Ports struct {
		HTTP struct {
			APIGateway  int `yaml:"api_gateway"`
			USPService  int `yaml:"usp_service"`
			DataService int `yaml:"data_service"`
			// NOTE: cwmp_service HTTP port removed - TR-069 handled by mtp-http on port 7547
		} `yaml:"http"`
		Health struct {
			APIGateway   int `yaml:"api_gateway"`
			DataService  int `yaml:"data_service"`
			USPService   int `yaml:"usp_service"`
			CWMPService  int `yaml:"cwmp_service"`
			MTPStomp     int `yaml:"mtp_stomp"`
			MTPMqtt      int `yaml:"mtp_mqtt"`
			MTPWebsocket int `yaml:"mtp_websocket"`
			MTPHttp      int `yaml:"mtp_http"`
		} `yaml:"health"`
		Metrics struct {
			APIGateway   int `yaml:"api_gateway"`
			DataService  int `yaml:"data_service"`
			USPService   int `yaml:"usp_service"`
			CWMPService  int `yaml:"cwmp_service"`
			MTPStomp     int `yaml:"mtp_stomp"`
			MTPMqtt      int `yaml:"mtp_mqtt"`
			MTPWebsocket int `yaml:"mtp_websocket"`
			MTPHttp      int `yaml:"mtp_http"`
		} `yaml:"metrics"`
	} `yaml:"ports"`

	Kafka struct {
		Brokers  []string `yaml:"brokers"`
		Producer struct {
			Acks           string `yaml:"acks"`
			Compression    string `yaml:"compression"`
			MaxRetries     int    `yaml:"max_retries"`
			RetryBackoffMs int    `yaml:"retry_backoff_ms"`
			BatchSize      int    `yaml:"batch_size"`
			LingerMs       int    `yaml:"linger_ms"`
			BufferMemory   int    `yaml:"buffer_memory"`
		} `yaml:"producer"`
		Consumer struct {
			GroupIDPrefix        string `yaml:"group_id_prefix"`
			AutoOffsetReset      string `yaml:"auto_offset_reset"`
			EnableAutoCommit     bool   `yaml:"enable_auto_commit"`
			AutoCommitIntervalMs int    `yaml:"auto_commit_interval_ms"`
			SessionTimeoutMs     int    `yaml:"session_timeout_ms"`
			HeartbeatIntervalMs  int    `yaml:"heartbeat_interval_ms"`
			MaxPollIntervalMs    int    `yaml:"max_poll_interval_ms"`
			MaxPollRecords       int    `yaml:"max_poll_records"`
			FetchMinBytes        int    `yaml:"fetch_min_bytes"`
			FetchMaxWaitMs       int    `yaml:"fetch_max_wait_ms"`
		} `yaml:"consumer"`
		Topics struct {
			USPMessagesInbound       string `yaml:"usp_messages_inbound"`
			USPMessagesOutbound      string `yaml:"usp_messages_outbound"`
			USPRecordsParsed         string `yaml:"usp_records_parsed"`
			USPDataRequest           string `yaml:"usp_data_request"`
			USPDataResponse          string `yaml:"usp_data_response"`
			USPAPIRequest            string `yaml:"usp_api_request"`
			USPAPIResponse           string `yaml:"usp_api_response"`
			DataDeviceCreated        string `yaml:"data_device_created"`
			DataDeviceUpdated        string `yaml:"data_device_updated"`
			DataDeviceDeleted        string `yaml:"data_device_deleted"`
			DataParameterUpdated     string `yaml:"data_parameter_updated"`
			DataObjectCreated        string `yaml:"data_object_created"`
			DataAlertCreated         string `yaml:"data_alert_created"`
			CWMPMessagesInbound      string `yaml:"cwmp_messages_inbound"`
			CWMPMessagesOutbound     string `yaml:"cwmp_messages_outbound"`
			CWMPDataRequest          string `yaml:"cwmp_data_request"`
			CWMPDataResponse         string `yaml:"cwmp_data_response"`
			CWMPAPIRequest           string `yaml:"cwmp_api_request"`
			CWMPAPIResponse          string `yaml:"cwmp_api_response"`
			AgentStatusUpdated       string `yaml:"agent_status_updated"`
			AgentOnboardingStarted   string `yaml:"agent_onboarding_started"`
			AgentOnboardingCompleted string `yaml:"agent_onboarding_completed"`
			MTPConnectionEstablished string `yaml:"mtp_connection_established"`
			MTPConnectionClosed      string `yaml:"mtp_connection_closed"`
			MTPStompEvents           string `yaml:"mtp_stomp_events"`
			MTPMqttEvents            string `yaml:"mtp_mqtt_events"`
			MTPWebsocketEvents       string `yaml:"mtp_websocket_events"`
			APIRequest               string `yaml:"api_request"`
			APIResponse              string `yaml:"api_response"`
		} `yaml:"topics"`
		Admin struct {
			TimeoutMs         int `yaml:"timeout_ms"`
			NumPartitions     int `yaml:"num_partitions"`
			ReplicationFactor int `yaml:"replication_factor"`
		} `yaml:"admin"`
	} `yaml:"kafka"`

	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Name     string `yaml:"name"`
		SSLMode  string `yaml:"sslmode"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"database"`

	Infrastructure struct {
		Kafka struct {
			Host   string `yaml:"host"`
			Port   int    `yaml:"port"`
			UIPort int    `yaml:"ui_port"`
		} `yaml:"kafka"`
		Postgres struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			Database string `yaml:"database"`
		} `yaml:"postgres"`
		RabbitMQ struct {
			Host      string `yaml:"host"`
			Port      int    `yaml:"port"`
			MgmtPort  int    `yaml:"mgmt_port"`
			STOMPPort int    `yaml:"stomp_port"`
		} `yaml:"rabbitmq"`
		Mosquitto struct {
			Host          string `yaml:"host"`
			Port          int    `yaml:"port"`
			WebsocketPort int    `yaml:"websocket_port"`
		} `yaml:"mosquitto"`
		Prometheus struct {
			Port int `yaml:"port"`
		} `yaml:"prometheus"`
		Grafana struct {
			Port int `yaml:"port"`
		} `yaml:"grafana"`
		Adminer struct {
			Port int `yaml:"port"`
		} `yaml:"adminer"`
	} `yaml:"infrastructure"`

	Services struct {
		DataService struct {
			URL string `yaml:"url"`
		} `yaml:"data_service"`
		APIGateway struct {
			URL string `yaml:"url"`
		} `yaml:"api_gateway"`
		CWMPService struct {
			URL string `yaml:"url"`
		} `yaml:"cwmp_service"`
		USPService struct {
			URL string `yaml:"url"`
		} `yaml:"usp_service"`
	} `yaml:"services"`

	Protocols struct {
		USP struct {
			WebsocketURL string `yaml:"websocket_url"`
		} `yaml:"usp"`
		CWMP struct {
			ACSURL string `yaml:"acs_url"`
		} `yaml:"cwmp"`
	} `yaml:"protocols"`

	Development struct {
		LogLevel      string `yaml:"log_level"`
		LogDir        string `yaml:"log_dir"`
		EnableDebug   bool   `yaml:"enable_debug"`
		EnableMetrics bool   `yaml:"enable_metrics"`
	} `yaml:"development"`

	Docker struct {
		Network string `yaml:"network"`
	} `yaml:"docker"`

	MTP struct {
		Transports struct {
			WebsocketEnabled bool `yaml:"websocket_enabled"`
			MQTTEnabled      bool `yaml:"mqtt_enabled"`
			STOMPEnabled     bool `yaml:"stomp_enabled"`
		} `yaml:"transports"`
		Websocket struct {
			ServerPort  int    `yaml:"server_port"`
			Subprotocol string `yaml:"subprotocol"`
		} `yaml:"websocket"`
		MQTT struct {
			BrokerURL string `yaml:"broker_url"`
		} `yaml:"mqtt"`
		STOMP struct {
			BrokerURL    string `yaml:"broker_url"`
			Username     string `yaml:"username"`
			Password     string `yaml:"password"`
			Destinations struct {
				Inbound   string `yaml:"inbound"`
				Outbound  string `yaml:"outbound"`
				Broadcast string `yaml:"broadcast"`
			} `yaml:"destinations"`
		} `yaml:"stomp"`
		HTTP struct {
			ServerURL          string `yaml:"server_url"`
			Timeout            string `yaml:"timeout"`
			MaxIdleConnections int    `yaml:"max_idle_connections"`
			IdleTimeout        string `yaml:"idle_timeout"`
			TLSEnabled         bool   `yaml:"tls_enabled"`
			Username           string `yaml:"username"`
			Password           string `yaml:"password"`
		} `yaml:"http"`
	} `yaml:"mtp"`

	USP struct {
		VersionSupport string `yaml:"version_support"`
		EndpointID     string `yaml:"endpoint_id"`
		DefaultTimeout int    `yaml:"default_timeout"`
	} `yaml:"usp"`

	USPService struct {
		Kafka struct {
			ConsumerGroup string `yaml:"consumer_group"`
		} `yaml:"kafka"`
		Timeouts struct {
			Shutdown string `yaml:"shutdown"`
		} `yaml:"timeouts"`
		HTTP struct {
			ReadTimeout  string `yaml:"read_timeout"`
			WriteTimeout string `yaml:"write_timeout"`
			IdleTimeout  string `yaml:"idle_timeout"`
		} `yaml:"http"`
	} `yaml:"usp_service"`

	TR181 struct {
		SchemaPath        string `yaml:"schema_path"`
		TypesPath         string `yaml:"types_path"`
		ValidationEnabled bool   `yaml:"validation_enabled"`
	} `yaml:"tr181"`

	CWMPService struct {
		Authentication struct {
			Enabled  bool   `yaml:"enabled"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
		} `yaml:"authentication"`
		Timeouts struct {
			Connection string `yaml:"connection"`
			Session    string `yaml:"session"`
			Shutdown   string `yaml:"shutdown"`
		} `yaml:"timeouts"`
		Sessions struct {
			MaxConcurrent int `yaml:"max_concurrent"`
		} `yaml:"sessions"`
		Kafka struct {
			ConsumerGroup string `yaml:"consumer_group"`
		} `yaml:"kafka"`
	} `yaml:"cwmp_service"`

	Security struct {
		API struct {
			CORSEnabled bool   `yaml:"cors_enabled"`
			CORSOrigins string `yaml:"cors_origins"`
		} `yaml:"api"`
		CWMP struct {
			BasicAuthEnabled bool `yaml:"basic_auth_enabled"`
		} `yaml:"cwmp"`
	} `yaml:"security"`

	// CWMP Agent Configuration
	CWMpAgent struct {
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
	} `yaml:"cwmp_agent"`

	USPAgent struct {
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
	} `yaml:"usp_agent"`
}

// Config holds all OpenUSP configuration
type Config struct {
	// HTTP Service Ports (external API access)
	APIGatewayPort      string
	DataServiceHTTPPort string
	USPServiceHTTPPort  string
	// Note: CWMP service no longer uses HTTP port - TR-069 handled by mtp-http on port 7547

	// Health Check Ports
	APIGatewayHealthPort   int
	DataServiceHealthPort  int
	USPServiceHealthPort   int
	CWMPServiceHealthPort  int
	MTPStompHealthPort     int
	MTPMqttHealthPort      int
	MTPWebsocketHealthPort int
	MTPHttpHealthPort      int

	// Metrics Ports
	APIGatewayMetricsPort   int
	DataServiceMetricsPort  int
	USPServiceMetricsPort   int
	CWMPServiceMetricsPort  int
	MTPStompMetricsPort     int
	MTPMqttMetricsPort      int
	MTPWebsocketMetricsPort int
	MTPHttpMetricsPort      int

	// Kafka Configuration
	Kafka KafkaConfig

	// Database Configuration
	Database DatabaseConfig

	// Service URLs (HTTP/REST only)
	DataServiceURL string
	APIGatewayURL  string
	CWMPServiceURL string
	USPServiceURL  string

	// Protocol Clients
	USPWebSocketURL string
	CWMPACS         CWMPConfig

	// Infrastructure Services
	Infrastructure InfraConfig

	// Development/Debug
	LogLevel      string
	LogDir        string
	EnableDebug   bool
	EnableMetrics bool

	// MTP Transport
	MTP MTPConfig

	// USP Protocol
	USP USPConfig

	// USP Service Configuration
	USPService USPServiceConfig

	// TR-181 Data Model
	TR181 TR181Config

	// CWMP Service Configuration
	CWMPService CWMPServiceConfig

	// Security
	Security SecurityConfig

	// CWMP Agent Device Metadata
	CWMPAgentDevice CWMPAgentDeviceConfig

	// USP Agent Device Metadata
	USPAgentDevice USPAgentDeviceConfig
}

// KafkaConfig holds Kafka broker and messaging configuration
type KafkaConfig struct {
	Brokers        []string
	ProducerConfig KafkaProducerConfig
	ConsumerConfig KafkaConsumerConfig
	Topics         KafkaTopics
	AdminConfig    KafkaAdminConfig
}

// KafkaProducerConfig holds Kafka producer settings
type KafkaProducerConfig struct {
	Acks           string
	Compression    string
	MaxRetries     int
	RetryBackoffMs int
	BatchSize      int
	LingerMs       int
	BufferMemory   int
}

// KafkaConsumerConfig holds Kafka consumer settings
type KafkaConsumerConfig struct {
	GroupIDPrefix        string
	AutoOffsetReset      string
	EnableAutoCommit     bool
	AutoCommitIntervalMs int
	SessionTimeoutMs     int
	HeartbeatIntervalMs  int
	MaxPollIntervalMs    int
	MaxPollRecords       int
	FetchMinBytes        int
	FetchMaxWaitMs       int
}

// KafkaTopics holds all Kafka topic names
type KafkaTopics struct {
	USPMessagesInbound       string
	USPMessagesOutbound      string
	USPRecordsParsed         string
	USPDataRequest           string
	USPDataResponse          string
	USPAPIRequest            string
	USPAPIResponse           string
	DataDeviceCreated        string
	DataDeviceUpdated        string
	DataDeviceDeleted        string
	DataParameterUpdated     string
	DataObjectCreated        string
	DataAlertCreated         string
	CWMPMessagesInbound      string
	CWMPMessagesOutbound     string
	CWMPDataRequest          string
	CWMPDataResponse         string
	CWMPAPIRequest           string
	CWMPAPIResponse          string
	AgentStatusUpdated       string
	AgentOnboardingStarted   string
	AgentOnboardingCompleted string
	MTPConnectionEstablished string
	MTPConnectionClosed      string
	MTPStompEvents           string
	MTPMqttEvents            string
	MTPWebsocketEvents       string
	APIRequest               string
	APIResponse              string
}

// KafkaAdminConfig holds Kafka admin/management settings
type KafkaAdminConfig struct {
	TimeoutMs         int
	NumPartitions     int
	ReplicationFactor int
}

// CWMPAgentDeviceConfig holds CWMP agent device identity fields
type CWMPAgentDeviceConfig struct {
	EndpointID      string
	ProductClass    string
	Manufacturer    string
	ModelName       string
	SerialNumber    string
	SoftwareVersion string
	HardwareVersion string
	DeviceType      string
	OUI             string
}

// USPAgentDeviceConfig holds USP agent device identity fields
type USPAgentDeviceConfig struct {
	EndpointID      string
	ProductClass    string
	Manufacturer    string
	ModelName       string
	SerialNumber    string
	SoftwareVersion string
	HardwareVersion string
	DeviceType      string
	OUI             string
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// CWMPConfig holds CWMP/TR-069 configuration
type CWMPConfig struct {
	URL      string
	Username string
	Password string
}

// InfraConfig holds infrastructure service configuration
type InfraConfig struct {
	PostgresHost      string
	PostgresPort      string
	RabbitMQHost      string
	RabbitMQPort      string
	RabbitMQMgmtPort  string
	RabbitMQSTOMPPort string
	RabbitMQUser      string
	RabbitMQPassword  string
	MosquittoHost     string
	MosquittoPort     string
	MosquittoWSPort   string
	PrometheusPort    string
	GrafanaPort       string
	GrafanaUser       string
	GrafanaPassword   string
	AdminerPort       string
	KafkaHost         string
	KafkaPort         string
	KafkaUIPort       string
}

// MTPConfig holds MTP transport configuration
type MTPConfig struct {
	WebSocketEnabled bool
	MQTTEnabled      bool
	STOMPEnabled     bool
	HTTPEnabled      bool
	// Transport-specific configurations
	STOMP     STOMPTransportConfig
	MQTT      MQTTTransportConfig
	Websocket WebsocketTransportConfig
	HTTP      HTTPTransportConfig
	GRPCPort  int
	Address   string
}

// STOMPTransportConfig holds STOMP-specific configuration
type STOMPTransportConfig struct {
	BrokerURL    string
	Username     string
	Password     string
	Destinations struct {
		Inbound   string // messages from agents (inbound to controller)
		Outbound  string // messages to agents (outbound from controller)
		Broadcast string
	}
}

// MQTTTransportConfig holds MQTT-specific configuration
type MQTTTransportConfig struct {
	BrokerURL string
}

// WebsocketTransportConfig holds WebSocket-specific configuration
type WebsocketTransportConfig struct {
	ServerPort  int
	Subprotocol string
}

// HTTPTransportConfig holds HTTP/HTTPS-specific configuration
type HTTPTransportConfig struct {
	ServerURL          string
	Timeout            string
	MaxIdleConnections int
	IdleTimeout        string
	TLSEnabled         bool
	Username           string
	Password           string
}

// USPConfig holds USP protocol configuration
type USPConfig struct {
	VersionSupport []string
	EndpointID     string
	DefaultTimeout int
}

// USPServiceConfig holds USP service configuration
type USPServiceConfig struct {
	ConsumerGroup   string
	ShutdownTimeout string
	ReadTimeout     string
	WriteTimeout    string
	IdleTimeout     string
}

// TR181Config holds TR-181 data model configuration
type TR181Config struct {
	SchemaPath        string
	TypesPath         string
	ValidationEnabled bool
}

// CWMPServiceConfig holds CWMP service configuration
type CWMPServiceConfig struct {
	AuthenticationEnabled bool
	Username              string
	Password              string
	ConnectionTimeout     string
	SessionTimeout        string
	ShutdownTimeout       string
	MaxConcurrentSessions int
	ConsumerGroup         string
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	CORSEnabled      bool
	CORSOrigins      string
	CWMPBasicAuth    bool
	TLSEnabled       bool
	TLSCertPath      string
	TLSKeyPath       string
	HTTPRedirectPort int
}

// loadYAMLConfig loads configuration from YAML file
func loadYAMLConfig(configPath string) (*YAMLConfig, error) {
	if configPath == "" {
		configPath = "configs/openusp.yml"
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var yamlConfig YAMLConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &yamlConfig, nil
}

// Load loads configuration from YAML file with environment variable overrides
func Load() *Config {
	return LoadWithPath("")
}

// LoadWithPath loads configuration from specified YAML file with environment variable overrides
func LoadWithPath(configPath string) *Config {
	// Try to load YAML configuration
	yamlConfig, err := loadYAMLConfig(configPath)
	if err != nil {
		// If YAML loading fails, fall back to environment-only configuration
		fmt.Printf("Warning: Failed to load YAML config (%v), using environment variables only\n", err)
		yamlConfig = &YAMLConfig{} // Use empty YAML config
	}

	// Merge YAML config with environment variables (env vars take precedence)
	cfg := &Config{
		// Service Ports - YAML authoritative (error if missing)
		APIGatewayPort:      intToStringOrError(yamlConfig.Ports.HTTP.APIGateway, "api_gateway http port"),
		DataServiceHTTPPort: intToStringOrError(yamlConfig.Ports.HTTP.DataService, "data_service http port"),
		USPServiceHTTPPort:  intToStringOrError(yamlConfig.Ports.HTTP.USPService, "usp_service http port"),
		// NOTE: CWMP service no longer uses HTTP port - TR-069 handled by mtp-http on port 7547

		// Health Ports - direct from YAML, no defaults
		DataServiceHealthPort:  yamlConfig.Ports.Health.DataService,
		USPServiceHealthPort:   yamlConfig.Ports.Health.USPService,
		CWMPServiceHealthPort:  yamlConfig.Ports.Health.CWMPService,
		APIGatewayHealthPort:   yamlConfig.Ports.Health.APIGateway,
		MTPStompHealthPort:     yamlConfig.Ports.Health.MTPStomp,
		MTPMqttHealthPort:      yamlConfig.Ports.Health.MTPMqtt,
		MTPWebsocketHealthPort: yamlConfig.Ports.Health.MTPWebsocket,
		MTPHttpHealthPort:      yamlConfig.Ports.Health.MTPHttp,

		// Metrics Ports - direct from YAML, no defaults
		DataServiceMetricsPort:  yamlConfig.Ports.Metrics.DataService,
		USPServiceMetricsPort:   yamlConfig.Ports.Metrics.USPService,
		CWMPServiceMetricsPort:  yamlConfig.Ports.Metrics.CWMPService,
		APIGatewayMetricsPort:   yamlConfig.Ports.Metrics.APIGateway,
		MTPStompMetricsPort:     yamlConfig.Ports.Metrics.MTPStomp,
		MTPMqttMetricsPort:      yamlConfig.Ports.Metrics.MTPMqtt,
		MTPWebsocketMetricsPort: yamlConfig.Ports.Metrics.MTPWebsocket,
		MTPHttpMetricsPort:      yamlConfig.Ports.Metrics.MTPHttp, // Database Configuration
		Database: DatabaseConfig{
			Host:     yamlConfig.Database.Host,
			Port:     fmt.Sprintf("%d", yamlConfig.Database.Port),
			Name:     yamlConfig.Database.Name,
			User:     yamlConfig.Database.User,
			Password: yamlConfig.Database.Password,
			SSLMode:  yamlConfig.Database.SSLMode,
		},

		// Kafka Configuration
		Kafka: KafkaConfig{
			Brokers: yamlConfig.Kafka.Brokers,
			ProducerConfig: KafkaProducerConfig{
				Acks:           yamlConfig.Kafka.Producer.Acks,
				Compression:    yamlConfig.Kafka.Producer.Compression,
				MaxRetries:     yamlConfig.Kafka.Producer.MaxRetries,
				RetryBackoffMs: yamlConfig.Kafka.Producer.RetryBackoffMs,
				BatchSize:      yamlConfig.Kafka.Producer.BatchSize,
				LingerMs:       yamlConfig.Kafka.Producer.LingerMs,
				BufferMemory:   yamlConfig.Kafka.Producer.BufferMemory,
			},
			ConsumerConfig: KafkaConsumerConfig{
				GroupIDPrefix:        yamlConfig.Kafka.Consumer.GroupIDPrefix,
				AutoOffsetReset:      yamlConfig.Kafka.Consumer.AutoOffsetReset,
				EnableAutoCommit:     yamlConfig.Kafka.Consumer.EnableAutoCommit,
				AutoCommitIntervalMs: yamlConfig.Kafka.Consumer.AutoCommitIntervalMs,
				SessionTimeoutMs:     yamlConfig.Kafka.Consumer.SessionTimeoutMs,
				HeartbeatIntervalMs:  yamlConfig.Kafka.Consumer.HeartbeatIntervalMs,
				MaxPollIntervalMs:    yamlConfig.Kafka.Consumer.MaxPollIntervalMs,
				MaxPollRecords:       yamlConfig.Kafka.Consumer.MaxPollRecords,
				FetchMinBytes:        yamlConfig.Kafka.Consumer.FetchMinBytes,
				FetchMaxWaitMs:       yamlConfig.Kafka.Consumer.FetchMaxWaitMs,
			},
			Topics: KafkaTopics{
				USPMessagesInbound:       yamlConfig.Kafka.Topics.USPMessagesInbound,
				USPMessagesOutbound:      yamlConfig.Kafka.Topics.USPMessagesOutbound,
				USPRecordsParsed:         yamlConfig.Kafka.Topics.USPRecordsParsed,
				USPDataRequest:           yamlConfig.Kafka.Topics.USPDataRequest,
				USPDataResponse:          yamlConfig.Kafka.Topics.USPDataResponse,
				USPAPIRequest:            yamlConfig.Kafka.Topics.USPAPIRequest,
				USPAPIResponse:           yamlConfig.Kafka.Topics.USPAPIResponse,
				DataDeviceCreated:        yamlConfig.Kafka.Topics.DataDeviceCreated,
				DataDeviceUpdated:        yamlConfig.Kafka.Topics.DataDeviceUpdated,
				DataDeviceDeleted:        yamlConfig.Kafka.Topics.DataDeviceDeleted,
				DataParameterUpdated:     yamlConfig.Kafka.Topics.DataParameterUpdated,
				DataObjectCreated:        yamlConfig.Kafka.Topics.DataObjectCreated,
				DataAlertCreated:         yamlConfig.Kafka.Topics.DataAlertCreated,
				CWMPMessagesInbound:      yamlConfig.Kafka.Topics.CWMPMessagesInbound,
				CWMPMessagesOutbound:     yamlConfig.Kafka.Topics.CWMPMessagesOutbound,
				CWMPDataRequest:          yamlConfig.Kafka.Topics.CWMPDataRequest,
				CWMPDataResponse:         yamlConfig.Kafka.Topics.CWMPDataResponse,
				CWMPAPIRequest:           yamlConfig.Kafka.Topics.CWMPAPIRequest,
				CWMPAPIResponse:          yamlConfig.Kafka.Topics.CWMPAPIResponse,
				AgentStatusUpdated:       yamlConfig.Kafka.Topics.AgentStatusUpdated,
				AgentOnboardingStarted:   yamlConfig.Kafka.Topics.AgentOnboardingStarted,
				AgentOnboardingCompleted: yamlConfig.Kafka.Topics.AgentOnboardingCompleted,
				MTPConnectionEstablished: yamlConfig.Kafka.Topics.MTPConnectionEstablished,
				MTPConnectionClosed:      yamlConfig.Kafka.Topics.MTPConnectionClosed,
				MTPStompEvents:           yamlConfig.Kafka.Topics.MTPStompEvents,
				MTPMqttEvents:            yamlConfig.Kafka.Topics.MTPMqttEvents,
				MTPWebsocketEvents:       yamlConfig.Kafka.Topics.MTPWebsocketEvents,
				APIRequest:               yamlConfig.Kafka.Topics.APIRequest,
				APIResponse:              yamlConfig.Kafka.Topics.APIResponse,
			},
			AdminConfig: KafkaAdminConfig{
				TimeoutMs:         yamlConfig.Kafka.Admin.TimeoutMs,
				NumPartitions:     yamlConfig.Kafka.Admin.NumPartitions,
				ReplicationFactor: yamlConfig.Kafka.Admin.ReplicationFactor,
			},
		},

		// Service Inter-communication - YAML authoritative
		DataServiceURL: validateYAMLOrError(yamlConfig.Services.DataService.URL, "services.data_service.url"),
		APIGatewayURL:  validateYAMLOrError(yamlConfig.Services.APIGateway.URL, "services.api_gateway.url"),
		CWMPServiceURL: validateYAMLOrError(yamlConfig.Services.CWMPService.URL, "services.cwmp_service.url"),

		// Protocol Clients
		USPWebSocketURL: getEnvWithYAMLFallback("OPENUSP_USP_WS_URL", yamlConfig.Protocols.USP.WebsocketURL, "ws://localhost:8081/ws"),
		CWMPACS: CWMPConfig{
			URL:      getEnvWithYAMLFallback("OPENUSP_CWMP_ACS_URL", yamlConfig.Protocols.CWMP.ACSURL, "http://localhost:7547"),
			Username: getEnv("OPENUSP_CWMP_USERNAME", "acs"),    // Credentials stay in env only
			Password: getEnv("OPENUSP_CWMP_PASSWORD", "acs123"), // Credentials stay in env only
		},

		// Infrastructure Services
		Infrastructure: InfraConfig{
			PostgresHost:      getEnvWithYAMLFallback("OPENUSP_POSTGRES_HOST", yamlConfig.Infrastructure.Postgres.Host, "localhost"),
			PostgresPort:      getEnvWithYAMLFallback("OPENUSP_POSTGRES_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.Postgres.Port), "5433"),
			RabbitMQHost:      getEnvWithYAMLFallback("OPENUSP_RABBITMQ_HOST", yamlConfig.Infrastructure.RabbitMQ.Host, "localhost"),
			RabbitMQPort:      getEnvWithYAMLFallback("OPENUSP_RABBITMQ_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.RabbitMQ.Port), "5672"),
			RabbitMQMgmtPort:  getEnvWithYAMLFallback("OPENUSP_RABBITMQ_MGMT_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.RabbitMQ.MgmtPort), "15672"),
			RabbitMQSTOMPPort: getEnvWithYAMLFallback("OPENUSP_RABBITMQ_STOMP_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.RabbitMQ.STOMPPort), "61613"),
			RabbitMQUser:      getEnv("OPENUSP_RABBITMQ_USER", "openusp"),        // Credentials stay in env only
			RabbitMQPassword:  getEnv("OPENUSP_RABBITMQ_PASSWORD", "openusp123"), // Credentials stay in env only
			MosquittoHost:     getEnvWithYAMLFallback("OPENUSP_MOSQUITTO_HOST", yamlConfig.Infrastructure.Mosquitto.Host, "localhost"),
			MosquittoPort:     getEnvWithYAMLFallback("OPENUSP_MOSQUITTO_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.Mosquitto.Port), "1883"),
			MosquittoWSPort:   getEnvWithYAMLFallback("OPENUSP_MOSQUITTO_WS_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.Mosquitto.WebsocketPort), "9001"),
			PrometheusPort:    getEnvWithYAMLFallback("OPENUSP_PROMETHEUS_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.Prometheus.Port), "9090"),
			GrafanaPort:       getEnvWithYAMLFallback("OPENUSP_GRAFANA_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.Grafana.Port), "3000"),
			GrafanaUser:       getEnv("OPENUSP_GRAFANA_USER", "openusp"),        // Credentials stay in env only
			GrafanaPassword:   getEnv("OPENUSP_GRAFANA_PASSWORD", "openusp123"), // Credentials stay in env only
			AdminerPort:       getEnvWithYAMLFallback("OPENUSP_ADMINER_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.Adminer.Port), "8080"),
			KafkaHost:         getEnvWithYAMLFallback("OPENUSP_KAFKA_HOST", yamlConfig.Infrastructure.Kafka.Host, "localhost"),
			KafkaPort:         getEnvWithYAMLFallback("OPENUSP_KAFKA_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.Kafka.Port), "9092"),
			KafkaUIPort:       getEnvWithYAMLFallback("OPENUSP_KAFKA_UI_PORT", fmt.Sprintf("%d", yamlConfig.Infrastructure.Kafka.UIPort), "8082"),
		},

		// Development/Debug
		LogLevel:      getEnvWithYAMLFallback("OPENUSP_LOG_LEVEL", yamlConfig.Development.LogLevel, "info"),
		LogDir:        getEnvWithYAMLFallback("OPENUSP_LOG_DIR", yamlConfig.Development.LogDir, "logs"),
		EnableDebug:   getBoolEnvWithYAMLFallback("OPENUSP_ENABLE_DEBUG", yamlConfig.Development.EnableDebug, false),
		EnableMetrics: getBoolEnvWithYAMLFallback("OPENUSP_ENABLE_METRICS", yamlConfig.Development.EnableMetrics, true),

		// MTP Transport
		MTP: MTPConfig{
			WebSocketEnabled: getBoolEnvWithYAMLFallback("OPENUSP_MTP_WEBSOCKET_ENABLED", yamlConfig.MTP.Transports.WebsocketEnabled, true),
			MQTTEnabled:      getBoolEnvWithYAMLFallback("OPENUSP_MTP_MQTT_ENABLED", yamlConfig.MTP.Transports.MQTTEnabled, true),
			STOMPEnabled:     getBoolEnvWithYAMLFallback("OPENUSP_MTP_STOMP_ENABLED", yamlConfig.MTP.Transports.STOMPEnabled, true),
			// STOMP configuration from YAML
			STOMP: STOMPTransportConfig{
				BrokerURL: yamlConfig.MTP.STOMP.BrokerURL,
				Username:  yamlConfig.MTP.STOMP.Username,
				Password:  yamlConfig.MTP.STOMP.Password,
				Destinations: struct {
					Inbound   string
					Outbound  string
					Broadcast string
				}{
					Inbound:   yamlConfig.MTP.STOMP.Destinations.Inbound,
					Outbound:  yamlConfig.MTP.STOMP.Destinations.Outbound,
					Broadcast: yamlConfig.MTP.STOMP.Destinations.Broadcast,
				},
			},
			// MQTT configuration from YAML
			MQTT: MQTTTransportConfig{
				BrokerURL: yamlConfig.MTP.MQTT.BrokerURL,
			},
			// WebSocket configuration from YAML
			Websocket: WebsocketTransportConfig{
				ServerPort:  yamlConfig.MTP.Websocket.ServerPort,
				Subprotocol: yamlConfig.MTP.Websocket.Subprotocol,
			},
			// HTTP configuration from YAML
			HTTP: HTTPTransportConfig{
				ServerURL:          yamlConfig.MTP.HTTP.ServerURL,
				Timeout:            yamlConfig.MTP.HTTP.Timeout,
				MaxIdleConnections: yamlConfig.MTP.HTTP.MaxIdleConnections,
				IdleTimeout:        yamlConfig.MTP.HTTP.IdleTimeout,
				TLSEnabled:         yamlConfig.MTP.HTTP.TLSEnabled,
				Username:           yamlConfig.MTP.HTTP.Username,
				Password:           yamlConfig.MTP.HTTP.Password,
			},
		},

		// USP Protocol
		USP: USPConfig{
			VersionSupport: strings.Split(getEnvWithYAMLFallback("OPENUSP_USP_VERSION_SUPPORT", yamlConfig.USP.VersionSupport, "1.3,1.4"), ","),
			EndpointID:     getEnvWithYAMLFallback("OPENUSP_USP_ENDPOINT_ID", yamlConfig.USP.EndpointID, "proto::usp.controller"),
			DefaultTimeout: getIntEnvWithYAMLFallback("OPENUSP_USP_DEFAULT_TIMEOUT", yamlConfig.USP.DefaultTimeout, 30),
		},

		// USP Service Configuration
		USPService: USPServiceConfig{
			ConsumerGroup:   getEnvWithYAMLFallback("OPENUSP_USP_CONSUMER_GROUP", yamlConfig.USPService.Kafka.ConsumerGroup, "usp-service"),
			ShutdownTimeout: getEnvWithYAMLFallback("OPENUSP_USP_SHUTDOWN_TIMEOUT", yamlConfig.USPService.Timeouts.Shutdown, "30s"),
			ReadTimeout:     getEnvWithYAMLFallback("OPENUSP_USP_READ_TIMEOUT", yamlConfig.USPService.HTTP.ReadTimeout, "30s"),
			WriteTimeout:    getEnvWithYAMLFallback("OPENUSP_USP_WRITE_TIMEOUT", yamlConfig.USPService.HTTP.WriteTimeout, "30s"),
			IdleTimeout:     getEnvWithYAMLFallback("OPENUSP_USP_IDLE_TIMEOUT", yamlConfig.USPService.HTTP.IdleTimeout, "120s"),
		},

		// TR-181 Data Model
		TR181: TR181Config{
			SchemaPath:        getEnvWithYAMLFallback("OPENUSP_TR181_SCHEMA_PATH", yamlConfig.TR181.SchemaPath, "pkg/datamodel/tr-181-2-19-1-usp-full.xml"),
			TypesPath:         getEnvWithYAMLFallback("OPENUSP_TR181_TYPES_PATH", yamlConfig.TR181.TypesPath, "pkg/datamodel/tr-106-types.xml"),
			ValidationEnabled: getBoolEnvWithYAMLFallback("OPENUSP_TR181_VALIDATION_ENABLED", yamlConfig.TR181.ValidationEnabled, false),
		},

		// CWMP Service Configuration
		CWMPService: CWMPServiceConfig{
			AuthenticationEnabled: getBoolEnvWithYAMLFallback("OPENUSP_CWMP_AUTH_ENABLED", yamlConfig.CWMPService.Authentication.Enabled, true),
			Username:              getEnvWithYAMLFallback("OPENUSP_CWMP_USERNAME", yamlConfig.CWMPService.Authentication.Username, "acs"),
			Password:              getEnvWithYAMLFallback("OPENUSP_CWMP_PASSWORD", yamlConfig.CWMPService.Authentication.Password, "acs123"),
			ConnectionTimeout:     getEnvWithYAMLFallback("OPENUSP_CWMP_CONNECTION_TIMEOUT", yamlConfig.CWMPService.Timeouts.Connection, "30s"),
			SessionTimeout:        getEnvWithYAMLFallback("OPENUSP_CWMP_SESSION_TIMEOUT", yamlConfig.CWMPService.Timeouts.Session, "300s"),
			ShutdownTimeout:       getEnvWithYAMLFallback("OPENUSP_CWMP_SHUTDOWN_TIMEOUT", yamlConfig.CWMPService.Timeouts.Shutdown, "30s"),
			MaxConcurrentSessions: getIntEnvWithYAMLFallback("OPENUSP_CWMP_MAX_SESSIONS", yamlConfig.CWMPService.Sessions.MaxConcurrent, 100),
			ConsumerGroup:         getEnvWithYAMLFallback("OPENUSP_CWMP_CONSUMER_GROUP", yamlConfig.CWMPService.Kafka.ConsumerGroup, "openusp-cwmp-service-group"),
		},

		// Security
		Security: SecurityConfig{
			CORSEnabled:      getBoolEnvWithYAMLFallback("OPENUSP_API_CORS_ENABLED", yamlConfig.Security.API.CORSEnabled, true),
			CORSOrigins:      getEnvWithYAMLFallback("OPENUSP_API_CORS_ORIGINS", yamlConfig.Security.API.CORSOrigins, "*"),
			CWMPBasicAuth:    getBoolEnvWithYAMLFallback("OPENUSP_CWMP_BASIC_AUTH_ENABLED", yamlConfig.Security.CWMP.BasicAuthEnabled, true),
			TLSEnabled:       getBoolEnv("OPENUSP_TLS_ENABLED", false),
			TLSCertPath:      getEnv("OPENUSP_TLS_CERT_PATH", ""),
			TLSKeyPath:       getEnv("OPENUSP_TLS_KEY_PATH", ""),
			HTTPRedirectPort: getIntEnv("OPENUSP_HTTP_REDIRECT_PORT", 6501),
		},

		// CWMP Agent Device Metadata (YAML authoritative; env fallback deprecated)
		CWMPAgentDevice: CWMPAgentDeviceConfig{
			EndpointID:      yamlConfig.CWMpAgent.Device.EndpointID,
			ProductClass:    yamlConfig.CWMpAgent.Device.ProductClass,
			Manufacturer:    yamlConfig.CWMpAgent.Device.Manufacturer,
			ModelName:       yamlConfig.CWMpAgent.Device.ModelName,
			SerialNumber:    yamlConfig.CWMpAgent.Device.SerialNumber,
			SoftwareVersion: yamlConfig.CWMpAgent.Device.SoftwareVersion,
			HardwareVersion: yamlConfig.CWMpAgent.Device.HardwareVersion,
			DeviceType:      yamlConfig.CWMpAgent.Device.DeviceType,
			OUI:             yamlConfig.CWMpAgent.Device.OUI,
		},

		// USP Agent Device Metadata (YAML authoritative; env fallback removed)
		USPAgentDevice: USPAgentDeviceConfig{
			EndpointID:      yamlConfig.USPAgent.Device.EndpointID,
			ProductClass:    yamlConfig.USPAgent.Device.ProductClass,
			Manufacturer:    yamlConfig.USPAgent.Device.Manufacturer,
			ModelName:       yamlConfig.USPAgent.Device.ModelName,
			SerialNumber:    yamlConfig.USPAgent.Device.SerialNumber,
			SoftwareVersion: yamlConfig.USPAgent.Device.SoftwareVersion,
			HardwareVersion: yamlConfig.USPAgent.Device.HardwareVersion,
			DeviceType:      yamlConfig.USPAgent.Device.DeviceType,
			OUI:             yamlConfig.USPAgent.Device.OUI,
		},
	}

	return cfg
}

// Validate validates that all critical configuration values are present
func (c *Config) Validate() error {
	if c.APIGatewayPort == "" || c.APIGatewayPort == "0" {
		return fmt.Errorf("missing api_gateway http port in openusp.yml")
	}
	if c.DataServiceHTTPPort == "" || c.DataServiceHTTPPort == "0" {
		return fmt.Errorf("missing data_service http port in openusp.yml")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("missing database.host in openusp.yml")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("missing database.name in openusp.yml")
	}
	if c.Database.User == "" {
		return fmt.Errorf("missing database.user in openusp.yml")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("missing database.password in openusp.yml")
	}
	if c.DataServiceURL == "" {
		return fmt.Errorf("missing services.data_service.url in openusp.yml")
	}

	// Validate Kafka configuration
	if len(c.Kafka.Brokers) == 0 {
		return fmt.Errorf("missing kafka.brokers in openusp.yml")
	}
	if c.Kafka.Topics.USPMessagesInbound == "" {
		return fmt.Errorf("missing kafka.topics.usp_messages_inbound in openusp.yml")
	}
	if c.Kafka.Topics.USPMessagesOutbound == "" {
		return fmt.Errorf("missing kafka.topics.usp_messages_outbound in openusp.yml")
	}

	return nil
}

// intToStringOrError converts a port int to string, panicking if zero (indicating missing YAML value)
func intToStringOrError(v int, label string) string {
	if v == 0 {
		panic(fmt.Sprintf("configuration error: missing %s in openusp.yml", label))
	}
	return fmt.Sprintf("%d", v)
}

// validateYAMLOrError returns the YAML value, panicking if empty (indicating missing YAML value)
func validateYAMLOrError(v, label string) string {
	if v == "" {
		panic(fmt.Sprintf("configuration error: missing %s in openusp.yml", label))
	}
	return v
} // GetDSN returns database connection string
func (c *Config) GetDSN() string {
	return "host=" + c.Database.Host +
		" port=" + c.Database.Port +
		" user=" + c.Database.User +
		" password=" + c.Database.Password +
		" dbname=" + c.Database.Name +
		" sslmode=" + c.Database.SSLMode
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

// getBoolEnv gets boolean environment variable with default value
func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

// getIntEnv gets integer environment variable with default value
func getIntEnv(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// getEnvWithYAMLFallback gets environment variable with YAML fallback, then default fallback
func getEnvWithYAMLFallback(envKey, yamlValue, defaultValue string) string {
	envValue := strings.TrimSpace(os.Getenv(envKey))
	if envValue != "" {
		return envValue
	}
	if yamlValue != "" && yamlValue != "0" {
		return yamlValue
	}
	return defaultValue
}

// getBoolEnvWithYAMLFallback gets boolean environment variable with YAML fallback, then default fallback
func getBoolEnvWithYAMLFallback(envKey string, yamlValue, defaultValue bool) bool {
	envValue := os.Getenv(envKey)
	if envValue != "" {
		boolValue, err := strconv.ParseBool(envValue)
		if err == nil {
			return boolValue
		}
	}
	// If YAML value is provided and not the default false value, use it
	if yamlValue != false {
		return yamlValue
	}
	return defaultValue
}

// getIntEnvWithYAMLFallback gets integer environment variable with YAML fallback, then default fallback
func getIntEnvWithYAMLFallback(envKey string, yamlValue, defaultValue int) int {
	envValue := os.Getenv(envKey)
	if envValue != "" {
		intValue, err := strconv.Atoi(envValue)
		if err == nil {
			return intValue
		}
	}
	if yamlValue != 0 {
		return yamlValue
	}
	return defaultValue
}
