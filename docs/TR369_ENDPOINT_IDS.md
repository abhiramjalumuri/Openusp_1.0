# TR-369 Endpoint ID Authority Schemes

## Overview

This document describes the endpoint ID formats supported by OpenUSP, based on the **BBF TR-369 (User Services Platform)** specification.

## What is an Endpoint ID?

An **Endpoint ID** uniquely identifies a USP entity (Agent or Controller) in the network. It appears in USP Record messages as:
- **`from_id`**: The sender's endpoint ID
- **`to_id`**: The recipient's endpoint ID

## TR-369 Authority Schemes

The TR-369 specification defines multiple authority schemes for endpoint IDs. OpenUSP supports all standard schemes:

### 1. **proto** - Protocol Authority
**Format:** `proto:<protocol>:<authority>`

Used for protocol-specific addressing.

**Examples:**
```
proto::usp.agent.001
proto::usp.agent.bedroom
proto::cwmp.agent.001
proto:1.1::002604889e3b
proto:usp:controller
proto::openusp.controller
```

**Common Usage:**
- Custom USP agents
- Development/testing environments
- Internal protocol routing

---

### 2. **os** - OUI-Serial (Obuspa Standard)
**Format:** `os:<OUI>-<ProductClass>-<SerialNumber>`

Used by **Obuspa** (Open Broadband USP Agent) reference implementation.

**Examples:**
```
os::012345-C25171DF1F85
os::012345-CPE-SN123456
os:AABBCC-Router-00001
```

**Components:**
- **OUI**: Organizationally Unique Identifier (6 hex digits)
- **ProductClass**: Device product class (optional)
- **SerialNumber**: Device serial number

**Common Usage:**
- Obuspa agents (BBF reference implementation)
- Devices without dedicated endpoint registration

---

### 3. **oui** - OUI Serial Number
**Format:** `oui:<OUI>-<SerialNumber>`

Simpler OUI-based identification.

**Examples:**
```
oui:00D09E-123456789
oui:AABBCC-SN987654
```

**Components:**
- **OUI**: 6-character OUI (Organizationally Unique Identifier)
- **SerialNumber**: Device serial number

**Common Usage:**
- IoT devices with manufacturer OUI
- Simple device identification

---

### 4. **cid** - Company/Custom ID
**Format:** `cid:<CompanyID>`

Custom company-specific identifier.

**Examples:**
```
cid:3561-MyDevice
cid:ACME-CPE-001
```

**Common Usage:**
- Vendor-specific device IDs
- Custom deployment schemes

---

### 5. **uuid** - Universally Unique Identifier
**Format:** `uuid:<UUID>`

Standard UUID (RFC 4122) format.

**Examples:**
```
uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6
uuid:123e4567-e89b-12d3-a456-426614174000
```

**Common Usage:**
- Cloud-based deployments
- Multi-vendor environments
- Globally unique identification

---

### 6. **imei** - International Mobile Equipment Identity
**Format:** `imei:<IMEI>`

15-digit IMEI number for mobile devices.

**Examples:**
```
imei:990000862471854
imei:123456789012345
```

**Common Usage:**
- Mobile broadband devices
- LTE/5G CPE equipment
- Cellular IoT devices

---

### 7. **imeisv** - IMEI Software Version
**Format:** `imeisv:<IMEISV>`

16-digit IMEI with software version.

**Examples:**
```
imeisv:9900008624718540
imeisv:1234567890123456
```

**Common Usage:**
- Mobile devices with software version tracking
- 5G equipment management

---

### 8. **ops** - Operations Support System ID
**Format:** `ops:<OPS-ID>`

OSS/BSS system identifier.

**Examples:**
```
ops:1234567890
ops:PROVIDER-001
```

**Common Usage:**
- Service provider deployments
- OSS/BSS integration
- Billing system integration

---

### 9. **self** - Self-Reference
**Format:** `self::self`

Special identifier for controller self-identification.

**Example:**
```
self::self
```

**Common Usage:**
- Controller endpoint identification
- Inter-controller communication

