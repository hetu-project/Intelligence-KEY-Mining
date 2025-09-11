package verifiers

import (
	"context"
	"fmt"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// TaskCreationVerifier handles task creation verification
type TaskCreationVerifier struct {
	*BaseVerifier
}

// NewTaskCreationVerifier creates a new task creation verifier
func NewTaskCreationVerifier() *TaskCreationVerifier {
	return &TaskCreationVerifier{
		BaseVerifier: NewBaseVerifier(models.TaskCreationTask),
	}
}

// ValidatePayload validates task creation payload format
func (tcv *TaskCreationVerifier) ValidatePayload(payload map[string]interface{}) error {
	// Check required fields
	requiredFields := []string{"project_name", "description", "twitter_username", "twitter_link", "tweet_id"}

	for _, field := range requiredFields {
		value, exists := payload[field]
		if !exists {
			return fmt.Errorf("%s is required", field)
		}

		if str, ok := value.(string); !ok || str == "" {
			return fmt.Errorf("%s must be a non-empty string", field)
		}
	}

	return nil
}

// ValidateSync performs synchronous task creation verification
// Task creation doesn't need verification, return success directly
func (tcv *TaskCreationVerifier) ValidateSync(ctx context.Context, payload map[string]interface{}) (bool, *models.TaskProof, error) {
	if err := tcv.ValidatePayload(payload); err != nil {
		return false, nil, err
	}

	// Build proof - task creation proof is the payload itself
	proof := &models.TaskProof{
		Provider:       "task-creation-internal",
		VerifiedAt:     time.Now(),
		Evidence:       payload,
		VerificationID: fmt.Sprintf("creation_%d", time.Now().Unix()),
		Signature:      "", // Internal verification doesn't need signature
	}

	return true, proof, nil
}

// RegisterAsyncWatch task creation doesn't need async monitoring
func (tcv *TaskCreationVerifier) RegisterAsyncWatch(ctx context.Context, payload map[string]interface{}) (string, error) {
	return "", fmt.Errorf("task creation does not support async watch")
}

// CheckAsyncStatus task creation doesn't need async status checking
func (tcv *TaskCreationVerifier) CheckAsyncStatus(ctx context.Context, watchID string) (bool, *models.TaskProof, error) {
	return false, nil, fmt.Errorf("task creation does not support async status check")
}
