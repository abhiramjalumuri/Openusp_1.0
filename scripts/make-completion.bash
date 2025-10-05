#!/bin/bash
# =============================================================================
# OpenUSP Make Target Bash Completion
# Provides tab completion for all Make targets in the OpenUSP project
# =============================================================================

_make_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Only complete for 'make' command
    if [[ "${COMP_WORDS[0]}" != "make" ]]; then
        return 0
    fi

    # Get the directory where the Makefile is located
    local makefile_dir
    if [[ -f "Makefile" ]]; then
        makefile_dir="."
    elif [[ -f "../Makefile" ]]; then
        makefile_dir=".."
    else
        # Try to find Makefile in common locations
        for dir in . .. ../.. ~/shibu/work/plume/openusp; do
            if [[ -f "$dir/Makefile" ]]; then
                makefile_dir="$dir"
                break
            fi
        done
    fi

    # If no Makefile found, return
    if [[ -z "$makefile_dir" ]]; then
        return 0
    fi

    # Extract all targets from Makefile
    # This matches lines that start with word characters followed by ':'
    # and excludes internal/variable targets
    local targets
    targets=$(grep -E '^[a-zA-Z0-9_-]+:' "$makefile_dir/Makefile" | \
              sed 's/:.*$//' | \
              grep -v '^#' | \
              sort -u)

    # Additional common make targets that might be implicit
    local common_targets="all clean install test help"
    
    # Combine all targets
    opts="$targets $common_targets"

    # Generate completions
    COMPREPLY=($(compgen -W "${opts}" -- ${cur}))
    return 0
}

# Register the completion function for 'make' command
complete -F _make_completion make

# =============================================================================
# OpenUSP Specific Make Targets (for reference)
# =============================================================================
# 
# Infrastructure:
#   infra-up, infra-down, infra-status, infra-clean
#   consul-status, consul-clean
#
# Building:
#   build-all, build-services, build-agents
#   build-api-gateway, build-data-service, build-usp-service
#   build-mtp-service, build-cwmp-service, build-connection-manager
#   build-usp-agent, build-cwmp-agent
#
# Services:
#   start-all, stop-all, clean-all
#   start-services, stop-services, clean-services
#   start-api-gateway, start-data-service, start-usp-service
#   start-mtp-service, start-cwmp-service, start-connection-manager
#
# Agents:
#   start-usp-agent, start-cwmp-agent
#   stop-usp-agent, stop-cwmp-agent
#   logs-usp-agent, logs-cwmp-agent
#
# Development:
#   quick-start, run-dev-swagger
#   version, endpoints, docs
#   logs-all, status-all
#
# Testing:
#   test, test-unit, test-integration
#   test-parsing, test-database
# =============================================================================