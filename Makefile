# =============================================================================
# OpenUSP Makefile - TR-369 User Service Platform
# Professional Build System for Microservice Architecture
# =============================================================================

# =============================================================================
# Configuration and Environment
# =============================================================================

# Load configuration files
# Note: Parse manually to avoid variable conflicts
# include configs/version.env
# include configs/openusp.env
# export VERSION RELEASE_NAME RELEASE_DATE

# Build configuration
VERSION := $(shell grep "^VERSION=" configs/version.env | cut -d'=' -f2)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X openusp/pkg/version.Version=$(VERSION) -X openusp/pkg/version.GitCommit=$(GIT_COMMIT) -X openusp/pkg/version.BuildTime=$(BUILD_TIME)

# Go configuration
GO := go
GOBUILD := $(GO) build
GOCLEAN := $(GO) clean
GOTEST := $(GO) test
GOMOD := $(GO) mod

# Directory configuration
BUILD_DIR := build
LOG_DIR := logs

# Service definitions
SERVICES := api-gateway data-service usp-service mtp-service cwmp-service connection-manager
AGENTS := usp-agent cwmp-agent

# Docker configuration
DOCKER_COMPOSE_INFRA := deployments/docker-compose.infra.yml

# Infrastructure volumes
INFRA_VOLUMES := \
	openusp-postgres-data \
	openusp-rabbitmq-dev-data \
	openusp-mosquitto-dev-data \
	openusp-mosquitto-dev-logs \
	openusp-prometheus-dev-data \
	openusp-grafana-dev-data

# =============================================================================
# PHONY Targets Declaration
# =============================================================================

.PHONY: help version clean docs
.PHONY: $(addprefix build-,$(SERVICES) $(AGENTS))
.PHONY: $(addprefix start-,$(SERVICES) $(AGENTS))
.PHONY: $(addprefix stop-,$(SERVICES) $(AGENTS))
.PHONY: $(addprefix logs-,$(SERVICES) $(AGENTS))
.PHONY: build-all build-services build-agents
.PHONY: start-all start-services stop-all stop-services
.PHONY: clean-all clean-services clean-agents
.PHONY: infra-up infra-down infra-status infra-clean infra-volumes
.PHONY: setup-grafana verify-grafana
.PHONY: wait-for-connection-manager wait-for-data-service wait-for-usp-service
.PHONY: consul-status service-status consul-cleanup consul-cleanup-force dev-reset dev-restart dev-status
.PHONY: fmt vet lint tidy test go-check
.PHONY: endpoints quick-start

.DEFAULT_GOAL := help

# =============================================================================
# Help and Information
# =============================================================================

help:
	@echo "OpenUSP - TR-369 User Service Platform"
	@echo "======================================"
	@echo ""
	@echo "🚀 Quick Start:"
	@echo "  quick-start     Complete setup: infrastructure + build + start all services"
	@echo "  start-all       Start all services (requires infra-up and build-all first)"
	@echo "  endpoints       Show all service endpoints and credentials"
	@echo ""
	@echo "🏗️  Build Targets:"
	@echo "  build-all       Build all services and agents"
	@echo "  build-services  Build all microservices"
	@echo "  build-agents    Build all protocol agents"
	@echo "  build-<name>    Build specific service/agent"
	@echo ""
	@echo "🚦 Service Management:"
	@echo "  start-services  Start all OpenUSP microservices with proper dependencies"
	@echo "  stop-services   Stop all OpenUSP microservices"
	@echo "  start-<name>    Start specific service/agent"
	@echo "  stop-<name>     Stop specific service/agent"
	@echo "  logs-<name>     Show logs for specific service/agent"
	@echo ""
	@echo "🔧 Infrastructure:"
	@echo "  infra-up        Start all infrastructure services (PostgreSQL, Consul, etc.)"
	@echo "  infra-down      Stop all infrastructure services"
	@echo "  infra-status    Show infrastructure status"
	@echo "  infra-clean     Clean all infrastructure volumes"
	@echo ""
	@echo "📊 Monitoring:"
	@echo "  setup-grafana   Configure Grafana dashboards"
	@echo "  consul-status   Show Consul service registry"
	@echo "  service-status  Show all service health status"
	@echo ""
	@echo "🧹 Maintenance:"
	@echo "  clean-all       Clean all build artifacts"
	@echo "  go-check        Run all Go quality checks"
	@echo "  fmt             Format Go code"
	@echo "  vet             Run go vet"
	@echo "  lint            Run golangci-lint"
	@echo "  test            Run all tests"
	@echo ""
	@echo "📚 Documentation:"
	@echo "  docs            Show documentation guide"
	@echo "  version         Show version information"
	@echo "  endpoints       Show service endpoints"
	@echo ""
	@echo "📋 Available Services: $(SERVICES)"
	@echo "📋 Available Agents: $(AGENTS)"
	@echo ""
	@echo "💡 Examples:"
	@echo "  make quick-start                 # Complete setup"
	@echo "  make build-api-gateway          # Build API gateway"
	@echo "  make start-connection-manager   # Start connection manager"
	@echo "  make logs-usp-service           # View USP service logs"
	@echo ""
	@echo "📖 Version: $(VERSION) | Commit: $(GIT_COMMIT)"

