package dag

import (
	"fmt"
	"sort"
	"time"
)

// Analyzer provides comprehensive dependency analysis capabilities
type Analyzer struct {
	graph *Graph
}

// NewAnalyzer creates a new analyzer for a graph
func NewAnalyzer(graph *Graph) *Analyzer {
	return &Analyzer{
		graph: graph,
	}
}

// AnalyzeDependencies is a convenience function for analyzing dependencies
func AnalyzeDependencies(graph *Graph) (*AnalysisResult, error) {
	analyzer := NewAnalyzer(graph)
	return analyzer.Analyze()
}

// Analyze performs comprehensive analysis of the graph
func (a *Analyzer) Analyze() (*AnalysisResult, error) {
	// Validate graph first
	if err := a.graph.Validate(); err != nil {
		return nil, fmt.Errorf("graph validation failed: %w", err)
	}

	result := &AnalysisResult{
		NodeCount:   a.graph.NodeCount(),
		EdgeCount:   a.graph.EdgeCount(),
		Levels:      make(map[string]int),
		Suggestions: make([]string, 0),
	}

	// Check for cycles
	hasCycle, cycles := a.graph.HasCycle()
	result.HasCycles = hasCycle
	if hasCycle {
		result.Cycles = [][]string{cycles}
		return result, fmt.Errorf("graph contains cycles: %v", cycles)
	}

	// Compute levels
	if err := a.graph.ComputeLevels(); err != nil {
		return nil, fmt.Errorf("failed to compute levels: %w", err)
	}

	result.MaxLevel = a.graph.MaxLevel
	for _, node := range a.graph.Nodes {
		result.Levels[node.ID] = node.Level
	}

	// Compute critical path
	criticalPath, duration, err := a.graph.CriticalPathMethod()
	if err != nil {
		return nil, fmt.Errorf("failed to compute critical path: %w", err)
	}
	result.CriticalPath = criticalPath
	result.CriticalPathDuration = duration

	// Compute parallel groups
	parallelGroups, err := a.graph.ComputeParallelGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to compute parallel groups: %w", err)
	}
	result.ParallelGroups = parallelGroups

	// Calculate parallelization factor
	if result.MaxLevel > 0 {
		result.ParallelizationFactor = float64(result.NodeCount) / float64(result.MaxLevel+1)
	}

	// Find bottlenecks
	result.Bottlenecks = a.findBottlenecks()

	// Perform risk analysis
	result.RiskAnalysis = a.analyzeRisk(criticalPath)

	// Generate optimization suggestions
	result.Suggestions = a.generateSuggestions(result)

	return result, nil
}

