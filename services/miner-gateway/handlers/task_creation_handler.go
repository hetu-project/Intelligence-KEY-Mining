package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/services"
)

// TaskCreationHandler handles task creation related requests
type TaskCreationHandler struct {
	taskService   *services.TaskService
	subnetService *services.SubnetService
}

// NewTaskCreationHandler creates a new task creation handler
func NewTaskCreationHandler(taskService *services.TaskService, subnetService *services.SubnetService) *TaskCreationHandler {
	return &TaskCreationHandler{
		taskService:   taskService,
		subnetService: subnetService,
	}
}

// parseIntParam parses an integer parameter with min/max validation
func parseIntParam(param string, min, max int) (int, error) {
	value, err := strconv.Atoi(param)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %s", param)
	}
	if value < min || value > max {
		return 0, fmt.Errorf("value %d out of range [%d, %d]", value, min, max)
	}
	return value, nil
}

// getTaskExpiryDays gets task expiry days from environment variable
// DEPRECATED: Now using deadline field from API request
func getTaskExpiryDays() int {
	expiryDaysStr := os.Getenv("TASK_EXPIRY_DAYS")
	if expiryDaysStr == "" {
		return 7 // Default 7 days
	}

	expiryDays, err := strconv.Atoi(expiryDaysStr)
	if err != nil || expiryDays <= 0 {
		return 7 // Default 7 days if invalid
	}

	return expiryDays
}

// CreateTwitterTask handles Twitter task creation
func (tch *TaskCreationHandler) CreateTwitterTask(c *gin.Context) {
	var req models.TaskCreationRequest

	// Bind request parameters
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Map API task type to internal task type
	var internalTaskType models.TaskType
	switch req.TaskType {
	case "twitter_retweet":
		internalTaskType = models.TwitterRetweetTask // Maps to "twitter_retweet"
	case "twitter_post":
		internalTaskType = models.TwitterPostTask // Maps to "twitter_post"
	case "task_creation":
		internalTaskType = models.TaskCreationTask // Maps to "task_creation"
	case "telegram_task":
		internalTaskType = models.TelegramTask // Maps to "telegram_task"
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid task type. Must be 'twitter_retweet', 'twitter_post', 'task_creation', or 'telegram_task'",
		})
		return
	}

	// Handle subnet creation/lookup
	var subnetID string
	if req.ProjectName != "" {
		subnetReq := &models.SubnetCreateRequest{
			Name:          req.ProjectName,
			Icon:          req.ProjectIcon,
			XURL:          req.XURL,    // New field for X/Twitter URL
			Website:       req.Website, // New field for official website
			CreatorWallet: req.UserWallet,
		}

		subnet, err := tch.subnetService.FindOrCreateSubnet(c.Request.Context(), subnetReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to handle subnet: " + err.Error(),
			})
			return
		}
		subnetID = subnet.ID
	}

	// Validate deadline from API request
	now := time.Now()
	if req.Deadline.Before(now) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Deadline must be in the future",
		})
		return
	}

	expiresAt := req.Deadline

	// Build task submission request with dynamic payload based on task type
	var payload map[string]interface{}

	switch internalTaskType {
	case models.TwitterRetweetTask:
		// Validate required Twitter fields
		if req.TwitterUsername == "" || req.TwitterLink == "" || req.TweetID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Twitter task requires twitter_username, twitter_link, and tweet_id",
			})
			return
		}
		payload = map[string]interface{}{
			"project_name":     req.ProjectName,
			"project_icon":     req.ProjectIcon,
			"description":      req.Description,
			"twitter_username": req.TwitterUsername,
			"twitter_link":     req.TwitterLink,
			"tweet_id":         req.TweetID,
			"subnet_id":        subnetID,
		}

	case models.TelegramTask:
		// Validate required Telegram fields
		if req.TelegramChannel == "" || req.ActionType == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Telegram task requires telegram_channel and action_type",
			})
			return
		}
		payload = map[string]interface{}{
			"project_name":     req.ProjectName,
			"project_icon":     req.ProjectIcon,
			"description":      req.Description,
			"telegram_channel": req.TelegramChannel,
			"action_type":      req.ActionType,
			"message_id":       req.MessageID,
			"telegram_link":    req.TelegramLink,
			"required_action":  req.RequiredAction,
			"subnet_id":        subnetID,
		}

	case models.TwitterPostTask:
		// Set default values for Twitter post fields if empty
		postID := req.PostID
		if postID == "" {
			postID = "default_post_" + fmt.Sprintf("%d", time.Now().Unix())
		}

		postLink := req.PostLink
		if postLink == "" {
			postLink = "https://x.com/default/status/" + postID
		}

		payload = map[string]interface{}{
			"project_name": req.ProjectName,
			"project_icon": req.ProjectIcon,
			"description":  req.Description,
			"post_id":      postID,
			"post_link":    postLink,
			"subnet_id":    subnetID,
		}

	case models.TaskCreationTask:
		// Task creation payload (existing logic)
		payload = map[string]interface{}{
			"project_name": req.ProjectName,
			"project_icon": req.ProjectIcon,
			"description":  req.Description,
			"subnet_id":    subnetID,
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Unsupported task type for payload generation",
		})
		return
	}

	taskReq := &models.TaskSubmitRequest{
		UserWallet: req.UserWallet,
		TaskType:   string(internalTaskType), // Use internal task type
		SubnetID:   subnetID,
		ExpiresAt:  &expiresAt,
		Payload:    payload,
	}

	// Submit task
	response, err := tch.taskService.SubmitTask(c.Request.Context(), taskReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create task: " + err.Error(),
		})
		return
	}

	// Build response
	taskCreationResponse := &models.TaskCreationResponse{
		Success:  response.Success,
		TaskID:   response.TaskID,
		Message:  response.Message,
		VLCValue: response.VLCValue,
	}

	if response.Success {
		c.JSON(http.StatusCreated, taskCreationResponse)
	} else {
		c.JSON(http.StatusBadRequest, taskCreationResponse)
	}
}

