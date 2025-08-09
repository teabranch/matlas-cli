package database

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabaseCmd(t *testing.T) {
	cmd := NewDatabaseCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "database", cmd.Use)
	assert.Equal(t, "Manage MongoDB databases and collections", cmd.Short)
	assert.Contains(t, cmd.Aliases, "db")
	assert.Contains(t, cmd.Aliases, "databases")
	assert.True(t, cmd.SilenceUsage)

	// Check that subcommands are added
	subcommands := cmd.Commands()
	assert.True(t, len(subcommands) > 0)

	// Verify specific subcommands exist
	subcommandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		subcommandNames[i] = subcmd.Use
	}

	assert.Contains(t, subcommandNames, "list")
	assert.Contains(t, subcommandNames, "create <database-name>")
	assert.Contains(t, subcommandNames, "delete <database-name>")
	assert.Contains(t, subcommandNames, "collections")
}

func TestNewListDatabasesCmd(t *testing.T) {
	cmd := newListDatabasesCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Contains(t, cmd.Aliases, "ls")
	assert.Equal(t, "List databases", cmd.Short)
	assert.Contains(t, cmd.Long, "List all databases in a MongoDB instance")
	assert.Contains(t, cmd.Example, "matlas database list")

	// Check flags
	connectionFlag := cmd.Flags().Lookup("connection-string")
	assert.NotNil(t, connectionFlag)

	clusterFlag := cmd.Flags().Lookup("cluster")
	assert.NotNil(t, clusterFlag)

	projectFlag := cmd.Flags().Lookup("project-id")
	assert.NotNil(t, projectFlag)

	tempUserFlag := cmd.Flags().Lookup("use-temp-user")
	assert.NotNil(t, tempUserFlag)

	databaseFlag := cmd.Flags().Lookup("database")
	assert.NotNil(t, databaseFlag)
}

func TestNewCreateDatabaseCmd(t *testing.T) {
	cmd := newCreateDatabaseCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "create <database-name>", cmd.Use)
	assert.Equal(t, "Create a database", cmd.Short)
	assert.Contains(t, cmd.Long, "Create a new MongoDB database")
	assert.Contains(t, cmd.Example, "matlas database create")

	// Check flags
	connectionFlag := cmd.Flags().Lookup("connection-string")
	assert.NotNil(t, connectionFlag)

	clusterFlag := cmd.Flags().Lookup("cluster")
	assert.NotNil(t, clusterFlag)

	projectFlag := cmd.Flags().Lookup("project-id")
	assert.NotNil(t, projectFlag)
}

func TestNewDeleteDatabaseCmd(t *testing.T) {
	cmd := newDeleteDatabaseCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "delete <database-name>", cmd.Use)
	assert.Contains(t, cmd.Aliases, "del")
	assert.Contains(t, cmd.Aliases, "rm")
	assert.Contains(t, cmd.Aliases, "remove")
	assert.Equal(t, "Delete a database", cmd.Short)
	assert.Contains(t, cmd.Long, "Delete a MongoDB database")
	assert.Contains(t, cmd.Example, "matlas database delete")

	// Check flags
	connectionFlag := cmd.Flags().Lookup("connection-string")
	assert.NotNil(t, connectionFlag)

	clusterFlag := cmd.Flags().Lookup("cluster")
	assert.NotNil(t, clusterFlag)

	projectFlag := cmd.Flags().Lookup("project-id")
	assert.NotNil(t, projectFlag)

	yesFlag := cmd.Flags().Lookup("yes")
	assert.NotNil(t, yesFlag)
}

// Note: Internal helper functions like validateConnectionArgs, validateDatabaseName,
// buildConnectionString are not exposed, so we focus on testing the command structure

func TestDatabaseCmd_Help(t *testing.T) {
	cmd := NewDatabaseCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Direct database operations for MongoDB databases")
	assert.Contains(t, helpOutput, "list")
	assert.Contains(t, helpOutput, "create")
	assert.Contains(t, helpOutput, "delete")
}

func TestListDatabasesCmd_Help(t *testing.T) {
	cmd := NewDatabaseCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"list", "--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "List databases")
	assert.Contains(t, helpOutput, "--connection-string")
	assert.Contains(t, helpOutput, "--cluster")
	assert.Contains(t, helpOutput, "--project-id")
}

func TestCreateDatabaseCmd_Help(t *testing.T) {
	cmd := NewDatabaseCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"create", "--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Create a new MongoDB database")
	assert.Contains(t, helpOutput, "--connection-string")
	assert.Contains(t, helpOutput, "--cluster")
}

func TestDeleteDatabaseCmd_Help(t *testing.T) {
	cmd := NewDatabaseCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"delete", "--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Delete a MongoDB database")
	assert.Contains(t, helpOutput, "--yes")
}

func TestCommandAliases(t *testing.T) {
	cmd := NewDatabaseCmd()

	// Test that aliases work
	aliases := []string{"db", "databases"}
	for _, alias := range aliases {
		assert.Contains(t, cmd.Aliases, alias)
	}

	// Test list command aliases
	listCmd := newListDatabasesCmd()
	assert.Contains(t, listCmd.Aliases, "ls")

	// Test delete command aliases
	deleteCmd := newDeleteDatabaseCmd()
	assert.Contains(t, deleteCmd.Aliases, "rm")
	assert.Contains(t, deleteCmd.Aliases, "remove")
}

func TestCommandStructure(t *testing.T) {
	// Test that commands have proper structure and don't panic
	cmd := NewDatabaseCmd()

	// Test that we can get help without errors
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	assert.NoError(t, err)

	helpOutput := output.String()
	assert.Contains(t, helpOutput, "database")
	assert.Contains(t, helpOutput, "Direct database operations for MongoDB databases")
}

func TestSubcommandExistence(t *testing.T) {
	cmd := NewDatabaseCmd()
	subcommands := cmd.Commands()

	// Verify we have the expected number of subcommands
	assert.True(t, len(subcommands) >= 4) // list, create, delete, collections

	// Test each subcommand individually
	for _, subcmd := range subcommands {
		t.Run(subcmd.Use, func(t *testing.T) {
			assert.NotEmpty(t, subcmd.Use)
			assert.NotEmpty(t, subcmd.Short)
		})
	}
}
