package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	confluentkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"google.golang.org/protobuf/proto"

	"openusp/internal/tr181"
	"openusp/pkg/config"
	"openusp/pkg/kafka"
	"openusp/pkg/metrics"
	v1_3 "openusp/pkg/proto/v1_3"
	v1_4 "openusp/pkg/proto/v1_4"
	"openusp/pkg/version"
)

type USPCoreService struct {
	deviceManager *tr181.DeviceManager
	kafkaClient   *kafka.Client
	kafkaConsumer *kafka.Consumer
	kafkaProducer *kafka.Producer
	config        *config.Config
	httpServer    *http.Server
	metrics       *metrics.OpenUSPMetrics
}

func NewUSPCoreService(cfg *config.Config) (*USPCoreService, error) {
	dm, err := tr181.LoadDefaultDataModel()
	if err != nil {
		return nil, fmt.Errorf("failed to load TR-181 data model: %w", err)
	}

	// Initialize Kafka client
	kafkaClient, err := kafka.NewClient(&cfg.Kafka)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka client: %w", err)
	}

	// Ensure Kafka topics exist
	topicsManager := kafka.NewTopicsManager(kafkaClient, &cfg.Kafka.Topics)
	if err := topicsManager.EnsureAllTopicsExist(); err != nil {
		return nil, fmt.Errorf("failed to create Kafka topics: %w", err)
	}
	log.Printf("✅ Kafka topics validated/created")

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(kafkaClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}

	// Initialize Kafka consumer
	kafkaConsumer, err := kafka.NewConsumer(&cfg.Kafka, cfg.USPService.ConsumerGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka consumer: %w", err)
	}

	return &USPCoreService{
		deviceManager: dm,
		kafkaClient:   kafkaClient,
		kafkaConsumer: kafkaConsumer,
		kafkaProducer: kafkaProducer,
		config:        cfg,
		metrics:       metrics.NewOpenUSPMetrics("usp-service"),
	}, nil
}

// setupKafkaConsumers configures Kafka subscriptions and message handlers
func (s *USPCoreService) setupKafkaConsumers() error {
	// Subscribe to all relevant topics for duplex communication
	topics := []string{
		s.config.Kafka.Topics.USPMessagesInbound,       // From MTP services (STOMP, MQTT, WebSocket)
		s.config.Kafka.Topics.USPAPIRequest,            // From API Gateway
		s.config.Kafka.Topics.USPDataRequest,           // From Data Service
		s.config.Kafka.Topics.DataDeviceCreated,        // Data events from Data Service
		s.config.Kafka.Topics.DataDeviceUpdated,        // Data events from Data Service
		s.config.Kafka.Topics.DataParameterUpdated,     // Data events from Data Service
		s.config.Kafka.Topics.MTPConnectionEstablished, // MTP connection events
		s.config.Kafka.Topics.MTPConnectionClosed,      // MTP connection events
	}

	// Define message handler
	handler := func(msg *confluentkafka.Message) error {
		topic := *msg.TopicPartition.Topic
		log.Printf("📨 Received message on topic %s: %d bytes", topic, len(msg.Value))

		// Route to appropriate handler based on topic
		switch topic {
		case s.config.Kafka.Topics.USPMessagesInbound:
			return s.handleUSPMessage(msg)
		case s.config.Kafka.Topics.USPAPIRequest:
			return s.handleAPIRequest(msg)
		case s.config.Kafka.Topics.USPDataRequest:
			return s.handleDataRequest(msg)
		case s.config.Kafka.Topics.DataDeviceCreated,
			s.config.Kafka.Topics.DataDeviceUpdated,
			s.config.Kafka.Topics.DataParameterUpdated:
			return s.handleDataEvent(msg)
		case s.config.Kafka.Topics.MTPConnectionEstablished,
			s.config.Kafka.Topics.MTPConnectionClosed:
			return s.handleMTPEvent(msg)
		default:
			log.Printf("⚠️  Unknown topic: %s", topic)
			return nil
		}
	}

	// Subscribe to topics
	if err := s.kafkaConsumer.Subscribe(topics, handler); err != nil {
		return fmt.Errorf("failed to subscribe to Kafka topics: %w", err)
	}

	log.Printf("✅ Subscribed to %d Kafka topics for duplex communication", len(topics))
	for _, topic := range topics {
		log.Printf("   └── %s", topic)
	}
	return nil
}

