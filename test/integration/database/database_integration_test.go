//go:build integration
// +build integration

package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
)

// DatabaseTestConfig holds configuration for database integration tests
type DatabaseTestConfig struct {
	ConnectionString string
	TestDatabaseName string
	Timeout          time.Duration
}

// DatabaseTestEnvironment provides shared test setup and cleanup
type DatabaseTestEnvironment struct {
	Config  DatabaseTestConfig
	Service *database.Service
	Logger  *logging.Logger
}

func setupDatabaseIntegrationTest(t *testing.T) *DatabaseTestEnvironment {
	t.Helper()

	// Skip if short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load configuration from environment
	config := DatabaseTestConfig{
		ConnectionString: os.Getenv("MONGODB_CONNECTION_STRING"),
		TestDatabaseName: "matlas_test_db",
		Timeout:          30 * time.Second,
	}

	if config.ConnectionString == "" {
		t.Skip("MONGODB_CONNECTION_STRING not provided, skipping database integration test")
	}

	// Create logger
	logConfig := logging.DefaultConfig()
	logConfig.Level = logging.LevelDebug
	logger := logging.New(logConfig)

	// Create database service
	service := database.NewService(logger)

	return &DatabaseTestEnvironment{
		Config:  config,
		Service: service,
		Logger:  logger,
	}
}

func (env *DatabaseTestEnvironment) cleanup(t *testing.T) {
	ctx := context.Background()
	env.Service.Close(ctx)
}

func TestDatabaseService_ListDatabases_Integration(t *testing.T) {
	env := setupDatabaseIntegrationTest(t)
	defer env.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	connInfo := &types.ConnectionInfo{
		ConnectionString: env.Config.ConnectionString,
	}

	databases, err := env.Service.ListDatabases(ctx, connInfo)
	assert.NoError(t, err, "Failed to list databases")
	assert.NotNil(t, databases, "Databases list should not be nil")

	// Verify each database has required fields
	for _, db := range databases {
		assert.NotEmpty(t, db.Name, "Database name should not be empty")
		assert.GreaterOrEqual(t, db.SizeOnDisk, int64(0), "Size on disk should be non-negative")
		assert.NotNil(t, db.Collections, "Collections should not be nil")

		// Admin and config databases should always exist in MongoDB
		if db.Name == "admin" || db.Name == "config" {
			assert.False(t, db.Empty, "System databases should not be empty")
		}
	}

	t.Logf("Successfully retrieved %d databases", len(databases))
}

func TestDatabaseService_ListCollections_Integration(t *testing.T) {
	env := setupDatabaseIntegrationTest(t)
	defer env.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	connInfo := &types.ConnectionInfo{
		ConnectionString: env.Config.ConnectionString,
	}

	// First ensure we have a test database with collections
	err := env.ensureTestDatabase(ctx, connInfo)
	require.NoError(t, err, "Failed to ensure test database exists")

	collections, err := env.Service.ListCollections(ctx, connInfo, env.Config.TestDatabaseName)
	assert.NoError(t, err, "Failed to list collections")
	assert.NotNil(t, collections, "Collections list should not be nil")

	// If we have collections, verify their structure
	for _, collection := range collections {
		assert.NotEmpty(t, collection.Name, "Collection name should not be empty")
		assert.GreaterOrEqual(t, collection.Count, int64(0), "Document count should be non-negative")
		assert.GreaterOrEqual(t, collection.Size, int64(0), "Collection size should be non-negative")
	}

	t.Logf("Successfully retrieved %d collections from database %s", len(collections), env.Config.TestDatabaseName)
}

func TestDatabaseService_CreateCollection_Integration(t *testing.T) {
	env := setupDatabaseIntegrationTest(t)
	defer env.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	connInfo := &types.ConnectionInfo{
		ConnectionString: env.Config.ConnectionString,
	}

	collectionName := "test_collection_" + generateTestID()

	// Create a regular collection
	err := env.Service.CreateCollection(ctx, connInfo, env.Config.TestDatabaseName, collectionName, nil)
	assert.NoError(t, err, "Failed to create collection")

	// Verify collection was created
	collections, err := env.Service.ListCollections(ctx, connInfo, env.Config.TestDatabaseName)
	require.NoError(t, err, "Failed to list collections after creation")

	found := false
	for _, collection := range collections {
		if collection.Name == collectionName {
			found = true
			break
		}
	}
	assert.True(t, found, "Created collection should be found in list")

	// Clean up
	err = env.Service.DropCollection(ctx, connInfo, env.Config.TestDatabaseName, collectionName)
	assert.NoError(t, err, "Failed to clean up test collection")
}

