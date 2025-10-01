package consul

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
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

// getHealthCheckHost determines the correct host for health checks based on environment
func getHealthCheckHost() string {
	// Check if running in Docker container (service itself in Docker)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "host.docker.internal"
	}

	// Check for Docker environment variables (service itself in Docker)
	if os.Getenv("DOCKER_CONTAINER") == "true" ||
		os.Getenv("IN_DOCKER") == "true" ||
		os.Getenv("HOSTNAME") != "" && os.Getenv("HOME") == "/" {
		return "host.docker.internal"
	}

	// Check if Consul is running in Docker (but our service is on host)
	// This is the common development scenario
	if isConsulInDocker() {
		return "host.docker.internal"
	}

	// Use 127.0.0.1 for fully local development
	return "127.0.0.1"
}

// isConsulInDocker checks if Consul is running in Docker container
func isConsulInDocker() bool {
	// Try to detect if we can reach a Docker container named consul
	// This is a simple heuristic for our development setup
	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		consulAddr = "localhost:8500"
	}

	// If we're connecting to localhost:8500 and Docker is available,
	// assume Consul is running in Docker (our common dev setup)
	if consulAddr == "localhost:8500" {
		// Quick check if Docker is available by looking for docker.sock
		if _, err := os.Stat("/var/run/docker.sock"); err == nil {
			return true
		}
		// On macOS, check for Docker Desktop's socket location
		if _, err := os.Stat("/Users/" + os.Getenv("USER") + "/.docker/run/docker.sock"); err == nil {
			return true
		}
	}

	return false
}

// DefaultConfig returns default Consul configuration
func DefaultConfig() *Config {
	return &Config{
		Address:       "localhost:8500",
		Datacenter:    "openusp-dev",
		CheckInterval: 10 * time.Second,
		CheckTimeout:  10 * time.Second, // Increased timeout for reliability
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
	healthHost := getHealthCheckHost()
	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: service.Address,
		Port:    service.Port,
		Tags:    service.Tags,
		Meta:    service.Meta,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", healthHost, service.Port),
			Interval:                       sr.config.CheckInterval.String(),
			Timeout:                        sr.config.CheckTimeout.String(),
			DeregisterCriticalServiceAfter: "90s", // Increased to handle startup delays
		},
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

	// First try Health API for healthy services
	healthServices, _, err := sr.client.Health().Service(serviceName, "", false, nil)
	if err == nil && len(healthServices) > 0 {
		// Use healthy service
		return sr.buildServiceInfoFromHealth(healthServices[0])
	}

	// Fallback to Catalog API for development (includes unhealthy services)
	catalogServices, _, err := sr.client.Catalog().Service(serviceName, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service %s: %w", serviceName, err)
	}

	if len(catalogServices) == 0 {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// Use the first service from catalog
	return sr.buildServiceInfoFromCatalog(catalogServices[0])
}

// buildServiceInfoFromHealth builds ServiceInfo from Health API response
func (sr *ServiceRegistry) buildServiceInfoFromHealth(healthEntry *api.ServiceEntry) (*ServiceInfo, error) {
	consulService := healthEntry.Service

	// Determine actual health status
	healthStatus := "critical"
	for _, check := range healthEntry.Checks {
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
	sr.services[consulService.Service] = service
	return service, nil
}

// buildServiceInfoFromCatalog builds ServiceInfo from Catalog API response
func (sr *ServiceRegistry) buildServiceInfoFromCatalog(catalogService *api.CatalogService) (*ServiceInfo, error) {
	grpcPort := 0
	if grpcPortStr, exists := catalogService.ServiceMeta["grpc_port"]; exists {
		if port, err := strconv.Atoi(grpcPortStr); err == nil {
			grpcPort = port
		}
	}

	service := &ServiceInfo{
		ID:       catalogService.ServiceID,
		Name:     catalogService.ServiceName,
		Address:  catalogService.ServiceAddress,
		Port:     catalogService.ServicePort,
		GRPCPort: grpcPort,
		Tags:     catalogService.ServiceTags,
		Meta:     catalogService.ServiceMeta,
		Protocol: catalogService.ServiceMeta["protocol"],
		Health:   "unknown", // Catalog doesn't provide health status
	}

	// Cache the discovered service
	sr.services[catalogService.ServiceName] = service
	return service, nil
}

// RegisterServiceWithPorts registers a service with Consul using specified ports
func (sr *ServiceRegistry) RegisterServiceWithPorts(ctx context.Context, serviceInfo *ServiceInfo) error {
	serviceID := fmt.Sprintf("%s-%d", serviceInfo.Name, time.Now().Unix())
	serviceInfo.ID = serviceID

	// Ensure metadata includes gRPC port if available
	if serviceInfo.GRPCPort > 0 {
		serviceInfo.Meta["grpc_port"] = strconv.Itoa(serviceInfo.GRPCPort)
	}

	// Register with Consul
	healthHost := getHealthCheckHost()
	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceInfo.Name,
		Address: serviceInfo.Address,
		Port:    serviceInfo.Port,
		Tags:    []string{"openusp", "v1.0.0", serviceInfo.Meta["service_type"]},
		Meta:    serviceInfo.Meta,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", healthHost, serviceInfo.Port),
			Interval:                       sr.config.CheckInterval.String(),
			Timeout:                        sr.config.CheckTimeout.String(),
			DeregisterCriticalServiceAfter: "90s",
		},
	}

	err := sr.client.Agent().ServiceRegister(registration)
	if err != nil {
		return fmt.Errorf("failed to register service %s with Consul: %w", serviceInfo.Name, err)
	}

	// Cache locally
	sr.services[serviceInfo.Name] = serviceInfo

	log.Printf("🎯 Service registered with Consul: %s (%s) at %s:%d",
		serviceInfo.Name, serviceInfo.Meta["service_type"], serviceInfo.Address, serviceInfo.Port)

	if serviceInfo.GRPCPort > 0 {
		log.Printf("   └── gRPC port: %d", serviceInfo.GRPCPort)
	}

	return nil
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

// UpdateServiceMetadata updates the metadata of an existing service registration
func (sr *ServiceRegistry) UpdateServiceMetadata(serviceName string, updatedMetadata map[string]string) error {
	// Get existing service info
	service, exists := sr.services[serviceName]
	if !exists {
		return fmt.Errorf("service %s not found in local cache", serviceName)
	}

	// Update the metadata
	for key, value := range updatedMetadata {
		service.Meta[key] = value
	}

	// Update local cache
	sr.services[serviceName] = service

	// Re-register the service with updated metadata
	registration := &api.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Name,
		Address: service.Address,
		Port:    service.Port,
		Tags:    service.Tags,
		Meta:    service.Meta,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://127.0.0.1:%d/health", service.Port),
			Interval:                       sr.config.CheckInterval.String(),
			Timeout:                        sr.config.CheckTimeout.String(),
			DeregisterCriticalServiceAfter: "90s",
		},
	}

	// Re-register with Consul (this updates the existing registration)
	err := sr.client.Agent().ServiceRegister(registration)
	if err != nil {
		return fmt.Errorf("failed to update service %s metadata in Consul: %w", serviceName, err)
	}

	log.Printf("🔄 Updated service metadata in Consul: %s", serviceName)
	return nil
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
