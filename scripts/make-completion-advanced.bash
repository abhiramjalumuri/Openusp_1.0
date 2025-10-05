#!/bin/bash
# =============================================================================
# Advanced OpenUSP Make Completion with Help Integration
# Enhanced bash completion with target descriptions and smart suggestions
# =============================================================================

_openusp_make_completion() {
    local cur prev opts base_targets service_targets agent_targets infra_targets
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Only complete for 'make' command
    if [[ "${COMP_WORDS[0]}" != "make" ]]; then
        return 0
    fi

    # Find Makefile location
    local makefile_dir
    for dir in . .. ../.. ~/shibu/work/plume/openusp; do
        if [[ -f "$dir/Makefile" ]]; then
            makefile_dir="$dir"
            break
        fi
    done

    if [[ -z "$makefile_dir" ]]; then
        return 0
    fi

    # Define target categories for better organization
    base_targets="help version endpoints docs quick-start start-all stop-all clean-all build-all"
    
    infra_targets="infra-up infra-down infra-status infra-clean consul-status consul-clean"
    
    service_targets="build-services start-services stop-services clean-services \
                    build-api-gateway start-api-gateway stop-api-gateway \
                    build-data-service start-data-service stop-data-service \
                    build-usp-service start-usp-service stop-usp-service \
                    build-mtp-service start-mtp-service stop-mtp-service \
                    build-cwmp-service start-cwmp-service stop-cwmp-service \
                    build-connection-manager start-connection-manager stop-connection-manager"
    
    agent_targets="build-agents clean-agents \
                  build-usp-agent start-usp-agent stop-usp-agent logs-usp-agent \
                  build-cwmp-agent start-cwmp-agent stop-cwmp-agent logs-cwmp-agent"

    # Development and testing targets
    dev_targets="run-dev-swagger test test-unit test-integration test-parsing test-database \
                logs-all status-all"

    # Combine all targets
    all_targets="$base_targets $infra_targets $service_targets $agent_targets $dev_targets"

    # Smart completion based on current input
    case "$cur" in
        infra*)
            opts="$infra_targets"
            ;;
        build*)
            opts="build-all build-services build-agents $service_targets $agent_targets"
            opts=$(echo "$opts" | tr ' ' '\n' | grep '^build-' | tr '\n' ' ')
            ;;
        start*)
            opts="start-all start-services $service_targets $agent_targets"
            opts=$(echo "$opts" | tr ' ' '\n' | grep '^start-' | tr '\n' ' ')
            ;;
        stop*)
            opts="stop-all stop-services $service_targets $agent_targets"
            opts=$(echo "$opts" | tr ' ' '\n' | grep '^stop-' | tr '\n' ' ')
            ;;
        clean*)
            opts="clean-all clean-services clean-agents"
            ;;
        logs*)
            opts="logs-all logs-usp-agent logs-cwmp-agent"
            ;;
        test*)
            opts="test test-unit test-integration test-parsing test-database"
            ;;
        *)
            # Extract actual targets from Makefile for completeness
            local makefile_targets
            makefile_targets=$(grep -E '^[a-zA-Z0-9_-]+:' "$makefile_dir/Makefile" | \
                              sed 's/:.*$//' | \
                              grep -v '^#' | \
                              sort -u)
            
            opts="$all_targets $makefile_targets"
            ;;
    esac

    # Remove duplicates and generate completions
    opts=$(echo "$opts" | tr ' ' '\n' | sort -u | tr '\n' ' ')
    COMPREPLY=($(compgen -W "${opts}" -- ${cur}))
    
    return 0
}

# Register the enhanced completion
complete -F _openusp_make_completion make

# =============================================================================
# Helper Function: Show Make Target Descriptions
# Usage: make-help [pattern]
# =============================================================================
make-help() {
    local pattern="${1:-.*}"
    local makefile_dir
    
    # Find Makefile
    for dir in . .. ../.. ~/shibu/work/plume/openusp; do
        if [[ -f "$dir/Makefile" ]]; then
            makefile_dir="$dir"
            break
        fi
    done
    
    if [[ -z "$makefile_dir" ]]; then
        echo "❌ Makefile not found"
        return 1
    fi
    
    echo "🎯 OpenUSP Make Targets (matching: $pattern)"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Extract targets and their descriptions from comments
    grep -E '^[a-zA-Z0-9_-]+:|^##' "$makefile_dir/Makefile" | \
    sed -n 'N;s/^## \(.*\)\n\([^:]*\):/\2|\1/p;s/^\([^:]*\):.*$/\1|/p' | \
    grep -E "^[^|]*$pattern" | \
    while IFS='|' read -r target desc; do
        if [[ -n "$desc" ]]; then
            printf "  %-25s %s\n" "$target" "$desc"
        else
            printf "  %-25s\n" "$target"
        fi
    done
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "💡 Usage: make <target>  |  Tab completion available"
    echo "📖 Full help: make help"
}

# Export the helper function
export -f make-help

echo "✅ OpenUSP Make completion loaded!"
echo "💡 Usage: make <TAB><TAB> to see available targets"
echo "📖 Helper: make-help [pattern] to see target descriptions"