package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/validator/services"
)

// ValidationHandler handles validation-related HTTP requests
type ValidationHandler struct {
	validationService *services.ValidationService
}

// NewValidationHandler creates a new validation handler
func NewValidationHandler(validationService *services.ValidationService) *ValidationHandler {
	return &ValidationHandler{
		validationService: validationService,
	}
}

// ValidateTask handles task validation requests
func (h *ValidationHandler) ValidateTask(c *gin.Context) {
	var minerOutput models.MinerOutput
	if err := c.ShouldBindJSON(&minerOutput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if minerOutput.EventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "event_id is required",
		})
		return
	}

	if minerOutput.TaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "task_id is required",
		})
		return
	}

	// Execute validation
	vote, err := h.validationService.ValidateMinerOutput(c.Request.Context(), &minerOutput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    vote,
	})
}

// GetConfig returns the validator configuration
func (h *ValidationHandler) GetConfig(c *gin.Context) {
	config := h.validationService.GetValidatorInfo()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":     config.ID,
			"role":   config.Role,
			"weight": config.Weight,
		},
	})
}
