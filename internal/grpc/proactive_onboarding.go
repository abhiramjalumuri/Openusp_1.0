// Package grpc provides proactive device onboarding for TR-369 compliance
package grpc

import (
	"fmt"
	"log"
	"time"

	"openusp/pkg/proto/uspservice"
	v1_3 "openusp/pkg/proto/v1_3"
	v1_4 "openusp/pkg/proto/v1_4"
	"openusp/pkg/service/client"

	"google.golang.org/protobuf/proto"
)

// ProactiveOnboardingManager handles proactive device onboarding
type ProactiveOnboardingManager struct {
	uspServer         *USPServiceServer
	connectionClient  *client.OpenUSPConnectionClient
	pendingAgents     map[string]*PendingAgent
	bootSubscriptions map[string]*BootSubscription
	responseWaiters   map[string]*ResponseWaiter // Track response waiters by agentID
}

// PendingAgent represents an agent pending onboarding
type PendingAgent struct {
	EndpointID        string
	FirstSeen         time.Time
	LastActivity      time.Time
	ConnectionContext map[string]interface{}
	OnboardingStatus  string
	AttemptCount      int
	Version           string
}

// BootSubscription represents a Boot! event subscription
type BootSubscription struct {
	AgentID        string
	SubscriptionID string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	Active         bool
}

// ResponseWaiter tracks waiting for device information responses
type ResponseWaiter struct {
	AgentID       string
	DeviceInfo    *uspservice.DeviceInfo
	InfoChan      chan *uspservice.DeviceInfo
	CreatedAt     time.Time
	PendingParams map[string]bool // Track which parameters we're waiting for
}

// NewProactiveOnboardingManager creates a new proactive onboarding manager
func NewProactiveOnboardingManager(uspServer *USPServiceServer, connectionClient *client.OpenUSPConnectionClient) *ProactiveOnboardingManager {
	return &ProactiveOnboardingManager{
		uspServer:         uspServer,
		connectionClient:  connectionClient,
		pendingAgents:     make(map[string]*PendingAgent),
		bootSubscriptions: make(map[string]*BootSubscription),
		responseWaiters:   make(map[string]*ResponseWaiter),
	}
}

// HandleWebSocketConnect processes new WebSocket connections for proactive onboarding
func (pom *ProactiveOnboardingManager) HandleWebSocketConnect(endpointID, version string, connCtx map[string]interface{}) {
	log.Printf("🔍 ProactiveOnboarding: New WebSocket connection detected from %s (USP %s)", endpointID, version)

	// Check if this agent is already being processed
	if existing, exists := pom.pendingAgents[endpointID]; exists {
		// Allow reprocessing of completed agents for testing/reconnection scenarios
		if existing.OnboardingStatus == "completed" {
			log.Printf("🔄 ProactiveOnboarding: Agent %s reconnected after completion - restarting onboarding", endpointID)
			delete(pom.pendingAgents, endpointID)
		} else {
			log.Printf("📋 ProactiveOnboarding: Agent %s already being processed (status: %s)", endpointID, existing.OnboardingStatus)
			existing.LastActivity = time.Now()
			existing.AttemptCount++
			return
		}
	}

	// Create new pending agent entry
	pending := &PendingAgent{
		EndpointID:        endpointID,
		FirstSeen:         time.Now(),
		LastActivity:      time.Now(),
		ConnectionContext: connCtx,
		OnboardingStatus:  "websocket_connected",
		AttemptCount:      1,
		Version:           version,
	}

	pom.pendingAgents[endpointID] = pending

	// Start proactive onboarding sequence in background
	go pom.StartProactiveOnboarding(endpointID, version)
}

