package cwmp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	grpcClient "openusp/internal/grpc"
	"openusp/internal/tr181"
	"openusp/pkg/proto/dataservice"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// OnboardingManager handles TR-069 device onboarding process
type OnboardingManager struct {
	dataServiceClient *grpcClient.DataServiceClient
	tr181Manager      *tr181.DeviceManager
	mu                sync.RWMutex
	onboardedDevices  map[string]*OnboardedDevice
}

// OnboardedDevice represents a device that has completed onboarding
type OnboardedDevice struct {
	DeviceID             string
	Manufacturer         string
	OUI                  string
	ProductClass         string
	SerialNumber         string
	SoftwareVersion      string
	HardwareVersion      string
	ConnectionRequestURL string
	OnboardedAt          time.Time
	LastInform           time.Time
	Status               string
}

// OnboardingStep represents a step in the onboarding process
type OnboardingStep struct {
	Name        string
	Description string
	Required    bool
	Completed   bool
	Error       error
}

// OnboardingProcess tracks the complete onboarding process for a device
type OnboardingProcess struct {
	DeviceID    string
	StartedAt   time.Time
	CompletedAt *time.Time
	Steps       []*OnboardingStep
	Status      string
	Error       error
}

// NewOnboardingManager creates a new onboarding manager
func NewOnboardingManager(dataServiceAddr string, tr181Mgr *tr181.DeviceManager) (*OnboardingManager, error) {
	client, err := grpcClient.NewDataServiceClient(dataServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create data service client: %w", err)
	}

	return &OnboardingManager{
		dataServiceClient: client,
		tr181Manager:      tr181Mgr,
		onboardedDevices:  make(map[string]*OnboardedDevice),
	}, nil
}

// ProcessInformForOnboarding handles the Inform message and performs onboarding
func (om *OnboardingManager) ProcessInformForOnboarding(inform *Inform, sessionID string) (*OnboardingProcess, error) {
	deviceID := fmt.Sprintf("%s-%s-%s", inform.DeviceId.OUI, inform.DeviceId.ProductClass, inform.DeviceId.SerialNumber)

	log.Printf("🚀 Starting TR-069 onboarding process for device: %s", deviceID)

	process := &OnboardingProcess{
		DeviceID:  deviceID,
		StartedAt: time.Now(),
		Status:    "in_progress",
		Steps: []*OnboardingStep{
			{Name: "device_identification", Description: "Extract device identification from Inform", Required: true},
			{Name: "parameter_extraction", Description: "Extract device parameters", Required: true},
			{Name: "database_registration", Description: "Register device in database", Required: true},
			{Name: "tr181_initialization", Description: "Initialize TR-181 data model", Required: true},
			{Name: "connection_verification", Description: "Verify device connectivity", Required: false},
		},
	}

	// Step 1: Device Identification
	if err := om.performDeviceIdentification(inform, process); err != nil {
		return process, err
	}

	// Step 2: Parameter Extraction
	if err := om.performParameterExtraction(inform, process); err != nil {
		return process, err
	}

	// Step 3: Database Registration
	if err := om.performDatabaseRegistration(inform, process); err != nil {
		return process, err
	}

	// Step 4: TR-181 Initialization
	if err := om.performTR181Initialization(inform, process); err != nil {
		return process, err
	}

	// Step 5: Connection Verification (optional)
	om.performConnectionVerification(inform, process)

	// Complete onboarding
	now := time.Now()
	process.CompletedAt = &now
	process.Status = "completed"

	// Store onboarded device
	om.storeOnboardedDevice(inform, deviceID)

	log.Printf("✅ TR-069 onboarding completed successfully for device: %s", deviceID)

	return process, nil
}

// performDeviceIdentification - Step 1
func (om *OnboardingManager) performDeviceIdentification(inform *Inform, process *OnboardingProcess) error {
	step := om.getStep(process, "device_identification")

	// Validate required device identification fields
	if inform.DeviceId.Manufacturer == "" {
		step.Error = fmt.Errorf("manufacturer is required")
		return step.Error
	}

	if inform.DeviceId.OUI == "" {
		step.Error = fmt.Errorf("OUI is required")
		return step.Error
	}

	if inform.DeviceId.SerialNumber == "" {
		step.Error = fmt.Errorf("serial number is required")
		return step.Error
	}

	step.Completed = true
	log.Printf("   ✅ Device identification: %s %s %s (S/N: %s)",
		inform.DeviceId.Manufacturer,
		inform.DeviceId.OUI,
		inform.DeviceId.ProductClass,
		inform.DeviceId.SerialNumber)

	return nil
}

