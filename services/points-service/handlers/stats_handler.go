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

// GetUserSubnets gets subnets that a user has participated in
func (sh *StatsHandler) GetUserSubnets(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "wallet address is required",
		})
		return
	}

	subnets, err := sh.statsService.GetUserSubnets(c.Request.Context(), userWallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get user subnets: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_wallet": userWallet,
			"subnets":     subnets,
			"count":       len(subnets),
		},
	})
}

// GetUserSubnetSummary gets a summary of user's subnet participation
func (sh *StatsHandler) GetUserSubnetSummary(c *gin.Context) {
	userWallet := c.Param("wallet")
	if userWallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "wallet address is required",
		})
		return
	}

	summary, err := sh.statsService.GetUserSubnetSummary(c.Request.Context(), userWallet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get user subnet summary: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// GetActiveMiners gets count of active miners
func (sh *StatsHandler) GetActiveMiners(c *gin.Context) {
	activeMiners, err := sh.statsService.GetActiveMinersCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get active miners count: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active_miners": activeMiners,
		},
	})
}

// GetActiveTasksCount gets count of active tasks
func (sh *StatsHandler) GetActiveTasksCount(c *gin.Context) {
	activeTasks, err := sh.statsService.GetActiveTasksCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get active tasks count: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active_tasks_count": activeTasks,
		},
	})
}

// GetSubnetLeaders gets top users in each subnet
func (sh *StatsHandler) GetSubnetLeaders(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	limit := 10 // Default limit for subnets (10 subnets * 3 leaders each = 30 results per page)
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}

	offset := (page - 1) * limit

	leaders, totalCount, err := sh.statsService.GetSubnetLeaders(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get subnet leaders: " + err.Error(),
		})
		return
	}

	totalPages := (totalCount + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"leaders": leaders,
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

// GetDailyPerformers gets users who completed tasks today
func (sh *StatsHandler) GetDailyPerformers(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	limit := 50 // Default limit for daily performers
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := (page - 1) * limit

	performers, totalCount, err := sh.statsService.GetDailyPerformers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get daily performers: " + err.Error(),
		})
		return
	}

	totalPages := (totalCount + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"performers": performers,
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

// GetDashboardStats gets comprehensive dashboard statistics
func (sh *StatsHandler) GetDashboardStats(c *gin.Context) {
	stats, err := sh.statsService.GetDashboardStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get dashboard stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetSubnetUserRanking gets user ranking by points in a specific subnet
func (sh *StatsHandler) GetSubnetUserRanking(c *gin.Context) {
	subnetID := c.Param("subnet_id")
	if subnetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "subnet_id is required",
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

	limit := 20 // Default limit
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := (page - 1) * limit

	rankings, totalCount, err := sh.statsService.GetSubnetUserRanking(c.Request.Context(), subnetID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get subnet user ranking: " + err.Error(),
		})
		return
	}

	totalPages := (totalCount + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"subnet_id": subnetID,
			"rankings":  rankings,
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

// GetSubnetsDailyPoints gets daily points for all subnets over the past N days
func (sh *StatsHandler) GetSubnetsDailyPoints(c *gin.Context) {
	// Parse days parameter (default 7, max 30)
	days := 7
	if daysParam := c.Query("days"); daysParam != "" {
		if parsedDays, err := strconv.Atoi(daysParam); err == nil && parsedDays > 0 && parsedDays <= 30 {
			days = parsedDays
		}
	}

	// Optional subnet_id filter
	subnetID := c.Query("subnet_id")

	// Query data
	subnets, err := sh.statsService.GetSubnetsDailyPoints(c.Request.Context(), days, subnetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get subnets daily points: " + err.Error(),
		})
		return
	}

	// Calculate date range for response
	startDate := ""
	endDate := ""
	if len(subnets) > 0 && len(subnets[0].DailyPoints) > 0 {
		startDate = subnets[0].DailyPoints[0].Date
		endDate = subnets[0].DailyPoints[len(subnets[0].DailyPoints)-1].Date
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"start_date":  startDate,
			"end_date":    endDate,
			"days":        days,
			"subnets":     subnets,
			"total_count": len(subnets),
		},
	})
}

// GetChatTasksDailyPoints gets daily points for chat tasks in all subnets over the past N days
func (sh *StatsHandler) GetChatTasksDailyPoints(c *gin.Context) {
	// Parse days parameter (default 7, max 30)
	days := 7
	if daysParam := c.Query("days"); daysParam != "" {
		if parsedDays, err := strconv.Atoi(daysParam); err == nil && parsedDays > 0 && parsedDays <= 30 {
			days = parsedDays
		}
	}

	// Optional subnet_id filter
	subnetID := c.Query("subnet_id")

	// Query data
	chatTasks, err := sh.statsService.GetChatTasksDailyPoints(c.Request.Context(), days, subnetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get chat tasks daily points: " + err.Error(),
		})
		return
	}

	// Calculate date range for response
	startDate := ""
	endDate := ""
	if len(chatTasks) > 0 && len(chatTasks[0].DailyPoints) > 0 {
		startDate = chatTasks[0].DailyPoints[0].Date
		endDate = chatTasks[0].DailyPoints[len(chatTasks[0].DailyPoints)-1].Date
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"start_date":  startDate,
			"end_date":    endDate,
			"days":        days,
			"chat_tasks":  chatTasks,
			"total_count": len(chatTasks),
		},
	})
}

// RegisterRoutes registers HTTP routes for stats
func (sh *StatsHandler) RegisterRoutes(router *gin.RouterGroup) {
	stats := router.Group("/stats")
	{
		// System statistics
		stats.GET("/total-points", sh.GetTotalPoints)
		stats.GET("/total-users", sh.GetTotalUsers)
		stats.GET("/overall", sh.GetOverallStats)
		stats.GET("/dashboard", sh.GetDashboardStats) // NEW: Comprehensive dashboard

		// Mining statistics
		stats.GET("/active-miners", sh.GetActiveMiners)          // NEW: Active miners count
		stats.GET("/active-tasks-count", sh.GetActiveTasksCount) // NEW: Active tasks count

		// Ranking and leaderboards
		stats.GET("/subnet-leaders", sh.GetSubnetLeaders)     // NEW: Subnet leaders (top 3 per subnet)
		stats.GET("/daily-performers", sh.GetDailyPerformers) // NEW: Daily task performers

		// Subnet statistics
		stats.GET("/subnets", sh.GetSubnetStats)
		stats.GET("/subnets/daily-points", sh.GetSubnetsDailyPoints) // NEW: Subnet daily points over N days
		stats.GET("/subnets/:subnet_id", sh.GetSubnetDetails)
		stats.GET("/subnets/:subnet_id/points-today", sh.GetSubnetPointsToday)
		stats.GET("/subnets/:subnet_id/users-count", sh.GetSubnetUsersCount)
		stats.GET("/subnets/:subnet_id/ranking", sh.GetSubnetUserRanking)

		// Chat task statistics
		stats.GET("/chat-tasks/daily-points", sh.GetChatTasksDailyPoints) // NEW: Chat task daily points over N days

		// User statistics
		stats.GET("/users/ranking", sh.GetUserPointsRanking)
		stats.GET("/users/:wallet/history", sh.GetUserPointsHistory)
		stats.GET("/users/:wallet/subnets", sh.GetUserSubnets)
		stats.GET("/users/:wallet/subnet-summary", sh.GetUserSubnetSummary)
	}

	// NOTE: Admin routes have been moved to admin_handler.go
	// Points adjustment system (new incremental design) replaces old override system
}
