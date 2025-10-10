# OpenUSP Scripts

This directory contains essential utility scripts for development and maintenance of the OpenUSP platform.

## Available Scripts

### `setup-infrastructure.sh`

**Purpose**: Set up and repair infrastructure services for OpenUSP

**Usage**:
```bash
# Setup complete infrastructure
./scripts/setup-infrastructure.sh
```

**What it does**:
- Starts all infrastructure services (PostgreSQL, Prometheus, Grafana, etc.)
- Configures Grafana dashboards
- Validates service health
- Works with static port configuration (no service discovery needed)

### `docker-health.sh`

**Purpose**: Comprehensive Docker health check and repair utility

**Usage**:
```bash
# Run health check
./scripts/docker-health.sh check

# Attempt to fix common issues
./scripts/docker-health.sh fix
```

**Features**:
- Docker daemon connectivity check
- Container health status
- Network connectivity verification
- Volume mount validation
- Automatic repair suggestions

### `setup-grafana.sh`

**Purpose**: Configure Grafana with dashboards, data sources, and settings

**Usage**:
```bash
# Setup Grafana (run after infra-up)
./scripts/setup-grafana.sh
```

**What it does**:
- Configures Prometheus data source
- Imports OpenUSP dashboards
- Sets up default settings
- Configures alerts and notifications

### `setup-grafana-dashboards.sh`

**Purpose**: Ensure Grafana dashboards are properly configured and accessible

**Usage**:
```bash
# Setup dashboards only
./scripts/setup-grafana-dashboards.sh
```

**Features**:
- Validates Grafana accessibility
- Imports dashboard configurations
- Updates existing dashboards
- Verifies dashboard functionality

## Integration with Makefile

These scripts are integrated with the Makefile targets:

```bash
# Use via Makefile (recommended)
make docker-health       # docker-health.sh check
make docker-fix          # docker-health.sh fix
make setup-grafana       # setup-grafana.sh
make verify-grafana      # verify-grafana.sh (inline implementation)
```

## Development Workflow

**Daily startup routine**:
```bash
make consul-cleanup      # Clean stale registrations
make infra-up           # Start infrastructure
make setup-grafana      # Configure monitoring
```

**Troubleshooting**:
```bash
make docker-health      # Diagnose issues
make docker-fix         # Attempt repairs
```