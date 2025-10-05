#!/bin/bash

# OpenUSP Development - Consul Cleanup Script
# Removes stale service registrations for clean development environment

set -e

CONSUL_ADDR=${CONSUL_ADDR:-localhost:8500}
SERVICES=(
    "openusp-api-gateway"
    "openusp-connection-manager" 
    "openusp-cwmp-service"
    "openusp-data-service"
    "openusp-mtp-service"
    "openusp-usp-service"
)

echo "🧹 OpenUSP Consul Cleanup"
echo "========================"
echo "Consul: $CONSUL_ADDR"
echo ""

# Function to cleanup a service
cleanup_service() {
    local service_name=$1
    echo "🔍 Checking $service_name..."
    
    # Get all instances of this service
    local instances=$(curl -s http://$CONSUL_ADDR/v1/catalog/service/$service_name | jq -r '.[].ServiceID' | grep -v null || true)
    
    if [ -z "$instances" ]; then
        echo "   ✅ No instances found"
        return
    fi
    
    local count=0
    local cleaned=0
    
    for instance_id in $instances; do
        count=$((count + 1))
        
        # Check if service is actually running (check health)
        local health=$(curl -s http://$CONSUL_ADDR/v1/health/service/$service_name | jq -r ".[] | select(.Service.ID == \"$instance_id\") | .Checks[] | select(.ServiceID == \"$instance_id\") | .Status" | head -1)
        
        if [ "$health" != "passing" ] && [ "$health" != "" ]; then
            echo "   🗑️  Removing unhealthy instance: $instance_id"
            curl -s -X PUT http://$CONSUL_ADDR/v1/agent/service/deregister/$instance_id > /dev/null
            cleaned=$((cleaned + 1))
        fi
    done
    
    if [ $cleaned -gt 0 ]; then
        echo "   ✅ Cleaned $cleaned/$count instances"
    else
        echo "   ✅ All $count instances healthy"
    fi
}

# Function to cleanup ALL instances of a service (force mode)
force_cleanup_service() {
    local service_name=$1
    echo "🔥 Force cleaning $service_name..."
    
    local instances=$(curl -s http://$CONSUL_ADDR/v1/catalog/service/$service_name | jq -r '.[].ServiceID' | grep -v null || true)
    
    for instance_id in $instances; do
        echo "   🗑️  Removing: $instance_id"
        curl -s -X PUT http://$CONSUL_ADDR/v1/agent/service/deregister/$instance_id > /dev/null
    done
}

# Check for force mode
if [ "$1" = "--force" ] || [ "$1" = "-f" ]; then
    echo "⚠️  FORCE MODE: Removing ALL service instances"
    echo ""
    
    for service in "${SERVICES[@]}"; do
        force_cleanup_service $service
    done
    
    echo ""
    echo "🔥 Force cleanup completed!"
    
elif [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --force, -f    Remove ALL service instances (use before restart)"
    echo "  --help, -h     Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0              # Clean only unhealthy instances"
    echo "  $0 --force      # Remove all instances (before service restart)"
    echo ""
    
else
    echo "🔍 Cleaning unhealthy service instances..."
    echo ""
    
    for service in "${SERVICES[@]}"; do
        cleanup_service $service
    done
    
    echo ""
    echo "✅ Cleanup completed!"
    echo ""
    echo "💡 Tips:"
    echo "   • Use '$0 --force' before restarting all services"
    echo "   • Add this to your development workflow"
    echo "   • Run after ungraceful service shutdowns"
fi

echo ""
echo "📊 Current service status:"
curl -s http://$CONSUL_ADDR/v1/catalog/services | jq -r 'keys[]' | grep openusp | while read service; do
    count=$(curl -s http://$CONSUL_ADDR/v1/catalog/service/$service | jq '. | length')
    echo "   $service: $count instances"
done