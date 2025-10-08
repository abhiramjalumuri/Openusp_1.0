#!/bin/bash

# Access OpenUSP Services
# Provides easy access to deployed services with port forwarding and service information

set -euo pipefail

# Configuration
CLUSTER_NAME="openusp-local"
NAMESPACE="openusp"
MONITORING_NAMESPACE="monitoring"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_cluster() {
    log_info "Checking cluster status..."
    
    # Check if cluster exists
    if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_error "Kind cluster '${CLUSTER_NAME}' not found"
        log_error "Please run ./setup-cluster.sh first"
        exit 1
    fi
    
    # Set kubectl context
    kubectl config use-context "kind-${CLUSTER_NAME}"
    
    # Check if namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_error "Namespace '$NAMESPACE' not found"
        log_error "Please run ./deploy-openusp.sh first"
        exit 1
    fi
    
    log_success "Cluster and namespace are ready"
}

show_service_status() {
    log_info "Service Status Overview"
    echo ""
    
    # Show pod status
    echo "=== Pods ==="
    kubectl get pods -n "$NAMESPACE" -o wide
    echo ""
    
    # Show service status
    echo "=== Services ==="
    kubectl get svc -n "$NAMESPACE"
    echo ""
    
    # Show ingress status
    echo "=== Ingress ==="
    kubectl get ingress -n "$NAMESPACE" 2>/dev/null || echo "No ingress found"
    echo ""
    
    # Show persistent volumes
    echo "=== Persistent Volumes ==="
    kubectl get pvc -n "$NAMESPACE"
    echo ""
}

show_access_info() {
    log_info "Service Access Information"
    echo ""
    
    echo "🌐 Web Interfaces (NodePort Access):"
    echo "  • API Gateway: http://localhost:8080"
    echo "  • Data Service: http://localhost:8081"
    echo "  • USP Service: http://localhost:8082"
    echo "  • MTP Service: http://localhost:8083"
    echo "  • CWMP Service: http://localhost:7547"
    echo "  • Connection Manager: http://localhost:8084"
    echo ""
    
    echo "🏗️ Infrastructure Services:"
    echo "  • PostgreSQL: localhost:5432 (openusp/openusp123)"
    echo "  • RabbitMQ Management: http://localhost:15672 (guest/guest)"
    echo "  • Mosquitto MQTT: localhost:1883"
    echo "  • Consul UI: http://localhost:8500"
    echo ""
    
    echo "📊 Monitoring Stack:"
    echo "  • Grafana: http://localhost:3000 (admin/admin)"
    echo "  • Prometheus: http://localhost:9090"
    echo ""
    
    echo "🔧 Development Tools:"
    echo "  • Kubernetes Dashboard: kubectl proxy (then visit dashboard URL)"
    echo "  • Port Forward: kubectl port-forward svc/<service> <local-port>:<service-port> -n $NAMESPACE"
    echo ""
}

test_service_health() {
    log_info "Testing service health..."
    
    local services=(
        "api-gateway:8080"
        "data-service:8081"
        "usp-service:8082"
        "mtp-service:8083"
        "cwmp-service:7547"
        "connection-manager:8084"
    )
    
    echo ""
    echo "Health Check Results:"
    
    for service_port in "${services[@]}"; do
        local service="${service_port%:*}"
        local port="${service_port#*:}"
        
        printf "  %-20s " "$service:"
        
        if curl -s --max-time 3 "http://localhost:$port/health" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Healthy${NC}"
        else
            echo -e "${RED}✗ Not responding${NC}"
        fi
    done
    echo ""
}

show_logs() {
    local service="$1"
    local follow="${2:-false}"
    
    log_info "Showing logs for $service..."
    
    if [[ "$follow" == "true" ]]; then
        kubectl logs -f "deployment/openusp-${service}" -n "$NAMESPACE"
    else
        kubectl logs "deployment/openusp-${service}" -n "$NAMESPACE" --tail=50
    fi
}

port_forward_service() {
    local service="$1"
    local local_port="$2"
    local service_port="$3"
    
    log_info "Port forwarding $service from localhost:$local_port to service:$service_port"
    log_info "Press Ctrl+C to stop port forwarding"
    
    kubectl port-forward "svc/openusp-${service}" "$local_port:$service_port" -n "$NAMESPACE"
}

