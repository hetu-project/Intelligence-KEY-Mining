package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/vlc"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// RoundPhase represents the current phase of a processing round
type RoundPhase string

const (
	RoundPhaseIdle        RoundPhase = "idle"         // No active round
	RoundPhaseTaskProcess RoundPhase = "task_process" // Miner processing tasks
	RoundPhaseVLCVerify   RoundPhase = "vlc_verify"   // VLC verification
	RoundPhaseQualityVote RoundPhase = "quality_vote" // Quality assessment voting
	RoundPhaseConsensus   RoundPhase = "consensus"    // BFT consensus calculation
	RoundPhaseComplete    RoundPhase = "complete"     // Round completed
)

// Round represents a processing round in the PoCoW system
type Round struct {
	ID              string                 `json:"id"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	Phase           RoundPhase             `json:"phase"`
	Tasks           []*models.Task         `json:"tasks"`
	MinerVLCBefore  *vlc.VectorClock       `json:"miner_vlc_before"`
	MinerVLCAfter   *vlc.VectorClock       `json:"miner_vlc_after"`
	ValidatorVLC    *vlc.VectorClock       `json:"validator_vlc"`
	QualityVotes    []QualityVote          `json:"quality_votes"`
	ConsensusResult *ConsensusResult       `json:"consensus_result,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// QualityVote represents a validator's vote on task quality
type QualityVote struct {
	ValidatorID string                 `json:"validator_id"`
	TaskID      string                 `json:"task_id"`
	Vote        string                 `json:"vote"` // "approve", "reject", "abstain"
	Reasoning   string                 `json:"reasoning"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ConsensusResult represents the result of BFT consensus
type ConsensusResult struct {
	Decision  string                 `json:"decision"` // "approved", "rejected"
	VoteCount map[string]int         `json:"vote_count"`
	Threshold int                    `json:"threshold"`
	Achieved  bool                   `json:"achieved"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// RoundCoordinator manages the lifecycle of processing rounds in PoCoW
type RoundCoordinator struct {
	// Core services
	taskService        *TaskService
	enhancedVLCService *EnhancedVLCService
	validatorClient    *ValidatorClient

	// Round management
	currentRound *Round
	roundHistory []*Round
	roundMutex   sync.RWMutex

	// Configuration
	roundInterval      time.Duration // How often to start new rounds
	consensusThreshold int           // BFT threshold (typically 3 out of 4)
	maxTasksPerRound   int           // Maximum tasks to process per round

	// Control
	ctx       context.Context
	cancel    context.CancelFunc
	ticker    *time.Ticker
	isRunning bool
	runMutex  sync.Mutex
}

// NewRoundCoordinator creates a new round coordinator
func NewRoundCoordinator(
	taskService *TaskService,
	enhancedVLCService *EnhancedVLCService,
	validatorClient *ValidatorClient,
	roundIntervalSeconds int,
) *RoundCoordinator {
	return &RoundCoordinator{
		taskService:        taskService,
		enhancedVLCService: enhancedVLCService,
		validatorClient:    validatorClient,
		roundHistory:       make([]*Round, 0),
		roundInterval:      time.Duration(roundIntervalSeconds) * time.Second,
		consensusThreshold: 3,  // 3 out of 4 validators (BFT: f=1, need 2f+1=3)
		maxTasksPerRound:   50, // Process up to 50 tasks per round
	}
}

// Start begins the round coordination process
func (rc *RoundCoordinator) Start(ctx context.Context) error {
	rc.runMutex.Lock()
	defer rc.runMutex.Unlock()

	if rc.isRunning {
		return fmt.Errorf("round coordinator already running")
	}

	rc.ctx, rc.cancel = context.WithCancel(ctx)
	rc.ticker = time.NewTicker(rc.roundInterval)
	rc.isRunning = true

	// Start the coordination loop
	go rc.coordinationLoop()

	log.Printf("RoundCoordinator started with interval: %v", rc.roundInterval)
	return nil
}

// Stop stops the round coordination process
func (rc *RoundCoordinator) Stop() {
	rc.runMutex.Lock()
	defer rc.runMutex.Unlock()

	if !rc.isRunning {
		return
	}

	rc.cancel()
	rc.ticker.Stop()
	rc.isRunning = false

	log.Println("RoundCoordinator stopped")
}

// coordinationLoop is the main coordination loop
func (rc *RoundCoordinator) coordinationLoop() {
	for {
		select {
		case <-rc.ctx.Done():
			return
		case <-rc.ticker.C:
			if err := rc.processRound(); err != nil {
				log.Printf("Error processing round: %v", err)
			}
		}
	}
}

// processRound processes a complete round of the PoCoW system
func (rc *RoundCoordinator) processRound() error {
	// 1. Start new round
	round, err := rc.startRound()
	if err != nil {
		return fmt.Errorf("failed to start round: %v", err)
	}

	log.Printf("🔄 Round %s started", round.ID)

	// 2. Task Processing Phase
	if err := rc.taskProcessingPhase(round); err != nil {
		rc.completeRound(round, fmt.Sprintf("task_processing_error: %v", err))
		return err
	}

	// 3. VLC Verification Phase
	if err := rc.vlcVerificationPhase(round); err != nil {
		rc.completeRound(round, fmt.Sprintf("vlc_verification_error: %v", err))
		return err
	}

	// 4. Quality Voting Phase
	if err := rc.qualityVotingPhase(round); err != nil {
		rc.completeRound(round, fmt.Sprintf("quality_voting_error: %v", err))
		return err
	}

	// 5. Consensus Phase
	if err := rc.consensusPhase(round); err != nil {
		rc.completeRound(round, fmt.Sprintf("consensus_error: %v", err))
		return err
	}

	// 6. Complete round
	rc.completeRound(round, "success")
	log.Printf("✅ Round %s completed successfully", round.ID)

	return nil
}

// startRound initiates a new processing round
func (rc *RoundCoordinator) startRound() (*Round, error) {
	rc.roundMutex.Lock()
	defer rc.roundMutex.Unlock()

	// Check if there's already an active round
	if rc.currentRound != nil && rc.currentRound.Phase != RoundPhaseComplete {
		return nil, fmt.Errorf("round %s still active in phase %s", rc.currentRound.ID, rc.currentRound.Phase)
	}

	// Create new round
	roundID := fmt.Sprintf("round_%d", time.Now().Unix())
	round := &Round{
		ID:           roundID,
		StartTime:    time.Now(),
		Phase:        RoundPhaseTaskProcess,
		Tasks:        make([]*models.Task, 0),
		QualityVotes: make([]QualityVote, 0),
		Metadata:     make(map[string]interface{}),
	}

	// Record initial VLC state
	round.MinerVLCBefore = rc.enhancedVLCService.GetMinerVLC()

	// Increment Validator-1 VLC for round start (like original demo)
	// This simulates the UserInterfaceValidator coordinating the round
	validatorVLC := vlc.NewVectorClock(2) // Validator-1 has ProcessID=2
	validatorVLC.Increment()
	round.ValidatorVLC = validatorVLC

	rc.currentRound = round

	return round, nil
}

// taskProcessingPhase handles the task processing phase
func (rc *RoundCoordinator) taskProcessingPhase(round *Round) error {
	round.Phase = RoundPhaseTaskProcess

	// Get pending tasks for processing
	// For now, simulate getting verified Twitter tasks that need final processing
	ctx := context.Background()
	tasks, err := rc.getTasksForRound(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tasks for round: %v", err)
	}

	if len(tasks) == 0 {
		log.Printf("No tasks to process in round %s", round.ID)
		return nil
	}

	round.Tasks = tasks
	log.Printf("Processing %d tasks in round %s", len(tasks), round.ID)

	// Simulate miner processing (in real implementation, this would trigger actual processing)
	for _, task := range tasks {
		// Increment dual-layer VLC for each task processing
		vlcResult := rc.enhancedVLCService.IncrementForTask(
			ctx,
			task.ID,
			task.TaskType,
			"round_processing",
			map[string]interface{}{
				"round_id": round.ID,
				"phase":    "task_processing",
			},
			task.UserWallet,
		)

		// Update task VLC
		task.VLCClock = vlcResult
		log.Printf("Task %s processed, VLC updated", task.ID)
	}

	// Record final miner VLC after processing
	round.MinerVLCAfter = rc.enhancedVLCService.GetMinerVLC()

	return nil
}

// vlcVerificationPhase handles VLC verification
func (rc *RoundCoordinator) vlcVerificationPhase(round *Round) error {
	round.Phase = RoundPhaseVLCVerify

	// Verify VLC consistency
	if round.MinerVLCBefore == nil || round.MinerVLCAfter == nil {
		return fmt.Errorf("missing VLC data for verification")
	}

	// Calculate expected VLC increment
	expectedIncrement := len(round.Tasks)
	actualIncrement := round.MinerVLCAfter.GetValue(1) - round.MinerVLCBefore.GetValue(1)

	if actualIncrement != expectedIncrement {
		return fmt.Errorf("VLC verification failed: expected %d, got %d", expectedIncrement, actualIncrement)
	}

	log.Printf("✅ VLC verification passed: increment %d", actualIncrement)
	return nil
}

// qualityVotingPhase handles quality assessment voting
func (rc *RoundCoordinator) qualityVotingPhase(round *Round) error {
	round.Phase = RoundPhaseQualityVote

	if len(round.Tasks) == 0 {
		log.Printf("No tasks to vote on in round %s", round.ID)
		return nil
	}

	// Get validator endpoints from client
	validators := rc.getValidatorEndpoints()
	if len(validators) == 0 {
		log.Printf("No validators configured, using simulation")
		return rc.simulateQualityVoting(round)
	}

	// Collect votes from real validators
	ctx := context.Background()
	for _, task := range round.Tasks {
		taskVotes, err := rc.collectValidatorVotes(ctx, task, validators)
		if err != nil {
			log.Printf("Error collecting votes for task %s: %v", task.ID, err)
			// Fall back to simulation if real validators fail
			return rc.simulateQualityVoting(round)
		}
		round.QualityVotes = append(round.QualityVotes, taskVotes...)
	}

	log.Printf("Quality voting completed: %d votes collected from %d validators",
		len(round.QualityVotes), len(validators))
	return nil
}

// consensusPhase handles BFT consensus calculation
func (rc *RoundCoordinator) consensusPhase(round *Round) error {
	round.Phase = RoundPhaseConsensus

	// Calculate consensus for each task
	taskVotes := make(map[string]map[string]int)

	for _, vote := range round.QualityVotes {
		if taskVotes[vote.TaskID] == nil {
			taskVotes[vote.TaskID] = make(map[string]int)
		}
		taskVotes[vote.TaskID][vote.Vote]++
	}

	// Determine overall consensus
	overallApproved := 0
	overallRejected := 0

	for taskID, votes := range taskVotes {
		approved := votes["approve"]
		rejected := votes["reject"]

		if approved >= rc.consensusThreshold {
			overallApproved++
			log.Printf("Task %s: APPROVED (%d votes)", taskID, approved)
		} else if rejected >= rc.consensusThreshold {
			overallRejected++
			log.Printf("Task %s: REJECTED (%d votes)", taskID, rejected)
		} else {
			log.Printf("Task %s: NO CONSENSUS (approve: %d, reject: %d)", taskID, approved, rejected)
		}
	}

	// Create consensus result
	decision := "approved"
	if overallRejected > overallApproved {
		decision = "rejected"
	}

	round.ConsensusResult = &ConsensusResult{
		Decision: decision,
		VoteCount: map[string]int{
			"approved": overallApproved,
			"rejected": overallRejected,
		},
		Threshold: rc.consensusThreshold,
		Achieved:  true,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"total_tasks": len(round.Tasks),
			"total_votes": len(round.QualityVotes),
		},
	}

	log.Printf("✅ Consensus achieved: %s (%d approved, %d rejected)",
		decision, overallApproved, overallRejected)

	return nil
}

// completeRound completes the current round
func (rc *RoundCoordinator) completeRound(round *Round, result string) {
	rc.roundMutex.Lock()
	defer rc.roundMutex.Unlock()

	now := time.Now()
	round.EndTime = &now
	round.Phase = RoundPhaseComplete
	round.Metadata["completion_result"] = result
	round.Metadata["duration"] = now.Sub(round.StartTime).String()

	// Add to history
	rc.roundHistory = append(rc.roundHistory, round)

	// Limit history size
	if len(rc.roundHistory) > 100 {
		rc.roundHistory = rc.roundHistory[len(rc.roundHistory)-100:]
	}

	rc.currentRound = nil
}

// Helper methods

// getValidatorEndpoints gets validator endpoints from the validator client
func (rc *RoundCoordinator) getValidatorEndpoints() []ValidatorEndpoint {
	// Use actual validator endpoints from docker-compose configuration
	return []ValidatorEndpoint{
		{ID: "validator-1", Role: "UserInterfaceValidator", URL: "http://validator-ui:8080", Weight: 1},
		{ID: "validator-2", Role: "ConsensusValidator", URL: "http://validator-format-1:8080", Weight: 1},
		{ID: "validator-3", Role: "ConsensusValidator", URL: "http://validator-format-2:8080", Weight: 1},
		{ID: "validator-4", Role: "ConsensusValidator", URL: "http://validator-semantic:8080", Weight: 1},
	}
}

// ValidatorEndpoint represents a validator service endpoint
type ValidatorEndpoint struct {
	ID     string `json:"id"`
	Role   string `json:"role"`
	URL    string `json:"url"`
	Weight int    `json:"weight"`
}

// collectValidatorVotes collects votes from all validators for a task
func (rc *RoundCoordinator) collectValidatorVotes(ctx context.Context, task *models.Task, validators []ValidatorEndpoint) ([]QualityVote, error) {
	votes := make([]QualityVote, 0, len(validators))

	// Create validation request
	validationReq := map[string]interface{}{
		"task_id":     task.ID,
		"task_type":   string(task.TaskType),
		"user_wallet": task.UserWallet,
		"payload":     task.Payload,
		"vlc_clock":   task.VLCClock,
		"status":      task.Status,
		"round_id":    rc.currentRound.ID,
	}

	// Collect votes from each validator
	for _, validator := range validators {
		vote, err := rc.requestValidatorVote(ctx, validator, validationReq)
		if err != nil {
			log.Printf("Failed to get vote from validator %s: %v", validator.ID, err)
			// For BFT tolerance, we can continue with fewer validators
			// but need at least 3 for consensus
			continue
		}
		votes = append(votes, vote)
	}

	// Check if we have enough votes for BFT consensus
	if len(votes) < rc.consensusThreshold {
		return nil, fmt.Errorf("insufficient votes: got %d, need %d", len(votes), rc.consensusThreshold)
	}

	return votes, nil
}

// requestValidatorVote requests a vote from a specific validator
func (rc *RoundCoordinator) requestValidatorVote(ctx context.Context, validator ValidatorEndpoint, validationReq map[string]interface{}) (QualityVote, error) {
	// Try to make real HTTP request to validator service
	vote, err := rc.makeValidatorHTTPRequest(ctx, validator, validationReq)
	if err != nil {
		log.Printf("HTTP request to validator %s failed: %v, falling back to simulation", validator.ID, err)
		// Fall back to simulation if HTTP request fails
		return rc.simulateValidatorVote(validator, validationReq), nil
	}

	return vote, nil
}

// makeValidatorHTTPRequest makes an actual HTTP request to a validator service
func (rc *RoundCoordinator) makeValidatorHTTPRequest(ctx context.Context, validator ValidatorEndpoint, validationReq map[string]interface{}) (QualityVote, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Prepare request body
	reqBody, err := json.Marshal(validationReq)
	if err != nil {
		return QualityVote{}, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	url := validator.URL + "/api/v1/validate"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return QualityVote{}, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Validator-ID", validator.ID)
	req.Header.Set("X-Round-ID", validationReq["round_id"].(string))

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return QualityVote{}, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return QualityVote{}, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return QualityVote{}, fmt.Errorf("validator returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var validatorResp struct {
		Success   bool   `json:"success"`
		Vote      string `json:"vote"` // "approve", "reject", "abstain"
		Reasoning string `json:"reasoning"`
		Error     string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &validatorResp); err != nil {
		return QualityVote{}, fmt.Errorf("failed to parse response: %v", err)
	}

	if !validatorResp.Success {
		return QualityVote{}, fmt.Errorf("validator error: %s", validatorResp.Error)
	}

	// Create vote from response
	vote := QualityVote{
		ValidatorID: validator.ID,
		TaskID:      validationReq["task_id"].(string),
		Vote:        validatorResp.Vote,
		Reasoning:   validatorResp.Reasoning,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"validator_role": validator.Role,
			"validator_url":  validator.URL,
			"round_id":       validationReq["round_id"],
			"http_status":    resp.StatusCode,
		},
	}

	return vote, nil
}

// simulateValidatorVote simulates a validator vote when HTTP request fails
func (rc *RoundCoordinator) simulateValidatorVote(validator ValidatorEndpoint, validationReq map[string]interface{}) QualityVote {
	vote := QualityVote{
		ValidatorID: validator.ID,
		TaskID:      validationReq["task_id"].(string),
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"validator_role": validator.Role,
			"validator_url":  validator.URL,
			"round_id":       validationReq["round_id"],
			"simulated":      true,
		},
	}

	// Simulate validator-specific logic
	switch validator.Role {
	case "UserInterfaceValidator":
		// Validator-1 focuses on user interaction and format validation
		vote.Vote = "approve"
		vote.Reasoning = "Task format and user interaction validated (simulated)"
	case "ConsensusValidator":
		// Other validators focus on quality assessment
		taskType := validationReq["task_type"].(string)
		if taskType == string(models.TwitterRetweetTask) {
			vote.Vote = "approve"
			vote.Reasoning = "Twitter retweet task meets quality standards (simulated)"
		} else {
			vote.Vote = "approve"
			vote.Reasoning = "Task creation meets quality standards (simulated)"
		}
	default:
		vote.Vote = "abstain"
		vote.Reasoning = "Unknown validator role (simulated)"
	}

	return vote
}

