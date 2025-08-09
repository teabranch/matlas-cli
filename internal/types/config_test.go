package types

import (
	"strings"
	"testing"
	"time"

	validator "github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// TestProjectConfig_YAMLMarshaling tests YAML marshaling and unmarshaling for ProjectConfig
func TestProjectConfig_YAMLMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		config ProjectConfig
	}{
		{
			name: "basic project config",
			config: ProjectConfig{
				Name:           "test-project",
				OrganizationID: "507f1f77bcf86cd799439011",
				Metadata: ResourceMetadata{
					Name: "test-project",
					Labels: map[string]string{
						"env":  "dev",
						"team": "backend",
					},
					Annotations: map[string]string{
						"description": "Test project for validation",
					},
					DeletionPolicy: DeletionPolicyRetain,
				},
			},
		},
		{
			name: "full project config",
			config: ProjectConfig{
				Name:           "production-project",
				OrganizationID: "507f1f77bcf86cd799439011",
				Metadata: ResourceMetadata{
					Name: "production-project",
					Labels: map[string]string{
						"env":     "prod",
						"team":    "platform",
						"version": "v1.0.0",
					},
					Annotations: map[string]string{
						"description":     "Production project",
						"contact":         "platform-team@company.com",
						"backup-schedule": "daily",
					},
					DeletionPolicy: DeletionPolicySnapshot,
					DependsOn:      []string{"shared-network", "base-security"},
				},
				Clusters: []ClusterConfig{
					{
						Metadata: ResourceMetadata{
							Name: "main-cluster",
						},
						Provider:     "AWS",
						Region:       "US_EAST_1",
						InstanceSize: "M10",
					},
				},
				DatabaseUsers: []DatabaseUserConfig{
					{
						Metadata: ResourceMetadata{
							Name: "app-user",
						},
						Username: "app_user",
						Roles: []DatabaseRoleConfig{
							{
								RoleName:     "readWrite",
								DatabaseName: "app_db",
							},
						},
					},
				},
				NetworkAccess: []NetworkAccessConfig{
					{
						Metadata: ResourceMetadata{
							Name: "office-access",
						},
						IPAddress: "192.168.1.0",
						CIDR:      "192.168.1.0/24",
						Comment:   "Office network access",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling to YAML
			yamlData, err := yaml.Marshal(tt.config)
			if err != nil {
				t.Errorf("Failed to marshal to YAML: %v", err)
				return
			}

			// Test unmarshaling from YAML
			var unmarshaled ProjectConfig
			err = yaml.Unmarshal(yamlData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal from YAML: %v", err)
				return
			}

			// Basic field comparisons
			if unmarshaled.Name != tt.config.Name {
				t.Errorf("Name mismatch: got %s, want %s", unmarshaled.Name, tt.config.Name)
			}
			if unmarshaled.OrganizationID != tt.config.OrganizationID {
				t.Errorf("OrganizationID mismatch: got %s, want %s", unmarshaled.OrganizationID, tt.config.OrganizationID)
			}

			// Test metadata
			if unmarshaled.Metadata.Name != tt.config.Metadata.Name {
				t.Errorf("Metadata.Name mismatch: got %s, want %s", unmarshaled.Metadata.Name, tt.config.Metadata.Name)
			}
			if unmarshaled.Metadata.DeletionPolicy != tt.config.Metadata.DeletionPolicy {
				t.Errorf("Metadata.DeletionPolicy mismatch: got %s, want %s", unmarshaled.Metadata.DeletionPolicy, tt.config.Metadata.DeletionPolicy)
			}

			// Test labels and annotations
			if len(unmarshaled.Metadata.Labels) != len(tt.config.Metadata.Labels) {
				t.Errorf("Labels count mismatch: got %d, want %d", len(unmarshaled.Metadata.Labels), len(tt.config.Metadata.Labels))
			}
			if len(unmarshaled.Metadata.Annotations) != len(tt.config.Metadata.Annotations) {
				t.Errorf("Annotations count mismatch: got %d, want %d", len(unmarshaled.Metadata.Annotations), len(tt.config.Metadata.Annotations))
			}
		})
	}
}

// TestClusterConfig_YAMLMarshaling tests YAML marshaling for ClusterConfig
func TestClusterConfig_YAMLMarshaling(t *testing.T) {
	diskSize := 20.0
	enabled := true
	minGB := 10
	maxGB := 100

	tests := []struct {
		name   string
		config ClusterConfig
	}{
		{
			name: "basic cluster",
			config: ClusterConfig{
				Metadata: ResourceMetadata{
					Name: "basic-cluster",
				},
				Provider:     "AWS",
				Region:       "us-west-2",
				InstanceSize: "M10",
			},
		},
		{
			name: "advanced cluster with autoscaling and encryption",
			config: ClusterConfig{
				Metadata: ResourceMetadata{
					Name: "advanced-cluster",
					Labels: map[string]string{
						"tier": "production",
					},
					DeletionPolicy: DeletionPolicySnapshot,
				},
				Provider:       "AWS",
				Region:         "US_EAST_1",
				InstanceSize:   "M30",
				DiskSizeGB:     &diskSize,
				BackupEnabled:  &enabled,
				TierType:       "dedicated",
				MongoDBVersion: "6.0",
				ClusterType:    "REPLICASET",
				AutoScaling: &AutoScalingConfig{
					DiskGB: &AutoScalingLimits{
						Enabled:   &enabled,
						MinimumGB: &minGB,
						MaximumGB: &maxGB,
					},
					Compute: &ComputeAutoScaling{
						Enabled:          &enabled,
						ScaleDownEnabled: &enabled,
						MinInstanceSize:  "M10",
						MaxInstanceSize:  "M40",
					},
				},
				Encryption: &EncryptionConfig{
					EncryptionAtRestProvider: "AWS",
					AWSKMSConfig: &AWSKMSConfig{
						Enabled:             &enabled,
						AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
						CustomerMasterKeyID: "1234abcd-12ab-34cd-56ef-1234567890ab",
						Region:              "US_EAST_1",
						RoleID:              "507f1f77bcf86cd799439011",
					},
				},
				BiConnector: &BiConnectorConfig{
					Enabled:        &enabled,
					ReadPreference: "secondary",
				},
				ReplicationSpecs: []ReplicationSpec{
					{
						ID:        "repl-1",
						NumShards: intPtr(1),
						ZoneName:  "Zone 1",
						RegionConfigs: []RegionConfig{
							{
								RegionName:     "US_EAST_1",
								Priority:       intPtr(7),
								ProviderName:   "AWS",
								ElectableNodes: intPtr(3),
								ReadOnlyNodes:  intPtr(0),
								AnalyticsNodes: intPtr(0),
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlData, err := yaml.Marshal(tt.config)
			if err != nil {
				t.Errorf("Failed to marshal to YAML: %v", err)
				return
			}

			var unmarshaled ClusterConfig
			err = yaml.Unmarshal(yamlData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal from YAML: %v", err)
				return
			}

			if unmarshaled.Metadata.Name != tt.config.Metadata.Name {
				t.Errorf("Name mismatch: got %s, want %s", unmarshaled.Metadata.Name, tt.config.Metadata.Name)
			}
			if unmarshaled.Provider != tt.config.Provider {
				t.Errorf("Provider mismatch: got %s, want %s", unmarshaled.Provider, tt.config.Provider)
			}
			if unmarshaled.Region != tt.config.Region {
				t.Errorf("Region mismatch: got %s, want %s", unmarshaled.Region, tt.config.Region)
			}
			if unmarshaled.InstanceSize != tt.config.InstanceSize {
				t.Errorf("InstanceSize mismatch: got %s, want %s", unmarshaled.InstanceSize, tt.config.InstanceSize)
			}
		})
	}
}

// TestDatabaseUserConfig_YAMLMarshaling tests YAML marshaling for DatabaseUserConfig
func TestDatabaseUserConfig_YAMLMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		config DatabaseUserConfig
	}{
		{
			name: "basic database user",
			config: DatabaseUserConfig{
				Metadata: ResourceMetadata{
					Name: "basic-user",
				},
				Username: "testuser",
				Password: "secretpassword123",
				Roles: []DatabaseRoleConfig{
					{
						RoleName:     "read",
						DatabaseName: "test_db",
					},
				},
				AuthDatabase: "admin",
			},
		},
		{
			name: "advanced database user with scopes",
			config: DatabaseUserConfig{
				Metadata: ResourceMetadata{
					Name: "advanced-user",
					Labels: map[string]string{
						"type": "application",
					},
				},
				Username: "app_user",
				Password: "complex_password_123!",
				Roles: []DatabaseRoleConfig{
					{
						RoleName:       "readWrite",
						DatabaseName:   "app_db",
						CollectionName: "users",
					},
					{
						RoleName:     "read",
						DatabaseName: "logs_db",
					},
				},
				AuthDatabase: "admin",
				Scopes: []UserScopeConfig{
					{
						Name: "main-cluster",
						Type: "CLUSTER",
					},
				},
				DependsOn: []string{"main-cluster"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlData, err := yaml.Marshal(tt.config)
			if err != nil {
				t.Errorf("Failed to marshal to YAML: %v", err)
				return
			}

			var unmarshaled DatabaseUserConfig
			err = yaml.Unmarshal(yamlData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal from YAML: %v", err)
				return
			}

			if unmarshaled.Username != tt.config.Username {
				t.Errorf("Username mismatch: got %s, want %s", unmarshaled.Username, tt.config.Username)
			}
			if len(unmarshaled.Roles) != len(tt.config.Roles) {
				t.Errorf("Roles count mismatch: got %d, want %d", len(unmarshaled.Roles), len(tt.config.Roles))
			}
		})
	}
}

// TestNetworkAccessConfig_YAMLMarshaling tests YAML marshaling for NetworkAccessConfig
func TestNetworkAccessConfig_YAMLMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		config NetworkAccessConfig
	}{
		{
			name: "IP address access",
			config: NetworkAccessConfig{
				Metadata: ResourceMetadata{
					Name: "office-ip",
				},
				IPAddress: "192.168.1.100",
				Comment:   "Office static IP",
			},
		},
		{
			name: "CIDR access with expiration",
			config: NetworkAccessConfig{
				Metadata: ResourceMetadata{
					Name: "temp-access",
				},
				CIDR:            "10.0.0.0/16",
				Comment:         "Temporary access for development",
				DeleteAfterDate: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			name: "AWS security group",
			config: NetworkAccessConfig{
				Metadata: ResourceMetadata{
					Name: "aws-sg",
				},
				AWSSecurityGroup: "sg-1234567890abcdef0",
				Comment:          "Production security group",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlData, err := yaml.Marshal(tt.config)
			if err != nil {
				t.Errorf("Failed to marshal to YAML: %v", err)
				return
			}

			var unmarshaled NetworkAccessConfig
			err = yaml.Unmarshal(yamlData, &unmarshaled)
			if err != nil {
				t.Errorf("Failed to unmarshal from YAML: %v", err)
				return
			}

			if unmarshaled.Metadata.Name != tt.config.Metadata.Name {
				t.Errorf("Name mismatch: got %s, want %s", unmarshaled.Metadata.Name, tt.config.Metadata.Name)
			}
			if unmarshaled.IPAddress != tt.config.IPAddress {
				t.Errorf("IPAddress mismatch: got %s, want %s", unmarshaled.IPAddress, tt.config.IPAddress)
			}
			if unmarshaled.CIDR != tt.config.CIDR {
				t.Errorf("CIDR mismatch: got %s, want %s", unmarshaled.CIDR, tt.config.CIDR)
			}
		})
	}
}

// TestApplyConfig_YAMLMarshaling tests YAML marshaling for ApplyConfig
func TestApplyConfig_YAMLMarshaling(t *testing.T) {
	config := ApplyConfig{
		APIVersion: "matlas.mongodb.com/v1",
		Kind:       "Project",
		Metadata: MetadataConfig{
			Name:      "test-project",
			Namespace: "default",
			Labels: map[string]string{
				"env": "test",
			},
			Annotations: map[string]string{
				"description": "Test project configuration",
			},
		},
		Spec: ProjectConfig{
			Name:           "test-project",
			OrganizationID: "507f1f77bcf86cd799439011",
			Metadata: ResourceMetadata{
				Name: "test-project",
			},
		},
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal to YAML: %v", err)
		return
	}

	var unmarshaled ApplyConfig
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal from YAML: %v", err)
		return
	}

	if unmarshaled.APIVersion != config.APIVersion {
		t.Errorf("APIVersion mismatch: got %s, want %s", unmarshaled.APIVersion, config.APIVersion)
	}
	if unmarshaled.Kind != config.Kind {
		t.Errorf("Kind mismatch: got %s, want %s", unmarshaled.Kind, config.Kind)
	}
	if unmarshaled.Metadata.Name != config.Metadata.Name {
		t.Errorf("Metadata.Name mismatch: got %s, want %s", unmarshaled.Metadata.Name, config.Metadata.Name)
	}
}

// TestValidation_ResourceMetadata tests validation rules for ResourceMetadata
func TestValidation_ResourceMetadata(t *testing.T) {
	tests := []struct {
		name      string
		metadata  ResourceMetadata
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid metadata",
			metadata: ResourceMetadata{
				Name: "valid-name",
				Labels: map[string]string{
					"env": "prod",
				},
				DeletionPolicy: DeletionPolicyRetain,
			},
			wantError: false,
		},
		{
			name: "empty name",
			metadata: ResourceMetadata{
				Name: "",
			},
			wantError: true,
			errorMsg:  "Name is required",
		},
		{
			name: "invalid deletion policy",
			metadata: ResourceMetadata{
				Name:           "test",
				DeletionPolicy: "InvalidPolicy",
			},
			wantError: true,
			errorMsg:  "DeletionPolicy must be one of: Delete, Retain, Snapshot",
		},
		{
			name: "name too long",
			metadata: ResourceMetadata{
				Name: "this-is-a-very-long-name-that-exceeds-the-maximum-allowed-length-for-resource-names",
			},
			wantError: true,
			errorMsg:  "Name must be between 1 and 64 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.metadata)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

// TestValidation_ClusterConfig tests validation rules for ClusterConfig
func TestValidation_ClusterConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    ClusterConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid cluster config",
			config: ClusterConfig{
				Metadata: ResourceMetadata{
					Name: "test-cluster",
				},
				Provider:     "AWS",
				Region:       "US_EAST_1",
				InstanceSize: "M10",
			},
			wantError: false,
		},
		{
			name: "missing required fields",
			config: ClusterConfig{
				Metadata: ResourceMetadata{
					Name: "test-cluster",
				},
			},
			wantError: true,
			errorMsg:  "Provider, Region, InstanceSize are required",
		},
		{
			name: "invalid provider",
			config: ClusterConfig{
				Metadata: ResourceMetadata{
					Name: "test-cluster",
				},
				Provider:     "INVALID",
				Region:       "US_EAST_1",
				InstanceSize: "M10",
			},
			wantError: true,
			errorMsg:  "Provider must be one of: AWS, GCP, AZURE, TENANT",
		},
		{
			name: "invalid instance size",
			config: ClusterConfig{
				Metadata: ResourceMetadata{
					Name: "test-cluster",
				},
				Provider:     "AWS",
				Region:       "US_EAST_1",
				InstanceSize: "INVALID",
			},
			wantError: true,
			errorMsg:  "InstanceSize must be valid Atlas instance size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.config)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

// TestValidation_DatabaseUserConfig tests validation rules for DatabaseUserConfig
func TestValidation_DatabaseUserConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    DatabaseUserConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid database user",
			config: DatabaseUserConfig{
				Metadata: ResourceMetadata{
					Name: "test-user",
				},
				Username: "testuser",
				Password: "validpassword123",
				Roles: []DatabaseRoleConfig{
					{
						RoleName:     "read",
						DatabaseName: "testdb",
					},
				},
			},
			wantError: false,
		},
		{
			name: "password too short",
			config: DatabaseUserConfig{
				Metadata: ResourceMetadata{
					Name: "test-user",
				},
				Username: "testuser",
				Password: "short",
				Roles: []DatabaseRoleConfig{
					{
						RoleName:     "read",
						DatabaseName: "testdb",
					},
				},
			},
			wantError: true,
			errorMsg:  "Password must be at least 8 characters",
		},
		{
			name: "no roles",
			config: DatabaseUserConfig{
				Metadata: ResourceMetadata{
					Name: "test-user",
				},
				Username: "testuser",
				Roles:    []DatabaseRoleConfig{},
			},
			wantError: true,
			errorMsg:  "At least one role is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.config)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

// TestValidation_NetworkAccessConfig tests validation rules for NetworkAccessConfig
func TestValidation_NetworkAccessConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    NetworkAccessConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid IP address",
			config: NetworkAccessConfig{
				Metadata: ResourceMetadata{
					Name: "test-access",
				},
				IPAddress: "192.168.1.1",
			},
			wantError: false,
		},
		{
			name: "valid CIDR",
			config: NetworkAccessConfig{
				Metadata: ResourceMetadata{
					Name: "test-access",
				},
				CIDR: "192.168.1.0/24",
			},
			wantError: false,
		},
		{
			name: "invalid IP address",
			config: NetworkAccessConfig{
				Metadata: ResourceMetadata{
					Name: "test-access",
				},
				IPAddress: "invalid-ip",
			},
			wantError: true,
			errorMsg:  "IPAddress must be a valid IP address",
		},
		{
			name: "invalid CIDR",
			config: NetworkAccessConfig{
				Metadata: ResourceMetadata{
					Name: "test-access",
				},
				CIDR: "invalid-cidr",
			},
			wantError: true,
			errorMsg:  "CIDR must be a valid CIDR notation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.config)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

// TestValidation_ApplyConfig tests validation rules for ApplyConfig
func TestValidation_ApplyConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    ApplyConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid apply config",
			config: ApplyConfig{
				APIVersion: "matlas.mongodb.com/v1",
				Kind:       "Project",
				Metadata: MetadataConfig{
					Name: "test-project",
				},
				Spec: ProjectConfig{
					Name:           "test-project",
					OrganizationID: "507f1f77bcf86cd799439011",
					Metadata: ResourceMetadata{
						Name: "test-project",
					},
				},
			},
			wantError: false,
		},
		{
			name: "invalid API version",
			config: ApplyConfig{
				APIVersion: "invalid/v1",
				Kind:       "Project",
				Metadata: MetadataConfig{
					Name: "test-project",
				},
				Spec: ProjectConfig{
					Name:           "test-project",
					OrganizationID: "507f1f77bcf86cd799439011",
				},
			},
			wantError: true,
			errorMsg:  "APIVersion must be valid",
		},
		{
			name: "invalid kind",
			config: ApplyConfig{
				APIVersion: "matlas.mongodb.com/v1",
				Kind:       "InvalidKind",
				Metadata: MetadataConfig{
					Name: "test-project",
				},
				Spec: ProjectConfig{
					Name:           "test-project",
					OrganizationID: "507f1f77bcf86cd799439011",
				},
			},
			wantError: true,
			errorMsg:  "Kind must be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.config)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}

// TestMultiDocumentYAMLParsing tests parsing of multi-document YAML files
func TestMultiDocumentYAMLParsing(t *testing.T) {
	multiDocYAML := `---
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: project1
spec:
  name: project1
  organizationId: "507f1f77bcf86cd799439011"
---
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: project2
spec:
  name: project2
  organizationId: "507f1f77bcf86cd799439011"
`

	decoder := yaml.NewDecoder(strings.NewReader(multiDocYAML))
	var configs []ApplyConfig

	for {
		var config ApplyConfig
		err := decoder.Decode(&config)
		if err != nil {
			break
		}
		configs = append(configs, config)
	}

	if len(configs) != 2 {
		t.Errorf("Expected 2 configurations, got %d", len(configs))
	}

	if configs[0].Metadata.Name != "project1" {
		t.Errorf("First config name mismatch: got %s, want project1", configs[0].Metadata.Name)
	}

	if configs[1].Metadata.Name != "project2" {
		t.Errorf("Second config name mismatch: got %s, want project2", configs[1].Metadata.Name)
	}
}

// TestComplexValidationScenarios tests complex validation scenarios
func TestComplexValidationScenarios(t *testing.T) {
	tests := []struct {
		name      string
		config    interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "autoscaling with invalid minimum greater than maximum",
			config: AutoScalingLimits{
				MinimumGB: intPtr(100),
				MaximumGB: intPtr(50), // This should fail gtfield validation
			},
			wantError: true,
			errorMsg:  "MaximumGB must be greater than MinimumGB",
		},
		{
			name: "encryption config with multiple providers",
			config: EncryptionConfig{
				EncryptionAtRestProvider: "AWS",
				AWSKMSConfig: &AWSKMSConfig{
					AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
					CustomerMasterKeyID: "1234abcd-12ab-34cd-56ef-1234567890ab",
					Region:              "US_EAST_1",
					RoleID:              "507f1f77bcf86cd799439011",
				},
			},
			wantError: false,
		},
		{
			name: "azure config with invalid UUID",
			config: AzureKeyVaultConfig{
				ClientID:       "not-a-uuid",
				SubscriptionID: "also-not-a-uuid",
			},
			wantError: true,
			errorMsg:  "ClientID and SubscriptionID must be valid UUIDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.Struct(tt.config)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}