exec_into_pod() {
    local service="$1"
    
    log_info "Executing shell into $service pod..."
    
    kubectl exec -it "deployment/openusp-${service}" -n "$NAMESPACE" -- /bin/sh
}

show_resource_usage() {
    log_info "Resource Usage"
    echo ""
    
    # Show node resource usage
    echo "=== Node Resource Usage ==="
    kubectl top nodes 2>/dev/null || echo "Metrics server not available"
    echo ""
    
    # Show pod resource usage
    echo "=== Pod Resource Usage ==="
    kubectl top pods -n "$NAMESPACE" 2>/dev/null || echo "Metrics server not available"
    echo ""
    
    # Show resource quotas
    echo "=== Resource Quotas ==="
    kubectl describe resourcequota -n "$NAMESPACE" 2>/dev/null || echo "No resource quotas found"
    echo ""
}

open_grafana_dashboards() {
    log_info "Opening Grafana dashboards..."
    
    # Check if Grafana is accessible
    if curl -s --max-time 3 "http://localhost:3000" > /dev/null 2>&1; then
        log_success "Grafana is accessible at http://localhost:3000"
        
        if command -v open &> /dev/null; then
            open "http://localhost:3000"
        elif command -v xdg-open &> /dev/null; then
            xdg-open "http://localhost:3000"
        else
            log_info "Please open http://localhost:3000 in your browser"
        fi
        
        echo ""
        echo "Grafana Login:"
        echo "  Username: admin"
        echo "  Password: admin"
    else
        log_error "Grafana is not accessible. Please check if monitoring stack is deployed."
    fi
}

show_help() {
    cat << EOF
OpenUSP Service Access Tool

USAGE:
    $0 [COMMAND] [OPTIONS]

COMMANDS:
    status              Show service status and pod information
    access              Show service access information
    health              Test service health endpoints
    logs <service>      Show logs for a specific service
    follow <service>    Follow logs for a specific service
    forward <service> <local-port> <service-port>  Port forward a service
    shell <service>     Execute shell into a service pod
    resources           Show resource usage
    grafana             Open Grafana dashboards
    help                Show this help message

EXAMPLES:
    # Show overall status
    $0 status

    # Show service access information
    $0 access

    # Test all service health
    $0 health

    # Show logs for API gateway
    $0 logs api-gateway

    # Follow logs for data service
    $0 follow data-service

    # Port forward USP service to local port 9082
    $0 forward usp-service 9082 8082

    # Execute shell into MTP service pod
    $0 shell mtp-service

    # Show resource usage
    $0 resources

    # Open Grafana
    $0 grafana

SERVICES:
    api-gateway, data-service, usp-service, mtp-service,
    cwmp-service, connection-manager, usp-agent, cwmp-agent

For more information, see the README.md file.
EOF
}

main() {
    case "${1:-status}" in
        "status")
            check_cluster
            show_service_status
            ;;
        "access")
            check_cluster
            show_access_info
            ;;
        "health")
            check_cluster
            test_service_health
            ;;
        "logs")
            if [[ -z "${2:-}" ]]; then
                log_error "Please specify a service name"
                echo "Available services: api-gateway, data-service, usp-service, mtp-service, cwmp-service, connection-manager, usp-agent, cwmp-agent"
                exit 1
            fi
            check_cluster
            show_logs "$2"
            ;;
        "follow")
            if [[ -z "${2:-}" ]]; then
                log_error "Please specify a service name"
                exit 1
            fi
            check_cluster
            show_logs "$2" "true"
            ;;
        "forward")
            if [[ -z "${2:-}" ]] || [[ -z "${3:-}" ]] || [[ -z "${4:-}" ]]; then
                log_error "Usage: $0 forward <service> <local-port> <service-port>"
                exit 1
            fi
            check_cluster
            port_forward_service "$2" "$3" "$4"
            ;;
        "shell")
            if [[ -z "${2:-}" ]]; then
                log_error "Please specify a service name"
                exit 1
            fi
            check_cluster
            exec_into_pod "$2"
            ;;
        "resources")
            check_cluster
            show_resource_usage
            ;;
        "grafana")
            check_cluster
            open_grafana_dashboards
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "Unknown command: $1"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"