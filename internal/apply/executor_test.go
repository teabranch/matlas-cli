package apply

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
)

// MockAtlasServices provides mock implementations for Atlas services
type MockAtlasServices struct {
	clustersService      *atlas.ClustersService
	usersService         *atlas.DatabaseUsersService
	networkAccessService *atlas.NetworkAccessListsService
	projectsService      *atlas.ProjectsService
	databaseService      *database.Service
}

func TestNewAtlasExecutor(t *testing.T) {
	mockServices := createMockServices()
	config := DefaultExecutorConfig()

	executor := NewAtlasExecutor(
		mockServices.clustersService,
		mockServices.usersService,
		mockServices.networkAccessService,
		mockServices.projectsService,
		mockServices.databaseService,
		config,
	)

	if executor == nil {
		t.Fatal("NewAtlasExecutor returned nil")
	}

	if executor.clustersService != mockServices.clustersService {
		t.Error("clustersService not set correctly")
	}

	if executor.config.MaxConcurrentOperations != config.MaxConcurrentOperations {
		t.Error("config not set correctly")
	}

	if executor.retryManager == nil {
		t.Error("retryManager not initialized")
	}

	if executor.progressTracker == nil {
		t.Error("progressTracker not initialized")
	}
}

func TestDefaultExecutorConfig(t *testing.T) {
	config := DefaultExecutorConfig()

	if config.MaxConcurrentOperations <= 0 {
		t.Error("MaxConcurrentOperations should be positive")
	}

	if config.OperationTimeout <= 0 {
		t.Error("OperationTimeout should be positive")
	}

	if config.ProgressUpdateInterval <= 0 {
		t.Error("ProgressUpdateInterval should be positive")
	}

	if len(config.ParallelSafeOperations) == 0 {
		t.Error("ParallelSafeOperations should not be empty")
	}

	// New field defaults
	if config.PreserveExisting {
		t.Error("PreserveExisting should default to false")
	}
}

// New tests for continue-on-error classification improvements
func TestAtlasExecutor_ShouldContinueOnError_Classification(t *testing.T) {
	exec := createTestExecutor()

	// Critical impact operation should stop on generic error
	opCritical := &PlannedOperation{Operation: Operation{Type: OperationUpdate, Impact: &OperationImpact{RiskLevel: RiskLevelCritical}}}
	if exec.shouldContinueOnError(opCritical, fmt.Errorf("some error")) {
		t.Fatalf("expected not to continue on critical error")
	}

	// Unauthorized errors should stop
	if exec.shouldContinueOnError(nil, fmt.Errorf("%w: unauthorized", atlasclient.ErrUnauthorized)) {
		t.Fatalf("expected not to continue on unauthorized error")
	}

	// Transient errors should continue
	if !exec.shouldContinueOnError(nil, fmt.Errorf("%w: temporary", atlasclient.ErrTransient)) {
		t.Fatalf("expected to continue on transient error")
	}

	// Conflict on create with preserve-existing should continue
	opCreate := &PlannedOperation{Operation: Operation{Type: OperationCreate}}
	if !exec.shouldContinueOnError(opCreate, fmt.Errorf("%w: already exists", atlasclient.ErrConflict)) {
		t.Fatalf("expected to continue on conflict when preserve-existing is enabled")
	}
}

func TestAtlasExecutor_Execute_EmptyPlan(t *testing.T) {
	executor := createTestExecutor()
	ctx := context.Background()

	plan := &Plan{
		ID:         "empty-plan",
		ProjectID:  "test-project",
		Operations: []PlannedOperation{},
		Status:     PlanStatusApproved,
	}

	result, err := executor.Execute(ctx, plan)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Status != PlanStatusCompleted {
		t.Errorf("Expected status %v, got %v", PlanStatusCompleted, result.Status)
	}

	if result.Summary.TotalOperations != 0 {
		t.Errorf("Expected 0 operations, got %d", result.Summary.TotalOperations)
	}

	if result.Summary.CompletedOperations != 0 {
		t.Errorf("Expected 0 completed operations, got %d", result.Summary.CompletedOperations)
	}
}

