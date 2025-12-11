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
	"github.com/teabranch/matlas-cli/internal/fileutil"
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
  matlas config template generate atlas --file atlas-config.yaml

  # Generate in JSON format
  matlas config template generate complete --format json --file config.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			templateType = args[0]
			return runGenerateTemplate(cmd, templateType, outputFile, format)
		},
	}

	cmd.Flags().StringVar(&outputFile, "file", "", "Output file path (default: stdout)")
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
		Long: `Import configuration from external files or formats.

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

	return cmd
}

func newExportCmd() *cobra.Command {
	var outputFile string
	var format string
	var includeSecrets bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export configuration to external formats",
		Long: `Export matlas-cli configuration to external formats.

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

	return cmd
}

func newMigrateCmd() *cobra.Command {
	var fromVersion string
	var toVersion string
	var backup bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate configuration between versions",
		Long: `Migrate configuration files between different matlas-cli versions.

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

	// Validate source file exists
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", sourceFile)
	}

	// Read source file
	sourceData, err := os.ReadFile(sourceFile) // #nosec G304 -- sourceFile is validated user input
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Detect format if not specified
	if format == "" {
		format = detectFileFormat(sourceFile, sourceData)
	}

	// Parse source data into configuration map
	var sourceConfig map[string]interface{}
	switch strings.ToLower(format) {
	case "yaml", "yml":
		if err := yaml.Unmarshal(sourceData, &sourceConfig); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	case "json":
		if err := json.Unmarshal(sourceData, &sourceConfig); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	case "env":
		sourceConfig = parseEnvFile(sourceData)
	default:
		return fmt.Errorf("unsupported source format: %s. Supported formats: yaml, json, env", format)
	}

	// Convert to standard config format
	normalizedConfig := normalizeConfigKeys(sourceConfig)

	// Determine target file path
	if targetFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		targetFile = filepath.Join(homeDir, config.DefaultConfigDir, "config.yaml")
	}

	var finalConfig map[string]interface{}

	// Handle merge if requested
	if merge {
		// Load existing config if it exists
		existingConfig := make(map[string]interface{})
		if _, err := os.Stat(targetFile); err == nil {
			existingData, err := os.ReadFile(targetFile) // #nosec G304 -- targetFile is controlled path
			if err != nil {
				return fmt.Errorf("failed to read existing config: %w", err)
			}
			if err := yaml.Unmarshal(existingData, &existingConfig); err != nil {
				return fmt.Errorf("failed to parse existing config: %w", err)
			}
		}

		// Merge configurations (source overwrites existing)
		finalConfig = mergeConfigs(existingConfig, normalizedConfig)
		fmt.Printf("Merging with existing configuration at: %s\n", targetFile)
	} else {
		finalConfig = normalizedConfig
	}

	// Convert to YAML
	outputData, err := yaml.Marshal(finalConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// SECURITY: Write file with secure permissions
	writer := fileutil.NewSecureFileWriter()
	if err := writer.WriteFile(targetFile, outputData); err != nil {
		return fmt.Errorf("failed to write target file: %w", err)
	}

	fmt.Printf("✅ Configuration imported successfully to: %s\n", targetFile)
	if merge {
		fmt.Printf("✅ Merged %d source keys with existing configuration\n", len(normalizedConfig))
	} else {
		fmt.Printf("✅ Imported %d configuration keys\n", len(normalizedConfig))
	}

	return nil
}

func runExportConfig(cmd *cobra.Command, outputFile, format string, includeSecrets bool) error {
	fmt.Printf("Exporting configuration to format: %s\n", format)

	// Load current configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Convert config to exportable map
	configMap := configToMap(cfg, includeSecrets)

	var output []byte
	switch strings.ToLower(format) {
	case "yaml", "yml":
		output, err = yaml.Marshal(configMap)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
	case "json":
		output, err = json.MarshalIndent(configMap, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	case "env":
		output = []byte(convertToEnvFormat(configMap, includeSecrets))
	case "shell":
		output = []byte(convertToShellExportFormat(configMap, includeSecrets))
	default:
		return fmt.Errorf("unsupported export format: %s. Supported formats: yaml, json, env, shell", format)
	}

	// Write to file or stdout
	if outputFile != "" {
		// SECURITY: Write file with secure permissions
		writer := fileutil.NewSecureFileWriter()
		if err := writer.WriteFile(outputFile, output); err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}
		fmt.Printf("✅ Configuration exported successfully to: %s\n", outputFile)
		if !includeSecrets {
			fmt.Printf("🔒 Sensitive values (API keys) were excluded. Use --include-secrets to include them.\n")
		}
	} else {
		fmt.Print(string(output))
	}

	return nil
}

func runMigrateConfig(cmd *cobra.Command, fromVersion, toVersion string, backup bool) error {
	// Determine config file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configFile := filepath.Join(homeDir, config.DefaultConfigDir, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configFile)
	}

	// Read current config
	configData, err := os.ReadFile(configFile) // #nosec G304 -- configFile is user-specified config path
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse current config
	var currentConfig map[string]interface{}
	if err := yaml.Unmarshal(configData, &currentConfig); err != nil {
		return fmt.Errorf("failed to parse current configuration: %w", err)
	}

	// Detect current version if not specified
	if fromVersion == "" {
		fromVersion = detectConfigVersion(currentConfig)
		fmt.Printf("Detected current config version: %s\n", fromVersion)
	}

	// Resolve target version
	if toVersion == "latest" {
		toVersion = "v2.0.0" // Current latest version
	}

	fmt.Printf("Migrating configuration from %s to %s\n", fromVersion, toVersion)

	// Check if migration is needed
	if fromVersion == toVersion {
		fmt.Printf("✅ Configuration is already at version %s, no migration needed\n", toVersion)
		return nil
	}

	// Create backup if requested
	if backup {
		backupFile := configFile + ".backup." + strings.ReplaceAll(fromVersion, ".", "_")
		// SECURITY: Write backup with secure permissions
		writer := fileutil.NewSecureFileWriter()
		if err := writer.WriteFile(backupFile, configData); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Printf("📁 Backup created: %s\n", backupFile)
	}

	// Apply migrations
	migratedConfig, err := applyMigrations(currentConfig, fromVersion, toVersion)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Write migrated config
	migratedData, err := yaml.Marshal(migratedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal migrated configuration: %w", err)
	}

	// SECURITY: Write file with secure permissions
	writer := fileutil.NewSecureFileWriter()
	if err := writer.WriteFile(configFile, migratedData); err != nil {
		return fmt.Errorf("failed to write migrated configuration: %w", err)
	}

	fmt.Printf("✅ Configuration migrated successfully to version %s\n", toVersion)
	fmt.Printf("📝 Configuration file updated: %s\n", configFile)

	return nil
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

// Helper functions for config import functionality

func detectFileFormat(filename string, data []byte) string {
	// First try to detect by file extension
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".env":
		return "env"
	}

	// Try to detect by content if extension doesn't help
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "json"
	}
	if strings.Contains(trimmed, "=") && !strings.Contains(trimmed, ":") {
		return "env"
	}
	// Default to YAML for structured data
	return "yaml"
}

func parseEnvFile(data []byte) map[string]interface{} {
	config := make(map[string]interface{})
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		// Convert common Atlas environment variables to config keys
		switch key {
		case "ATLAS_PROJECT_ID":
			config["projectId"] = value
		case "ATLAS_ORG_ID":
			config["orgId"] = value
		case "ATLAS_CLUSTER_NAME":
			config["clusterName"] = value
		case "ATLAS_API_KEY":
			config["apiKey"] = value
		case "ATLAS_PUB_KEY", "ATLAS_PUBLIC_KEY":
			config["publicKey"] = value
		case "ATLAS_OUTPUT":
			config["output"] = value
		case "ATLAS_TIMEOUT":
			config["timeout"] = value
		default:
			// For other variables, try to normalize the name
			normalKey := strings.ToLower(strings.TrimPrefix(key, "ATLAS_"))
			normalKey = strings.ReplaceAll(normalKey, "_", "")
			if normalKey != "" {
				config[normalKey] = value
			}
		}
	}

	return config
}

func normalizeConfigKeys(config map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{})

	for key, value := range config {
		// Convert various key formats to standard camelCase
		normalKey := key
		switch strings.ToLower(key) {
		case "project_id", "project-id", "projectid":
			normalKey = "projectId"
		case "org_id", "org-id", "orgid":
			normalKey = "orgId"
		case "cluster_name", "cluster-name", "clustername":
			normalKey = "clusterName"
		case "api_key", "api-key", "apikey":
			normalKey = "apiKey"
		case "public_key", "public-key", "pub_key", "pub-key", "publickey":
			normalKey = "publicKey"
		}

		normalized[normalKey] = value
	}

	return normalized
}

func mergeConfigs(existing, source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy existing config
	for key, value := range existing {
		result[key] = value
	}

	// Overlay source config (source wins conflicts)
	for key, value := range source {
		result[key] = value
	}

	return result
}

// Helper functions for config export functionality

func configToMap(cfg *config.Config, includeSecrets bool) map[string]interface{} {
	configMap := make(map[string]interface{})

	// Add non-empty values
	if cfg.ProjectID != "" {
		configMap["projectId"] = cfg.ProjectID
	}
	if cfg.OrgID != "" {
		configMap["orgId"] = cfg.OrgID
	}
	if cfg.ClusterName != "" {
		configMap["clusterName"] = cfg.ClusterName
	}
	if cfg.Output != "" && cfg.Output != "table" { // Don't export default value
		configMap["output"] = string(cfg.Output)
	}
	if cfg.Timeout != config.DefaultTimeout {
		configMap["timeout"] = cfg.Timeout.String()
	}

	// Handle secrets based on includeSecrets flag
	if includeSecrets {
		if cfg.APIKey != "" {
			configMap["apiKey"] = cfg.APIKey
		}
		if cfg.PublicKey != "" {
			configMap["publicKey"] = cfg.PublicKey
		}
	} else {
		// Show masked values if secrets exist
		if cfg.APIKey != "" {
			configMap["# apiKey"] = "[REDACTED - use --include-secrets to export]"
		}
		if cfg.PublicKey != "" {
			configMap["# publicKey"] = "[REDACTED - use --include-secrets to export]"
		}
	}

	return configMap
}

func convertToEnvFormat(configMap map[string]interface{}, includeSecrets bool) string {
	var lines []string
	lines = append(lines, "# matlas-cli configuration exported as environment variables")
	lines = append(lines, "# Source this file with: source config.env")
	lines = append(lines, "")

	// Map config keys to environment variable names
	envMap := map[string]string{
		"projectId":   "ATLAS_PROJECT_ID",
		"orgId":       "ATLAS_ORG_ID",
		"clusterName": "ATLAS_CLUSTER_NAME",
		"output":      "ATLAS_OUTPUT",
		"timeout":     "ATLAS_TIMEOUT",
		"apiKey":      "ATLAS_API_KEY",
		"publicKey":   "ATLAS_PUB_KEY",
	}

	for key, value := range configMap {
		// Skip comment keys (redacted secrets)
		if strings.HasPrefix(key, "#") {
			if includeSecrets {
				continue // Skip comment for redacted values when including secrets
			}
			lines = append(lines, fmt.Sprintf("# %s", value))
			continue
		}

		if envVar, exists := envMap[key]; exists {
			lines = append(lines, fmt.Sprintf("%s=%s", envVar, value))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func convertToShellExportFormat(configMap map[string]interface{}, includeSecrets bool) string {
	var lines []string
	lines = append(lines, "# matlas-cli configuration as shell export statements")
	lines = append(lines, "# Run this with: eval $(matlas config export --format shell)")
	lines = append(lines, "")

	// Map config keys to environment variable names
	envMap := map[string]string{
		"projectId":   "ATLAS_PROJECT_ID",
		"orgId":       "ATLAS_ORG_ID",
		"clusterName": "ATLAS_CLUSTER_NAME",
		"output":      "ATLAS_OUTPUT",
		"timeout":     "ATLAS_TIMEOUT",
		"apiKey":      "ATLAS_API_KEY",
		"publicKey":   "ATLAS_PUB_KEY",
	}

	for key, value := range configMap {
		// Skip comment keys (redacted secrets)
		if strings.HasPrefix(key, "#") {
			if includeSecrets {
				continue // Skip comment for redacted values when including secrets
			}
			lines = append(lines, fmt.Sprintf("# %s", value))
			continue
		}

		if envVar, exists := envMap[key]; exists {
			// Properly quote values that might contain special characters
			quotedValue := fmt.Sprintf("\"%s\"", strings.ReplaceAll(fmt.Sprintf("%v", value), "\"", "\\\""))
			lines = append(lines, fmt.Sprintf("export %s=%s", envVar, quotedValue))
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

// Helper functions for config migration functionality

func detectConfigVersion(config map[string]interface{}) string {
	// Check for version field in config
	if version, exists := config["version"]; exists {
		return fmt.Sprintf("%v", version)
	}

	// Check for schema indicators to detect version

	// v2.0.0+ indicators (current format)
	if _, hasProjectId := config["projectId"]; hasProjectId {
		// Check for new camelCase format
		if _, hasOrgId := config["orgId"]; hasOrgId {
			return "v2.0.0"
		}
		return "v1.5.0"
	}

	// v1.0.0 indicators (legacy format with underscores)
	if _, hasProjectId := config["project_id"]; hasProjectId {
		return "v1.0.0"
	}

	// Pre-v1.0.0 (very basic config)
	if len(config) > 0 {
		return "v0.9.0"
	}

	// Empty or new config
	return "v2.0.0"
}

func applyMigrations(config map[string]interface{}, fromVersion, toVersion string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Copy original config
	for k, v := range config {
		result[k] = v
	}

	// Define migration path
	migrations := []migration{
		{from: "v0.9.0", to: "v1.0.0", transform: migrateV0_9ToV1_0},
		{from: "v1.0.0", to: "v1.5.0", transform: migrateV1_0ToV1_5},
		{from: "v1.5.0", to: "v2.0.0", transform: migrateV1_5ToV2_0},
	}

	// Apply sequential migrations
	currentVersion := fromVersion
	for _, mig := range migrations {
		if shouldApplyMigration(currentVersion, toVersion, mig.from, mig.to) {
			fmt.Printf("  Applying migration: %s → %s\n", mig.from, mig.to)
			var err error
			result, err = mig.transform(result)
			if err != nil {
				return nil, fmt.Errorf("migration %s → %s failed: %w", mig.from, mig.to, err)
			}
			currentVersion = mig.to
		}
	}

	// Add version field to migrated config
	result["version"] = toVersion

	return result, nil
}

type migration struct {
	from      string
	to        string
	transform func(map[string]interface{}) (map[string]interface{}, error)
}

func shouldApplyMigration(currentVersion, targetVersion, migrationFrom, migrationTo string) bool {
	// Apply migration if:
	// 1. Current version is at or past the migration starting point
	// 2. Target version includes this migration step
	// 3. Current version is before the migration endpoint (to avoid applying migrations we've already done)
	return versionLessOrEqual(migrationFrom, currentVersion) &&
		versionLessOrEqual(migrationTo, targetVersion) &&
		versionLess(currentVersion, migrationTo)
}

func versionLessOrEqual(v1, v2 string) bool {
	// Simple string comparison for demo - use proper semantic versioning in production
	return v1 <= v2
}

func versionLess(v1, v2 string) bool {
	// Simple string comparison for demo - use proper semantic versioning in production
	return v1 < v2
}

// Migration transformations

func migrateV0_9ToV1_0(config map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Copy all existing values
	for k, v := range config {
		result[k] = v
	}

	// Add default values that were introduced in v1.0.0
	if _, exists := result["timeout"]; !exists {
		result["timeout"] = "30s"
	}
	if _, exists := result["output"]; !exists {
		result["output"] = "text"
	}

	return result, nil
}

func migrateV1_0ToV1_5(config map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Migrate from snake_case to camelCase (partial migration)
	keyMappings := map[string]string{
		"project_id":   "projectId",
		"org_id":       "orgId",
		"cluster_name": "clusterName",
		"api_key":      "apiKey",
		"public_key":   "publicKey",
	}

	for oldKey, newKey := range keyMappings {
		if value, exists := config[oldKey]; exists {
			result[newKey] = value
		}
	}

	// Copy other values as-is
	for k, v := range config {
		if _, isMapped := keyMappings[k]; !isMapped {
			result[k] = v
		}
	}

	return result, nil
}

func migrateV1_5ToV2_0(config map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Copy all existing values (v1.5.0 → v2.0.0 is mostly compatible)
	for k, v := range config {
		result[k] = v
	}

	// Ensure all keys are in proper camelCase format
	result = normalizeConfigKeys(result)

	// Remove any deprecated keys
	delete(result, "deprecated_field") // Example - remove any deprecated fields

	return result, nil
}
