package dag

import (
	"context"
	"strings"

	"github.com/teabranch/matlas-cli/internal/types"
)

// GetBuiltinRules returns all built-in dependency rules
func GetBuiltinRules() []Rule {
	return []Rule{
		// High priority: Project dependencies
		NewProjectDependencyRule(),
		
		// Medium-high priority: Resource kind dependencies
		NewClusterDependencyRule(),
		NewRoleDependencyRule(),
		NewVPCDependencyRule(),
		
		// Medium priority: Ordering rules
		NewNetworkAccessOrderingRule(),
		NewSearchIndexOrderingRule(),
		
		// Lower priority: Same-cluster conflict detection
		NewSameClusterConflictRule(),
	}
}

// NewProjectDependencyRule creates a rule for project dependencies
// All resources depend on their project
func NewProjectDependencyRule() Rule {
	return NewResourceKindRule(
		"project_dependency",
		"All resources must wait for their project to exist",
		200, // Highest priority
		types.KindCluster, // From any cluster
		types.KindProject, // To project
		DependencyTypeHard,
		func(from, to *PlannedOperation) bool {
			// Check if they're in the same project
			return extractProjectName(from.Spec) == extractProjectName(to.Spec)
		},
	)
}

// NewClusterDependencyRule creates a rule for cluster dependencies
// Database users, indexes, and roles depend on clusters
func NewClusterDependencyRule() Rule {
	return NewPropertyBasedRule(
		"cluster_dependency",
		"Database users, indexes, and roles require their cluster to exist",
		150,
		func(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
			// Check if from is a cluster-dependent resource
			clusterDependent := from.ResourceType == types.KindDatabaseUser ||
				from.ResourceType == types.KindDatabaseRole ||
				from.ResourceType == types.KindSearchIndex
			
			if !clusterDependent || to.ResourceType != types.KindCluster {
				return nil, nil
			}
			
			// Check if they reference the same cluster
			fromCluster := extractClusterName(from.Spec)
			toCluster := extractClusterName(to.Spec)
			
			if fromCluster != "" && toCluster != "" && fromCluster == toCluster {
				return &Edge{
					Type:   DependencyTypeHard,
					Weight: 1.0,
					Reason: "Resource requires cluster to exist",
				}, nil
			}
			
			return nil, nil
		},
	)
}

// NewRoleDependencyRule creates a rule for role dependencies
// Database users that reference custom roles depend on those roles
func NewRoleDependencyRule() Rule {
	return NewPropertyBasedRule(
		"role_dependency",
		"Database users require custom roles to exist first",
		140,
		func(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
			if from.ResourceType != types.KindDatabaseUser || to.ResourceType != types.KindDatabaseRole {
				return nil, nil
			}
			
			// Check if the user references this role
			userRoles := extractUserRoles(from.Spec)
			roleName := extractRoleName(to.Spec)
			
			for _, userRole := range userRoles {
				if userRole == roleName {
					return &Edge{
						Type:   DependencyTypeHard,
						Weight: 1.0,
						Reason: "User requires custom role to exist",
					}, nil
				}
			}
			
			return nil, nil
		},
	)
}

// NewVPCDependencyRule creates a rule for VPC endpoint dependencies
func NewVPCDependencyRule() Rule {
	return NewResourceKindRule(
		"vpc_dependency",
		"Clusters using VPC endpoints depend on the endpoint",
		145,
		types.KindCluster,
		types.KindVPCEndpoint,
		DependencyTypeHard,
		func(from, to *PlannedOperation) bool {
			// Check if cluster spec references this VPC endpoint
			clusterVPC := extractVPCEndpointID(from.Spec)
			vpcID := extractVPCID(to.Spec)
			return clusterVPC != "" && vpcID != "" && clusterVPC == vpcID
		},
	)
}

// NewNetworkAccessOrderingRule creates an ordering rule for network access
// Network access is typically configured after clusters (soft dependency)
func NewNetworkAccessOrderingRule() Rule {
	return NewResourceKindRule(
		"network_access_ordering",
		"Network access configuration typically follows cluster creation",
		50, // Lower priority
		types.KindNetworkAccess,
		types.KindCluster,
		DependencyTypeSoft,
		func(from, to *PlannedOperation) bool {
			// Same project
			return extractProjectName(from.Spec) == extractProjectName(to.Spec)
		},
	)
}

