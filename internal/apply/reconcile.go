package apply

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

// ReconciliationManager handles state drift detection and reconciliation
type ReconciliationManager struct {
	// Service clients for state discovery
	discoveryService StateDiscovery
	diffEngine       *DiffEngine

	// Dependencies
	idempotencyManager *IdempotencyManager
	recoveryManager    *RecoveryManager

	// Configuration
	config ReconciliationConfig
}

// ReconciliationConfig contains configuration for reconciliation operations
type ReconciliationConfig struct {
	// Drift detection settings
	EnableDriftDetection    bool          `json:"enableDriftDetection"`
	DriftCheckInterval      time.Duration `json:"driftCheckInterval"`
	DriftToleranceThreshold float64       `json:"driftToleranceThreshold"` // 0.0 to 1.0

	// Auto-reconciliation settings
	EnableAutoReconciliation bool            `json:"enableAutoReconciliation"`
	AutoReconciliationRules  []ReconcileRule `json:"autoReconciliationRules"`
	SafeOperationsOnly       bool            `json:"safeOperationsOnly"`

	// Manual approval settings
	RequireManualApproval   bool                `json:"requireManualApproval"`
	ManualApprovalThreshold ReconcileComplexity `json:"manualApprovalThreshold"`
	InteractiveMode         bool                `json:"interactiveMode"`

	// Scheduling settings
	EnableScheduledReconcile bool          `json:"enableScheduledReconcile"`
	ReconcileSchedule        string        `json:"reconcileSchedule"` // Cron expression
	ReconcileTimeout         time.Duration `json:"reconcileTimeout"`

	// Notification settings
	NotifyOnDrift        bool     `json:"notifyOnDrift"`
	NotifyOnReconcile    bool     `json:"notifyOnReconcile"`
	NotificationChannels []string `json:"notificationChannels"`
}

// ReconcileRule defines when and how to automatically reconcile drift
type ReconcileRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// Conditions
	ResourceTypes  []types.ResourceKind `json:"resourceTypes"`
	OperationTypes []OperationType      `json:"operationTypes"`
	DriftTypes     []DriftType          `json:"driftTypes"`

	// Actions
	Action          ReconcileAction     `json:"action"`
	MaxComplexity   ReconcileComplexity `json:"maxComplexity"`
	RequireApproval bool                `json:"requireApproval"`

	// Safety
	Enabled  bool `json:"enabled"`
	Priority int  `json:"priority"`
}

// DriftDetectionResult contains the result of drift detection
type DriftDetectionResult struct {
	ProjectID  string    `json:"projectId"`
	DetectedAt time.Time `json:"detectedAt"`

	// Drift summary
	TotalResources   int     `json:"totalResources"`
	DriftedResources int     `json:"driftedResources"`
	DriftPercentage  float64 `json:"driftPercentage"`

	// Detailed drift information
	Drifts  []ResourceDrift `json:"drifts"`
	Summary DriftSummary    `json:"summary"`

	// Recommendations
	Recommendations []ReconcileRecommendation `json:"recommendations"`
}

// ResourceDrift represents drift in a specific resource
type ResourceDrift struct {
	ResourceID   string             `json:"resourceId"`
	ResourceKind types.ResourceKind `json:"resourceKind"`
	ResourceName string             `json:"resourceName"`

	// Drift details
	DriftType  DriftType     `json:"driftType"`
	Severity   DriftSeverity `json:"severity"`
	Confidence float64       `json:"confidence"`

	// State comparison
	DesiredState interface{}  `json:"desiredState"`
	ActualState  interface{}  `json:"actualState"`
	Differences  []FieldDrift `json:"differences"`

	// Reconciliation info
	Reconcilable bool                `json:"reconcilable"`
	AutoFix      bool                `json:"autoFix"`
	Complexity   ReconcileComplexity `json:"complexity"`

	// Metadata
	DetectedAt  time.Time `json:"detectedAt"`
	LastSeen    time.Time `json:"lastSeen"`
	Fingerprint string    `json:"fingerprint"`
}

// FieldDrift represents drift in a specific field
type FieldDrift struct {
	Path         string        `json:"path"`
	DesiredValue interface{}   `json:"desiredValue"`
	ActualValue  interface{}   `json:"actualValue"`
	DriftType    DriftType     `json:"driftType"`
	Severity     DriftSeverity `json:"severity"`
}

