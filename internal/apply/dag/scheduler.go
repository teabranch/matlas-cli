package dag

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Scheduler manages the scheduling of operations
type Scheduler struct {
	config ScheduleConfig
}

// NewScheduler creates a new scheduler with the given configuration
func NewScheduler(config ScheduleConfig) *Scheduler {
	// Set defaults
	if config.MaxParallelOps == 0 {
		config.MaxParallelOps = 5
	}
	if config.Strategy == "" {
		config.Strategy = StrategyGreedy
	}
	
	return &Scheduler{
		config: config,
	}
}

// Schedule creates an optimized execution schedule from a graph
func (s *Scheduler) Schedule(ctx context.Context, graph *Graph) (*Schedule, error) {
	if graph == nil {
		return nil, fmt.Errorf("graph cannot be nil")
	}
	
	// Validate graph
	if err := graph.Validate(); err != nil {
		return nil, fmt.Errorf("invalid graph: %w", err)
	}
	
	// Check for cycles
	if hasCycle, cycle := graph.HasCycle(); hasCycle {
		return nil, fmt.Errorf("graph contains cycle: %v", cycle)
	}
	
	// Choose scheduling strategy
	switch s.config.Strategy {
	case StrategyGreedy:
		return s.scheduleGreedy(ctx, graph)
	case StrategyCriticalPathFirst:
		return s.scheduleCriticalPathFirst(ctx, graph)
	case StrategyRiskBasedEarly:
		return s.scheduleRiskBased(ctx, graph, true)
	case StrategyRiskBasedLate:
		return s.scheduleRiskBased(ctx, graph, false)
	case StrategyResourceLeveling:
		return s.scheduleResourceLeveling(ctx, graph)
	case StrategyBatchOptimized:
		return s.scheduleBatchOptimized(ctx, graph)
	default:
		return nil, fmt.Errorf("unknown scheduling strategy: %s", s.config.Strategy)
	}
}

// scheduleGreedy implements greedy parallelization
// Maximizes parallel operations at each stage
func (s *Scheduler) scheduleGreedy(ctx context.Context, graph *Graph) (*Schedule, error) {
	// Get topological ordering
	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("topological sort failed: %w", err)
	}
	
	// Compute levels (distance from sources)
	if err := graph.ComputeLevels(); err != nil {
		return nil, fmt.Errorf("failed to compute levels: %w", err)
	}
	
	// Group nodes by level
	levelMap := make(map[int][]*Node)
	maxLevel := 0
	for _, nodeID := range sorted {
		node := graph.Nodes[nodeID]
		levelMap[node.Level] = append(levelMap[node.Level], node)
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}
	
	// Create stages
	stages := make([][]*Node, 0, maxLevel+1)
	for level := 0; level <= maxLevel; level++ {
		nodes := levelMap[level]
		if len(nodes) == 0 {
			continue
		}
		
		// Sort nodes within level by priority
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Properties.Priority > nodes[j].Properties.Priority
		})
		
		// Split into batches based on maxParallelOps
		for i := 0; i < len(nodes); i += s.config.MaxParallelOps {
			end := i + s.config.MaxParallelOps
			if end > len(nodes) {
				end = len(nodes)
			}
			stages = append(stages, nodes[i:end])
		}
	}
	
	// Compute estimated duration
	totalDuration := time.Duration(0)
	for _, stage := range stages {
		stageDuration := time.Duration(0)
		for _, node := range stage {
			if node.Properties.EstimatedDuration > stageDuration {
				stageDuration = node.Properties.EstimatedDuration
			}
		}
		totalDuration += stageDuration
	}
	
	return &Schedule{
		Stages:             stages,
		Strategy:           s.config.Strategy,
		EstimatedDuration:  totalDuration,
		MaxParallelOps:     s.config.MaxParallelOps,
		CreatedAt:          time.Now(),
	}, nil
}

