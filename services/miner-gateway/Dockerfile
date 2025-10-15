# Build stage
FROM golang:1.21-alpine AS builder

# Use mainland China Go module mirrors for faster downloads
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn

# Install required packages
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Copy pkg directory first (needed for local replace before go mod download)
COPY pkg/ ./pkg/

# Download dependencies with cache mount
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy only the miner-gateway service source code
COPY services/miner-gateway/ ./services/miner-gateway/

# Build the application with optimizations and cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o main ./services/miner-gateway

# Runtime stage
FROM alpine:latest

# Install ca-certificates and tzdata
RUN apk --no-cache add ca-certificates tzdata

# Set timezone
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

# Create non-root user
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /app

# Copy binary from build stage
COPY --from=builder /app/main .

# Set file permissions
RUN chown appuser:appuser main

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]