package logging

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelCritical, "CRITICAL"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestLogLevel_ToSlogLevel(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected slog.Level
	}{
		{LevelDebug, slog.LevelDebug},
		{LevelInfo, slog.LevelInfo},
		{LevelWarn, slog.LevelWarn},
		{LevelError, slog.LevelError},
		{LevelCritical, slog.LevelError + 4}, // Critical maps to Error + 4
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.ToSlogLevel())
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	require.NotNil(t, config)
	assert.Equal(t, LevelInfo, config.Level)
	assert.Equal(t, "text", config.Format)
	assert.Equal(t, os.Stderr, config.Output)
	assert.False(t, config.IncludeSource)
	assert.False(t, config.Quiet)
	assert.False(t, config.Verbose)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{"default config", nil},
		{"verbose config", &Config{Verbose: true, Output: os.Stderr}},
		{"quiet config", &Config{Quiet: true, Output: os.Stderr}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			require.NotNil(t, logger)
			assert.NotNil(t, logger.slog)
			assert.NotNil(t, logger.config)
		})
	}
}

func TestNewWithWriter(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Verbose: true,
		Output:  &buf,
	}
	logger := New(config)

	require.NotNil(t, logger)
	assert.True(t, logger.config.Verbose)
	assert.NotNil(t, logger.slog)

	// Test that it actually writes to the buffer
	logger.Info("test message")
	assert.Contains(t, buf.String(), "test message")
}

func TestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer

	t.Run("verbose mode", func(t *testing.T) {
		buf.Reset()
		logger := New(&Config{Verbose: true, Output: &buf})
		logger.Debug("debug message")

		output := buf.String()
		assert.Contains(t, output, "debug message")
		assert.Contains(t, strings.ToUpper(output), "DEBUG")
	})

	t.Run("non-verbose mode", func(t *testing.T) {
		buf.Reset()
		logger := New(&Config{Verbose: false, Level: LevelInfo, Output: &buf})
		logger.Debug("debug message")

		// Debug messages should not appear in non-verbose mode when level is INFO
		assert.Empty(t, buf.String())
	})

	t.Run("quiet mode", func(t *testing.T) {
		buf.Reset()
		logger := New(&Config{Quiet: true, Output: &buf})
		logger.Debug("debug message")

		// Debug messages should not appear in quiet mode
		assert.Empty(t, buf.String())
	})
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{Output: &buf})

	logger.Info("info message")

	output := buf.String()
	assert.Contains(t, output, "info message")
	assert.Contains(t, strings.ToUpper(output), "INFO")
}

func TestLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{Output: &buf})

	logger.Warn("warning message")

	output := buf.String()
	assert.Contains(t, output, "warning message")
	assert.Contains(t, strings.ToUpper(output), "WARN")
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{Output: &buf})

	logger.Error("error message")

	output := buf.String()
	assert.Contains(t, output, "error message")
	assert.Contains(t, strings.ToUpper(output), "ERROR")
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{Output: &buf})

	ctx := context.WithValue(context.Background(), "key", "value")
	ctxLogger := logger.WithContext(ctx)

	assert.NotNil(t, ctxLogger)
	assert.Equal(t, ctx, ctxLogger.ctx)
	assert.Equal(t, logger.slog, ctxLogger.slog)
	assert.Equal(t, logger.config, ctxLogger.config)
}

func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{
		Format: "json",
		Output: &buf,
	})

	logger.Info("json test message")

	output := buf.String()
	assert.Contains(t, output, "json test message")
	// JSON format should contain quoted strings
	assert.Contains(t, output, `"`)
}

func TestLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{
		Format: "text",
		Output: &buf,
	})

	logger.Info("text test message")

	output := buf.String()
	assert.Contains(t, output, "text test message")
}

func TestLogger_WithSource(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{
		IncludeSource: true,
		Output:        &buf,
	})

	logger.Info("source test message")

	output := buf.String()
	assert.Contains(t, output, "source test message")
	// Should include source information (could be logger.go or logger_test.go)
	assert.Contains(t, output, "source=")
}

func TestLogger_QuietMode(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{
		Quiet:  true,
		Output: &buf,
	})

	// In quiet mode, only errors should appear
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.NotContains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLogger_VerboseMode(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{
		Verbose: true,
		Output:  &buf,
	})

	// In verbose mode, all messages should appear
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestConfig_LevelPriority(t *testing.T) {
	var buf bytes.Buffer

	// Test that quiet takes precedence over verbose
	logger := New(&Config{
		Quiet:   true,
		Verbose: true,
		Output:  &buf,
	})

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	// Should behave as quiet (only errors)
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.NotContains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLogger_WithContextExtended(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{Output: &buf})

	ctx := context.WithValue(context.Background(), "test", "value")
	contextLogger := logger.WithContext(ctx)

	assert.NotNil(t, contextLogger)
	assert.Equal(t, ctx, contextLogger.ctx)
	assert.Equal(t, logger.slog, contextLogger.slog)
	assert.Equal(t, logger.config, contextLogger.config)
}

func TestLogger_Critical(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{Output: &buf})

	logger.Critical("critical error occurred")

	output := buf.String()
	assert.Contains(t, output, "critical error occurred")
	assert.Contains(t, strings.ToUpper(output), "ERROR") // Critical maps to ERROR level
}

func TestLogger_ContextualLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&Config{
		Output:        &buf,
		IncludeSource: true,
	})

	logger.Info("contextual message", "key1", "value1", "key2", 42)

	output := buf.String()
	assert.Contains(t, output, "contextual message")
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "key2")
	assert.Contains(t, output, "42")
}

func TestDefaultConfig_AllFields(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, LevelInfo, config.Level)
	assert.Equal(t, "text", config.Format)
	assert.NotNil(t, config.Output) // Output can be stdout or stderr
	assert.False(t, config.IncludeSource)
	assert.False(t, config.Quiet)
	assert.False(t, config.Verbose)
	assert.False(t, config.EnableAPILogs)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.MaskSecrets)
	assert.Equal(t, 30*time.Second, config.RequestTimeout)
}

func TestLogLevel_Conversion(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected slog.Level
	}{
		{LevelDebug, slog.LevelDebug},
		{LevelInfo, slog.LevelInfo},
		{LevelWarn, slog.LevelWarn},
		{LevelError, slog.LevelError},
		{LevelCritical, slog.Level(12)}, // Critical maps to custom level 12
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			result := tt.level.ToSlogLevel()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogger_Formatting(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		expectJSONLike bool
	}{
		{
			name:           "JSON format",
			format:         "json",
			expectJSONLike: true,
		},
		{
			name:           "text format",
			format:         "text",
			expectJSONLike: false,
		},
		{
			name:           "default format",
			format:         "",
			expectJSONLike: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&Config{
				Format: tt.format,
				Output: &buf,
			})

			logger.Info("test message", "key", "value")

			output := buf.String()
			assert.Contains(t, output, "test message")

			if tt.expectJSONLike {
				// JSON format should contain quotes and braces
				assert.Contains(t, output, "\"")
			} else {
				// Text format should be more readable
				assert.Contains(t, output, "key=value")
			}
		})
	}
}