// handleUSPMessage processes incoming USP messages from MTP services
func (s *USPCoreService) handleUSPMessage(msg *confluentkafka.Message) error {
	log.Printf("📨 Received message from topic: %s (%d bytes)", *msg.TopicPartition.Topic, len(msg.Value))

	// Unmarshal Kafka USPMessageEvent envelope
	var event kafka.USPMessageEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("❌ Failed to unmarshal USPMessageEvent: %v", err)
		return err
	}

	log.Printf("📨 USP Message Event - EndpointID: %s, MessageType: %s, Payload: %d bytes, MTP: %s",
		event.EndpointID, event.MessageType, len(event.Payload), event.MTPProtocol)
	
	// Log MTP destination if present
	if event.MTPDestination.WebSocketURL != "" || event.MTPDestination.STOMPQueue != "" || event.MTPDestination.MQTTTopic != "" {
		log.Printf("📍 MTP Destination - WebSocket: %s, STOMP: %s, MQTT: %s",
			event.MTPDestination.WebSocketURL, event.MTPDestination.STOMPQueue, event.MTPDestination.MQTTTopic)
	}

	// Store the MTP routing information for this endpoint (will be used for future responses)
	s.storeEndpointMTPRouting(event.EndpointID, event.MTPProtocol, event.MTPDestination)

	// Process the USP protobuf Record from the event payload
	response, err := s.ProcessUSPMessage(event.Payload)
	if err != nil {
		log.Printf("❌ Error processing USP message: %v", err)
		return err
	}

	// If there's a response, publish it to outbound topic with routing information
	if response != nil {
		// Get routing information for this endpoint (from incoming message or database)
		destination := event.MTPDestination
		mtpProtocol := event.MTPProtocol
		
		// If no routing info in the incoming message, try to get it from database
		if destination.WebSocketURL == "" && destination.STOMPQueue == "" && destination.MQTTTopic == "" {
			destination, mtpProtocol = s.getEndpointMTPRouting(event.EndpointID)
		}
		
		// Wrap response in USPMessageEvent with endpoint ID, MTP protocol, and routing information
		err = s.kafkaProducer.PublishUSPMessageWithDestination(
			s.config.Kafka.Topics.USPMessagesOutbound,
			event.EndpointID,  // Route response back to the same endpoint
			fmt.Sprintf("resp-%d", time.Now().UnixNano()),
			"Response",
			response,
			mtpProtocol,       // Use the same MTP protocol as the incoming message
			destination,       // Include MTP-specific routing information
		)
		if err != nil {
			log.Printf("❌ Failed to publish USP response: %v", err)
			return err
		}
		log.Printf("✅ Published USP response to %s (EndpointID: %s, MTP: %s, %d bytes)",
			s.config.Kafka.Topics.USPMessagesOutbound, event.EndpointID, mtpProtocol, len(response))
	}

	return nil
}

// handleAPIRequest processes API requests from API Gateway
func (s *USPCoreService) handleAPIRequest(msg *confluentkafka.Message) error {
	log.Printf("📥 Processing API request from API Gateway")

	// Parse the request
	var req kafka.APIRequest
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		return fmt.Errorf("failed to parse API request: %w", err)
	}

	log.Printf("🔄 USP API Request: %s %s (correlation: %s)", req.Method, req.Operation, req.CorrelationID)

	// Process the request and build response
	response := kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Timestamp:     time.Now(),
	}

	// Handle different USP-specific operations
	switch req.Operation {
	case "GetUSPRecord", "SendUSPCommand", "GetUSPStatus":
		// Process USP-specific operations
		response.Status = http.StatusOK
		response.Data = map[string]interface{}{
			"message": fmt.Sprintf("USP operation %s processed successfully", req.Operation),
			"service": "usp-service",
		}
		log.Printf("✅ USP API operation processed: %s", req.Operation)
	default:
		response.Status = http.StatusNotImplemented
		response.Error = fmt.Sprintf("USP operation not implemented: %s", req.Operation)
		log.Printf("⚠️ USP API operation not implemented: %s", req.Operation)
	}

	// Send response back to API Gateway
	return s.sendAPIResponse(&response)
}

// handleDataRequest processes data requests from Data Service
func (s *USPCoreService) handleDataRequest(msg *confluentkafka.Message) error {
	log.Printf("📥 Processing data request from Data Service")

	// Parse the request
	var req kafka.APIRequest
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		return fmt.Errorf("failed to parse data request: %w", err)
	}

	log.Printf("🔄 USP Data Request: %s %s (correlation: %s)", req.Method, req.Operation, req.CorrelationID)

	// Process the request and build response
	response := kafka.APIResponse{
		CorrelationID: req.CorrelationID,
		Timestamp:     time.Now(),
	}

	// Handle different data operations
	switch req.Operation {
	case "GetDeviceParameters", "SyncDeviceData", "GetDeviceState":
		// Process data operations
		response.Status = http.StatusOK
		response.Data = map[string]interface{}{
			"message": fmt.Sprintf("USP data operation %s processed successfully", req.Operation),
			"service": "usp-service",
		}
		log.Printf("✅ USP data operation processed: %s", req.Operation)
	default:
		response.Status = http.StatusNotImplemented
		response.Error = fmt.Sprintf("USP data operation not implemented: %s", req.Operation)
		log.Printf("⚠️ USP data operation not implemented: %s", req.Operation)
	}

	// Send response back to Data Service
	return s.sendDataResponse(&response)
}

// sendAPIResponse publishes an API response to Kafka for API Gateway
func (s *USPCoreService) sendAPIResponse(resp *kafka.APIResponse) error {
	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal API response: %w", err)
	}

	err = s.kafkaProducer.PublishRaw(
		s.config.Kafka.Topics.USPAPIResponse,
		resp.CorrelationID,
		respData,
	)
	if err != nil {
		log.Printf("❌ Failed to publish API response: %v", err)
		return err
	}

	log.Printf("✅ Published API response to %s (correlation: %s)", s.config.Kafka.Topics.USPAPIResponse, resp.CorrelationID)
	return nil
}

