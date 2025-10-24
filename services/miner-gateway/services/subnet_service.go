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
	db               *sql.DB
	whitelistService *WhitelistService
}

// NewSubnetService creates a new subnet service
func NewSubnetService(db *sql.DB) *SubnetService {
	whitelistService := NewWhitelistService(db)
	return &SubnetService{
		db:               db,
		whitelistService: whitelistService,
	}
}

// FindOrCreateSubnet finds existing subnet by name or creates a new one
// Restriction: Each user can only create one subnet
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

	// Check if user is whitelisted (whitelisted users can create multiple subnets)
	isWhitelisted, err := ss.whitelistService.IsUserWhitelisted(ctx, req.CreatorWallet)
	if err != nil {
		log.Printf("Warning: Failed to check whitelist status for user %s: %v", req.CreatorWallet, err)
		isWhitelisted = false // Default to not whitelisted if check fails
	}

	// If user is not whitelisted, enforce one subnet per user limit
	if !isWhitelisted {
		userSubnet, err := ss.GetSubnetByCreator(ctx, req.CreatorWallet)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("error checking user's existing subnet: %v", err)
		}

		if userSubnet != nil {
			return nil, fmt.Errorf("user %s already has a subnet: %s. Each user can only create one subnet", req.CreatorWallet, userSubnet.Name)
		}
	} else {
		log.Printf("User %s is whitelisted, allowing multiple subnet creation", req.CreatorWallet)
	}

	// Create new subnet
	subnet := &models.Subnet{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Icon:          req.Icon,
		TVL:           req.TVL,
		Valuation:     req.Valuation,
		CreatorWallet: req.CreatorWallet,
		Status:        "active",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Set optional fields if provided
	if req.XURL != "" {
		subnet.XURL = &req.XURL
	}
	if req.Website != "" {
		subnet.Website = &req.Website
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
		INSERT INTO subnets (id, name, icon, x_url, website, tvl, valuation, creator_wallet, created_at, updated_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Convert pointers to interface{} for database insertion
	var xurl, website interface{}
	if subnet.XURL != nil {
		xurl = *subnet.XURL
	}
	if subnet.Website != nil {
		website = *subnet.Website
	}

	_, err := ss.db.ExecContext(ctx, query,
		subnet.ID,
		subnet.Name,
		subnet.Icon,
		xurl,
		website,
		subnet.TVL,
		subnet.Valuation,
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
		SELECT id, name, icon, x_url, website, tvl, valuation, creator_wallet, created_at, updated_at, status
		FROM subnets 
		WHERE name = ? AND status = 'active'
	`

	var subnet models.Subnet
	var xurl, website sql.NullString
	err := ss.db.QueryRowContext(ctx, query, name).Scan(
		&subnet.ID,
		&subnet.Name,
		&subnet.Icon,
		&xurl,
		&website,
		&subnet.TVL,
		&subnet.Valuation,
		&subnet.CreatorWallet,
		&subnet.CreatedAt,
		&subnet.UpdatedAt,
		&subnet.Status,
	)

	// Convert NullString to pointer
	if xurl.Valid {
		subnet.XURL = &xurl.String
	}
	if website.Valid {
		subnet.Website = &website.String
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subnet by name: %v", err)
	}

	return &subnet, nil
}

// GetSubnetByCreator gets subnet by creator wallet address
func (ss *SubnetService) GetSubnetByCreator(ctx context.Context, creatorWallet string) (*models.Subnet, error) {
	query := `
		SELECT id, name, icon, x_url, website, tvl, valuation, creator_wallet, created_at, updated_at, status
		FROM subnets 
		WHERE creator_wallet = ? AND status = 'active'
		LIMIT 1
	`

	var subnet models.Subnet
	var xurl, website sql.NullString
	err := ss.db.QueryRowContext(ctx, query, creatorWallet).Scan(
		&subnet.ID,
		&subnet.Name,
		&subnet.Icon,
		&xurl,
		&website,
		&subnet.TVL,
		&subnet.Valuation,
		&subnet.CreatorWallet,
		&subnet.CreatedAt,
		&subnet.UpdatedAt,
		&subnet.Status,
	)

	// Convert NullString to pointer
	if xurl.Valid {
		subnet.XURL = &xurl.String
	}
	if website.Valid {
		subnet.Website = &website.String
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get subnet by creator: %v", err)
	}

	return &subnet, nil
}

// GetSubnetByID gets subnet by ID
func (ss *SubnetService) GetSubnetByID(ctx context.Context, id string) (*models.Subnet, error) {
	query := `
		SELECT id, name, icon, x_url, website, creator_wallet, created_at, updated_at, status
		FROM subnets 
		WHERE id = ?
	`

	var subnet models.Subnet
	err := ss.db.QueryRowContext(ctx, query, id).Scan(
		&subnet.ID,
		&subnet.Name,
		&subnet.Icon,
		&subnet.XURL,
		&subnet.Website,
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
		SELECT id, name, icon, x_url, website, creator_wallet, created_at, updated_at, status
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
			&subnet.XURL,
			&subnet.Website,
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

// TransferSubnet transfers subnet ownership to another user
func (ss *SubnetService) TransferSubnet(ctx context.Context, subnetID, currentOwner, newOwner string) error {
	// Begin transaction
	tx, err := ss.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Verify current owner
	var currentCreator string
	checkQuery := `SELECT creator_wallet FROM subnets WHERE id = ? AND status = 'active'`
	err = tx.QueryRowContext(ctx, checkQuery, subnetID).Scan(&currentCreator)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("subnet not found or inactive")
		}
		return fmt.Errorf("failed to verify subnet ownership: %v", err)
	}

	if currentCreator != currentOwner {
		return fmt.Errorf("only the current owner can transfer the subnet")
	}

	// Verify new owner exists in user_profiles
	var userExists int
	userCheckQuery := `SELECT COUNT(*) FROM user_profiles WHERE wallet_address = ?`
	err = tx.QueryRowContext(ctx, userCheckQuery, newOwner).Scan(&userExists)
	if err != nil {
		return fmt.Errorf("failed to verify new owner: %v", err)
	}

	if userExists == 0 {
		return fmt.Errorf("new owner wallet address not found in user profiles")
	}

	// Update subnet creator_wallet
	updateQuery := `
		UPDATE subnets 
		SET creator_wallet = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ? AND creator_wallet = ?
	`
	result, err := tx.ExecContext(ctx, updateQuery, newOwner, subnetID, currentOwner)
	if err != nil {
		return fmt.Errorf("failed to transfer subnet: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subnet transfer failed: no rows updated")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("Subnet %s transferred from %s to %s", subnetID, currentOwner, newOwner)
	return nil
}

// UpdateSubnetFinancialData updates TVL and Valuation for a subnet
func (ss *SubnetService) UpdateSubnetFinancialData(ctx context.Context, subnetID string, req *models.SubnetFinancialUpdateRequest) error {
	// Begin transaction
	tx, err := ss.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Verify current owner and get current data
	var currentCreator string
	checkQuery := `SELECT creator_wallet FROM subnets WHERE id = ? AND status = 'active'`
	err = tx.QueryRowContext(ctx, checkQuery, subnetID).Scan(&currentCreator)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("subnet not found or inactive")
		}
		return fmt.Errorf("failed to verify subnet ownership: %v", err)
	}

	// Check if user is the creator or whitelisted
	canModify := false
	if currentCreator == req.CurrentOwner {
		canModify = true
	} else {
		// Check if user is whitelisted
		isWhitelisted, err := ss.whitelistService.IsUserWhitelisted(ctx, req.CurrentOwner)
		if err != nil {
			log.Printf("Warning: Failed to check whitelist status for user %s: %v", req.CurrentOwner, err)
		}
		canModify = isWhitelisted
	}

	if !canModify {
		return fmt.Errorf("only the subnet creator or whitelisted users can modify financial data")
	}

	// Update financial data
	updateQuery := `
		UPDATE subnets 
		SET tvl = ?, valuation = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE id = ? AND status = 'active'
	`
	result, err := tx.ExecContext(ctx, updateQuery, req.TVL, req.Valuation, subnetID)
	if err != nil {
		return fmt.Errorf("failed to update subnet financial data: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subnet financial data update failed: no rows updated")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("Subnet %s financial data updated by %s (TVL: %.2f, Valuation: %.2f)", subnetID, req.CurrentOwner, req.TVL, req.Valuation)
	return nil
}
