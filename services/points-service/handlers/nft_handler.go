package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/services"
)

// NFTHandler handles NFT-related HTTP requests
type NFTHandler struct {
	pointsService *services.PointsService
	nftService    *services.NFTService
}

// NewNFTHandler creates a new NFT handler
func NewNFTHandler(pointsService *services.PointsService, nftService *services.NFTService) *NFTHandler {
	return &NFTHandler{
		pointsService: pointsService,
		nftService:    nftService,
	}
}

// NFTPurchaseRequest represents NFT purchase bonus request
type NFTPurchaseRequest struct {
	UserWallet  string `json:"user_wallet" validate:"required"`
	NFTContract string `json:"nft_contract,omitempty"`
	TokenID     string `json:"token_id,omitempty"`
	PurchaseTx  string `json:"purchase_tx,omitempty"`
	BonusPoints int    `json:"bonus_points,omitempty"` // Optional, defaults to 5
}

// InvitationRewardRequest represents invitation reward request
type InvitationRewardRequest struct {
	InviterWallet  string `json:"inviter_wallet" validate:"required"`
	InviteeWallet  string `json:"invitee_wallet" validate:"required"`
	InvitationCode string `json:"invitation_code,omitempty"` // Optional, for duplicate prevention
}

// HandleNFTPurchaseBonus handles NFT purchase bonus points
func (nh *NFTHandler) HandleNFTPurchaseBonus(c *gin.Context) {
	var req NFTPurchaseRequest
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

	// Set default bonus points if not specified
	bonusPoints := req.BonusPoints
	if bonusPoints <= 0 {
		bonusPoints = 5 // Default NFT purchase bonus
	}

	// Add points to user
	err := nh.pointsService.AddDirectPoints(c.Request.Context(), &models.DirectPointsRequest{
		UserWallet:  req.UserWallet,
		Points:      bonusPoints,
		Source:      models.PointsSourceNFTPurchase,
		Description: "NFT Purchase Bonus",
		Reference:   req.PurchaseTx,
		Metadata: map[string]interface{}{
			"nft_contract": req.NFTContract,
			"token_id":     req.TokenID,
			"purchase_tx":  req.PurchaseTx,
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add NFT purchase bonus: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "NFT purchase bonus added successfully",
		"data": gin.H{
			"user_wallet":  req.UserWallet,
			"bonus_points": bonusPoints,
			"source":       models.PointsSourceNFTPurchase,
			"nft_contract": req.NFTContract,
			"token_id":     req.TokenID,
		},
	})
}

// HandleInvitationReward handles invitation reward points
func (nh *NFTHandler) HandleInvitationReward(c *gin.Context) {
	var req InvitationRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.InviterWallet == "" || req.InviteeWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Both inviter_wallet and invitee_wallet are required",
		})
		return
	}

	// Check if inviter has NFT for bonus calculation
	hasNFT, err := nh.nftService.CheckUserNFTOwnership(c.Request.Context(), req.InviterWallet)
	if err != nil {
		// Log error but continue with default points
		hasNFT = false
	}

	// Calculate reward points
	rewardPoints := 1 // Base invitation reward
	if hasNFT {
		rewardPoints = 2 // Double points for NFT holders
	}

	// Add points to inviter
	err = nh.pointsService.AddDirectPoints(c.Request.Context(), &models.DirectPointsRequest{
		UserWallet:  req.InviterWallet,
		Points:      rewardPoints,
		Source:      models.PointsSourceInvitationReward,
		Description: "Invitation Reward",
		Reference:   req.InvitationCode,
		Metadata: map[string]interface{}{
			"invitee_wallet":  req.InviteeWallet,
			"invitation_code": req.InvitationCode,
			"inviter_has_nft": hasNFT,
			"base_reward":     1,
			"nft_multiplier":  hasNFT,
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add invitation reward: " + err.Error(),
		})
		return
	}

	// Record invitation relationship (optional, for analytics)
	// This could be implemented in a separate service

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invitation reward added successfully",
		"data": gin.H{
			"inviter_wallet":  req.InviterWallet,
			"invitee_wallet":  req.InviteeWallet,
			"reward_points":   rewardPoints,
			"inviter_has_nft": hasNFT,
			"source":          models.PointsSourceInvitationReward,
		},
	})
}

