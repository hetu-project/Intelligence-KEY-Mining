#!/bin/bash

# PoCW Subnet-Only Script
# This script runs the subnet consensus system with VLC visualization
# No blockchain integration or KEY mining - pure subnet demonstration

echo "🔹 PoCW SUBNET-ONLY DEMONSTRATION"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Architecture: Pure subnet consensus with VLC graph visualization"
echo ""

# Preserve user's PATH when running with sudo
if [ -n "$SUDO_USER" ]; then
    USER_HOME=$(eval echo ~$SUDO_USER)
    # Common Go installation paths
    export PATH="/usr/local/go/bin:$USER_HOME/go/bin:$USER_HOME/.local/bin:/snap/bin:$PATH"
fi

# Check prerequisites
echo "🔍 Checking prerequisites..."
if ! command -v go &> /dev/null; then
    echo "❌ Go not found. Please install Go >= 1.21."
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo "❌ Node.js not found. Please install Node.js."
    exit 1
fi

echo "✅ All prerequisites found"

# Cleanup function
cleanup() {
    echo ""
    echo "🛑 Cleaning up processes..."
    
    # Stop Dgraph container
    echo "🔴 Stopping Dgraph container..."
    docker stop dgraph-standalone 2>/dev/null || true
    docker rm dgraph-standalone 2>/dev/null || true
    
    # Clean up Dgraph data directory
    echo "🧹 Cleaning up Dgraph data..."
    rm -rf ./dgraph-data 2>/dev/null || true
    
    echo "✅ Cleanup complete"
    exit 0
}

# Set up trap for Ctrl+C
trap cleanup SIGINT SIGTERM

# === PHASE 1: START DGRAPH FOR VLC VISUALIZATION ===
echo ""
echo "🚀 PHASE 1: Starting Dgraph Infrastructure"
echo "────────────────────────────────────────────────"

# === DGRAPH SETUP ===
echo "Setting up Dgraph for VLC event visualization..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "⚠️  Docker not found. Please install Docker to enable VLC visualization."
    echo "❌ Cannot proceed without Dgraph. Exiting..."
    exit 1
fi

echo "🐳 Docker found, setting up Dgraph container..."

# Stop any existing Dgraph containers
echo "🔴 Stopping existing Dgraph containers..."
docker stop dgraph-standalone 2>/dev/null || true
docker rm dgraph-standalone 2>/dev/null || true

# Remove any existing Dgraph data to start fresh
echo "🧹 Cleaning up previous Dgraph data..."
rm -rf ./dgraph-data 2>/dev/null || true
mkdir -p ./dgraph-data

# Clear old graph data from previous runs
echo "🧹 Clearing old graph data files..."
rm -rf ./graph-data 2>/dev/null || true

# Start new Dgraph container with proper setup
echo "🚀 Starting fresh Dgraph container..."
DGRAPH_OUTPUT=$(docker run --rm -d --name dgraph-standalone \
    -p 8080:8080 -p 9080:9080 -p 8000:8000 \
    -v $(pwd)/dgraph-data:/dgraph \
    dgraph/standalone:latest 2>&1)

if [ $? -eq 0 ]; then
    echo "✅ Dgraph container started successfully"
    echo "📊 Dgraph UI: http://localhost:8000"
    DGRAPH_STARTED=true
else
    echo "❌ Failed to start Dgraph container:"
    echo "$DGRAPH_OUTPUT"
    echo "❌ Cannot proceed without Dgraph. Exiting..."
    exit 1
fi

# Wait for Dgraph to be ready
echo "⏳ Waiting for Dgraph to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "✅ Dgraph is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Dgraph failed to start within 30 seconds"
        cleanup
        exit 1
    fi
    sleep 1
    echo -n "."
done

# === PHASE 2: RUN SUBNET CONSENSUS ===
echo ""
echo "🔗 PHASE 2: Starting Subnet Consensus"
echo "────────────────────────────────────────"

echo "🚀 Starting Go subnet system (subnet-only mode)..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Dgraph UI: http://localhost:8000 (VLC Graph Visualization)"
echo "⚠️  Press Ctrl+C to stop and cleanup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Set environment variable to disable bridge (subnet-only mode)
export SUBNET_ONLY_MODE=true

# Run the Go subnet system
go run main.go

# This line will be reached if main.go exits normally
echo ""
echo "🏁 Subnet consensus completed"

# Keep the system running for user interaction and debugging
echo ""
echo "🔄 Entering interactive mode..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Dgraph UI: http://localhost:8000 (VLC Graph Visualization)"
echo "⚠️  Press Ctrl+C to stop and cleanup when you're done exploring"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Forever loop to keep services running
while true; do
    sleep 5
    # Optional: Add heartbeat or status checks here
done