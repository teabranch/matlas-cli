package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/internal/logging"
)

// ShellIntegration manages shell integration features
type ShellIntegration struct {
	logger *logging.Logger
}

// NewShellIntegration creates a new shell integration manager
func NewShellIntegration(logger *logging.Logger) *ShellIntegration {
	return &ShellIntegration{
		logger: logger,
	}
}

// SetupCompletion configures autocompletion for the root command
func (si *ShellIntegration) SetupCompletion(rootCmd *cobra.Command) {
	// Add completion command
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate autocompletion script for your shell",
		Long: `Generate autocompletion script for matlas CLI.

The completion script for each shell will be printed to stdout.

To load completions:

Bash:
  $ source <(matlas completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ matlas completion bash > /etc/bash_completion.d/matlas
  # macOS:
  $ matlas completion bash > /usr/local/etc/bash_completion.d/matlas  # adjust path as needed

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ matlas completion zsh > "${fpath[1]}/_matlas"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ matlas completion fish | source

  # To load completions for each session, execute once:
  $ matlas completion fish > ~/.config/fish/completions/matlas.fish

PowerShell:
  PS> matlas completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> matlas completion powershell > matlas.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return fmt.Errorf("unsupported shell type: %s", args[0])
		},
	}

	rootCmd.AddCommand(completionCmd)

	// Setup dynamic completions for common flags and commands
	si.setupDynamicCompletion(rootCmd)
}

// setupDynamicCompletion configures dynamic completion for Atlas resources
func (si *ShellIntegration) setupDynamicCompletion(rootCmd *cobra.Command) {
	// Project ID completion
	si.registerProjectIDCompletion(rootCmd)

	// Output format completion
	si.registerOutputFormatCompletion(rootCmd)

	// File path completion for config files
	si.registerConfigFileCompletion(rootCmd)

	// Atlas resource completion
	si.registerAtlasResourceCompletion(rootCmd)
}

// registerProjectIDCompletion sets up completion for project IDs
func (si *ShellIntegration) registerProjectIDCompletion(rootCmd *cobra.Command) {
	projectCompletion := func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// This would ideally fetch from Atlas API, but for now return common examples
		suggestions := []string{
			"5e2c123456789abcdef012345\tProduction Project",
			"5e2c234567890abcdef123456\tStaging Project",
			"5e2c345678901abcdef234567\tDevelopment Project",
		}

		var matches []string
		for _, suggestion := range suggestions {
			if strings.HasPrefix(suggestion, toComplete) {
				matches = append(matches, suggestion)
			}
		}

		return matches, cobra.ShellCompDirectiveDefault
	}

	// Register for any flag that contains "project"
	_ = rootCmd.RegisterFlagCompletionFunc("project-id", projectCompletion)

	// Walk through subcommands and register for project flags
	walkCommands(rootCmd, func(cmd *cobra.Command) {
		if flag := cmd.Flags().Lookup("project-id"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("project-id", projectCompletion)
		}
		if flag := cmd.Flags().Lookup("project"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("project", projectCompletion)
		}
	})
}

// registerOutputFormatCompletion sets up completion for output formats
func (si *ShellIntegration) registerOutputFormatCompletion(rootCmd *cobra.Command) {
	outputCompletion := func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		formats := []string{
			"text\tHuman-readable text output",
			"json\tJSON formatted output",
			"yaml\tYAML formatted output",
			"table\tTable formatted output",
		}

		var matches []string
		for _, format := range formats {
			if strings.HasPrefix(format, toComplete) {
				matches = append(matches, format)
			}
		}

		return matches, cobra.ShellCompDirectiveDefault
	}

	// Register for output flags
	_ = rootCmd.RegisterFlagCompletionFunc("output", outputCompletion)
	_ = rootCmd.RegisterFlagCompletionFunc("format", outputCompletion)

	walkCommands(rootCmd, func(cmd *cobra.Command) {
		if flag := cmd.Flags().Lookup("output"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("output", outputCompletion)
		}
		if flag := cmd.Flags().Lookup("format"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("format", outputCompletion)
		}
	})
}

// registerConfigFileCompletion sets up completion for config files
func (si *ShellIntegration) registerConfigFileCompletion(rootCmd *cobra.Command) {
	configCompletion := func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Look for common config file extensions
		if toComplete == "" {
			// Suggest common locations
			homeDir, _ := os.UserHomeDir()
			suggestions := []string{
				filepath.Join(homeDir, ".matlas", "config.yaml"),
				filepath.Join(homeDir, ".config", "matlas", "config.yaml"),
				"./matlas.yaml",
				"./config.yaml",
			}

			var existing []string
			for _, path := range suggestions {
				if _, err := os.Stat(path); err == nil {
					existing = append(existing, path+"\tExisting config file")
				}
			}

			if len(existing) > 0 {
				return existing, cobra.ShellCompDirectiveDefault
			}
		}

		// Default to file completion
		return nil, cobra.ShellCompDirectiveDefault
	}

	_ = rootCmd.RegisterFlagCompletionFunc("config", configCompletion)

	walkCommands(rootCmd, func(cmd *cobra.Command) {
		if flag := cmd.Flags().Lookup("config"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("config", configCompletion)
		}
		if flag := cmd.Flags().Lookup("file"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("file", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
				// For file flags, prefer YAML files
				return nil, cobra.ShellCompDirectiveFilterFileExt
			})
		}
	})
}

// registerAtlasResourceCompletion sets up completion for Atlas resources
func (si *ShellIntegration) registerAtlasResourceCompletion(rootCmd *cobra.Command) {
	// Cluster name completion
	clusterCompletion := func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// In a real implementation, this would query the Atlas API
		// For now, return common cluster naming patterns
		suggestions := []string{
			"production-cluster\tProduction MongoDB cluster",
			"staging-cluster\tStaging MongoDB cluster",
			"dev-cluster\tDevelopment MongoDB cluster",
			"analytics-cluster\tAnalytics MongoDB cluster",
		}

		var matches []string
		for _, suggestion := range suggestions {
			if strings.HasPrefix(suggestion, toComplete) {
				matches = append(matches, suggestion)
			}
		}

		return matches, cobra.ShellCompDirectiveDefault
	}

	// Database name completion
	databaseCompletion := func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		suggestions := []string{
			"users\tUser management database",
			"products\tProduct catalog database",
			"orders\tOrder management database",
			"analytics\tAnalytics database",
			"logs\tApplication logs database",
		}

		var matches []string
		for _, suggestion := range suggestions {
			if strings.HasPrefix(suggestion, toComplete) {
				matches = append(matches, suggestion)
			}
		}

		return matches, cobra.ShellCompDirectiveDefault
	}

	// Register completions for various resource flags
	walkCommands(rootCmd, func(cmd *cobra.Command) {
		if flag := cmd.Flags().Lookup("cluster"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("cluster", clusterCompletion)
		}
		if flag := cmd.Flags().Lookup("cluster-name"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("cluster-name", clusterCompletion)
		}
		if flag := cmd.Flags().Lookup("database"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("database", databaseCompletion)
		}
		if flag := cmd.Flags().Lookup("db"); flag != nil {
			_ = cmd.RegisterFlagCompletionFunc("db", databaseCompletion)
		}
	})
}

// walkCommands recursively walks through all commands and subcommands
func walkCommands(cmd *cobra.Command, fn func(*cobra.Command)) {
	fn(cmd)
	for _, subCmd := range cmd.Commands() {
		walkCommands(subCmd, fn)
	}
}

// GenerateShellAliases generates useful shell aliases for common workflows
func (si *ShellIntegration) GenerateShellAliases(shell string) string {
	aliases := map[string]map[string]string{
		"bash": {
			"matlas-apply":    "matlas apply --file",
			"matlas-plan":     "matlas apply plan --file",
			"matlas-validate": "matlas apply validate --file",
			"matlas-discover": "matlas discover --output yaml",
			"matlas-list":     "matlas atlas clusters list",
			"matlas-status":   "matlas atlas clusters describe",
		},
		"zsh": {
			"matlas-apply":    "matlas apply --file",
			"matlas-plan":     "matlas apply plan --file",
			"matlas-validate": "matlas apply validate --file",
			"matlas-discover": "matlas discover --output yaml",
			"matlas-list":     "matlas atlas clusters list",
			"matlas-status":   "matlas atlas clusters describe",
		},
		"fish": {
			"matlas-apply":    "matlas apply --file $argv",
			"matlas-plan":     "matlas apply plan --file $argv",
			"matlas-validate": "matlas apply validate --file $argv",
			"matlas-discover": "matlas discover --output yaml $argv",
			"matlas-list":     "matlas atlas clusters list $argv",
			"matlas-status":   "matlas atlas clusters describe $argv",
		},
	}

	shellAliases, exists := aliases[shell]
	if !exists {
		return fmt.Sprintf("# Aliases not available for shell: %s\n", shell)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("# Useful aliases for %s shell\n", shell))
	result.WriteString("# Add these to your shell profile (~/.bashrc, ~/.zshrc, etc.)\n\n")

	for alias, command := range shellAliases {
		switch shell {
		case "bash", "zsh":
			result.WriteString(fmt.Sprintf("alias %s='%s'\n", alias, command))
		case "fish":
			result.WriteString(fmt.Sprintf("alias %s '%s'\n", alias, command))
		}
	}

	return result.String()
}

// InstallInstructions returns installation instructions for shell integration
func (si *ShellIntegration) InstallInstructions(shell string) string {
	switch shell {
	case "bash":
		return `# Bash completion installation:

# For current session:
source <(matlas completion bash)

# For all sessions (Linux):
matlas completion bash | sudo tee /etc/bash_completion.d/matlas > /dev/null

# For all sessions (macOS):
matlas completion bash > /usr/local/etc/bash_completion.d/matlas  # adjust path as needed

# Add aliases to ~/.bashrc:
echo '` + si.GenerateShellAliases("bash") + `' >> ~/.bashrc`

	case "zsh":
		return `# Zsh completion installation:

# Enable completion (if not already enabled):
echo "autoload -U compinit; compinit" >> ~/.zshrc

# For current session:
source <(matlas completion zsh)

# For all sessions:
matlas completion zsh > "${fpath[1]}/_matlas"

# Add aliases to ~/.zshrc:
echo '` + si.GenerateShellAliases("zsh") + `' >> ~/.zshrc

# Restart your shell or run: source ~/.zshrc`

	case "fish":
		return `# Fish completion installation:

# For current session:
matlas completion fish | source

# For all sessions:
matlas completion fish > ~/.config/fish/completions/matlas.fish

# Add aliases (fish functions):
echo '` + si.GenerateShellAliases("fish") + `' >> ~/.config/fish/config.fish`

	case "powershell":
		return `# PowerShell completion installation:

# For current session:
matlas completion powershell | Out-String | Invoke-Expression

# For all sessions, add to your PowerShell profile:
matlas completion powershell >> $PROFILE

# Create useful aliases (add to $PROFILE):
Set-Alias matlas-apply "matlas apply --file"
Set-Alias matlas-plan "matlas apply plan --file"
Set-Alias matlas-validate "matlas apply validate --file"`

	default:
		return fmt.Sprintf("Installation instructions not available for shell: %s", shell)
	}
}

