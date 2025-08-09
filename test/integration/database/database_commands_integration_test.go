//go:build integration
// +build integration

package database

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

// DatabaseCommandTestConfig holds configuration for database command integration tests
type DatabaseCommandTestConfig struct {
	ConnectionString string
	TestDatabaseName string
	TestClusterName  string
	TestProjectID    string
	BinaryPath       string
	Timeout          time.Duration
}

// DatabaseCommandTestEnvironment provides shared test setup for command-level testing
type DatabaseCommandTestEnvironment struct {
	Config             DatabaseCommandTestConfig
	CreatedCollections []string // Track created collections for cleanup
}

func setupDatabaseCommandIntegrationTest(t *testing.T) *DatabaseCommandTestEnvironment {
	t.Helper()

	// Skip if short tests
	if testing.Short() {
		t.Skip("Skipping command integration test in short mode")
	}

	// Load configuration from environment
	config := DatabaseCommandTestConfig{
		ConnectionString: os.Getenv("MONGODB_CONNECTION_STRING"),
		TestDatabaseName: "matlas_cmd_test_db",
		TestClusterName:  os.Getenv("TEST_CLUSTER_NAME"),
		TestProjectID:    os.Getenv("PROJECT_ID"),
		BinaryPath:       "./matlas", // Assuming binary is built in project root
		Timeout:          60 * time.Second,
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

	if config.ConnectionString == "" && (config.TestClusterName == "" || config.TestProjectID == "") {
		t.Skip("Neither MONGODB_CONNECTION_STRING nor TEST_CLUSTER_NAME+PROJECT_ID provided, skipping database command integration tests")
	}

	return &DatabaseCommandTestEnvironment{
		Config:             config,
		CreatedCollections: make([]string, 0),
	}
}

func (env *DatabaseCommandTestEnvironment) cleanup(t *testing.T) {
	// Clean up any collections created during tests
	for _, collectionName := range env.CreatedCollections {
		env.runCommand(t, "database", "collections", "delete", collectionName,
			"--database", env.Config.TestDatabaseName,
			"--yes", "--output", "json")
		// Ignore cleanup errors
	}
}

func (env *DatabaseCommandTestEnvironment) runCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, env.Config.BinaryPath, args...)
	cmd.Dir = "../../../" // Run from project root

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

func (env *DatabaseCommandTestEnvironment) runCommandWithConnectionString(t *testing.T, args ...string) (string, error) {
	if env.Config.ConnectionString != "" {
		args = append(args, "--connection-string", env.Config.ConnectionString)
	} else {
		args = append(args, "--cluster", env.Config.TestClusterName, "--project-id", env.Config.TestProjectID)
	}
	return env.runCommand(t, args...)
}

func TestDatabaseCommands_ListDatabases_Integration(t *testing.T) {
	env := setupDatabaseCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test basic database listing
	output, err := env.runCommandWithConnectionString(t, "database", "list", "--output", "json")
	require.NoError(t, err, "Database list command should succeed")

	// Parse JSON output
	var databases []map[string]interface{}
	err = json.Unmarshal([]byte(output), &databases)
	require.NoError(t, err, "Should parse JSON output")

	// Verify structure
	assert.Greater(t, len(databases), 0, "Should have at least one database")

	for _, db := range databases {
		assert.Contains(t, db, "name", "Database should have name field")
		assert.Contains(t, db, "sizeOnDisk", "Database should have sizeOnDisk field")
		assert.Contains(t, db, "empty", "Database should have empty field")
	}

	// Test table output format
	output, err = env.runCommandWithConnectionString(t, "database", "list")
	require.NoError(t, err, "Database list command with table output should succeed")
	assert.Contains(t, output, "NAME", "Table output should contain headers")
}

func TestDatabaseCommands_ListDatabases_ErrorScenarios_Integration(t *testing.T) {
	env := setupDatabaseCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test with invalid connection string
	_, err := env.runCommand(t, "database", "list", "--connection-string", "mongodb://invalid-host:27017")
	assert.Error(t, err, "Should fail with invalid connection string")

	// Test with missing connection parameters
	_, err = env.runCommand(t, "database", "list")
	assert.Error(t, err, "Should fail without connection parameters")
}

