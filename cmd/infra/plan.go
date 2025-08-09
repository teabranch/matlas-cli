package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
)

// PlanOptions contains the options for the plan command
type PlanOptions struct {
	Files            []string
	OutputFormat     string
	OutputFile       string
	Verbose          bool
	NoColor          bool
	StrictEnv        bool
	ProjectID        string
	Timeout          time.Duration
	PlanMode         string
	PreserveExisting bool
}

// NewPlanCmd creates the plan subcommand
func NewPlanCmd() *cobra.Command {
	opts := &PlanOptions{}

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate and save execution plans",
		Long: `Generate execution plans for applying configuration changes without actually applying them.

This command creates detailed execution plans that show what operations will be performed,
their dependencies, estimated durations, and risk levels. Plans can be saved for later execution.`,
		Example: `  # Generate a plan from configuration
  matlas infra plan -f config.yaml

  # Generate and save plan to file
  matlas infra plan -f config.yaml --output-file plan.json

  # Generate plan with detailed output
  matlas infra plan -f config.yaml --plan-mode detailed --verbose

  # Generate plan for specific project
  matlas infra plan -f config.yaml --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support positional arguments as files if no --file flag provided
			if len(opts.Files) == 0 && len(args) > 0 {
				opts.Files = args
			}
			return runPlan(cmd, opts)
		},
	}

	// File input flags
	cmd.Flags().StringSliceVarP(&opts.Files, "file", "f", []string{}, "Configuration files to plan (supports glob patterns)")

	// Output flags
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "table", "Output format: table, json, yaml, summary")
	// Alias --format to --output for ergonomics (binds to same variable)
	cmd.Flags().StringVar(&opts.OutputFormat, "format", "table", "Output format (alias for --output): table, json, yaml, summary")
	cmd.Flags().StringVar(&opts.OutputFile, "output-file", "", "Save plan to file (format determined by extension)")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")

	// Planning options
	cmd.Flags().StringVar(&opts.PlanMode, "plan-mode", "standard", "Planning mode: quick, standard, detailed")
	cmd.Flags().BoolVar(&opts.StrictEnv, "strict-env", false, "Fail on undefined environment variables")
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Atlas project ID (overrides config)")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 10*time.Minute, "Timeout for plan generation")
	cmd.Flags().BoolVar(&opts.PreserveExisting, "preserve-existing", false, "Only plan additions and updates, exclude deletions")

	return cmd
}

func runPlan(cmd *cobra.Command, opts *PlanOptions) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	// Validate options
	if err := validatePlanOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Expand file patterns
	files, err := expandFilePatterns(opts.Files)
	if err != nil {
		return fmt.Errorf("failed to expand file patterns: %w", err)
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

	// Load configurations
	configs, err := loadConfigurations(files, &ApplyOptions{
		StrictEnv: opts.StrictEnv,
		Verbose:   opts.Verbose,
	})
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	// Generate execution plan
	plan, err := generateExecutionPlan(ctx, configs, services, cfg, opts)
	if err != nil {
		return fmt.Errorf("failed to generate execution plan: %w", err)
	}

	// Save plan to file if specified
	if opts.OutputFile != "" {
		if err := savePlanToFile(plan, opts.OutputFile); err != nil {
			return fmt.Errorf("failed to save plan to file: %w", err)
		}
		// Standardize with formatter for user-visible confirmation
		formatter := output.NewFormatter(config.OutputText, os.Stdout)
		_ = formatter.Format(output.TableData{Headers: []string{"Info"}, Rows: [][]string{{"Plan saved to " + opts.OutputFile}}})
	}

	// Display plan
	return displayPlan(plan, opts)
}

func generateExecutionPlan(ctx context.Context, configs []*apply.LoadResult, services *ServiceClients, cfg *config.Config, opts *PlanOptions) (*apply.Plan, error) {
	if opts.Verbose {
		fmt.Println("Generating execution plan...")
	}

	// Create Atlas client for discovery
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Atlas client for discovery: %w", err)
	}

	// Initialize apply engine components
	discoveryService := apply.NewAtlasStateDiscovery(atlasClient)

	diffEngine := apply.NewDiffEngine()
	diffEngine.PreserveExisting = opts.PreserveExisting

	planOptimizer := apply.NewPlanOptimizer()

	// Get project name or ID and resolve to project ID
	projectNameOrID := getProjectID(configs, &ApplyOptions{ProjectID: opts.ProjectID})
	orgID := getOrganizationID(configs)

	resolvedProjectID, err := resolveProjectID(ctx, projectNameOrID, services.ProjectsService, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project ID for '%s': %w", projectNameOrID, err)
	}

	// Discover current state
	if opts.Verbose {
		fmt.Printf("Discovering current state for project %s (resolved from '%s')...\n", resolvedProjectID, projectNameOrID)
	}

	currentState, err := discoveryService.DiscoverProject(ctx, resolvedProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to discover current state: %w", err)
	}

	// Build desired state
	desiredState, err := buildDesiredState(configs)
	if err != nil {
		return nil, fmt.Errorf("failed to build desired state: %w", err)
	}

	// Compute diff
	if opts.Verbose {
		fmt.Println("Computing differences...")
	}

	diff, err := diffEngine.ComputeProjectDiff(desiredState, currentState)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	// Create execution plan
	planBuilder := apply.NewPlanBuilder(resolvedProjectID)
	planBuilder.AddOperations(diff.Operations)

	plan, err := planBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create execution plan: %w", err)
	}

	// Optimize plan
	if opts.Verbose {
		fmt.Println("Optimizing plan...")
	}

	optimizationResult, err := planOptimizer.OptimizePlan(plan)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize plan: %w", err)
	}

	return optimizationResult.OptimizedPlan, nil
}

func savePlanToFile(plan *apply.Plan, filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(plan, "", "  ")
	case ".yaml", ".yml":
		data, err = yaml.Marshal(plan)
	default:
		return fmt.Errorf("unsupported file format: %s (supported: .json, .yaml, .yml)", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func displayPlan(plan *apply.Plan, opts *PlanOptions) error {
	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		return displayPlanJSON(plan)
	case "yaml":
		return displayPlanYAML(plan)
	case "summary":
		return displayPlanSummary(plan, opts)
	default: // table
		return displayPlanTable(plan, opts)
	}
}

func displayPlanJSON(plan *apply.Plan) error {
	formatter := output.NewFormatter(config.OutputJSON, os.Stdout)
	return formatter.Format(plan)
}

func displayPlanYAML(plan *apply.Plan) error {
	formatter := output.NewFormatter(config.OutputYAML, os.Stdout)
	return formatter.Format(plan)
}

func displayPlanSummary(plan *apply.Plan, opts *PlanOptions) error {
	// Render summary using internal/output for consistency
	rows := [][]string{
		{"Plan ID", plan.ID},
		{"Project ID", plan.ProjectID},
		{"Created", plan.CreatedAt.Format(time.RFC3339)},
		{"Total Operations", fmt.Sprintf("%d", plan.Summary.TotalOperations)},
		{"Stages", fmt.Sprintf("%d", len(plan.Summary.OperationsByStage))},
		{"Estimated Duration", plan.Summary.EstimatedDuration.String()},
		{"Highest Risk Level", string(plan.Summary.HighestRiskLevel)},
		{"Parallelization Factor", fmt.Sprintf("%.2f", plan.Summary.ParallelizationFactor)},
		{"Destructive Operations", fmt.Sprintf("%d", plan.Summary.DestructiveOperations)},
	}
	if plan.Description != "" {
		rows = append([][]string{{"Description", plan.Description}}, rows...)
	}
	if plan.Summary.RequiresApproval {
		rows = append(rows, []string{"Requires Approval", "true"})
	}
	formatter := output.NewFormatter(config.OutputTable, os.Stdout)
	return formatter.Format(output.TableData{Headers: []string{"Field", "Value"}, Rows: rows})
}

func displayPlanTable(plan *apply.Plan, opts *PlanOptions) error {
	// Plan header using standard formatter
	header := output.TableData{Headers: []string{"Execution Plan", plan.ID}, Rows: [][]string{{"Project", plan.ProjectID}, {"Created", plan.CreatedAt.Format("2006-01-02 15:04:05")}}}
	_ = output.NewFormatter(config.OutputTable, os.Stdout).Format(header)

	// Group operations by stage
	stageToOperations := make(map[int][]apply.PlannedOperation)
	var stages []int
	for _, plannedOperation := range plan.Operations {
		if _, ok := stageToOperations[plannedOperation.Stage]; !ok {
			stages = append(stages, plannedOperation.Stage)
		}
		stageToOperations[plannedOperation.Stage] = append(stageToOperations[plannedOperation.Stage], plannedOperation)
	}

	// Sort stages for stable output
	sort.Ints(stages)

	// Render each stage as a table via formatter
	for _, stage := range stages {
		ops := stageToOperations[stage]
		fmt.Fprintf(os.Stdout, "\nStage %d (%d operations)\n\n", stage, len(ops))

		rows := make([][]string, 0, len(ops))
		for _, op := range ops {
			dependencies := strings.Join(op.Dependencies, ", ")
			if len(dependencies) > 60 {
				dependencies = dependencies[:57] + "..."
			}

			riskLevel := "N/A"
			duration := ""
			if op.Impact != nil {
				riskLevel = string(op.Impact.RiskLevel)
				if op.Impact.EstimatedDuration > 0 {
					duration = op.Impact.EstimatedDuration.String()
				}
			}

			rows = append(rows, []string{
				string(op.ResourceType),
				string(op.Type),
				op.ResourceName,
				riskLevel,
				duration,
				dependencies,
			})
		}

		table := output.TableData{
			Headers: []string{"Resource Type", "Operation", "Resource Name", "Risk", "Duration", "Dependencies"},
			Rows:    rows,
		}
		_ = output.NewFormatter(config.OutputTable, os.Stdout).Format(table)
	}

	// Optional verbose configuration block
	if opts.Verbose {
		cfgRows := [][]string{{"Require Approval", fmt.Sprintf("%t", plan.Config.RequireApproval)}}
		if plan.Config.MaxParallelOps > 0 {
			cfgRows = append(cfgRows, []string{"Max Parallel Ops", fmt.Sprintf("%d", plan.Config.MaxParallelOps)})
		}
		if plan.Config.DefaultTimeout > 0 {
			cfgRows = append(cfgRows, []string{"Default Timeout", plan.Config.DefaultTimeout.String()})
		}
		fmt.Fprintln(os.Stdout)
		_ = output.NewFormatter(config.OutputTable, os.Stdout).Format(output.TableData{Headers: []string{"Plan Configuration", "Value"}, Rows: cfgRows})
	}

	// Summary block via formatter
	sumRows := make([][]string, 0, len(plan.Summary.OperationsByType)+2)
	for opType, count := range plan.Summary.OperationsByType {
		sumRows = append(sumRows, []string{string(opType) + " operations", fmt.Sprintf("%d", count)})
	}
	sumRows = append(sumRows, []string{"Estimated Duration", plan.Summary.EstimatedDuration.String()})
	sumRows = append(sumRows, []string{"Highest Risk Level", string(plan.Summary.HighestRiskLevel)})
	if plan.Summary.DestructiveOperations > 0 {
		sumRows = append(sumRows, []string{"Destructive Operations", fmt.Sprintf("%d", plan.Summary.DestructiveOperations)})
	}
	fmt.Fprintln(os.Stdout)
	_ = output.NewFormatter(config.OutputTable, os.Stdout).Format(output.TableData{Headers: []string{"Summary", "Value"}, Rows: sumRows})

	return nil
}

func validatePlanOptions(opts *PlanOptions) error {
	if len(opts.Files) == 0 {
		return fmt.Errorf("at least one configuration file must be specified with --file")
	}

	// Validate output format
	validOutputFormats := []string{"table", "json", "yaml", "summary"}
	if !contains(validOutputFormats, strings.ToLower(opts.OutputFormat)) {
		return fmt.Errorf("invalid output format: %s (valid options: %s)", opts.OutputFormat, strings.Join(validOutputFormats, ", "))
	}

	// Validate plan mode
	validPlanModes := []string{"quick", "standard", "detailed"}
	if !contains(validPlanModes, strings.ToLower(opts.PlanMode)) {
		return fmt.Errorf("invalid plan mode: %s (valid options: %s)", opts.PlanMode, strings.Join(validPlanModes, ", "))
	}

	// Validate timeout
	if opts.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	// Validate output file extension if specified
	if opts.OutputFile != "" {
		ext := strings.ToLower(filepath.Ext(opts.OutputFile))
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			return fmt.Errorf("output file must have .json, .yaml, or .yml extension")
		}
	}

	return nil
}
