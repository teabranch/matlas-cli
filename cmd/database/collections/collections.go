package collections

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	atlasservice "github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func NewCollectionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "collections",
		Short:   "Manage MongoDB collections",
		Long:    "List, create, and manage MongoDB collections",
		Aliases: []string{"cols", "coll"},
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newIndexesCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var paginationFlags cli.PaginationFlags

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List collections",
		Long: `List all collections in a database.

This command retrieves and displays all collections in the specified database.
The output includes collection name, type, document count, size, and average object size.`,
		Example: `  # List collections using connection string
  matlas database collections list --connection-string "mongodb+srv://..." --database mydb

  # List collections using Atlas cluster reference
  matlas database collections list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb

  # Output as JSON for automation
  matlas database collections list --connection-string "mongodb+srv://..." --database mydb --output json

  # Using alias
  matlas db cols ls --connection-string "mongodb+srv://..." --database mydb`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListCollections(cmd, connectionString, clusterName, projectID, databaseName, &paginationFlags)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")

	// At least one connection method is required
	mustMarkFlagsOneRequired(cmd, "connection-string", "cluster")
	mustMarkFlagsRequiredTogether(cmd, "cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	cli.AddPaginationFlags(cmd, &paginationFlags)

	return cmd
}

func newCreateCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var capped bool
	var size int64
	var maxDocuments int64

	cmd := &cobra.Command{
		Use:   "create <collection-name>",
		Short: "Create a collection",
		Long:  "Create a new MongoDB collection",
		Args:  cobra.ExactArgs(1),
		Example: `  # Create a regular collection
  matlas database collections create mycollection --database mydb --connection-string "mongodb+srv://..."

  # Create a capped collection
  matlas database collections create mycappedcoll --database mydb --connection-string "mongodb+srv://..." --capped --size 1048576 --max-documents 1000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			collectionName := args[0]
			return runCreateCollection(cmd, connectionString, clusterName, projectID, databaseName, collectionName, capped, size, maxDocuments)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVar(&capped, "capped", false, "Create a capped collection")
	cmd.Flags().Int64Var(&size, "size", 0, "Maximum size in bytes for capped collection")
	cmd.Flags().Int64Var(&maxDocuments, "max-documents", 0, "Maximum number of documents for capped collection")

	// At least one connection method is required
	mustMarkFlagsOneRequired(cmd, "connection-string", "cluster")
	mustMarkFlagsRequiredTogether(cmd, "cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

func newDeleteCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <collection-name>",
		Aliases: []string{"del", "rm", "remove"},
		Short:   "Delete a collection",
		Long: `Delete a MongoDB collection and all its documents.

⚠️  WARNING: This operation permanently deletes the collection and all its data.
Use with caution in production environments.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Delete collection (with confirmation)
  matlas database collections delete mycollection --database mydb --connection-string "mongodb+srv://..."

  # Delete collection (skip confirmation)
  matlas database collections delete mycollection --database mydb --connection-string "mongodb+srv://..." --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			collectionName := args[0]
			return runDeleteCollection(cmd, connectionString, clusterName, projectID, databaseName, collectionName, yes)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	// At least one connection method is required
	mustMarkFlagsOneRequired(cmd, "connection-string", "cluster")
	mustMarkFlagsRequiredTogether(cmd, "cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")

	return cmd
}

func newIndexesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "indexes",
		Short:   "Manage collection indexes",
		Long:    "List, create, and delete indexes on MongoDB collections",
		Aliases: []string{"idx", "index"},
	}

	cmd.AddCommand(newListIndexesCmd())
	cmd.AddCommand(newCreateIndexCmd())
	cmd.AddCommand(newDeleteIndexCmd())

	return cmd
}

func newListIndexesCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var collectionName string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List indexes",
		Long: `List all indexes on a collection.

This command retrieves and displays all indexes on the specified collection,
including index names, keys, and options.`,
		Example: `  # List indexes using connection string
  matlas database collections indexes list --connection-string "mongodb+srv://..." --database mydb --collection mycoll

  # List indexes using Atlas cluster reference
  matlas database collections indexes list --cluster MyCluster --project-id 507f1f77bcf86cd799439011 --database mydb --collection mycoll

  # Output as JSON for automation
  matlas database collections indexes list --connection-string "mongodb+srv://..." --database mydb --collection mycoll --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListIndexes(cmd, connectionString, clusterName, projectID, databaseName, collectionName)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().StringVar(&collectionName, "collection", "", "Collection name (required)")

	// At least one connection method is required
	mustMarkFlagsOneRequired(cmd, "connection-string", "cluster")
	mustMarkFlagsRequiredTogether(cmd, "cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")
	mustMarkFlagRequired(cmd, "collection")

	return cmd
}

func newCreateIndexCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var collectionName string
	var indexName string
	var unique bool
	var sparse bool
	var background bool

	cmd := &cobra.Command{
		Use:   "create <field:order> [field:order...]",
		Short: "Create an index",
		Long: `Create an index on a collection.

Specify index keys as field:order pairs where order is 1 for ascending or -1 for descending.
Multiple fields can be specified to create a compound index.`,
		Args: cobra.MinimumNArgs(1),
		Example: `  # Create a simple ascending index
  matlas database collections indexes create username:1 --database mydb --collection users --connection-string "mongodb+srv://..."

  # Create a compound index
  matlas database collections indexes create category:1 createdAt:-1 --database mydb --collection posts --connection-string "mongodb+srv://..."

  # Create a unique index with a custom name
  matlas database collections indexes create email:1 --database mydb --collection users --connection-string "mongodb+srv://..." --unique --name email_unique_idx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateIndex(cmd, connectionString, clusterName, projectID, databaseName, collectionName, args, indexName, unique, sparse, background)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().StringVar(&collectionName, "collection", "", "Collection name (required)")
	cmd.Flags().StringVar(&indexName, "name", "", "Custom index name")
	cmd.Flags().BoolVar(&unique, "unique", false, "Create a unique index")
	cmd.Flags().BoolVar(&sparse, "sparse", false, "Create a sparse index")
	cmd.Flags().BoolVar(&background, "background", false, "Create index in background")

	// At least one connection method is required
	mustMarkFlagsOneRequired(cmd, "connection-string", "cluster")
	mustMarkFlagsRequiredTogether(cmd, "cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")
	mustMarkFlagRequired(cmd, "collection")

	return cmd
}

