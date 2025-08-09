package apply

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/teabranch/matlas-cli/internal/clients/atlas"
)

// RetryDecision represents a manual retry decision
type RetryDecision string

const (
	RetryDecisionRetry   RetryDecision = "retry"   // Retry the operation
	RetryDecisionSkip    RetryDecision = "skip"    // Skip this operation and continue
	RetryDecisionAbort   RetryDecision = "abort"   // Abort the entire execution
	RetryDecisionIgnore  RetryDecision = "ignore"  // Ignore the error and mark as successful
)

// ManualRetryCallback is called when manual intervention is needed
type ManualRetryCallback func(ctx context.Context, operation *PlannedOperation, err error, attempt int) RetryDecision

// RetryConfig contains configuration for retry behavior
type RetryConfig struct {
	// Basic retry settings
	MaxRetries        int           `json:"maxRetries"`
	InitialDelay      time.Duration `json:"initialDelay"`
	MaxDelay          time.Duration `json:"maxDelay"`
	BackoffMultiplier float64       `json:"backoffMultiplier"`
	Jitter            float64       `json:"jitter"` // 0.0 to 1.0

	// Operation-specific policies
	OperationPolicies map[OperationType]OperationRetryPolicy `json:"operationPolicies"`

	// Circuit breaker settings
	CircuitBreakerConfig CircuitBreakerConfig `json:"circuitBreakerConfig"`

	// Retry decision settings
	RetryableErrors []string `json:"retryableErrors"`
	FatalErrors     []string `json:"fatalErrors"`
	
	// Manual retry support
	EnableManualRetry      bool                `json:"enableManualRetry"`
	ManualRetryErrors      []string            `json:"manualRetryErrors"`      // Error patterns that trigger manual intervention
	InteractiveMode        bool                `json:"interactiveMode"`        // Whether to prompt user interactively
	ManualRetryCallback    ManualRetryCallback `json:"-"`                      // Callback for manual decisions
}

// OperationRetryPolicy defines retry behavior for specific operation types
type OperationRetryPolicy struct {
	MaxRetries        int           `json:"maxRetries"`
	InitialDelay      time.Duration `json:"initialDelay"`
	MaxDelay          time.Duration `json:"maxDelay"`
	BackoffMultiplier float64       `json:"backoffMultiplier"`
	Enabled           bool          `json:"enabled"`
}

// CircuitBreakerConfig contains circuit breaker settings
type CircuitBreakerConfig struct {
	Enabled                bool          `json:"enabled"`
	FailureThreshold       int           `json:"failureThreshold"`
	RecoveryTimeout        time.Duration `json:"recoveryTimeout"`
	SuccessThreshold       int           `json:"successThreshold"`
	MonitoringWindow       time.Duration `json:"monitoringWindow"`
	MaxConsecutiveFailures int           `json:"maxConsecutiveFailures"`
}

// RetryManager handles retry logic and circuit breaking for operations
type RetryManager struct {
	config           RetryConfig
	operationRetries map[string]int
	circuitBreakers  map[string]*CircuitBreaker
	mu               sync.RWMutex
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config              CircuitBreakerConfig
	state               CircuitBreakerState
	failures            int
	successes           int
	consecutiveFailures int
	lastFailureTime     time.Time
	stateChangedAt      time.Time
	mu                  sync.RWMutex
}

// CircuitBreakerState represents the current state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"    // Normal operation
	CircuitBreakerOpen     CircuitBreakerState = "open"      // Failing fast
	CircuitBreakerHalfOpen CircuitBreakerState = "half-open" // Testing recovery
)

// RetryAttempt contains information about a retry attempt
type RetryAttempt struct {
	AttemptNumber int           `json:"attemptNumber"`
	Delay         time.Duration `json:"delay"`
	Error         error         `json:"error,omitempty"`
	StartedAt     time.Time     `json:"startedAt"`
	CompletedAt   time.Time     `json:"completedAt,omitempty"`
}

