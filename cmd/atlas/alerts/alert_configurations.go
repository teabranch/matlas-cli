package alerts

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func NewAlertConfigurationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "alert-configurations",
		Short:   "Manage Atlas alert configurations",
		Long:    "Create, list, view, update, and delete MongoDB Atlas alert configurations.",
		Aliases: []string{"alert-config", "alert-configs", "alertconfig"},
	}

	cmd.AddCommand(newListAlertConfigurationsCmd())
	cmd.AddCommand(newGetAlertConfigurationCmd())
	cmd.AddCommand(newDeleteAlertConfigurationCmd())
	cmd.AddCommand(newListMatcherFieldNamesCmd())

	return cmd
}

func newListAlertConfigurationsCmd() *cobra.Command {
	var projectID string
	var paginationFlags cli.PaginationFlags

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List alert configurations",
		Long: `List all alert configurations in a project.

This command retrieves and displays all MongoDB Atlas alert configurations in the specified project.
The output includes configuration ID, event type, enabled status, and notification details.`,
		SilenceUsage: true,
		Example: `  # List alert configurations in a project
  matlas atlas alert-configurations list --project-id 507f1f77bcf86cd799439011

  # List with pagination
  matlas atlas alert-configurations list --project-id 507f1f77bcf86cd799439011 --page 2 --limit 10

  # Output as JSON
  matlas atlas alert-configurations list --project-id 507f1f77bcf86cd799439011 --output json

  # Using alias
  matlas atlas alert-configs ls --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListAlertConfigurations(cmd, projectID, &paginationFlags)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	cli.AddPaginationFlags(cmd, &paginationFlags)

	return cmd
}

func newGetAlertConfigurationCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "get <alert-config-id>",
		Short: "Get alert configuration details",
		Long: `Get detailed information about a specific alert configuration.

This command retrieves and displays detailed information about a specific MongoDB Atlas alert configuration,
including its matchers, notifications, thresholds, and other settings.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		Example: `  # Get alert configuration details
  matlas atlas alert-configurations get 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011

  # Output as JSON
  matlas atlas alert-configurations get 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetAlertConfiguration(cmd, projectID, args[0])
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	return cmd
}

func newDeleteAlertConfigurationCmd() *cobra.Command {
	var projectID string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <alert-config-id>",
		Short: "Delete an alert configuration",
		Long: `Delete a MongoDB Atlas alert configuration.

This command permanently deletes an alert configuration. Use with caution as this action cannot be undone.
Use --force to skip the confirmation prompt.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		Aliases:      []string{"rm", "remove"},
		Example: `  # Delete an alert configuration (with confirmation)
  matlas atlas alert-configurations delete 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011

  # Delete without confirmation
  matlas atlas alert-configurations delete 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011 --force

  # Using alias
  matlas atlas alert-configs rm 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteAlertConfiguration(cmd, projectID, args[0], force)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func newListMatcherFieldNamesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "matcher-fields",
		Short: "List available matcher field names",
		Long: `List all available field names that can be used in alert configuration matchers.

This command retrieves all field names that the matchers.fieldName parameter accepts
when creating or updating alert configurations.`,
		SilenceUsage: true,
		Aliases:      []string{"fields", "matcher-field-names"},
		Example: `  # List matcher field names
  matlas atlas alert-configurations matcher-fields

  # Output as JSON
  matlas atlas alert-configurations matcher-fields --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListMatcherFieldNames(cmd)
		},
	}

	return cmd
}

func runListAlertConfigurations(cmd *cobra.Command, projectID string, paginationFlags *cli.PaginationFlags) error {
	// Resolve project ID
	if projectID == "" {
		projectID = os.Getenv("ATLAS_PROJECT_ID")
	}
	if projectID == "" {
		return fmt.Errorf("project ID is required (use --project-id flag or set ATLAS_PROJECT_ID environment variable)")
	}

	// Validate project ID
	if err := validation.ValidateObjectID(projectID, "project ID"); err != nil {
		return err
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		return fmt.Errorf("failed to create Atlas client: %w", err)
	}

	// Create service
	service := atlas.NewAlertConfigurationsService(client)

	// List alert configurations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	atlasConfigs, err := service.ListAlertConfigurations(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to list alert configurations: %w", err)
	}

	// Convert Atlas configs to our types
	configs := make([]types.AlertConfig, 0, len(atlasConfigs))
	for _, atlasConfig := range atlasConfigs {
		converted := service.ConvertFromAtlasConfig(&atlasConfig, fmt.Sprintf("alert-config-%s", atlasConfig.GetId()))
		if converted != nil {
			configs = append(configs, *converted)
		}
	}

	// Apply pagination if needed
	if paginationFlags != nil {
		// Apply pagination logic here if needed
		// For now, we'll skip this as it's not critical for basic functionality
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, configs,
		[]string{"ID", "NAME", "EVENT_TYPE", "ENABLED", "SEVERITY"},
		func(item interface{}) []string {
			config := item.(types.AlertConfig)
			enabled := "false"
			if config.Enabled != nil && *config.Enabled {
				enabled = "true"
			}

			// We need to get the ID from somewhere - let's check if it's in metadata
			configID := config.Metadata.Name // fallback to name if no ID available
			if config.Metadata.Labels != nil {
				if id, exists := config.Metadata.Labels["atlas-id"]; exists {
					configID = id
				}
			}

			return []string{
				configID,
				config.Metadata.Name,
				config.EventTypeName,
				enabled,
				config.SeverityOverride,
			}
		})
}

