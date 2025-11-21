package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// PoCWQueryService handles PoCW data queries for the browser
type PoCWQueryService struct {
	db *sql.DB
}

// NewPoCWQueryService creates a new PoCW query service
func NewPoCWQueryService(db *sql.DB) *PoCWQueryService {
	return &PoCWQueryService{
		db: db,
	}
}

// RoundListItem represents a round in the list view
type RoundListItem struct {
	RoundID           string     `json:"round_id"`
	StartTime         time.Time  `json:"start_time"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	Phase             string     `json:"phase"`
	TaskCount         int        `json:"task_count"`
	VerifiedCount     int        `json:"verified_count"`
	FailedCount       int        `json:"failed_count"`
	ConsensusDecision string     `json:"consensus_decision,omitempty"`
	ApprovedCount     int        `json:"approved_count"`
	RejectedCount     int        `json:"rejected_count"`
	DurationSeconds   float64    `json:"duration_seconds,omitempty"`
	CompletionResult  string     `json:"completion_result,omitempty"`
	VLCIncrement      int        `json:"vlc_increment"`
}

// RoundDetail represents detailed round information
type RoundDetail struct {
	RoundListItem
	MinerVLCBefore    map[string]interface{} `json:"miner_vlc_before,omitempty"`
	MinerVLCAfter     map[string]interface{} `json:"miner_vlc_after,omitempty"`
	ValidatorVLC      map[string]interface{} `json:"validator_vlc,omitempty"`
	ConsensusMetadata map[string]interface{} `json:"consensus_metadata,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Tasks             []RoundTaskItem        `json:"tasks,omitempty"`
	Votes             []VoteItem             `json:"votes,omitempty"`
}

// RoundTaskItem represents a task in a round
type RoundTaskItem struct {
	TaskID          string                 `json:"task_id"`
	UserWallet      string                 `json:"user_wallet"`
	TaskType        string                 `json:"task_type"`
	SubnetID        string                 `json:"subnet_id,omitempty"`
	VLCIncrement    int                    `json:"vlc_increment"`
	VLCSnapshot     map[string]interface{} `json:"vlc_snapshot,omitempty"`
	ConsensusResult string                 `json:"consensus_result,omitempty"`
	PointsAwarded   int                    `json:"points_awarded"`
	ProcessedAt     time.Time              `json:"processed_at"`
}

// VoteItem represents a validator vote
type VoteItem struct {
	ValidatorID   string                 `json:"validator_id"`
	ValidatorRole string                 `json:"validator_role,omitempty"`
	TaskID        string                 `json:"task_id"`
	Vote          string                 `json:"vote"`
	QualityScore  float64                `json:"quality_score"`
	Weight        float64                `json:"weight"`
	Reasoning     string                 `json:"reasoning,omitempty"`
	VoteTimestamp time.Time              `json:"vote_timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ValidatorStats represents validator performance statistics
type ValidatorStats struct {
	ValidatorID     string  `json:"validator_id"`
	ValidatorRole   string  `json:"validator_role,omitempty"`
	TotalVotes      int     `json:"total_votes"`
	ApproveCount    int     `json:"approve_count"`
	RejectCount     int     `json:"reject_count"`
	AbstainCount    int     `json:"abstain_count"`
	AvgQualityScore float64 `json:"avg_quality_score"`
	ApproveRate     float64 `json:"approve_rate"`
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalRounds       int     `json:"total_rounds"`
	TodayRounds       int     `json:"today_rounds"`
	TotalTasks        int     `json:"total_tasks"`
	TotalApproved     int     `json:"total_approved"`
	TotalRejected     int     `json:"total_rejected"`
	ConsensusRate     float64 `json:"consensus_rate"`
	AvgDuration       float64 `json:"avg_duration_seconds"`
	AvgTasksPerRound  float64 `json:"avg_tasks_per_round"`
	ActiveValidators  int     `json:"active_validators"`
	Last24HoursRounds int     `json:"last_24h_rounds"`
}

// GetRoundsList retrieves a paginated list of rounds
func (pq *PoCWQueryService) GetRoundsList(ctx context.Context, page, limit int, status string) ([]RoundListItem, int, error) {
	offset := (page - 1) * limit

	// Build WHERE clause
	whereClause := ""
	args := []interface{}{}
	if status != "" && status != "all" {
		whereClause = "WHERE phase = ?"
		args = append(args, status)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM pocw_rounds %s", whereClause)
	var total int
	err := pq.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %v", err)
	}

	// Get rounds
	query := fmt.Sprintf(`
		SELECT 
			round_id, start_time, end_time, phase, task_count,
			verified_count, failed_count, consensus_decision,
			consensus_approved_count, consensus_rejected_count,
			duration_seconds, completion_result, vlc_increment
		FROM pocw_rounds
		%s
		ORDER BY start_time DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := pq.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query rounds: %v", err)
	}
	defer rows.Close()

	var rounds []RoundListItem
	for rows.Next() {
		var r RoundListItem
		var endTime sql.NullTime
		var consensusDecision, completionResult sql.NullString
		var durationSeconds sql.NullFloat64

		err := rows.Scan(
			&r.RoundID, &r.StartTime, &endTime, &r.Phase, &r.TaskCount,
			&r.VerifiedCount, &r.FailedCount, &consensusDecision,
			&r.ApprovedCount, &r.RejectedCount,
			&durationSeconds, &completionResult, &r.VLCIncrement,
		)
		if err != nil {
			log.Printf("Error scanning round: %v", err)
			continue
		}

		if endTime.Valid {
			r.EndTime = &endTime.Time
		}
		if consensusDecision.Valid {
			r.ConsensusDecision = consensusDecision.String
		}
		if completionResult.Valid {
			r.CompletionResult = completionResult.String
		}
		if durationSeconds.Valid {
			r.DurationSeconds = durationSeconds.Float64
		}

		rounds = append(rounds, r)
	}

	return rounds, total, nil
}