// CheckNFTOwnership checks NFT ownership for a user
func (nh *NFTHandler) CheckNFTOwnership(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "wallet address is required",
		})
		return
	}

	hasNFT, err := nh.nftService.CheckUserNFTOwnership(c.Request.Context(), userWallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check NFT ownership: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_wallet": userWallet,
			"has_nft":     hasNFT,
		},
	})
}

// GetNFTCacheStats gets NFT cache statistics
func (nh *NFTHandler) GetNFTCacheStats(c *gin.Context) {
	stats, err := nh.nftService.GetNFTCacheStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get NFT cache stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// CleanExpiredNFTCache cleans expired NFT cache entries
func (nh *NFTHandler) CleanExpiredNFTCache(c *gin.Context) {
	deletedCount, err := nh.nftService.CleanExpiredCache(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to clean expired cache: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Expired cache entries cleaned successfully",
		"data": gin.H{
			"deleted_count": deletedCount,
		},
	})
}

// RegisterRoutes registers HTTP routes for NFT operations
func (nh *NFTHandler) RegisterRoutes(router *gin.RouterGroup) {
	points := router.Group("/points")
	{
		// NFT and invitation reward endpoints
		points.POST("/nft-purchase", nh.HandleNFTPurchaseBonus)
		points.POST("/invitation-reward", nh.HandleInvitationReward)
		points.POST("/telegram-task-reward", nh.HandleTelegramTaskReward)

		// User completed tasks query
		points.GET("/completed-tasks/:wallet", nh.GetUserCompletedTasks)

		// Twitter post task reward
		points.POST("/twitter-post-reward", nh.HandleTwitterPostReward)
	}

	nft := router.Group("/nft")
	{
		// NFT ownership check
		nft.GET("/check/:wallet", nh.CheckNFTOwnership)

		// NFT cache management (admin endpoints)
		nft.GET("/cache/stats", nh.GetNFTCacheStats)
		nft.POST("/cache/clean", nh.CleanExpiredNFTCache)
	}
}

// HandleTelegramTaskReward handles Telegram task reward points
func (nh *NFTHandler) HandleTelegramTaskReward(c *gin.Context) {
	var req models.TelegramTaskRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.UserWallet == "" || req.TaskID == "" || req.TelegramID == "" || req.SubnetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "user_wallet, task_id, telegram_id, and subnet_id are required",
		})
		return
	}

	// Check if user has already completed a Telegram task for this subnet today
	today := time.Now().Format("2006-01-02")
	completed, err := nh.checkUserTelegramTaskToday(c.Request.Context(), req.UserWallet, req.SubnetID, today)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check daily completion status: " + err.Error(),
		})
		return
	}
	if completed {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"error":   "User has already completed a Telegram task for this subnet today",
		})
		return
	}

	// Check if user has NFT for bonus calculation
	hasNFT, err := nh.nftService.CheckUserNFTOwnership(c.Request.Context(), req.UserWallet)
	if err != nil {
		// Log error but continue with default points
		hasNFT = false
	}

	// Calculate reward points
	rewardPoints := 1 // Base Telegram task reward
	if hasNFT {
		rewardPoints = 2 // Double points for NFT holders
	}

	// Add points to user
	err = nh.pointsService.AddDirectPoints(c.Request.Context(), &models.DirectPointsRequest{
		UserWallet:  req.UserWallet,
		Points:      rewardPoints,
		Source:      models.PointsSourceTelegramTask,
		Description: "Telegram Task Reward",
		Reference:   req.TaskID,
		Metadata: map[string]interface{}{
			"task_id":        req.TaskID,
			"subnet_id":      req.SubnetID,
			"telegram_id":    req.TelegramID,
			"has_nft":        hasNFT,
			"base_reward":    1,
			"nft_multiplier": hasNFT,
			"reward_date":    today,
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add Telegram task reward: " + err.Error(),
		})
		return
	}

	// TODO: Create user_task_completions record for statistics
	// This should be done to ensure the task appears in subnet statistics
	// For now, we'll add a note that this needs to be implemented

	// Get updated user total points (optional, for response)
	// This could be optimized by returning the new total from AddDirectPoints
	newTotal := 0 // You might want to query this from the database

	c.JSON(http.StatusOK, &models.TelegramTaskRewardResponse{
		Success:     true,
		UserWallet:  req.UserWallet,
		TaskID:      req.TaskID,
		SubnetID:    req.SubnetID,
		PointsAdded: rewardPoints,
		NewTotal:    newTotal,
		HasNFT:      hasNFT,
		Message:     "Telegram task reward added successfully",
	})
}

