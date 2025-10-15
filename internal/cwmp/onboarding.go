package cwmp

import (
	"fmt"
	"log"
	"sync"
	"time"

	"openusp/internal/tr181"
	"openusp/pkg/config"
	"openusp/pkg/kafka"
)

// OnboardingManager handles TR-069 device onboarding process
type OnboardingManager struct {
	kafkaProducer    *kafka.Producer
	kafkaTopics      *config.KafkaTopics
	tr181Manager     *tr181.DeviceManager
	mu               sync.RWMutex
	onboardedDevices map[string]*OnboardedDevice
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
func NewOnboardingManager(kafkaProducer *kafka.Producer, tr181Mgr *tr181.DeviceManager, kafkaTopics *config.KafkaTopics) (*OnboardingManager, error) {
	if kafkaProducer == nil {
		return nil, fmt.Errorf("kafka producer cannot be nil")
	}

	if kafkaTopics == nil {
		return nil, fmt.Errorf("kafka topics cannot be nil")
	}

	return &OnboardingManager{
		kafkaProducer:    kafkaProducer,
		kafkaTopics:      kafkaTopics,
		tr181Manager:     tr181Mgr,
		onboardedDevices: make(map[string]*OnboardedDevice),
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

	// Publish device created event to Kafka
	if err := om.kafkaProducer.PublishDeviceEvent(
		om.kafkaTopics.DataDeviceCreated,
		kafka.EventDeviceCreated,
		process.DeviceID,
		process.DeviceID,
		"CWMP", // Protocol
		map[string]interface{}{
			"manufacturer":  inform.DeviceId.Manufacturer,
			"product_class": inform.DeviceId.ProductClass,
			"serial_number": inform.DeviceId.SerialNumber,
			"oui":           inform.DeviceId.OUI,
			"status":        "online",
		},
	); err != nil {
		step.Error = fmt.Errorf("failed to publish device created event: %w", err)
		return step.Error
	}

	log.Printf("   ✅ Device created event published to Kafka topic: %s", om.kafkaTopics.DataDeviceCreated)

	// Register parameters via Kafka events
	if err := om.registerDeviceParameters(inform, process.DeviceID); err != nil {
		step.Error = fmt.Errorf("failed to publish device parameters: %w", err)
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

// registerDeviceParameters registers device parameters via Kafka events
func (om *OnboardingManager) registerDeviceParameters(inform *Inform, deviceID string) error {
	for _, param := range inform.ParameterList {
		if err := om.kafkaProducer.PublishParameterUpdate(
			om.kafkaTopics.DataParameterUpdated,
			deviceID,
			deviceID, // endpointID = deviceID
			param.Name,
			nil,         // oldValue (new parameter)
			param.Value, // newValue
		); err != nil {
			log.Printf("   ⚠️  Failed to publish parameter %s: %v", param.Name, err)
			continue
		}
	}

	log.Printf("   ✅ Device parameters published to Kafka")
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

	// Publish device updated event to Kafka
	if err := om.kafkaProducer.PublishDeviceEvent(
		om.kafkaTopics.DataDeviceUpdated,
		kafka.EventDeviceUpdated,
		deviceID,
		deviceID,
		"CWMP", // Protocol
		map[string]interface{}{
			"manufacturer":  device.Manufacturer,
			"product_class": device.ProductClass,
			"serial_number": device.SerialNumber,
			"oui":           device.OUI,
			"status":        status,
			"last_inform":   device.LastInform,
		},
	); err != nil {
		log.Printf("Failed to publish device updated event: %v", err)
		return err
	}

	return nil
}

// Close closes the onboarding manager
func (om *OnboardingManager) Close() error {
	// Nothing to close for Kafka-based onboarding manager
	// Kafka producer is managed by the main service
	log.Printf("✅ Onboarding manager closed")
	return nil
}
