package vpcendpoints

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
	"os"
)

func NewVPCEndpointsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpc-endpoints",
		Short: "Manage Atlas VPC endpoints and Private Link connections",
		Long: `Atlas VPC endpoints and Private Link connections for secure connectivity to Atlas clusters.

VPC endpoints allow you to create private network connections to Atlas clusters, providing enhanced
security by avoiding traffic over the public internet. This feature supports AWS, Azure, and GCP.

You can create, list, get, update, and delete VPC endpoint services through the CLI or YAML configuration.`,
		Aliases: []string{"vpc-endpoint", "vpc"},
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var projectID, cloudProvider string
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List VPC endpoint services",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListVPCEndpoints(cmd, projectID, cloudProvider)
		},
	}
	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Optional cloud provider (AWS, AZURE, GCP)")
	return cmd
}

func newGetCmd() *cobra.Command {
	var projectID, cloudProvider, endpointID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get VPC endpoint service details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetVPCEndpoint(cmd, projectID, cloudProvider, endpointID)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider (AWS, AZURE, GCP)")
	cmd.Flags().StringVar(&endpointID, "endpoint-id", "", "VPC endpoint service ID")
	if err := cmd.MarkFlagRequired("project-id"); err != nil {
		panic(fmt.Errorf("failed to mark project-id flag as required: %w", err))
	}
	if err := cmd.MarkFlagRequired("cloud-provider"); err != nil {
		panic(fmt.Errorf("failed to mark cloud-provider flag as required: %w", err))
	}
	if err := cmd.MarkFlagRequired("endpoint-id"); err != nil {
		panic(fmt.Errorf("failed to mark endpoint-id flag as required: %w", err))
	}
	return cmd
}

func newCreateCmd() *cobra.Command {
	var projectID, cloudProvider, region string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VPC endpoint service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateVPCEndpoint(cmd, projectID, cloudProvider, region)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider (AWS, AZURE, GCP)")
	cmd.Flags().StringVar(&region, "region", "", "Cloud provider region")
	if err := cmd.MarkFlagRequired("project-id"); err != nil {
		panic(fmt.Errorf("failed to mark project-id flag as required: %w", err))
	}
	if err := cmd.MarkFlagRequired("cloud-provider"); err != nil {
		panic(fmt.Errorf("failed to mark cloud-provider flag as required: %w", err))
	}
	if err := cmd.MarkFlagRequired("region"); err != nil {
		panic(fmt.Errorf("failed to mark region flag as required: %w", err))
	}
	return cmd
}

func newUpdateCmd() *cobra.Command {
	var projectID, cloudProvider, endpointID string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a VPC endpoint service",
		Long:  "Update a VPC endpoint service. Note: Most VPC endpoint properties are immutable after creation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdateVPCEndpoint(cmd, projectID, cloudProvider, endpointID)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider (AWS, AZURE, GCP)")
	cmd.Flags().StringVar(&endpointID, "endpoint-id", "", "VPC endpoint service ID")
	if err := cmd.MarkFlagRequired("project-id"); err != nil {
		panic(fmt.Errorf("failed to mark project-id flag as required: %w", err))
	}
	if err := cmd.MarkFlagRequired("cloud-provider"); err != nil {
		panic(fmt.Errorf("failed to mark cloud-provider flag as required: %w", err))
	}
	if err := cmd.MarkFlagRequired("endpoint-id"); err != nil {
		panic(fmt.Errorf("failed to mark endpoint-id flag as required: %w", err))
	}
	return cmd
}

func newDeleteCmd() *cobra.Command {
	var projectID, cloudProvider, endpointID string
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a VPC endpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteVPCEndpoint(cmd, projectID, cloudProvider, endpointID, yes)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider (AWS, AZURE, GCP)")
	cmd.Flags().StringVar(&endpointID, "endpoint-id", "", "VPC endpoint service ID")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	if err := cmd.MarkFlagRequired("project-id"); err != nil {
		panic(fmt.Errorf("failed to mark project-id flag as required: %w", err))
	}
	if err := cmd.MarkFlagRequired("cloud-provider"); err != nil {
		panic(fmt.Errorf("failed to mark cloud-provider flag as required: %w", err))
	}
	if err := cmd.MarkFlagRequired("endpoint-id"); err != nil {
		panic(fmt.Errorf("failed to mark endpoint-id flag as required: %w", err))
	}
	return cmd
}

// runListVPCEndpoints lists VPC endpoint services for a project and optional provider
func runListVPCEndpoints(cmd *cobra.Command, projectID, cloudProvider string) error {
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	projectID = cfg.ResolveProjectID(projectID)
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner("Fetching VPC endpoint services...")
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}
	service := atlas.NewVPCEndpointsService(client)
	var entries []admin.EndpointService
	if cloudProvider == "" {
		all, err := service.ListAllPrivateEndpointServices(ctx, projectID)
		if err != nil {
			progress.StopSpinnerWithError("Failed to list VPC endpoint services")
			return fmt.Errorf("%w", err)
		}
		for _, list := range all {
			entries = append(entries, list...)
		}
	} else {
		list, err := service.ListPrivateEndpointServices(ctx, projectID, cloudProvider)
		if err != nil {
			progress.StopSpinnerWithError("Failed to list VPC endpoint services")
			return fmt.Errorf("%w", err)
		}
		entries = list
	}
	progress.StopSpinner("VPC endpoint services retrieved successfully")
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return output.FormatList(formatter, entries,
		[]string{"ID", "SERVICE_NAME", "PROVIDER", "REGION", "STATUS"},
		func(item interface{}) []string {
			svc := item.(admin.EndpointService)
			return []string{
				svc.GetId(),
				svc.GetEndpointServiceName(),
				svc.GetCloudProvider(),
				svc.GetRegionName(),
				svc.GetStatus(),
			}
		})
}