version:
	@echo "OpenUSP Version Information"
	@echo "=========================="
	@echo "Version: $(VERSION)"
	@echo "Release: $(RELEASE_NAME) ($(RELEASE_DATE))"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(shell $(GO) version)"
	@echo ""
	@echo "Protocol Support:"
	@echo "  USP: $(USP_VERSION_MIN) - $(USP_VERSION_MAX)"
	@echo "  CWMP: $(CWMP_VERSION)"
	@echo "  API: $(API_VERSION)"

endpoints:
	@echo "OpenUSP Service Endpoints"
	@echo "========================"
	@echo ""
	@echo "🌐 Public Endpoints:"
	@echo "  API Gateway:     http://localhost:$(OPENUSP_API_GATEWAY_PORT)"
	@echo "  Swagger UI:      http://localhost:$(OPENUSP_API_GATEWAY_PORT)/swagger/index.html"
	@echo "  MTP Service:     http://localhost:$(OPENUSP_MTP_SERVICE_PORT)"
	@echo "  CWMP Service:    http://localhost:$(OPENUSP_CWMP_SERVICE_PORT)"
	@echo "  WebSocket:       $(OPENUSP_USP_WS_URL)"
	@echo "  CWMP ACS:        $(OPENUSP_CWMP_ACS_URL)"
	@echo ""
	@echo "🏛️  Infrastructure:"
	@echo "  Consul UI:       http://localhost:8500"
	@echo "  Grafana:         http://localhost:3000 (admin/admin)"
	@echo "  Prometheus:      http://localhost:9090"
	@echo "  PostgreSQL:      localhost:$(OPENUSP_DB_PORT) ($(OPENUSP_DB_USER)/$(OPENUSP_DB_PASSWORD))"
	@echo ""
	@echo "🔐 Credentials:"
	@echo "  CWMP Auth:       $(OPENUSP_CWMP_USERNAME)/$(OPENUSP_CWMP_PASSWORD)"
	@echo "  Database:        $(OPENUSP_DB_USER)/$(OPENUSP_DB_PASSWORD)"

docs:
	@echo "📚 OpenUSP Documentation"
	@echo "========================"
	@echo ""
	@echo "📖 Available Documentation:"
	@echo "  docs/README.md           - Documentation index"
	@echo "  docs/QUICKSTART.md       - 5-minute quick start"
	@echo "  docs/MAKEFILE_GUIDE.md   - Complete Makefile reference"
	@echo "  docs/DEVELOPMENT.md      - Development environment setup"
	@echo "  docs/USER_GUIDE.md       - User guide for device management"
	@echo "  docs/API_REFERENCE.md    - REST API documentation"
	@echo "  docs/TROUBLESHOOTING.md  - Troubleshooting guide"
	@echo "  docs/DEPLOYMENT.md       - Production deployment"
	@echo ""
	@echo "🚀 Quick Access:"
	@echo "  cat docs/MAKEFILE_GUIDE.md | less"
	@echo "  open docs/README.md"
	@echo ""
	@if command -v code >/dev/null 2>&1; then \
		echo "💡 Open in VS Code: code docs/"; \
	fi

# =============================================================================
# Quick Start and Combined Operations
# =============================================================================

quick-start: infra-up build-all start-services
	@echo ""
	@echo "🎉 OpenUSP Platform is ready!"
	@echo ""
	@echo "📊 Service Endpoints:"
	@echo "   Swagger UI: http://localhost:$(OPENUSP_API_GATEWAY_PORT)/swagger/index.html"
	@echo "   Consul UI:  http://localhost:8500"
	@echo "   Grafana:    http://localhost:3000"
	@echo ""
	@echo "📝 Next steps:"
	@echo "   make service-status    # Check all services"
	@echo "   make logs-api-gateway  # View API gateway logs"
	@echo "   make endpoints         # Show all endpoints"

