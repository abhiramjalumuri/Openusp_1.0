# OpenUSP Configuration Guide

This directory contains configuration files for OpenUSP services and protocol agents.

## Configuration Files Overview

### Service Configuration
- `openusp.env` - Main environment configuration for all OpenUSP services
- `version.env` - Version information and build metadata

### Agent Configuration (YAML-based)
- `usp-agent.yaml` - TR-369 USP agent configuration (replaces tr369-agent.env)
- `cwmp-agent.yaml` - TR-069 CWMP agent configuration (replaces tr069-agent.env)

### Infrastructure Configuration
- `init-db.sql` - Database initialization script
- `mosquitto.conf` - MQTT broker configuration
- `prometheus-dev.yml` - Prometheus monitoring configuration
- `grafana-*.yml` - Grafana dashboard and data source configuration

## Quick Start

### TR-369 USP Agent (YAML Configuration)

```bash
# Use default configuration
make start-usp-agent

# Or customize and run manually
cp configs/usp-agent.yaml configs/my-usp-agent.yaml
# Edit device-specific settings
nano configs/my-usp-agent.yaml
# Run with custom config
./build/usp-agent --config configs/my-usp-agent.yaml

# Validate configuration only
./build/usp-agent --config configs/my-usp-agent.yaml --validate-only
```

### TR-069 CWMP Agent (YAML Configuration)

```bash
# Use default configuration  
make start-cwmp-agent

# Or customize and run manually
cp configs/cwmp-agent.yaml configs/my-cwmp-agent.yaml
# Edit device-specific settings
nano configs/my-cwmp-agent.yaml
# Run with custom config
./build/cwmp-agent --config configs/my-cwmp-agent.yaml
```

## YAML Configuration Structure

### USP Agent Configuration (`usp-agent.yaml`)

```yaml
# Device identification
device_info:
  endpoint_id: "proto://usp-agent-001"
  manufacturer: "OpenUSP"
  model_name: "USP Gateway"
  serial_number: "USP-001-2024"
  software_version: "1.1.0"
  hardware_version: "1.0"
  device_type: "gateway"
  oui: "00D04F"

# USP protocol settings
usp_protocol:
  version: "1.3"                        # USP version (1.3 or 1.4)
  supported_versions: ["1.3", "1.4"]   # Supported versions
  agent_role: "device"
  command_key: "usp-agent-001"

# Message Transport Protocol
mtp:
  type: "websocket"                     # websocket, mqtt, stomp, uds
  
  websocket:
    url: "ws://localhost:8081/ws"       # WebSocket endpoint
    ping_interval: "30s"
    reconnect_interval: "5s"
    max_reconnect_attempts: 10
    
  mqtt:
    broker_url: "tcp://localhost:1883"
    client_id: "usp-agent-001"
    topic_request: "usp/agent/request"
    topic_response: "usp/agent/response"
    
# Platform integration
platform:
  url: "http://localhost:6500"          # API Gateway URL
  auto_register: true                   # Auto-register device
  heartbeat_interval: "60s"

# Agent behavior
agent_behavior:
  periodic_inform_enabled: true
  periodic_inform_interval: "300s"
  parameter_update_notification: true
  event_notification: true

# Security (optional)
security:
  tls_enabled: false
  tls_cert_file: ""
  tls_key_file: ""
  
# Logging
logging:
  level: "info"                         # debug, info, warn, error
  format: "json"                        # json, text
  output: "stdout"                      # stdout, file
```

### CWMP Agent Configuration (`cwmp-agent.yaml`)

