package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"openusp/internal/cwmp"
	"openusp/internal/tr181"
	"openusp/pkg/config"
	"openusp/pkg/consul"
	"openusp/pkg/metrics"
	"openusp/pkg/version"
)

// CWMPService provides TR-069 protocol support for backward compatibility
type CWMPService struct {
	deployConfig      *config.DeploymentConfig
	registry          *consul.ServiceRegistry
	serviceInfo       *consul.ServiceInfo
	config            *Config
	server            *http.Server
	processor         *cwmp.MessageProcessor
	tr181Mgr          *tr181.DeviceManager
	metrics           *metrics.OpenUSPMetrics
	onboardingManager *cwmp.OnboardingManager
	mu                sync.RWMutex
	connections       map[string]*cwmp.Session
}

// Config holds configuration for CWMP service
type Config struct {
	Port                  int
	ACSUsername           string
	ACSPassword           string
	ConnectionTimeout     time.Duration
	SessionTimeout        time.Duration
	MaxConcurrentSessions int
	EnableAuthentication  bool
	DataServiceAddress    string
	TLS                   TLSConfig
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// NewCWMPService creates a new CWMP service instance
func NewCWMPService(config *Config, deployConfig *config.DeploymentConfig, registry *consul.ServiceRegistry, serviceInfo *consul.ServiceInfo) (*CWMPService, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Initialize TR-181 manager for data model support
	tr181Manager, err := tr181.NewDeviceManager("pkg/datamodel/tr-181-2-19-1-usp-full.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TR-181 manager: %w", err)
	}

	// Initialize CWMP message processor
	processor := cwmp.NewMessageProcessor(tr181Manager)

	// Initialize metrics
	metricsInstance := metrics.NewOpenUSPMetrics("cwmp-service")

	// Initialize onboarding manager
	onboardingManager, err := cwmp.NewOnboardingManager(config.DataServiceAddress, tr181Manager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize onboarding manager: %w", err)
	}

	// Set onboarding manager in processor
	processor.SetOnboardingManager(onboardingManager)

	service := &CWMPService{
		deployConfig:      deployConfig,
		registry:          registry,
		serviceInfo:       serviceInfo,
		config:            config,
		processor:         processor,
		metrics:           metricsInstance,
		tr181Mgr:          tr181Manager,
		onboardingManager: onboardingManager,
		connections:       make(map[string]*cwmp.Session),
	}

	// Create HTTP server with CWMP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", service.handleCWMPRequest)
	mux.HandleFunc("/health", service.handleHealth)
	mux.HandleFunc("/status", service.handleStatus)
	mux.Handle("/metrics", metrics.HTTPHandler())

	service.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  config.ConnectionTimeout,
		WriteTimeout: config.ConnectionTimeout,
		IdleTimeout:  config.SessionTimeout,
	}

	return service, nil
}

// DefaultConfig returns default CWMP service configuration
func DefaultConfig() *Config {
	// Get port from environment or use default
	port := 7547 // Standard TR-069 CWMP port
	if envPort := strings.TrimSpace(os.Getenv("OPENUSP_CWMP_SERVICE_PORT")); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	// Get data service address from environment or use default
	dataServiceAddr := "localhost:56400" // Default gRPC data service address
	if envAddr := strings.TrimSpace(os.Getenv("OPENUSP_DATA_SERVICE_ADDR")); envAddr != "" {
		dataServiceAddr = envAddr
	}

	return &Config{
		Port:                  port,
		ACSUsername:           "acs",
		ACSPassword:           "acs123",
		ConnectionTimeout:     30 * time.Second,
		SessionTimeout:        300 * time.Second,
		MaxConcurrentSessions: 100,
		EnableAuthentication:  true,
		DataServiceAddress:    dataServiceAddr,
		TLS: TLSConfig{
			Enabled:  false,
			CertFile: "",
			KeyFile:  "",
		},
	}
}

// Start starts the CWMP service
func (s *CWMPService) Start(ctx context.Context) error {
	log.Printf("🚀 Starting CWMP Service on port %d...", s.config.Port)

	// Start HTTP server
	go func() {
		var err error
		if s.config.TLS.Enabled {
			log.Printf("🔒 Starting HTTPS server with TLS...")
			err = s.server.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
		} else {
			log.Printf("🌐 Starting HTTP server...")
			err = s.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Printf("❌ CWMP server error: %v", err)
		}
	}()

	log.Printf("✅ CWMP Service started successfully")
	log.Printf("   📍 Endpoint: http://localhost:%d", s.config.Port)
	log.Printf("   🔧 TR-069 Protocol: Enabled")
	log.Printf("   🔧 Authentication: %v", s.config.EnableAuthentication)
	log.Printf("   🔧 Max Sessions: %d", s.config.MaxConcurrentSessions)
	log.Printf("   🔧 Health Check: http://localhost:%d/health", s.config.Port)
	log.Printf("   🔧 Status: http://localhost:%d/status", s.config.Port)

	// Wait for context cancellation
	<-ctx.Done()
	return s.Stop()
}

