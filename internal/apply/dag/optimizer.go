package dag

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// OptimizationStrategy defines different optimization approaches
type OptimizationStrategy string

const (
	// OptimizeForSpeed minimizes total execution time
	OptimizeForSpeed OptimizationStrategy = "speed"

	// OptimizeForCost minimizes total cost
	OptimizeForCost OptimizationStrategy = "cost"

	// OptimizeForReliability maximizes reliability (fail-safe ordering)
	OptimizeForReliability OptimizationStrategy = "reliability"

	// OptimizeForBalance balances speed, cost, and reliability
	OptimizeForBalance OptimizationStrategy = "balance"
)

// Optimizer optimizes execution plans
type Optimizer struct {
	strategy OptimizationStrategy
	config   ScheduleConfig
}

// NewOptimizer creates a new optimizer
func NewOptimizer(strategy OptimizationStrategy, config ScheduleConfig) *Optimizer {
	return &Optimizer{
		strategy: strategy,
		config:   config,
	}
}

// Optimize optimizes a graph for execution
func (o *Optimizer) Optimize(ctx context.Context, graph *Graph) (*Graph, error) {
	if graph == nil {
		return nil, fmt.Errorf("graph cannot be nil")
	}

	// Clone graph to avoid modifying original
	optimized := graph.Clone()

	switch o.strategy {
	case OptimizeForSpeed:
		return o.optimizeForSpeed(ctx, optimized)
	case OptimizeForCost:
		return o.optimizeForCost(ctx, optimized)
	case OptimizeForReliability:
		return o.optimizeForReliability(ctx, optimized)
	case OptimizeForBalance:
		return o.optimizeForBalance(ctx, optimized)
	default:
		return optimized, nil
	}
}

// optimizeForSpeed minimizes total execution time
func (o *Optimizer) optimizeForSpeed(ctx context.Context, graph *Graph) (*Graph, error) {
	// Remove redundant dependencies (transitive reduction)
	reduced := graph.TransitiveReduction()

	// Compute critical path
	criticalPath, duration, err := reduced.CriticalPathMethod()
	if err != nil {
		return nil, fmt.Errorf("failed to compute critical path: %w", err)
	}

	// Mark critical operations with high priority
	for _, nodeID := range criticalPath {
		node := reduced.Nodes[nodeID]
		node.Properties.Priority = 1000 // High priority for critical path
	}

	// Identify operations that can be parallelized
	if err := reduced.ComputeLevels(); err != nil {
		return nil, fmt.Errorf("failed to compute levels: %w", err)
	}

	// Optimize edge weights based on duration impact
	for _, edges := range reduced.Edges {
		for _, edge := range edges {
			fromNode := reduced.Nodes[edge.From]
			toNode := reduced.Nodes[edge.To]

			// Weight = impact on critical path
			if fromNode.IsCritical && toNode.IsCritical {
				edge.Weight = 10.0 // Critical edge
			} else if fromNode.IsCritical || toNode.IsCritical {
				edge.Weight = 5.0 // Semi-critical edge
			} else {
				edge.Weight = 1.0 // Normal edge
			}
		}
	}

	reduced.TotalDuration = duration
	reduced.CriticalPath = criticalPath

	return reduced, nil
}

// optimizeForCost minimizes total cost
func (o *Optimizer) optimizeForCost(ctx context.Context, graph *Graph) (*Graph, error) {
	optimized := graph.Clone()

	// Sort operations by cost efficiency (duration / cost)
	nodes := make([]*Node, 0, len(optimized.Nodes))
	for _, node := range optimized.Nodes {
		nodes = append(nodes, node)
	}

	// Assign priorities based on cost efficiency
	sort.Slice(nodes, func(i, j int) bool {
		// Lower cost = higher priority
		costi := nodes[i].Properties.Cost
		costj := nodes[j].Properties.Cost

		if costi == 0 {
			costi = 1.0
		}
		if costj == 0 {
			costj = 1.0
		}

		// Cost per unit time
		efficiencyi := costi / float64(nodes[i].Properties.EstimatedDuration)
		efficiencyj := costj / float64(nodes[j].Properties.EstimatedDuration)

		return efficiencyi < efficiencyj
	})

	// Set priorities (lower cost = higher priority)
	for i, node := range nodes {
		node.Properties.Priority = len(nodes) - i
	}

	// Prefer idempotent operations (can retry without additional cost)
	for _, node := range optimized.Nodes {
		if node.Properties.Idempotent {
			node.Properties.Priority += 100
		}
	}

	return optimized, nil
}

