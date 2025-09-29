#!/bin/bash

# OpenUSP Docker Compose Deployment Script
# This script manages Docker Compose deployments for testing and production

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DEPLOY_DIR="$(dirname "$0")"
ENV_DIR="${DEPLOY_DIR}/environments"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if Docker and Docker Compose are available
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running"
        exit 1
    fi
    
    print_success "Docker and Docker Compose are available"
}

# Function to build Docker images
build_images() {
    print_status "Building OpenUSP Docker images..."
    
    cd "$DEPLOY_DIR/.."
    
    # Build all services
    docker-compose -f docker-compose.yml build --parallel
    
    print_success "Docker images built successfully"
}

# Function to start testing environment
start_test() {
    print_status "Starting OpenUSP testing environment..."
    
    # Check if .env.test exists
    if [[ ! -f "${ENV_DIR}/.env.test" ]]; then
        print_error ".env.test file not found in ${ENV_DIR}"
        print_error "Please create .env.test based on .env.test.template"
        exit 1
    fi
    
    cd "$ENV_DIR"
    
    # Start services
    docker-compose -f docker-compose.test.yml --env-file .env.test up -d
    
    print_status "Waiting for services to be ready..."
    sleep 10
    
    # Check service health
    check_service_health "test"
    
    print_success "Testing environment started successfully"
    show_test_endpoints
}

# Function to start production environment
start_prod() {
    print_status "Starting OpenUSP production environment..."
    
    # Check if .env.prod exists
    if [[ ! -f "${ENV_DIR}/.env.prod" ]]; then
        print_error ".env.prod file not found in ${ENV_DIR}"
        print_error "Please create .env.prod based on .env.prod.template"
        exit 1
    fi
    
    cd "$ENV_DIR"
    
    # Start services
    docker-compose -f docker-compose.prod.yml --env-file .env.prod up -d
    
    print_status "Waiting for services to be ready..."
    sleep 15
    
    # Check service health
    check_service_health "prod"
    
    print_success "Production environment started successfully"
    show_prod_endpoints
}

# Function to check service health
check_service_health() {
    local env_type="$1"
    local base_port=""
    
    if [[ "$env_type" == "test" ]]; then
        base_port="18080"  # Test API Gateway port
    else
        base_port="8080"   # Production API Gateway port
    fi
    
    print_status "Checking service health..."
    
    # Wait for API Gateway
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s "http://localhost:${base_port}/health" > /dev/null 2>&1; then
            print_success "API Gateway is healthy"
            break
        fi
        
        if [[ $attempt -eq $max_attempts ]]; then
            print_warning "API Gateway health check timeout"
            break
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    echo
}

# Function to show testing environment endpoints
show_test_endpoints() {
    echo
    print_status "Testing Environment Endpoints:"
    echo "  API Gateway:     http://localhost:18080"
    echo "    Health:        http://localhost:18080/health"
    echo "    Status:        http://localhost:18080/status"
    echo "    Devices API:   http://localhost:18080/api/v1/devices"
    echo
    echo "  MTP Service:     http://localhost:18081"
    echo "    Demo UI:       http://localhost:18081/usp"
    echo "    Health:        http://localhost:18081/health"
    echo "    WebSocket:     ws://localhost:18081/ws"
    echo
    echo "  CWMP Service:    http://localhost:17547"
    echo "    Health:        http://localhost:17547/health"
    echo "    Auth:          acs/acs123"
    echo
    echo "  Data Service:    localhost:19092 (gRPC)"
    echo "  USP Service:     localhost:19093 (gRPC)"
    echo
    echo "  PostgreSQL:      localhost:15432"
    echo "  RabbitMQ:        http://localhost:15672 (admin/admin)"
    echo "  Mosquitto MQTT:  localhost:11883"
    echo
    echo "  Prometheus:      http://localhost:19090"
    echo "  Grafana:         http://localhost:13000 (admin/admin)"
    echo
}

# Function to show production environment endpoints
show_prod_endpoints() {
    echo
    print_status "Production Environment Endpoints:"
    echo "  API Gateway:     http://localhost:8080"
    echo "    Health:        http://localhost:8080/health"
    echo "    Status:        http://localhost:8080/status"
    echo "    Devices API:   http://localhost:8080/api/v1/devices"
    echo
    echo "  MTP Service:     http://localhost:8081"
    echo "    Demo UI:       http://localhost:8081/usp"
    echo "    Health:        http://localhost:8081/health"
    echo "    WebSocket:     ws://localhost:8081/ws"
    echo
    echo "  CWMP Service:    http://localhost:7547"
    echo "    Health:        http://localhost:7547/health"
    echo "    Auth:          acs/acs123"
    echo
    echo "  Data Service:    localhost:9092 (gRPC)"
    echo "  USP Service:     localhost:9093 (gRPC)"
    echo
    echo "  PostgreSQL:      localhost:5432"
    echo "  RabbitMQ:        http://localhost:15672 (admin/password)"
    echo "  Mosquitto MQTT:  localhost:1883"
    echo
    echo "  Prometheus:      http://localhost:9090"
    echo "  Grafana:         http://localhost:3000 (admin/admin)"
    echo
}

