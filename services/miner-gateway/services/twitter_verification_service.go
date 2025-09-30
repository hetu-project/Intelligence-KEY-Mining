package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// TwitterVerificationService handles Twitter retweet verification using the new external API
type TwitterVerificationService struct {
	retweetCheckURL string
	db              *sql.DB
	client          *http.Client
	batchSize       int
	concurrency     int
	callInterval    time.Duration
}

// TwitterRetweetCheckRequest represents the request to the external verification API
type TwitterRetweetCheckRequest struct {
	MediaAccount string `json:"media_account"` // Twitter account that posted the original tweet
	XID          string `json:"x_id"`          // User's Twitter ID
	PostID       string `json:"post_id"`       // Tweet ID
	StartTime    string `json:"start_time"`    // Start time for verification window
	EndTime      string `json:"end_time"`      // End time for verification window
}

// TwitterRetweetCheckResponse represents the response from the external verification API
type TwitterRetweetCheckResponse struct {
	HasRetweet bool   `json:"has_retweet"`
	Message    string `json:"message"`
}

// TwitterVerificationResult represents the result of verifying a single task
type TwitterVerificationResult struct {
	TaskID      string
	UserWallet  string
	TweetID     string
	Verified    bool
	Error       error
	APIResponse *TwitterRetweetCheckResponse
}

// TwitterBatchVerificationResult represents the result of Twitter batch verification
type TwitterBatchVerificationResult struct {
	TotalTasks    int
	VerifiedTasks int
	FailedTasks   int
	Results       []TwitterVerificationResult
	Duration      time.Duration
	VLCIncrement  int
}

// NewTwitterVerificationService creates a new Twitter verification service
func NewTwitterVerificationService(retweetCheckURL string, db *sql.DB) *TwitterVerificationService {
	return &TwitterVerificationService{
		retweetCheckURL: retweetCheckURL,
		db:              db,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		batchSize:    50,                     // Process 50 tasks per batch
		concurrency:  5,                      // 5 concurrent API calls
		callInterval: 100 * time.Millisecond, // 100ms between calls
	}
}

// VerifyTwitterRetweetTask verifies a single Twitter retweet task for ALL registered users
func (tvs *TwitterVerificationService) VerifyTwitterRetweetTask(ctx context.Context, task *models.Task) ([]*TwitterVerificationResult, error) {
	tweetID, twitterUsername, err := tvs.extractTaskInfo(task)
	if err != nil {
		return nil, fmt.Errorf("task info extraction failed: %v", err)
	}

	// Get all registered users
	users, err := tvs.getAllRegisteredUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get registered users: %v", err)
	}

	log.Printf("Verifying task %s (tweet: %s) for %d registered users", task.ID, tweetID, len(users))

	results := make([]*TwitterVerificationResult, 0, len(users))

	// Verify each user's retweet status for this task
	for _, userWallet := range users {
		result := &TwitterVerificationResult{
			TaskID:     task.ID,
			UserWallet: userWallet,
			TweetID:    tweetID,
			Verified:   false,
			Error:      nil,
		}

		// Skip verification if user has already completed this task
		alreadyCompleted, err := tvs.checkUserTaskCompletion(ctx, userWallet, task.ID)
		if err != nil {
			log.Printf("Failed to check task completion for user %s, task %s: %v", userWallet, task.ID, err)
			result.Error = fmt.Errorf("failed to check completion status: %v", err)
			results = append(results, result)
			continue
		}

		if alreadyCompleted {
			log.Printf("User %s has already completed task %s, skipping verification", userWallet, task.ID)
			continue // Skip users who have already completed this task
		}

		// Get user's Twitter ID
		userTwitterID, err := tvs.getUserTwitterID(ctx, userWallet)
		if err != nil {
			log.Printf("User %s Twitter ID not found: %v", userWallet, err)
			result.Error = fmt.Errorf("user twitter ID not found: %v", err)
			results = append(results, result)
			continue
		}

		// Verify if this user retweeted the task's tweet
		verified, apiResponse, err := tvs.callRetweetCheckAPIWithRetry(ctx, tweetID, twitterUsername, userTwitterID, task)
		if err != nil {
			log.Printf("API call failed for user %s, task %s: %v", userWallet, task.ID, err)
			result.Error = fmt.Errorf("API call failed: %v", err)
			result.APIResponse = apiResponse
			results = append(results, result)
			continue
		}

		result.Verified = verified
		result.APIResponse = apiResponse
		results = append(results, result)

		if verified {
			log.Printf("✅ User %s has retweeted tweet %s (task %s)", userWallet, tweetID, task.ID)
		} else {
			log.Printf("❌ User %s has not retweeted tweet %s (task %s)", userWallet, tweetID, task.ID)
		}
	}

	log.Printf("Task %s verification completed: %d results for %d users", task.ID, len(results), len(users))
	return results, nil
}

