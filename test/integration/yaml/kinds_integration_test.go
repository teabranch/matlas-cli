//go:build integration
// +build integration

package yaml

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

// TestProjectKindValidation tests Project kind YAML validation
func TestProjectKindValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Project kind validation test in short mode")
	}

	testCases := []struct {
		name        string
		yamlContent string
		expectValid bool
		description string
	}{
		{
			name: "ValidMinimalProject",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-project
spec:
  name: "Test Project"
  organizationId: "5f1d7f3a9d1e8b1234567890"`,
			expectValid: true,
			description: "Valid minimal Project configuration",
		},
		{
			name: "ValidProjectWithEmbeddedResources",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: comprehensive-project
spec:
  name: "Comprehensive Project"
  organizationId: "5f1d7f3a9d1e8b1234567890"
  clusters:
    - metadata:
        name: test-cluster
      provider: AWS
      region: US_EAST_1
      instanceSize: M0
  databaseUsers:
    - metadata:
        name: test-user
      username: testuser
      authDatabase: admin
      roles:
        - roleName: read
          databaseName: admin
  networkAccess:
    - metadata:
        name: test-access
      ipAddress: "192.0.2.1"
      comment: "Test access"`,
			expectValid: true,
			description: "Valid Project with embedded resources",
		},
		{
			name: "InvalidMissingOrganizationId",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: invalid-project
spec:
  name: "Invalid Project"`,
			expectValid: false,
			description: "Invalid Project missing organizationId",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateKindYAML(t, tc.yamlContent, tc.expectValid, tc.description)
		})
	}
}

// TestClusterKindValidation tests Cluster kind YAML validation
func TestClusterKindValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Cluster kind validation test in short mode")
	}

	testCases := []struct {
		name        string
		yamlContent string
		expectValid bool
		description string
	}{
		{
			name: "ValidBasicCluster",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: cluster-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: test-cluster
    spec:
      projectName: "test-project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M10`,
			expectValid: true,
			description: "Valid basic Cluster configuration",
		},
		{
			name: "ValidAdvancedCluster",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: advanced-cluster-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: advanced-cluster
      labels:
        environment: production
    spec:
      projectName: "test-project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M40
      diskSizeGB: 100
      backupEnabled: true
      mongodbVersion: "7.0"
      clusterType: "REPLICASET"
      tags:
        purpose: "production-workload"`,
			expectValid: true,
			description: "Valid advanced Cluster configuration",
		},
		{
			name: "InvalidMissingProvider",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-cluster-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: invalid-cluster
    spec:
      projectName: "test-project"
      region: US_EAST_1
      instanceSize: M10`,
			expectValid: false,
			description: "Invalid Cluster missing provider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateKindYAML(t, tc.yamlContent, tc.expectValid, tc.description)
		})
	}
}

// TestDatabaseUserKindValidation tests DatabaseUser kind YAML validation
func TestDatabaseUserKindValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping DatabaseUser kind validation test in short mode")
	}

	testCases := []struct {
		name        string
		yamlContent string
		expectValid bool
		description string
	}{
		{
			name: "ValidBasicDatabaseUser",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: user-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: test-user
    spec:
      projectName: "test-project"
      username: testuser
      authDatabase: admin
      password: "TestPassword123!"
      roles:
        - roleName: read
          databaseName: admin`,
			expectValid: true,
			description: "Valid basic DatabaseUser configuration",
		},
		{
			name: "ValidDatabaseUserWithScopes",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: scoped-user-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: scoped-user
    spec:
      projectName: "test-project"
      username: scopeduser
      authDatabase: admin
      password: "TestPassword123!"
      roles:
        - roleName: readWrite
          databaseName: myapp
        - roleName: read
          databaseName: analytics
      scopes:
        - name: "production-cluster"
          type: CLUSTER`,
			expectValid: true,
			description: "Valid DatabaseUser with scopes",
		},
		{
			name: "InvalidMissingRoles",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-user-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: invalid-user
    spec:
      projectName: "test-project"
      username: invaliduser
      authDatabase: admin
      password: "TestPassword123!"
      roles: []`,
			expectValid: false,
			description: "Invalid DatabaseUser with no roles",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateKindYAML(t, tc.yamlContent, tc.expectValid, tc.description)
		})
	}
}

// TestNetworkAccessKindValidation tests NetworkAccess kind YAML validation
func TestNetworkAccessKindValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping NetworkAccess kind validation test in short mode")
	}

	testCases := []struct {
		name        string
		yamlContent string
		expectValid bool
		description string
	}{
		{
			name: "ValidIPAddress",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: network-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: office-ip
    spec:
      projectName: "test-project"
      ipAddress: "203.0.113.1"
      comment: "Office IP access"`,
			expectValid: true,
			description: "Valid NetworkAccess with IP address",
		},
		{
			name: "ValidCIDRBlock",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: cidr-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: office-network
    spec:
      projectName: "test-project"
      cidr: "203.0.113.0/24"
      comment: "Office network access"
      deleteAfterDate: "2024-12-31T23:59:59Z"`,
			expectValid: true,
			description: "Valid NetworkAccess with CIDR block",
		},
		{
			name: "ValidAWSSecurityGroup",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: aws-sg-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: aws-security-group
    spec:
      projectName: "test-project"
      awsSecurityGroup: "sg-12345678"
      comment: "AWS security group access"`,
			expectValid: true,
			description: "Valid NetworkAccess with AWS security group",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateKindYAML(t, tc.yamlContent, tc.expectValid, tc.description)
		})
	}
}

