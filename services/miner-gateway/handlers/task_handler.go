package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/services"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	taskService *services.TaskService
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(taskService *services.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
	}
}

// SubmitTask handles task submission
func (h *TaskHandler) SubmitTask(c *gin.Context) {
	var req models.TaskSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.UserWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "user_wallet is required",
		})
		return
	}

	if req.TaskType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "task_type is required",
		})
		return
	}

	if req.Payload == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "payload is required",
		})
		return
	}

	// Submit task
	response, err := h.taskService.SubmitTask(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	if !response.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   response.Error,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetTaskStatus handles task status queries
func (h *TaskHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "task ID is required",
		})
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Task not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    task,
	})
}

// GetUserTasks handles user task history queries
func (h *TaskHandler) GetUserTasks(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "user wallet is required",
		})
		return
	}

	// Pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	tasks, total, err := h.taskService.GetUserTasks(c.Request.Context(), userWallet, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tasks": tasks,
			"pagination": gin.H{
				"page":  page,
				"limit": limit,
				"total": total,
			},
		},
	})
}
