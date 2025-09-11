package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/points-service/models"
)

// PointsService points service
type PointsService struct {
	db     *sql.DB
	config *models.PointsConfig
}

// NewPointsService creates points service
func NewPointsService(db *sql.DB, config *models.PointsConfig) *PointsService {
	if config == nil {
		config = models.DefaultPointsConfig()
	}

	return &PointsService{
		db:     db,
		config: config,
	}
}

// DistributePoints distributes points (core method)
func (ps *PointsService) DistributePoints(ctx context.Context, req *models.PointsDistributionRequest) (*models.PointsDistributionResult, error) {
	log.Printf("Starting points distribution for batch %s with %d tasks", req.BatchID, len(req.Tasks))

	// 1. Calculate total VLC for each type
	totalCreationVLC, totalRetweetVLC := ps.calculateTotalVLC(req.Tasks)

	if totalCreationVLC == 0 && totalRetweetVLC == 0 {
		return &models.PointsDistributionResult{
			BatchID:      req.BatchID,
			Status:       "failed",
			ErrorMessage: "No VLC to distribute",
			ProcessedAt:  time.Now(),
		}, fmt.Errorf("no VLC found for distribution")
	}

	// 2. Calculate points pool allocation
	creationPoints := int(float64(ps.config.TotalPoolPoints) * ps.config.CreationRatio)
	retweetPoints := int(float64(ps.config.TotalPoolPoints) * ps.config.RetweetRatio)

	log.Printf("VLC Stats - Creation: %d, Retweet: %d", totalCreationVLC, totalRetweetVLC)
	log.Printf("Points Pool - Creation: %d, Retweet: %d", creationPoints, retweetPoints)

	// 3. Aggregate VLC by user
	userVLCMap := ps.aggregateUserVLC(req.Tasks)

	// 4. Calculate points allocation for each user
	userAllocations := make([]models.UserPointsResult, 0, len(userVLCMap))
	totalDistributedPoints := 0

	for userWallet, vlcData := range userVLCMap {
		result := ps.calculateUserPoints(userWallet, vlcData, creationPoints, retweetPoints, totalCreationVLC, totalRetweetVLC)
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
		TotalPoolPoints:  ps.config.TotalPoolPoints,
		CreationPoints:   creationPoints,
		RetweetPoints:    retweetPoints,
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

// calculateUserPoints calculates individual user points
func (ps *PointsService) calculateUserPoints(userWallet string, vlcData map[string]int, creationPoints, retweetPoints, totalCreationVLC, totalRetweetVLC int) models.UserPointsResult {
	creationVLC := vlcData["creation"]
	retweetVLC := vlcData["retweet"]

	// Calculate points allocation
	var creationPointsEarned, retweetPointsEarned float64

	if totalCreationVLC > 0 && creationVLC > 0 {
		creationPointsEarned = float64(creationPoints) * float64(creationVLC) / float64(totalCreationVLC)
	}

	if totalRetweetVLC > 0 && retweetVLC > 0 {
		retweetPointsEarned = float64(retweetPoints) * float64(retweetVLC) / float64(totalRetweetVLC)
	}

	totalPoints := creationPointsEarned + retweetPointsEarned
	roundedPoints := int(math.Round(totalPoints))

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

	// Begin transaction
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Update user total points
	updateQuery := `
		UPDATE user_profiles 
		SET total_points = total_points + ?, 
		    updated_at = ?
		WHERE wallet_address = ?
	`
	_, err = tx.ExecContext(ctx, updateQuery, userResult.RoundedPoints, time.Now(), userResult.UserWallet)
	if err != nil {
		return fmt.Errorf("failed to update user total points: %w", err)
	}

	// 2. Record points history
	historyQuery := `
		INSERT INTO points_history (wallet_address, date, source, points, tx_ref, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	today := time.Now().Format("2006-01-02")
	source := "VLC Distribution"

	_, err = tx.ExecContext(ctx, historyQuery,
		userResult.UserWallet, today, source, userResult.RoundedPoints, batchID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert points history: %w", err)
	}

	// 3. Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

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
