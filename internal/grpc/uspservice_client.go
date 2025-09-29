package grpc

import (
	"context"
	"fmt"
	"time"

	"openusp/pkg/proto/uspservice"

	"google.golang.org/grpc"
)

// USPServiceClient wraps the gRPC client for USP service
type USPServiceClient struct {
	client uspservice.USPServiceClient
	conn   *grpc.ClientConn
}

// NewUSPServiceClient creates a new USP service client
func NewUSPServiceClient(address string) (*USPServiceClient, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to USP service: %w", err)
	}

	client := uspservice.NewUSPServiceClient(conn)

	return &USPServiceClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the connection to the USP service
func (c *USPServiceClient) Close() error {
	return c.conn.Close()
}

// ProcessUSPMessage sends a USP message to the USP service for processing
func (c *USPServiceClient) ProcessUSPMessage(ctx context.Context, uspData []byte, transportType, clientID string, metadata map[string]string) (*uspservice.USPMessageResponse, error) {
	req := &uspservice.USPMessageRequest{
		UspData:       uspData,
		TransportType: transportType,
		ClientId:      clientID,
		Timestamp:     time.Now().Unix(),
		Metadata:      metadata,
	}

	return c.client.ProcessUSPMessage(ctx, req)
}

// GetHealth checks the health of the USP service
func (c *USPServiceClient) GetHealth(ctx context.Context) (*uspservice.HealthResponse, error) {
	req := &uspservice.HealthRequest{}
	return c.client.GetServiceHealth(ctx, req)
}

// GetStatus gets the status of the USP service
func (c *USPServiceClient) GetStatus(ctx context.Context) (*uspservice.StatusResponse, error) {
	req := &uspservice.StatusRequest{}
	return c.client.GetServiceStatus(ctx, req)
}
