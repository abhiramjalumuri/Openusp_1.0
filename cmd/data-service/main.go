package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"openusp/internal/database"
	grpcserver "openusp/internal/grpc"
	"openusp/pkg/config"
	"openusp/pkg/consul"
	"openusp/pkg/metrics"
	pb "openusp/pkg/proto/dataservice"
	"openusp/pkg/version"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

// DataService represents a modern unified data service
type DataService struct {
	config      *config.DeploymentConfig
	registry    *consul.ServiceRegistry
	serviceInfo *consul.ServiceInfo
	grpcServer  *grpc.Server
	httpServer  *http.Server
	healthSrv   *health.Server
	database    *database.Database
	repos       *database.Repositories
	metrics     *metrics.OpenUSPMetrics
}

func main() {
	log.Printf("🚀 Starting OpenUSP Data Service...")

	// Command line flags
	var enableConsul = flag.Bool("consul", false, "Enable Consul service discovery")
	var port = flag.Int("port", 6400, "HTTP port")
	var showVersion = flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Println(version.GetFullVersion("OpenUSP Data Service"))
		return
	} // Override environment from flags
	if *enableConsul {
		os.Setenv("CONSUL_ENABLED", "true")
	}
	if *port != 6400 {
		os.Setenv("SERVICE_PORT", fmt.Sprintf("%d", *port))
	}

	// Create and start the service
	service, err := NewDataService()
	if err != nil {
		log.Fatalf("Failed to create data service: %v", err)
	}

	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start data service: %v", err)
	}

	// Wait for shutdown
	service.handleShutdown()
}

func NewDataService() (*DataService, error) {
	// Load configuration
	config := config.LoadDeploymentConfigWithPortEnv("openusp-data-service", "data-service", 6400, "OPENUSP_DATA_SERVICE_HTTP_PORT")

	service := &DataService{
		config:  config,
		metrics: metrics.NewOpenUSPMetrics("data-service"),
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
		service.registry = registry

		log.Printf("🏛️ Connected to Consul at %s", consulAddr)
	}

	// Initialize database
	db, err := database.NewDatabase(nil) // Use default config with environment variables
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	service.database = db

	// Run database migrations
	if err := db.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	service.repos = database.NewRepositories(db.DB)

	// Setup servers (but don't register with Consul yet)
	service.setupgRPCServer()
	service.setupHTTPServer()

	return service, nil
}

func (ds *DataService) registerService() error {
	if ds.registry == nil {
		log.Printf("🎯 Service starting: %s at localhost:%d (no service discovery)",
			ds.config.ServiceName, ds.config.ServicePort)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := ds.registry.RegisterServiceWithPorts(ctx, ds.serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to register with Consul: %w", err)
	}

	return nil
}

func (ds *DataService) setupgRPCServer() {
	// Configure gRPC server with keepalive enforcement to prevent ENHANCE_YOUR_CALM errors
	ds.grpcServer = grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    60 * time.Second, // Send keepalive pings every 60 seconds
			Timeout: 10 * time.Second, // Wait 10 seconds for keepalive ping ack
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second, // Minimum allowed time between client pings
			PermitWithoutStream: true,             // Allow pings even when no streams are active
		}),
	)
	ds.healthSrv = health.NewServer()

	// Register health service
	grpc_health_v1.RegisterHealthServer(ds.grpcServer, ds.healthSrv)

	// Register data service
	dataServiceServer := grpcserver.NewDataServiceServer(ds.database, ds.repos)
	pb.RegisterDataServiceServer(ds.grpcServer, dataServiceServer)

	// Set healthy status
	ds.healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
}

func (ds *DataService) setupHTTPServer() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health and status endpoints
	router.GET("/health", ds.healthHandler)
	router.GET("/status", ds.statusHandler)
	router.GET("/metrics", gin.WrapH(metrics.HTTPHandler()))

	port := ds.getHTTPPort()
	ds.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
}

func (ds *DataService) healthHandler(c *gin.Context) {
	// Test database connection
	dbStatus := "connected"
	if err := ds.database.Ping(); err != nil {
		dbStatus = "disconnected"
	}

	c.JSON(http.StatusOK, gin.H{
		"service":   "openusp-data-service",
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"http_port": ds.getHTTPPort(),
		"grpc_port": ds.getgRPCPort(),
		"consul":    ds.config.IsConsulEnabled(),
		"database":  dbStatus,
	})
}

func (ds *DataService) statusHandler(c *gin.Context) {
	consulInfo := gin.H{"enabled": false}
	if ds.config.IsConsulEnabled() && ds.serviceInfo != nil {
		consulInfo = gin.H{
			"enabled":      true,
			"service_id":   ds.serviceInfo.ID,
			"service_name": ds.serviceInfo.Name,
			"address":      ds.serviceInfo.Address,
			"port":         ds.serviceInfo.Port,
			"grpc_port":    ds.serviceInfo.GRPCPort,
			"health":       ds.serviceInfo.Health,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"service":        "openusp-data-service",
		"version":        version.Version,
		"status":         "running",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"http_port":      ds.getHTTPPort(),
		"grpc_port":      ds.getgRPCPort(),
		"consul":         consulInfo,
		"database_stats": ds.database.GetStats(),
	})
}

