#!/bin/bash
# Create missing Kafka topics for OpenUSP

set -e

KAFKA_BROKER="${KAFKA_BROKER:-localhost:9092}"

echo "🔧 Creating Kafka topics for OpenUSP..."

# Function to create topic if it doesn't exist
create_topic() {
    local topic=$1
    local partitions=${2:-3}
    local replication=${3:-1}
    
    echo "Creating topic: $topic (partitions: $partitions, replication: $replication)"
    kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" \
        --create \
        --if-not-exists \
        --topic "$topic" \
        --partitions "$partitions" \
        --replication-factor "$replication" || true
}

# USP Service Topics
echo ""
echo "📋 Creating USP Service Topics..."
create_topic "usp.messages.inbound" 3 1
create_topic "usp.messages.outbound" 3 1
create_topic "usp.api.request" 3 1
create_topic "usp.api.response" 3 1
create_topic "usp.data.request" 3 1
create_topic "usp.data.response" 3 1

# CWMP Service Topics
echo ""
echo "📋 Creating CWMP Service Topics..."
create_topic "cwmp.messages.inbound" 3 1
create_topic "cwmp.messages.outbound" 3 1
create_topic "cwmp.api.request" 3 1
create_topic "cwmp.api.response" 3 1
create_topic "cwmp.data.request" 3 1
create_topic "cwmp.data.response" 3 1

# Data Service Topics
echo ""
echo "📋 Creating Data Service Topics..."
create_topic "data.device.created" 3 1
create_topic "data.device.updated" 3 1
create_topic "data.device.deleted" 3 1
create_topic "data.parameter.created" 3 1
create_topic "data.parameter.updated" 3 1
create_topic "data.parameter.deleted" 3 1

# Agent Events Topics
echo ""
echo "📋 Creating Agent Events Topics..."
create_topic "agent.connected" 3 1
create_topic "agent.disconnected" 3 1
create_topic "agent.onboarding" 3 1

# MTP Events Topics
echo ""
echo "📋 Creating MTP Events Topics..."
create_topic "mtp.stomp.events" 3 1
create_topic "mtp.mqtt.events" 3 1
create_topic "mtp.websocket.events" 3 1
create_topic "mtp.connection.established" 3 1
create_topic "mtp.connection.closed" 3 1
create_topic "mtp.http.events" 3 1

# API Gateway Topics
echo ""
echo "📋 Creating API Gateway Topics..."
create_topic "api.request" 3 1
create_topic "api.response" 3 1

echo ""
echo "✅ Kafka topics created successfully!"
echo ""
echo "📊 Listing all topics:"
kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" --list | sort
