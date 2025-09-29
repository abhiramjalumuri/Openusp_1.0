package consul

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
)

// ServiceInfo represents service information in Consul
type ServiceInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	GRPCPort int               `json:"grpc_port,omitempty"`
	Tags     []string          `json:"tags"`
	Meta     map[string]string `json:"meta"`
	Protocol string            `json:"protocol"` // http, grpc, websocket
	Health   string            `json:"health"`   // passing, warning, critical
}

// ServiceRegistry provides Consul-based service discovery
type ServiceRegistry struct {
	client   *api.Client
	config   *Config
	services map[string]*ServiceInfo // Local cache
}

// Config contains Consul configuration
type Config struct {
	Address       string        `json:"address"`
	Datacenter    string        `json:"datacenter"`
	Token         string        `json:"token,omitempty"`
	CheckInterval time.Duration `json:"check_interval"`
	CheckTimeout  time.Duration `json:"check_timeout"`
}

// DefaultConfig returns default Consul configuration
func DefaultConfig() *Config {
	return &Config{
		Address:       "localhost:8500",
		Datacenter:    "openusp-dev",
		CheckInterval: 10 * time.Second,
		CheckTimeout:  5 * time.Second,
	}
}

// NewServiceRegistry creates a new Consul-based service registry
func NewServiceRegistry(config *Config) (*ServiceRegistry, error) {
	if config == nil {
		config = DefaultConfig()
	}

	consulConfig := api.DefaultConfig()
	consulConfig.Address = config.Address
	consulConfig.Datacenter = config.Datacenter

	if config.Token != "" {
		consulConfig.Token = config.Token
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	// Test connection
	if _, err := client.Status().Leader(); err != nil {
		return nil, fmt.Errorf("failed to connect to Consul: %w", err)
	}

	log.Printf("🏛️ Connected to Consul at %s (datacenter: %s)", config.Address, config.Datacenter)

	return &ServiceRegistry{
		client:   client,
		config:   config,
		services: make(map[string]*ServiceInfo),
	}, nil
}

// RegisterService registers a service with Consul and dynamic port allocation
func (sr *ServiceRegistry) RegisterService(ctx context.Context, serviceName, serviceType string) (*ServiceInfo, error) {
	// Get available ports
	httpPort, err := GetAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP port: %w", err)
	}

	var grpcPort int
	// Allocate gRPC port for backend services
	if serviceType != "api-gateway" && serviceType != "client" {
		grpcPort, err = GetAvailablePort()
		if err != nil {
			return nil, fmt.Errorf("failed to get gRPC port: %w", err)
		}
	}

	// Determine protocol based on service type
	protocol := "http"
	switch serviceType {
	case "data-service", "usp-service":
		protocol = "grpc"
	case "mtp-service":
		protocol = "websocket"
	}

	serviceID := fmt.Sprintf("%s-%d", serviceName, time.Now().Unix())

	service := &ServiceInfo{
		ID:       serviceID,
		Name:     serviceName,
		Address:  "localhost",
		Port:     httpPort,
		GRPCPort: grpcPort,
		Protocol: protocol,
		Tags:     []string{"openusp", "v1.0.0", serviceType},
		Meta: map[string]string{
			"service_type": serviceType,
			"protocol":     protocol,
			"version":      "1.0.0",
			"environment":  "development",
		},
		Health: "starting",
	}

	if grpcPort > 0 {
		service.Meta["grpc_port"] = strconv.Itoa(grpcPort)
	}

	// Register with Consul
	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: service.Address,
		Port:    service.Port,
		Tags:    service.Tags,
		Meta:    service.Meta,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", service.Address, service.Port),
			Interval:                       sr.config.CheckInterval.String(),
			Timeout:                        sr.config.CheckTimeout.String(),
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	// Use gRPC health check for gRPC services
	if protocol == "grpc" {
		port := grpcPort
		if port == 0 {
			port = httpPort
		}
		registration.Check = &api.AgentServiceCheck{
			GRPC:                           fmt.Sprintf("%s:%d", service.Address, port),
			Interval:                       sr.config.CheckInterval.String(),
			Timeout:                        sr.config.CheckTimeout.String(),
			DeregisterCriticalServiceAfter: "30s",
		}
	}

	err = sr.client.Agent().ServiceRegister(registration)
	if err != nil {
		return nil, fmt.Errorf("failed to register service %s with Consul: %w", serviceName, err)
	}

	// Cache locally
	sr.services[serviceName] = service

	log.Printf("🎯 Service registered with Consul: %s (%s) at %s:%d",
		service.Name, serviceType, service.Address, service.Port)

	if grpcPort > 0 {
		log.Printf("   └── gRPC port: %d", grpcPort)
	}

	return service, nil
}

