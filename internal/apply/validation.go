package apply

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/validation"
)

// ValidatorOptions provides configuration for the validation process
type ValidatorOptions struct {
	StrictMode      bool               // Fail on warnings
	AllowedVersions []types.APIVersion // Restrict to specific API versions
	MaxNameLength   int                // Maximum length for resource names
	SkipQuotaCheck  bool               // Skip Atlas quota validation
}

// DefaultValidatorOptions returns sensible defaults for validation
func DefaultValidatorOptions() *ValidatorOptions {
	return &ValidatorOptions{
		StrictMode: false,
		AllowedVersions: []types.APIVersion{
			types.APIVersionV1Alpha1,
			types.APIVersionV1Beta1,
			types.APIVersionV1,
		},
		MaxNameLength:  64,
		SkipQuotaCheck: false,
	}
}

// ValidationResult contains the results of configuration validation
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// ValidationError represents a validation error or warning
type ValidationError struct {
	Path     string `json:"path"`     // JSON path to the problematic field
	Field    string `json:"field"`    // Field name
	Value    string `json:"value"`    // Current value
	Message  string `json:"message"`  // Error message
	Code     string `json:"code"`     // Error code for programmatic handling
	Severity string `json:"severity"` // "error" or "warning"
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Path, ve.Message)
}

// ValidateApplyConfig validates a complete apply configuration
func ValidateApplyConfig(config *types.ApplyConfig, opts *ValidatorOptions) *ValidationResult {
	if opts == nil {
		opts = DefaultValidatorOptions()
	}

	result := &ValidationResult{Valid: true}

	// Validate basic structure
	validateBasicStructure(config, result, opts)

	// Validate project configuration
	if config.Spec.Name != "" {
		validateProjectConfig(&config.Spec, result, opts)
	}

	// Cross-field validation
	validateCrossFieldRules(config, result, opts)

	// Enhanced cross-resource dependency validation
	if !opts.SkipQuotaCheck {
		validateCrossResourceDependencies(config, result, opts)
	}

	// Check for warnings that should be treated as errors in strict mode
	if opts.StrictMode && len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			result.Errors = append(result.Errors, ValidationError{
				Path:     warning.Path,
				Field:    warning.Field,
				Value:    warning.Value,
				Message:  warning.Message + " (strict mode)",
				Code:     warning.Code,
				Severity: "error",
			})
		}
		result.Warnings = nil
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// ValidateApplyDocument validates a multi-resource apply document
func ValidateApplyDocument(doc *types.ApplyDocument, opts *ValidatorOptions) *ValidationResult {
	if opts == nil {
		opts = DefaultValidatorOptions()
	}

	result := &ValidationResult{Valid: true}

	// Validate document structure
	validateDocumentStructure(doc, result, opts)

	// Validate each resource manifest
	for i, resource := range doc.Resources {
		path := fmt.Sprintf("resources[%d]", i)
		validateResourceManifest(&resource, path, result, opts)
	}

	// Validate resource dependencies
	validateResourceDependencies(doc, result, opts)

	// Enhanced cross-document validation
	validateCrossDocumentDependencies(doc, result, opts)

	result.Valid = len(result.Errors) == 0
	return result
}

