// Package main implements the OpenUSP API Gateway service
//
//	@title						OpenUSP API Gateway
//	@version					1.0
//	@description				OpenUSP API Gateway provides RESTful APIs for managing TR-369 USP and TR-069 CWMP devices
//	@termsOfService				http://swagger.io/terms/
//
//	@contact.name				OpenUSP Support
//	@contact.url				https://github.com/plume-design-inc/openusp
//	@contact.email				support@openusp.io
//
//	@license.name				Apache 2.0
//	@license.url				http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@host						localhost:8080
//	@BasePath					/api/v1
//
//	@schemes					http https
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description				API key authentication. Use "Bearer <token>" format.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"openusp/pkg/config"
	"openusp/pkg/kafka"
	"openusp/pkg/metrics"
	"openusp/pkg/version"

	"github.com/gin-gonic/gin"
)

type APIGateway struct {
	config      *config.DeploymentConfig
	fullConfig  *config.Config
	router      *gin.Engine
	server      *http.Server
	kafkaClient *kafka.RequestResponseClient
	metrics     *metrics.OpenUSPMetrics
}

func NewAPIGateway() (*APIGateway, error) {
	fullConfig := config.Load()
	servicePort, _ := strconv.Atoi(fullConfig.APIGatewayPort)
	deploymentConfig := &config.DeploymentConfig{
		ServicePort: servicePort,
		ServiceName: "openusp-api-gateway",
		ServiceType: "api-gateway",
	}

	gateway := &APIGateway{
		config:     deploymentConfig,
		fullConfig: fullConfig,
		metrics:    metrics.NewOpenUSPMetrics("api-gateway"),
	}

	log.Printf("API Gateway starting at localhost:%d (Kafka-based)", gateway.config.ServicePort)

	kafkaClient, err := kafka.NewRequestResponseClient(
		fullConfig.Kafka.Brokers,
		fullConfig.Kafka.Topics.APIRequest,
		fullConfig.Kafka.Topics.APIResponse,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}
	gateway.kafkaClient = kafkaClient

	gateway.setupRoutes()
	return gateway, nil
}

func (gw *APIGateway) setupRoutes() {
	gin.SetMode(gin.ReleaseMode)
	gw.router = gin.Default()
	gw.router.Use(gw.corsMiddleware())
	gw.router.Use(gw.metricsMiddleware())

	gw.router.GET("/health", gw.healthCheck)
	gw.router.GET("/status", gw.getStatus)
	gw.router.GET("/metrics", gin.WrapH(metrics.HTTPHandler()))

	v1 := gw.router.Group("/api/v1")
	{
		devices := v1.Group("/devices")
		{
			devices.GET("", gw.listDevices)
			devices.POST("", gw.provisionDevice)
			devices.GET("/:device_id", gw.getDevice)
			devices.PUT("/:device_id", gw.updateDevice)
			devices.DELETE("/:device_id", gw.deleteDevice)
			devices.GET("/:device_id/parameters", gw.getParameters)
			devices.PUT("/:device_id/parameters", gw.setParameter)
			devices.POST("/:device_id/parameters/get", gw.bulkGetParameters)
			devices.POST("/:device_id/parameters/set", gw.bulkSetParameters)
			devices.GET("/:device_id/parameters/search", gw.searchParameters)
			devices.GET("/:device_id/objects", gw.getObjects)
			devices.GET("/:device_id/objects/*object_path", gw.getObjectByPath)
			devices.POST("/:device_id/commands/execute", gw.executeCommand)
			devices.POST("/:device_id/subscriptions", gw.createSubscription)
			devices.GET("/:device_id/subscriptions", gw.getSubscriptions)
			devices.DELETE("/:device_id/subscriptions/:subscription_id", gw.deleteSubscription)
			devices.GET("/:device_id/datamodel", gw.getDataModel)
			devices.GET("/:device_id/schema", gw.getSchema)
			devices.GET("/:device_id/tree", gw.getObjectTree)
			devices.GET("/:device_id/alerts", gw.getDeviceAlerts)
			devices.GET("/:device_id/sessions", gw.getDeviceSessions)
		}
		alerts := v1.Group("/alerts")
		{
			alerts.GET("", gw.listAlerts)
			alerts.POST("", gw.createAlert)
			alerts.PUT("/:id/resolve", gw.resolveAlert)
		}
		sessions := v1.Group("/sessions")
		{
			sessions.GET("", gw.listSessions)
			sessions.GET("/:id", gw.getSession)
		}
	}

	gw.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", gw.config.ServicePort),
		Handler: gw.router,
	}
}

func (gw *APIGateway) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (gw *APIGateway) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		gw.metrics.RecordHTTPRequest("api-gateway", c.Request.Method, c.FullPath(), status, duration)
	}
}

func (gw *APIGateway) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "API Gateway", "mode": "kafka"})
}

func (gw *APIGateway) getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "running", "service": "API Gateway", "mode": "kafka"})
}

