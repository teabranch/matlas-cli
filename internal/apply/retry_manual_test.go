package apply

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryManager_ManualRetry_WithCallback(t *testing.T) {
	config := DefaultRetryConfig()
	config.EnableManualRetry = true
	config.ManualRetryErrors = []string{"quota exceeded"}

	// Custom callback that decides to retry
	callbackCalled := false
	config.ManualRetryCallback = func(ctx context.Context, operation *PlannedOperation, err error, attempt int) RetryDecision {
		callbackCalled = true
		if attempt <= 2 {
			return RetryDecisionRetry
		}
		return RetryDecisionSkip
	}

	rm := NewRetryManager(config)

	operation := &PlannedOperation{
		Operation: Operation{Type: OperationCreate},
		ID:        "manual-test-op",
	}

	attemptCount := 0
	err := rm.ExecuteWithRetry(context.Background(), operation, func() error {
		attemptCount++
		return errors.New("quota exceeded")
	})

	// Should fail after callback decides to skip
	if err == nil {
		t.Fatal("Expected execution to fail")
	}

	if !callbackCalled {
		t.Error("Expected manual retry callback to be called")
	}

	// Should have attempted multiple times due to manual retry decisions
	if attemptCount < 2 {
		t.Errorf("Expected at least 2 attempts, got %d", attemptCount)
	}

	if !contains(err.Error(), "operation skipped by manual decision") {
		t.Errorf("Expected skip decision error, got: %v", err)
	}
}

func TestRetryManager_ManualRetry_AbortDecision(t *testing.T) {
	config := DefaultRetryConfig()
	config.EnableManualRetry = true
	config.ManualRetryErrors = []string{"payment required"}

	// Callback that decides to abort
	config.ManualRetryCallback = func(ctx context.Context, operation *PlannedOperation, err error, attempt int) RetryDecision {
		return RetryDecisionAbort
	}

	rm := NewRetryManager(config)

	operation := &PlannedOperation{
		Operation: Operation{Type: OperationCreate},
		ID:        "abort-test-op",
	}

	attemptCount := 0
	err := rm.ExecuteWithRetry(context.Background(), operation, func() error {
		attemptCount++
		return errors.New("payment required")
	})

	if err == nil {
		t.Fatal("Expected execution to fail")
	}

	if !contains(err.Error(), "execution aborted by manual decision") {
		t.Errorf("Expected abort decision error, got: %v", err)
	}

	// Should only attempt once before aborting
	if attemptCount != 1 {
		t.Errorf("Expected exactly 1 attempt, got %d", attemptCount)
	}
}

func TestRetryManager_ManualRetry_IgnoreDecision(t *testing.T) {
	config := DefaultRetryConfig()
	config.EnableManualRetry = true
	config.ManualRetryErrors = []string{"maintenance mode"}

	// Callback that decides to ignore the error
	config.ManualRetryCallback = func(ctx context.Context, operation *PlannedOperation, err error, attempt int) RetryDecision {
		return RetryDecisionIgnore
	}

	rm := NewRetryManager(config)

	operation := &PlannedOperation{
		Operation: Operation{Type: OperationCreate},
		ID:        "ignore-test-op",
	}

	attemptCount := 0
	err := rm.ExecuteWithRetry(context.Background(), operation, func() error {
		attemptCount++
		return errors.New("maintenance mode")
	})

	// Should succeed due to ignore decision
	if err != nil {
		t.Fatalf("Expected execution to succeed due to ignore decision, got error: %v", err)
	}

	// Should only attempt once before ignoring
	if attemptCount != 1 {
		t.Errorf("Expected exactly 1 attempt, got %d", attemptCount)
	}

	// Verify success was recorded
	if rm.GetRetryCount(operation.ID) != 0 {
		t.Errorf("Expected retry count to be 0 for ignored error, got %d", rm.GetRetryCount(operation.ID))
	}
}

func TestRetryManager_ManualRetry_DisabledByDefault(t *testing.T) {
	config := DefaultRetryConfig()
	// Manual retry is disabled by default
	rm := NewRetryManager(config)

	operation := &PlannedOperation{
		Operation: Operation{Type: OperationCreate},
		ID:        "disabled-test-op",
	}

	attemptCount := 0
	err := rm.ExecuteWithRetry(context.Background(), operation, func() error {
		attemptCount++
		return errors.New("quota exceeded") // This would trigger manual retry if enabled
	})

	// Should fail with non-retryable error since manual retry is disabled
	if err == nil {
		t.Fatal("Expected execution to fail")
	}

	if !contains(err.Error(), "non-retryable error") {
		t.Errorf("Expected non-retryable error, got: %v", err)
	}

	// Should only attempt once
	if attemptCount != 1 {
		t.Errorf("Expected exactly 1 attempt, got %d", attemptCount)
	}
}