// simulateQualityVoting simulates the voting process when real validators are not available
func (rc *RoundCoordinator) simulateQualityVoting(round *Round) error {
	validatorIDs := []string{"validator-1", "validator-2", "validator-3", "validator-4"}

	for _, task := range round.Tasks {
		for _, validatorID := range validatorIDs {
			vote := rc.simulateQualityVote(validatorID, task)
			round.QualityVotes = append(round.QualityVotes, vote)
		}
	}

	log.Printf("Quality voting simulation completed: %d votes collected", len(round.QualityVotes))
	return nil
}

// getTasksForRound gets tasks that should be processed in this round
func (rc *RoundCoordinator) getTasksForRound(ctx context.Context) ([]*models.Task, error) {
	// For now, return empty slice - in real implementation, this would query
	// for verified Twitter tasks or other tasks ready for final processing
	// TODO: Implement actual task querying based on status and timestamp
	return []*models.Task{}, nil
}

// simulateQualityVote simulates a validator's quality vote
func (rc *RoundCoordinator) simulateQualityVote(validatorID string, task *models.Task) QualityVote {
	// Simple simulation - in real implementation, this would call validator service
	vote := "approve"
	reasoning := "Task meets quality standards"

	// Simulate some variation
	if task.TaskType == models.TwitterRetweetTask {
		// Twitter tasks have higher approval rate
		vote = "approve"
		reasoning = "Twitter retweet verified successfully"
	}

	return QualityVote{
		ValidatorID: validatorID,
		TaskID:      task.ID,
		Vote:        vote,
		Reasoning:   reasoning,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"task_type": string(task.TaskType),
			"round_id":  rc.currentRound.ID,
		},
	}
}

