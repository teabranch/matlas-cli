package apply

import (
	"os"
	"strings"
	"testing"
)

func TestNewTemplateProcessor(t *testing.T) {
	tp := NewTemplateProcessor()
	if tp == nil {
		t.Fatal("NewTemplateProcessor returned nil")
	}
	if tp.StrictMode {
		t.Error("Default strict mode should be false")
	}
	if tp.DebugMode {
		t.Error("Default debug mode should be false")
	}
	if tp.Variables == nil {
		t.Error("Variables map should be initialized")
	}
}

func TestTemplateProcessor_WithMethods(t *testing.T) {
	tp := NewTemplateProcessor().
		WithStrictMode(true).
		WithDebugMode(true).
		WithVariables(map[string]string{"TEST": "value"})

	if !tp.StrictMode {
		t.Error("StrictMode should be true")
	}
	if !tp.DebugMode {
		t.Error("DebugMode should be true")
	}
	if tp.Variables["TEST"] != "value" {
		t.Error("Variables should contain TEST=value")
	}
}

func TestSubstituteEnvVars_Simple(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["TEST_VAR"] = "test_value"

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple substitution",
			content:  "Hello ${TEST_VAR}!",
			expected: "Hello test_value!",
		},
		{
			name:     "multiple substitutions",
			content:  "${TEST_VAR} and ${TEST_VAR} again",
			expected: "test_value and test_value again",
		},
		{
			name:     "no substitutions",
			content:  "Hello world!",
			expected: "Hello world!",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.SubstituteEnvVars(tt.content)
			if result.Content != tt.expected {
				t.Errorf("Content = %q, want %q", result.Content, tt.expected)
			}
			if len(result.Errors) > 0 {
				t.Errorf("Unexpected errors: %v", result.Errors)
			}
		})
	}
}

func TestSubstituteEnvVars_WithDefaults(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["SET_VAR"] = "set_value"
	tp.Variables["EMPTY_VAR"] = ""

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "default for unset variable",
			content:  "${UNSET_VAR:-default_value}",
			expected: "default_value",
		},
		{
			name:     "default for empty variable",
			content:  "${EMPTY_VAR:-default_value}",
			expected: "default_value",
		},
		{
			name:     "no default for set variable",
			content:  "${SET_VAR:-default_value}",
			expected: "set_value",
		},
		{
			name:     "dash only default for unset",
			content:  "${UNSET_VAR-default_value}",
			expected: "default_value",
		},
		{
			name:     "dash only default keeps empty",
			content:  "${EMPTY_VAR-default_value}",
			expected: "",
		},
		{
			name:     "empty default value",
			content:  "${UNSET_VAR:-}",
			expected: "",
		},
		{
			name:     "default with spaces",
			content:  "${UNSET_VAR:-hello world}",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.SubstituteEnvVars(tt.content)
			if result.Content != tt.expected {
				t.Errorf("Content = %q, want %q", result.Content, tt.expected)
			}
			if len(result.Errors) > 0 {
				t.Errorf("Unexpected errors: %v", result.Errors)
			}
		})
	}
}

func TestSubstituteEnvVars_ConditionalSet(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["SET_VAR"] = "set_value"
	tp.Variables["EMPTY_VAR"] = ""

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "conditional value for set variable",
			content:  "${SET_VAR:+conditional_value}",
			expected: "conditional_value",
		},
		{
			name:     "no conditional value for unset variable",
			content:  "${UNSET_VAR:+conditional_value}",
			expected: "",
		},
		{
			name:     "no conditional value for empty variable",
			content:  "${EMPTY_VAR:+conditional_value}",
			expected: "",
		},
		{
			name:     "conditional with spaces",
			content:  "${SET_VAR:+hello world}",
			expected: "hello world",
		},
		{
			name:     "empty conditional value",
			content:  "${SET_VAR:+}",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.SubstituteEnvVars(tt.content)
			if result.Content != tt.expected {
				t.Errorf("Content = %q, want %q", result.Content, tt.expected)
			}
			if len(result.Errors) > 0 {
				t.Errorf("Unexpected errors: %v", result.Errors)
			}
		})
	}
}

