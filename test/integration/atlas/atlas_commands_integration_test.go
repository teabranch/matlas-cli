//go:build integration
// +build integration

package atlas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AtlasCommandTestConfig holds configuration for Atlas command integration tests
type AtlasCommandTestConfig struct {
	PublicKey        string
	PrivateKey       string
	OrgID            string
	ProjectID        string
	TestProjectName  string
	BinaryPath       string
	Timeout          time.Duration
	TestClusterName  string
	TestUserUsername string
}

// AtlasCommandTestEnvironment provides shared test setup for command-level testing
type AtlasCommandTestEnvironment struct {
	Config            AtlasCommandTestConfig
	CreatedClusters   []string // Track created clusters for cleanup
	CreatedUsers      []string // Track created users for cleanup
	CreatedNetworkIPs []string // Track created network access for cleanup
	CreatedProjects   []string // Track created projects for cleanup
}

func setupAtlasCommandIntegrationTest(t *testing.T) *AtlasCommandTestEnvironment {
	t.Helper()

	// Skip if short tests
	if testing.Short() {
		t.Skip("Skipping Atlas command integration test in short mode")
	}

	// Load configuration from environment
	config := AtlasCommandTestConfig{
		PublicKey:        os.Getenv("ATLAS_PUBLIC_KEY"),
		PrivateKey:       os.Getenv("ATLAS_PRIVATE_KEY"),
		OrgID:            os.Getenv("ATLAS_ORG_ID"),
		ProjectID:        os.Getenv("ATLAS_PROJECT_ID"),
		TestProjectName:  fmt.Sprintf("matlas-test-%d", time.Now().Unix()),
		BinaryPath:       "./matlas",        // Assuming binary is built in project root
		Timeout:          120 * time.Second, // Atlas operations can be slow
		TestClusterName:  fmt.Sprintf("test-cluster-%d", time.Now().Unix()),
		TestUserUsername: fmt.Sprintf("testuser%d", time.Now().Unix()),
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
		t.Skip("ATLAS_PUBLIC_KEY and ATLAS_PRIVATE_KEY not provided, skipping Atlas command integration tests")
	}

	if config.OrgID == "" {
		t.Skip("ATLAS_ORG_ID not provided, skipping Atlas command integration tests")
	}

	return &AtlasCommandTestEnvironment{
		Config:            config,
		CreatedClusters:   make([]string, 0),
		CreatedUsers:      make([]string, 0),
		CreatedNetworkIPs: make([]string, 0),
		CreatedProjects:   make([]string, 0),
	}
}

func (env *AtlasCommandTestEnvironment) cleanup(t *testing.T) {
	t.Log("Starting cleanup of test resources...")

	// Clean up clusters (if any were created)
	for _, clusterName := range env.CreatedClusters {
		if env.Config.ProjectID != "" {
			env.runCommand(t, "atlas", "clusters", "delete", clusterName,
				"--project-id", env.Config.ProjectID, "--yes", "--output", "json")
		}
	}

	// Clean up users (if any were created)
	for _, username := range env.CreatedUsers {
		if env.Config.ProjectID != "" {
			env.runCommand(t, "atlas", "users", "delete", username,
				"--project-id", env.Config.ProjectID, "--yes", "--output", "json")
		}
	}

	// Clean up network access (if any were created)
	for _, ip := range env.CreatedNetworkIPs {
		if env.Config.ProjectID != "" {
			env.runCommand(t, "atlas", "network", "delete", ip,
				"--project-id", env.Config.ProjectID, "--yes", "--output", "json")
		}
	}

	// Clean up projects (if any were created)
	for _, projectID := range env.CreatedProjects {
		env.runCommand(t, "atlas", "projects", "delete", projectID,
			"--yes", "--output", "json")
	}

	t.Log("Cleanup completed")
}

