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
)

// BatchVerifier batch verification service
type BatchVerifier struct {
	taskService  *TaskService
	vlcService   *EnhancedVLCService
	pointsClient *points.Client // Points service client

	// Async processing queue
	taskQueue chan *models.Task
	workers   int
	mu        sync.RWMutex
	running   bool
}

// TaskInfo single task information
type TaskInfo struct {
	TweetID   string `json:"tweet_id"`
	TwitterID string `json:"twitter_id"`
}

// BatchVerificationPayload batch verification payload
type BatchVerificationPayload struct {
	StartTime string     `json:"start_time"`
	EndTime   string     `json:"end_time"`
	BatchSize int        `json:"batch_size"`
	Tasks     []TaskInfo `json:"tasks"`
}

// BatchVerificationResult batch verification result
type BatchVerificationResult struct {
	TotalTasks      int       `json:"total_tasks"`
	VerifiedTasks   int       `json:"verified_tasks"`
	UnverifiedTasks int       `json:"unverified_tasks"`
	VLCIncrement    int       `json:"vlc_increment"`
	ProcessedAt     time.Time `json:"processed_at"`
}

// NewBatchVerifier creates batch verifier
func NewBatchVerifier(taskService *TaskService, vlcService *EnhancedVLCService, pointsServiceURL string, workers int) *BatchVerifier {
	if workers <= 0 {
		workers = 5 // Default 5 workers
	}

	var pointsClient *points.Client
	if pointsServiceURL != "" {
		pointsClient = points.NewClient(pointsServiceURL)
	}

	return &BatchVerifier{
		taskService:  taskService,
		vlcService:   vlcService,
		pointsClient: pointsClient,
		taskQueue:    make(chan *models.Task, 1000), // Queue buffer
		workers:      workers,
	}
}

// Start starts batch verification service
func (bv *BatchVerifier) Start(ctx context.Context) error {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	if bv.running {
		return fmt.Errorf("batch verifier is already running")
	}

	bv.running = true

	// Start worker goroutines
	for i := 0; i < bv.workers; i++ {
		go bv.worker(ctx, i)
	}

	log.Printf("BatchVerifier started with %d workers", bv.workers)
	return nil
}

// Stop stops batch verification service
func (bv *BatchVerifier) Stop() {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	if !bv.running {
		return
	}

	bv.running = false
	close(bv.taskQueue)
	log.Println("BatchVerifier stopped")
}

// SubmitTask submits task for async verification
func (bv *BatchVerifier) SubmitTask(task *models.Task) error {
	bv.mu.RLock()
	defer bv.mu.RUnlock()

	if !bv.running {
		return fmt.Errorf("batch verifier is not running")
	}

	select {
	case bv.taskQueue <- task:
		// Immediately update task status to processing
		go func() {
			ctx := context.Background()
			bv.taskService.updateTaskStatus(ctx, task.ID, "PROCESSING")
		}()
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// worker working goroutine
func (bv *BatchVerifier) worker(ctx context.Context, workerID int) {
	log.Printf("BatchVerifier worker %d started", workerID)

	for task := range bv.taskQueue {
		select {
		case <-ctx.Done():
			return
		default:
			bv.processTask(ctx, task, workerID)
		}
	}

	log.Printf("BatchVerifier worker %d stopped", workerID)
}

// processTask processes a single task
func (bv *BatchVerifier) processTask(ctx context.Context, task *models.Task, workerID int) {
	log.Printf("Worker %d processing task %s", workerID, task.ID)

	var payload BatchVerificationPayload
	payloadJSON, _ := json.Marshal(task.Payload)
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		bv.handleError(ctx, task.ID, fmt.Errorf("failed to unmarshal payload: %w", err))
		return
	}

	// Process in batches to avoid calling too many APIs at once
	const batchSize = 10
	verifiedTasks := 0
	unverifiedTasks := 0

	for i := 0; i < len(payload.Tasks); i += batchSize {
		end := i + batchSize
		if end > len(payload.Tasks) {
			end = len(payload.Tasks)
		}

		batch := payload.Tasks[i:end]
		verified, unverified := bv.processBatch(ctx, batch, workerID)
		verifiedTasks += verified
		unverifiedTasks += unverified

		// Add delay to avoid API limits
		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Second):
			// Continue processing
		}
	}

	// Create verification result
	result := BatchVerificationResult{
		TotalTasks:      len(payload.Tasks),
		VerifiedTasks:   verifiedTasks,
		UnverifiedTasks: unverifiedTasks,
		VLCIncrement:    min(verifiedTasks, 10), // Maximum increment of 10
		ProcessedAt:     time.Now(),
	}

	resultJSON, _ := json.Marshal(result)

	// Update task status to verified
	if err := bv.taskService.updateTaskStatusWithProof(ctx, task.ID, "VERIFIED", resultJSON); err != nil {
		bv.handleError(ctx, task.ID, fmt.Errorf("failed to update task status: %w", err))
		return
	}

	// Trigger VLC increment
	if result.VLCIncrement > 0 {
		// Construct VLC increment payload
		vlcPayload := map[string]interface{}{
			"increment":      result.VLCIncrement,
			"batch_size":     result.TotalTasks,
			"verified_count": result.VerifiedTasks,
		}
		bv.vlcService.IncrementForTask(ctx, task.ID, "batch_verification", "verification", vlcPayload)
	}

	// 🎯 Key: Distribute points (after validator voting passes)
	if bv.pointsClient != nil && result.VerifiedTasks > 0 {
		if err := bv.distributePointsForBatch(ctx, task.ID, payload.Tasks, result.VerifiedTasks); err != nil {
			log.Printf("Warning: Failed to distribute points for batch %s: %v", task.ID, err)
			// Don't block main flow, only log errors
		}
	}

	log.Printf("Worker %d completed task %s: %d/%d verified", workerID, task.ID, verifiedTasks, len(payload.Tasks))
}