// findBottlenecks identifies bottleneck nodes in the graph
func (a *Analyzer) findBottlenecks() []*BottleneckInfo {
	bottlenecks := make([]*BottleneckInfo, 0)

	for _, node := range a.graph.Nodes {
		// Count how many nodes depend on this one (directly or transitively)
		blockedNodes := a.findBlockedNodes(node.ID)

		// A node is a bottleneck if:
		// 1. It blocks many other nodes
		// 2. It's on the critical path
		// 3. It has a long duration
		if len(blockedNodes) > 2 || node.IsCritical {
			impact := float64(len(blockedNodes)) / float64(a.graph.NodeCount())

			bottleneck := &BottleneckInfo{
				NodeID:       node.ID,
				NodeName:     node.Name,
				BlockedNodes: blockedNodes,
				BlockedCount: len(blockedNodes),
				Impact:       impact,
			}

			// Generate reason
			reasons := make([]string, 0)
			if node.IsCritical {
				reasons = append(reasons, "on critical path")
			}
			if len(blockedNodes) > 5 {
				reasons = append(reasons, fmt.Sprintf("blocks %d operations", len(blockedNodes)))
			}
			if node.Properties.EstimatedDuration > 5*time.Minute {
				reasons = append(reasons, "long duration")
			}

			if len(reasons) > 0 {
				bottleneck.Reason = fmt.Sprintf("Bottleneck because: %v", reasons)
			}

			// Generate mitigation suggestion
			if node.Properties.EstimatedDuration > 5*time.Minute {
				bottleneck.Mitigation = "Consider breaking this operation into smaller steps"
			} else if len(blockedNodes) > 5 {
				bottleneck.Mitigation = "Consider reordering operations to reduce dependencies"
			}

			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	// Sort by impact (descending)
	sort.Slice(bottlenecks, func(i, j int) bool {
		return bottlenecks[i].Impact > bottlenecks[j].Impact
	})

	return bottlenecks
}

// findBlockedNodes finds all nodes that transitively depend on the given node
func (a *Analyzer) findBlockedNodes(nodeID string) []string {
	blocked := make([]string, 0)
	visited := make(map[string]bool)

	a.findBlockedNodesUtil(nodeID, visited)

	for id := range visited {
		if id != nodeID {
			blocked = append(blocked, id)
		}
	}

	return blocked
}

// findBlockedNodesUtil is a recursive helper for finding blocked nodes
func (a *Analyzer) findBlockedNodesUtil(nodeID string, visited map[string]bool) {
	visited[nodeID] = true

	// Find all nodes that directly depend on this node (reverse edges)
	for _, edge := range a.graph.ReverseEdges[nodeID] {
		if !visited[edge.From] {
			a.findBlockedNodesUtil(edge.From, visited)
		}
	}
}

// analyzeRisk performs risk analysis on the graph
func (a *Analyzer) analyzeRisk(criticalPath []string) *RiskAnalysisResult {
	result := &RiskAnalysisResult{
		HighRiskOperations:     make([]*Node, 0),
		CriticalRiskOperations: make([]*Node, 0),
		RiskByLevel:            make(map[RiskLevel]int),
	}

	var totalRiskScore float64
	criticalPathMap := make(map[string]bool)
	for _, id := range criticalPath {
		criticalPathMap[id] = true
	}

	// Analyze each node
	for _, node := range a.graph.Nodes {
		riskLevel := node.Properties.RiskLevel
		result.RiskByLevel[riskLevel]++

		// Calculate risk score (0-100)
		riskScore := getRiskScore(riskLevel)
		if node.Properties.IsDestructive {
			riskScore += 20
		}
		if node.IsCritical {
			riskScore += 10
		}

		totalRiskScore += riskScore

		// Collect high-risk operations
		if riskLevel == RiskLevelHigh || riskLevel == RiskLevelCritical {
			result.HighRiskOperations = append(result.HighRiskOperations, node)

			// Check if it's on critical path
			if criticalPathMap[node.ID] {
				result.CriticalRiskOperations = append(result.CriticalRiskOperations, node)
			}
		}
	}

	// Calculate average risk
	if len(a.graph.Nodes) > 0 {
		result.TotalRiskScore = totalRiskScore / float64(len(a.graph.Nodes))
	}

	// Determine average risk level
	result.AverageRiskLevel = getRiskLevelFromScore(result.TotalRiskScore)

	// Sort by risk level
	sort.Slice(result.HighRiskOperations, func(i, j int) bool {
		return getRiskScore(result.HighRiskOperations[i].Properties.RiskLevel) >
			getRiskScore(result.HighRiskOperations[j].Properties.RiskLevel)
	})

	return result
}

// getRiskScore converts a risk level to a numeric score
func getRiskScore(level RiskLevel) float64 {
	switch level {
	case RiskLevelCritical:
		return 100
	case RiskLevelHigh:
		return 75
	case RiskLevelMedium:
		return 50
	case RiskLevelLow:
		return 25
	default:
		return 0
	}
}

// getRiskLevelFromScore converts a numeric score to a risk level
func getRiskLevelFromScore(score float64) RiskLevel {
	if score >= 80 {
		return RiskLevelCritical
	} else if score >= 60 {
		return RiskLevelHigh
	} else if score >= 40 {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

// generateSuggestions generates optimization suggestions based on analysis
func (a *Analyzer) generateSuggestions(analysis *AnalysisResult) []string {
	suggestions := make([]string, 0)

	// Check parallelization factor
	if analysis.ParallelizationFactor < 2.0 {
		suggestions = append(suggestions,
			fmt.Sprintf("Low parallelization factor (%.2f). Consider reducing dependencies to enable more parallel execution",
				analysis.ParallelizationFactor))
	} else if analysis.ParallelizationFactor > 4.0 {
		suggestions = append(suggestions,
			fmt.Sprintf("Excellent parallelization factor (%.2f)!", analysis.ParallelizationFactor))
	}

	// Check critical path
	if len(analysis.CriticalPath) > 0 {
		criticalNodes := make([]*Node, 0)
		for _, id := range analysis.CriticalPath {
			criticalNodes = append(criticalNodes, a.graph.Nodes[id])
		}

		// Find longest operation on critical path
		var longestOp *Node
		var maxDuration time.Duration
		for _, node := range criticalNodes {
			if node.Properties.EstimatedDuration > maxDuration {
				maxDuration = node.Properties.EstimatedDuration
				longestOp = node
			}
		}

		if longestOp != nil && maxDuration > 10*time.Minute {
			suggestions = append(suggestions,
				fmt.Sprintf("Operation '%s' on critical path takes %v. Consider optimizing or breaking into smaller steps",
					longestOp.Name, maxDuration))
		}
	}

	// Check bottlenecks
	if len(analysis.Bottlenecks) > 0 {
		topBottleneck := analysis.Bottlenecks[0]
		suggestions = append(suggestions,
			fmt.Sprintf("Bottleneck detected: '%s' blocks %d operations (%.1f%% of total)",
				topBottleneck.NodeName, topBottleneck.BlockedCount, topBottleneck.Impact*100))
	}

	// Check risk
	if analysis.RiskAnalysis != nil {
		if len(analysis.RiskAnalysis.CriticalRiskOperations) > 0 {
			suggestions = append(suggestions,
				fmt.Sprintf("%d high-risk operations on critical path. Consider moving them earlier (fail-fast) or adding validation steps",
					len(analysis.RiskAnalysis.CriticalRiskOperations)))
		}

		if analysis.RiskAnalysis.TotalRiskScore > 70 {
			suggestions = append(suggestions,
				"High overall risk detected. Consider adding checkpoints or enabling dry-run mode")
		}
	}

	// Check for sequential chains
	maxChainLength := a.findMaxSequentialChain()
	if maxChainLength > 10 {
		suggestions = append(suggestions,
			fmt.Sprintf("Long sequential chain detected (%d operations). Look for opportunities to parallelize",
				maxChainLength))
	}

	// Check node count vs levels
	avgNodesPerLevel := float64(analysis.NodeCount) / float64(analysis.MaxLevel+1)
	if avgNodesPerLevel < 1.5 {
		suggestions = append(suggestions,
			"Many operations are serialized. Review dependencies to enable more parallelism")
	}

	return suggestions
}

// findMaxSequentialChain finds the longest chain of nodes that must execute sequentially
func (a *Analyzer) findMaxSequentialChain() int {
	maxChain := 0

	for _, node := range a.graph.Nodes {
		// Find longest path starting from this node
		chainLength := a.findChainLength(node.ID, make(map[string]bool))
		if chainLength > maxChain {
			maxChain = chainLength
		}
	}

	return maxChain
}

// findChainLength recursively finds the length of the longest chain starting from a node
func (a *Analyzer) findChainLength(nodeID string, visited map[string]bool) int {
	visited[nodeID] = true
	maxLength := 1

	for _, edge := range a.graph.Edges[nodeID] {
		if !visited[edge.To] {
			length := 1 + a.findChainLength(edge.To, visited)
			if length > maxLength {
				maxLength = length
			}
		}
	}

	visited[nodeID] = false
	return maxLength
}

// WhatIfAnalysis performs what-if analysis for a scenario
func (a *Analyzer) WhatIfAnalysis(scenario *WhatIfScenario) (*WhatIfResult, error) {
	// Clone the graph
	modifiedGraph := a.graph.Clone()

	result := &WhatIfResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// Apply changes
	for _, node := range scenario.AddNodes {
		if err := modifiedGraph.AddNode(node); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to add node %s: %v", node.ID, err))
		}
	}

	for _, nodeID := range scenario.RemoveNodes {
		if err := modifiedGraph.RemoveNode(nodeID); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to remove node %s: %v", nodeID, err))
		}
	}

	for _, edge := range scenario.AddEdges {
		if err := modifiedGraph.AddEdge(edge); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to add edge %s->%s: %v", edge.From, edge.To, err))
		}
	}

	for _, edge := range scenario.RemoveEdges {
		if err := modifiedGraph.RemoveEdge(edge.From, edge.To); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to remove edge %s->%s: %v", edge.From, edge.To, err))
		}
	}

	if !result.Valid {
		return result, nil
	}

	// Analyze modified graph
	modifiedAnalyzer := NewAnalyzer(modifiedGraph)
	modifiedAnalysis, err := modifiedAnalyzer.Analyze()
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Analysis failed: %v", err))
		return result, nil
	}

	// Get original analysis for comparison
	originalAnalysis, err := a.Analyze()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze original graph: %w", err)
	}

	// Calculate impacts
	result.DurationChange = modifiedAnalysis.CriticalPathDuration - originalAnalysis.CriticalPathDuration
	result.StageCountChange = modifiedAnalysis.MaxLevel - originalAnalysis.MaxLevel
	result.ParallelismChange = modifiedAnalysis.ParallelizationFactor - originalAnalysis.ParallelizationFactor

	if originalAnalysis.RiskAnalysis != nil && modifiedAnalysis.RiskAnalysis != nil {
		result.RiskChange = modifiedAnalysis.RiskAnalysis.TotalRiskScore - originalAnalysis.RiskAnalysis.TotalRiskScore
	}

	result.NewCriticalPath = modifiedAnalysis.CriticalPath

	// Generate comparison text
	result.Comparison = a.generateComparisonText(originalAnalysis, modifiedAnalysis, result)

	return result, nil
}