// ValidateResourceUniqueness checks that resource names are unique within the configuration
func ValidateResourceUniqueness(config *types.ApplyConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}
	seen := make(map[string]string) // resourceKey -> path

	// Check clusters
	for i, cluster := range config.Spec.Clusters {
		key := fmt.Sprintf("cluster:%s", cluster.Metadata.Name)
		path := fmt.Sprintf("spec.clusters[%d].metadata.name", i)
		if existingPath, exists := seen[key]; exists {
			addError(result, path, "metadata.name", cluster.Metadata.Name,
				fmt.Sprintf("duplicate cluster name (also defined at %s)", existingPath),
				"DUPLICATE_RESOURCE_NAME")
		} else {
			seen[key] = path
		}
	}

	// Check database users
	for i, user := range config.Spec.DatabaseUsers {
		key := fmt.Sprintf("user:%s", user.Metadata.Name)
		path := fmt.Sprintf("spec.databaseUsers[%d].metadata.name", i)
		if existingPath, exists := seen[key]; exists {
			addError(result, path, "metadata.name", user.Metadata.Name,
				fmt.Sprintf("duplicate database user name (also defined at %s)", existingPath),
				"DUPLICATE_RESOURCE_NAME")
		} else {
			seen[key] = path
		}
	}

	// Check network access entries
	for i, netAccess := range config.Spec.NetworkAccess {
		key := fmt.Sprintf("network:%s", netAccess.Metadata.Name)
		path := fmt.Sprintf("spec.networkAccess[%d].metadata.name", i)
		if existingPath, exists := seen[key]; exists {
			addError(result, path, "metadata.name", netAccess.Metadata.Name,
				fmt.Sprintf("duplicate network access name (also defined at %s)", existingPath),
				"DUPLICATE_RESOURCE_NAME")
		} else {
			seen[key] = path
		}
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// ValidateCircularDependencies detects circular dependencies in resource configurations
func ValidateCircularDependencies(config *types.ApplyConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Build dependency graph
	graph := types.NewDependencyGraph()

	// Add all resources to the graph
	addResourceToGraph(graph, "project", types.KindProject, "", nil)

	for _, cluster := range config.Spec.Clusters {
		addResourceToGraph(graph, cluster.Metadata.Name, types.KindCluster, "", cluster.DependsOn)
	}

	for _, user := range config.Spec.DatabaseUsers {
		addResourceToGraph(graph, user.Metadata.Name, types.KindDatabaseUser, "", user.DependsOn)
	}

	for _, netAccess := range config.Spec.NetworkAccess {
		addResourceToGraph(graph, netAccess.Metadata.Name, types.KindNetworkAccess, "", netAccess.DependsOn)
	}

	// Check for cycles
	if hasCycle, cycle := graph.HasCycles(); hasCycle {
		addError(result, "spec", "dependencies", strings.Join(cycle, " -> "),
			fmt.Sprintf("circular dependency detected: %s", strings.Join(cycle, " -> ")),
			"CIRCULAR_DEPENDENCY")
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// Helper functions

func validateBasicStructure(config *types.ApplyConfig, result *ValidationResult, opts *ValidatorOptions) {
	// Validate API version
	if err := types.ValidateAPIVersion(types.APIVersion(config.APIVersion)); err != nil {
		addError(result, "apiVersion", "apiVersion", config.APIVersion,
			err.Error(), "INVALID_API_VERSION")
	}

	// Check if API version is allowed
	if len(opts.AllowedVersions) > 0 {
		allowed := false
		for _, version := range opts.AllowedVersions {
			if types.APIVersion(config.APIVersion) == version {
				allowed = true
				break
			}
		}
		if !allowed {
			addError(result, "apiVersion", "apiVersion", config.APIVersion,
				fmt.Sprintf("API version not allowed (allowed: %v)", opts.AllowedVersions),
				"API_VERSION_NOT_ALLOWED")
		}
	}

	// Validate kind
	if err := types.ValidateResourceKind(types.ResourceKind(config.Kind)); err != nil {
		addError(result, "kind", "kind", config.Kind,
			err.Error(), "INVALID_KIND")
	}

	// Validate metadata
	validateMetadata(&config.Metadata, "metadata", result, opts)

	// Validate required fields only for Project kind
	if types.ResourceKind(config.Kind) == types.KindProject {
		if config.Spec.Name == "" {
			addError(result, "spec.name", "name", "",
				"project name is required", "REQUIRED_FIELD_MISSING")
		}

		if config.Spec.OrganizationID == "" {
			addError(result, "spec.organizationId", "organizationId", "",
				"organization ID is required", "REQUIRED_FIELD_MISSING")
		}
	}
}

func validateProjectConfig(project *types.ProjectConfig, result *ValidationResult, opts *ValidatorOptions) {
	// Validate project name
	if err := validation.ValidateRequired(project.Name, "project name"); err != nil {
		addError(result, "spec.name", "name", project.Name, err.Error(), "INVALID_PROJECT_NAME")
	}

	if len(project.Name) > opts.MaxNameLength {
		addError(result, "spec.name", "name", project.Name,
			fmt.Sprintf("project name exceeds maximum length of %d characters", opts.MaxNameLength),
			"NAME_TOO_LONG")
	}

	// Validate organization ID
	if err := validation.ValidateOrganizationID(project.OrganizationID); err != nil {
		addError(result, "spec.organizationId", "organizationId", project.OrganizationID,
			err.Error(), "INVALID_ORGANIZATION_ID")
	}

	// Validate clusters
	for i, cluster := range project.Clusters {
		path := fmt.Sprintf("spec.clusters[%d]", i)
		validateClusterConfig(&cluster, path, result, opts)
	}

	// Validate database users
	for i, user := range project.DatabaseUsers {
		path := fmt.Sprintf("spec.databaseUsers[%d]", i)
		validateDatabaseUserConfig(&user, path, result, opts)
	}

	// Validate network access
	for i, netAccess := range project.NetworkAccess {
		path := fmt.Sprintf("spec.networkAccess[%d]", i)
		validateNetworkAccessConfig(&netAccess, path, result, opts)
	}
}

func validateClusterConfig(cluster *types.ClusterConfig, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Validate cluster name
	if err := validation.ValidateClusterName(cluster.Metadata.Name); err != nil {
		addError(result, basePath+".metadata.name", "name", cluster.Metadata.Name,
			err.Error(), "INVALID_CLUSTER_NAME")
	}

	// Validate provider
	if cluster.Provider == "" {
		addError(result, basePath+".provider", "provider", "",
			"provider is required", "REQUIRED_FIELD_MISSING")
	} else {
		validateProvider(cluster.Provider, basePath+".provider", result)
	}

	// Validate region
	if cluster.Region == "" {
		addError(result, basePath+".region", "region", "",
			"region is required", "REQUIRED_FIELD_MISSING")
	}

	// Validate instance size
	if cluster.InstanceSize == "" {
		addError(result, basePath+".instanceSize", "instanceSize", "",
			"instance size is required", "REQUIRED_FIELD_MISSING")
	} else {
		validateInstanceSize(cluster.InstanceSize, basePath+".instanceSize", result)
	}

	// Validate tier compatibility
	if cluster.TierType != "" && cluster.InstanceSize != "" {
		validateTierInstanceCompatibility(cluster.TierType, cluster.InstanceSize, basePath, result)
	}

	// Validate MongoDB version
	if cluster.MongoDBVersion != "" {
		validateMongoDBVersion(cluster.MongoDBVersion, basePath+".mongodbVersion", result)
	}

	// Validate autoscaling
	if cluster.AutoScaling != nil {
		validateAutoScalingConfig(cluster.AutoScaling, basePath+".autoScaling", result)
	}

	// Validate replication specs
	for i, spec := range cluster.ReplicationSpecs {
		path := fmt.Sprintf("%s.replicationSpecs[%d]", basePath, i)
		validateReplicationSpec(&spec, path, result)
	}
}

func validateDatabaseUserConfig(user *types.DatabaseUserConfig, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Validate username
	if err := validation.ValidateUsername(user.Username); err != nil {
		addError(result, basePath+".username", "username", user.Username,
			err.Error(), "INVALID_USERNAME")
	}

	// Validate roles
	if len(user.Roles) == 0 {
		addError(result, basePath+".roles", "roles", "",
			"at least one role is required", "REQUIRED_FIELD_MISSING")
	}

	for i, role := range user.Roles {
		path := fmt.Sprintf("%s.roles[%d]", basePath, i)
		validateDatabaseRole(&role, path, result)
	}

	// Validate auth database
	if user.AuthDatabase != "" {
		validateDatabaseName(user.AuthDatabase, basePath+".authDatabase", result)
	}

	// Validate scopes
	for i, scope := range user.Scopes {
		path := fmt.Sprintf("%s.scopes[%d]", basePath, i)
		validateUserScope(&scope, path, result)
	}
}

func validateNetworkAccessConfig(netAccess *types.NetworkAccessConfig, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Must have either IP address or CIDR
	if netAccess.IPAddress == "" && netAccess.CIDR == "" && netAccess.AWSSecurityGroup == "" {
		addError(result, basePath, "access", "",
			"must specify one of: ipAddress, cidr, or awsSecurityGroup", "REQUIRED_FIELD_MISSING")
		return
	}

	// Validate IP address
	if netAccess.IPAddress != "" {
		if ip := net.ParseIP(netAccess.IPAddress); ip == nil {
			addError(result, basePath+".ipAddress", "ipAddress", netAccess.IPAddress,
				"invalid IP address", "INVALID_IP_ADDRESS")
		}
	}

	// Validate CIDR
	if netAccess.CIDR != "" {
		if _, _, err := net.ParseCIDR(netAccess.CIDR); err != nil {
			addError(result, basePath+".cidr", "cidr", netAccess.CIDR,
				"invalid CIDR notation", "INVALID_CIDR")
		}
	}

	// Validate delete after date
	if netAccess.DeleteAfterDate != "" {
		if _, err := time.Parse(time.RFC3339, netAccess.DeleteAfterDate); err != nil {
			addError(result, basePath+".deleteAfterDate", "deleteAfterDate", netAccess.DeleteAfterDate,
				"invalid date format (use RFC3339)", "INVALID_DATE_FORMAT")
		}
	}
}

func validateMetadata(metadata *types.MetadataConfig, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Validate name
	if metadata.Name == "" {
		addError(result, basePath+".name", "name", "",
			"name is required", "REQUIRED_FIELD_MISSING")
	} else {
		validateResourceName(metadata.Name, basePath+".name", result, opts)
	}

	// Validate labels
	for key, value := range metadata.Labels {
		validateLabelKey(key, basePath+".labels", result)
		validateLabelValue(value, basePath+".labels", result)
	}

	// Validate annotations
	for key, value := range metadata.Annotations {
		validateAnnotationKey(key, basePath+".annotations", result)
		validateAnnotationValue(value, basePath+".annotations", result)
	}
}

func validateDocumentStructure(doc *types.ApplyDocument, result *ValidationResult, opts *ValidatorOptions) {
	// Validate API version
	if err := types.ValidateAPIVersion(doc.APIVersion); err != nil {
		addError(result, "apiVersion", "apiVersion", string(doc.APIVersion),
			err.Error(), "INVALID_API_VERSION")
	}

	// Validate kind
	if doc.Kind != types.KindApplyDocument {
		addError(result, "kind", "kind", string(doc.Kind),
			"kind must be 'ApplyDocument' for multi-resource documents", "INVALID_KIND")
	}

	// Validate metadata
	validateMetadata(&doc.Metadata, "metadata", result, opts)

	// Validate resources
	if len(doc.Resources) == 0 {
		addError(result, "resources", "resources", "",
			"at least one resource is required", "REQUIRED_FIELD_MISSING")
	}
}

func validateResourceManifest(manifest *types.ResourceManifest, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Validate API version
	if err := types.ValidateAPIVersion(manifest.APIVersion); err != nil {
		addError(result, basePath+".apiVersion", "apiVersion", string(manifest.APIVersion),
			err.Error(), "INVALID_API_VERSION")
	}

	// Validate kind
	if err := types.ValidateResourceKind(manifest.Kind); err != nil {
		addError(result, basePath+".kind", "kind", string(manifest.Kind),
			err.Error(), "INVALID_KIND")
	}

	// Validate metadata (pass resource kind for context-aware validation)
	validateResourceMetadataWithKind(&manifest.Metadata, manifest.Kind, basePath+".metadata", result, opts)

	// Validate resource-specific content based on kind
	validateResourceContent(manifest, basePath, result, opts)
}

// validateResourceContent validates the specific content of a resource based on its kind
func validateResourceContent(manifest *types.ResourceManifest, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	switch manifest.Kind {
	case types.KindDatabaseUser:
		validateDatabaseUserManifest(manifest, basePath, result, opts)
	case types.KindDatabaseRole:
		validateDatabaseRoleManifest(manifest, basePath, result, opts)
	case types.KindCluster:
		validateClusterManifest(manifest, basePath, result, opts)
	case types.KindNetworkAccess:
		validateNetworkAccessManifest(manifest, basePath, result, opts)
	case types.KindProject:
		validateProjectManifest(manifest, basePath, result, opts)
	case types.KindSearchIndex:
		validateSearchIndexManifest(manifest, basePath, result, opts)
	case types.KindVPCEndpoint:
		validateVPCEndpointManifest(manifest, basePath, result, opts)
	default:
		// For unknown resource types, log a warning but don't fail validation
		addWarning(result, basePath+".kind", "kind", string(manifest.Kind),
			"unknown resource kind - skipping specific validation", "UNKNOWN_RESOURCE_KIND")
	}
}

// validateDatabaseUserManifest validates a DatabaseUser resource manifest
func validateDatabaseUserManifest(manifest *types.ResourceManifest, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Try to convert the spec to DatabaseUserSpec
	var userSpec types.DatabaseUserSpec

	// Handle both map[string]interface{} (from YAML) and DatabaseUserSpec (from typed structs)
	switch spec := manifest.Spec.(type) {
	case types.DatabaseUserSpec:
		userSpec = spec
	case map[string]interface{}:
		// Convert from map to struct using JSON marshaling
		if err := convertMapToStruct(spec, &userSpec); err != nil {
			addError(result, basePath+".spec", "spec", "",
				fmt.Sprintf("invalid DatabaseUser spec format: %v", err), "INVALID_SPEC_FORMAT")
			return
		}
	default:
		addError(result, basePath+".spec", "spec", "",
			"DatabaseUser spec must be a valid structure", "INVALID_SPEC_TYPE")
		return
	}

	// Convert DatabaseUserSpec to DatabaseUserConfig for validation
	userConfig := types.DatabaseUserConfig{
		Metadata:     manifest.Metadata,
		Username:     userSpec.Username,
		Password:     userSpec.Password,
		Roles:        userSpec.Roles,
		AuthDatabase: userSpec.AuthDatabase,
		Scopes:       userSpec.Scopes,
	}

	// Use the existing validation function
	validateDatabaseUserConfig(&userConfig, basePath+".spec", result, opts)
}

// validateDatabaseRoleManifest validates a DatabaseRole resource manifest
func validateDatabaseRoleManifest(manifest *types.ResourceManifest, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Try to convert the spec to DatabaseRoleSpec
	var roleSpec types.DatabaseRoleSpec

	// Handle both map[string]interface{} (from YAML) and DatabaseRoleSpec (from typed structs)
	switch spec := manifest.Spec.(type) {
	case types.DatabaseRoleSpec:
		roleSpec = spec
	case map[string]interface{}:
		// Convert from map to struct using JSON marshaling
		if err := convertMapToStruct(spec, &roleSpec); err != nil {
			addError(result, basePath+".spec", "spec", "",
				fmt.Sprintf("invalid DatabaseRole spec format: %v", err), "INVALID_SPEC_FORMAT")
			return
		}
	default:
		addError(result, basePath+".spec", "spec", "",
			"DatabaseRole spec must be a valid structure", "INVALID_SPEC_TYPE")
		return
	}

	// Convert DatabaseRoleSpec to CustomDatabaseRoleConfig for validation
	roleConfig := types.CustomDatabaseRoleConfig{
		Metadata:       manifest.Metadata,
		RoleName:       roleSpec.RoleName,
		DatabaseName:   roleSpec.DatabaseName,
		Privileges:     roleSpec.Privileges,
		InheritedRoles: roleSpec.InheritedRoles,
	}

	// Use the existing validation function
	validateCustomDatabaseRoleConfig(&roleConfig, basePath+".spec", result, opts)
}

// validateClusterManifest validates a Cluster resource manifest
func validateClusterManifest(manifest *types.ResourceManifest, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Try to convert the spec to ClusterSpec
	var clusterSpec types.ClusterSpec

	switch spec := manifest.Spec.(type) {
	case types.ClusterSpec:
		clusterSpec = spec
	case map[string]interface{}:
		if err := convertMapToStruct(spec, &clusterSpec); err != nil {
			addError(result, basePath+".spec", "spec", "",
				fmt.Sprintf("invalid Cluster spec format: %v", err), "INVALID_SPEC_FORMAT")
			return
		}
	default:
		addError(result, basePath+".spec", "spec", "",
			"Cluster spec must be a valid structure", "INVALID_SPEC_TYPE")
		return
	}

	// Convert ClusterSpec to ClusterConfig for validation
	clusterConfig := types.ClusterConfig{
		Metadata:         manifest.Metadata,
		Provider:         clusterSpec.Provider,
		Region:           clusterSpec.Region,
		InstanceSize:     clusterSpec.InstanceSize,
		DiskSizeGB:       clusterSpec.DiskSizeGB,
		BackupEnabled:    clusterSpec.BackupEnabled,
		TierType:         clusterSpec.TierType,
		MongoDBVersion:   clusterSpec.MongoDBVersion,
		ClusterType:      clusterSpec.ClusterType,
		ReplicationSpecs: clusterSpec.ReplicationSpecs,
		AutoScaling:      clusterSpec.AutoScaling,
		Encryption:       clusterSpec.Encryption,
		BiConnector:      clusterSpec.BiConnector,
	}

	// Use the existing validation function
	validateClusterConfig(&clusterConfig, basePath+".spec", result, opts)
}

// validateNetworkAccessManifest validates a NetworkAccess resource manifest
func validateNetworkAccessManifest(manifest *types.ResourceManifest, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Try to convert the spec to NetworkAccessSpec
	var netSpec types.NetworkAccessSpec

	switch spec := manifest.Spec.(type) {
	case types.NetworkAccessSpec:
		netSpec = spec
	case map[string]interface{}:
		if err := convertMapToStruct(spec, &netSpec); err != nil {
			addError(result, basePath+".spec", "spec", "",
				fmt.Sprintf("invalid NetworkAccess spec format: %v", err), "INVALID_SPEC_FORMAT")
			return
		}
	default:
		addError(result, basePath+".spec", "spec", "",
			"NetworkAccess spec must be a valid structure", "INVALID_SPEC_TYPE")
		return
	}

	// Convert NetworkAccessSpec to NetworkAccessConfig for validation
	netConfig := types.NetworkAccessConfig{
		Metadata:         manifest.Metadata,
		IPAddress:        netSpec.IPAddress,
		CIDR:             netSpec.CIDR,
		AWSSecurityGroup: netSpec.AWSSecurityGroup,
		Comment:          netSpec.Comment,
		DeleteAfterDate:  netSpec.DeleteAfterDate,
	}

	// Use the existing validation function
	validateNetworkAccessConfig(&netConfig, basePath+".spec", result, opts)
}

// validateProjectManifest validates a Project resource manifest
func validateProjectManifest(manifest *types.ResourceManifest, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Project manifests in ApplyDocument should be rare, but handle them
	addWarning(result, basePath, "kind", "Project",
		"Project resources in ApplyDocument are unusual - consider using Project kind directly", "UNUSUAL_RESOURCE_PLACEMENT")
}

func validateResourceDependencies(doc *types.ApplyDocument, result *ValidationResult, opts *ValidatorOptions) {
	// Build a map of available resources
	availableResources := make(map[string]bool)
	for _, resource := range doc.Resources {
		key := fmt.Sprintf("%s:%s", resource.Kind, resource.Metadata.Name)
		availableResources[key] = true
	}

	// Check that all dependencies exist
	for i, resource := range doc.Resources {
		// Extract dependencies based on resource type
		var dependencies []string
		switch resource.Kind {
		case types.KindCluster:
			if spec, ok := resource.Spec.(types.ClusterSpec); ok {
				dependencies = append(dependencies, spec.ProjectName)
			}
		case types.KindDatabaseUser:
			if spec, ok := resource.Spec.(types.DatabaseUserSpec); ok {
				dependencies = append(dependencies, spec.ProjectName)
			}
		case types.KindNetworkAccess:
			if spec, ok := resource.Spec.(types.NetworkAccessSpec); ok {
				dependencies = append(dependencies, spec.ProjectName)
			}
		}

		// Check each dependency
		for _, dep := range dependencies {
			if !availableResources[dep] && dep != "" {
				path := fmt.Sprintf("resources[%d].spec", i)
				addError(result, path, "dependencies", dep,
					fmt.Sprintf("dependency '%s' not found in document", dep),
					"DEPENDENCY_NOT_FOUND")
			}
		}
	}
}

func validateCrossFieldRules(config *types.ApplyConfig, result *ValidationResult, opts *ValidatorOptions) {
	// Example cross-field validation rules

	// Rule: Clusters with backup enabled should have appropriate tier
	for i, cluster := range config.Spec.Clusters {
		if cluster.BackupEnabled != nil && *cluster.BackupEnabled {
			if cluster.TierType == "FREE" {
				path := fmt.Sprintf("spec.clusters[%d]", i)
				addWarning(result, path, "backupEnabled", "true",
					"backup is not available for free tier clusters", "BACKUP_NOT_AVAILABLE_FREE_TIER")
			}
		}
	}

	// Rule: Database users with admin roles should have auth database set to admin
	for i, user := range config.Spec.DatabaseUsers {
		hasAdminRole := false
		for _, role := range user.Roles {
			if strings.Contains(strings.ToLower(role.RoleName), "admin") {
				hasAdminRole = true
				break
			}
		}
		if hasAdminRole && user.AuthDatabase != "admin" {
			path := fmt.Sprintf("spec.databaseUsers[%d].authDatabase", i)
			addWarning(result, path, "authDatabase", user.AuthDatabase,
				"users with admin roles should typically use 'admin' auth database", "ADMIN_ROLE_AUTH_DATABASE")
		}
	}
}

// Validation helper functions

func validateProvider(provider, path string, result *ValidationResult) {
	validProviders := []string{"AWS", "GCP", "AZURE", "TENANT"}
	for _, valid := range validProviders {
		if provider == valid {
			return
		}
	}
	addError(result, path, "provider", provider,
		fmt.Sprintf("invalid provider (valid: %v)", validProviders), "INVALID_PROVIDER")
}

func validateInstanceSize(size, path string, result *ValidationResult) {
	if strings.TrimSpace(size) == "" {
		addError(result, path, "instanceSize", size,
			"instance size cannot be empty", "INVALID_INSTANCE_SIZE")
		return
	}

	// Valid Atlas instance sizes
	validSizes := map[string]bool{
		"M0": true, "M2": true, "M5": true, "M10": true, "M20": true, "M30": true,
		"M40": true, "M50": true, "M60": true, "M80": true, "M140": true,
		"M200": true, "M300": true, "M400": true, "M700": true,
		"R40": true, "R50": true, "R60": true, "R80": true, "R200": true,
		"R300": true, "R400": true, "R700": true,
	}

	if !validSizes[size] {
		addError(result, path, "instanceSize", size,
			"invalid Atlas instance size (e.g., M0, M10, M30, R40)", "INVALID_INSTANCE_SIZE")
	}
}

func validateTierInstanceCompatibility(tierType, instanceSize, basePath string, result *ValidationResult) {
	if tierType == "FREE" && instanceSize != "M0" {
		addError(result, basePath, "tierType/instanceSize", tierType+"/"+instanceSize,
			"free tier only supports M0 instance size", "TIER_INSTANCE_INCOMPATIBLE")
	}
}

func validateMongoDBVersion(version, path string, result *ValidationResult) {
	if strings.TrimSpace(version) == "" {
		addError(result, path, "mongodbVersion", version,
			"MongoDB version cannot be empty", "INVALID_MONGODB_VERSION")
		return
	}

	// Basic version format validation
	pattern := regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)
	if !pattern.MatchString(version) {
		addError(result, path, "mongodbVersion", version,
			"invalid MongoDB version format (expected X.Y or X.Y.Z)", "INVALID_MONGODB_VERSION")
		return
	}

	// Check major.minor version
	parts := strings.Split(version, ".")
	majorMinor := parts[0] + "." + parts[1]

	// For Atlas, only 4.4+ is supported
	if majorMinor == "2.0" || majorMinor == "3.6" || majorMinor == "4.0" || majorMinor == "4.2" {
		addError(result, path, "mongodbVersion", version,
			"unsupported MongoDB version for Atlas (minimum 4.4)", "INVALID_MONGODB_VERSION")
	}
}

func validateAutoScalingConfig(autoScaling *types.AutoScalingConfig, basePath string, result *ValidationResult) {
	// Validate disk autoscaling
	if autoScaling.DiskGB != nil {
		if autoScaling.DiskGB.MinimumGB != nil && autoScaling.DiskGB.MaximumGB != nil {
			if *autoScaling.DiskGB.MinimumGB >= *autoScaling.DiskGB.MaximumGB {
				addError(result, basePath+".diskGB", "minimumGB/maximumGB",
					fmt.Sprintf("%d/%d", *autoScaling.DiskGB.MinimumGB, *autoScaling.DiskGB.MaximumGB),
					"minimum disk size must be less than maximum", "INVALID_DISK_RANGE")
			}
		}
	}

	// Validate compute autoscaling
	if autoScaling.Compute != nil {
		// Could add validation for min/max instance size compatibility
		if autoScaling.Compute.MinInstanceSize != "" {
			validateInstanceSize(autoScaling.Compute.MinInstanceSize, basePath+".compute.minInstanceSize", result)
		}
		if autoScaling.Compute.MaxInstanceSize != "" {
			validateInstanceSize(autoScaling.Compute.MaxInstanceSize, basePath+".compute.maxInstanceSize", result)
		}
	}
}

func validateReplicationSpec(spec *types.ReplicationSpec, basePath string, result *ValidationResult) {
	// Validate number of shards
	if spec.NumShards != nil && *spec.NumShards < 1 {
		addError(result, basePath+".numShards", "numShards", fmt.Sprintf("%d", *spec.NumShards),
			"number of shards must be at least 1", "INVALID_SHARD_COUNT")
	}

	// Validate region configs
	for i, regionConfig := range spec.RegionConfigs {
		path := fmt.Sprintf("%s.regionConfigs[%d]", basePath, i)
		validateRegionConfig(&regionConfig, path, result)
	}
}

func validateRegionConfig(config *types.RegionConfig, basePath string, result *ValidationResult) {
	// Validate required fields
	if config.RegionName == "" {
		addError(result, basePath+".regionName", "regionName", "",
			"region name is required", "REQUIRED_FIELD_MISSING")
	}

	if config.ProviderName == "" {
		addError(result, basePath+".providerName", "providerName", "",
			"provider name is required", "REQUIRED_FIELD_MISSING")
	}

	// Validate priority
	if config.Priority != nil && (*config.Priority < 0 || *config.Priority > 7) {
		addError(result, basePath+".priority", "priority", fmt.Sprintf("%d", *config.Priority),
			"priority must be between 0 and 7", "INVALID_PRIORITY")
	}

	// Validate node counts
	if config.ElectableNodes != nil && *config.ElectableNodes < 0 {
		addError(result, basePath+".electableNodes", "electableNodes", fmt.Sprintf("%d", *config.ElectableNodes),
			"electable nodes must be non-negative", "INVALID_NODE_COUNT")
	}
}

func validateDatabaseRole(role *types.DatabaseRoleConfig, basePath string, result *ValidationResult) {
	// Validate role name
	if role.RoleName == "" {
		addError(result, basePath+".roleName", "roleName", "",
			"role name is required", "REQUIRED_FIELD_MISSING")
	}

	// Validate database name
	if role.DatabaseName == "" {
		addError(result, basePath+".databaseName", "databaseName", "",
			"database name is required", "REQUIRED_FIELD_MISSING")
	} else {
		validateDatabaseName(role.DatabaseName, basePath+".databaseName", result)
	}

	// Validate collection name if provided
	if role.CollectionName != "" {
		validateCollectionName(role.CollectionName, basePath+".collectionName", result)
	}
}

func validateCustomDatabaseRoleConfig(role *types.CustomDatabaseRoleConfig, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Validate role name
	if role.RoleName == "" {
		addError(result, basePath+".roleName", "roleName", "",
			"role name is required", "REQUIRED_FIELD_MISSING")
	}

	// Validate database name
	if role.DatabaseName == "" {
		addError(result, basePath+".databaseName", "databaseName", "",
			"database name is required", "REQUIRED_FIELD_MISSING")
	} else {
		validateDatabaseName(role.DatabaseName, basePath+".databaseName", result)
	}

	// Validate privileges
	for i, privilege := range role.Privileges {
		path := fmt.Sprintf("%s.privileges[%d]", basePath, i)
		validateCustomRolePrivilege(&privilege, path, result)
	}

	// Validate inherited roles
	for i, inheritedRole := range role.InheritedRoles {
		path := fmt.Sprintf("%s.inheritedRoles[%d]", basePath, i)
		validateCustomRoleInheritedRole(&inheritedRole, path, result)
	}

	// At least one privilege or inherited role should be specified
	if len(role.Privileges) == 0 && len(role.InheritedRoles) == 0 {
		addWarning(result, basePath, "privileges", "",
			"custom role should have at least one privilege or inherited role", "EMPTY_ROLE_DEFINITION")
	}
}

func validateCustomRolePrivilege(privilege *types.CustomRolePrivilegeConfig, basePath string, result *ValidationResult) {
	// Validate actions
	if len(privilege.Actions) == 0 {
		addError(result, basePath+".actions", "actions", "",
			"at least one action is required", "REQUIRED_FIELD_MISSING")
	}

	for i, action := range privilege.Actions {
		if action == "" {
			addError(result, basePath+".actions", "actions", fmt.Sprintf("index %d", i),
				"action cannot be empty", "INVALID_ACTION")
		}
	}

	// Validate resource
	validateCustomRoleResource(&privilege.Resource, basePath+".resource", result)
}

func validateCustomRoleResource(resource *types.CustomRoleResourceConfig, basePath string, result *ValidationResult) {
	// Validate database name
	if resource.Database == "" {
		addError(result, basePath+".database", "database", "",
			"database name is required", "REQUIRED_FIELD_MISSING")
	} else {
		validateDatabaseName(resource.Database, basePath+".database", result)
	}

	// Validate collection name if provided
	if resource.Collection != "" {
		validateCollectionName(resource.Collection, basePath+".collection", result)
	}
}

func validateCustomRoleInheritedRole(inheritedRole *types.CustomRoleInheritedRoleConfig, basePath string, result *ValidationResult) {
	// Validate role name
	if inheritedRole.RoleName == "" {
		addError(result, basePath+".roleName", "roleName", "",
			"inherited role name is required", "REQUIRED_FIELD_MISSING")
	}

	// Validate database name
	if inheritedRole.DatabaseName == "" {
		addError(result, basePath+".databaseName", "databaseName", "",
			"inherited role database name is required", "REQUIRED_FIELD_MISSING")
	} else {
		validateDatabaseName(inheritedRole.DatabaseName, basePath+".databaseName", result)
	}
}

func validateUserScope(scope *types.UserScopeConfig, basePath string, result *ValidationResult) {
	// Validate scope name
	if scope.Name == "" {
		addError(result, basePath+".name", "name", "",
			"scope name is required", "REQUIRED_FIELD_MISSING")
	}

	// Validate scope type
	validTypes := []string{"CLUSTER", "DATA_LAKE"}
	valid := false
	for _, validType := range validTypes {
		if scope.Type == validType {
			valid = true
			break
		}
	}
	if !valid {
		addError(result, basePath+".type", "type", scope.Type,
			fmt.Sprintf("invalid scope type (valid: %v)", validTypes), "INVALID_SCOPE_TYPE")
	}
}

func validateResourceMetadata(metadata *types.ResourceMetadata, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Validate name (default behavior)
	if metadata.Name == "" {
		addError(result, basePath+".name", "name", "",
			"name is required", "REQUIRED_FIELD_MISSING")
	} else {
		validateResourceName(metadata.Name, basePath+".name", result, opts)
	}

	// Validate other metadata fields
	validateMetadataFields(metadata, basePath, result, opts)
}

func validateResourceMetadataWithKind(metadata *types.ResourceMetadata, kind types.ResourceKind, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	// Validate name with context awareness
	if metadata.Name == "" {
		addError(result, basePath+".name", "name", "",
			"name is required", "REQUIRED_FIELD_MISSING")
	} else {
		validateResourceNameWithKind(metadata.Name, kind, basePath+".name", result, opts)
	}

	// Validate other metadata fields
	validateMetadataFields(metadata, basePath, result, opts)
}

func validateMetadataFields(metadata *types.ResourceMetadata, basePath string, result *ValidationResult, opts *ValidatorOptions) {

	// Validate deletion policy
	if metadata.DeletionPolicy != "" {
		validPolicies := []types.DeletionPolicy{
			types.DeletionPolicyDelete,
			types.DeletionPolicyRetain,
			types.DeletionPolicySnapshot,
		}
		valid := false
		for _, validPolicy := range validPolicies {
			if metadata.DeletionPolicy == validPolicy {
				valid = true
				break
			}
		}
		if !valid {
			addError(result, basePath+".deletionPolicy", "deletionPolicy", string(metadata.DeletionPolicy),
				fmt.Sprintf("invalid deletion policy (valid: %v)", validPolicies), "INVALID_DELETION_POLICY")
		}
	}

	// Validate labels and annotations
	for key, value := range metadata.Labels {
		validateLabelKey(key, basePath+".labels", result)
		validateLabelValue(value, basePath+".labels", result)
	}

	for key, value := range metadata.Annotations {
		validateAnnotationKey(key, basePath+".annotations", result)
		validateAnnotationValue(value, basePath+".annotations", result)
	}
}

func validateResourceNameWithKind(name string, kind types.ResourceKind, path string, result *ValidationResult, opts *ValidatorOptions) {
	// For NetworkAccess resources, allow CIDR notation (with dots and slashes)
	if kind == types.KindNetworkAccess {
		validateNetworkAccessName(name, path, result, opts)
	} else {
		// Use standard validation for other resource types
		validateResourceName(name, path, result, opts)
	}
}

func validateNetworkAccessName(name, path string, result *ValidationResult, opts *ValidatorOptions) {
	if strings.TrimSpace(name) == "" {
		addError(result, path, "name", name,
			"name cannot be empty", "EMPTY_NAME")
		return
	}

	if len(name) > opts.MaxNameLength {
		addError(result, path, "name", name,
			fmt.Sprintf("name exceeds maximum length of %d characters", opts.MaxNameLength),
			"NAME_TOO_LONG")
		return
	}

	// Check for spaces
	if strings.Contains(name, " ") {
		addError(result, path, "name", name,
			"name cannot contain spaces", "INVALID_NAME")
		return
	}

	// For network access, allow CIDR notation (dots and slashes are valid)
	// Pattern allows: IP addresses (192.168.1.1), CIDR notation (192.168.0.0/16), and regular names
	pattern := regexp.MustCompile(`^[a-z0-9._/-]+$`)
	if !pattern.MatchString(name) {
		addError(result, path, "name", name,
			"network access name must contain only lowercase letters, numbers, dots, underscores, hyphens, and slashes",
			"INVALID_NETWORK_ACCESS_NAME")
	}
}

func validateResourceName(name, path string, result *ValidationResult, opts *ValidatorOptions) {
	if strings.TrimSpace(name) == "" {
		addError(result, path, "name", name,
			"name cannot be empty", "EMPTY_NAME")
		return
	}

	if len(name) > opts.MaxNameLength {
		addError(result, path, "name", name,
			fmt.Sprintf("name exceeds maximum length of %d characters", opts.MaxNameLength),
			"NAME_TOO_LONG")
		return
	}

	// Check for invalid characters (spaces, uppercase, special chars except underscore and hyphen)
	if strings.Contains(name, " ") {
		addError(result, path, "name", name,
			"name cannot contain spaces", "INVALID_NAME")
		return
	}

	// Check for uppercase letters
	if name != strings.ToLower(name) {
		addError(result, path, "name", name,
			"name must be lowercase", "INVALID_NAME")
		return
	}

	// Check for invalid special characters
	invalidChars := []string{"!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "+", "=", "[", "]", "{", "}", "|", "\\", ":", ";", "\"", "'", "<", ">", ",", ".", "?", "/", "~", "`"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			addError(result, path, "name", name,
				fmt.Sprintf("name contains invalid character: %s", char), "INVALID_NAME")
			return
		}
	}

	// Check if name starts or ends with hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		addError(result, path, "name", name,
			"name cannot start or end with hyphen", "INVALID_NAME")
		return
	}

	// Check for double dots
	if strings.Contains(name, "..") {
		addError(result, path, "name", name,
			"name cannot contain consecutive dots", "INVALID_NAME")
		return
	}

	// Allow alphanumeric characters, underscores, hyphens, and single dots
	pattern := regexp.MustCompile(`^[a-z0-9_.-]+$`)
	if !pattern.MatchString(name) {
		addError(result, path, "name", name,
			"name must contain only lowercase letters, numbers, underscores, hyphens, and dots",
			"INVALID_RESOURCE_NAME")
	}
}