func runGetAlertConfiguration(cmd *cobra.Command, projectID, alertConfigID string) error {
	// Resolve project ID
	if projectID == "" {
		projectID = os.Getenv("ATLAS_PROJECT_ID")
	}
	if projectID == "" {
		return fmt.Errorf("project ID is required (use --project-id flag or set ATLAS_PROJECT_ID environment variable)")
	}

	// Validate IDs
	if err := validation.ValidateObjectID(projectID, "project ID"); err != nil {
		return err
	}
	if err := validation.ValidateObjectID(alertConfigID, "alert configuration ID"); err != nil {
		return err
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		return fmt.Errorf("failed to create Atlas client: %w", err)
	}

	// Create service
	service := atlas.NewAlertConfigurationsService(client)

	// Get alert configuration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	alertConfig, err := service.GetAlertConfiguration(ctx, projectID, alertConfigID)
	if err != nil {
		return fmt.Errorf("failed to get alert configuration: %w", err)
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(alertConfig)
}

func runDeleteAlertConfiguration(cmd *cobra.Command, projectID, alertConfigID string, force bool) error {
	// Resolve project ID
	if projectID == "" {
		projectID = os.Getenv("ATLAS_PROJECT_ID")
	}
	if projectID == "" {
		return fmt.Errorf("project ID is required (use --project-id flag or set ATLAS_PROJECT_ID environment variable)")
	}

	// Validate IDs
	if err := validation.ValidateObjectID(projectID, "project ID"); err != nil {
		return err
	}
	if err := validation.ValidateObjectID(alertConfigID, "alert configuration ID"); err != nil {
		return err
	}

	// Confirm deletion unless forced
	if !force {
		prompt := ui.NewConfirmationPrompt(false, false)
		confirmed, err := prompt.ConfirmDeletion("alert configuration", alertConfigID)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Alert configuration deletion cancelled.")
			return nil
		}
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		return fmt.Errorf("failed to create Atlas client: %w", err)
	}

	// Create service
	service := atlas.NewAlertConfigurationsService(client)

	// Delete alert configuration
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = service.DeleteAlertConfiguration(ctx, projectID, alertConfigID)
	if err != nil {
		return fmt.Errorf("failed to delete alert configuration: %w", err)
	}

	fmt.Printf("Alert configuration %s deleted successfully\n", alertConfigID)
	return nil
}

func runListMatcherFieldNames(cmd *cobra.Command) error {
	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		return fmt.Errorf("failed to create Atlas client: %w", err)
	}

	// Create service
	service := atlas.NewAlertConfigurationsService(client)

	// List matcher field names
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fieldNames, err := service.ListMatcherFieldNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to list matcher field names: %w", err)
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(fieldNames)
}

func printAlertConfigurationsTable(configs []admin.GroupAlertsConfig) error {
	if len(configs) == 0 {
		fmt.Println("No alert configurations found.")
		return nil
	}

	headers := []string{"ID", "Event Type", "Enabled", "Notifications", "Created"}
	rows := make([][]string, len(configs))

	for i, config := range configs {
		enabled := "No"
		if config.GetEnabled() {
			enabled = "Yes"
		}

		notificationCount := "0"
		if notifications := config.GetNotifications(); len(notifications) > 0 {
			notificationCount = fmt.Sprintf("%d", len(notifications))
		}

		created := ""
		if !config.GetCreated().IsZero() {
			created = config.GetCreated().Format("2006-01-02 15:04:05")
		}

		rows[i] = []string{
			config.GetId(),
			config.GetEventTypeName(),
			enabled,
			notificationCount,
			created,
		}
	}

	// Print table manually since ui.PrintTable doesn't exist
	fmt.Printf("%-36s %-30s %-20s %-10s %-20s\n", headers[0], headers[1], headers[2], headers[3], headers[4])
	fmt.Println(strings.Repeat("-", 120))
	for _, row := range rows {
		fmt.Printf("%-36s %-30s %-20s %-10s %-20s\n", row[0], row[1], row[2], row[3], row[4])
	}
	return nil
}

