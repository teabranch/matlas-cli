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
)

// DiffOptions contains the options for the diff command
type DiffOptions struct {
	Files            []string
	OutputFormat     string
	Verbose          bool
	NoColor          bool
	StrictEnv        bool
	ProjectID        string
	Timeout          time.Duration
	ShowContext      int
	Detailed         bool
	PreserveExisting bool
	NoTruncate       bool
}

// NewDiffCmd creates the diff subcommand
func NewDiffCmd() *cobra.Command {
	opts := &DiffOptions{}

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between current and desired state",
		Long: `Show differences between the current Atlas state and the desired state defined in configuration files.

This command displays what changes would be made without creating an execution plan or applying changes.
It's useful for quickly understanding what resources would be affected.`,
		Example: `  # Show differences for configuration
  matlas infra diff -f config.yaml

  # Show detailed differences with context
  matlas infra diff -f config.yaml --detailed --show-context 3

  # Show differences in JSON format
  matlas infra diff -f config.yaml --output json

  # Show differences with full resource names (no truncation)
  matlas infra diff -f config.yaml --no-truncate

  # Show differences for specific project
  matlas infra diff -f config.yaml --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support positional arguments as files if no --file flag provided
			if len(opts.Files) == 0 && len(args) > 0 {
				opts.Files = args
			}
			return runDiff(cmd, opts)
		},
	}

	// File input flags
	cmd.Flags().StringSliceVarP(&opts.Files, "file", "f", []string{}, "Configuration files to diff (supports glob patterns)")

	// Output flags
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "table", "Output format: table, unified, json, yaml, summary")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")

	// Diff options
	cmd.Flags().BoolVar(&opts.StrictEnv, "strict-env", false, "Fail on undefined environment variables")
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Atlas project ID (overrides config)")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 10*time.Minute, "Timeout for diff computation")
	cmd.Flags().IntVar(&opts.ShowContext, "show-context", 3, "Number of context lines to show around changes")
	cmd.Flags().BoolVar(&opts.Detailed, "detailed", false, "Show detailed field-level differences")
	cmd.Flags().BoolVar(&opts.PreserveExisting, "preserve-existing", false, "Only show additions and updates, exclude deletions")
	cmd.Flags().BoolVar(&opts.NoTruncate, "no-truncate", false, "Don't truncate long resource names in table output")

	return cmd
}

func runDiff(cmd *cobra.Command, opts *DiffOptions) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	// Validate options
	if err := validateDiffOptions(opts); err != nil {
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

	// Compute differences
	diff, err := computeDifferences(ctx, configs, services, cfg, opts)
	if err != nil {
		return fmt.Errorf("failed to compute differences: %w", err)
	}

	// Display differences
	return displayDifferences(diff, opts)
}

func computeDifferences(ctx context.Context, configs []*apply.LoadResult, services *ServiceClients, cfg *config.Config, opts *DiffOptions) (*apply.Diff, error) {
	if opts.Verbose {
		fmt.Println("Computing differences...")
	}

	// Create Atlas client for discovery
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Atlas client for discovery: %w", err)
	}

	// Initialize discovery service
	discoveryService := apply.NewAtlasStateDiscovery(atlasClient)

	// Initialize diff engine
	diffEngine := apply.NewDiffEngine()
	if opts.Detailed {
		diffEngine.CompareTimestamps = true
		diffEngine.IgnoreDefaults = false
	}
	diffEngine.PreserveExisting = opts.PreserveExisting

	// Discover current state
	if opts.Verbose {
		fmt.Println("Discovering current state...")
	}

	// Resolve project name to project ID (parity with plan/apply)
	projectNameOrID := getProjectID(configs, &ApplyOptions{ProjectID: opts.ProjectID})
	orgID := getOrganizationID(configs)

	resolvedProjectID := projectNameOrID
	if projectNameOrID != "" {
		id, err := resolveProjectID(ctx, projectNameOrID, services.ProjectsService, orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve project ID for '%s': %w", projectNameOrID, err)
		}
		resolvedProjectID = id
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

	return diff, nil
}

func displayDifferences(diff *apply.Diff, opts *DiffOptions) error {
	// When there are no operations, still honor output format for consistency
	if diff.Summary.TotalOperations == 0 {
		switch strings.ToLower(opts.OutputFormat) {
		case "json":
			return displayDiffJSON(diff)
		case "yaml":
			return displayDiffYAML(diff)
		case "summary":
			return displayDiffSummary(diff, opts)
		case "table":
			fmt.Println("No differences found - current state matches desired state")
			return nil
		default: // unified
			return displayDiffUnified(diff, opts)
		}
	}

	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		return displayDiffJSON(diff)
	case "yaml":
		return displayDiffYAML(diff)
	case "table":
		return displayDiffTable(diff, opts)
	case "summary":
		return displayDiffSummary(diff, opts)
	default: // unified
		return displayDiffUnified(diff, opts)
	}
}

func displayDiffJSON(diff *apply.Diff) error {
	formatter := output.NewFormatter(config.OutputJSON, os.Stdout)
	return formatter.Format(diff)
}

func displayDiffYAML(diff *apply.Diff) error {
	formatter := output.NewFormatter(config.OutputYAML, os.Stdout)
	return formatter.Format(diff)
}

func displayDiffSummary(diff *apply.Diff, opts *DiffOptions) error {
	// Reuse diff formatter summary for consistency
	formatter := apply.NewDiffFormatter()
	output, err := formatter.Format(diff, &apply.FormatOptions{Format: "summary", UseColors: !opts.NoColor, ShowNoChange: opts.Verbose, Verbose: opts.Verbose})
	if err != nil {
		return err
	}
	fmt.Print(output)
	return nil
}

func displayDiffTable(diff *apply.Diff, opts *DiffOptions) error {
	fmt.Printf("Resource Differences\n")
	fmt.Printf("===================\n\n")

	if len(diff.Operations) == 0 {
		if strings.ToLower(opts.OutputFormat) == "table" {
			fmt.Println("No differences found")
		}
		return nil
	}

	// Calculate column widths based on whether truncation is enabled
	var nameColWidth int
	var totalWidth int
	if opts.NoTruncate {
		// Calculate the maximum width needed for resource names
		nameColWidth = len("Resource Name")
		for _, op := range diff.Operations {
			if len(op.ResourceName) > nameColWidth {
				nameColWidth = len(op.ResourceName)
			}
		}
		totalWidth = 15 + 20 + nameColWidth + 10 + 10 + 4 // operation + type + name + risk + changes + spaces
	} else {
		nameColWidth = 25
		totalWidth = 90
	}

	fmt.Printf("%-15s %-20s %-*s %-10s %s\n", "Operation", "Resource Type", nameColWidth, "Resource Name", "Risk", "Changes")
	fmt.Printf("%s\n", strings.Repeat("-", totalWidth))

	for _, op := range diff.Operations {
		if op.Type == apply.OperationNoChange && !opts.Verbose {
			continue
		}

		riskColor := ""
		opColor := ""
		if !opts.NoColor {
			switch op.Type {
			case apply.OperationCreate:
				opColor = "\033[32m" // Green
			case apply.OperationUpdate:
				opColor = "\033[33m" // Yellow
			case apply.OperationDelete:
				opColor = "\033[31m" // Red
			}

			if op.Impact != nil {
				switch op.Impact.RiskLevel {
				case apply.RiskLevelHigh, apply.RiskLevelCritical:
					riskColor = "\033[31m" // Red
				case apply.RiskLevelMedium:
					riskColor = "\033[33m" // Yellow
				case apply.RiskLevelLow:
					riskColor = "\033[32m" // Green
				}
			}
		}
		resetColor := ""
		if opColor != "" || riskColor != "" {
			resetColor = "\033[0m"
		}

		changes := ""
		if op.Type == apply.OperationUpdate && opts.Detailed {
			// Show field-level changes
			if op.FieldChanges != nil {
				var fieldList []string
				for _, fieldChange := range op.FieldChanges {
					fieldList = append(fieldList, fieldChange.Path)
				}
				if len(fieldList) > 3 {
					changes = fmt.Sprintf("%s and %d more", strings.Join(fieldList[:3], ", "), len(fieldList)-3)
				} else {
					changes = strings.Join(fieldList, ", ")
				}
			}
		}

		riskLevel := "N/A"
		if op.Impact != nil {
			riskLevel = string(op.Impact.RiskLevel)
		}

		// Handle resource name truncation
		resourceName := op.ResourceName
		if !opts.NoTruncate && len(resourceName) > nameColWidth {
			resourceName = resourceName[:nameColWidth-3] + "..."
		}

		fmt.Printf("%s%-15s%s %-20s %-*s %s%-10s%s %s\n",
			opColor, op.Type, resetColor,
			op.ResourceType,
			nameColWidth, resourceName,
			riskColor, riskLevel, resetColor,
			changes)
	}

	return nil
}

func displayDiffUnified(diff *apply.Diff, opts *DiffOptions) error {
	// Use the existing diff formatter from the apply engine
	formatter := apply.NewDiffFormatter()

	formatOpts := &apply.FormatOptions{
		Format:       "unified",
		UseColors:    !opts.NoColor,
		ShowNoChange: opts.Verbose,
		Verbose:      opts.Verbose,
	}

	output, err := formatter.Format(diff, formatOpts)
	if err != nil {
		return fmt.Errorf("failed to format diff: %w", err)
	}

	fmt.Print(output)
	return nil
}

func validateDiffOptions(opts *DiffOptions) error {
	if len(opts.Files) == 0 {
		return fmt.Errorf("at least one configuration file must be specified with --file")
	}

	// Validate output format
	validOutputFormats := []string{"unified", "table", "json", "yaml", "summary"}
	if !contains(validOutputFormats, strings.ToLower(opts.OutputFormat)) {
		return fmt.Errorf("invalid output format: %s (valid options: %s)", opts.OutputFormat, strings.Join(validOutputFormats, ", "))
	}

	// Validate timeout
	if opts.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	// Validate show-context
	if opts.ShowContext < 0 {
		return fmt.Errorf("show-context must be non-negative")
	}

	return nil
}
