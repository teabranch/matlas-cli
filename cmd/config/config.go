package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/validation"
)

// NewConfigCmd creates the config command with all its subcommands
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long: `Manage matlas-cli configuration files and settings.

This command group provides operations for validating, generating, importing,
and exporting configuration files, as well as migration utilities.`,
		Aliases:      []string{"cfg"},
		SilenceUsage: true,
	}

	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newTemplateCmd())
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newMigrateCmd())

	return cmd
}

func newValidateCmd() *cobra.Command {
	var configFile string
	var schemaFile string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "validate [config-file]",
		Short: "Validate configuration files",
		Long: `Validate matlas-cli configuration files for syntax and structure.

This command checks configuration files for:
- YAML syntax errors
- Required fields
- Valid field values and formats
- Schema compliance (if schema file provided)`,
		Args: cobra.MaximumNArgs(1),
		Example: `  # Validate default config file
  matlas config validate

  # Validate specific config file
  matlas config validate myconfig.yaml

  # Validate with custom schema
  matlas config validate myconfig.yaml --schema custom-schema.json

  # Verbose validation output
  matlas config validate --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				configFile = args[0]
			}
			return runValidateConfig(cmd, configFile, schemaFile, verbose)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "Configuration file path")
	cmd.Flags().StringVar(&schemaFile, "schema", "", "JSON schema file for validation")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose validation output")

	return cmd
}

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Generate configuration templates",
		Long:  "Generate configuration file templates for different use cases",
	}

	cmd.AddCommand(newTemplateGenerateCmd())
	cmd.AddCommand(newTemplateListCmd())

	return cmd
}

func newTemplateGenerateCmd() *cobra.Command {
	var templateType string
	var outputFile string
	var format string

	cmd := &cobra.Command{
		Use:   "generate <template-type>",
		Short: "Generate a configuration template",
		Long: `Generate configuration templates for different scenarios.

Available template types:

CLI Configuration Templates:
- basic: Basic CLI configuration
- atlas: Atlas-specific CLI configuration
- database: Database connection CLI configuration
- complete: Complete CLI configuration with all options

Resource Configuration Templates:
- apply: MongoDB Atlas resource configuration for apply operations`,
		Args: cobra.ExactArgs(1),
		Example: `  # Generate basic configuration template
  matlas config template generate basic

  # Generate Atlas configuration template to file
  matlas config template generate atlas --output atlas-config.yaml

  # Generate in JSON format
  matlas config template generate complete --format json --output config.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			templateType = args[0]
			return runGenerateTemplate(cmd, templateType, outputFile, format)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().StringVarP(&format, "format", "f", "yaml", "Output format (yaml, json)")

	return cmd
}

func newTemplateListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available configuration templates",
		Long:  "List all available configuration templates with descriptions",
		Example: `  # List all templates
  matlas config template list

  # List templates in JSON format
  matlas config template list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListTemplates(cmd)
		},
	}

	return cmd
}

func newImportCmd() *cobra.Command {
	var sourceFile string
	var targetFile string
	var format string
	var merge bool

	cmd := &cobra.Command{
		Use:   "import <source-file>",
		Short: "Import configuration from external sources",
		Long: `Import configuration from external files or formats (experimental).

This command can import configurations from:
- Other matlas-cli config files
- Environment variable files (.env)
- JSON configuration files
- YAML configuration files`,
		Args: cobra.ExactArgs(1),
		Example: `  # Import from environment file
  matlas config import config.env

  # Import and merge with existing config
  matlas config import external.yaml --merge

  # Import to specific target file
  matlas config import source.json --target ~/.matlas/imported-config.yaml

  # Import with format conversion
  matlas config import config.json --format yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceFile = args[0]
			return runImportConfig(cmd, sourceFile, targetFile, format, merge)
		},
	}

	cmd.Flags().StringVarP(&targetFile, "target", "t", "", "Target configuration file")
	cmd.Flags().StringVarP(&format, "format", "f", "", "Output format (yaml, json)")
	cmd.Flags().BoolVar(&merge, "merge", false, "Merge with existing configuration")

	// Gate behind experimental flag
	cmd.Hidden = true
	return cmd
}

