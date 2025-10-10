package client

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"openusp/pkg/proto/connectionservice"
	"openusp/pkg/proto/dataservice"
	"openusp/pkg/proto/mtpservice"
	"openusp/pkg/proto/uspservice"

	"google.golang.org/grpc"
)

// OpenUSPConnectionClient provides a simple interface for services to get connections
type OpenUSPConnectionClient struct {
	connectionManager connectionservice.ConnectionServiceClient
	managerConn       *grpc.ClientConn

	// Cache for frequently used clients
	clientCache map[string]interface{}
	cacheMu     sync.RWMutex
	cacheExpiry time.Duration
}

// NewOpenUSPConnectionClient creates a new connection client
func NewOpenUSPConnectionClient(cacheExpiry time.Duration) *OpenUSPConnectionClient {
	var connectionManager connectionservice.ConnectionServiceClient
	var managerConn *grpc.ClientConn

	// Static port configuration - connect to connection manager on port 6201
	target := "localhost:6201"
	conn, err := grpc.Dial(target, grpc.WithInsecure(), grpc.WithTimeout(5*time.Second))
	if err == nil {
		connectionManager = connectionservice.NewConnectionServiceClient(conn)
		managerConn = conn
		log.Printf("✅ Connected to Connection Manager at %s", target)
	} else {
		log.Printf("⚠️ Failed to connect to Connection Manager: %v", err)
	}

	return &OpenUSPConnectionClient{
		connectionManager: connectionManager,
		managerConn:       managerConn,
		clientCache:       make(map[string]interface{}),
		cacheExpiry:       cacheExpiry,
	}
}

// GetDataServiceClient returns a data service client via connection manager only
func (c *OpenUSPConnectionClient) GetDataServiceClient() (dataservice.DataServiceClient, error) {
	if c.connectionManager == nil {
		return nil, fmt.Errorf("connection manager not available - required for service discovery")
	}

	// Retry logic with exponential backoff
	maxRetries := 5
	baseDelay := 500 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * baseDelay
			log.Printf("🔄 Retrying data service connection (attempt %d/%d) after %v", attempt+1, maxRetries, delay)
			time.Sleep(delay)
		}

		resp, err := c.connectionManager.GetConnection(context.Background(), &connectionservice.GetConnectionRequest{
			ServiceName: "openusp-data-service",
		})
		if err != nil {
			log.Printf("⚠️ Connection manager call failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("connection manager failed for data service after %d attempts: %w", maxRetries, err)
			}
			continue
		}

		if !resp.Success {
			log.Printf("⚠️ Connection manager returned failure (attempt %d/%d): %s", attempt+1, maxRetries, resp.Error)
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("connection manager could not connect to data service after %d attempts: %s", maxRetries, resp.Error)
			}
			continue
		}

		// Success - try to dial the target
		conn, err := grpc.Dial(resp.Target, grpc.WithInsecure())
		if err != nil {
			log.Printf("⚠️ gRPC dial failed (attempt %d/%d) to %s: %v", attempt+1, maxRetries, resp.Target, err)
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("failed to connect to data service at %s after %d attempts: %w", resp.Target, maxRetries, err)
			}
			continue
		}

		log.Printf("✅ Connected to Data Service via connection manager: %s (attempt %d/%d)", resp.Target, attempt+1, maxRetries)
		return dataservice.NewDataServiceClient(conn), nil
	}

	return nil, fmt.Errorf("exhausted all %d retry attempts for data service connection", maxRetries)
}

// GetMTPServiceClient returns an MTP service client via connection manager only
func (c *OpenUSPConnectionClient) GetMTPServiceClient() (mtpservice.MTPServiceClient, error) {
	if c.connectionManager == nil {
		return nil, fmt.Errorf("connection manager not available - required for service discovery")
	}

	// Retry logic with exponential backoff
	maxRetries := 5
	baseDelay := 500 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * baseDelay
			log.Printf("🔄 Retrying MTP service connection (attempt %d/%d) after %v", attempt+1, maxRetries, delay)
			time.Sleep(delay)
		}

		resp, err := c.connectionManager.GetConnection(context.Background(), &connectionservice.GetConnectionRequest{
			ServiceName: "openusp-mtp-service",
		})
		if err != nil {
			log.Printf("⚠️ Connection manager call failed for MTP service (attempt %d/%d): %v", attempt+1, maxRetries, err)
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("connection manager failed for MTP service after %d attempts: %w", maxRetries, err)
			}
			continue
		}

		if !resp.Success {
			log.Printf("⚠️ Connection manager returned failure for MTP service (attempt %d/%d): %s", attempt+1, maxRetries, resp.Error)
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("connection manager could not connect to MTP service after %d attempts: %s", maxRetries, resp.Error)
			}
			continue
		}

		// Success - try to dial the target
		conn, err := grpc.Dial(resp.Target, grpc.WithInsecure())
		if err != nil {
			log.Printf("⚠️ gRPC dial failed for MTP service (attempt %d/%d) to %s: %v", attempt+1, maxRetries, resp.Target, err)
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("failed to connect to MTP service at %s after %d attempts: %w", resp.Target, maxRetries, err)
			}
			continue
		}

		log.Printf("✅ Connected to MTP Service via connection manager: %s (attempt %d/%d)", resp.Target, attempt+1, maxRetries)
		return mtpservice.NewMTPServiceClient(conn), nil
	}

	return nil, fmt.Errorf("exhausted all %d retry attempts for MTP service connection", maxRetries)
}

