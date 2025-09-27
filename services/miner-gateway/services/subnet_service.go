package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// SubnetService manages subnet operations
type SubnetService struct {
	db *sql.DB
}

// NewSubnetService creates a new subnet service
func NewSubnetService(db *sql.DB) *SubnetService {
	return &SubnetService{
		db: db,
	}
}

// FindOrCreateSubnet finds existing subnet by name or creates a new one
func (ss *SubnetService) FindOrCreateSubnet(ctx context.Context, req *models.SubnetCreateRequest) (*models.Subnet, error) {
	// First, try to find existing subnet by name
	existing, err := ss.GetSubnetByName(ctx, req.Name)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("error checking existing subnet: %v", err)
	}

	if existing != nil {
		log.Printf("Found existing subnet: %s (ID: %s)", existing.Name, existing.ID)
		return existing, nil
	}

	// Create new subnet
	subnet := &models.Subnet{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Icon:          req.Icon,
		CreatorWallet: req.CreatorWallet,
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = ss.CreateSubnet(ctx, subnet)
	if err != nil {
		return nil, fmt.Errorf("error creating subnet: %v", err)
	}

	log.Printf("Created new subnet: %s (ID: %s) by %s", subnet.Name, subnet.ID, subnet.CreatorWallet)
	return subnet, nil
}

// CreateSubnet creates a new subnet
func (ss *SubnetService) CreateSubnet(ctx context.Context, subnet *models.Subnet) error {
	query := `
		INSERT INTO subnets (id, name, icon, creator_wallet, created_at, updated_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := ss.db.ExecContext(ctx, query,
		subnet.ID,
		subnet.Name,
		subnet.Icon,
		subnet.CreatorWallet,
		subnet.CreatedAt,
		subnet.UpdatedAt,
		subnet.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to create subnet: %v", err)
	}

	return nil
}

// GetSubnetByName gets subnet by name
func (ss *SubnetService) GetSubnetByName(ctx context.Context, name string) (*models.Subnet, error) {
	query := `
		SELECT id, name, icon, creator_wallet, created_at, updated_at, status
		FROM subnets 
		WHERE name = ? AND status = 'active'
	`

	var subnet models.Subnet
	err := ss.db.QueryRowContext(ctx, query, name).Scan(
		&subnet.ID,
		&subnet.Name,
		&subnet.Icon,
		&subnet.CreatorWallet,
		&subnet.CreatedAt,
		&subnet.UpdatedAt,
		&subnet.Status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subnet by name: %v", err)
	}

	return &subnet, nil
}

// GetSubnetByID gets subnet by ID
func (ss *SubnetService) GetSubnetByID(ctx context.Context, id string) (*models.Subnet, error) {
	query := `
		SELECT id, name, icon, creator_wallet, created_at, updated_at, status
		FROM subnets 
		WHERE id = ?
	`

	var subnet models.Subnet
	err := ss.db.QueryRowContext(ctx, query, id).Scan(
		&subnet.ID,
		&subnet.Name,
		&subnet.Icon,
		&subnet.CreatorWallet,
		&subnet.CreatedAt,
		&subnet.UpdatedAt,
		&subnet.Status,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get subnet by ID: %v", err)
	}

	return &subnet, nil
}

// ListSubnets lists all active subnets
func (ss *SubnetService) ListSubnets(ctx context.Context) ([]*models.Subnet, error) {
	query := `
		SELECT id, name, icon, creator_wallet, created_at, updated_at, status
		FROM subnets 
		WHERE status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := ss.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets: %v", err)
	}
	defer rows.Close()

	var subnets []*models.Subnet
	for rows.Next() {
		var subnet models.Subnet
		err := rows.Scan(
			&subnet.ID,
			&subnet.Name,
			&subnet.Icon,
			&subnet.CreatorWallet,
			&subnet.CreatedAt,
			&subnet.UpdatedAt,
			&subnet.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subnet: %v", err)
		}
		subnets = append(subnets, &subnet)
	}

	return subnets, nil
}