// GetTaskCreationStatus gets the status of a task creation
func (tch *TaskCreationHandler) GetTaskCreationStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Task ID is required",
		})
		return
	}

	// Get task status
	task, err := tch.taskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Task not found: " + err.Error(),
		})
		return
	}

	// Check if it's a task creation type
	if task.TaskType != models.TaskCreationTask {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Task is not a task creation type",
		})
		return
	}

	// Build response
	response := gin.H{
		"success":      true,
		"task_id":      task.ID,
		"status":       task.Status,
		"created_at":   task.CreatedAt,
		"updated_at":   task.UpdatedAt,
		"completed_at": task.CompletedAt,
		"payload":      task.Payload,
	}

	// Add VLC information
	if task.VLCClock != nil {
		response["vlc_value"] = task.VLCClock.GetValue(task.VLCClock.ProcessID)
		response["vlc_clock"] = task.VLCClock
	}

	// Add proof information
	if task.Proof != nil {
		response["proof"] = task.Proof
	}

	c.JSON(http.StatusOK, response)
}

// ListUserTaskCreations lists all task creations for a user
func (tch *TaskCreationHandler) ListUserTaskCreations(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "User wallet is required",
		})
		return
	}

	// Get query parameters
	limit := 50 // Default limit 50 records
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := parseIntParam(limitParam, 1, 100); err == nil {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if parsedOffset, err := parseIntParam(offsetParam, 0, 10000); err == nil {
			offset = parsedOffset
		}
	}

	// Get user's task creation records
	tasks, err := tch.taskService.GetUserTasksByType(c.Request.Context(), userWallet, models.TaskCreationTask, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get user task creations: " + err.Error(),
		})
		return
	}

	// Build response
	taskList := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		taskInfo := gin.H{
			"task_id":      task.ID,
			"status":       task.Status,
			"created_at":   task.CreatedAt,
			"updated_at":   task.UpdatedAt,
			"completed_at": task.CompletedAt,
			"payload":      task.Payload,
		}

		// Add VLC information
		if task.VLCClock != nil {
			taskInfo["vlc_value"] = task.VLCClock.GetValue(task.VLCClock.ProcessID)
		}

		// Add proof information
		if task.Proof != nil {
			taskInfo["verified"] = true
			taskInfo["verified_at"] = task.Proof.VerifiedAt
		} else {
			taskInfo["verified"] = false
		}

		taskList = append(taskList, taskInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tasks":  taskList,
			"total":  len(taskList),
			"limit":  limit,
			"offset": offset,
		},
	})
}

