package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	kafkago "github.com/confluentinc/confluent-kafka-go/kafka"
	"openusp/pkg/config"
	"openusp/pkg/kafka"
)

const (
	ServiceName    = "mtp-stomp"
	ServiceVersion = "1.0.0"
)

// STOMPMessageHandler defines the interface for handling USP messages via STOMP
type STOMPMessageHandler interface {
	ProcessUSPMessage(data []byte) ([]byte, error)
}

// STOMPBroker implements STOMP Message Transfer Protocol for USP
type STOMPBroker struct {
	brokerURL      string
	conn           net.Conn
	messageHandler STOMPMessageHandler
	destinations   []string
	username       string
	password       string
	mu             sync.RWMutex
	connected      bool
	authenticated  bool
}

// NewSTOMPBrokerWithAuth creates a new STOMP broker connection with specific credentials
func NewSTOMPBrokerWithAuth(brokerURL string, destinations []string, username, password string) (*STOMPBroker, error) {
	// Use provided destinations or default ones
	if len(destinations) == 0 {
		destinations = []string{
			"/queue/usp.inbound",
			"/queue/usp.outbound",
			"/topic/usp.broadcast",
		}
	}

	broker := &STOMPBroker{
		brokerURL:    brokerURL,
		destinations: destinations,
		username:     username,
		password:     password,
	}

	return broker, nil
}

// SetMessageHandler sets the message handler function
func (b *STOMPBroker) SetMessageHandler(handler STOMPMessageHandler) {
	b.messageHandler = handler
}

// Start connects to STOMP broker and starts listening
func (b *STOMPBroker) Start(ctx context.Context) error {
	log.Printf("🔌 Attempting STOMP connection to: %s", b.brokerURL)

	// Extract host:port from broker URL
	if len(b.brokerURL) > 6 && b.brokerURL[:6] == "tcp://" {
		hostPort := b.brokerURL[6:]
		log.Printf("📡 STOMP: Connecting to %s", hostPort)

		// Attempt TCP connection to STOMP broker
		conn, err := net.DialTimeout("tcp", hostPort, 5*time.Second)
		if err != nil {
			log.Printf("❌ STOMP: Failed to connect to broker: %v", err)
			log.Printf("❌ STOMP: Make sure STOMP broker is running and accessible")
			return fmt.Errorf("failed to connect to STOMP broker: %v", err)
		}

		b.mu.Lock()
		b.conn = conn
		b.connected = true
		b.mu.Unlock()

		log.Printf("✅ STOMP: TCP connection established to %s", hostPort)

		// Send STOMP CONNECT frame with credentials from configuration
		connectFrame := fmt.Sprintf("CONNECT\naccept-version:1.2\nhost:/\nlogin:%s\npasscode:%s\n\n\x00", b.username, b.password)
		_, err = conn.Write([]byte(connectFrame))
		if err != nil {
			log.Printf("❌ STOMP: Failed to send CONNECT frame: %v", err)
			return err
		}
		log.Printf("📤 STOMP: Sent CONNECT frame with user: %s", b.username)

		// Start message reading goroutine first to handle CONNECTED response
		go b.readMessages(ctx, conn)

		// Wait for CONNECTED response before subscribing
		log.Printf("⏳ STOMP: Waiting for CONNECTED response from broker...")

		// Wait up to 5 seconds for authentication
		authenticated := false
		for i := 0; i < 50; i++ {
			time.Sleep(100 * time.Millisecond)
			b.mu.RLock()
			authenticated = b.authenticated
			b.mu.RUnlock()
			if authenticated {
				break
			}
		}

		if authenticated {
			log.Printf("✅ STOMP: Authentication successful, proceeding with subscriptions")

			// Subscribe to destinations after connection is established
			for i, dest := range b.destinations {
				subscribeFrame := fmt.Sprintf("SUBSCRIBE\nid:sub-%d\ndestination:%s\nack:auto\n\n\x00", i, dest)
				_, err = conn.Write([]byte(subscribeFrame))
				if err != nil {
					log.Printf("❌ STOMP: Failed to subscribe to %s: %v", dest, err)
				} else {
					log.Printf("📡 STOMP: Successfully subscribed to destination: %s", dest)
					log.Printf("📥 STOMP: Now listening for messages on: %s", dest)
				}
			}
		} else {
			log.Printf("❌ STOMP: Authentication failed or timed out - subscriptions skipped")
		}

		log.Printf("📋 STOMP: Total subscriptions active: %d", len(b.destinations))

	} else {
		log.Printf("❌ STOMP: Invalid broker URL format: %s", b.brokerURL)
		return fmt.Errorf("invalid broker URL format: %s", b.brokerURL)
	}

	// Keep connection alive and handle context cancellation
	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 STOMP broker shutting down...")
			b.Close()
			return nil
		case <-time.After(30 * time.Second):
			// Send heartbeat
			log.Printf("💓 STOMP broker heartbeat")
		}
	}
}

