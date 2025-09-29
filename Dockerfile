# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all services
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/mtp-service cmd/mtp-service/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/cwmp-service cmd/cwmp-service/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/api-gateway cmd/api-gateway/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/usp-service cmd/usp-service/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates curl

# Create app directory
WORKDIR /root/

# Copy binaries from builder
COPY --from=builder /app/build/ .

# Copy data model files
COPY --from=builder /app/pkg/datamodel/ ./pkg/datamodel/

# Create non-root user
RUN adduser -D -s /bin/sh openusp

# Change ownership
RUN chown -R openusp:openusp /root/

# Switch to non-root user
USER openusp

# Expose ports
EXPOSE 8080 8081 7547

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8081/health || exit 1

# Default command (can be overridden)
CMD ["./mtp-service"]