// DiscoverService finds a service by name
func (sr *ServiceRegistry) DiscoverService(serviceName string) (*ServiceInfo, error) {
	// Check local cache first
	if service, exists := sr.services[serviceName]; exists {
		return service, nil
	}

	// Query Consul for services (including non-healthy ones for development)
	services, _, err := sr.client.Health().Service(serviceName, "", false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service %s: %w", serviceName, err)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// Get the first service (check its health status)
	consulService := services[0].Service

	// Determine actual health status
	healthStatus := "critical"
	for _, check := range services[0].Checks {
		if check.Status == "passing" {
			healthStatus = "passing"
			break
		} else if check.Status == "warning" {
			healthStatus = "warning"
		}
	}

	grpcPort := 0
	if grpcPortStr, exists := consulService.Meta["grpc_port"]; exists {
		if port, err := strconv.Atoi(grpcPortStr); err == nil {
			grpcPort = port
		}
	}

	service := &ServiceInfo{
		ID:       consulService.ID,
		Name:     consulService.Service,
		Address:  consulService.Address,
		Port:     consulService.Port,
		GRPCPort: grpcPort,
		Tags:     consulService.Tags,
		Meta:     consulService.Meta,
		Protocol: consulService.Meta["protocol"],
		Health:   healthStatus,
	}

	// Cache the discovered service
	sr.services[serviceName] = service

	return service, nil
}

// GetServiceURL returns the URL for a service
func (sr *ServiceRegistry) GetServiceURL(serviceName string) (string, error) {
	service, err := sr.DiscoverService(serviceName)
	if err != nil {
		return "", err
	}

	switch service.Protocol {
	case "grpc":
		if service.GRPCPort > 0 {
			return fmt.Sprintf("localhost:%d", service.GRPCPort), nil
		}
		return fmt.Sprintf("localhost:%d", service.Port), nil
	case "websocket":
		return fmt.Sprintf("ws://%s:%d", service.Address, service.Port), nil
	default:
		return fmt.Sprintf("http://%s:%d", service.Address, service.Port), nil
	}
}

// WaitForService waits for a service to become available
func (sr *ServiceRegistry) WaitForService(ctx context.Context, serviceName string, timeout time.Duration) (*ServiceInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Printf("🔍 Waiting for service: %s (timeout: %v)", serviceName, timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for service %s", serviceName)
		case <-ticker.C:
			service, err := sr.DiscoverService(serviceName)
			if err == nil {
				log.Printf("✅ Service %s is now available at %s:%d", serviceName, service.Address, service.Port)
				return service, nil
			}
		}
	}
}

// GetAllServices returns all registered OpenUSP services
func (sr *ServiceRegistry) GetAllServices() (map[string]*ServiceInfo, error) {
	// Get all services from Consul catalog
	catalogServices, _, err := sr.client.Catalog().Services(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog services: %w", err)
	}

	result := make(map[string]*ServiceInfo)

	// Iterate through each service and get its health info
	for serviceName, tags := range catalogServices {
		// Skip non-OpenUSP services
		hasOpenUSPTag := false
		for _, tag := range tags {
			if tag == "openusp" {
				hasOpenUSPTag = true
				break
			}
		}
		if !hasOpenUSPTag {
			continue
		}

		// Get health information for this service
		healthEntries, _, err := sr.client.Health().Service(serviceName, "", false, nil)
		if err != nil {
			continue // Skip this service if we can't get health info
		}

		if len(healthEntries) == 0 {
			continue // Skip if no instances
		}

		// Use the first instance
		entry := healthEntries[0]
		service := entry.Service

		// Determine health status
		health := "critical"
		for _, check := range entry.Checks {
			if check.Status == "passing" {
				health = "passing"
				break
			} else if check.Status == "warning" && health != "passing" {
				health = "warning"
			}
		}

		grpcPort := 0
		if grpcPortStr, exists := service.Meta["grpc_port"]; exists {
			if port, err := strconv.Atoi(grpcPortStr); err == nil {
				grpcPort = port
			}
		}

		result[service.Service] = &ServiceInfo{
			ID:       service.ID,
			Name:     service.Service,
			Address:  service.Address,
			Port:     service.Port,
			GRPCPort: grpcPort,
			Tags:     service.Tags,
			Meta:     service.Meta,
			Protocol: service.Meta["protocol"],
			Health:   health,
		}
	}

	return result, nil
}

// DeregisterService removes a service from Consul
func (sr *ServiceRegistry) DeregisterService(serviceName string) error {
	// Find service ID from local cache
	service, exists := sr.services[serviceName]
	if !exists {
		// Try to find by querying Consul
		discoveredService, err := sr.DiscoverService(serviceName)
		if err != nil {
			return fmt.Errorf("service %s not found for deregistration", serviceName)
		}
		service = discoveredService
	}

	err := sr.client.Agent().ServiceDeregister(service.ID)
	if err != nil {
		return fmt.Errorf("failed to deregister service %s: %w", serviceName, err)
	}

	// Remove from local cache
	delete(sr.services, serviceName)

	log.Printf("📤 Service deregistered from Consul: %s", serviceName)
	return nil
}

// Health checks if Consul is healthy
func (sr *ServiceRegistry) Health() error {
	_, err := sr.client.Status().Leader()
	return err
}

// GetAvailablePort finds an available port
func GetAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
