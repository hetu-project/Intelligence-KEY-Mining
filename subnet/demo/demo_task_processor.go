// Package demo - Task Processing Implementation
//
// This file implements the DemoTaskProcessor, which provides hardcoded AI responses
// for the PoC demonstration. It simulates different AI behaviors including:
//   - Immediate solution generation (OutputReady)
//   - Information requests for clarification (NeedMoreInfo)
//   - Quality variations that trigger different validator responses
//
// In production, this would be replaced with real AI models, LLMs, or other
// intelligent processing systems while maintaining the same TaskProcessor interface.
package demo

import (
	"fmt"

	"github.com/hetu-project/Intelligence-KEY-Mining/subnet"
)

// DemoTaskProcessor implements the TaskProcessor interface with predefined responses
// for demonstration purposes. Each input number triggers specific behavior patterns:
//
// Demo Scenario Map:
//   Input 1, 2, 5, 7: Immediate high-quality solutions (OutputReady)
//   Input 3, 6: Request additional context (NeedMoreInfo) then generate solutions
//   Input 4: Generate low-quality solution that validators should reject
//
// This enables testing all aspects of the PoCW protocol in a controlled manner.
type DemoTaskProcessor struct{}

// NewDemoTaskProcessor creates a new demo task processor
func NewDemoTaskProcessor() *DemoTaskProcessor {
	return &DemoTaskProcessor{}
}

// ProcessTask implements the demo scenario logic
func (d *DemoTaskProcessor) ProcessTask(input string, inputNumber int) (subnet.MinerOutputType, string, string) {
	switch inputNumber {
	case 3:
		// Input 3: Miner requests more info → normal flow
		fmt.Printf("Miner: Input %d - Requesting more information\n", inputNumber)
		return subnet.NeedMoreInfo, "", "Could you please provide more context about what specific aspect you'd like me to focus on?"

	case 6:
		// Input 6: Miner requests more info → will eventually be rejected by user
		fmt.Printf("Miner: Input %d - Requesting more information (will be rejected later)\n", inputNumber)
		return subnet.NeedMoreInfo, "", "I need clarification on the technical requirements. Could you specify the exact parameters?"

	default:
		// Normal processing for inputs 1, 2, 4, 5, 7
		output := d.generateOutput(input, inputNumber)
		fmt.Printf("Miner: Input %d - Generated output: %s\n", inputNumber, output)
		return subnet.OutputReady, output, ""
	}
}

// ProcessAdditionalInfo processes additional information for demo scenarios
func (d *DemoTaskProcessor) ProcessAdditionalInfo(originalInput string, additionalInfo string, inputNumber int) string {
	// Generate output based on original input + additional info
	combinedInput := fmt.Sprintf("%s [Additional context: %s]", originalInput, additionalInfo)
	output := d.generateOutput(combinedInput, inputNumber)
	fmt.Printf("Miner: Input %d - Generated output with additional info: %s\n", inputNumber, output)
	return output
}

// generateOutput simulates AI processing and generates demo-specific outputs
func (d *DemoTaskProcessor) generateOutput(input string, inputNumber int) string {
	// Simulate different types of outputs based on input number
	switch inputNumber {
	case 1:
		return "Analyzed your request and generated comprehensive solution A"
	case 2:
		return "Processed data and created detailed response B"
	case 3:
		return "With additional context, here's refined solution C"
	case 4:
		return "Generated output D that will be rejected by user"
	case 5:
		return "Standard processing result E"
	case 6:
		return "Enhanced output F with clarifications (will be rejected)"
	case 7:
		return "Final comprehensive solution G"
	default:
		return fmt.Sprintf("Processed input: %s", input)
	}
}