func TestSubstituteEnvVars_ConditionalError(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["SET_VAR"] = "set_value"
	tp.Variables["EMPTY_VAR"] = ""

	tests := []struct {
		name          string
		content       string
		expected      string
		expectError   bool
		expectedError string
	}{
		{
			name:     "no error for set variable",
			content:  "${SET_VAR:?Variable is required}",
			expected: "set_value",
		},
		{
			name:          "error for unset variable with message",
			content:       "${UNSET_VAR:?Variable is required}",
			expected:      "${UNSET_VAR:?Variable is required}", // Kept as-is on error
			expectError:   true,
			expectedError: "Variable is required",
		},
		{
			name:          "error for empty variable",
			content:       "${EMPTY_VAR:?Variable cannot be empty}",
			expected:      "${EMPTY_VAR:?Variable cannot be empty}",
			expectError:   true,
			expectedError: "Variable cannot be empty",
		},
		{
			name:          "default error message",
			content:       "${UNSET_VAR:?}",
			expected:      "${UNSET_VAR:?}",
			expectError:   true,
			expectedError: "variable 'UNSET_VAR' is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.SubstituteEnvVars(tt.content)
			if result.Content != tt.expected {
				t.Errorf("Content = %q, want %q", result.Content, tt.expected)
			}

			if tt.expectError {
				if len(result.Warnings) == 0 {
					t.Error("Expected warnings but got none")
				} else if !strings.Contains(result.Warnings[0].Message, tt.expectedError) {
					t.Errorf("Warning message = %q, want to contain %q", result.Warnings[0].Message, tt.expectedError)
				}
			} else {
				if len(result.Warnings) > 0 {
					t.Errorf("Unexpected warnings: %v", result.Warnings)
				}
			}
		})
	}
}

func TestSubstituteEnvVars_StrictMode(t *testing.T) {
	tp := NewTemplateProcessor().WithStrictMode(true)

	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "undefined variable in strict mode",
			content:     "${UNDEFINED_VAR}",
			expectError: true,
		},
		{
			name:        "conditional error in strict mode",
			content:     "${UNDEFINED_VAR:?Required variable}",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.SubstituteEnvVars(tt.content)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected errors but got none")
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestSubstituteEnvVars_EscapeSequences(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["TEST_VAR"] = "test_value"

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "escaped variable",
			content:  "\\${TEST_VAR}",
			expected: "${TEST_VAR}",
		},
		{
			name:     "mixed escaped and real",
			content:  "\\${LITERAL} and ${TEST_VAR}",
			expected: "${LITERAL} and test_value",
		},
		{
			name:     "multiple escapes",
			content:  "\\${ONE} \\${TWO}",
			expected: "${ONE} ${TWO}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.SubstituteEnvVars(tt.content)
			if result.Content != tt.expected {
				t.Errorf("Content = %q, want %q", result.Content, tt.expected)
			}
		})
	}
}

func TestSubstituteEnvVars_NestedSubstitution(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["PREFIX"] = "TEST"
	tp.Variables["SUFFIX"] = "VAR"
	tp.Variables["TEST_VAR"] = "nested_value"

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "nested variable substitution",
			content:  "${${PREFIX}_${SUFFIX}}",
			expected: "nested_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.SubstituteEnvVars(tt.content)
			if result.Content != tt.expected {
				t.Errorf("Content = %q, want %q", result.Content, tt.expected)
			}
		})
	}
}

func TestSubstituteEnvVars_MaxIterations(t *testing.T) {
	tp := NewTemplateProcessor()
	// Create a potential infinite loop
	tp.Variables["A"] = "${B}"
	tp.Variables["B"] = "${A}"

	result := tp.SubstituteEnvVars("${A}")

	// Should stop after max iterations and warn
	if len(result.Warnings) == 0 {
		t.Error("Expected warning about max iterations")
	}

	found := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning.Message, "maximum substitution iterations") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning about maximum iterations")
	}
}

