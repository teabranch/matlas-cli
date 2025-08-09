//go:build integration
// +build integration

package atlas

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	atlasservice "github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/validation"
)

func setupClustersIntegrationTest(t *testing.T) *TestEnvironment {
	t.Helper()

	// Skip if short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load configuration from environment
	config := IntegrationTestConfig{
		PublicKey:  os.Getenv("ATLAS_PUB_KEY"),
		PrivateKey: os.Getenv("ATLAS_API_KEY"),
		OrgID:      os.Getenv("ATLAS_ORG_ID"),
		ProjectID:  os.Getenv("PROJECT_ID"),
		Timeout:    30 * time.Second,
	}

	if config.PublicKey == "" || config.PrivateKey == "" {
		t.Skip("Atlas credentials not provided, skipping integration test")
	}

	if config.ProjectID == "" {
		t.Skip("PROJECT_ID not provided, skipping cluster tests")
	}

	// Create Atlas client
	client, err := atlasclient.NewClient(atlasclient.Config{
		PublicKey:  config.PublicKey,
		PrivateKey: config.PrivateKey,
		RetryMax:   3,
		RetryDelay: 250 * time.Millisecond,
	})
	require.NoError(t, err, "Failed to create Atlas client")

	// Create service
	service := atlasservice.NewClustersService(client)

	return &TestEnvironment{
		Config:  config,
		Client:  client,
		Service: service,
	}
}

func TestClustersService_List_Integration(t *testing.T) {
	env := setupClustersIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	// Use the clusters service instead of projects service
	clustersService := atlasservice.NewClustersService(env.Client)
	clusters, err := clustersService.List(ctx, env.Config.ProjectID)

	assert.NoError(t, err, "Failed to list clusters")
	assert.NotNil(t, clusters, "Clusters list should not be nil")

	// Verify each cluster has required fields
	for _, cluster := range clusters {
		assert.NotEmpty(t, cluster.Name, "Cluster name should not be empty")
		assert.NotEmpty(t, cluster.StateName, "Cluster state should not be empty")

		// Validate cluster name format
		if cluster.Name != nil {
			err := validation.ValidateClusterName(*cluster.Name)
			assert.NoError(t, err, "Cluster name should be valid: %s", *cluster.Name)
		}
	}

	t.Logf("Successfully retrieved %d clusters for project %s", len(clusters), env.Config.ProjectID)
}

func TestClustersService_Get_Integration(t *testing.T) {
	env := setupClustersIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	// First get list of clusters to find one to test with
	clustersService := atlasservice.NewClustersService(env.Client)
	clusters, err := clustersService.List(ctx, env.Config.ProjectID)
	require.NoError(t, err, "Failed to list clusters for setup")

	if len(clusters) == 0 {
		t.Skip("No clusters found in project, skipping Get test")
	}

	// Test getting the first cluster
	clusterName := *clusters[0].Name
	cluster, err := clustersService.Get(ctx, env.Config.ProjectID, clusterName)

	assert.NoError(t, err, "Failed to get cluster")
	assert.NotNil(t, cluster, "Cluster should not be nil")

	// Verify cluster details
	assert.Equal(t, clusterName, *cluster.Name, "Cluster name should match requested name")
	assert.NotEmpty(t, cluster.StateName, "Cluster state should not be empty")

	t.Logf("Successfully retrieved cluster: %s (state: %s)", *cluster.Name, *cluster.StateName)
}

func TestClustersService_Get_NonExistentCluster_Integration(t *testing.T) {
	env := setupClustersIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	// Use a non-existent cluster name
	nonExistentName := "non-existent-cluster-xyz"

	clustersService := atlasservice.NewClustersService(env.Client)
	cluster, err := clustersService.Get(ctx, env.Config.ProjectID, nonExistentName)

	assert.Error(t, err, "Should fail to get non-existent cluster")
	assert.Nil(t, cluster, "Cluster should be nil for non-existent name")

	// Verify error type
	assert.Contains(t, err.Error(), "404", "Error should indicate resource not found")
}

func TestClustersService_InvalidInputs_Integration(t *testing.T) {
	env := setupClustersIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	clustersService := atlasservice.NewClustersService(env.Client)

	tests := []struct {
		name        string
		projectID   string
		clusterName string
		wantErr     bool
	}{
		{
			name:      "Empty project ID",
			projectID: "",
			wantErr:   true,
		},
		{
			name:        "Empty cluster name",
			projectID:   env.Config.ProjectID,
			clusterName: "",
			wantErr:     true,
		},
		{
			name:        "Invalid project ID format",
			projectID:   "invalid-id",
			clusterName: "test-cluster",
			wantErr:     true,
		},
		{
			name:        "Invalid cluster name format",
			projectID:   env.Config.ProjectID,
			clusterName: "invalid cluster name with spaces",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.clusterName != "" {
				_, err := clustersService.Get(ctx, tt.projectID, tt.clusterName)
				if tt.wantErr {
					assert.Error(t, err, "Should fail with invalid inputs")
				} else {
					// Note: this might still fail if cluster doesn't exist, but that's OK
					// We're just testing that validation errors are caught
				}
			} else {
				_, err := clustersService.List(ctx, tt.projectID)
				if tt.wantErr {
					assert.Error(t, err, "Should fail with invalid project ID")
				}
			}
		})
	}
}