// Send sends a message to a STOMP destination
func (b *STOMPBroker) Send(destination string, payload []byte) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected || b.conn == nil {
		return fmt.Errorf("not connected to STOMP broker")
	}

	// Build STOMP SEND frame per TR-369 Section 4.3.2
	sendFrame := fmt.Sprintf("SEND\ndestination:%s\ncontent-type:application/vnd.bbf.usp.msg\ncontent-length:%d\n\n", destination, len(payload))

	// Debug: Log the STOMP frame headers (without binary payload)
	frameHeader := fmt.Sprintf("SEND\ndestination:%s\ncontent-type:application/vnd.bbf.usp.msg\ncontent-length:%d", destination, len(payload))
	log.Printf("🔍 STOMP SEND Frame:\n%s", frameHeader)

	sendFrame += string(payload) + "\x00"

	_, err := b.conn.Write([]byte(sendFrame))
	if err != nil {
		log.Printf("❌ STOMP: Failed to send message to %s: %v", destination, err)
		return fmt.Errorf("failed to send STOMP message: %v", err)
	}

	log.Printf("📤 STOMP: Sent message (%d bytes) to %s", len(payload), destination)
	return nil
}

// readMessages handles incoming STOMP messages from broker
func (b *STOMPBroker) readMessages(ctx context.Context, conn net.Conn) {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 STOMP message reader shutting down...")
			return
		default:
			// Set read timeout
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))

			n, err := conn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout is normal, continue reading
					continue
				}
				// Check if connection is being closed (expected during shutdown)
				if strings.Contains(err.Error(), "use of closed network connection") {
					log.Printf("🛑 STOMP: Connection closed, stopping message reader")
					return
				}
				log.Printf("❌ STOMP: Error reading from connection: %v", err)
				return
			}

			if n > 0 {
				message := string(buffer[:n])

				// Parse STOMP frame type
				if len(message) > 7 && message[:7] == "MESSAGE" {
					log.Printf("📥 STOMP: Received MESSAGE frame (%d bytes)", n)

					// Extract content-length header
					var contentLength int
					contentLengthIdx := strings.Index(message, "content-length:")
					if contentLengthIdx >= 0 {
						endIdx := strings.Index(message[contentLengthIdx:], "\n")
						if endIdx >= 0 {
							contentLengthLine := message[contentLengthIdx : contentLengthIdx+endIdx]
							fmt.Sscanf(contentLengthLine, "content-length:%d", &contentLength)
						}
					}

					// Find the blank line that separates headers from body
					bodyStart := strings.Index(message, "\n\n")
					if bodyStart >= 0 {
						bodyStart += 2 // Skip past the \n\n

						var uspPayload []byte
						if contentLength > 0 {
							uspPayload = buffer[bodyStart : bodyStart+contentLength]
						} else {
							bodyEnd := strings.IndexByte(message[bodyStart:], 0)
							if bodyEnd >= 0 {
								uspPayload = []byte(message[bodyStart : bodyStart+bodyEnd])
							}
						}

						if len(uspPayload) > 0 {
							if b.messageHandler != nil {
								response, err := b.messageHandler.ProcessUSPMessage(uspPayload)
								if err != nil {
									log.Printf("❌ STOMP: Error processing USP message: %v", err)
								} else if len(response) > 0 {
									err = b.Send("/queue/usp.outbound", response)
									if err != nil {
										log.Printf("❌ STOMP: Failed to send response: %v", err)
									}
								}
							}
						}
					}
				} else if len(message) > 9 && message[:9] == "CONNECTED" {
					log.Printf("✅ STOMP: Received CONNECTED confirmation from broker")

					b.mu.Lock()
					b.authenticated = true
					b.mu.Unlock()
				} else if len(message) > 5 && message[:5] == "ERROR" {
					log.Printf("❌ STOMP: Received ERROR frame from broker")
				}
			}
		}
	}
}

