package apply

import (
	"context"
	"fmt"

	"github.com/teabranch/matlas-cli/internal/types"
)

// QuotaValidator defines the interface for validating resource quotas
type QuotaValidator interface {
	ValidateProjectQuotas(ctx context.Context, orgID string, config types.ProjectConfig) error
	ValidateClusterQuotas(ctx context.Context, orgID string, clusters []types.ClusterConfig) error
	ValidateDatabaseUserQuotas(ctx context.Context, orgID string, users []types.DatabaseUserConfig) error
	ValidateNetworkAccessQuotas(ctx context.Context, orgID string, access []types.NetworkAccessConfig) error
}

// AtlasQuotaValidator implements quota validation using Atlas API
type AtlasQuotaValidator struct {
	// In a real implementation, this would have an Atlas client
	// For now, we'll use mock limits
	maxClusters           int
	maxDatabaseUsers      int
	maxNetworkAccessRules int
	maxProjectsPerOrg     int
}

// NewAtlasQuotaValidator creates a new quota validator
func NewAtlasQuotaValidator() *AtlasQuotaValidator {
	return &AtlasQuotaValidator{
		maxClusters:           25,  // Default Atlas limit for clusters per project
		maxDatabaseUsers:      100, // Default Atlas limit for database users per project
		maxNetworkAccessRules: 200, // Default Atlas limit for network access rules per project
		maxProjectsPerOrg:     250, // Default Atlas limit for projects per organization
	}
}

// OrganizationLimits represents the quota limits for an organization
type OrganizationLimits struct {
	MaxProjects           int      `json:"maxProjects"`
	MaxClustersPerProject int      `json:"maxClustersPerProject"`
	MaxUsersPerProject    int      `json:"maxUsersPerProject"`
	MaxNetworkRules       int      `json:"maxNetworkRules"`
	MaxInstanceSize       string   `json:"maxInstanceSize"`
	AllowedProviders      []string `json:"allowedProviders"`
	AllowedRegions        []string `json:"allowedRegions"`
}

// ProjectResourceCounts represents current resource usage in a project
type ProjectResourceCounts struct {
	Clusters      int `json:"clusters"`
	DatabaseUsers int `json:"databaseUsers"`
	NetworkRules  int `json:"networkRules"`
}

// QuotaValidationError represents a quota validation failure
type QuotaValidationError struct {
	ResourceType string `json:"resourceType"`
	Current      int    `json:"current"`
	Requested    int    `json:"requested"`
	Limit        int    `json:"limit"`
	Message      string `json:"message"`
}

func (e QuotaValidationError) Error() string {
	return e.Message
}

// ValidateProjectQuotas validates project-level quotas
func (v *AtlasQuotaValidator) ValidateProjectQuotas(ctx context.Context, orgID string, config types.ProjectConfig) error {
	// Validate organization limits
	limits, err := v.getOrganizationLimits(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get organization limits: %w", err)
	}

	// Validate cluster count
	if len(config.Clusters) > limits.MaxClustersPerProject {
		return QuotaValidationError{
			ResourceType: "Cluster",
			Current:      0, // In real implementation, get current count from Atlas
			Requested:    len(config.Clusters),
			Limit:        limits.MaxClustersPerProject,
			Message:      fmt.Sprintf("Project cannot have more than %d clusters (requested: %d)", limits.MaxClustersPerProject, len(config.Clusters)),
		}
	}

	// Validate database user count
	if len(config.DatabaseUsers) > limits.MaxUsersPerProject {
		return QuotaValidationError{
			ResourceType: "DatabaseUser",
			Current:      0,
			Requested:    len(config.DatabaseUsers),
			Limit:        limits.MaxUsersPerProject,
			Message:      fmt.Sprintf("Project cannot have more than %d database users (requested: %d)", limits.MaxUsersPerProject, len(config.DatabaseUsers)),
		}
	}

	// Validate network access rules count
	if len(config.NetworkAccess) > limits.MaxNetworkRules {
		return QuotaValidationError{
			ResourceType: "NetworkAccess",
			Current:      0,
			Requested:    len(config.NetworkAccess),
			Limit:        limits.MaxNetworkRules,
			Message:      fmt.Sprintf("Project cannot have more than %d network access rules (requested: %d)", limits.MaxNetworkRules, len(config.NetworkAccess)),
		}
	}

	return nil
}

