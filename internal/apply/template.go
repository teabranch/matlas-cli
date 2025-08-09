package apply

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// TemplateProcessor handles environment variable substitution in configuration files
type TemplateProcessor struct {
	// StrictMode determines whether undefined variables cause errors or warnings
	StrictMode bool
	// Variables provides custom variable values (overrides environment)
	Variables map[string]string
	// DebugMode enables verbose substitution logging
	DebugMode bool
}

// SubstitutionResult contains the result of template substitution
type SubstitutionResult struct {
	Content   string              `json:"content"`
	Errors    []SubstitutionError `json:"errors,omitempty"`
	Warnings  []SubstitutionError `json:"warnings,omitempty"`
	Variables map[string]string   `json:"variables,omitempty"` // Variables that were substituted
}

// SubstitutionError represents an error or warning during substitution
type SubstitutionError struct {
	Variable string `json:"variable"`
	Position int    `json:"position"`
	Message  string `json:"message"`
	Type     string `json:"type"` // "error" or "warning"
}

// Error implements the error interface
func (se SubstitutionError) Error() string {
	return fmt.Sprintf("variable '%s' at position %d: %s", se.Variable, se.Position, se.Message)
}

// NewTemplateProcessor creates a new template processor with default settings
func NewTemplateProcessor() *TemplateProcessor {
	return &TemplateProcessor{
		StrictMode: false,
		Variables:  make(map[string]string),
		DebugMode:  false,
	}
}

// WithStrictMode enables strict mode (undefined variables cause errors)
func (tp *TemplateProcessor) WithStrictMode(strict bool) *TemplateProcessor {
	tp.StrictMode = strict
	return tp
}

// WithDebugMode enables debug logging for substitutions
func (tp *TemplateProcessor) WithDebugMode(debug bool) *TemplateProcessor {
	tp.DebugMode = debug
	return tp
}

// WithVariables sets custom variables that override environment variables
func (tp *TemplateProcessor) WithVariables(vars map[string]string) *TemplateProcessor {
	if tp.Variables == nil {
		tp.Variables = make(map[string]string)
	}
	for k, v := range vars {
		tp.Variables[k] = v
	}
	return tp
}

