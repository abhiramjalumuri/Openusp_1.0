# OpenUSP - Kafka-Based Architecture Quick Reference

## ✅ What's Working Now

### Active Services
```bash
make build-services  # Builds all active services
make run-services    # Runs all active services in background
```

**Services:**
- ✅ `api-gateway` - REST API (port 6500)
- ✅ `data-service` - Data management
- ✅ `usp-service` - USP protocol handler
- ✅ `cwmp-service` - CWMP protocol handler
- ✅ `mtps-stomp` - **Kafka-based STOMP transport** 🆕

### MTP Transport Status

| Transport | Status | Build Command | Path |
|-----------|--------|---------------|------|
| **STOMP** | ✅ Active | `make build-mtps-stomp` | `cmd/mtps/stomp/` |
| MQTT | 🚧 Incomplete | (commented out) | `cmd/mtps/mqtt/` |
| WebSocket | 🚧 Incomplete | (commented out) | `cmd/mtps/websocket/` |
| HTTP | 🚧 Incomplete | (commented out) | `cmd/mtps/http/` |

## 🗑️ What Was Removed

### Deleted Legacy Services
- ❌ `cmd/mtp-service/` - Monolithic
- ❌ `cmd/mtp-stomp/` - gRPC-based
- ❌ `cmd/mtp-services/` - Old structure
- ❌ `cmd/mtp-mqtt/`, `cmd/mtp-websocket/`, `cmd/mtp-http/`, `cmd/mtp-uds/`

### Kept Components
- ✅ `internal/mtp/stomp.go` - **Still needed** (used by mtps-stomp)

## 🔧 Key Configuration

### STOMP Config Structure
```yaml
# configs/openusp.yml
mtp:
  stomp:
    broker_url: "rabbitmq:61613"
    username: "guest"
    password: "guest"
    destinations:
      inbound: "/queue/usp.controller"   # From agents
      outbound: "/queue/usp.agent"       # To agents
      broadcast: "/topic/usp.broadcast"
```

### Code Access
```go
// cmd/mtps/stomp/main.go
cfg.MTP.STOMP.Destinations.Inbound   // ✅ Correct
cfg.MTP.STOMP.Destinations.Outbound  // ✅ Correct
// Not: .Controller or .Agent ❌
```

## 📦 Build System

### Service Definitions
```makefile
# Makefile
OPENUSP_CORE_SERVICES := data-service usp-service cwmp-service
OPENUSP_MTP_SERVICES := mtps-stomp
OPENUSP_SERVICES := api-gateway $(OPENUSP_CORE_SERVICES) $(OPENUSP_MTP_SERVICES)
```

### Build Templates
```makefile
# Kafka-based services use cmd/mtps/<transport>
$(eval $(call MTP_BUILD_TEMPLATE,mtps-stomp,stomp))
```

## 🚀 Common Commands

```bash
# Infrastructure
make infra-up          # Start Kafka, RabbitMQ, PostgreSQL, etc.
make infra-status      # Check infrastructure status

# Build
make build-services    # Build all active services
make build-mtps-stomp  # Build STOMP transport only

# Run
make run-services      # Run all services in background
make run-mtps-stomp-background  # Run STOMP only

# Stop
make stop-mtps-stomp   # Stop STOMP service
make stop-all          # Stop all services

# Status
make status-services   # Check running services
```

## 🔍 Troubleshooting

### STOMP Service Won't Build
**Error:** `undefined: mtp.STOMPBroker`
**Fix:** Check that `internal/mtp/stomp.go` exists

### Config Error: "Controller" or "Agent" not found
**Fix:** Use `Inbound`/`Outbound` instead:
```go
cfg.MTP.STOMP.Destinations.Inbound   // ✅
cfg.MTP.STOMP.Destinations.Outbound  // ✅
```

### Other MTP Services Won't Build (MQTT, WebSocket, HTTP)
**Status:** These are commented out in Makefile - implementations incomplete
**Future:** Uncomment when broker implementations are ready

## 📁 Directory Structure

```
cmd/
├── api-gateway/          ✅ Active
├── data-service/         ✅ Active
├── usp-service/          ✅ Active
├── cwmp-service/         ✅ Active
├── agents/
│   ├── usp/             ✅ Active
│   └── cwmp/            ✅ Active
└── mtps/                🆕 Kafka-based MTP services
    ├── stomp/           ✅ Active (fully implemented)
    ├── mqtt/            🚧 Incomplete
    ├── websocket/       🚧 Incomplete
    ├── http/            🚧 Incomplete
    └── uds/             🚧 Incomplete

internal/
└── mtp/
    └── stomp.go         ✅ Used by cmd/mtps/stomp/
```

## ✅ Migration Complete

- **All legacy services removed**
- **Kafka-based STOMP active**
- **Build system updated**
- **Configuration aligned**
- **Ready for production testing**

## 📝 Next Steps

1. Test STOMP end-to-end with agents
2. Implement remaining transports (MQTT, WebSocket, HTTP)
3. Update Docker Compose deployment files
4. Add integration tests

---

**Documentation:** See `KAFKA_BASED_MTP_MIGRATION.md` for detailed migration notes.
