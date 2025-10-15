package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/google/uuid"
)

// APIRequest represents a request sent to the data service via Kafka
type APIRequest struct {
	CorrelationID string                 `json:"correlation_id"`
	Operation     string                 `json:"operation"` // e.g., "ListDevices", "GetDevice", "CreateDevice"
	Path          string                 `json:"path"`      // e.g., "/devices", "/devices/:id"
	Method        string                 `json:"method"`    // GET, POST, PUT, DELETE
	Params        map[string]string      `json:"params"`    // URL parameters
	Query         map[string]string      `json:"query"`     // Query parameters
	Body          map[string]interface{} `json:"body"`      // Request body
	Timestamp     time.Time              `json:"timestamp"`
}

// APIResponse represents a response from the data service via Kafka
type APIResponse struct {
	CorrelationID string                 `json:"correlation_id"`
	Status        int                    `json:"status"` // HTTP status code
	Data          map[string]interface{} `json:"data"`   // Response data
	Error         string                 `json:"error"`  // Error message if any
	Timestamp     time.Time              `json:"timestamp"`
}

// RequestResponseClient handles request-response pattern over Kafka
type RequestResponseClient struct {
	producer        *kafka.Producer
	consumer        *kafka.Consumer
	requestTopic    string
	responseTopic   string
	pendingRequests map[string]chan *APIResponse
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// NewRequestResponseClient creates a new Kafka request-response client
func NewRequestResponseClient(brokers []string, requestTopic, responseTopic string) (*RequestResponseClient, error) {
	// Create producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": joinBrokers(brokers),
		"acks":              "all",
		"retries":           3,
		"compression.type":  "snappy",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	// Create consumer with unique group ID for this instance
	consumerGroupID := fmt.Sprintf("api-gateway-%s", uuid.New().String())
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  joinBrokers(brokers),
		"group.id":           consumerGroupID,
		"auto.offset.reset":  "latest", // Only read new responses
		"enable.auto.commit": true,
	})
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Subscribe to response topic
	if err := consumer.Subscribe(responseTopic, nil); err != nil {
		producer.Close()
		consumer.Close()
		return nil, fmt.Errorf("failed to subscribe to response topic: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &RequestResponseClient{
		producer:        producer,
		consumer:        consumer,
		requestTopic:    requestTopic,
		responseTopic:   responseTopic,
		pendingRequests: make(map[string]chan *APIResponse),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Start response listener
	client.wg.Add(1)
	go client.listenForResponses()

	log.Printf("🔄 Request-response client initialized (request: %s, response: %s)", requestTopic, responseTopic)
	return client, nil
}

// SendRequest sends a request and waits for a response
func (c *RequestResponseClient) SendRequest(ctx context.Context, req *APIRequest) (*APIResponse, error) {
	// Generate correlation ID if not set
	if req.CorrelationID == "" {
		req.CorrelationID = uuid.New().String()
	}
	req.Timestamp = time.Now()

	// Create response channel
	responseChan := make(chan *APIResponse, 1)
	c.mu.Lock()
	c.pendingRequests[req.CorrelationID] = responseChan
	c.mu.Unlock()

	// Clean up on exit
	defer func() {
		c.mu.Lock()
		delete(c.pendingRequests, req.CorrelationID)
		c.mu.Unlock()
		close(responseChan)
	}()

	// Serialize request
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send to Kafka
	deliveryChan := make(chan kafka.Event, 1)
	err = c.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &c.requestTopic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(req.CorrelationID),
		Value: reqData,
	}, deliveryChan)

	if err != nil {
		return nil, fmt.Errorf("failed to produce request: %w", err)
	}

	// Wait for delivery confirmation
	select {
	case e := <-deliveryChan:
		m := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return nil, fmt.Errorf("delivery failed: %w", m.TopicPartition.Error)
		}
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("delivery timeout")
	}

	// Wait for response
	select {
	case resp := <-responseChan:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second): // 30 second timeout for response
		return nil, fmt.Errorf("response timeout")
	}
}

// listenForResponses continuously listens for responses
func (c *RequestResponseClient) listenForResponses() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			msg, err := c.consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if err.(kafka.Error).Code() == kafka.ErrTimedOut {
					continue
				}
				log.Printf("Error reading response: %v", err)
				continue
			}

			// Parse response
			var resp APIResponse
			if err := json.Unmarshal(msg.Value, &resp); err != nil {
				log.Printf("Failed to unmarshal response: %v", err)
				continue
			}

			// Find pending request
			c.mu.RLock()
			responseChan, exists := c.pendingRequests[resp.CorrelationID]
			c.mu.RUnlock()

			if exists {
				select {
				case responseChan <- &resp:
				default:
					log.Printf("Warning: response channel full for correlation ID %s", resp.CorrelationID)
				}
			}
		}
	}
}

// Close shuts down the client
func (c *RequestResponseClient) Close() error {
	c.cancel()
	c.wg.Wait()

	if c.producer != nil {
		c.producer.Close()
	}
	if c.consumer != nil {
		c.consumer.Close()
	}

	log.Println("🔄 Request-response client closed")
	return nil
}

// Helper function to join brokers
func joinBrokers(brokers []string) string {
	result := ""
	for i, broker := range brokers {
		if i > 0 {
			result += ","
		}
		result += broker
	}
	return result
}
