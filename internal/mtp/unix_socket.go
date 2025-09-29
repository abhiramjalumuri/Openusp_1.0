package mtp

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

// UnixSocketMessageHandler defines the function signature for Unix socket message handling
type UnixSocketMessageHandler func(clientID string, payload []byte)

// UnixSocketServer implements Unix Domain Socket Message Transfer Protocol for USP
type UnixSocketServer struct {
	socketPath     string
	listener       net.Listener
	clients        map[string]net.Conn
	messageHandler UnixSocketMessageHandler
	mu             sync.RWMutex
}

// NewUnixSocketServer creates a new Unix Domain Socket server
func NewUnixSocketServer(socketPath string) (*UnixSocketServer, error) {
	server := &UnixSocketServer{
		socketPath: socketPath,
		clients:    make(map[string]net.Conn),
	}

	return server, nil
}

// SetMessageHandler sets the message handler function
func (s *UnixSocketServer) SetMessageHandler(handler UnixSocketMessageHandler) {
	s.messageHandler = handler
}

// Start starts the Unix Domain Socket server
func (s *UnixSocketServer) Start(ctx context.Context) error {
	// Remove existing socket file
	if err := os.RemoveAll(s.socketPath); err != nil {
		log.Printf("⚠️ Failed to remove existing socket file: %v", err)
	}

	// Create Unix domain socket listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create Unix socket listener: %w", err)
	}
	s.listener = listener

	log.Printf("🔌 Unix Domain Socket server listening on %s", s.socketPath)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					log.Printf("❌ Unix socket accept error: %v", err)
					continue
				}
			}

			clientID := fmt.Sprintf("unix-client-%p", conn)
			s.mu.Lock()
			s.clients[clientID] = conn
			s.mu.Unlock()

			log.Printf("📱 Unix socket client connected: %s", clientID)

			go s.handleClient(clientID, conn)
		}
	}()

	log.Printf("✅ Unix Domain Socket server started on %s", s.socketPath)

	// Wait for context cancellation
	<-ctx.Done()
	log.Printf("🛑 Unix Domain Socket server shutting down...")

	return nil
}

// handleClient handles a Unix socket client connection
func (s *UnixSocketServer) handleClient(clientID string, conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		s.mu.Unlock()
		conn.Close()
		log.Printf("📱 Unix socket client disconnected: %s", clientID)
	}()

	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("❌ Unix socket read error: %v", err)
			}
			break
		}

		if n > 0 && s.messageHandler != nil {
			payload := make([]byte, n)
			copy(payload, buffer[:n])
			go s.messageHandler(clientID, payload)
		}
	}
}

// SendToClient sends a message to a specific Unix socket client
func (s *UnixSocketServer) SendToClient(clientID string, payload []byte) error {
	s.mu.RLock()
	conn, exists := s.clients[clientID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	_, err := conn.Write(payload)
	return err
}

// Broadcast sends a message to all connected Unix socket clients
func (s *UnixSocketServer) Broadcast(payload []byte) {
	s.mu.RLock()
	clients := make(map[string]net.Conn)
	for id, conn := range s.clients {
		clients[id] = conn
	}
	s.mu.RUnlock()

	for clientID, conn := range clients {
		if _, err := conn.Write(payload); err != nil {
			log.Printf("❌ Failed to send Unix socket message to client %s: %v", clientID, err)
		}
	}
}

// Close shuts down the Unix Domain Socket server
func (s *UnixSocketServer) Close() {
	if s.listener != nil {
		s.listener.Close()
	}

	// Close all client connections
	s.mu.Lock()
	for _, conn := range s.clients {
		conn.Close()
	}
	s.clients = make(map[string]net.Conn)
	s.mu.Unlock()

	// Remove socket file
	if err := os.RemoveAll(s.socketPath); err != nil {
		log.Printf("⚠️ Failed to remove socket file: %v", err)
	}

	log.Printf("✅ Unix Domain Socket server closed")
}