func TestValidateTemplate(t *testing.T) {
	tp := NewTemplateProcessor()

	tests := []struct {
		name        string
		content     string
		expectError bool
		errorText   string
	}{
		{
			name:    "valid template",
			content: "${VAR} and ${OTHER_VAR:-default}",
		},
		{
			name:        "unmatched opening brace",
			content:     "${VAR and ${OTHER}",
			expectError: true,
			errorText:   "unmatched braces",
		},
		{
			name:        "unmatched closing brace",
			content:     "${VAR} and OTHER}",
			expectError: true,
			errorText:   "unmatched braces",
		},
		{
			name:        "invalid variable name",
			content:     "${123INVALID}",
			expectError: true,
			errorText:   "invalid variable name",
		},
		{
			name:        "variable name starting with number",
			content:     "${0INVALID}",
			expectError: true,
			errorText:   "invalid variable name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.ValidateTemplate(tt.content)

			if tt.expectError {
				if len(result.Errors) == 0 {
					t.Error("Expected errors but got none")
				} else {
					found := false
					for _, err := range result.Errors {
						if strings.Contains(err.Message, tt.errorText) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing %q, got: %v", tt.errorText, result.Errors)
					}
				}
			} else {
				if len(result.Errors) > 0 {
					t.Errorf("Unexpected errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestExtractVariables(t *testing.T) {
	tp := NewTemplateProcessor()

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "simple variables",
			content:  "${VAR1} and ${VAR2}",
			expected: []string{"VAR1", "VAR2"},
		},
		{
			name:     "variables with defaults",
			content:  "${VAR1:-default} and ${VAR2:+conditional}",
			expected: []string{"VAR1", "VAR2"},
		},
		{
			name:     "duplicate variables",
			content:  "${VAR1} and ${VAR1} again",
			expected: []string{"VAR1"},
		},
		{
			name:     "no variables",
			content:  "plain text",
			expected: []string{},
		},
		{
			name:     "mixed variable types",
			content:  "${SIMPLE} ${WITH_DEFAULT:-def} ${CONDITIONAL:+val} ${ERROR:?msg}",
			expected: []string{"SIMPLE", "WITH_DEFAULT", "CONDITIONAL", "ERROR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.ExtractVariables(tt.content)

			if len(result) != len(tt.expected) {
				t.Errorf("Variables length = %d, want %d", len(result), len(tt.expected))
			}

			// Convert to map for easier comparison
			resultMap := make(map[string]bool)
			for _, v := range result {
				resultMap[v] = true
			}

			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("Expected variable %q not found in result: %v", expected, result)
				}
			}
		})
	}
}

func TestProcessFile(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["TEST_VAR"] = "test_value"

	tests := []struct {
		name        string
		content     string
		expected    string
		expectError bool
	}{
		{
			name:     "valid template",
			content:  "Hello ${TEST_VAR}!",
			expected: "Hello test_value!",
		},
		{
			name:        "invalid template",
			content:     "${INVALID",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tp.ProcessFile(tt.content)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result.Content != tt.expected {
					t.Errorf("Content = %q, want %q", result.Content, tt.expected)
				}
			}
		})
	}
}

func TestProcessFile_StrictMode(t *testing.T) {
	tp := NewTemplateProcessor().WithStrictMode(true)

	// Test with undefined variable
	result, err := tp.ProcessFile("${UNDEFINED_VAR}")

	if err == nil {
		t.Error("Expected error in strict mode for undefined variable")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected errors in result")
	}
}

func TestGetVariable(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["CUSTOM_VAR"] = "custom_value"

	// Set environment variable for testing
	os.Setenv("ENV_VAR", "env_value")
	defer os.Unsetenv("ENV_VAR")

	tests := []struct {
		name     string
		varName  string
		expected string
	}{
		{
			name:     "custom variable",
			varName:  "CUSTOM_VAR",
			expected: "custom_value",
		},
		{
			name:     "environment variable",
			varName:  "ENV_VAR",
			expected: "env_value",
		},
		{
			name:     "undefined variable",
			varName:  "UNDEFINED_VAR",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.getVariable(tt.varName)
			if result != tt.expected {
				t.Errorf("getVariable(%q) = %q, want %q", tt.varName, result, tt.expected)
			}
		})
	}
}

