package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/logging"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

// ProjectsService provides high-level helpers around the Atlas Projects API.
type ProjectsService struct {
	client *atlasclient.Client
	logger *logging.Logger
}

// NewProjectsService returns a ProjectsService bound to the provided Client.
func NewProjectsService(client *atlasclient.Client) *ProjectsService {
	return &ProjectsService{
		client: client,
		logger: logging.Default(),
	}
}

// NewProjectsServiceWithLogger returns a ProjectsService with a custom logger.
func NewProjectsServiceWithLogger(client *atlasclient.Client, logger *logging.Logger) *ProjectsService {
	if logger == nil {
		logger = logging.Default()
	}
	return &ProjectsService{
		client: client,
		logger: logger,
	}
}

// List returns all projects visible to the authenticated account.
func (s *ProjectsService) List(ctx context.Context) ([]admin.Group, error) {
	s.logger.Debug("Listing all projects")

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

	if err != nil {
		s.logger.Error("Failed to list projects", "error", err.Error())
		return nil, err
	}

	s.logger.Debug("Listed projects successfully", "count", len(out))
	return out, nil
}

// ListByOrg returns projects under the specified organization ID.
func (s *ProjectsService) ListByOrg(ctx context.Context, orgID string) ([]admin.Group, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgID is required")
	}

	s.logger.Debug("Listing projects by organization", "org_id", orgID)

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

	if err != nil {
		s.logger.Error("Failed to list projects by organization", "org_id", orgID, "error", err.Error())
		return nil, err
	}

	s.logger.Debug("Listed projects by organization successfully", "org_id", orgID, "count", len(out))
	return out, nil
}

// Get fetches a single project by its 24-digit ID.
func (s *ProjectsService) Get(ctx context.Context, projectID string) (*admin.Group, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}

	s.logger.Debug("Getting project", "project_id", projectID)

	var result *admin.Group
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		grp, _, err := api.ProjectsApi.GetProject(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		result = grp
		return nil
	})

	if err != nil {
		s.logger.Error("Failed to get project", "project_id", projectID, "error", err.Error())
		return nil, err
	}

	s.logger.Debug("Got project successfully", "project_id", projectID)
	return result, nil
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