# Function to show status
show_status() {
    print_status "Docker Compose Status:"
    echo
    
    # Check test environment
    if [[ -f "${ENV_DIR}/docker-compose.test.yml" ]]; then
        echo "Testing Environment:"
        cd "$ENV_DIR"
        docker-compose -f docker-compose.test.yml ps 2>/dev/null || echo "  Not running"
        echo
    fi
    
    # Check production environment
    if [[ -f "${ENV_DIR}/docker-compose.prod.yml" ]]; then
        echo "Production Environment:"
        cd "$ENV_DIR"
        docker-compose -f docker-compose.prod.yml ps 2>/dev/null || echo "  Not running"
        echo
    fi
}

# Function to stop services
stop_services() {
    local env_type="$1"
    
    cd "$ENV_DIR"
    
    if [[ "$env_type" == "test" ]]; then
        print_status "Stopping testing environment..."
        docker-compose -f docker-compose.test.yml down
        print_success "Testing environment stopped"
    elif [[ "$env_type" == "prod" ]]; then
        print_status "Stopping production environment..."
        docker-compose -f docker-compose.prod.yml down
        print_success "Production environment stopped"
    else
        print_status "Stopping all environments..."
        docker-compose -f docker-compose.test.yml down 2>/dev/null || true
        docker-compose -f docker-compose.prod.yml down 2>/dev/null || true
        print_success "All environments stopped"
    fi
}

# Function to clean up (stop and remove volumes)
cleanup() {
    local env_type="$1"
    
    cd "$ENV_DIR"
    
    if [[ "$env_type" == "test" ]]; then
        print_warning "Cleaning up testing environment (including volumes)..."
        docker-compose -f docker-compose.test.yml down -v
        print_success "Testing environment cleaned up"
    elif [[ "$env_type" == "prod" ]]; then
        print_warning "Cleaning up production environment (including volumes)..."
        docker-compose -f docker-compose.prod.yml down -v
        print_success "Production environment cleaned up"
    else
        print_warning "Cleaning up all environments (including volumes)..."
        docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
        docker-compose -f docker-compose.prod.yml down -v 2>/dev/null || true
        print_success "All environments cleaned up"
    fi
}

# Function to show logs
show_logs() {
    local env_type="$1"
    local service="${2:-}"
    
    cd "$ENV_DIR"
    
    if [[ "$env_type" == "test" ]]; then
        if [[ -n "$service" ]]; then
            docker-compose -f docker-compose.test.yml logs -f "$service"
        else
            docker-compose -f docker-compose.test.yml logs -f
        fi
    elif [[ "$env_type" == "prod" ]]; then
        if [[ -n "$service" ]]; then
            docker-compose -f docker-compose.prod.yml logs -f "$service"
        else
            docker-compose -f docker-compose.prod.yml logs -f
        fi
    else
        print_error "Please specify environment: test or prod"
        exit 1
    fi
}

# Function to show help
show_help() {
    echo "OpenUSP Docker Compose Deployment Script"
    echo
    echo "Usage: $0 [COMMAND] [ENVIRONMENT] [SERVICE]"
    echo
    echo "Commands:"
    echo "  build           Build all Docker images"
    echo "  start           Start environment (default: test)"
    echo "  stop            Stop environment"
    echo "  restart         Restart environment"
    echo "  status          Show status of all environments"
    echo "  logs            Show logs for environment"
    echo "  cleanup         Stop and remove volumes"
    echo "  help            Show this help message"
    echo
    echo "Environments:"
    echo "  test            Testing environment (different ports)"
    echo "  prod            Production environment (standard ports)"
    echo
    echo "Examples:"
    echo "  $0 build                    # Build all images"
    echo "  $0 start test              # Start testing environment"
    echo "  $0 start prod              # Start production environment"
    echo "  $0 logs test api-gateway   # Show API Gateway logs (test env)"
    echo "  $0 stop prod               # Stop production environment"
    echo "  $0 cleanup                 # Clean up all environments"
    echo
}

# Main function
main() {
    local command="${1:-start}"
    local environment="${2:-test}"
    local service="${3:-}"
    
    check_docker
    
    case "$command" in
        "build")
            build_images
            ;;
        "start")
            if [[ "$environment" == "test" ]]; then
                start_test
            elif [[ "$environment" == "prod" ]]; then
                start_prod
            else
                print_error "Unknown environment: $environment"
                exit 1
            fi
            ;;
        "stop")
            stop_services "$environment"
            ;;
        "restart")
            stop_services "$environment"
            sleep 2
            if [[ "$environment" == "test" ]]; then
                start_test
            elif [[ "$environment" == "prod" ]]; then
                start_prod
            fi
            ;;
        "status")
            show_status
            ;;
        "logs")
            show_logs "$environment" "$service"
            ;;
        "cleanup")
            cleanup "$environment"
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"