package database

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/atlas-sdk/v20250312005/admin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/teabranch/matlas-cli/cmd/database/collections"
	"github.com/teabranch/matlas-cli/cmd/database/roles"
	"github.com/teabranch/matlas-cli/cmd/database/users"
	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/logging"
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
	cmd.AddCommand(roles.NewRolesCmd())
	cmd.AddCommand(users.NewUsersCmd())

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
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name for temporary user access (default: readWriteAnyDatabase access to admin)")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: readWriteAnyDatabase@admin")

	// Hidden flag for advanced users to specify custom roles for temporary users
	cmd.Flags().String("temp-user-roles", "", "Advanced: Multiple custom roles for temporary user (format: 'role1@db1,role2@db2'). Default: readWriteAnyDatabase@admin")
	if err := cmd.Flags().MarkHidden("temp-user-roles"); err != nil {
		// This should not fail as the flag was just added
		panic(fmt.Errorf("failed to mark temp-user-roles flag as hidden: %w", err))
	}

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
	var role string
	var useTempUser bool
	var collectionName string
	var dbUsername string
	var dbPassword string

	cmd := &cobra.Command{
		Use:   "create <database-name>",
		Short: "Create a database with a collection",
		Long: `Create a new MongoDB database with an initial collection.

MongoDB creates databases lazily when the first collection is created.
This command creates the specified collection to ensure the database exists and is visible in Atlas UI.

Authentication options:
  1. Use existing database user: --username and --password
  2. Create temporary user: --use-temp-user (requires Atlas API keys)
  3. Direct connection string: --connection-string with embedded credentials`,
		Args: cobra.ExactArgs(1),
		Example: `  # Create database with collection using connection string with embedded credentials
  matlas database create mydb --collection mycoll --connection-string "mongodb+srv://user:pass@cluster.mongodb.net/"

  # Create database with collection using existing database user
  matlas database create mydb --collection mycoll --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --username myuser --password mypass

  # Create database with collection using temporary user (requires Atlas API keys in .env)
  matlas database create mydb --collection mycoll --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --use-temp-user`,
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseName := args[0]
			return runCreateDatabase(cmd, connectionString, clusterName, projectID, databaseName, collectionName, useTempUser, role, dbUsername, dbPassword)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&collectionName, "collection", "", "Collection name to create in the database (required)")
	cmd.Flags().StringVar(&dbUsername, "username", "", "Database username for authentication")
	cmd.Flags().StringVar(&dbPassword, "password", "", "Database password for authentication")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access (requires Atlas API keys)")
	cmd.Flags().StringVar(&role, "role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: readWrite@<database>")

	// Collection is required
	mustMarkFlagRequired(cmd, "collection")

	// Hidden flag for advanced users to specify custom roles for temporary users
	cmd.Flags().String("temp-user-roles", "", "Advanced: Multiple custom roles for temporary user (format: 'role1@db1,role2@db2'). Default: readWrite@<database>")
	if err := cmd.Flags().MarkHidden("temp-user-roles"); err != nil {
		// This should not fail as the flag was just added
		panic(fmt.Errorf("failed to mark temp-user-roles flag as hidden: %w", err))
	}

	// At least one connection method is required
	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")

	// Authentication validation
	cmd.MarkFlagsMutuallyExclusive("use-temp-user", "username")
	cmd.MarkFlagsMutuallyExclusive("use-temp-user", "password")
	cmd.MarkFlagsRequiredTogether("username", "password")

	// Only require --cluster when --use-temp-user is specified
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

func newDeleteDatabaseCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var useTempUser bool
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
  matlas database delete mydb --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --yes

  # Delete database with temporary user (recommended for Atlas clusters)
  matlas database delete mydb --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --use-temp-user --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			databaseName := args[0]
			return runDeleteDatabase(cmd, connectionString, clusterName, projectID, databaseName, useTempUser, yes)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access (recommended for Atlas clusters)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: readWriteAnyDatabase@admin")

	// Hidden flag for advanced users to specify custom roles for temporary users
	cmd.Flags().String("temp-user-roles", "", "Advanced: Multiple custom roles for temporary user (format: 'role1@db1,role2@db2'). Default: readWriteAnyDatabase@admin")
	if err := cmd.Flags().MarkHidden("temp-user-roles"); err != nil {
		// This should not fail as the flag was just added
		panic(fmt.Errorf("failed to mark temp-user-roles flag as hidden: %w", err))
	}

	// At least one connection method is required
	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")

	// Only require --cluster when --use-temp-user is specified
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

