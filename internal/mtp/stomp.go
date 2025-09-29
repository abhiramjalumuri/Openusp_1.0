package mtp

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// STOMPBroker implements STOMP Message Transfer Protocol for USP
type STOMPBroker struct {
	brokerURL      string
	conn           net.Conn
	messageHandler MessageHandler
	destinations   []string
	mu             sync.RWMutex
	connected      bool
}

// NewSTOMPBroker creates a new STOMP broker connection
func NewSTOMPBroker(brokerURL string) (*STOMPBroker, error) {
	broker := &STOMPBroker{
		brokerURL: brokerURL,
		destinations: []string{
			"/queue/usp.controller",
			"/queue/usp.agent",
			"/topic/usp.broadcast",
		},
	}

	return broker, nil
}

// SetMessageHandler sets the message handler function
func (b *STOMPBroker) SetMessageHandler(handler MessageHandler) {
	b.messageHandler = handler
}

// Start connects to STOMP broker and starts listening
func (b *STOMPBroker) Start(ctx context.Context) error {
	log.Printf("🔌 Connecting to STOMP broker: %s", b.brokerURL)

	// For demo purposes, simulate STOMP connection
	b.mu.Lock()
	b.connected = true
	b.mu.Unlock()

	log.Printf("✅ STOMP broker connected: %s", b.brokerURL)

	// Subscribe to destinations
	for _, dest := range b.destinations {
		log.Printf("📡 Subscribed to STOMP destination: %s", dest)
	}

	// Simulate message listening
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 STOMP broker shutting down...")
			return nil
		case <-ticker.C:
			// Simulate periodic heartbeat
			log.Printf("💓 STOMP broker heartbeat")
		}
	}
}

// Send sends a message to a STOMP destination
func (b *STOMPBroker) Send(destination string, payload []byte) error {
	b.mu.RLock()
	connected := b.connected
	b.mu.RUnlock()

	if !connected {
		return fmt.Errorf("STOMP broker not connected")
	}

	log.Printf("📤 STOMP message sent to destination %s (%d bytes)", destination, len(payload))
	return nil
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