// sendDataResponse publishes a data response to Kafka for Data Service
func (s *USPCoreService) sendDataResponse(resp *kafka.APIResponse) error {
	respData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal data response: %w", err)
	}

	err = s.kafkaProducer.PublishRaw(
		s.config.Kafka.Topics.USPDataResponse,
		resp.CorrelationID,
		respData,
	)
	if err != nil {
		log.Printf("❌ Failed to publish data response: %v", err)
		return err
	}

	log.Printf("✅ Published data response to %s (correlation: %s)", s.config.Kafka.Topics.USPDataResponse, resp.CorrelationID)
	return nil
}

// handleDataEvent processes data events from Data Service
func (s *USPCoreService) handleDataEvent(msg *confluentkafka.Message) error {
	topic := *msg.TopicPartition.Topic
	log.Printf("📥 Processing data event from topic: %s", topic)

	// TODO: Process data events (device created, updated, parameter changed, etc.)
	// This allows USP service to stay synchronized with data changes

	return nil
}

// handleMTPEvent processes MTP connection events
func (s *USPCoreService) handleMTPEvent(msg *confluentkafka.Message) error {
	topic := *msg.TopicPartition.Topic
	log.Printf("📥 Processing MTP event from topic: %s", topic)

	// TODO: Track MTP connection state, update device status, etc.

	return nil
}

// DetectUSPVersion detects USP protocol version from raw message data
func (s *USPCoreService) DetectUSPVersion(data []byte) (string, error) {
	log.Printf("🔍 Attempting to detect USP version from %d bytes of data", len(data))

	// Try to unmarshal as USP 1.3 first (most common)
	var record13 v1_3.Record
	err13 := proto.Unmarshal(data, &record13)
	if err13 == nil {
		log.Printf("🔍 Successfully unmarshaled as v1.3 Record - Version=%q, FromId=%q, ToId=%q, RecordType=%T",
			record13.Version, record13.FromId, record13.ToId, record13.RecordType)

		// Check if version field is set and valid
		if record13.Version != "" {
			log.Printf("🔍 Detected USP version from Record.Version field: %s", record13.Version)
			// Accept both "1.3" and variations
			if record13.Version == "1.3" || record13.Version == "1.3.0" {
				return "1.3", nil
			}
			if record13.Version == "1.4" || record13.Version == "1.4.0" {
				return "1.4", nil
			}
		}
		// If version field is not set but unmarshal succeeded with FromId/ToId, assume 1.3
		if record13.FromId != "" || record13.ToId != "" {
			log.Printf("🔍 Version field empty but Record has FromId=%q, ToId=%q, defaulting to USP 1.3",
				record13.FromId, record13.ToId)
			return "1.3", nil
		}
	} else {
		log.Printf("⚠️ Failed to unmarshal as v1.3 Record: %v", err13)
	}

	// Try to unmarshal as USP 1.4
	var record14 v1_4.Record
	err14 := proto.Unmarshal(data, &record14)
	if err14 == nil {
		log.Printf("🔍 Successfully unmarshaled as v1.4 Record - Version=%q, FromId=%q, ToId=%q",
			record14.Version, record14.FromId, record14.ToId)

		if record14.Version == "1.4" || record14.Version == "1.4.0" {
			log.Printf("🔍 Detected USP version 1.4 from Record.Version field")
			return "1.4", nil
		}
		// If version field is not set but unmarshal succeeded with FromId/ToId, check for 1.4 specific fields
		if record14.FromId != "" || record14.ToId != "" {
			log.Printf("🔍 Version field empty but Record has FromId=%q, ToId=%q, defaulting to USP 1.4",
				record14.FromId, record14.ToId)
			return "1.4", nil
		}
	} else {
		log.Printf("⚠️ Failed to unmarshal as v1.4 Record: %v", err14)
	}

	log.Printf("❌ Unable to detect USP version - Record.Version: v1.3=%q, v1.4=%q, FromId: v1.3=%q, v1.4=%q",
		record13.Version, record14.Version, record13.FromId, record14.FromId)

	// Debug: Show first 100 bytes of data in hex
	if len(data) > 0 {
		maxBytes := 100
		if len(data) < maxBytes {
			maxBytes = len(data)
		}
		log.Printf("🔍 First %d bytes (hex): %x", maxBytes, data[:maxBytes])
	}

	return "", fmt.Errorf("unable to detect USP version")
}

// ProcessUSPMessage processes incoming USP messages (both v1.3 and v1.4)
func (s *USPCoreService) ProcessUSPMessage(data []byte) ([]byte, error) {
	version, err := s.DetectUSPVersion(data)
	if err != nil {
		return nil, fmt.Errorf("failed to detect USP version: %w", err)
	}

	log.Printf("Processing USP %s message", version)

	switch version {
	case "1.4":
		return s.processUSP14Message(data)
	case "1.3":
		return s.processUSP13Message(data)
	default:
		return nil, fmt.Errorf("unsupported USP version: %s", version)
	}
}

