package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"openusp/internal/database"
	"openusp/pkg/kafka"
	"openusp/pkg/redis"

	"gorm.io/gorm"
)

// handleAgentConnect processes agent connection events
// Implements Phases 2-4: Connection tracking, database persistence, auto-registration
func (s *USPCoreService) handleAgentConnect(
	endpointID string,
	mtpProtocol string,
	mtpDestination kafka.MTPDestination,
	protocolVersion string,
) error {
	log.Printf("🔌 Processing agent connection: %s via %s", endpointID, mtpProtocol)

	// Phase 1 & 2: Store connection state in Redis
	conn := &redis.AgentConnection{
		EndpointID:      endpointID,
		MTPProtocol:     mtpProtocol,
		MTPDestination:  mtpDestination,
		ConnectedAt:     time.Now().Unix(),
		LastActivity:    time.Now().Unix(),
		ProtocolVersion: protocolVersion,
		Status:          "connected",
	}

	if err := s.redisClient.StoreAgentConnection(conn); err != nil {
		log.Printf("❌ Failed to store connection in Redis: %v", err)
		// Continue even if Redis fails - degrade gracefully
	} else {
		log.Printf("✅ Connection state stored in Redis for %s", endpointID)
	}

	// Phase 3 & 4: Check if device exists in database
	device, err := s.repos.Device.GetByEndpointID(endpointID)

	if err == gorm.ErrRecordNotFound {
		// Device doesn't exist - auto-register
		log.Printf("🆕 New agent detected: %s - auto-registering", endpointID)

		newDevice := &database.Device{
			EndpointID:  endpointID,
			MTPProtocol: mtpProtocol,
			Status:      "provisioning", // Will be updated to "online" after discovery
			LastSeen:    timePtr(time.Now()),
		}

		// Set MTP routing info
		switch mtpProtocol {
		case "websocket":
			newDevice.WebSocketURL = mtpDestination.WebSocketURL
		case "stomp":
			newDevice.STOMPQueue = mtpDestination.STOMPQueue
		case "mqtt":
			newDevice.MQTTTopic = mtpDestination.MQTTTopic
		case "http":
			newDevice.HTTPURL = mtpDestination.HTTPURL
		}

		if err := s.repos.Device.Create(newDevice); err != nil {
			log.Printf("❌ Failed to auto-register device: %v", err)
			return fmt.Errorf("failed to auto-register device: %w", err)
		}

		device = newDevice
		log.Printf("✅ Device auto-registered: %s (ID: %d)", endpointID, device.ID)

		// Trigger auto-discovery in background
		go s.triggerDeviceDiscovery(endpointID, mtpProtocol)

	} else if err == nil {
		// Device exists - update status and MTP info
		log.Printf("🔄 Known agent reconnected: %s", endpointID)

		if err := s.repos.Device.UpdateConnectionStatus(endpointID, "online"); err != nil {
			log.Printf("❌ Failed to update device status: %v", err)
		}

		// Update MTP routing information
		if err := s.repos.Device.UpdateMTPInfo(
			endpointID,
			mtpProtocol,
			mtpDestination.WebSocketURL,
			mtpDestination.STOMPQueue,
			mtpDestination.MQTTTopic,
			mtpDestination.HTTPURL,
		); err != nil {
			log.Printf("❌ Failed to update MTP info: %v", err)
		}
	} else {
		log.Printf("❌ Failed to check device existence: %v", err)
		return fmt.Errorf("failed to check device existence: %w", err)
	}

	// Phase 3: Record connection history
	history := &database.ConnectionHistory{
		DeviceID:        device.ID,
		EndpointID:      endpointID,
		MTPProtocol:     mtpProtocol,
		ProtocolVersion: protocolVersion,
		EventType:       "connected",
		ConnectedAt:     time.Now(),
	}

	if err := s.repos.ConnectionHistory.Create(history); err != nil {
		log.Printf("❌ Failed to record connection history: %v", err)
		// Continue even if history fails
	} else {
		log.Printf("📋 Connection history recorded (ID: %d)", history.ID)
	}

	// Publish Kafka connection event
	if err := s.publishConnectionEvent(endpointID, "connected", mtpProtocol); err != nil {
		log.Printf("❌ Failed to publish connection event: %v", err)
	}

	return nil
}

