package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/logging"
	admin "go.mongodb.org/atlas-sdk/v20250312010/admin"
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
			result, _, err := api.AtlasSearchApi.ListSearchIndex(ctx, projectID, clusterName, *collectionName, *databaseName).Execute()
			if err != nil {
				return err
			}
			indexes = result
		} else {
			// List all indexes for the cluster
			result, _, err := api.AtlasSearchApi.ListClusterSearchIndexes(ctx, projectID, clusterName).Execute()
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
// NOTE: This method uses the FTS API and returns ClusterSearchIndex type.
func (s *SearchService) GetSearchIndex(ctx context.Context, projectID, clusterName, indexID string) (*admin.ClusterSearchIndex, error) {
	if projectID == "" || clusterName == "" || indexID == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexID are required")
	}

	var index *admin.ClusterSearchIndex
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.AtlasSearchApi.GetClusterFtsIndex(ctx, projectID, clusterName, indexID).Execute()
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
		result, _, err := api.AtlasSearchApi.GetIndexByName(ctx, projectID, clusterName, collectionName, databaseName, indexName).Execute()
		if err != nil {
			return err
		}
		index = result
		return nil
	})
	return index, err
}

// CreateSearchIndex creates a new search index.
// NOTE: This method uses the FTS API and works with ClusterSearchIndex type.
func (s *SearchService) CreateSearchIndex(ctx context.Context, projectID, clusterName string, indexRequest admin.ClusterSearchIndex) (*admin.ClusterSearchIndex, error) {
	if projectID == "" || clusterName == "" {
		return nil, fmt.Errorf("projectID and clusterName are required")
	}

	// Validate the index request
	if err := s.validateClusterSearchIndexRequest(&indexRequest); err != nil {
		return nil, fmt.Errorf("index validation failed: %w", err)
	}

	var index *admin.ClusterSearchIndex
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.AtlasSearchApi.CreateClusterFtsIndex(ctx, projectID, clusterName, &indexRequest).Execute()
		if err != nil {
			return err
		}
		index = result
		return nil
	})
	return index, err
}

// UpdateSearchIndex updates an existing search index.
// NOTE: This method uses the FTS API and works with ClusterSearchIndex type.
func (s *SearchService) UpdateSearchIndex(ctx context.Context, projectID, clusterName, indexID string, updateRequest admin.ClusterSearchIndex) (*admin.ClusterSearchIndex, error) {
	if projectID == "" || clusterName == "" || indexID == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexID are required")
	}

	// Validate the update request
	if err := s.validateClusterSearchIndexRequest(&updateRequest); err != nil {
		return nil, fmt.Errorf("index validation failed: %w", err)
	}

	var index *admin.ClusterSearchIndex
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		result, _, err := api.AtlasSearchApi.UpdateClusterFtsIndex(ctx, projectID, clusterName, indexID, &updateRequest).Execute()
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
		_, err := api.AtlasSearchApi.DeleteClusterFtsIndex(ctx, projectID, clusterName, indexID).Execute()
		return err
	})
}

// DeleteSearchIndexByName removes a search index by name.
func (s *SearchService) DeleteSearchIndexByName(ctx context.Context, projectID, clusterName, databaseName, collectionName, indexName string) error {
	if projectID == "" || clusterName == "" || databaseName == "" || collectionName == "" || indexName == "" {
		return fmt.Errorf("all parameters are required")
	}

	return s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.AtlasSearchApi.DeleteIndexByName(ctx, projectID, clusterName, databaseName, collectionName, indexName).Execute()
		return err
	})
}

// validateClusterSearchIndexRequest validates the ClusterSearchIndex request.
func (s *SearchService) validateClusterSearchIndexRequest(request *admin.ClusterSearchIndex) error {
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

// validateSearchIndexCreateRequest validates the search index create request.
// Deprecated: Use validateClusterSearchIndexRequest instead.
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
// Deprecated: Use validateClusterSearchIndexRequest instead.
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
			indexes, _, err := api.AtlasSearchApi.ListClusterSearchIndexes(ctx, projectID, clusterName).Execute()
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

// ConvertClusterSearchIndexToResponse converts ClusterSearchIndex to SearchIndexResponse.
// This is needed for backward compatibility with formatters and existing code.
func ConvertClusterSearchIndexToResponse(idx *admin.ClusterSearchIndex) admin.SearchIndexResponse {
	if idx == nil {
		return admin.SearchIndexResponse{}
	}

	resp := admin.NewSearchIndexResponse()

	// Copy basic fields using GetOk methods
	if collName, ok := idx.GetCollectionNameOk(); ok && collName != nil {
		resp.SetCollectionName(*collName)
	}
	if db, ok := idx.GetDatabaseOk(); ok && db != nil {
		resp.SetDatabase(*db)
	}
	if name, ok := idx.GetNameOk(); ok && name != nil {
		resp.SetName(*name)
	}
	if indexID, ok := idx.GetIndexIDOk(); ok && indexID != nil {
		resp.SetIndexID(*indexID)
	}
	if status, ok := idx.GetStatusOk(); ok && status != nil {
		resp.SetStatus(*status)
	}
	if indexType, ok := idx.GetTypeOk(); ok && indexType != nil {
		resp.SetType(*indexType)
	}

	return *resp
}

// ConvertSearchIndexCreateRequestToClusterSearchIndex converts SearchIndexCreateRequest to ClusterSearchIndex.
func ConvertSearchIndexCreateRequestToClusterSearchIndex(req *admin.SearchIndexCreateRequest) *admin.ClusterSearchIndex {
	if req == nil {
		return nil
	}

	idx := admin.NewClusterSearchIndex(req.GetCollectionName(), req.GetDatabase(), req.GetName())

	if req.HasType() {
		idx.SetType(req.GetType())
	}

	// Note: Definition field structure may differ between types
	// Additional field mapping would be needed for full conversion

	return idx
}

// AdvancedSearchService provides operations for advanced search features
type AdvancedSearchService struct {
	searchService *SearchService
	logger        *logging.Logger
}

// NewAdvancedSearchService creates a new AdvancedSearchService instance
func NewAdvancedSearchService(searchService *SearchService) *AdvancedSearchService {
	return &AdvancedSearchService{
		searchService: searchService,
		logger:        logging.Default(),
	}
}

// GetSearchAnalyzers retrieves analyzer information for a search index
func (s *AdvancedSearchService) GetSearchAnalyzers(ctx context.Context, projectID, clusterName, indexName string) ([]map[string]interface{}, error) {
	if projectID == "" || clusterName == "" || indexName == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexName are required")
	}

	// Log the limitation
	s.logger.Error("Search analyzer extraction operation not supported by Atlas Admin API",
		"project_id", projectID,
		"cluster_name", clusterName,
		"index_name", indexName,
		"reason", "Atlas SDK does not expose analyzer details from index definitions")

	return nil, fmt.Errorf("search analyzer extraction operation not supported: Atlas SDK does not expose analyzer details from index definitions. For analyzer configuration, use the Atlas UI at https://cloud.mongodb.com")
}