// StartProactiveOnboarding implements the proactive TR-369 onboarding sequence
func (pom *ProactiveOnboardingManager) StartProactiveOnboarding(agentID, version string) {
	log.Printf("🚀 ProactiveOnboarding: Starting proactive onboarding for agent %s (USP %s)", agentID, version)

	pending := pom.pendingAgents[agentID]
	if pending == nil {
		log.Printf("❌ ProactiveOnboarding: Agent %s not found in pending list", agentID)
		return
	}

	// Step 1: Wait for agent stabilization (WebSocket handshake completion)
	log.Printf("⏳ Step 1: Waiting for agent %s to stabilize after WebSocket connection", agentID)
	time.Sleep(2 * time.Second)
	pending.OnboardingStatus = "stabilizing"

	// Step 2: Send GetSupportedDM to trigger agent response and discover capabilities
	log.Printf("📚 Step 2: Sending GetSupportedDM to trigger agent %s discovery", agentID)
	err := pom.sendGetSupportedDMRequest(agentID, version)
	if err != nil {
		log.Printf("❌ ProactiveOnboarding: Failed to send GetSupportedDM to agent %s: %v", agentID, err)
		pending.OnboardingStatus = "failed_discovery"
		return
	}
	pending.OnboardingStatus = "discovery_sent"

	// Step 3: Wait for response and then subscribe to Boot! event
	log.Printf("⏳ Step 3: Waiting for GetSupportedDM response from agent %s", agentID)
	time.Sleep(3 * time.Second)

	log.Printf("📡 Step 4: Subscribing to Boot! event for agent %s", agentID)
	err = pom.subscribeToBootEvent(agentID, version)
	if err != nil {
		log.Printf("❌ ProactiveOnboarding: Failed to subscribe to Boot! event for agent %s: %v", agentID, err)
		pending.OnboardingStatus = "failed_boot_subscription"
		// Continue anyway - some agents may not support subscriptions
	} else {
		pending.OnboardingStatus = "boot_subscribed"
	}

	// Step 5: Request device information via Get requests
	log.Printf("🔍 Step 5: Requesting device information from agent %s", agentID)
	err = pom.requestDeviceInformation(agentID, version)
	if err != nil {
		log.Printf("❌ ProactiveOnboarding: Failed to request device info from agent %s: %v", agentID, err)
		pending.OnboardingStatus = "failed_device_info"
		return
	}
	pending.OnboardingStatus = "device_info_requested"

	// Step 6: Wait for device information responses (TR-369 compliant)
	log.Printf("⏳ Step 6: Waiting for device information responses from agent %s", agentID)
	pending.OnboardingStatus = "awaiting_device_info"

	// Wait for responses with timeout
	deviceInfo := pom.waitForDeviceInfoResponses(agentID, 30*time.Second)
	if deviceInfo == nil {
		log.Printf("⚠️ ProactiveOnboarding: Timeout waiting for device info from agent %s - creating entry with defaults", agentID)
		// Create entry with defaults but mark as incomplete
		err = pom.createDeviceEntryWithDefaults(agentID, version, true)
	} else {
		log.Printf("✅ ProactiveOnboarding: Received device information from agent %s", agentID)
		log.Printf("   📝 Manufacturer: %s", deviceInfo.Manufacturer)
		log.Printf("   📝 Product Class: %s", deviceInfo.ProductClass)
		log.Printf("   📝 Serial Number: %s", deviceInfo.SerialNumber)

		// Step 7: Create device entry with actual discovered data
		log.Printf("💾 Step 7: Creating device entry with discovered data for agent %s", agentID)
		err = pom.createDeviceEntryWithDiscoveredData(agentID, version, deviceInfo)
	}

	if err != nil {
		log.Printf("❌ ProactiveOnboarding: Failed to create device entry for agent %s: %v", agentID, err)
		pending.OnboardingStatus = "failed_device_creation"
		return
	}

	pending.OnboardingStatus = "completed"
	log.Printf("✅ ProactiveOnboarding: Completed proactive onboarding for agent %s", agentID)

	// Clean up after successful onboarding
	time.AfterFunc(5*time.Minute, func() {
		delete(pom.pendingAgents, agentID)
		log.Printf("🧹 ProactiveOnboarding: Cleaned up completed onboarding for agent %s", agentID)
	})
}

