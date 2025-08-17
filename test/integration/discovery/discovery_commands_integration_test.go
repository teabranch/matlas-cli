//go:build integration
// +build integration

package discovery

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// DiscoveryCommandTestConfig holds configuration for discovery command integration tests
type DiscoveryCommandTestConfig struct {
	PublicKey     string
	PrivateKey    string
	OrgID         string
	ProjectID     string
	BinaryPath    string
	TempDirPath   string
	Timeout       time.Duration
}

// DiscoveryCommandTestEnvironment provides shared test setup for command-level testing
type DiscoveryCommandTestEnvironment struct {
	Config    DiscoveryCommandTestConfig
	TempFiles []string
}

func setupDiscoveryCommandIntegrationTest(t *testing.T) *DiscoveryCommandTestEnvironment {
	t.Helper()

	// Skip if short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load configuration from environment
	config := DiscoveryCommandTestConfig{
		PublicKey:  os.Getenv("ATLAS_PUB_KEY"),
		PrivateKey: os.Getenv("ATLAS_API_KEY"),
		OrgID:      os.Getenv("ATLAS_ORG_ID"),
		ProjectID:  os.Getenv("ATLAS_PROJECT_ID"),
		BinaryPath: findMatlasExecutable(t),
		Timeout:    3 * time.Minute,
	}

	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "discovery-cmd-test-*")
	require.NoError(t, err)
	config.TempDirPath = tempDir

	if config.PublicKey == "" || config.PrivateKey == "" || config.ProjectID == "" {
		t.Skip("Atlas credentials not provided, skipping discovery command integration test")
	}

	env := &DiscoveryCommandTestEnvironment{
		Config:    config,
		TempFiles: []string{},
	}

	// Setup cleanup
	t.Cleanup(func() {
		env.cleanup(t)
	})

	return env
}

func (env *DiscoveryCommandTestEnvironment) cleanup(t *testing.T) {
	// Clean up temp files
	for _, file := range env.TempFiles {
		if err := os.Remove(file); err != nil {
			t.Logf("Warning: Failed to cleanup temp file %s: %v", file, err)
		}
	}

	// Clean up temp directory
	if env.Config.TempDirPath != "" {
		if err := os.RemoveAll(env.Config.TempDirPath); err != nil {
			t.Logf("Warning: Failed to cleanup temp directory %s: %v", env.Config.TempDirPath, err)
		}
	}
}

func (env *DiscoveryCommandTestEnvironment) createTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(env.Config.TempDirPath, name)
	err := os.WriteFile(path, content, 0644)
	require.NoError(t, err)

	env.TempFiles = append(env.TempFiles, path)
	return path
}

func (env *DiscoveryCommandTestEnvironment) runMatlasCommand(args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, env.Config.BinaryPath, args...)

	// Set environment variables for Atlas credentials
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ATLAS_PUB_KEY=%s", env.Config.PublicKey),
		fmt.Sprintf("ATLAS_API_KEY=%s", env.Config.PrivateKey),
		fmt.Sprintf("ATLAS_PROJECT_ID=%s", env.Config.ProjectID),
		fmt.Sprintf("ATLAS_ORG_ID=%s", env.Config.OrgID),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func findMatlasExecutable(t *testing.T) string {
	t.Helper()

	// Look for matlas binary in common locations
	candidates := []string{
		"./matlas",
		"../../../matlas",
		"matlas", // in PATH
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			abs, err := filepath.Abs(candidate)
			if err == nil {
				return abs
			}
			return candidate
		}
	}

	// Try to build if not found
	projectRoot := "../../../"
	buildCmd := exec.Command("go", "build", "-o", "matlas", ".")
	buildCmd.Dir = projectRoot
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to find or build matlas binary: %v", err)
	}

	builtBinary := filepath.Join(projectRoot, "matlas")
	if _, err := os.Stat(builtBinary); err != nil {
		t.Fatalf("Built matlas binary not found at %s", builtBinary)
	}

	abs, err := filepath.Abs(builtBinary)
	if err != nil {
		return builtBinary
	}
	return abs
}

