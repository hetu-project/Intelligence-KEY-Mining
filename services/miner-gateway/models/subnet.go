package models

import (
	"time"
)

// Subnet represents a subnet (project) in the system
type Subnet struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Icon          string    `json:"icon" db:"icon"`
	XURL          string    `json:"x_url" db:"x_url"`     // Project X/Twitter URL
	Website       string    `json:"website" db:"website"` // Project official website
	CreatorWallet string    `json:"creator_wallet" db:"creator_wallet"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	Status        string    `json:"status" db:"status"`
}

// SubnetStats represents subnet statistics
type SubnetStats struct {
	SubnetID               string    `json:"subnet_id" db:"subnet_id"`
	SubnetName             string    `json:"subnet_name" db:"subnet_name"`
	SubnetIcon             string    `json:"subnet_icon" db:"subnet_icon"`
	CreatorWallet          string    `json:"creator_wallet" db:"creator_wallet"`
	SubnetCreatedAt        time.Time `json:"subnet_created_at" db:"subnet_created_at"`
	TotalTasks             int       `json:"total_tasks" db:"total_tasks"`
	CompletedTasks         int       `json:"completed_tasks" db:"completed_tasks"`
	UniqueUsers            int       `json:"unique_users" db:"unique_users"`
	TotalPointsDistributed int       `json:"total_points_distributed" db:"total_points_distributed"`
	TodayPointsDistributed int       `json:"today_points_distributed" db:"today_points_distributed"`
	TodayActiveUsers       int       `json:"today_active_users" db:"today_active_users"`
}

// UserTaskCompletion represents user task completion record
type UserTaskCompletion struct {
	ID           int64     `json:"id" db:"id"`
	UserWallet   string    `json:"user_wallet" db:"user_wallet"`
	TaskID       string    `json:"task_id" db:"task_id"`
	SubnetID     string    `json:"subnet_id" db:"subnet_id"`
	CompletedAt  time.Time `json:"completed_at" db:"completed_at"`
	VLCIncrement int       `json:"vlc_increment" db:"vlc_increment"`
	PointsEarned int       `json:"points_earned" db:"points_earned"`
}

// DailyDistributionLog represents daily distribution log
type DailyDistributionLog struct {
	ID                     int64      `json:"id" db:"id"`
	DistributionDate       string     `json:"distribution_date" db:"distribution_date"` // YYYY-MM-DD format
	Status                 string     `json:"status" db:"status"`                       // 'completed', 'failed', 'in_progress'
	TotalPointsDistributed int        `json:"total_points_distributed" db:"total_points_distributed"`
	TotalUsersAffected     int        `json:"total_users_affected" db:"total_users_affected"`
	TotalTasksProcessed    int        `json:"total_tasks_processed" db:"total_tasks_processed"`
	StartedAt              time.Time  `json:"started_at" db:"started_at"`
	CompletedAt            *time.Time `json:"completed_at" db:"completed_at"`
	ErrorMessage           string     `json:"error_message" db:"error_message"`
	Metadata               string     `json:"metadata" db:"metadata"` // JSON string
}

// NFTOwnershipCache represents NFT ownership cache
type NFTOwnershipCache struct {
	ID          int64     `json:"id" db:"id"`
	UserWallet  string    `json:"user_wallet" db:"user_wallet"`
	HasNFT      bool      `json:"has_nft" db:"has_nft"`
	CheckedAt   time.Time `json:"checked_at" db:"checked_at"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	APIResponse string    `json:"api_response" db:"api_response"` // JSON string
}

// InvitationReward represents invitation reward record
type InvitationReward struct {
	ID              int64     `json:"id" db:"id"`
	InviterWallet   string    `json:"inviter_wallet" db:"inviter_wallet"`
	InviteeWallet   string    `json:"invitee_wallet" db:"invitee_wallet"`
	RewardPoints    int       `json:"reward_points" db:"reward_points"`
	InviterHasNFT   bool      `json:"inviter_has_nft" db:"inviter_has_nft"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	PointsHistoryID *int64    `json:"points_history_id" db:"points_history_id"`
}

// SubnetCreateRequest represents subnet creation request
type SubnetCreateRequest struct {
	Name          string `json:"project_name" validate:"required,max=200"`
	Icon          string `json:"project_icon" validate:"url,max=500"`
	XURL          string `json:"x_url" validate:"omitempty,url,max=500"`   // Project X/Twitter URL
	Website       string `json:"website" validate:"omitempty,url,max=500"` // Project official website
	CreatorWallet string `json:"creator_wallet" validate:"required"`
}

// SubnetResponse represents subnet response
type SubnetResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Icon          string    `json:"icon"`
	XURL          string    `json:"x_url"`   // Project X/Twitter URL
	Website       string    `json:"website"` // Project official website
	CreatorWallet string    `json:"creator_wallet"`
	CreatedAt     time.Time `json:"created_at"`
	Status        string    `json:"status"`
}

// PointsSource constants for points history
const (
	PointsSourceTaskCreation     = "Task Creation"
	PointsSourceTwitterRetweet   = "Twitter Retweet Task"
	PointsSourceNFTPurchase      = "NFT Purchase Bonus"
	PointsSourceInvitationReward = "Invitation Reward"
	PointsSourceVLCDistribution  = "VLC Distribution"
)
