package grpc

import (
	"context"
	"fmt"
	"time"

	pb "openusp/pkg/proto/dataservice"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DataServiceClient wraps the gRPC client for the data service
type DataServiceClient struct {
	conn   *grpc.ClientConn
	client pb.DataServiceClient
}

// NewDataServiceClient creates a new gRPC client for the data service
func NewDataServiceClient(address string) (*DataServiceClient, error) {
	// Create connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to data service at %s: %w", address, err)
	}

	client := pb.NewDataServiceClient(conn)

	return &DataServiceClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close closes the gRPC connection
func (c *DataServiceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Health and Status methods
func (c *DataServiceClient) HealthCheck(ctx context.Context) (*pb.HealthCheckResponse, error) {
	return c.client.HealthCheck(ctx, &pb.HealthCheckRequest{})
}

func (c *DataServiceClient) GetStatus(ctx context.Context) (*pb.GetStatusResponse, error) {
	return c.client.GetStatus(ctx, &pb.GetStatusRequest{})
}

// Device operations
func (c *DataServiceClient) CreateDevice(ctx context.Context, device *pb.Device) (*pb.CreateDeviceResponse, error) {
	return c.client.CreateDevice(ctx, &pb.CreateDeviceRequest{Device: device})
}

func (c *DataServiceClient) GetDevice(ctx context.Context, id uint32) (*pb.GetDeviceResponse, error) {
	return c.client.GetDevice(ctx, &pb.GetDeviceRequest{Id: id})
}

func (c *DataServiceClient) GetDeviceByEndpoint(ctx context.Context, endpointID string) (*pb.GetDeviceByEndpointResponse, error) {
	return c.client.GetDeviceByEndpoint(ctx, &pb.GetDeviceByEndpointRequest{EndpointId: endpointID})
}

func (c *DataServiceClient) ListDevices(ctx context.Context, offset, limit int32) (*pb.ListDevicesResponse, error) {
	return c.client.ListDevices(ctx, &pb.ListDevicesRequest{Offset: offset, Limit: limit})
}

func (c *DataServiceClient) UpdateDevice(ctx context.Context, device *pb.Device) (*pb.UpdateDeviceResponse, error) {
	return c.client.UpdateDevice(ctx, &pb.UpdateDeviceRequest{Device: device})
}

func (c *DataServiceClient) DeleteDevice(ctx context.Context, id uint32) (*pb.DeleteDeviceResponse, error) {
	return c.client.DeleteDevice(ctx, &pb.DeleteDeviceRequest{Id: id})
}

// Parameter operations
func (c *DataServiceClient) CreateParameter(ctx context.Context, parameter *pb.Parameter) (*pb.CreateParameterResponse, error) {
	return c.client.CreateParameter(ctx, &pb.CreateParameterRequest{Parameter: parameter})
}

func (c *DataServiceClient) GetDeviceParameters(ctx context.Context, deviceID uint32) (*pb.GetDeviceParametersResponse, error) {
	return c.client.GetDeviceParameters(ctx, &pb.GetDeviceParametersRequest{DeviceId: deviceID})
}

func (c *DataServiceClient) GetParametersByPath(ctx context.Context, deviceID uint32, pathPattern string) (*pb.GetParametersByPathResponse, error) {
	return c.client.GetParametersByPath(ctx, &pb.GetParametersByPathRequest{
		DeviceId:    deviceID,
		PathPattern: pathPattern,
	})
}

func (c *DataServiceClient) DeleteParameter(ctx context.Context, deviceID uint32, path string) (*pb.DeleteParameterResponse, error) {
	return c.client.DeleteParameter(ctx, &pb.DeleteParameterRequest{
		DeviceId: deviceID,
		Path:     path,
	})
}

// Alert operations
func (c *DataServiceClient) CreateAlert(ctx context.Context, alert *pb.Alert) (*pb.CreateAlertResponse, error) {
	return c.client.CreateAlert(ctx, &pb.CreateAlertRequest{Alert: alert})
}

func (c *DataServiceClient) GetDeviceAlerts(ctx context.Context, deviceID uint32, resolved *bool) (*pb.GetDeviceAlertsResponse, error) {
	req := &pb.GetDeviceAlertsRequest{DeviceId: deviceID}
	if resolved != nil {
		req.Resolved = resolved
	}
	return c.client.GetDeviceAlerts(ctx, req)
}

func (c *DataServiceClient) ListAlerts(ctx context.Context, offset, limit int32) (*pb.ListAlertsResponse, error) {
	return c.client.ListAlerts(ctx, &pb.ListAlertsRequest{Offset: offset, Limit: limit})
}

func (c *DataServiceClient) ResolveAlert(ctx context.Context, id uint32) (*pb.ResolveAlertResponse, error) {
	return c.client.ResolveAlert(ctx, &pb.ResolveAlertRequest{Id: id})
}

// Session operations
func (c *DataServiceClient) CreateSession(ctx context.Context, session *pb.Session) (*pb.CreateSessionResponse, error) {
	return c.client.CreateSession(ctx, &pb.CreateSessionRequest{Session: session})
}

func (c *DataServiceClient) GetSession(ctx context.Context, sessionID string) (*pb.GetSessionResponse, error) {
	return c.client.GetSession(ctx, &pb.GetSessionRequest{SessionId: sessionID})
}

func (c *DataServiceClient) UpdateSessionActivity(ctx context.Context, sessionID string) (*pb.UpdateSessionActivityResponse, error) {
	return c.client.UpdateSessionActivity(ctx, &pb.UpdateSessionActivityRequest{SessionId: sessionID})
}

func (c *DataServiceClient) CloseSession(ctx context.Context, sessionID string) (*pb.CloseSessionResponse, error) {
	return c.client.CloseSession(ctx, &pb.CloseSessionRequest{SessionId: sessionID})
}

func (c *DataServiceClient) GetDeviceSessions(ctx context.Context, deviceID uint32) (*pb.GetDeviceSessionsResponse, error) {
	return c.client.GetDeviceSessions(ctx, &pb.GetDeviceSessionsRequest{DeviceId: deviceID})
}
