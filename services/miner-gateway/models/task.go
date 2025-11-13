package models

import (
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/vlc"
)

// TaskType represents different types of tasks
type TaskType string

const (
	TwitterRetweetTask TaskType = "twitter_retweet"
	TwitterPostTask    TaskType = "twitter_post"
	TwitterFollowTask  TaskType = "twitter_follow"
	DiscordMessageTask TaskType = "discord_message"
	EmailConfirmTask   TaskType = "email_confirm"
	// New task types
	TaskCreationTask   TaskType = "task_creation"
	TelegramTask       TaskType = "telegram_task"
	RegisterQRCodeTask TaskType = "register_qr_code"
	// Note: BatchVerificationTask removed - it's an operation, not a task type
	// Future extended task types
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	TaskSubmitted           TaskStatus = "SUBMITTED"
	TaskPendingVerification TaskStatus = "PENDING_VERIFICATION"
	TaskVerified            TaskStatus = "VERIFIED"
	TaskFailed              TaskStatus = "FAILED"
	TaskPendingReview       TaskStatus = "PENDING_REVIEW"
	TaskMinerOutputCreated  TaskStatus = "MINER_OUTPUT_CREATED"
	TaskVoted               TaskStatus = "VOTED"
	TaskConfirmed           TaskStatus = "CONFIRMED"
	TaskRejected            TaskStatus = "REJECTED"
	TaskExpired             TaskStatus = "EXPIRED"
)

// Task represents a user task in the system
type Task struct {
	ID          string                 `json:"id" db:"id"`
	UserWallet  string                 `json:"user_wallet" db:"user_wallet"`
	TaskType    TaskType               `json:"task_type" db:"task_type"`
	Status      TaskStatus             `json:"status" db:"status"`
	Payload     map[string]interface{} `json:"payload" db:"payload"`
	Proof       *TaskProof             `json:"proof,omitempty" db:"proof"`
	Attempts    int                    `json:"attempts" db:"attempts"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	VLCClock    *vlc.VectorClock       `json:"vlc_clock,omitempty"`
	EventID     string                 `json:"event_id,omitempty" db:"event_id"`
	SubnetID    string                 `json:"subnet_id,omitempty" db:"subnet_id"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty" db:"expires_at"`
}

// TaskProof represents verification proof from middle layer
type TaskProof struct {
	Provider       string                 `json:"provider"`        // "twitter-middle-layer"
	VerifiedAt     time.Time              `json:"verified_at"`     // Verification time
	Evidence       map[string]interface{} `json:"evidence"`        // Evidence snapshot
	VerificationID string                 `json:"verification_id"` // Middleware verification ID
	Signature      string                 `json:"signature"`       // Middleware signature
}

// TwitterRetweetPayload represents Twitter retweet task payload
type TwitterRetweetPayload struct {
	TweetID    string `json:"tweet_id"`    // Tweet ID to retweet
	TwitterID  string `json:"twitter_id"`  // User Twitter ID
	RetweetURL string `json:"retweet_url"` // Retweet URL
}

// TaskCreationPayload represents task creation payload
type TaskCreationPayload struct {
	ProjectName     string `json:"project_name"`     // Project name
	ProjectIcon     string `json:"project_icon"`     // Project icon URL
	Description     string `json:"description"`      // Task description
	TwitterUsername string `json:"twitter_username"` // Twitter username
	TwitterLink     string `json:"twitter_link"`     // Twitter link
	TweetID         string `json:"tweet_id"`         // Tweet ID
}

// TelegramTaskPayload represents Telegram task payload
type TelegramTaskPayload struct {
	ProjectName     string `json:"project_name"`     // Project name
	ProjectIcon     string `json:"project_icon"`     // Project icon URL
	Description     string `json:"description"`      // Task description
	TelegramChannel string `json:"telegram_channel"` // Telegram channel/group (@channel or https://t.me/channel)
	ActionType      string `json:"action_type"`      // Action type (join_channel, share_message, etc.)
	MessageID       string `json:"message_id"`       // Message ID (optional)
	TelegramLink    string `json:"telegram_link"`    // Specific Telegram link
	RequiredAction  string `json:"required_action"`  // Required action description
}

// TwitterPostTaskPayload represents Twitter post task payload
type TwitterPostTaskPayload struct {
	ProjectName string `json:"project_name"` // Project name
	ProjectIcon string `json:"project_icon"` // Project icon URL
	Description string `json:"description"`  // Task description
	PostID      string `json:"post_id"`      // Required post ID
	PostLink    string `json:"post_link"`    // Required post link
}

// RegisterQRCodeTaskPayload represents Register with QR code task payload
type RegisterQRCodeTaskPayload struct {
	Title       string `json:"title"`        // Title (max 15 words)
	Description string `json:"description"`  // Description (max 20 words)
	Detail      string `json:"detail"`       // Detail (max 200 words)
	RewardBadge string `json:"reward_badge"` // Reward badge image URL
	ProjectName string `json:"project_name"` // Project name
	ProjectIcon string `json:"project_icon"` // Project icon URL
	SubnetID    string `json:"subnet_id"`    // Subnet ID
}