// scheduleCriticalPathFirst implements critical path first scheduling
// Prioritizes operations on the critical path
func (s *Scheduler) scheduleCriticalPathFirst(ctx context.Context, graph *Graph) (*Schedule, error) {
	// Compute critical path
	criticalPath, totalDuration, err := graph.CriticalPathMethod()
	if err != nil {
		return nil, fmt.Errorf("critical path computation failed: %w", err)
	}
	
	// Mark critical nodes
	criticalSet := make(map[string]bool)
	for _, nodeID := range criticalPath {
		criticalSet[nodeID] = true
		graph.Nodes[nodeID].IsCritical = true
	}
	
	// Get topological ordering
	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("topological sort failed: %w", err)
	}
	
	// Compute levels
	if err := graph.ComputeLevels(); err != nil {
		return nil, fmt.Errorf("failed to compute levels: %w", err)
	}
	
	// Group by level, prioritizing critical nodes
	levelMap := make(map[int][]*Node)
	maxLevel := 0
	for _, nodeID := range sorted {
		node := graph.Nodes[nodeID]
		levelMap[node.Level] = append(levelMap[node.Level], node)
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}
	
	// Create stages with critical nodes first
	stages := make([][]*Node, 0)
	for level := 0; level <= maxLevel; level++ {
		nodes := levelMap[level]
		if len(nodes) == 0 {
			continue
		}
		
		// Separate critical and non-critical
		critical := make([]*Node, 0)
		nonCritical := make([]*Node, 0)
		for _, node := range nodes {
			if criticalSet[node.ID] {
				critical = append(critical, node)
			} else {
				nonCritical = append(nonCritical, node)
			}
		}
		
		// Process critical nodes first
		for _, node := range critical {
			stages = append(stages, []*Node{node})
		}
		
		// Then batch non-critical nodes
		for i := 0; i < len(nonCritical); i += s.config.MaxParallelOps {
			end := i + s.config.MaxParallelOps
			if end > len(nonCritical) {
				end = len(nonCritical)
			}
			stages = append(stages, nonCritical[i:end])
		}
	}
	
	return &Schedule{
		Stages:             stages,
		Strategy:           s.config.Strategy,
		EstimatedDuration:  totalDuration,
		CriticalPath:       criticalPath,
		MaxParallelOps:     s.config.MaxParallelOps,
		CreatedAt:          time.Now(),
	}, nil
}

// scheduleRiskBased implements risk-based scheduling
// If earlyRisk is true, high-risk operations are scheduled early (fail-fast)
// If earlyRisk is false, high-risk operations are scheduled late (minimize disruption)
func (s *Scheduler) scheduleRiskBased(ctx context.Context, graph *Graph, earlyRisk bool) (*Schedule, error) {
	// Get topological ordering
	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("topological sort failed: %w", err)
	}
	
	// Compute levels
	if err := graph.ComputeLevels(); err != nil {
		return nil, fmt.Errorf("failed to compute levels: %w", err)
	}
	
	// Group by level
	levelMap := make(map[int][]*Node)
	maxLevel := 0
	for _, nodeID := range sorted {
		node := graph.Nodes[nodeID]
		levelMap[node.Level] = append(levelMap[node.Level], node)
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}
	
	// Risk level priorities
	riskPriority := map[RiskLevel]int{
		RiskLevelCritical: 4,
		RiskLevelHigh:     3,
		RiskLevelMedium:   2,
		RiskLevelLow:      1,
	}
	
	// Create stages, sorting by risk within each level
	stages := make([][]*Node, 0)
	for level := 0; level <= maxLevel; level++ {
		nodes := levelMap[level]
		if len(nodes) == 0 {
			continue
		}
		
		// Sort by risk level
		sort.Slice(nodes, func(i, j int) bool {
			riski := riskPriority[nodes[i].Properties.RiskLevel]
			riskj := riskPriority[nodes[j].Properties.RiskLevel]
			
			if earlyRisk {
				return riski > riskj // High risk first
			}
			return riski < riskj // Low risk first
		})
		
		// Batch into stages
		for i := 0; i < len(nodes); i += s.config.MaxParallelOps {
			end := i + s.config.MaxParallelOps
			if end > len(nodes) {
				end = len(nodes)
			}
			stages = append(stages, nodes[i:end])
		}
	}
	
	// Compute estimated duration
	totalDuration := time.Duration(0)
	for _, stage := range stages {
		stageDuration := time.Duration(0)
		for _, node := range stage {
			if node.Properties.EstimatedDuration > stageDuration {
				stageDuration = node.Properties.EstimatedDuration
			}
		}
		totalDuration += stageDuration
	}
	
	return &Schedule{
		Stages:             stages,
		Strategy:           s.config.Strategy,
		EstimatedDuration:  totalDuration,
		MaxParallelOps:     s.config.MaxParallelOps,
		CreatedAt:          time.Now(),
	}, nil
}

