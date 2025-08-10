package atlas

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

func TestNewNetworkContainersService(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkContainersService(client)
	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
}

func TestNetworkContainersService_CreateNetworkContainer_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkContainersService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		container   *admin.CloudProviderContainer
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing project ID",
			projectID:   "",
			container:   &admin.CloudProviderContainer{},
			expectError: true,
			errorMsg:    "projectID and container are required",
		},
		{
			name:        "nil container",
			projectID:   "test-project",
			container:   nil,
			expectError: true,
			errorMsg:    "projectID and container are required",
		},
		{
			name:        "empty container",
			projectID:   "test-project",
			container:   &admin.CloudProviderContainer{},
			expectError: true,
			errorMsg:    "providerName is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateNetworkContainer(ctx, tt.projectID, tt.container)

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

func TestNetworkContainersService_validateNetworkContainer(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkContainersService(client)

	t.Run("missing provider name", func(t *testing.T) {
		container := &admin.CloudProviderContainer{}
		err := service.validateNetworkContainer(container)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "providerName is required")
	})

	// Additional validation tests would go here once we understand
	// the exact structure of admin.CloudProviderContainer from the Atlas SDK
}

func TestNetworkContainersService_validateCIDR(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkContainersService(client)

	tests := []struct {
		name        string
		cidr        string
		expectError bool
	}{
		{
			name:        "valid CIDR",
			cidr:        "10.0.0.0/16",
			expectError: false,
		},
		{
			name:        "valid CIDR /24",
			cidr:        "192.168.1.0/24",
			expectError: false,
		},
		{
			name:        "valid CIDR /8",
			cidr:        "10.0.0.0/8",
			expectError: false,
		},
		{
			name:        "invalid CIDR",
			cidr:        "invalid",
			expectError: true,
		},
		{
			name:        "invalid CIDR format",
			cidr:        "10.0.0.0/33",
			expectError: true,
		},
		{
			name:        "empty CIDR",
			cidr:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateCIDR(tt.cidr)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNetworkContainersService_validateCIDRSize(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkContainersService(client)

	tests := []struct {
		name        string
		provider    string
		cidr        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "AWS valid /16",
			provider:    "AWS",
			cidr:        "10.0.0.0/16",
			expectError: false,
		},
		{
			name:        "AWS valid /24",
			provider:    "AWS",
			cidr:        "10.0.0.0/24",
			expectError: false,
		},
		{
			name:        "AWS invalid /8",
			provider:    "AWS",
			cidr:        "10.0.0.0/8",
			expectError: true,
			errorMsg:    "AWS network containers require CIDR blocks between /16 and /24",
		},
		{
			name:        "AWS invalid /28",
			provider:    "AWS",
			cidr:        "10.0.0.0/28",
			expectError: true,
			errorMsg:    "AWS network containers require CIDR blocks between /16 and /24",
		},
		{
			name:        "GCP valid /16",
			provider:    "GCP",
			cidr:        "10.0.0.0/16",
			expectError: false,
		},
		{
			name:        "GCP valid /29",
			provider:    "GCP",
			cidr:        "10.0.0.0/29",
			expectError: false,
		},
		{
			name:        "GCP invalid /8",
			provider:    "GCP",
			cidr:        "10.0.0.0/8",
			expectError: true,
			errorMsg:    "GCP network containers require CIDR blocks between /16 and /29",
		},
		{
			name:        "Azure valid /16",
			provider:    "AZURE",
			cidr:        "10.0.0.0/16",
			expectError: false,
		},
		{
			name:        "Azure valid /24",
			provider:    "AZURE",
			cidr:        "10.0.0.0/24",
			expectError: false,
		},
		{
			name:        "Azure invalid /8",
			provider:    "AZURE",
			cidr:        "10.0.0.0/8",
			expectError: true,
			errorMsg:    "azure network containers require CIDR blocks between /16 and /24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateCIDRSize(tt.provider, tt.cidr)

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

func TestNetworkContainersService_SuggestAvailableCIDR(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkContainersService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		provider    string
		prefixLen   int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "AWS valid prefix",
			projectID:   "test-project",
			provider:    "AWS",
			prefixLen:   16,
			expectError: false, // Should succeed and return a suggested CIDR
		},
		{
			name:        "AWS invalid prefix",
			projectID:   "test-project",
			provider:    "AWS",
			prefixLen:   8,
			expectError: true,
			errorMsg:    "AWS network containers require CIDR blocks between /16 and /24",
		},
		{
			name:        "GCP valid prefix",
			projectID:   "test-project",
			provider:    "GCP",
			prefixLen:   20,
			expectError: false, // Should succeed and return a suggested CIDR
		},
		{
			name:        "invalid provider",
			projectID:   "test-project",
			provider:    "INVALID",
			prefixLen:   16,
			expectError: true,
			errorMsg:    "unsupported cloud provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.SuggestAvailableCIDR(ctx, tt.projectID, tt.provider, tt.prefixLen)

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

func TestNetworkContainersService_ValidateNoOverlappingCIDRs(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkContainersService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		newCIDR     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid CIDR format",
			projectID:   "test-project",
			newCIDR:     "invalid-cidr",
			expectError: true,
			errorMsg:    "invalid new CIDR",
		},
		{
			name:        "valid CIDR format",
			projectID:   "test-project",
			newCIDR:     "10.0.0.0/16",
			expectError: false, // No error for valid CIDR format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateNoOverlappingCIDRs(ctx, tt.projectID, tt.newCIDR)

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

func TestCIDROverlapDetection_Helper(t *testing.T) {
	containers := []admin.CloudProviderContainer{
		{AtlasCidrBlock: strPtr("10.0.0.0/16"), Id: strPtr("c1")},
		{AtlasCidrBlock: strPtr("172.16.0.0/20"), Id: strPtr("c2")},
	}

	// Overlapping with 10.0.0.0/16
	err := checkCIDROverlapWithContainers("10.0.5.0/24", containers)
	assert.Error(t, err)

	// Non-overlapping candidate
	err = checkCIDROverlapWithContainers("192.168.0.0/16", containers)
	assert.NoError(t, err)
}

func strPtr(s string) *string { return &s }

func TestNetworkContainersService_BasicOperations(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkContainersService(client)
	ctx := context.Background()

	// Test ListNetworkContainers
	t.Run("ListNetworkContainers", func(t *testing.T) {
		_, err := service.ListNetworkContainers(ctx, "test-project")
		assert.Error(t, err)
	})

	// Test GetNetworkContainer
	t.Run("GetNetworkContainer", func(t *testing.T) {
		_, err := service.GetNetworkContainer(ctx, "test-project", "container-1")
		assert.Error(t, err)
	})

	// Test DeleteNetworkContainer
	t.Run("DeleteNetworkContainer", func(t *testing.T) {
		err := service.DeleteNetworkContainer(ctx, "test-project", "container-1")
		assert.Error(t, err)
	})

	// Test GetNetworkContainersByRegion
	t.Run("GetNetworkContainersByRegion", func(t *testing.T) {
		_, err := service.GetNetworkContainersByRegion(ctx, "test-project", "US_EAST_1")
		// Should fail due to API call
		assert.Error(t, err)
	})

	// Test GetNetworkContainersByProvider
	t.Run("GetNetworkContainersByProvider", func(t *testing.T) {
		_, err := service.GetNetworkContainersByProvider(ctx, "test-project", "AWS")
		// Should fail due to API call
		assert.Error(t, err)
	})
}

func TestNetworkContainersService_ParameterValidation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkContainersService(client)
	ctx := context.Background()

	// Test empty project IDs
	t.Run("empty project ID validation", func(t *testing.T) {
		_, err := service.ListNetworkContainers(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID required")

		_, err = service.GetNetworkContainer(ctx, "", "container-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and containerID are required")

		err = service.DeleteNetworkContainer(ctx, "", "container-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and containerID are required")
	})

	// Test empty container IDs
	t.Run("empty container ID validation", func(t *testing.T) {
		_, err := service.GetNetworkContainer(ctx, "test-project", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and containerID are required")

		err = service.DeleteNetworkContainer(ctx, "test-project", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and containerID are required")
	})
}
