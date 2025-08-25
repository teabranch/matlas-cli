//go:build integration
// +build integration

package yaml

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

// TestDatabaseRoleYAMLValidation tests DatabaseRole YAML validation without making API calls
func TestDatabaseRoleYAMLValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping DatabaseRole YAML validation test in short mode")
	}

	// Create test scenarios for DatabaseRole validation
	testCases := []struct {
		name        string
		yamlContent string
		expectValid bool
		description string
	}{
		{
			name: "ValidDatabaseRoleWithPrivileges",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: valid-database-role-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: test-role
    spec:
      roleName: testRole
      databaseName: testdb
      privileges:
        - actions: ["find", "insert", "update", "remove"]
          resource:
            database: testdb
            collection: users
        - actions: ["find"]
          resource:
            database: testdb
            collection: logs
      inheritedRoles:
        - roleName: read
          databaseName: testdb`,
			expectValid: true,
			description: "Valid DatabaseRole with privileges and inherited roles",
		},
		{
			name: "ValidDatabaseRoleOnlyInheritedRoles",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: inherited-role-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: inherited-role
    spec:
      roleName: inheritedRole
      databaseName: testdb
      privileges: []
      inheritedRoles:
        - roleName: read
          databaseName: testdb
        - roleName: readWrite
          databaseName: analytics`,
			expectValid: true,
			description: "Valid DatabaseRole with only inherited roles",
		},
		{
			name: "InvalidEmptyRoleName",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-role-name-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: invalid-role
    spec:
      roleName: ""
      databaseName: testdb
      privileges:
        - actions: ["find"]
          resource:
            database: testdb
            collection: users`,
			expectValid: false,
			description: "Invalid DatabaseRole with empty roleName",
		},
		{
			name: "InvalidEmptyDatabaseName",
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
      databaseName: ""
      privileges:
        - actions: ["find"]
          resource:
            database: testdb
            collection: users`,
			expectValid: false,
			description: "Invalid DatabaseRole with empty databaseName",
		},
		{
			name: "InvalidEmptyPrivilegeActions",
			yamlContent: `apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: invalid-actions-test
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: invalid-actions-role
    spec:
      roleName: testRole
      databaseName: testdb
      privileges:
        - actions: []
          resource:
            database: testdb
            collection: users`,
			expectValid: false,
			description: "Invalid DatabaseRole with empty privilege actions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary YAML file
			tmpDir := t.TempDir()
			yamlFile := filepath.Join(tmpDir, fmt.Sprintf("%s.yaml", tc.name))

			err := os.WriteFile(yamlFile, []byte(tc.yamlContent), 0644)
			require.NoError(t, err, "Should write test YAML file")

			// Load as ApplyDocument
			document, err := apply.LoadApplyDocument(yamlFile)
			require.NoError(t, err, "Should load YAML as ApplyDocument")

			// Find DatabaseRole resource
			var roleResource *types.DatabaseRoleManifest
			for _, resource := range document.Resources {
				if role, ok := resource.(*types.DatabaseRoleManifest); ok {
					roleResource = role
					break
				}
			}
			require.NotNil(t, roleResource, "Should find DatabaseRole resource")

			// Validate using apply validator (without making API calls)
			validator := apply.NewValidator()
			result := validator.ValidateDocument(document)

			if tc.expectValid {
				assert.Empty(t, result.Errors, "Should pass validation: %s", tc.description)

				// Additional structure validation for valid cases
				assert.NotEmpty(t, roleResource.Spec.RoleName, "Should have roleName")
				assert.NotEmpty(t, roleResource.Spec.DatabaseName, "Should have databaseName")

				// Should have either privileges or inherited roles
				hasPrivileges := len(roleResource.Spec.Privileges) > 0
				hasInheritedRoles := len(roleResource.Spec.InheritedRoles) > 0
				assert.True(t, hasPrivileges || hasInheritedRoles,
					"Should have either privileges or inherited roles")

			} else {
				assert.NotEmpty(t, result.Errors, "Should fail validation: %s", tc.description)
			}

			// Log validation results for debugging
			if len(result.Warnings) > 0 {
				t.Logf("Validation warnings for %s:", tc.name)
				for _, warning := range result.Warnings {
					t.Logf("  - %s: %s", warning.Path, warning.Message)
				}
			}
			if len(result.Errors) > 0 {
				t.Logf("Validation errors for %s:", tc.name)
				for _, err := range result.Errors {
					t.Logf("  - %s: %s", err.Path, err.Message)
				}
			}
		})
	}
}