// Test Discovery Command Basic Functionality
func TestDiscoveryCommand_BasicDiscovery_Integration(t *testing.T) {
	env := setupDiscoveryCommandIntegrationTest(t)

	t.Run("DiscoverProject", func(t *testing.T) {
		outputFile := env.createTempFile(t, "basic-discovery.yaml", []byte{})

		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--output-file", outputFile,
			"--verbose",
		)

		require.NoError(t, err, "Discovery command failed. Stdout: %s, Stderr: %s", stdout, stderr)

		// Verify output file was created and contains data
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Greater(t, len(content), 0, "Output file should not be empty")

		// Verify it's valid YAML
		var result map[string]interface{}
		err = yaml.Unmarshal(content, &result)
		require.NoError(t, err, "Output should be valid YAML")

		// Verify basic structure
		assert.Equal(t, "DiscoveredProject", result["kind"])
		assert.Contains(t, result, "metadata")
		assert.Contains(t, result, "project")

		t.Logf("Discovery completed successfully. Output file size: %d bytes", len(content))
	})

	t.Run("DiscoverProjectToStdout", func(t *testing.T) {
		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
		)

		require.NoError(t, err, "Discovery command failed. Stderr: %s", stderr)
		assert.Greater(t, len(stdout), 0, "Should produce output to stdout")

		// Verify it's valid YAML
		var result map[string]interface{}
		err = yaml.Unmarshal([]byte(stdout), &result)
		require.NoError(t, err, "Stdout should be valid YAML")

		assert.Equal(t, "DiscoveredProject", result["kind"])
	})

	t.Run("DiscoverWithJSON", func(t *testing.T) {
		outputFile := env.createTempFile(t, "discovery.json", []byte{})

		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--output", "json",
			"--output-file", outputFile,
		)

		require.NoError(t, err, "JSON discovery failed. Stdout: %s, Stderr: %s", stdout, stderr)

		// Verify output file contains valid JSON
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result map[string]interface{}
		err = yaml.Unmarshal(content, &result) // yaml.Unmarshal can handle JSON
		require.NoError(t, err, "Output should be valid JSON")

		assert.Equal(t, "DiscoveredProject", result["kind"])
	})
}

// Test Discovery with Conversion
func TestDiscoveryCommand_ConvertToApplyDocument_Integration(t *testing.T) {
	env := setupDiscoveryCommandIntegrationTest(t)

	t.Run("ConvertToApplyDocument", func(t *testing.T) {
		outputFile := env.createTempFile(t, "converted-apply.yaml", []byte{})

		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--convert-to-apply",
			"--output-file", outputFile,
			"--verbose",
		)

		require.NoError(t, err, "Conversion failed. Stdout: %s, Stderr: %s", stdout, stderr)

		// Verify output file was created
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Greater(t, len(content), 0)

		// Verify it's ApplyDocument format
		var result map[string]interface{}
		err = yaml.Unmarshal(content, &result)
		require.NoError(t, err)

		assert.Equal(t, "ApplyDocument", result["kind"])
		assert.Equal(t, "matlas.mongodb.com/v1", result["apiVersion"])
		assert.Contains(t, result, "metadata")
		assert.Contains(t, result, "resources")

		// Verify resources array exists and has content
		resources, ok := result["resources"].([]interface{})
		assert.True(t, ok, "Resources should be an array")
		assert.GreaterOrEqual(t, len(resources), 1, "Should have at least one resource")

		t.Logf("Converted to ApplyDocument with %d resources", len(resources))
	})

	t.Run("ValidateConvertedDocument", func(t *testing.T) {
		outputFile := env.createTempFile(t, "validate-convert.yaml", []byte{})

		// First, convert project to ApplyDocument
		_, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--convert-to-apply",
			"--output-file", outputFile,
		)

		require.NoError(t, err, "Conversion failed. Stderr: %s", stderr)

		// Then validate the converted document
		stdout, stderr, err := env.runMatlasCommand(
			"infra", "validate",
			"-f", outputFile,
			"--project-id", env.Config.ProjectID,
		)

		require.NoError(t, err, "Validation failed. Stdout: %s, Stderr: %s", stdout, stderr)

		// Validation success indicates the converted document is well-formed
		t.Logf("Converted document validation successful")
	})
}

