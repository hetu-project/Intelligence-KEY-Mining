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
	taskService            *TaskService
	vlcService             *EnhancedVLCService
	pointsClient           *points.Client              // Points service client
	twitterVerificationSvc *TwitterVerificationService // New Twitter verification service
	subnetService          *SubnetService              // Subnet service for duplicate checking

	// Async processing queue
	taskQueue chan *models.Task
	workers   int
	mu        sync.RWMutex
	running   bool

	// Batch round management
	currentRound        *models.BatchVerificationRound
	roundMutex          sync.RWMutex
	roundCompletionChan chan *models.BatchVerificationRound
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
func NewBatchVerifier(taskService *TaskService, vlcService *EnhancedVLCService, pointsServiceURL string, workers int, twitterVerificationSvc *TwitterVerificationService, subnetService *SubnetService) *BatchVerifier {
	if workers <= 0 {
		workers = 5 // Default 5 workers
	}

	var pointsClient *points.Client
	if pointsServiceURL != "" {
		pointsClient = points.NewClient(pointsServiceURL)
	}

	return &BatchVerifier{
		taskService:            taskService,
		vlcService:             vlcService,
		pointsClient:           pointsClient,
		twitterVerificationSvc: twitterVerificationSvc,
		subnetService:          subnetService,
		taskQueue:              make(chan *models.Task, 1000), // Queue buffer
		workers:                workers,
		roundCompletionChan:    make(chan *models.BatchVerificationRound, 10), // Buffer for completed rounds
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

// processTask processes a single task with full fault tolerance
func (bv *BatchVerifier) processTask(ctx context.Context, task *models.Task, workerID int) {
	log.Printf("Worker %d processing task %s (type: %s)", workerID, task.ID, task.TaskType)

	if task.TaskType == models.TwitterRetweetTask {
		if bv.twitterVerificationSvc != nil {
			results, err := bv.twitterVerificationSvc.VerifyTwitterRetweetTask(ctx, task)
			if err != nil {
				log.Printf("Unexpected error in Twitter verification: %v", err)
				bv.handleTwitterTaskAsIncomplete(ctx, task, fmt.Errorf("verification service error: %v", err))
				return
			}

			log.Printf("Worker %d: Task %s verification completed for %d users", workerID, task.ID, len(results))

			// Process results for each user
			verifiedCount := 0
			for _, result := range results {
				if result.Verified {
					log.Printf("Worker %d: User %s verified task %s successfully", workerID, result.UserWallet, task.ID)
					bv.handleUserVerificationSuccess(ctx, task, result)
					verifiedCount++
				} else if result.Error != nil {
					log.Printf("Worker %d: User %s verification error for task %s: %v", workerID, result.UserWallet, task.ID, result.Error)
					// Don't mark task as incomplete for individual user errors
				} else {
					log.Printf("Worker %d: User %s has not completed task %s", workerID, result.UserWallet, task.ID)
					// User hasn't retweeted - this is normal, not an error
				}
			}

			log.Printf("Worker %d: Task %s processed - %d users verified out of %d checked", workerID, task.ID, verifiedCount, len(results))

			// Task remains PENDING_VERIFICATION regardless of individual user results
			// This allows future users to complete the task and allows re-verification
			bv.taskService.updateTaskStatus(ctx, task.ID, "PENDING_VERIFICATION")
		} else {
			log.Printf("Worker %d: Twitter verification service not available, marking task %s as incomplete", workerID, task.ID)
			bv.handleTwitterTaskAsIncomplete(ctx, task, fmt.Errorf("twitter verification service not configured"))
		}
		return
	}

	bv.processTaskLegacy(ctx, task, workerID)
}

// processTaskLegacy processes a single task using legacy batch verification logic
func (bv *BatchVerifier) processTaskLegacy(ctx context.Context, task *models.Task, workerID int) {
	log.Printf("Worker %d processing task %s with legacy logic", workerID, task.ID)

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

	// Keep task status as PENDING_VERIFICATION - no status change needed
	// Task remains available for future batch verifications
	// Status update removed to allow continuous verification

	// Trigger VLC increment
	if result.VLCIncrement > 0 {
		// Construct VLC increment payload
		// Batch verification is now an operation, not a task type
		// For now, just increment the miner VLC once for the batch operation
		// TODO: In step 4, we'll properly track individual task verification
		vlcPayload := map[string]interface{}{
			"batch_operation": true,
			"batch_size":      result.TotalTasks,
			"verified_count":  result.VerifiedTasks,
		}
		// Use empty user wallet to increment only miner VLC for batch operation
		bv.vlcService.IncrementForTask(ctx, task.ID, models.TwitterRetweetTask, "verification", vlcPayload, "")
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

// handleUserVerificationSuccess handles successful Twitter verification for a specific user
func (bv *BatchVerifier) handleUserVerificationSuccess(ctx context.Context, task *models.Task, result *TwitterVerificationResult) {
	// 1. Check if user has already completed this task (should be already checked, but double-check)
	if bv.subnetService != nil {
		alreadyCompleted, err := bv.subnetService.CheckTaskCompletion(ctx, result.UserWallet, task.ID)
		if err != nil {
			log.Printf("Failed to check task completion for user %s, task %s: %v", result.UserWallet, task.ID, err)
		} else if alreadyCompleted {
			log.Printf("User %s has already completed task %s, skipping VLC increment and points", result.UserWallet, task.ID)
			return // Don't track this as it's already completed
		}
	}

	// 2. Record verification proof but keep task status as PENDING_VERIFICATION for continuous verification
	proofData := map[string]interface{}{
		"verification_result": result,
		"verified_at":         time.Now(),
		"verification_type":   "twitter_retweet_check",
		"continuous_check":    true, // Indicates this is part of continuous verification
	}

	proofJSON, _ := json.Marshal(proofData)

	// Update task proof but keep status as PENDING_VERIFICATION for continuous verification
	if err := bv.taskService.updateTaskStatusWithProof(ctx, task.ID, models.TaskPendingVerification, proofJSON); err != nil {
		log.Printf("Failed to update task proof for %s: %v", task.ID, err)
		return
	}

	// 3. Increment VLC for verified Twitter task for the specific user
	if bv.vlcService != nil {
		vlcPayload := map[string]interface{}{
			"verification_type": "twitter_retweet_check",
			"tweet_id":          result.TweetID,
			"verified":          true,
			"user_wallet":       result.UserWallet,
		}
		bv.vlcService.IncrementForTask(ctx, task.ID, models.TwitterRetweetTask, "verification", vlcPayload, result.UserWallet)
		log.Printf("VLC incremented for user %s completing Twitter task %s", result.UserWallet, task.ID)
	}

	// 4. Record task completion to prevent future duplicates for this specific user
	if bv.subnetService != nil {
		completion := &models.UserTaskCompletion{
			UserWallet:   result.UserWallet, // Use the user who actually completed the task
			TaskID:       task.ID,
			SubnetID:     task.SubnetID,
			CompletedAt:  time.Now(),
			VLCIncrement: 1, // Assuming 1 VLC increment for Twitter tasks
			PointsEarned: 0, // Points will be calculated during PoCW consensus
		}

		if err := bv.subnetService.RecordTaskCompletion(ctx, completion); err != nil {
			log.Printf("Failed to record task completion for user %s, task %s: %v", result.UserWallet, task.ID, err)
		}
	}

	// NOTE: Points distribution moved to PoCW consensus completion
	// No immediate points distribution for Twitter tasks - wait for PoCW consensus
	log.Printf("User %s completed Twitter task %s, awaiting PoCW consensus for points distribution", result.UserWallet, task.ID)

	// Note: We don't track task completion in batch round since multiple users can complete the same task
	// Each user completion is independent
}

// handleTwitterTaskAsIncomplete handles Twitter task verification failure
func (bv *BatchVerifier) handleTwitterTaskAsIncomplete(ctx context.Context, task *models.Task, reason error) {
	var reasonStr string
	if reason != nil {
		reasonStr = reason.Error()
	} else {
		reasonStr = "user has not retweeted"
	}

	// Record verification failure but keep task available for future verification
	proofData := map[string]interface{}{
		"verification_status": "incomplete",
		"reason":              reasonStr,
		"timestamp":           time.Now(),
		"verification_type":   "twitter_retweet_check",
		"note":                "Task remains PENDING_VERIFICATION for future attempts",
	}

	proofJSON, _ := json.Marshal(proofData)

	// Always keep status as PENDING_VERIFICATION - no status change
	// This allows the task to be verified again in future batch runs
	// when the user completes the retweet
	if err := bv.taskService.updateTaskStatusWithProof(ctx, task.ID, models.TaskPendingVerification, proofJSON); err != nil {
		log.Printf("Failed to update task proof for %s: %v", task.ID, err)
	}

	log.Printf("Task %s verification incomplete: %s (task remains PENDING_VERIFICATION)", task.ID, reasonStr)

	// Track task completion in current batch round as not completed
	bv.trackTaskCompletion(task.ID, false)
}

// distributePointsForVerifiedTask distributes points for a single verified task
func (bv *BatchVerifier) distributePointsForVerifiedTask(ctx context.Context, task *models.Task) error {
	taskVLC := points.TaskVLC{
		UserWallet: task.UserWallet,
		TaskType:   string(task.TaskType),
		VLCValue:   1, // VLC value for this task
		TaskID:     task.ID,
	}

	pointsReq := &points.PointsDistributionRequest{
		BatchID:     fmt.Sprintf("twitter-task-%s", task.ID),
		TriggerType: "task_verification",
		Timestamp:   time.Now(),
		Tasks:       []points.TaskVLC{taskVLC},
	}

	_, err := bv.pointsClient.DistributePoints(ctx, pointsReq)
	return err
}

// ===== Batch Round Management Methods =====

// StartBatchRound starts a new batch verification round
func (bv *BatchVerifier) StartBatchRound(tasks []*models.Task, taskType models.TaskType) *models.BatchVerificationRound {
	bv.roundMutex.Lock()
	defer bv.roundMutex.Unlock()

	roundID := fmt.Sprintf("batch_%s_%d", taskType, time.Now().Unix())

	round := &models.BatchVerificationRound{
		RoundID:    roundID,
		StartTime:  time.Now(),
		TaskType:   taskType,
		TotalTasks: len(tasks),
		Tasks:      tasks,
		VLCBefore:  bv.vlcService.GetMinerVLC(),
		Status:     "processing",
		VerificationSummary: map[string]interface{}{
			"started_at": time.Now(),
			"task_type":  taskType,
		},
	}

	bv.currentRound = round
	log.Printf("Started batch verification round %s with %d %s tasks", roundID, len(tasks), taskType)

	return round
}

// CompleteBatchRound completes the current batch verification round
func (bv *BatchVerifier) CompleteBatchRound(verifiedCount, failedCount int) {
	bv.roundMutex.Lock()
	defer bv.roundMutex.Unlock()

	if bv.currentRound == nil {
		log.Printf("Warning: No active batch round to complete")
		return
	}

	now := time.Now()
	bv.currentRound.EndTime = &now
	bv.currentRound.VerifiedTasks = verifiedCount
	bv.currentRound.FailedTasks = failedCount
	bv.currentRound.VLCAfter = bv.vlcService.GetMinerVLC()
	bv.currentRound.Status = "completed"

	// Update verification summary
	bv.currentRound.VerificationSummary["completed_at"] = now
	bv.currentRound.VerificationSummary["duration_seconds"] = now.Sub(bv.currentRound.StartTime).Seconds()
	bv.currentRound.VerificationSummary["success_rate"] = float64(verifiedCount) / float64(bv.currentRound.TotalTasks)

	log.Printf("Completed batch verification round %s: %d/%d tasks verified",
		bv.currentRound.RoundID, verifiedCount, bv.currentRound.TotalTasks)

	// Send to round completion channel for PoCW consensus
	select {
	case bv.roundCompletionChan <- bv.currentRound:
		log.Printf("Batch round %s queued for PoCW consensus", bv.currentRound.RoundID)
	default:
		log.Printf("Warning: Round completion channel full, dropping round %s", bv.currentRound.RoundID)
	}

	bv.currentRound = nil
}

// GetCompletedRounds returns completed rounds waiting for PoCW consensus
func (bv *BatchVerifier) GetCompletedRounds() chan *models.BatchVerificationRound {
	return bv.roundCompletionChan
}

// GetCurrentRound returns the current active batch round
func (bv *BatchVerifier) GetCurrentRound() *models.BatchVerificationRound {
	bv.roundMutex.RLock()
	defer bv.roundMutex.RUnlock()
	return bv.currentRound
}

// trackTaskCompletion tracks completion of a task in the current round
func (bv *BatchVerifier) trackTaskCompletion(taskID string, success bool) {
	bv.roundMutex.Lock()
	defer bv.roundMutex.Unlock()

	if bv.currentRound == nil {
		return // No active round
	}

	// Update task completion status in the round
	for _, task := range bv.currentRound.Tasks {
		if task.ID == taskID {
			if success {
				bv.currentRound.VerifiedTasks++
			} else {
				bv.currentRound.FailedTasks++
			}
			break
		}
	}

	// Check if all tasks are completed
	completedTasks := bv.currentRound.VerifiedTasks + bv.currentRound.FailedTasks
	if completedTasks >= bv.currentRound.TotalTasks {
		log.Printf("All tasks completed in round %s, finalizing round", bv.currentRound.RoundID)
		bv.finalizeBatchRound()
	}
}

// finalizeBatchRound finalizes the current batch round (internal method)
func (bv *BatchVerifier) finalizeBatchRound() {
	if bv.currentRound == nil {
		return
	}

	now := time.Now()
	bv.currentRound.EndTime = &now
	bv.currentRound.VLCAfter = bv.vlcService.GetMinerVLC()
	bv.currentRound.Status = "completed"

	// Update verification summary
	bv.currentRound.VerificationSummary["completed_at"] = now
	bv.currentRound.VerificationSummary["duration_seconds"] = now.Sub(bv.currentRound.StartTime).Seconds()
	bv.currentRound.VerificationSummary["success_rate"] = float64(bv.currentRound.VerifiedTasks) / float64(bv.currentRound.TotalTasks)

	log.Printf("Finalized batch verification round %s: %d/%d tasks verified",
		bv.currentRound.RoundID, bv.currentRound.VerifiedTasks, bv.currentRound.TotalTasks)

	// Send to round completion channel for PoCW consensus
	select {
	case bv.roundCompletionChan <- bv.currentRound:
		log.Printf("Batch round %s queued for PoCW consensus", bv.currentRound.RoundID)
	default:
		log.Printf("Warning: Round completion channel full, dropping round %s", bv.currentRound.RoundID)
	}

	bv.currentRound = nil
}
