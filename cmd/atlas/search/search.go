package search

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search",
		Short:   "Manage Atlas Search indexes (unsupported in this build)",
		Long:    "Atlas Search index management commands are currently disabled in this build because the required SDK APIs are not yet available.",
		Aliases: []string{"search-index", "search-indexes"},
		Hidden:  true,
	}

	// Keep subcommands registered but keep parent hidden; each command returns a clear, consistent message.
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var projectID string
	var clusterName string
	var databaseName string
	var collectionName string
	var paginationFlags cli.PaginationFlags

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Atlas Search indexes (unsupported)",
		Long: `List all Atlas Search indexes in a cluster, database, or collection.

This command retrieves and displays all MongoDB Atlas Search indexes.
The output includes index name, status, type, and configuration details.`,
		Example: `  # List all search indexes in a cluster
  matlas atlas search list --project-id 507f1f77bcf86cd799439011 --cluster myCluster

  # List search indexes for a specific collection
  matlas atlas search list --project-id 507f1f77bcf86cd799439011 --cluster myCluster \
    --database myDB --collection myCollection

  # List with pagination
  matlas atlas search list --project-id 507f1f77bcf86cd799439011 --cluster myCluster --page 2 --limit 10

  # Output as JSON
  matlas atlas search list --project-id 507f1f77bcf86cd799439011 --cluster myCluster --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListSearchIndexes(cmd, projectID, clusterName, databaseName, collectionName, &paginationFlags)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Cluster name (required)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (optional)")
	cmd.Flags().StringVar(&collectionName, "collection", "", "Collection name (requires database)")
	mustMarkFlagRequired(cmd, "cluster")

	cli.AddPaginationFlags(cmd, &paginationFlags)

	return cmd
}

func newGetCmd() *cobra.Command {
	var projectID string
	var clusterName string
	var indexID string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get Atlas Search index details (unsupported)",
		Long: `Get detailed information about a specific Atlas Search index.

This command retrieves and displays detailed information about a MongoDB Atlas Search index,
including configuration, status, and mapping details.`,
		Example: `  # Get search index details
  matlas atlas search get --project-id 507f1f77bcf86cd799439011 --cluster myCluster --index-id 507f1f77bcf86cd799439012

  # Output as YAML
  matlas atlas search get --project-id 507f1f77bcf86cd799439011 --cluster myCluster --index-id 507f1f77bcf86cd799439012 --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetSearchIndex(cmd, projectID, clusterName, indexID)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Cluster name (required)")
	cmd.Flags().StringVar(&indexID, "index-id", "", "Search index ID (required)")
	mustMarkFlagRequired(cmd, "cluster")
	mustMarkFlagRequired(cmd, "index-id")

	return cmd
}