// optimizeForReliability maximizes reliability
func (o *Optimizer) optimizeForReliability(ctx context.Context, graph *Graph) (*Graph, error) {
	optimized := graph.Clone()

	// Prioritize retriable and idempotent operations
	for _, node := range optimized.Nodes {
		priority := 0

		// Idempotent operations are safer
		if node.Properties.Idempotent {
			priority += 200
		}

		// Retriable operations are safer
		if node.Properties.Retriable {
			priority += 150
		}

		// Lower risk operations first (fail-safe)
		switch node.Properties.RiskLevel {
		case RiskLevelLow:
			priority += 100
		case RiskLevelMedium:
			priority += 50
		case RiskLevelHigh:
			priority += 25
		case RiskLevelCritical:
			priority += 10
		}

		// Non-destructive operations first
		if !node.Properties.IsDestructive {
			priority += 75
		}

		node.Properties.Priority = priority
	}

	// Add soft dependencies between destructive operations
	// to ensure they run in safer order
	destructiveNodes := make([]*Node, 0)
	for _, node := range optimized.Nodes {
		if node.Properties.IsDestructive {
			destructiveNodes = append(destructiveNodes, node)
		}
	}

	// Sort destructive nodes by risk
	sort.Slice(destructiveNodes, func(i, j int) bool {
		return destructiveNodes[i].Properties.RiskLevel < destructiveNodes[j].Properties.RiskLevel
	})

	// Add soft ordering dependencies between destructive operations
	for i := 0; i < len(destructiveNodes)-1; i++ {
		from := destructiveNodes[i]
		to := destructiveNodes[i+1]

		// Only add if no path exists (avoid redundant edges)
		if !optimized.IsReachable(from.ID, to.ID) && !optimized.IsReachable(to.ID, from.ID) {
			edge := &Edge{
				From:   from.ID,
				To:     to.ID,
				Type:   DependencyTypeSoft,
				Weight: 0.5,
				Reason: "Reliability optimization: safer destructive operation ordering",
			}
			optimized.AddEdge(edge)
		}
	}

	return optimized, nil
}

// optimizeForBalance balances speed, cost, and reliability
func (o *Optimizer) optimizeForBalance(ctx context.Context, graph *Graph) (*Graph, error) {
	optimized := graph.Clone()

	// Compute critical path for speed consideration
	criticalPath, _, err := optimized.CriticalPathMethod()
	if err != nil {
		return nil, fmt.Errorf("failed to compute critical path: %w", err)
	}

	criticalSet := make(map[string]bool)
	for _, nodeID := range criticalPath {
		criticalSet[nodeID] = true
	}

	// Balanced scoring system
	for _, node := range optimized.Nodes {
		score := 0.0

		// Speed factor (30% weight)
		if criticalSet[node.ID] {
			score += 300.0 // Critical path nodes get high priority
		} else {
			score += 100.0 / float64(node.Level+1) // Earlier levels get higher priority
		}

		// Cost factor (30% weight)
		if node.Properties.Cost > 0 {
			costScore := 1000.0 / node.Properties.Cost // Lower cost = higher score
			score += costScore * 0.3
		} else {
			score += 150.0 // Default for zero-cost operations
		}

		// Reliability factor (40% weight)
		reliabilityScore := 0.0
		if node.Properties.Idempotent {
			reliabilityScore += 100.0
		}
		if node.Properties.Retriable {
			reliabilityScore += 75.0
		}
		if !node.Properties.IsDestructive {
			reliabilityScore += 50.0
		}

		switch node.Properties.RiskLevel {
		case RiskLevelLow:
			reliabilityScore += 40.0
		case RiskLevelMedium:
			reliabilityScore += 20.0
		case RiskLevelHigh:
			reliabilityScore += 10.0
		case RiskLevelCritical:
			reliabilityScore += 5.0
		}

		score += reliabilityScore * 0.4

		node.Properties.Priority = int(score)
	}

	// Apply transitive reduction for speed
	reduced := optimized.TransitiveReduction()

	return reduced, nil
}

