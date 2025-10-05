# 🔧 Troubleshooting Guide

Common issues and solutions for OpenUSP deployment and operation.

## Table of Contents

- [General Troubleshooting](#general-troubleshooting)
- [Service-Specific Issues](#service-specific-issues)
- [Database Issues](#database-issues)
- [Network and Connectivity](#network-and-connectivity)
- [Performance Issues](#performance-issues)
- [Protocol-Specific Problems](#protocol-specific-problems)
- [Docker and Container Issues](#docker-and-container-issues)
- [Kubernetes Issues](#kubernetes-issues)

## General Troubleshooting

### Basic Health Checks

```bash
# Check all service health
curl http://localhost:8080/health    # API Gateway
curl http://localhost:8081/health    # MTP Service
curl http://localhost:7547/health    # CWMP Service

# Check service status
make status

# View service logs
make logs

# Or individual service logs
tail -f logs/api-gateway.log
tail -f logs/data-service.log
tail -f logs/mtp-service.log
```

### Service Discovery

```bash
# Check Consul connectivity
curl http://localhost:8500/v1/status/leader

# List registered services
curl http://localhost:8500/v1/catalog/services

# Check specific service health
curl http://localhost:8500/v1/health/service/api-gateway
```

### Version Information

```bash
# Check service versions
./build/api-gateway --version
./build/data-service --version
./build/mtp-service --version
./build/usp-service --version
./build/cwmp-service --version

# Or use make target
make version
```

### Configuration Validation

```bash
# Validate configuration
./build/api-gateway --config-check
./build/data-service --config-check

# Check environment variables
env | grep OPENUSP
```

## Service-Specific Issues

### API Gateway Issues

#### Problem: Service Won't Start

**Symptoms:**
- Service exits immediately
- "Port already in use" error
- "Connection refused" to data service

**Solutions:**

```bash
# Check port availability
netstat -tlnp | grep 8080
lsof -i :8080

# Use alternative port
export OPENUSP_API_GATEWAY_PORT=8082
./build/api-gateway

# Check data service connectivity
telnet localhost 9092

# View detailed logs
export OPENUSP_LOG_LEVEL=debug
./build/api-gateway
```

#### Problem: gRPC Connection Failed

**Symptoms:**
- "connection refused" errors
- "transport: error while dialing" messages

**Solutions:**

```bash
# Verify data service is running
ps aux | grep data-service

# Check gRPC connectivity
grpcurl -plaintext localhost:9092 list

# Test with explicit address
export OPENUSP_API_GATEWAY_DATA_SERVICE_ADDRESS=localhost:9092
./build/api-gateway

# Check firewall/network
telnet localhost 9092
```

#### Problem: CORS Issues

**Symptoms:**
- Browser console shows CORS errors
- Preflight requests failing

**Solutions:**

```bash
# Enable CORS
export OPENUSP_API_GATEWAY_CORS_ENABLED=true

# Configure allowed origins
export OPENUSP_API_GATEWAY_CORS_ORIGINS="http://localhost:3000,https://app.example.com"

# Check CORS headers
curl -H "Origin: http://localhost:3000" \
     -H "Access-Control-Request-Method: POST" \
     -H "Access-Control-Request-Headers: X-Requested-With" \
     -X OPTIONS \
     http://localhost:8080/api/v1/devices
```

### Data Service Issues

#### Problem: Database Connection Failed

**Symptoms:**
- "connection refused" to PostgreSQL
- "authentication failed" errors
- "database does not exist" errors

**Solutions:**

```bash
# Check PostgreSQL status
sudo systemctl status postgresql
# Or for Docker
docker ps | grep postgres

# Test database connectivity
psql -h localhost -U openusp -d openusp -c "SELECT version();"

# Check connection string
export OPENUSP_DATABASE_HOST=localhost
export OPENUSP_DATABASE_PORT=5432
export OPENUSP_DATABASE_NAME=openusp
export OPENUSP_DATABASE_USER=openusp
export OPENUSP_DATABASE_PASSWORD=openusp123

# Create database if missing
sudo -u postgres createdb openusp
sudo -u postgres createuser openusp
sudo -u postgres psql -c "ALTER USER openusp PASSWORD 'openusp123';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE openusp TO openusp;"
```

#### Problem: gRPC Server Won't Start

**Symptoms:**
- "bind: address already in use"
- gRPC server startup failures

**Solutions:**

```bash
# Check port usage
netstat -tlnp | grep 9092
lsof -i :9092

# Kill existing process
pkill -f data-service

# Use alternative port
export OPENUSP_DATA_SERVICE_PORT=9093
./build/data-service

# Check for permission issues
sudo netstat -tlnp | grep 9092
```

### MTP Service Issues

#### Problem: WebSocket Connection Failed

**Symptoms:**
- WebSocket connection refused
- Connection drops immediately
- CORS errors on WebSocket

**Solutions:**

```bash
# Check WebSocket endpoint
curl -i -N -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Version: 13" \
     -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
     http://localhost:8081/ws

# Enable WebSocket debugging
export OPENUSP_LOG_LEVEL=debug
export OPENUSP_MTP_SERVICE_WEBSOCKET_DEBUG=true
./build/mtp-service

# Check firewall
sudo ufw status
sudo iptables -L | grep 8081
```

#### Problem: MQTT Connection Issues

**Symptoms:**
- "connection refused" to MQTT broker
- Authentication failures
- Message delivery failures

**Solutions:**

```bash
# Check MQTT broker
mosquitto_pub -h localhost -t test -m "hello"
mosquitto_sub -h localhost -t test

# Test with authentication
mosquitto_pub -h localhost -u username -P password -t test -m "hello"

# Configure MQTT settings
export OPENUSP_MQTT_BROKER_HOST=localhost
export OPENUSP_MQTT_BROKER_PORT=1883
export OPENUSP_MQTT_USERNAME=openusp
export OPENUSP_MQTT_PASSWORD=secure-password

# Check broker logs
docker logs mosquitto
# Or
sudo journalctl -u mosquitto
```

#### Problem: STOMP Connection Issues

**Symptoms:**
- RabbitMQ connection failures
- STOMP plugin not enabled
- Queue access denied

**Solutions:**

```bash
# Check RabbitMQ status
sudo systemctl status rabbitmq-server
# Or
docker ps | grep rabbitmq

# Enable STOMP plugin
sudo rabbitmq-plugins enable rabbitmq_stomp

# Check STOMP port
telnet localhost 61613

# Test STOMP connection
echo -e "CONNECT\nhost:/\n\n\x00" | nc localhost 61613

# Configure credentials
export OPENUSP_STOMP_USERNAME=guest
export OPENUSP_STOMP_PASSWORD=guest
```

### USP Service Issues

#### Problem: USP Message Parsing Failed

**Symptoms:**
- "failed to parse USP record" errors
- "unknown message type" errors
- Protocol buffer unmarshaling failures

**Solutions:**

```bash
# Test USP parsing
go run test/usp_parsing_demo.go

# Enable USP debugging
export OPENUSP_LOG_LEVEL=debug
export OPENUSP_USP_DEBUG_PARSING=true
./build/usp-service

# Validate USP message format
# Check if message is valid protobuf
# Verify USP version (1.3 or 1.4)

# Test with sample USP message
curl -X POST http://localhost:8081/ws \
     -H "Content-Type: application/octet-stream" \
     --data-binary @test/sample-usp-record.bin
```

### CWMP Service Issues

#### Problem: SOAP Parsing Failed

**Symptoms:**
- "failed to parse SOAP envelope" errors
- "invalid XML" errors
- Authentication failures

**Solutions:**

```bash
# Test CWMP endpoint
curl -X POST http://localhost:7547 \
     -H "Content-Type: text/xml" \
     -H "SOAPAction: urn:dslforum-org:cwmp-1-0#Inform" \
     --data-raw '<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Header/>
  <soap:Body>
    <cwmp:Inform xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
      <DeviceId>
        <Manufacturer>Example</Manufacturer>
        <OUI>123456</OUI>
        <ProductClass>Gateway</ProductClass>
        <SerialNumber>SN123</SerialNumber>
      </DeviceId>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>'

# Test with authentication
curl -X POST http://localhost:7547 \
     -u acs:acs123 \
     -H "Content-Type: text/xml" \
     --data-raw '<soap:Envelope>...</soap:Envelope>'

# Check authentication settings
export OPENUSP_CWMP_AUTH_USERNAME=acs
export OPENUSP_CWMP_AUTH_PASSWORD=acs123
```

## Database Issues

### Connection Problems

#### Problem: "Too Many Connections"

**Solutions:**

```bash
# Check current connections
psql -h localhost -U openusp -d openusp -c "SELECT count(*) FROM pg_stat_activity;"

# Show connection limits
psql -h localhost -U openusp -d openusp -c "SHOW max_connections;"

# Kill idle connections
psql -h localhost -U openusp -d openusp -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle' AND state_change < current_timestamp - interval '5 minutes';"

# Increase connection limit (PostgreSQL config)
sudo vim /etc/postgresql/15/main/postgresql.conf
# max_connections = 200

# Restart PostgreSQL
sudo systemctl restart postgresql
```

#### Problem: "Authentication Failed"

**Solutions:**

```bash
# Check pg_hba.conf
sudo vim /etc/postgresql/15/main/pg_hba.conf

# Add or modify line:
# local   openusp    openusp                     md5
# host    openusp    openusp    127.0.0.1/32     md5

# Reload configuration
sudo systemctl reload postgresql

# Reset password
sudo -u postgres psql -c "ALTER USER openusp PASSWORD 'openusp123';"

# Test connection
psql -h localhost -U openusp -d openusp -c "SELECT current_user;"
```

### Performance Issues

#### Problem: Slow Queries

**Solutions:**

```sql
-- Enable query logging
ALTER SYSTEM SET log_statement = 'all';
ALTER SYSTEM SET log_min_duration_statement = 1000;  -- Log queries > 1s
SELECT pg_reload_conf();

-- Check slow queries
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Add missing indices
CREATE INDEX CONCURRENTLY idx_devices_endpoint_id ON devices(endpoint_id);
CREATE INDEX CONCURRENTLY idx_parameters_device_path ON parameters(device_id, path);

-- Update table statistics
ANALYZE devices;
ANALYZE parameters;
ANALYZE alerts;
```

#### Problem: High Memory Usage

**Solutions:**

```sql
-- Check memory usage
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Optimize memory settings
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET work_mem = '4MB';
SELECT pg_reload_conf();

-- Vacuum and analyze
VACUUM ANALYZE;
```

## Network and Connectivity

### Port Conflicts

#### Problem: "Address Already in Use"

**Solutions:**

```bash
# Find process using port
lsof -i :8080
netstat -tlnp | grep 8080

# Kill process
kill -9 <PID>

# Use alternative ports
export OPENUSP_API_GATEWAY_PORT=8082
export OPENUSP_MTP_SERVICE_PORT=8083
export OPENUSP_CWMP_SERVICE_PORT=7548

# Check available ports
for port in {8080..8090}; do
  if ! nc -z localhost $port 2>/dev/null; then
    echo "Port $port is available"
  fi
done
```

### Firewall Issues

#### Problem: Connection Timeouts

**Solutions:**

```bash
# Check firewall status
sudo ufw status verbose
sudo iptables -L -n

# Allow OpenUSP ports
sudo ufw allow 8080/tcp  # API Gateway
sudo ufw allow 8081/tcp  # MTP Service
sudo ufw allow 7547/tcp  # CWMP Service
sudo ufw allow 9092/tcp  # Data Service (if external)

# Test connectivity
telnet localhost 8080
nc -zv localhost 8080

# Check listening ports
ss -tlnp | grep -E '(8080|8081|7547|9092)'
```

### DNS Issues

#### Problem: Service Discovery Failures

**Solutions:**

```bash
# Check Consul DNS
dig @localhost -p 8600 api-gateway.service.consul

# Check system DNS
nslookup api-gateway.service.consul

# Configure DNS resolver
echo "nameserver 127.0.0.1" | sudo tee -a /etc/resolv.conf
echo "port 8600" | sudo tee -a /etc/resolv.conf

# Test service resolution
curl http://api-gateway.service.consul:8080/health
```

## Performance Issues

### High CPU Usage

#### Problem: Services Using Too Much CPU

**Solutions:**

```bash
# Monitor CPU usage
top -p $(pgrep -f "api-gateway|data-service|mtp-service")
htop

# Check goroutine leaks
curl http://localhost:8080/debug/pprof/goroutine?debug=1

# Profile CPU usage
go tool pprof http://localhost:8080/debug/pprof/profile

# Tune garbage collection
export GOGC=200  # Less frequent GC
export GOMEMLIMIT=1GiB  # Memory limit

# Limit concurrent connections
export OPENUSP_API_GATEWAY_MAX_CONNECTIONS=100
export OPENUSP_DATA_SERVICE_MAX_CONNECTIONS=50
```

### High Memory Usage

#### Problem: Memory Leaks or High Memory Consumption

**Solutions:**

```bash
# Monitor memory usage
ps aux | grep -E "(api-gateway|data-service|mtp-service)"

# Check heap profile
go tool pprof http://localhost:8080/debug/pprof/heap

# Check for memory leaks
curl http://localhost:8080/debug/pprof/heap?debug=1

# Set memory limits
export GOMEMLIMIT=512MiB

# Tune database connection pools
export OPENUSP_DATA_SERVICE_MAX_CONNECTIONS=20
export OPENUSP_DATA_SERVICE_MAX_IDLE_CONNECTIONS=5
```

### Slow Response Times

#### Problem: High Latency

**Solutions:**

```bash
# Test API response times
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8080/api/v1/devices

# Where curl-format.txt contains:
#      time_namelookup:  %{time_namelookup}\n
#         time_connect:  %{time_connect}\n
#      time_appconnect:  %{time_appconnect}\n
#     time_pretransfer:  %{time_pretransfer}\n
#        time_redirect:  %{time_redirect}\n
#   time_starttransfer:  %{time_starttransfer}\n
#                     ----------\n
#           time_total:  %{time_total}\n

# Check database query performance
psql -h localhost -U openusp -d openusp -c "
SELECT query, mean_exec_time, calls, rows
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;"

# Enable database query optimization
export OPENUSP_DATA_SERVICE_QUERY_TIMEOUT=10s
export OPENUSP_DATA_SERVICE_PREPARE_STATEMENTS=true
```

## Protocol-Specific Problems

### USP Protocol Issues

#### Problem: Version Mismatch

**Symptoms:**
- "unsupported USP version" errors
- Parsing failures with valid messages

**Solutions:**

```bash
# Check supported versions
grep -r "version.*1\.[34]" internal/usp/

# Enable version debugging
export OPENUSP_USP_DEBUG_VERSION=true

# Test with specific version
# Create test message with version 1.4
# Verify protobuf definitions match version
```

#### Problem: Endpoint ID Conflicts

**Symptoms:**
- "duplicate endpoint ID" errors
- Message routing failures

**Solutions:**

```bash
# Check existing endpoint IDs
curl http://localhost:8080/api/v1/devices | jq '.data[].endpoint_id'

# Generate unique endpoint IDs
export OPENUSP_USP_ENDPOINT_PREFIX="controller-$(hostname)-"

# Clear conflicting devices
curl -X DELETE http://localhost:8080/api/v1/devices/conflicting-device-id
```

### CWMP Protocol Issues

#### Problem: XML Parsing Errors

**Symptoms:**
- "invalid XML" errors
- SOAP envelope parsing failures

**Solutions:**

```bash
# Validate XML format
xmllint --format --noout soap-message.xml

# Test with minimal SOAP envelope
curl -X POST http://localhost:7547 \
     -H "Content-Type: text/xml" \
     --data-raw '<?xml version="1.0"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <cwmp:Inform xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
      <DeviceId>
        <Manufacturer>Test</Manufacturer>
        <OUI>123456</OUI>
        <ProductClass>Test</ProductClass>
        <SerialNumber>TEST123</SerialNumber>
      </DeviceId>
    </cwmp:Inform>
  </soap:Body>
</soap:Envelope>'

# Enable XML debugging
export OPENUSP_CWMP_DEBUG_XML=true
```

## Docker and Container Issues

### Container Won't Start

#### Problem: Container Exits Immediately

**Solutions:**

```bash
# Check container logs
docker logs openusp-api-gateway

# Run container interactively
docker run -it --rm openusp/api-gateway:latest /bin/sh

# Check container health
docker inspect openusp-api-gateway | jq '.[0].State.Health'

# Override entrypoint for debugging
docker run -it --rm --entrypoint=/bin/sh openusp/api-gateway:latest
```

### Network Issues in Docker

#### Problem: Service-to-Service Communication Failed

**Solutions:**

```bash
# Check Docker network
docker network ls
docker network inspect openusp_default

# Test container connectivity
docker exec openusp-api-gateway ping openusp-data-service

# Check exposed ports
docker port openusp-api-gateway

# Use Docker Compose service names
export OPENUSP_DATA_SERVICE_ADDRESS=data-service:9092
```

### Volume Mount Issues

#### Problem: Configuration Files Not Found

**Solutions:**

```bash
# Check volume mounts
docker inspect openusp-api-gateway | jq '.[0].Mounts'

# Mount configuration correctly
docker run -v $(pwd)/configs:/app/configs openusp/api-gateway:latest

# Fix file permissions
sudo chown -R 1000:1000 configs/
sudo chmod -R 644 configs/
```

## Kubernetes Issues

### Pod Issues

#### Problem: Pods Not Starting

**Solutions:**

```bash
# Check pod status
kubectl get pods -n openusp

# Describe problematic pod
kubectl describe pod api-gateway-xxx -n openusp

# Check pod logs
kubectl logs api-gateway-xxx -n openusp

# Check events
kubectl get events -n openusp --sort-by='.firstTimestamp'

# Check resource constraints
kubectl top pods -n openusp
kubectl describe node
```

### Service Discovery Issues

#### Problem: Services Can't Reach Each Other

**Solutions:**

```bash
# Check service endpoints
kubectl get endpoints -n openusp

# Test service connectivity
kubectl exec -it api-gateway-xxx -n openusp -- nc -zv data-service 9092

# Check DNS resolution
kubectl exec -it api-gateway-xxx -n openusp -- nslookup data-service

# Verify service configuration
kubectl get service data-service -n openusp -o yaml
```

### ConfigMap and Secret Issues

#### Problem: Configuration Not Applied

**Solutions:**

```bash
# Check ConfigMap
kubectl get configmap openusp-config -n openusp -o yaml

# Update ConfigMap
kubectl create configmap openusp-config --from-env-file=configs/openusp.env -n openusp --dry-run=client -o yaml | kubectl apply -f -

# Restart deployment to pick up changes
kubectl rollout restart deployment/api-gateway -n openusp

# Check secret mounting
kubectl exec -it api-gateway-xxx -n openusp -- ls -la /etc/secrets/
```

## Debugging Tools and Commands

### Useful Commands

```bash
# Service health checks
make health-check

# View all logs
make logs

# Test API endpoints
make test-api

# Check service dependencies
make check-deps

# Generate debug information
make debug-info
```

### Log Analysis

```bash
# Follow logs in real-time
tail -f logs/*.log

# Search for errors
grep -i error logs/*.log

# Count error types
grep -i error logs/api-gateway.log | cut -d'"' -f4 | sort | uniq -c

# Filter by time range
awk '$1 >= "2024-01-15T10:00:00" && $1 <= "2024-01-15T11:00:00"' logs/api-gateway.log
```

### Network Debugging

```bash
# Test all service endpoints
for port in 8080 8081 7547 9092; do
  echo "Testing port $port:"
  nc -zv localhost $port
done

# Check service discovery
dig @localhost -p 8600 _api-gateway._tcp.service.consul SRV

# Monitor network traffic
sudo tcpdump -i lo port 8080
```

## Getting Help

### Log Collection for Support

```bash
# Collect all logs
mkdir debug-$(date +%Y%m%d-%H%M%S)
cp logs/*.log debug-*/
cp configs/*.env debug-*/

# System information
uname -a > debug-*/system-info.txt
docker version >> debug-*/system-info.txt
go version >> debug-*/system-info.txt

# Service status
make status > debug-*/service-status.txt

# Create archive
tar -czf openusp-debug-$(date +%Y%m%d-%H%M%S).tar.gz debug-*/
```

### Reporting Issues

When reporting issues, include:

1. **Environment**: OS, Docker version, Kubernetes version
2. **OpenUSP version**: `./build/api-gateway --version`
3. **Configuration**: Relevant environment variables (redact secrets)
4. **Logs**: Service logs around the time of the issue
5. **Steps to reproduce**: Exact commands and expected vs. actual behavior
6. **Network setup**: Service discovery configuration, firewall rules

### Community Resources

- **GitHub Issues**: Report bugs and request features
- **Documentation**: Check latest documentation for updates
- **Examples**: Review working protocol agents in `cmd/usp-agent/` and `cmd/cwmp-agent/`
- **Stack Overflow**: Tag questions with `openusp` and `tr-369`

---

This troubleshooting guide covers common issues. For specific problems not covered here, check the service logs and GitHub issues for similar problems and solutions.