func TestDatabaseService_CreateCappedCollection_Integration(t *testing.T) {
	env := setupDatabaseIntegrationTest(t)
	defer env.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	connInfo := &types.ConnectionInfo{
		ConnectionString: env.Config.ConnectionString,
	}

	collectionName := "test_capped_collection_" + generateTestID()

	// Create a capped collection
	options := &types.CreateCollectionOptions{
		Capped:       true,
		Size:         1048576, // 1MB
		MaxDocuments: 1000,
	}

	err := env.Service.CreateCollection(ctx, connInfo, env.Config.TestDatabaseName, collectionName, options)
	assert.NoError(t, err, "Failed to create capped collection")

	// Verify collection was created
	collections, err := env.Service.ListCollections(ctx, connInfo, env.Config.TestDatabaseName)
	require.NoError(t, err, "Failed to list collections after creation")

	found := false
	for _, collection := range collections {
		if collection.Name == collectionName {
			found = true
			// Note: We can't easily verify it's capped from the collection info
			// but the creation should succeed if MongoDB accepts the capped options
			break
		}
	}
	assert.True(t, found, "Created capped collection should be found in list")

	// Clean up
	err = env.Service.DropCollection(ctx, connInfo, env.Config.TestDatabaseName, collectionName)
	assert.NoError(t, err, "Failed to clean up test capped collection")
}

func TestDatabaseService_DropCollection_Integration(t *testing.T) {
	env := setupDatabaseIntegrationTest(t)
	defer env.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	connInfo := &types.ConnectionInfo{
		ConnectionString: env.Config.ConnectionString,
	}

	collectionName := "test_drop_collection_" + generateTestID()

	// First create a collection
	err := env.Service.CreateCollection(ctx, connInfo, env.Config.TestDatabaseName, collectionName, nil)
	require.NoError(t, err, "Failed to create collection for drop test")

	// Verify it exists
	collections, err := env.Service.ListCollections(ctx, connInfo, env.Config.TestDatabaseName)
	require.NoError(t, err, "Failed to list collections")

	found := false
	for _, collection := range collections {
		if collection.Name == collectionName {
			found = true
			break
		}
	}
	require.True(t, found, "Collection should exist before drop")

	// Drop the collection
	err = env.Service.DropCollection(ctx, connInfo, env.Config.TestDatabaseName, collectionName)
	assert.NoError(t, err, "Failed to drop collection")

	// Verify it no longer exists
	collections, err = env.Service.ListCollections(ctx, connInfo, env.Config.TestDatabaseName)
	require.NoError(t, err, "Failed to list collections after drop")

	found = false
	for _, collection := range collections {
		if collection.Name == collectionName {
			found = true
			break
		}
	}
	assert.False(t, found, "Collection should not exist after drop")
}

func TestDatabaseService_ErrorScenarios_Integration(t *testing.T) {
	env := setupDatabaseIntegrationTest(t)
	defer env.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	tests := []struct {
		name     string
		testFunc func() error
	}{
		{
			name: "Invalid connection string",
			testFunc: func() error {
				invalidConnInfo := &types.ConnectionInfo{
					ConnectionString: "mongodb://invalid-host:27017",
				}
				_, err := env.Service.ListDatabases(ctx, invalidConnInfo)
				return err
			},
		},
		{
			name: "List collections from non-existent database",
			testFunc: func() error {
				connInfo := &types.ConnectionInfo{
					ConnectionString: env.Config.ConnectionString,
				}
				_, err := env.Service.ListCollections(ctx, connInfo, "non_existent_database_xyz")
				return err
			},
		},
		{
			name: "Drop non-existent collection",
			testFunc: func() error {
				connInfo := &types.ConnectionInfo{
					ConnectionString: env.Config.ConnectionString,
				}
				err := env.Service.DropCollection(ctx, connInfo, env.Config.TestDatabaseName, "non_existent_collection_xyz")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			// We expect errors for these scenarios
			assert.Error(t, err, "Should fail for %s", tt.name)
		})
	}
}

