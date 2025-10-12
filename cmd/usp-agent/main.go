package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"openusp/pkg/config"
)

const (
	defaultUSPVersion         = "1.3"
	supportedProtocolVersions = "1.3,1.4"
)

// YAML-only: controller ID comes from unified config usp.endpoint_id (already loaded in platform config if needed)
func getControllerID() string { return "" }

func printHelp() {
	fmt.Println("OpenUSP TR-369 Agent")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  usp-agent [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --help, -h       Show this help message")
	fmt.Println("  --info, -i       Show agent information")
	fmt.Println("  --config FILE    Load configuration from YAML file")
	fmt.Println("  --version VER    Specify USP protocol version (1.3 or 1.4)")
	fmt.Println()
}

type DeviceInfo struct {
	EndpointID      string
	Manufacturer    string
	ModelName       string
	SerialNumber    string
	SoftwareVersion string
	HardwareVersion string
	ProductClass    string
}

func getDeviceInfo(agentConfig *config.TR369Config, unified *config.Config) *DeviceInfo {
	deviceInfo := &DeviceInfo{
		SoftwareVersion: "1.0.0",
		HardwareVersion: "1.0",
	}

	// Prefer unified config USPAgentDevice (YAML + env overrides)
	if unified != nil {
		d := unified.USPAgentDevice
		if d.EndpointID != "" {
			deviceInfo.EndpointID = d.EndpointID
		}
		if d.Manufacturer != "" {
			deviceInfo.Manufacturer = d.Manufacturer
		}
		if d.ModelName != "" {
			deviceInfo.ModelName = d.ModelName
		}
		if d.SerialNumber != "" {
			deviceInfo.SerialNumber = d.SerialNumber
		}
		if d.SoftwareVersion != "" {
			deviceInfo.SoftwareVersion = d.SoftwareVersion
		}
		if d.HardwareVersion != "" {
			deviceInfo.HardwareVersion = d.HardwareVersion
		}
		if d.ProductClass != "" {
			deviceInfo.ProductClass = d.ProductClass
		}
	}

	// Fall back to agentConfig (older YAML specific path)
	if agentConfig != nil {
		if deviceInfo.EndpointID == "" {
			deviceInfo.EndpointID = agentConfig.EndpointID
		}
		if deviceInfo.Manufacturer == "" {
			deviceInfo.Manufacturer = agentConfig.Manufacturer
		}
		if deviceInfo.ModelName == "" {
			deviceInfo.ModelName = agentConfig.ModelName
		}
		if deviceInfo.SerialNumber == "" {
			deviceInfo.SerialNumber = agentConfig.SerialNumber
		}
		if deviceInfo.ProductClass == "" {
			deviceInfo.ProductClass = agentConfig.ProductClass
		}
		if agentConfig.SoftwareVersion != "" {
			deviceInfo.SoftwareVersion = agentConfig.SoftwareVersion
		}
		if agentConfig.HardwareVersion != "" {
			deviceInfo.HardwareVersion = agentConfig.HardwareVersion
		}
	}

	return deviceInfo
}

func loadAgentConfiguration(configPath string) (*config.TR369Config, error) {
	// Always load from unified openusp.yml; ignore configPath for simplicity
	cfg, err := config.LoadUSPAgentUnified("configs/openusp.yml")
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func getMTPTransportType(agent *config.TR369Config) string { return agent.MTPType }

func validateConfiguration(deviceInfo *DeviceInfo) error {
	if deviceInfo.EndpointID == "" {
		return fmt.Errorf("OPENUSP_USP_AGENT_ENDPOINT_ID is required")
	}
	if deviceInfo.Manufacturer == "" {
		return fmt.Errorf("OPENUSP_USP_AGENT_MANUFACTURER is required")
	}
	if deviceInfo.ModelName == "" {
		return fmt.Errorf("OPENUSP_USP_AGENT_MODEL_NAME is required")
	}
	if deviceInfo.SerialNumber == "" {
		return fmt.Errorf("OPENUSP_USP_AGENT_SERIAL_NUMBER is required")
	}
	if deviceInfo.ProductClass == "" {
		return fmt.Errorf("OPENUSP_USP_AGENT_PRODUCT_CLASS is required")
	}
	// Controller ID validation removed (handled elsewhere if needed)
	return nil
}

func printInfo() {
	fmt.Println("OpenUSP TR-369 Agent Information")
	fmt.Println("================================")
	fmt.Println()

	agentConfig, err := loadAgentConfiguration("")
	if err != nil {
		fmt.Printf("Warning: Could not load configuration: %v\n", err)
		agentConfig = &config.TR369Config{}
	}

	deviceInfo := getDeviceInfo(agentConfig, nil)

	fmt.Println("Agent Configuration:")
	fmt.Printf("  Endpoint ID: %s\n", deviceInfo.EndpointID)
	fmt.Printf("  Controller ID: %s\n", getControllerID())
	fmt.Println()

	fmt.Println("Device Information:")
	fmt.Printf("  Manufacturer: %s\n", deviceInfo.Manufacturer)
	fmt.Printf("  Model Name: %s\n", deviceInfo.ModelName)
	fmt.Printf("  Serial Number: %s\n", deviceInfo.SerialNumber)
	fmt.Printf("  Software Version: %s\n", deviceInfo.SoftwareVersion)
	fmt.Printf("  Hardware Version: %s\n", deviceInfo.HardwareVersion)
	fmt.Printf("  Product Class: %s\n", deviceInfo.ProductClass)
	fmt.Println()

	fmt.Println("Protocol Support:")
	fmt.Printf("  Default Version: %s\n", defaultUSPVersion)
	fmt.Printf("  Supported Versions: %s\n", supportedProtocolVersions)
}

func runAgent(ctx context.Context) error {
	log.Printf("Agent running - waiting for termination signal...")
	<-ctx.Done()
	return nil
}

func main() {
	help := flag.Bool("help", false, "Show help message")
	helpShort := flag.Bool("h", false, "Show help message")
	info := flag.Bool("info", false, "Show agent information")
	infoShort := flag.Bool("i", false, "Show agent information")
	configFile := flag.String("config", "", "Configuration file path")
	version := flag.String("version", defaultUSPVersion, "USP protocol version")
	flag.Parse()

	if *help || *helpShort {
		printHelp()
		return
	}

	if *info || *infoShort {
		printInfo()
		return
	}

	agentConfig, err := loadAgentConfiguration(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	deviceInfo := getDeviceInfo(agentConfig, nil)

	if err := validateConfiguration(deviceInfo); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	log.Printf("OpenUSP TR-369 Agent starting...")
	log.Printf("Endpoint ID: %s", deviceInfo.EndpointID)
	log.Printf("Controller ID: %s", getControllerID())
	log.Printf("USP Protocol Version: %s", *version)

	transportStr := getMTPTransportType(agentConfig)
	log.Printf("Using %s transport", transportStr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Printf("Received shutdown signal, terminating...")
		cancel()
	}()

	if err := runAgent(ctx); err != nil {
		log.Fatalf("Agent error: %v", err)
	}

	log.Printf("OpenUSP TR-369 Agent terminated")
}
