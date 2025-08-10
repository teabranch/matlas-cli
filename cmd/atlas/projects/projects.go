package projects

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
	"go.uber.org/zap"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	atlasservice "github.com/teabranch/matlas-cli/internal/services/atlas"
)

func NewProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Manage Atlas projects",
		Long:  "List, get, and manage MongoDB Atlas projects",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newUpdateCmd())

	return cmd
}
func newUpdateCmd() *cobra.Command {
	var projectID string
	var newName string
	var tagKVs []string
	var clearTags bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a project",
		Long:  "Update an existing MongoDB Atlas project (name and tags).",
		Example: `  # Rename a project
  matlas atlas projects update --project-id 507f1f77bcf86cd799439011 --name "New Name"

  # Add or update tags
  matlas atlas projects update --project-id 507f1f77bcf86cd799439011 --tag env=dev --tag owner=platform

  # Clear all tags
  matlas atlas projects update --project-id 507f1f77bcf86cd799439011 --clear-tags`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config to resolve project ID
			cfg, err := config.Load(cmd, "")
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}
			projectID = cfg.ResolveProjectID(projectID)
			if projectID == "" {
				return fmt.Errorf("project-id is required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			client, err := createAtlasClient()
			if err != nil {
				return fmt.Errorf("failed to create Atlas client: %w", err)
			}
			service := atlasservice.NewProjectsService(client)

			// Build update payload
			upd := admin.GroupUpdate{}
			if newName != "" {
				upd.Name = &newName
			}

			if clearTags {
				empty := []admin.ResourceTag{}
				upd.Tags = &empty
			} else if len(tagKVs) > 0 {
				tags := make([]admin.ResourceTag, 0, len(tagKVs))
				for _, kv := range tagKVs {
					parts := strings.SplitN(kv, "=", 2)
					if len(parts) != 2 || parts[0] == "" {
						return fmt.Errorf("invalid --tag value: %s (expected key=value)", kv)
					}
					tags = append(tags, admin.ResourceTag{Key: parts[0], Value: parts[1]})
				}
				upd.Tags = &tags
			}

			if upd.Name == nil && upd.Tags == nil {
				return fmt.Errorf("nothing to update: specify --name, --tag, or --clear-tags")
			}

			updated, err := service.Update(ctx, projectID, upd)
			if err != nil {
				return fmt.Errorf("failed to update project: %w", err)
			}

			formatter := output.NewFormatter(config.OutputTable, os.Stdout)
			return formatter.Format(updated)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&newName, "name", "", "New project name")
	cmd.Flags().StringArrayVar(&tagKVs, "tag", nil, "Project tag in key=value form (repeatable)")
	cmd.Flags().BoolVar(&clearTags, "clear-tags", false, "Remove all tags from the project")

	return cmd
}

