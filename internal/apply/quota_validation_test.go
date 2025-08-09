package apply

import (
	"context"
	"strings"
	"testing"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestAtlasQuotaValidator_ValidateProjectQuotas(t *testing.T) {
	validator := NewAtlasQuotaValidator()
	ctx := context.Background()
	orgID := "507f1f77bcf86cd799439011"

	tests := []struct {
		name      string
		config    types.ProjectConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid project configuration",
			config: types.ProjectConfig{
				Name:           "test-project",
				OrganizationID: orgID,
				Clusters: []types.ClusterConfig{
					{
						Metadata:     types.ResourceMetadata{Name: "cluster1"},
						Provider:     "AWS",
						Region:       "US_EAST_1",
						InstanceSize: "M10",
					},
				},
				DatabaseUsers: []types.DatabaseUserConfig{
					{
						Metadata: types.ResourceMetadata{Name: "user1"},
						Username: "testuser",
						Roles: []types.DatabaseRoleConfig{
							{RoleName: "read", DatabaseName: "testdb"},
						},
					},
				},
				NetworkAccess: []types.NetworkAccessConfig{
					{
						Metadata:  types.ResourceMetadata{Name: "access1"},
						IPAddress: "192.168.1.1",
					},
				},
			},
			wantError: false,
		},
		{
			name: "too many clusters",
			config: types.ProjectConfig{
				Name:           "test-project",
				OrganizationID: orgID,
				Clusters:       make([]types.ClusterConfig, 26), // Exceeds limit of 25
			},
			wantError: true,
			errorMsg:  "cannot have more than 25 clusters",
		},
		{
			name: "too many database users",
			config: types.ProjectConfig{
				Name:           "test-project",
				OrganizationID: orgID,
				DatabaseUsers:  make([]types.DatabaseUserConfig, 101), // Exceeds limit of 100
			},
			wantError: true,
			errorMsg:  "cannot have more than 100 database users",
		},
		{
			name: "too many network access rules",
			config: types.ProjectConfig{
				Name:           "test-project",
				OrganizationID: orgID,
				NetworkAccess:  make([]types.NetworkAccessConfig, 201), // Exceeds limit of 200
			},
			wantError: true,
			errorMsg:  "cannot have more than 200 network access rules",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateProjectQuotas(ctx, orgID, tt.config)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAtlasQuotaValidator_ValidateClusterQuotas(t *testing.T) {
	validator := NewAtlasQuotaValidator()
	ctx := context.Background()
	orgID := "507f1f77bcf86cd799439011"

	tests := []struct {
		name      string
		clusters  []types.ClusterConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid cluster configuration",
			clusters: []types.ClusterConfig{
				{
					Metadata:     types.ResourceMetadata{Name: "cluster1"},
					Provider:     "AWS",
					Region:       "US_EAST_1",
					InstanceSize: "M10",
				},
			},
			wantError: false,
		},
		{
			name: "invalid provider",
			clusters: []types.ClusterConfig{
				{
					Metadata:     types.ResourceMetadata{Name: "cluster1"},
					Provider:     "INVALID",
					Region:       "US_EAST_1",
					InstanceSize: "M10",
				},
			},
			wantError: true,
			errorMsg:  "Provider INVALID is not allowed",
		},
		{
			name: "invalid region",
			clusters: []types.ClusterConfig{
				{
					Metadata:     types.ResourceMetadata{Name: "cluster1"},
					Provider:     "AWS",
					Region:       "invalid-region",
					InstanceSize: "M10",
				},
			},
			wantError: true,
			errorMsg:  "Region invalid-region is not allowed",
		},
		{
			name: "instance size too large",
			clusters: []types.ClusterConfig{
				{
					Metadata:     types.ResourceMetadata{Name: "cluster1"},
					Provider:     "AWS",
					Region:       "US_EAST_1",
					InstanceSize: "M1000", // Invalid size
				},
			},
			wantError: true,
			errorMsg:  "Instance size M1000 exceeds maximum",
		},
		{
			name: "too many nodes in replication spec",
			clusters: []types.ClusterConfig{
				{
					Metadata:     types.ResourceMetadata{Name: "cluster1"},
					Provider:     "AWS",
					Region:       "US_EAST_1",
					InstanceSize: "M10",
					ReplicationSpecs: []types.ReplicationSpec{
						{
							ID: "spec1",
							RegionConfigs: []types.RegionConfig{
								{
									RegionName:     "US_EAST_1",
									ProviderName:   "AWS",
									ElectableNodes: intPtr(51), // Exceeds limit
								},
							},
						},
					},
				},
			},
			wantError: true,
			errorMsg:  "Total nodes (51) in replication spec spec1 exceeds maximum allowed (50)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateClusterQuotas(ctx, orgID, tt.clusters)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAtlasQuotaValidator_ValidateDatabaseUserQuotas(t *testing.T) {
	validator := NewAtlasQuotaValidator()
	ctx := context.Background()
	orgID := "507f1f77bcf86cd799439011"

	tests := []struct {
		name      string
		users     []types.DatabaseUserConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid database users",
			users: []types.DatabaseUserConfig{
				{
					Metadata: types.ResourceMetadata{Name: "user1"},
					Username: "user1",
					Password: "password123",
					Roles: []types.DatabaseRoleConfig{
						{RoleName: "read", DatabaseName: "testdb"},
					},
				},
				{
					Metadata: types.ResourceMetadata{Name: "user2"},
					Username: "user2",
					Password: "password456",
					Roles: []types.DatabaseRoleConfig{
						{RoleName: "readWrite", DatabaseName: "testdb"},
					},
				},
			},
			wantError: false,
		},
		{
			name: "duplicate usernames",
			users: []types.DatabaseUserConfig{
				{
					Metadata: types.ResourceMetadata{Name: "user1"},
					Username: "duplicate",
					Roles: []types.DatabaseRoleConfig{
						{RoleName: "read", DatabaseName: "testdb"},
					},
				},
				{
					Metadata: types.ResourceMetadata{Name: "user2"},
					Username: "duplicate", // Same username
					Roles: []types.DatabaseRoleConfig{
						{RoleName: "readWrite", DatabaseName: "testdb"},
					},
				},
			},
			wantError: true,
			errorMsg:  "Duplicate database username: duplicate",
		},
		{
			name: "password too short",
			users: []types.DatabaseUserConfig{
				{
					Metadata: types.ResourceMetadata{Name: "user1"},
					Username: "user1",
					Password: "short", // Too short
					Roles: []types.DatabaseRoleConfig{
						{RoleName: "read", DatabaseName: "testdb"},
					},
				},
			},
			wantError: true,
			errorMsg:  "Password for user user1 must be at least 8 characters",
		},
		{
			name: "too many roles",
			users: []types.DatabaseUserConfig{
				{
					Metadata: types.ResourceMetadata{Name: "user1"},
					Username: "user1",
					Roles:    make([]types.DatabaseRoleConfig, 21), // Exceeds limit of 20
				},
			},
			wantError: true,
			errorMsg:  "User user1 cannot have more than 20 roles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDatabaseUserQuotas(ctx, orgID, tt.users)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAtlasQuotaValidator_ValidateNetworkAccessQuotas(t *testing.T) {
	validator := NewAtlasQuotaValidator()
	ctx := context.Background()
	orgID := "507f1f77bcf86cd799439011"

	tests := []struct {
		name      string
		access    []types.NetworkAccessConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid network access rules",
			access: []types.NetworkAccessConfig{
				{
					Metadata:  types.ResourceMetadata{Name: "rule1"},
					IPAddress: "192.168.1.1",
					Comment:   "Office IP",
				},
				{
					Metadata: types.ResourceMetadata{Name: "rule2"},
					CIDR:     "10.0.0.0/16",
					Comment:  "Development network",
				},
			},
			wantError: false,
		},
		{
			name: "duplicate CIDR ranges",
			access: []types.NetworkAccessConfig{
				{
					Metadata: types.ResourceMetadata{Name: "rule1"},
					CIDR:     "192.168.1.0/24",
				},
				{
					Metadata: types.ResourceMetadata{Name: "rule2"},
					CIDR:     "192.168.1.0/24", // Duplicate
				},
			},
			wantError: true,
			errorMsg:  "Duplicate CIDR range: 192.168.1.0/24",
		},
		{
			name: "comment too long",
			access: []types.NetworkAccessConfig{
				{
					Metadata:  types.ResourceMetadata{Name: "rule1"},
					IPAddress: "192.168.1.1",
					Comment:   "This comment is way too long and exceeds the maximum allowed length of 80 characters for network access rule comments",
				},
			},
			wantError: true,
			errorMsg:  "Comment for network access rule rule1 is too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateNetworkAccessQuotas(ctx, orgID, tt.access)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateConfiguration(t *testing.T) {
	validator := NewAtlasQuotaValidator()
	ctx := context.Background()
	orgID := "507f1f77bcf86cd799439011"

	tests := []struct {
		name      string
		config    types.ProjectConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid complete configuration",
			config: types.ProjectConfig{
				Name:           "test-project",
				OrganizationID: orgID,
				Clusters: []types.ClusterConfig{
					{
						Metadata:     types.ResourceMetadata{Name: "cluster1"},
						Provider:     "AWS",
						Region:       "US_EAST_1",
						InstanceSize: "M10",
					},
				},
				DatabaseUsers: []types.DatabaseUserConfig{
					{
						Metadata: types.ResourceMetadata{Name: "user1"},
						Username: "testuser",
						Password: "password123",
						Roles: []types.DatabaseRoleConfig{
							{RoleName: "read", DatabaseName: "testdb"},
						},
					},
				},
				NetworkAccess: []types.NetworkAccessConfig{
					{
						Metadata:  types.ResourceMetadata{Name: "access1"},
						IPAddress: "192.168.1.1",
						Comment:   "Office access",
					},
				},
			},
			wantError: false,
		},
		{
			name: "invalid cluster configuration",
			config: types.ProjectConfig{
				Name:           "test-project",
				OrganizationID: orgID,
				Clusters: []types.ClusterConfig{
					{
						Metadata:     types.ResourceMetadata{Name: "cluster1"},
						Provider:     "INVALID",
						Region:       "US_EAST_1",
						InstanceSize: "M10",
					},
				},
			},
			wantError: true,
			errorMsg:  "cluster quota validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfiguration(ctx, validator, orgID, tt.config)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestInstanceSizeValidation(t *testing.T) {
	validator := NewAtlasQuotaValidator()

	tests := []struct {
		name         string
		instanceSize string
		maxSize      string
		expected     bool
	}{
		{"M10 within M700 limit", "M10", "M700", true},
		{"M700 at M700 limit", "M700", "M700", true},
		{"M800 exceeds M700 limit", "M800", "M700", false}, // Invalid size should return false
		{"R50 within M700 limit", "R50", "M700", true},
		{"Invalid size", "INVALID", "M700", false},
		{"No max limit", "M10", "UNKNOWN", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isInstanceSizeAllowed(tt.instanceSize, tt.maxSize)
			if result != tt.expected {
				t.Errorf("isInstanceSizeAllowed(%s, %s) = %v, want %v", tt.instanceSize, tt.maxSize, result, tt.expected)
			}
		})
	}
}

func TestQuotaValidationError(t *testing.T) {
	err := QuotaValidationError{
		ResourceType: "Cluster",
		Current:      5,
		Requested:    10,
		Limit:        8,
		Message:      "Too many clusters requested",
	}

	if err.Error() != "Too many clusters requested" {
		t.Errorf("Error() = %q, want %q", err.Error(), "Too many clusters requested")
	}
}

// Helper functions

func intPtr(i int) *int {
	return &i
}
