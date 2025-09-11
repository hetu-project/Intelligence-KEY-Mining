package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/services"
)

// BatchVerificationHandler handles batch verification related requests
type BatchVerificationHandler struct {
	taskService   *services.TaskService
	batchVerifier *services.BatchVerifier
}

// NewBatchVerificationHandler creates a new batch verification handler
func NewBatchVerificationHandler(taskService *services.TaskService, batchVerifier *services.BatchVerifier) *BatchVerificationHandler {
	return &BatchVerificationHandler{
		taskService:   taskService,
		batchVerifier: batchVerifier,
	}
}

// BatchVerifyTasks handles batch verification request
func (bvh *BatchVerificationHandler) BatchVerifyTasks(c *gin.Context) {
	var req struct {
		UserWallet string `json:"user_wallet" binding:"required"`
		StartTime  string `json:"start_time" binding:"required"`
		EndTime    string `json:"end_time" binding:"required"`
		Tasks      []struct {
			TweetID   string `json:"tweet_id" binding:"required"`
			TwitterID string `json:"twitter_id" binding:"required"`
		} `json:"tasks" binding:"required"`
	}

	// Bind request parameters
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate time format
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid start_time format, expected RFC3339",
		})
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid end_time format, expected RFC3339",
		})
		return
	}

	// Validate time range
	if endTime.Before(startTime) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "end_time must be after start_time",
		})
		return
	}

	// Validate task count
	if len(req.Tasks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "At least one task is required",
		})
		return
	}

	if len(req.Tasks) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Too many tasks, maximum 100 allowed",
		})
		return
	}

	// Build task submission request
	taskReq := &models.TaskSubmitRequest{
		UserWallet: req.UserWallet,
		TaskType:   string(models.BatchVerificationTask),
		Payload: map[string]interface{}{
			"start_time": req.StartTime,
			"end_time":   req.EndTime,
			"batch_size": len(req.Tasks),
			"tasks":      req.Tasks,
		},
	}

	// Submit task
	response, err := bvh.taskService.SubmitTask(c.Request.Context(), taskReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to submit batch verification: " + err.Error(),
		})
		return
	}

	// If task submission successful, immediately submit to async verifier for processing
	if response.Success && bvh.batchVerifier != nil {
		// Get the just created task
		task, err := bvh.taskService.GetTask(c.Request.Context(), response.TaskID)
		if err == nil {
			// Submit to async verifier
			if err := bvh.batchVerifier.SubmitTask(task); err != nil {
				// Log error but don't affect response, as task has been created successfully
				// Can retry through other means
				// TODO: Add retry mechanism or error handling
			}
		}
	}

	// Build response
	batchResponse := gin.H{
		"success":      response.Success,
		"task_id":      response.TaskID,
		"message":      response.Message,
		"batch_size":   len(req.Tasks),
		"start_time":   req.StartTime,
		"end_time":     req.EndTime,
		"submitted_at": time.Now(),
	}

	if response.Success {
		c.JSON(http.StatusCreated, batchResponse)
	} else {
		c.JSON(http.StatusBadRequest, batchResponse)
	}
}

// GetBatchVerificationStatus gets the status of a batch verification
func (bvh *BatchVerificationHandler) GetBatchVerificationStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Task ID is required",
		})
		return
	}

	// Get task status
	task, err := bvh.taskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Task not found: " + err.Error(),
		})
		return
	}

	// Check if it's batch verification type
	if task.TaskType != models.BatchVerificationTask {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Task is not a batch verification type",
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
		response["vlc_increment"] = task.VLCClock.GetValue(task.VLCClock.ProcessID)
		response["vlc_clock"] = task.VLCClock
	}

	// Add batch verification results
	if task.Proof != nil {
		response["proof"] = task.Proof

		// Parse batch verification results
		if evidence, ok := task.Proof.Evidence["results"].([]interface{}); ok {
			verifiedCount := 0
			totalCount := len(evidence)

			for _, result := range evidence {
				if resultMap, ok := result.(map[string]interface{}); ok {
					if verified, ok := resultMap["verified"].(bool); ok && verified {
						verifiedCount++
					}
				}
			}

			response["batch_info"] = models.BatchVerificationInfo{
				TotalTasks:      totalCount,
				VerifiedTasks:   verifiedCount,
				UnverifiedTasks: totalCount - verifiedCount,
				VLCIncrement:    task.VLCClock.GetValue(task.VLCClock.ProcessID),
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// ListBatchVerifications lists batch verifications for a user
func (bvh *BatchVerificationHandler) ListBatchVerifications(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "User wallet is required",
		})
		return
	}

	// Get query parameters
	limit := 20 // Default limit 20 items
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := parseIntParam(limitParam, 1, 50); err == nil {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if parsedOffset, err := parseIntParam(offsetParam, 0, 10000); err == nil {
			offset = parsedOffset
		}
	}

	// Get user's batch verification records
	tasks, err := bvh.taskService.GetUserTasksByType(c.Request.Context(), userWallet, models.BatchVerificationTask, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get batch verifications: " + err.Error(),
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
		}

		// Extract batch information from payload
		if payload := task.Payload; payload != nil {
			if startTime, ok := payload["start_time"].(string); ok {
				taskInfo["start_time"] = startTime
			}
			if endTime, ok := payload["end_time"].(string); ok {
				taskInfo["end_time"] = endTime
			}
			if batchSize, ok := payload["batch_size"].(float64); ok {
				taskInfo["batch_size"] = int(batchSize)
			}
		}

		// Add VLC information
		if task.VLCClock != nil {
			taskInfo["vlc_increment"] = task.VLCClock.GetValue(task.VLCClock.ProcessID)
		}

		// Add verification result information
		if task.Proof != nil {
			taskInfo["verified"] = true
			taskInfo["verified_at"] = task.Proof.VerifiedAt

			// Parse batch verification result statistics
			if evidence, ok := task.Proof.Evidence["results"].([]interface{}); ok {
				verifiedCount := 0
				totalCount := len(evidence)

				for _, result := range evidence {
					if resultMap, ok := result.(map[string]interface{}); ok {
						if verified, ok := resultMap["verified"].(bool); ok && verified {
							verifiedCount++
						}
					}
				}

				taskInfo["batch_info"] = models.BatchVerificationInfo{
					TotalTasks:      totalCount,
					VerifiedTasks:   verifiedCount,
					UnverifiedTasks: totalCount - verifiedCount,
				}
			}
		} else {
			taskInfo["verified"] = false
		}

		taskList = append(taskList, taskInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"batch_verifications": taskList,
			"total":               len(taskList),
			"limit":               limit,
			"offset":              offset,
		},
	})
}

// GetBatchVerificationStats gets statistics for batch verifications
func (bvh *BatchVerificationHandler) GetBatchVerificationStats(c *gin.Context) {
	userWallet := c.Query("user_wallet")

	// Get statistics information
	stats, err := bvh.taskService.GetTaskTypeStats(c.Request.Context(), models.BatchVerificationTask, userWallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get batch verification stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// parseIntParam parses integer parameter with validation
func parseIntParam(param string, min, max int) (int, error) {
	value, err := strconv.Atoi(param)
	if err != nil {
		return 0, err
	}
	if value < min {
		value = min
	}
	if value > max {
		value = max
	}
	return value, nil
}
