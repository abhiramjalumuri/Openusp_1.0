#!/bin/bash

# OpenUSP Swagger UI Test Script
# Tests Swagger UI accessibility across different platforms

echo "🔍 Testing OpenUSP API Gateway Swagger UI"
echo "=========================================="

API_GATEWAY_PORT=${1:-6500}
API_GATEWAY_HOST=${2:-localhost}

echo "📊 Configuration:"
echo "   Host: $API_GATEWAY_HOST"
echo "   Port: $API_GATEWAY_PORT"
echo ""

# Test service accessibility
echo "🌐 Testing service accessibility..."
if curl -s "http://$API_GATEWAY_HOST:$API_GATEWAY_PORT/health" >/dev/null 2>&1; then
    echo "   ✅ API Gateway is accessible"
else
    echo "   ❌ API Gateway is not accessible at http://$API_GATEWAY_HOST:$API_GATEWAY_PORT"
    echo "   💡 Try: make run-api-gateway"
    exit 1
fi

# Test Swagger JSON endpoint
echo ""
echo "📋 Testing Swagger API definition..."
if curl -s "http://$API_GATEWAY_HOST:$API_GATEWAY_PORT/swagger/doc.json" | jq . >/dev/null 2>&1; then
    echo "   ✅ Swagger JSON is accessible and valid"
else
    echo "   ❌ Swagger JSON endpoint failed"
    echo "   🔧 Checking if endpoint returns data..."
    curl -s "http://$API_GATEWAY_HOST:$API_GATEWAY_PORT/swagger/doc.json" | head -c 200
    echo ""
fi

# Test Swagger UI
echo ""
echo "📱 Testing Swagger UI..."
if curl -s "http://$API_GATEWAY_HOST:$API_GATEWAY_PORT/swagger/index.html" | grep -q "Swagger UI" 2>/dev/null; then
    echo "   ✅ Swagger UI is accessible"
else
    echo "   ❌ Swagger UI endpoint failed"
fi

echo ""
echo "🎯 Access URLs:"
echo "   • API Gateway:  http://$API_GATEWAY_HOST:$API_GATEWAY_PORT"
echo "   • Health Check: http://$API_GATEWAY_HOST:$API_GATEWAY_PORT/health"
echo "   • Swagger UI:   http://$API_GATEWAY_HOST:$API_GATEWAY_PORT/swagger/index.html"
echo "   • API Docs:     http://$API_GATEWAY_HOST:$API_GATEWAY_PORT/swagger/doc.json"

echo ""
echo "💡 Platform-specific tips:"
echo "   • macOS/Windows: Use localhost"
echo "   • Linux (Docker): Use host IP or container access"
echo "   • Cross-platform: Access via the host where API Gateway is running"

echo ""
echo "🔧 If Swagger UI shows 'Failed to load API definition':"
echo "   1. Verify the API Gateway is running: curl http://$API_GATEWAY_HOST:$API_GATEWAY_PORT/health"
echo "   2. Check the API definition: curl http://$API_GATEWAY_HOST:$API_GATEWAY_PORT/swagger/doc.json"
echo "   3. Clear browser cache and refresh"
echo "   4. Try accessing from the same host where API Gateway is running"