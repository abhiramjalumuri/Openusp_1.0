package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DeploymentConfig holds unified configuration for all OpenUSP services
type DeploymentConfig struct {
	// Service Discovery Configuration
	ConsulEnabled bool   `json:"consul_enabled"`
	ConsulAddr    string `json:"consul_addr"`

	// Service Configuration
	ServicePort int    `json:"service_port"`
	ServiceName string `json:"service_name"`
	ServiceType string `json:"service_type"`

	// Database Configuration (for services that need it)
	DatabaseHost     string `json:"database_host"`
	DatabasePort     int    `json:"database_port"`
	DatabaseName     string `json:"database_name"`
	DatabaseUser     string `json:"database_user"`
	DatabasePassword string `json:"database_password"`
	DatabaseSSLMode  string `json:"database_ssl_mode"`

	// Logging Configuration
	LogLevel string `json:"log_level"`
	LogDir   string `json:"log_dir"`

	// Health Check Configuration
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
}

// LoadDeploymentConfig loads configuration from environment variables with defaults
func LoadDeploymentConfig(serviceName, serviceType string, defaultPort int) *DeploymentConfig {
	return LoadDeploymentConfigWithPortEnv(serviceName, serviceType, defaultPort, "SERVICE_PORT")
}

// LoadDeploymentConfigWithPortEnv loads configuration with custom port environment variable
func LoadDeploymentConfigWithPortEnv(serviceName, serviceType string, defaultPort int, portEnvVar string) *DeploymentConfig {
	config := &DeploymentConfig{
		// Service Discovery defaults (enabled by default for development)
		ConsulEnabled: getEnvBool("CONSUL_ENABLED", true),
		ConsulAddr:    getEnvString("CONSUL_ADDR", "localhost:8500"),

		// Service defaults
		ServicePort: getEnvInt(portEnvVar, defaultPort),
		ServiceName: serviceName,
		ServiceType: serviceType,

		// Database defaults
		DatabaseHost:     getEnvString("OPENUSP_DB_HOST", "localhost"),
		DatabasePort:     getEnvInt("OPENUSP_DB_PORT", 5433),
		DatabaseName:     getEnvString("OPENUSP_DB_NAME", "openusp_db"),
		DatabaseUser:     getEnvString("OPENUSP_DB_USER", "openusp"),
		DatabasePassword: getEnvString("OPENUSP_DB_PASSWORD", "openusp123"),
		DatabaseSSLMode:  getEnvString("OPENUSP_DB_SSLMODE", "disable"),

		// Logging defaults
		LogLevel: getEnvString("LOG_LEVEL", "info"),
		LogDir:   getEnvString("OPENUSP_LOG_DIR", "./logs"),

		// Health check defaults
		HealthCheckInterval: getEnvDuration("HEALTH_CHECK_INTERVAL", 10*time.Second),
		HealthCheckTimeout:  getEnvDuration("HEALTH_CHECK_TIMEOUT", 5*time.Second),
	}

	return config
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *DeploymentConfig) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DatabaseHost, c.DatabasePort, c.DatabaseUser, c.DatabasePassword, c.DatabaseName, c.DatabaseSSLMode)
}

// IsConsulEnabled returns whether Consul service discovery is enabled
func (c *DeploymentConfig) IsConsulEnabled() bool {
	return c.ConsulEnabled
}

// GetConsulConfig returns Consul-specific configuration
func (c *DeploymentConfig) GetConsulConfig() (string, time.Duration, time.Duration) {
	return c.ConsulAddr, c.HealthCheckInterval, c.HealthCheckTimeout
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
