package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// VPCEndpointsService provides CRUD operations for Atlas VPC Endpoints and Private Link.
// This service manages AWS PrivateLink endpoints for secure connectivity to Atlas clusters.
type VPCEndpointsService struct {
	client *atlasclient.Client
}

// NewVPCEndpointsService creates a new VPCEndpointsService instance.
func NewVPCEndpointsService(client *atlasclient.Client) *VPCEndpointsService {
	return &VPCEndpointsService{client: client}
}

// ListPrivateEndpoints returns all private endpoints for the specified project.
// Note: This is a placeholder implementation until the API becomes available in the SDK.
func (s *VPCEndpointsService) ListPrivateEndpoints(ctx context.Context, projectID string) ([]admin.PrivateLinkEndpoint, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}

	// The Atlas API organizes endpoints under an endpoint service per provider
	// We need to list services first, then for each service, query the endpoint(s) as needed
	var aggregate []admin.PrivateLinkEndpoint
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		// For now, iterate known providers to gather services
		providers := []string{"AWS", "AZURE", "GCP"}
		for _, provider := range providers {
			services, _, err := api.PrivateEndpointServicesApi.ListPrivateEndpointServices(ctx, projectID, provider).Execute()
			if err != nil {
				// Ignore provider not enabled errors; continue others
				if admin.IsErrorCode(err, "NOT_FOUND") || admin.IsErrorCode(err, "RESOURCE_NOT_FOUND") {
					continue
				}
				return err
			}
			for _, svc := range services {
				// Best effort: attempt to fetch a specific endpoint list if available
				// The API doesn't expose a direct list endpoints under service; rely on Get for known ids is not possible here.
				// So we append placeholder with service information as no list endpoint exists in this SDK.
				_ = svc // keep reserved for future expansion
			}
		}
		return nil
	})
	return aggregate, err
}

// GetPrivateEndpoint returns a specific private endpoint by ID.
func (s *VPCEndpointsService) GetPrivateEndpoint(ctx context.Context, projectID, endpointID string) (*admin.PrivateLinkEndpoint, error) {
	if projectID == "" || endpointID == "" {
		return nil, fmt.Errorf("projectID and endpointID are required")
	}

	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		// We must supply cloud provider and endpoint service id; without those, we cannot query.
		// Return a clear error to caller to provide full identifiers.
		return fmt.Errorf("cloud provider and endpoint service id are required to get a private endpoint via SDK")
	})
	return nil, err
}

// CreatePrivateEndpoint creates a new private endpoint service.
func (s *VPCEndpointsService) CreatePrivateEndpoint(ctx context.Context, projectID string, endpoint *admin.PrivateLinkEndpoint) (*admin.PrivateLinkEndpoint, error) {
	if projectID == "" || endpoint == nil {
		return nil, fmt.Errorf("projectID and endpoint are required")
	}

	// Validate the endpoint configuration
	if err := s.validatePrivateEndpoint(endpoint); err != nil {
		return nil, fmt.Errorf("endpoint validation failed: %w", err)
	}

	// Creating a Private Endpoint requires an existing Endpoint Service (per provider)
	return nil, fmt.Errorf("creating private endpoints requires endpointServiceId and provider; use the endpoint service create first")
}

// DeletePrivateEndpoint removes a private endpoint.
func (s *VPCEndpointsService) DeletePrivateEndpoint(ctx context.Context, projectID, endpointID string) error {
	if projectID == "" || endpointID == "" {
		return fmt.Errorf("projectID and endpointID are required")
	}

	return fmt.Errorf("deleting private endpoints requires provider and endpointServiceId; not enough identifiers provided")
}

// validatePrivateEndpoint validates the private endpoint configuration.
func (s *VPCEndpointsService) validatePrivateEndpoint(endpoint *admin.PrivateLinkEndpoint) error {
	// Basic validation - specific field validation will be added when
	// the Atlas SDK structure is finalized
	if endpoint == nil {
		return fmt.Errorf("endpoint configuration is required")
	}

	// Additional validation logic will be implemented when the
	// PrivateLinkEndpoint struct fields are available in the SDK
	return nil
}

// GetConnectionString generates the connection string for a private endpoint.
func (s *VPCEndpointsService) GetConnectionString(ctx context.Context, projectID, endpointID string) (string, error) {
	if projectID == "" || endpointID == "" {
		return "", fmt.Errorf("projectID and endpointID are required")
	}

	// Can't build connection string without complete endpoint info in current SDK
	return "", fmt.Errorf("connection string cannot be generated without endpoint details")
}

// WaitForEndpointAvailable waits for a private endpoint to become available.
func (s *VPCEndpointsService) WaitForEndpointAvailable(ctx context.Context, projectID, endpointID string) error {
	if projectID == "" || endpointID == "" {
		return fmt.Errorf("projectID and endpointID are required")
	}

	// Simple immediate check; for production implement polling with backoff
	// SDK requires additional identifiers we don't currently have; skip real polling
	return fmt.Errorf("endpoint status monitoring requires additional identifiers (provider, service id)")
}
