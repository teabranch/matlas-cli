package infra

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teabranch/matlas-cli/internal/apply"
)

func TestNewInfraCmd(t *testing.T) {
	cmd := NewInfraCmd()

	assert.Equal(t, "infra", cmd.Use)
	assert.Equal(t, "Manage infrastructure with declarative configuration", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)

	// Check that subcommands are added
	expectedSubcommands := []string{"validate", "plan", "diff", "show", "destroy"}
	for _, expectedCmd := range expectedSubcommands {
		subCmd, _, err := cmd.Find([]string{expectedCmd})
		assert.NoError(t, err)
		assert.Equal(t, expectedCmd, subCmd.Name())
	}
}

func TestApplyOptionsValidation(t *testing.T) {
	tests := []struct {
		name        string
		opts        *ApplyOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid options",
			opts: &ApplyOptions{
				Files:        []string{"config.yaml"},
				DryRun:       false,
				DryRunMode:   "quick",
				OutputFormat: "table",
				Timeout:      30 * time.Minute,
			},
			expectError: false,
		},
		{
			name: "no files specified",
			opts: &ApplyOptions{
				Files: []string{},
			},
			expectError: true,
			errorMsg:    "at least one configuration file must be specified",
		},
		{
			name: "invalid dry-run mode",
			opts: &ApplyOptions{
				Files:      []string{"config.yaml"},
				DryRun:     true,
				DryRunMode: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid dry-run mode",
		},
		{
			name: "invalid output format",
			opts: &ApplyOptions{
				Files:        []string{"config.yaml"},
				OutputFormat: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid output format",
		},
		{
			name: "negative timeout",
			opts: &ApplyOptions{
				Files:   []string{"config.yaml"},
				Timeout: -1 * time.Minute,
			},
			expectError: true,
			errorMsg:    "timeout must be positive",
		},
		{
			name: "watch mode with negative interval",
			opts: &ApplyOptions{
				Files:         []string{"config.yaml"},
				Watch:         true,
				WatchInterval: -1 * time.Minute,
			},
			expectError: true,
			errorMsg:    "watch interval must be positive",
		},
		{
			name: "watch mode with dry-run",
			opts: &ApplyOptions{
				Files:  []string{"config.yaml"},
				Watch:  true,
				DryRun: true,
			},
			expectError: true,
			errorMsg:    "watch mode cannot be used with dry-run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateApplyOptions(tt.opts)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExpandFilePatterns(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"config1.yaml", "config2.yml", "invalid.txt"}
	for _, file := range testFiles {
		filePath := tmpDir + "/" + file
		err := os.WriteFile(filePath, []byte("test: content"), 0644)
		require.NoError(t, err)
	}

	tests := []struct {
		name        string
		patterns    []string
		expected    int
		expectError bool
		errorMsg    string
	}{
		{
			name:     "stdin pattern",
			patterns: []string{"-"},
			expected: 1,
		},
		{
			name:     "single file",
			patterns: []string{tmpDir + "/config1.yaml"},
			expected: 1,
		},
		{
			name:     "glob pattern",
			patterns: []string{tmpDir + "/*.yaml"},
			expected: 1,
		},
		{
			name:     "multiple patterns",
			patterns: []string{tmpDir + "/*.yaml", tmpDir + "/*.yml"},
			expected: 2,
		},
		{
			name:        "multiple stdin patterns",
			patterns:    []string{"-", "-"},
			expectError: true,
			errorMsg:    "stdin (-) can only be specified once",
		},
		{
			name:        "non-existent file",
			patterns:    []string{tmpDir + "/nonexistent.yaml"},
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name:        "no matches for glob",
			patterns:    []string{tmpDir + "/*.json"},
			expectError: true,
			errorMsg:    "no files found matching pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := expandFilePatterns(tt.patterns)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, files, tt.expected)
			}
		})
	}
}

func TestHasGlobChars(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"simple.txt", false},
		{"*.yaml", true},
		{"file?.txt", true},
		{"file[0-9].txt", true},
		{"path/to/file.yaml", false},
		{"path/*/file.yaml", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := hasGlobChars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	assert.True(t, contains(slice, "banana"))
	assert.False(t, contains(slice, "orange"))
	assert.True(t, contains(slice, "apple"))
	assert.False(t, contains(slice, ""))
}

// Integration test for apply command flags
func TestApplyCmdFlags(t *testing.T) {
	cmd := NewInfraCmd()

	// Test setting flags
	args := []string{
		"--file", "config.yaml",
		"--dry-run",
		"--dry-run-mode", "thorough",
		"--output", "json",
		"--auto-approve",
		"--timeout", "45m",
		"--verbose",
		"--no-color",
		"--project-id", "507f1f77bcf86cd799439011",
		"--strict-env",
		"--watch",
		"--watch-interval", "10m",
	}

	cmd.SetArgs(args)
	err := cmd.ParseFlags(args)
	require.NoError(t, err)

	// Verify flags were set correctly
	files, _ := cmd.Flags().GetStringSlice("file")
	assert.Equal(t, []string{"config.yaml"}, files)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	assert.True(t, dryRun)

	dryRunMode, _ := cmd.Flags().GetString("dry-run-mode")
	assert.Equal(t, "thorough", dryRunMode)

	output, _ := cmd.Flags().GetString("output")
	assert.Equal(t, "json", output)

	autoApprove, _ := cmd.Flags().GetBool("auto-approve")
	assert.True(t, autoApprove)

	timeout, _ := cmd.Flags().GetDuration("timeout")
	assert.Equal(t, 45*time.Minute, timeout)

	verbose, _ := cmd.Flags().GetBool("verbose")
	assert.True(t, verbose)

	noColor, _ := cmd.Flags().GetBool("no-color")
	assert.True(t, noColor)

	projectID, _ := cmd.Flags().GetString("project-id")
	assert.Equal(t, "507f1f77bcf86cd799439011", projectID)

	strictEnv, _ := cmd.Flags().GetBool("strict-env")
	assert.True(t, strictEnv)

	watch, _ := cmd.Flags().GetBool("watch")
	assert.True(t, watch)

	watchInterval, _ := cmd.Flags().GetDuration("watch-interval")
	assert.Equal(t, 10*time.Minute, watchInterval)
}

func TestSubcommandCreation(t *testing.T) {
	tests := []struct {
		name        string
		createCmd   func() *cobra.Command
		expectedUse string
	}{
		{
			name:        "validate command",
			createCmd:   NewValidateCmd,
			expectedUse: "validate",
		},
		{
			name:        "plan command",
			createCmd:   NewPlanCmd,
			expectedUse: "plan",
		},
		{
			name:        "diff command",
			createCmd:   NewDiffCmd,
			expectedUse: "diff",
		},
		{
			name:        "show command",
			createCmd:   NewShowCmd,
			expectedUse: "show",
		},
		{
			name:        "destroy command",
			createCmd:   NewDestroyCmd,
			expectedUse: "destroy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.createCmd()
			assert.Equal(t, tt.expectedUse, cmd.Use)
			assert.NotEmpty(t, cmd.Short)
			assert.NotEmpty(t, cmd.Long)
			assert.NotEmpty(t, cmd.Example)
			assert.NotNil(t, cmd.RunE)
		})
	}
}

