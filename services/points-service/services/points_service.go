package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/models"
)

// PointsService points service
type PointsService struct {
	db         *sql.DB
	config     *models.PointsConfig
	nftService *NFTService
}

// NewPointsService creates points service
func NewPointsService(db *sql.DB, config *models.PointsConfig, nftService *NFTService) *PointsService {
	if config == nil {
		config = models.DefaultPointsConfig()
	}

	return &PointsService{
		db:         db,
		config:     config,
		nftService: nftService,
	}
}

// CheckUserTelegramTaskToday checks if user has completed a Telegram task for the subnet today
func (ps *PointsService) CheckUserTelegramTaskToday(ctx context.Context, userWallet, subnetID, date string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM points_history 
		WHERE wallet_address = ? 
		  AND source = ? 
		  AND DATE(date) = ?
		  AND task_id IN (SELECT id FROM tasks WHERE subnet_id = ?)
	`

	var count int
	err := ps.db.QueryRowContext(ctx, query,
		userWallet, models.PointsSourceTelegramTask, date, subnetID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// HasUserCompletedTwitterPostTaskToday checks if user has completed a Twitter post task today
func (ps *PointsService) HasUserCompletedTwitterPostTaskToday(ctx context.Context, userWallet, subnetID string) (bool, error) {
	date := time.Now().Format("2006-01-02")
	query := `SELECT COUNT(*) FROM points_history WHERE wallet_address = ? AND source = ? AND DATE(created_at) = ? AND subnet_id = ?`

	var count int
	err := ps.db.QueryRowContext(ctx, query,
		userWallet, models.PointsSourceTwitterPost, date, subnetID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetUserCompletedTasks gets completed tasks for a user
func (ps *PointsService) GetUserCompletedTasks(ctx context.Context, userWallet, taskType string, limit, offset int) ([]models.UserCompletedTask, int, error) {
	// Build query with optional task type filter
	whereClause := "WHERE ph.wallet_address = ?"
	args := []interface{}{userWallet}

	if taskType != "" && taskType != "all" {
		// Map task type to source
		var source string
		switch taskType {
		case "twitter_retweet":
			source = models.PointsSourceTwitterRetweet
		case "telegram_task":
			source = models.PointsSourceTelegramTask
		case "task_creation":
			source = models.PointsSourceTaskCreation
		default:
			return nil, 0, fmt.Errorf("unsupported task type: %s", taskType)
		}
		whereClause += " AND ph.source = ?"
		args = append(args, source)
	}

	// Query for completed tasks from points_history
	query := fmt.Sprintf(`
		SELECT 
			COALESCE(ph.task_id, '') as task_id,
			ph.source,
			'completed' as status,
			ph.date,
			ph.points,
			'{}' as metadata
		FROM points_history ph
		%s
		ORDER BY ph.date DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)

	rows, err := ps.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query completed tasks: %v", err)
	}
	defer rows.Close()

	var tasks []models.UserCompletedTask
	for rows.Next() {
		var task models.UserCompletedTask
		var metadataJSON string
		var source string

		err := rows.Scan(
			&task.TaskID,
			&source,
			&task.Status,
			&task.CompletedAt,
			&task.PointsEarned,
			&metadataJSON,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %v", err)
		}

		// Map source to task type
		switch source {
		case models.PointsSourceTwitterRetweet:
			task.TaskType = "twitter_retweet"
		case models.PointsSourceTelegramTask:
			task.TaskType = "telegram_task"
		case models.PointsSourceTaskCreation:
			task.TaskType = "task_creation"
		default:
			task.TaskType = "unknown"
		}

		// Parse metadata
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err == nil {
			task.TaskDetails = metadata
			if subnetID, ok := metadata["subnet_id"].(string); ok {
				task.SubnetID = subnetID
			}
		}

		// For now, assume all tasks are valid (Twitter link modification check would require miner-gateway integration)
		task.IsValid = true

		tasks = append(tasks, task)
	}

	// Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM points_history ph
		%s
	`, whereClause)

	var total int
	err = ps.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %v", err)
	}

	return tasks, total, nil
}

// DistributePoints distributes points (core method)
func (ps *PointsService) DistributePoints(ctx context.Context, req *models.PointsDistributionRequest) (*models.PointsDistributionResult, error) {
	log.Printf("Starting points distribution for batch %s with %d tasks", req.BatchID, len(req.Tasks))

	// 1. Calculate total VLC for each type (for logging purposes)
	totalCreationVLC, totalRetweetVLC := ps.calculateTotalVLC(req.Tasks)

	if totalCreationVLC == 0 && totalRetweetVLC == 0 {
		return &models.PointsDistributionResult{
			BatchID:      req.BatchID,
			Status:       "failed",
			ErrorMessage: "No VLC to distribute",
			ProcessedAt:  time.Now(),
		}, fmt.Errorf("no VLC found for distribution")
	}

	log.Printf("📊 VLC Stats - Creation: %d, Retweet: %d (NEW: VLC directly equals points)", totalCreationVLC, totalRetweetVLC)

	// 2. Aggregate VLC by user
	userVLCMap := ps.aggregateUserVLC(req.Tasks)

	// 3. Calculate points allocation for each user (NEW: direct VLC-to-points mapping)
	userAllocations := make([]models.UserPointsResult, 0, len(userVLCMap))
	totalDistributedPoints := 0

	for userWallet, vlcData := range userVLCMap {
		// Pass dummy values for pool-based params (no longer used)
		result := ps.calculateUserPoints(userWallet, vlcData, 0, 0, 0, 0)
		userAllocations = append(userAllocations, result)
		totalDistributedPoints += result.RoundedPoints
	}

	// 5. Batch update user points to SBT system
	successCount := 0
	for i := range userAllocations {
		err := ps.updateUserPointsInSBT(ctx, &userAllocations[i], req.BatchID)
		if err != nil {
			userAllocations[i].UpdateStatus = "failed"
			userAllocations[i].UpdateError = err.Error()
			log.Printf("Failed to update points for user %s: %v", userAllocations[i].UserWallet, err)
		} else {
			userAllocations[i].UpdateStatus = "success"
			successCount++
		}
	}

	// 6. Build result
	status := "success"
	if successCount == 0 {
		status = "failed"
	} else if successCount < len(userAllocations) {
		status = "partial"
	}

	result := &models.PointsDistributionResult{
		BatchID:          req.BatchID,
		TotalPoolPoints:  totalDistributedPoints, // NEW: Total distributed points instead of pool
		CreationPoints:   totalCreationVLC,       // NEW: Direct VLC values
		RetweetPoints:    totalRetweetVLC,        // NEW: Direct VLC values
		TotalCreationVLC: totalCreationVLC,
		TotalRetweetVLC:  totalRetweetVLC,
		UserAllocations:  userAllocations,
		ProcessedAt:      time.Now(),
		Status:           status,
	}

	log.Printf("Points distribution completed for batch %s: %s (%d/%d users updated)",
		req.BatchID, status, successCount, len(userAllocations))

	return result, nil
}

// calculateTotalVLC calculates total VLC
func (ps *PointsService) calculateTotalVLC(tasks []models.TaskVLC) (creationVLC, retweetVLC int) {
	for _, task := range tasks {
		switch task.TaskType {
		case "creation":
			creationVLC += task.VLCValue
		case "retweet":
			retweetVLC += task.VLCValue
		}
	}
	return
}

// aggregateUserVLC aggregates VLC by user
func (ps *PointsService) aggregateUserVLC(tasks []models.TaskVLC) map[string]map[string]int {
	userVLCMap := make(map[string]map[string]int)

	for _, task := range tasks {
		if _, exists := userVLCMap[task.UserWallet]; !exists {
			userVLCMap[task.UserWallet] = map[string]int{"creation": 0, "retweet": 0}
		}
		userVLCMap[task.UserWallet][task.TaskType] += task.VLCValue
	}

	return userVLCMap
}

// calculateUserPoints calculates individual user points - NEW: VLC directly corresponds to points
func (ps *PointsService) calculateUserPoints(userWallet string, vlcData map[string]int, creationPoints, retweetPoints, totalCreationVLC, totalRetweetVLC int) models.UserPointsResult {
	creationVLC := vlcData["creation"]
	retweetVLC := vlcData["retweet"]

	// NEW: VLC directly corresponds to points (1 VLC = 1 point)
	// No more complex percentage-based distribution
	var creationPointsEarned, retweetPointsEarned float64

	// For creation tasks: VLC value already represents the points (usually 50 for TaskCreation)
	creationPointsEarned = float64(creationVLC)

	// For retweet tasks: VLC value directly equals points
	retweetPointsEarned = float64(retweetVLC)

	totalPoints := creationPointsEarned + retweetPointsEarned
	roundedPoints := int(math.Round(totalPoints))

	log.Printf("🎯 Direct VLC-to-Points for user %s: Creation VLC=%d → %d points, Retweet VLC=%d → %d points, Total=%d points",
		userWallet[:10]+"...", creationVLC, int(creationPointsEarned), retweetVLC, int(retweetPointsEarned), roundedPoints)

	return models.UserPointsResult{
		UserWallet:     userWallet,
		CreationVLC:    creationVLC,
		RetweetVLC:     retweetVLC,
		CreationPoints: creationPointsEarned,
		RetweetPoints:  retweetPointsEarned,
		TotalPoints:    totalPoints,
		RoundedPoints:  roundedPoints,
	}
}

// updateUserPointsInSBT updates user points to SBT system
func (ps *PointsService) updateUserPointsInSBT(ctx context.Context, userResult *models.UserPointsResult, batchID string) error {
	if userResult.RoundedPoints <= 0 {
		return nil // No points to update
	}

	// Check if user has NFT for bonus calculation
	finalPoints := userResult.RoundedPoints
	if ps.nftService != nil {
		hasNFT, err := ps.nftService.CheckUserNFTOwnership(ctx, userResult.UserWallet)
		if err != nil {
			log.Printf("Warning: Failed to check NFT ownership for user %s: %v", userResult.UserWallet, err)
			// Continue with original points if NFT check fails
		} else if hasNFT {
			finalPoints = userResult.RoundedPoints * 2 // Double points for NFT holders
			log.Printf("🎯 NFT bonus applied for user %s: %d → %d points", userResult.UserWallet, userResult.RoundedPoints, finalPoints)
		}
	}

	// Begin transaction
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. total_points will be updated automatically by database trigger
	// No manual update needed - the trigger handles this after INSERT to points_history

	// 2. Record points history (with NFT bonus applied)
	historyQuery := `
		INSERT INTO points_history (wallet_address, date, source, points, tx_ref, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	today := time.Now().Format("2006-01-02")
	source := "VLC Distribution"

	_, err = tx.ExecContext(ctx, historyQuery,
		userResult.UserWallet, today, source, finalPoints, batchID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert points history: %w", err)
	}

	// 3. Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully distributed %d points to user %s (original: %d, NFT bonus: %s)",
		finalPoints, userResult.UserWallet, userResult.RoundedPoints,
		map[bool]string{true: "applied", false: "none"}[finalPoints > userResult.RoundedPoints])

	return nil
}

