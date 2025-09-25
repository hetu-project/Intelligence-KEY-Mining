package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"google.golang.org/grpc"
)

// Client represents a Dgraph client
type Client struct {
	dg *dgo.Dgraph
}

// NewClient creates a new Dgraph client
func NewClient(dgraphURL string) (*Client, error) {
	// Connect to Dgraph
	conn, err := grpc.Dial(dgraphURL, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Dgraph: %v", err)
	}

	// Create Dgraph client
	dg := dgo.NewDgraphClient(api.NewDgraphClient(conn))

	return &Client{
		dg: dg,
	}, nil
}

// TaskNode represents a task node in the graph
type TaskNode struct {
	UID         string                 `json:"uid,omitempty"`
	Type        string                 `json:"dgraph.type"`
	TaskID      string                 `json:"task_id"`
	UserWallet  string                 `json:"user_wallet"`
	TwitterID   string                 `json:"twitter_id,omitempty"`
	TaskType    string                 `json:"task_type"`
	Status      string                 `json:"status"`
	Payload     map[string]interface{} `json:"payload"`
	Proof       map[string]interface{} `json:"proof,omitempty"`
	VLCClock    map[string]interface{} `json:"vlc_clock,omitempty"`
	EventID     string                 `json:"event_id,omitempty"`
	Attempts    int                    `json:"attempts"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	CompletedAt string                 `json:"completed_at,omitempty"`
}

// ValidationNode represents a validation node in the graph
type ValidationNode struct {
	UID         string                 `json:"uid,omitempty"`
	Type        string                 `json:"dgraph.type"`
	EventID     string                 `json:"event_id"`
	ValidatorID string                 `json:"validator_id"`
	Vote        string                 `json:"vote"`
	Score       float64                `json:"score"`
	Weight      float64                `json:"weight"`
	VLCState    map[string]interface{} `json:"vlc_state"`
	Timestamp   string                 `json:"timestamp"`
	TaskRef     TaskNode               `json:"task_ref"`
}

// ConsensusNode represents a consensus result node
type ConsensusNode struct {
	UID              string           `json:"uid,omitempty"`
	Type             string           `json:"dgraph.type"`
	EventID          string           `json:"event_id"`
	FinalDecision    string           `json:"final_decision"`
	TotalWeight      float64          `json:"total_weight"`
	AcceptWeight     float64          `json:"accept_weight"`
	RejectWeight     float64          `json:"reject_weight"`
	ConsensusReached bool             `json:"consensus_reached"`
	Timestamp        string           `json:"timestamp"`
	TaskRef          TaskNode         `json:"task_ref"`
	Validations      []ValidationNode `json:"validations"`
}

// StoreTask stores a task in the graph database
func (c *Client) StoreTask(ctx context.Context, task *TaskNode) error {
	task.Type = "Task"

	mu := &api.Mutation{
		CommitNow: true,
	}

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %v", err)
	}

	mu.SetJson = taskJSON

	txn := c.dg.NewTxn()
	defer txn.Discard(ctx)

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return fmt.Errorf("failed to store task: %v", err)
	}

	log.Printf("Task stored in graph: %s", task.TaskID)
	return nil
}

// StoreValidation stores a validation result in the graph database
func (c *Client) StoreValidation(ctx context.Context, validation *ValidationNode) error {
	validation.Type = "Validation"

	mu := &api.Mutation{
		CommitNow: true,
	}

	validationJSON, err := json.Marshal(validation)
	if err != nil {
		return fmt.Errorf("failed to marshal validation: %v", err)
	}

	mu.SetJson = validationJSON

	txn := c.dg.NewTxn()
	defer txn.Discard(ctx)

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return fmt.Errorf("failed to store validation: %v", err)
	}

	log.Printf("Validation stored in graph: %s by %s", validation.EventID, validation.ValidatorID)
	return nil
}

// StoreConsensus stores a consensus result in the graph database
func (c *Client) StoreConsensus(ctx context.Context, consensus *ConsensusNode) error {
	consensus.Type = "Consensus"

	mu := &api.Mutation{
		CommitNow: true,
	}

	consensusJSON, err := json.Marshal(consensus)
	if err != nil {
		return fmt.Errorf("failed to marshal consensus: %v", err)
	}

	mu.SetJson = consensusJSON

	txn := c.dg.NewTxn()
	defer txn.Discard(ctx)

	_, err = txn.Mutate(ctx, mu)
	if err != nil {
		return fmt.Errorf("failed to store consensus: %v", err)
	}

	log.Printf("Consensus stored in graph: %s", consensus.EventID)
	return nil
}

// GetTask retrieves a task by task ID
func (c *Client) GetTask(ctx context.Context, taskID string) (*TaskNode, error) {
	query := `
		query getTask($taskID: string) {
			task(func: eq(task_id, $taskID)) {
				uid
				task_id
				user_wallet
				task_type
				status
				payload
				created_at
				updated_at
			}
		}
	`

	variables := map[string]string{
		"$taskID": taskID,
	}

	resp, err := c.dg.NewTxn().QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to query task: %v", err)
	}

	var result struct {
		Task []TaskNode `json:"task"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(result.Task) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return &result.Task[0], nil
}