// sendGetSupportedDMRequest sends GetSupportedDM request to trigger agent discovery
func (pom *ProactiveOnboardingManager) sendGetSupportedDMRequest(agentID, version string) error {
	log.Printf("📚 ProactiveOnboarding: Sending GetSupportedDM request to agent %s", agentID)

	// Create GetSupportedDM request based on USP version
	var requestData []byte
	var err error

	if version == "1.4" {
		requestData, err = pom.createGetSupportedDM14Request(agentID)
	} else {
		requestData, err = pom.createGetSupportedDM13Request(agentID)
	}

	if err != nil {
		return fmt.Errorf("failed to create GetSupportedDM request: %w", err)
	}

	// Send via MTP service
	return pom.sendMessageToAgent(agentID, requestData)
}

// createGetSupportedDM14Request creates USP 1.4 GetSupportedDM request
func (pom *ProactiveOnboardingManager) createGetSupportedDM14Request(agentID string) ([]byte, error) {
	// Create USP 1.4 GetSupportedDM request
	msg := &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   fmt.Sprintf("openusp-discovery-%d", time.Now().Unix()),
			MsgType: v1_4.Header_GET_SUPPORTED_DM,
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Request{
				Request: &v1_4.Request{
					ReqType: &v1_4.Request_GetSupportedDm{
						GetSupportedDm: &v1_4.GetSupportedDM{
							ObjPaths:       []string{"Device."},
							ReturnCommands: true,
							ReturnEvents:   true,
							ReturnParams:   true,
							FirstLevelOnly: false,
						},
					},
				},
			},
		},
	}

	// Marshal message
	msgData, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.4 message: %w", err)
	}

	// Create record
	record := &v1_4.Record{
		Version:         "1.4",
		ToId:            agentID,
		FromId:          "openusp-controller",
		PayloadSecurity: v1_4.Record_PLAINTEXT,
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: msgData,
			},
		},
	}

	return proto.Marshal(record)
}

// createGetSupportedDM13Request creates USP 1.3 GetSupportedDM request
func (pom *ProactiveOnboardingManager) createGetSupportedDM13Request(agentID string) ([]byte, error) {
	// Create USP 1.3 GetSupportedDM request
	msg := &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   fmt.Sprintf("openusp-discovery-%d", time.Now().Unix()),
			MsgType: v1_3.Header_GET_SUPPORTED_DM,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Request{
				Request: &v1_3.Request{
					ReqType: &v1_3.Request_GetSupportedDm{
						GetSupportedDm: &v1_3.GetSupportedDM{
							ObjPaths:       []string{"Device."},
							ReturnCommands: true,
							ReturnEvents:   true,
							ReturnParams:   true,
							FirstLevelOnly: false,
						},
					},
				},
			},
		},
	}

	// Marshal message
	msgData, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.3 message: %w", err)
	}

	// Create record
	record := &v1_3.Record{
		Version:         "1.3",
		ToId:            agentID,
		FromId:          "openusp-controller",
		PayloadSecurity: v1_3.Record_PLAINTEXT,
		RecordType: &v1_3.Record_NoSessionContext{
			NoSessionContext: &v1_3.NoSessionContextRecord{
				Payload: msgData,
			},
		},
	}

	return proto.Marshal(record)
}

