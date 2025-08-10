package apply

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DryRunOutputFormat represents the output format for dry-run results
type DryRunOutputFormat string

const (
	// DryRunFormatTable renders results as an ASCII table.
	DryRunFormatTable DryRunOutputFormat = "table"
	// DryRunFormatJSON renders results as JSON.
	DryRunFormatJSON DryRunOutputFormat = "json"
	// DryRunFormatYAML renders results as YAML.
	DryRunFormatYAML DryRunOutputFormat = "yaml"
	// DryRunFormatSummary renders a high-level summary only.
	DryRunFormatSummary DryRunOutputFormat = "summary"
	// DryRunFormatDetailed renders the table plus expanded details.
	DryRunFormatDetailed DryRunOutputFormat = "detailed"
)

// DryRunFormatter formats dry-run results for output
type DryRunFormatter struct {
	format      DryRunOutputFormat
	colorOutput bool
	verbose     bool
}

// NewDryRunFormatter creates a new dry-run formatter
func NewDryRunFormatter(format DryRunOutputFormat, colorOutput, verbose bool) *DryRunFormatter {
	return &DryRunFormatter{
		format:      format,
		colorOutput: colorOutput,
		verbose:     verbose,
	}
}

// Format formats the dry-run result according to the specified format
func (f *DryRunFormatter) Format(result *DryRunResult) (string, error) {
	switch f.format {
	case DryRunFormatTable:
		return f.formatTable(result), nil
	case DryRunFormatJSON:
		return f.formatJSON(result)
	case DryRunFormatYAML:
		return f.formatYAML(result)
	case DryRunFormatSummary:
		return f.formatSummary(result), nil
	case DryRunFormatDetailed:
		return f.formatDetailed(result), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", f.format)
	}
}

// formatTable formats the result as a table
func (f *DryRunFormatter) formatTable(result *DryRunResult) string {
	var output strings.Builder

	// Header
	f.writeHeader(&output, "Dry Run Results")

	// Summary section
	f.writeSummarySection(&output, result)

	// Operations table
	f.writeOperationsTable(&output, result)

	// Quota checks if any
	if len(result.QuotaChecks) > 0 {
		f.writeQuotaChecks(&output, result)
	}

	// Warnings and errors
	if len(result.Warnings) > 0 || len(result.Errors) > 0 {
		f.writeWarningsAndErrors(&output, result)
	}

	return output.String()
}

// formatJSON formats the result as JSON
func (f *DryRunFormatter) formatJSON(result *DryRunResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return string(data), nil
}

// formatYAML formats the result as YAML
func (f *DryRunFormatter) formatYAML(result *DryRunResult) (string, error) {
	data, err := yaml.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to YAML: %w", err)
	}
	return string(data), nil
}

// formatSummary formats a high-level summary only
func (f *DryRunFormatter) formatSummary(result *DryRunResult) string {
	var output strings.Builder

	f.writeHeader(&output, "Dry Run Summary")

	summary := result.Summary

	output.WriteString(fmt.Sprintf("Total Operations: %d\n", summary.TotalOperations))
	output.WriteString(fmt.Sprintf("Would Succeed: %s\n", f.colorize(fmt.Sprintf("%d", summary.OperationsWouldSucceed), "green")))

	if summary.OperationsWouldFail > 0 {
		output.WriteString(fmt.Sprintf("Would Fail: %s\n", f.colorize(fmt.Sprintf("%d", summary.OperationsWouldFail), "red")))
	}

	output.WriteString(fmt.Sprintf("Estimated Duration: %s\n", formatDuration(summary.EstimatedDuration)))
	output.WriteString(fmt.Sprintf("Highest Risk Level: %s\n", f.colorizeRiskLevel(string(summary.HighestRiskLevel))))

	if summary.QuotaViolations > 0 {
		output.WriteString(fmt.Sprintf("Quota Violations: %s\n", f.colorize(fmt.Sprintf("%d", summary.QuotaViolations), "red")))
	}

	if summary.ValidationErrors > 0 {
		output.WriteString(fmt.Sprintf("Validation Errors: %s\n", f.colorize(fmt.Sprintf("%d", summary.ValidationErrors), "red")))
	}

	if summary.Warnings > 0 {
		output.WriteString(fmt.Sprintf("Warnings: %s\n", f.colorize(fmt.Sprintf("%d", summary.Warnings), "yellow")))
	}

	return output.String()
}

