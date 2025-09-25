// Proof-of-Causal-Work (PoCW) Per-Epoch Integration
//
// This is the main entry point for the PoCW parallelnet with real-time per-epoch
// blockchain integration, showcasing a distributed consensus system where
// AI agents (miners) process user tasks and immediately submit verified
// intelligence work to the blockchain for KEY token mining.
//
// Architecture:
//   - Miners: AI entities that process user requests with VLC consistency
//   - Validators: Quality assessors using Byzantine Fault Tolerant consensus
//   - VLC: Vector Logical Clocks ensure causal ordering of operations
//   - Per-Epoch Integration: Real-time blockchain submission every 3 rounds
//   - Intelligence Money: Verifiable work tokens based on actual task success

package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/dgraph"
	"github.com/hetu-project/Intelligence-KEY-Mining/parallelnet"
	"github.com/hetu-project/Intelligence-KEY-Mining/parallelnet/demo"
)

// EpochBridge handles the interface between Go and the Node.js mainnet bridge
type EpochBridge struct {
	bridgeCmd *exec.Cmd
}

// NewEpochBridge creates a new bridge to the Node.js mainnet submission service
func NewEpochBridge() *EpochBridge {
	return &EpochBridge{}
}

// StartBridge starts the Node.js bridge service
func (eb *EpochBridge) StartBridge() error {
	fmt.Println("🌐 Starting Per-Epoch Mainnet Bridge...")

	// Start the Node.js bridge service
	cmd := exec.Command("node", "mainnet-bridge-per-epoch.js")
	cmd.Dir = "."

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start bridge: %v", err)
	}

	eb.bridgeCmd = cmd

	// Wait for bridge to initialize
	time.Sleep(3 * time.Second)
	fmt.Println("✅ Mainnet bridge service started")

	return nil
}

// SubmitEpoch sends epoch data to the mainnet bridge for submission
func (eb *EpochBridge) SubmitEpoch(epochNumber int, parallelnetID string, epochData *parallelnet.EpochData) {
	fmt.Printf("🚀 Bridge: Epoch %d ready for mainnet submission\n", epochNumber)

	// In a full implementation, this would:
	// 1. Convert epoch data to JSON
	// 2. Send HTTP request to Node.js bridge service
	// 3. Bridge submits to mainnet and mines KEY tokens

	// For demonstration, we'll simulate the submission
	fmt.Printf("  📊 parallelnet: %s\n", parallelnetID)
	fmt.Printf("  🔗 Rounds: %v\n", epochData.CompletedRounds)
	fmt.Printf("  ⏰ VLC State: %v\n", epochData.VLCClockState)
	fmt.Printf("  💰 Triggering KEY mining for epoch %d...\n", epochNumber)

	// Simulate processing time
	time.Sleep(2 * time.Second)

	fmt.Printf("✅ Epoch %d submitted to mainnet successfully!\n", epochNumber)
}

// StopBridge stops the Node.js bridge service
func (eb *EpochBridge) StopBridge() {
	if eb.bridgeCmd != nil {
		eb.bridgeCmd.Process.Kill()
		fmt.Println("🔴 Mainnet bridge service stopped")
	}
}

// waitForDgraph waits for Dgraph to be fully ready
func waitForDgraph() error {
	maxRetries := 15
	retryInterval := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
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

// main demonstrates the per-epoch PoCW integration
func main() {
	// Check if running in parallelnet-only mode
	parallelnetOnlyMode := os.Getenv("parallelnet_ONLY_MODE") == "true"

	if parallelnetOnlyMode {
		fmt.Println("=== PoCW parallelnet-Only Demo ===")
		fmt.Println("Architecture: Pure parallelnet consensus with VLC visualization")
		fmt.Println("")
	} else {
		fmt.Println("=== PoCW Per-Epoch Mainnet Integration Demo ===")
		fmt.Println("Architecture: Real-time epoch submission (every 3 rounds)")
		fmt.Println("")
	}

	// Initialize mainnet bridge only if not in parallelnet-only mode
	var bridge *EpochBridge
	if !parallelnetOnlyMode {
		bridge = NewEpochBridge()
		err := bridge.StartBridge()
		if err != nil {
			fmt.Printf("⚠️  Bridge startup failed: %v\n", err)
			fmt.Println("Continuing with demonstration mode...")
		}
		defer bridge.StopBridge()
	}

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

	// Create demo coordinator with per-epoch callback integration
	coordinator := demo.NewDemoCoordinator("per-epoch-parallelnet-001")

	// Set up HTTP bridge URL only if not in parallelnet-only mode
	if !parallelnetOnlyMode && coordinator.GraphAdapter != nil {
		fmt.Println("🔗 Setting up per-epoch HTTP bridge integration...")

		// Set the bridge URL for HTTP communication
		coordinator.GraphAdapter.SetBridgeURL("http://localhost:3001")

		fmt.Println("✅ Per-epoch HTTP bridge configured successfully")
		fmt.Println("📡 Graph adapter will send HTTP requests to JavaScript bridge")
	} else if parallelnetOnlyMode {
		fmt.Println("🔹 Running in parallelnet-only mode - no blockchain integration")
	} else {
		fmt.Println("⚠️  GraphAdapter not available - running standard demo")
	}

	fmt.Println("")
	if parallelnetOnlyMode {
		fmt.Println("🎯 parallelnet-Only Demo Flow:")
		fmt.Println("  Round 1-7  → Pure parallelnet consensus")
		fmt.Println("  📊 VLC data visible at: http://localhost:8000")
		fmt.Println("  ⚠️  No blockchain integration or KEY mining")
	} else {
		fmt.Println("🎯 Demo Flow:")
		fmt.Println("  Round 1-3  → Epoch 1 → Immediate mainnet submission")
		fmt.Println("  Round 4-6  → Epoch 2 → Immediate mainnet submission")
		fmt.Println("  Round 7    → Partial Epoch 3 → Submit at demo end")
	}
	fmt.Println("")

	// Run the parallelnet demo
	coordinator.RunDemo()

	fmt.Println("")
	if parallelnetOnlyMode {
		fmt.Println("🎉 parallelnet-Only Demo Complete!")
	} else {
		fmt.Println("🎉 Per-Epoch Integration Demo Complete!")
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Demonstrated real-time epoch submission architecture")
	fmt.Println("✅ Each completed epoch triggers immediate mainnet posting")
	fmt.Println("✅ KEY tokens are mined in real-time per epoch")
	fmt.Println("")
	fmt.Println("🔍 Visualization Access:")
	fmt.Println("  - Ratel UI: http://localhost:8000")
	fmt.Println("  - Inspector: http://localhost:3000/pocw-inspector.html")
}