func TestRetryManager_ManualRetry_InteractiveMode(t *testing.T) {
	config := DefaultRetryConfig()
	config.EnableManualRetry = true
	config.InteractiveMode = true
	config.ManualRetryErrors = []string{"cluster busy"}

	rm := NewRetryManager(config)

	operation := &PlannedOperation{
		Operation: Operation{Type: OperationCreate},
		ID:        "interactive-test-op",
	}

	attemptCount := 0
	err := rm.ExecuteWithRetry(context.Background(), operation, func() error {
		attemptCount++
		return errors.New("cluster busy")
	})

	// Should fail with skip decision (default for interactive mode in our implementation)
	if err == nil {
		t.Fatal("Expected execution to fail")
	}

	if !contains(err.Error(), "operation skipped by manual decision") {
		t.Errorf("Expected skip decision error, got: %v", err)
	}
}

func TestRetryManager_ManualRetry_ErrorPatternMatching(t *testing.T) {
	config := DefaultRetryConfig()
	config.EnableManualRetry = true
	config.ManualRetryErrors = []string{"quota", "limit"}
	
	// Remove "rate limit" from retryable errors so it can trigger manual retry
	config.RetryableErrors = []string{
		"timeout",
		"connection refused", 
		"temporary failure",
		"throttling",
		"service unavailable",
		"internal server error",
	}

	decisionCallCount := 0
	config.ManualRetryCallback = func(ctx context.Context, operation *PlannedOperation, err error, attempt int) RetryDecision {
		decisionCallCount++
		return RetryDecisionSkip
	}

	rm := NewRetryManager(config)

	operation := &PlannedOperation{
		Operation: Operation{Type: OperationCreate},
		ID:        "pattern-test-op",
	}

	tests := []struct {
		name          string
		errorMessage  string
		shouldTrigger bool
	}{
		{"exact match", "quota exceeded", true},
		{"partial match", "rate limit reached", true},
		{"no match", "unauthorized access", false},
		{"case sensitive", "QUOTA exceeded", false}, // Our matching is case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decisionCallCount = 0

			err := rm.ExecuteWithRetry(context.Background(), operation, func() error {
				return errors.New(tt.errorMessage)
			})

			if tt.shouldTrigger {
				if decisionCallCount == 0 {
					t.Error("Expected manual retry callback to be called")
				}
				if !contains(err.Error(), "operation skipped by manual decision") {
					t.Errorf("Expected manual decision error, got: %v", err)
				}
			} else {
				if decisionCallCount > 0 {
					t.Error("Did not expect manual retry callback to be called")
				}
				if !contains(err.Error(), "non-retryable error") {
					t.Errorf("Expected non-retryable error, got: %v", err)
				}
			}
		})
	}
}

func TestRetryManager_ManualRetry_ContextCancellation(t *testing.T) {
	config := DefaultRetryConfig()
	config.EnableManualRetry = true
	config.ManualRetryErrors = []string{"temporary failure"}

	// Callback that takes time and should be interrupted by context cancellation
	config.ManualRetryCallback = func(ctx context.Context, operation *PlannedOperation, err error, attempt int) RetryDecision {
		select {
		case <-ctx.Done():
			return RetryDecisionAbort
		case <-time.After(200 * time.Millisecond):
			return RetryDecisionRetry
		}
	}

	rm := NewRetryManager(config)

	operation := &PlannedOperation{
		Operation: Operation{Type: OperationCreate},
		ID:        "cancel-test-op",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := rm.ExecuteWithRetry(ctx, operation, func() error {
		return errors.New("temporary failure")
	})

	if err == nil {
		t.Fatal("Expected execution to fail due to context cancellation")
	}

	// Should be cancelled or aborted
	if !contains(err.Error(), "cancelled") && !contains(err.Error(), "aborted") {
		t.Errorf("Expected cancellation or abort error, got: %v", err)
	}
}

// Benchmark test for manual retry performance impact
func BenchmarkRetryManager_ManualRetryDisabled(b *testing.B) {
	config := DefaultRetryConfig()
	config.EnableManualRetry = false
	rm := NewRetryManager(config)

	operation := &PlannedOperation{
		Operation: Operation{Type: OperationCreate},
		ID:        "bench-disabled",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rm.ExecuteWithRetry(context.Background(), operation, func() error {
			return nil // Success
		})
	}
}

func BenchmarkRetryManager_ManualRetryEnabled(b *testing.B) {
	config := DefaultRetryConfig()
	config.EnableManualRetry = true
	config.ManualRetryErrors = []string{"test error"}
	rm := NewRetryManager(config)

	operation := &PlannedOperation{
		Operation: Operation{Type: OperationCreate},
		ID:        "bench-enabled",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rm.ExecuteWithRetry(context.Background(), operation, func() error {
			return nil // Success
		})
	}
}
