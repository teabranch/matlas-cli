package apply

import (
	"fmt"
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestPlanBuilder_Build_Basic(t *testing.T) {
	builder := NewPlanBuilder("test-project")

	// Test empty operations should fail
	_, err := builder.Build()
	if err == nil {
		t.Error("expected error for empty operations")
	}

	// Add an operation and build
	op := Operation{
		Type:         OperationCreate,
		ResourceType: types.KindCluster,
		ResourceName: "test-cluster",
	}
	builder.AddOperation(op)

	plan, err := builder.Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan == nil {
		t.Fatal("expected plan but got nil")
	}

	// Verify basic plan properties
	if plan.ProjectID != "test-project" {
		t.Errorf("expected ProjectID 'test-project', got '%s'", plan.ProjectID)
	}
	if len(plan.Operations) != 1 {
		t.Errorf("expected 1 operation, got %d", len(plan.Operations))
	}
	if plan.Status != PlanStatusDraft {
		t.Errorf("expected status %s, got %s", PlanStatusDraft, plan.Status)
	}
}

func TestPlan_Approve(t *testing.T) {
	plan := &Plan{
		Status: PlanStatusDraft,
		ApprovalInfo: &ApprovalInfo{
			Required: true,
		},
	}

	err := plan.Approve("test-user", "looks good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plan.Status != PlanStatusApproved {
		t.Errorf("expected status %s, got %s", PlanStatusApproved, plan.Status)
	}
	if !plan.ApprovalInfo.Approved {
		t.Error("expected plan to be approved")
	}
	if plan.ApprovalInfo.ApprovedBy != "test-user" {
		t.Errorf("expected approver 'test-user', got '%s'", plan.ApprovalInfo.ApprovedBy)
	}
}

func TestPlan_CanExecute(t *testing.T) {
	// Test approved plan can execute
	plan := &Plan{Status: PlanStatusApproved}
	if err := plan.CanExecute(); err != nil {
		t.Errorf("approved plan should be executable: %v", err)
	}

	// Test draft plan requiring approval cannot execute
	plan = &Plan{
		Status:       PlanStatusDraft,
		ApprovalInfo: &ApprovalInfo{Required: true, Approved: false},
	}
	if err := plan.CanExecute(); err == nil {
		t.Error("draft plan requiring approval should not be executable")
	}

	// Test executing plan cannot execute again
	plan = &Plan{Status: PlanStatusExecuting}
	if err := plan.CanExecute(); err == nil {
		t.Error("executing plan should not be executable again")
	}
}

func TestPlan_GetOperationsInStage(t *testing.T) {
	plan := &Plan{
		Operations: []PlannedOperation{
			{Stage: 0, Operation: Operation{ResourceName: "op1"}},
			{Stage: 1, Operation: Operation{ResourceName: "op2"}},
			{Stage: 0, Operation: Operation{ResourceName: "op3"}},
		},
	}

	stage0Ops := plan.GetOperationsInStage(0)
	if len(stage0Ops) != 2 {
		t.Errorf("expected 2 operations in stage 0, got %d", len(stage0Ops))
	}

	stage1Ops := plan.GetOperationsInStage(1)
	if len(stage1Ops) != 1 {
		t.Errorf("expected 1 operation in stage 1, got %d", len(stage1Ops))
	}

	stage2Ops := plan.GetOperationsInStage(2)
	if len(stage2Ops) != 0 {
		t.Errorf("expected 0 operations in stage 2, got %d", len(stage2Ops))
	}
}

func TestPlan_Serialization(t *testing.T) {
	originalPlan := &Plan{
		ID:        "test-plan",
		ProjectID: "test-project",
		CreatedAt: time.Now(),
		Operations: []PlannedOperation{
			{
				Operation: Operation{
					Type:         OperationCreate,
					ResourceType: types.KindCluster,
					ResourceName: "test-cluster",
				},
				ID:       "op-1",
				Stage:    0,
				Priority: 100,
				Status:   OperationStatusPending,
			},
		},
		Status: PlanStatusDraft,
	}

	// Test JSON serialization
	jsonData, err := originalPlan.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize plan: %v", err)
	}
	if len(jsonData) == 0 {
		t.Error("expected non-empty JSON data")
	}

	// Test JSON deserialization
	deserializedPlan, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("failed to deserialize plan: %v", err)
	}

	if deserializedPlan.ID != originalPlan.ID {
		t.Errorf("expected ID %s, got %s", originalPlan.ID, deserializedPlan.ID)
	}
	if deserializedPlan.ProjectID != originalPlan.ProjectID {
		t.Errorf("expected ProjectID %s, got %s", originalPlan.ProjectID, deserializedPlan.ProjectID)
	}
}

