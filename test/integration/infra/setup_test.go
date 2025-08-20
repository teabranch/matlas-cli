//go:build integration

package apply_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/apply"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
)

// TestEnvironment provides a managed environment for integration tests
type TestEnvironment struct {
	ProjectID            string
	ClusterService       *atlas.ClustersService
	DatabaseUserService  *atlas.DatabaseUsersService
	NetworkAccessService *atlas.NetworkAccessListsService
	ProjectService       *atlas.ProjectsService
	SearchService        *atlas.SearchService
	DatabaseService      *database.Service
	Resources            []TestResource
	cleanupFuncs         []func() error
}

// TestResource represents a resource created during testing that needs cleanup
type TestResource struct {
	Type string
	ID   string
	Name string
}

// IntegrationTestConfig holds configuration for integration tests
type IntegrationTestConfig struct {
	AtlasPublicKey  string
	AtlasPrivateKey string
	AtlasOrgID      string
	AtlasProjectID  string
	TestTimeout     time.Duration
	CleanupTimeout  time.Duration
	SkipCleanup     bool
	VerboseLogging  bool
}

// loadEnvFile loads environment variables from .env file if it exists
func loadEnvFile() error {
	// Look for .env file in current directory and parent directories
	envPaths := []string{
		".env",
		"../.env",
		"../../.env",
		"../../../.env",
	}

	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			return loadEnvFromFile(envPath)
		}
	}

	// No .env file found, that's ok
	return nil
}

// loadEnvFromFile loads environment variables from a specific file
func loadEnvFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Only set if not already set in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

// mapEnvVariables maps .env variable names to expected integration test variable names
func mapEnvVariables() {
	// Map .env variables to expected names
	mappings := map[string]string{
		"ATLAS_PUB_KEY": "ATLAS_PUBLIC_KEY",
		"ATLAS_API_KEY": "ATLAS_PRIVATE_KEY",
		"ORG_ID":        "ATLAS_ORG_ID",
		"PROJECT_ID":    "ATLAS_PROJECT_ID",
	}

	for envVar, targetVar := range mappings {
		if value := os.Getenv(envVar); value != "" && os.Getenv(targetVar) == "" {
			os.Setenv(targetVar, value)
		}
	}
}

// SetupTestEnvironment creates and configures a test environment for integration tests
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	// Load .env file if it exists
	if err := loadEnvFile(); err != nil {
		t.Logf("Warning: Failed to load .env file: %v", err)
	}

	// Map environment variables
	mapEnvVariables()

	config := getIntegrationTestConfig(t)

	if config.AtlasPublicKey == "" || config.AtlasPrivateKey == "" {
		t.Skip("Atlas credentials not provided - skipping integration test")
	}

	if config.AtlasProjectID == "" {
		t.Skip("Atlas project ID not provided - skipping integration test")
	}

	if config.VerboseLogging {
		t.Logf("Integration test config: PublicKey=%s..., ProjectID=%s",
			config.AtlasPublicKey[:8], config.AtlasProjectID)
	}

	// Create real Atlas client
	atlasClient, err := createAtlasClient(config)
	if err != nil {
		t.Fatalf("Failed to create Atlas client: %v", err)
	}

	// Create Atlas services with real implementations
	env := &TestEnvironment{
		ProjectID:            config.AtlasProjectID,
		ClusterService:       atlas.NewClustersService(atlasClient),
		DatabaseUserService:  atlas.NewDatabaseUsersService(atlasClient),
		NetworkAccessService: atlas.NewNetworkAccessListsService(atlasClient),
		ProjectService:       atlas.NewProjectsService(atlasClient),
		SearchService:        atlas.NewSearchService(atlasClient),
		DatabaseService:      nil, // Database service is separate and will be implemented later
		Resources:            []TestResource{},
		cleanupFuncs:         []func() error{},
	}

	// Register cleanup
	t.Cleanup(func() {
		if err := env.Cleanup(); err != nil {
			t.Errorf("Cleanup failed: %v", err)
		}
	})

	return env
}

// getIntegrationTestConfig reads integration test configuration from environment variables
func getIntegrationTestConfig(t *testing.T) IntegrationTestConfig {
	return IntegrationTestConfig{
		AtlasPublicKey:  os.Getenv("ATLAS_PUBLIC_KEY"),
		AtlasPrivateKey: os.Getenv("ATLAS_PRIVATE_KEY"),
		AtlasOrgID:      os.Getenv("ATLAS_ORG_ID"),
		AtlasProjectID:  os.Getenv("ATLAS_PROJECT_ID"),
		TestTimeout:     getEnvDuration("ATLAS_TEST_TIMEOUT", 5*time.Minute),
		CleanupTimeout:  getEnvDuration("ATLAS_CLEANUP_TIMEOUT", 2*time.Minute),
		SkipCleanup:     os.Getenv("ATLAS_SKIP_CLEANUP") == "true",
		VerboseLogging:  os.Getenv("ATLAS_VERBOSE") == "true",
	}
}

// getEnvDuration parses a duration from environment variable with fallback
func getEnvDuration(envVar string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return duration
}

