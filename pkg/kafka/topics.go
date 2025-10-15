package kafka

import (
	"fmt"
	"log"

	"openusp/pkg/config"
)

// TopicsManager helps manage Kafka topics
type TopicsManager struct {
	client *Client
	config *config.KafkaTopics
}

// NewTopicsManager creates a new topics manager
func NewTopicsManager(client *Client, topicsConfig *config.KafkaTopics) *TopicsManager {
	return &TopicsManager{
		client: client,
		config: topicsConfig,
	}
}

// EnsureAllTopicsExist creates all configured topics if they don't exist
func (tm *TopicsManager) EnsureAllTopicsExist() error {
	topics := tm.GetAllTopics()

	log.Printf("📋 Ensuring %d topics exist...", len(topics))

	err := tm.client.EnsureTopicsExist(topics)
	if err != nil {
		return fmt.Errorf("failed to ensure topics exist: %w", err)
	}

	log.Printf("✅ All topics verified")
	return nil
}

// GetAllTopics returns a list of all configured topic names
func (tm *TopicsManager) GetAllTopics() []string {
	return []string{
		// USP topics
		tm.config.USPMessagesInbound,
		tm.config.USPMessagesOutbound,
		tm.config.USPRecordsParsed,

		// Data topics
		tm.config.DataDeviceCreated,
		tm.config.DataDeviceUpdated,
		tm.config.DataDeviceDeleted,
		tm.config.DataParameterUpdated,
		tm.config.DataObjectCreated,
		tm.config.DataAlertCreated,

		// CWMP topics
		tm.config.CWMPMessagesInbound,
		tm.config.CWMPMessagesOutbound,

		// Agent topics
		tm.config.AgentStatusUpdated,
		tm.config.AgentOnboardingStarted,
		tm.config.AgentOnboardingCompleted,

		// MTP topics
		tm.config.MTPConnectionEstablished,
		tm.config.MTPConnectionClosed,

		// API Gateway topics
		tm.config.APIRequest,
		tm.config.APIResponse,
	}
}

// GetUSPTopics returns USP-related topics
func (tm *TopicsManager) GetUSPTopics() []string {
	return []string{
		tm.config.USPMessagesInbound,
		tm.config.USPMessagesOutbound,
		tm.config.USPRecordsParsed,
	}
}

// GetDataTopics returns data-related topics
func (tm *TopicsManager) GetDataTopics() []string {
	return []string{
		tm.config.DataDeviceCreated,
		tm.config.DataDeviceUpdated,
		tm.config.DataDeviceDeleted,
		tm.config.DataParameterUpdated,
		tm.config.DataObjectCreated,
		tm.config.DataAlertCreated,
	}
}

// GetCWMPTopics returns CWMP-related topics
func (tm *TopicsManager) GetCWMPTopics() []string {
	return []string{
		tm.config.CWMPMessagesInbound,
		tm.config.CWMPMessagesOutbound,
	}
}

// GetAgentTopics returns agent-related topics
func (tm *TopicsManager) GetAgentTopics() []string {
	return []string{
		tm.config.AgentStatusUpdated,
		tm.config.AgentOnboardingStarted,
		tm.config.AgentOnboardingCompleted,
	}
}

// GetMTPTopics returns MTP-related topics
func (tm *TopicsManager) GetMTPTopics() []string {
	return []string{
		tm.config.MTPConnectionEstablished,
		tm.config.MTPConnectionClosed,
	}
}

// ValidateTopics checks if all required topics are configured
func (tm *TopicsManager) ValidateTopics() error {
	requiredTopics := map[string]string{
		"usp_messages_inbound":   tm.config.USPMessagesInbound,
		"usp_messages_outbound":  tm.config.USPMessagesOutbound,
		"data_device_created":    tm.config.DataDeviceCreated,
		"data_device_updated":    tm.config.DataDeviceUpdated,
		"data_parameter_updated": tm.config.DataParameterUpdated,
	}

	for name, topic := range requiredTopics {
		if topic == "" {
			return fmt.Errorf("required topic %s is not configured", name)
		}
	}

	return nil
}
