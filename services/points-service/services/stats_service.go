package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
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
	ID            int64  `json:"id" db:"id"`
	WalletAddress string `json:"wallet_address" db:"wallet_address"`
	Date          string `json:"date" db:"date"`
	Source        string `json:"source" db:"source"`
	Points        int    `json:"points" db:"points"`
	Description   string `json:"description" db:"description"`
	TxRef         string `json:"tx_ref" db:"tx_ref"`
	SubnetID      string `json:"subnet_id" db:"subnet_id"`
	SubnetName    string `json:"subnet_name" db:"subnet_name"`
	CreatedAt     string `json:"created_at" db:"created_at"`
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

// SubnetLeader represents top user in a subnet
type SubnetLeader struct {
	SubnetID       string `json:"subnet_id" db:"subnet_id"`
	SubnetName     string `json:"subnet_name" db:"subnet_name"`
	SubnetIcon     string `json:"subnet_icon" db:"subnet_icon"`
	UserWallet     string `json:"user_wallet" db:"user_wallet"`
	DisplayName    string `json:"display_name" db:"display_name"`
	TotalPoints    int    `json:"total_points" db:"total_points"`
	CompletedTasks int    `json:"completed_tasks" db:"completed_tasks"`
	Rank           int    `json:"rank"`
}

// SubnetUserRank represents user ranking in a specific subnet
type SubnetUserRank struct {
	UserWallet     string `json:"user_wallet" db:"user_wallet"`
	DisplayName    string `json:"display_name" db:"display_name"`
	TotalPoints    int    `json:"total_points" db:"total_points"`
	CompletedTasks int    `json:"completed_tasks" db:"completed_tasks"`
	Rank           int    `json:"rank" db:"user_rank"`
}

// DailyPerformer represents user who completed tasks today
type DailyPerformer struct {
	UserWallet     string `json:"user_wallet" db:"user_wallet"`
	DisplayName    string `json:"display_name" db:"display_name"`
	TasksCompleted int    `json:"tasks_completed" db:"tasks_completed"`
	PointsEarned   int    `json:"points_earned" db:"points_earned"`
	Rank           int    `json:"rank"`
}

// DashboardStats represents comprehensive dashboard statistics
type DashboardStats struct {
	TotalPoints      int     `json:"total_points"`
	TotalUsers       int     `json:"total_users"`
	ActiveMiners     int     `json:"active_miners"`
	TotalSubnets     int     `json:"total_subnets"`
	TodayPoints      int     `json:"today_points"`
	ActiveTasksCount int     `json:"active_tasks_count"`
	TodayActiveUsers int     `json:"today_active_users"`
	UsersWithNFT     int     `json:"users_with_nft"`
	AvgPointsPerUser float64 `json:"avg_points_per_user"`
}

// DailyPointsData represents daily points data for a specific date
type DailyPointsData struct {
	Date        string `json:"date"`         // "2025-11-03"
	Points      int    `json:"points"`       // Total points for the day
	ActiveUsers int    `json:"active_users"` // Number of active users for the day
}

// SubnetDailyPoints represents subnet with 7-day daily points statistics
type SubnetDailyPoints struct {
	SubnetID           string            `json:"subnet_id"`
	SubnetName         string            `json:"subnet_name"`
	SubnetIcon         string            `json:"subnet_icon"`
	DailyPoints        []DailyPointsData `json:"daily_points"`        // Daily task points
	Total7Days         int               `json:"total_7days"`         // Total including overrides
	AvgPerDay          float64           `json:"avg_per_day"`         // Average per day
	OverrideAdjustment int               `json:"override_adjustment"` // Total override adjustment
	TaskPointsSum      int               `json:"task_points_sum"`     // Sum of actual task points
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
	query := `
		SELECT 
			s.id as subnet_id,
			s.name as subnet_name,
			s.icon as subnet_icon,
			s.creator_wallet,
			-- Total tasks: count all tasks in this subnet
			COUNT(DISTINCT t.id) as total_tasks,
			-- Completed tasks: count tasks that have points distributed (from points_history)
			COALESCE((
				SELECT COUNT(DISTINCT 
					CASE 
						WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
						ELSE ph.tx_ref
					END
				)
				FROM points_history ph 
				LEFT JOIN tasks t2 ON (
					CASE 
						WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
						ELSE ph.tx_ref
					END = t2.id
				)
				WHERE (t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci OR ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci)
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Chat Task', 'Register QR Code Task', 'Creator Commission')
			), 0) as completed_tasks,
			-- Unique users: count all users who got points for this subnet
			-- Use UNION to avoid double counting
			COALESCE((
				SELECT COUNT(DISTINCT wallet_address) FROM (
					SELECT ph.wallet_address
					FROM points_history ph 
					WHERE ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
					AND ph.source IN ('Chat Task', 'Creator Commission')
					UNION
					SELECT ph.wallet_address
					FROM points_history ph 
					INNER JOIN tasks t2 ON (
						CASE 
							WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
							ELSE ph.tx_ref
						END = t2.id
					)
					WHERE t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci
					AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Register QR Code Task')
				) AS all_users
			), 0) as unique_users,
			-- Total points distributed: sum all points for this subnet (including adjustments)
			-- Split into two parts to avoid double counting
			COALESCE((
				SELECT SUM(ph.points) 
				FROM points_history ph 
				WHERE ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
				AND ph.source IN ('Chat Task', 'Creator Commission')
			), 0) + COALESCE((
				SELECT SUM(ph.points) 
				FROM points_history ph 
				INNER JOIN tasks t2 ON (
					CASE 
						WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
						ELSE ph.tx_ref
					END = t2.id
				)
				WHERE t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Register QR Code Task')
			), 0) as total_points_distributed,
			-- Today points distributed: sum today's points for this subnet (NOT including adjustments)
			-- Split into two parts to avoid double counting
			COALESCE((
				SELECT SUM(ph.points) 
				FROM points_history ph 
				WHERE ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
				AND DATE(ph.created_at) = CURDATE()
				AND ph.source IN ('Chat Task', 'Creator Commission')
			), 0) + COALESCE((
				SELECT SUM(ph.points) 
				FROM points_history ph 
				INNER JOIN tasks t2 ON (
					CASE 
						WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
						ELSE ph.tx_ref
					END = t2.id
				)
				WHERE t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci
				AND DATE(ph.created_at) = CURDATE()
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Register QR Code Task')
			), 0) as today_points_distributed,
			-- Today active users: count users who got points today for this subnet
			-- Use UNION to avoid double counting
			COALESCE((
				SELECT COUNT(DISTINCT wallet_address) FROM (
					SELECT ph.wallet_address
					FROM points_history ph 
					WHERE ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
					AND DATE(ph.created_at) = CURDATE()
					AND ph.source IN ('Chat Task', 'Creator Commission')
					UNION
					SELECT ph.wallet_address
					FROM points_history ph 
					INNER JOIN tasks t2 ON (
						CASE 
							WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
							ELSE ph.tx_ref
						END = t2.id
					)
					WHERE t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci
					AND DATE(ph.created_at) = CURDATE()
					AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Register QR Code Task')
				) AS today_users
			), 0) as today_active_users
		FROM subnets s
		LEFT JOIN tasks t ON s.id = t.subnet_id COLLATE utf8mb4_unicode_ci
		WHERE s.status = 'active'
		GROUP BY s.id, s.name, s.icon, s.creator_wallet
		ORDER BY total_points_distributed DESC
	`

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

		// Add adjustment points to total
		var adjustmentTotal int
		err = ss.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(adjustment_points), 0)
			FROM subnet_points_adjustment
			WHERE subnet_id = ?
		`, stat.SubnetID).Scan(&adjustmentTotal)
		if err == nil {
			stat.TotalPointsDistributed += adjustmentTotal
		}

		// Add today's adjustment points to today's total
		var todayAdjustmentTotal int
		err = ss.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(adjustment_points), 0)
			FROM subnet_points_adjustment
			WHERE subnet_id = ?
			AND DATE(created_at) = CURDATE()
		`, stat.SubnetID).Scan(&todayAdjustmentTotal)
		if err == nil {
			stat.TodayPointsDistributed += todayAdjustmentTotal
		}

		stats = append(stats, &stat)
	}

	return stats, nil
}

