//go:build integration
// +build integration

package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

// TestSearchLifecycleValidation tests Atlas Search functionality without making API calls
func TestSearchLifecycleValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping search lifecycle validation test in short mode")
	}

	t.Run("ValidateBasicSearchIndex", func(t *testing.T) {
		testSearchIndexValidation(t, "basic")
	})

	t.Run("ValidateVectorSearchIndex", func(t *testing.T) {
		testSearchIndexValidation(t, "vector")
	})

	t.Run("ValidateMultiResourceSearchDocument", func(t *testing.T) {
		testMultiResourceSearchValidation(t)
	})
}

func testSearchIndexValidation(t *testing.T, indexType string) {
	timestamp := time.Now().Unix()
	tmpDir := t.TempDir()
	
	var yamlContent string
	var indexName string
	
	switch indexType {
	case "basic":
		indexName = fmt.Sprintf("test-search-%d", timestamp)
		yamlContent = fmt.Sprintf(`apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: search-test-basic
  labels:
    test: search-lifecycle
    timestamp: "%d"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: %s
      labels:
        type: basic-search
    spec:
      projectName: "test-project"
      clusterName: "test-cluster"
      databaseName: "sample_mflix"
      collectionName: "movies"
      indexName: "%s"
      indexType: "search"
      definition:
        mappings:
          dynamic: true`, timestamp, indexName, indexName)
	case "vector":
		indexName = fmt.Sprintf("test-vector-%d", timestamp)
		yamlContent = fmt.Sprintf(`apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: search-test-vector
  labels:
    test: search-lifecycle
    timestamp: "%d"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: %s
      labels:
        type: vector-search
    spec:
      projectName: "test-project"
      clusterName: "test-cluster"
      databaseName: "sample_mflix"
      collectionName: "movies"
      indexName: "%s"
      indexType: "vectorSearch"
      definition:
        fields:
          - type: "vector"
            path: "plot_embedding"
            numDimensions: 1536
            similarity: "cosine"`, timestamp, indexName, indexName)
	}
	
	configFile := filepath.Join(tmpDir, fmt.Sprintf("search-%s.yaml", indexType))
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err, "Should write test YAML file")

	// Test YAML loading using ConfigurationLoader (matches working pattern from loader tests)
	loader := apply.NewConfigurationLoader(apply.DefaultLoaderOptions())
	result, err := loader.LoadApplyDocument(configFile)
	require.NoError(t, err, "Should load YAML as ApplyDocument")
	require.NotNil(t, result, "LoadResult should not be nil")
	
	document, ok := result.Config.(*types.ApplyDocument)
	require.True(t, ok, "Should be ApplyDocument type")
	
	// Verify document structure
	assert.Equal(t, types.KindApplyDocument, document.Kind, "Should be ApplyDocument")
	assert.Len(t, document.Resources, 1, "Should have 1 resource")
	
	// Find SearchIndex resource using field access
	searchIndexFound := false
	for _, resource := range document.Resources {
		if resource.Kind == types.KindSearchIndex {
			searchIndexFound = true
			// Basic validation that we can access the resource
			assert.NotEmpty(t, resource.Metadata.Name, "SearchIndex should have name")
			break
		}
	}
	assert.True(t, searchIndexFound, "Should find SearchIndex resource")
	
	// Validate through apply validator (without making API calls)
	validationResult := apply.ValidateApplyDocument(document, apply.DefaultValidatorOptions())
	
	// Log validation results for debugging
	if len(validationResult.Warnings) > 0 {
		t.Logf("Validation warnings for %s search index:", indexType)
		for _, warning := range validationResult.Warnings {
			t.Logf("  - %s: %s", warning.Path, warning.Message)
		}
	}
	if len(validationResult.Errors) > 0 {
		t.Logf("Validation errors for %s search index:", indexType)
		for _, err := range validationResult.Errors {
			t.Logf("  - %s: %s", err.Path, err.Message)
		}
	}
	
	// Should pass validation
	assert.Empty(t, validationResult.Errors, "%s search index should pass validation", indexType)
}

func testMultiResourceSearchValidation(t *testing.T) {
	timestamp := time.Now().Unix()
	tmpDir := t.TempDir()
	
	yamlContent := fmt.Sprintf(`apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: search-test-multi
  labels:
    test: search-lifecycle
    timestamp: "%d"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: movies-text-%d
    spec:
      projectName: "test-project"
      clusterName: "test-cluster"
      databaseName: "sample_mflix"
      collectionName: "movies"
      indexName: "movies-text-%d"
      indexType: "search"
      definition:
        mappings:
          fields:
            title:
              type: "string"
              analyzer: "lucene.standard"
            plot:
              type: "string"
              analyzer: "lucene.standard"
            year:
              type: "number"
  - apiVersion: matlas.mongodb.com/v1
    kind: SearchIndex
    metadata:
      name: comments-search-%d
    spec:
      projectName: "test-project"
      clusterName: "test-cluster"
      databaseName: "sample_mflix"
      collectionName: "comments"
      indexName: "comments-search-%d"
      indexType: "search"
      definition:
        mappings:
          fields:
            text:
              type: "string"
              analyzer: "lucene.standard"
            date:
              type: "date"`, timestamp, timestamp, timestamp, timestamp, timestamp)
	
	configFile := filepath.Join(tmpDir, "search-multi.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err, "Should write test YAML file")

	// Test YAML loading
	loader := apply.NewConfigurationLoader(apply.DefaultLoaderOptions())
	result, err := loader.LoadApplyDocument(configFile)
	require.NoError(t, err, "Should load multi-resource YAML as ApplyDocument")
	require.NotNil(t, result, "LoadResult should not be nil")
	
	document, ok := result.Config.(*types.ApplyDocument)
	require.True(t, ok, "Should be ApplyDocument type")
	
	// Verify document structure
	assert.Equal(t, types.KindApplyDocument, document.Kind, "Should be ApplyDocument")
	assert.Len(t, document.Resources, 2, "Should have 2 resources")
	
	// Count SearchIndex resources
	var searchIndexCount int
	for _, resource := range document.Resources {
		if resource.Kind == types.KindSearchIndex {
			searchIndexCount++
		}
	}
	assert.Equal(t, 2, searchIndexCount, "Should have 2 SearchIndex resources")
	
	// Validate through apply validator
	validationResult := apply.ValidateApplyDocument(document, apply.DefaultValidatorOptions())
	
	// Should pass validation
	assert.Empty(t, validationResult.Errors, "Multi-resource search document should pass validation")
}