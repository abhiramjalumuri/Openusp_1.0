# OpenUSP Makefile
# TR-369 User Service Platform - Professional Build System

# Environment Configuration
include configs/version.env
include configs/openusp.env
export VERSION
export RELEASE_NAME
export RELEASE_DATE

# Build Configuration
BINARY_DIR := build
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null || echo 'unknown')
GIT_DIRTY := $(shell git diff --quiet 2>/dev/null || echo 'dirty')
BUILD_USER := $(shell whoami 2>/dev/null || echo 'unknown')
LDFLAGS := -ldflags "-X openusp/pkg/version.Version=$(VERSION) -X openusp/pkg/version.GitCommit=$(GIT_COMMIT) -X openusp/pkg/version.BuildDate=$(BUILD_DATE)"

# Service Configuration
SERVICES := api-gateway data-service usp-service mtp-service cwmp-service
DOCKER_COMPOSE_INFRA := deployments/docker-compose.infra.yml

# Export unified environment variables for use in targets
export OPENUSP_DB_HOST
export OPENUSP_DB_PORT
export OPENUSP_DB_NAME
export OPENUSP_DB_USER
export OPENUSP_DB_PASSWORD
export OPENUSP_DB_SSLMODE
export OPENUSP_DATA_SERVICE_ADDR
export OPENUSP_API_GATEWAY_PORT
export OPENUSP_DATA_SERVICE_GRPC_PORT
export OPENUSP_MTP_SERVICE_PORT
export OPENUSP_CWMP_SERVICE_PORT
export OPENUSP_USP_WS_URL
export OPENUSP_CWMP_ACS_URL
export OPENUSP_CWMP_USERNAME
export OPENUSP_CWMP_PASSWORD

# PHONY targets
.PHONY: help version infra-up infra-down infra-status infra-clean
.PHONY: infra-volumes setup-grafana
.PHONY: build-ou-all build-all clean-ou-all start-ou-all stop-ou-all ou-status consul-status
.PHONY: start-all stop-all clean-all show-static-endpoints
.PHONY: $(addprefix build-,$(SERVICES))
.PHONY: $(addprefix start-,$(SERVICES))
.PHONY: $(addprefix clean-,$(SERVICES))
.PHONY: $(addprefix logs-,$(SERVICES))
# Legacy consul targets removed - all services now support runtime --consul flag
# Legacy consul build targets removed
.PHONY: build-all-consul

# Logging configuration (use unified environment variable)
LOG_DIR := $(OPENUSP_LOG_DIR)
INFRA_VOLUMES := \
	openusp-postgres-data \
	openusp-rabbitmq-dev-data \
	openusp-mosquitto-dev-data \
	openusp-mosquitto-dev-logs \
	openusp-prometheus-dev-data \
	openusp-grafana-dev-data

.DEFAULT_GOAL := help


help:
	@echo "OpenUSP - TR-369 User Service Platform"
	@echo "======================================"
	@echo ""
	@echo "Infrastructure Targets:"
	@echo "  infra-up        Start all 3rd party services"
	@echo "  infra-down      Stop all 3rd party services"
	@echo "  infra-status    Show status of all 3rd party services"
	@echo "  infra-clean     Clean/remove all 3rd party services and volumes"
	@echo "  infra-volumes   List all named infrastructure volumes (OpenUSP)"
	@echo "  setup-grafana   Configure Grafana with OpenUSP dashboards and data sources"
	@echo "  verify-grafana  Verify Grafana dashboard data and connectivity"
	@echo ""
	@echo "OpenUSP Service Targets:"
	@echo "  start-ou-all    Start all OpenUSP services (requires infra-up first)"
	@echo "  stop-ou-all     Stop all OpenUSP services"
	@echo "  build-ou-all    Build all OpenUSP services"
	@echo "  clean-ou-all    Clean all OpenUSP service binaries"
	@echo "  ou-status       Show status of all OpenUSP services (Consul-aware)"
	@echo ""
	@echo "Combined Targets:"
	@echo "  start-all         Start all services (infrastructure + OpenUSP)"
	@echo "  stop-all        Stop all services (infrastructure + OpenUSP)"
	@echo "  clean-all       Clean all services (infrastructure + OpenUSP)"
	@echo "  show-static-endpoints  List all public/local service endpoints"
	@echo ""
	@echo "Agent Targets (Pure YAML Configuration):" 
	@echo "  build-tr369-agent   Build TR-369 USP agent binary"
	@echo "  build-tr069-agent   Build TR-069 agent binary" 
	@echo "  start-tr369-agent   Start TR-369 agent with default YAML config"
	@echo "  start-tr069-agent   Start TR-069 agent"
	@echo ""
	@echo "Service Management (all services support --consul flag for service discovery):"
	@echo "  build-all           Build all components (services + agents)"
	@echo "  build-ou-all        Build all OpenUSP services"
	@echo "  start-all           Start all services (background)"
	@echo "  stop-all            Stop all services"
	@echo "  USP WebSocket URL: $(OPENUSP_USP_WS_URL)"
	@echo "  CWMP ACS URL: $(OPENUSP_CWMP_ACS_URL) (auth: $(OPENUSP_CWMP_USERNAME)/$(OPENUSP_CWMP_PASSWORD))"
	@echo ""
	@echo "Individual Service Targets:"
	@echo "  build-<service> Build individual service"
	@echo "  start-<service> Start individual service in foreground (logs to logs/<service>.log)"
	@echo "  clean-<service> Clean individual service binary"
	@echo "  logs-<service>  Stream (tail -f) logs for a service (requires service started)"
	@echo ""
	@echo "Service Name Map: api-gateway | data-service | usp-service | mtp-service | cwmp-service"
	@echo "Examples: make start-api-gateway | make logs-api-gateway"
	@echo ""
	@echo "Logs: Stored under ./logs (<service>.log). Use: tail -f logs/api-gateway.log"
	@echo "Version: $(VERSION) Commit: $(GIT_COMMIT) Build Time: $(BUILD_DATE)"
	@echo ""
	@echo "Quick Start: make start-all  # then open Swagger UI -> http://localhost:$(OPENUSP_API_GATEWAY_PORT)/swagger/index.html"
	@echo ""
	@echo "Go Development Tools:"
	@echo "  fmt             Format Go code (go fmt ./...)"
	@echo "  vet             Vet Go code for common mistakes"
	@echo "  lint            Run golangci-lint (install required)"
	@echo "  tidy            Tidy Go modules"
	@echo "  test-syntax     Test Go syntax without running tests"
	@echo "  go-check        Run all Go quality checks (fmt, vet, tidy, test-syntax)"
	@echo ""
	@echo "Tip: Run 'make show-static-endpoints' for a full endpoint & credentials overview"
	@echo "     Run 'make version' for version and protocol information"

