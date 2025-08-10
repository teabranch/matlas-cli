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
	assertValidationTable(t, []validationCase[string]{
		{name: "Valid M10", in: "M10", wantErr: false, errCode: ""},
		{name: "Valid M20", in: "M20", wantErr: false, errCode: ""},
		{name: "Valid M700", in: "M700", wantErr: false, errCode: ""},
		{name: "Valid R40", in: "R40", wantErr: false, errCode: ""},
		{name: "Invalid size", in: "INVALID_SIZE", wantErr: true, errCode: "INVALID_INSTANCE_SIZE"},
		{name: "Empty size", in: "", wantErr: true, errCode: "INVALID_INSTANCE_SIZE"},
		{name: "Lowercase m10", in: "m10", wantErr: true, errCode: "INVALID_INSTANCE_SIZE"},
		{name: "Invalid M999", in: "M999", wantErr: true, errCode: "INVALID_INSTANCE_SIZE"},
	}, func(val string, res *ValidationResult) { validateInstanceSize(val, "test.instanceSize", res) })
}

func TestValidateMongoDBVersion(t *testing.T) {
	assertValidationTable(t, []validationCase[string]{
		{name: "Valid 4.4", in: "4.4", wantErr: false, errCode: ""},
		{name: "Valid 5.0", in: "5.0", wantErr: false, errCode: ""},
		{name: "Valid 6.0", in: "6.0", wantErr: false, errCode: ""},
		{name: "Valid 7.0", in: "7.0", wantErr: false, errCode: ""},
		{name: "Invalid 2.0", in: "2.0", wantErr: true, errCode: "INVALID_MONGODB_VERSION"},
		{name: "Invalid 3.6", in: "3.6", wantErr: true, errCode: "INVALID_MONGODB_VERSION"},
		{name: "Empty version", in: "", wantErr: true, errCode: "INVALID_MONGODB_VERSION"},
		{name: "Invalid version", in: "invalid", wantErr: true, errCode: "INVALID_MONGODB_VERSION"},
	}, func(val string, res *ValidationResult) { validateMongoDBVersion(val, "test.mongoDBVersion", res) })
}

func TestValidateResourceName(t *testing.T) {
	opts := DefaultValidatorOptions()
	assertValidationTable(t, []validationCase[string]{
		{name: "Valid name", in: "test-cluster", wantErr: false, errCode: ""},
		{name: "Valid with underscore", in: "test_cluster", wantErr: false, errCode: ""},
		{name: "Valid with numbers", in: "cluster123", wantErr: false, errCode: ""},
		{name: "Valid single char", in: "a", wantErr: false, errCode: ""},
		{name: "Empty name", in: "", wantErr: true, errCode: "EMPTY_NAME"},
		{name: "Name with space", in: "test cluster", wantErr: true, errCode: "INVALID_NAME"},
		{name: "Name with special char", in: "test@cluster", wantErr: true, errCode: "INVALID_NAME"},
		{name: "Name with uppercase", in: "Test-Cluster", wantErr: true, errCode: "INVALID_NAME"},
		{name: "Name starting with hyphen", in: "-test", wantErr: true, errCode: "INVALID_NAME"},
		{name: "Name ending with hyphen", in: "test-", wantErr: true, errCode: "INVALID_NAME"},
		{name: "Name with double dots", in: "test..cluster", wantErr: true, errCode: "INVALID_NAME"},
	}, func(val string, res *ValidationResult) { validateResourceName(val, "test.name", res, opts) })
}

func TestValidateDatabaseName(t *testing.T) {
	assertValidationTable(t, []validationCase[string]{
		{name: "Valid name", in: "testdb", wantErr: false, errCode: ""},
		{name: "Valid with underscore", in: "test_db", wantErr: false, errCode: ""},
		{name: "Valid with numbers", in: "test123", wantErr: false, errCode: ""},
		{name: "Valid mixed case", in: "myDatabase", wantErr: false, errCode: ""},
		{name: "Empty name", in: "", wantErr: true, errCode: "EMPTY_DATABASE_NAME"},
		{name: "Name with hyphen", in: "test-db", wantErr: true, errCode: "INVALID_DATABASE_NAME"},
		{name: "Name with space", in: "test db", wantErr: true, errCode: "INVALID_DATABASE_NAME"},
		{name: "Name with dot", in: "test.db", wantErr: true, errCode: "INVALID_DATABASE_NAME"},
		{name: "Name starting with number", in: "123test", wantErr: true, errCode: "INVALID_DATABASE_NAME"},
		{name: "Name with special char", in: "test$db", wantErr: true, errCode: "INVALID_DATABASE_NAME"},
	}, func(val string, res *ValidationResult) { validateDatabaseName(val, "test.databaseName", res) })
}

func TestValidateCollectionName(t *testing.T) {
	assertValidationTable(t, []validationCase[string]{
		{name: "Valid name", in: "users", wantErr: false, errCode: ""},
		{name: "Valid with underscore", in: "user_data", wantErr: false, errCode: ""},
		{name: "Valid camelCase", in: "userData", wantErr: false, errCode: ""},
		{name: "Valid with numbers", in: "users123", wantErr: false, errCode: ""},
		{name: "Empty name", in: "", wantErr: true, errCode: "EMPTY_COLLECTION_NAME"},
		{name: "Name with space", in: "user data", wantErr: true, errCode: "INVALID_COLLECTION_NAME"},
		{name: "Name starting with $", in: "$users", wantErr: true, errCode: "INVALID_COLLECTION_NAME"},
		{name: "System collection", in: "system.users", wantErr: true, errCode: "INVALID_COLLECTION_NAME"},
	}, func(val string, res *ValidationResult) { validateCollectionName(val, "test.collectionName", res) })
}

// Generic validation helpers to reduce duplication
type validationCase[T any] struct {
	name    string
	in      T
	wantErr bool
	errCode string
}

func assertValidationTable[T any](t *testing.T, cases []validationCase[T], validate func(val T, res *ValidationResult)) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{}
			validate(tt.in, result)

			hasErr := len(result.Errors) > 0
			if hasErr != tt.wantErr {
				t.Errorf("unexpected error presence: got %v, want %v", hasErr, tt.wantErr)
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