func newCreateCmd() *cobra.Command {
	var projectID string
	var clusterName string
	var databaseName string
	var collectionName string
	var indexName string
	var indexFile string
	var indexType string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an Atlas Search index (unsupported)",
		Long: `Create a new Atlas Search index for full-text search or vector search.

This command creates a new MongoDB Atlas Search index on a collection.
You can specify the index definition inline or provide a JSON file with the definition.`,
		Example: `  # Create a basic search index
  matlas atlas search create --project-id 507f1f77bcf86cd799439011 --cluster myCluster \
    --database myDB --collection myCollection --name mySearchIndex

  # Create search index from definition file
  matlas atlas search create --project-id 507f1f77bcf86cd799439011 --cluster myCluster \
    --database myDB --collection myCollection --name mySearchIndex --index-file search-definition.json

  # Create vector search index
  matlas atlas search create --project-id 507f1f77bcf86cd799439011 --cluster myCluster \
    --database myDB --collection myCollection --name myVectorIndex --type vectorSearch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateSearchIndex(cmd, projectID, clusterName, databaseName, collectionName, indexName, indexFile, indexType)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Cluster name (required)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().StringVar(&collectionName, "collection", "", "Collection name (required)")
	cmd.Flags().StringVar(&indexName, "name", "", "Search index name (required)")
	cmd.Flags().StringVar(&indexFile, "index-file", "", "Path to JSON file containing index definition")
	cmd.Flags().StringVar(&indexType, "type", "search", "Index type: search, vectorSearch")
	mustMarkFlagRequired(cmd, "cluster")
	mustMarkFlagRequired(cmd, "database")
	mustMarkFlagRequired(cmd, "collection")
	mustMarkFlagRequired(cmd, "name")

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var projectID string
	var clusterName string
	var indexID string
	var indexFile string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an Atlas Search index (unsupported)",
		Long: `Update an existing Atlas Search index configuration.

This command updates the configuration of an existing MongoDB Atlas Search index.
The new definition must be provided via a JSON file.`,
		Example: `  # Update search index from definition file
  matlas atlas search update --project-id 507f1f77bcf86cd799439011 --cluster myCluster \
    --index-id 507f1f77bcf86cd799439012 --index-file updated-search-definition.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdateSearchIndex(cmd, projectID, clusterName, indexID, indexFile)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Cluster name (required)")
	cmd.Flags().StringVar(&indexID, "index-id", "", "Search index ID (required)")
	cmd.Flags().StringVar(&indexFile, "index-file", "", "Path to JSON file containing updated index definition (required)")
	mustMarkFlagRequired(cmd, "cluster")
	mustMarkFlagRequired(cmd, "index-id")
	mustMarkFlagRequired(cmd, "index-file")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var projectID string
	var clusterName string
	var indexID string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an Atlas Search index (unsupported)",
		Long: `Delete an Atlas Search index.

This command deletes a MongoDB Atlas Search index. This action cannot be undone.
All search functionality depending on this index will stop working.`,
		Example: `  # Delete search index with confirmation
  matlas atlas search delete --project-id 507f1f77bcf86cd799439011 --cluster myCluster --index-id 507f1f77bcf86cd799439012

  # Delete without confirmation prompt
  matlas atlas search delete --project-id 507f1f77bcf86cd799439011 --cluster myCluster --index-id 507f1f77bcf86cd799439012 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteSearchIndex(cmd, projectID, clusterName, indexID, force)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Cluster name (required)")
	cmd.Flags().StringVar(&indexID, "index-id", "", "Search index ID (required)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	mustMarkFlagRequired(cmd, "cluster")
	mustMarkFlagRequired(cmd, "index-id")

	return cmd
}

func runListSearchIndexes(cmd *cobra.Command, projectID, clusterName, databaseName, collectionName string, paginationFlags *cli.PaginationFlags) error {
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

	if clusterName == "" {
		return cli.FormatValidationError("cluster", clusterName, "cluster name cannot be empty")
	}

	// Validate pagination
	_, err = paginationFlags.Validate()
	if err != nil {
		return err
	}

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner("Fetching Atlas Search indexes...")

	// Create Atlas client
	_, err = cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	// Note: Atlas Search API endpoints need to be implemented when available in SDK
	// For now, return a standardized unsupported message
	progress.StopSpinnerWithError("Atlas Search API not yet available in SDK")
	return cli.UnsupportedSearchAPIError()
}

func runGetSearchIndex(cmd *cobra.Command, projectID, clusterName, indexID string) error {
	// Similar implementation structure as list, but for getting a single index
	return fmt.Errorf("atlas search API not yet available in SDK - see 'matlas atlas search list --help' for alternatives")
}

func runCreateSearchIndex(cmd *cobra.Command, projectID, clusterName, databaseName, collectionName, indexName, indexFile, indexType string) error {
	// Similar implementation structure as list, but for creating an index
	return fmt.Errorf("atlas search API not yet available in SDK - see 'matlas atlas search list --help' for alternatives")
}

func runUpdateSearchIndex(cmd *cobra.Command, projectID, clusterName, indexID, indexFile string) error {
	// Similar implementation structure as list, but for updating an index
	return fmt.Errorf("atlas search API not yet available in SDK - see 'matlas atlas search list --help' for alternatives")
}

func runDeleteSearchIndex(cmd *cobra.Command, projectID, clusterName, indexID string, force bool) error {
	// Similar implementation structure as list, but for deleting an index
	return fmt.Errorf("atlas search API not yet available in SDK - see 'matlas atlas search list --help' for alternatives")
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

// mustMarkFlagRequired marks a flag as required and panics if it fails.
// This should never fail in normal execution and indicates a programmer error if it does.
func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Errorf("failed to mark flag %q required: %w", name, err))
	}
}