```yaml  
# Device identification
device_info:
  endpoint_id: "cwmp://cwmp-agent-001"
  manufacturer: "OpenUSP"
  model_name: "CWMP Gateway"
  serial_number: "CWMP-001-2024"
  oui: "00D04F"

# CWMP protocol settings
cwmp_protocol:
  version: "1.4"                        # CWMP version
  supported_versions: ["1.0", "1.1", "1.2", "1.3", "1.4"]
  agent_role: "cpe"

# ACS (Auto Configuration Server)
acs:
  url: "http://localhost:7547"          # Standard TR-069 port
  username: "acs"
  password: "acs123"
  periodic_inform_enabled: true
  periodic_inform_interval: "300s"

# Connection Request (for ACS-initiated sessions)
connection_request:
  url: "http://localhost:9999/cr"
  username: "cr"
  password: "cr123"

# Platform integration
platform:
  url: "http://localhost:6500"          # API Gateway URL
  auto_register: true

# SOAP/XML settings
soap_xml:
  soap_version: "1.1"
  strict_parsing: false
  pretty_print: false

# Session management
session:
  timeout: "300s"
  max_concurrent: 5
  keep_alive: true
```
CONSUL_API_GATEWAY_SERVICE_NAME=openusp-api-gateway
```

#### Consul Disabled (Static Configuration)
When `CONSUL_ENABLED=false`:
- All service endpoints must be explicitly configured
- No automatic service discovery
- Suitable for production deployments and remote devices
- Lower dependency footprint

Configuration:
```bash
CONSUL_ENABLED=false
OPENUSP_PLATFORM_URL=https://usp-platform.company.com
WEBSOCKET_URL=wss://mtp.company.com/ws
ACS_URL=https://acs.company.com/cwmp
```

### 6. Security Configuration
```bash
TLS_ENABLED=true                       # Enable TLS/SSL
TLS_CERT_FILE=/path/to/cert.crt       # Client certificate file
TLS_KEY_FILE=/path/to/key.key         # Private key file
TLS_CA_FILE=/path/to/ca.crt           # Certificate Authority file
TLS_SKIP_VERIFY=false                 # Skip certificate verification (dev only)
```

### 7. Authentication (TR-069)
```bash
DIGEST_AUTH_ENABLED=true              # Enable HTTP Digest authentication
BASIC_AUTH_ENABLED=true               # Enable HTTP Basic authentication
AUTH_REALM=CWMP                       # Authentication realm
AUTH_ALGORITHM=MD5                    # Digest algorithm (MD5, SHA-256)
```

### 8. Performance Tuning
```bash
BUFFER_SIZE=8192                      # I/O buffer size
MAX_MESSAGE_SIZE=2097152              # Maximum message size (2MB)
CONNECTION_TIMEOUT=30s                # Connection timeout
READ_TIMEOUT=60s                      # Read operation timeout
WRITE_TIMEOUT=60s                     # Write operation timeout
```

### 9. Logging Configuration
```bash
LOG_LEVEL=info                        # Log level (debug, info, warn, error)
LOG_FORMAT=json                       # Log format (json, text)
LOG_OUTPUT=stdout                     # Log output (stdout, file)
LOG_FILE=logs/agent.log               # Log file path (when LOG_OUTPUT=file)
```

## Environment-Specific Configurations

### Development Environment
```bash
DEVELOPMENT_MODE=true
CONSUL_ENABLED=true
LOG_LEVEL=debug
VERBOSE_LOGGING=true
TLS_SKIP_VERIFY=true
```

### Production Environment
```bash
DEVELOPMENT_MODE=false
CONSUL_ENABLED=false  # or true depending on infrastructure
LOG_LEVEL=warn
VERBOSE_LOGGING=false
TLS_ENABLED=true
TLS_SKIP_VERIFY=false
```

### Testing Environment
```bash
TEST_MODE=true
MOCK_DATA_ENABLED=true
LOG_LEVEL=debug
SIMULATE_DEVICE_RESPONSES=true
```

## Usage Patterns

### 1. Load Configuration from File
```go
// TR-369 Agent
config, err := config.LoadConfigFromYAML("tr369", "configs/")
if err != nil {
    log.Fatal("Failed to load configuration:", err)
}

// TR-069 Agent
config, err := config.LoadConfigFromYAML("tr069", "configs/")
if err != nil {
    log.Fatal("Failed to load configuration:", err)
}
```

### 2. Override with Environment Variables
Environment variables take precedence over configuration file values:
```bash
export USP_VERSION=1.3
export LOG_LEVEL=debug
./build/tr369-agent --config configs/my-tr369-agent.env
```

### 3. Consul Service Discovery
When Consul is enabled, the agent will:
1. Register itself with Consul for health monitoring
2. Discover OpenUSP platform services automatically
3. Use fallback static configuration if service discovery fails
4. Update endpoints dynamically when services change

### 4. Multiple Agent Deployment
Run multiple agents with different configurations:
```bash
# Gateway device
./build/tr369-agent --config configs/gateway-agent.env &

# Set-top box device  
./build/tr369-agent --config configs/stb-agent.env &

# CWMP legacy device
./build/tr069-agent --config configs/legacy-device.env &
```

## Best Practices

### 1. Security
- Always use TLS in production environments
- Store sensitive credentials in secure credential stores
- Use strong authentication mechanisms
- Regularly rotate certificates and passwords

### 2. Monitoring
- Enable health checks for service monitoring
- Configure appropriate log levels for different environments
- Use structured logging (JSON format) for log aggregation
- Monitor agent connectivity and message processing metrics

### 3. Performance
- Tune buffer sizes based on expected message volumes
- Configure appropriate timeout values for network conditions
- Use connection pooling for high-throughput scenarios
- Monitor memory usage and connection counts

### 4. Deployment
- Use Consul service discovery for dynamic environments
- Use static configuration for production/remote deployments
- Implement graceful shutdown handling
- Configure automatic restart on failure

## Troubleshooting

### Common Issues

1. **Service Discovery Failures**
   - Check Consul connectivity and service registration
   - Verify service names and tags in configuration
   - Use static fallback configuration as backup

2. **Authentication Errors**
   - Verify username/password credentials
   - Check certificate validity and paths
   - Ensure proper authentication method configuration

3. **Connection Issues**
   - Verify network connectivity to endpoints
   - Check firewall rules and port accessibility
   - Validate TLS certificate chain

4. **Protocol Errors**
   - Ensure compatible protocol versions
   - Verify message format and encoding
   - Check endpoint URL formats and paths

### Debug Configuration
Enable detailed logging for troubleshooting:
```bash
LOG_LEVEL=debug
VERBOSE_LOGGING=true
LOG_SOAP_MESSAGES=true    # TR-069 only
LOG_HTTP_REQUESTS=true    # TR-069 only
```

This comprehensive configuration system provides flexibility for various deployment scenarios while maintaining compatibility with the OpenUSP platform architecture.