package kafka

import (
	"encoding/json"
	"time"
)

// EventType represents the type of event
type EventType string

const (
	// USP Message Events
	EventUSPMessageInbound  EventType = "usp.message.inbound"
	EventUSPMessageOutbound EventType = "usp.message.outbound"
	EventUSPRecordParsed    EventType = "usp.record.parsed"

	// Data Events
	EventDeviceCreated    EventType = "data.device.created"
	EventDeviceUpdated    EventType = "data.device.updated"
	EventDeviceDeleted    EventType = "data.device.deleted"
	EventParameterUpdated EventType = "data.parameter.updated"
	EventObjectCreated    EventType = "data.object.created"
	EventAlertCreated     EventType = "data.alert.created"

	// CWMP Events
	EventCWMPMessageInbound  EventType = "cwmp.message.inbound"
	EventCWMPMessageOutbound EventType = "cwmp.message.outbound"

	// Agent Events
	EventAgentStatusUpdated       EventType = "agent.status.updated"
	EventAgentOnboardingStarted   EventType = "agent.onboarding.started"
	EventAgentOnboardingCompleted EventType = "agent.onboarding.completed"

	// MTP Events
	EventMTPConnectionEstablished EventType = "mtp.connection.established"
	EventMTPConnectionClosed      EventType = "mtp.connection.closed"
)

// BaseEvent is the common structure for all events
type BaseEvent struct {
	EventID   string                 `json:"event_id"`
	EventType EventType              `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// USPMessageEvent represents a USP protocol message
type USPMessageEvent struct {
	BaseEvent
	EndpointID   string `json:"endpoint_id"`
	MessageID    string `json:"message_id"`
	MessageType  string `json:"message_type"`
	Payload      []byte `json:"payload"` // Serialized USP Record
	MTPProtocol  string `json:"mtp_protocol,omitempty"`
	MTPSessionID string `json:"mtp_session_id,omitempty"`
}

// USPRecordParsedEvent represents a parsed USP record
type USPRecordParsedEvent struct {
	BaseEvent
	EndpointID  string                 `json:"endpoint_id"`
	MessageID   string                 `json:"message_id"`
	MessageType string                 `json:"message_type"`
	RecordType  string                 `json:"record_type"`
	ToID        string                 `json:"to_id"`
	FromID      string                 `json:"from_id"`
	Payload     map[string]interface{} `json:"payload"`
}

// DeviceEvent represents device lifecycle events
type DeviceEvent struct {
	BaseEvent
	DeviceID     string                 `json:"device_id"`
	EndpointID   string                 `json:"endpoint_id"`
	SerialNumber string                 `json:"serial_number,omitempty"`
	ProductClass string                 `json:"product_class,omitempty"`
	Manufacturer string                 `json:"manufacturer,omitempty"`
	ModelName    string                 `json:"model_name,omitempty"`
	Protocol     string                 `json:"protocol"` // "USP" or "CWMP"
	Status       string                 `json:"status,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// ParameterEvent represents parameter update events
type ParameterEvent struct {
	BaseEvent
	DeviceID      string                 `json:"device_id"`
	EndpointID    string                 `json:"endpoint_id"`
	ParameterPath string                 `json:"parameter_path"`
	OldValue      interface{}            `json:"old_value,omitempty"`
	NewValue      interface{}            `json:"new_value"`
	DataType      string                 `json:"data_type,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ObjectEvent represents data model object creation events
type ObjectEvent struct {
	BaseEvent
	DeviceID   string                 `json:"device_id"`
	EndpointID string                 `json:"endpoint_id"`
	ObjectPath string                 `json:"object_path"`
	ObjectType string                 `json:"object_type,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// AlertEvent represents notification/alert events
type AlertEvent struct {
	BaseEvent
	DeviceID   string                 `json:"device_id"`
	EndpointID string                 `json:"endpoint_id"`
	AlertType  string                 `json:"alert_type"`
	Severity   string                 `json:"severity"`
	Message    string                 `json:"message"`
	AlertData  map[string]interface{} `json:"alert_data,omitempty"`
}

// CWMPMessageEvent represents a CWMP/TR-069 protocol message
type CWMPMessageEvent struct {
	BaseEvent
	DeviceID     string `json:"device_id"`
	SerialNumber string `json:"serial_number"`
	MessageType  string `json:"message_type"`
	Payload      []byte `json:"payload"` // XML SOAP message
	SessionID    string `json:"session_id,omitempty"`
}

// AgentStatusEvent represents agent status changes
type AgentStatusEvent struct {
	BaseEvent
	AgentID    string                 `json:"agent_id"`
	AgentType  string                 `json:"agent_type"` // "USP" or "CWMP"
	Status     string                 `json:"status"`     // "online", "offline", "error"
	Details    string                 `json:"details,omitempty"`
	StatusData map[string]interface{} `json:"status_data,omitempty"`
}

// AgentOnboardingEvent represents agent onboarding lifecycle
type AgentOnboardingEvent struct {
	BaseEvent
	AgentID      string                 `json:"agent_id"`
	AgentType    string                 `json:"agent_type"` // "USP" or "CWMP"
	EndpointID   string                 `json:"endpoint_id"`
	SerialNumber string                 `json:"serial_number"`
	Status       string                 `json:"status"` // "started", "in_progress", "completed", "failed"
	Stage        string                 `json:"stage,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// MTPConnectionEvent represents MTP connection lifecycle
type MTPConnectionEvent struct {
	BaseEvent
	ConnectionID string                 `json:"connection_id"`
	Protocol     string                 `json:"protocol"` // "websocket", "mqtt", "stomp", "uds", "http"
	EndpointID   string                 `json:"endpoint_id,omitempty"`
	RemoteAddr   string                 `json:"remote_addr,omitempty"`
	Status       string                 `json:"status"` // "established", "closed", "error"
	Reason       string                 `json:"reason,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// NewBaseEvent creates a new base event with common fields
func NewBaseEvent(eventType EventType, source string) BaseEvent {
	return BaseEvent{
		EventID:   generateEventID(),
		EventType: eventType,
		Timestamp: time.Now(),
		Source:    source,
		Metadata:  make(map[string]interface{}),
	}
}

// ToJSON serializes an event to JSON
func ToJSON(event interface{}) ([]byte, error) {
	return json.Marshal(event)
}

// FromJSON deserializes an event from JSON
func FromJSON(data []byte, event interface{}) error {
	return json.Unmarshal(data, event)
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return time.Now().Format("20060102150405") + "-" + generateRandomString(8)
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