func newExportCmd() *cobra.Command {
	var outputFile string
	var format string
	var includeSecrets bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export configuration to external formats",
		Long: `Export matlas-cli configuration to external formats (experimental).

This command can export configurations to:
- Environment variable files (.env)
- JSON configuration files
- YAML configuration files
- Shell export statements`,
		Example: `  # Export to environment file
  matlas config export --format env --output config.env

  # Export to JSON
  matlas config export --format json --output config.json

  # Export including sensitive values
  matlas config export --include-secrets --format yaml

  # Export as shell export statements
  matlas config export --format shell`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExportConfig(cmd, outputFile, format, includeSecrets)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().StringVarP(&format, "format", "f", "yaml", "Export format (yaml, json, env, shell)")
	cmd.Flags().BoolVar(&includeSecrets, "include-secrets", false, "Include sensitive values in export")

	cmd.Hidden = true
	return cmd
}

func newMigrateCmd() *cobra.Command {
	var fromVersion string
	var toVersion string
	var backup bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate configuration between versions",
		Long: `Migrate configuration files between different matlas-cli versions (experimental).

This command helps upgrade or downgrade configuration files when
the configuration schema changes between versions.`,
		Example: `  # Migrate to latest version
  matlas config migrate

  # Migrate from specific version
  matlas config migrate --from v1.0.0 --to v2.0.0

  # Migrate with backup
  matlas config migrate --backup`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateConfig(cmd, fromVersion, toVersion, backup)
		},
	}

	cmd.Flags().StringVar(&fromVersion, "from", "", "Source version (auto-detected if not specified)")
	cmd.Flags().StringVar(&toVersion, "to", "latest", "Target version")
	cmd.Flags().BoolVar(&backup, "backup", true, "Create backup before migration")

	cmd.Hidden = true
	return cmd
}

// Implementation functions

func runValidateConfig(cmd *cobra.Command, configFile, schemaFile string, verbose bool) error {
	// Determine config file path
	if configFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configFile = filepath.Join(homeDir, config.DefaultConfigDir, "config.yaml")
	}

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configFile)
	}

	// Read and parse config file
	configData, err := os.ReadFile(configFile) //nolint:gosec // reading user-specified path is expected for CLI tool
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse YAML
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configData, &configMap); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Load and validate configuration
	cfg, err := config.Load(cmd, configFile)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Perform additional validations
	var validationErrors []string

	// Validate API keys format if present (basic length validation)
	if cfg.APIKey != "" {
		if len(cfg.APIKey) < 10 {
			validationErrors = append(validationErrors, "API key: too short (minimum 10 characters)")
		}
	}

	if cfg.PublicKey != "" {
		if len(cfg.PublicKey) < 10 {
			validationErrors = append(validationErrors, "Public key: too short (minimum 10 characters)")
		}
	}

	// Validate project ID format if present
	if cfg.ProjectID != "" {
		if err := validation.ValidateProjectID(cfg.ProjectID); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Project ID: %s", err.Error()))
		}
	}

	// Validate cluster name format if present
	if cfg.ClusterName != "" {
		if err := validation.ValidateClusterName(cfg.ClusterName); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Cluster name: %s", err.Error()))
		}
	}

	// Schema validation if schema file provided
	if schemaFile != "" {
		if err := validateWithSchema(configData, schemaFile); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Schema validation: %s", err.Error()))
		}
	}

	// Output results
	if len(validationErrors) > 0 {
		fmt.Printf("❌ Configuration validation failed for: %s\n\n", configFile)
		for _, errMsg := range validationErrors {
			fmt.Printf("  • %s\n", errMsg)
		}
		fmt.Println()
		return fmt.Errorf("configuration validation failed with %d error(s)", len(validationErrors))
	}

	fmt.Printf("✅ Configuration is valid: %s\n", configFile)

	if verbose {
		fmt.Println("\nConfiguration summary:")
		fmt.Printf("  Output format: %s\n", cfg.Output)
		fmt.Printf("  Timeout: %s\n", cfg.Timeout)
		if cfg.ProjectID != "" {
			fmt.Printf("  Project ID: %s\n", cfg.ProjectID)
		}
		if cfg.ClusterName != "" {
			fmt.Printf("  Cluster name: %s\n", cfg.ClusterName)
		}
		fmt.Printf("  API key configured: %t\n", cfg.APIKey != "")
		fmt.Printf("  Public key configured: %t\n", cfg.PublicKey != "")
	}

	return nil
}

