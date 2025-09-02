// Package subnet - Core Miner Implementation
//
// This file implements the CoreMiner, a generic AI agent that processes user tasks
// while maintaining Vector Logical Clock (VLC) consistency for Proof-of-Causal-Work.
// The miner uses pluggable task processors to enable different AI models and processing strategies.
package subnet

import (
	"sync"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/vlc"
)

// TaskProcessor defines the interface for pluggable AI task processing strategies.
// This abstraction enables the same core miner to work with different AI models,
// processing algorithms, or business logic while maintaining VLC consistency.
type TaskProcessor interface {
	// ProcessTask handles initial user input and determines response type.
	// Returns:
	//   - outputType: OutputReady (has solution) or NeedMoreInfo (needs clarification)
	//   - output: Generated solution (if OutputReady) or empty (if NeedMoreInfo)
	//   - infoRequest: Question for user (if NeedMoreInfo) or empty (if OutputReady)
	ProcessTask(input string, inputNumber int) (outputType MinerOutputType, output string, infoRequest string)
	
	// ProcessAdditionalInfo handles follow-up processing with user-provided context.
	// Called after ProcessTask returned NeedMoreInfo and user provided additional information.
	// Returns the final generated output incorporating both original and additional context.
	ProcessAdditionalInfo(originalInput string, additionalInfo string, inputNumber int) string
}

// CoreMiner represents a generic AI agent (miner) in the PoCW subnet architecture.
// It processes user tasks while maintaining causal consistency through Vector Logical Clocks.
// The miner's behavior is customizable through pluggable TaskProcessor implementations.
//
// VLC Clock Semantics:
//   - Each ProcessInput() increments clock (represents logical work)
//   - Each ProcessAdditionalInfo() increments clock (represents additional logical work)
//   - Clock values enable validators to verify causal ordering of operations
type CoreMiner struct {
	// Identity and network information
	ID       string // Unique miner identifier
	SubnetID string // Subnet this miner belongs to
	
	// VLC-based causal consistency
	VLCClock *vlc.Clock   // Vector clock tracking logical time of operations
	mu       sync.RWMutex // Protects concurrent access to miner state

	// Processing history and state
	processedInputs map[int]*MinerResponseMessage // Audit trail of processed tasks

	// Pluggable behavior strategy
	taskProcessor TaskProcessor // AI/processing logic implementation
}

// NewCoreMiner creates a new generic AI miner with empty state and configuration.
// The miner must be configured with a TaskProcessor before use via SetTaskProcessor().
//
// Parameters:
//   - id: Unique identifier for this miner (e.g., "miner-1")
//   - subnetID: Identifier of the subnet this miner joins
//
// Returns a miner with initialized VLC clock (starting at 0) and empty processing history.
func NewCoreMiner(id, subnetID string) *CoreMiner {
	return &CoreMiner{
		ID:              id,
		SubnetID:        subnetID,
		VLCClock:        vlc.New(), // Initialize VLC clock
		processedInputs: make(map[int]*MinerResponseMessage),
	}
}

// SetTaskProcessor sets the task processing strategy
func (m *CoreMiner) SetTaskProcessor(processor TaskProcessor) {
	m.taskProcessor = processor
}

// ProcessInput processes initial user input and determines the response type.
// This method represents the first logical operation in the PoCW protocol.
//
// Simplified VLC Behavior: 
//   - Miner ID = 1, Validator-1 ID = 2
//   - Only these two participants maintain VLC clocks
//   - Other validators (2-4) just vote without VLC tracking
//
// Process:
//   1. Increment VLC clock for miner (ID = 1)
//   2. Use pluggable TaskProcessor to analyze input
//   3. Generate response with either solution (OutputReady) or info request (NeedMoreInfo)
//   4. Store response in processing history
//
// Returns MinerResponseMessage that validators will evaluate for consensus.
func (m *CoreMiner) ProcessInput(input string, inputNumber int, requestID string) *MinerResponseMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment VLC clock for miner processing (miner ID = 1)
	m.VLCClock.Inc(1)

	response := &MinerResponseMessage{
		SubnetMessage: SubnetMessage{
			SubnetID:  m.SubnetID,
			RequestID: requestID,
			Type:      MinerResponseType,
			Sender:    m.ID,
			Timestamp: time.Now().Unix(),
		},
		VLCClock:    m.VLCClock,
		InputNumber: inputNumber,
	}

	// Use pluggable task processor
	if m.taskProcessor != nil {
		outputType, output, infoRequest := m.taskProcessor.ProcessTask(input, inputNumber)
		response.OutputType = outputType
		response.Output = output
		response.InfoRequest = infoRequest
	} else {
		// Default: process everything as ready
		response.OutputType = OutputReady
		response.Output = "Default processing completed"
	}

	// Store the response for tracking
	m.processedInputs[inputNumber] = response
	return response
}

// ProcessAdditionalInfo processes user-provided additional context to generate final output.
// This method represents a separate message and logical operation in the simplified VLC flow.
//
// Round-Based VLC: This is called after Validator-1 has incremented its clock to provide
// additional context, so miner processes this as the next logical operation in the round.
//
// Process:
//   1. Increment VLC clock for miner (ID = 1) - represents work of processing additional context
//   2. Use pluggable TaskProcessor to process original + additional context
//   3. Generate final response with OutputReady type
//   4. Update processing history with final response
//
// Called after ProcessInput() returned NeedMoreInfo and user provided clarification.
func (m *CoreMiner) ProcessAdditionalInfo(originalInput string, additionalInfo string, inputNumber int, requestID string) *MinerResponseMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment VLC clock for processing additional context (miner ID = 1)
	m.VLCClock.Inc(1)

	response := &MinerResponseMessage{
		SubnetMessage: SubnetMessage{
			SubnetID:  m.SubnetID,
			RequestID: requestID,
			Type:      MinerResponseType,
			Sender:    m.ID,
			Timestamp: time.Now().Unix(),
		},
		OutputType:  OutputReady,
		VLCClock:    m.VLCClock, // Use incremented clock
		InputNumber: inputNumber,
	}

	// Use pluggable task processor for additional info
	if m.taskProcessor != nil {
		response.Output = m.taskProcessor.ProcessAdditionalInfo(originalInput, additionalInfo, inputNumber)
	} else {
		// Default: simple concatenation
		response.Output = originalInput + " [Additional: " + additionalInfo + "]"
	}

	// Update stored response
	m.processedInputs[inputNumber] = response
	return response
}

// GetCurrentClock returns the current VLC clock value
func (m *CoreMiner) GetCurrentClock() *vlc.Clock {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.VLCClock.Copy()
}

// UpdateValidatorClock synchronizes miner's VLC with validator operations
// Called when the miner receives information about validator-1's VLC updates
// This maintains causal consistency between the two VLC participants
func (m *CoreMiner) UpdateValidatorClock(validatorClock *vlc.Clock) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Merge validator's VLC state into miner's clock for causal consistency
	m.VLCClock.Merge([]*vlc.Clock{validatorClock})
}

// GetProcessedInputs returns all processed inputs for debugging
func (m *CoreMiner) GetProcessedInputs() map[int]*MinerResponseMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[int]*MinerResponseMessage)
	for k, v := range m.processedInputs {
		result[k] = v
	}
	return result
}