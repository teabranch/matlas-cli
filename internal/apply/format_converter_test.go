package apply

import (
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestNewDiscoveredProjectConverter(t *testing.T) {
	converter := NewDiscoveredProjectConverter()
	if converter == nil {
		t.Fatal("NewDiscoveredProjectConverter returned nil")
	}
}

func TestDiscoveredProjectConverter_ConvertToApplyDocument(t *testing.T) {
	converter := NewDiscoveredProjectConverter()

	tests := []struct {
		name        string
		discovered  interface{}
		expectError bool
		description string
	}{
		{
			name:        "Valid DiscoveredProject",
			discovered:  createValidDiscoveredProject(),
			expectError: false,
			description: "Should convert valid discovered project successfully",
		},
		{
			name:        "Invalid Input Type",
			discovered:  "invalid-string",
			expectError: true,
			description: "Should return error for non-map input",
		},
		{
			name:        "Missing Project Info",
			discovered:  map[string]interface{}{},
			expectError: true,
			description: "Should return error for missing project information",
		},
		{
			name:        "Valid Project With Clusters",
			discovered:  createDiscoveredProjectWithClusters(),
			expectError: false,
			description: "Should convert project with clusters successfully",
		},
		{
			name:        "Valid Project With Users",
			discovered:  createDiscoveredProjectWithUsers(),
			expectError: false,
			description: "Should convert project with database users successfully",
		},
		{
			name:        "Valid Project With Network Access",
			discovered:  createDiscoveredProjectWithNetworkAccess(),
			expectError: false,
			description: "Should convert project with network access successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ConvertToApplyDocument(tt.discovered)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			// Validate basic structure
			if result.APIVersion != "matlas.mongodb.com/v1" {
				t.Errorf("Expected APIVersion 'matlas.mongodb.com/v1', got '%s'", result.APIVersion)
			}

			if result.Kind != "ApplyDocument" {
				t.Errorf("Expected Kind 'ApplyDocument', got '%s'", result.Kind)
			}

			if result.Metadata.Name == "" {
				t.Error("Expected metadata name to be set")
			}

						// Only require resources for test cases that actually have them
			if tt.name != "Valid DiscoveredProject" && len(result.Resources) == 0 {
				t.Errorf("Expected at least one resource in the result for test %s", tt.name)
			}
		})
	}
}

func TestDiscoveredProjectConverter_ConvertClusters(t *testing.T) {
	converter := NewDiscoveredProjectConverter()

	discoveredProject := createDiscoveredProjectWithClusters()
	result, err := converter.ConvertToApplyDocument(discoveredProject)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Find cluster resources
	clusterResources := 0
	for _, resource := range result.Resources {
		if resource.Kind == types.KindCluster {
			clusterResources++

			// Validate cluster resource structure
			if resource.APIVersion == "" {
				t.Error("Cluster resource missing APIVersion")
			}

			if resource.Metadata.Name == "" {
				t.Error("Cluster resource missing metadata name")
			}

			// Validate spec conversion
			if resource.Spec == nil {
				t.Error("Cluster resource missing spec")
			}
		}
	}

	if clusterResources == 0 {
		t.Error("Expected at least one cluster resource")
	}
}

func TestDiscoveredProjectConverter_ConvertUsers(t *testing.T) {
	converter := NewDiscoveredProjectConverter()

	discoveredProject := createDiscoveredProjectWithUsers()
	result, err := converter.ConvertToApplyDocument(discoveredProject)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Find database user resources
	userResources := 0
	for _, resource := range result.Resources {
		if resource.Kind == types.KindDatabaseUser {
			userResources++

			// Validate user resource structure
			if resource.APIVersion == "" {
				t.Error("Database user resource missing APIVersion")
			}

			if resource.Metadata.Name == "" {
				t.Error("Database user resource missing metadata name")
			}

			// Validate spec conversion
			if resource.Spec == nil {
				t.Error("Database user resource missing spec")
			}
		}
	}

	if userResources == 0 {
		t.Error("Expected at least one database user resource")
	}
}

