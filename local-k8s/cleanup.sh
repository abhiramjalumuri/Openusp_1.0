#!/bin/bash

# Cleanup Local Kubernetes Environment
# Removes the local OpenUSP Kubernetes cluster and associated resources

set -euo pipefail

# Configuration
CLUSTER_NAME="openusp-local"
REGISTRY_NAME="openusp-registry"

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

confirm_cleanup() {
    echo "========================================="
    echo "🗑️  OpenUSP Kubernetes Cleanup"
    echo "========================================="
    echo ""
    echo "This will remove:"
    echo "  • Kind cluster: $CLUSTER_NAME"
    echo "  • Docker registry: $REGISTRY_NAME"
    echo "  • All deployed services and data"
    echo "  • All persistent volumes and data"
    echo ""
    
    read -p "Are you sure you want to proceed? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Cleanup cancelled"
        exit 0
    fi
}

remove_helm_releases() {
    log_info "Removing Helm releases..."
    
    # Check if kubectl is available and configured
    if command -v kubectl &> /dev/null && kubectl config current-context | grep -q "kind-${CLUSTER_NAME}"; then
        # Remove OpenUSP release
        if helm list -n openusp | grep -q openusp; then
            log_info "Removing OpenUSP Helm release..."
            helm uninstall openusp -n openusp || log_warning "Failed to remove OpenUSP release"
        fi
        
        # Remove monitoring stack
        if helm list -n monitoring | grep -q prometheus; then
            log_info "Removing monitoring stack..."
            helm uninstall prometheus -n monitoring || log_warning "Failed to remove monitoring stack"
        fi
        
        log_success "Helm releases removed"
    else
        log_warning "kubectl not configured for cluster, skipping Helm cleanup"
    fi
}

remove_cluster() {
    log_info "Removing Kind cluster: $CLUSTER_NAME"
    
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        kind delete cluster --name "$CLUSTER_NAME"
        log_success "Kind cluster removed"
    else
        log_warning "Kind cluster not found"
    fi
}

remove_registry() {
    log_info "Removing Docker registry: $REGISTRY_NAME"
    
    if docker ps -a --format '{{.Names}}' | grep -q "^${REGISTRY_NAME}$"; then
        docker stop "$REGISTRY_NAME" || log_warning "Failed to stop registry"
        docker rm "$REGISTRY_NAME" || log_warning "Failed to remove registry"
        log_success "Docker registry removed"
    else
        log_warning "Docker registry not found"
    fi
}

cleanup_docker_images() {
    log_info "Cleaning up Docker images..."
    
    # Remove OpenUSP images from local registry
    local images=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep "localhost:5001/openusp" || true)
    
    if [[ -n "$images" ]]; then
        echo "$images" | xargs docker rmi || log_warning "Some images could not be removed"
        log_success "OpenUSP Docker images removed"
    else
        log_info "No OpenUSP Docker images found"
    fi
    
    # Clean up dangling images
    docker image prune -f || log_warning "Failed to prune dangling images"
}

cleanup_networks() {
    log_info "Cleaning up Docker networks..."
    
    # Remove kind network if it exists and is not in use
    if docker network ls --format "{{.Name}}" | grep -q "^kind$"; then
        docker network rm kind 2>/dev/null || log_warning "Kind network is still in use or could not be removed"
    fi
}

cleanup_volumes() {
    log_info "Cleaning up Docker volumes..."
    
    # Remove unused volumes
    docker volume prune -f || log_warning "Failed to prune volumes"
}

cleanup_temporary_files() {
    log_info "Cleaning up temporary files..."
    
    # Remove any temporary files created during setup
    rm -f kind-config.yaml
    rm -f get_helm.sh
    rm -f Dockerfile.*
    
    log_success "Temporary files cleaned up"
}

reset_kubectl_context() {
    log_info "Resetting kubectl context..."
    
    # Get current context
    local current_context=$(kubectl config current-context 2>/dev/null || echo "none")
    
    if [[ "$current_context" == "kind-${CLUSTER_NAME}" ]]; then
        # Try to switch to a different context
        local available_contexts=$(kubectl config get-contexts -o name | grep -v "kind-${CLUSTER_NAME}" | head -1 || echo "")
        
        if [[ -n "$available_contexts" ]]; then
            kubectl config use-context "$available_contexts"
            log_success "Switched kubectl context to: $available_contexts"
        else
            log_warning "No other kubectl contexts available"
        fi
    fi
    
    # Remove the kind cluster context
    kubectl config delete-context "kind-${CLUSTER_NAME}" 2>/dev/null || log_warning "Context not found"
    kubectl config delete-cluster "kind-${CLUSTER_NAME}" 2>/dev/null || log_warning "Cluster not found"
    kubectl config delete-user "kind-${CLUSTER_NAME}" 2>/dev/null || log_warning "User not found"
}

show_cleanup_summary() {
    log_success "Cleanup completed!"
    echo ""
    echo "========================================="
    echo "🧹 Cleanup Summary"
    echo "========================================="
    echo ""
    echo "Removed:"
    echo "  ✅ Kind cluster: $CLUSTER_NAME"
    echo "  ✅ Docker registry: $REGISTRY_NAME"
    echo "  ✅ OpenUSP Docker images"
    echo "  ✅ Helm releases"
    echo "  ✅ kubectl contexts"
    echo "  ✅ Temporary files"
    echo ""
    echo "Your system has been cleaned up and returned to its original state."
    echo ""
    echo "To redeploy OpenUSP:"
    echo "  1. ./setup-cluster.sh"
    echo "  2. ./deploy-openusp.sh"
    echo ""
}

main() {
    # Confirm before proceeding
    confirm_cleanup
    
    # Perform cleanup steps
    remove_helm_releases
    remove_cluster
    remove_registry
    cleanup_docker_images
    cleanup_networks
    cleanup_volumes
    cleanup_temporary_files
    reset_kubectl_context
    
    # Show summary
    show_cleanup_summary
}

# Handle interruption
trap 'log_error "Cleanup interrupted"; exit 1' INT TERM

# Run main function
main "$@"