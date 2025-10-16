package mtp

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
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

// NewSTOMPBroker creates a new STOMP broker connection with default guest credentials
func NewSTOMPBroker(brokerURL string, destinations []string) (*STOMPBroker, error) {
	return NewSTOMPBrokerWithAuth(brokerURL, destinations, "guest", "guest")
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

	// Parse the broker URL to get host and port
	// Expected format: tcp://localhost:61613

	// Note: Authentication may fail with RabbitMQ guest user restrictions

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
	// SEND
	// destination:/queue/xxx
	// content-type:application/vnd.bbf.usp.msg
	// content-length:nnn
	// <blank line>
	// <binary payload>
	// NULL
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
					// STOMP MESSAGE format:
					// MESSAGE
					// destination:/queue/...
					// content-length:nnn
					// <blank line>
					// <binary payload>
					// NULL byte

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
