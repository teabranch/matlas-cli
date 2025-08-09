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

// IntegrationTestConfig holds configuration for integration tests
type IntegrationTestConfig struct {
	PublicKey  string
	PrivateKey string
	OrgID      string
	ProjectID  string
	Timeout    time.Duration
}

// TestEnvironment provides shared test setup and cleanup
type TestEnvironment struct {
	Config  IntegrationTestConfig
	Client  *atlasclient.Client
	Service *atlasservice.ProjectsService
}

func setupProjectsIntegrationTest(t *testing.T) *TestEnvironment {
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

	// Create Atlas client
	client, err := atlasclient.NewClient(atlasclient.Config{
		PublicKey:  config.PublicKey,
		PrivateKey: config.PrivateKey,
		RetryMax:   3,
		RetryDelay: 250 * time.Millisecond,
	})
	require.NoError(t, err, "Failed to create Atlas client")

	// Create service
	service := atlasservice.NewProjectsService(client)

	return &TestEnvironment{
		Config:  config,
		Client:  client,
		Service: service,
	}
}

func TestProjectsService_List_Integration(t *testing.T) {
	env := setupProjectsIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	projects, err := env.Service.List(ctx)
	assert.NoError(t, err, "Failed to list projects")
	assert.NotNil(t, projects, "Projects list should not be nil")

	// Verify each project has required fields
	for _, project := range projects {
		assert.NotEmpty(t, project.Id, "Project ID should not be empty")
		assert.NotEmpty(t, project.Name, "Project name should not be empty")
		assert.NotEmpty(t, project.OrgId, "Organization ID should not be empty")

		// Validate project ID format
		err := validation.ValidateProjectID(*project.Id)
		assert.NoError(t, err, "Project ID should be valid: %s", *project.Id)
	}

	t.Logf("Successfully retrieved %d projects", len(projects))
}