// NewSearchIndexOrderingRule creates an ordering rule for search indexes
func NewSearchIndexOrderingRule() Rule {
	return NewPropertyBasedRule(
		"search_index_ordering",
		"Search indexes created after database users for proper permissions",
		45,
		func(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
			if from.ResourceType != types.KindSearchIndex || to.ResourceType != types.KindDatabaseUser {
				return nil, nil
			}
			
			// Same cluster
			if extractClusterName(from.Spec) == extractClusterName(to.Spec) {
				return &Edge{
					Type:   DependencyTypeSoft,
					Weight: 0.5,
					Reason: "Search index benefits from having users configured first",
				}, nil
			}
			
			return nil, nil
		},
	)
}

// NewSameClusterConflictRule creates a mutual exclusion rule for same-cluster modifications
func NewSameClusterConflictRule() Rule {
	return NewMutualExclusionRule(
		"same_cluster_conflict",
		"Operations modifying the same cluster cannot run in parallel",
		100,
		func(from, to *PlannedOperation) bool {
			// Check if both are cluster modifications
			if from.ResourceType == types.KindCluster && to.ResourceType == types.KindCluster {
				return from.ResourceName == to.ResourceName
			}
			return false
		},
	)
}

// Helper functions to extract information from resource specs

func extractProjectName(spec interface{}) string {
	switch s := spec.(type) {
	case *types.ClusterManifest:
		return s.Spec.ProjectName
	case types.ClusterManifest:
		return s.Spec.ProjectName
	case *types.DatabaseUserManifest:
		return s.Spec.ProjectName
	case types.DatabaseUserManifest:
		return s.Spec.ProjectName
	case *types.NetworkAccessManifest:
		return s.Spec.ProjectName
	case types.NetworkAccessManifest:
		return s.Spec.ProjectName
	case *types.ProjectManifest:
		return s.Metadata.Name
	case types.ProjectManifest:
		return s.Metadata.Name
	default:
		return ""
	}
}

func extractClusterName(spec interface{}) string {
	switch s := spec.(type) {
	case *types.ClusterManifest:
		return s.Metadata.Name
	case types.ClusterManifest:
		return s.Metadata.Name
	case *types.DatabaseUserManifest:
		// Users don't directly reference clusters, but are scoped to them
		// This would need more context from the spec
		return ""
	case *types.SearchIndexManifest:
		return s.Spec.ClusterName
	case types.SearchIndexManifest:
		return s.Spec.ClusterName
	default:
		return ""
	}
}

func extractUserRoles(spec interface{}) []string {
	switch s := spec.(type) {
	case *types.DatabaseUserManifest:
		roles := make([]string, 0, len(s.Spec.Roles))
		for _, role := range s.Spec.Roles {
			roles = append(roles, role.RoleName)
		}
		return roles
	case types.DatabaseUserManifest:
		roles := make([]string, 0, len(s.Spec.Roles))
		for _, role := range s.Spec.Roles {
			roles = append(roles, role.RoleName)
		}
		return roles
	default:
		return nil
	}
}

func extractRoleName(spec interface{}) string {
	switch s := spec.(type) {
	case *types.DatabaseRoleManifest:
		return s.Spec.RoleName
	case types.DatabaseRoleManifest:
		return s.Spec.RoleName
	default:
		return ""
	}
}

func extractVPCEndpointID(spec interface{}) string {
	// This would need to check cluster spec for VPC endpoint references
	// Placeholder implementation
	return ""
}

func extractVPCID(spec interface{}) string {
	// This would extract VPC endpoint ID from VPC endpoint manifest
	// Placeholder implementation
	return ""
}

// NewAPIQuotaRule creates a rule for respecting API quotas
// This is a resource-based dependency that limits parallel operations
func NewAPIQuotaRule(maxConcurrentOps int) Rule {
	return NewPropertyBasedRule(
		"api_quota_rule",
		"Respect Atlas API rate limits by limiting concurrent operations",
		75,
		func(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
			// This would need runtime tracking of concurrent operations
			// For now, we can mark certain operations as resource-dependent
			// The actual enforcement would happen in the scheduler
			
			// High-cost operations should be serialized
			highCost := isHighCostOperation(from) && isHighCostOperation(to)
			if highCost {
				return &Edge{
					Type:   DependencyTypeResource,
					Weight: 5.0,
					Reason: "High-cost operations should be rate-limited",
				}, nil
			}
			
			return nil, nil
		},
	)
}