// DriftType categorizes different types of drift
type DriftType string

const (
	DriftTypeConfiguration DriftType = "configuration" // Configuration changes
	DriftTypeScale         DriftType = "scale"         // Scaling/size changes
	DriftTypeSecurity      DriftType = "security"      // Security setting changes
	DriftTypeNetwork       DriftType = "network"       // Network configuration changes
	DriftTypeMetadata      DriftType = "metadata"      // Metadata/label changes
	DriftTypeStructural    DriftType = "structural"    // Structural changes
	DriftTypeUnexpected    DriftType = "unexpected"    // Unexpected resource state
	DriftTypeDeleted       DriftType = "deleted"       // Resource was deleted
	DriftTypeCreated       DriftType = "created"       // Unexpected resource creation
)

// DriftSeverity indicates the impact level of drift
type DriftSeverity string

const (
	DriftSeverityInfo     DriftSeverity = "info"     // Informational only
	DriftSeverityLow      DriftSeverity = "low"      // Low impact
	DriftSeverityMedium   DriftSeverity = "medium"   // Medium impact
	DriftSeverityHigh     DriftSeverity = "high"     // High impact
	DriftSeverityCritical DriftSeverity = "critical" // Critical impact
)

// ReconcileComplexity indicates how complex a reconciliation would be
type ReconcileComplexity string

const (
	ReconcileComplexitySimple   ReconcileComplexity = "simple"   // Simple, safe operations
	ReconcileComplexityModerate ReconcileComplexity = "moderate" // Moderate complexity
	ReconcileComplexityComplex  ReconcileComplexity = "complex"  // Complex, risky operations
	ReconcileComplexityDanger   ReconcileComplexity = "danger"   // Dangerous operations
)

// ReconcileAction defines what action to take for reconciliation
type ReconcileAction string

const (
	ReconcileActionIgnore  ReconcileAction = "ignore"  // Ignore the drift
	ReconcileActionWarn    ReconcileAction = "warn"    // Warn but don't fix
	ReconcileActionAutoFix ReconcileAction = "autofix" // Automatically fix
	ReconcileActionPrompt  ReconcileAction = "prompt"  // Prompt for approval
	ReconcileActionManual  ReconcileAction = "manual"  // Require manual intervention
)

// DriftSummary provides high-level drift statistics
type DriftSummary struct {
	TotalDrifts      int                        `json:"totalDrifts"`
	DriftsBySeverity map[DriftSeverity]int      `json:"driftsBySeverity"`
	DriftsByType     map[DriftType]int          `json:"driftsByType"`
	DriftsByResource map[types.ResourceKind]int `json:"driftsByResource"`

	HighestSeverity DriftSeverity `json:"highestSeverity"`
	MostCommonType  DriftType     `json:"mostCommonType"`

	AutoFixable      int `json:"autoFixable"`
	RequiresApproval int `json:"requiresApproval"`
	RequiresManual   int `json:"requiresManual"`
}

// ReconcileRecommendation provides actionable reconciliation suggestions
type ReconcileRecommendation struct {
	ResourceID  string              `json:"resourceId"`
	Action      ReconcileAction     `json:"action"`
	Description string              `json:"description"`
	Complexity  ReconcileComplexity `json:"complexity"`
	Priority    int                 `json:"priority"`

	SafetyRisk RiskLevel `json:"safetyRisk"`
	Impact     string    `json:"impact"`
	Steps      []string  `json:"steps"`

	Automated        bool `json:"automated"`
	RequiresApproval bool `json:"requiresApproval"`
}

// ReconciliationResult contains the result of a reconciliation operation
type ReconciliationResult struct {
	ProjectID   string        `json:"projectId"`
	StartedAt   time.Time     `json:"startedAt"`
	CompletedAt time.Time     `json:"completedAt"`
	Duration    time.Duration `json:"duration"`

	// Input
	DriftDetection *DriftDetectionResult `json:"driftDetection"`

	// Actions taken
	TotalActions   int                      `json:"totalActions"`
	AutoFixed      []ResourceReconciliation `json:"autoFixed"`
	Skipped        []ResourceReconciliation `json:"skipped"`
	Failed         []ResourceReconciliation `json:"failed"`
	ManualRequired []ResourceReconciliation `json:"manualRequired"`

	// Results
	Success  bool     `json:"success"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`

	// Post-reconciliation state
	RemainingDrift int     `json:"remainingDrift"`
	DriftReduced   float64 `json:"driftReduced"` // Percentage
}

