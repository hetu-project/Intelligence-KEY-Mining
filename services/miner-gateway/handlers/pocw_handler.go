package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/services"
)

// PoCWHandler handles PoCW browser API requests
type PoCWHandler struct {
	queryService *services.PoCWQueryService
}

// NewPoCWHandler creates a new PoCW handler
func NewPoCWHandler(queryService *services.PoCWQueryService) *PoCWHandler {
	return &PoCWHandler{
		queryService: queryService,
	}
}

// RegisterRoutes registers PoCW API routes
func (h *PoCWHandler) RegisterRoutes(router *gin.RouterGroup) {
	pocw := router.Group("/pocw")
	{
		// Round endpoints
		pocw.GET("/rounds", h.GetRoundsList)
		pocw.GET("/rounds/:round_id", h.GetRoundDetail)
		pocw.GET("/rounds/:round_id/tasks", h.GetRoundTasks)
		pocw.GET("/rounds/:round_id/votes", h.GetRoundVotes)

		// Statistics endpoints
		pocw.GET("/stats/dashboard", h.GetDashboardStats)
		pocw.GET("/stats/validators", h.GetValidatorStats)
		pocw.GET("/stats/trend", h.GetRoundTrend)
	}
}

// GetRoundsList retrieves a paginated list of rounds
// @Summary Get rounds list
// @Description Get paginated list of PoCW rounds
// @Tags PoCW
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param status query string false "Filter by status: all, idle, task_process, vlc_verify, quality_vote, consensus, complete"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pocw/rounds [get]
func (h *PoCWHandler) GetRoundsList(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.DefaultQuery("status", "all")

	// Validate parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Query rounds
	rounds, total, err := h.queryService.GetRoundsList(c.Request.Context(), page, limit, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get rounds: " + err.Error(),
		})
		return
	}

	// Calculate pagination info
	totalPages := (total + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"rounds":      rounds,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		},
	})
}

// GetRoundDetail retrieves detailed information about a specific round
// @Summary Get round detail
// @Description Get detailed information about a specific PoCW round
// @Tags PoCW
// @Param round_id path string true "Round ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pocw/rounds/{round_id} [get]
func (h *PoCWHandler) GetRoundDetail(c *gin.Context) {
	roundID := c.Param("round_id")

	if roundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "round_id is required",
		})
		return
	}

	detail, err := h.queryService.GetRoundDetail(c.Request.Context(), roundID)
	if err != nil {
		if err.Error() == "round not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Round not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get round detail: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}

// GetRoundTasks retrieves tasks for a specific round
// @Summary Get round tasks
// @Description Get all tasks for a specific PoCW round
// @Tags PoCW
// @Param round_id path string true "Round ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pocw/rounds/{round_id}/tasks [get]
func (h *PoCWHandler) GetRoundTasks(c *gin.Context) {
	roundID := c.Param("round_id")

	if roundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "round_id is required",
		})
		return
	}

	tasks, err := h.queryService.GetRoundTasks(c.Request.Context(), roundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get round tasks: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"tasks": tasks,
			"count": len(tasks),
		},
	})
}

// GetRoundVotes retrieves votes for a specific round
// @Summary Get round votes
// @Description Get all validator votes for a specific PoCW round
// @Tags PoCW
// @Param round_id path string true "Round ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pocw/rounds/{round_id}/votes [get]
func (h *PoCWHandler) GetRoundVotes(c *gin.Context) {
	roundID := c.Param("round_id")

	if roundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "round_id is required",
		})
		return
	}

	votes, err := h.queryService.GetRoundVotes(c.Request.Context(), roundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get round votes: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"votes": votes,
			"count": len(votes),
		},
	})
}

// GetDashboardStats retrieves dashboard statistics
// @Summary Get dashboard statistics
// @Description Get overall PoCW system statistics for dashboard
// @Tags PoCW
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pocw/stats/dashboard [get]
func (h *PoCWHandler) GetDashboardStats(c *gin.Context) {
	stats, err := h.queryService.GetDashboardStats(c.Request.Context())
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

// GetValidatorStats retrieves validator performance statistics
// @Summary Get validator statistics
// @Description Get performance statistics for all validators
// @Tags PoCW
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pocw/stats/validators [get]
func (h *PoCWHandler) GetValidatorStats(c *gin.Context) {
	stats, err := h.queryService.GetValidatorStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get validator stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"validators": stats,
			"count":      len(stats),
		},
	})
}

// GetRoundTrend retrieves round trend data for charts
// @Summary Get round trend
// @Description Get round trend data for the last N days
// @Tags PoCW
// @Param days query int false "Number of days (default: 7, max: 90)"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/pocw/stats/trend [get]
func (h *PoCWHandler) GetRoundTrend(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	// Validate days
	if days < 1 || days > 90 {
		days = 7
	}

	trend, err := h.queryService.GetRoundTrend(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get round trend: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"trend": trend,
			"days":  days,
		},
	})
}
