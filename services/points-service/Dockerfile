# Points Service Dockerfile
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install required packages
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o points-service ./services/points-service/main.go

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