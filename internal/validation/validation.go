package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Common validation patterns and helpers used across all Atlas services.

var (
	// objectIDRegex matches MongoDB 24-character hex ObjectID format
	objectIDRegex = regexp.MustCompile(`^[0-9a-fA-F]{24}$`)

	// usernameRegex allows alphanumeric, underscore, dash, and dot
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

	// clusterNameRegex matches Atlas cluster naming rules
	clusterNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

	// emailRegex for basic email validation
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// connectionStringRegex for MongoDB connection string validation
	connectionStringRegex = regexp.MustCompile(`^mongodb(\+srv)?://`)
)

// ValidationSeverity represents the severity of a validation issue
type ValidationSeverity int

const (
	SeverityError ValidationSeverity = iota
	SeverityWarning
	SeverityInfo
)

// ValidationIssue represents a structured validation problem
type ValidationIssue struct {
	Path        string             `json:"path"`
	Field       string             `json:"field"`
	Value       string             `json:"value"`
	Message     string             `json:"message"`
	Code        string             `json:"code"`
	Severity    ValidationSeverity `json:"severity"`
	Suggestions []string           `json:"suggestions,omitempty"`
}

// Error implements the error interface
func (vi ValidationIssue) Error() string {
	return fmt.Sprintf("%s: %s", vi.Path, vi.Message)
}

// ValidateObjectID validates a MongoDB ObjectID (24-character hex string).
func ValidateObjectID(id, fieldName string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	if !objectIDRegex.MatchString(id) {
		return fmt.Errorf("%s must be a 24-character hexadecimal string", fieldName)
	}
	return nil
}

// ValidateProjectID validates an Atlas project ID.
func ValidateProjectID(projectID string) error {
	return ValidateObjectID(projectID, "projectID")
}

// ValidateOrganizationID validates an Atlas organization ID.
func ValidateOrganizationID(orgID string) error {
	return ValidateObjectID(orgID, "organizationID")
}

// ValidateUsername validates a database username.
func ValidateUsername(username string) error {
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if len(username) > 1024 {
		return fmt.Errorf("username cannot exceed 1024 characters")
	}
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username contains invalid characters (allowed: a-z, A-Z, 0-9, ., _, -)")
	}
	return nil
}

// ValidateClusterName validates an Atlas cluster name.
func ValidateClusterName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}
	if len(name) < 1 || len(name) > 64 {
		return fmt.Errorf("cluster name must be 1-64 characters")
	}
	if !clusterNameRegex.MatchString(name) {
		return fmt.Errorf("cluster name must start/end with alphanumeric and contain only letters, numbers, and hyphens")
	}
	return nil
}

// ValidateRequired validates that a field is non-empty.
func ValidateRequired(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateMaxLength validates string length constraints.
func ValidateMaxLength(value, fieldName string, maxLen int) error {
	if len(value) > maxLen {
		return fmt.Errorf("%s cannot exceed %d characters", fieldName, maxLen)
	}
	return nil
}

// ValidateStringSlice validates that a slice contains valid strings.
func ValidateStringSlice(slice []string, fieldName string, allowEmpty bool) error {
	if !allowEmpty && len(slice) == 0 {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	for i, item := range slice {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("%s[%d] cannot be empty", fieldName, i)
		}
	}
	return nil
}

// ValidateEmail validates an email address format
func ValidateEmail(email, fieldName string) error {
	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("%s must be a valid email address", fieldName)
	}
	return nil
}

// ValidateIPAddress validates an IP address (IPv4 or IPv6)
func ValidateIPAddress(ip, fieldName string) error {
	if strings.TrimSpace(ip) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("%s must be a valid IP address", fieldName)
	}
	return nil
}

// ValidateCIDR validates CIDR notation
func ValidateCIDR(cidr, fieldName string) error {
	if strings.TrimSpace(cidr) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("%s must be valid CIDR notation (e.g., 192.168.1.0/24)", fieldName)
	}
	return nil
}

// ValidateConnectionString validates a MongoDB connection string format
func ValidateConnectionString(connStr, fieldName string) error {
	if strings.TrimSpace(connStr) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	if !connectionStringRegex.MatchString(connStr) {
		return fmt.Errorf("%s must be a valid MongoDB connection string (mongodb:// or mongodb+srv://)", fieldName)
	}
	return nil
}

// ValidateAtlasInstanceSize validates Atlas cluster instance sizes
func ValidateAtlasInstanceSize(size, fieldName string) error {
	validSizes := map[string]bool{
		"M0": true, "M2": true, "M5": true, "M10": true, "M20": true, "M30": true,
		"M40": true, "M50": true, "M60": true, "M80": true, "M140": true,
		"M200": true, "M300": true, "M400": true, "M700": true,
		"R40": true, "R50": true, "R60": true, "R80": true, "R200": true,
		"R300": true, "R400": true, "R700": true,
	}

	if strings.TrimSpace(size) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	if !validSizes[size] {
		return fmt.Errorf("%s must be a valid Atlas instance size (e.g., M0, M10, M30, R40)", fieldName)
	}
	return nil
}

// ValidateAtlasProvider validates Atlas cloud providers
func ValidateAtlasProvider(provider, fieldName string) error {
	validProviders := map[string]bool{
		"AWS":   true,
		"GCP":   true,
		"AZURE": true,
	}

	if strings.TrimSpace(provider) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	if !validProviders[strings.ToUpper(provider)] {
		return fmt.Errorf("%s must be one of: AWS, GCP, AZURE", fieldName)
	}
	return nil
}

