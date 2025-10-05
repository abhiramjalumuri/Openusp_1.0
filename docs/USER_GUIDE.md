# 👥 User Guide

Learn how to use OpenUSP for device management and monitoring.

## Overview

OpenUSP provides a comprehensive platform for managing TR-369 compliant devices through web interfaces, REST APIs, and various transport protocols.

## Getting Started

### Access the Web Interface

After starting OpenUSP (see [Quick Start](QUICKSTART.md)), open:
- **Main Dashboard**: http://localhost:6500/swagger/index.html

### Key Concepts

- **Device**: A TR-369 compliant device (router, modem, IoT device)
- **USP Agent**: Software running on the device
- **USP Controller**: OpenUSP acts as the controller
- **MTP**: Message Transport Protocol (WebSocket, MQTT, etc.)
- **Data Model**: TR-181 Device:2 hierarchical parameter structure

## Device Management

### Adding a Device

#### Via Web Interface
1. Open Swagger UI: http://localhost:6500/swagger/index.html
2. Navigate to **Devices** section
3. Use `POST /api/v1/devices` endpoint
4. Fill in device details:

```json
{
  "device_id": "device001",
  "endpoint_id": "device001::controller",
  "device_type": "residential_gateway",
  "manufacturer": "ExampleCorp",
  "model": "RG-1000",
  "software_version": "1.0.0"
}
```

#### Via API Call
```bash
curl -X POST http://localhost:6500/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device001",
    "endpoint_id": "device001::controller",
    "device_type": "residential_gateway",
    "manufacturer": "ExampleCorp",
    "model": "RG-1000",
    "software_version": "1.0.0"
  }'
```

### Viewing Devices

```bash
# Get all devices
curl http://localhost:6500/api/v1/devices

# Get specific device
curl http://localhost:6500/api/v1/devices/device001
```

### Device Status and Health

Monitor device connectivity and health through:
- **Dashboard**: Real-time status indicators
- **Health Endpoint**: `/api/v1/devices/{id}/health`
- **Logs**: Device communication logs

## Parameter Management

### Understanding Parameters

TR-181 parameters are organized hierarchically:
```
Device.
├── DeviceInfo.
│   ├── Manufacturer
│   ├── ModelName
│   └── SoftwareVersion
├── WiFi.
│   ├── Radio.{i}.
│   └── SSID.{i}.
└── Ethernet.
    └── Interface.{i}.
```

### Reading Parameters

#### Get Single Parameter
```bash
curl -X POST http://localhost:6500/api/v1/parameters \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device001",
    "parameter_path": "Device.DeviceInfo.Manufacturer",
    "operation": "get"
  }'
```

#### Get Multiple Parameters
```bash
curl -X POST http://localhost:6500/api/v1/parameters/batch \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device001",
    "parameter_paths": [
      "Device.DeviceInfo.Manufacturer",
      "Device.DeviceInfo.ModelName",
      "Device.WiFi.SSID.1.SSID"
    ],
    "operation": "get"
  }'
```

### Setting Parameters

#### Single Parameter
```bash
curl -X POST http://localhost:6500/api/v1/parameters \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device001",
    "parameter_path": "Device.WiFi.SSID.1.SSID",
    "parameter_value": "MyNewWiFiName",
    "operation": "set"
  }'
```

#### Multiple Parameters
```bash
curl -X POST http://localhost:6500/api/v1/parameters/batch \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device001",
    "parameters": [
      {
        "parameter_path": "Device.WiFi.SSID.1.SSID",
        "parameter_value": "MyWiFi"
      },
      {
        "parameter_path": "Device.WiFi.SSID.1.KeyPassphrase",
        "parameter_value": "SecurePassword123"
      }
    ],
    "operation": "set"
  }'
```

## Real-Time Communication

### WebSocket Connection

Connect to the MTP service for real-time updates:

