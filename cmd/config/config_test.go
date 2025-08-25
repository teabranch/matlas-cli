package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/teabranch/matlas-cli/internal/config"
)

func TestNewConfigCmd(t *testing.T) {
	cmd := NewConfigCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "config", cmd.Use)
	assert.Equal(t, "Manage CLI configuration", cmd.Short)
	assert.Contains(t, cmd.Aliases, "cfg")
	assert.True(t, cmd.SilenceUsage)

	// Check that subcommands are added
	subcommands := cmd.Commands()
	assert.True(t, len(subcommands) > 0)

	// Verify specific subcommands exist
	subcommandNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		subcommandNames[i] = subcmd.Use
	}

	assert.Contains(t, subcommandNames, "validate [config-file]")
	assert.Contains(t, subcommandNames, "template")
	assert.Contains(t, subcommandNames, "import <source-file>")
	assert.Contains(t, subcommandNames, "export")
	assert.Contains(t, subcommandNames, "migrate")
}

func TestNewValidateCmd(t *testing.T) {
	cmd := newValidateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "validate [config-file]", cmd.Use)
	assert.Equal(t, "Validate configuration files", cmd.Short)
	assert.Contains(t, cmd.Long, "YAML syntax errors")
	assert.Contains(t, cmd.Example, "matlas config validate")

	// Check flags
	configFlag := cmd.Flags().Lookup("config")
	assert.NotNil(t, configFlag)
	assert.Equal(t, "Configuration file path", configFlag.Usage)

	schemaFlag := cmd.Flags().Lookup("schema")
	assert.NotNil(t, schemaFlag)
	assert.Equal(t, "JSON schema file for validation", schemaFlag.Usage)

	verboseFlag := cmd.Flags().Lookup("verbose")
	assert.NotNil(t, verboseFlag)
	assert.Equal(t, "v", verboseFlag.Shorthand)
}

func TestNewTemplateCmd(t *testing.T) {
	cmd := newTemplateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "template", cmd.Use)
	assert.Equal(t, "Generate configuration templates", cmd.Short)
	assert.Contains(t, cmd.Long, "Generate configuration file templates for different use cases")

	// Test template has subcommands
	subcommands := cmd.Commands()
	assert.True(t, len(subcommands) >= 2) // generate and list
}

func TestNewImportCmd(t *testing.T) {
	cmd := newImportCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "import <source-file>", cmd.Use)
	assert.Equal(t, "Import configuration from external sources", cmd.Short)
	assert.Contains(t, cmd.Long, "Import configuration from external files or formats")

	// Check flags
	targetFlag := cmd.Flags().Lookup("target")
	assert.NotNil(t, targetFlag)

	formatFlag := cmd.Flags().Lookup("format")
	assert.NotNil(t, formatFlag)
}

func TestNewExportCmd(t *testing.T) {
	cmd := newExportCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "export", cmd.Use)
	assert.Equal(t, "Export configuration to external formats", cmd.Short)
	assert.Contains(t, cmd.Long, "Export matlas-cli configuration to external formats")

	// Check flags
	formatFlag := cmd.Flags().Lookup("format")
	assert.NotNil(t, formatFlag)

	outputFlag := cmd.Flags().Lookup("output")
	assert.NotNil(t, outputFlag)
}

func TestNewMigrateCmd(t *testing.T) {
	cmd := newMigrateCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "migrate", cmd.Use)
	assert.Equal(t, "Migrate configuration between versions", cmd.Short)
	assert.Contains(t, cmd.Long, "Migrate configuration files")

	// Check flags
	fromFlag := cmd.Flags().Lookup("from")
	assert.NotNil(t, fromFlag)

	toFlag := cmd.Flags().Lookup("to")
	assert.NotNil(t, toFlag)
}

func TestValidateCmd_Help(t *testing.T) {
	cmd := NewConfigCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"validate", "--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Validate matlas-cli configuration files")
	assert.Contains(t, helpOutput, "--config")
	assert.Contains(t, helpOutput, "--schema")
	assert.Contains(t, helpOutput, "--verbose")
}

