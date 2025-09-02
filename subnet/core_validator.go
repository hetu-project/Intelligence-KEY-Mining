// Package subnet implements the core Proof-of-Causal-Work (PoCW) subnet architecture.
// This package provides generic, reusable components for building validator-miner networks
// with Vector Logical Clock (VLC) based consensus and pluggable quality assessment.
package subnet

import (
	"fmt"
	"sync"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/vlc"
)

// ValidatorRole defines the specific role a validator plays in the subnet.
// Different roles enable specialized behavior and responsibilities.
type ValidatorRole int

const (
	// UserInterfaceValidator handles user communication, info requests, and user feedback simulation.
	// Typically assigned to one validator per subnet to manage user interactions.
	UserInterfaceValidator ValidatorRole = iota
	
	// ConsensusValidator focuses solely on quality assessment and voting.
	// Multiple consensus validators provide distributed quality evaluation.
	ConsensusValidator
)

// QualityAssessor defines the interface for pluggable quality assessment strategies.
// Implementations can provide domain-specific logic for evaluating miner output quality.
// This enables the same core validator to work with different quality metrics.
type QualityAssessor interface {
	// AssessQuality evaluates a miner's response and returns quality score (0.0-1.0) and acceptance decision.
	// The quality score represents confidence in the output, while accept determines voting behavior.
	AssessQuality(response *MinerResponseMessage) (quality float64, accept bool)
}

// UserInteractionHandler defines the interface for pluggable user interaction simulation.
// This abstraction allows different user behavior patterns for testing and demo scenarios.
type UserInteractionHandler interface {
	// SimulateUserInteraction models user feedback on miner output.
	// Returns user acceptance decision and textual feedback.
	SimulateUserInteraction(inputNumber int, output string) (accept bool, feedback string)
}

// CoreValidator represents a generic validator node in the PoCW subnet architecture.
// It provides VLC-based sequence validation, pluggable quality assessment, and consensus voting.
// The validator tracks miner state using Vector Logical Clocks to ensure causal consistency.
type CoreValidator struct {
	// Identity and network information
	ID       string        // Unique validator identifier
	SubnetID string        // Subnet this validator belongs to
	Role     ValidatorRole // Validator's specific role in the subnet
	Weight   float64       // Voting weight in consensus (e.g., 0.25 for 1/4 validators)
	
	// VLC-based state tracking
	MinerClock *vlc.Clock // Vector clock tracking miner's causal state
	mu         sync.RWMutex // Protects concurrent access to validator state

	// Consensus and quality assessment
	assessments map[string]*QualityAssessment // Per-request quality tracking

	// Pluggable behavior strategies
	qualityAssessor        QualityAssessor        // Strategy for evaluating output quality
	userInteractionHandler UserInteractionHandler // Strategy for simulating user behavior
}

// NewCoreValidator creates a new generic validator instance with specified parameters.
// The validator is initialized with an empty VLC clock and no pluggable strategies.
// Use SetQualityAssessor() and SetUserInteractionHandler() to configure behavior.
//
// Parameters:
//   - id: Unique identifier for this validator
//   - subnetID: Identifier of the subnet this validator joins
//   - role: Validator's role (UserInterfaceValidator or ConsensusValidator)
//   - weight: Voting weight in consensus decisions (typically 1.0/N for N validators)
func NewCoreValidator(id, subnetID string, role ValidatorRole, weight float64) *CoreValidator {
	return &CoreValidator{
		ID:          id,
		SubnetID:    subnetID,
		Role:        role,
		Weight:      weight,
		MinerClock:  vlc.New(), // Initialize VLC clock
		assessments: make(map[string]*QualityAssessment),
	}
}

// SetQualityAssessor sets the quality assessment strategy
func (v *CoreValidator) SetQualityAssessor(assessor QualityAssessor) {
	v.qualityAssessor = assessor
}

// SetUserInteractionHandler sets the user interaction strategy
func (v *CoreValidator) SetUserInteractionHandler(handler UserInteractionHandler) {
	v.userInteractionHandler = handler
}

// ValidateSequence validates the causal ordering using Vector Logical Clocks.
// In the simplified round-based system, only Miner (ID=1) and Validator-1 (ID=2) 
// participate in VLC tracking.
//
// VLC Validation Rules:
//   - Bootstrap: Accept first message from any participant
//   - Increment: Accept +1 increment for the sending participant
//   - Cross-tracking: Validate causal consistency between both participants
//
// Returns true if the clock represents valid causal progression.
func (v *CoreValidator) ValidateSequence(incomingClock *vlc.Clock, senderID uint64) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Check if this sender is bootstrapped in our tracking
	_, exists := v.MinerClock.Values[senderID]
	if !exists {
		// First message from this sender - bootstrap and accept
		v.MinerClock.Merge([]*vlc.Clock{incomingClock})
		fmt.Printf("Validator %s: Bootstrapped %s clock - %v\n", v.ID, getParticipantName(senderID), incomingClock.Values)
		return true
	}

	// Validate +1 increment for the sender
	if v.MinerClock.IsPlusOneIncrement(incomingClock, senderID) {
		v.MinerClock.Merge([]*vlc.Clock{incomingClock})
		fmt.Printf("Validator %s: VLC sequence validated (+1) for %s - %v\n", v.ID, getParticipantName(senderID), incomingClock.Values)
		return true
	}

	fmt.Printf("Validator %s: VLC sequence error for %s - expected +1 from %v, got %v\n",
		v.ID, getParticipantName(senderID), v.MinerClock.Values, incomingClock.Values)
	return false
}