```javascript
// JavaScript example
const ws = new WebSocket('ws://localhost:8081/ws');

ws.onopen = function() {
    console.log('Connected to OpenUSP');
    
    // Send USP message
    ws.send(JSON.stringify({
        type: 'usp_request',
        device_id: 'device001',
        message: 'get_parameter',
        path: 'Device.DeviceInfo.Manufacturer'
    }));
};

ws.onmessage = function(event) {
    const response = JSON.parse(event.data);
    console.log('Response:', response);
};
```

### MTP Demo Interface

Use the built-in demo at http://localhost:8081/usp to:
- Test WebSocket connections
- Send USP messages interactively
- View real-time responses
- Debug communication issues

## Monitoring and Alerts

### Setting Up Alerts

```bash
# Create an alert rule
curl -X POST http://localhost:6500/api/v1/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device001",
    "alert_type": "parameter_change",
    "parameter_path": "Device.DeviceInfo.UpTime",
    "condition": "threshold",
    "threshold_value": "86400",
    "alert_level": "warning"
  }'
```

### Viewing Alerts

```bash
# Get all alerts
curl http://localhost:6500/api/v1/alerts

# Get alerts for specific device
curl http://localhost:6500/api/v1/alerts?device_id=device001
```

### Monitoring Dashboard

Access Grafana at http://localhost:3000 (admin/admin) for:
- Device connectivity graphs
- Parameter value trends
- System performance metrics
- Custom dashboards

## Session Management

### Creating Sessions

```bash
curl -X POST http://localhost:6500/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device001",
    "session_type": "configuration",
    "timeout": 300
  }'
```

### Managing Sessions

```bash
# List active sessions
curl http://localhost:6500/api/v1/sessions

# Get session details
curl http://localhost:6500/api/v1/sessions/session123

# Close session
curl -X DELETE http://localhost:6500/api/v1/sessions/session123
```

## Bulk Operations

### Bulk Parameter Updates

For managing multiple devices:

```bash
curl -X POST http://localhost:6500/api/v1/bulk/parameters \
  -H "Content-Type: application/json" \
  -d '{
    "device_ids": ["device001", "device002", "device003"],
    "parameters": [
      {
        "parameter_path": "Device.WiFi.SSID.1.SSID",
        "parameter_value": "CompanyWiFi"
      }
    ],
    "operation": "set"
  }'
```

### Device Discovery

```bash
# Start device discovery
curl -X POST http://localhost:6500/api/v1/discovery/start

# Check discovery status
curl http://localhost:6500/api/v1/discovery/status

# Get discovered devices
curl http://localhost:6500/api/v1/discovery/devices
```

## Troubleshooting

### Common Issues

#### Device Not Connecting
1. Check network connectivity
2. Verify device USP agent configuration
3. Check MTP transport settings
4. Review logs: `tail -f logs/mtp-service.log`

#### Parameter Operations Failing
1. Verify parameter paths with TR-181 data model
2. Check device permissions
3. Ensure parameter is writable
4. Test with single parameter first

#### WebSocket Connection Issues
1. Check firewall settings
2. Verify port 8081 is accessible
3. Test with MTP Demo UI first
4. Check browser console for errors

### Getting Help

1. **Check Logs**: Service-specific logs in `logs/` directory
2. **Status Check**: `make status` to verify all services
3. **API Documentation**: Interactive Swagger UI
4. **Examples**: Working protocol agents in `cmd/usp-agent/` and `cmd/cwmp-agent/`
5. **Community**: GitHub issues and discussions

## Best Practices

### Security
- Use HTTPS in production
- Implement proper authentication
- Regularly update credentials
- Monitor access logs

### Performance
- Batch parameter operations when possible
- Use appropriate session timeouts
- Monitor system resources
- Implement caching for frequently accessed data

### Device Management
- Use descriptive device IDs
- Implement device groups for bulk operations
- Set up proper alerting rules
- Regular health checks

## Next Steps

- **Advanced Configuration**: [Configuration Guide](CONFIGURATION.md)
- **API Reference**: Detailed API documentation
- **Examples**: More complex use cases
- **Development**: Extend functionality with custom code

Happy device managing! 🚀