package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/services"
)

// AdminHandler 管理员操作handler
type AdminHandler struct {
	statsService *services.StatsService
}

// NewAdminHandler 创建管理员handler
func NewAdminHandler(statsService *services.StatsService) *AdminHandler {
	return &AdminHandler{
		statsService: statsService,
	}
}

// AddPointsAdjustmentRequest 添加积分调整请求
type AddPointsAdjustmentRequest struct {
	AdjustmentPoints int    `json:"adjustment_points" binding:"required"` // 可以是正数或负数
	AdminWallet      string `json:"admin_wallet" binding:"required"`
	Reason           string `json:"reason" binding:"required,max=20"` // 最多20个字（中文）
}

// AddPointsAdjustment 添加积分调整
// POST /api/v1/admin/subnets/:subnet_id/users/:wallet/points-adjustment
func (ah *AdminHandler) AddPointsAdjustment(c *gin.Context) {
	subnetID := c.Param("subnet_id")
	walletAddress := c.Param("wallet")

	if subnetID == "" || walletAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "subnet_id and wallet are required",
		})
		return
	}

	var req AddPointsAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	// 添加调整记录
	err := ah.statsService.AddPointsAdjustment(c.Request.Context(), subnetID, walletAddress, req.AdjustmentPoints, req.AdminWallet, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add points adjustment: " + err.Error(),
		})
		return
	}

	// 查询更新后的总积分
	totalPoints, err := ah.statsService.GetUserSubnetTotalPoints(c.Request.Context(), subnetID, walletAddress)
	if err != nil {
		totalPoints = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"subnet_id":         subnetID,
		"wallet_address":    walletAddress,
		"adjustment_points": req.AdjustmentPoints,
		"new_total_points":  totalPoints,
		"reason":            req.Reason,
		"message":           "Points adjustment added successfully",
	})
}

// GetAdjustmentHistory 获取积分调整历史
// GET /api/v1/admin/subnets/:subnet_id/users/:wallet/adjustment-history
func (ah *AdminHandler) GetAdjustmentHistory(c *gin.Context) {
	subnetID := c.Param("subnet_id")
	walletAddress := c.Param("wallet")

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	history, err := ah.statsService.GetAdjustmentHistory(c.Request.Context(), subnetID, walletAddress, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get adjustment history: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"subnet_id":      subnetID,
			"wallet_address": walletAddress,
			"adjustments":    history,
			"total":          len(history),
		},
	})
}

// SearchUserByWallet 精确搜索用户钱包地址
// GET /api/v1/admin/subnets/:subnet_id/users/search?wallet=0x...
func (ah *AdminHandler) SearchUserByWallet(c *gin.Context) {
	subnetID := c.Param("subnet_id")
	walletAddress := c.Query("wallet")

	if subnetID == "" || walletAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "subnet_id and wallet are required",
		})
		return
	}

	userInfo, err := ah.statsService.SearchUserByWallet(c.Request.Context(), subnetID, walletAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userInfo,
	})
}

// RegisterRoutes 注册管理员路由
func (ah *AdminHandler) RegisterRoutes(router *gin.RouterGroup) {
	admin := router.Group("/admin")
	{
		// 积分调整相关
		admin.POST("/subnets/:subnet_id/users/:wallet/points-adjustment", ah.AddPointsAdjustment)
		admin.GET("/subnets/:subnet_id/users/:wallet/adjustment-history", ah.GetAdjustmentHistory)

		// 用户搜索
		admin.GET("/subnets/:subnet_id/users/search", ah.SearchUserByWallet)
	}
}
