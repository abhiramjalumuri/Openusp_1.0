#!/bin/bash

# OpenUSP Grafana Dashboard Setup Script
# This script ensures Grafana dashboards are properly configured and accessible

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "🎯 Setting up OpenUSP Grafana Dashboards..."
echo "=============================================="

# Check if Grafana is running
if ! curl -s http://localhost:3000/api/health >/dev/null 2>&1; then
    echo "❌ Grafana is not running. Please start it first:"
    echo "   make infra-up"
    exit 1
fi

echo "✅ Grafana is running"

# Check if dashboards are provisioned
DASHBOARD_COUNT=$(curl -s -u admin:admin http://localhost:3000/api/search | jq -r '. | length')

if [ "$DASHBOARD_COUNT" -lt 4 ]; then
    echo "⚠️  Dashboards not properly provisioned. Restarting Grafana..."
    
    cd "$PROJECT_ROOT/deployments"
    docker-compose -f docker-compose.infra.yml stop grafana-dev
    docker-compose -f docker-compose.infra.yml rm -f grafana-dev
    docker-compose -f docker-compose.infra.yml up -d grafana-dev
    
    echo "⏳ Waiting for Grafana to initialize..."
    sleep 15
    
    # Verify dashboards are now available
    DASHBOARD_COUNT=$(curl -s -u admin:admin http://localhost:3000/api/search | jq -r '. | length')
fi

echo "📊 Found $DASHBOARD_COUNT dashboard(s)"

# List available dashboards
echo ""
echo "📋 Available OpenUSP Dashboards:"
curl -s -u admin:admin http://localhost:3000/api/search | jq -r '.[] | select(.type=="dash-db") | "   • \(.title) - http://localhost:3000\(.url)"'

echo ""
echo "🌐 Grafana Web UI: http://localhost:3000"
echo "   Username: admin"
echo "   Password: admin"
echo ""
echo "✅ Grafana dashboard setup completed!"