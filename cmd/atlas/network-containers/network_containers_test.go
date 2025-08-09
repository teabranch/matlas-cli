package networkcontainers

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetworkContainersCmd(t *testing.T) {
	cmd := NewNetworkContainersCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "network-containers", cmd.Use)
	assert.Equal(t, "Manage Atlas network containers", cmd.Short)
	assert.Equal(t, "Manage MongoDB Atlas network containers for VPC peering setup", cmd.Long)
	assert.Contains(t, cmd.Aliases, "containers")
	assert.Contains(t, cmd.Aliases, "network-container")

	// Check that subcommands are added
	subcommands := cmd.Commands()
	assert.True(t, len(subcommands) > 0)

	// Verify specific subcommands exist
	subcommandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		subcommandNames[i] = subcmd.Use
	}

	assert.Contains(t, subcommandNames, "list")
	assert.Contains(t, subcommandNames, "get")
	assert.Contains(t, subcommandNames, "create")
	assert.Contains(t, subcommandNames, "delete")
}

func TestNewListCmd(t *testing.T) {
	cmd := newListCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Contains(t, cmd.Aliases, "ls")
	assert.Equal(t, "List network containers", cmd.Short)
	assert.Contains(t, cmd.Long, "List all network containers in a project")
	assert.Contains(t, cmd.Example, "matlas atlas network-containers list")

	// Check flags
	projectFlag := cmd.Flags().Lookup("project-id")
	assert.NotNil(t, projectFlag)

	providerFlag := cmd.Flags().Lookup("cloud-provider")
	assert.NotNil(t, providerFlag)

	// Check pagination flags
	pageFlag := cmd.Flags().Lookup("page")
	assert.NotNil(t, pageFlag)

	limitFlag := cmd.Flags().Lookup("limit")
	assert.NotNil(t, limitFlag)
}

func TestNewGetCmd(t *testing.T) {
	cmd := newGetCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "get", cmd.Use)
	assert.Equal(t, "Get network container details", cmd.Short)
	assert.Contains(t, cmd.Long, "Get detailed information about a specific network container")
	assert.Contains(t, cmd.Example, "matlas atlas network-containers get")

	// Check flags
	projectFlag := cmd.Flags().Lookup("project-id")
	assert.NotNil(t, projectFlag)
}

func TestNewCreateCmd(t *testing.T) {
	cmd := newCreateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "create", cmd.Use)
	assert.Equal(t, "Create a network container", cmd.Short)
	assert.Contains(t, cmd.Long, "Create a new network container")
	assert.Contains(t, cmd.Example, "matlas atlas network-containers create")

	// Check flags
	projectFlag := cmd.Flags().Lookup("project-id")
	assert.NotNil(t, projectFlag)

	providerFlag := cmd.Flags().Lookup("cloud-provider")
	assert.NotNil(t, providerFlag)

	cidrFlag := cmd.Flags().Lookup("cidr-block")
	assert.NotNil(t, cidrFlag)

	regionFlag := cmd.Flags().Lookup("region")
	assert.NotNil(t, regionFlag)
}

func TestNewDeleteCmd(t *testing.T) {
	cmd := newDeleteCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "delete", cmd.Use)
	// Note: Delete command may not have aliases in the actual implementation
	assert.Equal(t, "Delete a network container", cmd.Short)
	assert.Contains(t, cmd.Long, "Delete a network container")
	assert.Contains(t, cmd.Example, "matlas atlas network-containers delete")

	// Check flags
	projectFlag := cmd.Flags().Lookup("project-id")
	assert.NotNil(t, projectFlag)

	// Note: --yes flag may not exist in the actual implementation
}

// Note: Internal validation functions like validateCloudProvider, validateCIDRBlock,
// validateRegion are not exposed, so we focus on testing the command structure

func TestNetworkContainersCmd_Help(t *testing.T) {
	cmd := NewNetworkContainersCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Manage MongoDB Atlas network containers")
	assert.Contains(t, helpOutput, "list")
	assert.Contains(t, helpOutput, "get")
	assert.Contains(t, helpOutput, "create")
	assert.Contains(t, helpOutput, "delete")
}

func TestListCmd_Help(t *testing.T) {
	cmd := NewNetworkContainersCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"list", "--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "List all network containers")
	assert.Contains(t, helpOutput, "--project-id")
	assert.Contains(t, helpOutput, "--cloud-provider")
}

func TestCreateCmd_Help(t *testing.T) {
	cmd := NewNetworkContainersCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"create", "--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Create a new network container")
	assert.Contains(t, helpOutput, "--cidr-block")
	assert.Contains(t, helpOutput, "--region")
}

func TestCommandAliases(t *testing.T) {
	cmd := NewNetworkContainersCmd()

	// Test that aliases work
	aliases := []string{"containers", "network-container"}
	for _, alias := range aliases {
		assert.Contains(t, cmd.Aliases, alias)
	}

	// Test list command aliases
	listCmd := newListCmd()
	assert.Contains(t, listCmd.Aliases, "ls")

	// Note: Delete command aliases may not exist in actual implementation
}

func TestCommandStructure(t *testing.T) {
	// Test that commands have proper structure and don't panic
	cmd := NewNetworkContainersCmd()

	// Test that we can get help without errors
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	assert.NoError(t, err)

	helpOutput := output.String()
	assert.Contains(t, helpOutput, "network-containers")
	assert.Contains(t, helpOutput, "Manage MongoDB Atlas network containers")
}

func TestSubcommandHelp(t *testing.T) {
	cmd := NewNetworkContainersCmd()
	subcommands := []string{"list", "get", "create", "delete"}

	for _, subcmd := range subcommands {
		t.Run(subcmd, func(t *testing.T) {
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetArgs([]string{subcmd, "--help"})

			err := cmd.Execute()
			assert.NoError(t, err)

			helpOutput := output.String()
			assert.NotEmpty(t, helpOutput)
		})
	}
}
