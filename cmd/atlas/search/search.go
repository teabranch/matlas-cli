package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/atlas-sdk/v20250312005/admin"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search",
		Short:   "Manage Atlas Search indexes",
		Long:    "Atlas Search index management commands for full-text and vector search capabilities.",
		Aliases: []string{"search-index", "search-indexes"},
		Hidden:  true, // Hide command as it's still in development
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
		Short:   "List Atlas Search indexes",
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
	var indexName string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get Atlas Search index details by ID or name",
		Long: `Get detailed information about a specific Atlas Search index.

This command retrieves and displays detailed information about a MongoDB Atlas Search index,
including configuration, status, and mapping details.`,
		Example: `  # Get search index details
  matlas atlas search get --project-id 507f1f77bcf86cd799439011 --cluster myCluster --index-id 507f1f77bcf86cd799439012

  # Output as YAML
  matlas atlas search get --project-id 507f1f77bcf86cd799439011 --cluster myCluster --index-id 507f1f77bcf86cd799439012 --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetSearchIndex(cmd, projectID, clusterName, indexID, indexName)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Cluster name (required)")
	cmd.Flags().StringVar(&indexID, "index-id", "", "Search index ID (mutually exclusive with name)")
	cmd.Flags().StringVar(&indexName, "name", "", "Search index name (mutually exclusive with index-id)")
	mustMarkFlagRequired(cmd, "cluster")
	// index-id or name validated at runtime
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
	var indexName string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an Atlas Search index by ID or name",
		Long:  `Delete a MongoDB Atlas Search index. This action cannot be undone.`,
		Example: `  # Delete by name
  matlas atlas search delete --project-id 507f1f77bcf86cd799439011 --cluster myCluster --name mySearchIndex --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteSearchIndex(cmd, projectID, clusterName, indexID, indexName, force)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Cluster name (required)")
	cmd.Flags().StringVar(&indexID, "index-id", "", "Search index ID (mutually exclusive with name)")
	cmd.Flags().StringVar(&indexName, "name", "", "Search index name (mutually exclusive with index-id)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	mustMarkFlagRequired(cmd, "cluster")
	// index-id or name validated at runtime
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
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	// Create search service
	searchService := atlas.NewSearchService(atlasClient)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Prepare optional database and collection names
	var dbName, colName *string
	if databaseName != "" {
		dbName = &databaseName
	}
	if collectionName != "" {
		colName = &collectionName
	}

	// List search indexes
	indexes, err := searchService.ListSearchIndexes(ctx, projectID, clusterName, dbName, colName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to list search indexes")
		return cli.WrapWithSuggestion(err, "Check your project ID and cluster name")
	}

	progress.StopSpinner(fmt.Sprintf("Found %d search index(es)", len(indexes)))

	// Format and display results
	formatter := output.CreateSearchIndexesFormatter()
	return formatter.FormatSearchIndexes(indexes, cmd.Flag("output").Value.String())
}

func runGetSearchIndex(cmd *cobra.Command, projectID, clusterName, indexID, indexName string) error {
	// Load configuration and resolve project ID
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	projectID = cfg.ResolveProjectID(projectID)
	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}
	if clusterName == "" {
		return cli.FormatValidationError("cluster", clusterName, "cluster name cannot be empty")
	}
	// Mutually exclusive flags
	if indexName != "" && indexID != "" {
		return fmt.Errorf("only one of --index-id or --name may be specified")
	}
	if indexName == "" && indexID == "" {
		return fmt.Errorf("either --index-id or --name must be specified")
	}
	// Create spinner
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner("Fetching Atlas Search index details...")
	// Initialize Atlas client and service
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}
	searchService := atlas.NewSearchService(atlasClient)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	// Retrieve index details
	var res *admin.SearchIndexResponse
	if indexName != "" {
		// Find index ID by name
		indexes, err := searchService.ListSearchIndexes(ctx, projectID, clusterName, nil, nil)
		if err != nil {
			progress.StopSpinnerWithError("Failed to list search indexes")
			return cli.WrapWithSuggestion(err, "Check your project ID and cluster name")
		}
		var foundID string
		for _, idx := range indexes {
			if idx.GetName() == indexName {
				foundID = idx.GetIndexID()
				break
			}
		}
		if foundID == "" {
			progress.StopSpinnerWithError("Search index not found")
			return fmt.Errorf("search index %q not found", indexName)
		}
		res, err = searchService.GetSearchIndex(ctx, projectID, clusterName, foundID)
		if err != nil {
			progress.StopSpinnerWithError("Failed to get search index details")
			return cli.WrapWithSuggestion(err, "Check your project ID, cluster name, and index identifier")
		}
	} else {
		res, err = searchService.GetSearchIndex(ctx, projectID, clusterName, indexID)
		if err != nil {
			progress.StopSpinnerWithError("Failed to get search index details")
			return cli.WrapWithSuggestion(err, "Check your project ID, cluster name, and index identifier")
		}
	}
	progress.StopSpinner(fmt.Sprintf("Fetched details for index %s", res.GetName()))
	// Format and display result
	formatter := output.CreateSearchIndexesFormatter()
	return formatter.FormatSearchIndex(*res, cmd.Flag("output").Value.String())
}

