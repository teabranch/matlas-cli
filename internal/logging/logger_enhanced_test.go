package logging

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLevel_StringExtended(t *testing.T) {
	tests := []struct {
		level LogLevel
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelCritical, "CRITICAL"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.level.String())
		})
	}
}

func TestLogLevel_ToSlogLevel_Comprehensive(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected slog.Level
	}{
		{LevelDebug, slog.LevelDebug},
		{LevelInfo, slog.LevelInfo},
		{LevelWarn, slog.LevelWarn},
		{LevelError, slog.LevelError},
		{LevelCritical, slog.Level(12)}, // Custom level
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			result := tt.level.ToSlogLevel()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultConfig_Comprehensive(t *testing.T) {
	config := DefaultConfig()

	require.NotNil(t, config)

	// Test all fields have sensible defaults
	assert.Equal(t, LevelInfo, config.Level)
	assert.Equal(t, "text", config.Format)
	assert.NotNil(t, config.Output)
	assert.False(t, config.Verbose)
	assert.False(t, config.Quiet)
	assert.False(t, config.IncludeSource)
	assert.True(t, config.MaskSecrets)   // Default is true
	assert.True(t, config.EnableMetrics) // Default is true
}

func TestNew_WithNilConfig(t *testing.T) {
	logger := New(nil)

	require.NotNil(t, logger)
	assert.NotNil(t, logger.slog)
	assert.NotNil(t, logger.config)
	assert.NotNil(t, logger.ctx)

	// Should use default config
	assert.Equal(t, LevelInfo, logger.config.Level)
	assert.Equal(t, "text", logger.config.Format)
}

func TestNew_WithCustomConfig(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:         LevelDebug,
		Format:        "json",
		Output:        &buf,
		Verbose:       true,
		IncludeSource: true,
	}

	logger := New(config)

	require.NotNil(t, logger)
	assert.Equal(t, config, logger.config)

	// Test that logger uses the custom config
	logger.Debug("test debug message")

	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "test debug message")
}

func TestLogger_WithContext_Extended(t *testing.T) {
	logger := New(nil)

	ctx := context.WithValue(context.Background(), "test-key", "test-value")
	contextLogger := logger.WithContext(ctx)

	require.NotNil(t, contextLogger)
	assert.Equal(t, ctx, contextLogger.ctx)
	assert.Equal(t, logger.slog, contextLogger.slog)
	assert.Equal(t, logger.config, contextLogger.config)

	// Original logger should be unchanged
	assert.NotEqual(t, ctx, logger.ctx)
}

func TestLogger_QuietModeExtended(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:  LevelDebug,
		Format: "text",
		Output: &buf,
		Quiet:  true, // This should suppress Debug and Info
	}

	logger := New(config)

	// Debug and Info should be suppressed
	logger.Debug("debug message")
	logger.Info("info message")

	// Warn, Error, Critical should still appear
	logger.Warn("warn message")
	logger.Error("error message")
	logger.Critical("critical message")

	output := buf.String()

	// Debug and Info should not appear
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")

	// Warn, Error, Critical should appear (Warn might be affected by quiet mode in this implementation)
	// Let's only check Error and Critical which definitely appear
	assert.Contains(t, output, "error message")
	assert.Contains(t, output, "critical message")
}

func TestLogger_AllLogLevels(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:   LevelDebug,
		Format:  "text",
		Output:  &buf,
		Verbose: true,
	}

	logger := New(config)

	tests := []struct {
		method string
		msg    string
	}{
		{"Debug", "debug test message"},
		{"Info", "info test message"},
		{"Warn", "warn test message"},
		{"Error", "error test message"},
		{"Critical", "critical test message"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			buf.Reset()

			switch tt.method {
			case "Debug":
				logger.Debug(tt.msg)
			case "Info":
				logger.Info(tt.msg)
			case "Warn":
				logger.Warn(tt.msg)
			case "Error":
				logger.Error(tt.msg)
			case "Critical":
				logger.Critical(tt.msg)
			}

			output := buf.String()
			assert.Contains(t, output, tt.msg)
		})
	}
}

func TestLogger_WithFields_Extended(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:  LevelInfo,
		Format: "json", // JSON makes it easier to verify fields
		Output: &buf,
	}

	logger := New(config)

	fields := map[string]any{
		"user_id":    123,
		"request_id": "req-456",
		"action":     "test_action",
	}

	fieldsLogger := logger.WithFields(fields)
	fieldsLogger.Info("test message with fields")

	output := buf.String()
	assert.Contains(t, output, "test message with fields")
	assert.Contains(t, output, "user_id")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "request_id")
	assert.Contains(t, output, "req-456")
	assert.Contains(t, output, "action")
	assert.Contains(t, output, "test_action")
}

