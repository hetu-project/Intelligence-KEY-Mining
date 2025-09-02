// Proof-of-Causal-Work (PoCW) Demonstration
//
// This is the main entry point for the PoCW subnet demonstration, showcasing
// a distributed consensus system where AI agents (miners) process user tasks
// while validators ensure quality through Vector Logical Clock consistency.
//
// Architecture:
//   - Miners: AI entities that process user requests and maintain causal consistency
//   - Validators: Quality assessors that vote on outputs using Byzantine Fault Tolerant consensus
//   - VLC: Vector Logical Clocks ensure causal ordering of operations
//   - Intelligence Money: Novel concept of verifiable work tokens (future enhancement)
//
// Usage:
//
//	go run main.go    # Run the PoCW subnet demonstration
//
// The subnet demo processes 7 test scenarios including info requests, quality assessment,
// validator consensus, and user interaction patterns.
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/dgraph"
	"github.com/hetu-project/Intelligence-KEY-Mining/subnet/demo"
)

// waitForDgraph waits for Dgraph to be fully ready
func waitForDgraph() error {
	maxRetries := 15
	retryInterval := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Try to connect to health endpoint
		resp, err := http.Get("http://localhost:8080/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		fmt.Printf("Dgraph not ready yet (attempt %d/%d), waiting %v...\n", i+1, maxRetries, retryInterval)
		time.Sleep(retryInterval)
	}

	return fmt.Errorf("dgraph not ready after %d attempts", maxRetries)
}

// main is the entry point for the PoCW subnet demonstration system.
func main() {
	runSubnetDemo()
}

// runSubnetDemo executes the complete PoCW subnet demonstration.
// Creates a coordinator with 1 miner and 4 validators, then processes
// 7 predefined test scenarios to showcase all aspects of the protocol:
//
// Demo Features:
//   - Normal task processing with immediate solutions
//   - Info request scenarios where miners ask for additional context
//   - Quality assessment and Byzantine Fault Tolerant consensus
//   - User interaction including acceptance and rejection patterns
//   - Vector Logical Clock consistency validation
//
// The demo uses hardcoded scenarios for predictable testing while
// utilizing the same core components that would work with real AI models.
func runSubnetDemo() {
	fmt.Println("=== PoCW Subnet Demo with VLC Graph Visualization ===")
	fmt.Println("Run './setup.sh' first to start Dgraph visualization")
	fmt.Println("")

	// Try to initialize Dgraph gracefully
	fmt.Println("Waiting for Dgraph to be ready...")
	if err := waitForDgraph(); err != nil {
		fmt.Printf("Dgraph not available: %v\n", err)
		fmt.Println("Running demo without graph visualization...")
	} else {
		fmt.Println("Initializing Dgraph connection...")
		dgraph.InitDgraph("localhost:9080")
		fmt.Println("Dgraph initialized successfully!")
	}

	coordinator := demo.NewDemoCoordinator("subnet-001")
	coordinator.RunDemo()
}
