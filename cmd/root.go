package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/cmd/atlas"
	configcmd "github.com/teabranch/matlas-cli/cmd/config"
	"github.com/teabranch/matlas-cli/cmd/database"
	"github.com/teabranch/matlas-cli/cmd/discover"
	"github.com/teabranch/matlas-cli/cmd/infra"
	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/logging"
)

var (
	verbose    bool
	quiet      bool
	configPath string
	logFormat  string

	// Keep references for global flags.
	outputFmt  string
	timeoutDur time.Duration
	apiKey     string
	publicKey  string

	// Build information
	appVersion string
	appCommit  string
	appDate    string
	appBuiltBy string

	logger           *logging.Logger
	cfg              *config.Config
	signalHandler    *cli.SignalHandler
	shellIntegration *cli.ShellIntegration
	errorFormatter   *cli.EnhancedErrorFormatter

	rootCmd = &cobra.Command{
		Use:          "matlas",
		Short:        "CLI for MongoDB Atlas and MongoDB databases",
		Long:         "matlas-cli enables unified management of MongoDB Atlas resources and standalone MongoDB databases.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 1. Initialize enhanced logging
			logConfig := &logging.Config{
				Level:         logging.LevelInfo,
				Format:        logFormat,
				Output:        os.Stderr,
				Quiet:         quiet,
				Verbose:       verbose,
				EnableAPILogs: false,
				EnableMetrics: true,
				MaskSecrets:   true,
			}

			logger = logging.New(logConfig)
			logging.SetDefault(logger)

			// 2. Initialize signal handler for graceful shutdown
			signalHandler = cli.NewSignalHandler(logger, 30) // 30 second timeout
			signalHandler.Start()

			// 3. Initialize enhanced error handling
			errorFormatter = cli.NewEnhancedErrorFormatter(verbose, logger)

			// 4. Load merged configuration
			var err error
			cfg, err = config.Load(cmd, configPath)
			if err != nil {
				return cli.WrapWithOperation(err, "load_config", configPath)
			}

			// 6. Setup shell integration (using cmd instead of rootCmd to avoid circular reference)
			shellIntegration = cli.NewShellIntegration(logger)
			shellIntegration.SetupCompletion(cmd.Root())

			logger.Debug("Root command initialization completed",
				"verbose", verbose,
				"quiet", quiet,
				"log_format", logFormat,
				"config_path", configPath)

			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// Perform any necessary cleanup
			if logger != nil {
				logger.Debug("Command execution completed", "command", cmd.Name())
			}
			return nil
		},
	}
)

// Execute runs the matlas root command.
func Execute(version, commit, date, builtBy string) {
	// Set build information
	appVersion = version
	appCommit = commit
	appDate = date
	appBuiltBy = builtBy
	// Use enhanced error handling with recovery
	err := cli.HandleWithRecovery("root_execution", func() error {
		return rootCmd.Execute()
	})

	if err != nil {
		// Use enhanced error formatting
		if errorFormatter != nil {
			fmt.Fprintln(os.Stderr, errorFormatter.FormatWithAnalysis(err))
		} else {
			// Fallback to basic error formatting
			fmt.Fprintln(os.Stderr, err)
		}

		// Log error if logger is available
		if logger != nil {
			logger.Error("Command execution failed", "error", err.Error())
		}

		os.Exit(1)
	}
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(atlas.NewAtlasCmd())
	rootCmd.AddCommand(database.NewDatabaseCmd())
	rootCmd.AddCommand(infra.NewInfraCmd())
	rootCmd.AddCommand(discover.NewDiscoverCmd())
	rootCmd.AddCommand(configcmd.NewConfigCmd())

	// Replace default help with enhanced help that supports --format markdown
	rootCmd.SetHelpCommand(newHelpCmd(rootCmd))

	// Configure error handling for all commands to prevent help text on errors
	cli.ConfigureCommandErrorHandling(rootCmd)

	// Logging and observability options
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging with detailed output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress all non-error output")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "Log output format: text, json")

	// Configuration discovery
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default $HOME/.matlas/config.yaml)")

	// Global formatting and runtime options
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", string(config.OutputTable), "Output format: table, text, json, yaml")
	rootCmd.PersistentFlags().DurationVar(&timeoutDur, "timeout", config.DefaultTimeout, "Context timeout (e.g., 30s, 1m)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "Atlas API key (discouraged on CLI; prefer env var)")
	rootCmd.PersistentFlags().StringVar(&publicKey, "pub-key", "", "Atlas public key (discouraged on CLI; prefer env var)")

	// Mark flags as mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("verbose", "quiet")

	// Add shell integration commands (will be created by shellIntegration in PersistentPreRunE)
	// The completion command and shell-integration commands will be added dynamically

	// Add version command with enhanced information
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Display build information
			fmt.Printf("matlas-cli version: %s\n", appVersion)
			fmt.Printf("Build time: %s\n", appDate)
			fmt.Printf("Git commit: %s\n", appCommit)
			fmt.Printf("Built by: %s\n", appBuiltBy)
			fmt.Println("Go version:", "go1.24.5")

			if verbose {
				fmt.Println("\nAdvanced Features:")
				fmt.Println("  ✅ Structured logging with configurable levels")
				fmt.Println("  ✅ Graceful signal handling (SIGINT/SIGTERM)")
				fmt.Println("  ✅ Enhanced error handling with context preservation")
				fmt.Println("  ✅ Shell integration (bash/zsh/fish/powershell)")
				fmt.Println("  ✅ API request/response logging with secret masking")
				fmt.Println("  ✅ Performance metrics tracking")
				fmt.Println("  ✅ Operation progress tracking")
				fmt.Println("  ✅ Resource cleanup on interruption")
			}

			return nil
		},
	}
	rootCmd.AddCommand(versionCmd)
}

// GetLogger returns the global logger instance
func GetLogger() *logging.Logger {
	return logger
}

// GetSignalHandler returns the global signal handler instance
func GetSignalHandler() *cli.SignalHandler {
	return signalHandler
}

// GetErrorFormatter returns the global error formatter instance
func GetErrorFormatter() *cli.EnhancedErrorFormatter {
	return errorFormatter
}

// GetShellIntegration returns the global shell integration instance
func GetShellIntegration() *cli.ShellIntegration {
	return shellIntegration
}
