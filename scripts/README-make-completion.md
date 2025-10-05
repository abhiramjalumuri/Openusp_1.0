# Make Target Bash Completion for OpenUSP

This directory contains bash completion scripts for OpenUSP Make targets.

## Installation

### Option 1: Temporary (Current Session Only)

```bash
# Load basic completion
source scripts/make-completion.bash

# OR load advanced completion with helper functions
source scripts/make-completion-advanced.bash
```

### Option 2: Permanent Installation

#### For Current User (~/.bashrc)

```bash
# Add to your ~/.bashrc or ~/.bash_profile
echo 'source ~/shibu/work/plume/openusp/scripts/make-completion-advanced.bash' >> ~/.bashrc
source ~/.bashrc
```

#### For All Users (System-wide)

```bash
# Copy to system completion directory (requires sudo)
sudo cp scripts/make-completion.bash /etc/bash_completion.d/openusp-make
# OR
sudo cp scripts/make-completion-advanced.bash /etc/bash_completion.d/openusp-make
```

#### For Project-specific Installation

```bash
# Create project-specific bashrc
cat >> .bashrc << 'EOF'
# OpenUSP Project Environment
source scripts/make-completion-advanced.bash
export OPENUSP_PROJECT_ROOT=$(pwd)
EOF

# Load it
source .bashrc
```

## Usage

### Basic Tab Completion

```bash
make <TAB><TAB>          # Show all available targets
make build<TAB><TAB>     # Show all build-* targets
make start<TAB><TAB>     # Show all start-* targets
make infra<TAB><TAB>     # Show all infra-* targets
```

### Advanced Features (with advanced script)

```bash
# Show help for targets matching pattern
make-help build          # Show all build targets
make-help service        # Show service-related targets
make-help                # Show all targets

# Smart context-aware completion
make build-<TAB>         # Only shows build targets
make start-<TAB>         # Only shows start targets
make stop-<TAB>          # Only shows stop targets
```

## Available Target Categories

### Infrastructure
- `infra-up`, `infra-down`, `infra-status`, `infra-clean`
- `consul-status`, `consul-clean`

### Building
- `build-all`, `build-services`, `build-agents`
- `build-api-gateway`, `build-data-service`, `build-usp-service`
- `build-mtp-service`, `build-cwmp-service`, `build-connection-manager`
- `build-usp-agent`, `build-cwmp-agent`

### Services Management
- `start-all`, `stop-all`, `clean-all`
- `start-services`, `stop-services`, `clean-services`
- Individual service targets: `start-api-gateway`, etc.

### Protocol Agents
- `start-usp-agent`, `start-cwmp-agent`
- `stop-usp-agent`, `stop-cwmp-agent`
- `logs-usp-agent`, `logs-cwmp-agent`

### Development
- `quick-start`, `run-dev-swagger`
- `version`, `endpoints`, `docs`
- `test`, `test-unit`, `test-integration`

## Features

### Basic Script (`make-completion.bash`)
- ✅ Tab completion for all Make targets
- ✅ Automatic Makefile discovery
- ✅ Works in any subdirectory of the project

### Advanced Script (`make-completion-advanced.bash`)
- ✅ All basic features
- ✅ Context-aware smart completion
- ✅ Target categorization (build*, start*, stop*, etc.)
- ✅ Helper function `make-help` with pattern matching
- ✅ Target descriptions extraction
- ✅ Enhanced user experience with colored output

## Troubleshooting

### Completion Not Working
```bash
# Check if completion is loaded
complete -p make

# Reload completion
source scripts/make-completion-advanced.bash

# Check if bash completion is available
ls /etc/bash_completion.d/
```

### Makefile Not Found
The script automatically searches for Makefile in:
- Current directory (`.`)
- Parent directory (`..`)
- Grandparent directory (`../..`)
- OpenUSP project root (`~/shibu/work/plume/openusp`)

### Performance Issues
If completion is slow with large Makefiles:
```bash
# Use basic completion instead of advanced
source scripts/make-completion.bash
```

## Integration with IDE/Editors

### VS Code
Add to your VS Code settings for integrated terminal:
```json
{
    "terminal.integrated.shellArgs.osx": ["-l", "-c", "source ~/shibu/work/plume/openusp/scripts/make-completion-advanced.bash && exec bash"]
}
```

### tmux
Add to your `~/.tmux.conf`:
```bash
set-option -g default-command "bash -l -c 'source ~/shibu/work/plume/openusp/scripts/make-completion-advanced.bash && exec bash'"
```

## Examples

```bash
# Quick development workflow with completion
make infra-up<TAB>       # Completes to infra-up
make build-all<TAB>      # Completes to build-all
make start-ser<TAB>      # Completes to start-services

# Using helper function
make-help start          # Shows all start-* targets with descriptions
make-help agent          # Shows agent-related targets

# Context-aware completion
make build-<TAB><TAB>    # Shows only: build-all, build-services, build-agents, build-*
make start-<TAB><TAB>    # Shows only: start-all, start-services, start-*
```

This completion system makes working with the extensive OpenUSP Makefile much more efficient and user-friendly!