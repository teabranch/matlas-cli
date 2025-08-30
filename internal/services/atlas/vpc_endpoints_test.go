package atlas

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

func TestNewVPCEndpointsService(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
}

func TestVPCEndpointsService_ListPrivateEndpointServices_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	tests := []struct {
		name          string
		projectID     string
		cloudProvider string
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "missing project ID",
			projectID:     "",
			cloudProvider: "AWS",
			expectError:   true,
			errorMsg:      "projectID and cloudProvider are required",
		},
		{
			name:          "missing cloud provider",
			projectID:     "test-project",
			cloudProvider: "",
			expectError:   true,
			errorMsg:      "projectID and cloudProvider are required",
		},
		{
			name:          "valid parameters",
			projectID:     "test-project",
			cloudProvider: "AWS",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ListPrivateEndpointServices(ctx, tt.projectID, tt.cloudProvider)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// Note: This may fail with API errors due to lack of real credentials
				// but the validation should pass
				if err != nil {
					// Check if it's a validation error or API error
					assert.NotContains(t, err.Error(), "projectID and cloudProvider are required")
				}
			}
		})
	}
}

func TestVPCEndpointsService_ListAllPrivateEndpointServices_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing project ID",
			projectID:   "",
			expectError: true,
			errorMsg:    "projectID required",
		},
		{
			name:        "valid project ID",
			projectID:   "test-project",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ListAllPrivateEndpointServices(ctx, tt.projectID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// Note: This may fail with API errors due to lack of real credentials
				// but the validation should pass
				if err != nil {
					// Check if it's a validation error or API error
					assert.NotContains(t, err.Error(), "projectID required")
				}
			}
		})
	}
}

func TestVPCEndpointsService_GetPrivateEndpointService_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	tests := []struct {
		name              string
		projectID         string
		cloudProvider     string
		endpointServiceID string
		expectError       bool
		errorMsg          string
	}{
		{
			name:              "missing project ID",
			projectID:         "",
			cloudProvider:     "AWS",
			endpointServiceID: "test-service",
			expectError:       true,
			errorMsg:          "projectID, cloudProvider, and endpointServiceID are required",
		},
		{
			name:              "missing cloud provider",
			projectID:         "test-project",
			cloudProvider:     "",
			endpointServiceID: "test-service",
			expectError:       true,
			errorMsg:          "projectID, cloudProvider, and endpointServiceID are required",
		},
		{
			name:              "missing endpoint service ID",
			projectID:         "test-project",
			cloudProvider:     "AWS",
			endpointServiceID: "",
			expectError:       true,
			errorMsg:          "projectID, cloudProvider, and endpointServiceID are required",
		},
		{
			name:              "valid parameters",
			projectID:         "test-project",
			cloudProvider:     "AWS",
			endpointServiceID: "test-service",
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetPrivateEndpointService(ctx, tt.projectID, tt.cloudProvider, tt.endpointServiceID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// Note: This may fail with API errors due to lack of real credentials
				// but the validation should pass
				if err != nil {
					// Check if it's a validation error or API error
					assert.NotContains(t, err.Error(), "projectID, cloudProvider, and endpointServiceID are required")
				}
			}
		})
	}
}

func TestVPCEndpointsService_CreatePrivateEndpointService_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	tests := []struct {
		name           string
		projectID      string
		cloudProvider  string
		serviceRequest admin.CloudProviderEndpointServiceRequest
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "missing project ID",
			projectID:      "",
			cloudProvider:  "AWS",
			serviceRequest: *admin.NewCloudProviderEndpointServiceRequest("AWS", "us-east-1"),
			expectError:    true,
			errorMsg:       "projectID and cloudProvider are required",
		},
		{
			name:           "missing cloud provider",
			projectID:      "test-project",
			cloudProvider:  "",
			serviceRequest: *admin.NewCloudProviderEndpointServiceRequest("AWS", "us-east-1"),
			expectError:    true,
			errorMsg:       "projectID and cloudProvider are required",
		},
		{
			name:           "invalid service request - empty provider",
			projectID:      "test-project",
			cloudProvider:  "AWS",
			serviceRequest: *admin.NewCloudProviderEndpointServiceRequest("", "us-east-1"),
			expectError:    true,
			errorMsg:       "provider name is required",
		},
		{
			name:           "invalid service request - empty region",
			projectID:      "test-project",
			cloudProvider:  "AWS",
			serviceRequest: *admin.NewCloudProviderEndpointServiceRequest("AWS", ""),
			expectError:    true,
			errorMsg:       "region is required",
		},
		{
			name:           "invalid service request - invalid provider",
			projectID:      "test-project",
			cloudProvider:  "AWS",
			serviceRequest: *admin.NewCloudProviderEndpointServiceRequest("INVALID", "us-east-1"),
			expectError:    true,
			errorMsg:       "invalid provider name",
		},
		{
			name:           "valid parameters",
			projectID:      "test-project",
			cloudProvider:  "AWS",
			serviceRequest: *admin.NewCloudProviderEndpointServiceRequest("AWS", "us-east-1"),
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreatePrivateEndpointService(ctx, tt.projectID, tt.cloudProvider, tt.serviceRequest)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// Note: This may fail with API errors due to lack of real credentials
				// but the validation should pass
				if err != nil {
					// Check if it's a validation error or API error
					assert.NotContains(t, err.Error(), "projectID and cloudProvider are required")
					assert.NotContains(t, err.Error(), "service request validation failed")
				}
			}
		})
	}
}

