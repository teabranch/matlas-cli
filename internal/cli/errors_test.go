package cli

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
)

func TestNewErrorFormatter(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{"verbose formatter", true},
		{"non-verbose formatter", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewErrorFormatter(tt.verbose)
			assert.NotNil(t, formatter)
			assert.Equal(t, tt.verbose, formatter.verbose)
		})
	}
}

func TestErrorFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		verbose  bool
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			verbose:  false,
			expected: "",
		},
		{
			name:     "atlas not found error",
			err:      atlasclient.ErrNotFound,
			verbose:  false,
			expected: "Resource not found. Please check your project ID, cluster name, or resource identifier.",
		},
		{
			name:     "atlas unauthorized error",
			err:      atlasclient.ErrUnauthorized,
			verbose:  false,
			expected: "Access denied. Please check your API key and public key or ensure you have the necessary permissions.\nHint: Set your keys using the ATLAS_API_KEY and ATLAS_PUB_KEY environment variables or --api-key and --pub-key flags.",
		},
		{
			name:     "atlas conflict error",
			err:      atlasclient.ErrConflict,
			verbose:  false,
			expected: "Resource conflict. The resource already exists or is in a conflicting state.\nHint: Check if the resource name is already taken.",
		},
		{
			name:     "atlas transient error",
			err:      atlasclient.ErrTransient,
			verbose:  false,
			expected: "Temporary error occurred (rate limit or server issue). Please wait and try again.\nHint: Use --timeout flag to increase the wait time for automatic retries.",
		},
		{
			name:     "required field error",
			err:      errors.New("field name is required"),
			verbose:  false,
			expected: "Missing required parameter: field name is required",
		},
		{
			name:     "invalid field error",
			err:      errors.New("field value is invalid"),
			verbose:  false,
			expected: "Invalid input: field value is invalid",
		},
		{
			name:     "timeout error",
			err:      errors.New("operation timeout exceeded"),
			verbose:  false,
			expected: "Operation timed out. Try increasing the timeout with --timeout flag or check your network connection.",
		},
		{
			name:     "context deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			verbose:  false,
			expected: "Operation timed out. Try increasing the timeout with --timeout flag or check your network connection.",
		},
		{
			name:     "connection error",
			err:      errors.New("connection refused"),
			verbose:  false,
			expected: "Network connection failed. Please check your internet connection and try again.",
		},
		{
			name:     "network error",
			err:      errors.New("network unreachable"),
			verbose:  false,
			expected: "Network connection failed. Please check your internet connection and try again.",
		},
		{
			name:     "generic error verbose",
			err:      errors.New("some generic error"),
			verbose:  true,
			expected: "Error: some generic error",
		},
		{
			name:     "generic error non-verbose",
			err:      errors.New("some generic error"),
			verbose:  false,
			expected: "some generic error",
		},
		{
			name:     "complex error with colons non-verbose",
			err:      errors.New("service: database: connection failed"),
			verbose:  false,
			expected: "Network connection failed. Please check your internet connection and try again.",
		},
		{
			name:     "HTTP status code in error",
			err:      errors.New("HTTP request failed with status code: 404"),
			verbose:  false,
			expected: "Resource not found. Please verify the resource exists and check your identifiers.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewErrorFormatter(tt.verbose)
			result := formatter.Format(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorFormatter_formatHTTPError(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		originalError string
		verbose       bool
		expected      string
	}{
		{
			name:          "400 bad request",
			statusCode:    http.StatusBadRequest,
			originalError: "invalid input",
			verbose:       false,
			expected:      "Bad request: Please check your input parameters.\nHint: Use --verbose for more details.",
		},
		{
			name:          "401 unauthorized",
			statusCode:    http.StatusUnauthorized,
			originalError: "invalid credentials",
			verbose:       false,
			expected:      "Authentication failed. Please check your API credentials.\nHint: Set ATLAS_API_KEY and ATLAS_PUB_KEY environment variables or use --api-key and --pub-key flags.",
		},
		{
			name:          "403 forbidden",
			statusCode:    http.StatusForbidden,
			originalError: "access denied",
			verbose:       false,
			expected:      "Access denied. You don't have permission for this operation.\nHint: Check your Atlas user permissions and project roles.",
		},
		{
			name:          "404 not found",
			statusCode:    http.StatusNotFound,
			originalError: "resource not found",
			verbose:       false,
			expected:      "Resource not found. Please verify the resource exists and check your identifiers.",
		},
		{
			name:          "409 conflict",
			statusCode:    http.StatusConflict,
			originalError: "resource exists",
			verbose:       false,
			expected:      "Resource conflict. The resource already exists or is in a conflicting state.\nHint: Check if the resource name is already taken.",
		},
		{
			name:          "429 rate limit",
			statusCode:    http.StatusTooManyRequests,
			originalError: "rate limit exceeded",
			verbose:       false,
			expected:      "Rate limit exceeded. Please wait before making more requests.\nHint: The CLI will automatically retry with exponential backoff.",
		},
		{
			name:          "500 internal server error",
			statusCode:    http.StatusInternalServerError,
			originalError: "internal error",
			verbose:       false,
			expected:      "Internal server error. Please try again later.\nHint: If the problem persists, contact MongoDB Atlas support.",
		},
		{
			name:          "503 service unavailable",
			statusCode:    http.StatusServiceUnavailable,
			originalError: "service down",
			verbose:       false,
			expected:      "Service temporarily unavailable. Please try again later.",
		},
		{
			name:          "unknown status verbose",
			statusCode:    418,
			originalError: "teapot error",
			verbose:       true,
			expected:      "HTTP 418 error: teapot error",
		},
		{
			name:          "unknown status non-verbose",
			statusCode:    418,
			originalError: "teapot error",
			verbose:       false,
			expected:      "Request failed with status 418. Use --verbose for more details.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewErrorFormatter(tt.verbose)
			result := formatter.formatHTTPError(tt.statusCode, tt.originalError)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		errStr   string
		expected int
	}{
		{
			name:     "status code pattern",
			errStr:   "request failed with status code: 404",
			expected: 404,
		},
		{
			name:     "HTTP pattern",
			errStr:   "HTTP 500 internal server error",
			expected: 500,
		},
		{
			name:     "status pattern",
			errStr:   "request failed with status 401",
			expected: 401,
		},
		{
			name:     "case insensitive",
			errStr:   "Request failed with STATUS CODE: 403",
			expected: 403,
		},
		{
			name:     "no status code",
			errStr:   "generic error message",
			expected: 0,
		},
		{
			name:     "invalid status code",
			errStr:   "status code: abc",
			expected: 0,
		},
		{
			name:     "partial status code",
			errStr:   "status code: 40",
			expected: 0,
		},
		{
			name:     "status code with extra characters",
			errStr:   "status code: 404 not found",
			expected: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHTTPStatus(tt.errStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    string
		reason   string
		expected string
	}{
		{
			name:     "standard validation error",
			field:    "username",
			value:    "invalid-user",
			reason:   "must contain only alphanumeric characters",
			expected: "validation failed for username 'invalid-user': must contain only alphanumeric characters",
		},
		{
			name:     "empty field",
			field:    "",
			value:    "test",
			reason:   "field cannot be empty",
			expected: "validation failed for  'test': field cannot be empty",
		},
		{
			name:     "empty value",
			field:    "password",
			value:    "",
			reason:   "cannot be empty",
			expected: "validation failed for password '': cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FormatValidationError(tt.field, tt.value, tt.reason)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestWrapWithSuggestion(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		suggestion string
		expected   string
	}{
		{
			name:       "wrap simple error",
			err:        errors.New("connection failed"),
			suggestion: "check your network connection",
			expected:   "connection failed\nHint: check your network connection",
		},
		{
			name:       "wrap formatted error",
			err:        fmt.Errorf("failed to connect to %s", "database"),
			suggestion: "ensure the database is running",
			expected:   "failed to connect to database\nHint: ensure the database is running",
		},
		{
			name:       "empty suggestion",
			err:        errors.New("some error"),
			suggestion: "",
			expected:   "some error\nHint: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapWithSuggestion(tt.err, tt.suggestion)
			assert.Equal(t, tt.expected, wrapped.Error())

			// Test that the original error is wrapped
			assert.True(t, errors.Is(wrapped, tt.err))
		})
	}
}
