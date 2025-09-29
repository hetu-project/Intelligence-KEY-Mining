package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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

// schedulerLoop main scheduler loop - supports both auto and daily modes
func (vs *ValidatorScheduler) schedulerLoop() {
	// Get scheduler mode from environment variable
	schedulerMode := os.Getenv("SCHEDULER_MODE")
	if schedulerMode == "" {
		schedulerMode = "daily" // Default to daily mode for production
	}

	log.Printf("ValidatorScheduler starting in %s mode", schedulerMode)

	if schedulerMode == "auto" {
		// Auto mode: Use VALIDATOR_POLL_INTERVAL_SECONDS for testing
		vs.runAutoMode()
	} else {
		// Daily mode: Fixed midnight execution for production
		vs.runDailyMode()
	}
}

// runAutoMode runs scheduler in auto mode (for testing)
func (vs *ValidatorScheduler) runAutoMode() {
	log.Printf("🔄 Auto mode: Running validation every %v", vs.pollInterval)

	// Start immediately
	log.Printf("🔄 Running initial validation at %v", time.Now().Format("2006-01-02 15:04:05"))
	vs.processPendingTasks()

	// Then run at regular intervals
	ticker := time.NewTicker(vs.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-vs.ctx.Done():
			return
		case <-ticker.C:
			log.Printf("🔄 Running scheduled validation at %v", time.Now().Format("2006-01-02 15:04:05"))
			vs.processPendingTasks()
		}
	}
}

// runDailyMode runs scheduler in daily mode (for production)
func (vs *ValidatorScheduler) runDailyMode() {
	log.Printf("🌙 Daily mode: Running validation at midnight daily")

	// Calculate time until next midnight
	now := time.Now()
	nextMidnight := vs.getNextMidnight(now)
	initialDelay := time.Until(nextMidnight)

	log.Printf("First validation will run at %v (in %v)", nextMidnight.Format("2006-01-02 15:04:05"), initialDelay)

	// Wait until first midnight
	select {
	case <-vs.ctx.Done():
		return
	case <-time.After(initialDelay):
		// Run first validation at midnight
		log.Printf("🌙 Running midnight validation at %v", time.Now().Format("2006-01-02 15:04:05"))
		vs.processPendingTasks()
	}

	// Then run every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-vs.ctx.Done():
			return
		case <-ticker.C:
			log.Printf("🌙 Running daily midnight validation at %v", time.Now().Format("2006-01-02 15:04:05"))
			vs.processPendingTasks()
		}
	}
}

// getNextMidnight calculates the next midnight time
func (vs *ValidatorScheduler) getNextMidnight(now time.Time) time.Time {
	// Get tomorrow's date at 00:00:00
	tomorrow := now.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
}

// processPendingTasks processes pending verification tasks with daily distribution check
func (vs *ValidatorScheduler) processPendingTasks() {
	ctx := vs.ctx
	today := time.Now()

	log.Printf("🔍 Starting daily task processing for %s", today.Format("2006-01-02"))

	// Check if today's distribution has already been completed
	// This would be implemented when we integrate with points service
	// For now, we proceed with task processing

	// 1. Process task creation tasks (individual verification)
	taskCreationTasks, err := vs.getTasksByTypeAndStatus(ctx, string(models.TaskCreationTask), "PENDING_VERIFICATION")
	if err != nil {
		log.Printf("Error fetching task creation tasks: %v", err)
	} else if len(taskCreationTasks) > 0 {
		// Check and filter expired tasks
		validTasks, err := vs.checkAndFilterExpiredTasks(ctx, taskCreationTasks)
		if err != nil {
			log.Printf("Error checking expired task creation tasks: %v", err)
		} else if len(validTasks) > 0 {
			log.Printf("Processing %d task creation tasks individually (filtered %d expired)", len(validTasks), len(taskCreationTasks)-len(validTasks))
			vs.processTaskCreationTasks(ctx, validTasks)
		} else {
			log.Printf("All %d task creation tasks have expired", len(taskCreationTasks))
		}
	} else {
		log.Printf("No task creation tasks found for processing")
	}

	// 2. Batch verify Twitter retweet tasks (this is an operation, not a task type!)
	twitterTasks, err := vs.getTasksByTypeAndStatus(ctx, "twitter_retweet", "PENDING_VERIFICATION")
	if err != nil {
		log.Printf("Error fetching twitter retweet tasks: %v", err)
	} else if len(twitterTasks) > 0 {
		// Check and filter expired tasks
		validTasks, err := vs.checkAndFilterExpiredTasks(ctx, twitterTasks)
		if err != nil {
			log.Printf("Error checking expired twitter tasks: %v", err)
		} else if len(validTasks) > 0 {
			log.Printf("Batch verifying %d twitter retweet tasks (filtered %d expired)", len(validTasks), len(twitterTasks)-len(validTasks))
			vs.batchVerifyTwitterTasks(ctx, validTasks)
		} else {
			log.Printf("All %d twitter tasks have expired", len(twitterTasks))
		}
	} else {
		log.Printf("No twitter retweet tasks found for processing")
	}

	log.Printf("✅ Completed daily task processing for %s", today.Format("2006-01-02"))
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
				// TaskCreation verification passed: direct VLC++, task completed
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
					// NOTE: TaskCreation points will be distributed during daily 0-point distribution
					// as 5% commission to subnet creators
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

// NOTE: distributeTaskCreationPoints method removed
// TaskCreation creators now receive 5% commission during daily 0-point distribution

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

// checkAndFilterExpiredTasks checks and marks expired tasks, returns valid tasks
func (vs *ValidatorScheduler) checkAndFilterExpiredTasks(ctx context.Context, tasks []*models.Task) ([]*models.Task, error) {
	var validTasks []*models.Task
	now := time.Now()

	for _, task := range tasks {
		// Check if task is expired
		if task.ExpiresAt != nil && now.After(*task.ExpiresAt) {
			// Mark as expired if not already expired
			if task.Status != models.TaskExpired {
				if err := vs.taskService.updateTaskStatus(ctx, task.ID, models.TaskExpired); err != nil {
					log.Printf("Failed to mark task %s as expired: %v", task.ID, err)
				} else {
					log.Printf("Task %s marked as expired (expired at: %v)", task.ID, *task.ExpiresAt)
				}
			}
			// Skip expired tasks from processing
			continue
		}

		// Task is still valid
		validTasks = append(validTasks, task)
	}

	return validTasks, nil
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
