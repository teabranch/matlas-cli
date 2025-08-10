package atlas

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockTransientError simulates Atlas API transient failures
type mockTransientError struct {
	message string
}

func (e mockTransientError) Error() string {
	return e.message
}

func TestRetry_StressTestTransientFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	tests := []struct {
		name          string
		maxAttempts   int
		backoff       time.Duration
		failureCount  int
		expectedCalls int
		shouldSucceed bool
	}{
		{
			name:          "succeed after 2 transient failures",
			maxAttempts:   5,
			backoff:       1 * time.Millisecond,
			failureCount:  2,
			expectedCalls: 3,
			shouldSucceed: true,
		},
		{
			name:          "exhaust all retries",
			maxAttempts:   3,
			backoff:       1 * time.Millisecond,
			failureCount:  5,
			expectedCalls: 3,
			shouldSucceed: false,
		},
		{
			name:          "immediate success",
			maxAttempts:   3,
			backoff:       1 * time.Millisecond,
			failureCount:  0,
			expectedCalls: 1,
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			callCount := 0

			err := retry(ctx, tt.maxAttempts, tt.backoff, func() error {
				callCount++
				if callCount <= tt.failureCount {
					return fmt.Errorf("%w: simulated transient failure #%d", ErrTransient, callCount)
				}
				return nil
			})

			if tt.shouldSucceed && err != nil {
				t.Errorf("expected success but got error: %v", err)
			}
			if !tt.shouldSucceed && err == nil {
				t.Error("expected failure but got success")
			}
			if callCount != tt.expectedCalls {
				t.Errorf("expected %d calls, got %d", tt.expectedCalls, callCount)
			}
		})
	}
}

func TestRetry_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent stress test in short mode")
	}

	const numGoroutines = 50
	const retriesPerGoroutine = 10

	ctx := context.Background()
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < retriesPerGoroutine; j++ {
				err := retry(ctx, 3, 1*time.Millisecond, func() error {
					// Simulate random transient failures
					if (id+j)%3 == 0 {
						return fmt.Errorf("%w: goroutine %d iteration %d", ErrTransient, id, j)
					}
					return nil
				})

				if err != nil && !IsTransient(err) {
					errors <- fmt.Errorf("goroutine %d: unexpected non-transient error: %v", id, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any unexpected errors
	for err := range errors {
		t.Error(err)
	}
}

func TestRetry_ContextCancellationStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping context cancellation stress test in short mode")
	}

	tests := []struct {
		name           string
		timeout        time.Duration
		operationDelay time.Duration
		expectTimeout  bool
	}{
		{
			name:           "context timeout before operation",
			timeout:        5 * time.Millisecond,
			operationDelay: 20 * time.Millisecond,
			expectTimeout:  true,
		},
		{
			name:           "operation completes before timeout",
			timeout:        50 * time.Millisecond,
			operationDelay: 5 * time.Millisecond,
			expectTimeout:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			startTime := time.Now()
			err := retry(ctx, 5, 1*time.Millisecond, func() error {
				time.Sleep(tt.operationDelay)
				return nil
			})
			elapsed := time.Since(startTime)

			if tt.expectTimeout {
				if err == nil {
					t.Error("expected context timeout error but got nil")
				}
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("expected context.DeadlineExceeded but got: %v", err)
				}
				// Should fail within reasonable time due to timeout (more tolerant for CI)
				maxAllowedTime := tt.timeout * 10 // More tolerant for CI environments
				if elapsed > maxAllowedTime {
					t.Errorf("operation took too long: %v (expected less than %v)", elapsed, maxAllowedTime)
				}
			} else {
				if err != nil {
					t.Errorf("expected success but got error: %v", err)
				}
			}
		})
	}
}

func TestRetry_BackoffTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping backoff timing test in short mode")
	}

	ctx := context.Background()
	baseBackoff := 10 * time.Millisecond
	attempts := 0

	startTime := time.Now()
	err := retry(ctx, 4, baseBackoff, func() error {
		attempts++
		if attempts < 4 {
			return fmt.Errorf("%w: attempt %d", ErrTransient, attempts)
		}
		return nil
	})
	elapsed := time.Since(startTime)

	if err != nil {
		t.Errorf("expected success but got error: %v", err)
	}
	if attempts != 4 {
		t.Errorf("expected 4 attempts, got %d", attempts)
	}

	// Expected delays: 0 + 10ms + 20ms + 40ms = 70ms minimum
	// (backoff doubles each time: 10ms, 20ms, 40ms)
	expectedMinimum := baseBackoff + 2*baseBackoff + 4*baseBackoff // 70ms
	if elapsed < expectedMinimum {
		t.Errorf("retry completed too quickly: %v (expected at least %v)", elapsed, expectedMinimum)
	}

	// Should not take dramatically longer (allow 50ms tolerance for test overhead)
	expectedMaximum := expectedMinimum + 50*time.Millisecond
	if elapsed > expectedMaximum {
		t.Errorf("retry took too long: %v (expected at most %v)", elapsed, expectedMaximum)
	}
}