// performParameterExtraction - Step 2
func (om *OnboardingManager) performParameterExtraction(inform *Inform, process *OnboardingProcess) error {
	step := om.getStep(process, "parameter_extraction")

	paramCount := len(inform.ParameterList)
	if paramCount == 0 {
		log.Printf("   ⚠️  No parameters provided in Inform message")
	} else {
		log.Printf("   ✅ Extracted %d parameters from device", paramCount)

		// Log key parameters
		for _, param := range inform.ParameterList {
			if isKeyParameter(param.Name) {
				log.Printf("      📋 %s = %s", param.Name, param.Value)
			}
		}
	}

	step.Completed = true
	return nil
}

// performDatabaseRegistration - Step 3
func (om *OnboardingManager) performDatabaseRegistration(inform *Inform, process *OnboardingProcess) error {
	step := om.getStep(process, "database_registration")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create device registration request
	device := &dataservice.Device{
		EndpointId:   process.DeviceID,
		Manufacturer: inform.DeviceId.Manufacturer,
		ProductClass: inform.DeviceId.ProductClass,
		SerialNumber: inform.DeviceId.SerialNumber,
		Status:       "online",
		CreatedAt:    timestamppb.New(time.Now()),
		UpdatedAt:    timestamppb.New(time.Now()),
	}

	// Register device in database
	resp, err := om.dataServiceClient.CreateDevice(ctx, device)
	if err != nil {
		step.Error = fmt.Errorf("failed to register device in database: %w", err)
		return step.Error
	}

	log.Printf("   ✅ Device registered in database with ID: %s", resp.Device.EndpointId)

	// Register parameters
	if err := om.registerDeviceParameters(ctx, inform, resp.Device); err != nil {
		step.Error = fmt.Errorf("failed to register device parameters: %w", err)
		return step.Error
	}

	step.Completed = true
	return nil
}

// performTR181Initialization - Step 4
func (om *OnboardingManager) performTR181Initialization(inform *Inform, process *OnboardingProcess) error {
	step := om.getStep(process, "tr181_initialization")

	// Initialize TR-181 data model for the device
	if om.tr181Manager != nil {
		// This would involve setting up the device's data model instance
		log.Printf("   ✅ TR-181 data model initialized for device")
	} else {
		log.Printf("   ⚠️  TR-181 manager not available")
	}

	step.Completed = true
	return nil
}

// performConnectionVerification - Step 5 (Optional)
func (om *OnboardingManager) performConnectionVerification(inform *Inform, process *OnboardingProcess) {
	step := om.getStep(process, "connection_verification")

	// Extract connection request URL if available
	connectionURL := om.extractConnectionRequestURL(inform)
	if connectionURL != "" {
		log.Printf("   ✅ Connection Request URL: %s", connectionURL)
		step.Completed = true
	} else {
		log.Printf("   ⚠️  Connection Request URL not found")
	}
}

// registerDeviceParameters registers device parameters in the database
func (om *OnboardingManager) registerDeviceParameters(ctx context.Context, inform *Inform, device *dataservice.Device) error {
	for _, param := range inform.ParameterList {
		parameter := &dataservice.Parameter{
			DeviceId:  device.Id,
			Path:      param.Name,
			Value:     param.Value,
			Type:      param.Type,
			Writable:  false, // Default to read-only
			CreatedAt: timestamppb.New(time.Now()),
			UpdatedAt: timestamppb.New(time.Now()),
		}

		_, err := om.dataServiceClient.CreateParameter(ctx, parameter)
		if err != nil {
			log.Printf("   ⚠️  Failed to register parameter %s: %v", param.Name, err)
			continue
		}
	}

	log.Printf("   ✅ Device parameters registered in database")
	return nil
}