// generateComparisonText generates a human-readable comparison
func (a *Analyzer) generateComparisonText(original, modified *AnalysisResult, whatIf *WhatIfResult) string {
	comparison := fmt.Sprintf("What-If Analysis Results:\n\n")

	// Duration comparison
	if whatIf.DurationChange > 0 {
		comparison += fmt.Sprintf("⚠️  Duration increased by %v (%.1f%%)\n",
			whatIf.DurationChange,
			float64(whatIf.DurationChange)/float64(original.CriticalPathDuration)*100)
	} else if whatIf.DurationChange < 0 {
		comparison += fmt.Sprintf("✅ Duration reduced by %v (%.1f%%)\n",
			-whatIf.DurationChange,
			-float64(whatIf.DurationChange)/float64(original.CriticalPathDuration)*100)
	} else {
		comparison += "➡️  Duration unchanged\n"
	}

	// Parallelization comparison
	if whatIf.ParallelismChange > 0.5 {
		comparison += fmt.Sprintf("✅ Parallelization improved by %.2fx\n", whatIf.ParallelismChange)
	} else if whatIf.ParallelismChange < -0.5 {
		comparison += fmt.Sprintf("⚠️  Parallelization reduced by %.2fx\n", -whatIf.ParallelismChange)
	}

	// Risk comparison
	if whatIf.RiskChange > 10 {
		comparison += fmt.Sprintf("⚠️  Risk increased by %.1f points\n", whatIf.RiskChange)
	} else if whatIf.RiskChange < -10 {
		comparison += fmt.Sprintf("✅ Risk reduced by %.1f points\n", -whatIf.RiskChange)
	}

	// Stage count comparison
	if whatIf.StageCountChange > 0 {
		comparison += fmt.Sprintf("Execution stages increased from %d to %d\n",
			original.MaxLevel+1, modified.MaxLevel+1)
	} else if whatIf.StageCountChange < 0 {
		comparison += fmt.Sprintf("Execution stages reduced from %d to %d\n",
			original.MaxLevel+1, modified.MaxLevel+1)
	}

	return comparison
}

