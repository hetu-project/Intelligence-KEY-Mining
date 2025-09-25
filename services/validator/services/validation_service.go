package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/crypto"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/plugins"
)

// ValidationService handles validator business logic
type ValidationService struct {
	config          *models.ValidatorConfig
	vlcService      *VLCService
	qualityAssessor plugins.QualityAssessor
	formatValidator plugins.FormatValidator
	processedEvents map[string]bool // Prevent replay attacks
}

// NewValidationService creates a new validation service
func NewValidationService(
	config *models.ValidatorConfig,
	vlcService *VLCService,
	qualityAssessor plugins.QualityAssessor,
	formatValidator plugins.FormatValidator,
) *ValidationService {
	return &ValidationService{
		config:          config,
		vlcService:      vlcService,
		qualityAssessor: qualityAssessor,
		formatValidator: formatValidator,
		processedEvents: make(map[string]bool),
	}
}

// ValidateMinerOutput validates miner output and generates vote
func (vs *ValidationService) ValidateMinerOutput(ctx context.Context, minerOutput *models.MinerOutput) (*models.ValidatorVote, error) {
	// 1. Prevent replay attack check
	if vs.processedEvents[minerOutput.EventID] {
		return nil, fmt.Errorf("event already processed: %s", minerOutput.EventID)
	}

	// 2. Validate time window (prevent too old or too new messages)
	now := time.Now()
	if minerOutput.Timestamp.Before(now.Add(-5*time.Minute)) ||
		minerOutput.Timestamp.After(now.Add(1*time.Minute)) {
		return nil, fmt.Errorf("invalid timestamp: %v", minerOutput.Timestamp)
	}

	// 3. Verify Miner signature
	if err := vs.verifyMinerSignature(minerOutput); err != nil {
		return nil, fmt.Errorf("invalid miner signature: %v", err)
	}

	// 4. Execute different validation logic based on validator role
	var vote *models.ValidatorVote
	var err error

	switch vs.config.Role {
	case models.UIValidatorRole:
		vote, err = vs.validateAsUIValidator(ctx, minerOutput)
	case models.FormatValidatorRole:
		vote, err = vs.validateAsFormatValidator(ctx, minerOutput)
	case models.SemanticValidatorRole:
		vote, err = vs.validateAsSemanticValidator(ctx, minerOutput)
	default:
		return nil, fmt.Errorf("unknown validator role: %s", vs.config.Role)
	}

	if err != nil {
		return nil, err
	}

	// 5. Sign vote
	if err := vs.signVote(vote); err != nil {
		return nil, fmt.Errorf("failed to sign vote: %v", err)
	}

	// 6. Record processed event
	vs.processedEvents[minerOutput.EventID] = true

	return vote, nil
}

// validateAsUIValidator performs UI validator specific validation (VLC focus)
func (vs *ValidationService) validateAsUIValidator(ctx context.Context, minerOutput *models.MinerOutput) (*models.ValidatorVote, error) {
	vote := &models.ValidatorVote{
		EventID:       minerOutput.EventID,
		ValidatorID:   vs.config.ID,
		ValidatorRole: vs.config.Role,
		Weight:        vs.config.Weight,
		Timestamp:     time.Now(),
	}

	// 1. VLC validation (core responsibility of UI validator)
	vlcValid := vs.vlcService.ValidateVLCSequence(minerOutput.VLCClock, 1) // Miner ID = 1
	if !vlcValid {
		vote.Vote = "reject"
		vote.Score = 0.0
		vote.Reason = "VLC validation failed: invalid sequence"
		vote.VLCState = vs.vlcService.GetCurrentClock() // Need to return VectorClock not map
		return vote, nil
	}

	// 2. Sync VLC state
	vs.vlcService.UpdateMinerClock(minerOutput.VLCClock)

	// 3. Basic format validation
	formatResult := vs.formatValidator.ValidateFormat(minerOutput)
	if !formatResult.Valid {
		vote.Vote = "reject"
		vote.Score = 0.2
		vote.Reason = fmt.Sprintf("Format validation failed: %s", formatResult.Reason)
		vote.VLCState = vs.vlcService.GetCurrentClock()
		return vote, nil
	}

	// 4. Quality assessment
	qualityResult := vs.qualityAssessor.AssessQuality(minerOutput)

	vote.Score = qualityResult.Score
	vote.VLCState = vs.vlcService.GetCurrentClock()

	if qualityResult.Accept {
		vote.Vote = "accept"
		vote.Reason = fmt.Sprintf("VLC valid, format valid, quality score: %.2f", qualityResult.Score)
	} else {
		vote.Vote = "reject"
		vote.Reason = fmt.Sprintf("Quality assessment failed: %.2f", qualityResult.Score)
	}

	return vote, nil
}

