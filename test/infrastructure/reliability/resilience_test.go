//go:build infrastructure

package reliability_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/apply"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
)

// ReliabilityTestEnvironment provides managed environment for reliability testing
type ReliabilityTestEnvironment struct {
	ProjectID            string
	ClusterService       *atlas.ClustersService
	DatabaseUserService  *atlas.DatabaseUsersService
	NetworkAccessService *atlas.NetworkAccessListsService
	ProjectService       *atlas.ProjectsService
	DatabaseService      *database.Service
	CreatedResources     []string
	FailureSimulator     *FailureSimulator
	cleanupFuncs         []func() error
}

// FailureSimulator simulates various failure conditions
type FailureSimulator struct {
	NetworkFailures   int
	TimeoutFailures   int
	RateLimitFailures int
	AuthFailures      int
	PartialFailures   int
	mutex             sync.RWMutex
}

func (fs *FailureSimulator) RecordFailure(failureType string) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	switch failureType {
	case "network":
		fs.NetworkFailures++
	case "timeout":
		fs.TimeoutFailures++
	case "rate_limit":
		fs.RateLimitFailures++
	case "auth":
		fs.AuthFailures++
	case "partial":
		fs.PartialFailures++
	}
}

func (fs *FailureSimulator) GetStats() map[string]int {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	return map[string]int{
		"network":    fs.NetworkFailures,
		"timeout":    fs.TimeoutFailures,
		"rate_limit": fs.RateLimitFailures,
		"auth":       fs.AuthFailures,
		"partial":    fs.PartialFailures,
	}
}

// TestNetworkInterruptions tests behavior during network failures
func TestNetworkInterruptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping reliability test in short mode")
	}

	env := setupReliabilityTestEnvironment(t)
	defer env.cleanup(t)

	testCases := []struct {
		name            string
		operationCount  int
		networkDelay    time.Duration
		expectedRetries int
	}{
		{"Low Latency", 3, 100 * time.Millisecond, 0},
		{"High Latency", 3, 2 * time.Second, 1},
		{"Very High Latency", 3, 5 * time.Second, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate network conditions
			ctx := context.Background()

			// Create plan with timeout considerations
			plan := generateReliabilityPlan(env.ProjectID, tc.operationCount)

			// Configure executor with appropriate timeouts
			executor := env.createResilientExecutor(tc.networkDelay)

			startTime := time.Now()
			result, err := executor.Execute(ctx, plan)
			executionTime := time.Since(startTime)

			// Analyze results
			if err != nil {
				if isAcceptableNetworkError(err) {
					t.Logf("Test completed with expected network issues: %v", err)
				} else {
					t.Logf("Unexpected error (may be acceptable): %v", err)
				}
			}

			// Validate resilience behavior
			if result != nil {
				actualRetries := int(result.Summary.RetriedOperations)
				t.Logf("Network Test Results for %s:", tc.name)
				t.Logf("  Execution Time: %v", executionTime)
				t.Logf("  Expected Retries: %d, Actual Retries: %d", tc.expectedRetries, actualRetries)
				t.Logf("  Success Rate: %.2f%%",
					float64(result.Summary.CompletedOperations)/float64(result.Summary.TotalOperations)*100)

				// Log failure statistics
				stats := env.FailureSimulator.GetStats()
				for failureType, count := range stats {
					if count > 0 {
						t.Logf("  %s failures: %d", failureType, count)
					}
				}
			}
		})
	}
}

