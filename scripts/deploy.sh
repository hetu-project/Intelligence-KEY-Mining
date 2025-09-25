#!/bin/bash

# Improved Deployment Script
# This script handles common deployment issues automatically

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

# Check and install dependencies
check_dependencies() {
    log_info "Checking required dependencies..."
    
    # Check if mysql client is installed
    if ! command -v mysql &> /dev/null; then
        log_warning "MySQL client not found, attempting to install..."
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y mysql-client
        elif command -v yum &> /dev/null; then
            sudo yum install -y mysql
        else
            log_error "Cannot install mysql-client automatically. Please install it manually:"
            log_error "  Ubuntu/Debian: apt-get install -y mysql-client"
            log_error "  CentOS/RHEL: yum install -y mysql"
            return 1
        fi
    fi
    
    # Check if docker-compose is installed
    if ! command -v docker-compose &> /dev/null; then
        log_error "docker-compose not found. Please install docker-compose first."
        return 1
    fi
    
    log_success "All dependencies are available"
    return 0
}

# Check environment file
check_env_file() {
    log_info "Checking environment configuration..."
    
    if [ ! -f ".env" ]; then
        log_warning ".env file does not exist"
        if [ -f "env.example" ]; then
            log_info "Creating .env from env.example..."
            cp env.example .env
            log_warning "Please edit .env file with your actual configuration"
            log_info "Opening .env file for editing..."
            ${EDITOR:-nano} .env
        else
            log_error "Neither .env nor env.example found"
            return 1
        fi
    fi
    
    log_success "Environment file ready"
    return 0
}

# Start infrastructure services
start_infrastructure() {
    log_info "Starting infrastructure services..."
    
    docker-compose up -d mysql redis dgraph-zero dgraph-alpha
    
    # Wait for MySQL to be ready
    log_info "Waiting for MySQL to be ready..."
    until docker-compose exec mysql mysqladmin ping -h localhost --silent; do
        sleep 2
    done
    
    log_success "Infrastructure services started"
}

# Run database migrations with improved error handling
run_migrations() {
    log_info "Running database migrations..."
    
    # Set environment variables for migration
    export DB_HOST="127.0.0.1"
    export DB_PORT="3306"
    export DB_DATABASE="${MYSQL_DATABASE:-pocw_db}"
    
    # Try with regular user first
    export DB_USER="${MYSQL_USER:-pocw_user}"
    export DB_PASSWORD="${MYSQL_PASSWORD:-pocw_password}"
    
    if ! ./scripts/database/migrate.sh; then
        log_warning "Migration failed with regular user, trying with root..."
        export DB_USER="root"
        export DB_PASSWORD="${MYSQL_ROOT_PASSWORD:-pocw_password}"
        
        if ! ./scripts/database/migrate.sh; then
            log_error "Migration failed even with root user"
            return 1
        fi
    fi
    
    log_success "Database migrations completed"
}

# Build and start application services
start_services() {
    log_info "Building and starting application services..."
    
    # Build services with no cache to ensure latest changes
    docker-compose build --no-cache points-service sbt-service miner-gateway
    
    # Start services
    docker-compose up -d points-service sbt-service miner-gateway
    
    # Wait for services to be healthy
    log_info "Waiting for services to be healthy..."
    sleep 30
    
    # Check service health
    for service in points-service sbt-service miner-gateway; do
        if docker-compose ps $service | grep -q "Up.*healthy\|Up.*starting"; then
            log_success "$service is running"
        else
            log_warning "$service may have issues, checking logs..."
            docker-compose logs --tail=20 $service
        fi
    done
}

# Test services
test_services() {
    log_info "Testing service endpoints..."
    
    sleep 10  # Give services time to fully start
    
    # Test points-service
    if curl -s http://localhost:8087/health > /dev/null; then
        log_success "Points service is healthy"
    else
        log_warning "Points service health check failed"
    fi
    
    # Test sbt-service
    if curl -s http://localhost:8086/health > /dev/null; then
        log_success "SBT service is healthy"
    else
        log_warning "SBT service health check failed"
    fi
    
    # Test miner-gateway
    if curl -s http://localhost:8081/health > /dev/null; then
        log_success "Miner gateway is healthy"
    else
        log_warning "Miner gateway health check failed"
    fi
}

# Main deployment function
deploy() {
    log_info "Starting deployment process..."
    
    # Check dependencies
    if ! check_dependencies; then
        log_error "Dependency check failed"
        exit 1
    fi
    
    # Check environment
    if ! check_env_file; then
        log_error "Environment setup failed"
        exit 1
    fi
    
    # Start infrastructure
    start_infrastructure
    
    # Run migrations
    run_migrations
    
    # Start services
    start_services
    
    # Test services
    test_services
    
    log_success "Deployment completed successfully!"
    log_info "Services are available at:"
    log_info "  Points Service: http://localhost:8087"
    log_info "  SBT Service: http://localhost:8086"
    log_info "  Miner Gateway: http://localhost:8081"
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy")
        deploy
        ;;
    "infrastructure")
        start_infrastructure
        ;;
    "migrate")
        run_migrations
        ;;
    "services")
        start_services
        ;;
    "test")
        test_services
        ;;
    *)
        echo "Usage: $0 [deploy|infrastructure|migrate|services|test]"
        echo "  deploy: Full deployment (default)"
        echo "  infrastructure: Start only infrastructure services"
        echo "  migrate: Run database migrations only"
        echo "  services: Start application services only"
        echo "  test: Test service endpoints only"
        exit 1
        ;;
esac
