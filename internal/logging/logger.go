// Package logging provides structured logging utilities used across the CLI and services.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"
)

// LogLevel represents different log levels.
type LogLevel int

const (
	// LevelDebug is verbose diagnostic logging.
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelCritical
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ToSlogLevel converts our LogLevel to slog.Level
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	case LevelCritical:
		return slog.LevelError + 4 // Higher than error
	default:
		return slog.LevelInfo
	}
}

// Config holds logging configuration
type Config struct {
	Level          LogLevel
	Format         string // "text" or "json"
	Output         io.Writer
	IncludeSource  bool
	Quiet          bool
	Verbose        bool
	EnableAPILogs  bool
	EnableMetrics  bool
	MaskSecrets    bool
	RequestTimeout time.Duration
}

// DefaultConfig returns the default logging configuration
func DefaultConfig() *Config {
	return &Config{
		Level:          LevelInfo,
		Format:         "text",
		Output:         os.Stderr,
		IncludeSource:  false,
		Quiet:          false,
		Verbose:        false,
		EnableAPILogs:  false,
		EnableMetrics:  true,
		MaskSecrets:    true,
		RequestTimeout: 30 * time.Second,
	}
}

// Logger provides structured logging with advanced features
type Logger struct {
	slog   *slog.Logger
	config *Config
	ctx    context.Context
}

// New creates a new logger with the given configuration
func New(config *Config) *Logger {
	if config == nil {
		config = DefaultConfig()
	}

	// Adjust level based on verbose/quiet flags
	level := config.Level
	if config.Quiet {
		level = LevelError
	} else if config.Verbose {
		level = LevelDebug
	}

	// Create slog handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level.ToSlogLevel(),
		AddSource: config.IncludeSource,
	}

	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(config.Output, opts)
	default:
		handler = slog.NewTextHandler(config.Output, opts)
	}

	slogLogger := slog.New(handler)

	return &Logger{
		slog:   slogLogger,
		config: config,
		ctx:    context.Background(),
	}
}

// WithContext returns a logger with the given context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		slog:   l.slog,
		config: l.config,
		ctx:    ctx,
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	if l.config.Quiet {
		return
	}
	l.slog.DebugContext(l.ctx, msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	if l.config.Quiet {
		return
	}
	l.slog.InfoContext(l.ctx, msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.WarnContext(l.ctx, msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.slog.ErrorContext(l.ctx, msg, args...)
}

// Critical logs a critical error message
func (l *Logger) Critical(msg string, args ...any) {
	// Use Error level with additional severity indicator
	allArgs := append([]any{"severity", "critical"}, args...)
	l.slog.ErrorContext(l.ctx, msg, allArgs...)
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]any) *Logger {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		// SECURITY: Mask if key suggests secret OR value looks like secret
		if l.config.MaskSecrets && (l.isSecret(k) || l.containsSecretValue(v)) {
			v = l.maskValue(v)
		}
		args = append(args, k, v)
	}

	return &Logger{
		slog:   l.slog.With(args...),
		config: l.config,
		ctx:    l.ctx,
	}
}

// Operation represents a long-running operation for logging
type Operation struct {
	ID        string
	Type      string
	StartTime time.Time
	logger    *Logger
}

// StartOperation begins tracking a long-running operation
func (l *Logger) StartOperation(id, opType string) *Operation {
	op := &Operation{
		ID:        id,
		Type:      opType,
		StartTime: time.Now(),
		logger:    l.WithFields(map[string]any{"operation_id": id, "operation_type": opType}),
	}

	op.logger.Info("Operation started")
	return op
}

// Progress logs operation progress
func (op *Operation) Progress(message string, percent float64) {
	op.logger.Info("Operation progress",
		"message", message,
		"percent", fmt.Sprintf("%.1f%%", percent),
		"elapsed", time.Since(op.StartTime).String())
}

// Complete marks the operation as completed
func (op *Operation) Complete(message string) {
	duration := time.Since(op.StartTime)
	op.logger.Info("Operation completed",
		"message", message,
		"duration", duration.String())
}

// Fail marks the operation as failed
func (op *Operation) Fail(err error, message string) {
	duration := time.Since(op.StartTime)
	op.logger.Error("Operation failed",
		"message", message,
		"error", err.Error(),
		"duration", duration.String())
}

// APIRequest represents an API request for logging
type APIRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
	Started time.Time
}

// APIResponse represents an API response for logging
type APIResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       string
	Duration   time.Duration
}

// LogAPIRequest logs an Atlas API request (with secret masking)
func (l *Logger) LogAPIRequest(req *APIRequest) {
	if !l.config.EnableAPILogs {
		return
	}

	fields := map[string]any{
		"api_method": req.Method,
		"api_url":    l.maskURL(req.URL),
		"timestamp":  req.Started.Format(time.RFC3339),
	}

	// Add headers (mask sensitive ones)
	if len(req.Headers) > 0 {
		maskedHeaders := make(map[string]string)
		for k, v := range req.Headers {
			if l.isSecret(k) {
				maskedHeaders[k] = l.maskValue(v).(string)
			} else {
				maskedHeaders[k] = v
			}
		}
		fields["api_headers"] = maskedHeaders
	}

	// Add body if not too large and not containing secrets
	if req.Body != "" && len(req.Body) < 1024 && !l.containsSecrets(req.Body) {
		fields["api_body"] = req.Body
	}

	l.WithFields(fields).Debug("API request sent")
}

