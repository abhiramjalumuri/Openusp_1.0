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
	cwmpServer        *http.Server // TR-069 protocol server (port 7547)
	healthServer      *http.Server // Health/status/metrics server (dynamic port)
	processor         *cwmp.MessageProcessor
	tr181Mgr          *tr181.DeviceManager
	metrics           *metrics.OpenUSPMetrics
	onboardingManager *cwmp.OnboardingManager
	mu                sync.RWMutex
	connections       map[string]*cwmp.Session
}

// Config holds configuration for CWMP service
type Config struct {
	CWMPPort              int // Standard TR-069 port (7547)
	HealthPort            int // Dynamic port for health/status/metrics
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
		config = DefaultConfig(0) // Use port 0 for testing (will be assigned dynamically)
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

	service := &CWMPService{
		deployConfig: deployConfig,
		registry:     registry,
		serviceInfo:  serviceInfo,
		config:       config,
		processor:    processor,
		metrics:      metricsInstance,
		tr181Mgr:     tr181Manager,
		connections:  make(map[string]*cwmp.Session),
	}

	// Now resolve the data service address using the service discovery method
	dataServiceAddr, err := service.getDataServiceAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve data service address: %w", err)
	}

	// Initialize onboarding manager with resolved address
	onboardingManager, err := cwmp.NewOnboardingManager(dataServiceAddr, tr181Manager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize onboarding manager: %w", err)
	}

	// Set onboarding manager in service and processor
	service.onboardingManager = onboardingManager
	processor.SetOnboardingManager(onboardingManager)

	// Create CWMP protocol server (TR-069 on standard port 7547)
	cwmpMux := http.NewServeMux()
	cwmpMux.HandleFunc("/", service.handleCWMPRequest)

	service.cwmpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.CWMPPort),
		Handler:      cwmpMux,
		ReadTimeout:  config.ConnectionTimeout,
		WriteTimeout: config.ConnectionTimeout,
		IdleTimeout:  config.SessionTimeout,
	}

	// Create health/admin server (dynamic port registered with Consul)
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", service.handleHealth)
	healthMux.HandleFunc("/status", service.handleStatus)
	healthMux.Handle("/metrics", metrics.HTTPHandler())

	service.healthServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.HealthPort),
		Handler:      healthMux,
		ReadTimeout:  config.ConnectionTimeout,
		WriteTimeout: config.ConnectionTimeout,
		IdleTimeout:  config.SessionTimeout,
	}

	return service, nil
}

// DefaultConfig returns default CWMP service configuration
func DefaultConfig(healthPort int) *Config {
	// CWMP protocol always uses standard TR-069 port
	cwmpPort := 7547

	// Data service address will be resolved later based on Consul availability
	dataServiceAddr := "" // Will be populated by getDataServiceAddress()

	return &Config{
		CWMPPort:              cwmpPort,
		HealthPort:            healthPort,
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
	log.Printf("🚀 Starting CWMP Service...")
	log.Printf("   📡 CWMP Protocol Port: %d (TR-069)", s.config.CWMPPort)
	log.Printf("   🏥 Health API Port: %d (Dynamic)", s.config.HealthPort)

	// Start the CWMP protocol server in a goroutine
	go func() {
		var err error
		if s.config.TLS.Enabled {
			log.Printf("🔒 TLS enabled, using cert: %s, key: %s", s.config.TLS.CertFile, s.config.TLS.KeyFile)
			err = s.cwmpServer.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
		} else {
			log.Printf("🔓 TLS disabled")
			err = s.cwmpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Printf("❌ CWMP protocol server error: %v", err)
		}
	}()

	// Start the health/admin server in a goroutine
	go func() {
		err := s.healthServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Health server error: %v", err)
		}
	}()

	log.Printf("✅ CWMP Service is running")
	log.Printf("   📍 CWMP Endpoint: http://localhost:%d", s.config.CWMPPort)
	log.Printf("   � Authentication: %s", map[bool]string{true: "✅ Enabled", false: "❌ Disabled"}[s.config.EnableAuthentication])
	log.Printf("   ⏱️  Connection Timeout: %v", s.config.ConnectionTimeout)
	log.Printf("   ⏱️  Session Timeout: %v", s.config.SessionTimeout)
	log.Printf("   🔧 Health Check: http://localhost:%d/health", s.config.HealthPort)
	log.Printf("   🔧 Status: http://localhost:%d/status", s.config.HealthPort)
	log.Printf("   📊 Metrics: http://localhost:%d/metrics", s.config.HealthPort)

	// Wait for context cancellation
	<-ctx.Done()
	return s.Stop()
}

// getDataServiceAddress resolves the data service address using Consul service discovery or environment variables
func (s *CWMPService) getDataServiceAddress() (string, error) {
	if s.registry != nil {
		// Try to discover data service from Consul
		service, err := s.registry.DiscoverService("openusp-data-service")
		if err == nil && service != nil {
			if grpcPort, exists := service.Meta["grpc_port"]; exists {
				return fmt.Sprintf("%s:%s", service.Address, grpcPort), nil
			}
		}
		log.Printf("⚠️  Data service not found in Consul, using fallback address")
	}

	// Fallback to environment variables or defaults
	dataServiceAddr := strings.TrimSpace(os.Getenv("OPENUSP_DATA_SERVICE_ADDR"))
	if dataServiceAddr == "" {
		dataServiceAddr = strings.TrimSpace(os.Getenv("DATA_SERVICE_ADDR"))
		if dataServiceAddr == "" {
			dataServiceAddr = "localhost:56400" // Default gRPC port
		}
	}
	return dataServiceAddr, nil
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

	// Shutdown both HTTP servers
	var shutdownErr error

	// Shutdown CWMP protocol server
	if err := s.cwmpServer.Shutdown(ctx); err != nil {
		log.Printf("❌ Error shutting down CWMP protocol server: %v", err)
		shutdownErr = err
	}

	// Shutdown health server
	if err := s.healthServer.Shutdown(ctx); err != nil {
		log.Printf("❌ Error shutting down health server: %v", err)
		if shutdownErr == nil {
			shutdownErr = err
		}
	}

	if shutdownErr != nil {
		return shutdownErr
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
	deployConfig := config.LoadDeploymentConfigWithPortEnv("openusp-cwmp-service", "cwmp-service", 7547, "OPENUSP_CWMP_SERVICE_PORT")

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

	// Load configuration with dual ports
	// CWMP protocol uses standard port 7547, health API uses dynamic port
	config := DefaultConfig(httpPort) // httpPort is the dynamic port for health API

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
