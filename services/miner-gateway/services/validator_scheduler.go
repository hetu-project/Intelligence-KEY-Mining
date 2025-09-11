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
) *ValidatorScheduler {
	return &ValidatorScheduler{
		taskService:          taskService,
		taskCreationVerifier: taskCreationVerifier,
		batchVerifier:        batchVerifier,
		pollInterval:         30 * time.Second, // Default poll every 30 seconds
		batchSize:            50,               // Process 50 tasks each time
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

	// Get pending task creation tasks for verification
	taskCreationTasks, err := vs.getTasksByTypeAndStatus(ctx, string(models.TaskCreationTask), "PENDING_VERIFICATION")
	if err != nil {
		log.Printf("Error fetching task creation tasks: %v", err)
	} else if len(taskCreationTasks) > 0 {
		log.Printf("Processing %d task creation tasks", len(taskCreationTasks))
		vs.processTaskCreationTasks(ctx, taskCreationTasks)
	}

	// Get pending batch verification tasks for verification
	batchTasks, err := vs.getTasksByTypeAndStatus(ctx, string(models.BatchVerificationTask), "SUBMITTED")
	if err != nil {
		log.Printf("Error fetching batch verification tasks: %v", err)
	} else if len(batchTasks) > 0 {
		log.Printf("Processing %d batch verification tasks", len(batchTasks))
		vs.processBatchVerificationTasks(ctx, batchTasks)
	}

	// Get pending Twitter retweet tasks for verification
	twitterTasks, err := vs.getTasksByTypeAndStatus(ctx, "twitter_retweet", "PENDING_VERIFICATION")
	if err != nil {
		log.Printf("Error fetching twitter retweet tasks: %v", err)
	} else if len(twitterTasks) > 0 {
		log.Printf("Processing %d twitter retweet tasks", len(twitterTasks))
		vs.processTwitterRetweetTasks(ctx, twitterTasks)
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

// processBatchVerificationTasks process tasks
func (vs *ValidatorScheduler) processBatchVerificationTasks(ctx context.Context, tasks []*models.Task) {
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return
		default:
			if vs.batchVerifier != nil {
				if err := vs.batchVerifier.SubmitTask(task); err != nil {
					log.Printf("Error submitting batch verification task %s: %v", task.ID, err)
				}
			}
		}
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
