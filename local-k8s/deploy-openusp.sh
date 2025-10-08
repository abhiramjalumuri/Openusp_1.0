#!/bin/bash

# Deploy OpenUSP Services to Local Kubernetes
# Deploys all OpenUSP components using Helm charts

set -euo pipefail

# Configuration
CLUSTER_NAME="openusp-local"
RELEASE_NAME="openusp"
NAMESPACE="openusp"
HELM_CHART_PATH="./helm/openusp"
VALUES_FILE="./helm/openusp/values.yaml"

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

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if cluster exists
    if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_error "Kind cluster '${CLUSTER_NAME}' not found"
        log_error "Please run ./setup-cluster.sh first"
        exit 1
    fi
    
    # Check if kubectl is configured for the right cluster
    if ! kubectl cluster-info --context "kind-${CLUSTER_NAME}" &> /dev/null; then
        log_error "kubectl not configured for cluster '${CLUSTER_NAME}'"
        exit 1
    fi
    
    # Set kubectl context
    kubectl config use-context "kind-${CLUSTER_NAME}"
    
    # Check if Helm is available
    if ! command -v helm &> /dev/null; then
        log_error "Helm not found. Please install Helm first."
        exit 1
    fi
    
    # Check if helm chart exists
    if [[ ! -d "$HELM_CHART_PATH" ]]; then
        log_error "Helm chart not found at: $HELM_CHART_PATH"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

build_local_images() {
    log_info "Building local Docker images..."
    
    # Local registry configuration
    local registry="localhost:5001"
    
    # Check if we have a local registry running
    if ! curl -s "http://localhost:5001/v2/" > /dev/null; then
        log_error "Local Docker registry not running"
        log_error "Please ensure the registry is started by setup-cluster.sh"
        exit 1
    fi
    
    # Build images for each service
    local services=("api-gateway" "data-service" "usp-service" "mtp-service" "cwmp-service" "connection-manager" "usp-agent" "cwmp-agent")
    
    for service in "${services[@]}"; do
        log_info "Building image for $service..."
        
        # Create a simple Dockerfile for demo purposes
        # In real deployment, these would be built from actual service code
        local dockerfile="Dockerfile.${service}"
        
        cat > "$dockerfile" << EOF
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN echo 'package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "OpenUSP ${service} - Hello from Kubernetes!")
    })
    
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "OK")
    })
    
    http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "READY")
    })
    
    http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain")
        fmt.Fprintf(w, "# OpenUSP ${service} metrics\\n")
        fmt.Fprintf(w, "${service}_requests_total 42\\n")
    })
    
    log.Printf("${service} starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}' > main.go

RUN go mod init ${service} && go build -o ${service} main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/${service} .
EXPOSE 8080
CMD ["./$(echo ${service} | tr '-' '_')"]
EOF
        
        # Build and push to local registry
        docker build -f "$dockerfile" -t "${registry}/openusp/${service}:latest" .
        docker push "${registry}/openusp/${service}:latest"
        
        # Cleanup
        rm -f "$dockerfile"
        
        log_success "Built and pushed: ${registry}/openusp/${service}:latest"
    done
    
    log_success "All images built and pushed to local registry"
}

create_namespace() {
    log_info "Creating namespace: $NAMESPACE"
    
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Namespace created: $NAMESPACE"
}

add_helm_repositories() {
    log_info "Adding required Helm repositories..."
    
    # Add Bitnami repository for PostgreSQL and RabbitMQ
    helm repo add bitnami https://charts.bitnami.com/bitnami
    
    # Add Prometheus community repository
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    
    # Update repositories
    helm repo update
    
    log_success "Helm repositories added and updated"
}

install_monitoring() {
    log_info "Installing monitoring stack..."
    
    # Create monitoring namespace
    kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
    
    # Install Prometheus
    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace monitoring \
        --set prometheus.service.type=NodePort \
        --set prometheus.service.nodePort=30090 \
        --set grafana.service.type=NodePort \
        --set grafana.service.nodePort=30000 \
        --set grafana.adminPassword=admin \
        --wait
    
    log_success "Monitoring stack installed"
}

