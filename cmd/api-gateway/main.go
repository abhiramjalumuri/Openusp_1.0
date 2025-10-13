// Package main implements the OpenUSP API Gateway service
//
//	@title			OpenUSP API Gateway
//	@version		1.0.0
//	@description	REST API Gateway for OpenUSP TR-369 User Service Platform
//	@description	Provides unified REST API access to all OpenUSP microservices
//	@description	including device management, parameters, alerts, and sessions
//
//	@contact.name	OpenUSP Support
//	@contact.url	https://github.com/plume-design-inc/openusp
//	@contact.email	support@openusp.org
//
//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@BasePath	/api/v1
//
//	@externalDocs.description	OpenUSP Documentation
//	@externalDocs.url			https://github.com/plume-design-inc/openusp/docs
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
	"strings"
	"syscall"
	"time"

	_ "openusp/api" // Import generated docs
	grpcclient "openusp/internal/grpc"
	"openusp/pkg/config"
	"openusp/pkg/metrics"
	pb "openusp/pkg/proto/dataservice"
	"openusp/pkg/version"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// APIGateway represents the REST API Gateway service
type APIGateway struct {
	config     *config.DeploymentConfig
	fullConfig *config.Config
	router     *gin.Engine
	server     *http.Server
	dataClient *grpcclient.DataServiceClient
	metrics    *metrics.OpenUSPMetrics
}

// NewAPIGateway creates a new API Gateway instance
func NewAPIGateway() (*APIGateway, error) {
	// Load full configuration including security settings
	fullConfig := config.Load()

	// Use fullConfig for all deployment settings
	servicePort, _ := strconv.Atoi(fullConfig.APIGatewayPort)
	log.Println("Configuration: ports sourced exclusively from configs/openusp.yml (env overrides disabled)")
	deploymentConfig := &config.DeploymentConfig{
		ServicePort: servicePort,
		ServiceName: "openusp-api-gateway",
		ServiceType: "api-gateway",
		// Add other fields as needed from fullConfig
	}

	gateway := &APIGateway{
		config:     deploymentConfig,
		fullConfig: fullConfig,
		metrics:    metrics.NewOpenUSPMetrics("api-gateway"),
	}

	// Fixed ports – no service discovery needed
	log.Printf("🎯 Service starting: %s at localhost:%d (fixed port)", gateway.config.ServiceName, gateway.config.ServicePort)

	// Get data service address from static configuration
	dataServiceAddr, err := gateway.getDataServiceAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get data service address: %w", err)
	}

	// Create gRPC client for data service
	dataClient, err := grpcclient.NewDataServiceClient(dataServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create data service client: %w", err)
	}
	gateway.dataClient = dataClient

	// Setup routes
	gateway.setupRoutes()

	return gateway, nil
}

func (gw *APIGateway) getDataServiceAddress() (string, error) {
	// Use unified YAML configuration only
	portStr := gw.fullConfig.DataServiceGRPCPort
	return fmt.Sprintf("localhost:%s", portStr), nil
}

func (gw *APIGateway) getHTTPPort() int {
	// API Gateway always uses static configured port for external client access
	// Unlike internal services which can use dynamic ports, API Gateway needs predictable endpoint
	return gw.config.ServicePort
}

