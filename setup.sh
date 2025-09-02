#!/bin/bash

# PoCW Subnet Demo Setup Script
# This script sets up the required Docker containers and environment for the PoCW demonstration

set -e  # Exit on any error

echo "=== PoCW Subnet Demo Setup ==="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Error: Docker is not installed"
    echo "Please install Docker first: https://docs.docker.com/get-docker/"
    exit 1
fi

echo "✅ Docker found"

# Check if Docker daemon is running
if ! docker info &> /dev/null; then
    echo "❌ Error: Docker daemon is not running"
    echo "Please start Docker daemon first"
    exit 1
fi

echo "✅ Docker daemon is running"

# Function to check if container is running
is_container_running() {
    docker ps --filter "name=$1" --format "{{.Names}}" | grep -q "^$1$"
}

# Function to wait for Dgraph to be ready
wait_for_dgraph() {
    echo "⏳ Waiting for Dgraph to be ready..."
    local max_attempts=20
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
            echo "✅ Dgraph is ready!"
            return 0
        fi
        
        echo "   Attempt $attempt/$max_attempts - Dgraph not ready yet..."
        sleep 3
        ((attempt++))
    done
    
    echo "❌ Error: Dgraph failed to start after $max_attempts attempts"
    return 1
}

# Setup Dgraph container
echo ""
echo "🐳 Setting up Dgraph container..."

if is_container_running "dgraph-standalone"; then
    echo "📦 Dgraph container is already running"
else
    # Stop and remove any existing container
    if docker ps -a --filter "name=dgraph-standalone" --format "{{.Names}}" | grep -q "dgraph-standalone"; then
        echo "🛑 Stopping existing dgraph-standalone container..."
        docker stop dgraph-standalone > /dev/null 2>&1 || true
        docker rm dgraph-standalone > /dev/null 2>&1 || true
    fi
    
    echo "🚀 Starting new Dgraph container..."
    docker run --rm -d --name dgraph-standalone \
        -p 8080:8080 -p 9080:9080 -p 8000:8000 \
        dgraph/standalone
    
    if [ $? -eq 0 ]; then
        echo "✅ Dgraph container started successfully"
    else
        echo "❌ Failed to start Dgraph container"
        exit 1
    fi
fi

# Wait for Dgraph to be ready
if ! wait_for_dgraph; then
    echo "❌ Setup failed: Dgraph is not responding"
    echo "💡 Try running: docker logs dgraph-standalone"
    exit 1
fi

echo ""
echo "🎉 Setup completed successfully!"
echo ""
echo "🌐 Dgraph Ratel UI: http://localhost:8000"
echo "📊 Dgraph API: http://localhost:8080"
echo "🔧 GraphQL: http://localhost:8080/graphql"
echo ""
echo "▶️  Now run: go run main.go"
echo ""
echo "🛑 To cleanup later: ./teardown.sh"