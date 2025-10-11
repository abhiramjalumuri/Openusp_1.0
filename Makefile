# =============================================================================
# OpenUSP Makefile - TR-369 User Service Platform
# Modern Build System for Microservice Architecture with Docker Networking
# =============================================================================

# =============================================================================
# Configuration
# =============================================================================

# Docker configuration - Static Port Infrastructure (No Consul)
DOCKER_COMPOSE_INFRA := deployments/docker-compose.infra.yml

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

# OpenUSP Service definitions
OPENUSP_SERVICES := api-gateway data-service connection-manager usp-service cwmp-service mtp-service
OPENUSP_AGENTS := usp-agent cwmp-agent

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
# PHONY target declarations
.PHONY: help
.PHONY: infra-up infra-down infra-status infra-clean infra-volumes
.PHONY: build build-services build-agents build-all
.PHONY: run run-services run-agents run-all
.PHONY: stop stop-services stop-agents stop-all force-stop stop-verify status-services status-quick status-debug
.PHONY: monitoring-cleanup prometheus-reload grafana-restart
.PHONY: $(addprefix build-,$(OPENUSP_SERVICES) $(OPENUSP_AGENTS))
.PHONY: $(addprefix run-,$(OPENUSP_SERVICES) $(OPENUSP_AGENTS))
.PHONY: $(addprefix stop-,$(OPENUSP_SERVICES) $(OPENUSP_AGENTS))
.PHONY: swagger swagger-install swagger-generate swagger-validate
.PHONY: docker-health docker-fix
.PHONY: clean fmt vet test
.PHONY: bash-completion bash-completion-install

.DEFAULT_GOAL := help

# =============================================================================
# Help and Information
# =============================================================================

# Default help target
.DEFAULT_GOAL := help

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
	@echo "OpenUSP Service Endpoints (Static Port Configuration)"
	@echo "==================================================="
	@echo ""
	@echo "🌐 OpenUSP Services:"
	@echo "  API Gateway:     http://localhost:6500 (Health: 6501)"
	@echo "  Swagger UI:      http://localhost:6500/swagger/index.html"
	@echo "  Data Service:    http://localhost:6100 (Health: 6101)"
	@echo "  Connection Mgr:  http://localhost:6200 (Health: 6201)"
	@echo "  USP Service:     http://localhost:6400 (Health: 6401)"
	@echo "  CWMP Service:    http://localhost:7547 (Health: 7548)"
	@echo "  MTP Service:     http://localhost:8081 (WebSocket), 8082 (Health)"
	@echo ""
	@echo "🏛️  Infrastructure:"
	@echo "  Grafana:         http://localhost:3000 (admin/openusp123)"
	@echo "  Prometheus:      http://localhost:9090"
	@echo "  PostgreSQL:      localhost:5433 (openusp/openusp123)"
	@echo "  RabbitMQ:        http://localhost:15672 (openusp/openusp123)"
	@echo ""
	@echo "🎯 All ports are static - no service discovery needed"
	@echo ""
	@echo "🔗 gRPC Service Communication:"
	@echo "  Service-to-service gRPC uses static ports (see docs/GRPC_SERVICES.md)"
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
# Quick Start Operations
# =============================================================================

# Quick Start Operation
all: build-agents run

# =============================================================================
# Cross-Platform Infrastructure Management
# =============================================================================
# Infrastructure Management (Prometheus, Grafana, Mosquitto, RabbitMQ, PostgreSQL)
# =============================================================================

infra-up: infra-volumes
	@echo "🏗️  Starting infrastructure services..."
	@echo "   📊 Prometheus,  Grafana, 🦟 Mosquitto, 🐰 RabbitMQ, 🐘 PostgreSQL"
	@echo "   🎯 Static port configuration - no service discovery needed"
	@docker compose -f $(DOCKER_COMPOSE_INFRA) up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo "✅ Infrastructure services started"
	@$(MAKE) infra-status

infra-down:
	@echo "🛑 Stopping infrastructure services..."
	@docker compose -f $(DOCKER_COMPOSE_INFRA) down
	@echo "✅ Infrastructure services stopped"

infra-status:
	@echo "📊 Infrastructure Services Status:"
	@echo "=================================="
	@docker compose -f $(DOCKER_COMPOSE_INFRA) ps
	@echo ""
	@echo "🌐 Network Information:"
	@echo "Network: openusp-dev"
	@echo "Service Resolution: Container service names"
	@docker network inspect openusp-dev --format '{{range .Containers}}{{.Name}}: {{.IPv4Address}}{{"\n"}}{{end}}' 2>/dev/null || echo "Network not found"

