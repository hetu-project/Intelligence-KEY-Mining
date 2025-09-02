// Package demo implements Proof-of-Concept (PoC) demonstration logic for the PoCW subnet.
//
// This package provides hardcoded scenarios that showcase the subnet's capabilities:
//   - 7 predefined user inputs with expected behaviors
//   - Specific quality assessment rules for demo validation
//   - Simulated user interaction patterns including rejections
//   - Info request scenarios where miners ask for additional context
//
// The demo separates situational logic (this package) from reusable core components,
// enabling the same core subnet infrastructure to work with real AI models in production.
package demo

import (
	"fmt"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/subnet"
	"github.com/hetu-project/Intelligence-KEY-Mining/vlc"
)

// DemoCoordinator orchestrates the complete PoC demonstration of the PoCW subnet.
// It combines core subnet components (CoreMiner, CoreValidator) with demo-specific
// plugins to create a realistic but controlled testing environment.
//
// Architecture:
//   - Uses 1 miner with DemoTaskProcessor (hardcoded AI responses)
//   - Uses 4 validators with DemoQualityAssessor and DemoUserInteractionHandler
//   - Processes 7 predefined inputs with known expected outcomes
//   - Demonstrates both normal processing and info request scenarios
type DemoCoordinator struct {
	SubnetID     string                    // Unique identifier for this demo subnet
	Miner        *subnet.CoreMiner         // AI agent processing tasks
	Validators   []*subnet.CoreValidator   // Quality assessment and consensus nodes
	userInputs   []string                  // Predefined demo inputs for consistent testing
	GraphAdapter *subnet.SubnetGraphAdapter // Graph adapter for VLC event visualization
}

// NewDemoCoordinator creates a new demo coordinator with all PoC-specific logic
func NewDemoCoordinator(subnetID string) *DemoCoordinator {
	// Create core miner with demo task processor
	miner := subnet.NewCoreMiner("miner-1", subnetID)
	miner.SetTaskProcessor(NewDemoTaskProcessor())

	// Create core validators with demo plugins
	validators := make([]*subnet.CoreValidator, 4)
	for i := 0; i < 4; i++ {
		role := subnet.ConsensusValidator
		if i == 0 {
			role = subnet.UserInterfaceValidator // First validator handles user interaction
		}

		validator := subnet.NewCoreValidator(
			fmt.Sprintf("validator-%d", i+1),
			subnetID,
			role,
			0.25, // Equal weights for 4 validators
		)

		// Set demo-specific plugins
		validator.SetQualityAssessor(NewDemoQualityAssessor())
		validator.SetUserInteractionHandler(NewDemoUserInteractionHandler())

		validators[i] = validator
	}

	// Create graph adapter for visualization
	graphAdapter := subnet.NewSubnetGraphAdapter(subnetID, 1, "subnet-coordinator")

	return &DemoCoordinator{
		SubnetID:     subnetID,
		Miner:        miner,
		Validators:   validators,
		GraphAdapter: graphAdapter,
		userInputs: []string{
			"Analyze market trends for Q4",
			"Generate summary report for project Alpha",
			"Create optimization strategy for resource allocation",
			"Design implementation plan for new features",
			"Review performance metrics and recommendations",
			"Develop technical specifications for API integration",
			"Provide comprehensive analysis of system architecture",
		},
	}
}