func TestDependencyResolver_Basic(t *testing.T) {
	resolver := NewDependencyResolver()

	// Add a cluster
	clusterSpec := types.ClusterSpec{ProjectName: "test-project"}
	resolver.AddResource("test-cluster", types.KindCluster, clusterSpec)

	// Add a database user that should depend on the cluster
	userSpec := types.DatabaseUserSpec{
		ProjectName: "test-project",
		Username:    "test-user",
	}
	resolver.AddResource("test-user", types.KindDatabaseUser, userSpec)

	analysis, err := resolver.ResolveDependencies()
	if err != nil {
		t.Fatalf("failed to resolve dependencies: %v", err)
	}
	if analysis == nil {
		t.Fatal("expected analysis but got nil")
	}

	// Should have detected automatic dependency: user -> cluster
	if analysis.TotalDependencies == 0 {
		t.Error("expected at least one dependency")
	}
	
	// Verify the dependency was detected correctly
	foundDep := false
	for _, dep := range analysis.Dependencies {
		if dep.Source == "test-user" && dep.Target == "test-cluster" && dep.Type == DependencyTypeAutomatic {
			foundDep = true
			break
		}
	}
	if !foundDep {
		t.Error("expected to find automatic dependency from test-user to test-cluster")
	}

	// Check that both resources are in the levels map
	_, clusterExists := analysis.Levels["test-cluster"]
	_, userExists := analysis.Levels["test-user"]

	if !clusterExists {
		t.Error("cluster not found in dependency levels")
	}
	if !userExists {
		t.Error("user not found in dependency levels")
	}
}

func TestDependencyResolver_CircularDependency(t *testing.T) {
	resolver := NewDependencyResolver()

	// Create two cluster specs with circular dependency
	cluster1Spec := types.ClusterManifest{
		Metadata: types.ResourceMetadata{
			Name:      "cluster1",
			DependsOn: []string{"cluster2"}, // Explicit dependency on cluster2
		},
		Spec: types.ClusterSpec{ProjectName: "test-project"},
	}

	cluster2Spec := types.ClusterManifest{
		Metadata: types.ResourceMetadata{
			Name:      "cluster2",
			DependsOn: []string{"cluster1"}, // Explicit dependency on cluster1 - creates cycle
		},
		Spec: types.ClusterSpec{ProjectName: "test-project"},
	}

	resolver.AddResource("cluster1", types.KindCluster, cluster1Spec)
	resolver.AddResource("cluster2", types.KindCluster, cluster2Spec)

	_, err := resolver.ResolveDependencies()
	if err == nil {
		t.Error("expected error for circular dependency but got none")
	}
	if !containsString(err.Error(), "circular dependencies detected") {
		t.Errorf("expected circular dependency error, got: %v", err)
	}
}

func TestDependencyResolver_ExplicitDependencies(t *testing.T) {
	resolver := NewDependencyResolver()

	// Add resources with explicit dependencies
	cluster1Spec := types.ClusterManifest{
		Metadata: types.ResourceMetadata{
			Name: "cluster1",
		},
		Spec: types.ClusterSpec{ProjectName: "test-project"},
	}

	cluster2Spec := types.ClusterManifest{
		Metadata: types.ResourceMetadata{
			Name:      "cluster2",
			DependsOn: []string{"cluster1"}, // Explicit dependency
		},
		Spec: types.ClusterSpec{ProjectName: "test-project"},
	}

	resolver.AddResource("cluster1", types.KindCluster, cluster1Spec)
	resolver.AddResource("cluster2", types.KindCluster, cluster2Spec)

	analysis, err := resolver.ResolveDependencies()
	if err != nil {
		t.Fatalf("failed to resolve dependencies: %v", err)
	}

	// Should have explicit dependency: cluster2 -> cluster1
	if analysis.TotalDependencies == 0 {
		t.Error("expected at least one dependency")
	}

	// Check dependency is explicit type
	foundExplicit := false
	for _, dep := range analysis.Dependencies {
		if dep.Source == "cluster2" && dep.Target == "cluster1" && dep.Type == DependencyTypeExplicit {
			foundExplicit = true
			break
		}
	}
	if !foundExplicit {
		t.Error("expected to find explicit dependency from cluster2 to cluster1")
	}
}

