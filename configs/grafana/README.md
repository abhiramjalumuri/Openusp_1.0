# Grafana Provisioning for OpenUSP

This directory contains Grafana provisioning configuration files that automatically configure datasources and dashboards when Grafana starts.

## Directory Structure

```
configs/grafana/provisioning/
├── datasources/
│   └── prometheus.yml          # Prometheus datasource configuration
└── dashboards/
    ├── dashboards.yml          # Dashboard provider configuration
    └── json/
        └── openusp-overview.json  # OpenUSP monitoring dashboard
```

## Datasources

### Prometheus
- **URL**: `http://prometheus-dev:9090`
- **Type**: Prometheus
- **Default**: Yes
- **Scrape Interval**: 5s

The Prometheus datasource is automatically configured to connect to the Prometheus service running in the Docker network.

## Dashboards

### 1. OpenUSP Platform Overview
- **UID**: `openusp-overview`
- **URL**: http://localhost:3000/d/openusp-overview/openusp-platform-overview
- **Tags**: openusp, tr-369, monitoring

Comprehensive platform monitoring dashboard showing:
- Service health and status
- Message processing metrics
- Error rates and types
- System resource utilization
- Service-level performance indicators

### 2. Data Service & Database Metrics
- **UID**: `openusp-data-service`
- **URL**: http://localhost:3000/d/openusp-data-service/data-service-and-database-metrics
- **Tags**: openusp, data-service, database, performance

Database and data service monitoring:
- Database connection pool metrics
- Query performance and latency
- CRUD operation statistics
- Data service throughput
- Database resource utilization

### 3. TR-069 CWMP Protocol Metrics
- **UID**: `openusp-cwmp-protocol`
- **URL**: http://localhost:3000/d/openusp-cwmp-protocol/tr-069-cwmp-protocol-metrics
- **Tags**: openusp, tr-069, cwmp, protocol

CWMP/TR-069 protocol specific metrics:
- CWMP message types and rates
- TR-069 RPC method statistics
- CWMP session management
- ACS-CPE communication metrics
- Protocol error tracking

### 4. TR-369 USP Protocol Metrics
- **UID**: `openusp-usp-protocol`
- **URL**: http://localhost:3000/d/openusp-usp-protocol/tr-369-usp-protocol-metrics
- **Tags**: openusp, tr-369, usp, protocol

USP/TR-369 protocol specific metrics:
- USP message types and rates
- Agent connection statistics
- USP Record processing
- MTP transport metrics (WebSocket, STOMP, MQTT)
- Protocol compliance and errors

## Accessing Grafana

1. Open browser to: http://localhost:3000
2. Login with:
   - **Username**: `admin`
   - **Password**: `openusp123`
3. Navigate to Dashboards → OpenUSP Overview

## Adding New Dashboards

To add new dashboards:

1. Create your dashboard JSON file in `dashboards/json/`
2. The dashboard will be automatically loaded on Grafana restart
3. Alternatively, use the Grafana UI to create dashboards (they will be editable even when provisioned)

## Metrics Configuration

The OpenUSP services expose Prometheus metrics that are scraped by the Prometheus service. Ensure your services are configured to expose metrics endpoints and that Prometheus is configured to scrape them in `configs/prometheus.yml`.

## Troubleshooting

### Dashboard not showing
- Verify the JSON file is valid: `cat dashboards/json/your-dashboard.json | python3 -m json.tool`
- Check Grafana logs: `docker logs openusp-grafana-dev`
- Restart Grafana: `docker restart openusp-grafana-dev`

### Datasource not working
- Verify Prometheus is running: `docker ps | grep prometheus`
- Check Prometheus is accessible from Grafana: `docker exec openusp-grafana-dev wget -O- http://prometheus-dev:9090/-/healthy`
- Verify datasource in Grafana UI: Settings → Data Sources

### No data in dashboards
- Check if services are exposing metrics: `curl http://localhost:8082/metrics` (MTP service example)
- Verify Prometheus is scraping targets: http://localhost:9090/targets
- Check Prometheus has data: http://localhost:9090/graph

## References

- [Grafana Provisioning Documentation](https://grafana.com/docs/grafana/latest/administration/provisioning/)
- [Prometheus Data Source](https://grafana.com/docs/grafana/latest/datasources/prometheus/)
- [OpenUSP Metrics](../prometheus.yml)
