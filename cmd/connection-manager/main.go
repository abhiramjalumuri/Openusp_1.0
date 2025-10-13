package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"openusp/pkg/config"
	"openusp/pkg/metrics"
	"openusp/pkg/proto/connectionservice"
	"openusp/pkg/version"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// ConnectionManagerService manages gRPC connections between OpenUSP services
type ConnectionManagerService struct {
	connectionservice.UnimplementedConnectionServiceServer

	// Service registry and discovery

	// Connection pools per service type
	connections map[string]*ServiceConnectionPool
	mu          sync.RWMutex

	// Configuration
	config *ConnectionManagerConfig

	// Actual ports for status reporting
	actualGRPCPort int
	httpPort       int

	// Metrics and monitoring
	metrics   *metrics.OpenUSPMetrics
	startTime time.Time

	// Background operations
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ServiceConnectionPool manages connections to a specific service type
type ServiceConnectionPool struct {
	serviceName    string
	connections    []*ManagedConnection
	activeIndex    int
	maxConnections int
	minConnections int
	mu             sync.RWMutex

	// Circuit breaker state
	circuitOpen     bool
	circuitOpenTime time.Time
	failureCount    int
	maxFailures     int
	resetTimeout    time.Duration

	// Health monitoring
	lastHealthCheck time.Time
	isHealthy       bool
}

// ManagedConnection represents a single managed gRPC connection
type ManagedConnection struct {
	conn       *grpc.ClientConn
	target     string
	state      connectivity.State
	lastUsed   time.Time
	errorCount int
	createdAt  time.Time

	// Connection metadata
	metadata map[string]string

	mu sync.RWMutex
}

// ConnectionManagerConfig holds configuration for the connection manager
type ConnectionManagerConfig struct {
	// Pool settings
	MaxConnectionsPerService int
	MinConnectionsPerService int
	ConnectionTimeout        time.Duration
	IdleTimeout              time.Duration

	// Circuit breaker settings
	MaxFailures    int
	CircuitTimeout time.Duration

	// Health check settings
	HealthCheckInterval time.Duration
	HealthCheckTimeout  time.Duration

	// Service discovery settings
	DiscoveryInterval time.Duration
	DiscoveryTimeout  time.Duration

	// gRPC settings
	KeepaliveTime    time.Duration
	KeepaliveTimeout time.Duration
}

// DefaultConnectionManagerConfig returns default configuration
func DefaultConnectionManagerConfig() *ConnectionManagerConfig {
	return &ConnectionManagerConfig{
		MaxConnectionsPerService: 5,
		MinConnectionsPerService: 1,
		ConnectionTimeout:        10 * time.Second,
		IdleTimeout:              300 * time.Second,
		MaxFailures:              3,
		CircuitTimeout:           30 * time.Second,
		HealthCheckInterval:      15 * time.Second,
		HealthCheckTimeout:       5 * time.Second,
		DiscoveryInterval:        30 * time.Second,
		DiscoveryTimeout:         5 * time.Second,
		KeepaliveTime:            30 * time.Second,
		KeepaliveTimeout:         5 * time.Second,
	}
}

// NewConnectionManagerService creates a new connection manager service
func NewConnectionManagerService(config *ConnectionManagerConfig) *ConnectionManagerService {
	ctx, cancel := context.WithCancel(context.Background())

	if config == nil {
		config = DefaultConnectionManagerConfig()
	}

	cms := &ConnectionManagerService{
		connections: make(map[string]*ServiceConnectionPool),
		config:      config,
		metrics:     metrics.NewOpenUSPMetrics("connection-manager"),
		startTime:   time.Now(),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start background services
	cms.startBackgroundServices()

	return cms
}

// getServiceTarget returns the target address for a service using environment configuration
func (cms *ConnectionManagerService) getServiceTarget(serviceName string) string {
	// Static mapping; in future read from unified YAML if needed.
	switch serviceName {
	case "openusp-data-service":
		return "localhost:50100"
	case "openusp-usp-service":
		return "localhost:50200"
	case "openusp-mtp-service":
		return "localhost:50300"
	case "openusp-connection-manager":
		return "localhost:50400"
	default:
		return ""
	}
}

// SetPorts updates the connection manager with actual port numbers for status reporting
func (cms *ConnectionManagerService) SetPorts(grpcPort, httpPort int) {
	cms.actualGRPCPort = grpcPort
	cms.httpPort = httpPort
}

// GetConnection returns a healthy connection to the specified service
func (cms *ConnectionManagerService) GetConnection(ctx context.Context, req *connectionservice.GetConnectionRequest) (*connectionservice.GetConnectionResponse, error) {
	pool, err := cms.getOrCreatePool(req.ServiceName)
	if err != nil {
		log.Printf("🚨 GetConnection pool creation failed for %s: %v", req.ServiceName, err)
		return &connectionservice.GetConnectionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get connection pool: %v", err),
		}, nil
	}

	// Debug logging before attempting to get connection
	pool.mu.RLock()
	totalConns := len(pool.connections)
	healthyConns := cms.countHealthyConnections(pool)
	circuitOpen := pool.circuitOpen
	failureCount := pool.failureCount
	isHealthy := pool.isHealthy
	pool.mu.RUnlock()

	log.Printf("🧪 GetConnection request: service=%s total=%d healthy=%d circuit_open=%v failures=%d pool_healthy=%v",
		req.ServiceName, totalConns, healthyConns, circuitOpen, failureCount, isHealthy)

	conn, err := pool.getConnection(ctx)
	if err != nil {
		log.Printf("🚨 GetConnection failed for %s: %v (pool: %d total, %d healthy, circuit: %v)",
			req.ServiceName, err, totalConns, healthyConns, circuitOpen)
		return &connectionservice.GetConnectionResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get connection: %v", err),
		}, nil
	}

	log.Printf("✅ GetConnection success for %s: target=%s state=%s", req.ServiceName, conn.target, conn.conn.GetState().String())

	return &connectionservice.GetConnectionResponse{
		Success:    true,
		Target:     conn.target,
		State:      conn.conn.GetState().String(),
		CreatedAt:  conn.createdAt.Unix(),
		LastUsed:   conn.lastUsed.Unix(),
		ErrorCount: int32(conn.errorCount),
		Metadata:   conn.metadata,
	}, nil
}