func (s *USPCoreService) processUSP14Message(data []byte) ([]byte, error) {
	var record v1_4.Record
	if err := proto.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.4 record: %w", err)
	}

	log.Printf("USP 1.4 message from %s to %s", record.FromId, record.ToId)

	// Handle MTP-layer connection records (no payload, no response needed)
	switch recordType := record.RecordType.(type) {
	case *v1_4.Record_StompConnect:
		log.Printf("✅ STOMP connection established for agent %s (version %v)", record.FromId, recordType.StompConnect.Version)
		return nil, nil // No response needed for MTP-layer connect
	case *v1_4.Record_WebsocketConnect:
		log.Printf("✅ WebSocket connection established for agent %s", record.FromId)
		return nil, nil // No response needed for MTP-layer connect
	case *v1_4.Record_MqttConnect:
		log.Printf("✅ MQTT connection established for agent %s (version %v)", record.FromId, recordType.MqttConnect.Version)
		return nil, nil // No response needed for MTP-layer connect
	case *v1_4.Record_UdsConnect:
		log.Printf("✅ UDS connection record received for agent %s (protocol compliance only)", record.FromId)
		return nil, nil // No response needed for MTP-layer connect
	case *v1_4.Record_Disconnect:
		log.Printf("👋 Agent %s disconnected", record.FromId)
		return nil, nil // No response needed for disconnect
	}

	// Extract payload for application-layer messages
	var payload []byte
	switch recordType := record.RecordType.(type) {
	case *v1_4.Record_NoSessionContext:
		payload = recordType.NoSessionContext.Payload
	default:
		return nil, fmt.Errorf("unsupported USP 1.4 record type: %T", recordType)
	}

	// Parse the USP message
	var msg v1_4.Msg
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.4 message: %w", err)
	}

	log.Printf("📥 RECV: USP Message Type: %s, MsgID: %s (from: %s)", msg.Header.MsgType, msg.Header.MsgId, record.FromId)

	// Create a simple response
	responseMsg := s.createUSP14Response(&msg)
	log.Printf("📤 SEND: Created response for MsgID: %s, ResponseType: %s (to: %s)", msg.Header.MsgId, responseMsg.Header.MsgType, record.FromId)

	// Marshal response
	responsePayload, err := proto.Marshal(responseMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.4 response: %w", err)
	}

	// Create response record
	responseRecord := &v1_4.Record{
		Version: "1.4",
		ToId:    record.FromId,
		FromId:  record.ToId,
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: responsePayload,
			},
		},
	}

	return proto.Marshal(responseRecord)
}

func (s *USPCoreService) processUSP13Message(data []byte) ([]byte, error) {
	var record v1_3.Record
	if err := proto.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.3 record: %w", err)
	}

	log.Printf("USP 1.3 message from %s to %s", record.FromId, record.ToId)

	agentEndpointID := record.FromId // Capture for potential auto-discovery

	// Handle MTP-layer connection records (no payload, no response needed)
	switch recordType := record.RecordType.(type) {
	case *v1_3.Record_StompConnect:
		log.Printf("✅ STOMP connection established for agent %s (version %v)", record.FromId, recordType.StompConnect.Version)
		return nil, nil // No response needed for MTP-layer connect
	case *v1_3.Record_WebsocketConnect:
		log.Printf("✅ WebSocket connection established for agent %s", record.FromId)
		return nil, nil // No response needed for MTP-layer connect
	case *v1_3.Record_MqttConnect:
		log.Printf("✅ MQTT connection established for agent %s (version %v)", record.FromId, recordType.MqttConnect.Version)
		return nil, nil // No response needed for MTP-layer connect
	case *v1_3.Record_UdsConnect:
		log.Printf("✅ UDS connection record received for agent %s (protocol compliance only)", record.FromId)
		return nil, nil // No response needed for MTP-layer connect
	case *v1_3.Record_Disconnect:
		log.Printf("👋 Agent %s disconnected", record.FromId)
		return nil, nil // No response needed for disconnect
	}

	// Extract payload for application-layer messages
	var payload []byte
	switch recordType := record.RecordType.(type) {
	case *v1_3.Record_NoSessionContext:
		payload = recordType.NoSessionContext.Payload
		log.Printf("📦 NoSessionContext payload size: %d bytes", len(payload))
	default:
		return nil, fmt.Errorf("unsupported USP 1.3 record type: %T", recordType)
	}

	// Parse the USP message
	var msg v1_3.Msg
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.3 message: %w", err)
	}

	log.Printf("📥 RECV: USP Message Type: %s, MsgID: %s (from: %s)", msg.Header.MsgType, msg.Header.MsgId, record.FromId)

	// Create a simple response
	responseMsg := s.createUSP13Response(&msg)
	log.Printf("📤 SEND: Created response for MsgID: %s, ResponseType: %s (to: %s)", msg.Header.MsgId, responseMsg.Header.MsgType, record.FromId)

	// Check if this was a Boot! Event - if so, trigger auto-discovery after response
	if msg.Header.MsgType == v1_3.Header_NOTIFY {
		if notify := msg.Body.GetRequest().GetNotify(); notify != nil {
			if event := notify.GetEvent(); event != nil && event.EventName == "Boot!" {
				// Trigger auto-discovery in background after sending response
				go s.triggerAgentAutoDiscovery(agentEndpointID, "1.3", event.Params)
			}
		}
	}

	// Marshal response
	responsePayload, err := proto.Marshal(responseMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.3 response: %w", err)
	}

	log.Printf("📦 Response payload size: %d bytes", len(responsePayload))

	// Create response record
	responseRecord := &v1_3.Record{
		Version: "1.3",
		ToId:    record.FromId,
		FromId:  record.ToId,
		RecordType: &v1_3.Record_NoSessionContext{
			NoSessionContext: &v1_3.NoSessionContextRecord{
				Payload: responsePayload,
			},
		},
	}

	responseBytes, err := proto.Marshal(responseRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response record: %w", err)
	}

	log.Printf("📤 Response Record size: %d bytes (ToId: %s, FromId: %s)", len(responseBytes), responseRecord.ToId, responseRecord.FromId)

	return responseBytes, nil
}

