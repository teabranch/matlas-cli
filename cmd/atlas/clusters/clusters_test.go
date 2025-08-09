package clusters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClustersCmd(t *testing.T) {
	cmd := NewClustersCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "clusters", cmd.Use)
	assert.Equal(t, "Manage Atlas clusters", cmd.Short)
	assert.Contains(t, cmd.Aliases, "cluster")

	// Check that all subcommands are added
	subcommands := cmd.Commands()
	commandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		commandNames[i] = subcmd.Use
	}

	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "get <cluster-name>")
	assert.Contains(t, commandNames, "create")
	assert.Contains(t, commandNames, "update <cluster-name>")
	assert.Contains(t, commandNames, "delete <cluster-name>")
}

func TestNewCreateCmd(t *testing.T) {
	cmd := newCreateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "create", cmd.Use)
	assert.Equal(t, "Create a cluster", cmd.Short)

	// Check that flags exist
	projectFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectFlag)

	nameFlag := cmd.Flags().Lookup("name")
	require.NotNil(t, nameFlag)

	tierFlag := cmd.Flags().Lookup("tier")
	require.NotNil(t, tierFlag)
	assert.Equal(t, "M10", tierFlag.DefValue)

	providerFlag := cmd.Flags().Lookup("provider")
	require.NotNil(t, providerFlag)
	assert.Equal(t, "AWS", providerFlag.DefValue)

	regionFlag := cmd.Flags().Lookup("region")
	require.NotNil(t, regionFlag)
	assert.Equal(t, "US_EAST_1", regionFlag.DefValue)

	diskSizeFlag := cmd.Flags().Lookup("disk-size")
	require.NotNil(t, diskSizeFlag)
	assert.Equal(t, "0", diskSizeFlag.DefValue)

	backupFlag := cmd.Flags().Lookup("backup")
	require.NotNil(t, backupFlag)
	assert.Equal(t, "true", backupFlag.DefValue)
}

func TestNewUpdateCmd(t *testing.T) {
	cmd := newUpdateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "update <cluster-name>", cmd.Use)
	assert.Equal(t, "Update a cluster", cmd.Short)
	// Note: Cannot directly compare cobra.PositionalArgs functions

	// Check that flags exist
	projectFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectFlag)

	tierFlag := cmd.Flags().Lookup("tier")
	require.NotNil(t, tierFlag)

	diskSizeFlag := cmd.Flags().Lookup("disk-size")
	require.NotNil(t, diskSizeFlag)
	assert.Equal(t, "0", diskSizeFlag.DefValue)

	backupFlag := cmd.Flags().Lookup("backup")
	require.NotNil(t, backupFlag)
	assert.Equal(t, "false", backupFlag.DefValue)
}

func TestNewDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "delete <cluster-name>", cmd.Use)
	assert.Equal(t, "Delete a cluster", cmd.Short)
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
			input:    stringPtr("test-cluster"),
			expected: "test-cluster",
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
	// Should fail because project-id and name are required
	assert.Contains(t, strings.ToLower(err.Error()), "required")
}

func TestUpdateCommandValidation(t *testing.T) {
	cmd := newUpdateCmd()

	// Test that command requires cluster name argument
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "arg")
}

func TestDeleteCommandValidation(t *testing.T) {
	cmd := newDeleteCmd()

	// Test that command requires cluster name argument
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "arg")
}
