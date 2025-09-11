#!/bin/bash

# Database migration script
# Executes database structural updates

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

# Default configuration
DEFAULT_HOST="localhost"
DEFAULT_PORT="3306"
DEFAULT_DATABASE="pocw_db"
DEFAULT_USER="pocw_user"
MIGRATIONS_DIR="$(dirname "$0")/migrations"

# Help message
show_help() {
    cat << EOF
Database Migration Tool

Usage:
    $0 [options] [command]

Commands:
    migrate     Run all pending migrations
    status      Show migration status
    rollback    Roll back the last migration (if supported)
    reset       Reset all migrations (dangerous)

Options:
    -h, --host HOST         Database host (default: $DEFAULT_HOST)
    -P, --port PORT         Database port (default: $DEFAULT_PORT)
    -d, --database DB       Database name (default: $DEFAULT_DATABASE)
    -u, --user USER         Database user (default: $DEFAULT_USER)
    -p, --password PASS     Database password
    -f, --file FILE         Execute a specific migration file
    --dry-run               Preview mode, no actual changes
    --help                  Show this help message

Environment variables:
    DB_HOST                Database host
    DB_PORT                Database port
    DB_DATABASE            Database name
    DB_USER                Database user
    DB_PASSWORD            Database password

Examples:
    $0 migrate                              # Run all migrations
    $0 status                               # Show status
    $0 -f 001_add_twitter_task_support.sql # Run specific file
    $0 --dry-run migrate                    # Preview migrations
EOF
}

# Parse command-line arguments
parse_args() {
    HOST="${DB_HOST:-$DEFAULT_HOST}"
    PORT="${DB_PORT:-$DEFAULT_PORT}"
    DATABASE="${DB_DATABASE:-$DEFAULT_DATABASE}"
    USER="${DB_USER:-$DEFAULT_USER}"
    PASSWORD="${DB_PASSWORD:-}"
    COMMAND=""
    MIGRATION_FILE=""
    DRY_RUN=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--host)
                HOST="$2"
                shift 2
                ;;
            -P|--port)
                PORT="$2"
                shift 2
                ;;
            -d|--database)
                DATABASE="$2"
                shift 2
                ;;
            -u|--user)
                USER="$2"
                shift 2
                ;;
            -p|--password)
                PASSWORD="$2"
                shift 2
                ;;
            -f|--file)
                MIGRATION_FILE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            migrate|status|rollback|reset)
                COMMAND="$1"
                shift
                ;;
            *)
                log_error "Unknown argument: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Default to migrate if no command specified
    if [[ -z "$COMMAND" && -z "$MIGRATION_FILE" ]]; then
        COMMAND="migrate"
    fi

    # If file specified, command is migrate
    if [[ -n "$MIGRATION_FILE" ]]; then
        COMMAND="migrate"
    fi
}

# Check required tools
check_requirements() {
    if ! command -v mysql &> /dev/null; then
        log_error "MySQL client not installed"
        exit 1
    fi
}

# Get database password
get_password() {
    if [[ -z "$PASSWORD" ]]; then
        read -s -p "Enter database password: " PASSWORD
        echo
    fi
}

# Build MySQL connection command
build_mysql_cmd() {
    MYSQL_CMD="mysql -h$HOST -P$PORT -u$USER -p$PASSWORD $DATABASE"
}

# Test database connection
test_connection() {
    log_info "Testing database connection..."
    
    if ! echo "SELECT 1;" | $MYSQL_CMD &> /dev/null; then
        log_error "Cannot connect to database"
        log_error "Host: $HOST:$PORT"
        log_error "Database: $DATABASE"
        log_error "User: $USER"
        exit 1
    fi
    
    log_success "Database connection successful"
}

# Initialize migrations table
init_migrations_table() {
    log_info "Initializing migrations table..."
    
    local sql="
CREATE TABLE IF NOT EXISTS migrations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    version VARCHAR(20) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Preview mode - will create migrations table"
        return
    fi
    
    echo "$sql" | $MYSQL_CMD
    log_success "Migrations table initialized"
}

# Get executed migrations
get_executed_migrations() {
    echo "SELECT version FROM migrations ORDER BY executed_at;" | $MYSQL_CMD -s -N 2>/dev/null || echo ""
}