func (gw *APIGateway) sendKafkaRequest(c *gin.Context, operation string) {
	var body map[string]interface{}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			body = nil
		}
	}

	params := make(map[string]string)
	for _, param := range c.Params {
		params[param.Key] = param.Value
	}

	query := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			query[key] = values[0]
		}
	}

	req := &kafka.APIRequest{
		Operation: operation,
		Path:      c.FullPath(),
		Method:    c.Request.Method,
		Params:    params,
		Query:     query,
		Body:      body,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := gw.kafkaClient.SendRequest(ctx, req)
	if err != nil {
		log.Printf("Kafka request failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if resp.Error != "" {
		c.JSON(resp.Status, gin.H{"error": resp.Error})
	} else {
		c.JSON(resp.Status, resp.Data)
	}
}

func (gw *APIGateway) listDevices(c *gin.Context)       { gw.sendKafkaRequest(c, "ListDevices") }
func (gw *APIGateway) getDevice(c *gin.Context)         { gw.sendKafkaRequest(c, "GetDevice") }
func (gw *APIGateway) provisionDevice(c *gin.Context)   { gw.sendKafkaRequest(c, "ProvisionDevice") }
func (gw *APIGateway) updateDevice(c *gin.Context)      { gw.sendKafkaRequest(c, "UpdateDevice") }
func (gw *APIGateway) deleteDevice(c *gin.Context)      { gw.sendKafkaRequest(c, "DeleteDevice") }
func (gw *APIGateway) getParameters(c *gin.Context)     { gw.sendKafkaRequest(c, "GetParameters") }
func (gw *APIGateway) setParameter(c *gin.Context)      { gw.sendKafkaRequest(c, "SetParameter") }
func (gw *APIGateway) bulkGetParameters(c *gin.Context) { gw.sendKafkaRequest(c, "BulkGetParameters") }
func (gw *APIGateway) bulkSetParameters(c *gin.Context) { gw.sendKafkaRequest(c, "BulkSetParameters") }
func (gw *APIGateway) searchParameters(c *gin.Context)  { gw.sendKafkaRequest(c, "SearchParameters") }
func (gw *APIGateway) getObjects(c *gin.Context)        { gw.sendKafkaRequest(c, "GetObjects") }
func (gw *APIGateway) getObjectByPath(c *gin.Context)   { gw.sendKafkaRequest(c, "GetObjectByPath") }
func (gw *APIGateway) executeCommand(c *gin.Context)    { gw.sendKafkaRequest(c, "ExecuteCommand") }
func (gw *APIGateway) createSubscription(c *gin.Context) {
	gw.sendKafkaRequest(c, "CreateSubscription")
}
func (gw *APIGateway) getSubscriptions(c *gin.Context) { gw.sendKafkaRequest(c, "GetSubscriptions") }
func (gw *APIGateway) deleteSubscription(c *gin.Context) {
	gw.sendKafkaRequest(c, "DeleteSubscription")
}
func (gw *APIGateway) getDataModel(c *gin.Context)      { gw.sendKafkaRequest(c, "GetDataModel") }
func (gw *APIGateway) getSchema(c *gin.Context)         { gw.sendKafkaRequest(c, "GetSchema") }
func (gw *APIGateway) getObjectTree(c *gin.Context)     { gw.sendKafkaRequest(c, "GetObjectTree") }
func (gw *APIGateway) listAlerts(c *gin.Context)        { gw.sendKafkaRequest(c, "ListAlerts") }
func (gw *APIGateway) createAlert(c *gin.Context)       { gw.sendKafkaRequest(c, "CreateAlert") }
func (gw *APIGateway) resolveAlert(c *gin.Context)      { gw.sendKafkaRequest(c, "ResolveAlert") }
func (gw *APIGateway) getDeviceAlerts(c *gin.Context)   { gw.sendKafkaRequest(c, "GetDeviceAlerts") }
func (gw *APIGateway) listSessions(c *gin.Context)      { gw.sendKafkaRequest(c, "ListSessions") }
func (gw *APIGateway) getSession(c *gin.Context)        { gw.sendKafkaRequest(c, "GetSession") }
func (gw *APIGateway) getDeviceSessions(c *gin.Context) { gw.sendKafkaRequest(c, "GetDeviceSessions") }

func (gw *APIGateway) Start(ctx context.Context) error {
	go func() {
		log.Printf("API Gateway started on http://localhost:%d", gw.config.ServicePort)
		gw.server.ListenAndServe()
	}()
	<-ctx.Done()
	return gw.Stop()
}

func (gw *APIGateway) Stop() error {
	log.Printf("Shutting down API Gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if gw.kafkaClient != nil {
		gw.kafkaClient.Close()
	}
	gw.server.Shutdown(ctx)
	log.Println("API Gateway stopped")
	return nil
}

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.Parse()

	if showVersion {
		fmt.Printf("OpenUSP API Gateway - Version: %s\n", version.Version)
		return
	}

	gateway, err := NewAPIGateway()
	if err != nil {
		log.Fatalf("Failed to create API Gateway: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if err := gateway.Start(ctx); err != nil {
		log.Fatalf("API Gateway error: %v", err)
	}
}