// VoteOnOutput evaluates a miner's response and generates a consensus vote.
// This method focuses purely on quality assessment - VLC validation should be done separately.
//
// Process:
//   1. Create/update quality assessment for this request  
//   2. Use pluggable quality assessor to evaluate output
//   3. Generate signed vote message with quality score and acceptance decision
//
// Note: VLC validation is performed separately as it's a local verification,
// while quality voting requires distributed consensus.
func (v *CoreValidator) VoteOnOutput(response *MinerResponseMessage) *ValidatorVoteMessage {

	v.mu.Lock()
	defer v.mu.Unlock()

	// Ensure assessment exists for this request
	if _, exists := v.assessments[response.RequestID]; !exists {
		v.assessments[response.RequestID] = &QualityAssessment{
			RequestID: response.RequestID,
		}
	}

	vote := &ValidatorVoteMessage{
		SubnetMessage: SubnetMessage{
			SubnetID:  v.SubnetID,
			RequestID: response.RequestID,
			Type:      ValidatorVoteType,
			Sender:    v.ID,
			Timestamp: time.Now().Unix(),
		},
		ValidatorID:    v.ID,
		Weight:         v.Weight,
		LastMinerClock: v.MinerClock.Copy(), // Include current VLC state for audit trail
	}

	// Use pluggable quality assessment
	var quality float64
	var accept bool
	if v.qualityAssessor != nil {
		quality, accept = v.qualityAssessor.AssessQuality(response)
	} else {
		// Default: accept everything with medium quality
		quality, accept = 0.75, true
	}

	vote.Quality = quality
	vote.Accept = accept

	// Add vote to assessment
	assessment := v.assessments[response.RequestID]
	assessment.AddVote(v.Weight, accept)

	fmt.Printf("Validator %s: Voted on Request %s - Accept: %t, Quality: %.2f\n",
		v.ID, response.RequestID, accept, quality)

	return vote
}

// RequestMoreInfo creates an information request message for user interaction.
// Only UserInterfaceValidator role can request additional information from users.
// This implements the interactive aspect of the PoCW protocol where miners can
// request clarification to improve their output quality.
//
// Returns nil if called on non-UI validator, otherwise returns info request message.
func (v *CoreValidator) RequestMoreInfo(requestID, question string) *InfoRequestMessage {
	if v.Role != UserInterfaceValidator {
		return nil // Only UI validator can request more info
	}

	return &InfoRequestMessage{
		SubnetMessage: SubnetMessage{
			SubnetID:  v.SubnetID,
			RequestID: requestID,
			Type:      InfoRequestType,
			Sender:    v.ID,
			Timestamp: time.Now().Unix(),
		},
		Question: question,
	}
}

// GetAssessment returns the current quality assessment for a request
func (v *CoreValidator) GetAssessment(requestID string) *QualityAssessment {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if assessment, exists := v.assessments[requestID]; exists {
		// Return a copy to avoid race conditions
		return &QualityAssessment{
			RequestID:   assessment.RequestID,
			TotalWeight: assessment.TotalWeight,
			AcceptVotes: assessment.AcceptVotes,
			RejectVotes: assessment.RejectVotes,
			VoteCount:   assessment.VoteCount,
			Consensus:   assessment.Consensus,
		}
	}
	return nil
}

// GetLastMinerClock returns the last validated miner clock
func (v *CoreValidator) GetLastMinerClock() *vlc.Clock {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.MinerClock.Copy()
}

// UpdateMinerClock synchronizes validator's VLC with miner operations
// Called when the validator receives miner responses to maintain causal consistency
func (v *CoreValidator) UpdateMinerClock(minerClock *vlc.Clock) {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	// Merge miner's VLC state into validator's clock for causal consistency
	v.MinerClock.Merge([]*vlc.Clock{minerClock})
}

// IncrementValidatorClock increments validator's own VLC for validator operations
// Called when validator performs round orchestration operations (user input, final output, etc.)
func (v *CoreValidator) IncrementValidatorClock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	const validatorID uint64 = 2 // Validator-1 ID in round-based system
	v.MinerClock.Inc(validatorID)
	fmt.Printf("Validator %s: Incremented VLC for validator operation - %v\n", v.ID, v.MinerClock.Values)
}

// SimulateUserInteraction uses pluggable user interaction logic
func (v *CoreValidator) SimulateUserInteraction(inputNumber int, output string) (bool, string) {
	if v.userInteractionHandler != nil {
		return v.userInteractionHandler.SimulateUserInteraction(inputNumber, output)
	}
	// Default: accept everything
	return true, "This looks good, thank you!"
}

// getParticipantName returns human-readable name for VLC participant IDs
func getParticipantName(id uint64) string {
	switch id {
	case 1:
		return "Miner"
	case 2:
		return "Validator-1"
	default:
		return fmt.Sprintf("Participant-%d", id)
	}
}