// GetRoundDetail retrieves detailed information about a specific round
func (pq *PoCWQueryService) GetRoundDetail(ctx context.Context, roundID string) (*RoundDetail, error) {
	query := `
		SELECT 
			round_id, start_time, end_time, phase, task_count,
			verified_count, failed_count, consensus_decision,
			consensus_approved_count, consensus_rejected_count,
			duration_seconds, completion_result, vlc_increment,
			miner_vlc_before, miner_vlc_after, validator_vlc,
			consensus_metadata, metadata
		FROM pocw_rounds
		WHERE round_id = ?
	`

	var detail RoundDetail
	var endTime sql.NullTime
	var consensusDecision, completionResult sql.NullString
	var durationSeconds sql.NullFloat64
	var minerVLCBeforeJSON, minerVLCAfterJSON, validatorVLCJSON sql.NullString
	var consensusMetadataJSON, metadataJSON sql.NullString

	err := pq.db.QueryRowContext(ctx, query, roundID).Scan(
		&detail.RoundID, &detail.StartTime, &endTime, &detail.Phase, &detail.TaskCount,
		&detail.VerifiedCount, &detail.FailedCount, &consensusDecision,
		&detail.ApprovedCount, &detail.RejectedCount,
		&durationSeconds, &completionResult, &detail.VLCIncrement,
		&minerVLCBeforeJSON, &minerVLCAfterJSON, &validatorVLCJSON,
		&consensusMetadataJSON, &metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("round not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query round: %v", err)
	}

	// Parse optional fields
	if endTime.Valid {
		detail.EndTime = &endTime.Time
	}
	if consensusDecision.Valid {
		detail.ConsensusDecision = consensusDecision.String
	}
	if completionResult.Valid {
		detail.CompletionResult = completionResult.String
	}
	if durationSeconds.Valid {
		detail.DurationSeconds = durationSeconds.Float64
	}

	// Parse JSON fields
	if minerVLCBeforeJSON.Valid {
		json.Unmarshal([]byte(minerVLCBeforeJSON.String), &detail.MinerVLCBefore)
	}
	if minerVLCAfterJSON.Valid {
		json.Unmarshal([]byte(minerVLCAfterJSON.String), &detail.MinerVLCAfter)
	}
	if validatorVLCJSON.Valid {
		json.Unmarshal([]byte(validatorVLCJSON.String), &detail.ValidatorVLC)
	}
	if consensusMetadataJSON.Valid {
		json.Unmarshal([]byte(consensusMetadataJSON.String), &detail.ConsensusMetadata)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &detail.Metadata)
	}

	// Get tasks for this round
	tasks, err := pq.GetRoundTasks(ctx, roundID)
	if err != nil {
		log.Printf("Failed to get round tasks: %v", err)
	} else {
		detail.Tasks = tasks
	}

	// Get votes for this round
	votes, err := pq.GetRoundVotes(ctx, roundID)
	if err != nil {
		log.Printf("Failed to get round votes: %v", err)
	} else {
		detail.Votes = votes
	}

	return &detail, nil
}