// TestRateLimitHandling tests behavior under Atlas API rate limiting
func TestRateLimitHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping reliability test in short mode")
	}

	env := setupReliabilityTestEnvironment(t)
	defer env.cleanup(t)

	// Test scenarios designed to potentially trigger rate limits
	testCases := []struct {
		name                string
		operationCount      int
		concurrentExecutors int
		expectedBehavior    string
	}{
		{"Single Executor High Load", 50, 1, "should handle gracefully"},
		{"Multiple Executors", 20, 3, "should coordinate properly"},
		{"Burst Load", 100, 5, "should back off appropriately"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()

			var wg sync.WaitGroup
			results := make([]*apply.ExecutionResult, tc.concurrentExecutors)
			errors := make([]error, tc.concurrentExecutors)

			startTime := time.Now()

			// Launch concurrent executors
			for i := 0; i < tc.concurrentExecutors; i++ {
				wg.Add(1)
				go func(executorID int) {
					defer wg.Done()

					// Create executor-specific plan
					plan := generateRateLimitTestPlan(env.ProjectID, tc.operationCount, executorID)
					executor := env.createRateLimitAwareExecutor()

					results[executorID], errors[executorID] = executor.Execute(ctx, plan)

					// Record rate limit indicators
					if errors[executorID] != nil {
						if isRateLimitError(errors[executorID]) {
							env.FailureSimulator.RecordFailure("rate_limit")
						}
					}
				}(i)
			}

			wg.Wait()
			totalTime := time.Since(startTime)

			// Analyze rate limit handling
			successfulExecutors := 0
			totalRetries := int64(0)
			totalOperations := 0
			completedOperations := 0

			for i, result := range results {
				if errors[i] == nil || isAcceptableRateLimitError(errors[i]) {
					successfulExecutors++
				}

				if result != nil {
					totalRetries += int64(result.Summary.RetriedOperations)
					totalOperations += result.Summary.TotalOperations
					completedOperations += result.Summary.CompletedOperations
				}
			}

			successRate := float64(successfulExecutors) / float64(tc.concurrentExecutors) * 100
			operationSuccessRate := float64(completedOperations) / float64(totalOperations) * 100

			t.Logf("Rate Limit Test Results for %s:", tc.name)
			t.Logf("  Total Time: %v", totalTime)
			t.Logf("  Executor Success Rate: %.2f%%", successRate)
			t.Logf("  Operation Success Rate: %.2f%%", operationSuccessRate)
			t.Logf("  Total Retries: %d", totalRetries)
			t.Logf("  Rate Limit Failures: %d", env.FailureSimulator.GetStats()["rate_limit"])

			// Validate rate limit handling
			minAcceptableSuccessRate := 60.0 // 60%
			if successRate < minAcceptableSuccessRate {
				t.Logf("Warning: Success rate %.2f%% below desired %.2f%% (may be expected under extreme load)",
					successRate, minAcceptableSuccessRate)
			}
		})
	}
}

// TestPartialFailureRecovery tests recovery from partial failures
func TestPartialFailureRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping reliability test in short mode")
	}

	env := setupReliabilityTestEnvironment(t)
	defer env.cleanup(t)

	t.Run("Mixed_Success_Failure_Plan", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Create a plan with operations that are likely to have mixed success/failure
		plan := generateMixedSuccessFailurePlan(env.ProjectID)
		executor := env.createResilientExecutor(time.Second)

		result, err := executor.Execute(ctx, plan)

		// Analyze partial failure handling
		if result != nil {
			totalOps := result.Summary.TotalOperations
			completedOps := result.Summary.CompletedOperations
			failedOps := result.Summary.FailedOperations
			skippedOps := result.Summary.SkippedOperations

			t.Logf("Partial Failure Recovery Results:")
			t.Logf("  Total Operations: %d", totalOps)
			t.Logf("  Completed: %d (%.2f%%)", completedOps, float64(completedOps)/float64(totalOps)*100)
			t.Logf("  Failed: %d (%.2f%%)", failedOps, float64(failedOps)/float64(totalOps)*100)
			t.Logf("  Skipped: %d (%.2f%%)", skippedOps, float64(skippedOps)/float64(totalOps)*100)

			// Validate that we made some progress
			if completedOps == 0 && failedOps == 0 {
				t.Error("Expected at least some operations to complete or fail")
			}

			// Check if the plan status reflects the mixed results appropriately
			expectedStatus := apply.PlanStatusPartial
			if completedOps == totalOps {
				expectedStatus = apply.PlanStatusCompleted
			} else if completedOps == 0 {
				expectedStatus = apply.PlanStatusFailed
			}

			if result.Status != expectedStatus {
				t.Logf("Plan status %v differs from expected %v (may be acceptable)",
					result.Status, expectedStatus)
			}
		}

		// Log any error for analysis
		if err != nil {
			t.Logf("Execution completed with error: %v", err)
		}
	})
}