# Version Information
version:
	@echo "OpenUSP Version Information"
	@echo "=========================="
	@echo "Version: $(VERSION)"
	@echo "Release: $(RELEASE_NAME) ($(RELEASE_DATE))"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(shell go version)"
	@echo ""
	@echo "Protocol Support:"
	@echo "  USP Versions: $(USP_VERSION_MIN) - $(USP_VERSION_MAX)"
	@echo "  CWMP Version: $(CWMP_VERSION)"
	@echo "  API Version: $(API_VERSION)"

# =============================================================================
# Go Development Tools
# =============================================================================

.PHONY: fmt vet lint tidy test-syntax

# Format Go code
fmt:
	@echo "🎨 Formatting Go code..."
	@go fmt ./...
	@echo "✅ Code formatting complete"

# Vet Go code for common mistakes
vet:
	@echo "🔍 Vetting Go code..."
	@go vet ./...
	@echo "✅ Code vetting complete"

# Run golangci-lint (install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	@echo "🧹 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "✅ Linting complete"; \
	else \
		echo "⚠️  golangci-lint not installed. Install with:"; \
		echo "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Tidy Go modules
tidy:
	@echo "📦 Tidying Go modules..."
	@go mod tidy
	@echo "✅ Module tidying complete"

# Test Go syntax without running tests
test-syntax:
	@echo "🧪 Testing Go syntax..."
	@go build -o /dev/null ./...
	@echo "✅ Syntax test complete"

# Run all Go quality checks
go-check: fmt vet tidy test-syntax
	@echo "✅ All Go quality checks passed"

# =============================================================================
# Agent Build and Start Targets (Pure YAML Configuration)
# =============================================================================
.PHONY: build-tr369-agent build-tr069-agent start-tr369-agent start-tr069-agent


# Build individual agents
build-tr369-agent:
	@echo "Building TR-369 USP agent..."
	@mkdir -p $(BINARY_DIR)
	@go build $(LDFLAGS) -o $(BINARY_DIR)/tr369-agent examples/tr369-agent/main.go
	@echo "TR-369 USP agent built -> $(BINARY_DIR)/tr369-agent"

build-tr069-agent:
	@echo "Building TR-069 agent..."
	@mkdir -p $(BINARY_DIR)
	@go build $(LDFLAGS) -o $(BINARY_DIR)/tr069-agent examples/tr069-agent/main.go
	@echo "TR-069 agent built -> $(BINARY_DIR)/tr069-agent"

# Start agents with default YAML configuration
start-tr369-agent: build-tr369-agent
	@echo "Starting TR-369 agent with default YAML configuration..."
	@echo "Configuration: configs/tr369-agent.yaml"
	@$(BINARY_DIR)/tr369-agent --config configs/tr369-agent.yaml || echo "Agent exited"

start-tr069-agent: build-tr069-agent
	@echo "Starting TR-069 agent..."
	@$(BINARY_DIR)/tr069-agent || echo "Agent exited"


# =================================
# Configuration and Testing Targets  
# =================================

