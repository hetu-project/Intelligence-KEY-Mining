package plugins

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/models"
)

// FormatValidator defines the interface for format validation
type FormatValidator interface {
	ValidateFormat(minerOutput *models.MinerOutput) *FormatResult
}

// FormatResult represents the result of format validation
type FormatResult struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
}

// TwitterFormatValidator implements format validation for Twitter tasks
type TwitterFormatValidator struct{}

// NewTwitterFormatValidator creates a new Twitter format validator
func NewTwitterFormatValidator() *TwitterFormatValidator {
	return &TwitterFormatValidator{}
}

// ValidateFormat validates the format of miner output
func (tfv *TwitterFormatValidator) ValidateFormat(minerOutput *models.MinerOutput) *FormatResult {
	// 1. Check basic fields
	if minerOutput.EventID == "" {
		return &FormatResult{
			Valid:  false,
			Reason: "missing event_id",
		}
	}

	if minerOutput.TaskID == "" {
		return &FormatResult{
			Valid:  false,
			Reason: "missing task_id",
		}
	}

	if minerOutput.TaskType == "" {
		return &FormatResult{
			Valid:  false,
			Reason: "missing task_type",
		}
	}

	// 2. Check task type
	if minerOutput.TaskType != "twitter_retweet" {
		return &FormatResult{
			Valid:  false,
			Reason: "unsupported task_type: " + minerOutput.TaskType,
		}
	}

	// 3. Check payload format
	if err := tfv.validatePayload(minerOutput.Payload); err != nil {
		return &FormatResult{
			Valid:  false,
			Reason: "invalid payload: " + err.Error(),
		}
	}

	// 4. Check proof format
	if minerOutput.Proof != nil {
		if err := tfv.validateProof(minerOutput.Proof); err != nil {
			return &FormatResult{
				Valid:  false,
				Reason: "invalid proof: " + err.Error(),
			}
		}
	}

	// 5. Check VLC format
	if minerOutput.VLCClock != nil {
		if err := tfv.validateVLCClock(minerOutput.VLCClock); err != nil {
			return &FormatResult{
				Valid:  false,
				Reason: "invalid vlc_clock: " + err.Error(),
			}
		}
	}

	// 6. Check timestamp
	if minerOutput.Timestamp.IsZero() {
		return &FormatResult{
			Valid:  false,
			Reason: "missing timestamp",
		}
	}

	// Check if timestamp is within reasonable range
	now := time.Now()
	if minerOutput.Timestamp.Before(now.Add(-1*time.Hour)) ||
		minerOutput.Timestamp.After(now.Add(5*time.Minute)) {
		return &FormatResult{
			Valid:  false,
			Reason: "timestamp out of valid range",
		}
	}

	return &FormatResult{
		Valid:  true,
		Reason: "format validation passed",
	}
}

// validatePayload validates the payload format
func (tfv *TwitterFormatValidator) validatePayload(payload map[string]interface{}) error {
	if payload == nil {
		return fmt.Errorf("payload cannot be nil")
	}

	// Check required fields
	tweetID, hasTweetID := payload["tweet_id"]
	if !hasTweetID {
		return fmt.Errorf("missing tweet_id")
	}

	twitterID, hasTwitterID := payload["twitter_id"]
	if !hasTwitterID {
		return fmt.Errorf("missing twitter_id")
	}

	// Check field types
	if _, ok := tweetID.(string); !ok {
		return fmt.Errorf("tweet_id must be string")
	}

	if _, ok := twitterID.(string); !ok {
		return fmt.Errorf("twitter_id must be string")
	}

	return nil
}

// validateProof validates the proof format
func (tfv *TwitterFormatValidator) validateProof(proof *models.TaskProof) error {
	// Extract required fields from Evidence
	tweetID, ok := proof.Evidence["tweet_id"].(string)
	if !ok || tweetID == "" {
		return fmt.Errorf("missing tweet_id in proof evidence")
	}

	twitterID, ok := proof.Evidence["twitter_id"].(string)
	if !ok || twitterID == "" {
		return fmt.Errorf("missing twitter_id in proof evidence")
	}

	// Check timestamp
	if proof.VerifiedAt.IsZero() {
		return fmt.Errorf("missing verification timestamp in proof")
	}

	// Validate signature format
	if proof.Signature != "" {
		if len(proof.Signature) < 10 {
			return fmt.Errorf("invalid signature format")
		}
	}

	// Validate evidence format
	if proof.Evidence != nil {
		if _, err := json.Marshal(proof.Evidence); err != nil {
			return fmt.Errorf("invalid evidence format: %v", err)
		}
	}

	return nil
}

// validateVLCClock validates the VLC clock format
func (tfv *TwitterFormatValidator) validateVLCClock(vlcClock interface{}) error {
	// Here should validate based on actual VLC structure
	if vlcClock == nil {
		return fmt.Errorf("vlc_clock cannot be nil")
	}

	// Try serialization to validate format
	if _, err := json.Marshal(vlcClock); err != nil {
		return fmt.Errorf("invalid vlc_clock format: %v", err)
	}

	return nil
}

// DefaultFormatValidator provides a default implementation
type DefaultFormatValidator struct{}

// NewDefaultFormatValidator creates a default format validator
func NewDefaultFormatValidator() *DefaultFormatValidator {
	return &DefaultFormatValidator{}
}

// ValidateFormat provides basic format validation
func (dfv *DefaultFormatValidator) ValidateFormat(minerOutput *models.MinerOutput) *FormatResult {
	// Basic format check
	if minerOutput.EventID == "" {
		return &FormatResult{
			Valid:  false,
			Reason: "missing event_id",
		}
	}

	if minerOutput.TaskID == "" {
		return &FormatResult{
			Valid:  false,
			Reason: "missing task_id",
		}
	}

	if minerOutput.TaskType == "" {
		return &FormatResult{
			Valid:  false,
			Reason: "missing task_type",
		}
	}

	return &FormatResult{
		Valid:  true,
		Reason: "basic format validation passed",
	}
}