// runGetVPCEndpoint retrieves details for a specific VPC endpoint service
func runGetVPCEndpoint(cmd *cobra.Command, projectID, cloudProvider, endpointID string) error {
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	projectID = cfg.ResolveProjectID(projectID)
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}
	if cloudProvider == "" {
		return fmt.Errorf("cloud-provider is required")
	}
	if endpointID == "" {
		return fmt.Errorf("endpoint-id is required")
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Fetching VPC endpoint service '%s'...", endpointID))
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}
	service := atlas.NewVPCEndpointsService(client)
	svc, err := service.GetPrivateEndpointService(ctx, projectID, cloudProvider, endpointID)
	if err != nil {
		progress.StopSpinnerWithError(fmt.Sprintf("Failed to fetch VPC endpoint service '%s'", endpointID))
		return fmt.Errorf("%w", err)
	}
	progress.StopSpinner(fmt.Sprintf("VPC endpoint service '%s' retrieved successfully", endpointID))
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(svc)
}

// runCreateVPCEndpoint creates a new VPC endpoint service
func runCreateVPCEndpoint(cmd *cobra.Command, projectID, cloudProvider, region string) error {
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	projectID = cfg.ResolveProjectID(projectID)
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}
	if cloudProvider == "" {
		return fmt.Errorf("cloud-provider is required")
	}
	if region == "" {
		return fmt.Errorf("region is required")
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Creating VPC endpoint service in region '%s'...", region))
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}
	service := atlas.NewVPCEndpointsService(client)
	request := admin.CloudProviderEndpointServiceRequest{ProviderName: cloudProvider, Region: region}
	svc, err := service.CreatePrivateEndpointService(ctx, projectID, cloudProvider, request)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create VPC endpoint service")
		return fmt.Errorf("%w", err)
	}
	progress.StopSpinner(fmt.Sprintf("VPC endpoint service created with ID '%s'", svc.GetId()))
	formatter := output.NewCreateResultFormatter(cfg.Output, os.Stdout)
	return formatter.FormatCreateResult(svc, "VPC endpoint service")
}

// runDeleteVPCEndpoint deletes a VPC endpoint service
func runDeleteVPCEndpoint(cmd *cobra.Command, projectID, cloudProvider, endpointID string, yes bool) error {
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	projectID = cfg.ResolveProjectID(projectID)
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}
	if cloudProvider == "" {
		return fmt.Errorf("cloud-provider is required")
	}
	if endpointID == "" {
		return fmt.Errorf("endpoint-id is required")
	}
	if !yes {
		confirm := ui.NewConfirmationPrompt(false, false)
		confirmed, err := confirm.Confirm(fmt.Sprintf("Are you sure you want to delete VPC endpoint service '%s'? This action cannot be undone.", endpointID))
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Deleting VPC endpoint service '%s'...", endpointID))
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}
	service := atlas.NewVPCEndpointsService(client)
	if err := service.DeletePrivateEndpointService(ctx, projectID, cloudProvider, endpointID); err != nil {
		progress.StopSpinnerWithError(fmt.Sprintf("Failed to delete VPC endpoint service '%s'", endpointID))
		return fmt.Errorf("%w", err)
	}
	progress.StopSpinner(fmt.Sprintf("VPC endpoint service '%s' deleted successfully", endpointID))
	return nil
}

// runUpdateVPCEndpoint updates a VPC endpoint service
func runUpdateVPCEndpoint(cmd *cobra.Command, projectID, cloudProvider, endpointID string) error {
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	projectID = cfg.ResolveProjectID(projectID)
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}
	if cloudProvider == "" {
		return fmt.Errorf("cloud-provider is required")
	}
	if endpointID == "" {
		return fmt.Errorf("endpoint-id is required")
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Updating VPC endpoint service '%s'...", endpointID))
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}
	service := atlas.NewVPCEndpointsService(client)

	// For VPC endpoints, most properties are immutable after creation
	// We can attempt to update the endpoint service, but it may be a no-op
	updated, err := service.UpdatePrivateEndpointService(ctx, projectID, cloudProvider, endpointID)
	if err != nil {
		progress.StopSpinnerWithError(fmt.Sprintf("Failed to update VPC endpoint service '%s'", endpointID))
		return fmt.Errorf("%w", err)
	}
	progress.StopSpinner(fmt.Sprintf("VPC endpoint service '%s' updated successfully", endpointID))
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(updated)
}
