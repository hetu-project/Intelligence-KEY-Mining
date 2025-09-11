package models

import (
	"crypto/ecdsa"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/vlc"
)

// ValidatorRole defines the role of a validator
type ValidatorRole string

const (
	UIValidatorRole       ValidatorRole = "ui_validator"       // UI validator, responsible for VLC sync
	FormatValidatorRole   ValidatorRole = "format_validator"   // Format validator
	SemanticValidatorRole ValidatorRole = "semantic_validator" // Semantic validator
)

// ValidatorConfig represents validator configuration
type ValidatorConfig struct {
	ID         string            `json:"id"`         // Validator ID
	Role       ValidatorRole     `json:"role"`       // Validator role
	Weight     float64           `json:"weight"`     // Voting weight
	PrivateKey *ecdsa.PrivateKey `json:"-"`          // Private key (not serialized)
	PublicKey  string            `json:"public_key"` // Public key (hex encoded)
	Endpoints  []string          `json:"endpoints"`  // Other validator endpoints
}

// MinerOutput represents miner output received for validation
type MinerOutput struct {
	TaskID    string                 `json:"task_id"`
	TaskType  string                 `json:"task_type"`
	MinerID   string                 `json:"miner_id"`
	EventID   string                 `json:"event_id"`
	VLCClock  *vlc.VectorClock       `json:"vlc_clock"`
	Payload   map[string]interface{} `json:"payload"`
	Proof     *TaskProof             `json:"proof"`
	Signature string                 `json:"signature"`
	Timestamp time.Time              `json:"timestamp"`
}

// TaskProof represents task verification proof
type TaskProof struct {
	Provider       string                 `json:"provider"`
	VerifiedAt     time.Time              `json:"verified_at"`
	Evidence       map[string]interface{} `json:"evidence"`
	VerificationID string                 `json:"verification_id"`
	Signature      string                 `json:"signature"`
}

// ValidatorVote represents a validator's vote on miner output
type ValidatorVote struct {
	EventID       string           `json:"event_id"`
	ValidatorID   string           `json:"validator_id"`
	ValidatorRole ValidatorRole    `json:"validator_role"`
	Vote          string           `json:"vote"`      // "accept" or "reject"
	Score         float64          `json:"score"`     // Quality score 0.0-1.0
	Weight        float64          `json:"weight"`    // Voting weight
	Reason        string           `json:"reason"`    // Voting reason
	VLCState      *vlc.VectorClock `json:"vlc_state"` // Current VLC state
	Signature     string           `json:"signature"` // Validator signature
	Timestamp     time.Time        `json:"timestamp"`
}

// ValidationRequest represents validation request
type ValidationRequest struct {
	MinerOutput *MinerOutput `json:"miner_output" validate:"required"`
}

// ValidationResponse represents validation response
type ValidationResponse struct {
	Success bool           `json:"success"`
	Vote    *ValidatorVote `json:"vote,omitempty"`
	Message string         `json:"message"`
}

// VLCValidationResult represents VLC validation result
type VLCValidationResult struct {
	Valid       bool             `json:"valid"`
	Reason      string           `json:"reason"`
	ExpectedVLC *vlc.VectorClock `json:"expected_vlc,omitempty"`
	ActualVLC   *vlc.VectorClock `json:"actual_vlc,omitempty"`
}

// QualityAssessmentResult represents quality assessment result
type QualityAssessmentResult struct {
	Score      float64  `json:"score"`      // 0.0-1.0
	Accept     bool     `json:"accept"`     // Whether to accept
	Confidence float64  `json:"confidence"` // Confidence level
	Reasons    []string `json:"reasons"`    // Assessment reasons
}
