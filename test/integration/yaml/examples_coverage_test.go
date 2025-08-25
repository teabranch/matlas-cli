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

// ExampleKindMapping tracks which YAML kinds are found in which example files
type ExampleKindMapping struct {
	Kind  types.ResourceKind
	Files []string
}

// TestExamplesCoverage ensures every documented YAML kind has at least one working example
func TestExamplesCoverage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping examples coverage test in short mode")
	}

	projectRoot := getProjectRootFromTest(t)
	examplesDir := filepath.Join(projectRoot, "examples")
	
	// Ensure examples directory exists
	_, err := os.Stat(examplesDir)
	require.NoError(t, err, "Examples directory should exist: %s", examplesDir)

	// Map of documented kinds that MUST have examples
	documentedKinds := map[types.ResourceKind]bool{
		types.KindProject:      false,
		types.KindCluster:      false,
		types.KindDatabaseUser: false,
		types.KindDatabaseRole: false,
		types.KindNetworkAccess: false,
		types.KindSearchIndex:  false,
		types.KindVPCEndpoint:  false,
		types.KindApplyDocument: false,
	}

	// Track coverage for each kind
	kindsCoverage := make(map[types.ResourceKind][]string)

	// Find all YAML files and analyze them
	yamlFiles, err := findYAMLFilesInDir(examplesDir)
	require.NoError(t, err, "Should be able to find YAML files in examples")
	require.NotEmpty(t, yamlFiles, "Should have at least one example YAML file")

	// Analyze each YAML file for kinds
	for _, yamlFile := range yamlFiles {
		t.Run(fmt.Sprintf("Analyze_%s", filepath.Base(yamlFile)), func(t *testing.T) {
			kinds := analyzeYAMLFileForKinds(t, yamlFile)
			
			for _, kind := range kinds {
				kindsCoverage[kind] = append(kindsCoverage[kind], yamlFile)
				if _, isDocumented := documentedKinds[kind]; isDocumented {
					documentedKinds[kind] = true
				}
			}
		})
	}

	// Verify coverage for all documented kinds
	t.Run("VerifyKindsCoverage", func(t *testing.T) {
		var missingKinds []types.ResourceKind
		
		for kind, found := range documentedKinds {
			if !found {
				missingKinds = append(missingKinds, kind)
			}
		}

		if len(missingKinds) > 0 {
			t.Errorf("Missing examples for documented YAML kinds: %v", missingKinds)
			t.Log("Required examples for:")
			for _, kind := range missingKinds {
				t.Logf("  - %s: Need at least one example file", kind)
			}
		}

		// Log coverage summary
		t.Log("Examples coverage summary:")
		for kind, files := range kindsCoverage {
			t.Logf("  - %s: %d examples (%v)", kind, len(files), 
				func() []string {
					var basenames []string
					for _, file := range files {
						basenames = append(basenames, filepath.Base(file))
					}
					return basenames
				}())
		}
	})

	// Verify example quality - each kind should have specific examples
	t.Run("VerifyExampleQuality", func(t *testing.T) {
		testCases := []struct {
			kind        types.ResourceKind
			minExamples int
			description string
		}{
			{
				kind:        types.KindSearchIndex,
				minExamples: 2, // Should have basic and vector search examples
				description: "SearchIndex should have both basic and vector search examples",
			},
			{
				kind:        types.KindVPCEndpoint,
				minExamples: 1,
				description: "VPCEndpoint should have at least one example",
			},
			{
				kind:        types.KindDatabaseRole,
				minExamples: 1,
				description: "DatabaseRole should have custom role examples",
			},
			{
				kind:        types.KindCluster,
				minExamples: 2, // Should have basic and advanced examples
				description: "Cluster should have multiple configuration examples",
			},
		}

		for _, tc := range testCases {
			if files, ok := kindsCoverage[tc.kind]; ok {
				assert.GreaterOrEqual(t, len(files), tc.minExamples,
					"%s: %s", tc.kind, tc.description)
			} else {
				t.Errorf("No examples found for %s", tc.kind)
			}
		}
	})
}

// TestExamplesLoadability ensures all example files can be loaded without errors
func TestExamplesLoadability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping examples loadability test in short mode")
	}

	projectRoot := getProjectRootFromTest(t)
	examplesDir := filepath.Join(projectRoot, "examples")
	
	yamlFiles, err := findYAMLFilesInDir(examplesDir)
	require.NoError(t, err, "Should find YAML files")

	for _, yamlFile := range yamlFiles {
		t.Run(filepath.Base(yamlFile), func(t *testing.T) {
			// Try to load as ApplyConfig first
			config, configErr := apply.LoadConfiguration(yamlFile)
			
			// Try to load as ApplyDocument
			document, docErr := apply.LoadApplyDocument(yamlFile)
			
			// At least one should succeed
			assert.True(t, configErr == nil || docErr == nil,
				"File should be loadable either as ApplyConfig or ApplyDocument: %s", yamlFile)

			// If loaded as config, verify basic structure
			if configErr == nil {
				assert.NotEmpty(t, config.APIVersion, "Should have apiVersion")
				assert.NotEmpty(t, config.Kind, "Should have kind")
				assert.NotEmpty(t, config.Metadata.Name, "Should have metadata.name")
			}

			// If loaded as document, verify resource structure
			if docErr == nil {
				assert.NotEmpty(t, document.APIVersion, "Document should have apiVersion")
				assert.Equal(t, types.KindApplyDocument, document.Kind, "Document should be ApplyDocument kind")
				assert.NotEmpty(t, document.Resources, "Document should have resources")
			}
		})
	}
}

