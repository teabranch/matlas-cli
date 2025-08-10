package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// ClustersService wraps Clusters API operations we support (read-only for now).
// Write operations (Create/Update/Delete) are stubbed for future implementation but not exercised in unit tests.
type ClustersService struct {
	client *atlasclient.Client
}

// NewClustersService creates a new ClustersService.
func NewClustersService(client *atlasclient.Client) *ClustersService {
	return &ClustersService{client: client}
}

// List returns all clusters in the specified project.
func (s *ClustersService) List(ctx context.Context, projectID string) ([]admin.ClusterDescription20240805, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	var clusters []admin.ClusterDescription20240805
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ClustersApi.ListClusters(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		if resp == nil || resp.Results == nil {
			return fmt.Errorf("%w: empty response", atlasclient.ErrNotFound)
		}
		clusters = *resp.Results
		return nil
	})
	return clusters, err
}

// Get returns a cluster by name in the specified project.
func (s *ClustersService) Get(ctx context.Context, projectID, clusterName string) (*admin.ClusterDescription20240805, error) {
	if projectID == "" || clusterName == "" {
		return nil, fmt.Errorf("projectID and clusterName are required")
	}
	var cluster *admin.ClusterDescription20240805
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ClustersApi.GetCluster(ctx, projectID, clusterName).Execute()
		if err != nil {
			return err
		}
		cluster = resp
		return nil
	})
	return cluster, err
}

// Create creates a new cluster in the specified project.
func (s *ClustersService) Create(ctx context.Context, projectID string, cluster *admin.ClusterDescription20240805) (*admin.ClusterDescription20240805, error) {
	if projectID == "" || cluster == nil {
		return nil, fmt.Errorf("projectID and cluster are required")
	}

	// Validate required cluster fields
	if cluster.Name == nil || *cluster.Name == "" {
		return nil, fmt.Errorf("cluster name is required")
	}

	var created *admin.ClusterDescription20240805
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ClustersApi.CreateCluster(ctx, projectID, cluster).Execute()
		if err != nil {
			return err
		}
		created = resp
		return nil
	})
	return created, err
}

// Update modifies an existing cluster.
func (s *ClustersService) Update(ctx context.Context, projectID, clusterName string, cluster *admin.ClusterDescription20240805) (*admin.ClusterDescription20240805, error) {
	if projectID == "" || clusterName == "" || cluster == nil {
		return nil, fmt.Errorf("projectID, clusterName, and cluster are required")
	}

	var updated *admin.ClusterDescription20240805
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ClustersApi.UpdateCluster(ctx, projectID, clusterName, cluster).Execute()
		if err != nil {
			return err
		}
		updated = resp
		return nil
	})
	return updated, err
}

// Delete removes a cluster from the specified project.
func (s *ClustersService) Delete(ctx context.Context, projectID, clusterName string) error {
	if projectID == "" || clusterName == "" {
		return fmt.Errorf("projectID and clusterName are required")
	}

	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.ClustersApi.DeleteCluster(ctx, projectID, clusterName).Execute()
		return err
	})
}
