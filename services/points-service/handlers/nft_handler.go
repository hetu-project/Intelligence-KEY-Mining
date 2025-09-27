package handlers

import (
	"net/http"

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
