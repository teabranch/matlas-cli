package networkpeering

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

func NewNetworkPeeringCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network-peering",
		Short:   "Manage Atlas network peering",
		Long:    "Manage MongoDB Atlas network peering connections for VPC connectivity",
		Aliases: []string{"peering", "vpc-peering"},
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
		Short:   "List network peering connections",
		Long: `List all network peering connections in a project.

This command retrieves and displays all MongoDB Atlas network peering connections in the specified project.
The output includes peering ID, status, cloud provider, VPC information, and creation date.`,
		Example: `  # List network peering connections in a project
  matlas atlas network-peering list --project-id 507f1f77bcf86cd799439011

  # List with pagination
  matlas atlas network-peering list --project-id 507f1f77bcf86cd799439011 --page 2 --limit 10

  # List all connections (no pagination)
  matlas atlas network-peering list --project-id 507f1f77bcf86cd799439011 --all

  # Output as JSON
  matlas atlas network-peering list --project-id 507f1f77bcf86cd799439011 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListNetworkPeering(cmd, projectID, &paginationFlags)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	cli.AddPaginationFlags(cmd, &paginationFlags)

	return cmd
}

func newGetCmd() *cobra.Command {
	var projectID string
	var peerID string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get network peering connection details",
		Long: `Get detailed information about a specific network peering connection.

This command retrieves and displays detailed information about a MongoDB Atlas network peering connection,
including configuration, status, and connection details.`,
		Example: `  # Get network peering connection details
  matlas atlas network-peering get --project-id 507f1f77bcf86cd799439011 --peer-id 5e2211c17a3e5a48f5497de3

  # Output as YAML
  matlas atlas network-peering get --project-id 507f1f77bcf86cd799439011 --peer-id 5e2211c17a3e5a48f5497de3 --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetNetworkPeering(cmd, projectID, peerID)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&peerID, "peer-id", "", "Network peering connection ID (required)")
	cmd.MarkFlagRequired("peer-id")

	return cmd
}

