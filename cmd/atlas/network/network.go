package network

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func NewNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage Atlas network access",
		Long:  "Manage MongoDB Atlas network access lists and IP whitelisting",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func runListNetworkAccess(cmd *cobra.Command, projectID string) error {
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

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner("Fetching network access entries...")

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkAccessListsService(client)

	// Fetch network access entries
	entries, err := service.List(ctx, projectID)
	if err != nil {
		progress.StopSpinnerWithError("Failed to fetch network access entries")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("Network access entries retrieved successfully")

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, entries,
		[]string{"IP ADDRESS", "CIDR BLOCK", "AWS SECURITY GROUP", "COMMENT"},
		func(item interface{}) []string {
			entry := item.(admin.NetworkPermissionEntry)
			ipAddress := getStringValue(entry.IpAddress)
			cidrBlock := getStringValue(entry.CidrBlock)
			awsSecurityGroup := getStringValue(entry.AwsSecurityGroup)
			comment := getStringValue(entry.Comment)

			return []string{ipAddress, cidrBlock, awsSecurityGroup, comment}
		})
}

func runGetNetworkAccess(cmd *cobra.Command, projectID, ipAddress string) error {
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

	if ipAddress == "" {
		return cli.FormatValidationError("ip-address", ipAddress, "IP address cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Fetching network access entry '%s'...", ipAddress))

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkAccessListsService(client)

	// Fetch network access entry
	entry, err := service.Get(ctx, projectID, ipAddress)
	if err != nil {
		progress.StopSpinnerWithError(fmt.Sprintf("Failed to fetch network access entry '%s'", ipAddress))
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Network access entry '%s' retrieved successfully", ipAddress))

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(entry)
}

func runCreateNetworkAccess(cmd *cobra.Command, projectID, ipAddress, cidrBlock, awsSecurityGroup, comment string) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate project ID first
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	// Validate that exactly one access type is specified
	accessTypesCount := 0
	if ipAddress != "" {
		accessTypesCount++
	}
	if cidrBlock != "" {
		accessTypesCount++
	}
	if awsSecurityGroup != "" {
		accessTypesCount++
	}

	if accessTypesCount == 0 {
		return fmt.Errorf("exactly one of --ip-address, --cidr-block, or --aws-security-group must be specified")
	}
	if accessTypesCount > 1 {
		return fmt.Errorf("only one of --ip-address, --cidr-block, or --aws-security-group can be specified")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkAccessListsService(client)

	// Create entry object
	entry := admin.NetworkPermissionEntry{}

	if ipAddress != "" {
		entry.IpAddress = &ipAddress
		progress.StartSpinner(fmt.Sprintf("Creating network access entry for IP address '%s'...", ipAddress))
	} else if cidrBlock != "" {
		entry.CidrBlock = &cidrBlock
		progress.StartSpinner(fmt.Sprintf("Creating network access entry for CIDR block '%s'...", cidrBlock))
	} else if awsSecurityGroup != "" {
		entry.AwsSecurityGroup = &awsSecurityGroup
		progress.StartSpinner(fmt.Sprintf("Creating network access entry for AWS security group '%s'...", awsSecurityGroup))
	}

	if comment != "" {
		entry.Comment = &comment
	}

	// Create the entry
	result, err := service.Create(ctx, projectID, []admin.NetworkPermissionEntry{entry})
	if err != nil {
		progress.StopSpinnerWithError("Failed to create network access entry")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("")

	// Display created entry details with prettier formatting
	formatter := output.NewCreateResultFormatter(cfg.Output, os.Stdout)
	return formatter.FormatCreateResult(result, "network access entry")
}

func runDeleteNetworkAccess(cmd *cobra.Command, projectID, ipAddress string, yes bool) error {
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

	if ipAddress == "" {
		return cli.FormatValidationError("ip-address", ipAddress, "IP address cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkAccessListsService(client)

	// Confirmation prompt (unless --yes flag is used)
	if !yes {
		confirmPrompt := ui.NewConfirmationPrompt(false, false)
		confirmed, err := confirmPrompt.Confirm(fmt.Sprintf("Are you sure you want to delete network access entry '%s' from project '%s'? This action cannot be undone.", ipAddress, projectID))
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("Network access entry deletion cancelled.")
			return nil
		}
	}

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Deleting network access entry '%s'...", ipAddress))

	// Delete the network access entry
	err = service.Delete(ctx, projectID, ipAddress)
	if err != nil {
		progress.StopSpinnerWithError("Failed to delete network access entry")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Network access entry '%s' deleted successfully", ipAddress))
	return nil
}

// Helper function
func getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

func newListCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List network access entries",
		Long: `List all network access list entries in a project.

This command retrieves and displays all MongoDB Atlas network access list entries
for the specified project. The output includes IP addresses, CIDR blocks, and 
AWS security groups that are allowed to connect to your Atlas clusters.`,
		Example: `  # List network access entries
  matlas atlas network list --project-id 507f1f77bcf86cd799439011

  # Output as JSON for automation
  matlas atlas network list --project-id 507f1f77bcf86cd799439011 --output json

  # Using alias
  matlas atlas network ls --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListNetworkAccess(cmd, projectID)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	return cmd
}

func newGetCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "get <ip-address>",
		Short: "Get network access entry details",
		Long:  "Get detailed information about a specific network access list entry",
		Args:  cobra.ExactArgs(1),
		Example: `  # Get network access entry details
  matlas atlas network get 192.168.1.1 --project-id 507f1f77bcf86cd799439011

  # Get CIDR block entry
  matlas atlas network get 192.168.1.0/24 --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ipAddress := args[0]
			return runGetNetworkAccess(cmd, projectID, ipAddress)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	return cmd
}

func newCreateCmd() *cobra.Command {
	var projectID string
	var ipAddress string
	var cidrBlock string
	var awsSecurityGroup string
	var comment string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create network access entry",
		Long: `Create a new network access list entry.

You can add access for IP addresses, CIDR blocks, or AWS security groups.
Only one type of access can be specified per entry.`,
		Example: `  # Allow access from a single IP address
  matlas atlas network create --project-id 507f1f77bcf86cd799439011 --ip-address 192.168.1.1 --comment "Office IP"

  # Allow access from a CIDR block
  matlas atlas network create --project-id 507f1f77bcf86cd799439011 --cidr-block 192.168.1.0/24 --comment "Office subnet"

  # Allow access from AWS security group
  matlas atlas network create --project-id 507f1f77bcf86cd799439011 --aws-security-group sg-12345678 --comment "Production servers"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateNetworkAccess(cmd, projectID, ipAddress, cidrBlock, awsSecurityGroup, comment)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&ipAddress, "ip-address", "", "IP address to allow access from")
	cmd.Flags().StringVar(&cidrBlock, "cidr-block", "", "CIDR block to allow access from")
	cmd.Flags().StringVar(&awsSecurityGroup, "aws-security-group", "", "AWS security group to allow access from")
	cmd.Flags().StringVar(&comment, "comment", "", "Optional comment for the network access entry")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var projectID string
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <ip-address>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete network access entry",
		Long:    "Delete a network access list entry by IP address or CIDR block",
		Args:    cobra.ExactArgs(1),
		Example: `  # Delete network access entry with confirmation
  matlas atlas network delete 192.168.1.1 --project-id 507f1f77bcf86cd799439011

  # Delete without confirmation prompt
  matlas atlas network delete 192.168.1.0/24 --project-id 507f1f77bcf86cd799439011 --yes

  # Using alias
  matlas atlas network rm 192.168.1.1 --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ipAddress := args[0]
			return runDeleteNetworkAccess(cmd, projectID, ipAddress, yes)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")

	mustMarkFlagRequired(cmd, "project-id")

	return cmd
}

// mustMarkFlagRequired marks a flag as required and panics if it fails.
// This should never fail in normal execution and indicates a programmer error if it does.
func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Errorf("failed to mark flag %q required: %w", name, err))
	}
}
