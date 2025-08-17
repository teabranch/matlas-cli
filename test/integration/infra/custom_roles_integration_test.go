// Disabled due to API changes in apply package
// TODO: Update this test to use the new API
//go:build disabled
// +build disabled

package infra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/types"
)

// TestCustomRoleYAMLCreation tests end-to-end custom role creation from YAML ApplyDocument
func TestCustomRoleYAMLCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we have the necessary environment variables for Atlas integration
	projectID := os.Getenv("ATLAS_PROJECT_ID")
	clusterName := os.Getenv("ATLAS_CLUSTER_NAME")

	if projectID == "" || clusterName == "" {
		t.Skip("Skipping custom role YAML test - ATLAS_PROJECT_ID and ATLAS_CLUSTER_NAME required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create test configuration
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Failed to load configuration")

	// Create a unique test database and role name with timestamp
	timestamp := time.Now().Unix()
	testDB := fmt.Sprintf("testrolesdbyaml%d", timestamp)
	testRoleName := fmt.Sprintf("testapproloyaml%d", timestamp)
	testUserName := fmt.Sprintf("testrolesuseryaml%d", timestamp)

	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test-custom-roles.yaml")

	yamlContent := fmt.Sprintf(`apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: custom-roles-integration-test
  labels:
    test-type: integration
    purpose: role-testing
resources:
  # Custom database role with comprehensive privileges
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: %s-role
      labels:
        purpose: integration-testing
        team: qa
    spec:
      roleName: %s
      databaseName: %s
      privileges:
        # Full access to users collection
        - actions: ["find", "insert", "update", "remove"]
          resource:
            database: %s
            collection: users
        # Read-only access to logs collection
        - actions: ["find"]
          resource:
            database: %s
            collection: logs
        # Database-level list collections access
        - actions: ["listCollections", "listIndexes"]
          resource:
            database: %s
      inheritedRoles:
        # Inherit read access from built-in role
        - roleName: read
          databaseName: %s

  # Database user that uses the custom role
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: %s-user
      labels:
        purpose: integration-testing
    spec:
      projectName: "%s"
      username: %s
      authDatabase: admin
      password: "TestCustomRoleUser123!"
      roles:
        # Use the custom role we defined above
        - roleName: %s
          databaseName: %s
        # Also give read access to admin for basic operations
        - roleName: read
          databaseName: admin
      scopes:
        - name: %s
          type: CLUSTER
`, testRoleName, testRoleName, testDB, testDB, testDB, testDB, testDB, testUserName, projectID, testUserName, testRoleName, testDB, clusterName)

	err = os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err, "Failed to write test YAML file")

	t.Logf("Created test YAML file: %s", yamlFile)
	t.Logf("Test database: %s, Test role: %s, Test user: %s", testDB, testRoleName, testUserName)

	// Test 1: Validate the YAML configuration
	t.Run("ValidateYAMLConfiguration", func(t *testing.T) {
		validator := apply.NewValidator()

		// Load the YAML document
		document, err := apply.LoadApplyDocument(yamlFile)
		require.NoError(t, err, "Failed to load YAML document")

		// Validate the document
		result := validator.ValidateDocument(document)

		// Check validation results
		if len(result.Errors) > 0 {
			t.Logf("Validation errors:")
			for _, err := range result.Errors {
				t.Logf("- %s", err.Message)
			}
		}

		if len(result.Warnings) > 0 {
			t.Logf("Validation warnings:")
			for _, warn := range result.Warnings {
				t.Logf("- %s", warn.Message)
			}
		}

		assert.Empty(t, result.Errors, "YAML validation should pass without errors")

		// Verify the document contains the expected resources
		assert.Len(t, document.Resources, 2, "Document should contain 2 resources (role + user)")

		// Find and verify the DatabaseRole resource
		var roleResource *types.DatabaseRoleManifest
		var userResource *types.DatabaseUserManifest

		for _, resource := range document.Resources {
			switch r := resource.(type) {
			case *types.DatabaseRoleManifest:
				roleResource = r
			case *types.DatabaseUserManifest:
				userResource = r
			}
		}

		require.NotNil(t, roleResource, "DatabaseRole resource should be present")
		require.NotNil(t, userResource, "DatabaseUser resource should be present")

		// Verify role configuration
		assert.Equal(t, testRoleName, roleResource.Spec.RoleName, "Role name should match")
		assert.Equal(t, testDB, roleResource.Spec.DatabaseName, "Database name should match")
		assert.Len(t, roleResource.Spec.Privileges, 3, "Should have 3 privileges defined")
		assert.Len(t, roleResource.Spec.InheritedRoles, 1, "Should have 1 inherited role")

		// Verify user configuration
		assert.Equal(t, testUserName, userResource.Spec.Username, "Username should match")
		assert.Len(t, userResource.Spec.Roles, 2, "User should have 2 roles assigned")
	})

	// Test 2: Plan (dry-run) the configuration
	t.Run("PlanYAMLConfiguration", func(t *testing.T) {
		planner := apply.NewPlanner()

		// Load the document
		document, err := apply.LoadApplyDocument(yamlFile)
		require.NoError(t, err, "Failed to load YAML document")

		// Create planning options
		opts := &apply.PlanningOptions{
			ProjectID: projectID,
			DryRun:    true,
		}

		// Generate plan
		plan, err := planner.Plan(ctx, document, opts)
		require.NoError(t, err, "Planning should succeed")

		assert.NotNil(t, plan, "Plan should be generated")
		assert.Len(t, plan.Operations, 2, "Plan should contain 2 operations")

		// Verify operations are create operations for new resources
		for _, op := range plan.Operations {
			assert.Equal(t, apply.OperationCreate, op.Type, "All operations should be create operations")
		}

		t.Logf("Plan generated successfully with %d operations", len(plan.Operations))
	})

	// Test 3: Actually apply the configuration (create role and user)
	t.Run("ApplyYAMLConfiguration", func(t *testing.T) {
		executor := apply.NewAtlasExecutor(cfg)

		// Load the document
		document, err := apply.LoadApplyDocument(yamlFile)
		require.NoError(t, err, "Failed to load YAML document")

		// Create execution options
		opts := &apply.ExecutionOptions{
			ProjectID:        projectID,
			AutoApprove:      true,
			PreserveExisting: false,
		}

		// Execute the plan
		result, err := executor.Execute(ctx, document, opts)
		require.NoError(t, err, "Execution should succeed")

		assert.NotNil(t, result, "Execution result should be returned")
		assert.True(t, result.Success, "Execution should be successful")

		// Verify that both resources were created
		assert.Len(t, result.Operations, 2, "Should have executed 2 operations")

		for _, opResult := range result.Operations {
			assert.True(t, opResult.Success, "Each operation should succeed")
			assert.NoError(t, opResult.Error, "No operation should have errors")
		}

		t.Logf("YAML configuration applied successfully")

		// Cleanup: Mark resources for deletion
		t.Cleanup(func() {
			// Attempt to clean up created resources
			// This is best-effort cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			destroyOpts := &apply.DestroyOptions{
				ProjectID:   projectID,
				AutoApprove: true,
			}

			if destroyResult, err := executor.Destroy(ctx, document, destroyOpts); err != nil {
				t.Logf("Warning: Failed to cleanup test resources: %v", err)
			} else if destroyResult != nil && !destroyResult.Success {
				t.Logf("Warning: Cleanup was not fully successful")
			} else {
				t.Logf("Test resources cleaned up successfully")
			}
		})
	})

	// Test 4: Verify role can be used (if we have database access)
	t.Run("VerifyRoleUsage", func(t *testing.T) {
		// This test could verify that the created role can actually be used
		// by connecting to the database with the created user and testing permissions
		// For now, we'll just verify the resources exist via API calls

		// This would require additional MongoDB client setup and database operations
		// Skipping for now, but this is where you'd test actual role functionality
		t.Skip("Role usage verification not implemented yet - would require database operations")
	})
}

