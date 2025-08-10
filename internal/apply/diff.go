package apply

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

// OperationType represents the type of change in a diff
type OperationType string

const (
	OperationCreate   OperationType = "Create"
	OperationUpdate   OperationType = "Update"
	OperationDelete   OperationType = "Delete"
	OperationNoChange OperationType = "NoChange"
)

// Diff represents the complete set of changes between desired and current state
type Diff struct {
	ProjectID   string      `json:"projectId"`
	Operations  []Operation `json:"operations"`
	Summary     DiffSummary `json:"summary"`
	GeneratedAt time.Time   `json:"generatedAt"`
}

// Operation represents a single change operation
type Operation struct {
	Type         OperationType      `json:"type"`
	ResourceType types.ResourceKind `json:"resourceType"`
	ResourceName string             `json:"resourceName"`
	Current      interface{}        `json:"current,omitempty"`
	Desired      interface{}        `json:"desired,omitempty"`
	FieldChanges []FieldChange      `json:"fieldChanges,omitempty"`
	Impact       *OperationImpact   `json:"impact,omitempty"`
}

// FieldChange represents a change to a specific field
type FieldChange struct {
	Path     string      `json:"path"`
	OldValue interface{} `json:"oldValue,omitempty"`
	NewValue interface{} `json:"newValue,omitempty"`
	Type     ChangeType  `json:"type"`
}

// ChangeType represents the type of field change
type ChangeType string

const (
	ChangeTypeAdd    ChangeType = "add"
	ChangeTypeRemove ChangeType = "remove"
	ChangeTypeModify ChangeType = "modify"
)

// OperationImpact represents the impact assessment of an operation
type OperationImpact struct {
	IsDestructive     bool          `json:"isDestructive"`
	RequiresDowntime  bool          `json:"requiresDowntime"`
	EstimatedDuration time.Duration `json:"estimatedDuration"`
	RiskLevel         RiskLevel     `json:"riskLevel"`
	Warnings          []string      `json:"warnings,omitempty"`
}

// RiskLevel represents the risk level of an operation
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// DiffSummary provides high-level statistics about the diff
type DiffSummary struct {
	TotalOperations       int           `json:"totalOperations"`
	CreateOperations      int           `json:"createOperations"`
	UpdateOperations      int           `json:"updateOperations"`
	DeleteOperations      int           `json:"deleteOperations"`
	NoChangeOperations    int           `json:"noChangeOperations"`
	DestructiveOperations int           `json:"destructiveOperations"`
	EstimatedDuration     time.Duration `json:"estimatedDuration"`
	HighestRiskLevel      RiskLevel     `json:"highestRiskLevel"`
}

// DiffEngine provides methods for computing diffs between states
type DiffEngine struct {
	// Configuration options for diff computation
	IgnoreOrderInSlices bool
	CompareTimestamps   bool
	IgnoreDefaults      bool
	PreserveExisting    bool
}

// NewDiffEngine creates a new diff engine with default settings
func NewDiffEngine() *DiffEngine {
	return &DiffEngine{
		IgnoreOrderInSlices: true,
		CompareTimestamps:   false,
		IgnoreDefaults:      true,
	}
}

// ComputeProjectDiff computes the diff between desired and current project state
func (d *DiffEngine) ComputeProjectDiff(desired *ProjectState, current *ProjectState) (*Diff, error) {
	diff := &Diff{
		ProjectID:   extractProjectID(desired, current),
		Operations:  []Operation{},
		GeneratedAt: time.Now().UTC(),
	}

	// Compute diffs for each resource type
	if err := d.computeProjectSettingsDiff(desired, current, diff); err != nil {
		return nil, fmt.Errorf("failed to compute project settings diff: %w", err)
	}

	if err := d.computeClustersDiff(desired, current, diff); err != nil {
		return nil, fmt.Errorf("failed to compute clusters diff: %w", err)
	}

	if err := d.computeDatabaseUsersDiff(desired, current, diff); err != nil {
		return nil, fmt.Errorf("failed to compute database users diff: %w", err)
	}

	if err := d.computeNetworkAccessDiff(desired, current, diff); err != nil {
		return nil, fmt.Errorf("failed to compute network access diff: %w", err)
	}

	// Compute summary
	diff.Summary = d.computeSummary(diff.Operations)

	return diff, nil
}

