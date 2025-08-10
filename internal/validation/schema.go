package validation

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SchemaValidator provides JSON schema validation for configuration files
type SchemaValidator struct {
	schemas map[string]*ConfigSchema
}

// ConfigSchema represents a configuration schema
type ConfigSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required"`
	Title      string                    `json:"title,omitempty"`
	Version    string                    `json:"version,omitempty"`
}

// PropertySchema represents a property within a schema
type PropertySchema struct {
	Type        string                    `json:"type"`
	Description string                    `json:"description,omitempty"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"`
	Items       *PropertySchema           `json:"items,omitempty"`
	Required    []string                  `json:"required,omitempty"`
	Enum        []interface{}             `json:"enum,omitempty"`
	Pattern     string                    `json:"pattern,omitempty"`
	MinLength   *int                      `json:"minLength,omitempty"`
	MaxLength   *int                      `json:"maxLength,omitempty"`
	Minimum     *float64                  `json:"minimum,omitempty"`
	Maximum     *float64                  `json:"maximum,omitempty"`
}

// SchemaValidationResult contains the result of schema validation
type SchemaValidationResult struct {
	Valid    bool                    `json:"valid"`
	Errors   []SchemaValidationError `json:"errors,omitempty"`
	Warnings []SchemaValidationError `json:"warnings,omitempty"`
	Schema   string                  `json:"schema,omitempty"`
}

// SchemaValidationError represents a schema validation error
type SchemaValidationError struct {
	Path     string      `json:"path"`
	Property string      `json:"property"`
	Value    interface{} `json:"value"`
	Expected string      `json:"expected"`
	Message  string      `json:"message"`
	Severity string      `json:"severity"`
}

// Error implements the error interface
func (sve SchemaValidationError) Error() string {
	return fmt.Sprintf("%s: %s", sve.Path, sve.Message)
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	validator := &SchemaValidator{
		schemas: make(map[string]*ConfigSchema),
	}
	validator.initializeBuiltinSchemas()
	return validator
}

// ValidateConfigWithSchema validates configuration data against a schema
func (sv *SchemaValidator) ValidateConfigWithSchema(configData []byte, schemaName string) (*SchemaValidationResult, error) {
	schema, exists := sv.schemas[schemaName]
	if !exists {
		return nil, fmt.Errorf("schema '%s' not found", schemaName)
	}

	// Parse configuration data as YAML first, then convert to interface{}
	var configObj interface{}
	if err := yaml.Unmarshal(configData, &configObj); err != nil {
		return &SchemaValidationResult{
			Valid: false,
			Errors: []SchemaValidationError{
				{
					Path:     "root",
					Property: "syntax",
					Message:  fmt.Sprintf("Invalid YAML syntax: %v", err),
					Severity: "error",
				},
			},
			Schema: schemaName,
		}, nil
	}

	// Validate against schema
	result := &SchemaValidationResult{
		Valid:  true,
		Schema: schemaName,
	}

	sv.validateObject(configObj, schema, "root", result)
	result.Valid = len(result.Errors) == 0

	return result, nil
}

// ValidateConfigFile validates a configuration file against a schema
func (sv *SchemaValidator) ValidateConfigFile(filePath, schemaName string) (*SchemaValidationResult, error) {
	configData, err := os.ReadFile(filePath) //nolint:gosec // reading user-specified path is expected for CLI tool
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	return sv.ValidateConfigWithSchema(configData, schemaName)
}

// LoadSchema loads a schema from a file
func (sv *SchemaValidator) LoadSchema(schemaPath, schemaName string) error {
	schemaData, err := os.ReadFile(schemaPath) //nolint:gosec // reading user-specified path is expected for CLI tool
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema ConfigSchema
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	sv.schemas[schemaName] = &schema
	return nil
}

// GetAvailableSchemas returns the list of available schema names
func (sv *SchemaValidator) GetAvailableSchemas() []string {
	var names []string
	for name := range sv.schemas {
		names = append(names, name)
	}
	return names
}

// validateObject validates an object against a schema
func (sv *SchemaValidator) validateObject(obj interface{}, schema *ConfigSchema, path string, result *SchemaValidationResult) {
	if schema.Type == "object" {
		objMap, ok := obj.(map[string]interface{})
		if !ok {
			sv.addError(result, path, "type", obj, "object",
				fmt.Sprintf("Expected object, got %T", obj), "error")
			return
		}

		// Check required properties
		for _, required := range schema.Required {
			if _, exists := objMap[required]; !exists {
				sv.addError(result, fmt.Sprintf("%s.%s", path, required), required, nil, "required",
					fmt.Sprintf("Required property '%s' is missing", required), "error")
			}
		}

		// Validate existing properties
		for propName, propValue := range objMap {
			propPath := fmt.Sprintf("%s.%s", path, propName)
			if propSchema, exists := schema.Properties[propName]; exists {
				sv.validateProperty(propValue, &propSchema, propPath, result)
			} else {
				sv.addError(result, propPath, propName, propValue, "unknown",
					fmt.Sprintf("Unknown property '%s'", propName), "warning")
			}
		}
	}
}