func TestTemplateCmd_Help(t *testing.T) {
	cmd := NewConfigCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)

	// Test help command
	cmd.SetArgs([]string{"template", "--help"})
	err := cmd.Execute()

	assert.NoError(t, err)
	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Generate configuration file templates for different use cases")
}

// Note: Internal helper functions like getConfigType, validateConfigFile, etc.
// are not exposed, so we focus on testing the command structure and public API

func TestCommandStructure(t *testing.T) {
	// Test that commands have proper structure and don't panic
	cmd := NewConfigCmd()

	// Test that we can get help without errors
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	assert.NoError(t, err)

	helpOutput := output.String()
	assert.Contains(t, helpOutput, "config")
	assert.Contains(t, helpOutput, "Manage matlas-cli configuration files and settings")
}

func TestSubcommandHelp(t *testing.T) {
	cmd := NewConfigCmd()
	subcommands := []string{"validate", "template", "import", "export", "migrate"}

	for _, subcmd := range subcommands {
		t.Run(subcmd, func(t *testing.T) {
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetArgs([]string{subcmd, "--help"})

			// Reset command state
			cmd.SetArgs([]string{subcmd, "--help"})

			err := cmd.Execute()
			assert.NoError(t, err)

			helpOutput := output.String()
			assert.NotEmpty(t, helpOutput)
		})
	}
}

// Tests for config import/export/migrate functionality