// UpdateTwitterLink updates the Twitter link for a retweet task
func (tch *TaskCreationHandler) UpdateTwitterLink(c *gin.Context) {
	var req struct {
		UserWallet     string `json:"user_wallet" binding:"required"`
		OldTweetID     string `json:"old_tweet_id" binding:"required"`
		NewTweetID     string `json:"new_tweet_id" binding:"required"`
		NewTwitterLink string `json:"new_twitter_link" binding:"required"`
	}

	// Bind request parameters
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate new tweet ID format (should be numeric and reasonable length)
	if err := validateTweetID(req.NewTweetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid tweet ID format: " + err.Error(),
		})
		return
	}

	// Optional: Validate that the tweet ID matches the link
	if extractedID, err := extractTweetIDFromLink(req.NewTwitterLink); err == nil {
		if extractedID != req.NewTweetID {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Tweet ID does not match the provided Twitter link",
			})
			return
		}
	}

	// Update the task
	err := tch.taskService.UpdateTwitterLink(c.Request.Context(), req.UserWallet, req.OldTweetID, req.NewTweetID, req.NewTwitterLink)
	if err != nil {
		// Handle different error types
		switch {
		case strings.Contains(err.Error(), "not found"):
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Task not found for the given user and tweet ID",
			})
		case strings.Contains(err.Error(), "permission denied"):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "Permission denied: only task creator can modify the task",
			})
		case strings.Contains(err.Error(), "processing"):
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": "Task is currently being processed, please try again later",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Failed to update Twitter link: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Twitter link updated successfully",
		"data": gin.H{
			"old_tweet_id":     req.OldTweetID,
			"new_tweet_id":     req.NewTweetID,
			"new_twitter_link": req.NewTwitterLink,
			"updated_at":       time.Now(),
		},
		"warning": "Users who have already retweeted the old link need to retweet the new link for verification",
	})
}

// extractTweetIDFromLink extracts tweet ID from Twitter/X URL
func extractTweetIDFromLink(twitterLink string) (string, error) {
	// Support both twitter.com and x.com domains
	// Pattern: https://twitter.com/username/status/1234567890
	// Pattern: https://x.com/username/status/1234567890

	// Remove trailing parameters and fragments
	if idx := strings.Index(twitterLink, "?"); idx != -1 {
		twitterLink = twitterLink[:idx]
	}
	if idx := strings.Index(twitterLink, "#"); idx != -1 {
		twitterLink = twitterLink[:idx]
	}

	// Extract tweet ID using regex or string manipulation
	parts := strings.Split(twitterLink, "/")
	if len(parts) < 6 {
		return "", fmt.Errorf("invalid Twitter URL format")
	}

	// Find "status" part and get the next element
	for i, part := range parts {
		if part == "status" && i+1 < len(parts) {
			tweetID := parts[i+1]
			// Validate tweet ID (should be numeric)
			if len(tweetID) < 10 || len(tweetID) > 20 {
				return "", fmt.Errorf("invalid tweet ID length")
			}
			// Check if it's numeric
			for _, char := range tweetID {
				if char < '0' || char > '9' {
					return "", fmt.Errorf("tweet ID must be numeric")
				}
			}
			return tweetID, nil
		}
	}

	return "", fmt.Errorf("could not extract tweet ID from URL")
}

// validateTweetID validates tweet ID format
func validateTweetID(tweetID string) error {
	// Check length (Twitter IDs are typically 10-20 digits)
	if len(tweetID) < 10 || len(tweetID) > 20 {
		return fmt.Errorf("invalid tweet ID length: must be between 10-20 characters")
	}

	// Check if it's numeric
	for _, char := range tweetID {
		if char < '0' || char > '9' {
			return fmt.Errorf("tweet ID must be numeric")
		}
	}

	return nil
}

// GetTaskCreationStats gets statistics for task creations
func (tch *TaskCreationHandler) GetTaskCreationStats(c *gin.Context) {
	userWallet := c.Query("user_wallet")

	// Get statistics information
	stats, err := tch.taskService.GetTaskTypeStats(c.Request.Context(), models.TaskCreationTask, userWallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get task creation stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
