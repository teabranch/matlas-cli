package cli

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/teabranch/matlas-cli/internal/logging"
)

// EnhancedErrorFormatter builds on the existing ErrorFormatter with additional features
type EnhancedErrorFormatter struct {
	*ErrorFormatter
	analyzer *ErrorAnalyzer
	verbose  bool
}

// NewEnhancedErrorFormatter creates an enhanced error formatter
func NewEnhancedErrorFormatter(verbose bool, logger *logging.Logger) *EnhancedErrorFormatter {
	return &EnhancedErrorFormatter{
		ErrorFormatter: NewErrorFormatter(verbose),
		analyzer:       NewErrorAnalyzer(logger),
		verbose:        verbose,
	}
}

// FormatWithAnalysis formats an error with analysis and suggestions
func (eef *EnhancedErrorFormatter) FormatWithAnalysis(err error) string {
	if err == nil {
		return ""
	}

	// Start with basic formatting
	baseMessage := eef.ErrorFormatter.Format(err)

	// If this is already a clean error message, don't add extra formatting
	if !strings.Contains(baseMessage, "Error:") && !strings.Contains(err.Error(), "usage") {
		baseMessage = fmt.Sprintf("Error: %s", baseMessage)
	}

	// Add analysis if verbose
	if eef.verbose {
		analysis := eef.analyzer.Analyze(err)
		if len(analysis.Suggestions) > 0 {
			baseMessage += "\n\nSuggestions:"
			for i, suggestion := range analysis.Suggestions {
				baseMessage += fmt.Sprintf("\n  %d. %s", i+1, suggestion)
			}
		}

		if analysis.RootCause != "" {
			baseMessage += fmt.Sprintf("\n\nRoot Cause: %s", analysis.RootCause)
		}
	}

	return baseMessage
}

// ErrorCategory represents different types of errors
type ErrorCategory string

const (
	CategoryAuthentication ErrorCategory = "authentication"
	CategoryAuthorization  ErrorCategory = "authorization"
	CategoryNetwork        ErrorCategory = "network"
	CategoryValidation     ErrorCategory = "validation"
	CategoryConfiguration  ErrorCategory = "configuration"
	CategoryResource       ErrorCategory = "resource"
	CategoryInternal       ErrorCategory = "internal"
	CategoryRetryable      ErrorCategory = "retryable"
	CategoryFatal          ErrorCategory = "fatal"
)

// ErrorAnalysis contains the results of error analysis
type ErrorAnalysis struct {
	Category       ErrorCategory  `json:"category"`
	Severity       string         `json:"severity"`
	Retryable      bool           `json:"retryable"`
	UserActionable bool           `json:"user_actionable"`
	Suggestions    []string       `json:"suggestions"`
	RootCause      string         `json:"root_cause"`
	Metadata       map[string]any `json:"metadata"`
}

// ErrorAnalyzer provides error analysis and categorization
type ErrorAnalyzer struct {
	logger *logging.Logger
}

// NewErrorAnalyzer creates a new error analyzer
func NewErrorAnalyzer(logger *logging.Logger) *ErrorAnalyzer {
	return &ErrorAnalyzer{logger: logger}
}

// Analyze analyzes an error and provides insights
func (ea *ErrorAnalyzer) Analyze(err error) ErrorAnalysis {
	if err == nil {
		return ErrorAnalysis{}
	}

	errStr := strings.ToLower(err.Error())

	analysis := ErrorAnalysis{
		Metadata: make(map[string]any),
	}

	// Categorize error
	analysis.Category = ea.categorizeError(errStr)

	// Determine severity
	analysis.Severity = ea.determineSeverity(errStr, analysis.Category)

	// Check if retryable
	analysis.Retryable = ea.isRetryable(errStr, analysis.Category)

	// Check if user actionable
	analysis.UserActionable = ea.isUserActionable(analysis.Category)

	// Generate suggestions
	analysis.Suggestions = ea.generateSuggestions(errStr, analysis.Category)

	// Determine root cause
	analysis.RootCause = ea.determineRootCause(errStr, analysis.Category)

	return analysis
}

func (ea *ErrorAnalyzer) categorizeError(errStr string) ErrorCategory {
	// Authentication errors
	if strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "authentication") ||
		strings.Contains(errStr, "api key") {
		return CategoryAuthentication
	}

	// Authorization errors
	if strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "permission") ||
		strings.Contains(errStr, "access denied") {
		return CategoryAuthorization
	}

	// Network errors
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "dns") {
		return CategoryNetwork
	}

	// Validation errors
	if strings.Contains(errStr, "invalid") ||
		strings.Contains(errStr, "validation") ||
		strings.Contains(errStr, "required") ||
		strings.Contains(errStr, "format") {
		return CategoryValidation
	}

	// Configuration errors
	if strings.Contains(errStr, "config") ||
		strings.Contains(errStr, "yaml") ||
		strings.Contains(errStr, "json") {
		return CategoryConfiguration
	}

	// Resource errors
	if strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "conflict") {
		return CategoryResource
	}

	return CategoryInternal
}

