package database

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/teabranch/matlas-cli/cmd/database/collections"
	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	atlasservice "github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

// NewDatabaseCmd creates the database command with all its subcommands
func NewDatabaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Manage MongoDB databases and collections",
		Long: `Direct database operations for MongoDB databases and collections.

This command group provides operations for working directly with MongoDB databases
and collections, supporting both Atlas clusters and direct connection strings.`,
		Aliases:      []string{"db", "databases"},
		SilenceUsage: true,
	}

	cmd.AddCommand(newListDatabasesCmd())
	cmd.AddCommand(newCreateDatabaseCmd())
	cmd.AddCommand(newDeleteDatabaseCmd())
	cmd.AddCommand(collections.NewCollectionsCmd())

	return cmd
}

func newListDatabasesCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var useTempUser bool
	var databaseName string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List databases",
		Long: `List all databases in a MongoDB instance.

This command connects to a MongoDB instance and retrieves information about all
available databases, including their size, whether they're empty, and collection count.`,
		Example: `  # List databases using connection string
  matlas database list --connection-string "mongodb+srv://user:pass@cluster.mongodb.net/"

  # List databases using Atlas cluster reference
  matlas database list --cluster MyCluster --project-id 507f1f77bcf86cd799439011

  # List databases with temporary user (recommended for Atlas clusters)
  matlas database list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --use-temp-user

  # List databases with temporary user for specific database access
  matlas database list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --use-temp-user --database myapp

  # Output as JSON for automation
  matlas database list --connection-string "mongodb+srv://..." --output json

  # Using aliases
  matlas db ls --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --use-temp-user --database myapp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListDatabases(cmd, connectionString, clusterName, projectID, useTempUser, databaseName)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access (recommended for Atlas clusters)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name for temporary user access (default: read access to all databases)")

	// At least one connection method is required
	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	// Only require --cluster when --use-temp-user is specified. Using MarkFlagsRequiredTogether
	// would force users to always pass --use-temp-user when setting --cluster, which is too strict
	// (users can authenticate via other means, e.g. connection string overrides).
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// If --use-temp-user is set, ensure --cluster is also provided.
		if cmd.Flags().Changed("use-temp-user") {
			if clusterName == "" {
				return fmt.Errorf("--use-temp-user requires --cluster to be specified")
			}
		}
		return nil
	}

	return cmd
}

func newCreateDatabaseCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string

	cmd := &cobra.Command{
		Use:   "create <database-name>",
		Short: "Create a database",
		Long: `Create a new MongoDB database.

Note: MongoDB creates databases lazily when the first collection is created.
This command creates a temporary collection and then removes it to ensure the database exists.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Create database using connection string
  matlas database create mydb --connection-string "mongodb+srv://user:pass@cluster.mongodb.net/"

  # Create database using Atlas cluster reference
  matlas database create mydb --cluster MyCluster --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseName := args[0]
			return runCreateDatabase(cmd, connectionString, clusterName, projectID, databaseName)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")

	// At least one connection method is required
	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")

	return cmd
}

func newDeleteDatabaseCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <database-name>",
		Aliases: []string{"del", "rm", "remove"},
		Short:   "Delete a database",
		Long: `Delete a MongoDB database and all its collections.

⚠️  WARNING: This operation permanently deletes the database and all its data.
Use with caution in production environments.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Delete database using connection string (with confirmation)
  matlas database delete mydb --connection-string "mongodb+srv://user:pass@cluster.mongodb.net/"

  # Delete database using Atlas cluster reference (skip confirmation)
  matlas database delete mydb --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseName := args[0]
			return runDeleteDatabase(cmd, connectionString, clusterName, projectID, databaseName, yes)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	// At least one connection method is required
	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")

	return cmd
}

func runCreateDatabase(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName string) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	// Resolve connection info
	connInfo, err := resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, false, "", progress)
	if err != nil {
		return err
	}

	// Set up cleanup for temporary user if one was created
	if connInfo.TempUser != nil && connInfo.TempUser.CleanupFunc != nil {
		defer func() {
			progress.StartSpinner("Cleaning up temporary user...")
			if cleanupErr := connInfo.TempUser.CleanupFunc(ctx); cleanupErr != nil {
				progress.StopSpinnerWithError("Failed to cleanup temporary user")
				fmt.Printf("Warning: Failed to cleanup temporary user: %v\n", cleanupErr)
			} else {
				progress.StopSpinner("Temporary user cleaned up")
			}
		}()
	}

	progress.StartSpinner(fmt.Sprintf("Creating database '%s'...", databaseName))

	// Create database service
	zapLogger, _ := zap.NewDevelopment()
	dbService := database.NewService(zapLogger)
	defer dbService.Close(ctx)

	// Create database
	err = dbService.CreateDatabase(ctx, connInfo, databaseName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create database")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Database '%s' created successfully", databaseName))
	return nil
}

func runDeleteDatabase(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName string, yes bool) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}

	// Get confirmation unless --yes flag is used
	if !yes {
		confirm := ui.NewConfirmationPrompt(false, false)
		confirmed, err := confirm.ConfirmDeletion("database", databaseName)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	// Resolve connection info
	connInfo, err := resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, false, "", progress)
	if err != nil {
		return err
	}

	// Set up cleanup for temporary user if one was created
	if connInfo.TempUser != nil && connInfo.TempUser.CleanupFunc != nil {
		defer func() {
			progress.StartSpinner("Cleaning up temporary user...")
			if cleanupErr := connInfo.TempUser.CleanupFunc(ctx); cleanupErr != nil {
				progress.StopSpinnerWithError("Failed to cleanup temporary user")
				fmt.Printf("Warning: Failed to cleanup temporary user: %v\n", cleanupErr)
			} else {
				progress.StopSpinner("Temporary user cleaned up")
			}
		}()
	}

	progress.StartSpinner(fmt.Sprintf("Deleting database '%s'...", databaseName))

	// Create database service
	zapLogger, _ := zap.NewDevelopment()
	dbService := database.NewService(zapLogger)
	defer dbService.Close(ctx)

	// Delete database
	err = dbService.DropDatabase(ctx, connInfo, databaseName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to delete database")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Database '%s' deleted successfully", databaseName))
	return nil
}

// resolveConnectionInfo resolves connection information from either direct connection string or Atlas cluster
func resolveConnectionInfo(ctx context.Context, cfg *config.Config, connectionString, clusterName, projectID string, useTempUser bool, databaseName string, progress *ui.ProgressIndicator) (*types.ConnectionInfo, error) {
	if connectionString != "" {
		// Direct connection string provided
		return &types.ConnectionInfo{
			ConnectionString: connectionString,
		}, nil
	}

	// Need to resolve Atlas cluster connection string
	if clusterName == "" || projectID == "" {
		return nil, fmt.Errorf("cluster name and project ID are required when not using connection string")
	}

	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return nil, cli.FormatValidationError("project-id", projectID, err.Error())
	}

	if err := validation.ValidateClusterName(clusterName); err != nil {
		return nil, cli.FormatValidationError("cluster", clusterName, err.Error())
	}

	progress.StartSpinner(fmt.Sprintf("Resolving connection string for cluster '%s'...", clusterName))

	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return nil, cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlasservice.NewClustersService(client)

	// Get cluster details
	cluster, err := service.Get(ctx, projectID, clusterName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to get cluster details")
		errorFormatter := cli.NewErrorFormatter(true) // verbose for troubleshooting
		return nil, fmt.Errorf("%s", errorFormatter.Format(err))
	}

	// Extract connection string from cluster
	if cluster.ConnectionStrings == nil || cluster.ConnectionStrings.StandardSrv == nil {
		progress.StopSpinnerWithError("No connection string available")
		return nil, fmt.Errorf("cluster '%s' does not have a connection string available", clusterName)
	}

	connectionString = *cluster.ConnectionStrings.StandardSrv
	progress.StopSpinner("Connection string resolved")

	// Create temporary user if requested
	if useTempUser {
		progress.StartSpinner("Creating temporary database user...")

		// Create temporary user manager
		usersService := atlasservice.NewDatabaseUsersService(client)
		tempUserManager := database.NewTempUserManager(usersService, projectID)

		// Create temporary user for discovery
		tempUser, err := tempUserManager.CreateTempUserForDiscovery(ctx, []string{clusterName}, databaseName)
		if err != nil {
			progress.StopSpinnerWithError("Failed to create temporary user")
			return nil, fmt.Errorf("failed to create temporary user: %w", err)
		}

		progress.StopSpinner(fmt.Sprintf("Temporary user '%s' created (expires at %s)",
			tempUser.Username, tempUser.ExpiresAt.Format("15:04:05")))

		// Give Atlas more time to propagate the user across all nodes
		// Atlas needs additional time to sync temporary users for reliable authentication
		time.Sleep(6 * time.Second)

		// Insert credentials into connection string with proper URL encoding
		encodedUsername := url.QueryEscape(tempUser.Username)
		encodedPassword := url.QueryEscape(tempUser.Password)
		connectionString = strings.Replace(connectionString, "mongodb+srv://",
			fmt.Sprintf("mongodb+srv://%s:%s@", encodedUsername, encodedPassword), 1)

		// Ensure the connection string has a database path before query parameters
		// Atlas connection strings typically don't include a database path, so we need to add one
		if !strings.Contains(connectionString, ".net/") && !strings.Contains(connectionString, ".com/") {
			// Look for the end of the host and add a default database path
			if idx := strings.Index(connectionString, "?"); idx != -1 {
				// There are query parameters, insert the database path before them
				connectionString = connectionString[:idx] + "/admin" + connectionString[idx:]
			} else {
				// No query parameters, just add the database path
				connectionString += "/admin"
			}
		}

		// Add authentication source for Atlas users (they authenticate against admin database)
		if strings.Contains(connectionString, "?") {
			connectionString += "&authSource=admin"
		} else {
			connectionString += "?authSource=admin"
		}

		// Debug: Show the connection string being used (mask password for security)
		maskedConnectionString := strings.Replace(connectionString, encodedPassword, "***", 1)
		fmt.Printf("DEBUG: Using connection string: %s\n", maskedConnectionString)

		return &types.ConnectionInfo{
			ConnectionString: connectionString,
			TempUser: &types.TempUserInfo{
				Username:    tempUser.Username,
				ExpiresAt:   tempUser.ExpiresAt,
				CleanupFunc: tempUser.CleanupFunc,
			},
		}, nil
	}

	return &types.ConnectionInfo{
		ConnectionString: connectionString,
	}, nil
}