func newCreateCmd() *cobra.Command {
	var projectID string
	var cloudProvider string
	var vpcID string
	var region string
	var cidrBlock string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a network peering connection",
		Long: `Create a new network peering connection for VPC connectivity.

This command creates a network peering connection using the Atlas API. After creation, it returns the created connection.
`,
		Example: `  # Create network peering for AWS
  matlas atlas network-peering create --project-id 507f1f77bcf86cd799439011 \
    --cloud-provider AWS --vpc-id vpc-123456 --region US_EAST_1 --cidr-block 10.0.0.0/16

  # Create network peering for Azure
  matlas atlas network-peering create --project-id 507f1f77bcf86cd799439011 \
    --cloud-provider AZURE --vpc-id /subscriptions/.../resourceGroups/.../providers/Microsoft.Network/virtualNetworks/myVNet \
    --region eastus --cidr-block 10.1.0.0/16`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateNetworkPeering(cmd, projectID, cloudProvider, vpcID, region, cidrBlock)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider (AWS, AZURE, GCP) (required)")
	cmd.Flags().StringVar(&vpcID, "vpc-id", "", "VPC/VNet ID to peer with (required)")
	cmd.Flags().StringVar(&region, "region", "", "Cloud provider region (required)")
	cmd.Flags().StringVar(&cidrBlock, "cidr-block", "", "CIDR block for the peering connection (required)")
	cmd.MarkFlagRequired("cloud-provider")
	cmd.MarkFlagRequired("vpc-id")
	cmd.MarkFlagRequired("region")
	cmd.MarkFlagRequired("cidr-block")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var projectID string
	var peerID string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a network peering connection",
		Long: `Delete a network peering connection.

This command deletes a MongoDB Atlas network peering connection. This action cannot be undone.
The connection must not be in use by any clusters.`,
		Example: `  # Delete network peering connection with confirmation
  matlas atlas network-peering delete --project-id 507f1f77bcf86cd799439011 --peer-id 5e2211c17a3e5a48f5497de3

  # Delete without confirmation prompt
  matlas atlas network-peering delete --project-id 507f1f77bcf86cd799439011 --peer-id 5e2211c17a3e5a48f5497de3 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteNetworkPeering(cmd, projectID, peerID, force)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&peerID, "peer-id", "", "Network peering connection ID (required)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	cmd.MarkFlagRequired("peer-id")

	return cmd
}

func runListNetworkPeering(cmd *cobra.Command, projectID string, paginationFlags *cli.PaginationFlags) error {
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
	progress.StartSpinner("Fetching network peering connections...")

	// Create Atlas client and service
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkPeeringService(client)

	// Fetch network peering connections
	connections, err := service.ListPeeringConnections(ctx, projectID)
	if err != nil {
		progress.StopSpinnerWithError("Failed to fetch network peering connections")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("Network peering connections retrieved successfully")

	// Apply pagination if needed
	if paginationOpts.ShouldPaginate() && !paginationFlags.All {
		skip := paginationOpts.CalculateSkip()
		end := skip + paginationOpts.Limit

		if skip >= len(connections) {
			connections = []admin.BaseNetworkPeeringConnectionSettings{}
		} else {
			if end > len(connections) {
				end = len(connections)
			}
			connections = connections[skip:end]
		}
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, connections,
		[]string{"ID", "STATUS", "PROVIDER", "VPC_ID", "REGION"},
		func(item interface{}) []string {
			connection := item.(admin.BaseNetworkPeeringConnectionSettings)
			id := getStringValue(connection.Id)
			// Align on StatusName consistently
			status := getStringValue(connection.StatusName)
			provider := getStringValue(connection.ProviderName)
			vpcID := getStringValue(connection.VpcId)
			region := getStringValue(connection.AccepterRegionName)

			return []string{id, status, provider, vpcID, region}
		})
}

func runGetNetworkPeering(cmd *cobra.Command, projectID, peerID string) error {
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

	if peerID == "" {
		return cli.FormatValidationError("peer-id", peerID, "peer ID cannot be empty")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Fetching network peering connection '%s'...", peerID))

	// Create Atlas client and service
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkPeeringService(client)

	// Fetch network peering connection
	connection, err := service.GetPeeringConnection(ctx, projectID, peerID)
	if err != nil {
		progress.StopSpinnerWithError(fmt.Sprintf("Failed to fetch network peering connection '%s'", peerID))
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Network peering connection '%s' retrieved successfully", peerID))

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(connection)
}

func runCreateNetworkPeering(cmd *cobra.Command, projectID, cloudProvider, vpcID, region, cidrBlock string) error {
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

	if vpcID == "" {
		return cli.FormatValidationError("vpc-id", vpcID, "VPC ID cannot be empty")
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
	progress.StartSpinner("Creating network peering connection...")

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkPeeringService(client)

	// Create network peering configuration
	connection := &admin.BaseNetworkPeeringConnectionSettings{
		ProviderName:        &cloudProvider,
		VpcId:               &vpcID,
		RouteTableCidrBlock: &cidrBlock,
		AccepterRegionName:  &region,
	}

	// Create the network peering connection
	createdConnection, err := service.CreatePeeringConnection(ctx, projectID, connection)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create network peering connection")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	// Optionally wait until AVAILABLE within the same timeout window
	waitErr := service.WaitForPeeringConnectionAvailable(ctx, projectID, getStringValue(createdConnection.Id))
	if waitErr != nil {
		// Non-fatal: continue to print created resource while surfacing info in verbose mode
		progress.StopSpinner("Network peering connection created (not yet AVAILABLE)")
	} else {
		progress.StopSpinner("Network peering connection created and AVAILABLE")
	}

	// Display created connection details with prettier formatting
	formatter := output.NewCreateResultFormatter(cfg.Output, os.Stdout)
	return formatter.FormatCreateResult(createdConnection, "network peering")
}

func runDeleteNetworkPeering(cmd *cobra.Command, projectID, peerID string, force bool) error {
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

	if peerID == "" {
		return cli.FormatValidationError("peer-id", peerID, "peer ID cannot be empty")
	}

	// Confirm deletion unless force flag is used
	if !force {
		prompt := ui.NewConfirmationPrompt(false, false)
		confirmed, err := prompt.ConfirmDeletion("network peering connection", peerID)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Network peering connection deletion cancelled")
			return nil
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Deleting network peering connection '%s'...", peerID))

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewNetworkPeeringService(client)

	// Delete the network peering connection
	err = service.DeletePeeringConnection(ctx, projectID, peerID)
	if err != nil {
		progress.StopSpinnerWithError("Failed to delete network peering connection")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Network peering connection '%s' deleted successfully", peerID))
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