// checkUserTelegramTaskToday checks if user has completed a Telegram task for the subnet today
func (nh *NFTHandler) checkUserTelegramTaskToday(ctx context.Context, userWallet, subnetID, date string) (bool, error) {
	// Delegate to pointsService to handle the database check
	return nh.pointsService.CheckUserTelegramTaskToday(ctx, userWallet, subnetID, date)
}

// GetUserCompletedTasks handles user completed tasks query
func (nh *NFTHandler) GetUserCompletedTasks(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "wallet address is required",
		})
		return
	}

	// Get query parameters
	taskType := c.DefaultQuery("task_type", "all")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	// Parse limit and offset
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	offset := 0
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	// Get completed tasks
	tasks, total, err := nh.pointsService.GetUserCompletedTasks(c.Request.Context(), userWallet, taskType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get completed tasks: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, &models.UserCompletedTasksResponse{
		Success: true,
		Data: models.UserCompletedTasksData{
			Tasks:  tasks,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// HandleTwitterPostReward handles Twitter post task reward points
func (nh *NFTHandler) HandleTwitterPostReward(c *gin.Context) {
	var req models.TwitterPostRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.UserWallet == "" || req.TaskID == "" || req.PostURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "user_wallet, task_id, and post_url are required",
		})
		return
	}

	// Check if user has NFT for bonus calculation
	hasNFT, err := nh.nftService.CheckUserNFTOwnership(c.Request.Context(), req.UserWallet)
	if err != nil {
		// Log error but continue with default points
		hasNFT = false
	}

	// Calculate reward points (no daily limit for Twitter posts as per requirement)
	rewardPoints := 1 // Base Twitter post reward
	if hasNFT {
		rewardPoints = 2 // Double points for NFT holders
	}

	// Add points to user
	err = nh.pointsService.AddDirectPoints(c.Request.Context(), &models.DirectPointsRequest{
		UserWallet:  req.UserWallet,
		Points:      rewardPoints,
		Source:      models.PointsSourceTwitterPost,
		Description: "Twitter Post Task Reward",
		Reference:   req.TaskID,
		Metadata: map[string]interface{}{
			"task_id":        req.TaskID,
			"post_url":       req.PostURL,
			"has_nft":        hasNFT,
			"base_reward":    1,
			"nft_multiplier": hasNFT,
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add Twitter post reward: " + err.Error(),
		})
		return
	}

	// TODO: Create user_task_completions record for statistics
	// This should be done to ensure the task appears in subnet statistics
	// For now, we'll add a note that this needs to be implemented

	// Get updated user total points (optional, for response)
	newTotal := 0 // You might want to query this from the database

	c.JSON(http.StatusOK, &models.TwitterPostRewardResponse{
		Success:     true,
		UserWallet:  req.UserWallet,
		TaskID:      req.TaskID,
		PostURL:     req.PostURL,
		PointsAdded: rewardPoints,
		NewTotal:    newTotal,
		HasNFT:      hasNFT,
		Message:     "Twitter post reward added successfully",
	})
}
