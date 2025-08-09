package infra

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/types"
)

// DestroyOptions contains the options for the destroy command
type DestroyOptions struct {
	Files           []string
	OutputFormat    string
	Verbose         bool
	NoColor         bool
	StrictEnv       bool
	ProjectID       string
	Timeout         time.Duration
	AutoApprove     bool
	Force           bool
	DryRun          bool
	DeleteSnapshots bool
	TargetResource  string
	DiscoveryOnly   bool
}

// NewDestroyCmd creates the destroy subcommand
func NewDestroyCmd() *cobra.Command {
	opts := &DestroyOptions{}

	cmd := &cobra.Command{
		Use:   "destroy",
		Short: "Delete Atlas resources defined in configuration files or discovered in projects",
		Long: `Delete Atlas resources defined in configuration files or all discovered resources in a project.

⚠️  WARNING: This command will permanently delete Atlas resources including clusters,
database users, and network access lists. This action is irreversible!

The command supports two modes:
- Configuration-based (default): Destroys only resources defined in YAML files AND existing in Atlas
- Discovery-only (--discovery-only): Destroys ALL resources discovered in the specified Atlas project

The command will show what resources will be deleted and require confirmation
unless --auto-approve is used.`,
		Example: `  # Show what would be destroyed (dry run)
  matlas infra destroy -f config.yaml --dry-run

  # Destroy resources with confirmation
  matlas infra destroy -f config.yaml

  # Destroy resources without confirmation (dangerous!)
  matlas infra destroy -f config.yaml --auto-approve

  # Destroy only specific resource type
  matlas infra destroy -f config.yaml --target clusters

  # Force destroy even with dependencies
  matlas infra destroy -f config.yaml --force
  
  # Destroy all discovered resources (ignore config files)
  matlas infra destroy --discovery-only --project-id PROJECT_ID`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support positional arguments as files if no --file flag provided
			if len(opts.Files) == 0 && len(args) > 0 {
				opts.Files = args
			}
			return runDestroy(cmd, opts)
		},
	}

	// File input flags
	cmd.Flags().StringSliceVarP(&opts.Files, "file", "f", []string{}, "Configuration files defining resources to destroy")

	// Output flags
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "table", "Output format: table, json, yaml, summary")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")

	// Destroy behavior flags
	cmd.Flags().BoolVar(&opts.AutoApprove, "auto-approve", false, "Skip interactive approval prompts (DANGEROUS)")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Force deletion even with dependencies or protection")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be destroyed without actually deleting")
	cmd.Flags().BoolVar(&opts.DeleteSnapshots, "delete-snapshots", false, "Also delete any cluster snapshots")
	cmd.Flags().StringVar(&opts.TargetResource, "target", "", "Only destroy specific resource type: clusters, users, network-access")
	cmd.Flags().BoolVar(&opts.DiscoveryOnly, "discovery-only", false, "Destroy all discovered resources, regardless of configuration files")

	// Configuration flags
	cmd.Flags().BoolVar(&opts.StrictEnv, "strict-env", false, "Fail on undefined environment variables")
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Atlas project ID (overrides config)")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 30*time.Minute, "Timeout for destroy operations")

	return cmd
}

func runDestroy(cmd *cobra.Command, opts *DestroyOptions) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	// Validate options
	if err := validateDestroyOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Show warning for destructive operation
	if !opts.DryRun {
		fmt.Printf("⚠️  WARNING: This will permanently delete Atlas resources!\n")
		fmt.Printf("This action cannot be undone.\n\n")
	}

	// Expand file patterns (skip if discovery-only mode)
	var files []string
	if !opts.DiscoveryOnly {
		var err error
		files, err = expandFilePatterns(opts.Files)
		if err != nil {
			return fmt.Errorf("failed to expand file patterns: %w", err)
		}
	}

	// Initialize services
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	services, err := initializeServices(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Load configurations (skip if discovery-only mode)
	var configs []*apply.LoadResult
	if !opts.DiscoveryOnly {
		configs, err = loadConfigurations(files, &ApplyOptions{
			StrictEnv: opts.StrictEnv,
			Verbose:   opts.Verbose,
		})
		if err != nil {
			return fmt.Errorf("failed to load configurations: %w", err)
		}
	}

	// Create destroy plan
	destroyPlan, err := createDestroyPlan(ctx, configs, services, cfg, opts)
	if err != nil {
		return fmt.Errorf("failed to create destroy plan: %w", err)
	}

	// Handle dry-run mode
	if opts.DryRun {
		return displayDestroyPlan(destroyPlan, opts)
	}

	// Get approval unless auto-approve is set
	if !opts.AutoApprove {
		if err := getDestroyApproval(destroyPlan, opts); err != nil {
			return err
		}
	}

	// Execute destroy plan
	return executeDestroyPlan(ctx, destroyPlan, services, opts)
}

