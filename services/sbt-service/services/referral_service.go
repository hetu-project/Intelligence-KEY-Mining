package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/sbt-service/models"
)

// ReferralService handles third-party referral API calls
type ReferralService struct {
	baseURL    string
	httpClient *http.Client
}

// ReferralAPIResponse represents the API response structure
type ReferralAPIResponse struct {
	Result ReferralResult `json:"result"`
}

// ReferralResult represents the referral data from API
type ReferralResult struct {
	InviterAddress   string   `json:"inviter_address"`
	InvitedAddresses []string `json:"invited_addresses"`
}

// NewReferralService creates a new referral service client
func NewReferralService(baseURL string) *ReferralService {
	if baseURL == "" {
		return nil // Return nil if no URL provided
	}

	return &ReferralService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetReferralInfo fetches referral information for a wallet address
func (rs *ReferralService) GetReferralInfo(ctx context.Context, walletAddress string) (*models.InvitationInfo, error) {
	if rs == nil {
		// Service not initialized, return empty info
		return &models.InvitationInfo{
			Invitees:     []string{},
			InviteeCount: 0,
		}, nil
	}

	// Build request URL with query parameters
	reqURL, err := url.Parse(rs.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid referral API URL: %v", err)
	}

	params := url.Values{}
	params.Add("address", walletAddress)
	reqURL.RawQuery = params.Encode()

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SBT-Service/1.0")

	// Execute request with retry logic
	resp, err := rs.executeWithRetry(req, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to call referral API: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("referral API returned status %d", resp.StatusCode)
	}

	// Parse response
	var apiResponse ReferralAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Convert to InvitationInfo
	invitationInfo := &models.InvitationInfo{
		Invitees:     apiResponse.Result.InvitedAddresses,
		InviteeCount: len(apiResponse.Result.InvitedAddresses),
	}

	// Set inviter information if exists
	if apiResponse.Result.InviterAddress != "" {
		invitationInfo.Inviter = apiResponse.Result.InviterAddress
		invitationInfo.InviterHash = createAddressHash(apiResponse.Result.InviterAddress)
	}

	return invitationInfo, nil
}

// executeWithRetry executes HTTP request with retry logic
func (rs *ReferralService) executeWithRetry(req *http.Request, maxRetries int) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Clone request for retry
		reqClone := req.Clone(req.Context())

		resp, err := rs.httpClient.Do(reqClone)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on last attempt
		if attempt == maxRetries {
			break
		}

		// Wait before retry with exponential backoff
		waitTime := time.Duration(attempt+1) * 2 * time.Second
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(waitTime):
			continue
		}
	}

	return nil, lastErr
}

// GetReferralInfoSafe is a safe wrapper that never returns errors
// Used to ensure referral API failures don't break SBT dynamic data
func (rs *ReferralService) GetReferralInfoSafe(ctx context.Context, walletAddress string) *models.InvitationInfo {
	// Set timeout for the operation
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	invitationInfo, err := rs.GetReferralInfo(ctx, walletAddress)
	if err != nil {
		// Log error but don't propagate it
		fmt.Printf("Warning: Failed to fetch referral info for %s: %v\n", walletAddress, err)

		// Return empty invitation info
		return &models.InvitationInfo{
			Invitees:     []string{},
			InviteeCount: 0,
		}
	}

	return invitationInfo
}

// createAddressHash creates a privacy-friendly hash of an address
func createAddressHash(address string) string {
	if len(address) > 10 {
		return address[:6] + "..." + address[len(address)-4:]
	}
	return address
}
