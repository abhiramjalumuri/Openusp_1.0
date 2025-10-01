#!/bin/bash

# Quick verification script for Grafana dashboard functionality
echo "🔍 OpenUSP Grafana Dashboard Verification"
echo "========================================="

# Check if Grafana is accessible
echo "1. Checking Grafana accessibility..."
if curl -s -f http://localhost:3000/api/health > /dev/null; then
    echo "   ✅ Grafana is accessible at http://localhost:3000"
else
    echo "   ❌ Grafana is not accessible"
    exit 1
fi

# Check admin login
echo "2. Testing admin authentication..."
if curl -s -u admin:admin http://localhost:3000/api/user > /dev/null; then
    echo "   ✅ Admin login working (admin/admin)"
else
    echo "   ❌ Admin login failed"
    echo "   💡 Run: docker exec openusp-grafana-dev grafana-cli admin reset-admin-password admin"
    exit 1
fi

# Check data source configuration
echo "3. Verifying Prometheus data source..."
DS_INFO=$(curl -s -u admin:admin http://localhost:3000/api/datasources | jq -r '.[0] | {uid: .uid, name: .name, url: .url}')
DS_UID=$(echo "$DS_INFO" | jq -r '.uid')
DS_URL=$(echo "$DS_INFO" | jq -r '.url')

if [ "$DS_UID" = "prometheus" ]; then
    echo "   ✅ Data source UID is correct: $DS_UID"
else
    echo "   ⚠️  Data source UID mismatch: $DS_UID (expected: prometheus)"
    echo "   🔄 Running setup script to fix..."
    ./scripts/setup-grafana.sh > /dev/null 2>&1
fi

echo "   📊 Data source URL: $DS_URL"

# Test data source connectivity
echo "4. Testing Prometheus connectivity..."
METRICS_COUNT=$(curl -s -u admin:admin -X POST -H "Content-Type: application/json" -d '{
  "queries": [
    {
      "refId": "A",
      "expr": "up{job=~\"openusp-.*\"}",
      "range": true,
      "intervalMs": 15000,
      "maxDataPoints": 1000,
      "datasource": {
        "uid": "prometheus",
        "type": "prometheus"
      }
    }
  ],
  "from": "now-1h",
  "to": "now"
}' http://localhost:3000/api/ds/query | jq '.results.A.frames | length')

if [ "$METRICS_COUNT" -gt 0 ]; then
    echo "   ✅ OpenUSP services metrics available: $METRICS_COUNT services"
else
    echo "   ❌ No OpenUSP metrics found"
    echo "   💡 Check if services are running: make ou-status"
fi

# Check specific dashboard
echo "5. Verifying platform overview dashboard..."
DASHBOARD_CHECK=$(curl -s -u admin:admin http://localhost:3000/api/dashboards/uid/openusp-overview | jq -r '.meta.isStarred')
if [ "$DASHBOARD_CHECK" != "null" ]; then
    echo "   ✅ Platform overview dashboard is accessible"
    echo "   🔗 http://localhost:3000/d/openusp-overview/openusp-platform-overview"
else
    echo "   ⚠️  Dashboard may not be properly imported"
    echo "   🔄 Run: ./scripts/setup-grafana.sh"
fi

echo ""
echo "✅ Grafana Dashboard Verification Complete!"
echo ""
echo "📊 Quick Access Links:"
echo "   Platform Overview: http://localhost:3000/d/openusp-overview"
echo "   USP Protocol: http://localhost:3000/d/openusp-usp-protocol"
echo "   CWMP Protocol: http://localhost:3000/d/openusp-cwmp-protocol"
echo "   Data Service: http://localhost:3000/d/openusp-data-service"
echo ""
echo "🔑 Login: admin / admin"