// processBatch processes a batch of verification tasks
func (bv *BatchVerifier) processBatch(ctx context.Context, tasks []TaskInfo, workerID int) (verified, unverified int) {
	log.Printf("Worker %d processing batch of %d tasks", workerID, len(tasks))

	for _, taskInfo := range tasks {
		select {
		case <-ctx.Done():
			return verified, unverified
		default:
			if bv.verifyTwitterTaskWithRetry(ctx, taskInfo.TweetID, taskInfo.TwitterID) {
				verified++
			} else {
				unverified++
			}
		}
	}
	return verified, unverified
}

// verifyTwitterTaskWithRetry Twitter verification with retry
func (bv *BatchVerifier) verifyTwitterTaskWithRetry(ctx context.Context, tweetID, twitterID string) bool {
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return false
		default:
			if bv.verifyTwitterTask(ctx, tweetID, twitterID) {
				return true
			}

			// Retry delay
			retryDelay := time.Duration(i+1) * time.Second
			select {
			case <-ctx.Done():
				return false
			case <-time.After(retryDelay):
				continue
			}
		}
	}
	return false
}

// verifyTwitterTask verifies Twitter task (mock implementation)
func (bv *BatchVerifier) verifyTwitterTask(ctx context.Context, tweetID, twitterID string) bool {
	// Should call actual Twitter API verification here
	// Currently return mock results

	// Mock API call delay
	select {
	case <-ctx.Done():
		return false
	case <-time.After(100 * time.Millisecond):
		// Mock 80% success rate
		return time.Now().UnixNano()%5 != 0
	}
}

// handleError handles verification error
func (bv *BatchVerifier) handleError(ctx context.Context, taskID string, err error) {
	log.Printf("Error processing task %s: %v", taskID, err)

	errorResult := map[string]interface{}{
		"error":     err.Error(),
		"timestamp": time.Now(),
	}
	errorJSON, _ := json.Marshal(errorResult)
	bv.taskService.updateTaskStatusWithProof(ctx, taskID, "FAILED", errorJSON)
}

// GetQueueStats gets queue statistics information
func (bv *BatchVerifier) GetQueueStats() map[string]interface{} {
	bv.mu.RLock()
	defer bv.mu.RUnlock()

	return map[string]interface{}{
		"running":    bv.running,
		"workers":    bv.workers,
		"queue_size": len(bv.taskQueue),
		"queue_cap":  cap(bv.taskQueue),
	}
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// distributePointsForBatch distributes points for batch
func (bv *BatchVerifier) distributePointsForBatch(ctx context.Context, batchID string, tasks []TaskInfo, verifiedCount int) error {
	log.Printf("Starting points distribution for batch %s with %d verified tasks", batchID, verifiedCount)

	// 🔍 TODO: Need to get actual VLC and user information from database here
	// Currently using mock data as example
	pointsTasks := make([]points.TaskVLC, 0, len(tasks))

	for _, taskInfo := range tasks {
		// 🔍 TODO: Query task details from database
		// - Query actual user wallet address based on TaskInfo.TweetID and TwitterID
		// - Determine task type (creation vs retweet)
		// - Get actual VLC value

		// Temporary mock data (needs to be replaced with actual queries)
		taskVLC := points.TaskVLC{
			UserWallet: "0x" + taskInfo.TwitterID,              // Temporary: use TwitterID as wallet address
			TaskType:   bv.determineTaskType(taskInfo.TweetID), // Needs implementation
			VLCValue:   1,                                      // Temporary: each verified task VLC=1
			TaskID:     taskInfo.TweetID,
		}
		pointsTasks = append(pointsTasks, taskVLC)
	}

	// Build points distribution request
	req := &points.PointsDistributionRequest{
		BatchID:     batchID,
		TriggerType: "validator_voting",
		Timestamp:   time.Now(),
		Tasks:       pointsTasks,
	}

	// Call points service to distribute points
	result, err := bv.pointsClient.DistributePoints(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to call points service: %w", err)
	}

	// Record distribution results
	log.Printf("Points distribution completed for batch %s: status=%s, users=%d, total_points=%d",
		batchID, result.Status, len(result.UserAllocations), result.TotalPoolPoints)

	if result.Status == "failed" {
		return fmt.Errorf("points distribution failed: %s", result.ErrorMessage)
	}

	// Statistics of distribution results
	successCount := 0
	for _, allocation := range result.UserAllocations {
		if allocation.UpdateStatus == "success" {
			successCount++
		}
	}

	log.Printf("Points distribution summary for batch %s: %d/%d users updated successfully",
		batchID, successCount, len(result.UserAllocations))

	return nil
}

// determineTaskType determines task type (temporary implementation, needs to be improved based on actual business logic)
func (bv *BatchVerifier) determineTaskType(tweetID string) string {
	// 🔍 TODO: Implement actual task type judgment logic
	// May need:
	// 1. Query task records in database
	// 2. Judge based on tweet content or task source
	// 3. Or directly include task type information in TaskInfo

	// Temporary implementation: simple rules
	if len(tweetID) > 0 && tweetID[0] == 'c' {
		return "creation" // Assume tasks starting with 'c' are creation tasks
	}
	return "retweet" // Default to retweet tasks
}
