package handlers

import (
	"context"
	"fmt"
	"log"
	"math"
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

// GetUserChatScore gets user's chat task cumulative score
func (nh *NFTHandler) GetUserChatScore(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "wallet address is required",
		})
		return
	}

	score, err := nh.pointsService.GetUserChatScore(c.Request.Context(), userWallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get chat score: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    score,
	})
}

// GetUserTodayEarnings gets user's today earnings breakdown
func (nh *NFTHandler) GetUserTodayEarnings(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "wallet address is required",
		})
		return
	}

	earnings, err := nh.pointsService.GetUserTodayEarnings(c.Request.Context(), userWallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get today's earnings: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    earnings,
	})
}

// addCreatorCommission adds 5% commission for subnet creator
func (nh *NFTHandler) addCreatorCommission(ctx context.Context, subnetID string, userPoints int) error {
	// 1. Get subnet creator
	creatorWallet, err := nh.pointsService.GetSubnetCreator(ctx, subnetID)
	if err != nil || creatorWallet == "" {
		return fmt.Errorf("failed to get subnet creator: %v", err)
	}

	// 2. Calculate 5% commission (float)
	commission := float64(userPoints) * 0.05

	// 3. Get accumulated commission
	accumulated, err := nh.pointsService.GetAccumulatedCommission(ctx, creatorWallet)
	if err != nil {
		return fmt.Errorf("failed to get accumulated commission: %v", err)
	}

	// 4. Calculate total accumulated commission
	totalAccumulated := accumulated + commission
	commissionPoints := int(math.Floor(totalAccumulated))
	remainingAccumulated := totalAccumulated - float64(commissionPoints)

	// 5. Update accumulated commission
	if err := nh.pointsService.UpdateAccumulatedCommission(ctx, creatorWallet, remainingAccumulated, commissionPoints); err != nil {
		return fmt.Errorf("failed to update accumulated commission: %v", err)
	}

	// 6. If we have integer points, distribute them immediately
	if commissionPoints > 0 {
		err = nh.pointsService.AddDirectPoints(ctx, &models.DirectPointsRequest{
			UserWallet:  creatorWallet,
			Points:      commissionPoints,
			Source:      models.PointsSourceCreatorCommission,
			Description: "Creator Commission from Chat Task",
			Reference:   fmt.Sprintf("chat-commission-%s", subnetID),
			SubnetID:    subnetID,
			Metadata: map[string]interface{}{
				"subnet_id":        subnetID,
				"source_task_type": "chat",
			},
		})
		if err != nil {
			return fmt.Errorf("failed to distribute commission points: %v", err)
		}
		log.Printf("💰 Creator %s received %d commission points from chat task", creatorWallet[:10]+"...", commissionPoints)
	}

	return nil
}