func (s *USPCoreService) createUSP14Response(msg *v1_4.Msg) *v1_4.Msg {
	switch msg.Header.MsgType {
	case v1_4.Header_GET:
		return s.handleUSP14Get(msg)
	case v1_4.Header_GET_SUPPORTED_DM:
		return s.handleUSP14GetSupportedDM(msg)
	case v1_4.Header_NOTIFY:
		return s.handleUSP14Notify(msg)
	default:
		// Return error response for unsupported operations
		return &v1_4.Msg{
			Header: &v1_4.Header{
				MsgId:   fmt.Sprintf("error-resp-%s", msg.Header.MsgId),
				MsgType: v1_4.Header_ERROR,
			},
			Body: &v1_4.Body{
				MsgBody: &v1_4.Body_Error{
					Error: &v1_4.Error{
						ErrCode: 7000,
						ErrMsg:  "Operation not implemented",
					},
				},
			},
		}
	}
}

func (s *USPCoreService) createUSP13Response(msg *v1_3.Msg) *v1_3.Msg {
	switch msg.Header.MsgType {
	case v1_3.Header_GET:
		return s.handleUSP13Get(msg)
	case v1_3.Header_GET_SUPPORTED_DM:
		return s.handleUSP13GetSupportedDM(msg)
	case v1_3.Header_NOTIFY:
		return s.handleUSP13Notify(msg)
	default:
		// Return error response for unsupported operations
		return &v1_3.Msg{
			Header: &v1_3.Header{
				MsgId:   fmt.Sprintf("error-resp-%s", msg.Header.MsgId),
				MsgType: v1_3.Header_ERROR,
			},
			Body: &v1_3.Body{
				MsgBody: &v1_3.Body_Error{
					Error: &v1_3.Error{
						ErrCode: 7000,
						ErrMsg:  "Operation not implemented",
					},
				},
			},
		}
	}
}

func (s *USPCoreService) handleUSP14Get(msg *v1_4.Msg) *v1_4.Msg {
	req := msg.Body.GetRequest().GetGet()

	var requestedPathResults []*v1_4.GetResp_RequestedPathResult

	for _, path := range req.ParamPaths {
		if s.deviceManager.ValidateUSPPath(path) {
			// Create resolved path result
			resolvedResult := &v1_4.GetResp_ResolvedPathResult{
				ResolvedPath: path,
				ResultParams: map[string]string{
					path: s.getParameterValue(path),
				},
			}

			requestedPathResults = append(requestedPathResults, &v1_4.GetResp_RequestedPathResult{
				RequestedPath:       path,
				ErrCode:             0,
				ErrMsg:              "",
				ResolvedPathResults: []*v1_4.GetResp_ResolvedPathResult{resolvedResult},
			})
		} else {
			requestedPathResults = append(requestedPathResults, &v1_4.GetResp_RequestedPathResult{
				RequestedPath: path,
				ErrCode:       7004,
				ErrMsg:        "Parameter path not supported",
			})
		}
	}

	return &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   fmt.Sprintf("get-resp-%s", msg.Header.MsgId),
			MsgType: v1_4.Header_GET_RESP,
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Response{
				Response: &v1_4.Response{
					RespType: &v1_4.Response_GetResp{
						GetResp: &v1_4.GetResp{
							ReqPathResults: requestedPathResults,
						},
					},
				},
			},
		},
	}
}

func (s *USPCoreService) handleUSP13Get(msg *v1_3.Msg) *v1_3.Msg {
	req := msg.Body.GetRequest().GetGet()

	var requestedPathResults []*v1_3.GetResp_RequestedPathResult

	for _, path := range req.ParamPaths {
		if s.deviceManager.ValidateUSPPath(path) {
			// Create resolved path result
			resolvedResult := &v1_3.GetResp_ResolvedPathResult{
				ResolvedPath: path,
				ResultParams: map[string]string{
					path: s.getParameterValue(path),
				},
			}

			requestedPathResults = append(requestedPathResults, &v1_3.GetResp_RequestedPathResult{
				RequestedPath:       path,
				ErrCode:             0,
				ErrMsg:              "",
				ResolvedPathResults: []*v1_3.GetResp_ResolvedPathResult{resolvedResult},
			})
		} else {
			requestedPathResults = append(requestedPathResults, &v1_3.GetResp_RequestedPathResult{
				RequestedPath: path,
				ErrCode:       7004,
				ErrMsg:        "Parameter path not supported",
			})
		}
	}

	return &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   fmt.Sprintf("get-resp-%s", msg.Header.MsgId),
			MsgType: v1_3.Header_GET_RESP,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Response{
				Response: &v1_3.Response{
					RespType: &v1_3.Response_GetResp{
						GetResp: &v1_3.GetResp{
							ReqPathResults: requestedPathResults,
						},
					},
				},
			},
		},
	}
}

