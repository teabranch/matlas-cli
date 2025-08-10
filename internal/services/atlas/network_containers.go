package atlas

import (
	"context"
	"fmt"
	"net"
	"strings"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// NetworkContainersService provides CRUD operations for Atlas Network Containers.
// Network containers define the CIDR blocks for Atlas clusters and are required for VPC peering.
type NetworkContainersService struct {
	client *atlasclient.Client
}

// NewNetworkContainersService creates a new NetworkContainersService instance.
func NewNetworkContainersService(client *atlasclient.Client) *NetworkContainersService {
	return &NetworkContainersService{client: client}
}

// ListNetworkContainers returns all network containers for the specified project.
func (s *NetworkContainersService) ListNetworkContainers(ctx context.Context, projectID string) ([]admin.CloudProviderContainer, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}

	var containers []admin.CloudProviderContainer
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.NetworkPeeringApi.ListPeeringContainers(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		if resp == nil || resp.Results == nil {
			return fmt.Errorf("%w: empty response", atlasclient.ErrNotFound)
		}
		containers = *resp.Results
		return nil
	})
	return containers, err
}

// GetNetworkContainer returns a specific network container by ID.
func (s *NetworkContainersService) GetNetworkContainer(ctx context.Context, projectID, containerID string) (*admin.CloudProviderContainer, error) {
	if projectID == "" || containerID == "" {
		return nil, fmt.Errorf("projectID and containerID are required")
	}

	var container *admin.CloudProviderContainer
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.NetworkPeeringApi.GetPeeringContainer(ctx, projectID, containerID).Execute()
		if err != nil {
			return err
		}
		container = resp
		return nil
	})
	return container, err
}

// CreateNetworkContainer creates a new network container.
func (s *NetworkContainersService) CreateNetworkContainer(ctx context.Context, projectID string, container *admin.CloudProviderContainer) (*admin.CloudProviderContainer, error) {
	if projectID == "" || container == nil {
		return nil, fmt.Errorf("projectID and container are required")
	}

	// Validate the network container configuration
	if err := s.validateNetworkContainer(container); err != nil {
		return nil, fmt.Errorf("network container validation failed: %w", err)
	}

	var created *admin.CloudProviderContainer
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.NetworkPeeringApi.CreatePeeringContainer(ctx, projectID, container).Execute()
		if err != nil {
			return err
		}
		created = resp
		return nil
	})
	return created, err
}

// UpdateNetworkContainer modifies an existing network container.
func (s *NetworkContainersService) UpdateNetworkContainer(ctx context.Context, projectID, containerID string, container *admin.CloudProviderContainer) (*admin.CloudProviderContainer, error) {
	if projectID == "" || containerID == "" || container == nil {
		return nil, fmt.Errorf("projectID, containerID, and container are required")
	}

	// Validate the updated configuration
	if err := s.validateNetworkContainer(container); err != nil {
		return nil, fmt.Errorf("network container validation failed: %w", err)
	}

	var updated *admin.CloudProviderContainer
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.NetworkPeeringApi.UpdatePeeringContainer(ctx, projectID, containerID, container).Execute()
		if err != nil {
			return err
		}
		updated = resp
		return nil
	})
	return updated, err
}

// DeleteNetworkContainer removes a network container.
func (s *NetworkContainersService) DeleteNetworkContainer(ctx context.Context, projectID, containerID string) error {
	if projectID == "" || containerID == "" {
		return fmt.Errorf("projectID and containerID are required")
	}

	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.NetworkPeeringApi.DeletePeeringContainer(ctx, projectID, containerID).Execute()
		return err
	})
}

// validateNetworkContainer validates the network container configuration.
func (s *NetworkContainersService) validateNetworkContainer(container *admin.CloudProviderContainer) error {
	if container.ProviderName == nil {
		return fmt.Errorf("providerName is required")
	}

	// Validate cloud provider is supported
	validProviders := []string{"AWS", "GCP", "AZURE"}
	isValidProvider := false
	for _, provider := range validProviders {
		if *container.ProviderName == provider {
			isValidProvider = true
			break
		}
	}
	if !isValidProvider {
		return fmt.Errorf("unsupported cloud provider: %s (supported: AWS, GCP, AZURE)", *container.ProviderName)
	}

	if container.AtlasCidrBlock == nil || *container.AtlasCidrBlock == "" {
		return fmt.Errorf("atlasCidrBlock is required")
	}

	// Validate CIDR block format
	if err := s.validateCIDR(*container.AtlasCidrBlock); err != nil {
		return fmt.Errorf("invalid atlasCidrBlock: %w", err)
	}

	// Validate CIDR block size based on provider requirements
	if err := s.validateCIDRSize(*container.ProviderName, *container.AtlasCidrBlock); err != nil {
		return err
	}

	// Provider-specific validation
	switch *container.ProviderName {
	case "AWS":
		return s.validateAWSNetworkContainer(container)
	case "GCP":
		return s.validateGCPNetworkContainer(container)
	case "AZURE":
		return s.validateAzureNetworkContainer(container)
	}

	return nil
}