// RegisterRoutes registers HTTP routes for NFT operations
func (nh *NFTHandler) RegisterRoutes(router *gin.RouterGroup) {
	points := router.Group("/points")
	{
		// NFT and invitation reward endpoints
		points.POST("/nft-purchase", nh.HandleNFTPurchaseBonus)
		points.POST("/invitation-reward", nh.HandleInvitationReward)
		points.POST("/telegram-task-reward", nh.HandleTelegramTaskReward)
		points.POST("/twitter-post-reward", nh.HandleTwitterPostReward)
		points.POST("/twitter-follow-reward", nh.HandleTwitterFollowReward)
		points.POST("/chat-task-reward", nh.HandleChatTaskReward)              // NEW: Chat task reward
		points.POST("/register-qr-code-reward", nh.HandleRegisterQRCodeReward) // NEW: Register QR code task

		// User completed tasks query
		points.GET("/completed-tasks/:wallet", nh.GetUserCompletedTasks)

		// NEW: User chat score and today earnings
		points.GET("/chat-score/:wallet", nh.GetUserChatScore)
		points.GET("/earned-today/:wallet", nh.GetUserTodayEarnings)
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

// HandleChatTaskReward handles Chat task reward points
func (nh *NFTHandler) HandleChatTaskReward(c *gin.Context) {
	var req models.ChatTaskRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.UserWallet == "" || req.SubnetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "user_wallet and subnet_id are required",
		})
		return
	}

	// Check if user has already completed a chat task for this subnet today
	today := time.Now().Format("2006-01-02")
	completed, err := nh.pointsService.CheckUserChatTaskToday(c.Request.Context(), req.UserWallet, req.SubnetID, today)
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
			"error":   "User has already completed a chat task for this subnet today",
		})
		return
	}

	// Check if user has NFT for bonus calculation
	hasNFT, err := nh.nftService.CheckUserNFTOwnership(c.Request.Context(), req.UserWallet)
	if err != nil {
		hasNFT = false
	}

	// Calculate reward points
	rewardPoints := 1 // Base chat task reward
	if hasNFT {
		rewardPoints = 2 // Double points for NFT holders
	}

	// Add points to user
	err = nh.pointsService.AddDirectPoints(c.Request.Context(), &models.DirectPointsRequest{
		UserWallet:  req.UserWallet,
		Points:      rewardPoints,
		Source:      models.PointsSourceChatTask,
		Description: "Chat Task Reward",
		Reference:   fmt.Sprintf("chat-%s-%s", req.SubnetID, today),
		SubnetID:    req.SubnetID,
		Metadata: map[string]interface{}{
			"subnet_id":      req.SubnetID,
			"has_nft":        hasNFT,
			"base_reward":    1,
			"nft_multiplier": hasNFT,
			"reward_date":    today,
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add chat task reward: " + err.Error(),
		})
		return
	}

	// Add creator 5% commission
	if err := nh.addCreatorCommission(c.Request.Context(), req.SubnetID, rewardPoints); err != nil {
		log.Printf("Failed to add creator commission for chat task: %v", err)
		// Don't fail the main flow
	}

	// Get updated user total points
	newTotal, err := nh.pointsService.GetUserPoints(c.Request.Context(), req.UserWallet)
	if err != nil {
		newTotal = 0
	}

	c.JSON(http.StatusOK, &models.ChatTaskRewardResponse{
		Success:     true,
		UserWallet:  req.UserWallet,
		SubnetID:    req.SubnetID,
		PointsAdded: rewardPoints,
		NewTotal:    newTotal,
		HasNFT:      hasNFT,
		Message:     "Chat task reward added successfully",
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

// HandleTwitterFollowReward handles Twitter follow task reward points
func (nh *NFTHandler) HandleTwitterFollowReward(c *gin.Context) {
	var req models.TwitterFollowRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.UserWallet == "" || req.TaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "user_wallet and task_id are required",
		})
		return
	}

	// Lookup subnet_id from tasks table
	subnetID, err := nh.pointsService.GetTaskSubnetID(c.Request.Context(), req.TaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to resolve task subnet: " + err.Error(),
		})
		return
	}

	// Check if user has NFT for bonus calculation (currently globally disabled in NFT service)
	hasNFT, err := nh.nftService.CheckUserNFTOwnership(c.Request.Context(), req.UserWallet)
	if err != nil {
		hasNFT = false
	}

	// Calculate reward points
	rewardPoints := 1
	if hasNFT {
		rewardPoints = 2
	}

	// Add points to user (tx_ref = task_id, subnet_id remains NULL for non-chat tasks)
	err = nh.pointsService.AddDirectPoints(c.Request.Context(), &models.DirectPointsRequest{
		UserWallet:  req.UserWallet,
		Points:      rewardPoints,
		Source:      models.PointsSourceTwitterFollow,
		Description: "Twitter Follow Task Reward",
		Reference:   req.TaskID,
		Metadata: map[string]interface{}{
			"task_id":        req.TaskID,
			"has_nft":        hasNFT,
			"base_reward":    1,
			"nft_multiplier": hasNFT,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add Twitter follow reward: " + err.Error(),
		})
		return
	}

	// Compute new total for this subnet including override logic
	newTotal := 0
	if subnetID != "" {
		if total, err2 := nh.pointsService.GetUserSubnetTotalPoints(c.Request.Context(), subnetID, req.UserWallet); err2 == nil {
			newTotal = total
		}
	}

	c.JSON(http.StatusOK, &models.TwitterFollowRewardResponse{
		Success:     true,
		UserWallet:  req.UserWallet,
		TaskID:      req.TaskID,
		PointsAdded: rewardPoints,
		NewTotal:    newTotal,
		HasNFT:      hasNFT,
		Message:     "Twitter follow reward added successfully",
	})
}

// HandleRegisterQRCodeReward handles Register QR code task reward
// This task gives 0 points but records completion and saves badge snapshot
func (nh *NFTHandler) HandleRegisterQRCodeReward(c *gin.Context) {
	var req models.RegisterQRCodeRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.UserWallet == "" || req.TaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "user_wallet and task_id are required",
		})
		return
	}

	// Check if user already completed this task (one time only)
	alreadyCompleted, err := nh.pointsService.CheckUserCompletedTask(c.Request.Context(), req.UserWallet, req.TaskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check task completion status: " + err.Error(),
		})
		return
	}

	if alreadyCompleted {
		c.JSON(http.StatusOK, &models.RegisterQRCodeRewardResponse{
			Success:     true,
			UserWallet:  req.UserWallet,
			TaskID:      req.TaskID,
			PointsAdded: 0,
			RewardBadge: "",
			AlreadyDone: true,
			Message:     "User has already completed this task",
		})
		return
	}

	// Get reward badge from task payload (snapshot)
	rewardBadge, err := nh.pointsService.GetTaskRewardBadge(c.Request.Context(), req.TaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to get reward badge from task: " + err.Error(),
		})
		return
	}

	// Add points record with 0 points but save badge snapshot
	err = nh.pointsService.AddDirectPoints(c.Request.Context(), &models.DirectPointsRequest{
		UserWallet:  req.UserWallet,
		Points:      0, // Register QR code tasks give 0 points
		Source:      models.PointsSourceRegisterQRCode,
		Description: "Register QR Code Task Completion",
		Reference:   req.TaskID,
		Metadata: map[string]interface{}{
			"task_id":      req.TaskID,
			"reward_badge": rewardBadge, // Save badge snapshot
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to record Register QR code completion: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, &models.RegisterQRCodeRewardResponse{
		Success:     true,
		UserWallet:  req.UserWallet,
		TaskID:      req.TaskID,
		PointsAdded: 0,
		RewardBadge: rewardBadge,
		AlreadyDone: false,
		Message:     "Register QR code task completed successfully",
	})
}
