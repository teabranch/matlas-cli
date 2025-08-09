//go:build integration
// +build integration

package apply

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ApplyCommandTestConfig holds configuration for apply command integration tests
type ApplyCommandTestConfig struct {
	PublicKey       string
	PrivateKey      string
	OrgID           string
	ProjectID       string
	BinaryPath      string
	Timeout         time.Duration
	TestConfigDir   string
	TestClusterName string
}

// ApplyCommandTestEnvironment provides shared test setup for apply command testing
type ApplyCommandTestEnvironment struct {
	Config      ApplyCommandTestConfig
	ConfigFiles []string // Track created config files for cleanup
}

func setupApplyCommandIntegrationTest(t *testing.T) *ApplyCommandTestEnvironment {
	t.Helper()

	// Skip if short tests
	if testing.Short() {
		t.Skip("Skipping apply command integration test in short mode")
	}

	// Create temporary directory for test configs
	tempDir, err := ioutil.TempDir("", "apply-test-configs")
	require.NoError(t, err)

	// Load configuration from environment
	config := ApplyCommandTestConfig{
		PublicKey:       os.Getenv("ATLAS_PUBLIC_KEY"),
		PrivateKey:      os.Getenv("ATLAS_PRIVATE_KEY"),
		OrgID:           os.Getenv("ATLAS_ORG_ID"),
		ProjectID:       os.Getenv("ATLAS_PROJECT_ID"),
		BinaryPath:      "./matlas",
		Timeout:         180 * time.Second, // Apply operations can be very slow
		TestConfigDir:   tempDir,
		TestClusterName: fmt.Sprintf("apply-test-%d", time.Now().Unix()),
	}

	// Check if binary exists, if not try to build it
	if _, err := os.Stat(config.BinaryPath); os.IsNotExist(err) {
		t.Log("Binary not found, attempting to build matlas binary...")
		buildCmd := exec.Command("go", "build", "-o", "matlas", ".")
		buildCmd.Dir = "../../../" // Go back to project root
		if err := buildCmd.Run(); err != nil {
			t.Fatalf("Failed to build matlas binary: %v", err)
		}
		config.BinaryPath = "../../../matlas"
	}

	// Verify required Atlas credentials
	if config.PublicKey == "" || config.PrivateKey == "" {
		t.Skip("ATLAS_PUBLIC_KEY and ATLAS_PRIVATE_KEY not provided, skipping apply command integration tests")
	}

	if config.OrgID == "" || config.ProjectID == "" {
		t.Skip("ATLAS_ORG_ID and ATLAS_PROJECT_ID not provided, skipping apply command integration tests")
	}

	return &ApplyCommandTestEnvironment{
		Config:      config,
		ConfigFiles: make([]string, 0),
	}
}

func (env *ApplyCommandTestEnvironment) cleanup(t *testing.T) {
	t.Log("Starting cleanup of test configuration files...")

	// Remove all created config files
	for _, configFile := range env.ConfigFiles {
		os.Remove(configFile)
	}

	// Remove test config directory
	os.RemoveAll(env.Config.TestConfigDir)

	t.Log("Cleanup completed")
}

func (env *ApplyCommandTestEnvironment) runCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, env.Config.BinaryPath, args...)
	cmd.Dir = "../../../" // Run from project root

	// Set Atlas environment variables
	cmd.Env = append(os.Environ(),
		"ATLAS_PUBLIC_KEY="+env.Config.PublicKey,
		"ATLAS_PRIVATE_KEY="+env.Config.PrivateKey,
		"ATLAS_ORG_ID="+env.Config.OrgID,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	t.Logf("Command: %s %s", env.Config.BinaryPath, strings.Join(args, " "))
	t.Logf("Stdout: %s", stdout.String())
	if stderr.Len() > 0 {
		t.Logf("Stderr: %s", stderr.String())
	}

	return stdout.String(), err
}

func (env *ApplyCommandTestEnvironment) createConfigFile(t *testing.T, content string) string {
	t.Helper()

	configFile := filepath.Join(env.Config.TestConfigDir, fmt.Sprintf("config-%d.yaml", time.Now().UnixNano()))
	err := ioutil.WriteFile(configFile, []byte(content), 0644)
	require.NoError(t, err, "Should be able to create config file")

	env.ConfigFiles = append(env.ConfigFiles, configFile)
	return configFile
}