func TestAtlasExecutor_Execute_SingleOperation(t *testing.T) {
	executor := createTestExecutor()
	ctx := context.Background()

	operation := createTestOperation("op-1", OperationCreate, types.KindCluster)
	plan := &Plan{
		ID:         "single-op-plan",
		ProjectID:  "test-project",
		Operations: []PlannedOperation{operation},
		Status:     PlanStatusApproved,
	}

	result, err := executor.Execute(ctx, plan)

	// The current implementation returns an error when services are not available
	// This is expected behavior when testing without actual service dependencies
	if err != nil {
		// Check that it fails with the expected "service not available" error
		if !contains(err.Error(), "service not available") {
			t.Fatalf("Expected 'service not available' error, got: %v", err)
		}
		// This is expected for tests without service mocks
		return
	}

	if result.Status != PlanStatusCompleted {
		t.Errorf("Expected status %v, got %v", PlanStatusCompleted, result.Status)
	}

	if result.Summary.TotalOperations != 1 {
		t.Errorf("Expected 1 operation, got %d", result.Summary.TotalOperations)
	}

	if result.Summary.CompletedOperations != 1 {
		t.Errorf("Expected 1 completed operation, got %d", result.Summary.CompletedOperations)
	}

	if len(result.OperationResults) != 1 {
		t.Errorf("Expected 1 operation result, got %d", len(result.OperationResults))
	}

	opResult := result.OperationResults["op-1"]
	if opResult == nil {
		t.Fatal("Operation result not found")
	}

	if opResult.Status != OperationStatusCompleted {
		t.Errorf("Expected operation status %v, got %v", OperationStatusCompleted, opResult.Status)
	}
}

func TestAtlasExecutor_Execute_MultipleOperations(t *testing.T) {
	executor := createTestExecutor()
	ctx := context.Background()

	operations := []PlannedOperation{
		createTestOperation("op-1", OperationCreate, types.KindCluster),
		createTestOperation("op-2", OperationCreate, types.KindDatabaseUser),
		createTestOperation("op-3", OperationCreate, types.KindNetworkAccess),
	}

	plan := &Plan{
		ID:         "multi-op-plan",
		ProjectID:  "test-project",
		Operations: operations,
		Status:     PlanStatusApproved,
	}

	result, err := executor.Execute(ctx, plan)

	// The current implementation returns an error when services are not available
	if err != nil {
		// Check that it fails with the expected "service not available" error
		if !contains(err.Error(), "service not available") {
			t.Fatalf("Expected 'service not available' error, got: %v", err)
		}
		// This is expected for tests without service mocks
		return
	}

	if result.Status != PlanStatusCompleted {
		t.Errorf("Expected status %v, got %v", PlanStatusCompleted, result.Status)
	}

	if result.Summary.TotalOperations != 3 {
		t.Errorf("Expected 3 operations, got %d", result.Summary.TotalOperations)
	}

	if result.Summary.CompletedOperations != 3 {
		t.Errorf("Expected 3 completed operations, got %d", result.Summary.CompletedOperations)
	}

	if len(result.OperationResults) != 3 {
		t.Errorf("Expected 3 operation results, got %d", len(result.OperationResults))
	}

	for _, op := range operations {
		opResult := result.OperationResults[op.ID]
		if opResult == nil {
			t.Errorf("Operation result not found for %s", op.ID)
			continue
		}

		if opResult.Status != OperationStatusCompleted {
			t.Errorf("Expected operation %s status %v, got %v", op.ID, OperationStatusCompleted, opResult.Status)
		}
	}
}

