package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// DailyDistributionService handles daily distribution logging and checking
type DailyDistributionService struct {
	db *sql.DB
}

// NewDailyDistributionService creates a new daily distribution service
func NewDailyDistributionService(db *sql.DB) *DailyDistributionService {
	return &DailyDistributionService{
		db: db,
	}
}

// DailyDistributionLog represents daily distribution log
type DailyDistributionLog struct {
	ID                     int64      `json:"id" db:"id"`
	DistributionDate       string     `json:"distribution_date" db:"distribution_date"` // YYYY-MM-DD format
	Status                 string     `json:"status" db:"status"`                       // 'completed', 'failed', 'in_progress'
	TotalPointsDistributed int        `json:"total_points_distributed" db:"total_points_distributed"`
	TotalUsersAffected     int        `json:"total_users_affected" db:"total_users_affected"`
	TotalTasksProcessed    int        `json:"total_tasks_processed" db:"total_tasks_processed"`
	StartedAt              time.Time  `json:"started_at" db:"started_at"`
	CompletedAt            *time.Time `json:"completed_at" db:"completed_at"`
	ErrorMessage           string     `json:"error_message" db:"error_message"`
	Metadata               string     `json:"metadata" db:"metadata"` // JSON string
}

// CheckDailyDistribution checks if daily distribution has been completed for a given date
func (dds *DailyDistributionService) CheckDailyDistribution(ctx context.Context, date time.Time) (*DailyDistributionLog, error) {
	dateStr := date.Format("2006-01-02")

	query := `
		SELECT id, distribution_date, status, total_points_distributed, total_users_affected, 
		       total_tasks_processed, started_at, completed_at, error_message, metadata
		FROM daily_distribution_log 
		WHERE distribution_date = ?
	`

	var log DailyDistributionLog
	var completedAt sql.NullTime
	var errorMessage, metadata sql.NullString

	err := dds.db.QueryRowContext(ctx, query, dateStr).Scan(
		&log.ID,
		&log.DistributionDate,
		&log.Status,
		&log.TotalPointsDistributed,
		&log.TotalUsersAffected,
		&log.TotalTasksProcessed,
		&log.StartedAt,
		&completedAt,
		&errorMessage,
		&metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No distribution found for this date
		}
		return nil, fmt.Errorf("failed to check daily distribution: %v", err)
	}

	// Handle nullable fields
	if completedAt.Valid {
		log.CompletedAt = &completedAt.Time
	}
	if errorMessage.Valid {
		log.ErrorMessage = errorMessage.String
	}
	if metadata.Valid {
		log.Metadata = metadata.String
	}

	return &log, nil
}

// StartDailyDistribution starts a new daily distribution
func (dds *DailyDistributionService) StartDailyDistribution(ctx context.Context, date time.Time) (*DailyDistributionLog, error) {
	dateStr := date.Format("2006-01-02")

	// Check if already exists
	existing, err := dds.CheckDailyDistribution(ctx, date)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		if existing.Status == "completed" {
			return nil, fmt.Errorf("daily distribution for %s already completed", dateStr)
		}
		if existing.Status == "in_progress" {
			return nil, fmt.Errorf("daily distribution for %s already in progress", dateStr)
		}
		// If failed, we can restart
		log.Printf("Restarting failed daily distribution for %s", dateStr)
	}

	// Create or update distribution log
	distributionLog := &DailyDistributionLog{
		DistributionDate:       dateStr,
		Status:                 "in_progress",
		TotalPointsDistributed: 0,
		TotalUsersAffected:     0,
		TotalTasksProcessed:    0,
		StartedAt:              time.Now(),
		CompletedAt:            nil,
		ErrorMessage:           "",
		Metadata:               "",
	}

	if existing != nil {
		// Update existing record
		distributionLog.ID = existing.ID
		err = dds.updateDistributionLog(ctx, distributionLog)
	} else {
		// Insert new record
		err = dds.insertDistributionLog(ctx, distributionLog)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to start daily distribution: %v", err)
	}

	log.Printf("Started daily distribution for %s", dateStr)
	return distributionLog, nil
}