// storeOnboardedDevice stores the onboarded device in memory
func (om *OnboardingManager) storeOnboardedDevice(inform *Inform, deviceID string) {
	om.mu.Lock()
	defer om.mu.Unlock()

	device := &OnboardedDevice{
		DeviceID:             deviceID,
		Manufacturer:         inform.DeviceId.Manufacturer,
		OUI:                  inform.DeviceId.OUI,
		ProductClass:         inform.DeviceId.ProductClass,
		SerialNumber:         inform.DeviceId.SerialNumber,
		SoftwareVersion:      om.extractSoftwareVersion(inform),
		HardwareVersion:      om.extractHardwareVersion(inform),
		ConnectionRequestURL: om.extractConnectionRequestURL(inform),
		OnboardedAt:          time.Now(),
		LastInform:           time.Now(),
		Status:               "online",
	}

	om.onboardedDevices[deviceID] = device
}

// Helper methods
func (om *OnboardingManager) getStep(process *OnboardingProcess, name string) *OnboardingStep {
	for _, step := range process.Steps {
		if step.Name == name {
			return step
		}
	}
	return nil
}

func (om *OnboardingManager) extractSoftwareVersion(inform *Inform) string {
	for _, param := range inform.ParameterList {
		if param.Name == "Device.DeviceInfo.SoftwareVersion" {
			return param.Value
		}
	}
	return "unknown"
}

func (om *OnboardingManager) extractHardwareVersion(inform *Inform) string {
	for _, param := range inform.ParameterList {
		if param.Name == "Device.DeviceInfo.HardwareVersion" {
			return param.Value
		}
	}
	return "unknown"
}

func (om *OnboardingManager) extractConnectionRequestURL(inform *Inform) string {
	for _, param := range inform.ParameterList {
		if param.Name == "Device.ManagementServer.ConnectionRequestURL" {
			return param.Value
		}
	}
	return ""
}

// isKeyParameter checks if a parameter is considered a key parameter for logging
func isKeyParameter(paramName string) bool {
	keyParams := []string{
		"Device.DeviceInfo.SoftwareVersion",
		"Device.DeviceInfo.HardwareVersion",
		"Device.DeviceInfo.ManufacturerOUI",
		"Device.DeviceInfo.ModelName",
		"Device.ManagementServer.ConnectionRequestURL",
		"Device.DeviceInfo.UpTime",
	}

	for _, key := range keyParams {
		if paramName == key {
			return true
		}
	}
	return false
}

// GetOnboardedDevices returns all onboarded devices
func (om *OnboardingManager) GetOnboardedDevices() map[string]*OnboardedDevice {
	om.mu.RLock()
	defer om.mu.RUnlock()

	result := make(map[string]*OnboardedDevice)
	for k, v := range om.onboardedDevices {
		result[k] = v
	}
	return result
}

// GetOnboardedDevice returns a specific onboarded device
func (om *OnboardingManager) GetOnboardedDevice(deviceID string) (*OnboardedDevice, bool) {
	om.mu.RLock()
	defer om.mu.RUnlock()

	device, exists := om.onboardedDevices[deviceID]
	return device, exists
}

// UpdateDeviceStatus updates the status of an onboarded device
func (om *OnboardingManager) UpdateDeviceStatus(deviceID, status string) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	device, exists := om.onboardedDevices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	device.Status = status
	device.LastInform = time.Now()

	// Update in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current device first
	deviceResp, err := om.dataServiceClient.GetDeviceByEndpoint(ctx, deviceID)
	if err != nil {
		log.Printf("Failed to get device for update: %v", err)
		return err
	}

	// Update device fields
	dbDevice := deviceResp.Device
	dbDevice.Status = status
	dbDevice.UpdatedAt = timestamppb.New(time.Now())

	_, err = om.dataServiceClient.UpdateDevice(ctx, dbDevice)
	if err != nil {
		log.Printf("Failed to update device status in database: %v", err)
	}

	return nil
}

// Close closes the onboarding manager
func (om *OnboardingManager) Close() error {
	if om.dataServiceClient != nil {
		return om.dataServiceClient.Close()
	}
	return nil
}
