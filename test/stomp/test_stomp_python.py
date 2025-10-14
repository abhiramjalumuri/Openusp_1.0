import socket
import struct

# Create a proper USP 1.3 StompConnect record (protobuf)
# This is manually encoded based on protobuf structure:
# message Record {
#   string version = 1;    // "1.3"
#   string to_id = 2;      // "proto::openusp.controller"
#   string from_id = 3;    // "os::012345-AAADF3FFCA7F"
#   oneof record_type {
#     STOMPConnectRecord stomp_connect = 10;
#   }
# }
# message STOMPConnectRecord {
#   enum STOMPVersion { V1_2 = 0; }
#   STOMPVersion version = 1;
# }

# Field 1: version = "1.3"
field1 = b'\x0a\x03' + b'1.3'  # tag 1 (0x0a = field 1, wire type 2=length-delimited), length 3, value "1.3"

# Field 2: to_id = "proto::openusp.controller"
to_id = b"proto::openusp.controller"
field2 = b'\x12' + bytes([len(to_id)]) + to_id  # tag 2 (0x12 = field 2, wire type 2)

# Field 3: from_id = "os::012345-AAADF3FFCA7F"  
from_id = b"os::012345-AAADF3FFCA7F"
field3 = b'\x1a' + bytes([len(from_id)]) + from_id  # tag 3 (0x1a = field 3, wire type 2)

# Field 11: stomp_connect with version = V1_2 (0)
stomp_connect_msg = b'\x08\x00'  # tag 1 (0x08 = field 1, wire type 0=varint), value 0 (V1_2)
field11 = b'\x5a' + bytes([len(stomp_connect_msg)]) + stomp_connect_msg  # tag 11 (0x5a = field 11, wire type 2)

# Combine all fields
usp_record = field1 + field2 + field3 + field11

print(f"📦 Created USP 1.3 StompConnect record ({len(usp_record)} bytes)")
print(f"   From: {from_id.decode()}")
print(f"   To: {to_id.decode()}")
print(f"   Version: 1.3")

# Connect to RabbitMQ STOMP
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.connect(('localhost', 61613))

# Send STOMP CONNECT frame
connect_frame = "CONNECT\naccept-version:1.2\nhost:/\nlogin:openusp\npasscode:openusp123\n\n\x00"
sock.send(connect_frame.encode())

# Read CONNECTED response
response = sock.recv(1024)
if b'CONNECTED' not in response:
    print(f"❌ STOMP connection failed: {response.decode('utf-8', errors='ignore')}")
    sock.close()
    exit(1)
    
print(f"✅ STOMP CONNECTED successfully")

# Send MESSAGE frame with USP record
message_frame = (
    "SEND\n"
    "destination:/queue/usp.controller\n"
    f"content-length:{len(usp_record)}\n"
    "content-type:application/vnd.bbf.usp.msg\n"
    f"usp-endpoint-id:{from_id.decode()}\n"
    "\n"
)
sock.send(message_frame.encode() + usp_record + b'\x00')

print(f"✅ Sent USP StompConnect record ({len(usp_record)} bytes) to /queue/usp.controller")

# Subscribe to agent queue to receive response
subscribe_frame = (
    "SUBSCRIBE\n"
    "id:sub-agent\n"
    "destination:/queue/usp.agent\n"
    "ack:auto\n"
    "\n\x00"
)
sock.send(subscribe_frame.encode())

print("📡 Subscribed to /queue/usp.agent for responses...")

# Wait for response
import time
time.sleep(2)

# Check for messages
sock.settimeout(3.0)
try:
    response = sock.recv(4096)
    print(f"📬 Received response ({len(response)} bytes)")
    print(f"Response preview: {response[:100]}")
except socket.timeout:
    print("⏱️  No response received within timeout")

# Disconnect
disconnect_frame = "DISCONNECT\n\n\x00"
sock.send(disconnect_frame.encode())
sock.close()
