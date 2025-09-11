#!/bin/bash

# Points-system test script
# Tests the complete flow of point distribution

set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
POINTS_SERVICE_URL="http://localhost:8087"
SBT_SERVICE_URL="http://localhost:8086"
TEST_WALLET_1="0x1234567890abcdef1234567890abcdef12345678"
TEST_WALLET_2="0xabcdef1234567890abcdef1234567890abcdef12"
TEST_WALLET_3="0x9876543210fedcba9876543210fedcba98765432"

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

# Wait until service is ready
wait_for_service() {
    local url=$1
    local service_name=$2
    local max_attempts=30
    local attempt=1
    
    log_info "Waiting for $service_name to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f "$url/health" &> /dev/null; then
            log_success "$service_name is ready"
            return 0
        fi
        
        log_info "Attempt $attempt/$max_attempts - waiting for $service_name..."
        sleep 2
        ((attempt++))
    done
    
    log_error "$service_name startup timed out"
    return 1
}

# Test health-check endpoints
test_health_check() {
    log_info "=== Testing service health checks ==="
    
    # Points service
    if curl -f "$POINTS_SERVICE_URL/health" &> /dev/null; then
        log_success "Points-service health check passed"
    else
        log_error "Points-service health check failed"
        return 1
    fi
    
    # SBT service
    if curl -f "$SBT_SERVICE_URL/health" &> /dev/null; then
        log_success "SBT-service health check passed"
    else
        log_error "SBT-service health check failed"
        return 1
    fi
}

# Test points configuration
test_points_config() {
    log_info "=== Testing points configuration ==="
    
    local response=$(curl -s "$POINTS_SERVICE_URL/api/v1/points/config")
    
    if echo "$response" | jq -e '.status == "success"' &> /dev/null; then
        log_success "Points configuration retrieved successfully"
        echo "$response" | jq '.data'
    else
        log_error "Failed to retrieve points configuration"
        echo "$response"
        return 1
    fi
}

# Test points distribution
test_points_distribution() {
    log_info "=== Testing points distribution ==="
    
    # Build test payload
    local test_data=$(cat <<EOF
{
    "batch_id": "test_batch_$(date +%s)",
    "trigger_type": "validator_voting",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "tasks": [
        {
            "user_wallet": "$TEST_WALLET_1",
            "task_type": "creation",
            "vlc_value": 2,
            "task_id": "task_creation_1"
        },
        {
            "user_wallet": "$TEST_WALLET_2",
            "task_type": "creation", 
            "vlc_value": 1,
            "task_id": "task_creation_2"
        },
        {
            "user_wallet": "$TEST_WALLET_1",
            "task_type": "retweet",
            "vlc_value": 3,
            "task_id": "task_retweet_1"
        },
        {
            "user_wallet": "$TEST_WALLET_3",
            "task_type": "retweet",
            "vlc_value": 1,
            "task_id": "task_retweet_2"
        }
    ]
}
EOF
)
    
    log_info "Sending points-distribution request..."
    local response=$(curl -s -X POST "$POINTS_SERVICE_URL/api/v1/points/distribute" \
        -H "Content-Type: application/json" \
        -d "$test_data")
    
    if echo "$response" | jq -e '.status == "success"' &> /dev/null; then
        log_success "Points-distribution request succeeded"
        
        # Show distribution result
        echo "=== Distribution Result ==="
        echo "$response" | jq '.data | {
            batch_id,
            status,
            total_pool_points,
            creation_points,
            retweet_points,
            total_creation_vlc,
            total_retweet_vlc,
            user_count: (.user_allocations | length)
        }'
        
        # Show user-allocation details
        echo
        echo "=== User Points Allocation ==="
        echo "$response" | jq -r '.data.user_allocations[] | "User: \(.user_wallet[0:10])... | Creation VLC: \(.creation_vlc) | Retweet VLC: \(.retweet_vlc) | Total Points: \(.rounded_points) | Status: \(.update_status)"'
        
    else
        log_error "Points-distribution request failed"
        echo "$response"
        return 1
    fi
}

# Test user-points query
test_user_points_query() {
    log_info "=== Testing user-points query ==="
    
    for wallet in "$TEST_WALLET_1" "$TEST_WALLET_2" "$TEST_WALLET_3"; do
        log_info "Querying points for user: ${wallet:0:10}..."
        
        local response=$(curl -s "$POINTS_SERVICE_URL/api/v1/points/user/$wallet")
        
        if echo "$response" | jq -e '.status == "success"' &> /dev/null; then
            local points=$(echo "$response" | jq -r '.data.total_points')
            log_success "User ${wallet:0:10}... points: $points"
        else
            log_warning "Points query failed for user ${wallet:0:10}... (user may not exist)"
        fi
    done
}

