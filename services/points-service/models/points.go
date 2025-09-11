package models

import (
	"time"
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

// PointsDistributionResult points distribution result
type PointsDistributionResult struct {
	BatchID          string             `json:"batch_id"`
	TotalPoolPoints  int                `json:"total_pool_points"`  // Total points pool
	CreationPoints   int                `json:"creation_points"`    // Creation task points pool
	RetweetPoints    int                `json:"retweet_points"`     // Retweet task points pool
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
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// PointsConfig points configuration
type PointsConfig struct {
	TotalPoolPoints int     `json:"total_pool_points"` // Total points pool per round, default 100
	CreationRatio   float64 `json:"creation_ratio"`    // Creation task ratio, default 0.4
	RetweetRatio    float64 `json:"retweet_ratio"`     // Retweet task ratio, default 0.6
	HistoryLimit    int     `json:"history_limit"`     // History record limit, default 1000
}

// DefaultPointsConfig default points configuration
func DefaultPointsConfig() *PointsConfig {
	return &PointsConfig{
		TotalPoolPoints: 100,
		CreationRatio:   0.4,
		RetweetRatio:    0.6,
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