// GetSubnetDetails gets detailed statistics for a specific subnet
func (ss *StatsService) GetSubnetDetails(ctx context.Context, subnetID string) (*SubnetStats, error) {
	query := `
		SELECT 
			s.id as subnet_id,
			s.name as subnet_name,
			s.icon as subnet_icon,
			s.creator_wallet,
			-- Total tasks: count all tasks in this subnet
			COUNT(DISTINCT t.id) as total_tasks,
			-- Completed tasks: count tasks that have points distributed (from points_history)
			COALESCE((
				SELECT COUNT(DISTINCT 
					CASE 
						WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
						ELSE ph.tx_ref
					END
				)
				FROM points_history ph 
				LEFT JOIN tasks t2 ON (
					CASE 
						WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
						ELSE ph.tx_ref
					END = t2.id
				)
				WHERE (t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci OR ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci)
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Chat Task', 'Register QR Code Task', 'Creator Commission')
			), 0) as completed_tasks,
			-- Unique users: count all users who got points for this subnet
			-- Use UNION to avoid double counting
			COALESCE((
				SELECT COUNT(DISTINCT wallet_address) FROM (
					SELECT ph.wallet_address
					FROM points_history ph 
					WHERE ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
					AND ph.source IN ('Chat Task', 'Creator Commission')
					UNION
					SELECT ph.wallet_address
					FROM points_history ph 
					INNER JOIN tasks t2 ON (
						CASE 
							WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
							ELSE ph.tx_ref
						END = t2.id
					)
					WHERE t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci
					AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Register QR Code Task')
				) AS all_users
			), 0) as unique_users,
			-- Total points distributed: sum all points for this subnet (including adjustments)
			-- Split into two parts to avoid double counting
			COALESCE((
				SELECT SUM(ph.points) 
				FROM points_history ph 
				WHERE ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
				AND ph.source IN ('Chat Task', 'Creator Commission')
			), 0) + COALESCE((
				SELECT SUM(ph.points) 
				FROM points_history ph 
				INNER JOIN tasks t2 ON (
					CASE 
						WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
						ELSE ph.tx_ref
					END = t2.id
				)
				WHERE t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Register QR Code Task')
			), 0) as total_points_distributed,
			-- Today points distributed: sum today's points for this subnet (NOT including adjustments)
			-- Split into two parts to avoid double counting
			COALESCE((
				SELECT SUM(ph.points) 
				FROM points_history ph 
				WHERE ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
				AND DATE(ph.created_at) = CURDATE()
				AND ph.source IN ('Chat Task', 'Creator Commission')
			), 0) + COALESCE((
				SELECT SUM(ph.points) 
				FROM points_history ph 
				INNER JOIN tasks t2 ON (
					CASE 
						WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
						ELSE ph.tx_ref
					END = t2.id
				)
				WHERE t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci
				AND DATE(ph.created_at) = CURDATE()
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Register QR Code Task')
			), 0) as today_points_distributed,
			-- Today active users: count users who got points today for this subnet
			-- Use UNION to avoid double counting
			COALESCE((
				SELECT COUNT(DISTINCT wallet_address) FROM (
					SELECT ph.wallet_address
					FROM points_history ph 
					WHERE ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
					AND DATE(ph.created_at) = CURDATE()
					AND ph.source IN ('Chat Task', 'Creator Commission')
					UNION
					SELECT ph.wallet_address
					FROM points_history ph 
					INNER JOIN tasks t2 ON (
						CASE 
							WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
							ELSE ph.tx_ref
						END = t2.id
					)
					WHERE t2.subnet_id = s.id COLLATE utf8mb4_unicode_ci
					AND DATE(ph.created_at) = CURDATE()
					AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Twitter Follow Task', 'Telegram Task', 'Register QR Code Task')
				) AS today_users
			), 0) as today_active_users
		FROM subnets s
		LEFT JOIN tasks t ON s.id = t.subnet_id COLLATE utf8mb4_unicode_ci
		WHERE s.status = 'active' AND s.id = ?
		GROUP BY s.id, s.name, s.icon, s.creator_wallet
	`

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

	// Add adjustment points to total
	var adjustmentTotal int
	err = ss.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(adjustment_points), 0)
		FROM subnet_points_adjustment
		WHERE subnet_id = ?
	`, subnetID).Scan(&adjustmentTotal)
	if err == nil {
		stat.TotalPointsDistributed += adjustmentTotal
	}

	// Add today's adjustment points to today's total
	var todayAdjustmentTotal int
	err = ss.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(adjustment_points), 0)
		FROM subnet_points_adjustment
		WHERE subnet_id = ?
		AND DATE(created_at) = CURDATE()
	`, subnetID).Scan(&todayAdjustmentTotal)
	if err == nil {
		stat.TodayPointsDistributed += todayAdjustmentTotal
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

// GetUserSubnets gets subnets that a user has participated in (with override support)
func (ss *StatsService) GetUserSubnets(ctx context.Context, userWallet string) ([]*SubnetStats, error) {
	// Get distinct subnets the user has participated in
	query := `
		SELECT DISTINCT subnet_id FROM (
			-- From Chat Task
			SELECT DISTINCT ph.subnet_id
			FROM points_history ph
			WHERE ph.wallet_address = ?
				AND ph.subnet_id IS NOT NULL
				AND ph.source = 'Chat Task'
			
			UNION
			
			-- From other tasks
			SELECT DISTINCT t.subnet_id
			FROM points_history ph
			INNER JOIN tasks t ON (
				CASE 
					WHEN ph.tx_ref LIKE 'pocw-consensus-%' 
					THEN SUBSTRING(ph.tx_ref, 16)
					ELSE ph.tx_ref
				END = t.id COLLATE utf8mb4_unicode_ci
			)
			WHERE ph.wallet_address = ?
				AND t.subnet_id IS NOT NULL
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Twitter Follow Task', 'Register QR Code Task')
		) AS user_subnets
	`

	rows, err := ss.db.QueryContext(ctx, query, userWallet, userWallet)
	if err != nil {
		return nil, fmt.Errorf("failed to query user subnets: %v", err)
	}
	defer rows.Close()

	var subnetIDs []string
	for rows.Next() {
		var subnetID string
		if err := rows.Scan(&subnetID); err != nil {
			return nil, fmt.Errorf("failed to scan subnet ID: %v", err)
		}
		subnetIDs = append(subnetIDs, subnetID)
	}

	// For each subnet, get detailed stats
	var subnets []*SubnetStats
	for _, subnetID := range subnetIDs {
		// Get subnet info
		var subnet SubnetStats
		var status string
		err := ss.db.QueryRowContext(ctx, `
			SELECT id, name, icon, creator_wallet, status
			FROM subnets
			WHERE id = ? AND status = 'active'
		`, subnetID).Scan(
			&subnet.SubnetID,
			&subnet.SubnetName,
			&subnet.SubnetIcon,
			&subnet.CreatorWallet,
			&status,
		)
		if err != nil {
			continue // Skip if subnet not found or not active
		}

		// Get user's total points in this subnet (with adjustment)
		totalPoints, err := ss.GetUserSubnetTotalPoints(ctx, subnetID, userWallet)
		if err == nil {
			subnet.TotalPointsDistributed = totalPoints
		}

		// Count completed tasks for this user in this subnet
		var completedTasks int
		err = ss.db.QueryRowContext(ctx, `
			SELECT COUNT(DISTINCT tx_ref) FROM (
				SELECT ph.tx_ref
				FROM points_history ph
				WHERE ph.wallet_address = ?
					AND ph.subnet_id = ?
					AND ph.source = 'Chat Task'
				
				UNION ALL
				
				SELECT CASE 
					WHEN ph.tx_ref LIKE 'pocw-consensus-%' 
					THEN SUBSTRING(ph.tx_ref, 16)
					ELSE ph.tx_ref
				END as tx_ref
				FROM points_history ph
				INNER JOIN tasks t ON (
					CASE 
						WHEN ph.tx_ref LIKE 'pocw-consensus-%' 
						THEN SUBSTRING(ph.tx_ref, 16)
						ELSE ph.tx_ref
					END = t.id COLLATE utf8mb4_unicode_ci
				)
				WHERE ph.wallet_address = ?
					AND t.subnet_id = ?
					AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Twitter Follow Task', 'Register QR Code Task')
			) AS user_tasks
		`, userWallet, subnetID, userWallet, subnetID).Scan(&completedTasks)

		if err == nil {
			subnet.CompletedTasks = completedTasks
		}

		subnets = append(subnets, &subnet)
	}

	return subnets, nil
}

// GetUserSubnetSummary gets a summary of user's subnet participation (with override support)
func (ss *StatsService) GetUserSubnetSummary(ctx context.Context, userWallet string) (map[string]interface{}, error) {
	// Get user subnets (already includes override support)
	subnets, err := ss.GetUserSubnets(ctx, userWallet)
	if err != nil {
		return nil, err
	}

	participatedSubnets := len(subnets)

	// Calculate total points and completed tasks from subnets
	totalPointsEarned := 0
	completedTasks := 0
	for _, subnet := range subnets {
		totalPointsEarned += subnet.TotalPointsDistributed
		completedTasks += subnet.CompletedTasks
	}

	// Get today's points earned (from points_history)
	var todayPointsEarned int
	err = ss.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(points), 0)
		FROM points_history
		WHERE wallet_address = ?
			AND DATE(created_at) = CURDATE()
			AND source IN ('Chat Task', 'VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Register QR Code Task')
	`, userWallet).Scan(&todayPointsEarned)

	if err != nil {
		todayPointsEarned = 0 // Default to 0 if query fails
	}

	return map[string]interface{}{
		"user_wallet":          userWallet,
		"participated_subnets": participatedSubnets,
		"completed_tasks":      completedTasks,
		"total_points_earned":  totalPointsEarned,
		"today_points_earned":  todayPointsEarned,
		"subnets":              subnets,
	}, nil
}

// GetSubnetUserRanking gets user ranking by points in a specific subnet
func (ss *StatsService) GetSubnetUserRanking(ctx context.Context, subnetID string, limit, offset int) ([]*SubnetUserRank, int, error) {
	// Get total count of users in this subnet from points_history
	// Use UNION to count users from both Chat Task (direct subnet_id) and other tasks (via tasks table)
	countQuery := `
		SELECT COUNT(DISTINCT wallet_address) FROM (
			SELECT DISTINCT ph.wallet_address
			FROM points_history ph
			WHERE ph.subnet_id = ?
				AND ph.source = 'Chat Task'
			
			UNION
			
			SELECT DISTINCT ph.wallet_address
			FROM points_history ph
			INNER JOIN tasks t ON (
				CASE 
					WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
					ELSE ph.tx_ref
				END = t.id COLLATE utf8mb4_unicode_ci
			)
			WHERE t.subnet_id = ?
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Twitter Follow Task', 'Register QR Code Task')
		) AS all_users
	`
	var totalCount int
	err := ss.db.QueryRowContext(ctx, countQuery, subnetID, subnetID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get subnet users count: %v", err)
	}

	// Get user ranking based on points_history with adjustment support
	// Use UNION ALL to aggregate points from both Chat Task and other tasks
	// Total = task_points + adjustment_points
	query := `
		SELECT 
			ranked_users.user_wallet,
			up.display_name,
			ranked_users.total_points,
			ranked_users.completed_tasks,
			ranked_users.user_rank
		FROM (
			SELECT 
				user_wallet,
				task_points + COALESCE(adjustment_points, 0) as total_points,
				completed_tasks,
				ROW_NUMBER() OVER (ORDER BY (task_points + COALESCE(adjustment_points, 0)) DESC) as user_rank
			FROM (
				-- Aggregate task points from both Chat and other tasks
				SELECT 
					wallet_address as user_wallet,
					SUM(total_points) as task_points,
					SUM(completed_tasks) as completed_tasks
				FROM (
					-- Chat Task: direct subnet_id in points_history
					SELECT 
						ph.wallet_address,
						SUM(ph.points) as total_points,
						COUNT(DISTINCT ph.tx_ref) as completed_tasks
					FROM points_history ph
					WHERE ph.subnet_id = ?
						AND ph.source = 'Chat Task'
					GROUP BY ph.wallet_address
					
					UNION ALL
					
					-- Other tasks: subnet_id in tasks table
					SELECT 
						ph.wallet_address,
						SUM(ph.points) as total_points,
						COUNT(DISTINCT 
							CASE 
								WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
								ELSE ph.tx_ref
							END
						) as completed_tasks
					FROM points_history ph
					INNER JOIN tasks t ON (
						CASE 
							WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
							ELSE ph.tx_ref
						END = t.id COLLATE utf8mb4_unicode_ci
					)
					WHERE t.subnet_id = ?
						AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Twitter Follow Task', 'Register QR Code Task')
					GROUP BY ph.wallet_address
				) AS all_tasks
				GROUP BY wallet_address
			) AS task_summary
			LEFT JOIN (
				-- Get adjustment points for each user
				SELECT 
					wallet_address,
					SUM(adjustment_points) as adjustment_points
				FROM subnet_points_adjustment
				WHERE subnet_id = ?
				GROUP BY wallet_address
			) AS adjustments ON task_summary.user_wallet = adjustments.wallet_address COLLATE utf8mb4_unicode_ci
		) ranked_users
		LEFT JOIN user_profiles up ON ranked_users.user_wallet = up.wallet_address COLLATE utf8mb4_unicode_ci
		ORDER BY ranked_users.user_rank
		LIMIT ? OFFSET ?
	`

	rows, err := ss.db.QueryContext(ctx, query, subnetID, subnetID, subnetID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query subnet user ranking: %v", err)
	}
	defer rows.Close()

	var rankings []*SubnetUserRank
	for rows.Next() {
		var ranking SubnetUserRank
		var displayName sql.NullString

		err := rows.Scan(
			&ranking.UserWallet,
			&displayName,
			&ranking.TotalPoints,
			&ranking.CompletedTasks,
			&ranking.Rank,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan subnet user ranking row: %v", err)
		}

		if displayName.Valid {
			ranking.DisplayName = displayName.String
		}

		rankings = append(rankings, &ranking)
	}

	return rankings, totalCount, nil
}

// GetActiveMinersCount gets count of users who have participated in retweet tasks (all time)
func (ss *StatsService) GetActiveMinersCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(DISTINCT user_wallet) FROM user_task_completions`

	var activeMiners int
	err := ss.db.QueryRowContext(ctx, query).Scan(&activeMiners)
	if err != nil {
		return 0, fmt.Errorf("failed to get active miners count: %v", err)
	}

	return activeMiners, nil
}

