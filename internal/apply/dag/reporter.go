package dag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ReportFormat defines the format for reports
type ReportFormat string

const (
	// ReportFormatText generates plain text reports
	ReportFormatText ReportFormat = "text"
	
	// ReportFormatMarkdown generates Markdown reports
	ReportFormatMarkdown ReportFormat = "markdown"
	
	// ReportFormatJSON generates JSON reports
	ReportFormatJSON ReportFormat = "json"
	
	// ReportFormatYAML generates YAML reports
	ReportFormatYAML ReportFormat = "yaml"
)

// Reporter generates comprehensive reports
type Reporter struct {
	format ReportFormat
}

// NewReporter creates a new reporter
func NewReporter(format ReportFormat) *Reporter {
	return &Reporter{format: format}
}

// GenerateDependencyReport generates a comprehensive dependency analysis report
func (r *Reporter) GenerateDependencyReport(analysis *AnalysisResult) (string, error) {
	if analysis == nil {
		return "", fmt.Errorf("analysis result cannot be nil")
	}
	
	switch r.format {
	case ReportFormatText:
		return r.generateTextDependencyReport(analysis)
	case ReportFormatMarkdown:
		return r.generateMarkdownDependencyReport(analysis)
	case ReportFormatJSON:
		data, err := json.MarshalIndent(analysis, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported report format: %s", r.format)
	}
}

// generateTextDependencyReport generates a plain text dependency report
func (r *Reporter) generateTextDependencyReport(analysis *AnalysisResult) (string, error) {
	var buf bytes.Buffer
	
	// Header
	buf.WriteString("Dependency Analysis Report\n")
	buf.WriteString(strings.Repeat("=", 70) + "\n\n")
	buf.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))
	
	// Overview
	buf.WriteString("OVERVIEW\n")
	buf.WriteString(strings.Repeat("-", 70) + "\n")
	buf.WriteString(fmt.Sprintf("Total Operations:      %d\n", analysis.NodeCount))
	buf.WriteString(fmt.Sprintf("Dependencies:          %d\n", analysis.EdgeCount))
	buf.WriteString(fmt.Sprintf("Dependency Levels:     %d\n", analysis.MaxLevel+1))
	buf.WriteString(fmt.Sprintf("Has Cycles:            %v\n", analysis.HasCycles))
	
	if analysis.ParallelizationFactor > 0 {
		buf.WriteString(fmt.Sprintf("Parallelization Factor: %.2fx\n", analysis.ParallelizationFactor))
	}
	
	buf.WriteString("\n")
	
	// Critical Path
	if len(analysis.CriticalPath) > 0 {
		buf.WriteString("CRITICAL PATH\n")
		buf.WriteString(strings.Repeat("-", 70) + "\n")
		buf.WriteString(fmt.Sprintf("Length:   %d operations\n", len(analysis.CriticalPath)))
		buf.WriteString(fmt.Sprintf("Duration: %v\n", analysis.CriticalPathDuration))
		buf.WriteString("\nOperations on Critical Path:\n")
		for i, nodeID := range analysis.CriticalPath {
			buf.WriteString(fmt.Sprintf("  %d. %s\n", i+1, nodeID))
		}
		buf.WriteString("\n")
	}
	
	// Bottlenecks
	if len(analysis.Bottlenecks) > 0 {
		buf.WriteString("BOTTLENECKS\n")
		buf.WriteString(strings.Repeat("-", 70) + "\n")
		for i, bottleneck := range analysis.Bottlenecks {
			buf.WriteString(fmt.Sprintf("\n%d. %s (%s)\n", i+1, bottleneck.NodeID, bottleneck.NodeName))
			buf.WriteString(fmt.Sprintf("   Blocks:     %d operations (%.1f%% impact)\n", 
				bottleneck.BlockedCount, bottleneck.Impact*100))
			if bottleneck.Reason != "" {
				buf.WriteString(fmt.Sprintf("   Reason:     %s\n", bottleneck.Reason))
			}
			if bottleneck.Mitigation != "" {
				buf.WriteString(fmt.Sprintf("   Mitigation: %s\n", bottleneck.Mitigation))
			}
		}
		buf.WriteString("\n")
	}
	
	// Risk Analysis
	if analysis.RiskAnalysis != nil {
		buf.WriteString("RISK ANALYSIS\n")
		buf.WriteString(strings.Repeat("-", 70) + "\n")
		buf.WriteString(fmt.Sprintf("Total Risk Score:      %.1f\n", analysis.RiskAnalysis.TotalRiskScore))
		buf.WriteString(fmt.Sprintf("Average Risk Level:    %s\n", analysis.RiskAnalysis.AverageRiskLevel))
		buf.WriteString(fmt.Sprintf("High-Risk Operations:  %d\n", len(analysis.RiskAnalysis.HighRiskOperations)))
		buf.WriteString(fmt.Sprintf("Critical-Risk Ops:     %d (on critical path)\n", 
			len(analysis.RiskAnalysis.CriticalRiskOperations)))
		
		buf.WriteString("\nRisk Distribution:\n")
		for level, count := range analysis.RiskAnalysis.RiskByLevel {
			buf.WriteString(fmt.Sprintf("  %-10s: %d operations\n", level, count))
		}
		buf.WriteString("\n")
	}
	
	// Optimization Suggestions
	if len(analysis.Suggestions) > 0 {
		buf.WriteString("OPTIMIZATION SUGGESTIONS\n")
		buf.WriteString(strings.Repeat("-", 70) + "\n")
		for i, suggestion := range analysis.Suggestions {
			buf.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, suggestion))
		}
		buf.WriteString("\n")
	}
	
	return buf.String(), nil
}