// SuggestOptimizations analyzes a graph and suggests optimizations
func (o *Optimizer) SuggestOptimizations(graph *Graph) []OptimizationSuggestion {
	suggestions := make([]OptimizationSuggestion, 0)

	// Check for redundant dependencies
	closure := graph.TransitiveClosure()
	redundantCount := 0
	for from := range graph.Nodes {
		for _, edge := range graph.Edges[from] {
			to := edge.To
			// Check if there's an alternative path
			for intermediate := range graph.Nodes {
				if intermediate != from && intermediate != to {
					if closure[from][intermediate] && closure[intermediate][to] {
						redundantCount++
						break
					}
				}
			}
		}
	}

	if redundantCount > 0 {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "redundant_dependencies",
			Severity:    "medium",
			Description: fmt.Sprintf("Found %d redundant dependencies that could be removed", redundantCount),
			Impact:      "Simplifies graph, may improve performance",
			Action:      "Run transitive reduction",
		})
	}

	// Check for bottlenecks
	analysis, err := AnalyzeDependencies(graph)
	if err == nil && len(analysis.Bottlenecks) > 0 {
		for _, bottleneck := range analysis.Bottlenecks {
			if bottleneck.Impact > 0.3 { // Significant impact
				suggestions = append(suggestions, OptimizationSuggestion{
					Type:        "bottleneck",
					Severity:    "high",
					Description: fmt.Sprintf("Node %s blocks %d operations", bottleneck.NodeID, bottleneck.BlockedCount),
					Impact:      fmt.Sprintf("%.1f%% of operations affected", bottleneck.Impact*100),
					Action:      bottleneck.Mitigation,
				})
			}
		}
	}

	// Check for parallelization opportunities
	if err := graph.ComputeLevels(); err == nil {
		levelMap := make(map[int]int)
		for _, node := range graph.Nodes {
			levelMap[node.Level]++
		}

		// Find levels with many operations
		for level, count := range levelMap {
			if count > o.config.MaxParallelOps*2 {
				suggestions = append(suggestions, OptimizationSuggestion{
					Type:        "parallelization",
					Severity:    "low",
					Description: fmt.Sprintf("Level %d has %d operations (max parallel: %d)", level, count, o.config.MaxParallelOps),
					Impact:      "May cause scheduling delays",
					Action:      "Consider increasing max parallel operations or splitting level",
				})
			}
		}
	}

	// Check for long critical path
	if criticalPath, duration, err := graph.CriticalPathMethod(); err == nil {
		avgDuration := duration / time.Duration(len(graph.Nodes))
		criticalDuration := time.Duration(0)
		for _, nodeID := range criticalPath {
			node := graph.Nodes[nodeID]
			criticalDuration += node.Properties.EstimatedDuration
		}

		// If critical path is much longer than average, it's a problem
		if criticalDuration > avgDuration*time.Duration(len(graph.Nodes)/2) {
			suggestions = append(suggestions, OptimizationSuggestion{
				Type:        "long_critical_path",
				Severity:    "high",
				Description: fmt.Sprintf("Critical path is %v (avg per operation: %v)", criticalDuration, avgDuration),
				Impact:      "Total execution time dominated by critical path",
				Action:      "Optimize operations on critical path or parallelize dependencies",
			})
		}
	}

	// Check for high-risk operations
	highRiskCount := 0
	for _, node := range graph.Nodes {
		if node.Properties.RiskLevel == RiskLevelHigh || node.Properties.RiskLevel == RiskLevelCritical {
			highRiskCount++
		}
	}

	if highRiskCount > len(graph.Nodes)/4 { // More than 25% high risk
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "high_risk",
			Severity:    "medium",
			Description: fmt.Sprintf("%d high-risk operations (%.1f%% of total)", highRiskCount, float64(highRiskCount)/float64(len(graph.Nodes))*100),
			Impact:      "Increased failure probability",
			Action:      "Review high-risk operations, add retry logic, or run with risk-based scheduling",
		})
	}

	// Sort by severity
	severityOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
	sort.Slice(suggestions, func(i, j int) bool {
		return severityOrder[suggestions[i].Severity] > severityOrder[suggestions[j].Severity]
	})

	return suggestions
}

