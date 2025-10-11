package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcImpl "openusp/internal/grpc"
	"openusp/internal/tr181"
	"openusp/pkg/config"
	"openusp/pkg/metrics"
	"openusp/pkg/proto/uspservice"
	v1_3 "openusp/pkg/proto/v1_3"
	v1_4 "openusp/pkg/proto/v1_4"
	"openusp/pkg/service/client"
	"openusp/pkg/version"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/proto"
)

type USPCoreService struct {
	deviceManager *tr181.DeviceManager
}

func NewUSPCoreService() (*USPCoreService, error) {
	dm, err := tr181.LoadDefaultDataModel()
	if err != nil {
		return nil, fmt.Errorf("failed to load TR-181 data model: %w", err)
	}

	return &USPCoreService{
		deviceManager: dm,
	}, nil
}

// DetectUSPVersion detects USP protocol version from raw message data
func (s *USPCoreService) DetectUSPVersion(data []byte) (string, error) {
	// Try to unmarshal as USP 1.4 first
	var record14 v1_4.Record
	if err := proto.Unmarshal(data, &record14); err == nil {
		if record14.Version == "1.4" {
			return "1.4", nil
		}
	}

	// Try to unmarshal as USP 1.3
	var record13 v1_3.Record
	if err := proto.Unmarshal(data, &record13); err == nil {
		if record13.Version == "1.3" {
			return "1.3", nil
		}
	}

	return "", fmt.Errorf("unable to detect USP version")
}

// ProcessUSPMessage processes incoming USP messages (both v1.3 and v1.4)
func (s *USPCoreService) ProcessUSPMessage(data []byte) ([]byte, error) {
	version, err := s.DetectUSPVersion(data)
	if err != nil {
		return nil, fmt.Errorf("failed to detect USP version: %w", err)
	}

	log.Printf("Processing USP %s message", version)

	switch version {
	case "1.4":
		return s.processUSP14Message(data)
	case "1.3":
		return s.processUSP13Message(data)
	default:
		return nil, fmt.Errorf("unsupported USP version: %s", version)
	}
}

func (s *USPCoreService) processUSP14Message(data []byte) ([]byte, error) {
	var record v1_4.Record
	if err := proto.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.4 record: %w", err)
	}

	log.Printf("USP 1.4 message from %s to %s", record.FromId, record.ToId)

	// Extract payload
	var payload []byte
	switch recordType := record.RecordType.(type) {
	case *v1_4.Record_NoSessionContext:
		payload = recordType.NoSessionContext.Payload
	default:
		return nil, fmt.Errorf("unsupported USP 1.4 record type")
	}

	// Parse the USP message
	var msg v1_4.Msg
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.4 message: %w", err)
	}

	// Create a simple response (simplified for demo)
	responseMsg := s.createUSP14Response(&msg)

	// Marshal response
	responsePayload, err := proto.Marshal(responseMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.4 response: %w", err)
	}

	// Create response record
	responseRecord := &v1_4.Record{
		Version: "1.4",
		ToId:    record.FromId,
		FromId:  record.ToId,
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: responsePayload,
			},
		},
	}

	return proto.Marshal(responseRecord)
}

func (s *USPCoreService) processUSP13Message(data []byte) ([]byte, error) {
	var record v1_3.Record
	if err := proto.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.3 record: %w", err)
	}

	log.Printf("USP 1.3 message from %s to %s", record.FromId, record.ToId)

	// Extract payload
	var payload []byte
	switch recordType := record.RecordType.(type) {
	case *v1_3.Record_NoSessionContext:
		payload = recordType.NoSessionContext.Payload
	default:
		return nil, fmt.Errorf("unsupported USP 1.3 record type")
	}

	// Parse the USP message
	var msg v1_3.Msg
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal USP 1.3 message: %w", err)
	}

	// Create a simple response (simplified for demo)
	responseMsg := s.createUSP13Response(&msg)

	// Marshal response
	responsePayload, err := proto.Marshal(responseMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal USP 1.3 response: %w", err)
	}

	// Create response record
	responseRecord := &v1_3.Record{
		Version: "1.3",
		ToId:    record.FromId,
		FromId:  record.ToId,
		RecordType: &v1_3.Record_NoSessionContext{
			NoSessionContext: &v1_3.NoSessionContextRecord{
				Payload: responsePayload,
			},
		},
	}

	return proto.Marshal(responseRecord)
}