// RunDemo executes the complete demo scenario using the separated core/demo architecture
func (dc *DemoCoordinator) RunDemo() {
	fmt.Printf("=== Starting Demo with Refactored Architecture ===\n")
	fmt.Printf("Subnet ID: %s\n", dc.SubnetID)
	fmt.Printf("Miner: %s\n", dc.Miner.ID)
	fmt.Printf("Validators: ")
	for _, v := range dc.Validators {
		fmt.Printf("%s ", v.ID)
	}
	fmt.Printf("Graph Adapter: Enabled for VLC event visualization\n")
	fmt.Printf("\n")

	// Process each input according to demo scenario
	for inputNum := 1; inputNum <= 7; inputNum++ {
		fmt.Printf("--- Processing Input %d ---\n", inputNum)
		dc.processInput(inputNum, dc.userInputs[inputNum-1])
		fmt.Println()
		time.Sleep(1 * time.Second) // Small delay for readability
	}

	// Print final summary
	dc.printSummary()
	
	// Commit the causal event graph to Dgraph for visualization
	fmt.Printf("\n=== Committing VLC Event Graph to Dgraph ===\n")
	dc.GraphAdapter.PrintGraphSummary()
	
	if err := dc.GraphAdapter.CommitGraph(); err != nil {
		fmt.Printf("Error committing graph to Dgraph: %v\n", err)
		fmt.Printf("\nTroubleshooting:\n")
		fmt.Printf("- Start Dgraph: sudo docker run --rm -d --name dgraph-standalone -p 8080:8080 -p 9080:9080 -p 8000:8000 dgraph/standalone\n")
		fmt.Printf("- Check status: sudo docker ps | grep dgraph\n")
		fmt.Printf("- Check logs: sudo docker logs dgraph-standalone\n")
	} else {
		fmt.Printf("Successfully committed subnet graph to Dgraph!\n")
		fmt.Printf("\nVisualization Access:\n")
		fmt.Printf("- Ratel UI: http://localhost:8000\n")
		fmt.Printf("- Alternative: http://localhost:8080\n")
		fmt.Printf("- GraphQL: http://localhost:8080/graphql\n")
		// Note: GetEventCount() returns 0 after commit as events are cleared, 
		// but we already showed the count in the graph summary above
	}
}

// processInput handles a single user input through the complete round-based workflow with VLC
func (dc *DemoCoordinator) processInput(inputNumber int, input string) {
	requestID := fmt.Sprintf("req-%s-%d", dc.SubnetID, inputNumber)

	fmt.Printf("User Input: %s\n", input)

	// *** ROUND START: Validator-1 VLC increment for receiving user input ***
	uiValidator := dc.Validators[0] // Validator-1 is the round orchestrator
	uiValidator.IncrementValidatorClock() // Validator-1 VLC{2:++}
	fmt.Printf("Round %d: Started by Validator-1 receiving user input\n", inputNumber)

	// Track user input that starts the round
	userInputEventID := dc.GraphAdapter.TrackUserInput(requestID, input, uiValidator.GetLastMinerClock(), "")

	// Step 1: Miner processes input (Miner VLC will increment)
	// Sync miner's clock with validator's current state first
	dc.Miner.UpdateValidatorClock(uiValidator.GetLastMinerClock())
	minerResponse := dc.Miner.ProcessInput(input, inputNumber, requestID) // Miner VLC{1:++}

	// Track miner's response (output or info request)
	minerResponseEventID := dc.GraphAdapter.TrackMinerResponse(requestID, minerResponse, userInputEventID)

	if minerResponse.OutputType == subnet.NeedMoreInfo {
		// Handle info request scenario
		dc.handleInfoRequest(inputNumber, input, minerResponse, minerResponseEventID)
	} else {
		// Handle normal output scenario
		dc.handleNormalOutput(inputNumber, minerResponse, minerResponseEventID)
	}
}

