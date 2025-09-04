// Package subnet - Graph Adapter for VLC-Consistent Round Visualization
//
// This file integrates the PoCW subnet architecture with the Dgraph-based
// causal event graph system. It creates visual representations of the
// round-based VLC interactions between Miner and Validator-1.
package subnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/dgraph"
	"github.com/hetu-project/Intelligence-KEY-Mining/vlc"
)

// EpochFinalizedCallback is called when an epoch is finalized
type EpochFinalizedCallback func(epochNumber int, subnetID string, epochData *EpochData)

// RoundData contains detailed information about a single round
type RoundData struct {
	RoundNumber     int                 `json:"roundNumber"`
	RequestID       string              `json:"requestId"`
	UserInput       string              `json:"userInput"`
	MinerOutput     string              `json:"minerOutput"`
	MinerOutputType string              `json:"minerOutputType"`
	InfoRequest     string              `json:"infoRequest,omitempty"`
	InfoResponse    string              `json:"infoResponse,omitempty"`
	ConsensusResult string              `json:"consensusResult"`
	UserFeedback    string              `json:"userFeedback"`
	UserAccept      bool                `json:"userAccept"`
	FinalResult     string              `json:"finalResult"`
	VLCClockState   map[int]int         `json:"vlcClockState"`
	Success         bool                `json:"success"`
}

// EpochData contains the data for a completed epoch
type EpochData struct {
	EpochNumber       int                 `json:"epochNumber"`
	SubnetID          string              `json:"subnetId"`
	CompletedRounds   []string            `json:"completedRounds"`         // Legacy event IDs
	DetailedRounds    []RoundData         `json:"detailedRounds"`          // Rich round data
	VLCClockState     map[int]int         `json:"vlcClockState"`
	EpochEventID      string              `json:"epochEventId"`
	ParentRoundEventID string             `json:"parentRoundEventId"`
}

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
	EventGraph        *dgraph.EventGraph     // Dgraph event graph for visualization
	SubnetID          string                 // Subnet identifier
	mu                sync.RWMutex           // Protects event tracking state
	roundCounters     map[string]int         // Per-request round counters
	completedRounds   []string               // Track completed rounds for epoch creation
	epochCount        int                    // Current epoch number
	lastEventInChain  string                 // Last event for continuous chaining
	genesisEventID    string                 // Genesis state event ID
	roundsInEpoch     int                    // Counter for rounds within current epoch
	epochCallback     EpochFinalizedCallback // Callback triggered when epoch is finalized
	bridgeURL         string                 // URL of the JavaScript bridge service
	currentRounds     map[string]*RoundData  // Track detailed data for rounds in current epoch
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
		bridgeURL:        "", // No default bridge URL - must be explicitly set
		currentRounds:    make(map[string]*RoundData),
	}
	
	// Create Genesis State immediately
	sga.createGenesisState()
	return sga
}

// SetEpochFinalizedCallback sets the callback function to be triggered when an epoch is finalized
func (sga *SubnetGraphAdapter) SetEpochFinalizedCallback(callback EpochFinalizedCallback) {
	sga.mu.Lock()
	defer sga.mu.Unlock()
	sga.epochCallback = callback
}

// SetBridgeURL sets the URL for the JavaScript bridge service
func (sga *SubnetGraphAdapter) SetBridgeURL(url string) {
	sga.mu.Lock()
	defer sga.mu.Unlock()
	sga.bridgeURL = url
}

