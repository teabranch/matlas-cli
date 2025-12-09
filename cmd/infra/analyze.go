package infra

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/apply/dag"
	"github.com/teabranch/matlas-cli/internal/config"
)

// AnalyzeOptions contains the options for the analyze command
type AnalyzeOptions struct {
	Files        []string
	OutputFormat string
	OutputFile   string
	Verbose      bool
	NoColor      bool
	StrictEnv    bool
	ProjectID    string
	Timeout      time.Duration
	ShowCycles   bool
	ShowRisk     bool
}

// NewAnalyzeCmd creates the analyze subcommand
func NewAnalyzeCmd() *cobra.Command {
	opts := &AnalyzeOptions{}

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze dependency graph and identify issues",
		Long: `Analyze the dependency graph for a configuration and identify:
- Critical path operations that determine total execution time
- Bottlenecks that block many other operations
- Cycles in dependencies (if any)
- Risk analysis for operations on critical path
- Parallelization opportunities`,
		Example: `  # Analyze dependencies in configuration
  matlas infra analyze -f config.yaml

  # Analyze with detailed risk analysis
  matlas infra analyze -f config.yaml --show-risk

  # Analyze and detect cycles
  matlas infra analyze -f config.yaml --show-cycles

	# Export analysis as JSON
  matlas infra analyze -f config.yaml --format json --output-file analysis.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support positional arguments as files if no --file flag provided
			if len(opts.Files) == 0 && len(args) > 0 {
				opts.Files = args
			}
			return runAnalyze(cmd, opts)
		},
	}

	// File input flags
	cmd.Flags().StringSliceVarP(&opts.Files, "file", "f", []string{}, "Configuration files to analyze (supports glob patterns)")

	// Output flags
	cmd.Flags().StringVar(&opts.OutputFormat, "format", "text", "Report format: text, markdown, json")
	cmd.Flags().StringVar(&opts.OutputFile, "output-file", "", "Save analysis to file")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")

	// Analysis options
	cmd.Flags().BoolVar(&opts.ShowCycles, "show-cycles", false, "Show dependency cycles (if any)")
	cmd.Flags().BoolVar(&opts.ShowRisk, "show-risk", false, "Show detailed risk analysis")
	cmd.Flags().BoolVar(&opts.StrictEnv, "strict-env", false, "Fail on undefined environment variables")
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Atlas project ID (overrides config)")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 5*time.Minute, "Timeout for analysis")

	return cmd
}

func runAnalyze(cmd *cobra.Command, opts *AnalyzeOptions) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	// Validate options
	if len(opts.Files) == 0 {
		return fmt.Errorf("no configuration files specified (use -f or provide files as arguments)")
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
	plan, err := generateExecutionPlan(ctx, configs, services, cfg, &PlanOptions{
		ProjectID: opts.ProjectID,
		Verbose:   opts.Verbose,
		StrictEnv: opts.StrictEnv,
	})
	if err != nil {
		return fmt.Errorf("failed to generate execution plan: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("Analyzing %d operations...\n", len(plan.Operations))
	}

	// Build DAG from plan
	graph := buildGraphFromPlan(plan)

	// Run analysis
	analyzer := dag.NewAnalyzer(graph)
	analysis, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	// Generate report
	reportFormat := dag.ReportFormatText
	switch opts.OutputFormat {
	case "text":
		reportFormat = dag.ReportFormatText
	case "markdown", "md":
		reportFormat = dag.ReportFormatMarkdown
	case "json":
		reportFormat = dag.ReportFormatJSON
	default:
		return fmt.Errorf("unsupported output format: %s (use text, markdown, or json)", opts.OutputFormat)
	}

	reporter := dag.NewReporter(reportFormat)
	report, err := reporter.GenerateDependencyReport(analysis)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Save to file or print to stdout
	if opts.OutputFile != "" {
		if err := os.WriteFile(opts.OutputFile, []byte(report), 0644); err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
		fmt.Printf("Analysis report saved to %s\n", opts.OutputFile)
	} else {
		fmt.Print(report)
	}

	return nil
}

// buildGraphFromPlan converts a Plan into a DAG Graph
func buildGraphFromPlan(plan *apply.Plan) *dag.Graph {
	graph := dag.NewGraph(dag.GraphMetadata{
		Name:      "Execution Plan",
		ProjectID: plan.ProjectID,
		CreatedAt: plan.CreatedAt,
	})

	// Add all operations as nodes
	for _, op := range plan.Operations {
		props := dag.NodeProperties{
			EstimatedDuration: 5 * time.Second,     // Default duration
			RiskLevel:         dag.RiskLevelMedium, // Default risk level
		}

		// Estimate duration based on operation type
		switch op.Type {
		case apply.OperationCreate:
			if op.ResourceType == "Cluster" {
				props.EstimatedDuration = 10 * time.Minute // Cluster creation is slow
			} else {
				props.EstimatedDuration = 30 * time.Second
			}
		case apply.OperationUpdate:
			props.EstimatedDuration = 1 * time.Minute
		case apply.OperationDelete:
			props.EstimatedDuration = 30 * time.Second
		}

		// Determine risk level
		switch op.Type {
		case apply.OperationDelete:
			props.RiskLevel = dag.RiskLevelHigh
			props.IsDestructive = true
		case apply.OperationUpdate:
			props.RiskLevel = dag.RiskLevelMedium
		case apply.OperationCreate:
			props.RiskLevel = dag.RiskLevelLow
		}

		node := &dag.Node{
			ID:           op.ID,
			Name:         op.ResourceName,
			ResourceType: op.ResourceType,
			Properties:   props,
		}
		graph.AddNode(node)
	}

	// Add dependencies as edges
	for _, op := range plan.Operations {
		for _, depID := range op.Dependencies {
			// Edge direction: From=dependent, To=dependency (op depends on depID)
			edge := &dag.Edge{
				From:   op.ID,
				To:     depID,
				Type:   dag.DependencyTypeHard,
				Weight: 1.0,
			}
			graph.AddEdge(edge)
		}
	}

	return graph
}
