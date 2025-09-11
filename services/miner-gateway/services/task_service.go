package services

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/crypto"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/verifiers"
)

// TaskService handles task business logic
type TaskService struct {
	db                 *sql.DB
	verifierRegistry   *verifiers.VerifierRegistry
	vlcService         *VLCService
	enhancedVLCService *EnhancedVLCService
	validatorClient    *ValidatorClient
	minerPrivateKey    *ecdsa.PrivateKey
	minerID            string
}

// NewTaskService creates a new task service
func NewTaskService(
	db *sql.DB,
	verifierRegistry *verifiers.VerifierRegistry,
	vlcService *VLCService,
	validatorClient *ValidatorClient,
	minerPrivateKey *ecdsa.PrivateKey,
	minerID string,
) *TaskService {
	// Create enhanced VLC service
	enhancedVLCService := NewEnhancedVLCService(NewDefaultVLCStrategy())

	return &TaskService{
		db:                 db,
		verifierRegistry:   verifierRegistry,
		vlcService:         vlcService,
		enhancedVLCService: enhancedVLCService,
		validatorClient:    validatorClient,
		minerPrivateKey:    minerPrivateKey,
		minerID:            minerID,
	}
}

// SubmitTask handles task submission from users
func (ts *TaskService) SubmitTask(ctx context.Context, req *models.TaskSubmitRequest) (*models.TaskSubmitResponse, error) {
	// 1. Validate payload format
	verifier, err := ts.verifierRegistry.GetVerifier(models.TaskType(req.TaskType))
	if err != nil {
		return &models.TaskSubmitResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported task type: %s", req.TaskType),
		}, nil
	}

	if err := verifier.ValidatePayload(req.Payload); err != nil {
		return &models.TaskSubmitResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid payload: %v", err),
		}, nil
	}

	// 2. Create task record
	task := &models.Task{
		ID:         uuid.New().String(),
		UserWallet: req.UserWallet,
		TaskType:   models.TaskType(req.TaskType),
		Status:     models.TaskSubmitted,
		Payload:    req.Payload,
		Attempts:   0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 3. Check if VLC increment is needed on submission
	vlcClock := ts.enhancedVLCService.IncrementForTask(ctx, task.ID, task.TaskType, "submission", req.Payload)
	task.VLCClock = vlcClock

	// 4. Save to database
	if err := ts.saveTask(ctx, task); err != nil {
		return &models.TaskSubmitResponse{
			Success: false,
			Message: "Failed to save task",
		}, err
	}

	// 5. Handle task validation asynchronously
	go ts.processTaskAsync(ctx, task)

	response := &models.TaskSubmitResponse{
		Success: true,
		TaskID:  task.ID,
		Message: "Task submitted successfully",
	}

	// 6. If it's task creation type, return VLC value
	if task.TaskType == models.TaskCreationTask {
		response.VLCValue = vlcClock.GetValue(vlcClock.ProcessID)
	}

	return response, nil
}

// processTaskAsync processes task verification asynchronously
func (ts *TaskService) processTaskAsync(ctx context.Context, task *models.Task) {
	// 1. Get validator
	verifier, err := ts.verifierRegistry.GetVerifier(task.TaskType)
	if err != nil {
		ts.updateTaskStatus(ctx, task.ID, models.TaskFailed)
		return
	}

	// 2. Try synchronous validation
	verified, proof, err := verifier.ValidateSync(ctx, task.Payload)
	if err != nil {
		ts.updateTaskStatus(ctx, task.ID, models.TaskFailed)
		return
	}

	if verified {
		// Synchronous validation successful
		ts.handleTaskVerified(ctx, task, proof)
		return
	}

	// 3. Register async listener
	task.Status = models.TaskPendingVerification
	ts.updateTaskStatus(ctx, task.ID, models.TaskPendingVerification)

	watchID, err := verifier.RegisterAsyncWatch(ctx, task.Payload)
	if err != nil {
		ts.updateTaskStatus(ctx, task.ID, models.TaskFailed)
		return
	}

	// 4. Poll task status
	ts.pollTaskStatus(ctx, task, verifier, watchID)
}

