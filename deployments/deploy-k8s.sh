#!/bin/bash

# OpenUSP Kubernetes Deployment Script
# This script deploys the complete OpenUSP platform to Kubernetes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="openusp"
DEPLOY_DIR="$(dirname "$0")"
KUBE_DIR="${DEPLOY_DIR}/kubernetes"

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

# Function to check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        print_error "Unable to connect to Kubernetes cluster"
        exit 1
    fi
    
    print_success "kubectl is available and connected to cluster"
}

# Function to check if Docker images exist
check_images() {
    local images=(
        "openusp/api-gateway:latest"
        "openusp/data-service:latest"
        "openusp/mtp-service:latest"
        "openusp/cwmp-service:latest"
        "openusp/usp-service:latest"
    )
    
    print_status "Checking Docker images..."
    for image in "${images[@]}"; do
        if ! docker image inspect "$image" &> /dev/null; then
            print_warning "Image $image not found locally. Make sure to build images first:"
            echo "  make docker-build"
            echo "  or"
            echo "  docker-compose -f docker-compose.yml build"
        else
            print_success "Image $image found"
        fi
    done
}

# Function to create namespace and secrets
setup_namespace() {
    print_status "Setting up namespace and secrets..."
    kubectl apply -f "${KUBE_DIR}/00-namespace.yaml"
    print_success "Namespace and secrets configured"
}

# Function to deploy PostgreSQL
deploy_postgres() {
    print_status "Deploying PostgreSQL..."
    kubectl apply -f "${KUBE_DIR}/01-postgres.yaml"
    
    print_status "Waiting for PostgreSQL to be ready..."
    kubectl wait --for=condition=ready pod -l app=postgres -n "$NAMESPACE" --timeout=300s
    print_success "PostgreSQL deployed successfully"
}

# Function to deploy data service
deploy_data_service() {
    print_status "Deploying Data Service..."
    kubectl apply -f "${KUBE_DIR}/02-data-service.yaml"
    
    print_status "Waiting for Data Service to be ready..."
    kubectl wait --for=condition=available deployment/data-service -n "$NAMESPACE" --timeout=300s
    print_success "Data Service deployed successfully"
}

# Function to deploy API Gateway
deploy_api_gateway() {
    print_status "Deploying API Gateway..."
    kubectl apply -f "${KUBE_DIR}/03-api-gateway.yaml"
    
    print_status "Waiting for API Gateway to be ready..."
    kubectl wait --for=condition=available deployment/api-gateway -n "$NAMESPACE" --timeout=300s
    print_success "API Gateway deployed successfully"
}

# Function to deploy MTP Service
deploy_mtp_service() {
    print_status "Deploying MTP Service..."
    kubectl apply -f "${KUBE_DIR}/04-mtp-service.yaml"
    
    print_status "Waiting for MTP Service to be ready..."
    kubectl wait --for=condition=available deployment/mtp-service -n "$NAMESPACE" --timeout=300s
    print_success "MTP Service deployed successfully"
}

# Function to deploy CWMP Service
deploy_cwmp_service() {
    print_status "Deploying CWMP Service..."
    kubectl apply -f "${KUBE_DIR}/05-cwmp-service.yaml"
    
    print_status "Waiting for CWMP Service to be ready..."
    kubectl wait --for=condition=available deployment/cwmp-service -n "$NAMESPACE" --timeout=300s
    print_success "CWMP Service deployed successfully"
}

# Function to deploy USP Service
deploy_usp_service() {
    print_status "Deploying USP Service..."
    kubectl apply -f "${KUBE_DIR}/06-usp-service.yaml"
    
    print_status "Waiting for USP Service to be ready..."
    kubectl wait --for=condition=available deployment/usp-service -n "$NAMESPACE" --timeout=300s
    print_success "USP Service deployed successfully"
}

# Function to deploy message brokers
deploy_brokers() {
    print_status "Deploying RabbitMQ..."
    kubectl apply -f "${KUBE_DIR}/07-rabbitmq.yaml"
    
    print_status "Deploying Mosquitto MQTT..."
    kubectl apply -f "${KUBE_DIR}/08-mosquitto.yaml"
    
    print_status "Waiting for message brokers to be ready..."
    kubectl wait --for=condition=ready pod -l app=rabbitmq -n "$NAMESPACE" --timeout=300s
    kubectl wait --for=condition=available deployment/mosquitto -n "$NAMESPACE" --timeout=300s
    print_success "Message brokers deployed successfully"
}

# Function to deploy monitoring stack
deploy_monitoring() {
    print_status "Deploying Prometheus..."
    kubectl apply -f "${KUBE_DIR}/09-prometheus.yaml"
    
    print_status "Deploying Grafana..."
    kubectl apply -f "${KUBE_DIR}/10-grafana.yaml"
    
    print_status "Waiting for monitoring stack to be ready..."
    kubectl wait --for=condition=available deployment/prometheus -n "$NAMESPACE" --timeout=300s
    kubectl wait --for=condition=available deployment/grafana -n "$NAMESPACE" --timeout=300s
    print_success "Monitoring stack deployed successfully"
}

