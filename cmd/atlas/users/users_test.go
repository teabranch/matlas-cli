package users

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

func TestNewUsersCmd(t *testing.T) {
	cmd := NewUsersCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "users", cmd.Use)
	assert.Equal(t, "Manage Atlas database users", cmd.Short)
	assert.Contains(t, cmd.Aliases, "user")

	// Check that all subcommands are added
	subcommands := cmd.Commands()
	commandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		commandNames[i] = subcmd.Use
	}

	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "get <username>")
	assert.Contains(t, commandNames, "create")
	assert.Contains(t, commandNames, "update <username>")
	assert.Contains(t, commandNames, "delete <username>")
}

func TestNewCreateCmd(t *testing.T) {
	cmd := newCreateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "create", cmd.Use)
	assert.Equal(t, "Create a database user", cmd.Short)

	// Check that flags exist
	projectFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectFlag)

	usernameFlag := cmd.Flags().Lookup("username")
	require.NotNil(t, usernameFlag)

	rolesFlag := cmd.Flags().Lookup("roles")
	require.NotNil(t, rolesFlag)

	passwordFlag := cmd.Flags().Lookup("password")
	require.NotNil(t, passwordFlag)

	databaseFlag := cmd.Flags().Lookup("database-name")
	require.NotNil(t, databaseFlag)
	assert.Equal(t, "admin", databaseFlag.DefValue)
}

func TestNewUpdateCmd(t *testing.T) {
	cmd := newUpdateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "update <username>", cmd.Use)
	assert.Equal(t, "Update a database user", cmd.Short)
	// Note: Cannot directly compare cobra.PositionalArgs functions

	// Check that flags exist
	projectFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectFlag)

	rolesFlag := cmd.Flags().Lookup("roles")
	require.NotNil(t, rolesFlag)

	passwordFlag := cmd.Flags().Lookup("password")
	require.NotNil(t, passwordFlag)
}

func TestNewDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "delete <username>", cmd.Use)
	assert.Equal(t, "Delete a database user", cmd.Short)
	// Note: Cannot directly compare cobra.PositionalArgs functions
	assert.Contains(t, cmd.Aliases, "rm")
	assert.Contains(t, cmd.Aliases, "remove")

	// Check that flags exist
	projectFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectFlag)

	yesFlag := cmd.Flags().Lookup("yes")
	require.NotNil(t, yesFlag)
	assert.Equal(t, "false", yesFlag.DefValue)
}

func TestParseRoles(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		expected    []admin.DatabaseUserRole
		expectError bool
	}{
		{
			name:  "single role",
			input: []string{"readWrite@mydb"},
			expected: []admin.DatabaseUserRole{
				{RoleName: "readWrite", DatabaseName: "mydb"},
			},
			expectError: false,
		},
		{
			name:  "multiple roles",
			input: []string{"read@db1", "readWrite@db2", "dbAdmin@admin"},
			expected: []admin.DatabaseUserRole{
				{RoleName: "read", DatabaseName: "db1"},
				{RoleName: "readWrite", DatabaseName: "db2"},
				{RoleName: "dbAdmin", DatabaseName: "admin"},
			},
			expectError: false,
		},
		{
			name:        "invalid format - no @",
			input:       []string{"readWrite"},
			expectError: true,
		},
		{
			name:        "invalid format - empty role name",
			input:       []string{"@mydb"},
			expectError: true,
		},
		{
			name:        "invalid format - empty database name",
			input:       []string{"readWrite@"},
			expectError: true,
		},
		{
			name:        "invalid format - multiple @",
			input:       []string{"read@write@mydb"},
			expectError: true,
		},
		{
			name:        "empty input",
			input:       []string{},
			expectError: true,
		},
		{
			name:  "roles with whitespace",
			input: []string{" readWrite @ mydb ", "read@admin"},
			expected: []admin.DatabaseUserRole{
				{RoleName: "readWrite", DatabaseName: "mydb"},
				{RoleName: "read", DatabaseName: "admin"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRoles(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatRoles(t *testing.T) {
	tests := []struct {
		name     string
		input    *[]admin.DatabaseUserRole
		expected string
	}{
		{
			name:     "nil roles",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty roles",
			input:    &[]admin.DatabaseUserRole{},
			expected: "",
		},
		{
			name: "single role",
			input: &[]admin.DatabaseUserRole{
				{RoleName: "readWrite", DatabaseName: "mydb"},
			},
			expected: "readWrite@mydb",
		},
		{
			name: "multiple roles",
			input: &[]admin.DatabaseUserRole{
				{RoleName: "read", DatabaseName: "db1"},
				{RoleName: "readWrite", DatabaseName: "db2"},
				{RoleName: "dbAdmin", DatabaseName: "admin"},
			},
			expected: "read@db1, readWrite@db2, dbAdmin@admin",
		},
		{
			name: "roles with empty names",
			input: &[]admin.DatabaseUserRole{
				{RoleName: "", DatabaseName: "mydb"},
				{RoleName: "read", DatabaseName: ""},
				{RoleName: "readWrite", DatabaseName: "admin"},
			},
			expected: "readWrite@admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRoles(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty string",
			input:    stringPtr(""),
			expected: "",
		},
		{
			name:     "non-empty string",
			input:    stringPtr("test-value"),
			expected: "test-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create string pointers for testing
func stringPtr(s string) *string {
	return &s
}

// Test command validation without actual execution
func TestCreateCommandValidation(t *testing.T) {
	cmd := newCreateCmd()

	// Test that command fails without required flags
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "required")
}

func TestUpdateCommandValidation(t *testing.T) {
	cmd := newUpdateCmd()

	// Test that command requires username argument
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "arg")
}

func TestDeleteCommandValidation(t *testing.T) {
	cmd := newDeleteCmd()

	// Test that command requires username argument
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "arg")
}