start-all: infra-up build-all start-services

stop-all: stop-services infra-down

clean-all: clean-services clean-agents
	@echo "🧹 Cleaned all build artifacts"

# =============================================================================
# Build Targets
# =============================================================================

build-all: build-services build-agents

build-services: $(addprefix build-,$(SERVICES))
	@echo "✅ All services built successfully"

build-agents: $(addprefix build-,$(AGENTS))
	@echo "✅ All agents built successfully"

# Individual service build targets
define BUILD_SERVICE_TEMPLATE
build-$(1):
	@echo "🔨 Building $(1)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(1) ./cmd/$(1)
	@echo "✅ $(1) built successfully"
endef

$(foreach service,$(SERVICES),$(eval $(call BUILD_SERVICE_TEMPLATE,$(service))))
$(foreach agent,$(AGENTS),$(eval $(call BUILD_SERVICE_TEMPLATE,$(agent))))

# =============================================================================
# Service Management
# =============================================================================

start-services: build-services infra-up
	@echo ""
	@echo "🚀 Starting all OpenUSP services with proper dependency order..."
	@echo "📌 Step 1: Starting Connection Manager (core dependency)..."
	@$(MAKE) start-connection-manager-background
	@$(MAKE) wait-for-connection-manager
	@echo ""
	@echo "📌 Step 2: Starting Data Service (database layer)..."
	@$(MAKE) start-data-service-background
	@$(MAKE) wait-for-data-service
	@echo ""
	@echo "📌 Step 3: Starting USP Service (protocol engine)..."
	@$(MAKE) start-usp-service-background
	@$(MAKE) wait-for-usp-service
	@echo ""
	@echo "📌 Step 4: Starting remaining services..."
	@$(MAKE) start-api-gateway-background
	@$(MAKE) start-mtp-service-background
	@$(MAKE) start-cwmp-service-background
	@echo ""
	@sleep 3
	@echo "✅ All OpenUSP services started successfully"
	@echo "📋 Check status: make service-status"
	@echo "📝 View logs: make logs-<service>"

stop-services:
	@echo "🛑 Stopping all OpenUSP services..."
	@pkill -f "api-gateway.*--consul" 2>/dev/null || true
	@pkill -f "data-service.*--consul" 2>/dev/null || true
	@pkill -f "usp-service.*--consul" 2>/dev/null || true
	@pkill -f "mtp-service.*--consul" 2>/dev/null || true
	@pkill -f "cwmp-service.*--consul" 2>/dev/null || true
	@pkill -f "connection-manager.*--consul" 2>/dev/null || true
	@echo "✅ All OpenUSP services stopped"

clean-services:
	@echo "🧹 Cleaning service binaries..."
	@rm -f $(addprefix $(BUILD_DIR)/,$(SERVICES))
	@echo "✅ Service binaries cleaned"

clean-agents:
	@echo "🧹 Cleaning agent binaries..."
	@rm -f $(addprefix $(BUILD_DIR)/,$(AGENTS))
	@echo "✅ Agent binaries cleaned"

# Individual service start targets (background with logging)
define START_SERVICE_TEMPLATE
start-$(1): build-$(1)
	@echo "🚀 Starting $(1) in foreground..."
	@mkdir -p $(LOG_DIR)
	@CONSUL_ENABLED=true ./$(BUILD_DIR)/$(1) --consul

start-$(1)-background: build-$(1)
	@echo "🚀 Starting $(1) in background..."
	@mkdir -p $(LOG_DIR)
	@CONSUL_ENABLED=true ./$(BUILD_DIR)/$(1) --consul > $(LOG_DIR)/$(1).log 2>&1 &
	@sleep 2

stop-$(1):
	@echo "🛑 Stopping $(1)..."
	@pkill -f "$(1).*--consul" 2>/dev/null || true
	@echo "✅ $(1) stopped"

logs-$(1):
	@echo "📝 Showing logs for $(1)..."
	@tail -f $(LOG_DIR)/$(1).log
endef

# Generate service start/stop/logs targets (excludes agents which have custom configs)
$(foreach service,$(SERVICES),$(eval $(call START_SERVICE_TEMPLATE,$(service))))

# Agent management targets with YAML configuration
start-usp-agent: build-usp-agent
	@echo "🚀 Starting USP Agent with YAML configuration..."
	@mkdir -p $(LOG_DIR)
	@./$(BUILD_DIR)/usp-agent --config configs/usp-agent.yaml

