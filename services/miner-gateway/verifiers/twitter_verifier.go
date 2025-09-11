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

// TwitterVerifier handles Twitter retweet task verification
type TwitterVerifier struct {
	*BaseVerifier
	middleLayerURL string
	apiKey         string
	client         *http.Client
}

// NewTwitterVerifier creates a new Twitter verifier
func NewTwitterVerifier(middleLayerURL, apiKey string) *TwitterVerifier {
	return &TwitterVerifier{
		BaseVerifier:   NewBaseVerifier(models.TwitterRetweetTask),
		middleLayerURL: middleLayerURL,
		apiKey:         apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidatePayload validates Twitter retweet payload format
func (tv *TwitterVerifier) ValidatePayload(payload map[string]interface{}) error {
	tweetID, exists := payload["tweet_id"]
	if !exists || tweetID == "" {
		return fmt.Errorf("tweet_id is required")
	}

	twitterID, exists := payload["twitter_id"]
	if !exists || twitterID == "" {
		return fmt.Errorf("twitter_id is required")
	}

	return nil
}

// ValidateSync performs synchronous Twitter retweet verification
func (tv *TwitterVerifier) ValidateSync(ctx context.Context, payload map[string]interface{}) (bool, *models.TaskProof, error) {
	if err := tv.ValidatePayload(payload); err != nil {
		return false, nil, err
	}

	// Build verification request
	verifyReq := map[string]interface{}{
		"tweet_id":   payload["tweet_id"],
		"twitter_id": payload["twitter_id"],
		"action":     "verify_retweet",
	}

	reqBody, err := json.Marshal(verifyReq)
	if err != nil {
		return false, nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Call middleware API
	req, err := http.NewRequestWithContext(ctx, "POST", tv.middleLayerURL+"/verify-retweet", bytes.NewBuffer(reqBody))
	if err != nil {
		return false, nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tv.apiKey)

	resp, err := tv.client.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("failed to call middle layer: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success        bool                   `json:"success"`
		Verified       bool                   `json:"verified"`
		VerificationID string                 `json:"verification_id"`
		Evidence       map[string]interface{} `json:"evidence"`
		Signature      string                 `json:"signature"`
		Message        string                 `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if !result.Success {
		return false, nil, fmt.Errorf("verification failed: %s", result.Message)
	}

	if !result.Verified {
		return false, nil, nil // Not completed retweet, but not an error
	}

	// Build proof
	proof := &models.TaskProof{
		Provider:       "twitter-middle-layer",
		VerifiedAt:     time.Now(),
		Evidence:       result.Evidence,
		VerificationID: result.VerificationID,
		Signature:      result.Signature,
	}

	return true, proof, nil
}

// RegisterAsyncWatch registers async watch for Twitter retweet
func (tv *TwitterVerifier) RegisterAsyncWatch(ctx context.Context, payload map[string]interface{}) (string, error) {
	if err := tv.ValidatePayload(payload); err != nil {
		return "", err
	}

	// Build async monitoring request
	watchReq := map[string]interface{}{
		"tweet_id":     payload["tweet_id"],
		"twitter_id":   payload["twitter_id"],
		"action":       "register_watch",
		"callback_url": "", // If webhook callback is needed
	}

	reqBody, err := json.Marshal(watchReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tv.middleLayerURL+"/register-watch", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tv.apiKey)

	resp, err := tv.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call middle layer: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		WatchID string `json:"watch_id"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if !result.Success {
		return "", fmt.Errorf("failed to register watch: %s", result.Message)
	}

	return result.WatchID, nil
}

// CheckAsyncStatus checks the status of an async watch
func (tv *TwitterVerifier) CheckAsyncStatus(ctx context.Context, watchID string) (bool, *models.TaskProof, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tv.middleLayerURL+"/check-watch/"+watchID, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+tv.apiKey)

	resp, err := tv.client.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("failed to call middle layer: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success        bool                   `json:"success"`
		Completed      bool                   `json:"completed"`
		Verified       bool                   `json:"verified"`
		VerificationID string                 `json:"verification_id"`
		Evidence       map[string]interface{} `json:"evidence"`
		Signature      string                 `json:"signature"`
		Message        string                 `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if !result.Success {
		return false, nil, fmt.Errorf("failed to check status: %s", result.Message)
	}

	if !result.Completed {
		return false, nil, nil // Not completed yet
	}

	if !result.Verified {
		return false, nil, fmt.Errorf("verification failed")
	}

	// Build proof
	proof := &models.TaskProof{
		Provider:       "twitter-middle-layer",
		VerifiedAt:     time.Now(),
		Evidence:       result.Evidence,
		VerificationID: result.VerificationID,
		Signature:      result.Signature,
	}

	return true, proof, nil
}
