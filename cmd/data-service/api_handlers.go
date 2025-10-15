package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	confluentkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"openusp/pkg/kafka"
)

// handleAPIRequest processes API requests from the API Gateway via Kafka
func (ds *DataService) handleAPIRequest(msg *confluentkafka.Message) error {
	// Parse the request
	var req kafka.APIRequest
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		return fmt.Errorf("failed to parse API request: %w", err)
	}

	log.Printf("🔄 API Request: %s %s (correlation: %s)", req.Method, req.Operation, req.CorrelationID)

	// Process the request based on operation
	var response kafka.APIResponse
	response.CorrelationID = req.CorrelationID

	switch req.Operation {
	case "ListDevices":
		response = ds.handleListDevices(&req)
	case "GetDevice":
		response = ds.handleGetDevice(&req)
	case "ProvisionDevice":
		response = ds.handleProvisionDevice(&req)
	case "UpdateDevice":
		response = ds.handleUpdateDevice(&req)
	case "DeleteDevice":
		response = ds.handleDeleteDevice(&req)
	case "GetParameters":
		response = ds.handleGetParameters(&req)
	case "SetParameter":
		response = ds.handleSetParameter(&req)
	case "BulkGetParameters":
		response = ds.handleBulkGetParameters(&req)
	case "BulkSetParameters":
		response = ds.handleBulkSetParameters(&req)
	case "SearchParameters":
		response = ds.handleSearchParameters(&req)
	case "GetObjects":
		response = ds.handleGetObjects(&req)
	case "GetObjectByPath":
		response = ds.handleGetObjectByPath(&req)
	case "GetDataModel":
		response = ds.handleGetDataModel(&req)
	case "GetSchema":
		response = ds.handleGetSchema(&req)
	case "GetObjectTree":
		response = ds.handleGetObjectTree(&req)
	case "ListAlerts":
		response = ds.handleListAlerts(&req)
	case "CreateAlert":
		response = ds.handleCreateAlert(&req)
	case "ResolveAlert":
		response = ds.handleResolveAlert(&req)
	case "GetDeviceAlerts":
		response = ds.handleGetDeviceAlerts(&req)
	case "ListSessions":
		response = ds.handleListSessions(&req)
	case "GetSession":
		response = ds.handleGetSession(&req)
	case "GetDeviceSessions":
		response = ds.handleGetDeviceSessions(&req)
	default:
		response = kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusNotImplemented,
			Error:         fmt.Sprintf("Operation not implemented: %s", req.Operation),
		}
	}

	// Send response back via Kafka
	return ds.sendAPIResponse(&response)
}

// sendAPIResponse publishes an API response to Kafka
func (ds *DataService) sendAPIResponse(resp *kafka.APIResponse) error {
	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal API response: %w", err)
	}

	return ds.kafkaProducer.PublishRaw(
		ds.config.Kafka.Topics.APIResponse,
		resp.CorrelationID,
		respData,
	)
}

// Device handlers
func (ds *DataService) handleListDevices(req *kafka.APIRequest) kafka.APIResponse {
	// Parse query parameters
	limit := 100
	offset := 0

	if limitStr, ok := req.Query["limit"]; ok {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr, ok := req.Query["offset"]; ok {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get devices from database
	devices, err := ds.repos.Device.GetAll(offset, limit)
	if err != nil {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusInternalServerError,
			Error:         fmt.Sprintf("Failed to list devices: %v", err),
		}
	}

	// Convert to response format
	deviceList := make([]map[string]interface{}, len(devices))
	for i, dev := range devices {
		deviceList[i] = map[string]interface{}{
			"id":               dev.ID,
			"endpoint_id":      dev.EndpointID,
			"product_class":    dev.ProductClass,
			"manufacturer":     dev.Manufacturer,
			"model_name":       dev.ModelName,
			"serial_number":    dev.SerialNumber,
			"software_version": dev.SoftwareVersion,
			"hardware_version": dev.HardwareVersion,
			"status":           dev.Status,
			"ip_address":       dev.IPAddress,
			"connection_type":  dev.ConnectionType,
			"created_at":       dev.CreatedAt,
			"updated_at":       dev.UpdatedAt,
		}
	}

	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusOK,
		Data: map[string]interface{}{
			"devices": deviceList,
			"total":   len(deviceList),
			"limit":   limit,
			"offset":  offset,
		},
	}
}

func (ds *DataService) handleGetDevice(req *kafka.APIRequest) kafka.APIResponse {
	deviceID := req.Params["device_id"]
	if deviceID == "" {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusBadRequest,
			Error:         "Missing device_id parameter",
		}
	}

	// Convert to uint
	id, err := strconv.ParseUint(deviceID, 10, 32)
	if err != nil {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusBadRequest,
			Error:         "Invalid device_id format",
		}
	}

	device, err := ds.repos.Device.GetByID(uint(id))
	if err != nil {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusNotFound,
			Error:         fmt.Sprintf("Device not found: %v", err),
		}
	}

	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusOK,
		Data: map[string]interface{}{
			"id":               device.ID,
			"endpoint_id":      device.EndpointID,
			"product_class":    device.ProductClass,
			"manufacturer":     device.Manufacturer,
			"model_name":       device.ModelName,
			"serial_number":    device.SerialNumber,
			"software_version": device.SoftwareVersion,
			"hardware_version": device.HardwareVersion,
			"status":           device.Status,
			"ip_address":       device.IPAddress,
			"connection_type":  device.ConnectionType,
			"created_at":       device.CreatedAt,
			"updated_at":       device.UpdatedAt,
		},
	}
}

