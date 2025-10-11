#!/bin/bash
# OpenUSP Infrastructure Startup Script
# Automatically detects the platform and uses the appropriate configuration

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.infra.yml"

# Detect platform
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "🐧 Linux detected - Using Linux-specific configuration"
    LINUX_OVERRIDE="$SCRIPT_DIR/docker-compose.linux.yml"
    
    if [[ -f "$LINUX_OVERRIDE" ]]; then
        echo "📦 Starting infrastructure with Linux host network configuration..."
        docker-compose -f "$COMPOSE_FILE" -f "$LINUX_OVERRIDE" up -d
    else
        echo "⚠️  Linux override file not found, using bridge network with gateway IP"
        # Update the main config to use Linux bridge gateway
        sed -i.bak 's/host\.docker\.internal/172.17.0.1/g' ../configs/prometheus-static.yml
        docker-compose -f "$COMPOSE_FILE" up -d
        # Restore original config
        mv ../configs/prometheus-static.yml.bak ../configs/prometheus-static.yml
    fi
    
elif [[ "$OSTYPE" == "darwin"* ]]; then
    echo "🍎 macOS detected - Using Docker Desktop configuration"
    docker-compose -f "$COMPOSE_FILE" up -d
    
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    echo "🪟 Windows detected - Using Docker Desktop configuration"
    docker-compose -f "$COMPOSE_FILE" up -d
    
else
    echo "❓ Unknown platform: $OSTYPE"
    echo "Trying default configuration..."
    docker-compose -f "$COMPOSE_FILE" up -d
fi

echo "✅ Infrastructure started successfully"
echo ""
echo "🌐 Access URLs:"
echo "  📊 Grafana:    http://localhost:3000 (admin/openusp123)"
echo "  📈 Prometheus: http://localhost:9090"
echo "  🐘 PostgreSQL: localhost:5433 (openusp/openusp123)"
echo "  🐰 RabbitMQ:   http://localhost:15672 (guest/guest)"
echo "  🦟 Mosquitto:  localhost:1883"