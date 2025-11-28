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

// TaskListItem represents a task in the list view
type TaskListItem struct {
	TaskID      string     `json:"task_id"`
	UserWallet  string     `json:"user_wallet"`
	TaskType    string     `json:"task_type"`
	SubnetID    string     `json:"subnet_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	RoundID     string     `json:"round_id,omitempty"`
	Verdict     string     `json:"verdict"` // awaiting, approved, rejected
	Credit      int        `json:"credit"`  // VLC increment
	NewVLC      int64      `json:"new_vlc"` // User's VLC after this task
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// TaskDetail represents detailed task information
type TaskDetail struct {
	TaskID      string     `json:"task_id"`
	UserWallet  string     `json:"user_wallet"`
	TaskType    string     `json:"task_type"`
	SubnetID    string     `json:"subnet_id,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// PoCW Round info
	RoundID     string     `json:"round_id,omitempty"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`

	// Consensus info
	Verdict       string                 `json:"verdict"`
	VLCIncrement  int                    `json:"vlc_increment"`
	VLCBefore     int64                  `json:"vlc_before"`
	VLCAfter      int64                  `json:"vlc_after"`
	VLCSnapshot   map[string]interface{} `json:"vlc_snapshot,omitempty"`
	PointsAwarded int                    `json:"points_awarded"`

	// Task payload
	Payload map[string]interface{} `json:"payload,omitempty"`
	Proof   map[string]interface{} `json:"proof,omitempty"`

	// Voting details
	Votes        []VoteItem `json:"votes,omitempty"`
	TotalVotes   int        `json:"total_votes"`
	ApproveVotes int        `json:"approve_votes"`
	RejectVotes  int        `json:"reject_votes"`
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

// VLCNodeStats represents VLC statistics for a single node (miner or validator)
type VLCNodeStats struct {
	ProcessID       int       `json:"process_id"`
	Role            string    `json:"role"`
	CurrentValue    int64     `json:"current_value"`
	RecentIncrement int64     `json:"recent_increment"`
	LastUpdated     time.Time `json:"last_updated"`
}

// VLCStats represents overall VLC statistics
type VLCStats struct {
	Miners       []VLCNodeStats `json:"miners"`
	Validators   []VLCNodeStats `json:"validators"`
	TotalVLC     int64          `json:"total_vlc"`
	SnapshotTime time.Time      `json:"snapshot_time"`
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

// GetTasksList retrieves paginated list of tasks with PoCW consensus info
func (pq *PoCWQueryService) GetTasksList(ctx context.Context, page, limit int, filters map[string]string) ([]TaskListItem, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Build WHERE clause based on filters
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if roundID, ok := filters["round_id"]; ok && roundID != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("rt.round_id = ?"))
		args = append(args, roundID)
		argIndex++
	}

	if userWallet, ok := filters["user_wallet"]; ok && userWallet != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("rt.user_wallet = ?"))
		args = append(args, userWallet)
		argIndex++
	}

	if taskType, ok := filters["task_type"]; ok && taskType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("rt.task_type = ?"))
		args = append(args, taskType)
		argIndex++
	}

	if subnetID, ok := filters["subnet_id"]; ok && subnetID != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("rt.subnet_id = ?"))
		args = append(args, subnetID)
		argIndex++
	}

	if verdict, ok := filters["verdict"]; ok && verdict != "" {
		if verdict == "awaiting" {
			whereConditions = append(whereConditions, "rt.consensus_result IS NULL")
		} else {
			whereConditions = append(whereConditions, fmt.Sprintf("rt.consensus_result = ?"))
			args = append(args, verdict)
			argIndex++
		}
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("%s", whereConditions[0])
		for i := 1; i < len(whereConditions); i++ {
			whereClause += " AND " + whereConditions[i]
		}
	}

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM pocw_round_tasks rt
		LEFT JOIN tasks t ON rt.task_id = t.id
		%s
	`, whereClause)

	var total int
	err := pq.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// Query tasks
	query := fmt.Sprintf(`
		SELECT 
			rt.task_id,
			rt.user_wallet,
			rt.task_type,
			rt.subnet_id,
			t.created_at,
			rt.round_id,
			rt.consensus_result,
			rt.vlc_increment,
			rt.vlc_snapshot,
			rt.processed_at,
			COALESCE(r.miner_vlc_after, r.miner_vlc_before) as round_vlc
		FROM pocw_round_tasks rt
		LEFT JOIN tasks t ON rt.task_id = t.id
		LEFT JOIN pocw_rounds r ON rt.round_id = r.round_id
		%s
		ORDER BY rt.processed_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := pq.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]TaskListItem, 0)
	for rows.Next() {
		var task TaskListItem
		var consensusResult sql.NullString
		var vlcSnapshotJSON sql.NullString
		var roundVLCJSON sql.NullString
		var processedAt sql.NullTime
		var createdAt sql.NullTime
		var subnetID sql.NullString

		err := rows.Scan(
			&task.TaskID,
			&task.UserWallet,
			&task.TaskType,
			&subnetID,
			&createdAt,
			&task.RoundID,
			&consensusResult,
			&task.Credit,
			&vlcSnapshotJSON,
			&processedAt,
			&roundVLCJSON,
		)
		if err != nil {
			log.Printf("Error scanning task row: %v", err)
			continue
		}

		if subnetID.Valid {
			task.SubnetID = subnetID.String
		}

		if createdAt.Valid {
			task.CreatedAt = createdAt.Time
		}

		if processedAt.Valid {
			task.ProcessedAt = &processedAt.Time
		}

		// Determine verdict
		if consensusResult.Valid {
			task.Verdict = consensusResult.String
		} else {
			task.Verdict = "awaiting"
		}

		// Extract Miner's VLC from Round VLC (prefer this over task vlc_snapshot)
		if roundVLCJSON.Valid && roundVLCJSON.String != "" {
			var roundVLC map[string]interface{}
			if err := json.Unmarshal([]byte(roundVLCJSON.String), &roundVLC); err == nil {
				// VLC structure: {"process_id": 1, "values": {"1": 3}}
				if values, ok := roundVLC["values"].(map[string]interface{}); ok {
					if processID, ok := roundVLC["process_id"].(float64); ok {
						processIDStr := fmt.Sprintf("%.0f", processID)
						if vlc, ok := values[processIDStr].(float64); ok {
							task.NewVLC = int64(vlc)
						}
					}
				}
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, total, nil
}

// GetTaskDetail retrieves detailed information about a specific task
func (pq *PoCWQueryService) GetTaskDetail(ctx context.Context, taskID string) (*TaskDetail, error) {
	query := `
		SELECT 
			t.id,
			t.user_wallet,
			t.task_type,
			t.subnet_id,
			t.status,
			t.created_at,
			t.completed_at,
			t.payload,
			t.proof,
			rt.round_id,
			rt.processed_at,
			rt.consensus_result,
			rt.vlc_increment,
			rt.vlc_snapshot,
			rt.points_awarded
		FROM tasks t
		LEFT JOIN pocw_round_tasks rt ON t.id = rt.task_id
		WHERE t.id = ?
	`

	var detail TaskDetail
	var subnetID, status sql.NullString
	var completedAt, processedAt sql.NullTime
	var roundID, consensusResult sql.NullString
	var payloadJSON, proofJSON, vlcSnapshotJSON sql.NullString
	var vlcIncrement, pointsAwarded sql.NullInt64

	err := pq.db.QueryRowContext(ctx, query, taskID).Scan(
		&detail.TaskID,
		&detail.UserWallet,
		&detail.TaskType,
		&subnetID,
		&status,
		&detail.CreatedAt,
		&completedAt,
		&payloadJSON,
		&proofJSON,
		&roundID,
		&processedAt,
		&consensusResult,
		&vlcIncrement,
		&vlcSnapshotJSON,
		&pointsAwarded,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task: %w", err)
	}

	// Set optional fields
	if subnetID.Valid {
		detail.SubnetID = subnetID.String
	}
	if status.Valid {
		detail.Status = status.String
	}
	if completedAt.Valid {
		detail.CompletedAt = &completedAt.Time
	}
	if roundID.Valid {
		detail.RoundID = roundID.String
	}
	if processedAt.Valid {
		detail.ProcessedAt = &processedAt.Time
	}
	if vlcIncrement.Valid {
		detail.VLCIncrement = int(vlcIncrement.Int64)
	}
	if pointsAwarded.Valid {
		detail.PointsAwarded = int(pointsAwarded.Int64)
	}

	// Determine verdict
	if consensusResult.Valid {
		detail.Verdict = consensusResult.String
	} else if roundID.Valid {
		detail.Verdict = "awaiting"
	} else {
		detail.Verdict = "not_in_consensus"
	}

	// Parse JSON fields
	if payloadJSON.Valid && payloadJSON.String != "" {
		if err := json.Unmarshal([]byte(payloadJSON.String), &detail.Payload); err != nil {
			log.Printf("Error parsing payload JSON: %v", err)
		}
	}

	if proofJSON.Valid && proofJSON.String != "" {
		if err := json.Unmarshal([]byte(proofJSON.String), &detail.Proof); err != nil {
			log.Printf("Error parsing proof JSON: %v", err)
		}
	}

	if vlcSnapshotJSON.Valid && vlcSnapshotJSON.String != "" {
		if err := json.Unmarshal([]byte(vlcSnapshotJSON.String), &detail.VLCSnapshot); err != nil {
			log.Printf("Error parsing VLC snapshot JSON: %v", err)
		} else {
			// Calculate VLC before and after
			// VLC snapshot structure: {"process_id": 1, "values": {"1": 1001, "2": 500}}
			if values, ok := detail.VLCSnapshot["values"].(map[string]interface{}); ok {
				if processID, ok := detail.VLCSnapshot["process_id"].(float64); ok {
					processIDStr := fmt.Sprintf("%.0f", processID)
					if vlc, ok := values[processIDStr].(float64); ok {
						detail.VLCAfter = int64(vlc)
						detail.VLCBefore = detail.VLCAfter - int64(detail.VLCIncrement)
					}
				}
			}
		}
	}

	// Get votes for this task
	votesQuery := `
		SELECT 
			validator_id,
			validator_role,
			vote,
			quality_score,
			weight,
			reasoning,
			vote_timestamp
		FROM pocw_votes
		WHERE task_id = ?
		ORDER BY vote_timestamp ASC
	`

	rows, err := pq.db.QueryContext(ctx, votesQuery, taskID)
	if err != nil {
		log.Printf("Error querying votes: %v", err)
	} else {
		defer rows.Close()

		votes := make([]VoteItem, 0)
		approveCount := 0
		rejectCount := 0

		for rows.Next() {
			var vote VoteItem
			var validatorRole, reasoning sql.NullString
			var qualityScore, weight sql.NullFloat64

			err := rows.Scan(
				&vote.ValidatorID,
				&validatorRole,
				&vote.Vote,
				&qualityScore,
				&weight,
				&reasoning,
				&vote.VoteTimestamp,
			)
			if err != nil {
				log.Printf("Error scanning vote row: %v", err)
				continue
			}

			vote.TaskID = taskID
			if validatorRole.Valid {
				vote.ValidatorRole = validatorRole.String
			}
			if qualityScore.Valid {
				vote.QualityScore = qualityScore.Float64
			}
			if weight.Valid {
				vote.Weight = weight.Float64
			}
			if reasoning.Valid {
				vote.Reasoning = reasoning.String
			}

			if vote.Vote == "approve" {
				approveCount++
			} else if vote.Vote == "reject" {
				rejectCount++
			}

			votes = append(votes, vote)
		}

		detail.Votes = votes
		detail.TotalVotes = len(votes)
		detail.ApproveVotes = approveCount
		detail.RejectVotes = rejectCount
	}

	return &detail, nil
}

// GetVLCStats retrieves current VLC statistics for all nodes
func (pq *PoCWQueryService) GetVLCStats(ctx context.Context) (*VLCStats, error) {
	// Get the latest round (prefer miner_vlc_after, fallback to miner_vlc_before)
	query := `
		SELECT 
			COALESCE(miner_vlc_after, miner_vlc_before) as miner_vlc,
			validator_vlc,
			end_time
		FROM pocw_rounds
		ORDER BY start_time DESC
		LIMIT 1
	`

	var minerVLCJSON, validatorVLCJSON sql.NullString
	var snapshotTime sql.NullTime

	err := pq.db.QueryRowContext(ctx, query).Scan(&minerVLCJSON, &validatorVLCJSON, &snapshotTime)
	if err == sql.ErrNoRows {
		// No rounds yet, return initial state
		return &VLCStats{
			Miners: []VLCNodeStats{
				{ProcessID: 1, Role: "miner", CurrentValue: 0, RecentIncrement: 0, LastUpdated: time.Now()},
			},
			Validators: []VLCNodeStats{
				{ProcessID: 2, Role: "ui_validator", CurrentValue: 0, RecentIncrement: 0, LastUpdated: time.Now()},
				{ProcessID: 3, Role: "format_validator", CurrentValue: 0, RecentIncrement: 0, LastUpdated: time.Now()},
				{ProcessID: 4, Role: "semantic_validator", CurrentValue: 0, RecentIncrement: 0, LastUpdated: time.Now()},
			},
			TotalVLC:     0,
			SnapshotTime: time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query VLC data: %w", err)
	}

	stats := &VLCStats{
		Miners:     []VLCNodeStats{},
		Validators: []VLCNodeStats{},
		TotalVLC:   0,
	}

	if snapshotTime.Valid {
		stats.SnapshotTime = snapshotTime.Time
	} else {
		stats.SnapshotTime = time.Now()
	}

	// If VLC data is NULL, return initial state
	if !minerVLCJSON.Valid && !validatorVLCJSON.Valid {
		stats.Miners = []VLCNodeStats{
			{ProcessID: 1, Role: "miner", CurrentValue: 0, RecentIncrement: 0, LastUpdated: stats.SnapshotTime},
		}
		stats.Validators = []VLCNodeStats{
			{ProcessID: 2, Role: "ui_validator", CurrentValue: 0, RecentIncrement: 0, LastUpdated: stats.SnapshotTime},
			{ProcessID: 3, Role: "format_validator", CurrentValue: 0, RecentIncrement: 0, LastUpdated: stats.SnapshotTime},
			{ProcessID: 4, Role: "semantic_validator", CurrentValue: 0, RecentIncrement: 0, LastUpdated: stats.SnapshotTime},
		}
		return stats, nil
	}

	// Parse miner VLC
	if minerVLCJSON.Valid && minerVLCJSON.String != "" {
		var minerVLC map[string]interface{}
		if err := json.Unmarshal([]byte(minerVLCJSON.String), &minerVLC); err == nil {
			if values, ok := minerVLC["values"].(map[string]interface{}); ok {
				for processIDStr, vlcValue := range values {
					if vlc, ok := vlcValue.(float64); ok {
						processID := 0
						fmt.Sscanf(processIDStr, "%d", &processID)

						stats.Miners = append(stats.Miners, VLCNodeStats{
							ProcessID:       processID,
							Role:            "miner",
							CurrentValue:    int64(vlc),
							RecentIncrement: 0, // TODO: Calculate from previous round
							LastUpdated:     stats.SnapshotTime,
						})
						stats.TotalVLC += int64(vlc)
					}
				}
			}
		}
	}

	// Parse validator VLC
	if validatorVLCJSON.Valid && validatorVLCJSON.String != "" {
		var validatorVLC map[string]interface{}
		if err := json.Unmarshal([]byte(validatorVLCJSON.String), &validatorVLC); err == nil {
			if values, ok := validatorVLC["values"].(map[string]interface{}); ok {
				for processIDStr, vlcValue := range values {
					if vlc, ok := vlcValue.(float64); ok {
						processID := 0
						fmt.Sscanf(processIDStr, "%d", &processID)

						// Determine validator role based on process ID
						role := "validator"
						if processID == 2 {
							role = "ui_validator"
						} else if processID == 3 {
							role = "format_validator"
						} else if processID == 4 {
							role = "semantic_validator"
						}

						stats.Validators = append(stats.Validators, VLCNodeStats{
							ProcessID:       processID,
							Role:            role,
							CurrentValue:    int64(vlc),
							RecentIncrement: 0, // TODO: Calculate from previous round
							LastUpdated:     stats.SnapshotTime,
						})
						stats.TotalVLC += int64(vlc)
					}
				}
			}
		}
	}

	return stats, nil
}
