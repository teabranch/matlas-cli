package projects

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectsCmd(t *testing.T) {
	cmd := NewProjectsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "projects", cmd.Use)
	assert.Equal(t, "Manage Atlas projects", cmd.Short)
	assert.Equal(t, "List, get, and manage MongoDB Atlas projects", cmd.Long)

	// Check that all subcommands are added
	subcommands := cmd.Commands()
	commandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		commandNames[i] = subcmd.Use
	}

	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "get")
	assert.Contains(t, commandNames, "create <name>")
	assert.Contains(t, commandNames, "delete <project-id>")
}

func TestNewListCmd(t *testing.T) {
	cmd := newListCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Equal(t, "List all projects", cmd.Short)
	assert.Equal(t, "List all projects visible to the authenticated account or within a specific organization", cmd.Long)

	// Check that flags exist
	orgFlag := cmd.Flags().Lookup("org-id")
	require.NotNil(t, orgFlag)
	assert.Equal(t, "Organization ID to filter projects", orgFlag.Usage)
	assert.Equal(t, "", orgFlag.DefValue)
}

func TestNewGetCmd(t *testing.T) {
	cmd := newGetCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "get", cmd.Use)
	assert.Equal(t, "Get a specific project", cmd.Short)
	assert.Equal(t, "Get details for a specific project by ID", cmd.Long)

	// Check that flags exist
	projectFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectFlag)
	assert.Equal(t, "Project ID (can be set via ATLAS_PROJECT_ID env var)", projectFlag.Usage)
	assert.Equal(t, "", projectFlag.DefValue)
}

func TestNewCreateCmd(t *testing.T) {
	cmd := newCreateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "create <name>", cmd.Use)
	assert.Equal(t, "Create a new project", cmd.Short)
	assert.Equal(t, "Create a new MongoDB Atlas project in the specified organization", cmd.Long)
	// Note: Cannot directly compare cobra.PositionalArgs functions

	// Check that flags exist
	orgFlag := cmd.Flags().Lookup("org-id")
	require.NotNil(t, orgFlag)
	assert.Equal(t, "Organization ID where the project will be created (can be set via ATLAS_ORG_ID env var)", orgFlag.Usage)
	assert.Equal(t, "", orgFlag.DefValue)

	// Check examples are provided
	assert.NotEmpty(t, cmd.Example)
	assert.Contains(t, cmd.Example, "matlas atlas projects create")
}

func TestNewDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "delete <project-id>", cmd.Use)
	assert.Equal(t, "Delete a project", cmd.Short)
	// Note: Cannot directly compare cobra.PositionalArgs functions
	assert.Contains(t, cmd.Aliases, "rm")
	assert.Contains(t, cmd.Aliases, "remove")

	// Check that flags exist
	yesFlag := cmd.Flags().Lookup("yes")
	require.NotNil(t, yesFlag)
	assert.Equal(t, "Skip confirmation prompt", yesFlag.Usage)
	assert.Equal(t, "false", yesFlag.DefValue)

	// Check long description includes warning
	assert.Contains(t, cmd.Long, "WARNING")
	assert.Contains(t, cmd.Long, "cannot be undone")

	// Check examples are provided
	assert.NotEmpty(t, cmd.Example)
	assert.Contains(t, cmd.Example, "matlas atlas projects delete")
	assert.Contains(t, cmd.Example, "--yes")
}

