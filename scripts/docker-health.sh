#!/bin/bash

# Docker Health Check and Repair Utility for OpenUSP
# Helps diagnose and fix common Docker issues

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}🔧 OpenUSP Docker Health Check${NC}"
    echo "================================="
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check Docker installation
check_docker_installation() {
    print_info "Checking Docker installation..."
    
    if ! command -v docker >/dev/null 2>&1; then
        print_error "Docker not found"
        echo "Please install Docker from https://docs.docker.com/get-docker/"
        return 1
    fi
    
    print_success "Docker CLI found"
    return 0
}

# Check Docker daemon
check_docker_daemon() {
    print_info "Checking Docker daemon..."
    
    if ! docker version >/dev/null 2>&1; then
        print_error "Docker daemon not running"
        echo "Please start Docker Desktop or Docker daemon"
        return 1
    fi
    
    print_success "Docker daemon running"
    return 0
}

# Check Docker Compose
check_docker_compose() {
    print_info "Checking Docker Compose..."
    
    if ! docker compose version >/dev/null 2>&1; then
        print_error "Docker Compose plugin not found"
        echo "Please install or update Docker to get the Compose plugin"
        return 1
    fi
    
    print_success "Docker Compose plugin available"
    return 0
}

# Check broken Docker plugins
check_docker_plugins() {
    print_info "Checking Docker plugins..."
    
    local plugins_dir="$HOME/.docker/cli-plugins"
    local broken_plugins=0
    
    if [[ -d "$plugins_dir" ]]; then
        for plugin in "$plugins_dir"/docker-*; do
            if [[ -L "$plugin" ]] && [[ ! -e "$plugin" ]]; then
                local plugin_name=$(basename "$plugin")
                print_warning "Broken plugin symlink: $plugin_name"
                
                read -p "Remove broken plugin $plugin_name? (y/N): " confirm
                if [[ "$confirm" == "y" ]] || [[ "$confirm" == "Y" ]]; then
                    rm "$plugin"
                    print_success "Removed broken plugin: $plugin_name"
                fi
                
                ((broken_plugins++))
            fi
        done
        
        if [[ $broken_plugins -eq 0 ]]; then
            print_success "No broken plugins found"
        fi
    else
        print_info "No Docker plugins directory found"
    fi
    
    return 0
}

# Check Docker context
check_docker_context() {
    print_info "Checking Docker context..."
    
    local context=$(docker context show 2>/dev/null || echo "unknown")
    echo "Current context: $context"
    
    case "$context" in
        desktop*)
            print_success "Using Docker Desktop context"
            ;;
        default)
            print_info "Using default context"
            ;;
        *)
            print_warning "Unknown context: $context"
            ;;
    esac
    
    return 0
}

# Check Docker network connectivity
check_docker_network() {
    print_info "Checking Docker network connectivity..."
    
    if docker run --rm alpine:latest ping -c 1 8.8.8.8 >/dev/null 2>&1; then
        print_success "Docker network connectivity working"
    else
        print_warning "Docker network connectivity issues"
        echo "This might affect container-to-internet communication"
    fi
    
    return 0
}

# Platform-specific checks
check_platform_specific() {
    local platform="$(uname -s)"
    
    print_info "Running platform-specific checks for $platform..."
    
    case "$platform" in
        Darwin)
            # macOS specific checks
            if command -v docker-desktop >/dev/null 2>&1; then
                print_success "Docker Desktop CLI found"
            else
                print_info "Docker Desktop CLI not in PATH (this is normal)"
            fi
            
            # Check if Docker Desktop app is running
            if pgrep -f "Docker Desktop" >/dev/null 2>&1; then
                print_success "Docker Desktop app is running"
            else
                print_warning "Docker Desktop app may not be running"
            fi
            ;;
            
        Linux)
            # Linux specific checks
            if systemctl is-active --quiet docker 2>/dev/null; then
                print_success "Docker service is active"
            elif service docker status >/dev/null 2>&1; then
                print_success "Docker service is running"
            else
                print_warning "Docker service status unclear"
            fi
            
            # Check user permissions
            if groups | grep -q docker; then
                print_success "User is in docker group"
            else
                print_warning "User not in docker group - may need sudo"
                echo "Consider running: sudo usermod -aG docker \$USER"
            fi
            ;;
    esac
    
    return 0
}

# Fix common issues
fix_common_issues() {
    print_info "Attempting to fix common Docker issues..."
    
    # Clean up broken plugins (already done in check_docker_plugins)
    
    # Clean up Docker system
    read -p "Clean up Docker system (removes unused containers, networks, images)? (y/N): " confirm
    if [[ "$confirm" == "y" ]] || [[ "$confirm" == "Y" ]]; then
        print_info "Cleaning Docker system..."
        docker system prune -f
        print_success "Docker system cleaned"
    fi
    
    return 0
}

# Main health check
main() {
    local command="${1:-check}"
    
    print_header
    echo ""
    
    case "$command" in
        check)
            check_docker_installation || exit 1
            check_docker_daemon || exit 1  
            check_docker_compose || exit 1
            check_docker_plugins
            check_docker_context
            check_docker_network
            check_platform_specific
            
            echo ""
            print_success "Docker health check completed!"
            echo ""
            print_info "If you're still having issues, try:"
            echo "  $0 fix    - Attempt to fix common issues"
            echo "  $0 info   - Show detailed Docker information"
            ;;
            
        fix)
            check_docker_installation || exit 1
            fix_common_issues
            echo ""
            print_success "Docker fix completed!"
            ;;
            
        info)
            echo "Docker Version Information:"
            docker version 2>/dev/null || echo "Docker version not available"
            echo ""
            echo "Docker System Information:"
            docker info 2>/dev/null || echo "Docker info not available"
            echo ""
            echo "Docker Context Information:"
            docker context ls 2>/dev/null || echo "Docker context not available"
            ;;
            
        *)
            echo "Usage: $0 {check|fix|info}"
            echo ""
            echo "Commands:"
            echo "  check  - Run comprehensive Docker health check"
            echo "  fix    - Attempt to fix common Docker issues"  
            echo "  info   - Show detailed Docker information"
            exit 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi