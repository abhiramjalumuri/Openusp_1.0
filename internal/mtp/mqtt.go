// Package mtp provides Message Transfer Protocol implementations for USP
package mtp

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MessageHandler defines the function signature for handling messages
type MessageHandler func(topic string, payload []byte)

// MQTTBroker implements MQTT Message Transfer Protocol for USP
type MQTTBroker struct {
	brokerURL      string
	client         mqtt.Client
	messageHandler MessageHandler
	topics         []string
}

// NewMQTTBroker creates a new MQTT broker connection
func NewMQTTBroker(brokerURL string) (*MQTTBroker, error) {
	// Validate broker URL
	_, err := url.Parse(brokerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid MQTT broker URL: %w", err)
	}

	broker := &MQTTBroker{
		brokerURL: brokerURL,
		topics: []string{
			"/usp/controller/+", // Messages to controllers
			"/usp/agent/+",      // Messages to agents
			"/usp/broadcast",    // Broadcast messages
		},
	}

	return broker, nil
}

// SetMessageHandler sets the message handler function
func (b *MQTTBroker) SetMessageHandler(handler MessageHandler) {
	b.messageHandler = handler
}

// Start connects to MQTT broker and starts listening
func (b *MQTTBroker) Start(ctx context.Context) error {
	log.Printf("🔌 Connecting to MQTT broker: %s", b.brokerURL)

	// Create MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(b.brokerURL)
	opts.SetClientID("usp-mtp-service")
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(5 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)

	// Set connection handlers
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("✅ MQTT broker connected: %s", b.brokerURL)

		// Subscribe to USP topics
		for _, topic := range b.topics {
			if token := client.Subscribe(topic, 1, b.onMessage); token.Wait() && token.Error() != nil {
				log.Printf("❌ Failed to subscribe to MQTT topic %s: %v", topic, token.Error())
			} else {
				log.Printf("📡 Subscribed to MQTT topic: %s", topic)
			}
		}
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("⚠️ MQTT connection lost: %v", err)
	})

	// Create and connect client
	b.client = mqtt.NewClient(opts)
	if token := b.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Printf("🛑 MQTT broker shutting down...")

	return nil
}

// onMessage handles incoming MQTT messages
func (b *MQTTBroker) onMessage(client mqtt.Client, msg mqtt.Message) {
	if b.messageHandler != nil {
		go b.messageHandler(msg.Topic(), msg.Payload())
	}
}

// Publish sends a message to an MQTT topic
func (b *MQTTBroker) Publish(topic string, payload []byte) error {
	if b.client == nil || !b.client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	token := b.client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish MQTT message: %w", token.Error())
	}

	return nil
}

// Close disconnects from the MQTT broker
func (b *MQTTBroker) Close() {
	if b.client != nil && b.client.IsConnected() {
		b.client.Disconnect(250)
		log.Printf("✅ MQTT broker disconnected")
	}
}