// computeProjectSettingsDiff computes diffs for project settings
func (d *DiffEngine) computeProjectSettingsDiff(desired *ProjectState, current *ProjectState, diff *Diff) error {
	var desiredProject *types.ProjectManifest
	var currentProject *types.ProjectManifest

	if desired != nil {
		desiredProject = desired.Project
	}
	if current != nil {
		currentProject = current.Project
	}

	// Skip if both projects are nil
	if desiredProject == nil && currentProject == nil {
		return nil
	}

	op := d.computeResourceDiff(
		types.KindProject,
		getResourceName(desiredProject, currentProject),
		desiredProject,
		currentProject,
	)

	if op != nil {
		diff.Operations = append(diff.Operations, *op)
	}

	return nil
}

// computeClustersDiff computes diffs for clusters
func (d *DiffEngine) computeClustersDiff(desired *ProjectState, current *ProjectState, diff *Diff) error {
	desiredMap := make(map[string]interface{})
	currentMap := make(map[string]interface{})

	if desired != nil {
		for i := range desired.Clusters {
			cluster := &desired.Clusters[i]
			desiredMap[cluster.Metadata.Name] = cluster
		}
	}

	if current != nil {
		for i := range current.Clusters {
			cluster := &current.Clusters[i]
			currentMap[cluster.Metadata.Name] = cluster
		}
	}

	d.computeDiffFromNamedMaps(types.KindCluster, desiredMap, currentMap, diff)
	return nil
}

// computeDatabaseUsersDiff computes diffs for database users
func (d *DiffEngine) computeDatabaseUsersDiff(desired *ProjectState, current *ProjectState, diff *Diff) error {
	desiredUsers := make(map[string]*types.DatabaseUserManifest)
	currentUsers := make(map[string]*types.DatabaseUserManifest)

	if desired != nil {
		for i := range desired.DatabaseUsers {
			user := &desired.DatabaseUsers[i]
			// Use a composite key: authDatabase/username
			key := fmt.Sprintf("%s/%s", user.Spec.AuthDatabase, user.Spec.Username)
			desiredUsers[key] = user
		}
	}

	if current != nil {
		for i := range current.DatabaseUsers {
			user := &current.DatabaseUsers[i]
			key := fmt.Sprintf("%s/%s", user.Spec.AuthDatabase, user.Spec.Username)
			currentUsers[key] = user
		}
	}

	// Find all unique user keys
	allKeys := make(map[string]bool)
	for key := range desiredUsers {
		allKeys[key] = true
	}
	for key := range currentUsers {
		allKeys[key] = true
	}

	// Compute diff for each user
	for key := range allKeys {
		desired := desiredUsers[key]
		current := currentUsers[key]

		var resourceName string
		if desired != nil {
			resourceName = desired.Metadata.Name
		} else if current != nil {
			resourceName = current.Metadata.Name
		}

		op := d.computeResourceDiff(types.KindDatabaseUser, resourceName, desired, current)
		if op != nil {
			diff.Operations = append(diff.Operations, *op)
		}
	}

	return nil
}

// computeNetworkAccessDiff computes diffs for network access entries
func (d *DiffEngine) computeNetworkAccessDiff(desired *ProjectState, current *ProjectState, diff *Diff) error {
	desiredMap := make(map[string]interface{})
	currentMap := make(map[string]interface{})

	if desired != nil {
		for i := range desired.NetworkAccess {
			entry := &desired.NetworkAccess[i]
			desiredMap[entry.Metadata.Name] = entry
		}
	}

	if current != nil {
		for i := range current.NetworkAccess {
			entry := &current.NetworkAccess[i]
			currentMap[entry.Metadata.Name] = entry
		}
	}

	d.computeDiffFromNamedMaps(types.KindNetworkAccess, desiredMap, currentMap, diff)
	return nil
}

