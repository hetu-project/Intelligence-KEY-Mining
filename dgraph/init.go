package dgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/hetu-project/Intelligence-KEY-Mining/models"
)

// EventGraph represents the causal event graph
type EventGraph struct {
	Events   []models.Event
	UIDMap   map[string]string
	EventMu  sync.RWMutex
	Depth    int
	NodeID   int
	NodeAddr string
}

// NewEventGraph creates a new event graph instance
func NewEventGraph(nodeID int, nodeAddr string) *EventGraph {
	return &EventGraph{
		Events:   make([]models.Event, 0),
		UIDMap:   make(map[string]string),
		Depth:    0,
		NodeID:   nodeID,
		NodeAddr: nodeAddr,
	}
}

// VectorClockToString converts a vector clock to JSON string
func VectorClockToString(vc map[int]int) string {
	b, _ := json.Marshal(vc)
	return string(b)
}

// AddEvent adds a new event to the graph
func (eg *EventGraph) AddEvent(name string, key string, value string, clock map[int]int, parentIDs []string) string {
	eg.EventMu.Lock()
	defer eg.EventMu.Unlock()

	eg.Depth++
	eventID := fmt.Sprintf("e%d_%d", eg.NodeID, eg.Depth)

	eventUID := fmt.Sprintf("_:%s_%s_%s_%d", name, key, value, eg.Depth)

	for id, uid := range eg.UIDMap {
		if strings.Contains(uid, eventUID) {
			return id
		}
	}

	event := models.Event{
		UID:   eventUID,
		ID:    eventID,
		Name:  name,
		Clock: VectorClockToString(clock),
		Depth: eg.Depth,
		Key:   key,
		Value: value,
		Node:  eg.NodeAddr,
	}

	if len(parentIDs) > 0 {
		event.Parent = make([]models.ParentRef, 0, len(parentIDs))
		for _, id := range parentIDs {
			if uid, ok := eg.UIDMap[id]; ok {
				event.Parent = append(event.Parent, models.ParentRef{UID: uid})
			}
		}
	}

	eg.Events = append(eg.Events, event)
	eg.UIDMap[eventID] = event.UID

	return eventID
}

// CommitToGraph commits all pending events to Dgraph
func (eg *EventGraph) CommitToGraph() error {
	eg.EventMu.Lock()
	defer eg.EventMu.Unlock()

	if len(eg.Events) == 0 {
		return nil
	}

	mutationJSON, err := json.Marshal(eg.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %v", err)
	}

	txn := Dg.NewTxn()
	defer txn.Discard(context.Background())

	mu := &api.Mutation{
		SetJson:   mutationJSON,
		CommitNow: true,
	}

	if _, err := txn.Mutate(context.Background(), mu); err != nil {
		return fmt.Errorf("failed to commit events to Dgraph: %v", err)
	}

	eg.Events = make([]models.Event, 0)

	log.Println("Chrono event graph committed to Dgraph")
	return nil
}

// StartAutoCommit starts automatic periodic commits to Dgraph
func (eg *EventGraph) StartAutoCommit(interval time.Duration) chan struct{} {
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := eg.CommitToGraph(); err != nil {
					log.Printf("Auto-commit error: %v", err)
				}
			case <-done:
				return
			}
		}
	}()

	return done
}