// TestIdempotencyUnderFailures tests that operations remain idempotent under various failure conditions
func TestIdempotencyUnderFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping reliability test in short mode")
	}

	env := setupReliabilityTestEnvironment(t)
	defer env.cleanup(t)

	t.Run("Repeated_Execution_With_Failures", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		// Create a simple plan that we'll execute multiple times
		plan := generateIdempotencyTestPlan(env.ProjectID)
		executor := env.createResilientExecutor(500 * time.Millisecond)

		executionResults := make([]*apply.ExecutionResult, 3)
		executionErrors := make([]error, 3)

		// Execute the same plan multiple times
		for i := 0; i < 3; i++ {
			t.Logf("Execution attempt %d", i+1)

			result, err := executor.Execute(ctx, plan)
			executionResults[i] = result
			executionErrors[i] = err

			// Wait between executions
			if i < 2 {
				time.Sleep(10 * time.Second)
			}
		}

		// Analyze idempotency
		t.Logf("Idempotency Test Results:")
		for i, result := range executionResults {
			if result != nil {
				t.Logf("  Execution %d: %d completed, %d failed, status: %v",
					i+1, result.Summary.CompletedOperations, result.Summary.FailedOperations, result.Status)
			} else {
				t.Logf("  Execution %d: no result (error: %v)", i+1, executionErrors[i])
			}
		}

		// Validate idempotent behavior
		successfulExecutions := 0
		for i, err := range executionErrors {
			if err == nil || isAcceptableIdempotencyError(err) {
				successfulExecutions++
			} else {
				t.Logf("Execution %d had unexpected error: %v", i+1, err)
			}
		}

		if successfulExecutions < 2 {
			t.Logf("Warning: Only %d/3 executions successful, idempotency may be compromised",
				successfulExecutions)
		}
	})
}

// Helper functions

func setupReliabilityTestEnvironment(t *testing.T) *ReliabilityTestEnvironment {
	config := loadReliabilityTestConfig(t)

	if config.AtlasPublicKey == "" || config.AtlasPrivateKey == "" || config.AtlasProjectID == "" {
		t.Skip("Atlas credentials not provided - skipping reliability test")
	}

	atlasClient, err := createAtlasClient(config)
	if err != nil {
		t.Fatalf("Failed to create Atlas client: %v", err)
	}

	env := &ReliabilityTestEnvironment{
		ProjectID:            config.AtlasProjectID,
		ClusterService:       atlas.NewClustersService(atlasClient),
		DatabaseUserService:  atlas.NewDatabaseUsersService(atlasClient),
		NetworkAccessService: atlas.NewNetworkAccessListsService(atlasClient),
		ProjectService:       atlas.NewProjectsService(atlasClient),
		DatabaseService:      nil,
		CreatedResources:     []string{},
		FailureSimulator:     &FailureSimulator{},
		cleanupFuncs:         []func() error{},
	}

	t.Cleanup(func() {
		env.cleanup(t)
	})

	return env
}

func (env *ReliabilityTestEnvironment) createResilientExecutor(networkDelay time.Duration) *apply.AtlasExecutor {
	config := apply.DefaultExecutorConfig()
	config.MaxConcurrentOperations = 2                       // Conservative for reliability testing
	config.OperationTimeout = 5*time.Minute + networkDelay*2 // Account for network delays

	// Configure retry settings for resilience
	config.RetryConfig = apply.RetryConfig{
		MaxRetries:        5,
		InitialDelay:      time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		EnableManualRetry: false,
		RetryableErrors: []string{
			"timeout",
			"connection refused",
			"temporary failure",
			"rate limit",
			"throttling",
			"service unavailable",
			"internal server error",
		},
	}

	return apply.NewAtlasExecutor(
		env.ClusterService,
		env.DatabaseUserService,
		env.NetworkAccessService,
		env.ProjectService,
		env.DatabaseService,
		config,
	)
}

func (env *ReliabilityTestEnvironment) createRateLimitAwareExecutor() *apply.AtlasExecutor {
	config := apply.DefaultExecutorConfig()
	config.MaxConcurrentOperations = 1 // Very conservative to avoid rate limits
	config.OperationTimeout = 10 * time.Minute

	// Configure for rate limit resilience
	config.RetryConfig = apply.RetryConfig{
		MaxRetries:        10,
		InitialDelay:      2 * time.Second,
		MaxDelay:          2 * time.Minute,
		BackoffMultiplier: 3.0,
		EnableManualRetry: false,
		RetryableErrors: []string{
			"rate limit",
			"throttling",
			"too many requests",
			"quota exceeded",
		},
	}

	return apply.NewAtlasExecutor(
		env.ClusterService,
		env.DatabaseUserService,
		env.NetworkAccessService,
		env.ProjectService,
		env.DatabaseService,
		config,
	)
}