func TestHasVariable(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["CUSTOM_VAR"] = "custom_value"

	// Set environment variable for testing
	os.Setenv("ENV_VAR", "env_value")
	defer os.Unsetenv("ENV_VAR")

	tests := []struct {
		name     string
		varName  string
		expected bool
	}{
		{
			name:     "custom variable exists",
			varName:  "CUSTOM_VAR",
			expected: true,
		},
		{
			name:     "environment variable exists",
			varName:  "ENV_VAR",
			expected: true,
		},
		{
			name:     "undefined variable",
			varName:  "UNDEFINED_VAR",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tp.hasVariable(tt.varName)
			if result != tt.expected {
				t.Errorf("hasVariable(%q) = %v, want %v", tt.varName, result, tt.expected)
			}
		})
	}
}

func TestGetEnvironmentSnapshot(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_SNAPSHOT_VAR", "test_value")
	defer os.Unsetenv("TEST_SNAPSHOT_VAR")

	env := GetEnvironmentSnapshot()

	if env["TEST_SNAPSHOT_VAR"] != "test_value" {
		t.Errorf("Expected TEST_SNAPSHOT_VAR=test_value, got %q", env["TEST_SNAPSHOT_VAR"])
	}

	// Should contain at least some environment variables
	if len(env) == 0 {
		t.Error("Expected non-empty environment snapshot")
	}
}

func TestFilterEnvironmentVariables(t *testing.T) {
	// Set test environment variables
	os.Setenv("ATLAS_TEST_VAR1", "value1")
	os.Setenv("ATLAS_TEST_VAR2", "value2")

	// Clean up
	defer func() {
		os.Unsetenv("ATLAS_TEST_VAR1")
		os.Unsetenv("ATLAS_TEST_VAR2")
	}()

	// Test filtering
	env := FilterEnvironmentVariables("ATLAS_")

	if env["ATLAS_TEST_VAR1"] != "value1" {
		t.Errorf("Expected ATLAS_TEST_VAR1=value1, got %q", env["ATLAS_TEST_VAR1"])
	}

	if env["ATLAS_TEST_VAR2"] != "value2" {
		t.Errorf("Expected ATLAS_TEST_VAR2=value2, got %q", env["ATLAS_TEST_VAR2"])
	}

	// Test that all returned variables have the correct prefix
	for key := range env {
		// Should only contain ATLAS_ prefixed variables
		if len(key) > 0 && key[0] != '_' { // Skip empty keys and internal variables
			if !strings.HasPrefix(key, "ATLAS_") {
				t.Errorf("Variable %q should have ATLAS_ prefix", key)
			}
		}
	}
}

func TestSubstitutionError_Error(t *testing.T) {
	err := SubstitutionError{
		Variable: "TEST_VAR",
		Position: 10,
		Message:  "undefined variable",
		Type:     "error",
	}

	expected := "variable 'TEST_VAR' at position 10: undefined variable"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestSubstituteEnvVars_ComplexExample(t *testing.T) {
	tp := NewTemplateProcessor()
	tp.Variables["ENV"] = "production"
	tp.Variables["DB_HOST"] = "prod.example.com"
	tp.Variables["DB_PORT"] = "5432"
	tp.Variables["DEBUG"] = ""

	content := `
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: ${ENV}-project
  labels:
    environment: ${ENV}
    debug: ${DEBUG:+enabled}
spec:
  organizationId: ${ORG_ID:?Organization ID is required}
  clusters:
    - metadata:
        name: ${ENV}-cluster
      provider: AWS
      region: ${REGION:-US_EAST_1}
      instanceSize: ${INSTANCE_SIZE:-M10}
      connectionString: mongodb://${DB_HOST}:${DB_PORT}
`

	expected := `
apiVersion: matlas.mongodb.com/v1
kind: Project
metadata:
  name: production-project
  labels:
    environment: production
    debug: 
spec:
  organizationId: ${ORG_ID:?Organization ID is required}
  clusters:
    - metadata:
        name: production-cluster
      provider: AWS
      region: US_EAST_1
      instanceSize: M10
      connectionString: mongodb://prod.example.com:5432
`

	result := tp.SubstituteEnvVars(content)

	if result.Content != expected {
		t.Errorf("Complex substitution failed.\nGot:\n%s\nWant:\n%s", result.Content, expected)
	}

	// Should have warning about ORG_ID
	if len(result.Warnings) == 0 {
		t.Error("Expected warning about ORG_ID but got none")
	}
}