func (ea *ErrorAnalyzer) determineSeverity(errStr string, category ErrorCategory) string {
	switch category {
	case CategoryAuthentication, CategoryAuthorization:
		return "critical"
	case CategoryFatal:
		return "critical"
	case CategoryNetwork:
		if strings.Contains(errStr, "timeout") {
			return "warning"
		}
		return "error"
	case CategoryValidation, CategoryConfiguration:
		return "error"
	case CategoryResource:
		return "warning"
	default:
		return "error"
	}
}

func (ea *ErrorAnalyzer) isRetryable(errStr string, category ErrorCategory) bool {
	switch category {
	case CategoryNetwork:
		return true
	case CategoryRetryable:
		return true
	case CategoryAuthentication, CategoryAuthorization:
		return false
	case CategoryValidation, CategoryConfiguration:
		return false
	case CategoryResource:
		return strings.Contains(errStr, "not found") || strings.Contains(errStr, "conflict")
	default:
		return false
	}
}

func (ea *ErrorAnalyzer) isUserActionable(category ErrorCategory) bool {
	switch category {
	case CategoryAuthentication, CategoryAuthorization, CategoryValidation, CategoryConfiguration:
		return true
	case CategoryResource:
		return true
	case CategoryNetwork:
		return false // Usually infrastructure-related
	default:
		return false
	}
}

func (ea *ErrorAnalyzer) generateSuggestions(errStr string, category ErrorCategory) []string {
	switch category {
	case CategoryAuthentication:
		return []string{
			"Check your API key and public key",
			"Verify credentials are set in environment variables",
			"Ensure you have access to the MongoDB Atlas organization",
		}
	case CategoryAuthorization:
		return []string{
			"Verify you have the necessary permissions",
			"Check your role assignments in MongoDB Atlas",
			"Contact your administrator for access",
		}
	case CategoryNetwork:
		return []string{
			"Check your internet connection",
			"Verify MongoDB Atlas is accessible",
			"Try again later if this is a temporary issue",
		}
	case CategoryValidation:
		return []string{
			"Check the format of your input",
			"Verify all required fields are provided",
			"Review the configuration schema",
		}
	case CategoryConfiguration:
		return []string{
			"Validate your YAML/JSON syntax",
			"Check the configuration file path",
			"Review the configuration documentation",
		}
	case CategoryResource:
		if strings.Contains(errStr, "not found") {
			return []string{
				"Verify the resource ID exists",
				"Check you're using the correct project",
				"Ensure the resource hasn't been deleted",
			}
		}
		if strings.Contains(errStr, "already exists") {
			return []string{
				"Use a different name for the resource",
				"Check if the resource already exists",
				"Consider updating instead of creating",
			}
		}
	}

	return []string{"Review the error message for specific guidance"}
}

func (ea *ErrorAnalyzer) determineRootCause(errStr string, category ErrorCategory) string {
	switch category {
	case CategoryAuthentication:
		return "Invalid or missing authentication credentials"
	case CategoryAuthorization:
		return "Insufficient permissions for the requested operation"
	case CategoryNetwork:
		return "Network connectivity or service availability issue"
	case CategoryValidation:
		return "Input data does not meet validation requirements"
	case CategoryConfiguration:
		return "Configuration file format or content issue"
	case CategoryResource:
		if strings.Contains(errStr, "not found") {
			return "Requested resource does not exist"
		}
		if strings.Contains(errStr, "already exists") {
			return "Resource name conflict"
		}
		return "Resource state or availability issue"
	default:
		return "Internal application or service error"
	}
}

