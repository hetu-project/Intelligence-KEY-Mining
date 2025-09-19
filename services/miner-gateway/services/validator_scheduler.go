package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/points"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/verifiers"
)

// ValidatorScheduler validator scheduler
type ValidatorScheduler struct {
	taskService          *TaskService
	taskCreationVerifier *verifiers.TaskCreationVerifier
	batchVerifier        *BatchVerifier
	pointsClient         *points.Client // New: for TaskCreation points distribution

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
	pointsServiceURL string,
	pollIntervalSeconds int,
) *ValidatorScheduler {
	pollInterval := time.Duration(pollIntervalSeconds) * time.Second

	// Initialize points client for TaskCreation immediate points distribution
	var pointsClient *points.Client
	if pointsServiceURL != "" {
		pointsClient = points.NewClient(pointsServiceURL)
	}

	return &ValidatorScheduler{
		taskService:          taskService,
		taskCreationVerifier: taskCreationVerifier,
		batchVerifier:        batchVerifier,
		pointsClient:         pointsClient,
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

// processTaskCreationTasks process tasks - simplified flow
func (vs *ValidatorScheduler) processTaskCreationTasks(ctx context.Context, tasks []*models.Task) {
	log.Printf("Processing %d TaskCreation tasks with simplified flow", len(tasks))

	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return
		default:
			// TaskCreation is just the action of creating a task
			// Simple validation: check if payload contains required fields
			if vs.validateTaskCreationPayload(task.Payload) {
				// TaskCreation验证通过：直接VLC++，任务完成
				log.Printf("TaskCreation %s validated successfully - completing task", task.ID)

				// Create simple proof
				proof := map[string]interface{}{
					"validation_type": "task_creation",
					"validated_at":    time.Now(),
					"status":          "completed",
					"message":         "Task creation action completed successfully",
				}
				proofJSON, _ := json.Marshal(proof)

				// Update to VERIFIED status (TaskCreation tasks can be VERIFIED as they're just creation actions)
				if err := vs.taskService.updateTaskStatusWithProof(ctx, task.ID, "VERIFIED", proofJSON); err != nil {
					log.Printf("Error updating TaskCreation status %s: %v", task.ID, err)
				} else {
					log.Printf("✅ TaskCreation %s completed successfully", task.ID)

					// 🆕 NEW: Immediately distribute 50 points for TaskCreation
					vs.distributeTaskCreationPoints(ctx, task)
				}
			} else {
				// Validation failed
				log.Printf("TaskCreation %s validation failed - invalid payload", task.ID)
				proof := map[string]interface{}{
					"validation_type": "task_creation",
					"validated_at":    time.Now(),
					"status":          "failed",
					"message":         "Invalid task creation payload",
				}
				proofJSON, _ := json.Marshal(proof)

				if err := vs.taskService.updateTaskStatusWithProof(ctx, task.ID, "FAILED", proofJSON); err != nil {
					log.Printf("Error updating TaskCreation status %s: %v", task.ID, err)
				}
			}
		}
	}
}

// validateTaskCreationPayload validates TaskCreation payload (simple validation)
func (vs *ValidatorScheduler) validateTaskCreationPayload(payload map[string]interface{}) bool {
	// Simple validation for task creation
	if payload == nil {
		return false
	}

	// Check if required fields exist (basic validation)
	if taskDescription, exists := payload["task_description"]; exists {
		if desc, ok := taskDescription.(string); ok && len(desc) > 0 {
			return true
		}
	}

	// If no task_description, check for other basic fields
	if len(payload) > 0 {
		return true // Basic validation: payload is not empty
	}

	return false
}

// distributeTaskCreationPoints immediately distributes 50 points for TaskCreation completion
func (vs *ValidatorScheduler) distributeTaskCreationPoints(ctx context.Context, task *models.Task) {
	if vs.pointsClient == nil {
		log.Printf("No points client available for TaskCreation %s", task.ID)
		return
	}

	log.Printf("💰 Distributing 50 points for TaskCreation %s to user %s", task.ID, task.UserWallet)

	// Create points distribution request for TaskCreation (fixed 50 points)
	taskVLC := points.TaskVLC{
		UserWallet: task.UserWallet,
		TaskType:   "creation",
		VLCValue:   50, // Fixed 50 points for TaskCreation
		TaskID:     task.ID,
	}

	pointsReq := &points.PointsDistributionRequest{
		BatchID:     fmt.Sprintf("task-creation-%s", task.ID),
		TriggerType: "task_creation_completed",
		Timestamp:   time.Now(),
		Tasks:       []points.TaskVLC{taskVLC},
	}

	// Distribute points with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	result, err := vs.pointsClient.DistributePoints(ctxWithTimeout, pointsReq)
	if err != nil {
		log.Printf("❌ Failed to distribute TaskCreation points for %s: %v", task.ID, err)
	} else {
		log.Printf("✅ TaskCreation points distributed successfully for %s", task.ID)
		if result != nil && len(result.UserAllocations) > 0 {
			userResult := result.UserAllocations[0]
			if userResult.UpdateStatus == "success" {
				log.Printf("   💰 User %s received %d points for TaskCreation", userResult.UserWallet[:10]+"...", userResult.RoundedPoints)
			}
		}
	}
}

// batchVerifyTwitterTasks performs batch verification of Twitter retweet tasks
// Uses batch round management for PoCW consensus
func (vs *ValidatorScheduler) batchVerifyTwitterTasks(ctx context.Context, tasks []*models.Task) {
	if vs.batchVerifier != nil {
		log.Printf("Starting batch verification round for %d Twitter tasks", len(tasks))

		// Start a new batch verification round
		round := vs.batchVerifier.StartBatchRound(tasks, models.TwitterRetweetTask)

		// Process all tasks in the round
		for _, task := range tasks {
			if err := vs.batchVerifier.SubmitTask(task); err != nil {
				log.Printf("Error submitting Twitter task %s: %v", task.ID, err)
			}
			// Note: Actual verification happens asynchronously
			// Task completion tracking is handled in BatchVerifier
		}

		log.Printf("Batch round %s: submitted %d tasks for verification", round.RoundID, len(tasks))
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
