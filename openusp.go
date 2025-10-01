// Package openusp provides the main entry point and documentation for the OpenUSP platform.
//
// OpenUSP is a cloud-native microservice implementation of the Broadband Forum's
// TR-369 based User Service Platform (USP). The platform provides integrated service
// suite with built-in features for remote device management, lifecycle management,
// and network management.
//
// This package serves as the root package for Go tooling compatibility and contains
// project-wide documentation and metadata.
//
// For service implementations, see:
//   - cmd/api-gateway - REST API Gateway service
//   - cmd/data-service - gRPC Data Service for database operations
//   - cmd/mtp-service - Message Transport Protocol service
//   - cmd/usp-service - USP Core service with TR-369 compliance
//   - cmd/cwmp-service - CWMP/TR-069 service for backward compatibility
//
// For library code, see:
//   - pkg/ - Public library packages
//   - internal/ - Private application packages
package openusp

// Version information for the OpenUSP platform
const (
	// PlatformName is the name of the platform
	PlatformName = "OpenUSP"

	// PlatformDescription is a brief description of the platform
	PlatformDescription = "TR-369 User Service Platform - Cloud-native microservice implementation"

	// PlatformLicense is the license under which the platform is distributed
	PlatformLicense = "Apache 2.0"
)
