#!/bin/bash

# Core Services Startup Script
# Starts SBT, MinerGateway, Validator, and Points services

set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
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

log_service() {
    echo -e "${PURPLE}[SERVICE]${NC} $1"
}

# Check environment file
check_env_file() {
    log_info "Checking environment configuration..."
    
    if [ ! -f ".env" ]; then
        log_warning ".env file does not exist"
        if [ -f "env.example" ]; then
            log_info "Creating .env from env.example..."
            cp env.example .env
            log_warning "Please edit .env file with your actual configuration before running services"
            return 1
        else
            log_error "Neither .env nor env.example found"
            return 1
        fi
    fi
    
    log_success "Environment file check passed"
    return 0
}

# Check dependencies
check_dependencies() {
    log_info "Checking system dependencies..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        log_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    # Check Go (for development mode)
    if ! command -v go &> /dev/null; then
        log_warning "Go is not installed. Docker mode only."
    fi
    
    log_success "System dependencies check passed"
}

# Start infrastructure services
start_infrastructure() {
    log_info "Starting infrastructure services..."
    
    # Start MySQL
    log_service "Starting MySQL database..."
    docker-compose up -d mysql
    
    # Start Redis  
    log_service "Starting Redis cache..."
    docker-compose up -d redis
    
    # Start Dgraph
    log_service "Starting Dgraph database..."
    docker-compose up -d dgraph-zero dgraph-alpha
    
    # Wait for infrastructure to be ready
    log_info "Waiting for infrastructure services to be ready..."
    sleep 15
    
    # Check MySQL health
    local mysql_ready=false
    for i in {1..30}; do
        if docker-compose exec -T mysql mysqladmin ping -h localhost --silent; then
            mysql_ready=true
            break
        fi
        sleep 2
    done
    
    if [ "$mysql_ready" = true ]; then
        log_success "MySQL is ready"
    else
        log_error "MySQL failed to start properly"
        return 1
    fi
    
    log_success "Infrastructure services started successfully"
}

# Start core business services
start_core_services() {
    log_info "Starting core business services..."
    
    # Start Points Service first (dependency for others)
    log_service "Starting Points Service..."
    docker-compose up -d points-service
    sleep 5
    
    # Start SBT Service
    log_service "Starting SBT Service..."
    docker-compose up -d sbt-service
    sleep 3
    
    # Start Miner Gateway
    log_service "Starting Miner Gateway..."
    docker-compose up -d miner-gateway
    sleep 3
    
    # Start Validators
    log_service "Starting Validator Services..."
    docker-compose up -d validator-ui validator-format-1 validator-format-2 validator-semantic
    
    log_success "Core business services started successfully"
}

# Health check for services
health_check() {
    log_info "Performing service health checks..."
    
    local services=(
        "points-service:8087"
        "sbt-service:8086"
        "miner-gateway:8081"
        "validator-ui:8082"
        "validator-format-1:8083"
    )
    
    local all_healthy=true
    
    for service_info in "${services[@]}"; do
        IFS=':' read -r service_name port <<< "$service_info"
        
        if curl -f "http://localhost:$port/health" &> /dev/null; then
            log_success "$service_name health check passed"
        else
            log_error "$service_name health check failed"
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = true ]; then
        log_success "All service health checks passed"
        return 0
    else
        log_error "Some services failed health checks"
        return 1
    fi
}

# Show service status
show_service_status() {
    log_info "Service Status Information:"
    echo
    echo "🎯 Points Service:"
    echo "   - Port: 8087"
    echo "   - Health: http://localhost:8087/health"
    echo "   - API: http://localhost:8087/api/v1/points/"
    echo
    echo "🏆 SBT Service:"
    echo "   - Port: 8086"
    echo "   - Health: http://localhost:8086/health"
    echo "   - Register: POST http://localhost:8086/api/v1/sbt/register"
    echo
    echo "⛏️  Miner Gateway:"
    echo "   - Port: 8081"
    echo "   - Health: http://localhost:8081/health"
    echo "   - Submit Task: POST http://localhost:8081/api/v1/tasks/submit"
    echo "   - Create Task: POST http://localhost:8081/api/v1/tasks/create"
    echo
    echo "✅ Validators:"
    echo "   - UI Validator: http://localhost:8082/health"
    echo "   - Format Validator 1: http://localhost:8083/health"
    echo "   - Format Validator 2: http://localhost:8084/health"
    echo "   - Semantic Validator: http://localhost:8085/health"
    echo
    echo "🗄️  Infrastructure:"
    echo "   - MySQL: localhost:3306"
    echo "   - Redis: localhost:6379"
    echo "   - Dgraph: http://localhost:8080 (API), http://localhost:8000 (UI)"
    echo
    echo "🔧 Management Commands:"
    echo "   - View logs: docker-compose logs -f [service-name]"
    echo "   - Stop services: docker-compose down"
    echo "   - Restart service: docker-compose restart [service-name]"
    echo "   - Scale service: docker-compose up -d --scale [service-name]=2"
}

# Run database migrations
run_migrations() {
    log_info "Running database migrations..."
    
    if [ -f "scripts/database/migrate.sh" ]; then
        chmod +x scripts/database/migrate.sh
        ./scripts/database/migrate.sh
        log_success "Database migrations completed"
    else
        log_warning "No migration script found, skipping..."
    fi
}

# Development mode startup
start_dev_mode() {
    log_info "Starting services in development mode..."
    
    # Start infrastructure only
    start_infrastructure
    
    # Run migrations
    run_migrations
    
    log_info "Infrastructure ready. Start services manually:"
    echo
    echo "🚀 Development Commands:"
    echo "   make dev-points-service   # Start Points Service"
    echo "   make dev-sbt-service      # Start SBT Service"  
    echo "   make dev-miner-gateway    # Start Miner Gateway"
    echo "   make dev-validator        # Start Validator"
    echo
    echo "Or use Docker for specific services:"
    echo "   docker-compose up points-service"
    echo "   docker-compose up sbt-service"
    echo "   docker-compose up miner-gateway"
}

# Main function
main() {
    echo "🚀 Intelligence KEY Mining - Core Services Startup"
    echo "=================================================="
    
    # Parse arguments
    local dev_mode=false
    local skip_health=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dev|--development)
                dev_mode=true
                shift
                ;;
            --skip-health)
                skip_health=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo
                echo "Options:"
                echo "  --dev, --development    Start in development mode (infrastructure only)"
                echo "  --skip-health          Skip health checks"
                echo "  --help, -h             Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Check dependencies
    check_dependencies
    
    # Check environment
    if ! check_env_file; then
        log_error "Environment configuration required. Please edit .env file first."
        exit 1
    fi
    
    if [ "$dev_mode" = true ]; then
        start_dev_mode
        return 0
    fi
    
    # Start services
    start_infrastructure
    run_migrations
    start_core_services
    
    # Wait for services to be ready
    log_info "Waiting for services to be fully ready..."
    sleep 10
    
    # Health check
    if [ "$skip_health" = false ]; then
        if health_check; then
            show_service_status
            log_success "🎉 All core services started successfully!"
        else
            log_error "Some services failed to start properly. Check logs:"
            echo "docker-compose logs -f"
            exit 1
        fi
    else
        show_service_status
        log_success "🎉 Core services startup completed!"
    fi
}

# Execute main function with all arguments
main "$@"
