package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/services"
)

// PointsHandler handles points-related operations
type PointsHandler struct {
	pointsService *services.PointsService
}

// NewPointsHandler creates a new points handler
func NewPointsHandler(pointsService *services.PointsService) *PointsHandler {
	return &PointsHandler{
		pointsService: pointsService,
	}
}

// DistributePoints distributes points to users
// POST /api/v1/points/distribute
func (ph *PointsHandler) DistributePoints(c *gin.Context) {
	var req models.PointsDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate request
	if err := ph.validateDistributionRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Execute points distribution
	result, err := ph.pointsService.DistributePoints(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to distribute points",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   result,
	})
}

// GetUserPoints retrieves user points
// GET /api/v1/points/user/:wallet_address
func (ph *PointsHandler) GetUserPoints(c *gin.Context) {
	walletAddress := c.Param("wallet_address")
	if walletAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Wallet address is required",
		})
		return
	}

	points, err := ph.pointsService.GetUserPoints(c.Request.Context(), walletAddress)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Failed to get user points",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"wallet_address": walletAddress,
			"total_points":   points,
		},
	})
}

// GetPointsHistory retrieves user points history
// GET /api/v1/points/history/:wallet_address?limit=50
func (ph *PointsHandler) GetPointsHistory(c *gin.Context) {
	walletAddress := c.Param("wallet_address")
	if walletAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Wallet address is required",
		})
		return
	}

	// Parse limit parameter
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	history, err := ph.pointsService.GetPointsHistory(c.Request.Context(), walletAddress, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get points history",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"wallet_address": walletAddress,
			"history":        history,
			"count":          len(history),
		},
	})
}

// GetPointsStats retrieves points statistics
// GET /api/v1/points/stats
func (ph *PointsHandler) GetPointsStats(c *gin.Context) {
	stats, err := ph.pointsService.GetPointsStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get points statistics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// GetConfig retrieves points configuration
// GET /api/v1/points/config
func (ph *PointsHandler) GetConfig(c *gin.Context) {
	config := ph.pointsService.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   config,
	})
}

// UpdateConfig updates points configuration
// PUT /api/v1/points/config
func (ph *PointsHandler) UpdateConfig(c *gin.Context) {
	var config models.PointsConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid configuration format",
			"details": err.Error(),
		})
		return
	}

	// Validate configuration
	if err := ph.validateConfig(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid configuration",
			"details": err.Error(),
		})
		return
	}

	ph.pointsService.UpdateConfig(&config)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Configuration updated successfully",
		"data":    config,
	})
}

// TestDistribution tests points distribution (for development)
// POST /api/v1/points/test
func (ph *PointsHandler) TestDistribution(c *gin.Context) {
	// Create test data
	testReq := models.PointsDistributionRequest{
		BatchID:     fmt.Sprintf("test_%d", c.Request.Context().Value("timestamp")),
		TriggerType: "validator_voting",
		Tasks: []models.TaskVLC{
			{UserWallet: "0x1234...abcd", TaskType: "creation", VLCValue: 2, TaskID: "task_1"},
			{UserWallet: "0x5678...efgh", TaskType: "creation", VLCValue: 1, TaskID: "task_2"},
			{UserWallet: "0x1234...abcd", TaskType: "retweet", VLCValue: 3, TaskID: "task_3"},
			{UserWallet: "0x9999...1111", TaskType: "retweet", VLCValue: 1, TaskID: "task_4"},
		},
	}

	result, err := ph.pointsService.DistributePoints(c.Request.Context(), &testReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Test distribution failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Test distribution completed",
		"data":    result,
	})
}

// validateDistributionRequest validates distribution request
func (ph *PointsHandler) validateDistributionRequest(req *models.PointsDistributionRequest) error {
	if req.BatchID == "" {
		return fmt.Errorf("batch_id is required")
	}

	if len(req.Tasks) == 0 {
		return fmt.Errorf("at least one task is required")
	}

	for i, task := range req.Tasks {
		if task.UserWallet == "" {
			return fmt.Errorf("task[%d]: user_wallet is required", i)
		}
		if task.TaskType != "creation" && task.TaskType != "retweet" {
			return fmt.Errorf("task[%d]: task_type must be 'creation' or 'retweet'", i)
		}
		if task.VLCValue < 0 {
			return fmt.Errorf("task[%d]: vlc_value cannot be negative", i)
		}
	}

	return nil
}

// validateConfig validates configuration
func (ph *PointsHandler) validateConfig(config *models.PointsConfig) error {
	if config.TotalPoolPoints <= 0 {
		return fmt.Errorf("total_pool_points must be positive")
	}

	if config.CreationRatio < 0 || config.CreationRatio > 1 {
		return fmt.Errorf("creation_ratio must be between 0 and 1")
	}

	if config.RetweetRatio < 0 || config.RetweetRatio > 1 {
		return fmt.Errorf("retweet_ratio must be between 0 and 1")
	}

	if config.CreationRatio+config.RetweetRatio != 1.0 {
		return fmt.Errorf("creation_ratio + retweet_ratio must equal 1.0")
	}

	if config.HistoryLimit <= 0 {
		return fmt.Errorf("history_limit must be positive")
	}

	return nil
}

// RegisterRoutes registers HTTP routes
func (ph *PointsHandler) RegisterRoutes(router *gin.RouterGroup) {
	points := router.Group("/points")
	{
		// Points distribution
		points.POST("/distribute", ph.DistributePoints)

		// User points queries
		points.GET("/user/:wallet_address", ph.GetUserPoints)
		points.GET("/history/:wallet_address", ph.GetPointsHistory)

		// System statistics
		points.GET("/stats", ph.GetPointsStats)

		// Configuration management
		points.GET("/config", ph.GetConfig)
		points.PUT("/config", ph.UpdateConfig)

		// Test endpoints
		points.POST("/test", ph.TestDistribution)
	}
}
