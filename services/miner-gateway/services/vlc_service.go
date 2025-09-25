package services

import (
	"sync"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/vlc"
)

// VLCService manages Vector Logical Clock for the miner
type VLCService struct {
	clock *vlc.VectorClock
	mutex sync.RWMutex
}

// NewVLCService creates a new VLC service
func NewVLCService() *VLCService {
	return &VLCService{
		clock: vlc.NewVectorClock(1), // Miner ID = 1
	}
}

// IncrementMinerClock increments the miner's logical clock
func (vs *VLCService) IncrementMinerClock() *vlc.VectorClock {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	vs.clock.Increment()
	return vs.clock.Copy()
}

// GetCurrentClock returns a copy of the current clock
func (vs *VLCService) GetCurrentClock() *vlc.VectorClock {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()

	return vs.clock.Copy()
}

// UpdateClock updates the clock based on received clock
func (vs *VLCService) UpdateClock(receivedClock *vlc.VectorClock) {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	vs.clock.Update(receivedClock)
}

// GetClockValue returns the current clock value for a specific process
func (vs *VLCService) GetClockValue(processID int) int {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()

	return vs.clock.GetValue(processID)
}

// GetClockState returns the current state of the VLC
func (vs *VLCService) GetClockState() map[string]interface{} {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()

	return map[string]interface{}{
		"process_id": vs.clock.ProcessID,
		"values":     vs.clock.Values,
		"timestamp":  time.Now(),
	}
}
