package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/protocol"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// ValidatorClient handles communication with validator services
type ValidatorClient struct {
	config     *protocol.NetworkConfig
	httpClient *http.Client
	minerID    string
}

// NewValidatorClient creates a new validator client
func NewValidatorClient(config *protocol.NetworkConfig, minerID string) *ValidatorClient {
	return &ValidatorClient{
		config:  config,
		minerID: minerID,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
		},
	}
}

// SendMinerOutput sends miner output to all validators for voting
func (vc *ValidatorClient) SendMinerOutput(ctx context.Context, minerOutput *models.MinerOutput) error {
	// Build protocol message
	request := &protocol.MinerOutputRequest{
		BaseMessage: protocol.BaseMessage{
			Type:      protocol.MinerOutputMessage,
			MessageID: uuid.New().String(),
			Timestamp: time.Now(),
		},
		TaskID:   minerOutput.TaskID,
		MinerID:  minerOutput.MinerID,
		EventID:  minerOutput.EventID,
		VLCClock: minerOutput.VLCClock,
		Payload:  minerOutput.Payload,
		Proof: &protocol.TaskProof{
			Provider:       minerOutput.Proof.Provider,
			VerifiedAt:     minerOutput.Proof.VerifiedAt,
			Evidence:       minerOutput.Proof.Evidence,
			VerificationID: minerOutput.Proof.VerificationID,
			Signature:      minerOutput.Proof.Signature,
		},
		RequestID: uuid.New().String(),
	}

	// Sign message
	if err := vc.signMessage(&request.BaseMessage); err != nil {
		return fmt.Errorf("failed to sign message: %v", err)
	}

	// Send to all validators in parallel
	var wg sync.WaitGroup
	votes := make([]*protocol.ValidatorVoteResponse, 0, len(vc.config.ValidatorEndpoints))
	errors := make([]error, 0, len(vc.config.ValidatorEndpoints))
	voteMutex := sync.Mutex{}

	for _, endpoint := range vc.config.ValidatorEndpoints {
		wg.Add(1)
		go func(ep protocol.ValidatorEndpoint) {
			defer wg.Done()

			vote, err := vc.sendToValidator(ctx, request, ep)

			voteMutex.Lock()
			defer voteMutex.Unlock()

			if err != nil {
				errors = append(errors, fmt.Errorf("validator %s error: %v", ep.ID, err))
			} else if vote != nil {
				votes = append(votes, vote)
			}
		}(endpoint)
	}

	wg.Wait()

	// Check if there are enough votes
	if len(votes) == 0 {
		return fmt.Errorf("no votes received from validators: %v", errors)
	}

	// Send votes to aggregator
	if err := vc.sendVotesToAggregator(ctx, request.EventID, votes); err != nil {
		return fmt.Errorf("failed to send votes to aggregator: %v", err)
	}

	return nil
}

// sendToValidator sends request to a single validator
func (vc *ValidatorClient) sendToValidator(ctx context.Context, request *protocol.MinerOutputRequest, endpoint protocol.ValidatorEndpoint) (*protocol.ValidatorVoteResponse, error) {
	// Build verification request
	validationReq := &protocol.ValidationRequest{
		MinerOutput: request,
	}

	reqBody, err := json.Marshal(validationReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	url := endpoint.URL + protocol.ValidateEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Miner-ID", vc.minerID)
	req.Header.Set("X-Request-ID", request.RequestID)

	// Retry logic
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= vc.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(vc.config.RetryInterval)
		}

		resp, lastErr = vc.httpClient.Do(req)
		if lastErr == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed after %d attempts: %v", vc.config.MaxRetries+1, lastErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("validator returned status %d", resp.StatusCode)
	}

	// Parse response
	var validationResp protocol.ValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validationResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if !validationResp.Success {
		return nil, fmt.Errorf("validation failed: %s", validationResp.Error)
	}

	// Verify vote signature
	if err := vc.verifyVoteSignature(validationResp.Vote); err != nil {
		return nil, fmt.Errorf("invalid vote signature: %v", err)
	}

	return validationResp.Vote, nil
}