// GetSubnetStats gets subnet statistics
func (ss *SubnetService) GetSubnetStats(ctx context.Context) ([]*models.SubnetStats, error) {
	query := `SELECT * FROM subnet_stats ORDER BY total_points_distributed DESC`

	rows, err := ss.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get subnet stats: %v", err)
	}
	defer rows.Close()

	var stats []*models.SubnetStats
	for rows.Next() {
		var stat models.SubnetStats
		err := rows.Scan(
			&stat.SubnetID,
			&stat.SubnetName,
			&stat.SubnetIcon,
			&stat.CreatorWallet,
			&stat.SubnetCreatedAt,
			&stat.TotalTasks,
			&stat.CompletedTasks,
			&stat.UniqueUsers,
			&stat.TotalPointsDistributed,
			&stat.TodayPointsDistributed,
			&stat.TodayActiveUsers,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subnet stats: %v", err)
		}
		stats = append(stats, &stat)
	}

	return stats, nil
}

// GetSubnetStatsByID gets subnet statistics by ID
func (ss *SubnetService) GetSubnetStatsByID(ctx context.Context, subnetID string) (*models.SubnetStats, error) {
	query := `SELECT * FROM subnet_stats WHERE subnet_id = ?`

	var stat models.SubnetStats
	err := ss.db.QueryRowContext(ctx, query, subnetID).Scan(
		&stat.SubnetID,
		&stat.SubnetName,
		&stat.SubnetIcon,
		&stat.CreatorWallet,
		&stat.SubnetCreatedAt,
		&stat.TotalTasks,
		&stat.CompletedTasks,
		&stat.UniqueUsers,
		&stat.TotalPointsDistributed,
		&stat.TodayPointsDistributed,
		&stat.TodayActiveUsers,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get subnet stats by ID: %v", err)
	}

	return &stat, nil
}

// RecordTaskCompletion records user task completion
func (ss *SubnetService) RecordTaskCompletion(ctx context.Context, completion *models.UserTaskCompletion) error {
	query := `
		INSERT INTO user_task_completions (user_wallet, task_id, subnet_id, completed_at, vlc_increment, points_earned)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			completed_at = VALUES(completed_at),
			vlc_increment = VALUES(vlc_increment),
			points_earned = VALUES(points_earned)
	`

	_, err := ss.db.ExecContext(ctx, query,
		completion.UserWallet,
		completion.TaskID,
		completion.SubnetID,
		completion.CompletedAt,
		completion.VLCIncrement,
		completion.PointsEarned,
	)

	if err != nil {
		return fmt.Errorf("failed to record task completion: %v", err)
	}

	return nil
}

// CheckTaskCompletion checks if user has completed a task
func (ss *SubnetService) CheckTaskCompletion(ctx context.Context, userWallet, taskID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM user_task_completions 
		WHERE user_wallet = ? AND task_id = ?
	`

	var count int
	err := ss.db.QueryRowContext(ctx, query, userWallet, taskID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check task completion: %v", err)
	}

	return count > 0, nil
}

// GetTotalSubnetsCount gets total number of subnets
func (ss *SubnetService) GetTotalSubnetsCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM subnets WHERE status = 'active'`

	var count int
	err := ss.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total subnets count: %v", err)
	}

	return count, nil
}

// GetSubnetPointsToday gets points distributed today for a subnet
func (ss *SubnetService) GetSubnetPointsToday(ctx context.Context, subnetID string) (int, error) {
	query := `
		SELECT COALESCE(SUM(points_earned), 0) 
		FROM user_task_completions 
		WHERE subnet_id = ? AND DATE(completed_at) = CURDATE()
	`

	var points int
	err := ss.db.QueryRowContext(ctx, query, subnetID).Scan(&points)
	if err != nil {
		return 0, fmt.Errorf("failed to get subnet points today: %v", err)
	}

	return points, nil
}

// GetSubnetUsersCount gets number of users who completed tasks in a subnet
func (ss *SubnetService) GetSubnetUsersCount(ctx context.Context, subnetID string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT user_wallet) 
		FROM user_task_completions 
		WHERE subnet_id = ?
	`

	var count int
	err := ss.db.QueryRowContext(ctx, query, subnetID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get subnet users count: %v", err)
	}

	return count, nil
}
