package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// SearchService provides CRUD operations for Atlas Search indexes.
// This service manages Atlas Search indexes for full-text and vector search.
type SearchService struct {
	client *atlasclient.Client
}

// NewSearchService creates a new SearchService instance.
func NewSearchService(client *atlasclient.Client) *SearchService {
	return &SearchService{client: client}
}

// ListSearchIndexes returns all search indexes for the specified cluster or collection.
func (s *SearchService) ListSearchIndexes(ctx context.Context, projectID, clusterName string, databaseName, collectionName *string) ([]admin.SearchIndexResponse, error) {
	if projectID == "" || clusterName == "" {
		return nil, fmt.Errorf("projectID and clusterName are required")
	}

	var indexes []admin.SearchIndexResponse
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		if databaseName != nil && collectionName != nil {
			// List indexes for a specific collection
			result, _, err := api.AtlasSearchApi.ListAtlasSearchIndexes(ctx, projectID, clusterName, *collectionName, *databaseName).Execute()
			if err != nil {
				return err
			}
			indexes = result
		} else {
			// List all indexes for the cluster
			result, _, err := api.AtlasSearchApi.ListAtlasSearchIndexesCluster(ctx, projectID, clusterName).Execute()
			if err != nil {
				return err
			}
			indexes = result
		}
		return nil
	})
	return indexes, err
}

// GetSearchIndex returns a specific search index by ID.
func (s *SearchService) GetSearchIndex(ctx context.Context, projectID, clusterName, indexID string) (*admin.SearchIndexResponse, error) {
	if projectID == "" || clusterName == "" || indexID == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexID are required")
	}

	var index *admin.SearchIndexResponse
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.AtlasSearchApi.GetAtlasSearchIndex(ctx, projectID, clusterName, indexID).Execute()
		if err != nil {
			return err
		}
		index = result
		return nil
	})
	return index, err
}

// GetSearchIndexByName returns a specific search index by name.
func (s *SearchService) GetSearchIndexByName(ctx context.Context, projectID, clusterName, databaseName, collectionName, indexName string) (*admin.SearchIndexResponse, error) {
	if projectID == "" || clusterName == "" || databaseName == "" || collectionName == "" || indexName == "" {
		return nil, fmt.Errorf("all parameters are required")
	}

	var index *admin.SearchIndexResponse
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.AtlasSearchApi.GetAtlasSearchIndexByName(ctx, projectID, clusterName, collectionName, databaseName, indexName).Execute()
		if err != nil {
			return err
		}
		index = result
		return nil
	})
	return index, err
}

// CreateSearchIndex creates a new search index.
func (s *SearchService) CreateSearchIndex(ctx context.Context, projectID, clusterName string, indexRequest admin.SearchIndexCreateRequest) (*admin.SearchIndexResponse, error) {
	if projectID == "" || clusterName == "" {
		return nil, fmt.Errorf("projectID and clusterName are required")
	}

	// Validate the index request
	if err := s.validateSearchIndexCreateRequest(&indexRequest); err != nil {
		return nil, fmt.Errorf("index validation failed: %w", err)
	}

	var index *admin.SearchIndexResponse
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.AtlasSearchApi.CreateAtlasSearchIndex(ctx, projectID, clusterName, &indexRequest).Execute()
		if err != nil {
			return err
		}
		index = result
		return nil
	})
	return index, err
}

// UpdateSearchIndex updates an existing search index.
func (s *SearchService) UpdateSearchIndex(ctx context.Context, projectID, clusterName, indexID string, updateRequest admin.SearchIndexUpdateRequest) (*admin.SearchIndexResponse, error) {
	if projectID == "" || clusterName == "" || indexID == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexID are required")
	}

	// Validate the update request
	if err := s.validateSearchIndexUpdateRequest(&updateRequest); err != nil {
		return nil, fmt.Errorf("index validation failed: %w", err)
	}

	var index *admin.SearchIndexResponse
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.AtlasSearchApi.UpdateAtlasSearchIndex(ctx, projectID, clusterName, indexID, &updateRequest).Execute()
		if err != nil {
			return err
		}
		index = result
		return nil
	})
	return index, err
}

// DeleteSearchIndex removes a search index.
func (s *SearchService) DeleteSearchIndex(ctx context.Context, projectID, clusterName, indexID string) error {
	if projectID == "" || clusterName == "" || indexID == "" {
		return fmt.Errorf("projectID, clusterName, and indexID are required")
	}

	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.AtlasSearchApi.DeleteAtlasSearchIndex(ctx, projectID, clusterName, indexID).Execute()
		return err
	})
}

// DeleteSearchIndexByName removes a search index by name.
func (s *SearchService) DeleteSearchIndexByName(ctx context.Context, projectID, clusterName, databaseName, collectionName, indexName string) error {
	if projectID == "" || clusterName == "" || databaseName == "" || collectionName == "" || indexName == "" {
		return fmt.Errorf("all parameters are required")
	}

	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.AtlasSearchApi.DeleteAtlasSearchIndexByName(ctx, projectID, clusterName, databaseName, collectionName, indexName).Execute()
		return err
	})
}

// validateSearchIndexCreateRequest validates the search index create request.
func (s *SearchService) validateSearchIndexCreateRequest(request *admin.SearchIndexCreateRequest) error {
	if request == nil {
		return fmt.Errorf("index request is required")
	}

	if request.GetCollectionName() == "" {
		return fmt.Errorf("collection name is required")
	}

	if request.GetDatabase() == "" {
		return fmt.Errorf("database name is required")
	}

	if request.GetName() == "" {
		return fmt.Errorf("index name is required")
	}

	return nil
}

// validateSearchIndexUpdateRequest validates the search index update request.
func (s *SearchService) validateSearchIndexUpdateRequest(request *admin.SearchIndexUpdateRequest) error {
	if request == nil {
		return fmt.Errorf("update request is required")
	}

	// Additional validation can be added here for specific update fields
	// when the SDK provides more detailed structure

	return nil
}

// ListAllIndexes returns all search indexes across all clusters in a project
func (s *SearchService) ListAllIndexes(ctx context.Context, projectID string) ([]admin.SearchIndexResponse, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}

	// First, get all clusters in the project
	var allIndexes []admin.SearchIndexResponse
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		// Get all clusters for this project
		clusters, _, err := api.ClustersApi.ListClusters(ctx, projectID).Execute()
		if err != nil {
			return fmt.Errorf("failed to list clusters: %w", err)
		}

		// For each cluster, get all search indexes
		for _, cluster := range clusters.GetResults() {
			clusterName := cluster.GetName()
			indexes, _, err := api.AtlasSearchApi.ListAtlasSearchIndexesCluster(ctx, projectID, clusterName).Execute()
			if err != nil {
				// Log error but continue with other clusters
				fmt.Printf("Warning: failed to list search indexes for cluster %s: %v\n", clusterName, err)
				continue
			}
			allIndexes = append(allIndexes, indexes...)
		}
		return nil
	})

	return allIndexes, err
}

// stringValue safely returns the string value or empty string if nil
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