// sendVotesToAggregator sends collected votes to aggregator
func (vc *ValidatorClient) sendVotesToAggregator(ctx context.Context, eventID string, votes []*protocol.ValidatorVoteResponse) error {
	if vc.config.AggregatorEndpoint == "" {
		// If no aggregator, process consensus locally
		return vc.processConsensusLocally(ctx, eventID, votes)
	}

	// Send to aggregator service
	for _, vote := range votes {
		if err := vc.sendSingleVoteToAggregator(ctx, vote); err != nil {
			// Log error but continue sending other votes
			fmt.Printf("Warning: failed to send vote to aggregator: %v\n", err)
		}
	}

	return nil
}

// sendSingleVoteToAggregator sends a single vote to aggregator
func (vc *ValidatorClient) sendSingleVoteToAggregator(ctx context.Context, vote *protocol.ValidatorVoteResponse) error {
	reqBody, err := json.Marshal(vote)
	if err != nil {
		return fmt.Errorf("failed to marshal vote: %v", err)
	}

	url := vc.config.AggregatorEndpoint + protocol.SubmitVoteEndpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Miner-ID", vc.minerID)

	resp, err := vc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("aggregator returned status %d", resp.StatusCode)
	}

	return nil
}

// processConsensusLocally processes consensus locally (fallback)
func (vc *ValidatorClient) processConsensusLocally(ctx context.Context, eventID string, votes []*protocol.ValidatorVoteResponse) error {
	// Implement local PoCW consensus logic (reuse existing logic)
	totalWeight := 0.0
	acceptWeight := 0.0

	for _, vote := range votes {
		totalWeight += vote.Weight
		if vote.Vote == "accept" {
			acceptWeight += vote.Weight
		}
	}

	// PoCW consensus threshold: accept with over 50% weight
	consensusReached := acceptWeight > totalWeight*0.5
	finalDecision := "rejected"
	if consensusReached {
		finalDecision = "accepted"
	}

	fmt.Printf("🏛️ Consensus Result for Event %s:\n", eventID)
	fmt.Printf("   Total Weight: %.2f\n", totalWeight)
	fmt.Printf("   Accept Weight: %.2f\n", acceptWeight)
	fmt.Printf("   Final Decision: %s\n", finalDecision)

	// Here can trigger subsequent processing (such as blockchain submission)
	if consensusReached {
		// Trigger KEY token mining or other reward mechanisms
		fmt.Printf("✅ Task verified - triggering rewards\n")
	}

	return nil
}

// Health check methods

// CheckValidatorHealth checks health of all validators
func (vc *ValidatorClient) CheckValidatorHealth(ctx context.Context) map[string]bool {
	results := make(map[string]bool)
	var wg sync.WaitGroup
	resultMutex := sync.Mutex{}

	for _, endpoint := range vc.config.ValidatorEndpoints {
		wg.Add(1)
		go func(ep protocol.ValidatorEndpoint) {
			defer wg.Done()

			healthy := vc.checkSingleValidatorHealth(ctx, ep)

			resultMutex.Lock()
			results[ep.ID] = healthy
			resultMutex.Unlock()
		}(endpoint)
	}

	wg.Wait()
	return results
}

// checkSingleValidatorHealth checks health of a single validator
func (vc *ValidatorClient) checkSingleValidatorHealth(ctx context.Context, endpoint protocol.ValidatorEndpoint) bool {
	url := endpoint.URL + protocol.HealthEndpoint
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := vc.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// Signature methods (placeholder - implement with actual crypto)

func (vc *ValidatorClient) signMessage(message *protocol.BaseMessage) error {
	// Implement message signing
	// Need to use Miner's private key to sign the message
	message.Signature = "miner_signature_placeholder"
	return nil
}

func (vc *ValidatorClient) verifyVoteSignature(vote *protocol.ValidatorVoteResponse) error {
	// Implement vote signature verification
	// Need to verify Validator's signature
	if vote.Signature == "" {
		return fmt.Errorf("missing vote signature")
	}
	return nil
}
