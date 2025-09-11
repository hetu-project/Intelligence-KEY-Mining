package models

// TaskSubmitRequest represents a task submission request
type TaskSubmitRequest struct {
	UserWallet string                 `json:"user_wallet" binding:"required"`
	TaskType   string                 `json:"task_type" binding:"required"`
	Payload    map[string]interface{} `json:"payload" binding:"required"`
}

// TaskSubmitResponse represents a task submission response
type TaskSubmitResponse struct {
	Success  bool   `json:"success"`
	TaskID   string `json:"task_id,omitempty"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
	VLCValue int    `json:"vlc_value,omitempty"`
}

// ErrorResponse represents a generic error response
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}
