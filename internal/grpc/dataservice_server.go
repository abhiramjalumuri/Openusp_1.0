package grpc

import (
	"context"
	"fmt"
	"time"

	"openusp/internal/database"
	pb "openusp/pkg/proto/dataservice"
	"openusp/pkg/version"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DataServiceServer implements the gRPC DataService interface
type DataServiceServer struct {
	pb.UnimplementedDataServiceServer
	db    *database.Database
	repos *database.Repositories
}

// NewDataServiceServer creates a new gRPC server instance
func NewDataServiceServer(db *database.Database, repos *database.Repositories) *DataServiceServer {
	return &DataServiceServer{
		db:    db,
		repos: repos,
	}
}

// Health and Status methods
func (s *DataServiceServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	if err := s.db.Ping(); err != nil {
		return &pb.HealthCheckResponse{
			Status:  "unhealthy",
			Version: version.GetShortVersion(),
		}, status.Errorf(codes.Unavailable, "database unavailable: %v", err)
	}

	return &pb.HealthCheckResponse{
		Status:  "healthy",
		Version: version.GetShortVersion(),
	}, nil
}

func (s *DataServiceServer) GetStatus(ctx context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	stats := s.db.GetStats()
	deviceCount, _ := s.repos.Device.Count()

	// Convert stats to string map
	statsMap := make(map[string]string)
	for k, v := range stats {
		statsMap[k] = fmt.Sprintf("%v", v)
	}

	return &pb.GetStatusResponse{
		Service:       "OpenUSP Data Service",
		Version:       version.GetShortVersion(),
		DatabaseStats: statsMap,
		DeviceCount:   deviceCount,
		Timestamp:     time.Now().Unix(),
	}, nil
}

// Device operations
func (s *DataServiceServer) CreateDevice(ctx context.Context, req *pb.CreateDeviceRequest) (*pb.CreateDeviceResponse, error) {
	if req.Device == nil {
		return nil, status.Errorf(codes.InvalidArgument, "device is required")
	}

	device := database.ConvertProtoToDevice(req.Device)
	if err := s.repos.Device.CreateOrUpdate(device); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create device: %v", err)
	}

	return &pb.CreateDeviceResponse{
		Device: database.ConvertDeviceToProto(device),
	}, nil
}

func (s *DataServiceServer) GetDevice(ctx context.Context, req *pb.GetDeviceRequest) (*pb.GetDeviceResponse, error) {
	if req.Id == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "device ID is required")
	}

	device, err := s.repos.Device.GetByID(uint(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "device not found: %v", err)
	}

	return &pb.GetDeviceResponse{
		Device: database.ConvertDeviceToProto(device),
	}, nil
}

func (s *DataServiceServer) GetDeviceByEndpoint(ctx context.Context, req *pb.GetDeviceByEndpointRequest) (*pb.GetDeviceByEndpointResponse, error) {
	if req.EndpointId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "endpoint ID is required")
	}

	device, err := s.repos.Device.GetByEndpointID(req.EndpointId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "device not found: %v", err)
	}

	return &pb.GetDeviceByEndpointResponse{
		Device: database.ConvertDeviceToProto(device),
	}, nil
}

func (s *DataServiceServer) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	offset := int(req.Offset)
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}

	devices, err := s.repos.Device.GetAll(offset, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list devices: %v", err)
	}

	total, _ := s.repos.Device.Count()

	return &pb.ListDevicesResponse{
		Devices: database.ConvertDevicesToProto(devices),
		Total:   total,
	}, nil
}

func (s *DataServiceServer) UpdateDevice(ctx context.Context, req *pb.UpdateDeviceRequest) (*pb.UpdateDeviceResponse, error) {
	if req.Device == nil {
		return nil, status.Errorf(codes.InvalidArgument, "device is required")
	}

	device := database.ConvertProtoToDevice(req.Device)
	if err := s.repos.Device.Update(device); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update device: %v", err)
	}

	return &pb.UpdateDeviceResponse{
		Device: database.ConvertDeviceToProto(device),
	}, nil
}

