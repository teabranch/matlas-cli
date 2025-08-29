package users

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
	"golang.org/x/term"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func NewUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage Atlas database users",
		Long: `List, create, and manage MongoDB Atlas database users.

These are Atlas-managed database users created via the Atlas Admin API. They are
assigned built-in MongoDB roles and managed centrally at the project level.

For database-specific users with custom roles, use 'matlas database users' commands.`,
		Aliases: []string{"user"},
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var projectID string
	var paginationFlags cli.PaginationFlags

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List database users",
		Long: `List all database users in a project.

This command retrieves and displays all MongoDB Atlas database users for the specified project.
The output includes username, authentication database, and assigned roles.`,
		Example: `  # List database users in a project
  matlas atlas users list --project-id 507f1f77bcf86cd799439011

  # List with pagination
  matlas atlas users list --project-id 507f1f77bcf86cd799439011 --page 2 --limit 10

  # Output as JSON for automation
  matlas atlas users list --project-id 507f1f77bcf86cd799439011 --output json

  # Using alias
  matlas atlas users ls --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListUsers(cmd, projectID, &paginationFlags)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	cli.AddPaginationFlags(cmd, &paginationFlags)

	return cmd
}

func newGetCmd() *cobra.Command {
	var projectID string
	var databaseName string

	cmd := &cobra.Command{
		Use:   "get <username>",
		Short: "Get database user details",
		Long:  "Get detailed information about a specific database user",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get database user details
  matlas atlas users get myuser --project-id 507f1f77bcf86cd799439011 --database-name admin

  # Output as YAML
  matlas atlas users get myuser --project-id 507f1f77bcf86cd799439011 --database-name admin --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			return runGetUser(cmd, projectID, databaseName, username)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&databaseName, "database-name", "admin", "Authentication database name")

	return cmd
}

func newCreateCmd() *cobra.Command {
	var projectID string
	var databaseName string
	var username string
	var password string
	var roles []string
	var showPassword bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a database user",
		Long:  "Create a new MongoDB Atlas database user",
		Example: `  # Create a read-only user
  matlas atlas users create --project-id 507f1f77bcf86cd799439011 --username myuser --database-name admin --roles readWriteAnyDatabase@admin

  # Create user with multiple roles
  matlas atlas users create --project-id 507f1f77bcf86cd799439011 --username myuser --database-name admin --roles read@mydb,readWrite@anotherdb

  # Create user with password prompt and show the password
  matlas atlas users create --project-id 507f1f77bcf86cd799439011 --username myuser --database-name admin --roles readWriteAnyDatabase@admin --show-password`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateUser(cmd, projectID, databaseName, username, password, roles, showPassword)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&databaseName, "database-name", "admin", "Authentication database name")
	cmd.Flags().StringVar(&username, "username", "", "Database username (required)")
	cmd.Flags().StringVar(&password, "password", "", "Database password (will prompt if not provided)")
	cmd.Flags().StringSliceVar(&roles, "roles", []string{}, "Database roles in format roleName@databaseName (required)")
	cmd.Flags().BoolVar(&showPassword, "show-password", false, "Print the user password after creation")

	mustMarkFlagRequired(cmd, "username")
	mustMarkFlagRequired(cmd, "roles")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var projectID string
	var databaseName string
	var password string
	var roles []string

	cmd := &cobra.Command{
		Use:   "update <username>",
		Short: "Update a database user",
		Long:  "Update an existing MongoDB Atlas database user",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update user roles
  matlas atlas users update myuser --project-id 507f1f77bcf86cd799439011 --database-name admin --roles readWriteAnyDatabase@admin

  # Update user password (will prompt)
  matlas atlas users update myuser --project-id 507f1f77bcf86cd799439011 --database-name admin --password`,
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			return runUpdateUser(cmd, projectID, databaseName, username, password, roles)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&databaseName, "database-name", "admin", "Authentication database name")
	cmd.Flags().StringVar(&password, "password", "", "New database password (will prompt if flag provided without value)")
	cmd.Flags().StringSliceVar(&roles, "roles", []string{}, "New database roles in format roleName@databaseName")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var projectID string
	var databaseName string
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <username>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a database user",
		Long: `Delete a MongoDB Atlas database user.

This command permanently removes a database user from the specified project.
Use with caution as this operation cannot be undone.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Delete a database user (with confirmation)
  matlas atlas users delete myuser --project-id 507f1f77bcf86cd799439011 --database-name admin

  # Delete without confirmation (automation use)
  matlas atlas users delete myuser --project-id 507f1f77bcf86cd799439011 --database-name admin --yes

  # Using alias
  matlas atlas users rm myuser --project-id 507f1f77bcf86cd799439011 --database-name admin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			return runDeleteUser(cmd, projectID, databaseName, username, yes)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&databaseName, "database-name", "admin", "Authentication database name")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")

	return cmd
}

func runListUsers(cmd *cobra.Command, projectID string, paginationFlags *cli.PaginationFlags) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate project ID
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	// Validate pagination
	paginationOpts, err := paginationFlags.Validate()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	progress.StartSpinner("Fetching database users...")

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewDatabaseUsersService(client)

	// Fetch users with server-side pagination when available
	users, err := service.ListWithPagination(ctx, projectID, paginationOpts.Page, paginationOpts.Limit, paginationFlags.All)
	if err != nil {
		progress.StopSpinnerWithError("Failed to fetch database users")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("Database users retrieved successfully")

	// Apply pagination if needed
	if paginationOpts.ShouldPaginate() && !paginationFlags.All {
		skip := paginationOpts.CalculateSkip()
		end := skip + paginationOpts.Limit

		if skip >= len(users) {
			users = []admin.CloudDatabaseUser{}
		} else {
			if end > len(users) {
				end = len(users)
			}
			users = users[skip:end]
		}
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, users,
		[]string{"USERNAME", "DATABASE", "ROLES"},
		func(item interface{}) []string {
			user := item.(admin.CloudDatabaseUser)
			username := user.Username
			database := user.DatabaseName
			roles := formatRoles(user.Roles)

			return []string{username, database, roles}
		})
}

func runGetUser(cmd *cobra.Command, projectID, databaseName, username string) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	if err := validation.ValidateUsername(username); err != nil {
		return cli.FormatValidationError("username", username, err.Error())
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	progress.StartSpinner(fmt.Sprintf("Fetching user '%s'...", username))

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewDatabaseUsersService(client)

	// Fetch user
	user, err := service.Get(ctx, projectID, databaseName, username)
	if err != nil {
		progress.StopSpinnerWithError(fmt.Sprintf("Failed to fetch user '%s'", username))
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("User '%s' retrieved successfully", username))

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(user)
}

func runCreateUser(cmd *cobra.Command, projectID, databaseName, username, password string, roles []string, showPassword bool) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	if err := validation.ValidateUsername(username); err != nil {
		return cli.FormatValidationError("username", username, err.Error())
	}

	if databaseName == "" {
		return cli.FormatValidationError("database-name", databaseName, "database name cannot be empty")
	}

	if len(roles) == 0 {
		return cli.FormatValidationError("roles", "", "at least one role must be specified")
	}

	// Parse roles
	parsedRoles, err := parseRoles(roles)
	if err != nil {
		return cli.FormatValidationError("roles", strings.Join(roles, ","), err.Error())
	}

	// Handle password (prompt if not provided)
	if password == "" {
		password, err = readPassword("Enter password for database user: ")
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		if password == "" {
			return fmt.Errorf("password cannot be empty")
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Creating database user '%s'...", username))

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewDatabaseUsersService(client)

	// Create user object
	user := &admin.CloudDatabaseUser{
		Username:     username,
		DatabaseName: databaseName,
		Password:     admin.PtrString(password),
		Roles:        &parsedRoles,
	}

	// Create the user
	createdUser, err := service.Create(ctx, projectID, user)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create database user")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("")

	// Display created user details with prettier formatting
	formatter := output.NewCreateResultFormatter(cfg.Output, os.Stdout)
	if showPassword {
		return formatter.FormatCreateResultWithPassword(createdUser, "database user", password)
	}
	return formatter.FormatCreateResult(createdUser, "database user")
}

func runUpdateUser(cmd *cobra.Command, projectID, databaseName, username, password string, roles []string) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	if err := validation.ValidateUsername(username); err != nil {
		return cli.FormatValidationError("username", username, err.Error())
	}

	if databaseName == "" {
		return cli.FormatValidationError("database-name", databaseName, "database name cannot be empty")
	}

	// Must have at least one thing to update
	if password == "" && len(roles) == 0 {
		return fmt.Errorf("at least one of --password or --roles must be specified for update")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewDatabaseUsersService(client)

	// First, get the existing user to preserve unchanged fields
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Fetching existing user '%s'...", username))

	existingUser, err := service.Get(ctx, projectID, databaseName, username)
	if err != nil {
		progress.StopSpinnerWithError("Failed to fetch existing user")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	// Create update object starting from existing user
	updateUser := &admin.CloudDatabaseUser{
		Username:     existingUser.Username,
		DatabaseName: existingUser.DatabaseName,
		Roles:        existingUser.Roles,
	}

	// Handle password update
	if password != "" {
		updateUser.Password = admin.PtrString(password)
	} else if cmd.Flags().Lookup("password").Changed && password == "" {
		// Password flag was provided but no value, prompt for it
		password, err = readPassword("Enter new password for database user: ")
		if err != nil {
			progress.StopSpinnerWithError("Failed to read password")
			return fmt.Errorf("failed to read password: %w", err)
		}
		if password == "" {
			progress.StopSpinnerWithError("Password cannot be empty")
			return fmt.Errorf("password cannot be empty")
		}
		updateUser.Password = admin.PtrString(password)
	}

	// Handle roles update
	if len(roles) > 0 {
		parsedRoles, err := parseRoles(roles)
		if err != nil {
			progress.StopSpinnerWithError("Invalid roles format")
			return cli.FormatValidationError("roles", strings.Join(roles, ","), err.Error())
		}
		updateUser.Roles = &parsedRoles
	}

	progress.StopSpinner("Existing user fetched")
	progress.StartSpinner(fmt.Sprintf("Updating database user '%s'...", username))

	// Update the user
	updatedUser, err := service.Update(ctx, projectID, databaseName, username, updateUser)
	if err != nil {
		progress.StopSpinnerWithError("Failed to update database user")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Database user '%s' updated successfully", username))

	// Display updated user details
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(updatedUser)
}

func runDeleteUser(cmd *cobra.Command, projectID, databaseName, username string, yes bool) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	if username == "" {
		return cli.FormatValidationError("username", username, "username cannot be empty")
	}

	if databaseName == "" {
		return cli.FormatValidationError("database-name", databaseName, "database name cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Initialize Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	// Create database users service
	usersService := atlas.NewDatabaseUsersService(client)

	// Confirmation prompt (unless --yes flag is used)
	if !yes {
		confirmPrompt := ui.NewConfirmationPrompt(false, false)
		confirmed, err := confirmPrompt.Confirm(fmt.Sprintf("Are you sure you want to delete database user '%s' from database '%s' in project '%s'? This action cannot be undone.", username, databaseName, projectID))
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("User deletion cancelled.")
			return nil
		}
	}

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Deleting database user '%s'...", username))

	// Delete the database user
	err = usersService.Delete(ctx, projectID, databaseName, username)
	if err != nil {
		progress.StopSpinnerWithError("Failed to delete database user")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Database user '%s' deleted successfully", username))
	return nil
}

// Helper functions
func getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func formatRoles(roles *[]admin.DatabaseUserRole) string {
	if roles == nil || len(*roles) == 0 {
		return ""
	}

	var roleStrings []string
	for _, role := range *roles {
		roleName := role.RoleName
		databaseName := role.DatabaseName
		if roleName != "" && databaseName != "" {
			roleStrings = append(roleStrings, fmt.Sprintf("%s@%s", roleName, databaseName))
		}
	}

	return strings.Join(roleStrings, ", ")
}

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Add newline after password input
	if err != nil {
		return "", err
	}
	return string(password), nil
}

// mustMarkFlagRequired marks a flag as required and panics if it fails.
// This should never fail in normal execution and indicates a programmer error if it does.
func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Errorf("failed to mark flag %q required: %w", name, err))
	}
}

// parseRoles converts role strings in format "roleName@databaseName" to Atlas SDK role objects
func parseRoles(roleStrings []string) ([]admin.DatabaseUserRole, error) {
	var roles []admin.DatabaseUserRole

	for _, roleStr := range roleStrings {
		parts := strings.Split(roleStr, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid role format '%s': expected 'roleName@databaseName'", roleStr)
		}

		roleName := strings.TrimSpace(parts[0])
		databaseName := strings.TrimSpace(parts[1])

		if roleName == "" || databaseName == "" {
			return nil, fmt.Errorf("invalid role format '%s': role name and database name cannot be empty", roleStr)
		}

		roles = append(roles, admin.DatabaseUserRole{
			RoleName:     roleName,
			DatabaseName: databaseName,
		})
	}

	if len(roles) == 0 {
		return nil, fmt.Errorf("at least one role must be specified")
	}

	return roles, nil
}