# Get available migration files
get_available_migrations() {
    if [[ ! -d "$MIGRATIONS_DIR" ]]; then
        log_warning "Migrations directory not found: $MIGRATIONS_DIR"
        return
    fi
    
    find "$MIGRATIONS_DIR" -name "*.sql" -type f | sort
}

# Extract migration version
extract_version() {
    local file="$1"
    basename "$file" | sed 's/^\([0-9]\+\)_.*/\1/'
}

# Execute migration file
execute_migration() {
    local file="$1"
    local version=$(extract_version "$file")
    
    log_info "Executing migration: $(basename "$file")"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Preview mode - will execute migration file: $file"
        return
    fi
    
    # Execute migration file
    if $MYSQL_CMD < "$file"; then
        log_success "Migration executed successfully: $(basename "$file")"
        
        # Record in migrations table (if not already recorded)
        local description=$(head -5 "$file" | grep -E "^-- Description:" | sed 's/^-- Description: *//' || echo "Migration $version")
        local check_sql="SELECT COUNT(*) FROM migrations WHERE version = '$version';"
        local exists=$(echo "$check_sql" | $MYSQL_CMD -s -N)
        
        if [[ "$exists" == "0" ]]; then
            local record_sql="INSERT INTO migrations (version, description) VALUES ('$version', '$description');"
            echo "$record_sql" | $MYSQL_CMD
        fi
    else
        log_error "Migration failed: $(basename "$file")"
        exit 1
    fi
}

# Run all migrations
run_migrations() {
    log_info "Starting database migrations..."
    
    # Get executed migrations
    local executed_migrations=$(get_executed_migrations)
    
    # Get available migration files
    local available_migrations=$(get_available_migrations)
    
    if [[ -z "$available_migrations" ]]; then
        log_warning "No migration files found"
        return
    fi
    
    local migration_count=0
    
    for file in $available_migrations; do
        local version=$(extract_version "$file")
        
        # Check if already executed
        if echo "$executed_migrations" | grep -q "^$version$"; then
            log_info "Skipping executed migration: $(basename "$file")"
            continue
        fi
        
        execute_migration "$file"
        ((migration_count++))
    done
    
    if [[ $migration_count -eq 0 ]]; then
        log_success "All migrations are up to date"
    else
        log_success "Successfully executed $migration_count migrations"
    fi
}

# Run specific migration file
run_specific_migration() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        # Try to find in migrations directory
        local full_path="$MIGRATIONS_DIR/$file"
        if [[ -f "$full_path" ]]; then
            file="$full_path"
        else
            log_error "Migration file not found: $file"
            exit 1
        fi
    fi
    
    log_info "Executing specific migration file: $(basename "$file")"
    execute_migration "$file"
}

# Show migration status
show_status() {
    log_info "Showing migration status..."
    
    # Get executed migrations
    local executed_migrations=$(get_executed_migrations)
    
    # Get available migration files
    local available_migrations=$(get_available_migrations)
    
    echo
    echo "=== Migration Status ==="
    echo
    
    if [[ -z "$available_migrations" ]]; then
        echo "No migration files found"
        return
    fi
    
    for file in $available_migrations; do
        local version=$(extract_version "$file")
        local filename=$(basename "$file")
        
        if echo "$executed_migrations" | grep -q "^$version$"; then
            echo -e "${GREEN}✓${NC} $filename (executed)"
        else
            echo -e "${YELLOW}○${NC} $filename (pending)"
        fi
    done
    
    echo
    echo "Executed migrations: $(echo "$executed_migrations" | wc -l)"
    echo "Available migrations: $(echo "$available_migrations" | wc -l)"
}

# Main function
main() {
    parse_args "$@"
    check_requirements
    get_password
    build_mysql_cmd
    test_connection
    init_migrations_table
    
    case "$COMMAND" in
        migrate)
            if [[ -n "$MIGRATION_FILE" ]]; then
                run_specific_migration "$MIGRATION_FILE"
            else
                run_migrations
            fi
            ;;
        status)
            show_status
            ;;
        rollback)
            log_error "Rollback not yet implemented"
            exit 1
            ;;
        reset)
            log_error "Reset not yet implemented (dangerous operation)"
            exit 1
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            show_help
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"