// DeleteDevice performs cascading deletion of a device and all associated data
// This includes:
// - All TR-181 parameters and data model entries
// - All device alerts and notifications
// - All active and historical sessions
// - The device record itself
// The deletion is performed in dependency order to maintain referential integrity
func (s *DataServiceServer) DeleteDevice(ctx context.Context, req *pb.DeleteDeviceRequest) (*pb.DeleteDeviceResponse, error) {
	if req.Id == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "device ID is required")
	}

	deviceID := uint(req.Id)

	// Cascade delete all associated data in proper order
	// 1. Delete all parameters associated with the device (TR-181 data model entries)
	if err := s.repos.Parameter.DeleteByDeviceID(deviceID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete device parameters: %v", err)
	}

	// 2. Delete all alerts associated with the device
	if err := s.repos.Alert.DeleteByDeviceID(deviceID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete device alerts: %v", err)
	}

	// 3. Delete all sessions associated with the device
	if err := s.repos.Session.DeleteByDeviceID(deviceID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete device sessions: %v", err)
	}

	// 4. Finally, delete the device itself
	if err := s.repos.Device.Delete(deviceID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete device: %v", err)
	}

	return &pb.DeleteDeviceResponse{
		Success: true,
	}, nil
}

// Parameter operations
func (s *DataServiceServer) CreateParameter(ctx context.Context, req *pb.CreateParameterRequest) (*pb.CreateParameterResponse, error) {
	if req.Parameter == nil {
		return nil, status.Errorf(codes.InvalidArgument, "parameter is required")
	}

	parameter := database.ConvertProtoToParameter(req.Parameter)
	if err := s.repos.Parameter.CreateOrUpdate(parameter); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create parameter: %v", err)
	}

	return &pb.CreateParameterResponse{
		Parameter: database.ConvertParameterToProto(parameter),
	}, nil
}

func (s *DataServiceServer) GetDeviceParameters(ctx context.Context, req *pb.GetDeviceParametersRequest) (*pb.GetDeviceParametersResponse, error) {
	if req.DeviceId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "device ID is required")
	}

	parameters, err := s.repos.Parameter.GetByDeviceID(uint(req.DeviceId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get device parameters: %v", err)
	}

	return &pb.GetDeviceParametersResponse{
		Parameters: database.ConvertParametersToProto(parameters),
	}, nil
}

func (s *DataServiceServer) GetParametersByPath(ctx context.Context, req *pb.GetParametersByPathRequest) (*pb.GetParametersByPathResponse, error) {
	if req.DeviceId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "device ID is required")
	}

	pathPattern := req.PathPattern
	if pathPattern == "" {
		pathPattern = "%"
	}

	parameters, err := s.repos.Parameter.GetByPath(uint(req.DeviceId), pathPattern)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get parameters by path: %v", err)
	}

	return &pb.GetParametersByPathResponse{
		Parameters: database.ConvertParametersToProto(parameters),
	}, nil
}

func (s *DataServiceServer) DeleteParameter(ctx context.Context, req *pb.DeleteParameterRequest) (*pb.DeleteParameterResponse, error) {
	if req.DeviceId == 0 || req.Path == "" {
		return nil, status.Errorf(codes.InvalidArgument, "device ID and path are required")
	}

	if err := s.repos.Parameter.Delete(uint(req.DeviceId), req.Path); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete parameter: %v", err)
	}

	return &pb.DeleteParameterResponse{
		Success: true,
	}, nil
}

func (s *DataServiceServer) GetParametersByEndpoint(ctx context.Context, req *pb.GetParametersByEndpointRequest) (*pb.GetParametersByEndpointResponse, error) {
	if req.EndpointId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "endpoint ID is required")
	}

	// First, find the device by endpoint ID
	device, err := s.repos.Device.GetByEndpointID(req.EndpointId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "device not found with endpoint ID: %s", req.EndpointId)
	}

	// Get parameters for the device
	var parameters []database.Parameter
	if req.PathPattern != "" {
		// Filter by path pattern if provided
		parameters, err = s.repos.Parameter.GetByPath(device.ID, req.PathPattern)
	} else {
		// Get all parameters for the device
		parameters, err = s.repos.Parameter.GetByDeviceID(device.ID)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get parameters: %v", err)
	}

	return &pb.GetParametersByEndpointResponse{
		Parameters: database.ConvertParametersToProto(parameters),
		Device:     database.ConvertDeviceToProto(device),
	}, nil
}