# Test points-history query
test_points_history() {
    log_info "=== Testing points-history query ==="
    
    local response=$(curl -s "$POINTS_SERVICE_URL/api/v1/points/history/$TEST_WALLET_1?limit=10")
    
    if echo "$response" | jq -e '.status == "success"' &> /dev/null; then
        local count=$(echo "$response" | jq -r '.data.count')
        log_success "User ${TEST_WALLET_1:0:10}... history record count: $count"
        
        if [ "$count" -gt 0 ]; then
            echo "Latest records:"
            echo "$response" | jq -r '.data.history[0:3][] | "Date: \(.date) | Source: \(.source) | Points: \(.points)"'
        fi
    else
        log_warning "Points-history query failed"
    fi
}

# Test points statistics
test_points_stats() {
    log_info "=== Testing points statistics ==="
    
    local response=$(curl -s "$POINTS_SERVICE_URL/api/v1/points/stats")
    
    if echo "$response" | jq -e '.status == "success"' &> /dev/null; then
        log_success "Points-statistics query succeeded"
        echo "$response" | jq '.data | {
            total_distributions,
            total_points_issued,
            active_users,
            avg_points_per_user
        }'
    else
        log_error "Points-statistics query failed"
        echo "$response"
        return 1
    fi
}

# Test SBT dynamic data
test_sbt_dynamic_data() {
    log_info "=== Testing SBT dynamic-data integration ==="
    
    # Note: user must already be registered in SBT system
    local response=$(curl -s "$SBT_SERVICE_URL/api/v1/sbt/dynamic/$TEST_WALLET_1")
    
    if echo "$response" | jq -e '.dynamic_attributes' &> /dev/null; then
        log_success "SBT dynamic-data query succeeded"
        
        # Look for points-related attributes
        local total_points=$(echo "$response" | jq -r '.dynamic_attributes[] | select(.trait_type == "Total Points") | .value')
        if [ "$total_points" != "null" ] && [ "$total_points" != "" ]; then
            log_success "Total points in SBT: $total_points"
        else
            log_warning "No points data found in SBT"
        fi
    else
        log_warning "SBT dynamic-data query failed (user may not be registered)"
    fi
}

# Stress test
test_performance() {
    log_info "=== Points-system performance test ==="
    
    log_info "Running concurrent points-distribution test..."
    
    # Launch multiple concurrent requests
    for i in {1..5}; do
        (
            local batch_id="perf_test_${i}_$(date +%s)"
            local test_data=$(cat <<EOF
{
    "batch_id": "$batch_id",
    "trigger_type": "validator_voting",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "tasks": [
        {
            "user_wallet": "0x${i}234567890abcdef1234567890abcdef12345678",
            "task_type": "creation",
            "vlc_value": $i,
            "task_id": "perf_task_$i"
        }
    ]
}
EOF
)
            
            curl -s -X POST "$POINTS_SERVICE_URL/api/v1/points/distribute" \
                -H "Content-Type: application/json" \
                -d "$test_data" > /dev/null
            
            if [ $? -eq 0 ]; then
                echo "Concurrent request $i succeeded"
            else
                echo "Concurrent request $i failed"
            fi
        ) &
    done
    
    # Wait for all background jobs
    wait
    
    log_success "Concurrent test completed"
}

# Main test function
main() {
    log_info "=== Points-system integration test ==="
    
    # Check required tools
    if ! command -v curl &> /dev/null; then
        log_error "curl not found, please install curl"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_error "jq not found, please install jq"
        exit 1
    fi
    
    # Wait for services
    wait_for_service "$POINTS_SERVICE_URL" "points-service" || exit 1
    wait_for_service "$SBT_SERVICE_URL" "sbt-service" || exit 1
    
    # Run tests
    local failed_tests=0
    
    test_health_check || ((failed_tests++))
    echo
    
    test_points_config || ((failed_tests++))
    echo
    
    test_points_distribution || ((failed_tests++))
    echo
    
    test_user_points_query || ((failed_tests++))
    echo
    
    test_points_history || ((failed_tests++))
    echo
    
    test_points_stats || ((failed_tests++))
    echo
    
    test_sbt_dynamic_data || ((failed_tests++))
    echo
    
    # Optional performance test
    if [ "${1:-}" = "--performance" ]; then
        test_performance || ((failed_tests++))
        echo
    fi
    
    # Summary
    if [ $failed_tests -eq 0 ]; then
        log_success "=== All tests passed! Points system is working correctly ==="
    else
        log_error "=== $failed_tests test(s) failed, please check system status ==="
        exit 1
    fi
}

# Execute main function
main "$@"