start-cwmp-agent: build-cwmp-agent
	@echo "🚀 Starting CWMP Agent with YAML configuration..."
	@mkdir -p $(LOG_DIR)
	@./$(BUILD_DIR)/cwmp-agent --config configs/cwmp-agent.yaml

stop-usp-agent:
	@echo "🛑 Stopping USP Agent..."
	@pkill -f "usp-agent" 2>/dev/null || true
	@echo "✅ USP Agent stopped"

stop-cwmp-agent:
	@echo "🛑 Stopping CWMP Agent..."
	@pkill -f "cwmp-agent" 2>/dev/null || true
	@echo "✅ CWMP Agent stopped"

logs-usp-agent:
	@echo "📝 Showing logs for USP Agent..."
	@tail -f $(LOG_DIR)/usp-agent.log

logs-cwmp-agent:
	@echo "📝 Showing logs for CWMP Agent..."
	@tail -f $(LOG_DIR)/cwmp-agent.log

# =============================================================================
# Service Dependency Management
# =============================================================================

wait-for-connection-manager:
	@echo "⏳ Waiting for Connection Manager to be ready..."
	@timeout=30; \
	while [ $$timeout -gt 0 ]; do \
		if curl -s "http://localhost:8500/v1/catalog/service/openusp-connection-manager" | jq -e '.[0]' >/dev/null 2>&1; then \
			echo "✅ Connection Manager is registered and ready!"; \
			break; \
		fi; \
		echo -n "."; \
		sleep 2; \
		timeout=$$((timeout-2)); \
	done; \
	if [ $$timeout -le 0 ]; then \
		echo "❌ Connection Manager failed to become ready within 30 seconds"; \
		exit 1; \
	fi

wait-for-data-service:
	@echo "⏳ Waiting for Data Service to be ready..."
	@timeout=30; \
	while [ $$timeout -gt 0 ]; do \
		if curl -s "http://localhost:8500/v1/catalog/service/openusp-data-service" | jq -e '.[0]' >/dev/null 2>&1; then \
			echo "✅ Data Service is registered and ready!"; \
			break; \
		fi; \
		echo -n "."; \
		sleep 2; \
		timeout=$$((timeout-2)); \
	done; \
	if [ $$timeout -le 0 ]; then \
		echo "❌ Data Service failed to become ready within 30 seconds"; \
		exit 1; \
	fi

wait-for-usp-service:
	@echo "⏳ Waiting for USP Service to be ready..."
	@timeout=30; \
	while [ $$timeout -gt 0 ]; do \
		if curl -s "http://localhost:8500/v1/catalog/service/openusp-usp-service" | jq -e '.[0]' >/dev/null 2>&1; then \
			echo "✅ USP Service is registered and ready!"; \
			break; \
		fi; \
		echo -n "."; \
		sleep 2; \
		timeout=$$((timeout-2)); \
	done; \
	if [ $$timeout -le 0 ]; then \
		echo "❌ USP Service failed to become ready within 30 seconds"; \
		exit 1; \
	fi

# =============================================================================
# Infrastructure Management
# =============================================================================

infra-up:
	@echo "🏗️  Starting infrastructure services..."
	@docker-compose -f $(DOCKER_COMPOSE_INFRA) up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo "✅ Infrastructure services started"
	@$(MAKE) infra-status

infra-down:
	@echo "🛑 Stopping infrastructure services..."
	@docker-compose -f $(DOCKER_COMPOSE_INFRA) down
	@echo "✅ Infrastructure services stopped"

infra-status:
	@echo "📊 Infrastructure Status:"
	@docker-compose -f $(DOCKER_COMPOSE_INFRA) ps

infra-clean:
	@echo "🧹 Cleaning infrastructure (this will remove all data!)..."
	@read -p "Are you sure? This will delete all volumes and data. (y/N): " confirm && \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		docker-compose -f $(DOCKER_COMPOSE_INFRA) down -v; \
		docker volume rm $(INFRA_VOLUMES) 2>/dev/null || true; \
		echo "✅ Infrastructure cleaned"; \
	else \
		echo "❌ Cancelled"; \
	fi

infra-volumes:
	@echo "📦 Infrastructure Volumes:"
	@docker volume ls --filter name=openusp

# =============================================================================
# Monitoring and Status
# =============================================================================

setup-grafana:
	@echo "📊 Setting up Grafana dashboards..."
	@./scripts/setup-grafana.sh
	@echo "✅ Grafana setup complete"

verify-grafana:
	@echo "🔍 Verifying Grafana setup..."
	@./scripts/verify-grafana.sh

