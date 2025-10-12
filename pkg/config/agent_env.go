package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadUSPAgentEnvConfig loads USP agent configuration from environment variables
func LoadUSPAgentEnvConfig() *TR369Config {
	config := &TR369Config{}

	// Agent Config (embedded)
	config.EndpointID = getEnvOrDefault("OPENUSP_USP_AGENT_ENDPOINT_ID", "")
	config.ProductClass = getEnvOrDefault("OPENUSP_USP_AGENT_PRODUCT_CLASS", "")
	config.Manufacturer = getEnvOrDefault("OPENUSP_USP_AGENT_MANUFACTURER", "")
	config.ModelName = getEnvOrDefault("OPENUSP_USP_AGENT_MODEL_NAME", "")
	config.SerialNumber = getEnvOrDefault("OPENUSP_USP_AGENT_SERIAL_NUMBER", "")
	config.SoftwareVersion = getEnvOrDefault("OPENUSP_USP_AGENT_SOFTWARE_VERSION", "")
	config.HardwareVersion = getEnvOrDefault("OPENUSP_USP_AGENT_HARDWARE_VERSION", "")
	config.DeviceType = getEnvOrDefault("OPENUSP_USP_AGENT_DEVICE_TYPE", "")
	config.OUI = getEnvOrDefault("OPENUSP_USP_AGENT_OUI", "")

	// USP Protocol
	config.USPVersion = getEnvOrDefault("OPENUSP_USP_AGENT_VERSION", "1.3")
	if versions := getEnvOrDefault("OPENUSP_USP_AGENT_SUPPORTED_VERSIONS", "1.3,1.4"); versions != "" {
		config.USPSupportedVersions = strings.Split(versions, ",")
	}
	config.USPAgentRole = getEnvOrDefault("OPENUSP_USP_AGENT_ROLE", "device")
	config.USPCommandKey = getEnvOrDefault("OPENUSP_USP_AGENT_COMMAND_KEY", "")

	// MTP Configuration
	config.MTPType = getEnvOrDefault("OPENUSP_USP_AGENT_MTP_TYPE", "websocket")

	// WebSocket MTP
	config.WebSocketURL = getEnvOrDefault("OPENUSP_USP_AGENT_WEBSOCKET_URL", "")
	config.WebSocketSubprotocol = getEnvOrDefault("OPENUSP_USP_AGENT_WEBSOCKET_SUBPROTOCOL", "v1.usp")

	// STOMP MTP
	config.STOMPBrokerURL = getEnvOrDefault("OPENUSP_USP_AGENT_STOMP_BROKER_URL", "")
	config.STOMPUsername = getEnvOrDefault("OPENUSP_USP_AGENT_STOMP_USERNAME", "")
	config.STOMPPassword = getEnvOrDefault("OPENUSP_USP_AGENT_STOMP_PASSWORD", "")
	config.STOMPDestinationRequest = getEnvOrDefault("OPENUSP_USP_AGENT_STOMP_DESTINATION_REQUEST", "/queue/usp.agent.request")
	config.STOMPDestinationResponse = getEnvOrDefault("OPENUSP_USP_AGENT_STOMP_DESTINATION_RESPONSE", "/queue/usp.agent.response")
	config.STOMPDestinationController = getEnvOrDefault("OPENUSP_USP_AGENT_STOMP_DESTINATION_CONTROLLER", "/queue/usp.controller")
	config.STOMPDestinationAgent = getEnvOrDefault("OPENUSP_USP_AGENT_STOMP_DESTINATION_AGENT", "/queue/usp.agent")
	config.STOMPDestinationBroadcast = getEnvOrDefault("OPENUSP_USP_AGENT_STOMP_DESTINATION_BROADCAST", "/topic/usp.broadcast")

	// MQTT MTP
	config.MQTTBrokerURL = getEnvOrDefault("OPENUSP_USP_AGENT_MQTT_BROKER_URL", "")
	config.MQTTClientID = getEnvOrDefault("OPENUSP_USP_AGENT_MQTT_CLIENT_ID", "")
	config.MQTTUsername = getEnvOrDefault("OPENUSP_USP_AGENT_MQTT_USERNAME", "")
	config.MQTTPassword = getEnvOrDefault("OPENUSP_USP_AGENT_MQTT_PASSWORD", "")

	// Platform Integration (fallback to existing values)
	config.PlatformURL = getEnvOrDefault("OPENUSP_API_GATEWAY_URL", "https://localhost:6500")

	return config
}

