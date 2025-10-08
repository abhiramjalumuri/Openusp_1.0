#!/bin/bash

# Setup Local Kubernetes Cluster for OpenUSP
# Creates a Kind cluster optimized for OpenUSP services

set -euo pipefail

# Configuration
CLUSTER_NAME="openusp-local"
KUBERNETES_VERSION="v1.28.0"
REGISTRY_NAME="openusp-registry"
REGISTRY_PORT="5001"

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
    
    local missing_tools=()
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        missing_tools+=("docker")
    fi
    
    # Check Kind
    if ! command -v kind &> /dev/null; then
        log_warning "Kind not found. Installing..."
        install_kind
    fi
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        missing_tools+=("kubectl")
    fi
    
    # Check Helm
    if ! command -v helm &> /dev/null; then
        log_warning "Helm not found. Installing..."
        install_helm
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install them manually:"
        echo "  - Docker: https://docs.docker.com/get-docker/"
        echo "  - kubectl: https://kubernetes.io/docs/tasks/tools/"
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info &> /dev/null; then
        log_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    
    log_success "All prerequisites satisfied"
}

install_kind() {
    log_info "Installing Kind..."
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if command -v brew &> /dev/null; then
            brew install kind
        else
            curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-darwin-amd64
            chmod +x ./kind
            sudo mv ./kind /usr/local/bin/kind
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
        chmod +x ./kind
        sudo mv ./kind /usr/local/bin/kind
    else
        log_error "Unsupported OS for automatic Kind installation"
        exit 1
    fi
    
    log_success "Kind installed successfully"
}

install_helm() {
    log_info "Installing Helm..."
    
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
    chmod 700 get_helm.sh
    ./get_helm.sh
    rm get_helm.sh
    
    log_success "Helm installed successfully"
}

create_kind_config() {
    log_info "Creating Kind cluster configuration..."
    
    cat > kind-config.yaml << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${CLUSTER_NAME}
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  # API Gateway
  - containerPort: 30080
    hostPort: 8080
    protocol: TCP
  # Data Service
  - containerPort: 30081
    hostPort: 8081
    protocol: TCP
  # USP Service
  - containerPort: 30082
    hostPort: 8082
    protocol: TCP
  # MTP Service
  - containerPort: 30083
    hostPort: 8083
    protocol: TCP
  # CWMP Service
  - containerPort: 30547
    hostPort: 7547
    protocol: TCP
  # Connection Manager
  - containerPort: 30084
    hostPort: 8084
    protocol: TCP
  # Grafana
  - containerPort: 30000
    hostPort: 3000
    protocol: TCP
  # Prometheus
  - containerPort: 30090
    hostPort: 9090
    protocol: TCP
  # PostgreSQL
  - containerPort: 30432
    hostPort: 5432
    protocol: TCP
  # RabbitMQ Management
  - containerPort: 30672
    hostPort: 15672
    protocol: TCP
  # Mosquitto MQTT
  - containerPort: 31883
    hostPort: 1883
    protocol: TCP
  # Consul
  - containerPort: 30500
    hostPort: 8500
    protocol: TCP
- role: worker
- role: worker
EOF
}

setup_local_registry() {
    log_info "Setting up local Docker registry..."
    
    # Check if registry already exists
    if docker ps --format '{{.Names}}' | grep -q "^${REGISTRY_NAME}$"; then
        log_info "Local registry already running"
        return 0
    fi
    
    # Start local registry
    docker run -d --restart=always -p "${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" registry:2
    
    # Wait for registry to be ready
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if curl -s "http://localhost:${REGISTRY_PORT}/v2/" > /dev/null; then
            log_success "Local registry is ready"
            return 0
        fi
        
        log_info "Waiting for registry to be ready... (attempt $((attempt + 1))/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    log_error "Local registry failed to start"
    return 1
}

create_cluster() {
    log_info "Creating Kind cluster: ${CLUSTER_NAME}..."
    
    # Check if cluster already exists
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_warning "Cluster ${CLUSTER_NAME} already exists. Deleting..."
        kind delete cluster --name "${CLUSTER_NAME}"
    fi
    
    # Create the cluster
    kind create cluster --config kind-config.yaml --image "kindest/node:${KUBERNETES_VERSION}"
    
    # Wait for cluster to be ready
    log_info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s
    
    log_success "Cluster created successfully"
}

