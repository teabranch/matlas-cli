package atlas

import (
	"context"
	"fmt"
	"net"
	"strings"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

// NetworkAccessListsService provides CRUD operations for Atlas IP Access Lists (network access control).
type NetworkAccessListsService struct {
	client *atlasclient.Client
}

// NewNetworkAccessListsService creates a new NetworkAccessListsService.
func NewNetworkAccessListsService(client *atlasclient.Client) *NetworkAccessListsService {
	return &NetworkAccessListsService{client: client}
}

// List returns all network access list entries for the specified project.
func (s *NetworkAccessListsService) List(ctx context.Context, projectID string) ([]admin.NetworkPermissionEntry, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	var entries []admin.NetworkPermissionEntry
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ProjectIPAccessListApi.ListProjectIpAccessLists(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		if resp == nil || resp.Results == nil {
			return fmt.Errorf("%w: empty response", atlasclient.ErrNotFound)
		}
		entries = *resp.Results
		return nil
	})
	return entries, err
}

// Get returns a specific network access list entry by IP/CIDR.
func (s *NetworkAccessListsService) Get(ctx context.Context, projectID, ipAddress string) (*admin.NetworkPermissionEntry, error) {
	if projectID == "" || ipAddress == "" {
		return nil, fmt.Errorf("projectID and ipAddress are required")
	}
	var entry *admin.NetworkPermissionEntry
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ProjectIPAccessListApi.GetProjectIpList(ctx, projectID, ipAddress).Execute()
		if err != nil {
			return err
		}
		entry = resp
		return nil
	})
	return entry, err
}

// Create adds a new network access list entry.
func (s *NetworkAccessListsService) Create(ctx context.Context, projectID string, entries []admin.NetworkPermissionEntry) (*admin.PaginatedNetworkAccess, error) {
	if projectID == "" || len(entries) == 0 {
		return nil, fmt.Errorf("projectID and entries are required")
	}

	// Validate IP addresses/CIDR blocks
	for i, entry := range entries {
		if err := s.validateIPEntry(entry); err != nil {
			return nil, fmt.Errorf("entry %d validation failed: %w", i, err)
		}
	}

	var result *admin.PaginatedNetworkAccess
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ProjectIPAccessListApi.CreateProjectIpAccessList(ctx, projectID, &entries).Execute()
		if err != nil {
			return err
		}
		result = resp
		return nil
	})
	return result, err
}

// Delete removes a network access list entry.
func (s *NetworkAccessListsService) Delete(ctx context.Context, projectID, ipAddress string) error {
	if projectID == "" || ipAddress == "" {
		return fmt.Errorf("projectID and ipAddress are required")
	}
	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.ProjectIPAccessListApi.DeleteProjectIpAccessList(ctx, projectID, ipAddress).Execute()
		return err
	})
}

// validateIPEntry validates that the network permission entry has valid IP address or CIDR block.
func (s *NetworkAccessListsService) validateIPEntry(entry admin.NetworkPermissionEntry) error {
	if entry.IpAddress != nil {
		// Single IP address
		if net.ParseIP(*entry.IpAddress) == nil {
			return fmt.Errorf("invalid IP address: %s", *entry.IpAddress)
		}
	} else if entry.CidrBlock != nil {
		// CIDR block
		_, _, err := net.ParseCIDR(*entry.CidrBlock)
		if err != nil {
			return fmt.Errorf("invalid CIDR block: %s", *entry.CidrBlock)
		}
	} else if entry.AwsSecurityGroup != nil {
		// AWS Security Group - just check it's not empty
		if strings.TrimSpace(*entry.AwsSecurityGroup) == "" {
			return fmt.Errorf("AWS security group cannot be empty")
		}
	} else {
		return fmt.Errorf("entry must specify ipAddress, cidrBlock, or awsSecurityGroup")
	}
	return nil
}

// ValidateIP is a helper function to validate IP addresses before creating entries.
func ValidateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	return nil
}

// ValidateCIDR is a helper function to validate CIDR blocks before creating entries.
func ValidateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR block: %s", cidr)
	}
	return nil
}
