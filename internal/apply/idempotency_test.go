package apply

import (
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestNewIdempotencyManager(t *testing.T) {
	config := DefaultIdempotencyConfig()
	manager := NewIdempotencyManager(config)

	if manager == nil {
		t.Fatal("NewIdempotencyManager returned nil")
	}
	if manager.operationStates == nil {
		t.Error("operationStates should not be nil")
	}
	if manager.resourceFingerprints == nil {
		t.Error("resourceFingerprints should not be nil")
	}
	if manager.resourceOwnership == nil {
		t.Error("resourceOwnership should not be nil")
	}
	if manager.checkpoints == nil {
		t.Error("checkpoints should not be nil")
	}
}

func TestDefaultIdempotencyConfig(t *testing.T) {
	config := DefaultIdempotencyConfig()

	if !config.EnableStateTracking {
		t.Error("EnableStateTracking should be true by default")
	}
	if !config.EnableFingerprinting {
		t.Error("EnableFingerprinting should be true by default")
	}
	if !config.EnableDeduplication {
		t.Error("EnableDeduplication should be true by default")
	}
	if !config.EnableOwnershipTracking {
		t.Error("EnableOwnershipTracking should be true by default")
	}
	if config.StateTTL != 24*time.Hour {
		t.Errorf("Expected StateTTL to be 24h, got %v", config.StateTTL)
	}
	if config.DeduplicationWindow != 1*time.Hour {
		t.Errorf("Expected DeduplicationWindow to be 1h, got %v", config.DeduplicationWindow)
	}
	if config.OwnershipTTL != 2*time.Hour {
		t.Errorf("Expected OwnershipTTL to be 2h, got %v", config.OwnershipTTL)
	}
}

func TestOperationStateLifecycle(t *testing.T) {
	manager := NewIdempotencyManager(DefaultIdempotencyConfig())

	// Create test operation
	operation := &PlannedOperation{
		Operation: Operation{
			Type:         OperationCreate,
			ResourceType: types.KindCluster,
			ResourceName: "test-cluster",
		},
		ID: "op-123",
	}

	// Create operation state
	state := manager.CreateOperationState(operation, "fingerprint-123")
	if state == nil {
		t.Fatal("CreateOperationState returned nil")
	}
	if state.ID != "op-123" {
		t.Errorf("Expected ID to be 'op-123', got %s", state.ID)
	}
	if state.Status != OperationStatusPending {
		t.Errorf("Expected status to be pending, got %s", state.Status)
	}
	if state.ResourceID != "test-cluster" {
		t.Errorf("Expected ResourceID to be 'test-cluster', got %s", state.ResourceID)
	}
	if state.ResourceKind != types.KindCluster {
		t.Errorf("Expected ResourceKind to be cluster, got %s", state.ResourceKind)
	}
	if state.Fingerprint != "fingerprint-123" {
		t.Errorf("Expected Fingerprint to be 'fingerprint-123', got %s", state.Fingerprint)
	}

	// Get operation state
	retrievedState, exists := manager.GetOperationState("op-123")
	if !exists {
		t.Error("GetOperationState should return true for existing state")
	}
	if retrievedState.ID != state.ID {
		t.Errorf("Retrieved state ID mismatch: expected %s, got %s", state.ID, retrievedState.ID)
	}
	if retrievedState.Status != state.Status {
		t.Errorf("Retrieved state status mismatch: expected %s, got %s", state.Status, retrievedState.Status)
	}

	// Update operation state
	retrievedState.Status = OperationStatusRunning
	err := manager.UpdateOperationState(retrievedState)
	if err != nil {
		t.Errorf("UpdateOperationState should not return error: %v", err)
	}

	// Verify update
	updatedState, exists := manager.GetOperationState("op-123")
	if !exists {
		t.Error("GetOperationState should return true after update")
	}
	if updatedState.Status != OperationStatusRunning {
		t.Errorf("Expected status to be running after update, got %s", updatedState.Status)
	}
}

func TestResourceFingerprinting(t *testing.T) {
	manager := NewIdempotencyManager(DefaultIdempotencyConfig())

	// Test resource
	resource := map[string]interface{}{
		"name":         "test-cluster",
		"instanceSize": "M10",
		"region":       "US_EAST_1",
		"createdAt":    "2023-01-01T00:00:00Z", // This should be ignored
	}

	// Compute fingerprint
	fingerprint1, err := manager.ComputeResourceFingerprint(resource, types.KindCluster)
	if err != nil {
		t.Errorf("ComputeResourceFingerprint should not return error: %v", err)
	}
	if fingerprint1 == "" {
		t.Error("Fingerprint should not be empty")
	}

	// Same resource should produce same fingerprint
	fingerprint2, err := manager.ComputeResourceFingerprint(resource, types.KindCluster)
	if err != nil {
		t.Errorf("ComputeResourceFingerprint should not return error: %v", err)
	}
	if fingerprint1 != fingerprint2 {
		t.Error("Same resource should produce same fingerprint")
	}

	// Modified resource should produce different fingerprint
	modifiedResource := map[string]interface{}{
		"name":         "test-cluster",
		"instanceSize": "M20", // Changed
		"region":       "US_EAST_1",
		"createdAt":    "2023-01-01T00:00:00Z",
	}

	fingerprint3, err := manager.ComputeResourceFingerprint(modifiedResource, types.KindCluster)
	if err != nil {
		t.Errorf("ComputeResourceFingerprint should not return error: %v", err)
	}
	if fingerprint1 == fingerprint3 {
		t.Error("Modified resource should produce different fingerprint")
	}
}

func TestIdempotencyChecking(t *testing.T) {
	manager := NewIdempotencyManager(DefaultIdempotencyConfig())

	operation := &PlannedOperation{
		Operation: Operation{
			Type:         OperationCreate,
			ResourceType: types.KindCluster,
			ResourceName: "test-cluster",
		},
		ID: "op-123",
	}

	// Create operation state with fingerprint
	state := manager.CreateOperationState(operation, "fingerprint-123")
	state.Status = OperationStatusCompleted
	err := manager.UpdateOperationState(state)
	if err != nil {
		t.Errorf("UpdateOperationState should not return error: %v", err)
	}

	// Check idempotency with same fingerprint
	isIdempotent, err := manager.IsOperationIdempotent("op-123", "fingerprint-123")
	if err != nil {
		t.Errorf("IsOperationIdempotent should not return error: %v", err)
	}
	if !isIdempotent {
		t.Error("Operation with same fingerprint and completed status should be idempotent")
	}

	// Check idempotency with different fingerprint
	isIdempotent, err = manager.IsOperationIdempotent("op-123", "fingerprint-456")
	if err != nil {
		t.Errorf("IsOperationIdempotent should not return error: %v", err)
	}
	if isIdempotent {
		t.Error("Operation with different fingerprint should not be idempotent")
	}

	// Check idempotency with failed operation
	state.Status = OperationStatusFailed
	err = manager.UpdateOperationState(state)
	if err != nil {
		t.Errorf("UpdateOperationState should not return error: %v", err)
	}

	isIdempotent, err = manager.IsOperationIdempotent("op-123", "fingerprint-123")
	if err != nil {
		t.Errorf("IsOperationIdempotent should not return error: %v", err)
	}
	if isIdempotent {
		t.Error("Failed operations should not be idempotent")
	}
}

func TestResourceOwnership(t *testing.T) {
	manager := NewIdempotencyManager(DefaultIdempotencyConfig())

	resourceID := "test-cluster"
	resourceKind := types.KindCluster
	planID1 := "plan-123"
	planID2 := "plan-456"
	operationID1 := "op-123"
	operationID2 := "op-456"

	// Acquire ownership
	ownership1, err := manager.AcquireResourceOwnership(resourceID, resourceKind, planID1, operationID1)
	if err != nil {
		t.Errorf("AcquireResourceOwnership should not return error: %v", err)
	}
	if ownership1 == nil {
		t.Fatal("AcquireResourceOwnership should return ownership")
	}
	if ownership1.ResourceID != resourceID {
		t.Errorf("Expected ResourceID to be %s, got %s", resourceID, ownership1.ResourceID)
	}
	if ownership1.OwnerPlanID != planID1 {
		t.Errorf("Expected OwnerPlanID to be %s, got %s", planID1, ownership1.OwnerPlanID)
	}
	if ownership1.OwnerOpID != operationID1 {
		t.Errorf("Expected OwnerOpID to be %s, got %s", operationID1, ownership1.OwnerOpID)
	}

	// Try to acquire ownership from different plan - should fail
	ownership2, err := manager.AcquireResourceOwnership(resourceID, resourceKind, planID2, operationID2)
	if err == nil {
		t.Error("AcquireResourceOwnership should return error when resource is already owned")
	}
	if ownership2 != nil {
		t.Error("AcquireResourceOwnership should return nil when resource is already owned")
	}

	// Release ownership
	err = manager.ReleaseResourceOwnership(resourceID, resourceKind, planID1)
	if err != nil {
		t.Errorf("ReleaseResourceOwnership should not return error: %v", err)
	}

	// Should be able to acquire from different plan now
	ownership4, err := manager.AcquireResourceOwnership(resourceID, resourceKind, planID2, operationID2)
	if err != nil {
		t.Errorf("AcquireResourceOwnership should not return error after release: %v", err)
	}
	if ownership4 == nil {
		t.Error("AcquireResourceOwnership should return ownership after release")
	}
	if ownership4.OwnerPlanID != planID2 {
		t.Errorf("Expected new OwnerPlanID to be %s, got %s", planID2, ownership4.OwnerPlanID)
	}
}

func TestCheckpoints(t *testing.T) {
	manager := NewIdempotencyManager(DefaultIdempotencyConfig())

	operationID := "op-123"
	planID := "plan-123"
	stage := "pre-create"

	checkpointData := map[string]interface{}{
		"step":     1,
		"progress": 0.5,
	}

	resourceState := map[string]interface{}{
		"status": "creating",
		"id":     "partial-id",
	}

	// Create checkpoint
	checkpoint, err := manager.CreateCheckpoint(operationID, planID, stage, checkpointData, resourceState)
	if err != nil {
		t.Errorf("CreateCheckpoint should not return error: %v", err)
	}
	if checkpoint == nil {
		t.Fatal("CreateCheckpoint should return checkpoint")
	}
	if checkpoint.OperationID != operationID {
		t.Errorf("Expected OperationID to be %s, got %s", operationID, checkpoint.OperationID)
	}
	if checkpoint.PlanID != planID {
		t.Errorf("Expected PlanID to be %s, got %s", planID, checkpoint.PlanID)
	}
	if checkpoint.Stage != stage {
		t.Errorf("Expected Stage to be %s, got %s", stage, checkpoint.Stage)
	}

	// Get latest checkpoint
	latestCheckpoint, exists := manager.GetLatestCheckpoint(operationID)
	if !exists {
		t.Error("GetLatestCheckpoint should return true for existing checkpoint")
	}
	if latestCheckpoint.ID != checkpoint.ID {
		t.Errorf("Expected latest checkpoint ID to be %s, got %s", checkpoint.ID, latestCheckpoint.ID)
	}

	// Non-existent operation should return false
	_, exists = manager.GetLatestCheckpoint("non-existent")
	if exists {
		t.Error("GetLatestCheckpoint should return false for non-existent operation")
	}
}

func TestUpdateOperationStateValidation(t *testing.T) {
	manager := NewIdempotencyManager(DefaultIdempotencyConfig())

	// Test with nil state
	err := manager.UpdateOperationState(nil)
	if err == nil {
		t.Error("UpdateOperationState should return error for nil state")
	}
	if err != nil && err.Error() != "operation state cannot be nil" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestIdempotencyManagerWithDisabledFeatures(t *testing.T) {
	// Test with all features disabled
	config := IdempotencyConfig{
		EnableStateTracking:     false,
		EnableFingerprinting:    false,
		EnableDeduplication:     false,
		EnableOwnershipTracking: false,
	}

	manager := NewIdempotencyManager(config)

	// Fingerprinting should return empty
	fingerprint, err := manager.ComputeResourceFingerprint(map[string]interface{}{"test": "value"}, types.KindCluster)
	if err != nil {
		t.Errorf("ComputeResourceFingerprint should not return error: %v", err)
	}
	if fingerprint != "" {
		t.Error("ComputeResourceFingerprint should return empty string when disabled")
	}

	// Idempotency check should return false
	isIdempotent, err := manager.IsOperationIdempotent("op-123", "fingerprint-123")
	if err != nil {
		t.Errorf("IsOperationIdempotent should not return error: %v", err)
	}
	if isIdempotent {
		t.Error("IsOperationIdempotent should return false when fingerprinting is disabled")
	}

	// Ownership operations should return nil
	ownership, err := manager.AcquireResourceOwnership("test-cluster", types.KindCluster, "plan-123", "op-123")
	if err != nil {
		t.Errorf("AcquireResourceOwnership should not return error: %v", err)
	}
	if ownership != nil {
		t.Error("AcquireResourceOwnership should return nil when ownership tracking is disabled")
	}

	err = manager.ReleaseResourceOwnership("test-cluster", types.KindCluster, "plan-123")
	if err != nil {
		t.Errorf("ReleaseResourceOwnership should not return error: %v", err)
	}
}