func validateWithSchema(configData []byte, schemaFile string) error {
	// Use the schema validator
	schemaValidator := validation.NewSchemaValidator()

	// If schemaFile is provided, load it
	if schemaFile != "" {
		err := schemaValidator.LoadSchema(schemaFile, "custom")
		if err != nil {
			return fmt.Errorf("failed to load schema: %w", err)
		}

		result, err := schemaValidator.ValidateConfigWithSchema(configData, "custom")
		if err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}

		if !result.Valid {
			var errorMessages []string
			for _, error := range result.Errors {
				errorMessages = append(errorMessages, error.Message)
			}
			return fmt.Errorf("schema validation errors: %s", strings.Join(errorMessages, "; "))
		}

		// Report warnings if any
		if len(result.Warnings) > 0 {
			fmt.Println("Schema validation warnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("  • %s\n", warning.Message)
			}
		}
	} else {
		// Use built-in matlas-config schema
		result, err := schemaValidator.ValidateConfigWithSchema(configData, "matlas-config")
		if err != nil {
			return fmt.Errorf("built-in schema validation failed: %w", err)
		}

		if !result.Valid {
			var errorMessages []string
			for _, error := range result.Errors {
				errorMessages = append(errorMessages, error.Message)
			}
			return fmt.Errorf("configuration schema errors: %s", strings.Join(errorMessages, "; "))
		}

		// Report warnings
		if len(result.Warnings) > 0 {
			fmt.Println("Configuration warnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("  • %s\n", warning.Message)
			}
		}
	}

	return nil
}