func runCreateSearchIndex(cmd *cobra.Command, projectID, clusterName, databaseName, collectionName, indexName, indexFile, indexType string) error {
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

	if databaseName == "" {
		return cli.FormatValidationError("database", databaseName, "database name cannot be empty")
	}

	if collectionName == "" {
		return cli.FormatValidationError("collection", collectionName, "collection name cannot be empty")
	}

	if indexName == "" {
		return cli.FormatValidationError("name", indexName, "index name cannot be empty")
	}

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner("Creating Atlas Search index...")

	// Create Atlas client
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	// Create search service
	searchService := atlas.NewSearchService(atlasClient)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	// Prepare index definition
	var indexDefinition *admin.BaseSearchIndexCreateRequestDefinition
	if indexFile != "" {
		// Load definition from file - validate file path to prevent directory traversal
		if strings.Contains(indexFile, "..") {
			progress.StopSpinnerWithError("Invalid file path")
			return fmt.Errorf("invalid file path: %s", indexFile)
		}
		definitionData, err := os.ReadFile(filepath.Clean(indexFile))
		if err != nil {
			progress.StopSpinnerWithError("Failed to read index definition file")
			return fmt.Errorf("failed to read index file %s: %w", indexFile, err)
		}

		// Parse JSON definition
		var rawDefinition map[string]interface{}
		if err := json.Unmarshal(definitionData, &rawDefinition); err != nil {
			progress.StopSpinnerWithError("Failed to parse index definition")
			return fmt.Errorf("failed to parse index definition: %w", err)
		}

		// Convert to SDK definition
		indexDefinition, err = convertToSearchIndexDefinition(rawDefinition)
		if err != nil {
			progress.StopSpinnerWithError("Invalid index definition")
			return fmt.Errorf("invalid index definition: %w", err)
		}
	} else {
		// Create default definition based on index type
		indexDefinition = createDefaultSearchIndexDefinition(indexType)
	}

	// Create the search index request
	indexRequest := admin.NewSearchIndexCreateRequest(collectionName, databaseName, indexName)

	// Set index type if specified
	if indexType != "" {
		indexRequest.SetType(indexType)
	}

	// Set definition
	if indexDefinition != nil {
		indexRequest.SetDefinition(*indexDefinition)
	}

	// Create the search index
	result, err := searchService.CreateSearchIndex(ctx, projectID, clusterName, *indexRequest)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create search index")
		return cli.WrapWithSuggestion(err, "Check your project ID, cluster name, and index configuration")
	}

	progress.StopSpinner("Search index created successfully")

	// Format and display the result
	formatter := output.CreateSearchIndexesFormatter()
	return formatter.FormatSearchIndex(*result, cmd.Flag("output").Value.String())
}

func runUpdateSearchIndex(cmd *cobra.Command, projectID, clusterName, indexID, indexFile string) error {
	// Similar implementation structure as list, but for updating an index
	return fmt.Errorf("atlas search API not yet available in SDK - see 'matlas atlas search list --help' for alternatives")
}