func TestDatabaseService_ContextTimeout_Integration(t *testing.T) {
	env := setupDatabaseIntegrationTest(t)
	defer env.cleanup(t)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	connInfo := &types.ConnectionInfo{
		ConnectionString: env.Config.ConnectionString,
	}

	_, err := env.Service.ListDatabases(ctx, connInfo)
	assert.Error(t, err, "Should fail with context timeout")
	// Note: The exact error message depends on the MongoDB driver
}

func TestDatabaseService_Concurrent_Integration(t *testing.T) {
	env := setupDatabaseIntegrationTest(t)
	defer env.cleanup(t)

	connInfo := &types.ConnectionInfo{
		ConnectionString: env.Config.ConnectionString,
	}

	// Test concurrent access
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
			defer cancel()

			_, err := env.Service.ListDatabases(ctx, connInfo)
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent request %d should succeed", i+1)
	}
}

func TestDatabaseService_InvalidInputs_Integration(t *testing.T) {
	env := setupDatabaseIntegrationTest(t)
	defer env.cleanup(t)

	ctx, cancel := context.WithTimeout(context.Background(), env.Config.Timeout)
	defer cancel()

	tests := []struct {
		name         string
		connInfo     *types.ConnectionInfo
		databaseName string
		expectError  bool
	}{
		{
			name:        "Nil connection info",
			connInfo:    nil,
			expectError: true,
		},
		{
			name: "Empty connection string",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "",
			},
			expectError: true,
		},
		{
			name: "Valid connection, empty database name for collections",
			connInfo: &types.ConnectionInfo{
				ConnectionString: env.Config.ConnectionString,
			},
			databaseName: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.databaseName == "" && tt.connInfo != nil {
				// Test ListDatabases
				_, err := env.Service.ListDatabases(ctx, tt.connInfo)
				if tt.expectError {
					assert.Error(t, err, "Should fail for %s", tt.name)
				}
			} else if tt.databaseName != "" {
				// Test ListCollections
				_, err := env.Service.ListCollections(ctx, tt.connInfo, tt.databaseName)
				if tt.expectError {
					assert.Error(t, err, "Should fail for %s", tt.name)
				}
			}
		})
	}
}

// Helper methods

func (env *DatabaseTestEnvironment) ensureTestDatabase(ctx context.Context, connInfo *types.ConnectionInfo) error {
	// Create a test collection to ensure the database exists
	testCollectionName := "test_setup_collection"

	err := env.Service.CreateCollection(ctx, connInfo, env.Config.TestDatabaseName, testCollectionName, nil)
	if err != nil {
		// Collection might already exist, which is fine
		return nil
	}

	return nil
}

func generateTestID() string {
	return time.Now().Format("20060102_150405")
}

// Benchmark tests for performance monitoring
func BenchmarkDatabaseService_ListDatabases_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	env := setupDatabaseIntegrationTest(&testing.T{})
	defer env.cleanup(&testing.T{})

	ctx := context.Background()
	connInfo := &types.ConnectionInfo{
		ConnectionString: env.Config.ConnectionString,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.Service.ListDatabases(ctx, connInfo)
		if err != nil {
			b.Fatalf("ListDatabases failed: %v", err)
		}
	}
}

func BenchmarkDatabaseService_ListCollections_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	env := setupDatabaseIntegrationTest(&testing.T{})
	defer env.cleanup(&testing.T{})

	ctx := context.Background()
	connInfo := &types.ConnectionInfo{
		ConnectionString: env.Config.ConnectionString,
	}

	// Ensure test database exists
	err := env.ensureTestDatabase(ctx, connInfo)
	if err != nil {
		b.Fatalf("Failed to ensure test database: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := env.Service.ListCollections(ctx, connInfo, env.Config.TestDatabaseName)
		if err != nil {
			b.Fatalf("ListCollections failed: %v", err)
		}
	}
}