func TestDetectFileFormat(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		expected string
	}{
		{
			name:     "JSON by extension",
			filename: "config.json",
			content:  `{"key": "value"}`,
			expected: "json",
		},
		{
			name:     "YAML by extension",
			filename: "config.yaml",
			content:  `key: value`,
			expected: "yaml",
		},
		{
			name:     "ENV by extension",
			filename: "config.env",
			content:  `KEY=value`,
			expected: "env",
		},
		{
			name:     "JSON by content",
			filename: "config",
			content:  `{"key": "value"}`,
			expected: "json",
		},
		{
			name:     "ENV by content",
			filename: "config",
			content:  `KEY=value\nANOTHER=test`,
			expected: "env",
		},
		{
			name:     "YAML by default",
			filename: "config",
			content:  `key: value\nother: test`,
			expected: "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectFileFormat(tt.filename, []byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]interface{}
	}{
		{
			name: "Basic env file",
			content: `ATLAS_PROJECT_ID=12345
ATLAS_API_KEY=secret123`,
			expected: map[string]interface{}{
				"projectId": "12345",
				"apiKey":    "secret123",
			},
		},
		{
			name: "Env file with quotes",
			content: `ATLAS_PROJECT_ID="12345"
ATLAS_CLUSTER_NAME='my-cluster'`,
			expected: map[string]interface{}{
				"projectId":   "12345",
				"clusterName": "my-cluster",
			},
		},
		{
			name: "Env file with comments",
			content: `# This is a comment
ATLAS_PROJECT_ID=12345
# Another comment
ATLAS_API_KEY=secret123`,
			expected: map[string]interface{}{
				"projectId": "12345",
				"apiKey":    "secret123",
			},
		},
		{
			name: "Env file with various formats",
			content: `ATLAS_PUB_KEY=public123
ATLAS_PUBLIC_KEY=public456
ATLAS_OUTPUT=json`,
			expected: map[string]interface{}{
				"publicKey": "public456", // ATLAS_PUBLIC_KEY overwrites ATLAS_PUB_KEY
				"output":    "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEnvFile([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeConfigKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Snake case to camel case",
			input: map[string]interface{}{
				"project_id":   "12345",
				"cluster_name": "my-cluster",
				"api_key":      "secret",
			},
			expected: map[string]interface{}{
				"projectId":   "12345",
				"clusterName": "my-cluster",
				"apiKey":      "secret",
			},
		},
		{
			name: "Kebab case to camel case",
			input: map[string]interface{}{
				"project-id":   "12345",
				"cluster-name": "my-cluster",
				"public-key":   "public",
			},
			expected: map[string]interface{}{
				"projectId":   "12345",
				"clusterName": "my-cluster",
				"publicKey":   "public",
			},
		},
		{
			name: "Mixed formats",
			input: map[string]interface{}{
				"projectid":    "12345",
				"cluster_name": "my-cluster",
				"api-key":      "secret",
				"output":       "json", // Already correct
			},
			expected: map[string]interface{}{
				"projectId":   "12345",
				"clusterName": "my-cluster",
				"apiKey":      "secret",
				"output":      "json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeConfigKeys(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeConfigs(t *testing.T) {
	existing := map[string]interface{}{
		"projectId": "original-project",
		"output":    "text",
		"timeout":   "30s",
	}

	source := map[string]interface{}{
		"projectId":   "new-project",
		"clusterName": "new-cluster",
	}

	expected := map[string]interface{}{
		"projectId":   "new-project", // Source overwrites existing
		"output":      "text",        // Preserved from existing
		"timeout":     "30s",         // Preserved from existing
		"clusterName": "new-cluster", // Added from source
	}

	result := mergeConfigs(existing, source)
	assert.Equal(t, expected, result)
}

func TestConfigToMap(t *testing.T) {
	cfg := &config.Config{
		ProjectID:   "12345",
		ClusterName: "my-cluster",
		Output:      "json",
		Timeout:     45 * time.Second,
		APIKey:      "secret123",
		PublicKey:   "public456",
	}

	t.Run("Include secrets", func(t *testing.T) {
		result := configToMap(cfg, true)
		expected := map[string]interface{}{
			"projectId":   "12345",
			"clusterName": "my-cluster",
			"output":      "json",
			"timeout":     "45s",
			"apiKey":      "secret123",
			"publicKey":   "public456",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("Exclude secrets", func(t *testing.T) {
		result := configToMap(cfg, false)
		expected := map[string]interface{}{
			"projectId":    "12345",
			"clusterName":  "my-cluster",
			"output":       "json",
			"timeout":      "45s",
			"# apiKey":     "[REDACTED - use --include-secrets to export]",
			"# publicKey":  "[REDACTED - use --include-secrets to export]",
		}
		assert.Equal(t, expected, result)
	})
}

func TestConvertToEnvFormat(t *testing.T) {
	configMap := map[string]interface{}{
		"projectId":   "12345",
		"clusterName": "my-cluster",
		"output":      "json",
		"apiKey":      "secret123",
	}

	result := convertToEnvFormat(configMap, true)
	
	assert.Contains(t, result, "ATLAS_PROJECT_ID=12345")
	assert.Contains(t, result, "ATLAS_CLUSTER_NAME=my-cluster")
	assert.Contains(t, result, "ATLAS_OUTPUT=json")
	assert.Contains(t, result, "ATLAS_API_KEY=secret123")
	assert.Contains(t, result, "# matlas-cli configuration exported as environment variables")
}

func TestConvertToShellExportFormat(t *testing.T) {
	configMap := map[string]interface{}{
		"projectId": "12345",
		"output":    "json",
	}

	result := convertToShellExportFormat(configMap, true)
	
	assert.Contains(t, result, `export ATLAS_PROJECT_ID="12345"`)
	assert.Contains(t, result, `export ATLAS_OUTPUT="json"`)
	assert.Contains(t, result, "# matlas-cli configuration as shell export statements")
}

func TestDetectConfigVersion(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected string
	}{
		{
			name: "Explicit version",
			config: map[string]interface{}{
				"version":   "v1.5.0",
				"projectId": "12345",
			},
			expected: "v1.5.0",
		},
		{
			name: "v2.0.0 format (camelCase with orgId)",
			config: map[string]interface{}{
				"projectId": "12345",
				"orgId":     "67890",
			},
			expected: "v2.0.0",
		},
		{
			name: "v1.5.0 format (camelCase without orgId)",
			config: map[string]interface{}{
				"projectId": "12345",
				"output":    "json",
			},
			expected: "v1.5.0",
		},
		{
			name: "v1.0.0 format (snake_case)",
			config: map[string]interface{}{
				"project_id": "12345",
				"output":     "json",
			},
			expected: "v1.0.0",
		},
		{
			name: "v0.9.0 format (basic)",
			config: map[string]interface{}{
				"output": "json",
			},
			expected: "v0.9.0",
		},
		{
			name:     "Empty config",
			config:   map[string]interface{}{},
			expected: "v2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectConfigVersion(tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestApplyMigrations(t *testing.T) {
	t.Run("Migration v1.0.0 to v2.0.0", func(t *testing.T) {
		input := map[string]interface{}{
			"project_id":   "12345",
			"cluster_name": "my-cluster",
			"output":       "text",
		}

		result, err := applyMigrations(input, "v1.0.0", "v2.0.0")
		require.NoError(t, err)

		expected := map[string]interface{}{
			"projectId":   "12345",
			"clusterName": "my-cluster",
			"output":      "text",
			"version":     "v2.0.0",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("Same version migration", func(t *testing.T) {
		input := map[string]interface{}{
			"projectId": "12345",
			"output":    "json",
		}

		result, err := applyMigrations(input, "v2.0.0", "v2.0.0")
		require.NoError(t, err)

		expected := map[string]interface{}{
			"projectId": "12345",
			"output":    "json",
			"version":   "v2.0.0",
		}
		assert.Equal(t, expected, result)
	})
}

func TestMigrationTransformations(t *testing.T) {
	t.Run("migrateV0_9ToV1_0", func(t *testing.T) {
		input := map[string]interface{}{
			"output": "json",
		}

		result, err := migrateV0_9ToV1_0(input)
		require.NoError(t, err)

		assert.Equal(t, "json", result["output"])
		assert.Equal(t, "30s", result["timeout"])
	})

	t.Run("migrateV1_0ToV1_5", func(t *testing.T) {
		input := map[string]interface{}{
			"project_id":   "12345",
			"cluster_name": "my-cluster",
			"api_key":      "secret",
			"output":       "json",
		}

		result, err := migrateV1_0ToV1_5(input)
		require.NoError(t, err)

		expected := map[string]interface{}{
			"projectId":   "12345",
			"clusterName": "my-cluster",
			"apiKey":      "secret",
			"output":      "json",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("migrateV1_5ToV2_0", func(t *testing.T) {
		input := map[string]interface{}{
			"projectId":       "12345",
			"cluster-name":    "my-cluster", // Mixed format
			"deprecated_field": "should-be-removed",
		}

		result, err := migrateV1_5ToV2_0(input)
		require.NoError(t, err)

		assert.Equal(t, "12345", result["projectId"])
		assert.Equal(t, "my-cluster", result["clusterName"]) // Normalized
		assert.NotContains(t, result, "deprecated_field")    // Removed
	})
}

// Integration test for config import functionality
func TestConfigImportIntegration(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "config-test-*")
	require.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: failed to clean up temp directory: %v", err)
		}
	}()

	t.Run("Import from ENV file", func(t *testing.T) {
		// Create source env file
		envContent := `ATLAS_PROJECT_ID=test-project-123
ATLAS_CLUSTER_NAME=test-cluster
ATLAS_OUTPUT=json
# This is a comment
ATLAS_API_KEY=secret-key-123`

		sourceFile := filepath.Join(tempDir, "test.env")
		err := os.WriteFile(sourceFile, []byte(envContent), 0o600)
		require.NoError(t, err)

		// Test import functionality (we can't run the full command easily in unit tests,
		// but we can test the core logic)
		sourceData, err := os.ReadFile(sourceFile) // #nosec G304 -- sourceFile is test-controlled path
		require.NoError(t, err)

		format := detectFileFormat(sourceFile, sourceData)
		assert.Equal(t, "env", format)

		sourceConfig := parseEnvFile(sourceData)
		normalizedConfig := normalizeConfigKeys(sourceConfig)

		expected := map[string]interface{}{
			"projectId":   "test-project-123",
			"clusterName": "test-cluster",
			"output":      "json",
			"apiKey":      "secret-key-123",
		}
		assert.Equal(t, expected, normalizedConfig)
	})
}