// handleAgentDisconnect processes agent disconnection events
func (s *USPCoreService) handleAgentDisconnect(endpointID string, mtpProtocol string) error {
	log.Printf("👋 Processing agent disconnection: %s", endpointID)

	// Update Redis connection status
	if err := s.redisClient.UpdateConnectionStatus(endpointID, "disconnected"); err != nil {
		log.Printf("⚠️ Failed to update connection status in Redis: %v", err)
	}

	// Update device status in database
	if err := s.repos.Device.UpdateConnectionStatus(endpointID, "offline"); err != nil {
		log.Printf("❌ Failed to update device status: %v", err)
	}

	// Update connection history
	activeConn, err := s.repos.ConnectionHistory.GetActiveConnection(endpointID)
	if err == nil && activeConn != nil {
		if err := s.repos.ConnectionHistory.UpdateDisconnection(activeConn.ID); err != nil {
			log.Printf("❌ Failed to update connection history: %v", err)
		} else {
			log.Printf("📋 Connection history updated with disconnection")
		}
	}

	// Publish Kafka disconnection event
	if err := s.publishConnectionEvent(endpointID, "disconnected", mtpProtocol); err != nil {
		log.Printf("❌ Failed to publish disconnection event: %v", err)
	}

	return nil
}

// publishConnectionEvent publishes connection state changes to Kafka
func (s *USPCoreService) publishConnectionEvent(endpointID, status, mtpProtocol string) error {
	event := map[string]interface{}{
		"endpoint_id":  endpointID,
		"status":       status, // "connected", "disconnected"
		"mtp_protocol": mtpProtocol,
		"timestamp":    time.Now().Unix(),
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal connection event: %w", err)
	}

	topic := s.config.Kafka.Topics.MTPConnectionEstablished
	if status == "disconnected" {
		topic = s.config.Kafka.Topics.MTPConnectionClosed
	}

	return s.kafkaProducer.PublishRaw(topic, endpointID, eventData)
}

// triggerDeviceDiscovery initiates device discovery after auto-registration
// Sends Get request for Device.DeviceInfo.* to gather device metadata
func (s *USPCoreService) triggerDeviceDiscovery(endpointID, mtpProtocol string) {
	// Wait a bit for the agent to fully establish connection
	time.Sleep(2 * time.Second)

	log.Printf("🔍 Triggering device discovery for %s", endpointID)

	// TODO: Implement Get request for Device.DeviceInfo.* parameters
	// This will populate:
	// - Manufacturer
	// - ModelName
	// - SerialNumber
	// - SoftwareVersion
	// - HardwareVersion
	// - ProductClass

	// For now, just log that discovery should happen
	log.Printf("📝 Device discovery for %s scheduled (implementation pending)", endpointID)
}

// monitorConnections runs periodically to check agent connection health
// Phase 5: Connection monitoring and timeout handling
func (s *USPCoreService) monitorConnections() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	inactiveThreshold := 5 * time.Minute
	disconnectThreshold := 10 * time.Minute

	log.Printf("🔍 Connection monitor started (check interval: 1m, inactive: %v, disconnect: %v)",
		inactiveThreshold, disconnectThreshold)

	for range ticker.C {
		// Get inactive agents from Redis
		inactive, err := s.redisClient.GetInactiveAgents(inactiveThreshold)
		if err != nil {
			log.Printf("❌ Failed to get inactive agents: %v", err)
			continue
		}

		for _, conn := range inactive {
			timeSinceActivity := time.Now().Unix() - conn.LastActivity
			duration := time.Duration(timeSinceActivity) * time.Second

			if duration > disconnectThreshold {
				// Agent hasn't been active for too long - mark as disconnected
				log.Printf("⏱️ Connection timeout for agent %s (inactive for %v)", conn.EndpointID, duration)

				if err := s.handleAgentDisconnect(conn.EndpointID, conn.MTPProtocol); err != nil {
					log.Printf("❌ Failed to handle disconnection for %s: %v", conn.EndpointID, err)
				}

				// Remove from Redis
				if err := s.redisClient.RemoveAgentConnection(conn.EndpointID); err != nil {
					log.Printf("❌ Failed to remove connection from Redis: %v", err)
				}
			} else {
				// Just log warning about inactivity
				log.Printf("⚠️ Agent %s inactive for %v", conn.EndpointID, duration)

				// Update status to "inactive" but don't disconnect yet
				if err := s.redisClient.UpdateConnectionStatus(conn.EndpointID, "inactive"); err != nil {
					log.Printf("❌ Failed to update status to inactive: %v", err)
				}
			}
		}

		// Log connection stats
		count, err := s.redisClient.GetConnectionCount()
		if err == nil {
			log.Printf("📊 Active connections: %d", count)
		}
	}
}

// Helper function to get time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