func TestProjectsService_ListByOrg_Integration(t *testing.T) {
	env := setupProjectsIntegrationTest(t)

	if env.Config.OrgID == "" {
		t.Skip("ATLAS_ORG_ID not provided, skipping organization-specific test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	projects, err := env.Service.ListByOrg(ctx, env.Config.OrgID)
	assert.NoError(t, err, "Failed to list projects by organization")
	assert.NotNil(t, projects, "Projects list should not be nil")

	// Verify all projects belong to the specified organization
	for _, project := range projects {
		assert.Equal(t, env.Config.OrgID, *project.OrgId,
			"Project %s should belong to organization %s", *project.Id, env.Config.OrgID)
	}

	t.Logf("Successfully retrieved %d projects for organization %s", len(projects), env.Config.OrgID)
}

func TestProjectsService_Get_Integration(t *testing.T) {
	env := setupProjectsIntegrationTest(t)

	if env.Config.ProjectID == "" {
		t.Skip("PROJECT_ID not provided, skipping specific project test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	project, err := env.Service.Get(ctx, env.Config.ProjectID)
	assert.NoError(t, err, "Failed to get project")
	assert.NotNil(t, project, "Project should not be nil")

	// Verify project details
	assert.Equal(t, env.Config.ProjectID, *project.Id, "Project ID should match requested ID")
	assert.NotEmpty(t, project.Name, "Project name should not be empty")
	assert.NotEmpty(t, project.OrgId, "Organization ID should not be empty")

	t.Logf("Successfully retrieved project: %s (%s)", *project.Name, *project.Id)
}

func TestProjectsService_Get_NonExistentProject_Integration(t *testing.T) {
	env := setupProjectsIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	// Use a valid format but non-existent project ID
	nonExistentID := "507f1f77bcf86cd799439011"

	project, err := env.Service.Get(ctx, nonExistentID)
	assert.Error(t, err, "Should fail to get non-existent project")
	assert.Nil(t, project, "Project should be nil for non-existent ID")

	// Verify error type
	assert.Contains(t, err.Error(), "404", "Error should indicate resource not found")
}

func TestProjectsService_InvalidInputs_Integration(t *testing.T) {
	env := setupProjectsIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	tests := []struct {
		name      string
		projectID string
		orgID     string
		wantErr   bool
	}{
		{
			name:      "Empty project ID",
			projectID: "",
			wantErr:   true,
		},
		{
			name:      "Invalid project ID format",
			projectID: "invalid-id",
			wantErr:   true,
		},
		{
			name:    "Empty org ID for ListByOrg",
			orgID:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.projectID != "" {
				_, err := env.Service.Get(ctx, tt.projectID)
				if tt.wantErr {
					assert.Error(t, err, "Should fail with invalid project ID")
				} else {
					assert.NoError(t, err, "Should succeed with valid project ID")
				}
			}

			if tt.orgID == "" && tt.wantErr {
				_, err := env.Service.ListByOrg(ctx, tt.orgID)
				assert.Error(t, err, "Should fail with empty org ID")
			}
		})
	}
}

func TestProjectsService_ContextTimeout_Integration(t *testing.T) {
	env := setupProjectsIntegrationTest(t)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := env.Service.List(ctx)
	assert.Error(t, err, "Should fail with context timeout")
	assert.Contains(t, err.Error(), "context", "Error should mention context timeout")
}

func TestProjectsService_ErrorScenarios_Integration(t *testing.T) {
	env := setupProjectsIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	tests := []struct {
		name        string
		testFunc    func() error
		expectedErr string
	}{
		{
			name: "Invalid project ID format",
			testFunc: func() error {
				_, err := env.Service.Get(ctx, "not-a-valid-objectid")
				return err
			},
			expectedErr: "invalid",
		},
		{
			name: "Project ID too short",
			testFunc: func() error {
				_, err := env.Service.Get(ctx, "507f1f77bcf86cd79943901")
				return err
			},
			expectedErr: "invalid",
		},
		{
			name: "Project ID too long",
			testFunc: func() error {
				_, err := env.Service.Get(ctx, "507f1f77bcf86cd7994390111")
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

func TestProjectsService_Concurrent_Integration(t *testing.T) {
	env := setupProjectsIntegrationTest(t)

	if env.Config.ProjectID == "" {
		t.Skip("PROJECT_ID not provided, skipping concurrent test")
	}

	// Test concurrent access to the same project
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
			defer cancel()

			_, err := env.Service.Get(ctx, env.Config.ProjectID)
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent request %d should succeed", i+1)
	}
}

func TestProjectsService_RateLimitHandling_Integration(t *testing.T) {
	env := setupProjectsIntegrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Make rapid requests to test rate limit handling
	const numRequests = 100
	errors := 0
	successes := 0

	for i := 0; i < numRequests; i++ {
		_, err := env.Service.List(ctx)
		if err != nil {
			errors++
			// Rate limit errors should be handled gracefully
			if i > 50 { // Allow some errors after many requests
				t.Logf("Request %d failed (expected with rate limiting): %v", i+1, err)
			}
		} else {
			successes++
		}

		// Small delay to avoid overwhelming the API
		time.Sleep(10 * time.Millisecond)
	}

	t.Logf("Completed %d requests: %d successes, %d errors", numRequests, successes, errors)

	// Should have at least some successes
	assert.Greater(t, successes, 0, "Should have at least some successful requests")

	// If we get rate limited, it should be handled gracefully (not panic)
	// The exact ratio depends on Atlas API limits
}

// Benchmark tests for performance monitoring
func BenchmarkProjectsService_List_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	env := setupProjectsIntegrationTest(&testing.T{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.Service.List(ctx)
		if err != nil {
			b.Fatalf("List failed: %v", err)
		}
	}
}

func BenchmarkProjectsService_Get_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	env := setupProjectsIntegrationTest(&testing.T{})

	if env.Config.ProjectID == "" {
		b.Skip("PROJECT_ID not provided, skipping benchmark")
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.Service.Get(ctx, env.Config.ProjectID)
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}
