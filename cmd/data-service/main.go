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

	confluentkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"

	"openusp/internal/database"
	"openusp/pkg/config"
	"openusp/pkg/kafka"
	"openusp/pkg/metrics"
	"openusp/pkg/version"
)

// DataService represents a modern unified data service
type DataService struct {
	config        *config.Config
	httpPort      int
	httpServer    *http.Server
	database      *database.Database
	repos         *database.Repositories
	metrics       *metrics.OpenUSPMetrics
	kafkaClient   *kafka.Client
	kafkaConsumer *kafka.Consumer
	kafkaProducer *kafka.Producer
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
	cfg := config.Load()

	// Convert HTTP port from string to int
	httpPort, err := strconv.Atoi(cfg.DataServiceHTTPPort)
	if err != nil || httpPort == 0 {
		return nil, fmt.Errorf("invalid data service HTTP port: %s", cfg.DataServiceHTTPPort)
	}

	service := &DataService{
		config:   cfg,
		httpPort: httpPort,
		metrics:  metrics.NewOpenUSPMetrics("data-service"),
	}

	// Initialize Kafka client
	kafkaClient, err := kafka.NewClient(&cfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka client: %w", err)
	}
	service.kafkaClient = kafkaClient

	// Ensure Kafka topics exist
	topicsManager := kafka.NewTopicsManager(kafkaClient, &cfg.Kafka.Topics)
	if err := topicsManager.EnsureAllTopicsExist(); err != nil {
		return nil, fmt.Errorf("failed to create Kafka topics: %w", err)
	}
	log.Printf("✅ Kafka topics validated/created")

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(kafkaClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}
	service.kafkaProducer = kafkaProducer

	// Initialize Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(&cfg.Kafka, "data-service")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka consumer: %w", err)
	}
	service.kafkaConsumer = kafkaConsumer

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

	// Setup HTTP server
	service.setupHTTPServer()

	// Setup Kafka event handlers
	service.setupKafkaConsumers()

	return service, nil
}

// setupKafkaConsumers configures Kafka topic subscriptions and handlers
func (ds *DataService) setupKafkaConsumers() {
	// Subscribe to data topics for consuming events
	topics := []string{
		ds.config.Kafka.Topics.DataDeviceCreated,
		ds.config.Kafka.Topics.DataDeviceUpdated,
		ds.config.Kafka.Topics.DataDeviceDeleted,
		ds.config.Kafka.Topics.DataParameterUpdated,
		ds.config.Kafka.Topics.DataObjectCreated,
		ds.config.Kafka.Topics.DataAlertCreated,
		ds.config.Kafka.Topics.APIRequest, // Handle API requests from API Gateway
	}

	// Define message handler - takes *confluentkafka.Message
	handler := func(msg *confluentkafka.Message) error {
		topic := *msg.TopicPartition.Topic
		log.Printf("📨 Received message on topic %s: %d bytes", topic, len(msg.Value))

		switch topic {
		case ds.config.Kafka.Topics.DataDeviceCreated:
			return ds.handleDeviceCreatedEvent(msg)
		case ds.config.Kafka.Topics.DataDeviceUpdated:
			return ds.handleDeviceUpdatedEvent(msg)
		case ds.config.Kafka.Topics.DataDeviceDeleted:
			return ds.handleDeviceDeletedEvent(msg)
		case ds.config.Kafka.Topics.DataParameterUpdated:
			return ds.handleParameterUpdatedEvent(msg)
		case ds.config.Kafka.Topics.DataObjectCreated:
			return ds.handleObjectCreatedEvent(msg)
		case ds.config.Kafka.Topics.DataAlertCreated:
			return ds.handleAlertCreatedEvent(msg)
		case ds.config.Kafka.Topics.APIRequest:
			return ds.handleAPIRequest(msg)
		default:
			log.Printf("⚠️  Unknown topic: %s", topic)
			return nil
		}
	}

	// Subscribe to topics
	if err := ds.kafkaConsumer.Subscribe(topics, handler); err != nil {
		log.Printf("❌ Failed to subscribe to Kafka topics: %v", err)
	} else {
		log.Printf("✅ Subscribed to %d Kafka topics", len(topics))
	}
}