// TestSearchIndexKindValidation tests SearchIndex kind YAML validation
func TestSearchIndexKindValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SearchIndex kind validation test in short mode")
	}

	testCases := []struct {
		name        string
		yamlContent string
		expectValid bool
		description string
	}{
		{
			name: "ValidBasicSearchIndex",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: search-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: basic-search
    spec:
      projectName: "test-project"
      clusterName: "test-cluster"
      databaseName: "testdb"
      collectionName: "documents"
      indexName: "default"
      indexType: "search"
      definition:
        mappings:
          dynamic: true`,
			expectValid: true,
			description: "Valid basic SearchIndex configuration",
		},
		{
			name: "ValidVectorSearchIndex",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: vector-search-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: vector-search
    spec:
      projectName: "test-project"
      clusterName: "test-cluster"
      databaseName: "testdb"
      collectionName: "embeddings"
      indexName: "vector-index"
      indexType: "vectorSearch"
      definition:
        fields:
          - type: "vector"
            path: "embedding"
            numDimensions: 1536
            similarity: "cosine"`,
			expectValid: true,
			description: "Valid vector SearchIndex configuration",
		},
		{
			name: "InvalidMissingIndexName",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-search-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: invalid-search
    spec:
      projectName: "test-project"
      clusterName: "test-cluster"
      databaseName: "testdb"
      collectionName: "documents"
      indexType: "search"`,
			expectValid: false,
			description: "Invalid SearchIndex missing indexName",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateKindYAML(t, tc.yamlContent, tc.expectValid, tc.description)
		})
	}
}

// TestVPCEndpointKindValidation tests VPCEndpoint kind YAML validation
func TestVPCEndpointKindValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VPCEndpoint kind validation test in short mode")
	}

	testCases := []struct {
		name        string
		yamlContent string
		expectValid bool
		description string
	}{
		{
			name: "ValidAWSVPCEndpoint",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: vpc-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: aws-vpc-endpoint
    spec:
      projectName: "test-project"
      cloudProvider: "AWS"
      region: "us-east-1"`,
			expectValid: true,
			description: "Valid AWS VPCEndpoint configuration",
		},
		{
			name: "ValidAzureVPCEndpoint",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: azure-vpc-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: azure-vpc-endpoint
    spec:
      projectName: "test-project"
      cloudProvider: "AZURE"
      region: "eastus"`,
			expectValid: true,
			description: "Valid Azure VPCEndpoint configuration",
		},
		{
			name: "ValidGCPVPCEndpoint",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: gcp-vpc-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: gcp-vpc-endpoint
    spec:
      projectName: "test-project"
      cloudProvider: "GCP"
      region: "us-central1"`,
			expectValid: true,
			description: "Valid GCP VPCEndpoint configuration",
		},
		{
			name: "InvalidCloudProvider",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-vpc-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: invalid-vpc-endpoint
    spec:
      projectName: "test-project"
      cloudProvider: "INVALID"
      region: "us-east-1"`,
			expectValid: false,
			description: "Invalid VPCEndpoint with invalid cloud provider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validateKindYAML(t, tc.yamlContent, tc.expectValid, tc.description)
		})
	}
}

// TestApplyDocumentKindValidation tests ApplyDocument kind YAML validation
func TestApplyDocumentKindValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ApplyDocument kind validation test in short mode")
	}

	timestamp := time.Now().Unix()
	yamlContent := fmt.Sprintf(`apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: comprehensive-test-%d
  labels:
    test-type: validation
    purpose: comprehensive-testing
resources:
  # Cluster resource
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: test-cluster-%d
    spec:
      projectName: "test-project"
      provider: AWS
      region: US_EAST_1
      instanceSize: M10

  # DatabaseUser resource
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: test-user-%d
    spec:
      projectName: "test-project"
      username: testuser%d
      authDatabase: admin
      password: "TestPassword123!"
      roles:
        - roleName: readWrite
          databaseName: myapp
      scopes:
        - name: "test-cluster-%d"
          type: CLUSTER

  # NetworkAccess resource
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata:
      name: test-access-%d
    spec:
      projectName: "test-project"
      ipAddress: "203.0.113.100"
      comment: "Test access for validation"

  # SearchIndex resource
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: test-search-%d
    spec:
      projectName: "test-project"
      clusterName: "test-cluster-%d"
      databaseName: "myapp"
      collectionName: "documents"
      indexName: "search-index"
      indexType: "search"
      definition:
        mappings:
          dynamic: true

  # VPCEndpoint resource
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: test-vpc-%d
    spec:
      projectName: "test-project"
      cloudProvider: "AWS"
      region: "us-east-1"
`, timestamp, timestamp, timestamp, timestamp, timestamp, timestamp, timestamp, timestamp, timestamp)

	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "comprehensive-test.yaml")
	
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err, "Should write test YAML file")

	// Load as ApplyDocument
	document, err := apply.LoadApplyDocument(yamlFile)
	require.NoError(t, err, "Should load comprehensive YAML as ApplyDocument")

	// Verify document structure
	assert.Equal(t, types.KindApplyDocument, document.Kind, "Should be ApplyDocument")
	assert.Len(t, document.Resources, 5, "Should have 5 resources")

	// Verify each resource type is present
	kindsCounts := make(map[types.ResourceKind]int)
	for _, resource := range document.Resources {
		switch resource.(type) {
		case *types.ClusterManifest:
			kindsCounts[types.KindCluster]++
		case *types.DatabaseUserManifest:
			kindsCounts[types.KindDatabaseUser]++
		case *types.NetworkAccessManifest:
			kindsCounts[types.KindNetworkAccess]++
		case *types.SearchIndexManifest:
			kindsCounts[types.KindSearchIndex]++
		case *types.VPCEndpointManifest:
			kindsCounts[types.KindVPCEndpoint]++
		}
	}

	// Verify we have one of each expected resource type
	assert.Equal(t, 1, kindsCounts[types.KindCluster], "Should have 1 Cluster")
	assert.Equal(t, 1, kindsCounts[types.KindDatabaseUser], "Should have 1 DatabaseUser")
	assert.Equal(t, 1, kindsCounts[types.KindNetworkAccess], "Should have 1 NetworkAccess")
	assert.Equal(t, 1, kindsCounts[types.KindSearchIndex], "Should have 1 SearchIndex")
	assert.Equal(t, 1, kindsCounts[types.KindVPCEndpoint], "Should have 1 VPCEndpoint")

	// Validate through apply validator
	validator := apply.NewValidator()
	result := validator.ValidateDocument(document)

	// Log validation results
	if len(result.Warnings) > 0 {
		t.Log("Validation warnings:")
		for _, warning := range result.Warnings {
			t.Logf("  - %s: %s", warning.Path, warning.Message)
		}
	}
	if len(result.Errors) > 0 {
		t.Log("Validation errors:")
		for _, err := range result.Errors {
			t.Logf("  - %s: %s", err.Path, err.Message)
		}
	}

	// Should pass validation
	assert.Empty(t, result.Errors, "Comprehensive ApplyDocument should pass validation")
}

// Helper function to validate YAML content
func validateKindYAML(t *testing.T, yamlContent string, expectValid bool, description string) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yaml")
	
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err, "Should write test YAML file")

	// Try to load the YAML
	var validationResult *apply.ValidationResult
	
	// Try as ApplyConfig first
	if config, err := apply.LoadConfiguration(yamlFile); err == nil {
		validator := apply.NewValidator()
		validationResult = validator.ValidateConfiguration(config)
	} else {
		// Try as ApplyDocument
		document, err := apply.LoadApplyDocument(yamlFile)
		require.NoError(t, err, "Should load YAML file")
		
		validator := apply.NewValidator()
		validationResult = validator.ValidateDocument(document)
	}

	require.NotNil(t, validationResult, "Should get validation result")

	// Log validation results for debugging
	if len(validationResult.Warnings) > 0 {
		t.Logf("Validation warnings for %s:", description)
		for _, warning := range validationResult.Warnings {
			t.Logf("  - %s: %s", warning.Path, warning.Message)
		}
	}
	if len(validationResult.Errors) > 0 {
		t.Logf("Validation errors for %s:", description)
		for _, err := range validationResult.Errors {
			t.Logf("  - %s: %s", err.Path, err.Message)
		}
	}

	// Check validation result
	if expectValid {
		assert.Empty(t, validationResult.Errors, "Should pass validation: %s", description)
	} else {
		assert.NotEmpty(t, validationResult.Errors, "Should fail validation: %s", description)
	}
}