// GetActiveTasksCount gets count of active (non-expired) tasks
func (ss *StatsService) GetActiveTasksCount(ctx context.Context) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM tasks 
		WHERE expires_at > NOW() AND status != 'EXPIRED'
	`

	var activeTasks int
	err := ss.db.QueryRowContext(ctx, query).Scan(&activeTasks)
	if err != nil {
		return 0, fmt.Errorf("failed to get active tasks count: %v", err)
	}

	return activeTasks, nil
}

// GetSubnetLeaders gets top 3 users by points in each subnet with pagination (with override support)
func (ss *StatsService) GetSubnetLeaders(ctx context.Context, limit, offset int) ([]*SubnetLeader, int, error) {
	// Get total count of subnets first
	countQuery := `SELECT COUNT(*) FROM subnets WHERE status = 'active'`
	var totalCount int
	err := ss.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get subnets count: %v", err)
	}

	// Get top 3 users per subnet with adjustment support
	query := `
		SELECT 
			ranked_users.subnet_id,
			s.name as subnet_name,
			s.icon as subnet_icon,
			ranked_users.user_wallet,
			up.display_name,
			ranked_users.total_points,
			ranked_users.completed_tasks,
			ranked_users.rank_in_subnet
		FROM (
			SELECT 
				subnet_id,
				user_wallet,
				task_points + COALESCE(adjustment_points, 0) as total_points,
				completed_tasks,
				ROW_NUMBER() OVER (
					PARTITION BY subnet_id 
					ORDER BY (task_points + COALESCE(adjustment_points, 0)) DESC
				) as rank_in_subnet
			FROM (
				-- Aggregate task points from both Chat and other tasks
				SELECT 
					subnet_id,
					wallet_address as user_wallet,
					SUM(total_points) as task_points,
					SUM(completed_tasks) as completed_tasks
				FROM (
					-- Chat Task: direct subnet_id in points_history
					SELECT 
						ph.subnet_id,
						ph.wallet_address,
						SUM(ph.points) as total_points,
						COUNT(DISTINCT ph.tx_ref) as completed_tasks
					FROM points_history ph
					WHERE ph.subnet_id IS NOT NULL
						AND ph.source = 'Chat Task'
					GROUP BY ph.subnet_id, ph.wallet_address
					
					UNION ALL
					
					-- Other tasks: subnet_id in tasks table
					SELECT 
						t.subnet_id,
						ph.wallet_address,
						SUM(ph.points) as total_points,
						COUNT(DISTINCT 
							CASE 
								WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
								ELSE ph.tx_ref
							END
						) as completed_tasks
					FROM points_history ph
					INNER JOIN tasks t ON (
						CASE 
							WHEN ph.tx_ref LIKE 'pocw-consensus-%' THEN SUBSTRING(ph.tx_ref, 16)
							ELSE ph.tx_ref
						END = t.id COLLATE utf8mb4_unicode_ci
					)
					WHERE t.subnet_id IS NOT NULL
						AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Twitter Follow Task', 'Register QR Code Task')
					GROUP BY t.subnet_id, ph.wallet_address
				) AS all_tasks
				GROUP BY subnet_id, wallet_address
			) AS task_summary
			LEFT JOIN (
				-- Get adjustment points for each user
				SELECT 
					subnet_id,
					wallet_address,
					SUM(adjustment_points) as adjustment_points
				FROM subnet_points_adjustment
				GROUP BY subnet_id, wallet_address
			) AS adjustments ON task_summary.subnet_id = adjustments.subnet_id COLLATE utf8mb4_unicode_ci
				AND task_summary.user_wallet = adjustments.wallet_address COLLATE utf8mb4_unicode_ci
		) ranked_users
		INNER JOIN subnets s ON ranked_users.subnet_id = s.id COLLATE utf8mb4_unicode_ci
		LEFT JOIN user_profiles up ON ranked_users.user_wallet = up.wallet_address COLLATE utf8mb4_unicode_ci
		WHERE ranked_users.rank_in_subnet <= 3 AND s.status = 'active'
		ORDER BY s.name, ranked_users.rank_in_subnet
		LIMIT ? OFFSET ?
	`

	rows, err := ss.db.QueryContext(ctx, query, limit*3, offset*3) // limit*3 because we want top 3 per subnet
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query subnet leaders: %v", err)
	}
	defer rows.Close()

	var leaders []*SubnetLeader
	for rows.Next() {
		var leader SubnetLeader
		var displayName sql.NullString

		err := rows.Scan(
			&leader.SubnetID,
			&leader.SubnetName,
			&leader.SubnetIcon,
			&leader.UserWallet,
			&displayName,
			&leader.TotalPoints,
			&leader.CompletedTasks,
			&leader.Rank,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan subnet leader: %v", err)
		}

		if displayName.Valid {
			leader.DisplayName = displayName.String
		} else {
			leader.DisplayName = leader.UserWallet[:8] + "..." // Fallback display name
		}

		leaders = append(leaders, &leader)
	}

	return leaders, totalCount, nil
}

// GetDailyPerformers gets users who completed tasks today, ranked by task count
func (ss *StatsService) GetDailyPerformers(ctx context.Context, limit, offset int) ([]*DailyPerformer, int, error) {
	// Get total count first
	countQuery := `
		SELECT COUNT(DISTINCT user_wallet) 
		FROM user_task_completions 
		WHERE DATE(completed_at) = CURDATE()
	`
	var totalCount int
	err := ss.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get daily performers count: %v", err)
	}

	// Get daily performers ranked by tasks completed
	query := `
		SELECT 
			utc.user_wallet,
			up.display_name,
			COUNT(utc.task_id) as tasks_completed,
			COALESCE(SUM(utc.points_earned), 0) as points_earned,
			ROW_NUMBER() OVER (ORDER BY COUNT(utc.task_id) DESC, SUM(utc.points_earned) DESC) as rank_num
		FROM user_task_completions utc
		LEFT JOIN user_profiles up ON utc.user_wallet = up.wallet_address
		WHERE DATE(utc.completed_at) = CURDATE()
		GROUP BY utc.user_wallet, up.display_name
		ORDER BY tasks_completed DESC, points_earned DESC
		LIMIT ? OFFSET ?
	`

	rows, err := ss.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query daily performers: %v", err)
	}
	defer rows.Close()

	var performers []*DailyPerformer
	for rows.Next() {
		var performer DailyPerformer
		var displayName sql.NullString

		err := rows.Scan(
			&performer.UserWallet,
			&displayName,
			&performer.TasksCompleted,
			&performer.PointsEarned,
			&performer.Rank,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan daily performer: %v", err)
		}

		if displayName.Valid {
			performer.DisplayName = displayName.String
		} else {
			performer.DisplayName = performer.UserWallet[:8] + "..." // Fallback display name
		}

		performers = append(performers, &performer)
	}

	return performers, totalCount, nil
}

// GetDashboardStats gets comprehensive dashboard statistics
func (ss *StatsService) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}

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

	// Get active miners count
	activeMiners, err := ss.GetActiveMinersCount(ctx)
	if err != nil {
		return nil, err
	}
	stats.ActiveMiners = activeMiners

	// Get active tasks count
	activeTasks, err := ss.GetActiveTasksCount(ctx)
	if err != nil {
		return nil, err
	}
	stats.ActiveTasksCount = activeTasks

	// Get other stats from existing overall stats
	overallStats, err := ss.GetOverallStats(ctx)
	if err != nil {
		return nil, err
	}

	stats.TotalSubnets = overallStats.TotalSubnets
	stats.TodayPoints = overallStats.TodayPoints
	stats.TodayActiveUsers = overallStats.TodayActiveUsers
	stats.UsersWithNFT = overallStats.UsersWithNFT
	stats.AvgPointsPerUser = overallStats.AvgPointsPerUser

	return stats, nil
}

// GetSubnetsDailyPoints gets daily points for all subnets over the past N days
func (ss *StatsService) GetSubnetsDailyPoints(ctx context.Context, days int, subnetID string) ([]*SubnetDailyPoints, error) {
	if days <= 0 || days > 30 {
		days = 7 // Default to 7 days, max 30
	}

	// Build WHERE clause for optional subnet filter
	// Note: We need the days parameter twice (for each query in UNION)
	subnetFilter := ""
	daysParam := days - 1
	var args []interface{}

	if subnetID != "" {
		subnetFilter = "AND s.id = ?"
		// Parameters: days for query1, subnet_id for query1, days for query2, subnet_id for query2
		args = []interface{}{daysParam, subnetID, daysParam, subnetID}
	} else {
		// Parameters: days for query1, days for query2
		args = []interface{}{daysParam, daysParam}
	}

	// Query daily points data using UNION ALL for better performance
	// Split Chat Task and other tasks into separate queries
	query := fmt.Sprintf(`
		SELECT 
			s.id as subnet_id,
			s.name as subnet_name,
			s.icon as subnet_icon,
			DATE(ph.created_at) as date,
			SUM(ph.points) as points,
			COUNT(DISTINCT ph.wallet_address) as active_users
		FROM subnets s
		INNER JOIN points_history ph ON ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
			AND ph.source IN ('Chat Task', 'Creator Commission')
			AND ph.created_at IS NOT NULL
			AND DATE(ph.created_at) BETWEEN DATE_SUB(CURDATE(), INTERVAL ? DAY) AND CURDATE()
		WHERE s.status = 'active' %s
		GROUP BY s.id, s.name, s.icon, DATE(ph.created_at)
		
		UNION ALL
		
		SELECT 
			s.id as subnet_id,
			s.name as subnet_name,
			s.icon as subnet_icon,
			DATE(ph.created_at) as date,
			SUM(ph.points) as points,
			COUNT(DISTINCT ph.wallet_address) as active_users
		FROM subnets s
		INNER JOIN tasks t ON t.subnet_id = s.id COLLATE utf8mb4_unicode_ci
		INNER JOIN points_history ph ON (
			ph.subnet_id IS NULL
			AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Twitter Follow Task', 'Register QR Code Task')
			AND ph.created_at IS NOT NULL
			AND DATE(ph.created_at) BETWEEN DATE_SUB(CURDATE(), INTERVAL ? DAY) AND CURDATE()
			AND t.id = CASE 
				WHEN ph.tx_ref LIKE 'pocw-consensus-%%' 
				THEN SUBSTRING(ph.tx_ref, 16)
				ELSE ph.tx_ref
			END
		)
		WHERE s.status = 'active' %s
		GROUP BY s.id, s.name, s.icon, DATE(ph.created_at)
		
		ORDER BY subnet_id, date
	`, subnetFilter, subnetFilter)

	rows, err := ss.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query subnet daily points: %v", err)
	}
	defer rows.Close()

	// Organize data by subnet and date (merge UNION ALL results)
	type dateKey struct {
		subnetID string
		date     string
	}
	dateDataMap := make(map[dateKey]*DailyPointsData)
	subnetInfoMap := make(map[string]struct {
		name string
		icon string
	})

	for rows.Next() {
		var subnetID, subnetName, subnetIcon string
		var date sql.NullString
		var points, activeUsers int

		err := rows.Scan(&subnetID, &subnetName, &subnetIcon, &date, &points, &activeUsers)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// Store subnet info
		if _, exists := subnetInfoMap[subnetID]; !exists {
			subnetInfoMap[subnetID] = struct {
				name string
				icon string
			}{name: subnetName, icon: subnetIcon}
		}

		// Merge data for same subnet+date (from UNION ALL results)
		if date.Valid {
			// Extract only date part (YYYY-MM-DD) from datetime string
			dateStr := date.String
			if len(dateStr) >= 10 {
				dateStr = dateStr[:10]
			}

			key := dateKey{subnetID: subnetID, date: dateStr}
			if existing, exists := dateDataMap[key]; exists {
				// Merge with existing data
				existing.Points += points
				existing.ActiveUsers += activeUsers
			} else {
				// Create new entry
				dateDataMap[key] = &DailyPointsData{
					Date:        dateStr,
					Points:      points,
					ActiveUsers: activeUsers,
				}
			}
		}
	}

	// Organize by subnet
	subnetMap := make(map[string]*SubnetDailyPoints)
	for key, data := range dateDataMap {
		if _, exists := subnetMap[key.subnetID]; !exists {
			info := subnetInfoMap[key.subnetID]
			subnetMap[key.subnetID] = &SubnetDailyPoints{
				SubnetID:    key.subnetID,
				SubnetName:  info.name,
				SubnetIcon:  info.icon,
				DailyPoints: []DailyPointsData{},
			}
		}
		subnetMap[key.subnetID].DailyPoints = append(subnetMap[key.subnetID].DailyPoints, *data)
	}

	// Get end date from database (to match MySQL timezone)
	var dbDate string
	err = ss.db.QueryRowContext(ctx, "SELECT CURDATE()").Scan(&dbDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get current date: %v", err)
	}

	// Fill missing dates and calculate totals (with override support)
	result := make([]*SubnetDailyPoints, 0, len(subnetMap))
	for _, subnet := range subnetMap {
		// Fill missing dates with zeros (using database date)
		subnet.DailyPoints = fillMissingDatesFromEnd(subnet.DailyPoints, days, dbDate)

		// Calculate task points sum
		taskPointsSum := 0
		for _, daily := range subnet.DailyPoints {
			taskPointsSum += daily.Points
		}
		subnet.TaskPointsSum = taskPointsSum

		// Calculate adjustment points for this subnet (新系统：直接累加所有调整积分)
		adjustmentTotal := 0
		adjustmentQuery := `
			SELECT COALESCE(SUM(adjustment_points), 0)
			FROM subnet_points_adjustment
			WHERE subnet_id = ?
		`

		err := ss.db.QueryRowContext(ctx, adjustmentQuery, subnet.SubnetID).Scan(&adjustmentTotal)
		if err != nil {
			adjustmentTotal = 0 // Ignore errors, default to 0
		}

		subnet.OverrideAdjustment = adjustmentTotal // 保留字段名以保持API兼容性
		subnet.Total7Days = taskPointsSum + adjustmentTotal

		if days > 0 {
			subnet.AvgPerDay = float64(subnet.Total7Days) / float64(days)
		}

		result = append(result, subnet)
	}

	return result, nil
}

// GetChatTasksDailyPoints gets daily points for chat tasks in all subnets over the past N days
func (ss *StatsService) GetChatTasksDailyPoints(ctx context.Context, days int, subnetID string) ([]*SubnetDailyPoints, error) {
	if days <= 0 || days > 30 {
		days = 7 // Default to 7 days, max 30
	}

	// Build WHERE clause for optional subnet filter
	subnetFilter := ""
	daysParam := days - 1
	var args []interface{}

	if subnetID != "" {
		subnetFilter = "AND s.id = ?"
		args = []interface{}{daysParam, subnetID}
	} else {
		args = []interface{}{daysParam}
	}

	// Query chat task daily points data
	// Only query Chat Task source with direct subnet_id
	query := fmt.Sprintf(`
		SELECT 
			s.id as subnet_id,
			s.name as subnet_name,
			s.icon as subnet_icon,
			DATE(ph.created_at) as date,
			SUM(ph.points) as points,
			COUNT(DISTINCT ph.wallet_address) as chat_users
		FROM subnets s
		INNER JOIN points_history ph ON ph.subnet_id = s.id COLLATE utf8mb4_unicode_ci
			AND ph.source = 'Chat Task'
			AND ph.created_at IS NOT NULL
			AND DATE(ph.created_at) BETWEEN DATE_SUB(CURDATE(), INTERVAL ? DAY) AND CURDATE()
		WHERE s.status = 'active' %s
		GROUP BY s.id, s.name, s.icon, DATE(ph.created_at)
		ORDER BY s.id, date
	`, subnetFilter)

	rows, err := ss.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat tasks daily points: %v", err)
	}
	defer rows.Close()

	// Organize data by subnet and date
	type dateKey struct {
		subnetID string
		date     string
	}
	dateDataMap := make(map[dateKey]*DailyPointsData)
	subnetInfoMap := make(map[string]struct {
		name string
		icon string
	})

	for rows.Next() {
		var subnetID, subnetName, subnetIcon string
		var date sql.NullString
		var points, chatUsers int

		err := rows.Scan(&subnetID, &subnetName, &subnetIcon, &date, &points, &chatUsers)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// Store subnet info
		if _, exists := subnetInfoMap[subnetID]; !exists {
			subnetInfoMap[subnetID] = struct {
				name string
				icon string
			}{name: subnetName, icon: subnetIcon}
		}

		// Store daily data
		if date.Valid {
			// Extract only date part (YYYY-MM-DD) from datetime string
			dateStr := date.String
			if len(dateStr) >= 10 {
				dateStr = dateStr[:10]
			}

			dateDataMap[dateKey{subnetID: subnetID, date: dateStr}] = &DailyPointsData{
				Date:        dateStr,
				Points:      points,
				ActiveUsers: chatUsers, // This is the number of chat users per day
			}
		}
	}

	// Get end date from database (to match MySQL timezone)
	var dbDate string
	err = ss.db.QueryRowContext(ctx, "SELECT CURDATE()").Scan(&dbDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get current date: %v", err)
	}

	// Organize by subnet
	subnetMap := make(map[string]*SubnetDailyPoints)
	for key, data := range dateDataMap {
		if _, exists := subnetMap[key.subnetID]; !exists {
			info := subnetInfoMap[key.subnetID]
			subnetMap[key.subnetID] = &SubnetDailyPoints{
				SubnetID:    key.subnetID,
				SubnetName:  info.name,
				SubnetIcon:  info.icon,
				DailyPoints: []DailyPointsData{},
			}
		}
		subnetMap[key.subnetID].DailyPoints = append(subnetMap[key.subnetID].DailyPoints, *data)
	}

	// Fill missing dates and calculate totals (with override support - chat tasks only)
	result := make([]*SubnetDailyPoints, 0, len(subnetMap))
	for _, subnet := range subnetMap {
		// Fill missing dates with zeros (using database date)
		subnet.DailyPoints = fillMissingDatesFromEnd(subnet.DailyPoints, days, dbDate)

		// Calculate task points sum (chat tasks only)
		taskPointsSum := 0
		for _, daily := range subnet.DailyPoints {
			taskPointsSum += daily.Points
		}
		subnet.TaskPointsSum = taskPointsSum

		// Calculate override adjustments for this subnet
		// Note: For chat tasks endpoint, we still calculate total override adjustment
		// Calculate adjustment points for this subnet (新系统：直接累加所有调整积分)
		adjustmentTotal := 0
		adjustmentQuery := `
			SELECT COALESCE(SUM(adjustment_points), 0)
			FROM subnet_points_adjustment
			WHERE subnet_id = ?
		`

		err := ss.db.QueryRowContext(ctx, adjustmentQuery, subnet.SubnetID).Scan(&adjustmentTotal)
		if err != nil {
			adjustmentTotal = 0 // Ignore errors, default to 0
		}

		subnet.OverrideAdjustment = adjustmentTotal // 保留字段名以保持API兼容性
		subnet.Total7Days = taskPointsSum + adjustmentTotal

		if days > 0 {
			subnet.AvgPerDay = float64(subnet.Total7Days) / float64(days)
		}

		result = append(result, subnet)
	}

	return result, nil
}

// fillMissingDatesFromEnd fills missing dates in daily points data with zeros
// Uses the provided end date (from database) to ensure timezone consistency
func fillMissingDatesFromEnd(dailyPoints []DailyPointsData, days int, endDate string) []DailyPointsData {
	// Parse end date
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		// Fallback to current time if parse fails
		end = time.Now()
	}

	// Create a map of existing dates
	dataMap := make(map[string]DailyPointsData)
	for _, data := range dailyPoints {
		dataMap[data.Date] = data
	}

	// Generate all dates for the past N days from end date
	result := make([]DailyPointsData, days)
	for i := days - 1; i >= 0; i-- {
		date := end.AddDate(0, 0, -i).Format("2006-01-02")
		if data, exists := dataMap[date]; exists {
			result[days-1-i] = data
		} else {
			result[days-1-i] = DailyPointsData{
				Date:        date,
				Points:      0,
				ActiveUsers: 0,
			}
		}
	}

	return result
}

// ============================================
// Points Override System
// ============================================

// PointsOverride represents a points override record
type PointsOverride struct {
	ID             int       `json:"id" db:"id"`
	SubnetID       string    `json:"subnet_id" db:"subnet_id"`
	WalletAddress  string    `json:"wallet_address" db:"wallet_address"`
	OverridePoints int       `json:"override_points" db:"override_points"`
	AdminWallet    string    `json:"admin_wallet" db:"admin_wallet"`
	Reason         string    `json:"reason" db:"reason"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// GetUserSubnetTotalPoints calculates user's total points in a subnet (considering override)
