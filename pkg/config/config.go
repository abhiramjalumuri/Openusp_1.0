package config

import (
	"os"
	"strconv"
	"strings"
)

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
	WebSocketEnabled  bool
	MQTTEnabled       bool
	STOMPEnabled      bool
	UnixSocketEnabled bool
	UnixSocketPath    string
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

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		// Service Ports
		APIGatewayPort:  getEnv("OPENUSP_API_GATEWAY_PORT", "6500"),
		DataServicePort: getEnv("OPENUSP_DATA_SERVICE_GRPC_PORT", "6101"),
		MTPServicePort:  getEnv("OPENUSP_MTP_SERVICE_PORT", "6100"),
		CWMPServicePort: getEnv("OPENUSP_CWMP_SERVICE_PORT", "6300"),
		USPServicePort:  getEnv("OPENUSP_USP_SERVICE_GRPC_PORT", "56250"),

		// Database Configuration
		Database: DatabaseConfig{
			Host:     getEnv("OPENUSP_DB_HOST", "localhost"),
			Port:     getEnv("OPENUSP_DB_PORT", "5433"),
			Name:     getEnv("OPENUSP_DB_NAME", "openusp_db"),
			User:     getEnv("OPENUSP_DB_USER", "openusp"),
			Password: getEnv("OPENUSP_DB_PASSWORD", "openusp123"),
			SSLMode:  getEnv("OPENUSP_DB_SSLMODE", "disable"),
		},

		// Service Inter-communication
		DataServiceAddr: getEnv("OPENUSP_DATA_SERVICE_ADDR", "localhost:6101"),
		APIGatewayURL:   getEnv("OPENUSP_API_GATEWAY_URL", "https://localhost:6500"),
		MTPServiceURL:   getEnv("OPENUSP_MTP_SERVICE_URL", "http://localhost:6100"),
		CWMPServiceURL:  getEnv("OPENUSP_CWMP_SERVICE_URL", "http://localhost:6300"),

		// Protocol Clients
		USPWebSocketURL: getEnv("OPENUSP_USP_WS_URL", "ws://localhost:6100/ws"),
		CWMPACS: CWMPConfig{
			URL:      getEnv("OPENUSP_CWMP_ACS_URL", "http://localhost:7547"),
			Username: getEnv("OPENUSP_CWMP_USERNAME", "acs"),
			Password: getEnv("OPENUSP_CWMP_PASSWORD", "acs123"),
		},

		// Infrastructure Services
		Infrastructure: InfraConfig{
			PostgresHost:      getEnv("OPENUSP_POSTGRES_HOST", "localhost"),
			PostgresPort:      getEnv("OPENUSP_POSTGRES_PORT", "5433"),
			RabbitMQHost:      getEnv("OPENUSP_RABBITMQ_HOST", "localhost"),
			RabbitMQPort:      getEnv("OPENUSP_RABBITMQ_PORT", "5672"),
			RabbitMQMgmtPort:  getEnv("OPENUSP_RABBITMQ_MGMT_PORT", "15672"),
			RabbitMQSTOMPPort: getEnv("OPENUSP_RABBITMQ_STOMP_PORT", "61613"),
			RabbitMQUser:      getEnv("OPENUSP_RABBITMQ_USER", "admin"),
			RabbitMQPassword:  getEnv("OPENUSP_RABBITMQ_PASSWORD", "admin"),
			MosquittoHost:     getEnv("OPENUSP_MOSQUITTO_HOST", "localhost"),
			MosquittoPort:     getEnv("OPENUSP_MOSQUITTO_PORT", "1883"),
			MosquittoWSPort:   getEnv("OPENUSP_MOSQUITTO_WS_PORT", "9001"),
			PrometheusPort:    getEnv("OPENUSP_PROMETHEUS_PORT", "9090"),
			GrafanaPort:       getEnv("OPENUSP_GRAFANA_PORT", "3000"),
			GrafanaUser:       getEnv("OPENUSP_GRAFANA_USER", "admin"),
			GrafanaPassword:   getEnv("OPENUSP_GRAFANA_PASSWORD", "admin"),
			AdminerPort:       getEnv("OPENUSP_ADMINER_PORT", "8080"),
		},

		// Development/Debug
		LogLevel:      getEnv("OPENUSP_LOG_LEVEL", "info"),
		LogDir:        getEnv("OPENUSP_LOG_DIR", "logs"),
		EnableDebug:   getBoolEnv("OPENUSP_ENABLE_DEBUG", false),
		EnableMetrics: getBoolEnv("OPENUSP_ENABLE_METRICS", true),

		// MTP Transport
		MTP: MTPConfig{
			WebSocketEnabled:  getBoolEnv("OPENUSP_MTP_WEBSOCKET_ENABLED", true),
			MQTTEnabled:       getBoolEnv("OPENUSP_MTP_MQTT_ENABLED", true),
			STOMPEnabled:      getBoolEnv("OPENUSP_MTP_STOMP_ENABLED", true),
			UnixSocketEnabled: getBoolEnv("OPENUSP_MTP_UNIX_SOCKET_ENABLED", true),
			UnixSocketPath:    getEnv("OPENUSP_MTP_UNIX_SOCKET_PATH", "/tmp/usp-agent.sock"),
		},

		// USP Protocol
		USP: USPConfig{
			VersionSupport: strings.Split(getEnv("OPENUSP_USP_VERSION_SUPPORT", "1.3,1.4"), ","),
			EndpointID:     getEnv("OPENUSP_USP_ENDPOINT_ID", "proto::usp.controller"),
			DefaultTimeout: getIntEnv("OPENUSP_USP_DEFAULT_TIMEOUT", 30),
		},

		// TR-181 Data Model
		TR181: TR181Config{
			SchemaPath:        getEnv("OPENUSP_TR181_SCHEMA_PATH", "pkg/datamodel/tr-181-2-19-1-usp-full.xml"),
			TypesPath:         getEnv("OPENUSP_TR181_TYPES_PATH", "pkg/datamodel/tr-106-types.xml"),
			ValidationEnabled: getBoolEnv("OPENUSP_TR181_VALIDATION_ENABLED", true),
		},

		// Security
		Security: SecurityConfig{
			CORSEnabled:      getBoolEnv("OPENUSP_API_CORS_ENABLED", true),
			CORSOrigins:      getEnv("OPENUSP_API_CORS_ORIGINS", "*"),
			CWMPBasicAuth:    getBoolEnv("OPENUSP_CWMP_BASIC_AUTH_ENABLED", true),
			TLSEnabled:       getBoolEnv("OPENUSP_TLS_ENABLED", false),
			TLSCertPath:      getEnv("OPENUSP_TLS_CERT_PATH", ""),
			TLSKeyPath:       getEnv("OPENUSP_TLS_KEY_PATH", ""),
			HTTPRedirectPort: getIntEnv("OPENUSP_HTTP_REDIRECT_PORT", 6501),
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
