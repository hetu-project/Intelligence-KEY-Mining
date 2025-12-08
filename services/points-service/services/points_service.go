package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"
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
			source = models.PointsSourceVLCDistribution // twitter_retweet tasks generate VLC Distribution points
		case "twitter_post":
			source = models.PointsSourceTwitterPost
		case "twitter_follow":
			source = models.PointsSourceTwitterFollow
		case "telegram_task":
			source = models.PointsSourceTelegramTask
		case "task_creation":
			source = models.PointsSourceTaskCreation
		case "chat":
			source = models.PointsSourceChatTask
		case "register_qr_code":
			source = models.PointsSourceRegisterQRCode
		default:
			return nil, 0, fmt.Errorf("unsupported task type: %s", taskType)
		}
		whereClause += " AND ph.source = ?"
		args = append(args, source)
	}

	// Query for completed tasks from points_history with subnet info and modification check
	query := fmt.Sprintf(`
		SELECT 
			COALESCE(ph.tx_ref, '') as task_id,
			ph.source,
			'completed' as status,
			ph.date,
			ph.points,
			COALESCE(t.subnet_id, '') as subnet_id,
			COALESCE(s.name, '') as subnet_name,
			ph.created_at as completed_at,
			t.link_modified_at
		FROM points_history ph
		LEFT JOIN tasks t ON (
			CASE 
				WHEN ph.tx_ref LIKE 'pocw-consensus-%%' THEN SUBSTRING(ph.tx_ref, 16)
				ELSE ph.tx_ref
			END = t.id
		)
		LEFT JOIN subnets s ON t.subnet_id = s.id
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
		var source string
		var rawTaskID string
		var completedAt time.Time
		var linkModifiedAt sql.NullTime

		err := rows.Scan(
			&rawTaskID,
			&source,
			&task.Status,
			&task.CompletedAt,
			&task.PointsEarned,
			&task.SubnetID,
			&task.SubnetName,
			&completedAt,
			&linkModifiedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan task: %v", err)
		}

		// Extract actual task ID from tx_ref
		if strings.HasPrefix(rawTaskID, "pocw-consensus-") {
			task.TaskID = strings.TrimPrefix(rawTaskID, "pocw-consensus-")
		} else {
			task.TaskID = rawTaskID
		}

		// Map source to task type
		switch source {
		case models.PointsSourceVLCDistribution:
			task.TaskType = "twitter_retweet" // VLC Distribution comes from twitter_retweet tasks
		case models.PointsSourceTelegramTask:
			task.TaskType = "telegram_task"
		case models.PointsSourceTaskCreation:
			task.TaskType = "task_creation"
		case models.PointsSourceTwitterPost:
			task.TaskType = "twitter_post"
		case models.PointsSourceTwitterFollow:
			task.TaskType = "twitter_follow"
		case models.PointsSourceChatTask:
			task.TaskType = "chat"
		case models.PointsSourceRegisterQRCode:
			task.TaskType = "register_qr_code"
		case models.PointsSourceTelegramVoteCreate:
			task.TaskType = "telegram_vote_create"
		case models.PointsSourceTelegramVoteParticipate:
			task.TaskType = "telegram_vote_participate"
		default:
			task.TaskType = "unknown"
		}

		// Set empty task details for now
		task.TaskDetails = make(map[string]interface{})

		// Check if task is still valid by comparing completion time with link modification time
		task.IsValid = ps.isTaskCompletionStillValid(source, completedAt, linkModifiedAt)

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

// GetTaskSubnetID returns subnet_id for a given task id
func (ps *PointsService) GetTaskSubnetID(ctx context.Context, taskID string) (string, error) {
	query := `SELECT subnet_id FROM tasks WHERE id = ?`
	var subnetID sql.NullString
	err := ps.db.QueryRowContext(ctx, query, taskID).Scan(&subnetID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("task not found: %s", taskID)
		}
		return "", fmt.Errorf("failed to get task subnet: %v", err)
	}
	if !subnetID.Valid {
		return "", nil
	}
	return subnetID.String, nil
}

// CheckUserCompletedTask checks if user has already completed a specific task
func (ps *PointsService) CheckUserCompletedTask(ctx context.Context, userWallet, taskID string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM points_history 
		WHERE wallet_address = ? 
		  AND tx_ref = ?
		  AND source = ?
	`
	var count int
	err := ps.db.QueryRowContext(ctx, query, userWallet, taskID, models.PointsSourceRegisterQRCode).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check task completion: %v", err)
	}
	return count > 0, nil
}