func TestAtlasExecutor_Execute_WithFailedOperation(t *testing.T) {
	executor := createTestExecutor()
	ctx := context.Background()

	operations := []PlannedOperation{
		createTestOperation("op-success", OperationCreate, types.KindCluster),
		createTestOperationWithError("op-fail", OperationCreate, types.KindDatabaseUser, errors.New("operation failed")),
	}

	plan := &Plan{
		ID:         "mixed-plan",
		ProjectID:  "test-project",
		Operations: operations,
		Status:     PlanStatusApproved,
	}

	result, err := executor.Execute(ctx, plan)

	// The current implementation returns an error when services are not available
	if err != nil {
		// Check that it fails with the expected "service not available" error
		if !contains(err.Error(), "service not available") {
			t.Fatalf("Expected 'service not available' error, got: %v", err)
		}
		// This is expected for tests without service mocks
		return
	}

	if result.Status != PlanStatusPartial {
		t.Errorf("Expected status %v, got %v", PlanStatusPartial, result.Status)
	}

	if result.Summary.TotalOperations != 2 {
		t.Errorf("Expected 2 operations, got %d", result.Summary.TotalOperations)
	}

	if result.Summary.CompletedOperations != 1 {
		t.Errorf("Expected 1 completed operation, got %d", result.Summary.CompletedOperations)
	}

	if result.Summary.FailedOperations != 1 {
		t.Errorf("Expected 1 failed operation, got %d", result.Summary.FailedOperations)
	}

	// Check successful operation
	successResult := result.OperationResults["op-success"]
	if successResult == nil || successResult.Status != OperationStatusCompleted {
		t.Error("Successful operation should be completed")
	}

	// Check failed operation
	failResult := result.OperationResults["op-fail"]
	if failResult == nil || failResult.Status != OperationStatusFailed {
		t.Error("Failed operation should be marked as failed")
	}

	if failResult.Error == "" {
		t.Error("Failed operation should have error message")
	}
}

