package apply

import (
	"context"
	"fmt"
	"time"

	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
)

// RecoveryManager handles failure recovery, rollback, and cleanup operations
type RecoveryManager struct {
	// Service clients for recovery operations
	clustersService      *atlas.ClustersService
	usersService         *atlas.DatabaseUsersService
	networkAccessService *atlas.NetworkAccessListsService
	projectsService      *atlas.ProjectsService
	databaseService      *database.Service

	// Dependencies
	idempotencyManager *IdempotencyManager

	// Configuration
	config RecoveryConfig
}

// RecoveryConfig contains configuration for recovery operations
type RecoveryConfig struct {
	// Recovery strategy settings
	DefaultStrategy     RecoveryStrategy                   `json:"defaultStrategy"`
	OperationStrategies map[OperationType]RecoveryStrategy `json:"operationStrategies"`

	// Rollback settings
	EnableRollback      bool          `json:"enableRollback"`
	RollbackTimeout     time.Duration `json:"rollbackTimeout"`
	MaxRollbackAttempts int           `json:"maxRollbackAttempts"`

	// Cleanup settings
	EnableCleanup         bool          `json:"enableCleanup"`
	CleanupTimeout        time.Duration `json:"cleanupTimeout"`
	RetainFailedResources bool          `json:"retainFailedResources"`

	// Analysis settings
	EnableFailureAnalysis  bool `json:"enableFailureAnalysis"`
	GenerateRecoveryReport bool `json:"generateRecoveryReport"`

	// Interactive settings
	InteractiveMode   bool `json:"interactiveMode"`
	PromptForStrategy bool `json:"promptForStrategy"`
}

// RecoveryStrategy represents different recovery approaches
type RecoveryStrategy string

const (
	RecoveryStrategyRetry    RecoveryStrategy = "retry"    // Retry the failed operation
	RecoveryStrategySkip     RecoveryStrategy = "skip"     // Skip the failed operation and continue
	RecoveryStrategyRollback RecoveryStrategy = "rollback" // Roll back the failed operation
	RecoveryStrategyAbort    RecoveryStrategy = "abort"    // Abort the entire execution
	RecoveryStrategyManual   RecoveryStrategy = "manual"   // Request manual intervention
	RecoveryStrategyCleanup  RecoveryStrategy = "cleanup"  // Clean up partial resources and continue
)

