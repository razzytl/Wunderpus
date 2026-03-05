# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o wonderpus -ldflags="-s -w" ./cmd/wonderpus

# Production stage
FROM alpine:3.19.1

LABEL org.opencontainers.image.source="https://github.com/wonderpus/wonderpus"
LABEL org.opencontainers.image.description="Universal Autonomous AI Agent in Go"

# Create non-root user
RUN addgroup -S wonderpus && adduser -S wonderpus -G wonderpus

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create data directory and set permissions
RUN mkdir -p /app/data && chown -R wonderpus:wonderpus /app

# Copy binary from builder
COPY --from=builder /app/wonderpus .
COPY --from=builder --chown=wonderpus:wonderpus /app/config.example.yaml ./config.yaml

# Switch to non-root user
USER wonderpus

# Expose ports
# Health & Metrics: 8080
# WebSocket: 9090 (default)
EXPOSE 8080 9090

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/live || exit 1

ENTRYPOINT ["./wonderpus"]