// ValidateClusterQuotas validates cluster-specific quotas
func (v *AtlasQuotaValidator) ValidateClusterQuotas(ctx context.Context, orgID string, clusters []types.ClusterConfig) error {
	limits, err := v.getOrganizationLimits(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get organization limits: %w", err)
	}

	for _, cluster := range clusters {
		// Validate provider restrictions
		if !v.isProviderAllowed(cluster.Provider, limits.AllowedProviders) {
			return QuotaValidationError{
				ResourceType: "Cluster",
				Message:      fmt.Sprintf("Provider %s is not allowed for this organization. Allowed providers: %v", cluster.Provider, limits.AllowedProviders),
			}
		}

		// Validate region restrictions
		if !v.isRegionAllowed(cluster.Region, limits.AllowedRegions) {
			return QuotaValidationError{
				ResourceType: "Cluster",
				Message:      fmt.Sprintf("Region %s is not allowed for this organization. Allowed regions: %v", cluster.Region, limits.AllowedRegions),
			}
		}

		// Validate instance size restrictions
		if !v.isInstanceSizeAllowed(cluster.InstanceSize, limits.MaxInstanceSize) {
			return QuotaValidationError{
				ResourceType: "Cluster",
				Message:      fmt.Sprintf("Instance size %s exceeds maximum allowed size %s", cluster.InstanceSize, limits.MaxInstanceSize),
			}
		}

		// Validate replication specs
		for _, spec := range cluster.ReplicationSpecs {
			totalNodes := 0
			for _, region := range spec.RegionConfigs {
				if region.ElectableNodes != nil {
					totalNodes += *region.ElectableNodes
				}
				if region.ReadOnlyNodes != nil {
					totalNodes += *region.ReadOnlyNodes
				}
				if region.AnalyticsNodes != nil {
					totalNodes += *region.AnalyticsNodes
				}
			}

			if totalNodes > 50 { // Atlas limit for total nodes per cluster
				return QuotaValidationError{
					ResourceType: "Cluster",
					Message:      fmt.Sprintf("Total nodes (%d) in replication spec %s exceeds maximum allowed (50)", totalNodes, spec.ID),
				}
			}
		}
	}

	return nil
}

// ValidateDatabaseUserQuotas validates database user quotas
func (v *AtlasQuotaValidator) ValidateDatabaseUserQuotas(ctx context.Context, orgID string, users []types.DatabaseUserConfig) error {
	// Check for duplicate usernames
	usernames := make(map[string]bool)
	for _, user := range users {
		if usernames[user.Username] {
			return QuotaValidationError{
				ResourceType: "DatabaseUser",
				Message:      fmt.Sprintf("Duplicate database username: %s", user.Username),
			}
		}
		usernames[user.Username] = true

		// Validate password complexity if provided
		if user.Password != "" && len(user.Password) < 8 {
			return QuotaValidationError{
				ResourceType: "DatabaseUser",
				Message:      fmt.Sprintf("Password for user %s must be at least 8 characters", user.Username),
			}
		}

		// Validate role limits
		if len(user.Roles) > 20 { // Atlas limit for roles per user
			return QuotaValidationError{
				ResourceType: "DatabaseUser",
				Message:      fmt.Sprintf("User %s cannot have more than 20 roles (has: %d)", user.Username, len(user.Roles)),
			}
		}
	}

	return nil
}

