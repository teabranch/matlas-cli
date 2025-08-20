package apply

import (
	"context"
	"fmt"
	"time"

	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
)

// EnhancedExecutor provides an enhanced execution engine with idempotency and recovery
type EnhancedExecutor struct {
	// Base executor
	baseExecutor Executor

	// Enhancement systems
	idempotencyManager *IdempotencyManager
	recoveryManager    *RecoveryManager

	// Configuration
	config EnhancedExecutorConfig
}

// EnhancedExecutorConfig contains configuration for the enhanced executor
type EnhancedExecutorConfig struct {
	// Base executor config
	BaseConfig ExecutorConfig `json:"baseConfig"`

	// Idempotency settings
	IdempotencyConfig IdempotencyConfig `json:"idempotencyConfig"`

	// Recovery settings
	RecoveryConfig RecoveryConfig `json:"recoveryConfig"`

	// Enhancement settings
	EnableIdempotencyChecks bool `json:"enableIdempotencyChecks"`
	EnableRecovery          bool `json:"enableRecovery"`
	CreateCheckpoints       bool `json:"createCheckpoints"`
	SkipIdempotentOps       bool `json:"skipIdempotentOps"`
}

// NewEnhancedExecutor creates a new enhanced executor
func NewEnhancedExecutor(
	clustersService *atlas.ClustersService,
	usersService *atlas.DatabaseUsersService,
	networkAccessService *atlas.NetworkAccessListsService,
	projectsService *atlas.ProjectsService,
	searchService *atlas.SearchService,
	vpcEndpointsService *atlas.VPCEndpointsService,
	databaseService *database.Service,
	config EnhancedExecutorConfig,
) *EnhancedExecutor {
	// Create base executor
	baseExecutor := &AtlasExecutor{
		clustersService:      clustersService,
		usersService:         usersService,
		networkAccessService: networkAccessService,
		projectsService:      projectsService,
		searchService:        searchService,
		vpcEndpointsService:  vpcEndpointsService,
		databaseService:      databaseService,
		retryManager:         NewRetryManager(config.BaseConfig.RetryConfig),
		config:               config.BaseConfig,
	}

	// Create idempotency manager
	idempotencyManager := NewIdempotencyManager(config.IdempotencyConfig)

	// Create recovery manager
	recoveryManager := NewRecoveryManager(
		clustersService,
		usersService,
		networkAccessService,
		projectsService,
		databaseService,
		idempotencyManager,
		config.RecoveryConfig,
	)

	return &EnhancedExecutor{
		baseExecutor:       baseExecutor,
		idempotencyManager: idempotencyManager,
		recoveryManager:    recoveryManager,
		config:             config,
	}
}

// DefaultEnhancedExecutorConfig returns a default enhanced executor configuration
func DefaultEnhancedExecutorConfig() EnhancedExecutorConfig {
	return EnhancedExecutorConfig{
		BaseConfig:              DefaultExecutorConfig(),
		IdempotencyConfig:       DefaultIdempotencyConfig(),
		RecoveryConfig:          DefaultRecoveryConfig(),
		EnableIdempotencyChecks: true,
		EnableRecovery:          true,
		CreateCheckpoints:       true,
		SkipIdempotentOps:       true,
	}
}

// Note: DefaultExecutorConfig is defined in executor.go

// Execute runs the entire plan with enhanced features
func (e *EnhancedExecutor) Execute(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	// Start cleanup worker for idempotency manager
	cleanupCtx, cleanupCancel := context.WithCancel(ctx)
	defer cleanupCancel()

	if e.config.IdempotencyConfig.EnableStateTracking {
		go e.idempotencyManager.StartCleanupWorker(cleanupCtx, 5*time.Minute)
	}

	// Enhance plan with idempotency checks
	enhancedPlan, err := e.enhancePlan(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("failed to enhance plan: %w", err)
	}

	// Execute the enhanced plan
	result, err := e.executeEnhancedPlan(ctx, enhancedPlan)
	if err != nil {
		// Attempt recovery if enabled
		if e.config.EnableRecovery {
			recoveryResult, recoveryErr := e.attemptRecovery(ctx, enhancedPlan, result, err)
			if recoveryErr == nil {
				return recoveryResult, nil
			}
		}
		return result, err
	}

	return result, nil
}