func newListCmd() *cobra.Command {
	var orgID string
	var showTags bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Long:  "List all projects visible to the authenticated account or within a specific organization",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration to get output format
			cfg, err := config.Load(cmd, "")
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Create client and service
			client, err := cfg.CreateAtlasClient()
			if err != nil {
				return err
			}

			service := atlasservice.NewProjectsService(client)

			// List projects
			var projects []interface{}
			if orgID != "" {
				orgProjects, err := service.ListByOrg(ctx, orgID)
				if err != nil {
					return fmt.Errorf("failed to list projects by organization: %w", err)
				}
				for _, p := range orgProjects {
					projects = append(projects, p)
				}
			} else {
				allProjects, err := service.List(ctx)
				if err != nil {
					return fmt.Errorf("failed to list projects: %w", err)
				}
				for _, p := range allProjects {
					projects = append(projects, p)
				}
			}

			// Format and output results using proper formatter
			formatter := output.NewFormatter(cfg.Output, os.Stdout)

			// For table format, use FormatList with proper headers and row function
			if cfg.Output == config.OutputTable || cfg.Output == config.OutputText || cfg.Output == "" {
				headers := []string{"ID", "NAME", "ORG_ID", "CREATED"}
				return output.FormatList(formatter, projects, headers, func(item interface{}) []string {
					// Type assertion to admin.Group from Atlas SDK
					if project, ok := item.(admin.Group); ok {
						id := ""
						if project.Id != nil {
							id = *project.Id
						}
						name := ""
						if project.Name != "" {
							name = project.Name
						}
						orgID := ""
						if project.OrgId != "" {
							orgID = project.OrgId
						}
						created := ""
						if !project.Created.IsZero() {
							created = project.Created.Format("2006-01-02")
						}

						// Optionally include tags as comma-separated k=v
						if showTags && project.Tags != nil {
							pairs := make([]string, 0, len(*project.Tags))
							for _, t := range *project.Tags {
								pairs = append(pairs, fmt.Sprintf("%s=%s", t.Key, t.Value))
							}
							return []string{id, name, orgID, created + "  [" + strings.Join(pairs, ",") + "]"}
						}
						return []string{id, name, orgID, created}
					}
					// Fallback for unknown project type
					return []string{"N/A", "N/A", "N/A", "N/A"}
				})
			}

			// For JSON/YAML, use direct formatting
			return formatter.Format(projects)
		},
	}

	cmd.Flags().StringVar(&orgID, "org-id", "", "Organization ID to filter projects")
	cmd.Flags().String("output", "", "Output format (table, json, yaml)")
	cmd.Flags().BoolVar(&showTags, "show-tags", false, "Include tags column in table output")

	return cmd
}

