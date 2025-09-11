package services

import (
	"context"
	"fmt"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/vlc"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// VLCStrategy defines when and how to increment VLC for different task types
type VLCStrategy interface {
	ShouldIncrementOnSubmission(taskType models.TaskType) bool
	ShouldIncrementOnVerification(taskType models.TaskType) bool
	GetIncrementCount(taskType models.TaskType, payload map[string]interface{}) int
	GetEventDescription(taskType models.TaskType, stage string) string
}

// DefaultVLCStrategy implements the default VLC increment strategy
type DefaultVLCStrategy struct{}

// NewDefaultVLCStrategy creates a new default VLC strategy
func NewDefaultVLCStrategy() *DefaultVLCStrategy {
	return &DefaultVLCStrategy{}
}

// ShouldIncrementOnSubmission determines if VLC should increment when task is submitted
func (dvs *DefaultVLCStrategy) ShouldIncrementOnSubmission(taskType models.TaskType) bool {
	switch taskType {
	case models.TaskCreationTask:
		// Immediately increment VLC on task creation as it's an independent event
		return true
	case models.BatchVerificationTask:
		// Don't increment VLC on batch verification task submission, wait for completion
		return false
	case models.TwitterRetweetTask:
		// Don't increment VLC on traditional Twitter retweet task submission
		return false
	default:
		return false
	}
}

// ShouldIncrementOnVerification determines if VLC should increment when task is verified
func (dvs *DefaultVLCStrategy) ShouldIncrementOnVerification(taskType models.TaskType) bool {
	switch taskType {
	case models.TaskCreationTask:
		// Don't increment VLC on task creation verification (already incremented on submission)
		return false
	case models.BatchVerificationTask:
		// Increment VLC on batch verification completion
		return true
	case models.TwitterRetweetTask:
		// Increment VLC on traditional Twitter retweet verification completion
		return true
	default:
		return true
	}
}

// GetIncrementCount determines how much to increment VLC
func (dvs *DefaultVLCStrategy) GetIncrementCount(taskType models.TaskType, payload map[string]interface{}) int {
	switch taskType {
	case models.BatchVerificationTask:
		// Batch verification increments VLC based on number of tasks processed
		if tasks, ok := payload["tasks"].([]interface{}); ok {
			count := len(tasks)
			// Minimum increment 1, maximum increment 10 (avoid VLC growing too fast)
			if count > 10 {
				return 10
			}
			if count < 1 {
				return 1
			}
			return count
		}
		return 1
	default:
		// Other task types default increment 1
		return 1
	}
}

// GetEventDescription provides a description for the VLC increment event
func (dvs *DefaultVLCStrategy) GetEventDescription(taskType models.TaskType, stage string) string {
	switch taskType {
	case models.TaskCreationTask:
		if stage == "submission" {
			return "Task creation submitted"
		}
		return "Task creation verified"
	case models.BatchVerificationTask:
		if stage == "verification" {
			return "Batch verification completed"
		}
		return "Batch verification submitted"
	case models.TwitterRetweetTask:
		if stage == "verification" {
			return "Twitter retweet verified"
		}
		return "Twitter retweet submitted"
	default:
		return fmt.Sprintf("%s %s", taskType, stage)
	}
}

// VLCEvent represents a VLC increment event
type VLCEvent struct {
	TaskID      string                 `json:"task_id"`
	TaskType    models.TaskType        `json:"task_type"`
	Stage       string                 `json:"stage"` // "submission" or "verification"
	Description string                 `json:"description"`
	Increment   int                    `json:"increment"`
	VLCBefore   *vlc.VectorClock       `json:"vlc_before"`
	VLCAfter    *vlc.VectorClock       `json:"vlc_after"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
}

// EnhancedVLCService extends VLCService with strategy-based increments
type EnhancedVLCService struct {
	*VLCService
	strategy VLCStrategy
	events   []VLCEvent // Store VLC event history
}

// NewEnhancedVLCService creates a new enhanced VLC service
func NewEnhancedVLCService(strategy VLCStrategy) *EnhancedVLCService {
	return &EnhancedVLCService{
		VLCService: NewVLCService(),
		strategy:   strategy,
		events:     make([]VLCEvent, 0),
	}
}

// IncrementForTask increments VLC based on task and stage
func (evs *EnhancedVLCService) IncrementForTask(
	ctx context.Context,
	taskID string,
	taskType models.TaskType,
	stage string,
	payload map[string]interface{},
) *vlc.VectorClock {
	// Check if VLC should be incremented
	shouldIncrement := false
	switch stage {
	case "submission":
		shouldIncrement = evs.strategy.ShouldIncrementOnSubmission(taskType)
	case "verification":
		shouldIncrement = evs.strategy.ShouldIncrementOnVerification(taskType)
	}

	if !shouldIncrement {
		return evs.GetCurrentClock()
	}

	// Get increment amount
	incrementCount := evs.strategy.GetIncrementCount(taskType, payload)

	// Record state before increment
	vlcBefore := evs.GetCurrentClock()

	// Execute increment
	var vlcAfter *vlc.VectorClock
	for i := 0; i < incrementCount; i++ {
		vlcAfter = evs.IncrementMinerClock()
	}

	// Record event
	event := VLCEvent{
		TaskID:      taskID,
		TaskType:    taskType,
		Stage:       stage,
		Description: evs.strategy.GetEventDescription(taskType, stage),
		Increment:   incrementCount,
		VLCBefore:   vlcBefore,
		VLCAfter:    vlcAfter,
		Payload:     payload,
	}

	evs.events = append(evs.events, event)

	// Limit event history length
	if len(evs.events) > 1000 {
		evs.events = evs.events[len(evs.events)-1000:]
	}

	return vlcAfter
}

// GetVLCEvents returns recent VLC events
func (evs *EnhancedVLCService) GetVLCEvents(limit int) []VLCEvent {
	if limit <= 0 || limit > len(evs.events) {
		return evs.events
	}

	start := len(evs.events) - limit
	return evs.events[start:]
}

// GetVLCEventsForTask returns VLC events for a specific task
func (evs *EnhancedVLCService) GetVLCEventsForTask(taskID string) []VLCEvent {
	var taskEvents []VLCEvent
	for _, event := range evs.events {
		if event.TaskID == taskID {
			taskEvents = append(taskEvents, event)
		}
	}
	return taskEvents
}