// ExecuteOperation runs a single operation with enhanced features
func (e *EnhancedExecutor) ExecuteOperation(ctx context.Context, operation *PlannedOperation) (*OperationResult, error) {
	return e.executeEnhancedOperation(ctx, operation)
}

// Cancel cancels any running operations
func (e *EnhancedExecutor) Cancel() error {
	return e.baseExecutor.Cancel()
}

// GetProgress returns current execution progress
func (e *EnhancedExecutor) GetProgress() *ExecutorProgress {
	return e.baseExecutor.GetProgress()
}

// enhancePlan enhances a plan with idempotency and recovery features
func (e *EnhancedExecutor) enhancePlan(ctx context.Context, plan *Plan) (*Plan, error) {
	enhancedPlan := *plan // Copy the plan
	enhancedOperations := make([]PlannedOperation, 0, len(plan.Operations))

	for _, operation := range plan.Operations {
		enhanced, skip, err := e.enhanceOperation(ctx, &operation)
		if err != nil {
			return nil, fmt.Errorf("failed to enhance operation %s: %w", operation.ID, err)
		}

		if skip && e.config.SkipIdempotentOps {
			// Mark as completed in the enhanced operation
			enhanced.Status = OperationStatusCompleted
		}

		enhancedOperations = append(enhancedOperations, *enhanced)
	}

	enhancedPlan.Operations = enhancedOperations
	return &enhancedPlan, nil
}

// enhanceOperation enhances a single operation with idempotency checks
func (e *EnhancedExecutor) enhanceOperation(ctx context.Context, operation *PlannedOperation) (*PlannedOperation, bool, error) {
	enhanced := *operation // Copy the operation

	// Check for duplicate operations
	if e.config.EnableIdempotencyChecks && e.config.IdempotencyConfig.EnableDeduplication {
		isDuplicate, duplicateState, err := e.idempotencyManager.IsDuplicateOperation(operation)
		if err != nil {
			return nil, false, fmt.Errorf("failed to check for duplicate operation: %w", err)
		}

		if isDuplicate && duplicateState.Status == OperationStatusCompleted {
			// Skip this operation as it's already completed
			return &enhanced, true, nil
		}
	}

	// Compute resource fingerprint
	var fingerprint string
	if e.config.EnableIdempotencyChecks && e.config.IdempotencyConfig.EnableFingerprinting {
		var err error
		fingerprint, err = e.idempotencyManager.ComputeResourceFingerprint(operation.Current, operation.ResourceType)
		if err != nil {
			return nil, false, fmt.Errorf("failed to compute resource fingerprint: %w", err)
		}

		// Check if operation is idempotent
		isIdempotent, err := e.idempotencyManager.IsOperationIdempotent(operation.ID, fingerprint)
		if err != nil {
			return nil, false, fmt.Errorf("failed to check operation idempotency: %w", err)
		}

		if isIdempotent {
			// Skip this operation as it's idempotent
			return &enhanced, true, nil
		}
	}

	// Create operation state
	if e.config.IdempotencyConfig.EnableStateTracking {
		state := e.idempotencyManager.CreateOperationState(operation, fingerprint)
		if state == nil {
			return nil, false, fmt.Errorf("failed to create operation state")
		}

		// Set plan ID if available
		state.PlanID = fmt.Sprintf("stage-%d", enhanced.Stage) // Use stage as a plan identifier for now
		err := e.idempotencyManager.UpdateOperationState(state)
		if err != nil {
			return nil, false, fmt.Errorf("failed to update operation state: %w", err)
		}
	}

	// Acquire resource ownership
	if e.config.IdempotencyConfig.EnableOwnershipTracking {
		_, err := e.idempotencyManager.AcquireResourceOwnership(
			operation.ResourceName,
			operation.ResourceType,
			fmt.Sprintf("plan-%d", enhanced.Stage), // Use stage as plan ID
			operation.ID,
		)
		if err != nil {
			return nil, false, fmt.Errorf("failed to acquire resource ownership: %w", err)
		}
	}

	return &enhanced, false, nil
}