// ErrorCollector collects multiple errors for batch processing
type ErrorCollector struct {
	errors   []error
	contexts []ErrorContext
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors:   make([]error, 0),
		contexts: make([]ErrorContext, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error, ctx ...ErrorContext) {
	if err == nil {
		return
	}

	ec.errors = append(ec.errors, err)

	if len(ctx) > 0 {
		ec.contexts = append(ec.contexts, ctx[0])
	} else {
		ec.contexts = append(ec.contexts, ErrorContext{Timestamp: time.Now()})
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Count returns the number of errors
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// FirstError returns the first error or nil
func (ec *ErrorCollector) FirstError() error {
	if len(ec.errors) == 0 {
		return nil
	}
	return ec.errors[0]
}

// Combine combines all errors into a single error
func (ec *ErrorCollector) Combine() error {
	if len(ec.errors) == 0 {
		return nil
	}

	if len(ec.errors) == 1 {
		return ec.errors[0]
	}

	var messages []string
	for _, err := range ec.errors {
		messages = append(messages, err.Error())
	}

	return fmt.Errorf("multiple errors occurred: %s", strings.Join(messages, "; "))
}

// ContextualError wraps an error with enhanced context
type ContextualError struct {
	Err       error        `json:"error"`
	Context   ErrorContext `json:"context"`
	CallStack []string     `json:"call_stack,omitempty"`
}

// Error implements the error interface
func (ce *ContextualError) Error() string {
	if ce.Context.Operation != "" {
		return fmt.Sprintf("[%s] %s", ce.Context.Operation, ce.Err.Error())
	}
	return ce.Err.Error()
}

// Unwrap implements the unwrapper interface for errors.Is/As
func (ce *ContextualError) Unwrap() error {
	return ce.Err
}

// Is implements error identity checking
func (ce *ContextualError) Is(target error) bool {
	return errors.Is(ce.Err, target)
}

// As implements error type assertion
func (ce *ContextualError) As(target interface{}) bool {
	return errors.As(ce.Err, target)
}

// WrapWithContext wraps an error with context
func WrapWithContext(err error, ctx ErrorContext) error {
	if err == nil {
		return nil
	}

	// Add timestamp if not set
	if ctx.Timestamp.IsZero() {
		ctx.Timestamp = time.Now()
	}

	return &ContextualError{
		Err:     err,
		Context: ctx,
	}
}

// WrapWithOperation wraps an error with operation context
func WrapWithOperation(err error, operation, resource string) error {
	if err == nil {
		return nil
	}

	ctx := ErrorContext{
		Operation: operation,
		Resource:  resource,
		Timestamp: time.Now(),
	}

	return WrapWithContext(err, ctx)
}

// WrapWithStack wraps an error with call stack information
func WrapWithStack(err error, operation string) error {
	if err == nil {
		return nil
	}

	ctx := ErrorContext{
		Operation: operation,
		Timestamp: time.Now(),
	}

	return &ContextualError{
		Err:       fmt.Errorf("%s: %w", operation, err),
		Context:   ctx,
		CallStack: GetCallStack(2), // Skip this function and the caller
	}
}

// GetCallStack returns the current call stack
func GetCallStack(skip int) []string {
	var stack []string
	for i := skip; i < skip+10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		// Extract just the filename from the full path
		parts := strings.Split(file, "/")
		filename := parts[len(parts)-1]

		stack = append(stack, fmt.Sprintf("%s:%d %s", filename, line, fn.Name()))
	}
	return stack
}

// HandleWithRecovery handles panics and converts them to errors
func HandleWithRecovery(operation string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx := ErrorContext{
				Operation: "panic_recovery",
				Command:   operation,
				Timestamp: time.Now(),
			}

			var panicErr error
			switch v := r.(type) {
			case error:
				panicErr = v
			case string:
				panicErr = errors.New(v)
			default:
				panicErr = fmt.Errorf("panic: %v", v)
			}

			err = &ContextualError{
				Err:       panicErr,
				Context:   ctx,
				CallStack: GetCallStack(2),
			}
		}
	}()

	return fn()
}

// ChainErrors creates an error from multiple errors
func ChainErrors(primary error, secondary ...error) error {
	if primary == nil {
		return nil
	}

	if len(secondary) == 0 {
		return primary
	}

	var messages []string
	messages = append(messages, primary.Error())

	for _, err := range secondary {
		if err != nil {
			messages = append(messages, err.Error())
		}
	}

	return fmt.Errorf("%s (related errors: %s)",
		primary.Error(),
		strings.Join(messages[1:], "; "))
}

// ConfigureCommandErrorHandling sets up proper error handling for a command
// to prevent help text from being displayed when errors occur
func ConfigureCommandErrorHandling(cmd *cobra.Command) {
	// Silence usage text on errors
	cmd.SilenceUsage = true

	// Apply to all subcommands recursively
	for _, subCmd := range cmd.Commands() {
		ConfigureCommandErrorHandling(subCmd)
	}
}

// WrapCommandWithCleanErrors wraps a command's RunE function to provide clean error output
func WrapCommandWithCleanErrors(cmd *cobra.Command, runE func(cmd *cobra.Command, args []string) error) {
	if runE == nil {
		return
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := runE(cmd, args)
		if err != nil {
			// Ensure usage is silenced
			cmd.SilenceUsage = true
			return err
		}
		return nil
	}
}
