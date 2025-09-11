package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/services"
)

// TaskCreationHandler handles task creation related requests
type TaskCreationHandler struct {
	taskService *services.TaskService
}

// NewTaskCreationHandler creates a new task creation handler
func NewTaskCreationHandler(taskService *services.TaskService) *TaskCreationHandler {
	return &TaskCreationHandler{
		taskService: taskService,
	}
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

	// Build task submission request
	taskReq := &models.TaskSubmitRequest{
		UserWallet: req.UserWallet,
		TaskType:   string(models.TaskCreationTask),
		Payload: map[string]interface{}{
			"project_name":     req.ProjectName,
			"project_icon":     req.ProjectIcon,
			"description":      req.Description,
			"twitter_username": req.TwitterUsername,
			"twitter_link":     req.TwitterLink,
			"tweet_id":         req.TweetID,
		},
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
