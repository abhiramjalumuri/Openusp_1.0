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
	config     *config.DeploymentConfig
	httpPort   int
	grpcPort   int
	grpcServer *grpc.Server
	httpServer *http.Server
	healthSrv  *health.Server
	database   *database.Database
	repos      *database.Repositories
	metrics    *metrics.OpenUSPMetrics
}

func main() {
	log.Printf("🚀 Starting OpenUSP Data Service...")

	// Command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Println(version.GetFullVersion("OpenUSP Data Service"))
		return
	}

	// Fixed ports – no environment overrides needed

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
	fullConfig := config.Load()
	grpcPort, _ := strconv.Atoi(fullConfig.DataServiceGRPCPort)
	httpPortYAML, _ := strconv.Atoi(fullConfig.DataServiceHTTPPort)
	if grpcPort == 0 || httpPortYAML == 0 {
		return nil, fmt.Errorf("missing data service ports in YAML configuration")
	}
	deploymentConfig := &config.DeploymentConfig{
		ServicePort: httpPortYAML,
		ServiceName: "openusp-data-service",
		ServiceType: "data-service",
	}

	service := &DataService{
		config:   deploymentConfig,
		httpPort: httpPortYAML,
		grpcPort: grpcPort,
		metrics:  metrics.NewOpenUSPMetrics("data-service"),
	}

	// Fixed ports – no service discovery needed

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

	// Setup servers
	service.setupgRPCServer()
	service.setupHTTPServer()

	return service, nil
}

// Service registration removed - using fixed ports

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
		"database":  dbStatus,
	})
}

func (ds *DataService) statusHandler(c *gin.Context) {
	// Port configuration status
	c.JSON(http.StatusOK, gin.H{
		"service":        "openusp-data-service",
		"version":        version.Version,
		"status":         "running",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"http_port":      ds.getHTTPPort(),
		"grpc_port":      ds.getgRPCPort(),
		"database_stats": ds.database.GetStats(),
		"service_discovery": gin.H{
			"enabled":      false,
			"type":         "static_ports",
			"service_name": ds.config.ServiceName,
			"address":      "localhost",
			"port":         ds.config.ServicePort,
			"grpc_port":    ds.config.ServicePort + 1,
			"health":       "running",
		},
	})
}

func (ds *DataService) getHTTPPort() int { return ds.httpPort }

func (ds *DataService) getgRPCPort() int { return ds.grpcPort }

func (ds *DataService) Start() error {
	// Environment-based port configuration
	log.Printf("🎯 Data Service starting with environment configuration:")
	log.Printf("   └── HTTP Port: %d", ds.getHTTPPort())
	log.Printf("   └── gRPC Port: %d", ds.getgRPCPort())

	// Start gRPC server
	go ds.startgRPCServer()

	// Start HTTP server
	go ds.startHTTPServer()

	// Wait for servers to set their ports
	time.Sleep(2 * time.Second)

	// No service registration needed with environment configuration
	log.Printf("✅ Data Service servers started with environment configuration")

	log.Printf("🚀 Data Service started successfully")
	httpPort := ds.getHTTPPort()
	grpcPort := ds.getgRPCPort()
	log.Printf("   └── gRPC Port: %d", grpcPort)
	log.Printf("   └── HTTP Port: %d", httpPort)
	log.Printf("   └── Service Discovery: Environment configuration")
	log.Printf("   └── Health Check: http://localhost:%d/health", httpPort)
	log.Printf("   └── Status: http://localhost:%d/status", httpPort)

	return nil
}

func (ds *DataService) startgRPCServer() {
	// Listen on fixed port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", ds.getgRPCPort()))
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
	// Listen on fixed port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", ds.getHTTPPort()))
	if err != nil {
		log.Fatalf("Failed to create HTTP listener: %v", err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	log.Printf("🔌 Starting HTTP server on port %d", actualPort)

	if err := ds.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func (ds *DataService) Stop() error {
	log.Printf("🛑 Shutting down Data Service...")

	// Fixed ports – no service deregistration needed

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