// SubstituteEnvVars processes a string and substitutes environment variables
func (tp *TemplateProcessor) SubstituteEnvVars(content string) *SubstitutionResult {
	result := &SubstitutionResult{
		Content:   content,
		Variables: make(map[string]string),
	}

	// Define regex patterns for different substitution types
	patterns := []struct {
		name    string
		regex   *regexp.Regexp
		handler func(match []string, position int) (string, error)
	}{
		{
			name:    "conditional_set",
			regex:   regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*):(\+)([^}]*)\}`),
			handler: tp.handleConditionalSet,
		},
		{
			name:    "conditional_error",
			regex:   regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*):(\?)([^}]*)\}`),
			handler: tp.handleConditionalError,
		},
		{
			name:    "with_default_colon",
			regex:   regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*):(-[^}]*)\}`),
			handler: tp.handleWithDefault,
		},
		{
			name:    "with_default_simple",
			regex:   regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)-([^}]*)\}`),
			handler: tp.handleSimpleDefault,
		},
		{
			name:    "simple",
			regex:   regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`),
			handler: tp.handleSimple,
		},
	}

	// Protect escape sequences first
	result.Content = tp.protectEscapeSequences(result.Content)

	// Process each pattern in order of specificity
	for _, pattern := range patterns {
		result.Content = tp.processPattern(result.Content, pattern.regex, pattern.handler, result)
	}

	// Handle nested substitutions (repeat until no more substitutions occur)
	maxIterations := 10
	iteration := 0
	for iteration < maxIterations {
		oldContent := result.Content
		for _, pattern := range patterns {
			result.Content = tp.processPattern(result.Content, pattern.regex, pattern.handler, result)
		}
		if result.Content == oldContent {
			break // No more substitutions
		}
		iteration++
	}

	// Restore escape sequences at the very end, after all processing
	result.Content = tp.restoreEscapeSequences(result.Content)

	if iteration == maxIterations {
		result.Warnings = append(result.Warnings, SubstitutionError{
			Variable: "",
			Position: 0,
			Message:  "maximum substitution iterations reached, possible circular reference",
			Type:     "warning",
		})
	}

	return result
}

// processPattern applies a regex pattern and substitution handler to content
func (tp *TemplateProcessor) processPattern(content string, regex *regexp.Regexp, handler func([]string, int) (string, error), result *SubstitutionResult) string {
	return regex.ReplaceAllStringFunc(content, func(match string) string {
		matches := regex.FindStringSubmatch(match)
		if len(matches) == 0 {
			return match
		}

		// Find position of match in original content
		position := strings.Index(content, match)

		replacement, err := handler(matches, position)
		if err != nil {
			if tp.StrictMode {
				result.Errors = append(result.Errors, SubstitutionError{
					Variable: matches[1],
					Position: position,
					Message:  err.Error(),
					Type:     "error",
				})
				return match // Keep original on error in strict mode
			} else {
				result.Warnings = append(result.Warnings, SubstitutionError{
					Variable: matches[1],
					Position: position,
					Message:  err.Error(),
					Type:     "warning",
				})
				return match // Keep original on error in non-strict mode
			}
		}

		// Record successful substitution
		if len(matches) > 1 {
			result.Variables[matches[1]] = replacement
		}

		if tp.DebugMode {
			fmt.Printf("Template substitution: %s -> %s\n", match, replacement)
		}

		return replacement
	})
}

// handleSimple handles simple variable substitution: ${VAR}
func (tp *TemplateProcessor) handleSimple(matches []string, position int) (string, error) {
	if len(matches) < 2 {
		return "", fmt.Errorf("invalid match pattern")
	}

	varName := matches[1]
	value := tp.getVariable(varName)

	if value == "" && !tp.hasVariable(varName) {
		return "", fmt.Errorf("undefined variable '%s'", varName)
	}

	return value, nil
}

// handleWithDefault handles variable substitution with colon default: ${VAR:-default}
func (tp *TemplateProcessor) handleWithDefault(matches []string, position int) (string, error) {
	if len(matches) < 3 {
		return "", fmt.Errorf("invalid match pattern")
	}

	varName := matches[1]
	defaultWithOperator := matches[2]       // This is "-default" for ${VAR:-default}
	defaultValue := defaultWithOperator[1:] // Remove the leading dash

	value := tp.getVariable(varName)

	// ${VAR:-default} - use default if variable is unset or empty
	if !tp.hasVariable(varName) || value == "" {
		return defaultValue, nil
	}
	return value, nil
}

// handleSimpleDefault handles variable substitution with simple default: ${VAR-default}
func (tp *TemplateProcessor) handleSimpleDefault(matches []string, position int) (string, error) {
	if len(matches) < 3 {
		return "", fmt.Errorf("invalid match pattern")
	}

	varName := matches[1]
	defaultValue := matches[2]

	value := tp.getVariable(varName)

	// ${VAR-default} - use default if variable is unset
	if !tp.hasVariable(varName) {
		return defaultValue, nil
	}
	return value, nil
}

// handleConditionalSet handles conditional substitution when variable is set: ${VAR:+value}
func (tp *TemplateProcessor) handleConditionalSet(matches []string, position int) (string, error) {
	if len(matches) < 4 {
		return "", fmt.Errorf("invalid match pattern")
	}

	varName := matches[1]
	conditionalValue := matches[3]

	if tp.hasVariable(varName) && tp.getVariable(varName) != "" {
		return conditionalValue, nil
	}

	return "", nil // Return empty string if variable is not set or empty
}

// handleConditionalError handles conditional error when variable is unset: ${VAR:?error}
func (tp *TemplateProcessor) handleConditionalError(matches []string, position int) (string, error) {
	if len(matches) < 4 {
		return "", fmt.Errorf("invalid match pattern")
	}

	varName := matches[1]
	errorMessage := matches[3]

	if !tp.hasVariable(varName) || tp.getVariable(varName) == "" {
		if errorMessage == "" {
			return "", fmt.Errorf("variable '%s' is required", varName)
		}
		return "", fmt.Errorf("%s", errorMessage)
	}

	return tp.getVariable(varName), nil
}

// protectEscapeSequences replaces escape sequences with placeholders before processing
func (tp *TemplateProcessor) protectEscapeSequences(content string) string {
	// Replace escaped sequences with a unique placeholder that won't match variable patterns
	placeholder := "__MATLAS_ESCAPED_DOLLAR__"
	return strings.ReplaceAll(content, "\\${", placeholder)
}

// restoreEscapeSequences converts placeholders back to literal ${} text
func (tp *TemplateProcessor) restoreEscapeSequences(content string) string {
	// After all substitutions, restore escaped sequences as literal text
	placeholder := "__MATLAS_ESCAPED_DOLLAR__"
	return strings.ReplaceAll(content, placeholder, "${")
}

// getVariable retrieves a variable value, checking custom variables first, then environment
func (tp *TemplateProcessor) getVariable(name string) string {
	// Check custom variables first
	if value, exists := tp.Variables[name]; exists {
		return value
	}

	// Fall back to environment variables
	return os.Getenv(name)
}

// hasVariable checks if a variable exists in custom variables or environment
func (tp *TemplateProcessor) hasVariable(name string) bool {
	// Check custom variables first
	if _, exists := tp.Variables[name]; exists {
		return true
	}

	// Check environment variables
	_, exists := os.LookupEnv(name)
	return exists
}

// ValidateTemplate checks a template for syntax errors without substitution
func (tp *TemplateProcessor) ValidateTemplate(content string) *SubstitutionResult {
	result := &SubstitutionResult{
		Content:   content,
		Variables: make(map[string]string),
	}

	// Check for unmatched braces
	openBraces := strings.Count(content, "${")
	closeBraces := strings.Count(content, "}")

	if openBraces != closeBraces {
		result.Errors = append(result.Errors, SubstitutionError{
			Variable: "",
			Position: 0,
			Message:  fmt.Sprintf("unmatched braces: %d opening ${, %d closing }", openBraces, closeBraces),
			Type:     "error",
		})
	}

	// Check for invalid variable names (simple variables only, not with operators)
	invalidVarRegex := regexp.MustCompile(`\$\{([^A-Za-z_:][^}]*)\}`)
	matches := invalidVarRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			position := strings.Index(content, match[0])
			result.Errors = append(result.Errors, SubstitutionError{
				Variable: match[1],
				Position: position,
				Message:  "invalid variable name (must start with letter or underscore, contain only alphanumeric and underscore)",
				Type:     "error",
			})
		}
	}

	// Check for nested braces that might cause issues
	nestedRegex := regexp.MustCompile(`\$\{[^}]*\$\{[^}]*\}[^}]*\}`)
	if nestedRegex.MatchString(content) {
		result.Warnings = append(result.Warnings, SubstitutionError{
			Variable: "",
			Position: 0,
			Message:  "nested variable substitutions detected, ensure proper escaping",
			Type:     "warning",
		})
	}

	return result
}

// ExtractVariables extracts all variable references from a template
func (tp *TemplateProcessor) ExtractVariables(content string) []string {
	var variables []string
	seen := make(map[string]bool)

	// Pattern to match all variable references
	regex := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?:[:][+-?]?[^}]*)?\}`)
	matches := regex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !seen[varName] {
				variables = append(variables, varName)
				seen[varName] = true
			}
		}
	}

	return variables
}

// ProcessFile processes a file and returns the substituted content
func (tp *TemplateProcessor) ProcessFile(content string) (*SubstitutionResult, error) {
	// First validate the template
	validation := tp.ValidateTemplate(content)
	if len(validation.Errors) > 0 {
		return validation, fmt.Errorf("template validation failed: %d errors", len(validation.Errors))
	}

	// Perform substitution
	result := tp.SubstituteEnvVars(content)

	// Merge validation warnings
	result.Warnings = append(result.Warnings, validation.Warnings...)

	// Check if we have errors in strict mode
	if tp.StrictMode && len(result.Errors) > 0 {
		return result, fmt.Errorf("template substitution failed: %d errors", len(result.Errors))
	}

	return result, nil
}

// GetEnvironmentSnapshot returns a snapshot of all environment variables
func GetEnvironmentSnapshot() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}
	return env
}

// FilterEnvironmentVariables filters environment variables by prefix
func FilterEnvironmentVariables(prefix string) map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 && strings.HasPrefix(pair[0], prefix) {
			env[pair[0]] = pair[1]
		}
	}
	return env
}
