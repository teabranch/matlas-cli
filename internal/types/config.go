package types

// DeletionPolicy defines how resources should be handled when removed from configuration
type DeletionPolicy string

const (
	DeletionPolicyDelete   DeletionPolicy = "Delete"   // Delete the resource immediately
	DeletionPolicyRetain   DeletionPolicy = "Retain"   // Keep the resource but stop managing it
	DeletionPolicySnapshot DeletionPolicy = "Snapshot" // Take a snapshot before deletion (clusters only)
)

// ResourceMetadata contains common metadata for all resources
type ResourceMetadata struct {
	Name           string            `yaml:"name" json:"name" validate:"required,min=1,max=64,hostname"`
	Labels         map[string]string `yaml:"labels,omitempty" json:"labels,omitempty" validate:"dive,keys,min=1,max=63,endkeys,min=0,max=63"`
	Annotations    map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty" validate:"dive,keys,min=1,max=253,endkeys,min=0,max=512"`
	DeletionPolicy DeletionPolicy    `yaml:"deletionPolicy,omitempty" json:"deletionPolicy,omitempty" validate:"omitempty,oneof=Delete Retain Snapshot"`
	DependsOn      []string          `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty" validate:"dive,min=1,max=64"`
}

// ProjectConfig represents a declarative project configuration
type ProjectConfig struct {
	Name           string                `yaml:"name" json:"name" validate:"required,min=1,max=64"`
	OrganizationID string                `yaml:"organizationId" json:"organizationId" validate:"required,len=24,alphanum"`
	Tags           map[string]string     `yaml:"tags,omitempty" json:"tags,omitempty" validate:"omitempty,dive,keys,min=1,max=255,endkeys,min=1,max=255,max=50"`
	Metadata       ResourceMetadata      `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Clusters       []ClusterConfig       `yaml:"clusters,omitempty" json:"clusters,omitempty" validate:"dive"`
	DatabaseUsers  []DatabaseUserConfig  `yaml:"databaseUsers,omitempty" json:"databaseUsers,omitempty" validate:"dive"`
	NetworkAccess  []NetworkAccessConfig `yaml:"networkAccess,omitempty" json:"networkAccess,omitempty" validate:"dive"`
}

// AutoScalingConfig represents cluster autoscaling configuration
type AutoScalingConfig struct {
	DiskGB  *AutoScalingLimits  `yaml:"diskGB,omitempty" json:"diskGB,omitempty" validate:"omitempty"`
	Compute *ComputeAutoScaling `yaml:"compute,omitempty" json:"compute,omitempty" validate:"omitempty"`
}

// AutoScalingLimits defines min/max values for autoscaling
type AutoScalingLimits struct {
	Enabled   *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MinimumGB *int  `yaml:"minimumGB,omitempty" json:"minimumGB,omitempty" validate:"omitempty,min=1,max=4096"`
	MaximumGB *int  `yaml:"maximumGB,omitempty" json:"maximumGB,omitempty" validate:"omitempty,min=1,max=4096,gtfield=MinimumGB"`
}

// ComputeAutoScaling represents compute-based autoscaling
type ComputeAutoScaling struct {
	Enabled          *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	ScaleDownEnabled *bool  `yaml:"scaleDownEnabled,omitempty" json:"scaleDownEnabled,omitempty"`
	MinInstanceSize  string `yaml:"minInstanceSize,omitempty" json:"minInstanceSize,omitempty" validate:"omitempty,oneof=M2 M5 M10 M20 M30 M40 M50 M60 M80 M140 M200 M300 M400 M700 R40 R50 R60 R80 R200 R300 R400 R700"`
	MaxInstanceSize  string `yaml:"maxInstanceSize,omitempty" json:"maxInstanceSize,omitempty" validate:"omitempty,oneof=M2 M5 M10 M20 M30 M40 M50 M60 M80 M140 M200 M300 M400 M700 R40 R50 R60 R80 R200 R300 R400 R700"`
}

// EncryptionConfig represents cluster encryption settings
type EncryptionConfig struct {
	EncryptionAtRestProvider string                `yaml:"encryptionAtRestProvider,omitempty" json:"encryptionAtRestProvider,omitempty" validate:"omitempty,oneof=AWS AZURE GCP NONE"`
	AWSKMSConfig             *AWSKMSConfig         `yaml:"awsKms,omitempty" json:"awsKms,omitempty" validate:"omitempty"`
	AzureKeyVaultConfig      *AzureKeyVaultConfig  `yaml:"azureKeyVault,omitempty" json:"azureKeyVault,omitempty" validate:"omitempty"`
	GoogleCloudKMSConfig     *GoogleCloudKMSConfig `yaml:"googleCloudKms,omitempty" json:"googleCloudKms,omitempty" validate:"omitempty"`
}

// AWSKMSConfig represents AWS KMS encryption configuration
type AWSKMSConfig struct {
	Enabled             *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	AccessKeyID         string `yaml:"accessKeyId,omitempty" json:"accessKeyId,omitempty" validate:"omitempty,min=16,max=128"`
	SecretAccessKey     string `yaml:"secretAccessKey,omitempty" json:"secretAccessKey,omitempty" validate:"omitempty,min=40"`
	CustomerMasterKeyID string `yaml:"customerMasterKeyId,omitempty" json:"customerMasterKeyId,omitempty" validate:"omitempty,min=1,max=2048"`
	Region              string `yaml:"region,omitempty" json:"region,omitempty" validate:"omitempty,min=9,max=20"`
	RoleID              string `yaml:"roleId,omitempty" json:"roleId,omitempty" validate:"omitempty,len=24,alphanum"`
}

// AzureKeyVaultConfig represents Azure Key Vault encryption configuration
type AzureKeyVaultConfig struct {
	Enabled           *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	ClientID          string `yaml:"clientId,omitempty" json:"clientId,omitempty" validate:"omitempty,uuid4"`
	AzureEnvironment  string `yaml:"azureEnvironment,omitempty" json:"azureEnvironment,omitempty" validate:"omitempty,oneof=AZURE AZURE_CHINA AZURE_GERMANY AZURE_US_GOVERNMENT"`
	SubscriptionID    string `yaml:"subscriptionId,omitempty" json:"subscriptionId,omitempty" validate:"omitempty,uuid4"`
	ResourceGroupName string `yaml:"resourceGroupName,omitempty" json:"resourceGroupName,omitempty" validate:"omitempty,min=1,max=90"`
	KeyVaultName      string `yaml:"keyVaultName,omitempty" json:"keyVaultName,omitempty" validate:"omitempty,min=3,max=24,alphanum"`
	KeyIdentifier     string `yaml:"keyIdentifier,omitempty" json:"keyIdentifier,omitempty" validate:"omitempty,url"`
	Secret            string `yaml:"secret,omitempty" json:"secret,omitempty" validate:"omitempty,min=1"`
	TenantID          string `yaml:"tenantId,omitempty" json:"tenantId,omitempty" validate:"omitempty,uuid4"`
}

// GoogleCloudKMSConfig represents Google Cloud KMS encryption configuration
type GoogleCloudKMSConfig struct {
	Enabled              *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	ServiceAccountKey    string `yaml:"serviceAccountKey,omitempty" json:"serviceAccountKey,omitempty" validate:"omitempty,json"`
	KeyVersionResourceID string `yaml:"keyVersionResourceId,omitempty" json:"keyVersionResourceId,omitempty" validate:"omitempty,min=1,max=2048"`
}

// BiConnectorConfig represents BI Connector configuration
type BiConnectorConfig struct {
	Enabled        *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	ReadPreference string `yaml:"readPreference,omitempty" json:"readPreference,omitempty" validate:"omitempty,oneof=primary primaryPreferred secondary secondaryPreferred nearest"`
}

// ClusterConfig represents a declarative cluster configuration
type ClusterConfig struct {
	Metadata         ResourceMetadata   `yaml:"metadata" json:"metadata" validate:"required"`
	Tags             map[string]string  `yaml:"tags,omitempty" json:"tags,omitempty" validate:"omitempty,dive,keys,min=1,max=255,endkeys,min=1,max=255,max=50"`
	Provider         string             `yaml:"provider" json:"provider" validate:"required,oneof=AWS GCP AZURE TENANT"`
	Region           string             `yaml:"region" json:"region" validate:"required,min=1,max=50"`
	InstanceSize     string             `yaml:"instanceSize" json:"instanceSize" validate:"required,oneof=M0 M2 M5 M10 M20 M30 M40 M50 M60 M80 M140 M200 M300 M400 M700 R40 R50 R60 R80 R200 R300 R400 R700"`
	DiskSizeGB       *float64           `yaml:"diskSizeGB,omitempty" json:"diskSizeGB,omitempty" validate:"omitempty,min=1,max=4096"`
	BackupEnabled    *bool              `yaml:"backupEnabled,omitempty" json:"backupEnabled,omitempty"`
	TierType         string             `yaml:"tierType,omitempty" json:"tierType,omitempty" validate:"omitempty,oneof=dedicated shared"`
	MongoDBVersion   string             `yaml:"mongodbVersion,omitempty" json:"mongodbVersion,omitempty" validate:"omitempty,min=3,max=10"`
	ClusterType      string             `yaml:"clusterType,omitempty" json:"clusterType,omitempty" validate:"omitempty,oneof=REPLICASET SHARDED GEOSHARDED"`
	ReplicationSpecs []ReplicationSpec  `yaml:"replicationSpecs,omitempty" json:"replicationSpecs,omitempty" validate:"dive"`
	AutoScaling      *AutoScalingConfig `yaml:"autoScaling,omitempty" json:"autoScaling,omitempty" validate:"omitempty"`
	Encryption       *EncryptionConfig  `yaml:"encryption,omitempty" json:"encryption,omitempty" validate:"omitempty"`
	BiConnector      *BiConnectorConfig `yaml:"biConnector,omitempty" json:"biConnector,omitempty" validate:"omitempty"`
	DependsOn        []string           `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty" validate:"dive,min=1,max=64"`
}