// subscribeToBootEvent creates Boot! event subscription
func (pom *ProactiveOnboardingManager) subscribeToBootEvent(agentID, version string) error {
	log.Printf("📡 ProactiveOnboarding: Creating Boot! event subscription for agent %s", agentID)

	// Create ADD request to establish Boot! event subscription
	var requestData []byte
	var err error

	if version == "1.4" {
		requestData, err = pom.createBootSubscription14Request(agentID)
	} else {
		requestData, err = pom.createBootSubscription13Request(agentID)
	}

	if err != nil {
		return fmt.Errorf("failed to create Boot! subscription request: %w", err)
	}

	// Send via MTP service
	err = pom.sendMessageToAgent(agentID, requestData)
	if err != nil {
		return fmt.Errorf("failed to send Boot! subscription: %w", err)
	}

	// Track subscription
	subscriptionID := fmt.Sprintf("boot-sub-%s-%d", agentID, time.Now().Unix())
	pom.bootSubscriptions[agentID] = &BootSubscription{
		AgentID:        agentID,
		SubscriptionID: subscriptionID,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Active:         true,
	}

	return nil
}

// createBootSubscription14Request creates USP 1.4 Boot! subscription request
func (pom *ProactiveOnboardingManager) createBootSubscription14Request(agentID string) ([]byte, error) {
	// Create USP 1.4 ADD request for Boot! subscription
	msg := &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   fmt.Sprintf("openusp-boot-sub-%d", time.Now().Unix()),
			MsgType: v1_4.Header_ADD,
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Request{
				Request: &v1_4.Request{
					ReqType: &v1_4.Request_Add{
						Add: &v1_4.Add{
							AllowPartial: false,
							CreateObjs: []*v1_4.Add_CreateObject{
								{
									ObjPath: "Device.LocalAgent.Subscription.",
									ParamSettings: []*v1_4.Add_CreateParamSetting{
										{
											Param: "Enable",
											Value: "true",
										},
										{
											Param: "ID",
											Value: fmt.Sprintf("boot-subscription-%d", time.Now().Unix()),
										},
										{
											Param: "NotifType",
											Value: "Event",
										},
										{
											Param: "ReferenceList",
											Value: "Device.Boot!",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Marshal message
	msgData, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.4 Boot subscription: %w", err)
	}

	// Create record
	record := &v1_4.Record{
		Version:         "1.4",
		ToId:            agentID,
		FromId:          "openusp-controller",
		PayloadSecurity: v1_4.Record_PLAINTEXT,
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: msgData,
			},
		},
	}

	return proto.Marshal(record)
}

// createBootSubscription13Request creates USP 1.3 Boot! subscription request
func (pom *ProactiveOnboardingManager) createBootSubscription13Request(agentID string) ([]byte, error) {
	// Create USP 1.3 ADD request for Boot! subscription
	msg := &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   fmt.Sprintf("openusp-boot-sub-%d", time.Now().Unix()),
			MsgType: v1_3.Header_ADD,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Request{
				Request: &v1_3.Request{
					ReqType: &v1_3.Request_Add{
						Add: &v1_3.Add{
							AllowPartial: false,
							CreateObjs: []*v1_3.Add_CreateObject{
								{
									ObjPath: "Device.LocalAgent.Subscription.",
									ParamSettings: []*v1_3.Add_CreateParamSetting{
										{
											Param: "Enable",
											Value: "true",
										},
										{
											Param: "ID",
											Value: fmt.Sprintf("boot-subscription-%d", time.Now().Unix()),
										},
										{
											Param: "NotifType",
											Value: "Event",
										},
										{
											Param: "ReferenceList",
											Value: "Device.Boot!",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Marshal message
	msgData, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.3 Boot subscription: %w", err)
	}

	// Create record
	record := &v1_3.Record{
		Version:         "1.3",
		ToId:            agentID,
		FromId:          "openusp-controller",
		PayloadSecurity: v1_3.Record_PLAINTEXT,
		RecordType: &v1_3.Record_NoSessionContext{
			NoSessionContext: &v1_3.NoSessionContextRecord{
				Payload: msgData,
			},
		},
	}

	return proto.Marshal(record)
}

// requestDeviceInformation requests device info via Get requests
func (pom *ProactiveOnboardingManager) requestDeviceInformation(agentID, version string) error {
	log.Printf("🔍 ProactiveOnboarding: Requesting device information from agent %s", agentID)

	// Request key device information parameters
	devicePaths := []string{
		"Device.DeviceInfo.Manufacturer",
		"Device.DeviceInfo.ProductClass",
		"Device.DeviceInfo.SerialNumber",
		"Device.DeviceInfo.HardwareVersion",
		"Device.DeviceInfo.SoftwareVersion",
		"Device.LocalAgent.EndpointID",
	}

	var requestData []byte
	var err error

	if version == "1.4" {
		requestData, err = pom.createGetRequest14(agentID, devicePaths)
	} else {
		requestData, err = pom.createGetRequest13(agentID, devicePaths)
	}

	if err != nil {
		return fmt.Errorf("failed to create Get request: %w", err)
	}

	return pom.sendMessageToAgent(agentID, requestData)
}

// createGetRequest14 creates USP 1.4 Get request
func (pom *ProactiveOnboardingManager) createGetRequest14(agentID string, paths []string) ([]byte, error) {
	msg := &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   fmt.Sprintf("openusp-get-info-%d", time.Now().Unix()),
			MsgType: v1_4.Header_GET,
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Request{
				Request: &v1_4.Request{
					ReqType: &v1_4.Request_Get{
						Get: &v1_4.Get{
							ParamPaths: paths,
						},
					},
				},
			},
		},
	}

	msgData, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.4 Get request: %w", err)
	}

	record := &v1_4.Record{
		Version:         "1.4",
		ToId:            agentID,
		FromId:          "openusp-controller",
		PayloadSecurity: v1_4.Record_PLAINTEXT,
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: msgData,
			},
		},
	}

	return proto.Marshal(record)
}

// createGetRequest13 creates USP 1.3 Get request
func (pom *ProactiveOnboardingManager) createGetRequest13(agentID string, paths []string) ([]byte, error) {
	msg := &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   fmt.Sprintf("openusp-get-info-%d", time.Now().Unix()),
			MsgType: v1_3.Header_GET,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Request{
				Request: &v1_3.Request{
					ReqType: &v1_3.Request_Get{
						Get: &v1_3.Get{
							ParamPaths: paths,
						},
					},
				},
			},
		},
	}

	msgData, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.3 Get request: %w", err)
	}

	record := &v1_3.Record{
		Version:         "1.3",
		ToId:            agentID,
		FromId:          "openusp-controller",
		PayloadSecurity: v1_3.Record_PLAINTEXT,
		RecordType: &v1_3.Record_NoSessionContext{
			NoSessionContext: &v1_3.NoSessionContextRecord{
				Payload: msgData,
			},
		},
	}

	return proto.Marshal(record)
}

// waitForDeviceInfoResponses waits for device information responses from agent
func (pom *ProactiveOnboardingManager) waitForDeviceInfoResponses(agentID string, timeout time.Duration) *uspservice.DeviceInfo {
	log.Printf("⏳ ProactiveOnboarding: Waiting for device info responses from agent %s (timeout: %v)", agentID, timeout)

	// Create a response waiter
	infoChan := make(chan *uspservice.DeviceInfo, 1)
	waiter := &ResponseWaiter{
		AgentID:    agentID,
		DeviceInfo: &uspservice.DeviceInfo{EndpointId: agentID},
		InfoChan:   infoChan,
		CreatedAt:  time.Now(),
		PendingParams: map[string]bool{
			"Device.DeviceInfo.Manufacturer":    true,
			"Device.DeviceInfo.ProductClass":    true,
			"Device.DeviceInfo.SerialNumber":    true,
			"Device.DeviceInfo.ModelName":       true,
			"Device.DeviceInfo.SoftwareVersion": true,
		},
	}

	// Register the waiter
	pom.responseWaiters[agentID] = waiter

	// Set up a timeout
	timeoutChan := time.After(timeout)

	log.Printf("📡 ProactiveOnboarding: Response waiter registered for agent %s, waiting for %d parameters",
		agentID, len(waiter.PendingParams))

	// Wait for responses or timeout
	select {
	case deviceInfo := <-infoChan:
		log.Printf("✅ ProactiveOnboarding: Received complete device info for agent %s", agentID)
		// Clean up the waiter
		delete(pom.responseWaiters, agentID)
		return deviceInfo
	case <-timeoutChan:
		log.Printf("⏰ ProactiveOnboarding: Timeout waiting for device info from agent %s", agentID)
		// Clean up the waiter
		delete(pom.responseWaiters, agentID)
		return nil
	}
}

// HandleGetResponse processes Get responses from agents during proactive onboarding
func (pom *ProactiveOnboardingManager) HandleGetResponse(agentID string, paramPath string, paramValue string) {
	// Check if we have a response waiter for this agent
	waiter, exists := pom.responseWaiters[agentID]
	if !exists {
		log.Printf("🔍 ProactiveOnboarding: No waiter found for agent %s, ignoring Get response for %s", agentID, paramPath)
		return
	}

	log.Printf("📨 ProactiveOnboarding: Received Get response from agent %s for %s = %s", agentID, paramPath, paramValue)

	// Update device information based on parameter path
	switch paramPath {
	case "Device.DeviceInfo.Manufacturer":
		waiter.DeviceInfo.Manufacturer = paramValue
		delete(waiter.PendingParams, paramPath)
	case "Device.DeviceInfo.ProductClass":
		waiter.DeviceInfo.ProductClass = paramValue
		delete(waiter.PendingParams, paramPath)
	case "Device.DeviceInfo.SerialNumber":
		waiter.DeviceInfo.SerialNumber = paramValue
		delete(waiter.PendingParams, paramPath)
	case "Device.DeviceInfo.ModelName":
		waiter.DeviceInfo.ModelName = paramValue
		delete(waiter.PendingParams, paramPath)
	case "Device.DeviceInfo.SoftwareVersion":
		waiter.DeviceInfo.SoftwareVersion = paramValue
		delete(waiter.PendingParams, paramPath)
	default:
		log.Printf("🔍 ProactiveOnboarding: Unknown parameter path %s from agent %s", paramPath, agentID)
		return
	}

	log.Printf("📊 ProactiveOnboarding: Agent %s now has %d pending parameters remaining",
		agentID, len(waiter.PendingParams))

	// Check if all parameters have been received
	if len(waiter.PendingParams) == 0 {
		log.Printf("✅ ProactiveOnboarding: All device info collected for agent %s", agentID)
		// Send the complete device info
		select {
		case waiter.InfoChan <- waiter.DeviceInfo:
			log.Printf("📤 ProactiveOnboarding: Device info sent to waiter for agent %s", agentID)
		default:
			log.Printf("⚠️ ProactiveOnboarding: InfoChan full for agent %s", agentID)
		}
	}
}

// createDeviceEntryWithDefaults creates device entry with default values
func (pom *ProactiveOnboardingManager) createDeviceEntryWithDefaults(agentID, version string, incomplete bool) error {
	log.Printf("💾 ProactiveOnboarding: Creating device entry with defaults for agent %s (incomplete: %t)", agentID, incomplete)

	status := "online"
	if incomplete {
		status = "pending_discovery" // Mark as needing more information
	}

	// Create device info with default/placeholder information
	deviceInfo := &uspservice.DeviceInfo{
		EndpointId:                agentID,
		ProductClass:              "USP Agent (Unknown)",
		SerialNumber:              fmt.Sprintf("PROACTIVE-%d", time.Now().Unix()),
		Manufacturer:              "Unknown - Discovery Pending",
		ModelName:                 "Proactive Discovery Agent",
		SoftwareVersion:           "Unknown",
		HardwareVersion:           "Unknown",
		SupportedProtocolVersions: version,
	}

	// Call USP service to create device with defaults
	if pom.uspServer != nil {
		log.Printf("🔄 ProactiveOnboarding: Creating device with defaults for agent %s", agentID)
		err := pom.uspServer.createProactiveDeviceWithStatus(deviceInfo, status)
		if err != nil {
			log.Printf("❌ ProactiveOnboarding: Failed to create device with defaults: %v", err)
			return fmt.Errorf("failed to create device with defaults: %w", err)
		}
	} else {
		log.Printf("⚠️ ProactiveOnboarding: USP service not available")
		return fmt.Errorf("USP service not available")
	}

	log.Printf("✅ ProactiveOnboarding: Device entry with defaults created for agent %s", agentID)
	return nil
}

// createDeviceEntryWithDiscoveredData creates device entry with actual discovered data
func (pom *ProactiveOnboardingManager) createDeviceEntryWithDiscoveredData(agentID, version string, deviceInfo *uspservice.DeviceInfo) error {
	log.Printf("💾 ProactiveOnboarding: Creating device entry with discovered data for agent %s", agentID)

	// Ensure the endpoint ID matches
	deviceInfo.EndpointId = agentID
	deviceInfo.SupportedProtocolVersions = version

	// Call USP service to create device with discovered data
	if pom.uspServer != nil {
		log.Printf("🔄 ProactiveOnboarding: Creating device with discovered data for agent %s", agentID)
		err := pom.uspServer.createProactiveDeviceWithStatus(deviceInfo, "online")
		if err != nil {
			log.Printf("❌ ProactiveOnboarding: Failed to create device with discovered data: %v", err)
			return fmt.Errorf("failed to create device with discovered data: %w", err)
		}
	} else {
		log.Printf("⚠️ ProactiveOnboarding: USP service not available")
		return fmt.Errorf("USP service not available")
	}

	log.Printf("✅ ProactiveOnboarding: Device entry with discovered data created for agent %s", agentID)
	return nil
}

// sendMessageToAgent sends a message to an agent via connection manager
func (pom *ProactiveOnboardingManager) sendMessageToAgent(agentID string, messageData []byte) error {
	if pom.connectionClient == nil {
		log.Printf("⚠️ ProactiveOnboarding: Connection client not available - cannot send message to agent %s", agentID)
		return fmt.Errorf("connection client not available")
	}

	log.Printf("📤 ProactiveOnboarding: Sending message to agent %s (would use MTP service via connection manager)", agentID)

	// For now, just log the intent - actual implementation would use the connection manager
	// to route the message through the appropriate MTP service
	log.Printf("   Message size: %d bytes", len(messageData))
	log.Printf("   This would be sent via WebSocket MTP to agent %s", agentID)

	return nil
}

// GetOnboardingStatus returns the current onboarding status
func (pom *ProactiveOnboardingManager) GetOnboardingStatus() map[string]*PendingAgent {
	result := make(map[string]*PendingAgent)
	for k, v := range pom.pendingAgents {
		result[k] = v
	}
	return result
}

// CleanupExpiredAgents removes expired pending agents
func (pom *ProactiveOnboardingManager) CleanupExpiredAgents() {
	expiry := time.Now().Add(-1 * time.Hour)

	for agentID, pending := range pom.pendingAgents {
		if pending.LastActivity.Before(expiry) {
			log.Printf("🧹 ProactiveOnboarding: Cleaning up expired pending agent %s", agentID)
			delete(pom.pendingAgents, agentID)
		}
	}

	// Also cleanup expired boot subscriptions
	for agentID, subscription := range pom.bootSubscriptions {
		if subscription.ExpiresAt.Before(time.Now()) {
			log.Printf("🧹 ProactiveOnboarding: Cleaning up expired Boot! subscription for agent %s", agentID)
			delete(pom.bootSubscriptions, agentID)
		}
	}
}