// Test Resource-specific Discovery
func TestDiscoveryCommand_ResourceSpecific_Integration(t *testing.T) {
	env := setupDiscoveryCommandIntegrationTest(t)

	t.Run("DiscoverClustersOnly", func(t *testing.T) {
		outputFile := env.createTempFile(t, "clusters-only.yaml", []byte{})

		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--include", "clusters",
			"--output-file", outputFile,
		)

		require.NoError(t, err, "Cluster discovery failed. Stdout: %s, Stderr: %s", stdout, stderr)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result map[string]interface{}
		err = yaml.Unmarshal(content, &result)
		require.NoError(t, err)

		// Should have clusters but not users or network access
		assert.Contains(t, result, "clusters")
		
		// Check that other resource types are not present or empty
		if users, exists := result["databaseUsers"]; exists {
			assert.Nil(t, users, "Database users should not be included")
		}
		if network, exists := result["networkAccess"]; exists {
			assert.Nil(t, network, "Network access should not be included")
		}
	})

	t.Run("DiscoverUsersOnly", func(t *testing.T) {
		outputFile := env.createTempFile(t, "users-only.yaml", []byte{})

		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--include", "users",
			"--output-file", outputFile,
		)

		require.NoError(t, err, "User discovery failed. Stdout: %s, Stderr: %s", stdout, stderr)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result map[string]interface{}
		err = yaml.Unmarshal(content, &result)
		require.NoError(t, err)

		// Should have users but not clusters or network access
		assert.Contains(t, result, "databaseUsers")
		
		// Check that other resource types are not present or empty
		if clusters, exists := result["clusters"]; exists {
			assert.Nil(t, clusters, "Clusters should not be included")
		}
		if network, exists := result["networkAccess"]; exists {
			assert.Nil(t, network, "Network access should not be included")
		}
	})

	t.Run("ExcludeNetworkAccess", func(t *testing.T) {
		outputFile := env.createTempFile(t, "exclude-network.yaml", []byte{})

		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--exclude", "network",
			"--output-file", outputFile,
		)

		require.NoError(t, err, "Exclude discovery failed. Stdout: %s, Stderr: %s", stdout, stderr)

		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)

		var result map[string]interface{}
		err = yaml.Unmarshal(content, &result)
		require.NoError(t, err)

		// Should not have network access
		if network, exists := result["networkAccess"]; exists {
			assert.Nil(t, network, "Network access should be excluded")
		}

		// Should still have project and other resources
		assert.Contains(t, result, "project")
	})
}

