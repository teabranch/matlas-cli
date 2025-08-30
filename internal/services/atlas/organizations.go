package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

// OrganizationsService provides helpers around Atlas Organizations API.
type OrganizationsService struct {
	client *atlasclient.Client
}

// NewOrganizationsService creates a new OrganizationsService.
func NewOrganizationsService(client *atlasclient.Client) *OrganizationsService {
	return &OrganizationsService{client: client}
}

// List returns organizations visible to the authenticated principal.
func (s *OrganizationsService) List(ctx context.Context) ([]admin.AtlasOrganization, error) {
	var out []admin.AtlasOrganization
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.OrganizationsApi.ListOrganizations(ctx).Execute()
		if err != nil {
			return err
		}
		if resp == nil || resp.Results == nil {
			return fmt.Errorf("%w: empty response", atlasclient.ErrNotFound)
		}
		out = *resp.Results
		return nil
	})
	return out, err
}

// Get fetches an organization by ID.
func (s *OrganizationsService) Get(ctx context.Context, orgID string) (*admin.AtlasOrganization, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgID required")
	}
	var org *admin.AtlasOrganization
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		o, _, err := api.OrganizationsApi.GetOrganization(ctx, orgID).Execute()
		if err != nil {
			return err
		}
		org = o
		return nil
	})
	return org, err
}