// executeEnhancedPlan executes an enhanced plan
func (e *EnhancedExecutor) executeEnhancedPlan(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	// Set the current plan in the base executor so operations can access project ID
	if atlasExecutor, ok := e.baseExecutor.(*AtlasExecutor); ok {
		atlasExecutor.mu.Lock()
		atlasExecutor.currentPlan = plan
		atlasExecutor.mu.Unlock()
	}

	result := &ExecutionResult{
		PlanID:           plan.ID,
		Status:           PlanStatusExecuting,
		StartedAt:        time.Now(),
		OperationResults: make(map[string]*OperationResult),
		Summary:          ExecutionSummary{TotalOperations: len(plan.Operations)},
		Errors:           []ExecutionError{},
	}

	// Execute operations
	for _, operation := range plan.Operations {
		// Skip operations that are already completed
		if operation.Status == OperationStatusCompleted {
			result.Summary.CompletedOperations++
			continue
		}

		operationResult, err := e.executeEnhancedOperation(ctx, &operation)
		result.OperationResults[operation.ID] = operationResult

		if err != nil {
			result.Summary.FailedOperations++
			result.Errors = append(result.Errors, ExecutionError{
				OperationID: operation.ID,
				Message:     err.Error(),
				ErrorType:   "execution_error",
				Timestamp:   time.Now(),
				Recoverable: true,
			})

			// Attempt recovery for this operation
			if e.config.EnableRecovery {
				recoveryResult, recoveryErr := e.recoveryManager.RecoverFromFailure(ctx, &operation, err)
				if recoveryErr == nil && recoveryResult.Success {
					// Recovery successful, update the operation result
					operationResult.Status = OperationStatusCompleted
					operationResult.Error = ""
					result.Summary.FailedOperations--
					result.Summary.CompletedOperations++
				}
			}
		} else {
			result.Summary.CompletedOperations++
		}
	}

	// Finalize result
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	if result.Summary.FailedOperations == 0 {
		result.Status = PlanStatusCompleted
	} else {
		result.Status = PlanStatusFailed
	}

	return result, nil
}