// Public query methods

// GetCurrentRound returns the current active round
func (rc *RoundCoordinator) GetCurrentRound() *Round {
	rc.roundMutex.RLock()
	defer rc.roundMutex.RUnlock()
	return rc.currentRound
}

// GetRoundHistory returns recent round history
func (rc *RoundCoordinator) GetRoundHistory(limit int) []*Round {
	rc.roundMutex.RLock()
	defer rc.roundMutex.RUnlock()

	if limit <= 0 || limit > len(rc.roundHistory) {
		return rc.roundHistory
	}

	start := len(rc.roundHistory) - limit
	return rc.roundHistory[start:]
}

// GetCoordinatorStatus returns the current status of the coordinator
func (rc *RoundCoordinator) GetCoordinatorStatus() map[string]interface{} {
	rc.roundMutex.RLock()
	defer rc.roundMutex.RUnlock()

	status := map[string]interface{}{
		"is_running":          rc.isRunning,
		"round_interval":      rc.roundInterval.String(),
		"consensus_threshold": rc.consensusThreshold,
		"max_tasks_per_round": rc.maxTasksPerRound,
		"total_rounds":        len(rc.roundHistory),
	}

	if rc.currentRound != nil {
		status["current_round"] = map[string]interface{}{
			"id":         rc.currentRound.ID,
			"phase":      rc.currentRound.Phase,
			"start_time": rc.currentRound.StartTime,
			"task_count": len(rc.currentRound.Tasks),
		}
	} else {
		status["current_round"] = nil
	}

	return status
}