// GetRoundTasks retrieves tasks for a specific round
func (pq *PoCWQueryService) GetRoundTasks(ctx context.Context, roundID string) ([]RoundTaskItem, error) {
	query := `
		SELECT 
			task_id, user_wallet, task_type, subnet_id,
			vlc_increment, vlc_snapshot, consensus_result,
			points_awarded, processed_at
		FROM pocw_round_tasks
		WHERE round_id = ?
		ORDER BY processed_at
	`

	rows, err := pq.db.QueryContext(ctx, query, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %v", err)
	}
	defer rows.Close()

	var tasks []RoundTaskItem
	for rows.Next() {
		var t RoundTaskItem
		var subnetID, consensusResult sql.NullString
		var vlcSnapshotJSON sql.NullString

		err := rows.Scan(
			&t.TaskID, &t.UserWallet, &t.TaskType, &subnetID,
			&t.VLCIncrement, &vlcSnapshotJSON, &consensusResult,
			&t.PointsAwarded, &t.ProcessedAt,
		)
		if err != nil {
			log.Printf("Error scanning task: %v", err)
			continue
		}

		if subnetID.Valid {
			t.SubnetID = subnetID.String
		}
		if consensusResult.Valid {
			t.ConsensusResult = consensusResult.String
		}
		if vlcSnapshotJSON.Valid {
			json.Unmarshal([]byte(vlcSnapshotJSON.String), &t.VLCSnapshot)
		}

		tasks = append(tasks, t)
	}

	return tasks, nil
}

// GetRoundVotes retrieves votes for a specific round
func (pq *PoCWQueryService) GetRoundVotes(ctx context.Context, roundID string) ([]VoteItem, error) {
	query := `
		SELECT 
			validator_id, validator_role, task_id, vote,
			quality_score, weight, reasoning, vote_timestamp, metadata
		FROM pocw_votes
		WHERE round_id = ?
		ORDER BY vote_timestamp
	`

	rows, err := pq.db.QueryContext(ctx, query, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to query votes: %v", err)
	}
	defer rows.Close()

	var votes []VoteItem
	for rows.Next() {
		var v VoteItem
		var validatorRole, reasoning sql.NullString
		var metadataJSON sql.NullString

		err := rows.Scan(
			&v.ValidatorID, &validatorRole, &v.TaskID, &v.Vote,
			&v.QualityScore, &v.Weight, &reasoning, &v.VoteTimestamp, &metadataJSON,
		)
		if err != nil {
			log.Printf("Error scanning vote: %v", err)
			continue
		}

		if validatorRole.Valid {
			v.ValidatorRole = validatorRole.String
		}
		if reasoning.Valid {
			v.Reasoning = reasoning.String
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &v.Metadata)
		}

		votes = append(votes, v)
	}

	return votes, nil
}

// GetValidatorStats retrieves validator performance statistics
func (pq *PoCWQueryService) GetValidatorStats(ctx context.Context) ([]ValidatorStats, error) {
	query := `
		SELECT 
			validator_id,
			validator_role,
			COUNT(*) as total_votes,
			SUM(CASE WHEN vote = 'approve' THEN 1 ELSE 0 END) as approve_count,
			SUM(CASE WHEN vote = 'reject' THEN 1 ELSE 0 END) as reject_count,
			SUM(CASE WHEN vote = 'abstain' THEN 1 ELSE 0 END) as abstain_count,
			AVG(quality_score) as avg_quality_score
		FROM pocw_votes
		GROUP BY validator_id, validator_role
		ORDER BY validator_id
	`

	rows, err := pq.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query validator stats: %v", err)
	}
	defer rows.Close()

	var stats []ValidatorStats
	for rows.Next() {
		var s ValidatorStats
		var validatorRole sql.NullString

		err := rows.Scan(
			&s.ValidatorID, &validatorRole, &s.TotalVotes,
			&s.ApproveCount, &s.RejectCount, &s.AbstainCount,
			&s.AvgQualityScore,
		)
		if err != nil {
			log.Printf("Error scanning validator stats: %v", err)
			continue
		}

		if validatorRole.Valid {
			s.ValidatorRole = validatorRole.String
		}

		// Calculate approve rate
		if s.TotalVotes > 0 {
			s.ApproveRate = float64(s.ApproveCount) / float64(s.TotalVotes) * 100
		}

		stats = append(stats, s)
	}

	return stats, nil
}