// CreateExecutor creates a configured executor for integration testing
func (env *TestEnvironment) CreateExecutor() *apply.AtlasExecutor {
	config := apply.DefaultExecutorConfig()
	config.MaxConcurrentOperations = 1 // Sequential for predictable testing

	return apply.NewAtlasExecutor(
		env.ClusterService,
		env.DatabaseUserService,
		env.NetworkAccessService,
		env.ProjectService,
		env.SearchService,
		env.DatabaseService,
		config,
	)
}

// CreateTestCluster creates a test cluster for integration testing
func (env *TestEnvironment) CreateTestCluster(name string) (*types.ClusterConfig, error) {
	// Generate unique cluster name
	clusterName := fmt.Sprintf("test-%s-%d", name, time.Now().Unix())

	cluster := &types.ClusterConfig{
		Metadata: types.ResourceMetadata{
			Name: clusterName,
		},
		Provider:       "AWS",
		Region:         "US_EAST_1",
		InstanceSize:   "M0", // Free tier for testing
		MongoDBVersion: "6.0",
		ClusterType:    "REPLICASET",
	}

	// Register for cleanup
	env.RegisterResource(TestResource{
		Type: "cluster",
		ID:   clusterName,
		Name: clusterName,
	})

	return cluster, nil
}

// CreateTestDatabaseUser creates a test database user for integration testing
func (env *TestEnvironment) CreateTestDatabaseUser(username string) (*types.DatabaseUserConfig, error) {
	// Generate unique username
	userName := fmt.Sprintf("test-%s-%d", username, time.Now().Unix())

	user := &types.DatabaseUserConfig{
		Metadata: types.ResourceMetadata{
			Name: userName,
		},
		Username: userName,
		Password: "TestPassword123!",
		Roles: []types.DatabaseRoleConfig{
			{
				RoleName:     "readWrite",
				DatabaseName: "admin",
			},
		},
	}

	// Register for cleanup
	env.RegisterResource(TestResource{
		Type: "databaseUser",
		ID:   userName,
		Name: userName,
	})

	return user, nil
}

// RegisterResource registers a resource for cleanup
func (env *TestEnvironment) RegisterResource(resource TestResource) {
	env.Resources = append(env.Resources, resource)
}

// RegisterCleanup registers a cleanup function
func (env *TestEnvironment) RegisterCleanup(fn func() error) {
	env.cleanupFuncs = append(env.cleanupFuncs, fn)
}

// Cleanup cleans up all resources created during testing
func (env *TestEnvironment) Cleanup() error {
	var errors []error

	// Run custom cleanup functions
	for _, cleanupFn := range env.cleanupFuncs {
		if err := cleanupFn(); err != nil {
			errors = append(errors, fmt.Errorf("custom cleanup failed: %w", err))
		}
	}

	// Cleanup registered resources
	for _, resource := range env.Resources {
		if err := env.cleanupResource(resource); err != nil {
			errors = append(errors, fmt.Errorf("cleanup %s %s failed: %w", resource.Type, resource.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// cleanupResource cleans up a specific resource
func (env *TestEnvironment) cleanupResource(resource TestResource) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch resource.Type {
	case "cluster":
		// Would delete cluster: return env.ClusterService.Delete(ctx, env.ProjectID, resource.ID)
		return nil // Placeholder - clusters are expensive and take time to delete
	case "databaseUser":
		// Delete database user using real Atlas service
		if env.DatabaseUserService != nil {
			err := env.DatabaseUserService.Delete(ctx, env.ProjectID, "admin", resource.ID)
			if err != nil {
				// Log but don't fail cleanup for "not found" errors
				if strings.Contains(err.Error(), "NOT_FOUND") || strings.Contains(err.Error(), "does not exist") {
					return nil // User already doesn't exist, cleanup successful
				}
				return fmt.Errorf("failed to delete database user %s: %w", resource.ID, err)
			}
		}
		return nil
	case "networkAccess":
		// Would delete network access: return env.NetworkAccessService.Delete(ctx, env.ProjectID, resource.ID)
		return nil // Placeholder
	default:
		return fmt.Errorf("unknown resource type: %s", resource.Type)
	}
}

// WaitForClusterReady waits for a cluster to become ready
func (env *TestEnvironment) WaitForClusterReady(clusterName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for cluster %s to become ready", clusterName)
		case <-ticker.C:
			// Would check cluster status: status, err := env.ClusterService.GetStatus(ctx, env.ProjectID, clusterName)
			// For now, assume ready after first check
			return nil
		}
	}
}

// ValidateTestEnvironment validates that the test environment is properly configured
func ValidateTestEnvironment(t *testing.T, env *TestEnvironment) {
	if env.ProjectID == "" {
		t.Fatal("Test environment project ID is empty")
	}

	// Additional validations would go here
	// For example, checking that services are properly initialized
	// and can communicate with Atlas
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

// GetProjectDir returns the project root directory by looking for go.mod
func GetProjectDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find project root (go.mod)")
}

// createAtlasClient creates a real Atlas client with the provided configuration
func createAtlasClient(config IntegrationTestConfig) (*atlasclient.Client, error) {
	clientConfig := atlasclient.Config{
		// The atlasclient.Config will use environment variables by default
		// since we've already loaded and mapped them from .env
	}

	return atlasclient.NewClient(clientConfig)
}
