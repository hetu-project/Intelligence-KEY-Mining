package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// CallbackService handles third-party callback operations
type CallbackService struct {
	client *http.Client
}

// NewCallbackService creates a new callback service
func NewCallbackService() *CallbackService {
	return &CallbackService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PointsDistributionCallbackRequest represents the callback request
type PointsDistributionCallbackRequest struct {
	Address string `json:"address"`
}

// PointsDistributionCallbackResponse represents the callback response
type PointsDistributionCallbackResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// CallPointsDistributionCallback calls the third-party callback after points distribution
func (cs *CallbackService) CallPointsDistributionCallback(ctx context.Context, userWallet string) {
	// Use goroutine to avoid blocking the main flow
	go func() {
		success, err := cs.doCallbackRequest(context.Background(), userWallet)
		if err != nil {
			log.Printf("Points distribution callback failed for user %s: %v", userWallet, err)
		} else if success {
			log.Printf("Points distribution callback successful for user %s", userWallet)
		} else {
			log.Printf("Points distribution callback returned false for user %s", userWallet)
		}
	}()
}

// CallPointsDistributionCallbackBatch calls callback for multiple users
func (cs *CallbackService) CallPointsDistributionCallbackBatch(ctx context.Context, userWallets []string) {
	if len(userWallets) == 0 {
		return
	}

	// Use goroutine to avoid blocking the main flow
	go func() {
		log.Printf("Starting points distribution callback for %d users", len(userWallets))

		successCount := 0
		for _, wallet := range userWallets {
			success, err := cs.doCallbackRequest(context.Background(), wallet)
			if err != nil {
				log.Printf("Callback failed for user %s: %v", wallet, err)
			} else if success {
				successCount++
			} else {
				log.Printf("Callback returned false for user %s", wallet)
			}

			// Small delay to avoid overwhelming the third-party service
			time.Sleep(100 * time.Millisecond)
		}

		log.Printf("Completed points distribution callback: %d/%d successful", successCount, len(userWallets))
	}()
}

// doCallbackRequest performs the actual HTTP request
func (cs *CallbackService) doCallbackRequest(ctx context.Context, userWallet string) (bool, error) {
	callbackURL := os.Getenv("POINTS_DISTRIBUTION_CALLBACK_URL")
	if callbackURL == "" {
		// No callback URL configured, skip silently
		return true, nil
	}

	// Create request body
	requestBody := PointsDistributionCallbackRequest{
		Address: userWallet,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", callbackURL, bytes.NewReader(requestJSON))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}

	// Note: No API key required for this callback
	req.Header.Set("Content-Type", "application/json")

	resp, err := cs.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to call callback API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var callbackResp PointsDistributionCallbackResponse
	if err := json.Unmarshal(body, &callbackResp); err != nil {
		// Try to parse as boolean response for backward compatibility
		bodyStr := string(body)
		if bodyStr == "true" {
			return true, nil
		} else if bodyStr == "false" {
			return false, nil
		}
		return false, fmt.Errorf("failed to parse response: %v", err)
	}

	return callbackResp.Success, nil
}