// Event handlers for different data topics
func (ds *DataService) handleDeviceCreatedEvent(msg *confluentkafka.Message) error {
	event, err := kafka.ConsumeDeviceEvent(msg)
	if err != nil {
		return fmt.Errorf("failed to parse device created event: %w", err)
	}

	log.Printf("📱 Device created event received: %s (endpoint: %s)", event.DeviceID, event.EndpointID)

	// Extract device data from event
	device := &database.Device{
		EndpointID:     event.EndpointID,
		Manufacturer:   event.Manufacturer,
		ModelName:      event.ModelName,
		SerialNumber:   event.SerialNumber,
		ProductClass:   event.ProductClass,
		Status:         event.Status,
		ConnectionType: event.Protocol, // USP protocol type
	}

	// Extract additional fields from Data map if available
	if event.Data != nil {
		if sv, ok := event.Data["software_version"].(string); ok {
			device.SoftwareVersion = sv
		}
		if hv, ok := event.Data["hardware_version"].(string); ok {
			device.HardwareVersion = hv
		}
		if ip, ok := event.Data["ip_address"].(string); ok {
			device.IPAddress = ip
		}
	}

	// Use CreateOrUpdate to handle both new devices and onboarding updates
	if err := ds.repos.Device.CreateOrUpdate(device); err != nil {
		log.Printf("❌ Failed to persist device %s to database: %v", event.EndpointID, err)
		return fmt.Errorf("failed to persist device to database: %w", err)
	}

	log.Printf("✅ Device persisted to database: %s (manufacturer: %s, model: %s, serial: %s)",
		event.EndpointID, event.Manufacturer, event.ModelName, event.SerialNumber)

	return nil
}

func (ds *DataService) handleDeviceUpdatedEvent(msg *confluentkafka.Message) error {
	event, err := kafka.ConsumeDeviceEvent(msg)
	if err != nil {
		return fmt.Errorf("failed to parse device updated event: %w", err)
	}

	log.Printf("🔄 Device updated: %s", event.DeviceID)
	return nil
}

func (ds *DataService) handleDeviceDeletedEvent(msg *confluentkafka.Message) error {
	event, err := kafka.ConsumeDeviceEvent(msg)
	if err != nil {
		return fmt.Errorf("failed to parse device deleted event: %w", err)
	}

	log.Printf("🗑️  Device deleted: %s", event.DeviceID)
	return nil
}

func (ds *DataService) handleParameterUpdatedEvent(msg *confluentkafka.Message) error {
	event, err := kafka.ConsumeParameterEvent(msg)
	if err != nil {
		return fmt.Errorf("failed to parse parameter updated event: %w", err)
	}

	log.Printf("⚙️  Parameter updated: %s = %v", event.ParameterPath, event.NewValue)
	return nil
}

func (ds *DataService) handleObjectCreatedEvent(msg *confluentkafka.Message) error {
	log.Printf("📦 Object created event received")
	// TODO: Parse and handle object created event
	return nil
}

func (ds *DataService) handleAlertCreatedEvent(msg *confluentkafka.Message) error {
	log.Printf("🚨 Alert created event received")
	// TODO: Parse and handle alert created event
	return nil
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

	// Test Kafka connection
	kafkaStatus := "connected"
	if err := ds.kafkaClient.Ping(); err != nil {
		kafkaStatus = "disconnected"
	}

	c.JSON(http.StatusOK, gin.H{
		"service":   "openusp-data-service",
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"http_port": ds.getHTTPPort(),
		"database":  dbStatus,
		"kafka":     kafkaStatus,
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
		"database_stats": ds.database.GetStats(),
		"kafka_brokers":  ds.config.Kafka.Brokers,
		"kafka_topics": gin.H{
			"device_created":    ds.config.Kafka.Topics.DataDeviceCreated,
			"device_updated":    ds.config.Kafka.Topics.DataDeviceUpdated,
			"parameter_updated": ds.config.Kafka.Topics.DataParameterUpdated,
		},
	})
}

func (ds *DataService) getHTTPPort() int { return ds.httpPort }

func (ds *DataService) Start() error {
	log.Printf("🎯 Data Service starting with Kafka configuration:")
	log.Printf("   └── HTTP Port: %d", ds.getHTTPPort())
	log.Printf("   └── Kafka Brokers: %v", ds.config.Kafka.Brokers)

	// Start Kafka consumer
	go ds.kafkaConsumer.Start()

	// Start HTTP server
	go ds.startHTTPServer()

	// Wait for servers to initialize
	time.Sleep(2 * time.Second)

	log.Printf("🚀 Data Service started successfully")
	httpPort := ds.getHTTPPort()
	log.Printf("   └── HTTP Port: %d", httpPort)
	log.Printf("   └── Kafka Consumer Group: data-service")
	log.Printf("   └── Subscribed Topics: 6 data topics")
	log.Printf("   └── Health Check: http://localhost:%d/health", httpPort)
	log.Printf("   └── Status: http://localhost:%d/status", httpPort)

	return nil
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

	// Stop Kafka consumer
	if ds.kafkaConsumer != nil {
		log.Printf("⏹️  Stopping Kafka consumer...")
		ds.kafkaConsumer.Close()
	}

	// Stop Kafka producer
	if ds.kafkaProducer != nil {
		log.Printf("⏹️  Flushing and closing Kafka producer...")
		ds.kafkaProducer.Close()
	}

	// Close Kafka client
	if ds.kafkaClient != nil {
		log.Printf("⏹️  Closing Kafka client...")
		ds.kafkaClient.Close()
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