// Alert operations
func (s *DataServiceServer) CreateAlert(ctx context.Context, req *pb.CreateAlertRequest) (*pb.CreateAlertResponse, error) {
	if req.Alert == nil {
		return nil, status.Errorf(codes.InvalidArgument, "alert is required")
	}

	alert := database.ConvertProtoToAlert(req.Alert)
	if err := s.repos.Alert.Create(alert); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create alert: %v", err)
	}

	return &pb.CreateAlertResponse{
		Alert: database.ConvertAlertToProto(alert),
	}, nil
}

func (s *DataServiceServer) GetDeviceAlerts(ctx context.Context, req *pb.GetDeviceAlertsRequest) (*pb.GetDeviceAlertsResponse, error) {
	if req.DeviceId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "device ID is required")
	}

	var resolved *bool
	if req.Resolved != nil {
		resolved = req.Resolved
	}

	alerts, err := s.repos.Alert.GetByDeviceID(uint(req.DeviceId), resolved)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get device alerts: %v", err)
	}

	return &pb.GetDeviceAlertsResponse{
		Alerts: database.ConvertAlertsToProto(alerts),
	}, nil
}

func (s *DataServiceServer) ListAlerts(ctx context.Context, req *pb.ListAlertsRequest) (*pb.ListAlertsResponse, error) {
	offset := int(req.Offset)
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}

	alerts, err := s.repos.Alert.GetAll(offset, limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list alerts: %v", err)
	}

	return &pb.ListAlertsResponse{
		Alerts: database.ConvertAlertsToProto(alerts),
	}, nil
}

func (s *DataServiceServer) ResolveAlert(ctx context.Context, req *pb.ResolveAlertRequest) (*pb.ResolveAlertResponse, error) {
	if req.Id == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "alert ID is required")
	}

	if err := s.repos.Alert.Resolve(uint(req.Id)); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve alert: %v", err)
	}

	return &pb.ResolveAlertResponse{
		Success: true,
	}, nil
}

// Session operations
func (s *DataServiceServer) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.CreateSessionResponse, error) {
	if req.Session == nil {
		return nil, status.Errorf(codes.InvalidArgument, "session is required")
	}

	session := database.ConvertProtoToSession(req.Session)
	if err := s.repos.Session.Create(session); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}

	return &pb.CreateSessionResponse{
		Session: database.ConvertSessionToProto(session),
	}, nil
}

func (s *DataServiceServer) GetSession(ctx context.Context, req *pb.GetSessionRequest) (*pb.GetSessionResponse, error) {
	if req.SessionId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "session ID is required")
	}

	session, err := s.repos.Session.GetBySessionID(req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	return &pb.GetSessionResponse{
		Session: database.ConvertSessionToProto(session),
	}, nil
}

func (s *DataServiceServer) UpdateSessionActivity(ctx context.Context, req *pb.UpdateSessionActivityRequest) (*pb.UpdateSessionActivityResponse, error) {
	if req.SessionId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "session ID is required")
	}

	if err := s.repos.Session.UpdateActivity(req.SessionId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update session activity: %v", err)
	}

	return &pb.UpdateSessionActivityResponse{
		Success: true,
	}, nil
}

func (s *DataServiceServer) CloseSession(ctx context.Context, req *pb.CloseSessionRequest) (*pb.CloseSessionResponse, error) {
	if req.SessionId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "session ID is required")
	}

	if err := s.repos.Session.Close(req.SessionId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to close session: %v", err)
	}

	return &pb.CloseSessionResponse{
		Success: true,
	}, nil
}

func (s *DataServiceServer) GetDeviceSessions(ctx context.Context, req *pb.GetDeviceSessionsRequest) (*pb.GetDeviceSessionsResponse, error) {
	if req.DeviceId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "device ID is required")
	}

	sessions, err := s.repos.Session.GetActiveSessionsForDevice(uint(req.DeviceId))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get device sessions: %v", err)
	}

	return &pb.GetDeviceSessionsResponse{
		Sessions: database.ConvertSessionsToProto(sessions),
	}, nil
}
