package verifiers

import (
	"context"
	"fmt"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// GenericVerifier handles generic task verification with minimal validation
type GenericVerifier struct {
	*BaseVerifier
}

// NewGenericVerifier creates a new generic verifier
func NewGenericVerifier(taskType models.TaskType) *GenericVerifier {
	return &GenericVerifier{
		BaseVerifier: NewBaseVerifier(taskType),
	}
}

// ValidatePayload validates generic payload format with minimal requirements
func (gv *GenericVerifier) ValidatePayload(payload map[string]interface{}) error {
	// Check basic required fields that all tasks should have
	requiredFields := []string{"project_name", "description"}

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

// ValidateSync performs synchronous validation (auto-approve for generic tasks)
func (gv *GenericVerifier) ValidateSync(ctx context.Context, payload map[string]interface{}) (bool, *models.TaskProof, error) {
	if err := gv.ValidatePayload(payload); err != nil {
		return false, nil, err
	}

	// For generic tasks, auto-approve without external validation
	proof := &models.TaskProof{
		Provider: "generic-verifier-auto",
		Evidence: map[string]interface{}{
			"validation_type": "auto_approved",
			"verified":        true,
			"task_id":         payload["task_id"],
			"user_wallet":     payload["user_wallet"],
		},
	}

	return true, proof, nil
}

// ValidateAsync performs asynchronous validation (placeholder for future implementation)
func (gv *GenericVerifier) ValidateAsync(ctx context.Context, payload map[string]interface{}) (*models.TaskProof, error) {
	if err := gv.ValidatePayload(payload); err != nil {
		return nil, err
	}

	// For now, generic tasks are automatically considered valid
	// In the future, this could be extended with specific validation logic
	proof := &models.TaskProof{
		Provider: "generic-verifier",
		Evidence: map[string]interface{}{
			"validation_type": "generic",
			"auto_verified":   true,
			"task_id":         payload["task_id"],
			"user_wallet":     payload["user_wallet"],
		},
	}

	return proof, nil
}

// RegisterAsyncWatch registers an async watch for generic tasks (placeholder)
func (gv *GenericVerifier) RegisterAsyncWatch(ctx context.Context, payload map[string]interface{}) (string, error) {
	// For generic tasks, we don't need async watching
	// Return a dummy watch ID
	return "generic-watch-" + fmt.Sprintf("%v", payload["task_id"]), nil
}

// CheckAsyncStatus checks the status of an async watch for generic tasks
func (gv *GenericVerifier) CheckAsyncStatus(ctx context.Context, watchID string) (bool, *models.TaskProof, error) {
	// For generic tasks, always return completed with a basic proof
	proof := &models.TaskProof{
		Provider: "generic-verifier",
		Evidence: map[string]interface{}{
			"validation_type": "generic",
			"auto_verified":   true,
			"watch_id":        watchID,
		},
	}
	return true, proof, nil
}
