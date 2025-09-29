# Multi-stage Dockerfile for OpenUSP Services
# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates make

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all services with version info
ARG VERSION=dev
ARG BUILD_TIME
ARG BUILD_USER
ARG GIT_COMMIT
ARG GIT_BRANCH
ARG GIT_TAG

# Set linker flags for version injection
ARG LDFLAGS='-X "openusp/pkg/version.Version='${VERSION}'" \
  -X "openusp/pkg/version.GitCommit='${GIT_COMMIT}'" \
  -X "openusp/pkg/version.GitBranch='${GIT_BRANCH}'" \
  -X "openusp/pkg/version.GitTag='${GIT_TAG}'" \
  -X "openusp/pkg/version.BuildDate='${BUILD_TIME}'" \
  -X "openusp/pkg/version.BuildUser='${BUILD_USER}'" \
  -w -s'

# Build all services
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o build/api-gateway cmd/api-gateway/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o build/data-service cmd/data-service/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o build/mtp-service cmd/mtp-service/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o build/usp-service cmd/usp-service/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "${LDFLAGS}" -o build/cwmp-service cmd/cwmp-service/main.go

# Runtime stage
FROM alpine:latest AS runtime

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl tzdata && \
    update-ca-certificates

# Create app directory
WORKDIR /app

# Create non-root user
RUN addgroup -g 1001 openusp && \
    adduser -D -s /bin/sh -u 1001 -G openusp openusp

# Copy binaries from builder
COPY --from=builder /app/build/ ./bin/

# Copy data model files
COPY --from=builder /app/pkg/datamodel/ ./pkg/datamodel/

# Create necessary directories
RUN mkdir -p /app/logs /app/data && \
    chown -R openusp:openusp /app

# Switch to non-root user
USER openusp

# Health check script
COPY --from=builder /app/scripts/healthcheck.sh ./scripts/
RUN chmod +x ./scripts/healthcheck.sh

# Default command (can be overridden)
CMD ["./bin/api-gateway"]

# Expose common ports
EXPOSE 8080 8081 7547 9092