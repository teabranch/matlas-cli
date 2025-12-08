package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312010/admin"
)

// EncryptionService provides CRUD operations for Atlas Encryption at Rest.
// This service manages AWS KMS, Azure Key Vault, and GCP KMS encryption configurations.
type EncryptionService struct {
	client *atlasclient.Client
}

// NewEncryptionService creates a new EncryptionService instance.
func NewEncryptionService(client *atlasclient.Client) *EncryptionService {
	return &EncryptionService{client: client}
}

// GetEncryptionAtRest returns the current encryption at rest configuration for a project.
func (s *EncryptionService) GetEncryptionAtRest(ctx context.Context, projectID string) (*admin.EncryptionAtRest, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}

	var enc *admin.EncryptionAtRest
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.EncryptionAtRestUsingCustomerKeyManagementApi.GetEncryptionAtRest(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		enc = resp
		return nil
	})
	return enc, err
}

// UpdateEncryptionAtRest updates the encryption at rest configuration for a project.
func (s *EncryptionService) UpdateEncryptionAtRest(ctx context.Context, projectID string, encryption *admin.EncryptionAtRest) (*admin.EncryptionAtRest, error) {
	if projectID == "" || encryption == nil {
		return nil, fmt.Errorf("projectID and encryption configuration are required")
	}

	// Validate the encryption configuration
	if err := s.validateEncryptionConfig(encryption); err != nil {
		return nil, fmt.Errorf("encryption configuration validation failed: %w", err)
	}

	var updated *admin.EncryptionAtRest
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.EncryptionAtRestUsingCustomerKeyManagementApi.UpdateEncryptionAtRest(ctx, projectID, encryption).Execute()
		if err != nil {
			return err
		}
		updated = resp
		return nil
	})
	return updated, err
}

// EnableAWSKMSEncryption enables AWS KMS encryption at rest for a project.
func (s *EncryptionService) EnableAWSKMSEncryption(ctx context.Context, projectID string, config *AWSKMSConfiguration) (*admin.EncryptionAtRest, error) {
	if projectID == "" || config == nil {
		return nil, fmt.Errorf("projectID and AWS KMS configuration are required")
	}

	// Validate AWS KMS configuration
	if err := s.validateAWSKMSConfig(config); err != nil {
		return nil, fmt.Errorf("AWS KMS configuration validation failed: %w", err)
	}

	// Build encryption at rest configuration
	encryption := &admin.EncryptionAtRest{
		AwsKms: &admin.AWSKMSConfiguration{
			Enabled:             admin.PtrBool(true),
			AccessKeyID:         admin.PtrString(config.AccessKeyID),
			SecretAccessKey:     admin.PtrString(config.SecretAccessKey),
			CustomerMasterKeyID: admin.PtrString(config.CustomerMasterKeyID),
			Region:              admin.PtrString(config.Region),
		},
	}

	// If RoleID is provided, use IAM role instead of access keys
	if config.RoleID != "" {
		encryption.AwsKms.RoleId = admin.PtrString(config.RoleID)
		// Clear access keys when using IAM role
		encryption.AwsKms.AccessKeyID = nil
		encryption.AwsKms.SecretAccessKey = nil
	}

	return s.UpdateEncryptionAtRest(ctx, projectID, encryption)
}

// DisableEncryption disables encryption at rest for a project.
func (s *EncryptionService) DisableEncryption(ctx context.Context, projectID string) (*admin.EncryptionAtRest, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}

	// Get current encryption configuration
	current, err := s.GetEncryptionAtRest(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current encryption config: %w", err)
	}

	// Disable all encryption providers
	encryption := &admin.EncryptionAtRest{}
	if current.AwsKms != nil {
		encryption.AwsKms = &admin.AWSKMSConfiguration{
			Enabled: admin.PtrBool(false),
		}
	}
	if current.AzureKeyVault != nil {
		encryption.AzureKeyVault = &admin.AzureKeyVault{
			Enabled: admin.PtrBool(false),
		}
	}
	if current.GoogleCloudKms != nil {
		encryption.GoogleCloudKms = &admin.GoogleCloudKMS{
			Enabled: admin.PtrBool(false),
		}
	}

	return s.UpdateEncryptionAtRest(ctx, projectID, encryption)
}

// GetEncryptionStatus returns a simplified status of encryption configuration.
func (s *EncryptionService) GetEncryptionStatus(ctx context.Context, projectID string) (*EncryptionStatus, error) {
	encryption, err := s.GetEncryptionAtRest(ctx, projectID)
	if err != nil {
		return nil, err
	}

	status := &EncryptionStatus{
		ProjectID: projectID,
		AWSKMSEnabled: encryption.AwsKms != nil &&
			encryption.AwsKms.Enabled != nil &&
			*encryption.AwsKms.Enabled,
		AzureKeyVaultEnabled: encryption.AzureKeyVault != nil &&
			encryption.AzureKeyVault.Enabled != nil &&
			*encryption.AzureKeyVault.Enabled,
		GoogleCloudKMSEnabled: encryption.GoogleCloudKms != nil &&
			encryption.GoogleCloudKms.Enabled != nil &&
			*encryption.GoogleCloudKms.Enabled,
	}

	// Extract key information
	if status.AWSKMSEnabled && encryption.AwsKms != nil {
		if encryption.AwsKms.CustomerMasterKeyID != nil {
			status.AWSKMSKeyID = *encryption.AwsKms.CustomerMasterKeyID
		}
		if encryption.AwsKms.Region != nil {
			status.AWSKMSRegion = *encryption.AwsKms.Region
		}
	}

	return status, nil
}

