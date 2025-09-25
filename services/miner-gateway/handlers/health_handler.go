package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

// Health handles basic health check
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "miner-gateway",
	})
}

// Ready handles readiness check
func (h *HealthHandler) Ready(c *gin.Context) {
	if err := h.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "miner-gateway",
	})
}