func (env *AtlasCommandTestEnvironment) runCommand(t *testing.T, args ...string) (string, error) {
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

func TestAtlasCommands_Projects_Integration(t *testing.T) {
	env := setupAtlasCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test projects list
	output, err := env.runCommand(t, "atlas", "projects", "list", "--output", "json")
	require.NoError(t, err, "Projects list command should succeed")

	var projects []map[string]interface{}
	err = json.Unmarshal([]byte(output), &projects)
	require.NoError(t, err, "Should parse JSON output")
	assert.Greater(t, len(projects), 0, "Should have at least one project")

	// Test projects get (if we have a project ID)
	if env.Config.ProjectID != "" {
		output, err = env.runCommand(t, "atlas", "projects", "get", env.Config.ProjectID, "--output", "json")
		require.NoError(t, err, "Projects get command should succeed")

		var project map[string]interface{}
		err = json.Unmarshal([]byte(output), &project)
		require.NoError(t, err, "Should parse project JSON output")
		assert.Contains(t, project, "id", "Project should have ID field")
		assert.Contains(t, project, "name", "Project should have name field")
	}

	// Test project creation (if org ID is available)
	if env.Config.OrgID != "" {
		output, err = env.runCommand(t, "atlas", "projects", "create", env.Config.TestProjectName,
			"--org-id", env.Config.OrgID, "--output", "json")

		if err == nil {
			// Parse project creation response to get project ID
			var createResponse map[string]interface{}
			err = json.Unmarshal([]byte(output), &createResponse)
			if err == nil {
				if projectData, ok := createResponse["project"].(map[string]interface{}); ok {
					if projectID, ok := projectData["id"].(string); ok {
						env.CreatedProjects = append(env.CreatedProjects, projectID)
						t.Logf("Created test project: %s", projectID)
					}
				}
			}
		} else {
			// Project creation might fail due to permissions, which is okay for testing
			t.Logf("Project creation failed (may be expected): %v", err)
		}
	}
}

func TestAtlasCommands_Clusters_Integration(t *testing.T) {
	env := setupAtlasCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Skip if no project ID
	if env.Config.ProjectID == "" {
		t.Skip("ATLAS_PROJECT_ID not provided, skipping cluster tests")
	}

	// Test clusters list
	output, err := env.runCommand(t, "atlas", "clusters", "list",
		"--project-id", env.Config.ProjectID, "--output", "json")
	require.NoError(t, err, "Clusters list command should succeed")

	var clusters []map[string]interface{}
	err = json.Unmarshal([]byte(output), &clusters)
	require.NoError(t, err, "Should parse clusters JSON output")

	// Test cluster creation (M0 free tier for testing)
	t.Log("Attempting to create test cluster...")
	output, err = env.runCommand(t, "atlas", "clusters", "create", env.Config.TestClusterName,
		"--project-id", env.Config.ProjectID,
		"--provider", "AWS",
		"--region", "US_EAST_1",
		"--tier", "M0",
		"--output", "json")

	if err == nil {
		env.CreatedClusters = append(env.CreatedClusters, env.Config.TestClusterName)
		t.Logf("Created test cluster: %s", env.Config.TestClusterName)

		// Verify cluster creation response
		var createResponse map[string]interface{}
		err = json.Unmarshal([]byte(output), &createResponse)
		assert.NoError(t, err, "Should parse cluster creation JSON")

		// Wait a bit and test cluster get
		time.Sleep(5 * time.Second)
		output, err = env.runCommand(t, "atlas", "clusters", "get", env.Config.TestClusterName,
			"--project-id", env.Config.ProjectID, "--output", "json")
		assert.NoError(t, err, "Should be able to get created cluster")
	} else {
		// Cluster creation might fail due to limits/permissions
		t.Logf("Cluster creation failed (may be expected): %v", err)
	}
}

func TestAtlasCommands_Users_Integration(t *testing.T) {
	env := setupAtlasCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Skip if no project ID
	if env.Config.ProjectID == "" {
		t.Skip("ATLAS_PROJECT_ID not provided, skipping user tests")
	}

	// Test users list
	output, err := env.runCommand(t, "atlas", "users", "list",
		"--project-id", env.Config.ProjectID, "--output", "json")
	require.NoError(t, err, "Users list command should succeed")

	var users []map[string]interface{}
	err = json.Unmarshal([]byte(output), &users)
	require.NoError(t, err, "Should parse users JSON output")

	// Test user creation with read role
	t.Log("Attempting to create test database user...")
	output, err = env.runCommand(t, "atlas", "users", "create", env.Config.TestUserUsername,
		"--project-id", env.Config.ProjectID,
		"--role", "read@admin",
		"--password", "TestPassword123!",
		"--output", "json")

	if err == nil {
		env.CreatedUsers = append(env.CreatedUsers, env.Config.TestUserUsername)
		t.Logf("Created test user: %s", env.Config.TestUserUsername)

		// Verify user creation response
		var createResponse map[string]interface{}
		err = json.Unmarshal([]byte(output), &createResponse)
		assert.NoError(t, err, "Should parse user creation JSON")

		// Test user get
		output, err = env.runCommand(t, "atlas", "users", "get", env.Config.TestUserUsername,
			"--project-id", env.Config.ProjectID, "--output", "json")
		assert.NoError(t, err, "Should be able to get created user")
	} else {
		// User creation might fail due to existing users or permissions
		t.Logf("User creation failed (may be expected): %v", err)
	}
}

func TestAtlasCommands_Network_Integration(t *testing.T) {
	env := setupAtlasCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Skip if no project ID
	if env.Config.ProjectID == "" {
		t.Skip("ATLAS_PROJECT_ID not provided, skipping network tests")
	}

	// Test network access list
	output, err := env.runCommand(t, "atlas", "network", "list",
		"--project-id", env.Config.ProjectID, "--output", "json")
	require.NoError(t, err, "Network list command should succeed")

	var networkEntries []map[string]interface{}
	err = json.Unmarshal([]byte(output), &networkEntries)
	require.NoError(t, err, "Should parse network access JSON output")

	// Test network access creation (add current IP)
	testIP := "203.0.113.1" // RFC5737 test IP
	t.Log("Attempting to create test network access entry...")
	output, err = env.runCommand(t, "atlas", "network", "create", testIP,
		"--project-id", env.Config.ProjectID,
		"--comment", "Test IP from matlas integration test",
		"--output", "json")

	if err == nil {
		env.CreatedNetworkIPs = append(env.CreatedNetworkIPs, testIP)
		t.Logf("Created test network access for IP: %s", testIP)

		// Verify network access creation response
		var createResponse map[string]interface{}
		err = json.Unmarshal([]byte(output), &createResponse)
		assert.NoError(t, err, "Should parse network access creation JSON")

		// Test network access get
		output, err = env.runCommand(t, "atlas", "network", "get", testIP,
			"--project-id", env.Config.ProjectID, "--output", "json")
		assert.NoError(t, err, "Should be able to get created network access")
	} else {
		// Network access creation might fail due to existing entries or limits
		t.Logf("Network access creation failed (may be expected): %v", err)
	}
}

func TestAtlasCommands_ErrorScenarios_Integration(t *testing.T) {
	env := setupAtlasCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test commands without required project ID
	_, err := env.runCommand(t, "atlas", "clusters", "list")
	assert.Error(t, err, "Should fail without project ID")

	_, err = env.runCommand(t, "atlas", "users", "list")
	assert.Error(t, err, "Should fail without project ID")

	_, err = env.runCommand(t, "atlas", "network", "list")
	assert.Error(t, err, "Should fail without project ID")

	// Test with invalid project ID
	_, err = env.runCommand(t, "atlas", "clusters", "list",
		"--project-id", "invalid-project-id", "--output", "json")
	assert.Error(t, err, "Should fail with invalid project ID")

	// Test invalid cluster operations
	_, err = env.runCommand(t, "atlas", "clusters", "get", "non-existent-cluster",
		"--project-id", env.Config.ProjectID, "--output", "json")
	assert.Error(t, err, "Should fail getting non-existent cluster")

	// Test invalid user operations
	_, err = env.runCommand(t, "atlas", "users", "get", "non-existent-user",
		"--project-id", env.Config.ProjectID, "--output", "json")
	assert.Error(t, err, "Should fail getting non-existent user")
}

func TestAtlasCommands_OutputFormats_Integration(t *testing.T) {
	env := setupAtlasCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Skip if no project ID
	if env.Config.ProjectID == "" {
		t.Skip("ATLAS_PROJECT_ID not provided, skipping output format tests")
	}

	// Test different output formats for projects list
	formats := []string{"json", "table"}

	for _, format := range formats {
		t.Run("ProjectsList_"+format, func(t *testing.T) {
			output, err := env.runCommand(t, "atlas", "projects", "list", "--output", format)
			require.NoError(t, err, "Projects list with %s format should succeed", format)
			assert.NotEmpty(t, output, "Output should not be empty")

			if format == "json" {
				// Verify it's valid JSON
				var result interface{}
				err = json.Unmarshal([]byte(output), &result)
				assert.NoError(t, err, "JSON output should be valid")
			}
		})
	}

	// Test different output formats for clusters list
	for _, format := range formats {
		t.Run("ClustersList_"+format, func(t *testing.T) {
			output, err := env.runCommand(t, "atlas", "clusters", "list",
				"--project-id", env.Config.ProjectID, "--output", format)
			require.NoError(t, err, "Clusters list with %s format should succeed", format)
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

func TestAtlasCommands_HelpAndAliases_Integration(t *testing.T) {
	env := setupAtlasCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test help commands
	helpTests := []struct {
		name string
		args []string
	}{
		{"atlas help", []string{"atlas", "--help"}},
		{"clusters help", []string{"atlas", "clusters", "--help"}},
		{"users help", []string{"atlas", "users", "--help"}},
		{"network help", []string{"atlas", "network", "--help"}},
		{"projects help", []string{"atlas", "projects", "--help"}},
	}

	for _, test := range helpTests {
		t.Run(test.name, func(t *testing.T) {
			output, err := env.runCommand(t, test.args...)
			// Help commands might exit with non-zero, but should produce output
			assert.NotEmpty(t, output, "Help output should not be empty")
			assert.Contains(t, output, "Usage:", "Help output should contain usage")
		})
	}
}

// Benchmark test for command performance
func BenchmarkAtlasCommands_ProjectsList_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	env := setupAtlasCommandIntegrationTest(&testing.T{})
	defer env.cleanup(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.runCommand(&testing.T{}, "atlas", "projects", "list", "--output", "json")
		if err != nil {
			b.Fatalf("Projects list command failed: %v", err)
		}
	}
}