// TestExamplesConsistency verifies that examples follow consistent patterns
func TestExamplesConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping examples consistency test in short mode")
	}

	projectRoot := getProjectRootFromTest(t)
	examplesDir := filepath.Join(projectRoot, "examples")
	
	yamlFiles, err := findYAMLFilesInDir(examplesDir)
	require.NoError(t, err, "Should find YAML files")

	for _, yamlFile := range yamlFiles {
		t.Run(filepath.Base(yamlFile), func(t *testing.T) {
			// Read file content to check consistency patterns
			content, err := os.ReadFile(yamlFile)
			require.NoError(t, err, "Should be able to read file")
			
			contentStr := string(content)

			// Should use consistent API version
			assert.Contains(t, contentStr, "apiVersion: matlas.mongodb.com/v1",
				"Should use standard API version: %s", yamlFile)

			// Should not contain real credentials
			problematicPatterns := []string{
				"password: \"real-password",
				"apiKey:",
				"privateKey:",
				"@mongodb.com",
			}

			for _, pattern := range problematicPatterns {
				assert.NotContains(t, strings.ToLower(contentStr), strings.ToLower(pattern),
					"Example should not contain real credentials pattern '%s': %s", pattern, yamlFile)
			}

			// Should use placeholder values
			if strings.Contains(contentStr, "projectName:") || strings.Contains(contentStr, "organizationId:") {
				placeholderPatterns := []string{
					"your-project",
					"your-org",
					"${",
					"example",
					"test",
				}
				
				hasPlaceholder := false
				for _, pattern := range placeholderPatterns {
					if strings.Contains(strings.ToLower(contentStr), pattern) {
						hasPlaceholder = true
						break
					}
				}
				
				assert.True(t, hasPlaceholder,
					"Example with project/org references should use placeholder values: %s", yamlFile)
			}
		})
	}
}

// analyzeYAMLFileForKinds extracts all YAML kinds present in a file
func analyzeYAMLFileForKinds(t *testing.T, yamlFile string) []types.ResourceKind {
	var kinds []types.ResourceKind

	// Try loading as ApplyConfig (single resource)
	if config, err := apply.LoadConfiguration(yamlFile); err == nil {
		kinds = append(kinds, types.ResourceKind(config.Kind))
		
		// If it's a Project, also check for embedded resources
		if types.ResourceKind(config.Kind) == types.KindProject {
			if len(config.Spec.Clusters) > 0 {
				kinds = append(kinds, types.KindCluster)
			}
			if len(config.Spec.DatabaseUsers) > 0 {
				kinds = append(kinds, types.KindDatabaseUser)
			}
			if len(config.Spec.NetworkAccess) > 0 {
				kinds = append(kinds, types.KindNetworkAccess)
			}
		}
	}

	// Try loading as ApplyDocument (multiple resources)
	if document, err := apply.LoadApplyDocument(yamlFile); err == nil {
		kinds = append(kinds, types.KindApplyDocument)
		
		// Analyze each resource in the document
		for _, resource := range document.Resources {
			switch resource.(type) {
			case *types.ClusterManifest:
				kinds = append(kinds, types.KindCluster)
			case *types.DatabaseUserManifest:
				kinds = append(kinds, types.KindDatabaseUser)
			case *types.DatabaseRoleManifest:
				kinds = append(kinds, types.KindDatabaseRole)
			case *types.NetworkAccessManifest:
				kinds = append(kinds, types.KindNetworkAccess)
			case *types.SearchIndexManifest:
				kinds = append(kinds, types.KindSearchIndex)
			case *types.VPCEndpointManifest:
				kinds = append(kinds, types.KindVPCEndpoint)
			case *types.ProjectManifest:
				kinds = append(kinds, types.KindProject)
			}
		}
	}

	// Remove duplicates
	uniqueKinds := make([]types.ResourceKind, 0)
	seen := make(map[types.ResourceKind]bool)
	for _, kind := range kinds {
		if !seen[kind] {
			uniqueKinds = append(uniqueKinds, kind)
			seen[kind] = true
		}
	}

	return uniqueKinds
}

// Helper functions

func getProjectRootFromTest(t *testing.T) string {
	_, filename, _, ok := runtime.Caller(1)
	require.True(t, ok, "Should be able to get test file path")
	
	testDir := filepath.Dir(filename)
	projectRoot := filepath.Join(testDir, "..", "..", "..")
	absPath, err := filepath.Abs(projectRoot)
	require.NoError(t, err, "Should be able to get absolute path")
	
	return absPath
}

func findYAMLFilesInDir(dir string) ([]string, error) {
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