// extractTaskInfo
func (tvs *TwitterVerificationService) extractTaskInfo(task *models.Task) (tweetID, twitterUsername string, err error) {
	tweetID, ok := task.Payload["tweet_id"].(string)
	if !ok || tweetID == "" {
		return "", "", fmt.Errorf("missing or invalid tweet_id")
	}

	twitterUsername, ok = task.Payload["twitter_username"].(string)
	if !ok || twitterUsername == "" {
		return tweetID, "", fmt.Errorf("missing or invalid twitter_username")
	}

	return tweetID, twitterUsername, nil
}

// BatchVerifyTwitterTasks verifies multiple Twitter retweet tasks
func (tvs *TwitterVerificationService) BatchVerifyTwitterTasks(ctx context.Context, tasks []*models.Task) (*TwitterBatchVerificationResult, error) {
	startTime := time.Now()

	log.Printf("Starting batch verification of %d Twitter tasks", len(tasks))

	result := &TwitterBatchVerificationResult{
		TotalTasks: len(tasks),
		Results:    make([]TwitterVerificationResult, 0, len(tasks)),
	}

	// Process tasks in batches with concurrency control
	for i := 0; i < len(tasks); i += tvs.batchSize {
		end := i + tvs.batchSize
		if end > len(tasks) {
			end = len(tasks)
		}

		batch := tasks[i:end]
		log.Printf("Processing batch %d-%d of %d tasks", i+1, end, len(tasks))

		batchResults, err := tvs.processBatch(ctx, batch)
		if err != nil {
			log.Printf("Error processing batch %d-%d: %v", i+1, end, err)
			// Continue with next batch even if this one fails
		}

		result.Results = append(result.Results, batchResults...)

		// Add delay between batches to avoid overwhelming the API
		if end < len(tasks) {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(time.Second):
				// Continue to next batch
			}
		}
	}

	// Calculate final statistics
	for _, res := range result.Results {
		if res.Verified {
			result.VerifiedTasks++
		} else {
			result.FailedTasks++
		}
	}

	result.Duration = time.Since(startTime)
	result.VLCIncrement = result.VerifiedTasks // VLC increment based on successful verifications

	log.Printf("Batch verification completed: %d verified, %d failed, duration: %v",
		result.VerifiedTasks, result.FailedTasks, result.Duration)

	return result, nil
}

// processBatch processes a single batch of tasks with concurrency control
func (tvs *TwitterVerificationService) processBatch(ctx context.Context, tasks []*models.Task) ([]TwitterVerificationResult, error) {
	results := make([]TwitterVerificationResult, len(tasks))
	var wg sync.WaitGroup

	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, tvs.concurrency)

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, t *models.Task) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Add delay between API calls
			if index > 0 {
				select {
				case <-ctx.Done():
					results[index] = TwitterVerificationResult{
						TaskID:     t.ID,
						UserWallet: t.UserWallet,
						Verified:   false,
						Error:      ctx.Err(),
					}
					return
				case <-time.After(tvs.callInterval):
					// Continue
				}
			}

			// Note: This method is deprecated - just create a placeholder result
			results[index] = TwitterVerificationResult{
				TaskID:     t.ID,
				UserWallet: t.UserWallet,
				Verified:   false,
				Error:      fmt.Errorf("BatchVerifyTwitterTasks is deprecated - use VerifyTwitterRetweetTask directly"),
			}
		}(i, task)
	}

	wg.Wait()
	return results, nil
}