// validateAsFormatValidator performs format validation
func (vs *ValidationService) validateAsFormatValidator(ctx context.Context, minerOutput *models.MinerOutput) (*models.ValidatorVote, error) {
	vote := &models.ValidatorVote{
		EventID:       minerOutput.EventID,
		ValidatorID:   vs.config.ID,
		ValidatorRole: vs.config.Role,
		Weight:        vs.config.Weight,
		Timestamp:     time.Now(),
		VLCState:      vs.vlcService.GetCurrentClock(),
	}

	// 1. VLC format check (simplified)
	if minerOutput.VLCClock == nil || len(minerOutput.VLCClock.Values) == 0 {
		vote.Vote = "reject"
		vote.Score = 0.0
		vote.Reason = "Invalid VLC format"
		return vote, nil
	}

	// 2. Format validation
	formatResult := vs.formatValidator.ValidateFormat(minerOutput)
	if !formatResult.Valid {
		vote.Vote = "reject"
		vote.Score = 0.1
		vote.Reason = fmt.Sprintf("Format validation failed: %s", formatResult.Reason)
		return vote, nil
	}

	// 3. Proof signature verification (if exists)
	if minerOutput.Proof != nil && minerOutput.Proof.Signature != "" {
		if !vs.verifyProofSignature(minerOutput.Proof) {
			vote.Vote = "reject"
			vote.Score = 0.3
			vote.Reason = "Invalid proof signature"
			return vote, nil
		}
	}

	// 4. Basic quality assessment
	qualityResult := vs.qualityAssessor.AssessQuality(minerOutput)

	vote.Score = qualityResult.Score
	if qualityResult.Accept {
		vote.Vote = "accept"
		vote.Reason = "Format validation passed"
	} else {
		vote.Vote = "reject"
		vote.Reason = "Quality score too low"
	}

	return vote, nil
}

// validateAsSemanticValidator performs semantic validation
func (vs *ValidationService) validateAsSemanticValidator(ctx context.Context, minerOutput *models.MinerOutput) (*models.ValidatorVote, error) {
	vote := &models.ValidatorVote{
		EventID:       minerOutput.EventID,
		ValidatorID:   vs.config.ID,
		ValidatorRole: vs.config.Role,
		Weight:        vs.config.Weight,
		Timestamp:     time.Now(),
		VLCState:      vs.vlcService.GetCurrentClock(),
	}

	// 1. VLC format check
	if minerOutput.VLCClock == nil || len(minerOutput.VLCClock.Values) == 0 {
		vote.Vote = "reject"
		vote.Score = 0.0
		vote.Reason = "Invalid VLC format"
		return vote, nil
	}

	// 2. Business logic validation
	if err := vs.validateBusinessLogic(minerOutput); err != nil {
		vote.Vote = "reject"
		vote.Score = 0.2
		vote.Reason = fmt.Sprintf("Business logic validation failed: %v", err)
		return vote, nil
	}

	// 3. Deep quality assessment
	qualityResult := vs.qualityAssessor.AssessQuality(minerOutput)

	vote.Score = qualityResult.Score
	if qualityResult.Accept && qualityResult.Score > 0.6 {
		vote.Vote = "accept"
		vote.Reason = "Semantic validation passed"
	} else {
		vote.Vote = "reject"
		vote.Reason = "Semantic quality insufficient"
	}

	return vote, nil
}

// verifyMinerSignature verifies miner's signature
func (vs *ValidationService) verifyMinerSignature(minerOutput *models.MinerOutput) error {
	// Reconstruct signature data
	data := map[string]interface{}{
		"task_id":   minerOutput.TaskID,
		"miner_id":  minerOutput.MinerID,
		"event_id":  minerOutput.EventID,
		"vlc_clock": minerOutput.VLCClock,
		"timestamp": minerOutput.Timestamp.Unix(),
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Need to get Miner's public key for verification here
	// In actual implementation, need to get Miner public key from config or registry
	// Temporarily skip signature verification, return true
	// TODO: Implement proper public key retrieval and signature verification
	_ = dataBytes // Avoid unused variable warning
	return nil
}

// verifyProofSignature verifies proof signature from middle layer
func (vs *ValidationService) verifyProofSignature(proof *models.TaskProof) bool {
	// In actual implementation, need to verify middleware signature
	// Simplified processing here
	return proof.Signature != ""
}

// validateBusinessLogic validates business logic specific to task type
func (vs *ValidationService) validateBusinessLogic(minerOutput *models.MinerOutput) error {
	// Execute different business logic validation based on task type
	// For example: verify points calculation, subnet membership, etc.
	return nil
}

// signVote signs the validator vote
func (vs *ValidationService) signVote(vote *models.ValidatorVote) error {
	data := map[string]interface{}{
		"event_id":     vote.EventID,
		"validator_id": vote.ValidatorID,
		"vote":         vote.Vote,
		"score":        vote.Score,
		"timestamp":    vote.Timestamp.Unix(),
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	signature, err := crypto.SignData(vs.config.PrivateKey, dataBytes)
	if err != nil {
		return err
	}

	vote.Signature = signature
	return nil
}

// GetValidatorInfo returns validator information
func (vs *ValidationService) GetValidatorInfo() *models.ValidatorConfig {
	return vs.config
}