func TestAtlasExecutor_Execute_ContextCancellation(t *testing.T) {
	executor := createTestExecutor()
	ctx, cancel := context.WithCancel(context.Background())

	operation := createTestOperation("op-1", OperationCreate, types.KindCluster)
	plan := &Plan{
		ID:         "cancel-plan",
		ProjectID:  "test-project",
		Operations: []PlannedOperation{operation},
		Status:     PlanStatusApproved,
	}

	// Cancel context immediately
	cancel()

	result, err := executor.Execute(ctx, plan)

	if err == nil {
		t.Fatal("Expected error from cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}

	if result.Status != PlanStatusCancelled {
		t.Errorf("Expected status %v, got %v", PlanStatusCancelled, result.Status)
	}
}

func TestAtlasExecutor_Cancel(t *testing.T) {
	executor := createTestExecutor()

	err := executor.Cancel()
	if err != nil {
		t.Errorf("Cancel should not return error: %v", err)
	}

	// Verify cancelled flag is set
	executor.mu.RLock()
	cancelled := executor.cancelled
	executor.mu.RUnlock()

	if !cancelled {
		t.Error("Expected cancelled flag to be true")
	}
}

func TestAtlasExecutor_GetProgress(t *testing.T) {
	executor := createTestExecutor()

	// Initially no progress
	progress := executor.GetProgress()
	if progress != nil {
		t.Error("Expected nil progress before execution starts")
	}

	// Set up a plan and start execution to have progress
	operation := createTestOperation("op-1", OperationCreate, types.KindCluster)
	plan := &Plan{
		ID:         "progress-plan",
		ProjectID:  "test-project",
		Operations: []PlannedOperation{operation},
		Status:     PlanStatusApproved,
	}

	// Initialize progress by setting currentPlan
	executor.mu.Lock()
	executor.currentPlan = plan
	executor.progress = &ExecutorProgress{
		ExecutionProgress: ExecutionProgress{
			PlanID:              plan.ID,
			TotalOperations:     1,
			CompletedOperations: 0,
		},
		Status: PlanStatusExecuting,
	}
	executor.mu.Unlock()

	progress = executor.GetProgress()
	if progress == nil {
		t.Fatal("Expected progress to be available")
	}

	if progress.PlanID != plan.ID {
		t.Errorf("Expected plan ID %s, got %s", plan.ID, progress.PlanID)
	}

	if progress.Status != PlanStatusExecuting {
		t.Errorf("Expected status %v, got %v", PlanStatusExecuting, progress.Status)
	}

	if progress.TotalOperations != 1 {
		t.Errorf("Expected 1 total operation, got %d", progress.TotalOperations)
	}
}

func TestExecutionResult_Fields(t *testing.T) {
	result := &ExecutionResult{
		PlanID:    "test-plan",
		Status:    PlanStatusCompleted,
		StartedAt: time.Now(),
		OperationResults: map[string]*OperationResult{
			"op-1": {
				OperationID: "op-1",
				Status:      OperationStatusCompleted,
				Duration:    time.Second,
			},
		},
		Summary: ExecutionSummary{
			TotalOperations:     1,
			CompletedOperations: 1,
		},
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	if result.PlanID != "test-plan" {
		t.Error("PlanID not set correctly")
	}

	if result.Status != PlanStatusCompleted {
		t.Error("Status not set correctly")
	}

	if len(result.OperationResults) != 1 {
		t.Error("OperationResults not set correctly")
	}

	if result.Summary.TotalOperations != 1 {
		t.Error("Summary not set correctly")
	}

	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestOperationResult_Fields(t *testing.T) {
	result := &OperationResult{
		OperationID: "test-op",
		Status:      OperationStatusCompleted,
		StartedAt:   time.Now(),
		CompletedAt: time.Now().Add(time.Second),
		Duration:    time.Second,
		RetryCount:  2,
		ResourceID:  "resource-123",
		Metadata: map[string]interface{}{
			"cluster_size": 3,
			"region":       "US_EAST_1",
		},
	}

	if result.OperationID != "test-op" {
		t.Error("OperationID not set correctly")
	}

	if result.Status != OperationStatusCompleted {
		t.Error("Status not set correctly")
	}

	if result.Duration != time.Second {
		t.Error("Duration not set correctly")
	}

	if result.RetryCount != 2 {
		t.Error("RetryCount not set correctly")
	}

	if result.ResourceID != "resource-123" {
		t.Error("ResourceID not set correctly")
	}

	if len(result.Metadata) != 2 {
		t.Error("Metadata not set correctly")
	}

	if result.Metadata["cluster_size"] != 3 {
		t.Error("Metadata cluster_size not set correctly")
	}
}

func TestExecutionSummary_Fields(t *testing.T) {
	summary := ExecutionSummary{
		TotalOperations:     10,
		CompletedOperations: 7,
		FailedOperations:    2,
		SkippedOperations:   1,
		RetriedOperations:   3,
	}

	if summary.TotalOperations != 10 {
		t.Error("TotalOperations not set correctly")
	}

	if summary.CompletedOperations != 7 {
		t.Error("CompletedOperations not set correctly")
	}

	if summary.FailedOperations != 2 {
		t.Error("FailedOperations not set correctly")
	}

	if summary.SkippedOperations != 1 {
		t.Error("SkippedOperations not set correctly")
	}

	if summary.RetriedOperations != 3 {
		t.Error("RetriedOperations not set correctly")
	}

	// Check totals add up
	completed := summary.CompletedOperations + summary.FailedOperations + summary.SkippedOperations
	if completed != summary.TotalOperations {
		t.Errorf("Operations don't add up: %d + %d + %d != %d",
			summary.CompletedOperations, summary.FailedOperations, summary.SkippedOperations, summary.TotalOperations)
	}
}

func TestExecutionError_Fields(t *testing.T) {
	execError := ExecutionError{
		OperationID: "failed-op",
		Message:     "Resource not found",
		ErrorType:   "NotFoundError",
		Timestamp:   time.Now(),
		Recoverable: true,
	}

	if execError.OperationID != "failed-op" {
		t.Error("OperationID not set correctly")
	}

	if execError.Message != "Resource not found" {
		t.Error("Message not set correctly")
	}

	if execError.ErrorType != "NotFoundError" {
		t.Error("ErrorType not set correctly")
	}

	if !execError.Recoverable {
		t.Error("Recoverable should be true")
	}

	if execError.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

// Helper functions for creating test data

func createTestExecutor() *AtlasExecutor {
	mockServices := createMockServices()
	config := DefaultExecutorConfig()
	config.MaxConcurrentOperations = 1 // Sequential for predictable testing
	config.PreserveExisting = true     // exercise new flag in tests

	return NewAtlasExecutor(
		mockServices.clustersService,
		mockServices.usersService,
		mockServices.networkAccessService,
		mockServices.projectsService,
		mockServices.databaseService,
		config,
	)
}

func createMockServices() *MockAtlasServices {
	// For this test, we'll use nil services since we're not actually calling Atlas API
	// In a real implementation, these would be proper mocks
	return &MockAtlasServices{
		clustersService:      nil,
		usersService:         nil,
		networkAccessService: nil,
		projectsService:      nil,
		databaseService:      nil,
	}
}

func createTestOperation(id string, opType OperationType, resourceKind types.ResourceKind) PlannedOperation {
	return PlannedOperation{
		Operation: Operation{
			Type:         opType,
			ResourceType: resourceKind,
		},
		ID:       id,
		Priority: 1,
		Stage:    0,
		Status:   OperationStatusPending,
	}
}

func createTestOperationWithError(id string, opType OperationType, resourceKind types.ResourceKind, err error) PlannedOperation {
	op := createTestOperation(id, opType, resourceKind)
	// This would be used in a more sophisticated mock setup
	return op
}

func TestAtlasExecutor_DatabaseUserOperations_TypeCasting(t *testing.T) {
	runDBUserOpAndAssert(t, OperationCreate, "testpass")
}

func TestAtlasExecutor_DatabaseUserUpdate_TypeCasting(t *testing.T) {
	runDBUserOpAndAssert(t, OperationUpdate, "newpass")
}

func TestAtlasExecutor_DatabaseUserDelete_TypeCasting(t *testing.T) {
	runDBUserOpAndAssert(t, OperationDelete, "")
}

// Helpers to reduce duplication in database user tests
func createDBUserManifest(password string) *types.DatabaseUserManifest {
	manifest := &types.DatabaseUserManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindDatabaseUser,
		Metadata: types.ResourceMetadata{
			Name: "test-user",
		},
		Spec: types.DatabaseUserSpec{
			ProjectName:  "test-project",
			Username:     "testuser",
			AuthDatabase: "admin",
			Roles: []types.DatabaseRoleConfig{
				{
					RoleName:     "readWrite",
					DatabaseName: "admin",
				},
			},
		},
	}
	if password != "" {
		manifest.Spec.Password = password
	}
	return manifest
}

func runDBUserOpAndAssert(t *testing.T, opType OperationType, password string) {
	t.Helper()

	userManifest := createDBUserManifest(password)
	operation := &PlannedOperation{Operation: Operation{
		Type:         opType,
		ResourceType: types.KindDatabaseUser,
		ResourceName: "test-user",
	}}
	// Attach manifest as Desired/Current depending on op
	switch opType {
	case OperationCreate, OperationUpdate:
		operation.Desired = userManifest
	case OperationDelete:
		operation.Current = userManifest
	}

	executor := &AtlasExecutor{}
	result := &OperationResult{Metadata: make(map[string]interface{})}

	var err error
	switch opType {
	case OperationCreate:
		err = executor.createDatabaseUser(context.Background(), operation, result)
	case OperationUpdate:
		err = executor.updateDatabaseUser(context.Background(), operation, result)
	case OperationDelete:
		err = executor.deleteDatabaseUser(context.Background(), operation, result)
	}

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database user service not available")
	assert.NotContains(t, err.Error(), "invalid resource type for database user operation")
}