func newGetCmd() *cobra.Command {
	var projectID string
	var showTags bool

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a specific project",
		Long:  "Get details for a specific project by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get configuration first to resolve project ID if not provided
			cfg, err := config.Load(cmd, "")
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Resolve project ID from flag or config/env
			projectID = cfg.ResolveProjectID(projectID)

			if projectID == "" {
				return fmt.Errorf("project-id is required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Create client and service
			client, err := cfg.CreateAtlasClient()
			if err != nil {
				return err
			}

			service := atlasservice.NewProjectsService(client)

			// Get project
			project, err := service.Get(ctx, projectID)
			if err != nil {
				return fmt.Errorf("failed to get project: %w", err)
			}

			// Format and output result using proper formatter
			formatter := output.NewFormatter(cfg.Output, os.Stdout)
			if (cfg.Output == config.OutputTable || cfg.Output == config.OutputText || cfg.Output == "") && showTags && project.Tags != nil {
				// Pretty table with tags
				headers := []string{"ID", "NAME", "ORG_ID", "CREATED", "TAGS"}
				tags := ""
				if project.Tags != nil {
					parts := make([]string, 0, len(*project.Tags))
					for _, t := range *project.Tags {
						parts = append(parts, fmt.Sprintf("%s=%s", t.Key, t.Value))
					}
					tags = strings.Join(parts, ",")
				}
				row := []string{*project.Id, project.Name, project.OrgId, project.Created.Format("2006-01-02"), tags}
				return output.NewFormatter(config.OutputTable, os.Stdout).Format(output.TableData{Headers: headers, Rows: [][]string{row}})
			}
			return formatter.Format(project)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().BoolVar(&showTags, "show-tags", false, "Include tags in table output")
	cmd.Flags().String("output", "", "Output format (table, json, yaml)")

	return cmd
}

func newCreateCmd() *cobra.Command {
	var orgID string
	var tagKVs []string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new project",
		Long:  "Create a new MongoDB Atlas project in the specified organization",
		Args:  cobra.ExactArgs(1),
		Example: `  # Create a new project in an organization
  matlas atlas projects create myproject --org-id 507f1f77bcf86cd799439011

  # Create project with a descriptive name
  matlas atlas projects create "My Production Project" --org-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get configuration first to resolve org ID if not provided
			cfg, err := config.Load(cmd, "")
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Resolve org ID from flag or config/env
			orgID = cfg.ResolveOrgID(orgID)

			name := args[0]
			if name == "" {
				return fmt.Errorf("project name is required")
			}

			if orgID == "" {
				return fmt.Errorf("organization ID is required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Create client and service
			client, err := createAtlasClient()
			if err != nil {
				return fmt.Errorf("failed to create Atlas client: %w", err)
			}

			service := atlasservice.NewProjectsService(client)

			// Create project
			fmt.Printf("Creating project '%s' in organization %s...\n", name, orgID)
			// Parse tags of form key=value
			tags := map[string]string{}
			for _, kv := range tagKVs {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 || parts[0] == "" {
					return fmt.Errorf("invalid --tag value: %s (expected key=value)", kv)
				}
				tags[parts[0]] = parts[1]
			}

			project, err := service.Create(ctx, name, orgID, tags)
			if err != nil {
				return fmt.Errorf("failed to create project: %w", err)
			}

			// Display created project details with prettier formatting
			formatter := output.NewCreateResultFormatter(config.OutputText, os.Stdout)
			return formatter.FormatCreateResult(project, "project")
		},
	}

	cmd.Flags().StringVar(&orgID, "org-id", "", "Organization ID where the project will be created (can be set via ATLAS_ORG_ID env var)")
	cmd.Flags().StringArrayVar(&tagKVs, "tag", nil, "Project tag in key=value form (repeatable)")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var projectID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete a project",
		Long: `Delete a MongoDB Atlas project permanently.

WARNING: This action cannot be undone. The project must be empty (no clusters) before deletion.`,
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"rm", "remove"},
		Example: `  # Delete a project with confirmation
  matlas atlas projects delete 507f1f77bcf86cd799439011

  # Delete without confirmation prompt
  matlas atlas projects delete 507f1f77bcf86cd799439011 --yes

  # Using alias
  matlas atlas projects rm 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID = args[0]
			if projectID == "" {
				return fmt.Errorf("project ID is required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Create client and service
			client, err := createAtlasClient()
			if err != nil {
				return fmt.Errorf("failed to create Atlas client: %w", err)
			}

			service := atlasservice.NewProjectsService(client)

			// Get project details first for confirmation
			project, err := service.Get(ctx, projectID)
			if err != nil {
				return fmt.Errorf("failed to get project details: %w", err)
			}

			// Confirmation prompt
			if !yes {
				fmt.Printf("WARNING: You are about to permanently delete the following project:\n")
				fmt.Printf("  Project ID: %s\n", *project.Id)
				fmt.Printf("  Name: %s\n", project.Name)
				fmt.Printf("  Organization ID: %s\n", project.OrgId)
				fmt.Printf("\nThis action cannot be undone. Are you sure? (y/N): ")

				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					return fmt.Errorf("failed to read confirmation: %w", err)
				}
				if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
					fmt.Println("Project deletion cancelled.")
					return nil
				}
			}

			// Delete project
			fmt.Printf("Deleting project '%s' (%s)...\n", project.Name, projectID)
			err = service.Delete(ctx, projectID)
			if err != nil {
				return fmt.Errorf("failed to delete project: %w", err)
			}

			fmt.Printf("✓ Project '%s' deleted successfully\n", project.Name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")

	return cmd
}

// createAtlasClient creates an Atlas client using environment variables or config
func createAtlasClient() (*atlasclient.Client, error) {
	logger := zap.NewNop()

	// When running inside `go test` we want deterministic behaviour even if
	// the developer happens to have Atlas credentials in their shell
	if strings.HasSuffix(os.Args[0], ".test") {
		return nil, fmt.Errorf("ATLAS_PUB_KEY and ATLAS_API_KEY environment variables are required")
	}

	// Try to get credentials from environment variables
	publicKey := os.Getenv("ATLAS_PUB_KEY")
	privateKey := os.Getenv("ATLAS_API_KEY")

	if publicKey == "" || privateKey == "" {
		return nil, fmt.Errorf("ATLAS_PUB_KEY and ATLAS_API_KEY environment variables are required")
	}

	client, err := atlasclient.NewClient(atlasclient.Config{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		RetryMax:   3,
		RetryDelay: 250 * time.Millisecond,
		Logger:     logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Atlas client: %w", err)
	}

	return client, nil
}
