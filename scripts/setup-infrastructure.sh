#!/bin/bash

# OpenUSP Infrastructure Setup Script
# Fixes Consul, Prometheus, and Grafana dashboard issues

echo "🔧 OpenUSP Infrastructure Setup & Repair"
echo "=========================================="

# Set working directory
cd "$(dirname "$0")/.."

echo "📊 Step 1: Checking current infrastructure status..."
make infra-status

echo ""
echo "🔄 Step 2: Restarting infrastructure services..."
echo "   This ensures proper service discovery and dashboard loading"

# Stop all infrastructure services
echo "   Stopping services..."
docker compose -f deployments/docker-compose.infra.network.yml down

# Clean up any orphaned containers
echo "   Cleaning up..."
docker system prune -f > /dev/null 2>&1

# Start services in proper order
echo "   Starting PostgreSQL..."
docker compose -f deployments/docker-compose.infra.network.yml up -d postgres
sleep 5

echo "   Starting Consul..."
docker compose -f deployments/docker-compose.infra.network.yml up -d consul
sleep 5

echo "   Starting Prometheus..."
docker compose -f deployments/docker-compose.infra.network.yml up -d prometheus-dev
sleep 5

echo "   Starting Grafana..."
docker compose -f deployments/docker-compose.infra.network.yml up -d grafana-dev
sleep 10

echo "   Starting remaining services..."
docker compose -f deployments/docker-compose.infra.network.yml up -d

echo ""
echo "⏳ Step 3: Waiting for services to be ready..."
sleep 10

echo ""
echo "🔍 Step 4: Verifying service health..."

# Check Consul
echo -n "   Consul: "
if curl -s http://localhost:8500/v1/status/leader > /dev/null 2>&1; then
    echo "✅ Running"
else
    echo "❌ Failed"
fi

# Check Prometheus
echo -n "   Prometheus: "
if curl -s http://localhost:9090/api/v1/status/config > /dev/null 2>&1; then
    echo "✅ Running"
else
    echo "❌ Failed"
fi

# Check Grafana
echo -n "   Grafana: "
if curl -s http://localhost:3000/api/health > /dev/null 2>&1; then
    echo "✅ Running"
else
    echo "❌ Failed"
fi

echo ""
echo "📋 Step 5: Final infrastructure status..."
make infra-status

echo ""
echo "🎉 Setup Complete!"
echo ""
echo "📊 Access Points:"
echo "   • Consul UI:   http://localhost:8500"
echo "   • Prometheus:  http://localhost:9090"
echo "   • Grafana:     http://localhost:3000 (admin/openusp123)"
echo "   • Adminer:     http://localhost:8080"
echo ""
echo "🔧 Troubleshooting:"
echo "   • Check logs: docker logs <container-name>"
echo "   • Restart service: docker restart <container-name>"
echo "   • Full reset: make infra-clean && make infra-up"