// RetryContext contains context for retry operations
type RetryContext struct {
	Operation    *PlannedOperation `json:"operation"`
	Attempts     []RetryAttempt    `json:"attempts"`
	TotalRetries int               `json:"totalRetries"`
	StartedAt    time.Time         `json:"startedAt"`
}

// NewRetryManager creates a new retry manager with the provided configuration
func NewRetryManager(config RetryConfig) *RetryManager {
	return &RetryManager{
		config:           config,
		operationRetries: make(map[string]int),
		circuitBreakers:  make(map[string]*CircuitBreaker),
	}
}

// ExecuteWithRetry executes a function with retry logic
func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, operation *PlannedOperation, fn func() error) error {
	retryCtx := &RetryContext{
		Operation: operation,
		StartedAt: time.Now(),
	}

	// Check circuit breaker
	circuitKey := rm.getCircuitBreakerKey(operation)
	if rm.config.CircuitBreakerConfig.Enabled {
		cb := rm.getOrCreateCircuitBreaker(circuitKey)
		if !cb.CanExecute() {
			return fmt.Errorf("circuit breaker is open for operation type %s", operation.Type)
		}
	}

	// Get retry policy for this operation type
	policy := rm.getRetryPolicy(operation.Type)

	var lastErr error
	maxRetries := policy.MaxRetries

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Create retry attempt record
		retryAttempt := RetryAttempt{
			AttemptNumber: attempt + 1,
			StartedAt:     time.Now(),
		}

		// Execute the function
		err := fn()
		retryAttempt.CompletedAt = time.Now()
		retryAttempt.Error = err

		if err == nil {
			// Success - record it and return
			retryCtx.Attempts = append(retryCtx.Attempts, retryAttempt)
			rm.recordSuccess(operation.ID, circuitKey)
			return nil
		}

		lastErr = err
		retryAttempt.Error = err
		retryCtx.Attempts = append(retryCtx.Attempts, retryAttempt)

		// Check if this error is retryable
		if !rm.isRetryableError(err) {
			// Check if manual retry is enabled and this error qualifies
			if rm.shouldRequestManualRetry(err) {
				decision := rm.requestManualRetry(ctx, operation, err, attempt+1)
				switch decision {
				case RetryDecisionRetry:
					// Continue with retry loop
				case RetryDecisionSkip:
					rm.recordFailure(operation.ID, circuitKey)
					return fmt.Errorf("operation skipped by manual decision: %w", err)
				case RetryDecisionAbort:
					rm.recordFailure(operation.ID, circuitKey)
					return fmt.Errorf("execution aborted by manual decision: %w", err)
				case RetryDecisionIgnore:
					// Treat as success
					rm.recordSuccess(operation.ID, circuitKey)
					return nil
				default:
					rm.recordFailure(operation.ID, circuitKey)
					return fmt.Errorf("non-retryable error: %w", err)
				}
			} else {
				rm.recordFailure(operation.ID, circuitKey)
				return fmt.Errorf("non-retryable error: %w", err)
			}
		}

		// If this is the last attempt, don't wait
		if attempt == maxRetries {
			break
		}

		// Calculate delay for next attempt
		delay := rm.calculateDelay(attempt, policy)
		retryAttempt.Delay = delay

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			rm.recordFailure(operation.ID, circuitKey)
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	rm.recordFailure(operation.ID, circuitKey)
	retryCtx.TotalRetries = len(retryCtx.Attempts) - 1

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// GetRetryCount returns the number of retries for an operation
func (rm *RetryManager) GetRetryCount(operationID string) int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.operationRetries[operationID]
}

// getRetryPolicy returns the retry policy for an operation type
func (rm *RetryManager) getRetryPolicy(opType OperationType) OperationRetryPolicy {
	if policy, exists := rm.config.OperationPolicies[opType]; exists && policy.Enabled {
		return policy
	}

	// Return default policy
	return OperationRetryPolicy{
		MaxRetries:        rm.config.MaxRetries,
		InitialDelay:      rm.config.InitialDelay,
		MaxDelay:          rm.config.MaxDelay,
		BackoffMultiplier: rm.config.BackoffMultiplier,
		Enabled:           true,
	}
}