func TestDependencyResolver_Visualization(t *testing.T) {
	resolver := NewDependencyResolver()

	clusterSpec := types.ClusterSpec{ProjectName: "test-project"}
	userSpec := types.DatabaseUserSpec{ProjectName: "test-project"}

	resolver.AddResource("test-cluster", types.KindCluster, clusterSpec)
	resolver.AddResource("test-user", types.KindDatabaseUser, userSpec)

	analysis, err := resolver.ResolveDependencies()
	if err != nil {
		t.Fatalf("failed to resolve dependencies: %v", err)
	}

	visualization := resolver.VisualizeDependencies(analysis)

	expectedSections := []string{
		"Dependency Analysis",
		"Critical Path",
		"Execution Stages",
		"Dependencies",
	}

	for _, section := range expectedSections {
		if !containsString(visualization, section) {
			t.Errorf("visualization missing section: %s", section)
		}
	}
}

func TestPlanOptimizer_Basic(t *testing.T) {
	// Create a plan with multiple operations
	operations := []PlannedOperation{
		{
			Operation: Operation{
				Type:         OperationCreate,
				ResourceType: types.KindCluster,
				ResourceName: "cluster1",
				Impact: &OperationImpact{
					EstimatedDuration: 5 * time.Minute,
					RiskLevel:         RiskLevelLow,
				},
			},
			ID:           "op-1",
			Stage:        0,
			Priority:     100,
			Status:       OperationStatusPending,
			Dependencies: []string{},
		},
		{
			Operation: Operation{
				Type:         OperationCreate,
				ResourceType: types.KindDatabaseUser,
				ResourceName: "user1",
				Impact: &OperationImpact{
					EstimatedDuration: 30 * time.Second,
					RiskLevel:         RiskLevelLow,
				},
			},
			ID:           "op-2",
			Stage:        1,
			Priority:     80,
			Status:       OperationStatusPending,
			Dependencies: []string{"op-1"},
		},
	}

	plan := &Plan{
		ID:         "test-plan",
		ProjectID:  "test-project",
		Operations: operations,
		Summary: PlanSummary{
			TotalOperations:       2,
			EstimatedDuration:     5*time.Minute + 30*time.Second,
			OperationsByStage:     map[int]int{0: 1, 1: 1},
			ParallelizationFactor: 1.0,
		},
	}

	optimizer := NewPlanOptimizer()
	result, err := optimizer.OptimizePlan(plan)

	if err != nil {
		t.Fatalf("failed to optimize plan: %v", err)
	}
	if result == nil {
		t.Fatal("expected optimization result but got nil")
	}
	
	if result.OriginalPlan != plan {
		t.Error("expected original plan to match input")
	}
	if result.OptimizedPlan == nil {
		t.Error("expected optimized plan")
	}
	if result.Statistics.OptimizationTime == 0 {
		t.Error("expected non-zero optimization time")
	}
}

func TestPlanOptimizer_ParallelExecution(t *testing.T) {
	optimizer := NewPlanOptimizer().WithMaxParallelOps(5)

	// Create plan with operations that can be parallelized
	operations := []PlannedOperation{
		{
			Operation: Operation{
				Type:         OperationCreate,
				ResourceType: types.KindDatabaseUser,
				ResourceName: "user1",
			},
			ID:    "op-1",
			Stage: 0,
		},
		{
			Operation: Operation{
				Type:         OperationCreate,
				ResourceType: types.KindDatabaseUser,
				ResourceName: "user2",
			},
			ID:    "op-2",
			Stage: 0,
		},
		{
			Operation: Operation{
				Type:         OperationCreate,
				ResourceType: types.KindNetworkAccess,
				ResourceName: "access1",
			},
			ID:    "op-3",
			Stage: 0,
		},
		{
			Operation: Operation{
				Type:         OperationCreate,
				ResourceType: types.KindNetworkAccess,
				ResourceName: "access2",
			},
			ID:    "op-4",
			Stage: 0,
		},
	}

	plan := &Plan{
		Operations: operations,
		Summary: PlanSummary{
			OperationsByStage: map[int]int{0: 4},
		},
	}

	actions := optimizer.optimizeParallelExecution(plan)
	
	// Should have identified parallel groups
	if len(actions) == 0 {
		t.Error("expected parallel execution optimizations")
	}
	
	// Check that parallel groups were identified
	hasParallelGroup := false
	for _, action := range actions {
		if action.Type == OptimizationParallelization && len(action.Operations) >= 2 {
			hasParallelGroup = true
			break
		}
	}
	if !hasParallelGroup {
		t.Error("expected to find parallel execution groups")
	}
}