func TestApplyCommands_Validate_Integration(t *testing.T) {
	env := setupApplyCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test valid configuration
	validConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-validation
spec:
  name: test-validation
  organizationId: "%s"
  clusters:
    - metadata:
        name: %s
      provider: AWS
      region: US_EAST_1
      tier: M0
`, env.Config.OrgID, env.Config.TestClusterName)

	configFile := env.createConfigFile(t, validConfig)

	// Test infra validate
	output, err := env.runCommand(t, "infra", "validate", "-f", configFile, "--output", "json")
	require.NoError(t, err, "Validate command should succeed for valid config")

	var validateResults []map[string]interface{}
	err = json.Unmarshal([]byte(output), &validateResults)
	require.NoError(t, err, "Should parse validate JSON output (array)")
	require.GreaterOrEqual(t, len(validateResults), 1, "Should contain at least one file result")
	// Ensure first result reports validity
	if v, ok := validateResults[0]["isValid"].(bool); ok {
		assert.Equal(t, true, v, "Expected valid file result for valid config")
	}

	// Test invalid configuration
	invalidConfig := `
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-invalid
spec:
  name: test-invalid
  # Missing organizationId
  clusters:
    - metadata:
        name: invalid-cluster
      provider: INVALID_PROVIDER
      region: INVALID_REGION
      tier: INVALID_TIER
`

	invalidConfigFile := env.createConfigFile(t, invalidConfig)

	// Test infra validate with invalid config
	_, err = env.runCommand(t, "infra", "validate", "-f", invalidConfigFile, "--output", "json")
	assert.Error(t, err, "Validate command should fail for invalid config")
}

func TestApplyCommands_Plan_Integration(t *testing.T) {
	env := setupApplyCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Create a test configuration for a simple cluster
	testConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-plan
spec:
  name: test-plan
  organizationId: "%s"
  clusters:
    - metadata:
        name: %s
      provider: AWS
      region: US_EAST_1
      tier: M0
      diskSizeGb: 0.5
`, env.Config.OrgID, env.Config.TestClusterName)

	configFile := env.createConfigFile(t, testConfig)

	// Test infra plan
	output, err := env.runCommand(t, "infra", "plan", "-f", configFile,
		"--project-id", env.Config.ProjectID, "--output", "json")
	require.NoError(t, err, "Plan command should succeed")

	var planResult map[string]interface{}
	err = json.Unmarshal([]byte(output), &planResult)
	require.NoError(t, err, "Should parse plan JSON output")

	// Verify plan structure (top-level fields)
	assert.Contains(t, planResult, "operations", "Plan should contain operations")
	if operations, ok := planResult["operations"].([]interface{}); ok {
		assert.GreaterOrEqual(t, len(operations), 1, "Should have at least one operation")
	}
	assert.Contains(t, planResult, "summary", "Plan should contain summary")

	// Test plan with different output formats
	output, err = env.runCommand(t, "infra", "plan", "-f", configFile,
		"--project-id", env.Config.ProjectID, "--output", "table")
	require.NoError(t, err, "Plan command with table output should succeed")
	assert.NotEmpty(t, output, "Table output should not be empty")
}

func TestApplyCommands_Diff_Integration(t *testing.T) {
	env := setupApplyCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Create a test configuration
	testConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-diff
spec:
  name: test-diff
  organizationId: "%s"
  clusters:
    - metadata:
        name: %s
      provider: AWS
      region: US_EAST_1
      tier: M0
`, env.Config.OrgID, env.Config.TestClusterName)

	configFile := env.createConfigFile(t, testConfig)

	// Test infra diff
	output, err := env.runCommand(t, "infra", "diff", "-f", configFile,
		"--project-id", env.Config.ProjectID, "--output", "json")
	require.NoError(t, err, "Diff command should succeed")

	var diffResult map[string]interface{}
	err = json.Unmarshal([]byte(output), &diffResult)
	require.NoError(t, err, "Should parse diff JSON output")

	// Verify diff structure
	assert.Contains(t, diffResult, "diff", "Diff result should contain diff field")
	if diff, ok := diffResult["diff"].(map[string]interface{}); ok {
		assert.Contains(t, diff, "operations", "Diff should contain operations")
	}

	// Test diff with unified output
	output, err = env.runCommand(t, "infra", "diff", "-f", configFile,
		"--project-id", env.Config.ProjectID, "--output", "unified")
	require.NoError(t, err, "Diff command with unified output should succeed")
	assert.NotEmpty(t, output, "Unified output should not be empty")
}

func TestApplyCommands_Show_Integration(t *testing.T) {
	env := setupApplyCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Create a test configuration
	testConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-show
spec:
  name: test-show
  organizationId: "%s"
  clusters:
    - metadata:
        name: %s
      provider: AWS
      region: US_EAST_1
      tier: M0
  databaseUsers:
    - metadata:
        name: testuser
      authDatabase: admin
      roles:
        - role: read
          database: admin
`, env.Config.OrgID, env.Config.TestClusterName)

	configFile := env.createConfigFile(t, testConfig)

	// Test infra show (no file flag is supported by show)
	output, err := env.runCommand(t, "infra", "show",
		"--project-id", env.Config.ProjectID, "--output", "json")
	require.NoError(t, err, "Show command should succeed")

	var showResult map[string]interface{}
	err = json.Unmarshal([]byte(output), &showResult)
	require.NoError(t, err, "Should parse show JSON output")

	// Verify show structure (ProjectState fields)
	assert.Contains(t, showResult, "clusters")
	assert.Contains(t, showResult, "databaseUsers")
	assert.Contains(t, showResult, "networkAccess")

	// Test show with summary output
	output, err = env.runCommand(t, "infra", "show", "-f", configFile,
		"--project-id", env.Config.ProjectID, "--output", "summary")
	require.NoError(t, err, "Show command with summary output should succeed")
	assert.NotEmpty(t, output, "Summary output should not be empty")
}

