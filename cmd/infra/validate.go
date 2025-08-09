package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/types"
	vld "github.com/teabranch/matlas-cli/internal/validation"
)

// ValidateOptions contains the options for the validate command
type ValidateOptions struct {
	Files        []string
	OutputFormat string
	Verbose      bool
	NoColor      bool
	StrictEnv    bool
	StrictMode   bool
	LintRules    bool
	BatchMode    bool
}

// NewValidateCmd creates the validate subcommand
func NewValidateCmd() *cobra.Command {
	opts := &ValidateOptions{}

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration files without applying them",
		Long: `Validate YAML configuration files for syntax, schema compliance, and dependency correctness.

This command checks configuration files for errors without making any changes to Atlas resources.
It performs syntax checking, schema validation, dependency verification, and optional linting.`,
		Example: `  # Validate a single configuration file
  matlas infra validate -f config.yaml
  matlas infra validate config.yaml

  # Validate multiple files with detailed output
  matlas infra validate -f "configs/*.yaml" --verbose

  # Validate with strict environment variable checking
  matlas infra validate config.yaml --strict-env

  # Validate with linting rules enabled
  matlas infra validate config.yaml --lint

  # Validate in batch mode (multiple files)
  matlas infra validate -f cluster.yaml -f users.yaml --batch
  matlas infra validate cluster.yaml users.yaml --batch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support positional arguments as files if no --file flag provided
			if len(opts.Files) == 0 && len(args) > 0 {
				opts.Files = args
			}
			return runValidate(cmd, opts)
		},
	}

	// File input flags
	cmd.Flags().StringSliceVarP(&opts.Files, "file", "f", []string{}, "Configuration files to validate (supports glob patterns)")

	// Output and behavior flags
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "table", "Output format: table, json, yaml, summary")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")

	// Validation behavior flags
	cmd.Flags().BoolVar(&opts.StrictEnv, "strict-env", false, "Fail on undefined environment variables")
	cmd.Flags().BoolVar(&opts.StrictMode, "strict", false, "Enable strict validation mode")
	cmd.Flags().BoolVar(&opts.LintRules, "lint", false, "Enable linting rules for best practices")
	cmd.Flags().BoolVar(&opts.BatchMode, "batch", false, "Enable batch validation mode for multiple files")

	return cmd
}

func runValidate(cmd *cobra.Command, opts *ValidateOptions) error {
	ctx := cmd.Context()

	// Validate options
	if err := validateValidateOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Expand file patterns
	files, err := expandFilePatterns(opts.Files)
	if err != nil {
		return fmt.Errorf("failed to expand file patterns: %w", err)
	}

	// Initialize configuration loader
	loaderOpts := &apply.LoaderOptions{
		StrictEnv:    opts.StrictEnv,
		Debug:        opts.Verbose,
		CacheEnabled: true,
		AllowStdin:   true,
		MaxFileSize:  10 * 1024 * 1024, // 10MB
	}

	loader := apply.NewConfigurationLoader(loaderOpts)

	// Initialize validation options
	validationOpts := apply.DefaultValidatorOptions()
	validationOpts.StrictMode = opts.StrictMode

	// Perform validation
	var results []*ValidationFileResult

	if opts.BatchMode {
		results, err = validateBatch(ctx, files, loader, validationOpts, opts)
	} else {
		results, err = validateIndividual(ctx, files, loader, validationOpts, opts)
	}

	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Display results
	return displayValidationResults(results, opts)
}

type ValidationFileResult struct {
	FilePath          string                          `json:"filePath"`
	LoadResult        *apply.LoadResult               `json:"loadResult,omitempty"`
	ValidationResult  *apply.ResourceValidationResult `json:"validationResult,omitempty"`
	LintingResults    []apply.ValidationError         `json:"lintingResults,omitempty"`
	DependencyResults []apply.ValidationError         `json:"dependencyResults,omitempty"`
	IsValid           bool                            `json:"isValid"`
	Errors            []string                        `json:"errors,omitempty"`
	Warnings          []string                        `json:"warnings,omitempty"`
	ProcessingTime    time.Duration                   `json:"processingTime"`
}

func validateIndividual(ctx context.Context, files []string, loader *apply.ConfigurationLoader, validationOpts *apply.ValidatorOptions, opts *ValidateOptions) ([]*ValidationFileResult, error) {
	var results []*ValidationFileResult

	for _, file := range files {
		start := time.Now()

		if opts.Verbose {
			fmt.Printf("Validating %s...\n", file)
		}

		result := &ValidationFileResult{
			FilePath:       file,
			ProcessingTime: 0,
		}

		// Load configuration
		loadResult, err := loader.LoadApplyConfig(file)
		if err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to load configuration: %v", err))
			result.ProcessingTime = time.Since(start)
			results = append(results, result)
			continue
		}

		result.LoadResult = loadResult

		// Report load warnings
		for _, warning := range loadResult.Warnings {
			result.Warnings = append(result.Warnings, warning.Message)
		}

		// Report load errors
		if len(loadResult.Errors) > 0 {
			result.IsValid = false
			for _, errMsg := range loadResult.Errors {
				result.Errors = append(result.Errors, errMsg.Message)
			}
			result.ProcessingTime = time.Since(start)
			results = append(results, result)
			continue
		}

		// Validate configuration
		if applyConfig, ok := loadResult.Config.(*types.ApplyConfig); ok {
			validationResult := apply.ValidateApplyConfig(applyConfig, validationOpts)
			result.ValidationResult = &apply.ResourceValidationResult{
				ResourceName: applyConfig.Metadata.Name,
				ResourceType: applyConfig.Kind,
				Valid:        validationResult.Valid,
				Errors:       []string{},
				Warnings:     []string{},
			}

			// Convert validation errors
			for _, validationErr := range validationResult.Errors {
				result.Errors = append(result.Errors, validationErr.Message)
				result.ValidationResult.Errors = append(result.ValidationResult.Errors, validationErr.Message)
			}

			// Convert validation warnings
			for _, validationWarn := range validationResult.Warnings {
				result.Warnings = append(result.Warnings, validationWarn.Message)
				result.ValidationResult.Warnings = append(result.ValidationResult.Warnings, validationWarn.Message)
			}

			// Check if overall validation passed
			result.IsValid = validationResult.Valid && len(result.Errors) == 0
		} else if applyDoc, ok := loadResult.Config.(*types.ApplyDocument); ok {
			validationResult := apply.ValidateApplyDocument(applyDoc, validationOpts)
			result.ValidationResult = &apply.ResourceValidationResult{
				ResourceName: applyDoc.Metadata.Name,
				ResourceType: string(applyDoc.Kind),
				Valid:        validationResult.Valid,
				Errors:       []string{},
				Warnings:     []string{},
			}

			// Convert validation errors
			for _, validationErr := range validationResult.Errors {
				result.Errors = append(result.Errors, validationErr.Message)
				result.ValidationResult.Errors = append(result.ValidationResult.Errors, validationErr.Message)
			}

			// Convert validation warnings
			for _, validationWarn := range validationResult.Warnings {
				result.Warnings = append(result.Warnings, validationWarn.Message)
				result.ValidationResult.Warnings = append(result.ValidationResult.Warnings, validationWarn.Message)
			}

			// Check if overall validation passed
			result.IsValid = validationResult.Valid && len(result.Errors) == 0
		} else {
			// Handle case where config is not a recognized type
			result.IsValid = len(result.Errors) == 0
		}

		result.ProcessingTime = time.Since(start)
		results = append(results, result)
	}

	return results, nil
}

func validateBatch(ctx context.Context, files []string, loader *apply.ConfigurationLoader, validationOpts *apply.ValidatorOptions, opts *ValidateOptions) ([]*ValidationFileResult, error) {
	if opts.Verbose {
		fmt.Printf("Batch validating %d files...\n", len(files))
	}

	// Load all configurations first
	var loadResults []*apply.LoadResult
	var results []*ValidationFileResult

	for _, file := range files {
		start := time.Now()

		result := &ValidationFileResult{
			FilePath: file,
		}

		loadResult, err := loader.LoadApplyConfig(file)
		if err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to load configuration: %v", err))
			result.ProcessingTime = time.Since(start)
			results = append(results, result)
			continue
		}

		result.LoadResult = loadResult
		loadResults = append(loadResults, loadResult)

		// Report load warnings and errors
		for _, warning := range loadResult.Warnings {
			result.Warnings = append(result.Warnings, warning.Message)
		}

		if len(loadResult.Errors) > 0 {
			result.IsValid = false
			for _, errMsg := range loadResult.Errors {
				result.Errors = append(result.Errors, errMsg.Message)
			}
		}

		result.ProcessingTime = time.Since(start)
		results = append(results, result)
	}

	// Cross-file dependency validation across all loaded configs
	// Aggregate project configs and run dependency validation per project
	projectAggregates, projectToResultIndexes := aggregateProjectConfigs(loadResults)

	depValidator := vld.NewDependencyValidator(validationOpts.StrictMode)
	for projectName, projectCfg := range projectAggregates {
		issues, err := depValidator.ValidateProjectDependencies(projectCfg)
		if err != nil {
			// Attach a high-level dependency error to all associated files for this project
			for _, idx := range projectToResultIndexes[projectName] {
				results[idx].IsValid = false
				results[idx].DependencyResults = append(results[idx].DependencyResults, apply.ValidationError{
					Path:     "spec",
					Field:    "dependencies",
					Value:    projectName,
					Message:  fmt.Sprintf("Dependency validation failed: %v", err),
					Code:     "DEPENDENCY_VALIDATION_ERROR",
					Severity: "error",
				})
				results[idx].Errors = append(results[idx].Errors, fmt.Sprintf("Dependency validation failed for project %s: %v", projectName, err))
			}
			continue
		}

		// Map issues to file results belonging to the same project
		for _, issue := range issues {
			sev := "warning"
			if issue.Severity == "error" {
				sev = "error"
			}
			ve := apply.ValidationError{
				Path:     issue.SourceResource,
				Field:    issue.DependencyType,
				Value:    issue.TargetResource,
				Message:  issue.Message,
				Code:     strings.ToUpper(strings.ReplaceAll(issue.DependencyType, " ", "_")),
				Severity: sev,
			}
			for _, idx := range projectToResultIndexes[projectName] {
				results[idx].DependencyResults = append(results[idx].DependencyResults, ve)
				if sev == "error" {
					results[idx].IsValid = false
					results[idx].Errors = append(results[idx].Errors, issue.Error())
				} else if opts.Verbose {
					results[idx].Warnings = append(results[idx].Warnings, issue.Error())
				}
			}
		}
	}

	// Individual validation for each file
	for _, result := range results {
		if result.LoadResult != nil && len(result.LoadResult.Errors) == 0 {
			if applyConfig, ok := result.LoadResult.Config.(*types.ApplyConfig); ok {
				validationResult := apply.ValidateApplyConfig(applyConfig, validationOpts)
				result.ValidationResult = &apply.ResourceValidationResult{
					ResourceName: applyConfig.Metadata.Name,
					ResourceType: applyConfig.Kind,
					Valid:        validationResult.Valid,
					Errors:       []string{},
					Warnings:     []string{},
				}

				// Convert validation errors
				for _, validationErr := range validationResult.Errors {
					result.Errors = append(result.Errors, validationErr.Message)
					result.ValidationResult.Errors = append(result.ValidationResult.Errors, validationErr.Message)
				}

				// Convert validation warnings
				for _, validationWarn := range validationResult.Warnings {
					result.Warnings = append(result.Warnings, validationWarn.Message)
					result.ValidationResult.Warnings = append(result.ValidationResult.Warnings, validationWarn.Message)
				}

				if !validationResult.Valid {
					result.IsValid = false
				}
			} else if applyDoc, ok := result.LoadResult.Config.(*types.ApplyDocument); ok {
				validationResult := apply.ValidateApplyDocument(applyDoc, validationOpts)
				result.ValidationResult = &apply.ResourceValidationResult{
					ResourceName: applyDoc.Metadata.Name,
					ResourceType: string(applyDoc.Kind),
					Valid:        validationResult.Valid,
					Errors:       []string{},
					Warnings:     []string{},
				}

				// Convert validation errors
				for _, validationErr := range validationResult.Errors {
					result.Errors = append(result.Errors, validationErr.Message)
					result.ValidationResult.Errors = append(result.ValidationResult.Errors, validationErr.Message)
				}

				// Convert validation warnings
				for _, validationWarn := range validationResult.Warnings {
					result.Warnings = append(result.Warnings, validationWarn.Message)
					result.ValidationResult.Warnings = append(result.ValidationResult.Warnings, validationWarn.Message)
				}

				if !validationResult.Valid {
					result.IsValid = false
				}
			}
		}
	}

	return results, nil
}

func displayValidationResults(results []*ValidationFileResult, opts *ValidateOptions) error {
	// Prepare formatter
	var outputFormat config.OutputFormat
	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		outputFormat = config.OutputJSON
	case "yaml":
		outputFormat = config.OutputYAML
	default:
		outputFormat = config.OutputText
	}
	formatter := output.NewFormatter(outputFormat, os.Stdout)

	// Count totals
	var totalFiles, validFiles, invalidFiles int
	var totalErrors, totalWarnings int

	for _, result := range results {
		totalFiles++
		if result.IsValid {
			validFiles++
		} else {
			invalidFiles++
		}
		totalErrors += len(result.Errors)
		totalWarnings += len(result.Warnings)
	}

	// Display results based on format
	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		return formatter.Format(results)
	case "yaml":
		return formatter.Format(results)
	case "summary":
		return displayValidationSummary(totalFiles, validFiles, invalidFiles, totalErrors, totalWarnings, opts)
	default: // table
		return displayValidationTable(results, opts)
	}
}

func displayValidationSummary(totalFiles, validFiles, invalidFiles, totalErrors, totalWarnings int, opts *ValidateOptions) error {
	fmt.Printf("Validation Summary:\n")
	fmt.Printf("  Total files: %d\n", totalFiles)
	fmt.Printf("  Valid files: %d\n", validFiles)
	fmt.Printf("  Invalid files: %d\n", invalidFiles)
	fmt.Printf("  Total errors: %d\n", totalErrors)
	fmt.Printf("  Total warnings: %d\n", totalWarnings)

	if invalidFiles > 0 {
		fmt.Printf("\n❌ Validation failed\n")
		return fmt.Errorf("validation failed for %d files", invalidFiles)
	} else {
		fmt.Printf("\n✅ All files are valid\n")
	}

	return nil
}

func displayValidationTable(results []*ValidationFileResult, opts *ValidateOptions) error {
	fmt.Printf("Configuration Validation Results:\n\n")

	for _, result := range results {
		status := "✅ VALID"
		if !result.IsValid {
			status = "❌ INVALID"
		}

		fmt.Printf("File: %s [%s] (%.2fms)\n", result.FilePath, status, float64(result.ProcessingTime.Nanoseconds())/1e6)

		if len(result.Errors) > 0 {
			fmt.Printf("  Errors:\n")
			for _, err := range result.Errors {
				fmt.Printf("    - %s\n", err)
			}
		}

		if len(result.Warnings) > 0 && opts.Verbose {
			fmt.Printf("  Warnings:\n")
			for _, warning := range result.Warnings {
				fmt.Printf("    - %s\n", warning)
			}
		}

		fmt.Println()
	}

	// Count invalid files for exit code
	invalidCount := 0
	for _, result := range results {
		if !result.IsValid {
			invalidCount++
		}
	}

	if invalidCount > 0 {
		return fmt.Errorf("validation failed for %d files", invalidCount)
	}

	return nil
}

func validateValidateOptions(opts *ValidateOptions) error {
	if len(opts.Files) == 0 {
		return fmt.Errorf("at least one configuration file must be specified with --file")
	}

	// Validate output format
	validOutputFormats := []string{"table", "json", "yaml", "summary"}
	if !contains(validOutputFormats, strings.ToLower(opts.OutputFormat)) {
		return fmt.Errorf("invalid output format: %s (valid options: %s)", opts.OutputFormat, strings.Join(validOutputFormats, ", "))
	}

	return nil
}

// aggregateProjectConfigs builds a map of project name to aggregated types.ProjectConfig
// from a set of loaded configuration results (ApplyConfig or ApplyDocument).
func aggregateProjectConfigs(loadResults []*apply.LoadResult) (map[string]*types.ProjectConfig, map[string][]int) {
	projectMap := make(map[string]*types.ProjectConfig)
	projectToResultIndexes := make(map[string][]int)

	// Helper to ensure project exists
	ensureProject := func(name string) *types.ProjectConfig {
		if name == "" {
			name = "<unknown>"
		}
		if _, ok := projectMap[name]; !ok {
			projectMap[name] = &types.ProjectConfig{
				Name:          name,
				Metadata:      types.ResourceMetadata{Name: name},
				Clusters:      []types.ClusterConfig{},
				DatabaseUsers: []types.DatabaseUserConfig{},
				NetworkAccess: []types.NetworkAccessConfig{},
			}
		}
		return projectMap[name]
	}

	for idx, lr := range loadResults {
		switch cfg := lr.Config.(type) {
		case *types.ApplyConfig:
			proj := ensureProject(cfg.Spec.Name)
			// Carry over org if present and not already set
			if proj.OrganizationID == "" && cfg.Spec.OrganizationID != "" {
				proj.OrganizationID = cfg.Spec.OrganizationID
			}
			proj.Tags = mergeStringMaps(proj.Tags, cfg.Spec.Tags)
			proj.Clusters = append(proj.Clusters, cfg.Spec.Clusters...)
			proj.DatabaseUsers = append(proj.DatabaseUsers, cfg.Spec.DatabaseUsers...)
			proj.NetworkAccess = append(proj.NetworkAccess, cfg.Spec.NetworkAccess...)
			projectToResultIndexes[proj.Name] = appendUniqueInt(projectToResultIndexes[proj.Name], idx)
		case *types.ApplyDocument:
			// Convert manifests into ProjectConfig entries keyed by project name
			for _, res := range cfg.Resources {
				switch res.Kind {
				case types.KindProject:
					if pm, ok := any(res.Spec).(types.ProjectConfig); ok {
						proj := ensureProject(pm.Name)
						if proj.OrganizationID == "" && pm.OrganizationID != "" {
							proj.OrganizationID = pm.OrganizationID
						}
						proj.Tags = mergeStringMaps(proj.Tags, pm.Tags)
						projectToResultIndexes[proj.Name] = appendUniqueInt(projectToResultIndexes[proj.Name], idx)
					} else if pm, ok := decodeSpec[types.ProjectConfig](res.Spec); ok {
						proj := ensureProject(pm.Name)
						if proj.OrganizationID == "" && pm.OrganizationID != "" {
							proj.OrganizationID = pm.OrganizationID
						}
						proj.Tags = mergeStringMaps(proj.Tags, pm.Tags)
						projectToResultIndexes[proj.Name] = appendUniqueInt(projectToResultIndexes[proj.Name], idx)
					}
				case types.KindCluster:
					// Extract project name from spec and convert
					if cm, ok := any(res.Spec).(types.ClusterSpec); ok {
						proj := ensureProject(cm.ProjectName)
						proj.Clusters = append(proj.Clusters, convertClusterSpecToConfig(res.Metadata, cm))
						projectToResultIndexes[proj.Name] = appendUniqueInt(projectToResultIndexes[proj.Name], idx)
					} else if cm, ok := decodeSpec[types.ClusterSpec](res.Spec); ok {
						proj := ensureProject(cm.ProjectName)
						proj.Clusters = append(proj.Clusters, convertClusterSpecToConfig(res.Metadata, cm))
						projectToResultIndexes[proj.Name] = appendUniqueInt(projectToResultIndexes[proj.Name], idx)
					}
				case types.KindDatabaseUser:
					if um, ok := any(res.Spec).(types.DatabaseUserSpec); ok {
						proj := ensureProject(um.ProjectName)
						proj.DatabaseUsers = append(proj.DatabaseUsers, convertUserSpecToConfig(res.Metadata, um))
						projectToResultIndexes[proj.Name] = appendUniqueInt(projectToResultIndexes[proj.Name], idx)
					} else if um, ok := decodeSpec[types.DatabaseUserSpec](res.Spec); ok {
						proj := ensureProject(um.ProjectName)
						proj.DatabaseUsers = append(proj.DatabaseUsers, convertUserSpecToConfig(res.Metadata, um))
						projectToResultIndexes[proj.Name] = appendUniqueInt(projectToResultIndexes[proj.Name], idx)
					}
				case types.KindNetworkAccess:
					if nm, ok := any(res.Spec).(types.NetworkAccessSpec); ok {
						proj := ensureProject(nm.ProjectName)
						proj.NetworkAccess = append(proj.NetworkAccess, convertNetworkSpecToConfig(res.Metadata, nm))
						projectToResultIndexes[proj.Name] = appendUniqueInt(projectToResultIndexes[proj.Name], idx)
					} else if nm, ok := decodeSpec[types.NetworkAccessSpec](res.Spec); ok {
						proj := ensureProject(nm.ProjectName)
						proj.NetworkAccess = append(proj.NetworkAccess, convertNetworkSpecToConfig(res.Metadata, nm))
						projectToResultIndexes[proj.Name] = appendUniqueInt(projectToResultIndexes[proj.Name], idx)
					}
				}
			}
		}
	}

	return projectMap, projectToResultIndexes
}

func convertClusterSpecToConfig(meta types.ResourceMetadata, spec types.ClusterSpec) types.ClusterConfig {
	return types.ClusterConfig{
		Metadata:         meta,
		Tags:             nil,
		Provider:         spec.Provider,
		Region:           spec.Region,
		InstanceSize:     spec.InstanceSize,
		DiskSizeGB:       spec.DiskSizeGB,
		BackupEnabled:    spec.BackupEnabled,
		TierType:         spec.TierType,
		MongoDBVersion:   spec.MongoDBVersion,
		ClusterType:      spec.ClusterType,
		ReplicationSpecs: spec.ReplicationSpecs,
		AutoScaling:      spec.AutoScaling,
		Encryption:       spec.Encryption,
		BiConnector:      spec.BiConnector,
		DependsOn:        meta.DependsOn,
	}
}

func convertUserSpecToConfig(meta types.ResourceMetadata, spec types.DatabaseUserSpec) types.DatabaseUserConfig {
	return types.DatabaseUserConfig{
		Metadata:     meta,
		Username:     spec.Username,
		Password:     spec.Password,
		Roles:        spec.Roles,
		AuthDatabase: spec.AuthDatabase,
		Scopes:       spec.Scopes,
		DependsOn:    meta.DependsOn,
	}
}

func convertNetworkSpecToConfig(meta types.ResourceMetadata, spec types.NetworkAccessSpec) types.NetworkAccessConfig {
	return types.NetworkAccessConfig{
		Metadata:         meta,
		IPAddress:        spec.IPAddress,
		CIDR:             spec.CIDR,
		AWSSecurityGroup: spec.AWSSecurityGroup,
		Comment:          spec.Comment,
		DeleteAfterDate:  spec.DeleteAfterDate,
		DependsOn:        meta.DependsOn,
	}
}

func mergeStringMaps(a, b map[string]string) map[string]string {
	if a == nil && b == nil {
		return nil
	}
	out := make(map[string]string)
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func appendUniqueInt(slice []int, v int) []int {
	for _, x := range slice {
		if x == v {
			return slice
		}
	}
	return append(slice, v)
}

// decodeSpec attempts to decode an untyped manifest spec (map[string]any) into a typed struct T
func decodeSpec[T any](raw interface{}) (T, bool) {
	var zero T
	// Fast-path: if already the correct type
	if v, ok := raw.(T); ok {
		return v, true
	}
	// Attempt JSON round-trip into the target type
	b, err := json.Marshal(raw)
	if err != nil {
		return zero, false
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return zero, false
	}
	return out, true
}