// validateAWSNetworkContainer validates AWS-specific network container settings.
func (s *NetworkContainersService) validateAWSNetworkContainer(container *admin.CloudProviderContainer) error {
	if container.RegionName == nil || *container.RegionName == "" {
		return fmt.Errorf("regionName is required for AWS network containers")
	}

	// Validate AWS region format
	if !strings.Contains(*container.RegionName, "-") {
		return fmt.Errorf("invalid AWS region format: %s", *container.RegionName)
	}

	return nil
}

// validateGCPNetworkContainer validates GCP-specific network container settings.
func (s *NetworkContainersService) validateGCPNetworkContainer(container *admin.CloudProviderContainer) error {
	if container.RegionName == nil || *container.RegionName == "" {
		return fmt.Errorf("regionName is required for GCP network containers")
	}

	return nil
}

// validateAzureNetworkContainer validates Azure-specific network container settings.
func (s *NetworkContainersService) validateAzureNetworkContainer(container *admin.CloudProviderContainer) error {
	if container.RegionName == nil || *container.RegionName == "" {
		return fmt.Errorf("regionName is required for Azure network containers")
	}

	return nil
}

// validateCIDR validates CIDR block format.
func (s *NetworkContainersService) validateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR block: %s", cidr)
	}
	return nil
}

// validateCIDRSize validates CIDR block size requirements for each cloud provider.
func (s *NetworkContainersService) validateCIDRSize(provider, cidr string) error {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	// Get the prefix length
	prefixLen, _ := network.Mask.Size()

	switch provider {
	case "AWS":
		// AWS requires /16 to /24 for Atlas network containers
		if prefixLen < 16 || prefixLen > 24 {
			return fmt.Errorf("AWS network containers require CIDR blocks between /16 and /24, got /%d", prefixLen)
		}
	case "GCP":
		// GCP requires /16 to /29 for Atlas network containers
		if prefixLen < 16 || prefixLen > 29 {
			return fmt.Errorf("GCP network containers require CIDR blocks between /16 and /29, got /%d", prefixLen)
		}
	case "AZURE":
		// Azure requires /16 to /24 for Atlas network containers
		if prefixLen < 16 || prefixLen > 24 {
			return fmt.Errorf("azure network containers require CIDR blocks between /16 and /24, got /%d", prefixLen)
		}
	}

	return nil
}

// GetNetworkContainersByRegion returns network containers filtered by region.
func (s *NetworkContainersService) GetNetworkContainersByRegion(ctx context.Context, projectID, region string) ([]admin.CloudProviderContainer, error) {
	containers, err := s.ListNetworkContainers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var filteredContainers []admin.CloudProviderContainer
	for _, container := range containers {
		if container.RegionName != nil && *container.RegionName == region {
			filteredContainers = append(filteredContainers, container)
		}
	}

	return filteredContainers, nil
}

