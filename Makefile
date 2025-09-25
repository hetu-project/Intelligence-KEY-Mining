# Intelligence KEY Mining Project Makefile
# Build and install all microservices

# Go configuration
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Project configuration
PROJECT_NAME=Intelligence-KEY-Mining
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date +%FT%T%z)
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Service directories
SERVICES_DIR=services
MINER_GATEWAY_DIR=$(SERVICES_DIR)/miner-gateway
SBT_SERVICE_DIR=$(SERVICES_DIR)/sbt-service
VALIDATOR_DIR=$(SERVICES_DIR)/validator
POINTS_SERVICE_DIR=$(SERVICES_DIR)/points-service

# Binary names
MINER_GATEWAY_BINARY=miner-gateway
SBT_SERVICE_BINARY=sbt-service
VALIDATOR_BINARY=validator
POINTS_SERVICE_BINARY=points-service

# Install directory (GOPATH/bin)
INSTALL_DIR=$(shell go env GOPATH)/bin

# Build targets
.PHONY: all build clean test deps install install-deps

# Default target
all: build

# Build all services
build: build-miner-gateway build-sbt-service build-validator build-points-service
	@echo "✅ All services built successfully"

# Build individual services
build-miner-gateway:
	@echo "🔨 Building Miner Gateway..."
	cd $(MINER_GATEWAY_DIR) && $(GOBUILD) $(LDFLAGS) -o ../../build/$(MINER_GATEWAY_BINARY) .

build-sbt-service:
	@echo "🔨 Building SBT Service..."
	cd $(SBT_SERVICE_DIR) && $(GOBUILD) $(LDFLAGS) -o ../../build/$(SBT_SERVICE_BINARY) .

build-validator:
	@echo "🔨 Building Validator..."
	cd $(VALIDATOR_DIR) && $(GOBUILD) $(LDFLAGS) -o ../../build/$(VALIDATOR_BINARY) .

build-points-service:
	@echo "🔨 Building Points Service..."
	cd $(POINTS_SERVICE_DIR) && $(GOBUILD) $(LDFLAGS) -o ../../build/$(POINTS_SERVICE_BINARY) .

# Install all services to GOPATH/bin
install: build
	@echo "📦 Installing services to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	cp build/$(MINER_GATEWAY_BINARY) $(INSTALL_DIR)/
	cp build/$(SBT_SERVICE_BINARY) $(INSTALL_DIR)/
	cp build/$(VALIDATOR_BINARY) $(INSTALL_DIR)/
	cp build/$(POINTS_SERVICE_BINARY) $(INSTALL_DIR)/
	@echo "✅ All services installed to $(INSTALL_DIR)"
	@echo ""
	@echo "🚀 You can now run:"
	@echo "   $(MINER_GATEWAY_BINARY)    # Start Miner Gateway"
	@echo "   $(SBT_SERVICE_BINARY)      # Start SBT Service"
	@echo "   $(VALIDATOR_BINARY)        # Start Validator"
	@echo "   $(POINTS_SERVICE_BINARY)   # Start Points Service"

# Install dependencies
install-deps:
	@echo "📦 Installing Go dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	cd $(POINTS_SERVICE_DIR) && $(GOMOD) download && $(GOMOD) tidy
	@echo "✅ Dependencies installed"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf build/
	rm -f $(INSTALL_DIR)/$(MINER_GATEWAY_BINARY)
	rm -f $(INSTALL_DIR)/$(SBT_SERVICE_BINARY)
	rm -f $(INSTALL_DIR)/$(VALIDATOR_BINARY)
	rm -f $(INSTALL_DIR)/$(POINTS_SERVICE_BINARY)
	@echo "✅ Clean completed"

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...
	cd $(POINTS_SERVICE_DIR) && $(GOTEST) -v ./...
	@echo "✅ Tests completed"

# Development targets
dev-miner-gateway:
	@echo "🚀 Starting Miner Gateway in development mode..."
	cd $(MINER_GATEWAY_DIR) && $(GOCMD) run .

dev-sbt-service:
	@echo "🚀 Starting SBT Service in development mode..."
	cd $(SBT_SERVICE_DIR) && $(GOCMD) run .

dev-validator:
	@echo "🚀 Starting Validator in development mode..."
	cd $(VALIDATOR_DIR) && $(GOCMD) run .

dev-points-service:
	@echo "🚀 Starting Points Service in development mode..."
	cd $(POINTS_SERVICE_DIR) && $(GOCMD) run .

# Docker targets
docker-build:
	@echo "🐳 Building Docker images..."
	docker-compose build

docker-up:
	@echo "🐳 Starting services with Docker Compose..."
	docker-compose up -d

docker-down:
	@echo "🐳 Stopping Docker services..."
	docker-compose down

docker-logs:
	@echo "🐳 Showing Docker logs..."
	docker-compose logs -f

# Database migration
migrate:
	@echo "🗄️  Running database migrations..."
	./scripts/database/migrate.sh

# Show help
help:
	@echo "Intelligence KEY Mining Project - Available Commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build              Build all services"
	@echo "  make install           Build and install all services to GOPATH/bin"
	@echo "  make install-deps      Install Go dependencies"
	@echo "  make clean             Clean build artifacts"
	@echo ""
	@echo "Development Commands:"
	@echo "  make dev-miner-gateway  Run Miner Gateway in dev mode"
	@echo "  make dev-sbt-service    Run SBT Service in dev mode"
	@echo "  make dev-validator      Run Validator in dev mode"
	@echo "  make dev-points-service Run Points Service in dev mode"
	@echo ""
	@echo "Docker Commands:"
	@echo "  make docker-build       Build Docker images"
	@echo "  make docker-up          Start services with Docker"
	@echo "  make docker-down        Stop Docker services"
	@echo "  make docker-logs        Show Docker logs"
	@echo ""
	@echo "Other Commands:"
	@echo "  make test              Run tests"
	@echo "  make migrate           Run database migrations"
	@echo "  make help              Show this help message"

# Create build directory
build-dir:
	@mkdir -p build

# Ensure build directory exists before building
build-miner-gateway: build-dir
build-sbt-service: build-dir
build-validator: build-dir
build-points-service: build-dir