func (ds *DataService) getHTTPPort() int {
	if ds.serviceInfo != nil {
		return ds.serviceInfo.Port
	}
	return ds.config.ServicePort
}

func (ds *DataService) getgRPCPort() int {
	if ds.serviceInfo != nil && ds.serviceInfo.GRPCPort > 0 {
		return ds.serviceInfo.GRPCPort
	}
	return ds.config.ServicePort + 1000 // 6400 -> 56400
}

func (ds *DataService) Start() error {
	// Initialize service info if using Consul
	if ds.config.IsConsulEnabled() {
		ds.serviceInfo = &consul.ServiceInfo{
			Name:     ds.config.ServiceName,
			Address:  "localhost",
			Port:     0, // Will be set by HTTP server
			GRPCPort: 0, // Will be set by gRPC server
			Protocol: "grpc",
			Meta: map[string]string{
				"service_type": ds.config.ServiceType,
				"protocol":     "grpc",
				"version":      "1.0.0",
				"environment":  "development",
			},
		}
	}

	// Start gRPC server
	go ds.startgRPCServer()

	// Start HTTP server
	go ds.startHTTPServer()

	// Wait for servers to set their ports
	time.Sleep(2 * time.Second)

	// Register with service discovery AFTER servers have set their ports
	if ds.config.IsConsulEnabled() {
		if err := ds.registerService(); err != nil {
			log.Printf("⚠️  Failed to register with Consul: %v", err)
		}
	}

	log.Printf("🚀 Data Service started successfully")
	if ds.config.IsConsulEnabled() && ds.serviceInfo != nil {
		log.Printf("   └── gRPC Port: %d", ds.serviceInfo.GRPCPort)
		log.Printf("   └── HTTP Port: %d", ds.serviceInfo.Port)
		log.Printf("   └── Consul Service Discovery: ✅ Enabled")
		log.Printf("   └── Health Check: http://localhost:%d/health", ds.serviceInfo.Port)
		log.Printf("   └── Status: http://localhost:%d/status", ds.serviceInfo.Port)
		log.Printf("   └── Consul UI: http://localhost:8500/ui/")
	} else {
		httpPort := ds.getHTTPPort()
		grpcPort := ds.getgRPCPort()
		log.Printf("   └── gRPC Port: %d", grpcPort)
		log.Printf("   └── HTTP Port: %d", httpPort)
		log.Printf("   └── Consul Service Discovery: ❌ Disabled")
		log.Printf("   └── Health Check: http://localhost:%d/health", httpPort)
		log.Printf("   └── Status: http://localhost:%d/status", httpPort)
	}

	return nil
}

func (ds *DataService) startgRPCServer() {
	var lis net.Listener
	var err error

	if ds.config.IsConsulEnabled() {
		// Dynamic port for Consul - bind to IPv4 specifically
		lis, err = net.Listen("tcp4", "127.0.0.1:0")
	} else {
		// Fixed port for traditional deployment
		lis, err = net.Listen("tcp", fmt.Sprintf(":%d", ds.getgRPCPort()))
	}

	if err != nil {
		log.Fatalf("Failed to create gRPC listener: %v", err)
	}

	actualPort := lis.Addr().(*net.TCPAddr).Port
	log.Printf("🔌 Starting gRPC server on port %d", actualPort)

	// Update service info with actual gRPC port if using Consul
	if ds.config.IsConsulEnabled() && ds.serviceInfo != nil {
		ds.serviceInfo.GRPCPort = actualPort
		ds.serviceInfo.Meta["grpc_port"] = strconv.Itoa(actualPort)
		log.Printf("🎯 Updated service info with actual gRPC port: %d", actualPort)
	}

	if err := ds.grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func (ds *DataService) startHTTPServer() {
	var listener net.Listener
	var err error

	if ds.config.IsConsulEnabled() {
		// Dynamic port for Consul - bind to IPv4 specifically
		listener, err = net.Listen("tcp4", "127.0.0.1:0")
	} else {
		// Fixed port for traditional deployment
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", ds.getHTTPPort()))
	}

	if err != nil {
		log.Fatalf("Failed to create HTTP listener: %v", err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	log.Printf("🔌 Starting HTTP server on port %d", actualPort)

	// Update service info with actual HTTP port if using Consul
	if ds.config.IsConsulEnabled() && ds.serviceInfo != nil {
		ds.serviceInfo.Port = actualPort
	}

	if err := ds.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func (ds *DataService) Stop() error {
	log.Printf("🛑 Shutting down Data Service...")

	// Deregister from Consul
	if ds.registry != nil && ds.serviceInfo != nil {
		if err := ds.registry.DeregisterService(ds.serviceInfo.ID); err != nil {
			log.Printf("⚠️  Failed to deregister from Consul: %v", err)
		} else {
			log.Printf("✅ Deregistered from Consul successfully")
		}
	}

	// Stop gRPC server
	if ds.grpcServer != nil {
		ds.grpcServer.GracefulStop()
	}

	// Stop HTTP server
	if ds.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ds.httpServer.Shutdown(ctx)
	}

	// Close database
	if ds.database != nil {
		ds.database.Close()
	}

	log.Printf("✅ Data Service stopped successfully")
	return nil
}

func (ds *DataService) handleShutdown() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	sig := <-signalChan
	log.Printf("🔔 Received signal: %v", sig)

	ds.Stop()
}