// generateMarkdownDependencyReport generates a Markdown dependency report
func (r *Reporter) generateMarkdownDependencyReport(analysis *AnalysisResult) (string, error) {
	var buf bytes.Buffer
	
	// Header
	buf.WriteString("# Dependency Analysis Report\n\n")
	buf.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))
	
	// Overview
	buf.WriteString("## Overview\n\n")
	buf.WriteString("| Metric | Value |\n")
	buf.WriteString("|--------|-------|\n")
	buf.WriteString(fmt.Sprintf("| Total Operations | %d |\n", analysis.NodeCount))
	buf.WriteString(fmt.Sprintf("| Dependencies | %d |\n", analysis.EdgeCount))
	buf.WriteString(fmt.Sprintf("| Dependency Levels | %d |\n", analysis.MaxLevel+1))
	buf.WriteString(fmt.Sprintf("| Has Cycles | %v |\n", analysis.HasCycles))
	
	if analysis.ParallelizationFactor > 0 {
		buf.WriteString(fmt.Sprintf("| Parallelization Factor | %.2fx |\n", analysis.ParallelizationFactor))
	}
	
	buf.WriteString("\n")
	
	// Critical Path
	if len(analysis.CriticalPath) > 0 {
		buf.WriteString("## Critical Path\n\n")
		buf.WriteString(fmt.Sprintf("**Length:** %d operations  \n", len(analysis.CriticalPath)))
		buf.WriteString(fmt.Sprintf("**Duration:** %v\n\n", analysis.CriticalPathDuration))
		buf.WriteString("### Operations on Critical Path\n\n")
		for i, nodeID := range analysis.CriticalPath {
			buf.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, nodeID))
		}
		buf.WriteString("\n")
	}
	
	// Bottlenecks
	if len(analysis.Bottlenecks) > 0 {
		buf.WriteString("## Bottlenecks\n\n")
		for i, bottleneck := range analysis.Bottlenecks {
			buf.WriteString(fmt.Sprintf("### %d. %s (%s)\n\n", i+1, bottleneck.NodeID, bottleneck.NodeName))
			buf.WriteString(fmt.Sprintf("- **Blocks:** %d operations (%.1f%% impact)\n", 
				bottleneck.BlockedCount, bottleneck.Impact*100))
			if bottleneck.Reason != "" {
				buf.WriteString(fmt.Sprintf("- **Reason:** %s\n", bottleneck.Reason))
			}
			if bottleneck.Mitigation != "" {
				buf.WriteString(fmt.Sprintf("- **Mitigation:** %s\n", bottleneck.Mitigation))
			}
			buf.WriteString("\n")
		}
	}
	
	// Risk Analysis
	if analysis.RiskAnalysis != nil {
		buf.WriteString("## Risk Analysis\n\n")
		buf.WriteString(fmt.Sprintf("- **Total Risk Score:** %.1f\n", analysis.RiskAnalysis.TotalRiskScore))
		buf.WriteString(fmt.Sprintf("- **Average Risk Level:** %s\n", analysis.RiskAnalysis.AverageRiskLevel))
		buf.WriteString(fmt.Sprintf("- **High-Risk Operations:** %d\n", len(analysis.RiskAnalysis.HighRiskOperations)))
		buf.WriteString(fmt.Sprintf("- **Critical-Risk Operations:** %d (on critical path)\n\n", 
			len(analysis.RiskAnalysis.CriticalRiskOperations)))
		
		buf.WriteString("### Risk Distribution\n\n")
		buf.WriteString("| Risk Level | Count |\n")
		buf.WriteString("|------------|-------|\n")
		for level, count := range analysis.RiskAnalysis.RiskByLevel {
			buf.WriteString(fmt.Sprintf("| %s | %d |\n", level, count))
		}
		buf.WriteString("\n")
	}
	
	// Optimization Suggestions
	if len(analysis.Suggestions) > 0 {
		buf.WriteString("## Optimization Suggestions\n\n")
		for i, suggestion := range analysis.Suggestions {
			buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, suggestion))
		}
		buf.WriteString("\n")
	}
	
	return buf.String(), nil
}

