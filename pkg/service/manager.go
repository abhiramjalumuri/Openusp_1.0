package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"openusp/pkg/config"
	"openusp/pkg/metrics"
)

// ServiceManager handles unified service lifecycle management
type ServiceManager struct {
	config   *config.DeploymentConfig
	metrics  *metrics.OpenUSPMetrics
	listener net.Listener
}

// ServiceOptions contains service-specific configuration
type ServiceOptions struct {
	RequiresDatabase bool
	RequiresgRPC     bool
	RequiresHTTP     bool
	DefaultHTTPPort  int
	DefaultgRPCPort  int
}

// NewServiceManager creates a new unified service manager
func NewServiceManager(serviceName, serviceType string, opts ServiceOptions) (*ServiceManager, error) {
	// Load configuration from environment
	defaultPort := opts.DefaultHTTPPort
	if opts.RequiresgRPC && !opts.RequiresHTTP {
		defaultPort = opts.DefaultgRPCPort
	}
	cfg := config.LoadDeploymentConfig(serviceName, serviceType, defaultPort)

	manager := &ServiceManager{
		config:  cfg,
		metrics: metrics.NewOpenUSPMetrics(serviceType),
	}

	// Ports are fixed via configuration; service discovery disabled
	return manager, nil
}

// RegisterService logs service startup with configured ports
func (sm *ServiceManager) RegisterService(ctx context.Context) error {
	log.Printf("🎯 Service starting: %s (%s) at localhost:%d",
		sm.config.ServiceName, sm.config.ServiceType, sm.config.ServicePort)
	return nil
}

// GetServicePort returns the primary port this service should use
func (sm *ServiceManager) GetServicePort() int { return sm.config.ServicePort }

// GetgRPCPort returns the gRPC port this service should use
func (sm *ServiceManager) GetgRPCPort() int { return sm.config.ServicePort + 1 }

// CreateListener creates a network listener for the service
func (sm *ServiceManager) CreateListener() (net.Listener, error) {
	addr := fmt.Sprintf(":%d", sm.config.ServicePort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	sm.listener = listener
	return listener, nil
}

// DiscoverService returns information about another service using configured ports
func (sm *ServiceManager) DiscoverService(ctx context.Context, serviceName string, timeout time.Duration) (*ServiceInfo, error) {
	return sm.getServiceInfo(serviceName), nil
}

// ServiceInfo contains resolved service addressing
type ServiceInfo struct {
	Name     string
	Address  string
	Port     int
	GRPCPort int
	Health   string
}

// getServiceInfo returns service info using environment-based configuration with defaults
func (sm *ServiceManager) getServiceInfo(serviceName string) *ServiceInfo {

	// Map service names to their environment-configurable ports
	serviceMap := map[string]*ServiceInfo{
		"openusp-data-service": {
			Name:     "openusp-data-service",
			Address:  "localhost",
			Port:     6400,
			GRPCPort: 50100,
			Health:   "passing",
		},
		"openusp-api-gateway": {
			Name:    "openusp-api-gateway",
			Address: "localhost",
			Port:    6500,
			Health:  "passing",
		},
		"openusp-mtp-service": {
			Name:     "openusp-mtp-service",
			Address:  "localhost",
			Port:     8081,
			GRPCPort: 50300,
			Health:   "passing",
		},
		"openusp-cwmp-service": {
			Name:     "openusp-cwmp-service",
			Address:  "localhost",
			Port:     7547,
			GRPCPort: 0, // CWMP service doesn't provide gRPC
			Health:   "passing",
		},
		"openusp-usp-service": {
			Name:     "openusp-usp-service",
			Address:  "localhost",
			Port:     6250,
			GRPCPort: 50200,
			Health:   "passing",
		},
		"openusp-connection-manager": {
			Name:     "openusp-connection-manager",
			Address:  "localhost",
			Port:     6200,
			GRPCPort: 50400,
			Health:   "passing",
		},
	}

	if service, exists := serviceMap[serviceName]; exists {
		return service
	}

	// Return default service info for unknown services
	return &ServiceInfo{
		Name:    serviceName,
		Address: "localhost",
		Port:    8080,
		Health:  "unknown",
	}
}

// GetConfig returns the deployment configuration
func (sm *ServiceManager) GetConfig() *config.DeploymentConfig {
	return sm.config
}

// GetMetrics returns the metrics instance
func (sm *ServiceManager) GetMetrics() *metrics.OpenUSPMetrics {
	return sm.metrics
}

// GetServiceInfo returns information about this service (self) using configured ports
func (sm *ServiceManager) GetServiceInfo() *ServiceInfo {
	return sm.getServiceInfo(sm.config.ServiceName)
}

// IsConsulEnabled returns whether Consul is enabled (always false now)
func (sm *ServiceManager) IsConsulEnabled() bool { return false }

// Shutdown gracefully shuts down the service
func (sm *ServiceManager) Shutdown() error {
	log.Printf("🛑 Shutting down service...")

	// Close listener
	if sm.listener != nil {
		sm.listener.Close()
	}

	log.Printf("✅ Service stopped successfully")
	return nil
}
