# ⚙️ Configuration Guide

Complete configuration reference for OpenUSP services.

## Table of Contents

- [Environment Variables](#environment-variables)
- [Configuration Files](#configuration-files)
- [Service-Specific Configuration](#service-specific-configuration)
- [Database Configuration](#database-configuration)
- [Message Transport Configuration](#message-transport-configuration)
- [Monitoring Configuration](#monitoring-configuration)
- [Security Configuration](#security-configuration)

## Environment Variables

OpenUSP uses environment variables for runtime configuration. All variables are prefixed with `OPENUSP_`.

### Core Configuration

```bash
# Service Discovery
CONSUL_ENABLED=true               # Enable Consul service discovery (default: true for dev)
CONSUL_ADDRESS=localhost:8500     # Consul server address
CONSUL_DATACENTER=dc1            # Consul datacenter
CONSUL_TOKEN=                     # Consul ACL token

# Database
OPENUSP_DATABASE_HOST=localhost   # PostgreSQL host
OPENUSP_DATABASE_PORT=5432        # PostgreSQL port
OPENUSP_DATABASE_NAME=openusp     # Database name
OPENUSP_DATABASE_USER=openusp     # Database user
OPENUSP_DATABASE_PASSWORD=openusp123  # Database password
OPENUSP_DATABASE_SSLMODE=disable  # SSL mode (disable, require, verify-ca, verify-full)

# Logging
OPENUSP_LOG_LEVEL=info           # Log level (debug, info, warn, error)
OPENUSP_LOG_FORMAT=json          # Log format (json, text)
OPENUSP_LOG_FILE=                # Log file path (empty for stdout)

# Metrics
OPENUSP_METRICS_ENABLED=true     # Enable Prometheus metrics
OPENUSP_METRICS_PORT=9090        # Metrics server port
```

### Service-Specific Variables

#### API Gateway
```bash
OPENUSP_API_GATEWAY_PORT=8080           # HTTP server port
OPENUSP_API_GATEWAY_HOST=0.0.0.0        # Bind address
OPENUSP_API_GATEWAY_CORS_ENABLED=true   # Enable CORS
OPENUSP_API_GATEWAY_AUTH_ENABLED=false  # Enable authentication
OPENUSP_API_GATEWAY_RATE_LIMIT=1000     # Requests per minute
OPENUSP_API_GATEWAY_DATA_SERVICE_ADDRESS=localhost:9092  # Data service gRPC address
```

#### Data Service
```bash
OPENUSP_DATA_SERVICE_PORT=9092          # gRPC server port
OPENUSP_DATA_SERVICE_HOST=0.0.0.0       # Bind address
OPENUSP_DATA_SERVICE_MAX_CONNECTIONS=100 # Max database connections
OPENUSP_DATA_SERVICE_TIMEOUT=30s        # Request timeout
```

#### MTP Service
```bash
OPENUSP_MTP_SERVICE_PORT=8081           # HTTP server port
OPENUSP_MTP_SERVICE_HOST=0.0.0.0        # Bind address
OPENUSP_MTP_SERVICE_WEBSOCKET_ENABLED=true  # Enable WebSocket MTP
OPENUSP_MTP_SERVICE_MQTT_ENABLED=true   # Enable MQTT MTP
OPENUSP_MTP_SERVICE_STOMP_ENABLED=true  # Enable STOMP MTP
OPENUSP_MTP_SERVICE_UDS_ENABLED=true    # Enable Unix Domain Socket MTP

# MQTT Configuration
OPENUSP_MQTT_BROKER_HOST=localhost      # MQTT broker host
OPENUSP_MQTT_BROKER_PORT=1883           # MQTT broker port
OPENUSP_MQTT_CLIENT_ID=openusp-mtp      # MQTT client ID
OPENUSP_MQTT_USERNAME=                  # MQTT username
OPENUSP_MQTT_PASSWORD=                  # MQTT password
OPENUSP_MQTT_TOPIC_PREFIX=usp           # Topic prefix

# STOMP Configuration
OPENUSP_STOMP_BROKER_HOST=localhost     # STOMP broker host
OPENUSP_STOMP_BROKER_PORT=61613         # STOMP broker port
OPENUSP_STOMP_USERNAME=guest            # STOMP username
OPENUSP_STOMP_PASSWORD=guest            # STOMP password
OPENUSP_STOMP_DESTINATION=/queue/usp    # STOMP destination

# Unix Domain Socket Configuration
OPENUSP_UDS_SOCKET_PATH=/tmp/openusp.sock  # Socket file path
```

#### USP Service
```bash
OPENUSP_USP_SERVICE_PORT=8082           # HTTP server port (if HTTP enabled)
OPENUSP_USP_SERVICE_HOST=0.0.0.0        # Bind address
OPENUSP_USP_CONTROLLER_ID=controller-001  # USP Controller ID
OPENUSP_USP_AGENT_CERT_PATH=            # Agent certificate path
OPENUSP_USP_CONTROLLER_CERT_PATH=       # Controller certificate path
```

#### CWMP Service
```bash
OPENUSP_CWMP_SERVICE_PORT=7547          # HTTP server port (TR-069 standard)
OPENUSP_CWMP_SERVICE_HOST=0.0.0.0       # Bind address
OPENUSP_CWMP_AUTH_USERNAME=acs          # Basic auth username
OPENUSP_CWMP_AUTH_PASSWORD=acs123       # Basic auth password
OPENUSP_CWMP_SESSION_TIMEOUT=300        # Session timeout (seconds)
OPENUSP_CWMP_MAX_ENVELOPE_SIZE=65536    # Max SOAP envelope size
```

### Agent-Specific Variables

#### USP Agent  
```bash
USP_WS_URL=ws://localhost:8081/ws       # Override WebSocket URL
API_GATEWAY_URL=http://localhost:6500   # Override API Gateway URL
USP_VERSION=1.3                         # Override USP protocol version
USP_ENDPOINT_ID=proto://custom-agent    # Override endpoint ID
```

#### CWMP Agent
```bash
CWMP_ACS_HOST=localhost                 # CWMP ACS host
CWMP_ACS_PORT=7547                      # CWMP ACS port  
CWMP_USERNAME=acs                       # CWMP authentication username
CWMP_PASSWORD=acs123                    # CWMP authentication password
CWMP_MANUFACTURER=OpenUSP               # Device manufacturer
CWMP_SERIAL_NUMBER=DEMO123456           # Device serial number
```

## Configuration Files

### Main Configuration File

Location: `configs/openusp.env`

### Agent Configuration Files

OpenUSP now provides dedicated YAML configuration files for protocol agents:

#### USP Agent Configuration
Location: `configs/usp-agent.yaml`

```yaml
# TR-369 USP Agent Configuration
device_info:
  endpoint_id: "proto://usp-agent-001"
  manufacturer: "OpenUSP"
  model_name: "USP Gateway"
  serial_number: "USP-001-2024"

usp_protocol:
  version: "1.3"
  supported_versions: ["1.3", "1.4"]

mtp:
  type: "websocket"
  websocket:
    url: "ws://localhost:8081/ws"
    ping_interval: "30s"
    reconnect_interval: "5s"

platform:
  url: "http://localhost:6500"
  auto_register: true
```

#### CWMP Agent Configuration  
Location: `configs/cwmp-agent.yaml`

```yaml
# TR-069 CWMP Agent Configuration
device_info:
  endpoint_id: "cwmp://cwmp-agent-001"
  manufacturer: "OpenUSP"  
  model_name: "CWMP Gateway"
  serial_number: "CWMP-001-2024"

cwmp_protocol:
  version: "1.4"
  supported_versions: ["1.0", "1.1", "1.2", "1.3", "1.4"]

acs:
  url: "http://localhost:7547"
  username: "acs"
  password: "acs123"

platform:
  url: "http://localhost:6500"  
  auto_register: true
```

```bash
# OpenUSP Platform Configuration
# This file contains default values for all services

# Core Settings
CONSUL_ENABLED=true
CONSUL_ADDRESS=localhost:8500
CONSUL_DATACENTER=dc1

# Database Configuration
OPENUSP_DATABASE_HOST=localhost
OPENUSP_DATABASE_PORT=5432
OPENUSP_DATABASE_NAME=openusp
OPENUSP_DATABASE_USER=openusp
OPENUSP_DATABASE_PASSWORD=openusp123
OPENUSP_DATABASE_SSLMODE=disable

# Service Ports (Production)
OPENUSP_API_GATEWAY_PORT=8080
OPENUSP_DATA_SERVICE_PORT=9092
OPENUSP_MTP_SERVICE_PORT=8081
OPENUSP_CWMP_SERVICE_PORT=7547

# Logging
OPENUSP_LOG_LEVEL=info
OPENUSP_LOG_FORMAT=json

# Features
OPENUSP_METRICS_ENABLED=true
OPENUSP_API_GATEWAY_CORS_ENABLED=true
OPENUSP_MTP_SERVICE_WEBSOCKET_ENABLED=true
OPENUSP_MTP_SERVICE_MQTT_ENABLED=true
OPENUSP_MTP_SERVICE_STOMP_ENABLED=true

# Transport Configuration
OPENUSP_MQTT_BROKER_HOST=localhost
OPENUSP_MQTT_BROKER_PORT=1883
OPENUSP_STOMP_BROKER_HOST=localhost
OPENUSP_STOMP_BROKER_PORT=61613
OPENUSP_STOMP_USERNAME=guest
OPENUSP_STOMP_PASSWORD=guest
```

### Development Configuration

Location: `configs/openusp.dev.env`

```bash
# Development Environment Configuration
# Override values for development

# Alternative ports to avoid conflicts
OPENUSP_API_GATEWAY_PORT=8082
OPENUSP_MTP_SERVICE_PORT=8083
OPENUSP_CWMP_SERVICE_PORT=7548
OPENUSP_DATA_SERVICE_PORT=9093

# Database (development instance)
OPENUSP_DATABASE_PORT=5433

# Enhanced logging for development
OPENUSP_LOG_LEVEL=debug
OPENUSP_LOG_FORMAT=text

# Development features
OPENUSP_API_GATEWAY_AUTH_ENABLED=false
OPENUSP_METRICS_ENABLED=true
```

### Docker Compose Configuration

Location: `deployments/docker-compose.infra.yml`

Key services configuration:

```yaml
services:
  postgres:
    environment:
      POSTGRES_DB: openusp
      POSTGRES_USER: openusp
      POSTGRES_PASSWORD: openusp123
    ports:
      - "5432:5432"
  
  rabbitmq:
    ports:
      - "61613:61613"  # STOMP
      - "15672:15672"  # Management UI
  
  mosquitto:
    ports:
      - "1883:1883"    # MQTT
  
  consul:
    ports:
      - "8500:8500"    # HTTP API
      - "8600:8600/udp"  # DNS
```

## Service-Specific Configuration

### API Gateway Configuration

The API Gateway uses configuration from multiple sources:

```go
// Internal configuration structure
type APIGatewayConfig struct {
    Port                int
    Host                string
    CORSEnabled         bool
    AuthEnabled         bool
    RateLimit           int
    DataServiceAddress  string
    TLSEnabled          bool
    CertFile            string
    KeyFile             string
}
```

**Configuration Priority:**
1. Command line flags
2. Environment variables
3. Configuration file
4. Default values

**Example configuration:**
```bash
# API Gateway runs in HTTP-only mode (HTTPS/TLS support removed)
OPENUSP_API_GATEWAY_PORT=6500

# Enable authentication
OPENUSP_API_GATEWAY_AUTH_ENABLED=true
OPENUSP_API_GATEWAY_AUTH_TYPE=jwt
OPENUSP_API_GATEWAY_JWT_SECRET=your-secret-key

# CORS configuration
OPENUSP_API_GATEWAY_CORS_ORIGINS=http://localhost:3000,http://app.example.com
OPENUSP_API_GATEWAY_CORS_METHODS=GET,POST,PUT,DELETE
```

### Data Service Configuration

Database-specific configuration:

```bash
# Connection pooling
OPENUSP_DATA_SERVICE_MAX_IDLE_CONNECTIONS=10
OPENUSP_DATA_SERVICE_MAX_OPEN_CONNECTIONS=100
OPENUSP_DATA_SERVICE_CONNECTION_MAX_LIFETIME=1h

# Query timeouts
OPENUSP_DATA_SERVICE_QUERY_TIMEOUT=30s
OPENUSP_DATA_SERVICE_TRANSACTION_TIMEOUT=60s

# Backup configuration
OPENUSP_DATA_SERVICE_BACKUP_ENABLED=false
OPENUSP_DATA_SERVICE_BACKUP_INTERVAL=24h
OPENUSP_DATA_SERVICE_BACKUP_RETENTION=7d
```

### MTP Service Configuration

Transport-specific settings:

```bash
# WebSocket
OPENUSP_MTP_WEBSOCKET_MAX_CONNECTIONS=1000
OPENUSP_MTP_WEBSOCKET_BUFFER_SIZE=4096
OPENUSP_MTP_WEBSOCKET_PING_INTERVAL=30s

# MQTT
OPENUSP_MQTT_QOS=1
OPENUSP_MQTT_RETAIN=false
OPENUSP_MQTT_CLEAN_SESSION=true
OPENUSP_MQTT_KEEP_ALIVE=60s
OPENUSP_MQTT_CONNECT_TIMEOUT=30s

# STOMP
OPENUSP_STOMP_HEARTBEAT_SEND=10000
OPENUSP_STOMP_HEARTBEAT_RECEIVE=10000
OPENUSP_STOMP_CONNECT_TIMEOUT=30s

# Unix Domain Socket
OPENUSP_UDS_SOCKET_PERMISSIONS=0666
OPENUSP_UDS_BUFFER_SIZE=8192
```

## Database Configuration

### PostgreSQL Setup

```sql
-- Create database and user
CREATE DATABASE openusp;
CREATE USER openusp WITH ENCRYPTED PASSWORD 'openusp123';
GRANT ALL PRIVILEGES ON DATABASE openusp TO openusp;

-- Connect to openusp database
\c openusp

-- Grant schema privileges
GRANT ALL ON SCHEMA public TO openusp;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO openusp;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO openusp;
```

### Connection String Format

```bash
# Standard format
postgresql://username:password@host:port/database?sslmode=disable

# With SSL
postgresql://username:password@host:port/database?sslmode=require

# With additional parameters
postgresql://username:password@host:port/database?sslmode=disable&connect_timeout=10&application_name=openusp
```

### Database Migration

The system automatically creates tables on startup. For manual migration:

```bash
# Run database initialization
psql -h localhost -U openusp -d openusp -f configs/init-db.sql
```

### Performance Tuning

PostgreSQL configuration for optimal performance:

```ini
# postgresql.conf
max_connections = 200
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.7
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 4MB
min_wal_size = 1GB
max_wal_size = 4GB
```

## Message Transport Configuration

### MQTT Configuration

```bash
# Broker connection
OPENUSP_MQTT_BROKER_HOST=localhost
OPENUSP_MQTT_BROKER_PORT=1883
OPENUSP_MQTT_CLIENT_ID=openusp-mtp-$(hostname)

# Authentication
OPENUSP_MQTT_USERNAME=openusp
OPENUSP_MQTT_PASSWORD=secure-password
OPENUSP_MQTT_CLIENT_CERT=/path/to/client.crt
OPENUSP_MQTT_CLIENT_KEY=/path/to/client.key
OPENUSP_MQTT_CA_CERT=/path/to/ca.crt

# Topics
OPENUSP_MQTT_TOPIC_PREFIX=usp/v1
OPENUSP_MQTT_AGENT_TOPIC=agent/+/controller/+
OPENUSP_MQTT_CONTROLLER_TOPIC=controller/+/agent/+

# Connection settings
OPENUSP_MQTT_KEEP_ALIVE=60
OPENUSP_MQTT_CONNECT_TIMEOUT=30
OPENUSP_MQTT_QOS=1
OPENUSP_MQTT_RETAIN=false
OPENUSP_MQTT_CLEAN_SESSION=true
```

### STOMP Configuration

```bash
# Broker connection
OPENUSP_STOMP_BROKER_HOST=localhost
OPENUSP_STOMP_BROKER_PORT=61613
OPENUSP_STOMP_PROTOCOL=tcp

# Authentication
OPENUSP_STOMP_USERNAME=guest
OPENUSP_STOMP_PASSWORD=guest

# Destinations
OPENUSP_STOMP_AGENT_QUEUE=/queue/usp.agent
OPENUSP_STOMP_CONTROLLER_QUEUE=/queue/usp.controller
OPENUSP_STOMP_REPLY_QUEUE=/temp-queue/reply

# Connection settings
OPENUSP_STOMP_HEARTBEAT_SEND=10000
OPENUSP_STOMP_HEARTBEAT_RECEIVE=10000
OPENUSP_STOMP_CONNECT_TIMEOUT=30s
OPENUSP_STOMP_RECEIPT_TIMEOUT=10s
```

### WebSocket Configuration

```bash
# Server settings
OPENUSP_WEBSOCKET_READ_BUFFER_SIZE=4096
OPENUSP_WEBSOCKET_WRITE_BUFFER_SIZE=4096
OPENUSP_WEBSOCKET_MAX_MESSAGE_SIZE=65536

# Connection limits
OPENUSP_WEBSOCKET_MAX_CONNECTIONS=1000
OPENUSP_WEBSOCKET_CONNECTION_TIMEOUT=60s

# Ping/Pong settings
OPENUSP_WEBSOCKET_PING_INTERVAL=30s
OPENUSP_WEBSOCKET_PONG_TIMEOUT=10s

# Compression
OPENUSP_WEBSOCKET_ENABLE_COMPRESSION=true
OPENUSP_WEBSOCKET_COMPRESSION_LEVEL=6
```

## Monitoring Configuration

### Prometheus Metrics

```bash
# Metrics server
OPENUSP_METRICS_ENABLED=true
OPENUSP_METRICS_PORT=9090
OPENUSP_METRICS_PATH=/metrics
OPENUSP_METRICS_HOST=0.0.0.0

# Custom metrics
OPENUSP_METRICS_INCLUDE_SYSTEM=true
OPENUSP_METRICS_INCLUDE_RUNTIME=true
OPENUSP_METRICS_HISTOGRAM_BUCKETS=0.1,0.25,0.5,1,2.5,5,10
```

### Grafana Dashboard

Configuration file: `configs/grafana-datasources.yml`

```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
  - name: PostgreSQL
    type: postgres
    url: postgres:5432
    database: openusp
    user: openusp
    secureJsonData:
      password: openusp123
```

### Health Checks

```bash
# Health check configuration
OPENUSP_HEALTH_CHECK_ENABLED=true
OPENUSP_HEALTH_CHECK_INTERVAL=30s
OPENUSP_HEALTH_CHECK_TIMEOUT=10s
OPENUSP_HEALTH_CHECK_ENDPOINT=/health

# Readiness check
OPENUSP_READINESS_CHECK_ENABLED=true
OPENUSP_READINESS_CHECK_ENDPOINT=/ready
```

## Security Configuration

### HTTP-Only Mode

The API Gateway has been simplified to run in HTTP-only mode. HTTPS/TLS support has been removed for simplicity.

```bash
# API Gateway configuration (HTTP-only)
OPENUSP_API_GATEWAY_PORT=6500
OPENUSP_API_GATEWAY_URL=http://localhost:6500

# For production deployments, consider using a reverse proxy 
# (nginx, Apache, or cloud load balancer) for TLS termination
```

### Authentication Configuration

```bash
# JWT Authentication
OPENUSP_AUTH_JWT_ENABLED=true
OPENUSP_AUTH_JWT_SECRET=your-256-bit-secret
OPENUSP_AUTH_JWT_EXPIRATION=24h
OPENUSP_AUTH_JWT_ISSUER=openusp
OPENUSP_AUTH_JWT_AUDIENCE=openusp-api

# API Key Authentication
OPENUSP_AUTH_API_KEY_ENABLED=true
OPENUSP_AUTH_API_KEY_HEADER=X-API-Key
OPENUSP_AUTH_API_KEY_QUERY=api_key

# OAuth 2.0 (future)
OPENUSP_AUTH_OAUTH_ENABLED=false
OPENUSP_AUTH_OAUTH_PROVIDER=generic
OPENUSP_AUTH_OAUTH_CLIENT_ID=your-client-id
OPENUSP_AUTH_OAUTH_CLIENT_SECRET=your-client-secret
OPENUSP_AUTH_OAUTH_REDIRECT_URL=http://localhost:8080/auth/callback
```

### Rate Limiting

```bash
# Global rate limiting
OPENUSP_RATE_LIMIT_ENABLED=true
OPENUSP_RATE_LIMIT_REQUESTS_PER_MINUTE=1000
OPENUSP_RATE_LIMIT_BURST=100

# Per-endpoint rate limiting
OPENUSP_RATE_LIMIT_DEVICES_GET=500
OPENUSP_RATE_LIMIT_DEVICES_POST=100
OPENUSP_RATE_LIMIT_PARAMETERS_SET=50

# WebSocket rate limiting
OPENUSP_WEBSOCKET_RATE_LIMIT_MESSAGES_PER_MINUTE=100
OPENUSP_WEBSOCKET_RATE_LIMIT_BURST=20
```

## Configuration Validation

### Validation Rules

```bash
# Port ranges
OPENUSP_*_PORT: 1024-65535

# Required fields
OPENUSP_DATABASE_HOST: required
OPENUSP_DATABASE_NAME: required
OPENUSP_DATABASE_USER: required
OPENUSP_DATABASE_PASSWORD: required

# Format validation
OPENUSP_LOG_LEVEL: debug|info|warn|error
OPENUSP_LOG_FORMAT: json|text
OPENUSP_DATABASE_SSLMODE: disable|require|verify-ca|verify-full
```

### Configuration Check

Run configuration validation:

```bash
# Check configuration
make config-check

# Validate specific service
./build/api-gateway --config-check

# Validate environment
./scripts/validate-config.sh
```

## Environment-Specific Configurations

### Development

```bash
# Load development configuration
source configs/openusp.dev.env
```

### Testing

```bash
# Load test configuration
source configs/openusp.test.env
```

### Production

```bash
# Load production configuration
source configs/openusp.env

# Or set specific production values
export OPENUSP_DATABASE_HOST=prod-db.example.com
export OPENUSP_DATABASE_PASSWORD=$(vault kv get -field=password secret/openusp/db)
```

## Configuration Best Practices

### Security
- Use strong, unique passwords
- Enable TLS in production
- Rotate secrets regularly
- Use environment variables for sensitive data
- Never commit secrets to version control

### Performance
- Tune database connection pools
- Configure appropriate timeouts
- Enable compression where beneficial
- Monitor and adjust based on metrics

### Reliability
- Configure health checks
- Set reasonable timeouts
- Enable graceful shutdown
- Configure retry policies
- Monitor resource usage

### Observability
- Enable structured logging
- Configure appropriate log levels
- Set up metrics collection
- Configure distributed tracing (if available)
- Monitor configuration drift

---

For more configuration examples, see the `configs/` directory in the repository.