func TestPlanOptimizer_BatchingDetailed(t *testing.T) {
	optimizer := NewPlanOptimizer().WithBatching(true)

	// Create plan with many similar operations that can be batched
	var operations []PlannedOperation
	for i := 0; i < 15; i++ {
		op := PlannedOperation{
			Operation: Operation{
				Type:         OperationCreate,
				ResourceType: types.KindDatabaseUser,
				ResourceName: fmt.Sprintf("user%d", i),
			},
			ID:    fmt.Sprintf("op-%d", i),
			Stage: 0,
		}
		operations = append(operations, op)
	}

	plan := &Plan{
		Operations: operations,
		Summary: PlanSummary{
			OperationsByStage: map[int]int{0: 15},
		},
	}

	actions := optimizer.optimizeBatching(plan)
	
	// Should have created batching actions
	if len(actions) == 0 {
		t.Error("expected batching optimizations for 15 similar operations")
	}
	
	// Check that operations were actually batched
	batchedCount := 0
	for _, op := range plan.Operations {
		if op.BatchID != "" {
			batchedCount++
		}
	}
	if batchedCount < 10 { // Should batch most of the 15 operations
		t.Errorf("expected at least 10 operations to be batched, got %d", batchedCount)
	}
}

func TestExecutionProgress_ProgressPercentage(t *testing.T) {
	tests := []struct {
		name               string
		completedOps       int
		totalOps           int
		expectedPercentage float64
	}{
		{"no operations", 0, 0, 0},
		{"half complete", 5, 10, 50},
		{"fully complete", 10, 10, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := ExecutionProgress{
				CompletedOperations: tt.completedOps,
				TotalOperations:     tt.totalOps,
			}

			percentage := progress.ProgressPercentage()
			if percentage != tt.expectedPercentage {
				t.Errorf("expected %.2f%%, got %.2f%%", tt.expectedPercentage, percentage)
			}
		})
	}
}

func TestPlanBuilder_AutomaticDependencies(t *testing.T) {
	builder := NewPlanBuilder("test-project")

	// Add cluster first
	clusterOp := Operation{
		Type:         OperationCreate,
		ResourceType: types.KindCluster,
		ResourceName: "test-cluster",
	}

	// Add database user second
	userOp := Operation{
		Type:         OperationCreate,
		ResourceType: types.KindDatabaseUser,
		ResourceName: "test-user",
	}

	builder.AddOperation(clusterOp).AddOperation(userOp)

	plan, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to build plan: %v", err)
	}

	// Find the user operation
	var userPlannedOp *PlannedOperation
	for i := range plan.Operations {
		if plan.Operations[i].ResourceType == types.KindDatabaseUser {
			userPlannedOp = &plan.Operations[i]
			break
		}
	}

	if userPlannedOp == nil {
		t.Fatal("user operation not found in plan")
	}

	// User should depend on cluster
	if len(userPlannedOp.Dependencies) == 0 {
		t.Error("expected user operation to have dependencies")
	}

	// User should be in a later stage than cluster
	var clusterStage, userStage int
	for _, op := range plan.Operations {
		if op.ResourceType == types.KindCluster {
			clusterStage = op.Stage
		} else if op.ResourceType == types.KindDatabaseUser {
			userStage = op.Stage
		}
	}
	if userStage <= clusterStage {
		t.Errorf("user stage (%d) should be greater than cluster stage (%d)", userStage, clusterStage)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())
}
