package apply

import (
	"testing"
	"time"

	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teabranch/matlas-cli/internal/types"
)

func TestBuildAtlasClusterFromConfig_BasicAndTagsEncryptionBiConnector(t *testing.T) {
	diskSize := 20.0
	backup := true
	biEnabled := true
	cfg := &types.ClusterConfig{
		Metadata:       types.ResourceMetadata{Name: "test-cluster"},
		Provider:       "AWS",
		Region:         "US_EAST_1",
		InstanceSize:   "M10",
		DiskSizeGB:     &diskSize,
		BackupEnabled:  &backup,
		MongoDBVersion: "7.0",
		ClusterType:    "REPLICASET",
		Tags:           map[string]string{"env": "dev", "owner": "team-x"},
		Encryption:     &types.EncryptionConfig{EncryptionAtRestProvider: "AWS"},
		BiConnector:    &types.BiConnectorConfig{Enabled: &biEnabled, ReadPreference: "secondary"},
	}

	atlasCluster, err := buildAtlasClusterFromConfig(cfg)
	require.NoError(t, err)
	require.NotNil(t, atlasCluster)

	assert.Equal(t, "test-cluster", atlasCluster.GetName())
	assert.Equal(t, "REPLICASET", atlasCluster.GetClusterType())
	assert.Equal(t, "7.0", atlasCluster.GetMongoDBMajorVersion())
	assert.Equal(t, backup, atlasCluster.GetBackupEnabled())

	// Tags mapped
	require.NotNil(t, atlasCluster.Tags)
	tags := *atlasCluster.Tags
	assert.ElementsMatch(t, []admin.ResourceTag{
		{Key: "env", Value: "dev"},
		{Key: "owner", Value: "team-x"},
	}, tags)

	// Encryption provider flag mapped
	assert.Equal(t, "AWS", atlasCluster.GetEncryptionAtRestProvider())

	// BI connector mapped
	require.NotNil(t, atlasCluster.BiConnector)
	assert.Equal(t, true, atlasCluster.BiConnector.GetEnabled())
	assert.Equal(t, "secondary", atlasCluster.BiConnector.GetReadPreference())
}

func TestConvertClusterDiscovery_IncludesTagsInManifest(t *testing.T) {
	d := &AtlasStateDiscovery{}
	name := "test"
	groupID := "507f1f77bcf86cd799439011"
	state := "IDLE"
	cluster := admin.ClusterDescription20240805{
		Name:      &name,
		GroupId:   &groupID,
		StateName: &state,
	}
	// Attach tags
	tags := []admin.ResourceTag{{Key: "env", Value: "dev"}, {Key: "owner", Value: "team-x"}}
	cluster.Tags = &tags

	manifest := d.convertClusterToManifest(&cluster, "my-project")
	require.NotNil(t, manifest.Spec.Tags)
	assert.Equal(t, "dev", manifest.Spec.Tags["env"])
	assert.Equal(t, "team-x", manifest.Spec.Tags["owner"])
}

func TestBuildReplicationSpecsFromConfig_WithRegionAndAutoscaling(t *testing.T) {
	enabled := true
	cfg := &types.ClusterConfig{
		Metadata:     types.ResourceMetadata{Name: "with-autoscaling"},
		Provider:     "AWS",
		Region:       "US_EAST_1",
		InstanceSize: "M30",
		AutoScaling: &types.AutoScalingConfig{
			DiskGB:  &types.AutoScalingLimits{Enabled: &enabled},
			Compute: &types.ComputeAutoScaling{Enabled: &enabled, ScaleDownEnabled: &enabled},
		},
		ReplicationSpecs: []types.ReplicationSpec{
			{
				RegionConfigs: []types.RegionConfig{
					{RegionName: "US_EAST_1", ProviderName: "AWS"},
				},
			},
		},
	}

	specs, err := buildReplicationSpecsFromConfig(cfg)
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.NotNil(t, specs[0].RegionConfigs)
	require.Len(t, *specs[0].RegionConfigs, 1)
	rc := (*specs[0].RegionConfigs)[0]

	// autoscaling should be present on region config
	require.NotNil(t, rc.AutoScaling)
	assert.NotNil(t, rc.AutoScaling.DiskGB)
	assert.Equal(t, true, rc.AutoScaling.DiskGB.GetEnabled())
	assert.NotNil(t, rc.AutoScaling.Compute)
	assert.Equal(t, true, rc.AutoScaling.Compute.GetEnabled())
	assert.Equal(t, true, rc.AutoScaling.Compute.GetScaleDownEnabled())
}

func TestConvertDatabaseUserConfigToAtlas_RolesScopesAndOptionalPassword(t *testing.T) {
	enabled := true
	userCfg := types.DatabaseUserConfig{
		Metadata: types.ResourceMetadata{Name: "user"},
		Username: "alice",
		// No password to verify optional update behavior
		Roles: []types.DatabaseRoleConfig{
			{RoleName: "readWrite", DatabaseName: "db1", CollectionName: "coll"},
		},
		AuthDatabase: "admin",
		Scopes:       []types.UserScopeConfig{{Name: "Cluster0", Type: "CLUSTER"}},
		DependsOn:    []string{},
	}

	atlasUser, err := convertDatabaseUserConfigToAtlas(userCfg)
	require.NoError(t, err)
	require.NotNil(t, atlasUser)
	assert.Equal(t, "alice", atlasUser.GetUsername())
	assert.Equal(t, "admin", atlasUser.GetDatabaseName())
	// Password omitted -> pointer should be nil
	assert.Nil(t, atlasUser.Password)
	require.NotNil(t, atlasUser.Roles)
	roles := *atlasUser.Roles
	require.Len(t, roles, 1)
	assert.Equal(t, "readWrite", roles[0].GetRoleName())
	assert.Equal(t, "db1", roles[0].GetDatabaseName())
	assert.Equal(t, "coll", roles[0].GetCollectionName())
	require.NotNil(t, atlasUser.Scopes)
	scopes := *atlasUser.Scopes
	require.Len(t, scopes, 1)
	assert.Equal(t, "Cluster0", scopes[0].GetName())
	assert.Equal(t, "CLUSTER", scopes[0].GetType())

	_ = enabled // silence unused if not used
}

func TestConvertNetworkAccessManifestToEntry_WithDeleteAfterDate(t *testing.T) {
	date := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second).Format(time.RFC3339)
	manifest := &types.NetworkAccessManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindNetworkAccess,
		Metadata:   types.ResourceMetadata{Name: "entry"},
		Spec: types.NetworkAccessSpec{
			ProjectName:     "proj",
			IPAddress:       "1.2.3.4",
			Comment:         "temp access",
			DeleteAfterDate: date,
		},
	}

	entry, err := convertNetworkAccessManifestToEntry(manifest)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3.4", entry.GetIpAddress())
	assert.Equal(t, "temp access", entry.GetComment())
	require.NotNil(t, entry.DeleteAfterDate)
	assert.Equal(t, date, entry.DeleteAfterDate.UTC().Format(time.RFC3339))
}
