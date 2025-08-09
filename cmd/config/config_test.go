package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigCmd(t *testing.T) {
	cmd := NewConfigCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "config", cmd.Use)
	assert.Equal(t, "Manage CLI configuration", cmd.Short)
	assert.Contains(t, cmd.Aliases, "cfg")
	assert.True(t, cmd.SilenceUsage)

	// Check that subcommands are added
	subcommands := cmd.Commands()
	assert.True(t, len(subcommands) > 0)

	// Verify specific subcommands exist
	subcommandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		subcommandNames[i] = subcmd.Use
	}

	assert.Contains(t, subcommandNames, "validate [config-file]")
	assert.Contains(t, subcommandNames, "template")
	assert.Contains(t, subcommandNames, "import <source-file>")
	assert.Contains(t, subcommandNames, "export")
	assert.Contains(t, subcommandNames, "migrate")
}

func TestNewValidateCmd(t *testing.T) {
	cmd := newValidateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "validate [config-file]", cmd.Use)
	assert.Equal(t, "Validate configuration files", cmd.Short)
	assert.Contains(t, cmd.Long, "YAML syntax errors")
	assert.Contains(t, cmd.Example, "matlas config validate")

	// Check flags
	configFlag := cmd.Flags().Lookup("config")
	assert.NotNil(t, configFlag)
	assert.Equal(t, "Configuration file path", configFlag.Usage)

	schemaFlag := cmd.Flags().Lookup("schema")
	assert.NotNil(t, schemaFlag)
	assert.Equal(t, "JSON schema file for validation", schemaFlag.Usage)

	verboseFlag := cmd.Flags().Lookup("verbose")
	assert.NotNil(t, verboseFlag)
	assert.Equal(t, "v", verboseFlag.Shorthand)
}

func TestNewTemplateCmd(t *testing.T) {
	cmd := newTemplateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "template", cmd.Use)
	assert.Equal(t, "Generate configuration templates", cmd.Short)
	assert.Contains(t, cmd.Long, "Generate configuration file templates for different use cases")

	// Test template has subcommands
	subcommands := cmd.Commands()
	assert.True(t, len(subcommands) >= 2) // generate and list
}

func TestNewImportCmd(t *testing.T) {
	cmd := newImportCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "import <source-file>", cmd.Use)
	assert.Equal(t, "Import configuration from external sources", cmd.Short)
	assert.Contains(t, cmd.Long, "Import configuration from external files or formats")

	// Check flags
	targetFlag := cmd.Flags().Lookup("target")
	assert.NotNil(t, targetFlag)

	formatFlag := cmd.Flags().Lookup("format")
	assert.NotNil(t, formatFlag)
}

func TestNewExportCmd(t *testing.T) {
	cmd := newExportCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "export", cmd.Use)
	assert.Equal(t, "Export configuration to external formats", cmd.Short)
	assert.Contains(t, cmd.Long, "Export matlas-cli configuration to external formats")

	// Check flags
	formatFlag := cmd.Flags().Lookup("format")
	assert.NotNil(t, formatFlag)

	outputFlag := cmd.Flags().Lookup("output")
	assert.NotNil(t, outputFlag)
}

func TestNewMigrateCmd(t *testing.T) {
	cmd := newMigrateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "migrate", cmd.Use)
	assert.Equal(t, "Migrate configuration between versions", cmd.Short)
	assert.Contains(t, cmd.Long, "Migrate configuration files")

	// Check flags
	fromFlag := cmd.Flags().Lookup("from")
	assert.NotNil(t, fromFlag)

	toFlag := cmd.Flags().Lookup("to")
	assert.NotNil(t, toFlag)
}

func TestValidateCmd_Help(t *testing.T) {
	cmd := NewConfigCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"validate", "--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Validate matlas-cli configuration files")
	assert.Contains(t, helpOutput, "--config")
	assert.Contains(t, helpOutput, "--schema")
	assert.Contains(t, helpOutput, "--verbose")
}

func TestTemplateCmd_Help(t *testing.T) {
	cmd := NewConfigCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"template", "--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Generate configuration file templates for different use cases")
}

// Note: Internal helper functions like getConfigType, validateConfigFile, etc.
// are not exposed, so we focus on testing the command structure and public API

func TestCommandStructure(t *testing.T) {
	// Test that commands have proper structure and don't panic
	cmd := NewConfigCmd()

	// Test that we can get help without errors
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	assert.NoError(t, err)

	helpOutput := output.String()
	assert.Contains(t, helpOutput, "config")
	assert.Contains(t, helpOutput, "Manage matlas-cli configuration files and settings")
}

func TestSubcommandHelp(t *testing.T) {
	cmd := NewConfigCmd()
	subcommands := []string{"validate", "template", "import", "export", "migrate"}

	for _, subcmd := range subcommands {
		t.Run(subcmd, func(t *testing.T) {
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetArgs([]string{subcmd, "--help"})

			// Reset command state
			cmd.SetArgs([]string{subcmd, "--help"})

			err := cmd.Execute()
			assert.NoError(t, err)

			helpOutput := output.String()
			assert.NotEmpty(t, helpOutput)
		})
	}
}
