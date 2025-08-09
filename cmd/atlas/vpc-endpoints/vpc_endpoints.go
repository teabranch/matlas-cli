package vpcendpoints

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func NewVPCEndpointsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vpc-endpoints",
		Short:   "Manage Atlas VPC endpoints (unsupported in this build)",
		Long:    "Atlas VPC endpoints and Private Link connections are not yet supported in this build due to missing SDK coverage.",
		Aliases: []string{"vpc-endpoint", "vpc"},
		Hidden:  true,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var projectID string
	var paginationFlags cli.PaginationFlags

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List VPC endpoints (unsupported)",
		Long:    "Listing VPC endpoints is not yet supported in this build.",
		Example: `  # List VPC endpoints in a project
  matlas atlas vpc-endpoints list --project-id 507f1f77bcf86cd799439011

  # List with pagination
  matlas atlas vpc-endpoints list --project-id 507f1f77bcf86cd799439011 --page 2 --limit 10

  # List all endpoints (no pagination)
  matlas atlas vpc-endpoints list --project-id 507f1f77bcf86cd799439011 --all

  # Output as JSON
  matlas atlas vpc-endpoints list --project-id 507f1f77bcf86cd799439011 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.UnsupportedFeatureError("VPC endpoints", "Requires PrivateEndpointServicesApi in SDK")
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	cli.AddPaginationFlags(cmd, &paginationFlags)

	return cmd
}

func newGetCmd() *cobra.Command {
	var projectID string
	var endpointID string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get VPC endpoint details (unsupported)",
		Long:  "Getting VPC endpoint details is not yet supported in this build.",
		Example: `  # Get VPC endpoint details
  matlas atlas vpc-endpoints get --project-id 507f1f77bcf86cd799439011 --endpoint-id 5e2211c17a3e5a48f5497de3

  # Output as YAML
  matlas atlas vpc-endpoints get --project-id 507f1f77bcf86cd799439011 --endpoint-id 5e2211c17a3e5a48f5497de3 --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.UnsupportedFeatureError("VPC endpoints", "Requires PrivateEndpointServicesApi in SDK")
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&endpointID, "endpoint-id", "", "VPC endpoint ID (required)")
	cmd.MarkFlagRequired("endpoint-id")

	return cmd
}

