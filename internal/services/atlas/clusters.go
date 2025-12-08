package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/logging"
	admin "go.mongodb.org/atlas-sdk/v20250312010/admin"
)

// ClustersService wraps Clusters API operations we support (read-only for now).
// Write operations (Create/Update/Delete) are stubbed for future implementation but not exercised in unit tests.
type ClustersService struct {
	client *atlasclient.Client
	logger *logging.Logger
}

// NewClustersService creates a new ClustersService.
func NewClustersService(client *atlasclient.Client) *ClustersService {
	return &ClustersService{
		client: client,
		logger: logging.Default(),
	}
}

// NewClustersServiceWithLogger creates a new ClustersService with a custom logger.
func NewClustersServiceWithLogger(client *atlasclient.Client, logger *logging.Logger) *ClustersService {
	if logger == nil {
		logger = logging.Default()
	}
	return &ClustersService{
		client: client,
		logger: logger,
	}
}

// List returns all clusters in the specified project.
func (s *ClustersService) List(ctx context.Context, projectID string) ([]admin.ClusterDescription20240805, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}

	s.logger.Debug("Listing clusters", "project_id", projectID)

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

	if err != nil {
		s.logger.Error("Failed to list clusters", "project_id", projectID, "error", err.Error())
		return nil, err
	}

	s.logger.Debug("Listed clusters successfully", "project_id", projectID, "count", len(clusters))
	return clusters, nil
}

// Get returns a cluster by name in the specified project.
func (s *ClustersService) Get(ctx context.Context, projectID, clusterName string) (*admin.ClusterDescription20240805, error) {
	if projectID == "" || clusterName == "" {
		return nil, fmt.Errorf("projectID and clusterName are required")
	}

	s.logger.Debug("Getting cluster", "project_id", projectID, "cluster_name", clusterName)

	var cluster *admin.ClusterDescription20240805
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ClustersApi.GetCluster(ctx, projectID, clusterName).Execute()
		if err != nil {
			return err
		}
		cluster = resp
		return nil
	})

	if err != nil {
		s.logger.Error("Failed to get cluster", "project_id", projectID, "cluster_name", clusterName, "error", err.Error())
		return nil, err
	}

	s.logger.Debug("Got cluster successfully", "project_id", projectID, "cluster_name", clusterName)
	return cluster, nil
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

	clusterName := *cluster.Name
	s.logger.Info("Creating cluster", "project_id", projectID, "cluster_name", clusterName)

	var created *admin.ClusterDescription20240805
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ClustersApi.CreateCluster(ctx, projectID, cluster).Execute()
		if err != nil {
			return err
		}
		created = resp
		return nil
	})

	if err != nil {
		s.logger.Error("Failed to create cluster", "project_id", projectID, "cluster_name", clusterName, "error", err.Error())
		return nil, err
	}

	s.logger.Info("Created cluster successfully", "project_id", projectID, "cluster_name", clusterName)
	return created, nil
}

// Update modifies an existing cluster.
func (s *ClustersService) Update(ctx context.Context, projectID, clusterName string, cluster *admin.ClusterDescription20240805) (*admin.ClusterDescription20240805, error) {
	if projectID == "" || clusterName == "" || cluster == nil {
		return nil, fmt.Errorf("projectID, clusterName, and cluster are required")
	}

	s.logger.Info("Updating cluster", "project_id", projectID, "cluster_name", clusterName)

	var updated *admin.ClusterDescription20240805
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.ClustersApi.UpdateCluster(ctx, projectID, clusterName, cluster).Execute()
		if err != nil {
			return err
		}
		updated = resp
		return nil
	})

	if err != nil {
		s.logger.Error("Failed to update cluster", "project_id", projectID, "cluster_name", clusterName, "error", err.Error())
		return nil, err
	}

	s.logger.Info("Updated cluster successfully", "project_id", projectID, "cluster_name", clusterName)
	return updated, nil
}

// Delete removes a cluster from the specified project.
func (s *ClustersService) Delete(ctx context.Context, projectID, clusterName string) error {
	if projectID == "" || clusterName == "" {
		return fmt.Errorf("projectID and clusterName are required")
	}

	s.logger.Info("Deleting cluster", "project_id", projectID, "cluster_name", clusterName)

	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.ClustersApi.DeleteCluster(ctx, projectID, clusterName).Execute()
		return err
	})

	if err != nil {
		s.logger.Error("Failed to delete cluster", "project_id", projectID, "cluster_name", clusterName, "error", err.Error())
		return err
	}

	s.logger.Info("Deleted cluster successfully", "project_id", projectID, "cluster_name", clusterName)
	return nil
}
