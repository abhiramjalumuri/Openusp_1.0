#!/bin/bash

# OpenUSP API Gateway Health Check Script
# This script finds the current API Gateway port and tests the health endpoint

echo "🔍 OpenUSP API Gateway Health Check"
echo "==================================="

# Check if Consul is available
if ! curl -s http://localhost:8500/api/v1/status/leader > /dev/null 2>&1; then
    echo "❌ Consul is not available at localhost:8500"
    echo "💡 Make sure infrastructure is running: make infra-up"
    exit 1
fi

# Get API Gateway service information from Consul
echo "🔍 Looking up API Gateway service in Consul..."
API_GW_INFO=$(curl -s http://localhost:8500/v1/catalog/service/openusp-api-gateway | jq -r '.[0]' 2>/dev/null)

if [ "$API_GW_INFO" = "null" ] || [ -z "$API_GW_INFO" ]; then
    echo "❌ API Gateway service not found in Consul"
    echo "💡 Make sure API Gateway is running: make start-api-gateway"
    exit 1
fi

# Extract service details
API_GW_PORT=$(echo "$API_GW_INFO" | jq -r '.ServicePort')
API_GW_ADDRESS=$(echo "$API_GW_INFO" | jq -r '.ServiceAddress')
API_GW_HEALTH=$(curl -s http://localhost:8500/v1/health/service/openusp-api-gateway | jq -r '.[0].Checks[1].Status' 2>/dev/null || echo "unknown")

echo "📍 API Gateway found:"
echo "   Address: $API_GW_ADDRESS:$API_GW_PORT"
echo "   Consul Health: $API_GW_HEALTH"

# Test the health endpoint
API_GW_URL="http://localhost:$API_GW_PORT"
echo ""
echo "🏥 Testing health endpoint..."
echo "   URL: ${API_GW_URL}/health"

HEALTH_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" "${API_GW_URL}/health" 2>/dev/null)
HTTP_CODE=$(echo "$HEALTH_RESPONSE" | tail -n1 | sed 's/HTTP_CODE://')
RESPONSE_BODY=$(echo "$HEALTH_RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ Health endpoint is working!"
    echo ""
    echo "📊 Health Status:"
    echo "$RESPONSE_BODY" | jq . 2>/dev/null || echo "$RESPONSE_BODY"
elif [ "$HTTP_CODE" = "503" ]; then
    echo "⚠️  API Gateway is unhealthy (HTTP $HTTP_CODE)"
    echo ""
    echo "🔍 Error Details:"
    echo "$RESPONSE_BODY" | jq . 2>/dev/null || echo "$RESPONSE_BODY"
elif [ -z "$HTTP_CODE" ]; then
    echo "❌ Could not connect to API Gateway"
    echo "💡 Check if the service is running: make ou-status"
else
    echo "❌ Health endpoint returned HTTP $HTTP_CODE"
    echo "🔍 Response:"
    echo "$RESPONSE_BODY"
fi

echo ""
echo "🔗 Available Endpoints:"
echo "   Health: ${API_GW_URL}/health"
echo "   Status: ${API_GW_URL}/status"
echo "   Metrics: ${API_GW_URL}/metrics"
echo "   Swagger UI: ${API_GW_URL}/swagger/index.html"
echo "   API Base: ${API_GW_URL}/api/v1"

echo ""
echo "💡 Tip: API Gateway uses dynamic ports with Consul service discovery"
echo "    Always use 'make ou-status' to find current service ports"