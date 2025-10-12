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
			MTPService  int `yaml:"mtp_service"`
			USPService  int `yaml:"usp_service"`
			CWMPService int `yaml:"cwmp_service"`
			DataService int `yaml:"data_service"`
		} `yaml:"http"`
		GRPC struct {
			DataService       int `yaml:"data_service"`
			USPService        int `yaml:"usp_service"`
			MTPService        int `yaml:"mtp_service"`
			ConnectionManager int `yaml:"connection_manager"`
			CWMPService       int `yaml:"cwmp_service"`
		} `yaml:"grpc"`
	} `yaml:"ports"`

	Database struct {
		Host    string `yaml:"host"`
		Port    int    `yaml:"port"`
		Name    string `yaml:"name"`
		SSLMode string `yaml:"sslmode"`
	} `yaml:"database"`

	Infrastructure struct {
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
			Addr string `yaml:"addr"`
		} `yaml:"data_service"`
		APIGateway struct {
			URL string `yaml:"url"`
		} `yaml:"api_gateway"`
		MTPService struct {
			URL string `yaml:"url"`
		} `yaml:"mtp_service"`
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
			UDSEnabled       bool `yaml:"uds_enabled"`
		} `yaml:"transports"`
		Websocket struct {
			Subprotocol string `yaml:"subprotocol"`
		} `yaml:"websocket"`
		MQTT struct {
			BrokerURL string `yaml:"broker_url"`
		} `yaml:"mqtt"`
		STOMP struct {
			BrokerURL    string `yaml:"broker_url"`
			Destinations struct {
				Controller string `yaml:"controller"`
				Agent      string `yaml:"agent"`
				Broadcast  string `yaml:"broadcast"`
			} `yaml:"destinations"`
		} `yaml:"stomp"`
		UDS struct {
			SocketPath string `yaml:"socket_path"`
		} `yaml:"uds"`
	} `yaml:"mtp"`

	USP struct {
		VersionSupport string `yaml:"version_support"`
		EndpointID     string `yaml:"endpoint_id"`
		DefaultTimeout int    `yaml:"default_timeout"`
	} `yaml:"usp"`

	TR181 struct {
		SchemaPath        string `yaml:"schema_path"`
		TypesPath         string `yaml:"types_path"`
		ValidationEnabled bool   `yaml:"validation_enabled"`
	} `yaml:"tr181"`

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
	// Service Ports
	APIGatewayPort  string
	DataServicePort string
	MTPServicePort  string
	CWMPServicePort string
	USPServicePort  string

	// Database Configuration
	Database DatabaseConfig

	// Service Inter-communication
	DataServiceAddr string
	APIGatewayURL   string
	MTPServiceURL   string
	CWMPServiceURL  string

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

	// TR-181 Data Model
	TR181 TR181Config

	// Security
	Security SecurityConfig

	// CWMP Agent Device Metadata
	CWMPAgentDevice CWMPAgentDeviceConfig

	// USP Agent Device Metadata
	USPAgentDevice USPAgentDeviceConfig
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
}

// MTPConfig holds MTP transport configuration
type MTPConfig struct {
	WebSocketEnabled           bool
	MQTTEnabled                bool
	STOMPEnabled               bool
	UnixSocketEnabled          bool
	UnixSocketPath             string
	STOMPDestinationController string
	STOMPDestinationAgent      string
	STOMPDestinationBroadcast  string
}

// USPConfig holds USP protocol configuration
type USPConfig struct {
	VersionSupport []string
	EndpointID     string
	DefaultTimeout int
}

