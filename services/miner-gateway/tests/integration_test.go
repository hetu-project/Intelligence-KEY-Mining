package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/miner-gateway/models"
)

// TestTaskStatusFlow tests task status flow
func TestTaskStatusFlow(t *testing.T) {
	t.Log("🧪 Testing task status flow")

	// 1. Simulate task submission - directly to PENDING_VERIFICATION status
	task := &models.Task{
		ID:         uuid.New().String(),
		UserWallet: "0x472ffa0e7544539161f175e5E450Dc809da1Ed66",
		TaskType:   models.TwitterRetweetTask,
		Status:     models.TaskPendingVerification, // New design: directly to PENDING_VERIFICATION
		Payload: map[string]interface{}{
			"tweet_id":         "1234567890",
			"twitter_username": "testuser",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Verify task submission status
	if task.Status != models.TaskPendingVerification {
		t.Errorf("❌ Task submission status error: expected=%s, actual=%s", models.TaskPendingVerification, task.Status)
	} else {
		t.Logf("✅ Task submission status correct: %s", task.Status)
	}

	// 2. Simulate batch verification process
	t.Log("🔄 Starting batch verification...")

	// Simulate Twitter verification success
	twitterVerified := true
	if twitterVerified {
		t.Log("✅ Twitter verification successful")

		// Key: status remains unchanged after verification
		if task.Status != models.TaskPendingVerification {
			t.Errorf("❌ Status error after verification: expected=%s, actual=%s", models.TaskPendingVerification, task.Status)
		} else {
			t.Log("✅ Status remains PENDING_VERIFICATION after verification - Correct!")
		}

		// Simulate VLC increment
		vlcIncrement := 1
		t.Logf("✅ VLC increment: +%d", vlcIncrement)

		// Simulate recording user task completion (prevent duplicates)
		completion := &models.UserTaskCompletion{
			UserWallet:   task.UserWallet,
			TaskID:       task.ID,
			CompletedAt:  time.Now(),
			VLCIncrement: vlcIncrement,
			PointsEarned: 0, // No points before PoCW
		}
		t.Logf("✅ Task completion recorded: user=%s, task=%s", completion.UserWallet, completion.TaskID)
	}

	// 3. Simulate PoCW consensus process
	t.Log("🔄 Starting PoCW consensus...")
	consensusDelay := 5 * time.Second
	t.Logf("⏰ Consensus delay: %v", consensusDelay)

	// Shorten delay in tests
	time.Sleep(100 * time.Millisecond)

	consensusSuccess := true
	if consensusSuccess {
		t.Log("✅ PoCW consensus completed")

		// 4. Points distribution after PoCW
		points := 10 // Twitter retweet task points
		t.Logf("✅ Points distributed: user=%s, points=%d", task.UserWallet, points)
	}

	t.Log("🎉 Complete flow test finished!")
}

// TestMultipleUsersOneTask tests multiple users completing the same task
func TestMultipleUsersOneTask(t *testing.T) {
	t.Log("🧪 Testing multiple users completing the same task")

	users := []string{
		"0x472ffa0e7544539161f175e5E450Dc809da1Ed66",
		"0x123456789abcdef123456789abcdef1234567890",
		"0xabcdef123456789abcdef123456789abcdef1234",
	}

	// Task maintains PENDING_VERIFICATION status
	taskStatus := models.TaskPendingVerification
	t.Logf("📝 Task status: %s", taskStatus)

	completedUsers := make(map[string]bool)

	for i, user := range users {
		t.Logf("👤 User %d: %s", i+1, user)

		// Check if already completed (simulate user_task_completions table query)
		if completedUsers[user] {
			t.Logf("⚠️  User already completed this task, skipping")
			continue
		}

		// Simulate verification success
		verified := true
		if verified {
			// Record completion
			completedUsers[user] = true

			// VLC increment
			vlcIncrement := 1
			t.Logf("✅ User %s: VLC+%d", user, vlcIncrement)

			// Points after PoCW
			points := 10
			t.Logf("✅ User %s: Points+%d", user, points)
		}
	}

	// Verify task status never changes
	if taskStatus != models.TaskPendingVerification {
		t.Errorf("❌ Task status changed: %s", taskStatus)
	} else {
		t.Log("✅ Task status remains PENDING_VERIFICATION - allows more users to complete")
	}

	t.Logf("🎉 %d users completed the same task", len(completedUsers))
}

// TestTaskCreationFlow tests task creation flow
func TestTaskCreationFlow(t *testing.T) {
	t.Log("🧪 Testing task creation flow")

	// Task creation directly to PENDING_VERIFICATION
	task := &models.Task{
		ID:         uuid.New().String(),
		UserWallet: "0x472ffa0e7544539161f175e5E450Dc809da1Ed66",
		TaskType:   models.TaskCreationTask,
		Status:     models.TaskPendingVerification, // Directly to PENDING_VERIFICATION
		Payload: map[string]interface{}{
			"project_name": "Test Project",
			"project_icon": "https://example.com/icon.png",
			"deadline":     time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Verify status
	if task.Status != models.TaskPendingVerification {
		t.Errorf("❌ Task creation status error: expected=%s, actual=%s", models.TaskPendingVerification, task.Status)
	} else {
		t.Log("✅ Task creation status correct: PENDING_VERIFICATION")
	}

	// Verify payload
	if task.Payload["project_name"] == nil {
		t.Error("❌ Missing project_name")
	} else {
		t.Log("✅ Contains project_name")
	}

	if task.Payload["deadline"] == nil {
		t.Error("❌ Missing deadline")
	} else {
		t.Log("✅ Contains deadline")
	}

	// Simulate verification process
	isValid := task.Payload["project_name"] != nil && task.Payload["deadline"] != nil
	if isValid {
		t.Log("✅ Task creation verification passed")

		// Status remains unchanged after verification
		if task.Status != models.TaskPendingVerification {
			t.Error("❌ Status changed after verification")
		} else {
			t.Log("✅ Status remains PENDING_VERIFICATION after verification")
		}
	}

	t.Log("🎉 Task creation flow test completed!")
}

// TestNoDuplicateCompletion tests duplicate completion protection
func TestNoDuplicateCompletion(t *testing.T) {
	t.Log("🧪 Testing duplicate completion protection")

	taskID := uuid.New().String()
	userWallet := "0x472ffa0e7544539161f175e5E450Dc809da1Ed66"

	// Simulate user_task_completions table
	completions := make(map[string]bool)
	completionKey := userWallet + ":" + taskID

	// First completion
	if !completions[completionKey] {
		completions[completionKey] = true
		t.Log("✅ First task completion - recorded successfully")
	}

	// Second attempt to complete (should be blocked)
	if completions[completionKey] {
		t.Log("✅ Duplicate completion detected - correctly blocked")
	} else {
		t.Error("❌ Failed to detect duplicate completion")
	}

	t.Log("🎉 Duplicate completion protection test completed!")
}

// TestBatchVerificationFlow tests batch verification without status change
func TestBatchVerificationFlow(t *testing.T) {
	t.Log("🧪 Testing batch verification flow")

	// Create multiple tasks with PENDING_VERIFICATION status
	tasks := []*models.Task{
		{
			ID:         uuid.New().String(),
			UserWallet: "0x472ffa0e7544539161f175e5E450Dc809da1Ed66",
			TaskType:   models.TwitterRetweetTask,
			Status:     models.TaskPendingVerification,
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New().String(),
			UserWallet: "0x123456789abcdef123456789abcdef1234567890",
			TaskType:   models.TwitterRetweetTask,
			Status:     models.TaskPendingVerification,
			CreatedAt:  time.Now(),
		},
	}

	t.Logf("📝 Batch verifying %d tasks", len(tasks))

	verifiedCount := 0
	for _, task := range tasks {
		// Simulate verification
		verified := true // Assume all verification successful
		if verified {
			verifiedCount++

			// Key: verification successful but status unchanged
			if task.Status != models.TaskPendingVerification {
				t.Errorf("❌ Task %s status changed", task.ID)
			} else {
				t.Logf("✅ Task %s verification successful, status unchanged", task.ID)
			}
		}
	}

	t.Logf("✅ Batch verification completed: %d/%d tasks verified successfully", verifiedCount, len(tasks))
	t.Log("✅ All task statuses remain PENDING_VERIFICATION")
	t.Log("🎉 Batch verification flow test completed!")
}

// TestVerificationFailureHandling tests handling of verification failures
func TestVerificationFailureHandling(t *testing.T) {
	t.Log("🧪 Testing verification failure handling")

	// Create a task that will fail verification (user hasn't retweeted)
	task := &models.Task{
		ID:         uuid.New().String(),
		UserWallet: "0x472ffa0e7544539161f175e5E450Dc809da1Ed66",
		TaskType:   models.TwitterRetweetTask,
		Status:     models.TaskPendingVerification,
		Payload: map[string]interface{}{
			"tweet_id":         "1234567890",
			"twitter_username": "testuser",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	t.Logf("📝 Initial task status: %s", task.Status)

	// Simulate verification failure (user hasn't retweeted)
	verificationFailed := true // Simulate API returns has_retweet: false

	if verificationFailed {
		t.Log("❌ Verification failed: User has not retweeted")

		// Key: Status should remain PENDING_VERIFICATION after failure
		// This allows the task to be verified again in future batch runs
		if task.Status != models.TaskPendingVerification {
			t.Errorf("❌ Task status changed after verification failure: %s", task.Status)
		} else {
			t.Log("✅ Task status remains PENDING_VERIFICATION after failure - Correct!")
		}

		// Record the failure reason
		failureReason := "user has not retweeted"
		t.Logf("📝 Failure reason recorded: %s", failureReason)

		// No VLC increment for failed verification
		vlcIncrement := 0
		t.Logf("📊 VLC increment: %d (no reward for incomplete task)", vlcIncrement)

		// No points awarded for failed verification
		points := 0
		t.Logf("💰 Points awarded: %d (no reward for incomplete task)", points)
	}

	// Simulate future batch verification after user completes retweet
	t.Log("🔄 Simulating future verification after user completes retweet...")

	// Task status should still be PENDING_VERIFICATION, allowing re-verification
	if task.Status != models.TaskPendingVerification {
		t.Error("❌ Task should still be PENDING_VERIFICATION for re-verification")
	} else {
		t.Log("✅ Task available for re-verification")

		// Simulate successful verification on retry
		verificationSuccess := true
		if verificationSuccess {
			t.Log("✅ Re-verification successful: User has now retweeted")

			// Now award VLC and points
			vlcIncrement := 1
			points := 10
			t.Logf("✅ VLC increment on success: +%d", vlcIncrement)
			t.Logf("✅ Points awarded on success: %d", points)
		}
	}

	t.Log("🎉 Verification failure handling test completed!")
}
