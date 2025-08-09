package apply

import (
	"strings"
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestDiffFormatter_FormatTable(t *testing.T) {
	formatter := NewDiffFormatter()

	diff := createTestDiff()

	output := formatter.FormatTable(diff)

	if !strings.Contains(output, "Plan for project:") {
		t.Error("Expected table output to contain project header")
	}

	if !strings.Contains(output, "CREATE") {
		t.Error("Expected table output to contain CREATE operation")
	}

	if !strings.Contains(output, "test-cluster") {
		t.Error("Expected table output to contain cluster name")
	}
}

func TestDiffFormatter_FormatUnified(t *testing.T) {
	formatter := NewDiffFormatter()

	diff := createTestDiff()

	output := formatter.FormatUnified(diff)

	if !strings.Contains(output, "diff --git") {
		t.Error("Expected unified output to contain git diff header")
	}

	if !strings.Contains(output, "+++") {
		t.Error("Expected unified output to contain +++ for CREATE")
	}
}

func TestDiffFormatter_FormatJSON(t *testing.T) {
	formatter := NewDiffFormatter()

	diff := createTestDiff()

	output, err := formatter.FormatJSON(diff)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, `"projectId"`) {
		t.Error("Expected JSON output to contain projectId field")
	}

	if !strings.Contains(output, `"operations"`) {
		t.Error("Expected JSON output to contain operations field")
	}
}

func TestDiffFormatter_FormatYAML(t *testing.T) {
	formatter := NewDiffFormatter()

	diff := createTestDiff()

	output, err := formatter.FormatYAML(diff)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "projectid:") && !strings.Contains(output, "projectId:") {
		t.Error("Expected YAML output to contain projectId field")
	}

	if !strings.Contains(output, "operations:") {
		t.Error("Expected YAML output to contain operations field")
	}
}

func TestDiffFormatter_FormatSummary(t *testing.T) {
	formatter := NewDiffFormatter()

	diff := createTestDiff()

	output := formatter.FormatSummary(diff)

	if !strings.Contains(output, "Diff Summary") {
		t.Error("Expected summary output to contain header")
	}

	if !strings.Contains(output, "to create") {
		t.Error("Expected summary output to contain create count")
	}
}

func TestDiffFormatter_Format_AllFormats(t *testing.T) {
	formatter := NewDiffFormatter()
	diff := createTestDiff()

	formats := []string{"table", "unified", "json", "yaml", "summary"}

	for _, format := range formats {
		opts := &FormatOptions{
			Format:    format,
			UseColors: false, // Disable colors for testing
			Verbose:   false,
		}

		output, err := formatter.Format(diff, opts)
		if err != nil {
			t.Errorf("Error formatting as %s: %v", format, err)
			continue
		}

		if len(output) == 0 {
			t.Errorf("Empty output for format %s", format)
		}
	}
}

func TestDiffFormatter_Format_UnsupportedFormat(t *testing.T) {
	formatter := NewDiffFormatter()
	diff := createTestDiff()

	opts := &FormatOptions{
		Format: "unsupported",
	}

	_, err := formatter.Format(diff, opts)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}

	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("Expected 'unsupported format' error, got: %v", err)
	}
}

func TestDiffFormatter_Colorization(t *testing.T) {
	formatter := NewDiffFormatter()
	formatter.UseColors = true

	// Test operation colorization
	create := formatter.colorizeOperation("Create")
	if !strings.Contains(create, "\033[32m") { // Green
		t.Error("Expected CREATE to be colored green")
	}

	update := formatter.colorizeOperation("Update")
	if !strings.Contains(update, "\033[33m") { // Yellow
		t.Error("Expected UPDATE to be colored yellow")
	}

	delete := formatter.colorizeOperation("Delete")
	if !strings.Contains(delete, "\033[31m") { // Red
		t.Error("Expected DELETE to be colored red")
	}

	// Test risk level colorization
	critical := formatter.colorizeRiskLevel("critical")
	if !strings.Contains(critical, "\033[35m") { // Purple
		t.Error("Expected CRITICAL to be colored purple")
	}
}

func TestDiffFormatter_NoColors(t *testing.T) {
	formatter := NewDiffFormatter()
	formatter.UseColors = false

	create := formatter.colorizeOperation("Create")
	if strings.Contains(create, "\033[") {
		t.Error("Expected no color codes when UseColors is false")
	}
}

func TestFormatDiffStats(t *testing.T) {
	summary := DiffSummary{
		CreateOperations: 2,
		UpdateOperations: 1,
		DeleteOperations: 1,
	}

	output := FormatDiffStats(summary)
	expected := "Plan: 2 to create, 1 to update, 1 to delete"

	if output != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output)
	}
}

// Helper function to create a test diff for testing
func createTestDiff() *Diff {
	return &Diff{
		ProjectID: "test-project",
		Operations: []Operation{
			{
				Type:         OperationCreate,
				ResourceType: types.KindCluster,
				ResourceName: "test-cluster",
				Desired: &types.ClusterManifest{
					Metadata: types.ResourceMetadata{Name: "test-cluster"},
					Spec: types.ClusterSpec{
						TierType:       "M10",
						MongoDBVersion: "7.0",
					},
				},
				Impact: &OperationImpact{
					EstimatedDuration: time.Minute * 15,
					RiskLevel:         RiskLevelMedium,
					Warnings:          []string{"Cluster creation will incur costs"},
				},
			},
		},
		Summary: DiffSummary{
			TotalOperations:   1,
			CreateOperations:  1,
			EstimatedDuration: time.Minute * 15,
			HighestRiskLevel:  RiskLevelMedium,
		},
		GeneratedAt: time.Now().UTC(),
	}
}