func (s *USPCoreService) handleUSP14GetSupportedDM(msg *v1_4.Msg) *v1_4.Msg {
	req := msg.Body.GetRequest().GetGetSupportedDm()

	var results []*v1_4.GetSupportedDMResp_RequestedObjectResult

	for _, objPath := range req.ObjPaths {
		supportedObj := &v1_4.GetSupportedDMResp_SupportedObjectResult{
			SupportedObjPath: objPath,
			Access:           v1_4.GetSupportedDMResp_OBJ_READ_ONLY,
			IsMultiInstance:  false,
		}

		results = append(results, &v1_4.GetSupportedDMResp_RequestedObjectResult{
			ReqObjPath:       objPath,
			ErrCode:          0,
			ErrMsg:           "",
			DataModelInstUri: "urn:broadband-forum-org:tr-181-2-19-1",
			SupportedObjs:    []*v1_4.GetSupportedDMResp_SupportedObjectResult{supportedObj},
		})
	}

	return &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   fmt.Sprintf("get-supported-dm-resp-%s", msg.Header.MsgId),
			MsgType: v1_4.Header_GET_SUPPORTED_DM_RESP,
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Response{
				Response: &v1_4.Response{
					RespType: &v1_4.Response_GetSupportedDmResp{
						GetSupportedDmResp: &v1_4.GetSupportedDMResp{
							ReqObjResults: results,
						},
					},
				},
			},
		},
	}
}

func (s *USPCoreService) handleUSP13GetSupportedDM(msg *v1_3.Msg) *v1_3.Msg {
	req := msg.Body.GetRequest().GetGetSupportedDm()

	var results []*v1_3.GetSupportedDMResp_RequestedObjectResult

	for _, objPath := range req.ObjPaths {
		supportedObj := &v1_3.GetSupportedDMResp_SupportedObjectResult{
			SupportedObjPath: objPath,
			Access:           v1_3.GetSupportedDMResp_OBJ_READ_ONLY,
			IsMultiInstance:  false,
		}

		results = append(results, &v1_3.GetSupportedDMResp_RequestedObjectResult{
			ReqObjPath:       objPath,
			ErrCode:          0,
			ErrMsg:           "",
			DataModelInstUri: "urn:broadband-forum-org:tr-181-2-19-1",
			SupportedObjs:    []*v1_3.GetSupportedDMResp_SupportedObjectResult{supportedObj},
		})
	}

	return &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   fmt.Sprintf("get-supported-dm-resp-%s", msg.Header.MsgId),
			MsgType: v1_3.Header_GET_SUPPORTED_DM_RESP,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Response{
				Response: &v1_3.Response{
					RespType: &v1_3.Response_GetSupportedDmResp{
						GetSupportedDmResp: &v1_3.GetSupportedDMResp{
							ReqObjResults: results,
						},
					},
				},
			},
		},
	}
}

// handleUSP14Notify handles NOTIFY messages (including Boot! Event) for USP 1.4
func (s *USPCoreService) handleUSP14Notify(msg *v1_4.Msg) *v1_4.Msg {
	notify := msg.Body.GetRequest().GetNotify()

	// Log the notification details
	if notify != nil {
		log.Printf("📨 NOTIFY: Subscription ID: %s", notify.SubscriptionId)

		// Check if this is an Event notification (including Boot! Event)
		if event := notify.GetEvent(); event != nil {
			log.Printf("📨 EVENT: ObjPath=%s, EventName=%s", event.ObjPath, event.EventName)

			// Handle Boot! Event
			if event.EventName == "Boot!" {
				log.Printf("🔔 Received Boot! Event from agent (TR-369 onboarding)")
				// Log boot parameters
				for k, v := range event.Params {
					log.Printf("   Boot Param: %s = %s", k, v)
				}
			}
		}
	}

	// Return NotifyResp acknowledgment
	return &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   fmt.Sprintf("notify-resp-%s", msg.Header.MsgId),
			MsgType: v1_4.Header_NOTIFY_RESP,
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Response{
				Response: &v1_4.Response{
					RespType: &v1_4.Response_NotifyResp{
						NotifyResp: &v1_4.NotifyResp{
							SubscriptionId: notify.GetSubscriptionId(),
						},
					},
				},
			},
		},
	}
}

// handleUSP13Notify handles NOTIFY messages (including Boot! Event) for USP 1.3
func (s *USPCoreService) handleUSP13Notify(msg *v1_3.Msg) *v1_3.Msg {
	notify := msg.Body.GetRequest().GetNotify()

	// Log the notification details
	if notify != nil {
		log.Printf("📨 NOTIFY: Subscription ID: %s", notify.SubscriptionId)

		// Check if this is an Event notification (including Boot! Event)
		if event := notify.GetEvent(); event != nil {
			log.Printf("📨 EVENT: ObjPath=%s, EventName=%s", event.ObjPath, event.EventName)

			// Handle Boot! Event
			if event.EventName == "Boot!" {
				log.Printf("🔔 Received Boot! Event from agent (TR-369 onboarding)")
				// Log boot parameters
				for k, v := range event.Params {
					log.Printf("   Boot Param: %s = %s", k, v)
				}
			}
		}
	}

	// Return NotifyResp acknowledgment
	return &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   fmt.Sprintf("notify-resp-%s", msg.Header.MsgId),
			MsgType: v1_3.Header_NOTIFY_RESP,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Response{
				Response: &v1_3.Response{
					RespType: &v1_3.Response_NotifyResp{
						NotifyResp: &v1_3.NotifyResp{
							SubscriptionId: notify.GetSubscriptionId(),
						},
					},
				},
			},
		},
	}
}

