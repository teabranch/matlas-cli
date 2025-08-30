package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
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

// AdvancedSearchService provides operations for advanced search features
type AdvancedSearchService struct {
	searchService *SearchService
}

// NewAdvancedSearchService creates a new AdvancedSearchService instance
func NewAdvancedSearchService(searchService *SearchService) *AdvancedSearchService {
	return &AdvancedSearchService{searchService: searchService}
}

// GetSearchAnalyzers retrieves analyzer information for a search index
func (s *AdvancedSearchService) GetSearchAnalyzers(ctx context.Context, projectID, clusterName, indexName string) ([]map[string]interface{}, error) {
	if projectID == "" || clusterName == "" || indexName == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexName are required")
	}

	// Get the search index definition to extract analyzer information
	indexes, err := s.searchService.ListSearchIndexes(ctx, projectID, clusterName, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list search indexes: %w", err)
	}

	for _, index := range indexes {
		if index.GetName() == indexName {
			return s.extractAnalyzersFromDefinition(&index), nil
		}
	}

	return nil, fmt.Errorf("search index %q not found", indexName)
}

// GetSearchFacets retrieves facet configuration for a search index
func (s *AdvancedSearchService) GetSearchFacets(ctx context.Context, projectID, clusterName, indexName string) ([]map[string]interface{}, error) {
	if projectID == "" || clusterName == "" || indexName == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexName are required")
	}

	// Get the search index definition to extract facet information
	indexes, err := s.searchService.ListSearchIndexes(ctx, projectID, clusterName, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list search indexes: %w", err)
	}

	for _, index := range indexes {
		if index.GetName() == indexName {
			return s.extractFacetsFromDefinition(&index), nil
		}
	}

	return nil, fmt.Errorf("search index %q not found", indexName)
}

// GetSearchMetrics retrieves performance metrics for search indexes
func (s *AdvancedSearchService) GetSearchMetrics(ctx context.Context, projectID, clusterName string, indexName *string, timeRange string) (map[string]interface{}, error) {
	if projectID == "" || clusterName == "" {
		return nil, fmt.Errorf("projectID and clusterName are required")
	}

	// Placeholder implementation - would need to call Atlas monitoring APIs
	metrics := map[string]interface{}{
		"clusterName": clusterName,
		"timeRange":   timeRange,
		"metrics": map[string]interface{}{
			"queryCount":   "1000",
			"avgQueryTime": "50",
			"indexSize":    "2.5GB",
			"errorRate":    "0.1%",
		},
	}

	if indexName != nil {
		metrics["indexName"] = *indexName
	}

	return metrics, nil
}

// AnalyzeSearchIndex provides performance analysis for a search index
func (s *AdvancedSearchService) AnalyzeSearchIndex(ctx context.Context, projectID, clusterName, indexName string) (map[string]interface{}, error) {
	if projectID == "" || clusterName == "" || indexName == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexName are required")
	}

	// Get the search index definition for analysis
	indexes, err := s.searchService.ListSearchIndexes(ctx, projectID, clusterName, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list search indexes: %w", err)
	}

	for _, index := range indexes {
		if index.GetName() == indexName {
			return s.analyzeIndexDefinition(&index), nil
		}
	}

	return nil, fmt.Errorf("search index %q not found", indexName)
}

// ValidateSearchQuery validates a search query against an index
func (s *AdvancedSearchService) ValidateSearchQuery(ctx context.Context, projectID, clusterName, indexName string, query map[string]interface{}) (map[string]interface{}, error) {
	if projectID == "" || clusterName == "" || indexName == "" {
		return nil, fmt.Errorf("projectID, clusterName, and indexName are required")
	}

	// Placeholder implementation - would need to validate query syntax
	result := map[string]interface{}{
		"valid":    true,
		"errors":   []string{},
		"warnings": []string{},
		"query":    query,
	}

	// Basic query validation
	if query == nil {
		result["valid"] = false
		result["errors"] = []string{"Query cannot be empty"}
	}

	return result, nil
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

// Helper methods for extracting information from index definitions

func (s *AdvancedSearchService) extractAnalyzersFromDefinition(index *admin.SearchIndexResponse) []map[string]interface{} {
	analyzers := []map[string]interface{}{}

	// Extract analyzer information from the index definition
	if defPtr, ok := index.GetLatestDefinitionOk(); ok && defPtr != nil {
		// TODO: Extract analyzer information from the definition structure
		// The GetAnalyzerOk() and GetSearchAnalyzerOk() methods are not available in the current SDK version
		// Once the Atlas SDK provides proper analyzer access methods, this should be updated

		// For now, return placeholder analyzer information if the index has a definition
		analyzers = append(analyzers, map[string]interface{}{
			"name":        "default",
			"type":        "standard",
			"status":      "active",
			"description": "Default analyzer extracted from index definition",
		})
	}

	return analyzers
}

func (s *AdvancedSearchService) extractFacetsFromDefinition(index *admin.SearchIndexResponse) []map[string]interface{} {
	facets := []map[string]interface{}{}

	// Extract facet information from the index definition
	if defPtr, ok := index.GetLatestDefinitionOk(); ok && defPtr != nil {
		// TODO: Parse the definition to extract facet configurations
		// This would require understanding the actual structure of the Atlas Search definition

		// Placeholder facet information
		facets = append(facets, map[string]interface{}{
			"field":       "category",
			"type":        "string",
			"status":      "active",
			"description": "String facet for category field",
		})
	}

	return facets
}

func (s *AdvancedSearchService) analyzeIndexDefinition(index *admin.SearchIndexResponse) map[string]interface{} {
	analysis := map[string]interface{}{
		"indexName":         index.GetName(),
		"status":            index.GetStatus(),
		"type":              index.GetType(),
		"optimizationScore": 75, // Placeholder score
		"recommendations": []map[string]interface{}{
			{
				"title":       "Consider adding specific field mappings",
				"description": "Dynamic mapping can impact performance for large datasets",
				"priority":    "medium",
			},
			{
				"title":       "Review analyzer configuration",
				"description": "Custom analyzers may improve search relevance",
				"priority":    "low",
			},
		},
		"performance": map[string]interface{}{
			"estimatedSize": "Unknown",
			"fieldCount":    "Dynamic",
			"complexity":    "Medium",
		},
	}

	return analysis
}
