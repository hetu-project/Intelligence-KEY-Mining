package points

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client points service client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates points service client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PointsDistributionRequest points distribution request
type PointsDistributionRequest struct {
	BatchID     string    `json:"batch_id"`
	TriggerType string    `json:"trigger_type"`
	Timestamp   time.Time `json:"timestamp"`
	Tasks       []TaskVLC `json:"tasks"`
}

// TaskVLC task VLC information
type TaskVLC struct {
	UserWallet string `json:"user_wallet"`
	TaskType   string `json:"task_type"` // "creation" or "retweet"
	VLCValue   int    `json:"vlc_value"`
	TaskID     string `json:"task_id"`
}

// PointsDistributionResult points distribution result
type PointsDistributionResult struct {
	BatchID          string             `json:"batch_id"`
	TotalPoolPoints  int                `json:"total_pool_points"`
	CreationPoints   int                `json:"creation_points"`
	RetweetPoints    int                `json:"retweet_points"`
	TotalCreationVLC int                `json:"total_creation_vlc"`
	TotalRetweetVLC  int                `json:"total_retweet_vlc"`
	UserAllocations  []UserPointsResult `json:"user_allocations"`
	ProcessedAt      time.Time          `json:"processed_at"`
	Status           string             `json:"status"`
	ErrorMessage     string             `json:"error_message,omitempty"`
}

// UserPointsResult user points allocation result
type UserPointsResult struct {
	UserWallet     string  `json:"user_wallet"`
	CreationVLC    int     `json:"creation_vlc"`
	RetweetVLC     int     `json:"retweet_vlc"`
	CreationPoints float64 `json:"creation_points"`
	RetweetPoints  float64 `json:"retweet_points"`
	TotalPoints    float64 `json:"total_points"`
	RoundedPoints  int     `json:"rounded_points"`
	UpdateStatus   string  `json:"update_status"`
	UpdateError    string  `json:"update_error,omitempty"`
}

// DistributePoints distributes points
func (c *Client) DistributePoints(ctx context.Context, req *PointsDistributionRequest) (*PointsDistributionResult, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/points/distribute", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		return nil, fmt.Errorf("points service error (status %d): %v", resp.StatusCode, errorResp)
	}

	var response struct {
		Status string                   `json:"status"`
		Data   PointsDistributionResult `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response.Data, nil
}

// GetUserPoints gets user points
func (c *Client) GetUserPoints(ctx context.Context, walletAddress string) (int, error) {
	url := fmt.Sprintf("%s/api/v1/points/user/%s", c.baseURL, walletAddress)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get user points (status %d)", resp.StatusCode)
	}

	var response struct {
		Status string `json:"status"`
		Data   struct {
			TotalPoints int `json:"total_points"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Data.TotalPoints, nil
}

// HealthCheck health check
func (c *Client) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed (status %d)", resp.StatusCode)
	}

	return nil
}
