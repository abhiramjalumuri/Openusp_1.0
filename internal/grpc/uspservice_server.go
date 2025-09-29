package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	"openusp/internal/usp"
	"openusp/pkg/proto/dataservice"
	"openusp/pkg/proto/uspservice"
	"openusp/pkg/proto/v1_3"
	"openusp/pkg/proto/v1_4"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// USPServiceServer implements the USP service gRPC server
type USPServiceServer struct {
	uspservice.UnimplementedUSPServiceServer
	dataClient        dataservice.DataServiceClient
	processedMessages int32
	onboardedDevices  int32
	startTime         time.Time
}

// NewUSPServiceServer creates a new USP service server
func NewUSPServiceServer(dataClient dataservice.DataServiceClient) *USPServiceServer {
	return &USPServiceServer{
		dataClient:        dataClient,
		processedMessages: 0,
		onboardedDevices:  0,
		startTime:         time.Now(),
	}
}

// ProcessUSPMessage processes incoming USP messages
func (s *USPServiceServer) ProcessUSPMessage(ctx context.Context, req *uspservice.USPMessageRequest) (*uspservice.USPMessageResponse, error) {
	s.processedMessages++

	log.Printf("🔄 USP Service processing message from client %s (transport: %s)", req.ClientId, req.TransportType)

	// Parse the USP message
	parser := usp.NewUSPParser()
	parsed, err := parser.ParseUSP(req.UspData)
	if err != nil {
		log.Printf("❌ Failed to parse USP message: %v", err)
		return &uspservice.USPMessageResponse{
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Failed to parse USP message: %v", err),
		}, nil
	}

	log.Printf("📋 Parsed USP %s message: %s -> %s (%s)",
		string(parsed.Record.Version), parsed.Record.FromID, parsed.Record.ToID, parsed.Message.MsgType)

	// Check if this is a device onboarding message
	deviceInfo, isOnboarding := s.checkDeviceOnboarding(parsed)
	if isOnboarding {
		log.Printf("🎯 Device onboarding detected for endpoint: %s", deviceInfo.EndpointId)
		log.Printf("   Manufacturer: %s", deviceInfo.Manufacturer)
		log.Printf("   Product Class: %s", deviceInfo.ProductClass)
		log.Printf("   Serial Number: %s", deviceInfo.SerialNumber)
		log.Printf("   Protocol Versions: %s", deviceInfo.SupportedProtocolVersions)

		// Create device record via data service
		device := &dataservice.Device{
			EndpointId:      deviceInfo.EndpointId,
			ProductClass:    deviceInfo.ProductClass,
			SerialNumber:    deviceInfo.SerialNumber,
			Manufacturer:    deviceInfo.Manufacturer,
			ModelName:       deviceInfo.ModelName,
			SoftwareVersion: deviceInfo.SoftwareVersion,
			HardwareVersion: deviceInfo.HardwareVersion,
			Status:          "online",
			LastSeen:        timestamppb.Now(),
			IpAddress:       "127.0.0.1", // Will be updated by transport layer
			ConnectionType:  req.TransportType,
		}

		createReq := &dataservice.CreateDeviceRequest{Device: device}
		resp, err := s.dataClient.CreateDevice(context.Background(), createReq)
		if err != nil {
			log.Printf("❌ Failed to create device record: %v", err)
		} else {
			s.onboardedDevices++
			log.Printf("✅ Device onboarded successfully: %s (Database ID: %d)", deviceInfo.EndpointId, resp.Device.Id)
			log.Printf("   📝 Manufacturer: %s", resp.Device.Manufacturer)
			log.Printf("   📝 Model: %s", resp.Device.ModelName)
			log.Printf("   📝 Serial: %s", resp.Device.SerialNumber)
			log.Printf("   📝 Status: %s", resp.Device.Status)
			log.Printf("   📝 Connection: %s", resp.Device.ConnectionType)
		}
	}

	// Generate USP response
	responseData, err := s.generateUSPResponse(parsed)
	if err != nil {
		log.Printf("❌ Failed to generate USP response: %v", err)
		return &uspservice.USPMessageResponse{
			Status:       "error",
			ErrorMessage: fmt.Sprintf("Failed to generate response: %v", err),
		}, nil
	}

	response := &uspservice.USPMessageResponse{
		ResponseData:    responseData,
		Status:          "success",
		DeviceOnboarded: isOnboarding,
	}

	if isOnboarding {
		response.DeviceInfo = deviceInfo
	}

	log.Printf("✅ USP message processed successfully, response size: %d bytes", len(responseData))
	return response, nil
}