// callRetweetCheckAPIWithRetry
func (tvs *TwitterVerificationService) callRetweetCheckAPIWithRetry(ctx context.Context, tweetID, twitterUsername, userTwitterID string, task *models.Task) (bool, *TwitterRetweetCheckResponse, error) {
	const maxRetries = 3
	const timeoutPerCall = 10 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		verified, resp, err := tvs.callRetweetCheckAPISingle(ctx, tweetID, twitterUsername, userTwitterID, task, timeoutPerCall)

		if err == nil {
			return verified, resp, nil
		}

		log.Printf("Twitter API call attempt %d/%d failed for task %s: %v", attempt, maxRetries, task.ID, err)

		if attempt == maxRetries {
			return false, resp, fmt.Errorf("all %d attempts failed, last error: %v", maxRetries, err)
		}

		select {
		case <-ctx.Done():
			return false, resp, ctx.Err()
		case <-time.After(time.Duration(attempt) * time.Second):
			continue
		}
	}

	return false, nil, fmt.Errorf("unexpected retry loop exit")
}

// callRetweetCheckAPISingle
func (tvs *TwitterVerificationService) callRetweetCheckAPISingle(ctx context.Context, tweetID, twitterUsername, userTwitterID string, task *models.Task, timeout time.Duration) (bool, *TwitterRetweetCheckResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime, endTime := tvs.calculateTimeWindow(task)

	req := TwitterRetweetCheckRequest{
		MediaAccount: twitterUsername,
		XID:          userTwitterID,
		PostID:       tweetID,
		StartTime:    startTime,
		EndTime:      endTime,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return false, nil, fmt.Errorf("request marshal failed: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(callCtx, "POST", tvs.retweetCheckURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return false, nil, fmt.Errorf("request creation failed: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := tvs.client.Do(httpReq)
	if err != nil {
		return false, nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp TwitterRetweetCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {

		return false, &apiResp, fmt.Errorf("response decode failed: %v", err)
	}

	return apiResp.HasRetweet, &apiResp, nil
}

// callRetweetCheckAPI
func (tvs *TwitterVerificationService) callRetweetCheckAPI(ctx context.Context, req TwitterRetweetCheckRequest) (bool, *TwitterRetweetCheckResponse, error) {
	task := &models.Task{
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	return tvs.callRetweetCheckAPIWithRetry(ctx, req.PostID, req.MediaAccount, req.XID, task)
}

// getAllRegisteredUsers gets all registered users from the database
func (tvs *TwitterVerificationService) getAllRegisteredUsers(ctx context.Context) ([]string, error) {
	query := "SELECT wallet_address FROM user_profiles"
	rows, err := tvs.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %v", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var wallet string
		if err := rows.Scan(&wallet); err != nil {
			log.Printf("Failed to scan user wallet: %v", err)
			continue
		}
		users = append(users, wallet)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %v", err)
	}

	return users, nil
}

// checkUserTaskCompletion checks if a user has already completed a specific task
func (tvs *TwitterVerificationService) checkUserTaskCompletion(ctx context.Context, userWallet, taskID string) (bool, error) {
	query := "SELECT COUNT(*) FROM user_task_completions WHERE user_wallet = ? AND task_id = ?"
	var count int
	err := tvs.db.QueryRowContext(ctx, query, userWallet, taskID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check task completion: %v", err)
	}
	return count > 0, nil
}

// getUserTwitterID gets the user's Twitter ID from the database
func (tvs *TwitterVerificationService) getUserTwitterID(ctx context.Context, userWallet string) (string, error) {
	var twitterID sql.NullString
	query := "SELECT twitter_id FROM user_profiles WHERE wallet_address = ?"

	err := tvs.db.QueryRowContext(ctx, query, userWallet).Scan(&twitterID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user not found: %s", userWallet)
		}
		return "", fmt.Errorf("database query failed: %v", err)
	}

	if !twitterID.Valid || twitterID.String == "" {
		return "", fmt.Errorf("user %s has no Twitter ID", userWallet)
	}

	return twitterID.String, nil
}

// calculateTimeWindow calculates the time window for verification
func (tvs *TwitterVerificationService) calculateTimeWindow(task *models.Task) (string, string) {
	// End time is current time
	endTime := time.Now().UTC()

	// Start time is 2 hours ago (or since task creation, whichever is more recent)
	startTime := endTime.Add(-2 * time.Hour)
	if task.CreatedAt.After(startTime) {
		startTime = task.CreatedAt
	}

	// Format as ISO 8601
	return startTime.Format("2006-01-02T15:04:05Z"), endTime.Format("2006-01-02T15:04:05Z")
}
