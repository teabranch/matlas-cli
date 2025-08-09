package discover

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

// TestValidateResourceDiscoveryOptions tests the validation function
func TestValidateResourceDiscoveryOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        *DiscoverOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid resource-specific discovery",
			opts: &DiscoverOptions{
				ResourceType: "cluster",
				ResourceName: "my-cluster",
			},
			expectError: false,
		},
		{
			name: "resource-type without resource-name",
			opts: &DiscoverOptions{
				ResourceType: "cluster",
			},
			expectError: true,
			errorMsg:    "--resource-name is required when --resource-type is specified",
		},
		{
			name: "resource-name without resource-type",
			opts: &DiscoverOptions{
				ResourceName: "my-cluster",
			},
			expectError: true,
			errorMsg:    "--resource-type is required when --resource-name is specified",
		},
		{
			name: "invalid resource type",
			opts: &DiscoverOptions{
				ResourceType: "invalid",
				ResourceName: "my-resource",
			},
			expectError: true,
			errorMsg:    "invalid resource type 'invalid'",
		},
		{
			name: "resource-specific with include types",
			opts: &DiscoverOptions{
				ResourceType: "cluster",
				ResourceName: "my-cluster",
				IncludeTypes: []string{"clusters"},
			},
			expectError: true,
			errorMsg:    "--include cannot be used with resource-specific discovery",
		},
		{
			name: "resource-specific with exclude types",
			opts: &DiscoverOptions{
				ResourceType: "cluster",
				ResourceName: "my-cluster",
				ExcludeTypes: []string{"users"},
			},
			expectError: true,
			errorMsg:    "--exclude cannot be used with resource-specific discovery",
		},
		{
			name: "resource-specific with include databases",
			opts: &DiscoverOptions{
				ResourceType:     "cluster",
				ResourceName:     "my-cluster",
				IncludeDatabases: true,
			},
			expectError: true,
			errorMsg:    "--include-databases cannot be used with resource-specific discovery",
		},
		{
			name: "normal discovery without resource flags",
			opts: &DiscoverOptions{
				IncludeTypes: []string{"clusters", "users"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResourceDiscoveryOptions(tt.opts)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFindSpecificResource tests the resource finding function
func TestFindSpecificResource(t *testing.T) {
	// Create mock project state
	projectState := &apply.ProjectState{
		Project: &types.ProjectManifest{
			APIVersion: types.APIVersionV1,
			Kind:       types.KindProject,
			Metadata: types.ResourceMetadata{
				Name: "test-project",
			},
		},
		Clusters: []types.ClusterManifest{
			{
				APIVersion: types.APIVersionV1,
				Kind:       types.KindCluster,
				Metadata: types.ResourceMetadata{
					Name: "cluster-1",
				},
				Spec: types.ClusterSpec{
					Provider:     "AWS",
					Region:       "US_EAST_1",
					InstanceSize: "M30",
				},
			},
			{
				APIVersion: types.APIVersionV1,
				Kind:       types.KindCluster,
				Metadata: types.ResourceMetadata{
					Name: "cluster-2",
				},
				Spec: types.ClusterSpec{
					Provider:     "GCP",
					Region:       "us-central1",
					InstanceSize: "M20",
				},
			},
		},
		DatabaseUsers: []types.DatabaseUserManifest{
			{
				APIVersion: types.APIVersionV1,
				Kind:       types.KindDatabaseUser,
				Metadata: types.ResourceMetadata{
					Name: "app-user",
				},
				Spec: types.DatabaseUserSpec{
					Username: "appuser",
					Password: "secret123",
				},
			},
		},
		NetworkAccess: []types.NetworkAccessManifest{
			{
				APIVersion: types.APIVersionV1,
				Kind:       types.KindNetworkAccess,
				Metadata: types.ResourceMetadata{
					Name: "office-access",
				},
				Spec: types.NetworkAccessSpec{
					CIDR:    "203.0.113.0/24",
					Comment: "Office network",
				},
			},
		},
	}

	tests := []struct {
		name         string
		resourceType string
		resourceName string
		expectError  bool
		expectType   interface{}
	}{
		{
			name:         "find existing cluster",
			resourceType: "cluster",
			resourceName: "cluster-1",
			expectError:  false,
			expectType:   types.ClusterManifest{},
		},
		{
			name:         "find existing database user",
			resourceType: "user",
			resourceName: "app-user",
			expectError:  false,
			expectType:   &types.DatabaseUserManifest{},
		},
		{
			name:         "find existing network access",
			resourceType: "network",
			resourceName: "office-access",
			expectError:  false,
			expectType:   types.NetworkAccessManifest{},
		},
		{
			name:         "find existing project",
			resourceType: "project",
			resourceName: "test-project",
			expectError:  false,
			expectType:   &types.ProjectManifest{},
		},
		{
			name:         "cluster not found",
			resourceType: "cluster",
			resourceName: "nonexistent",
			expectError:  true,
		},
		{
			name:         "user not found",
			resourceType: "user",
			resourceName: "nonexistent",
			expectError:  true,
		},
		{
			name:         "network not found",
			resourceType: "network",
			resourceName: "nonexistent",
			expectError:  true,
		},
		{
			name:         "unsupported resource type",
			resourceType: "invalid",
			resourceName: "some-resource",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := findSpecificResource(projectState, tt.resourceType, tt.resourceName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resource)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resource)

				// Verify the returned resource type matches expected
				switch tt.expectType.(type) {
				case types.ClusterManifest:
					_, ok := resource.(types.ClusterManifest)
					assert.True(t, ok, "Expected ClusterManifest")
				case *types.DatabaseUserManifest:
					_, ok := resource.(*types.DatabaseUserManifest)
					assert.True(t, ok, "Expected DatabaseUserManifest pointer")
				case types.NetworkAccessManifest:
					_, ok := resource.(types.NetworkAccessManifest)
					assert.True(t, ok, "Expected NetworkAccessManifest")
				case *types.ProjectManifest:
					_, ok := resource.(*types.ProjectManifest)
					assert.True(t, ok, "Expected ProjectManifest pointer")
				}
			}
		})
	}
}

// TestMaskResourceSecrets tests the masking function for individual resources
func TestMaskResourceSecrets(t *testing.T) {
	tests := []struct {
		name     string
		resource interface{}
		validate func(t *testing.T, resource interface{})
	}{

		{
			name: "mask database user password (pointer)",
			resource: &types.DatabaseUserManifest{
				Spec: types.DatabaseUserSpec{
					Username: "testuser",
					Password: "secret456",
				},
			},
			validate: func(t *testing.T, resource interface{}) {
				user := resource.(*types.DatabaseUserManifest)
				assert.Equal(t, "***MASKED***", user.Spec.Password)
			},
		},
		{
			name: "cluster should not be modified",
			resource: types.ClusterManifest{
				Spec: types.ClusterSpec{
					Provider: "AWS",
					Region:   "US_EAST_1",
				},
			},
			validate: func(t *testing.T, resource interface{}) {
				cluster := resource.(types.ClusterManifest)
				assert.Equal(t, "AWS", cluster.Spec.Provider)
				assert.Equal(t, "US_EAST_1", cluster.Spec.Region)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maskResourceSecrets(tt.resource)
			tt.validate(t, tt.resource)
		})
	}
}
