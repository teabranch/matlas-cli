package networkcontainers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	admin "go.mongodb.org/atlas-sdk/v20250312010/admin"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func NewNetworkContainersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network-containers",
		Short:   "Manage Atlas network containers",
		Long:    "Manage MongoDB Atlas network containers for VPC peering setup",
		Aliases: []string{"containers", "network-container"},
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var projectID string
	var cloudProvider string
	var paginationFlags cli.PaginationFlags

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List network containers",
		Long: `List all network containers in a project.

This command retrieves and displays all MongoDB Atlas network containers in the specified project.
Network containers define the CIDR blocks for Atlas clusters and are required for VPC peering.`,
		Example: `  # List all network containers in a project
  matlas atlas network-containers list --project-id 507f1f77bcf86cd799439011

  # List containers for specific cloud provider
  matlas atlas network-containers list --project-id 507f1f77bcf86cd799439011 --cloud-provider AWS

  # List with pagination
  matlas atlas network-containers list --project-id 507f1f77bcf86cd799439011 --page 2 --limit 10

  # Output as JSON
  matlas atlas network-containers list --project-id 507f1f77bcf86cd799439011 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListNetworkContainers(cmd, projectID, cloudProvider, &paginationFlags)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Filter by cloud provider (AWS, AZURE, GCP)")

	cli.AddPaginationFlags(cmd, &paginationFlags)

	return cmd
}

func newGetCmd() *cobra.Command {
	var projectID string
	var containerID string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get network container details",
		Long: `Get detailed information about a specific network container.

This command retrieves and displays detailed information about a MongoDB Atlas network container,
including CIDR block, cloud provider, region, and status.`,
		Example: `  # Get network container details
  matlas atlas network-containers get --project-id 507f1f77bcf86cd799439011 --container-id 507f1f77bcf86cd799439012

  # Output as YAML
  matlas atlas network-containers get --project-id 507f1f77bcf86cd799439011 --container-id 507f1f77bcf86cd799439012 --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetNetworkContainer(cmd, projectID, containerID)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&containerID, "container-id", "", "Network container ID (required)")
	mustMarkFlagRequired(cmd, "container-id")

	return cmd
}