deploy_openusp() {
    log_info "Deploying OpenUSP services..."
    
    # Validate Helm chart
    helm lint "$HELM_CHART_PATH"
    
    # Deploy with Helm
    helm upgrade --install "$RELEASE_NAME" "$HELM_CHART_PATH" \
        --namespace "$NAMESPACE" \
        --values "$VALUES_FILE" \
        --set global.registry=localhost:5001 \
        --set services.apiGateway.serviceType=NodePort \
        --set services.dataService.serviceType=NodePort \
        --set services.uspService.serviceType=NodePort \
        --set services.mtpService.serviceType=NodePort \
        --set services.cwmpService.serviceType=NodePort \
        --set services.connectionManager.serviceType=NodePort \
        --set mosquitto.service.type=NodePort \
        --set consul.service.type=NodePort \
        --wait \
        --timeout=600s
    
    log_success "OpenUSP services deployed successfully"
}

wait_for_pods() {
    log_info "Waiting for all pods to be ready..."
    
    # Wait for all pods in the namespace to be ready
    kubectl wait --for=condition=Ready pods --all -n "$NAMESPACE" --timeout=300s
    
    log_success "All pods are ready"
}

display_service_info() {
    log_success "OpenUSP deployment completed!"
    echo ""
    echo "========================================="
    echo "🎉 OpenUSP Services Deployed Successfully!"
    echo "========================================="
    echo ""
    
    # Display pod status
    echo "Pod Status:"
    kubectl get pods -n "$NAMESPACE" -o wide
    echo ""
    
    # Display service information
    echo "Service Access Points:"
    echo "  🌐 API Gateway: http://localhost:8080"
    echo "  💾 Data Service: http://localhost:8081"
    echo "  🔗 USP Service: http://localhost:8082"
    echo "  📡 MTP Service: http://localhost:8083"
    echo "  📞 CWMP Service: http://localhost:7547"
    echo "  🔌 Connection Manager: http://localhost:8084"
    echo ""
    
    echo "Infrastructure Services:"
    echo "  🐘 PostgreSQL: localhost:5432"
    echo "  🐰 RabbitMQ Management: http://localhost:15672"
    echo "  🦟 Mosquitto MQTT: localhost:1883"
    echo "  🏛️  Consul UI: http://localhost:8500"
    echo ""
    
    echo "Monitoring:"
    echo "  📊 Grafana: http://localhost:3000 (admin/admin)"
    echo "  📈 Prometheus: http://localhost:9090"
    echo ""
    
    echo "Useful Commands:"
    echo "  • View all pods: kubectl get pods -n $NAMESPACE"
    echo "  • View services: kubectl get svc -n $NAMESPACE"
    echo "  • View logs: kubectl logs -f deployment/<service-name> -n $NAMESPACE"
    echo "  • Port forward: kubectl port-forward svc/<service-name> 8080:8080 -n $NAMESPACE"
    echo "  • Shell into pod: kubectl exec -it deployment/<service-name> -n $NAMESPACE -- /bin/sh"
    echo ""
    
    echo "Helm Commands:"
    echo "  • View release: helm list -n $NAMESPACE"
    echo "  • Upgrade: helm upgrade $RELEASE_NAME $HELM_CHART_PATH -n $NAMESPACE"
    echo "  • Uninstall: helm uninstall $RELEASE_NAME -n $NAMESPACE"
    echo ""
}

test_services() {
    log_info "Testing service endpoints..."
    
    local services=("api-gateway:8080" "data-service:8081" "usp-service:8082" "mtp-service:8083" "cwmp-service:7547" "connection-manager:8084")
    
    for service_port in "${services[@]}"; do
        local service="${service_port%:*}"
        local port="${service_port#*:}"
        
        log_info "Testing $service on port $port..."
        
        # Wait a bit for services to start
        sleep 2
        
        if curl -s --max-time 5 "http://localhost:$port/health" > /dev/null; then
            log_success "$service is responding"
        else
            log_warning "$service may not be ready yet"
        fi
    done
}

cleanup_build_artifacts() {
    log_info "Cleaning up build artifacts..."
    
    # Remove any temporary Dockerfiles
    rm -f Dockerfile.*
    
    log_success "Build artifacts cleaned up"
}

main() {
    echo "========================================="
    echo "🚀 OpenUSP Kubernetes Deployment"
    echo "========================================="
    echo ""
    
    # Execute deployment steps
    check_prerequisites
    build_local_images
    create_namespace
    add_helm_repositories
    install_monitoring
    deploy_openusp
    wait_for_pods
    test_services
    cleanup_build_artifacts
    display_service_info
}

# Handle interruption
trap 'log_error "Deployment interrupted"; cleanup_build_artifacts; exit 1' INT TERM

# Run main function
main "$@"