func (s *USPCoreService) createUSP14Response(msg *v1_4.Msg) *v1_4.Msg {
	switch msg.Header.MsgType {
	case v1_4.Header_GET:
		return s.handleUSP14Get(msg)
	case v1_4.Header_GET_SUPPORTED_DM:
		return s.handleUSP14GetSupportedDM(msg)
	default:
		// Return error response for unsupported operations
		return &v1_4.Msg{
			Header: &v1_4.Header{
				MsgId:   fmt.Sprintf("error-resp-%s", msg.Header.MsgId),
				MsgType: v1_4.Header_ERROR,
			},
			Body: &v1_4.Body{
				MsgBody: &v1_4.Body_Error{
					Error: &v1_4.Error{
						ErrCode: 7000,
						ErrMsg:  "Operation not implemented in demo",
					},
				},
			},
		}
	}
}

func (s *USPCoreService) createUSP13Response(msg *v1_3.Msg) *v1_3.Msg {
	switch msg.Header.MsgType {
	case v1_3.Header_GET:
		return s.handleUSP13Get(msg)
	case v1_3.Header_GET_SUPPORTED_DM:
		return s.handleUSP13GetSupportedDM(msg)
	default:
		// Return error response for unsupported operations
		return &v1_3.Msg{
			Header: &v1_3.Header{
				MsgId:   fmt.Sprintf("error-resp-%s", msg.Header.MsgId),
				MsgType: v1_3.Header_ERROR,
			},
			Body: &v1_3.Body{
				MsgBody: &v1_3.Body_Error{
					Error: &v1_3.Error{
						ErrCode: 7000,
						ErrMsg:  "Operation not implemented in demo",
					},
				},
			},
		}
	}
}

func (s *USPCoreService) handleUSP14Get(msg *v1_4.Msg) *v1_4.Msg {
	req := msg.Body.GetRequest().GetGet()

	var requestedPathResults []*v1_4.GetResp_RequestedPathResult

	for _, path := range req.ParamPaths {
		if s.deviceManager.ValidateUSPPath(path) {
			// Create resolved path result
			resolvedResult := &v1_4.GetResp_ResolvedPathResult{
				ResolvedPath: path,
				ResultParams: map[string]string{
					path: s.getParameterValue(path),
				},
			}

			requestedPathResults = append(requestedPathResults, &v1_4.GetResp_RequestedPathResult{
				RequestedPath:       path,
				ErrCode:             0,
				ErrMsg:              "",
				ResolvedPathResults: []*v1_4.GetResp_ResolvedPathResult{resolvedResult},
			})
		} else {
			requestedPathResults = append(requestedPathResults, &v1_4.GetResp_RequestedPathResult{
				RequestedPath: path,
				ErrCode:       7004,
				ErrMsg:        "Parameter path not supported",
			})
		}
	}

	return &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   fmt.Sprintf("get-resp-%s", msg.Header.MsgId),
			MsgType: v1_4.Header_GET_RESP,
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Response{
				Response: &v1_4.Response{
					RespType: &v1_4.Response_GetResp{
						GetResp: &v1_4.GetResp{
							ReqPathResults: requestedPathResults,
						},
					},
				},
			},
		},
	}
}

func (s *USPCoreService) handleUSP13Get(msg *v1_3.Msg) *v1_3.Msg {
	req := msg.Body.GetRequest().GetGet()

	var requestedPathResults []*v1_3.GetResp_RequestedPathResult

	for _, path := range req.ParamPaths {
		if s.deviceManager.ValidateUSPPath(path) {
			// Create resolved path result
			resolvedResult := &v1_3.GetResp_ResolvedPathResult{
				ResolvedPath: path,
				ResultParams: map[string]string{
					path: s.getParameterValue(path),
				},
			}

			requestedPathResults = append(requestedPathResults, &v1_3.GetResp_RequestedPathResult{
				RequestedPath:       path,
				ErrCode:             0,
				ErrMsg:              "",
				ResolvedPathResults: []*v1_3.GetResp_ResolvedPathResult{resolvedResult},
			})
		} else {
			requestedPathResults = append(requestedPathResults, &v1_3.GetResp_RequestedPathResult{
				RequestedPath: path,
				ErrCode:       7004,
				ErrMsg:        "Parameter path not supported",
			})
		}
	}

	return &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   fmt.Sprintf("get-resp-%s", msg.Header.MsgId),
			MsgType: v1_3.Header_GET_RESP,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Response{
				Response: &v1_3.Response{
					RespType: &v1_3.Response_GetResp{
						GetResp: &v1_3.GetResp{
							ReqPathResults: requestedPathResults,
						},
					},
				},
			},
		},
	}
}

func (s *USPCoreService) handleUSP14GetSupportedDM(msg *v1_4.Msg) *v1_4.Msg {
	req := msg.Body.GetRequest().GetGetSupportedDm()

	var results []*v1_4.GetSupportedDMResp_RequestedObjectResult

	for _, objPath := range req.ObjPaths {
		supportedObj := &v1_4.GetSupportedDMResp_SupportedObjectResult{
			SupportedObjPath: objPath,
			Access:           v1_4.GetSupportedDMResp_OBJ_READ_ONLY,
			IsMultiInstance:  false,
		}

		results = append(results, &v1_4.GetSupportedDMResp_RequestedObjectResult{
			ReqObjPath:       objPath,
			ErrCode:          0,
			ErrMsg:           "",
			DataModelInstUri: "urn:broadband-forum-org:tr-181-2-19-1",
			SupportedObjs:    []*v1_4.GetSupportedDMResp_SupportedObjectResult{supportedObj},
		})
	}

	return &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   fmt.Sprintf("get-supported-dm-resp-%s", msg.Header.MsgId),
			MsgType: v1_4.Header_GET_SUPPORTED_DM_RESP,
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Response{
				Response: &v1_4.Response{
					RespType: &v1_4.Response_GetSupportedDmResp{
						GetSupportedDmResp: &v1_4.GetSupportedDMResp{
							ReqObjResults: results,
						},
					},
				},
			},
		},
	}
}

func (s *USPCoreService) handleUSP13GetSupportedDM(msg *v1_3.Msg) *v1_3.Msg {
	req := msg.Body.GetRequest().GetGetSupportedDm()

	var results []*v1_3.GetSupportedDMResp_RequestedObjectResult

	for _, objPath := range req.ObjPaths {
		supportedObj := &v1_3.GetSupportedDMResp_SupportedObjectResult{
			SupportedObjPath: objPath,
			Access:           v1_3.GetSupportedDMResp_OBJ_READ_ONLY,
			IsMultiInstance:  false,
		}

		results = append(results, &v1_3.GetSupportedDMResp_RequestedObjectResult{
			ReqObjPath:       objPath,
			ErrCode:          0,
			ErrMsg:           "",
			DataModelInstUri: "urn:broadband-forum-org:tr-181-2-19-1",
			SupportedObjs:    []*v1_3.GetSupportedDMResp_SupportedObjectResult{supportedObj},
		})
	}

	return &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   fmt.Sprintf("get-supported-dm-resp-%s", msg.Header.MsgId),
			MsgType: v1_3.Header_GET_SUPPORTED_DM_RESP,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Response{
				Response: &v1_3.Response{
					RespType: &v1_3.Response_GetSupportedDmResp{
						GetSupportedDmResp: &v1_3.GetSupportedDMResp{
							ReqObjResults: results,
						},
					},
				},
			},
		},
	}
}

func (s *USPCoreService) getParameterValue(path string) string {
	// Get value from TR-181 device manager
	deviceInfo := s.deviceManager.GetDeviceInfo()
	if value, exists := deviceInfo[path]; exists {
		return fmt.Sprintf("%v", value)
	}

	// Default values for common parameters
	switch path {
	case "Device.DeviceInfo.Manufacturer":
		return "OpenUSP Demo"
	case "Device.DeviceInfo.ManufacturerOUI":
		return "00D09E"
	case "Device.DeviceInfo.ModelName":
		return "TR-369 USP Agent"
	case "Device.DeviceInfo.SerialNumber":
		return "USP-DEMO-001"
	case "Device.DeviceInfo.SoftwareVersion":
		return "1.0.0"
	case "Device.DeviceInfo.HardwareVersion":
		return "1.0"
	default:
		return "unknown"
	}
}

// Demo functions to show dual protocol support
func (s *USPCoreService) DemoUSPOperations() {
	fmt.Println("\n--- USP Multi-Version Protocol Demonstration ---")

	// Demo USP 1.4 operations
	s.demoUSP14Operations()

	// Demo USP 1.3 operations
	s.demoUSP13Operations()

	// Demo version detection
	s.demoVersionDetection()

	// Demo device info
	s.demoDeviceInfo()
}

func (s *USPCoreService) demoUSP14Operations() {
	fmt.Println("\n1. USP 1.4 Message Creation:")

	// Test paths for validation
	testPaths := []string{
		"Device.DeviceInfo.Manufacturer",
		"Device.DeviceInfo.SerialNumber",
		"Device.InvalidPath.Test",
	}

	validPaths := []string{}
	for _, path := range testPaths {
		if s.deviceManager.ValidateUSPPath(path) {
			validPaths = append(validPaths, path)
			fmt.Printf("   ✅ %s - Valid\n", path)
		} else {
			fmt.Printf("   ❌ %s - Invalid\n", path)
		}
	}

	// Create USP 1.4 Get message
	msg := &v1_4.Msg{
		Header: &v1_4.Header{
			MsgId:   "demo-v14-get-001",
			MsgType: v1_4.Header_GET,
		},
		Body: &v1_4.Body{
			MsgBody: &v1_4.Body_Request{
				Request: &v1_4.Request{
					ReqType: &v1_4.Request_Get{
						Get: &v1_4.Get{
							ParamPaths: validPaths,
						},
					},
				},
			},
		},
	}

	msgData, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal USP 1.4 message: %v", err)
		return
	}

	fmt.Printf("   📦 Created USP 1.4 Get message (%d bytes)\n", len(msgData))
	fmt.Printf("   📋 Message ID: %s\n", msg.Header.MsgId)
}

func (s *USPCoreService) demoUSP13Operations() {
	fmt.Println("\n2. USP 1.3 Message Creation:")

	validPaths := []string{
		"Device.DeviceInfo.Manufacturer",
		"Device.DeviceInfo.SerialNumber",
	}

	// Create USP 1.3 Get message
	msg := &v1_3.Msg{
		Header: &v1_3.Header{
			MsgId:   "demo-v13-get-001",
			MsgType: v1_3.Header_GET,
		},
		Body: &v1_3.Body{
			MsgBody: &v1_3.Body_Request{
				Request: &v1_3.Request{
					ReqType: &v1_3.Request_Get{
						Get: &v1_3.Get{
							ParamPaths: validPaths,
						},
					},
				},
			},
		},
	}

	msgData, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal USP 1.3 message: %v", err)
		return
	}

	fmt.Printf("   📦 Created USP 1.3 Get message (%d bytes)\n", len(msgData))
	fmt.Printf("   📋 Message ID: %s\n", msg.Header.MsgId)
}

func (s *USPCoreService) demoVersionDetection() {
	fmt.Println("\n3. USP Version Detection:")

	// Create sample USP 1.4 record
	record14 := &v1_4.Record{
		Version: "1.4",
		ToId:    "agent::demo",
		FromId:  "controller::demo",
		RecordType: &v1_4.Record_NoSessionContext{
			NoSessionContext: &v1_4.NoSessionContextRecord{
				Payload: []byte("demo-payload"),
			},
		},
	}

	data14, _ := proto.Marshal(record14)
	version14, _ := s.DetectUSPVersion(data14)
	fmt.Printf("   📋 Detected version for USP 1.4 record: %s\n", version14)

	// Create sample USP 1.3 record
	record13 := &v1_3.Record{
		Version: "1.3",
		ToId:    "agent::demo",
		FromId:  "controller::demo",
		RecordType: &v1_3.Record_NoSessionContext{
			NoSessionContext: &v1_3.NoSessionContextRecord{
				Payload: []byte("demo-payload"),
			},
		},
	}

	data13, _ := proto.Marshal(record13)
	version13, _ := s.DetectUSPVersion(data13)
	fmt.Printf("   📋 Detected version for USP 1.3 record: %s\n", version13)
}

func (s *USPCoreService) demoDeviceInfo() {
	fmt.Println("\n4. Device Information (TR-181 Based):")

	deviceInfo := s.deviceManager.GetDeviceInfo()
	for path, value := range deviceInfo {
		fmt.Printf("   📍 %s = %v\n", path, value)
	}
}

