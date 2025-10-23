package database

import (
	"time"

	"gorm.io/gorm"
)

// Device represents a USP device/agent in the database
type Device struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	EndpointID      string     `gorm:"uniqueIndex;not null" json:"endpoint_id"`
	ProductClass    string     `json:"product_class"`
	Manufacturer    string     `json:"manufacturer"`
	ModelName       string     `json:"model_name"`
	SerialNumber    string     `json:"serial_number"`
	SoftwareVersion string     `json:"software_version"`
	HardwareVersion string     `json:"hardware_version"`
	Status          string     `gorm:"default:'offline'" json:"status"` // online, offline, provisioning
	LastSeen        *time.Time `json:"last_seen"`
	IPAddress       string     `json:"ip_address"`
	ConnectionType  string     `json:"connection_type"` // MQTT, STOMP, WebSocket, Unix

	// MTP routing information (stored as MTPProtocol for backward compatibility)
	MTPProtocol  string `json:"mtp_protocol"`  // websocket, stomp, mqtt, http
	WebSocketURL string `json:"websocket_url"` // WebSocket endpoint URL
	STOMPQueue   string `json:"stomp_queue"`   // STOMP destination queue
	MQTTTopic    string `json:"mqtt_topic"`    // MQTT response topic
	HTTPURL      string `json:"http_url"`      // HTTP callback URL

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Parameters []Parameter `gorm:"foreignKey:DeviceID" json:"parameters,omitempty"`
	Alerts     []Alert     `gorm:"foreignKey:DeviceID" json:"alerts,omitempty"`
	Sessions   []Session   `gorm:"foreignKey:DeviceID" json:"sessions,omitempty"`
}

// Parameter represents a TR-181 parameter value for a device
type Parameter struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DeviceID    uint      `gorm:"not null;index" json:"device_id"`
	Path        string    `gorm:"not null;index" json:"path"` // TR-181 parameter path
	Value       string    `json:"value"`
	Type        string    `json:"type"` // string, int, boolean, dateTime, etc.
	Writable    bool      `gorm:"default:false" json:"writable"`
	LastUpdated time.Time `json:"last_updated"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationship
	Device Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// Alert represents a device alert/event
type Alert struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	DeviceID   uint       `gorm:"not null;index" json:"device_id"`
	Type       string     `gorm:"not null" json:"type"` // error, warning, info
	Code       string     `json:"code"`
	Message    string     `gorm:"not null" json:"message"`
	Severity   int        `gorm:"default:1" json:"severity"` // 1=low, 2=medium, 3=high, 4=critical
	Resolved   bool       `gorm:"default:false" json:"resolved"`
	ResolvedAt *time.Time `json:"resolved_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Relationship
	Device Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// Session represents a USP session between controller and agent
type Session struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	DeviceID     uint       `gorm:"not null;index" json:"device_id"`
	SessionID    string     `gorm:"uniqueIndex;not null" json:"session_id"`
	Protocol     string     `gorm:"not null" json:"protocol"`       // MQTT, STOMP, WebSocket, Unix
	Status       string     `gorm:"default:'active'" json:"status"` // active, closed, timeout
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	LastActivity time.Time  `json:"last_activity"`
	MessageCount int        `gorm:"default:0" json:"message_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relationship
	Device Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// ConnectionHistory represents the connection history of agents
type ConnectionHistory struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	DeviceID        uint       `gorm:"index" json:"device_id"`            // Foreign key to Device (nullable for auto-registration)
	EndpointID      string     `gorm:"index;not null" json:"endpoint_id"` // Agent endpoint ID
	MTPProtocol     string     `gorm:"not null" json:"mtp_protocol"`      // websocket, stomp, mqtt, http
	ProtocolVersion string     `json:"protocol_version"`                  // MTP protocol version (STOMP 1.2, MQTT 5.0, etc.)
	EventType       string     `gorm:"not null" json:"event_type"`        // "connected", "disconnected"
	ConnectedAt     time.Time  `json:"connected_at"`                      // When connection was established
	DisconnectedAt  *time.Time `json:"disconnected_at"`                   // When connection was closed (NULL if still connected)
	Duration        int64      `json:"duration"`                          // Connection duration in seconds (computed on disconnect)
	CreatedAt       time.Time  `json:"created_at"`

	// Relationship
	Device *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// TableName methods for custom table names (optional)
func (Device) TableName() string {
	return "devices"
}

func (Parameter) TableName() string {
	return "parameters"
}

func (Alert) TableName() string {
	return "alerts"
}

func (Session) TableName() string {
	return "sessions"
}

func (ConnectionHistory) TableName() string {
	return "connection_history"
}
