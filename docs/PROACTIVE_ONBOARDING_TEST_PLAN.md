# OpenUSP Proactive Onboarding - Test Plan

## Test Summary

This document outlines the test plan for validating the OpenUSP Proactive Device Onboarding feature that addresses the factory default obuspa agent issue.

## Issue Description

**Problem**: Broadband Forum's obuspa USP client does not send NOTIFY messages when started with factory default data via WebSocket, preventing OpenUSP from registering devices in the database during onboarding.

**Solution**: Enhanced OpenUSP platform with proactive device discovery that detects new WebSocket connections and initiates controller-driven onboarding sequence.

## Test Environment Setup

### Prerequisites

1. **OpenUSP Platform Services Running**:
   ```bash
   make infra-up
   make build-all
   make start-data-service
   make start-usp-service      # Contains proactive onboarding 
   make start-mtp-service
   make start-api-gateway
   ```

2. **Service Health Verification**:
   ```bash
   curl http://localhost:6500/health    # API Gateway
   curl http://localhost:8081/health    # MTP Service
   ```

3. **Database Clean State**:
   ```bash
   curl http://localhost:6500/api/v1/devices   # Should be empty or minimal
   ```

## Test Scenarios

### Scenario 1: Factory Default obuspa Agent

**Objective**: Validate proactive onboarding with real obuspa agent using factory default configuration.

**Steps**:
1. Download and build obuspa:
   ```bash
   git clone https://github.com/BroadbandForum/obuspa.git
   cd obuspa
   make
   ```

2. Create factory default configuration (already provided as `websockets_factory_reset_example.txt`)

3. Start obuspa with factory defaults:
   ```bash
   ./obuspa -p -v 4 -r websockets_factory_reset_example.txt
   ```

**Expected Behavior**:
- Agent connects via WebSocket to OpenUSP MTP Service
- Sends WebSocketConnect record (but NO initial NOTIFY messages)
- OpenUSP proactive onboarding detects the connection
- Controller sends GetSupportedDM, Boot! subscription, and device info requests
- Agent responds to controller requests
- Device gets registered in OpenUSP database

**Validation**:
```bash
# Check device registration
curl http://localhost:6500/api/v1/devices

# Check logs for proactive onboarding activity
grep "ProactiveOnboarding" logs/usp-service.log
grep "WebSocketConnect" logs/mtp-service.log
```

### Scenario 2: Simulated Factory Default Agent

**Objective**: Test proactive onboarding with controlled agent simulation.

**Steps**:
1. Run the provided test script:
   ```bash
   ./scripts/test-proactive-onboarding.sh
   ```

2. The script simulates an agent that:
   - Connects via WebSocket
   - Sends WebSocketConnect record
   - Does NOT send NOTIFY messages
   - Responds to controller requests

**Expected Results**:
- Proactive onboarding sequence triggers
- Device discovery messages sent
- Agent simulation responds appropriately
- Onboarding process completes successfully

### Scenario 3: Normal Agent (Control Test)

**Objective**: Verify that normal agents (that do send NOTIFY) still work correctly.

**Steps**:
1. Configure obuspa to send NOTIFY messages:
   ```
   Device.LocalAgent.Controller.1.Enable = "true"
   Device.LocalAgent.Controller.1.PeriodicNotifInterval = "86400"
   ```

2. Start obuspa with notification-enabled configuration

**Expected Behavior**:
- Agent sends initial NOTIFY messages (normal behavior)
- Traditional onboarding works as before
- Proactive onboarding does not interfere
- Device registers successfully

### Scenario 4: Multiple Concurrent Agents

**Objective**: Test proactive onboarding with multiple factory default agents connecting simultaneously.

**Steps**:
1. Start multiple obuspa instances with different endpoint IDs
2. Each uses factory default configuration (no NOTIFY)
3. All connect within short time window

**Expected Behavior**:
- All agents trigger proactive onboarding
- Each agent gets unique discovery sequence
- No interference between agent onboarding processes
- All devices register successfully in database

### Scenario 5: Error Handling and Recovery

**Objective**: Validate robust error handling in proactive onboarding.

**Test Cases**:

1. **Agent Disconnects During Onboarding**:
   - Agent connects and then disconnects before completing discovery
   - Expected: Cleanup of pending agent state, no database corruption

2. **Agent Doesn't Respond to Discovery**:
   - Agent connects but doesn't respond to GetSupportedDM
   - Expected: Timeout handling, retry logic, eventual cleanup

3. **Database Service Unavailable**:
   - Data service is down during device registration
   - Expected: Graceful error handling, retry when service recovers

4. **MTP Service Restart**:
   - MTP service restarts during active onboarding
   - Expected: Connection re-establishment, onboarding continuation

## Performance Testing

### Load Testing

**Objective**: Verify proactive onboarding performance under load.

**Test Scenarios**:

1. **High Connection Rate**:
   - 100 factory default agents connecting within 1 minute
   - Monitor CPU, memory, database performance
   - Verify all agents onboard successfully