func TestDiscoveredProjectConverter_ConvertNetworkAccess(t *testing.T) {
	converter := NewDiscoveredProjectConverter()

	discoveredProject := createDiscoveredProjectWithNetworkAccess()
	result, err := converter.ConvertToApplyDocument(discoveredProject)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Find network access resources
	networkResources := 0
	for _, resource := range result.Resources {
		if resource.Kind == types.KindNetworkAccess {
			networkResources++

			// Validate network access resource structure
			if resource.APIVersion == "" {
				t.Error("Network access resource missing APIVersion")
			}

			if resource.Metadata.Name == "" {
				t.Error("Network access resource missing metadata name")
			}

			// Validate spec conversion
			if resource.Spec == nil {
				t.Error("Network access resource missing spec")
			}
		}
	}

	if networkResources == 0 {
		t.Error("Expected at least one network access resource")
	}
}

// Helper functions for creating test data

func createValidDiscoveredProject() map[string]interface{} {
	return map[string]interface{}{
		"kind": "DiscoveredProject",
		"metadata": map[string]interface{}{
			"projectId": "test-project-id",
			"name":      "test-project",
			"orgId":     "test-org-id",
		},
		"projectInfo": map[string]interface{}{
			"id":           "test-project-id",
			"name":         "test-project",
			"orgId":        "test-org-id",
			"clusterCount": 1,
			"created":      time.Now().Format(time.RFC3339),
			"lastModified": time.Now().Format(time.RFC3339),
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func createDiscoveredProjectWithClusters() map[string]interface{} {
	return map[string]interface{}{
		"kind": "DiscoveredProject",
		"metadata": map[string]interface{}{
			"projectId": "test-project-id",
			"name":      "test-project",
		},
		"clusters": []interface{}{
			map[string]interface{}{
				"apiVersion": "matlas.mongodb.com/v1",
				"kind":       "Cluster",
				"metadata": map[string]interface{}{
					"name": "test-cluster",
				},
				"spec": map[string]interface{}{
					"name":           "test-cluster",
					"provider":       "AWS",
					"region":         "US_EAST_1",
					"instanceSize":   "M10",
					"diskSizeGB":     20,
					"mongodbVersion": "7.0",
					"clusterType":    "REPLICASET",
				},
			},
		},
	}
}

func createDiscoveredProjectWithUsers() map[string]interface{} {
	return map[string]interface{}{
		"kind": "DiscoveredProject",
		"metadata": map[string]interface{}{
			"projectId": "test-project-id",
			"name":      "test-project",
		},
		"databaseUsers": []interface{}{
			map[string]interface{}{
				"apiVersion": "matlas.mongodb.com/v1",
				"kind":       "DatabaseUser",
				"metadata": map[string]interface{}{
					"name": "test-user",
				},
				"spec": map[string]interface{}{
					"username":     "test-user",
					"authDatabase": "admin",
					"scopes": []map[string]interface{}{
						{
							"name": "test-cluster",
							"type": "CLUSTER",
						},
					},
					"roles": []map[string]interface{}{
						{
							"roleName":     "readWrite",
							"databaseName": "test-db",
						},
					},
				},
			},
		},
	}
}

func createDiscoveredProjectWithNetworkAccess() map[string]interface{} {
	return map[string]interface{}{
		"kind": "DiscoveredProject",
		"metadata": map[string]interface{}{
			"projectId": "test-project-id",
			"name":      "test-project",
		},
		"networkAccess": []interface{}{
			map[string]interface{}{
				"apiVersion": "matlas.mongodb.com/v1",
				"kind":       "NetworkAccess",
				"metadata": map[string]interface{}{
					"name": "office-access",
				},
				"spec": map[string]interface{}{
					"ipAddress": "203.0.113.0/24",
					"comment":   "Test network access",
				},
			},
		},
	}
}
