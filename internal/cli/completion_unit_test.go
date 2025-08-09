package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/logging"
)

func TestShellIntegration_NewShellIntegration(t *testing.T) {
	logger := logging.New(nil)

	si := NewShellIntegration(logger)

	require.NotNil(t, si)
	assert.NotNil(t, si.logger)
}

func TestShellIntegration_GenerateShellAliases(t *testing.T) {
	logger := logging.New(nil)
	si := NewShellIntegration(logger)

	tests := []struct {
		name  string
		shell string
	}{
		{
			name:  "bash aliases",
			shell: "bash",
		},
		{
			name:  "zsh aliases",
			shell: "zsh",
		},
		{
			name:  "fish aliases",
			shell: "fish",
		},
		{
			name:  "unknown shell",
			shell: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliases := si.GenerateShellAliases(tt.shell)

			// Should return some output (not empty)
			assert.NotEmpty(t, aliases)

			// For known shells, should contain shell-specific syntax
			if tt.shell == "bash" || tt.shell == "zsh" || tt.shell == "fish" {
				assert.Contains(t, aliases, "alias")
			}
		})
	}
}

func TestShellIntegration_InstallInstructions(t *testing.T) {
	logger := logging.New(nil)
	si := NewShellIntegration(logger)

	tests := []struct {
		name  string
		shell string
	}{
		{
			name:  "bash instructions",
			shell: "bash",
		},
		{
			name:  "zsh instructions",
			shell: "zsh",
		},
		{
			name:  "fish instructions",
			shell: "fish",
		},
		{
			name:  "powershell instructions",
			shell: "powershell",
		},
		{
			name:  "unknown shell",
			shell: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instructions := si.InstallInstructions(tt.shell)

			// Should return some output (not empty)
			assert.NotEmpty(t, instructions)

			// Should contain installation-related keywords (except for unknown shells)
			if tt.shell != "unknown" {
				assert.Contains(t, instructions, "install")
			}
		})
	}
}

func TestShellIntegration_SetupAdvancedCompletion(t *testing.T) {
	logger := logging.New(nil)
	si := NewShellIntegration(logger)

	rootCmd := &cobra.Command{
		Use:   "matlas",
		Short: "MongoDB Atlas CLI",
	}

	tests := []struct {
		name        string
		apiKey      string
		publicKey   string
		shouldSetup bool
	}{
		{
			name:        "with valid credentials",
			apiKey:      "test-api-key",
			publicKey:   "test-public-key",
			shouldSetup: true,
		},
		{
			name:        "empty api key",
			apiKey:      "",
			publicKey:   "test-public-key",
			shouldSetup: false,
		},
		{
			name:        "empty public key",
			apiKey:      "test-api-key",
			publicKey:   "",
			shouldSetup: false,
		},
		{
			name:        "both empty",
			apiKey:      "",
			publicKey:   "",
			shouldSetup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This method doesn't return anything, so we just test it doesn't panic
			si.SetupAdvancedCompletion(rootCmd, tt.apiKey, tt.publicKey)

			// If it doesn't panic, the test passes
			// The actual functionality is placeholder for now
		})
	}
}

