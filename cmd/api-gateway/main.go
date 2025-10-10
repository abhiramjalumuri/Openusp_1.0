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
//	@host		localhost:6500
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
	"syscall"
	"time"

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
	config      *config.DeploymentConfig
	router      *gin.Engine
	server      *http.Server
	dataClient  *grpcclient.DataServiceClient
	metrics     *metrics.OpenUSPMetrics
}

// NewAPIGateway creates a new API Gateway instance
func NewAPIGateway() (*APIGateway, error) {
	// Load configuration with service-specific port environment variable
	config := config.LoadDeploymentConfigWithPortEnv("openusp-api-gateway", "api-gateway", 6500, "OPENUSP_API_GATEWAY_PORT")

	gateway := &APIGateway{
		config:  config,
		metrics: metrics.NewOpenUSPMetrics("api-gateway"),
	}

	// Static port configuration - no service discovery needed

	// Register with service discovery
	if err := gateway.registerService(); err != nil {
		return nil, fmt.Errorf("failed to register service: %w", err)
	}

	// Get data service address (either from Consul or configuration)
	dataServiceAddr, err := gateway.getDataServiceAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to discover data service: %w", err)
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

func (gw *APIGateway) registerService() error {
	// Static port configuration - just log service startup
	log.Printf("🎯 Service starting: %s at localhost:%d (static port configuration)",
		gw.config.ServiceName, gw.config.ServicePort)
	return nil
}

func (gw *APIGateway) getDataServiceAddress() (string, error) {
	// Static port configuration for data service
	return "localhost:6101", nil // Data service gRPC port from configs/services.yaml
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
	gw.router.GET("/metrics", gin.WrapH(metrics.HTTPHandler()))

	// Swagger UI endpoint - Use relative URL for cross-platform compatibility
	gw.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("./doc.json"),  // Relative URL - works from any host
		ginSwagger.DeepLinking(true),
		ginSwagger.DocExpansion("none"),
	))

	// API v1 routes
	v1 := gw.router.Group("/api/v1")
	{
		// Health and status endpoints (for Swagger compatibility)
		v1.GET("/health", gw.healthCheck)
		v1.GET("/status", gw.getStatus)

		// Device management endpoints
		devices := v1.Group("/devices")
		{
			devices.GET("", gw.listDevices)
			devices.POST("", gw.createDevice)
			devices.GET("/:id", gw.getDevice)
			devices.PUT("/:id", gw.updateDevice)
			devices.DELETE("/:id", gw.deleteDevice)
			devices.GET("/:id/parameters", gw.getDeviceParameters)
			devices.GET("/:id/alerts", gw.getDeviceAlerts)
			devices.GET("/:id/sessions", gw.getDeviceSessions)
		}

		// Parameter management endpoints
		parameters := v1.Group("/parameters")
		{
			parameters.GET("/endpoint/:endpoint_id", gw.getParametersByEndpoint)
			parameters.POST("", gw.createParameter)
			parameters.DELETE("/:device_id/:path", gw.deleteParameter)
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
	gw.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", gw.getHTTPPort()),
		Handler: gw.router,
	}

	log.Printf("🚀 API Gateway starting on port %d", gw.getHTTPPort())
	return gw.server.ListenAndServe()
}

// Stop gracefully stops the server
func (gw *APIGateway) Stop() error {
	log.Printf("🛑 Shutting down API Gateway...")

	// Static port configuration - no service deregistration needed

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

// createDevice handles device creation
//
//	@Summary		Create Device
//	@Description	Register a new device in the system
//	@Tags			Devices
//	@Accept			json
//	@Produce		json
//	@Param			device	body		map[string]interface{}	true	"Device information"
//	@Success		201		{object}	map[string]interface{}	"Device created successfully"
//	@Failure		400		{object}	map[string]interface{}	"Invalid request body"
//	@Failure		500		{object}	map[string]interface{}	"Internal server error"
//	@Router			/devices [post]
func (gw *APIGateway) createDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var device pb.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set timestamps
	now := timestamppb.Now()
	device.CreatedAt = now
	device.UpdatedAt = now

	// Call data service
	resp, err := gw.dataClient.CreateDevice(ctx, &device)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create device",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, resp.Device)
}

