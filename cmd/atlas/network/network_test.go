package network

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetworkCmd(t *testing.T) {
	cmd := NewNetworkCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "network", cmd.Use)
	assert.Equal(t, "Manage Atlas network access", cmd.Short)

	// Check that all subcommands are added
	subcommands := cmd.Commands()
	commandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		commandNames[i] = subcmd.Use
	}

	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "get <ip-address>")
	assert.Contains(t, commandNames, "create")
	assert.Contains(t, commandNames, "delete <ip-address>")
}

func TestNewListCmd(t *testing.T) {
	cmd := newListCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List network access entries", cmd.Short)
	assert.Contains(t, cmd.Aliases, "ls")

	// Check that flags exist
	projectFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectFlag)
}

func TestNewGetCmd(t *testing.T) {
	cmd := newGetCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "get <ip-address>", cmd.Use)
	assert.Equal(t, "Get network access entry details", cmd.Short)
	// Note: Cannot directly compare cobra.PositionalArgs functions

	// Check that flags exist
	projectFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectFlag)
}

func TestNewCreateCmd(t *testing.T) {
	cmd := newCreateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "create", cmd.Use)
	assert.Equal(t, "Create network access entry", cmd.Short)

	// Check that flags exist
	projectFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectFlag)

	ipFlag := cmd.Flags().Lookup("ip-address")
	require.NotNil(t, ipFlag)

	cidrFlag := cmd.Flags().Lookup("cidr-block")
	require.NotNil(t, cidrFlag)

	awsFlag := cmd.Flags().Lookup("aws-security-group")
	require.NotNil(t, awsFlag)

	commentFlag := cmd.Flags().Lookup("comment")
	require.NotNil(t, commentFlag)
}

func TestNewDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "delete <ip-address>", cmd.Use)
	assert.Equal(t, "Delete network access entry", cmd.Short)
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
			input:    stringPtr("192.168.1.1"),
			expected: "192.168.1.1",
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
	// Test that command fails without any flags
	// Behavior depends on whether ATLAS_PROJECT_ID is set in environment
	cmd1 := newCreateCmd()
	cmd1.SetArgs([]string{})
	err := cmd1.Execute()
	assert.Error(t, err)

	// Could fail on either project-id validation or access type validation
	// depending on environment variables
	errorMsg := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(errorMsg, "projectid cannot be empty") ||
			strings.Contains(errorMsg, "exactly one of"),
		"Expected either project-id validation error or access type validation error, got: %s", err.Error())

	// Test that command fails when project-id is provided but no access type flags
	cmd2 := newCreateCmd()
	cmd2.SetArgs([]string{"--project-id", "507f1f77bcf86cd799439011"})
	err = cmd2.Execute()
	assert.Error(t, err)
	// Should fail because exactly one of the access type flags must be specified
	assert.Contains(t, strings.ToLower(err.Error()), "exactly one of")
}

func TestGetCommandValidation(t *testing.T) {
	cmd := newGetCmd()

	// Test that command requires IP address argument
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "arg")
}

func TestDeleteCommandValidation(t *testing.T) {
	cmd := newDeleteCmd()

	// Test that command requires IP address argument
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "arg")
}
