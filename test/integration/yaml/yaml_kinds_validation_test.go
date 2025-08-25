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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

// TestYAMLKindsValidation tests that all example YAML files can be loaded and validated
func TestYAMLKindsValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping YAML validation integration test in short mode")
	}

	projectRoot := getProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples")

	// Find all YAML files in examples directory
	yamlFiles, err := findYAMLFiles(examplesDir)
	require.NoError(t, err, "Should be able to find YAML files in examples directory")
	require.NotEmpty(t, yamlFiles, "Should find at least one YAML file in examples")

	// Keep track of which kinds we've seen
	kindsFound := make(map[types.ResourceKind][]string)

	for _, yamlFile := range yamlFiles {
		t.Run(filepath.Base(yamlFile), func(t *testing.T) {
			// Load the YAML file
			config, err := apply.LoadConfiguration(yamlFile)
			require.NoError(t, err, "Should be able to load YAML file: %s", yamlFile)
			require.NotNil(t, config, "Configuration should not be nil")

			// Validate basic structure
			assert.NotEmpty(t, config.APIVersion, "API version should be present")
			assert.NotEmpty(t, config.Kind, "Kind should be present")
			assert.NotEmpty(t, config.Metadata.Name, "Metadata name should be present")

			// Validate API version format
			assert.True(t, strings.HasPrefix(config.APIVersion, "matlas.mongodb.com/"),
				"API version should start with matlas.mongodb.com/")

			// Record which kind this is
			resourceKind := types.ResourceKind(config.Kind)
			kindsFound[resourceKind] = append(kindsFound[resourceKind], yamlFile)

			// Validate specific to kind
			switch resourceKind {
			case types.KindProject:
				validateProjectKind(t, config, yamlFile)
			case types.KindApplyDocument:
				validateApplyDocumentKind(t, config, yamlFile, kindsFound)
			default:
				// For other kinds loaded as top-level resources
				validateGenericKind(t, config, yamlFile)
			}

			// Run full validation through the validator
			validator := apply.NewValidator()
			result := validator.ValidateConfiguration(config)

			// Log any validation warnings for inspection
			if len(result.Warnings) > 0 {
				t.Logf("Validation warnings for %s:", yamlFile)
				for _, warning := range result.Warnings {
					t.Logf("  - %s: %s", warning.Path, warning.Message)
				}
			}

			// Should not have validation errors
			if len(result.Errors) > 0 {
				t.Errorf("Validation errors for %s:", yamlFile)
				for _, err := range result.Errors {
					t.Errorf("  - %s: %s", err.Path, err.Message)
				}
			}
			assert.Empty(t, result.Errors, "YAML file should pass validation: %s", yamlFile)
		})
	}

	// Verify we have examples for all documented kinds
	t.Run("KindsCoverage", func(t *testing.T) {
		expectedKinds := []types.ResourceKind{
			types.KindProject,
			types.KindCluster,
			types.KindDatabaseUser,
			types.KindDatabaseRole,
			types.KindNetworkAccess,
			types.KindSearchIndex,
			types.KindVPCEndpoint,
			types.KindApplyDocument,
		}

		for _, expectedKind := range expectedKinds {
			assert.Contains(t, kindsFound, expectedKind,
				"Should have at least one example for kind: %s", expectedKind)

			if files, ok := kindsFound[expectedKind]; ok {
				t.Logf("Found %d examples for kind %s: %v", len(files), expectedKind, files)
			}
		}
	})
}

// validateProjectKind validates Project-specific requirements
func validateProjectKind(t *testing.T, config *types.ApplyConfig, yamlFile string) {
	assert.NotEmpty(t, config.Spec.Name, "Project should have a name: %s", yamlFile)
	assert.NotEmpty(t, config.Spec.OrganizationID, "Project should have organizationId: %s", yamlFile)

	// Validate embedded resources if present
	if len(config.Spec.Clusters) > 0 {
		for i, cluster := range config.Spec.Clusters {
			assert.NotEmpty(t, cluster.Metadata.Name, "Cluster %d should have name in %s", i, yamlFile)
			assert.NotEmpty(t, cluster.Provider, "Cluster %d should have provider in %s", i, yamlFile)
		}
	}
	if len(config.Spec.DatabaseUsers) > 0 {
		for i, user := range config.Spec.DatabaseUsers {
			assert.NotEmpty(t, user.Metadata.Name, "DatabaseUser %d should have name in %s", i, yamlFile)
			assert.NotEmpty(t, user.Username, "DatabaseUser %d should have username in %s", i, yamlFile)
		}
	}
}