// LoadCWMPAgentEnvConfig loads CWMP agent configuration from environment variables
func LoadCWMPAgentEnvConfig() *TR069Config {
	config := &TR069Config{}

	// Agent Config (embedded)
	config.EndpointID = getEnvOrDefault("OPENUSP_CWMP_AGENT_ENDPOINT_ID", "")
	config.ProductClass = getEnvOrDefault("OPENUSP_CWMP_AGENT_PRODUCT_CLASS", "")
	config.Manufacturer = getEnvOrDefault("OPENUSP_CWMP_AGENT_MANUFACTURER", "")
	config.ModelName = getEnvOrDefault("OPENUSP_CWMP_AGENT_MODEL_NAME", "")
	config.SerialNumber = getEnvOrDefault("OPENUSP_CWMP_AGENT_SERIAL_NUMBER", "")
	config.SoftwareVersion = getEnvOrDefault("OPENUSP_CWMP_AGENT_SOFTWARE_VERSION", "")
	config.HardwareVersion = getEnvOrDefault("OPENUSP_CWMP_AGENT_HARDWARE_VERSION", "")
	config.DeviceType = getEnvOrDefault("OPENUSP_CWMP_AGENT_DEVICE_TYPE", "")
	config.OUI = getEnvOrDefault("OPENUSP_CWMP_AGENT_OUI", "")

	// CWMP Protocol
	config.CWMPVersion = getEnvOrDefault("OPENUSP_CWMP_AGENT_VERSION", "1.2")
	if versions := getEnvOrDefault("OPENUSP_CWMP_AGENT_SUPPORTED_VERSIONS", "1.0,1.1,1.2,1.3,1.4"); versions != "" {
		config.CWMPSupportedVersions = strings.Split(versions, ",")
	}
	config.CWMPAgentRole = getEnvOrDefault("OPENUSP_CWMP_AGENT_ROLE", "cpe")
	config.ACSParameterKey = getEnvOrDefault("OPENUSP_CWMP_AGENT_PARAMETER_KEY", "")

	// ACS Configuration
	config.ACSURL = getEnvOrDefault("OPENUSP_CWMP_AGENT_ACS_URL", "http://localhost:7547")
	config.ACSUsername = getEnvOrDefault("OPENUSP_CWMP_AGENT_ACS_USERNAME", "")
	config.ACSPassword = getEnvOrDefault("OPENUSP_CWMP_AGENT_ACS_PASSWORD", "")

	// Parse periodic inform enabled
	if enabledStr := getEnvOrDefault("OPENUSP_CWMP_AGENT_PERIODIC_INFORM_ENABLED", "true"); enabledStr != "" {
		config.ACSPeriodicInformEnabled, _ = strconv.ParseBool(enabledStr)
	}

	// Parse periodic inform interval
	if intervalStr := getEnvOrDefault("OPENUSP_CWMP_AGENT_PERIODIC_INFORM_INTERVAL", "300s"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			config.ACSPeriodicInformInterval = interval
		}
	}

	// Connection Request
	config.ConnectionRequestURL = getEnvOrDefault("OPENUSP_CWMP_AGENT_CR_URL", "http://localhost:9999/cr")
	config.ConnectionRequestUsername = getEnvOrDefault("OPENUSP_CWMP_AGENT_CR_USERNAME", "")
	config.ConnectionRequestPassword = getEnvOrDefault("OPENUSP_CWMP_AGENT_CR_PASSWORD", "")

	// Platform Integration (fallback to existing values)
	config.PlatformURL = getEnvOrDefault("OPENUSP_API_GATEWAY_URL", "https://localhost:6500")

	return config
}

// LoadAgentConfigWithEnvFallback loads agent configuration with environment variable fallback
func LoadAgentConfigWithEnvFallback(configPath string, agentType string) (interface{}, error) {
	// Try to load from YAML first
	if configPath != "" {
		if agentType == "usp" {
			if yamlConfig, err := LoadYAMLTR369Config(configPath); err == nil {
				return enrichUSPConfigWithEnv(yamlConfig), nil
			}
		} else if agentType == "cwmp" {
			if yamlConfig, err := LoadYAMLTR069Config(configPath); err == nil {
				return enrichCWMPConfigWithEnv(yamlConfig), nil
			}
		}
	}

	// Fallback to environment variables
	if agentType == "usp" {
		return LoadUSPAgentEnvConfig(), nil
	} else if agentType == "cwmp" {
		return LoadCWMPAgentEnvConfig(), nil
	}

	return nil, fmt.Errorf("unsupported agent type: %s", agentType)
}

// enrichUSPConfigWithEnv enriches YAML config with environment variable overrides
func enrichUSPConfigWithEnv(yamlConfig *TR369Config) *TR369Config {
	envConfig := LoadUSPAgentEnvConfig()

	// Override YAML values with environment variables if they exist
	if envConfig.EndpointID != "" {
		yamlConfig.EndpointID = envConfig.EndpointID
	}
	if envConfig.MTPType != "" {
		yamlConfig.MTPType = envConfig.MTPType
	}
	if envConfig.STOMPBrokerURL != "" {
		yamlConfig.STOMPBrokerURL = envConfig.STOMPBrokerURL
	}
	if envConfig.STOMPUsername != "" {
		yamlConfig.STOMPUsername = envConfig.STOMPUsername
	}
	if envConfig.STOMPPassword != "" {
		yamlConfig.STOMPPassword = envConfig.STOMPPassword
	}
	// Add more overrides as needed...

	return yamlConfig
}

// enrichCWMPConfigWithEnv enriches YAML config with environment variable overrides
func enrichCWMPConfigWithEnv(yamlConfig *TR069Config) *TR069Config {
	envConfig := LoadCWMPAgentEnvConfig()

	// Override YAML values with environment variables if they exist
	if envConfig.EndpointID != "" {
		yamlConfig.EndpointID = envConfig.EndpointID
	}
	if envConfig.ACSURL != "" {
		yamlConfig.ACSURL = envConfig.ACSURL
	}
	// Add more overrides as needed...

	return yamlConfig
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}