start-all-consul:
	@echo "🚀 Starting All OpenUSP Services with Consul Service Discovery"
	@echo "=============================================================="
	@echo ""
	@echo "⚠️  This will start all services in background."
	@echo "   Use Ctrl+C and run individual 'make start-*-consul' for interactive mode."
	@echo ""
	@echo "Prerequisites: make infra-up (PostgreSQL + Consul + etc.)"
	@echo ""
	@echo "Starting services..."
	@echo "1. Data Service (Database + gRPC)..."
	@nohup make start-data-consul > logs/data-service-consul.log 2>&1 & echo "   └── Data Service started (PID: $$!)"
	@sleep 3
	@echo "2. API Gateway (REST + Service Discovery)..."
	@nohup make start-api-consul > logs/api-gateway-consul.log 2>&1 & echo "   └── API Gateway started (PID: $$!)"
	@sleep 2
	@echo "3. MTP Service (WebSocket + USP)..."
	@nohup make start-mtp-consul > logs/mtp-service-consul.log 2>&1 & echo "   └── MTP Service started (PID: $$!)"
	@sleep 2
	@echo "4. USP Core Service (gRPC)..."
	@nohup make start-usp-consul > logs/usp-service-consul.log 2>&1 & echo "   └── USP Service started (PID: $$!)"
	@sleep 2
	@echo "5. CWMP Service (TR-069)..."
	@nohup make start-cwmp-consul > logs/cwmp-service-consul.log 2>&1 & echo "   └── CWMP Service started (PID: $$!)"
	@sleep 3
	@echo ""
	@echo "✅ All services started! Check status:"
	@echo "   🔍 Consul UI: http://localhost:8500/ui/"
	@echo "   📊 Service Status: make ou-status"
	@echo "   📋 Individual Logs: tail -f logs/*-consul.log"
	@echo ""
	@echo "🧪 Test with TR-369 client: make start-tr369-consul"

stop-all-consul:
	@echo "🛑 Stopping All Consul Services"
	@echo "==============================="
	@pkill -f "cmd.*-consul/main.go" || echo "No Consul services running"
	@echo "✅ All Consul services stopped"

# Infrastructure Commands
infra-up:
	@echo "Starting all 3rd party infrastructure services..."
	@docker-compose -f $(DOCKER_COMPOSE_INFRA) up -d --remove-orphans
	@echo "Infrastructure services started successfully"

infra-down:
	@echo "Stopping all 3rd party infrastructure services..."
	@docker-compose -f $(DOCKER_COMPOSE_INFRA) down 2>/dev/null || true
	@echo "Infrastructure services stopped"

infra-status:
	@echo "Infrastructure Services Status:"
	@echo "==============================="
	@docker-compose -f $(DOCKER_COMPOSE_INFRA) ps 2>/dev/null || echo "  Infrastructure services not unning"

