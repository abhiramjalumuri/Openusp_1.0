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

	_ "openusp/api" // Import swagger docs
	"openusp/pkg/config"
	"openusp/pkg/kafka"
	"openusp/pkg/metrics"
	"openusp/pkg/version"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	// Swagger documentation
	gw.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

// healthCheck godoc
//
//	@Summary		Health check endpoint
//	@Description	Check if the API Gateway service is healthy
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Router			/health [get]
func (gw *APIGateway) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "API Gateway", "mode": "kafka"})
}

// getStatus godoc
//
//	@Summary		Get service status
//	@Description	Get detailed status information about the API Gateway
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Router			/status [get]
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

// listDevices godoc
//
//	@Summary		List all devices
//	@Description	Get a list of all registered devices
//	@Tags			Devices
//	@Produce		json
//	@Param			offset	query		int		false	"Pagination offset"
//	@Param			limit	query		int		false	"Pagination limit"
//	@Success		200		{array}		map[string]interface{}
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/devices [get]
func (gw *APIGateway) listDevices(c *gin.Context) { gw.sendKafkaRequest(c, "ListDevices") }

// getDevice godoc
//
//	@Summary		Get device by ID
//	@Description	Get detailed information about a specific device
//	@Tags			Devices
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id} [get]
func (gw *APIGateway) getDevice(c *gin.Context) { gw.sendKafkaRequest(c, "GetDevice") }

// provisionDevice godoc
//
//	@Summary		Create a new device
//	@Description	Register a new device in the system
//	@Tags			Devices
//	@Accept			json
//	@Produce		json
//	@Param			device	body		map[string]interface{}	true	"Device information"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/devices [post]
func (gw *APIGateway) provisionDevice(c *gin.Context) { gw.sendKafkaRequest(c, "ProvisionDevice") }

// updateDevice godoc
//
//	@Summary		Update device
//	@Description	Update device information
//	@Tags			Devices
//	@Accept			json
//	@Produce		json
//	@Param			device_id	path		string					true	"Device ID"
//	@Param			device		body		map[string]interface{}	true	"Device information"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id} [put]
func (gw *APIGateway) updateDevice(c *gin.Context) { gw.sendKafkaRequest(c, "UpdateDevice") }

// deleteDevice godoc
//
//	@Summary		Delete device
//	@Description	Remove a device from the system
//	@Tags			Devices
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		204			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id} [delete]
func (gw *APIGateway) deleteDevice(c *gin.Context) { gw.sendKafkaRequest(c, "DeleteDevice") }

// getParameters godoc
//
//	@Summary		Get device parameters
//	@Description	Retrieve all parameters for a device
//	@Tags			Parameters
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{array}		map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/parameters [get]
func (gw *APIGateway) getParameters(c *gin.Context) { gw.sendKafkaRequest(c, "GetParameters") }

// setParameter godoc
//
//	@Summary		Set device parameter
//	@Description	Update a device parameter value
//	@Tags			Parameters
//	@Accept			json
//	@Produce		json
//	@Param			device_id	path		string					true	"Device ID"
//	@Param			parameter	body		map[string]interface{}	true	"Parameter data"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/parameters [put]
func (gw *APIGateway) setParameter(c *gin.Context) { gw.sendKafkaRequest(c, "SetParameter") }

// bulkGetParameters godoc
//
//	@Summary		Bulk get parameters
//	@Description	Get multiple device parameters at once
//	@Tags			Parameters
//	@Accept			json
//	@Produce		json
//	@Param			device_id	path		string					true	"Device ID"
//	@Param			paths		body		map[string]interface{}	true	"Parameter paths"
//	@Success		200			{array}		map[string]interface{}
//	@Failure		400			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/parameters/get [post]
func (gw *APIGateway) bulkGetParameters(c *gin.Context) { gw.sendKafkaRequest(c, "BulkGetParameters") }

// bulkSetParameters godoc
//
//	@Summary		Bulk set parameters
//	@Description	Set multiple device parameters at once
//	@Tags			Parameters
//	@Accept			json
//	@Produce		json
//	@Param			device_id	path		string					true	"Device ID"
//	@Param			parameters	body		map[string]interface{}	true	"Parameters to set"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/parameters/set [post]
func (gw *APIGateway) bulkSetParameters(c *gin.Context) { gw.sendKafkaRequest(c, "BulkSetParameters") }

// searchParameters godoc
//
//	@Summary		Search parameters
//	@Description	Search for device parameters by path
//	@Tags			Parameters
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Param			path		query		string	false	"Parameter path to search"
//	@Success		200			{array}		map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/parameters/search [get]
func (gw *APIGateway) searchParameters(c *gin.Context) { gw.sendKafkaRequest(c, "SearchParameters") }

// getObjects godoc
//
//	@Summary		Get device objects
//	@Description	Retrieve all objects for a device
//	@Tags			Objects
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{array}		map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/objects [get]
func (gw *APIGateway) getObjects(c *gin.Context) { gw.sendKafkaRequest(c, "GetObjects") }

// getObjectByPath godoc
//
//	@Summary		Get object by path
//	@Description	Retrieve a specific object by its path
//	@Tags			Objects
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Param			object_path	path		string	true	"Object path"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/objects/{object_path} [get]
func (gw *APIGateway) getObjectByPath(c *gin.Context) { gw.sendKafkaRequest(c, "GetObjectByPath") }