// validateProperty validates a property against its schema
func (sv *SchemaValidator) validateProperty(value interface{}, schema *PropertySchema, path string, result *SchemaValidationResult) {
	// Type validation
	if !sv.isValidType(value, schema.Type) {
		sv.addError(result, path, "type", value, schema.Type,
			fmt.Sprintf("Expected %s, got %T", schema.Type, value), "error")
		return
	}

	// Enum validation
	if len(schema.Enum) > 0 {
		if !sv.isInEnum(value, schema.Enum) {
			sv.addError(result, path, "enum", value, "one of enum values",
				fmt.Sprintf("Value must be one of: %v", schema.Enum), "error")
		}
	}

	// String-specific validations
	if schema.Type == "string" {
		if str, ok := value.(string); ok {
			if schema.MinLength != nil && len(str) < *schema.MinLength {
				sv.addError(result, path, "minLength", value, fmt.Sprintf("min %d chars", *schema.MinLength),
					fmt.Sprintf("String too short (minimum %d characters)", *schema.MinLength), "error")
			}
			if schema.MaxLength != nil && len(str) > *schema.MaxLength {
				sv.addError(result, path, "maxLength", value, fmt.Sprintf("max %d chars", *schema.MaxLength),
					fmt.Sprintf("String too long (maximum %d characters)", *schema.MaxLength), "error")
			}
			if schema.Pattern != "" {
				// For simplicity, we'll skip regex validation for now
				// In production, you'd use regexp.MatchString here
			}
		}
	}

	// Number-specific validations
	if schema.Type == "number" || schema.Type == "integer" {
		if num, ok := value.(float64); ok {
			if schema.Minimum != nil && num < *schema.Minimum {
				sv.addError(result, path, "minimum", value, fmt.Sprintf("min %v", *schema.Minimum),
					fmt.Sprintf("Value too small (minimum %v)", *schema.Minimum), "error")
			}
			if schema.Maximum != nil && num > *schema.Maximum {
				sv.addError(result, path, "maximum", value, fmt.Sprintf("max %v", *schema.Maximum),
					fmt.Sprintf("Value too large (maximum %v)", *schema.Maximum), "error")
			}
		}
	}

	// Object validation
	if schema.Type == "object" && schema.Properties != nil {
		if objMap, ok := value.(map[string]interface{}); ok {
			// Check required properties
			for _, required := range schema.Required {
				if _, exists := objMap[required]; !exists {
					sv.addError(result, fmt.Sprintf("%s.%s", path, required), required, nil, "required",
						fmt.Sprintf("Required property '%s' is missing", required), "error")
				}
			}

			// Validate properties
			for propName, propValue := range objMap {
				propPath := fmt.Sprintf("%s.%s", path, propName)
				if propSchema, exists := schema.Properties[propName]; exists {
					sv.validateProperty(propValue, &propSchema, propPath, result)
				}
			}
		}
	}

	// Array validation
	if schema.Type == "array" && schema.Items != nil {
		if arr, ok := value.([]interface{}); ok {
			for i, item := range arr {
				itemPath := fmt.Sprintf("%s[%d]", path, i)
				sv.validateProperty(item, schema.Items, itemPath, result)
			}
		}
	}
}

// isValidType checks if a value matches the expected type
func (sv *SchemaValidator) isValidType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		if !ok {
			_, ok = value.(int)
		}
		return ok
	case "integer":
		_, ok := value.(int)
		if !ok {
			if f, ok := value.(float64); ok {
				return f == float64(int(f)) // Check if it's a whole number
			}
		}
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "null":
		return value == nil
	default:
		return true // Unknown types pass for now
	}
}

// isInEnum checks if a value is in the enum list
func (sv *SchemaValidator) isInEnum(value interface{}, enum []interface{}) bool {
	for _, enumValue := range enum {
		if value == enumValue {
			return true
		}
	}
	return false
}

// addError adds an error to the validation result
func (sv *SchemaValidator) addError(result *SchemaValidationResult, path, property string, value interface{}, expected, message, severity string) {
	error := SchemaValidationError{
		Path:     path,
		Property: property,
		Value:    value,
		Expected: expected,
		Message:  message,
		Severity: severity,
	}

	if severity == "error" {
		result.Errors = append(result.Errors, error)
	} else {
		result.Warnings = append(result.Warnings, error)
	}
}

