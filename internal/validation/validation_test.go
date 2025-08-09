package validation

import (
	"strings"
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestValidateObjectID(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		fieldName   string
		expectError bool
	}{
		{
			name:        "valid ObjectID",
			id:          "507f1f77bcf86cd799439011",
			fieldName:   "projectID",
			expectError: false,
		},
		{
			name:        "empty ObjectID",
			id:          "",
			fieldName:   "projectID",
			expectError: true,
		},
		{
			name:        "too short ObjectID",
			id:          "507f1f77bcf86cd799439",
			fieldName:   "projectID",
			expectError: true,
		},
		{
			name:        "invalid characters",
			id:          "507f1f77bcf86cd79943901g",
			fieldName:   "projectID",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateObjectID(tt.id, tt.fieldName)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestValidateAtlasInstanceSize(t *testing.T) {
	tests := []struct {
		name        string
		size        string
		expectError bool
	}{
		{"valid M0", "M0", false},
		{"valid M10", "M10", false},
		{"valid R40", "R40", false},
		{"invalid size", "X10", true},
		{"empty size", "", true},
		{"lowercase", "m10", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAtlasInstanceSize(tt.size, "instanceSize")
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestValidateAtlasProvider(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		expectError bool
	}{
		{"valid AWS", "AWS", false},
		{"valid GCP", "GCP", false},
		{"valid AZURE", "AZURE", false},
		{"lowercase aws", "aws", false}, // Should work due to ToUpper
		{"invalid provider", "DIGITAL_OCEAN", true},
		{"empty provider", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAtlasProvider(tt.provider, "provider")
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestValidateAtlasRegion(t *testing.T) {
	tests := []struct {
		name        string
		region      string
		provider    string
		expectError bool
	}{
		{"valid AWS region (Atlas style)", "US_EAST_1", "AWS", false},
		{"valid AWS region (provider style)", "eu-west-1", "AWS", false},
		{"valid GCP region", "us-central1", "GCP", false},
		{"valid Azure region", "eastus", "AZURE", false},
		{"invalid AWS region", "invalid-region", "AWS", true},
		{"empty region", "", "AWS", true},
		{"invalid provider now errors", "US_EAST_1", "INVALID", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAtlasRegion(tt.region, tt.provider, "region")
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestDependencyValidator(t *testing.T) {
	validator := NewDependencyValidator(false)

	// Test basic project config
	config := &types.ProjectConfig{
		Name:           "test-project",
		OrganizationID: "507f1f77bcf86cd799439011",
		Clusters: []types.ClusterConfig{
			{
				Metadata: types.ResourceMetadata{
					Name: "cluster1",
				},
				Provider:     "AWS",
				Region:       "US_EAST_1",
				InstanceSize: "M10",
			},
		},
		DatabaseUsers: []types.DatabaseUserConfig{
			{
				Username: "testuser",
				Roles: []types.DatabaseRoleConfig{
					{
						RoleName:     "readWrite",
						DatabaseName: "mydb",
					},
				},
				Scopes: []types.UserScopeConfig{
					{
						Type: "CLUSTER",
						Name: "cluster1",
					},
				},
			},
		},
		NetworkAccess: []types.NetworkAccessConfig{
			{
				IPAddress: "192.168.1.1",
			},
		},
	}

	issues, err := validator.ValidateProjectDependencies(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have no issues for valid configuration
	if len(issues) > 0 {
		t.Errorf("Expected no issues, got %d issues", len(issues))
		for _, issue := range issues {
			t.Logf("Issue: %s", issue.Message)
		}
	}
}

func TestDependencyValidator_MissingCluster(t *testing.T) {
	validator := NewDependencyValidator(false)

	// Test config with user referencing non-existent cluster
	config := &types.ProjectConfig{
		Name:           "test-project",
		OrganizationID: "507f1f77bcf86cd799439011",
		Clusters:       []types.ClusterConfig{}, // No clusters
		DatabaseUsers: []types.DatabaseUserConfig{
			{
				Username: "testuser",
				Roles: []types.DatabaseRoleConfig{
					{
						RoleName:     "readWrite",
						DatabaseName: "mydb",
					},
				},
				Scopes: []types.UserScopeConfig{
					{
						Type: "CLUSTER",
						Name: "nonexistent-cluster", // References non-existent cluster
					},
				},
			},
		},
	}

	issues, err := validator.ValidateProjectDependencies(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have dependency issues
	foundMissingResource := false
	for _, issue := range issues {
		if issue.DependencyType == "missing_resource" &&
			strings.Contains(issue.Message, "nonexistent-cluster") {
			foundMissingResource = true
			break
		}
	}

	if !foundMissingResource {
		t.Errorf("Expected missing resource issue for nonexistent cluster")
	}
}

func TestDependencyValidator_DuplicateNames(t *testing.T) {
	validator := NewDependencyValidator(false)

	// Test config with duplicate cluster names
	config := &types.ProjectConfig{
		Name:           "test-project",
		OrganizationID: "507f1f77bcf86cd799439011",
		Clusters: []types.ClusterConfig{
			{
				Metadata: types.ResourceMetadata{
					Name: "duplicate-name",
				},
				Provider:     "AWS",
				Region:       "US_EAST_1",
				InstanceSize: "M10",
			},
			{
				Metadata: types.ResourceMetadata{
					Name: "duplicate-name", // Duplicate name
				},
				Provider:     "GCP",
				Region:       "us-central1",
				InstanceSize: "M20",
			},
		},
	}

	issues, err := validator.ValidateProjectDependencies(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have naming conflict issues
	foundNamingConflict := false
	for _, issue := range issues {
		if issue.DependencyType == "naming_conflict" {
			foundNamingConflict = true
			break
		}
	}

	if !foundNamingConflict {
		t.Errorf("Expected naming conflict issue for duplicate cluster names")
	}
}

func TestDependencyValidator_NetworkAccessExpiration(t *testing.T) {
	validator := NewDependencyValidator(false)

	// Test network access with past expiration date
	pastDate := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)

	config := &types.ProjectConfig{
		Name:           "test-project",
		OrganizationID: "507f1f77bcf86cd799439011",
		NetworkAccess: []types.NetworkAccessConfig{
			{
				IPAddress:       "192.168.1.1",
				DeleteAfterDate: pastDate,
			},
		},
	}

	issues, err := validator.ValidateProjectDependencies(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have temporal issues
	foundTemporalIssue := false
	for _, issue := range issues {
		if issue.DependencyType == "temporal" && issue.Severity == "error" {
			foundTemporalIssue = true
			break
		}
	}

	if !foundTemporalIssue {
		t.Errorf("Expected temporal issue for expired network access rule")
	}
}

func TestDependencyValidator_ProviderRegionCompatibility(t *testing.T) {
	validator := NewDependencyValidator(false)

	// Test incompatible provider-region combination
	config := &types.ProjectConfig{
		Name:           "test-project",
		OrganizationID: "507f1f77bcf86cd799439011",
		Clusters: []types.ClusterConfig{
			{
				Metadata: types.ResourceMetadata{
					Name: "test-cluster",
				},
				Provider:     "AWS",
				Region:       "us-central1", // GCP region with AWS provider
				InstanceSize: "M10",
			},
		},
	}

	issues, err := validator.ValidateProjectDependencies(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have compatibility issues
	foundCompatibilityIssue := false
	for _, issue := range issues {
		if issue.DependencyType == "compatibility" && issue.Severity == "error" {
			foundCompatibilityIssue = true
			break
		}
	}

	if !foundCompatibilityIssue {
		t.Errorf("Expected compatibility issue for incompatible provider-region combination")
	}
}

func TestSchemaValidator(t *testing.T) {
	validator := NewSchemaValidator()

	// Test valid configuration
	validConfig := `
output: json
timeout: 30s
projectId: "507f1f77bcf86cd799439011"
clusterName: "test-cluster"
apiKey: "test-api-key"
publicKey: "test-public-key"
`

	result, err := validator.ValidateConfigWithSchema([]byte(validConfig), "matlas-config")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid configuration, got invalid")
		for _, error := range result.Errors {
			t.Logf("Error: %s", error.Message)
		}
	}

	// Test invalid configuration
	invalidConfig := `
output: invalid-format
projectId: "too-short"
clusterName: ""
`

	result, err = validator.ValidateConfigWithSchema([]byte(invalidConfig), "matlas-config")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Valid {
		t.Errorf("Expected invalid configuration, got valid")
	}

	if len(result.Errors) == 0 {
		t.Errorf("Expected validation errors, got none")
	}
}

func TestSchemaValidator_ApplyConfig(t *testing.T) {
	validator := NewSchemaValidator()

	validApplyConfig := `
apiVersion: v1
kind: Project
metadata:
  name: test-project
spec:
  name: test-project
  organizationId: "507f1f77bcf86cd799439011"
  clusters:
    - metadata:
        name: test-cluster
      provider: AWS
      region: US_EAST_1
      instanceSize: M10
  databaseUsers:
    - username: testuser
      roles:
        - roleName: readWrite
          databaseName: mydb
`

	result, err := validator.ValidateConfigWithSchema([]byte(validApplyConfig), "apply-config")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid apply configuration, got invalid")
		for _, error := range result.Errors {
			t.Logf("Error: %s", error.Message)
		}
	}
}

func TestSchemaValidator_InvalidYAML(t *testing.T) {
	validator := NewSchemaValidator()

	invalidYAML := `
output: json
invalid: yaml: syntax
  missing: colon
`

	result, err := validator.ValidateConfigWithSchema([]byte(invalidYAML), "matlas-config")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Valid {
		t.Errorf("Expected invalid YAML to fail validation")
	}

	if len(result.Errors) == 0 {
		t.Errorf("Expected YAML syntax errors")
	}
}

func TestValidationIssue_Error(t *testing.T) {
	issue := ValidationIssue{
		Path:    "spec.clusters[0].name",
		Message: "Name is required",
	}

	expected := "spec.clusters[0].name: Name is required"
	if issue.Error() != expected {
		t.Errorf("Expected error string %q, got %q", expected, issue.Error())
	}
}

func TestValidateEnum(t *testing.T) {
	allowedValues := []string{"json", "yaml", "table"}

	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		{"valid enum value", "json", false},
		{"invalid enum value", "xml", true},
		{"empty value", "", true},
		{"case sensitive", "JSON", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnum(tt.value, "output", allowedValues)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestValidateRange(t *testing.T) {
	tests := []struct {
		name        string
		value       int
		min         int
		max         int
		expectError bool
	}{
		{"valid range", 5, 1, 10, false},
		{"at minimum", 1, 1, 10, false},
		{"at maximum", 10, 1, 10, false},
		{"below minimum", 0, 1, 10, true},
		{"above maximum", 11, 1, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRange(tt.value, "port", tt.min, tt.max)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error for %s, got %v", tt.name, err)
			}
		})
	}
}
