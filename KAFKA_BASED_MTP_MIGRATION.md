# Kafka-Based MTP Services Migration

**Date:** October 16, 2025  
**Status:** ✅ Completed

## Overview

Successfully migrated OpenUSP to use **Kafka-based MTP services** exclusively, removing legacy implementations.

## What Changed

### ✅ Removed Legacy Services

The following legacy directories were **deleted**:

1. **`cmd/mtp-service/`** - Monolithic MTP service with STOMP broker
2. **`cmd/mtp-stomp/`** - gRPC-based STOMP service  
3. **`cmd/mtp-services/`** - Old service structure
4. **`cmd/mtp-mqtt/`** - Legacy MQTT service
5. **`cmd/mtp-websocket/`** - Legacy WebSocket service
6. **`cmd/mtp-http/`** - Legacy HTTP service
7. **`cmd/mtp-uds/`** - Legacy Unix Domain Socket service

### ✅ Active Kafka-Based Services

**Current Structure:** `cmd/mtps/`

| Service | Path | Status | Description |
|---------|------|--------|-------------|
| **STOMP** | `cmd/mtps/stomp/` | ✅ Active | Fully implemented Kafka-based STOMP transport |
| MQTT | `cmd/mtps/mqtt/` | 🚧 Incomplete | Missing MQTTBroker implementation |
| WebSocket | `cmd/mtps/websocket/` | 🚧 Incomplete | Partial implementation |
| HTTP | `cmd/mtps/http/` | 🚧 Incomplete | Partial implementation |

### ✅ Makefile Updates

**Before:**
```makefile
OPENUSP_MTP_SERVICES := mtp-stomp mtp-mqtt mtp-websocket mtp-http
# Build from cmd/mtp-services/
```

**After:**
```makefile
OPENUSP_MTP_SERVICES := mtps-stomp
# Build from cmd/mtps/ (Kafka-based only)
# TODO: mtps-mqtt mtps-websocket mtps-http when ready
```

### ✅ Configuration Alignment

**Fixed STOMP Service** (`cmd/mtps/stomp/main.go`):
- Changed from `cfg.MTP.STOMP.Destinations.Controller/Agent`
- To use correct config: `cfg.MTP.STOMP.Destinations.Inbound/Outbound`

**Config Structure** (`pkg/config/config.go`):
```go
type STOMPTransportConfig struct {
    BrokerURL    string
    Username     string
    Password     string
    Destinations struct {
        Inbound   string // messages from agents (inbound to controller)
        Outbound  string // messages to agents (outbound from controller)
        Broadcast string
    }
}
```

### ✅ Kept Components

**`internal/mtp/stomp.go`** - **Retained** ✅
- Reason: Used by active Kafka-based `cmd/mtps/stomp/` service
- Provides reusable STOMP protocol implementation (`STOMPBroker`)
- Correct architecture: internal package for shared protocol logic

## Build Verification

```bash
make build-services
```

**Output:**
```
✅ api-gateway built successfully
✅ data-service built successfully
✅ usp-service built successfully
✅ cwmp-service built successfully
✅ mtps-stomp built successfully
✅ All OpenUSP services built successfully
```

## Running Services

```bash
make run-services
```

**Active Services:**
- 🌐 API Gateway (port 6500)
- 🗄️ Data Service
- 📡 USP Service
- 📞 CWMP Service
- 🚀 **mtps-stomp** (Kafka-based STOMP transport)

## Infrastructure

All services use **Kafka** for message transport:
- **Kafka Brokers:** Defined in `configs/openusp.yml`
- **Network:** `deployments_openusp-dev`
- **No gRPC** dependencies for MTP services

## Future Work

### 🚧 Complete Remaining MTP Services

To enable MQTT, WebSocket, and HTTP transports:

1. **MQTT:** Create `internal/mtp/mqtt.go` with `MQTTBroker` implementation
2. **WebSocket:** Complete implementation in `cmd/mtps/websocket/`
3. **HTTP:** Complete implementation in `cmd/mtps/http/`

Then uncomment in Makefile:
```makefile
OPENUSP_MTP_SERVICES := mtps-stomp mtps-mqtt mtps-websocket mtps-http
```

### 📝 Update Deployment Files

**Note:** Docker Compose production/test files still reference old `mtp-service`:
- `deployments/environments/docker-compose.prod.yml`
- `deployments/environments/docker-compose.test.yml`

**Action Required:** Update these when deploying to production.

## Architecture Benefits

### ✅ Kafka-Based Advantages
- **Unified Message Bus:** All MTP services use Kafka
- **Scalability:** Services can scale independently
- **Decoupled:** No direct service-to-service communication
- **Fault Tolerance:** Kafka handles message persistence

### ✅ Code Organization
- **Clear Separation:** `cmd/mtps/` for Kafka-based services only
- **Reusable Libraries:** `internal/mtp/` for protocol implementations
- **No Legacy Code:** Old monolithic/gRPC implementations removed

## Testing Checklist

- [x] All core services build successfully
- [x] mtps-stomp service builds
- [x] Configuration uses correct field names (Inbound/Outbound)
- [x] internal/mtp/stomp.go verified as still needed
- [x] Makefile updated to use cmd/mtps/ paths
- [ ] Integration testing with STOMP agents
- [ ] Deploy to test environment

## Files Modified

1. ✏️ `Makefile` - Updated service list and build paths
2. ✏️ `cmd/mtps/stomp/main.go` - Fixed config field names
3. 🗑️ Deleted 7 legacy service directories

## Next Steps

1. ✅ **Verify STOMP service works end-to-end** with agents
2. 🚧 **Implement MQTT broker** (`internal/mtp/mqtt.go`)
3. 🚧 **Complete WebSocket/HTTP** implementations
4. 📝 **Update Docker Compose** deployment files
5. 🧪 **Add integration tests** for Kafka-based MTP services

---

**Summary:** Successfully consolidated to Kafka-based MTP architecture. Only STOMP is active; other transports await full implementation.