func TestApplyCommands_DryRun_Integration(t *testing.T) {
	env := setupApplyCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Create a test configuration with network access
	testConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-dryrun
spec:
  name: test-dryrun
  organizationId: "%s"
  networkAccess:
    - ipAddress: "203.0.113.1"
      comment: "Test IP for dry run"
`, env.Config.OrgID)

	configFile := env.createConfigFile(t, testConfig)

	// Test apply dry-run with quick mode
	output, err := env.runCommand(t, "infra", "-f", configFile,
		"--project-id", env.Config.ProjectID,
		"--dry-run", "--dry-run-mode", "quick",
		"--output", "json")
	require.NoError(t, err, "Dry-run command should succeed")

	var dryRunResult map[string]interface{}
	err = json.Unmarshal([]byte(output), &dryRunResult)
	require.NoError(t, err, "Should parse dry-run JSON output")

	// Verify dry-run structure (top-level fields)
	assert.Contains(t, dryRunResult, "mode", "Dry-run should contain mode")
	assert.Equal(t, "quick", dryRunResult["mode"], "Should use quick mode")
	assert.Contains(t, dryRunResult, "simulatedResults", "Dry-run should contain simulatedResults")
	assert.Contains(t, dryRunResult, "summary", "Dry-run should contain summary")

	// Test dry-run with thorough mode
	output, err = env.runCommand(t, "infra", "-f", configFile,
		"--project-id", env.Config.ProjectID,
		"--dry-run", "--dry-run-mode", "thorough",
		"--output", "json")
	require.NoError(t, err, "Thorough dry-run should succeed")

	err = json.Unmarshal([]byte(output), &dryRunResult)
	require.NoError(t, err, "Should parse thorough dry-run JSON output")
}

func TestApplyCommands_MultipleFiles_Integration(t *testing.T) {
	env := setupApplyCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Create multiple configuration files
	projectConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-multi
spec:
  name: test-multi
  organizationId: "%s"
`, env.Config.OrgID)

	clusterConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-multi
spec:
  name: test-multi
  organizationId: "%s"
  clusters:
    - metadata:
        name: %s
      provider: AWS
      region: US_EAST_1
      tier: M0
`, env.Config.OrgID, env.Config.TestClusterName)

	projectFile := env.createConfigFile(t, projectConfig)
	clusterFile := env.createConfigFile(t, clusterConfig)

	// Test plan with multiple files
	output, err := env.runCommand(t, "infra", "plan",
		"-f", projectFile, "-f", clusterFile,
		"--project-id", env.Config.ProjectID, "--output", "json")
	require.NoError(t, err, "Plan with multiple files should succeed")

	var planResult map[string]interface{}
	err = json.Unmarshal([]byte(output), &planResult)
	require.NoError(t, err, "Should parse multi-file plan JSON output")

	// Test with glob pattern
	globPattern := filepath.Join(env.Config.TestConfigDir, "*.yaml")
	output, err = env.runCommand(t, "infra", "plan",
		"-f", globPattern,
		"--project-id", env.Config.ProjectID, "--output", "json")
	require.NoError(t, err, "Plan with glob pattern should succeed")
}

func TestApplyCommands_ErrorScenarios_Integration(t *testing.T) {
	env := setupApplyCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test with missing config file
	_, err := env.runCommand(t, "infra", "plan", "-f", "non-existent-file.yaml")
	assert.Error(t, err, "Should fail with missing config file")

	// Test with invalid YAML
	invalidYaml := `