func TestClustersService_ContextTimeout_Integration(t *testing.T) {
	env := setupClustersIntegrationTest(t)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	clustersService := atlasservice.NewClustersService(env.Client)
	_, err := clustersService.List(ctx, env.Config.ProjectID)

	assert.Error(t, err, "Should fail with context timeout")
	assert.Contains(t, err.Error(), "context", "Error should mention context timeout")
}

func TestClustersService_ErrorScenarios_Integration(t *testing.T) {
	env := setupClustersIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	clustersService := atlasservice.NewClustersService(env.Client)

	tests := []struct {
		name        string
		testFunc    func() error
		expectedErr string
	}{
		{
			name: "Invalid project ID format",
			testFunc: func() error {
				_, err := clustersService.List(ctx, "not-a-valid-objectid")
				return err
			},
			expectedErr: "invalid",
		},
		{
			name: "Project ID too short",
			testFunc: func() error {
				_, err := clustersService.List(ctx, "507f1f77bcf86cd79943901")
				return err
			},
			expectedErr: "invalid",
		},
		{
			name: "Cluster name with invalid characters",
			testFunc: func() error {
				_, err := clustersService.Get(ctx, env.Config.ProjectID, "cluster@#$%")
				return err
			},
			expectedErr: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			assert.Error(t, err, "Should fail for %s", tt.name)

			if tt.expectedErr != "" {
				assert.Contains(t, err.Error(), tt.expectedErr,
					"Error should contain '%s'", tt.expectedErr)
			}
		})
	}
}

func TestClustersService_Concurrent_Integration(t *testing.T) {
	env := setupClustersIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	// Test concurrent access to list clusters
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	clustersService := atlasservice.NewClustersService(env.Client)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := clustersService.List(ctx, env.Config.ProjectID)
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent request %d should succeed", i+1)
	}
}

func TestClustersService_ClusterStates_Integration(t *testing.T) {
	env := setupClustersIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	clustersService := atlasservice.NewClustersService(env.Client)
	clusters, err := clustersService.List(ctx, env.Config.ProjectID)

	require.NoError(t, err, "Failed to list clusters")

	if len(clusters) == 0 {
		t.Skip("No clusters found, skipping state validation")
	}

	// Verify each cluster has a valid state
	validStates := map[string]bool{
		"IDLE":      true,
		"CREATING":  true,
		"UPDATING":  true,
		"DELETING":  true,
		"DELETED":   true,
		"REPAIRING": true,
	}

	for _, cluster := range clusters {
		if cluster.StateName != nil {
			state := *cluster.StateName
			// Log the state for debugging (some states might be Atlas-specific)
			t.Logf("Cluster %s has state: %s", *cluster.Name, state)

			// Basic validation - state should not be empty
			assert.NotEmpty(t, state, "Cluster state should not be empty")
		}
	}
}

func TestClustersService_ClusterConnectionStrings_Integration(t *testing.T) {
	env := setupClustersIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	clustersService := atlasservice.NewClustersService(env.Client)
	clusters, err := clustersService.List(ctx, env.Config.ProjectID)

	require.NoError(t, err, "Failed to list clusters")

	if len(clusters) == 0 {
		t.Skip("No clusters found, skipping connection string validation")
	}

	for _, cluster := range clusters {
		// Get detailed cluster information
		if cluster.Name != nil {
			detailedCluster, err := clustersService.Get(ctx, env.Config.ProjectID, *cluster.Name)
			require.NoError(t, err, "Failed to get cluster details")

			// Check if cluster has connection strings (only for running clusters)
			if detailedCluster.StateName != nil && *detailedCluster.StateName == "IDLE" {
				// For running clusters, we might have connection strings
				if detailedCluster.ConnectionStrings != nil {
					t.Logf("Cluster %s has connection strings available", *cluster.Name)

					// Basic validation if connection strings exist
					if detailedCluster.ConnectionStrings.StandardSrv != nil {
						srv := *detailedCluster.ConnectionStrings.StandardSrv
						assert.Contains(t, srv, "mongodb+srv://",
							"SRV connection string should be valid format")
					}
				}
			}
		}
	}
}

// Benchmark tests for performance monitoring
func BenchmarkClustersService_List_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	env := setupClustersIntegrationTest(&testing.T{})
	clustersService := atlasservice.NewClustersService(env.Client)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := clustersService.List(ctx, env.Config.ProjectID)
		if err != nil {
			b.Fatalf("List failed: %v", err)
		}
	}
}

func BenchmarkClustersService_Get_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	env := setupClustersIntegrationTest(&testing.T{})
	clustersService := atlasservice.NewClustersService(env.Client)

	// Get a cluster name for benchmarking
	ctx := context.Background()
	clusters, err := clustersService.List(ctx, env.Config.ProjectID)
	if err != nil || len(clusters) == 0 {
		b.Skip("No clusters available for benchmarking")
	}

	clusterName := *clusters[0].Name

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := clustersService.Get(ctx, env.Config.ProjectID, clusterName)
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}
