#!/bin/bash

# Grafana Configuration Setup Script for OpenUSP Platform
# This script configures Grafana with dashboards, data sources, and settings

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GRAFANA_URL="http://localhost:3000"
GRAFANA_USER="admin"
GRAFANA_PASS="admin"

echo "🔧 OpenUSP Grafana Configuration Setup"
echo "======================================="

# Function to wait for Grafana to be ready
wait_for_grafana() {
    echo "⏳ Waiting for Grafana to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "${GRAFANA_URL}/api/health" > /dev/null 2>&1; then
            echo "✅ Grafana is ready!"
            return 0
        fi
        echo "   Attempt $attempt/$max_attempts - waiting..."
        sleep 2
        ((attempt++))
    done
    
    echo "❌ Grafana failed to become ready after $max_attempts attempts"
    return 1
}

# Function to configure Prometheus data source
configure_datasource() {
    echo "📊 Configuring Prometheus data source..."
    
    local datasource_config='{
        "name": "Prometheus",
        "type": "prometheus",
        "url": "http://host.docker.internal:9090",
        "access": "proxy",
        "isDefault": true,
        "basicAuth": false,
        "jsonData": {
            "httpMethod": "POST",
            "queryTimeout": "60s",
            "timeInterval": "15s"
        }
    }'
    
    # Check if data source already exists
    if curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" \
        "${GRAFANA_URL}/api/datasources/name/Prometheus" > /dev/null 2>&1; then
        echo "   ⚠️  Prometheus data source already exists, updating..."
        local ds_id=$(curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" \
            "${GRAFANA_URL}/api/datasources/name/Prometheus" | jq -r '.id')
        curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" \
            -X PUT \
            -H "Content-Type: application/json" \
            -d "$datasource_config" \
            "${GRAFANA_URL}/api/datasources/${ds_id}" > /dev/null
    else
        echo "   ➕ Creating new Prometheus data source..."
        curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" \
            -X POST \
            -H "Content-Type: application/json" \
            -d "$datasource_config" \
            "${GRAFANA_URL}/api/datasources" > /dev/null
    fi
    
    # Clean up any duplicate data sources
    local ds_list=$(curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" "${GRAFANA_URL}/api/datasources")
    local duplicate_count=$(echo "$ds_list" | jq '[.[] | select(.name | contains("Prometheus"))] | length')
    if [ "$duplicate_count" -gt 1 ]; then
        echo "   🧹 Cleaning up duplicate data sources..."
        echo "$ds_list" | jq -r '.[] | select(.name | contains("Prometheus-")) | .id' | while read ds_id; do
            curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" -X DELETE "${GRAFANA_URL}/api/datasources/${ds_id}" > /dev/null
        done
    fi
    
    echo "   ✅ Prometheus data source configured"
}

# Function to create folder for OpenUSP dashboards
create_folder() {
    echo "📁 Creating OpenUSP folder..."
    
    local folder_config='{
        "title": "OpenUSP Platform",
        "tags": ["openusp", "tr-369", "tr-069"]
    }'
    
    # Check if folder already exists
    if curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" \
        "${GRAFANA_URL}/api/folders/openusp" > /dev/null 2>&1; then
        echo "   ⚠️  OpenUSP folder already exists"
    else
        curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" \
            -X POST \
            -H "Content-Type: application/json" \
            -d "$folder_config" \
            "${GRAFANA_URL}/api/folders" > /dev/null
        echo "   ✅ OpenUSP folder created"
    fi
}

# Function to import dashboard
import_dashboard() {
    local dashboard_file=$1
    local dashboard_name=$(basename "$dashboard_file" .json)
    
    echo "   📊 Importing $dashboard_name..."
    
    # Read dashboard JSON and prepare for import
    local dashboard_json=$(cat "$dashboard_file")
    local import_payload=$(jq -n \
        --argjson dashboard "$dashboard_json" \
        '{
            dashboard: $dashboard,
            folderId: null,
            overwrite: true,
            inputs: [],
            message: "Imported via setup script"
        }')
    
    local result=$(curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$import_payload" \
        "${GRAFANA_URL}/api/dashboards/db")
    
    if echo "$result" | jq -e '.status == "success"' > /dev/null 2>&1; then
        local dashboard_url=$(echo "$result" | jq -r '.url')
        echo "      ✅ Successfully imported: ${GRAFANA_URL}${dashboard_url}"
    else
        echo "      ❌ Failed to import $dashboard_name"
        echo "         Error: $(echo "$result" | jq -r '.message // .error // "Unknown error"')"
        return 1
    fi
}

# Function to import all dashboards
import_dashboards() {
    echo "📊 Importing OpenUSP dashboards..."
    
    local dashboards_dir="${PROJECT_ROOT}/configs/grafana-dashboards"
    
    if [ ! -d "$dashboards_dir" ]; then
        echo "❌ Dashboards directory not found: $dashboards_dir"
        return 1
    fi
    
    local dashboard_count=0
    local success_count=0
    
    for dashboard_file in "$dashboards_dir"/*.json; do
        if [ -f "$dashboard_file" ]; then
            ((dashboard_count++))
            if import_dashboard "$dashboard_file"; then
                ((success_count++))
            fi
        fi
    done
    
    echo "   📊 Imported $success_count/$dashboard_count dashboards successfully"
}

# Function to set up Grafana preferences
configure_preferences() {
    echo "⚙️  Configuring Grafana preferences..."
    
    # Set default home dashboard to OpenUSP Overview
    local preferences='{
        "theme": "dark",
        "homeDashboardId": 0,
        "timezone": "browser"
    }'
    
    curl -s -u "${GRAFANA_USER}:${GRAFANA_PASS}" \
        -X PUT \
        -H "Content-Type: application/json" \
        -d "$preferences" \
        "${GRAFANA_URL}/api/org/preferences" > /dev/null
    
    echo "   ✅ Preferences configured"
}

# Main execution
main() {
    echo "🚀 Starting Grafana configuration..."
    echo ""
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        echo "❌ jq is required but not installed. Please install jq first."
        exit 1
    fi
    
    # Wait for Grafana to be ready
    if ! wait_for_grafana; then
        echo "❌ Cannot proceed without Grafana being ready"
        exit 1
    fi
    
    # Configure components
    configure_datasource
    create_folder
    import_dashboards
    configure_preferences
    
    echo ""
    echo "✅ Grafana configuration completed successfully!"
    echo ""
    echo "🌐 Access Grafana at: ${GRAFANA_URL}"
    echo "   Username: ${GRAFANA_USER}"
    echo "   Password: ${GRAFANA_PASS}"
    echo ""
    echo "📊 Available Dashboards:"
    echo "   • OpenUSP Platform Overview - Service health and performance"
    echo "   • TR-369 USP Protocol Metrics - USP message processing and operations"
    echo "   • TR-069 CWMP Protocol Metrics - CWMP sessions and RPC methods"
    echo "   • Data Service & Database Metrics - Database performance and gRPC operations"
    echo ""
    echo "🔗 Direct Links:"
    echo "   Platform Overview: ${GRAFANA_URL}/d/openusp-overview"
    echo "   USP Protocol: ${GRAFANA_URL}/d/openusp-usp-protocol"
    echo "   CWMP Protocol: ${GRAFANA_URL}/d/openusp-cwmp-protocol"
    echo "   Data Service: ${GRAFANA_URL}/d/openusp-data-service"
}

# Run main function
main "$@"