# Multi-stage build for minimal size

# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download -x

# Copy source
COPY . .

# Build static binary with symbol table and debug info stripped
RUN CGO_ENABLED=0 GOOS=linux go build -o wonderpus \
    -ldflags="-s -w -buildid=" \
    -trimpath \
    ./cmd/wonderpus

# Production stage - Ultra minimal
FROM scratch

LABEL org.opencontainers.image.source="https://github.com/wunderpus/wunderpus"
LABEL org.opencontainers.image.description="Universal Autonomous AI Agent in Go"

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy binary
COPY --from=builder /app/wonderpus /usr/local/bin/wunderpus

# Copy config
COPY --from=builder /app/config.example.yaml /app/config.yaml

# Set working directory
WORKDIR /app

# Expose ports
# Health & Metrics: 8080
# WebSocket: 9090
EXPOSE 8080 9090

# Non-root user (using numeric UID for scratch compatibility)
USER 1000:1000

ENTRYPOINT ["/usr/local/bin/wunderpus"]
