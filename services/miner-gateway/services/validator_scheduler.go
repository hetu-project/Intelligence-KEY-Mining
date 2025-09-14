package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/verifiers"
)

// ValidatorScheduler validator scheduler
type ValidatorScheduler struct {
	taskService          *TaskService
	taskCreationVerifier *verifiers.TaskCreationVerifier
	batchVerifier        *BatchVerifier

	// Scheduling configuration
	pollInterval time.Duration
	batchSize    int

	// Runtime state
	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewValidatorScheduler creates validator scheduler
func NewValidatorScheduler(
	taskService *TaskService,
	taskCreationVerifier *verifiers.TaskCreationVerifier,
	batchVerifier *BatchVerifier,
	pollIntervalSeconds int,
) *ValidatorScheduler {
	pollInterval := time.Duration(pollIntervalSeconds) * time.Second
	return &ValidatorScheduler{
		taskService:          taskService,
		taskCreationVerifier: taskCreationVerifier,
		batchVerifier:        batchVerifier,
		pollInterval:         pollInterval, // Use configurable interval
		batchSize:            50,           // Process 50 tasks each time
	}
}

// Start starts the scheduler
func (vs *ValidatorScheduler) Start(parentCtx context.Context) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.running {
		return fmt.Errorf("validator scheduler is already running")
	}

	vs.ctx, vs.cancel = context.WithCancel(parentCtx)
	vs.running = true

	// Start scheduling goroutine
	go vs.schedulerLoop()

	log.Println("ValidatorScheduler started")
	return nil
}

// Stop stops the scheduler
func (vs *ValidatorScheduler) Stop() {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if !vs.running {
		return
	}

	vs.cancel()
	vs.running = false
	log.Println("ValidatorScheduler stopped")
}

// schedulerLoop main scheduler loop
func (vs *ValidatorScheduler) schedulerLoop() {
	ticker := time.NewTicker(vs.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-vs.ctx.Done():
			return
		case <-ticker.C:
			vs.processPendingTasks()
		}
	}
}

// processPendingTasks processes pending verification tasks
func (vs *ValidatorScheduler) processPendingTasks() {
	ctx := vs.ctx

	// 1. Process task creation tasks (individual verification)
	taskCreationTasks, err := vs.getTasksByTypeAndStatus(ctx, string(models.TaskCreationTask), "PENDING_VERIFICATION")
	if err != nil {
		log.Printf("Error fetching task creation tasks: %v", err)
	} else if len(taskCreationTasks) > 0 {
		log.Printf("Processing %d task creation tasks individually", len(taskCreationTasks))
		vs.processTaskCreationTasks(ctx, taskCreationTasks)
	}

	// 2. Batch verify Twitter retweet tasks (this is an operation, not a task type!)
	twitterTasks, err := vs.getTasksByTypeAndStatus(ctx, "twitter_retweet", "PENDING_VERIFICATION")
	if err != nil {
		log.Printf("Error fetching twitter retweet tasks: %v", err)
	} else if len(twitterTasks) > 0 {
		log.Printf("Batch verifying %d twitter retweet tasks", len(twitterTasks))
		vs.batchVerifyTwitterTasks(ctx, twitterTasks)
	}
}

// getTasksByTypeAndStatus get tasks by type and status
func (vs *ValidatorScheduler) getTasksByTypeAndStatus(ctx context.Context, taskType, status string) ([]*models.Task, error) {
	return vs.taskService.GetTasksByTypeAndStatus(ctx, taskType, status, vs.batchSize)
}

// processTaskCreationTasks process tasks
func (vs *ValidatorScheduler) processTaskCreationTasks(ctx context.Context, tasks []*models.Task) {
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return
		default:
			if vs.taskCreationVerifier != nil {
				// Execute sync verification
				valid, proof, err := vs.taskCreationVerifier.ValidateSync(ctx, task.Payload)
				if err != nil {
					log.Printf("Error verifying task creation task %s: %v", task.ID, err)
					continue
				}

				// Update task status
				var status models.TaskStatus
				if valid {
					status = "VERIFIED"
				} else {
					status = "FAILED"
				}

				var proofJSON []byte
				if proof != nil {
					proofJSON, _ = json.Marshal(proof)
				}

				if err := vs.taskService.updateTaskStatusWithProof(ctx, task.ID, status, proofJSON); err != nil {
					log.Printf("Error updating task creation status %s: %v", task.ID, err)
				}
			}
		}
	}
}

