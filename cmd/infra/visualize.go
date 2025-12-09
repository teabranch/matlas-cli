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

// VisualizeOptions contains the options for the visualize command
type VisualizeOptions struct {
	Files                 []string
	OutputFormat          string
	OutputFile            string
	Verbose               bool
	StrictEnv             bool
	ProjectID             string
	Timeout               time.Duration
	ShowDurations         bool
	ShowRisk              bool
	HighlightCriticalPath bool
	ShowLevels            bool
	CompactMode           bool
	ColorScheme           string
}

// NewVisualizeCmd creates the visualize subcommand
func NewVisualizeCmd() *cobra.Command {
	opts := &VisualizeOptions{}

	cmd := &cobra.Command{
		Use:   "visualize",
		Short: "Visualize dependency graph",
		Long: `Visualize the dependency graph for a configuration in various formats:
- DOT: Graphviz format (render with 'dot -Tpng graph.dot -o graph.png')
- Mermaid: Mermaid diagram format (for markdown/documentation)
- ASCII: Terminal-friendly ASCII art
- JSON: Structured JSON data`,
		Example: `  # Visualize as ASCII art in terminal
  matlas infra visualize -f config.yaml

  # Export as Graphviz DOT format
  matlas infra visualize -f config.yaml -o dot --output-file graph.dot

  # Export as Mermaid diagram
  matlas infra visualize -f config.yaml -o mermaid --output-file graph.mmd

  # Visualize with critical path highlighted
  matlas infra visualize -f config.yaml --highlight-critical-path

  # Compact ASCII visualization
  matlas infra visualize -f config.yaml -o ascii --compact`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support positional arguments as files if no --file flag provided
			if len(opts.Files) == 0 && len(args) > 0 {
				opts.Files = args
			}
			return runVisualize(cmd, opts)
		},
	}

	// File input flags
	cmd.Flags().StringSliceVarP(&opts.Files, "file", "f", []string{}, "Configuration files to visualize (supports glob patterns)")

	// Output flags
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "ascii", "Output format: dot, mermaid, ascii, json")
	cmd.Flags().StringVar(&opts.OutputFile, "output-file", "", "Save visualization to file")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")

	// Visualization options
	cmd.Flags().BoolVar(&opts.ShowDurations, "show-durations", true, "Show estimated durations")
	cmd.Flags().BoolVar(&opts.ShowRisk, "show-risk", true, "Show risk levels")
	cmd.Flags().BoolVar(&opts.HighlightCriticalPath, "highlight-critical-path", false, "Highlight critical path")
	cmd.Flags().BoolVar(&opts.ShowLevels, "show-levels", false, "Show dependency levels")
	cmd.Flags().BoolVar(&opts.CompactMode, "compact", false, "Use compact mode (less detail)")
	cmd.Flags().StringVar(&opts.ColorScheme, "color-scheme", "default", "Color scheme: default, monochrome, vibrant")

	// Other options
	cmd.Flags().BoolVar(&opts.StrictEnv, "strict-env", false, "Fail on undefined environment variables")
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Atlas project ID (overrides config)")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 5*time.Minute, "Timeout for visualization")

	return cmd
}

func runVisualize(cmd *cobra.Command, opts *VisualizeOptions) error {
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
		fmt.Printf("Visualizing %d operations...\n", len(plan.Operations))
	}

	// Build DAG from plan
	graph := buildGraphFromPlan(plan)

	// Determine visualization format
	vizFormat := dag.FormatASCII
	switch opts.OutputFormat {
	case "dot", "graphviz":
		vizFormat = dag.FormatDOT
	case "mermaid", "mmd":
		vizFormat = dag.FormatMermaid
	case "ascii", "text":
		vizFormat = dag.FormatASCII
	case "json":
		vizFormat = dag.FormatJSON
	default:
		return fmt.Errorf("unsupported output format: %s (use dot, mermaid, ascii, or json)", opts.OutputFormat)
	}

	// Configure visualizer
	vizOptions := dag.VisualizerOptions{
		ShowDurations:         opts.ShowDurations,
		ShowRisk:              opts.ShowRisk,
		HighlightCriticalPath: opts.HighlightCriticalPath,
		ShowLevels:            opts.ShowLevels,
		CompactMode:           opts.CompactMode,
		ColorScheme:           opts.ColorScheme,
	}

	// Create visualizer
	visualizer := dag.NewVisualizer(vizFormat, vizOptions)

	// Generate visualization
	visualization, err := visualizer.Visualize(graph)
	if err != nil {
		return fmt.Errorf("failed to generate visualization: %w", err)
	}

	// Save to file or print to stdout
	if opts.OutputFile != "" {
		if err := os.WriteFile(opts.OutputFile, []byte(visualization), 0644); err != nil {
			return fmt.Errorf("failed to write visualization to file: %w", err)
		}
		fmt.Printf("Visualization saved to %s\n", opts.OutputFile)

		// Provide helpful hint for DOT format
		if vizFormat == dag.FormatDOT {
			fmt.Printf("\nTo render the graph as an image, run:\n")
			fmt.Printf("  dot -Tpng %s -o graph.png\n", opts.OutputFile)
		}
	} else {
		fmt.Print(visualization)
	}

	return nil
}
