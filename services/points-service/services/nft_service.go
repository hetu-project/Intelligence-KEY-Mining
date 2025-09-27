package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// NFTService handles NFT-related operations
type NFTService struct {
	db     *sql.DB
	client *http.Client
}

// NewNFTService creates a new NFT service
func NewNFTService(db *sql.DB) *NFTService {
	return &NFTService{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NFTCheckResponse represents the response from NFT check API
type NFTCheckResponse struct {
	HasNFT  bool   `json:"has_nft"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CheckUserNFTOwnership checks if a user owns an NFT
func (ns *NFTService) CheckUserNFTOwnership(ctx context.Context, userWallet string) (bool, error) {
	// 1. Check cache first
	cached, err := ns.getCachedNFTStatus(ctx, userWallet)
	if err == nil && cached != nil {
		log.Printf("Using cached NFT status for user %s: %v", userWallet, cached.HasNFT)
		return cached.HasNFT, nil
	}

	// 2. Call external API
	hasNFT, apiResponse, err := ns.callNFTCheckAPI(ctx, userWallet)
	if err != nil {
		log.Printf("Failed to check NFT ownership for user %s: %v", userWallet, err)
		return false, err
	}

	// 3. Cache the result
	cacheDuration := ns.getCacheDuration()
	expiresAt := time.Now().Add(cacheDuration)

	if err := ns.cacheNFTStatus(ctx, userWallet, hasNFT, apiResponse, expiresAt); err != nil {
		log.Printf("Failed to cache NFT status for user %s: %v", userWallet, err)
		// Don't fail the request if caching fails
	}

	log.Printf("NFT ownership check for user %s: %v", userWallet, hasNFT)
	return hasNFT, nil
}

// callNFTCheckAPI calls the external NFT check API
func (ns *NFTService) callNFTCheckAPI(ctx context.Context, userWallet string) (bool, string, error) {
	apiURL := os.Getenv("NFT_CHECK_API_URL")
	if apiURL == "" {
		return false, "", fmt.Errorf("NFT_CHECK_API_URL not configured")
	}

	// Replace {wallet} placeholder with actual wallet address
	apiURL = strings.ReplaceAll(apiURL, "{wallet}", userWallet)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %v", err)
	}

	// Add API key if configured
	if apiKey := os.Getenv("NFT_CHECK_API_KEY"); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ns.client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to call NFT API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, string(body), fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var nftResp NFTCheckResponse
	if err := json.Unmarshal(body, &nftResp); err != nil {
		// Try to parse as boolean response
		if strings.ToLower(strings.TrimSpace(string(body))) == "true" {
			return true, string(body), nil
		} else if strings.ToLower(strings.TrimSpace(string(body))) == "false" {
			return false, string(body), nil
		}
		return false, string(body), fmt.Errorf("failed to parse response: %v", err)
	}

	if nftResp.Error != "" {
		return false, string(body), fmt.Errorf("API error: %s", nftResp.Error)
	}

	return nftResp.HasNFT, string(body), nil
}

// getCachedNFTStatus gets cached NFT status
func (ns *NFTService) getCachedNFTStatus(ctx context.Context, userWallet string) (*NFTOwnershipCache, error) {
	query := `
		SELECT user_wallet, has_nft, checked_at, expires_at, api_response 
		FROM nft_ownership_cache 
		WHERE user_wallet = ? AND expires_at > NOW()
	`

	var cache NFTOwnershipCache
	err := ns.db.QueryRowContext(ctx, query, userWallet).Scan(
		&cache.UserWallet,
		&cache.HasNFT,
		&cache.CheckedAt,
		&cache.ExpiresAt,
		&cache.APIResponse,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No cache found
		}
		return nil, err
	}

	return &cache, nil
}

// cacheNFTStatus caches NFT status
func (ns *NFTService) cacheNFTStatus(ctx context.Context, userWallet string, hasNFT bool, apiResponse string, expiresAt time.Time) error {
	query := `
		INSERT INTO nft_ownership_cache (user_wallet, has_nft, checked_at, expires_at, api_response)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			has_nft = VALUES(has_nft),
			checked_at = VALUES(checked_at),
			expires_at = VALUES(expires_at),
			api_response = VALUES(api_response)
	`

	_, err := ns.db.ExecContext(ctx, query, userWallet, hasNFT, time.Now(), expiresAt, apiResponse)
	return err
}

// getCacheDuration gets cache duration from environment
func (ns *NFTService) getCacheDuration() time.Duration {
	hoursStr := os.Getenv("NFT_CACHE_DURATION_HOURS")
	if hoursStr == "" {
		return 24 * time.Hour // Default 24 hours
	}

	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours <= 0 {
		return 24 * time.Hour // Default 24 hours
	}

	return time.Duration(hours) * time.Hour
}

// CleanExpiredCache cleans expired NFT cache entries
func (ns *NFTService) CleanExpiredCache(ctx context.Context) (int, error) {
	query := `DELETE FROM nft_ownership_cache WHERE expires_at < NOW()`
	result, err := ns.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// NFTOwnershipCache represents cached NFT ownership data
type NFTOwnershipCache struct {
	UserWallet  string    `json:"user_wallet"`
	HasNFT      bool      `json:"has_nft"`
	CheckedAt   time.Time `json:"checked_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	APIResponse string    `json:"api_response"`
}

// GetNFTCacheStats gets NFT cache statistics
func (ns *NFTService) GetNFTCacheStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	// Total cache entries
	var totalEntries int
	err := ns.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM nft_ownership_cache").Scan(&totalEntries)
	if err != nil {
		return nil, err
	}
	stats["total_entries"] = totalEntries

	// Active cache entries (not expired)
	var activeEntries int
	err = ns.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM nft_ownership_cache WHERE expires_at > NOW()").Scan(&activeEntries)
	if err != nil {
		return nil, err
	}
	stats["active_entries"] = activeEntries

	// Expired entries
	stats["expired_entries"] = totalEntries - activeEntries

	// Users with NFT
	var usersWithNFT int
	err = ns.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM nft_ownership_cache WHERE has_nft = true AND expires_at > NOW()").Scan(&usersWithNFT)
	if err != nil {
		return nil, err
	}
	stats["users_with_nft"] = usersWithNFT

	return stats, nil
}
