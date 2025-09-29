package mtp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessageHandler defines the function signature for WebSocket message handling
type WebSocketMessageHandler func(clientID string, payload []byte)

// WebSocketServer implements WebSocket Message Transfer Protocol for USP
type WebSocketServer struct {
	port           int
	server         *http.Server
	upgrader       websocket.Upgrader
	clients        map[string]*websocket.Conn
	messageHandler WebSocketMessageHandler
	mu             sync.RWMutex
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(port int) (*WebSocketServer, error) {
	server := &WebSocketServer{
		port:    port,
		clients: make(map[string]*websocket.Conn),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}

	return server, nil
}

// SetMessageHandler sets the message handler function
func (s *WebSocketServer) SetMessageHandler(handler WebSocketMessageHandler) {
	s.messageHandler = handler
}

// Start starts the WebSocket server
func (s *WebSocketServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/usp", s.handleUSPWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	log.Printf("🔌 Starting WebSocket server on port %d", s.port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ WebSocket server error: %v", err)
		}
	}()

	log.Printf("✅ WebSocket server started on port %d", s.port)

	// Wait for context cancellation
	<-ctx.Done()
	log.Printf("🛑 WebSocket server shutting down...")

	return s.server.Shutdown(ctx)
}

// handleUSPWebSocket handles WebSocket connections for USP messages
func (s *WebSocketServer) handleUSPWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}

	clientID := r.Header.Get("X-Client-ID")
	if clientID == "" {
		clientID = fmt.Sprintf("client-%d", len(s.clients))
	}

	s.mu.Lock()
	s.clients[clientID] = conn
	s.mu.Unlock()

	log.Printf("📱 WebSocket client connected: %s", clientID)

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		s.mu.Unlock()
		conn.Close()
		log.Printf("📱 WebSocket client disconnected: %s", clientID)
	}()

	// Handle messages from client
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error: %v", err)
			}
			break
		}

		if messageType == websocket.BinaryMessage && s.messageHandler != nil {
			go s.messageHandler(clientID, payload)
		}
	}
}

// handleHealth provides a health check endpoint
func (s *WebSocketServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	s.mu.RLock()
	clientCount := len(s.clients)
	s.mu.RUnlock()

	fmt.Fprintf(w, `{"status":"healthy","connected_clients":%d}`, clientCount)
}

// SendToClient sends a message to a specific WebSocket client
func (s *WebSocketServer) SendToClient(clientID string, payload []byte) error {
	s.mu.RLock()
	conn, exists := s.clients[clientID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	return conn.WriteMessage(websocket.BinaryMessage, payload)
}

// Broadcast sends a message to all connected WebSocket clients
func (s *WebSocketServer) Broadcast(payload []byte) {
	s.mu.RLock()
	clients := make(map[string]*websocket.Conn)
	for id, conn := range s.clients {
		clients[id] = conn
	}
	s.mu.RUnlock()

	for clientID, conn := range clients {
		if err := conn.WriteMessage(websocket.BinaryMessage, payload); err != nil {
			log.Printf("❌ Failed to send WebSocket message to client %s: %v", clientID, err)
		}
	}
}

// Close shuts down the WebSocket server
func (s *WebSocketServer) Close() {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(ctx)
	}
	log.Printf("✅ WebSocket server closed")
}
