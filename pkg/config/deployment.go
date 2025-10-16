package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DeploymentConfig holds unified configuration for all OpenUSP services
type DeploymentConfig struct {
	ServicePort         int           `json:"service_port"`
	ServiceName         string        `json:"service_name"`
	ServiceType         string        `json:"service_type"`
	DatabaseHost        string        `json:"database_host"`
	DatabasePort        int           `json:"database_port"`
	DatabaseName        string        `json:"database_name"`
	DatabaseUser        string        `json:"database_user"`
	DatabasePassword    string        `json:"database_password"`
	DatabaseSSLMode     string        `json:"database_ssl_mode"`
	LogLevel            string        `json:"log_level"`
	LogDir              string        `json:"log_dir"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	MTPConfig           *MTPConfig    `json:"mtp_config,omitempty"`
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *DeploymentConfig) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DatabaseHost, c.DatabasePort, c.DatabaseUser, c.DatabasePassword, c.DatabaseName, c.DatabaseSSLMode)
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// LoadMTPConfig loads MTP-specific configuration from environment variables
func LoadMTPConfig() *MTPConfig {
	// Load YAML config
	yamlPath := "configs/openusp.yml"
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to read %s: %v", yamlPath, err))
	}
	var unified struct {
		MTP struct {
			Transports struct {
				WebSocketEnabled bool `yaml:"websocket_enabled"`
				MQTTEnabled      bool `yaml:"mqtt_enabled"`
				STOMPEnabled     bool `yaml:"stomp_enabled"`
			} `yaml:"transports"`
			WebSocket struct {
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
		} `yaml:"mtp"`
		USPService struct {
			GRPCPort int    `yaml:"grpc_port"`
			Address  string `yaml:"address"`
		} `yaml:"usp_service"`
	}
	if err := yaml.Unmarshal(data, &unified); err != nil {
		panic(fmt.Sprintf("Failed to parse %s: %v", yamlPath, err))
	}
	return &MTPConfig{
		WebSocketEnabled: unified.MTP.Transports.WebSocketEnabled,
		MQTTEnabled:      unified.MTP.Transports.MQTTEnabled,
		STOMPEnabled:     unified.MTP.Transports.STOMPEnabled,
		GRPCPort:         unified.USPService.GRPCPort,
		Address:          unified.USPService.Address,
	}
}

// LoadDeploymentConfig loads configuration from environment variables with defaults
func LoadDeploymentConfig(serviceName, serviceType string, defaultPort int) *DeploymentConfig {
	// We intentionally keep this environment-first to avoid circular dependence on the heavy full config loader for lightweight services.
	// If the unified YAML is present, we can optionally read log level & dir for consistency.
	logLevel := getEnvString("OPENUSP_LOG_LEVEL", "info")
	logDir := getEnvString("OPENUSP_LOG_DIR", "logs")

	servicePort := getEnvInt(strings.ToUpper(strings.ReplaceAll(serviceName, "-", "_"))+"_PORT", defaultPort)
	// Fallback generic vars
	if servicePort == defaultPort {
		switch serviceType {
		case "data-service":
			servicePort = getEnvInt("OPENUSP_DATA_SERVICE_HTTP_PORT", defaultPort)
		case "api-gateway":
			servicePort = getEnvInt("OPENUSP_API_GATEWAY_PORT", defaultPort)
		case "mtp-service":
			servicePort = getEnvInt("OPENUSP_MTP_SERVICE_PORT", defaultPort)
		case "cwmp-service":
			servicePort = getEnvInt("OPENUSP_CWMP_SERVICE_PORT", defaultPort)
		case "usp-service":
			servicePort = getEnvInt("OPENUSP_USP_SERVICE_PORT", defaultPort)
		}
	}

	cfg := &DeploymentConfig{
		ServicePort:         servicePort,
		ServiceName:         serviceName,
		ServiceType:         serviceType,
		DatabaseHost:        getEnvString("OPENUSP_DB_HOST", "localhost"),
		DatabasePort:        getEnvInt("OPENUSP_DB_PORT", 5433),
		DatabaseName:        getEnvString("OPENUSP_DB_NAME", "openusp_db"),
		DatabaseUser:        getEnvString("OPENUSP_DB_USER", "openusp"),
		DatabasePassword:    getEnvString("OPENUSP_DB_PASSWORD", "openusp123"),
		DatabaseSSLMode:     getEnvString("OPENUSP_DB_SSLMODE", "disable"),
		LogLevel:            logLevel,
		LogDir:              logDir,
		HealthCheckInterval: getEnvDuration("OPENUSP_HEALTHCHECK_INTERVAL", 30*time.Second),
		HealthCheckTimeout:  getEnvDuration("OPENUSP_HEALTHCHECK_TIMEOUT", 5*time.Second),
	}
	return cfg
}