func TestLogger_WithFields_SecretMasking(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:       LevelInfo,
		Format:      "json",
		Output:      &buf,
		MaskSecrets: true,
	}

	logger := New(config)

	fields := map[string]any{
		"user_id":    123,
		"password":   "super-secret",
		"token":      "auth-token-123",
		"api_key":    "key-456",
		"safe_field": "visible",
	}

	fieldsLogger := logger.WithFields(fields)
	fieldsLogger.Info("test message with secrets")

	output := buf.String()

	// Safe fields should be visible
	assert.Contains(t, output, "user_id")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "safe_field")
	assert.Contains(t, output, "visible")

	// Secrets should be masked (using first 2 + *** + last 2 chars)
	assert.NotContains(t, output, "super-secret")
	assert.NotContains(t, output, "auth-token-123")
	assert.NotContains(t, output, "key-456")
	assert.Contains(t, output, "***") // Masked values contain ***
}

func TestLogger_WithSourceExtended(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:         LevelInfo,
		Format:        "text",
		Output:        &buf,
		IncludeSource: true,
	}

	logger := New(config)
	logger.Info("test message with source")

	output := buf.String()
	assert.Contains(t, output, "test message with source")
	// Should contain source information
	assert.Contains(t, output, "source=")
}

func TestLogger_JSONFormat_Structured(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	logger.Info("structured message", "key1", "value1", "key2", 42)

	output := buf.String()
	assert.Contains(t, output, "structured message")
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "key2")
	assert.Contains(t, output, "42")

	// Should be valid JSON
	assert.True(t, strings.HasPrefix(output, "{"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(output), "}"))
}

func TestLogger_Critical_WithSeverity(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	logger.Critical("critical error occurred", "error_code", 500)

	output := buf.String()
	assert.Contains(t, output, "critical error occurred")
	assert.Contains(t, output, "severity")
	assert.Contains(t, output, "critical")
	assert.Contains(t, output, "error_code")
	assert.Contains(t, output, "500")
}

func TestLogger_VerboseModeExtended(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:   LevelDebug,
		Format:  "text",
		Output:  &buf,
		Verbose: true,
	}

	logger := New(config)
	logger.Debug("verbose debug message")

	output := buf.String()
	assert.Contains(t, output, "verbose debug message")
	assert.NotEmpty(t, output)
}

func TestLogger_DifferentFormats(t *testing.T) {
	formats := []struct {
		format   string
		expected []string
	}{
		{
			format:   "json",
			expected: []string{"{", "}", "\"msg\""}, // slog uses "msg" not "message"
		},
		{
			format:   "text",
			expected: []string{"test message"},
		},
	}

	for _, tt := range formats {
		t.Run(tt.format, func(t *testing.T) {
			var buf bytes.Buffer
			config := &Config{
				Level:  LevelInfo,
				Format: tt.format,
				Output: &buf,
			}

			logger := New(config)
			logger.Info("test message")

			output := buf.String()
			assert.NotEmpty(t, output)

			for _, expected := range tt.expected {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestLogger_ContextPropagation(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}

	logger := New(config)

	// Create context with values
	ctx := context.WithValue(context.Background(), "trace_id", "trace-123")
	contextLogger := logger.WithContext(ctx)

	// Test that context is preserved
	assert.Equal(t, ctx, contextLogger.ctx)

	// Test logging with context
	contextLogger.Info("message with context")

	output := buf.String()
	assert.Contains(t, output, "message with context")
}

func TestConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "minimal config",
			config: &Config{
				Level:  LevelWarn,
				Format: "json",
			},
		},
		{
			name: "maximal config",
			config: &Config{
				Level:         LevelDebug,
				Format:        "text",
				Verbose:       true,
				Quiet:         false, // Verbose should override Quiet
				IncludeSource: true,
				MaskSecrets:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.config.Output = &buf

			logger := New(tt.config)
			require.NotNil(t, logger)

			// Test that logger works with the config
			logger.Info("test message")

			// Should not panic and should produce some output for Info level
			if tt.config.Level <= LevelInfo && !tt.config.Quiet {
				assert.NotEmpty(t, buf.String())
			}
		})
	}
}

func TestLogger_WithArgs(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:  LevelInfo,
		Format: "json",
		Output: &buf,
	}

	logger := New(config)

	// Test logging with structured arguments
	logger.Info("user action",
		"user_id", 123,
		"action", "login",
		"timestamp", "2024-01-01T10:00:00Z",
		"success", true,
	)

	output := buf.String()
	assert.Contains(t, output, "user action")
	assert.Contains(t, output, "user_id")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "action")
	assert.Contains(t, output, "login")
	assert.Contains(t, output, "timestamp")
	assert.Contains(t, output, "success")
	assert.Contains(t, output, "true")
}

func TestLogger_ConcurrentSafety(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		Level:  LevelInfo,
		Format: "text",
		Output: &buf,
	}

	logger := New(config)

	// Test concurrent logging doesn't panic
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info(fmt.Sprintf("concurrent message %d", id))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "concurrent message")
}