// RecoveryResult represents the result of a recovery operation
type RecoveryResult struct {
	OperationID string           `json:"operationId"`
	Strategy    RecoveryStrategy `json:"strategy"`
	Success     bool             `json:"success"`

	// Actions taken
	RollbackPerformed  bool     `json:"rollbackPerformed"`
	CleanupPerformed   bool     `json:"cleanupPerformed"`
	ResourcesRecovered []string `json:"resourcesRecovered"`
	ResourcesCleaned   []string `json:"resourcesCleaned"`

	// Timing
	StartedAt   time.Time     `json:"startedAt"`
	CompletedAt time.Time     `json:"completedAt"`
	Duration    time.Duration `json:"duration"`

	// Results
	Error       string   `json:"error,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	NextActions []string `json:"nextActions,omitempty"`

	// Analysis
	FailureAnalysis *FailureAnalysis `json:"failureAnalysis,omitempty"`
}

// FailureAnalysis provides detailed analysis of operation failures
type FailureAnalysis struct {
	OperationID  string      `json:"operationId"`
	FailureType  FailureType `json:"failureType"`
	RootCause    string      `json:"rootCause"`
	Contributing []string    `json:"contributingFactors"`

	// Recovery recommendations
	Recommendations []RecoveryRecommendation `json:"recommendations"`
	PreventionSteps []string                 `json:"preventionSteps"`

	// Resource impact
	AffectedResources []AffectedResource `json:"affectedResources"`

	// Metadata
	AnalyzedAt time.Time `json:"analyzedAt"`
	Confidence float64   `json:"confidence"` // 0.0 to 1.0
}

// FailureType categorizes different types of failures
type FailureType string

const (
	FailureTypeNetwork        FailureType = "network"        // Network connectivity issues
	FailureTypeAuthentication FailureType = "authentication" // Auth/permission issues
	FailureTypeQuota          FailureType = "quota"          // Resource quota/limit issues
	FailureTypeConflict       FailureType = "conflict"       // Resource conflict/concurrency
	FailureTypeValidation     FailureType = "validation"     // Input validation issues
	FailureTypeDependency     FailureType = "dependency"     // Dependency/prerequisite issues
	FailureTypeResource       FailureType = "resource"       // Resource state issues
	FailureTypeTimeout        FailureType = "timeout"        // Operation timeout
	FailureTypeInternal       FailureType = "internal"       // Internal/unknown errors
)

// RecoveryRecommendation provides actionable recovery suggestions
type RecoveryRecommendation struct {
	Strategy    RecoveryStrategy `json:"strategy"`
	Description string           `json:"description"`
	Confidence  float64          `json:"confidence"` // 0.0 to 1.0
	Risk        RiskLevel        `json:"risk"`
	Steps       []string         `json:"steps"`
	Automated   bool             `json:"automated"`
}

// AffectedResource describes a resource impacted by a failure
type AffectedResource struct {
	ResourceID   string             `json:"resourceId"`
	ResourceKind types.ResourceKind `json:"resourceKind"`
	State        ResourceState      `json:"state"`
	Impact       ResourceImpact     `json:"impact"`
	Recovery     ResourceRecovery   `json:"recovery"`
}

// ResourceState describes the current state of a resource
type ResourceState string

const (
	ResourceStateUnknown      ResourceState = "unknown"
	ResourceStatePartial      ResourceState = "partial"      // Partially created/updated
	ResourceStateInconsistent ResourceState = "inconsistent" // In inconsistent state
	ResourceStateOrphaned     ResourceState = "orphaned"     // Created but not properly linked
	ResourceStateCorrupted    ResourceState = "corrupted"    // Data corruption detected
	ResourceStateHealthy      ResourceState = "healthy"      // No issues detected
)

// ResourceImpact describes how a resource was impacted
type ResourceImpact string

const (
	ResourceImpactNone      ResourceImpact = "none"      // No impact
	ResourceImpactPartial   ResourceImpact = "partial"   // Partially affected
	ResourceImpactComplete  ResourceImpact = "complete"  // Completely affected
	ResourceImpactCorrupted ResourceImpact = "corrupted" // Data corrupted
	ResourceImpactOrphaned  ResourceImpact = "orphaned"  // Left in orphaned state
)

// ResourceRecovery describes recovery options for a resource
type ResourceRecovery struct {
	Possible  bool      `json:"possible"`
	Strategy  string    `json:"strategy"`
	Steps     []string  `json:"steps"`
	Risk      RiskLevel `json:"risk"`
	Automated bool      `json:"automated"`
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(
	clustersService *atlas.ClustersService,
	usersService *atlas.DatabaseUsersService,
	networkAccessService *atlas.NetworkAccessListsService,
	projectsService *atlas.ProjectsService,
	databaseService *database.Service,
	idempotencyManager *IdempotencyManager,
	config RecoveryConfig,
) *RecoveryManager {
	return &RecoveryManager{
		clustersService:      clustersService,
		usersService:         usersService,
		networkAccessService: networkAccessService,
		projectsService:      projectsService,
		databaseService:      databaseService,
		idempotencyManager:   idempotencyManager,
		config:               config,
	}
}

// DefaultRecoveryConfig returns a default recovery configuration
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		DefaultStrategy: RecoveryStrategyRetry,
		OperationStrategies: map[OperationType]RecoveryStrategy{
			OperationCreate: RecoveryStrategyRollback,
			OperationUpdate: RecoveryStrategyRetry,
			OperationDelete: RecoveryStrategySkip,
		},
		EnableRollback:         true,
		RollbackTimeout:        10 * time.Minute,
		MaxRollbackAttempts:    3,
		EnableCleanup:          true,
		CleanupTimeout:         5 * time.Minute,
		RetainFailedResources:  false,
		EnableFailureAnalysis:  true,
		GenerateRecoveryReport: true,
		InteractiveMode:        false,
		PromptForStrategy:      false,
	}
}

// RecoverFromFailure attempts to recover from a failed operation
func (rm *RecoveryManager) RecoverFromFailure(ctx context.Context, operation *PlannedOperation, err error) (*RecoveryResult, error) {
	result := &RecoveryResult{
		OperationID: operation.ID,
		StartedAt:   time.Now(),
	}

	// Analyze the failure first
	if rm.config.EnableFailureAnalysis {
		analysis := rm.analyzeFailure(operation, err)
		result.FailureAnalysis = analysis

		// Use analysis to determine best recovery strategy
		if len(analysis.Recommendations) > 0 {
			result.Strategy = analysis.Recommendations[0].Strategy
		}
	}

	// Determine recovery strategy if not set by analysis
	if result.Strategy == "" {
		result.Strategy = rm.determineRecoveryStrategy(operation, err)
	}

	// Execute recovery strategy
	switch result.Strategy {
	case RecoveryStrategyRetry:
		err = rm.handleRetryRecovery(ctx, operation, result)
	case RecoveryStrategyRollback:
		err = rm.handleRollbackRecovery(ctx, operation, result)
	case RecoveryStrategyCleanup:
		err = rm.handleCleanupRecovery(ctx, operation, result)
	case RecoveryStrategySkip:
		err = rm.handleSkipRecovery(ctx, operation, result)
	case RecoveryStrategyAbort:
		err = rm.handleAbortRecovery(ctx, operation, result)
	case RecoveryStrategyManual:
		err = rm.handleManualRecovery(ctx, operation, result)
	default:
		err = fmt.Errorf("unsupported recovery strategy: %s", result.Strategy)
	}

	// Finalize result
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Success = err == nil

	if err != nil {
		result.Error = err.Error()
	}

	return result, err
}

// analyzeFailure performs detailed analysis of an operation failure
func (rm *RecoveryManager) analyzeFailure(operation *PlannedOperation, err error) *FailureAnalysis {
	analysis := &FailureAnalysis{
		OperationID:       operation.ID,
		AnalyzedAt:        time.Now(),
		AffectedResources: []AffectedResource{},
		Recommendations:   []RecoveryRecommendation{},
		PreventionSteps:   []string{},
	}

	// Classify the failure type
	analysis.FailureType = rm.classifyFailure(err)
	analysis.RootCause = err.Error()

	// Generate recommendations based on failure type and operation
	analysis.Recommendations = rm.generateRecoveryRecommendations(operation, analysis.FailureType, err)

	// Analyze affected resources
	analysis.AffectedResources = rm.analyzeAffectedResources(operation)

	// Set confidence based on failure classification accuracy
	analysis.Confidence = rm.calculateAnalysisConfidence(analysis.FailureType, err)

	return analysis
}

// classifyFailure categorizes the type of failure
func (rm *RecoveryManager) classifyFailure(err error) FailureType {
	errStr := err.Error()

	// Network-related errors
	if containsSubstring(errStr, "timeout") || containsSubstring(errStr, "connection") {
		return FailureTypeTimeout
	}

	// Authentication/authorization errors
	if containsSubstring(errStr, "unauthorized") || containsSubstring(errStr, "forbidden") || containsSubstring(errStr, "authentication") {
		return FailureTypeAuthentication
	}

	// Quota/limit errors
	if containsSubstring(errStr, "quota") || containsSubstring(errStr, "limit") || containsSubstring(errStr, "exceeded") {
		return FailureTypeQuota
	}

	// Conflict errors
	if containsSubstring(errStr, "conflict") || containsSubstring(errStr, "already exists") || containsSubstring(errStr, "duplicate") {
		return FailureTypeConflict
	}

	// Validation errors
	if containsSubstring(errStr, "invalid") || containsSubstring(errStr, "validation") || containsSubstring(errStr, "malformed") {
		return FailureTypeValidation
	}

	// Dependency errors
	if containsSubstring(errStr, "dependency") || containsSubstring(errStr, "prerequisite") || containsSubstring(errStr, "not found") {
		return FailureTypeDependency
	}

	// Resource state errors
	if containsSubstring(errStr, "state") || containsSubstring(errStr, "busy") || containsSubstring(errStr, "maintenance") {
		return FailureTypeResource
	}

	// Default to internal for unknown errors
	return FailureTypeInternal
}

// generateRecoveryRecommendations creates recovery recommendations based on failure analysis
func (rm *RecoveryManager) generateRecoveryRecommendations(operation *PlannedOperation, failureType FailureType, err error) []RecoveryRecommendation {
	recommendations := []RecoveryRecommendation{}

	switch failureType {
	case FailureTypeNetwork, FailureTypeTimeout:
		recommendations = append(recommendations, RecoveryRecommendation{
			Strategy:    RecoveryStrategyRetry,
			Description: "Retry operation after network issue resolves",
			Confidence:  0.8,
			Risk:        RiskLevelLow,
			Steps:       []string{"Wait for network connectivity", "Retry operation"},
			Automated:   true,
		})

	case FailureTypeAuthentication:
		recommendations = append(recommendations, RecoveryRecommendation{
			Strategy:    RecoveryStrategyManual,
			Description: "Manual intervention required for authentication issues",
			Confidence:  0.9,
			Risk:        RiskLevelMedium,
			Steps:       []string{"Check credentials", "Verify permissions", "Update configuration"},
			Automated:   false,
		})

	case FailureTypeQuota:
		recommendations = append(recommendations, RecoveryRecommendation{
			Strategy:    RecoveryStrategyManual,
			Description: "Manual intervention required for quota/limit issues",
			Confidence:  0.9,
			Risk:        RiskLevelHigh,
			Steps:       []string{"Review quotas", "Request limit increase", "Modify configuration"},
			Automated:   false,
		})

	case FailureTypeConflict:
		if operation.Type == OperationCreate {
			recommendations = append(recommendations, RecoveryRecommendation{
				Strategy:    RecoveryStrategySkip,
				Description: "Resource already exists, skip creation",
				Confidence:  0.7,
				Risk:        RiskLevelLow,
				Steps:       []string{"Verify resource exists", "Check configuration matches"},
				Automated:   true,
			})
		} else {
			recommendations = append(recommendations, RecoveryRecommendation{
				Strategy:    RecoveryStrategyRetry,
				Description: "Retry after resolving conflict",
				Confidence:  0.6,
				Risk:        RiskLevelMedium,
				Steps:       []string{"Wait for conflict resolution", "Retry operation"},
				Automated:   true,
			})
		}

	case FailureTypeValidation:
		recommendations = append(recommendations, RecoveryRecommendation{
			Strategy:    RecoveryStrategyAbort,
			Description: "Abort due to validation errors in configuration",
			Confidence:  0.9,
			Risk:        RiskLevelLow,
			Steps:       []string{"Fix configuration", "Validate input", "Restart operation"},
			Automated:   false,
		})

	case FailureTypeDependency:
		recommendations = append(recommendations, RecoveryRecommendation{
			Strategy:    RecoveryStrategyRetry,
			Description: "Retry after dependency becomes available",
			Confidence:  0.7,
			Risk:        RiskLevelMedium,
			Steps:       []string{"Check dependency status", "Wait for availability", "Retry operation"},
			Automated:   true,
		})

	case FailureTypeResource:
		recommendations = append(recommendations, RecoveryRecommendation{
			Strategy:    RecoveryStrategyRetry,
			Description: "Retry after resource state stabilizes",
			Confidence:  0.6,
			Risk:        RiskLevelMedium,
			Steps:       []string{"Wait for resource to stabilize", "Check resource status", "Retry operation"},
			Automated:   true,
		})

	default:
		recommendations = append(recommendations, RecoveryRecommendation{
			Strategy:    RecoveryStrategyManual,
			Description: "Manual intervention required for unknown error",
			Confidence:  0.3,
			Risk:        RiskLevelHigh,
			Steps:       []string{"Investigate error", "Determine appropriate action", "Execute recovery"},
			Automated:   false,
		})
	}

	return recommendations
}

// analyzeAffectedResources identifies resources affected by the failure
func (rm *RecoveryManager) analyzeAffectedResources(operation *PlannedOperation) []AffectedResource {
	affected := []AffectedResource{}

	// Analyze the primary resource
	resource := AffectedResource{
		ResourceID:   operation.ResourceName,
		ResourceKind: operation.ResourceType,
		State:        rm.determineResourceState(operation),
		Impact:       rm.determineResourceImpact(operation),
	}

	resource.Recovery = rm.determineResourceRecovery(resource)
	affected = append(affected, resource)

	// Analyze dependent resources using any available idempotency state
	// and simple heuristics derived from operation type/kind.
	// This is a scoped MVP until full dependency graph is wired here.
	if rm.idempotencyManager != nil {
		// Try to determine projectID to filter relevant states
		projectID, _ := rm.getProjectIDForOperation(operation)
		var states []*OperationState
		if projectID != "" {
			states = rm.idempotencyManager.ListOperationStatesByProject(projectID)
		} else {
			states = rm.idempotencyManager.ListOperationStates()
		}

		// Simple dependent resource inference rules
		// - DatabaseUser depends on Cluster in same project
		// - NetworkAccess often associated with Cluster reachability
		for _, st := range states {
			if st == nil {
				continue
			}
			// Skip the primary resource if names match
			if st.ResourceID == operation.ResourceName && st.ResourceKind == operation.ResourceType {
				continue
			}

			// Only consider resources that are likely related to the same project (best-effort via metadata)
			if projectID != "" {
				if st.Metadata == nil {
					continue
				}
				if pid, ok := st.Metadata["projectID"].(string); !ok || pid != projectID {
					continue
				}
			}

			// Heuristics for likely dependencies
			related := false
			switch operation.ResourceType {
			case types.KindDatabaseUser:
				// Likely depends on cluster
				related = st.ResourceKind == types.KindCluster
			case types.KindNetworkAccess:
				// Network access typically applied for clusters reachability
				related = st.ResourceKind == types.KindCluster
			case types.KindCluster:
				// Clusters are parents of users and network access
				related = st.ResourceKind == types.KindDatabaseUser || st.ResourceKind == types.KindNetworkAccess
			}

			if !related {
				continue
			}

			depState := ResourceStateUnknown // nolint:ineffassign // kept for future logic wiring
			switch st.Status {
			case OperationStatusPending:
				depState = ResourceStateUnknown
			case OperationStatusRunning:
				depState = ResourceStatePartial
			case OperationStatusCompleted:
				depState = ResourceStateHealthy
			case OperationStatusFailed:
				depState = ResourceStateInconsistent
			default:
				depState = ResourceStateUnknown
			}

			depImpact := ResourceImpactNone // nolint:ineffassign // kept for future logic wiring
			switch depState {
			case ResourceStatePartial:
				depImpact = ResourceImpactPartial
			case ResourceStateInconsistent:
				depImpact = ResourceImpactCorrupted
			default:
				depImpact = ResourceImpactNone
			}

			dep := AffectedResource{
				ResourceID:   st.ResourceID,
				ResourceKind: st.ResourceKind,
				State:        depState,
				Impact:       depImpact,
			}
			dep.Recovery = rm.determineResourceRecovery(dep)
			affected = append(affected, dep)
		}
	}

	return affected
}

// determineResourceState assesses the current state of a resource
func (rm *RecoveryManager) determineResourceState(operation *PlannedOperation) ResourceState {
	// Check if we have a checkpoint for this operation
	if rm.idempotencyManager != nil {
		if checkpoint, exists := rm.idempotencyManager.GetLatestCheckpoint(operation.ID); exists {
			if checkpoint.ResourceState != nil {
				// Analyze checkpoint data to determine state
				return ResourceStatePartial
			}
		}
	}

	// Check operation state
	if rm.idempotencyManager != nil {
		if state, exists := rm.idempotencyManager.GetOperationState(operation.ID); exists {
			switch state.Status {
			case OperationStatusPending:
				return ResourceStateUnknown
			case OperationStatusRunning:
				return ResourceStatePartial
			case OperationStatusCompleted:
				return ResourceStateHealthy
			case OperationStatusFailed:
				return ResourceStateInconsistent
			}
		}
	}

	return ResourceStateUnknown
}

// determineResourceImpact assesses how the resource was impacted
func (rm *RecoveryManager) determineResourceImpact(operation *PlannedOperation) ResourceImpact {
	state := rm.determineResourceState(operation)

	switch state {
	case ResourceStatePartial:
		return ResourceImpactPartial
	case ResourceStateInconsistent, ResourceStateCorrupted:
		return ResourceImpactCorrupted
	case ResourceStateOrphaned:
		return ResourceImpactOrphaned
	case ResourceStateHealthy:
		return ResourceImpactNone
	default:
		return ResourceImpactPartial
	}
}

// determineResourceRecovery determines recovery options for a resource
func (rm *RecoveryManager) determineResourceRecovery(resource AffectedResource) ResourceRecovery {
	recovery := ResourceRecovery{
		Possible:  true,
		Steps:     []string{},
		Risk:      RiskLevelMedium,
		Automated: false,
	}

	switch resource.State {
	case ResourceStatePartial:
		recovery.Strategy = "cleanup_and_recreate"
		recovery.Steps = []string{"Clean up partial resource", "Recreate from configuration"}
		recovery.Automated = true
		recovery.Risk = RiskLevelLow

	case ResourceStateInconsistent:
		recovery.Strategy = "rollback_and_retry"
		recovery.Steps = []string{"Roll back to previous state", "Retry operation"}
		recovery.Risk = RiskLevelMedium

	case ResourceStateOrphaned:
		recovery.Strategy = "cleanup"
		recovery.Steps = []string{"Clean up orphaned resource"}
		recovery.Automated = true
		recovery.Risk = RiskLevelLow

	case ResourceStateCorrupted:
		recovery.Strategy = "manual_intervention"
		recovery.Steps = []string{"Manual data recovery", "Restore from backup if available"}
		recovery.Risk = RiskLevelHigh
		recovery.Possible = false

	case ResourceStateHealthy:
		recovery.Strategy = "none"
		recovery.Steps = []string{}
		recovery.Risk = RiskLevelLow
		recovery.Automated = true

	default:
		recovery.Strategy = "investigate"
		recovery.Steps = []string{"Investigate resource state", "Determine appropriate action"}
		recovery.Risk = RiskLevelHigh
	}

	return recovery
}

// calculateAnalysisConfidence calculates confidence in the failure analysis
func (rm *RecoveryManager) calculateAnalysisConfidence(failureType FailureType, err error) float64 {
	// Base confidence on how well we can classify the error
	switch failureType {
	case FailureTypeAuthentication, FailureTypeValidation, FailureTypeQuota:
		return 0.9 // High confidence for clear error types
	case FailureTypeNetwork, FailureTypeTimeout, FailureTypeConflict:
		return 0.8 // Good confidence for network/timing issues
	case FailureTypeDependency, FailureTypeResource:
		return 0.7 // Medium confidence for state-related issues
	case FailureTypeInternal:
		return 0.3 // Low confidence for unknown errors
	default:
		return 0.5 // Default medium-low confidence
	}
}

// determineRecoveryStrategy selects the appropriate recovery strategy
func (rm *RecoveryManager) determineRecoveryStrategy(operation *PlannedOperation, err error) RecoveryStrategy {
	// Check for operation-specific strategy
	if strategy, exists := rm.config.OperationStrategies[operation.Type]; exists {
		return strategy
	}

	// Use failure type to determine strategy
	failureType := rm.classifyFailure(err)
	switch failureType {
	case FailureTypeNetwork, FailureTypeTimeout, FailureTypeResource:
		return RecoveryStrategyRetry
	case FailureTypeConflict:
		if operation.Type == OperationCreate {
			return RecoveryStrategySkip
		}
		return RecoveryStrategyRetry
	case FailureTypeValidation:
		return RecoveryStrategyAbort
	case FailureTypeAuthentication, FailureTypeQuota:
		return RecoveryStrategyManual
	default:
		return rm.config.DefaultStrategy
	}
}

// handleRetryRecovery implements retry recovery strategy
func (rm *RecoveryManager) handleRetryRecovery(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	result.NextActions = append(result.NextActions, "Operation will be retried by retry manager")
	return nil // Retry is handled by the retry manager
}

// handleRollbackRecovery implements rollback recovery strategy
func (rm *RecoveryManager) handleRollbackRecovery(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	if !rm.config.EnableRollback {
		return fmt.Errorf("rollback is disabled")
	}

	// Create rollback context with timeout
	rollbackCtx := ctx
	if rm.config.RollbackTimeout > 0 {
		var cancel context.CancelFunc
		rollbackCtx, cancel = context.WithTimeout(ctx, rm.config.RollbackTimeout)
		defer cancel()
	}

	// Perform rollback based on operation type
	switch operation.Type {
	case OperationCreate:
		return rm.rollbackCreate(rollbackCtx, operation, result)
	case OperationUpdate:
		return rm.rollbackUpdate(rollbackCtx, operation, result)
	case OperationDelete:
		return rm.rollbackDelete(rollbackCtx, operation, result)
	default:
		return fmt.Errorf("rollback not supported for operation type: %s", operation.Type)
	}
}

// handleCleanupRecovery implements cleanup recovery strategy
func (rm *RecoveryManager) handleCleanupRecovery(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	if !rm.config.EnableCleanup {
		return fmt.Errorf("cleanup is disabled")
	}

	// Create cleanup context with timeout
	cleanupCtx := ctx
	if rm.config.CleanupTimeout > 0 {
		var cancel context.CancelFunc
		cleanupCtx, cancel = context.WithTimeout(ctx, rm.config.CleanupTimeout)
		defer cancel()
	}

	// Perform cleanup based on operation type and resource kind
	return rm.performCleanup(cleanupCtx, operation, result)
}

// handleSkipRecovery implements skip recovery strategy
func (rm *RecoveryManager) handleSkipRecovery(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	result.NextActions = append(result.NextActions, "Operation skipped, continuing with next operation")
	return nil
}

// handleAbortRecovery implements abort recovery strategy
func (rm *RecoveryManager) handleAbortRecovery(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	result.NextActions = append(result.NextActions, "Execution aborted, fix issues and restart")
	return fmt.Errorf("execution aborted due to unrecoverable error")
}

// handleManualRecovery implements manual recovery strategy
func (rm *RecoveryManager) handleManualRecovery(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	result.NextActions = append(result.NextActions, "Manual intervention required")
	if rm.config.InteractiveMode {
		// TODO: Implement interactive recovery prompts
		return fmt.Errorf("manual intervention required - not implemented in non-interactive mode")
	}
	return fmt.Errorf("manual intervention required")
}

// rollbackCreate rolls back a failed create operation
func (rm *RecoveryManager) rollbackCreate(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	// For create operations, rollback means deleting the partially created resource
	switch operation.ResourceType {
	case types.KindCluster:
		return rm.rollbackClusterCreate(ctx, operation, result)
	case types.KindDatabaseUser:
		return rm.rollbackUserCreate(ctx, operation, result)
	case types.KindNetworkAccess:
		return rm.rollbackNetworkAccessCreate(ctx, operation, result)
	default:
		return fmt.Errorf("rollback not implemented for resource type: %s", operation.ResourceType)
	}
}

// rollbackUpdate rolls back a failed update operation
func (rm *RecoveryManager) rollbackUpdate(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	// For update operations, rollback means restoring to previous state
	// This requires having stored the previous state in a checkpoint
	if rm.idempotencyManager == nil {
		return fmt.Errorf("cannot rollback update: idempotency manager not available")
	}

	checkpoint, exists := rm.idempotencyManager.GetLatestCheckpoint(operation.ID)
	if !exists {
		return fmt.Errorf("cannot rollback update: no checkpoint found")
	}

	// Restore from checkpoint
	switch operation.ResourceType {
	case types.KindCluster:
		return rm.rollbackClusterUpdate(ctx, operation, result, checkpoint)
	case types.KindDatabaseUser:
		return rm.rollbackUserUpdate(ctx, operation, result, checkpoint)
	case types.KindNetworkAccess:
		return rm.rollbackNetworkAccessUpdate(ctx, operation, result, checkpoint)
	default:
		return fmt.Errorf("rollback not implemented for resource type: %s", operation.ResourceType)
	}
}

// rollbackDelete rolls back a failed delete operation
func (rm *RecoveryManager) rollbackDelete(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	// For delete operations, rollback means recreating the resource
	// This requires having stored the resource state before deletion
	if rm.idempotencyManager == nil {
		return fmt.Errorf("cannot rollback delete: idempotency manager not available")
	}

	checkpoint, exists := rm.idempotencyManager.GetLatestCheckpoint(operation.ID)
	if !exists {
		return fmt.Errorf("cannot rollback delete: no checkpoint found")
	}

	// Recreate from checkpoint
	switch operation.ResourceType {
	case types.KindCluster:
		return rm.rollbackClusterDelete(ctx, operation, result, checkpoint)
	case types.KindDatabaseUser:
		return rm.rollbackUserDelete(ctx, operation, result, checkpoint)
	case types.KindNetworkAccess:
		return rm.rollbackNetworkAccessDelete(ctx, operation, result, checkpoint)
	default:
		return fmt.Errorf("rollback not implemented for resource type: %s", operation.ResourceType)
	}
}

// rollbackClusterCreate deletes a partially created cluster
func (rm *RecoveryManager) rollbackClusterCreate(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	if rm.clustersService == nil {
		return fmt.Errorf("clusters service not available for rollback")
	}

	clusterName := operation.ResourceName

	// Extract project ID from checkpoint or idempotency state
	projectID, err := rm.getProjectIDForOperation(operation)
	if err != nil {
		return fmt.Errorf("project ID not available for cluster rollback: %w", err)
	}

	// Try to delete the partially created cluster
	err = rm.clustersService.Delete(ctx, projectID, clusterName)
	if err != nil {
		// Check if cluster doesn't exist (already cleaned up)
		if rm.isNotFoundError(err) {
			result.RollbackPerformed = true
			result.ResourcesCleaned = append(result.ResourcesCleaned, clusterName)
			result.NextActions = append(result.NextActions, fmt.Sprintf("Cluster %s was already deleted", clusterName))
			return nil
		}

		result.NextActions = append(result.NextActions, fmt.Sprintf("Manual cluster cleanup required for %s: %v", clusterName, err))
		return fmt.Errorf("failed to rollback cluster %s: %w", clusterName, err)
	}

	result.RollbackPerformed = true
	result.ResourcesCleaned = append(result.ResourcesCleaned, clusterName)
	result.NextActions = append(result.NextActions, fmt.Sprintf("Successfully rolled back cluster %s", clusterName))

	return nil
}

// rollbackUserCreate deletes a partially created database user
func (rm *RecoveryManager) rollbackUserCreate(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	if rm.usersService == nil {
		return fmt.Errorf("users service not available for rollback")
	}

	userName := operation.ResourceName

	// Extract project ID from operation context
	projectID, err := rm.getProjectIDForOperation(operation)
	if err != nil {
		return fmt.Errorf("project ID not available for user rollback: %w", err)
	}

	// Extract auth database from the user spec (default to "admin")
	authDatabase := "admin"
	if operation.Desired != nil {
		if userManifest, ok := operation.Desired.(*types.DatabaseUserManifest); ok {
			if userManifest.Spec.AuthDatabase != "" {
				authDatabase = userManifest.Spec.AuthDatabase
			}
		}
	}

	// Try to delete the partially created user
	err = rm.usersService.Delete(ctx, projectID, authDatabase, userName)
	if err != nil {
		// Check if user doesn't exist (already cleaned up)
		if rm.isNotFoundError(err) {
			result.RollbackPerformed = true
			result.ResourcesCleaned = append(result.ResourcesCleaned, userName)
			result.NextActions = append(result.NextActions, fmt.Sprintf("Database user %s was already deleted", userName))
			return nil
		}

		result.NextActions = append(result.NextActions, fmt.Sprintf("Manual user cleanup required for %s: %v", userName, err))
		return fmt.Errorf("failed to rollback database user %s: %w", userName, err)
	}

	result.RollbackPerformed = true
	result.ResourcesCleaned = append(result.ResourcesCleaned, userName)
	result.NextActions = append(result.NextActions, fmt.Sprintf("Successfully rolled back database user %s", userName))

	return nil
}

// rollbackNetworkAccessCreate deletes a partially created network access entry
func (rm *RecoveryManager) rollbackNetworkAccessCreate(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	if rm.networkAccessService == nil {
		return fmt.Errorf("network access service not available for rollback")
	}

	// Extract project ID from operation context
	projectID, err := rm.getProjectIDForOperation(operation)
	if err != nil {
		return fmt.Errorf("project ID not available for network access rollback: %w", err)
	}

	// Extract the IP address or CIDR for deletion
	var entryIdentifier string
	if operation.Desired != nil {
		if networkManifest, ok := operation.Desired.(*types.NetworkAccessManifest); ok {
			if networkManifest.Spec.IPAddress != "" {
				entryIdentifier = networkManifest.Spec.IPAddress
			} else if networkManifest.Spec.CIDR != "" {
				entryIdentifier = networkManifest.Spec.CIDR
			} else if networkManifest.Spec.AWSSecurityGroup != "" {
				entryIdentifier = networkManifest.Spec.AWSSecurityGroup
			}
		}
	}

	if entryIdentifier == "" {
		return fmt.Errorf("could not determine network access entry identifier for rollback")
	}

	// Try to delete the partially created network access entry
	err = rm.networkAccessService.Delete(ctx, projectID, entryIdentifier)
	if err != nil {
		// Check if entry doesn't exist (already cleaned up)
		if rm.isNotFoundError(err) {
			result.RollbackPerformed = true
			result.ResourcesCleaned = append(result.ResourcesCleaned, entryIdentifier)
			result.NextActions = append(result.NextActions, fmt.Sprintf("Network access entry %s was already deleted", entryIdentifier))
			return nil
		}

		result.NextActions = append(result.NextActions, fmt.Sprintf("Manual network access cleanup required for %s: %v", entryIdentifier, err))
		return fmt.Errorf("failed to rollback network access entry %s: %w", entryIdentifier, err)
	}

	result.RollbackPerformed = true
	result.ResourcesCleaned = append(result.ResourcesCleaned, entryIdentifier)
	result.NextActions = append(result.NextActions, fmt.Sprintf("Successfully rolled back network access entry %s", entryIdentifier))

	return nil
}

// rollbackClusterUpdate restores cluster to previous state
func (rm *RecoveryManager) rollbackClusterUpdate(ctx context.Context, operation *PlannedOperation, result *RecoveryResult, checkpoint *Checkpoint) error {
	// Extract previous cluster state from checkpoint
	if checkpoint.ResourceState == nil {
		return fmt.Errorf("no resource state in checkpoint for cluster rollback")
	}

	// Restore cluster to previous configuration
	// This is complex and depends on what specifically was being updated
	result.NextActions = append(result.NextActions, "Manual cluster rollback may be required")
	return fmt.Errorf("automatic cluster update rollback not yet implemented")
}

// rollbackUserUpdate restores user to previous state
func (rm *RecoveryManager) rollbackUserUpdate(ctx context.Context, operation *PlannedOperation, result *RecoveryResult, checkpoint *Checkpoint) error {
	// Extract previous user state from checkpoint
	if checkpoint.ResourceState == nil {
		return fmt.Errorf("no resource state in checkpoint for user rollback")
	}

	// Restore user to previous configuration
	result.NextActions = append(result.NextActions, "Manual user rollback may be required")
	return fmt.Errorf("automatic user update rollback not yet implemented")
}

// rollbackNetworkAccessUpdate restores network access to previous state
func (rm *RecoveryManager) rollbackNetworkAccessUpdate(ctx context.Context, operation *PlannedOperation, result *RecoveryResult, checkpoint *Checkpoint) error {
	// Extract previous network access state from checkpoint
	if checkpoint.ResourceState == nil {
		return fmt.Errorf("no resource state in checkpoint for network access rollback")
	}

	// Restore network access to previous configuration
	result.NextActions = append(result.NextActions, "Manual network access rollback may be required")
	return fmt.Errorf("automatic network access update rollback not yet implemented")
}

// rollbackClusterDelete recreates a deleted cluster
func (rm *RecoveryManager) rollbackClusterDelete(ctx context.Context, operation *PlannedOperation, result *RecoveryResult, checkpoint *Checkpoint) error {
	result.NextActions = append(result.NextActions, "Cluster deletion rollback requires manual recreation")
	return fmt.Errorf("cluster deletion rollback not yet implemented")
}

// rollbackUserDelete recreates a deleted user
func (rm *RecoveryManager) rollbackUserDelete(ctx context.Context, operation *PlannedOperation, result *RecoveryResult, checkpoint *Checkpoint) error {
	result.NextActions = append(result.NextActions, "User deletion rollback requires manual recreation")
	return fmt.Errorf("user deletion rollback not yet implemented")
}

// rollbackNetworkAccessDelete recreates a deleted network access entry
func (rm *RecoveryManager) rollbackNetworkAccessDelete(ctx context.Context, operation *PlannedOperation, result *RecoveryResult, checkpoint *Checkpoint) error {
	result.NextActions = append(result.NextActions, "Network access deletion rollback requires manual recreation")
	return fmt.Errorf("network access deletion rollback not yet implemented")
}

// performCleanup cleans up partial resources
func (rm *RecoveryManager) performCleanup(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	// Determine what cleanup is needed based on operation state
	state := rm.determineResourceState(operation)

	switch state {
	case ResourceStatePartial, ResourceStateOrphaned:
		// Clean up partial/orphaned resources
		return rm.cleanupPartialResource(ctx, operation, result)
	case ResourceStateInconsistent:
		// Try to fix inconsistent state
		return rm.fixInconsistentResource(ctx, operation, result)
	default:
		result.NextActions = append(result.NextActions, "No cleanup needed")
		return nil
	}
}

// cleanupPartialResource removes partially created resources
func (rm *RecoveryManager) cleanupPartialResource(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	// This is similar to rollback for create operations
	switch operation.ResourceType {
	case types.KindCluster:
		return rm.rollbackClusterCreate(ctx, operation, result)
	case types.KindDatabaseUser:
		return rm.rollbackUserCreate(ctx, operation, result)
	case types.KindNetworkAccess:
		return rm.rollbackNetworkAccessCreate(ctx, operation, result)
	default:
		return fmt.Errorf("cleanup not implemented for resource type: %s", operation.ResourceType)
	}
}

// fixInconsistentResource attempts to fix resources in inconsistent state
func (rm *RecoveryManager) fixInconsistentResource(ctx context.Context, operation *PlannedOperation, result *RecoveryResult) error {
	result.NextActions = append(result.NextActions, "Manual intervention may be required to fix inconsistent resource state")
	return fmt.Errorf("automatic fix for inconsistent resources not yet implemented")
}

// getProjectIDForOperation extracts project ID from operation context
func (rm *RecoveryManager) getProjectIDForOperation(operation *PlannedOperation) (string, error) {
	// Try to get from operation metadata first
	if operation.Current != nil {
		// Extract project ID from current resource if available
		switch resource := operation.Current.(type) {
		case *types.ClusterManifest:
			if resource.Spec.ProjectName != "" {
				return resource.Spec.ProjectName, nil
			}
		case *types.DatabaseUserManifest:
			if resource.Spec.ProjectName != "" {
				return resource.Spec.ProjectName, nil
			}
		case *types.NetworkAccessManifest:
			if resource.Spec.ProjectName != "" {
				return resource.Spec.ProjectName, nil
			}
		case *types.ProjectManifest:
			if resource.Metadata.Name != "" {
				return resource.Metadata.Name, nil
			}
		}
	}

	// Try to get from desired resource
	if operation.Desired != nil {
		switch resource := operation.Desired.(type) {
		case *types.ClusterManifest:
			if resource.Spec.ProjectName != "" {
				return resource.Spec.ProjectName, nil
			}
		case *types.DatabaseUserManifest:
			if resource.Spec.ProjectName != "" {
				return resource.Spec.ProjectName, nil
			}
		case *types.NetworkAccessManifest:
			if resource.Spec.ProjectName != "" {
				return resource.Spec.ProjectName, nil
			}
		case *types.ProjectManifest:
			if resource.Metadata.Name != "" {
				return resource.Metadata.Name, nil
			}
		}
	}

	// Try to get from idempotency state
	if rm.idempotencyManager != nil {
		if state, exists := rm.idempotencyManager.GetOperationState(operation.ID); exists {
			if projectID, ok := state.Metadata["projectID"].(string); ok && projectID != "" {
				return projectID, nil
			}
		}

		// Try to get from checkpoint
		if checkpoint, exists := rm.idempotencyManager.GetLatestCheckpoint(operation.ID); exists {
			if checkpoint.ResourceState != nil {
				if stateMap, ok := checkpoint.ResourceState.(map[string]interface{}); ok {
					if projectID, ok := stateMap["projectID"].(string); ok && projectID != "" {
						return projectID, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("project ID not found in operation context")
}

// isNotFoundError checks if an error indicates a resource was not found
func (rm *RecoveryManager) isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for common "not found" error patterns
	notFoundPatterns := []string{
		"not found",
		"does not exist",
		"404",
		"NOT_FOUND",
		"CLUSTER_NOT_FOUND",
		"DATABASE_USER_NOT_FOUND",
		"NETWORK_ACCESS_NOT_FOUND",
		"PROJECT_NOT_FOUND",
	}

	for _, pattern := range notFoundPatterns {
		if containsSubstring(errStr, pattern) {
			return true
		}
	}

	return false
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsSubstring(str, substr string) bool {
	return len(substr) > 0 && (str == substr ||
		(len(str) >= len(substr) && (str[:len(substr)] == substr ||
			str[len(str)-len(substr):] == substr ||
			(len(str) > len(substr) && str[1:len(substr)+1] == substr))))
}
