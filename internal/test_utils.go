package internal
// Package testutils provides test utilities for the OpenUSP platform
package testutils

import (
	"os"
	"testing"
)

// TestMain sets up test environment
func TestMain(m *testing.M) {
	// Set test environment variables
	os.Setenv("ENV", "test")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/openusp_test?sslmode=disable")
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	os.Exit(code)
}

// GetTestDatabaseURL returns test database connection string
func GetTestDatabaseURL() string {
	return "postgres://test:test@localhost:5432/openusp_test?sslmode=disable"
}