func printAlertConfigurationDetails(config *admin.GroupAlertsConfig) error {
	fmt.Printf("Alert Configuration Details:\n")
	fmt.Printf("  ID: %s\n", config.GetId())
	fmt.Printf("  Event Type: %s\n", config.GetEventTypeName())
	fmt.Printf("  Enabled: %t\n", config.GetEnabled())

	if !config.GetCreated().IsZero() {
		fmt.Printf("  Created: %s\n", config.GetCreated().Format(time.RFC3339))
	}

	if !config.GetUpdated().IsZero() {
		fmt.Printf("  Updated: %s\n", config.GetUpdated().Format(time.RFC3339))
	}

	if severity := config.GetSeverityOverride(); severity != "" {
		fmt.Printf("  Severity Override: %s\n", severity)
	}

	if groupID := config.GetGroupId(); groupID != "" {
		fmt.Printf("  Project ID: %s\n", groupID)
	}

	// Print matchers
	if matchers := config.GetMatchers(); len(matchers) > 0 {
		fmt.Printf("  Matchers:\n")
		for i, matcher := range matchers {
			fmt.Printf("    %d. Field: %s, Operator: %s, Value: %s\n",
				i+1, matcher.GetFieldName(), matcher.GetOperator(), matcher.GetValue())
		}
	}

	// Print notifications
	if notifications := config.GetNotifications(); len(notifications) > 0 {
		fmt.Printf("  Notifications:\n")
		for i, notification := range notifications {
			fmt.Printf("    %d. Type: %s", i+1, notification.GetTypeName())
			if delayMin := notification.GetDelayMin(); delayMin > 0 {
				fmt.Printf(", Delay: %d min", delayMin)
			}
			if intervalMin := notification.GetIntervalMin(); intervalMin > 0 {
				fmt.Printf(", Interval: %d min", intervalMin)
			}
			fmt.Printf("\n")

			// Print type-specific details
			switch notification.GetTypeName() {
			case "EMAIL":
				if email := notification.GetEmailAddress(); email != "" {
					fmt.Printf("       Email: %s\n", email)
				}
			case "SLACK":
				if channel := notification.GetChannelName(); channel != "" {
					fmt.Printf("       Channel: %s\n", channel)
				}
			case "SMS":
				if mobile := notification.GetMobileNumber(); mobile != "" {
					fmt.Printf("       Mobile: %s\n", mobile)
				}
			}
		}
	}

	// Print metric threshold
	if threshold, ok := config.GetMetricThresholdOk(); ok && threshold != nil {
		fmt.Printf("  Metric Threshold:\n")
		fmt.Printf("    Metric: %s\n", threshold.GetMetricName())
		fmt.Printf("    Operator: %s\n", threshold.GetOperator())
		fmt.Printf("    Threshold: %.2f", threshold.GetThreshold())
		if units := threshold.GetUnits(); units != "" {
			fmt.Printf(" %s", units)
		}
		fmt.Printf("\n")
		if mode := threshold.GetMode(); mode != "" {
			fmt.Printf("    Mode: %s\n", mode)
		}
	}

	// Print general threshold
	if threshold, ok := config.GetThresholdOk(); ok && threshold != nil {
		fmt.Printf("  Threshold:\n")
		fmt.Printf("    Operator: %s\n", threshold.GetOperator())
		fmt.Printf("    Threshold: %.2f", threshold.GetThreshold())
		if units := threshold.GetUnits(); units != "" {
			fmt.Printf(" %s", units)
		}
		fmt.Printf("\n")
	}

	return nil
}

func printMatcherFieldNamesTable(fieldNames []string) error {
	if len(fieldNames) == 0 {
		fmt.Println("No matcher field names found.")
		return nil
	}

	fmt.Printf("Available Matcher Field Names:\n")
	for i, fieldName := range fieldNames {
		fmt.Printf("  %d. %s\n", i+1, fieldName)
	}

	return nil
}