// GenerateScheduleReport generates a schedule analysis report
func (r *Reporter) GenerateScheduleReport(schedule *Schedule, analysis *ScheduleAnalysis) (string, error) {
	if schedule == nil {
		return "", fmt.Errorf("schedule cannot be nil")
	}
	
	switch r.format {
	case ReportFormatText:
		return r.generateTextScheduleReport(schedule, analysis)
	case ReportFormatMarkdown:
		return r.generateMarkdownScheduleReport(schedule, analysis)
	case ReportFormatJSON:
		data := struct {
			Schedule *Schedule         `json:"schedule"`
			Analysis *ScheduleAnalysis `json:"analysis"`
		}{
			Schedule: schedule,
			Analysis: analysis,
		}
		result, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return "", err
		}
		return string(result), nil
	default:
		return "", fmt.Errorf("unsupported report format: %s", r.format)
	}
}

// generateTextScheduleReport generates a plain text schedule report
func (r *Reporter) generateTextScheduleReport(schedule *Schedule, analysis *ScheduleAnalysis) (string, error) {
	var buf bytes.Buffer
	
	// Header
	buf.WriteString("Schedule Analysis Report\n")
	buf.WriteString(strings.Repeat("=", 70) + "\n\n")
	buf.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))
	
	// Strategy
	buf.WriteString(fmt.Sprintf("Strategy:  %s\n", schedule.Strategy))
	buf.WriteString(fmt.Sprintf("Duration:  %v\n", schedule.EstimatedDuration))
	buf.WriteString("\n")
	
	// Metrics
	if analysis != nil {
		buf.WriteString("METRICS\n")
		buf.WriteString(strings.Repeat("-", 70) + "\n")
		buf.WriteString(fmt.Sprintf("Total Operations:       %d\n", analysis.TotalOperations))
		buf.WriteString(fmt.Sprintf("Total Stages:           %d\n", analysis.TotalStages))
		buf.WriteString(fmt.Sprintf("Avg Stage Size:         %.2f operations\n", analysis.AvgStageSize))
		buf.WriteString(fmt.Sprintf("Max Stage Size:         %d operations\n", analysis.MaxStageSize))
		buf.WriteString(fmt.Sprintf("Min Stage Size:         %d operations\n", analysis.MinStageSize))
		buf.WriteString(fmt.Sprintf("Parallelization Factor: %.2fx\n", analysis.ParallelizationFactor))
		buf.WriteString(fmt.Sprintf("Efficiency:             %.1f%%\n", analysis.Efficiency*100))
		buf.WriteString("\n")
	}
	
	// Stages
	buf.WriteString("EXECUTION STAGES\n")
	buf.WriteString(strings.Repeat("-", 70) + "\n\n")
	
	for i, stage := range schedule.Stages {
		// Calculate stage duration
		stageDuration := time.Duration(0)
		for _, node := range stage {
			if node.Properties.EstimatedDuration > stageDuration {
				stageDuration = node.Properties.EstimatedDuration
			}
		}
		
		buf.WriteString(fmt.Sprintf("Stage %d: %d operations (%v duration)\n", 
			i+1, len(stage), stageDuration))
		
		// List operations
		for j, node := range stage {
			marker := " "
			if node.IsCritical {
				marker = "*"
			}
			buf.WriteString(fmt.Sprintf("  %s %d. %s (%v)\n", 
				marker, j+1, node.Name, node.Properties.EstimatedDuration))
		}
		buf.WriteString("\n")
	}
	
	return buf.String(), nil
}