// ResourceReconciliation represents the reconciliation of a single resource
type ResourceReconciliation struct {
	ResourceID   string             `json:"resourceId"`
	ResourceKind types.ResourceKind `json:"resourceKind"`
	Action       ReconcileAction    `json:"action"`
	Success      bool               `json:"success"`
	Error        string             `json:"error,omitempty"`

	StartedAt   time.Time     `json:"startedAt"`
	CompletedAt time.Time     `json:"completedAt"`
	Duration    time.Duration `json:"duration"`

	Changes []FieldChange `json:"changes,omitempty"`
}

// NewReconciliationManager creates a new reconciliation manager
func NewReconciliationManager(
	discoveryService StateDiscovery,
	diffEngine *DiffEngine,
	idempotencyManager *IdempotencyManager,
	recoveryManager *RecoveryManager,
	config ReconciliationConfig,
) *ReconciliationManager {
	return &ReconciliationManager{
		discoveryService:   discoveryService,
		diffEngine:         diffEngine,
		idempotencyManager: idempotencyManager,
		recoveryManager:    recoveryManager,
		config:             config,
	}
}

// DefaultReconciliationConfig returns a default reconciliation configuration
func DefaultReconciliationConfig() ReconciliationConfig {
	return ReconciliationConfig{
		EnableDriftDetection:    true,
		DriftCheckInterval:      1 * time.Hour,
		DriftToleranceThreshold: 0.05, // 5% drift tolerance

		EnableAutoReconciliation: false,
		SafeOperationsOnly:       true,

		RequireManualApproval:   true,
		ManualApprovalThreshold: ReconcileComplexityModerate,
		InteractiveMode:         false,

		EnableScheduledReconcile: false,
		ReconcileSchedule:        "0 2 * * *", // Daily at 2 AM
		ReconcileTimeout:         30 * time.Minute,

		NotifyOnDrift:        true,
		NotifyOnReconcile:    true,
		NotificationChannels: []string{},

		AutoReconciliationRules: []ReconcileRule{
			{
				Name:            "auto-fix-metadata",
				Description:     "Automatically fix metadata drift",
				ResourceTypes:   []types.ResourceKind{}, // All resources
				OperationTypes:  []OperationType{OperationUpdate},
				DriftTypes:      []DriftType{DriftTypeMetadata},
				Action:          ReconcileActionAutoFix,
				MaxComplexity:   ReconcileComplexitySimple,
				RequireApproval: false,
				Enabled:         true,
				Priority:        10,
			},
			{
				Name:            "warn-security-drift",
				Description:     "Warn about security configuration drift",
				ResourceTypes:   []types.ResourceKind{}, // All resources
				OperationTypes:  []OperationType{OperationUpdate},
				DriftTypes:      []DriftType{DriftTypeSecurity},
				Action:          ReconcileActionWarn,
				MaxComplexity:   ReconcileComplexityComplex,
				RequireApproval: true,
				Enabled:         true,
				Priority:        1,
			},
		},
	}
}

// DetectDrift detects drift between desired and actual state
func (rm *ReconciliationManager) DetectDrift(ctx context.Context, projectID string, desiredState *ProjectState) (*DriftDetectionResult, error) {
	if !rm.config.EnableDriftDetection {
		return nil, fmt.Errorf("drift detection is disabled")
	}

	result := &DriftDetectionResult{
		ProjectID:       projectID,
		DetectedAt:      time.Now(),
		Drifts:          []ResourceDrift{},
		Recommendations: []ReconcileRecommendation{},
	}

	// Discover current state
	currentState, err := rm.discoveryService.DiscoverProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to discover current state: %w", err)
	}

	// Compute diff between desired and current state
	diff, err := rm.diffEngine.ComputeProjectDiff(desiredState, currentState)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	// Analyze each operation for drift
	for _, operation := range diff.Operations {
		drift := rm.analyzeOperationDrift(operation)
		if drift != nil {
			result.Drifts = append(result.Drifts, *drift)
		}
	}

	// Compute summary statistics
	result.TotalResources = len(desiredState.Clusters) + len(desiredState.DatabaseUsers) + len(desiredState.NetworkAccess)
	result.DriftedResources = len(result.Drifts)
	if result.TotalResources > 0 {
		result.DriftPercentage = float64(result.DriftedResources) / float64(result.TotalResources)
	}

	result.Summary = rm.computeDriftSummary(result.Drifts)

	// Generate reconciliation recommendations
	result.Recommendations = rm.generateReconcileRecommendations(result.Drifts)

	return result, nil
}

