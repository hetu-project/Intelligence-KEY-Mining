#!/bin/bash

# Points-system startup script
# Starts the points service and its related dependencies

set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check Docker and Docker Compose
check_dependencies() {
    log_info "Checking system dependencies..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed, please install Docker first"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed, please install Docker Compose first"
        exit 1
    fi
    
    log_success "System-dependency check passed"
}

# Check environment variables
check_env() {
    log_info "Checking environment variables..."
    
    # Check for .env file
    if [ ! -f ".env" ]; then
        log_warning ".env file does not exist, using default configuration"
        if [ -f "env.example" ]; then
            log_info "Copying configuration from env.example..."
            cp env.example .env
        fi
    fi
    
    log_success "Environment-variable check completed"
}

# Build and start services
start_services() {
    log_info "Starting points-system related services..."
    
    # Start infrastructure (MySQL)
    log_info "Starting database service..."
    docker-compose up -d mysql
    
    # Wait for database to be ready
    log_info "Waiting for database service to be ready..."
    sleep 10
    
    # Start points service
    log_info "Starting points service..."
    docker-compose up -d points-service
    
    # Wait for points service to be ready
    log_info "Waiting for points service to be ready..."
    sleep 5
    
    # Start SBT service (depends on points service)
    log_info "Starting SBT service..."
    docker-compose up -d sbt-service
    
    # Start Miner Gateway (integrates points distribution)
    log_info "Starting Miner Gateway service..."
    docker-compose up -d miner-gateway
    
    log_success "Points-system services started successfully"
}

# Health check
health_check() {
    log_info "Performing service health checks..."
    
    # Points service
    if curl -f http://localhost:8087/health &> /dev/null; then
        log_success "Points-service health check passed"
    else
        log_error "Points-service health check failed"
        return 1
    fi
    
    # SBT service
    if curl -f http://localhost:8086/health &> /dev/null; then
        log_success "SBT-service health check passed"
    else
        log_error "SBT-service health check failed"
        return 1
    fi
    
    # Miner Gateway
    if curl -f http://localhost:8081/health &> /dev/null; then
        log_success "Miner Gateway health check passed"
    else
        log_error "Miner Gateway health check failed"
        return 1
    fi
    
    log_success "All service health checks passed"
}

# Display service status
show_status() {
    log_info "Service status information:"
    echo
    echo "🎯 Points Service:"
    echo "   - Port: 8087"
    echo "   - Health: http://localhost:8087/health"
    echo "   - API Docs: http://localhost:8087/api/v1/points/config"
    echo
    echo "🏆 SBT Service:"
    echo "   - Port: 8086" 
    echo "   - Health: http://localhost:8086/health"
    echo
    echo "⛏️  Miner Gateway:"
    echo "   - Port: 8081"
    echo "   - Health: http://localhost:8081/health"
    echo
    echo "📊 Database Admin:"
    echo "   - phpMyAdmin: http://localhost:8089 (start with --profile debug)"
    echo
    echo "🔧 Management Commands:"
    echo "   - View logs: docker-compose logs -f points-service"
    echo "   - Stop services: docker-compose down"
    echo "   - Restart service: docker-compose restart points-service"
}

# Test points distribution
test_points_distribution() {
    log_info "Testing points-distribution function..."
    
    # Wait for services to be fully ready
    sleep 5
    
    # Call test endpoint
    if curl -X POST http://localhost:8087/api/v1/points/test \
        -H "Content-Type: application/json" &> /dev/null; then
        log_success "Points-distribution test succeeded"
    else
        log_warning "Points-distribution test failed, please check service status"
    fi
}

# Main function
main() {
    log_info "=== Points-System Startup Script ==="
    
    # Check dependencies
    check_dependencies
    
    # Check environment variables
    check_env
    
    # Start services
    start_services
    
    # Health check
    if health_check; then
        # Show status
        show_status
        
        # Test functionality
        if [ "${1:-}" = "--test" ]; then
            test_points_distribution
        fi
        
        log_success "Points system started successfully!"
    else
        log_error "Service startup failed, please check logs"
        docker-compose logs --tail=50 points-service sbt-service miner-gateway
        exit 1
    fi
}

# Execute main function
main "$@"