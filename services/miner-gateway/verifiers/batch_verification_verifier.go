package verifiers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// BatchVerificationVerifier handles batch verification of Twitter tasks
type BatchVerificationVerifier struct {
	*BaseVerifier
	middleLayerURL string
	apiKey         string
	client         *http.Client
}

// NewBatchVerificationVerifier creates a new batch verification verifier
func NewBatchVerificationVerifier(middleLayerURL, apiKey string) *BatchVerificationVerifier {
	return &BatchVerificationVerifier{
		BaseVerifier:   NewBaseVerifier(models.BatchVerificationTask),
		middleLayerURL: middleLayerURL,
		apiKey:         apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second, // Batch verification may need more time
		},
	}
}

// BatchVerificationRequest represents the request to middle layer
type BatchVerificationRequest struct {
	TimeRange struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	} `json:"time_range"`
	Tasks []VerificationTask `json:"tasks"`
}

// VerificationTask represents a single task to verify
type VerificationTask struct {
	TweetID   string `json:"tweet_id"`
	TwitterID string `json:"twitter_id"`
}

// BatchVerificationResponse represents the response from middle layer
type BatchVerificationResponse struct {
	Success bool                 `json:"success"`
	Results []VerificationResult `json:"results"`
	Message string               `json:"message"`
}

// VerificationResult represents a single verification result
type VerificationResult struct {
	TwitterID string `json:"twitter_id"`
	TweetID   string `json:"tweet_id"`
	Verified  bool   `json:"verified"`
}

// ValidatePayload validates batch verification payload format
func (bvv *BatchVerificationVerifier) ValidatePayload(payload map[string]interface{}) error {
	startTime, exists := payload["start_time"]
	if !exists || startTime == "" {
		return fmt.Errorf("start_time is required")
	}

	endTime, exists := payload["end_time"]
	if !exists || endTime == "" {
		return fmt.Errorf("end_time is required")
	}

	// Validate time format
	if _, err := time.Parse(time.RFC3339, startTime.(string)); err != nil {
		return fmt.Errorf("invalid start_time format, expected RFC3339")
	}

	if _, err := time.Parse(time.RFC3339, endTime.(string)); err != nil {
		return fmt.Errorf("invalid end_time format, expected RFC3339")
	}

	return nil
}

// ValidateSync performs synchronous batch verification
func (bvv *BatchVerificationVerifier) ValidateSync(ctx context.Context, payload map[string]interface{}) (bool, *models.TaskProof, error) {
	if err := bvv.ValidatePayload(payload); err != nil {
		return false, nil, err
	}

	// Extract task list from payload
	tasksInterface, exists := payload["tasks"]
	if !exists {
		return false, nil, fmt.Errorf("tasks list is required")
	}

	tasksSlice, ok := tasksInterface.([]interface{})
	if !ok {
		return false, nil, fmt.Errorf("tasks must be an array")
	}

	// Build batch verification request
	var tasks []VerificationTask
	for _, taskInterface := range tasksSlice {
		taskMap, ok := taskInterface.(map[string]interface{})
		if !ok {
			continue
		}

		tweetID, _ := taskMap["tweet_id"].(string)
		twitterID, _ := taskMap["twitter_id"].(string)

		if tweetID != "" && twitterID != "" {
			tasks = append(tasks, VerificationTask{
				TweetID:   tweetID,
				TwitterID: twitterID,
			})
		}
	}

	if len(tasks) == 0 {
		return false, nil, fmt.Errorf("no valid tasks to verify")
	}

	// Build request
	batchReq := BatchVerificationRequest{
		Tasks: tasks,
	}
	batchReq.TimeRange.StartTime = payload["start_time"].(string)
	batchReq.TimeRange.EndTime = payload["end_time"].(string)

	reqBody, err := json.Marshal(batchReq)
	if err != nil {
		return false, nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Call middleware API
	req, err := http.NewRequestWithContext(ctx, "POST", bvv.middleLayerURL+"/batch-verify", bytes.NewBuffer(reqBody))
	if err != nil {
		return false, nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bvv.apiKey)

	resp, err := bvv.client.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("failed to call middle layer: %v", err)
	}
	defer resp.Body.Close()

	var result BatchVerificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if !result.Success {
		return false, nil, fmt.Errorf("batch verification failed: %s", result.Message)
	}

	// Build proof
	proof := &models.TaskProof{
		Provider:   "batch-verification-middle-layer",
		VerifiedAt: time.Now(),
		Evidence: map[string]interface{}{
			"total_tasks":    len(tasks),
			"results":        result.Results,
			"verified_count": countVerifiedResults(result.Results),
		},
		VerificationID: fmt.Sprintf("batch_%d", time.Now().Unix()),
		Signature:      "", // Batch verification uses internal signature mechanism
	}

	return true, proof, nil
}

// countVerifiedResults counts how many results were verified
func countVerifiedResults(results []VerificationResult) int {
	count := 0
	for _, result := range results {
		if result.Verified {
			count++
		}
	}
	return count
}

// RegisterAsyncWatch batch verification doesn't need async monitoring
func (bvv *BatchVerificationVerifier) RegisterAsyncWatch(ctx context.Context, payload map[string]interface{}) (string, error) {
	return "", fmt.Errorf("batch verification does not support async watch")
}

// CheckAsyncStatus batch verification doesn't need async status checking
func (bvv *BatchVerificationVerifier) CheckAsyncStatus(ctx context.Context, watchID string) (bool, *models.TaskProof, error) {
	return false, nil, fmt.Errorf("batch verification does not support async status check")
}