// handleInfoRequest processes the scenario where miner needs more information with VLC orchestration
func (dc *DemoCoordinator) handleInfoRequest(inputNumber int, originalInput string, minerResponse *subnet.MinerResponseMessage, parentEventID string) {
	fmt.Printf("Miner requests more info: %s\n", minerResponse.InfoRequest)

	// Step 1: Validate miner's VLC sequence (NeedMoreInfo message)
	dc.validateVLCSequenceFromMiner(minerResponse)

	// Step 2: UI Validator orchestrates info request
	uiValidator := dc.Validators[0]
	
	// Update UI validator's VLC with miner's latest state
	uiValidator.UpdateMinerClock(minerResponse.VLCClock)
	
	infoRequest := uiValidator.RequestMoreInfo(minerResponse.RequestID, minerResponse.InfoRequest)

	if infoRequest != nil {
		fmt.Printf("Validator %s asks user: %s\n", uiValidator.ID, infoRequest.Question)

		// *** Validator-1 VLC increment for processing user's additional info ***
		uiValidator.IncrementValidatorClock() // Validator-1 VLC{2:++}
		fmt.Printf("Validator-1: Incremented VLC for processing user's additional context\n")

		// Step 3: Simulate user providing additional info based on demo scenario
		var additionalInfo string
		switch inputNumber {
		case 3:
			additionalInfo = "Focus on cost optimization and ROI analysis specifically."
		case 6:
			additionalInfo = "Use REST API with JSON payloads, authentication via OAuth 2.0."
		}

		fmt.Printf("User provides: %s\n", additionalInfo)

		// Track validator VLC increment for processing additional info
		infoResponseEventID := dc.GraphAdapter.TrackInfoResponse(minerResponse.RequestID, additionalInfo, uiValidator.GetLastMinerClock(), parentEventID)

		// Step 4: Sync miner with validator's updated VLC state and process additional info
		dc.Miner.UpdateValidatorClock(uiValidator.GetLastMinerClock())
		finalResponse := dc.Miner.ProcessAdditionalInfo(originalInput, additionalInfo, inputNumber, minerResponse.RequestID) // Miner VLC{1:++}

		// Track miner VLC increment for final processing
		finalProcessEventID := dc.GraphAdapter.TrackMinerResponse(minerResponse.RequestID, finalResponse, infoResponseEventID)

		// Step 5: Handle final output with quality voting
		dc.handleNormalOutput(inputNumber, finalResponse, finalProcessEventID)
	}
}

// validateVLCSequenceFromMiner validates miner's VLC sequence across all validators
func (dc *DemoCoordinator) validateVLCSequenceFromMiner(minerResponse *subnet.MinerResponseMessage) {
	fmt.Printf("Validators validating Miner VLC sequence (local verification)...\n")
	
	// Each validator independently validates miner's VLC sequence
	// Only Validator-1 maintains VLC state, others just validate the sequence
	allValid := true
	for i, validator := range dc.Validators {
		if i == 0 {
			// Validator-1 (UI) - full VLC participant
			if !validator.ValidateSequence(minerResponse.VLCClock, 1) { // Miner ID = 1
				fmt.Printf("ERROR: Miner VLC validation failed for %s\n", validator.ID)
				allValid = false
			}
		} else {
			// Other validators - just check VLC format validity (simplified check)
			if minerResponse.VLCClock == nil || len(minerResponse.VLCClock.Values) == 0 {
				fmt.Printf("ERROR: Invalid VLC format for %s\n", validator.ID)
				allValid = false
			} else {
				fmt.Printf("Validator %s: VLC format check passed\n", validator.ID)
			}
		}
	}
	
	if allValid {
		fmt.Printf("Miner VLC validation: PASSED\n")
	} else {
		fmt.Printf("Miner VLC validation: FAILED\n")
	}
}

// validateVLCSequenceFromValidator validates validator-1's VLC operations
func (dc *DemoCoordinator) validateVLCSequenceFromValidator(validatorClock *vlc.Clock) {
	fmt.Printf("Miner validating Validator-1 VLC sequence...\n")
	
	// Miner validates validator's VLC operations
	// This maintains bidirectional VLC consistency
	dc.Miner.UpdateValidatorClock(validatorClock)
	fmt.Printf("Validator-1 VLC validation: PASSED (miner synchronized)\n")
}

