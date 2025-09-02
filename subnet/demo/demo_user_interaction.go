// Package demo - User Interaction Simulation
//
// This file implements user behavior simulation for the PoC demonstration.
// It provides hardcoded user responses that create realistic interaction patterns:
//   - User acceptance of good outputs
//   - User rejection despite validator approval (demonstrating user sovereignty)
//   - Realistic feedback messages for different scenarios
//
// In production, this would be replaced with actual user interfaces,
// feedback collection systems, or user preference learning algorithms.
package demo

// DemoUserInteractionHandler simulates realistic user feedback patterns for demonstration.
// Models different user behavior scenarios to test the complete PoCW workflow:
//
// User Behavior Patterns:
//   - Default: Accept most outputs (positive user experience)
//   - Input 4: Reject poor quality output (user quality standards)
//   - Input 6: Reject despite validator approval (user has final authority)
//
// This demonstrates user sovereignty in the PoCW protocol - users make final decisions.
type DemoUserInteractionHandler struct{}

// NewDemoUserInteractionHandler creates a new demo user interaction handler
func NewDemoUserInteractionHandler() *DemoUserInteractionHandler {
	return &DemoUserInteractionHandler{}
}

// SimulateUserInteraction models realistic user feedback for different output scenarios.
// Provides varied responses that test different aspects of the PoCW protocol:
//
// Feedback Scenarios:
//   - Input 4: User catches poor quality that should align with validator rejection
//   - Input 6: User exercises final authority despite validator consensus
//   - Others: User satisfaction with quality outputs
//
// Returns user decision (accept/reject) and human-readable feedback message.
func (d *DemoUserInteractionHandler) SimulateUserInteraction(inputNumber int, output string) (bool, string) {
	// Simulate different user response patterns for comprehensive testing
	switch inputNumber {
	case 4:
		// User rejects poor quality output (aligns with validator rejection)
		return false, "I don't think this output addresses my needs correctly."
	case 6:
		// User exercises sovereignty - rejects despite validator acceptance
		// Demonstrates that user has final authority in PoCW protocol
		return false, "This still doesn't meet my requirements despite the additional context."
	default:
		// User accepts satisfactory output (typical positive scenario)
		return true, "This looks good, thank you!"
	}
}