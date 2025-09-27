package services

import (
	"context"
	"fmt"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// TaskExpiryService handles task expiry logic
type TaskExpiryService struct {
	taskService *TaskService
}

// NewTaskExpiryService creates a new task expiry service
func NewTaskExpiryService(taskService *TaskService) *TaskExpiryService {
	return &TaskExpiryService{
		taskService: taskService,
	}
}

// CheckAndMarkExpiredTasks checks and marks expired tasks
func (tes *TaskExpiryService) CheckAndMarkExpiredTasks(ctx context.Context, tasks []*models.Task) ([]*models.Task, error) {
	var validTasks []*models.Task
	now := time.Now()

	for _, task := range tasks {
		// Check if task is expired
		if task.ExpiresAt != nil && now.After(*task.ExpiresAt) {
			// Mark as expired if not already expired
			if task.Status != models.TaskExpired {
				task.Status = models.TaskExpired
				task.UpdatedAt = now

				// Update in database
				if err := tes.taskService.updateTaskStatus(ctx, task.ID, models.TaskExpired); err != nil {
					// Log error but continue processing other tasks
					fmt.Printf("Failed to mark task %s as expired: %v\n", task.ID, err)
				} else {
					fmt.Printf("Task %s marked as expired (expired at: %v)\n", task.ID, *task.ExpiresAt)
				}
			}
			// Skip expired tasks from processing
			continue
		}

		// Task is still valid
		validTasks = append(validTasks, task)
	}

	return validTasks, nil
}

// IsTaskExpired checks if a single task is expired
func (tes *TaskExpiryService) IsTaskExpired(task *models.Task) bool {
	if task.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*task.ExpiresAt)
}

// FilterExpiredTasks filters out expired tasks without updating database
func (tes *TaskExpiryService) FilterExpiredTasks(tasks []*models.Task) []*models.Task {
	var validTasks []*models.Task
	now := time.Now()

	for _, task := range tasks {
		if task.ExpiresAt == nil || now.Before(*task.ExpiresAt) {
			validTasks = append(validTasks, task)
		}
	}

	return validTasks
}

// GetExpiryInfo returns expiry information for a task
func (tes *TaskExpiryService) GetExpiryInfo(task *models.Task) map[string]interface{} {
	info := map[string]interface{}{
		"has_expiry": task.ExpiresAt != nil,
		"is_expired": false,
	}

	if task.ExpiresAt != nil {
		info["expires_at"] = *task.ExpiresAt
		info["is_expired"] = time.Now().After(*task.ExpiresAt)

		if !info["is_expired"].(bool) {
			remaining := time.Until(*task.ExpiresAt)
			info["time_remaining"] = remaining.String()
			info["days_remaining"] = int(remaining.Hours() / 24)
		}
	}

	return info
}