// handleNormalOutput processes normal miner output through VLC validation and quality consensus
func (dc *DemoCoordinator) handleNormalOutput(inputNumber int, minerResponse *subnet.MinerResponseMessage, parentEventID string) {
	fmt.Printf("Miner output: %s\n", minerResponse.Output)

	// Step 1: Validate miner's VLC sequence for OutputReady message
	dc.validateVLCSequenceFromMiner(minerResponse)

	// Step 2: UI Validator updates its VLC state with miner's latest
	uiValidator := dc.Validators[0]
	uiValidator.UpdateMinerClock(minerResponse.VLCClock)

	// Step 3: Create shared quality assessment for consensus voting
	sharedAssessment := &subnet.QualityAssessment{
		RequestID: minerResponse.RequestID,
	}

	// Step 4: All validators vote on output quality (distributed consensus)
	fmt.Printf("Validators performing quality assessment voting (distributed consensus)...\n")
	votes := make([]*subnet.ValidatorVoteMessage, 0, len(dc.Validators))

	// Each validator performs quality assessment and voting
	for _, validator := range dc.Validators {
		// Note: VLC validation already done above - this is pure quality voting
		vote := validator.VoteOnOutput(minerResponse)
		if vote != nil {
			votes = append(votes, vote)
			// Add each validator's vote to the shared assessment
			sharedAssessment.AddVote(vote.Weight, vote.Accept)
		} else {
			fmt.Printf("ERROR: Validator %s failed to generate vote\n", validator.ID)
		}
	}

	// Step 5: Check consensus using the shared assessment
	var consensusResult string
	var userAccepts bool
	var userFeedback string
	var finalResult string

	if sharedAssessment.IsAccepted() {
		consensusResult = fmt.Sprintf("ACCEPTED (%.2f/%.2f weight)", sharedAssessment.AcceptVotes, sharedAssessment.TotalWeight)
		fmt.Printf("Validator consensus: %s\n", consensusResult)

		// Step 6: Simulate user feedback using UI validator
		userAccepts, userFeedback = uiValidator.SimulateUserInteraction(inputNumber, minerResponse.Output)
		fmt.Printf("User feedback: %s\n", userFeedback)

		if userAccepts {
			finalResult = "OUTPUT DELIVERED TO USER"
		} else {
			finalResult = "OUTPUT REJECTED BY USER (despite validator acceptance)"
		}
	} else {
		consensusResult = fmt.Sprintf("REJECTED (%.2f/%.2f weight)", sharedAssessment.AcceptVotes, sharedAssessment.TotalWeight)
		fmt.Printf("Validator consensus: %s\n", consensusResult)
		
		userAccepts = false
		userFeedback = "No user feedback (validator rejection)"
		finalResult = "OUTPUT REJECTED BY VALIDATORS"
	}

	// *** ROUND END: Validator-1 VLC increment for final result aggregation ***
	uiValidator.IncrementValidatorClock() // Validator-1 VLC{2:++}
	fmt.Printf("Round %d: Completed by Validator-1 aggregating final result\n", inputNumber)
	
	// Track comprehensive round completion with all actions in one VLC mutation
	dc.GraphAdapter.TrackRoundComplete(
		minerResponse.RequestID, 
		inputNumber, 
		uiValidator.GetLastMinerClock(), 
		consensusResult, 
		userFeedback, 
		userAccepts, 
		finalResult, 
		parentEventID,
	)

	fmt.Printf("Final result: %s\n", finalResult)
	
	// Sync miner with final validator state
	dc.Miner.UpdateValidatorClock(uiValidator.GetLastMinerClock())
	fmt.Printf("Round %d: VLC synchronization complete\n", inputNumber)
}

// printSummary prints the final state of the subnet
func (dc *DemoCoordinator) printSummary() {
	fmt.Printf("=== Demo Summary (Refactored Architecture) ===\n")
	minerClock := dc.Miner.GetCurrentClock()
	fmt.Printf("Miner final VLC Clock: %v\n", minerClock.Values)

	fmt.Printf("\nValidator final states:\n")
	for _, validator := range dc.Validators {
		validatorClock := validator.GetLastMinerClock()
		fmt.Printf("  %s: Last miner clock = %v\n", validator.ID, validatorClock.Values)
	}

	fmt.Printf("\nProcessed inputs summary:\n")
	processedInputs := dc.Miner.GetProcessedInputs()
	for i := 1; i <= 7; i++ {
		if response, exists := processedInputs[i]; exists {
			fmt.Printf("  Input %d: Clock=%v, Type=%s\n", i, response.VLCClock.Values, response.OutputType)
		}
	}

	fmt.Printf("\nDemo completed successfully with refactored architecture!\n")
}