# Function to show deployment status
show_status() {
    print_status "Deployment Status:"
    echo
    kubectl get pods -n "$NAMESPACE" -o wide
    echo
    kubectl get services -n "$NAMESPACE"
    echo
    kubectl get ingress -n "$NAMESPACE"
}

# Function to show service endpoints
show_endpoints() {
    print_status "Service Endpoints:"
    echo
    
    # Get LoadBalancer external IPs
    API_GATEWAY_IP=$(kubectl get service api-gateway -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    MTP_WEBSOCKET_IP=$(kubectl get service mtp-websocket -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    CWMP_EXTERNAL_IP=$(kubectl get service cwmp-external -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    MOSQUITTO_EXTERNAL_IP=$(kubectl get service mosquitto-external -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    
    echo "External Services:"
    echo "  API Gateway:     http://${API_GATEWAY_IP}:8080"
    echo "  MTP WebSocket:   ws://${MTP_WEBSOCKET_IP}:8081/ws"
    echo "  CWMP Service:    http://${CWMP_EXTERNAL_IP}:7547"
    echo "  MQTT Broker:     mqtt://${MOSQUITTO_EXTERNAL_IP}:1883"
    echo
    
    echo "Internal Services (kubectl port-forward):"
    echo "  kubectl port-forward -n $NAMESPACE svc/api-gateway 8080:8080"
    echo "  kubectl port-forward -n $NAMESPACE svc/mtp-service 8081:8081"
    echo "  kubectl port-forward -n $NAMESPACE svc/cwmp-service 7547:7547"
    echo "  kubectl port-forward -n $NAMESPACE svc/grafana 3000:3000"
    echo "  kubectl port-forward -n $NAMESPACE svc/prometheus 9090:9090"
    echo "  kubectl port-forward -n $NAMESPACE svc/rabbitmq 15672:15672"
    echo
    
    echo "Health Check URLs (after port-forward):"
    echo "  API Gateway:     http://localhost:8080/health"
    echo "  MTP Service:     http://localhost:8081/health"
    echo "  CWMP Service:    http://localhost:7547/health"
    echo "  Grafana:         http://localhost:3000 (admin/admin)"
    echo "  Prometheus:      http://localhost:9090"
    echo "  RabbitMQ:        http://localhost:15672 (admin/password)"
}

# Function to clean up deployment
cleanup() {
    print_warning "Cleaning up OpenUSP deployment..."
    
    # Delete in reverse order
    kubectl delete -f "${KUBE_DIR}/10-grafana.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/09-prometheus.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/08-mosquitto.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/07-rabbitmq.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/06-usp-service.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/05-cwmp-service.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/04-mtp-service.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/03-api-gateway.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/02-data-service.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/01-postgres.yaml" --ignore-not-found=true
    kubectl delete -f "${KUBE_DIR}/00-namespace.yaml" --ignore-not-found=true
    
    print_success "Cleanup completed"
}

# Function to show help
show_help() {
    echo "OpenUSP Kubernetes Deployment Script"
    echo
    echo "Usage: $0 [COMMAND]"
    echo
    echo "Commands:"
    echo "  deploy      Deploy the complete OpenUSP platform (default)"
    echo "  status      Show deployment status"
    echo "  endpoints   Show service endpoints"
    echo "  cleanup     Remove all OpenUSP resources"
    echo "  help        Show this help message"
    echo
    echo "Environment Variables:"
    echo "  NAMESPACE   Kubernetes namespace (default: openusp)"
    echo
    echo "Examples:"
    echo "  $0                    # Deploy everything"
    echo "  $0 deploy            # Deploy everything"
    echo "  $0 status            # Show status"
    echo "  $0 cleanup           # Clean up"
    echo
}

# Main deployment function
deploy_all() {
    print_status "Starting OpenUSP Kubernetes deployment..."
    
    check_kubectl
    check_images
    
    # Deploy core infrastructure
    setup_namespace
    deploy_postgres
    deploy_data_service
    
    # Deploy OpenUSP services
    deploy_api_gateway
    deploy_mtp_service
    deploy_cwmp_service
    deploy_usp_service
    
    # Deploy supporting services
    deploy_brokers
    deploy_monitoring
    
    print_success "OpenUSP deployment completed successfully!"
    echo
    show_endpoints
}

# Parse command line arguments
case "${1:-deploy}" in
    "deploy")
        deploy_all
        ;;
    "status")
        show_status
        ;;
    "endpoints")
        show_endpoints
        ;;
    "cleanup")
        cleanup
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac