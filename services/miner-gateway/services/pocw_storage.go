package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// PoCWStorageService handles persistence of PoCW round data
type PoCWStorageService struct {
	db *sql.DB
}

// NewPoCWStorageService creates a new PoCW storage service
func NewPoCWStorageService(db *sql.DB) *PoCWStorageService {
	return &PoCWStorageService{
		db: db,
	}
}

// SaveRoundStart saves the initial round data when a round starts
func (ps *PoCWStorageService) SaveRoundStart(ctx context.Context, round *Round) error {
	minerVLCBeforeJSON, _ := json.Marshal(round.MinerVLCBefore)
	validatorVLCJSON, _ := json.Marshal(round.ValidatorVLC)
	metadataJSON, _ := json.Marshal(round.Metadata)

	query := `
		INSERT INTO pocw_rounds (
			round_id, start_time, phase, task_count,
			miner_vlc_before, validator_vlc, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := ps.db.ExecContext(ctx, query,
		round.ID,
		round.StartTime,
		string(round.Phase),
		len(round.Tasks),
		minerVLCBeforeJSON,
		validatorVLCJSON,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to save round start: %v", err)
	}

	log.Printf("✅ Saved round start: %s", round.ID)
	return nil
}

// UpdateRoundPhase updates the current phase of a round
func (ps *PoCWStorageService) UpdateRoundPhase(ctx context.Context, roundID string, phase RoundPhase) error {
	query := `UPDATE pocw_rounds SET phase = ?, updated_at = NOW() WHERE round_id = ?`
	_, err := ps.db.ExecContext(ctx, query, string(phase), roundID)
	if err != nil {
		return fmt.Errorf("failed to update round phase: %v", err)
	}
	return nil
}

// SaveRoundVLCAfter saves the VLC state after task processing
func (ps *PoCWStorageService) SaveRoundVLCAfter(ctx context.Context, roundID string, vlcAfter interface{}, vlcIncrement int) error {
	vlcAfterJSON, _ := json.Marshal(vlcAfter)

	query := `
		UPDATE pocw_rounds 
		SET miner_vlc_after = ?, vlc_increment = ?, updated_at = NOW()
		WHERE round_id = ?
	`

	_, err := ps.db.ExecContext(ctx, query, vlcAfterJSON, vlcIncrement, roundID)
	if err != nil {
		return fmt.Errorf("failed to save VLC after: %v", err)
	}
	return nil
}

// SaveVote saves a validator vote
func (ps *PoCWStorageService) SaveVote(ctx context.Context, roundID string, vote QualityVote) error {
	metadataJSON, _ := json.Marshal(vote.Metadata)

	query := `
		INSERT INTO pocw_votes (
			round_id, task_id, validator_id, vote,
			quality_score, weight, reasoning, vote_timestamp, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := ps.db.ExecContext(ctx, query,
		roundID,
		vote.TaskID,
		vote.ValidatorID,
		vote.Vote,
		vote.Score,
		vote.Weight,
		vote.Reasoning,
		vote.Timestamp,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to save vote: %v", err)
	}
	return nil
}

// SaveRoundTasks saves the association between round and tasks
func (ps *PoCWStorageService) SaveRoundTasks(ctx context.Context, roundID string, tasks []*models.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	// Use transaction for batch insert
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO pocw_round_tasks (
			round_id, task_id, user_wallet, task_type, subnet_id,
			vlc_increment, vlc_snapshot
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			user_wallet = VALUES(user_wallet),
			task_type = VALUES(task_type),
			subnet_id = VALUES(subnet_id)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for _, task := range tasks {
		vlcSnapshotJSON, _ := json.Marshal(task.VLCClock)

		_, err := stmt.ExecContext(ctx,
			roundID,
			task.ID,
			task.UserWallet,
			string(task.TaskType),
			task.SubnetID,
			1, // VLC increment per task
			vlcSnapshotJSON,
		)
		if err != nil {
			log.Printf("Failed to save round task %s: %v", task.ID, err)
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("✅ Saved %d round tasks for round %s", len(tasks), roundID)
	return nil
}

// UpdateRoundConsensus updates the consensus result
func (ps *PoCWStorageService) UpdateRoundConsensus(ctx context.Context, roundID string, result *ConsensusResult) error {
	metadataJSON, _ := json.Marshal(result.Metadata)

	approvedCount := result.VoteCount["approved"]
	rejectedCount := result.VoteCount["rejected"]

	query := `
		UPDATE pocw_rounds 
		SET consensus_decision = ?,
		    consensus_approved_count = ?,
		    consensus_rejected_count = ?,
		    consensus_metadata = ?,
		    updated_at = NOW()
		WHERE round_id = ?
	`

	_, err := ps.db.ExecContext(ctx, query,
		result.Decision,
		approvedCount,
		rejectedCount,
		metadataJSON,
		roundID,
	)

	if err != nil {
		return fmt.Errorf("failed to update consensus: %v", err)
	}

	log.Printf("✅ Saved consensus result for round %s: %s", roundID, result.Decision)
	return nil
}

// CompleteRound marks a round as complete and saves final statistics
func (ps *PoCWStorageService) CompleteRound(ctx context.Context, round *Round, completionResult string) error {
	var duration float64
	if round.EndTime != nil {
		duration = round.EndTime.Sub(round.StartTime).Seconds()
	}

	metadataJSON, _ := json.Marshal(round.Metadata)

	query := `
		UPDATE pocw_rounds 
		SET end_time = ?,
		    phase = ?,
		    completion_result = ?,
		    duration_seconds = ?,
		    metadata = ?,
		    updated_at = NOW()
		WHERE round_id = ?
	`

	_, err := ps.db.ExecContext(ctx, query,
		round.EndTime,
		string(RoundPhaseComplete),
		completionResult,
		duration,
		metadataJSON,
		round.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to complete round: %v", err)
	}

	log.Printf("✅ Completed round %s: %s (%.2fs)", round.ID, completionResult, duration)
	return nil
}

// UpdateTaskConsensusResult updates the consensus result for a specific task
func (ps *PoCWStorageService) UpdateTaskConsensusResult(ctx context.Context, roundID, taskID, consensusResult string, pointsAwarded int) error {
	query := `
		UPDATE pocw_round_tasks 
		SET consensus_result = ?, points_awarded = ?
		WHERE round_id = ? AND task_id = ?
	`

	_, err := ps.db.ExecContext(ctx, query, consensusResult, pointsAwarded, roundID, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task consensus result: %v", err)
	}
	return nil
}

// GetRecentRounds retrieves recent rounds with pagination
func (ps *PoCWStorageService) GetRecentRounds(ctx context.Context, limit, offset int) ([]*Round, error) {
	query := `
		SELECT round_id, start_time, end_time, phase, task_count,
		       consensus_decision, duration_seconds, completion_result
		FROM pocw_rounds
		ORDER BY start_time DESC
		LIMIT ? OFFSET ?
	`

	rows, err := ps.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query rounds: %v", err)
	}
	defer rows.Close()

	var rounds []*Round
	for rows.Next() {
		round := &Round{
			Metadata: make(map[string]interface{}),
		}
		var endTime sql.NullTime
		var consensusDecision, completionResult sql.NullString
		var durationSeconds sql.NullFloat64
		var phase string
		var taskCount int

		err := rows.Scan(
			&round.ID,
			&round.StartTime,
			&endTime,
			&phase,
			&taskCount,
			&consensusDecision,
			&durationSeconds,
			&completionResult,
		)
		if err != nil {
			log.Printf("Error scanning round: %v", err)
			continue
		}

		round.Phase = RoundPhase(phase)
		round.Metadata["task_count"] = taskCount
		if endTime.Valid {
			round.EndTime = &endTime.Time
		}
		if consensusDecision.Valid {
			round.Metadata["consensus_decision"] = consensusDecision.String
		}
		if durationSeconds.Valid {
			round.Metadata["duration_seconds"] = durationSeconds.Float64
		}
		if completionResult.Valid {
			round.Metadata["completion_result"] = completionResult.String
		}

		rounds = append(rounds, round)
	}

	return rounds, nil
}
