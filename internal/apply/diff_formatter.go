package apply

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DiffFormatter provides various output formats for diffs
type DiffFormatter struct {
	UseColors    bool
	ShowNoChange bool
	Verbose      bool
}

// NewDiffFormatter creates a new diff formatter with default settings
func NewDiffFormatter() *DiffFormatter {
	return &DiffFormatter{
		UseColors:    true,
		ShowNoChange: false,
		Verbose:      false,
	}
}

// FormatOptions configures the formatting output
type FormatOptions struct {
	Format       string // "table", "unified", "json", "yaml", "summary"
	UseColors    bool
	ShowNoChange bool
	Verbose      bool
}

// DefaultFormatOptions returns sensible default format options
func DefaultFormatOptions() *FormatOptions {
	return &FormatOptions{
		Format:       "table",
		UseColors:    true,
		ShowNoChange: false,
		Verbose:      false,
	}
}

// Format formats the diff according to the specified options
func (f *DiffFormatter) Format(diff *Diff, opts *FormatOptions) (string, error) {
	if opts == nil {
		opts = DefaultFormatOptions()
	}

	f.UseColors = opts.UseColors
	f.ShowNoChange = opts.ShowNoChange
	f.Verbose = opts.Verbose

	switch opts.Format {
	case "table":
		return f.FormatTable(diff), nil
	case "unified":
		return f.FormatUnified(diff), nil
	case "json":
		return f.FormatJSON(diff)
	case "yaml":
		return f.FormatYAML(diff)
	case "summary":
		return f.FormatSummary(diff), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", opts.Format)
	}
}

// FormatTable formats the diff as a colored table
func (f *DiffFormatter) FormatTable(diff *Diff) string {
	var output strings.Builder

	// Header
	output.WriteString(f.colorize("Plan for project: "+diff.ProjectID, ColorBold))
	output.WriteString("\n\n")

	// Summary
	summary := diff.Summary
	output.WriteString(f.formatSummaryLine(summary))
	output.WriteString("\n\n")

	// Operations table
	if len(diff.Operations) > 0 {
		output.WriteString(f.formatOperationsTable(diff.Operations))
	} else {
		output.WriteString(f.colorize("No changes detected.", ColorGreen))
		output.WriteString("\n")
	}

	return output.String()
}

// FormatUnified formats the diff in unified diff format (similar to git diff)
func (f *DiffFormatter) FormatUnified(diff *Diff) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", diff.ProjectID, diff.ProjectID))
	output.WriteString(fmt.Sprintf("--- a/%s\n", diff.ProjectID))
	output.WriteString(fmt.Sprintf("+++ b/%s\n", diff.ProjectID))
	output.WriteString(fmt.Sprintf("@@ Generated at %s @@\n", diff.GeneratedAt.Format(time.RFC3339)))
	output.WriteString("\n")

	for _, op := range diff.Operations {
		if op.Type == OperationNoChange && !f.ShowNoChange {
			continue
		}

		output.WriteString(f.formatUnifiedOperation(op))
		output.WriteString("\n")
	}

	return output.String()
}

// FormatJSON formats the diff as JSON
func (f *DiffFormatter) FormatJSON(diff *Diff) (string, error) {
	data, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal diff to JSON: %w", err)
	}
	return string(data), nil
}

// FormatYAML formats the diff as YAML
func (f *DiffFormatter) FormatYAML(diff *Diff) (string, error) {
	data, err := yaml.Marshal(diff)
	if err != nil {
		return "", fmt.Errorf("failed to marshal diff to YAML: %w", err)
	}
	return string(data), nil
}

// FormatSummary formats just the summary statistics
func (f *DiffFormatter) FormatSummary(diff *Diff) string {
	var output strings.Builder

	summary := diff.Summary
	output.WriteString(f.colorize("Diff Summary", ColorBold))
	output.WriteString("\n")
	output.WriteString(strings.Repeat("=", 50))
	output.WriteString("\n")

	output.WriteString(f.formatSummaryLine(summary))
	output.WriteString("\n")

	if summary.DestructiveOperations > 0 {
		output.WriteString(f.colorize(fmt.Sprintf("⚠️  %d destructive operations detected!",
			summary.DestructiveOperations), ColorRed))
		output.WriteString("\n")
	}

	if summary.EstimatedDuration > 0 {
		output.WriteString(fmt.Sprintf("Estimated execution time: %s\n",
			f.formatDuration(summary.EstimatedDuration)))
	}

	output.WriteString(fmt.Sprintf("Highest risk level: %s\n",
		f.colorizeRiskLevel(string(summary.HighestRiskLevel))))

	return output.String()
}