// GetUserSubnetTotalPoints 计算用户在 subnet 的总积分（包含调整）
func (ss *StatsService) GetUserSubnetTotalPoints(ctx context.Context, subnetID, walletAddress string) (int, error) {
	// 计算历史任务积分
	var taskPoints int
	err := ss.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(points), 0) FROM (
			-- Chat Task: direct subnet_id
			SELECT SUM(ph.points) as points
			FROM points_history ph
			WHERE ph.wallet_address = ?
				AND ph.subnet_id = ?
				AND ph.source = 'Chat Task'
			
			UNION ALL
			
			-- Other tasks: subnet_id in tasks table
			SELECT SUM(ph.points) as points
			FROM points_history ph
			INNER JOIN tasks t ON (
				CASE 
					WHEN ph.tx_ref LIKE 'pocw-consensus-%' 
					THEN SUBSTRING(ph.tx_ref, 16)
					ELSE ph.tx_ref
				END = t.id COLLATE utf8mb4_unicode_ci
			)
			WHERE ph.wallet_address = ?
				AND t.subnet_id = ?
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Twitter Follow Task', 'Register QR Code Task')
		) AS all_points
	`, walletAddress, subnetID, walletAddress, subnetID).Scan(&taskPoints)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate task points: %v", err)
	}

	// 计算所有调整积分（增量累加）
	var adjustmentPoints int
	err = ss.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(adjustment_points), 0)
		FROM subnet_points_adjustment
		WHERE subnet_id = ?
			AND wallet_address = ?
	`, subnetID, walletAddress).Scan(&adjustmentPoints)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate adjustment points: %v", err)
	}

	return taskPoints + adjustmentPoints, nil
}