func TestCollectionsCommands_List_Integration(t *testing.T) {
	env := setupDatabaseCommandIntegrationTest(t)
	defer env.cleanup(t)

	// First ensure we have a test database with at least one collection
	testCollectionName := fmt.Sprintf("test_collection_%d", time.Now().Unix())
	env.CreatedCollections = append(env.CreatedCollections, testCollectionName)

	// Create a test collection first
	_, err := env.runCommandWithConnectionString(t, "database", "collections", "create", testCollectionName,
		"--database", env.Config.TestDatabaseName, "--output", "json")
	require.NoError(t, err, "Should be able to create test collection")

	// Test collections listing with JSON output
	output, err := env.runCommandWithConnectionString(t, "database", "collections", "list",
		"--database", env.Config.TestDatabaseName, "--output", "json")
	require.NoError(t, err, "Collections list command should succeed")

	// Parse JSON output
	var collections []map[string]interface{}
	err = json.Unmarshal([]byte(output), &collections)
	require.NoError(t, err, "Should parse JSON output")

	// Verify our test collection is in the list
	found := false
	for _, coll := range collections {
		if name, ok := coll["name"].(string); ok && name == testCollectionName {
			found = true
			assert.Contains(t, coll, "type", "Collection should have type field")
			break
		}
	}
	assert.True(t, found, "Should find our test collection in the list")

	// Test table output format
	output, err = env.runCommandWithConnectionString(t, "database", "collections", "list",
		"--database", env.Config.TestDatabaseName)
	require.NoError(t, err, "Collections list command with table output should succeed")
	assert.Contains(t, output, "NAME", "Table output should contain headers")
	assert.Contains(t, output, testCollectionName, "Should find test collection in table output")
}

func TestCollectionsCommands_CreateAndDelete_Integration(t *testing.T) {
	env := setupDatabaseCommandIntegrationTest(t)
	defer env.cleanup(t)

	testCollectionName := fmt.Sprintf("test_create_delete_%d", time.Now().Unix())

	// Test collection creation
	output, err := env.runCommandWithConnectionString(t, "database", "collections", "create", testCollectionName,
		"--database", env.Config.TestDatabaseName, "--output", "json")
	require.NoError(t, err, "Should be able to create collection")

	// Verify creation output
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "Should parse JSON creation output")
	assert.Contains(t, result, "collection", "Creation output should contain collection info")

	// Verify collection exists by listing
	listOutput, err := env.runCommandWithConnectionString(t, "database", "collections", "list",
		"--database", env.Config.TestDatabaseName, "--output", "json")
	require.NoError(t, err, "Should be able to list collections after creation")

	var collections []map[string]interface{}
	err = json.Unmarshal([]byte(listOutput), &collections)
	require.NoError(t, err, "Should parse collections list")

	found := false
	for _, coll := range collections {
		if name, ok := coll["name"].(string); ok && name == testCollectionName {
			found = true
			break
		}
	}
	assert.True(t, found, "Created collection should appear in list")

	// Test collection deletion
	deleteOutput, err := env.runCommandWithConnectionString(t, "database", "collections", "delete", testCollectionName,
		"--database", env.Config.TestDatabaseName, "--yes", "--output", "json")
	require.NoError(t, err, "Should be able to delete collection")

	// Verify deletion output
	var deleteResult map[string]interface{}
	err = json.Unmarshal([]byte(deleteOutput), &deleteResult)
	require.NoError(t, err, "Should parse JSON deletion output")

	// Verify collection is gone by listing again
	listOutput2, err := env.runCommandWithConnectionString(t, "database", "collections", "list",
		"--database", env.Config.TestDatabaseName, "--output", "json")
	require.NoError(t, err, "Should be able to list collections after deletion")

	var collections2 []map[string]interface{}
	err = json.Unmarshal([]byte(listOutput2), &collections2)
	require.NoError(t, err, "Should parse collections list after deletion")

	found = false
	for _, coll := range collections2 {
		if name, ok := coll["name"].(string); ok && name == testCollectionName {
			found = true
			break
		}
	}
	assert.False(t, found, "Deleted collection should not appear in list")
}

func TestCollectionsCommands_CreateCapped_Integration(t *testing.T) {
	env := setupDatabaseCommandIntegrationTest(t)
	defer env.cleanup(t)

	testCollectionName := fmt.Sprintf("test_capped_%d", time.Now().Unix())
	env.CreatedCollections = append(env.CreatedCollections, testCollectionName)

	// Test capped collection creation
	output, err := env.runCommandWithConnectionString(t, "database", "collections", "create", testCollectionName,
		"--database", env.Config.TestDatabaseName,
		"--capped", "--size", "1048576", "--max-documents", "1000",
		"--output", "json")
	require.NoError(t, err, "Should be able to create capped collection")

	// Verify creation output
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "Should parse JSON creation output")

	// List collections to verify the capped collection exists
	listOutput, err := env.runCommandWithConnectionString(t, "database", "collections", "list",
		"--database", env.Config.TestDatabaseName, "--output", "json")
	require.NoError(t, err, "Should be able to list collections")

	var collections []map[string]interface{}
	err = json.Unmarshal([]byte(listOutput), &collections)
	require.NoError(t, err, "Should parse collections list")

	found := false
	for _, coll := range collections {
		if name, ok := coll["name"].(string); ok && name == testCollectionName {
			found = true
			// Note: The exact field names for capped info depend on the MongoDB driver
			// We just verify the collection exists
			break
		}
	}
	assert.True(t, found, "Capped collection should appear in list")
}