// Helper methods for formatting

func (f *DiffFormatter) formatSummaryLine(summary DiffSummary) string {
	parts := []string{}

	if summary.CreateOperations > 0 {
		parts = append(parts, f.colorize(fmt.Sprintf("%d to create", summary.CreateOperations), ColorGreen))
	}
	if summary.UpdateOperations > 0 {
		parts = append(parts, f.colorize(fmt.Sprintf("%d to update", summary.UpdateOperations), ColorYellow))
	}
	if summary.DeleteOperations > 0 {
		parts = append(parts, f.colorize(fmt.Sprintf("%d to delete", summary.DeleteOperations), ColorRed))
	}
	if summary.NoChangeOperations > 0 && f.ShowNoChange {
		parts = append(parts, f.colorize(fmt.Sprintf("%d unchanged", summary.NoChangeOperations), ColorGray))
	}

	if len(parts) == 0 {
		return "No changes"
	}

	return strings.Join(parts, ", ")
}

func (f *DiffFormatter) formatOperationsTable(operations []Operation) string {
	var output strings.Builder

	// Sort operations by type and name for consistent output
	sorted := make([]Operation, len(operations))
	copy(sorted, operations)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Type != sorted[j].Type {
			return operationTypePriority(sorted[i].Type) < operationTypePriority(sorted[j].Type)
		}
		return sorted[i].ResourceName < sorted[j].ResourceName
	})

	// Table header
	output.WriteString(f.formatTableHeader())
	output.WriteString("\n")

	// Table rows
	for _, op := range sorted {
		if op.Type == OperationNoChange && !f.ShowNoChange {
			continue
		}
		output.WriteString(f.formatTableRow(op))
		output.WriteString("\n")

		// Show field changes in verbose mode
		if f.Verbose && op.Type == OperationUpdate && len(op.FieldChanges) > 0 {
			output.WriteString(f.formatFieldChanges(op.FieldChanges))
		}

		// Show warnings
		if op.Impact != nil && len(op.Impact.Warnings) > 0 {
			output.WriteString(f.formatWarnings(op.Impact.Warnings))
		}
	}

	return output.String()
}

func (f *DiffFormatter) formatTableHeader() string {
	header := fmt.Sprintf("%-8s %-15s %-20s %-10s %-15s",
		"ACTION", "TYPE", "NAME", "RISK", "DURATION")
	return f.colorize(header, ColorBold)
}

func (f *DiffFormatter) formatTableRow(op Operation) string {
	action := f.colorizeOperation(string(op.Type))
	resourceType := string(op.ResourceType)
	name := op.ResourceName

	var risk, duration string
	if op.Impact != nil {
		risk = f.colorizeRiskLevel(string(op.Impact.RiskLevel))
		duration = f.formatDuration(op.Impact.EstimatedDuration)
	} else {
		risk = "-"
		duration = "-"
	}

	return fmt.Sprintf("%-8s %-15s %-20s %-10s %-15s",
		action, resourceType, name, risk, duration)
}

func (f *DiffFormatter) formatFieldChanges(changes []FieldChange) string {
	var output strings.Builder

	for _, change := range changes {
		prefix := "    "
		switch change.Type {
		case ChangeTypeAdd:
			line := fmt.Sprintf("%s+ %s: %v", prefix, change.Path, change.NewValue)
			output.WriteString(f.colorize(line, ColorGreen))
		case ChangeTypeRemove:
			line := fmt.Sprintf("%s- %s: %v", prefix, change.Path, change.OldValue)
			output.WriteString(f.colorize(line, ColorRed))
		case ChangeTypeModify:
			line := fmt.Sprintf("%s~ %s: %v → %v", prefix, change.Path, change.OldValue, change.NewValue)
			output.WriteString(f.colorize(line, ColorYellow))
		}
		output.WriteString("\n")
	}

	return output.String()
}