// GetConnectionStatus returns status of all managed connections
func (cms *ConnectionManagerService) GetConnectionStatus(ctx context.Context, req *connectionservice.GetConnectionStatusRequest) (*connectionservice.GetConnectionStatusResponse, error) {
	cms.mu.RLock()
	defer cms.mu.RUnlock()

	status := &connectionservice.GetConnectionStatusResponse{
		Services: make(map[string]*connectionservice.ServiceStatus),
	}

	for serviceName, pool := range cms.connections {
		pool.mu.RLock()
		serviceStatus := &connectionservice.ServiceStatus{
			ServiceName:        serviceName,
			TotalConnections:   int32(len(pool.connections)),
			HealthyConnections: int32(cms.countHealthyConnections(pool)),
			CircuitOpen:        pool.circuitOpen,
			IsHealthy:          pool.isHealthy,
			LastHealthCheck:    pool.lastHealthCheck.Unix(),
			FailureCount:       int32(pool.failureCount),
		}

		// Add connection details
		for i, conn := range pool.connections {
			conn.mu.RLock()
			connDetail := &connectionservice.ConnectionDetail{
				Index:      int32(i),
				Target:     conn.target,
				State:      conn.conn.GetState().String(),
				LastUsed:   conn.lastUsed.Unix(),
				ErrorCount: int32(conn.errorCount),
				CreatedAt:  conn.createdAt.Unix(),
			}
			serviceStatus.Connections = append(serviceStatus.Connections, connDetail)
			conn.mu.RUnlock()
		}

		pool.mu.RUnlock()
		status.Services[serviceName] = serviceStatus
	}

	return status, nil
}

// RegisterServiceDependency registers a service dependency for proactive connection management
func (cms *ConnectionManagerService) RegisterServiceDependency(ctx context.Context, req *connectionservice.RegisterDependencyRequest) (*connectionservice.RegisterDependencyResponse, error) {
	log.Printf("🔗 Registering service dependency: %s -> %s", req.ServiceName, req.DependsOnService)

	// Pre-create connection pool for the dependency
	_, err := cms.getOrCreatePool(req.DependsOnService)
	if err != nil {
		return &connectionservice.RegisterDependencyResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create connection pool: %v", err),
		}, nil
	}

	return &connectionservice.RegisterDependencyResponse{
		Success: true,
	}, nil
}

