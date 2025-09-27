package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// UpdateTaskWithSubnetAndExpiry updates task with subnet and expiry information
func (ts *TaskService) UpdateTaskWithSubnetAndExpiry(ctx context.Context, taskID, subnetID string, expiresAt *time.Time) error {
	query := `UPDATE tasks SET subnet_id = ?, expires_at = ?, updated_at = ? WHERE id = ?`
	_, err := ts.db.ExecContext(ctx, query, subnetID, expiresAt, time.Now(), taskID)
	return err
}

// GetTasksWithExpiry gets tasks with expiry information
func (ts *TaskService) GetTasksWithExpiry(ctx context.Context, userWallet string, taskType models.TaskType, includeExpired bool) ([]*models.Task, error) {
	query := `
		SELECT id, user_wallet, task_type, status, payload, attempts, created_at, updated_at, 
		       completed_at, event_id, subnet_id, expires_at
		FROM tasks 
		WHERE user_wallet = ? AND task_type = ?
	`

	if !includeExpired {
		query += ` AND (expires_at IS NULL OR expires_at > NOW())`
	}

	query += ` ORDER BY created_at DESC`

	rows, err := ts.db.QueryContext(ctx, query, userWallet, taskType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		task := &models.Task{}
		var payloadJSON []byte
		var completedAt, expiresAt *time.Time
		var eventID, subnetID *string

		err := rows.Scan(
			&task.ID, &task.UserWallet, &task.TaskType, &task.Status,
			&payloadJSON, &task.Attempts, &task.CreatedAt, &task.UpdatedAt,
			&completedAt, &eventID, &subnetID, &expiresAt,
		)
		if err != nil {
			return nil, err
		}

		// Handle optional fields
		if completedAt != nil {
			task.CompletedAt = completedAt
		}
		if eventID != nil {
			task.EventID = *eventID
		}
		if subnetID != nil {
			task.SubnetID = *subnetID
		}
		if expiresAt != nil {
			task.ExpiresAt = expiresAt
		}

		// Unmarshal payload
		if err := json.Unmarshal(payloadJSON, &task.Payload); err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}
