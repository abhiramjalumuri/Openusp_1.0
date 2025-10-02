# Protocol Agent Configuration Guide

This directory contains comprehensive configuration files for TR-369 USP and TR-069 CWMP agents with support for both Consul-enabled and standalone deployments.

## Configuration Files Overview

### Base Configuration Files
- `tr369-agent.env` - Default TR-369 USP agent configuration with all available options
- `tr069-agent.env` - Default TR-069 CWMP agent configuration with all available options

### Example Configuration Files
- `examples/tr369-consul-enabled.env` - Production TR-369 setup with Consul service discovery
- `examples/tr369-consul-disabled.env` - Standalone TR-369 setup with static configuration
- `examples/tr069-consul-enabled.env` - Production TR-069 setup with Consul service discovery  
- `examples/tr069-consul-disabled.env` - Standalone TR-069 setup with static configuration

## Quick Start

### TR-369 USP Agent

#### With Consul Service Discovery (Recommended for Development)
```bash
# Copy example configuration
cp configs/examples/tr369-consul-enabled.env configs/my-tr369-agent.env

# Edit device-specific settings
nano configs/my-tr369-agent.env

# Run the agent
./build/tr369-agent --config configs/my-tr369-agent.env
```

#### Standalone Deployment (Production/Remote Devices)
```bash
# Copy standalone configuration
cp configs/examples/tr369-consul-disabled.env configs/my-tr369-agent.env

# Configure static endpoints and device information
nano configs/my-tr369-agent.env

# Run the agent
./build/tr369-agent --config configs/my-tr369-agent.env
```

### TR-069 CWMP Agent

#### With Consul Service Discovery (Recommended for Development)
```bash
# Copy example configuration
cp configs/examples/tr069-consul-enabled.env configs/my-tr069-agent.env

# Edit device-specific settings
nano configs/my-tr069-agent.env

# Run the agent
./build/tr069-agent --config configs/my-tr069-agent.env
```

#### Standalone Deployment (Production/Remote Devices)
```bash
# Copy standalone configuration
cp configs/examples/tr069-consul-disabled.env configs/my-tr069-agent.env

# Configure static endpoints and device information
nano configs/my-tr069-agent.env

# Run the agent
./build/tr069-agent --config configs/my-tr069-agent.env
```

## Configuration Sections

### 1. Service Discovery
Configure Consul integration for automatic service endpoint discovery:
```bash
CONSUL_ENABLED=true                    # Enable/disable Consul integration
CONSUL_ADDR=localhost:8500             # Consul server address
CONSUL_DATACENTER=openusp-dev          # Consul datacenter
```

### 2. Device Information
Define device identity and capabilities:
```bash
TR369_ENDPOINT_ID=proto://my-device-001    # Unique device identifier
TR369_MANUFACTURER=ACME Corp               # Device manufacturer
TR369_MODEL_NAME=ACME-Gateway-Pro          # Device model
TR369_SERIAL_NUMBER=ACME-001234            # Device serial number
TR369_SOFTWARE_VERSION=1.1.0              # Current software version
```

### 3. Protocol Configuration

#### TR-369 USP Protocol
```bash
USP_VERSION=1.4                        # USP protocol version (1.3 or 1.4)
USP_SUPPORTED_VERSIONS=1.3,1.4         # Supported USP versions
MTP_TYPE=websocket                     # Message Transport Protocol type
```

#### TR-069 CWMP Protocol
```bash
CWMP_VERSION=1.4                       # CWMP protocol version
CWMP_SUPPORTED_VERSIONS=1.0,1.1,1.2,1.3,1.4  # Supported CWMP versions
ACS_URL=http://localhost:7547          # Auto Configuration Server URL
```

### 4. Message Transport Protocols (MTP)

#### WebSocket MTP (TR-369)
```bash
MTP_TYPE=websocket
WEBSOCKET_URL=ws://localhost:8081/ws   # WebSocket endpoint URL
WEBSOCKET_PING_INTERVAL=30s           # WebSocket ping interval
WEBSOCKET_RECONNECT_INTERVAL=5s       # Reconnection interval on failure
```

#### MQTT MTP (TR-369)
```bash
MTP_TYPE=mqtt
MQTT_BROKER_URL=tcp://localhost:1883  # MQTT broker URL
MQTT_CLIENT_ID=usp-agent-001          # MQTT client identifier
MQTT_USERNAME=usp                     # MQTT authentication username
MQTT_PASSWORD=usp123                  # MQTT authentication password
```

#### STOMP MTP (TR-369)
```bash
MTP_TYPE=stomp
STOMP_BROKER_URL=tcp://localhost:61613       # STOMP broker URL
STOMP_DESTINATION_REQUEST=/queue/usp.request # STOMP request destination
STOMP_USERNAME=usp                           # STOMP authentication username
```

### 5. Consul Integration Scenarios

#### Consul Enabled (Automatic Service Discovery)
When `CONSUL_ENABLED=true`:
- Agent automatically discovers OpenUSP platform services
- MTP service endpoints are resolved dynamically
- API Gateway endpoints are discovered automatically
- Health checks are registered with Consul
- Service tags enable environment-specific discovery

Configuration:
```bash
CONSUL_ENABLED=true
CONSUL_SERVICE_DISCOVERY_ENABLED=true
CONSUL_MTP_SERVICE_NAME=openusp-mtp-service
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
config, err := config.LoadTR369Config("configs/my-tr369-agent.env")
if err != nil {
    log.Fatal("Failed to load configuration:", err)
}

// TR-069 Agent
config, err := config.LoadTR069Config("configs/my-tr069-agent.env")
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