// getOrCreatePool gets or creates a connection pool for the specified service
func (cms *ConnectionManagerService) getOrCreatePool(serviceName string) (*ServiceConnectionPool, error) {
	cms.mu.RLock()
	pool, exists := cms.connections[serviceName]
	cms.mu.RUnlock()

	if exists {
		return pool, nil
	}

	cms.mu.Lock()
	defer cms.mu.Unlock()

	// Double-check pattern
	if pool, exists = cms.connections[serviceName]; exists {
		return pool, nil
	}

	// Create new connection pool
	pool = &ServiceConnectionPool{
		serviceName:     serviceName,
		connections:     make([]*ManagedConnection, 0),
		maxConnections:  cms.config.MaxConnectionsPerService,
		minConnections:  cms.config.MinConnectionsPerService,
		maxFailures:     cms.config.MaxFailures,
		resetTimeout:    cms.config.CircuitTimeout,
		lastHealthCheck: time.Now(),
		isHealthy:       false,
	}

	cms.connections[serviceName] = pool

	// Initialize minimum connections synchronously to avoid race condition
	log.Printf("🔧 Creating connection pool for service: %s, initializing %d connections...", serviceName, pool.minConnections)
	for i := 0; i < pool.minConnections; i++ {
		if err := cms.addConnectionToPool(pool); err != nil {
			log.Printf("⚠️ Failed to add initial connection %d/%d to pool %s: %v", i+1, pool.minConnections, pool.serviceName, err)
		} else {
			log.Printf("✅ Added initial connection %d/%d to pool %s", i+1, pool.minConnections, pool.serviceName)
		}
	}

	log.Printf("🔧 Created connection pool for service: %s with %d/%d initial connections", serviceName, len(pool.connections), pool.minConnections)
	return pool, nil
}

// getConnection returns a healthy connection from the pool
func (pool *ServiceConnectionPool) getConnection(ctx context.Context) (*ManagedConnection, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	log.Printf("🔍 pool.getConnection for %s: checking %d connections, circuit_open=%v",
		pool.serviceName, len(pool.connections), pool.circuitOpen)

	// Check circuit breaker
	if pool.circuitOpen {
		if time.Since(pool.circuitOpenTime) < pool.resetTimeout {
			log.Printf("🔴 Circuit breaker still open for %s (opened: %v ago, timeout: %v)",
				pool.serviceName, time.Since(pool.circuitOpenTime), pool.resetTimeout)
			return nil, fmt.Errorf("circuit breaker open for service %s", pool.serviceName)
		}
		// Reset circuit breaker
		pool.circuitOpen = false
		pool.failureCount = 0
		log.Printf("🔄 Circuit breaker reset for service: %s", pool.serviceName)
	}

	// Find healthy connection using round-robin
	readyConnections := 0
	for i := 0; i < len(pool.connections); i++ {
		index := (pool.activeIndex + i) % len(pool.connections)
		conn := pool.connections[index]

		conn.mu.RLock()
		state := conn.conn.GetState()
		conn.mu.RUnlock()

		log.Printf("🔍 Connection[%d] to %s: target=%s state=%s", index, pool.serviceName, conn.target, state.String())

		// Accept both READY and IDLE connections as usable
		if state == connectivity.Ready || state == connectivity.Idle {
			readyConnections++
			pool.activeIndex = (index + 1) % len(pool.connections)

			conn.mu.Lock()
			conn.lastUsed = time.Now()
			conn.mu.Unlock()

			log.Printf("✅ Selected %s connection[%d] for %s: %s", state.String(), index, pool.serviceName, conn.target)
			return conn, nil
		}
	}

	log.Printf("🚨 No healthy connections found for %s: total=%d healthy=%d",
		pool.serviceName, len(pool.connections), readyConnections)
	return nil, fmt.Errorf("no healthy connections available for service %s", pool.serviceName)
}