invalid: yaml: content
  missing: quotes
  bad: [structure
`
	invalidFile := env.createConfigFile(t, invalidYaml)
	_, err = env.runCommand(t, "infra", "validate", "-f", invalidFile)
	assert.Error(t, err, "Should fail with invalid YAML")

	// Test with missing project ID
	validConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-error
spec:
  name: test-error
  organizationId: "%s"
`, env.Config.OrgID)

	configFile := env.createConfigFile(t, validConfig)
	_, err = env.runCommand(t, "infra", "plan", "-f", configFile)
	assert.Error(t, err, "Should fail without project ID")

	// Test with invalid project ID
	_, err = env.runCommand(t, "infra", "plan", "-f", configFile,
		"--project-id", "invalid-project-id")
	assert.Error(t, err, "Should fail with invalid project ID")
}

func TestApplyCommands_OutputFormats_Integration(t *testing.T) {
	env := setupApplyCommandIntegrationTest(t)
	defer env.cleanup(t)

	testConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-formats
spec:
  name: test-formats
  organizationId: "%s"
`, env.Config.OrgID)

	configFile := env.createConfigFile(t, testConfig)

	// Test different output formats
	formats := []string{"json", "table", "summary"}

	for _, format := range formats {
		t.Run("Plan_"+format, func(t *testing.T) {
			output, err := env.runCommand(t, "infra", "plan", "-f", configFile,
				"--project-id", env.Config.ProjectID, "--output", format)
			require.NoError(t, err, "Plan with %s format should succeed", format)
			assert.NotEmpty(t, output, "Output should not be empty")

			if format == "json" {
				// Verify it's valid JSON
				var result interface{}
				err = json.Unmarshal([]byte(output), &result)
				assert.NoError(t, err, "JSON output should be valid")
			}
		})
	}
}

func TestApplyCommands_ConfigValidation_Integration(t *testing.T) {
	env := setupApplyCommandIntegrationTest(t)
	defer env.cleanup(t)

	testCases := []struct {
		name        string
		config      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid minimal config",
			config: fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-valid
spec:
  name: test-valid
  organizationId: "%s"
`, env.Config.OrgID),
			expectError: false,
		},
		{
			name: "invalid API version",
			config: `
apiVersion: invalid/v1
kind: Project
metadata:
  name: test-invalid-api
spec:
  name: test-invalid-api
`,
			expectError: true,
			errorMsg:    "unsupported API version",
		},
		{
			name: "missing organization ID",
			config: `
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-missing-org
spec:
  name: test-missing-org
`,
			expectError: true,
			errorMsg:    "organizationId is required",
		},
		{
			name: "invalid cluster tier",
			config: fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-invalid-tier
spec:
  name: test-invalid-tier
  organizationId: "%s"
  clusters:
    - metadata:
        name: invalid-tier-cluster
      provider: AWS
      region: US_EAST_1
      tier: INVALID_TIER
`, env.Config.OrgID),
			expectError: true,
			errorMsg:    "invalid tier",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configFile := env.createConfigFile(t, tc.config)

			output, err := env.runCommand(t, "infra", "validate", "-f", configFile, "--output", "json")

			if tc.expectError {
				assert.Error(t, err, "Should fail for %s", tc.name)
				if tc.errorMsg != "" {
					assert.Contains(t, output, tc.errorMsg, "Error should contain expected message")
				}
			} else {
				assert.NoError(t, err, "Should succeed for %s", tc.name)
			}
		})
	}
}

// Benchmark test for apply command performance
func BenchmarkApplyCommands_Plan_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	env := setupApplyCommandIntegrationTest(&testing.T{})
	defer env.cleanup(&testing.T{})

	testConfig := fmt.Sprintf(`
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: bench-test
spec:
  name: bench-test
  organizationId: "%s"
`, env.Config.OrgID)

	configFile := env.createConfigFile(&testing.T{}, testConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.runCommand(&testing.T{}, "infra", "plan", "-f", configFile,
			"--project-id", env.Config.ProjectID, "--output", "json")
		if err != nil {
			b.Fatalf("Apply plan command failed: %v", err)
		}
	}
}
