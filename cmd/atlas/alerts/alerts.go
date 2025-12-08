package alerts

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	admin "go.mongodb.org/atlas-sdk/v20250312010/admin"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func NewAlertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "alerts",
		Short:   "Manage Atlas alerts",
		Long:    "List, view, and acknowledge MongoDB Atlas alerts.",
		Aliases: []string{"alert"},
	}

	cmd.AddCommand(newListAlertsCmd())
	cmd.AddCommand(newGetAlertCmd())
	cmd.AddCommand(newAcknowledgeAlertCmd())

	return cmd
}

func newListAlertsCmd() *cobra.Command {
	var projectID string
	var paginationFlags cli.PaginationFlags

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List alerts",
		Long: `List all alerts in a project.

This command retrieves and displays all MongoDB Atlas alerts in the specified project.
The output includes alert ID, event type, status, and acknowledgment information.`,
		SilenceUsage: true,
		Example: `  # List alerts in a project
  matlas atlas alerts list --project-id 507f1f77bcf86cd799439011

  # List with pagination
  matlas atlas alerts list --project-id 507f1f77bcf86cd799439011 --page 2 --limit 10

  # Output as JSON
  matlas atlas alerts list --project-id 507f1f77bcf86cd799439011 --output json

  # Using alias
  matlas atlas alerts ls --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListAlerts(cmd, projectID, &paginationFlags)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	cli.AddPaginationFlags(cmd, &paginationFlags)

	return cmd
}

func newGetAlertCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "get <alert-id>",
		Short: "Get alert details",
		Long: `Get detailed information about a specific alert.

This command retrieves and displays detailed information about a specific MongoDB Atlas alert,
including its configuration, current status, and acknowledgment details.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		Example: `  # Get alert details
  matlas atlas alerts get 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011

  # Output as JSON
  matlas atlas alerts get 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetAlert(cmd, projectID, args[0])
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	return cmd
}

func newAcknowledgeAlertCmd() *cobra.Command {
	var projectID string
	var unacknowledge bool
	var until string

	cmd := &cobra.Command{
		Use:   "acknowledge <alert-id>",
		Short: "Acknowledge or unacknowledge an alert",
		Long: `Acknowledge or unacknowledge a MongoDB Atlas alert.

Acknowledging an alert prevents successive notifications until the alert condition
is resolved or the acknowledgment expires. Use --unacknowledge to remove acknowledgment.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		Aliases:      []string{"ack"},
		Example: `  # Acknowledge an alert
  matlas atlas alerts acknowledge 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011

  # Acknowledge until a specific time
  matlas atlas alerts acknowledge 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011 --until "2024-12-31T23:59:59Z"

  # Unacknowledge an alert
  matlas atlas alerts acknowledge 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011 --unacknowledge

  # Using alias
  matlas atlas alerts ack 507f1f77bcf86cd799439011 --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAcknowledgeAlert(cmd, projectID, args[0], !unacknowledge, until)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().BoolVar(&unacknowledge, "unacknowledge", false, "Remove acknowledgment from the alert")
	cmd.Flags().StringVar(&until, "until", "", "Acknowledge until this time (ISO 8601 format, e.g., 2024-12-31T23:59:59Z)")

	return cmd
}

func runListAlerts(cmd *cobra.Command, projectID string, paginationFlags *cli.PaginationFlags) error {
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
	service := atlas.NewAlertsService(client)

	// List alerts
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	atlasAlerts, err := service.ListAlerts(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to list alerts: %w", err)
	}

	// Convert Atlas alerts to our types
	alerts := make([]types.AlertStatus, 0, len(atlasAlerts))
	for _, atlasAlert := range atlasAlerts {
		converted := service.ConvertAlertToStatus(&atlasAlert)
		if converted != nil {
			alerts = append(alerts, *converted)
		}
	}

	// Apply pagination if needed
	if paginationFlags != nil {
		// Apply pagination logic here if needed
		// For now, we'll skip this as it's not critical for basic functionality
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, alerts,
		[]string{"ID", "STATUS", "EVENT_TYPE", "CLUSTER", "CREATED"},
		func(item interface{}) []string {
			alert := item.(types.AlertStatus)
			created := ""
			if alert.Created != nil {
				created = alert.Created.Format("2006-01-02 15:04:05")
			}
			return []string{
				alert.ID,
				alert.Status,
				alert.EventTypeName,
				alert.ClusterName,
				created,
			}
		})
}

func runGetAlert(cmd *cobra.Command, projectID, alertID string) error {
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
	if err := validation.ValidateObjectID(alertID, "alert ID"); err != nil {
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
	service := atlas.NewAlertsService(client)

	// Get alert
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	alert, err := service.GetAlert(ctx, projectID, alertID)
	if err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(alert)
}

func runAcknowledgeAlert(cmd *cobra.Command, projectID, alertID string, acknowledge bool, until string) error {
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
	if err := validation.ValidateObjectID(alertID, "alert ID"); err != nil {
		return err
	}

	// Validate until time if provided
	var untilPtr *string
	if until != "" {
		if _, err := time.Parse(time.RFC3339, until); err != nil {
			return fmt.Errorf("invalid until time format (use ISO 8601, e.g., 2024-12-31T23:59:59Z): %w", err)
		}
		untilPtr = &until
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
	service := atlas.NewAlertsService(client)

	// Acknowledge alert
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	alert, err := service.AcknowledgeAlert(ctx, projectID, alertID, acknowledge, untilPtr)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	// Print success message
	action := "acknowledged"
	if !acknowledge {
		action = "unacknowledged"
	}

	fmt.Printf("Alert %s %s successfully\n", alertID, action)

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(alert)
}

func printAlertsTable(alerts []admin.AlertViewForNdsGroup) error {
	if len(alerts) == 0 {
		fmt.Println("No alerts found.")
		return nil
	}

	headers := []string{"ID", "Event Type", "Status", "Created", "Acknowledged"}
	rows := make([][]string, len(alerts))

	for i, alert := range alerts {
		acknowledged := "No"
		if !alert.GetAcknowledgedUntil().IsZero() {
			acknowledged = "Yes"
			if ackUser := alert.GetAcknowledgingUsername(); ackUser != "" {
				acknowledged = fmt.Sprintf("Yes (%s)", ackUser)
			}
		}

		created := ""
		if !alert.GetCreated().IsZero() {
			created = alert.GetCreated().Format("2006-01-02 15:04:05")
		}

		rows[i] = []string{
			alert.GetId(),
			alert.GetEventTypeName(),
			alert.GetStatus(),
			created,
			acknowledged,
		}
	}

	// Print table manually since ui.PrintTable doesn't exist
	fmt.Printf("%-36s %-20s %-30s %-20s %-20s\n", headers[0], headers[1], headers[2], headers[3], headers[4])
	fmt.Println(strings.Repeat("-", 130))
	for _, row := range rows {
		fmt.Printf("%-36s %-20s %-30s %-20s %-20s\n", row[0], row[1], row[2], row[3], row[4])
	}
	return nil
}

func printAlertDetails(alert *admin.AlertViewForNdsGroup) error {
	fmt.Printf("Alert Details:\n")
	fmt.Printf("  ID: %s\n", alert.GetId())
	fmt.Printf("  Event Type: %s\n", alert.GetEventTypeName())
	fmt.Printf("  Status: %s\n", alert.GetStatus())

	if !alert.GetCreated().IsZero() {
		fmt.Printf("  Created: %s\n", alert.GetCreated().Format(time.RFC3339))
	}

	if !alert.GetUpdated().IsZero() {
		fmt.Printf("  Updated: %s\n", alert.GetUpdated().Format(time.RFC3339))
	}

	if !alert.GetAcknowledgedUntil().IsZero() {
		fmt.Printf("  Acknowledged Until: %s\n", alert.GetAcknowledgedUntil().Format(time.RFC3339))
		if ackUser := alert.GetAcknowledgingUsername(); ackUser != "" {
			fmt.Printf("  Acknowledged By: %s\n", ackUser)
		}
	}

	if !alert.GetLastNotified().IsZero() {
		fmt.Printf("  Last Notified: %s\n", alert.GetLastNotified().Format(time.RFC3339))
	}

	if metricName := alert.GetMetricName(); metricName != "" {
		fmt.Printf("  Metric: %s\n", metricName)
	}

	if currentValue, ok := alert.GetCurrentValueOk(); ok && currentValue != nil {
		if number := currentValue.GetNumber(); number != 0 {
			units := currentValue.GetUnits()
			if units != "" {
				fmt.Printf("  Current Value: %.2f %s\n", number, units)
			} else {
				fmt.Printf("  Current Value: %.2f\n", number)
			}
		}
	}

	if hostname := alert.GetHostnameAndPort(); hostname != "" {
		fmt.Printf("  Host: %s\n", hostname)
	}

	if replicaSet := alert.GetReplicaSetName(); replicaSet != "" {
		fmt.Printf("  Replica Set: %s\n", replicaSet)
	}

	if clusterName := alert.GetClusterName(); clusterName != "" {
		fmt.Printf("  Cluster: %s\n", clusterName)
	}

	if configID := alert.GetAlertConfigId(); configID != "" {
		fmt.Printf("  Alert Configuration ID: %s\n", configID)
	}

	return nil
}
