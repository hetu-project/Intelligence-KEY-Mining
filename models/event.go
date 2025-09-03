package models

// Event represents a causal event in the CHRONO graph
type Event struct {
	UID    string      `json:"uid,omitempty"`
	ID     string      `json:"id,omitempty"`
	Name   string      `json:"name,omitempty"`
	Clock  string      `json:"clock,omitempty"`
	Depth  int         `json:"depth,omitempty"`
	Parent []ParentRef `json:"parent,omitempty"`
	Value  string      `json:"value,omitempty"`
	Key    string      `json:"key,omitempty"`
	Node   string      `json:"node,omitempty"`
}

// ParentRef represents a reference to a parent event
type ParentRef struct {
	UID string `json:"uid,omitempty"`
}

// EventInfo holds information about an event for graph reconstruction
type EventInfo struct {
	Key       string
	Value     string
	EventName string
	KeyNum    int
	NodeID    uint64
}