func runGenerateTemplate(cmd *cobra.Command, templateType, outputFile, format string) error {
	var template map[string]interface{}

	switch templateType {
	case "basic":
		template = generateBasicTemplate()
	case "atlas":
		template = generateAtlasTemplate()
	case "database":
		template = generateDatabaseTemplate()
	case "apply":
		template = generateApplyTemplate()
	case "complete":
		template = generateCompleteTemplate()
	default:
		return fmt.Errorf("unknown template type: %s. Available types: basic, atlas, database, apply, complete", templateType)
	}

	// Convert to requested format
	var output []byte
	var err error

	switch format {
	case "yaml", "yml":
		output, err = yaml.Marshal(template)
	case "json":
		output, err = json.MarshalIndent(template, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s. Supported formats: yaml, json", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	// Output to file or stdout
	if outputFile != "" {
		if err := os.WriteFile(outputFile, output, 0o600); err != nil {
			return fmt.Errorf("failed to write template to file: %w", err)
		}
		fmt.Printf("Template generated: %s\n", outputFile)
	} else {
		fmt.Print(string(output))
	}

	return nil
}

func runListTemplates(cmd *cobra.Command) error {
	templates := []map[string]string{
		{
			"name":        "basic",
			"description": "Basic CLI configuration with essential settings",
		},
		{
			"name":        "atlas",
			"description": "Atlas-specific configuration with API keys and project settings",
		},
		{
			"name":        "database",
			"description": "Database connection configuration for MongoDB operations",
		},
		{
			"name":        "apply",
			"description": "Resource configuration template for Atlas apply operations (NOT CLI config)",
		},
		{
			"name":        "complete",
			"description": "Complete configuration template with all available options",
		},
	}

	// Get output format from global flags or default to text
	cfg, _ := config.Load(cmd, "")
	if cfg == nil {
		cfg = config.New()
	}

	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, templates,
		[]string{"NAME", "DESCRIPTION"},
		func(item interface{}) []string {
			template := item.(map[string]string)
			return []string{
				template["name"],
				template["description"],
			}
		})
}

func runImportConfig(cmd *cobra.Command, sourceFile, targetFile, format string, merge bool) error {
	fmt.Printf("Importing configuration from: %s\n", sourceFile)

	// This is a placeholder for import functionality
	// In a real implementation, you would:
	// 1. Read the source file
	// 2. Parse it based on its format
	// 3. Convert to internal configuration format
	// 4. Merge with existing config if requested
	// 5. Write to target file

	return fmt.Errorf("import functionality not yet implemented")
}

func runExportConfig(cmd *cobra.Command, outputFile, format string, includeSecrets bool) error {
	fmt.Printf("Exporting configuration to format: %s\n", format)

	// This is a placeholder for export functionality
	// In a real implementation, you would:
	// 1. Load current configuration
	// 2. Convert to requested format
	// 3. Optionally exclude secrets
	// 4. Write to output file or stdout

	return fmt.Errorf("export functionality not yet implemented")
}

func runMigrateConfig(cmd *cobra.Command, fromVersion, toVersion string, backup bool) error {
	fmt.Printf("Migrating configuration from %s to %s\n", fromVersion, toVersion)

	// This is a placeholder for migration functionality
	// In a real implementation, you would:
	// 1. Detect current config version
	// 2. Create backup if requested
	// 3. Apply migration transformations
	// 4. Update config file

	return fmt.Errorf("migration functionality not yet implemented")
}

// Template generation functions

func generateBasicTemplate() map[string]interface{} {
	return map[string]interface{}{
		"output":  "text",
		"timeout": "30s",
	}
}

func generateAtlasTemplate() map[string]interface{} {
	return map[string]interface{}{
		"output":    "text",
		"timeout":   "30s",
		"projectId": "your-atlas-project-id",
		"apiKey":    "your-atlas-api-key",
		"publicKey": "your-atlas-public-key",
	}
}

func generateDatabaseTemplate() map[string]interface{} {
	return map[string]interface{}{
		"output":      "text",
		"timeout":     "30s",
		"projectId":   "your-atlas-project-id",
		"clusterName": "your-cluster-name",
	}
}

func generateApplyTemplate() map[string]interface{} {
	return map[string]interface{}{
		"# NOTE":     "This is a RESOURCE configuration template for MongoDB Atlas resources, NOT a CLI configuration file",
		"# Usage":    "Save this as a .yaml file and use with: matlas apply -f filename.yaml",
		"apiVersion": "matlas.mongodb.com/v1",
		"kind":       "ApplyDocument",
		"metadata": map[string]interface{}{
			"name": "example-project",
			"labels": map[string]string{
				"environment": "development",
				"team":        "your-team",
			},
			"annotations": map[string]string{
				"description": "Example MongoDB Atlas project configuration",
			},
		},
		"spec": map[string]interface{}{
			"name": "example-project",
			"clusters": []map[string]interface{}{
				{
					"metadata": map[string]interface{}{
						"name": "example-cluster",
						"labels": map[string]string{
							"environment": "development",
							"tier":        "basic",
						},
					},
					"provider":       "AWS",
					"region":         "US_EAST_1",
					"instanceSize":   "M10",
					"diskSizeGB":     10,
					"backupEnabled":  true,
					"mongodbVersion": "7.0",
					"clusterType":    "REPLICASET",
				},
			},
			"databaseUsers": []map[string]interface{}{
				{
					"metadata": map[string]interface{}{
						"name": "app-user",
						"labels": map[string]string{
							"purpose": "application",
						},
					},
					"username":     "app-user",
					"authDatabase": "admin",
					"password":     "${APP_PASSWORD}",
					"roles": []map[string]interface{}{
						{
							"roleName":     "readWrite",
							"databaseName": "app-database",
						},
					},
				},
			},
			"databaseRoles": []map[string]interface{}{
				{
					"metadata": map[string]interface{}{
						"name": "custom-app-role",
						"labels": map[string]string{
							"purpose": "application",
						},
					},
					"roleName":     "appDataRole",
					"databaseName": "app-database",
					"privileges": []map[string]interface{}{
						{
							"actions": []string{"find", "insert", "update", "remove"},
							"resource": map[string]interface{}{
								"database":   "app-database",
								"collection": "users",
							},
						},
						{
							"actions": []string{"find"},
							"resource": map[string]interface{}{
								"database": "logs",
							},
						},
					},
					"inheritedRoles": []map[string]interface{}{
						{
							"roleName":     "read",
							"databaseName": "reference-data",
						},
					},
				},
			},
			"networkAccess": []map[string]interface{}{
				{
					"metadata": map[string]interface{}{
						"name": "office-access",
						"labels": map[string]string{
							"type": "office",
						},
					},
					"ipAddress": "203.0.113.0/24",
					"comment":   "Office network access",
				},
			},
		},
	}
}

func generateCompleteTemplate() map[string]interface{} {
	return map[string]interface{}{
		"# CLI Configuration": nil,
		"output":              "text", // text, json, yaml
		"timeout":             "30s",  // timeout for API operations

		"# Atlas Configuration": nil,
		"projectId":             "your-atlas-project-id",
		"clusterName":           "your-default-cluster-name",
		"apiKey":                "your-atlas-api-key",
		"publicKey":             "your-atlas-public-key",
	}
}