// GetUserPoints gets user points information
func (ps *PointsService) GetUserPoints(ctx context.Context, walletAddress string) (int, error) {
	query := "SELECT total_points FROM user_profiles WHERE wallet_address = ?"
	var totalPoints int
	err := ps.db.QueryRowContext(ctx, query, walletAddress).Scan(&totalPoints)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("user not found: %s", walletAddress)
		}
		return 0, fmt.Errorf("failed to get user points: %w", err)
	}
	return totalPoints, nil
}

// GetPointsHistory gets user points history
func (ps *PointsService) GetPointsHistory(ctx context.Context, walletAddress string, limit int) ([]models.PointsRecord, error) {
	if limit <= 0 || limit > ps.config.HistoryLimit {
		limit = ps.config.HistoryLimit
	}

	query := `
		SELECT wallet_address, date, source, points, COALESCE(tx_ref, '') as tx_ref, created_at
		FROM points_history 
		WHERE wallet_address = ? 
		ORDER BY created_at DESC 
		LIMIT ?
	`

	rows, err := ps.db.QueryContext(ctx, query, walletAddress, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query points history: %w", err)
	}
	defer rows.Close()

	var records []models.PointsRecord
	for rows.Next() {
		var record models.PointsRecord
		err := rows.Scan(&record.WalletAddress, &record.Date, &record.Source, &record.Points, &record.TxRef, &record.CreatedAt)
		if err != nil {
			log.Printf("Error scanning points record: %v", err)
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// GetPointsStats gets points statistics
func (ps *PointsService) GetPointsStats(ctx context.Context) (*models.PointsStats, error) {
	// Get total distribution count and total points
	statsQuery := `
		SELECT 
			COUNT(DISTINCT tx_ref) as total_distributions,
			COALESCE(SUM(points), 0) as total_points,
			COUNT(DISTINCT wallet_address) as active_users,
			MAX(created_at) as last_distribution
		FROM points_history 
		WHERE tx_ref IS NOT NULL AND tx_ref != ''
	`

	var stats models.PointsStats
	var lastDist sql.NullTime

	err := ps.db.QueryRowContext(ctx, statsQuery).Scan(
		&stats.TotalDistributions,
		&stats.TotalPointsIssued,
		&stats.ActiveUsers,
		&lastDist,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get points stats: %w", err)
	}

	if lastDist.Valid {
		stats.LastDistribution = lastDist.Time
	}

	// Calculate average points
	if stats.ActiveUsers > 0 {
		stats.AvgPointsPerUser = float64(stats.TotalPointsIssued) / float64(stats.ActiveUsers)
	}

	return &stats, nil
}

// UpdateConfig updates points configuration
func (ps *PointsService) UpdateConfig(config *models.PointsConfig) {
	if config != nil {
		ps.config = config
		log.Printf("Points configuration updated: %+v", config)
	}
}

// GetConfig gets current configuration
func (ps *PointsService) GetConfig() *models.PointsConfig {
	return ps.config
}

// AddDirectPoints adds points directly to a user (for NFT bonuses, invitations, etc.)
func (ps *PointsService) AddDirectPoints(ctx context.Context, req *models.DirectPointsRequest) error {
	log.Printf("Adding %d points directly to user %s (source: %s)", req.Points, req.UserWallet, req.Source)

	// 1. Check if user exists, create if not
	if err := ps.ensureUserExists(ctx, req.UserWallet); err != nil {
		return fmt.Errorf("failed to ensure user exists: %v", err)
	}

	// 2. Add points history record
	today := time.Now().Format("2006-01-02")
	pointsRecord := &models.PointsRecord{
		WalletAddress: req.UserWallet,
		Date:          today,
		Source:        req.Source,
		Points:        req.Points,
		TxRef:         req.Reference,
		CreatedAt:     time.Now(),
	}

	if err := ps.addPointsRecord(ctx, pointsRecord); err != nil {
		return fmt.Errorf("failed to add points record: %v", err)
	}

	// 3. total_points will be updated automatically by database trigger
	// No manual update needed - the trigger handles this after INSERT to points_history

	log.Printf("Successfully added %d points to user %s", req.Points, req.UserWallet)
	return nil
}

// ensureUserExists ensures user exists in user_profiles table
func (ps *PointsService) ensureUserExists(ctx context.Context, userWallet string) error {
	// Check if user exists
	var count int
	checkQuery := `SELECT COUNT(*) FROM user_profiles WHERE wallet_address = ?`
	err := ps.db.QueryRowContext(ctx, checkQuery, userWallet).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %v", err)
	}

	if count > 0 {
		return nil // User already exists
	}

	// Create user profile if not exists
	createQuery := `
		INSERT IGNORE INTO user_profiles (
			wallet_address, display_name, registration_date, 
			total_points, today_contribution, token_uri, ipfs_hash
		) VALUES (?, ?, ?, 0, 0, '', '')
	`

	_, err = ps.db.ExecContext(ctx, createQuery,
		userWallet,
		userWallet, // Use wallet as display name initially
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create user profile: %v", err)
	}

	log.Printf("Created new user profile for wallet: %s", userWallet)
	return nil
}

// addPointsRecord adds a points record to the database
func (ps *PointsService) addPointsRecord(ctx context.Context, record *models.PointsRecord) error {
	query := `
		INSERT INTO points_history (wallet_address, date, source, points, tx_ref, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := ps.db.ExecContext(ctx, query,
		record.WalletAddress,
		record.Date,
		record.Source,
		record.Points,
		record.TxRef,
		record.CreatedAt,
	)

	return err
}
