package utils

import (
	"fmt"
	"math"
)

// FormatFinancialValue formats financial values for frontend display
// Values are displayed with K/M/B abbreviations and one decimal place
// Examples: 1500 -> "1.5K", 2000000 -> "2.0M", 3500000000 -> "3.5B"
func FormatFinancialValue(value float64) string {
	if value == 0 {
		return "0"
	}

	absValue := math.Abs(value)
	sign := ""
	if value < 0 {
		sign = "-"
	}

	switch {
	case absValue >= 1e9: // Billions
		formatted := absValue / 1e9
		return fmt.Sprintf("%s%.1fB", sign, formatted)
	case absValue >= 1e6: // Millions
		formatted := absValue / 1e6
		return fmt.Sprintf("%s%.1fM", sign, formatted)
	case absValue >= 1e3: // Thousands
		formatted := absValue / 1e3
		return fmt.Sprintf("%s%.1fK", sign, formatted)
	default:
		// For values less than 1000, show as is with one decimal place
		return fmt.Sprintf("%s%.1f", sign, absValue)
	}
}

// ParseFinancialValue parses formatted financial values back to float64
// This is useful for API responses that need both raw and formatted values
func ParseFinancialValue(formatted string) (float64, error) {
	// This function would parse "1.5K" back to 1500.0
	// Implementation depends on whether you need bidirectional conversion
	// For now, we'll keep it simple and just return an error
	return 0, fmt.Errorf("parsing formatted financial values not implemented")
}
