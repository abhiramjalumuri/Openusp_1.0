# OpenUSP Cross-Platform Infrastructure Setup

This document explains the infrastructure setup that works seamlessly across macOS, Linux, and Windows.

## Host Network Mode Benefits

The monitoring infrastructure (Prometheus and Grafana) now uses Docker's **host network mode**, which provides:

✅ **Cross-platform compatibility**: Works on macOS, Linux, and Windows  
✅ **No host.docker.internal issues**: Direct localhost access  
✅ **Simplified configuration**: No complex networking setup  
✅ **Better performance**: Reduced network overhead  

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Host Machine                         │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │   Prometheus    │  │     Grafana     │                  │
│  │ (host network)  │  │ (host network)  │                  │
│  │  localhost:9090 │  │  localhost:3000 │                  │
│  └─────────────────┘  └─────────────────┘                  │
│           │                      │                         │
│           └──────────┬───────────┘                         │
│                      │                                     │
│  ┌─────────────────────────────────────────────────────────┤
│  │              OpenUSP Services                           │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │  │API Gate │ │Data Svc │ │Conn Mgr │ │USP Svc  │  ...  │
│  │  │  :6500  │ │  :6100  │ │  :6200  │ │  :6400  │       │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
│  └─────────────────────────────────────────────────────────┘
│                                                             │
│  ┌─────────────────────────────────────────────────────────┤
│  │          Other Infrastructure (Bridge Network)          │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │  │Postgres │ │RabbitMQ │ │Mosquitto│ │  Other  │       │
│  │  │  :5433  │ │ :15672  │ │  :1883  │ │Services │       │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
│  └─────────────────────────────────────────────────────────┘
└─────────────────────────────────────────────────────────────┘
```

## Configuration Files

### Prometheus Configuration
- **File**: `configs/prometheus-static-host.yml`
- **Target URLs**: `localhost:6100`, `localhost:6200`, etc.
- **Network**: Host network mode for direct access

### Grafana Configuration
- **File**: `configs/grafana-datasources-host.yml`
- **Prometheus URL**: `http://localhost:9090`
- **Network**: Host network mode for direct Prometheus access

## Usage

### Start Infrastructure
```bash
# Works on all platforms
docker-compose -f deployments/docker-compose.infra.yml up -d
```

### Access URLs
- **Grafana**: http://localhost:3000 (admin/openusp123)
- **Prometheus**: http://localhost:9090
- **PostgreSQL**: localhost:5433 (openusp/openusp123)
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)

### Available Dashboards
1. **OpenUSP Platform Overview** - High-level system metrics
2. **Data Service & Database Metrics** - Database performance 
3. **TR-369 USP Protocol Metrics** - USP protocol statistics
4. **TR-069 CWMP Protocol Metrics** - CWMP protocol statistics

## Platform-Specific Notes

### Linux
- Host network mode works natively
- No additional configuration needed

### macOS (Docker Desktop)
- Host network mode supported
- Direct localhost access works perfectly

### Windows (Docker Desktop)
- Host network mode supported with Docker Desktop
- WSL2 backend recommended

## Troubleshooting

### Port Conflicts
If you see "port already in use" errors:
```bash
# Check what's using the ports
lsof -i :3000 -i :9090

# Stop conflicting services
sudo pkill -f grafana
sudo pkill -f prometheus
```

### Service Discovery Issues
The monitoring stack will automatically discover services at:
- API Gateway: `localhost:6500/metrics`
- Data Service: `localhost:6100/metrics`
- Connection Manager: `localhost:6200/metrics`
- USP Service: `localhost:6400/metrics`
- CWMP Service: `localhost:7547/metrics`
- MTP Service: `localhost:8082/metrics`