// validateApplyDocumentKind validates ApplyDocument-specific requirements
func validateApplyDocumentKind(t *testing.T, config *types.ApplyConfig, yamlFile string, kindsFound map[types.ResourceKind][]string) {
	// For ApplyDocument, we need to load it differently
	document, err := apply.LoadApplyDocument(yamlFile)
	require.NoError(t, err, "Should be able to load ApplyDocument: %s", yamlFile)

	assert.NotEmpty(t, document.Resources, "ApplyDocument should have resources: %s", yamlFile)

	// Track kinds found in this document
	for _, resource := range document.Resources {
		switch r := resource.(type) {
		case *types.ClusterManifest:
			kindsFound[types.KindCluster] = append(kindsFound[types.KindCluster], yamlFile)
			assert.NotEmpty(t, r.Metadata.Name, "Cluster resource should have name")
			assert.NotEmpty(t, r.Spec.Provider, "Cluster resource should have provider")
		case *types.DatabaseUserManifest:
			kindsFound[types.KindDatabaseUser] = append(kindsFound[types.KindDatabaseUser], yamlFile)
			assert.NotEmpty(t, r.Metadata.Name, "DatabaseUser resource should have name")
			assert.NotEmpty(t, r.Spec.Username, "DatabaseUser resource should have username")
		case *types.DatabaseRoleManifest:
			kindsFound[types.KindDatabaseRole] = append(kindsFound[types.KindDatabaseRole], yamlFile)
			assert.NotEmpty(t, r.Metadata.Name, "DatabaseRole resource should have name")
			assert.NotEmpty(t, r.Spec.RoleName, "DatabaseRole resource should have roleName")
		case *types.NetworkAccessManifest:
			kindsFound[types.KindNetworkAccess] = append(kindsFound[types.KindNetworkAccess], yamlFile)
			assert.NotEmpty(t, r.Metadata.Name, "NetworkAccess resource should have name")
		case *types.SearchIndexManifest:
			kindsFound[types.KindSearchIndex] = append(kindsFound[types.KindSearchIndex], yamlFile)
			assert.NotEmpty(t, r.Metadata.Name, "SearchIndex resource should have name")
			assert.NotEmpty(t, r.Spec.IndexName, "SearchIndex resource should have indexName")
		case *types.VPCEndpointManifest:
			kindsFound[types.KindVPCEndpoint] = append(kindsFound[types.KindVPCEndpoint], yamlFile)
			assert.NotEmpty(t, r.Metadata.Name, "VPCEndpoint resource should have name")
			assert.NotEmpty(t, r.Spec.CloudProvider, "VPCEndpoint resource should have cloudProvider")
		}
	}
}

// validateGenericKind validates common requirements for any kind
func validateGenericKind(t *testing.T, config *types.ApplyConfig, yamlFile string) {
	// Basic validation that applies to all kinds
	assert.NotEmpty(t, config.APIVersion, "Should have apiVersion: %s", yamlFile)
	assert.NotEmpty(t, config.Kind, "Should have kind: %s", yamlFile)
	assert.NotEmpty(t, config.Metadata.Name, "Should have metadata.name: %s", yamlFile)
}

// TestYAMLSchemaCompliance tests that all YAML files match their documented schemas
func TestYAMLSchemaCompliance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping schema compliance test in short mode")
	}

	projectRoot := getProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples")

	// Test cases for specific schema requirements
	testCases := []struct {
		name     string
		pattern  string
		validate func(t *testing.T, yamlFile string)
	}{
		{
			name:     "SearchIndex examples have required fields",
			pattern:  "*search*.yaml",
			validate: validateSearchIndexSchema,
		},
		{
			name:     "VPCEndpoint examples have required fields",
			pattern:  "*vpc*.yaml",
			validate: validateVPCEndpointSchema,
		},
		{
			name:     "Cluster examples have required fields",
			pattern:  "*cluster*.yaml",
			validate: validateClusterSchema,
		},
		{
			name:     "DatabaseRole examples have required fields",
			pattern:  "*role*.yaml",
			validate: validateDatabaseRoleSchema,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pattern := filepath.Join(examplesDir, tc.pattern)
			matches, err := filepath.Glob(pattern)
			require.NoError(t, err, "Should be able to glob pattern: %s", pattern)

			if len(matches) == 0 {
				t.Skipf("No files found matching pattern: %s", pattern)
				return
			}

			for _, yamlFile := range matches {
				tc.validate(t, yamlFile)
			}
		})
	}
}

// Schema validation functions for specific kinds

func validateSearchIndexSchema(t *testing.T, yamlFile string) {
	document, err := apply.LoadApplyDocument(yamlFile)
	require.NoError(t, err, "Should load ApplyDocument: %s", yamlFile)

	found := false
	for _, resource := range document.Resources {
		if searchIndex, ok := resource.(*types.SearchIndexManifest); ok {
			found = true
			assert.NotEmpty(t, searchIndex.Spec.ProjectName, "SearchIndex should have projectName")
			assert.NotEmpty(t, searchIndex.Spec.ClusterName, "SearchIndex should have clusterName")
			assert.NotEmpty(t, searchIndex.Spec.DatabaseName, "SearchIndex should have databaseName")
			assert.NotEmpty(t, searchIndex.Spec.CollectionName, "SearchIndex should have collectionName")
			assert.NotEmpty(t, searchIndex.Spec.IndexName, "SearchIndex should have indexName")
			assert.Contains(t, []string{"search", "vectorSearch"}, searchIndex.Spec.IndexType,
				"SearchIndex should have valid indexType")
		}
	}
	assert.True(t, found, "Should find SearchIndex resource in file: %s", yamlFile)
}

