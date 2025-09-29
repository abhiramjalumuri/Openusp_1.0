package consul

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ServiceManager handles service lifecycle with Consul
type ServiceManager struct {
	registry   *ServiceRegistry
	services   map[string]*ServiceInfo
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownCh chan struct{}
}

// NewServiceManager creates a new service manager with Consul
func NewServiceManager(registry *ServiceRegistry) *ServiceManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ServiceManager{
		registry:   registry,
		services:   make(map[string]*ServiceInfo),
		ctx:        ctx,
		cancel:     cancel,
		shutdownCh: make(chan struct{}),
	}
}

// RegisterService registers a service with Consul
func (sm *ServiceManager) RegisterService(serviceName, serviceType string) (*ServiceInfo, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if already registered locally
	if service, exists := sm.services[serviceName]; exists {
		log.Printf("⚠️ Service %s already registered on port %d", serviceName, service.Port)
		return service, nil
	}

	// Register with Consul
	service, err := sm.registry.RegisterService(sm.ctx, serviceName, serviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to register service %s: %w", serviceName, err)
	}

	// Cache locally
	sm.services[serviceName] = service

	return service, nil
}

// DiscoverService finds a service
func (sm *ServiceManager) DiscoverService(serviceName string) (*ServiceInfo, error) {
	return sm.registry.DiscoverService(serviceName)
}

// GetServiceURL returns the URL for a service
func (sm *ServiceManager) GetServiceURL(serviceName string) (string, error) {
	return sm.registry.GetServiceURL(serviceName)
}

// WaitForService waits for a service to become available
func (sm *ServiceManager) WaitForService(serviceName string, timeout time.Duration) (*ServiceInfo, error) {
	return sm.registry.WaitForService(sm.ctx, serviceName, timeout)
}

// StartService runs a service with automatic Consul registration
func (sm *ServiceManager) StartService(serviceName, serviceType string, startFn func(port int, grpcPort int) error) error {
	service, err := sm.RegisterService(serviceName, serviceType)
	if err != nil {
		return fmt.Errorf("failed to register service with Consul: %w", err)
	}

	// Handle graceful shutdown
	go sm.handleShutdown(serviceName)

	log.Printf("🚀 Starting %s service on port %d", serviceName, service.Port)

	// Start the service
	err = startFn(service.Port, service.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w", serviceName, err)
	}

	return nil
}

// GetAllServices returns all services
func (sm *ServiceManager) GetAllServices() (map[string]*ServiceInfo, error) {
	return sm.registry.GetAllServices()
}

// handleShutdown handles graceful service shutdown
func (sm *ServiceManager) handleShutdown(serviceName string) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Printf("📡 Received shutdown signal for service: %s", serviceName)
		sm.UnregisterService(serviceName)
	case <-sm.shutdownCh:
		return
	}
}

// UnregisterService removes a service from Consul
func (sm *ServiceManager) UnregisterService(serviceName string) {
	sm.mu.Lock()
	delete(sm.services, serviceName)
	sm.mu.Unlock()

	err := sm.registry.DeregisterService(serviceName)
	if err != nil {
		log.Printf("❌ Failed to deregister service %s: %v", serviceName, err)
	}
}

// Shutdown gracefully shuts down the service manager
func (sm *ServiceManager) Shutdown(ctx context.Context) error {
	log.Printf("🛑 Shutting down service manager...")

	close(sm.shutdownCh)
	sm.cancel()

	// Deregister all services
	sm.mu.Lock()
	for serviceName := range sm.services {
		sm.registry.DeregisterService(serviceName)
	}
	sm.services = make(map[string]*ServiceInfo)
	sm.mu.Unlock()

	return nil
}

// GetConsulStatus returns Consul health status
func (sm *ServiceManager) GetConsulStatus() error {
	return sm.registry.Health()
}