// GetUSPServiceClient returns a USP service client via connection manager only
func (c *OpenUSPConnectionClient) GetUSPServiceClient() (uspservice.USPServiceClient, error) {
	if c.connectionManager == nil {
		return nil, fmt.Errorf("connection manager not available - required for service discovery")
	}

	// Retry logic with exponential backoff
	maxRetries := 5
	baseDelay := 500 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * baseDelay
			log.Printf("🔄 Retrying USP service connection (attempt %d/%d) after %v", attempt+1, maxRetries, delay)
			time.Sleep(delay)
		}

		resp, err := c.connectionManager.GetConnection(context.Background(), &connectionservice.GetConnectionRequest{
			ServiceName: "openusp-usp-service",
		})
		if err != nil {
			log.Printf("⚠️ Connection manager call failed for USP service (attempt %d/%d): %v", attempt+1, maxRetries, err)
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("connection manager failed for USP service after %d attempts: %w", maxRetries, err)
			}
			continue
		}

		if !resp.Success {
			log.Printf("⚠️ Connection manager returned failure for USP service (attempt %d/%d): %s", attempt+1, maxRetries, resp.Error)
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("connection manager could not connect to USP service after %d attempts: %s", maxRetries, resp.Error)
			}
			continue
		}

		// Success - try to dial the target
		conn, err := grpc.Dial(resp.Target, grpc.WithInsecure())
		if err != nil {
			log.Printf("⚠️ gRPC dial failed for USP service (attempt %d/%d) to %s: %v", attempt+1, maxRetries, resp.Target, err)
			if attempt == maxRetries-1 {
				return nil, fmt.Errorf("failed to connect to USP service at %s after %d attempts: %w", resp.Target, maxRetries, err)
			}
			continue
		}

		log.Printf("✅ Connected to USP Service via connection manager: %s (attempt %d/%d)", resp.Target, attempt+1, maxRetries)
		return uspservice.NewUSPServiceClient(conn), nil
	}

	return nil, fmt.Errorf("exhausted all %d retry attempts for USP service connection", maxRetries)
}

// RegisterDependency registers a service dependency with the connection manager
func (c *OpenUSPConnectionClient) RegisterDependency(serviceName, dependsOnService string, minConn, maxConn int) error {
	if c.connectionManager == nil {
		log.Printf("⚠️ Connection Manager not available, skipping dependency registration")
		return nil // Don't fail if connection manager is not available
	}

	_, err := c.connectionManager.RegisterServiceDependency(context.Background(), &connectionservice.RegisterDependencyRequest{
		ServiceName:      serviceName,
		DependsOnService: dependsOnService,
		MinConnections:   int32(minConn),
		MaxConnections:   int32(maxConn),
	})

	if err != nil {
		log.Printf("⚠️ Failed to register dependency %s -> %s: %v", serviceName, dependsOnService, err)
	} else {
		log.Printf("✅ Registered dependency: %s -> %s", serviceName, dependsOnService)
	}

	return err
}

// GetConnectionStatus returns the status of all managed connections
func (c *OpenUSPConnectionClient) GetConnectionStatus() (*connectionservice.GetConnectionStatusResponse, error) {
	if c.connectionManager == nil {
		return nil, fmt.Errorf("connection manager not available")
	}

	return c.connectionManager.GetConnectionStatus(context.Background(), &connectionservice.GetConnectionStatusRequest{})
}

// IsConnectionManagerAvailable checks if the connection manager is available
func (c *OpenUSPConnectionClient) IsConnectionManagerAvailable() bool {
	return c.connectionManager != nil
}

// Close closes the connection to the connection manager
func (c *OpenUSPConnectionClient) Close() error {
	if c.managerConn != nil {
		return c.managerConn.Close()
	}
	return nil
}

// Fallback methods for direct service discovery
