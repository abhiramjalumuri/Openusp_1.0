# OpenUSP Makefile - TR-369 Agent Version-Specific Targets

## Overview
The OpenUSP Makefile has been enhanced with version-specific targets for the TR-369 agent to support both USP v1.3 and v1.4 protocols.

## New Makefile Targets

### Version-Specific Agent Targets

#### `make start-tr369-v13`
- **Description**: Start TR-369 agent with USP Protocol v1.3 (default)
- **Command**: `build/tr369-agent -version 1.3`
- **Usage**: Launches the agent using the USP v1.3 protocol buffers and message structures
- **Dependencies**: Automatically builds the agent binary if needed

#### `make start-tr369-v14`
- **Description**: Start TR-369 agent with USP Protocol v1.4
- **Command**: `build/tr369-agent -version 1.4`
- **Usage**: Launches the agent using the USP v1.4 protocol buffers and message structures
- **Dependencies**: Automatically builds the agent binary if needed

### Existing Targets (Enhanced)

#### `make start-tr369-agent`
- **Description**: Start TR-369 USP agent (uses default version - v1.3)
- **Command**: `build/tr369-agent` (no version flag = default v1.3)
- **Usage**: Standard agent startup with default protocol version

## Usage Examples

### Quick Testing
```bash
# Test both protocol versions
make start-tr369-v13    # Test USP v1.3
make start-tr369-v14    # Test USP v1.4

# Check help and info
./build/tr369-agent -help
./build/tr369-agent -info
```

### Development Workflow
```bash
# 1. Start OpenUSP platform
make infra-up
make build-all
make start-all

# 2. Test with different USP versions
make start-tr369-v13    # Default protocol
make start-tr369-v14    # Enhanced protocol

# 3. Verify both versions work
# Check logs and responses for version-specific behavior
```

### Manual Testing
```bash
# Direct binary execution
./build/tr369-agent                    # Default v1.3
./build/tr369-agent -version 1.3       # Explicit v1.3
./build/tr369-agent -version 1.4       # USP v1.4
./build/tr369-agent -help             # Show help
./build/tr369-agent -info             # Show agent info
```

## Target Output Examples

### USP v1.3 Output
```
Building TR-369 USP agent...
TR-369 USP agent built -> build/tr369-agent
Starting TR-369 agent with USP Protocol v1.3...
Command: build/tr369-agent -version 1.3
2025/10/01 22:11:11 USP Protocol Version: 1.3
2025/10/01 22:11:12 Sending USP Record (size: 163 bytes, version: 1.3)
✅ TR-369 USP Client demonstration completed!
```

### USP v1.4 Output
```
Building TR-369 USP agent...
TR-369 USP agent built -> build/tr369-agent
Starting TR-369 agent with USP Protocol v1.4...
Command: build/tr369-agent -version 1.4
2025/10/01 22:11:58 USP Protocol Version: 1.4
2025/10/01 22:11:59 Sending USP Record (size: 163 bytes, version: 1.4)
✅ TR-369 USP Client demonstration completed!
```

## Help Output
The Makefile help now includes the new targets:

```
Example Agent Targets:
  build-tr369-agent   Build TR-369 USP agent binary
  build-tr069-agent   Build TR-069 agent binary
  start-tr369-agent   Start TR-369 USP agent (uses OPENUSP_USP_WS_URL)
  start-tr369-v13     Start TR-369 agent with USP v1.3 (default)
  start-tr369-v14     Start TR-369 agent with USP v1.4
  start-tr069-agent   Start TR-069 agent
  Note: All services support --consul flag for service discovery
```

## Implementation Details

### Makefile Changes
1. **Help Section**: Added new targets to help output with clear descriptions
2. **PHONY Targets**: Added version-specific targets to .PHONY declaration
3. **Target Implementation**: Created `start-tr369-v13` and `start-tr369-v14` targets
4. **Dependency Management**: Both targets depend on `build-tr369-agent`
5. **Command Display**: Shows the exact command being executed for transparency

### Target Structure
```makefile
start-tr369-v13: build-tr369-agent
	@echo "Starting TR-369 agent with USP Protocol v1.3..."
	@echo "Command: $(BINARY_DIR)/tr369-agent -version 1.3"
	@$(BINARY_DIR)/tr369-agent -version 1.3 || echo "Agent exited"

start-tr369-v14: build-tr369-agent
	@echo "Starting TR-369 agent with USP Protocol v1.4..."
	@echo "Command: $(BINARY_DIR)/tr369-agent -version 1.4"
	@$(BINARY_DIR)/tr369-agent -version 1.4 || echo "Agent exited"
```

## Benefits

### Development Efficiency
- **Quick Protocol Testing**: Easy switching between USP versions
- **Clear Command Visibility**: Shows exact command being executed
- **Automatic Building**: Ensures binary is built before execution
- **Consistent Interface**: Follows existing Makefile patterns

### Protocol Validation
- **Version Comparison**: Easy comparison of v1.3 vs v1.4 behavior
- **Regression Testing**: Validate both protocols work correctly
- **Feature Testing**: Test version-specific protocol features
- **Compliance Verification**: Ensure both versions meet TR-369 standards

### Operational Benefits
- **Documentation**: Self-documenting commands in help output
- **Error Handling**: Graceful handling of agent exit conditions
- **Integration**: Works seamlessly with existing OpenUSP infrastructure
- **Flexibility**: Supports both manual and automated testing workflows

---

**Status**: ✅ **IMPLEMENTED** - Version-specific Makefile targets for TR-369 agent
**Validation**: Both `make start-tr369-v13` and `make start-tr369-v14` tested and working
**Integration**: Fully integrated with existing OpenUSP Makefile structure and help system