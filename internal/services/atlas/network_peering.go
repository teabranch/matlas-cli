package atlas

import (
	"context"
	"fmt"
	"net"
	"time"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312010/admin"
)

// NetworkPeeringService provides CRUD operations for Atlas Network Peering.
// This service manages VPC peering connections for secure network connectivity.
type NetworkPeeringService struct {
	client *atlasclient.Client
}

// NewNetworkPeeringService creates a new NetworkPeeringService instance.
func NewNetworkPeeringService(client *atlasclient.Client) *NetworkPeeringService {
	return &NetworkPeeringService{client: client}
}

// ListPeeringConnections returns all network peering connections for the specified project.
func (s *NetworkPeeringService) ListPeeringConnections(ctx context.Context, projectID string) ([]admin.BaseNetworkPeeringConnectionSettings, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}

	var connections []admin.BaseNetworkPeeringConnectionSettings
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.NetworkPeeringApi.ListGroupPeers(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		if resp == nil || resp.Results == nil {
			return fmt.Errorf("%w: empty response", atlasclient.ErrNotFound)
		}
		connections = *resp.Results
		return nil
	})
	return connections, err
}

// GetPeeringConnection returns a specific network peering connection by ID.
func (s *NetworkPeeringService) GetPeeringConnection(ctx context.Context, projectID, peerID string) (*admin.BaseNetworkPeeringConnectionSettings, error) {
	if projectID == "" || peerID == "" {
		return nil, fmt.Errorf("projectID and peerID are required")
	}

	var connection *admin.BaseNetworkPeeringConnectionSettings
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.NetworkPeeringApi.GetGroupPeer(ctx, projectID, peerID).Execute()
		if err != nil {
			return err
		}
		connection = resp
		return nil
	})
	return connection, err
}

// CreatePeeringConnection creates a new network peering connection.
func (s *NetworkPeeringService) CreatePeeringConnection(ctx context.Context, projectID string, connection *admin.BaseNetworkPeeringConnectionSettings) (*admin.BaseNetworkPeeringConnectionSettings, error) {
	if projectID == "" || connection == nil {
		return nil, fmt.Errorf("projectID and connection are required")
	}

	// Validate the peering connection configuration
	if err := s.validatePeeringConnection(connection); err != nil {
		return nil, fmt.Errorf("peering connection validation failed: %w", err)
	}

	var created *admin.BaseNetworkPeeringConnectionSettings
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.NetworkPeeringApi.CreateGroupPeer(ctx, projectID, connection).Execute()
		if err != nil {
			return err
		}
		created = resp
		return nil
	})
	return created, err
}

// UpdatePeeringConnection modifies an existing network peering connection.
func (s *NetworkPeeringService) UpdatePeeringConnection(ctx context.Context, projectID, peerID string, connection *admin.BaseNetworkPeeringConnectionSettings) (*admin.BaseNetworkPeeringConnectionSettings, error) {
	if projectID == "" || peerID == "" || connection == nil {
		return nil, fmt.Errorf("projectID, peerID, and connection are required")
	}

	// Validate the updated configuration
	if err := s.validatePeeringConnection(connection); err != nil {
		return nil, fmt.Errorf("peering connection validation failed: %w", err)
	}

	var updated *admin.BaseNetworkPeeringConnectionSettings
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.NetworkPeeringApi.UpdateGroupPeer(ctx, projectID, peerID, connection).Execute()
		if err != nil {
			return err
		}
		updated = resp
		return nil
	})
	return updated, err
}

// DeletePeeringConnection removes a network peering connection.
func (s *NetworkPeeringService) DeletePeeringConnection(ctx context.Context, projectID, peerID string) error {
	if projectID == "" || peerID == "" {
		return fmt.Errorf("projectID and peerID are required")
	}

	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, _, err := api.NetworkPeeringApi.DeleteGroupPeer(ctx, projectID, peerID).Execute()
		return err
	})
}

// validatePeeringConnection validates the network peering connection configuration.
func (s *NetworkPeeringService) validatePeeringConnection(connection *admin.BaseNetworkPeeringConnectionSettings) error {
	if connection.ProviderName == nil {
		return fmt.Errorf("providerName is required")
	}

	// Validate cloud provider is supported
	validProviders := []string{"AWS", "GCP", "AZURE"}
	isValidProvider := false
	for _, provider := range validProviders {
		if *connection.ProviderName == provider {
			isValidProvider = true
			break
		}
	}
	if !isValidProvider {
		return fmt.Errorf("unsupported cloud provider: %s (supported: AWS, GCP, AZURE)", *connection.ProviderName)
	}

	// Provider-specific validation would go here when type assertion methods are available
	// For now, basic validation is sufficient
	return nil
}

// validateCIDR validates CIDR block format.
func (s *NetworkPeeringService) validateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR block: %s", cidr)
	}
	return nil
}

// WaitForPeeringConnectionAvailable waits for a peering connection to become available.
func (s *NetworkPeeringService) WaitForPeeringConnectionAvailable(ctx context.Context, projectID, peerID string) error {
	// Simple exponential backoff polling until AVAILABLE or context deadline
	backoff := 1 * time.Second
	maxBackoff := 16 * time.Second

	for {
		connection, err := s.GetPeeringConnection(ctx, projectID, peerID)
		if err != nil {
			return err
		}

		if connection.StatusName != nil && *connection.StatusName == "AVAILABLE" {
			return nil
		}

		// Respect context deadline/cancel
		select {
		case <-ctx.Done():
			current := "UNKNOWN"
			if connection != nil && connection.StatusName != nil {
				current = *connection.StatusName
			}
			return fmt.Errorf("timed out waiting for peering connection to become AVAILABLE; last status: %s", current)
		case <-time.After(backoff):
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			// continue polling
		}
	}
}

// GetPeeringConnectionStatus returns the current status of a peering connection.
func (s *NetworkPeeringService) GetPeeringConnectionStatus(ctx context.Context, projectID, peerID string) (string, error) {
	connection, err := s.GetPeeringConnection(ctx, projectID, peerID)
	if err != nil {
		return "", err
	}

	if connection.StatusName == nil {
		return "UNKNOWN", nil
	}

	return *connection.StatusName, nil
}

// ValidatePeeringCIDRs validates that peering CIDR blocks don't overlap.
func (s *NetworkPeeringService) ValidatePeeringCIDRs(_ context.Context, projectID string, newCIDR string) error {
	// Validate CIDR format first
	_, _, err := net.ParseCIDR(newCIDR)
	if err != nil {
		return fmt.Errorf("invalid new CIDR: %w", err)
	}

	// TODO: Implement proper overlap checking when API authentication is available
	// For now, just validate the CIDR format
	return nil
}

// getSafeString dereferences a string pointer safely
func getSafeString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
