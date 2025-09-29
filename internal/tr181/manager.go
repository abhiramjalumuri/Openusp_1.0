package tr181

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// DeviceManager manages the TR-181 Device data model for USP operations
type DeviceManager struct {
	namespace *DeviceNamespace
}

// NewDeviceManager creates a new Device Manager with the TR-181 data model
func NewDeviceManager(dataModelPath string) (*DeviceManager, error) {
	// Load the TR-181 XML file
	xmlData, err := os.ReadFile(dataModelPath)
	if err != nil {
		return nil, err
	}

	// Create the device namespace
	namespace, err := NewDeviceNamespace(xmlData)
	if err != nil {
		return nil, err
	}

	log.Printf("Loaded TR-181 Device:2 data model with %d objects", len(namespace.objects))

	return &DeviceManager{
		namespace: namespace,
	}, nil
}

// GetNamespace returns the TR-181 device namespace
func (dm *DeviceManager) GetNamespace() *DeviceNamespace {
	return dm.namespace
}

// LoadDefaultDataModel loads the TR-181 data model from the default path
func LoadDefaultDataModel() (*DeviceManager, error) {
	// Default path to the TR-181 XML file
	dataModelPath := filepath.Join("pkg", "datamodel", "tr-181-2-19-1-usp-full.xml")
	return NewDeviceManager(dataModelPath)
}

// GetDeviceInfo returns basic device information structure
func (dm *DeviceManager) GetDeviceInfo() map[string]interface{} {
	return map[string]interface{}{
		"Device.DeviceInfo.Manufacturer":    "OpenUSP",
		"Device.DeviceInfo.ManufacturerOUI": "001122",
		"Device.DeviceInfo.ModelName":       "USP-Agent-1.0",
		"Device.DeviceInfo.SerialNumber":    "USP001122334455",
		"Device.DeviceInfo.HardwareVersion": "1.0",
		"Device.DeviceInfo.SoftwareVersion": "1.0.0",
		"Device.DeviceInfo.Description":     "OpenUSP TR-369 Device Agent",
		"Device.DeviceInfo.ProductClass":    "USP-Device",
		"Device.DeviceInfo.UpTime":          3600,
	}
}

// ValidateUSPPath validates if a USP path is valid according to TR-181
func (dm *DeviceManager) ValidateUSPPath(path string) bool {
	return dm.namespace.ValidatePath(path)
}

// GetSupportedDataModel returns the supported data model information for USP GetSupportedDM
func (dm *DeviceManager) GetSupportedDataModel() map[string]interface{} {
	objectPaths := dm.namespace.GetSupportedObjectPaths()

	supportedObjects := make([]map[string]interface{}, 0)

	// Add key objects for USP support
	keyObjects := []string{
		"Device.",
		"Device.DeviceInfo.",
		"Device.LocalAgent.",
		"Device.WiFi.",
		"Device.Ethernet.",
		"Device.IP.",
	}

	for _, objPath := range keyObjects {
		if obj, exists := dm.namespace.GetObject(objPath); exists {
			objectInfo := map[string]interface{}{
				"path":        objPath,
				"access":      obj.Access,
				"minEntries":  obj.MinEntries,
				"maxEntries":  obj.MaxEntries,
				"version":     obj.Version,
				"description": obj.Description,
			}

			// Add parameters info
			params := dm.namespace.GetObjectParameters(objPath)
			paramInfo := make([]map[string]interface{}, 0)
			for _, param := range params {
				paramType, _ := dm.namespace.GetParameterType(objPath + param.Name)
				paramInfo = append(paramInfo, map[string]interface{}{
					"name":    param.Name,
					"access":  param.Access,
					"type":    paramType,
					"version": param.Version,
				})
			}
			objectInfo["parameters"] = paramInfo

			// Add commands info
			commands := dm.namespace.GetObjectCommands(objPath)
			commandInfo := make([]map[string]interface{}, 0)
			for _, cmd := range commands {
				commandInfo = append(commandInfo, map[string]interface{}{
					"name":    cmd.Name,
					"version": cmd.Version,
					"async":   cmd.Async,
				})
			}
			objectInfo["commands"] = commandInfo

			supportedObjects = append(supportedObjects, objectInfo)
		}
	}

	return map[string]interface{}{
		"dataModelVersion": "Device:2.19",
		"supportedObjects": supportedObjects,
		"totalObjects":     len(objectPaths),
	}
}