// GetDashboardStats retrieves dashboard statistics
func (pq *PoCWQueryService) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	var stats DashboardStats

	// Total rounds
	err := pq.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pocw_rounds").Scan(&stats.TotalRounds)
	if err != nil {
		return nil, fmt.Errorf("failed to get total rounds: %v", err)
	}

	// Today's rounds
	err = pq.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pocw_rounds 
		WHERE DATE(start_time) = CURDATE()
	`).Scan(&stats.TodayRounds)
	if err != nil {
		stats.TodayRounds = 0
	}

	// Last 24 hours rounds
	err = pq.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pocw_rounds 
		WHERE start_time >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
	`).Scan(&stats.Last24HoursRounds)
	if err != nil {
		stats.Last24HoursRounds = 0
	}

	// Total tasks, approved, rejected
	err = pq.db.QueryRowContext(ctx, `
		SELECT 
			SUM(task_count),
			SUM(consensus_approved_count),
			SUM(consensus_rejected_count)
		FROM pocw_rounds
		WHERE phase = 'complete'
	`).Scan(&stats.TotalTasks, &stats.TotalApproved, &stats.TotalRejected)
	if err != nil {
		log.Printf("Failed to get task stats: %v", err)
	}

	// Consensus rate
	if stats.TotalTasks > 0 {
		stats.ConsensusRate = float64(stats.TotalApproved) / float64(stats.TotalTasks) * 100
	}

	// Average duration
	err = pq.db.QueryRowContext(ctx, `
		SELECT AVG(duration_seconds)
		FROM pocw_rounds
		WHERE duration_seconds IS NOT NULL
	`).Scan(&stats.AvgDuration)
	if err != nil {
		stats.AvgDuration = 0
	}

	// Average tasks per round
	if stats.TotalRounds > 0 {
		stats.AvgTasksPerRound = float64(stats.TotalTasks) / float64(stats.TotalRounds)
	}

	// Active validators
	err = pq.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT validator_id)
		FROM pocw_votes
		WHERE vote_timestamp >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
	`).Scan(&stats.ActiveValidators)
	if err != nil {
		stats.ActiveValidators = 0
	}

	return &stats, nil
}

// GetRoundTrend retrieves round trend data for charts
func (pq *PoCWQueryService) GetRoundTrend(ctx context.Context, days int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			DATE(start_time) as date,
			COUNT(*) as round_count,
			SUM(task_count) as total_tasks,
			SUM(consensus_approved_count) as approved_tasks,
			AVG(duration_seconds) as avg_duration
		FROM pocw_rounds
		WHERE start_time >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY DATE(start_time)
		ORDER BY date DESC
	`

	rows, err := pq.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query trend: %v", err)
	}
	defer rows.Close()

	var trend []map[string]interface{}
	for rows.Next() {
		var date string
		var roundCount, totalTasks, approvedTasks int
		var avgDuration sql.NullFloat64

		err := rows.Scan(&date, &roundCount, &totalTasks, &approvedTasks, &avgDuration)
		if err != nil {
			log.Printf("Error scanning trend: %v", err)
			continue
		}

		item := map[string]interface{}{
			"date":           date,
			"round_count":    roundCount,
			"total_tasks":    totalTasks,
			"approved_tasks": approvedTasks,
		}

		if avgDuration.Valid {
			item["avg_duration"] = avgDuration.Float64
		}

		trend = append(trend, item)
	}

	return trend, nil
}