// Get device by ID
func (gw *APIGateway) getDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

// Delete device
// @Summary Delete Device
// @Description Delete a device by ID from the OpenUSP platform
// @Tags Devices
// @Accept json
// @Produce json
// @Param id path int true "Device ID"
// @Success 200 {object} map[string]interface{} "Device deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid device ID"
// @Failure 500 {object} map[string]interface{} "Failed to delete device"
// @Router /devices/{id} [delete]
func (gw *APIGateway) deleteDevice(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
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
			"error":   "Failed to delete device",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Device deleted successfully",
	})
}

// Get device parameters
func (gw *APIGateway) getDeviceParameters(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	// Call data service
	resp, err := gw.dataClient.GetDeviceParameters(ctx, uint32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch device parameters",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"parameters": resp.Parameters,
	})
}

// Get device alerts
func (gw *APIGateway) getDeviceAlerts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

// Parameter management handlers

// Create parameter
func (gw *APIGateway) createParameter(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var parameter pb.Parameter
	if err := c.ShouldBindJSON(&parameter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set timestamps
	now := timestamppb.Now()
	parameter.LastUpdated = now
	parameter.CreatedAt = now
	parameter.UpdatedAt = now

	// Call data service
	resp, err := gw.dataClient.CreateParameter(ctx, &parameter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create parameter",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, resp.Parameter)
}

// Delete parameter
func (gw *APIGateway) deleteParameter(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deviceID, err := strconv.ParseUint(c.Param("device_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device ID",
		})
		return
	}

	path := c.Param("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Parameter path is required",
		})
		return
	}

	// Call data service
	_, err = gw.dataClient.DeleteParameter(ctx, uint32(deviceID), path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete parameter",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Parameter deleted successfully",
	})
}

// @Summary Get parameters by endpoint ID
// @Description Get device parameters filtered by endpoint ID with optional path pattern
// @Tags Parameters
// @Accept json
// @Produce json
// @Param endpoint_id path string true "Endpoint ID"
// @Param path_pattern query string false "Path pattern for filtering parameters"
// @Success 200 {object} map[string]interface{} "Parameters retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid endpoint ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /parameters/endpoint/{endpoint_id} [get]
func (gw *APIGateway) getParametersByEndpoint(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	endpointID := c.Param("endpoint_id")
	if endpointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Endpoint ID is required",
		})
		return
	}

	// Get optional path pattern from query parameter
	pathPattern := c.Query("path_pattern")

	// Call data service
	resp, err := gw.dataClient.GetParametersByEndpoint(ctx, endpointID, pathPattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch parameters by endpoint",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device":     resp.Device,
		"parameters": resp.Parameters,
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

func main() {
	log.Printf("🚀 Starting OpenUSP API Gateway...")

	// Command line flags
	var port = flag.Int("port", 6500, "HTTP port")
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
		fmt.Println("  CONSUL_ENABLED           - Enable Consul service discovery (default: true)")
		fmt.Println("  OPENUSP_API_GATEWAY_PORT - Server port (default: 6500)")
		fmt.Println("  DATA_SERVICE_ADDR        - Data service gRPC address (default: localhost:6101)")
		return
	}

	// Static port configuration - no environment overrides needed
	if *port != 6500 {
		os.Setenv("OPENUSP_API_GATEWAY_PORT", fmt.Sprintf("%d", *port))
	}

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
	log.Printf("   └── HTTP Port: %d (static configuration)", httpPort)
	log.Printf("   └── Service Discovery: Static port configuration")
	log.Printf("   └── Health Check: :%d/health", httpPort)
	log.Printf("   └── Status: :%d/status", httpPort)
	log.Printf("   └── Swagger UI: :%d/swagger/index.html", httpPort)

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