// computeDiffFromNamedMaps computes diffs given name-indexed desired and current maps
func (d *DiffEngine) computeDiffFromNamedMaps(resourceType types.ResourceKind, desiredMap, currentMap map[string]interface{}, diff *Diff) {
	// Find all unique names
	allNames := make(map[string]struct{})
	for name := range desiredMap {
		allNames[name] = struct{}{}
	}
	for name := range currentMap {
		allNames[name] = struct{}{}
	}

	// Compute diff for each name
	for name := range allNames {
		desired := desiredMap[name]
		current := currentMap[name]

		op := d.computeResourceDiff(resourceType, name, desired, current)
		if op != nil {
			diff.Operations = append(diff.Operations, *op)
		}
	}
}

// computeResourceDiff computes the diff for a single resource
func (d *DiffEngine) computeResourceDiff(resourceType types.ResourceKind, resourceName string, desired, current interface{}) *Operation {
	// Handle Go's interface{} nil gotcha - check for typed nil values
	if desired != nil {
		switch v := desired.(type) {
		case *types.ClusterManifest:
			if v == nil {
				desired = nil
			}
		case *types.DatabaseUserManifest:
			if v == nil {
				desired = nil
			}
		case *types.NetworkAccessManifest:
			if v == nil {
				desired = nil
			}
		case *types.ProjectManifest:
			if v == nil {
				desired = nil
			}
		}
	}

	if current != nil {
		switch v := current.(type) {
		case *types.ClusterManifest:
			if v == nil {
				current = nil
			}
		case *types.DatabaseUserManifest:
			if v == nil {
				current = nil
			}
		case *types.NetworkAccessManifest:
			if v == nil {
				current = nil
			}
		case *types.ProjectManifest:
			if v == nil {
				current = nil
			}
		}
	}

	if desired == nil && current == nil {
		return nil
	}

	op := &Operation{
		ResourceType: resourceType,
		ResourceName: resourceName,
		Current:      current,
		Desired:      desired,
	}

	// Determine operation type
	if desired == nil {
		if d.PreserveExisting {
			// Skip delete operations when preserving existing resources
			return nil
		}
		op.Type = OperationDelete
	} else if current == nil {
		op.Type = OperationCreate
	} else {
		// Compare the resources to see if they're different
		if d.resourcesEqual(desired, current) {
			op.Type = OperationNoChange
		} else {
			op.Type = OperationUpdate
			// Compute field-level changes
			op.FieldChanges = d.computeFieldChanges(desired, current)
		}
	}

	// Compute impact assessment
	op.Impact = d.computeOperationImpact(op)

	return op
}

// resourcesEqual compares two resources for equality
func (d *DiffEngine) resourcesEqual(desired, current interface{}) bool {
	if desired == nil && current == nil {
		return true
	}
	if desired == nil || current == nil {
		return false
	}

	// Convert to JSON for comparison to handle complex nested structures
	desiredJSON, err1 := json.Marshal(d.normalizeForComparison(desired))
	currentJSON, err2 := json.Marshal(d.normalizeForComparison(current))

	if err1 != nil || err2 != nil {
		// Fallback to reflect.DeepEqual
		return reflect.DeepEqual(desired, current)
	}

	return string(desiredJSON) == string(currentJSON)
}

// normalizeForComparison normalizes resources for comparison
func (d *DiffEngine) normalizeForComparison(resource interface{}) interface{} {
	if resource == nil {
		return nil
	}

	// Create a copy and remove fields that shouldn't be compared
	resourceValue := reflect.ValueOf(resource)
	if resourceValue.Kind() == reflect.Ptr {
		if resourceValue.IsNil() {
			return nil
		}
		resourceValue = resourceValue.Elem()
	}

	// Handle different manifest types
	switch v := resource.(type) {
	case *types.ClusterManifest:
		if v == nil {
			return nil
		}
		normalized := *v
		// Remove status and metadata fields that shouldn't be compared
		normalized.Status = nil
		if d.IgnoreDefaults {
			// Remove default values from comparison
		}
		return normalized
	case *types.DatabaseUserManifest:
		if v == nil {
			return nil
		}
		normalized := *v
		normalized.Status = nil
		// Don't compare password for security
		normalized.Spec.Password = ""
		return normalized
	case *types.NetworkAccessManifest:
		if v == nil {
			return nil
		}
		normalized := *v
		normalized.Status = nil
		return normalized
	case *types.ProjectManifest:
		if v == nil {
			return nil
		}
		normalized := *v
		normalized.Status = nil
		return normalized
	default:
		return resource
	}
}

