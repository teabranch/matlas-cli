package apply

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/teabranch/matlas-cli/internal/apply/dag"
	"github.com/teabranch/matlas-cli/internal/types"
)

// Plan represents an ordered execution plan for applying configuration changes
type Plan struct {
	// Basic metadata
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   string    `json:"createdBy,omitempty"`
	Description string    `json:"description,omitempty"`

	// Plan content
	Operations []PlannedOperation `json:"operations"`
	Summary    PlanSummary        `json:"summary"`

	// Execution tracking
	Status       PlanStatus    `json:"status"`
	StartedAt    *time.Time    `json:"startedAt,omitempty"`
	CompletedAt  *time.Time    `json:"completedAt,omitempty"`
	LastError    string        `json:"lastError,omitempty"`
	ApprovalInfo *ApprovalInfo `json:"approvalInfo,omitempty"`

	// Configuration
	Config PlanConfig `json:"config"`
}

// PlannedOperation extends Operation with execution metadata
type PlannedOperation struct {
	Operation

	// Execution metadata
	ID           string   `json:"id"`
	Dependencies []string `json:"dependencies,omitempty"`
	Priority     int      `json:"priority"`
	Stage        int      `json:"stage"` // For parallel execution grouping

	// Execution tracking
	Status      OperationStatus `json:"status"`
	StartedAt   *time.Time      `json:"startedAt,omitempty"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
	Error       string          `json:"error,omitempty"`
	RetryCount  int             `json:"retryCount"`

	// Batching support
	BatchID   string `json:"batchId,omitempty"`
	BatchSize int    `json:"batchSize,omitempty"`
}

// PlanStatus represents the current status of a plan
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"     // Plan created but not approved
	PlanStatusApproved  PlanStatus = "approved"  // Plan approved for execution
	PlanStatusExecuting PlanStatus = "executing" // Plan currently being executed
	PlanStatusCompleted PlanStatus = "completed" // Plan successfully completed
	PlanStatusFailed    PlanStatus = "failed"    // Plan execution failed
	PlanStatusCancelled PlanStatus = "cancelled" // Plan execution was cancelled
	PlanStatusPartial   PlanStatus = "partial"   // Plan partially completed with some failures
)

// OperationStatus represents the current status of an operation
type OperationStatus string

const (
	OperationStatusPending   OperationStatus = "pending"   // Not yet started
	OperationStatusRunning   OperationStatus = "running"   // Currently executing
	OperationStatusCompleted OperationStatus = "completed" // Successfully completed
	OperationStatusFailed    OperationStatus = "failed"    // Failed to execute
	OperationStatusSkipped   OperationStatus = "skipped"   // Skipped due to dependency failure
	OperationStatusRetrying  OperationStatus = "retrying"  // Being retried after failure
)

// ApprovalInfo contains information about plan approval
type ApprovalInfo struct {
	Required    bool       `json:"required"`
	Approved    bool       `json:"approved"`
	ApprovedBy  string     `json:"approvedBy,omitempty"`
	ApprovedAt  *time.Time `json:"approvedAt,omitempty"`
	Comments    string     `json:"comments,omitempty"`
	AutoApprove bool       `json:"autoApprove"`
}

// PlanSummary provides high-level statistics about the plan
type PlanSummary struct {
	TotalOperations       int                   `json:"totalOperations"`
	OperationsByType      map[OperationType]int `json:"operationsByType"`
	OperationsByStage     map[int]int           `json:"operationsByStage"`
	EstimatedDuration     time.Duration         `json:"estimatedDuration"`
	HighestRiskLevel      RiskLevel             `json:"highestRiskLevel"`
	DestructiveOperations int                   `json:"destructiveOperations"`
	RequiresApproval      bool                  `json:"requiresApproval"`
	ParallelizationFactor float64               `json:"parallelizationFactor"`
}

// PlanConfig contains configuration options for plan execution
type PlanConfig struct {
	// Execution behavior
	FailFast       bool          `json:"failFast"`
	MaxParallelOps int           `json:"maxParallelOps"`
	DefaultTimeout time.Duration `json:"defaultTimeout"`
	RetryAttempts  int           `json:"retryAttempts"`

	// Safety settings
	RequireApproval      bool      `json:"requireApproval"`
	DryRunMode           bool      `json:"dryRunMode"`
	AutoApproveThreshold RiskLevel `json:"autoApproveThreshold"`

	// Progress tracking
	ShowProgress  bool `json:"showProgress"`
	VerboseOutput bool `json:"verboseOutput"`

	// DAG engine settings
	UseDAGEngine         bool   `json:"useDAGEngine"`         // Enable DAG-based dependency engine
	OptimizationStrategy string `json:"optimizationStrategy"` // speed, cost, reliability, balanced
	SchedulingStrategy   string `json:"schedulingStrategy"`   // greedy, critical_path_first, risk_based_early, etc.
}

// PlanBuilder helps construct execution plans
type PlanBuilder struct {
	projectID   string
	operations  []Operation
	config      PlanConfig
	dependGraph *types.DependencyGraph
}

// NewPlanBuilder creates a new plan builder
func NewPlanBuilder(projectID string) *PlanBuilder {
	return &PlanBuilder{
		projectID:   projectID,
		operations:  make([]Operation, 0),
		dependGraph: types.NewDependencyGraph(),
		config: PlanConfig{
			FailFast:             true,
			MaxParallelOps:       5,
			DefaultTimeout:       30 * time.Minute,
			RetryAttempts:        3,
			RequireApproval:      false,
			DryRunMode:           false,
			AutoApproveThreshold: RiskLevelMedium,
			ShowProgress:         true,
			VerboseOutput:        false,
		},
	}
}

// AddOperation adds an operation to the plan
func (pb *PlanBuilder) AddOperation(op Operation) *PlanBuilder {
	pb.operations = append(pb.operations, op)
	return pb
}

// AddOperations adds multiple operations to the plan
func (pb *PlanBuilder) AddOperations(ops []Operation) *PlanBuilder {
	pb.operations = append(pb.operations, ops...)
	return pb
}

// WithConfig sets the plan configuration
func (pb *PlanBuilder) WithConfig(config PlanConfig) *PlanBuilder {
	pb.config = config
	return pb
}

// WithMaxParallelOps sets the maximum parallel operations.
func (pb *PlanBuilder) WithMaxParallelOps(maxOps int) *PlanBuilder {
	pb.config.MaxParallelOps = maxOps
	return pb
}

// WithTimeout sets the default operation timeout
func (pb *PlanBuilder) WithTimeout(timeout time.Duration) *PlanBuilder {
	pb.config.DefaultTimeout = timeout
	return pb
}

// RequireApproval enables approval requirement for the plan
func (pb *PlanBuilder) RequireApproval(required bool) *PlanBuilder {
	pb.config.RequireApproval = required
	return pb
}

// WithDAGEngine enables the DAG-based dependency engine
func (pb *PlanBuilder) WithDAGEngine(enabled bool) *PlanBuilder {
	pb.config.UseDAGEngine = enabled
	return pb
}

// WithOptimizationStrategy sets the optimization strategy for the DAG engine
// Valid values: "speed", "cost", "reliability", "balanced"
func (pb *PlanBuilder) WithOptimizationStrategy(strategy string) *PlanBuilder {
	pb.config.OptimizationStrategy = strategy
	return pb
}

// WithSchedulingStrategy sets the scheduling strategy for the DAG engine
// Valid values: "greedy", "critical_path_first", "risk_based_early", "risk_based_late", "resource_leveling", "batch_optimized"
func (pb *PlanBuilder) WithSchedulingStrategy(strategy string) *PlanBuilder {
	pb.config.SchedulingStrategy = strategy
	return pb
}

// Build creates the execution plan
func (pb *PlanBuilder) Build() (*Plan, error) {
	if len(pb.operations) == 0 {
		return nil, fmt.Errorf("cannot create plan with no operations")
	}

	// Generate unique plan ID
	planID := fmt.Sprintf("plan-%d", time.Now().Unix())

	// Convert operations to planned operations
	plannedOps, err := pb.buildPlannedOperations()
	if err != nil {
		return nil, fmt.Errorf("failed to build planned operations: %w", err)
	}

	// Calculate plan summary
	summary := pb.calculateSummary(plannedOps)

	// Determine approval requirements
	approvalInfo := &ApprovalInfo{
		Required:    pb.config.RequireApproval || summary.RequiresApproval,
		AutoApprove: !pb.config.RequireApproval && summary.HighestRiskLevel <= pb.config.AutoApproveThreshold,
	}

	plan := &Plan{
		ID:           planID,
		ProjectID:    pb.projectID,
		CreatedAt:    time.Now(),
		Operations:   plannedOps,
		Summary:      summary,
		Status:       PlanStatusDraft,
		ApprovalInfo: approvalInfo,
		Config:       pb.config,
	}

	return plan, nil
}

// buildPlannedOperations converts operations to planned operations with dependencies and staging
func (pb *PlanBuilder) buildPlannedOperations() ([]PlannedOperation, error) {
	plannedOps := make([]PlannedOperation, len(pb.operations))

	for i, op := range pb.operations {
		plannedOp := PlannedOperation{
			Operation:  op,
			ID:         fmt.Sprintf("op-%d", i),
			Priority:   pb.calculatePriority(op),
			Status:     OperationStatusPending,
			RetryCount: 0,
		}

		// Add automatic dependencies based on resource types
		deps := pb.detectAutomaticDependencies(op, pb.operations[:i])
		plannedOp.Dependencies = deps

		plannedOps[i] = plannedOp
	}

	// Assign stages for parallel execution
	if err := pb.assignStages(plannedOps); err != nil {
		return nil, err
	}

	return plannedOps, nil
}

// calculatePriority determines operation priority based on type and risk
func (pb *PlanBuilder) calculatePriority(op Operation) int {
	priority := 100 // Default priority

	// Adjust based on operation type
	switch op.Type {
	case OperationCreate:
		priority += 10 // Create operations get higher priority
	case OperationUpdate:
		priority += 5
	case OperationDelete:
		priority -= 10 // Delete operations get lower priority
	}

	// Adjust based on risk level
	if op.Impact != nil {
		switch op.Impact.RiskLevel {
		case RiskLevelCritical:
			priority -= 20
		case RiskLevelHigh:
			priority -= 10
		case RiskLevelMedium:
			// No adjustment
		case RiskLevelLow:
			priority += 5
		}
	}

	// Adjust based on resource type (clusters should come before users)
	switch op.ResourceType {
	case types.KindProject:
		priority += 50
	case types.KindCluster:
		priority += 40
	case types.KindDatabaseRole:
		// Plan custom roles before users that may reference them
		priority += 35
	case types.KindNetworkAccess:
		priority += 30
	case types.KindDatabaseUser:
		priority += 20
	}

	return priority
}

// detectAutomaticDependencies identifies dependencies based on resource relationships
func (pb *PlanBuilder) detectAutomaticDependencies(op Operation, previousOps []Operation) []string {
	var deps []string

	// Database users depend on clusters
	if op.ResourceType == types.KindDatabaseUser {
		for i, prevOp := range previousOps {
			if prevOp.ResourceType == types.KindCluster {
				deps = append(deps, fmt.Sprintf("op-%d", i))
			}
			// Users also may depend on roles being available
			if prevOp.ResourceType == types.KindDatabaseRole {
				deps = append(deps, fmt.Sprintf("op-%d", i))
			}
		}
	}

	// Network access can be created before or after clusters, no strict dependency

	return deps
}

// assignStages groups operations into stages for parallel execution
func (pb *PlanBuilder) assignStages(ops []PlannedOperation) error {
	// Use DAG engine if enabled
	if pb.config.UseDAGEngine {
		return pb.assignStagesWithDAG(ops)
	}

	// Fall back to simple topological sort
	return pb.assignStagesSimple(ops)
}

// assignStagesSimple uses the simple topological sort (legacy behavior)
func (pb *PlanBuilder) assignStagesSimple(ops []PlannedOperation) error {
	// Build dependency map
	depMap := make(map[string][]string)
	for _, op := range ops {
		depMap[op.ID] = op.Dependencies
	}

	// Assign stages using topological sort
	assigned := make(map[string]int)
	stage := 0

	for len(assigned) < len(ops) {
		stageOps := make([]string, 0)

		// Find operations with no unassigned dependencies
		for _, op := range ops {
			if _, alreadyAssigned := assigned[op.ID]; alreadyAssigned {
				continue
			}

			canAssign := true
			for _, dep := range op.Dependencies {
				if _, depAssigned := assigned[dep]; !depAssigned {
					canAssign = false
					break
				}
			}

			if canAssign {
				stageOps = append(stageOps, op.ID)
			}
		}

		if len(stageOps) == 0 {
			return fmt.Errorf("circular dependency detected in operations")
		}

		// Assign stage to operations
		for _, opID := range stageOps {
			assigned[opID] = stage
		}

		stage++
	}

	// Update operations with stage assignments
	for i := range ops {
		ops[i].Stage = assigned[ops[i].ID]
	}

	return nil
}

// assignStagesWithDAG uses the DAG engine for optimized stage assignment
func (pb *PlanBuilder) assignStagesWithDAG(ops []PlannedOperation) error {
	// Build DAG graph from operations
	graph := pb.buildDAGFromOperations(ops)

	// Determine optimization strategy
	var optimizationStrategy dag.OptimizationStrategy
	switch pb.config.OptimizationStrategy {
	case "speed":
		optimizationStrategy = dag.OptimizeForSpeed
	case "cost":
		optimizationStrategy = dag.OptimizeForCost
	case "reliability":
		optimizationStrategy = dag.OptimizeForReliability
	case "balanced", "":
		optimizationStrategy = dag.OptimizeForBalance
	default:
		optimizationStrategy = dag.OptimizeForBalance
	}

	// Create optimizer and optimize graph
	scheduleConfig := dag.ScheduleConfig{
		Strategy:       pb.parseSchedulingStrategy(),
		MaxParallelOps: pb.config.MaxParallelOps,
	}
	optimizer := dag.NewOptimizer(optimizationStrategy, scheduleConfig)
	optimizedGraph, err := optimizer.Optimize(context.Background(), graph)
	if err != nil {
		return fmt.Errorf("failed to optimize graph: %w", err)
	}

	// Create scheduler and generate schedule
	scheduler := dag.NewScheduler(scheduleConfig)
	schedule, err := scheduler.Schedule(context.Background(), optimizedGraph)
	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	// Convert schedule stages back to operation stage assignments
	for stageIdx, stage := range schedule.Stages {
		for _, node := range stage {
			// Find operation with this node ID and assign stage
			for i := range ops {
				if ops[i].ID == node.ID {
					ops[i].Stage = stageIdx
					// Also update priority from DAG analysis
					ops[i].Priority = node.Properties.Priority
					break
				}
			}
		}
	}

	return nil
}

// buildDAGFromOperations converts operations to a DAG graph
func (pb *PlanBuilder) buildDAGFromOperations(ops []PlannedOperation) *dag.Graph {
	graph := dag.NewGraph(dag.GraphMetadata{
		Name:      "Execution Plan",
		ProjectID: pb.projectID,
		CreatedAt: time.Now(),
	})

	// Add all operations as nodes
	for _, op := range ops {
		props := dag.NodeProperties{
			EstimatedDuration: pb.estimateOperationDuration(op),
			RiskLevel:         pb.mapRiskLevel(op),
			IsDestructive:     pb.isDestructiveOperation(op),
			Priority:          op.Priority,
			Retriable:         true,
			Idempotent:        pb.isIdempotentOperation(op),
		}

		node := &dag.Node{
			ID:           op.ID,
			Name:         op.ResourceName,
			ResourceType: op.ResourceType,
			Properties:   props,
		}
		graph.AddNode(node)
	}

	// Add dependencies as edges
	for _, op := range ops {
		for _, depID := range op.Dependencies {
			edge := &dag.Edge{
				From:   op.ID,
				To:     depID,
				Type:   dag.DependencyTypeHard,
				Weight: 1.0,
			}
			graph.AddEdge(edge)
		}
	}

	return graph
}

// parseSchedulingStrategy converts string to SchedulingStrategy
func (pb *PlanBuilder) parseSchedulingStrategy() dag.SchedulingStrategy {
	switch pb.config.SchedulingStrategy {
	case "greedy":
		return dag.StrategyGreedy
	case "critical_path_first":
		return dag.StrategyCriticalPathFirst
	case "risk_based_early":
		return dag.StrategyRiskBasedEarly
	case "risk_based_late":
		return dag.StrategyRiskBasedLate
	case "resource_leveling":
		return dag.StrategyResourceLeveling
	case "batch_optimized":
		return dag.StrategyBatchOptimized
	default:
		return dag.StrategyGreedy
	}
}

// estimateOperationDuration estimates how long an operation will take
func (pb *PlanBuilder) estimateOperationDuration(op PlannedOperation) time.Duration {
	if op.Impact != nil {
		return op.Impact.EstimatedDuration
	}

	// Default estimates based on operation type
	switch op.Type {
	case OperationCreate:
		if op.ResourceType == types.KindCluster {
			return 10 * time.Minute // Cluster creation is slow
		}
		return 30 * time.Second
	case OperationUpdate:
		return 1 * time.Minute
	case OperationDelete:
		return 30 * time.Second
	default:
		return 5 * time.Second
	}
}

// mapRiskLevel maps apply.RiskLevel to dag.RiskLevel
func (pb *PlanBuilder) mapRiskLevel(op PlannedOperation) dag.RiskLevel {
	if op.Impact == nil {
		return dag.RiskLevelMedium
	}

	switch op.Impact.RiskLevel {
	case RiskLevelLow:
		return dag.RiskLevelLow
	case RiskLevelMedium:
		return dag.RiskLevelMedium
	case RiskLevelHigh:
		return dag.RiskLevelHigh
	case RiskLevelCritical:
		return dag.RiskLevelCritical
	default:
		return dag.RiskLevelMedium
	}
}

// isDestructiveOperation checks if an operation is destructive
func (pb *PlanBuilder) isDestructiveOperation(op PlannedOperation) bool {
	if op.Impact != nil {
		return op.Impact.IsDestructive
	}
	return op.Type == OperationDelete
}

// isIdempotentOperation checks if an operation is idempotent
func (pb *PlanBuilder) isIdempotentOperation(op PlannedOperation) bool {
	// Most create/update operations are idempotent
	// Delete operations are not (can't delete twice)
	return op.Type != OperationDelete
}

// calculateSummary generates plan summary statistics
func (pb *PlanBuilder) calculateSummary(ops []PlannedOperation) PlanSummary {
	summary := PlanSummary{
		TotalOperations:   len(ops),
		OperationsByType:  make(map[OperationType]int),
		OperationsByStage: make(map[int]int),
		HighestRiskLevel:  RiskLevelLow,
	}

	var totalDuration time.Duration
	var destructiveCount int
	var requiresApproval bool

	for _, op := range ops {
		// Count by type
		summary.OperationsByType[op.Type]++

		// Count by stage
		summary.OperationsByStage[op.Stage]++

		// Track impact
		if op.Impact != nil {
			totalDuration += op.Impact.EstimatedDuration

			if op.Impact.IsDestructive {
				destructiveCount++
			}

			if op.Impact.RiskLevel > summary.HighestRiskLevel {
				summary.HighestRiskLevel = op.Impact.RiskLevel
			}

			if op.Impact.RiskLevel >= RiskLevelHigh {
				requiresApproval = true
			}
		}
	}

	summary.EstimatedDuration = totalDuration
	summary.DestructiveOperations = destructiveCount
	summary.RequiresApproval = requiresApproval || destructiveCount > 0

	// Calculate parallelization factor
	stageCount := len(summary.OperationsByStage)
	if stageCount > 0 {
		summary.ParallelizationFactor = float64(summary.TotalOperations) / float64(stageCount)
	}

	return summary
}

// Approve approves the plan for execution
func (p *Plan) Approve(approvedBy string, comments string) error {
	if p.Status != PlanStatusDraft {
		return fmt.Errorf("cannot approve plan in status %s", p.Status)
	}

	if p.ApprovalInfo == nil {
		return fmt.Errorf("plan does not have approval info")
	}

	now := time.Now()
	p.ApprovalInfo.Approved = true
	p.ApprovalInfo.ApprovedBy = approvedBy
	p.ApprovalInfo.ApprovedAt = &now
	p.ApprovalInfo.Comments = comments
	p.Status = PlanStatusApproved

	return nil
}

// CanExecute checks if the plan can be executed
func (p *Plan) CanExecute() error {
	switch p.Status {
	case PlanStatusApproved:
		return nil
	case PlanStatusDraft:
		if p.ApprovalInfo != nil && p.ApprovalInfo.Required && !p.ApprovalInfo.Approved {
			return fmt.Errorf("plan requires approval before execution")
		}
		return nil
	case PlanStatusExecuting:
		return fmt.Errorf("plan is already executing")
	case PlanStatusCompleted:
		return fmt.Errorf("plan has already been executed")
	case PlanStatusFailed:
		return fmt.Errorf("plan execution failed")
	case PlanStatusCancelled:
		return fmt.Errorf("plan execution was cancelled")
	default:
		return fmt.Errorf("unknown plan status: %s", p.Status)
	}
}

// GetOperationsInStage returns all operations for a given stage
func (p *Plan) GetOperationsInStage(stage int) []PlannedOperation {
	var ops []PlannedOperation
	for _, op := range p.Operations {
		if op.Stage == stage {
			ops = append(ops, op)
		}
	}
	return ops
}

// GetMaxStage returns the highest stage number in the plan
func (p *Plan) GetMaxStage() int {
	maxStage := -1
	for _, op := range p.Operations {
		if op.Stage > maxStage {
			maxStage = op.Stage
		}
	}
	return maxStage
}

// ToJSON serializes the plan to JSON
func (p *Plan) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// FromJSON deserializes a plan from JSON
func FromJSON(data []byte) (*Plan, error) {
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}
	return &plan, nil
}

// PlanExecutor defines the interface for executing plans
type PlanExecutor interface {
	// Execute executes the entire plan
	Execute(ctx context.Context, plan *Plan) error

	// ExecuteOperation executes a single operation
	ExecuteOperation(ctx context.Context, op *PlannedOperation) error

	// GetProgress returns the current execution progress
	GetProgress() ExecutionProgress
}

// ExecutionProgress provides real-time execution progress information
type ExecutionProgress struct {
	PlanID              string        `json:"planId"`
	CurrentStage        int           `json:"currentStage"`
	TotalStages         int           `json:"totalStages"`
	CompletedOperations int           `json:"completedOperations"`
	TotalOperations     int           `json:"totalOperations"`
	FailedOperations    int           `json:"failedOperations"`
	ElapsedTime         time.Duration `json:"elapsedTime"`
	EstimatedTimeLeft   time.Duration `json:"estimatedTimeLeft"`
	CurrentOperation    string        `json:"currentOperation,omitempty"`
}

// ProgressPercentage calculates the completion percentage
func (ep *ExecutionProgress) ProgressPercentage() float64 {
	if ep.TotalOperations == 0 {
		return 0
	}
	return float64(ep.CompletedOperations) / float64(ep.TotalOperations) * 100
}
