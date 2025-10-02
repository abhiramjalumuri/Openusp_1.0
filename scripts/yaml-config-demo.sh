#!/bin/bash

# YAML Configuration System Demo for OpenUSP Agents
# =================================================

echo "🎯 OpenUSP YAML Configuration System Demo"
echo "=========================================="
echo ""

# Change to project directory
cd "$(dirname "$0")/.."

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}1. Building agents with YAML configuration support...${NC}"
echo ""

# Build agents
echo "Building TR-369 agent..."
go build -o build/tr369-agent examples/tr369-agent/main.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ TR-369 agent built successfully${NC}"
else
    echo "❌ Failed to build TR-369 agent"
    exit 1
fi

echo ""
echo -e "${BLUE}2. Testing YAML Configuration Validation...${NC}"
echo ""

# Test main configuration files
echo "🔍 Validating main TR-369 configuration:"
./build/tr369-agent --config configs/tr369-agent.yaml --validate-only
echo ""

# Test example configurations
echo "🔍 Validating TR-369 Consul-enabled configuration:"
./build/tr369-agent --config configs/examples/yaml/tr369-consul-enabled.yaml --validate-only
echo ""

echo "🔍 Validating TR-369 standalone configuration:"
./build/tr369-agent --config configs/examples/yaml/tr369-consul-disabled.yaml --validate-only
echo ""

echo -e "${BLUE}3. Configuration File Structure Demo...${NC}"
echo ""

echo "📁 Available YAML configuration files:"
echo ""
echo "Main configurations:"
ls -la configs/*.yaml 2>/dev/null || echo "  No main YAML configs found"
echo ""

echo "Example configurations:"
ls -la configs/examples/yaml/*.yaml 2>/dev/null || echo "  No example YAML configs found"
echo ""

echo -e "${BLUE}4. YAML vs ENV Configuration Demo...${NC}"
echo ""

echo "🆚 Comparing YAML vs Environment-based configuration:"
echo ""

echo "YAML Configuration (hierarchical):"
echo "-----------------------------------"
cat configs/tr369-agent.yaml | head -20
echo "..."
echo ""

echo "Environment Configuration (flat):"
echo "---------------------------------"
if [ -f configs/tr369-agent.env ]; then
    cat configs/tr369-agent.env | head -20
    echo "..."
else
    echo "  tr369-agent.env not found (YAML preferred)"
fi
echo ""

echo -e "${BLUE}5. Configuration Loading Priority Demo...${NC}"
echo ""

echo "🔄 Testing configuration loading priority:"
echo ""

echo "1. Direct YAML file (highest priority):"
./build/tr369-agent --config configs/tr369-agent.yaml --validate-only | head -1
echo ""

echo "2. Automatic detection (YAML preferred over ENV):"
./build/tr369-agent --validate-only | head -1
echo ""

echo -e "${BLUE}6. Dynamic Service Discovery Demo...${NC}"
echo ""

echo "🕵️  Checking Consul service registration:"
if curl -s "http://localhost:8500/v1/catalog/services" >/dev/null 2>&1; then
    echo "✅ Consul is available"
    
    # Check for registered services
    echo ""
    echo "Registered OpenUSP services:"
    curl -s "http://localhost:8500/v1/catalog/services" | jq -r 'to_entries[] | select(.key | startswith("openusp-")) | "  \(.key): \(.value | join(", "))"' 2>/dev/null || echo "  (jq not available - raw output)"
    
    echo ""
    echo "🔍 Service discovery test:"
    
    # Get MTP service port
    MTP_PORT=$(curl -s "http://localhost:8500/v1/catalog/service/openusp-mtp-service" | jq -r '.[0].ServicePort // empty' 2>/dev/null)
    API_PORT=$(curl -s "http://localhost:8500/v1/catalog/service/openusp-api-gateway" | jq -r '.[0].ServicePort // empty' 2>/dev/null)
    
    if [ -n "$MTP_PORT" ] && [ "$MTP_PORT" != "null" ]; then
        echo "  MTP Service discovered at port: $MTP_PORT"
    else
        echo "  MTP Service: Not registered or not available"
    fi
    
    if [ -n "$API_PORT" ] && [ "$API_PORT" != "null" ]; then
        echo "  API Gateway discovered at port: $API_PORT"
    else
        echo "  API Gateway: Not registered or not available"
    fi
    
    echo ""
    echo "🚀 Testing dynamic service discovery with Consul-enabled config:"
    echo "(Running for 8 seconds...)"
    timeout 8s ./build/tr369-agent --config configs/examples/yaml/tr369-consul-enabled.yaml 2>&1 | grep -E "(Discovering|Discovered|Registered|Connected)" || echo "  No services available for discovery"
    
else
    echo "❌ Consul not available (expected if not running)"
    echo "   Start with: make infra-up"
fi

echo ""

echo -e "${BLUE}7. Advanced Configuration Features...${NC}"
echo ""

echo "🎛️  YAML Configuration Features:"
echo ""
echo "✅ Hierarchical structure with nested sections"
echo "✅ Type-safe parsing (strings, integers, booleans, durations)"
echo "✅ Comment support for documentation"
echo "✅ Array and object support"
echo "✅ Environment-specific configurations"
echo "✅ Automatic validation and error reporting"
echo "✅ Backward compatibility with ENV files"
echo ""

echo -e "${BLUE}8. Usage Examples...${NC}"
echo ""

echo "📖 Common usage patterns:"
echo ""
echo "# Use default configuration (auto-detection):"
echo "./build/tr369-agent"
echo ""
echo "# Use specific YAML configuration:"
echo "./build/tr369-agent --config configs/tr369-agent.yaml"
echo ""
echo "# Use example configuration:"
echo "./build/tr369-agent --config configs/examples/yaml/tr369-consul-enabled.yaml"
echo ""
echo "# Validate configuration only:"
echo "./build/tr369-agent --config my-config.yaml --validate-only"
echo ""
echo "# Override USP version:"
echo "./build/tr369-agent --config configs/tr369-agent.yaml --version 1.3"
echo ""

echo -e "${GREEN}🎉 YAML Configuration System Demo Complete!${NC}"
echo ""
echo "📚 Documentation:"
echo "  - configs/README-YAML.md          # Complete YAML configuration guide"
echo "  - configs/tr369-agent.yaml        # Main TR-369 configuration template"
echo "  - configs/tr069-agent.yaml        # Main TR-069 configuration template"
echo "  - configs/examples/yaml/          # Example configurations directory"
echo ""
echo "🚀 Next Steps:"
echo "  1. Copy example configurations as templates"
echo "  2. Customize for your deployment environment"
echo "  3. Use --validate-only to test configurations"
echo "  4. Deploy with hierarchical YAML for better maintainability"
echo ""