// CompleteDailyDistribution marks daily distribution as completed
func (dds *DailyDistributionService) CompleteDailyDistribution(ctx context.Context, logID int64, totalPoints, totalUsers, totalTasks int, metadata map[string]interface{}) error {
	now := time.Now()
	metadataJSON, _ := json.Marshal(metadata)

	query := `
		UPDATE daily_distribution_log 
		SET status = 'completed', 
		    total_points_distributed = ?, 
		    total_users_affected = ?, 
		    total_tasks_processed = ?,
		    completed_at = ?, 
		    metadata = ?
		WHERE id = ?
	`

	_, err := dds.db.ExecContext(ctx, query, totalPoints, totalUsers, totalTasks, now, string(metadataJSON), logID)
	if err != nil {
		return fmt.Errorf("failed to complete daily distribution: %v", err)
	}

	log.Printf("Completed daily distribution (ID: %d) - Points: %d, Users: %d, Tasks: %d", logID, totalPoints, totalUsers, totalTasks)
	return nil
}

// FailDailyDistribution marks daily distribution as failed
func (dds *DailyDistributionService) FailDailyDistribution(ctx context.Context, logID int64, errorMsg string) error {
	query := `
		UPDATE daily_distribution_log 
		SET status = 'failed', error_message = ?
		WHERE id = ?
	`

	_, err := dds.db.ExecContext(ctx, query, errorMsg, logID)
	if err != nil {
		return fmt.Errorf("failed to mark distribution as failed: %v", err)
	}

	log.Printf("Marked daily distribution (ID: %d) as failed: %s", logID, errorMsg)
	return nil
}

// insertDistributionLog inserts a new distribution log
func (dds *DailyDistributionService) insertDistributionLog(ctx context.Context, log *DailyDistributionLog) error {
	query := `
		INSERT INTO daily_distribution_log (
			distribution_date, status, total_points_distributed, total_users_affected, 
			total_tasks_processed, started_at, completed_at, error_message, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := dds.db.ExecContext(ctx, query,
		log.DistributionDate,
		log.Status,
		log.TotalPointsDistributed,
		log.TotalUsersAffected,
		log.TotalTasksProcessed,
		log.StartedAt,
		log.CompletedAt,
		log.ErrorMessage,
		log.Metadata,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	log.ID = id
	return nil
}

// updateDistributionLog updates an existing distribution log
func (dds *DailyDistributionService) updateDistributionLog(ctx context.Context, log *DailyDistributionLog) error {
	query := `
		UPDATE daily_distribution_log 
		SET status = ?, total_points_distributed = ?, total_users_affected = ?, 
		    total_tasks_processed = ?, started_at = ?, completed_at = ?, 
		    error_message = ?, metadata = ?
		WHERE id = ?
	`

	_, err := dds.db.ExecContext(ctx, query,
		log.Status,
		log.TotalPointsDistributed,
		log.TotalUsersAffected,
		log.TotalTasksProcessed,
		log.StartedAt,
		log.CompletedAt,
		log.ErrorMessage,
		log.Metadata,
		log.ID,
	)

	return err
}

// GetDistributionHistory gets distribution history
func (dds *DailyDistributionService) GetDistributionHistory(ctx context.Context, limit int) ([]*DailyDistributionLog, error) {
	query := `
		SELECT id, distribution_date, status, total_points_distributed, total_users_affected, 
		       total_tasks_processed, started_at, completed_at, error_message, metadata
		FROM daily_distribution_log 
		ORDER BY distribution_date DESC 
		LIMIT ?
	`

	rows, err := dds.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution history: %v", err)
	}
	defer rows.Close()

	var logs []*DailyDistributionLog
	for rows.Next() {
		var log DailyDistributionLog
		var completedAt sql.NullTime
		var errorMessage, metadata sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.DistributionDate,
			&log.Status,
			&log.TotalPointsDistributed,
			&log.TotalUsersAffected,
			&log.TotalTasksProcessed,
			&log.StartedAt,
			&completedAt,
			&errorMessage,
			&metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan distribution log: %v", err)
		}

		// Handle nullable fields
		if completedAt.Valid {
			log.CompletedAt = &completedAt.Time
		}
		if errorMessage.Valid {
			log.ErrorMessage = errorMessage.String
		}
		if metadata.Valid {
			log.Metadata = metadata.String
		}

		logs = append(logs, &log)
	}

	return logs, nil
}

// CleanupOldLogs cleans up old distribution logs (keep last 30 days)
func (dds *DailyDistributionService) CleanupOldLogs(ctx context.Context) (int, error) {
	query := `DELETE FROM daily_distribution_log WHERE distribution_date < DATE_SUB(CURDATE(), INTERVAL 30 DAY)`

	result, err := dds.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old logs: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}
