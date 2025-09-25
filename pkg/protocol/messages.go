package protocol

import (
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/vlc"
)

// MessageType represents the type of message
type MessageType string

const (
	MinerOutputMessage     MessageType = "miner_output"
	ValidatorVoteMessage   MessageType = "validator_vote"
	ConsensusResultMessage MessageType = "consensus_result"
	HeartbeatMessage       MessageType = "heartbeat"
)

// BaseMessage represents the base structure for all messages
type BaseMessage struct {
	Type      MessageType `json:"type"`
	MessageID string      `json:"message_id"`
	Timestamp time.Time   `json:"timestamp"`
	Signature string      `json:"signature"`
}

// MinerOutputRequest represents miner output sent to validators
type MinerOutputRequest struct {
	BaseMessage

	// Core fields
	TaskID   string           `json:"task_id"`
	MinerID  string           `json:"miner_id"`
	EventID  string           `json:"event_id"`
	VLCClock *vlc.VectorClock `json:"vlc_clock"`

	// Task data
	Payload map[string]interface{} `json:"payload"`
	Proof   *TaskProof             `json:"proof"`

	// Network fields
	RequestID string `json:"request_id"` // For tracking requests
}

// TaskProof represents task verification proof
type TaskProof struct {
	Provider       string                 `json:"provider"`        // "twitter-middle-layer"
	VerifiedAt     time.Time              `json:"verified_at"`     // Verification time
	Evidence       map[string]interface{} `json:"evidence"`        // Evidence snapshot
	VerificationID string                 `json:"verification_id"` // Middleware verification ID
	Signature      string                 `json:"signature"`       // Middleware signature
}

// ValidatorVoteResponse represents validator's vote response
type ValidatorVoteResponse struct {
	BaseMessage

	// Vote fields
	EventID       string           `json:"event_id"`
	ValidatorID   string           `json:"validator_id"`
	ValidatorRole string           `json:"validator_role"` // "ui_validator", "format_validator", "semantic_validator"
	Vote          string           `json:"vote"`           // "accept" or "reject"
	Score         float64          `json:"score"`          // Quality score 0.0-1.0
	Weight        float64          `json:"weight"`         // Voting weight
	Reason        string           `json:"reason"`         // Voting reason
	VLCState      *vlc.VectorClock `json:"vlc_state"`      // Current VLC state

	// Network fields
	RequestID string `json:"request_id"` // Corresponding request ID
}

// ConsensusResult represents the final consensus result
type ConsensusResult struct {
	BaseMessage

	EventID          string                   `json:"event_id"`
	TaskID           string                   `json:"task_id"`
	Votes            []*ValidatorVoteResponse `json:"votes"`
	TotalWeight      float64                  `json:"total_weight"`
	AcceptWeight     float64                  `json:"accept_weight"`
	RejectWeight     float64                  `json:"reject_weight"`
	FinalDecision    string                   `json:"final_decision"` // "accepted" or "rejected"
	ConsensusReached bool                     `json:"consensus_reached"`
	AggregatorID     string                   `json:"aggregator_id"`
}

// HeartbeatRequest represents heartbeat message
type HeartbeatRequest struct {
	BaseMessage
	NodeID   string `json:"node_id"`
	NodeType string `json:"node_type"` // "miner", "validator", "aggregator"
	Status   string `json:"status"`    // "healthy", "degraded", "error"
}

// HeartbeatResponse represents heartbeat response
type HeartbeatResponse struct {
	BaseMessage
	NodeID string `json:"node_id"`
	Status string `json:"status"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	BaseMessage
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// API endpoints constants
const (
	// Validator endpoints
	ValidateEndpoint = "/api/v1/validate"
	VoteEndpoint     = "/api/v1/vote"
	HealthEndpoint   = "/api/v1/health"

	// Aggregator endpoints
	SubmitVoteEndpoint   = "/api/v1/aggregator/vote"
	GetConsensusEndpoint = "/api/v1/aggregator/consensus"

	// Miner endpoints
	TaskSubmitEndpoint = "/api/v1/tasks/submit"
	TaskStatusEndpoint = "/api/v1/tasks/status"
)

// HTTP Status codes for protocol
const (
	StatusSuccess            = 200
	StatusBadRequest         = 400
	StatusUnauthorized       = 401
	StatusNotFound           = 404
	StatusValidationFailed   = 422
	StatusInternalError      = 500
	StatusServiceUnavailable = 503
)

// Validation request/response for validator service
type ValidationRequest struct {
	MinerOutput *MinerOutputRequest `json:"miner_output" validate:"required"`
}

type ValidationResponse struct {
	Success bool                   `json:"success"`
	Vote    *ValidatorVoteResponse `json:"vote,omitempty"`
	Message string                 `json:"message"`
	Error   string                 `json:"error,omitempty"`
}

// Network configuration
type NetworkConfig struct {
	// Validator endpoints
	ValidatorEndpoints []ValidatorEndpoint `json:"validator_endpoints"`

	// Aggregator endpoint
	AggregatorEndpoint string `json:"aggregator_endpoint"`

	// Timeouts
	RequestTimeout  time.Duration `json:"request_timeout"`
	ResponseTimeout time.Duration `json:"response_timeout"`

	// Retry policy
	MaxRetries    int           `json:"max_retries"`
	RetryInterval time.Duration `json:"retry_interval"`

	// Security
	TLSEnabled bool   `json:"tls_enabled"`
	CertFile   string `json:"cert_file,omitempty"`
	KeyFile    string `json:"key_file,omitempty"`
}

type ValidatorEndpoint struct {
	ID       string  `json:"id"`       // validator-1, validator-2, etc.
	Role     string  `json:"role"`     // ui_validator, format_validator, semantic_validator
	URL      string  `json:"url"`      // http://validator-1:8080
	Weight   float64 `json:"weight"`   // 0.4, 0.2, 0.2, 0.2
	Priority int     `json:"priority"` // 1 (highest) to N (lowest)
}
