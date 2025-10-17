# Redis-Based MTP Routing Architecture

## Overview

The USP service uses **Redis as the single source of truth** for agent connection state and MTP routing information. This enables the Controller to send messages to agents regardless of which service instance handled the initial connection.

## Architecture

```
┌─────────────┐
│   Agent     │
│ (endpoint1) │
└──────┬──────┘
       │ 1. Connect via WebSocket
       ▼
┌─────────────────┐
│  MTP-WebSocket  │
│    Service      │
└────────┬────────┘
         │ 2. Publish USPMessageEvent to Kafka
         │    (includes MTPDestination)
         ▼
┌──────────────────────┐
│   Kafka Topic        │
│ usp.messages.inbound │
└──────────┬───────────┘
           │
           ▼
┌──────────────────────────────────────────────────────────┐
│              USP Service                                  │
│                                                           │
│  3. handleUSPMessage()                                    │
│     ┌─────────────────────────────────────┐              │
│     │ Extract event.EndpointID            │              │
│     │ Extract event.MTPProtocol           │              │
│     │ Extract event.MTPDestination        │              │
│     └──────────────┬──────────────────────┘              │
│                    │                                      │
│  4. ProcessUSPMessageWithEvent()                          │
│     ┌──────────────▼──────────────────────┐              │
│     │ If Connect Record:                  │              │
│     │   handleAgentConnect()              │              │
│     │   ├─► Store to Redis ✓              │              │
│     │   ├─► Persist to DB                 │              │
│     │   ├─► Auto-register device          │              │
│     │   ├─► Publish connection event      │              │
│     │   └─► Return                        │              │
│     └─────────────────────────────────────┘              │
│                                                           │
│  5. If Response needed:                                   │
│     ┌─────────────────────────────────────┐              │
│     │ getEndpointMTPRouting(endpoint1)    │              │
│     │   └─► Query Redis ✓                 │              │
│     │   └─► Get MTPDestination            │              │
│     └──────────────┬──────────────────────┘              │
│                    │                                      │
│  6. PublishUSPMessageWithDestination()                    │
│     └─► Kafka: usp.messages.outbound                     │
└───────────────────────────┬──────────────────────────────┘
                            │
                            ▼
                 ┌──────────────────────┐
                 │   Kafka Topic        │
                 │ usp.messages.outbound│
                 └──────────┬───────────┘
                            │
                            ▼
                 ┌─────────────────┐
                 │  MTP-WebSocket  │
                 │    Service      │
                 └────────┬────────┘
                          │ 7. Send via WebSocket
                          ▼
                 ┌─────────────┐
                 │   Agent     │
                 │ (endpoint1) │
                 └─────────────┘
```

## Redis Schema

### Connection Information
**Key:** `agent:connection:<endpoint_id>`  
**Type:** Hash  
**Fields:**
```json
{
  "endpoint_id": "proto://agent.001",
  "mtp_protocol": "websocket",
  "mtp_destination": {
    "websocket_url": "ws://mtp-websocket:8081/ws",
    "stomp_queue": "",
    "mqtt_topic": ""
  },
  "status": "online",
  "connected_at": "2025-10-17T10:00:00Z",
  "last_activity": "2025-10-17T10:05:00Z"
}
```

**TTL:** Auto-expires after inactivity threshold (configurable)

## Flow Details

### 1. Agent Connect Flow

When an agent connects via MTP (WebSocket/STOMP/MQTT):

1. **MTP Service** receives connection
2. **MTP Service** publishes `USPMessageEvent` to Kafka with:
   - `EndpointID`: Agent's endpoint identifier
   - `MTPProtocol`: Protocol type (websocket/stomp/mqtt)
   - `MTPDestination`: Connection details (URL, queue, topic)
   - `Payload`: USP protobuf Record bytes

3. **USP Service** consumes event and:
   - Calls `handleAgentConnect()` if it's a Connect record
   - **Stores connection to Redis** via `redisClient.StoreAgentConnection()`
   - Persists to database for audit trail
   - Auto-registers device if new
   - Publishes connection event to Kafka

### 2. Controller → Agent Message Flow

When the Controller needs to send a message to an agent:

1. **API Gateway** receives REST request
2. **Data Service** publishes message to Kafka
3. **USP Service** processes message:
   - Needs to route to specific agent endpoint
   - Calls `getEndpointMTPRouting(endpointID)`
   - **Queries Redis** via `redisClient.GetAgentConnection()`
   - Gets `MTPDestination` (WebSocket URL, STOMP queue, etc.)
   - Publishes to `usp.messages.outbound` with routing info