// pollTaskStatus polls the verification status
func (ts *TaskService) pollTaskStatus(ctx context.Context, task *models.Task, verifier verifiers.TaskVerifier, watchID string) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute) // 10 minutes timeout

	for {
		select {
		case <-timeout:
			ts.updateTaskStatus(ctx, task.ID, models.TaskFailed)
			return
		case <-ticker.C:
			completed, proof, err := verifier.CheckAsyncStatus(ctx, watchID)
			if err != nil {
				continue // Continue polling
			}

			if completed {
				ts.handleTaskVerified(ctx, task, proof)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// handleTaskVerified handles verified task
func (ts *TaskService) handleTaskVerified(ctx context.Context, task *models.Task, proof *models.TaskProof) {
	// 1. Check if VLC increment is needed on validation
	vlcClock := ts.enhancedVLCService.IncrementForTask(ctx, task.ID, task.TaskType, "verification", task.Payload)
	task.VLCClock = vlcClock

	// 2. Update task status and proof
	task.Status = models.TaskVerified
	task.Proof = proof
	task.UpdatedAt = time.Now()

	if err := ts.updateTask(ctx, task); err != nil {
		return
	}

	// 3. Create MinerOutput
	minerOutput, err := ts.createMinerOutput(ctx, task)
	if err != nil {
		ts.updateTaskStatus(ctx, task.ID, models.TaskFailed)
		return
	}

	// 4. Send to Validators for voting
	if err := ts.sendToValidators(ctx, minerOutput); err != nil {
		ts.updateTaskStatus(ctx, task.ID, models.TaskFailed)
		return
	}

	// 5. Update task status
	ts.updateTaskStatus(ctx, task.ID, models.TaskMinerOutputCreated)
}

// createMinerOutput creates miner output with VLC and signature
func (ts *TaskService) createMinerOutput(ctx context.Context, task *models.Task) (*models.MinerOutput, error) {
	// 1. Use task's VLC clock (already updated during verification)
	vlcClock := task.VLCClock
	if vlcClock == nil {
		// If task has no VLC clock, use current clock
		vlcClock = ts.vlcService.GetCurrentClock()
	}

	// 2. Create event ID
	eventID := fmt.Sprintf("task_%s_%d", task.ID, time.Now().Unix())

	// 3. Create MinerOutput
	minerOutput := &models.MinerOutput{
		TaskID:    task.ID,
		TaskType:  string(task.TaskType),
		MinerID:   ts.minerID,
		EventID:   eventID,
		VLCClock:  vlcClock,
		Payload:   task.Payload,
		Proof:     task.Proof,
		Timestamp: time.Now(),
	}

	// 4. Sign
	signature, err := ts.signMinerOutput(minerOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to sign miner output: %v", err)
	}
	minerOutput.Signature = signature

	// 5. Save event ID to task
	task.EventID = eventID
	task.VLCClock = vlcClock
	ts.updateTask(ctx, task)

	return minerOutput, nil
}

// signMinerOutput signs the miner output
func (ts *TaskService) signMinerOutput(output *models.MinerOutput) (string, error) {
	// Create signature data
	data := map[string]interface{}{
		"task_id":   output.TaskID,
		"miner_id":  output.MinerID,
		"event_id":  output.EventID,
		"vlc_clock": output.VLCClock,
		"timestamp": output.Timestamp.Unix(),
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return crypto.SignData(ts.minerPrivateKey, dataBytes)
}

// sendToValidators sends miner output to all validators
func (ts *TaskService) sendToValidators(ctx context.Context, minerOutput *models.MinerOutput) error {
	return ts.validatorClient.SendMinerOutput(ctx, minerOutput)
}

// Database operations

func (ts *TaskService) saveTask(ctx context.Context, task *models.Task) error {
	payloadJSON, _ := json.Marshal(task.Payload)

	query := `
		INSERT INTO tasks (id, user_wallet, task_type, status, payload, attempts, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := ts.db.ExecContext(ctx, query,
		task.ID, task.UserWallet, task.TaskType, task.Status,
		payloadJSON, task.Attempts, task.CreatedAt, task.UpdatedAt,
	)

	return err
}

func (ts *TaskService) updateTask(ctx context.Context, task *models.Task) error {
	payloadJSON, _ := json.Marshal(task.Payload)
	proofJSON, _ := json.Marshal(task.Proof)
	vlcJSON, _ := json.Marshal(task.VLCClock)

	query := `
		UPDATE tasks 
		SET status = ?, payload = ?, proof = ?, updated_at = ?, event_id = ?, vlc_clock = ?
		WHERE id = ?
	`

	_, err := ts.db.ExecContext(ctx, query,
		task.Status, payloadJSON, proofJSON, task.UpdatedAt,
		task.EventID, vlcJSON, task.ID,
	)

	return err
}

func (ts *TaskService) updateTaskStatus(ctx context.Context, taskID string, status models.TaskStatus) error {
	query := `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`
	_, err := ts.db.ExecContext(ctx, query, status, time.Now(), taskID)
	return err
}

// updateTaskStatusWithProof updates task status and proof
func (ts *TaskService) updateTaskStatusWithProof(ctx context.Context, taskID string, status models.TaskStatus, proof []byte) error {
	var query string
	var args []interface{}

	if proof != nil {
		query = `UPDATE tasks SET status = ?, proof = ?, updated_at = ?, completed_at = ? WHERE id = ?`
		completedAt := sql.NullTime{}
		if status == "VERIFIED" || status == "FAILED" {
			completedAt = sql.NullTime{Time: time.Now(), Valid: true}
		}
		args = []interface{}{status, proof, time.Now(), completedAt, taskID}
	} else {
		query = `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`
		args = []interface{}{status, time.Now(), taskID}
	}

	_, err := ts.db.ExecContext(ctx, query, args...)
	return err
}

// GetTask retrieves a task by ID
func (ts *TaskService) GetTask(ctx context.Context, taskID string) (*models.Task, error) {
	query := `
		SELECT id, user_wallet, task_type, status, payload, proof, attempts, 
		       created_at, updated_at, completed_at, event_id, vlc_clock
		FROM tasks 
		WHERE id = ?
	`

	row := ts.db.QueryRowContext(ctx, query, taskID)

	var task models.Task
	var payloadJSON, proofJSON []byte
	var completedAt, eventID sql.NullString
	var vlcClock sql.NullString

	err := row.Scan(
		&task.ID, &task.UserWallet, &task.TaskType, &task.Status,
		&payloadJSON, &proofJSON, &task.Attempts,
		&task.CreatedAt, &task.UpdatedAt, &completedAt, &eventID, &vlcClock,
	)

	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if err := json.Unmarshal(payloadJSON, &task.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %v", err)
	}

	if len(proofJSON) > 0 {
		if err := json.Unmarshal(proofJSON, &task.Proof); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proof: %v", err)
		}
	}

	if vlcClock.Valid && len(vlcClock.String) > 0 {
		if err := json.Unmarshal([]byte(vlcClock.String), &task.VLCClock); err != nil {
			return nil, fmt.Errorf("failed to unmarshal vlc_clock: %v", err)
		}
	}

	// Process optional fields
	if completedAt.Valid {
		if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
			task.CompletedAt = &t
		}
	}

	if eventID.Valid {
		task.EventID = eventID.String
	}

	return &task, nil
}

// GetUserTasksByType retrieves tasks for a user by type
func (ts *TaskService) GetUserTasksByType(ctx context.Context, userWallet string, taskType models.TaskType, limit, offset int) ([]*models.Task, error) {
	query := `
		SELECT id, user_wallet, task_type, status, payload, proof, attempts, 
		       created_at, updated_at, completed_at, event_id, vlc_clock
		FROM tasks 
		WHERE user_wallet = ? AND task_type = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := ts.db.QueryContext(ctx, query, userWallet, taskType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task

	for rows.Next() {
		var task models.Task
		var payloadJSON, proofJSON []byte
		var completedAt, eventID, vlcClock sql.NullString

		err := rows.Scan(
			&task.ID, &task.UserWallet, &task.TaskType, &task.Status,
			&payloadJSON, &proofJSON, &task.Attempts,
			&task.CreatedAt, &task.UpdatedAt, &completedAt, &eventID, &vlcClock,
		)

		if err != nil {
			return nil, err
		}

		// Parse JSON fields
		if err := json.Unmarshal(payloadJSON, &task.Payload); err != nil {
			continue // Skip failed records
		}

		if len(proofJSON) > 0 {
			if err := json.Unmarshal(proofJSON, &task.Proof); err == nil {
				// proofparsing successful
			}
		}

		if vlcClock.Valid && len(vlcClock.String) > 0 {
			if err := json.Unmarshal([]byte(vlcClock.String), &task.VLCClock); err == nil {
				// VLCparsing successful
			}
		}

		// Process optional fields
		if completedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
				task.CompletedAt = &t
			}
		}

		if eventID.Valid {
			task.EventID = eventID.String
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// GetTaskTypeStats retrieves statistics for a specific task type
func (ts *TaskService) GetTaskTypeStats(ctx context.Context, taskType models.TaskType, userWallet string) (map[string]interface{}, error) {
	baseQuery := `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'VERIFIED' THEN 1 ELSE 0 END) as verified,
			SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) as failed,
			SUM(CASE WHEN status = 'PENDING_VERIFICATION' THEN 1 ELSE 0 END) as pending,
			MIN(created_at) as first_created,
			MAX(created_at) as last_created
		FROM tasks 
		WHERE task_type = ?
	`

	args := []interface{}{taskType}

	if userWallet != "" {
		baseQuery += " AND user_wallet = ?"
		args = append(args, userWallet)
	}

	row := ts.db.QueryRowContext(ctx, baseQuery, args...)

	var total, verified, failed, pending int
	var firstCreated, lastCreated sql.NullTime

	err := row.Scan(&total, &verified, &failed, &pending, &firstCreated, &lastCreated)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"task_type": taskType,
		"total":     total,
		"verified":  verified,
		"failed":    failed,
		"pending":   pending,
	}

	if userWallet != "" {
		stats["user_wallet"] = userWallet
	}

	if firstCreated.Valid {
		stats["first_created"] = firstCreated.Time
	}

	if lastCreated.Valid {
		stats["last_created"] = lastCreated.Time
	}

	// Calculate success rate
	if total > 0 {
		stats["success_rate"] = float64(verified) / float64(total) * 100
	} else {
		stats["success_rate"] = 0.0
	}

	return stats, nil
}

// GetUserTasks retrieves tasks for a specific user with pagination
func (ts *TaskService) GetUserTasks(ctx context.Context, userWallet string, page, limit int) ([]*models.Task, int, error) {
	// Calculateoffset
	offset := (page - 1) * limit

	// Query total count
	countQuery := `SELECT COUNT(*) FROM tasks WHERE user_wallet = ?`
	var total int
	err := ts.db.QueryRowContext(ctx, countQuery, userWallet).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Query task list
	query := `
		SELECT id, user_wallet, task_type, status, payload, proof, attempts, 
		       created_at, updated_at, completed_at, event_id, vlc_clock
		FROM tasks 
		WHERE user_wallet = ? 
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?
	`

	rows, err := ts.db.QueryContext(ctx, query, userWallet, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		var payloadJSON, proofJSON []byte
		var completedAt, eventID, vlcClock sql.NullString

		err := rows.Scan(
			&task.ID, &task.UserWallet, &task.TaskType, &task.Status,
			&payloadJSON, &proofJSON, &task.Attempts,
			&task.CreatedAt, &task.UpdatedAt, &completedAt, &eventID, &vlcClock,
		)
		if err != nil {
			continue // Skip failed records
		}

		// Parse JSON fields
		if err := json.Unmarshal(payloadJSON, &task.Payload); err != nil {
			continue // Skip failed records
		}

		if len(proofJSON) > 0 {
			if err := json.Unmarshal(proofJSON, &task.Proof); err == nil {
				// proofparsing successful
			}
		}

		if vlcClock.Valid && len(vlcClock.String) > 0 {
			if err := json.Unmarshal([]byte(vlcClock.String), &task.VLCClock); err == nil {
				// VLCparsing successful
			}
		}

		// Process optional fields
		if completedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
				task.CompletedAt = &t
			}
		}

		if eventID.Valid {
			task.EventID = eventID.String
		}

		tasks = append(tasks, &task)
	}

	return tasks, total, nil
}

// GetTasksByTypeAndStatus get tasks by type and statuslist
func (ts *TaskService) GetTasksByTypeAndStatus(ctx context.Context, taskType, status string, limit int) ([]*models.Task, error) {
	query := `
		SELECT id, user_wallet, task_type, status, payload, proof, attempts, 
		       created_at, updated_at, completed_at, event_id, vlc_clock
		FROM tasks 
		WHERE task_type = ? AND status = ?
		ORDER BY created_at ASC 
		LIMIT ?
	`

	rows, err := ts.db.QueryContext(ctx, query, taskType, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		var payloadJSON, proofJSON []byte
		var completedAt, eventID, vlcClock sql.NullString

		err := rows.Scan(
			&task.ID, &task.UserWallet, &task.TaskType, &task.Status,
			&payloadJSON, &proofJSON, &task.Attempts,
			&task.CreatedAt, &task.UpdatedAt, &completedAt, &eventID, &vlcClock,
		)
		if err != nil {
			continue // Skip failed records
		}

		// Parse JSON fields
		if err := json.Unmarshal(payloadJSON, &task.Payload); err != nil {
			continue // Skip failed records
		}

		if len(proofJSON) > 0 {
			if err := json.Unmarshal(proofJSON, &task.Proof); err == nil {
				// proofparsing successful
			}
		}

		if vlcClock.Valid && len(vlcClock.String) > 0 {
			if err := json.Unmarshal([]byte(vlcClock.String), &task.VLCClock); err == nil {
				// VLCparsing successful
			}
		}

		// Process optional fields
		if completedAt.Valid {
			if t, err := time.Parse("2006-01-02 15:04:05", completedAt.String); err == nil {
				task.CompletedAt = &t
			}
		}

		if eventID.Valid {
			task.EventID = eventID.String
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}