infra-clean:
	@echo "🧹 Cleaning infrastructure (this will remove all data!)..."
	@read -p "Are you sure? This will delete all volumes and data. (y/N): " confirm && \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		docker compose -f $(DOCKER_COMPOSE_INFRA) down -v; \
		docker volume rm $(INFRA_VOLUMES) 2>/dev/null || true; \
		echo "✅ Infrastructure cleaned"; \
	else \
		echo "❌ Cancelled"; \
	fi

# =============================================================================
# OpenUSP Services Management (API Gateway, Data Service, Connection Manager, USP Service, CWMP Service, MTP Service)
# =============================================================================

# Build targets
build: build-services build-agents

build-services: $(addprefix build-,$(OPENUSP_SERVICES))
	@echo "✅ All OpenUSP services built successfully"

build-agents: $(addprefix build-,$(OPENUSP_AGENTS))
	@echo "✅ All OpenUSP agents built successfully"

build-all: build-services build-agents
	@echo "✅ All OpenUSP components built successfully"

# Run targets (depends on infrastructure)
run: run-services

run-services: build-services infra-up
	@echo "🚀 Starting OpenUSP services..."
	@echo "   🌐 API Gateway, 🗄️  Data Service, 🔗 Connection Manager"
	@echo "   📡 USP Service, 📞 CWMP Service, 🚀 MTP Service"
	@$(MAKE) run-connection-manager-background
	@$(MAKE) run-data-service-background
	@$(MAKE) run-api-gateway-background
	@$(MAKE) run-usp-service-background
	@$(MAKE) run-cwmp-service-background
	@$(MAKE) run-mtp-service-background
	@echo "✅ All OpenUSP services started"

run-agents: build-agents
	@echo "🤖 OpenUSP Agents (Console Applications):"
	@echo "   📋 Available agents:"
	@echo "      • make run-usp-agent    - Run USP Agent (interactive)"
	@echo "      • make run-cwmp-agent   - Run CWMP Agent (interactive)"
	@echo ""
	@echo "   ⚠️  Note: Agents are console applications and must be run individually."
	@echo "   ⚠️  They cannot be run in background. Use separate terminals for each agent."

run-all: run-services
	@echo "✅ All OpenUSP services running in background"
	@echo ""
	@echo "🤖 To run agents (console applications):"
	@echo "   make run-usp-agent    # In separate terminal"
	@echo "   make run-cwmp-agent   # In separate terminal"

# Stop targets
stop: stop-services stop-agents

stop-services:
	@echo "🛑 Stopping OpenUSP services..."
	@for service in $(OPENUSP_SERVICES); do \
		$(MAKE) stop-$$service; \
	done
	@echo "✅ OpenUSP services stopped"

stop-agents:
	@echo "🛑 Stopping OpenUSP agents..."
	@pkill -f "usp-agent" 2>/dev/null || true
	@pkill -f "cwmp-agent" 2>/dev/null || true
	@echo "✅ OpenUSP agents stopped (if any were running)"

stop-all: stop-services stop-agents
	@echo "✅ All OpenUSP components stopped"