// setupRoutes configures all HTTP routes
func (gw *APIGateway) setupRoutes() {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create router with middleware
	gw.router = gin.New()
	gw.router.Use(gin.Logger())
	gw.router.Use(gin.Recovery())
	gw.router.Use(gw.corsMiddleware())
	gw.router.Use(gw.metricsMiddleware())

	// Health and status endpoints
	gw.router.GET("/health", gw.healthCheck)
	gw.router.GET("/status", gw.getStatus)

	// Metrics endpoint - support both GET and HEAD for Prometheus
	metricsHandler := gin.WrapH(metrics.HTTPHandler())
	gw.router.GET("/metrics", metricsHandler)
	gw.router.HEAD("/metrics", metricsHandler)

	// Swagger UI endpoint - no host specification for dynamic association
	gw.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
		ginSwagger.DeepLinking(true),
		ginSwagger.DocExpansion("none"),
		ginSwagger.DefaultModelsExpandDepth(1),
	))

	// API v1 routes
	v1 := gw.router.Group("/api/v1")
	{
		// Health and status endpoints (for Swagger compatibility)
		v1.GET("/health", gw.healthCheck)
		v1.GET("/status", gw.getStatus)

		// Device management endpoints (TR-181 compliant)
		devices := v1.Group("/devices")
		{
			// Basic device lifecycle management
			devices.GET("", gw.listDevices)
			devices.POST("", gw.provisionDevice)
			devices.GET("/:device_id", gw.getDevice)
			devices.PUT("/:device_id", gw.updateDevice)
			devices.DELETE("/:device_id", gw.deleteDevice)

			// TR-181 Parameter operations
			devices.GET("/:device_id/parameters", gw.getParameters)
			devices.PUT("/:device_id/parameters", gw.setParameter)
			devices.POST("/:device_id/parameters/get", gw.bulkGetParameters)
			devices.POST("/:device_id/parameters/set", gw.bulkSetParameters)
			devices.GET("/:device_id/parameters/search", gw.searchParameters)

			// TR-181 Object operations
			devices.GET("/:device_id/objects", gw.getObjects)
			devices.GET("/:device_id/objects/:object_path", gw.getObjectByPath)
			devices.POST("/:device_id/objects/:object_path/instances", gw.createInstance)
			devices.PUT("/:device_id/objects/:object_path/instances/:instance_id", gw.updateInstance)
			devices.DELETE("/:device_id/objects/:object_path/instances/:instance_id", gw.deleteInstance)

			// TR-181 Command operations
			devices.POST("/:device_id/commands/execute", gw.executeCommand)
			devices.GET("/:device_id/objects/:object_path/commands", gw.getObjectCommands)

			// TR-181 Subscription operations
			devices.POST("/:device_id/subscriptions", gw.createSubscription)
			devices.GET("/:device_id/subscriptions", gw.getSubscriptions)
			devices.DELETE("/:device_id/subscriptions/:subscription_id", gw.deleteSubscription)

			// TR-181 Data model operations
			devices.GET("/:device_id/datamodel", gw.getDataModel)
			devices.GET("/:device_id/schema", gw.getSchema)
			devices.GET("/:device_id/tree", gw.getObjectTree)

			// Legacy endpoints (backward compatibility)
			devices.GET("/:device_id/alerts", gw.getDeviceAlerts)
			devices.GET("/:device_id/sessions", gw.getDeviceSessions)
		}

		// Alert management endpoints
		alerts := v1.Group("/alerts")
		{
			alerts.GET("", gw.listAlerts)
			alerts.POST("", gw.createAlert)
			alerts.PUT("/:id/resolve", gw.resolveAlert)
		}

		// Session management endpoints
		sessions := v1.Group("/sessions")
		{
			sessions.POST("", gw.createSession)
			sessions.PUT("/:session_id/activity", gw.updateSessionActivity)
			sessions.PUT("/:session_id/close", gw.closeSession)
		}
	}
}

// CORS middleware
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

// Metrics middleware
func (gw *APIGateway) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())

		gw.metrics.RecordHTTPRequest(
			"api-gateway",
			c.Request.Method,
			c.FullPath(),
			status,
			duration,
		)
	}
}

// Start starts the HTTP server
func (gw *APIGateway) Start() error {
	// Start HTTP server only
	gw.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", gw.getHTTPPort()),
		Handler: gw.router,
	}
	log.Printf("� API Gateway starting HTTP server on port %d", gw.getHTTPPort())
	log.Printf("   └── HTTP server mode (no HTTPS/TLS)")
	return gw.server.ListenAndServe()
}

// Stop gracefully stops the server
func (gw *APIGateway) Stop() error {
	log.Printf("🛑 Shutting down API Gateway...")

	// Fixed ports – no service deregistration needed

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Close gRPC client connection
	if gw.dataClient != nil {
		if err := gw.dataClient.Close(); err != nil {
			log.Printf("Error closing data service client: %v", err)
		}
	}

	// Shutdown HTTP server
	if gw.server != nil {
		return gw.server.Shutdown(ctx)
	}
	return nil
}

// healthCheck handles health check requests
//
//	@Summary		Health Check
//	@Description	Get the health status of the API Gateway and connected services
//	@Tags			Health
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}	"Service is healthy"
//	@Failure		503	{object}	map[string]interface{}	"Service is unhealthy"
//	@Router			/health [get]
func (gw *APIGateway) healthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check data service health
	healthResp, err := gw.dataClient.HealthCheck(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":       "unhealthy",
			"service":      "API Gateway",
			"version":      version.GetShortVersion(),
			"data_service": "unavailable",
			"error":        err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "healthy",
		"service":      "API Gateway",
		"version":      version.GetShortVersion(),
		"data_service": healthResp,
	})
}

// Status endpoint
func (gw *APIGateway) getStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get data service status
	statusResp, err := gw.dataClient.GetStatus(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":       "error",
			"service":      "API Gateway",
			"version":      version.GetShortVersion(),
			"data_service": "unavailable",
			"error":        err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service":      "API Gateway",
		"version":      version.GetShortVersion(),
		"data_service": statusResp,
		"timestamp":    time.Now().Unix(),
	})
}

// Device management handlers

