package apply

import (
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestNewDiffEngine(t *testing.T) {
	engine := NewDiffEngine()

	if engine == nil {
		t.Fatal("NewDiffEngine returned nil")
	}

	if !engine.IgnoreOrderInSlices {
		t.Error("Expected IgnoreOrderInSlices to be true by default")
	}

	if engine.CompareTimestamps {
		t.Error("Expected CompareTimestamps to be false by default")
	}

	if !engine.IgnoreDefaults {
		t.Error("Expected IgnoreDefaults to be true by default")
	}
}

func TestDiffEngine_ComputeProjectDiff_EmptyStates(t *testing.T) {
	engine := NewDiffEngine()

	// Test with both states nil
	diff, err := engine.ComputeProjectDiff(nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(diff.Operations) != 0 {
		t.Errorf("Expected 0 operations, got %d", len(diff.Operations))
	}

	if diff.Summary.TotalOperations != 0 {
		t.Errorf("Expected 0 total operations, got %d", diff.Summary.TotalOperations)
	}
}

func TestDiffEngine_ComputeProjectDiff_CreateCluster(t *testing.T) {
	engine := NewDiffEngine()

	desired := &ProjectState{
		Clusters: []types.ClusterManifest{
			{
				Metadata: types.ResourceMetadata{Name: "test-cluster"},
				Spec: types.ClusterSpec{
					TierType:       "M10",
					MongoDBVersion: "7.0",
				},
			},
		},
	}

	current := &ProjectState{}

	diff, err := engine.ComputeProjectDiff(desired, current)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(diff.Operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(diff.Operations))
	}

	op := diff.Operations[0]
	if op.Type != OperationCreate {
		t.Errorf("Expected CREATE operation, got %s", op.Type)
	}

	if op.ResourceType != types.KindCluster {
		t.Errorf("Expected Cluster resource type, got %s", op.ResourceType)
	}

	if op.ResourceName != "test-cluster" {
		t.Errorf("Expected test-cluster name, got %s", op.ResourceName)
	}

	if op.Impact == nil {
		t.Error("Expected impact assessment")
	} else {
		if op.Impact.EstimatedDuration == 0 {
			t.Error("Expected non-zero estimated duration")
		}
		if op.Impact.RiskLevel == "" {
			t.Error("Expected risk level to be set")
		}
	}
}

func TestDiffEngine_ComputeProjectDiff_UpdateCluster(t *testing.T) {
	engine := NewDiffEngine()

	currentCluster := types.ClusterManifest{
		Metadata: types.ResourceMetadata{Name: "test-cluster"},
		Spec: types.ClusterSpec{
			TierType:       "M10",
			MongoDBVersion: "7.0",
		},
	}

	desiredCluster := types.ClusterManifest{
		Metadata: types.ResourceMetadata{Name: "test-cluster"},
		Spec: types.ClusterSpec{
			TierType:       "M20", // Changed
			MongoDBVersion: "7.0",
		},
	}

	desired := &ProjectState{
		Clusters: []types.ClusterManifest{desiredCluster},
	}

	current := &ProjectState{
		Clusters: []types.ClusterManifest{currentCluster},
	}

	diff, err := engine.ComputeProjectDiff(desired, current)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(diff.Operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(diff.Operations))
	}

	op := diff.Operations[0]
	if op.Type != OperationUpdate {
		t.Errorf("Expected UPDATE operation, got %s", op.Type)
	}

	if len(op.FieldChanges) == 0 {
		t.Error("Expected field changes for update operation")
	}
}

func TestDiffEngine_ComputeProjectDiff_DeleteCluster(t *testing.T) {
	engine := NewDiffEngine()

	current := &ProjectState{
		Clusters: []types.ClusterManifest{
			{
				Metadata: types.ResourceMetadata{Name: "test-cluster"},
				Spec: types.ClusterSpec{
					TierType:       "M10",
					MongoDBVersion: "7.0",
				},
			},
		},
	}

	desired := &ProjectState{}

	diff, err := engine.ComputeProjectDiff(desired, current)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(diff.Operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(diff.Operations))
	}

	op := diff.Operations[0]
	if op.Type != OperationDelete {
		t.Errorf("Expected DELETE operation, got %s", op.Type)
	}

	if op.Impact == nil {
		t.Error("Expected impact assessment")
	} else {
		if !op.Impact.IsDestructive {
			t.Error("Expected cluster deletion to be marked as destructive")
		}
		if op.Impact.RiskLevel != RiskLevelHigh {
			t.Errorf("Expected HIGH risk level for cluster deletion, got %s", op.Impact.RiskLevel)
		}
	}
}

