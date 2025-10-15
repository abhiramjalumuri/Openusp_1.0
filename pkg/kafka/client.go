package kafka

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"openusp/pkg/config"
)

// Client wraps Kafka producer and consumer functionality
type Client struct {
	config   *config.KafkaConfig
	producer *kafka.Producer
	admin    *kafka.AdminClient
}

// NewClient creates a new Kafka client
func NewClient(cfg *config.KafkaConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kafka config is required")
	}

	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one Kafka broker is required")
	}

	client := &Client{
		config: cfg,
	}

	// Initialize producer
	producer, err := client.createProducer()
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}
	client.producer = producer

	// Initialize admin client for topic management
	admin, err := client.createAdminClient()
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create admin client: %w", err)
	}
	client.admin = admin

	log.Printf("✅ Kafka client initialized successfully (brokers: %v)", cfg.Brokers)

	return client, nil
}

// createProducer creates a Kafka producer with configured settings
func (c *Client) createProducer() (*kafka.Producer, error) {
	configMap := kafka.ConfigMap{
		"bootstrap.servers": joinStrings(c.config.Brokers, ","),
		"acks":              c.config.ProducerConfig.Acks,
		"compression.type":  c.config.ProducerConfig.Compression,
		"retries":           c.config.ProducerConfig.MaxRetries,
		"retry.backoff.ms":  c.config.ProducerConfig.RetryBackoffMs,
		"batch.size":        c.config.ProducerConfig.BatchSize,
		"linger.ms":         c.config.ProducerConfig.LingerMs,
		// Note: buffer.memory is Java-specific, Go uses queue.buffering.max.kbytes (default: 1GB)
	}

	producer, err := kafka.NewProducer(&configMap)
	if err != nil {
		return nil, err
	}

	// Start delivery report handler in background
	go c.handleDeliveryReports(producer)

	return producer, nil
}

// createAdminClient creates a Kafka admin client
func (c *Client) createAdminClient() (*kafka.AdminClient, error) {
	configMap := kafka.ConfigMap{
		"bootstrap.servers": joinStrings(c.config.Brokers, ","),
	}

	return kafka.NewAdminClient(&configMap)
}

// handleDeliveryReports processes producer delivery reports
func (c *Client) handleDeliveryReports(producer *kafka.Producer) {
	for e := range producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Printf("❌ Delivery failed: %v", ev.TopicPartition.Error)
			} else {
				log.Printf("✅ Message delivered to %v [%d] at offset %v",
					*ev.TopicPartition.Topic,
					ev.TopicPartition.Partition,
					ev.TopicPartition.Offset)
			}
		case kafka.Error:
			log.Printf("⚠️ Kafka error: %v", ev)
		}
	}
}

// GetProducer returns the underlying Kafka producer
func (c *Client) GetProducer() *kafka.Producer {
	return c.producer
}

// GetAdminClient returns the underlying Kafka admin client
func (c *Client) GetAdminClient() *kafka.AdminClient {
	return c.admin
}

// EnsureTopicsExist creates topics if they don't exist
func (c *Client) EnsureTopicsExist(topics []string) error {
	if c.admin == nil {
		return fmt.Errorf("admin client not initialized")
	}

	// Get existing topics
	metadata, err := c.admin.GetMetadata(nil, false, int(c.config.AdminConfig.TimeoutMs))
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	existingTopics := make(map[string]bool)
	for topic := range metadata.Topics {
		existingTopics[topic] = true
	}

	// Prepare topics to create
	var topicsToCreate []kafka.TopicSpecification
	for _, topic := range topics {
		if !existingTopics[topic] {
			topicsToCreate = append(topicsToCreate, kafka.TopicSpecification{
				Topic:             topic,
				NumPartitions:     c.config.AdminConfig.NumPartitions,
				ReplicationFactor: c.config.AdminConfig.ReplicationFactor,
			})
		}
	}

	if len(topicsToCreate) == 0 {
		log.Printf("✅ All topics already exist")
		return nil
	}

	// Create topics
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.config.AdminConfig.TimeoutMs)*time.Millisecond)
	defer cancel()

	results, err := c.admin.CreateTopics(ctx, topicsToCreate)
	if err != nil {
		return fmt.Errorf("failed to create topics: %w", err)
	}

	// Check results
	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError && result.Error.Code() != kafka.ErrTopicAlreadyExists {
			log.Printf("⚠️ Failed to create topic %s: %v", result.Topic, result.Error)
		} else {
			log.Printf("✅ Topic created: %s", result.Topic)
		}
	}

	return nil
}

// Close closes the Kafka client and releases resources
func (c *Client) Close() {
	if c.producer != nil {
		// Flush pending messages
		remaining := c.producer.Flush(5000) // 5 second timeout
		if remaining > 0 {
			log.Printf("⚠️ %d messages were not delivered before close", remaining)
		}
		c.producer.Close()
		log.Printf("✅ Kafka producer closed")
	}

	if c.admin != nil {
		c.admin.Close()
		log.Printf("✅ Kafka admin client closed")
	}
}

// Ping checks if the Kafka broker is reachable
func (c *Client) Ping() error {
	if c.admin == nil {
		return fmt.Errorf("admin client not initialized")
	}

	_, err := c.admin.GetMetadata(nil, false, 5000) // 5 second timeout
	if err != nil {
		return fmt.Errorf("kafka broker unreachable: %w", err)
	}

	return nil
}

// joinStrings joins a slice of strings with a delimiter
func joinStrings(strs []string, delim string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += delim + strs[i]
	}
	return result
}
