package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/services"
)

// WhitelistHandler handles whitelist-related requests
type WhitelistHandler struct {
	whitelistService *services.WhitelistService
}

// NewWhitelistHandler creates a new whitelist handler
func NewWhitelistHandler(whitelistService *services.WhitelistService) *WhitelistHandler {
	return &WhitelistHandler{
		whitelistService: whitelistService,
	}
}

// AddWhitelistUserRequest represents the request to add a user to whitelist
type AddWhitelistUserRequest struct {
	WalletAddress string `json:"wallet_address" binding:"required"`
	CreatedBy     string `json:"created_by" binding:"required"`
	Reason        string `json:"reason"`
}

// RemoveWhitelistUserRequest represents the request to remove a user from whitelist
type RemoveWhitelistUserRequest struct {
	WalletAddress string `json:"wallet_address" binding:"required"`
}

// AddWhitelistUser adds a user to the whitelist
func (wh *WhitelistHandler) AddWhitelistUser(c *gin.Context) {
	var req AddWhitelistUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	err := wh.whitelistService.AddUserToWhitelist(c.Request.Context(), req.WalletAddress, req.CreatedBy, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add user to whitelist: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User added to whitelist successfully",
		"data": gin.H{
			"wallet_address": req.WalletAddress,
			"created_by":     req.CreatedBy,
			"reason":         req.Reason,
		},
	})
}

// RemoveWhitelistUser removes a user from the whitelist
func (wh *WhitelistHandler) RemoveWhitelistUser(c *gin.Context) {
	var req RemoveWhitelistUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	err := wh.whitelistService.RemoveUserFromWhitelist(c.Request.Context(), req.WalletAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to remove user from whitelist: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User removed from whitelist successfully",
		"data": gin.H{
			"wallet_address": req.WalletAddress,
		},
	})
}

// GetWhitelistUsers returns paginated list of whitelist users
func (wh *WhitelistHandler) GetWhitelistUsers(c *gin.Context) {
	// Parse query parameters
	status := c.DefaultQuery("status", "all")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	users, total, err := wh.whitelistService.GetWhitelistUsers(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get whitelist users: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users":  users,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// CheckWhitelistStatus checks if a user is whitelisted
func (wh *WhitelistHandler) CheckWhitelistStatus(c *gin.Context) {
	walletAddress := c.Param("wallet")
	if walletAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Wallet address is required",
		})
		return
	}

	isWhitelisted, err := wh.whitelistService.IsUserWhitelisted(c.Request.Context(), walletAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check whitelist status: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"wallet_address": walletAddress,
			"is_whitelisted": isWhitelisted,
		},
	})
}