// checkDeviceOnboarding checks if the message is a device onboarding request
func (s *USPServiceServer) checkDeviceOnboarding(parsed *usp.ParsedUSP) (*uspservice.DeviceInfo, bool) {
	if parsed == nil || parsed.Message == nil || parsed.Message.MsgType != "NOTIFY" {
		return nil, false
	}

	// Check if we have a USP 1.4 message with onboard request
	if parsed.Record.Version == usp.Version14 && parsed.Message.Message14 != nil {
		return s.checkUSP14Onboarding(parsed)
	}

	// Check if we have a USP 1.3 message with onboard request
	if parsed.Record.Version == usp.Version13 && parsed.Message.Message13 != nil {
		return s.checkUSP13Onboarding(parsed)
	}

	return nil, false
}

// checkUSP14Onboarding checks for onboarding in USP 1.4 messages
func (s *USPServiceServer) checkUSP14Onboarding(parsed *usp.ParsedUSP) (*uspservice.DeviceInfo, bool) {
	msg := parsed.Message.Message14
	if msg.Body == nil {
		return nil, false
	}

	// Check if this is a request
	request := msg.Body.GetRequest()
	if request == nil {
		return nil, false
	}

	// Check if this is a notify request
	notify := request.GetNotify()
	if notify == nil {
		return nil, false
	}

	// Check if this is an onboard request
	onBoardReq := notify.GetOnBoardReq()
	if onBoardReq == nil {
		return nil, false
	}

	log.Printf("🎯 Device onboarding detected for endpoint: %s", parsed.Record.FromID)

	// Extract device information from the onboard request
	endpointID := parsed.Record.FromID
	productClass := onBoardReq.ProductClass
	serialNumber := onBoardReq.SerialNumber
	supportedVersions := onBoardReq.AgentSupportedProtocolVersions
	oui := onBoardReq.Oui

	// Generate manufacturer from OUI (basic lookup)
	manufacturer := s.getManufacturerFromOUI(oui)

	// Set defaults if not provided
	if productClass == "" {
		productClass = "OpenUSP Demo Agent"
	}
	if serialNumber == "" {
		serialNumber = "DEMO-001-2024"
	}
	if manufacturer == "" {
		manufacturer = "OpenUSP"
	}

	deviceInfo := &uspservice.DeviceInfo{
		EndpointId:                endpointID,
		ProductClass:              productClass,
		SerialNumber:              serialNumber,
		Manufacturer:              manufacturer,
		ModelName:                 productClass,
		SoftwareVersion:           "1.0.0",
		HardwareVersion:           "1.0",
		SupportedProtocolVersions: supportedVersions,
	}

	return deviceInfo, true
}

// checkUSP13Onboarding checks for onboarding in USP 1.3 messages
func (s *USPServiceServer) checkUSP13Onboarding(parsed *usp.ParsedUSP) (*uspservice.DeviceInfo, bool) {
	msg := parsed.Message.Message13
	if msg.Body == nil {
		return nil, false
	}

	// Check if this is a request
	request := msg.Body.GetRequest()
	if request == nil {
		return nil, false
	}

	// Check if this is a notify request
	notify := request.GetNotify()
	if notify == nil {
		return nil, false
	}

	// Check if this is an onboard request
	onBoardReq := notify.GetOnBoardReq()
	if onBoardReq == nil {
		return nil, false
	}

	log.Printf("🎯 Device onboarding detected for endpoint: %s", parsed.Record.FromID)

	// Extract device information from the onboard request
	endpointID := parsed.Record.FromID
	productClass := onBoardReq.ProductClass
	serialNumber := onBoardReq.SerialNumber
	supportedVersions := onBoardReq.AgentSupportedProtocolVersions
	oui := onBoardReq.Oui

	// Generate manufacturer from OUI
	manufacturer := s.getManufacturerFromOUI(oui)

	// Set defaults if not provided
	if productClass == "" {
		productClass = "OpenUSP Demo Agent"
	}
	if serialNumber == "" {
		serialNumber = "DEMO-001-2024"
	}
	if manufacturer == "" {
		manufacturer = "OpenUSP"
	}

	deviceInfo := &uspservice.DeviceInfo{
		EndpointId:                endpointID,
		ProductClass:              productClass,
		SerialNumber:              serialNumber,
		Manufacturer:              manufacturer,
		ModelName:                 productClass,
		SoftwareVersion:           "1.0.0",
		HardwareVersion:           "1.0",
		SupportedProtocolVersions: supportedVersions,
	}

	return deviceInfo, true
}