// GetTaskRewardBadge gets the reward_badge from task payload (for Register QR code tasks)
func (ps *PointsService) GetTaskRewardBadge(ctx context.Context, taskID string) (string, error) {
	query := `
		SELECT JSON_EXTRACT(payload, '$.reward_badge') 
		FROM tasks 
		WHERE id = ? 
		  AND task_type = 'register_qr_code'
	`
	var rewardBadge sql.NullString
	err := ps.db.QueryRowContext(ctx, query, taskID).Scan(&rewardBadge)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("register_qr_code task not found: %s", taskID)
		}
		return "", fmt.Errorf("failed to get reward badge: %v", err)
	}
	if !rewardBadge.Valid || rewardBadge.String == "" {
		return "", fmt.Errorf("reward_badge not found in task payload")
	}
	// MySQL JSON_EXTRACT returns quoted strings, so we need to remove quotes
	badge := rewardBadge.String
	if len(badge) >= 2 && badge[0] == '"' && badge[len(badge)-1] == '"' {
		badge = badge[1 : len(badge)-1]
	}
	return badge, nil
}

// GetUserSubnetTotalPoints calculates user's total points in a subnet (incremental only, no override table)
func (ps *PointsService) GetUserSubnetTotalPoints(ctx context.Context, subnetID, walletAddress string) (int, error) {
	var totalPoints int
	err := ps.db.QueryRowContext(ctx, `
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
				END = t.id
			)
			WHERE ph.wallet_address = ?
				AND t.subnet_id = ?
				AND ph.source IN ('VLC Distribution', 'Twitter Post Task', 'Telegram Task', 'Twitter Follow Task', 'Telegram Vote Create', 'Telegram Vote Participate')
		) AS all_points
	`, walletAddress, subnetID, walletAddress, subnetID).Scan(&totalPoints)
	if err != nil {
		return 0, fmt.Errorf("failed to sum subnet points: %v", err)
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

// GetSubnetCreator gets the creator wallet address for a subnet
func (ps *PointsService) GetSubnetCreator(ctx context.Context, subnetID string) (string, error) {
	var creatorWallet string
	query := `SELECT creator_wallet FROM subnets WHERE id = ? AND status = 'active'`

	err := ps.db.QueryRowContext(ctx, query, subnetID).Scan(&creatorWallet)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("subnet not found or inactive")
		}
		return "", fmt.Errorf("failed to query subnet creator: %v", err)
	}

	return creatorWallet, nil
}

// GetAccumulatedCommission gets the accumulated commission for a creator
func (ps *PointsService) GetAccumulatedCommission(ctx context.Context, creatorWallet string) (float64, error) {
	query := `SELECT accumulated_commission FROM creator_commission_accumulation WHERE creator_wallet = ?`
	var accumulated float64
	err := ps.db.QueryRowContext(ctx, query, creatorWallet).Scan(&accumulated)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0.0, nil // No previous accumulation
		}
		return 0.0, fmt.Errorf("failed to query accumulated commission: %v", err)
	}
	return accumulated, nil
}

