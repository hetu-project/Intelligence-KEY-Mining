package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/services"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/utils"
)

// SubnetHandler handles subnet-related requests
type SubnetHandler struct {
	subnetService *services.SubnetService
}

// NewSubnetHandler creates a new subnet handler
func NewSubnetHandler(subnetService *services.SubnetService) *SubnetHandler {
	return &SubnetHandler{
		subnetService: subnetService,
	}
}

// TransferSubnetRequest represents the request to transfer subnet ownership
type TransferSubnetRequest struct {
	CurrentOwner string `json:"current_owner" binding:"required"`
	NewOwner     string `json:"new_owner" binding:"required"`
}

// TransferSubnet transfers subnet ownership to another user
func (sh *SubnetHandler) TransferSubnet(c *gin.Context) {
	subnetID := c.Param("subnet_id")
	if subnetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Subnet ID is required",
		})
		return
	}

	var req TransferSubnetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	err := sh.subnetService.TransferSubnet(c.Request.Context(), subnetID, req.CurrentOwner, req.NewOwner)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to transfer subnet: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subnet transferred successfully",
		"data": gin.H{
			"subnet_id":     subnetID,
			"current_owner": req.CurrentOwner,
			"new_owner":     req.NewOwner,
		},
	})
}

// UpdateSubnetFinancialData updates TVL and Valuation for a subnet
func (sh *SubnetHandler) UpdateSubnetFinancialData(c *gin.Context) {
	subnetID := c.Param("subnet_id")
	if subnetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Subnet ID is required",
		})
		return
	}

	var req models.SubnetFinancialUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request: " + err.Error(),
		})
		return
	}

	err := sh.subnetService.UpdateSubnetFinancialData(c.Request.Context(), subnetID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update subnet financial data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subnet financial data updated successfully",
		"data": gin.H{
			"subnet_id":           subnetID,
			"tvl":                 req.TVL,
			"valuation":           req.Valuation,
			"tvl_formatted":       utils.FormatFinancialValue(req.TVL),
			"valuation_formatted": utils.FormatFinancialValue(req.Valuation),
			"updated_by":          req.CurrentOwner,
		},
	})
}
