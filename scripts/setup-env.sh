#!/bin/bash

# Environment Setup Script
# Helps manage .env files without Git conflicts

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

# Check if .env exists
check_env_exists() {
    if [ -f ".env" ]; then
        return 0
    else
        return 1
    fi
}

# Create .env from template
create_env_from_template() {
    local template_file="$1"
    
    if [ ! -f "$template_file" ]; then
        log_error "Template file $template_file not found"
        return 1
    fi
    
    log_info "Creating .env from $template_file..."
    cp "$template_file" .env
    log_success ".env file created"
    
    echo
    log_warning "⚠️  IMPORTANT: Please edit .env file with your actual configuration:"
    echo "   - Replace all REPLACE_WITH_* placeholders"
    echo "   - Add your actual API keys and private keys"
    echo "   - Update URLs and endpoints"
    echo
    echo "🔐 Security reminder:"
    echo "   - Never commit .env with real secrets to git"
    echo "   - .env is already in .gitignore"
    echo
}

# Backup existing .env
backup_env() {
    if check_env_exists; then
        local backup_name=".env.backup.$(date +%Y%m%d_%H%M%S)"
        log_info "Backing up existing .env to $backup_name"
        cp .env "$backup_name"
        log_success "Backup created: $backup_name"
    fi
}

# Validate .env file
validate_env() {
    if ! check_env_exists; then
        log_error ".env file does not exist"
        return 1
    fi
    
    log_info "Validating .env file..."
    
    # Check for placeholder values that need to be replaced
    local placeholders=(
        "your_secure_root_password"
        "your_secure_db_password"
        "your_pinata_api_key"
        "your_pinata_secret_key"
        "your_twitter_api_key"
        "your_project_id"
        "your_deployed_sbt_contract_address"
        "your_sbt_contract_private_key"
        "your-twitter-middleware.com"
    )
    
    local has_placeholders=false
    
    for placeholder in "${placeholders[@]}"; do
        if grep -q "$placeholder" .env; then
            log_warning "Found placeholder: $placeholder"
            has_placeholders=true
        fi
    done
    
    if [ "$has_placeholders" = true ]; then
        log_error "Validation failed: Please replace all placeholder values in .env"
        return 1
    else
        log_success "Validation passed: No placeholder values found"
        return 0
    fi
}

# Show .env status
show_env_status() {
    echo "📋 Environment Configuration Status:"
    echo "=================================="
    
    if check_env_exists; then
        log_success ".env file exists"
        
        # Show file size and modification time
        local file_info=$(ls -la .env | awk '{print $5, $6, $7, $8}')
        echo "   File info: $file_info"
        
        # Count non-empty, non-comment lines
        local config_lines=$(grep -v '^#' .env | grep -v '^$' | wc -l | tr -d ' ')
        echo "   Configuration lines: $config_lines"
        
        # Check for common required variables
        local required_vars=(
            "MYSQL_ROOT_PASSWORD"
            "PINATA_API_KEY"
            "TWITTER_API_KEY"
            "ETH_RPC_URL"
            "SBT_CONTRACT_ADDRESS"
            "MINER_PRIVATE_KEY"
        )
        
        echo "   Required variables status:"
        for var in "${required_vars[@]}"; do
            if grep -q "^$var=" .env && ! grep -q "^$var=your_" .env; then
                echo "     ✅ $var: configured"
            elif grep -q "^$var=your_" .env; then
                echo "     ⚠️  $var: needs configuration"
            else
                echo "     ❌ $var: missing"
            fi
        done
        
    else
        log_warning ".env file does not exist"
        
        # Check for available template
        echo "   Available template:"
        if [ -f "env.example" ]; then
            echo "     📄 env.example"
        else
            echo "     ❌ No template found"
        fi
    fi
    
    echo
}

# Interactive setup
interactive_setup() {
    echo "🔧 Interactive Environment Setup"
    echo "================================"
    
    show_env_status
    
    if check_env_exists; then
        echo
        read -p "❓ .env already exists. Do you want to recreate it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Keeping existing .env file"
            return 0
        fi
        backup_env
    fi
    
    # Use the only available template
    local template="env.example"
    if [ ! -f "env.example" ]; then
        log_error "env.example template not found"
        return 1
    fi
    
    create_env_from_template "$template"
    
    echo
    read -p "❓ Do you want to open .env for editing now? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Try to open with common editors
        if command -v code &> /dev/null; then
            code .env
        elif command -v nano &> /dev/null; then
            nano .env
        elif command -v vim &> /dev/null; then
            vim .env
        else
            log_info "Please edit .env with your preferred editor"
        fi
    fi
}

# Main function
main() {
    case "${1:-help}" in
        "create"|"init")
            if [ -n "$2" ]; then
                create_env_from_template "$2"
            else
                create_env_from_template "env.example"
            fi
            ;;
        "backup")
            backup_env
            ;;
        "validate"|"check")
            validate_env
            ;;
        "status")
            show_env_status
            ;;
        "interactive"|"setup")
            interactive_setup
            ;;
        "help"|*)
            echo "🔧 Environment Setup Script"
            echo "=========================="
            echo
            echo "Usage: $0 <command> [options]"
            echo
            echo "Commands:"
            echo "  create [template]    Create .env from template (default: env.example)"
            echo "  backup              Backup existing .env file"
            echo "  validate            Validate .env configuration"
            echo "  status              Show .env status"
            echo "  interactive         Interactive setup wizard"
            echo "  help                Show this help message"
            echo
            echo "Examples:"
            echo "  $0 create                    # Create from env.example"
            echo "  $0 create env.example        # Create from env.example"
            echo "  $0 interactive               # Run setup wizard"
            echo "  $0 validate                  # Check configuration"
            echo
            echo "💡 Tips:"
            echo "  - Use 'interactive' for first-time setup"
            echo "  - Use 'validate' before deploying"
            echo "  - .env is automatically ignored by git"
            ;;
    esac
}

# Execute main function with all arguments
main "$@"