// ReplicationSpec represents cluster replication specification
type ReplicationSpec struct {
	ID            string         `yaml:"id,omitempty" json:"id,omitempty" validate:"omitempty,min=1,max=64"`
	NumShards     *int           `yaml:"numShards,omitempty" json:"numShards,omitempty" validate:"omitempty,min=1,max=50"`
	ZoneName      string         `yaml:"zoneName,omitempty" json:"zoneName,omitempty" validate:"omitempty,min=1,max=64"`
	RegionConfigs []RegionConfig `yaml:"regionConfigs,omitempty" json:"regionConfigs,omitempty" validate:"required,min=1,dive"`
}

// RegionConfig represents region-specific cluster configuration
type RegionConfig struct {
	RegionName     string `yaml:"regionName" json:"regionName" validate:"required,min=1,max=50"`
	Priority       *int   `yaml:"priority,omitempty" json:"priority,omitempty" validate:"omitempty,min=0,max=7"`
	ProviderName   string `yaml:"providerName" json:"providerName" validate:"required,oneof=AWS GCP AZURE TENANT"`
	ElectableNodes *int   `yaml:"electableNodes,omitempty" json:"electableNodes,omitempty" validate:"omitempty,min=0,max=50"`
	ReadOnlyNodes  *int   `yaml:"readOnlyNodes,omitempty" json:"readOnlyNodes,omitempty" validate:"omitempty,min=0,max=50"`
	AnalyticsNodes *int   `yaml:"analyticsNodes,omitempty" json:"analyticsNodes,omitempty" validate:"omitempty,min=0,max=50"`
}