// ValidateNetworkAccessQuotas validates network access quotas
func (v *AtlasQuotaValidator) ValidateNetworkAccessQuotas(ctx context.Context, orgID string, access []types.NetworkAccessConfig) error {
	// Check for conflicting CIDR ranges
	cidrs := make([]string, 0)
	for _, rule := range access {
		if rule.CIDR != "" {
			cidrs = append(cidrs, rule.CIDR)
		}

		// Validate comment length
		if len(rule.Comment) > 80 {
			return QuotaValidationError{
				ResourceType: "NetworkAccess",
				Message:      fmt.Sprintf("Comment for network access rule %s is too long (max 80 characters)", rule.Metadata.Name),
			}
		}
	}

	// In a real implementation, check for overlapping CIDR ranges
	// For now, just check for duplicates
	seen := make(map[string]bool)
	for _, cidr := range cidrs {
		if seen[cidr] {
			return QuotaValidationError{
				ResourceType: "NetworkAccess",
				Message:      fmt.Sprintf("Duplicate CIDR range: %s", cidr),
			}
		}
		seen[cidr] = true
	}

	return nil
}

// getOrganizationLimits fetches the quota limits for an organization
// In a real implementation, this would make an API call to Atlas
func (v *AtlasQuotaValidator) getOrganizationLimits(ctx context.Context, orgID string) (*OrganizationLimits, error) {
	// Mock implementation - in reality this would call Atlas API
	return &OrganizationLimits{
		MaxProjects:           v.maxProjectsPerOrg,
		MaxClustersPerProject: v.maxClusters,
		MaxUsersPerProject:    v.maxDatabaseUsers,
		MaxNetworkRules:       v.maxNetworkAccessRules,
		MaxInstanceSize:       "M700", // Maximum instance size allowed
		AllowedProviders:      []string{"AWS", "GCP", "AZURE"},
		AllowedRegions:        []string{"US_EAST_1", "us-west-2", "eu-west-1", "ap-southeast-1"},
	}, nil
}

// Helper functions for validation

func (v *AtlasQuotaValidator) isProviderAllowed(provider string, allowedProviders []string) bool {
	for _, allowed := range allowedProviders {
		if provider == allowed {
			return true
		}
	}
	return false
}

func (v *AtlasQuotaValidator) isRegionAllowed(region string, allowedRegions []string) bool {
	// If no restrictions, allow all regions
	if len(allowedRegions) == 0 {
		return true
	}

	for _, allowed := range allowedRegions {
		if region == allowed {
			return true
		}
	}
	return false
}

func (v *AtlasQuotaValidator) isInstanceSizeAllowed(instanceSize, maxInstanceSize string) bool {
	// Define instance size hierarchy
	sizeHierarchy := map[string]int{
		"M0": 0, "M2": 1, "M5": 2, "M10": 3, "M20": 4, "M30": 5,
		"M40": 6, "M50": 7, "M60": 8, "M80": 9, "M140": 10,
		"M200": 11, "M300": 12, "M400": 13, "M700": 14,
		"R40": 6, "R50": 7, "R60": 8, "R80": 9, "R200": 11,
		"R300": 12, "R400": 13, "R700": 14,
	}

	requestedLevel, exists := sizeHierarchy[instanceSize]
	if !exists {
		return false // Unknown instance size
	}

	maxLevel, exists := sizeHierarchy[maxInstanceSize]
	if !exists {
		return true // No restriction if max size is unknown
	}

	return requestedLevel <= maxLevel
}

// ValidateConfiguration is a convenience function that validates all aspects of a configuration
func ValidateConfiguration(ctx context.Context, validator QuotaValidator, orgID string, config types.ProjectConfig) error {
	if err := validator.ValidateProjectQuotas(ctx, orgID, config); err != nil {
		return fmt.Errorf("project quota validation failed: %w", err)
	}

	if err := validator.ValidateClusterQuotas(ctx, orgID, config.Clusters); err != nil {
		return fmt.Errorf("cluster quota validation failed: %w", err)
	}

	if err := validator.ValidateDatabaseUserQuotas(ctx, orgID, config.DatabaseUsers); err != nil {
		return fmt.Errorf("database user quota validation failed: %w", err)
	}

	if err := validator.ValidateNetworkAccessQuotas(ctx, orgID, config.NetworkAccess); err != nil {
		return fmt.Errorf("network access quota validation failed: %w", err)
	}

	return nil
}
