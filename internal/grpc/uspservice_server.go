package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	"openusp/internal/usp"
	"openusp/pkg/proto/dataservice"
	"openusp/pkg/proto/mtpservice"
	"openusp/pkg/proto/uspservice"
	"openusp/pkg/proto/v1_3"
	"openusp/pkg/proto/v1_4"
	"openusp/pkg/service/client"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// USPServiceServer implements the USP service gRPC server
type USPServiceServer struct {
	uspservice.UnimplementedUSPServiceServer
	dataClient                 dataservice.DataServiceClient
	connectionClient           *client.OpenUSPConnectionClient
	processedMessages          int32
	onboardedDevices           int32
	startTime                  time.Time
	proactiveOnboardingManager *ProactiveOnboardingManager
}

// NewUSPServiceServer creates a new USP service server
func NewUSPServiceServer(dataClient dataservice.DataServiceClient, connectionClient *client.OpenUSPConnectionClient) *USPServiceServer {
	server := &USPServiceServer{
		dataClient:        dataClient,
		connectionClient:  connectionClient,
		processedMessages: 0,
		onboardedDevices:  0,
		startTime:         time.Now(),
	}

	// Initialize proactive onboarding manager
	server.proactiveOnboardingManager = NewProactiveOnboardingManager(server, connectionClient)

	return server
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

	// Log different information based on whether this is a record-only or record+message
	if parsed.Message != nil {
		log.Printf("📋 Parsed USP %s message: %s -> %s (%s)",
			string(parsed.Record.Version), parsed.Record.FromID, parsed.Record.ToID, parsed.Message.MsgType)
	} else {
		log.Printf("📋 Parsed USP %s record: %s -> %s (%s)",
			string(parsed.Record.Version), parsed.Record.FromID, parsed.Record.ToID, parsed.Record.RecordType)
	}

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

	// Check for Get responses during proactive onboarding
	if parsed.Message != nil && s.proactiveOnboardingManager != nil {
		s.handleProactiveOnboardingResponses(parsed)
	}

	// Handle different types of USP records and messages
	var responseData []byte

	if parsed.Message != nil {
		// We have a USP message - process normally
		responseData, err = s.generateUSPResponse(parsed)
		if err != nil {
			log.Printf("❌ Failed to generate USP response: %v", err)
			return &uspservice.USPMessageResponse{
				Status:       "error",
				ErrorMessage: fmt.Sprintf("Failed to generate response: %v", err),
			}, nil
		}
	} else {
		// We have a connect/disconnect record without a USP message
		recordType := parsed.Record.RecordType

		switch recordType {
		case "WebSocketConnect", "MQTTConnect", "StompConnect", "UDSConnect":
			log.Printf("🔗 Processing %s record from agent %s", recordType, parsed.Record.FromID)

			// Generate connect acknowledgment
			responseData, err = s.generateConnectAcknowledgment(parsed)
			if err != nil {
				log.Printf("❌ Failed to generate connect acknowledgment: %v", err)
				return &uspservice.USPMessageResponse{
					Status:       "error",
					ErrorMessage: fmt.Sprintf("Failed to generate connect ack: %v", err),
				}, nil
			}

			// Debug: Verify what we're sending
			log.Printf("🔍 DEBUG: Connect ack response data size: %d bytes", len(responseData))
			if len(responseData) > 0 {
				// Try to parse it back to verify it's a Record, not a Message
				parser := usp.NewUSPParser()
				verifyParsed, verifyErr := parser.ParseUSP(responseData)
				if verifyErr == nil {
					log.Printf("✅ DEBUG: Response is a valid USP %s Record with type: %s", verifyParsed.Record.Version, verifyParsed.Record.RecordType)
					if verifyParsed.Message != nil {
						log.Printf("⚠️ WARNING: Response Record contains a USP Message (type: %s) - this is WRONG for Connect ack!", verifyParsed.Message.MsgType)
					} else {
						log.Printf("✅ DEBUG: Response Record has NO message body - correct for Connect ack")
					}
				} else {
					log.Printf("⚠️ DEBUG: Response could not be parsed: %v", verifyErr)
				}
			}

			// After sending connect ack, start proactive onboarding for new agents
			if s.isNewAgent(parsed.Record.FromID) {
				log.Printf("🆕 New agent detected: %s - starting proactive onboarding", parsed.Record.FromID)

				// Start both traditional and proactive onboarding sequences
				go s.startTR369OnboardingSequence(parsed.Record.FromID, string(parsed.Record.Version))

				// Also start proactive onboarding (handles agents that don't send NOTIFY)
				if recordType == "WebSocketConnect" {
					connectionContext := map[string]interface{}{
						"transport":  "websocket",
						"client_id":  req.ClientId,
						"first_seen": time.Now(),
					}
					s.proactiveOnboardingManager.HandleWebSocketConnect(parsed.Record.FromID, string(parsed.Record.Version), connectionContext)
				}
			} else {
				log.Printf("🔄 Known agent reconnected: %s", parsed.Record.FromID)
			}

		case "Disconnect":
			log.Printf("🔌 Processing disconnect record from agent %s", parsed.Record.FromID)
			// For disconnect, no response is typically needed
			responseData = nil

		default:
			log.Printf("❌ Unknown record type: %s from agent %s", recordType, parsed.Record.FromID)
			responseData = nil
		}
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

// createProactiveDevice creates a device entry for proactive onboarding
func (s *USPServiceServer) createProactiveDevice(deviceInfo *uspservice.DeviceInfo) error {
	log.Printf("💾 USPService: Creating proactive device entry for %s", deviceInfo.EndpointId)

	// Create device entry using the same logic as regular onboarding
	device := &dataservice.Device{
		EndpointId:      deviceInfo.EndpointId,
		Manufacturer:    deviceInfo.Manufacturer,
		ProductClass:    deviceInfo.ProductClass,
		SerialNumber:    deviceInfo.SerialNumber,
		ModelName:       deviceInfo.ModelName,
		SoftwareVersion: deviceInfo.SoftwareVersion,
		HardwareVersion: deviceInfo.HardwareVersion,
		Status:          "online",
		LastSeen:        timestamppb.Now(),
		IpAddress:       "127.0.0.1", // Will be updated by transport layer
		ConnectionType:  "websocket", // Proactive onboarding is triggered by WebSocket
	}

	createReq := &dataservice.CreateDeviceRequest{Device: device}
	resp, err := s.dataClient.CreateDevice(context.Background(), createReq)
	if err != nil {
		log.Printf("❌ Failed to create proactive device record: %v", err)
		return fmt.Errorf("failed to create device: %w", err)
	}

	// Update counters and log success
	s.onboardedDevices++
	log.Printf("✅ Proactive device created successfully: %s (Database ID: %d)", deviceInfo.EndpointId, resp.Device.Id)
	log.Printf("   📝 Manufacturer: %s", resp.Device.Manufacturer)
	log.Printf("   📝 Model: %s", resp.Device.ModelName)
	log.Printf("   📝 Serial: %s", resp.Device.SerialNumber)
	log.Printf("   📝 Status: %s", resp.Device.Status)
	log.Printf("   📝 Connection: %s", resp.Device.ConnectionType)

	return nil
}

// createProactiveDeviceWithStatus creates a device entry for proactive onboarding with specific status
func (s *USPServiceServer) createProactiveDeviceWithStatus(deviceInfo *uspservice.DeviceInfo, status string) error {
	log.Printf("💾 USPService: Creating proactive device entry for %s with status: %s", deviceInfo.EndpointId, status)

	// Create device entry using the same logic as regular onboarding but with custom status
	device := &dataservice.Device{
		EndpointId:      deviceInfo.EndpointId,
		Manufacturer:    deviceInfo.Manufacturer,
		ProductClass:    deviceInfo.ProductClass,
		SerialNumber:    deviceInfo.SerialNumber,
		ModelName:       deviceInfo.ModelName,
		SoftwareVersion: deviceInfo.SoftwareVersion,
		HardwareVersion: deviceInfo.HardwareVersion,
		Status:          status, // Use provided status instead of hardcoded "online"
		LastSeen:        timestamppb.Now(),
		IpAddress:       "127.0.0.1", // Will be updated by transport layer
		ConnectionType:  "websocket", // Proactive onboarding is triggered by WebSocket
	}

	createReq := &dataservice.CreateDeviceRequest{Device: device}
	resp, err := s.dataClient.CreateDevice(context.Background(), createReq)
	if err != nil {
		log.Printf("❌ Failed to create proactive device record: %v", err)
		return fmt.Errorf("failed to create device: %w", err)
	}

	// Update counters and log success
	s.onboardedDevices++
	log.Printf("✅ Proactive device created successfully: %s (Database ID: %d)", deviceInfo.EndpointId, resp.Device.Id)
	log.Printf("   📝 Manufacturer: %s", resp.Device.Manufacturer)
	log.Printf("   📝 Model: %s", resp.Device.ModelName)
	log.Printf("   📝 Serial: %s", resp.Device.SerialNumber)
	log.Printf("   📝 Status: %s", resp.Device.Status)
	log.Printf("   📝 Connection: %s", resp.Device.ConnectionType)

	return nil
}

// handleProactiveOnboardingResponses checks for and handles Get responses during proactive onboarding
func (s *USPServiceServer) handleProactiveOnboardingResponses(parsed *usp.ParsedUSP) {
	// Only process responses (not requests)
	if parsed.Message.MsgType != "GET_RESP" {
		return
	}

	agentID := parsed.Record.FromID
	log.Printf("📨 USP Service: Received Get response from agent %s during proactive onboarding", agentID)

	// Extract parameter values from the response based on USP version
	if parsed.Record.Version == usp.Version14 && parsed.Message.Message14 != nil {
		s.handleUSP14GetResponse(agentID, parsed)
	} else if parsed.Record.Version == usp.Version13 && parsed.Message.Message13 != nil {
		s.handleUSP13GetResponse(agentID, parsed)
	}
}

// handleUSP14GetResponse handles USP 1.4 Get responses
func (s *USPServiceServer) handleUSP14GetResponse(agentID string, parsed *usp.ParsedUSP) {
	msg := parsed.Message.Message14
	if msg.Body == nil {
		return
	}

	response := msg.Body.GetResponse()
	if response == nil {
		return
	}

	getResp := response.GetGetResp()
	if getResp == nil {
		return
	}

	// Process each requested path result
	for _, result := range getResp.ReqPathResults {
		for _, resolvedPath := range result.ResolvedPathResults {
			// Process parameters in this path (result_params is a map<string, string>)
			for paramName, paramValue := range resolvedPath.ResultParams {
				fullParamPath := resolvedPath.ResolvedPath + paramName
				log.Printf("📊 USP 1.4 Get Response: %s = %s (agent: %s)", fullParamPath, paramValue, agentID)

				// Forward to proactive onboarding manager
				s.proactiveOnboardingManager.HandleGetResponse(agentID, fullParamPath, paramValue)
			}
		}
	}
}

// handleUSP13GetResponse handles USP 1.3 Get responses
func (s *USPServiceServer) handleUSP13GetResponse(agentID string, parsed *usp.ParsedUSP) {
	msg := parsed.Message.Message13
	if msg.Body == nil {
		return
	}

	response := msg.Body.GetResponse()
	if response == nil {
		return
	}

	getResp := response.GetGetResp()
	if getResp == nil {
		return
	}

	// Process each requested path result
	for _, result := range getResp.ReqPathResults {
		for _, resolvedPath := range result.ResolvedPathResults {
			// Process parameters in this path (result_params is a map<string, string>)
			for paramName, paramValue := range resolvedPath.ResultParams {
				fullParamPath := resolvedPath.ResolvedPath + paramName
				log.Printf("📊 USP 1.3 Get Response: %s = %s (agent: %s)", fullParamPath, paramValue, agentID)

				// Forward to proactive onboarding manager
				s.proactiveOnboardingManager.HandleGetResponse(agentID, fullParamPath, paramValue)
			}
		}
	}
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
	}, nil
}

// HandleProactiveOnboarding implements TR-369 proactive onboarding when MTP connection succeeds
func (s *USPServiceServer) HandleProactiveOnboarding(ctx context.Context, req *uspservice.ProactiveOnboardingRequest) (*uspservice.ProactiveOnboardingResponse, error) {
	log.Printf("🔍 ProactiveOnboarding: Received request for endpoint %s (USP %s)", req.EndpointId, req.UspVersion)

	// Log connection context for debugging
	for key, value := range req.ConnectionContext {
		log.Printf("   %s: %s", key, value)
	}

	// Check if proactive onboarding manager is available
	if s.proactiveOnboardingManager == nil {
		log.Printf("❌ ProactiveOnboarding: Manager not initialized")
		return &uspservice.ProactiveOnboardingResponse{
			Success:          false,
			Message:          "Proactive onboarding manager not available",
			OnboardingStatus: "manager_unavailable",
			AgentId:          req.EndpointId,
		}, nil
	}

	// Convert connection context map to the expected format
	connectionContext := make(map[string]interface{})
	for key, value := range req.ConnectionContext {
		connectionContext[key] = value
	}

	// Trigger proactive onboarding per TR-369 Section 3.3.2
	log.Printf("🚀 ProactiveOnboarding: Initiating controller-driven discovery for %s", req.EndpointId)
	s.proactiveOnboardingManager.HandleWebSocketConnect(req.EndpointId, req.UspVersion, connectionContext)

	return &uspservice.ProactiveOnboardingResponse{
		Success:          true,
		Message:          "Proactive onboarding initiated successfully",
		OnboardingStatus: "initiated",
		AgentId:          req.EndpointId,
	}, nil
}

// startTR369OnboardingSequence implements the complete TR-369 onboarding flow
func (s *USPServiceServer) startTR369OnboardingSequence(agentID, version string) {
	log.Printf("🚀 Starting TR-369 onboarding sequence for agent %s (USP %s)", agentID, version)

	// Step 1: Subscribe to Boot! event first
	log.Printf("📡 Step 1: Subscribing to Boot! event for agent %s", agentID)
	err := s.subscribeToBootEvent(agentID, version)
	if err != nil {
		log.Printf("❌ Failed to subscribe to Boot! event for agent %s: %v", agentID, err)
		return
	}

	// Step 2: Wait for Boot! notification (or timeout)
	log.Printf("⏳ Step 2: Waiting for Boot! notification from agent %s", agentID)
	bootReceived := s.waitForBootNotification(agentID, 30*time.Second)

	if !bootReceived {
		log.Printf("⚠️ Boot! notification timeout for agent %s - proceeding with discovery anyway", agentID)
	} else {
		log.Printf("✅ Boot! notification received from agent %s", agentID)
	}

	// Step 3: Start discovery sequence
	log.Printf("🔍 Step 3: Starting discovery sequence for agent %s", agentID)
	s.startDiscoverySequence(agentID, version)

	log.Printf("✅ TR-369 onboarding sequence completed for agent %s", agentID)
}

// subscribeToBootEvent subscribes to Boot! events from the agent
func (s *USPServiceServer) subscribeToBootEvent(agentID, version string) error {
	log.Printf("📡 Subscribing to Boot! event for agent %s", agentID)

	// Create ADD request to subscribe to Boot! event
	// This would typically create a subscription object in the agent's data model
	// For now, we'll log the intent - actual implementation would send ADD request
	log.Printf("📋 Would send ADD request to subscribe to Device.LocalAgent.Subscription.{i}.Enable=true")
	log.Printf("📋 With reference path: Device.LocalAgent.Controller.{i}.BootParameter")

	// TODO: Implement actual USP ADD request generation and sending
	// This would create a subscription in the agent's data model

	return nil
}

// waitForBootNotification waits for Boot! notification or times out
func (s *USPServiceServer) waitForBootNotification(agentID string, timeout time.Duration) bool {
	log.Printf("⏳ Waiting up to %v for Boot! notification from agent %s", timeout, agentID)

	// For now, simulate waiting - in real implementation this would:
	// 1. Set up a channel to receive Boot! notifications
	// 2. Wait for the notification or timeout
	// 3. Return true if Boot! received, false if timeout

	// Simulate short wait
	time.Sleep(2 * time.Second)

	// For demo purposes, assume Boot! is received
	log.Printf("📢 Simulated Boot! notification received from agent %s", agentID)
	return true
}

// startDiscoverySequence starts the device discovery sequence
func (s *USPServiceServer) startDiscoverySequence(agentID, version string) {
	log.Printf("🔍 Starting device discovery sequence for agent %s", agentID)

	// Wait a brief moment to ensure agent is ready
	time.Sleep(500 * time.Millisecond)

	// Step 1: GetSupportedDM - discover what the agent supports
	log.Printf("📚 Step 1: Sending GetSupportedDM request to agent %s", agentID)
	s.sendGetSupportedDMRequest(agentID, version)

	// Wait between requests
	time.Sleep(1 * time.Second)

	// Step 2: GetInstances - discover object instances
	log.Printf("📊 Step 2: Sending GetInstances request to agent %s", agentID)
	s.sendGetInstancesRequest(agentID, version)

	// Wait between requests
	time.Sleep(1 * time.Second)

	// Step 3: Get Device Info - get basic device information
	log.Printf("📋 Step 3: Sending Get Device Info request to agent %s", agentID)
	s.sendGetDeviceInfoRequest(agentID, version)

	log.Printf("✅ Discovery sequence initiated for agent %s", agentID)
}

// sendGetSupportedDMRequest sends GetSupportedDM request
func (s *USPServiceServer) sendGetSupportedDMRequest(agentID, version string) {
	log.Printf("📚 Generating GetSupportedDM request for agent %s", agentID)

	// Get MTP client dynamically via connection manager
	mtpClient, err := s.connectionClient.GetMTPServiceClient()
	if err != nil {
		log.Printf("📋 Would send: GetSupportedDM request to discover supported data model (MTP client not available: %v)", err)
		return
	}

	// Generate USP GetSupportedDM message
	uspMessage, err := s.generateGetSupportedDMMessage(agentID, version)
	if err != nil {
		log.Printf("❌ Failed to generate GetSupportedDM message: %v", err)
		return
	}

	// Send via MTP Service
	req := &mtpservice.SendMessageRequest{
		AgentId:       agentID,
		UspMessage:    uspMessage,
		TransportType: "WebSocket",
		Metadata: map[string]string{
			"message_type":  "GetSupportedDM",
			"controller_id": getControllerID(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := mtpClient.SendMessageToAgent(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to send GetSupportedDM request to agent %s: %v", agentID, err)
		return
	}

	if resp.Success {
		log.Printf("✅ GetSupportedDM request sent to agent %s via %s", agentID, resp.TransportUsed)
	} else {
		log.Printf("❌ Failed to send GetSupportedDM request to agent %s: %s", agentID, resp.ErrorMessage)
	}
}

// generateGetSupportedDMMessage generates a USP GetSupportedDM message
func (s *USPServiceServer) generateGetSupportedDMMessage(agentID, version string) ([]byte, error) {
	messageID := fmt.Sprintf("GetSupportedDM-%d", time.Now().Unix())

	if version == "1.4" {
		// Create USP 1.4 GetSupportedDM message
		getSupportedDM := &v1_4.GetSupportedDM{
			ObjPaths:       []string{"Device."},
			FirstLevelOnly: false,
			ReturnCommands: true,
			ReturnEvents:   true,
			ReturnParams:   true,
		}

		request := &v1_4.Request{
			ReqType: &v1_4.Request_GetSupportedDm{
				GetSupportedDm: getSupportedDM,
			},
		}

		body := &v1_4.Body{
			MsgBody: &v1_4.Body_Request{
				Request: request,
			},
		}

		message := &v1_4.Msg{
			Header: &v1_4.Header{
				MsgId:   messageID,
				MsgType: v1_4.Header_GET_SUPPORTED_DM,
			},
			Body: body,
		}

		// Marshal the message first
		messageBytes, err := proto.Marshal(message)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal USP 1.4 message: %w", err)
		}

		record := &v1_4.Record{
			Version:         "1.4",
			ToId:            agentID,
			FromId:          getControllerID(),
			PayloadSecurity: v1_4.Record_PLAINTEXT,
			RecordType: &v1_4.Record_NoSessionContext{
				NoSessionContext: &v1_4.NoSessionContextRecord{
					Payload: messageBytes,
				},
			},
		}

		return proto.Marshal(record)

	} else {
		// Create USP 1.3 GetSupportedDM message
		getSupportedDM := &v1_3.GetSupportedDM{
			ObjPaths:       []string{"Device."},
			FirstLevelOnly: false,
			ReturnCommands: true,
			ReturnEvents:   true,
			ReturnParams:   true,
		}

		request := &v1_3.Request{
			ReqType: &v1_3.Request_GetSupportedDm{
				GetSupportedDm: getSupportedDM,
			},
		}

		body := &v1_3.Body{
			MsgBody: &v1_3.Body_Request{
				Request: request,
			},
		}

		message := &v1_3.Msg{
			Header: &v1_3.Header{
				MsgId:   messageID,
				MsgType: v1_3.Header_GET_SUPPORTED_DM,
			},
			Body: body,
		}

		// Marshal the message first
		messageBytes, err := proto.Marshal(message)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal USP 1.3 message: %w", err)
		}

		record := &v1_3.Record{
			Version:         "1.3",
			ToId:            agentID,
			FromId:          getControllerID(),
			PayloadSecurity: v1_3.Record_PLAINTEXT,
			RecordType: &v1_3.Record_NoSessionContext{
				NoSessionContext: &v1_3.NoSessionContextRecord{
					Payload: messageBytes,
				},
			},
		}

		return proto.Marshal(record)
	}
}

// sendGetInstancesRequest sends GetInstances request for Device root
func (s *USPServiceServer) sendGetInstancesRequest(agentID, version string) {
	log.Printf("📊 Generating GetInstances request for agent %s", agentID)

	// Get MTP client dynamically via connection manager
	mtpClient, err := s.connectionClient.GetMTPServiceClient()
	if err != nil {
		log.Printf("📋 Would send: GetInstances request for path='Device.' (Failed to get MTP client: %v)", err)
		return
	}

	// Generate USP GetInstances message
	uspMessage, err := s.generateGetInstancesMessage(agentID, version)
	if err != nil {
		log.Printf("❌ Failed to generate GetInstances message: %v", err)
		return
	}

	// Send via MTP Service
	req := &mtpservice.SendMessageRequest{
		AgentId:       agentID,
		UspMessage:    uspMessage,
		TransportType: "WebSocket",
		Metadata: map[string]string{
			"message_type":  "GetInstances",
			"controller_id": getControllerID(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := mtpClient.SendMessageToAgent(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to send GetInstances request to agent %s: %v", agentID, err)
		return
	}

	if resp.Success {
		log.Printf("✅ GetInstances request sent to agent %s via %s", agentID, resp.TransportUsed)
	} else {
		log.Printf("❌ Failed to send GetInstances request to agent %s: %s", agentID, resp.ErrorMessage)
	}
}

// generateGetInstancesMessage generates a USP GetInstances message
func (s *USPServiceServer) generateGetInstancesMessage(agentID, version string) ([]byte, error) {
	messageID := fmt.Sprintf("GetInstances-%d", time.Now().Unix())

	if version == "1.4" {
		// Create USP 1.4 GetInstances message
		getInstances := &v1_4.GetInstances{
			ObjPaths:       []string{"Device."},
			FirstLevelOnly: false,
		}

		request := &v1_4.Request{
			ReqType: &v1_4.Request_GetInstances{
				GetInstances: getInstances,
			},
		}

		body := &v1_4.Body{
			MsgBody: &v1_4.Body_Request{
				Request: request,
			},
		}

		message := &v1_4.Msg{
			Header: &v1_4.Header{
				MsgId:   messageID,
				MsgType: v1_4.Header_GET_INSTANCES,
			},
			Body: body,
		}

		// Marshal the message first
		messageBytes, err := proto.Marshal(message)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal USP 1.4 message: %w", err)
		}

		record := &v1_4.Record{
			Version:         "1.4",
			ToId:            agentID,
			FromId:          getControllerID(),
			PayloadSecurity: v1_4.Record_PLAINTEXT,
			RecordType: &v1_4.Record_NoSessionContext{
				NoSessionContext: &v1_4.NoSessionContextRecord{
					Payload: messageBytes,
				},
			},
		}

		return proto.Marshal(record)

	} else {
		// Create USP 1.3 GetInstances message
		getInstances := &v1_3.GetInstances{
			ObjPaths:       []string{"Device."},
			FirstLevelOnly: false,
		}

		request := &v1_3.Request{
			ReqType: &v1_3.Request_GetInstances{
				GetInstances: getInstances,
			},
		}

		body := &v1_3.Body{
			MsgBody: &v1_3.Body_Request{
				Request: request,
			},
		}

		message := &v1_3.Msg{
			Header: &v1_3.Header{
				MsgId:   messageID,
				MsgType: v1_3.Header_GET_INSTANCES,
			},
			Body: body,
		}

		// Marshal the message first
		messageBytes, err := proto.Marshal(message)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal USP 1.3 message: %w", err)
		}

		record := &v1_3.Record{
			Version:         "1.3",
			ToId:            agentID,
			FromId:          getControllerID(),
			PayloadSecurity: v1_3.Record_PLAINTEXT,
			RecordType: &v1_3.Record_NoSessionContext{
				NoSessionContext: &v1_3.NoSessionContextRecord{
					Payload: messageBytes,
				},
			},
		}

		return proto.Marshal(record)
	}
}

// sendGetDeviceInfoRequest sends Get request for device information
func (s *USPServiceServer) sendGetDeviceInfoRequest(agentID, version string) {
	log.Printf("📋 Generating Get Device Info request for agent %s", agentID)

	// Get MTP client dynamically via connection manager
	mtpClient, err := s.connectionClient.GetMTPServiceClient()
	if err != nil {
		log.Printf("📋 Would send: Get request for Device.DeviceInfo.Manufacturer, ModelName, SerialNumber, etc. (Failed to get MTP client: %v)", err)
		return
	}

	// Generate USP Get message for device info
	uspMessage, err := s.generateGetDeviceInfoMessage(agentID, version)
	if err != nil {
		log.Printf("❌ Failed to generate Get Device Info message: %v", err)
		return
	}

	// Send via MTP Service
	req := &mtpservice.SendMessageRequest{
		AgentId:       agentID,
		UspMessage:    uspMessage,
		TransportType: "WebSocket",
		Metadata: map[string]string{
			"message_type":  "Get",
			"controller_id": getControllerID(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := mtpClient.SendMessageToAgent(ctx, req)
	if err != nil {
		log.Printf("❌ Failed to send Get Device Info request to agent %s: %v", agentID, err)
		return
	}

	if resp.Success {
		log.Printf("✅ Get Device Info request sent to agent %s via %s", agentID, resp.TransportUsed)
	} else {
		log.Printf("❌ Failed to send Get Device Info request to agent %s: %s", agentID, resp.ErrorMessage)
	}
}

// generateGetDeviceInfoMessage generates a USP Get message for device information
func (s *USPServiceServer) generateGetDeviceInfoMessage(agentID, version string) ([]byte, error) {
	messageID := fmt.Sprintf("GetDeviceInfo-%d", time.Now().Unix())

	// Device info parameters to request
	deviceInfoParams := []string{
		"Device.DeviceInfo.Manufacturer",
		"Device.DeviceInfo.ModelName",
		"Device.DeviceInfo.SerialNumber",
		"Device.DeviceInfo.SoftwareVersion",
		"Device.DeviceInfo.HardwareVersion",
		"Device.DeviceInfo.ProductClass",
	}

	if version == "1.4" {
		// Create USP 1.4 Get message
		get := &v1_4.Get{
			ParamPaths: deviceInfoParams,
		}

		request := &v1_4.Request{
			ReqType: &v1_4.Request_Get{
				Get: get,
			},
		}

		body := &v1_4.Body{
			MsgBody: &v1_4.Body_Request{
				Request: request,
			},
		}

		message := &v1_4.Msg{
			Header: &v1_4.Header{
				MsgId:   messageID,
				MsgType: v1_4.Header_GET,
			},
			Body: body,
		}

		// Marshal the message first
		messageBytes, err := proto.Marshal(message)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal USP 1.4 message: %w", err)
		}

		record := &v1_4.Record{
			Version:         "1.4",
			ToId:            agentID,
			FromId:          getControllerID(),
			PayloadSecurity: v1_4.Record_PLAINTEXT,
			RecordType: &v1_4.Record_NoSessionContext{
				NoSessionContext: &v1_4.NoSessionContextRecord{
					Payload: messageBytes,
				},
			},
		}

		return proto.Marshal(record)

	} else {
		// Create USP 1.3 Get message
		get := &v1_3.Get{
			ParamPaths: deviceInfoParams,
		}

		request := &v1_3.Request{
			ReqType: &v1_3.Request_Get{
				Get: get,
			},
		}

		body := &v1_3.Body{
			MsgBody: &v1_3.Body_Request{
				Request: request,
			},
		}

		message := &v1_3.Msg{
			Header: &v1_3.Header{
				MsgId:   messageID,
				MsgType: v1_3.Header_GET,
			},
			Body: body,
		}

		// Marshal the message first
		messageBytes, err := proto.Marshal(message)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal USP 1.3 message: %w", err)
		}

		record := &v1_3.Record{
			Version:         "1.3",
			ToId:            agentID,
			FromId:          getControllerID(),
			PayloadSecurity: v1_3.Record_PLAINTEXT,
			RecordType: &v1_3.Record_NoSessionContext{
				NoSessionContext: &v1_3.NoSessionContextRecord{
					Payload: messageBytes,
				},
			},
		}

		return proto.Marshal(record)
	}
}

// isNewAgent checks if this is the first time we're seeing this agent
func (s *USPServiceServer) isNewAgent(agentID string) bool {
	// Check if agent exists in database
	getReq := &dataservice.GetDeviceByEndpointRequest{
		EndpointId: agentID,
	}

	_, err := s.dataClient.GetDeviceByEndpoint(context.Background(), getReq)
	if err != nil {
		// Agent not found in database - it's new
		log.Printf("🔍 Agent %s not found in database - treating as new agent", agentID)
		return true
	}

	// Agent exists - not new
	return false
}

// generateConnectAcknowledgment generates appropriate connect acknowledgment responses
func (s *USPServiceServer) generateConnectAcknowledgment(parsed *usp.ParsedUSP) ([]byte, error) {
	recordType := parsed.Record.RecordType
	agentID := parsed.Record.FromID
	controllerID := parsed.Record.ToID

	log.Printf("🔌 Generating %s acknowledgment for agent %s", recordType, agentID)

	if parsed.Record.Version == usp.Version14 {
		return s.generateConnectAck14(recordType, agentID, controllerID)
	} else if parsed.Record.Version == usp.Version13 {
		return s.generateConnectAck13(recordType, agentID, controllerID)
	}

	return nil, fmt.Errorf("unsupported USP version for connect ack: %s", string(parsed.Record.Version))
}

// generateConnectAck14 generates USP 1.4 connect acknowledgments
func (s *USPServiceServer) generateConnectAck14(recordType, agentID, controllerID string) ([]byte, error) {
	switch recordType {
	case "WebSocketConnect":
		// Create WebSocketConnect acknowledgment record
		record := &v1_4.Record{
			Version: "1.4",
			ToId:    agentID,      // Send to agent
			FromId:  controllerID, // From controller
			RecordType: &v1_4.Record_WebsocketConnect{
				WebsocketConnect: &v1_4.WebSocketConnectRecord{},
			},
		}

		return proto.Marshal(record)

	case "MQTTConnect":
		// Create MQTTConnect acknowledgment record
		record := &v1_4.Record{
			Version: "1.4",
			ToId:    agentID,
			FromId:  controllerID,
			RecordType: &v1_4.Record_MqttConnect{
				MqttConnect: &v1_4.MQTTConnectRecord{},
			},
		}

		return proto.Marshal(record)

	case "StompConnect":
		// Create StompConnect acknowledgment record
		record := &v1_4.Record{
			Version: "1.4",
			ToId:    agentID,
			FromId:  controllerID,
			RecordType: &v1_4.Record_StompConnect{
				StompConnect: &v1_4.STOMPConnectRecord{
					Version:               v1_4.STOMPConnectRecord_V1_2,
					SubscribedDestination: "/queue/usp.controller", // Where controller listens for agent messages
				},
			},
		}

		return proto.Marshal(record)

	default:
		return nil, fmt.Errorf("unsupported connect record type for USP 1.4: %s", recordType)
	}
}

// generateConnectAck13 generates USP 1.3 connect acknowledgments
func (s *USPServiceServer) generateConnectAck13(recordType, agentID, controllerID string) ([]byte, error) {
	switch recordType {
	case "WebSocketConnect":
		// Create WebSocketConnect acknowledgment record
		record := &v1_3.Record{
			Version: "1.3",
			ToId:    agentID,
			FromId:  controllerID,
			RecordType: &v1_3.Record_WebsocketConnect{
				WebsocketConnect: &v1_3.WebSocketConnectRecord{},
			},
		}

		return proto.Marshal(record)

	case "StompConnect":
		// Create StompConnect acknowledgment record
		record := &v1_3.Record{
			Version: "1.3",
			ToId:    agentID,
			FromId:  controllerID,
			RecordType: &v1_3.Record_StompConnect{
				StompConnect: &v1_3.STOMPConnectRecord{
					Version:               v1_3.STOMPConnectRecord_V1_2,
					SubscribedDestination: "/queue/usp.controller", // Where controller listens for agent messages
				},
			},
		}

		return proto.Marshal(record)

	default:
		return nil, fmt.Errorf("unsupported connect record type for USP 1.3: %s", recordType)
	}
}