func newCreateCmd() *cobra.Command {
	var projectID string
	var cloudProvider string
	var region string
	var cidrBlock string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a network container",
		Long: `Create a new network container for VPC peering.

This command creates a new MongoDB Atlas network container that defines the CIDR block
for Atlas clusters in a specific region, enabling VPC peering connectivity.`,
		Example: `  # Create network container for AWS
  matlas atlas network-containers create --project-id 507f1f77bcf86cd799439011 \
    --cloud-provider AWS --region US_EAST_1 --cidr-block 10.8.0.0/21

  # Create network container for Azure
  matlas atlas network-containers create --project-id 507f1f77bcf86cd799439011 \
    --cloud-provider AZURE --region eastus --cidr-block 10.8.0.0/21`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateNetworkContainer(cmd, projectID, cloudProvider, region, cidrBlock)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider (AWS, AZURE, GCP) (required)")
	cmd.Flags().StringVar(&region, "region", "", "Cloud provider region (required)")
	cmd.Flags().StringVar(&cidrBlock, "cidr-block", "", "CIDR block for the network container (required)")
	mustMarkFlagRequired(cmd, "cloud-provider")
	mustMarkFlagRequired(cmd, "region")
	mustMarkFlagRequired(cmd, "cidr-block")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var projectID string
	var containerID string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a network container",
		Long: `Delete a network container.

This command deletes a MongoDB Atlas network container. This action cannot be undone.
The container must not be in use by any clusters or peering connections.`,
		Example: `  # Delete network container with confirmation
  matlas atlas network-containers delete --project-id 507f1f77bcf86cd799439011 --container-id 507f1f77bcf86cd799439012

  # Delete without confirmation prompt
  matlas atlas network-containers delete --project-id 507f1f77bcf86cd799439011 --container-id 507f1f77bcf86cd799439012 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteNetworkContainer(cmd, projectID, containerID, force)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&containerID, "container-id", "", "Network container ID (required)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	mustMarkFlagRequired(cmd, "container-id")

	return cmd
}

func runListNetworkContainers(cmd *cobra.Command, projectID, cloudProvider string, paginationFlags *cli.PaginationFlags) error {
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
	_, err = paginationFlags.Validate()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner("Fetching network containers...")

	// Create Atlas client and service
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkContainersService(client)

	// Fetch network containers
	containers, err := service.ListNetworkContainers(ctx, projectID)
	if err != nil {
		progress.StopSpinnerWithError("Failed to fetch network containers")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("Network containers retrieved successfully")

	// Filter by cloud provider if specified
	if cloudProvider != "" {
		var filteredContainers []admin.CloudProviderContainer
		for _, container := range containers {
			if container.ProviderName != nil && *container.ProviderName == cloudProvider {
				filteredContainers = append(filteredContainers, container)
			}
		}
		containers = filteredContainers
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, containers,
		[]string{"ID", "PROVIDER", "REGION", "CIDR_BLOCK", "STATUS"},
		func(item interface{}) []string {
			container := item.(admin.CloudProviderContainer)
			id := getStringValue(container.Id)
			provider := getStringValue(container.ProviderName)
			region := getStringValue(container.RegionName)
			cidr := getStringValue(container.AtlasCidrBlock)
			status := formatBoolValue(container.Provisioned)

			return []string{id, provider, region, cidr, status}
		})
}

func runGetNetworkContainer(cmd *cobra.Command, projectID, containerID string) error {
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

	if containerID == "" {
		return cli.FormatValidationError("container-id", containerID, "container ID cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Fetching network container '%s'...", containerID))

	// Create Atlas client and service
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkContainersService(client)

	// Fetch network container
	container, err := service.GetNetworkContainer(ctx, projectID, containerID)
	if err != nil {
		progress.StopSpinnerWithError(fmt.Sprintf("Failed to fetch network container '%s'", containerID))
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Network container '%s' retrieved successfully", containerID))

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(container)
}

func runCreateNetworkContainer(cmd *cobra.Command, projectID, cloudProvider, region, cidrBlock string) error {
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

	if cidrBlock == "" {
		return cli.FormatValidationError("cidr-block", cidrBlock, "CIDR block cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner("Creating network container...")

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkContainersService(client)

	// Best-effort overlap validation to avoid conflicting CIDRs
	if err := service.ValidateNoOverlappingCIDRs(ctx, projectID, cidrBlock); err != nil {
		progress.StopSpinnerWithError("CIDR overlaps with existing network containers")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	// Create network container configuration
	container := &admin.CloudProviderContainer{
		ProviderName:   &cloudProvider,
		RegionName:     &region,
		AtlasCidrBlock: &cidrBlock,
	}

	// Create the network container
	createdContainer, err := service.CreateNetworkContainer(ctx, projectID, container)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create network container")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("")

	// Display created container details with prettier formatting
	formatter := output.NewCreateResultFormatter(cfg.Output, os.Stdout)
	return formatter.FormatCreateResult(createdContainer, "network container")
}

func runDeleteNetworkContainer(cmd *cobra.Command, projectID, containerID string, force bool) error {
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

	if containerID == "" {
		return cli.FormatValidationError("container-id", containerID, "container ID cannot be empty")
	}

	// Confirm deletion unless force flag is used
	if !force {
		prompt := ui.NewConfirmationPrompt(false, false)
		confirmed, err := prompt.ConfirmDeletion("network container", containerID)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Network container deletion cancelled")
			return nil
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Deleting network container '%s'...", containerID))

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkContainersService(client)

	// Delete the network container
	err = service.DeleteNetworkContainer(ctx, projectID, containerID)
	if err != nil {
		progress.StopSpinnerWithError("Failed to delete network container")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Network container '%s' deleted successfully", containerID))
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

func formatBoolValue(ptr *bool) string {
	if ptr == nil {
		return "unknown"
	}
	if *ptr {
		return "provisioned"
	}
	return "provisioning"
}

// mustMarkFlagRequired marks a flag as required and panics if it fails.
// This should never fail in normal execution and indicates a programmer error if it does.
func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Errorf("failed to mark flag %q required: %w", name, err))
	}
}
