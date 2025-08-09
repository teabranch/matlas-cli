package cli

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
)

// ErrorFormatter provides user-friendly error formatting
type ErrorFormatter struct {
	verbose bool
}

// NewErrorFormatter creates a new error formatter
func NewErrorFormatter(verbose bool) *ErrorFormatter {
	return &ErrorFormatter{verbose: verbose}
}

// Format converts an error to a user-friendly message
func (e *ErrorFormatter) Format(err error) string {
	if err == nil {
		return ""
	}

	// Handle Atlas client errors
	if errors.Is(err, atlasclient.ErrNotFound) {
		return "Resource not found. Please check your project ID, cluster name, or resource identifier."
	}

	if errors.Is(err, atlasclient.ErrUnauthorized) {
		return "Access denied. Please check your API key and public key or ensure you have the necessary permissions.\n" +
			"Hint: Set your keys using the ATLAS_API_KEY and ATLAS_PUB_KEY environment variables or --api-key and --pub-key flags."
	}

	if errors.Is(err, atlasclient.ErrConflict) {
		return "Resource conflict. The resource already exists or is in a conflicting state.\n" +
			"Hint: Check if the resource name is already taken."
	}

	if errors.Is(err, atlasclient.ErrTransient) {
		return "Temporary error occurred (rate limit or server issue). Please wait and try again.\n" +
			"Hint: Use --timeout flag to increase the wait time for automatic retries."
	}

	// Handle HTTP status codes in error messages
	errStr := err.Error()
	if statusCode := extractHTTPStatus(errStr); statusCode > 0 {
		return e.formatHTTPError(statusCode, errStr)
	}

	// Handle common validation errors
	if strings.Contains(errStr, "required") {
		return fmt.Sprintf("Missing required parameter: %s", err.Error())
	}

	if strings.Contains(errStr, "invalid") || strings.Contains(errStr, "must be") {
		return fmt.Sprintf("Invalid input: %s", err.Error())
	}

	// For network/timeout errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "context deadline exceeded") {
		return "Operation timed out. Try increasing the timeout with --timeout flag or check your network connection."
	}

	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "network") {
		return "Network connection failed. Please check your internet connection and try again."
	}

	// Default formatting
	if e.verbose {
		return fmt.Sprintf("Error: %s", err.Error())
	}

	// For non-verbose mode, try to extract the most relevant part
	if parts := strings.Split(errStr, ":"); len(parts) > 1 {
		// Return the last part which is usually the most specific error
		return strings.TrimSpace(parts[len(parts)-1])
	}

	return err.Error()
}

// formatHTTPError formats errors based on HTTP status codes
func (e *ErrorFormatter) formatHTTPError(statusCode int, originalError string) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "Bad request: Please check your input parameters.\n" +
			"Hint: Use --verbose for more details."
	case http.StatusUnauthorized:
		return "Authentication failed. Please check your API credentials.\n" +
			"Hint: Set ATLAS_API_KEY and ATLAS_PUB_KEY environment variables or use --api-key and --pub-key flags."
	case http.StatusForbidden:
		return "Access denied. You don't have permission for this operation.\n" +
			"Hint: Check your Atlas user permissions and project roles."
	case http.StatusNotFound:
		return "Resource not found. Please verify the resource exists and check your identifiers."
	case http.StatusConflict:
		return "Resource conflict. The resource already exists or is in a conflicting state.\n" +
			"Hint: Check if the resource name is already taken."
	case http.StatusTooManyRequests:
		return "Rate limit exceeded. Please wait before making more requests.\n" +
			"Hint: The CLI will automatically retry with exponential backoff."
	case http.StatusInternalServerError:
		return "Internal server error. Please try again later.\n" +
			"Hint: If the problem persists, contact MongoDB Atlas support."
	case http.StatusServiceUnavailable:
		return "Service temporarily unavailable. Please try again later."
	default:
		if e.verbose {
			return fmt.Sprintf("HTTP %d error: %s", statusCode, originalError)
		}
		return fmt.Sprintf("Request failed with status %d. Use --verbose for more details.", statusCode)
	}
}

// extractHTTPStatus tries to extract HTTP status code from error string
func extractHTTPStatus(errStr string) int {
	// Common patterns for HTTP status codes in error messages
	patterns := []string{
		"status code: ",
		"HTTP ",
		"status ",
	}

	for _, pattern := range patterns {
		if idx := strings.Index(strings.ToLower(errStr), strings.ToLower(pattern)); idx >= 0 {
			start := idx + len(pattern)
			if start < len(errStr) {
				// Extract up to 3 digits
				var codeStr strings.Builder
				for i := start; i < len(errStr) && i < start+3; i++ {
					if errStr[i] >= '0' && errStr[i] <= '9' {
						codeStr.WriteByte(errStr[i])
					} else {
						break
					}
				}
				if codeStr.Len() == 3 {
					var code int
					if n, err := fmt.Sscanf(codeStr.String(), "%d", &code); n == 1 && err == nil {
						return code
					}
				}
			}
		}
	}
	return 0
}

// FormatValidationError formats validation errors with helpful context
func FormatValidationError(field, value, reason string) error {
	return fmt.Errorf("validation failed for %s '%s': %s", field, value, reason)
}

// WrapWithSuggestion wraps an error with a helpful suggestion
func WrapWithSuggestion(err error, suggestion string) error {
	return fmt.Errorf("%w\nHint: %s", err, suggestion)
}