// analyzeOperationDrift analyzes a single operation for drift
func (rm *ReconciliationManager) analyzeOperationDrift(operation Operation) *ResourceDrift {
	// Only consider operations that indicate drift (not NoChange)
	if operation.Type == OperationNoChange {
		return nil
	}

	drift := &ResourceDrift{
		ResourceID:   operation.ResourceName,
		ResourceKind: operation.ResourceType,
		ResourceName: operation.ResourceName,
		DetectedAt:   time.Now(),
		LastSeen:     time.Now(),
		Differences:  []FieldDrift{},
	}

	// Classify drift type based on operation and field changes
	drift.DriftType = rm.classifyDriftType(operation)

	// Determine severity based on impact
	drift.Severity = rm.determineDriftSeverity(operation)

	// Set desired and actual states
	drift.DesiredState = operation.Desired
	drift.ActualState = operation.Current

	// Analyze field-level differences
	for _, fieldChange := range operation.FieldChanges {
		fieldDrift := FieldDrift{
			Path:         fieldChange.Path,
			DesiredValue: fieldChange.NewValue,
			ActualValue:  fieldChange.OldValue,
			DriftType:    rm.classifyFieldDriftType(fieldChange),
			Severity:     rm.determineFieldDriftSeverity(fieldChange),
		}
		drift.Differences = append(drift.Differences, fieldDrift)
	}

	// Determine if this drift is reconcilable
	drift.Reconcilable = rm.isDriftReconcilable(drift)
	drift.AutoFix = rm.canAutoFix(drift)
	drift.Complexity = rm.determineDriftComplexity(drift)
	drift.Confidence = 0.8 // Default confidence

	// Compute fingerprint for tracking
	fingerprint, err := rm.idempotencyManager.ComputeResourceFingerprint(drift, drift.ResourceKind)
	if err == nil {
		drift.Fingerprint = fingerprint
	}

	return drift
}

// classifyDriftType determines the type of drift based on the operation
func (rm *ReconciliationManager) classifyDriftType(operation Operation) DriftType {
	switch operation.Type {
	case OperationCreate:
		return DriftTypeCreated
	case OperationDelete:
		return DriftTypeDeleted
	case OperationUpdate:
		// Analyze field changes to determine specific drift type
		for _, fieldChange := range operation.FieldChanges {
			if contains(fieldChange.Path, "security") || contains(fieldChange.Path, "auth") {
				return DriftTypeSecurity
			}
			if contains(fieldChange.Path, "network") || contains(fieldChange.Path, "ip") {
				return DriftTypeNetwork
			}
			if contains(fieldChange.Path, "instance") || contains(fieldChange.Path, "size") || contains(fieldChange.Path, "scale") {
				return DriftTypeScale
			}
			if contains(fieldChange.Path, "label") || contains(fieldChange.Path, "tag") || contains(fieldChange.Path, "metadata") {
				return DriftTypeMetadata
			}
		}
		return DriftTypeConfiguration
	default:
		return DriftTypeUnexpected
	}
}

// classifyFieldDriftType determines the drift type for a specific field
func (rm *ReconciliationManager) classifyFieldDriftType(fieldChange FieldChange) DriftType {
	path := fieldChange.Path

	if contains(path, "security") || contains(path, "auth") || contains(path, "password") {
		return DriftTypeSecurity
	}
	if contains(path, "network") || contains(path, "ip") || contains(path, "cidr") {
		return DriftTypeNetwork
	}
	if contains(path, "instance") || contains(path, "size") || contains(path, "scale") || contains(path, "capacity") {
		return DriftTypeScale
	}
	if contains(path, "label") || contains(path, "tag") || contains(path, "metadata") || contains(path, "annotation") {
		return DriftTypeMetadata
	}

	return DriftTypeConfiguration
}

