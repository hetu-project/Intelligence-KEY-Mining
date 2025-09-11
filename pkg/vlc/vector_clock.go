package vlc

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// VectorClock represents a vector logical clock
type VectorClock struct {
	ProcessID int          `json:"process_id"`
	Values    map[int]int  `json:"values"`
	mutex     sync.RWMutex `json:"-"`
	Timestamp time.Time    `json:"timestamp"`
}

// NewVectorClock creates a new vector clock for the given process
func NewVectorClock(processID int) *VectorClock {
	return &VectorClock{
		ProcessID: processID,
		Values:    make(map[int]int),
		Timestamp: time.Now(),
	}
}

// Increment increments the clock for the local process
func (vc *VectorClock) Increment() {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()

	vc.Values[vc.ProcessID]++
	vc.Timestamp = time.Now()
}

// Update updates the clock based on a received clock
func (vc *VectorClock) Update(other *VectorClock) {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()

	// Update clock values for all processes
	for processID, value := range other.Values {
		if currentValue, exists := vc.Values[processID]; !exists || value > currentValue {
			vc.Values[processID] = value
		}
	}

	// Increment local clock
	vc.Values[vc.ProcessID]++
	vc.Timestamp = time.Now()
}

// GetValue returns the clock value for a specific process
func (vc *VectorClock) GetValue(processID int) int {
	vc.mutex.RLock()
	defer vc.mutex.RUnlock()

	if value, exists := vc.Values[processID]; exists {
		return value
	}
	return 0
}

// Copy creates a deep copy of the vector clock
func (vc *VectorClock) Copy() *VectorClock {
	vc.mutex.RLock()
	defer vc.mutex.RUnlock()

	newClock := &VectorClock{
		ProcessID: vc.ProcessID,
		Values:    make(map[int]int),
		Timestamp: vc.Timestamp,
	}

	for processID, value := range vc.Values {
		newClock.Values[processID] = value
	}

	return newClock
}

// Compare compares two vector clocks
// Returns: -1 if vc < other, 0 if concurrent, 1 if vc > other
func (vc *VectorClock) Compare(other *VectorClock) int {
	vc.mutex.RLock()
	defer vc.mutex.RUnlock()

	if other == nil {
		return 1
	}

	// Collect all process IDs
	allProcesses := make(map[int]bool)
	for processID := range vc.Values {
		allProcesses[processID] = true
	}
	for processID := range other.Values {
		allProcesses[processID] = true
	}

	lessThan := false
	greaterThan := false

	for processID := range allProcesses {
		vcValue := vc.GetValue(processID)
		otherValue := other.GetValue(processID)

		if vcValue < otherValue {
			lessThan = true
		} else if vcValue > otherValue {
			greaterThan = true
		}
	}

	if lessThan && !greaterThan {
		return -1 // vc < other
	} else if !lessThan && greaterThan {
		return 1 // vc > other
	} else {
		return 0 // concurrent
	}
}

// HappensBefore checks if this event happened before another
func (vc *VectorClock) HappensBefore(other *VectorClock) bool {
	return vc.Compare(other) == -1
}

// HappensAfter checks if this event happened after another
func (vc *VectorClock) HappensAfter(other *VectorClock) bool {
	return vc.Compare(other) == 1
}

// IsConcurrent checks if two events are concurrent
func (vc *VectorClock) IsConcurrent(other *VectorClock) bool {
	return vc.Compare(other) == 0
}

// String returns a string representation of the vector clock
func (vc *VectorClock) String() string {
	data, _ := json.Marshal(vc)
	return string(data)
}

// ToJSON converts the vector clock to JSON bytes
func (vc *VectorClock) ToJSON() ([]byte, error) {
	vc.mutex.RLock()
	defer vc.mutex.RUnlock()

	return json.Marshal(vc)
}

// FromJSON creates a vector clock from JSON bytes
func FromJSON(data []byte) (*VectorClock, error) {
	var vc VectorClock
	err := json.Unmarshal(data, &vc)
	if err != nil {
		return nil, err
	}

	if vc.Values == nil {
		vc.Values = make(map[int]int)
	}

	return &vc, nil
}

// Validate validates the vector clock structure
func (vc *VectorClock) Validate() error {
	vc.mutex.RLock()
	defer vc.mutex.RUnlock()

	if vc.Values == nil {
		return fmt.Errorf("values map cannot be nil")
	}

	if vc.ProcessID <= 0 {
		return fmt.Errorf("process ID must be positive")
	}

	// Check if local process clock value exists
	if _, exists := vc.Values[vc.ProcessID]; !exists {
		return fmt.Errorf("missing clock value for local process %d", vc.ProcessID)
	}

	// Check if all clock values are non-negative
	for processID, value := range vc.Values {
		if value < 0 {
			return fmt.Errorf("negative clock value for process %d: %d", processID, value)
		}
	}

	return nil
}

// GetProcesses returns all process IDs in the clock
func (vc *VectorClock) GetProcesses() []int {
	vc.mutex.RLock()
	defer vc.mutex.RUnlock()

	processes := make([]int, 0, len(vc.Values))
	for processID := range vc.Values {
		processes = append(processes, processID)
	}

	return processes
}

// IsEmpty checks if the vector clock is empty
func (vc *VectorClock) IsEmpty() bool {
	vc.mutex.RLock()
	defer vc.mutex.RUnlock()

	return len(vc.Values) == 0
}

// Reset resets the vector clock
func (vc *VectorClock) Reset() {
	vc.mutex.Lock()
	defer vc.mutex.Unlock()

	vc.Values = make(map[int]int)
	vc.Timestamp = time.Now()
}
