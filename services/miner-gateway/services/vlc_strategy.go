package services

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

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
	case models.TwitterRetweetTask:
		// Increment VLC on Twitter retweet task creation (user creates retweet task)
		return true
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
	case models.TwitterRetweetTask:
		// Increment VLC on Twitter retweet verification completion
		return true
	default:
		return true
	}
}

// GetIncrementCount determines how much to increment VLC
func (dvs *DefaultVLCStrategy) GetIncrementCount(taskType models.TaskType, payload map[string]interface{}) int {
	// All task types increment VLC by 1
	// (Batch operations are handled at the operation level, not task level)
	return 1
}

// GetEventDescription provides a description for the VLC increment event
func (dvs *DefaultVLCStrategy) GetEventDescription(taskType models.TaskType, stage string) string {
	switch taskType {
	case models.TaskCreationTask:
		if stage == "submission" {
			return "Task creation submitted"
		}
		return "Task creation verified"
	case models.TwitterRetweetTask:
		if stage == "verification" {
			return "Twitter retweet task verified"
		}
		return "Twitter retweet task created"
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
// Supports dual-layer VLC: Miner node VLC + User SBT VLC
type EnhancedVLCService struct {
	minerVLC  *VLCService                 // Miner node VLC (ID=1)
	userVLCs  map[string]*vlc.VectorClock // User SBT VLCs (key=wallet, ProcessID=hash)
	userMux   sync.RWMutex                // Protects userVLCs map
	strategy  VLCStrategy
	events    []VLCEvent   // Store VLC event history
	eventsMux sync.RWMutex // Protects events slice
}

// NewEnhancedVLCService creates a new enhanced VLC service with dual-layer VLC
func NewEnhancedVLCService(strategy VLCStrategy) *EnhancedVLCService {
	return &EnhancedVLCService{
		minerVLC: NewVLCService(),                   // Miner node VLC (ID=1)
		userVLCs: make(map[string]*vlc.VectorClock), // User SBT VLCs
		strategy: strategy,
		events:   make([]VLCEvent, 0),
	}
}

// getUserVLCValues safely extracts VLC values from a VectorClock
func getUserVLCValues(vlc *vlc.VectorClock) map[int]int {
	if vlc != nil {
		return vlc.Values
	}
	return nil
}

// generateUserProcessID generates a unique ProcessID for a user wallet
func (evs *EnhancedVLCService) generateUserProcessID(userWallet string) int {
	hash := sha256.Sum256([]byte(userWallet))
	// Use first 4 bytes as ProcessID, ensure it's >= 2 (since 1 is reserved for miner)
	processID := int(binary.BigEndian.Uint32(hash[:4]))
	if processID < 2 {
		processID = processID + 2
	}
	return processID
}

// getUserVLC gets or creates a VLC for a specific user
func (evs *EnhancedVLCService) getUserVLC(userWallet string) *vlc.VectorClock {
	evs.userMux.Lock()
	defer evs.userMux.Unlock()

	if userVLC, exists := evs.userVLCs[userWallet]; exists {
		return userVLC
	}

	// Create new user VLC
	processID := evs.generateUserProcessID(userWallet)
	userVLC := vlc.NewVectorClock(processID)
	evs.userVLCs[userWallet] = userVLC
	return userVLC
}

// IncrementForTask increments dual-layer VLC based on task and stage
func (evs *EnhancedVLCService) IncrementForTask(
	ctx context.Context,
	taskID string,
	taskType models.TaskType,
	stage string,
	payload map[string]interface{},
	userWallet string, // New parameter for SBT user identification
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
		// Return current user VLC if available, otherwise miner VLC
		if userWallet != "" {
			return evs.getUserVLC(userWallet).Copy()
		}
		return evs.minerVLC.GetCurrentClock()
	}

	// Get increment amount
	incrementCount := evs.strategy.GetIncrementCount(taskType, payload)

	// Dual-layer VLC increment:
	// 1. Always increment Miner node VLC (represents total processing)
	minerVLCBefore := evs.minerVLC.GetCurrentClock()
	var minerVLCAfter *vlc.VectorClock
	for i := 0; i < incrementCount; i++ {
		minerVLCAfter = evs.minerVLC.IncrementMinerClock()
	}

	// 2. Increment User SBT VLC if user wallet provided
	var userVLCBefore, userVLCAfter *vlc.VectorClock
	if userWallet != "" {
		userVLC := evs.getUserVLC(userWallet)
		userVLCBefore = userVLC.Copy()

		// Increment user VLC
		evs.userMux.Lock()
		for i := 0; i < incrementCount; i++ {
			userVLC.Increment()
		}
		userVLCAfter = userVLC.Copy()
		evs.userMux.Unlock()
	}

	// Record dual-layer event
	evs.eventsMux.Lock()
	event := VLCEvent{
		TaskID:      taskID,
		TaskType:    taskType,
		Stage:       stage,
		Description: evs.strategy.GetEventDescription(taskType, stage),
		Increment:   incrementCount,
		VLCBefore:   minerVLCBefore, // Primary is miner VLC
		VLCAfter:    minerVLCAfter,
		Payload: map[string]interface{}{
			"original_payload": payload,
			"user_wallet":      userWallet,
			"miner_vlc_before": minerVLCBefore.Values,
			"miner_vlc_after":  minerVLCAfter.Values,
			"user_vlc_before":  getUserVLCValues(userVLCBefore),
			"user_vlc_after":   getUserVLCValues(userVLCAfter),
		},
	}

	evs.events = append(evs.events, event)

	// Limit event history length
	if len(evs.events) > 1000 {
		evs.events = evs.events[len(evs.events)-1000:]
	}
	evs.eventsMux.Unlock()

	// Return user VLC if available, otherwise miner VLC
	if userVLCAfter != nil {
		return userVLCAfter
	}
	return minerVLCAfter
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

// ===== Compatibility methods for existing code =====

// GetCurrentClock returns the miner node VLC for backward compatibility
func (evs *EnhancedVLCService) GetCurrentClock() *vlc.VectorClock {
	return evs.minerVLC.GetCurrentClock()
}

// IncrementMinerClock increments the miner node VLC for backward compatibility
func (evs *EnhancedVLCService) IncrementMinerClock() *vlc.VectorClock {
	return evs.minerVLC.IncrementMinerClock()
}

// UpdateClock updates the miner node VLC for backward compatibility
func (evs *EnhancedVLCService) UpdateClock(receivedClock *vlc.VectorClock) {
	evs.minerVLC.UpdateClock(receivedClock)
}

// ===== New dual-layer VLC methods =====

// GetMinerVLC returns the current miner node VLC
func (evs *EnhancedVLCService) GetMinerVLC() *vlc.VectorClock {
	return evs.minerVLC.GetCurrentClock()
}

// GetUserVLC returns the VLC for a specific user wallet
func (evs *EnhancedVLCService) GetUserVLC(userWallet string) *vlc.VectorClock {
	return evs.getUserVLC(userWallet).Copy()
}

// GetAllUserVLCs returns all user VLCs
func (evs *EnhancedVLCService) GetAllUserVLCs() map[string]*vlc.VectorClock {
	evs.userMux.RLock()
	defer evs.userMux.RUnlock()

	result := make(map[string]*vlc.VectorClock)
	for wallet, vlcClock := range evs.userVLCs {
		result[wallet] = vlcClock.Copy()
	}
	return result
}

// GetVLCState returns the state of both miner and user VLCs
func (evs *EnhancedVLCService) GetVLCState() map[string]interface{} {
	return map[string]interface{}{
		"miner_vlc": evs.minerVLC.GetClockState(),
		"user_vlcs": evs.GetAllUserVLCs(),
		"timestamp": time.Now(),
	}
}
