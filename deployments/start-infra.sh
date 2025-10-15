#!/bin/bash
# OpenUSP Unified Infrastructure Startup
# Works on macOS, Linux, and Windows with Docker Desktop

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.infra.yml"

echo "🚀 Starting OpenUSP Infrastructure Services..."
echo "   📊 Prometheus, Grafana, 🦟 Mosquitto, 🐰 RabbitMQ, 🐘 PostgreSQL, 📨 Kafka"
echo "   🌍 Cross-platform configuration (macOS/Linux/Windows)"

# Start services
docker-compose -f "$COMPOSE_FILE" up -d

echo "⏳ Waiting for services to be ready..."
sleep 15

echo "✅ Infrastructure started successfully"
echo ""
echo "🌐 Access URLs:"
echo "  📊 Grafana:    http://localhost:3000 (admin/openusp123)"
echo "  📈 Prometheus: http://localhost:9090"
echo "  🐘 PostgreSQL: localhost:5433 (openusp/openusp123)"
echo "  🐰 RabbitMQ:   http://localhost:15672 (openusp/openusp123)"
echo "  🦟 Mosquitto:  localhost:1883"
echo "  📨 Kafka:      localhost:9092"
echo "  🎛️  Kafka UI:   http://localhost:8082"
echo ""
echo "💡 To stop: docker-compose -f $COMPOSE_FILE down"
echo "💡 To build and start services: make build-all && make start-all"