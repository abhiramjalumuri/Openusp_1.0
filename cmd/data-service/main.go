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
	config := config.LoadDeploymentConfig("openusp-data-service", "data-service", 6400)

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
	service.repos = database.NewRepositories(db.DB)

	// Register with service discovery
	if err := service.registerService(); err != nil {
		return nil, fmt.Errorf("failed to register service: %w", err)
	}

	// Setup servers
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

	serviceInfo, err := ds.registry.RegisterService(ctx, ds.config.ServiceName, ds.config.ServiceType)
	if err != nil {
		return fmt.Errorf("failed to register with Consul: %w", err)
	}

	ds.serviceInfo = serviceInfo
	log.Printf("🎯 Service registered with Consul: %s (%s) at localhost:%d",
		serviceInfo.Name, serviceInfo.Meta["service_type"], serviceInfo.Port)

	return nil
}

func (ds *DataService) setupgRPCServer() {
	ds.grpcServer = grpc.NewServer()
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
	// Start gRPC server
	go ds.startgRPCServer()

	// Start HTTP server
	go ds.startHTTPServer()

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
		// Dynamic port for Consul
		lis, err = net.Listen("tcp", ":0")
	} else {
		// Fixed port for traditional deployment
		lis, err = net.Listen("tcp", fmt.Sprintf(":%d", ds.getgRPCPort()))
	}

	if err != nil {
		log.Fatalf("Failed to create gRPC listener: %v", err)
	}

	actualPort := lis.Addr().(*net.TCPAddr).Port
	log.Printf("🔌 Starting gRPC server on port %d", actualPort)

	if err := ds.grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}

func (ds *DataService) startHTTPServer() {
	log.Printf("🔌 Starting HTTP server on port %d", ds.getHTTPPort())
	if err := ds.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
