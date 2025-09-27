package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/services"
)

// StatsHandler handles statistics and query HTTP requests
type StatsHandler struct {
	pointsService *services.PointsService
	statsService  *services.StatsService
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(pointsService *services.PointsService, statsService *services.StatsService) *StatsHandler {
	return &StatsHandler{
		pointsService: pointsService,
		statsService:  statsService,
	}
}

// GetTotalPoints gets total points across all users
func (sh *StatsHandler) GetTotalPoints(c *gin.Context) {
	totalPoints, err := sh.statsService.GetTotalPoints(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get total points: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_points": totalPoints,
		},
	})
}

// GetTotalUsers gets total number of registered users
func (sh *StatsHandler) GetTotalUsers(c *gin.Context) {
	totalUsers, err := sh.statsService.GetTotalUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get total users: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_users": totalUsers,
		},
	})
}

// GetSubnetStats gets subnet statistics
func (sh *StatsHandler) GetSubnetStats(c *gin.Context) {
	subnets, err := sh.statsService.GetSubnetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get subnet stats: " + err.Error(),
		})
		return
	}

	// Calculate summary
	totalSubnets := len(subnets)
	totalSubnetPoints := 0
	totalSubnetUsers := 0

	for _, subnet := range subnets {
		totalSubnetPoints += subnet.TotalPointsDistributed
		totalSubnetUsers += subnet.UniqueUsers
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_subnets":       totalSubnets,
			"total_subnet_points": totalSubnetPoints,
			"total_subnet_users":  totalSubnetUsers,
			"subnets":             subnets,
		},
	})
}

// GetSubnetDetails gets detailed statistics for a specific subnet
func (sh *StatsHandler) GetSubnetDetails(c *gin.Context) {
	subnetID := c.Param("subnet_id")
	if subnetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "subnet_id is required",
		})
		return
	}

	details, err := sh.statsService.GetSubnetDetails(c.Request.Context(), subnetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get subnet details: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    details,
	})
}

// GetSubnetPointsToday gets points distributed today for a subnet
func (sh *StatsHandler) GetSubnetPointsToday(c *gin.Context) {
	subnetID := c.Param("subnet_id")
	if subnetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "subnet_id is required",
		})
		return
	}

	todayPoints, err := sh.statsService.GetSubnetPointsToday(c.Request.Context(), subnetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get today's subnet points: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"subnet_id":    subnetID,
			"today_points": todayPoints,
		},
	})
}

// GetSubnetUsersCount gets number of users who completed tasks in a subnet
func (sh *StatsHandler) GetSubnetUsersCount(c *gin.Context) {
	subnetID := c.Param("subnet_id")
	if subnetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "subnet_id is required",
		})
		return
	}

	usersCount, err := sh.statsService.GetSubnetUsersCount(c.Request.Context(), subnetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get subnet users count: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"subnet_id":   subnetID,
			"users_count": usersCount,
		},
	})
}

// GetUserPointsRanking gets user points ranking
func (sh *StatsHandler) GetUserPointsRanking(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	limit := 50 // Default limit
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := (page - 1) * limit

	ranking, totalCount, err := sh.statsService.GetUserPointsRanking(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get user points ranking: " + err.Error(),
		})
		return
	}

	totalPages := (totalCount + limit - 1) / limit // Ceiling division

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"ranking": ranking,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total_count": totalCount,
				"total_pages": totalPages,
				"has_next":    page < totalPages,
				"has_prev":    page > 1,
			},
		},
	})
}

// GetUserPointsHistory gets detailed points history for a user
func (sh *StatsHandler) GetUserPointsHistory(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "wallet address is required",
		})
		return
	}

	// Parse pagination parameters
	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	limit := 20 // Default limit for history
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := (page - 1) * limit

	history, totalCount, err := sh.statsService.GetUserPointsHistory(c.Request.Context(), userWallet, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get user points history: " + err.Error(),
		})
		return
	}

	totalPages := (totalCount + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_wallet": userWallet,
			"history":     history,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total_count": totalCount,
				"total_pages": totalPages,
				"has_next":    page < totalPages,
				"has_prev":    page > 1,
			},
		},
	})
}

// GetOverallStats gets comprehensive system statistics
func (sh *StatsHandler) GetOverallStats(c *gin.Context) {
	stats, err := sh.statsService.GetOverallStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get overall stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
