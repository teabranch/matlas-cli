package types

import (
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// AtlasProject represents a MongoDB Atlas project
type AtlasProject = admin.Group

// AtlasCluster represents a MongoDB Atlas cluster
type AtlasCluster = admin.ClusterDescription20240805

// AtlasDatabaseUser represents a MongoDB Atlas database user
type AtlasDatabaseUser = admin.CloudDatabaseUser

// AtlasNetworkAccessEntry represents a network access list entry
type AtlasNetworkAccessEntry = admin.NetworkPermissionEntry

// AtlasOrganization represents a MongoDB Atlas organization
type AtlasOrganization = admin.AtlasOrganization
