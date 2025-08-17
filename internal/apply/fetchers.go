package apply

import (
	"fmt"
	"strings"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// convertClusterToManifest converts an Atlas cluster to our ClusterManifest type
func (d *AtlasStateDiscovery) convertClusterToManifest(cluster *admin.ClusterDescription20240805, projectName string) types.ClusterManifest {
	metadata := types.ResourceMetadata{
		Name: cluster.GetName(),
		Labels: map[string]string{
			"atlas.mongodb.com/cluster-id": cluster.GetId(),
			"atlas.mongodb.com/project-id": cluster.GetGroupId(),
		},
	}

	spec := types.ClusterSpec{
		ProjectName:    projectName,
		Provider:       extractClusterProvider(*cluster),
		Region:         extractClusterRegion(*cluster),
		InstanceSize:   extractClusterTier(*cluster),
		TierType:       cluster.GetClusterType(),
		MongoDBVersion: cluster.GetMongoDBVersion(),
		ClusterType:    cluster.GetClusterType(),
	}

	// Map Atlas resource tags into manifest spec.tags if present
	if cluster.Tags != nil && len(*cluster.Tags) > 0 {
		tagsMap := make(map[string]string, len(*cluster.Tags))
		for _, rt := range *cluster.Tags {
			key := rt.GetKey()
			val := rt.GetValue()
			if key != "" {
				tagsMap[key] = val
			}
		}
		if len(tagsMap) > 0 {
			spec.Tags = tagsMap
		}
	}

	// Set backup enabled if available
	if backupEnabled := cluster.BackupEnabled; backupEnabled != nil {
		spec.BackupEnabled = backupEnabled
	}

	return types.ClusterManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindCluster,
		Metadata:   metadata,
		Spec:       spec,
		Status: &types.ResourceStatusInfo{
			Phase:      convertClusterStatus(cluster.GetStateName()),
			Message:    fmt.Sprintf("Cluster is %s", cluster.GetStateName()),
			LastUpdate: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// convertUserToManifest converts an Atlas database user to our DatabaseUserManifest type
func (d *AtlasStateDiscovery) convertUserToManifest(user *admin.CloudDatabaseUser) types.DatabaseUserManifest {
	metadata := types.ResourceMetadata{
		Name: user.GetUsername(),
		Labels: map[string]string{
			"atlas.mongodb.com/project-id":    user.GetGroupId(),
			"atlas.mongodb.com/auth-database": user.GetDatabaseName(),
			"atlas.mongodb.com/username":      user.GetUsername(),
		},
	}

	roles := make([]types.DatabaseRoleConfig, 0)
	if userRoles := user.Roles; userRoles != nil {
		for _, role := range *userRoles {
			roles = append(roles, types.DatabaseRoleConfig{
				RoleName:     role.GetRoleName(),
				DatabaseName: role.GetDatabaseName(),
			})
		}
	}

	scopes := make([]types.UserScopeConfig, 0)
	if userScopes := user.Scopes; userScopes != nil {
		for _, scope := range *userScopes {
			scopes = append(scopes, types.UserScopeConfig{
				Name: scope.GetName(),
				Type: scope.GetType(),
			})
		}
	}

	spec := types.DatabaseUserSpec{
		Username:     user.GetUsername(),
		Roles:        roles,
		AuthDatabase: user.GetDatabaseName(),
		Scopes:       scopes,
		// Note: Password is not included for security reasons
	}

	return types.DatabaseUserManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindDatabaseUser,
		Metadata:   metadata,
		Spec:       spec,
		Status: &types.ResourceStatusInfo{
			Phase:      types.StatusReady,
			Message:    "Database user is active",
			LastUpdate: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// convertNetworkAccessToManifest converts an Atlas network access entry to our NetworkAccessManifest type
func (d *AtlasStateDiscovery) convertNetworkAccessToManifest(entry *admin.NetworkPermissionEntry) types.NetworkAccessManifest {
	// Use the IP address or CIDR block as the resource name
	name := entry.GetIpAddress()
	if cidr := entry.GetCidrBlock(); cidr != "" {
		name = cidr
	}
	if aws := entry.GetAwsSecurityGroup(); aws != "" {
		name = aws
	}

	metadata := types.ResourceMetadata{
		Name: name,
		Labels: map[string]string{
			"atlas.mongodb.com/project-id": entry.GetGroupId(),
		},
	}

	if comment := entry.GetComment(); comment != "" {
		metadata.Labels["atlas.mongodb.com/comment"] = comment
	}

	spec := types.NetworkAccessSpec{
		IPAddress:        entry.GetIpAddress(),
		CIDR:             entry.GetCidrBlock(),
		AWSSecurityGroup: entry.GetAwsSecurityGroup(),
		Comment:          entry.GetComment(),
	}

	if deleteAfter := entry.DeleteAfterDate; deleteAfter != nil {
		spec.DeleteAfterDate = deleteAfter.Format(time.RFC3339)
	}

	return types.NetworkAccessManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindNetworkAccess,
		Metadata:   metadata,
		Spec:       spec,
		Status: &types.ResourceStatusInfo{
			Phase:      types.StatusReady,
			Message:    "Network access entry is active",
			LastUpdate: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// convertClusterStatus converts Atlas cluster state to our status type
func convertClusterStatus(stateName string) types.ResourceStatus {
	switch strings.ToUpper(stateName) {
	case "IDLE":
		return types.StatusReady
	case "CREATING":
		return types.StatusCreating
	case "UPDATING":
		return types.StatusUpdating
	case "DELETING":
		return types.StatusDeleting
	case "DELETED":
		return types.StatusDeleting
	default:
		return types.StatusUnknown
	}
}

// extractClusterTier extracts the tier (instance size) from cluster replication specs
func extractClusterTier(cluster admin.ClusterDescription20240805) string {
	if cluster.ReplicationSpecs == nil || len(*cluster.ReplicationSpecs) == 0 {
		return "N/A"
	}

	// Get first replication spec
	repSpec := (*cluster.ReplicationSpecs)[0]
	if repSpec.RegionConfigs == nil || len(*repSpec.RegionConfigs) == 0 {
		return "N/A"
	}

	// Get first region config
	regionConfig := (*repSpec.RegionConfigs)[0]

	// Try electable specs first (most common)
	if regionConfig.ElectableSpecs != nil {
		if instanceSize := regionConfig.ElectableSpecs.GetInstanceSize(); instanceSize != "" {
			return instanceSize
		}
	}

	// Try analytics specs
	if regionConfig.AnalyticsSpecs != nil {
		if instanceSize := regionConfig.AnalyticsSpecs.GetInstanceSize(); instanceSize != "" {
			return instanceSize
		}
	}

	// Try read-only specs
	if regionConfig.ReadOnlySpecs != nil {
		if instanceSize := regionConfig.ReadOnlySpecs.GetInstanceSize(); instanceSize != "" {
			return instanceSize
		}
	}

	return "N/A"
}

// extractClusterProvider extracts the cloud provider from cluster replication specs
func extractClusterProvider(cluster admin.ClusterDescription20240805) string {
	if cluster.ReplicationSpecs == nil || len(*cluster.ReplicationSpecs) == 0 {
		return "N/A"
	}

	// Get first replication spec
	repSpec := (*cluster.ReplicationSpecs)[0]
	if repSpec.RegionConfigs == nil || len(*repSpec.RegionConfigs) == 0 {
		return "N/A"
	}

	// Get first region config
	regionConfig := (*repSpec.RegionConfigs)[0]

	// Get provider name
	if provider := regionConfig.GetProviderName(); provider != "" {
		return provider
	}

	// Fall back to backing provider name for tenant clusters
	if backingProvider := regionConfig.GetBackingProviderName(); backingProvider != "" {
		return backingProvider
	}

	return "N/A"
}

// extractClusterRegion extracts the region from cluster replication specs
func extractClusterRegion(cluster admin.ClusterDescription20240805) string {
	if cluster.ReplicationSpecs == nil || len(*cluster.ReplicationSpecs) == 0 {
		return "N/A"
	}

	// Get first replication spec
	repSpec := (*cluster.ReplicationSpecs)[0]
	if repSpec.RegionConfigs == nil || len(*repSpec.RegionConfigs) == 0 {
		return "N/A"
	}

	// Get first region config
	regionConfig := (*repSpec.RegionConfigs)[0]

	// Get region name
	if region := regionConfig.GetRegionName(); region != "" {
		return region
	}

	return "N/A"
}

// convertDatabaseRoleToManifest would convert a MongoDB custom role to our DatabaseRoleManifest type
// NOTE: Custom database roles are not available through Atlas API and require direct MongoDB connection
// This function serves as a placeholder for future implementation that would need:
// 1. MongoDB connection string resolution
// 2. Database authentication with appropriate privileges  
// 3. MongoDB rolesInfo command execution to fetch role details
func (d *AtlasStateDiscovery) convertDatabaseRoleToManifest(roleName, databaseName, projectName string) types.DatabaseRoleManifest {
	// Since we cannot fetch custom database roles through Atlas API,
	// we can only create a minimal manifest based on the known role name and database
	metadata := types.ResourceMetadata{
		Name: roleName,
		Labels: map[string]string{
			"atlas.mongodb.com/role-name":    roleName,
			"atlas.mongodb.com/database":     databaseName,
			"atlas.mongodb.com/project-name": projectName,
		},
		Annotations: map[string]string{
			"matlas.mongodb.com/fetch-limitation": "Custom database roles cannot be fetched through Atlas API - requires direct MongoDB connection",
		},
	}

	spec := types.DatabaseRoleSpec{
		RoleName:     roleName,
		DatabaseName: databaseName,
		// Privileges and InheritedRoles cannot be populated without direct MongoDB access
		Privileges:     []types.CustomRolePrivilegeConfig{},
		InheritedRoles: []types.CustomRoleInheritedRoleConfig{},
	}

	return types.DatabaseRoleManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindDatabaseRole,
		Metadata:   metadata,
		Spec:       spec,
		Status: &types.ResourceStatusInfo{
			Phase:      types.StatusUnknown,
			Message:    "Custom database roles cannot be fetched through Atlas API - requires direct MongoDB connection",
			LastUpdate: time.Now().UTC().Format(time.RFC3339),
		},
	}
}
