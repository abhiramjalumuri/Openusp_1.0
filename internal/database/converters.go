package database

import (
	"time"

	pb "openusp/pkg/proto/dataservice"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConvertDeviceToProto converts database Device to protobuf Device
func ConvertDeviceToProto(device *Device) *pb.Device {
	if device == nil {
		return nil
	}

	pbDevice := &pb.Device{
		Id:              uint32(device.ID),
		EndpointId:      device.EndpointID,
		ProductClass:    device.ProductClass,
		Manufacturer:    device.Manufacturer,
		ModelName:       device.ModelName,
		SerialNumber:    device.SerialNumber,
		SoftwareVersion: device.SoftwareVersion,
		HardwareVersion: device.HardwareVersion,
		Status:          device.Status,
		IpAddress:       device.IPAddress,
		ConnectionType:  device.ConnectionType,
		CreatedAt:       timestamppb.New(device.CreatedAt),
		UpdatedAt:       timestamppb.New(device.UpdatedAt),
	}

	if device.LastSeen != nil {
		pbDevice.LastSeen = timestamppb.New(*device.LastSeen)
	}

	return pbDevice
}

// ConvertProtoToDevice converts protobuf Device to database Device
func ConvertProtoToDevice(pbDevice *pb.Device) *Device {
	if pbDevice == nil {
		return nil
	}

	device := &Device{
		ID:              uint(pbDevice.Id),
		EndpointID:      pbDevice.EndpointId,
		ProductClass:    pbDevice.ProductClass,
		Manufacturer:    pbDevice.Manufacturer,
		ModelName:       pbDevice.ModelName,
		SerialNumber:    pbDevice.SerialNumber,
		SoftwareVersion: pbDevice.SoftwareVersion,
		HardwareVersion: pbDevice.HardwareVersion,
		Status:          pbDevice.Status,
		IPAddress:       pbDevice.IpAddress,
		ConnectionType:  pbDevice.ConnectionType,
	}

	if pbDevice.LastSeen != nil {
		lastSeen := pbDevice.LastSeen.AsTime()
		device.LastSeen = &lastSeen
	}

	if pbDevice.CreatedAt != nil {
		device.CreatedAt = pbDevice.CreatedAt.AsTime()
	}

	if pbDevice.UpdatedAt != nil {
		device.UpdatedAt = pbDevice.UpdatedAt.AsTime()
	}

	return device
}

// ConvertParameterToProto converts database Parameter to protobuf Parameter
func ConvertParameterToProto(param *Parameter) *pb.Parameter {
	if param == nil {
		return nil
	}

	return &pb.Parameter{
		Id:          uint32(param.ID),
		DeviceId:    uint32(param.DeviceID),
		Path:        param.Path,
		Value:       param.Value,
		Type:        param.Type,
		Writable:    param.Writable,
		LastUpdated: timestamppb.New(param.LastUpdated),
		CreatedAt:   timestamppb.New(param.CreatedAt),
		UpdatedAt:   timestamppb.New(param.UpdatedAt),
	}
}

// ConvertProtoToParameter converts protobuf Parameter to database Parameter
func ConvertProtoToParameter(pbParam *pb.Parameter) *Parameter {
	if pbParam == nil {
		return nil
	}

	param := &Parameter{
		ID:       uint(pbParam.Id),
		DeviceID: uint(pbParam.DeviceId),
		Path:     pbParam.Path,
		Value:    pbParam.Value,
		Type:     pbParam.Type,
		Writable: pbParam.Writable,
	}

	if pbParam.LastUpdated != nil {
		param.LastUpdated = pbParam.LastUpdated.AsTime()
	} else {
		param.LastUpdated = time.Now()
	}

	if pbParam.CreatedAt != nil {
		param.CreatedAt = pbParam.CreatedAt.AsTime()
	}

	if pbParam.UpdatedAt != nil {
		param.UpdatedAt = pbParam.UpdatedAt.AsTime()
	}

	return param
}

// ConvertAlertToProto converts database Alert to protobuf Alert
func ConvertAlertToProto(alert *Alert) *pb.Alert {
	if alert == nil {
		return nil
	}

	pbAlert := &pb.Alert{
		Id:        uint32(alert.ID),
		DeviceId:  uint32(alert.DeviceID),
		Type:      alert.Type,
		Code:      alert.Code,
		Message:   alert.Message,
		Severity:  int32(alert.Severity),
		Resolved:  alert.Resolved,
		CreatedAt: timestamppb.New(alert.CreatedAt),
		UpdatedAt: timestamppb.New(alert.UpdatedAt),
	}

	if alert.ResolvedAt != nil {
		pbAlert.ResolvedAt = timestamppb.New(*alert.ResolvedAt)
	}

	return pbAlert
}

// ConvertProtoToAlert converts protobuf Alert to database Alert
func ConvertProtoToAlert(pbAlert *pb.Alert) *Alert {
	if pbAlert == nil {
		return nil
	}

	alert := &Alert{
		ID:       uint(pbAlert.Id),
		DeviceID: uint(pbAlert.DeviceId),
		Type:     pbAlert.Type,
		Code:     pbAlert.Code,
		Message:  pbAlert.Message,
		Severity: int(pbAlert.Severity),
		Resolved: pbAlert.Resolved,
	}

	if pbAlert.ResolvedAt != nil {
		resolvedAt := pbAlert.ResolvedAt.AsTime()
		alert.ResolvedAt = &resolvedAt
	}

	if pbAlert.CreatedAt != nil {
		alert.CreatedAt = pbAlert.CreatedAt.AsTime()
	}

	if pbAlert.UpdatedAt != nil {
		alert.UpdatedAt = pbAlert.UpdatedAt.AsTime()
	}

	return alert
}

// ConvertSessionToProto converts database Session to protobuf Session
func ConvertSessionToProto(session *Session) *pb.Session {
	if session == nil {
		return nil
	}

	pbSession := &pb.Session{
		Id:           uint32(session.ID),
		DeviceId:     uint32(session.DeviceID),
		SessionId:    session.SessionID,
		Protocol:     session.Protocol,
		Status:       session.Status,
		StartedAt:    timestamppb.New(session.StartedAt),
		LastActivity: timestamppb.New(session.LastActivity),
		MessageCount: int32(session.MessageCount),
		CreatedAt:    timestamppb.New(session.CreatedAt),
		UpdatedAt:    timestamppb.New(session.UpdatedAt),
	}

	if session.EndedAt != nil {
		pbSession.EndedAt = timestamppb.New(*session.EndedAt)
	}

	return pbSession
}

// ConvertProtoToSession converts protobuf Session to database Session
func ConvertProtoToSession(pbSession *pb.Session) *Session {
	if pbSession == nil {
		return nil
	}

	session := &Session{
		ID:           uint(pbSession.Id),
		DeviceID:     uint(pbSession.DeviceId),
		SessionID:    pbSession.SessionId,
		Protocol:     pbSession.Protocol,
		Status:       pbSession.Status,
		MessageCount: int(pbSession.MessageCount),
	}

	if pbSession.StartedAt != nil {
		session.StartedAt = pbSession.StartedAt.AsTime()
	} else {
		session.StartedAt = time.Now()
	}

	if pbSession.EndedAt != nil {
		endedAt := pbSession.EndedAt.AsTime()
		session.EndedAt = &endedAt
	}

	if pbSession.LastActivity != nil {
		session.LastActivity = pbSession.LastActivity.AsTime()
	} else {
		session.LastActivity = time.Now()
	}

	if pbSession.CreatedAt != nil {
		session.CreatedAt = pbSession.CreatedAt.AsTime()
	}

	if pbSession.UpdatedAt != nil {
		session.UpdatedAt = pbSession.UpdatedAt.AsTime()
	}

	return session
}

// Helper functions to convert slices
func ConvertDevicesToProto(devices []Device) []*pb.Device {
	result := make([]*pb.Device, len(devices))
	for i, device := range devices {
		result[i] = ConvertDeviceToProto(&device)
	}
	return result
}

func ConvertParametersToProto(params []Parameter) []*pb.Parameter {
	result := make([]*pb.Parameter, len(params))
	for i, param := range params {
		result[i] = ConvertParameterToProto(&param)
	}
	return result
}

func ConvertAlertsToProto(alerts []Alert) []*pb.Alert {
	result := make([]*pb.Alert, len(alerts))
	for i, alert := range alerts {
		result[i] = ConvertAlertToProto(&alert)
	}
	return result
}

func ConvertSessionsToProto(sessions []Session) []*pb.Session {
	result := make([]*pb.Session, len(sessions))
	for i, session := range sessions {
		result[i] = ConvertSessionToProto(&session)
	}
	return result
}