connect_registry_to_cluster() {
    log_info "Connecting local registry to cluster..."
    
    # Connect the registry to the cluster network if not already connected
    if ! docker network ls | grep -q "kind"; then
        log_warning "Kind network not found, creating..."
        docker network create kind
    fi
    
    # Connect registry to kind network
    docker network connect "kind" "${REGISTRY_NAME}" 2>/dev/null || true
    
    # Document the local registry
    kubectl apply -f - << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REGISTRY_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
}

install_ingress_controller() {
    log_info "Installing NGINX Ingress Controller..."
    
    # Install NGINX Ingress Controller
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
    
    # Wait for ingress controller to be ready
    log_info "Waiting for ingress controller to be ready..."
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=300s
    
    log_success "Ingress controller installed"
}

setup_storage_class() {
    log_info "Setting up storage class..."
    
    # Create a default storage class for local development
    kubectl apply -f - << EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: openusp-local-storage
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: rancher.io/local-path
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
EOF
    
    log_success "Storage class configured"
}

create_namespaces() {
    log_info "Creating namespaces..."
    
    # Create namespaces
    kubectl create namespace openusp --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace openusp-infra --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Namespaces created"
}

add_helm_repositories() {
    log_info "Adding Helm repositories..."
    
    # Add necessary Helm repositories
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo add grafana https://grafana.github.io/helm-charts
    helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
    
    # Update repositories
    helm repo update
    
    log_success "Helm repositories added and updated"
}

display_cluster_info() {
    log_success "Kubernetes cluster setup completed!"
    echo ""
    echo "========================================="
    echo "🎉 OpenUSP Local Kubernetes Cluster Ready!"
    echo "========================================="
    echo ""
    echo "Cluster Information:"
    echo "  • Cluster Name: ${CLUSTER_NAME}"
    echo "  • Kubernetes Version: ${KUBERNETES_VERSION}"
    echo "  • Nodes: $(kubectl get nodes --no-headers | wc -l | tr -d ' ')"
    echo "  • Local Registry: localhost:${REGISTRY_PORT}"
    echo ""
    echo "Service Access Points:"
    echo "  • API Gateway: http://localhost:8080"
    echo "  • Data Service: http://localhost:8081"
    echo "  • USP Service: http://localhost:8082"
    echo "  • MTP Service: http://localhost:8083"
    echo "  • CWMP Service: http://localhost:7547"
    echo "  • Connection Manager: http://localhost:8084"
    echo "  • Grafana: http://localhost:3000"
    echo "  • Prometheus: http://localhost:9090"
    echo "  • PostgreSQL: localhost:5432"
    echo "  • RabbitMQ: http://localhost:15672"
    echo "  • Mosquitto MQTT: localhost:1883"
    echo "  • Consul: http://localhost:8500"
    echo ""
    echo "Next Steps:"
    echo "  1. Deploy OpenUSP services: ./deploy-openusp.sh"
    echo "  2. Monitor deployments: kubectl get pods -A"
    echo "  3. View cluster dashboard: kubectl proxy"
    echo ""
    echo "Useful Commands:"
    echo "  • kubectl get nodes"
    echo "  • kubectl get pods -A"
    echo "  • helm list -A"
    echo "  • kind delete cluster --name ${CLUSTER_NAME}"
    echo ""
}

cleanup() {
    log_info "Cleaning up temporary files..."
    rm -f kind-config.yaml get_helm.sh
}

main() {
    echo "========================================="
    echo "🚀 OpenUSP Local Kubernetes Setup"
    echo "========================================="
    echo ""
    
    # Setup steps
    check_prerequisites
    create_kind_config
    setup_local_registry
    create_cluster
    connect_registry_to_cluster
    install_ingress_controller
    setup_storage_class
    create_namespaces
    add_helm_repositories
    
    # Cleanup and display info
    cleanup
    display_cluster_info
}

# Handle interruption
trap 'log_error "Setup interrupted"; cleanup; exit 1' INT TERM

# Run main function
main "$@"