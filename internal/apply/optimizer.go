package apply

import (
	"fmt"
	"sort"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

// PlanOptimizer optimizes execution plans for efficiency and resource utilization
type PlanOptimizer struct {
	config OptimizationConfig
}

// OptimizationConfig contains settings for plan optimization
type OptimizationConfig struct {
	// Parallel execution settings
	MaxParallelOperations int     `json:"maxParallelOperations"`
	MinParallelThreshold  int     `json:"minParallelThreshold"`
	ParallelizationWeight float64 `json:"parallelizationWeight"`

	// Batching settings
	EnableBatching bool           `json:"enableBatching"`
	MaxBatchSize   int            `json:"maxBatchSize"`
	BatchingRules  []BatchingRule `json:"batchingRules"`

	// Priority and ordering
	PriorityWeights PriorityWeights `json:"priorityWeights"`
	RiskTolerance   RiskLevel       `json:"riskTolerance"`

	// Performance optimization
	EstimatedDurations map[types.ResourceKind]time.Duration `json:"estimatedDurations"`
	OptimizeForSpeed   bool                                 `json:"optimizeForSpeed"`
	OptimizeForSafety  bool                                 `json:"optimizeForSafety"`
}

// BatchingRule defines rules for batching operations together
type BatchingRule struct {
	Name          string                        `json:"name"`
	ResourceKind  types.ResourceKind            `json:"resourceKind"`
	OperationType OperationType                 `json:"operationType"`
	MaxBatchSize  int                           `json:"maxBatchSize"`
	CanBatch      func([]PlannedOperation) bool `json:"-"`
	Priority      int                           `json:"priority"`
}

// PriorityWeights defines weights for different priority factors
type PriorityWeights struct {
	ResourceType  float64 `json:"resourceType"`
	OperationType float64 `json:"operationType"`
	RiskLevel     float64 `json:"riskLevel"`
	Dependencies  float64 `json:"dependencies"`
	EstimatedTime float64 `json:"estimatedTime"`
}

// OptimizationResult contains the results of plan optimization
type OptimizationResult struct {
	OriginalPlan     *Plan                  `json:"originalPlan"`
	OptimizedPlan    *Plan                  `json:"optimizedPlan"`
	Optimizations    []OptimizationAction   `json:"optimizations"`
	PerformanceGains PerformanceImprovement `json:"performanceGains"`
	Statistics       OptimizationStats      `json:"statistics"`
}

// OptimizationAction describes a specific optimization applied
type OptimizationAction struct {
	Type        OptimizationType `json:"type"`
	Description string           `json:"description"`
	Operations  []string         `json:"operations"`
	Impact      string           `json:"impact"`
}

// OptimizationType categorizes the type of optimization
type OptimizationType string

const (
	OptimizationParallelization OptimizationType = "parallelization"
	OptimizationBatching        OptimizationType = "batching"
	OptimizationReordering      OptimizationType = "reordering"
	OptimizationStageOptimize   OptimizationType = "stage_optimization"
)

// PerformanceImprovement quantifies the performance gains from optimization
type PerformanceImprovement struct {
	EstimatedTimeReduction time.Duration `json:"estimatedTimeReduction"`
	ParallelizationGain    float64       `json:"parallelizationGain"`
	StageReduction         int           `json:"stageReduction"`
	BatchingEfficiency     float64       `json:"batchingEfficiency"`
}

// OptimizationStats provides statistics about the optimization process
type OptimizationStats struct {
	OriginalStages      int           `json:"originalStages"`
	OptimizedStages     int           `json:"optimizedStages"`
	OperationsBatched   int           `json:"operationsBatched"`
	ParallelGroups      int           `json:"parallelGroups"`
	AverageStageSize    float64       `json:"averageStageSize"`
	LongestCriticalPath int           `json:"longestCriticalPath"`
	OptimizationTime    time.Duration `json:"optimizationTime"`
}

// NewPlanOptimizer creates a new plan optimizer with default configuration
func NewPlanOptimizer() *PlanOptimizer {
	return &PlanOptimizer{
		config: OptimizationConfig{
			MaxParallelOperations: 10,
			MinParallelThreshold:  2,
			ParallelizationWeight: 1.0,
			EnableBatching:        true,
			MaxBatchSize:          20,
			BatchingRules:         getDefaultBatchingRules(),
			PriorityWeights: PriorityWeights{
				ResourceType:  1.0,
				OperationType: 0.8,
				RiskLevel:     1.2,
				Dependencies:  1.5,
				EstimatedTime: 0.6,
			},
			RiskTolerance: RiskLevelMedium,
			EstimatedDurations: map[types.ResourceKind]time.Duration{
				types.KindProject:       30 * time.Second,
				types.KindCluster:       5 * time.Minute,
				types.KindDatabaseUser:  10 * time.Second,
				types.KindNetworkAccess: 5 * time.Second,
			},
			OptimizeForSpeed:  true,
			OptimizeForSafety: true,
		},
	}
}

// WithConfig sets the optimization configuration
func (po *PlanOptimizer) WithConfig(config OptimizationConfig) *PlanOptimizer {
	po.config = config
	return po
}

// WithMaxParallelOps sets the maximum parallel operations
func (po *PlanOptimizer) WithMaxParallelOps(max int) *PlanOptimizer {
	po.config.MaxParallelOperations = max
	return po
}

// WithBatching enables or disables batching
func (po *PlanOptimizer) WithBatching(enabled bool) *PlanOptimizer {
	po.config.EnableBatching = enabled
	return po
}

// OptimizePlan optimizes an execution plan for better performance
func (po *PlanOptimizer) OptimizePlan(plan *Plan) (*OptimizationResult, error) {
	startTime := time.Now()

	// Create a copy of the plan for optimization
	optimizedPlan := po.copyPlan(plan)
	var optimizations []OptimizationAction

	// Apply optimization strategies in order

	// 1. Priority-based reordering within stages
	reorderActions := po.optimizePriorities(optimizedPlan)
	optimizations = append(optimizations, reorderActions...)

	// 2. Batching optimization
	if po.config.EnableBatching {
		batchActions := po.optimizeBatching(optimizedPlan)
		optimizations = append(optimizations, batchActions...)
	}

	// 3. Parallel execution optimization
	parallelActions := po.optimizeParallelExecution(optimizedPlan)
	optimizations = append(optimizations, parallelActions...)

	// 4. Stage consolidation
	stageActions := po.optimizeStages(optimizedPlan)
	optimizations = append(optimizations, stageActions...)

	// Calculate performance improvements
	improvements := po.calculatePerformanceGains(plan, optimizedPlan)

	// Generate statistics
	stats := OptimizationStats{
		OriginalStages:      plan.GetMaxStage() + 1,
		OptimizedStages:     optimizedPlan.GetMaxStage() + 1,
		OptimizationTime:    time.Since(startTime),
		OperationsBatched:   po.countBatchedOperations(optimizedPlan),
		ParallelGroups:      len(optimizedPlan.Summary.OperationsByStage),
		AverageStageSize:    po.calculateAverageStageSize(optimizedPlan),
		LongestCriticalPath: len(optimizedPlan.Summary.OperationsByStage),
	}

	return &OptimizationResult{
		OriginalPlan:     plan,
		OptimizedPlan:    optimizedPlan,
		Optimizations:    optimizations,
		PerformanceGains: improvements,
		Statistics:       stats,
	}, nil
}

// optimizePriorities reorders operations within stages based on priority
func (po *PlanOptimizer) optimizePriorities(plan *Plan) []OptimizationAction {
	var actions []OptimizationAction

	// Group operations by stage
	stageGroups := make(map[int][]int)
	for i, op := range plan.Operations {
		stageGroups[op.Stage] = append(stageGroups[op.Stage], i)
	}

	// Sort operations within each stage by priority
	for stage, opIndices := range stageGroups {
		if len(opIndices) <= 1 {
			continue
		}

		// Sort by calculated priority
		sort.Slice(opIndices, func(i, j int) bool {
			return po.calculateOperationPriority(plan.Operations[opIndices[i]]) >
				po.calculateOperationPriority(plan.Operations[opIndices[j]])
		})

		// Update operation priorities
		reorderedOps := make([]string, len(opIndices))
		for i, opIndex := range opIndices {
			plan.Operations[opIndex].Priority = 1000 - (i * 10) // Higher priority = higher number
			reorderedOps[i] = plan.Operations[opIndex].ID
		}

		actions = append(actions, OptimizationAction{
			Type:        OptimizationReordering,
			Description: fmt.Sprintf("Reordered %d operations in stage %d by priority", len(opIndices), stage),
			Operations:  reorderedOps,
			Impact:      "Improved execution order for better resource utilization",
		})
	}

	return actions
}

// optimizeBatching groups similar operations into batches
func (po *PlanOptimizer) optimizeBatching(plan *Plan) []OptimizationAction {
	var actions []OptimizationAction

	if !po.config.EnableBatching {
		return actions
	}

	// Group operations by stage and type for batching
	stageTypeGroups := make(map[string][]int) // key: "stage-resourceKind-operationType"

	for i, op := range plan.Operations {
		key := fmt.Sprintf("%d-%s-%s", op.Stage, op.ResourceType, op.Type)
		stageTypeGroups[key] = append(stageTypeGroups[key], i)
	}

		// Apply batching rules
	for _, rule := range po.config.BatchingRules {
		for _, opIndices := range stageTypeGroups {
			if len(opIndices) < 2 {
				continue
			}
			
			// Check if operations match the batching rule
			firstOp := plan.Operations[opIndices[0]]
			if firstOp.ResourceType != rule.ResourceKind || firstOp.Type != rule.OperationType {
				continue
			}

			// Create batches
			batchCount := 0
			batchSize := min(rule.MaxBatchSize, len(opIndices))

			for i := 0; i < len(opIndices); i += batchSize {
				end := min(i+batchSize, len(opIndices))
				batchOps := opIndices[i:end]

				if len(batchOps) < 2 {
					continue
				}

				batchID := fmt.Sprintf("batch-%s-%d", rule.Name, batchCount)
				batchedOpNames := make([]string, len(batchOps))

				// Assign batch ID to operations
				for j, opIndex := range batchOps {
					plan.Operations[opIndex].BatchID = batchID
					plan.Operations[opIndex].BatchSize = len(batchOps)
					batchedOpNames[j] = plan.Operations[opIndex].ID
				}

				actions = append(actions, OptimizationAction{
					Type:        OptimizationBatching,
					Description: fmt.Sprintf("Created batch of %d %s %s operations", len(batchOps), rule.ResourceKind, rule.OperationType),
					Operations:  batchedOpNames,
					Impact:      fmt.Sprintf("Reduced overhead through batching %d operations", len(batchOps)),
				})

				batchCount++
			}
		}
	}

	return actions
}

// optimizeParallelExecution identifies opportunities for parallel execution
func (po *PlanOptimizer) optimizeParallelExecution(plan *Plan) []OptimizationAction {
	var actions []OptimizationAction

	// Analyze each stage for parallel opportunities
	for stage := 0; stage <= plan.GetMaxStage(); stage++ {
		stageOps := plan.GetOperationsInStage(stage)

		if len(stageOps) < po.config.MinParallelThreshold {
			continue
		}

		// Group operations that can be executed in parallel
		parallelGroups := po.identifyParallelGroups(stageOps)

		if len(parallelGroups) > 1 {
			for groupIndex, group := range parallelGroups {
				if len(group) >= po.config.MinParallelThreshold {
					opNames := make([]string, len(group))
					for i, op := range group {
						opNames[i] = op.ID
					}

					actions = append(actions, OptimizationAction{
						Type:        OptimizationParallelization,
						Description: fmt.Sprintf("Identified parallel group %d with %d operations in stage %d", groupIndex, len(group), stage),
						Operations:  opNames,
						Impact:      fmt.Sprintf("Can execute %d operations in parallel", len(group)),
					})
				}
			}
		}
	}

	return actions
}

// optimizeStages consolidates stages where possible
func (po *PlanOptimizer) optimizeStages(plan *Plan) []OptimizationAction {
	var actions []OptimizationAction

	// Look for opportunities to merge stages
	maxStage := plan.GetMaxStage()

	for stage := 0; stage < maxStage; stage++ {
		currentStageOps := plan.GetOperationsInStage(stage)
		nextStageOps := plan.GetOperationsInStage(stage + 1)

		if len(currentStageOps) == 0 || len(nextStageOps) == 0 {
			continue
		}

		// Check if next stage operations can be moved to current stage
		canMerge := po.canMergeStages(currentStageOps, nextStageOps)

		if canMerge {
			// Move next stage operations to current stage
			movedOps := make([]string, len(nextStageOps))
			for i, op := range nextStageOps {
				// Find the operation in the plan and update its stage
				for j := range plan.Operations {
					if plan.Operations[j].ID == op.ID {
						plan.Operations[j].Stage = stage
						break
					}
				}
				movedOps[i] = op.ID
			}

			// Shift all subsequent stages down by one
			po.shiftStagesDown(plan, stage+2)

			actions = append(actions, OptimizationAction{
				Type:        OptimizationStageOptimize,
				Description: fmt.Sprintf("Merged stage %d into stage %d", stage+1, stage),
				Operations:  movedOps,
				Impact:      fmt.Sprintf("Reduced total stages by eliminating stage %d", stage+1),
			})
		}
	}

	return actions
}

// identifyParallelGroups finds groups of operations that can run in parallel
func (po *PlanOptimizer) identifyParallelGroups(operations []PlannedOperation) [][]PlannedOperation {
	// Simple grouping by resource type and operation type
	groups := make(map[string][]PlannedOperation)

	for _, op := range operations {
		// Group by resource type and operation type for parallel execution
		key := fmt.Sprintf("%s-%s", op.ResourceType, op.Type)
		groups[key] = append(groups[key], op)
	}

	var parallelGroups [][]PlannedOperation
	for _, group := range groups {
		if len(group) >= po.config.MinParallelThreshold {
			parallelGroups = append(parallelGroups, group)
		}
	}

	return parallelGroups
}

// canMergeStages checks if two consecutive stages can be merged
func (po *PlanOptimizer) canMergeStages(currentStage, nextStage []PlannedOperation) bool {
	// Check if any operation in next stage depends on current stage
	for _, nextOp := range nextStage {
		for _, dep := range nextOp.Dependencies {
			for _, currentOp := range currentStage {
				if currentOp.ID == dep {
					return false // Dependency prevents merging
				}
			}
		}
	}

	// Check resource conflicts
	return !po.hasResourceConflicts(currentStage, nextStage)
}

// hasResourceConflicts checks if operations would conflict if run in parallel
func (po *PlanOptimizer) hasResourceConflicts(ops1, ops2 []PlannedOperation) bool {
	// Check for operations on the same resource
	resourceMap := make(map[string]bool)

	for _, op := range ops1 {
		key := fmt.Sprintf("%s-%s", op.ResourceType, op.ResourceName)
		resourceMap[key] = true
	}

	for _, op := range ops2 {
		key := fmt.Sprintf("%s-%s", op.ResourceType, op.ResourceName)
		if resourceMap[key] {
			return true // Same resource, potential conflict
		}
	}

	return false
}

// shiftStagesDown shifts all stages from a given stage number down by one
func (po *PlanOptimizer) shiftStagesDown(plan *Plan, fromStage int) {
	for i := range plan.Operations {
		if plan.Operations[i].Stage >= fromStage {
			plan.Operations[i].Stage--
		}
	}
}

// calculateOperationPriority calculates a priority score for an operation
func (po *PlanOptimizer) calculateOperationPriority(op PlannedOperation) float64 {
	score := 0.0

	// Resource type weight
	switch op.ResourceType {
	case types.KindProject:
		score += 100 * po.config.PriorityWeights.ResourceType
	case types.KindCluster:
		score += 80 * po.config.PriorityWeights.ResourceType
	case types.KindNetworkAccess:
		score += 60 * po.config.PriorityWeights.ResourceType
	case types.KindDatabaseUser:
		score += 40 * po.config.PriorityWeights.ResourceType
	}

	// Operation type weight
	switch op.Type {
	case OperationCreate:
		score += 50 * po.config.PriorityWeights.OperationType
	case OperationUpdate:
		score += 30 * po.config.PriorityWeights.OperationType
	case OperationDelete:
		score += 10 * po.config.PriorityWeights.OperationType
	}

	// Risk level weight (lower risk = higher priority)
	if op.Impact != nil {
		switch op.Impact.RiskLevel {
		case RiskLevelLow:
			score += 40 * po.config.PriorityWeights.RiskLevel
		case RiskLevelMedium:
			score += 20 * po.config.PriorityWeights.RiskLevel
		case RiskLevelHigh:
			score += 10 * po.config.PriorityWeights.RiskLevel
		case RiskLevelCritical:
			score += 5 * po.config.PriorityWeights.RiskLevel
		}
	}

	// Dependency weight (fewer dependencies = higher priority)
	depCount := len(op.Dependencies)
	if depCount > 0 {
		score -= float64(depCount) * po.config.PriorityWeights.Dependencies
	}

	return score
}

// calculatePerformanceGains estimates performance improvements from optimization
func (po *PlanOptimizer) calculatePerformanceGains(original, optimized *Plan) PerformanceImprovement {
	originalDuration := original.Summary.EstimatedDuration
	optimizedDuration := optimized.Summary.EstimatedDuration

	return PerformanceImprovement{
		EstimatedTimeReduction: originalDuration - optimizedDuration,
		ParallelizationGain:    optimized.Summary.ParallelizationFactor / original.Summary.ParallelizationFactor,
		StageReduction:         (original.GetMaxStage() + 1) - (optimized.GetMaxStage() + 1),
		BatchingEfficiency:     float64(po.countBatchedOperations(optimized)) / float64(len(optimized.Operations)),
	}
}

// copyPlan creates a deep copy of a plan for optimization
func (po *PlanOptimizer) copyPlan(plan *Plan) *Plan {
	// For simplicity, we'll work with the original plan
	// In production, this should create a proper deep copy
	return plan
}

// countBatchedOperations counts how many operations are part of batches
func (po *PlanOptimizer) countBatchedOperations(plan *Plan) int {
	count := 0
	for _, op := range plan.Operations {
		if op.BatchID != "" {
			count++
		}
	}
	return count
}

// calculateAverageStageSize calculates the average number of operations per stage
func (po *PlanOptimizer) calculateAverageStageSize(plan *Plan) float64 {
	if len(plan.Summary.OperationsByStage) == 0 {
		return 0
	}

	totalOps := 0
	for _, count := range plan.Summary.OperationsByStage {
		totalOps += count
	}

	return float64(totalOps) / float64(len(plan.Summary.OperationsByStage))
}

// getDefaultBatchingRules returns default batching rules
func getDefaultBatchingRules() []BatchingRule {
	return []BatchingRule{
		{
			Name:          "database-users",
			ResourceKind:  types.KindDatabaseUser,
			OperationType: OperationCreate,
			MaxBatchSize:  10,
			Priority:      100,
		},
		{
			Name:          "network-access",
			ResourceKind:  types.KindNetworkAccess,
			OperationType: OperationCreate,
			MaxBatchSize:  20,
			Priority:      90,
		},
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