func runDeleteSearchIndex(cmd *cobra.Command, projectID, clusterName, indexID, indexName string, force bool) error {
	// Load configuration and resolve project ID
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	projectID = cfg.ResolveProjectID(projectID)
	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}
	if clusterName == "" {
		return cli.FormatValidationError("cluster", clusterName, "cluster name cannot be empty")
	}
	// Mutually exclusive flags
	if indexName != "" && indexID != "" {
		return fmt.Errorf("only one of --index-id or --name may be specified")
	}
	if indexName == "" && indexID == "" {
		return fmt.Errorf("either --index-id or --name must be specified")
	}
	// Create spinner
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner("Deleting Atlas Search index...")
	// Initialize Atlas client
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}
	searchService := atlas.NewSearchService(atlasClient)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	// Determine deletion method
	var deleteErr error
	if indexName != "" {
		// Delete by name: list and find index ID
		indexes, err := searchService.ListSearchIndexes(ctx, projectID, clusterName, nil, nil)
		if err != nil {
			progress.StopSpinnerWithError("Failed to list search indexes")
			return cli.WrapWithSuggestion(err, "Check your project ID and cluster name")
		}
		var foundID string
		for _, idx := range indexes {
			if idx.GetName() == indexName {
				foundID = idx.GetIndexID()
				break
			}
		}
		if foundID == "" {
			progress.StopSpinnerWithError("Search index not found")
			return fmt.Errorf("search index %q not found", indexName)
		}
		deleteErr = searchService.DeleteSearchIndex(ctx, projectID, clusterName, foundID)
	} else {
		// Delete by ID
		deleteErr = searchService.DeleteSearchIndex(ctx, projectID, clusterName, indexID)
	}
	if deleteErr != nil {
		progress.StopSpinnerWithError("Failed to delete search index")
		return cli.WrapWithSuggestion(deleteErr, "Check your project ID, cluster name, and index information")
	}
	progress.StopSpinner("Search index deleted successfully")
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

// mustMarkFlagRequired marks a flag as required and panics if it fails.
// This should never fail in normal execution and indicates a programmer error if it does.
func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Errorf("failed to mark flag %q required: %w", name, err))
	}
}

// createDefaultSearchIndexDefinition creates a default search index definition based on type
func createDefaultSearchIndexDefinition(indexType string) *admin.BaseSearchIndexCreateRequestDefinition {
	definition := admin.NewBaseSearchIndexCreateRequestDefinitionWithDefaults()

	if indexType == "vectorSearch" {
		// Create default vector search definition
		// Note: Vector search requires specific field definitions
		// This is a basic example - users should provide proper vector field definitions
		fields := []any{
			map[string]interface{}{
				"type":          "vector",
				"path":          "vector_field", // Default vector field name
				"numDimensions": 1536,           // Common dimension size for embeddings
				"similarity":    "cosine",
			},
		}
		definition.SetFields(fields)
		definition.Analyzer = nil
		definition.SearchAnalyzer = nil
	} else {
		// Create default text search definition with dynamic mapping
		mappings := admin.SearchMappings{}
		// Set dynamic mapping to true (allows indexing all fields)
		dynamic := true
		mappings.SetDynamic(dynamic)
		definition.SetMappings(mappings)
	}

	return definition
}

// convertToSearchIndexDefinition converts a raw definition to SDK definition
func convertToSearchIndexDefinition(rawDefinition map[string]interface{}) (*admin.BaseSearchIndexCreateRequestDefinition, error) {
	definition := admin.NewBaseSearchIndexCreateRequestDefinitionWithDefaults()

	// Convert mappings if present
	if mappingsRaw, ok := rawDefinition["mappings"]; ok {
		if mappingsMap, ok := mappingsRaw.(map[string]interface{}); ok {
			mappings := admin.SearchMappings{}

			// Handle dynamic mapping
			if dynamic, ok := mappingsMap["dynamic"]; ok {
				if dynamicBool, ok := dynamic.(bool); ok {
					mappings.SetDynamic(dynamicBool)
				}
			}

			// Handle fields mapping
			if fields, ok := mappingsMap["fields"]; ok {
				if fieldsMap, ok := fields.(map[string]interface{}); ok {
					mappings.SetFields(fieldsMap)
				}
			}

			definition.SetMappings(mappings)
		}
	}

	// Convert fields if present (for vector search)
	if fieldsRaw, ok := rawDefinition["fields"]; ok {
		if fieldsSlice, ok := fieldsRaw.([]interface{}); ok {
			definition.SetFields(fieldsSlice)
		}
	}

	// Convert analyzer if present (only for non-vector search)
	// Note: Vector search indexes don't support analyzers
	if analyzer, ok := rawDefinition["analyzer"]; ok {
		if analyzerStr, ok := analyzer.(string); ok {
			definition.SetAnalyzer(analyzerStr)
		}
	}

	// Convert searchAnalyzer if present (only for non-vector search)
	if searchAnalyzer, ok := rawDefinition["searchAnalyzer"]; ok {
		if searchAnalyzerStr, ok := searchAnalyzer.(string); ok {
			definition.SetSearchAnalyzer(searchAnalyzerStr)
		}
	}

	return definition, nil
}
