package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"openusp/pkg/config"
	"openusp/pkg/consul"
	"openusp/pkg/metrics"
)

// ServiceManager handles unified service lifecycle management
type ServiceManager struct {
	config      *config.DeploymentConfig
	registry    *consul.ServiceRegistry
	serviceInfo *consul.ServiceInfo
	metrics     *metrics.OpenUSPMetrics
	listener    net.Listener
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

	// Initialize Consul if enabled
	if config.IsConsulEnabled() {
		consulAddr, interval, timeout := config.GetConsulConfig()
		consulConfig := &consul.Config{
			Address:       consulAddr,
			Datacenter:    "openusp-dev",
			CheckInterval: interval,
			CheckTimeout:  timeout,
		}

		registry, err := consul.NewServiceRegistry(consulConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Consul: %w", err)
		}
		manager.registry = registry

		log.Printf("🏛️ Connected to Consul at %s (datacenter: %s)", consulAddr, consulConfig.Datacenter)
	}

	return manager, nil
}

// RegisterService registers the service with appropriate discovery mechanism
func (sm *ServiceManager) RegisterService(ctx context.Context) error {
	if sm.registry != nil {
		// Register with Consul
		serviceInfo, err := sm.registry.RegisterService(ctx, sm.config.ServiceName, sm.config.ServiceType)
		if err != nil {
			return fmt.Errorf("failed to register with Consul: %w", err)
		}

		sm.serviceInfo = serviceInfo
		log.Printf("🎯 Service registered with Consul: %s (%s) at localhost:%d",
			serviceInfo.Name, serviceInfo.Meta["service_type"], serviceInfo.Port)

		return nil
	}

	// Traditional deployment - just log the configuration
	log.Printf("🎯 Service starting: %s (%s) at localhost:%d",
		sm.config.ServiceName, sm.config.ServiceType, sm.config.ServicePort)

	return nil
}

// GetServicePort returns the port this service should use
func (sm *ServiceManager) GetServicePort() int {
	if sm.serviceInfo != nil {
		return sm.serviceInfo.Port
	}
	return sm.config.ServicePort
}

// GetgRPCPort returns the gRPC port this service should use
func (sm *ServiceManager) GetgRPCPort() int {
	if sm.serviceInfo != nil && sm.serviceInfo.GRPCPort > 0 {
		return sm.serviceInfo.GRPCPort
	}
	// Return service port + 1 as gRPC port for non-Consul deployments
	return sm.config.ServicePort + 1
}

// CreateListener creates a network listener for the service
func (sm *ServiceManager) CreateListener() (net.Listener, error) {
	var addr string

	if sm.registry != nil {
		// Dynamic port allocation for Consul
		addr = ":0"
	} else {
		// Fixed port for traditional deployment
		addr = fmt.Sprintf(":%d", sm.config.ServicePort)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	sm.listener = listener
	return listener, nil
}

// DiscoverService discovers another service (used by clients)
func (sm *ServiceManager) DiscoverService(ctx context.Context, serviceName string, timeout time.Duration) (*consul.ServiceInfo, error) {
	if sm.registry != nil {
		// Use Consul service discovery
		return sm.registry.WaitForService(ctx, serviceName, timeout)
	}

	// Return hardcoded service info for traditional deployment
	return sm.getHardcodedServiceInfo(serviceName), nil
}

// getHardcodedServiceInfo returns service info for traditional deployments
func (sm *ServiceManager) getHardcodedServiceInfo(serviceName string) *consul.ServiceInfo {
	// Map service names to their default ports
	serviceMap := map[string]*consul.ServiceInfo{
		"openusp-data-service": {
			Name:     "openusp-data-service",
			Address:  "localhost",
			Port:     6400,  // HTTP port
			GRPCPort: 56400, // gRPC port
			Health:   "passing",
		},
		"openusp-api-gateway": {
			Name:    "openusp-api-gateway",
			Address: "localhost",
			Port:    6500,
			Health:  "passing",
		},
		"openusp-mtp-service": {
			Name:    "openusp-mtp-service",
			Address: "localhost",
			Port:    8081,
			Health:  "passing",
		},
		"openusp-cwmp-service": {
			Name:    "openusp-cwmp-service",
			Address: "localhost",
			Port:    7547,
			Health:  "passing",
		},
		"openusp-usp-service": {
			Name:     "openusp-usp-service",
			Address:  "localhost",
			GRPCPort: 56250,
			Health:   "passing",
		},
	}

	if service, exists := serviceMap[serviceName]; exists {
		return service
	}

	// Return default service info
	return &consul.ServiceInfo{
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

// GetServiceInfo returns the registered service information
func (sm *ServiceManager) GetServiceInfo() *consul.ServiceInfo {
	return sm.serviceInfo
}

// IsConsulEnabled returns whether Consul is enabled
func (sm *ServiceManager) IsConsulEnabled() bool {
	return sm.config.IsConsulEnabled()
}

// Shutdown gracefully shuts down the service
func (sm *ServiceManager) Shutdown() error {
	log.Printf("🛑 Shutting down service...")

	// Deregister from Consul
	if sm.registry != nil && sm.serviceInfo != nil {
		if err := sm.registry.DeregisterService(sm.serviceInfo.ID); err != nil {
			log.Printf("⚠️  Failed to deregister from Consul: %v", err)
		} else {
			log.Printf("✅ Deregistered from Consul successfully")
		}
	}

	// Close listener
	if sm.listener != nil {
		sm.listener.Close()
	}

	log.Printf("✅ Service stopped successfully")
	return nil
}