// ==================== 新的积分调整系统（Adjustment，增量方式）====================

// PointsAdjustment 积分调整记录
type PointsAdjustment struct {
	ID               int       `json:"id"`
	SubnetID         string    `json:"subnet_id"`
	WalletAddress    string    `json:"wallet_address"`
	AdjustmentPoints int       `json:"adjustment_points"`
	AdminWallet      string    `json:"admin_wallet"`
	Reason           string    `json:"reason"`
	CreatedAt        time.Time `json:"created_at"`
}

// AddPointsAdjustment 添加积分调整（增量方式）
func (ss *StatsService) AddPointsAdjustment(ctx context.Context, subnetID, walletAddress string, adjustmentPoints int, adminWallet, reason string) error {
	// 验证备注长度（最多20个中文字符 = 60字节），可为空
	if reason != "" && len(reason) > 60 {
		return fmt.Errorf("reason too long: max 20 Chinese characters (60 bytes)")
	}

	query := `
		INSERT INTO subnet_points_adjustment (
			subnet_id, wallet_address, adjustment_points, admin_wallet, reason, created_at
		) VALUES (?, ?, ?, ?, ?, NOW())
	`

	_, err := ss.db.ExecContext(ctx, query, subnetID, walletAddress, adjustmentPoints, adminWallet, reason)
	if err != nil {
		return fmt.Errorf("failed to add points adjustment: %v", err)
	}

	return nil
}

