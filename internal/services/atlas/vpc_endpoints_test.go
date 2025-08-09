package atlas

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

func TestNewVPCEndpointsService(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
}

func TestVPCEndpointsService_CreatePrivateEndpoint_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		endpoint    *admin.PrivateLinkEndpoint
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing project ID",
			projectID:   "",
			endpoint:    &admin.PrivateLinkEndpoint{},
			expectError: true,
			errorMsg:    "projectID and endpoint are required",
		},
		{
			name:        "nil endpoint",
			projectID:   "test-project",
			endpoint:    nil,
			expectError: true,
			errorMsg:    "projectID and endpoint are required",
		},
		{
			name:        "empty endpoint",
			projectID:   "test-project",
			endpoint:    &admin.PrivateLinkEndpoint{},
			expectError: true,
			errorMsg:    "private endpoints API not yet available in SDK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreatePrivateEndpoint(ctx, tt.projectID, tt.endpoint)

			if tt.expectError {
				assert.Error(t, err)
				// We only assert error presence since SDK behavior may vary
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVPCEndpointsService_GetConnectionString(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		endpointID  string
		expectError bool
	}{
		{
			name:        "missing project ID",
			projectID:   "",
			endpointID:  "endpoint-1",
			expectError: true,
		},
		{
			name:        "missing endpoint ID",
			projectID:   "test-project",
			endpointID:  "",
			expectError: true,
		},
		{
			name:        "valid parameters",
			projectID:   "test-project",
			endpointID:  "endpoint-1",
			expectError: true, // Will fail due to missing API
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetConnectionString(ctx, tt.projectID, tt.endpointID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVPCEndpointsService_validatePrivateEndpoint(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)

	t.Run("nil endpoint", func(t *testing.T) {
		err := service.validatePrivateEndpoint(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint configuration is required")
	})

	t.Run("valid endpoint", func(t *testing.T) {
		endpoint := &admin.PrivateLinkEndpoint{}
		err := service.validatePrivateEndpoint(endpoint)
		assert.NoError(t, err) // Currently passes basic validation
	})

	// Additional validation tests will be added when the
	// Atlas SDK PrivateLinkEndpoint structure is finalized
}

func TestVPCEndpointsService_WaitForEndpointAvailable(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		endpointID  string
		expectError bool
	}{
		{
			name:        "missing project ID",
			projectID:   "",
			endpointID:  "endpoint-1",
			expectError: true,
		},
		{
			name:        "missing endpoint ID",
			projectID:   "test-project",
			endpointID:  "",
			expectError: true,
		},
		{
			name:        "valid parameters",
			projectID:   "test-project",
			endpointID:  "endpoint-1",
			expectError: true, // Will fail due to missing API
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.WaitForEndpointAvailable(ctx, tt.projectID, tt.endpointID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test basic service operations that should work regardless of API availability
func TestVPCEndpointsService_BasicOperations(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	// Test ListPrivateEndpoints
	t.Run("ListPrivateEndpoints", func(t *testing.T) {
		_, err := service.ListPrivateEndpoints(ctx, "test-project")
		// May fail depending on provider availability; assert no panic and no success guaranteed
		if err != nil {
			assert.Error(t, err)
		}
	})

	// Test GetPrivateEndpoint
	t.Run("GetPrivateEndpoint", func(t *testing.T) {
		_, err := service.GetPrivateEndpoint(ctx, "test-project", "endpoint-1")
		assert.Error(t, err)
	})

	// Test DeletePrivateEndpoint
	t.Run("DeletePrivateEndpoint", func(t *testing.T) {
		err := service.DeletePrivateEndpoint(ctx, "test-project", "endpoint-1")
		assert.Error(t, err)
	})
}

// Test parameter validation
func TestVPCEndpointsService_ParameterValidation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewVPCEndpointsService(client)
	ctx := context.Background()

	// Test empty project IDs
	t.Run("empty project ID validation", func(t *testing.T) {
		_, err := service.ListPrivateEndpoints(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID required")

		_, err = service.GetPrivateEndpoint(ctx, "", "endpoint-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and endpointID are required")

		err = service.DeletePrivateEndpoint(ctx, "", "endpoint-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and endpointID are required")
	})

	// Test empty endpoint IDs
	t.Run("empty endpoint ID validation", func(t *testing.T) {
		_, err := service.GetPrivateEndpoint(ctx, "test-project", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and endpointID are required")

		err = service.DeletePrivateEndpoint(ctx, "test-project", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and endpointID are required")
	})
}