func validateLabelKey(key, basePath string, result *ValidationResult) {
	if len(key) > 63 {
		addError(result, basePath, "label key", key,
			"label key exceeds maximum length of 63 characters", "LABEL_KEY_TOO_LONG")
	}

	pattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`)
	if !pattern.MatchString(key) {
		addError(result, basePath, "label key", key,
			"invalid label key format", "INVALID_LABEL_KEY")
	}
}

func validateLabelValue(value, basePath string, result *ValidationResult) {
	if len(value) > 63 {
		addError(result, basePath, "label value", value,
			"label value exceeds maximum length of 63 characters", "LABEL_VALUE_TOO_LONG")
	}
}

func validateAnnotationKey(key, basePath string, result *ValidationResult) {
	if len(key) > 253 {
		addError(result, basePath, "annotation key", key,
			"annotation key exceeds maximum length of 253 characters", "ANNOTATION_KEY_TOO_LONG")
	}
}

func validateAnnotationValue(value, basePath string, result *ValidationResult) {
	if len(value) > 10000 {
		addError(result, basePath, "annotation value", value,
			"annotation value exceeds maximum length of 10000 characters", "ANNOTATION_VALUE_TOO_LONG")
	}
}

func validateDatabaseName(name, path string, result *ValidationResult) {
	if strings.TrimSpace(name) == "" {
		addError(result, path, "databaseName", name,
			"database name cannot be empty", "EMPTY_DATABASE_NAME")
		return
	}

	if len(name) > 64 {
		addError(result, path, "databaseName", name,
			"database name exceeds maximum length of 64 characters", "DATABASE_NAME_TOO_LONG")
		return
	}

	// Check if name starts with a number
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		addError(result, path, "databaseName", name,
			"database name cannot start with a number", "INVALID_DATABASE_NAME")
		return
	}

	// MongoDB database name restrictions
	invalidChars := []string{"/", "\\", ".", " ", "\"", "$", "*", "<", ">", ":", "|", "?", "-"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			addError(result, path, "databaseName", name,
				fmt.Sprintf("database name contains invalid character: %s", char), "INVALID_DATABASE_NAME")
			return
		}
	}
}

func validateCollectionName(name, path string, result *ValidationResult) {
	if strings.TrimSpace(name) == "" {
		addError(result, path, "collectionName", name,
			"collection name cannot be empty", "EMPTY_COLLECTION_NAME")
		return
	}

	if len(name) > 120 {
		addError(result, path, "collectionName", name,
			"collection name exceeds maximum length of 120 characters", "COLLECTION_NAME_TOO_LONG")
		return
	}

	// Check for spaces
	if strings.Contains(name, " ") {
		addError(result, path, "collectionName", name,
			"collection names cannot contain spaces", "INVALID_COLLECTION_NAME")
		return
	}

	// MongoDB collection name restrictions
	if strings.HasPrefix(name, "system.") {
		addError(result, path, "collectionName", name,
			"collection names cannot start with 'system.'", "INVALID_COLLECTION_NAME")
		return
	}

	if strings.Contains(name, "$") {
		addError(result, path, "collectionName", name,
			"collection names cannot contain '$'", "INVALID_COLLECTION_NAME")
		return
	}
}

func addResourceToGraph(graph *types.DependencyGraph, name string, kind types.ResourceKind, namespace string, dependencies []string) {
	node := &types.ResourceNode{
		Name:         name,
		Kind:         kind,
		Namespace:    namespace,
		Dependencies: dependencies,
	}
	graph.AddResource(node)
}

func addError(result *ValidationResult, path, field, value, message, code string) {
	result.Errors = append(result.Errors, ValidationError{
		Path:     path,
		Field:    field,
		Value:    value,
		Message:  message,
		Code:     code,
		Severity: "error",
	})
}

func addWarning(result *ValidationResult, path, field, value, message, code string) {
	result.Warnings = append(result.Warnings, ValidationError{
		Path:     path,
		Field:    field,
		Value:    value,
		Message:  message,
		Code:     code,
		Severity: "warning",
	})
}

// validateCrossResourceDependencies performs enhanced cross-resource dependency validation
func validateCrossResourceDependencies(config *types.ApplyConfig, result *ValidationResult, opts *ValidatorOptions) {
	// Import the dependency validator
	depValidator := validation.NewDependencyValidator(opts.StrictMode)

	// Run dependency validation
	depIssues, err := depValidator.ValidateProjectDependencies(&config.Spec)
	if err != nil {
		addError(result, "spec", "dependencies", "",
			fmt.Sprintf("Dependency validation failed: %v", err), "DEPENDENCY_VALIDATION_ERROR")
		return
	}

	// Convert dependency issues to validation errors
	for _, issue := range depIssues {
		severity := "warning"
		if issue.Severity == "error" {
			severity = "error"
		}

		validationError := ValidationError{
			Path:     issue.SourceResource,
			Field:    issue.DependencyType,
			Value:    issue.TargetResource,
			Message:  issue.Message,
			Code:     strings.ToUpper(strings.Replace(issue.DependencyType, " ", "_", -1)),
			Severity: severity,
		}

		if severity == "error" {
			result.Errors = append(result.Errors, validationError)
		} else {
			result.Warnings = append(result.Warnings, validationError)
		}
	}

	// Business logic validation
	validateBusinessLogicRules(config, result, opts)
}

// validateCrossDocumentDependencies validates dependencies across multiple resources in a document
func validateCrossDocumentDependencies(doc *types.ApplyDocument, result *ValidationResult, opts *ValidatorOptions) {
	// Check for resource name conflicts across the document
	resourceNames := make(map[string][]string)

	for i, resource := range doc.Resources {
		resourceType := fmt.Sprintf("%s.%s", resource.APIVersion, resource.Kind)
		if resourceNames[resourceType] == nil {
			resourceNames[resourceType] = []string{}
		}

		// Extract resource name based on type
		var name string
		if resource.Kind == "Project" && resource.Spec != nil {
			if projectSpec, ok := resource.Spec.(map[string]interface{}); ok {
				if nameVal, ok := projectSpec["name"].(string); ok {
					name = nameVal
				}
			}
		}

		if name != "" {
			// Check for duplicates
			for j, existingName := range resourceNames[resourceType] {
				if existingName == name {
					addError(result, fmt.Sprintf("resources[%d]", i), "name", name,
						fmt.Sprintf("Duplicate resource name '%s' conflicts with resources[%d]", name, j),
						"DUPLICATE_RESOURCE_NAME")
				}
			}
			resourceNames[resourceType] = append(resourceNames[resourceType], name)
		}
	}
}

// validateBusinessLogicRules validates Atlas-specific business logic rules
func validateBusinessLogicRules(config *types.ApplyConfig, result *ValidationResult, opts *ValidatorOptions) {
	// Validate cluster configurations against Atlas constraints
	for i, cluster := range config.Spec.Clusters {
		path := fmt.Sprintf("spec.clusters[%d]", i)

		// Validate instance size and storage compatibility
		if cluster.DiskSizeGB != nil {
			if !isInstanceSizeStorageCompatible(cluster.InstanceSize, *cluster.DiskSizeGB) {
				addWarning(result, path+".diskSizeGB", "diskSizeGB", fmt.Sprintf("%.1f", *cluster.DiskSizeGB),
					fmt.Sprintf("Disk size %.1fGB may not be optimal for instance size %s", *cluster.DiskSizeGB, cluster.InstanceSize),
					"SUBOPTIMAL_DISK_SIZE")
			}
		}

		// Validate backup settings for instance size
		if cluster.BackupEnabled != nil && *cluster.BackupEnabled {
			if !isBackupSupportedForInstanceSize(cluster.InstanceSize) {
				addError(result, path+".backupEnabled", "backupEnabled", "true",
					fmt.Sprintf("Backup is not supported for instance size %s", cluster.InstanceSize),
					"UNSUPPORTED_BACKUP_INSTANCE_SIZE")
			}
		}

		// Validate MongoDB version compatibility
		if cluster.MongoDBVersion != "" {
			if !isMongoDBVersionSupported(cluster.MongoDBVersion) {
				addError(result, path+".mongodbVersion", "mongodbVersion", cluster.MongoDBVersion,
					fmt.Sprintf("MongoDB version %s is not supported", cluster.MongoDBVersion),
					"UNSUPPORTED_MONGODB_VERSION")
			}
		}

		// Validate cluster type and replication specs compatibility
		if cluster.ClusterType != "" && len(cluster.ReplicationSpecs) > 0 {
			if !isClusterTypeReplicationCompatible(cluster.ClusterType, cluster.ReplicationSpecs) {
				addError(result, path+".replicationSpecs", "replicationSpecs", fmt.Sprintf("%d specs", len(cluster.ReplicationSpecs)),
					fmt.Sprintf("Replication specs are not compatible with cluster type %s", cluster.ClusterType),
					"INCOMPATIBLE_CLUSTER_TYPE_REPLICATION")
			}
		}
	}

	// Validate database user configurations
	for i, user := range config.Spec.DatabaseUsers {
		path := fmt.Sprintf("spec.databaseUsers[%d]", i)

		// Validate role combinations
		if !areRolesCombinationValid(user.Roles) {
			addWarning(result, path+".roles", "roles", fmt.Sprintf("%d roles", len(user.Roles)),
				"Some role combinations may conflict or be redundant",
				"SUBOPTIMAL_ROLE_COMBINATION")
		}

		// Validate authentication database
		if user.AuthDatabase != "" {
			if !isAuthDatabaseValid(user.AuthDatabase, user.Roles) {
				addError(result, path+".authDatabase", "authDatabase", user.AuthDatabase,
					fmt.Sprintf("Authentication database %s is not valid for the specified roles", user.AuthDatabase),
					"INVALID_AUTH_DATABASE")
			}
		}
	}

	// Validate network access configurations
	for i, netAccess := range config.Spec.NetworkAccess {
		path := fmt.Sprintf("spec.networkAccess[%d]", i)

		// Validate security implications
		if netAccess.IPAddress == "0.0.0.0" || netAccess.CIDR == "0.0.0.0/0" {
			addWarning(result, path, "ipAddress", netAccess.IPAddress,
				"Allowing access from 0.0.0.0 creates a security risk",
				"SECURITY_RISK_OPEN_ACCESS")
		}

		// Validate AWS security group format
		if netAccess.AWSSecurityGroup != "" {
			if !isValidAWSSecurityGroup(netAccess.AWSSecurityGroup) {
				addError(result, path+".awsSecurityGroup", "awsSecurityGroup", netAccess.AWSSecurityGroup,
					"Invalid AWS security group format",
					"INVALID_AWS_SECURITY_GROUP")
			}
		}
	}
}

// Helper functions for business logic validation

func isInstanceSizeStorageCompatible(instanceSize string, diskSizeGB float64) bool {
	// Define storage recommendations for instance sizes
	storageRecommendations := map[string]struct{ min, max float64 }{
		"M0":   {min: 0.5, max: 5},
		"M2":   {min: 2, max: 8},
		"M5":   {min: 5, max: 20},
		"M10":  {min: 10, max: 80},
		"M20":  {min: 20, max: 160},
		"M30":  {min: 40, max: 400},
		"M40":  {min: 80, max: 800},
		"M50":  {min: 160, max: 1600},
		"M60":  {min: 320, max: 3200},
		"M80":  {min: 750, max: 3200},
		"M140": {min: 1000, max: 4096},
		"M200": {min: 1500, max: 4096},
		"M300": {min: 2000, max: 4096},
	}

	if rec, exists := storageRecommendations[instanceSize]; exists {
		return diskSizeGB >= rec.min && diskSizeGB <= rec.max
	}
	return true // Unknown instance sizes pass
}

func isBackupSupportedForInstanceSize(instanceSize string) bool {
	// M0 clusters don't support backup
	return instanceSize != "M0"
}

func isMongoDBVersionSupported(version string) bool {
	supportedVersions := []string{"4.4", "5.0", "6.0", "7.0"}
	for _, supported := range supportedVersions {
		if strings.HasPrefix(version, supported) {
			return true
		}
	}
	return false
}

func isClusterTypeReplicationCompatible(clusterType string, replicationSpecs []types.ReplicationSpec) bool {
	switch clusterType {
	case "REPLICASET":
		// Replica sets should have exactly one replication spec
		return len(replicationSpecs) == 1
	case "SHARDED":
		// Sharded clusters can have multiple replication specs
		return len(replicationSpecs) >= 1
	case "GEOSHARDED":
		// Geo-sharded clusters need multiple replication specs across regions
		return len(replicationSpecs) >= 2
	}
	return true
}

func areRolesCombinationValid(roles []types.DatabaseRoleConfig) bool {
	// Check for conflicting or redundant roles
	roleNames := make(map[string]bool)
	for _, role := range roles {
		if roleNames[role.RoleName] {
			return false // Duplicate role
		}
		roleNames[role.RoleName] = true
	}

	// Check for conflicting combinations
	if roleNames["atlasAdmin"] && (roleNames["read"] || roleNames["readWrite"]) {
		return false // atlasAdmin already includes read/readWrite
	}

	return true
}

func isAuthDatabaseValid(authDatabase string, roles []types.DatabaseRoleConfig) bool {
	// For administrative roles, auth database should be "admin"
	adminRoles := map[string]bool{
		"atlasAdmin": true, "readWriteAnyDatabase": true, "readAnyDatabase": true,
		"clusterAdmin": true, "clusterManager": true, "clusterMonitor": true,
		"hostManager": true, "backup": true, "restore": true,
	}

	hasAdminRole := false
	for _, role := range roles {
		if adminRoles[role.RoleName] {
			hasAdminRole = true
			break
		}
	}

	if hasAdminRole && authDatabase != "admin" {
		return false
	}

	return true
}

func isValidAWSSecurityGroup(sgID string) bool {
	// AWS security group IDs start with "sg-" followed by 8 or 17 hex characters
	if !strings.HasPrefix(sgID, "sg-") {
		return false
	}

	suffix := sgID[3:]
	if len(suffix) != 8 && len(suffix) != 17 {
		return false
	}

	// Check if all characters after "sg-" are hex
	for _, char := range suffix {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}

	return true
}

// convertMapToStruct converts a map[string]interface{} to a struct using JSON marshaling
func convertMapToStruct(input map[string]interface{}, output interface{}) error {
	// Marshal the map to JSON
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal map to JSON: %w", err)
	}

	// Unmarshal JSON to the target struct
	if err := json.Unmarshal(jsonBytes, output); err != nil {
		return fmt.Errorf("failed to unmarshal JSON to struct: %w", err)
	}

	return nil
}

// validateSearchIndexManifest validates a SearchIndex resource manifest
func validateSearchIndexManifest(manifest *types.ResourceManifest, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	spec, ok := manifest.Spec.(types.SearchIndexSpec)
	if !ok {
		// Try to convert from map[string]interface{}
		if specMap, ok := manifest.Spec.(map[string]interface{}); ok {
			spec = convertToSearchIndexSpec(specMap)
		} else {
			addError(result, basePath+".spec", "spec", "",
				"search index spec must be an object", "INVALID_SPEC_TYPE")
			return
		}
	}

	// Validate required fields
	if spec.ProjectName == "" {
		addError(result, basePath+".spec.projectName", "projectName", "",
			"project name is required", "REQUIRED_FIELD_MISSING")
	}

	if spec.ClusterName == "" {
		addError(result, basePath+".spec.clusterName", "clusterName", "",
			"cluster name is required", "REQUIRED_FIELD_MISSING")
	}

	if spec.DatabaseName == "" {
		addError(result, basePath+".spec.databaseName", "databaseName", "",
			"database name is required", "REQUIRED_FIELD_MISSING")
	}

	if spec.CollectionName == "" {
		addError(result, basePath+".spec.collectionName", "collectionName", "",
			"collection name is required", "REQUIRED_FIELD_MISSING")
	}

	if spec.IndexName == "" {
		addError(result, basePath+".spec.indexName", "indexName", "",
			"index name is required", "REQUIRED_FIELD_MISSING")
	}

	// Validate index type if provided
	if spec.IndexType != "" && spec.IndexType != "search" && spec.IndexType != "vectorSearch" {
		addError(result, basePath+".spec.indexType", "indexType", spec.IndexType,
			"index type must be 'search' or 'vectorSearch'", "INVALID_INDEX_TYPE")
	}

	// Validate definition is provided
	if spec.Definition == nil {
		addError(result, basePath+".spec.definition", "definition", "",
			"index definition is required", "REQUIRED_FIELD_MISSING")
	}
}

// validateVPCEndpointManifest validates a VPCEndpoint resource manifest
func validateVPCEndpointManifest(manifest *types.ResourceManifest, basePath string, result *ValidationResult, opts *ValidatorOptions) {
	spec, ok := manifest.Spec.(types.VPCEndpointSpec)
	if !ok {
		// Try to convert from map[string]interface{}
		if specMap, ok := manifest.Spec.(map[string]interface{}); ok {
			spec = convertToVPCEndpointSpec(specMap)
		} else {
			addError(result, basePath+".spec", "spec", "",
				"VPC endpoint spec must be an object", "INVALID_SPEC_TYPE")
			return
		}
	}

	// Validate required fields
	if spec.ProjectName == "" {
		addError(result, basePath+".spec.projectName", "projectName", "",
			"project name is required", "REQUIRED_FIELD_MISSING")
	}

	if spec.CloudProvider == "" {
		addError(result, basePath+".spec.cloudProvider", "cloudProvider", "",
			"cloud provider is required", "REQUIRED_FIELD_MISSING")
	} else {
		// Validate cloud provider is supported
		validProviders := map[string]bool{
			"AWS":   true,
			"AZURE": true,
			"GCP":   true,
		}
		if !validProviders[spec.CloudProvider] {
			addError(result, basePath+".spec.cloudProvider", "cloudProvider", spec.CloudProvider,
				"cloud provider must be one of: AWS, AZURE, GCP", "INVALID_CLOUD_PROVIDER")
		}
	}

	if spec.Region == "" {
		addError(result, basePath+".spec.region", "region", "",
			"region is required", "REQUIRED_FIELD_MISSING")
	}
}

// convertToSearchIndexSpec converts a map to SearchIndexSpec
func convertToSearchIndexSpec(specMap map[string]interface{}) types.SearchIndexSpec {
	spec := types.SearchIndexSpec{}

	if val, ok := specMap["projectName"].(string); ok {
		spec.ProjectName = val
	}
	if val, ok := specMap["clusterName"].(string); ok {
		spec.ClusterName = val
	}
	if val, ok := specMap["databaseName"].(string); ok {
		spec.DatabaseName = val
	}
	if val, ok := specMap["collectionName"].(string); ok {
		spec.CollectionName = val
	}
	if val, ok := specMap["indexName"].(string); ok {
		spec.IndexName = val
	}
	if val, ok := specMap["indexType"].(string); ok {
		spec.IndexType = val
	}
	if val, ok := specMap["definition"]; ok {
		if definitionMap, ok := val.(map[string]interface{}); ok {
			spec.Definition = definitionMap
		}
	}
	if val, ok := specMap["dependsOn"].([]interface{}); ok {
		spec.DependsOn = make([]string, len(val))
		for i, dep := range val {
			if depStr, ok := dep.(string); ok {
				spec.DependsOn[i] = depStr
			}
		}
	}

	return spec
}

// convertToVPCEndpointSpec converts a map to VPCEndpointSpec
func convertToVPCEndpointSpec(specMap map[string]interface{}) types.VPCEndpointSpec {
	spec := types.VPCEndpointSpec{}

	if val, ok := specMap["projectName"].(string); ok {
		spec.ProjectName = val
	}
	if val, ok := specMap["cloudProvider"].(string); ok {
		spec.CloudProvider = val
	}
	if val, ok := specMap["region"].(string); ok {
		spec.Region = val
	}
	if val, ok := specMap["endpointId"].(string); ok {
		spec.EndpointID = val
	}
	if val, ok := specMap["dependsOn"].([]interface{}); ok {
		spec.DependsOn = make([]string, len(val))
		for i, dep := range val {
			if depStr, ok := dep.(string); ok {
				spec.DependsOn[i] = depStr
			}
		}
	}

	return spec
}
