# OpenUSP Scripts

This directory contains utility scripts for development and maintenance of the OpenUSP platform.

## Available Scripts

### `install-make-completion.sh` 🆕

**Purpose**: Install bash completion for Make targets with smart tab completion

**Usage**:
```bash
# Interactive installer
./scripts/install-make-completion.sh

# Quick install for current session
source scripts/make-completion-advanced.bash

# Use completion
make <TAB><TAB>          # Show all targets
make build<TAB><TAB>     # Show build targets
make-help build          # Show build targets with descriptions
```

**Features**:
- ✅ Tab completion for all Make targets
- ✅ Context-aware smart completion (build*, start*, stop*)
- ✅ Helper function with pattern matching
- ✅ Multiple installation options (session, user, project, system)

See `README-make-completion.md` for detailed documentation.

### `cleanup-consul.sh`

**Purpose**: Clean up stale Consul service registrations during development

**Usage**:
```bash
# Clean only unhealthy service instances
./scripts/cleanup-consul.sh

# Force clean ALL service instances (use before restart)
./scripts/cleanup-consul.sh --force

# Show help
./scripts/cleanup-consul.sh --help
```

**When to use**:
- **Daily**: Before starting development (`./scripts/cleanup-consul.sh --force`)
- **Issues**: When services can't find each other
- **Errors**: "circuit breaker open", "connection refused" errors
- **Restarts**: Before major service restarts

**Integration with Makefile**:
```bash
# Makefile targets that use this script:
make consul-cleanup        # Calls cleanup-consul.sh
make consul-cleanup-force  # Calls cleanup-consul.sh --force
make dev-restart          # Uses cleanup-consul.sh --force
make dev-reset            # Uses cleanup-consul.sh --force
```

### Other Scripts

- `setup-grafana.sh` - Configure Grafana dashboards
- `verify-grafana.sh` - Verify Grafana setup
- `start-dev-with-swagger.sh` - Development startup with Swagger UI

## Development Workflow Integration

These scripts are designed to integrate with the Makefile-based development workflow:

```bash
# Morning routine
make dev-status              # Check current state
make consul-cleanup-force    # Clean stale registrations
make dev-restart            # Clean restart

# Development iteration
make build-usp-service      # Make changes and rebuild
make consul-cleanup         # Clean unhealthy instances
make restart-usp-service    # Restart with changes

# Problem resolution
./scripts/cleanup-consul.sh --force  # Emergency cleanup
make dev-reset                      # Nuclear option
```

## Script Maintenance

All scripts should:
- Be executable (`chmod +x`)
- Handle errors gracefully (`set -e`)
- Provide help text (`--help` flag)
- Work with the current OpenUSP service architecture
- Integrate with the Makefile where appropriate

For more information, see [MAKEFILE_GUIDE.md](../docs/MAKEFILE_GUIDE.md).