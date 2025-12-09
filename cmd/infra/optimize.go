package infra

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/internal/apply/dag"
	"github.com/teabranch/matlas-cli/internal/config"
)

// OptimizeOptions contains the options for the optimize command
type OptimizeOptions struct {
	Files        []string
	OutputFormat string
	OutputFile   string
	Verbose      bool
	StrictEnv    bool
	ProjectID    string
	Timeout      time.Duration
	Strategy     string
}

// NewOptimizeCmd creates the optimize subcommand
func NewOptimizeCmd() *cobra.Command {
	opts := &OptimizeOptions{}

	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Suggest optimizations for execution plan",
		Long: `Analyze the execution plan and suggest optimizations to:
- Reduce total execution time
- Improve parallelization
- Minimize resource usage
- Reduce risk

The command will analyze the dependency graph and provide actionable recommendations.`,
		Example: `  # Get optimization suggestions
  matlas infra optimize -f config.yaml

  # Optimize for speed
  matlas infra optimize -f config.yaml --strategy speed

  # Optimize for reliability
  matlas infra optimize -f config.yaml --strategy reliability

  # Export suggestions as JSON
  matlas infra optimize -f config.yaml -o json --output-file optimizations.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support positional arguments as files if no --file flag provided
			if len(opts.Files) == 0 && len(args) > 0 {
				opts.Files = args
			}
			return runOptimize(cmd, opts)
		},
	}

	// File input flags
	cmd.Flags().StringSliceVarP(&opts.Files, "file", "f", []string{}, "Configuration files to analyze (supports glob patterns)")

	// Output flags
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "text", "Output format: text, markdown, json")
	cmd.Flags().StringVar(&opts.OutputFile, "output-file", "", "Save suggestions to file")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")

	// Optimization options
	cmd.Flags().StringVar(&opts.Strategy, "strategy", "balanced", "Optimization strategy: speed, cost, reliability, balanced")
	cmd.Flags().BoolVar(&opts.StrictEnv, "strict-env", false, "Fail on undefined environment variables")
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Atlas project ID (overrides config)")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 5*time.Minute, "Timeout for optimization")

	return cmd
}

func runOptimize(cmd *cobra.Command, opts *OptimizeOptions) error {
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
		fmt.Printf("Analyzing %d operations for optimizations...\n", len(plan.Operations))
	}

	// Build DAG from plan
	graph := buildGraphFromPlan(plan)

	// Determine optimization strategy
	var strategy dag.OptimizationStrategy
	switch opts.Strategy {
	case "speed":
		strategy = dag.OptimizeForSpeed
	case "cost":
		strategy = dag.OptimizeForCost
	case "reliability":
		strategy = dag.OptimizeForReliability
	case "balanced", "balance":
		strategy = dag.OptimizeForBalance
	default:
		return fmt.Errorf("unsupported strategy: %s (use speed, cost, reliability, or balanced)", opts.Strategy)
	}

	// Create optimizer
	config := dag.ScheduleConfig{
		Strategy:       dag.StrategyGreedy,
		MaxParallelOps: 5,
	}
	optimizer := dag.NewOptimizer(strategy, config)

	// Apply optimization
	optimizedGraph, err := optimizer.Optimize(cmd.Context(), graph)
	if err != nil {
		return fmt.Errorf("failed to optimize graph: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("Applied %s optimization strategy\n", opts.Strategy)
	}

	// Get optimization suggestions
	suggestions := optimizer.SuggestOptimizations(graph)

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
	report, err := reporter.GenerateOptimizationReport(suggestions)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Add summary of changes if verbose
	if opts.Verbose && reportFormat == dag.ReportFormatText {
		originalEdges := graph.EdgeCount()
		optimizedEdges := optimizedGraph.EdgeCount()

		fmt.Printf("\nOptimization Summary:\n")
		fmt.Printf("  Strategy: %s\n", opts.Strategy)
		fmt.Printf("  Operations: %d\n", graph.NodeCount())
		fmt.Printf("  Dependencies: %d → %d (%+d)\n", originalEdges, optimizedEdges, optimizedEdges-originalEdges)
		fmt.Printf("\n")
	}

	// Save to file or print to stdout
	if opts.OutputFile != "" {
		if err := os.WriteFile(opts.OutputFile, []byte(report), 0644); err != nil {
			return fmt.Errorf("failed to write report to file: %w", err)
		}
		fmt.Printf("Optimization report saved to %s\n", opts.OutputFile)
	} else {
		fmt.Print(report)
	}

	return nil
}