// initializePool creates initial connections for the pool
func (cms *ConnectionManagerService) initializePool(pool *ServiceConnectionPool) {
	for i := 0; i < pool.minConnections; i++ {
		if err := cms.addConnectionToPool(pool); err != nil {
			log.Printf("⚠️ Failed to add initial connection to pool %s: %v", pool.serviceName, err)
		}
	}
}

// addConnectionToPool adds a new connection to the pool
func (cms *ConnectionManagerService) addConnectionToPool(pool *ServiceConnectionPool) error {
	// Service discovery via configured target (no dynamic registry)
	target := cms.getServiceTarget(pool.serviceName)
	if target == "" {
		return fmt.Errorf("unknown service: %s", pool.serviceName)
	}

	// Create gRPC connection with keepalive and timeout settings
	conn, err := grpc.DialContext(cms.ctx, target,
		grpc.WithInsecure(),
		grpc.WithTimeout(cms.config.ConnectionTimeout),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cms.config.KeepaliveTime,
			Timeout:             cms.config.KeepaliveTimeout,
			PermitWithoutStream: true,
		}),
	)

	if err != nil {
		return fmt.Errorf("gRPC dial failed: %w", err)
	}

	// Create managed connection
	managedConn := &ManagedConnection{
		conn:      conn,
		target:    target,
		state:     conn.GetState(),
		lastUsed:  time.Now(),
		createdAt: time.Now(),
		metadata:  make(map[string]string),
	}

	// Add basic metadata for configured connection
	managedConn.metadata["service_name"] = pool.serviceName
	managedConn.metadata["target"] = target

	pool.mu.Lock()
	pool.connections = append(pool.connections, managedConn)
	pool.mu.Unlock()

	log.Printf("✅ Added connection to pool %s: %s", pool.serviceName, target)
	return nil
}

// startBackgroundServices starts background monitoring and maintenance services
func (cms *ConnectionManagerService) startBackgroundServices() {
	// Health monitoring
	cms.wg.Add(1)
	go cms.healthMonitor()

	// Connection cleanup
	cms.wg.Add(1)
	go cms.connectionCleaner()

	// Service discovery refresh
	cms.wg.Add(1)
	go cms.serviceDiscoveryRefresh()
}

// healthMonitor performs periodic health checks on all connections
func (cms *ConnectionManagerService) healthMonitor() {
	defer cms.wg.Done()

	ticker := time.NewTicker(cms.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cms.performHealthChecks()
		case <-cms.ctx.Done():
			return
		}
	}
}

// performHealthChecks checks health of all connection pools
func (cms *ConnectionManagerService) performHealthChecks() {
	cms.mu.RLock()
	pools := make([]*ServiceConnectionPool, 0, len(cms.connections))
	for _, pool := range cms.connections {
		pools = append(pools, pool)
	}
	cms.mu.RUnlock()

	for _, pool := range pools {
		cms.checkPoolHealth(pool)
	}
}

// checkPoolHealth performs health check on a connection pool
func (cms *ConnectionManagerService) checkPoolHealth(pool *ServiceConnectionPool) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	healthyCount := 0
	totalCount := len(pool.connections)

	for _, conn := range pool.connections {
		conn.mu.Lock()
		state := conn.conn.GetState()
		conn.state = state

		// Consider both READY and IDLE as healthy states
		if state == connectivity.Ready || state == connectivity.Idle {
			healthyCount++
		} else if state == connectivity.Shutdown || state == connectivity.TransientFailure {
			conn.errorCount++
		}
		conn.mu.Unlock()
	}

	pool.lastHealthCheck = time.Now()
	pool.isHealthy = healthyCount > 0

	// Update circuit breaker state
	if healthyCount == 0 && totalCount > 0 {
		pool.failureCount++
		if pool.failureCount >= pool.maxFailures {
			pool.circuitOpen = true
			pool.circuitOpenTime = time.Now()
			log.Printf("🔴 Circuit breaker opened for service %s (failures: %d)", pool.serviceName, pool.failureCount)
		}
	} else if healthyCount > 0 {
		pool.failureCount = 0
		pool.circuitOpen = false
	}

	log.Printf("🔍 Health check for %s: %d/%d healthy connections", pool.serviceName, healthyCount, totalCount)
}