// ValidateAtlasRegion validates region names for a given provider
func ValidateAtlasRegion(region, provider, fieldName string) error {
	if strings.TrimSpace(region) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	providerUpper := strings.ToUpper(strings.TrimSpace(provider))

	// Authoritative region sets (representative, not exhaustive)
	awsRegions := map[string]bool{
		// Americas
		"us-east-1": true, "us-east-2": true, "us-west-1": true, "us-west-2": true,
		"ca-central-1": true, "sa-east-1": true,
		// Europe
		"eu-west-1": true, "eu-west-2": true, "eu-west-3": true, "eu-central-1": true, "eu-north-1": true, "eu-south-1": true,
		// APAC/ME/Africa
		"ap-south-1": true, "ap-south-2": true, "ap-southeast-1": true, "ap-southeast-2": true, "ap-southeast-3": true,
		"ap-northeast-1": true, "ap-northeast-2": true, "ap-northeast-3": true,
		"me-south-1": true, "me-central-1": true, "af-south-1": true,
	}

	gcpRegions := map[string]bool{
		"us-central1": true, "us-east1": true, "us-east4": true, "us-west1": true, "us-west2": true,
		"europe-west1": true, "europe-west2": true, "europe-west3": true, "europe-west4": true, "europe-west6": true, "europe-north1": true,
		"asia-east1": true, "asia-east2": true, "asia-south1": true, "asia-south2": true,
		"asia-southeast1": true, "asia-southeast2": true, "asia-northeast1": true, "asia-northeast2": true, "asia-northeast3": true,
		"australia-southeast1": true, "australia-southeast2": true, "southamerica-east1": true, "me-central1": true,
	}

	azureRegions := map[string]bool{
		// Americas
		"eastus": true, "eastus2": true, "westus": true, "westus2": true, "westus3": true,
		"centralus": true, "northcentralus": true, "southcentralus": true, "brazilsouth": true,
		"canadacentral": true, "canadaeast": true,
		// Europe
		"northeurope": true, "westeurope": true, "uksouth": true, "ukwest": true, "francecentral": true, "germanywestcentral": true,
		"switzerlandnorth": true, "norwayeast": true, "swedencentral": true, "italynorth": true, "polandcentral": true,
		// APAC/MEA
		"eastasia": true, "southeastasia": true, "australiaeast": true, "australiasoutheast": true,
		"japaneast": true, "japanwest": true, "koreacentral": true, "southafricanorth": true, "uaenorth": true,
	}

	switch providerUpper {
	case "AWS":
		// Normalize Atlas style (US_EAST_1) to provider style (us-east-1)
		normalized := strings.ToLower(strings.ReplaceAll(region, "_", "-"))
		if !awsRegions[normalized] {
			return fmt.Errorf("%s must be a valid AWS region (e.g., US_EAST_1 or us-east-1)", fieldName)
		}
	case "GCP":
		normalized := strings.ToLower(region)
		if !gcpRegions[normalized] {
			return fmt.Errorf("%s must be a valid GCP region (e.g., us-central1, europe-west1)", fieldName)
		}
	case "AZURE":
		normalized := strings.ToLower(region)
		if !azureRegions[normalized] {
			return fmt.Errorf("%s must be a valid Azure region (e.g., eastus, westeurope)", fieldName)
		}
	default:
		// Unknown provider - treat as error for stricter validation
		return fmt.Errorf("%s has unknown provider '%s' for region validation", fieldName, provider)
	}

	return nil
}

// ValidateEnum validates that a value is one of the allowed options
func ValidateEnum(value, fieldName string, allowedValues []string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}

	return fmt.Errorf("%s must be one of: %s", fieldName, strings.Join(allowedValues, ", "))
}

// ValidateRange validates that an integer is within a specified range
func ValidateRange(value int, fieldName string, min, max int) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %d and %d (got %d)", fieldName, min, max, value)
	}
	return nil
}

// ValidateAtlasResourceTags validates Atlas resource tags according to Atlas requirements
func ValidateAtlasResourceTags(tags map[string]string, fieldName string) error {
	if len(tags) == 0 {
		return nil // Tags are optional
	}

	// Atlas allows maximum 50 tags per resource
	if len(tags) > 50 {
		return fmt.Errorf("%s cannot have more than 50 tags (got %d)", fieldName, len(tags))
	}

	// Allowed characters for Atlas tags: letters, numbers, spaces, semicolons, at symbols, underscores, dashes, periods, plus signs
	validTagCharRegex := regexp.MustCompile(`^[a-zA-Z0-9\s;@_\-.+]*$`)

	for key, value := range tags {
		// Validate key
		if key == "" {
			return fmt.Errorf("%s key cannot be empty", fieldName)
		}
		if len(key) > 255 {
			return fmt.Errorf("%s key '%s' cannot exceed 255 characters (got %d)", fieldName, key, len(key))
		}
		if !validTagCharRegex.MatchString(key) {
			return fmt.Errorf("%s key '%s' contains invalid characters (allowed: letters, numbers, spaces, ;@_-.+)", fieldName, key)
		}

		// Validate value
		if len(value) > 255 {
			return fmt.Errorf("%s value for key '%s' cannot exceed 255 characters (got %d)", fieldName, key, len(value))
		}
		if !validTagCharRegex.MatchString(value) {
			return fmt.Errorf("%s value '%s' for key '%s' contains invalid characters (allowed: letters, numbers, spaces, ;@_-.+)", fieldName, value, key)
		}
	}

	return nil
}