// UpdateAccumulatedCommission updates the accumulated commission for a creator
func (ps *PointsService) UpdateAccumulatedCommission(ctx context.Context, creatorWallet string, newAccumulated float64, distributedPoints int) error {
	query := `
		INSERT INTO creator_commission_accumulation (creator_wallet, accumulated_commission, total_distributed)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE 
		accumulated_commission = ?,
		total_distributed = total_distributed + ?,
		last_updated = CURRENT_TIMESTAMP
	`

	_, err := ps.db.ExecContext(ctx, query,
		creatorWallet, newAccumulated, distributedPoints,
		newAccumulated, distributedPoints,
	)

	if err != nil {
		return fmt.Errorf("failed to update accumulated commission: %v", err)
	}

	return nil
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
		SubnetID:      req.SubnetID,
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
		INSERT INTO points_history (wallet_address, date, source, points, tx_ref, subnet_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	// Convert empty subnet_id to NULL for foreign key constraint
	var subnetID interface{}
	if record.SubnetID == "" {
		subnetID = nil
	} else {
		subnetID = record.SubnetID
	}

	_, err := ps.db.ExecContext(ctx, query,
		record.WalletAddress,
		record.Date,
		record.Source,
		record.Points,
		record.TxRef,
		subnetID,
		record.CreatedAt,
	)

	return err
}

// isTaskCompletionStillValid checks if a task completion is still valid
func (ps *PointsService) isTaskCompletionStillValid(source string, completedAt time.Time, linkModifiedAt sql.NullTime) bool {
	// For non-VLC Distribution (non-Twitter retweet) tasks, assume they are valid
	if source != "VLC Distribution" {
		return true
	}

	// For Twitter retweet tasks (VLC Distribution), check if link was modified after completion
	if !linkModifiedAt.Valid {
		// No modification recorded, task is still valid
		return true
	}

	// If link was modified after the user completed the task, it's invalid
	return completedAt.After(linkModifiedAt.Time)
}

// CheckUserChatTaskToday checks if user has completed a chat task for the subnet today
func (ps *PointsService) CheckUserChatTaskToday(ctx context.Context, userWallet, subnetID, date string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM points_history 
		WHERE wallet_address = ? 
		  AND source = ? 
		  AND DATE(created_at) = ?
		  AND subnet_id = ?
	`

	var count int
	err := ps.db.QueryRowContext(ctx, query,
		userWallet, models.PointsSourceChatTask, date, subnetID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetUserChatScore gets user's total chat task points
func (ps *PointsService) GetUserChatScore(ctx context.Context, userWallet string) (map[string]interface{}, error) {
	query := `
		SELECT 
			COALESCE(SUM(points), 0) as total_points,
			COUNT(*) as tasks_count,
			COUNT(DISTINCT subnet_id) as subnets_count
		FROM points_history
		WHERE wallet_address = ? 
		AND source = ?
	`

	var totalPoints, tasksCount, subnetsCount int
	err := ps.db.QueryRowContext(ctx, query,
		userWallet,
		models.PointsSourceChatTask,
	).Scan(&totalPoints, &tasksCount, &subnetsCount)

	if err != nil {
		return nil, fmt.Errorf("failed to get chat score: %v", err)
	}

	return map[string]interface{}{
		"user_wallet":          userWallet,
		"chat_total_points":    totalPoints,
		"chat_tasks_count":     tasksCount,
		"participated_subnets": subnetsCount,
	}, nil
}

// GetUserTodayEarnings gets user's today earnings breakdown
func (ps *PointsService) GetUserTodayEarnings(ctx context.Context, userWallet string) (map[string]interface{}, error) {
	query := `
		SELECT 
			source,
			SUM(points) as points
		FROM points_history
		WHERE wallet_address = ? 
		AND DATE(created_at) = CURDATE()
		GROUP BY source
	`

	rows, err := ps.db.QueryContext(ctx, query, userWallet)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's earnings: %v", err)
	}
	defer rows.Close()

	breakdown := make(map[string]int)
	totalPoints := 0

	for rows.Next() {
		var source string
		var points int
		if err := rows.Scan(&source, &points); err != nil {
			continue
		}

		// Map source to simplified name
		simpleName := mapSourceToSimpleName(source)
		breakdown[simpleName] = points
		totalPoints += points
	}

	return map[string]interface{}{
		"user_wallet":        userWallet,
		"today_total_points": totalPoints,
		"breakdown":          breakdown,
		"date":               time.Now().Format("2006-01-02"),
	}, nil
}

// mapSourceToSimpleName maps point source to simplified name
func mapSourceToSimpleName(source string) string {
	switch source {
	case models.PointsSourceChatTask:
		return "chat"
	case models.PointsSourceTwitterPost:
		return "twitter_post"
	case models.PointsSourceTwitterFollow:
		return "twitter_follow"
	case models.PointsSourceRegisterQRCode:
		return "register_qr_code"
	case models.PointsSourceVLCDistribution:
		return "twitter_retweet"
	case models.PointsSourceTelegramTask:
		return "telegram_task"
	case models.PointsSourceTaskCreation:
		return "task_creation"
	case models.PointsSourceCreatorCommission:
		return "creator_commission"
	case models.PointsSourceInvitationReward:
		return "invitation"
	case models.PointsSourceNFTPurchase:
		return "nft_purchase"
	case models.PointsSourceTelegramVoteCreate, models.PointsSourceTelegramVoteParticipate:
		return "telegram_vote"
	default:
		return "other"
	}
}

// SpendPointsForVote handles spending points for Telegram vote actions (create/participate)
func (ps *PointsService) SpendPointsForVote(ctx context.Context, req *models.TelegramVoteSpendRequest) (*models.TelegramVoteSpendResponse, error) {
	action := strings.ToLower(req.Action)

	var cost int
	var source string
	var err error
	switch action {
	case "create":
		cost = 5
		source = models.PointsSourceTelegramVoteCreate
	case "participate":
		cost = 1
		source = models.PointsSourceTelegramVoteParticipate
	default:
		return nil, fmt.Errorf("unsupported action: %s (must be 'create' or 'participate')", req.Action)
	}

	// Ensure user profile exists
	if err := ps.ensureUserExists(ctx, req.UserWallet); err != nil {
		return nil, fmt.Errorf("failed to ensure user exists: %v", err)
	}

	// Balance check: subnet-scoped if subnet_id provided, otherwise global total_points
	var currentBalance int
	if strings.TrimSpace(req.SubnetID) != "" {
		currentBalance, err = ps.GetUserSubnetTotalPoints(ctx, req.SubnetID, req.UserWallet)
		if err != nil {
			return nil, fmt.Errorf("failed to get current subnet balance: %v", err)
		}
	} else {
		currentBalance, err = ps.GetUserPoints(ctx, req.UserWallet)
		if err != nil {
			return nil, fmt.Errorf("failed to get user total points: %v", err)
		}
	}
	if currentBalance < cost {
		return nil, fmt.Errorf("insufficient points: need %d, available %d", cost, currentBalance)
	}

	// Insert negative points record
	today := time.Now().Format("2006-01-02")
	record := &models.PointsRecord{
		WalletAddress: req.UserWallet,
		Date:          today,
		Source:        source,
		Points:        -cost,
		TxRef:         req.VoteID,
		SubnetID:      strings.TrimSpace(req.SubnetID),
		CreatedAt:     time.Now(),
	}

	if err := ps.addPointsRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to record spend: %v", err)
	}

	// Get new balance
	var newBalance int
	if strings.TrimSpace(req.SubnetID) != "" {
		newBalance, err = ps.GetUserSubnetTotalPoints(ctx, req.SubnetID, req.UserWallet)
		if err != nil {
			newBalance = currentBalance - cost
		}
	} else {
		newBalance, err = ps.GetUserPoints(ctx, req.UserWallet)
		if err != nil {
			newBalance = currentBalance - cost
		}
	}

	return &models.TelegramVoteSpendResponse{
		Success:        true,
		UserWallet:     req.UserWallet,
		SubnetID:       strings.TrimSpace(req.SubnetID),
		VoteID:         req.VoteID,
		Action:         action,
		PointsDeducted: cost,
		NewTotal:       newBalance,
		Message:        "Telegram vote spend recorded",
	}, nil
}