// sendEpochToBridge sends epoch data to the JavaScript bridge via HTTP POST
func (sga *SubnetGraphAdapter) sendEpochToBridge(epochData *EpochData) error {
	// Prepare the payload for the bridge
	payload := map[string]interface{}{
		"epochNumber":    epochData.EpochNumber,
		"subnetId":       epochData.SubnetID,
		"completedRounds": epochData.CompletedRounds,
		"detailedRounds": epochData.DetailedRounds,
		"vlcClockState":  epochData.VLCClockState,
		"epochEventId":   epochData.EpochEventID,
		"parentRoundEventId": epochData.ParentRoundEventID,
		"timestamp":      time.Now().Unix(),
	}
	
	// Debug log the detailed rounds being sent
	fmt.Printf("🔍 DEBUG - Sending %d detailed rounds to bridge:\n", len(epochData.DetailedRounds))
	for i, round := range epochData.DetailedRounds {
		inputPreview := round.UserInput
		if len(inputPreview) > 40 {
			inputPreview = inputPreview[:40] + "..."
		}
		fmt.Printf("   Round %d: %s (Success: %t)\n", i+1, inputPreview, round.Success)
	}

	// Convert to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal epoch data: %v", err)
	}
	
	// Debug: Print summary of payload
	fmt.Printf("📤 Sending epoch data: %d detailed rounds, %d bytes\n", len(epochData.DetailedRounds), len(jsonPayload))

	// Create HTTP request
	req, err := http.NewRequest("POST", sga.bridgeURL+"/submit-epoch", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send epoch data to bridge: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bridge returned error status: %d", resp.StatusCode)
	}

	fmt.Printf("✅ Epoch %d data sent to bridge successfully (Status: %d)\n", epochData.EpochNumber, resp.StatusCode)
	return nil
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

	// Initialize or update round data
	if sga.currentRounds[requestID] == nil {
		// Calculate the round number within the current epoch (1, 2, or 3)
		// roundsInEpoch starts at 0, so we add 1 to get the current round number
		epochRoundNumber := sga.roundsInEpoch + 1
		sga.currentRounds[requestID] = &RoundData{
			RoundNumber:   epochRoundNumber,
			RequestID:     requestID,
			VLCClockState: make(map[int]int),
		}
		// Debug: Log round creation
		// fmt.Printf("🔍 Created round %d data for request %s\n", epochRoundNumber, requestID)
	}
	sga.currentRounds[requestID].UserInput = input
	sga.currentRounds[requestID].VLCClockState = vlcToMap(validatorClock)

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

	// Update round data with miner response
	if round := sga.currentRounds[requestID]; round != nil {
		if response.OutputType == OutputReady {
			round.MinerOutput = response.Output
			round.MinerOutputType = "output_ready"
		} else {
			round.InfoRequest = response.InfoRequest
			round.MinerOutputType = "info_request"
		}
		// Update VLC state
		for k, v := range vlcToMap(response.VLCClock) {
			round.VLCClockState[k] = v
		}
	}

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

	// Update round data with info response
	if round := sga.currentRounds[requestID]; round != nil {
		round.InfoResponse = additionalInfo
		// Update VLC state
		for k, v := range vlcToMap(validatorClock) {
			round.VLCClockState[k] = v
		}
	}

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

	// Complete round data with final results
	if round := sga.currentRounds[requestID]; round != nil {
		round.ConsensusResult = consensusResult
		round.UserFeedback = userFeedback
		round.UserAccept = userAccept
		round.FinalResult = finalResult
		round.Success = userAccept && finalResult == "OUTPUT DELIVERED TO USER"
		// Final VLC state update
		for k, v := range vlcToMap(validatorClock) {
			round.VLCClockState[k] = v
		}
	}

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
	
	// Trigger epoch finalized callback or HTTP bridge if configured
	if sga.epochCallback != nil || sga.bridgeURL != "" {
		epochData := &EpochData{
			EpochNumber:        sga.epochCount,
			SubnetID:           sga.SubnetID,
			CompletedRounds:    make([]string, len(sga.completedRounds)),
			DetailedRounds:     make([]RoundData, 0),
			VLCClockState:      make(map[int]int),
			EpochEventID:       epochEventID,
			ParentRoundEventID: parentRoundEventID,
		}
		
		// Copy completed rounds for this epoch (last 3 rounds)
		copy(epochData.CompletedRounds, sga.completedRounds)
		
		// IMPORTANT: Copy detailed round data BEFORE clearing currentRounds
		fmt.Printf("🔍 DEBUG - Current rounds in memory: %d\n", len(sga.currentRounds))
		for requestID, roundData := range sga.currentRounds {
			if roundData != nil {
				// Create a copy of the round data
				epochData.DetailedRounds = append(epochData.DetailedRounds, *roundData)
				inputPreview := roundData.UserInput
				if len(inputPreview) > 50 {
					inputPreview = inputPreview[:50] + "..."
				}
				fmt.Printf("📋 Including round %d data for request %s: %s (Success: %t)\n", roundData.RoundNumber, requestID, inputPreview, roundData.Success)
			}
		}
		fmt.Printf("🔍 DEBUG - Copied %d detailed rounds to epochData\n", len(epochData.DetailedRounds))
		
		// Copy VLC clock state
		for nodeID, value := range validatorClock.Values {
			epochData.VLCClockState[int(nodeID)] = int(value)
		}
		
		fmt.Printf("🚀 Epoch %d finalized - triggering mainnet submission\n", sga.epochCount)
		
		// Send epoch data to JavaScript bridge asynchronously
		go func() {
			// Try HTTP bridge first if URL is set
			if sga.bridgeURL != "" {
				fmt.Printf("📡 Sending Epoch %d data to JavaScript bridge...\n", epochData.EpochNumber)
				if err := sga.sendEpochToBridge(epochData); err != nil {
					fmt.Printf("❌ Failed to send epoch data to bridge: %v\n", err)
					if sga.epochCallback != nil {
						fmt.Printf("🔄 Falling back to callback method...\n")
						sga.epochCallback(sga.epochCount, sga.SubnetID, epochData)
					}
				} else {
					fmt.Printf("✅ Epoch %d submitted to mainnet via bridge!\n", epochData.EpochNumber)
				}
			} else if sga.epochCallback != nil {
				// Use callback method if no bridge URL
				sga.epochCallback(sga.epochCount, sga.SubnetID, epochData)
			}
		}()
	}
	
	// Reset completed rounds and current round data for next epoch
	sga.completedRounds = make([]string, 0)
	sga.currentRounds = make(map[string]*RoundData)
	
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