func TestShellIntegration_CompletionCommand(t *testing.T) {
	logger := logging.New(nil)
	si := NewShellIntegration(logger)

	cmd := si.CompletionCommand()

	require.NotNil(t, cmd)
	assert.Equal(t, "shell-integration", cmd.Use)
	assert.Equal(t, "Manage shell integration features", cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Should have subcommands
	subcommands := cmd.Commands()
	assert.Greater(t, len(subcommands), 0)

	// Check for specific subcommands
	var aliasCmd, installCmd *cobra.Command
	for _, subCmd := range subcommands {
		switch subCmd.Use {
		case "aliases [bash|zsh|fish]":
			aliasCmd = subCmd
		case "install [bash|zsh|fish|powershell]":
			installCmd = subCmd
		}
	}

	assert.NotNil(t, aliasCmd, "aliases subcommand should exist")
	assert.NotNil(t, installCmd, "install subcommand should exist")

	if aliasCmd != nil {
		assert.Equal(t, "Generate shell aliases for common workflows", aliasCmd.Short)
		assert.NotNil(t, aliasCmd.RunE)
	}

	if installCmd != nil {
		assert.Equal(t, "Show installation instructions for shell integration", installCmd.Short)
		assert.NotNil(t, installCmd.RunE)
	}
}

func TestShellIntegration_CompletionCommand_AliasesExecution(t *testing.T) {
	logger := logging.New(nil)
	si := NewShellIntegration(logger)

	cmd := si.CompletionCommand()

	// Find aliases subcommand
	var aliasCmd *cobra.Command
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "aliases [bash|zsh|fish]" {
			aliasCmd = subCmd
			break
		}
	}

	require.NotNil(t, aliasCmd)
	require.NotNil(t, aliasCmd.RunE)

	// Test execution with bash
	err := aliasCmd.RunE(aliasCmd, []string{"bash"})
	assert.NoError(t, err)
}

func TestShellIntegration_CompletionCommand_InstallExecution(t *testing.T) {
	logger := logging.New(nil)
	si := NewShellIntegration(logger)

	cmd := si.CompletionCommand()

	// Find install subcommand
	var installCmd *cobra.Command
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "install [bash|zsh|fish|powershell]" {
			installCmd = subCmd
			break
		}
	}

	require.NotNil(t, installCmd)
	require.NotNil(t, installCmd.RunE)

	// Test execution with bash
	err := installCmd.RunE(installCmd, []string{"bash"})
	assert.NoError(t, err)
}

func TestShellIntegration_CompletionCommand_Structure(t *testing.T) {
	logger := logging.New(nil)
	si := NewShellIntegration(logger)

	cmd := si.CompletionCommand()

	// Test basic properties
	assert.Equal(t, "shell-integration", cmd.Use)
	assert.Contains(t, cmd.Short, "shell integration")
	assert.Contains(t, cmd.Long, "shell integration")

	// Test that it has the expected subcommands
	subcommands := cmd.Commands()
	assert.Equal(t, 2, len(subcommands), "Should have exactly 2 subcommands")

	uses := make([]string, len(subcommands))
	for i, subCmd := range subcommands {
		uses[i] = subCmd.Use
	}

	assert.Contains(t, uses, "aliases [bash|zsh|fish]")
	assert.Contains(t, uses, "install [bash|zsh|fish|powershell]")
}

func TestShellIntegration_GenerateShellAliases_Content(t *testing.T) {
	logger := logging.New(nil)
	si := NewShellIntegration(logger)

	bashAliases := si.GenerateShellAliases("bash")
	zshAliases := si.GenerateShellAliases("zsh")
	fishAliases := si.GenerateShellAliases("fish")

	// Test that different shells produce different output
	// (even if they're similar, they shouldn't be identical)
	assert.NotEqual(t, bashAliases, fishAliases)

	// Test that bash and zsh might be similar but contain shell-specific elements
	if bashAliases == zshAliases {
		// If they're the same, they should at least contain alias declarations
		assert.Contains(t, bashAliases, "alias")
	}

	// All shells use alias syntax in this implementation
	assert.Contains(t, fishAliases, "alias")
}

func TestShellIntegration_InstallInstructions_Content(t *testing.T) {
	logger := logging.New(nil)
	si := NewShellIntegration(logger)

	bashInstructions := si.InstallInstructions("bash")
	fishInstructions := si.InstallInstructions("fish")
	powershellInstructions := si.InstallInstructions("powershell")

	// Different shells should have different instructions
	assert.NotEqual(t, bashInstructions, fishInstructions)
	assert.NotEqual(t, bashInstructions, powershellInstructions)
	assert.NotEqual(t, fishInstructions, powershellInstructions)

	// All should contain installation guidance
	for _, instructions := range []string{bashInstructions, fishInstructions, powershellInstructions} {
		assert.Contains(t, instructions, "install")
	}
}