func (s *USPCoreService) getParameterValue(path string) string {
	// Get value from TR-181 device manager
	deviceInfo := s.deviceManager.GetDeviceInfo()
	if value, exists := deviceInfo[path]; exists {
		return fmt.Sprintf("%v", value)
	}

	// Get values from USP Agent configuration (from YAML)
	agentConfig := s.config.USPAgentDevice

	switch path {
	case "Device.DeviceInfo.Manufacturer":
		if agentConfig.Manufacturer != "" {
			return agentConfig.Manufacturer
		}
		return "Unknown"
	case "Device.DeviceInfo.ManufacturerOUI":
		if agentConfig.OUI != "" {
			return agentConfig.OUI
		}
		return "000000"
	case "Device.DeviceInfo.ModelName":
		if agentConfig.ModelName != "" {
			return agentConfig.ModelName
		}
		return "TR-369 USP Agent"
	case "Device.DeviceInfo.SerialNumber":
		if agentConfig.SerialNumber != "" {
			return agentConfig.SerialNumber
		}
		return "UNKNOWN"
	case "Device.DeviceInfo.SoftwareVersion":
		if agentConfig.SoftwareVersion != "" {
			return agentConfig.SoftwareVersion
		}
		return version.Version
	case "Device.DeviceInfo.HardwareVersion":
		if agentConfig.HardwareVersion != "" {
			return agentConfig.HardwareVersion
		}
		return "1.0"
	case "Device.DeviceInfo.ProductClass":
		if agentConfig.ProductClass != "" {
			return agentConfig.ProductClass
		}
		return "TR-369"
	default:
		return "unknown"
	}
}

func main() {
	log.Printf("🚀 Starting OpenUSP Core Service...")

	// Command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.GetFullVersion("OpenUSP Core Service"))
		return
	}

	if *showHelp {
		fmt.Println("OpenUSP Core Service - TR-369 USP Protocol Engine")
		fmt.Println("=================================================")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  usp-service [flags]")
		fmt.Println("")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("All configuration loaded from configs/openusp.yml")
		fmt.Println("Environment variables can override YAML values (see openusp.yml comments)")
		return
	}

	fmt.Println("OpenUSP Core Service - Multi-Version TR-369 Protocol Engine")
	fmt.Println("==========================================================")

	// Print version info
	version.PrintVersionInfo("OpenUSP Core Service")
	fmt.Println()

	// Load configuration
	cfg := config.Load()

	// Convert HTTP port from string to int
	httpPort, err := strconv.Atoi(cfg.USPServiceHTTPPort)
	if err != nil || httpPort == 0 {
		log.Fatalf("Invalid USP service HTTP port: %s", cfg.USPServiceHTTPPort)
	}

	// Initialize the USP core service with Kafka
	uspService, err := NewUSPCoreService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize USP Core Service: %v", err)
	}
	log.Printf("✅ USP Core Service initialized with TR-181 data model")

	// Setup Kafka consumers
	if err := uspService.setupKafkaConsumers(); err != nil {
		log.Fatalf("Failed to setup Kafka consumers: %v", err)
	}

	// Start Kafka consumer
	go uspService.kafkaConsumer.Start()

	// Start HTTP server for health/metrics
	go func() {
		mux := http.NewServeMux()

		// Health check endpoint
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			// Test Kafka connection
			kafkaStatus := "connected"
			if err := uspService.kafkaClient.Ping(); err != nil {
				kafkaStatus = "disconnected"
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := fmt.Sprintf(`{
				"service": "usp-service",
				"status": "healthy",
				"version": "1.0.0",
				"http_port": %d,
				"kafka": "%s",
				"kafka_brokers": %q,
				"timestamp": "%s"
			}`, httpPort, kafkaStatus, cfg.Kafka.Brokers, time.Now().Format(time.RFC3339))
			w.Write([]byte(response))
		})

		// Status endpoint
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := fmt.Sprintf(`{
				"service": "usp-service",
				"status": "running",
				"version": "%s",
				"http_port": %d,
				"tr181_objects": 822,
				"usp_versions": ["1.3", "1.4"],
				"kafka_topics": {
					"inbound": "%s",
					"outbound": "%s"
				},
				"timestamp": "%s"
			}`, version.Version, httpPort, cfg.Kafka.Topics.USPMessagesInbound, cfg.Kafka.Topics.USPMessagesOutbound, time.Now().Format(time.RFC3339))
			w.Write([]byte(response))
		})

		// Metrics endpoint
		mux.Handle("/metrics", metrics.HTTPHandler())

		// Parse HTTP timeouts from config
		readTimeout, _ := time.ParseDuration(cfg.USPService.ReadTimeout)
		writeTimeout, _ := time.ParseDuration(cfg.USPService.WriteTimeout)
		idleTimeout, _ := time.ParseDuration(cfg.USPService.IdleTimeout)

		uspService.httpServer = &http.Server{
			Addr:         fmt.Sprintf(":%d", httpPort),
			Handler:      mux,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		}

		log.Printf("🌐 Starting HTTP server on port %d", httpPort)
		if err := uspService.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Printf("✅ USP Service started - Kafka consumer: %s, HTTP: %d",
		cfg.Kafka.Topics.USPMessagesInbound, httpPort)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Printf("🛑 Received shutdown signal")
	log.Printf("🛑 Stopping USP Service...")

	// Stop Kafka consumer
	if uspService.kafkaConsumer != nil {
		uspService.kafkaConsumer.Close()
	}

	// Stop Kafka producer
	if uspService.kafkaProducer != nil {
		uspService.kafkaProducer.Close()
	}

	// Close Kafka client
	if uspService.kafkaClient != nil {
		uspService.kafkaClient.Close()
	}

	// Stop HTTP server with configured timeout
	if uspService.httpServer != nil {
		shutdownTimeout, _ := time.ParseDuration(cfg.USPService.ShutdownTimeout)
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		uspService.httpServer.Shutdown(ctx)
	}

	log.Printf("✅ USP Service stopped")
}