func runCreateDatabase(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, collectionName string, useTempUser bool, role, dbUsername, dbPassword string) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}

	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}

	// Validate authentication method
	if connectionString == "" {
		// For cluster-based connections, need either temp user or username/password
		if !useTempUser && (dbUsername == "" || dbPassword == "") {
			return fmt.Errorf("when using --cluster, you must specify either --use-temp-user OR both --username and --password")
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
	connInfo, err := resolveConnectionInfoWithCmd(ctx, cmd, cfg, connectionString, clusterName, projectID, useTempUser, "", progress)
	if err != nil {
		return err
	}

	// Set up cleanup for temporary user if one was created
	if connInfo.TempUser != nil && connInfo.TempUser.CleanupFunc != nil {
		defer func() {
			progress.StartSpinner("Cleaning up temporary user...")
			// Create a fresh context for cleanup to avoid using an expired context
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()

			if cleanupErr := connInfo.TempUser.CleanupFunc(cleanupCtx); cleanupErr != nil {
				progress.StopSpinnerWithError("Failed to cleanup temporary user")
				fmt.Printf("Warning: Failed to cleanup temporary user: %v\n", cleanupErr)
			} else {
				progress.StopSpinner("Temporary user cleaned up")
			}
		}()
	}

	// If user provided credentials and we have a cluster connection, inject them
	if dbUsername != "" && dbPassword != "" && !useTempUser && connectionString == "" {
		// We need to inject user credentials into the connection string
		encodedUsername := url.QueryEscape(dbUsername)
		encodedPassword := url.QueryEscape(dbPassword)

		connInfo.ConnectionString = strings.Replace(connInfo.ConnectionString, "mongodb+srv://",
			fmt.Sprintf("mongodb+srv://%s:%s@", encodedUsername, encodedPassword), 1)

		// Ensure there's a database path before adding query parameters
		if !hasDatabasePath(connInfo.ConnectionString) {
			// Look for query parameters and add database path before them
			if idx := strings.Index(connInfo.ConnectionString, "?"); idx != -1 {
				// There are query parameters, insert the database path before them
				connInfo.ConnectionString = connInfo.ConnectionString[:idx] + "/admin" + connInfo.ConnectionString[idx:]
			} else {
				// No query parameters, just add the database path
				connInfo.ConnectionString += "/admin"
			}
		}

		// Add authentication source
		if strings.Contains(connInfo.ConnectionString, "?") {
			connInfo.ConnectionString += "&authSource=admin"
		} else {
			connInfo.ConnectionString += "?authSource=admin"
		}
	}

	progress.StartSpinner(fmt.Sprintf("Creating database '%s' with collection '%s'...", databaseName, collectionName))

	// Create database service
	logger := logging.Default()
	dbService := database.NewService(logger)
	defer func() {
		if err := dbService.Close(ctx); err != nil {
			fmt.Printf("Warning: Failed to close database service: %v\n", err)
		}
	}()

	// Create database with collection
	err = dbService.CreateDatabaseWithCollection(ctx, connInfo, databaseName, collectionName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create database")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Database '%s' with collection '%s' created successfully", databaseName, collectionName))
	return nil
}

