# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies (allow toolchain download for newer Go versions)
RUN GOTOOLCHAIN=auto go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux GOTOOLCHAIN=auto go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o whatsapp-llm-bot ./cmd/bot

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create app user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from build stage
COPY --from=builder /build/whatsapp-llm-bot .

# Copy web assets
COPY --from=builder /build/web ./web

# Copy default config (can be overridden by volume mount)
COPY --from=builder /build/config.yaml ./config.yaml.example

# Create directories for data
RUN mkdir -p /data /config && \
    chown -R appuser:appuser /app /data /config

# Switch to non-root user
USER appuser

# Expose HTTP port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Set environment variables
ENV CONFIG_PATH=/config/config.yaml \
    WHATSAPP_SESSION_PATH=/data/whatsapp_session

# Run the application
CMD ["./whatsapp-llm-bot"]
