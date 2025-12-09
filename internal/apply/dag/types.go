package dag

import (
	"sync"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

// DependencyType represents the type of dependency relationship
type DependencyType string

const (
	// DependencyTypeHard - Must complete before dependent can start
	DependencyTypeHard DependencyType = "hard"

	// DependencyTypeSoft - Preferred order but not required
	DependencyTypeSoft DependencyType = "soft"

	// DependencyTypeConditional - Depends on resource properties or runtime state
	DependencyTypeConditional DependencyType = "conditional"

	// DependencyTypeMutualExclusion - Cannot run in parallel
	DependencyTypeMutualExclusion DependencyType = "mutual_exclusion"

	// DependencyTypeOrdering - Relative ordering without strict dependencies
	DependencyTypeOrdering DependencyType = "ordering"

	// DependencyTypeResource - Depends on resource availability (e.g., API rate limits)
	DependencyTypeResource DependencyType = "resource"
)

// Node represents a node in the DAG (an operation)
type Node struct {
	// Identity
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	ResourceType types.ResourceKind `json:"resourceType"`

	// Properties
	Properties NodeProperties    `json:"properties"`
	Labels     map[string]string `json:"labels,omitempty"`

	// Dependencies (outgoing edges)
	Dependencies []*Edge `json:"dependencies,omitempty"`

	// Computed properties (set by algorithms)
	Level         int           `json:"level"`         // Dependency level (0 = no deps)
	EarliestStart time.Duration `json:"earliestStart"` // CPM earliest start time
	LatestStart   time.Duration `json:"latestStart"`   // CPM latest start time
	Slack         time.Duration `json:"slack"`         // Slack time (LS - ES)
	IsCritical    bool          `json:"isCritical"`    // On critical path
}

// NodeProperties contains operation-specific properties
type NodeProperties struct {
	// Execution properties
	EstimatedDuration time.Duration `json:"estimatedDuration"`
	MinDuration       time.Duration `json:"minDuration,omitempty"`
	MaxDuration       time.Duration `json:"maxDuration,omitempty"`

	// Resource requirements
	ResourceRequirements ResourceRequirements `json:"resourceRequirements,omitempty"`

	// Risk assessment
	RiskLevel     RiskLevel `json:"riskLevel"`
	IsDestructive bool      `json:"isDestructive"`

	// Execution hints
	Priority   int  `json:"priority"`
	Retriable  bool `json:"retriable"`
	Idempotent bool `json:"idempotent"`

	// Cost estimation
	Cost float64 `json:"cost,omitempty"` // Arbitrary cost metric
}

// ResourceRequirements defines resource needs for an operation
type ResourceRequirements struct {
	// Concurrency limits
	MaxParallelOps int `json:"maxParallelOps,omitempty"`

	// API quota usage
	APICallsRequired int `json:"apiCallsRequired,omitempty"`

	// Memory/CPU (for future use)
	MemoryMB int `json:"memoryMB,omitempty"`
	CPUCores int `json:"cpuCores,omitempty"`
}

// RiskLevel represents operation risk level
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// Edge represents a directed edge (dependency) in the DAG
type Edge struct {
	// Identity
	From string         `json:"from"` // Source node ID
	To   string         `json:"to"`   // Target node ID
	Type DependencyType `json:"type"`

	// Properties
	Weight    float64           `json:"weight"`              // Edge weight (higher = stronger dependency)
	Condition *Condition        `json:"condition,omitempty"` // For conditional dependencies
	Reason    string            `json:"reason,omitempty"`    // Human-readable reason
	Metadata  map[string]string `json:"metadata,omitempty"`

	// Computed properties
	IsCritical bool `json:"isCritical"` // On critical path
}

// Condition represents a conditional dependency expression
type Condition struct {
	// Property-based condition
	PropertyPath string      `json:"propertyPath,omitempty"` // e.g., "spec.provider"
	Operator     string      `json:"operator,omitempty"`     // e.g., "==", "!=", "contains"
	Value        interface{} `json:"value,omitempty"`

	// Logical operators
	And []*Condition `json:"and,omitempty"`
	Or  []*Condition `json:"or,omitempty"`
	Not *Condition   `json:"not,omitempty"`

	// Runtime condition (evaluated at execution time)
	RuntimeEvaluator func() bool `json:"-"`
}

// Graph represents a directed acyclic graph of operations
type Graph struct {
	// Nodes
	Nodes map[string]*Node `json:"nodes"`

	// Edges (adjacency list)
	Edges map[string][]*Edge `json:"edges"` // from -> list of edges

	// Reverse edges (for dependency tracking)
	ReverseEdges map[string][]*Edge `json:"reverseEdges"` // to -> list of edges

	// Metadata
	Metadata GraphMetadata `json:"metadata"`

	// Computed properties
	CriticalPath  []string      `json:"criticalPath,omitempty"`
	TotalDuration time.Duration `json:"totalDuration,omitempty"`
	MaxLevel      int           `json:"maxLevel,omitempty"`

	// Concurrency control (not exported to JSON)
	mu sync.RWMutex `json:"-"`
}

// GraphMetadata contains metadata about the graph
type GraphMetadata struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	ProjectID   string            `json:"projectId,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// SchedulingStrategy defines how operations should be scheduled
type SchedulingStrategy string

const (
	// StrategyGreedy - Maximize parallel operations at each stage
	StrategyGreedy SchedulingStrategy = "greedy"

	// StrategyCriticalPathFirst - Execute critical path operations first
	StrategyCriticalPathFirst SchedulingStrategy = "critical_path_first"

	// StrategyRiskBasedEarly - High-risk operations early (fail-fast)
	StrategyRiskBasedEarly SchedulingStrategy = "risk_based_early"

	// StrategyRiskBasedLate - High-risk operations late (minimize disruption)
	StrategyRiskBasedLate SchedulingStrategy = "risk_based_late"

	// StrategyResourceLeveling - Balance resource usage across stages
	StrategyResourceLeveling SchedulingStrategy = "resource_leveling"

	// StrategyBatchOptimized - Group similar operations for efficiency
	StrategyBatchOptimized SchedulingStrategy = "batch_optimized"
)

// ScheduleConfig contains configuration for scheduling
type ScheduleConfig struct {
	Strategy          SchedulingStrategy `json:"strategy"`
	MaxParallelOps    int                `json:"maxParallelOps"`
	MaxAPICallsPerSec int                `json:"maxApiCallsPerSec,omitempty"`
	PreferIdempotent  bool               `json:"preferIdempotent"`

	// Resource constraints
	MaxMemoryMB int `json:"maxMemoryMB,omitempty"`
	MaxCPUCores int `json:"maxCpuCores,omitempty"`
}

// Schedule represents an optimized execution schedule
type Schedule struct {
	// Stages of execution (each stage can run in parallel)
	Stages [][]*Node `json:"stages"`

	// Metadata
	Strategy      SchedulingStrategy `json:"strategy"`
	TotalDuration time.Duration      `json:"totalDuration"`
	CriticalPath  []string           `json:"criticalPath,omitempty"`

	// Scheduler-specific fields
	EstimatedDuration time.Duration `json:"estimatedDuration"`
	MaxParallelOps    int           `json:"maxParallelOps"`
	CreatedAt         time.Time     `json:"createdAt"`

	// Metrics
	Metrics ScheduleMetrics `json:"metrics,omitempty"`
}

// ScheduleMetrics contains metrics about the schedule
type ScheduleMetrics struct {
	TotalOperations       int           `json:"totalOperations"`
	TotalStages           int           `json:"totalStages"`
	ParallelizationFactor float64       `json:"parallelizationFactor"` // ops / stages
	AverageStageSize      float64       `json:"averageStageSize"`
	MaxStageSize          int           `json:"maxStageSize"`
	CriticalPathLength    int           `json:"criticalPathLength"`
	EstimatedDuration     time.Duration `json:"estimatedDuration"`

	// Resource utilization
	AvgParallelOps float64 `json:"avgParallelOps"`
	MaxParallelOps int     `json:"maxParallelOps"`
}

// AnalysisResult contains the results of dependency analysis
type AnalysisResult struct {
	// Graph properties
	NodeCount int        `json:"nodeCount"`
	EdgeCount int        `json:"edgeCount"`
	HasCycles bool       `json:"hasCycles"`
	Cycles    [][]string `json:"cycles,omitempty"`

	// Dependency levels
	Levels   map[string]int `json:"levels"`
	MaxLevel int            `json:"maxLevel"`

	// Critical path
	CriticalPath         []string      `json:"criticalPath"`
	CriticalPathDuration time.Duration `json:"criticalPathDuration"`

	// Parallelization
	ParallelGroups        [][]*Node `json:"parallelGroups"`
	ParallelizationFactor float64   `json:"parallelizationFactor"`

	// Bottlenecks
	Bottlenecks []*BottleneckInfo `json:"bottlenecks,omitempty"`

	// Risk analysis
	RiskAnalysis *RiskAnalysisResult `json:"riskAnalysis,omitempty"`

	// Optimization suggestions
	Suggestions []string `json:"suggestions,omitempty"`
}

// BottleneckInfo describes a bottleneck in the graph
type BottleneckInfo struct {
	NodeID       string   `json:"nodeId"`
	NodeName     string   `json:"nodeName"`
	BlockedNodes []string `json:"blockedNodes"` // Nodes that depend on this
	BlockedCount int      `json:"blockedCount"`
	Impact       float64  `json:"impact"` // 0.0 to 1.0
	Reason       string   `json:"reason"`
	Mitigation   string   `json:"mitigation,omitempty"`
}

// RiskAnalysisResult contains risk analysis information
type RiskAnalysisResult struct {
	HighRiskOperations     []*Node           `json:"highRiskOperations"`
	CriticalRiskOperations []*Node           `json:"criticalRiskOperations"` // High risk on critical path
	TotalRiskScore         float64           `json:"totalRiskScore"`
	AverageRiskLevel       RiskLevel         `json:"averageRiskLevel"`
	RiskByLevel            map[RiskLevel]int `json:"riskByLevel"`
}

// WhatIfScenario represents a what-if analysis scenario
type WhatIfScenario struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// Changes to apply
	AddNodes    []*Node  `json:"addNodes,omitempty"`
	RemoveNodes []string `json:"removeNodes,omitempty"`
	AddEdges    []*Edge  `json:"addEdges,omitempty"`
	RemoveEdges []*Edge  `json:"removeEdges,omitempty"`

	// Results (populated after analysis)
	Result *WhatIfResult `json:"result,omitempty"`
}

// WhatIfResult contains the results of a what-if analysis
type WhatIfResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`

	// Impact metrics
	DurationChange    time.Duration `json:"durationChange"`
	StageCountChange  int           `json:"stageCountChange"`
	ParallelismChange float64       `json:"parallelismChange"`
	RiskChange        float64       `json:"riskChange"`

	// New critical path
	NewCriticalPath []string `json:"newCriticalPath,omitempty"`

	// Comparison
	Comparison string `json:"comparison,omitempty"`
}