// triggerAgentAutoDiscovery initiates the auto-discovery workflow after successful Boot! Event onboarding
func (s *USPCoreService) triggerAgentAutoDiscovery(agentEndpointID, uspVersion string, bootParams map[string]string) {
	log.Printf("🔍 Starting auto-discovery for agent: %s (USP %s)", agentEndpointID, uspVersion)

	// Wait a moment for the Boot! response to be delivered
	time.Sleep(2 * time.Second)

	// Step 1: Register/Update device in data service
	if err := s.registerDeviceFromBootParams(agentEndpointID, bootParams); err != nil {
		log.Printf("❌ Failed to register device %s: %v", agentEndpointID, err)
	} else {
		log.Printf("✅ Device %s registered successfully", agentEndpointID)
	}

	// Step 2: Send GetSupportedDM to discover data model
	log.Printf("📋 Sending GetSupportedDM request to %s", agentEndpointID)
	// TODO: Implement sending GetSupportedDM request
	// This requires creating a USP message, wrapping in Record, and publishing to Kafka

	// Step 3: Send Get request for key parameters
	log.Printf("📋 Sending Get request for Device.DeviceInfo. parameters to %s", agentEndpointID)
	// TODO: Implement sending Get request

	log.Printf("✅ Auto-discovery workflow initiated for agent: %s", agentEndpointID)
}

// registerDeviceFromBootParams creates/updates device record using Boot! Event parameters
func (s *USPCoreService) registerDeviceFromBootParams(agentEndpointID string, bootParams map[string]string) error {
	log.Printf("📝 Registering device from Boot! Event parameters")

	// Extract key device information from Boot! params
	deviceData := map[string]interface{}{
		"endpoint_id":      agentEndpointID,
		"manufacturer":     bootParams["Manufacturer"],
		"model_name":       bootParams["ModelName"],
		"serial_number":    bootParams["SerialNumber"],
		"product_class":    bootParams["ProductClass"],
		"software_version": bootParams["SoftwareVersion"],
		"firmware_version": bootParams["SoftwareVersion"], // Often the same
		"hardware_version": bootParams["HardwareVersion"],
		"boot_cause":       bootParams["Cause"],
		"onboarded_at":     time.Now().Format(time.RFC3339),
		"last_boot_at":     time.Now().Format(time.RFC3339),
		"status":           "online",
	}

	// Publish device registration event to Kafka
	// The data-service will consume this and persist to database
	err := s.kafkaProducer.PublishDeviceEvent(
		s.config.Kafka.Topics.DataDeviceCreated,
		"data.device.created", // event type
		agentEndpointID,       // deviceID
		agentEndpointID,       // endpointID
		"usp",                 // protocol
		deviceData,
	)

	if err != nil {
		return fmt.Errorf("failed to publish device.onboarded event: %w", err)
	}

	log.Printf("✅ Published device.onboarded event for %s to topic: %s",
		agentEndpointID, s.config.Kafka.Topics.DataDeviceCreated)

	return nil
}

// endpointMTPCache stores MTP routing information temporarily (in-memory cache)
// In production, this should be stored in database or distributed cache
var endpointMTPCache = struct {
	sync.RWMutex
	routes map[string]struct {
		protocol    string
		destination kafka.MTPDestination
	}
}{
	routes: make(map[string]struct {
		protocol    string
		destination kafka.MTPDestination
	}),
}

// storeEndpointMTPRouting stores MTP routing information for an endpoint
func (s *USPCoreService) storeEndpointMTPRouting(endpointID string, mtpProtocol string, destination kafka.MTPDestination) {
	endpointMTPCache.Lock()
	defer endpointMTPCache.Unlock()
	
	endpointMTPCache.routes[endpointID] = struct {
		protocol    string
		destination kafka.MTPDestination
	}{
		protocol:    mtpProtocol,
		destination: destination,
	}
	
	log.Printf("💾 Stored MTP routing for %s: protocol=%s, websocket=%s, stomp=%s, mqtt=%s",
		endpointID, mtpProtocol, destination.WebSocketURL, destination.STOMPQueue, destination.MQTTTopic)
}

// getEndpointMTPRouting retrieves MTP routing information for an endpoint
func (s *USPCoreService) getEndpointMTPRouting(endpointID string) (kafka.MTPDestination, string) {
	endpointMTPCache.RLock()
	defer endpointMTPCache.RUnlock()
	
	if route, exists := endpointMTPCache.routes[endpointID]; exists {
		log.Printf("📖 Retrieved MTP routing for %s: protocol=%s", endpointID, route.protocol)
		return route.destination, route.protocol
	}
	
	log.Printf("⚠️ No MTP routing found for %s in cache", endpointID)
	return kafka.MTPDestination{}, ""
}