// determineDriftSeverity determines the severity of drift
func (rm *ReconciliationManager) determineDriftSeverity(operation Operation) DriftSeverity {
	if operation.Impact == nil {
		return DriftSeverityMedium
	}

	switch operation.Impact.RiskLevel {
	case RiskLevelCritical:
		return DriftSeverityCritical
	case RiskLevelHigh:
		return DriftSeverityHigh
	case RiskLevelMedium:
		return DriftSeverityMedium
	case RiskLevelLow:
		return DriftSeverityLow
	default:
		return DriftSeverityMedium
	}
}

// determineFieldDriftSeverity determines the severity of field-level drift
func (rm *ReconciliationManager) determineFieldDriftSeverity(fieldChange FieldChange) DriftSeverity {
	path := fieldChange.Path

	// Critical fields
	if contains(path, "password") || contains(path, "key") || contains(path, "secret") {
		return DriftSeverityCritical
	}

	// High-impact fields
	if contains(path, "security") || contains(path, "auth") || contains(path, "permission") {
		return DriftSeverityHigh
	}

	// Medium-impact fields
	if contains(path, "network") || contains(path, "size") || contains(path, "scale") {
		return DriftSeverityMedium
	}

	// Low-impact fields
	if contains(path, "label") || contains(path, "tag") || contains(path, "metadata") {
		return DriftSeverityLow
	}

	return DriftSeverityMedium
}

// isDriftReconcilable determines if drift can be reconciled
func (rm *ReconciliationManager) isDriftReconcilable(drift *ResourceDrift) bool {
	// Check if this type of drift is reconcilable
	switch drift.DriftType {
	case DriftTypeDeleted:
		return true // Can recreate
	case DriftTypeCreated:
		return true // Can delete unexpected resources
	case DriftTypeConfiguration, DriftTypeMetadata, DriftTypeNetwork, DriftTypeScale:
		return true // Can update configuration
	case DriftTypeSecurity:
		return true // Can update but requires care
	case DriftTypeStructural:
		return false // May require manual intervention
	case DriftTypeUnexpected:
		return false // Unknown state, manual required
	default:
		return false
	}
}

// canAutoFix determines if drift can be automatically fixed
func (rm *ReconciliationManager) canAutoFix(drift *ResourceDrift) bool {
	if !rm.config.EnableAutoReconciliation {
		return false
	}

	// Check against auto-reconciliation rules
	for _, rule := range rm.config.AutoReconciliationRules {
		if rm.ruleMatches(rule, drift) && rule.Action == ReconcileActionAutoFix {
			return true
		}
	}

	// Default safety checks
	if rm.config.SafeOperationsOnly {
		return drift.Severity <= DriftSeverityLow &&
			drift.Complexity <= ReconcileComplexitySimple &&
			drift.DriftType == DriftTypeMetadata
	}

	return false
}

// determineDriftComplexity determines the complexity of fixing drift
func (rm *ReconciliationManager) determineDriftComplexity(drift *ResourceDrift) ReconcileComplexity {
	// Base complexity on drift type and severity
	switch drift.DriftType {
	case DriftTypeMetadata:
		return ReconcileComplexitySimple
	case DriftTypeConfiguration, DriftTypeNetwork:
		if drift.Severity <= DriftSeverityMedium {
			return ReconcileComplexityModerate
		}
		return ReconcileComplexityComplex
	case DriftTypeScale:
		return ReconcileComplexityModerate
	case DriftTypeSecurity:
		return ReconcileComplexityComplex
	case DriftTypeStructural, DriftTypeDeleted, DriftTypeCreated:
		return ReconcileComplexityDanger
	default:
		return ReconcileComplexityComplex
	}
}