// computeFieldChanges computes field-level changes between two resources
func (d *DiffEngine) computeFieldChanges(desired, current interface{}) []FieldChange {
	changes := []FieldChange{}

	desiredValue := reflect.ValueOf(desired)
	currentValue := reflect.ValueOf(current)

	if desiredValue.Kind() == reflect.Ptr {
		desiredValue = desiredValue.Elem()
	}
	if currentValue.Kind() == reflect.Ptr {
		currentValue = currentValue.Elem()
	}

	d.compareFields("", desiredValue, currentValue, &changes)

	return changes
}

// compareFields recursively compares fields and builds field changes
func (d *DiffEngine) compareFields(path string, desired, current reflect.Value, changes *[]FieldChange) {
	if !desired.IsValid() && !current.IsValid() {
		return
	}

	if !desired.IsValid() {
		*changes = append(*changes, FieldChange{
			Path:     path,
			OldValue: getReflectValue(current),
			Type:     ChangeTypeRemove,
		})
		return
	}

	if !current.IsValid() {
		*changes = append(*changes, FieldChange{
			Path:     path,
			NewValue: getReflectValue(desired),
			Type:     ChangeTypeAdd,
		})
		return
	}

	if desired.Type() != current.Type() {
		*changes = append(*changes, FieldChange{
			Path:     path,
			OldValue: getReflectValue(current),
			NewValue: getReflectValue(desired),
			Type:     ChangeTypeModify,
		})
		return
	}

	switch desired.Kind() {
	case reflect.Struct:
		d.compareStructFields(path, desired, current, changes)
	case reflect.Slice, reflect.Array:
		d.compareSliceFields(path, desired, current, changes)
	case reflect.Map:
		d.compareMapFields(path, desired, current, changes)
	case reflect.Ptr:
		if desired.IsNil() && current.IsNil() {
			return
		}
		if desired.IsNil() || current.IsNil() {
			*changes = append(*changes, FieldChange{
				Path:     path,
				OldValue: getReflectValue(current),
				NewValue: getReflectValue(desired),
				Type:     ChangeTypeModify,
			})
			return
		}
		d.compareFields(path, desired.Elem(), current.Elem(), changes)
	default:
		// Compare primitive values
		if !reflect.DeepEqual(desired.Interface(), current.Interface()) {
			*changes = append(*changes, FieldChange{
				Path:     path,
				OldValue: getReflectValue(current),
				NewValue: getReflectValue(desired),
				Type:     ChangeTypeModify,
			})
		}
	}
}

// compareStructFields compares struct fields
func (d *DiffEngine) compareStructFields(path string, desired, current reflect.Value, changes *[]FieldChange) {
	for i := 0; i < desired.NumField(); i++ {
		field := desired.Type().Field(i)
		if !field.IsExported() {
			continue
		}

		fieldPath := field.Name
		if path != "" {
			fieldPath = path + "." + field.Name
		}

		desiredField := desired.Field(i)
		currentField := current.Field(i)

		d.compareFields(fieldPath, desiredField, currentField, changes)
	}
}

// compareSliceFields compares slice fields
func (d *DiffEngine) compareSliceFields(path string, desired, current reflect.Value, changes *[]FieldChange) {
	if d.IgnoreOrderInSlices {
		// For semantic comparison, sort slices if they contain comparable elements
		if d.isComparableSlice(desired) {
			desired = d.sortSlice(desired)
			current = d.sortSlice(current)
		}
	}

	maxLen := desired.Len()
	if current.Len() > maxLen {
		maxLen = current.Len()
	}

	for i := 0; i < maxLen; i++ {
		indexPath := fmt.Sprintf("%s[%d]", path, i)

		var desiredElem, currentElem reflect.Value
		if i < desired.Len() {
			desiredElem = desired.Index(i)
		}
		if i < current.Len() {
			currentElem = current.Index(i)
		}

		d.compareFields(indexPath, desiredElem, currentElem, changes)
	}
}