func TestDiffEngine_ComputeProjectDiff_NoChanges(t *testing.T) {
	engine := NewDiffEngine()

	cluster := types.ClusterManifest{
		Metadata: types.ResourceMetadata{Name: "test-cluster"},
		Spec: types.ClusterSpec{
			TierType:       "M10",
			MongoDBVersion: "7.0",
		},
	}

	state := &ProjectState{
		Clusters: []types.ClusterManifest{cluster},
	}

	diff, err := engine.ComputeProjectDiff(state, state)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(diff.Operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(diff.Operations))
	}

	op := diff.Operations[0]
	if op.Type != OperationNoChange {
		t.Errorf("Expected NO_CHANGE operation, got %s", op.Type)
	}

	if diff.Summary.NoChangeOperations != 1 {
		t.Errorf("Expected 1 no-change operation in summary, got %d", diff.Summary.NoChangeOperations)
	}
}

func TestDiffEngine_ComputeFieldChanges(t *testing.T) {
	engine := NewDiffEngine()

	current := &types.ClusterManifest{
		Metadata: types.ResourceMetadata{Name: "test-cluster"},
		Spec: types.ClusterSpec{
			TierType:       "M10",
			MongoDBVersion: "7.0",
		},
	}

	desired := &types.ClusterManifest{
		Metadata: types.ResourceMetadata{Name: "test-cluster"},
		Spec: types.ClusterSpec{
			TierType:       "M20", // Changed
			MongoDBVersion: "7.0",
		},
	}

	changes := engine.computeFieldChanges(desired, current)

	if len(changes) == 0 {
		t.Error("Expected field changes to be detected")
	}

	// Look for the TierType change
	found := false
	for _, change := range changes {
		if change.Path == "Spec.TierType" {
			found = true
			if change.Type != ChangeTypeModify {
				t.Errorf("Expected modify change type, got %s", change.Type)
			}
			if change.OldValue != "M10" {
				t.Errorf("Expected old value M10, got %v", change.OldValue)
			}
			if change.NewValue != "M20" {
				t.Errorf("Expected new value M20, got %v", change.NewValue)
			}
		}
	}

	if !found {
		t.Error("Expected to find TierType change in field changes")
	}
}

func TestDiffEngine_ComputeSummary(t *testing.T) {
	engine := NewDiffEngine()

	operations := []Operation{
		{
			Type:         OperationCreate,
			ResourceType: types.KindCluster,
			Impact: &OperationImpact{
				EstimatedDuration: time.Minute * 15,
				RiskLevel:         RiskLevelMedium,
			},
		},
		{
			Type:         OperationUpdate,
			ResourceType: types.KindDatabaseUser,
			Impact: &OperationImpact{
				EstimatedDuration: time.Second * 30,
				RiskLevel:         RiskLevelLow,
			},
		},
		{
			Type:         OperationDelete,
			ResourceType: types.KindCluster,
			Impact: &OperationImpact{
				IsDestructive:     true,
				EstimatedDuration: time.Minute * 10,
				RiskLevel:         RiskLevelHigh,
			},
		},
		{
			Type:         OperationNoChange,
			ResourceType: types.KindNetworkAccess,
		},
	}

	summary := engine.computeSummary(operations)

	if summary.TotalOperations != 4 {
		t.Errorf("Expected 4 total operations, got %d", summary.TotalOperations)
	}

	if summary.CreateOperations != 1 {
		t.Errorf("Expected 1 create operation, got %d", summary.CreateOperations)
	}

	if summary.UpdateOperations != 1 {
		t.Errorf("Expected 1 update operation, got %d", summary.UpdateOperations)
	}

	if summary.DeleteOperations != 1 {
		t.Errorf("Expected 1 delete operation, got %d", summary.DeleteOperations)
	}

	if summary.NoChangeOperations != 1 {
		t.Errorf("Expected 1 no-change operation, got %d", summary.NoChangeOperations)
	}

	if summary.DestructiveOperations != 1 {
		t.Errorf("Expected 1 destructive operation, got %d", summary.DestructiveOperations)
	}

	expectedDuration := time.Minute*15 + time.Second*30 + time.Minute*10
	if summary.EstimatedDuration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, summary.EstimatedDuration)
	}

	if summary.HighestRiskLevel != RiskLevelHigh {
		t.Errorf("Expected highest risk level HIGH, got %s", summary.HighestRiskLevel)
	}
}

