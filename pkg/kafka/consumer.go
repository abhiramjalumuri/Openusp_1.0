package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"openusp/pkg/config"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// MessageHandler is a function type for handling consumed messages
type MessageHandler func(msg *kafka.Message) error

// Consumer wraps Kafka consumer with convenience methods
type Consumer struct {
	config   *config.KafkaConfig
	consumer *kafka.Consumer
	handlers map[string]MessageHandler
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *config.KafkaConfig, groupID string) (*Consumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kafka config is required")
	}

	if groupID == "" {
		groupID = cfg.ConsumerConfig.GroupIDPrefix + "-default"
	} else if cfg.ConsumerConfig.GroupIDPrefix != "" {
		groupID = cfg.ConsumerConfig.GroupIDPrefix + "-" + groupID
	}

	configMap := kafka.ConfigMap{
		"bootstrap.servers":       joinStrings(cfg.Brokers, ","),
		"group.id":                groupID,
		"auto.offset.reset":       cfg.ConsumerConfig.AutoOffsetReset,
		"enable.auto.commit":      cfg.ConsumerConfig.EnableAutoCommit,
		"auto.commit.interval.ms": cfg.ConsumerConfig.AutoCommitIntervalMs,
		"session.timeout.ms":      cfg.ConsumerConfig.SessionTimeoutMs,
	}

	// Add optional configuration with defaults if not set
	if cfg.ConsumerConfig.HeartbeatIntervalMs > 0 {
		configMap["heartbeat.interval.ms"] = cfg.ConsumerConfig.HeartbeatIntervalMs
	}
	if cfg.ConsumerConfig.MaxPollIntervalMs > 0 {
		configMap["max.poll.interval.ms"] = cfg.ConsumerConfig.MaxPollIntervalMs
	}
	if cfg.ConsumerConfig.FetchMinBytes > 0 {
		configMap["fetch.min.bytes"] = cfg.ConsumerConfig.FetchMinBytes
	}
	if cfg.ConsumerConfig.FetchMaxWaitMs > 0 {
		configMap["fetch.wait.max.ms"] = cfg.ConsumerConfig.FetchMaxWaitMs
	}

	consumer, err := kafka.NewConsumer(&configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Consumer{
		config:   cfg,
		consumer: consumer,
		handlers: make(map[string]MessageHandler),
		ctx:      ctx,
		cancel:   cancel,
	}

	log.Printf("✅ Kafka consumer initialized (group: %s)", groupID)

	return c, nil
}

// Subscribe subscribes to one or more topics
func (c *Consumer) Subscribe(topics []string, handler MessageHandler) error {
	if len(topics) == 0 {
		return fmt.Errorf("at least one topic is required")
	}

	err := c.consumer.SubscribeTopics(topics, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	// Register handler for all subscribed topics
	for _, topic := range topics {
		c.handlers[topic] = handler
	}

	log.Printf("✅ Subscribed to topics: %v", topics)
	return nil
}

// Start starts consuming messages in a background goroutine
func (c *Consumer) Start() {
	go c.consumeLoop()
}

// consumeLoop is the main consumer loop
func (c *Consumer) consumeLoop() {
	log.Printf("🚀 Starting consumer loop...")

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("🛑 Consumer loop stopped")
			return
		default:
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				// Timeout is expected when no messages are available
				if err.(kafka.Error).Code() != kafka.ErrTimedOut {
					log.Printf("⚠️ Consumer error: %v", err)
				}
				continue
			}

			c.handleMessage(msg)
		}
	}
}

// handleMessage processes a consumed message
func (c *Consumer) handleMessage(msg *kafka.Message) {
	topic := *msg.TopicPartition.Topic

	log.Printf("📥 Received message from topic: %s (partition: %d, offset: %d, size: %d bytes)",
		topic,
		msg.TopicPartition.Partition,
		msg.TopicPartition.Offset,
		len(msg.Value))

	// Find handler for this topic
	handler, exists := c.handlers[topic]
	if !exists {
		log.Printf("⚠️ No handler registered for topic: %s", topic)
		return
	}

	// Call handler
	err := handler(msg)
	if err != nil {
		log.Printf("❌ Error handling message from topic %s: %v", topic, err)
	}
}

// ConsumeUSPMessage is a helper to parse USP message events
func ConsumeUSPMessage(msg *kafka.Message) (*USPMessageEvent, error) {
	var event USPMessageEvent
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP message event: %w", err)
	}
	return &event, nil
}

// ConsumeDeviceEvent is a helper to parse device events
func ConsumeDeviceEvent(msg *kafka.Message) (*DeviceEvent, error) {
	var event DeviceEvent
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal device event: %w", err)
	}
	return &event, nil
}

// ConsumeParameterEvent is a helper to parse parameter events
func ConsumeParameterEvent(msg *kafka.Message) (*ParameterEvent, error) {
	var event ParameterEvent
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameter event: %w", err)
	}
	return &event, nil
}

// ConsumeCWMPMessage is a helper to parse CWMP message events
func ConsumeCWMPMessage(msg *kafka.Message) (*CWMPMessageEvent, error) {
	var event CWMPMessageEvent
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal CWMP message event: %w", err)
	}
	return &event, nil
}

// ConsumeAgentStatus is a helper to parse agent status events
func ConsumeAgentStatus(msg *kafka.Message) (*AgentStatusEvent, error) {
	var event AgentStatusEvent
	err := json.Unmarshal(msg.Value, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent status event: %w", err)
	}
	return &event, nil
}

// CommitMessage manually commits a message offset
func (c *Consumer) CommitMessage(msg *kafka.Message) error {
	_, err := c.consumer.CommitMessage(msg)
	return err
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	log.Printf("🛑 Stopping consumer...")
	c.cancel()
}

// Close closes the consumer and releases resources
func (c *Consumer) Close() {
	c.Stop()
	if c.consumer != nil {
		err := c.consumer.Close()
		if err != nil {
			log.Printf("⚠️ Error closing consumer: %v", err)
		} else {
			log.Printf("✅ Consumer closed")
		}
	}
}
