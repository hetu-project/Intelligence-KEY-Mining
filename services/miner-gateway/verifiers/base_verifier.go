package verifiers

import (
	"context"
	"errors"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// TaskVerifier defines the interface for pluggable task verification strategies
type TaskVerifier interface {
	// ValidateSync performs synchronous validation check
	ValidateSync(ctx context.Context, payload map[string]interface{}) (bool, *models.TaskProof, error)

	// RegisterAsyncWatch registers an async watch for task completion
	RegisterAsyncWatch(ctx context.Context, payload map[string]interface{}) (watchID string, err error)

	// CheckAsyncStatus checks the status of an async watch
	CheckAsyncStatus(ctx context.Context, watchID string) (completed bool, proof *models.TaskProof, err error)

	// GetTaskType returns the task type this verifier handles
	GetTaskType() models.TaskType

	// ValidatePayload validates the payload format for this task type
	ValidatePayload(payload map[string]interface{}) error
}

// BaseVerifier provides common functionality for all verifiers
type BaseVerifier struct {
	TaskType models.TaskType
}

// NewBaseVerifier creates a new base verifier
func NewBaseVerifier(taskType models.TaskType) *BaseVerifier {
	return &BaseVerifier{
		TaskType: taskType,
	}
}

// GetTaskType returns the task type
func (bv *BaseVerifier) GetTaskType() models.TaskType {
	return bv.TaskType
}

// VerifierRegistry manages all task verifiers
type VerifierRegistry struct {
	verifiers map[models.TaskType]TaskVerifier
}

// NewVerifierRegistry creates a new verifier registry
func NewVerifierRegistry() *VerifierRegistry {
	return &VerifierRegistry{
		verifiers: make(map[models.TaskType]TaskVerifier),
	}
}

// Register registers a verifier for a task type
func (vr *VerifierRegistry) Register(verifier TaskVerifier) {
	vr.verifiers[verifier.GetTaskType()] = verifier
}

// GetVerifier returns a verifier for the given task type
func (vr *VerifierRegistry) GetVerifier(taskType models.TaskType) (TaskVerifier, error) {
	verifier, exists := vr.verifiers[taskType]
	if !exists {
		return nil, errors.New("no verifier registered for task type: " + string(taskType))
	}
	return verifier, nil
}

// RegisterVerifier registers a new verifier for a task type
func (vr *VerifierRegistry) RegisterVerifier(taskType string, verifier TaskVerifier) {
	vr.verifiers[models.TaskType(taskType)] = verifier
}

// GetSupportedTaskTypes returns all supported task types
func (vr *VerifierRegistry) GetSupportedTaskTypes() []models.TaskType {
	types := make([]models.TaskType, 0, len(vr.verifiers))
	for taskType := range vr.verifiers {
		types = append(types, taskType)
	}
	return types
}