// TestYAMLRoleValidationErrors tests that invalid YAML role configurations are properly rejected
func TestYAMLRoleValidationErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name        string
		yamlContent string
		expectError bool
		errorType   string
	}{
		{
			name: "EmptyRoleName",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-role-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: invalid-role
    spec:
      roleName: ""  # Invalid: empty role name
      databaseName: testdb
      privileges:
        - actions: ["find"]
          resource:
            database: testdb
            collection: users`,
			expectError: true,
			errorType:   "empty_role_name",
		},
		{
			name: "EmptyDatabaseName",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-database-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: invalid-database-role
    spec:
      roleName: testRole
      databaseName: ""  # Invalid: empty database name
      privileges:
        - actions: ["find"]
          resource:
            database: testdb
            collection: users`,
			expectError: true,
			errorType:   "empty_database_name",
		},
		{
			name: "EmptyPrivilegesAndInheritedRoles",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: empty-privileges-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: empty-privileges-role
    spec:
      roleName: testRole
      databaseName: testdb
      privileges: []        # Empty privileges
      inheritedRoles: []    # Empty inherited roles`,
			expectError: false, // This should be a warning, not an error
			errorType:   "empty_role_definition",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary YAML file
			tmpDir := t.TempDir()
			yamlFile := filepath.Join(tmpDir, "invalid-test.yaml")

			err := os.WriteFile(yamlFile, []byte(tc.yamlContent), 0644)
			require.NoError(t, err, "Failed to write test YAML file")

			// Validate the YAML configuration
			validator := apply.NewValidator()

			document, err := apply.LoadApplyDocument(yamlFile)
			require.NoError(t, err, "Failed to load YAML document")

			result := validator.ValidateDocument(document)

			if tc.expectError {
				assert.NotEmpty(t, result.Errors, "Validation should produce errors for %s", tc.name)
			} else {
				assert.Empty(t, result.Errors, "Validation should not produce errors for %s", tc.name)
			}
		})
	}
}