// listDevices handles device listing with pagination
//
//	@Summary		List Devices
//	@Description	Get a paginated list of all registered devices
//	@Tags			Devices
//	@Accept			json
//	@Produce		json
//	@Param			offset	query		int	false	"Pagination offset"	default(0)
//	@Param			limit	query		int	false	"Pagination limit"	default(10)
//	@Success		200		{object}	map[string]interface{}	"List of devices"
//	@Failure		500		{object}	map[string]interface{}	"Internal server error"
//	@Router			/devices [get]
func (gw *APIGateway) listDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Call data service
	resp, err := gw.dataClient.ListDevices(ctx, int32(offset), int32(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch devices",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": resp.Devices,
		"total":   resp.Total,
		"offset":  offset,
		"limit":   limit,
	})
}

// provisionDevice handles device provisioning
//
//	@Summary		Provision Device
//	@Description	Provision and register a new device in the OpenUSP platform
//	@Description	This endpoint handles the initial device registration process
//	@Description	including endpoint ID assignment, certificate management, and
//	@Description	initial configuration according to TR-181/TR-369 specifications
//	@Tags			Devices
//	@Accept			json
//	@Produce		json
//	@Param			device	body		map[string]interface{}	true	"Device provisioning information (endpoint_id, serial_number, product_class, etc.)"
//	@Success		201		{object}	map[string]interface{}	"Device provisioned successfully"
//	@Failure		400		{object}	map[string]interface{}	"Invalid provisioning request"
//	@Failure		409		{object}	map[string]interface{}	"Device already provisioned"
//	@Failure		500		{object}	map[string]interface{}	"Device provisioning failed"
//	@Router			/devices [post]
func (gw *APIGateway) provisionDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var device pb.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid provisioning request",
			"details": err.Error(),
		})
		return
	}

	// Set timestamps for device provisioning
	now := timestamppb.Now()
	device.CreatedAt = now
	device.UpdatedAt = now

	// Call data service to provision the device
	resp, err := gw.dataClient.CreateDevice(ctx, &device)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Device provisioning failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Device provisioned successfully",
		"device":  resp.Device,
	})
}

// Get device by ID
func (gw *APIGateway) getDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try both parameter names for backward compatibility
	idStr := c.Param("device_id")
	if idStr == "" {
		idStr = c.Param("id")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	// Call data service
	resp, err := gw.dataClient.GetDevice(ctx, uint32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Device not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp.Device)
}

// Update device
func (gw *APIGateway) updateDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try both parameter names for backward compatibility
	idStr := c.Param("device_id")
	if idStr == "" {
		idStr = c.Param("id")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	var device pb.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set ID and update timestamp
	device.Id = uint32(id)
	device.UpdatedAt = timestamppb.Now()

	// Call data service
	resp, err := gw.dataClient.UpdateDevice(ctx, &device)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update device",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp.Device)
}

// Delete device (deprovision)
// @Summary Deprovision Device
// @Description Deprovision and remove a device from the OpenUSP platform
// @Description This endpoint handles complete device deprovisioning including:
// @Description - Deletion of all device parameters and data model entries
// @Description - Removal of all device alerts and notifications
// @Description - Termination of all active sessions
// @Description - Certificate revocation and configuration cleanup
// @Description - Complete removal from the platform database
// @Tags Devices
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} map[string]interface{} "Device deprovisioned successfully"
// @Failure 400 {object} map[string]interface{} "Invalid device ID"
// @Failure 404 {object} map[string]interface{} "Device not found"
// @Failure 500 {object} map[string]interface{} "Device deprovisioning failed"
// @Router /devices/{device_id} [delete]
func (gw *APIGateway) deleteDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try both parameter names for backward compatibility
	idStr := c.Param("device_id")
	if idStr == "" {
		idStr = c.Param("id")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	// Call data service
	_, err = gw.dataClient.DeleteDevice(ctx, uint32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to deprovision device",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Device deprovisioned successfully",
	})
}

// Get device alerts
func (gw *APIGateway) getDeviceAlerts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try both parameter names for backward compatibility
	idStr := c.Param("device_id")
	if idStr == "" {
		idStr = c.Param("id")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	// Parse resolved filter
	var resolved *bool
	if r := c.Query("resolved"); r != "" {
		if b, err := strconv.ParseBool(r); err == nil {
			resolved = &b
		}
	}

	// Call data service
	resp, err := gw.dataClient.GetDeviceAlerts(ctx, uint32(id), resolved)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch device alerts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": resp.Alerts,
	})
}

// Get device sessions
func (gw *APIGateway) getDeviceSessions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try both parameter names for backward compatibility
	idStr := c.Param("device_id")
	if idStr == "" {
		idStr = c.Param("id")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	// Call data service
	resp, err := gw.dataClient.GetDeviceSessions(ctx, uint32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch device sessions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": resp.Sessions,
	})
}