// CompareGraphs compares two graphs and returns the differences
func CompareGraphs(g1, g2 *Graph) string {
	comparison := "Graph Comparison:\n\n"

	// Node count
	comparison += fmt.Sprintf("Nodes: %d → %d (%+d)\n", g1.NodeCount(), g2.NodeCount(), g2.NodeCount()-g1.NodeCount())

	// Edge count
	comparison += fmt.Sprintf("Edges: %d → %d (%+d)\n", g1.EdgeCount(), g2.EdgeCount(), g2.EdgeCount()-g1.EdgeCount())

	// Added nodes
	addedNodes := make([]string, 0)
	for id := range g2.Nodes {
		if _, exists := g1.Nodes[id]; !exists {
			addedNodes = append(addedNodes, id)
		}
	}
	if len(addedNodes) > 0 {
		comparison += fmt.Sprintf("\nAdded nodes (%d): %v\n", len(addedNodes), addedNodes)
	}

	// Removed nodes
	removedNodes := make([]string, 0)
	for id := range g1.Nodes {
		if _, exists := g2.Nodes[id]; !exists {
			removedNodes = append(removedNodes, id)
		}
	}
	if len(removedNodes) > 0 {
		comparison += fmt.Sprintf("\nRemoved nodes (%d): %v\n", len(removedNodes), removedNodes)
	}

	return comparison
}