// OptimizationSuggestion represents a suggested optimization
type OptimizationSuggestion struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"` // low, medium, high
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Action      string `json:"action"`
}

// CompareSchedules compares two schedules and returns metrics
func CompareSchedules(schedule1, schedule2 *Schedule) *ScheduleComparison {
	if schedule1 == nil || schedule2 == nil {
		return nil
	}

	// Count total operations
	ops1 := 0
	for _, stage := range schedule1.Stages {
		ops1 += len(stage)
	}

	ops2 := 0
	for _, stage := range schedule2.Stages {
		ops2 += len(stage)
	}

	// Compute parallelization factors
	parallel1 := float64(ops1) / float64(len(schedule1.Stages))
	parallel2 := float64(ops2) / float64(len(schedule2.Stages))

	// Duration comparison
	durationDiff := schedule2.EstimatedDuration - schedule1.EstimatedDuration
	durationPercent := 0.0
	if schedule1.EstimatedDuration > 0 {
		durationPercent = float64(durationDiff) / float64(schedule1.EstimatedDuration) * 100
	}

	// Stage count comparison
	stageDiff := len(schedule2.Stages) - len(schedule1.Stages)
	stagePercent := 0.0
	if len(schedule1.Stages) > 0 {
		stagePercent = float64(stageDiff) / float64(len(schedule1.Stages)) * 100
	}

	return &ScheduleComparison{
		Schedule1:              "Schedule 1",
		Schedule2:              "Schedule 2",
		DurationDifference:     durationDiff,
		DurationPercentChange:  durationPercent,
		StageDifference:        stageDiff,
		StagePercentChange:     stagePercent,
		ParallelizationFactor1: parallel1,
		ParallelizationFactor2: parallel2,
		ParallelizationChange:  parallel2 - parallel1,
		Recommendation:         generateRecommendation(durationDiff, stageDiff, parallel1, parallel2),
	}
}

// generateRecommendation generates a recommendation based on comparison
func generateRecommendation(durationDiff time.Duration, stageDiff int, parallel1, parallel2 float64) string {
	if durationDiff < 0 {
		return fmt.Sprintf("Schedule 2 is faster by %v (%.1f%%). Recommended.",
			-durationDiff, float64(-durationDiff)/float64(durationDiff+durationDiff)*100)
	} else if durationDiff > 0 {
		return fmt.Sprintf("Schedule 1 is faster by %v. Use Schedule 1.", durationDiff)
	}

	if parallel2 > parallel1 {
		return fmt.Sprintf("Schedule 2 has better parallelization (%.2f vs %.2f). Recommended.", parallel2, parallel1)
	}

	return "Schedules are equivalent. Choose based on other criteria."
}

// ScheduleComparison contains comparison metrics between two schedules
type ScheduleComparison struct {
	Schedule1              string        `json:"schedule1"`
	Schedule2              string        `json:"schedule2"`
	DurationDifference     time.Duration `json:"durationDifference"`
	DurationPercentChange  float64       `json:"durationPercentChange"`
	StageDifference        int           `json:"stageDifference"`
	StagePercentChange     float64       `json:"stagePercentChange"`
	ParallelizationFactor1 float64       `json:"parallelizationFactor1"`
	ParallelizationFactor2 float64       `json:"parallelizationFactor2"`
	ParallelizationChange  float64       `json:"parallelizationChange"`
	Recommendation         string        `json:"recommendation"`
}