// Stop stops the CWMP service
func (s *CWMPService) Stop() error {
	log.Printf("🛑 Stopping CWMP Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close all active sessions
	s.mu.Lock()
	for sessionID, session := range s.connections {
		log.Printf("🔌 Closing session: %s", sessionID)
		session.Close()
	}
	s.connections = make(map[string]*cwmp.Session)
	s.mu.Unlock()

	// Close onboarding manager
	if s.onboardingManager != nil {
		if err := s.onboardingManager.Close(); err != nil {
			log.Printf("❌ Error closing onboarding manager: %v", err)
		}
	}

	// Shutdown HTTP server
	if err := s.server.Shutdown(ctx); err != nil {
		log.Printf("❌ Error shutting down CWMP server: %v", err)
		return err
	}

	log.Printf("✅ CWMP Service stopped")
	return nil
}

// handleCWMPRequest handles incoming CWMP requests
func (s *CWMPService) handleCWMPRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("📥 CWMP Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	// Only accept POST requests for CWMP
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication if enabled
	if s.config.EnableAuthentication {
		username, password, ok := r.BasicAuth()
		if !ok || username != s.config.ACSUsername || password != s.config.ACSPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="CWMP ACS"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Get or create session
	sessionID := s.getSessionID(r)
	session := s.getOrCreateSession(sessionID, r.RemoteAddr)

	// Process CWMP message
	if err := s.processor.ProcessRequest(session, w, r); err != nil {
		log.Printf("❌ CWMP processing error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ CWMP Request processed successfully")
}

// handleHealth handles health check requests
func (s *CWMPService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	healthStatus := map[string]interface{}{
		"service":      "cwmp-service",
		"status":       "healthy",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"version":      "1.0.0",
		"protocol":     "TR-069",
		"sessions":     len(s.connections),
		"max_sessions": s.config.MaxConcurrentSessions,
	}

	if err := json.NewEncoder(w).Encode(healthStatus); err != nil {
		log.Printf("❌ Health check encoding error: %v", err)
	}
}

// handleStatus handles detailed status requests
func (s *CWMPService) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	status := map[string]interface{}{
		"service":            "cwmp-service",
		"status":             "running",
		"timestamp":          time.Now().UTC().Format(time.RFC3339),
		"active_sessions":    len(s.connections),
		"max_sessions":       s.config.MaxConcurrentSessions,
		"authentication":     s.config.EnableAuthentication,
		"tls_enabled":        s.config.TLS.Enabled,
		"connection_timeout": s.config.ConnectionTimeout.String(),
		"session_timeout":    s.config.SessionTimeout.String(),
		"tr181_status":       "enabled",
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("❌ Status encoding error: %v", err)
	}
}

// getSessionID extracts session ID from request
func (s *CWMPService) getSessionID(r *http.Request) string {
	// Try to get session ID from headers first
	if sessionID := r.Header.Get("X-CWMP-Session-ID"); sessionID != "" {
		return sessionID
	}

	// Fallback to remote address as session identifier
	return r.RemoteAddr
}

// getOrCreateSession gets existing session or creates new one
func (s *CWMPService) getOrCreateSession(sessionID, remoteAddr string) *cwmp.Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.connections[sessionID]; exists {
		return session
	}

	// Check session limit
	if len(s.connections) >= s.config.MaxConcurrentSessions {
		log.Printf("⚠️ Maximum sessions reached, cleaning up oldest sessions")
		s.cleanupOldestSessions(10) // Clean up 10% of sessions
	}

	// Create new session
	session := cwmp.NewSession(sessionID, remoteAddr, s.config.SessionTimeout)
	s.connections[sessionID] = session

	log.Printf("🔌 New CWMP session created: %s", sessionID)
	return session
}

// cleanupOldestSessions removes oldest sessions to make room for new ones
func (s *CWMPService) cleanupOldestSessions(percentage int) {
	if percentage <= 0 || percentage > 100 {
		return
	}

	toRemove := len(s.connections) * percentage / 100
	if toRemove == 0 {
		toRemove = 1
	}

	// Simple cleanup - remove first N sessions (in production, use proper LRU)
	count := 0
	for sessionID, session := range s.connections {
		if count >= toRemove {
			break
		}
		session.Close()
		delete(s.connections, sessionID)
		count++
	}

	log.Printf("🧹 Cleaned up %d old sessions", count)
}

func main() {
	log.Printf("🚀 Starting OpenUSP CWMP Service...")

	// Command line flags
	var enableConsul = flag.Bool("consul", false, "Enable Consul service discovery")
	var port = flag.Int("port", 7547, "HTTP port")
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.GetFullVersion("OpenUSP CWMP Service"))
		return
	}

	if *showHelp {
		fmt.Println("OpenUSP CWMP Service - TR-069 Protocol Support")
		fmt.Println("==============================================")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  cwmp-service [flags]")
		fmt.Println("")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Environment Variables:")
		fmt.Println("  CONSUL_ENABLED     - Enable Consul service discovery (default: true)")
		fmt.Println("  SERVICE_PORT       - HTTP port (default: 7547)")
		return
	}

	// Override environment from flags
	if *enableConsul {
		os.Setenv("CONSUL_ENABLED", "true")
	}
	if *port != 7547 {
		os.Setenv("SERVICE_PORT", fmt.Sprintf("%d", *port))
	}

	// Load configuration
	deployConfig := config.LoadDeploymentConfig("openusp-cwmp-service", "cwmp-service", 7547)

	// Initialize Consul if enabled
	var registry *consul.ServiceRegistry
	var serviceInfo *consul.ServiceInfo
	if deployConfig.IsConsulEnabled() {
		consulAddr, interval, timeout := deployConfig.GetConsulConfig()
		consulConfig := &consul.Config{
			Address:       consulAddr,
			Datacenter:    "openusp-dev",
			CheckInterval: interval,
			CheckTimeout:  timeout,
		}

		var err error
		registry, err = consul.NewServiceRegistry(consulConfig)
		if err != nil {
			log.Fatalf("Failed to connect to Consul: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		serviceInfo, err = registry.RegisterService(ctx, deployConfig.ServiceName, deployConfig.ServiceType)
		if err != nil {
			log.Fatalf("Failed to register with Consul: %v", err)
		}

		log.Printf("🏛️ Connected to Consul at %s", consulAddr)
		log.Printf("🎯 Service registered with Consul: %s (%s) at localhost:%d",
			serviceInfo.Name, serviceInfo.Meta["service_type"], serviceInfo.Port)
	}

	// Determine the HTTP port to use
	var httpPort int
	if serviceInfo != nil {
		httpPort = serviceInfo.Port
	} else {
		httpPort = deployConfig.ServicePort
	}
	fmt.Println()

	// Load configuration
	config := DefaultConfig()
	config.Port = httpPort // Use the determined port

	// Create CWMP service
	cwmpService, err := NewCWMPService(config, deployConfig, registry, serviceInfo)
	if err != nil {
		log.Fatalf("Failed to create CWMP service: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("� Received shutdown signal")

		// Deregister from Consul
		if registry != nil && serviceInfo != nil {
			if err := registry.DeregisterService(serviceInfo.ID); err != nil {
				log.Printf("⚠️  Failed to deregister from Consul: %v", err)
			} else {
				log.Printf("✅ Deregistered from Consul successfully")
			}
		}

		cancel()
	}()

	// Start CWMP service and show status
	log.Printf("🚀 CWMP Service started successfully")
	if deployConfig.IsConsulEnabled() && serviceInfo != nil {
		log.Printf("   └── HTTP Port: %d", serviceInfo.Port)
		log.Printf("   └── Consul Service Discovery: ✅ Enabled")
		log.Printf("   └── Health Check: http://localhost:%d/health", serviceInfo.Port)
		log.Printf("   └── Status: http://localhost:%d/status", serviceInfo.Port)
		log.Printf("   └── Consul UI: http://localhost:8500/ui/")
	} else {
		log.Printf("   └── HTTP Port: %d", httpPort)
		log.Printf("   └── Consul Service Discovery: ❌ Disabled")
		log.Printf("   └── Health Check: http://localhost:%d/health", httpPort)
		log.Printf("   └── Status: http://localhost:%d/status", httpPort)
	}

	if err := cwmpService.Start(ctx); err != nil {
		log.Fatalf("CWMP service error: %v", err)
	}

	log.Printf("✅ CWMP Service stopped successfully")
}
