package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"openusp/pkg/consul"
	"openusp/pkg/proto/dataservice"
	"openusp/pkg/proto/mtpservice"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
)

// ConnectionState represents the current state of a service connection
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateFailed
)

// ServiceConnection manages a single gRPC service connection with circuit breaker pattern
type ServiceConnection struct {
	serviceName     string
	conn            *grpc.ClientConn
	state           ConnectionState
	lastError       error
	retryCount      int
	maxRetries      int
	retryDelay      time.Duration
	circuitOpen     bool
	circuitOpenTime time.Time
	circuitTimeout  time.Duration
	mu              sync.RWMutex
}

// ConnectionManager manages all inter-service gRPC connections with industry-standard patterns
type ConnectionManager struct {
	registry    *consul.ServiceRegistry
	connections map[string]*ServiceConnection
	mu          sync.RWMutex

	// Connection pool settings
	maxRetries     int
	baseRetryDelay time.Duration
	circuitTimeout time.Duration
	healthInterval time.Duration

	// Service discovery settings
	discoveryTimeout time.Duration

	// Background health checking
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewConnectionManager creates a new connection manager with industry-standard settings
func NewConnectionManager(registry *consul.ServiceRegistry) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	cm := &ConnectionManager{
		registry:         registry,
		connections:      make(map[string]*ServiceConnection),
		maxRetries:       5,
		baseRetryDelay:   time.Second,
		circuitTimeout:   30 * time.Second,
		healthInterval:   10 * time.Second,
		discoveryTimeout: 5 * time.Second,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Start background health monitoring
	cm.wg.Add(1)
	go cm.healthMonitor()

	return cm
}

// GetDataServiceClient returns a data service client with automatic connection management
func (cm *ConnectionManager) GetDataServiceClient() (dataservice.DataServiceClient, error) {
	conn, err := cm.getConnection("openusp-data-service")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to data service: %w", err)
	}

	return dataservice.NewDataServiceClient(conn), nil
}

// GetMTPServiceClient returns an MTP service client with automatic connection management
func (cm *ConnectionManager) GetMTPServiceClient() (mtpservice.MTPServiceClient, error) {
	conn, err := cm.getConnection("openusp-mtp-service")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MTP service: %w", err)
	}

	return mtpservice.NewMTPServiceClient(conn), nil
}

// getConnection retrieves or creates a connection to the specified service
func (cm *ConnectionManager) getConnection(serviceName string) (*grpc.ClientConn, error) {
	cm.mu.RLock()
	serviceConn, exists := cm.connections[serviceName]
	cm.mu.RUnlock()

	if !exists {
		cm.mu.Lock()
		// Double-check pattern
		if serviceConn, exists = cm.connections[serviceName]; !exists {
			serviceConn = &ServiceConnection{
				serviceName:    serviceName,
				state:          StateDisconnected,
				maxRetries:     cm.maxRetries,
				retryDelay:     cm.baseRetryDelay,
				circuitTimeout: cm.circuitTimeout,
			}
			cm.connections[serviceName] = serviceConn
		}
		cm.mu.Unlock()
	}

	return cm.ensureConnection(serviceConn)
}

// ensureConnection ensures the service connection is available and healthy
func (cm *ConnectionManager) ensureConnection(serviceConn *ServiceConnection) (*grpc.ClientConn, error) {
	serviceConn.mu.Lock()
	defer serviceConn.mu.Unlock()

	// Check circuit breaker state
	if serviceConn.circuitOpen {
		if time.Since(serviceConn.circuitOpenTime) < serviceConn.circuitTimeout {
			return nil, fmt.Errorf("circuit breaker open for service %s", serviceConn.serviceName)
		}
		// Reset circuit breaker after timeout
		serviceConn.circuitOpen = false
		serviceConn.retryCount = 0
		log.Printf("🔄 Circuit breaker reset for service: %s", serviceConn.serviceName)
	}

	// Check existing connection health
	if serviceConn.conn != nil {
		switch serviceConn.conn.GetState() {
		case connectivity.Ready:
			return serviceConn.conn, nil
		case connectivity.Connecting:
			// Wait for connection to be ready (with timeout)
			ctx, cancel := context.WithTimeout(cm.ctx, 5*time.Second)
			defer cancel()
			if serviceConn.conn.WaitForStateChange(ctx, connectivity.Connecting) {
				if serviceConn.conn.GetState() == connectivity.Ready {
					return serviceConn.conn, nil
				}
			}
		case connectivity.TransientFailure, connectivity.Shutdown:
			// Close and recreate connection
			serviceConn.conn.Close()
			serviceConn.conn = nil
		}
	}

	// Create new connection with retry logic
	return cm.createConnectionWithRetry(serviceConn)
}

