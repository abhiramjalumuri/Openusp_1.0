# OpenUSP Local Kubernetes Setup

This directory contains scripts and configurations for setting up a local Kubernetes cluster and deploying OpenUSP services using Helm charts.

## Prerequisites

- Docker Desktop or Podman
- kubectl
- Helm 3.x
- Kind (Kubernetes in Docker) or Minikube

## Quick Start

1. **Setup local Kubernetes cluster**:
   ```bash
   ./setup-cluster.sh
   ```

2. **Deploy OpenUSP services**:
   ```bash
   ./deploy-openusp.sh
   ```

3. **Access services**:
   ```bash
   ./access-services.sh
   ```

## Components

- `setup-cluster.sh` - Creates local Kubernetes cluster
- `deploy-openusp.sh` - Deploys all OpenUSP services via Helm
- `helm/` - Helm charts for all OpenUSP components
- `config/` - Kubernetes configurations
- `monitoring/` - Monitoring stack setup

## Services Deployed

- API Gateway (port 8080)
- Data Service (port 8081) 
- USP Service (port 8082)
- MTP Service (port 8083)
- CWMP Service (port 7547)
- Connection Manager (port 8084)
- USP Agent
- CWMP Agent
- PostgreSQL Database
- RabbitMQ
- Mosquitto MQTT
- Consul
- Prometheus & Grafana (monitoring)