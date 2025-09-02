// Package subnet - Graph Adapter for VLC-Consistent Round Visualization
//
// This file integrates the PoCW subnet architecture with the Dgraph-based
// causal event graph system. It creates visual representations of the
// round-based VLC interactions between Miner and Validator-1.
package subnet

import (
	"fmt"
	"sync"

	"github.com/hetu-project/Intelligence-KEY-Mining/dgraph"
	"github.com/hetu-project/Intelligence-KEY-Mining/vlc"
)

// SubnetGraphAdapter adapts PoCW subnet events for causal graph visualization.
// It tracks round-based interactions and creates Dgraph events that show
// the VLC-consistent causal ordering of the subnet protocol.
//
// Event Types Tracked:
//   - UserInput: User submits task to Validator-1 (round start)
//   - MinerProcess: Miner processes input with VLC increment
//   - ValidatorVote: Validators perform quality assessment
//   - UserFeedback: User provides feedback (round end)
//   - InfoRequest: Miner requests additional information
//   - InfoResponse: User provides additional context
//
// VLC Integration:
//   - Each event includes VLC clock state for causal ordering
//   - Parent-child relationships reflect VLC causality
//   - Only events from VLC participants (Miner=1, Validator-1=2) have full VLC data
type SubnetGraphAdapter struct {
	EventGraph        *dgraph.EventGraph // Dgraph event graph for visualization
	SubnetID          string             // Subnet identifier
	mu                sync.RWMutex       // Protects event tracking state
	roundCounters     map[string]int     // Per-request round counters
	completedRounds   []string           // Track completed rounds for epoch creation
	epochCount        int                // Current epoch number
	lastEventInChain  string             // Last event for continuous chaining
	genesisEventID    string             // Genesis state event ID
	roundsInEpoch     int                // Counter for rounds within current epoch
}

// NewSubnetGraphAdapter creates a new graph adapter for subnet visualization
func NewSubnetGraphAdapter(subnetID string, nodeID int, nodeAddr string) *SubnetGraphAdapter {
	sga := &SubnetGraphAdapter{
		EventGraph:       dgraph.NewEventGraph(nodeID, nodeAddr),
		SubnetID:         subnetID,
		roundCounters:    make(map[string]int),
		completedRounds:  make([]string, 0),
		epochCount:       0,
		lastEventInChain: "",
		genesisEventID:   "",
		roundsInEpoch:    0,
	}
	
	// Create Genesis State immediately
	sga.createGenesisState()
	return sga
}

// createGenesisState creates the initial genesis state for the blockchain
func (sga *SubnetGraphAdapter) createGenesisState() {
	eventName := "GenesisState"
	key := "genesis_0"
	value := "Blockchain Genesis: PoCW Subnet initialized with VLC consensus"
	
	// Genesis has empty VLC clock (no participants yet)
	clockMap := make(map[int]int)
	
	genesisEventID := sga.EventGraph.AddEvent(
		eventName,
		key, 
		value,
		clockMap,
		nil, // Genesis has no parents
	)
	
	sga.genesisEventID = genesisEventID
	sga.lastEventInChain = genesisEventID
}

// TrackUserInput records user input that starts a round (validator VLC increment)
func (sga *SubnetGraphAdapter) TrackUserInput(requestID string, input string, validatorClock *vlc.Clock, parentEventID string) string {
	sga.mu.Lock()
	defer sga.mu.Unlock()

	// Increment round counter for this request
	sga.roundCounters[requestID]++
	roundNum := sga.roundCounters[requestID]

	// Create semantic event name
	eventName := "UserInput"
	key := fmt.Sprintf("user_input_%d", roundNum)
	
	// Convert VLC clock to map format
	clockMap := vlcToMap(validatorClock)

	// Create descriptive value
	value := fmt.Sprintf("User submits: %s", input)

	// Connect to blockchain chain - always connect to last event in chain
	var parents []string
	if sga.lastEventInChain != "" {
		parents = append(parents, sga.lastEventInChain)
	}
	if parentEventID != "" && parentEventID != sga.lastEventInChain {
		parents = append(parents, parentEventID)
	}

	eventID := sga.EventGraph.AddEvent(
		eventName,
		key,
		value,
		clockMap,
		parents,
	)

	return eventID
}