// createConnectionWithRetry creates a new connection with exponential backoff retry
func (cm *ConnectionManager) createConnectionWithRetry(serviceConn *ServiceConnection) (*grpc.ClientConn, error) {
	var lastErr error

	for attempt := 0; attempt <= serviceConn.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := time.Duration(attempt) * serviceConn.retryDelay
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}

			log.Printf("🔄 Retrying connection to %s in %v (attempt %d/%d)",
				serviceConn.serviceName, delay, attempt, serviceConn.maxRetries)

			select {
			case <-time.After(delay):
			case <-cm.ctx.Done():
				return nil, fmt.Errorf("connection manager shutting down")
			}
		}

		// Discover service via Consul (with timeout)
		ctx, cancel := context.WithTimeout(cm.ctx, cm.discoveryTimeout)
		serviceInfo, err := cm.discoverService(ctx, serviceConn.serviceName)
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("service discovery failed: %w", err)
			log.Printf("⚠️ Service discovery failed for %s: %v", serviceConn.serviceName, err)
			continue
		}

		// Create gRPC connection
		addr := fmt.Sprintf("localhost:%d", serviceInfo.Port)
		if grpcPort, exists := serviceInfo.Meta["grpc_port"]; exists {
			addr = fmt.Sprintf("localhost:%s", grpcPort)
		}

		conn, err := grpc.DialContext(cm.ctx, addr,
			grpc.WithInsecure(),
			grpc.WithTimeout(10*time.Second),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                30 * time.Second,
				Timeout:             5 * time.Second,
				PermitWithoutStream: true,
			}),
		)

		if err != nil {
			lastErr = fmt.Errorf("gRPC dial failed: %w", err)
			log.Printf("⚠️ gRPC connection failed to %s at %s: %v", serviceConn.serviceName, addr, err)
			continue
		}

		// Test connection with health check
		ctx, cancel = context.WithTimeout(cm.ctx, 3*time.Second)
		if err := cm.testConnection(ctx, conn, serviceConn.serviceName); err != nil {
			cancel()
			conn.Close()
			lastErr = fmt.Errorf("health check failed: %w", err)
			log.Printf("⚠️ Health check failed for %s: %v", serviceConn.serviceName, err)
			continue
		}
		cancel()

		// Connection successful
		serviceConn.conn = conn
		serviceConn.state = StateConnected
		serviceConn.lastError = nil
		serviceConn.retryCount = 0
		serviceConn.circuitOpen = false

		log.Printf("✅ Successfully connected to %s at %s", serviceConn.serviceName, addr)
		return conn, nil
	}

	// All retries failed - open circuit breaker
	serviceConn.state = StateFailed
	serviceConn.lastError = lastErr
	serviceConn.circuitOpen = true
	serviceConn.circuitOpenTime = time.Now()
	serviceConn.retryCount++

	log.Printf("❌ Circuit breaker opened for %s after %d failed attempts",
		serviceConn.serviceName, serviceConn.maxRetries+1)

	return nil, fmt.Errorf("all connection attempts failed for %s: %w", serviceConn.serviceName, lastErr)
}

// discoverService discovers a service via Consul with proper error handling
func (cm *ConnectionManager) discoverService(ctx context.Context, serviceName string) (*consul.ServiceInfo, error) {
	if cm.registry == nil {
		return nil, fmt.Errorf("consul registry not available")
	}

	serviceInfo, err := cm.registry.DiscoverService(serviceName)
	if err != nil {
		return nil, err
	}

	if serviceInfo == nil {
		return nil, fmt.Errorf("service %s not found in consul", serviceName)
	}

	return serviceInfo, nil
}

