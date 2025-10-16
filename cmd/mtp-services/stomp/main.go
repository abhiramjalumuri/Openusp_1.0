package main

import (
	"context"
	"encoding/json"
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

	confluentkafka "github.com/confluentinc/confluent-kafka-go/kafka"

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

// IsConnected returns true if STOMP broker is connected
func (b *STOMPBroker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
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

	// Setup Kafka consumer for outbound messages (responses from USP service)
	if err := svc.setupKafkaConsumer(); err != nil {
		log.Fatalf("❌ Failed to setup Kafka consumer: %v", err)
	}

	log.Printf("✅ %s started successfully", ServiceName)
	log.Printf("   └── STOMP Broker: %s", cfg.MTP.STOMP.BrokerURL)
	log.Printf("   └── STOMP Inbound Queue: %s", cfg.MTP.STOMP.Destinations.Inbound)
	log.Printf("   └── STOMP Outbound Queue: %s", cfg.MTP.STOMP.Destinations.Outbound)
	log.Printf("   └── Kafka Inbound Topic: %s", cfg.Kafka.Topics.USPMessagesInbound)
	log.Printf("   └── Kafka Outbound Topic: %s", cfg.Kafka.Topics.USPMessagesOutbound)
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

	broker := &STOMPBroker{
		brokerURL:    stompConfig.BrokerURL,
		destinations: destinations,
		username:     stompConfig.Username,
		password:     stompConfig.Password,
	}

	// Set message handler to process incoming STOMP messages
	broker.SetMessageHandler(s)
	s.stompBroker = broker

	log.Printf("✅ STOMP broker initialized")
	return nil
}

// NewSTOMPBrokerWithAuth creates a new STOMP broker connection with specific credentials
func NewSTOMPBrokerWithAuth(brokerURL string, destinations []string, username, password string) (*STOMPBroker, error) {
	// Use provided destinations or default ones
	if len(destinations) == 0 {
		destinations = []string{
			"/queue/usp.controller",
			"/queue/usp.agent",
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

		// Attempt TCP connection to RabbitMQ STOMP port
		conn, err := net.DialTimeout("tcp", hostPort, 5*time.Second)
		if err != nil {
			log.Printf("❌ STOMP: Failed to connect to RabbitMQ: %v", err)
			log.Printf("❌ STOMP: Make sure RabbitMQ is running with STOMP plugin enabled")
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
		log.Printf("⏳ STOMP: Waiting for CONNECTED response from RabbitMQ...")

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

// readMessages handles incoming STOMP messages from RabbitMQ
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
					log.Printf("🎯 STOMP: Processing agent message from /queue/usp.controller")

					// Extract content-length header - CRITICAL for binary data
					var contentLength int
					contentLengthIdx := strings.Index(message, "content-length:")
					if contentLengthIdx >= 0 {
						endIdx := strings.Index(message[contentLengthIdx:], "\n")
						if endIdx >= 0 {
							contentLengthLine := message[contentLengthIdx : contentLengthIdx+endIdx]
							log.Printf("📏 STOMP: Frame header: %s", contentLengthLine)
							// Parse the number
							fmt.Sscanf(contentLengthLine, "content-length:%d", &contentLength)
						}
					}

					// Extract USP payload from STOMP MESSAGE frame
					// Find the blank line that separates headers from body
					bodyStart := strings.Index(message, "\n\n")
					if bodyStart >= 0 {
						bodyStart += 2 // Skip past the \n\n

						var uspPayload []byte
						if contentLength > 0 {
							// Use content-length if available (REQUIRED for binary data per STOMP spec)
							uspPayload = buffer[bodyStart : bodyStart+contentLength]
							log.Printf("📦 STOMP: Extracted USP payload (%d bytes) using content-length", len(uspPayload))
						} else {
							// Fallback: Find the NULL terminator (for text payloads)
							bodyEnd := strings.IndexByte(message[bodyStart:], 0)
							if bodyEnd >= 0 {
								uspPayload = []byte(message[bodyStart : bodyStart+bodyEnd])
								log.Printf("📦 STOMP: Extracted USP payload (%d bytes) using NULL terminator", len(uspPayload))
							}
						}

						if len(uspPayload) > 0 {
							// Call message handler if set
							if b.messageHandler != nil {
								log.Printf("🔄 STOMP: Calling message handler to process USP message")
								response, err := b.messageHandler.ProcessUSPMessage(uspPayload)
								if err != nil {
									log.Printf("❌ STOMP: Error processing USP message: %v", err)
								} else if len(response) > 0 {
									log.Printf("📤 STOMP: Sending response (%d bytes) to /queue/usp.agent", len(response))
									// Send response back to agent
									err = b.Send("/queue/usp.agent", response)
									if err != nil {
										log.Printf("❌ STOMP: Failed to send response: %v", err)
									} else {
										log.Printf("✅ STOMP: Response sent successfully to agent")
									}
								} else {
									log.Printf("ℹ️  STOMP: No response payload from handler")
								}
							} else {
								log.Printf("⚠️  STOMP: No message handler set - message not processed")
							}
						}
					}
				} else if len(message) > 9 && message[:9] == "CONNECTED" {
					log.Printf("✅ STOMP: Received CONNECTED confirmation from RabbitMQ")
					log.Printf("🎉 STOMP: Successfully authenticated with RabbitMQ")

					// Set authenticated flag
					b.mu.Lock()
					b.authenticated = true
					b.mu.Unlock()
				} else if len(message) > 5 && message[:5] == "ERROR" {
					log.Printf("❌ STOMP: Received ERROR frame from RabbitMQ")

					// Parse STOMP ERROR frame for better diagnostics
					lines := strings.Split(message, "\n")
					subscriptionID := ""
					for i, line := range lines {
						if strings.Contains(line, "message:") {
							log.Printf("❌ STOMP Error Message: %s", line)
						} else if strings.Contains(line, "content-length:") || strings.Contains(line, "content-type:") {
							log.Printf("📋 STOMP Error Header: %s", line)
						} else if strings.HasPrefix(line, "subscription:") {
							subscriptionID = strings.TrimPrefix(line, "subscription:")
							log.Printf("📋 STOMP Error Detail: %s", line)
						} else if i < 10 && line != "" && line != "ERROR" {
							log.Printf("📋 STOMP Error Detail: %s", line)
						}
					}

					// Handle subscription cancellation
					if strings.Contains(message, "cancelled subscription") || strings.Contains(message, "canceled subscription") {
						log.Printf("🔄 STOMP: Subscription cancelled by server, attempting to resubscribe...")

						// Resubscribe if we know which subscription was cancelled
						if subscriptionID != "" && len(b.destinations) > 0 {
							// Parse subscription index from "sub-N" format
							var subIndex int
							if _, err := fmt.Sscanf(subscriptionID, "sub-%d", &subIndex); err == nil && subIndex < len(b.destinations) {
								dest := b.destinations[subIndex]
								log.Printf("🔄 STOMP: Resubscribing to %s (subscription ID: %s)", dest, subscriptionID)

								// Send new SUBSCRIBE frame
								subscribeFrame := fmt.Sprintf("SUBSCRIBE\nid:%s\ndestination:%s\nack:auto\n\n\x00", subscriptionID, dest)
								if _, err := conn.Write([]byte(subscribeFrame)); err != nil {
									log.Printf("❌ STOMP: Failed to resubscribe to %s: %v", dest, err)
								} else {
									log.Printf("✅ STOMP: Resubscribed to %s", dest)
								}
							}
						}
					}

					// Common RabbitMQ STOMP issues
					if strings.Contains(message, "Authentication") || strings.Contains(message, "ACCESS_REFUSED") {
						log.Printf("🔑 STOMP: Authentication issue - check RabbitMQ user permissions")
					} else if strings.Contains(message, "NOT_FOUND") {
						log.Printf("📂 STOMP: Queue/Exchange not found - queues will be created on first message")
					} else if strings.Contains(message, "version") {
						log.Printf("🔧 STOMP: Protocol version mismatch - using STOMP 1.0/1.1/1.2")
					}
				} else {
					// Truncate long messages for logging
					displayMsg := message
					if len(message) > 200 {
						displayMsg = message[:200] + "..."
					}
					log.Printf("📥 STOMP: Received frame (%d bytes): %s", n, displayMsg)
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

// ProcessUSPMessage implements the MessageProcessor interface
// This is called by the STOMP broker when a message is received from agents
func (s *STOMPMTPService) ProcessUSPMessage(data []byte) ([]byte, error) {
	log.Printf("📥 STOMP: Received USP message from agent (%d bytes)", len(data))

	// Extract endpoint ID from USP Record to properly route responses
	endpointID := s.extractEndpointID(data)
	if endpointID == "" {
		endpointID = "stomp-client-unknown" // Fallback
		log.Printf("⚠️ STOMP: Could not extract endpoint ID from message, using fallback")
	}

	// Publish USP message to Kafka inbound topic for USP service to process
	err := s.kafkaProducer.PublishUSPMessage(
		s.config.Kafka.Topics.USPMessagesInbound,
		endpointID,                                   // endpointID extracted from USP Record
		fmt.Sprintf("msg-%d", time.Now().UnixNano()), // messageID
		"Request", // messageType
		data,      // payload
		"stomp",   // mtpProtocol
	)

	if err != nil {
		log.Printf("❌ STOMP: Failed to publish to Kafka inbound topic: %v", err)
		return nil, fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	log.Printf("✅ STOMP: USP message published to Kafka inbound topic (EndpointID: %s)", endpointID)

	// In event-driven architecture, we don't wait for synchronous response
	// Responses will come through Kafka outbound topic
	return nil, nil
}

// extractEndpointID extracts the FromId (endpoint ID) from a USP Record
func (s *STOMPMTPService) extractEndpointID(data []byte) string {
	// Scan for "proto::usp.agent" pattern which is the agent's endpoint ID format
	// This avoids matching "proto::openusp.controller" (the ToId)
	dataStr := string(data)
	
	// Look specifically for agent endpoint patterns
	patterns := []string{"proto::usp.agent", "proto::cwmp.agent", "proto::device"}
	for _, pattern := range patterns {
		if idx := strings.Index(dataStr, pattern); idx >= 0 {
			// Extract endpoint ID (format: proto::usp.agent.XXX)
			remaining := dataStr[idx:]
			// Find the null terminator or whitespace (but not colon, as it's part of proto::)
			endIdx := strings.IndexAny(remaining, "\x00\n\r\t ")
			if endIdx > 0 {
				return remaining[:endIdx]
			}
			// If no terminator found within reasonable length, take first 25 chars
			if len(remaining) > 25 {
				return remaining[:25]
			}
			return remaining
		}
	}
	
	return ""
}

// setupKafkaConsumer configures Kafka consumer to receive outbound messages from USP service
func (s *STOMPMTPService) setupKafkaConsumer() error {
	// Subscribe to outbound topic to receive responses from USP service
	topics := []string{s.config.Kafka.Topics.USPMessagesOutbound}

	handler := func(msg *confluentkafka.Message) error {
		log.Printf("📨 STOMP: Received outbound message from Kafka (%d bytes)", len(msg.Value))
		
		// Unmarshal Kafka USPMessageEvent envelope
		var event kafka.USPMessageEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("❌ STOMP: Failed to unmarshal USPMessageEvent: %v", err)
			return err
		}

		log.Printf("📨 STOMP: USP Response - EndpointID: %s, MessageType: %s, Payload: %d bytes",
			event.EndpointID, event.MessageType, len(event.Payload))

		// Send message to agent via STOMP
		if s.stompBroker != nil && s.stompBroker.IsConnected() {
			// Send to the outbound STOMP destination (agents subscribe to this)
			outboundDest := s.config.MTP.STOMP.Destinations.Outbound
			if outboundDest == "" {
				outboundDest = "/queue/usp.agent" // Default fallback
			}
			
			// Send the raw USP protobuf payload (not the JSON envelope)
			err := s.stompBroker.Send(outboundDest, event.Payload)
			if err != nil {
				log.Printf("❌ STOMP: Failed to send message to agent %s via %s: %v", event.EndpointID, outboundDest, err)
				return err
			}
			log.Printf("✅ STOMP: Sent response to agent %s via %s (%d bytes)", event.EndpointID, outboundDest, len(event.Payload))
		} else {
			log.Printf("⚠️ STOMP: Broker not connected, cannot send message to agent")
		}
		
		return nil
	}

	// Subscribe to Kafka outbound topic
	if err := s.kafkaConsumer.Subscribe(topics, handler); err != nil {
		return fmt.Errorf("failed to subscribe to Kafka outbound topic: %w", err)
	}

	log.Printf("✅ STOMP: Subscribed to Kafka outbound topic: %s", s.config.Kafka.Topics.USPMessagesOutbound)
	return nil
}
