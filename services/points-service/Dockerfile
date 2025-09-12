# Points Service Dockerfile
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install required packages
RUN apk add --no-cache git ca-certificates

# Copy points-service specific go mod files
COPY services/points-service/go.mod services/points-service/go.sum ./

# Copy pkg directory first (needed for local replace)
COPY pkg/ ./pkg/

# Download dependencies
RUN go mod download

# Copy points-service source code
COPY services/points-service/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o points-service main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy executable from builder stage
COPY --from=builder /app/points-service .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./points-service"]