package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
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

// ListPrivateEndpointServices returns all private endpoint services for the specified project and provider.
func (s *VPCEndpointsService) ListPrivateEndpointServices(ctx context.Context, projectID, cloudProvider string) ([]admin.EndpointService, error) {
	if projectID == "" || cloudProvider == "" {
		return nil, fmt.Errorf("projectID and cloudProvider are required")
	}

	var services []admin.EndpointService
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.PrivateEndpointServicesApi.ListPrivateEndpointServices(ctx, projectID, cloudProvider).Execute()
		if err != nil {
			return err
		}
		services = result
		return nil
	})
	return services, err
}

// ListAllPrivateEndpointServices returns all private endpoint services across all cloud providers.
func (s *VPCEndpointsService) ListAllPrivateEndpointServices(ctx context.Context, projectID string) (map[string][]admin.EndpointService, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}

	result := make(map[string][]admin.EndpointService)
	providers := []string{"AWS", "AZURE", "GCP"}

	for _, provider := range providers {
		services, err := s.ListPrivateEndpointServices(ctx, projectID, provider)
		if err != nil {
			// Ignore provider not enabled errors; continue with others
			if admin.IsErrorCode(err, "NOT_FOUND") || admin.IsErrorCode(err, "RESOURCE_NOT_FOUND") {
				continue
			}
			return nil, fmt.Errorf("failed to list services for provider %s: %w", provider, err)
		}
		if len(services) > 0 {
			result[provider] = services
		}
	}
	return result, nil
}

// GetPrivateEndpointService returns a specific private endpoint service by ID.
func (s *VPCEndpointsService) GetPrivateEndpointService(ctx context.Context, projectID, cloudProvider, endpointServiceID string) (*admin.EndpointService, error) {
	if projectID == "" || cloudProvider == "" || endpointServiceID == "" {
		return nil, fmt.Errorf("projectID, cloudProvider, and endpointServiceID are required")
	}

	var service *admin.EndpointService
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.PrivateEndpointServicesApi.GetPrivateEndpointService(ctx, projectID, cloudProvider, endpointServiceID).Execute()
		if err != nil {
			return err
		}
		service = result
		return nil
	})
	return service, err
}

// CreatePrivateEndpointService creates a new private endpoint service.
func (s *VPCEndpointsService) CreatePrivateEndpointService(ctx context.Context, projectID, cloudProvider string, serviceRequest admin.CloudProviderEndpointServiceRequest) (*admin.EndpointService, error) {
	if projectID == "" || cloudProvider == "" {
		return nil, fmt.Errorf("projectID and cloudProvider are required")
	}

	// Validate the service request
	if err := s.validateEndpointServiceRequest(&serviceRequest); err != nil {
		return nil, fmt.Errorf("service request validation failed: %w", err)
	}

	var service *admin.EndpointService
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.PrivateEndpointServicesApi.CreatePrivateEndpointService(ctx, projectID, &serviceRequest).Execute()
		if err != nil {
			return err
		}
		service = result
		return nil
	})
	return service, err
}

// DeletePrivateEndpointService removes a private endpoint service.
func (s *VPCEndpointsService) DeletePrivateEndpointService(ctx context.Context, projectID, cloudProvider, endpointServiceID string) error {
	if projectID == "" || cloudProvider == "" || endpointServiceID == "" {
		return fmt.Errorf("projectID, cloudProvider, and endpointServiceID are required")
	}

	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.PrivateEndpointServicesApi.DeletePrivateEndpointService(ctx, projectID, cloudProvider, endpointServiceID).Execute()
		return err
	})
}

// UpdatePrivateEndpointService updates an existing private endpoint service.
// Note: Most VPC endpoint properties are immutable after creation, so this may be a no-op.
func (s *VPCEndpointsService) UpdatePrivateEndpointService(ctx context.Context, projectID, cloudProvider, endpointServiceID string) (*admin.EndpointService, error) {
	if projectID == "" || cloudProvider == "" || endpointServiceID == "" {
		return nil, fmt.Errorf("projectID, cloudProvider, and endpointServiceID are required")
	}

	// Since VPC endpoint services are largely immutable after creation,
	// we'll just return the current state of the service
	// In a real implementation, you might support updating tags or other mutable properties
	return s.GetPrivateEndpointService(ctx, projectID, cloudProvider, endpointServiceID)
}

// validateEndpointServiceRequest validates the endpoint service request configuration.
func (s *VPCEndpointsService) validateEndpointServiceRequest(request *admin.CloudProviderEndpointServiceRequest) error {
	if request == nil {
		return fmt.Errorf("service request is required")
	}

	if request.GetProviderName() == "" {
		return fmt.Errorf("provider name is required")
	}

	if request.GetRegion() == "" {
		return fmt.Errorf("region is required")
	}

	// Validate provider name is one of the supported values
	validProviders := map[string]bool{
		"AWS":   true,
		"AZURE": true,
		"GCP":   true,
	}

	if !validProviders[request.GetProviderName()] {
		return fmt.Errorf("invalid provider name: %s. Must be one of: AWS, AZURE, GCP", request.GetProviderName())
	}

	return nil
}