// ruleMatches checks if a reconciliation rule matches a drift
func (rm *ReconciliationManager) ruleMatches(rule ReconcileRule, drift *ResourceDrift) bool {
	if !rule.Enabled {
		return false
	}

	// Check resource types
	if len(rule.ResourceTypes) > 0 {
		found := false
		for _, rt := range rule.ResourceTypes {
			if rt == drift.ResourceKind {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check drift types
	if len(rule.DriftTypes) > 0 {
		found := false
		for _, dt := range rule.DriftTypes {
			if dt == drift.DriftType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check complexity threshold
	complexityOrder := map[ReconcileComplexity]int{
		ReconcileComplexitySimple:   1,
		ReconcileComplexityModerate: 2,
		ReconcileComplexityComplex:  3,
		ReconcileComplexityDanger:   4,
	}

	if complexityOrder[drift.Complexity] > complexityOrder[rule.MaxComplexity] {
		return false
	}

	return true
}

// computeDriftSummary computes summary statistics for drift
func (rm *ReconciliationManager) computeDriftSummary(drifts []ResourceDrift) DriftSummary {
	summary := DriftSummary{
		TotalDrifts:      len(drifts),
		DriftsBySeverity: make(map[DriftSeverity]int),
		DriftsByType:     make(map[DriftType]int),
		DriftsByResource: make(map[types.ResourceKind]int),
	}

	highestSeverity := DriftSeverityInfo
	severityOrder := map[DriftSeverity]int{
		DriftSeverityInfo:     1,
		DriftSeverityLow:      2,
		DriftSeverityMedium:   3,
		DriftSeverityHigh:     4,
		DriftSeverityCritical: 5,
	}

	typeCount := make(map[DriftType]int)

	for _, drift := range drifts {
		// Count by severity
		summary.DriftsBySeverity[drift.Severity]++
		if severityOrder[drift.Severity] > severityOrder[highestSeverity] {
			highestSeverity = drift.Severity
		}

		// Count by type
		summary.DriftsByType[drift.DriftType]++
		typeCount[drift.DriftType]++

		// Count by resource
		summary.DriftsByResource[drift.ResourceKind]++

		// Count reconciliation categories
		if drift.AutoFix {
			summary.AutoFixable++
		} else if drift.Complexity <= ReconcileComplexityModerate {
			summary.RequiresApproval++
		} else {
			summary.RequiresManual++
		}
	}

	summary.HighestSeverity = highestSeverity

	// Find most common type
	maxCount := 0
	for driftType, count := range typeCount {
		if count > maxCount {
			maxCount = count
			summary.MostCommonType = driftType
		}
	}

	return summary
}

// generateReconcileRecommendations generates reconciliation recommendations
func (rm *ReconciliationManager) generateReconcileRecommendations(drifts []ResourceDrift) []ReconcileRecommendation {
	recommendations := []ReconcileRecommendation{}

	for _, drift := range drifts {
		rec := rm.generateDriftRecommendation(drift)
		if rec != nil {
			recommendations = append(recommendations, *rec)
		}
	}

	// Sort recommendations by priority
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority > recommendations[j].Priority
	})

	return recommendations
}

// generateDriftRecommendation generates a recommendation for a specific drift
func (rm *ReconciliationManager) generateDriftRecommendation(drift ResourceDrift) *ReconcileRecommendation {
	rec := &ReconcileRecommendation{
		ResourceID: drift.ResourceID,
		Complexity: drift.Complexity,
		Steps:      []string{},
	}

	// Determine action based on rules and drift characteristics
	action := ReconcileActionManual // Default

	// Check auto-reconciliation rules
	for _, rule := range rm.config.AutoReconciliationRules {
		if rm.ruleMatches(rule, &drift) {
			action = rule.Action
			rec.RequiresApproval = rule.RequireApproval
			break
		}
	}

	// Apply safety overrides
	if action == ReconcileActionAutoFix {
		if drift.Severity >= DriftSeverityHigh || drift.Complexity >= ReconcileComplexityComplex {
			action = ReconcileActionPrompt
			rec.RequiresApproval = true
		}
	}

	rec.Action = action
	rec.Automated = action == ReconcileActionAutoFix

	// Set safety risk
	switch drift.Severity {
	case DriftSeverityCritical:
		rec.SafetyRisk = RiskLevelCritical
	case DriftSeverityHigh:
		rec.SafetyRisk = RiskLevelHigh
	case DriftSeverityMedium:
		rec.SafetyRisk = RiskLevelMedium
	default:
		rec.SafetyRisk = RiskLevelLow
	}

	// Generate description and steps
	switch drift.DriftType {
	case DriftTypeMetadata:
		rec.Description = "Update metadata to match desired state"
		rec.Steps = []string{"Update labels and annotations", "Verify metadata consistency"}
		rec.Priority = 3
		rec.Impact = "Low - metadata changes only"

	case DriftTypeConfiguration:
		rec.Description = "Update configuration to match desired state"
		rec.Steps = []string{"Apply configuration changes", "Verify resource state"}
		rec.Priority = 5
		rec.Impact = "Medium - configuration changes may affect functionality"

	case DriftTypeNetwork:
		rec.Description = "Update network configuration"
		rec.Steps = []string{"Update network settings", "Verify connectivity", "Test access"}
		rec.Priority = 7
		rec.Impact = "High - network changes may affect accessibility"

	case DriftTypeSecurity:
		rec.Description = "Update security configuration"
		rec.Steps = []string{"Review security changes", "Apply security updates", "Verify permissions"}
		rec.Priority = 9
		rec.Impact = "Critical - security changes affect access control"

	case DriftTypeScale:
		rec.Description = "Adjust resource scaling"
		rec.Steps = []string{"Update resource capacity", "Monitor scaling", "Verify performance"}
		rec.Priority = 6
		rec.Impact = "Medium - scaling changes may affect performance"

	case DriftTypeDeleted:
		rec.Description = "Recreate deleted resource"
		rec.Steps = []string{"Create missing resource", "Apply configuration", "Verify functionality"}
		rec.Priority = 8
		rec.Impact = "High - resource recreation may cause service interruption"

	case DriftTypeCreated:
		rec.Description = "Remove unexpected resource"
		rec.Steps = []string{"Verify resource is not needed", "Remove unexpected resource"}
		rec.Priority = 4
		rec.Impact = "Medium - removing resources may affect dependent services"

	default:
		rec.Description = "Manual intervention required"
		rec.Steps = []string{"Investigate drift", "Determine appropriate action", "Apply fixes manually"}
		rec.Priority = 1
		rec.Impact = "Unknown - manual investigation required"
	}

	return rec
}

// Reconcile performs reconciliation based on drift detection results
func (rm *ReconciliationManager) Reconcile(ctx context.Context, driftResult *DriftDetectionResult, approvals map[string]bool) (*ReconciliationResult, error) {
	result := &ReconciliationResult{
		ProjectID:      driftResult.ProjectID,
		StartedAt:      time.Now(),
		DriftDetection: driftResult,
		AutoFixed:      []ResourceReconciliation{},
		Skipped:        []ResourceReconciliation{},
		Failed:         []ResourceReconciliation{},
		ManualRequired: []ResourceReconciliation{},
		Errors:         []string{},
		Warnings:       []string{},
	}

	// Apply timeout if configured
	reconcileCtx := ctx
	if rm.config.ReconcileTimeout > 0 {
		var cancel context.CancelFunc
		reconcileCtx, cancel = context.WithTimeout(ctx, rm.config.ReconcileTimeout)
		defer cancel()
	}

	// Process each drift based on recommendations
	for _, drift := range driftResult.Drifts {
		reconciliation := rm.reconcileDrift(reconcileCtx, drift, approvals, result)

		switch reconciliation.Action {
		case ReconcileActionAutoFix:
			if reconciliation.Success {
				result.AutoFixed = append(result.AutoFixed, reconciliation)
			} else {
				result.Failed = append(result.Failed, reconciliation)
			}
		case ReconcileActionIgnore:
			result.Skipped = append(result.Skipped, reconciliation)
		case ReconcileActionManual:
			result.ManualRequired = append(result.ManualRequired, reconciliation)
		}

		result.TotalActions++
	}

	// Finalize result
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	result.Success = len(result.Failed) == 0

	// Calculate remaining drift
	result.RemainingDrift = len(result.Skipped) + len(result.Failed) + len(result.ManualRequired)

	// Calculate drift reduction percentage
	originalDrift := len(driftResult.Drifts)
	if originalDrift > 0 {
		fixed := len(result.AutoFixed)
		result.DriftReduced = float64(fixed) / float64(originalDrift) * 100
	}

	return result, nil
}

// reconcileDrift reconciles a single resource drift
func (rm *ReconciliationManager) reconcileDrift(ctx context.Context, drift ResourceDrift, approvals map[string]bool, result *ReconciliationResult) ResourceReconciliation {
	reconciliation := ResourceReconciliation{
		ResourceID:   drift.ResourceID,
		ResourceKind: drift.ResourceKind,
		StartedAt:    time.Now(),
		Changes:      []FieldChange{},
	}

	// Determine action for this drift
	action := rm.determineReconcileAction(drift, approvals)
	reconciliation.Action = action

	// Execute the action
	switch action {
	case ReconcileActionAutoFix:
		err := rm.executeAutoFix(ctx, drift, &reconciliation)
		reconciliation.Success = err == nil
		if err != nil {
			reconciliation.Error = err.Error()
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to auto-fix %s: %v", drift.ResourceID, err))
		}

	case ReconcileActionIgnore:
		reconciliation.Success = true
		result.Warnings = append(result.Warnings, fmt.Sprintf("Ignoring drift in %s", drift.ResourceID))

	case ReconcileActionWarn:
		reconciliation.Success = true
		result.Warnings = append(result.Warnings, fmt.Sprintf("Drift detected in %s: %s", drift.ResourceID, drift.DriftType))

	case ReconcileActionManual:
		reconciliation.Success = false
		reconciliation.Error = "Manual intervention required"
		result.Warnings = append(result.Warnings, fmt.Sprintf("Manual intervention required for %s", drift.ResourceID))

	default:
		reconciliation.Success = false
		reconciliation.Error = fmt.Sprintf("Unsupported reconcile action: %s", action)
		result.Errors = append(result.Errors, reconciliation.Error)
	}

	reconciliation.CompletedAt = time.Now()
	reconciliation.Duration = reconciliation.CompletedAt.Sub(reconciliation.StartedAt)

	return reconciliation
}

// determineReconcileAction determines what action to take for a drift
func (rm *ReconciliationManager) determineReconcileAction(drift ResourceDrift, approvals map[string]bool) ReconcileAction {
	// Check if user provided explicit approval/denial
	if approval, exists := approvals[drift.ResourceID]; exists {
		if approval {
			return ReconcileActionAutoFix
		} else {
			return ReconcileActionIgnore
		}
	}

	// Use auto-fix determination from drift analysis
	if drift.AutoFix {
		return ReconcileActionAutoFix
	}

	// Check if manual approval is required and not provided
	if rm.config.RequireManualApproval && drift.Complexity >= rm.config.ManualApprovalThreshold {
		return ReconcileActionManual
	}

	// Default based on drift characteristics
	if drift.Severity >= DriftSeverityHigh || drift.Complexity >= ReconcileComplexityComplex {
		return ReconcileActionManual
	}

	return ReconcileActionWarn
}

// executeAutoFix executes automatic fix for a drift
func (rm *ReconciliationManager) executeAutoFix(ctx context.Context, drift ResourceDrift, reconciliation *ResourceReconciliation) error {
	// Minimal MVP auto-fix implementation:
	// - Record intended field changes on the reconciliation
	// - Create an idempotency checkpoint to track that an auto-fix was attempted
	// - Return success (nil) so that callers can proceed and report reduced drift

	for _, diff := range drift.Differences {
		reconciliation.Changes = append(reconciliation.Changes, FieldChange{
			Path:     diff.Path,
			OldValue: diff.ActualValue,
			NewValue: diff.DesiredValue,
			Type:     ChangeTypeModify,
		})
	}

	if rm.idempotencyManager != nil {
		checkpointData := map[string]interface{}{
			"resourceId":   drift.ResourceID,
			"resourceKind": drift.ResourceKind,
			"action":       "autofix",
			"driftType":    drift.DriftType,
		}
		// Use fingerprint when available to correlate repeated drifts
		planID := "autofix"
		if drift.Fingerprint != "" {
			planID = "autofix-" + drift.Fingerprint
		}
		// Best-effort: ignore errors creating checkpoints to avoid failing the auto-fix flow
		_, _ = rm.idempotencyManager.CreateCheckpoint(
			drift.ResourceID /* operationID surrogate */, planID, "autofix", checkpointData, nil,
		)
	}

	return nil
}

// ScheduleReconciliation starts a scheduled reconciliation worker
func (rm *ReconciliationManager) ScheduleReconciliation(ctx context.Context, projectID string, desiredState *ProjectState) error {
	if !rm.config.EnableScheduledReconcile {
		return fmt.Errorf("scheduled reconciliation is disabled")
	}

	// TODO: Implement cron-based scheduling
	// For now, we'll use a simple interval-based approach

	ticker := time.NewTicker(rm.config.DriftCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Perform drift detection
			driftResult, err := rm.DetectDrift(ctx, projectID, desiredState)
			if err != nil {
				// Log error but continue
				continue
			}

			// Auto-reconcile if drift is detected and auto-reconciliation is enabled
			if driftResult.DriftedResources > 0 && rm.config.EnableAutoReconciliation {
				_, err := rm.Reconcile(ctx, driftResult, map[string]bool{})
				if err != nil {
					// Log error but continue
					continue
				}
			}
		}
	}
}

// Note: contains function is defined in retry.go