func TestVPCEndpointsService_DeletePrivateEndpointService_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	tests := []struct {
		name              string
		projectID         string
		cloudProvider     string
		endpointServiceID string
		expectError       bool
		errorMsg          string
	}{
		{
			name:              "missing project ID",
			projectID:         "",
			cloudProvider:     "AWS",
			endpointServiceID: "test-service",
			expectError:       true,
			errorMsg:          "projectID, cloudProvider, and endpointServiceID are required",
		},
		{
			name:              "missing cloud provider",
			projectID:         "test-project",
			cloudProvider:     "",
			endpointServiceID: "test-service",
			expectError:       true,
			errorMsg:          "projectID, cloudProvider, and endpointServiceID are required",
		},
		{
			name:              "missing endpoint service ID",
			projectID:         "test-project",
			cloudProvider:     "AWS",
			endpointServiceID: "",
			expectError:       true,
			errorMsg:          "projectID, cloudProvider, and endpointServiceID are required",
		},
		{
			name:              "valid parameters",
			projectID:         "test-project",
			cloudProvider:     "AWS",
			endpointServiceID: "test-service",
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeletePrivateEndpointService(ctx, tt.projectID, tt.cloudProvider, tt.endpointServiceID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// Note: This may fail with API errors due to lack of real credentials
				// but the validation should pass
				if err != nil {
					// Check if it's a validation error or API error
					assert.NotContains(t, err.Error(), "projectID, cloudProvider, and endpointServiceID are required")
				}
			}
		})
	}
}

func TestVPCEndpointsService_UpdatePrivateEndpointService_Validation(t *testing.T) {
	tests := []struct {
		name                string
		projectID           string
		cloudProvider       string
		endpointServiceID   string
		expectError         bool
		expectedErrorSubstr string
	}{
		{
			name:                "missing project ID",
			projectID:           "",
			cloudProvider:       "AWS",
			endpointServiceID:   "service-123",
			expectError:         true,
			expectedErrorSubstr: "projectID",
		},
		{
			name:                "missing cloud provider",
			projectID:           "project-123",
			cloudProvider:       "",
			endpointServiceID:   "service-123",
			expectError:         true,
			expectedErrorSubstr: "cloudProvider",
		},
		{
			name:                "missing endpoint service ID",
			projectID:           "project-123",
			cloudProvider:       "AWS",
			endpointServiceID:   "",
			expectError:         true,
			expectedErrorSubstr: "endpointServiceID",
		},
		{
			name:              "valid parameters",
			projectID:         "project-123",
			cloudProvider:     "AWS",
			endpointServiceID: "service-123",
			expectError:       true, // Expected since we don't have a real client setup
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := atlasclient.NewClient(atlasclient.Config{})
			require.NoError(t, err)
			service := NewVPCEndpointsService(client)

			result, err := service.UpdatePrivateEndpointService(context.Background(), tt.projectID, tt.cloudProvider, tt.endpointServiceID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorSubstr != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorSubstr)
				}
				assert.Nil(t, result)
			}
		})
	}
}

func TestVPCEndpointsService_validateEndpointServiceRequest(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)

	tests := []struct {
		name        string
		request     *admin.CloudProviderEndpointServiceRequest
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
			errorMsg:    "service request is required",
		},
		{
			name:        "empty provider name",
			request:     admin.NewCloudProviderEndpointServiceRequest("", "us-east-1"),
			expectError: true,
			errorMsg:    "provider name is required",
		},
		{
			name:        "empty region",
			request:     admin.NewCloudProviderEndpointServiceRequest("AWS", ""),
			expectError: true,
			errorMsg:    "region is required",
		},
		{
			name:        "invalid provider name",
			request:     admin.NewCloudProviderEndpointServiceRequest("INVALID", "us-east-1"),
			expectError: true,
			errorMsg:    "invalid provider name",
		},
		{
			name:        "valid AWS request",
			request:     admin.NewCloudProviderEndpointServiceRequest("AWS", "us-east-1"),
			expectError: false,
		},
		{
			name:        "valid AZURE request",
			request:     admin.NewCloudProviderEndpointServiceRequest("AZURE", "eastus"),
			expectError: false,
		},
		{
			name:        "valid GCP request",
			request:     admin.NewCloudProviderEndpointServiceRequest("GCP", "us-central1"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateEndpointServiceRequest(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