// Alert management handlers

// List alerts
func (gw *APIGateway) listAlerts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Call data service
	resp, err := gw.dataClient.ListAlerts(ctx, int32(offset), int32(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch alerts",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": resp.Alerts,
		"offset": offset,
		"limit":  limit,
	})
}

// Create alert
func (gw *APIGateway) createAlert(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var alert pb.Alert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set timestamps
	now := timestamppb.Now()
	alert.CreatedAt = now
	alert.UpdatedAt = now

	// Call data service
	resp, err := gw.dataClient.CreateAlert(ctx, &alert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create alert",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, resp.Alert)
}

// Resolve alert
func (gw *APIGateway) resolveAlert(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid alert ID",
		})
		return
	}

	// Call data service
	_, err = gw.dataClient.ResolveAlert(ctx, uint32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to resolve alert",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Alert resolved successfully",
	})
}

// Session management handlers

// Create session
func (gw *APIGateway) createSession(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var session pb.Session
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set timestamps
	now := timestamppb.Now()
	session.StartedAt = now
	session.LastActivity = now
	session.CreatedAt = now
	session.UpdatedAt = now

	// Call data service
	resp, err := gw.dataClient.CreateSession(ctx, &session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create session",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, resp.Session)
}

// Update session activity
func (gw *APIGateway) updateSessionActivity(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	// Call data service
	_, err := gw.dataClient.UpdateSessionActivity(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update session activity",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session activity updated successfully",
	})
}

// Close session
func (gw *APIGateway) closeSession(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	// Call data service
	_, err := gw.dataClient.CloseSession(ctx, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to close session",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session closed successfully",
	})
}

// TR-181 Parameter Operations Handlers

// @Summary Get Device Parameters (TR-181)
// @Description Get all parameters for a specific device, optionally filtered by path pattern
// @Tags TR-181 Parameters
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param path query string false "Parameter path pattern (e.g., Device.DeviceInfo.*)"
// @Param partial_path query bool false "Enable partial path matching"
// @Success 200 {object} map[string]interface{} "Parameters retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid device ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /devices/{device_id}/parameters [get]
func (gw *APIGateway) getParameters(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	// Get optional filters from query parameters
	pathPattern := c.DefaultQuery("path", "*")
	partialPath, _ := strconv.ParseBool(c.DefaultQuery("partial_path", "false"))

	// Call data service
	resp, err := gw.dataClient.GetParametersByPath(ctx, uint32(deviceID), pathPattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch device parameters",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":    deviceID,
		"path_pattern": pathPattern,
		"partial_path": partialPath,
		"parameters":   resp.Parameters,
		"count":        len(resp.Parameters),
	})
}

// @Summary Set Single Parameter (TR-181)
// @Description Set a single parameter value for a device
// @Tags TR-181 Parameters
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body map[string]interface{} true "Parameter set request with path and value"
// @Success 200 {object} map[string]interface{} "Parameter set successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /devices/{device_id}/parameters [put]
func (gw *APIGateway) setParameter(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	var request struct {
		Path  string      `json:"path" binding:"required"`
		Value interface{} `json:"value" binding:"required"`
		Type  string      `json:"type,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Convert value to string for now (in a real implementation, you'd handle types properly)
	valueStr := fmt.Sprintf("%v", request.Value)

	// Create parameter object
	parameter := &pb.Parameter{
		DeviceId:    uint32(deviceID),
		Path:        request.Path,
		Value:       valueStr,
		Type:        request.Type,
		LastUpdated: timestamppb.Now(),
		UpdatedAt:   timestamppb.Now(),
	}

	// Call data service to update parameter
	resp, err := gw.dataClient.CreateParameter(ctx, parameter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to set parameter",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Parameter set successfully",
		"parameter": resp.Parameter,
	})
}

// @Summary Bulk Get Parameters (TR-181)
// @Description Get multiple parameters by paths in a single request
// @Tags TR-181 Parameters
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body map[string]interface{} true "Parameter get request with paths array"
// @Success 200 {object} map[string]interface{} "Parameters retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /devices/{device_id}/parameters/get [post]
func (gw *APIGateway) bulkGetParameters(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	var request struct {
		Paths []string `json:"paths" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Get all parameters for the device and filter by paths
	allParams, err := gw.dataClient.GetParametersByPath(ctx, uint32(deviceID), "*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch parameters",
			"details": err.Error(),
		})
		return
	}

	// Filter parameters by requested paths
	var filteredParams []*pb.Parameter
	for _, param := range allParams.Parameters {
		for _, requestedPath := range request.Paths {
			if param.Path == requestedPath {
				filteredParams = append(filteredParams, param)
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":  deviceID,
		"paths":      request.Paths,
		"parameters": filteredParams,
		"count":      len(filteredParams),
	})
}