consul-status:
	@echo "🏛️  Consul Service Registry:"
	@curl -s http://localhost:8500/v1/catalog/services | jq 'keys[]' | grep openusp || echo "No OpenUSP services registered"

service-status:
	@echo "📊 OpenUSP Service Status:"
	@echo "=========================="
	@for service in connection-manager data-service usp-service api-gateway mtp-service cwmp-service; do \
		echo -n "$$service: "; \
		if curl -s "http://localhost:8500/v1/catalog/service/openusp-$$service" | jq -e '.[0]' >/dev/null 2>&1; then \
			echo "✅ Running"; \
		else \
			echo "❌ Not registered"; \
		fi; \
	done

# =============================================================================
# Development Environment Management
# =============================================================================

consul-cleanup:
	@echo "🧹 Cleaning up stale Consul registrations..."
	@./scripts/cleanup-consul.sh
	@echo "✅ Consul cleanup complete"

consul-cleanup-force:
	@echo "🔥 Force cleaning ALL Consul registrations..."
	@./scripts/cleanup-consul.sh --force
	@echo "✅ Force cleanup complete"

dev-reset:
	@echo "🔄 Resetting development environment..."
	@echo "1. Stopping all services..."
	@$(MAKE) stop-all || true
	@echo "2. Cleaning Consul registrations..."
	@./scripts/cleanup-consul.sh --force
	@echo "3. Building all services..."
	@$(MAKE) build-all
	@echo "4. Starting services..."
	@$(MAKE) start-all
	@echo "✅ Development environment reset complete"

dev-restart:
	@echo "🔄 Restarting OpenUSP services (preserving infrastructure)..."
	@echo "1. Cleanup Consul..."
	@./scripts/cleanup-consul.sh --force
	@echo "2. Stop services..."
	@$(MAKE) stop-all || true
	@sleep 2
	@echo "3. Start services..."
	@$(MAKE) start-all
	@echo "✅ Services restarted"

dev-status:
	@echo "📊 Development Environment Status"
	@echo "================================"
	@echo ""
	@echo "🏗️  Infrastructure:"
	@docker-compose -f $(DOCKER_COMPOSE_INFRA) ps --format "table {{.Service}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "Infrastructure not running"
	@echo ""
	@echo "🏛️  Service Discovery:"
	@if curl -s http://localhost:8500/v1/status/leader >/dev/null 2>&1; then \
		echo "Consul: ✅ Running"; \
		echo "Registered services:"; \
		curl -s http://localhost:8500/v1/catalog/services | jq -r 'keys[]' | grep openusp | sed 's/^/  - /' || echo "  No OpenUSP services"; \
	else \
		echo "Consul: ❌ Not running"; \
	fi
	@echo ""
	@echo "🚀 OpenUSP Services:"
	@$(MAKE) service-status
	@echo ""
	@echo "📊 System Resources:"
	@echo "Process count: $$(ps aux | grep -E '(usp-service|mtp-service|api-gateway|data-service|connection-manager|cwmp-service)' | grep -v grep | wc -l)"
	@echo "Memory usage: $$(ps aux | grep -E '(usp-service|mtp-service|api-gateway|data-service|connection-manager|cwmp-service)' | grep -v grep | awk '{sum += $$4} END {printf \"%.1f%%\", sum}')"

# =============================================================================
# Development and Quality Assurance
# =============================================================================

fmt:
	@echo "🎨 Formatting Go code..."
	@$(GO) fmt ./...
	@echo "✅ Code formatted"

vet:
	@echo "🔍 Running go vet..."
	@$(GO) vet ./...
	@echo "✅ Code vetted"

lint:
	@echo "🧹 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "✅ Linting complete"; \
	else \
		echo "⚠️  golangci-lint not installed. Install with:"; \
		echo "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

tidy:
	@echo "📦 Tidying Go modules..."
	@$(GOMOD) tidy
	@echo "✅ Modules tidied"

test:
	@echo "🧪 Running tests..."
	@$(GOTEST) -v ./...
	@echo "✅ Tests completed"

go-check: fmt vet tidy
	@echo "✅ All Go quality checks passed"

# =============================================================================
# Utility Targets
# =============================================================================

clean:
	@echo "🧹 Cleaning build directory..."
	@rm -rf $(BUILD_DIR)/*
	@echo "✅ Build directory cleaned"

# Create necessary directories
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

$(LOG_DIR):
	@mkdir -p $(LOG_DIR)

# =============================================================================
# End of Makefile
# =============================================================================