// Test Discovery Caching
func TestDiscoveryCommand_Caching_Integration(t *testing.T) {
	env := setupDiscoveryCommandIntegrationTest(t)

	t.Run("CacheEnabled", func(t *testing.T) {
		outputFile1 := env.createTempFile(t, "cache-test1.yaml", []byte{})
		outputFile2 := env.createTempFile(t, "cache-test2.yaml", []byte{})

		// First discovery (populate cache)
		start1 := time.Now()
		stdout1, stderr1, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--output-file", outputFile1,
			"--cache-stats",
			"--verbose",
		)
		duration1 := time.Since(start1)

		require.NoError(t, err, "First discovery failed. Stdout: %s, Stderr: %s", stdout1, stderr1)

		// Second discovery (should use cache)
		start2 := time.Now()
		stdout2, stderr2, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--output-file", outputFile2,
			"--cache-stats",
			"--verbose",
		)
		duration2 := time.Since(start2)

		require.NoError(t, err, "Second discovery failed. Stdout: %s, Stderr: %s", stdout2, stderr2)

		t.Logf("First discovery: %v, Second discovery: %v", duration1, duration2)

		// Cache stats should be in stderr
		if strings.Contains(stderr2, "Cache stats:") {
			t.Logf("Cache stats found in output")
		}

		// Verify both outputs are similar (same project state)
		content1, err := os.ReadFile(outputFile1)
		require.NoError(t, err)
		content2, err := os.ReadFile(outputFile2)
		require.NoError(t, err)

		var result1, result2 map[string]interface{}
		err = yaml.Unmarshal(content1, &result1)
		require.NoError(t, err)
		err = yaml.Unmarshal(content2, &result2)
		require.NoError(t, err)

		// Fingerprints should be the same (same project state)
		if metadata1, ok := result1["metadata"].(map[string]interface{}); ok {
			if metadata2, ok := result2["metadata"].(map[string]interface{}); ok {
				fingerprint1 := metadata1["fingerprint"]
				fingerprint2 := metadata2["fingerprint"]
				if fingerprint1 != nil && fingerprint2 != nil {
					assert.Equal(t, fingerprint1, fingerprint2, "Fingerprints should match")
				}
			}
		}
	})

	t.Run("CacheDisabled", func(t *testing.T) {
		outputFile := env.createTempFile(t, "no-cache-test.yaml", []byte{})

		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--no-cache",
			"--output-file", outputFile,
			"--verbose",
		)

		require.NoError(t, err, "No-cache discovery failed. Stdout: %s, Stderr: %s", stdout, stderr)

		// Verify output was created
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Greater(t, len(content), 0)

		var result map[string]interface{}
		err = yaml.Unmarshal(content, &result)
		require.NoError(t, err)
		assert.Equal(t, "DiscoveredProject", result["kind"])
	})
}

// Test Discovery Error Handling
func TestDiscoveryCommand_ErrorHandling_Integration(t *testing.T) {
	env := setupDiscoveryCommandIntegrationTest(t)

	t.Run("InvalidProjectID", func(t *testing.T) {
		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", "invalid-project-id-123",
		)

		// Should fail with invalid project ID
		assert.Error(t, err, "Should fail with invalid project ID")
		assert.Contains(t, stderr, "failed", "Error message should indicate failure")

		t.Logf("Expected error occurred: %v", err)
		t.Logf("Error output: %s", stderr)
	})

	t.Run("MissingProjectID", func(t *testing.T) {
		stdout, stderr, err := env.runMatlasCommand(
			"discover",
		)

		// Should fail without project ID
		assert.Error(t, err, "Should fail without project ID")
		
		// Error should mention missing project ID
		combined := stdout + stderr
		assert.True(t, 
			strings.Contains(combined, "project-id") || strings.Contains(combined, "required"),
			"Error should mention project-id requirement. Output: %s", combined)
	})

	t.Run("InvalidOutputFormat", func(t *testing.T) {
		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--output", "invalid-format",
		)

		// Should fail with invalid output format
		assert.Error(t, err, "Should fail with invalid output format")
		
		combined := stdout + stderr
		assert.Contains(t, combined, "unsupported", "Error should mention unsupported format")
	})
}

// Test Discovery with Timeout
func TestDiscoveryCommand_Timeout_Integration(t *testing.T) {
	env := setupDiscoveryCommandIntegrationTest(t)

	t.Run("ShortTimeout", func(t *testing.T) {
		// Use very short timeout to test timeout handling
		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--timeout", "1ms", // Very short timeout
		)

		// Should timeout
		assert.Error(t, err, "Should timeout with very short timeout")
		
		combined := stdout + stderr
		if !strings.Contains(combined, "timeout") && !strings.Contains(combined, "context") {
			t.Logf("Warning: Timeout error may not be explicitly mentioned. Output: %s", combined)
		}
	})

	t.Run("ReasonableTimeout", func(t *testing.T) {
		outputFile := env.createTempFile(t, "timeout-test.yaml", []byte{})

		stdout, stderr, err := env.runMatlasCommand(
			"discover",
			"--project-id", env.Config.ProjectID,
			"--timeout", "2m",
			"--output-file", outputFile,
		)

		require.NoError(t, err, "Should complete with reasonable timeout. Stdout: %s, Stderr: %s", stdout, stderr)

		// Verify output was created
		content, err := os.ReadFile(outputFile)
		require.NoError(t, err)
		assert.Greater(t, len(content), 0)
	})
}