// DatabaseUserConfig represents a declarative database user configuration
type DatabaseUserConfig struct {
	Metadata     ResourceMetadata     `yaml:"metadata" json:"metadata" validate:"required"`
	Username     string               `yaml:"username" json:"username" validate:"required,min=1,max=1024"`
	Password     string               `yaml:"password,omitempty" json:"password,omitempty" validate:"omitempty,min=8,max=256"`
	Roles        []DatabaseRoleConfig `yaml:"roles" json:"roles" validate:"required,min=1,dive"`
	AuthDatabase string               `yaml:"authDatabase,omitempty" json:"authDatabase,omitempty" validate:"omitempty,min=1,max=63"`
	Scopes       []UserScopeConfig    `yaml:"scopes,omitempty" json:"scopes,omitempty" validate:"dive"`
	DependsOn    []string             `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty" validate:"dive,min=1,max=64"`
}

// DatabaseRoleConfig represents a database role
type DatabaseRoleConfig struct {
	RoleName       string `yaml:"roleName" json:"roleName" validate:"required,min=1,max=64"`
	DatabaseName   string `yaml:"databaseName" json:"databaseName" validate:"required,min=1,max=63"`
	CollectionName string `yaml:"collectionName,omitempty" json:"collectionName,omitempty" validate:"omitempty,min=1,max=127"`
}

