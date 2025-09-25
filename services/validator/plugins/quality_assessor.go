package plugins

import (
	"encoding/json"
	"fmt"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/models"
)

// QualityAssessor defines the interface for quality assessment
type QualityAssessor interface {
	AssessQuality(minerOutput *models.MinerOutput) *QualityResult
}

// QualityResult represents the result of quality assessment
type QualityResult struct {
	Accept bool    `json:"accept"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// TwitterQualityAssessor implements quality assessment for Twitter tasks
type TwitterQualityAssessor struct {
	role models.ValidatorRole
}

// NewTwitterQualityAssessor creates a new Twitter quality assessor
func NewTwitterQualityAssessor(role models.ValidatorRole) *TwitterQualityAssessor {
	return &TwitterQualityAssessor{
		role: role,
	}
}

// AssessQuality assesses the quality of a miner output
func (tqa *TwitterQualityAssessor) AssessQuality(minerOutput *models.MinerOutput) *QualityResult {
	switch tqa.role {
	case models.UIValidatorRole:
		return tqa.assessUIValidator(minerOutput)
	case models.FormatValidatorRole:
		return tqa.assessFormatValidator(minerOutput)
	case models.SemanticValidatorRole:
		return tqa.assessSemanticValidator(minerOutput)
	default:
		return &QualityResult{
			Accept: false,
			Score:  0.0,
			Reason: "Unknown validator role",
		}
	}
}

// assessUIValidator performs UI validator specific quality assessment
func (tqa *TwitterQualityAssessor) assessUIValidator(minerOutput *models.MinerOutput) *QualityResult {
	score := 0.8 // UI validator mainly focuses on VLC, quality assessment has lower weight

	// Basic checks
	if minerOutput.Proof == nil {
		return &QualityResult{
			Accept: false,
			Score:  0.0,
			Reason: "Missing proof",
		}
	}

	// Check basic fields of proof
	tweetID, hasTweetID := minerOutput.Proof.Evidence["tweet_id"].(string)
	twitterID, hasTwitterID := minerOutput.Proof.Evidence["twitter_id"].(string)
	if !hasTweetID || tweetID == "" || !hasTwitterID || twitterID == "" {
		score -= 0.3
	}

	// Check timestamp reasonableness
	if minerOutput.Proof.VerifiedAt.IsZero() {
		score -= 0.2
	}

	return &QualityResult{
		Accept: score > 0.5,
		Score:  score,
		Reason: fmt.Sprintf("UI validation score: %.2f", score),
	}
}

// assessFormatValidator performs format validator specific quality assessment
func (tqa *TwitterQualityAssessor) assessFormatValidator(minerOutput *models.MinerOutput) *QualityResult {
	score := 1.0

	// Check task type
	if minerOutput.TaskType != "twitter_retweet" {
		return &QualityResult{
			Accept: false,
			Score:  0.0,
			Reason: "Unsupported task type for Twitter format validation",
		}
	}

	// Check proof format
	if minerOutput.Proof == nil {
		return &QualityResult{
			Accept: false,
			Score:  0.0,
			Reason: "Missing proof",
		}
	}

	// Check Twitter ID format
	twitterID, hasTwitterID := minerOutput.Proof.Evidence["twitter_id"].(string)
	if !hasTwitterID || !isValidTwitterID(twitterID) {
		score -= 0.4
	}

	// Check Tweet ID format
	tweetID, hasTweetID := minerOutput.Proof.Evidence["tweet_id"].(string)
	if !hasTweetID || !isValidTweetID(tweetID) {
		score -= 0.4
	}

	// Check signature format
	if minerOutput.Proof.Signature == "" {
		score -= 0.2
	}

	reason := fmt.Sprintf("Format validation score: %.2f", score)
	if score < 0.6 {
		reason += " (format issues detected)"
	}

	return &QualityResult{
		Accept: score >= 0.6,
		Score:  score,
		Reason: reason,
	}
}

// assessSemanticValidator performs semantic validator specific quality assessment
func (tqa *TwitterQualityAssessor) assessSemanticValidator(minerOutput *models.MinerOutput) *QualityResult {
	score := 1.0

	// Semantic level deep analysis
	if minerOutput.Proof == nil {
		return &QualityResult{
			Accept: false,
			Score:  0.0,
			Reason: "Missing proof for semantic analysis",
		}
	}

	// Analyze semantic reasonableness of task payload
	semanticScore := tqa.analyzeTaskSemantics(minerOutput)
	score *= semanticScore

	// Analyze semantic consistency of proof
	proofScore := tqa.analyzeProofSemantics(minerOutput)
	score *= proofScore

	reason := fmt.Sprintf("Semantic validation score: %.2f", score)
	if score < 0.6 {
		reason += " (semantic issues detected)"
	}

	return &QualityResult{
		Accept: score >= 0.6,
		Score:  score,
		Reason: reason,
	}
}

// analyzeTaskSemantics analyzes the semantic consistency of task
func (tqa *TwitterQualityAssessor) analyzeTaskSemantics(minerOutput *models.MinerOutput) float64 {
	score := 1.0

	// Check semantic structure of payload
	if minerOutput.Payload == nil {
		return 0.0
	}

	// Check semantic reasonableness of required fields
	tweetID, hasTweetID := minerOutput.Payload["tweet_id"]
	twitterID, hasTwitterID := minerOutput.Payload["twitter_id"]

	if !hasTweetID || !hasTwitterID {
		score -= 0.5
	}

	// Check semantic reasonableness of field values
	if hasTweetID && tweetID != "" {
		if !isValidTweetID(fmt.Sprintf("%v", tweetID)) {
			score -= 0.3
		}
	}

	if hasTwitterID && twitterID != "" {
		if !isValidTwitterID(fmt.Sprintf("%v", twitterID)) {
			score -= 0.3
		}
	}

	return score
}

// analyzeProofSemantics analyzes the semantic consistency of proof
func (tqa *TwitterQualityAssessor) analyzeProofSemantics(minerOutput *models.MinerOutput) float64 {
	score := 1.0

	proof := minerOutput.Proof

	// Check consistency between proof and payload
	if payloadTweetID, exists := minerOutput.Payload["tweet_id"]; exists {
		if proofTweetID, ok := proof.Evidence["tweet_id"].(string); ok {
			if fmt.Sprintf("%v", payloadTweetID) != proofTweetID {
				score -= 0.4
			}
		}
	}

	if payloadTwitterID, exists := minerOutput.Payload["twitter_id"]; exists {
		if proofTwitterID, ok := proof.Evidence["twitter_id"].(string); ok {
			if fmt.Sprintf("%v", payloadTwitterID) != proofTwitterID {
				score -= 0.4
			}
		}
	}

	// Check reasonableness of evidence
	if proof.Evidence != nil {
		if evidenceData, err := json.Marshal(proof.Evidence); err == nil {
			if len(evidenceData) < 10 { // Too simple verification data
				score -= 0.2
			}
		}
	}

	return score
}

// isValidTwitterID validates Twitter ID format
func isValidTwitterID(twitterID string) bool {
	if len(twitterID) < 3 || len(twitterID) > 15 {
		return false
	}

	// Twitter ID can only contain letters, numbers and underscores
	for _, char := range twitterID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}

	return true
}

// isValidTweetID validates Tweet ID format
func isValidTweetID(tweetID string) bool {
	if len(tweetID) < 10 || len(tweetID) > 25 {
		return false
	}

	// Tweet ID should be pure numbers
	for _, char := range tweetID {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// DefaultQualityAssessor provides a default implementation
type DefaultQualityAssessor struct {
	role models.ValidatorRole
}

// NewDefaultQualityAssessor creates a default quality assessor
func NewDefaultQualityAssessor(role models.ValidatorRole) *DefaultQualityAssessor {
	return &DefaultQualityAssessor{
		role: role,
	}
}

// AssessQuality provides a basic quality assessment
func (dqa *DefaultQualityAssessor) AssessQuality(minerOutput *models.MinerOutput) *QualityResult {
	score := 0.8 // Default score

	// Basic checks
	if minerOutput.TaskType == "" {
		score -= 0.3
	}

	if minerOutput.Proof == nil {
		score -= 0.5
	}

	return &QualityResult{
		Accept: score > 0.5,
		Score:  score,
		Reason: fmt.Sprintf("Default quality assessment: %.2f", score),
	}
}