// generateMarkdownScheduleReport generates a Markdown schedule report
func (r *Reporter) generateMarkdownScheduleReport(schedule *Schedule, analysis *ScheduleAnalysis) (string, error) {
	var buf bytes.Buffer
	
	// Header
	buf.WriteString("# Schedule Analysis Report\n\n")
	buf.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))
	
	// Strategy
	buf.WriteString(fmt.Sprintf("**Strategy:** %s  \n", schedule.Strategy))
	buf.WriteString(fmt.Sprintf("**Estimated Duration:** %v\n\n", schedule.EstimatedDuration))
	
	// Metrics
	if analysis != nil {
		buf.WriteString("## Metrics\n\n")
		buf.WriteString("| Metric | Value |\n")
		buf.WriteString("|--------|-------|\n")
		buf.WriteString(fmt.Sprintf("| Total Operations | %d |\n", analysis.TotalOperations))
		buf.WriteString(fmt.Sprintf("| Total Stages | %d |\n", analysis.TotalStages))
		buf.WriteString(fmt.Sprintf("| Avg Stage Size | %.2f operations |\n", analysis.AvgStageSize))
		buf.WriteString(fmt.Sprintf("| Parallelization Factor | %.2fx |\n", analysis.ParallelizationFactor))
		buf.WriteString(fmt.Sprintf("| Efficiency | %.1f%% |\n", analysis.Efficiency*100))
		buf.WriteString("\n")
	}
	
	// Stages
	buf.WriteString("## Execution Stages\n\n")
	
	for i, stage := range schedule.Stages {
		// Calculate stage duration
		stageDuration := time.Duration(0)
		for _, node := range stage {
			if node.Properties.EstimatedDuration > stageDuration {
				stageDuration = node.Properties.EstimatedDuration
			}
		}
		
		buf.WriteString(fmt.Sprintf("### Stage %d\n\n", i+1))
		buf.WriteString(fmt.Sprintf("**Operations:** %d  \n", len(stage)))
		buf.WriteString(fmt.Sprintf("**Duration:** %v\n\n", stageDuration))
		
		// List operations
		for j, node := range stage {
			marker := ""
			if node.IsCritical {
				marker = " ⚡"
			}
			buf.WriteString(fmt.Sprintf("%d. `%s` (%v)%s\n", 
				j+1, node.Name, node.Properties.EstimatedDuration, marker))
		}
		buf.WriteString("\n")
	}
	
	return buf.String(), nil
}