// GetSearchFacets retrieves facet configuration for a search index
func (s *AdvancedSearchService) GetSearchFacets(ctx context.Context, projectID, clusterName, indexName string) ([]map[string]interface{}, error) {
	if projectID == "" || clusterName == "" || indexName == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexName are required")
	}

	// Log the limitation
	s.logger.Error("Search facet extraction operation not supported by Atlas Admin API",
		"project_id", projectID,
		"cluster_name", clusterName,
		"index_name", indexName,
		"reason", "Atlas SDK does not expose facet details from index definitions")

	return nil, fmt.Errorf("search facet extraction operation not supported: Atlas SDK does not expose facet details from index definitions. For facet configuration, use the Atlas UI at https://cloud.mongodb.com")
}

// GetSearchMetrics retrieves performance metrics for search indexes
func (s *AdvancedSearchService) GetSearchMetrics(ctx context.Context, projectID, clusterName string, indexName *string, timeRange string) (map[string]interface{}, error) {
	if projectID == "" || clusterName == "" {
		return nil, fmt.Errorf("projectID and clusterName are required")
	}

	// Log the limitation
	s.logger.Error("Search metrics operation not supported by Atlas Admin API",
		"project_id", projectID,
		"cluster_name", clusterName,
		"reason", "Atlas Admin API does not expose real-time metrics endpoints")

	return nil, fmt.Errorf("search metrics operation not supported: Atlas Admin API does not provide real-time metrics endpoints. For real metrics, use the Atlas UI at https://cloud.mongodb.com")
}

// AnalyzeSearchIndex provides performance analysis for a search index
func (s *AdvancedSearchService) AnalyzeSearchIndex(ctx context.Context, projectID, clusterName, indexName string) (map[string]interface{}, error) {
	if projectID == "" || clusterName == "" || indexName == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexName are required")
	}

	// Log the limitation
	s.logger.Error("Search index analysis operation not supported by Atlas Admin API",
		"project_id", projectID,
		"cluster_name", clusterName,
		"index_name", indexName,
		"reason", "Atlas Admin API does not provide index optimization analysis")

	return nil, fmt.Errorf("search index analysis operation not supported: Atlas Admin API does not provide optimization analysis endpoints. For real optimization insights, use Atlas Performance Advisor in the Atlas UI")
}

// ValidateSearchQuery validates a search query against an index
func (s *AdvancedSearchService) ValidateSearchQuery(ctx context.Context, projectID, clusterName, indexName string, query map[string]interface{}) (map[string]interface{}, error) {
	if projectID == "" || clusterName == "" || indexName == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexName are required")
	}

	// Log the limitation
	s.logger.Error("Search query validation operation not supported by Atlas Admin API",
		"project_id", projectID,
		"cluster_name", clusterName,
		"index_name", indexName,
		"reason", "Atlas Admin API does not provide query validation endpoints")

	return nil, fmt.Errorf("search query validation operation not supported: Atlas Admin API does not provide query validation endpoints. For real query validation, test queries directly in Atlas UI or MongoDB Compass")
}

// ValidateSearchIndex validates a search index configuration
func (s *AdvancedSearchService) ValidateSearchIndex(ctx context.Context, projectID, clusterName string, indexConfig map[string]interface{}) (map[string]interface{}, error) {
	if projectID == "" || clusterName == "" {
		return nil, fmt.Errorf("projectID and clusterName are required")
	}

	result := map[string]interface{}{
		"valid":    true,
		"errors":   []string{},
		"warnings": []string{},
		"config":   indexConfig,
	}

	// Basic configuration validation
	if indexConfig == nil {
		result["valid"] = false
		result["errors"] = []string{"Index configuration cannot be empty"}
		return result, nil
	}

	// Validate required fields
	if _, ok := indexConfig["mappings"]; !ok {
		if _, ok := indexConfig["fields"]; !ok {
			result["valid"] = false
			result["errors"] = append(result["errors"].([]string), "Index must have either mappings or fields defined")
		}
	}

	return result, nil
}

// All helper methods that returned placeholder data have been removed.
// Advanced search features that are not supported by the Atlas Admin API now return proper errors.
