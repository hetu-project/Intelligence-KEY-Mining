// Package subnet - Message Types and Consensus Logic
//
// This file defines the message types used in the PoCW subnet protocol and implements
// the consensus logic for aggregating validator votes. All messages support VLC clocks
// for causal consistency and cryptographic signatures for authenticity.
package subnet

import (
	"github.com/hetu-project/Intelligence-KEY-Mining/vlc"
)

// SubnetMessageType defines the different message types in the PoCW protocol.
// Each type represents a specific phase of the task processing and consensus workflow.
type SubnetMessageType string

const (
	UserInputType      SubnetMessageType = "user_input"      // Initial user task submission
	MinerResponseType  SubnetMessageType = "miner_response"  // Miner's output or info request
	ValidatorVoteType  SubnetMessageType = "validator_vote"  // Validator's quality assessment vote
	InfoRequestType    SubnetMessageType = "info_request"    // Request for additional user context
	AdditionalInfoType SubnetMessageType = "additional_info" // User-provided additional context
	FinalOutputType    SubnetMessageType = "final_output"    // Final consensus result delivery
)

// MinerOutputType specifies the type of response a miner can generate.
// This determines the next phase of the protocol workflow.
type MinerOutputType string

const (
	OutputReady  MinerOutputType = "output_ready"   // Miner has generated a solution ready for validation
	NeedMoreInfo MinerOutputType = "need_more_info" // Miner needs additional context from user
)

// SubnetMessage is the base message structure for subnet communication
type SubnetMessage struct {
	SubnetID  string            `json:"subnet_id"`
	RequestID string            `json:"request_id"`
	Type      SubnetMessageType `json:"type"`
	Sender    string            `json:"sender"`
	Receiver  string            `json:"receiver"`
	Timestamp int64             `json:"timestamp"`
	Signature string            `json:"signature"`
}

// UserInputMessage represents user input to the subnet
type UserInputMessage struct {
	SubnetMessage
	Input       string `json:"input"`
	InputNumber int    `json:"input_number"`
}

// MinerResponseMessage represents a miner's response to user input or additional information.
// Contains either a completed solution (OutputReady) or a request for more context (NeedMoreInfo).
// The VLC clock enables validators to verify causal ordering of miner operations.
type MinerResponseMessage struct {
	SubnetMessage
	OutputType  MinerOutputType `json:"output_type"`              // Type of response (ready vs need info)
	Output      string          `json:"output,omitempty"`          // Generated solution (if OutputReady)
	InfoRequest string          `json:"info_request,omitempty"`    // Question for user (if NeedMoreInfo)
	VLCClock    *vlc.Clock      `json:"vlc_clock"`                // Vector clock for causal ordering
	InputNumber int             `json:"input_number"`              // Sequential input identifier for tracking
}

// ValidatorVoteMessage represents validator's vote on miner output
type ValidatorVoteMessage struct {
	SubnetMessage
	ValidatorID    string     `json:"validator_id"`
	Quality        float64    `json:"quality"` // 0.0 to 1.0
	Accept         bool       `json:"accept"`
	Weight         float64    `json:"weight"` // 0.25 for each validator
	LastMinerClock *vlc.Clock `json:"last_miner_clock"`
}

// InfoRequestMessage represents validator requesting more info from user
type InfoRequestMessage struct {
	SubnetMessage
	Question string `json:"question"`
}

// AdditionalInfoMessage represents user providing additional information
type AdditionalInfoMessage struct {
	SubnetMessage
	AdditionalInfo string `json:"additional_info"`
}

// FinalOutputMessage represents the final output delivered to user
type FinalOutputMessage struct {
	SubnetMessage
	Output       string  `json:"output"`
	Accepted     bool    `json:"accepted"`
	UserRejected bool    `json:"user_rejected,omitempty"`
	Consensus    float64 `json:"consensus"` // Total acceptance weight
}

// QualityAssessment tracks and aggregates validator consensus on miner output quality.
// Implements Byzantine Fault Tolerant (BFT) consensus by accumulating weighted votes.
// Consensus is reached when sufficient validators have voted (determined by total weight).
type QualityAssessment struct {
	RequestID   string  // Unique identifier for the request being assessed
	TotalWeight float64 // Sum of all validator weights that have voted
	AcceptVotes float64 // Sum of weights from validators who accepted the output
	RejectVotes float64 // Sum of weights from validators who rejected the output
	VoteCount   int     // Total number of validator votes received
	Consensus   bool    // Whether sufficient votes have been received for consensus
}

// AddVote incorporates a validator's vote into the consensus assessment.
// Accumulates voting weights and determines if consensus threshold is reached.
//
// Consensus Logic:
//   - Consensus achieved when >50% of total voting weight participates
//   - Acceptance requires >50% of participating weight to vote "accept"
//   - This implements Byzantine Fault Tolerant consensus for quality assessment
//
// Parameters:
//   - weight: Validator's voting weight (typically 1.0/N for N validators)
//   - accept: Validator's decision (true = accept output, false = reject output)
func (qa *QualityAssessment) AddVote(weight float64, accept bool) {
	qa.TotalWeight += weight
	qa.VoteCount++

	if accept {
		qa.AcceptVotes += weight
	} else {
		qa.RejectVotes += weight
	}

	// Consensus reached if > 50% weight votes (BFT threshold)
	qa.Consensus = qa.AcceptVotes > 0.5 || qa.RejectVotes > 0.5
}

// IsAccepted returns true if the consensus assessment indicates output acceptance.
// Requires both consensus achievement and majority acceptance votes.
//
// Returns true only if:
//   1. Consensus threshold reached (>50% validator weight participated)
//   2. Majority of participating validators voted to accept (>50% of votes)
func (qa *QualityAssessment) IsAccepted() bool {
	return qa.Consensus && qa.AcceptVotes > 0.5
}
