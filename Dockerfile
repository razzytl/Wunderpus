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
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/wonderpus .
COPY --from=builder /app/config.example.yaml ./config.yaml

# Expose ports
# Health & Metrics: 8080
# WebSocket: 9090 (default)
EXPOSE 8080 9090

ENTRYPOINT ["./wonderpus"]
