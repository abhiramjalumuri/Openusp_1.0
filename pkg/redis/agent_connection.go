package redis

import (
	"fmt"
	"time"

	"openusp/pkg/kafka"
)

const (
	// Redis key prefixes
	KeyPrefixAgentConnection = "agent:connection:"
	KeyPrefixAgentActivity   = "agent:activity:"

	// Default TTL for connection state (24 hours)
	DefaultConnectionTTL = 24 * time.Hour
)

// AgentConnection represents the connection state of an agent
type AgentConnection struct {
	EndpointID      string               `json:"endpoint_id"`
	MTPProtocol     string               `json:"mtp_protocol"` // "websocket", "stomp", "mqtt"
	MTPDestination  kafka.MTPDestination `json:"mtp_destination"`
	ConnectedAt     int64                `json:"connected_at"`     // Unix timestamp
	LastActivity    int64                `json:"last_activity"`    // Unix timestamp
	ProtocolVersion string               `json:"protocol_version"` // MTP protocol version
	Status          string               `json:"status"`           // "connected", "disconnected", "inactive"
}

// StoreAgentConnection stores agent connection state in Redis
func (c *Client) StoreAgentConnection(conn *AgentConnection) error {
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	key := KeyPrefixAgentConnection + conn.EndpointID

	// Store connection with TTL
	if err := c.Set(key, conn, DefaultConnectionTTL); err != nil {
		return fmt.Errorf("failed to store agent connection: %w", err)
	}

	// Also update last activity timestamp
	if err := c.UpdateAgentActivity(conn.EndpointID); err != nil {
		// Log but don't fail on activity update
		fmt.Printf("Warning: Failed to update agent activity: %v\n", err)
	}

	return nil
}

// GetAgentConnection retrieves agent connection state from Redis
func (c *Client) GetAgentConnection(endpointID string) (*AgentConnection, error) {
	key := KeyPrefixAgentConnection + endpointID

	var conn AgentConnection
	if err := c.Get(key, &conn); err != nil {
		return nil, fmt.Errorf("failed to get agent connection: %w", err)
	}

	return &conn, nil
}

// UpdateAgentActivity updates the last activity timestamp for an agent
func (c *Client) UpdateAgentActivity(endpointID string) error {
	// Get existing connection
	conn, err := c.GetAgentConnection(endpointID)
	if err != nil {
		return err
	}

	// Update last activity
	conn.LastActivity = time.Now().Unix()

	// Store back
	return c.StoreAgentConnection(conn)
}

// RemoveAgentConnection removes agent connection from Redis
func (c *Client) RemoveAgentConnection(endpointID string) error {
	key := KeyPrefixAgentConnection + endpointID
	return c.Delete(key)
}

// GetAllAgentConnections retrieves all agent connections
func (c *Client) GetAllAgentConnections() ([]*AgentConnection, error) {
	pattern := KeyPrefixAgentConnection + "*"
	keys, err := c.Keys(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection keys: %w", err)
	}

	connections := make([]*AgentConnection, 0, len(keys))
	for _, key := range keys {
		var conn AgentConnection
		if err := c.Get(key, &conn); err != nil {
			// Skip invalid entries but continue
			fmt.Printf("Warning: Failed to get connection for key %s: %v\n", key, err)
			continue
		}
		connections = append(connections, &conn)
	}

	return connections, nil
}

// GetConnectedAgents returns all agents with "connected" status
func (c *Client) GetConnectedAgents() ([]*AgentConnection, error) {
	all, err := c.GetAllAgentConnections()
	if err != nil {
		return nil, err
	}

	connected := make([]*AgentConnection, 0)
	for _, conn := range all {
		if conn.Status == "connected" {
			connected = append(connected, conn)
		}
	}

	return connected, nil
}

// GetInactiveAgents returns agents that haven't been active for the specified duration
func (c *Client) GetInactiveAgents(inactiveDuration time.Duration) ([]*AgentConnection, error) {
	all, err := c.GetAllAgentConnections()
	if err != nil {
		return nil, err
	}

	threshold := time.Now().Add(-inactiveDuration).Unix()
	inactive := make([]*AgentConnection, 0)

	for _, conn := range all {
		if conn.LastActivity < threshold && conn.Status == "connected" {
			inactive = append(inactive, conn)
		}
	}

	return inactive, nil
}

// UpdateConnectionStatus updates the status of an agent connection
func (c *Client) UpdateConnectionStatus(endpointID, status string) error {
	conn, err := c.GetAgentConnection(endpointID)
	if err != nil {
		return err
	}

	conn.Status = status
	conn.LastActivity = time.Now().Unix()

	return c.StoreAgentConnection(conn)
}

// GetAgentsByProtocol returns all agents using a specific MTP protocol
func (c *Client) GetAgentsByProtocol(protocol string) ([]*AgentConnection, error) {
	all, err := c.GetAllAgentConnections()
	if err != nil {
		return nil, err
	}

	agents := make([]*AgentConnection, 0)
	for _, conn := range all {
		if conn.MTPProtocol == protocol {
			agents = append(agents, conn)
		}
	}

	return agents, nil
}

// GetConnectionCount returns the total number of active connections
func (c *Client) GetConnectionCount() (int, error) {
	pattern := KeyPrefixAgentConnection + "*"
	keys, err := c.Keys(pattern)
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

// CleanupExpiredConnections removes connections that have been inactive beyond TTL
// This is called periodically to clean up stale data
func (c *Client) CleanupExpiredConnections(maxInactiveDuration time.Duration) (int, error) {
	inactive, err := c.GetInactiveAgents(maxInactiveDuration)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, conn := range inactive {
		if err := c.RemoveAgentConnection(conn.EndpointID); err != nil {
			fmt.Printf("Warning: Failed to remove inactive connection %s: %v\n", conn.EndpointID, err)
			continue
		}
		count++
	}

	return count, nil
}