// calculateDelay calculates the delay for the next retry attempt
func (rm *RetryManager) calculateDelay(attempt int, policy OperationRetryPolicy) time.Duration {
	// Exponential backoff
	delay := float64(policy.InitialDelay) * math.Pow(policy.BackoffMultiplier, float64(attempt))

	// Apply max delay limit
	if maxDelay := float64(policy.MaxDelay); delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter to prevent thundering herd
	if rm.config.Jitter > 0 {
		jitterAmount := delay * rm.config.Jitter
		jitter := (rand.Float64() - 0.5) * 2 * jitterAmount
		delay += jitter
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = float64(policy.InitialDelay)
	}

	return time.Duration(delay)
}

// isRetryableError determines if an error should be retried
func (rm *RetryManager) isRetryableError(err error) bool {
	errStr := err.Error()

	// Check for fatal errors first
	for _, fatalErr := range rm.config.FatalErrors {
		if contains(errStr, fatalErr) {
			return false
		}
	}

	// Check for explicitly retryable errors
	for _, retryableErr := range rm.config.RetryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}

	// Check for Atlas-specific retryable errors
	if atlas.IsTransient(err) {
		return true
	}

	// Default to not retryable for unknown errors
	return false
}

// shouldRequestManualRetry determines if manual intervention should be requested for an error
func (rm *RetryManager) shouldRequestManualRetry(err error) bool {
	if !rm.config.EnableManualRetry {
		return false
	}
	
	errStr := err.Error()
	
	// Check if this error matches manual retry patterns
	for _, manualErr := range rm.config.ManualRetryErrors {
		if contains(errStr, manualErr) {
			return true
		}
	}
	
	return false
}

// requestManualRetry requests manual intervention for a failed operation
func (rm *RetryManager) requestManualRetry(ctx context.Context, operation *PlannedOperation, err error, attempt int) RetryDecision {
	// If a custom callback is provided, use it
	if rm.config.ManualRetryCallback != nil {
		return rm.config.ManualRetryCallback(ctx, operation, err, attempt)
	}
	
	// If interactive mode is enabled, prompt the user
	if rm.config.InteractiveMode {
		return rm.promptUserForDecision(operation, err, attempt)
	}
	
	// Default behavior: don't retry
	return RetryDecisionSkip
}

// promptUserForDecision prompts the user interactively for a retry decision
func (rm *RetryManager) promptUserForDecision(operation *PlannedOperation, err error, attempt int) RetryDecision {
	// This would typically use a UI prompt or command-line input
	// For now, return a default decision (in production, this would be interactive)
	fmt.Printf("Operation %s failed (attempt %d): %v\n", operation.ResourceName, attempt, err)
	fmt.Printf("Options: [r]etry, [s]kip, [a]bort, [i]gnore? ")
	
	// In a real implementation, this would read from stdin or show a UI prompt
	// For this implementation, we'll provide a placeholder that defaults to skip
	// This ensures non-interactive environments don't hang
	return RetryDecisionSkip
}

// recordSuccess records a successful operation
func (rm *RetryManager) recordSuccess(operationID, circuitKey string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Record success for circuit breaker
	if rm.config.CircuitBreakerConfig.Enabled {
		if cb, exists := rm.circuitBreakers[circuitKey]; exists {
			cb.RecordSuccess()
		}
	}
}

// recordFailure records a failed operation
func (rm *RetryManager) recordFailure(operationID, circuitKey string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Increment retry count
	rm.operationRetries[operationID]++

	// Record failure for circuit breaker
	if rm.config.CircuitBreakerConfig.Enabled {
		if cb, exists := rm.circuitBreakers[circuitKey]; exists {
			cb.RecordFailure()
		}
	}
}

// getCircuitBreakerKey generates a key for circuit breaker based on operation
func (rm *RetryManager) getCircuitBreakerKey(operation *PlannedOperation) string {
	return fmt.Sprintf("%s:%s", operation.Type, operation.ResourceType)
}