func newCreateCmd() *cobra.Command {
	var projectID string
	var cloudProvider string
	var region string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VPC endpoint (unsupported)",
		Long:  "Creating VPC endpoints is not yet supported in this build.",
		Example: `  # Create VPC endpoint for AWS
  matlas atlas vpc-endpoints create --project-id 507f1f77bcf86cd799439011 --cloud-provider AWS --region US_EAST_1

  # Create VPC endpoint for Azure
  matlas atlas vpc-endpoints create --project-id 507f1f77bcf86cd799439011 --cloud-provider AZURE --region eastus`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.UnsupportedFeatureError("VPC endpoints", "Requires PrivateEndpointServicesApi in SDK")
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider (AWS, AZURE, GCP) (required)")
	cmd.Flags().StringVar(&region, "region", "", "Cloud provider region (required)")
	cmd.MarkFlagRequired("cloud-provider")
	cmd.MarkFlagRequired("region")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var projectID string
	var endpointID string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a VPC endpoint (unsupported)",
		Long:  "Deleting VPC endpoints is not yet supported in this build.",
		Example: `  # Delete VPC endpoint with confirmation
  matlas atlas vpc-endpoints delete --project-id 507f1f77bcf86cd799439011 --endpoint-id 5e2211c17a3e5a48f5497de3

  # Delete without confirmation prompt
  matlas atlas vpc-endpoints delete --project-id 507f1f77bcf86cd799439011 --endpoint-id 5e2211c17a3e5a48f5497de3 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.UnsupportedFeatureError("VPC endpoints", "Requires PrivateEndpointServicesApi in SDK")
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&endpointID, "endpoint-id", "", "VPC endpoint ID (required)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.MarkFlagRequired("endpoint-id")

	return cmd
}

func runListVPCEndpoints(cmd *cobra.Command, projectID string, paginationFlags *cli.PaginationFlags) error {
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
	progress.StartSpinner("Fetching VPC endpoints...")

	// Create Atlas client and service
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewVPCEndpointsService(client)

	// Fetch VPC endpoints
	endpoints, err := service.ListPrivateEndpoints(ctx, projectID)
	if err != nil {
		progress.StopSpinnerWithError("Failed to fetch VPC endpoints")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("VPC endpoints retrieved successfully")

	// Apply pagination if needed
	if paginationOpts.ShouldPaginate() && !paginationFlags.All {
		skip := paginationOpts.CalculateSkip()
		end := skip + paginationOpts.Limit

		if skip >= len(endpoints) {
			endpoints = []admin.PrivateLinkEndpoint{}
		} else {
			if end > len(endpoints) {
				end = len(endpoints)
			}
			endpoints = endpoints[skip:end]
		}
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, endpoints,
		[]string{"INTERFACE_ID", "STATUS", "PROVIDER", "CONNECTION_STATUS"},
		func(item interface{}) []string {
			endpoint := item.(admin.PrivateLinkEndpoint)
			id := getStringValue(endpoint.InterfaceEndpointId)
			status := getStringValue(endpoint.Status)
			provider := endpoint.CloudProvider
			connectionStatus := getStringValue(endpoint.ConnectionStatus)

			return []string{id, status, provider, connectionStatus}
		})
}

func runGetVPCEndpoint(cmd *cobra.Command, projectID, endpointID string) error {
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

	if endpointID == "" {
		return cli.FormatValidationError("endpoint-id", endpointID, "endpoint ID cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Fetching VPC endpoint '%s'...", endpointID))

	// Create Atlas client and service
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewVPCEndpointsService(client)

	// Fetch VPC endpoint
	endpoint, err := service.GetPrivateEndpoint(ctx, projectID, endpointID)
	if err != nil {
		progress.StopSpinnerWithError(fmt.Sprintf("Failed to fetch VPC endpoint '%s'", endpointID))
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("VPC endpoint '%s' retrieved successfully", endpointID))

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(endpoint)
}

func runCreateVPCEndpoint(cmd *cobra.Command, projectID, cloudProvider, region string) error {
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

	if cloudProvider == "" {
		return cli.FormatValidationError("cloud-provider", cloudProvider, "cloud provider cannot be empty")
	}

	if region == "" {
		return cli.FormatValidationError("region", region, "region cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner("Creating VPC endpoint...")

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewVPCEndpointsService(client)

	// Create VPC endpoint configuration
	endpoint := &admin.PrivateLinkEndpoint{
		CloudProvider: cloudProvider,
	}

	// Create the VPC endpoint
	createdEndpoint, err := service.CreatePrivateEndpoint(ctx, projectID, endpoint)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create VPC endpoint")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("")

	// Display created endpoint details with prettier formatting
	formatter := output.NewCreateResultFormatter(cfg.Output, os.Stdout)
	return formatter.FormatCreateResult(createdEndpoint, "vpc endpoint")
}

func runDeleteVPCEndpoint(cmd *cobra.Command, projectID, endpointID string, force bool) error {
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

	if endpointID == "" {
		return cli.FormatValidationError("endpoint-id", endpointID, "endpoint ID cannot be empty")
	}

	// Confirm deletion unless force flag is used
	if !force {
		prompt := ui.NewConfirmationPrompt(false, false)
		confirmed, err := prompt.ConfirmDeletion("VPC endpoint", endpointID)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("VPC endpoint deletion cancelled")
			return nil
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Deleting VPC endpoint '%s'...", endpointID))

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewVPCEndpointsService(client)

	// Delete the VPC endpoint
	err = service.DeletePrivateEndpoint(ctx, projectID, endpointID)
	if err != nil {
		progress.StopSpinnerWithError("Failed to delete VPC endpoint")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("VPC endpoint '%s' deleted successfully", endpointID))
	return nil
}

// Helper functions for formatting output
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func formatTimeValue(ptr *time.Time) string {
	if ptr == nil {
		return ""
	}
	return ptr.Format("2006-01-02 15:04:05")
}
