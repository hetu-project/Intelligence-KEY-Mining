package services

import (
	"strconv"
	"sync"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/vlc"
)

// VLCService manages Vector Logical Clock for the validator
type VLCService struct {
	clock      *vlc.VectorClock
	minerClock *vlc.VectorClock // Track Miner's clock state
	mutex      sync.RWMutex
}

// NewVLCService creates a new VLC service for validator
func NewVLCService(validatorID string) *VLCService {
	// Extract numeric part from validator ID as process ID
	processID := extractProcessIDFromValidatorID(validatorID)

	return &VLCService{
		clock:      vlc.NewVectorClock(processID),
		minerClock: vlc.NewVectorClock(1), // Miner fixed as process ID 1
	}
}

// ValidateVLCSequence validates the VLC sequence from miner
func (vs *VLCService) ValidateVLCSequence(minerVLC *vlc.VectorClock, minerID int) bool {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()

	if minerVLC == nil || len(minerVLC.Values) == 0 {
		return false
	}

	// Check if VLC increments legally
	if minerVLC.GetValue(minerID) <= vs.minerClock.GetValue(minerID) {
		return false
	}

	// Check for causality violations
	for processID, value := range vs.minerClock.Values {
		if processID != minerID && minerVLC.GetValue(processID) < value {
			return false
		}
	}

	return true
}

// UpdateMinerClock updates the tracked miner clock
func (vs *VLCService) UpdateMinerClock(minerVLC *vlc.VectorClock) {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	if minerVLC != nil {
		vs.minerClock.Update(minerVLC)
	}
}

// IncrementValidatorClock increments the validator's own clock
func (vs *VLCService) IncrementValidatorClock() *vlc.VectorClock {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	vs.clock.Increment()
	return vs.clock.Copy()
}

// GetCurrentVLCState returns the current VLC state
func (vs *VLCService) GetCurrentVLCState() map[string]interface{} {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()

	return map[string]interface{}{
		"validator_clock": map[string]interface{}{
			"process_id": vs.clock.ProcessID,
			"values":     vs.clock.Values,
		},
		"miner_clock": map[string]interface{}{
			"process_id": vs.minerClock.ProcessID,
			"values":     vs.minerClock.Values,
		},
		"timestamp": time.Now(),
	}
}

// GetMinerClockState returns the current miner clock state
func (vs *VLCService) GetMinerClockState() *vlc.VectorClock {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()

	return vs.minerClock.Copy()
}

// GetCurrentClock returns the current validator clock state
func (vs *VLCService) GetCurrentClock() *vlc.VectorClock {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()

	return vs.clock.Copy()
}

// extractProcessIDFromValidatorID extracts process ID from validator ID
// validator-1 -> 2, validator-2 -> 3, etc.
func extractProcessIDFromValidatorID(validatorID string) int {
	// Simple mapping: validator-1 -> 2, validator-2 -> 3, ...
	// Reserve process ID 1 for Miner
	if len(validatorID) > 10 && validatorID[:10] == "validator-" {
		if id, err := strconv.Atoi(validatorID[10:]); err == nil {
			return id + 1 // validator-1 -> 2, validator-2 -> 3
		}
	}

	// Return a safe default process ID
	return 2
}