// TestDatabaseRoleWithDatabaseUser tests complete YAML with both DatabaseRole and DatabaseUser
func TestDatabaseRoleWithDatabaseUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping DatabaseRole with DatabaseUser validation test in short mode")
	}

	timestamp := time.Now().Unix()
	yamlContent := fmt.Sprintf(`apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: role-and-user-test-%d
  labels:
    test-type: validation
    purpose: role-user-testing
resources:
  # Custom database role
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseRole
    metadata:
      name: app-role-%d
      labels:
        purpose: application
        team: backend
    spec:
      roleName: appRole%d
      databaseName: myapp
      privileges:
        # Full access to user data
        - actions: ["find", "insert", "update", "remove"]
          resource:
            database: myapp
            collection: users
        # Read-only access to logs
        - actions: ["find"]
          resource:
            database: myapp
            collection: logs
        # Database-level read access
        - actions: ["listCollections", "listIndexes"]
          resource:
            database: myapp
      inheritedRoles:
        - roleName: read
          databaseName: reference

  # Database user that uses the custom role
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: app-user-%d
      labels:
        purpose: application
    spec:
      projectName: "test-project"
      username: appUser%d
      authDatabase: admin
      password: "TestPassword123!"
      roles:
        # Use the custom role
        - roleName: appRole%d
          databaseName: myapp
        # Also give basic admin access
        - roleName: read
          databaseName: admin
      scopes:
        - name: "test-cluster"
          type: CLUSTER
`, timestamp, timestamp, timestamp, timestamp, timestamp, timestamp)

	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "role-user-test.yaml")

	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err, "Should write test YAML file")

	// Load as ApplyDocument
	document, err := apply.LoadApplyDocument(yamlFile)
	require.NoError(t, err, "Should load YAML as ApplyDocument")

	// Verify document structure
	assert.Len(t, document.Resources, 2, "Should have 2 resources (role + user)")
	assert.Equal(t, types.KindApplyDocument, document.Kind, "Should be ApplyDocument")
	assert.NotEmpty(t, document.Metadata.Name, "Should have document name")

	// Find and verify resources
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

	// Verify role resource
	require.NotNil(t, roleResource, "Should find DatabaseRole resource")
	assert.Contains(t, roleResource.Spec.RoleName, "appRole", "Role name should contain appRole")
	assert.Equal(t, "myapp", roleResource.Spec.DatabaseName, "Database name should be myapp")
	assert.Len(t, roleResource.Spec.Privileges, 3, "Should have 3 privileges")
	assert.Len(t, roleResource.Spec.InheritedRoles, 1, "Should have 1 inherited role")

	// Verify user resource
	require.NotNil(t, userResource, "Should find DatabaseUser resource")
	assert.Contains(t, userResource.Spec.Username, "appUser", "Username should contain appUser")
	assert.Len(t, userResource.Spec.Roles, 2, "User should have 2 roles")
	assert.Len(t, userResource.Spec.Scopes, 1, "User should have 1 scope")

	// Validate through apply validator (without making API calls)
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
	assert.Empty(t, result.Errors, "Combined role and user YAML should pass validation")
}

// TestDatabaseRoleExampleFiles tests the actual example files for DatabaseRole
func TestDatabaseRoleExampleFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping DatabaseRole example files test in short mode")
	}

	projectRoot := getProjectRootForRoleTest(t)
	examplesDir := filepath.Join(projectRoot, "examples")

	// Find DatabaseRole examples
	roleExamples := []string{
		"custom-roles-example.yaml",
		"custom-roles-and-users.yaml",
		"atlas-vs-database-users-roles.yaml",
	}

	for _, exampleFile := range roleExamples {
		examplePath := filepath.Join(examplesDir, exampleFile)

		// Check if file exists
		if _, err := os.Stat(examplePath); os.IsNotExist(err) {
			t.Logf("Skipping missing example file: %s", exampleFile)
			continue
		}

		t.Run(exampleFile, func(t *testing.T) {
			// Load as ApplyDocument
			document, err := apply.LoadApplyDocument(examplePath)
			require.NoError(t, err, "Should load example file: %s", examplePath)

			// Find DatabaseRole resources
			var roleCount int
			for _, resource := range document.Resources {
				if role, ok := resource.(*types.DatabaseRoleManifest); ok {
					roleCount++

					// Validate role structure
					assert.NotEmpty(t, role.Metadata.Name, "Role should have metadata name")
					assert.NotEmpty(t, role.Spec.RoleName, "Role should have roleName")
					assert.NotEmpty(t, role.Spec.DatabaseName, "Role should have databaseName")

					// Should have either privileges or inherited roles
					hasPrivileges := len(role.Spec.Privileges) > 0
					hasInheritedRoles := len(role.Spec.InheritedRoles) > 0
					assert.True(t, hasPrivileges || hasInheritedRoles,
						"Role should have either privileges or inherited roles")
				}
			}

			assert.Greater(t, roleCount, 0, "Example should contain at least one DatabaseRole")

			// Validate through apply validator
			validator := apply.NewValidator()
			result := validator.ValidateDocument(document)

			if len(result.Errors) > 0 {
				t.Errorf("Example file validation errors for %s:", exampleFile)
				for _, err := range result.Errors {
					t.Errorf("  - %s: %s", err.Path, err.Message)
				}
			}
			assert.Empty(t, result.Errors, "Example file should pass validation: %s", exampleFile)
		})
	}
}

// Helper function
func getProjectRootForRoleTest(t *testing.T) string {
	_, filename, _, ok := runtime.Caller(1)
	require.True(t, ok, "Should be able to get test file path")

	testDir := filepath.Dir(filename)
	projectRoot := filepath.Join(testDir, "..", "..", "..")
	absPath, err := filepath.Abs(projectRoot)
	require.NoError(t, err, "Should be able to get absolute path")

	return absPath
}
