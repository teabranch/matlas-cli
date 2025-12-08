package users

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/atlas-sdk/v20250312010/admin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

// NewUsersCmd creates the database users command
func NewUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage MongoDB database users",
		Long: `Create, list, and manage MongoDB database users directly in databases.

These are database-level users created directly in MongoDB databases, distinct from
Atlas-managed database users. Database users can be assigned custom roles created
with 'matlas database roles' commands.

Use 'matlas atlas users' for Atlas-managed database users with built-in roles.`,
		Example: `  # List database users in a database
  matlas database users list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb

  # Create a database user with custom role
  matlas database users create dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --roles "customRole@mydb"

  # Create a database user with built-in roles
  matlas database users create dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --roles "readWrite@mydb,read@logs"`,
	}

	cmd.AddCommand(
		newListDatabaseUsersCmd(),
		newCreateDatabaseUserCmd(),
		newDeleteDatabaseUserCmd(),
		newGetDatabaseUserCmd(),
		newUpdateDatabaseUserCmd(),
	)

	return cmd
}

func newListDatabaseUsersCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var useTempUser bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List database users",
		Long:  "List all users defined in a MongoDB database",
		Example: `  # List database users using cluster reference
  matlas database users list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb

  # List database users using connection string
  matlas database users list --connection-string "mongodb+srv://user:pass@cluster.mongodb.net/mydb"

  # List database users with temporary user
  matlas database users list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --use-temp-user`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListDatabaseUsers(cmd, connectionString, clusterName, projectID, databaseName, useTempUser)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: dbAdminAnyDatabase@admin")

	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

func newCreateDatabaseUserCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var useTempUser bool
	var password string
	var roles []string
	var showPassword bool

	cmd := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a MongoDB database user",
		Long: `Create a MongoDB database user with specified roles.

Database users are created directly in MongoDB databases and can be assigned
both built-in roles (read, readWrite, dbAdmin) and custom roles created with
'matlas database roles create'.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Create a database user with built-in roles
  matlas database users create dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb \
    --roles "readWrite@mydb,read@logs" --password "SecurePass123!"

  # Create a database user with custom roles and show password
  matlas database users create appuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database myapp \
    --roles "customRole@myapp" --password "SecurePass123!" --show-password

  # Create user with temporary Atlas user for admin access
  matlas database users create dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb \
    --use-temp-user --roles "readWrite@mydb" --password "SecurePass123!"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			return runCreateDatabaseUser(cmd, connectionString, clusterName, projectID, databaseName, username, password, roles, useTempUser, showPassword)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access")
	cmd.Flags().StringVar(&password, "password", "", "Password for the user (required)")
	cmd.Flags().StringSliceVar(&roles, "roles", []string{}, "Roles in format 'role@database' (e.g., 'readWrite@mydb', 'customRole@myapp')")
	cmd.Flags().BoolVar(&showPassword, "show-password", false, "Print the user password after creation")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: dbAdminAnyDatabase@admin")

	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")
	mustMarkFlagRequired(cmd, "password")
	mustMarkFlagRequired(cmd, "roles")

	return cmd
}

func newDeleteDatabaseUserCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var useTempUser bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <username>",
		Short: "Delete a MongoDB database user",
		Long:  "Delete a MongoDB database user from a database",
		Args:  cobra.ExactArgs(1),
		Example: `  # Delete a database user
  matlas database users delete dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb

  # Delete a user without confirmation
  matlas database users delete dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			return runDeleteDatabaseUser(cmd, connectionString, clusterName, projectID, databaseName, username, useTempUser, yes)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: dbAdminAnyDatabase@admin")

	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

func newGetDatabaseUserCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var useTempUser bool

	cmd := &cobra.Command{
		Use:   "get <username>",
		Short: "Get details of a MongoDB database user",
		Long:  "Get detailed information about a MongoDB database user",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get user details
  matlas database users get dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb`,
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			return runGetDatabaseUser(cmd, connectionString, clusterName, projectID, databaseName, username, useTempUser)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: dbAdminAnyDatabase@admin")

	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

func newUpdateDatabaseUserCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var useTempUser bool
	var password string
	var roles []string
	var addRoles []string
	var removeRoles []string

	cmd := &cobra.Command{
		Use:   "update <username>",
		Short: "Update a MongoDB database user",
		Long: `Update a MongoDB database user's password or roles.

You can either replace all roles with --roles, or incrementally modify roles
with --add-roles and --remove-roles.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Update user password
  matlas database users update dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --password "NewPass123!"

  # Replace all roles
  matlas database users update dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --roles "read@mydb,read@logs"

  # Add roles incrementally
  matlas database users update dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --add-roles "write@logs"

  # Remove roles incrementally
  matlas database users update dbuser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --remove-roles "write@logs"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			return runUpdateDatabaseUser(cmd, connectionString, clusterName, projectID, databaseName, username, password, roles, addRoles, removeRoles, useTempUser)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access")
	cmd.Flags().StringVar(&password, "password", "", "New password for the user")
	cmd.Flags().StringSliceVar(&roles, "roles", []string{}, "Replace all roles with these (format: 'role@database')")
	cmd.Flags().StringSliceVar(&addRoles, "add-roles", []string{}, "Add these roles (format: 'role@database')")
	cmd.Flags().StringSliceVar(&removeRoles, "remove-roles", []string{}, "Remove these roles (format: 'role@database')")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: dbAdminAnyDatabase@admin")

	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	cmd.MarkFlagsMutuallyExclusive("roles", "add-roles")
	cmd.MarkFlagsMutuallyExclusive("roles", "remove-roles")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

// Implementation functions (stub implementations for now - need to implement MongoDB user management)

func runListDatabaseUsers(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName string, useTempUser bool) error {
	ctx := context.Background()
	verbose := cmd.Flag("verbose").Changed
	progress := ui.NewProgressIndicator(verbose, false)
	progress.StartSpinner("Listing database users")
	defer progress.StopSpinner("Database users listed")

	// Get connection info
	connInfo, err := resolveConnectionInfoForUsers(ctx, cmd, connectionString, clusterName, projectID, useTempUser, databaseName, progress)
	if err != nil {
		progress.StopSpinnerWithError("Failed to resolve connection")
		return fmt.Errorf("failed to resolve connection: %w", err)
	}

	// Set up cleanup for temporary user if one was created
	if connInfo.TempUser != nil && connInfo.TempUser.CleanupFunc != nil {
		defer func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			if cleanupErr := connInfo.TempUser.CleanupFunc(cleanupCtx); cleanupErr != nil {
				fmt.Printf("Warning: Failed to cleanup temporary user: %v\n", cleanupErr)
			}
		}()
	}

	// Connect to MongoDB using connection string
	clientOptions := options.Client().ApplyURI(connInfo.ConnectionString)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		progress.StopSpinnerWithError("Failed to connect to MongoDB")
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			fmt.Printf("Warning: Failed to disconnect from MongoDB: %v\n", err)
		}
	}()

	// Get the admin database for user operations
	// Note: All user management operations should be performed on admin database
	db := client.Database("admin")

	// List users using usersInfo command
	result := db.RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: 1},
	})

	var usersResponse bson.M
	if err := result.Decode(&usersResponse); err != nil {
		progress.StopSpinnerWithError("Failed to get users")
		return fmt.Errorf("failed to get users: %w", err)
	}

	progress.StopSpinner("Database users retrieved")

	// Extract users from response
	users, ok := usersResponse["users"].(bson.A)
	if !ok {
		return fmt.Errorf("unexpected response format from usersInfo command")
	}

	if len(users) == 0 {
		// Use structured output for "no results" message
		outputFormat := config.OutputText
		if format := cmd.Flag("output"); format != nil && format.Value.String() != "" {
			switch format.Value.String() {
			case "json":
				outputFormat = config.OutputJSON
			case "yaml":
				outputFormat = config.OutputYAML
			case "table":
				outputFormat = config.OutputTable
			}
		}

		formatter := output.NewFormatter(outputFormat, cmd.OutOrStdout())
		return formatter.Format(output.TableData{
			Headers: []string{"Info"},
			Rows:    [][]string{{fmt.Sprintf("No database users found in database '%s'", databaseName)}},
		})
	}

	// Format output
	outputFormat := config.OutputText
	if format := cmd.Flag("output"); format != nil && format.Value.String() != "" {
		switch format.Value.String() {
		case "json":
			outputFormat = config.OutputJSON
		case "yaml":
			outputFormat = config.OutputYAML
		case "table":
			outputFormat = config.OutputTable
		}
	}

	formatter := output.NewFormatter(outputFormat, cmd.OutOrStdout())
	return formatter.Format(users)
}