func validateVPCEndpointSchema(t *testing.T, yamlFile string) {
	document, err := apply.LoadApplyDocument(yamlFile)
	require.NoError(t, err, "Should load ApplyDocument: %s", yamlFile)

	found := false
	for _, resource := range document.Resources {
		if vpcEndpoint, ok := resource.(*types.VPCEndpointManifest); ok {
			found = true
			assert.NotEmpty(t, vpcEndpoint.Spec.ProjectName, "VPCEndpoint should have projectName")
			assert.Contains(t, []string{"AWS", "AZURE", "GCP"}, vpcEndpoint.Spec.CloudProvider,
				"VPCEndpoint should have valid cloudProvider")
			assert.NotEmpty(t, vpcEndpoint.Spec.Region, "VPCEndpoint should have region")
		}
	}
	assert.True(t, found, "Should find VPCEndpoint resource in file: %s", yamlFile)
}

func validateClusterSchema(t *testing.T, yamlFile string) {
	// Can be either standalone or in ApplyDocument
	config, err := apply.LoadConfiguration(yamlFile)
	if err == nil && types.ResourceKind(config.Kind) == types.KindProject {
		// Project with embedded clusters
		for _, cluster := range config.Spec.Clusters {
			validateClusterFields(t, cluster.Provider, cluster.Region, cluster.InstanceSize, yamlFile)
		}
		return
	}

	// Try as ApplyDocument
	document, err := apply.LoadApplyDocument(yamlFile)
	require.NoError(t, err, "Should load as ApplyDocument: %s", yamlFile)

	found := false
	for _, resource := range document.Resources {
		if cluster, ok := resource.(*types.ClusterManifest); ok {
			found = true
			validateClusterFields(t, cluster.Spec.Provider, cluster.Spec.Region, cluster.Spec.InstanceSize, yamlFile)
		}
	}
	assert.True(t, found, "Should find Cluster resource in file: %s", yamlFile)
}

func validateClusterFields(t *testing.T, provider, region, instanceSize, yamlFile string) {
	assert.Contains(t, []string{"AWS", "GCP", "AZURE"}, provider,
		"Cluster should have valid provider in %s", yamlFile)
	assert.NotEmpty(t, region, "Cluster should have region in %s", yamlFile)
	assert.NotEmpty(t, instanceSize, "Cluster should have instanceSize in %s", yamlFile)

	// Validate common instance sizes
	validSizes := []string{"M0", "M2", "M5", "M10", "M20", "M30", "M40", "M50", "M60", "M80", "M140", "M200", "M300"}
	assert.Contains(t, validSizes, instanceSize, "Cluster should have valid instanceSize in %s", yamlFile)
}

func validateDatabaseRoleSchema(t *testing.T, yamlFile string) {
	document, err := apply.LoadApplyDocument(yamlFile)
	require.NoError(t, err, "Should load ApplyDocument: %s", yamlFile)

	found := false
	for _, resource := range document.Resources {
		if role, ok := resource.(*types.DatabaseRoleManifest); ok {
			found = true
			assert.NotEmpty(t, role.Spec.RoleName, "DatabaseRole should have roleName")
			assert.NotEmpty(t, role.Spec.DatabaseName, "DatabaseRole should have databaseName")

			// Should have either privileges or inherited roles
			hasPrivileges := len(role.Spec.Privileges) > 0
			hasInheritedRoles := len(role.Spec.InheritedRoles) > 0
			assert.True(t, hasPrivileges || hasInheritedRoles,
				"DatabaseRole should have either privileges or inheritedRoles")

			// Validate privilege structure
			for _, privilege := range role.Spec.Privileges {
				assert.NotEmpty(t, privilege.Actions, "Privilege should have actions")
				assert.NotEmpty(t, privilege.Resource.Database, "Privilege resource should have database")
			}
		}
	}
	assert.True(t, found, "Should find DatabaseRole resource in file: %s", yamlFile)
}

// Helper functions

func getProjectRoot(t *testing.T) string {
	// Get the directory of this test file
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "Should be able to get test file path")

	// Navigate up to project root (test/integration/yaml -> ../../..)
	testDir := filepath.Dir(filename)
	projectRoot := filepath.Join(testDir, "..", "..", "..")
	absPath, err := filepath.Abs(projectRoot)
	require.NoError(t, err, "Should be able to get absolute path")

	return absPath
}

func findYAMLFiles(dir string) ([]string, error) {
	var yamlFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			yamlFiles = append(yamlFiles, path)
		}

		return nil
	})

	return yamlFiles, err
}