ou-status:
	@echo "OpenUSP Services Status:"
	@echo "========================"
	@echo ""
	@# Check if Consul is available
	@consul_available=$$(curl -s http://localhost:8500/v1/agent/services 2>/dev/null | grep -o 'openusp' 2>/dev/null || echo ""); \
	if [ -n "$$consul_available" ]; then \
		echo "🏛️  Service Discovery Mode: Consul (Dynamic Ports)"; \
		echo "🔍 Checking Consul-registered services..."; \
		echo ""; \
		printf "%-20s %-10s %-15s %-10s %s\n" "SERVICE" "STATUS" "PORT" "HEALTH" "ENDPOINT"; \
		printf "%-20s %-10s %-15s %-10s %s\n" "-------" "------" "----" "------" "--------"; \
		services=$$(curl -s http://localhost:8500/v1/agent/services 2>/dev/null | jq -r 'to_entries[] | select(.value.Service | startswith("openusp-")) | "\(.value.Service):\(.value.Port)"' 2>/dev/null || echo ""); \
		if [ -n "$$services" ]; then \
			echo "$$services" | while IFS=: read -r service_name port; do \
				display_name=$$(echo $$service_name | sed 's/openusp-//'); \
				health=$$(curl -s "http://localhost:8500/v1/agent/checks" 2>/dev/null | jq -r ".[] | select(.ServiceName == \"$$service_name\") | .Status" 2>/dev/null | head -1); \
				if [ "$$health" = "passing" ]; then \
					health_icon="✅"; \
				elif [ "$$health" = "critical" ]; then \
					health_icon="❌"; \
				else \
					health_icon="⚠️"; \
				fi; \
				endpoint="http://localhost:$$port"; \
				printf "%-20s %-10s %-15s %-10s %s\n" "$$display_name" "🔗 CONSUL" "$$port" "$$health_icon $$health" "$$endpoint"; \
			done; \
		else \
			echo "No OpenUSP services found in Consul"; \
		fi; \
	else \
		echo "🔧 Service Discovery Mode: Traditional (Fixed Ports)"; \
		echo "🔍 Checking traditional service processes and ports..."; \
		echo ""; \
		printf "%-20s %-10s %-15s %-10s %s\n" "SERVICE" "STATUS" "PORT" "PID" "ENDPOINT"; \
		printf "%-20s %-10s %-15s %-10s %s\n" "-------" "------" "----" "---" "--------"; \
		pid=$$(ps aux | grep -v grep | grep 'api-gateway' | awk '{print $$2}' | head -1); \
		port_check=$$(lsof -ti:$(OPENUSP_API_GATEWAY_PORT) 2>/dev/null); \
		if [ -n "$$pid" ] && [ -n "$$port_check" ]; then \
			printf "%-20s %-10s %-15s %-10s %s\n" "api-gateway" "✅ RUNNING" "$(OPENUSP_API_GATEWAY_PORT)" "$$pid" "http://localhost:$(OPENUSP_API_GATEWAY_PORT)"; \
		else \
			printf "%-20s %-10s %-15s %-10s %s\n" "api-gateway" "❌ STOPPED" "$(OPENUSP_API_GATEWAY_PORT)" "-" "-"; \
		fi; \
		port_check=$$(lsof -ti:$(OPENUSP_DATA_SERVICE_GRPC_PORT) 2>/dev/null | head -1); \
		if [ -n "$$port_check" ]; then \
			pid=$$(lsof -ti:$(OPENUSP_DATA_SERVICE_GRPC_PORT) 2>/dev/null | head -1); \
			printf "%-20s %-10s %-15s %-10s %s\n" "data-service" "✅ RUNNING" "$(OPENUSP_DATA_SERVICE_GRPC_PORT)" "$$pid" "gRPC://localhost:$(OPENUSP_DATA_SERVICE_GRPC_PORT)"; \
		else \
			printf "%-20s %-10s %-15s %-10s %s\n" "data-service" "❌ STOPPED" "$(OPENUSP_DATA_SERVICE_GRPC_PORT)" "-" "-"; \
		fi; \
		pid=$$(ps aux | grep -v grep | grep 'mtp-service' | awk '{print $$2}' | head -1); \
		port_check=$$(lsof -ti:$(OPENUSP_MTP_SERVICE_PORT) 2>/dev/null); \
		if [ -n "$$pid" ] && [ -n "$$port_check" ]; then \
			printf "%-20s %-10s %-15s %-10s %s\n" "mtp-service" "✅ RUNNING" "$(OPENUSP_MTP_SERVICE_PORT)" "$$pid" "http://localhost:$(OPENUSP_MTP_SERVICE_PORT)"; \
		else \
			printf "%-20s %-10s %-15s %-10s %s\n" "mtp-service" "❌ STOPPED" "$(OPENUSP_MTP_SERVICE_PORT)" "-" "-"; \
		fi; \
		pid=$$(ps aux | grep -v grep | grep 'cwmp-service' | awk '{print $$2}' | head -1); \
		port_check=$$(lsof -ti:$(OPENUSP_CWMP_SERVICE_PORT) 2>/dev/null); \
		if [ -n "$$pid" ] && [ -n "$$port_check" ]; then \
			printf "%-20s %-10s %-15s %-10s %s\n" "cwmp-service" "✅ RUNNING" "$(OPENUSP_CWMP_SERVICE_PORT)" "$$pid" "http://localhost:$(OPENUSP_CWMP_SERVICE_PORT)"; \
		else \
			printf "%-20s %-10s %-15s %-10s %s\n" "cwmp-service" "❌ STOPPED" "$(OPENUSP_CWMP_SERVICE_PORT)" "-" "-"; \
		fi; \
		pid=$$(ps aux | grep -v grep | grep 'usp-service' | awk '{print $$2}' | head -1); \
		port_check=$$(lsof -ti:$(OPENUSP_USP_SERVICE_PORT) 2>/dev/null); \
		if [ -n "$$pid" ] && [ -n "$$port_check" ]; then \
			printf "%-20s %-10s %-15s %-10s %s\n" "usp-service" "✅ RUNNING" "$(OPENUSP_USP_SERVICE_PORT)" "$$pid" "http://localhost:$(OPENUSP_USP_SERVICE_PORT)"; \
		else \
			printf "%-20s %-10s %-15s %-10s %s\n" "usp-service" "❌ STOPPED" "$(OPENUSP_USP_SERVICE_PORT)" "-" "-"; \
		fi; \
	fi
	@echo ""
	@echo "� Quick Health Check:"
	@# Try to find actual running services and test their health endpoints
	@data_services=$$(ps aux | grep -v grep | grep 'data-service' | awk '{print $$2}' || echo ""); \
	cwmp_services=$$(ps aux | grep -v grep | grep 'cwmp-service' | awk '{print $$2}' || echo ""); \
	if [ -n "$$data_services" ]; then \
		data_port=$$(lsof -Pan -p $$(echo $$data_services | head -1) 2>/dev/null | grep LISTEN | grep -v '127.0.0.1:.*->127.0.0.1' | head -1 | sed 's/.*:\([0-9]*\).*/\1/' || echo ""); \
		if [ -n "$$data_port" ]; then \
			response=$$(curl -s http://127.0.0.1:$$data_port/health 2>/dev/null | grep -o '"status"' 2>/dev/null || echo ""); \
			if [ -n "$$response" ]; then \
				echo "  ✅ Data Service is responding on port $$data_port"; \
			else \
				echo "  ⚠️  Data Service running on port $$data_port but not responding to /health"; \
			fi; \
		else \
			echo "  ⚠️  Data Service process found but port detection failed"; \
		fi; \
	else \
		echo "  ❌ Data Service not running"; \
	fi; \
	if [ -n "$$cwmp_services" ]; then \
		cwmp_port=$$(lsof -Pan -p $$(echo $$cwmp_services | head -1) 2>/dev/null | grep LISTEN | grep -v '127.0.0.1:.*->127.0.0.1' | head -1 | sed 's/.*:\([0-9]*\).*/\1/' || echo ""); \
		if [ -n "$$cwmp_port" ]; then \
			response=$$(curl -s http://127.0.0.1:$$cwmp_port/health 2>/dev/null | grep -o '"status"' 2>/dev/null || echo ""); \
			if [ -n "$$response" ]; then \
				echo "  ✅ CWMP Service is responding on port $$cwmp_port"; \
			else \
				echo "  ⚠️  CWMP Service running on port $$cwmp_port but not responding to /health"; \
			fi; \
		else \
			echo "  ⚠️  CWMP Service process found but port detection failed"; \
		fi; \
	else \
		echo "  ❌ CWMP Service not running"; \
	fi

infra-clean:
	@echo "Cleaning all 3rd party infrastructure services and named volumes..."
	@docker-compose -f $(DOCKER_COMPOSE_INFRA) down --remove-orphans 2>/dev/null || true
	@echo "Removing named volumes (ignore 'No such volume' warnings if already gone)..."
	@for v in $(INFRA_VOLUMES); do \
		if docker volume inspect $$v >/dev/null 2>&1; then \
			echo "  - removing $$v"; docker volume rm -f $$v >/dev/null 2>&1 || true; \
		else \
			echo "  - $$v (already removed)"; \
		fi; \
	done
	@echo "Removing network openusp-dev (if exists)..."
	@docker network rm openusp-dev 2>/dev/null || true
	@echo "Pruning dangling volumes & builder cache (safe)..."
	@docker system prune -f --volumes >/dev/null 2>&1 || true
	@echo "Remaining OpenUSP-related volumes (if any):"
	@docker volume ls | awk '{print $$2}' | grep -E '^openusp-' || echo "  (none)"
	@echo "Infrastructure cleaned"

infra-volumes:
	@echo "Named Infrastructure Volumes:"
	@for v in $(INFRA_VOLUMES); do echo "  - $$v"; done
	@echo "Use: make infra-clean to remove them"

setup-grafana:
	@echo "Setting up Grafana with OpenUSP dashboards..."
	@./scripts/setup-grafana-dashboards.sh

verify-grafana:
	@echo "Verifying Grafana dashboard data and connectivity..."
	@./scripts/verify-grafana.sh

# OpenUSP Service Commands
build-ou-all: $(addprefix build-,$(SERVICES))
	@echo "All OpenUSP services built successfully"

# Build everything (services + agents)
build-all: build-ou-all
	@echo "All OpenUSP components built successfully (services + agents)"

clean-ou-all: $(addprefix clean-,$(SERVICES))
	@echo "All OpenUSP service binaries cleaned"

start-ou-all:
	@echo "Starting all OpenUSP services..."
	@echo "Checking for port conflicts..."
	@if lsof -i :$(OPENUSP_API_GATEWAY_PORT) >/dev/null 2>&1; then \
		echo "ERROR: Port $(OPENUSP_API_GATEWAY_PORT) is in use. Run 'make stop-ou-all' or 'make infra-down' first."; \
		exit 1; \
	fi
	@if ! docker ps --format "table {{.Names}}" | grep -q openusp-postgres; then \
		echo "ERROR: PostgreSQL not running. Run 'make infra-up' first."; \
		exit 1; \
	fi
	@echo "Starting Data Service with Consul (PostgreSQL dependency)..."
	@$(MAKE) start-data-service-consul &
	@sleep 8
	@echo "Starting remaining services with Consul..."
	@$(MAKE) start-api-gateway-consul &
	@$(MAKE) start-mtp-service-consul &
	@$(MAKE) start-cwmp-service-consul &
	@$(MAKE) start-usp-service-consul &
	@sleep 2
	@echo "All OpenUSP services started (check individual logs via: make logs-<service>)"

stop-ou-all:
	@echo "Stopping all OpenUSP services..."
	@pkill -f "api-gateway" 2>/dev/null || true
	@pkill -f "data-service" 2>/dev/null || true
	@pkill -f "usp-service" 2>/dev/null || true
	@pkill -f "mtp-service" 2>/dev/null || true
	@pkill -f "cwmp-service" 2>/dev/null || true
	@# Kill any processes on OpenUSP ports as fallback
	@lsof -ti:$(OPENUSP_API_GATEWAY_PORT) 2>/dev/null | xargs kill -9 2>/dev/null || true
	@lsof -ti:$(OPENUSP_DATA_SERVICE_HTTP_PORT) 2>/dev/null | xargs kill -9 2>/dev/null || true
	@lsof -ti:$(OPENUSP_MTP_SERVICE_PORT) 2>/dev/null | xargs kill -9 2>/dev/null || true
	@lsof -ti:$(OPENUSP_CWMP_SERVICE_PORT) 2>/dev/null | xargs kill -9 2>/dev/null || true
	@lsof -ti:$(OPENUSP_USP_SERVICE_PORT) 2>/dev/null | xargs kill -9 2>/dev/null || true
	@sleep 1
	@echo "All OpenUSP services stopped"

# Combined Commands
start-all: infra-up build-ou-all start-ou-all
	@echo "All services (infrastructure + OpenUSP) are running!"

stop-all: stop-ou-all infra-down
	@echo "All services stopped"

clean-all: clean-ou-all infra-clean
	@echo "All services cleaned"

# Endpoint Overview
show-static-endpoints:
	@echo "Service Endpoints (unified configuration)"
	@echo "========================================"
	@echo "API Gateway:      http://localhost:$(OPENUSP_API_GATEWAY_PORT)"
	@echo "  Health:         http://localhost:$(OPENUSP_API_GATEWAY_PORT)/health"
	@echo "  Status:         http://localhost:$(OPENUSP_API_GATEWAY_PORT)/status"
	@echo "  Swagger UI:     http://localhost:$(OPENUSP_API_GATEWAY_PORT)/swagger/index.html"
	@echo "Data Service:     localhost:$(OPENUSP_DATA_SERVICE_GRPC_PORT) (gRPC)"
	@echo "MTP Demo UI:      http://localhost:$(OPENUSP_MTP_SERVICE_PORT)/usp"
	@echo "MTP WebSocket:    ws://localhost:$(OPENUSP_MTP_SERVICE_PORT)/ws"
	@echo "MTP Health:       http://localhost:$(OPENUSP_MTP_SERVICE_PORT)/health"
	@echo "CWMP Service:     http://localhost:$(OPENUSP_CWMP_SERVICE_PORT)"
	@echo "CWMP Health:      http://localhost:$(OPENUSP_CWMP_SERVICE_PORT)/health"
	@echo "  Auth:           $(OPENUSP_CWMP_USERNAME) / $(OPENUSP_CWMP_PASSWORD)"
	@echo "Prometheus:       http://localhost:$(OPENUSP_PROMETHEUS_PORT)"
	@echo "Grafana:          http://localhost:$(OPENUSP_GRAFANA_PORT)"
	@echo "  Login:          $(OPENUSP_GRAFANA_USER) / $(OPENUSP_GRAFANA_PASSWORD)"
	@echo "RabbitMQ:         http://localhost:$(OPENUSP_RABBITMQ_MGMT_PORT)"
	@echo "  Credentials:    $(OPENUSP_RABBITMQ_USER) / $(OPENUSP_RABBITMQ_PASSWORD)"
	@echo "MQTT Broker:      mqtt://localhost:$(OPENUSP_MOSQUITTO_PORT)"
	@echo "MQTT WebSocket:   ws://localhost:$(OPENUSP_MOSQUITTO_WS_PORT)"
	@echo "PostgreSQL:       localhost:$(OPENUSP_POSTGRES_PORT)"
	@echo "Adminer:          http://localhost:$(OPENUSP_ADMINER_PORT)"
	@echo "  DB Credentials: $(OPENUSP_DB_USER) / $(OPENUSP_DB_PASSWORD) ($(OPENUSP_DB_NAME))"
	@echo "------------------------------------------------"
	@echo "Use: make start-all (start everything) | make stop-all | make show-static-endpoints"

# Swagger Documentation Generation
.PHONY: swagger-gen
swagger-gen: ## Generate Swagger documentation
	@echo "📚 Generating Swagger documentation..."
	@which swag > /dev/null || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	@swag init -g cmd/api-gateway/main.go -o api/ --parseInternal
	@echo "✅ Swagger documentation generated in api/"

# Individual Service Build Targets
build-api-gateway: swagger-gen
	@echo "Building API Gateway..."
	@mkdir -p $(BINARY_DIR)
	@go build $(LDFLAGS) -o $(BINARY_DIR)/api-gateway cmd/api-gateway/main.go
	@echo "API Gateway built -> $(BINARY_DIR)/api-gateway"

build-data-service:
	@echo "Building Data Service..."
	@mkdir -p $(BINARY_DIR)
	@go build $(LDFLAGS) -o $(BINARY_DIR)/data-service cmd/data-service/main.go
	@echo "Data Service built -> $(BINARY_DIR)/data-service"

build-usp-service:
	@echo "Building USP Service..."
	@mkdir -p $(BINARY_DIR)
	@go build $(LDFLAGS) -o $(BINARY_DIR)/usp-service cmd/usp-service/main.go
	@echo "USP Service built -> $(BINARY_DIR)/usp-service"

build-mtp-service:
	@echo "Building MTP Service..."
	@mkdir -p $(BINARY_DIR)
	@go build $(LDFLAGS) -o $(BINARY_DIR)/mtp-service cmd/mtp-service/main.go
	@echo "MTP Service built -> $(BINARY_DIR)/mtp-service"

build-cwmp-service:
	@echo "Building CWMP Service..."
	@mkdir -p $(BINARY_DIR)
	@go build $(LDFLAGS) -o $(BINARY_DIR)/cwmp-service cmd/cwmp-service/main.go
	@echo "CWMP Service built -> $(BINARY_DIR)/cwmp-service"

# Consul Service Build Targets
# Legacy consul build targets removed - all services now use single binary with --consul flag

# build-all-consul: Updated to use single binaries with --consul flag
build-all-consul:
	@echo "Building all services (with --consul flag support)..."
	@make build-data-service
	@make build-api-gateway
	@make build-mtp-service
	@make build-usp-service
	@make build-cwmp-service
	@echo "✅ All Consul services built"

# Individual Service Start Targets
start-api-gateway:
	@echo "Starting API Gateway on port $(OPENUSP_API_GATEWAY_PORT) (traditional mode)..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] API Gateway starting (port=$(OPENUSP_API_GATEWAY_PORT))" > $(OPENUSP_LOG_DIR)/api-gateway.log
	@SERVICE_PORT=$(OPENUSP_API_GATEWAY_PORT) DATA_SERVICE_ADDR=$(OPENUSP_DATA_SERVICE_ADDR) go run cmd/api-gateway/main.go --port $(OPENUSP_API_GATEWAY_PORT) 2>&1 | tee -a $(OPENUSP_LOG_DIR)/api-gateway.log

start-api-gateway-consul:
	@echo "Starting API Gateway with Consul service discovery..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] API Gateway starting (Consul mode)" > $(OPENUSP_LOG_DIR)/api-gateway-consul.log
	@CONSUL_ENABLED=true DATA_SERVICE_ADDR=$(OPENUSP_DATA_SERVICE_ADDR) go run cmd/api-gateway/main.go --consul 2>&1 | tee -a $(OPENUSP_LOG_DIR)/api-gateway-consul.log

start-data-service:
	@echo "Starting Data Service on port $(OPENUSP_DATA_SERVICE_HTTP_PORT) (traditional mode)..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] Data Service starting (port=$(OPENUSP_DATA_SERVICE_HTTP_PORT))" > $(OPENUSP_LOG_DIR)/data-service.log
	@echo "Database config: $(OPENUSP_DB_HOST):$(OPENUSP_DB_PORT)/$(OPENUSP_DB_NAME) (user: $(OPENUSP_DB_USER))" >> $(OPENUSP_LOG_DIR)/data-service.log
	@DB_HOST=$(OPENUSP_DB_HOST) DB_PORT=$(OPENUSP_DB_PORT) DB_NAME=$(OPENUSP_DB_NAME) DB_USER=$(OPENUSP_DB_USER) DB_PASSWORD=$(OPENUSP_DB_PASSWORD) DB_SSLMODE=$(OPENUSP_DB_SSLMODE) go run cmd/data-service/main.go --port $(OPENUSP_DATA_SERVICE_HTTP_PORT) 2>&1 | tee -a $(OPENUSP_LOG_DIR)/data-service.log

start-data-service-consul:
	@echo "Starting Data Service with Consul service discovery..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] Data Service starting (Consul mode)" > $(OPENUSP_LOG_DIR)/data-service-consul.log
	@echo "Database config: $(OPENUSP_DB_HOST):$(OPENUSP_DB_PORT)/$(OPENUSP_DB_NAME) (user: $(OPENUSP_DB_USER))" >> $(OPENUSP_LOG_DIR)/data-service-consul.log
	@DB_HOST=$(OPENUSP_DB_HOST) DB_PORT=$(OPENUSP_DB_PORT) DB_NAME=$(OPENUSP_DB_NAME) DB_USER=$(OPENUSP_DB_USER) DB_PASSWORD=$(OPENUSP_DB_PASSWORD) DB_SSLMODE=$(OPENUSP_DB_SSLMODE) CONSUL_ENABLED=true go run cmd/data-service/main.go --consul 2>&1 | tee -a $(OPENUSP_LOG_DIR)/data-service-consul.log

start-usp-service:
	@echo "Starting USP Service on port $(OPENUSP_USP_SERVICE_GRPC_PORT) (traditional mode)..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] USP Service starting (port=$(OPENUSP_USP_SERVICE_GRPC_PORT))" > $(OPENUSP_LOG_DIR)/usp-service.log
	@go run cmd/usp-service/main.go --port $(OPENUSP_USP_SERVICE_GRPC_PORT) 2>&1 | tee -a $(OPENUSP_LOG_DIR)/usp-service.log

start-usp-service-consul:
	@echo "Starting USP Service with Consul service discovery..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] USP Service starting (Consul mode)" > $(OPENUSP_LOG_DIR)/usp-service-consul.log
	@CONSUL_ENABLED=true go run cmd/usp-service/main.go --consul 2>&1 | tee -a $(OPENUSP_LOG_DIR)/usp-service-consul.log

start-mtp-service:
	@echo "Starting MTP Service on port $(OPENUSP_MTP_SERVICE_PORT) (traditional mode)..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] MTP Service starting (port=$(OPENUSP_MTP_SERVICE_PORT))" > $(OPENUSP_LOG_DIR)/mtp-service.log
	@go run cmd/mtp-service/main.go --port $(OPENUSP_MTP_SERVICE_PORT) 2>&1 | tee -a $(OPENUSP_LOG_DIR)/mtp-service.log

start-mtp-service-consul:
	@echo "Starting MTP Service with Consul service discovery..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] MTP Service starting (Consul mode)" > $(OPENUSP_LOG_DIR)/mtp-service-consul.log
	@CONSUL_ENABLED=true go run cmd/mtp-service/main.go --consul 2>&1 | tee -a $(OPENUSP_LOG_DIR)/mtp-service-consul.log

start-cwmp-service:
	@echo "Starting CWMP Service on port $(OPENUSP_CWMP_SERVICE_PORT) (traditional mode)..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] CWMP Service starting (port=$(OPENUSP_CWMP_SERVICE_PORT))" > $(OPENUSP_LOG_DIR)/cwmp-service.log
	@go run cmd/cwmp-service/main.go --port $(OPENUSP_CWMP_SERVICE_PORT) 2>&1 | tee -a $(OPENUSP_LOG_DIR)/cwmp-service.log

start-cwmp-service-consul:
	@echo "Starting CWMP Service with Consul service discovery..."
	@mkdir -p $(OPENUSP_LOG_DIR)
	@echo "[start $(BUILD_DATE)] CWMP Service starting (Consul mode)" > $(OPENUSP_LOG_DIR)/cwmp-service-consul.log
	@CONSUL_ENABLED=true go run cmd/cwmp-service/main.go --consul 2>&1 | tee -a $(OPENUSP_LOG_DIR)/cwmp-service-consul.log

# Individual Service Clean Targets
clean-api-gateway:
	@echo "Cleaning API Gateway binary..."
	@rm -f $(BINARY_DIR)/api-gateway
	@echo "API Gateway binary cleaned"

clean-data-service:
	@echo "Cleaning Data Service binary..."
	@rm -f $(BINARY_DIR)/data-service
	@echo "Data Service binary cleaned"

clean-usp-service:
	@echo "Cleaning USP Service binary..."
	@rm -f $(BINARY_DIR)/usp-service
	@echo "USP Service binary cleaned"

clean-mtp-service:
	@echo "Cleaning MTP Service binary..."
	@rm -f $(BINARY_DIR)/mtp-service
	@echo "MTP Service binary cleaned"

clean-cwmp-service:
	@echo "Cleaning CWMP Service binary..."
	@rm -f $(BINARY_DIR)/cwmp-service
	@echo "CWMP Service binary cleaned"

# Log Targets (file-based tail)
logs-api-gateway:
	@test -f $(LOG_DIR)/api-gateway.log || { echo "No API Gateway log file. Start service first: make start-api-gateway"; exit 1; }
	@echo "Streaming API Gateway logs (Ctrl+C to exit)..."
	@tail -F $(LOG_DIR)/api-gateway.log

logs-data-service:
	@test -f $(LOG_DIR)/data-service.log || { echo "No Data Service log file. Start service first: make start-data-service"; exit 1; }
	@echo "Streaming Data Service logs (Ctrl+C to exit)..."
	@tail -F $(LOG_DIR)/data-service.log

logs-usp-service:
	@test -f $(LOG_DIR)/usp-service.log || { echo "No USP Service log file. Start service first: make start-usp-service"; exit 1; }
	@echo "Streaming USP Service logs (Ctrl+C to exit)..."
	@tail -F $(LOG_DIR)/usp-service.log

logs-mtp-service:
	@test -f $(LOG_DIR)/mtp-service.log || { echo "No MTP Service log file. Start service first: make start-mtp-service"; exit 1; }
	@echo "Streaming MTP Service logs (Ctrl+C to exit)..."
	@tail -F $(LOG_DIR)/mtp-service.log

logs-cwmp-service:
	@test -f $(LOG_DIR)/cwmp-service.log || { echo "No CWMP Service log file. Start service first: make start-cwmp-service"; exit 1; }
	@echo "Streaming CWMP Service logs (Ctrl+C to exit)..."
	@tail -F $(LOG_DIR)/cwmp-service.log
