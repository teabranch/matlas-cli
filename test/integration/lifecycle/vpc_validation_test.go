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

// TestVPCEndpointLifecycleValidation tests VPC Endpoint functionality without making API calls
func TestVPCEndpointLifecycleValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VPC endpoint lifecycle validation test in short mode")
	}

	t.Run("ValidateBasicVPCEndpoint", func(t *testing.T) {
		testVPCEndpointValidation(t, "basic")
	})

	t.Run("ValidateMultiProviderVPCEndpoints", func(t *testing.T) {
		testVPCEndpointValidation(t, "multi-provider")
	})
}

func testVPCEndpointValidation(t *testing.T, endpointType string) {
	timestamp := time.Now().Unix()
	tmpDir := t.TempDir()

	var yamlContent string
	var configFile string

	switch endpointType {
	case "basic":
		configFile = filepath.Join(tmpDir, "vpc-basic.yaml")
		yamlContent = fmt.Sprintf(`apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: vpc-endpoint-test-basic
  labels:
    test: vpc-endpoints-lifecycle
    timestamp: "%d"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: test-vpc-endpoint-%d
      labels:
        provider: aws
        environment: test
    spec:
      projectName: "test-project"
      cloudProvider: "AWS"
      region: "us-east-1"`, timestamp, timestamp)

	case "multi-provider":
		configFile = filepath.Join(tmpDir, "vpc-multi-provider.yaml")
		yamlContent = fmt.Sprintf(`apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: vpc-endpoint-test-multi
  labels:
    test: vpc-endpoints-lifecycle
    timestamp: "%d"
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: aws-vpc-endpoint-%d
    spec:
      projectName: "test-project"
      cloudProvider: "AWS"
      region: "us-east-1"
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: azure-vpc-endpoint-%d
    spec:
      projectName: "test-project"
      cloudProvider: "AZURE"
      region: "eastus"
  - apiVersion: matlas.mongodb.com/v1
    kind: VPCEndpoint
    metadata:
      name: gcp-vpc-endpoint-%d
    spec:
      projectName: "test-project"
      cloudProvider: "GCP"
      region: "us-central1"`, timestamp, timestamp, timestamp, timestamp)
	}

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err, "Should write test YAML file")

	// Test VPCEndpoint in ApplyDocument using ConfigurationLoader
	loader := apply.NewConfigurationLoader(apply.DefaultLoaderOptions())
	result, err := loader.LoadApplyDocument(configFile)
	require.NoError(t, err, "Should load YAML as ApplyDocument")
	require.NotNil(t, result, "LoadResult should not be nil")

	document, ok := result.Config.(*types.ApplyDocument)
	require.True(t, ok, "Should be ApplyDocument type")

	// Verify document structure
	assert.Equal(t, types.KindApplyDocument, document.Kind, "Should be ApplyDocument")
	assert.NotEmpty(t, document.Resources, "Should have resources")

	// Count VPCEndpoint resources using field access
	var vpcEndpointCount int
	for _, resource := range document.Resources {
		if resource.Kind == types.KindVPCEndpoint {
			vpcEndpointCount++
			// Basic validation that we can access the resource
			assert.NotEmpty(t, resource.Metadata.Name, "VPCEndpoint should have name")
		}
	}

	// Verify expected resource counts
	switch endpointType {
	case "basic":
		assert.Equal(t, 1, vpcEndpointCount, "Basic config should have 1 VPCEndpoint")
	case "multi-provider":
		assert.Equal(t, 3, vpcEndpointCount, "Multi-provider config should have 3 VPCEndpoints")
	}

	// Validate through apply validator
	validationResult := apply.ValidateApplyDocument(document, apply.DefaultValidatorOptions())

	// Log validation results for debugging
	if len(validationResult.Warnings) > 0 {
		t.Logf("Validation warnings for %s VPC endpoint:", endpointType)
		for _, warning := range validationResult.Warnings {
			t.Logf("  - %s: %s", warning.Path, warning.Message)
		}
	}
	if len(validationResult.Errors) > 0 {
		t.Logf("Validation errors for %s VPC endpoint:", endpointType)
		for _, err := range validationResult.Errors {
			t.Logf("  - %s: %s", err.Path, err.Message)
		}
	}

	// Should pass validation
	assert.Empty(t, validationResult.Errors, "%s VPC endpoint configuration should pass validation", endpointType)
}
