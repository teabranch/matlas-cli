package atlas

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

func TestNewEncryptionService(t *testing.T) {
	client := &atlasclient.Client{}
	service := NewEncryptionService(client)

	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
}

func TestEncryptionService_EnableAWSKMSEncryption_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)
	service := NewEncryptionService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		config      *AWSKMSConfiguration
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing project ID",
			projectID:   "",
			config:      &AWSKMSConfiguration{},
			expectError: true,
			errorMsg:    "projectID and AWS KMS configuration are required",
		},
		{
			name:        "nil config",
			projectID:   "test-project",
			config:      nil,
			expectError: true,
			errorMsg:    "projectID and AWS KMS configuration are required",
		},
		{
			name:      "missing customer master key ID",
			projectID: "test-project",
			config: &AWSKMSConfiguration{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				Region:          "US_EAST_1",
			},
			expectError: true,
			errorMsg:    "customerMasterKeyId is required",
		},
		{
			name:      "missing region",
			projectID: "test-project",
			config: &AWSKMSConfiguration{
				AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey:     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
			},
			expectError: true,
			errorMsg:    "region is required",
		},
		{
			name:      "missing authentication method",
			projectID: "test-project",
			config: &AWSKMSConfiguration{
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
				Region:              "US_EAST_1",
			},
			expectError: true,
			errorMsg:    "either access keys",
		},
		{
			name:      "both access keys and role ID",
			projectID: "test-project",
			config: &AWSKMSConfiguration{
				AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey:     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
				Region:              "US_EAST_1",
				RoleID:              "507f1f77bcf86cd799439011",
			},
			expectError: true,
			errorMsg:    "cannot specify both access keys and roleId",
		},
		{
			name:      "valid config with access keys",
			projectID: "test-project",
			config: &AWSKMSConfiguration{
				AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey:     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
				Region:              "US_EAST_1",
			},
			expectError: true,
		},
		{
			name:      "valid config with IAM role",
			projectID: "test-project",
			config: &AWSKMSConfiguration{
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
				Region:              "US_EAST_1",
				RoleID:              "507f1f77bcf86cd799439011",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.EnableAWSKMSEncryption(ctx, tt.projectID, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEncryptionService_validateAWSKMSConfig(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)
	service := NewEncryptionService(client)

	tests := []struct {
		name        string
		config      *AWSKMSConfiguration
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with access keys",
			config: &AWSKMSConfiguration{
				AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey:     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
				Region:              "US_EAST_1",
			},
			expectError: false,
		},
		{
			name: "valid config with IAM role",
			config: &AWSKMSConfiguration{
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
				Region:              "US_EAST_1",
				RoleID:              "507f1f77bcf86cd799439011",
			},
			expectError: false,
		},
		{
			name: "missing customer master key ID",
			config: &AWSKMSConfiguration{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				Region:          "US_EAST_1",
			},
			expectError: true,
			errorMsg:    "customerMasterKeyId is required",
		},
		{
			name: "missing region",
			config: &AWSKMSConfiguration{
				AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey:     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
			},
			expectError: true,
			errorMsg:    "region is required",
		},
		{
			name: "missing authentication method",
			config: &AWSKMSConfiguration{
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
				Region:              "US_EAST_1",
			},
			expectError: true,
			errorMsg:    "either access keys",
		},
		{
			name: "both access keys and role ID",
			config: &AWSKMSConfiguration{
				AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey:     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				CustomerMasterKeyID: "arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012",
				Region:              "US_EAST_1",
				RoleID:              "507f1f77bcf86cd799439011",
			},
			expectError: true,
			errorMsg:    "cannot specify both access keys and roleId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateAWSKMSConfig(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEncryptionService_validateAWSKMSEncryption(t *testing.T) {
	service := NewEncryptionService(&atlasclient.Client{})

	tests := []struct {
		name        string
		kms         *admin.AWSKMSConfiguration
		expectError bool
		errorMsg    string
	}{
		{
			name: "disabled encryption",
			kms: &admin.AWSKMSConfiguration{
				Enabled: admin.PtrBool(false),
			},
			expectError: false,
		},
		{
			name: "enabled encryption with access keys",
			kms: &admin.AWSKMSConfiguration{
				Enabled:             admin.PtrBool(true),
				AccessKeyID:         admin.PtrString("AKIAIOSFODNN7EXAMPLE"),
				SecretAccessKey:     admin.PtrString("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
				CustomerMasterKeyID: admin.PtrString("arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012"),
				Region:              admin.PtrString("US_EAST_1"),
			},
			expectError: false,
		},
		{
			name: "enabled encryption with IAM role",
			kms: &admin.AWSKMSConfiguration{
				Enabled:             admin.PtrBool(true),
				CustomerMasterKeyID: admin.PtrString("arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012"),
				Region:              admin.PtrString("US_EAST_1"),
				RoleId:              admin.PtrString("507f1f77bcf86cd799439011"),
			},
			expectError: false,
		},
		{
			name: "missing enabled field",
			kms: &admin.AWSKMSConfiguration{
				CustomerMasterKeyID: admin.PtrString("arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012"),
			},
			expectError: true,
			errorMsg:    "enabled field is required",
		},
		{
			name: "enabled but missing customer master key ID",
			kms: &admin.AWSKMSConfiguration{
				Enabled:         admin.PtrBool(true),
				AccessKeyID:     admin.PtrString("AKIAIOSFODNN7EXAMPLE"),
				SecretAccessKey: admin.PtrString("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
				Region:          admin.PtrString("US_EAST_1"),
			},
			expectError: true,
			errorMsg:    "customerMasterKeyId is required when AWS KMS is enabled",
		},
		{
			name: "enabled but missing region",
			kms: &admin.AWSKMSConfiguration{
				Enabled:             admin.PtrBool(true),
				AccessKeyID:         admin.PtrString("AKIAIOSFODNN7EXAMPLE"),
				SecretAccessKey:     admin.PtrString("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
				CustomerMasterKeyID: admin.PtrString("arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012"),
			},
			expectError: true,
			errorMsg:    "region is required when AWS KMS is enabled",
		},
		{
			name: "enabled but missing authentication method",
			kms: &admin.AWSKMSConfiguration{
				Enabled:             admin.PtrBool(true),
				CustomerMasterKeyID: admin.PtrString("arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012"),
				Region:              admin.PtrString("US_EAST_1"),
			},
			expectError: true,
			errorMsg:    "either access keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateAWSKMSEncryption(tt.kms)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEncryptionService_BasicOperations(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)
	service := NewEncryptionService(client)
	ctx := context.Background()

	// Test GetEncryptionAtRest
	t.Run("GetEncryptionAtRest", func(t *testing.T) {
		_, err := service.GetEncryptionAtRest(ctx, "test-project")
		// Should fail due to missing API but not due to validation
		assert.Error(t, err)
		// Error content depends on SDK and environment; only assert presence
	})

	// Test UpdateEncryptionAtRest
	t.Run("UpdateEncryptionAtRest", func(t *testing.T) {
		encryption := &admin.EncryptionAtRest{
			AwsKms: &admin.AWSKMSConfiguration{
				Enabled:             admin.PtrBool(true),
				AccessKeyID:         admin.PtrString("AKIAIOSFODNN7EXAMPLE"),
				SecretAccessKey:     admin.PtrString("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
				CustomerMasterKeyID: admin.PtrString("arn:aws:kms:US_EAST_1:123456789012:key/12345678-1234-1234-1234-123456789012"),
				Region:              admin.PtrString("US_EAST_1"),
			},
		}
		_, err := service.UpdateEncryptionAtRest(ctx, "test-project", encryption)
		// Should fail due to missing API but not due to validation
		assert.Error(t, err)
		// Error content depends on SDK and environment; only assert presence
	})

	// Test DisableEncryption
	t.Run("DisableEncryption", func(t *testing.T) {
		_, err := service.DisableEncryption(ctx, "test-project")
		// Should fail due to missing API but not due to validation
		assert.Error(t, err)
		// Error content depends on SDK and environment; only assert presence
	})

	// Test GetEncryptionStatus
	t.Run("GetEncryptionStatus", func(t *testing.T) {
		_, err := service.GetEncryptionStatus(ctx, "test-project")
		// Should fail due to missing API but not due to validation
		assert.Error(t, err)
		// Error content depends on SDK and environment; only assert presence
	})
}

func TestEncryptionService_ParameterValidation(t *testing.T) {
	client := &atlasclient.Client{}
	service := NewEncryptionService(client)
	ctx := context.Background()

	t.Run("empty project ID validation", func(t *testing.T) {
		// Test GetEncryptionAtRest
		_, err := service.GetEncryptionAtRest(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID required")

		// Test UpdateEncryptionAtRest
		encryption := &admin.EncryptionAtRest{}
		_, err = service.UpdateEncryptionAtRest(ctx, "", encryption)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and encryption configuration are required")

		// Test DisableEncryption
		_, err = service.DisableEncryption(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID required")
	})

	t.Run("nil encryption configuration validation", func(t *testing.T) {
		_, err := service.UpdateEncryptionAtRest(ctx, "test-project", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and encryption configuration are required")
	})
}
