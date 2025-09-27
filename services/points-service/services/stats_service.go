package services

import (
	"context"
	"database/sql"
	"fmt"
)

// StatsService handles statistics and queries
type StatsService struct {
	db *sql.DB
}

// NewStatsService creates a new stats service
func NewStatsService(db *sql.DB) *StatsService {
	return &StatsService{
		db: db,
	}
}

// SubnetStats represents subnet statistics
type SubnetStats struct {
	SubnetID               string `json:"subnet_id" db:"subnet_id"`
	SubnetName             string `json:"subnet_name" db:"subnet_name"`
	SubnetIcon             string `json:"subnet_icon" db:"subnet_icon"`
	CreatorWallet          string `json:"creator_wallet" db:"creator_wallet"`
	TotalTasks             int    `json:"total_tasks" db:"total_tasks"`
	CompletedTasks         int    `json:"completed_tasks" db:"completed_tasks"`
	UniqueUsers            int    `json:"unique_users" db:"unique_users"`
	TotalPointsDistributed int    `json:"total_points_distributed" db:"total_points_distributed"`
	TodayPointsDistributed int    `json:"today_points_distributed" db:"today_points_distributed"`
	TodayActiveUsers       int    `json:"today_active_users" db:"today_active_users"`
}

// UserRanking represents user points ranking
type UserRanking struct {
	Ranking             int    `json:"ranking" db:"ranking"`
	WalletAddress       string `json:"wallet_address" db:"wallet_address"`
	DisplayName         string `json:"display_name" db:"display_name"`
	TotalPoints         int    `json:"total_points" db:"total_points"`
	TodayContribution   int    `json:"today_contribution" db:"today_contribution"`
	CompletedTasksCount int    `json:"completed_tasks_count" db:"completed_tasks_count"`
	ActiveSubnetsCount  int    `json:"active_subnets_count" db:"active_subnets_count"`
	HasNFT              bool   `json:"has_nft" db:"has_nft"`
}

// PointsHistoryItem represents a points history record
type PointsHistoryItem struct {
	ID          int64  `json:"id" db:"id"`
	Date        string `json:"date" db:"date"`
	Source      string `json:"source" db:"source"`
	Points      int    `json:"points" db:"points"`
	Description string `json:"description" db:"description"`
	TxRef       string `json:"tx_ref" db:"tx_ref"`
	SubnetID    string `json:"subnet_id" db:"subnet_id"`
	SubnetName  string `json:"subnet_name" db:"subnet_name"`
	CreatedAt   string `json:"created_at" db:"created_at"`
}

// OverallStats represents comprehensive system statistics
type OverallStats struct {
	TotalPoints      int     `json:"total_points"`
	TotalUsers       int     `json:"total_users"`
	TotalSubnets     int     `json:"total_subnets"`
	TotalTasks       int     `json:"total_tasks"`
	CompletedTasks   int     `json:"completed_tasks"`
	TodayPoints      int     `json:"today_points"`
	TodayActiveUsers int     `json:"today_active_users"`
	UsersWithNFT     int     `json:"users_with_nft"`
	AvgPointsPerUser float64 `json:"avg_points_per_user"`
}

// GetTotalPoints gets total points across all users
func (ss *StatsService) GetTotalPoints(ctx context.Context) (int, error) {
	query := `SELECT COALESCE(SUM(total_points), 0) FROM user_profiles`

	var totalPoints int
	err := ss.db.QueryRowContext(ctx, query).Scan(&totalPoints)
	if err != nil {
		return 0, fmt.Errorf("failed to get total points: %v", err)
	}

	return totalPoints, nil
}