// Close disconnects from the STOMP broker
func (b *STOMPBroker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.conn != nil {
		b.conn.Close()
	}
	b.connected = false
	log.Printf("✅ STOMP broker disconnected")
}

// STOMPMTPService handles STOMP transport for USP messages
type STOMPMTPService struct {
	config        *config.Config
	stompBroker   *STOMPBroker
	kafkaClient   *kafka.Client
	kafkaProducer *kafka.Producer
	kafkaConsumer *kafka.Consumer
}

func main() {
	log.Printf("🚀 Starting %s v%s", ServiceName, ServiceVersion)

	// Load configuration from YAML
	cfg := config.Load()

	// Initialize Kafka client
	kafkaClient, err := kafka.NewClient(&cfg.Kafka)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Kafka client: %v", err)
	}
	defer kafkaClient.Close()

	// Create Kafka producer
	kafkaProducer, err := kafka.NewProducer(kafkaClient)
	if err != nil {
		log.Fatalf("❌ Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Create Kafka consumer for outbound messages
	kafkaConsumer, err := kafka.NewConsumer(&cfg.Kafka, "mtp-stomp")
	if err != nil {
		log.Fatalf("❌ Failed to create Kafka consumer: %v", err)
	}
	defer kafkaConsumer.Close()

	// Ensure topics exist
	topicsManager := kafka.NewTopicsManager(kafkaClient, &cfg.Kafka.Topics)
	if err := topicsManager.EnsureAllTopicsExist(); err != nil {
		log.Printf("⚠️ Failed to ensure topics exist: %v", err)
	}

	// Create service instance
	svc := &STOMPMTPService{
		config:        cfg,
		kafkaClient:   kafkaClient,
		kafkaProducer: kafkaProducer,
		kafkaConsumer: kafkaConsumer,
	}

	// Initialize STOMP broker
	if err := svc.initSTOMPBroker(); err != nil {
		log.Fatalf("❌ Failed to initialize STOMP broker: %v", err)
	}

	// Setup Kafka consumer for outbound messages
	if err := svc.setupKafkaConsumer(); err != nil {
		log.Fatalf("❌ Failed to setup Kafka consumer: %v", err)
	}

	// Validate and get ports from configuration - no hardcoded defaults
	healthPort := svc.config.MTPStompHealthPort
	if healthPort == 0 {
		log.Fatalf("Health port for mtp-stomp not configured in openusp.yml (ports.health.mtp_stomp)")
	}

	metricsPort := svc.config.MTPStompMetricsPort
	if metricsPort == 0 {
		log.Fatalf("Metrics port for mtp-stomp not configured in openusp.yml (ports.metrics.mtp_stomp)")
	}

	log.Printf("📋 Using ports - Health: %d, Metrics: %d", healthPort, metricsPort)

	// Start health endpoint
	go func() {
		healthMux := http.NewServeMux()
		healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy","service":"` + ServiceName + `","version":"` + ServiceVersion + `"}`))
		})
		log.Printf("🏥 Health endpoint listening on port %d", healthPort)
		http.ListenAndServe(fmt.Sprintf(":%d", healthPort), healthMux)
	}()

	// Start metrics endpoint
	go func() {
		metricsMux := http.NewServeMux()
		metricsMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# HELP mtp_stomp_messages_total Total messages processed\n# TYPE mtp_stomp_messages_total counter\nmtp_stomp_messages_total 0\n"))
		})
		log.Printf("📊 Metrics endpoint listening on port %d", metricsPort)
		http.ListenAndServe(fmt.Sprintf(":%d", metricsPort), metricsMux)
	}()

	// TODO: Start gRPC server on grpcPort
	// This will be implemented after proto definitions are created

	// Start STOMP broker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := svc.stompBroker.Start(ctx); err != nil {
			log.Printf("❌ STOMP broker error: %v", err)
		}
	}()

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── STOMP Broker: %s", cfg.MTP.STOMP.BrokerURL)
	log.Printf("   └── STOMP Inbound Queue: %s", cfg.MTP.STOMP.Destinations.Inbound)
	log.Printf("   └── STOMP Outbound Queue: %s", cfg.MTP.STOMP.Destinations.Outbound)
	log.Printf("   └── Health Port: %d", healthPort)
	log.Printf("   └── Kafka Brokers: %v", cfg.Kafka.Brokers)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("🛑 Shutting down %s...", ServiceName)
	cancel()

	// Cleanup
	if svc.stompBroker != nil {
		svc.stompBroker.Close()
	}

	log.Printf("✅ %s stopped gracefully", ServiceName)
}