func runDeleteDatabase(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName string, useTempUser bool, yes bool) error {
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
	connInfo, err := resolveConnectionInfoWithCmd(ctx, cmd, cfg, connectionString, clusterName, projectID, useTempUser, "", progress)
	if err != nil {
		return err
	}

	// Set up cleanup for temporary user if one was created
	if connInfo.TempUser != nil && connInfo.TempUser.CleanupFunc != nil {
		defer func() {
			progress.StartSpinner("Cleaning up temporary user...")
			// Create a fresh context for cleanup to avoid using an expired context
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()

			if cleanupErr := connInfo.TempUser.CleanupFunc(cleanupCtx); cleanupErr != nil {
				progress.StopSpinnerWithError("Failed to cleanup temporary user")
				fmt.Printf("Warning: Failed to cleanup temporary user: %v\n", cleanupErr)
			} else {
				progress.StopSpinner("Temporary user cleaned up")
			}
		}()
	}

	progress.StartSpinner(fmt.Sprintf("Deleting database '%s'...", databaseName))

	// Create database service
	logger := logging.Default()
	dbService := database.NewService(logger)
	defer func() {
		if err := dbService.Close(ctx); err != nil {
			fmt.Printf("Warning: Failed to close database service: %v\n", err)
		}
	}()

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
func resolveConnectionInfo(ctx context.Context, cfg *config.Config, connectionString, clusterName, projectID string, useTempUser bool, databaseName string, customRoles []admin.DatabaseUserRole, verbose bool, progress *ui.ProgressIndicator) (*types.ConnectionInfo, error) {
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

		// Create temporary user for discovery with custom roles if provided
		var tempUser *database.TempUserResult
		if len(customRoles) > 0 {
			tempUser, err = tempUserManager.CreateTempUserForDiscoveryWithRoles(ctx, []string{clusterName}, databaseName, customRoles)
		} else {
			tempUser, err = tempUserManager.CreateTempUserForDiscovery(ctx, []string{clusterName}, databaseName)
		}
		if err != nil {
			progress.StopSpinnerWithError("Failed to create temporary user")
			return nil, fmt.Errorf("failed to create temporary user: %w", err)
		}

		progress.StopSpinner(fmt.Sprintf("Temporary user '%s' created (expires at %s)",
			tempUser.Username, tempUser.ExpiresAt.Format("15:04:05")))

		// Give Atlas more time to propagate the user across all nodes
		// Atlas needs time for both API propagation AND MongoDB cluster synchronization
		// Based on user testing: 2+ minutes needed for reliable authentication

		// Calculate available time for propagation (leave some buffer for the actual operation)
		deadline, hasDeadline := ctx.Deadline()
		var propagationTime time.Duration

		if hasDeadline {
			remaining := time.Until(deadline)
			// Use 75% of remaining time for propagation, leaving 25% for the actual database operation
			propagationTime = time.Duration(float64(remaining) * 0.75)

			// Cap at reasonable limits
			if propagationTime > 120*time.Second {
				propagationTime = 120 * time.Second // Max 2 minutes
			} else if propagationTime < 15*time.Second {
				propagationTime = 15 * time.Second // Min 15 seconds
			}
		} else {
			// No context deadline, use a reasonable default
			propagationTime = 60 * time.Second // 1 minute default
		}

		progress.StartSpinner(fmt.Sprintf("Waiting for user propagation and MongoDB cluster sync (%ds)...", int(propagationTime.Seconds())))

		select {
		case <-ctx.Done():
			progress.StopSpinnerWithError("User propagation cancelled due to timeout")
			return nil, fmt.Errorf("operation cancelled while waiting for user propagation: %w", ctx.Err())
		case <-time.After(propagationTime):
			progress.StopSpinner(fmt.Sprintf("User propagation and MongoDB cluster sync completed (%ds)", int(propagationTime.Seconds())))
		}

		// Insert credentials into connection string with proper URL encoding
		encodedUsername := url.QueryEscape(tempUser.Username)
		encodedPassword := url.QueryEscape(tempUser.Password)

		// Debug: Show credential encoding
		if verbose {
			fmt.Printf("Debug: Temp user credentials - Username: %s, Encoded: %s\n", tempUser.Username, encodedUsername)
			fmt.Printf("Debug: Password length: %d, Encoded length: %d\n", len(tempUser.Password), len(encodedPassword))
			fmt.Printf("Debug: Original connection string: %s\n", connectionString)
		}

		connectionString = strings.Replace(connectionString, "mongodb+srv://",
			fmt.Sprintf("mongodb+srv://%s:%s@", encodedUsername, encodedPassword), 1)

		// Debug: Show connection string after credential insertion
		if verbose {
			fmt.Printf("Debug: Connection string after credentials: %s\n", maskConnectionString(connectionString))
		}

		// Add database path if not present
		// Atlas connection strings typically don't include a database path, so we need to add one
		if !hasDatabasePath(connectionString) {
			// Look for query parameters and add database path before them
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

		// Additional debugging and validation
		if verbose {
			fmt.Printf("Debug: Final connection string: %s\n", maskConnectionString(connectionString))
			fmt.Printf("Debug: Connection string contains '@': %v\n", strings.Contains(connectionString, "@"))
			fmt.Printf("Debug: Connection string contains 'authSource=admin': %v\n", strings.Contains(connectionString, "authSource=admin"))
		}

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

// resolveConnectionInfoWithCmd is a helper that handles custom roles parsing from cmd flags
func resolveConnectionInfoWithCmd(ctx context.Context, cmd *cobra.Command, cfg *config.Config, connectionString, clusterName, projectID string, useTempUser bool, databaseName string, progress *ui.ProgressIndicator) (*types.ConnectionInfo, error) {
	var customRoles []admin.DatabaseUserRole
	var err error

	// Check for the simple --role flag first (takes precedence)
	roleStr, _ := cmd.Flags().GetString("role")
	if roleStr != "" {
		customRoles, err = parseSimpleRole(roleStr)
		if err != nil {
			return nil, fmt.Errorf("invalid role format: %w", err)
		}
	} else {
		// Fall back to the advanced --temp-user-roles flag
		customRolesStr, _ := cmd.Flags().GetString("temp-user-roles")
		customRoles, err = parseCustomRoles(customRolesStr)
		if err != nil {
			return nil, fmt.Errorf("invalid temp-user-roles format: %w", err)
		}
	}

	// If simple role was provided without database, scope it to the target database
	if len(customRoles) == 1 && databaseName != "" {
		roleStr, _ := cmd.Flags().GetString("role")
		if roleStr != "" && !strings.Contains(roleStr, "@") {
			// Simple role without database specified, scope to target database
			customRoles[0].DatabaseName = databaseName
		}
	}

	verbose := cmd.Flag("verbose").Changed
	return resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, useTempUser, databaseName, customRoles, verbose, progress)
}

// parseCustomRoles parses a custom roles string into DatabaseUserRole slice
func parseCustomRoles(rolesStr string) ([]admin.DatabaseUserRole, error) {
	if rolesStr == "" {
		return nil, nil
	}

	var roles []admin.DatabaseUserRole
	rolePairs := strings.Split(rolesStr, ",")

	for _, pair := range rolePairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.Split(pair, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid role format '%s': expected 'role@database'", pair)
		}

		roleName := strings.TrimSpace(parts[0])
		dbName := strings.TrimSpace(parts[1])

		if roleName == "" || dbName == "" {
			return nil, fmt.Errorf("invalid role format '%s': role and database cannot be empty", pair)
		}

		roles = append(roles, admin.DatabaseUserRole{
			RoleName:     roleName,
			DatabaseName: dbName,
		})
	}

	return roles, nil
}

// parseSimpleRole parses a simple role string in format "role@database" or just "role" (defaults to admin)
func parseSimpleRole(roleStr string) ([]admin.DatabaseUserRole, error) {
	if roleStr == "" {
		return nil, nil
	}

	roleStr = strings.TrimSpace(roleStr)
	if roleStr == "" {
		return nil, nil
	}

	var roleName, dbName string

	// Check if role contains @ symbol for database specification
	if strings.Contains(roleStr, "@") {
		parts := strings.Split(roleStr, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid role format '%s': expected 'role@database' or just 'role'", roleStr)
		}
		roleName = strings.TrimSpace(parts[0])
		dbName = strings.TrimSpace(parts[1])
	} else {
		// No database specified, will be scoped to target database by caller
		roleName = roleStr
		dbName = "admin" // Temporary default, will be overridden if needed
	}

	if roleName == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}
	if dbName == "" {
		return nil, fmt.Errorf("database name cannot be empty")
	}

	return []admin.DatabaseUserRole{
		{
			RoleName:     roleName,
			DatabaseName: dbName,
		},
	}, nil
}