// GenerateOptimizationReport generates an optimization suggestions report
func (r *Reporter) GenerateOptimizationReport(suggestions []OptimizationSuggestion) (string, error) {
	switch r.format {
	case ReportFormatText:
		return r.generateTextOptimizationReport(suggestions)
	case ReportFormatMarkdown:
		return r.generateMarkdownOptimizationReport(suggestions)
	case ReportFormatJSON:
		data, err := json.MarshalIndent(suggestions, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported report format: %s", r.format)
	}
}

// generateTextOptimizationReport generates a plain text optimization report
func (r *Reporter) generateTextOptimizationReport(suggestions []OptimizationSuggestion) (string, error) {
	var buf bytes.Buffer
	
	buf.WriteString("Optimization Suggestions Report\n")
	buf.WriteString(strings.Repeat("=", 70) + "\n\n")
	buf.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC3339)))
	
	if len(suggestions) == 0 {
		buf.WriteString("No optimization suggestions. The graph is well-optimized!\n")
		return buf.String(), nil
	}
	
	// Group by severity
	highSeverity := make([]OptimizationSuggestion, 0)
	mediumSeverity := make([]OptimizationSuggestion, 0)
	lowSeverity := make([]OptimizationSuggestion, 0)
	
	for _, sug := range suggestions {
		switch sug.Severity {
		case "high":
			highSeverity = append(highSeverity, sug)
		case "medium":
			mediumSeverity = append(mediumSeverity, sug)
		case "low":
			lowSeverity = append(lowSeverity, sug)
		}
	}
	
	// High severity
	if len(highSeverity) > 0 {
		buf.WriteString("HIGH SEVERITY\n")
		buf.WriteString(strings.Repeat("-", 70) + "\n")
		for i, sug := range highSeverity {
			buf.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, sug.Description))
			buf.WriteString(fmt.Sprintf("   Type:   %s\n", sug.Type))
			buf.WriteString(fmt.Sprintf("   Impact: %s\n", sug.Impact))
			buf.WriteString(fmt.Sprintf("   Action: %s\n", sug.Action))
		}
		buf.WriteString("\n")
	}
	
	// Medium severity
	if len(mediumSeverity) > 0 {
		buf.WriteString("MEDIUM SEVERITY\n")
		buf.WriteString(strings.Repeat("-", 70) + "\n")
		for i, sug := range mediumSeverity {
			buf.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, sug.Description))
			buf.WriteString(fmt.Sprintf("   Impact: %s\n", sug.Impact))
			buf.WriteString(fmt.Sprintf("   Action: %s\n", sug.Action))
		}
		buf.WriteString("\n")
	}
	
	// Low severity
	if len(lowSeverity) > 0 {
		buf.WriteString("LOW SEVERITY\n")
		buf.WriteString(strings.Repeat("-", 70) + "\n")
		for i, sug := range lowSeverity {
			buf.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, sug.Description))
			buf.WriteString(fmt.Sprintf("   Action: %s\n", sug.Action))
		}
		buf.WriteString("\n")
	}
	
	return buf.String(), nil
}

// generateMarkdownOptimizationReport generates a Markdown optimization report
func (r *Reporter) generateMarkdownOptimizationReport(suggestions []OptimizationSuggestion) (string, error) {
	var buf bytes.Buffer
	
	buf.WriteString("# Optimization Suggestions Report\n\n")
	buf.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))
	
	if len(suggestions) == 0 {
		buf.WriteString("✅ No optimization suggestions. The graph is well-optimized!\n")
		return buf.String(), nil
	}
	
	// Group by severity
	severityGroups := make(map[string][]OptimizationSuggestion)
	severityGroups["high"] = make([]OptimizationSuggestion, 0)
	severityGroups["medium"] = make([]OptimizationSuggestion, 0)
	severityGroups["low"] = make([]OptimizationSuggestion, 0)
	
	for _, sug := range suggestions {
		severityGroups[sug.Severity] = append(severityGroups[sug.Severity], sug)
	}
	
	// High severity
	if len(severityGroups["high"]) > 0 {
		buf.WriteString("## 🔴 High Severity\n\n")
		for i, sug := range severityGroups["high"] {
			buf.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, sug.Description))
			buf.WriteString(fmt.Sprintf("- **Type:** `%s`\n", sug.Type))
			buf.WriteString(fmt.Sprintf("- **Impact:** %s\n", sug.Impact))
			buf.WriteString(fmt.Sprintf("- **Recommended Action:** %s\n\n", sug.Action))
		}
	}
	
	// Medium severity
	if len(severityGroups["medium"]) > 0 {
		buf.WriteString("## 🟡 Medium Severity\n\n")
		for i, sug := range severityGroups["medium"] {
			buf.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, sug.Description))
			buf.WriteString(fmt.Sprintf("- **Impact:** %s\n", sug.Impact))
			buf.WriteString(fmt.Sprintf("- **Recommended Action:** %s\n\n", sug.Action))
		}
	}
	
	// Low severity
	if len(severityGroups["low"]) > 0 {
		buf.WriteString("## 🟢 Low Severity\n\n")
		for i, sug := range severityGroups["low"] {
			buf.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, sug.Description, sug.Action))
		}
		buf.WriteString("\n")
	}
	
	return buf.String(), nil
}