// executeEnhancedOperation executes a single operation with enhanced features
func (e *EnhancedExecutor) executeEnhancedOperation(ctx context.Context, operation *PlannedOperation) (*OperationResult, error) {
	// Create checkpoint before execution
	if e.config.CreateCheckpoints {
		checkpointData := map[string]interface{}{
			"operation_type": operation.Type,
			"resource_type":  operation.ResourceType,
			"resource_name":  operation.ResourceName,
			"stage":          "pre-execution",
		}

		_, err := e.idempotencyManager.CreateCheckpoint(
			operation.ID,
			fmt.Sprintf("plan-%d", operation.Stage),
			"pre-execution",
			checkpointData,
			operation.Current,
		)
		if err != nil {
			// Log warning but don't fail the operation
			fmt.Printf("Warning: failed to create checkpoint: %v\n", err)
		}
	}

	// Update operation state to running
	if e.config.IdempotencyConfig.EnableStateTracking {
		if state, exists := e.idempotencyManager.GetOperationState(operation.ID); exists {
			now := time.Now()
			state.Status = OperationStatusRunning
			state.StartedAt = &now
			if err := e.idempotencyManager.UpdateOperationState(state); err != nil {
				// Non-fatal: keep executing but record the issue in result errors
				fmt.Printf("Warning: failed to update operation state: %v\n", err)
			}
		}
	}

	// Execute the operation using the base executor
	result, err := e.baseExecutor.ExecuteOperation(ctx, operation)

	// Update operation state based on result
	if e.config.IdempotencyConfig.EnableStateTracking {
		if state, exists := e.idempotencyManager.GetOperationState(operation.ID); exists {
			now := time.Now()
			state.CompletedAt = &now
			if err != nil {
				state.Status = OperationStatusFailed
				state.LastError = err.Error()
				state.ErrorHistory = append(state.ErrorHistory, OperationError{
					Timestamp:   now,
					Error:       err.Error(),
					ErrorType:   "execution_error",
					Recoverable: true,
					RetryCount:  result.RetryCount,
				})
			} else {
				state.Status = OperationStatusCompleted
			}
			if err := e.idempotencyManager.UpdateOperationState(state); err != nil {
				fmt.Printf("Warning: failed to update operation state: %v\n", err)
			}
		}
	}

	// Create checkpoint after execution
	if e.config.CreateCheckpoints {
		stage := "post-execution"
		if err != nil {
			stage = "post-execution-failed"
		}

		checkpointData := map[string]interface{}{
			"operation_type": operation.Type,
			"resource_type":  operation.ResourceType,
			"resource_name":  operation.ResourceName,
			"stage":          stage,
			"success":        err == nil,
		}

		if result != nil {
			checkpointData["resource_id"] = result.ResourceID
		}

		_, checkpointErr := e.idempotencyManager.CreateCheckpoint(
			operation.ID,
			fmt.Sprintf("plan-%d", operation.Stage),
			stage,
			checkpointData,
			nil, // Post-execution state could be fetched from Atlas if needed
		)
		if checkpointErr != nil {
			// Log warning but don't fail the operation
			fmt.Printf("Warning: failed to create post-execution checkpoint: %v\n", checkpointErr)
		}
	}

	// Release resource ownership on completion
	if e.config.IdempotencyConfig.EnableOwnershipTracking {
		releaseErr := e.idempotencyManager.ReleaseResourceOwnership(
			operation.ResourceName,
			operation.ResourceType,
			fmt.Sprintf("plan-%d", operation.Stage),
		)
		if releaseErr != nil {
			// Log warning but don't fail the operation
			fmt.Printf("Warning: failed to release resource ownership: %v\n", releaseErr)
		}
	}

	return result, err
}

// attemptRecovery attempts to recover from execution failures
func (e *EnhancedExecutor) attemptRecovery(ctx context.Context, plan *Plan, result *ExecutionResult, err error) (*ExecutionResult, error) {
	if !e.config.EnableRecovery {
		return result, err
	}

	// Identify failed operations
	failedOperations := []PlannedOperation{}
	for _, operation := range plan.Operations {
		if opResult, exists := result.OperationResults[operation.ID]; exists {
			if opResult.Status == OperationStatusFailed {
				failedOperations = append(failedOperations, operation)
			}
		}
	}

	if len(failedOperations) == 0 {
		return result, nil
	}

	// Attempt recovery for each failed operation
	recoveredCount := 0
	for _, operation := range failedOperations {
		opResult := result.OperationResults[operation.ID]
		opErr := fmt.Errorf("%s", opResult.Error)

		recoveryResult, recoveryErr := e.recoveryManager.RecoverFromFailure(ctx, &operation, opErr)
		if recoveryErr == nil && recoveryResult.Success {
			// Update the operation result to indicate successful recovery
			opResult.Status = OperationStatusCompleted
			opResult.Error = ""
			opResult.Metadata["recovered"] = true
			opResult.Metadata["recovery_strategy"] = recoveryResult.Strategy

			recoveredCount++
		}
	}

	// Update result summary
	result.Summary.FailedOperations -= recoveredCount
	result.Summary.CompletedOperations += recoveredCount

	if result.Summary.FailedOperations == 0 {
		result.Status = PlanStatusCompleted
		return result, nil
	}

	return result, fmt.Errorf("recovery partially successful: %d operations recovered, %d still failed", recoveredCount, result.Summary.FailedOperations)
}
