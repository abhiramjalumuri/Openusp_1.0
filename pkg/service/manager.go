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

	config := config.LoadDeploymentConfig(serviceName, serviceType, defaultPort)

	manager := &ServiceManager{
		config:  config,
		metrics: metrics.NewOpenUSPMetrics(serviceType),
	}

	// Static port configuration - no service discovery needed

	return manager, nil
}

// RegisterService logs service startup with static port configuration
func (sm *ServiceManager) RegisterService(ctx context.Context) error {
	// Static port deployment - log the configuration
	log.Printf("🎯 Service starting: %s (%s) at localhost:%d",
		sm.config.ServiceName, sm.config.ServiceType, sm.config.ServicePort)

	return nil
}

// GetServicePort returns the port this service should use
func (sm *ServiceManager) GetServicePort() int {
	return sm.config.ServicePort
}

// GetgRPCPort returns the gRPC port this service should use
func (sm *ServiceManager) GetgRPCPort() int {
	// Static port configuration: service port + 1
	return sm.config.ServicePort + 1
}

// CreateListener creates a network listener for the service
func (sm *ServiceManager) CreateListener() (net.Listener, error) {
	// Fixed port for static configuration
	addr := fmt.Sprintf(":%d", sm.config.ServicePort)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	sm.listener = listener
	return listener, nil
}

// DiscoverService discovers another service using static configuration
func (sm *ServiceManager) DiscoverService(ctx context.Context, serviceName string, timeout time.Duration) (*ServiceInfo, error) {
	// Use static service configuration from configs/services.yaml
	return sm.getStaticServiceInfo(serviceName), nil
}

// Simple ServiceInfo struct for static configuration
type ServiceInfo struct {
	Name     string
	Address  string
	Port     int
	GRPCPort int
	Health   string
}

// getStaticServiceInfo returns service info using static configuration
func (sm *ServiceManager) getStaticServiceInfo(serviceName string) *ServiceInfo {
	// Map service names to their static ports from configs/services.yaml
	serviceMap := map[string]*ServiceInfo{
		"openusp-data-service": {
			Name:     "openusp-data-service",
			Address:  "localhost",
			Port:     6100, // HTTP port
			GRPCPort: 6101, // gRPC port
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
			GRPCPort: 8082,
			Health:   "passing",
		},
		"openusp-cwmp-service": {
			Name:     "openusp-cwmp-service",
			Address:  "localhost",
			Port:     7547,
			GRPCPort: 7548,
			Health:   "passing",
		},
		"openusp-usp-service": {
			Name:     "openusp-usp-service",
			Address:  "localhost",
			Port:     6400,
			GRPCPort: 6401,
			Health:   "passing",
		},
		"openusp-connection-manager": {
			Name:     "openusp-connection-manager",
			Address:  "localhost",
			Port:     6200,
			GRPCPort: 6201,
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

// GetServiceInfo returns the service information using static configuration
func (sm *ServiceManager) GetServiceInfo() *ServiceInfo {
	return sm.getStaticServiceInfo(sm.config.ServiceName)
}

// IsConsulEnabled returns whether Consul is enabled (always false now)
func (sm *ServiceManager) IsConsulEnabled() bool {
	return false // Static port configuration - no service discovery
}

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