// GenerateComparisonReport generates a comparison report for schedules or graphs
func GenerateComparisonReport(comparison *ScheduleComparison, format ReportFormat) (string, error) {
	if comparison == nil {
		return "", fmt.Errorf("comparison cannot be nil")
	}
	
	var buf bytes.Buffer
	
	switch format {
	case ReportFormatText:
		buf.WriteString("Schedule Comparison Report\n")
		buf.WriteString(strings.Repeat("=", 70) + "\n\n")
		
		buf.WriteString(fmt.Sprintf("%s vs %s\n\n", comparison.Schedule1, comparison.Schedule2))
		
		buf.WriteString("Duration:\n")
		if comparison.DurationDifference < 0 {
			buf.WriteString(fmt.Sprintf("  %s is faster by %v (%.1f%%)\n", 
				comparison.Schedule2, -comparison.DurationDifference, -comparison.DurationPercentChange))
		} else if comparison.DurationDifference > 0 {
			buf.WriteString(fmt.Sprintf("  %s is faster by %v (%.1f%%)\n", 
				comparison.Schedule1, comparison.DurationDifference, comparison.DurationPercentChange))
		} else {
			buf.WriteString("  Equal duration\n")
		}
		
		buf.WriteString("\nStages:\n")
		// Note: StageDifference is schedule2.stages - schedule1.stages
		if comparison.StageDifference != 0 {
			buf.WriteString(fmt.Sprintf("  Difference: %+d stages (%.1f%%)\n", 
				comparison.StageDifference, comparison.StagePercentChange))
		} else {
			buf.WriteString("  Same number of stages\n")
		}
		
		buf.WriteString("\nParallelization:\n")
		buf.WriteString(fmt.Sprintf("  %s: %.2fx\n", comparison.Schedule1, comparison.ParallelizationFactor1))
		buf.WriteString(fmt.Sprintf("  %s: %.2fx\n", comparison.Schedule2, comparison.ParallelizationFactor2))
		
		buf.WriteString(fmt.Sprintf("\nRecommendation: %s\n", comparison.Recommendation))
		
		return buf.String(), nil
		
	case ReportFormatJSON:
		data, err := json.MarshalIndent(comparison, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
		
	default:
		return "", fmt.Errorf("unsupported report format: %s", format)
	}
}

// GenerateSummaryReport generates a high-level summary report
func GenerateSummaryReport(graph *Graph, schedule *Schedule, format ReportFormat) (string, error) {
	var buf bytes.Buffer
	
	switch format {
	case ReportFormatText, ReportFormatMarkdown:
		separator := "="
		if format == ReportFormatMarkdown {
			buf.WriteString("# ")
		}
		buf.WriteString("Execution Plan Summary\n")
		if format == ReportFormatText {
			buf.WriteString(strings.Repeat(separator, 70) + "\n\n")
		} else {
			buf.WriteString("\n")
		}
		
		// Graph metrics
		if format == ReportFormatMarkdown {
			buf.WriteString("## ")
		}
		buf.WriteString("Graph Metrics\n")
		if format == ReportFormatText {
			buf.WriteString(strings.Repeat("-", 70) + "\n")
		} else {
			buf.WriteString("\n")
		}
		
		buf.WriteString(fmt.Sprintf("Operations: %d\n", graph.NodeCount()))
		buf.WriteString(fmt.Sprintf("Dependencies: %d\n", graph.EdgeCount()))
		if graph.MaxLevel > 0 {
			buf.WriteString(fmt.Sprintf("Dependency Levels: %d\n", graph.MaxLevel+1))
		}
		
		// Schedule metrics
		if schedule != nil {
			buf.WriteString("\n")
			if format == ReportFormatMarkdown {
				buf.WriteString("## ")
			}
			buf.WriteString("Schedule Metrics\n")
			if format == ReportFormatText {
				buf.WriteString(strings.Repeat("-", 70) + "\n")
			} else {
				buf.WriteString("\n")
			}
			
			buf.WriteString(fmt.Sprintf("Strategy: %s\n", schedule.Strategy))
			buf.WriteString(fmt.Sprintf("Stages: %d\n", len(schedule.Stages)))
			buf.WriteString(fmt.Sprintf("Estimated Duration: %v\n", schedule.EstimatedDuration))
			buf.WriteString(fmt.Sprintf("Max Parallel Ops: %d\n", schedule.MaxParallelOps))
		}
		
		return buf.String(), nil
		
	default:
		return "", fmt.Errorf("unsupported report format: %s", format)
	}
}