func TestDiffEngine_ImpactAssessment_ClusterOperations(t *testing.T) {
	engine := NewDiffEngine()

	// Test cluster creation
	createOp := &Operation{
		Type:         OperationCreate,
		ResourceType: types.KindCluster,
		ResourceName: "test-cluster",
	}
	impact := engine.computeOperationImpact(createOp)

	if impact.RiskLevel != RiskLevelMedium {
		t.Errorf("Expected MEDIUM risk for cluster creation, got %s", impact.RiskLevel)
	}

	if len(impact.Warnings) == 0 {
		t.Error("Expected warnings for cluster creation")
	}

	// Test cluster deletion
	deleteOp := &Operation{
		Type:         OperationDelete,
		ResourceType: types.KindCluster,
		ResourceName: "test-cluster",
	}
	impact = engine.computeOperationImpact(deleteOp)

	if !impact.IsDestructive {
		t.Error("Expected cluster deletion to be destructive")
	}

	if impact.RiskLevel != RiskLevelHigh {
		t.Errorf("Expected HIGH risk for cluster deletion, got %s", impact.RiskLevel)
	}

	if !impact.RequiresDowntime {
		t.Error("Expected cluster deletion to require downtime")
	}
}

func TestDiffEngine_ImpactAssessment_ClusterUpdates(t *testing.T) {
	engine := NewDiffEngine()

	// Test cluster update with instance size change
	updateOp := &Operation{
		Type:         OperationUpdate,
		ResourceType: types.KindCluster,
		ResourceName: "test-cluster",
		FieldChanges: []FieldChange{
			{
				Path:     "Spec.InstanceSize",
				OldValue: "M10",
				NewValue: "M20",
				Type:     ChangeTypeModify,
			},
		},
	}

	impact := engine.computeOperationImpact(updateOp)

	if !impact.RequiresDowntime {
		t.Error("Expected instance size change to require downtime")
	}

	if impact.RiskLevel != RiskLevelHigh {
		t.Errorf("Expected HIGH risk for instance size change, got %s", impact.RiskLevel)
	}

	// Test disk size decrease (should be critical)
	updateOp.FieldChanges = []FieldChange{
		{
			Path:     "Spec.DiskSizeGB",
			OldValue: float64(100),
			NewValue: float64(50),
			Type:     ChangeTypeModify,
		},
	}

	impact = engine.computeOperationImpact(updateOp)

	if !impact.IsDestructive {
		t.Error("Expected disk size decrease to be destructive")
	}

	if impact.RiskLevel != RiskLevelCritical {
		t.Errorf("Expected CRITICAL risk for disk size decrease, got %s", impact.RiskLevel)
	}
}

func TestDiffEngine_DatabaseUsers(t *testing.T) {
	engine := NewDiffEngine()

	desired := &ProjectState{
		DatabaseUsers: []types.DatabaseUserManifest{
			{
				Metadata: types.ResourceMetadata{Name: "test-user"},
				Spec: types.DatabaseUserSpec{
					Username:     "testuser",
					AuthDatabase: "admin",
					Roles: []types.DatabaseRoleConfig{
						{RoleName: "readWrite", DatabaseName: "testdb"},
					},
				},
			},
		},
	}

	current := &ProjectState{}

	diff, err := engine.ComputeProjectDiff(desired, current)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(diff.Operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(diff.Operations))
	}

	op := diff.Operations[0]
	if op.Type != OperationCreate {
		t.Errorf("Expected CREATE operation, got %s", op.Type)
	}

	if op.ResourceType != types.KindDatabaseUser {
		t.Errorf("Expected DatabaseUser resource type, got %s", op.ResourceType)
	}
}

func TestDiffEngine_NetworkAccess(t *testing.T) {
	engine := NewDiffEngine()

	desired := &ProjectState{
		NetworkAccess: []types.NetworkAccessManifest{
			{
				Metadata: types.ResourceMetadata{Name: "office-ip"},
				Spec: types.NetworkAccessSpec{
					IPAddress: "192.168.1.1",
					Comment:   "Office network",
				},
			},
		},
	}

	current := &ProjectState{}

	diff, err := engine.ComputeProjectDiff(desired, current)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(diff.Operations) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(diff.Operations))
	}

	op := diff.Operations[0]
	if op.Type != OperationCreate {
		t.Errorf("Expected CREATE operation, got %s", op.Type)
	}

	if op.ResourceType != types.KindNetworkAccess {
		t.Errorf("Expected NetworkAccess resource type, got %s", op.ResourceType)
	}
}

func TestDiffEngine_RiskLevelComparison(t *testing.T) {
	engine := NewDiffEngine()

	testCases := []struct {
		level    RiskLevel
		expected int
	}{
		{RiskLevelLow, 1},
		{RiskLevelMedium, 2},
		{RiskLevelHigh, 3},
		{RiskLevelCritical, 4},
	}

	for _, tc := range testCases {
		actual := engine.riskLevelValue(tc.level)
		if actual != tc.expected {
			t.Errorf("Expected risk level value %d for %s, got %d", tc.expected, tc.level, actual)
		}
	}
}
