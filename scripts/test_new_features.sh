#!/bin/bash

# Script to test new features
# Verifies task-creation and batch-verification endpoints

set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8001}"
TEST_WALLET="0x1234567890abcdef1234567890abcdef12345678"

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

# Check whether the service is running
check_service() {
    log_info "Checking service health..."
    
    if curl -s "$BASE_URL/health" > /dev/null; then
        log_success "Service is running"
    else
        log_error "Service is not running or unreachable"
        log_error "Please ensure the service is running on $BASE_URL"
        exit 1
    fi
}

# Test task creation
test_task_creation() {
    log_info "Testing task-creation feature..."
    
    local response=$(curl -s -X POST "$BASE_URL/api/v1/task-creation/create" \
        -H "Content-Type: application/json" \
        -d '{
            "user_wallet": "'$TEST_WALLET'",
            "project_name": "Hetu Network Test",
            "project_icon": "https://example.com/icon.png",
            "description": "Test task creation functionality",
            "twitter_username": "@HetuNetwork",
            "twitter_link": "https://twitter.com/HetuNetwork",
            "tweet_id": "1234567890"
        }')
    
    if echo "$response" | grep -q '"success":true'; then
        local task_id=$(echo "$response" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
        log_success "Task created successfully, Task ID: $task_id"
        
        # Test retrieving task status
        log_info "Retrieving task status..."
        local status_response=$(curl -s "$BASE_URL/api/v1/task-creation/status/$task_id")
        
        if echo "$status_response" | grep -q '"success":true'; then
            log_success "Task status retrieved successfully"
        else
            log_error "Failed to retrieve task status"
            echo "$status_response"
        fi
        
        echo "$task_id"
    else
        log_error "Task creation failed"
        echo "$response"
        return 1
    fi
}

# Test batch verification
test_batch_verification() {
    log_info "Testing batch-verification feature..."
    
    local response=$(curl -s -X POST "$BASE_URL/api/v1/batch-verification/verify" \
        -H "Content-Type: application/json" \
        -d '{
            "user_wallet": "'$TEST_WALLET'",
            "start_time": "2024-01-01T00:00:00Z",
            "end_time": "2024-01-01T23:59:59Z",
            "tasks": [
                {
                    "tweet_id": "1234567890",
                    "twitter_id": "@testuser1"
                },
                {
                    "tweet_id": "1234567891",
                    "twitter_id": "@testuser2"
                }
            ]
        }')
    
    if echo "$response" | grep -q '"success":true'; then
        local task_id=$(echo "$response" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
        log_success "Batch-verification submitted successfully, Task ID: $task_id"
        
        # Test retrieving batch-verification status
        log_info "Retrieving batch-verification status..."
        local status_response=$(curl -s "$BASE_URL/api/v1/batch-verification/status/$task_id")
        
        if echo "$status_response" | grep -q '"success":true'; then
            log_success "Batch-verification status retrieved successfully"
        else
            log_error "Failed to retrieve batch-verification status"
            echo "$status_response"
        fi
        
        echo "$task_id"
    else
        log_error "Batch-verification submission failed"
        echo "$response"
        return 1
    fi
}

# Test user-task list
test_user_tasks() {
    log_info "Testing user-task-list feature..."
    
    # Test task-creation list
    local creation_response=$(curl -s "$BASE_URL/api/v1/task-creation/user/$TEST_WALLET?limit=10")
    
    if echo "$creation_response" | grep -q '"success":true'; then
        log_success "Task-creation list retrieved successfully"
    else
        log_error "Failed to retrieve task-creation list"
        echo "$creation_response"
    fi
    
    # Test batch-verification list
    local batch_response=$(curl -s "$BASE_URL/api/v1/batch-verification/user/$TEST_WALLET?limit=10")
    
    if echo "$batch_response" | grep -q '"success":true'; then
        log_success "Batch-verification list retrieved successfully"
    else
        log_error "Failed to retrieve batch-verification list"
        echo "$batch_response"
    fi
}

# Test statistics
test_stats() {
    log_info "Testing statistics feature..."
    
    # Test task-creation statistics
    local creation_stats=$(curl -s "$BASE_URL/api/v1/task-creation/stats?user_wallet=$TEST_WALLET")
    
    if echo "$creation_stats" | grep -q '"success":true'; then
        log_success "Task-creation statistics retrieved successfully"
    else
        log_error "Failed to retrieve task-creation statistics"
        echo "$creation_stats"
    fi
    
    # Test batch-verification statistics
    local batch_stats=$(curl -s "$BASE_URL/api/v1/batch-verification/stats?user_wallet=$TEST_WALLET")
    
    if echo "$batch_stats" | grep -q '"success":true'; then
        log_success "Batch-verification statistics retrieved successfully"
    else
        log_error "Failed to retrieve batch-verification statistics"
        echo "$batch_stats"
    fi
}

# Test traditional-task endpoint
test_traditional_tasks() {
    log_info "Testing traditional-task endpoint..."
    
    local response=$(curl -s -X POST "$BASE_URL/api/v1/tasks/submit" \
        -H "Content-Type: application/json" \
        -d '{
            "user_wallet": "'$TEST_WALLET'",
            "task_type": "twitter_retweet",
            "payload": {
                "tweet_id": "1234567890",
                "twitter_id": "@testuser",
                "retweet_url": "https://twitter.com/testuser/status/1234567890"
            }
        }')
    
    if echo "$response" | grep -q '"success":true'; then
        local task_id=$(echo "$response" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
        log_success "Traditional task submitted successfully, Task ID: $task_id"
        
        # Test retrieving task status
        log_info "Retrieving traditional-task status..."
        local status_response=$(curl -s "$BASE_URL/api/v1/tasks/status/$task_id")
        
        if echo "$status_response" | grep -q '"success":true'; then
            log_success "Traditional-task status retrieved successfully"
        else
            log_error "Failed to retrieve traditional-task status"
            echo "$status_response"
        fi
    else
        log_error "Traditional-task submission failed"
        echo "$response"
    fi
}

# Main test function
main() {
    echo "======================================"
    echo "      New-Feature Test Script"
    echo "======================================"
    echo
    
    # Check service
    check_service
    echo
    
    # Test task creation
    echo "1. Task-Creation Feature Test"
    echo "--------------------------------------"
    test_task_creation
    echo
    
    # Test batch verification
    echo "2. Batch-Verification Feature Test"
    echo "--------------------------------------"
    test_batch_verification
    echo
    
    # Test user-task list
    echo "3. User-Task List Test"
    echo "--------------------------------------"
    test_user_tasks
    echo
    
    # Test statistics
    echo "4. Statistics Test"
    echo "--------------------------------------"
    test_stats
    echo
    
    # Test traditional-task endpoint
    echo "5. Traditional-Task Endpoint Test"
    echo "--------------------------------------"
    test_traditional_tasks
    echo
    
    echo "======================================"
    log_success "All tests completed!"
    echo "======================================"
}

# Show help
show_help() {
    cat << EOF
New-Feature Test Script

Usage:
    $0 [options]

Options:
    --base-url URL      Service base URL (default: http://localhost:8001)
    --wallet WALLET     Test wallet address (default: 0x1234567890abcdef1234567890abcdef12345678)
    --help              Show this help message

Environment variables:
    BASE_URL            Service base URL

Examples:
    $0                                      # Test with default config
    $0 --base-url http://localhost:8080     # Use different service address
    BASE_URL=http://localhost:8080 $0       # Use environment variable
EOF
}

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --base-url)
            BASE_URL="$2"
            shift 2
            ;;
        --wallet)
            TEST_WALLET="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown argument: $1"
            show_help
            exit 1
            ;;
    esac
done

# Execute main function
main