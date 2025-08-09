package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
)

// ErrorRecoveryManager provides advanced error handling with recovery suggestions
type ErrorRecoveryManager struct {
	verbose          bool
	context          ErrorContext
	suggestionEngine *SuggestionEngine
}

// ErrorContext provides context about the current operation for better error handling
type ErrorContext struct {
	Command       string                 `json:"command"`
	Resource      string                 `json:"resource"`
	Operation     string                 `json:"operation"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// ErrorSuggestion represents a suggested fix for an error
type ErrorSuggestion struct {
	Type           string   `json:"type"`               // "fix", "workaround", "documentation"
	Priority       int      `json:"priority"`           // 1-5, where 1 is highest priority
	Title          string   `json:"title"`              // Short description
	Description    string   `json:"description"`        // Detailed explanation
	Commands       []string `json:"commands,omitempty"` // CLI commands to run
	Links          []string `json:"links,omitempty"`    // Documentation links
	AutoApplicable bool     `json:"autoApplicable"`     // Can be automatically applied
}

// RecoveryResult contains the result of error analysis and recovery suggestions
type RecoveryResult struct {
	Error       error             `json:"error"`
	ErrorType   string            `json:"errorType"`
	Severity    string            `json:"severity"`
	Suggestions []ErrorSuggestion `json:"suggestions"`
	Context     ErrorContext      `json:"context"`
	Recoverable bool              `json:"recoverable"`
}

// SuggestionEngine generates context-aware suggestions for errors
type SuggestionEngine struct {
	knowledgeBase map[string][]ErrorSuggestion
}

// NewErrorRecoveryManager creates a new error recovery manager
func NewErrorRecoveryManager(verbose bool) *ErrorRecoveryManager {
	return &ErrorRecoveryManager{
		verbose:          verbose,
		suggestionEngine: NewSuggestionEngine(),
	}
}

// SetContext sets the current operation context for better error analysis
func (erm *ErrorRecoveryManager) SetContext(command, resource, operation string, config map[string]interface{}) {
	erm.context = ErrorContext{
		Command:       command,
		Resource:      resource,
		Operation:     operation,
		Configuration: config,
		Timestamp:     time.Now(),
	}
}

// AnalyzeAndRecover analyzes an error and provides recovery suggestions
func (erm *ErrorRecoveryManager) AnalyzeAndRecover(ctx context.Context, err error) *RecoveryResult {
	if err == nil {
		return nil
	}

	result := &RecoveryResult{
		Error:   err,
		Context: erm.context,
	}

	// Determine error type and severity
	result.ErrorType, result.Severity = erm.classifyError(err)
	result.Recoverable = erm.isErrorRecoverable(err)

	// Generate suggestions based on error type
	result.Suggestions = erm.generateSuggestions(err, result.ErrorType, erm.context)

	return result
}

// Format converts a RecoveryResult to a user-friendly message
func (erm *ErrorRecoveryManager) Format(result *RecoveryResult) string {
	if result == nil || result.Error == nil {
		return ""
	}

	var output strings.Builder

	// Error header
	output.WriteString(fmt.Sprintf("❌ %s Error in %s %s\n",
		strings.Title(result.Severity), result.Context.Command, result.Context.Operation))
	output.WriteString(fmt.Sprintf("   %s\n\n", result.Error.Error()))

	// Context information
	if erm.verbose && result.Context.Resource != "" {
		output.WriteString(fmt.Sprintf("📋 Context:\n"))
		output.WriteString(fmt.Sprintf("   Resource: %s\n", result.Context.Resource))
		output.WriteString(fmt.Sprintf("   Operation: %s\n", result.Context.Operation))
		output.WriteString(fmt.Sprintf("   Time: %s\n\n", result.Context.Timestamp.Format(time.RFC3339)))
	}

	// Recovery suggestions
	if len(result.Suggestions) > 0 {
		output.WriteString("💡 Suggested Solutions:\n\n")

		for i, suggestion := range result.Suggestions {
			priority := strings.Repeat("⭐", suggestion.Priority)
			output.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, priority, suggestion.Title))
			output.WriteString(fmt.Sprintf("   %s\n", suggestion.Description))

			if len(suggestion.Commands) > 0 {
				output.WriteString("   Commands to try:\n")
				for _, cmd := range suggestion.Commands {
					output.WriteString(fmt.Sprintf("   $ %s\n", cmd))
				}
			}

			if len(suggestion.Links) > 0 && erm.verbose {
				output.WriteString("   Documentation:\n")
				for _, link := range suggestion.Links {
					output.WriteString(fmt.Sprintf("   📖 %s\n", link))
				}
			}
			output.WriteString("\n")
		}
	}

	// Recovery status
	if result.Recoverable {
		output.WriteString("✅ This error can typically be resolved by following the suggestions above.\n")
	} else {
		output.WriteString("⚠️  This error may require manual intervention or Atlas support.\n")
	}

	return output.String()
}

// classifyError determines the error type and severity
func (erm *ErrorRecoveryManager) classifyError(err error) (errorType, severity string) {
	errStr := strings.ToLower(err.Error())

	// Atlas API errors
	if atlasclient.IsNotFound(err) {
		return "not_found", "error"
	}
	if atlasclient.IsUnauthorized(err) {
		return "authentication", "critical"
	}
	if atlasclient.IsConflict(err) {
		return "conflict", "error"
	}
	if atlasclient.IsTransient(err) {
		return "transient", "warning"
	}

	// Network errors
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "timeout") {
		return "network", "warning"
	}

	// Validation errors
	if strings.Contains(errStr, "invalid") || strings.Contains(errStr, "required") {
		return "validation", "error"
	}

	// Configuration errors
	if strings.Contains(errStr, "config") || strings.Contains(errStr, "yaml") {
		return "configuration", "error"
	}

	// Permission errors
	if strings.Contains(errStr, "permission") || strings.Contains(errStr, "forbidden") {
		return "permission", "error"
	}

	// Quota errors
	if strings.Contains(errStr, "quota") || strings.Contains(errStr, "limit") {
		return "quota", "error"
	}

	return "unknown", "error"
}

// isErrorRecoverable determines if an error can be automatically or easily resolved
func (erm *ErrorRecoveryManager) isErrorRecoverable(err error) bool {
	if atlasclient.IsTransient(err) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	recoverablePatterns := []string{
		"validation", "invalid", "required", "format", "syntax",
		"connection", "timeout", "not found", "conflict",
	}

	for _, pattern := range recoverablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// generateSuggestions creates context-aware suggestions for error recovery
func (erm *ErrorRecoveryManager) generateSuggestions(err error, errorType string, context ErrorContext) []ErrorSuggestion {
	suggestions := erm.suggestionEngine.GetSuggestions(errorType, err.Error(), context)

	// Add context-specific suggestions
	contextSuggestions := erm.generateContextSpecificSuggestions(err, context)
	suggestions = append(suggestions, contextSuggestions...)

	// Sort by priority
	return erm.sortSuggestionsByPriority(suggestions)
}

// generateContextSpecificSuggestions creates suggestions based on the current operation context
func (erm *ErrorRecoveryManager) generateContextSpecificSuggestions(err error, context ErrorContext) []ErrorSuggestion {
	var suggestions []ErrorSuggestion

	switch context.Command {
	case "atlas":
		suggestions = append(suggestions, erm.generateAtlasSuggestions(err, context)...)
	case "apply":
		suggestions = append(suggestions, erm.generateApplySuggestions(err, context)...)
	case "database":
		suggestions = append(suggestions, erm.generateDatabaseSuggestions(err, context)...)
	case "config":
		suggestions = append(suggestions, erm.generateConfigSuggestions(err, context)...)
	}

	return suggestions
}

// generateAtlasSuggestions creates Atlas-specific suggestions
func (erm *ErrorRecoveryManager) generateAtlasSuggestions(err error, context ErrorContext) []ErrorSuggestion {
	var suggestions []ErrorSuggestion
	errStr := strings.ToLower(err.Error())

	if atlasclient.IsUnauthorized(err) {
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Priority:    1,
			Title:       "Check API credentials",
			Description: "Verify your Atlas API key and public key are correctly configured",
			Commands: []string{
				"export ATLAS_API_KEY='your-api-key'",
				"export ATLAS_PUB_KEY='your-public-key'",
				"matlas config validate",
			},
			Links:          []string{"https://docs.atlas.mongodb.com/configure-api-access/"},
			AutoApplicable: false,
		})
	}

	if atlasclient.IsNotFound(err) && context.Resource == "cluster" {
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Priority:    1,
			Title:       "Verify cluster exists",
			Description: "Check if the cluster name is correct and exists in your project",
			Commands: []string{
				"matlas atlas clusters list",
				fmt.Sprintf("matlas atlas clusters get %s", context.Configuration["name"]),
			},
			AutoApplicable: false,
		})
	}

	if strings.Contains(errStr, "project") && strings.Contains(errStr, "not found") {
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Priority:    1,
			Title:       "Check project ID",
			Description: "Verify the project ID is correct and you have access to it",
			Commands: []string{
				"matlas atlas projects list",
				"matlas config validate",
			},
			AutoApplicable: false,
		})
	}

	return suggestions
}

// generateApplySuggestions creates apply-specific suggestions
func (erm *ErrorRecoveryManager) generateApplySuggestions(err error, context ErrorContext) []ErrorSuggestion {
	var suggestions []ErrorSuggestion
	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "yaml") || strings.Contains(errStr, "syntax") {
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Priority:    1,
			Title:       "Fix YAML syntax",
			Description: "Your configuration file has YAML syntax errors",
			Commands: []string{
				"matlas apply validate your-config.yaml",
				"matlas config template generate basic > new-config.yaml",
			},
			Links:          []string{"https://yaml.org/", "https://www.yamllint.com/"},
			AutoApplicable: false,
		})
	}

	if strings.Contains(errStr, "validation") {
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Priority:    1,
			Title:       "Fix validation errors",
			Description: "Your configuration has validation errors that need to be resolved",
			Commands: []string{
				"matlas apply validate --verbose your-config.yaml",
				"matlas apply plan your-config.yaml",
			},
			AutoApplicable: false,
		})
	}

	if strings.Contains(errStr, "dependency") || strings.Contains(errStr, "circular") {
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Priority:    1,
			Title:       "Resolve dependency issues",
			Description: "There are dependency conflicts in your configuration",
			Commands: []string{
				"matlas apply plan --show-dependencies your-config.yaml",
			},
			AutoApplicable: false,
		})
	}

	return suggestions
}

// generateDatabaseSuggestions creates database-specific suggestions
func (erm *ErrorRecoveryManager) generateDatabaseSuggestions(err error, context ErrorContext) []ErrorSuggestion {
	var suggestions []ErrorSuggestion
	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "connection") {
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Priority:    1,
			Title:       "Check database connection",
			Description: "Verify your connection string and network access",
			Commands: []string{
				"matlas database list --connection-string your-connection-string",
				"matlas atlas network list",
			},
			AutoApplicable: false,
		})
	}

	if strings.Contains(errStr, "authentication") {
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Priority:    1,
			Title:       "Check database credentials",
			Description: "Verify database username and password are correct",
			Commands: []string{
				"matlas atlas users list",
			},
			AutoApplicable: false,
		})
	}

	return suggestions
}

// generateConfigSuggestions creates config-specific suggestions
func (erm *ErrorRecoveryManager) generateConfigSuggestions(err error, context ErrorContext) []ErrorSuggestion {
	var suggestions []ErrorSuggestion
	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "not found") {
		suggestions = append(suggestions, ErrorSuggestion{
			Type:        "fix",
			Priority:    1,
			Title:       "Create configuration file",
			Description: "Configuration file doesn't exist, create a new one",
			Commands: []string{
				"matlas config template generate basic > ~/.matlas/config.yaml",
				"matlas config validate",
			},
			AutoApplicable: true,
		})
	}

	return suggestions
}

// sortSuggestionsByPriority sorts suggestions by priority (1 = highest)
func (erm *ErrorRecoveryManager) sortSuggestionsByPriority(suggestions []ErrorSuggestion) []ErrorSuggestion {
	// Simple bubble sort by priority
	for i := 0; i < len(suggestions)-1; i++ {
		for j := 0; j < len(suggestions)-i-1; j++ {
			if suggestions[j].Priority > suggestions[j+1].Priority {
				suggestions[j], suggestions[j+1] = suggestions[j+1], suggestions[j]
			}
		}
	}
	return suggestions
}

// NewSuggestionEngine creates a new suggestion engine with predefined knowledge base
func NewSuggestionEngine() *SuggestionEngine {
	engine := &SuggestionEngine{
		knowledgeBase: make(map[string][]ErrorSuggestion),
	}
	engine.initializeKnowledgeBase()
	return engine
}

// GetSuggestions retrieves suggestions for a given error type
func (se *SuggestionEngine) GetSuggestions(errorType, errorMessage string, context ErrorContext) []ErrorSuggestion {
	suggestions := se.knowledgeBase[errorType]

	// Filter suggestions based on error message content
	var relevantSuggestions []ErrorSuggestion
	for _, suggestion := range suggestions {
		if se.isSuggestionRelevant(suggestion, errorMessage, context) {
			relevantSuggestions = append(relevantSuggestions, suggestion)
		}
	}

	return relevantSuggestions
}

// isSuggestionRelevant checks if a suggestion is relevant to the specific error
func (se *SuggestionEngine) isSuggestionRelevant(suggestion ErrorSuggestion, errorMessage string, context ErrorContext) bool {
	// For now, return all suggestions - in production this would have more sophisticated matching
	return true
}

// initializeKnowledgeBase populates the knowledge base with common error patterns and solutions
func (se *SuggestionEngine) initializeKnowledgeBase() {
	// Authentication errors
	se.knowledgeBase["authentication"] = []ErrorSuggestion{
		{
			Type:        "fix",
			Priority:    1,
			Title:       "Verify API credentials",
			Description: "Check that your Atlas API key and public key are correctly set",
			Commands:    []string{"matlas config validate", "echo $ATLAS_API_KEY"},
			Links:       []string{"https://docs.atlas.mongodb.com/configure-api-access/"},
		},
		{
			Type:        "workaround",
			Priority:    2,
			Title:       "Use different authentication method",
			Description: "Try using configuration file instead of environment variables",
			Commands:    []string{"matlas config template generate auth > ~/.matlas/config.yaml"},
		},
	}

	// Network errors
	se.knowledgeBase["network"] = []ErrorSuggestion{
		{
			Type:        "fix",
			Priority:    1,
			Title:       "Check internet connection",
			Description: "Verify you have a stable internet connection",
			Commands:    []string{"ping atlas.mongodb.com", "curl -I https://cloud.mongodb.com"},
		},
		{
			Type:        "workaround",
			Priority:    2,
			Title:       "Increase timeout",
			Description: "Try increasing the timeout for the operation",
			Commands:    []string{"matlas --timeout 30s atlas clusters list"},
		},
	}

	// Validation errors
	se.knowledgeBase["validation"] = []ErrorSuggestion{
		{
			Type:        "fix",
			Priority:    1,
			Title:       "Fix configuration syntax",
			Description: "Check your configuration file for syntax errors",
			Commands:    []string{"matlas apply validate --verbose your-config.yaml"},
		},
		{
			Type:        "documentation",
			Priority:    3,
			Title:       "Review configuration schema",
			Description: "Check the configuration schema documentation",
			Links:       []string{"https://docs.atlas.mongodb.com/"},
		},
	}

	// Not found errors
	se.knowledgeBase["not_found"] = []ErrorSuggestion{
		{
			Type:        "fix",
			Priority:    1,
			Title:       "Verify resource exists",
			Description: "Check that the resource you're trying to access actually exists",
		},
		{
			Type:        "fix",
			Priority:    2,
			Title:       "Check resource identifiers",
			Description: "Verify project ID, cluster name, and other identifiers are correct",
		},
	}

	// Quota errors
	se.knowledgeBase["quota"] = []ErrorSuggestion{
		{
			Type:        "fix",
			Priority:    1,
			Title:       "Check Atlas quotas",
			Description: "You may have reached your Atlas resource limits",
			Links:       []string{"https://docs.atlas.mongodb.com/reference/atlas-limits/"},
		},
		{
			Type:        "workaround",
			Priority:    2,
			Title:       "Upgrade Atlas tier",
			Description: "Consider upgrading your Atlas organization tier for higher limits",
		},
	}
}