// SetupAdvancedCompletion configures advanced dynamic completion using Atlas API
func (si *ShellIntegration) SetupAdvancedCompletion(rootCmd *cobra.Command, atlasAPIKey, atlasPublicKey string) {
	if atlasAPIKey == "" || atlasPublicKey == "" {
		si.logger.Debug("Atlas API credentials not provided, skipping advanced completion")
		return
	}

	// This would integrate with the Atlas client to provide real-time completions
	// For now, we'll implement it as a placeholder for future enhancement

	si.logger.Debug("Advanced completion setup with Atlas API integration")

	// TODO: Implement real Atlas API integration for:
	// - Real project IDs from user's Atlas account
	// - Actual cluster names from projects
	// - Database names from clusters
	// - User names and roles
	// - Network access entries
}

// CompletionCommand creates a command for shell integration management
func (si *ShellIntegration) CompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell-integration",
		Short: "Manage shell integration features",
		Long:  "Commands to help set up and manage shell integration features like autocompletion and aliases.",
	}

	// Subcommand for generating aliases
	aliasCmd := &cobra.Command{
		Use:   "aliases [bash|zsh|fish]",
		Short: "Generate shell aliases for common workflows",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			fmt.Print(si.GenerateShellAliases(shell))
			return nil
		},
	}

	// Subcommand for installation instructions
	installCmd := &cobra.Command{
		Use:   "install [bash|zsh|fish|powershell]",
		Short: "Show installation instructions for shell integration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			fmt.Print(si.InstallInstructions(shell))
			return nil
		},
	}

	cmd.AddCommand(aliasCmd, installCmd)
	return cmd
}
