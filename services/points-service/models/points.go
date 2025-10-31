package models

import (
	"time"
)

// Points source constants
const (
	PointsSourceTaskCreation      = "Task Creation"
	PointsSourceTwitterRetweet    = "Twitter Retweet Task"
	PointsSourceTwitterPost       = "Twitter Post Task"
	PointsSourceVLCDistribution   = "VLC Distribution"
	PointsSourceNFTPurchase       = "NFT Purchase Bonus"
	PointsSourceInvitationReward  = "Invitation Reward"
	PointsSourceCreatorCommission = "Creator Commission"
	PointsSourceTelegramTask      = "Telegram Task"
	PointsSourceChatTask          = "Chat Task"
)

// PointsDistributionRequest points distribution request
type PointsDistributionRequest struct {
	BatchID     string    `json:"batch_id" validate:"required"`    // Batch ID
	TriggerType string    `json:"trigger_type"`                    // Trigger type: "validator_voting"
	Timestamp   time.Time `json:"timestamp"`                       // Distribution time
	Tasks       []TaskVLC `json:"tasks" validate:"required,min=1"` // Task VLC list
}

// TaskVLC task VLC information
type TaskVLC struct {
	UserWallet string `json:"user_wallet" validate:"required"` // User wallet address
	TaskType   string `json:"task_type" validate:"required"`   // Task type: "creation", "retweet"
	VLCValue   int    `json:"vlc_value" validate:"min=0"`      // VLC value
	TaskID     string `json:"task_id"`                         // Task ID (optional, for records)
}

// PointsDistributionResult points distribution result - NEW: VLC directly equals points
type PointsDistributionResult struct {
	BatchID          string             `json:"batch_id"`
	TotalPoolPoints  int                `json:"total_pool_points"`  // NEW: Total actually distributed points
	CreationPoints   int                `json:"creation_points"`    // NEW: Total creation VLC (equals creation points)
	RetweetPoints    int                `json:"retweet_points"`     // NEW: Total retweet VLC (equals retweet points)
	TotalCreationVLC int                `json:"total_creation_vlc"` // Total creation VLC
	TotalRetweetVLC  int                `json:"total_retweet_vlc"`  // Total retweet VLC
	UserAllocations  []UserPointsResult `json:"user_allocations"`   // User allocation results
	ProcessedAt      time.Time          `json:"processed_at"`
	Status           string             `json:"status"` // "success", "failed", "partial"
	ErrorMessage     string             `json:"error_message,omitempty"`
}

// UserPointsResult user points allocation result
type UserPointsResult struct {
	UserWallet     string  `json:"user_wallet"`
	CreationVLC    int     `json:"creation_vlc"`    // User creation VLC
	RetweetVLC     int     `json:"retweet_vlc"`     // User retweet VLC
	CreationPoints float64 `json:"creation_points"` // Earned creation points
	RetweetPoints  float64 `json:"retweet_points"`  // Earned retweet points
	TotalPoints    float64 `json:"total_points"`    // Total points
	RoundedPoints  int     `json:"rounded_points"`  // Rounded points
	UpdateStatus   string  `json:"update_status"`   // "success", "failed"
	UpdateError    string  `json:"update_error,omitempty"`
}

