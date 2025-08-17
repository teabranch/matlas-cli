package database

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/ui"
)

func runListDatabases(cmd *cobra.Command, connectionString, clusterName, projectID string, useTempUser bool, databaseName string) error {
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
	connInfo, err := resolveConnectionInfoWithCmd(ctx, cmd, cfg, connectionString, clusterName, projectID, useTempUser, databaseName, progress)
	if err != nil {
		return err
	}

	// Set up cleanup for temporary user if one was created
	if connInfo.TempUser != nil && connInfo.TempUser.CleanupFunc != nil {
		defer func() {
			progress.StartSpinner("Cleaning up temporary user...")
			if cleanupErr := connInfo.TempUser.CleanupFunc(ctx); cleanupErr != nil {
				progress.StopSpinnerWithError("Failed to cleanup temporary user")
				// Log the error but don't fail the main operation
				fmt.Printf("Warning: Failed to cleanup temporary user: %v\n", cleanupErr)
			} else {
				progress.StopSpinner("Temporary user cleaned up")
			}
		}()
	}

	// Add debugging information
	verbose := cmd.Flag("verbose").Changed
	if verbose {
		fmt.Printf("Debug: Using connection string: %s\n", maskConnectionString(connInfo.ConnectionString))
		if connInfo.TempUser != nil {
			fmt.Printf("Debug: Using temporary user: %s (expires: %s)\n",
				connInfo.TempUser.Username, connInfo.TempUser.ExpiresAt.Format("15:04:05"))
			fmt.Printf("Debug: Connection string length: %d characters\n", len(connInfo.ConnectionString))

			// Check connection string format
			if strings.Contains(connInfo.ConnectionString, "@") {
				fmt.Printf("Debug: Connection string contains credentials (good)\n")
			} else {
				fmt.Printf("Debug: WARNING - Connection string does NOT contain credentials\n")
			}

			if strings.Contains(connInfo.ConnectionString, "authSource=admin") {
				fmt.Printf("Debug: Connection string has authSource=admin (good)\n")
			} else {
				fmt.Printf("Debug: WARNING - Connection string missing authSource=admin\n")
			}
		} else {
			fmt.Printf("Debug: No temporary user information available\n")
		}
	}

	progress.StartSpinner("Listing databases...")

	// Create database service
	logger := logging.Default() // For compatibility with database service
	dbService := database.NewService(logger)
	defer func() {
		if err := dbService.Close(ctx); err != nil {
			fmt.Printf("Warning: Failed to close database service: %v\n", err)
		}
	}()

	// List databases
	databases, err := dbService.ListDatabases(ctx, connInfo)
	if err != nil {
		progress.StopSpinnerWithError("Failed to list databases")

		// Enhanced error reporting
		if verbose {
			fmt.Printf("Debug: Database listing failed with error: %v\n", err)
			fmt.Printf("Debug: Connection info - TempUser: %v, ConnString length: %d\n",
				connInfo.TempUser != nil, len(connInfo.ConnectionString))
			fmt.Printf("Debug: Error type: %T\n", err)
			fmt.Printf("Debug: Error string contains 'auth': %v\n", strings.Contains(err.Error(), "auth"))
			fmt.Printf("Debug: Error string contains 'permission': %v\n", strings.Contains(err.Error(), "permission"))
		}

		errorFormatter := cli.NewErrorFormatter(verbose)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	// Enhanced database listing results
	if verbose {
		fmt.Printf("Debug: Successfully listed %d databases\n", len(databases))
		if len(databases) == 0 {
			fmt.Printf("Debug: WARNING - No databases returned (this might be due to empty cluster or permissions)\n")
		} else {
			fmt.Printf("Debug: Database names: ")
			for i, db := range databases {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", db.Name)
			}
			fmt.Printf("\n")
		}
	}

	progress.StopSpinner(fmt.Sprintf("Found %d database(s)", len(databases)))

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, databases,
		[]string{"NAME", "SIZE_ON_DISK", "EMPTY", "COLLECTIONS"},
		func(item interface{}) []string {
			db := item.(types.DatabaseInfo)
			sizeStr := fmt.Sprintf("%.2f MB", float64(db.SizeOnDisk)/(1024*1024))
			emptyStr := "No"
			if db.Empty {
				emptyStr = "Yes"
			}
			collectionsStr := fmt.Sprintf("%d", len(db.Collections))

			return []string{db.Name, sizeStr, emptyStr, collectionsStr}
		})
}
