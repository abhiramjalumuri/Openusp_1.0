package consul

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// FileServiceRegistry provides file-based service discovery without Consul
type FileServiceRegistry struct {
	registryDir  string
	services     map[string]*ServiceInfo
	mu           sync.RWMutex
	lastScan     time.Time
	scanInterval time.Duration
}

// NewFileServiceRegistry creates a new file-based service registry
func NewFileServiceRegistry() *FileServiceRegistry {
	registryDir := os.Getenv("OPENUSP_SERVICE_REGISTRY_DIR")
	if registryDir == "" {
		registryDir = "/tmp/openusp-services"
	}

	// Create registry directory if it doesn't exist
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		log.Printf("⚠️ Failed to create registry directory %s: %v", registryDir, err)
	}

	return &FileServiceRegistry{
		registryDir:  registryDir,
		services:     make(map[string]*ServiceInfo),
		scanInterval: 5 * time.Second, // Scan every 5 seconds
	}
}

// RegisterService registers a service in the file-based registry
func (fsr *FileServiceRegistry) RegisterService(serviceInfo *ServiceInfo) error {
	fsr.mu.Lock()
	defer fsr.mu.Unlock()

	// Create service file
	serviceFile := filepath.Join(fsr.registryDir, fmt.Sprintf("%s.json", serviceInfo.Name))

	// Add timestamp for health checking
	serviceInfo.Meta["registered_at"] = fmt.Sprintf("%d", time.Now().Unix())
	serviceInfo.Meta["pid"] = fmt.Sprintf("%d", os.Getpid())

	data, err := json.MarshalIndent(serviceInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	if err := os.WriteFile(serviceFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Update local cache
	fsr.services[serviceInfo.Name] = serviceInfo

	log.Printf("📝 Registered service %s at %s:%d (file: %s)",
		serviceInfo.Name, serviceInfo.Address, serviceInfo.Port, serviceFile)

	return nil
}

// DiscoverService finds a service by name using file-based registry
func (fsr *FileServiceRegistry) DiscoverService(serviceName string) (*ServiceInfo, error) {
	// Scan for services if needed
	if time.Since(fsr.lastScan) > fsr.scanInterval {
		fsr.scanServices()
	}

	fsr.mu.RLock()
	service, exists := fsr.services[serviceName]
	fsr.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service %s not found in file registry", serviceName)
	}

	// Check if service is still healthy (process still running)
	if !fsr.isServiceHealthy(service) {
		fsr.mu.Lock()
		delete(fsr.services, serviceName)
		fsr.mu.Unlock()
		return nil, fmt.Errorf("service %s is no longer healthy", serviceName)
	}

	return service, nil
}

// scanServices scans the registry directory for service files
func (fsr *FileServiceRegistry) scanServices() {
	fsr.mu.Lock()
	defer fsr.mu.Unlock()

	fsr.lastScan = time.Now()

	files, err := os.ReadDir(fsr.registryDir)
	if err != nil {
		log.Printf("⚠️ Failed to scan registry directory: %v", err)
		return
	}

	newServices := make(map[string]*ServiceInfo)

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			serviceFile := filepath.Join(fsr.registryDir, file.Name())

			data, err := os.ReadFile(serviceFile)
			if err != nil {
				log.Printf("⚠️ Failed to read service file %s: %v", serviceFile, err)
				continue
			}

			var serviceInfo ServiceInfo
			if err := json.Unmarshal(data, &serviceInfo); err != nil {
				log.Printf("⚠️ Failed to unmarshal service file %s: %v", serviceFile, err)
				continue
			}

			// Check if service is still healthy
			if fsr.isServiceHealthy(&serviceInfo) {
				newServices[serviceInfo.Name] = &serviceInfo
			} else {
				// Remove stale service file
				log.Printf("🧹 Removing stale service file: %s", serviceFile)
				os.Remove(serviceFile)
			}
		}
	}

	fsr.services = newServices
}

// isServiceHealthy checks if a service is still running
func (fsr *FileServiceRegistry) isServiceHealthy(service *ServiceInfo) bool {
	// Check if port is listening
	address := net.JoinHostPort(service.Address, fmt.Sprintf("%d", service.Port))
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()

	// Additional check: if we have PID, check if process is running
	if pidStr, exists := service.Meta["pid"]; exists {
		if pid, err := strconv.Atoi(pidStr); err == nil {
			if !isProcessRunning(pid) {
				return false
			}
		}
	}

	return true
}

// isProcessRunning checks if a process with given PID is running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix systems, Signal(0) checks if process exists
	err = process.Signal(os.Signal(nil))
	return err == nil
}

// Deregister removes a service from the registry
func (fsr *FileServiceRegistry) Deregister(serviceName string) error {
	fsr.mu.Lock()
	defer fsr.mu.Unlock()

	serviceFile := filepath.Join(fsr.registryDir, fmt.Sprintf("%s.json", serviceName))

	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	delete(fsr.services, serviceName)

	log.Printf("🗑️ Deregistered service %s", serviceName)
	return nil
}

// ListServices returns all registered services
func (fsr *FileServiceRegistry) ListServices() ([]*ServiceInfo, error) {
	// Scan for latest services
	fsr.scanServices()

	fsr.mu.RLock()
	defer fsr.mu.RUnlock()

	services := make([]*ServiceInfo, 0, len(fsr.services))
	for _, service := range fsr.services {
		services = append(services, service)
	}

	return services, nil
}

// GetServiceHealth returns the health status of a service
func (fsr *FileServiceRegistry) GetServiceHealth(serviceName string) (string, error) {
	service, err := fsr.DiscoverService(serviceName)
	if err != nil {
		return "critical", err
	}

	if fsr.isServiceHealthy(service) {
		return "passing", nil
	}

	return "critical", fmt.Errorf("service %s is not healthy", serviceName)
}