// TrackMinerResponse records miner processing with semantic naming (miner VLC increment)
func (sga *SubnetGraphAdapter) TrackMinerResponse(requestID string, response *MinerResponseMessage, parentEventID string) string {
	sga.mu.Lock()
	defer sga.mu.Unlock()

	// Create semantic event name based on miner response type
	var eventName, key, value string
	if response.OutputType == OutputReady {
		eventName = "MinerOutput"
		key = fmt.Sprintf("miner_output_%d", response.InputNumber)
		value = fmt.Sprintf("Miner provides: %s", response.Output)
	} else {
		eventName = "InfoRequest"
		key = fmt.Sprintf("info_request_%d", response.InputNumber)
		value = fmt.Sprintf("Miner requests: %s", response.InfoRequest)
	}

	clockMap := vlcToMap(response.VLCClock)

	// Add event with parent relationship
	var parents []string
	if parentEventID != "" {
		parents = append(parents, parentEventID)
	}

	eventID := sga.EventGraph.AddEvent(
		eventName,
		key,
		value,
		clockMap,
		parents,
	)

	return eventID
}

// TrackInfoResponse records user providing additional context (validator VLC increment)
func (sga *SubnetGraphAdapter) TrackInfoResponse(requestID string, additionalInfo string, validatorClock *vlc.Clock, parentEventID string) string {
	sga.mu.Lock()
	defer sga.mu.Unlock()

	eventName := "InfoResponse"
	key := fmt.Sprintf("info_response_%s", requestID)
	value := fmt.Sprintf("User clarifies: %s", additionalInfo)

	clockMap := vlcToMap(validatorClock)

	var parents []string
	if parentEventID != "" {
		parents = append(parents, parentEventID)
	}

	eventID := sga.EventGraph.AddEvent(
		eventName,
		key,
		value,
		clockMap,
		parents,
	)

	return eventID
}

// TrackRoundComplete records round completion with comprehensive workflow result (validator VLC increment)
func (sga *SubnetGraphAdapter) TrackRoundComplete(requestID string, roundNum int, validatorClock *vlc.Clock, consensusResult string, userFeedback string, userAccept bool, finalResult string, parentEventID string) string {
	sga.mu.Lock()
	defer sga.mu.Unlock()

	// Determine semantic event name based on final outcome
	var eventName string
	if userAccept && finalResult == "OUTPUT DELIVERED TO USER" {
		eventName = "RoundSuccess" // Will be colored green
	} else {
		eventName = "RoundFailed" // Will be colored red
	}

	key := fmt.Sprintf("round_%d_complete", roundNum)
	
	// Create comprehensive value showing the complete workflow result
	value := fmt.Sprintf("Round %d: %s | Consensus: %s | User: %s", 
		roundNum, finalResult, consensusResult, userFeedback)

	clockMap := vlcToMap(validatorClock)

	var parents []string
	if parentEventID != "" {
		parents = append(parents, parentEventID)
	}

	eventID := sga.EventGraph.AddEvent(
		eventName,
		key,
		value,
		clockMap,
		parents,
	)

	// Track completed round and update chain
	sga.completedRounds = append(sga.completedRounds, eventID)
	sga.lastEventInChain = eventID
	sga.roundsInEpoch++
	
	// Determine what comes next in the blockchain structure
	if sga.roundsInEpoch == 3 {
		// End of epoch - create EpochFinalized
		epochEventID := sga.createEpochFinalization(validatorClock, eventID)
		sga.lastEventInChain = epochEventID
		sga.roundsInEpoch = 0 // Reset for next epoch
		return epochEventID
	} else {
		// Middle of epoch - create NextRound connector
		nextRoundEventID := sga.createNextRoundConnector(validatorClock, eventID)
		sga.lastEventInChain = nextRoundEventID
		return nextRoundEventID
	}
}