// formatDetailed formats a detailed view with all information
func (f *DryRunFormatter) formatDetailed(result *DryRunResult) string {
	var output strings.Builder

	// Start with table format
	output.WriteString(f.formatTable(result))

	// Add detailed operation information
	if f.verbose {
		f.writeDetailedOperations(&output, result)
	}

	return output.String()
}

// Helper methods for formatting sections

func (f *DryRunFormatter) writeHeader(output *strings.Builder, title string) {
	separator := strings.Repeat("=", len(title))
	fmt.Fprintf(output, "%s\n%s\n%s\n\n", separator, title, separator)
}

func (f *DryRunFormatter) writeSummarySection(output *strings.Builder, result *DryRunResult) {
	output.WriteString("Summary:\n")
	fmt.Fprintf(output, "  Mode: %s\n", result.Mode)
	fmt.Fprintf(output, "  Total Operations: %d\n", result.Summary.TotalOperations)
	fmt.Fprintf(output, "  Would Succeed: %s\n", f.colorize(fmt.Sprintf("%d", result.Summary.OperationsWouldSucceed), "green"))

	if result.Summary.OperationsWouldFail > 0 {
		fmt.Fprintf(output, "  Would Fail: %s\n", f.colorize(fmt.Sprintf("%d", result.Summary.OperationsWouldFail), "red"))
	}

	fmt.Fprintf(output, "  Estimated Duration: %s\n", formatDuration(result.Summary.EstimatedDuration))
	fmt.Fprintf(output, "  Highest Risk Level: %s\n", f.colorizeRiskLevel(string(result.Summary.HighestRiskLevel)))
	output.WriteString("\n")
}

func (f *DryRunFormatter) writeOperationsTable(output *strings.Builder, result *DryRunResult) {
	if len(result.SimulatedResults) == 0 {
		output.WriteString("No operations to display.\n\n")
		return
	}

	output.WriteString("Operations:\n")

	// Table headers
	headers := []string{"Operation", "Resource", "Type", "Status", "Duration", "Risk"}
	f.writeTableRow(output, headers, true)
	f.writeTableSeparator(output, headers)

	// Table rows
	for _, simResult := range result.SimulatedResults {
		op := simResult.Operation
		status := "✓ Success"
		if !simResult.WouldSucceed {
			status = "✗ Fail"
		}

		riskLevel := "Low"
		if op.Impact != nil {
			riskLevel = string(op.Impact.RiskLevel)
		}

		row := []string{
			string(op.Type),
			fmt.Sprintf("%s/%s", op.ResourceType, op.ResourceName),
			string(op.ResourceType),
			f.colorizeStatus(status, simResult.WouldSucceed),
			formatDuration(simResult.ExpectedDuration),
			f.colorizeRiskLevel(riskLevel),
		}
		f.writeTableRow(output, row, false)
	}
	output.WriteString("\n")
}

func (f *DryRunFormatter) writeQuotaChecks(output *strings.Builder, result *DryRunResult) {
	output.WriteString("Quota Checks:\n")

	headers := []string{"Resource Type", "Current", "Requested", "Limit", "Status", "Usage %"}
	f.writeTableRow(output, headers, true)
	f.writeTableSeparator(output, headers)

	for _, quota := range result.QuotaChecks {
		status := "✓ OK"
		if quota.WouldExceed {
			status = "✗ Exceed"
		}

		row := []string{
			quota.ResourceType,
			fmt.Sprintf("%d", quota.CurrentUsage),
			fmt.Sprintf("%d", quota.RequestedAdd),
			fmt.Sprintf("%d", quota.Limit),
			f.colorizeStatus(status, !quota.WouldExceed),
			fmt.Sprintf("%.1f%%", quota.Percentage),
		}
		f.writeTableRow(output, row, false)
	}
	output.WriteString("\n")
}