// scheduleResourceLeveling implements resource leveling
// Balances resource usage across stages to avoid bottlenecks
func (s *Scheduler) scheduleResourceLeveling(ctx context.Context, graph *Graph) (*Schedule, error) {
	// Get topological ordering
	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("topological sort failed: %w", err)
	}
	
	// Compute levels
	if err := graph.ComputeLevels(); err != nil {
		return nil, fmt.Errorf("failed to compute levels: %w", err)
	}
	
	// Track available nodes (nodes whose dependencies are satisfied)
	available := make([]*Node, 0)
	inDegree := make(map[string]int)
	
	// Initialize in-degrees
	for _, nodeID := range sorted {
		inDegree[nodeID] = len(graph.Edges[nodeID])
		if inDegree[nodeID] == 0 {
			available = append(available, graph.Nodes[nodeID])
		}
	}
	
	// Resource tracking
	targetAPICallsPerSec := s.config.MaxAPICallsPerSec
	if targetAPICallsPerSec == 0 {
		targetAPICallsPerSec = 100 // Default limit
	}
	
	stages := make([][]*Node, 0)
	processed := make(map[string]bool)
	
	for len(available) > 0 {
		// Sort available nodes by resource requirements
		sort.Slice(available, func(i, j int) bool {
			return available[i].Properties.ResourceRequirements.APICallsRequired < 
				   available[j].Properties.ResourceRequirements.APICallsRequired
		})
		
		// Fill stage up to resource limits
		stage := make([]*Node, 0)
		stageAPICallsPerSec := 0
		
		for len(available) > 0 && len(stage) < s.config.MaxParallelOps {
			node := available[0]
			available = available[1:]
			
			apiCalls := node.Properties.ResourceRequirements.APICallsRequired
			if apiCalls == 0 {
				apiCalls = 1 // Default minimum
			}
			
			// Check if adding this node would exceed resource limits
			if stageAPICallsPerSec + apiCalls > targetAPICallsPerSec && len(stage) > 0 {
				// Put it back for next stage
				available = append([]*Node{node}, available...)
				break
			}
			
			stage = append(stage, node)
			stageAPICallsPerSec += apiCalls
			processed[node.ID] = true
			
			// Add newly available nodes
			for _, dependent := range graph.GetDependents(node.ID) {
				inDegree[dependent]--
				if inDegree[dependent] == 0 && !processed[dependent] {
					available = append(available, graph.Nodes[dependent])
				}
			}
		}
		
		if len(stage) > 0 {
			stages = append(stages, stage)
		} else {
			// No nodes could be scheduled - might be resource constraints too tight
			if len(available) > 0 {
				// Force schedule at least one node
				stage = []*Node{available[0]}
				available = available[1:]
				processed[stage[0].ID] = true
				stages = append(stages, stage)
			}
		}
	}
	
	// Compute estimated duration
	totalDuration := time.Duration(0)
	for _, stage := range stages {
		stageDuration := time.Duration(0)
		for _, node := range stage {
			if node.Properties.EstimatedDuration > stageDuration {
				stageDuration = node.Properties.EstimatedDuration
			}
		}
		totalDuration += stageDuration
	}
	
	return &Schedule{
		Stages:             stages,
		Strategy:           s.config.Strategy,
		EstimatedDuration:  totalDuration,
		MaxParallelOps:     s.config.MaxParallelOps,
		CreatedAt:          time.Now(),
	}, nil
}