// TR181Config holds TR-181 data model configuration
type TR181Config struct {
	SchemaPath        string
	TypesPath         string
	ValidationEnabled bool
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
		// Service Ports - Environment variables override YAML
		APIGatewayPort:  getEnvWithYAMLFallback("OPENUSP_API_GATEWAY_PORT", fmt.Sprintf("%d", yamlConfig.Ports.HTTP.APIGateway), "6500"),
		DataServicePort: getEnvWithYAMLFallback("OPENUSP_DATA_SERVICE_GRPC_PORT", fmt.Sprintf("%d", yamlConfig.Ports.GRPC.DataService), "50100"),
		MTPServicePort:  getEnvWithYAMLFallback("OPENUSP_MTP_SERVICE_PORT", fmt.Sprintf("%d", yamlConfig.Ports.HTTP.MTPService), "8081"),
		CWMPServicePort: getEnvWithYAMLFallback("OPENUSP_CWMP_SERVICE_PORT", fmt.Sprintf("%d", yamlConfig.Ports.HTTP.CWMPService), "7547"),
		USPServicePort:  getEnvWithYAMLFallback("OPENUSP_USP_SERVICE_GRPC_PORT", fmt.Sprintf("%d", yamlConfig.Ports.GRPC.USPService), "50200"),

		// Database Configuration
		Database: DatabaseConfig{
			Host:     getEnvWithYAMLFallback("OPENUSP_DB_HOST", yamlConfig.Database.Host, "localhost"),
			Port:     getEnvWithYAMLFallback("OPENUSP_DB_PORT", fmt.Sprintf("%d", yamlConfig.Database.Port), "5433"),
			Name:     getEnvWithYAMLFallback("OPENUSP_DB_NAME", yamlConfig.Database.Name, "openusp_db"),
			User:     getEnv("OPENUSP_DB_USER", "openusp"),        // Credentials stay in env only
			Password: getEnv("OPENUSP_DB_PASSWORD", "openusp123"), // Credentials stay in env only
			SSLMode:  getEnvWithYAMLFallback("OPENUSP_DB_SSLMODE", yamlConfig.Database.SSLMode, "disable"),
		},

		// Service Inter-communication
		DataServiceAddr: getEnvWithYAMLFallback("OPENUSP_DATA_SERVICE_ADDR", yamlConfig.Services.DataService.Addr, "localhost:50100"),
		APIGatewayURL:   getEnvWithYAMLFallback("OPENUSP_API_GATEWAY_URL", yamlConfig.Services.APIGateway.URL, "http://localhost:6500"),
		MTPServiceURL:   getEnvWithYAMLFallback("OPENUSP_MTP_SERVICE_URL", yamlConfig.Services.MTPService.URL, "http://localhost:8081"),
		CWMPServiceURL:  getEnvWithYAMLFallback("OPENUSP_CWMP_SERVICE_URL", yamlConfig.Services.CWMPService.URL, "http://localhost:7547"),

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
		},

		// Development/Debug
		LogLevel:      getEnvWithYAMLFallback("OPENUSP_LOG_LEVEL", yamlConfig.Development.LogLevel, "info"),
		LogDir:        getEnvWithYAMLFallback("OPENUSP_LOG_DIR", yamlConfig.Development.LogDir, "logs"),
		EnableDebug:   getBoolEnvWithYAMLFallback("OPENUSP_ENABLE_DEBUG", yamlConfig.Development.EnableDebug, false),
		EnableMetrics: getBoolEnvWithYAMLFallback("OPENUSP_ENABLE_METRICS", yamlConfig.Development.EnableMetrics, true),

		// MTP Transport
		MTP: MTPConfig{
			WebSocketEnabled:           getBoolEnvWithYAMLFallback("OPENUSP_MTP_WEBSOCKET_ENABLED", yamlConfig.MTP.Transports.WebsocketEnabled, true),
			MQTTEnabled:                getBoolEnvWithYAMLFallback("OPENUSP_MTP_MQTT_ENABLED", yamlConfig.MTP.Transports.MQTTEnabled, true),
			STOMPEnabled:               getBoolEnvWithYAMLFallback("OPENUSP_MTP_STOMP_ENABLED", yamlConfig.MTP.Transports.STOMPEnabled, true),
			UnixSocketEnabled:          getBoolEnvWithYAMLFallback("OPENUSP_MTP_UNIX_SOCKET_ENABLED", yamlConfig.MTP.Transports.UDSEnabled, true),
			UnixSocketPath:             getEnvWithYAMLFallback("OPENUSP_MTP_UNIX_SOCKET_PATH", yamlConfig.MTP.UDS.SocketPath, "/tmp/usp-agent.sock"),
			STOMPDestinationController: getEnvWithYAMLFallback("OPENUSP_MTP_STOMP_DESTINATION_CONTROLLER", yamlConfig.MTP.STOMP.Destinations.Controller, "/topic/usp.controller"),
			STOMPDestinationAgent:      getEnvWithYAMLFallback("OPENUSP_MTP_STOMP_DESTINATION_AGENT", yamlConfig.MTP.STOMP.Destinations.Agent, "/queue/usp.agent"),
			STOMPDestinationBroadcast:  getEnvWithYAMLFallback("OPENUSP_MTP_STOMP_DESTINATION_BROADCAST", yamlConfig.MTP.STOMP.Destinations.Broadcast, "/topic/usp.broadcast"),
		},

		// USP Protocol
		USP: USPConfig{
			VersionSupport: strings.Split(getEnvWithYAMLFallback("OPENUSP_USP_VERSION_SUPPORT", yamlConfig.USP.VersionSupport, "1.3,1.4"), ","),
			EndpointID:     getEnvWithYAMLFallback("OPENUSP_USP_ENDPOINT_ID", yamlConfig.USP.EndpointID, "proto::usp.controller"),
			DefaultTimeout: getIntEnvWithYAMLFallback("OPENUSP_USP_DEFAULT_TIMEOUT", yamlConfig.USP.DefaultTimeout, 30),
		},

		// TR-181 Data Model
		TR181: TR181Config{
			SchemaPath:        getEnvWithYAMLFallback("OPENUSP_TR181_SCHEMA_PATH", yamlConfig.TR181.SchemaPath, "pkg/datamodel/tr-181-2-19-1-usp-full.xml"),
			TypesPath:         getEnvWithYAMLFallback("OPENUSP_TR181_TYPES_PATH", yamlConfig.TR181.TypesPath, "pkg/datamodel/tr-106-types.xml"),
			ValidationEnabled: getBoolEnvWithYAMLFallback("OPENUSP_TR181_VALIDATION_ENABLED", yamlConfig.TR181.ValidationEnabled, false),
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

// GetDSN returns database connection string
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