// CWMP-specific methods for TR-069 compatibility

// ParameterInfo represents parameter information for CWMP
type ParameterInfo struct {
	Name     string
	Writable bool
}

// GetParameterNames returns parameter names for CWMP GetParameterNames
func (dm *DeviceManager) GetParameterNames(parameterPath string, nextLevel bool) ([]ParameterInfo, error) {
	var paramInfos []ParameterInfo

	// Get object paths that match the parameter path
	objectPaths := dm.namespace.GetObjectPaths(parameterPath, nextLevel)

	for _, objPath := range objectPaths {
		// Add object path itself
		paramInfos = append(paramInfos, ParameterInfo{
			Name:     objPath,
			Writable: false, // Objects are not writable
		})

		// Add parameters if not next level only
		if !nextLevel {
			parameters := dm.namespace.GetObjectParameters(objPath)
			for _, param := range parameters {
				writable := param.Access == "readWrite"
				paramInfos = append(paramInfos, ParameterInfo{
					Name:     objPath + param.Name,
					Writable: writable,
				})
			}
		}
	}

	return paramInfos, nil
}

// GetParameterValue returns parameter value for CWMP GetParameterValues
func (dm *DeviceManager) GetParameterValue(parameterName string) (string, string, error) {
	// For demo purposes, return mock values based on parameter type
	// In production, this would query actual device values

	paramType, exists := dm.namespace.GetParameterType(parameterName)
	if !exists {
		return "", "", fmt.Errorf("parameter not found: %s", parameterName)
	}

	// Return mock values based on parameter type
	mockValue := dm.getMockValue(parameterName, paramType)

	return mockValue, paramType, nil
}

// SetParameterValue sets parameter value for CWMP SetParameterValues
func (dm *DeviceManager) SetParameterValue(parameterName, value, dataType string) error {
	// Validate parameter exists and is writable
	paramType, exists := dm.namespace.GetParameterType(parameterName)
	if !exists {
		return fmt.Errorf("parameter not found: %s", parameterName)
	}

	// Check if parameter is writable
	if !dm.namespace.IsParameterWritable(parameterName) {
		return fmt.Errorf("parameter is read-only: %s", parameterName)
	}

	// Validate data type matches
	if dataType != "" && dataType != paramType {
		return fmt.Errorf("data type mismatch for %s: expected %s, got %s",
			parameterName, paramType, dataType)
	}

	// In production, this would actually set the device parameter
	log.Printf("📝 CWMP Set Parameter: %s = %s (%s)", parameterName, value, paramType)

	return nil
}

// getMockValue returns mock values for demonstration
func (dm *DeviceManager) getMockValue(parameterName, paramType string) string {
	switch paramType {
	case "boolean":
		return "true"
	case "int", "unsignedInt":
		return "0"
	case "long", "unsignedLong":
		return "0"
	case "string":
		if contains(parameterName, "Name") {
			return "OpenUSP-Device"
		} else if contains(parameterName, "Version") {
			return "1.0.0"
		} else if contains(parameterName, "Address") {
			return "192.168.1.100"
		}
		return "mock-value"
	case "dateTime":
		return "2025-09-27T15:30:00Z"
	case "base64":
		return "bW9jay1kYXRh" // "mock-data" in base64
	default:
		return "unknown"
	}
}

// contains checks if a string contains a substring (helper function)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findSubstring(s, substr))))
}

// findSubstring is a simple substring search helper
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