func (f *DiffFormatter) formatWarnings(warnings []string) string {
	var output strings.Builder

	for _, warning := range warnings {
		line := fmt.Sprintf("    ⚠️  %s", warning)
		output.WriteString(f.colorize(line, ColorYellow))
		output.WriteString("\n")
	}

	return output.String()
}

func (f *DiffFormatter) formatUnifiedOperation(op Operation) string {
	var output strings.Builder

	switch op.Type {
	case OperationCreate:
		output.WriteString(f.colorize(fmt.Sprintf("+++ %s/%s", op.ResourceType, op.ResourceName), ColorGreen))
	case OperationDelete:
		output.WriteString(f.colorize(fmt.Sprintf("--- %s/%s", op.ResourceType, op.ResourceName), ColorRed))
	case OperationUpdate:
		output.WriteString(f.colorize(fmt.Sprintf("~~~ %s/%s", op.ResourceType, op.ResourceName), ColorYellow))
		// Show field changes
		for _, change := range op.FieldChanges {
			output.WriteString("\n")
			switch change.Type {
			case ChangeTypeAdd:
				output.WriteString(f.colorize(fmt.Sprintf("+    %s: %v", change.Path, change.NewValue), ColorGreen))
			case ChangeTypeRemove:
				output.WriteString(f.colorize(fmt.Sprintf("-    %s: %v", change.Path, change.OldValue), ColorRed))
			case ChangeTypeModify:
				output.WriteString(f.colorize(fmt.Sprintf("-    %s: %v", change.Path, change.OldValue), ColorRed))
				output.WriteString("\n")
				output.WriteString(f.colorize(fmt.Sprintf("+    %s: %v", change.Path, change.NewValue), ColorGreen))
			}
		}
	case OperationNoChange:
		output.WriteString(f.colorize(fmt.Sprintf("=== %s/%s", op.ResourceType, op.ResourceName), ColorGray))
	}

	return output.String()
}

// Color definitions and utilities

type Color string

const (
	ColorReset  Color = "\033[0m"
	ColorBold   Color = "\033[1m"
	ColorRed    Color = "\033[31m"
	ColorGreen  Color = "\033[32m"
	ColorYellow Color = "\033[33m"
	ColorBlue   Color = "\033[34m"
	ColorPurple Color = "\033[35m"
	ColorCyan   Color = "\033[36m"
	ColorGray   Color = "\033[90m"
)

func (f *DiffFormatter) colorize(text string, color Color) string {
	if !f.UseColors {
		return text
	}
	return string(color) + text + string(ColorReset)
}

func (f *DiffFormatter) colorizeOperation(opType string) string {
	switch OperationType(opType) {
	case OperationCreate:
		return f.colorize("CREATE", ColorGreen)
	case OperationUpdate:
		return f.colorize("UPDATE", ColorYellow)
	case OperationDelete:
		return f.colorize("DELETE", ColorRed)
	case OperationNoChange:
		return f.colorize("NO-CHANGE", ColorGray)
	default:
		return opType
	}
}

func (f *DiffFormatter) colorizeRiskLevel(risk string) string {
	switch RiskLevel(risk) {
	case RiskLevelLow:
		return f.colorize("LOW", ColorGreen)
	case RiskLevelMedium:
		return f.colorize("MEDIUM", ColorYellow)
	case RiskLevelHigh:
		return f.colorize("HIGH", ColorRed)
	case RiskLevelCritical:
		return f.colorize("CRITICAL", ColorPurple)
	default:
		return risk
	}
}

func (f *DiffFormatter) formatDuration(d time.Duration) string {
	if d == 0 {
		return "instant"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// Helper functions

func operationTypePriority(opType OperationType) int {
	switch opType {
	case OperationCreate:
		return 1
	case OperationUpdate:
		return 2
	case OperationDelete:
		return 3
	case OperationNoChange:
		return 4
	default:
		return 5
	}
}

// FormatDiffStats formats just the numerical statistics
func FormatDiffStats(summary DiffSummary) string {
	return fmt.Sprintf("Plan: %d to create, %d to update, %d to delete",
		summary.CreateOperations,
		summary.UpdateOperations,
		summary.DeleteOperations)
}