// PointsRecord points record (compatible with SBT service)
type PointsRecord struct {
	WalletAddress string    `json:"wallet_address" db:"wallet_address"`
	Date          string    `json:"date" db:"date"`     // "2023-10-20"
	Source        string    `json:"source" db:"source"` // "VLC Distribution"
	Points        int       `json:"points" db:"points"`
	TxRef         string    `json:"tx_ref,omitempty" db:"tx_ref"` // Batch ID as reference
	SubnetID      string    `json:"subnet_id,omitempty" db:"subnet_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// PointsConfig points configuration - NEW: Simplified for direct VLC-to-points mapping
type PointsConfig struct {
	TotalPoolPoints int     `json:"total_pool_points"` // DEPRECATED: No longer used (kept for API compatibility)
	CreationRatio   float64 `json:"creation_ratio"`    // DEPRECATED: No longer used (kept for API compatibility)
	RetweetRatio    float64 `json:"retweet_ratio"`     // DEPRECATED: No longer used (kept for API compatibility)
	HistoryLimit    int     `json:"history_limit"`     // History record limit, default 1000
}

// DefaultPointsConfig default points configuration - NEW: Simplified
func DefaultPointsConfig() *PointsConfig {
	return &PointsConfig{
		TotalPoolPoints: 0,   // DEPRECATED: No longer relevant
		CreationRatio:   1.0, // DEPRECATED: No longer relevant
		RetweetRatio:    1.0, // DEPRECATED: No longer relevant
		HistoryLimit:    1000,
	}
}

// PointsStats points statistics
type PointsStats struct {
	TotalDistributions int       `json:"total_distributions"` // Total distributions
	TotalPointsIssued  int       `json:"total_points_issued"` // Total points issued
	ActiveUsers        int       `json:"active_users"`        // Active users count
	LastDistribution   time.Time `json:"last_distribution"`   // Last distribution time
	AvgPointsPerUser   float64   `json:"avg_points_per_user"` // Average points per user
}

// DirectPointsRequest represents a direct points addition request
type DirectPointsRequest struct {
	UserWallet  string                 `json:"user_wallet" validate:"required"`
	Points      int                    `json:"points" validate:"required,min=1"`
	Source      string                 `json:"source" validate:"required"`
	Description string                 `json:"description,omitempty"`
	Reference   string                 `json:"reference,omitempty"` // Transaction hash, invitation code, etc.
	SubnetID    string                 `json:"subnet_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DirectPointsResponse represents a direct points addition response
type DirectPointsResponse struct {
	Success      bool   `json:"success"`
	UserWallet   string `json:"user_wallet"`
	PointsAdded  int    `json:"points_added"`
	NewTotal     int    `json:"new_total"`
	Source       string `json:"source"`
	Message      string `json:"message,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// TelegramTaskRewardRequest represents a Telegram task reward request
type TelegramTaskRewardRequest struct {
	UserWallet string `json:"user_wallet" validate:"required"`
	TaskID     string `json:"task_id" validate:"required"`
	TelegramID string `json:"telegram_id" validate:"required"`
	SubnetID   string `json:"subnet_id" validate:"required"`
}

// TelegramTaskRewardResponse represents a Telegram task reward response
type TelegramTaskRewardResponse struct {
	Success      bool   `json:"success"`
	UserWallet   string `json:"user_wallet"`
	TaskID       string `json:"task_id"`
	SubnetID     string `json:"subnet_id"`
	PointsAdded  int    `json:"points_added"`
	NewTotal     int    `json:"new_total"`
	HasNFT       bool   `json:"has_nft"`
	Message      string `json:"message,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// TwitterPostRewardRequest represents a Twitter post reward request
type TwitterPostRewardRequest struct {
	UserWallet string `json:"user_wallet" validate:"required"`
	TaskID     string `json:"task_id" validate:"required"`
	PostURL    string `json:"post_url" validate:"required"` // URL of the posted tweet
}

// TwitterPostRewardResponse represents a Twitter post reward response
type TwitterPostRewardResponse struct {
	Success      bool   `json:"success"`
	UserWallet   string `json:"user_wallet"`
	TaskID       string `json:"task_id"`
	PostURL      string `json:"post_url"`
	PointsAdded  int    `json:"points_added"`
	NewTotal     int    `json:"new_total"`
	HasNFT       bool   `json:"has_nft"`
	Message      string `json:"message,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// UserCompletedTask represents a completed task by user
type UserCompletedTask struct {
	TaskID       string                 `json:"task_id"`
	TaskType     string                 `json:"task_type"`
	Status       string                 `json:"status"`
	CompletedAt  string                 `json:"completed_at"`
	PointsEarned int                    `json:"points_earned"`
	TaskDetails  map[string]interface{} `json:"task_details"`
	SubnetID     string                 `json:"subnet_id"`
	SubnetName   string                 `json:"subnet_name"`
	IsValid      bool                   `json:"is_valid"` // For Twitter tasks with modified links
}

// UserCompletedTasksResponse represents the response for user completed tasks query
type UserCompletedTasksResponse struct {
	Success bool                   `json:"success"`
	Data    UserCompletedTasksData `json:"data"`
	Message string                 `json:"message,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// UserCompletedTasksData represents the data structure for completed tasks
type UserCompletedTasksData struct {
	Tasks  []UserCompletedTask `json:"tasks"`
	Total  int                 `json:"total"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

// ChatTaskRewardRequest represents a Chat task reward request
type ChatTaskRewardRequest struct {
	UserWallet string `json:"user_wallet" validate:"required"`
	SubnetID   string `json:"subnet_id" validate:"required"`
}

// ChatTaskRewardResponse represents a Chat task reward response
type ChatTaskRewardResponse struct {
	Success      bool   `json:"success"`
	UserWallet   string `json:"user_wallet"`
	SubnetID     string `json:"subnet_id"`
	PointsAdded  int    `json:"points_added"`
	NewTotal     int    `json:"new_total"`
	HasNFT       bool   `json:"has_nft"`
	Message      string `json:"message,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}
