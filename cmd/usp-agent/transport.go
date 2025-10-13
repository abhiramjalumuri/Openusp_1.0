package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"openusp/pkg/config"

	"github.com/gorilla/websocket"
)

// Transport is an abstraction over an MTP transport (WebSocket, STOMP, etc.)
type Transport interface {
	Connect() error
	Send([]byte) error
	Read(timeout time.Duration) ([]byte, error)
	Close() error
	Name() string
}

// ----------------------------------------------------------------------------
// WebSocket Transport
// ----------------------------------------------------------------------------
type WebSocketTransport struct {
	url         string
	subprotocol string
	conn        *websocket.Conn
}

func NewWebSocketTransport(url, subprotocol string) *WebSocketTransport {
	return &WebSocketTransport{url: url, subprotocol: subprotocol}
}

func (w *WebSocketTransport) Name() string { return "websocket" }

func (w *WebSocketTransport) Connect() error {
	log.Printf("Connecting via WebSocket to: %s", w.url)
	var headers http.Header
	if w.subprotocol != "" {
		headers = http.Header{}
		headers.Set("Sec-WebSocket-Protocol", w.subprotocol)
	}
	conn, _, err := websocket.DefaultDialer.Dial(w.url, headers)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	w.conn = conn
	return nil
}

func (w *WebSocketTransport) Send(b []byte) error {
	if w.conn == nil {
		return errors.New("websocket not connected")
	}
	return w.conn.WriteMessage(websocket.BinaryMessage, b)
}

func (w *WebSocketTransport) Read(timeout time.Duration) ([]byte, error) {
	if w.conn == nil {
		return nil, errors.New("websocket not connected")
	}
	_ = w.conn.SetReadDeadline(time.Now().Add(timeout))
	_, msg, err := w.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (w *WebSocketTransport) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// ----------------------------------------------------------------------------
// STOMP Transport: Only /queue/usp.agent and /queue/usp.controller are used
// ----------------------------------------------------------------------------
type StompTransport struct {
	brokerURL     string
	username      string
	password      string
	destSend      string
	destSubscribe string
	hbSend        time.Duration
	hbRecv        time.Duration
	conn          net.Conn
	sessionID     string
	inbox         chan []byte
	errs          chan error
	closeOnce     sync.Once
	closed        chan struct{}
}

func NewStompTransport(cfg *config.TR369Config) *StompTransport {
	// Fallbacks
	// Use config values for STOMP destinations
	sendDest := cfg.STOMPDestinationController
	subDest := cfg.STOMPDestinationAgent
	log.Println("Stomp queue Tx (agent->controller): ", sendDest)
	log.Println("Stomp queue Rx (controller->agent): ", subDest)
	
	// Log STOMP credentials (mask password)
	username := cfg.STOMPUsername
	password := cfg.STOMPPassword
	if username == "" {
		username = "openusp"
		log.Println("⚠️  STOMP username not in config, using default: openusp")
	} else {
		log.Printf("🔑 STOMP username from config: %s", username)
	}
	if password == "" {
		password = "openusp123"
		log.Println("⚠️  STOMP password not in config, using default: ****")
	} else {
		log.Println("🔑 STOMP password from config: ****")
	}
	
	return &StompTransport{
		brokerURL:     cfg.STOMPBrokerURL,
		username:      username,
		password:      password,
		destSend:      sendDest,
		destSubscribe: subDest,
		hbSend:        cfg.STOMPHeartbeatSend,
		hbRecv:        cfg.STOMPHeartbeatReceive,
		inbox:         make(chan []byte, 8),
		errs:          make(chan error, 1),
		closed:        make(chan struct{}),
	}
}

func (s *StompTransport) Name() string { return "stomp" }

func (s *StompTransport) parseHostPort() (string, string, error) {
	url := s.brokerURL
	if strings.HasPrefix(url, "stomp://") {
		url = strings.TrimPrefix(url, "stomp://")
	} else if strings.HasPrefix(url, "tcp://") {
		url = strings.TrimPrefix(url, "tcp://")
	}
	host, port, ok := strings.Cut(url, ":")
	if !ok {
		port = "61613"
	}
	if port == "" {
		port = "61613"
	}
	return host, port, nil
}

func (s *StompTransport) Connect() error {
	host, port, _ := s.parseHostPort()
	log.Printf("🔌 STOMP: Connecting to %s:%s", host, port)
	conn, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return fmt.Errorf("stomp dial: %w", err)
	}
	s.conn = conn
	
	// Username and password should already be set from config in NewStompTransport
	// These are just safety fallbacks
	if s.username == "" {
		s.username = "openusp"
		log.Printf("⚠️  STOMP username empty, using fallback: openusp")
	}
	if s.password == "" {
		s.password = "openusp123"
		log.Printf("⚠️  STOMP password empty, using fallback")
	}
	
	log.Printf("🔐 STOMP: Authenticating as user: %s", s.username)
	frame := fmt.Sprintf("CONNECT\naccept-version:1.2\nhost:/\nlogin:%s\npasscode:%s", s.username, s.password)
	if s.hbSend > 0 || s.hbRecv > 0 {
		frame += fmt.Sprintf("\nheart-beat:%d,%d", s.hbSend.Milliseconds(), s.hbRecv.Milliseconds())
	}
	frame += "\n\n\x00"
	if _, err = s.conn.Write([]byte(frame)); err != nil {
		return fmt.Errorf("stomp connect write: %w", err)
	}
	// Read CONNECTED
	_ = s.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 2048)
	n, err := s.conn.Read(buf)
	if err != nil {
		return fmt.Errorf("stomp connect read: %w", err)
	}
	resp := string(buf[:n])
	if !strings.HasPrefix(resp, "CONNECTED") {
		return fmt.Errorf("stomp expected CONNECTED, got: %s", resp)
	}
	for _, line := range strings.Split(resp, "\n") {
		if strings.HasPrefix(line, "session:") {
			s.sessionID = strings.TrimPrefix(line, "session:")
			break
		}
	}
	// Subscribe
	sub := fmt.Sprintf("SUBSCRIBE\nid:sub-1\ndestination:%s\nack:auto\n\n\x00", s.destSubscribe)
	if _, err = s.conn.Write([]byte(sub)); err != nil {
		return fmt.Errorf("stomp subscribe: %w", err)
	}
	log.Printf("STOMP subscribed to %s", s.destSubscribe)
	// Start reader & heartbeat goroutines
	go s.readLoop()
	if s.hbSend > 0 {
		go s.heartbeatLoop()
	}
	return nil
}