func createDestroyPlan(ctx context.Context, configs []*apply.LoadResult, services *ServiceClients, cfg *config.Config, opts *DestroyOptions) (*apply.Plan, error) {
	if opts.Verbose {
		fmt.Println("Creating destroy plan...")
	}

	// Create Atlas client for discovery
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Atlas client for discovery: %w", err)
	}

	// Initialize discovery service
	discoveryService := apply.NewAtlasStateDiscovery(atlasClient)

	// Discover current state
	var resolvedProjectID string

	if opts.DiscoveryOnly {
		// In discovery-only mode, use project ID directly
		resolvedProjectID = opts.ProjectID
	} else {
		// In configuration-based mode, resolve from configs
		projectNameOrID := getProjectID(configs, &ApplyOptions{ProjectID: opts.ProjectID})

		// Extract organization ID from configuration for project resolution
		orgID := getOrganizationID(configs)

		// Resolve project name to project ID
		resolvedProjectID = projectNameOrID
		if projectNameOrID != "" {
			resolvedID, err := resolveProjectID(ctx, projectNameOrID, services.ProjectsService, orgID)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve project ID for '%s': %w", projectNameOrID, err)
			}
			resolvedProjectID = resolvedID
		}
	}

	currentState, err := discoveryService.DiscoverProject(ctx, resolvedProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to discover current state: %w", err)
	}

	// Build desired state from configurations (unless discovery-only mode)
	var desiredState *apply.ProjectState

	if opts.DiscoveryOnly {
		// In discovery-only mode, use current state as desired state to destroy everything discovered
		desiredState = currentState
	} else {
		var err error
		desiredState, err = buildDesiredState(configs)
		if err != nil {
			return nil, fmt.Errorf("failed to build desired state: %w", err)
		}

		// Filter desired state based on target resource if specified
		if opts.TargetResource != "" {
			desiredState = filterDesiredStateByTarget(desiredState, opts.TargetResource)
		}
	}

	// Create destroy operations
	var operations []apply.PlannedOperation

	if opts.DiscoveryOnly {
		// In discovery-only mode, destroy ALL discovered resources

		// Add cluster destroy operations
		for _, cluster := range currentState.Clusters {
			op := apply.PlannedOperation{
				Operation: apply.Operation{
					Type:         apply.OperationDelete,
					ResourceType: types.KindCluster,
					ResourceName: cluster.Metadata.Name,
					Current:      &cluster,
					Impact: &apply.OperationImpact{
						IsDestructive:     true,
						RequiresDowntime:  true,
						EstimatedDuration: 15 * time.Minute,
						RiskLevel:         apply.RiskLevelHigh,
						Warnings:          []string{"This will permanently delete the cluster and all data"},
					},
				},
				ID:       fmt.Sprintf("destroy-cluster-%s", cluster.Metadata.Name),
				Priority: 200,
				Status:   apply.OperationStatusPending,
			}
			operations = append(operations, op)
		}

		// Add user destroy operations
		for _, user := range currentState.DatabaseUsers {
			op := apply.PlannedOperation{
				Operation: apply.Operation{
					Type:         apply.OperationDelete,
					ResourceType: types.KindDatabaseUser,
					ResourceName: user.Spec.Username,
					Current:      &user,
					Impact: &apply.OperationImpact{
						IsDestructive:     true,
						RequiresDowntime:  false,
						EstimatedDuration: 30 * time.Second,
						RiskLevel:         apply.RiskLevelMedium,
						Warnings:          []string{"This will remove database access for the user"},
					},
				},
				ID:       fmt.Sprintf("destroy-user-%s", user.Spec.Username),
				Priority: 100,
				Status:   apply.OperationStatusPending,
			}
			operations = append(operations, op)
		}

		// Add network access destroy operations
		for _, access := range currentState.NetworkAccess {
			var accessIdentifier string
			if access.Spec.IPAddress != "" {
				accessIdentifier = access.Spec.IPAddress
			} else if access.Spec.CIDR != "" {
				accessIdentifier = access.Spec.CIDR
			} else if access.Spec.AWSSecurityGroup != "" {
				accessIdentifier = access.Spec.AWSSecurityGroup
			}

			if accessIdentifier != "" {
				op := apply.PlannedOperation{
					Operation: apply.Operation{
						Type:         apply.OperationDelete,
						ResourceType: types.KindNetworkAccess,
						ResourceName: accessIdentifier,
						Current:      &access,
						Impact: &apply.OperationImpact{
							IsDestructive:     false,
							RequiresDowntime:  false,
							EstimatedDuration: 15 * time.Second,
							RiskLevel:         apply.RiskLevelLow,
							Warnings:          []string{"This will remove network access from this source"},
						},
					},
					ID:       fmt.Sprintf("destroy-access-%s", accessIdentifier),
					Priority: 150,
					Status:   apply.OperationStatusPending,
				}
				operations = append(operations, op)
			}
		}
	} else {
		// In configuration-based mode, only destroy resources that exist in BOTH desired state AND current state

		// Add cluster destroy operations
		for _, cluster := range desiredState.Clusters {
			clusterName := cluster.Metadata.Name
			var currentCluster *types.ClusterManifest
			for _, current := range currentState.Clusters {
				if current.Metadata.Name == clusterName {
					currentCluster = &current
					break
				}
			}
			if currentCluster != nil {
				op := apply.PlannedOperation{
					Operation: apply.Operation{
						Type:         apply.OperationDelete,
						ResourceType: types.KindCluster,
						ResourceName: cluster.Metadata.Name,
						Current:      currentCluster,
						Impact: &apply.OperationImpact{
							IsDestructive:     true,
							RequiresDowntime:  true,
							EstimatedDuration: 15 * time.Minute,
							RiskLevel:         apply.RiskLevelHigh,
							Warnings:          []string{"This will permanently delete the cluster and all data"},
						},
					},
					ID:       fmt.Sprintf("destroy-cluster-%s", cluster.Metadata.Name),
					Priority: 200,
					Status:   apply.OperationStatusPending,
				}
				operations = append(operations, op)
			}
		}

		// Add user destroy operations
		for _, user := range desiredState.DatabaseUsers {
			userName := user.Spec.Username
			authDB := user.Spec.AuthDatabase
			if authDB == "" {
				authDB = "admin"
			}

			var currentUser *types.DatabaseUserManifest
			for _, current := range currentState.DatabaseUsers {
				currentAuthDB := current.Spec.AuthDatabase
				if currentAuthDB == "" {
					currentAuthDB = "admin"
				}
				if current.Spec.Username == userName && currentAuthDB == authDB {
					currentUser = &current
					break
				}
			}
			if currentUser != nil {
				op := apply.PlannedOperation{
					Operation: apply.Operation{
						Type:         apply.OperationDelete,
						ResourceType: types.KindDatabaseUser,
						ResourceName: user.Spec.Username,
						Current:      currentUser,
						Impact: &apply.OperationImpact{
							IsDestructive:     true,
							RequiresDowntime:  false,
							EstimatedDuration: 30 * time.Second,
							RiskLevel:         apply.RiskLevelMedium,
							Warnings:          []string{"This will remove database access for the user"},
						},
					},
					ID:       fmt.Sprintf("destroy-user-%s", user.Spec.Username),
					Priority: 100,
					Status:   apply.OperationStatusPending,
				}
				operations = append(operations, op)
			}
		}

		// Add network access destroy operations
		for _, access := range desiredState.NetworkAccess {
			var accessIdentifier, currentIdentifier string

			// Get identifier from desired state
			if access.Spec.IPAddress != "" {
				accessIdentifier = access.Spec.IPAddress
			} else if access.Spec.CIDR != "" {
				accessIdentifier = access.Spec.CIDR
			} else if access.Spec.AWSSecurityGroup != "" {
				accessIdentifier = access.Spec.AWSSecurityGroup
			}

			var currentAccess *types.NetworkAccessManifest
			for _, current := range currentState.NetworkAccess {
				// Get identifier from current state
				if current.Spec.IPAddress != "" {
					currentIdentifier = current.Spec.IPAddress
				} else if current.Spec.CIDR != "" {
					currentIdentifier = current.Spec.CIDR
				} else if current.Spec.AWSSecurityGroup != "" {
					currentIdentifier = current.Spec.AWSSecurityGroup
				}

				if currentIdentifier == accessIdentifier {
					currentAccess = &current
					break
				}
			}
			if currentAccess != nil {
				op := apply.PlannedOperation{
					Operation: apply.Operation{
						Type:         apply.OperationDelete,
						ResourceType: types.KindNetworkAccess,
						ResourceName: accessIdentifier,
						Current:      currentAccess,
						Impact: &apply.OperationImpact{
							IsDestructive:     false,
							RequiresDowntime:  false,
							EstimatedDuration: 15 * time.Second,
							RiskLevel:         apply.RiskLevelLow,
							Warnings:          []string{"This will remove network access from this source"},
						},
					},
					ID:       fmt.Sprintf("destroy-access-%s", accessIdentifier),
					Priority: 150,
					Status:   apply.OperationStatusPending,
				}
				operations = append(operations, op)
			}
		}
	}

	// Set up dependencies (delete users and network access before clusters)
	for i := range operations {
		if operations[i].ResourceType == types.KindCluster {
			// Find user and network access operations that should be deleted before clusters
			for j := range operations {
				if operations[j].ResourceType == types.KindDatabaseUser || operations[j].ResourceType == types.KindNetworkAccess {
					operations[i].Dependencies = append(operations[i].Dependencies, operations[j].ID)
				}
			}
		}
	}

	// Create plan summary
	summary := apply.PlanSummary{
		TotalOperations:       len(operations),
		OperationsByType:      map[apply.OperationType]int{apply.OperationDelete: len(operations)},
		DestructiveOperations: len(operations),
		RequiresApproval:      true,
		HighestRiskLevel:      apply.RiskLevelHigh,
	}

	// Calculate estimated duration
	var totalDuration time.Duration
	for _, op := range operations {
		totalDuration += op.Impact.EstimatedDuration
	}
	summary.EstimatedDuration = totalDuration

	plan := &apply.Plan{
		ID:          fmt.Sprintf("destroy-plan-%d", time.Now().Unix()),
		ProjectID:   resolvedProjectID,
		CreatedAt:   time.Now(),
		Operations:  operations,
		Summary:     summary,
		Status:      apply.PlanStatusDraft,
		Config:      apply.PlanConfig{RequireApproval: true},
		Description: "Destroy plan for Atlas resources",
	}

	return plan, nil
}