// @Summary Bulk Set Parameters (TR-181)
// @Description Set multiple parameters in a single request
// @Tags TR-181 Parameters
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body map[string]interface{} true "Parameter set request with parameters array"
// @Success 200 {object} map[string]interface{} "Parameters set successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /devices/{device_id}/parameters/set [post]
func (gw *APIGateway) bulkSetParameters(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	var request struct {
		Parameters []struct {
			Path  string      `json:"path" binding:"required"`
			Value interface{} `json:"value" binding:"required"`
			Type  string      `json:"type,omitempty"`
		} `json:"parameters" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	var results []map[string]interface{}
	var successCount, errorCount int

	// Process each parameter
	for _, paramReq := range request.Parameters {
		valueStr := fmt.Sprintf("%v", paramReq.Value)
		parameter := &pb.Parameter{
			DeviceId:    uint32(deviceID),
			Path:        paramReq.Path,
			Value:       valueStr,
			Type:        paramReq.Type,
			LastUpdated: timestamppb.Now(),
			UpdatedAt:   timestamppb.Now(),
		}

		resp, err := gw.dataClient.CreateParameter(ctx, parameter)
		if err != nil {
			results = append(results, map[string]interface{}{
				"path":    paramReq.Path,
				"success": false,
				"error":   err.Error(),
			})
			errorCount++
		} else {
			results = append(results, map[string]interface{}{
				"path":      paramReq.Path,
				"success":   true,
				"parameter": resp.Parameter,
			})
			successCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":     deviceID,
		"total":         len(request.Parameters),
		"success_count": successCount,
		"error_count":   errorCount,
		"results":       results,
	})
}

// @Summary Search Parameters (TR-181)
// @Description Search parameters using advanced filters and patterns
// @Tags TR-181 Parameters
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param path query string false "Path pattern to search"
// @Param value query string false "Value pattern to search"
// @Param type query string false "Parameter type filter"
// @Success 200 {object} map[string]interface{} "Parameters found"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /devices/{device_id}/parameters/search [get]
func (gw *APIGateway) searchParameters(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	// Get search filters
	pathPattern := c.Query("path")
	valuePattern := c.Query("value")
	typeFilter := c.Query("type")

	// Get all parameters for the device
	allParams, err := gw.dataClient.GetParametersByPath(ctx, uint32(deviceID), "*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch parameters",
			"details": err.Error(),
		})
		return
	}

	// Apply filters
	var filteredParams []*pb.Parameter
	for _, param := range allParams.Parameters {
		match := true

		// Apply path filter
		if pathPattern != "" && !strings.Contains(param.Path, pathPattern) {
			match = false
		}

		// Apply value filter
		if valuePattern != "" && !strings.Contains(param.Value, valuePattern) {
			match = false
		}

		// Apply type filter
		if typeFilter != "" && param.Type != typeFilter {
			match = false
		}

		if match {
			filteredParams = append(filteredParams, param)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":     deviceID,
		"path_pattern":  pathPattern,
		"value_pattern": valuePattern,
		"type_filter":   typeFilter,
		"parameters":    filteredParams,
		"count":         len(filteredParams),
	})
}

// TR-181 Object Operations Handlers

// @Summary Get Device Objects (TR-181)
// @Description Get all objects/object instances for a device
// @Tags TR-181 Objects
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param path query string false "Object path filter"
// @Success 200 {object} map[string]interface{} "Objects retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid device ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /devices/{device_id}/objects [get]
func (gw *APIGateway) getObjects(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	pathFilter := c.Query("path")

	// For now, return a mock response - in a real implementation, you'd have object management
	objects := []map[string]interface{}{
		{
			"path":           "Device.DeviceInfo.",
			"object_type":    "single",
			"access":         "readOnly",
			"parameters":     []string{"Manufacturer", "ModelName", "SerialNumber"},
			"commands":       []string{},
			"instance_count": 1,
		},
		{
			"path":           "Device.WiFi.Radio.",
			"object_type":    "multi",
			"access":         "readWrite",
			"parameters":     []string{"Enable", "Status", "Channel"},
			"commands":       []string{"Reset", "Restart"},
			"instance_count": 2,
		},
	}

	// Apply path filter if provided
	if pathFilter != "" {
		var filteredObjects []map[string]interface{}
		for _, obj := range objects {
			if strings.Contains(obj["path"].(string), pathFilter) {
				filteredObjects = append(filteredObjects, obj)
			}
		}
		objects = filteredObjects
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":   deviceID,
		"path_filter": pathFilter,
		"objects":     objects,
		"count":       len(objects),
	})
}

// @Summary Get Object by Path (TR-181)
// @Description Get specific object details by path
// @Tags TR-181 Objects
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param path path string true "Object path"
// @Success 200 {object} map[string]interface{} "Object details retrieved"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Object not found"
// @Router /devices/{device_id}/objects/{path} [get]
func (gw *APIGateway) getObjectByPath(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	objectPath := c.Param("object_path")
	if objectPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Object path is required",
		})
		return
	}

	// Mock object details - in real implementation, fetch from data service
	objectDetails := map[string]interface{}{
		"device_id":      deviceID,
		"path":           objectPath,
		"object_type":    "multi",
		"access":         "readWrite",
		"description":    "WiFi Radio configuration object",
		"parameters":     []string{"Enable", "Status", "Channel", "PowerLevel"},
		"commands":       []string{"Reset", "Restart", "ScanChannels"},
		"instance_count": 2,
		"instances": []map[string]interface{}{
			{"instance_id": "1", "alias": "Radio_2.4GHz", "status": "Up"},
			{"instance_id": "2", "alias": "Radio_5GHz", "status": "Up"},
		},
	}

	c.JSON(http.StatusOK, objectDetails)
}

// @Summary Create Object Instance (TR-181)
// @Description Create a new instance of a multi-instance object
// @Tags TR-181 Objects
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param path path string true "Object path"
// @Param request body map[string]interface{} true "Instance creation request"
// @Success 201 {object} map[string]interface{} "Instance created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /devices/{device_id}/objects/{path}/instances [post]
func (gw *APIGateway) createInstance(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	objectPath := c.Param("object_path")

	var request struct {
		Alias      string                 `json:"alias,omitempty"`
		Parameters map[string]interface{} `json:"parameters,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Mock instance creation - in real implementation, call data service
	newInstance := map[string]interface{}{
		"device_id":   deviceID,
		"object_path": objectPath,
		"instance_id": "3",
		"alias":       request.Alias,
		"status":      "Created",
		"parameters":  request.Parameters,
		"created_at":  time.Now().Unix(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Object instance created successfully",
		"instance": newInstance,
	})
}

// @Summary Update Object Instance (TR-181)
// @Description Update an existing object instance
// @Tags TR-181 Objects
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param path path string true "Object path"
// @Param instance_id path string true "Instance ID"
// @Param request body map[string]interface{} true "Instance update request"
// @Success 200 {object} map[string]interface{} "Instance updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Instance not found"
// @Router /devices/{device_id}/objects/{path}/instances/{instance_id} [put]
func (gw *APIGateway) updateInstance(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	objectPath := c.Param("object_path")
	instanceID := c.Param("instance_id")

	var request struct {
		Alias      string                 `json:"alias,omitempty"`
		Parameters map[string]interface{} `json:"parameters,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Mock instance update
	updatedInstance := map[string]interface{}{
		"device_id":   deviceID,
		"object_path": objectPath,
		"instance_id": instanceID,
		"alias":       request.Alias,
		"status":      "Updated",
		"parameters":  request.Parameters,
		"updated_at":  time.Now().Unix(),
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Object instance updated successfully",
		"instance": updatedInstance,
	})
}

// @Summary Delete Object Instance (TR-181)
// @Description Delete an object instance
// @Tags TR-181 Objects
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param path path string true "Object path"
// @Param instance_id path string true "Instance ID"
// @Success 200 {object} map[string]interface{} "Instance deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Instance not found"
// @Router /devices/{device_id}/objects/{path}/instances/{instance_id} [delete]
func (gw *APIGateway) deleteInstance(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	objectPath := c.Param("object_path")
	instanceID := c.Param("instance_id")

	// Mock instance deletion
	c.JSON(http.StatusOK, gin.H{
		"message":     "Object instance deleted successfully",
		"device_id":   deviceID,
		"object_path": objectPath,
		"instance_id": instanceID,
		"deleted_at":  time.Now().Unix(),
	})
}

// TR-181 Command Operations Handlers

// @Summary Execute Command (TR-181)
// @Description Execute a command on a device or object
// @Tags TR-181 Commands
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body map[string]interface{} true "Command execution request"
// @Success 200 {object} map[string]interface{} "Command executed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Command execution failed"
// @Router /devices/{device_id}/commands/execute [post]
func (gw *APIGateway) executeCommand(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	var request struct {
		Command    string                 `json:"command" binding:"required"`
		ObjectPath string                 `json:"object_path,omitempty"`
		Parameters map[string]interface{} `json:"parameters,omitempty"`
		Async      bool                   `json:"async,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Mock command execution
	executionID := fmt.Sprintf("exec_%d_%d", deviceID, time.Now().Unix())

	result := map[string]interface{}{
		"execution_id": executionID,
		"device_id":    deviceID,
		"command":      request.Command,
		"object_path":  request.ObjectPath,
		"status":       "completed",
		"result":       "Command executed successfully",
		"executed_at":  time.Now().Unix(),
	}

	if request.Async {
		result["status"] = "pending"
		result["result"] = "Command execution started asynchronously"
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Get Object Commands (TR-181)
// @Description Get available commands for a specific object
// @Tags TR-181 Commands
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param path path string true "Object path"
// @Success 200 {object} map[string]interface{} "Commands retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Router /devices/{device_id}/objects/{path}/commands [get]
func (gw *APIGateway) getObjectCommands(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	objectPath := c.Param("object_path")

	// Mock command list based on object path
	commands := []map[string]interface{}{
		{
			"name":        "Reset",
			"description": "Reset the object to default values",
			"parameters":  []string{},
			"async":       false,
		},
		{
			"name":        "Restart",
			"description": "Restart the object/service",
			"parameters":  []string{},
			"async":       true,
		},
	}

	// Add object-specific commands
	if strings.Contains(objectPath, "WiFi.Radio") {
		commands = append(commands, map[string]interface{}{
			"name":        "ScanChannels",
			"description": "Scan for available WiFi channels",
			"parameters":  []string{"duration", "band"},
			"async":       true,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":   deviceID,
		"object_path": objectPath,
		"commands":    commands,
		"count":       len(commands),
	})
}

// TR-181 Subscription Operations Handlers

// @Summary Create Subscription (TR-181)
// @Description Create a notification subscription for parameter changes
// @Tags TR-181 Subscriptions
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param request body map[string]interface{} true "Subscription request"
// @Success 201 {object} map[string]interface{} "Subscription created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /devices/{device_id}/subscriptions [post]
func (gw *APIGateway) createSubscription(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	var request struct {
		Paths      []string `json:"paths" binding:"required"`
		NotifyType string   `json:"notify_type" binding:"required"`
		Recipient  string   `json:"recipient" binding:"required"`
		Enabled    bool     `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Mock subscription creation
	subscriptionID := fmt.Sprintf("sub_%d_%d", deviceID, time.Now().Unix())

	subscription := map[string]interface{}{
		"subscription_id": subscriptionID,
		"device_id":       deviceID,
		"paths":           request.Paths,
		"notify_type":     request.NotifyType,
		"recipient":       request.Recipient,
		"enabled":         request.Enabled,
		"created_at":      time.Now().Unix(),
		"status":          "active",
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Subscription created successfully",
		"subscription": subscription,
	})
}

// @Summary Get Subscriptions (TR-181)
// @Description Get all subscriptions for a device
// @Tags TR-181 Subscriptions
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} map[string]interface{} "Subscriptions retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid device ID"
// @Router /devices/{device_id}/subscriptions [get]
func (gw *APIGateway) getSubscriptions(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	// Mock subscriptions list
	subscriptions := []map[string]interface{}{
		{
			"subscription_id": "sub_1_12345",
			"device_id":       deviceID,
			"paths":           []string{"Device.WiFi.Radio.*.Status"},
			"notify_type":     "ValueChange",
			"recipient":       "controller@example.com",
			"enabled":         true,
			"created_at":      time.Now().Unix() - 86400,
			"status":          "active",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":     deviceID,
		"subscriptions": subscriptions,
		"count":         len(subscriptions),
	})
}

// @Summary Delete Subscription (TR-181)
// @Description Delete a notification subscription
// @Tags TR-181 Subscriptions
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param subscription_id path string true "Subscription ID"
// @Success 200 {object} map[string]interface{} "Subscription deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Subscription not found"
// @Router /devices/{device_id}/subscriptions/{subscription_id} [delete]
func (gw *APIGateway) deleteSubscription(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	subscriptionID := c.Param("subscription_id")

	c.JSON(http.StatusOK, gin.H{
		"message":         "Subscription deleted successfully",
		"device_id":       deviceID,
		"subscription_id": subscriptionID,
		"deleted_at":      time.Now().Unix(),
	})
}

// TR-181 Data Model Operations Handlers

// @Summary Get Data Model (TR-181)
// @Description Get the complete data model schema for a device
// @Tags TR-181 Data Model
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} map[string]interface{} "Data model retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid device ID"
// @Router /devices/{device_id}/datamodel [get]
func (gw *APIGateway) getDataModel(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	// Mock data model - in real implementation, this would be device-specific
	dataModel := map[string]interface{}{
		"device_id": deviceID,
		"version":   "1.0",
		"schema":    "TR-181 Issue 2 Amendment 15",
		"objects": []map[string]interface{}{
			{
				"path":        "Device.",
				"type":        "object",
				"access":      "readOnly",
				"description": "Top-level object for a Device",
			},
			{
				"path":        "Device.DeviceInfo.",
				"type":        "object",
				"access":      "readOnly",
				"description": "Device information",
			},
			{
				"path":        "Device.WiFi.",
				"type":        "object",
				"access":      "readWrite",
				"description": "WiFi configuration",
			},
		},
		"generated_at": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, dataModel)
}

// @Summary Get Schema (TR-181)
// @Description Get the TR-181 schema information supported by the device
// @Tags TR-181 Data Model
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Success 200 {object} map[string]interface{} "Schema information retrieved"
// @Failure 400 {object} map[string]interface{} "Invalid device ID"
// @Router /devices/{device_id}/schema [get]
func (gw *APIGateway) getSchema(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	schema := map[string]interface{}{
		"device_id":    deviceID,
		"schema_name":  "TR-181",
		"version":      "Issue 2 Amendment 15",
		"date":         "September 2020",
		"organization": "Broadband Forum",
		"supported_profiles": []string{
			"Baseline:1",
			"EthernetInterface:1",
			"WiFiDevice:1",
			"Routing:1",
		},
		"extensions": []string{
			"VendorSpecific.OpenUSP",
		},
		"retrieved_at": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, schema)
}

// @Summary Get Object Tree (TR-181)
// @Description Get the hierarchical object tree structure for a device
// @Tags TR-181 Data Model
// @Accept json
// @Produce json
// @Param device_id path string true "Device ID"
// @Param depth query int false "Maximum depth to retrieve" default(3)
// @Success 200 {object} map[string]interface{} "Object tree retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid device ID"
// @Router /devices/{device_id}/tree [get]
func (gw *APIGateway) getObjectTree(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	depth, _ := strconv.Atoi(c.DefaultQuery("depth", "3"))

	// Mock object tree structure
	objectTree := map[string]interface{}{
		"device_id": deviceID,
		"depth":     depth,
		"tree": map[string]interface{}{
			"Device": map[string]interface{}{
				"type":           "object",
				"instance_count": 1,
				"children": map[string]interface{}{
					"DeviceInfo": map[string]interface{}{
						"type":           "object",
						"instance_count": 1,
						"parameters":     []string{"Manufacturer", "ModelName", "SerialNumber"},
					},
					"WiFi": map[string]interface{}{
						"type":           "object",
						"instance_count": 1,
						"children": map[string]interface{}{
							"Radio": map[string]interface{}{
								"type":           "multi_object",
								"instance_count": 2,
								"parameters":     []string{"Enable", "Status", "Channel"},
							},
						},
					},
				},
			},
		},
		"generated_at": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, objectTree)
}

func main() {
	log.Printf("🚀 Starting OpenUSP API Gateway...")

	// Command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.GetFullVersion("OpenUSP API Gateway"))
		return
	}

	if *showHelp {
		fmt.Println("OpenUSP API Gateway - REST API Service with gRPC Backend")
		fmt.Println("========================================================")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  api-gateway [flags]")
		fmt.Println("")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Environment Variables:")
		fmt.Println("  OPENUSP_TLS_ENABLED       - Enable TLS/HTTPS (default: false)")
		fmt.Println("  OPENUSP_TLS_CERT_PATH     - TLS certificate file path (default: certs/server.crt)")
		fmt.Println("  OPENUSP_TLS_KEY_PATH      - TLS private key file path (default: certs/server.key)")
		fmt.Println("")
		fmt.Println("Port Configuration:")
		fmt.Println("  All service ports are now configured in configs/openusp.yml file only.")
		fmt.Println("")
		fmt.Println("TLS Configuration:")
		fmt.Println("  If both certificate and key files exist, HTTPS will be enabled automatically.")
		fmt.Println("  Use OPENUSP_TLS_CERT_PATH and OPENUSP_TLS_KEY_PATH to specify custom paths.")
		return
	}

	// Port is now configured via YAML config, flag is for backward compatibility only

	// Create API Gateway
	gateway, err := NewAPIGateway()
	if err != nil {
		log.Fatalf("Failed to initialize API Gateway: %v", err)
	}

	// Start server in goroutine
	go func() {
		if err := gateway.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start API Gateway: %v", err)
		}
	}()

	httpPort := gateway.getHTTPPort()

	log.Printf("🚀 API Gateway started successfully")
	log.Printf("   └── Protocol: HTTP (port %d)", httpPort)
	log.Printf("   └── Service Discovery: fixed ports")
	log.Printf("   └── Health Check: http://localhost:%d/health", httpPort)
	log.Printf("   └── Status: http://localhost:%d/status", httpPort)
	log.Printf("   └── Swagger UI: http://localhost:%d/swagger/index.html", httpPort)

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("� Received shutdown signal")
	if err := gateway.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Printf("✅ API Gateway stopped successfully")
}