func (s *StompTransport) heartbeatLoop() {
	ticker := time.NewTicker(s.hbSend)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
			if _, err := s.conn.Write([]byte("\n")); err != nil {
				s.errs <- fmt.Errorf("stomp heartbeat write: %w", err)
				return
			}
		case <-s.closed:
			return
		}
	}
}

func (s *StompTransport) readLoop() {
	reader := bufio.NewReader(s.conn)
	for {
		s.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		frame, err := readStompFrame(reader)
		if err != nil {
			select {
			case s.errs <- err:
			default:
			}
			// Use sync.Once to prevent panic from closing already-closed channel
			s.closeOnce.Do(func() {
				close(s.closed)
			})
			return
		}
		if frame.command == "MESSAGE" {
			s.inbox <- frame.body
		}
	}
}

type stompFrame struct {
	command string
	headers map[string]string
	body    []byte
}

func readStompFrame(r *bufio.Reader) (*stompFrame, error) {
	// Read command line
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" { // heartbeat newline
		return &stompFrame{command: "HEARTBEAT", headers: map[string]string{}, body: nil}, nil
	}
	f := &stompFrame{command: line, headers: map[string]string{}}
	// Headers
	for {
		h, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		h = strings.TrimRight(h, "\r\n")
		if h == "" {
			break
		}
		if k, v, ok := strings.Cut(h, ":"); ok {
			f.headers[k] = v
		}
	}
	// Body until null 0
	body, err := r.ReadBytes(0x00)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		body = body[:len(body)-1]
	}
	f.body = body
	return f, nil
}

func (s *StompTransport) Send(b []byte) error {
	if s.conn == nil {
		return errors.New("stomp not connected")
	}
	header := fmt.Sprintf("SEND\ndestination:%s\ncontent-type:application/octet-stream\ncontent-length:%d\n\n", s.destSend, len(b))
	frame := make([]byte, len(header)+len(b)+1)
	copy(frame, header)
	copy(frame[len(header):], b)
	frame[len(frame)-1] = 0
	s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := s.conn.Write(frame)
	return err
}

func (s *StompTransport) Read(timeout time.Duration) ([]byte, error) {
	select {
	case msg := <-s.inbox:
		return msg, nil
	case err := <-s.errs:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("stomp read timeout after %s", timeout)
	}
}

func (s *StompTransport) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closed)
		if s.conn != nil {
			disc := fmt.Sprintf("DISCONNECT\nsession:%s\n\n\x00", s.sessionID)
			s.conn.Write([]byte(disc))
			err = s.conn.Close()
		}
	})
	return err
}
