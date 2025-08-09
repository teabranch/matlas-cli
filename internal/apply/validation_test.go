package apply

import (
	"testing"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestDefaultValidatorOptions(t *testing.T) {
	opts := DefaultValidatorOptions()

	if opts.StrictMode {
		t.Error("Default StrictMode should be false")
	}

	if len(opts.AllowedVersions) != 3 {
		t.Errorf("Expected 3 allowed versions, got %d", len(opts.AllowedVersions))
	}

	if opts.MaxNameLength != 64 {
		t.Errorf("Expected MaxNameLength to be 64, got %d", opts.MaxNameLength)
	}

	if opts.SkipQuotaCheck {
		t.Error("Default SkipQuotaCheck should be false")
	}

	// Check that all expected API versions are included
	expectedVersions := []types.APIVersion{
		types.APIVersionV1Alpha1,
		types.APIVersionV1Beta1,
		types.APIVersionV1,
	}

	for _, expected := range expectedVersions {
		found := false
		for _, allowed := range opts.AllowedVersions {
			if allowed == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected API version %s not found in allowed versions", expected)
		}
	}
}

func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Path:     "spec.clusters[0].name",
		Field:    "name",
		Value:    "invalid-name!",
		Message:  "invalid character in name",
		Code:     "INVALID_NAME",
		Severity: "error",
	}

	expected := "spec.clusters[0].name: invalid character in name"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}

func TestValidateProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
		errCode  string
	}{
		{"Valid AWS", "AWS", false, ""},
		{"Valid GCP", "GCP", false, ""},
		{"Valid AZURE", "AZURE", false, ""},
		{"Invalid provider", "INVALID_PROVIDER", true, "INVALID_PROVIDER"},
		{"Empty provider", "", true, "INVALID_PROVIDER"},
		{"Lowercase aws", "aws", true, "INVALID_PROVIDER"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateProvider(tt.provider, "test.provider", result)

			hasErr := len(result.Errors) > 0
			if hasErr != tt.wantErr {
				t.Errorf("validateProvider() error = %v, wantErr %v", hasErr, tt.wantErr)
			}

			if tt.wantErr && len(result.Errors) > 0 {
				if result.Errors[0].Code != tt.errCode {
					t.Errorf("Expected error code %s, got %s", tt.errCode, result.Errors[0].Code)
				}
			}
		})
	}
}

func TestValidateInstanceSize(t *testing.T) {
	tests := []struct {
		name    string
		size    string
		wantErr bool
		errCode string
	}{
		{"Valid M10", "M10", false, ""},
		{"Valid M20", "M20", false, ""},
		{"Valid M700", "M700", false, ""},
		{"Valid R40", "R40", false, ""},
		{"Invalid size", "INVALID_SIZE", true, "INVALID_INSTANCE_SIZE"},
		{"Empty size", "", true, "INVALID_INSTANCE_SIZE"},
		{"Lowercase m10", "m10", true, "INVALID_INSTANCE_SIZE"},
		{"Invalid M999", "M999", true, "INVALID_INSTANCE_SIZE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateInstanceSize(tt.size, "test.instanceSize", result)

			hasErr := len(result.Errors) > 0
			if hasErr != tt.wantErr {
				t.Errorf("validateInstanceSize() error = %v, wantErr %v", hasErr, tt.wantErr)
			}

			if tt.wantErr && len(result.Errors) > 0 {
				if result.Errors[0].Code != tt.errCode {
					t.Errorf("Expected error code %s, got %s", tt.errCode, result.Errors[0].Code)
				}
			}
		})
	}
}

func TestValidateMongoDBVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
		errCode string
	}{
		{"Valid 4.4", "4.4", false, ""},
		{"Valid 5.0", "5.0", false, ""},
		{"Valid 6.0", "6.0", false, ""},
		{"Valid 7.0", "7.0", false, ""},
		{"Invalid 2.0", "2.0", true, "INVALID_MONGODB_VERSION"},
		{"Invalid 3.6", "3.6", true, "INVALID_MONGODB_VERSION"},
		{"Empty version", "", true, "INVALID_MONGODB_VERSION"},
		{"Invalid version", "invalid", true, "INVALID_MONGODB_VERSION"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateMongoDBVersion(tt.version, "test.mongoDBVersion", result)

			hasErr := len(result.Errors) > 0
			if hasErr != tt.wantErr {
				t.Errorf("validateMongoDBVersion() error = %v, wantErr %v", hasErr, tt.wantErr)
			}

			if tt.wantErr && len(result.Errors) > 0 {
				if result.Errors[0].Code != tt.errCode {
					t.Errorf("Expected error code %s, got %s", tt.errCode, result.Errors[0].Code)
				}
			}
		})
	}
}