// Test command help output
func TestCommandHelp(t *testing.T) {
	cmd := NewInfraCmd()

	// Capture help output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})

	_ = cmd.Execute()
	// Help command should "fail" with exit code but that's expected

	helpOutput := buf.String()
	assert.Contains(t, helpOutput, "Apply declarative configuration")
	assert.Contains(t, helpOutput, "Available Commands:")
	assert.Contains(t, helpOutput, "validate")
	assert.Contains(t, helpOutput, "plan")
	assert.Contains(t, helpOutput, "diff")
	assert.Contains(t, helpOutput, "show")
	assert.Contains(t, helpOutput, "destroy")
}

// Benchmark tests
func BenchmarkExpandFilePatterns(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test files
	for i := 0; i < 100; i++ {
		filePath := tmpDir + "/" + string(rune('a'+i%26)) + ".yaml"
		err := os.WriteFile(filePath, []byte("test: content"), 0644)
		require.NoError(b, err)
	}

	patterns := []string{tmpDir + "/*.yaml"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := expandFilePatterns(patterns)
		require.NoError(b, err)
	}
}

func BenchmarkHasGlobChars(b *testing.B) {
	testStrings := []string{
		"simple.txt",
		"*.yaml",
		"file?.txt",
		"file[0-9].txt",
		"path/to/file.yaml",
		"path/*/file.yaml",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range testStrings {
			hasGlobChars(s)
		}
	}
}

// Error handling tests
func TestApplyCommandErrorHandling(t *testing.T) {
	cmd := NewInfraCmd()

	// Test with invalid flags
	cmd.SetArgs([]string{"--invalid-flag"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestGetProjectID(t *testing.T) {
	// Test with explicit project ID
	configs := []*apply.LoadResult{}
	opts := &ApplyOptions{ProjectID: "explicit-project-id"}

	projectID := getProjectID(configs, opts)
	assert.Equal(t, "explicit-project-id", projectID)

	// Test with empty options and configs
	opts = &ApplyOptions{}
	projectID = getProjectID(configs, opts)
	assert.Equal(t, "", projectID)
}