// testConnection performs a basic health check on the gRPC connection
func (cm *ConnectionManager) testConnection(ctx context.Context, conn *grpc.ClientConn, serviceName string) error {
	// Try to make a simple call to verify the connection works
	switch serviceName {
	case "openusp-data-service":
		client := dataservice.NewDataServiceClient(conn)
		_, err := client.HealthCheck(ctx, &dataservice.HealthCheckRequest{})
		return err
	case "openusp-mtp-service":
		client := mtpservice.NewMTPServiceClient(conn)
		_, err := client.GetServiceHealth(ctx, &mtpservice.HealthRequest{})
		return err
	default:
		// Generic connectivity test - try to get connection state
		if conn.GetState() == connectivity.Ready {
			return nil
		}
		return fmt.Errorf("connection not ready")
	}
}

// healthMonitor runs background health checks on all connections
func (cm *ConnectionManager) healthMonitor() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.performHealthChecks()
		case <-cm.ctx.Done():
			return
		}
	}
}

// performHealthChecks checks health of all active connections
func (cm *ConnectionManager) performHealthChecks() {
	cm.mu.RLock()
	connections := make([]*ServiceConnection, 0, len(cm.connections))
	for _, conn := range cm.connections {
		connections = append(connections, conn)
	}
	cm.mu.RUnlock()

	for _, serviceConn := range connections {
		cm.checkConnectionHealth(serviceConn)
	}
}

// checkConnectionHealth performs health check on a single connection
func (cm *ConnectionManager) checkConnectionHealth(serviceConn *ServiceConnection) {
	serviceConn.mu.RLock()
	conn := serviceConn.conn
	serviceName := serviceConn.serviceName
	serviceConn.mu.RUnlock()

	if conn == nil {
		return
	}

	ctx, cancel := context.WithTimeout(cm.ctx, 3*time.Second)
	defer cancel()

	if err := cm.testConnection(ctx, conn, serviceName); err != nil {
		serviceConn.mu.Lock()
		if serviceConn.conn == conn { // Make sure connection hasn't changed
			serviceConn.state = StateFailed
			serviceConn.lastError = err
			log.Printf("⚠️ Health check failed for %s: %v", serviceName, err)
		}
		serviceConn.mu.Unlock()
	} else {
		serviceConn.mu.Lock()
		if serviceConn.conn == conn {
			serviceConn.state = StateConnected
			serviceConn.lastError = nil
		}
		serviceConn.mu.Unlock()
	}
}

// GetConnectionStatus returns the status of all managed connections
func (cm *ConnectionManager) GetConnectionStatus() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	status := make(map[string]interface{})

	for name, conn := range cm.connections {
		conn.mu.RLock()
		connStatus := map[string]interface{}{
			"state":        conn.state,
			"retry_count":  conn.retryCount,
			"circuit_open": conn.circuitOpen,
			"last_error":   nil,
		}
		if conn.lastError != nil {
			connStatus["last_error"] = conn.lastError.Error()
		}
		if conn.conn != nil {
			connStatus["grpc_state"] = conn.conn.GetState().String()
		}
		conn.mu.RUnlock()

		status[name] = connStatus
	}

	return status
}

// Close gracefully shuts down the connection manager
func (cm *ConnectionManager) Close() error {
	cm.cancel()
	cm.wg.Wait()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	var lastErr error
	for _, conn := range cm.connections {
		conn.mu.Lock()
		if conn.conn != nil {
			if err := conn.conn.Close(); err != nil {
				lastErr = err
				log.Printf("⚠️ Error closing connection to %s: %v", conn.serviceName, err)
			}
		}
		conn.mu.Unlock()
	}

	return lastErr
}

// IsServiceAvailable checks if a service is currently available (for dependency checking)
func (cm *ConnectionManager) IsServiceAvailable(serviceName string) bool {
	cm.mu.RLock()
	serviceConn, exists := cm.connections[serviceName]
	cm.mu.RUnlock()

	if !exists {
		return false
	}

	serviceConn.mu.RLock()
	defer serviceConn.mu.RUnlock()

	return serviceConn.state == StateConnected && !serviceConn.circuitOpen
}