// UserScopeConfig represents database user scoping (for cluster-specific access)
type UserScopeConfig struct {
	Name string `yaml:"name" json:"name" validate:"required,min=1,max=64"`
	Type string `yaml:"type" json:"type" validate:"required,oneof=CLUSTER DATA_LAKE"`
}

// CustomDatabaseRoleConfig represents a custom MongoDB role configuration
type CustomDatabaseRoleConfig struct {
	Metadata        ResourceMetadata                   `yaml:"metadata" json:"metadata" validate:"required"`
	RoleName        string                             `yaml:"roleName" json:"roleName" validate:"required,min=1,max=64"`
	DatabaseName    string                             `yaml:"databaseName" json:"databaseName" validate:"required,min=1,max=63"`
	Privileges      []CustomRolePrivilegeConfig        `yaml:"privileges,omitempty" json:"privileges,omitempty" validate:"dive"`
	InheritedRoles  []CustomRoleInheritedRoleConfig    `yaml:"inheritedRoles,omitempty" json:"inheritedRoles,omitempty" validate:"dive"`
	DependsOn       []string                           `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty" validate:"dive,min=1,max=64"`
}

// CustomRolePrivilegeConfig represents a privilege within a custom role
type CustomRolePrivilegeConfig struct {
	Actions  []string                       `yaml:"actions" json:"actions" validate:"required,min=1,dive,min=1"`
	Resource CustomRoleResourceConfig       `yaml:"resource" json:"resource" validate:"required"`
}

// CustomRoleResourceConfig represents a resource that a privilege applies to
type CustomRoleResourceConfig struct {
	Database   string `yaml:"database" json:"database" validate:"required,min=1,max=63"`
	Collection string `yaml:"collection,omitempty" json:"collection,omitempty" validate:"omitempty,min=1,max=127"`
}

// CustomRoleInheritedRoleConfig represents a role that this custom role inherits from
type CustomRoleInheritedRoleConfig struct {
	RoleName     string `yaml:"roleName" json:"roleName" validate:"required,min=1,max=64"`
	DatabaseName string `yaml:"databaseName" json:"databaseName" validate:"required,min=1,max=63"`
}

// NetworkAccessConfig represents a declarative network access configuration
type NetworkAccessConfig struct {
	Metadata         ResourceMetadata `yaml:"metadata" json:"metadata" validate:"required"`
	IPAddress        string           `yaml:"ipAddress,omitempty" json:"ipAddress,omitempty" validate:"omitempty,ip"`
	CIDR             string           `yaml:"cidr,omitempty" json:"cidr,omitempty" validate:"omitempty,cidr"`
	AWSSecurityGroup string           `yaml:"awsSecurityGroup,omitempty" json:"awsSecurityGroup,omitempty" validate:"omitempty,min=11,max=20"`
	Comment          string           `yaml:"comment,omitempty" json:"comment,omitempty" validate:"omitempty,max=80"`
	DeleteAfterDate  string           `yaml:"deleteAfterDate,omitempty" json:"deleteAfterDate,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	DependsOn        []string         `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty" validate:"dive,min=1,max=64"`
}

// ApplyConfig represents the root configuration for declarative operations
type ApplyConfig struct {
	APIVersion string         `yaml:"apiVersion" json:"apiVersion" validate:"required,oneof=matlas.mongodb.com/v1 matlas.mongodb.com/v1alpha1 matlas.mongodb.com/v1beta1"`
	Kind       string         `yaml:"kind" json:"kind" validate:"required,oneof=Project ClusterSet DatabaseUserSet NetworkAccessSet"`
	Metadata   MetadataConfig `yaml:"metadata" json:"metadata" validate:"required"`
	Spec       ProjectConfig  `yaml:"spec" json:"spec" validate:"required"`
}

// MetadataConfig represents metadata for configuration resources
type MetadataConfig struct {
	Name        string            `yaml:"name" json:"name" validate:"required,min=1,max=64,hostname"`
	Namespace   string            `yaml:"namespace,omitempty" json:"namespace,omitempty" validate:"omitempty,min=1,max=63,hostname"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty" validate:"dive,keys,min=1,max=63,endkeys,min=0,max=63"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty" validate:"dive,keys,min=1,max=253,endkeys,min=0,max=512"`
}
