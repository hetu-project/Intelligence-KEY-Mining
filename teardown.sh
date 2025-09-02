#!/bin/bash

# PoCW Subnet Demo Teardown Script
# This script stops and cleans up Docker containers used by the PoCW demonstration

set -e  # Exit on any error

echo "=== PoCW Subnet Demo Teardown ==="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found - nothing to teardown"
    exit 0
fi

# Function to check if container exists (running or stopped)
container_exists() {
    docker ps -a --filter "name=$1" --format "{{.Names}}" | grep -q "^$1$"
}

# Function to check if container is running
is_container_running() {
    docker ps --filter "name=$1" --format "{{.Names}}" | grep -q "^$1$"
}

# Stop and remove Dgraph container
echo "🐳 Cleaning up Dgraph container..."

if container_exists "dgraph-standalone"; then
    if is_container_running "dgraph-standalone"; then
        echo "🛑 Stopping dgraph-standalone container..."
        docker stop dgraph-standalone
        echo "✅ Dgraph container stopped"
    else
        echo "📦 Dgraph container already stopped"
    fi
    
    # Remove the container (if it wasn't started with --rm)
    echo "🗑️  Removing dgraph-standalone container..."
    docker rm dgraph-standalone 2>/dev/null || true
    echo "✅ Dgraph container removed"
else
    echo "📝 No dgraph-standalone container found"
fi

# Clean up any dangling volumes (optional)
echo ""
echo "🧹 Cleaning up unused Docker resources..."
docker system prune -f --volumes > /dev/null 2>&1 || true

echo ""
echo "🎉 Teardown completed successfully!"
echo ""
echo "💡 To start again, run: ./setup.sh"