// GetValidationsByEventID retrieves all validations for an event
func (c *Client) GetValidationsByEventID(ctx context.Context, eventID string) ([]ValidationNode, error) {
	query := `
		query getValidations($eventID: string) {
			validations(func: eq(event_id, $eventID)) {
				uid
				event_id
				validator_id
				vote
				score
				weight
				vlc_state
				timestamp
				task_ref {
					task_id
					user_wallet
					task_type
				}
			}
		}
	`

	variables := map[string]string{
		"$eventID": eventID,
	}

	resp, err := c.dg.NewTxn().QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to query validations: %v", err)
	}

	var result struct {
		Validations []ValidationNode `json:"validations"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result.Validations, nil
}

// GetConsensus retrieves consensus result by event ID
func (c *Client) GetConsensus(ctx context.Context, eventID string) (*ConsensusNode, error) {
	query := `
		query getConsensus($eventID: string) {
			consensus(func: eq(event_id, $eventID)) {
				uid
				event_id
				final_decision
				total_weight
				accept_weight
				reject_weight
				consensus_reached
				timestamp
				task_ref {
					task_id
					user_wallet
					task_type
				}
				validations {
					validator_id
					vote
					score
					weight
				}
			}
		}
	`

	variables := map[string]string{
		"$eventID": eventID,
	}

	resp, err := c.dg.NewTxn().QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to query consensus: %v", err)
	}

	var result struct {
		Consensus []ConsensusNode `json:"consensus"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(result.Consensus) == 0 {
		return nil, fmt.Errorf("consensus not found: %s", eventID)
	}

	return &result.Consensus[0], nil
}

