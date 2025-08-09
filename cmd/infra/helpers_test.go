package infra

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

// Define LoadResult type alias for testing
type LoadResult = apply.LoadResult

func TestExpandFilePatternsAdvanced(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := ioutil.TempDir("", "apply-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{
		"config1.yaml",
		"config2.yml",
		"config3.json",
		"subdir/nested.yaml",
		"other.txt",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = ioutil.WriteFile(fullPath, []byte("test content"), 0644)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		patterns      []string
		expectedCount int
		shouldContain []string
		expectError   bool
	}{
		{
			name:          "single file",
			patterns:      []string{filepath.Join(tempDir, "config1.yaml")},
			expectedCount: 1,
			shouldContain: []string{"config1.yaml"},
		},
		{
			name:          "glob pattern for yaml files",
			patterns:      []string{filepath.Join(tempDir, "*.yaml")},
			expectedCount: 1,
			shouldContain: []string{"config1.yaml"},
		},
		{
			name:          "stdin",
			patterns:      []string{"-"},
			expectedCount: 1,
			shouldContain: []string{"-"},
		},
		{
			name:        "non-existent file",
			patterns:    []string{filepath.Join(tempDir, "non-existent.yaml")},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandFilePatterns(tt.patterns)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.expectedCount)

			// Check that expected files are present
			for _, expected := range tt.shouldContain {
				found := false
				for _, actual := range result {
					if strings.Contains(actual, expected) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find %s in result", expected)
			}
		})
	}
}

func TestGetProjectIDHelper(t *testing.T) {
	tests := []struct {
		name     string
		configs  []*LoadResult
		opts     *ApplyOptions
		expected string
	}{
		{
			name: "project ID from options",
			configs: []*LoadResult{
				{Config: &types.ApplyConfig{Spec: types.ProjectConfig{Name: "test-project"}}},
			},
			opts:     &ApplyOptions{ProjectID: "override-project"},
			expected: "override-project",
		},
		{
			name: "project ID from config",
			configs: []*LoadResult{
				{Config: &types.ApplyConfig{Spec: types.ProjectConfig{Name: "test-project"}}},
			},
			opts:     &ApplyOptions{},
			expected: "test-project",
		},
		{
			name: "project ID from Project kind config",
			configs: []*LoadResult{
				{Config: &types.ApplyConfig{
					Kind: "Project",
					Spec: types.ProjectConfig{Name: "project-from-manifest"},
				}},
			},
			opts:     &ApplyOptions{},
			expected: "project-from-manifest",
		},
		{
			name:     "no project ID available",
			configs:  []*LoadResult{},
			opts:     &ApplyOptions{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getProjectID(tt.configs, tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigurationLoadingErrorScenarios(t *testing.T) {
	// Create temporary directory
	tempDir, err := ioutil.TempDir("", "apply-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		fileContent string
		fileName    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid YAML config",
			fileContent: `
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-project
spec:
  name: test-project
  organizationId: "507f1f77bcf86cd799439011"
`,
			fileName:    "valid.yaml",
			expectError: false,
		},
		{
			name: "invalid YAML syntax",
			fileContent: `
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: test-project
spec:
  name: test-project
  organizationId: 507f1f77bcf86cd799439011
  invalid: [unclosed array
`,
			fileName:    "invalid.yaml",
			expectError: true,
			errorMsg:    "failed to parse",
		},
		{
			name:        "empty file",
			fileContent: "",
			fileName:    "empty.yaml",
			expectError: true,
			errorMsg:    "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tempDir, tt.fileName)
			err := ioutil.WriteFile(filePath, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			// Test loading
			configs, err := loadConfigurations([]string{filePath}, &ApplyOptions{})

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, configs, 1)
				assert.NotNil(t, configs[0].Config)
			}
		})
	}
}

func TestValidateOutputFormats(t *testing.T) {
	validFormats := []string{"table", "json", "yaml", "summary", "detailed"}
	invalidFormats := []string{"xml", "csv", "html", "invalid"}

	for _, format := range validFormats {
		t.Run("valid_"+format, func(t *testing.T) {
			opts := &ApplyOptions{
				Files:        []string{"config.yaml"},
				OutputFormat: format,
			}
			err := validateApplyOptions(opts)
			// This might fail for other reasons, but not due to output format
			if err != nil {
				assert.NotContains(t, err.Error(), "invalid output format")
			}
		})
	}

	for _, format := range invalidFormats {
		t.Run("invalid_"+format, func(t *testing.T) {
			opts := &ApplyOptions{
				Files:        []string{"config.yaml"},
				OutputFormat: format,
			}
			err := validateApplyOptions(opts)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid output format")
		})
	}
}

func TestValidateDryRunModes(t *testing.T) {
	validModes := []string{"quick", "thorough", "detailed"}
	invalidModes := []string{"fast", "slow", "invalid"}

	for _, mode := range validModes {
		t.Run("valid_"+mode, func(t *testing.T) {
			opts := &ApplyOptions{
				Files:      []string{"config.yaml"},
				DryRunMode: mode,
			}
			err := validateApplyOptions(opts)
			// This might fail for other reasons, but not due to dry run mode
			if err != nil {
				assert.NotContains(t, err.Error(), "invalid dry run mode")
			}
		})
	}

	for _, mode := range invalidModes {
		t.Run("invalid_"+mode, func(t *testing.T) {
			opts := &ApplyOptions{
				Files:        []string{"config.yaml"},
				DryRunMode:   mode,
				OutputFormat: "table", // Set valid output format to test dry run mode validation
				DryRun:       true,    // Need to enable dry run to trigger dry run mode validation
			}
			err := validateApplyOptions(opts)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid dry-run mode")
		})
	}
}

func TestFilePatternEdgeCases(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "apply-edge-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		setupFunc   func() []string
		expectError bool
		errorMsg    string
	}{
		{
			name: "stdin and file together",
			setupFunc: func() []string {
				configFile := filepath.Join(tempDir, "config.yaml")
				_ = ioutil.WriteFile(configFile, []byte("test"), 0644)
				return []string{"-", configFile}
			},
			expectError: false,
		},
		{
			name: "directory instead of file",
			setupFunc: func() []string {
				return []string{tempDir}
			},
			expectError: true,
			errorMsg:    "no valid configuration files found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := tt.setupFunc()
			result, err := expandFilePatterns(patterns)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// Test helper function
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()

	tempFile, err := ioutil.TempFile("", "config-*.yaml")
	require.NoError(t, err)

	_, err = tempFile.WriteString(content)
	require.NoError(t, err)

	err = tempFile.Close()
	require.NoError(t, err)

	return tempFile.Name()
}

// Benchmark test for file expansion performance
func BenchmarkExpandFilePatternsAdvanced(b *testing.B) {
	tempDir, err := ioutil.TempDir("", "apply-bench")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	// Create many test files
	for i := 0; i < 100; i++ {
		fileName := filepath.Join(tempDir, fmt.Sprintf("config%d.yaml", i))
		err := ioutil.WriteFile(fileName, []byte("test content"), 0644)
		require.NoError(b, err)
	}

	pattern := filepath.Join(tempDir, "*.yaml")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := expandFilePatterns([]string{pattern})
		if err != nil {
			b.Fatalf("expandFilePatterns failed: %v", err)
		}
	}
}