// createNextRoundConnector creates transition nodes between rounds within an epoch
func (sga *SubnetGraphAdapter) createNextRoundConnector(validatorClock *vlc.Clock, parentRoundEventID string) string {
	eventName := "NextRound"
	key := fmt.Sprintf("next_round_%d_%d", sga.epochCount+1, sga.roundsInEpoch+1)
	value := fmt.Sprintf("Epoch %d: Transition to Round %d", sga.epochCount+1, sga.roundsInEpoch+1)
	
	clockMap := vlcToMap(validatorClock)
	
	nextRoundEventID := sga.EventGraph.AddEvent(
		eventName,
		key,
		value,
		clockMap,
		[]string{parentRoundEventID}, // Connect to the completed round
	)
	
	return nextRoundEventID
}

// createEpochFinalization creates epoch milestone events that chain rounds together blockchain-style
func (sga *SubnetGraphAdapter) createEpochFinalization(validatorClock *vlc.Clock, parentRoundEventID string) string {
	sga.epochCount++
	
	eventName := "EpochFinalized"
	key := fmt.Sprintf("epoch_%d_finalized", sga.epochCount)
	
	value := fmt.Sprintf("Epoch %d: Finalized with 3 rounds | VLC State: Miner=%d, Validator=%d", 
		sga.epochCount, 
		validatorClock.Values[1], 
		validatorClock.Values[2])
	
	clockMap := vlcToMap(validatorClock)
	
	// EpochFinalized connects to the last round of the epoch
	epochEventID := sga.EventGraph.AddEvent(
		eventName,
		key,
		value,
		clockMap,
		[]string{parentRoundEventID}, // Connect to round 3 of the epoch
	)
	
	return epochEventID
}

// CommitGraph commits all tracked events to Dgraph for visualization
func (sga *SubnetGraphAdapter) CommitGraph() error {
	sga.mu.Lock()
	eventCount := len(sga.EventGraph.Events) // Get count before committing (as commit clears events)
	sga.mu.Unlock()
	
	// Handle case where Dgraph is not available
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Dgraph commit failed: %v (continuing without graph visualization)\n", r)
		}
	}()
	
	err := sga.EventGraph.CommitToGraph()
	if err == nil {
		fmt.Printf("Committed %d events to Dgraph successfully!\n", eventCount)
	}
	return err
}

// GetEventCount returns the number of events tracked
func (sga *SubnetGraphAdapter) GetEventCount() int {
	sga.mu.RLock()
	defer sga.mu.RUnlock()
	
	return len(sga.EventGraph.Events)
}

// vlcToMap converts VLC clock to map format for JSON serialization
func vlcToMap(clock *vlc.Clock) map[int]int {
	if clock == nil {
		return make(map[int]int)
	}
	
	// Convert uint64 keys to int keys for JSON compatibility
	result := make(map[int]int)
	for k, v := range clock.Values {
		result[int(k)] = int(v)
	}
	return result
}

// PrintGraphSummary prints a summary of tracked events
func (sga *SubnetGraphAdapter) PrintGraphSummary() {
	sga.mu.RLock()
	defer sga.mu.RUnlock()

	fmt.Printf("\n=== Subnet Graph Summary ===\n")
	fmt.Printf("Subnet ID: %s\n", sga.SubnetID)
	fmt.Printf("Total Events: %d\n", len(sga.EventGraph.Events))
	fmt.Printf("Requests Processed: %d\n", len(sga.roundCounters))
	
	// Print event breakdown by type
	eventTypes := make(map[string]int)
	for _, event := range sga.EventGraph.Events {
		eventTypes[event.Name]++
	}
	
	fmt.Printf("\nEvent Type Breakdown:\n")
	for eventType, count := range eventTypes {
		fmt.Printf("  %s: %d\n", eventType, count)
	}
	
	fmt.Printf("\nRound Counters:\n")
	for requestID, rounds := range sga.roundCounters {
		fmt.Printf("  %s: %d rounds\n", requestID, rounds)
	}
}