// BatchVerificationPayload represents batch verification payload
type BatchVerificationPayload struct {
	StartTime string `json:"start_time"` // Verification start time
	EndTime   string `json:"end_time"`   // Verification end time
	BatchSize int    `json:"batch_size"` // Batch size
}

// MinerOutput represents the output generated by miner after task verification
type MinerOutput struct {
	TaskID    string                 `json:"task_id"`
	TaskType  string                 `json:"task_type"`
	MinerID   string                 `json:"miner_id"`
	EventID   string                 `json:"event_id"`
	VLCClock  *vlc.VectorClock       `json:"vlc_clock"`
	Payload   map[string]interface{} `json:"payload"`
	Proof     *TaskProof             `json:"proof"`
	Signature string                 `json:"signature"` // Miner signature
	Timestamp time.Time              `json:"timestamp"`
}

// API request response structures

// TaskCreationRequest represents task creation request
type TaskCreationRequest struct {
	UserWallet  string    `json:"user_wallet" binding:"required"`
	TaskType    string    `json:"task_type" binding:"required"` // API task type string
	ProjectName string    `json:"project_name" binding:"required"`
	ProjectIcon string    `json:"project_icon"`
	XURL        string    `json:"x_url"`     // Project X/Twitter URL
	Website     string    `json:"website"`   // Project official website
	TVL         float64   `json:"tvl"`       // Total Value Locked in USD
	Valuation   float64   `json:"valuation"` // Project Valuation in USD
	Description string    `json:"description" binding:"required"`
	Deadline    time.Time `json:"deadline" binding:"required"` // Task deadline

	// Twitter task fields (required when task_type = "twitter_retweet")
	TwitterUsername string `json:"twitter_username"`
	TwitterLink     string `json:"twitter_link"`
	TweetID         string `json:"tweet_id"`

	// Telegram task fields (required when task_type = "telegram_task")
	TelegramChannel string `json:"telegram_channel"`
	ActionType      string `json:"action_type"`
	MessageID       string `json:"message_id"`
	TelegramLink    string `json:"telegram_link"`
	RequiredAction  string `json:"required_action"`

	// Twitter post task fields (required when task_type = "twitter_post")
	PostID   string `json:"post_id"`   // Required post ID
	PostLink string `json:"post_link"` // Required post link

	// Twitter follow task fields (required when task_type = "twitter_follow")
	Title               string `json:"title"`                 // Required: follow task title
	FollowAccountID     string `json:"follow_account_id"`     // Optional but recommended
	FollowAccountHandle string `json:"follow_account_handle"` // Optional
	FollowAccountURL    string `json:"follow_account_url"`    // Optional

	// Register QR code task fields (required when task_type = "register_qr_code")
	Detail      string `json:"detail"`       // Detail (max 200 words)
	RewardBadge string `json:"reward_badge"` // Reward badge image URL
}

// TaskCreationResponse represents task creation response
type TaskCreationResponse struct {
	Success  bool   `json:"success"`
	TaskID   string `json:"task_id,omitempty"`
	SubnetID string `json:"subnet_id,omitempty"` // Subnet ID
	Message  string `json:"message"`
	VLCValue int    `json:"vlc_value,omitempty"` // Current VLC value
}

// BatchVerificationInfo represents batch verification info
type BatchVerificationInfo struct {
	TotalTasks      int `json:"total_tasks"`
	VerifiedTasks   int `json:"verified_tasks"`
	UnverifiedTasks int `json:"unverified_tasks"`
	VLCIncrement    int `json:"vlc_increment"`
}

// BatchVerificationRound represents a batch verification round for PoCW consensus
type BatchVerificationRound struct {
	RoundID             string                 `json:"round_id"`
	StartTime           time.Time              `json:"start_time"`
	EndTime             *time.Time             `json:"end_time,omitempty"`
	TaskType            TaskType               `json:"task_type"`
	TotalTasks          int                    `json:"total_tasks"`
	VerifiedTasks       int                    `json:"verified_tasks"`
	FailedTasks         int                    `json:"failed_tasks"`
	Tasks               []*Task                `json:"tasks"`
	VerificationSummary map[string]interface{} `json:"verification_summary"`
	VLCBefore           *vlc.VectorClock       `json:"vlc_before"`
	VLCAfter            *vlc.VectorClock       `json:"vlc_after"`
	Status              string                 `json:"status"` // "processing", "completed", "failed"
}

// BatchRoundResult represents the result of a batch verification round
type BatchRoundResult struct {
	RoundID      string                 `json:"round_id"`
	Success      bool                   `json:"success"`
	TaskResults  map[string]TaskResult  `json:"task_results"` // task_id -> result
	TotalTasks   int                    `json:"total_tasks"`
	SuccessTasks int                    `json:"success_tasks"`
	FailedTasks  int                    `json:"failed_tasks"`
	VLCIncrement int                    `json:"vlc_increment"`
	Summary      map[string]interface{} `json:"summary"`
}

// TaskResult represents individual task verification result within a batch round
type TaskResult struct {
	TaskID   string                 `json:"task_id"`
	Success  bool                   `json:"success"`
	Reason   string                 `json:"reason,omitempty"`
	Evidence map[string]interface{} `json:"evidence,omitempty"`
}