func isHighCostOperation(op *PlannedOperation) bool {
	// Cluster creation/modification is high cost
	if op.ResourceType == types.KindCluster {
		return true
	}
	
	// VPC endpoint operations are high cost
	if op.ResourceType == types.KindVPCEndpoint {
		return true
	}
	
	return false
}

// NewConditionalDependencyRule creates a rule for conditional dependencies
// Example: Backup-enabled clusters depend on backup configuration
func NewConditionalDependencyRule() Rule {
	return NewPropertyBasedRule(
		"conditional_backup_dependency",
		"Clusters with backup enabled depend on backup configuration",
		120,
		func(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
			if from.ResourceType != types.KindCluster {
				return nil, nil
			}
			
			// Check if cluster has backup enabled
			if hasBackupEnabled(from.Spec) {
				// Would depend on backup config resource if it exists
				// This is a placeholder for demonstration
				return &Edge{
					Type:      DependencyTypeConditional,
					Weight:    1.0,
					Reason:    "Cluster with backup requires backup configuration",
					Condition: &Condition{
						PropertyPath: "spec.backupEnabled",
						Operator:     "==",
						Value:        true,
					},
				}, nil
			}
			
			return nil, nil
		},
	)
}

func hasBackupEnabled(spec interface{}) bool {
	switch s := spec.(type) {
	case *types.ClusterManifest:
		return s.Spec.BackupEnabled != nil && *s.Spec.BackupEnabled
	case types.ClusterManifest:
		return s.Spec.BackupEnabled != nil && *s.Spec.BackupEnabled
	default:
		return false
	}
}

// NewSameResourceUpdateRule prevents concurrent updates to the same resource
func NewSameResourceUpdateRule() Rule {
	return NewMutualExclusionRule(
		"same_resource_update",
		"Updates to the same resource must be serialized",
		200, // Very high priority
		func(from, to *PlannedOperation) bool {
			// If both operations target the same resource, they conflict
			return from.ResourceType == to.ResourceType &&
				from.ResourceName != "" &&
				from.ResourceName == to.ResourceName
		},
	)
}

// NewCrossRegionOrderingRule creates ordering for cross-region operations
func NewCrossRegionOrderingRule() Rule {
	return NewPropertyBasedRule(
		"cross_region_ordering",
		"Cross-region resources should follow a specific order",
		60,
		func(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
			fromRegion := extractRegion(from.Spec)
			toRegion := extractRegion(to.Spec)
			
			// If different regions, create soft ordering
			if fromRegion != "" && toRegion != "" && fromRegion != toRegion {
				return &Edge{
					Type:   DependencyTypeOrdering,
					Weight: 0.3,
					Reason: "Cross-region operations ordered for consistency",
				}, nil
			}
			
			return nil, nil
		},
	)
}

func extractRegion(spec interface{}) string {
	switch s := spec.(type) {
	case *types.ClusterManifest:
		return s.Spec.Region
	case types.ClusterManifest:
		return s.Spec.Region
	default:
		return ""
	}
}

// NewNamePrefixOrderingRule creates ordering based on resource name prefixes
// Useful for ensuring dev resources are created before prod, etc.
func NewNamePrefixOrderingRule(precedence []string) Rule {
	return NewPropertyBasedRule(
		"name_prefix_ordering",
		"Order resources based on name prefix (e.g., dev before prod)",
		30,
		func(ctx context.Context, from, to *PlannedOperation) (*Edge, error) {
			fromIdx := -1
			toIdx := -1
			
			for i, prefix := range precedence {
				if strings.HasPrefix(from.ResourceName, prefix) {
					fromIdx = i
				}
				if strings.HasPrefix(to.ResourceName, prefix) {
					toIdx = i
				}
			}
			
			// If 'from' has higher precedence (lower index), it depends on 'to'
			if fromIdx > toIdx && toIdx >= 0 {
				return &Edge{
					Type:   DependencyTypeOrdering,
					Weight: 0.2,
					Reason: "Resource name prefix ordering",
				}, nil
			}
			
			return nil, nil
		},
	)
}