// executeCommand godoc
//
//	@Summary		Execute command
//	@Description	Execute a command on a device
//	@Tags			Commands
//	@Accept			json
//	@Produce		json
//	@Param			device_id	path		string					true	"Device ID"
//	@Param			command		body		map[string]interface{}	true	"Command data"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/commands/execute [post]
func (gw *APIGateway) executeCommand(c *gin.Context) { gw.sendKafkaRequest(c, "ExecuteCommand") }

// createSubscription godoc
//
//	@Summary		Create subscription
//	@Description	Create a new subscription for device events
//	@Tags			Subscriptions
//	@Accept			json
//	@Produce		json
//	@Param			device_id		path		string					true	"Device ID"
//	@Param			subscription	body		map[string]interface{}	true	"Subscription data"
//	@Success		201				{object}	map[string]interface{}
//	@Failure		400				{object}	map[string]string
//	@Failure		404				{object}	map[string]string
//	@Failure		500				{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/subscriptions [post]
func (gw *APIGateway) createSubscription(c *gin.Context) {
	gw.sendKafkaRequest(c, "CreateSubscription")
}

// getSubscriptions godoc
//
//	@Summary		Get subscriptions
//	@Description	Get all subscriptions for a device
//	@Tags			Subscriptions
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{array}		map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/subscriptions [get]
func (gw *APIGateway) getSubscriptions(c *gin.Context) { gw.sendKafkaRequest(c, "GetSubscriptions") }

// deleteSubscription godoc
//
//	@Summary		Delete subscription
//	@Description	Delete a specific subscription
//	@Tags			Subscriptions
//	@Produce		json
//	@Param			device_id			path		string	true	"Device ID"
//	@Param			subscription_id		path		string	true	"Subscription ID"
//	@Success		204					{object}	map[string]interface{}
//	@Failure		404					{object}	map[string]string
//	@Failure		500					{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/subscriptions/{subscription_id} [delete]
func (gw *APIGateway) deleteSubscription(c *gin.Context) {
	gw.sendKafkaRequest(c, "DeleteSubscription")
}

// getDataModel godoc
//
//	@Summary		Get device data model
//	@Description	Retrieve the complete data model for a device
//	@Tags			DataModel
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/datamodel [get]
func (gw *APIGateway) getDataModel(c *gin.Context) { gw.sendKafkaRequest(c, "GetDataModel") }

// getSchema godoc
//
//	@Summary		Get device schema
//	@Description	Retrieve the schema for a device
//	@Tags			DataModel
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/schema [get]
func (gw *APIGateway) getSchema(c *gin.Context) { gw.sendKafkaRequest(c, "GetSchema") }

// getObjectTree godoc
//
//	@Summary		Get object tree
//	@Description	Retrieve the object tree structure for a device
//	@Tags			DataModel
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{object}	map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/tree [get]
func (gw *APIGateway) getObjectTree(c *gin.Context) { gw.sendKafkaRequest(c, "GetObjectTree") }

// listAlerts godoc
//
//	@Summary		List all alerts
//	@Description	Get a list of all alerts in the system
//	@Tags			Alerts
//	@Produce		json
//	@Param			offset	query		int		false	"Pagination offset"
//	@Param			limit	query		int		false	"Pagination limit"
//	@Success		200		{array}		map[string]interface{}
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/alerts [get]
func (gw *APIGateway) listAlerts(c *gin.Context) { gw.sendKafkaRequest(c, "ListAlerts") }

// createAlert godoc
//
//	@Summary		Create alert
//	@Description	Create a new alert
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Param			alert	body		map[string]interface{}	true	"Alert data"
//	@Success		201		{object}	map[string]interface{}
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/alerts [post]
func (gw *APIGateway) createAlert(c *gin.Context) { gw.sendKafkaRequest(c, "CreateAlert") }

// resolveAlert godoc
//
//	@Summary		Resolve alert
//	@Description	Mark an alert as resolved
//	@Tags			Alerts
//	@Produce		json
//	@Param			id	path		string	true	"Alert ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/api/v1/alerts/{id}/resolve [put]
func (gw *APIGateway) resolveAlert(c *gin.Context) { gw.sendKafkaRequest(c, "ResolveAlert") }

// getDeviceAlerts godoc
//
//	@Summary		Get device alerts
//	@Description	Get all alerts for a specific device
//	@Tags			Alerts
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{array}		map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/alerts [get]
func (gw *APIGateway) getDeviceAlerts(c *gin.Context) { gw.sendKafkaRequest(c, "GetDeviceAlerts") }

// listSessions godoc
//
//	@Summary		List all sessions
//	@Description	Get a list of all sessions
//	@Tags			Sessions
//	@Produce		json
//	@Success		200	{array}		map[string]interface{}
//	@Failure		500	{object}	map[string]string
//	@Router			/api/v1/sessions [get]
func (gw *APIGateway) listSessions(c *gin.Context) { gw.sendKafkaRequest(c, "ListSessions") }

// getSession godoc
//
//	@Summary		Get session by ID
//	@Description	Get detailed information about a specific session
//	@Tags			Sessions
//	@Produce		json
//	@Param			id	path		string	true	"Session ID"
//	@Success		200	{object}	map[string]interface{}
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/api/v1/sessions/{id} [get]
func (gw *APIGateway) getSession(c *gin.Context) { gw.sendKafkaRequest(c, "GetSession") }

// getDeviceSessions godoc
//
//	@Summary		Get device sessions
//	@Description	Get all sessions for a specific device
//	@Tags			Sessions
//	@Produce		json
//	@Param			device_id	path		string	true	"Device ID"
//	@Success		200			{array}		map[string]interface{}
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/api/v1/devices/{device_id}/sessions [get]
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
