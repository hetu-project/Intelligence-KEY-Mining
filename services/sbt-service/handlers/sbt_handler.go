package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/sbt-service/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/sbt-service/services"
)

// SBTHandler handles SBT-related HTTP requests
type SBTHandler struct {
	metadataService   *services.MetadataService
	blockchainService *services.BlockchainService
}

// NewSBTHandler creates a new SBT handler
func NewSBTHandler(metadataService *services.MetadataService, blockchainService *services.BlockchainService) *SBTHandler {
	return &SBTHandler{
		metadataService:   metadataService,
		blockchainService: blockchainService,
	}
}

// RegisterUser handles user SBT registration requests
// POST /api/v1/sbt/register
func (h *SBTHandler) RegisterUser(c *gin.Context) {
	var req models.UserRegistrationRequest

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Validate required fields
	if err := h.validateRegistrationRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": err.Error(),
		})
		return
	}

	// Generate SBT metadata
	response, err := h.metadataService.GenerateSBT(c.Request.Context(), &req)
	if err != nil {
		// Check if user already exists error
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "User already exists",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate SBT metadata",
			"message": err.Error(),
		})
		return
	}

	// Call blockchain to mint SBT
	inviterAddress := req.InviteFrom
	if inviterAddress == "" {
		inviterAddress = "0x0000000000000000000000000000000000000000" // Zero address if no inviter
	}
	tokenId, err := h.blockchainService.MintSBT(c.Request.Context(), req.WalletAddress, req.DisplayName, inviterAddress, response.TokenURI)
	if err != nil {
		// Minting failed, but metadata already generated, log error but don't affect response
		// In production environment, may need to rollback metadata or implement retry mechanism
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to mint SBT on blockchain",
			"message": err.Error(),
		})
		return
	}

	// Update response, add token ID
	response.TokenID = tokenId.String()
	response.ContractAddress = h.getContractAddress()

	// Return success response
	c.JSON(http.StatusOK, response)
}

// GetDynamicMetadata retrieves dynamic metadata (for NFT platform calls)
// GET /api/v1/sbt/dynamic/:wallet
func (h *SBTHandler) GetDynamicMetadata(c *gin.Context) {
	walletAddress := c.Param("wallet")

	// Validate wallet address format
	if !isValidWalletAddress(walletAddress) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid wallet address",
			"message": "Wallet address must be a valid Ethereum address",
		})
		return
	}

	// Get dynamic metadata
	metadata, err := h.metadataService.GetDynamicMetadata(c.Request.Context(), walletAddress)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get dynamic metadata",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, metadata)
}

// GetUserProfile retrieves user profile
// GET /api/v1/sbt/profile/:wallet
func (h *SBTHandler) GetUserProfile(c *gin.Context) {
	walletAddress := c.Param("wallet")

	// Validate wallet address format
	if !isValidWalletAddress(walletAddress) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid wallet address",
			"message": "Wallet address must be a valid Ethereum address",
		})
		return
	}

	// Get user profile
	profile, err := h.metadataService.GetUserProfile(c.Request.Context(), walletAddress)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get user profile",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateUserProfile updates user profile
// PUT /api/v1/sbt/profile/:wallet
func (h *SBTHandler) UpdateUserProfile(c *gin.Context) {
	walletAddress := c.Param("wallet")

	// Validate wallet address format
	if !isValidWalletAddress(walletAddress) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid wallet address",
			"message": "Wallet address must be a valid Ethereum address",
		})
		return
	}

	var req models.UpdateProfileRequest

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Set wallet address
	req.WalletAddress = walletAddress

	// Update user profile
	err := h.metadataService.UpdateUserProfile(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update user profile",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
	})
}

// validateRegistrationRequest validates registration request
func (h *SBTHandler) validateRegistrationRequest(req *models.UserRegistrationRequest) error {
	if req.WalletAddress == "" {
		return fmt.Errorf("wallet_address is required")
	}

	if req.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}

	if !isValidWalletAddress(req.WalletAddress) {
		return fmt.Errorf("invalid wallet address format")
	}

	// Validate avatar data (choose one)
	if req.AvatarBase64 == "" && req.ImageURL == "" {
		return fmt.Errorf("either avatar_base64 or image_url is required")
	}

	// Validate inviter address format (if provided)
	if req.InviteFrom != "" && !isValidWalletAddress(req.InviteFrom) {
		return fmt.Errorf("invalid invite_from address format")
	}

	// Validate invitee address format (if provided)
	for _, invitee := range req.InviteTo {
		if !isValidWalletAddress(invitee) {
			return fmt.Errorf("invalid invite_to address format: %s", invitee)
		}
	}

	return nil
}

// isValidWalletAddress validates Ethereum wallet address format
func isValidWalletAddress(address string) bool {
	if len(address) != 42 {
		return false
	}

	if !strings.HasPrefix(address, "0x") {
		return false
	}

	// Check if valid hexadecimal characters
	for _, char := range address[2:] {
		if !((char >= '0' && char <= '9') ||
			(char >= 'a' && char <= 'f') ||
			(char >= 'A' && char <= 'F')) {
			return false
		}
	}

	return true
}

// UpdateInviteRelation updates invitation relationship
// PUT /api/v1/sbt/invite/:wallet
func (h *SBTHandler) UpdateInviteRelation(c *gin.Context) {
	walletAddress := c.Param("wallet")
	if !isValidWalletAddress(walletAddress) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid wallet address format",
			"message": "Wallet address must be a valid Ethereum address",
		})
		return
	}

	var req models.UpdateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Ensure path parameter and request body wallet addresses match
	req.WalletAddress = walletAddress

	// Validate inviter address format (if provided)
	if req.InviteFrom != "" && !isValidWalletAddress(req.InviteFrom) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid invite_from address format",
			"message": "invite_from must be a valid Ethereum address",
		})
		return
	}

	// Validate invitee address format
	for _, addr := range req.InviteTo {
		if !isValidWalletAddress(addr) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid invite_to address format",
				"message": fmt.Sprintf("Address %s is not a valid Ethereum address", addr),
			})
			return
		}
	}

	// Update invitation relationship
	err := h.metadataService.UpdateInviteRelation(c.Request.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update invite relation",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Invite relation updated successfully",
	})
}

// getContractAddress retrieves contract address
func (h *SBTHandler) getContractAddress() string {
	return os.Getenv("SBT_CONTRACT_ADDRESS")
}
