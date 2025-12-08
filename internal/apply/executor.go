package apply

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
	admin "go.mongodb.org/atlas-sdk/v20250312010/admin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Executor defines the interface for executing planned operations
type Executor interface {
	// Execute runs the entire plan and returns the execution result
	Execute(ctx context.Context, plan *Plan) (*ExecutionResult, error)

	// ExecuteOperation runs a single operation and returns the result
	ExecuteOperation(ctx context.Context, operation *PlannedOperation) (*OperationResult, error)

	// Cancel cancels any running operations
	Cancel() error

	// GetProgress returns current execution progress
	GetProgress() *ExecutorProgress
}

// AtlasExecutor implements the Executor interface for Atlas operations
type AtlasExecutor struct {
	// Atlas service clients
	clustersService      *atlas.ClustersService
	usersService         *atlas.DatabaseUsersService
	networkAccessService *atlas.NetworkAccessListsService
	projectsService      *atlas.ProjectsService
	searchService        *atlas.SearchService
	vpcEndpointsService  *atlas.VPCEndpointsService

	// Database service clients
	databaseService *database.Service

	// Execution state
	mu              sync.RWMutex
	currentPlan     *Plan
	progress        *ExecutorProgress
	cancelled       bool
	retryManager    *RetryManager
	progressTracker *ProgressTracker

	// Idempotency and recovery systems
	idempotencyManager *IdempotencyManager
	recoveryManager    *RecoveryManager

	// Configuration
	config ExecutorConfig
}

// ExecutorConfig contains configuration for the executor
type ExecutorConfig struct {
	// Parallel execution settings
	MaxConcurrentOperations int             `json:"maxConcurrentOperations"`
	ParallelSafeOperations  []OperationType `json:"parallelSafeOperations"`

	// Timing settings
	OperationTimeout       time.Duration `json:"operationTimeout"`
	ProgressUpdateInterval time.Duration `json:"progressUpdateInterval"`

	// Retry settings
	RetryConfig RetryConfig `json:"retryConfig"`

	// Logging and progress
	VerboseLogging bool `json:"verboseLogging"`
	QuietMode      bool `json:"quietMode"`

	// Safety/intent settings
	// When true, executor treats conflict errors on create as non-fatal for idempotent resources
	// and preserves existing resources instead of failing hard.
	PreserveExisting bool `json:"preserveExisting"`
}

// ExecutionResult contains the overall result of plan execution
type ExecutionResult struct {
	PlanID           string                      `json:"planId"`
	Status           PlanStatus                  `json:"status"`
	StartedAt        time.Time                   `json:"startedAt"`
	CompletedAt      time.Time                   `json:"completedAt"`
	Duration         time.Duration               `json:"duration"`
	OperationResults map[string]*OperationResult `json:"operationResults"`
	Summary          ExecutionSummary            `json:"summary"`
	Errors           []ExecutionError            `json:"errors,omitempty"`
}

