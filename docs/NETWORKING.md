# OpenUSP Network Architecture

## Overview

OpenUSP now uses **Docker service name networking** as the default approach for container-to-container communication. This eliminates platform-specific networking issues and provides consistent behavior across Linux, macOS, and Windows.

## Network Approaches

### 1. Network-Aware (Default) ✅ Recommended

Uses Docker service names for reliable container-to-container communication:

```bash
# Start network-aware infrastructure
make infra-up                # Uses network-aware by default
make infra-network-up        # Explicit network-aware startup
```

**Benefits:**
- ✅ **Cross-platform compatibility** - Works identically on Linux, macOS, Windows
- ✅ **No host-specific configuration** - No need for `host.docker.internal` workarounds
- ✅ **Built-in service discovery** - Automatic DNS resolution within Docker network
- ✅ **Container isolation** - Services communicate through dedicated Docker network
- ✅ **Simplified configuration** - Single configuration works everywhere

**Service Resolution:**
- Prometheus scrapes: `openusp-consul:8500`, `openusp-rabbitmq-dev:15692`
- Database connections: `openusp-postgres:5432`  
- Message brokers: `openusp-rabbitmq-dev:5672`, `openusp-mosquitto-dev:1883`

### 2. Platform-Specific (Fallback)

Uses platform-optimized networking with `host.docker.internal` or host IP detection:

```bash
# Platform-specific infrastructure
make infra-platform-up       # Platform-optimized startup
make platform-info          # Show platform detection info
```

**When to use:**
- Legacy compatibility requirements
- Debugging network issues
- Performance optimization needs
- Specific platform requirements

### 3. Legacy (Basic)

Basic Docker Compose networking without optimizations:

```bash
# Legacy infrastructure
make infra-legacy-up         # Basic Docker Compose networking
```

## Network Configuration

### Docker Network: `openusp-dev`

```yaml
networks:
  openusp-dev:
    driver: bridge
    # No custom subnet - uses Docker's automatic assignment
```

### Service Names

| Service | Container Name | Network Address | External Port |
|---------|----------------|-----------------|---------------|
| PostgreSQL | `openusp-postgres` | `openusp-postgres:5432` | `5433` |
| Consul | `openusp-consul` | `openusp-consul:8500` | `8500` |
| RabbitMQ | `openusp-rabbitmq-dev` | `openusp-rabbitmq-dev:5672` | `5672` |
| Mosquitto | `openusp-mosquitto-dev` | `openusp-mosquitto-dev:1883` | `1883` |
| Prometheus | `openusp-prometheus-dev` | `openusp-prometheus-dev:9090` | `9090` |
| Grafana | `openusp-grafana-dev` | `openusp-grafana-dev:3000` | `3000` |
| Adminer | `openusp-adminer` | `openusp-adminer:8080` | `8080` |

## Migration Guide

### From Host-Based to Service Names

**Old Configuration (host-based):**
```yaml
scrape_configs:
  - job_name: 'consul'
    static_configs:
      - targets: ['host.docker.internal:8500']
```

**New Configuration (service names):**
```yaml
scrape_configs:
  - job_name: 'infrastructure-services'
    static_configs:
      - targets: ['openusp-consul:8500']
```

### Application Configuration

Update your service configurations to use service names:

```yaml
# Database connection
database:
  host: openusp-postgres  # Instead of localhost or host.docker.internal
  port: 5432              # Internal port, not external 5433

# Service registry
consul:
  address: openusp-consul:8500

# Message broker  
rabbitmq:
  host: openusp-rabbitmq-dev
  port: 5672
```

## Network Troubleshooting

### Check Network Status
```bash
make infra-network-status    # Show network-aware infrastructure
make platform               # Platform detection + infrastructure
```

### Network Inspection
```bash
# View Docker network details
docker network inspect openusp-dev

# Test service resolution
docker run --rm --network openusp-dev alpine:latest nslookup openusp-consul

# Test connectivity
docker run --rm --network openusp-dev alpine:latest nc -zv openusp-postgres 5432
```

### Common Issues

**Service not found:**
```bash
# Check if service is running
docker compose -f deployments/docker-compose.infra.network.yml ps

# Restart infrastructure
make infra-down && make infra-up
```

**Network conflicts:**
```bash
# Remove conflicting networks
docker network prune

# Check for IP conflicts
docker network ls
```

## Performance Considerations

### Network-Aware Advantages:
- **Lower latency** - Direct container-to-container communication
- **Better throughput** - No host networking overhead
- **Reduced CPU usage** - Optimized Docker networking
- **Container isolation** - Security through network segmentation

### Monitoring Network Performance:
```bash
# Container network statistics
docker stats --format "table {{.Container}}\t{{.NetIO}}"

# Network-specific metrics in Prometheus
curl http://localhost:9090/api/v1/query?query=container_network_receive_bytes_total
```

## Best Practices

1. **Use service names** in all container-to-container communications
2. **External access** still uses `localhost` and external ports
3. **Health checks** should use internal service names and ports
4. **Configuration files** should reference service names, not IPs
5. **Testing** should verify both internal and external connectivity

## Configuration Files

- `deployments/docker-compose.infra.network.yml` - Network-aware infrastructure
- `configs/prometheus-network.yml` - Service name-based monitoring
- `Makefile` - Default network-aware targets
- `Makefile.linux` / `Makefile.macos` - Platform-specific fallbacks

## External Resources

- [Docker Networking Documentation](https://docs.docker.com/network/)
- [Docker Compose Networking](https://docs.docker.com/compose/networking/)
- [Service Discovery Best Practices](https://docs.docker.com/compose/networking/#links)