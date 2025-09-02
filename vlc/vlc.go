package vlc

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Clock represents a verifiable logical clock
type Clock struct {
	Values map[uint64]uint64 `json:"values"` // [Node_clock_id] -> [clock_value]
}

// Comparison result constants
const (
	Less         = -1 // This clock is strictly less than the other
	Equal        = 0  // Clocks are identical
	Greater      = 1  // This clock is strictly greater than the other
	Incomparable = -2 // Clocks have conflicting entries
)

// New creates a new Clock pointer
func New() *Clock {
	return &Clock{
		Values: make(map[uint64]uint64),
	}
}

// Inc increments the clock for a given Node id
func (c *Clock) Inc(id uint64) {
	if c == nil {
		return
	} // Should not happen if initialized with New()
	if c.Values == nil {
		c.Values = make(map[uint64]uint64)
	}
	c.Values[id] = c.Values[id] + 1
}

// Clear resets the clock
func (c *Clock) Clear() {
	if c == nil {
		return
	}
	c.Values = make(map[uint64]uint64)
}

// Merge combines this clock with other clocks by taking the maximum for each entry
func (c *Clock) Merge(others []*Clock) {
	if c == nil {
		return
	} // Cannot merge into nil
	if c.Values == nil {
		c.Values = make(map[uint64]uint64)
	}
	for _, other := range others {
		if other == nil || other.Values == nil {
			continue
		}
		for id, value := range other.Values {
			if currValue, exists := c.Values[id]; !exists || currValue < value {
				c.Values[id] = value
			}
		}
	}
}

// Compare compares two clocks (c vs other)
func (c *Clock) Compare(other *Clock) int {
	cIsNil := c == nil || c.Values == nil || len(c.Values) == 0
	otherIsNil := other == nil || other.Values == nil || len(other.Values) == 0

	if cIsNil {
		if otherIsNil {
			return Equal
		}
		return Less // nil < non-nil
	}
	if otherIsNil {
		return Greater // non-nil > nil
	}

	// Both are non-nil from here
	cLessThanOther := false
	cGreaterThanOther := false

	// Check entries in c
	for id, cValue := range c.Values {
		otherValue, exists := other.Values[id]
		if !exists {
			cGreaterThanOther = true
		} else if cValue < otherValue {
			cLessThanOther = true
		} else if cValue > otherValue {
			cGreaterThanOther = true
		}
	}

	// Check entries in other that might not be in c
	for id := range other.Values {
		if _, exists := c.Values[id]; !exists {
			cLessThanOther = true
		}
	}

	if cLessThanOther && cGreaterThanOther {
		return Incomparable
	}
	if cLessThanOther {
		return Less
	}
	if cGreaterThanOther {
		return Greater
	}
	return Equal
}

// MarshalJSON implements JSON serialization
func (c *Clock) MarshalJSON() ([]byte, error) {
	if c == nil || c.Values == nil {
		return json.Marshal(make(map[uint64]uint64)) // Serialize nil/empty as {}
	}
	return json.Marshal(c.Values)
}

// UnmarshalJSON implements JSON deserialization
func (c *Clock) UnmarshalJSON(data []byte) error {
	// This method is called on a pointer receiver, but the pointer might be nil initially.
	// We need to ensure c itself is allocated if it's nil before unmarshalling into Values.
	// Assuming c is allocated by the caller.
	if c == nil {
		// This shouldn't happen if caller uses &vlc.Clock{} or similar
		return fmt.Errorf("cannot unmarshal into nil Clock pointer")
	}
	c.Values = make(map[uint64]uint64) // Ensure map is initialized
	if string(data) == "null" {
		return nil
	} // Allow unmarshalling null
	return json.Unmarshal(data, &c.Values)
}

// Equals checks if two clocks are semantically equal
func (c *Clock) Equals(other *Clock) bool {
	cIsNil := c == nil || c.Values == nil || len(c.Values) == 0
	otherIsNil := other == nil || other.Values == nil || len(other.Values) == 0
	if cIsNil {
		return otherIsNil
	}
	if otherIsNil {
		return false
	}
	return reflect.DeepEqual(c.Values, other.Values)
}

// Copy creates a deep copy of the Clock and returns a POINTER to it
func (c *Clock) Copy() *Clock {
	newClock := New() // New returns *Clock
	if c != nil && c.Values != nil {
		// newClock.Values is already initialized by New()
		for id, val := range c.Values {
			newClock.Values[id] = val
		}
	}
	return newClock
}

// IsPlusOneIncrement checks if otherClk is exactly +1 ahead of c for senderID,
// and not ahead for any other ID.
func (c *Clock) IsPlusOneIncrement(otherClk *Clock, senderID uint64) bool {
	if otherClk == nil || otherClk.Values == nil {
		return false
	} // Target clock must exist

	otherSenderVal, otherSenderExists := otherClk.Values[senderID]
	if !otherSenderExists {
		return false
	} // Sender must be in target clock

	localSenderVal := uint64(0)
	if c != nil && c.Values != nil {
		localSenderVal = c.Values[senderID]
	}

	if otherSenderVal != localSenderVal+1 {
		return false
	} // Must be exactly +1 for sender

	// For all other entries in otherClk, they must not be ahead of c
	for id, otherVal := range otherClk.Values {
		if id == senderID {
			continue
		}
		localVal := uint64(0)
		if c != nil && c.Values != nil {
			localVal = c.Values[id]
		}
		if otherVal > localVal {
			return false
		} // Other clock cannot be ahead for other IDs
	}

	// For all entries in c not in otherClk, this is fine (otherClk doesn't require them)
	if c != nil && c.Values != nil {
		for id := range c.Values {
			if _, exists := otherClk.Values[id]; !exists {
				// If c has an entry (e.g., {2:5}) and otherClk doesn't (e.g. otherClk={1:1} for sender 1)
				// This is fine. otherClk isn't requiring knowledge about ID 2.
			}
		}
	}
	return true
}