func main() {
	log.Printf("🚀 Starting OpenUSP Core Service...")

	// Command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.GetFullVersion("OpenUSP Core Service"))
		return
	}

	if *showHelp {
		fmt.Println("OpenUSP Core Service - TR-369 USP Protocol Engine")
		fmt.Println("=================================================")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  usp-service [flags]")
		fmt.Println("")
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("Environment Variables:")
		fmt.Println("  SERVICE_PORT       - gRPC port (default: 56250)")
		return
	}

	// Static port configuration - no environment overrides needed

	// Load configuration
	deployConfig := config.LoadDeploymentConfigWithPortEnv("openusp-usp-service", "usp-service", 6400, "OPENUSP_USP_SERVICE_PORT")

	// Static port configuration - no service discovery needed

	// Static port configuration - use configured port directly
	grpcPort := deployConfig.ServicePort + 1 // gRPC is HTTP + 1

	fmt.Println("OpenUSP Core Service - Multi-Version TR-369 Protocol Engine")
	fmt.Println("==========================================================")

	// Print version info
	version.PrintVersionInfo("OpenUSP Core Service")
	fmt.Println()

	// Load configuration
	cfg := config.Load()

	// Initialize the USP core service (for TR-181 data model validation)
	_, err := NewUSPCoreService()
	if err != nil {
		log.Fatalf("Failed to initialize USP Core Service: %v", err)
	}

	log.Printf("✅ USP Core Service initialized with TR-181 data model")

	// Create connection client for dynamic service discovery
	connectionClient := client.NewOpenUSPConnectionClient(30 * time.Second)
	if err != nil {
		log.Fatalf("Failed to create connection client: %v", err)
	}
	log.Printf("✅ Created connection client for dynamic service discovery")

	// Get Data Service client via connection manager (dynamic discovery)
	dataClient, err := connectionClient.GetDataServiceClient()
	if err != nil {
		log.Fatalf("Failed to get data service client via connection manager: %v", err)
	}
	log.Printf("✅ Connected to Data Service via connection manager")

	// Start HTTP server for health checks and metrics - static port configuration
	httpPort := 6400 // Static HTTP port for USP Service

	go func() {
		mux := http.NewServeMux()

		// Health check endpoint
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := fmt.Sprintf(`{
				"service": "usp-service",
				"status": "healthy",
				"version": "1.0.0",
				"grpc_port": %d,
				"http_port": %d,
				"timestamp": "%s"
			}`, grpcPort, httpPort, time.Now().Format(time.RFC3339))
			w.Write([]byte(response))
		})

		// Status endpoint
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := fmt.Sprintf(`{
				"service": "usp-service",
				"status": "running",
				"version": "1.0.0",
				"grpc_port": %d,
				"http_port": %d,
				"tr181_objects": 822,
				"usp_versions": ["1.3", "1.4"],
				"timestamp": "%s"
			}`, grpcPort, httpPort, time.Now().Format(time.RFC3339))
			w.Write([]byte(response))
		})

		// Metrics endpoint
		mux.Handle("/metrics", metrics.HTTPHandler())

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", httpPort),
			Handler: mux,
		}

		log.Printf("� Starting HTTP server on port %d", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", grpcPort, err)
	}

	// Configure gRPC server with keepalive enforcement to prevent ENHANCE_YOUR_CALM errors
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    60 * time.Second, // Send keepalive pings every 60 seconds
			Timeout: 10 * time.Second, // Wait 10 seconds for keepalive ping ack
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second, // Minimum allowed time between client pings
			PermitWithoutStream: true,             // Allow pings even when no streams are active
		}),
	)
	uspServiceServer := grpcImpl.NewUSPServiceServer(dataClient, connectionClient)
	uspservice.RegisterUSPServiceServer(grpcServer, uspServiceServer)

	// Start gRPC server in background
	go func() {
		log.Printf("🚀 USP Service gRPC server starting on port %d", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	log.Printf("🚀 USP Service started successfully")
	log.Printf("   └── gRPC Port: %d", grpcPort)
	log.Printf("   └── HTTP Port: %d", httpPort)
	log.Printf("   └── Static Port Configuration: ✅ Enabled")
	log.Printf("   └── Health Check: http://localhost:%d/health", httpPort)
	log.Printf("   └── Status: http://localhost:%d/status", httpPort)
	log.Printf("   └── TR-181 Device:2 data model loaded")
	log.Printf("   └── USP protocol versions 1.3 and 1.4 supported")
	fmt.Printf("   🔧 HTTP Endpoints: http://localhost:%d/health, /status, /metrics\n", httpPort)
	fmt.Println("   🔧 TR-181 Device:2 data model loaded")
	fmt.Println("   🔧 USP protocol versions 1.3 and 1.4 supported")
	fmt.Println("   🔧 Automatic version detection enabled")
	fmt.Println("   🔧 Device onboarding and lifecycle management")
	fmt.Printf("   🔧 Data Service: localhost:%s\n", cfg.DataServicePort)

	// Handle graceful shutdown
	fmt.Println("\n💡 Press Ctrl+C to exit...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Printf("� Received shutdown signal")

	// Static port configuration - no service deregistration needed

	grpcServer.GracefulStop()
	log.Printf("✅ USP Service stopped successfully")
}