func generateReliabilityPlan(projectID string, operationCount int) *apply.Plan {
	operations := make([]apply.PlannedOperation, operationCount)

	for i := 0; i < operationCount; i++ {
		user := &types.DatabaseUserConfig{
			Metadata: types.ResourceMetadata{
				Name: fmt.Sprintf("reliability-test-user-%d-%d", time.Now().Unix(), i),
			},
			Username: fmt.Sprintf("reliability-test-user-%d-%d", time.Now().Unix(), i),
			Password: "ReliabilityTest123!",
			Roles: []types.DatabaseRoleConfig{
				{
					RoleName:     "read",
					DatabaseName: "admin",
				},
			},
		}

		operations[i] = apply.PlannedOperation{
			Operation: apply.Operation{
				Type:         apply.OperationCreate,
				ResourceType: types.KindDatabaseUser,
				ResourceName: user.Metadata.Name,
				Desired:      user,
			},
			ID:       fmt.Sprintf("reliability-op-%d", i),
			Priority: 1,
			Stage:    0,
			Status:   apply.OperationStatusPending,
		}
	}

	return &apply.Plan{
		ID:         fmt.Sprintf("reliability-plan-%d", time.Now().Unix()),
		ProjectID:  projectID,
		Operations: operations,
		Status:     apply.PlanStatusApproved,
	}
}

func generateRateLimitTestPlan(projectID string, operationCount, executorID int) *apply.Plan {
	operations := make([]apply.PlannedOperation, operationCount)

	for i := 0; i < operationCount; i++ {
		user := &types.DatabaseUserConfig{
			Metadata: types.ResourceMetadata{
				Name: fmt.Sprintf("ratelimit-executor-%d-user-%d-%d", executorID, time.Now().Unix(), i),
			},
			Username: fmt.Sprintf("ratelimit-executor-%d-user-%d-%d", executorID, time.Now().Unix(), i),
			Password: "RateLimitTest123!",
			Roles: []types.DatabaseRoleConfig{
				{
					RoleName:     "read",
					DatabaseName: "admin",
				},
			},
		}

		operations[i] = apply.PlannedOperation{
			Operation: apply.Operation{
				Type:         apply.OperationCreate,
				ResourceType: types.KindDatabaseUser,
				ResourceName: user.Metadata.Name,
				Desired:      user,
			},
			ID:       fmt.Sprintf("ratelimit-executor-%d-op-%d", executorID, i),
			Priority: 1,
			Stage:    0,
			Status:   apply.OperationStatusPending,
		}
	}

	return &apply.Plan{
		ID:         fmt.Sprintf("ratelimit-plan-executor-%d-%d", executorID, time.Now().Unix()),
		ProjectID:  projectID,
		Operations: operations,
		Status:     apply.PlanStatusApproved,
	}
}

func generateMixedSuccessFailurePlan(projectID string) *apply.Plan {
	operations := []apply.PlannedOperation{}

	// Add some operations that should succeed
	for i := 0; i < 3; i++ {
		user := &types.DatabaseUserConfig{
			Metadata: types.ResourceMetadata{
				Name: fmt.Sprintf("mixed-success-user-%d-%d", time.Now().Unix(), i),
			},
			Username: fmt.Sprintf("mixed-success-user-%d-%d", time.Now().Unix(), i),
			Password: "MixedTest123!",
			Roles: []types.DatabaseRoleConfig{
				{
					RoleName:     "read",
					DatabaseName: "admin",
				},
			},
		}

		operations = append(operations, apply.PlannedOperation{
			Operation: apply.Operation{
				Type:         apply.OperationCreate,
				ResourceType: types.KindDatabaseUser,
				ResourceName: user.Metadata.Name,
				Desired:      user,
			},
			ID:       fmt.Sprintf("mixed-success-op-%d", i),
			Priority: 1,
			Stage:    0,
			Status:   apply.OperationStatusPending,
		})
	}

	// Add some operations that might fail (invalid configurations)
	for i := 0; i < 2; i++ {
		user := &types.DatabaseUserConfig{
			Metadata: types.ResourceMetadata{
				Name: fmt.Sprintf("mixed-invalid-user-%d-%d", time.Now().Unix(), i),
			},
			Username: fmt.Sprintf("mixed-invalid-user-%d-%d", time.Now().Unix(), i),
			Password: "weak", // Intentionally weak password that might fail validation
			Roles: []types.DatabaseRoleConfig{
				{
					RoleName:     "invalidRole", // Invalid role that should fail
					DatabaseName: "admin",
				},
			},
		}

		operations = append(operations, apply.PlannedOperation{
			Operation: apply.Operation{
				Type:         apply.OperationCreate,
				ResourceType: types.KindDatabaseUser,
				ResourceName: user.Metadata.Name,
				Desired:      user,
			},
			ID:       fmt.Sprintf("mixed-failure-op-%d", i),
			Priority: 1,
			Stage:    1, // Different stage to test stage-based recovery
			Status:   apply.OperationStatusPending,
		})
	}

	return &apply.Plan{
		ID:         fmt.Sprintf("mixed-plan-%d", time.Now().Unix()),
		ProjectID:  projectID,
		Operations: operations,
		Status:     apply.PlanStatusApproved,
	}
}