// hasDatabasePath checks if a MongoDB connection string already has a database path
func hasDatabasePath(connectionString string) bool {
	// Remove the protocol
	withoutProtocol := strings.TrimPrefix(connectionString, "mongodb+srv://")
	withoutProtocol = strings.TrimPrefix(withoutProtocol, "mongodb://")

	// Remove credentials if present
	if idx := strings.Index(withoutProtocol, "@"); idx != -1 {
		withoutProtocol = withoutProtocol[idx+1:]
	}

	// Check if there's a path after the host
	if idx := strings.Index(withoutProtocol, "/"); idx != -1 {
		path := withoutProtocol[idx+1:]
		// Remove query parameters
		if qIdx := strings.Index(path, "?"); qIdx != -1 {
			path = path[:qIdx]
		}
		return path != ""
	}

	return false
}

// maskConnectionString masks sensitive information in connection strings for logging
func maskConnectionString(connectionString string) string {
	if connectionString == "" {
		return ""
	}

	// Find credentials in the connection string
	if strings.Contains(connectionString, "@") {
		parts := strings.Split(connectionString, "@")
		if len(parts) >= 2 {
			// Replace everything before @ with masked version
			credPart := parts[0]
			if strings.Contains(credPart, "://") {
				schemeParts := strings.Split(credPart, "://")
				if len(schemeParts) == 2 {
					return schemeParts[0] + "://***:***@" + strings.Join(parts[1:], "@")
				}
			}
		}
	}

	// If no credentials detected, just show first 20 characters
	if len(connectionString) > 20 {
		return connectionString[:20] + "..."
	}
	return connectionString
}

// testConnection attempts a quick connection test to verify user authentication
func testConnection(ctx context.Context, connectionString string) bool {
	// Set a short timeout for connection testing
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Create MongoDB client options
	clientOptions := options.Client().ApplyURI(connectionString)
	
	// Set connection timeout and server selection timeout
	clientOptions.SetConnectTimeout(5 * time.Second)
	clientOptions.SetServerSelectionTimeout(5 * time.Second)
	
	// Create MongoDB client
	client, err := mongo.Connect(testCtx, clientOptions)
	if err != nil {
		logging.Debug("MongoDB connection failed during client creation: %v", err)
		return false
	}
	defer client.Disconnect(testCtx)

	// Attempt to ping the database to verify connection and authentication
	err = client.Ping(testCtx, nil)
	if err != nil {
		logging.Debug("MongoDB connection test failed during ping: %v", err)
		return false
	}

	logging.Debug("MongoDB connection test successful")
	return true
}

// mustMarkFlagRequired marks a flag as required and panics if it cannot be marked
func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Errorf("failed to mark flag %q required: %w", name, err))
	}
}