func TestListCommandValidation(t *testing.T) {
	// Temporarily unset environment variables to ensure test isolation
	originalAPIKey := os.Getenv("ATLAS_API_KEY")
	originalPublicKey := os.Getenv("ATLAS_PUBLIC_KEY")
	if err := os.Unsetenv("ATLAS_API_KEY"); err != nil {
		t.Fatalf("failed to unset ATLAS_API_KEY: %v", err)
	}
	if err := os.Unsetenv("ATLAS_PUBLIC_KEY"); err != nil {
		t.Fatalf("failed to unset ATLAS_PUBLIC_KEY: %v", err)
	}
	defer func() {
		if originalAPIKey != "" {
			if err := os.Setenv("ATLAS_API_KEY", originalAPIKey); err != nil {
				t.Fatalf("failed to restore ATLAS_API_KEY: %v", err)
			}
		}
		if originalPublicKey != "" {
			if err := os.Setenv("ATLAS_PUBLIC_KEY", originalPublicKey); err != nil {
				t.Fatalf("failed to restore ATLAS_PUBLIC_KEY: %v", err)
			}
		}
	}()

	cmd := newListCmd()

	// Test command can be executed without arguments (should show help or error about missing credentials)
	err := cmd.Execute()
	require.Error(t, err) // Should fail due to missing ATLAS credentials, which is expected

	// The error should be about Atlas credentials, not command structure
	assert.Contains(t, err.Error(), "atlas api key")
}

func TestGetCommandValidation(t *testing.T) {
	// Temporarily unset environment variable to ensure test isolation
	originalProjectID := os.Getenv("ATLAS_PROJECT_ID")
	if err := os.Unsetenv("ATLAS_PROJECT_ID"); err != nil {
		t.Fatalf("failed to unset ATLAS_PROJECT_ID: %v", err)
	}
	defer func() {
		if originalProjectID != "" {
			if err := os.Setenv("ATLAS_PROJECT_ID", originalProjectID); err != nil {
				t.Fatalf("failed to restore ATLAS_PROJECT_ID: %v", err)
			}
		}
	}()

	cmd := newGetCmd()

	// Test command validation - should fail because project-id is required
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)

	// Should fail due to project-id being required during runtime validation
	assert.Contains(t, err.Error(), "project-id is required")
}

func TestCreateCommandValidation(t *testing.T) {
	cmd := newCreateCmd()

	// Test command validation - should fail because no arguments provided
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)

	// Should fail due to wrong number of arguments
	assert.Contains(t, err.Error(), "accepts 1 arg")
}

func TestDeleteCommandValidation(t *testing.T) {
	cmd := newDeleteCmd()

	// Test command validation - should fail because no arguments provided
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)

	// Should fail due to wrong number of arguments
	assert.Contains(t, err.Error(), "accepts 1 arg")
}

func TestCreateAtlasClient(t *testing.T) {
	// Test that createAtlasClient returns proper error when credentials are missing
	// This should fail because ATLAS_PUB_KEY and ATLAS_API_KEY are not set
	client, err := createAtlasClient()
	require.Error(t, err)
	require.Nil(t, client)

	// Should mention required environment variables
	assert.Contains(t, err.Error(), "ATLAS_PUB_KEY")
	assert.Contains(t, err.Error(), "ATLAS_API_KEY")
}

func TestCommandExamples(t *testing.T) {
	tests := []struct {
		name string
		cmd  func() interface{}
	}{
		{"create", func() interface{} { return newCreateCmd() }},
		{"delete", func() interface{} { return newDeleteCmd() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd()
			switch c := cmd.(type) {
			case *cobra.Command:
				// Verify examples are not empty and contain the command name
				assert.NotEmpty(t, c.Example, "Command %s should have examples", tt.name)
				assert.Contains(t, c.Example, "matlas atlas projects", "Example should show full command path")
			default:
				t.Errorf("Expected *cobra.Command, got %T", cmd)
			}
		})
	}
}

func TestCommandDescriptions(t *testing.T) {
	tests := []struct {
		name        string
		cmd         *cobra.Command
		expectShort bool
		expectLong  bool
	}{
		{"list", newListCmd(), true, true},
		{"get", newGetCmd(), true, true},
		{"create", newCreateCmd(), true, true},
		{"delete", newDeleteCmd(), true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectShort {
				assert.NotEmpty(t, tt.cmd.Short, "Command %s should have a short description", tt.name)
			}
			if tt.expectLong {
				assert.NotEmpty(t, tt.cmd.Long, "Command %s should have a long description", tt.name)
			}
		})
	}
}
