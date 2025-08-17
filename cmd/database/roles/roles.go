package roles

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/atlas-sdk/v20250312005/admin"
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

// NewRolesCmd creates the roles command
func NewRolesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "roles",
		Short: "Manage custom MongoDB roles",
		Long: `Create, list, and manage custom MongoDB roles within databases.
		
Custom roles can be created with specific privileges and actions, and then assigned to database users.
These are database-level roles, not Atlas-level user roles.`,
		Example: `  # List custom roles in a database
  matlas database roles list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb

  # Create a custom role
  matlas database roles create myCustomRole --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --privileges "read@mydb"

  # Create a custom role with multiple privileges
  matlas database roles create appRole --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database myapp --privileges "readWrite@myapp,read@logs"`,
	}

	cmd.AddCommand(
		newListRolesCmd(),
		newCreateRoleCmd(),
		newDeleteRoleCmd(),
		newGetRoleCmd(),
	)

	return cmd
}

func newListRolesCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var useTempUser bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List custom roles in a database",
		Long:  "List all custom roles defined in a MongoDB database",
		Example: `  # List custom roles using cluster reference
  matlas database roles list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb

  # List custom roles using connection string
  matlas database roles list --connection-string "mongodb+srv://user:pass@cluster.mongodb.net/mydb"

  # List custom roles with temporary user
  matlas database roles list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --use-temp-user`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListRoles(cmd, connectionString, clusterName, projectID, databaseName, useTempUser)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: dbAdmin@admin")

	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

func newCreateRoleCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var useTempUser bool
	var privileges []string
	var inheritedRoles []string

	cmd := &cobra.Command{
		Use:   "create <role-name>",
		Short: "Create a custom MongoDB role",
		Long: `Create a custom MongoDB role with specified privileges.
		
Privileges should be specified in the format 'action@resource' where:
- action: MongoDB action like 'read', 'readWrite', 'insert', 'update', 'remove', etc.
- resource: Database or collection like 'mydb' or 'mydb.mycollection'

For built-in privilege combinations, use:
- read@database: Equivalent to read role
- readWrite@database: Equivalent to readWrite role
- dbAdmin@database: Equivalent to dbAdmin role`,
		Args: cobra.ExactArgs(1),
		Example: `  # Create a custom role with read access to specific collections
  matlas database roles create reportReader --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database myapp \
    --privileges "find@myapp.reports,find@myapp.analytics"

  # Create a custom role with multiple database access
  matlas database roles create multiDbUser --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database myapp \
    --privileges "readWrite@myapp,read@logs"

  # Create a role that inherits from built-in roles
  matlas database roles create enhancedReader --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database myapp \
    --inherited-roles "read@myapp" --privileges "insert@myapp.auditLog"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			roleName := args[0]
			return runCreateRole(cmd, connectionString, clusterName, projectID, databaseName, roleName, privileges, inheritedRoles, useTempUser)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access")
	cmd.Flags().StringSliceVar(&privileges, "privileges", []string{}, "Privileges in format 'action@resource' (e.g., 'read@mydb', 'insert@mydb.collection')")
	cmd.Flags().StringSliceVar(&inheritedRoles, "inherited-roles", []string{}, "Roles to inherit from in format 'role@database'")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: dbAdmin@admin")

	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

func newDeleteRoleCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var useTempUser bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <role-name>",
		Short: "Delete a custom MongoDB role",
		Long:  "Delete a custom MongoDB role from a database",
		Args:  cobra.ExactArgs(1),
		Example: `  # Delete a custom role
  matlas database roles delete myCustomRole --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb

  # Delete a role without confirmation
  matlas database roles delete myCustomRole --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			roleName := args[0]
			return runDeleteRole(cmd, connectionString, clusterName, projectID, databaseName, roleName, useTempUser, yes)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: dbAdmin@admin")

	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

func newGetRoleCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var useTempUser bool

	cmd := &cobra.Command{
		Use:   "get <role-name>",
		Short: "Get details of a custom MongoDB role",
		Long:  "Get detailed information about a custom MongoDB role",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get role details
  matlas database roles get myCustomRole --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb`,
		RunE: func(cmd *cobra.Command, args []string) error {
			roleName := args[0]
			return runGetRole(cmd, connectionString, clusterName, projectID, databaseName, roleName, useTempUser)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&useTempUser, "use-temp-user", false, "Create temporary database user for access")
	cmd.Flags().String("role", "", "Role for temporary user (format: 'role@database' or just 'role' for admin). Use with --use-temp-user. Default: dbAdmin@admin")

	cmd.MarkFlagsOneRequired("connection-string", "cluster")
	cmd.MarkFlagsRequiredTogether("cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

// Role management functions

func runListRoles(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName string, useTempUser bool) error {
	ctx := context.Background()
	verbose := cmd.Flag("verbose").Changed
	progress := ui.NewProgressIndicator(verbose, false)
	progress.StartSpinner("Listing custom roles")
	defer progress.StopSpinner("Custom roles listed")

	// Get connection info (this will handle temp user creation if needed)
	connInfo, err := resolveConnectionInfoForRoles(ctx, cmd, connectionString, clusterName, projectID, useTempUser, databaseName, progress)
	if err != nil {
		progress.StopSpinnerWithError("Failed to resolve connection")
		return fmt.Errorf("failed to resolve connection: %w", err)
	}

	// Connect to MongoDB using connection string
	clientOptions := options.Client().ApplyURI(connInfo.ConnectionString)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		progress.StopSpinnerWithError("Failed to connect to MongoDB")
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer client.Disconnect(ctx)

	// Get the database
	db := client.Database(databaseName)

	// List custom roles using rolesInfo command
	result := db.RunCommand(ctx, bson.D{
		{Key: "rolesInfo", Value: 1},
		{Key: "showPrivileges", Value: true},
	})

	var rolesResponse bson.M
	if err := result.Decode(&rolesResponse); err != nil {
		progress.StopSpinnerWithError("Failed to get roles")
		return fmt.Errorf("failed to get roles: %w", err)
	}

	progress.StopSpinner("Custom roles retrieved")

	// Extract roles from response
	roles, ok := rolesResponse["roles"].(bson.A)
	if !ok {
		return fmt.Errorf("unexpected response format from rolesInfo command")
	}

	// Filter custom roles (exclude built-in roles)
	var customRoles []interface{}
	for _, role := range roles {
		if roleDoc, ok := role.(bson.M); ok {
			if roleName, ok := roleDoc["role"].(string); ok {
				// Skip built-in roles
				if !isBuiltInRole(roleName) {
					customRoles = append(customRoles, role)
				}
			}
		}
	}

	if len(customRoles) == 0 {
		fmt.Printf("No custom roles found in database '%s'\n", databaseName)
		return nil
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
	return formatter.Format(customRoles)
}

func runCreateRole(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, roleName string, privileges, inheritedRoles []string, useTempUser bool) error {
	ctx := context.Background()
	verbose := cmd.Flag("verbose").Changed
	progress := ui.NewProgressIndicator(verbose, false)
	progress.StartSpinner(fmt.Sprintf("Creating custom role '%s'", roleName))
	defer progress.StopSpinner(fmt.Sprintf("Custom role '%s' created", roleName))

	// Parse privileges and inherited roles
	rolePrivileges, err := parsePrivileges(privileges)
	if err != nil {
		progress.StopSpinnerWithError("Invalid privileges")
		return fmt.Errorf("invalid privileges: %w", err)
	}

	parsedInheritedRoles, err := parseInheritedRoles(inheritedRoles)
	if err != nil {
		progress.StopSpinnerWithError("Invalid inherited roles")
		return fmt.Errorf("invalid inherited roles: %w", err)
	}

	// Get connection info
	connInfo, err := resolveConnectionInfoForRoles(ctx, cmd, connectionString, clusterName, projectID, useTempUser, databaseName, progress)
	if err != nil {
		progress.StopSpinnerWithError("Failed to resolve connection")
		return fmt.Errorf("failed to resolve connection: %w", err)
	}

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(connInfo.ConnectionString)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		progress.StopSpinnerWithError("Failed to connect to MongoDB")
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer client.Disconnect(ctx)

	// Get the database
	db := client.Database(databaseName)

	// Create role using createRole command with enhanced retry logic
	createRoleDoc := bson.D{
		{Key: "createRole", Value: roleName},
		{Key: "privileges", Value: rolePrivileges},
		{Key: "roles", Value: parsedInheritedRoles},
	}

	// Enhanced retry logic for role creation
	maxRetries := 3
	retryDelay := 30 * time.Second
	verbose = cmd.Flag("verbose").Changed

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if verbose {
			fmt.Printf("ℹ Role creation attempt %d of %d...\n", attempt, maxRetries)
		}

		result := db.RunCommand(ctx, createRoleDoc)
		if result.Err() != nil {
			if verbose {
				fmt.Printf("⚠ Role creation attempt %d failed with exit code 1\n", attempt)
			}

			// Check if this is an authentication error
			errMsg := result.Err().Error()
			isAuthError := strings.Contains(errMsg, "not authorized") ||
				strings.Contains(errMsg, "Unauthorized") ||
				strings.Contains(errMsg, "authentication failed")

			if isAuthError && attempt < maxRetries {
				if verbose {
					fmt.Printf("ℹ Authentication error detected - waiting additional %ds for user propagation...\n", int(retryDelay.Seconds()))
				}

				// Wait for additional user propagation
				select {
				case <-ctx.Done():
					progress.StopSpinnerWithError("Operation cancelled during retry")
					return fmt.Errorf("operation cancelled during retry: %w", ctx.Err())
				case <-time.After(retryDelay):
					// Continue to next attempt
				}
				continue
			} else {
				// Non-auth error or final attempt
				if isAuthError {
					progress.StopSpinnerWithError("Authentication error persists after all retry attempts")
					// Check if this might be due to Atlas cluster tier limitations
					errMsg := result.Err().Error()
					if strings.Contains(errMsg, "not authorized") && strings.Contains(errMsg, "createRole") {
						return fmt.Errorf("failed to create role: %w\n\nMongoDB Atlas Limitation: Custom role creation is not supported on M0, M10+, and Flex clusters.\nIf you're using one of these cluster tiers, this operation is restricted by MongoDB Atlas.\nFor custom roles, consider upgrading to a dedicated cluster tier that supports this feature.\n\nAlternatively, this could be due to:\n• User propagation delay (Atlas users take time to sync across cluster nodes)\n• Incorrect credentials or insufficient permissions\n• Try using --use-temp-user flag for automatic user creation\n• Use --verbose for detailed error information", result.Err())
					}
					return fmt.Errorf("failed to create role: %w\n\nMongoDB authentication failed. This could be due to:\n• User propagation delay (Atlas users take time to sync across cluster nodes)\n• Incorrect credentials or insufficient permissions\n• Try using --use-temp-user flag for automatic user creation\n• Use --verbose for detailed error information", result.Err())
				} else {
					progress.StopSpinnerWithError("Failed to create role")
					return fmt.Errorf("failed to create role: %w", result.Err())
				}
			}
		} else {
			// Success
			break
		}
	}

	progress.StopSpinner(fmt.Sprintf("Custom role '%s' created successfully", roleName))
	fmt.Printf("✓ Custom role '%s' created successfully in database '%s'\n", roleName, databaseName)
	return nil
}

func runDeleteRole(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, roleName string, useTempUser, yes bool) error {
	ctx := context.Background()
	verbose := cmd.Flag("verbose").Changed

	if !yes {
		confirmPrompt := ui.NewConfirmationPrompt(false, false)
		confirmed, err := confirmPrompt.Confirm(fmt.Sprintf("Delete custom role '%s' from database '%s'?", roleName, databaseName))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	progress := ui.NewProgressIndicator(verbose, false)
	progress.StartSpinner(fmt.Sprintf("Deleting custom role '%s'", roleName))
	defer progress.StopSpinner(fmt.Sprintf("Custom role '%s' deleted", roleName))

	// Get connection info
	connInfo, err := resolveConnectionInfoForRoles(ctx, cmd, connectionString, clusterName, projectID, useTempUser, databaseName, progress)
	if err != nil {
		progress.StopSpinnerWithError("Failed to resolve connection")
		return fmt.Errorf("failed to resolve connection: %w", err)
	}

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(connInfo.ConnectionString)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		progress.StopSpinnerWithError("Failed to connect to MongoDB")
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer client.Disconnect(ctx)

	// Get the database
	db := client.Database(databaseName)

	// Delete role using dropRole command
	result := db.RunCommand(ctx, bson.D{
		{Key: "dropRole", Value: roleName},
	})
	if result.Err() != nil {
		progress.StopSpinnerWithError("Failed to delete role")
		return fmt.Errorf("failed to delete role: %w", result.Err())
	}

	progress.StopSpinner(fmt.Sprintf("Custom role '%s' deleted successfully", roleName))
	fmt.Printf("✓ Custom role '%s' deleted successfully from database '%s'\n", roleName, databaseName)
	return nil
}

func runGetRole(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, roleName string, useTempUser bool) error {
	ctx := context.Background()
	verbose := cmd.Flag("verbose").Changed
	progress := ui.NewProgressIndicator(verbose, false)
	progress.StartSpinner(fmt.Sprintf("Getting role '%s' details", roleName))
	defer progress.StopSpinner(fmt.Sprintf("Role '%s' details retrieved", roleName))

	// Get connection info
	connInfo, err := resolveConnectionInfoForRoles(ctx, cmd, connectionString, clusterName, projectID, useTempUser, databaseName, progress)
	if err != nil {
		progress.StopSpinnerWithError("Failed to resolve connection")
		return fmt.Errorf("failed to resolve connection: %w", err)
	}

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(connInfo.ConnectionString)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		progress.StopSpinnerWithError("Failed to connect to MongoDB")
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer client.Disconnect(ctx)

	// Get the database
	db := client.Database(databaseName)

	// Get role details using rolesInfo command
	result := db.RunCommand(ctx, bson.D{
		{Key: "rolesInfo", Value: roleName},
		{Key: "showPrivileges", Value: true},
	})

	var rolesResponse bson.M
	if err := result.Decode(&rolesResponse); err != nil {
		progress.StopSpinnerWithError("Failed to get role details")
		return fmt.Errorf("failed to get role details: %w", err)
	}

	progress.StopSpinner(fmt.Sprintf("Role '%s' details retrieved", roleName))

	// Extract role from response
	roles, ok := rolesResponse["roles"].(bson.A)
	if !ok || len(roles) == 0 {
		return fmt.Errorf("role '%s' not found in database '%s'", roleName, databaseName)
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
	return formatter.Format(roles[0])
}

// Helper functions

func resolveConnectionInfoForRoles(ctx context.Context, cmd *cobra.Command, connectionString, clusterName, projectID string, useTempUser bool, databaseName string, progress *ui.ProgressIndicator) (*types.ConnectionInfo, error) {
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

		// Parse custom roles from role flag (for role creation we need admin privileges)
		var customRoles []admin.DatabaseUserRole
		roleStr, _ := cmd.Flags().GetString("role")
		if roleStr != "" {
			customRoles, err = parseSimpleRole(roleStr)
			if err != nil {
				progress.StopSpinnerWithError("Invalid role format")
				return nil, fmt.Errorf("invalid role format: %w", err)
			}
		} else {
			// Default to Atlas-supported roles for role creation operations
			// Note: Atlas doesn't support userAdminAnyDatabase for database users
			// Use atlasAdmin which provides broader administrative access
			customRoles = []admin.DatabaseUserRole{
				{
					RoleName:     "atlasAdmin",
					DatabaseName: "admin",
				},
			}
		}

		// Create temporary user manager
		usersService := atlas.NewDatabaseUsersService(client)
		tempUserManager := database.NewTempUserManager(usersService, projectID)

		// Create temporary user for role operations with admin privileges
		tempUser, err := tempUserManager.CreateTempUserForDiscoveryWithRoles(ctx, []string{clusterName}, databaseName, customRoles)
		if err != nil {
			progress.StopSpinnerWithError("Failed to create temporary user")
			return nil, fmt.Errorf("failed to create temporary user: %w", err)
		}

		progress.StopSpinner(fmt.Sprintf("Temporary user '%s' created (expires at %s)",
			tempUser.Username, tempUser.ExpiresAt.Format("15:04:05")))

		// Enhanced user propagation with retry logic specifically for role operations
		// Role creation requires proper authentication and may need more time
		deadline, hasDeadline := ctx.Deadline()
		var propagationTime time.Duration

		if hasDeadline {
			remaining := time.Until(deadline)
			// Use 60% of remaining time for propagation for role operations
			propagationTime = time.Duration(float64(remaining) * 0.60)

			// Cap at reasonable limits - role operations may need more time
			if propagationTime > 180*time.Second {
				propagationTime = 180 * time.Second // Max 3 minutes for role operations
			} else if propagationTime < 30*time.Second {
				propagationTime = 30 * time.Second // Min 30 seconds
			}
		} else {
			// No context deadline, use a longer default for role operations
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

func parsePrivileges(privilegeStrs []string) ([]bson.M, error) {
	var privileges []bson.M

	for _, privStr := range privilegeStrs {
		parts := strings.Split(privStr, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid privilege format '%s': expected 'action@resource'", privStr)
		}

		action := strings.TrimSpace(parts[0])
		resource := strings.TrimSpace(parts[1])

		if action == "" || resource == "" {
			return nil, fmt.Errorf("action and resource cannot be empty in privilege '%s'", privStr)
		}

		// Handle special built-in privilege combinations
		var actions []string
		switch action {
		case "read":
			actions = []string{"find", "listIndexes", "listCollections"}
		case "readWrite":
			actions = []string{"find", "insert", "update", "remove", "createIndex", "dropIndex", "listIndexes", "listCollections"}
		case "dbAdmin":
			actions = []string{"listCollections", "listIndexes", "dbStats", "collStats", "createIndex", "dropIndex"}
		default:
			actions = []string{action}
		}

		// Parse resource (can be database or database.collection)
		var resourceDoc bson.M
		if strings.Contains(resource, ".") {
			// Collection-level resource
			parts := strings.SplitN(resource, ".", 2)
			resourceDoc = bson.M{
				"db":         parts[0],
				"collection": parts[1],
			}
		} else {
			// Database-level resource
			resourceDoc = bson.M{
				"db": resource,
			}
		}

		// Create privilege document
		privilege := bson.M{
			"resource": resourceDoc,
			"actions":  actions,
		}

		privileges = append(privileges, privilege)
	}

	return privileges, nil
}

func parseInheritedRoles(roleStrs []string) ([]bson.M, error) {
	var roles []bson.M

	for _, roleStr := range roleStrs {
		parts := strings.Split(roleStr, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid inherited role format '%s': expected 'role@database'", roleStr)
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

func isBuiltInRole(roleName string) bool {
	builtInRoles := []string{
		"read", "readWrite", "dbAdmin", "dbOwner", "userAdmin", "userAdminAnyDatabase",
		"readAnyDatabase", "readWriteAnyDatabase", "dbAdminAnyDatabase", "clusterAdmin",
		"clusterManager", "clusterMonitor", "hostManager", "backup", "restore", "root",
	}

	for _, builtIn := range builtInRoles {
		if roleName == builtIn {
			return true
		}
	}
	return false
}

func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Errorf("failed to mark flag %q required: %w", name, err))
	}
}