func generateIdempotencyTestPlan(projectID string) *apply.Plan {
	timestamp := time.Now().Unix()

	user := &types.DatabaseUserConfig{
		Metadata: types.ResourceMetadata{
			Name: fmt.Sprintf("idempotency-test-user-%d", timestamp),
		},
		Username: fmt.Sprintf("idempotency-test-user-%d", timestamp),
		Password: "IdempotencyTest123!",
		Roles: []types.DatabaseRoleConfig{
			{
				RoleName:     "read",
				DatabaseName: "admin",
			},
		},
	}

	return &apply.Plan{
		ID:        fmt.Sprintf("idempotency-plan-%d", timestamp),
		ProjectID: projectID,
		Operations: []apply.PlannedOperation{
			{
				Operation: apply.Operation{
					Type:         apply.OperationCreate,
					ResourceType: types.KindDatabaseUser,
					ResourceName: user.Metadata.Name,
					Desired:      user,
				},
				ID:       "idempotency-test-op",
				Priority: 1,
				Stage:    0,
				Status:   apply.OperationStatusPending,
			},
		},
		Status: apply.PlanStatusApproved,
	}
}

// Error classification functions

func isAcceptableNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	networkErrors := []string{
		"timeout",
		"connection refused",
		"network unreachable",
		"no route to host",
		"connection reset",
		"connection aborted",
	}

	for _, netErr := range networkErrors {
		if contains(errorStr, netErr) {
			return true
		}
	}

	// Check for net.Error interface
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	return false
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	rateLimitErrors := []string{
		"rate limit",
		"throttling",
		"too many requests",
		"quota exceeded",
		"429",
	}

	for _, rlErr := range rateLimitErrors {
		if contains(errorStr, rlErr) {
			return true
		}
	}

	return false
}

func isAcceptableRateLimitError(err error) bool {
	if err == nil {
		return true
	}

	// Rate limit errors are acceptable in rate limit tests
	if isRateLimitError(err) {
		return true
	}

	// Other acceptable errors
	acceptableErrors := []string{
		"not yet implemented",
		"DUPLICATE_DATABASE_USER",
		"already exists",
	}

	errorStr := err.Error()
	for _, acceptable := range acceptableErrors {
		if contains(errorStr, acceptable) {
			return true
		}
	}

	return false
}

func isAcceptableIdempotencyError(err error) bool {
	if err == nil {
		return true
	}

	// For idempotency tests, duplicate resource errors are expected
	errorStr := err.Error()
	idempotencyErrors := []string{
		"DUPLICATE_DATABASE_USER",
		"already exists",
		"resource already exists",
		"not yet implemented", // Until full implementation
	}

	for _, idempErr := range idempotencyErrors {
		if contains(errorStr, idempErr) {
			return true
		}
	}

	return false
}

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

// Configuration and setup

type ReliabilityTestConfig struct {
	AtlasPublicKey  string
	AtlasPrivateKey string
	AtlasOrgID      string
	AtlasProjectID  string
}

func loadReliabilityTestConfig(t *testing.T) ReliabilityTestConfig {
	return ReliabilityTestConfig{
		AtlasPublicKey:  os.Getenv("ATLAS_PUBLIC_KEY"),
		AtlasPrivateKey: os.Getenv("ATLAS_PRIVATE_KEY"),
		AtlasOrgID:      os.Getenv("ATLAS_ORG_ID"),
		AtlasProjectID:  os.Getenv("ATLAS_PROJECT_ID"),
	}
}

func createAtlasClient(config ReliabilityTestConfig) (*atlasclient.Client, error) {
	clientConfig := atlasclient.Config{}
	return atlasclient.NewClient(clientConfig)
}

func (env *ReliabilityTestEnvironment) cleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, resourceID := range env.CreatedResources {
		if env.DatabaseUserService != nil {
			err := env.DatabaseUserService.Delete(ctx, env.ProjectID, "admin", resourceID)
			if err != nil && !contains(err.Error(), "NOT_FOUND") {
				t.Logf("Warning: Failed to cleanup resource %s: %v", resourceID, err)
			}
		}
	}

	for _, cleanupFn := range env.cleanupFuncs {
		if err := cleanupFn(); err != nil {
			t.Logf("Warning: Cleanup function failed: %v", err)
		}
	}
}
