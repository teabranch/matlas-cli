package database

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
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
	connInfo, err := resolveConnectionInfo(ctx, cfg, connectionString, clusterName, projectID, useTempUser, databaseName, progress)
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

	progress.StartSpinner("Listing databases...")

	// Create database service
	zapLogger, _ := zap.NewDevelopment() // For compatibility with database service
	dbService := database.NewService(zapLogger)
	defer dbService.Close(ctx)

	// List databases
	databases, err := dbService.ListDatabases(ctx, connInfo)
	if err != nil {
		progress.StopSpinnerWithError("Failed to list databases")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
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