// compareMapFields compares map fields
func (d *DiffEngine) compareMapFields(path string, desired, current reflect.Value, changes *[]FieldChange) {
	// Get all keys from both maps
	allKeys := make(map[interface{}]bool)
	for _, key := range desired.MapKeys() {
		allKeys[key.Interface()] = true
	}
	for _, key := range current.MapKeys() {
		allKeys[key.Interface()] = true
	}

	for keyInterface := range allKeys {
		key := reflect.ValueOf(keyInterface)
		keyPath := fmt.Sprintf("%s[%v]", path, keyInterface)

		desiredValue := desired.MapIndex(key)
		currentValue := current.MapIndex(key)

		d.compareFields(keyPath, desiredValue, currentValue, changes)
	}
}

// Helper functions

func extractProjectID(desired, current *ProjectState) string {
	if desired != nil && desired.Project != nil && desired.Project.Metadata.Labels != nil {
		if projectID, ok := desired.Project.Metadata.Labels["atlas.mongodb.com/project-id"]; ok {
			return projectID
		}
	}
	if current != nil && current.Project != nil && current.Project.Metadata.Labels != nil {
		if projectID, ok := current.Project.Metadata.Labels["atlas.mongodb.com/project-id"]; ok {
			return projectID
		}
	}
	return "unknown"
}

func getResourceName(desired, current interface{}) string {
	if desired != nil {
		switch v := desired.(type) {
		case *types.ClusterManifest:
			if v != nil {
				return v.Metadata.Name
			}
		case *types.DatabaseUserManifest:
			if v != nil {
				return v.Metadata.Name
			}
		case *types.NetworkAccessManifest:
			if v != nil {
				return v.Metadata.Name
			}
		case *types.ProjectManifest:
			if v != nil {
				return v.Metadata.Name
			}
		}
	}
	if current != nil {
		switch v := current.(type) {
		case *types.ClusterManifest:
			if v != nil {
				return v.Metadata.Name
			}
		case *types.DatabaseUserManifest:
			if v != nil {
				return v.Metadata.Name
			}
		case *types.NetworkAccessManifest:
			if v != nil {
				return v.Metadata.Name
			}
		case *types.ProjectManifest:
			if v != nil {
				return v.Metadata.Name
			}
		}
	}
	return "unknown"
}

func getReflectValue(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}
	return v.Interface()
}

