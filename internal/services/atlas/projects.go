package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// ProjectsService provides high-level helpers around the Atlas Projects API.
type ProjectsService struct {
	client *atlasclient.Client
}

// NewProjectsService returns a ProjectsService bound to the provided Client.
func NewProjectsService(client *atlasclient.Client) *ProjectsService {
	return &ProjectsService{client: client}
}

// List returns all projects visible to the authenticated account.
func (s *ProjectsService) List(ctx context.Context) ([]admin.Group, error) {
	var out []admin.Group
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ProjectsApi.ListProjects(ctx).Execute()
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

// ListByOrg returns projects under the specified organization ID.
func (s *ProjectsService) ListByOrg(ctx context.Context, orgID string) ([]admin.Group, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgID is required")
	}
	var out []admin.Group
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.OrganizationsApi.ListOrganizationProjects(ctx, orgID).Execute()
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

// Get fetches a single project by its 24-digit ID.
func (s *ProjectsService) Get(ctx context.Context, projectID string) (*admin.Group, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}
	var result *admin.Group
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		grp, _, err := api.ProjectsApi.GetProject(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		result = grp
		return nil
	})
	return result, err
}

// Create creates a new project under the specified organization. Optional tags can be supplied.
func (s *ProjectsService) Create(ctx context.Context, name, orgID string, tags map[string]string) (*admin.Group, error) {
	if name == "" || orgID == "" {
		return nil, fmt.Errorf("name and orgID are required")
	}
	// Avoid SDK defaults that set RegionUsageRestrictions (rejected in Commercial Atlas)
	newGrp := &admin.Group{
		Name:  name,
		OrgId: orgID,
	}
	if len(tags) > 0 {
		tagSlice := make([]admin.ResourceTag, 0, len(tags))
		for k, v := range tags {
			tagSlice = append(tagSlice, admin.ResourceTag{Key: k, Value: v})
		}
		newGrp.Tags = &tagSlice
	}

	var created *admin.Group
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		grp, _, err := api.ProjectsApi.CreateProject(ctx, newGrp).Execute()
		if err != nil {
			return err
		}
		created = grp
		return nil
	})
	return created, err
}

// Delete removes a project permanently. Atlas requires the project to be empty of clusters.
func (s *ProjectsService) Delete(ctx context.Context, projectID string) error {
	if projectID == "" {
		return fmt.Errorf("projectID is required")
	}
	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.ProjectsApi.DeleteProject(ctx, projectID).Execute()
		return err
	})
}

// Update updates mutable project fields (name, tags).
func (s *ProjectsService) Update(ctx context.Context, projectID string, update admin.GroupUpdate) (*admin.Group, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}
	var updated *admin.Group
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ProjectsApi.UpdateProject(ctx, projectID, &update).Execute()
		if err != nil {
			return err
		}
		updated = resp
		return nil
	})
	return updated, err
}