func TestCollectionsCommands_ErrorScenarios_Integration(t *testing.T) {
	env := setupDatabaseCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test creating collection without database name
	_, err := env.runCommandWithConnectionString(t, "database", "collections", "create", "test-collection")
	assert.Error(t, err, "Should fail without database name")

	// Test deleting non-existent collection
	_, err = env.runCommandWithConnectionString(t, "database", "collections", "delete", "non-existent-collection",
		"--database", env.Config.TestDatabaseName, "--yes")
	assert.Error(t, err, "Should fail when deleting non-existent collection")

	// Test listing collections from non-existent database
	_, err = env.runCommandWithConnectionString(t, "database", "collections", "list",
		"--database", "non_existent_database_xyz")
	// This might not error in MongoDB, but let's test it
	// MongoDB typically returns empty list for non-existent databases

	// Test invalid connection parameters
	_, err = env.runCommand(t, "database", "collections", "list", "--database", "test")
	assert.Error(t, err, "Should fail without connection parameters")
}

func TestCollectionsCommands_Pagination_Integration(t *testing.T) {
	env := setupDatabaseCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Create multiple test collections for pagination testing
	baseCollectionName := fmt.Sprintf("test_pagination_%d", time.Now().Unix())
	numCollections := 5

	for i := 0; i < numCollections; i++ {
		collectionName := fmt.Sprintf("%s_%d", baseCollectionName, i)
		env.CreatedCollections = append(env.CreatedCollections, collectionName)

		_, err := env.runCommandWithConnectionString(t, "database", "collections", "create", collectionName,
			"--database", env.Config.TestDatabaseName, "--output", "json")
		require.NoError(t, err, "Should be able to create collection %d", i)
	}

	// Test with page size limit
	output, err := env.runCommandWithConnectionString(t, "database", "collections", "list",
		"--database", env.Config.TestDatabaseName,
		"--page-size", "3", "--output", "json")
	require.NoError(t, err, "Should be able to list collections with pagination")

	var collections []map[string]interface{}
	err = json.Unmarshal([]byte(output), &collections)
	require.NoError(t, err, "Should parse paginated collections list")

	// The exact number might vary based on existing collections, but we should get results
	assert.Greater(t, len(collections), 0, "Should get some collections")
	t.Logf("Retrieved %d collections with page size 3", len(collections))
}

func TestDatabaseCommands_AliasesAndHelpText_Integration(t *testing.T) {
	env := setupDatabaseCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test command aliases
	testCases := []struct {
		name    string
		args    []string
		success bool
	}{
		{
			name:    "database alias 'db'",
			args:    []string{"db", "list", "--help"},
			success: true,
		},
		{
			name:    "collections alias 'cols'",
			args:    []string{"database", "cols", "list", "--help"},
			success: true,
		},
		{
			name:    "list alias 'ls'",
			args:    []string{"database", "collections", "ls", "--help"},
			success: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := env.runCommand(t, tc.args...)
			if tc.success {
				assert.NoError(t, err, "Alias command should work")
				assert.Contains(t, output, "Usage:", "Help output should contain usage")
			} else {
				assert.Error(t, err, "Invalid command should fail")
			}
		})
	}
}

func TestDatabaseCommands_OutputFormats_Integration(t *testing.T) {
	env := setupDatabaseCommandIntegrationTest(t)
	defer env.cleanup(t)

	// Test different output formats for database list
	formats := []string{"json", "table", "yaml"}

	for _, format := range formats {
		t.Run("DatabaseList_"+format, func(t *testing.T) {
			args := append([]string{"database", "list"}, "--output", format)
			output, err := env.runCommandWithConnectionString(t, args...)

			// Some formats might not be implemented, so we check for specific behavior
			if err != nil && strings.Contains(err.Error(), "unknown output format") {
				t.Skipf("Output format '%s' not implemented", format)
			}

			require.NoError(t, err, "Database list with %s format should succeed", format)
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

// Benchmark test for command performance
func BenchmarkDatabaseCommands_ListDatabases_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	env := setupDatabaseCommandIntegrationTest(&testing.T{})
	defer env.cleanup(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.runCommandWithConnectionString(&testing.T{}, "database", "list", "--output", "json")
		if err != nil {
			b.Fatalf("Database list command failed: %v", err)
		}
	}
}
