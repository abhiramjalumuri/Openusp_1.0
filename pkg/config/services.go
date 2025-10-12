package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ServiceConfig represents the service configuration directory
type ServiceConfig struct {
	Services map[string]Service `yaml:"services"`
}

// Service represents a service with port configuration
type Service struct {
	Port       int      `yaml:"port"`
	HealthPort int      `yaml:"health_port"`
	GRPCPort   int      `yaml:"grpc_port"`
	Protocol   string   `yaml:"protocol"`
	Endpoints  []string `yaml:"endpoints"`
}

// serviceConfigInstance holds the loaded configuration
var serviceConfigInstance *ServiceConfig

// LoadServiceConfig loads the service configuration from file
func LoadServiceConfig(configPath string) error {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	config := &ServiceConfig{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	serviceConfigInstance = config
	return nil
}

// GetServiceConfig returns the current service configuration directory
func GetServiceConfig() *ServiceConfig {
	if serviceConfigInstance == nil {
		// Load default configuration if not loaded
		defaultConfig := &ServiceConfig{
			Services: map[string]Service{
				"api-gateway": {
					Port:       6500,
					HealthPort: 6501,
					GRPCPort:   6502,
					Protocol:   "http",
					Endpoints:  []string{"/health", "/metrics", "/swagger/index.html"},
				},
				"data-service": {
					Port:       6100,
					HealthPort: 6101,
					GRPCPort:   6102,
					Protocol:   "http",
					Endpoints:  []string{"/health", "/metrics", "/status"},
				},
				"connection-manager": {
					Port:       6200,
					HealthPort: 6201,
					GRPCPort:   6202,
					Protocol:   "http",
					Endpoints:  []string{"/health", "/metrics"},
				},
				"usp-service": {
					Port:       6400,
					HealthPort: 6401,
					GRPCPort:   6402,
					Protocol:   "http",
					Endpoints:  []string{"/health", "/metrics"},
				},
				"cwmp-service": {
					Port:       7547,
					HealthPort: 7548,
					GRPCPort:   7549,
					Protocol:   "http",
					Endpoints:  []string{"/health", "/metrics"},
				},
				"mtp-service": {
					Port:       8081,
					HealthPort: 8082,
					GRPCPort:   8083,
					Protocol:   "websocket",
					Endpoints:  []string{"/health", "/metrics", "/usp", "/ws"},
				},
			},
		}
		serviceConfigInstance = defaultConfig
	}
	return serviceConfigInstance
}

// GetServiceGRPCPort returns the gRPC port for a given service
func GetServiceGRPCPort(serviceName string) (int, error) {
	config := GetServiceConfig()

	service, exists := config.Services[serviceName]
	if !exists {
		return 0, fmt.Errorf("service %s not found in configuration", serviceName)
	}

	if service.GRPCPort == 0 {
		return 0, fmt.Errorf("gRPC port not configured for service %s", serviceName)
	}

	return service.GRPCPort, nil
}

// GetServicePort returns the main port for a given service
func GetServicePort(serviceName string) (int, error) {
	config := GetServiceConfig()

	service, exists := config.Services[serviceName]
	if !exists {
		return 0, fmt.Errorf("service %s not found in configuration", serviceName)
	}

	return service.Port, nil
}

// GetServiceHealthPort returns the health check port for a given service
func GetServiceHealthPort(serviceName string) (int, error) {
	config := GetServiceConfig()

	service, exists := config.Services[serviceName]
	if !exists {
		return 0, fmt.Errorf("service %s not found in configuration", serviceName)
	}

	return service.HealthPort, nil
}

// GetServiceGRPCTarget returns the gRPC target address for a service
func GetServiceGRPCTarget(serviceName string) (string, error) {
	port, err := GetServiceGRPCPort(serviceName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("localhost:%d", port), nil
}