func TestValidateResourceName(t *testing.T) {
	opts := DefaultValidatorOptions()

	tests := []struct {
		name         string
		resourceName string
		wantErr      bool
		errCode      string
	}{
		{"Valid name", "test-cluster", false, ""},
		{"Valid with underscore", "test_cluster", false, ""},
		{"Valid with numbers", "cluster123", false, ""},
		{"Valid single char", "a", false, ""},
		{"Empty name", "", true, "EMPTY_NAME"},
		{"Name with space", "test cluster", true, "INVALID_NAME"},
		{"Name with special char", "test@cluster", true, "INVALID_NAME"},
		{"Name with uppercase", "Test-Cluster", true, "INVALID_NAME"},
		{"Name starting with hyphen", "-test", true, "INVALID_NAME"},
		{"Name ending with hyphen", "test-", true, "INVALID_NAME"},
		{"Name with double dots", "test..cluster", true, "INVALID_NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateResourceName(tt.resourceName, "test.name", result, opts)

			hasErr := len(result.Errors) > 0
			if hasErr != tt.wantErr {
				t.Errorf("validateResourceName() error = %v, wantErr %v", hasErr, tt.wantErr)
			}

			if tt.wantErr && len(result.Errors) > 0 {
				if result.Errors[0].Code != tt.errCode {
					t.Errorf("Expected error code %s, got %s", tt.errCode, result.Errors[0].Code)
				}
			}
		})
	}
}

func TestValidateDatabaseName(t *testing.T) {
	tests := []struct {
		name    string
		dbName  string
		wantErr bool
		errCode string
	}{
		{"Valid name", "testdb", false, ""},
		{"Valid with underscore", "test_db", false, ""},
		{"Valid with numbers", "test123", false, ""},
		{"Valid mixed case", "myDatabase", false, ""},
		{"Empty name", "", true, "EMPTY_DATABASE_NAME"},
		{"Name with hyphen", "test-db", true, "INVALID_DATABASE_NAME"},
		{"Name with space", "test db", true, "INVALID_DATABASE_NAME"},
		{"Name with dot", "test.db", true, "INVALID_DATABASE_NAME"},
		{"Name starting with number", "123test", true, "INVALID_DATABASE_NAME"},
		{"Name with special char", "test$db", true, "INVALID_DATABASE_NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateDatabaseName(tt.dbName, "test.databaseName", result)

			hasErr := len(result.Errors) > 0
			if hasErr != tt.wantErr {
				t.Errorf("validateDatabaseName() error = %v, wantErr %v", hasErr, tt.wantErr)
			}

			if tt.wantErr && len(result.Errors) > 0 {
				if result.Errors[0].Code != tt.errCode {
					t.Errorf("Expected error code %s, got %s", tt.errCode, result.Errors[0].Code)
				}
			}
		})
	}
}

func TestValidateCollectionName(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
		wantErr        bool
		errCode        string
	}{
		{"Valid name", "users", false, ""},
		{"Valid with underscore", "user_data", false, ""},
		{"Valid camelCase", "userData", false, ""},
		{"Valid with numbers", "users123", false, ""},
		{"Empty name", "", true, "EMPTY_COLLECTION_NAME"},
		{"Name with space", "user data", true, "INVALID_COLLECTION_NAME"},
		{"Name starting with $", "$users", true, "INVALID_COLLECTION_NAME"},
		{"System collection", "system.users", true, "INVALID_COLLECTION_NAME"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validateCollectionName(tt.collectionName, "test.collectionName", result)

			hasErr := len(result.Errors) > 0
			if hasErr != tt.wantErr {
				t.Errorf("validateCollectionName() error = %v, wantErr %v", hasErr, tt.wantErr)
			}

			if tt.wantErr && len(result.Errors) > 0 {
				if result.Errors[0].Code != tt.errCode {
					t.Errorf("Expected error code %s, got %s", tt.errCode, result.Errors[0].Code)
				}
			}
		})
	}
}

func TestValidationResult_Operations(t *testing.T) {
	result := &ValidationResult{Valid: true}

	// Add an error
	err := ValidationError{
		Path:     "test.path",
		Field:    "testField",
		Value:    "testValue",
		Message:  "Test error message",
		Code:     "TEST_ERROR",
		Severity: "error",
	}

	result.Errors = append(result.Errors, err)
	result.Valid = false

	if result.Valid {
		t.Error("Expected Valid to be false after adding error")
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	if result.Errors[0].Code != "TEST_ERROR" {
		t.Errorf("Expected error code TEST_ERROR, got %s", result.Errors[0].Code)
	}

	// Add a warning
	warning := ValidationError{
		Path:     "test.path",
		Field:    "testField",
		Value:    "testValue",
		Message:  "Test warning message",
		Code:     "TEST_WARNING",
		Severity: "warning",
	}

	result.Warnings = append(result.Warnings, warning)

	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	}

	if result.Warnings[0].Code != "TEST_WARNING" {
		t.Errorf("Expected warning code TEST_WARNING, got %s", result.Warnings[0].Code)
	}
}