// GetAdjustmentHistory 获取用户在 subnet 的调整历史
func (ss *StatsService) GetAdjustmentHistory(ctx context.Context, subnetID, walletAddress string, limit int) ([]*PointsAdjustment, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	query := `
		SELECT id, subnet_id, wallet_address, adjustment_points, admin_wallet, reason, created_at
		FROM subnet_points_adjustment
		WHERE subnet_id = ?
			AND wallet_address = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := ss.db.QueryContext(ctx, query, subnetID, walletAddress, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query adjustment history: %v", err)
	}
	defer rows.Close()

	var adjustments []*PointsAdjustment
	for rows.Next() {
		var adj PointsAdjustment
		err := rows.Scan(&adj.ID, &adj.SubnetID, &adj.WalletAddress, &adj.AdjustmentPoints, &adj.AdminWallet, &adj.Reason, &adj.CreatedAt)
		if err != nil {
			continue
		}
		adjustments = append(adjustments, &adj)
	}

	return adjustments, nil
}

// SearchUserByWallet 精确搜索钱包地址并返回用户在指定 subnet 的积分
func (ss *StatsService) SearchUserByWallet(ctx context.Context, subnetID, walletAddress string) (map[string]interface{}, error) {
	// 1. 查询用户基本信息
	var displayName sql.NullString
	err := ss.db.QueryRowContext(ctx, `
		SELECT COALESCE(display_name, '')
		FROM user_profiles
		WHERE wallet_address = ?
	`, walletAddress).Scan(&displayName)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", walletAddress)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %v", err)
	}

	// 2. 计算用户在该 subnet 的总积分（包含调整）
	totalPoints, err := ss.GetUserSubnetTotalPoints(ctx, subnetID, walletAddress)
	if err != nil {
		return nil, err
	}

	// 3. 查询调整积分总和
	var adjustmentTotal int
	ss.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(adjustment_points), 0)
		FROM subnet_points_adjustment
		WHERE subnet_id = ?
			AND wallet_address = ?
	`, subnetID, walletAddress).Scan(&adjustmentTotal)

	// 4. 查询任务积分
	taskPoints := totalPoints - adjustmentTotal

	// 5. 查询完成任务数
	var completedTasks int
	ss.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT DISTINCT ph.tx_ref
			FROM points_history ph
			WHERE ph.wallet_address = ?
				AND ph.subnet_id = ?
				AND ph.source = 'Chat Task'
			
			UNION
			
			SELECT DISTINCT ph.tx_ref
			FROM points_history ph
			INNER JOIN tasks t ON (
				CASE 
					WHEN ph.tx_ref LIKE 'pocw-consensus-%' 
					THEN SUBSTRING(ph.tx_ref, 16)
					ELSE ph.tx_ref
				END = t.id COLLATE utf8mb4_unicode_ci
			)
			WHERE ph.wallet_address = ?
				AND t.subnet_id = ?
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Twitter Follow Task', 'Register QR Code Task')
		) AS tasks
	`, walletAddress, subnetID, walletAddress, subnetID).Scan(&completedTasks)

	result := map[string]interface{}{
		"wallet_address":    walletAddress,
		"display_name":      displayName.String,
		"subnet_id":         subnetID,
		"total_points":      totalPoints,
		"task_points":       taskPoints,
		"adjustment_points": adjustmentTotal,
		"completed_tasks":   completedTasks,
	}

	return result, nil
}