func runCreateDatabaseUser(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, username, password string, roles []string, useTempUser, showPassword bool) error {
	// MongoDB Atlas does not support direct database user creation via createUser command
	// All user management in Atlas must be done through the Atlas API
	return fmt.Errorf(`MongoDB Atlas does not support direct database user creation.

Database users in Atlas must be created through the Atlas UI or API.
Use the 'atlas users' commands instead:

  matlas atlas users create --username %s --password %s --project-id %s

For more information, see: https://docs.mongodb.com/atlas/security-add-mongodb-users/`, username, "[password]", projectID)
}

func runDeleteDatabaseUser(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, username string, useTempUser, yes bool) error {
	// MongoDB Atlas does not support direct database user management via dropUser command
	// All user management in Atlas must be done through the Atlas API
	return fmt.Errorf(`MongoDB Atlas does not support direct database user deletion.

Database users in Atlas must be managed through the Atlas UI or API.
Use the 'atlas users' commands instead:

  matlas atlas users delete --username %s --project-id %s

For more information, see: https://docs.mongodb.com/atlas/security-add-mongodb-users/`, username, projectID)
}

func runGetDatabaseUser(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, username string, useTempUser bool) error {
	// MongoDB Atlas does not support direct database user querying via usersInfo command
	// All user information in Atlas must be accessed through the Atlas API
	return fmt.Errorf(`MongoDB Atlas does not support direct database user querying.

Database user information in Atlas must be accessed through the Atlas UI or API.
Use the 'atlas users' commands instead:

  matlas atlas users get --username %s --project-id %s

For more information, see: https://docs.mongodb.com/atlas/security-add-mongodb-users/`, username, projectID)
}

func runUpdateDatabaseUser(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, username, password string, roles, addRoles, removeRoles []string, useTempUser bool) error {
	// MongoDB Atlas does not support direct database user updates via updateUser command
	// All user management in Atlas must be done through the Atlas API
	return fmt.Errorf(`MongoDB Atlas does not support direct database user updates.

Database users in Atlas must be managed through the Atlas UI or API.
Use the 'atlas users' commands instead:

  matlas atlas users update --username %s --project-id %s

For more information, see: https://docs.mongodb.com/atlas/security-add-mongodb-users/`, username, projectID)
}

func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Errorf("failed to mark flag %q required: %w", name, err))
	}
}

// Helper functions