4. **MTP Service** consumes outbound message:
   - Extracts `MTPDestination` from event
   - Sends to agent via appropriate protocol

### 3. Connection Monitoring

The `monitorConnections()` goroutine runs every 1 minute:

1. Calls `redisClient.GetAllAgentConnections()`
2. Checks `last_activity` timestamp
3. If inactive > 5 minutes:
   - Updates status to "inactive"
4. If inactive > 10 minutes:
   - Disconnects agent
   - Updates database
   - Publishes disconnect event

## Key Functions

### `handleAgentConnect()`
**Purpose:** Process agent connection (Connect record)  
**Actions:**
- Store connection to Redis
- Check database for existing device
- Auto-register if new device
- Record connection history
- Publish Kafka event

### `getEndpointMTPRouting()`
**Purpose:** Lookup MTP routing for sending messages  
**Actions:**
- Query Redis for agent connection
- Extract MTPDestination
- Return routing info to caller

### `monitorConnections()`
**Purpose:** Periodic health checks  
**Actions:**
- Scan all Redis connections
- Check activity timestamps
- Mark inactive/disconnect as needed

## Configuration

### Redis Settings (configs/openusp.yml)
```yaml
redis:
  host: localhost
  port: 6379
  password: openusp123
  db: 0
```

### Connection Thresholds
```go
const (
    INACTIVE_THRESHOLD = 5 * time.Minute  // Mark as inactive
    DISCONNECT_THRESHOLD = 10 * time.Minute  // Disconnect
    MONITOR_INTERVAL = 1 * time.Minute  // Check frequency
)
```

## Benefits

### ✅ Single Source of Truth
- Redis is authoritative for connection state
- No dual-cache inconsistencies
- All services query same data

### ✅ Distributed System Support
- Multiple USP service instances can scale
- Connection info shared across instances
- No local memory limitations

### ✅ Persistence & Recovery
- Database stores connection history
- Audit trail for connections
- Recovery after Redis restart

### ✅ Real-time Updates
- Immediate connection state changes
- Fast lookups (in-memory Redis)
- Sub-millisecond query times

## Testing

### Verify Redis Storage
```bash
# Check agent connection in Redis
redis-cli -h localhost -p 6379 -a openusp123
> KEYS agent:connection:*
> HGETALL agent:connection:proto://agent.001
```

### Verify Routing Lookup
```bash
# Check USP service logs for routing queries
tail -f logs/usp-service.log | grep "Retrieved MTP routing"
```

### Test Controller → Agent Message
```bash
# Send message via REST API
curl -X POST http://localhost:8080/api/v1/devices/proto://agent.001/get \
  -H "Content-Type: application/json" \
  -d '{"path": "Device.DeviceInfo."}'

# Should see in logs:
# - "📖 Retrieved MTP routing from Redis for proto://agent.001"
# - "✅ Published USP response to usp.messages.outbound"
```

## Migration Notes

### Before (In-Memory Cache)
- ❌ `endpointMTPCache` (sync.RWMutex map)
- ❌ `storeEndpointMTPRouting()` - stored locally
- ❌ Lost on service restart
- ❌ Not shared across instances

### After (Redis Cache)
- ✅ Redis hash: `agent:connection:<endpoint_id>`
- ✅ `handleAgentConnect()` - stores to Redis
- ✅ Persists across restarts (with DB backup)
- ✅ Shared across all service instances

## Troubleshooting

### Agent not receiving messages?
1. Check Redis for connection:
   ```bash
   redis-cli -h localhost -p 6379 -a openusp123 HGETALL agent:connection:<endpoint_id>
   ```

2. Check connection status:
   ```sql
   SELECT * FROM connection_history WHERE endpoint_id = '<endpoint_id>' ORDER BY connected_at DESC LIMIT 1;
   ```

3. Check USP service logs:
   ```bash
   tail -f logs/usp-service.log | grep "<endpoint_id>"
   ```

### Connection marked as inactive/offline?
- Agent must send periodic messages (e.g., periodic Notify)
- Each message updates `last_activity` in Redis
- Monitor checks every 1 minute
- Inactive threshold: 5 minutes
- Disconnect threshold: 10 minutes

### Redis connection lost?
- USP service will log errors
- Database still has connection history
- Agents can reconnect (will repopulate Redis)
- Health endpoint shows Redis status

## References

- [USP Connect Flow](./USP_CONNECT_FLOW.md)
- [Redis Client Package](../pkg/redis/README.md)
- [Connection Handler](../cmd/usp-service/connection_handler.go)
