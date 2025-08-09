package infra

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

// captureStdout temporarily redirects os.Stdout to capture output for assertions
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestDisplayPlanTable_UsesFormatterAndPrintsStages(t *testing.T) {
	plan := &apply.Plan{
		ID:        "plan-123",
		ProjectID: "507f1f77bcf86cd799439011",
		CreatedAt: time.Now(),
		Operations: []apply.PlannedOperation{
			{
				Operation: apply.Operation{
					Type:         apply.OperationCreate,
					ResourceType: types.KindCluster,
					ResourceName: "cluster-a",
					Impact:       &apply.OperationImpact{EstimatedDuration: 2 * time.Minute, RiskLevel: apply.RiskLevelLow},
				},
				ID:    "op-1",
				Stage: 0,
			},
			{
				Operation: apply.Operation{
					Type:         apply.OperationUpdate,
					ResourceType: types.KindDatabaseUser,
					ResourceName: "user-a",
					Impact:       &apply.OperationImpact{EstimatedDuration: 30 * time.Second, RiskLevel: apply.RiskLevelMedium},
				},
				ID:           "op-2",
				Dependencies: []string{"op-1"},
				Stage:        1,
			},
		},
		Summary: apply.PlanSummary{
			TotalOperations:       2,
			OperationsByType:      map[apply.OperationType]int{apply.OperationCreate: 1, apply.OperationUpdate: 1},
			OperationsByStage:     map[int]int{0: 1, 1: 1},
			EstimatedDuration:     150 * time.Second,
			HighestRiskLevel:      apply.RiskLevelMedium,
			DestructiveOperations: 0,
		},
	}

	output := captureStdout(t, func() {
		// No color for stable output
		_ = displayPlanTable(plan, &PlanOptions{NoColor: true})
	})

	// Basic assertions to ensure formatter output and stage headers are present
	assert.Contains(t, output, "Execution Plan  plan-123")
	assert.Contains(t, output, "Project         507f1f77bcf86cd799439011")
	assert.Contains(t, output, "Stage 0 (1 operations)")
	assert.Contains(t, output, "Stage 1 (1 operations)")
	assert.Contains(t, output, "Resource Type  Operation  Resource Name  Risk  Duration  Dependencies")
	assert.Contains(t, output, "Cluster        Create     cluster-a")
	assert.Contains(t, output, "DatabaseUser   Update     user-a")
	assert.Contains(t, output, "Summary             Value")
	assert.Contains(t, output, "Create operations   1")
	assert.Contains(t, output, "Update operations   1")
}

func TestValidatePlanOptions(t *testing.T) {
	tests := []struct {
		name       string
		files      []string
		output     string
		planMode   string
		outputFile string
		timeoutMs  int
		expectErr  bool
	}{
		{name: "missing files", files: []string{}, output: "table", planMode: "standard", expectErr: true},
		{name: "invalid output", files: []string{"config.yaml"}, output: "csv", planMode: "standard", expectErr: true},
		{name: "invalid plan mode", files: []string{"config.yaml"}, output: "json", planMode: "fast", expectErr: true},
		{name: "invalid timeout", files: []string{"config.yaml"}, output: "yaml", planMode: "quick", timeoutMs: -1, expectErr: true},
		{name: "invalid output file ext", files: []string{"config.yaml"}, output: "json", planMode: "detailed", outputFile: "plan.txt", expectErr: true},
		{name: "valid table", files: []string{"config.yaml"}, output: "table", planMode: "standard", expectErr: false},
		{name: "valid json", files: []string{"config.yaml"}, output: "json", planMode: "quick", outputFile: "plan.json", expectErr: false},
		{name: "valid yaml", files: []string{"config.yaml"}, output: "yaml", planMode: "detailed", outputFile: "plan.yaml", expectErr: false},
		{name: "valid summary", files: []string{"config.yaml"}, output: "summary", planMode: "standard", expectErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &PlanOptions{Files: tt.files, OutputFormat: tt.output, PlanMode: tt.planMode, OutputFile: tt.outputFile}
			if tt.timeoutMs != 0 {
				opts.Timeout = time.Duration(tt.timeoutMs) * time.Millisecond
			} else {
				opts.Timeout = 1 * time.Minute
			}
			err := validatePlanOptions(opts)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
