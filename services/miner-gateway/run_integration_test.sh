#!/bin/bash

echo "🧪 Running Integration Tests: Batch Verification → PoCW → Points Distribution Flow"
echo "=================================================================================="

cd "$(dirname "$0")"

# Check if necessary dependencies are installed
echo "📦 Checking test dependencies..."
go mod tidy

# Run integration tests
echo "🚀 Starting integration tests..."
go test -v ./tests/integration_test.go -run TestTaskStatusFlow

echo ""
echo "🔄 Running multiple users flow test..."
go test -v ./tests/integration_test.go -run TestMultipleUsersOneTask

echo ""
echo "📝 Running task creation flow test..."
go test -v ./tests/integration_test.go -run TestTaskCreationFlow

echo ""
echo "🛡️ Running duplicate completion protection test..."
go test -v ./tests/integration_test.go -run TestNoDuplicateCompletion

echo ""
echo "🔄 Running batch verification flow test..."
go test -v ./tests/integration_test.go -run TestBatchVerificationFlow

echo ""
echo "❌ Running verification failure handling test..."
go test -v ./tests/integration_test.go -run TestVerificationFailureHandling

echo ""
echo "✅ All integration tests completed!"