// connectionCleaner removes idle and failed connections
func (cms *ConnectionManagerService) connectionCleaner() {
	defer cms.wg.Done()

	ticker := time.NewTicker(60 * time.Second) // Clean every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cms.cleanupConnections()
		case <-cms.ctx.Done():
			return
		}
	}
}

// cleanupConnections removes idle and failed connections
func (cms *ConnectionManagerService) cleanupConnections() {
	cms.mu.RLock()
	pools := make([]*ServiceConnectionPool, 0, len(cms.connections))
	for _, pool := range cms.connections {
		pools = append(pools, pool)
	}
	cms.mu.RUnlock()

	for _, pool := range pools {
		cms.cleanupPool(pool)
	}
}

// cleanupPool cleans up a single connection pool
func (cms *ConnectionManagerService) cleanupPool(pool *ServiceConnectionPool) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	now := time.Now()
	activeConnections := make([]*ManagedConnection, 0)

	for _, conn := range pool.connections {
		conn.mu.RLock()
		shouldKeep := true

		// Remove connections that are shutdown or have been idle too long
		if conn.conn.GetState() == connectivity.Shutdown {
			shouldKeep = false
		} else if now.Sub(conn.lastUsed) > cms.config.IdleTimeout && len(activeConnections) >= pool.minConnections {
			shouldKeep = false
		}

		conn.mu.RUnlock()

		if shouldKeep {
			activeConnections = append(activeConnections, conn)
		} else {
			conn.conn.Close()
			log.Printf("🧹 Cleaned up idle connection to %s: %s", pool.serviceName, conn.target)
		}
	}

	pool.connections = activeConnections

	// Ensure minimum connections
	if len(pool.connections) < pool.minConnections {
		go cms.ensureMinimumConnections(pool)
	}
}

// ensureMinimumConnections ensures pool has minimum required connections
func (cms *ConnectionManagerService) ensureMinimumConnections(pool *ServiceConnectionPool) {
	needed := pool.minConnections - len(pool.connections)
	for i := 0; i < needed; i++ {
		if err := cms.addConnectionToPool(pool); err != nil {
			log.Printf("⚠️ Failed to restore minimum connection to %s: %v", pool.serviceName, err)
			break
		}
	}
}

// serviceDiscoveryRefresh periodically refreshes service discovery information
func (cms *ConnectionManagerService) serviceDiscoveryRefresh() {
	defer cms.wg.Done()

	ticker := time.NewTicker(cms.config.DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cms.refreshServiceDiscovery()
		case <-cms.ctx.Done():
			return
		}
	}
}

// refreshServiceDiscovery refreshes service discovery for all known services
func (cms *ConnectionManagerService) refreshServiceDiscovery() {
	cms.mu.RLock()
	serviceNames := make([]string, 0, len(cms.connections))
	for serviceName := range cms.connections {
		serviceNames = append(serviceNames, serviceName)
	}
	cms.mu.RUnlock()

	for _, serviceName := range serviceNames {
		cms.refreshServiceInfo(serviceName)
	}
}

// refreshServiceInfo refreshes service information for a specific service
func (cms *ConnectionManagerService) refreshServiceInfo(serviceName string) {
	// Ports are fixed - just log current target
	target := cms.getServiceTarget(serviceName)
	if target != "" {
		log.Printf("🔄 Service info for %s: %s", serviceName, target)
	}
}

// countHealthyConnections counts healthy connections in a pool
func (cms *ConnectionManagerService) countHealthyConnections(pool *ServiceConnectionPool) int {
	count := 0
	for _, conn := range pool.connections {
		conn.mu.RLock()
		if conn.conn.GetState() == connectivity.Ready {
			count++
		}
		conn.mu.RUnlock()
	}
	return count
}

// Shutdown gracefully shuts down the connection manager
func (cms *ConnectionManagerService) Shutdown() error {
	log.Printf("🛑 Shutting down Connection Manager Service...")

	cms.cancel()
	cms.wg.Wait()

	cms.mu.Lock()
	defer cms.mu.Unlock()

	var lastErr error
	for serviceName, pool := range cms.connections {
		pool.mu.Lock()
		for _, conn := range pool.connections {
			if err := conn.conn.Close(); err != nil {
				lastErr = err
				log.Printf("⚠️ Error closing connection to %s: %v", serviceName, err)
			}
		}
		pool.mu.Unlock()
	}

	return lastErr
}