// batchVerifyTwitterTasks performs batch verification of Twitter retweet tasks
// This is an operation that processes multiple TwitterRetweetTask instances together
func (vs *ValidatorScheduler) batchVerifyTwitterTasks(ctx context.Context, tasks []*models.Task) {
	if vs.batchVerifier != nil {
		// Create a batch verification operation
		batchTask := vs.createBatchVerificationOperation(tasks)
		if err := vs.batchVerifier.SubmitTask(batchTask); err != nil {
			log.Printf("Error submitting batch verification operation: %v", err)
		}
	} else {
		// Fallback: process each Twitter task individually
		log.Printf("No batch verifier available, processing Twitter tasks individually")
		vs.processTwitterRetweetTasks(ctx, tasks)
	}
}

// processTwitterRetweetTasks process tasks
func (vs *ValidatorScheduler) processTwitterRetweetTasks(ctx context.Context, tasks []*models.Task) {
	// Can implement Twitter retweet task verification logic here
	// Or submit to other validators for processing
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return
		default:
			// Mock verification processing
			if err := vs.processTwitterRetweetTask(ctx, task); err != nil {
				log.Printf("Error processing twitter retweet task %s: %v", task.ID, err)
			}
		}
	}
}

// processTwitterRetweetTask process tasks
func (vs *ValidatorScheduler) processTwitterRetweetTask(ctx context.Context, task *models.Task) error {
	// Should implement actual Twitter retweet verification logic here
	// Currently mock verification process

	// Mock verification delay
	time.Sleep(100 * time.Millisecond)

	// mock 80% success rate
	verified := time.Now().UnixNano()%5 != 0

	var status models.TaskStatus
	var proof map[string]interface{}

	if verified {
		status = "VERIFIED"
		proof = map[string]interface{}{
			"verified":    true,
			"verified_at": time.Now(),
			"method":      "twitter_api",
		}
	} else {
		status = "FAILED"
		proof = map[string]interface{}{
			"verified":    false,
			"verified_at": time.Now(),
			"error":       "Retweet not found or not accessible",
		}
	}

	proofJSON, _ := json.Marshal(proof)
	return vs.taskService.updateTaskStatusWithProof(ctx, task.ID, status, proofJSON)
}

// createBatchVerificationOperation creates a batch verification operation from multiple Twitter tasks
func (vs *ValidatorScheduler) createBatchVerificationOperation(twitterTasks []*models.Task) *models.Task {
	// Create a synthetic batch operation task
	batchTaskID := fmt.Sprintf("batch_%d_%d", time.Now().Unix(), len(twitterTasks))

	// Extract task information for batch processing
	taskInfos := make([]map[string]interface{}, 0, len(twitterTasks))
	for _, task := range twitterTasks {
		taskInfo := map[string]interface{}{
			"task_id":     task.ID,
			"user_wallet": task.UserWallet,
			"tweet_id":    "", // Extract from payload if available
			"twitter_id":  "", // Extract from payload if available
		}

		// Extract Twitter-specific data from payload
		if tweetID, ok := task.Payload["tweet_id"].(string); ok {
			taskInfo["tweet_id"] = tweetID
		}
		if twitterID, ok := task.Payload["twitter_id"].(string); ok {
			taskInfo["twitter_id"] = twitterID
		}

		taskInfos = append(taskInfos, taskInfo)
	}

	// Create batch operation payload
	batchPayload := map[string]interface{}{
		"operation_type": "batch_twitter_verification",
		"tasks":          taskInfos,
		"batch_size":     len(twitterTasks),
		"created_at":     time.Now().Format(time.RFC3339),
	}

	// Return a synthetic task representing the batch operation
	return &models.Task{
		ID:         batchTaskID,
		UserWallet: "system",          // System operation
		TaskType:   "batch_operation", // Not a real task type, just for internal use
		Status:     models.TaskSubmitted,
		Payload:    batchPayload,
		Attempts:   0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// SetPollInterval set poll interval
func (vs *ValidatorScheduler) SetPollInterval(interval time.Duration) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.pollInterval = interval
}

// SetBatchSize sets batch processing size
func (vs *ValidatorScheduler) SetBatchSize(size int) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.batchSize = size
}

// GetStats gets statistics information
func (vs *ValidatorScheduler) GetStats() map[string]interface{} {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	return map[string]interface{}{
		"running":       vs.running,
		"poll_interval": vs.pollInterval.String(),
		"batch_size":    vs.batchSize,
	}
}