func filterDesiredStateByTarget(state *apply.ProjectState, target string) *apply.ProjectState {
	filtered := &apply.ProjectState{
		Clusters:      []types.ClusterManifest{},
		DatabaseUsers: []types.DatabaseUserManifest{},
		NetworkAccess: []types.NetworkAccessManifest{},
	}

	switch strings.ToLower(target) {
	case "clusters", "cluster":
		filtered.Clusters = state.Clusters
	case "users", "user", "database-users":
		filtered.DatabaseUsers = state.DatabaseUsers
	case "network-access", "network":
		filtered.NetworkAccess = state.NetworkAccess
	default:
		// Invalid target, return empty state
		return filtered
	}

	return filtered
}

func displayDestroyPlan(plan *apply.Plan, opts *DestroyOptions) error {
	fmt.Printf("Destroy Plan Preview\n")
	fmt.Printf("===================\n\n")

	if len(plan.Operations) == 0 {
		fmt.Println("No resources to destroy - no matching resources found in current state")
		return nil
	}

	fmt.Printf("The following resources will be PERMANENTLY DELETED:\n\n")

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		formatter := output.NewFormatter(config.OutputJSON, os.Stdout)
		return formatter.Format(plan)
	case "yaml":
		formatter := output.NewFormatter(config.OutputYAML, os.Stdout)
		return formatter.Format(plan)
	case "summary":
		return displayDestroySummary(plan, opts)
	default: // table
		return displayDestroyTable(plan, opts)
	}
}