---

## Implementation in OpenUSP

### MTP Services (WebSocket, STOMP, MQTT)

All MTP services extract endpoint IDs from incoming USP Records using the `extractEndpointID()` function, which:

1. **Scans for authority schemes** in this order:
   ```go
   authoritySchemes := []string{
       "proto:",   // Most common for agents
       "os:",      // Obuspa agents
       "oui:",     // OUI-based
       "cid:",     // Company ID
       "uuid:",    // UUID
       "imei:",    // IMEI
       "imeisv:",  // IMEISV
       "ops:",     // OSS/BSS
       "self::",   // Controller
   }
   ```

2. **Extracts valid characters**: `a-z`, `A-Z`, `0-9`, `:`, `.`, `-`, `_`

3. **Filters controller endpoints**: Ignores endpoints containing "controller" or "openusp" (these are `to_id`, not `from_id`)

4. **Returns agent endpoint ID** for routing and caching

5. **Rejects invalid messages**: If endpoint ID cannot be extracted, returns error and drops message (no fallback)

### Code Location

- **WebSocket MTP**: `cmd/mtp-services/websocket/main.go::extractEndpointID()`
- **STOMP MTP**: `cmd/mtp-services/stomp/main.go::extractEndpointID()`
- **MQTT MTP**: `cmd/mtp-services/mqtt/main.go::extractEndpointID()`

### Example Logs

**Success:**
```
2025/10/17 23:45:32 📥 STOMP: Received USP message from agent (80 bytes)
2025/10/17 23:45:32 📤 Published message to topic: usp.messages.inbound (EndpointID: os::012345-C25171DF1F85, Queue: /queue/controller->agent)
```

**Failure (Invalid Message):**
```
2025/10/18 10:15:42 📥 STOMP: Received USP message from agent (65 bytes)
2025/10/18 10:15:42 ❌ STOMP: Failed to extract endpoint ID from USP Record - invalid message
2025/10/18 10:15:42 ❌ STOMP: Message processing failed: endpoint ID (from_id) is mandatory but not found in USP Record
```

## Validation Rules

### Valid Endpoint ID Characteristics

✅ **Must:**
- Start with a valid authority scheme (`proto:`, `os:`, `oui:`, etc.)
- Contain only valid characters: `a-z`, `A-Z`, `0-9`, `:`, `.`, `-`, `_`
- Be unique within the deployment
- Be stable across reboots (for persistent device identification)
- **Be present in every USP Record** (mandatory field per TR-369)

❌ **Must Not:**
- Contain spaces or special characters (except `:`, `.`, `-`, `_`)
- Be empty or null
- Exceed reasonable length (typically < 60 characters)
- Match controller endpoints (for agent extraction)

### Message Rejection Policy

**If endpoint ID extraction fails:**
- ❌ Message is **immediately rejected**
- ❌ No fallback values used
- ❌ No further processing occurs
- ❌ Error logged and returned to MTP layer
- ✅ Enforces TR-369 compliance
- ✅ Prevents silent failures and routing issues

**Previous Behavior (Deprecated):**
```
EndpointID: stomp-client-unknown   ❌ No longer used
EndpointID: ws-client-<UUID>       ❌ No longer used
EndpointID: mqtt-client-unknown    ❌ No longer used
```

✅ **Must:**
- Start with a valid authority scheme (`proto:`, `os:`, `oui:`, etc.)
- Contain only valid characters: `a-z`, `A-Z`, `0-9`, `:`, `.`, `-`, `_`
- Be unique within the deployment
- Be stable across reboots (for persistent device identification)

❌ **Must Not:**
- Contain spaces or special characters (except `:`, `.`, `-`, `_`)
- Be empty or null
- Exceed reasonable length (typically < 60 characters)
- Match controller endpoints (for agent extraction)

## Testing Endpoint IDs

### Test with Different Schemes

1. **Proto Scheme** (Custom Agent):
   ```bash
   # Agent config
   endpoint_id: "proto::usp.agent.test001"
   
   # Expected in logs
   EndpointID: proto::usp.agent.test001
   ```