func main() {
	log.Printf("🚀 Starting OpenUSP Connection Manager Service...")

	cfg := config.Load()
	grpcPort, _ := strconv.Atoi(cfg.ConnectionManagerGRPC)
	httpPortCfg, _ := strconv.Atoi(cfg.ConnectionManagerHTTP)
	if grpcPort == 0 || httpPortCfg == 0 {
		log.Fatalf("missing connection manager ports in YAML configuration")
	}

	// Command line flags (primarily for --help / --version); defaults from YAML
	var port = flag.Int("port", grpcPort, "gRPC port (from YAML)")
	var httpPort = flag.Int("http-port", httpPortCfg, "HTTP port for health/status (from YAML)")
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.GetFullVersion("OpenUSP Connection Manager Service"))
		return
	}

	if *showHelp {
		fmt.Println("OpenUSP Connection Manager Service - gRPC Connection Management")
		fmt.Println("============================================================")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  connection-manager [flags]")
		fmt.Println("")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Ports sourced from configs/openusp.yml; environment overrides enforced.")
		return
	}

	// Environment-based port configuration

	// Create connection manager service
	connectionManager := NewConnectionManagerService(DefaultConnectionManagerConfig())

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	// Register services
	connectionservice.RegisterConnectionServiceServer(grpcServer, connectionManager)
	grpc_health_v1.RegisterHealthServer(grpcServer, health.NewServer())
	reflection.Register(grpcServer)

	// Determine HTTP port (configured)
	httpPortToUse := *httpPort

	// Start HTTP server for health checks and status
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status":    "healthy",
			"service":   "connection-manager",
			"timestamp": time.Now().Unix(),
			"uptime":    time.Since(connectionManager.startTime).String(),
		}
		json.NewEncoder(w).Encode(response)
	})

	httpMux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get connection status
		statusReq := &connectionservice.GetConnectionStatusRequest{}
		statusResp, _ := connectionManager.GetConnectionStatus(context.Background(), statusReq)

		response := map[string]interface{}{
			"service":     "OpenUSP Connection Manager Service",
			"version":     "1.0.0",
			"uptime":      time.Since(connectionManager.startTime).String(),
			"grpc_port":   connectionManager.actualGRPCPort,
			"http_port":   connectionManager.httpPort,
			"connections": statusResp.Services,
		}
		json.NewEncoder(w).Encode(response)
	})

	httpMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement Prometheus metrics endpoint
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "# OpenUSP Connection Manager Metrics")
		fmt.Fprintln(w, "# TODO: Implement Prometheus metrics")
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPortToUse),
		Handler: httpMux,
	}

	// Start HTTP server
	go func() {
		log.Printf("🌐 HTTP server starting on port %d", httpPortToUse)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("⚠️ HTTP server error: %v", err)
		}
	}()

	// Start gRPC server - environment-based port configuration
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port: %v", err)
	}

	actualGRPCPort := listener.Addr().(*net.TCPAddr).Port
	log.Printf("🚀 Connection Manager gRPC server starting on port %d", actualGRPCPort)

	// Update connection manager with actual ports for status reporting
	connectionManager.SetPorts(actualGRPCPort, httpPortToUse)

	// Environment-based port configuration - no service registration needed

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("🛑 Shutdown signal received...")

		// Graceful shutdown
		connectionManager.Shutdown()

		// Stop HTTP server
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)

		// Stop gRPC server
		grpcServer.GracefulStop()
	}()

	// Log startup information
	log.Printf("🚀 Connection Manager Service started successfully")
	log.Printf("   └── gRPC Port: %d", actualGRPCPort)
	log.Printf("   └── HTTP Port: %d", httpPortToUse)
	log.Printf("   └── Health Check: http://localhost:%d/health", httpPortToUse)
	log.Printf("   └── Status: http://localhost:%d/status", httpPortToUse)

	log.Printf("💡 Press Ctrl+C to exit...")

	// Start serving
	log.Printf("🚀 Connection Manager gRPC server starting on port %d", actualGRPCPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