// scheduleBatchOptimized implements batch-optimized scheduling
// Groups similar operations together for efficiency
func (s *Scheduler) scheduleBatchOptimized(ctx context.Context, graph *Graph) (*Schedule, error) {
	// Get topological ordering
	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("topological sort failed: %w", err)
	}
	
	// Compute levels
	if err := graph.ComputeLevels(); err != nil {
		return nil, fmt.Errorf("failed to compute levels: %w", err)
	}
	
	// Group by level and resource type
	levelTypeMap := make(map[int]map[string][]*Node)
	maxLevel := 0
	
	for _, nodeID := range sorted {
		node := graph.Nodes[nodeID]
		level := node.Level
		
		if levelTypeMap[level] == nil {
			levelTypeMap[level] = make(map[string][]*Node)
		}
		
		resourceType := string(node.ResourceType)
		levelTypeMap[level][resourceType] = append(levelTypeMap[level][resourceType], node)
		
		if level > maxLevel {
			maxLevel = level
		}
	}
	
	// Create stages, batching by resource type
	stages := make([][]*Node, 0)
	for level := 0; level <= maxLevel; level++ {
		typeMap := levelTypeMap[level]
		if len(typeMap) == 0 {
			continue
		}
		
		// Process each resource type
		for _, nodes := range typeMap {
			// Sort by priority
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].Properties.Priority > nodes[j].Properties.Priority
			})
			
			// Batch into stages
			for i := 0; i < len(nodes); i += s.config.MaxParallelOps {
				end := i + s.config.MaxParallelOps
				if end > len(nodes) {
					end = len(nodes)
				}
				stages = append(stages, nodes[i:end])
			}
		}
	}
	
	// Compute estimated duration
	totalDuration := time.Duration(0)
	for _, stage := range stages {
		stageDuration := time.Duration(0)
		for _, node := range stage {
			if node.Properties.EstimatedDuration > stageDuration {
				stageDuration = node.Properties.EstimatedDuration
			}
		}
		totalDuration += stageDuration
	}
	
	return &Schedule{
		Stages:             stages,
		Strategy:           s.config.Strategy,
		EstimatedDuration:  totalDuration,
		MaxParallelOps:     s.config.MaxParallelOps,
		CreatedAt:          time.Now(),
	}, nil
}

// AnalyzeSchedule analyzes a schedule and returns metrics
func (s *Scheduler) AnalyzeSchedule(schedule *Schedule) *ScheduleAnalysis {
	if schedule == nil {
		return nil
	}
	
	totalOps := 0
	maxStageSize := 0
	minStageSize := 0
	avgStageSize := 0.0
	
	for _, stage := range schedule.Stages {
		stageSize := len(stage)
		totalOps += stageSize
		
		if stageSize > maxStageSize {
			maxStageSize = stageSize
		}
		if minStageSize == 0 || stageSize < minStageSize {
			minStageSize = stageSize
		}
	}
	
	if len(schedule.Stages) > 0 {
		avgStageSize = float64(totalOps) / float64(len(schedule.Stages))
	}
	
	// Compute parallelization factor
	// This is the ratio of total operations to stages
	// Higher means more parallelism
	parallelizationFactor := 1.0
	if len(schedule.Stages) > 0 {
		parallelizationFactor = float64(totalOps) / float64(len(schedule.Stages))
	}
	
	// Compute efficiency
	// This is the ratio of actual parallelization to maximum possible
	efficiency := parallelizationFactor / float64(schedule.MaxParallelOps)
	if efficiency > 1.0 {
		efficiency = 1.0
	}
	
	return &ScheduleAnalysis{
		TotalOperations:       totalOps,
		TotalStages:           len(schedule.Stages),
		AvgStageSize:          avgStageSize,
		MaxStageSize:          maxStageSize,
		MinStageSize:          minStageSize,
		ParallelizationFactor: parallelizationFactor,
		Efficiency:            efficiency,
		EstimatedDuration:     schedule.EstimatedDuration,
	}
}

// ScheduleAnalysis contains metrics about a schedule
type ScheduleAnalysis struct {
	TotalOperations       int
	TotalStages           int
	AvgStageSize          float64
	MaxStageSize          int
	MinStageSize          int
	ParallelizationFactor float64
	Efficiency            float64
	EstimatedDuration     time.Duration
}