func displayDestroySummary(plan *apply.Plan, opts *DestroyOptions) error {
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total resources to destroy: %d\n", plan.Summary.TotalOperations)
	fmt.Printf("  Estimated duration: %s\n", plan.Summary.EstimatedDuration)
	fmt.Printf("  Highest risk level: %s\n", plan.Summary.HighestRiskLevel)
	fmt.Printf("  Destructive operations: %d\n", plan.Summary.DestructiveOperations)

	return nil
}

func displayDestroyTable(plan *apply.Plan, opts *DestroyOptions) error {
	fmt.Printf("%-20s %-15s %-10s %-12s %s\n", "Resource Type", "Resource Name", "Risk", "Duration", "Warnings")
	fmt.Printf("%s\n", strings.Repeat("-", 80))

	for _, op := range plan.Operations {
		riskColor := ""
		if !opts.NoColor {
			switch op.Impact.RiskLevel {
			case apply.RiskLevelHigh, apply.RiskLevelCritical:
				riskColor = "\033[31m" // Red
			case apply.RiskLevelMedium:
				riskColor = "\033[33m" // Yellow
			case apply.RiskLevelLow:
				riskColor = "\033[32m" // Green
			}
		}
		resetColor := ""
		if riskColor != "" {
			resetColor = "\033[0m"
		}

		warnings := "None"
		if len(op.Impact.Warnings) > 0 {
			warnings = strings.Join(op.Impact.Warnings, "; ")
			if len(warnings) > 40 {
				warnings = warnings[:37] + "..."
			}
		}

		fmt.Printf("%-20s %-15s %s%-10s%s %-12s %s\n",
			op.ResourceType,
			op.ResourceName,
			riskColor,
			op.Impact.RiskLevel,
			resetColor,
			op.Impact.EstimatedDuration,
			warnings)
	}

	fmt.Printf("\n⚠️  WARNING: This operation will permanently delete %d resources!\n", len(plan.Operations))
	return nil
}