force-stop:
	@echo "🛑 Force stopping all OpenUSP processes..."
	@echo "⚠️  This will aggressively terminate all matching processes"
	@for service in $(OPENUSP_SERVICES); do \
		echo "Force stopping $$service..."; \
		pkill -9 -f "$$service" 2>/dev/null && echo "  Killed processes matching $$service" || echo "  No processes found for $$service"; \
	done
	@for agent in $(OPENUSP_AGENTS); do \
		echo "Force stopping $$agent..."; \
		pkill -9 -f "$$agent" 2>/dev/null && echo "  Killed processes matching $$agent" || echo "  No processes found for $$agent"; \
	done
	@echo "🧹 Cleaning up PID files..."
	@rm -f logs/*.pid 2>/dev/null || true
	@echo "✅ Force stop completed"

stop-verify:
	@echo "🔍 Verifying all OpenUSP services are stopped..."
	@still_running=false; \
	for service in $(OPENUSP_SERVICES); do \
		printf "%-20s: " "$$service"; \
		pid=$$(pgrep -f "./$(BUILD_DIR)/$$service$$" | head -1); \
		if [ -z "$$pid" ]; then \
			pid=$$(pgrep -f "$(BUILD_DIR)/$$service$$" | head -1); \
		fi; \
		if [ -z "$$pid" ]; then \
			pid=$$(pgrep -f "/$$service$$" | head -1); \
		fi; \
		if [ -n "$$pid" ]; then \
			echo "⚠️  Still running (PID: $$pid)"; \
			still_running=true; \
		else \
			echo "✅ Stopped"; \
		fi; \
	done; \
	if [ "$$still_running" = "true" ]; then \
		echo ""; \
		echo "❌ Some services are still running. Try:"; \
		echo "   make force-stop    # Aggressive termination"; \
		echo "   make status-debug  # Detailed process info"; \
	else \
		echo ""; \
		echo "✅ All OpenUSP services are stopped"; \
	fi

# Helper function for finding service processes (avoids matching tail, grep, etc.)
# Usage: $(call FIND_SERVICE_PID,service-name)
define FIND_SERVICE_PID
$(shell pid=$$(pgrep -f "./$(BUILD_DIR)/$(1)$$" | head -1); \
if [ -z "$$pid" ]; then \
	pid=$$(pgrep -f "$(BUILD_DIR)/$(1)$$" | head -1); \
fi; \
if [ -z "$$pid" ]; then \
	pid=$$(pgrep -f "/$(1)$$" | head -1); \
fi; \
echo "$$pid")
endef

# Status targets
status-services:
	@echo "📊 OpenUSP Services Status (Process Level):"
	@echo "==========================================="
	@for service in $(OPENUSP_SERVICES); do \
		printf "%-20s: " "$$service"; \
		if [ -f logs/$$service.pid ]; then \
			pid=$$(cat logs/$$service.pid); \
			if ps -p $$pid > /dev/null 2>&1; then \
				uptime=$$(ps -o etime= -p $$pid | tr -d ' '); \
				printf "✅ Running (PID: $$pid, Uptime: $$uptime)"; \
				if [ -f logs/$$service.log ]; then \
					logsize=$$(wc -l < logs/$$service.log); \
					printf " [Log: $$logsize lines]"; \
				fi; \
				echo ""; \
			else \
				echo "❌ Dead (stale PID file - cleaned)"; \
				rm -f logs/$$service.pid; \
			fi; \
		else \
			pid=$$(pgrep -f "./$(BUILD_DIR)/$$service$$" | head -1); \
			if [ -z "$$pid" ]; then \
				pid=$$(pgrep -f "$(BUILD_DIR)/$$service$$" | head -1); \
			fi; \
			if [ -z "$$pid" ]; then \
				pid=$$(pgrep -f "/$$service$$" | head -1); \
			fi; \
			if [ -n "$$pid" ]; then \
				uptime=$$(ps -o etime= -p $$pid | tr -d ' '); \
				echo "⚠️  Running (PID: $$pid, Uptime: $$uptime) [No PID file]"; \
			else \
				echo "⭕ Stopped"; \
			fi; \
		fi; \
	done

status-debug:
	@echo "🔍 OpenUSP Status Debug Information:"
	@echo "===================================="
	@echo ""
	@echo "📁 PID Files in logs/:"
	@ls -la logs/*.pid 2>/dev/null || echo "No PID files found"
	@echo ""
	@echo "🔍 Process detection for each service:"
	@for service in $(OPENUSP_SERVICES); do \
		echo "--- $$service ---"; \
		echo "All processes containing service name:"; \
		ps aux | grep -E "$$service" | grep -v grep | head -3 || echo "  No processes found"; \
		echo "Specific patterns used by status system:"; \
		pid1=$$(pgrep -f "./$(BUILD_DIR)/$$service$$" | head -1); \
		pid2=$$(pgrep -f "$(BUILD_DIR)/$$service$$" | head -1); \
		pid3=$$(pgrep -f "/$$service$$" | head -1); \
		[ -n "$$pid1" ] && echo "  ./build/$$service$$: PID $$pid1" || echo "  ./build/$$service$$: Not found"; \
		[ -n "$$pid2" ] && echo "  build/$$service$$: PID $$pid2" || echo "  build/$$service$$: Not found"; \
		[ -n "$$pid3" ] && echo "  /$$service$$: PID $$pid3" || echo "  /$$service$$: Not found"; \
		echo ""; \
	done
	@echo ""
	@echo "🔍 All processes in build directory:"
	@ps aux | grep -E "$(BUILD_DIR)/" | grep -v grep || echo "No build processes found"
	@echo ""
	@echo "📊 OpenUSP Services Status: Using static port configuration"

# Individual service build targets
define BUILD_TEMPLATE
build-$(1):
	@echo "🔨 Building $(1)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(1) ./cmd/$(1)
	@echo "✅ $(1) built successfully"
endef

$(foreach service,$(OPENUSP_SERVICES),$(eval $(call BUILD_TEMPLATE,$(service))))
$(foreach agent,$(OPENUSP_AGENTS),$(eval $(call BUILD_TEMPLATE,$(agent))))

# Individual service run targets (with background support)
define SERVICE_RUN_TEMPLATE
run-$(1):
	@echo "🚀 Starting $(1)..."
	@./$(BUILD_DIR)/$(1)

run-$(1)-background:
	@echo "🚀 Starting $(1) in background..."
	@mkdir -p logs
	@nohup ./$(BUILD_DIR)/$(1) > logs/$(1).log 2>&1 & echo $$! > logs/$(1).pid
	@sleep 2
	@echo "✅ $(1) started (PID: $$(cat logs/$(1).pid))"

stop-$(1):
	@echo "🛑 Stopping $(1)..."
	@stopped=false; \
	if [ -f logs/$(1).pid ]; then \
		pid=$$(cat logs/$(1).pid); \
		if kill $$pid 2>/dev/null; then \
			echo "  Stopped via PID file ($$pid)"; \
			stopped=true; \
		fi; \
		rm -f logs/$(1).pid; \
	fi; \
	if pkill -f "./$(BUILD_DIR)/$(1)" 2>/dev/null; then \
		echo "  Stopped $(1) processes"; \
		stopped=true; \
	fi; \
	if [ "$$stopped" = "false" ]; then \
		echo "  No running $(1) process found"; \
	fi; \
	echo "✅ $(1) stop completed"
endef

# Individual agent run targets (console applications only)
define AGENT_RUN_TEMPLATE
run-$(1): build-$(1)
	@echo "🚀 Starting $(1) (console application)..."
	@echo "   Config: configs/$(1).yaml"
	@echo "   Press Ctrl+C to stop"
	@./$(BUILD_DIR)/$(1) --config configs/$(1).yaml
endef

$(foreach service,$(OPENUSP_SERVICES),$(eval $(call SERVICE_RUN_TEMPLATE,$(service))))
$(foreach agent,$(OPENUSP_AGENTS),$(eval $(call AGENT_RUN_TEMPLATE,$(agent))))

# =============================================================================
# Swagger API Documentation
# =============================================================================

swagger: swagger-generate swagger-validate

swagger-install:
	@echo "🔧 Installing Swagger tools..."
	@echo "📍 Go binary path: $$(go env GOPATH)/bin"
	@echo "📍 Current PATH: $$PATH"
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install github.com/go-swagger/go-swagger/cmd/swagger@latest
	@echo "✅ Swagger tools installed"
	@echo ""
	@echo "💡 If you get 'command not found' errors, add Go bin to PATH:"
	@echo "   export PATH=\$$PATH:\$$(go env GOPATH)/bin"
	@echo "   echo 'export PATH=\$$PATH:\$$(go env GOPATH)/bin' >> ~/.bashrc"
	@echo "   source ~/.bashrc"

swagger-generate:
	@echo "📚 Generating Swagger documentation..."
	@command -v swag >/dev/null 2>&1 || { \
		echo "⚠️  Installing swag..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		echo "✅ swag installed to $$(go env GOPATH)/bin/swag"; \
	}
	@SWAG_CMD=$$(command -v swag 2>/dev/null || echo "$$(go env GOPATH)/bin/swag"); \
	if [ ! -f "$$SWAG_CMD" ]; then \
		echo "❌ swag not found at $$SWAG_CMD"; \
		echo "💡 Please ensure $$(go env GOPATH)/bin is in your PATH"; \
		echo "💡 Or run: export PATH=\$$PATH:\$$(go env GOPATH)/bin"; \
		exit 1; \
	fi; \
	$$SWAG_CMD init -g cmd/api-gateway/main.go -o api/
	@echo "✅ Swagger documentation generated"

swagger-validate:
	@echo "✅ Validating Swagger documentation..."
	@command -v swagger >/dev/null 2>&1 || { \
		echo "⚠️  Installing swagger validator..."; \
		go install github.com/go-swagger/go-swagger/cmd/swagger@latest; \
		echo "✅ swagger validator installed to $$(go env GOPATH)/bin/swagger"; \
	}
	@SWAGGER_CMD=$$(command -v swagger 2>/dev/null || echo "$$(go env GOPATH)/bin/swagger"); \
	if [ ! -f "$$SWAGGER_CMD" ]; then \
		echo "❌ swagger validator not found at $$SWAGGER_CMD"; \
		echo "💡 Please ensure $$(go env GOPATH)/bin is in your PATH"; \
		echo "💡 Or run: export PATH=\$$PATH:\$$(go env GOPATH)/bin"; \
		exit 1; \
	fi; \
	$$SWAGGER_CMD validate api/swagger.yaml
	@echo "✅ Swagger documentation is valid"

infra-volumes:
	@echo "📦 Creating infrastructure volumes..."
	@$(foreach vol,$(INFRA_VOLUMES),docker volume create $(vol) >/dev/null 2>&1 || true;)
	@echo "✅ Infrastructure volumes ready"
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
	@if curl -s -f http://localhost:3000/api/health > /dev/null; then \
		echo "✅ Grafana is accessible at http://localhost:3000"; \
	else \
		echo "❌ Grafana is not accessible. Make sure infrastructure is running (make infra-up)"; \
		exit 1; \
	fi

service-status:
	@echo "� OpenUSP Service Status (Static Port Configuration):"
	@echo "===================================================="
	@printf "%-20s: " "api-gateway"; \
	if curl -s http://localhost:6500/health >/dev/null 2>&1; then \
		echo "✅ Accessible (http://localhost:6500)"; \
	else \
		echo "❌ Not accessible"; \
	fi
	@printf "%-20s: " "data-service"; \
	if curl -s http://localhost:6100/health >/dev/null 2>&1; then \
		echo "✅ Accessible (http://localhost:6100)"; \
	else \
		echo "❌ Not accessible"; \
	fi
	@printf "%-20s: " "connection-manager"; \
	if curl -s http://localhost:6200/health >/dev/null 2>&1; then \
		echo "✅ Accessible (http://localhost:6200)"; \
	else \
		echo "❌ Not accessible"; \
	fi
	@printf "%-20s: " "usp-service"; \
	if curl -s http://localhost:6400/health >/dev/null 2>&1; then \
		echo "✅ Accessible (http://localhost:6400)"; \
	else \
		echo "❌ Not accessible"; \
	fi
	@printf "%-20s: " "cwmp-service"; \
	if curl -s http://localhost:7547/health >/dev/null 2>&1; then \
		echo "✅ Accessible (http://localhost:7547)"; \
	else \
		echo "❌ Not accessible"; \
	fi
	@printf "%-20s: " "mtp-service"; \
	if curl -s http://localhost:8082/health >/dev/null 2>&1; then \
		echo "✅ Accessible (http://localhost:8082)"; \
	else \
		echo "❌ Not accessible"; \
	fi
	@echo ""
	@echo "🔗 gRPC Port Connectivity:"
	@printf "%-20s: " "api-gateway-grpc"; \
	echo "ℹ️  HTTP only (no gRPC)"
	@printf "%-20s: " "data-service-grpc"; \
	if timeout 2 bash -c "</dev/tcp/localhost/6101" 2>/dev/null; then \
		echo "✅ Port 6101 open"; \
	else \
		echo "❌ Port 6101 closed"; \
	fi
	@printf "%-20s: " "connection-mgr-grpc"; \
	if timeout 2 bash -c "</dev/tcp/localhost/6201" 2>/dev/null; then \
		echo "✅ Port 6201 open"; \
	else \
		echo "❌ Port 6201 closed"; \
	fi
	@printf "%-20s: " "usp-service-grpc"; \
	if timeout 2 bash -c "</dev/tcp/localhost/6401" 2>/dev/null; then \
		echo "✅ Port 6401 open"; \
	else \
		echo "❌ Port 6401 closed"; \
	fi
	@printf "%-20s: " "cwmp-service-grpc"; \
	if timeout 2 bash -c "</dev/tcp/localhost/7548" 2>/dev/null; then \
		echo "✅ Port 7548 open"; \
	else \
		echo "❌ Port 7548 closed"; \
	fi
	@printf "%-20s: " "mtp-service-grpc"; \
	if timeout 2 bash -c "</dev/tcp/localhost/8083" 2>/dev/null; then \
		echo "✅ Port 8083 open"; \
	else \
		echo "❌ Port 8083 closed"; \
	fi

# =============================================================================
# Development Environment Management
# =============================================================================

# Static port configuration - no service discovery dependencies

dev-reset:
	@echo "🔄 Resetting development environment..."
	@echo "1. Stopping all services..."
	@$(MAKE) stop-all || true
	@echo "2. Building all services..."
	@$(MAKE) build-all
	@echo "3. Starting services..."
	@$(MAKE) run-services
	@echo "✅ Development environment reset complete"

dev-restart:
	@echo "🔄 Restarting OpenUSP services (preserving infrastructure)..."
	@echo "1. Stop services..."
	@$(MAKE) stop-all || true
	@sleep 2
	@echo "2. Start services..."
	@$(MAKE) run-services
	@echo "✅ Services restarted"



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
# Development Environment Management
# =============================================================================

.PHONY: docker-health
docker-health:
	@./scripts/docker-health.sh check

.PHONY: docker-fix
docker-fix:
	@./scripts/docker-health.sh fix

# =============================================================================
# Convenient Aliases
# =============================================================================

.PHONY: status

status: 
	@echo "🚀 OpenUSP Platform Status"
	@echo "=========================="
	@echo ""
	@echo "📊 Quick Summary:"
	@echo "=================" 
	@printf "Infrastructure: "; \
	if docker compose -f $(DOCKER_COMPOSE_INFRA) ps -q | wc -l | grep -q '^[1-9]'; then \
		echo "✅ Running"; \
	else \
		echo "❌ Stopped"; \
	fi
	@printf "OpenUSP Services: "; \
	running=0; total=0; \
	for service in $(OPENUSP_SERVICES); do \
		total=$$((total + 1)); \
		if [ -f logs/$$service.pid ] && ps -p $$(cat logs/$$service.pid) >/dev/null 2>&1; then \
			running=$$((running + 1)); \
		else \
			pid=$$(pgrep -f "./$(BUILD_DIR)/$$service$$" | head -1); \
			if [ -z "$$pid" ]; then \
				pid=$$(pgrep -f "$(BUILD_DIR)/$$service$$" | head -1); \
			fi; \
			if [ -z "$$pid" ]; then \
				pid=$$(pgrep -f "/$$service$$" | head -1); \
			fi; \
			if [ -n "$$pid" ]; then \
				running=$$((running + 1)); \
			fi; \
		fi; \
	done; \
	if [ $$running -eq $$total ]; then \
		echo "✅ All running ($$running/$$total)"; \
	elif [ $$running -gt 0 ]; then \
		echo "⚠️  Partial ($$running/$$total running)"; \
	else \
		echo "❌ None running (0/$$total)"; \
	fi
	@echo ""
	@$(MAKE) infra-status
	@echo ""
	@$(MAKE) status-services
	@echo ""
	@$(MAKE) service-status
	@echo ""
	@echo "🌐 Network Connectivity:"
	@echo "========================"
	@printf "%-20s: " "API Gateway"; \
	if curl -s http://localhost:6500/health >/dev/null 2>&1; then \
		echo "✅ Accessible (http://localhost:6500)"; \
	else \
		echo "❌ Not accessible"; \
	fi
	@printf "%-20s: " "MTP WebSocket"; \
	if timeout 2 bash -c "</dev/tcp/localhost/8081" 2>/dev/null; then \
		echo "✅ Accessible (ws://localhost:8081)"; \
	else \
		echo "❌ Not accessible"; \
	fi
	@echo ""
	@echo "📊 Status Validation:"
	@echo "===================="
	@process_count=0; accessible_count=0; \
	for service in $(OPENUSP_SERVICES); do \
		if [ -f logs/$$service.pid ] && ps -p $$(cat logs/$$service.pid) >/dev/null 2>&1; then \
			process_count=$$((process_count + 1)); \
		else \
			pid=$$(pgrep -f "./$(BUILD_DIR)/$$service$$" | head -1); \
			if [ -z "$$pid" ]; then pid=$$(pgrep -f "$(BUILD_DIR)/$$service$$" | head -1); fi; \
			if [ -z "$$pid" ]; then pid=$$(pgrep -f "/$$service$$" | head -1); fi; \
			if [ -n "$$pid" ]; then process_count=$$((process_count + 1)); fi; \
		fi; \
		if [ "$$service" = "api-gateway" ] && curl -s http://localhost:6500/health >/dev/null 2>&1; then \
			accessible_count=$$((accessible_count + 1)); \
		elif [ "$$service" = "data-service" ] && curl -s http://localhost:6100/health >/dev/null 2>&1; then \
			accessible_count=$$((accessible_count + 1)); \
		elif [ "$$service" = "connection-manager" ] && curl -s http://localhost:6200/health >/dev/null 2>&1; then \
			accessible_count=$$((accessible_count + 1)); \
		elif [ "$$service" = "usp-service" ] && curl -s http://localhost:6400/health >/dev/null 2>&1; then \
			accessible_count=$$((accessible_count + 1)); \
		elif [ "$$service" = "cwmp-service" ] && curl -s http://localhost:7547/health >/dev/null 2>&1; then \
			accessible_count=$$((accessible_count + 1)); \
		elif [ "$$service" = "mtp-service" ] && curl -s http://localhost:8081/health >/dev/null 2>&1; then \
			accessible_count=$$((accessible_count + 1)); \
		fi; \
	done; \
	if [ $$process_count -ne $$accessible_count ]; then \
		echo "⚠️  Status Mismatch: $$process_count processes running, $$accessible_count accessible"; \
		echo "   Run: make status-debug"; \
	else \
		echo "✅ Status Consistent: $$process_count services running and accessible"; \
	fi
	@echo ""
	@echo "�📋 Quick Commands:"
	@echo "  make run-services      - Start all services"
	@echo "  make stop-services     - Stop all services"
	@echo "  make status-quick      - Brief status overview"
	@echo "  make status-services   - Check service processes"
	@echo "  make service-status    - Check service accessibility"
	@echo "  make status-debug      - Debug status mismatches"

status-quick:
	@echo "🚀 OpenUSP Quick Status"
	@echo "======================"
	@printf "Infrastructure: "; \
	if docker compose -f $(DOCKER_COMPOSE_INFRA) ps -q | wc -l | grep -q '^[1-9]'; then \
		echo "✅ Running"; \
	else \
		echo "❌ Stopped (run: make infra-up)"; \
	fi
	@printf "Services: "; \
	running=0; total=0; \
	for service in $(OPENUSP_SERVICES); do \
		total=$$((total + 1)); \
		if [ -f logs/$$service.pid ] && ps -p $$(cat logs/$$service.pid) >/dev/null 2>&1; then \
			running=$$((running + 1)); \
		else \
			pid=$$(pgrep -f "./$(BUILD_DIR)/$$service$$" | head -1); \
			if [ -z "$$pid" ]; then \
				pid=$$(pgrep -f "$(BUILD_DIR)/$$service$$" | head -1); \
			fi; \
			if [ -z "$$pid" ]; then \
				pid=$$(pgrep -f "/$$service$$" | head -1); \
			fi; \
			if [ -n "$$pid" ]; then \
				running=$$((running + 1)); \
			fi; \
		fi; \
	done; \
	if [ $$running -eq $$total ]; then \
		echo "✅ All running ($$running/$$total)"; \
	elif [ $$running -gt 0 ]; then \
		echo "⚠️  Partial ($$running/$$total running - run: make status-services)"; \
	else \
		echo "❌ None running (run: make run-services)"; \
	fi
	@printf "API Gateway: "; \
	if curl -s http://localhost:6500/health >/dev/null 2>&1; then \
		echo "✅ http://localhost:6500"; \
	else \
		echo "❌ Not accessible"; \
	fi
	@printf "MTP WebSocket: "; \
	if timeout 2 bash -c "</dev/tcp/localhost/8081" 2>/dev/null; then \
		echo "✅ ws://localhost:8081"; \
	else \
		echo "❌ Not accessible"; \
	fi

# =============================================================================
# Enhanced Help System
# =============================================================================

.PHONY: help
help:
	@echo "🚀 OpenUSP - TR-369 User Service Platform"
	@echo "=========================================="
	@echo ""
	@echo "📦 Infrastructure Services (Prometheus, Grafana, etc.):"
	@echo "  infra-up               - Start all infrastructure services"
	@echo "  infra-down             - Stop all infrastructure services"
	@echo "  infra-status           - Show infrastructure status"
	@echo "  infra-clean            - Clean infrastructure (removes all data)"
	@echo ""
	@echo "🚀 OpenUSP Services (API Gateway, Data Service, etc.):"
	@echo "  build                  - Build all OpenUSP services"
	@echo "  build-services         - Build OpenUSP services only"
	@echo "  run                    - Run all OpenUSP services"
	@echo "  run-services           - Run OpenUSP services only"
	@echo "  stop                   - Stop all OpenUSP services"
	@echo "  force-stop             - Force stop all OpenUSP processes"
	@echo "  stop-verify            - Verify all services stopped"
	@echo "  status                 - Show comprehensive platform status"
	@echo "  status-services        - Show status of OpenUSP services"
	@echo ""
	@echo "🤖 OpenUSP Agents (Console Applications):"
	@echo "  build-agents           - Build OpenUSP agents (usp-agent, cwmp-agent)"
	@echo "  run-agents             - Show how to run individual agents"
	@echo "  run-usp-agent          - Run USP Agent (interactive console)"
	@echo "  run-cwmp-agent         - Run CWMP Agent (interactive console)"
	@echo ""
	@echo "📚 API Documentation:"
	@echo "  swagger                - Generate and validate Swagger docs"
	@echo "  swagger-install        - Install Swagger tools (swag & swagger)"
	@echo "  swagger-generate       - Generate Swagger documentation"
	@echo "  swagger-validate       - Validate Swagger documentation"
	@echo ""
	@echo "📊 Monitoring & Observability:"
	@echo "  monitoring-cleanup     - Clean up and reload monitoring stack"
	@echo "  prometheus-reload      - Reload Prometheus configuration"
	@echo "  grafana-restart        - Restart Grafana container"
	@echo ""
	@echo "🛠️  Development Tools:"
	@echo "  status-quick           - Show brief status overview"
	@echo "  status-debug           - Debug status detection issues"
	@echo "  service-status         - Check service accessibility"
	@echo "  clean                  - Clean build artifacts"
	@echo "  fmt                    - Format Go code"
	@echo "  vet                    - Run Go vet"
	@echo "  test                   - Run tests"

	@echo "  docker-health          - Check Docker health"
	@echo ""
	@echo "⚡ Shell Integration:"
	@echo "  bash-completion        - Generate bash completion script for make targets"
	@echo "  bash-completion-install- Install bash completion system-wide"
	@echo ""
	@echo "📖 Documentation:"
	@echo "   - docs/README.md       - Main documentation"
	@echo "   - docs/NETWORKING.md   - Network architecture"
	@echo "   - docs/QUICKSTART.md   - Quick start guide"

# =============================================================================
# Monitoring and Observability Commands
# =============================================================================

monitoring-cleanup: prometheus-reload grafana-restart
	@echo "✅ Monitoring stack cleaned up"

prometheus-reload:
	@echo "🔄 Reloading Prometheus configuration..."
	@curl -X POST http://localhost:9090/-/reload 2>/dev/null || echo "⚠️  Prometheus not accessible"

grafana-restart:
	@echo "🔄 Restarting Grafana..."
	@docker restart openusp-grafana-dev

# =============================================================================
# Bash Completion
# =============================================================================

bash-completion: ## Generate bash completion script for all make targets
	@echo "#!/usr/bin/env bash" > .openusp-completion.bash
	@echo "# OpenUSP Makefile bash completion script" >> .openusp-completion.bash
	@echo "# Source this file to enable bash completion for make targets" >> .openusp-completion.bash
	@echo "# Usage: source .openusp-completion.bash" >> .openusp-completion.bash
	@echo "" >> .openusp-completion.bash
	@echo "_openusp_make_completion() {" >> .openusp-completion.bash
	@echo "    local cur prev opts" >> .openusp-completion.bash
	@echo "    COMPREPLY=()" >> .openusp-completion.bash
	@echo "    cur=\"\$${COMP_WORDS[COMP_CWORD]}\"" >> .openusp-completion.bash
	@echo "    prev=\"\$${COMP_WORDS[COMP_CWORD-1]}\"" >> .openusp-completion.bash
	@echo "" >> .openusp-completion.bash
	@echo "    # Make targets extracted from Makefile" >> .openusp-completion.bash
	@printf "    opts=\"" >> .openusp-completion.bash
	@$(MAKE) -qp 2>/dev/null | awk -F':' '/^[a-zA-Z0-9][^$$#\/\t=]*:([^=]|$$)/ {split($$1,A,/ /);for(i in A)print A[i]}' | grep -v '^Makefile$$' | sort -u | tr '\n' ' ' >> .openusp-completion.bash
	@echo "\"" >> .openusp-completion.bash
	@echo "" >> .openusp-completion.bash
	@echo "    COMPREPLY=( \$$(compgen -W \"\$$opts\" -- \$$cur) )" >> .openusp-completion.bash
	@echo "    return 0" >> .openusp-completion.bash
	@echo "}" >> .openusp-completion.bash
	@echo "" >> .openusp-completion.bash
	@echo "# Register completion for 'make' command in this directory" >> .openusp-completion.bash
	@echo "complete -F _openusp_make_completion make" >> .openusp-completion.bash
	@echo "" >> .openusp-completion.bash
	@echo "echo \"✅ OpenUSP make bash completion loaded! Try: make <TAB><TAB>\"" >> .openusp-completion.bash
	@echo "📝 Generated bash completion script: .openusp-completion.bash"
	@echo "💡 To enable completion, run: source .openusp-completion.bash"

bash-completion-install: bash-completion ## Generate and install bash completion system-wide
	@echo "🔧 Installing bash completion for OpenUSP make targets..."
	@if [ -d "/usr/local/etc/bash_completion.d" ]; then \
		sudo cp .openusp-completion.bash /usr/local/etc/bash_completion.d/openusp-make; \
		echo "✅ Installed to /usr/local/etc/bash_completion.d/openusp-make"; \
	elif [ -d "/etc/bash_completion.d" ]; then \
		sudo cp .openusp-completion.bash /etc/bash_completion.d/openusp-make; \
		echo "✅ Installed to /etc/bash_completion.d/openusp-make"; \
	else \
		echo "⚠️  No system bash completion directory found"; \
		echo "💡 Manual setup: Add 'source $(PWD)/.openusp-completion.bash' to your ~/.bashrc"; \
	fi
	@echo "🎯 Bash completion installed! Restart your shell or run: source ~/.bashrc"
	@echo "💫 Test with: make <TAB><TAB>"

# =============================================================================
# End of Makefile
# =============================================================================