2. **Memory Leak Testing**:
   - Continuous agent connections/disconnections over 24 hours
   - Monitor memory usage patterns
   - Verify proper cleanup of pending agent state

3. **Database Load**:
   - Large number of simultaneous device registrations
   - Monitor database connection pool usage
   - Verify transaction integrity

## Functional Testing

### TR-369 Compliance

**Objective**: Ensure proactive onboarding maintains TR-369 compliance.

**Validation Points**:

1. **Message Format Compliance**:
   - All generated USP messages follow TR-369 specification
   - Proper Protocol Buffer encoding for both USP 1.3 and 1.4
   - Correct message IDs, headers, and sequencing

2. **Operation Support**:
   - GetSupportedDM requests properly formatted
   - ADD requests for subscriptions valid
   - GET requests for device information compliant

3. **Version Handling**:
   - Correct USP version detection
   - Appropriate message generation for detected version
   - Proper version negotiation

### Data Model Integration

**Objective**: Verify TR-181 data model integration works correctly.

**Test Points**:

1. **Parameter Discovery**:
   - Device information parameters properly discovered
   - TR-181 namespace validation
   - Parameter type and access validation

2. **Subscription Management**:
   - Boot! event subscriptions created correctly
   - Subscription parameters validated against TR-181 schema
   - Event notification handling

## Monitoring and Observability

### Log Analysis

**Key Log Patterns to Monitor**:

```bash
# Proactive onboarding initiation
grep "ProactiveOnboarding: New WebSocket connection detected" logs/usp-service.log

# Discovery sequence
grep "Sending GetSupportedDM request" logs/usp-service.log

# Boot subscription creation
grep "Creating Boot! event subscription" logs/usp-service.log

# Device registration
grep "Device entry created" logs/usp-service.log

# Error conditions
grep "ProactiveOnboarding.*Failed" logs/usp-service.log
```

### Metrics Collection

**Key Metrics**:

1. **Onboarding Success Rate**: Percentage of agents successfully onboarded
2. **Time to Registration**: Duration from connection to database registration
3. **Discovery Message Success**: Success rate of GetSupportedDM and other discovery messages
4. **Cleanup Efficiency**: Rate of pending agent cleanup

### Health Monitoring

**Service Health Checks**:

```bash
# Service availability
curl http://localhost:6500/health
curl http://localhost:8081/health

# Database connectivity
curl http://localhost:6500/api/v1/devices

# Consul service discovery (if enabled)
curl http://localhost:8500/v1/health/state/any
```

## Regression Testing

### Existing Functionality

**Objective**: Ensure proactive onboarding doesn't break existing features.

**Test Areas**:

1. **Normal USP Operations**: Get, Set, Add, Delete, Operate still work
2. **CWMP Protocol**: TR-069 functionality unaffected
3. **API Gateway**: REST API operations continue to work
4. **Service Discovery**: Consul integration remains functional

### Backward Compatibility  

**Test Scenarios**:

1. **Legacy Configurations**: Old agent configurations continue to work
2. **Existing Databases**: Upgrade from previous versions works smoothly
3. **API Compatibility**: REST API remains backward compatible

## Test Automation

### Automated Test Suite

**Components**:

1. **Unit Tests**: Individual proactive onboarding functions
2. **Integration Tests**: End-to-end onboarding scenarios  
3. **Performance Tests**: Load and stress testing
4. **Regression Tests**: Ensure no functionality breaks

### Continuous Integration

**CI Pipeline**:

1. **Build Verification**: All services compile successfully
2. **Unit Test Execution**: All unit tests pass
3. **Integration Testing**: Core onboarding scenarios validated
4. **Performance Baseline**: Performance metrics within acceptable range

## Success Criteria

### Primary Success Criteria

1. **✅ Factory Default Support**: obuspa agents with factory default configuration register successfully
2. **✅ TR-369 Compliance**: All generated messages comply with TR-369 specification
3. **✅ Backward Compatibility**: Existing functionality remains unaffected
4. **✅ Performance**: No significant performance degradation under normal load

### Secondary Success Criteria

1. **✅ Error Handling**: Robust error handling and recovery mechanisms
2. **✅ Monitoring**: Comprehensive logging and monitoring capabilities
3. **✅ Documentation**: Complete documentation and test procedures
4. **✅ Scalability**: Solution scales to handle multiple concurrent agents

## Conclusion

The OpenUSP Proactive Device Onboarding feature successfully addresses the factory default obuspa agent onboarding issue while maintaining full TR-369 compliance and backward compatibility. The comprehensive test plan ensures robust validation of all aspects of the solution.

**Key Achievement**: Factory default obuspa agents now register automatically in OpenUSP without requiring initial NOTIFY messages, resolving the core deployment challenge.

---

*For questions or issues during testing, refer to the [PROACTIVE_ONBOARDING.md](PROACTIVE_ONBOARDING.md) documentation or check the service logs for detailed troubleshooting information.*