func resolveConnectionInfoForUsers(ctx context.Context, cmd *cobra.Command, connectionString, clusterName, projectID string, useTempUser bool, databaseName string, progress *ui.ProgressIndicator) (*types.ConnectionInfo, error) {
	if connectionString != "" {
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

	cfg, err := config.Load(cmd, "")
	if err != nil {
		progress.StopSpinnerWithError("Failed to load configuration")
		return nil, cli.WrapWithSuggestion(err, "Check your configuration file")
	}

	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return nil, cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewClustersService(client)

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

		verbose := cmd.Flag("verbose").Changed

		// Parse custom roles from role flag (for user creation we need userAdmin privileges)
		var customRoles []admin.DatabaseUserRole
		roleStr, _ := cmd.Flags().GetString("role")
		if roleStr != "" {
			customRoles, err = parseSimpleRole(roleStr)
			if err != nil {
				progress.StopSpinnerWithError("Invalid role format")
				return nil, fmt.Errorf("invalid role format: %w", err)
			}
		} else {
			// Default to basic Atlas-supported roles for database operations
			// Note: Atlas does not support user creation via createUser command
			customRoles = []admin.DatabaseUserRole{
				{
					RoleName:     "readWriteAnyDatabase",
					DatabaseName: "admin",
				},
				{
					RoleName:     "dbAdminAnyDatabase",
					DatabaseName: "admin",
				},
			}
		}

		// Create temporary user manager
		usersService := atlas.NewDatabaseUsersService(client)
		tempUserManager := database.NewTempUserManager(usersService, projectID)

		// Create temporary user for user operations with admin privileges
		tempUser, err := tempUserManager.CreateTempUserForDiscoveryWithRoles(ctx, []string{clusterName}, databaseName, customRoles)
		if err != nil {
			progress.StopSpinnerWithError("Failed to create temporary user")
			return nil, fmt.Errorf("failed to create temporary user: %w", err)
		}

		progress.StopSpinner(fmt.Sprintf("Temporary user '%s' created (expires at %s)",
			tempUser.Username, tempUser.ExpiresAt.Format("15:04:05")))

		// Enhanced user propagation with retry logic specifically for user operations
		// User creation requires proper authentication and may need more time
		deadline, hasDeadline := ctx.Deadline()
		var propagationTime time.Duration

		if hasDeadline {
			remaining := time.Until(deadline)
			// Use 60% of remaining time for propagation for user operations
			propagationTime = time.Duration(float64(remaining) * 0.60)

			// Cap at reasonable limits - user operations may need more time
			if propagationTime > 180*time.Second {
				propagationTime = 180 * time.Second // Max 3 minutes for user operations
			} else if propagationTime < 30*time.Second {
				propagationTime = 30 * time.Second // Min 30 seconds
			}
		} else {
			// No context deadline, use a longer default for user operations
			propagationTime = 90 * time.Second // 1.5 minutes default
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

		// Add database path if not present and databaseName is specified
		if databaseName != "" && !strings.Contains(connectionString, "/"+databaseName) {
			if strings.Contains(connectionString, "?") {
				// Connection string has query parameters, insert database before them
				connectionString = strings.Replace(connectionString, "?", "/"+databaseName+"?", 1)
			} else {
				// No query parameters, append database
				connectionString += "/" + databaseName
			}
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

func parseUserRoles(roleStrs []string) ([]bson.M, error) {
	var roles []bson.M

	for _, roleStr := range roleStrs {
		parts := strings.Split(roleStr, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid role format '%s': expected 'role@database'", roleStr)
		}

		roleName := strings.TrimSpace(parts[0])
		dbName := strings.TrimSpace(parts[1])

		if roleName == "" || dbName == "" {
			return nil, fmt.Errorf("role name and database name cannot be empty in '%s'", roleStr)
		}

		roles = append(roles, bson.M{
			"role": roleName,
			"db":   dbName,
		})
	}

	return roles, nil
}

// parseSimpleRole parses a simple role string in format "role@database" or just "role" (defaults to admin)
func parseSimpleRole(roleStr string) ([]admin.DatabaseUserRole, error) {
	if roleStr == "" {
		return nil, nil
	}

	var roles []admin.DatabaseUserRole

	if strings.Contains(roleStr, "@") {
		// Format: role@database
		parts := strings.Split(roleStr, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid role format '%s': expected 'role@database'", roleStr)
		}

		roleName := strings.TrimSpace(parts[0])
		dbName := strings.TrimSpace(parts[1])

		if roleName == "" || dbName == "" {
			return nil, fmt.Errorf("invalid role format '%s': role and database cannot be empty", roleStr)
		}

		roles = append(roles, admin.DatabaseUserRole{
			RoleName:     roleName,
			DatabaseName: dbName,
		})
	} else {
		// Format: just role name (defaults to admin database)
		roleName := strings.TrimSpace(roleStr)
		if roleName == "" {
			return nil, fmt.Errorf("role name cannot be empty")
		}

		roles = append(roles, admin.DatabaseUserRole{
			RoleName:     roleName,
			DatabaseName: "admin",
		})
	}

	return roles, nil
}

// maskConnectionString masks sensitive information in connection strings for logging
func maskConnectionString(connStr string) string {
	if connStr == "" {
		return ""
	}

	// Simple credential masking
	if strings.Contains(connStr, "://") && strings.Contains(connStr, "@") {
		parts := strings.Split(connStr, "://")
		if len(parts) == 2 {
			afterProtocol := parts[1]
			if credIndex := strings.Index(afterProtocol, "@"); credIndex != -1 {
				beforeCreds := parts[0] + "://"
				afterCreds := afterProtocol[credIndex:]
				return beforeCreds + "***:***" + afterCreds
			}
		}
	}

	return connStr
}