// GetUserTasks retrieves all tasks for a user
func (c *Client) GetUserTasks(ctx context.Context, userWallet string, limit int) ([]TaskNode, error) {
	query := `
		query getUserTasks($userWallet: string, $limit: int) {
			tasks(func: eq(user_wallet, $userWallet), first: $limit, orderasc: created_at) {
				uid
				task_id
				user_wallet
				task_type
				status
				created_at
				updated_at
			}
		}
	`

	variables := map[string]string{
		"$userWallet": userWallet,
		"$limit":      fmt.Sprintf("%d", limit),
	}

	resp, err := c.dg.NewTxn().QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to query user tasks: %v", err)
	}

	var result struct {
		Tasks []TaskNode `json:"tasks"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result.Tasks, nil
}

// SetupSchema sets up the Dgraph schema
func (c *Client) SetupSchema(ctx context.Context) error {
	schema := `
		type Task {
			task_id: string @index(exact) .
			user_wallet: string @index(exact) .
			twitter_id: string @index(exact) .
			task_type: string @index(exact) .
			status: string @index(exact) .
			payload: string .
			proof: string .
			vlc_clock: string .
			event_id: string @index(exact) .
			attempts: int .
			created_at: datetime @index(hour) .
			updated_at: datetime .
			completed_at: datetime .
		}

		type Validation {
			event_id: string @index(exact) .
			validator_id: string @index(exact) .
			vote: string @index(exact) .
			score: float .
			weight: float .
			vlc_state: string .
			timestamp: datetime @index(hour) .
			task_ref: [uid] .
		}

		type Consensus {
			event_id: string @index(exact) .
			final_decision: string @index(exact) .
			total_weight: float .
			accept_weight: float .
			reject_weight: float .
			consensus_reached: bool .
			timestamp: datetime @index(hour) .
			task_ref: [uid] .
			validations: [uid] .
		}

		task_id: string @index(exact) .
		user_wallet: string @index(exact) .
		twitter_id: string @index(exact) .
		task_type: string @index(exact) .
		status: string @index(exact) .
		event_id: string @index(exact) .
		validator_id: string @index(exact) .
		vote: string @index(exact) .
		final_decision: string @index(exact) .
		attempts: int .
		created_at: datetime @index(hour) .
		updated_at: datetime .
		completed_at: datetime .
		timestamp: datetime @index(hour) .
	`

	op := &api.Operation{
		Schema: schema,
	}

	return c.dg.Alter(ctx, op)
}

// GetTasksByTwitterID retrieves all tasks for a Twitter ID
func (c *Client) GetTasksByTwitterID(ctx context.Context, twitterID string, limit int) ([]TaskNode, error) {
	query := `
		query getTasksByTwitterID($twitterID: string, $limit: int) {
			tasks(func: eq(twitter_id, $twitterID), first: $limit, orderasc: created_at) {
				uid
				task_id
				user_wallet
				twitter_id
				task_type
				status
				created_at
				updated_at
			}
		}
	`

	variables := map[string]string{
		"$twitterID": twitterID,
		"$limit":     fmt.Sprintf("%d", limit),
	}

	resp, err := c.dg.NewTxn().QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by twitter ID: %v", err)
	}

	var result struct {
		Tasks []TaskNode `json:"tasks"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result.Tasks, nil
}

// GetTasksByTypeAndStatus retrieves tasks by type and status
func (c *Client) GetTasksByTypeAndStatus(ctx context.Context, taskType, status string, limit int) ([]TaskNode, error) {
	query := `
		query getTasksByTypeAndStatus($taskType: string, $status: string, $limit: int) {
			tasks(func: eq(task_type, $taskType)) @filter(eq(status, $status)) {
				uid
				task_id
				user_wallet
				twitter_id
				task_type
				status
				payload
				created_at
				updated_at
			}
		}
	`

	variables := map[string]string{
		"$taskType": taskType,
		"$status":   status,
		"$limit":    fmt.Sprintf("%d", limit),
	}

	resp, err := c.dg.NewTxn().QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by type and status: %v", err)
	}

	var result struct {
		Tasks []TaskNode `json:"tasks"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result.Tasks, nil
}

// GetTaskCreationStats retrieves statistics for task creation
func (c *Client) GetTaskCreationStats(ctx context.Context, userWallet string) (map[string]interface{}, error) {
	query := `
		query getTaskCreationStats($userWallet: string) {
			total: var(func: eq(task_type, "task_creation")) @filter(eq(user_wallet, $userWallet)) {
				count(uid)
			}
			
			verified: var(func: eq(task_type, "task_creation")) @filter(eq(user_wallet, $userWallet) AND eq(status, "VERIFIED")) {
				count(uid)
			}
			
			failed: var(func: eq(task_type, "task_creation")) @filter(eq(user_wallet, $userWallet) AND eq(status, "FAILED")) {
				count(uid)
			}
			
			pending: var(func: eq(task_type, "task_creation")) @filter(eq(user_wallet, $userWallet) AND eq(status, "PENDING_VERIFICATION")) {
				count(uid)
			}
		}
	`

	variables := map[string]string{
		"$userWallet": userWallet,
	}

	resp, err := c.dg.NewTxn().QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to query task creation stats: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return result, nil
}

// Close closes the Dgraph connection
func (c *Client) Close() error {
	// Dgraph client doesn't have a direct close method
	// The connection will be closed when the gRPC connection is closed
	return nil
}