func getDestroyApproval(plan *apply.Plan, opts *DestroyOptions) error {
	fmt.Printf("\nConfirmation Required\n")
	fmt.Printf("====================\n\n")
	fmt.Printf("You are about to destroy %d Atlas resources.\n", len(plan.Operations))
	fmt.Printf("This action is IRREVERSIBLE and will permanently delete:\n")

	// Count resources by type
	resourceCounts := make(map[string]int)
	for _, op := range plan.Operations {
		resourceCounts[string(op.ResourceType)]++
	}

	for resourceType, count := range resourceCounts {
		fmt.Printf("  - %d %s(s)\n", count, resourceType)
	}

	fmt.Printf("\nEstimated time: %s\n", plan.Summary.EstimatedDuration)
	fmt.Printf("\n")

	// Require explicit confirmation for destructive operations
	fmt.Printf("Type 'destroy' to confirm destruction: ")
	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "destroy" {
		return fmt.Errorf("destroy cancelled - confirmation text did not match 'destroy'")
	}

	// Additional confirmation for high-risk operations
	hasHighRisk := false
	for _, op := range plan.Operations {
		if op.Impact.RiskLevel == apply.RiskLevelHigh || op.Impact.RiskLevel == apply.RiskLevelCritical {
			hasHighRisk = true
			break
		}
	}

	if hasHighRisk && !opts.Force {
		fmt.Printf("\nHigh-risk operations detected. Are you absolutely sure? (yes/no): ")
		var finalConfirmation string
		fmt.Scanln(&finalConfirmation)

		if strings.ToLower(finalConfirmation) != "yes" {
			return fmt.Errorf("destroy cancelled by user")
		}
	}

	return nil
}

func executeDestroyPlan(ctx context.Context, plan *apply.Plan, services *ServiceClients, opts *DestroyOptions) error {
	fmt.Printf("Executing destroy plan...\n")

	// Create enhanced executor
	enhancedExecutor := apply.NewEnhancedExecutor(
		services.ClustersService,
		services.UsersService,
		services.NetworkAccessService,
		services.ProjectsService,
		services.DatabaseService,
		apply.DefaultEnhancedExecutorConfig(),
	)

	// Execute the plan
	result, err := enhancedExecutor.Execute(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to execute destroy plan: %w", err)
	}

	// Display results
	fmt.Printf("\nDestroy operation completed in %s\n", result.Duration)
	fmt.Printf("Resources destroyed: %d completed, %d failed, %d skipped\n",
		result.Summary.CompletedOperations,
		result.Summary.FailedOperations,
		result.Summary.SkippedOperations)

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors encountered:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err.Message)
		}
		return fmt.Errorf("destroy completed with errors")
	}

	// Note: ExecutionResult doesn't have Warnings field, removing this section

	fmt.Printf("\n✅ Destroy operation completed successfully\n")
	return nil
}

func validateDestroyOptions(opts *DestroyOptions) error {
	if len(opts.Files) == 0 && !opts.DiscoveryOnly {
		return fmt.Errorf("at least one configuration file must be specified with --file (or use --discovery-only)")
	}

	if opts.DiscoveryOnly && opts.ProjectID == "" {
		return fmt.Errorf("--project-id is required when using --discovery-only")
	}

	// Validate output format
	validOutputFormats := []string{"table", "json", "yaml", "summary"}
	if !contains(validOutputFormats, strings.ToLower(opts.OutputFormat)) {
		return fmt.Errorf("invalid output format: %s (valid options: %s)", opts.OutputFormat, strings.Join(validOutputFormats, ", "))
	}

	// Validate target resource
	if opts.TargetResource != "" {
		validTargets := []string{"clusters", "cluster", "users", "user", "database-users", "network-access", "network"}
		if !contains(validTargets, strings.ToLower(opts.TargetResource)) {
			return fmt.Errorf("invalid target resource: %s (valid options: clusters, users, network-access)", opts.TargetResource)
		}
	}

	// Validate timeout
	if opts.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}
