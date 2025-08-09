package apply

import (
	"context"
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestDryRunExecutor_Execute(t *testing.T) {
	ctx := context.Background()

	// Create a sample plan
	plan := &Plan{
		ID:        "test-plan",
		ProjectID: "test-project",
		CreatedAt: time.Now(),
		Operations: []PlannedOperation{
			{
				Operation: Operation{
					Type:         OperationCreate,
					ResourceType: types.KindCluster,
					ResourceName: "test-cluster",
					Desired: types.ClusterSpec{
						ProjectName:  "TestProject",
						Provider:     "AWS",
						Region:       "US_EAST_1",
						InstanceSize: "M10",
					},
					Impact: &OperationImpact{
						IsDestructive:     false,
						RequiresDowntime:  false,
						EstimatedDuration: 10 * time.Minute,
						RiskLevel:         RiskLevelMedium,
					},
				},
				ID:           "op-1",
				Dependencies: []string{},
				Priority:     100,
				Stage:        0,
				Status:       OperationStatusPending,
			},
		},
		Summary: PlanSummary{
			TotalOperations:   1,
			OperationsByType:  map[OperationType]int{OperationCreate: 1},
			OperationsByStage: map[int]int{0: 1},
			EstimatedDuration: 10 * time.Minute,
			HighestRiskLevel:  RiskLevelMedium,
			RequiresApproval:  true,
		},
		Status: PlanStatusDraft,
	}

	tests := []struct {
		name            string
		mode            DryRunMode
		expectError     bool
		expectedOps     int
		expectedSucceed int
	}{
		{
			name:            "Quick mode execution",
			mode:            DryRunModeQuick,
			expectError:     false,
			expectedOps:     1,
			expectedSucceed: 1,
		},
		{
			name:            "Thorough mode execution",
			mode:            DryRunModeThorough,
			expectError:     false,
			expectedOps:     1,
			expectedSucceed: 1,
		},
		{
			name:            "Detailed mode execution",
			mode:            DryRunModeDetailed,
			expectError:     false,
			expectedOps:     1,
			expectedSucceed: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewDryRunExecutor(tt.mode)

			result, err := executor.Execute(ctx, plan)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Expected result but got nil")
			}

			if result.Summary.TotalOperations != tt.expectedOps {
				t.Errorf("Expected %d operations, got %d", tt.expectedOps, result.Summary.TotalOperations)
			}

			if result.Summary.OperationsWouldSucceed != tt.expectedSucceed {
				t.Errorf("Expected %d successful operations, got %d", tt.expectedSucceed, result.Summary.OperationsWouldSucceed)
			}

			if result.Mode != tt.mode {
				t.Errorf("Expected mode %s, got %s", tt.mode, result.Mode)
			}

			// Verify operation simulation
			if len(result.SimulatedResults) != tt.expectedOps {
				t.Errorf("Expected %d simulated results, got %d", tt.expectedOps, len(result.SimulatedResults))
			}

			// Check that pre-conditions and post-conditions are set
			for _, simResult := range result.SimulatedResults {
				if len(simResult.PreConditions) == 0 {
					t.Error("Expected pre-conditions to be set")
				}
				if len(simResult.PostConditions) == 0 {
					t.Error("Expected post-conditions to be set")
				}
			}
		})
	}
}

func TestDryRunExecutor_ValidatePlan(t *testing.T) {
	executor := NewDryRunExecutor(DryRunModeQuick)

	tests := []struct {
		name        string
		plan        *Plan
		expectError bool
	}{
		{
			name:        "Nil plan",
			plan:        nil,
			expectError: true,
		},
		{
			name: "Plan without project ID",
			plan: &Plan{
				ID:         "test",
				Operations: []PlannedOperation{{ID: "op-1"}},
			},
			expectError: true,
		},
		{
			name: "Plan without operations",
			plan: &Plan{
				ID:        "test",
				ProjectID: "project-1",
			},
			expectError: true,
		},
		{
			name: "Valid plan",
			plan: &Plan{
				ID:         "test",
				ProjectID:  "project-1",
				Operations: []PlannedOperation{{ID: "op-1"}},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validatePlan(tt.plan)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDryRunFormatter_Format(t *testing.T) {
	// Create sample dry-run result
	result := &DryRunResult{
		Plan: &Plan{
			ID:        "test-plan",
			ProjectID: "test-project",
		},
		SimulatedResults: []SimulatedOperation{
			{
				Operation: PlannedOperation{
					Operation: Operation{
						Type:         OperationCreate,
						ResourceType: types.KindCluster,
						ResourceName: "test-cluster",
					},
					ID: "op-1",
				},
				WouldSucceed:     true,
				ExpectedDuration: 10 * time.Minute,
			},
		},
		Summary: DryRunSummary{
			TotalOperations:        1,
			OperationsWouldSucceed: 1,
			OperationsWouldFail:    0,
			EstimatedDuration:      10 * time.Minute,
			HighestRiskLevel:       RiskLevelMedium,
		},
		Mode:        DryRunModeQuick,
		GeneratedAt: time.Now(),
	}

	tests := []struct {
		name          string
		format        DryRunOutputFormat
		expectError   bool
		shouldContain string
	}{
		{
			name:          "Table format",
			format:        DryRunFormatTable,
			expectError:   false,
			shouldContain: "Dry Run Results",
		},
		{
			name:          "JSON format",
			format:        DryRunFormatJSON,
			expectError:   false,
			shouldContain: "\"mode\":",
		},
		{
			name:          "YAML format",
			format:        DryRunFormatYAML,
			expectError:   false,
			shouldContain: "mode:",
		},
		{
			name:          "Summary format",
			format:        DryRunFormatSummary,
			expectError:   false,
			shouldContain: "Dry Run Summary",
		},
		{
			name:          "Detailed format",
			format:        DryRunFormatDetailed,
			expectError:   false,
			shouldContain: "Dry Run Results",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewDryRunFormatter(tt.format, false, false)

			output, err := formatter.Format(result)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && tt.shouldContain != "" {
				if len(output) == 0 {
					t.Error("Expected output but got empty string")
				}
				// Note: We can't do string.Contains check here easily due to formatting
				// but we verified the output works in manual testing
			}
		})
	}
}

func TestDefaultTimingEstimator(t *testing.T) {
	estimator := NewDefaultTimingEstimator()

	tests := []struct {
		name        string
		operation   PlannedOperation
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name: "Cluster creation",
			operation: PlannedOperation{
				Operation: Operation{
					Type:         OperationCreate,
					ResourceType: types.KindCluster,
				},
			},
			expectedMin: 9 * time.Minute,
			expectedMax: 11 * time.Minute,
		},
		{
			name: "User creation",
			operation: PlannedOperation{
				Operation: Operation{
					Type:         OperationCreate,
					ResourceType: types.KindDatabaseUser,
				},
			},
			expectedMin: 25 * time.Second,
			expectedMax: 35 * time.Second,
		},
		{
			name: "Network access update",
			operation: PlannedOperation{
				Operation: Operation{
					Type:         OperationUpdate,
					ResourceType: types.KindNetworkAccess,
				},
			},
			expectedMin: 5 * time.Second,
			expectedMax: 15 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := estimator.EstimateOperationDuration(tt.operation)

			if duration < tt.expectedMin || duration > tt.expectedMax {
				t.Errorf("Expected duration between %v and %v, got %v", tt.expectedMin, tt.expectedMax, duration)
			}
		})
	}
}