// GetTotalUsers gets total number of registered users
func (ss *StatsService) GetTotalUsers(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM user_profiles`

	var totalUsers int
	err := ss.db.QueryRowContext(ctx, query).Scan(&totalUsers)
	if err != nil {
		return 0, fmt.Errorf("failed to get total users: %v", err)
	}

	return totalUsers, nil
}

// GetSubnetStats gets subnet statistics
func (ss *StatsService) GetSubnetStats(ctx context.Context) ([]*SubnetStats, error) {
	query := `SELECT * FROM subnet_stats ORDER BY total_points_distributed DESC`

	rows, err := ss.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query subnet stats: %v", err)
	}
	defer rows.Close()

	var stats []*SubnetStats
	for rows.Next() {
		var stat SubnetStats
		err := rows.Scan(
			&stat.SubnetID,
			&stat.SubnetName,
			&stat.SubnetIcon,
			&stat.CreatorWallet,
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

// GetSubnetDetails gets detailed statistics for a specific subnet
func (ss *StatsService) GetSubnetDetails(ctx context.Context, subnetID string) (*SubnetStats, error) {
	query := `SELECT * FROM subnet_stats WHERE subnet_id = ?`

	var stat SubnetStats
	err := ss.db.QueryRowContext(ctx, query, subnetID).Scan(
		&stat.SubnetID,
		&stat.SubnetName,
		&stat.SubnetIcon,
		&stat.CreatorWallet,
		&stat.TotalTasks,
		&stat.CompletedTasks,
		&stat.UniqueUsers,
		&stat.TotalPointsDistributed,
		&stat.TodayPointsDistributed,
		&stat.TodayActiveUsers,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("subnet not found")
		}
		return nil, fmt.Errorf("failed to get subnet details: %v", err)
	}

	return &stat, nil
}

// GetSubnetPointsToday gets points distributed today for a subnet
func (ss *StatsService) GetSubnetPointsToday(ctx context.Context, subnetID string) (int, error) {
	query := `
		SELECT COALESCE(SUM(ph.points), 0)
		FROM points_history ph
		WHERE ph.subnet_id = ? AND DATE(ph.created_at) = CURDATE()
	`

	var todayPoints int
	err := ss.db.QueryRowContext(ctx, query, subnetID).Scan(&todayPoints)
	if err != nil {
		return 0, fmt.Errorf("failed to get today's subnet points: %v", err)
	}

	return todayPoints, nil
}

// GetSubnetUsersCount gets number of users who completed tasks in a subnet
func (ss *StatsService) GetSubnetUsersCount(ctx context.Context, subnetID string) (int, error) {
	query := `
		SELECT COUNT(DISTINCT utc.user_wallet)
		FROM user_task_completions utc
		WHERE utc.subnet_id = ?
	`

	var usersCount int
	err := ss.db.QueryRowContext(ctx, query, subnetID).Scan(&usersCount)
	if err != nil {
		return 0, fmt.Errorf("failed to get subnet users count: %v", err)
	}

	return usersCount, nil
}

// GetUserPointsRanking gets user points ranking with pagination
func (ss *StatsService) GetUserPointsRanking(ctx context.Context, limit, offset int) ([]*UserRanking, int, error) {
	// Get total count first
	countQuery := `SELECT COUNT(*) FROM user_profiles`
	var totalCount int
	err := ss.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %v", err)
	}

	// Get ranking data
	query := `SELECT * FROM user_points_ranking LIMIT ? OFFSET ?`

	rows, err := ss.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user ranking: %v", err)
	}
	defer rows.Close()

	var rankings []*UserRanking
	for rows.Next() {
		var ranking UserRanking
		err := rows.Scan(
			&ranking.WalletAddress,
			&ranking.DisplayName,
			&ranking.TotalPoints,
			&ranking.TodayContribution,
			&ranking.Ranking,
			&ranking.CompletedTasksCount,
			&ranking.ActiveSubnetsCount,
			&ranking.HasNFT,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user ranking: %v", err)
		}
		rankings = append(rankings, &ranking)
	}

	return rankings, totalCount, nil
}

// GetUserPointsHistory gets detailed points history for a user
func (ss *StatsService) GetUserPointsHistory(ctx context.Context, userWallet string, limit, offset int) ([]*PointsHistoryItem, int, error) {
	// Get total count first
	countQuery := `SELECT COUNT(*) FROM points_history WHERE wallet_address = ?`
	var totalCount int
	err := ss.db.QueryRowContext(ctx, countQuery, userWallet).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get history count: %v", err)
	}

	// Get history data
	query := `SELECT * FROM points_history_detailed WHERE wallet_address = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := ss.db.QueryContext(ctx, query, userWallet, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query points history: %v", err)
	}
	defer rows.Close()

	var history []*PointsHistoryItem
	for rows.Next() {
		var item PointsHistoryItem
		var displayName, subnetName, description sql.NullString
		var subnetID, txRef sql.NullString

		err := rows.Scan(
			&item.ID,
			&item.WalletAddress,
			&displayName,
			&item.Date,
			&item.Source,
			&item.Points,
			&item.TxRef,
			&subnetID,
			&subnetName,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan points history: %v", err)
		}

		// Handle nullable fields
		if subnetID.Valid {
			item.SubnetID = subnetID.String
		}
		if subnetName.Valid {
			item.SubnetName = subnetName.String
		}
		if description.Valid {
			item.Description = description.String
		}
		if txRef.Valid {
			item.TxRef = txRef.String
		}

		history = append(history, &item)
	}

	return history, totalCount, nil
}

// GetOverallStats gets comprehensive system statistics
func (ss *StatsService) GetOverallStats(ctx context.Context) (*OverallStats, error) {
	stats := &OverallStats{}

	// Get total points
	totalPoints, err := ss.GetTotalPoints(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalPoints = totalPoints

	// Get total users
	totalUsers, err := ss.GetTotalUsers(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = totalUsers

	// Get total subnets
	err = ss.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM subnets WHERE status = 'active'`).Scan(&stats.TotalSubnets)
	if err != nil {
		return nil, fmt.Errorf("failed to get total subnets: %v", err)
	}

	// Get total tasks
	err = ss.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&stats.TotalTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get total tasks: %v", err)
	}

	// Get completed tasks
	err = ss.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE status = 'CONFIRMED'`).Scan(&stats.CompletedTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed tasks: %v", err)
	}

	// Get today's points
	err = ss.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(points), 0) FROM points_history WHERE DATE(created_at) = CURDATE()`).Scan(&stats.TodayPoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's points: %v", err)
	}

	// Get today's active users
	err = ss.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT wallet_address) FROM points_history WHERE DATE(created_at) = CURDATE()`).Scan(&stats.TodayActiveUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's active users: %v", err)
	}

	// Get users with NFT
	err = ss.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nft_ownership_cache WHERE has_nft = true AND expires_at > NOW()`).Scan(&stats.UsersWithNFT)
	if err != nil {
		// NFT table might not exist yet, so don't fail
		stats.UsersWithNFT = 0
	}

	// Calculate average points per user
	if stats.TotalUsers > 0 {
		stats.AvgPointsPerUser = float64(stats.TotalPoints) / float64(stats.TotalUsers)
	}

	return stats, nil
}