// validateEncryptionConfig validates the encryption at rest configuration.
func (s *EncryptionService) validateEncryptionConfig(encryption *admin.EncryptionAtRest) error {
	if encryption == nil {
		return fmt.Errorf("encryption configuration is required")
	}

	// At least one encryption provider must be specified
	hasProvider := false

	if encryption.AwsKms != nil {
		hasProvider = true
		if err := s.validateAWSKMSEncryption(encryption.AwsKms); err != nil {
			return fmt.Errorf("aws KMS validation failed: %w", err)
		}
	}

	if encryption.AzureKeyVault != nil {
		hasProvider = true
		if err := s.validateAzureKeyVaultEncryption(encryption.AzureKeyVault); err != nil {
			return fmt.Errorf("azure key vault validation failed: %w", err)
		}
	}

	if encryption.GoogleCloudKms != nil {
		hasProvider = true
		if err := s.validateGoogleCloudKMSEncryption(encryption.GoogleCloudKms); err != nil {
			return fmt.Errorf("google cloud KMS validation failed: %w", err)
		}
	}

	if !hasProvider {
		return fmt.Errorf("at least one encryption provider must be specified")
	}

	return nil
}

// validateAWSKMSEncryption validates AWS KMS specific configuration.
func (s *EncryptionService) validateAWSKMSEncryption(kms *admin.AWSKMSConfiguration) error {
	if kms.Enabled == nil {
		return fmt.Errorf("enabled field is required")
	}

	if !*kms.Enabled {
		return nil // No further validation needed if disabled
	}

	if kms.CustomerMasterKeyID == nil || *kms.CustomerMasterKeyID == "" {
		return fmt.Errorf("customerMasterKeyId is required when AWS KMS is enabled")
	}

	if kms.Region == nil || *kms.Region == "" {
		return fmt.Errorf("region is required when AWS KMS is enabled")
	}

	// Validate authentication method
	hasAccessKeys := kms.AccessKeyID != nil && *kms.AccessKeyID != "" &&
		kms.SecretAccessKey != nil && *kms.SecretAccessKey != ""
	hasRole := kms.RoleId != nil && *kms.RoleId != ""

	if !hasAccessKeys && !hasRole {
		return fmt.Errorf("either access keys (accessKeyId and secretAccessKey) or roleId must be provided")
	}

	if hasAccessKeys && hasRole {
		return fmt.Errorf("cannot specify both access keys and roleId - choose one authentication method")
	}

	return nil
}

// validateAzureKeyVaultEncryption validates Azure Key Vault specific configuration.
func (s *EncryptionService) validateAzureKeyVaultEncryption(vault *admin.AzureKeyVault) error {
	if vault.Enabled == nil {
		return fmt.Errorf("enabled field is required")
	}

	// Basic validation - more specific validation can be added when needed
	return nil
}

// validateGoogleCloudKMSEncryption validates Google Cloud KMS specific configuration.
func (s *EncryptionService) validateGoogleCloudKMSEncryption(kms *admin.GoogleCloudKMS) error {
	if kms.Enabled == nil {
		return fmt.Errorf("enabled field is required")
	}

	// Basic validation - more specific validation can be added when needed
	return nil
}

// validateAWSKMSConfig validates the AWS KMS configuration input.
func (s *EncryptionService) validateAWSKMSConfig(config *AWSKMSConfiguration) error {
	if config.CustomerMasterKeyID == "" {
		return fmt.Errorf("customerMasterKeyId is required")
	}

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	// Validate authentication method
	hasAccessKeys := config.AccessKeyID != "" && config.SecretAccessKey != ""
	hasRole := config.RoleID != ""

	if !hasAccessKeys && !hasRole {
		return fmt.Errorf("either access keys (accessKeyId and secretAccessKey) or roleId must be provided")
	}

	if hasAccessKeys && hasRole {
		return fmt.Errorf("cannot specify both access keys and roleId - choose one authentication method")
	}

	return nil
}

// AWSKMSConfiguration represents the AWS KMS configuration for encryption.
type AWSKMSConfiguration struct {
	AccessKeyID         string
	SecretAccessKey     string
	CustomerMasterKeyID string
	Region              string
	RoleID              string
}

// EncryptionStatus represents the current encryption status of a project.
type EncryptionStatus struct {
	ProjectID             string
	AWSKMSEnabled         bool
	AWSKMSKeyID           string
	AWSKMSRegion          string
	AzureKeyVaultEnabled  bool
	GoogleCloudKMSEnabled bool
}
