// Package demo - Quality Assessment Implementation
//
// This file implements the DemoQualityAssessor, which provides hardcoded quality
// scoring logic for the PoC demonstration. It maps input numbers to predetermined
// quality scores and acceptance decisions to create predictable test scenarios.
//
// In production, this would be replaced with sophisticated quality metrics:
//   - AI model confidence scores
//   - Domain-specific quality indicators
//   - Multi-dimensional evaluation criteria
//   - Machine learning-based quality prediction
package demo

import "github.com/hetu-project/Intelligence-KEY-Mining/subnet"

// DemoQualityAssessor implements predetermined quality assessment for demonstration.
// Uses hardcoded rules based on input numbers to create consistent test scenarios:
//
// Quality Assessment Rules:
//   - Input 1,2,5,7: High quality (0.85), Accept - represents good AI output
//   - Input 3,6: Medium quality (0.75), Accept - represents acceptable output with context
//   - Input 4: Low quality (0.45), Reject - represents poor AI output validators catch
//
// This enables testing validator consensus, user interaction, and rejection scenarios.
type DemoQualityAssessor struct{}

// NewDemoQualityAssessor creates a new demo quality assessor
func NewDemoQualityAssessor() *DemoQualityAssessor {
	return &DemoQualityAssessor{}
}

// AssessQuality evaluates miner output using predetermined demo logic.
// Maps input numbers to specific quality scores and acceptance decisions
// to create predictable test scenarios for the PoC demonstration.
//
// Quality Scoring Logic:
//   - 0.85: High quality, clear acceptance (inputs 1,2,5,7)
//   - 0.75: Medium quality, conditional acceptance (inputs 3,6)
//   - 0.45: Low quality, clear rejection (input 4)
//   - 0.60: Default quality for unknown inputs
//
// Acceptance decisions simulate real validator behavior patterns.
func (d *DemoQualityAssessor) AssessQuality(response *subnet.MinerResponseMessage) (float64, bool) {
	// Map input numbers to predetermined quality assessments for demo consistency
	inputNum := response.InputNumber

	switch inputNum {
	case 1, 2, 5, 7:
		// High-quality outputs that validators confidently accept
		return 0.85, true
	case 3, 6:
		// Medium-quality outputs accepted by validators but may face user rejection
		return 0.75, true
	case 4:
		// Low-quality output that validators properly reject (quality gate functioning)
		return 0.45, false
	default:
		// Default moderate quality for extensibility
		return 0.60, true
	}
}