func (f *DryRunFormatter) writeWarningsAndErrors(output *strings.Builder, result *DryRunResult) {
	if len(result.Errors) > 0 {
		output.WriteString("Errors:\n")
		for _, err := range result.Errors {
			fmt.Fprintf(output, "  • %s\n", f.colorize(err, "red"))
		}
		output.WriteString("\n")
	}

	if len(result.Warnings) > 0 {
		output.WriteString("Warnings:\n")
		for _, warning := range result.Warnings {
			fmt.Fprintf(output, "  • %s\n", f.colorize(warning, "yellow"))
		}
		output.WriteString("\n")
	}
}

func (f *DryRunFormatter) writeDetailedOperations(output *strings.Builder, result *DryRunResult) {
	output.WriteString("Detailed Operation Information:\n")
	output.WriteString(strings.Repeat("-", 50) + "\n\n")

	for i, simResult := range result.SimulatedResults {
		op := simResult.Operation

		fmt.Fprintf(output, "Operation %d: %s %s/%s\n", i+1, op.Type, op.ResourceType, op.ResourceName)
		fmt.Fprintf(output, "  ID: %s\n", op.ID)
		fmt.Fprintf(output, "  Would Succeed: %t\n", simResult.WouldSucceed)
		fmt.Fprintf(output, "  Expected Duration: %s\n", formatDuration(simResult.ExpectedDuration))

		if len(simResult.Dependencies) > 0 {
			fmt.Fprintf(output, "  Dependencies: %s\n", strings.Join(simResult.Dependencies, ", "))
		}

		// Pre-conditions
		if len(simResult.PreConditions) > 0 {
			output.WriteString("  Pre-conditions:\n")
			for _, cond := range simResult.PreConditions {
				status := "✓"
				if !cond.Satisfied {
					status = "✗"
				}
				fmt.Fprintf(output, "    %s %s\n", status, cond.Description)
				if cond.Reason != "" {
					fmt.Fprintf(output, "      Reason: %s\n", cond.Reason)
				}
			}
		}

		// Post-conditions
		if len(simResult.PostConditions) > 0 {
			output.WriteString("  Post-conditions:\n")
			for _, cond := range simResult.PostConditions {
				fmt.Fprintf(output, "    • %s: %s\n", cond.Description, cond.ExpectedValue)
				if cond.Impact != "" {
					fmt.Fprintf(output, "      Impact: %s\n", cond.Impact)
				}
			}
		}

		// Errors and warnings
		if len(simResult.Errors) > 0 {
			output.WriteString("  Errors:\n")
			for _, err := range simResult.Errors {
				fmt.Fprintf(output, "    • %s\n", f.colorize(err, "red"))
			}
		}

		if len(simResult.Warnings) > 0 {
			output.WriteString("  Warnings:\n")
			for _, warning := range simResult.Warnings {
				fmt.Fprintf(output, "    • %s\n", f.colorize(warning, "yellow"))
			}
		}

		output.WriteString("\n")
	}
}

// Table formatting helpers

func (f *DryRunFormatter) writeTableRow(output *strings.Builder, columns []string, isHeader bool) {
	// Simple table formatting - in a real implementation, you might want proper column alignment
	row := "| " + strings.Join(columns, " | ") + " |"
	if isHeader && f.colorOutput {
		row = f.colorize(row, "bold")
	}
	output.WriteString(row + "\n")
}

func (f *DryRunFormatter) writeTableSeparator(output *strings.Builder, columns []string) {
	separator := "|"
	for range columns {
		separator += "---|"
	}
	output.WriteString(separator + "\n")
}

// Color and formatting helpers

func (f *DryRunFormatter) colorize(text, color string) string {
	if !f.colorOutput {
		return text
	}

	colors := map[string]string{
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"bold":   "\033[1m",
		"reset":  "\033[0m",
	}

	if colorCode, exists := colors[color]; exists {
		return colorCode + text + colors["reset"]
	}
	return text
}

func (f *DryRunFormatter) colorizeStatus(status string, success bool) string {
	if success {
		return f.colorize(status, "green")
	}
	return f.colorize(status, "red")
}

func (f *DryRunFormatter) colorizeRiskLevel(level string) string {
	switch level {
	case "low":
		return f.colorize(level, "green")
	case "medium":
		return f.colorize(level, "yellow")
	case "high", "critical":
		return f.colorize(level, "red")
	default:
		return level
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}
