package kafka

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// Producer wraps Kafka producer with convenience methods
type Producer struct {
	client   *Client
	producer *kafka.Producer
}

// NewProducer creates a new Kafka producer
func NewProducer(client *Client) (*Producer, error) {
	if client == nil {
		return nil, fmt.Errorf("kafka client is required")
	}

	return &Producer{
		client:   client,
		producer: client.GetProducer(),
	}, nil
}

// PublishEvent publishes an event to a topic
func (p *Producer) PublishEvent(topic string, event interface{}) error {
	// Serialize event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return p.PublishRaw(topic, "", data)
}

// PublishRaw publishes raw bytes to a topic
func (p *Producer) PublishRaw(topic string, key string, data []byte) error {
	if topic == "" {
		return fmt.Errorf("topic is required")
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: data,
	}

	if key != "" {
		msg.Key = []byte(key)
	}

	// Produce message asynchronously
	err := p.producer.Produce(msg, nil)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	log.Printf("📤 Published message to topic: %s (size: %d bytes)", topic, len(data))
	return nil
}

// PublishUSPMessage publishes a USP message event
func (p *Producer) PublishUSPMessage(topic string, endpointID, messageID, messageType string, payload []byte, mtpProtocol string) error {
	return p.PublishUSPMessageWithDestination(topic, endpointID, messageID, messageType, payload, mtpProtocol, MTPDestination{})
}

// PublishUSPMessageWithDestination publishes a USP message event with MTP routing information
func (p *Producer) PublishUSPMessageWithDestination(topic string, endpointID, messageID, messageType string, payload []byte, mtpProtocol string, destination MTPDestination) error {
	event := USPMessageEvent{
		BaseEvent:      NewBaseEvent(EventUSPMessageInbound, "mtp-service"),
		EndpointID:     endpointID,
		MessageID:      messageID,
		MessageType:    messageType,
		Payload:        payload,
		MTPProtocol:    mtpProtocol,
		MTPDestination: destination,
	}

	return p.PublishEvent(topic, event)
}

// PublishCWMPMessage publishes a CWMP message event
func (p *Producer) PublishCWMPMessage(topic string, deviceID, messageID, messageType string, payload []byte, mtpProtocol string) error {
	event := CWMPMessageEvent{
		BaseEvent:   NewBaseEvent(EventCWMPMessageInbound, "mtp-http"),
		DeviceID:    deviceID,
		MessageType: messageType,
		Payload:     payload,
		SessionID:   messageID, // Use messageID as session ID for now
	}

	return p.PublishEvent(topic, event)
}

// PublishDeviceEvent publishes a device lifecycle event
func (p *Producer) PublishDeviceEvent(topic string, eventType EventType, deviceID, endpointID, protocol string, data map[string]interface{}) error {
	event := DeviceEvent{
		BaseEvent:  NewBaseEvent(eventType, "usp-service"),
		DeviceID:   deviceID,
		EndpointID: endpointID,
		Protocol:   protocol,
		Data:       data,
	}

	// Extract common device fields from data map to top-level fields for easier consumption
	if manufacturer, ok := data["manufacturer"].(string); ok {
		event.Manufacturer = manufacturer
	}
	if modelName, ok := data["model_name"].(string); ok {
		event.ModelName = modelName
	}
	if serialNumber, ok := data["serial_number"].(string); ok {
		event.SerialNumber = serialNumber
	}
	if productClass, ok := data["product_class"].(string); ok {
		event.ProductClass = productClass
	}
	if status, ok := data["status"].(string); ok {
		event.Status = status
	}

	return p.PublishEvent(topic, event)
}

// PublishParameterUpdate publishes a parameter update event
func (p *Producer) PublishParameterUpdate(topic string, deviceID, endpointID, paramPath string, oldValue, newValue interface{}) error {
	event := ParameterEvent{
		BaseEvent:     NewBaseEvent(EventParameterUpdated, "data-service"),
		DeviceID:      deviceID,
		EndpointID:    endpointID,
		ParameterPath: paramPath,
		OldValue:      oldValue,
		NewValue:      newValue,
	}

	return p.PublishEvent(topic, event)
}

// PublishAgentStatus publishes an agent status event
func (p *Producer) PublishAgentStatus(topic string, agentID, agentType, status string, details string) error {
	event := AgentStatusEvent{
		BaseEvent: NewBaseEvent(EventAgentStatusUpdated, fmt.Sprintf("%s-agent", agentType)),
		AgentID:   agentID,
		AgentType: agentType,
		Status:    status,
		Details:   details,
	}

	return p.PublishEvent(topic, event)
}

// PublishMTPConnection publishes an MTP connection event
func (p *Producer) PublishMTPConnection(topic string, eventType EventType, connID, protocol, endpointID, remoteAddr, status string) error {
	event := MTPConnectionEvent{
		BaseEvent:    NewBaseEvent(eventType, fmt.Sprintf("mtp-%s", protocol)),
		ConnectionID: connID,
		Protocol:     protocol,
		EndpointID:   endpointID,
		RemoteAddr:   remoteAddr,
		Status:       status,
	}

	return p.PublishEvent(topic, event)
}

// Flush waits for all pending messages to be delivered
func (p *Producer) Flush(timeoutMs int) int {
	return p.producer.Flush(timeoutMs)
}

// Close closes the producer
func (p *Producer) Close() {
	if p.producer != nil {
		remaining := p.producer.Flush(5000)
		if remaining > 0 {
			log.Printf("⚠️ %d messages not delivered before close", remaining)
		}
	}
}