// initSTOMPBroker initializes STOMP broker connection
func (s *STOMPMTPService) initSTOMPBroker() error {
	// Get STOMP configuration from YAML
	stompConfig := s.config.MTP.STOMP

	// Controller subscribes to the inbound queue to receive messages from agents
	destinations := []string{stompConfig.Destinations.Inbound}

	log.Printf("🔌 Creating STOMP broker connection...")
	log.Printf("   └── Broker URL: %s", stompConfig.BrokerURL)
	log.Printf("   └── Username: %s", stompConfig.Username)
	log.Printf("   └── Subscriptions: %v", destinations)

	broker, err := NewSTOMPBrokerWithAuth(
		stompConfig.BrokerURL,
		destinations,
		stompConfig.Username,
		stompConfig.Password,
	)
	if err != nil {
		return fmt.Errorf("failed to create STOMP broker: %w", err)
	}

	// Set message handler to process incoming STOMP messages
	broker.SetMessageHandler(s)
	s.stompBroker = broker

	log.Printf("✅ STOMP broker initialized")
	return nil
}

// setupKafkaConsumer subscribes to outbound messages from Kafka
func (s *STOMPMTPService) setupKafkaConsumer() error {
	log.Printf("🔌 Setting up Kafka consumer for outbound messages...")

	// Subscribe to USP outbound topic
	topics := []string{s.config.Kafka.Topics.USPMessagesOutbound}

	handler := func(msg *kafkago.Message) error {
		log.Printf("📥 Kafka: Received outbound USP message (%d bytes) from topic: %s", len(msg.Value), *msg.TopicPartition.Topic)

		// Extract STOMP destination from message metadata or use default
		// Send to outbound queue for agents
		destination := s.config.MTP.STOMP.Destinations.Outbound

		// Forward message to STOMP broker
		err := s.stompBroker.Send(destination, msg.Value)
		if err != nil {
			log.Printf("❌ Failed to send message to STOMP destination %s: %v", destination, err)
			return fmt.Errorf("failed to send to STOMP: %w", err)
		}

		log.Printf("✅ Forwarded message to STOMP destination: %s", destination)
		return nil
	}

	if err := s.kafkaConsumer.Subscribe(topics, handler); err != nil {
		return fmt.Errorf("failed to subscribe to Kafka topics: %w", err)
	}

	log.Printf("✅ Subscribed to Kafka topic: %s", s.config.Kafka.Topics.USPMessagesOutbound)

	// Start consuming in background
	go func() {
		log.Printf("🔄 Starting Kafka consumer for outbound messages...")
		s.kafkaConsumer.Start()
	}()

	return nil
}

// ProcessUSPMessage implements the MessageProcessor interface
// This is called by the STOMP broker when a message is received
func (s *STOMPMTPService) ProcessUSPMessage(data []byte) ([]byte, error) {
	log.Printf("📥 STOMP: Received USP TR-369 protobuf message (%d bytes)", len(data))

	// Publish raw TR-369 protobuf bytes directly to Kafka
	// The USP service will parse the protobuf Record
	err := s.kafkaProducer.PublishRaw(
		s.config.Kafka.Topics.USPMessagesInbound,
		"", // key
		data, // raw TR-369 protobuf bytes
	)

	if err != nil {
		log.Printf("❌ STOMP: Failed to publish to Kafka: %v", err)
		return nil, fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	log.Printf("✅ STOMP: USP TR-369 message published to Kafka topic: %s", s.config.Kafka.Topics.USPMessagesInbound)

	// In event-driven architecture, we don't wait for synchronous response
	return nil, nil
}
