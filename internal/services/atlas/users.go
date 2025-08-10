package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// DatabaseUsersService provides CRUD operations for MongoDB database users in Atlas.
type DatabaseUsersService struct {
	client *atlasclient.Client
}

// NewDatabaseUsersService creates a new DatabaseUsersService.
func NewDatabaseUsersService(client *atlasclient.Client) *DatabaseUsersService {
	return &DatabaseUsersService{client: client}
}

// List returns all database users for the specified project.
func (s *DatabaseUsersService) List(ctx context.Context, projectID string) ([]admin.CloudDatabaseUser, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	var users []admin.CloudDatabaseUser
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.DatabaseUsersApi.ListDatabaseUsers(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		if resp == nil || resp.Results == nil {
			return fmt.Errorf("%w: empty response", atlasclient.ErrNotFound)
		}
		users = *resp.Results
		return nil
	})
	return users, err
}

// ListWithPagination returns database users using server-side pagination when available.
// If all is true, it will paginate through all pages and return the full list.
// When all is false, it will request the specific page and limit from the server.
func (s *DatabaseUsersService) ListWithPagination(ctx context.Context, projectID string, page, limit int, all bool) ([]admin.CloudDatabaseUser, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	if !all {
		if page < 1 {
			return nil, fmt.Errorf("page must be >= 1")
		}
		if limit < 1 {
			return nil, fmt.Errorf("limit must be >= 1")
		}
	}

	var aggregated []admin.CloudDatabaseUser

	if all {
		// Iterate pages until fewer than limit results are returned.
		// Use a reasonable default page size when fetching all.
		currentPage := 1
		pageSize := 500 // match cli.MaxPageSize
		for {
			var pageResults []admin.CloudDatabaseUser
			err := s.client.Do(ctx, func(api *admin.APIClient) error {
				req := api.DatabaseUsersApi.ListDatabaseUsers(ctx, projectID)
				// The Atlas SDK exposes pagination setters on requests in most list APIs.
				// We optimistically use them; if not present in a future SDK, compilation will catch it.
				req = req.ItemsPerPage(pageSize).PageNum(currentPage)
				resp, _, err := req.Execute()
				if err != nil {
					return err
				}
				if resp == nil || resp.Results == nil {
					pageResults = nil
					return nil
				}
				pageResults = *resp.Results
				return nil
			})
			if err != nil {
				return nil, err
			}
			if len(pageResults) == 0 {
				break
			}
			aggregated = append(aggregated, pageResults...)
			if len(pageResults) < pageSize {
				break
			}
			currentPage++
		}

		return aggregated, nil
	}

	// Single page
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		req := api.DatabaseUsersApi.ListDatabaseUsers(ctx, projectID)
		req = req.ItemsPerPage(limit).PageNum(page)
		resp, _, err := req.Execute()
		if err != nil {
			return err
		}
		if resp == nil || resp.Results == nil {
			return nil
		}
		aggregated = *resp.Results
		return nil
	})
	return aggregated, err
}

// Get returns a specific database user by username and authentication database.
func (s *DatabaseUsersService) Get(ctx context.Context, projectID, databaseName, username string) (*admin.CloudDatabaseUser, error) {
	if projectID == "" || databaseName == "" || username == "" {
		return nil, fmt.Errorf("projectID, databaseName, and username are required")
	}
	var user *admin.CloudDatabaseUser
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.DatabaseUsersApi.GetDatabaseUser(ctx, projectID, databaseName, username).Execute()
		if err != nil {
			return err
		}
		user = resp
		return nil
	})
	return user, err
}

// Create creates a new database user.
func (s *DatabaseUsersService) Create(ctx context.Context, projectID string, user *admin.CloudDatabaseUser) (*admin.CloudDatabaseUser, error) {
	if projectID == "" || user == nil {
		return nil, fmt.Errorf("projectID and user are required")
	}
	var created *admin.CloudDatabaseUser
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.DatabaseUsersApi.CreateDatabaseUser(ctx, projectID, user).Execute()
		if err != nil {
			return err
		}
		created = resp
		return nil
	})
	return created, err
}

// Update modifies an existing database user.
func (s *DatabaseUsersService) Update(ctx context.Context, projectID, databaseName, username string, user *admin.CloudDatabaseUser) (*admin.CloudDatabaseUser, error) {
	if projectID == "" || databaseName == "" || username == "" || user == nil {
		return nil, fmt.Errorf("projectID, databaseName, username, and user are required")
	}
	var updated *admin.CloudDatabaseUser
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.DatabaseUsersApi.UpdateDatabaseUser(ctx, projectID, databaseName, username, user).Execute()
		if err != nil {
			return err
		}
		updated = resp
		return nil
	})
	return updated, err
}

// Delete removes a database user.
func (s *DatabaseUsersService) Delete(ctx context.Context, projectID, databaseName, username string) error {
	if projectID == "" || databaseName == "" || username == "" {
		return fmt.Errorf("projectID, databaseName, and username are required")
	}
	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.DatabaseUsersApi.DeleteDatabaseUser(ctx, projectID, databaseName, username).Execute()
		return err
	})
}