// GetNetworkContainersByProvider returns network containers filtered by cloud provider.
func (s *NetworkContainersService) GetNetworkContainersByProvider(ctx context.Context, projectID, provider string) ([]admin.CloudProviderContainer, error) {
	containers, err := s.ListNetworkContainers(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var filteredContainers []admin.CloudProviderContainer
	for _, container := range containers {
		if container.ProviderName != nil && *container.ProviderName == provider {
			filteredContainers = append(filteredContainers, container)
		}
	}

	return filteredContainers, nil
}

// ValidateNoOverlappingCIDRs ensures that a new CIDR block doesn't overlap with existing containers.
func (s *NetworkContainersService) ValidateNoOverlappingCIDRs(ctx context.Context, projectID, newCIDR string) error {
	// Validate CIDR format first
	if _, _, err := net.ParseCIDR(newCIDR); err != nil {
		return fmt.Errorf("invalid new CIDR: %w", err)
	}

	// Fetch existing containers to check for overlap (best-effort)
	containers, err := s.ListNetworkContainers(ctx, projectID)
	if err != nil {
		// If API listing fails, skip overlap enforcement per current limitations
		return nil
	}

	return checkCIDROverlapWithContainers(newCIDR, containers)
}

// SuggestAvailableCIDR suggests an available CIDR block for a new network container.
func (s *NetworkContainersService) SuggestAvailableCIDR(ctx context.Context, projectID, provider string, prefixLen int) (string, error) {
	// Validate provider first
	validProviders := []string{"AWS", "GCP", "AZURE"}
	isValidProvider := false
	for _, validProvider := range validProviders {
		if provider == validProvider {
			isValidProvider = true
			break
		}
	}
	if !isValidProvider {
		return "", fmt.Errorf("unsupported cloud provider: %s (supported: AWS, GCP, AZURE)", provider)
	}

	// Validate prefix length for provider
	if err := s.validateCIDRSize(provider, fmt.Sprintf("10.0.0.0/%d", prefixLen)); err != nil {
		return "", err
	}

	// Simple algorithm to suggest an available CIDR
	// In production, you'd want more sophisticated logic
	baseCIDRs := []string{
		"10.0.0.0",
		"10.1.0.0",
		"10.2.0.0",
		"10.3.0.0",
		"172.16.0.0",
		"172.17.0.0",
		"172.18.0.0",
		"172.19.0.0",
	}

	for _, base := range baseCIDRs {
		candidate := fmt.Sprintf("%s/%d", base, prefixLen)
		if err := s.ValidateNoOverlappingCIDRs(ctx, projectID, candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no available CIDR blocks found for prefix length /%d", prefixLen)
}

// cidrOverlaps returns true if two CIDR networks overlap.
func cidrOverlaps(a, b *net.IPNet) bool {
	// If either contains the other's network address or broadcast ranges overlap, they overlap.
	return a.Contains(b.IP) || b.Contains(a.IP) || networkRangeOverlaps(a, b)
}

// networkRangeOverlaps checks overlap by computing start/end of each range.
func networkRangeOverlaps(a, b *net.IPNet) bool {
	aStart := a.IP.Mask(a.Mask)
	bStart := b.IP.Mask(b.Mask)

	aEnd := lastIP(a)
	bEnd := lastIP(b)

	// a starts before b ends AND b starts before a ends → overlap
	return bytesCompare(aStart, bEnd) <= 0 && bytesCompare(bStart, aEnd) <= 0
}

func lastIP(n *net.IPNet) net.IP {
	ip := n.IP.Mask(n.Mask)
	// Make a copy to avoid mutating original
	ip = append(net.IP(nil), ip...)
	for i := range ip {
		ip[i] |= ^n.Mask[i]
	}
	return ip
}

func bytesCompare(a, b net.IP) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) == len(b) {
		return 0
	}
	if len(a) < len(b) {
		return -1
	}
	return 1
}

func getString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// checkCIDROverlapWithContainers performs overlap detection against a provided set of containers.
// This is separated for unit testing without requiring live API calls.
func checkCIDROverlapWithContainers(newCIDR string, containers []admin.CloudProviderContainer) error {
	_, newNet, err := net.ParseCIDR(newCIDR)
	if err != nil {
		return fmt.Errorf("invalid new CIDR: %w", err)
	}
	for _, c := range containers {
		if c.AtlasCidrBlock == nil || *c.AtlasCidrBlock == "" {
			continue
		}
		_, existingNet, parseErr := net.ParseCIDR(*c.AtlasCidrBlock)
		if parseErr != nil {
			continue
		}
		if cidrOverlaps(newNet, existingNet) {
			return fmt.Errorf("CIDR %s overlaps with existing container %s (%s)", newCIDR, getString(c.Id), *c.AtlasCidrBlock)
		}
	}
	return nil
}

// GetNetworkContainerStatus returns the status of a network container.
func (s *NetworkContainersService) GetNetworkContainerStatus(ctx context.Context, projectID, containerID string) (string, error) {
	container, err := s.GetNetworkContainer(ctx, projectID, containerID)
	if err != nil {
		return "", err
	}

	// Network containers don't typically have a status field like other resources
	// but we can infer the status based on other fields
	if container.Id != nil {
		return "AVAILABLE", nil
	}

	return "UNKNOWN", nil
}