// LogAPIResponse logs an Atlas API response
func (l *Logger) LogAPIResponse(req *APIRequest, resp *APIResponse) {
	if !l.config.EnableAPILogs {
		return
	}

	fields := map[string]any{
		"api_method":      req.Method,
		"api_url":         l.maskURL(req.URL),
		"api_status_code": resp.StatusCode,
		"api_duration":    resp.Duration.String(),
		"api_latency_ms":  resp.Duration.Milliseconds(),
	}

	// Add response body if not too large
	if resp.Body != "" && len(resp.Body) < 2048 {
		fields["api_response_body"] = resp.Body
	}

	logLevel := LevelDebug
	if resp.StatusCode >= 400 {
		logLevel = LevelWarn
	}
	if resp.StatusCode >= 500 {
		logLevel = LevelError
	}

	switch logLevel {
	case LevelDebug:
		l.WithFields(fields).Debug("API response received")
	case LevelWarn:
		l.WithFields(fields).Warn("API response with client error")
	case LevelError:
		l.WithFields(fields).Error("API response with server error")
	}
}

// LogMetric logs a performance metric
func (l *Logger) LogMetric(name string, value float64, unit string, tags map[string]string) {
	if !l.config.EnableMetrics {
		return
	}

	fields := map[string]any{
		"metric_name":  name,
		"metric_value": value,
		"metric_unit":  unit,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	for k, v := range tags {
		fields[fmt.Sprintf("tag_%s", k)] = v
	}

	l.WithFields(fields).Info("Performance metric")
}

// Helper methods for secret masking

func (l *Logger) isSecret(key string) bool {
	// Expanded keyword list for comprehensive secret detection
	secretKeywords := []string{
		"api_key", "apikey", "api-key",
		"password", "passwd", "pwd", "pass",
		"token", "auth", "authorization", "bearer",
		"secret", "private_key", "private-key", "privatekey",
		"connection_string", "connection-string", "connectionstring",
		"mongodb_uri", "mongo_uri", "uri",
		"credential", "creds",
		"access_key", "accesskey", "access-key",
		"session", "cookie",
		"certificate", "cert", "key",
		"public_key", "publickey", "public-key",
	}

	lowerKey := strings.ToLower(key)
	for _, keyword := range secretKeywords {
		if strings.Contains(lowerKey, keyword) {
			return true
		}
	}

	return false
}

// containsSecretValue performs pattern-based secret detection on values
func (l *Logger) containsSecretValue(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		return false
	}

	// Minimum length check - very short strings are unlikely to be secrets
	if len(str) < 8 {
		return false
	}

	// Check for common secret patterns
	patterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"jwt", regexp.MustCompile(`^[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.[A-Za-z0-9-_.+/=]+$`)},
		{"base64_key", regexp.MustCompile(`^[A-Za-z0-9+/]{32,}={0,2}$`)},
		{"mongodb_uri", regexp.MustCompile(`^mongodb(\+srv)?://.*@`)},
		{"aws_key", regexp.MustCompile(`^AKIA[0-9A-Z]{16}$`)},
		{"hex_key", regexp.MustCompile(`^[a-fA-F0-9]{32,}$`)},
	}

	for _, p := range patterns {
		if p.pattern.MatchString(str) {
			return true
		}
	}

	return false
}

func (l *Logger) maskValue(value any) any {
	str, ok := value.(string)
	if !ok {
		return "***"
	}

	if len(str) <= 4 {
		return "***"
	}

	// Show first 2 and last 2 characters for API keys, etc.
	return str[:2] + "***" + str[len(str)-2:]
}

func (l *Logger) maskURL(url string) string {
	if !l.config.MaskSecrets {
		return url
	}

	// Mask query parameters that might contain secrets
	if strings.Contains(url, "?") {
		parts := strings.Split(url, "?")
		return parts[0] + "?<masked>"
	}
	return url
}

func (l *Logger) containsSecrets(text string) bool {
	lowerText := strings.ToLower(text)
	secretIndicators := []string{
		"password", "token", "key", "secret", "auth",
	}

	for _, indicator := range secretIndicators {
		if strings.Contains(lowerText, indicator) {
			return true
		}
	}
	return false
}

// Global logger instance
var defaultLogger *Logger

// SetDefault sets the default global logger
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// Default returns the default global logger.
func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger = New(DefaultConfig())
	}
	return defaultLogger
}

// Global convenience functions.
func Debug(msg string, args ...any)    { Default().Debug(msg, args...) }
func Info(msg string, args ...any)     { Default().Info(msg, args...) }
func Warn(msg string, args ...any)     { Default().Warn(msg, args...) }
func Error(msg string, args ...any)    { Default().Error(msg, args...) }
func Critical(msg string, args ...any) { Default().Critical(msg, args...) }
