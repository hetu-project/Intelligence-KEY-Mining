package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/services"
)

// BatchVerificationHandler handles batch verification operations
// Note: Batch verification is an operation, not a task type
// TODO: Will be redesigned in step 4 to handle batch operations properly
type BatchVerificationHandler struct {
	taskService   *services.TaskService
	batchVerifier *services.BatchVerifier
}

// NewBatchVerificationHandler creates a new batch verification handler
func NewBatchVerificationHandler(taskService *services.TaskService, batchVerifier *services.BatchVerifier) *BatchVerificationHandler {
	return &BatchVerificationHandler{
		taskService:   taskService,
		batchVerifier: batchVerifier,
	}
}

// BatchVerifyTasks handles batch verification request
// TODO: Redesign this as batch operation handler in step 4
func (bvh *BatchVerificationHandler) BatchVerifyTasks(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": "Batch verification temporarily disabled - being redesigned as operation",
		"code":  "SERVICE_UNAVAILABLE",
	})
}

// GetBatchVerificationStatus gets the status of a batch verification
// TODO: Redesign this as batch operation status handler in step 4
func (bvh *BatchVerificationHandler) GetBatchVerificationStatus(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": "Batch verification status temporarily disabled - being redesigned as operation",
		"code":  "SERVICE_UNAVAILABLE",
	})
}

// ListBatchVerifications lists batch verifications for a user
// TODO: Redesign this as batch operation list handler in step 4
func (bvh *BatchVerificationHandler) ListBatchVerifications(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": "Batch verification list temporarily disabled - being redesigned as operation",
		"code":  "SERVICE_UNAVAILABLE",
	})
}

// GetBatchVerificationStats gets batch verification statistics
// TODO: Redesign this as batch operation stats handler in step 4
func (bvh *BatchVerificationHandler) GetBatchVerificationStats(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": "Batch verification stats temporarily disabled - being redesigned as operation",
		"code":  "SERVICE_UNAVAILABLE",
	})
}
