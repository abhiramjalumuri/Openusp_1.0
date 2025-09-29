package main

import (
	"fmt"
	"log"
	"time"

	"openusp/internal/cwmp"
	"openusp/internal/tr181"
)

// TR069OnboardingDemo demonstrates the TR-069 onboarding functionality
func main() {
	fmt.Println("🚀 TR-069 Device Onboarding Demo")
	fmt.Println("================================")

	// Initialize TR-181 manager
	tr181Manager, err := tr181.NewDeviceManager("pkg/datamodel/tr-181-2-19-1-usp-full.xml")
	if err != nil {
		log.Fatalf("Failed to initialize TR-181 manager: %v", err)
	}

	// Initialize onboarding manager (using localhost for demo)
	onboardingManager, err := cwmp.NewOnboardingManager("localhost:56400", tr181Manager)
	if err != nil {
		log.Fatalf("Failed to initialize onboarding manager: %v", err)
	}
	defer func() {
		if err := onboardingManager.Close(); err != nil {
			log.Printf("Error closing onboarding manager: %v", err)
		}
	}()

	// Create a sample Inform message that would come from a TR-069 device
	sampleInform := &cwmp.Inform{
		DeviceId: cwmp.DeviceIdStruct{
			Manufacturer: "OpenUSP",
			OUI:          "00D4FE",
			ProductClass: "HomeGateway",
			SerialNumber: "DEMO123456",
		},
		Event: []cwmp.EventStruct{
			{
				EventCode:  "0 BOOTSTRAP",
				CommandKey: "",
			},
			{
				EventCode:  "1 BOOT",
				CommandKey: "",
			},
		},
		MaxEnvelopes: 1,
		CurrentTime:  time.Now().Format(time.RFC3339),
		RetryCount:   0,
		ParameterList: []cwmp.ParameterValueStruct{
			{
				Name:  "Device.DeviceInfo.Manufacturer",
				Value: "OpenUSP",
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.ProductClass",
				Value: "HomeGateway",
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.SerialNumber",
				Value: "DEMO123456",
				Type:  "xsd:string",
			},
			{
				Name:  "Device.DeviceInfo.SoftwareVersion",
				Value: "1.0.0",
				Type:  "xsd:string",
			},
			{
				Name:  "Device.ManagementServer.ConnectionRequestURL",
				Value: "http://192.168.1.100:7547/ConnectionRequest",
				Type:  "xsd:string",
			},
			{
				Name:  "Device.IP.Interface.1.IPv4Address.1.IPAddress",
				Value: "192.168.1.100",
				Type:  "xsd:string",
			},
		},
	}

	fmt.Printf("📱 Sample Device Information:\n")
	fmt.Printf("   Manufacturer: %s\n", sampleInform.DeviceId.Manufacturer)
	fmt.Printf("   Product Class: %s\n", sampleInform.DeviceId.ProductClass)
	fmt.Printf("   Serial Number: %s\n", sampleInform.DeviceId.SerialNumber)
	fmt.Printf("   Parameters: %d\n", len(sampleInform.ParameterList))
	fmt.Println()

	// Note: This demo shows the onboarding structure, but requires a running data service
	// to actually perform database operations
	fmt.Println("⚠️  Note: This demo shows the onboarding process structure.")
	fmt.Println("   For full functionality, the data service must be running at localhost:56400")
	fmt.Println()

	// Create an onboarding process (this will fail without data service, but shows the flow)
	process, err := onboardingManager.ProcessInformForOnboarding(sampleInform, "demo-session-123")
	if err != nil {
		fmt.Printf("❌ Onboarding process failed (expected without data service): %v\n", err)
		fmt.Println()
		fmt.Println("To see full onboarding in action:")
		fmt.Println("1. Start the data service: make run-data-service")
		fmt.Println("2. Start the CWMP service: make run-cwmp-service")
		fmt.Println("3. Use the TR-069 client example: make run-tr069-example")
		return
	}

	// Display process steps (this would show if successful)
	fmt.Printf("✅ Onboarding process created for device: %s\n", process.DeviceID)
	fmt.Printf("   Started at: %s\n", process.StartedAt.Format(time.RFC3339))
	fmt.Printf("   Steps: %d\n", len(process.Steps))

	for _, step := range process.Steps {
		status := "⏳ Pending"
		if step.Completed {
			status = "✅ Completed"
		} else if step.Error != nil {
			status = fmt.Sprintf("❌ Failed: %v", step.Error)
		}
		fmt.Printf("   - %s: %s\n", step.Name, status)
	}

	fmt.Println()
	fmt.Println("🎉 TR-069 Onboarding Demo Complete!")
}