// initializeBuiltinSchemas sets up built-in schemas for common configuration types
func (sv *SchemaValidator) initializeBuiltinSchemas() {
	// Basic matlas configuration schema
	sv.schemas["matlas-config"] = &ConfigSchema{
		Type:    "object",
		Title:   "MongoDB Atlas CLI Configuration",
		Version: "1.0",
		Properties: map[string]PropertySchema{
			"output": {
				Type:        "string",
				Description: "Output format for commands",
				Enum:        []interface{}{"json", "yaml", "table"},
			},
			"timeout": {
				Type:        "string",
				Description: "Default timeout for operations",
				Pattern:     `^\d+[smh]$`,
			},
			"projectId": {
				Type:        "string",
				Description: "Default Atlas project ID",
				MinLength:   &[]int{24}[0],
				MaxLength:   &[]int{24}[0],
			},
			"clusterName": {
				Type:        "string",
				Description: "Default cluster name",
				MinLength:   &[]int{1}[0],
				MaxLength:   &[]int{64}[0],
			},
			"apiKey": {
				Type:        "string",
				Description: "Atlas API key",
				MinLength:   &[]int{8}[0],
			},
			"publicKey": {
				Type:        "string",
				Description: "Atlas public key",
				MinLength:   &[]int{8}[0],
			},
		},
		Required: []string{"output"},
	}

	// Apply configuration schema
	sv.schemas["apply-config"] = &ConfigSchema{
		Type:    "object",
		Title:   "MongoDB Atlas Apply Configuration",
		Version: "1.0",
		Properties: map[string]PropertySchema{
			"apiVersion": {
				Type:        "string",
				Description: "API version",
				Enum:        []interface{}{"v1alpha1", "v1beta1", "v1"},
			},
			"kind": {
				Type:        "string",
				Description: "Resource kind",
				Enum:        []interface{}{"Project", "ApplyDocument"},
			},
			"metadata": {
				Type:        "object",
				Description: "Resource metadata",
				Properties: map[string]PropertySchema{
					"name": {
						Type:        "string",
						Description: "Resource name",
						MinLength:   &[]int{1}[0],
						MaxLength:   &[]int{64}[0],
					},
					"labels": {
						Type:        "object",
						Description: "Resource labels",
					},
					"annotations": {
						Type:        "object",
						Description: "Resource annotations",
					},
				},
				Required: []string{"name"},
			},
			"spec": {
				Type:        "object",
				Description: "Resource specification",
				Properties: map[string]PropertySchema{
					"name": {
						Type:        "string",
						Description: "Project name",
						MinLength:   &[]int{1}[0],
						MaxLength:   &[]int{64}[0],
					},
					"organizationId": {
						Type:        "string",
						Description: "Organization ID",
						MinLength:   &[]int{24}[0],
						MaxLength:   &[]int{24}[0],
					},
					"clusters": {
						Type:        "array",
						Description: "Cluster configurations",
						Items: &PropertySchema{
							Type: "object",
							Properties: map[string]PropertySchema{
								"metadata": {
									Type:        "object",
									Description: "Cluster metadata",
									Properties: map[string]PropertySchema{
										"name": {
											Type:        "string",
											Description: "Cluster name",
											MinLength:   &[]int{1}[0],
											MaxLength:   &[]int{64}[0],
										},
									},
									Required: []string{"name"},
								},
								"provider": {
									Type:        "string",
									Description: "Cloud provider",
									Enum:        []interface{}{"AWS", "GCP", "AZURE"},
								},
								"region": {
									Type:        "string",
									Description: "Cloud region",
									MinLength:   &[]int{1}[0],
								},
								"instanceSize": {
									Type:        "string",
									Description: "Instance size",
									Enum:        []interface{}{"M0", "M2", "M5", "M10", "M20", "M30", "M40", "M50", "M60", "M80", "M140", "M200", "M300", "M400", "M700"},
								},
							},
							Required: []string{"metadata", "provider", "region", "instanceSize"},
						},
					},
					"databaseUsers": {
						Type:        "array",
						Description: "Database user configurations",
						Items: &PropertySchema{
							Type: "object",
							Properties: map[string]PropertySchema{
								"username": {
									Type:        "string",
									Description: "Database username",
									MinLength:   &[]int{1}[0],
									MaxLength:   &[]int{1024}[0],
								},
								"roles": {
									Type:        "array",
									Description: "User roles",
									Items: &PropertySchema{
										Type: "object",
										Properties: map[string]PropertySchema{
											"roleName": {
												Type:        "string",
												Description: "Role name",
												MinLength:   &[]int{1}[0],
											},
											"databaseName": {
												Type:        "string",
												Description: "Database name",
												MinLength:   &[]int{1}[0],
											},
										},
										Required: []string{"roleName"},
									},
								},
							},
							Required: []string{"username", "roles"},
						},
					},
				},
				Required: []string{"name", "organizationId"},
			},
		},
		Required: []string{"apiVersion", "kind", "spec"},
	}
}