func (d *DiffEngine) isComparableSlice(v reflect.Value) bool {
	if v.Len() == 0 {
		return false
	}

	elemType := v.Type().Elem()
	switch elemType.Kind() {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func (d *DiffEngine) sortSlice(v reflect.Value) reflect.Value {
	if !d.isComparableSlice(v) {
		return v
	}

	// Create a copy of the slice
	copied := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
	reflect.Copy(copied, v)

	// Convert to interface slice for sorting
	interfaces := make([]interface{}, copied.Len())
	for i := 0; i < copied.Len(); i++ {
		interfaces[i] = copied.Index(i).Interface()
	}

	// Sort using string representation
	sort.Slice(interfaces, func(i, j int) bool {
		return fmt.Sprintf("%v", interfaces[i]) < fmt.Sprintf("%v", interfaces[j])
	})

	// Copy back to reflected slice
	for i, iface := range interfaces {
		copied.Index(i).Set(reflect.ValueOf(iface))
	}

	return copied
}

// computeSummary computes the diff summary
func (d *DiffEngine) computeSummary(operations []Operation) DiffSummary {
	summary := DiffSummary{}

	summary.TotalOperations = len(operations)
	highestRisk := RiskLevelLow

	for _, op := range operations {
		switch op.Type {
		case OperationCreate:
			summary.CreateOperations++
		case OperationUpdate:
			summary.UpdateOperations++
		case OperationDelete:
			summary.DeleteOperations++
		case OperationNoChange:
			summary.NoChangeOperations++
		}

		if op.Impact != nil {
			if op.Impact.IsDestructive {
				summary.DestructiveOperations++
			}
			summary.EstimatedDuration += op.Impact.EstimatedDuration

			// Track highest risk level
			if d.riskLevelValue(op.Impact.RiskLevel) > d.riskLevelValue(highestRisk) {
				highestRisk = op.Impact.RiskLevel
			}
		}
	}

	summary.HighestRiskLevel = highestRisk
	return summary
}

// computeOperationImpact assesses the impact of an operation
func (d *DiffEngine) computeOperationImpact(op *Operation) *OperationImpact {
	impact := &OperationImpact{
		IsDestructive:     false,
		RequiresDowntime:  false,
		EstimatedDuration: time.Second * 30, // Default 30 seconds
		RiskLevel:         RiskLevelLow,
		Warnings:          []string{},
	}

	// Assess impact based on operation type and resource type
	switch op.Type {
	case OperationCreate:
		d.assessCreateImpact(op, impact)
	case OperationUpdate:
		d.assessUpdateImpact(op, impact)
	case OperationDelete:
		d.assessDeleteImpact(op, impact)
	case OperationNoChange:
		impact.EstimatedDuration = 0
		impact.RiskLevel = RiskLevelLow
	}

	return impact
}

// assessCreateImpact assesses the impact of create operations
func (d *DiffEngine) assessCreateImpact(op *Operation, impact *OperationImpact) {
	switch op.ResourceType {
	case types.KindProject:
		impact.EstimatedDuration = time.Minute * 2
		impact.RiskLevel = RiskLevelLow

	case types.KindCluster:
		impact.EstimatedDuration = time.Minute * 15 // Clusters take time to provision
		impact.RiskLevel = RiskLevelMedium
		impact.Warnings = append(impact.Warnings, "Cluster creation will incur costs")

	case types.KindDatabaseUser:
		impact.EstimatedDuration = time.Second * 30
		impact.RiskLevel = RiskLevelLow

	case types.KindNetworkAccess:
		impact.EstimatedDuration = time.Second * 10
		impact.RiskLevel = RiskLevelLow
	}
}

// assessUpdateImpact assesses the impact of update operations
func (d *DiffEngine) assessUpdateImpact(op *Operation, impact *OperationImpact) {
	switch op.ResourceType {
	case types.KindProject:
		impact.EstimatedDuration = time.Minute * 1
		impact.RiskLevel = RiskLevelLow

	case types.KindCluster:
		// Analyze specific field changes for clusters
		d.assessClusterUpdateImpact(op, impact)

	case types.KindDatabaseUser:
		// Check if password or roles are changing
		d.assessUserUpdateImpact(op, impact)

	case types.KindNetworkAccess:
		impact.EstimatedDuration = time.Second * 15
		impact.RiskLevel = RiskLevelLow
	}
}

// assessDeleteImpact assesses the impact of delete operations
func (d *DiffEngine) assessDeleteImpact(op *Operation, impact *OperationImpact) {
	switch op.ResourceType {
	case types.KindProject:
		impact.IsDestructive = true
		impact.RequiresDowntime = true
		impact.EstimatedDuration = time.Minute * 5
		impact.RiskLevel = RiskLevelCritical
		impact.Warnings = append(impact.Warnings, "Project deletion will permanently remove all resources")

	case types.KindCluster:
		impact.IsDestructive = true
		impact.RequiresDowntime = true
		impact.EstimatedDuration = time.Minute * 10
		impact.RiskLevel = RiskLevelHigh
		impact.Warnings = append(impact.Warnings, "Cluster deletion will permanently destroy all data")

	case types.KindDatabaseUser:
		impact.IsDestructive = true
		impact.EstimatedDuration = time.Second * 30
		impact.RiskLevel = RiskLevelMedium
		impact.Warnings = append(impact.Warnings, "User deletion will revoke all database access")

	case types.KindNetworkAccess:
		impact.IsDestructive = true
		impact.EstimatedDuration = time.Second * 10
		impact.RiskLevel = RiskLevelMedium
		impact.Warnings = append(impact.Warnings, "Network access removal may block connections")
	}
}

// assessClusterUpdateImpact assesses specific cluster update impacts
func (d *DiffEngine) assessClusterUpdateImpact(op *Operation, impact *OperationImpact) {
	impact.EstimatedDuration = time.Minute * 5 // Default for cluster updates
	impact.RiskLevel = RiskLevelMedium

	// Analyze field changes to determine specific impact
	for _, change := range op.FieldChanges {
		switch {
		case strings.Contains(change.Path, "InstanceSize"):
			impact.RequiresDowntime = true
			impact.EstimatedDuration = time.Minute * 20
			impact.RiskLevel = RiskLevelHigh
			impact.Warnings = append(impact.Warnings, "Instance size changes require cluster restart")

		case strings.Contains(change.Path, "MongoDBVersion"):
			impact.RequiresDowntime = true
			impact.EstimatedDuration = time.Minute * 30
			impact.RiskLevel = RiskLevelHigh
			impact.Warnings = append(impact.Warnings, "MongoDB version upgrades cannot be reversed")

		case strings.Contains(change.Path, "DiskSizeGB"):
			if d.isDiskSizeDecrease(change) {
				impact.IsDestructive = true
				impact.RiskLevel = RiskLevelCritical
				impact.Warnings = append(impact.Warnings, "Disk size cannot be decreased")
			} else {
				impact.EstimatedDuration = time.Minute * 10
				impact.RiskLevel = RiskLevelMedium
			}

		case strings.Contains(change.Path, "BackupEnabled"):
			if d.isBackupDisabled(change) {
				impact.RiskLevel = RiskLevelHigh
				impact.Warnings = append(impact.Warnings, "Disabling backups increases data loss risk")
			}

		case strings.Contains(change.Path, "ReplicationSpecs"):
			impact.RequiresDowntime = true
			impact.EstimatedDuration = time.Minute * 15
			impact.RiskLevel = RiskLevelHigh
			impact.Warnings = append(impact.Warnings, "Replication changes may affect availability")
		}
	}
}

// assessUserUpdateImpact assesses specific database user update impacts
func (d *DiffEngine) assessUserUpdateImpact(op *Operation, impact *OperationImpact) {
	impact.EstimatedDuration = time.Second * 30
	impact.RiskLevel = RiskLevelLow

	for _, change := range op.FieldChanges {
		switch {
		case strings.Contains(change.Path, "Password"):
			impact.RiskLevel = RiskLevelMedium
			impact.Warnings = append(impact.Warnings, "Password changes will require application reconnection")

		case strings.Contains(change.Path, "Roles"):
			impact.RiskLevel = RiskLevelMedium
			impact.Warnings = append(impact.Warnings, "Role changes may affect application permissions")

		case strings.Contains(change.Path, "Scopes"):
			impact.RiskLevel = RiskLevelMedium
			impact.Warnings = append(impact.Warnings, "Scope changes may restrict database access")
		}
	}
}

// Helper methods for impact assessment

func (d *DiffEngine) isDiskSizeDecrease(change FieldChange) bool {
	if change.OldValue == nil || change.NewValue == nil {
		return false
	}

	oldSize, ok1 := change.OldValue.(float64)
	newSize, ok2 := change.NewValue.(float64)

	return ok1 && ok2 && newSize < oldSize
}

func (d *DiffEngine) isBackupDisabled(change FieldChange) bool {
	if change.OldValue == nil || change.NewValue == nil {
		return false
	}

	oldEnabled, ok1 := change.OldValue.(bool)
	newEnabled, ok2 := change.NewValue.(bool)

	return ok1 && ok2 && oldEnabled && !newEnabled
}

// riskLevelValue returns numeric value for risk level comparison
func (d *DiffEngine) riskLevelValue(level RiskLevel) int {
	switch level {
	case RiskLevelLow:
		return 1
	case RiskLevelMedium:
		return 2
	case RiskLevelHigh:
		return 3
	case RiskLevelCritical:
		return 4
	default:
		return 0
	}
}