func (ds *DataService) handleProvisionDevice(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "ProvisionDevice not yet implemented",
	}
}

func (ds *DataService) handleUpdateDevice(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "UpdateDevice not yet implemented",
	}
}

func (ds *DataService) handleDeleteDevice(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "DeleteDevice not yet implemented",
	}
}

// Parameter handlers
func (ds *DataService) handleGetParameters(req *kafka.APIRequest) kafka.APIResponse {
	deviceID := req.Params["device_id"]
	if deviceID == "" {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusBadRequest,
			Error:         "Missing device_id parameter",
		}
	}

	id, err := strconv.ParseUint(deviceID, 10, 32)
	if err != nil {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusBadRequest,
			Error:         "Invalid device_id format",
		}
	}

	parameters, err := ds.repos.Parameter.GetByDeviceID(uint(id))
	if err != nil {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusInternalServerError,
			Error:         fmt.Sprintf("Failed to get parameters: %v", err),
		}
	}

	paramList := make([]map[string]interface{}, len(parameters))
	for i, param := range parameters {
		paramList[i] = map[string]interface{}{
			"id":         param.ID,
			"device_id":  param.DeviceID,
			"path":       param.Path,
			"value":      param.Value,
			"type":       param.Type,
			"writable":   param.Writable,
			"created_at": param.CreatedAt,
			"updated_at": param.UpdatedAt,
		}
	}

	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusOK,
		Data: map[string]interface{}{
			"parameters": paramList,
			"total":      len(paramList),
		},
	}
}

func (ds *DataService) handleSetParameter(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "SetParameter not yet implemented",
	}
}

func (ds *DataService) handleBulkGetParameters(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "BulkGetParameters not yet implemented",
	}
}

func (ds *DataService) handleBulkSetParameters(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "BulkSetParameters not yet implemented",
	}
}

func (ds *DataService) handleSearchParameters(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "SearchParameters not yet implemented",
	}
}

// Object handlers
func (ds *DataService) handleGetObjects(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "GetObjects not yet implemented",
	}
}

func (ds *DataService) handleGetObjectByPath(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "GetObjectByPath not yet implemented",
	}
}

// Data model handlers
func (ds *DataService) handleGetDataModel(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "GetDataModel not yet implemented",
	}
}

func (ds *DataService) handleGetSchema(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "GetSchema not yet implemented",
	}
}

func (ds *DataService) handleGetObjectTree(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "GetObjectTree not yet implemented",
	}
}

// Alert handlers
func (ds *DataService) handleListAlerts(req *kafka.APIRequest) kafka.APIResponse {
	limit := 100
	offset := 0

	if limitStr, ok := req.Query["limit"]; ok {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr, ok := req.Query["offset"]; ok {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	alerts, err := ds.repos.Alert.GetAll(offset, limit)
	if err != nil {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusInternalServerError,
			Error:         fmt.Sprintf("Failed to list alerts: %v", err),
		}
	}

	alertList := make([]map[string]interface{}, len(alerts))
	for i, alert := range alerts {
		alertList[i] = map[string]interface{}{
			"id":         alert.ID,
			"device_id":  alert.DeviceID,
			"type":       alert.Type,
			"severity":   alert.Severity,
			"message":    alert.Message,
			"resolved":   alert.Resolved,
			"created_at": alert.CreatedAt,
			"updated_at": alert.UpdatedAt,
		}
	}

	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusOK,
		Data: map[string]interface{}{
			"alerts": alertList,
			"total":  len(alertList),
			"limit":  limit,
			"offset": offset,
		},
	}
}

func (ds *DataService) handleCreateAlert(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "CreateAlert not yet implemented",
	}
}

func (ds *DataService) handleResolveAlert(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "ResolveAlert not yet implemented",
	}
}

func (ds *DataService) handleGetDeviceAlerts(req *kafka.APIRequest) kafka.APIResponse {
	deviceID := req.Params["device_id"]
	if deviceID == "" {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusBadRequest,
			Error:         "Missing device_id parameter",
		}
	}

	id, err := strconv.ParseUint(deviceID, 10, 32)
	if err != nil {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusBadRequest,
			Error:         "Invalid device_id format",
		}
	}

	alerts, err := ds.repos.Alert.GetByDeviceID(uint(id), nil)
	if err != nil {
		return kafka.APIResponse{
			CorrelationID: req.CorrelationID,
			Status:        http.StatusInternalServerError,
			Error:         fmt.Sprintf("Failed to get device alerts: %v", err),
		}
	}

	alertList := make([]map[string]interface{}, len(alerts))
	for i, alert := range alerts {
		alertList[i] = map[string]interface{}{
			"id":         alert.ID,
			"device_id":  alert.DeviceID,
			"type":       alert.Type,
			"severity":   alert.Severity,
			"message":    alert.Message,
			"resolved":   alert.Resolved,
			"created_at": alert.CreatedAt,
			"updated_at": alert.UpdatedAt,
		}
	}

	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusOK,
		Data: map[string]interface{}{
			"alerts": alertList,
			"total":  len(alertList),
		},
	}
}

// Session handlers
func (ds *DataService) handleListSessions(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "ListSessions not yet implemented",
	}
}

func (ds *DataService) handleGetSession(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "GetSession not yet implemented",
	}
}

func (ds *DataService) handleGetDeviceSessions(req *kafka.APIRequest) kafka.APIResponse {
	return kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Status:        http.StatusNotImplemented,
		Error:         "GetDeviceSessions not yet implemented",
	}
}