2. **OS Scheme** (Obuspa):
   ```bash
   # Obuspa config
   EndpointID: "os::012345-C25171DF1F85"
   
   # Expected in logs
   EndpointID: os::012345-C25171DF1F85
   ```

3. **OUI Scheme**:
   ```bash
   endpoint_id: "oui:00D09E-123456789"
   
   # Expected in logs
   EndpointID: oui:00D09E-123456789
   ```

4. **UUID Scheme**:
   ```bash
   endpoint_id: "uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6"
   
   # Expected in logs
   EndpointID: uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6
   ```

### Verify Extraction

```bash
# Check MTP logs for endpoint extraction
tail -f logs/mtp-stomp.log | grep "EndpointID:"
tail -f logs/mtp-websocket.log | grep "EndpointID:"

# Should see actual endpoint ID, NOT fallback values
```

### Check Redis Cache

```bash
redis-cli -h localhost -p 6379 -a openusp123
> KEYS agent:connection:*
> HGETALL agent:connection:os::012345-C25171DF1F85
```

## Troubleshooting

### Problem: "Failed to extract endpoint ID from USP Record - invalid message"

**Cause:** Endpoint ID extraction failed - message rejected

**Severity:** ❌ **CRITICAL** - Message will be dropped, no further processing

**Why:** The `from_id` field in a USP Record is **mandatory** per TR-369 specification. Without it, the Controller cannot:
- Route responses back to the agent
- Track agent connections in Redis
- Register the device in the database
- Identify which agent sent the message

**Solutions:**
1. **Verify USP Record structure**: Check that `from_id` field is populated
   ```bash
   # Inspect raw protobuf data
   tail -f logs/mtp-stomp.log | grep -A 5 "Received MESSAGE"
   ```

2. **Validate with protobuf decoder**:
   ```bash
   protoc --decode=Record usp-msg-1-3.proto < message.bin
   # Should see: from_id: "proto::usp.agent.001" (or similar)
   ```

3. **Check agent configuration**: Ensure agent has valid endpoint ID configured
   ```yaml
   # Example agent config
   endpoint_id: "proto::usp.agent.001"
   # OR for obuspa
   endpoint_id: "os::012345-C25171DF1F85"
   ```

4. **Verify TR-369 compliance**: Endpoint ID must use a valid authority scheme:
   - `proto:`, `os:`, `oui:`, `cid:`, `uuid:`, `imei:`, `imeisv:`, `ops:`, `self::`

5. **Check MTP logs for extraction attempts**:
   ```bash
   # WebSocket
   tail -f logs/mtp-websocket.log | grep "extract endpoint"
   
   # STOMP
   tail -f logs/mtp-stomp.log | grep "extract endpoint"
   
   # MQTT  
   tail -f logs/mtp-mqtt.log | grep "extract endpoint"
   ```

**No Fallback Values:**
- Previous versions used fallback values like `"stomp-client-unknown"` or `"ws-client-<UUID>"`
- **These are now removed** - invalid messages are rejected immediately
- This enforces TR-369 compliance and prevents silent failures

### Problem: Endpoint ID truncated or malformed

**Cause:** Invalid characters or parsing issue

**Solutions:**
1. Check for special characters in endpoint ID
2. Verify total length < 60 characters
3. Ensure no embedded null bytes or control characters

## References

- **BBF TR-369**: User Services Platform Specification
- **RFC 4122**: UUID Standard
- **Obuspa**: Open Broadband USP Agent (uses `os::` scheme)
- **OpenUSP Docs**: [USP Connect Flow](./USP_CONNECT_FLOW.md)

## Related Files

- `cmd/mtp-services/stomp/main.go` - STOMP endpoint extraction
- `cmd/mtp-services/websocket/main.go` - WebSocket endpoint extraction
- `pkg/redis/agent_connection.go` - Connection tracking by endpoint ID
- `internal/database/models.go` - Device model with endpoint_id field