func newDeleteIndexCmd() *cobra.Command {
	var connectionString string
	var clusterName string
	var projectID string
	var databaseName string
	var collectionName string
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <index-name>",
		Aliases: []string{"del", "rm", "remove"},
		Short:   "Delete an index",
		Long: `Delete an index from a collection.

⚠️  WARNING: Deleting an index may impact query performance.
Make sure you understand the impact before deleting production indexes.`,
		Args: cobra.ExactArgs(1),
		Example: `  # Delete an index (with confirmation)
  matlas database collections indexes delete email_unique_idx --database mydb --collection users --connection-string "mongodb+srv://..."

  # Delete an index (skip confirmation)
  matlas database collections indexes delete email_unique_idx --database mydb --collection users --connection-string "mongodb+srv://..." --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			indexName := args[0]
			return runDeleteIndex(cmd, connectionString, clusterName, projectID, databaseName, collectionName, indexName, yes)
		},
	}

	cmd.Flags().StringVar(&connectionString, "connection-string", "", "MongoDB connection string")
	cmd.Flags().StringVar(&clusterName, "cluster", "", "Atlas cluster name (requires --project-id)")
	cmd.Flags().StringVar(&projectID, "project-id", "", "Atlas project ID (used with --cluster)")
	cmd.Flags().StringVar(&databaseName, "database", "", "Database name (required)")
	cmd.Flags().StringVar(&collectionName, "collection", "", "Collection name (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	// At least one connection method is required
	mustMarkFlagsOneRequired(cmd, "connection-string", "cluster")
	mustMarkFlagsRequiredTogether(cmd, "cluster", "project-id")
	mustMarkFlagRequired(cmd, "database")
	mustMarkFlagRequired(cmd, "collection")

	return cmd
}

// Implementation functions

func runListCollections(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName string, paginationFlags *cli.PaginationFlags) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate pagination
	paginationOpts, err := paginationFlags.Validate()
	if err != nil {
		return err
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	// Resolve connection info
	connInfo, err := resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, false, "", progress)
	if err != nil {
		return err
	}

	// Set up cleanup for temporary user if one was created
	if connInfo.TempUser != nil && connInfo.TempUser.CleanupFunc != nil {
		defer func() {
			progress.StartSpinner("Cleaning up temporary user...")
			if cleanupErr := connInfo.TempUser.CleanupFunc(ctx); cleanupErr != nil {
				progress.StopSpinnerWithError("Failed to cleanup temporary user")
				fmt.Printf("Warning: Failed to cleanup temporary user: %v\n", cleanupErr)
			} else {
				progress.StopSpinner("Temporary user cleaned up")
			}
		}()
	}

	progress.StartSpinner(fmt.Sprintf("Listing collections in database '%s'...", databaseName))

	// Create database service
	zapLogger, _ := zap.NewDevelopment()
	dbService := database.NewService(zapLogger)
	defer func() {
		if err := dbService.Close(ctx); err != nil {
			fmt.Printf("Warning: Failed to close database service: %v\n", err)
		}
	}()

	// List collections
	collections, err := dbService.ListCollections(ctx, connInfo, databaseName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to list collections")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Found %d collection(s)", len(collections)))

	// Apply pagination if needed
	if paginationOpts.ShouldPaginate() && !paginationFlags.All {
		skip := paginationOpts.CalculateSkip()
		end := skip + paginationOpts.Limit

		if skip >= len(collections) {
			collections = []types.CollectionInfo{}
		} else {
			if end > len(collections) {
				end = len(collections)
			}
			collections = collections[skip:end]
		}
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, collections,
		[]string{"NAME", "TYPE", "COUNT", "SIZE", "AVG_OBJ_SIZE"},
		func(item interface{}) []string {
			coll := item.(types.CollectionInfo)
			sizeStr := fmt.Sprintf("%.2f MB", float64(coll.Info.Size)/(1024*1024))
			avgSizeStr := fmt.Sprintf("%.0f bytes", float64(coll.Info.AvgObjSize))

			return []string{
				coll.Name,
				coll.Type,
				fmt.Sprintf("%d", coll.Info.Count),
				sizeStr,
				avgSizeStr,
			}
		})
}

func runCreateCollection(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, collectionName string, capped bool, size, maxDocuments int64) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	// Resolve connection info
	connInfo, err := resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, false, "", progress)
	if err != nil {
		return err
	}

	// Prepare collection options
	var opts map[string]interface{}
	if capped {
		opts = map[string]interface{}{
			"capped": true,
		}
		if size > 0 {
			opts["size"] = size
		}
		if maxDocuments > 0 {
			opts["max"] = maxDocuments
		}
	}

	progress.StartSpinner(fmt.Sprintf("Creating collection '%s'...", collectionName))

	// Create database service
	zapLogger, _ := zap.NewDevelopment()
	dbService := database.NewService(zapLogger)
	defer func() {
		if err := dbService.Close(ctx); err != nil {
			fmt.Printf("Warning: Failed to close database service: %v\n", err)
		}
	}()

	// Create collection
	err = dbService.CreateCollection(ctx, connInfo, databaseName, collectionName, opts)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create collection")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Collection '%s' created successfully", collectionName))
	return nil
}

func runDeleteCollection(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, collectionName string, yes bool) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}

	// Get confirmation unless --yes flag is used
	if !yes {
		confirm := ui.NewConfirmationPrompt(false, false)
		confirmed, err := confirm.ConfirmDeletion("collection", fmt.Sprintf("%s.%s", databaseName, collectionName))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	// Resolve connection info
	connInfo, err := resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, false, "", progress)
	if err != nil {
		return err
	}

	progress.StartSpinner(fmt.Sprintf("Deleting collection '%s'...", collectionName))

	// Create database service
	zapLogger, _ := zap.NewDevelopment()
	dbService := database.NewService(zapLogger)
	defer func() {
		if err := dbService.Close(ctx); err != nil {
			fmt.Printf("Warning: Failed to close database service: %v\n", err)
		}
	}()

	// Delete collection
	err = dbService.DropCollection(ctx, connInfo, databaseName, collectionName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to delete collection")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Collection '%s' deleted successfully", collectionName))
	return nil
}

func runListIndexes(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, collectionName string) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	// Resolve connection info
	connInfo, err := resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, false, "", progress)
	if err != nil {
		return err
	}

	progress.StartSpinner(fmt.Sprintf("Listing indexes for collection '%s.%s'...", databaseName, collectionName))

	// Create database service
	zapLogger, _ := zap.NewDevelopment()
	dbService := database.NewService(zapLogger)
	defer func() {
		if err := dbService.Close(ctx); err != nil {
			fmt.Printf("Warning: Failed to close database service: %v\n", err)
		}
	}()

	// List indexes
	indexes, err := dbService.ListIndexes(ctx, connInfo, databaseName, collectionName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to list indexes")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Found %d index(es)", len(indexes)))

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, indexes,
		[]string{"NAME", "KEYS", "UNIQUE", "SPARSE", "VERSION"},
		func(item interface{}) []string {
			index := item.(types.IndexInfo)

			// Format keys
			var keysStr string
			if len(index.Keys) > 0 {
				keyParts := make([]string, 0, len(index.Keys))
				for field, order := range index.Keys {
					keyParts = append(keyParts, fmt.Sprintf("%s:%v", field, order))
				}
				keysStr = strings.Join(keyParts, ", ")
			}

			uniqueStr := "false"
			if index.Unique {
				uniqueStr = "true"
			}

			sparseStr := "false"
			if index.Sparse {
				sparseStr = "true"
			}

			return []string{
				index.Name,
				keysStr,
				uniqueStr,
				sparseStr,
				fmt.Sprintf("%d", index.Version),
			}
		})
}

func runCreateIndex(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, collectionName string, keySpecs []string, indexName string, unique, sparse, background bool) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}
	if len(keySpecs) == 0 {
		return fmt.Errorf("at least one index key specification is required")
	}

	// Parse key specifications
	keys := make(map[string]int)
	for _, keySpec := range keySpecs {
		parts := strings.Split(keySpec, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid key specification '%s': expected format 'field:order'", keySpec)
		}

		field := parts[0]
		orderStr := parts[1]

		order, err := strconv.Atoi(orderStr)
		if err != nil {
			return fmt.Errorf("invalid order value '%s' in key specification '%s': must be 1 or -1", orderStr, keySpec)
		}

		if order != 1 && order != -1 {
			return fmt.Errorf("invalid order value %d in key specification '%s': must be 1 (ascending) or -1 (descending)", order, keySpec)
		}

		keys[field] = order
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	// Resolve connection info
	connInfo, err := resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, false, "", progress)
	if err != nil {
		return err
	}

	// Prepare index options
	opts := make(map[string]interface{})
	if indexName != "" {
		opts["name"] = indexName
	}
	if unique {
		opts["unique"] = true
	}
	if sparse {
		opts["sparse"] = true
	}
	if background {
		opts["background"] = true
	}

	progress.StartSpinner(fmt.Sprintf("Creating index on collection '%s.%s'...", databaseName, collectionName))

	// Create database service
	zapLogger, _ := zap.NewDevelopment()
	dbService := database.NewService(zapLogger)
	defer func() {
		if err := dbService.Close(ctx); err != nil {
			fmt.Printf("Warning: Failed to close database service: %v\n", err)
		}
	}()

	// Create index
	createdIndexName, err := dbService.CreateIndex(ctx, connInfo, databaseName, collectionName, keys, opts)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create index")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Index '%s' created successfully", createdIndexName))
	return nil
}

func runDeleteIndex(cmd *cobra.Command, connectionString, clusterName, projectID, databaseName, collectionName, indexName string, yes bool) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}
	if indexName == "" {
		return fmt.Errorf("index name is required")
	}

	// Get confirmation unless --yes flag is used
	if !yes {
		confirm := ui.NewConfirmationPrompt(false, false)
		confirmed, err := confirm.ConfirmDeletion("index", fmt.Sprintf("%s.%s.%s", databaseName, collectionName, indexName))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Get configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	// Resolve connection info
	connInfo, err := resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, false, "", progress)
	if err != nil {
		return err
	}

	progress.StartSpinner(fmt.Sprintf("Deleting index '%s'...", indexName))

	// Create database service
	zapLogger, _ := zap.NewDevelopment()
	dbService := database.NewService(zapLogger)
	defer func() {
		if err := dbService.Close(ctx); err != nil {
			fmt.Printf("Warning: Failed to close database service: %v\n", err)
		}
	}()

	// Delete index
	err = dbService.DropIndex(ctx, connInfo, databaseName, collectionName, indexName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to delete index")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Index '%s' deleted successfully", indexName))
	return nil
}

// resolveConnectionInfo resolves connection information from either direct connection string or Atlas cluster
func resolveConnectionInfo(ctx context.Context, cfg *config.Config, connectionString, clusterName, projectID string, useTempUser bool, databaseName string, progress *ui.ProgressIndicator) (*types.ConnectionInfo, error) {
	if connectionString != "" {
		// Direct connection string provided
		return &types.ConnectionInfo{
			ConnectionString: connectionString,
		}, nil
	}

	// Need to resolve Atlas cluster connection string
	if clusterName == "" || projectID == "" {
		return nil, fmt.Errorf("cluster name and project ID are required when not using connection string")
	}

	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return nil, cli.FormatValidationError("project-id", projectID, err.Error())
	}

	if err := validation.ValidateClusterName(clusterName); err != nil {
		return nil, cli.FormatValidationError("cluster", clusterName, err.Error())
	}

	progress.StartSpinner(fmt.Sprintf("Resolving connection string for cluster '%s'...", clusterName))

	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return nil, cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlasservice.NewClustersService(client)

	// Get cluster details
	cluster, err := service.Get(ctx, projectID, clusterName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to get cluster details")
		errorFormatter := cli.NewErrorFormatter(true) // verbose for troubleshooting
		return nil, fmt.Errorf("%s", errorFormatter.Format(err))
	}

	// Extract connection string from cluster
	if cluster.ConnectionStrings == nil || cluster.ConnectionStrings.StandardSrv == nil {
		progress.StopSpinnerWithError("No connection string available")
		return nil, fmt.Errorf("cluster '%s' does not have a connection string available", clusterName)
	}

	connectionString = *cluster.ConnectionStrings.StandardSrv
	progress.StopSpinner("Connection string resolved")

	return &types.ConnectionInfo{
		ConnectionString: connectionString,
	}, nil
}

// Helpers to enforce required flags during command setup
func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Errorf("failed to mark flag %q required: %w", name, err))
	}
}

func mustMarkFlagsOneRequired(cmd *cobra.Command, name1, name2 string) {
	// Under our cobra version these helpers do not return an error
	cmd.MarkFlagsOneRequired(name1, name2)
}

func mustMarkFlagsRequiredTogether(cmd *cobra.Command, name1, name2 string) {
	// Under our cobra version these helpers do not return an error
	cmd.MarkFlagsRequiredTogether(name1, name2)
}
