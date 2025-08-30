package atlas

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

func TestNewNetworkPeeringService(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkPeeringService(client)
	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
}

func TestNetworkPeeringService_CreatePeeringConnection_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkPeeringService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		connection  *admin.BaseNetworkPeeringConnectionSettings
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing project ID",
			projectID:   "",
			connection:  &admin.BaseNetworkPeeringConnectionSettings{},
			expectError: true,
			errorMsg:    "projectID and connection are required",
		},
		{
			name:        "nil connection",
			projectID:   "test-project",
			connection:  nil,
			expectError: true,
			errorMsg:    "projectID and connection are required",
		},
		{
			name:        "empty connection",
			projectID:   "test-project",
			connection:  &admin.BaseNetworkPeeringConnectionSettings{},
			expectError: true,
			errorMsg:    "providerName is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreatePeeringConnection(ctx, tt.projectID, tt.connection)

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

func TestNetworkPeeringService_UpdatePeeringConnection_Validation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkPeeringService(client)
	ctx := context.Background()

	tests := []struct {
		name        string
		projectID   string
		peerID      string
		connection  *admin.BaseNetworkPeeringConnectionSettings
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing project ID",
			projectID:   "",
			peerID:      "peer-1",
			connection:  &admin.BaseNetworkPeeringConnectionSettings{},
			expectError: true,
			errorMsg:    "projectID, peerID, and connection are required",
		},
		{
			name:        "missing peer ID",
			projectID:   "test-project",
			peerID:      "",
			connection:  &admin.BaseNetworkPeeringConnectionSettings{},
			expectError: true,
			errorMsg:    "projectID, peerID, and connection are required",
		},
		{
			name:        "nil connection",
			projectID:   "test-project",
			peerID:      "peer-1",
			connection:  nil,
			expectError: true,
			errorMsg:    "projectID, peerID, and connection are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.UpdatePeeringConnection(ctx, tt.projectID, tt.peerID, tt.connection)

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

func TestNetworkPeeringService_validatePeeringConnection(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkPeeringService(client)

	t.Run("missing provider name", func(t *testing.T) {
		connection := &admin.BaseNetworkPeeringConnectionSettings{}
		err := service.validatePeeringConnection(connection)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "providerName is required")
	})

	// Additional validation tests would go here once we understand
	// the exact structure of admin.BaseNetworkPeeringConnectionSettings from the Atlas SDK
}

func TestNetworkPeeringService_ValidatePeeringCIDRs(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkPeeringService(client)
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
			err := service.ValidatePeeringCIDRs(ctx, tt.projectID, tt.newCIDR)

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

func TestNetworkPeeringService_validateCIDR(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkPeeringService(client)

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
			name:        "invalid CIDR",
			cidr:        "invalid",
			expectError: true,
		},
		{
			name:        "invalid CIDR format",
			cidr:        "10.0.0.0/33",
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

func TestNetworkPeeringService_BasicOperations(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkPeeringService(client)
	ctx := context.Background()

	// Test ListPeeringConnections
	t.Run("ListPeeringConnections", func(t *testing.T) {
		_, err := service.ListPeeringConnections(ctx, "test-project")
		// Should fail due to missing API but not due to validation
		assert.Error(t, err)
	})

	// Test GetPeeringConnection
	t.Run("GetPeeringConnection", func(t *testing.T) {
		_, err := service.GetPeeringConnection(ctx, "test-project", "peer-1")
		// Should fail due to missing API but not due to validation
		assert.Error(t, err)
	})

	// Test DeletePeeringConnection
	t.Run("DeletePeeringConnection", func(t *testing.T) {
		err := service.DeletePeeringConnection(ctx, "test-project", "peer-1")
		// Should fail due to missing API but not due to validation
		assert.Error(t, err)
	})

	// Test GetPeeringConnectionStatus
	t.Run("GetPeeringConnectionStatus", func(t *testing.T) {
		_, err := service.GetPeeringConnectionStatus(ctx, "test-project", "peer-1")
		// Should fail due to missing API but not due to validation
		assert.Error(t, err)
	})
}

func TestNetworkPeeringService_ParameterValidation(t *testing.T) {
	client, err := atlasclient.NewClient(atlasclient.Config{})
	require.NoError(t, err)

	service := NewNetworkPeeringService(client)
	ctx := context.Background()

	// Test empty project IDs
	t.Run("empty project ID validation", func(t *testing.T) {
		_, err := service.ListPeeringConnections(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID required")

		_, err = service.GetPeeringConnection(ctx, "", "peer-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and peerID are required")

		err = service.DeletePeeringConnection(ctx, "", "peer-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and peerID are required")
	})

	// Test empty peer IDs
	t.Run("empty peer ID validation", func(t *testing.T) {
		_, err := service.GetPeeringConnection(ctx, "test-project", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and peerID are required")

		err = service.DeletePeeringConnection(ctx, "test-project", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "projectID and peerID are required")
	})
}