// OperationResult contains the result of a single operation
type OperationResult struct {
	OperationID string                 `json:"operationId"`
	Status      OperationStatus        `json:"status"`
	StartedAt   time.Time              `json:"startedAt"`
	CompletedAt time.Time              `json:"completedAt"`
	Duration    time.Duration          `json:"duration"`
	RetryCount  int                    `json:"retryCount"`
	Error       string                 `json:"error,omitempty"`
	ResourceID  string                 `json:"resourceId,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionSummary provides high-level statistics about execution
type ExecutionSummary struct {
	TotalOperations     int `json:"totalOperations"`
	CompletedOperations int `json:"completedOperations"`
	FailedOperations    int `json:"failedOperations"`
	SkippedOperations   int `json:"skippedOperations"`
	RetriedOperations   int `json:"retriedOperations"`
}

// ExecutionError represents an error that occurred during execution
type ExecutionError struct {
	OperationID string    `json:"operationId"`
	Message     string    `json:"message"`
	ErrorType   string    `json:"errorType"`
	Timestamp   time.Time `json:"timestamp"`
	Recoverable bool      `json:"recoverable"`
}

// ExecutorProgress extends ExecutionProgress with additional executor-specific fields
type ExecutorProgress struct {
	ExecutionProgress
	Status            PlanStatus                 `json:"status"`
	StartedAt         time.Time                  `json:"startedAt"`
	StageProgress     float64                    `json:"stageProgress"`   // 0.0 to 1.0
	OverallProgress   float64                    `json:"overallProgress"` // 0.0 to 1.0
	OperationStatuses map[string]OperationStatus `json:"operationStatuses"`
}

// NewAtlasExecutor creates a new AtlasExecutor with the provided services
func NewAtlasExecutor(
	clustersService *atlas.ClustersService,
	usersService *atlas.DatabaseUsersService,
	networkAccessService *atlas.NetworkAccessListsService,
	projectsService *atlas.ProjectsService,
	searchService *atlas.SearchService,
	vpcEndpointsService *atlas.VPCEndpointsService,
	databaseService *database.Service,
	config ExecutorConfig,
) *AtlasExecutor {
	return &AtlasExecutor{
		clustersService:      clustersService,
		usersService:         usersService,
		networkAccessService: networkAccessService,
		projectsService:      projectsService,
		searchService:        searchService,
		vpcEndpointsService:  vpcEndpointsService,
		databaseService:      databaseService,
		config:               config,
		retryManager:         NewRetryManager(config.RetryConfig),
		progressTracker:      NewProgressTracker(config.ProgressUpdateInterval),
	}
}

// Execute implements the Executor interface
func (e *AtlasExecutor) Execute(ctx context.Context, plan *Plan) (*ExecutionResult, error) {
	e.mu.Lock()
	e.currentPlan = plan
	e.cancelled = false
	e.progress = &ExecutorProgress{
		ExecutionProgress: ExecutionProgress{
			PlanID:              plan.ID,
			CurrentStage:        0,
			TotalStages:         plan.GetMaxStage() + 1,
			CompletedOperations: 0,
			TotalOperations:     len(plan.Operations),
			EstimatedTimeLeft:   0,
			CurrentOperation:    "",
		},
		Status:            PlanStatusExecuting,
		StartedAt:         time.Now(),
		OperationStatuses: make(map[string]OperationStatus),
	}
	e.mu.Unlock()

	// Initialize operation statuses
	for _, op := range plan.Operations {
		e.progress.OperationStatuses[op.ID] = OperationStatusPending
	}

	result := &ExecutionResult{
		PlanID:           plan.ID,
		Status:           PlanStatusExecuting,
		StartedAt:        time.Now(),
		OperationResults: make(map[string]*OperationResult),
		Summary:          ExecutionSummary{TotalOperations: len(plan.Operations)},
	}

	// Start progress tracking
	e.progressTracker.Start(ctx, e.progress)
	defer e.progressTracker.Stop()

	// Execute operations stage by stage
	maxStage := plan.GetMaxStage()
	for stage := 0; stage <= maxStage; stage++ {
		select {
		case <-ctx.Done():
			return e.finalizeResult(result, PlanStatusCancelled, ctx.Err())
		default:
		}

		if e.isCancelled() {
			return e.finalizeResult(result, PlanStatusCancelled, fmt.Errorf("execution cancelled"))
		}

		e.updateProgress(stage, 0.0, "")

		// Get operations for this stage
		stageOperations := plan.GetOperationsInStage(stage)
		if len(stageOperations) == 0 {
			continue
		}

		// Execute stage operations (with parallelization where safe)
		if err := e.executeStage(ctx, stageOperations, result); err != nil {
			return e.finalizeResult(result, PlanStatusFailed, err)
		}
	}

	return e.finalizeResult(result, PlanStatusCompleted, nil)
}

// executeStage executes all operations in a stage, with parallelization where appropriate
func (e *AtlasExecutor) executeStage(ctx context.Context, operations []PlannedOperation, result *ExecutionResult) error {
	// Determine which operations can run in parallel
	parallelOps, sequentialOps := e.categorizeOperations(operations)

	// Execute parallel operations first
	if len(parallelOps) > 0 {
		if err := e.executeParallel(ctx, parallelOps, result); err != nil {
			return err
		}
	}

	// Execute sequential operations
	for _, op := range sequentialOps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if e.isCancelled() {
			return fmt.Errorf("execution cancelled")
		}

		opResult, err := e.ExecuteOperation(ctx, op)
		result.OperationResults[op.ID] = opResult
		e.updateSummary(&result.Summary, opResult)

		if err != nil && !e.shouldContinueOnError(op, err) {
			return fmt.Errorf("operation %s failed: %w", op.ID, err)
		}
	}

	return nil
}

// executeParallel executes operations in parallel with concurrency limits
func (e *AtlasExecutor) executeParallel(ctx context.Context, operations []*PlannedOperation, result *ExecutionResult) error {
	semaphore := make(chan struct{}, e.config.MaxConcurrentOperations)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstError error

	for _, op := range operations {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case semaphore <- struct{}{}:
		}

		if e.isCancelled() {
			<-semaphore
			return fmt.Errorf("execution cancelled")
		}

		wg.Add(1)
		go func(op *PlannedOperation) {
			defer wg.Done()
			defer func() { <-semaphore }()

			opResult, err := e.ExecuteOperation(ctx, op)

			mu.Lock()
			result.OperationResults[op.ID] = opResult
			e.updateSummary(&result.Summary, opResult)
			if err != nil && firstError == nil && !e.shouldContinueOnError(op, err) {
				firstError = fmt.Errorf("operation %s failed: %w", op.ID, err)
			}
			mu.Unlock()
		}(op)
	}

	wg.Wait()
	return firstError
}

// ExecuteOperation implements the Executor interface
func (e *AtlasExecutor) ExecuteOperation(ctx context.Context, operation *PlannedOperation) (*OperationResult, error) {
	result := &OperationResult{
		OperationID: operation.ID,
		Status:      OperationStatusRunning,
		StartedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Update progress
	e.updateOperationStatus(operation.ID, OperationStatusRunning)
	e.updateProgress(-1, -1, fmt.Sprintf("Executing: %s", operation.ResourceName))

	// Apply operation timeout
	opCtx := ctx
	if e.config.OperationTimeout > 0 {
		var cancel context.CancelFunc
		opCtx, cancel = context.WithTimeout(ctx, e.config.OperationTimeout)
		defer cancel()
	}

	// Execute with retry logic
	err := e.retryManager.ExecuteWithRetry(opCtx, operation, func() error {
		return e.executeOperationInternal(opCtx, operation, result)
	})

	// Finalize result
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.RetryCount = e.retryManager.GetRetryCount(operation.ID)

	if err != nil {
		result.Status = OperationStatusFailed
		result.Error = err.Error()
		e.updateOperationStatus(operation.ID, OperationStatusFailed)
	} else {
		result.Status = OperationStatusCompleted
		e.updateOperationStatus(operation.ID, OperationStatusCompleted)
	}

	return result, err
}

// executeOperationInternal performs the actual operation execution
func (e *AtlasExecutor) executeOperationInternal(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	switch operation.Type {
	case OperationCreate:
		return e.executeCreate(ctx, operation, result)
	case OperationUpdate:
		return e.executeUpdate(ctx, operation, result)
	case OperationDelete:
		return e.executeDelete(ctx, operation, result)
	case OperationNoChange:
		// No-op operations complete immediately
		return nil
	default:
		return fmt.Errorf("unsupported operation type: %s", operation.Type)
	}
}

// executeCreate handles create operations
func (e *AtlasExecutor) executeCreate(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	switch operation.ResourceType {
	case types.KindCluster:
		return e.createCluster(ctx, operation, result)
	case types.KindDatabaseUser:
		return e.createDatabaseUser(ctx, operation, result)
	case types.KindDatabaseRole:
		return e.createDatabaseRole(ctx, operation, result)
	case types.KindNetworkAccess:
		return e.createNetworkAccess(ctx, operation, result)
	case types.KindSearchIndex:
		return e.createSearchIndex(ctx, operation, result)
	case types.KindSearchMetrics:
		return e.executeSearchMetrics(ctx, operation, result)
	case types.KindSearchOptimization:
		return e.executeSearchOptimization(ctx, operation, result)
	case types.KindSearchQueryValidation:
		return e.executeSearchQueryValidation(ctx, operation, result)
	case types.KindVPCEndpoint:
		return e.createVPCEndpoint(ctx, operation, result)
	default:
		return fmt.Errorf("unsupported resource type for create: %s", operation.ResourceType)
	}
}

// executeUpdate handles update operations
func (e *AtlasExecutor) executeUpdate(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	switch operation.ResourceType {
	case types.KindProject:
		// Support updating project tags and name when provided
		if e.projectsService == nil {
			result.Metadata["operation"] = "updateProject"
			result.Metadata["resourceName"] = operation.ResourceName
			return fmt.Errorf("projects service not available")
		}
		// Desired/current are *types.ProjectManifest when diff engine includes project changes
		var desiredProject *types.ProjectManifest
		if pm, ok := operation.Desired.(*types.ProjectManifest); ok {
			desiredProject = pm
		}
		if desiredProject == nil {
			// Nothing to update if no desired manifest
			return nil
		}
		projectID := ""
		if e.currentPlan != nil {
			projectID = e.currentPlan.ProjectID
		}
		if projectID == "" {
			return fmt.Errorf("project ID not available for project update")
		}

		update := admin.GroupUpdate{}
		// Map tags if present
		if len(desiredProject.Spec.Tags) > 0 {
			tags := make([]admin.ResourceTag, 0, len(desiredProject.Spec.Tags))
			for k, v := range desiredProject.Spec.Tags {
				tags = append(tags, admin.ResourceTag{Key: k, Value: v})
			}
			update.Tags = &tags
		}
		// Optionally allow name update if changed
		if desiredProject.Spec.Name != "" {
			name := desiredProject.Spec.Name
			update.Name = &name
		}

		updated, err := e.projectsService.Update(ctx, projectID, update)
		if err != nil {
			result.Metadata["operation"] = "updateProject"
			result.Metadata["resourceName"] = operation.ResourceName
			result.Metadata["error"] = err.Error()
			return fmt.Errorf("failed to update project: %w", err)
		}
		result.Metadata["operation"] = "updateProject"
		result.Metadata["resourceName"] = operation.ResourceName
		if updated != nil && updated.Id != nil {
			result.Metadata["atlasResourceId"] = *updated.Id
		}
		return nil
	case types.KindCluster:
		return e.updateCluster(ctx, operation, result)
	case types.KindDatabaseUser:
		return e.updateDatabaseUser(ctx, operation, result)
	case types.KindDatabaseRole:
		return e.updateDatabaseRole(ctx, operation, result)
	case types.KindNetworkAccess:
		return e.updateNetworkAccess(ctx, operation, result)
	case types.KindVPCEndpoint:
		return e.updateVPCEndpoint(ctx, operation, result)
	default:
		return fmt.Errorf("unsupported resource type for update: %s", operation.ResourceType)
	}
}

// executeDelete handles delete operations
func (e *AtlasExecutor) executeDelete(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	switch operation.ResourceType {
	case types.KindCluster:
		return e.deleteCluster(ctx, operation, result)
	case types.KindDatabaseUser:
		return e.deleteDatabaseUser(ctx, operation, result)
	case types.KindDatabaseRole:
		return e.deleteDatabaseRole(ctx, operation, result)
	case types.KindNetworkAccess:
		return e.deleteNetworkAccess(ctx, operation, result)
	case types.KindVPCEndpoint:
		return e.deleteVPCEndpoint(ctx, operation, result)
	case types.KindProject:
		// Projects are typically not deleted directly through apply operations
		// Log this and treat as a no-op for now
		result.Metadata["operation"] = "skipProjectDelete"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["reason"] = "Project deletion skipped - projects are managed separately"
		return nil
	default:
		return fmt.Errorf("unsupported resource type for delete: %s", operation.ResourceType)
	}
}

// shouldIgnoreConflictError checks if we should ignore a conflict error based on preserve-existing behavior
func (e *AtlasExecutor) shouldIgnoreConflictError(err error) bool {
	if err == nil {
		return false
	}
	// Respect explicit intent
	if !e.config.PreserveExisting {
		return false
	}
	// Treat typed conflict errors from Atlas as preservable
	if atlasclient.IsConflict(err) {
		return true
	}
	// Fallback: tolerate common conflict substrings for older SDK variants
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "already exists") || strings.Contains(lower, "conflict")
}

// Resource-specific operation implementations
func (e *AtlasExecutor) createCluster(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.clustersService == nil {
		result.Metadata["operation"] = "createCluster"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("clusters service not available")
	}

	// Convert from apply types to Atlas SDK types
	var clusterConfig *types.ClusterConfig
	switch desired := operation.Desired.(type) {
	case *types.ClusterConfig:
		clusterConfig = desired
	case *types.ClusterManifest:
		// Convert ClusterManifest to ClusterConfig
		clusterConfig = &types.ClusterConfig{
			Metadata:       desired.Metadata,
			Provider:       desired.Spec.Provider,
			Region:         desired.Spec.Region,
			InstanceSize:   desired.Spec.InstanceSize,
			DiskSizeGB:     desired.Spec.DiskSizeGB,
			BackupEnabled:  desired.Spec.BackupEnabled,
			MongoDBVersion: desired.Spec.MongoDBVersion,
			ClusterType:    desired.Spec.ClusterType,
		}
	default:
		return fmt.Errorf("invalid resource type for cluster operation: expected ClusterConfig or ClusterManifest, got %T", operation.Desired)
	}

	// Build Atlas cluster object from config (includes replication specs and advanced fields)
	atlasCluster, err := buildAtlasClusterFromConfig(clusterConfig)
	if err != nil {
		return fmt.Errorf("failed to build cluster model: %w", err)
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for cluster creation")
	}

	// Create the cluster
	created, err := e.clustersService.Create(ctx, projectID, atlasCluster)
	if err != nil {
		result.Metadata["operation"] = "createCluster"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "createCluster"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["clusterName"] = created.GetName()
	result.Metadata["atlasResourceId"] = created.GetName()

	return nil
}

func (e *AtlasExecutor) updateCluster(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.clustersService == nil {
		result.Metadata["operation"] = "updateCluster"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("clusters service not available")
	}

	// Convert from apply types to Atlas SDK types
	var clusterConfig *types.ClusterConfig
	switch desired := operation.Desired.(type) {
	case *types.ClusterConfig:
		clusterConfig = desired
	case *types.ClusterManifest:
		// Convert ClusterManifest to ClusterConfig
		clusterConfig = &types.ClusterConfig{
			Metadata:       desired.Metadata,
			Provider:       desired.Spec.Provider,
			Region:         desired.Spec.Region,
			InstanceSize:   desired.Spec.InstanceSize,
			DiskSizeGB:     desired.Spec.DiskSizeGB,
			BackupEnabled:  desired.Spec.BackupEnabled,
			MongoDBVersion: desired.Spec.MongoDBVersion,
			ClusterType:    desired.Spec.ClusterType,
		}
	default:
		return fmt.Errorf("invalid resource type for cluster operation: expected ClusterConfig or ClusterManifest, got %T", operation.Desired)
	}

	// Build Atlas SDK cluster object with updatable fields
	atlasCluster, err := buildAtlasClusterFromConfig(clusterConfig)
	if err != nil {
		return fmt.Errorf("failed to build cluster model: %w", err)
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for cluster update")
	}

	// Update the cluster
	updated, err := e.clustersService.Update(ctx, projectID, clusterConfig.Metadata.Name, atlasCluster)
	if err != nil {
		result.Metadata["operation"] = "updateCluster"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "updateCluster"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["clusterName"] = updated.GetName()
	result.Metadata["atlasResourceId"] = updated.GetName()

	return nil
}

func (e *AtlasExecutor) deleteCluster(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.clustersService == nil {
		result.Metadata["operation"] = "deleteCluster"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("clusters service not available")
	}

	// Extract cluster name from operation
	clusterName := operation.ResourceName
	if operation.Current != nil {
		if clusterConfig, ok := operation.Current.(*types.ClusterConfig); ok {
			clusterName = clusterConfig.Metadata.Name
		}
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for cluster deletion")
	}

	// Delete the cluster
	err := e.clustersService.Delete(ctx, projectID, clusterName)
	if err != nil {
		// Check if cluster doesn't exist (already deleted) - treat as success
		if atlasclient.IsNotFound(err) {
			result.Metadata["operation"] = "deleteCluster"
			result.Metadata["resourceName"] = operation.ResourceName
			result.Metadata["clusterName"] = clusterName
			result.Metadata["atlasResourceId"] = clusterName
			result.Metadata["note"] = "cluster was already deleted"
			return nil
		}

		result.Metadata["operation"] = "deleteCluster"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "deleteCluster"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["clusterName"] = clusterName
	result.Metadata["atlasResourceId"] = clusterName

	return nil
}

func (e *AtlasExecutor) createDatabaseUser(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.usersService == nil {
		result.Metadata["operation"] = "createDatabaseUser"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("database user service not available")
	}

	// Convert from apply types to Atlas SDK types
	var userSpec types.DatabaseUserConfig
	switch desired := operation.Desired.(type) {
	case *types.DatabaseUserManifest:
		// Convert DatabaseUserSpec to DatabaseUserConfig
		userSpec = types.DatabaseUserConfig{
			Metadata:     desired.Metadata,
			Username:     desired.Spec.Username,
			Password:     desired.Spec.Password,
			Roles:        desired.Spec.Roles,
			AuthDatabase: desired.Spec.AuthDatabase,
			Scopes:       desired.Spec.Scopes,
		}
	case *types.DatabaseUserConfig:
		userSpec = *desired
	default:
		return fmt.Errorf("invalid resource type for database user operation: expected DatabaseUserManifest or DatabaseUserConfig, got %T", operation.Desired)
	}

	// Convert to Atlas model
	atlasUser, err := convertDatabaseUserConfigToAtlas(userSpec)
	if err != nil {
		return err
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for database user creation")
	}

	// Create the database user
	created, err := e.usersService.Create(ctx, projectID, atlasUser)
	if err != nil {
		// Check if this is a conflict and if we should preserve existing resources
		if e.shouldIgnoreConflictError(err) {
			// Treat this as success - the user already exists and we're preserving it
			result.Metadata["operation"] = "createDatabaseUser"
			result.Metadata["resourceName"] = operation.ResourceName
			result.Metadata["username"] = userSpec.Username
			result.Metadata["databaseName"] = "admin"
			result.Metadata["atlasResourceId"] = userSpec.Username
			result.Metadata["note"] = "User already exists, preserved with --preserve-existing"
			return nil
		}

		result.Metadata["operation"] = "createDatabaseUser"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		return fmt.Errorf("failed to create database user: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "createDatabaseUser"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["username"] = created.GetUsername()
	result.Metadata["databaseName"] = created.GetDatabaseName()
	result.Metadata["atlasResourceId"] = created.GetUsername() // Use username as resource ID

	return nil
}

func (e *AtlasExecutor) updateDatabaseUser(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.usersService == nil {
		result.Metadata["operation"] = "updateDatabaseUser"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("database user service not available")
	}

	// Convert from apply types to Atlas SDK types
	var userSpec types.DatabaseUserConfig
	switch desired := operation.Desired.(type) {
	case *types.DatabaseUserManifest:
		// Convert DatabaseUserSpec to DatabaseUserConfig
		userSpec = types.DatabaseUserConfig{
			Metadata:     desired.Metadata,
			Username:     desired.Spec.Username,
			Password:     desired.Spec.Password,
			Roles:        desired.Spec.Roles,
			AuthDatabase: desired.Spec.AuthDatabase,
			Scopes:       desired.Spec.Scopes,
		}
	case *types.DatabaseUserConfig:
		userSpec = *desired
	default:
		return fmt.Errorf("invalid resource type for database user operation: expected DatabaseUserManifest or DatabaseUserConfig, got %T", operation.Desired)
	}

	// Convert to Atlas model (update semantics: password optional)
	atlasUser, err := convertDatabaseUserConfigToAtlas(userSpec)
	if err != nil {
		return err
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for database user update")
	}

	// Set auth database default if not specified
	authDatabase := userSpec.AuthDatabase
	if authDatabase == "" {
		authDatabase = "admin"
	}

	// Update the database user
	updated, err := e.usersService.Update(ctx, projectID, authDatabase, userSpec.Username, atlasUser)
	if err != nil {
		result.Metadata["operation"] = "updateDatabaseUser"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		return fmt.Errorf("failed to update database user: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "updateDatabaseUser"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["username"] = updated.GetUsername()
	result.Metadata["databaseName"] = updated.GetDatabaseName()
	result.Metadata["atlasResourceId"] = updated.GetUsername()

	return nil
}

func (e *AtlasExecutor) deleteDatabaseUser(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.usersService == nil {
		result.Metadata["operation"] = "deleteDatabaseUser"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("database user service not available")
	}

	// Extract user info from operation
	username := operation.ResourceName
	authDatabase := "admin" // default

	if operation.Current != nil {
		if userManifest, ok := operation.Current.(*types.DatabaseUserManifest); ok {
			username = userManifest.Spec.Username
			if userManifest.Spec.AuthDatabase != "" {
				authDatabase = userManifest.Spec.AuthDatabase
			}
		}
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for database user deletion")
	}

	// Delete the database user
	err := e.usersService.Delete(ctx, projectID, authDatabase, username)
	if err != nil {
		// Check if user doesn't exist (already deleted) - treat as success
		if atlasclient.IsNotFound(err) {
			result.Metadata["operation"] = "deleteDatabaseUser"
			result.Metadata["resourceName"] = operation.ResourceName
			result.Metadata["username"] = username
			result.Metadata["databaseName"] = authDatabase
			result.Metadata["atlasResourceId"] = username
			result.Metadata["note"] = "database user was already deleted"
			result.Status = OperationStatusCompleted
			return nil
		}

		result.Metadata["operation"] = "deleteDatabaseUser"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		result.Status = OperationStatusFailed
		return fmt.Errorf("failed to delete database user: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "deleteDatabaseUser"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["username"] = username
	result.Metadata["databaseName"] = authDatabase
	result.Metadata["atlasResourceId"] = username
	result.Status = OperationStatusCompleted

	return nil
}

func (e *AtlasExecutor) createNetworkAccess(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.networkAccessService == nil {
		result.Metadata["operation"] = "createNetworkAccess"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("network access service not available")
	}

	// Convert from apply types to Atlas SDK types
	networkManifest, ok := operation.Desired.(*types.NetworkAccessManifest)
	if !ok {
		return fmt.Errorf("invalid resource type for network access operation")
	}

	// Create Atlas SDK network access entry
	entry, err := convertNetworkAccessManifestToEntry(networkManifest)
	if err != nil {
		return err
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for network access creation")
	}

	// Create the network access entry
	created, err := e.networkAccessService.Create(ctx, projectID, []admin.NetworkPermissionEntry{entry})
	if err != nil {
		result.Metadata["operation"] = "createNetworkAccess"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		return fmt.Errorf("failed to create network access entry: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "createNetworkAccess"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["atlasResourceId"] = networkManifest.Metadata.Name

	// Add entry-specific metadata
	if created != nil && created.Results != nil && len(*created.Results) > 0 {
		firstEntry := (*created.Results)[0]
		if firstEntry.IpAddress != nil {
			result.Metadata["ipAddress"] = *firstEntry.IpAddress
		}
		if firstEntry.CidrBlock != nil {
			result.Metadata["cidrBlock"] = *firstEntry.CidrBlock
		}
		if firstEntry.AwsSecurityGroup != nil {
			result.Metadata["awsSecurityGroup"] = *firstEntry.AwsSecurityGroup
		}
	}

	return nil
}

func (e *AtlasExecutor) updateNetworkAccess(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	// Network access entries cannot be updated in Atlas - they must be deleted and recreated
	// This is a limitation of the Atlas API
	result.Metadata["operation"] = "updateNetworkAccess"
	result.Metadata["resourceName"] = operation.ResourceName
	return fmt.Errorf("network access entries cannot be updated - delete and recreate instead")
}

func (e *AtlasExecutor) deleteNetworkAccess(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.networkAccessService == nil {
		result.Metadata["operation"] = "deleteNetworkAccess"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("network access service not available")
	}

	// Extract entry info from operation
	ipAddress := ""

	if operation.Current != nil {
		if networkManifest, ok := operation.Current.(*types.NetworkAccessManifest); ok {
			if networkManifest.Spec.IPAddress != "" {
				ipAddress = networkManifest.Spec.IPAddress
			} else if networkManifest.Spec.CIDR != "" {
				ipAddress = networkManifest.Spec.CIDR
			} else if networkManifest.Spec.AWSSecurityGroup != "" {
				ipAddress = networkManifest.Spec.AWSSecurityGroup
			}
		}
	}

	if ipAddress == "" {
		ipAddress = operation.ResourceName
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for network access deletion")
	}

	// Delete the network access entry
	err := e.networkAccessService.Delete(ctx, projectID, ipAddress)
	if err != nil {
		// Check if network access entry doesn't exist (already deleted) - treat as success
		if atlasclient.IsNotFound(err) {
			result.Metadata["operation"] = "deleteNetworkAccess"
			result.Metadata["resourceName"] = operation.ResourceName
			result.Metadata["ipAddress"] = ipAddress
			result.Metadata["atlasResourceId"] = ipAddress
			result.Metadata["note"] = "network access entry was already deleted"
			result.Status = OperationStatusCompleted
			return nil
		}

		result.Metadata["operation"] = "deleteNetworkAccess"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		result.Status = OperationStatusFailed
		return fmt.Errorf("failed to delete network access entry: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "deleteNetworkAccess"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["ipAddress"] = ipAddress
	result.Metadata["atlasResourceId"] = ipAddress
	result.Status = OperationStatusCompleted

	return nil
}

// createSearchIndex creates a new search index
func (e *AtlasExecutor) createSearchIndex(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.searchService == nil {
		result.Metadata["operation"] = "createSearchIndex"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("search service not available")
	}

	// Convert from apply types to search index request
	searchManifest, ok := operation.Desired.(*types.SearchIndexManifest)
	if !ok {
		return fmt.Errorf("invalid resource type for search index operation: expected SearchIndexManifest, got %T", operation.Desired)
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}
	if projectID == "" {
		return fmt.Errorf("project ID not available for search index creation")
	}

	// Create the search index request (SDK now uses ClusterSearchIndex)
	indexRequest := admin.NewClusterSearchIndex(
		searchManifest.Spec.CollectionName,
		searchManifest.Spec.DatabaseName,
		searchManifest.Spec.IndexName,
	)

	// Set index type if specified
	if searchManifest.Spec.IndexType != "" {
		indexRequest.SetType(searchManifest.Spec.IndexType)
	}

	// Convert and set definition
	// Note: Definition structure may differ between old SearchIndexCreateRequest and new ClusterSearchIndex
	// For now, we'll skip the definition conversion as it requires more complex field mapping
	// TODO: Implement full definition conversion when needed
	if searchManifest.Spec.Definition != nil {
		// Print warning about definition conversion (executor has no logger)
		fmt.Fprintf(os.Stderr, "Warning: Search index definition conversion not fully implemented in new SDK version (index: %s, cluster: %s)\n",
			searchManifest.Spec.IndexName, searchManifest.Spec.ClusterName)
	}

	// Create the search index
	created, err := e.searchService.CreateSearchIndex(ctx, projectID, searchManifest.Spec.ClusterName, *indexRequest)
	if err != nil {
		result.Metadata["operation"] = "createSearchIndex"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		return fmt.Errorf("failed to create search index: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "createSearchIndex"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["indexName"] = searchManifest.Spec.IndexName
	result.Metadata["clusterName"] = searchManifest.Spec.ClusterName
	result.Metadata["databaseName"] = searchManifest.Spec.DatabaseName
	result.Metadata["collectionName"] = searchManifest.Spec.CollectionName
	if created.GetIndexID() != "" {
		result.Metadata["atlasResourceId"] = created.GetIndexID()
	}
	result.Status = OperationStatusCompleted

	return nil
}

// Cancel implements the Executor interface
func (e *AtlasExecutor) Cancel() error {
	e.mu.Lock()
	e.cancelled = true
	e.mu.Unlock()
	return nil
}

// Helper functions for cluster configuration

// getClusterType ensures we have a valid cluster type, defaulting if needed
func getClusterType(clusterType string) string {
	if clusterType == "" {
		return "REPLICASET"
	}
	return clusterType
}

// getMongoDBVersion ensures we have a valid MongoDB version, defaulting if needed
func getMongoDBVersion(version string) string {
	if version == "" {
		return "7.0"
	}
	return version
}

// buildReplicationSpecsFromConfig builds replication specs from cluster config
func buildReplicationSpecsFromConfig(config *types.ClusterConfig) ([]admin.ReplicationSpec20240805, error) {
	// Default to single replica set if no replication specs defined
	if len(config.ReplicationSpecs) == 0 {
		// Build default replication spec for single region
		regionConfig, err := buildDefaultRegionConfig(config)
		if err != nil {
			return nil, err
		}

		replicationSpec := admin.ReplicationSpec20240805{
			RegionConfigs: &[]admin.CloudRegionConfig20240805{regionConfig},
		}

		return []admin.ReplicationSpec20240805{replicationSpec}, nil
	}

	// Convert existing replication specs
	var atlasSpecs []admin.ReplicationSpec20240805
	for _, spec := range config.ReplicationSpecs {
		var regionConfigs []admin.CloudRegionConfig20240805
		for _, regionConfig := range spec.RegionConfigs {
			atlasRegionConfig := convertRegionConfig(regionConfig, config)
			regionConfigs = append(regionConfigs, atlasRegionConfig)
		}

		atlasSpec := admin.ReplicationSpec20240805{
			RegionConfigs: &regionConfigs,
		}

		atlasSpecs = append(atlasSpecs, atlasSpec)
	}

	return atlasSpecs, nil
}

// buildDefaultRegionConfig creates a default region config from basic cluster settings
func buildDefaultRegionConfig(config *types.ClusterConfig) (admin.CloudRegionConfig20240805, error) {
	// Default values
	nodeCount := 3
	priority := 7
	diskIOPS := 3000
	ebsVolumeType := "STANDARD"

	// Build hardware specs
	electableSpecs := admin.HardwareSpec20240805{
		InstanceSize:  &config.InstanceSize,
		NodeCount:     &nodeCount,
		DiskIOPS:      &diskIOPS,
		EbsVolumeType: &ebsVolumeType,
	}

	// Set disk size if specified
	if config.DiskSizeGB != nil && *config.DiskSizeGB > 0 {
		electableSpecs.DiskSizeGB = config.DiskSizeGB
	}

	regionConfig := admin.CloudRegionConfig20240805{
		ProviderName:   &config.Provider,
		RegionName:     &config.Region,
		Priority:       &priority,
		ElectableSpecs: &electableSpecs,
	}

	return regionConfig, nil
}

// convertRegionConfig converts internal region config to Atlas SDK format
func convertRegionConfig(regionConfig types.RegionConfig, clusterConfig *types.ClusterConfig) admin.CloudRegionConfig20240805 {
	// Use cluster-level defaults for missing region-specific values
	provider := regionConfig.ProviderName
	if provider == "" {
		provider = clusterConfig.Provider
	}

	instanceSize := clusterConfig.InstanceSize
	diskSizeGB := clusterConfig.DiskSizeGB

	// Default values
	diskIOPS := 3000
	ebsVolumeType := "STANDARD"
	nodeCount := 3
	if regionConfig.ElectableNodes != nil {
		nodeCount = *regionConfig.ElectableNodes
	}

	// Build hardware specs
	electableSpecs := admin.HardwareSpec20240805{
		InstanceSize:  &instanceSize,
		NodeCount:     &nodeCount,
		DiskIOPS:      &diskIOPS,
		EbsVolumeType: &ebsVolumeType,
	}

	if diskSizeGB != nil {
		electableSpecs.DiskSizeGB = diskSizeGB
	}

	atlasRegionConfig := admin.CloudRegionConfig20240805{
		ProviderName:   &provider,
		RegionName:     &regionConfig.RegionName,
		ElectableSpecs: &electableSpecs,
	}

	if regionConfig.Priority != nil {
		atlasRegionConfig.Priority = regionConfig.Priority
	}

	// Map cluster-level autoscaling settings to region autoScaling if provided
	if clusterConfig.AutoScaling != nil {
		advanced := admin.AdvancedAutoScalingSettings{}
		// Disk GB autoscaling
		if clusterConfig.AutoScaling.DiskGB != nil && clusterConfig.AutoScaling.DiskGB.Enabled != nil {
			advanced.DiskGB = &admin.DiskGBAutoScaling{Enabled: clusterConfig.AutoScaling.DiskGB.Enabled}
		}
		// Compute autoscaling
		if clusterConfig.AutoScaling.Compute != nil {
			compute := admin.AdvancedComputeAutoScaling{}
			if clusterConfig.AutoScaling.Compute.Enabled != nil {
				compute.Enabled = clusterConfig.AutoScaling.Compute.Enabled
			}
			if clusterConfig.AutoScaling.Compute.ScaleDownEnabled != nil {
				compute.ScaleDownEnabled = clusterConfig.AutoScaling.Compute.ScaleDownEnabled
			}
			advanced.Compute = &compute
		}
		// Only set when any autoscaling setting is present
		if advanced.DiskGB != nil || advanced.Compute != nil {
			atlasRegionConfig.AutoScaling = &advanced
		}
	}

	return atlasRegionConfig
}

// GetProgress implements the Executor interface
func (e *AtlasExecutor) GetProgress() *ExecutorProgress {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.progress == nil {
		return nil
	}

	// Return a copy to avoid concurrent access issues
	progress := *e.progress
	return &progress
}

// Helper methods

func (e *AtlasExecutor) isCancelled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cancelled
}

func (e *AtlasExecutor) updateProgress(stage int, stageProgress float64, currentOperation string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.progress == nil {
		return
	}

	if stage >= 0 {
		e.progress.CurrentStage = stage
	}
	if stageProgress >= 0 {
		e.progress.StageProgress = stageProgress
	}
	if currentOperation != "" {
		e.progress.CurrentOperation = currentOperation
	}

	// Calculate overall progress
	if e.progress.TotalStages > 0 {
		baseProgress := float64(e.progress.CurrentStage) / float64(e.progress.TotalStages)
		stageContribution := e.progress.StageProgress / float64(e.progress.TotalStages)
		e.progress.OverallProgress = baseProgress + stageContribution
	}

	// Update completion count
	completed := 0
	for _, status := range e.progress.OperationStatuses {
		if status == OperationStatusCompleted {
			completed++
		}
	}
	e.progress.CompletedOperations = completed
}

func (e *AtlasExecutor) updateOperationStatus(operationID string, status OperationStatus) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.progress != nil {
		e.progress.OperationStatuses[operationID] = status
	}
}

func (e *AtlasExecutor) categorizeOperations(operations []PlannedOperation) ([]*PlannedOperation, []*PlannedOperation) {
	var parallel []*PlannedOperation
	var sequential []*PlannedOperation

	for i := range operations {
		op := &operations[i]
		if e.isParallelSafe(op) {
			parallel = append(parallel, op)
		} else {
			sequential = append(sequential, op)
		}
	}

	return parallel, sequential
}

func (e *AtlasExecutor) isParallelSafe(operation *PlannedOperation) bool {
	for _, safeType := range e.config.ParallelSafeOperations {
		if operation.Type == safeType {
			return true
		}
	}
	return false
}

func (e *AtlasExecutor) shouldContinueOnError(operation *PlannedOperation, err error) bool {
	// Continue on non-critical errors unless they are authentication or configuration errors.
	if err == nil {
		return true
	}

	// Treat unauthorized/authentication errors as fatal.
	if atlasclient.IsUnauthorized(err) {
		return false
	}

	// Conflicts during Create may be ignored when preserve-existing is on.
	if operation != nil && operation.Type == OperationCreate && e.shouldIgnoreConflictError(err) {
		return true
	}

	// Transient errors are retried at lower layers; treat as non-fatal to allow other ops to proceed.
	if atlasclient.IsTransient(err) {
		return true
	}

	// Service wiring errors (e.g., "clusters service not available") are fatal to preserve existing test expectations
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "service not available") {
		return false
	}

	// Validation/configuration style errors should stop the execution for critical-impact operations.
	if operation != nil && operation.Impact != nil && operation.Impact.RiskLevel == RiskLevelCritical {
		return false
	}

	// Default: continue to process other operations.
	return true
}

func (e *AtlasExecutor) updateSummary(summary *ExecutionSummary, result *OperationResult) {
	switch result.Status {
	case OperationStatusCompleted:
		summary.CompletedOperations++
	case OperationStatusFailed:
		summary.FailedOperations++
	case OperationStatusSkipped:
		summary.SkippedOperations++
	}

	if result.RetryCount > 0 {
		summary.RetriedOperations++
	}
}

func (e *AtlasExecutor) finalizeResult(result *ExecutionResult, status PlanStatus, err error) (*ExecutionResult, error) {
	result.Status = status
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	if err != nil {
		result.Errors = append(result.Errors, ExecutionError{
			Message:     err.Error(),
			ErrorType:   "execution_error",
			Timestamp:   time.Now(),
			Recoverable: false,
		})
	}

	// Update plan status
	if e.currentPlan != nil {
		e.currentPlan.Status = status
		e.currentPlan.CompletedAt = &result.CompletedAt
		if err != nil {
			e.currentPlan.LastError = err.Error()
		}
	}

	return result, err
}

// DefaultExecutorConfig returns a default configuration for the executor
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		MaxConcurrentOperations: 5,
		ParallelSafeOperations: []OperationType{
			OperationCreate, // Safe for independent resources
			OperationUpdate, // Safe for independent resources
		},
		OperationTimeout:       30 * time.Minute,
		ProgressUpdateInterval: 1 * time.Second,
		RetryConfig:            DefaultRetryConfig(),
		VerboseLogging:         false,
		QuietMode:              false,
	}
}

// buildAtlasClusterFromConfig builds an Atlas SDK cluster object from our internal cluster configuration
func buildAtlasClusterFromConfig(config *types.ClusterConfig) (*admin.ClusterDescription20240805, error) {
	if config == nil {
		return nil, fmt.Errorf("cluster config is nil")
	}

	atlasCluster := &admin.ClusterDescription20240805{
		Name:                &config.Metadata.Name,
		ClusterType:         admin.PtrString(getClusterType(config.ClusterType)),
		MongoDBMajorVersion: admin.PtrString(getMongoDBVersion(config.MongoDBVersion)),
		BackupEnabled:       config.BackupEnabled,
	}

	// Replication specs
	replicationSpecs, err := buildReplicationSpecsFromConfig(config)
	if err != nil {
		return nil, err
	}
	atlasCluster.ReplicationSpecs = &replicationSpecs

	// Tags
	if len(config.Tags) > 0 {
		tags := make([]admin.ResourceTag, 0, len(config.Tags))
		for k, v := range config.Tags {
			tags = append(tags, admin.ResourceTag{Key: k, Value: v})
		}
		atlasCluster.Tags = &tags
	}

	// Encryption at rest provider flag on cluster (project-level KMS config handled by EncryptionService)
	if config.Encryption != nil && config.Encryption.EncryptionAtRestProvider != "" && strings.ToUpper(config.Encryption.EncryptionAtRestProvider) != "NONE" {
		provider := config.Encryption.EncryptionAtRestProvider
		atlasCluster.EncryptionAtRestProvider = &provider
	}

	// BI Connector
	if config.BiConnector != nil {
		bi := admin.BiConnector{}
		if config.BiConnector.Enabled != nil {
			bi.Enabled = config.BiConnector.Enabled
		}
		if config.BiConnector.ReadPreference != "" {
			rp := config.BiConnector.ReadPreference
			bi.ReadPreference = &rp
		}
		atlasCluster.BiConnector = &bi
	}

	return atlasCluster, nil
}

// convertDatabaseUserConfigToAtlas converts a DatabaseUserConfig into Atlas model including roles and scopes.
func convertDatabaseUserConfigToAtlas(userSpec types.DatabaseUserConfig) (*admin.CloudDatabaseUser, error) {
	if userSpec.Username == "" {
		return nil, fmt.Errorf("database user username is required")
	}

	authDB := userSpec.AuthDatabase
	if authDB == "" {
		authDB = "admin"
	}

	atlasUser := &admin.CloudDatabaseUser{
		Username:     userSpec.Username,
		DatabaseName: authDB,
	}

	if userSpec.Password != "" {
		atlasUser.Password = admin.PtrString(userSpec.Password)
	}

	if len(userSpec.Roles) == 0 {
		return nil, fmt.Errorf("database user %s must have at least one role defined", userSpec.Username)
	}
	roles := make([]admin.DatabaseUserRole, len(userSpec.Roles))
	for i, role := range userSpec.Roles {
		if role.RoleName == "" || role.DatabaseName == "" {
			return nil, fmt.Errorf("invalid role at index %d: roleName and databaseName are required", i)
		}
		r := admin.DatabaseUserRole{RoleName: role.RoleName, DatabaseName: role.DatabaseName}
		if role.CollectionName != "" {
			r.CollectionName = &role.CollectionName
		}
		roles[i] = r
	}
	atlasUser.Roles = &roles

	if len(userSpec.Scopes) > 0 {
		scopes := make([]admin.UserScope, len(userSpec.Scopes))
		for i, s := range userSpec.Scopes {
			scopes[i] = admin.UserScope{Name: s.Name, Type: s.Type}
		}
		atlasUser.Scopes = &scopes
	}

	return atlasUser, nil
}

func (e *AtlasExecutor) createDatabaseRole(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	result.Metadata["operation"] = "createDatabaseRole"
	result.Metadata["resourceName"] = operation.ResourceName

	// Convert from apply types to our internal types
	var roleSpec types.DatabaseRoleSpec
	switch desired := operation.Desired.(type) {
	case *types.DatabaseRoleManifest:
		roleSpec = desired.Spec
	case *types.CustomDatabaseRoleConfig:
		// Convert CustomDatabaseRoleConfig to DatabaseRoleSpec
		roleSpec = types.DatabaseRoleSpec{
			RoleName:       desired.RoleName,
			DatabaseName:   desired.DatabaseName,
			Privileges:     desired.Privileges,
			InheritedRoles: desired.InheritedRoles,
		}
	default:
		return fmt.Errorf("invalid resource type for database role operation: expected DatabaseRoleManifest or CustomDatabaseRoleConfig, got %T", operation.Desired)
	}

	result.Metadata["roleName"] = roleSpec.RoleName
	result.Metadata["databaseName"] = roleSpec.DatabaseName

	connString, err := e.resolveRoleConnectionString(operation)
	if err != nil {
		return err
	}

	clientOpts := options.Client().ApplyURI(connString)
	mClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB for role creation: %w", err)
	}
	defer func() { _ = mClient.Disconnect(ctx) }()

	db := mClient.Database(roleSpec.DatabaseName)

	// Build privileges array from spec
	var privileges []bson.M
	for _, p := range roleSpec.Privileges {
		res := bson.M{"db": p.Resource.Database}
		if strings.TrimSpace(p.Resource.Collection) != "" {
			res["collection"] = p.Resource.Collection
		}
		privileges = append(privileges, bson.M{
			"resource": res,
			"actions":  p.Actions,
		})
	}

	// Build inherited roles
	var inherited []bson.M
	for _, r := range roleSpec.InheritedRoles {
		inherited = append(inherited, bson.M{"role": r.RoleName, "db": r.DatabaseName})
	}

	cmd := bson.D{{Key: "createRole", Value: roleSpec.RoleName}, {Key: "privileges", Value: privileges}, {Key: "roles", Value: inherited}}
	if err := db.RunCommand(ctx, cmd).Err(); err != nil {
		return fmt.Errorf("failed to create database role '%s' in '%s': %w", roleSpec.RoleName, roleSpec.DatabaseName, err)
	}

	result.Metadata["atlasResourceId"] = roleSpec.RoleName
	return nil
}

func (e *AtlasExecutor) updateDatabaseRole(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	result.Metadata["operation"] = "updateDatabaseRole"
	result.Metadata["resourceName"] = operation.ResourceName

	// Convert from apply types to our internal types
	var roleSpec types.DatabaseRoleSpec
	switch desired := operation.Desired.(type) {
	case *types.DatabaseRoleManifest:
		roleSpec = desired.Spec
	case *types.CustomDatabaseRoleConfig:
		// Convert CustomDatabaseRoleConfig to DatabaseRoleSpec
		roleSpec = types.DatabaseRoleSpec{
			RoleName:       desired.RoleName,
			DatabaseName:   desired.DatabaseName,
			Privileges:     desired.Privileges,
			InheritedRoles: desired.InheritedRoles,
		}
	default:
		return fmt.Errorf("invalid resource type for database role operation: expected DatabaseRoleManifest or CustomDatabaseRoleConfig, got %T", operation.Desired)
	}

	result.Metadata["roleName"] = roleSpec.RoleName
	result.Metadata["databaseName"] = roleSpec.DatabaseName

	connString, err := e.resolveRoleConnectionString(operation)
	if err != nil {
		return err
	}

	clientOpts := options.Client().ApplyURI(connString)
	mClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB for role update: %w", err)
	}
	defer func() { _ = mClient.Disconnect(ctx) }()

	db := mClient.Database(roleSpec.DatabaseName)

	// Build privileges and inherited roles documents
	var privileges []bson.M
	for _, p := range roleSpec.Privileges {
		res := bson.M{"db": p.Resource.Database}
		if strings.TrimSpace(p.Resource.Collection) != "" {
			res["collection"] = p.Resource.Collection
		}
		privileges = append(privileges, bson.M{"	resource": res, "actions": p.Actions})
	}
	var inherited []bson.M
	for _, r := range roleSpec.InheritedRoles {
		inherited = append(inherited, bson.M{"role": r.RoleName, "db": r.DatabaseName})
	}

	cmd := bson.D{{Key: "updateRole", Value: roleSpec.RoleName}, {Key: "privileges", Value: privileges}, {Key: "roles", Value: inherited}}
	if err := db.RunCommand(ctx, cmd).Err(); err != nil {
		return fmt.Errorf("failed to update database role '%s' in '%s': %w", roleSpec.RoleName, roleSpec.DatabaseName, err)
	}

	result.Metadata["atlasResourceId"] = roleSpec.RoleName
	return nil
}

func (e *AtlasExecutor) deleteDatabaseRole(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	result.Metadata["operation"] = "deleteDatabaseRole"
	result.Metadata["resourceName"] = operation.ResourceName

	// For delete operations, we might have the resource name from the operation
	// Extract role name and database name from the operation metadata
	roleName := operation.ResourceName
	databaseName := ""

	// Try to extract more information from the desired state if available
	switch desired := operation.Desired.(type) {
	case *types.DatabaseRoleManifest:
		roleName = desired.Spec.RoleName
		databaseName = desired.Spec.DatabaseName
	case *types.CustomDatabaseRoleConfig:
		roleName = desired.RoleName
		databaseName = desired.DatabaseName
	}

	result.Metadata["roleName"] = roleName
	if databaseName != "" {
		result.Metadata["databaseName"] = databaseName
	}

	connString, err := e.resolveRoleConnectionString(operation)
	if err != nil {
		return err
	}

	if databaseName == "" {
		return fmt.Errorf("database name is required for role deletion")
	}

	clientOpts := options.Client().ApplyURI(connString)
	mClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB for role deletion: %w", err)
	}
	defer func() { _ = mClient.Disconnect(ctx) }()

	db := mClient.Database(databaseName)
	cmd := bson.D{{Key: "dropRole", Value: roleName}}
	if err := db.RunCommand(ctx, cmd).Err(); err != nil {
		return fmt.Errorf("failed to delete database role '%s' in '%s': %w", roleName, databaseName, err)
	}

	result.Metadata["atlasResourceId"] = roleName
	return nil
}

// resolveRoleConnectionString resolves a MongoDB connection string for role operations.
// Precedence:
// 1) Resource metadata annotation "matlas.mongodb.com/connection-string"
// 2) Environment var MATLAS_ROLE_CONN_STRING
// If neither is present, returns an error.
func (e *AtlasExecutor) resolveRoleConnectionString(operation *PlannedOperation) (string, error) {
	// Try to read from manifest metadata annotations
	var annotations map[string]string
	switch desired := operation.Desired.(type) {
	case *types.DatabaseRoleManifest:
		annotations = desired.Metadata.Annotations
	case *types.CustomDatabaseRoleConfig:
		if desired.Metadata.Annotations != nil {
			annotations = desired.Metadata.Annotations
		}
	}

	if annotations != nil {
		if cs, ok := annotations["matlas.mongodb.com/connection-string"]; ok && strings.TrimSpace(cs) != "" {
			return cs, nil
		}
	}

	if cs := os.Getenv("MATLAS_ROLE_CONN_STRING"); strings.TrimSpace(cs) != "" {
		return cs, nil
	}

	return "", fmt.Errorf("connection string not provided for DatabaseRole operation. Set metadata.annotations['matlas.mongodb.com/connection-string'] or MATLAS_ROLE_CONN_STRING env var")
}

// convertNetworkAccessManifestToEntry converts a NetworkAccessManifest to Atlas NetworkPermissionEntry.
func convertNetworkAccessManifestToEntry(manifest *types.NetworkAccessManifest) (admin.NetworkPermissionEntry, error) {
	if manifest == nil {
		return admin.NetworkPermissionEntry{}, fmt.Errorf("network access manifest is nil")
	}

	entry := admin.NetworkPermissionEntry{}
	if manifest.Spec.IPAddress != "" {
		entry.IpAddress = &manifest.Spec.IPAddress
	} else if manifest.Spec.CIDR != "" {
		entry.CidrBlock = &manifest.Spec.CIDR
	} else if manifest.Spec.AWSSecurityGroup != "" {
		entry.AwsSecurityGroup = &manifest.Spec.AWSSecurityGroup
	} else {
		return admin.NetworkPermissionEntry{}, fmt.Errorf("network access entry must specify IP address, CIDR block, or AWS security group")
	}

	if manifest.Spec.Comment != "" {
		entry.Comment = &manifest.Spec.Comment
	}
	if manifest.Spec.DeleteAfterDate != "" {
		// Parse RFC3339 date
		if t, err := time.Parse(time.RFC3339, manifest.Spec.DeleteAfterDate); err == nil {
			entry.DeleteAfterDate = &t
		} else {
			return admin.NetworkPermissionEntry{}, fmt.Errorf("invalid deleteAfterDate format (must be RFC3339): %w", err)
		}
	}
	return entry, nil
}

// convertSearchDefinitionToSDK converts a raw search definition to Atlas SDK format
func convertSearchDefinitionToSDK(rawDefinition map[string]interface{}, indexType string) (*admin.BaseSearchIndexCreateRequestDefinition, error) {
	definition := admin.NewBaseSearchIndexCreateRequestDefinitionWithDefaults()

	// Convert mappings if present (for text search)
	if mappingsRaw, ok := rawDefinition["mappings"]; ok {
		if mappingsMap, ok := mappingsRaw.(map[string]interface{}); ok {
			mappings := admin.SearchMappings{}

			// Handle dynamic mapping
			if dynamic, ok := mappingsMap["dynamic"]; ok {
				if dynamicBool, ok := dynamic.(bool); ok {
					mappings.SetDynamic(dynamicBool)
				}
			}

			// Handle fields mapping
			if fields, ok := mappingsMap["fields"]; ok {
				if fieldsMap, ok := fields.(map[string]interface{}); ok {
					mappings.SetFields(fieldsMap)
				}
			}

			definition.SetMappings(mappings)
		}
	}

	// Convert fields if present (for vector search)
	if fieldsRaw, ok := rawDefinition["fields"]; ok {
		if fieldsSlice, ok := fieldsRaw.([]interface{}); ok {
			definition.SetFields(fieldsSlice)
		}
	}

	// Only set analyzer and searchAnalyzer for non-vector search indexes
	// Vector search doesn't support these attributes
	if indexType != "vectorSearch" {
		// Convert analyzer if present
		if analyzer, ok := rawDefinition["analyzer"]; ok {
			if analyzerStr, ok := analyzer.(string); ok {
				definition.SetAnalyzer(analyzerStr)
			}
		}

		// Convert searchAnalyzer if present
		if searchAnalyzer, ok := rawDefinition["searchAnalyzer"]; ok {
			if searchAnalyzerStr, ok := searchAnalyzer.(string); ok {
				definition.SetSearchAnalyzer(searchAnalyzerStr)
			}
		}
	}

	// Remove default analyzer attributes for vector search, which does not support analyzer fields
	if indexType == "vectorSearch" {
		definition.Analyzer = nil
		definition.SearchAnalyzer = nil
	}

	return definition, nil
}

// enhanceDefinitionWithAdvancedFeatures adds advanced search features to the search index definition
func enhanceDefinitionWithAdvancedFeatures(definition *admin.BaseSearchIndexCreateRequestDefinition, spec *types.SearchIndexSpec) error {
	// For now, we'll skip custom analyzers enhancement to prevent the reference error
	// The issue is that Atlas API expects analyzers to be at the definition root level
	// but the current SDK structure doesn't provide a clean way to set them
	// Custom analyzers should be defined as built-in analyzers for now
	if len(spec.Analyzers) > 0 {
		// Log that custom analyzers are being skipped for now
		// This prevents the "non-existent analyzers" error
		// TODO: Implement proper analyzer support when Atlas SDK provides the right structure
	}

	// Convert facets, autocomplete, highlighting, and fuzzy search into field mappings
	if definition.Mappings != nil {
		if fields, ok := definition.Mappings.GetFieldsOk(); ok && fields != nil {
			fieldsMap := *fields
			if fieldsMap == nil {
				fieldsMap = make(map[string]interface{})
			}

			// Add facet configurations
			for _, facet := range spec.Facets {
				fieldConfig, exists := fieldsMap[facet.Field]
				if !exists {
					fieldConfig = make(map[string]interface{})
				}
				fieldMap, ok := fieldConfig.(map[string]interface{})
				if !ok {
					fieldMap = make(map[string]interface{})
				}

				facetConfig := map[string]interface{}{
					"type": facet.Type,
				}
				if facet.NumBuckets != nil {
					facetConfig["numBuckets"] = *facet.NumBuckets
				}
				if facet.Boundaries != nil {
					facetConfig["boundaries"] = facet.Boundaries
				}
				if facet.Default != nil {
					facetConfig["default"] = *facet.Default
				}

				fieldMap["facet"] = facetConfig
				fieldsMap[facet.Field] = fieldMap
			}

			// Add autocomplete configurations
			for _, autocomplete := range spec.Autocomplete {
				fieldConfig, exists := fieldsMap[autocomplete.Field]
				if !exists {
					fieldConfig = make(map[string]interface{})
				}
				fieldMap, ok := fieldConfig.(map[string]interface{})
				if !ok {
					fieldMap = make(map[string]interface{})
				}

				autocompleteConfig := map[string]interface{}{}
				if autocomplete.MaxEdits > 0 {
					autocompleteConfig["maxEdits"] = autocomplete.MaxEdits
				}
				if autocomplete.PrefixLength > 0 {
					autocompleteConfig["prefixLength"] = autocomplete.PrefixLength
				}
				if autocomplete.FuzzyMaxEdits > 0 {
					autocompleteConfig["fuzzyMaxEdits"] = autocomplete.FuzzyMaxEdits
				}

				fieldMap["autocomplete"] = autocompleteConfig
				fieldsMap[autocomplete.Field] = fieldMap
			}

			// Add highlighting configurations
			for _, highlighting := range spec.Highlighting {
				fieldConfig, exists := fieldsMap[highlighting.Field]
				if !exists {
					fieldConfig = make(map[string]interface{})
				}
				fieldMap, ok := fieldConfig.(map[string]interface{})
				if !ok {
					fieldMap = make(map[string]interface{})
				}

				highlightConfig := map[string]interface{}{}
				if highlighting.MaxCharsToExamine > 0 {
					highlightConfig["maxCharsToExamine"] = highlighting.MaxCharsToExamine
				}
				if highlighting.MaxNumPassages > 0 {
					highlightConfig["maxNumPassages"] = highlighting.MaxNumPassages
				}

				fieldMap["highlight"] = highlightConfig
				fieldsMap[highlighting.Field] = fieldMap
			}

			// Add fuzzy search configurations
			for _, fuzzy := range spec.FuzzySearch {
				fieldConfig, exists := fieldsMap[fuzzy.Field]
				if !exists {
					fieldConfig = make(map[string]interface{})
				}
				fieldMap, ok := fieldConfig.(map[string]interface{})
				if !ok {
					fieldMap = make(map[string]interface{})
				}

				fuzzyConfig := map[string]interface{}{}
				if fuzzy.MaxEdits > 0 {
					fuzzyConfig["maxEdits"] = fuzzy.MaxEdits
				}
				if fuzzy.PrefixLength > 0 {
					fuzzyConfig["prefixLength"] = fuzzy.PrefixLength
				}
				if fuzzy.MaxExpansions > 0 {
					fuzzyConfig["maxExpansions"] = fuzzy.MaxExpansions
				}

				fieldMap["fuzzy"] = fuzzyConfig
				fieldsMap[fuzzy.Field] = fieldMap
			}

			definition.Mappings.SetFields(fieldsMap)
		}
	}

	// Convert synonyms (add to root level of definition)
	if len(spec.Synonyms) > 0 {
		synonyms := make([]map[string]interface{}, len(spec.Synonyms))
		for i, synonym := range spec.Synonyms {
			synonymMap := map[string]interface{}{
				"name":  synonym.Name,
				"input": synonym.Input,
			}
			if synonym.Output != "" {
				synonymMap["output"] = synonym.Output
			}
			synonymMap["explicit"] = synonym.Explicit
			synonyms[i] = synonymMap
		}

		// Synonyms are typically added at the root level of the definition
		// Note: This may need adjustment based on actual Atlas Search API requirements
	}

	return nil
}

// VPC Endpoint operation implementations

func (e *AtlasExecutor) createVPCEndpoint(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.vpcEndpointsService == nil {
		result.Metadata["operation"] = "createVPCEndpoint"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("VPC endpoints service not available")
	}

	// Convert from apply types to VPCEndpointManifest
	var vpcEndpoint *types.VPCEndpointManifest
	switch desired := operation.Desired.(type) {
	case *types.VPCEndpointManifest:
		vpcEndpoint = desired
	default:
		return fmt.Errorf("invalid resource type for VPC endpoint operation: expected VPCEndpointManifest, got %T", operation.Desired)
	}

	// Get project ID from operation context or VPC endpoint spec
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}
	if projectID == "" {
		projectID = vpcEndpoint.Spec.ProjectName
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for VPC endpoint creation")
	}

	// Build Atlas VPC endpoint service request
	serviceRequest := admin.CloudProviderEndpointServiceRequest{
		ProviderName: vpcEndpoint.Spec.CloudProvider,
		Region:       vpcEndpoint.Spec.Region,
	}

	// Create the VPC endpoint service
	created, err := e.vpcEndpointsService.CreatePrivateEndpointService(ctx, projectID, vpcEndpoint.Spec.CloudProvider, serviceRequest)
	if err != nil {
		// Check if we should ignore conflict errors
		if e.shouldIgnoreConflictError(err) {
			result.Metadata["operation"] = "createVPCEndpoint"
			result.Metadata["resourceName"] = operation.ResourceName
			result.Metadata["skipped"] = "true"
			result.Metadata["reason"] = "VPC endpoint already exists (preserve-existing enabled)"
			return nil
		}

		result.Metadata["operation"] = "createVPCEndpoint"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		return fmt.Errorf("failed to create VPC endpoint: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "createVPCEndpoint"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["endpointServiceName"] = created.GetEndpointServiceName()
	result.Metadata["atlasResourceId"] = created.GetId()
	result.Metadata["cloudProvider"] = created.GetCloudProvider()
	result.Metadata["region"] = created.GetRegionName()

	return nil
}

func (e *AtlasExecutor) updateVPCEndpoint(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.vpcEndpointsService == nil {
		result.Metadata["operation"] = "updateVPCEndpoint"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("VPC endpoints service not available")
	}

	// Convert from apply types to VPCEndpointManifest
	var vpcEndpoint *types.VPCEndpointManifest
	switch desired := operation.Desired.(type) {
	case *types.VPCEndpointManifest:
		vpcEndpoint = desired
	default:
		return fmt.Errorf("invalid resource type for VPC endpoint operation: expected VPCEndpointManifest, got %T", operation.Desired)
	}

	// Get project ID from operation context or VPC endpoint spec
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}
	if projectID == "" {
		projectID = vpcEndpoint.Spec.ProjectName
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for VPC endpoint update")
	}

	// For VPC endpoints, most properties are immutable after creation
	// This is essentially a no-op but we log that update was requested
	result.Metadata["operation"] = "updateVPCEndpoint"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["endpointId"] = vpcEndpoint.Spec.EndpointID
	result.Metadata["reason"] = "VPC endpoint properties are immutable after creation"

	return nil
}

func (e *AtlasExecutor) deleteVPCEndpoint(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.vpcEndpointsService == nil {
		result.Metadata["operation"] = "deleteVPCEndpoint"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("VPC endpoints service not available")
	}

	// Convert from apply types to VPCEndpointManifest
	var vpcEndpoint *types.VPCEndpointManifest
	switch current := operation.Current.(type) {
	case *types.VPCEndpointManifest:
		vpcEndpoint = current
	default:
		return fmt.Errorf("invalid resource type for VPC endpoint operation: expected VPCEndpointManifest, got %T", operation.Current)
	}

	// Get project ID from operation context or VPC endpoint spec
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}
	if projectID == "" {
		projectID = vpcEndpoint.Spec.ProjectName
	}

	if projectID == "" {
		return fmt.Errorf("project ID not available for VPC endpoint deletion")
	}

	// Delete the VPC endpoint service
	if err := e.vpcEndpointsService.DeletePrivateEndpointService(ctx, projectID, vpcEndpoint.Spec.CloudProvider, vpcEndpoint.Spec.EndpointID); err != nil {
		result.Metadata["operation"] = "deleteVPCEndpoint"
		result.Metadata["resourceName"] = operation.ResourceName
		result.Metadata["error"] = err.Error()
		return fmt.Errorf("failed to delete VPC endpoint: %w", err)
	}

	// Record success metadata
	result.Metadata["operation"] = "deleteVPCEndpoint"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["endpointId"] = vpcEndpoint.Spec.EndpointID
	result.Metadata["cloudProvider"] = vpcEndpoint.Spec.CloudProvider

	return nil
}

// executeSearchMetrics retrieves search metrics
func (e *AtlasExecutor) executeSearchMetrics(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.searchService == nil {
		result.Metadata["operation"] = "executeSearchMetrics"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("search service not available")
	}

	// Convert from apply types to search metrics request
	metricsManifest, ok := operation.Desired.(*types.SearchMetricsManifest)
	if !ok {
		return fmt.Errorf("invalid resource type for search metrics operation: expected SearchMetricsManifest, got %T", operation.Desired)
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}
	if projectID == "" {
		projectID = metricsManifest.Spec.ProjectName
	}
	if projectID == "" {
		return fmt.Errorf("project ID not available for search metrics operation")
	}

	// Create advanced search service
	advancedService := atlas.NewAdvancedSearchService(e.searchService)

	// Get metrics
	timeRange := metricsManifest.Spec.TimeRange
	if timeRange == "" {
		timeRange = "24h"
	}

	metrics, err := advancedService.GetSearchMetrics(ctx, projectID, metricsManifest.Spec.ClusterName, metricsManifest.Spec.IndexName, timeRange)
	if err != nil {
		result.Metadata["operation"] = "executeSearchMetrics"
		result.Metadata["resourceName"] = operation.ResourceName
		return err
	}

	// Record success metadata
	result.Metadata["operation"] = "executeSearchMetrics"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["clusterName"] = metricsManifest.Spec.ClusterName
	result.Metadata["timeRange"] = timeRange
	if metricsManifest.Spec.IndexName != nil {
		result.Metadata["indexName"] = *metricsManifest.Spec.IndexName
	}
	result.Metadata["metrics"] = metrics

	return nil
}

// executeSearchOptimization analyzes search indexes for optimization
func (e *AtlasExecutor) executeSearchOptimization(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.searchService == nil {
		result.Metadata["operation"] = "executeSearchOptimization"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("search service not available")
	}

	// Convert from apply types to search optimization request
	optimizationManifest, ok := operation.Desired.(*types.SearchOptimizationManifest)
	if !ok {
		return fmt.Errorf("invalid resource type for search optimization operation: expected SearchOptimizationManifest, got %T", operation.Desired)
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}
	if projectID == "" {
		projectID = optimizationManifest.Spec.ProjectName
	}
	if projectID == "" {
		return fmt.Errorf("project ID not available for search optimization operation")
	}

	// Create advanced search service
	advancedService := atlas.NewAdvancedSearchService(e.searchService)

	var analysisResults map[string]interface{}

	if optimizationManifest.Spec.IndexName != nil {
		// Analyze specific index
		analysis, err := advancedService.AnalyzeSearchIndex(ctx, projectID, optimizationManifest.Spec.ClusterName, *optimizationManifest.Spec.IndexName)
		if err != nil {
			result.Metadata["operation"] = "executeSearchOptimization"
			result.Metadata["resourceName"] = operation.ResourceName
			return err
		}
		analysisResults = analysis
	} else {
		// Analyze all indexes
		indexes, err := e.searchService.ListSearchIndexes(ctx, projectID, optimizationManifest.Spec.ClusterName, nil, nil)
		if err != nil {
			result.Metadata["operation"] = "executeSearchOptimization"
			result.Metadata["resourceName"] = operation.ResourceName
			return err
		}

		analyses := make(map[string]interface{})
		for _, index := range indexes {
			indexAnalysis, err := advancedService.AnalyzeSearchIndex(ctx, projectID, optimizationManifest.Spec.ClusterName, index.GetName())
			if err != nil {
				// Continue with other indexes if one fails
				continue
			}
			analyses[index.GetName()] = indexAnalysis
		}

		analysisResults = map[string]interface{}{
			"cluster":    optimizationManifest.Spec.ClusterName,
			"indexes":    analyses,
			"analyzed":   len(analyses),
			"analyzeAll": optimizationManifest.Spec.AnalyzeAll,
			"categories": optimizationManifest.Spec.Categories,
		}
	}

	// Record success metadata
	result.Metadata["operation"] = "executeSearchOptimization"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["clusterName"] = optimizationManifest.Spec.ClusterName
	result.Metadata["analyzeAll"] = optimizationManifest.Spec.AnalyzeAll
	if optimizationManifest.Spec.IndexName != nil {
		result.Metadata["indexName"] = *optimizationManifest.Spec.IndexName
	}
	result.Metadata["analysisResults"] = analysisResults

	return nil
}

// executeSearchQueryValidation validates a search query
func (e *AtlasExecutor) executeSearchQueryValidation(ctx context.Context, operation *PlannedOperation, result *OperationResult) error {
	if e.searchService == nil {
		result.Metadata["operation"] = "executeSearchQueryValidation"
		result.Metadata["resourceName"] = operation.ResourceName
		return fmt.Errorf("search service not available")
	}

	// Convert from apply types to search query validation request
	validationManifest, ok := operation.Desired.(*types.SearchQueryValidationManifest)
	if !ok {
		return fmt.Errorf("invalid resource type for search query validation operation: expected SearchQueryValidationManifest, got %T", operation.Desired)
	}

	// Get project ID from operation context
	projectID := ""
	if e.currentPlan != nil {
		projectID = e.currentPlan.ProjectID
	}
	if projectID == "" {
		projectID = validationManifest.Spec.ProjectName
	}
	if projectID == "" {
		return fmt.Errorf("project ID not available for search query validation operation")
	}

	// Create advanced search service
	advancedService := atlas.NewAdvancedSearchService(e.searchService)

	// Validate the query
	validationResult, err := advancedService.ValidateSearchQuery(ctx, projectID, validationManifest.Spec.ClusterName, validationManifest.Spec.IndexName, validationManifest.Spec.Query)
	if err != nil {
		result.Metadata["operation"] = "executeSearchQueryValidation"
		result.Metadata["resourceName"] = operation.ResourceName
		return err
	}

	// Add additional analysis in test mode
	if validationManifest.Spec.TestMode {
		indexAnalysis, err := advancedService.AnalyzeSearchIndex(ctx, projectID, validationManifest.Spec.ClusterName, validationManifest.Spec.IndexName)
		if err == nil {
			validationResult["indexAnalysis"] = indexAnalysis
			validationResult["testMode"] = true
		}
	}

	// Add validation configuration
	validationResult["validationTypes"] = validationManifest.Spec.Validate

	// Record success metadata
	result.Metadata["operation"] = "executeSearchQueryValidation"
	result.Metadata["resourceName"] = operation.ResourceName
	result.Metadata["clusterName"] = validationManifest.Spec.ClusterName
	result.Metadata["indexName"] = validationManifest.Spec.IndexName
	result.Metadata["testMode"] = validationManifest.Spec.TestMode
	result.Metadata["validationResult"] = validationResult

	return nil
}