// getManufacturerFromOUI returns manufacturer name from IEEE OUI
func (s *USPServiceServer) getManufacturerFromOUI(oui string) string {
	// Basic OUI to manufacturer mapping
	ouiMap := map[string]string{
		"001122":   "OpenUSP Demo",
		"00D09E":   "Broadband Forum",
		"001CF0":   "Comcast",
		"5C:E0:C5": "Plume Design",
		"":         "Unknown",
	}

	if manufacturer, exists := ouiMap[oui]; exists {
		return manufacturer
	}

	return "Unknown Manufacturer"
}

// generateUSPResponse generates a USP response based on the parsed message
func (s *USPServiceServer) generateUSPResponse(parsed *usp.ParsedUSP) ([]byte, error) {
	if parsed.Record.Version == usp.Version14 {
		return s.generateUSP14Response(parsed)
	} else if parsed.Record.Version == usp.Version13 {
		return s.generateUSP13Response(parsed)
	}

	return nil, fmt.Errorf("unsupported USP version: %s", string(parsed.Record.Version))
}

// generateUSP14Response generates USP 1.4 response
func (s *USPServiceServer) generateUSP14Response(parsed *usp.ParsedUSP) ([]byte, error) {
	// Create a simple acknowledgment response
	responseMsg := &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   fmt.Sprintf("resp-%s", parsed.Message.MsgID),
			MsgType: v1_4.Header_NOTIFY_RESP, // For onboarding notifications
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Response{
				Response: &v1_4.Response{
					RespType: &v1_4.Response_NotifyResp{
						NotifyResp: &v1_4.NotifyResp{
							SubscriptionId: "onboard-response",
						},
					},
				},
			},
		},
	}

	// If it's a GET request, handle it properly
	if parsed.Message.MsgType == "GET" {
		responseMsg.Header.MsgType = v1_4.Header_GET_RESP
		responseMsg.Body = &v1_4.Body{
			MsgBody: &v1_4.Body_Response{
				Response: &v1_4.Response{
					RespType: &v1_4.Response_GetResp{
						GetResp: &v1_4.GetResp{
							ReqPathResults: []*v1_4.GetResp_RequestedPathResult{
								{
									RequestedPath: "Device.DeviceInfo.",
									ErrCode:       0,
									ErrMsg:        "",
								},
							},
						},
					},
				},
			},
		}
	}

	// Marshal response message
	responsePayload, err := proto.Marshal(responseMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.4 response: %w", err)
	}

	// Create response record
	responseRecord := &v1_4.Record{
		Version: "1.4",
		ToId:    parsed.Record.FromID,
		FromId:  parsed.Record.ToID,
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: responsePayload,
			},
		},
	}

	return proto.Marshal(responseRecord)
}

// generateUSP13Response generates USP 1.3 response
func (s *USPServiceServer) generateUSP13Response(parsed *usp.ParsedUSP) ([]byte, error) {
	// Create a simple acknowledgment response
	responseMsg := &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   fmt.Sprintf("resp-%s", parsed.Message.MsgID),
			MsgType: v1_3.Header_NOTIFY_RESP,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Response{
				Response: &v1_3.Response{
					RespType: &v1_3.Response_NotifyResp{
						NotifyResp: &v1_3.NotifyResp{
							SubscriptionId: "onboard-response",
						},
					},
				},
			},
		},
	}

	// Marshal response message
	responsePayload, err := proto.Marshal(responseMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.3 response: %w", err)
	}

	// Create response record
	responseRecord := &v1_3.Record{
		Version: "1.3",
		ToId:    parsed.Record.FromID,
		FromId:  parsed.Record.ToID,
		RecordType: &v1_3.Record_NoSessionContext{
			NoSessionContext: &v1_3.NoSessionContextRecord{
				Payload: responsePayload,
			},
		},
	}

	return proto.Marshal(responseRecord)
}

// GetServiceHealth returns the health status
func (s *USPServiceServer) GetServiceHealth(ctx context.Context, req *uspservice.HealthRequest) (*uspservice.HealthResponse, error) {
	return &uspservice.HealthResponse{
		Status:    "healthy",
		Message:   "USP Service is running normally",
		Timestamp: time.Now().Unix(),
	}, nil
}

// GetServiceStatus returns detailed status information
func (s *USPServiceServer) GetServiceStatus(ctx context.Context, req *uspservice.StatusRequest) (*uspservice.StatusResponse, error) {
	uptime := time.Since(s.startTime)

	return &uspservice.StatusResponse{
		ServiceName:       "OpenUSP Core Service",
		Version:           "1.0.0",
		Uptime:            uptime.String(),
		ProcessedMessages: s.processedMessages,
		OnboardedDevices:  s.onboardedDevices,
		Statistics: map[string]string{
			"start_time":        s.startTime.Format(time.RFC3339),
			"data_service_conn": "connected",
			"usp_versions":      "1.3, 1.4",
		},
	}, nil
}