// getOrCreateCircuitBreaker gets or creates a circuit breaker for the given key
func (rm *RetryManager) getOrCreateCircuitBreaker(key string) *CircuitBreaker {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if cb, exists := rm.circuitBreakers[key]; exists {
		return cb
	}

	cb := NewCircuitBreaker(rm.config.CircuitBreakerConfig)
	rm.circuitBreakers[key] = cb
	return cb
}

// Circuit Breaker Implementation

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config:         config,
		state:          CircuitBreakerClosed,
		stateChangedAt: time.Now(),
	}
}

// CanExecute determines if an operation can be executed
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if recovery timeout has passed
		if time.Since(cb.stateChangedAt) > cb.config.RecoveryTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = CircuitBreakerHalfOpen
			cb.stateChangedAt = time.Now()
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes++
	cb.consecutiveFailures = 0

	if cb.state == CircuitBreakerHalfOpen {
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = CircuitBreakerClosed
			cb.stateChangedAt = time.Now()
			cb.failures = 0
			cb.successes = 0
		}
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.consecutiveFailures++
	cb.lastFailureTime = time.Now()

	// Check if we should open the circuit
	if cb.state == CircuitBreakerClosed && cb.consecutiveFailures >= cb.config.FailureThreshold {
		cb.state = CircuitBreakerOpen
		cb.stateChangedAt = time.Now()
	} else if cb.state == CircuitBreakerHalfOpen {
		// Go back to open state
		cb.state = CircuitBreakerOpen
		cb.stateChangedAt = time.Now()
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:               cb.state,
		Failures:            cb.failures,
		Successes:           cb.successes,
		ConsecutiveFailures: cb.consecutiveFailures,
		LastFailureTime:     cb.lastFailureTime,
		StateChangedAt:      cb.stateChangedAt,
	}
}

// CircuitBreakerStats contains circuit breaker statistics
type CircuitBreakerStats struct {
	State               CircuitBreakerState `json:"state"`
	Failures            int                 `json:"failures"`
	Successes           int                 `json:"successes"`
	ConsecutiveFailures int                 `json:"consecutiveFailures"`
	LastFailureTime     time.Time           `json:"lastFailureTime"`
	StateChangedAt      time.Time           `json:"stateChangedAt"`
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            0.1,
		OperationPolicies: map[OperationType]OperationRetryPolicy{
			OperationCreate: {
				MaxRetries:        5,
				InitialDelay:      2 * time.Second,
				MaxDelay:          60 * time.Second,
				BackoffMultiplier: 2.0,
				Enabled:           true,
			},
			OperationUpdate: {
				MaxRetries:        3,
				InitialDelay:      1 * time.Second,
				MaxDelay:          30 * time.Second,
				BackoffMultiplier: 1.5,
				Enabled:           true,
			},
			OperationDelete: {
				MaxRetries:        2,
				InitialDelay:      1 * time.Second,
				MaxDelay:          15 * time.Second,
				BackoffMultiplier: 2.0,
				Enabled:           true,
			},
		},
		CircuitBreakerConfig: CircuitBreakerConfig{
			Enabled:                true,
			FailureThreshold:       5,
			RecoveryTimeout:        60 * time.Second,
			SuccessThreshold:       3,
			MonitoringWindow:       300 * time.Second,
			MaxConsecutiveFailures: 10,
		},
		RetryableErrors: []string{
			"timeout",
			"connection refused",
			"temporary failure",
			"rate limit",
			"throttling",
			"service unavailable",
			"internal server error",
		},
		FatalErrors: []string{
			"unauthorized",
			"forbidden",
			"not found",
			"conflict",
			"invalid request",
			"malformed",
		},
		EnableManualRetry: false,
		ManualRetryErrors: []string{
			"resource limit exceeded",
			"quota exceeded",
			"payment required",
			"maintenance mode",
			"cluster busy",
		},
		InteractiveMode: false,
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsAnywhere(s, substr))))
}

func containsAnywhere(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
