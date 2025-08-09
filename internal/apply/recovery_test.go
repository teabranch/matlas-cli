package apply

import (
	"errors"
	"testing"

	"github.com/teabranch/matlas-cli/internal/types"
)

// Recovery unit tests focus on helper functions and analysis logic.
// Service integration tests are handled separately in integration test suite.

// Note: Rollback tests require service integration and are covered in integration tests.
// These tests focus on the helper functions and logic that can be tested in isolation.

func TestRecoveryManager_GetProjectIDForOperation_FromDesired(t *testing.T) {
	rm := &RecoveryManager{
		config: DefaultRecoveryConfig(),
	}

	operation := &PlannedOperation{
		Operation: Operation{
			Desired: &types.ClusterManifest{
				Spec: types.ClusterSpec{
					ProjectName: "test-project-from-desired",
				},
			},
		},
		ID: "test-op",
	}

	projectID, err := rm.getProjectIDForOperation(operation)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if projectID != "test-project-from-desired" {
		t.Errorf("Expected 'test-project-from-desired', got: %s", projectID)
	}
}

func TestRecoveryManager_GetProjectIDForOperation_FromCurrent(t *testing.T) {
	rm := &RecoveryManager{
		config: DefaultRecoveryConfig(),
	}

	operation := &PlannedOperation{
		Operation: Operation{
			Current: &types.DatabaseUserManifest{
				Spec: types.DatabaseUserSpec{
					ProjectName: "test-project-from-current",
				},
			},
		},
		ID: "test-op",
	}

	projectID, err := rm.getProjectIDForOperation(operation)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if projectID != "test-project-from-current" {
		t.Errorf("Expected 'test-project-from-current', got: %s", projectID)
	}
}

func TestRecoveryManager_GetProjectIDForOperation_NotFound(t *testing.T) {
	rm := &RecoveryManager{
		config: DefaultRecoveryConfig(),
	}

	operation := &PlannedOperation{
		Operation: Operation{},
		ID:        "test-op",
	}

	projectID, err := rm.getProjectIDForOperation(operation)

	if err == nil {
		t.Error("Expected error but got nil")
	}
	if projectID != "" {
		t.Errorf("Expected empty string, got: %s", projectID)
	}
}

func TestRecoveryManager_IsNotFoundError(t *testing.T) {
	rm := &RecoveryManager{}

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "not found error",
			err:      errors.New("resource not found"),
			expected: true,
		},
		{
			name:     "cluster not found",
			err:      errors.New("CLUSTER_NOT_FOUND"),
			expected: true,
		},
		{
			name:     "404 error",
			err:      errors.New("HTTP 404 not found"),
			expected: true,
		},
		{
			name:     "does not exist",
			err:      errors.New("cluster does not exist"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("internal server error"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := rm.isNotFoundError(tc.err)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for error: %v", tc.expected, result, tc.err)
			}
		})
	}
}

func TestRecoveryManager_AnalyzeFailure(t *testing.T) {
	rm := &RecoveryManager{
		config:             DefaultRecoveryConfig(),
		idempotencyManager: nil, // Test should handle nil gracefully
	}

	operation := &PlannedOperation{
		Operation: Operation{
			Type:         OperationCreate,
			ResourceName: "test-resource",
		},
		ID: "test-op",
	}

	testCases := []struct {
		name         string
		err          error
		expectedType FailureType
	}{
		{
			name:         "network error",
			err:          errors.New("connection timeout"),
			expectedType: FailureTypeTimeout, // connection timeout is classified as timeout, not network
		},
		{
			name:         "authentication error",
			err:          errors.New("unauthorized access"),
			expectedType: FailureTypeAuthentication,
		},
		{
			name:         "quota error",
			err:          errors.New("quota exceeded"),
			expectedType: FailureTypeQuota,
		},
		{
			name:         "conflict error",
			err:          errors.New("resource already exists"),
			expectedType: FailureTypeConflict,
		},
		{
			name:         "validation error",
			err:          errors.New("invalid input"),
			expectedType: FailureTypeValidation,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			analysis := rm.analyzeFailure(operation, tc.err)

			if analysis.OperationID != operation.ID {
				t.Errorf("Expected operation ID %s, got %s", operation.ID, analysis.OperationID)
			}
			if analysis.FailureType != tc.expectedType {
				t.Errorf("Expected failure type %s, got %s", tc.expectedType, analysis.FailureType)
			}
			if analysis.RootCause == "" {
				t.Error("Expected non-empty root cause")
			}
			if len(analysis.Recommendations) == 0 {
				t.Error("Expected non-empty recommendations")
			}
			if analysis.Confidence <= 0 {
				t.Error("Expected confidence > 0")
			}
		})
	}
}

func TestRecoveryManager_AnalyzeAffectedResources_DependencyInference(t *testing.T) {
	// Prepare idempotency manager with states for a cluster and a user in same project
	im := NewIdempotencyManager(DefaultIdempotencyConfig())

	// Create a cluster operation state
	clusterOp := &PlannedOperation{
		Operation: Operation{
			Type:         OperationCreate,
			ResourceType: types.KindCluster,
			ResourceName: "cluster-a",
		},
		ID: "op-cluster",
	}
	clusterState := im.CreateOperationState(clusterOp, "fp-cluster")
	clusterState.Status = OperationStatusCompleted
	if clusterState.Metadata == nil {
		clusterState.Metadata = make(map[string]interface{})
	}
	clusterState.Metadata["projectID"] = "project-1"
	_ = im.UpdateOperationState(clusterState)

	// Create a database user operation state
	userOp := &PlannedOperation{
		Operation: Operation{
			Type:         OperationCreate,
			ResourceType: types.KindDatabaseUser,
			ResourceName: "user-a",
		},
		ID: "op-user",
	}
	userState := im.CreateOperationState(userOp, "fp-user")
	userState.Status = OperationStatusRunning
	if userState.Metadata == nil {
		userState.Metadata = make(map[string]interface{})
	}
	userState.Metadata["projectID"] = "project-1"
	_ = im.UpdateOperationState(userState)

	// Recovery manager with idempotency
	rm := &RecoveryManager{
		config:             DefaultRecoveryConfig(),
		idempotencyManager: im,
	}

	// Analyze affected resources for a user operation in the same project
	op := &PlannedOperation{
		Operation: Operation{
			Type:         OperationCreate,
			ResourceType: types.KindDatabaseUser,
			ResourceName: "user-a",
			Desired: &types.DatabaseUserManifest{ // to allow project resolution
				Spec: types.DatabaseUserSpec{ProjectName: "project-1", Username: "user-a"},
			},
		},
		ID: "op-user",
	}

	affected := rm.analyzeAffectedResources(op)

	if len(affected) < 2 {
		t.Fatalf("expected at least 2 affected resources (primary + dependent), got %d", len(affected))
	}

	// Verify that the cluster is included as a dependent resource
	foundCluster := false
	for _, a := range affected {
		if a.ResourceKind == types.KindCluster && a.ResourceID == "cluster-a" {
			foundCluster = true
			// completed state should map to healthy/none
			if a.State != ResourceStateHealthy {
				t.Errorf("expected cluster state Healthy, got %s", a.State)
			}
			if a.Impact != ResourceImpactNone {
				t.Errorf("expected cluster impact None, got %s", a.Impact)
			}
		}
	}
	if !foundCluster {
		t.Errorf("expected dependent cluster to be detected in affected resources")
	}
}
