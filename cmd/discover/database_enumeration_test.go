package discover

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/teabranch/matlas-cli/internal/types"
)

// TestNewDatabaseEnumerator tests the database enumerator creation
func TestNewDatabaseEnumerator(t *testing.T) {
	opts := &DiscoverOptions{
		MaxConcurrency: 5,
		Timeout:        30 * time.Second,
		Verbose:        true,
		MongoURI:       "mongodb+srv://user:pass@host/",
		MongoUsername:  "user",
		MongoPassword:  "pass",
	}

	enumerator := NewDatabaseEnumerator(nil, opts)
	require.NotNil(t, enumerator)

	assert.Equal(t, 5, enumerator.maxConcurrency)
	assert.Equal(t, 30*time.Second, enumerator.timeout)
	assert.True(t, enumerator.verbose)
	assert.Equal(t, "mongodb+srv://user:pass@host/", enumerator.mongoURI)
	assert.Equal(t, "user", enumerator.mongoUsername)
	assert.Equal(t, "pass", enumerator.mongoPassword)
}

// TestIsSystemDatabase tests system database detection
func TestIsSystemDatabase(t *testing.T) {
	enumerator := &DatabaseEnumerator{}

	tests := []struct {
		dbName   string
		expected bool
	}{
		{"admin", true},
		{"local", true},
		{"config", true},
		{"myapp", false},
		{"test", false},
		{"production", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.dbName, func(t *testing.T) {
			result := enumerator.isSystemDatabase(tt.dbName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsSystemCollection tests system collection detection
func TestIsSystemCollection(t *testing.T) {
	enumerator := &DatabaseEnumerator{}

	tests := []struct {
		collName string
		expected bool
	}{
		{"system.users", true},
		{"system.indexes", true},
		{"system.", false}, // Too short
		{"fs.files", true},
		{"fs.chunks", true},
		{"users", false},
		{"products", false},
		{"orders", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.collName, func(t *testing.T) {
			result := enumerator.isSystemCollection(tt.collName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetClusterConnectionString tests connection string generation
func TestGetClusterConnectionString(t *testing.T) {
	enumerator := &DatabaseEnumerator{}

	tests := []struct {
		name     string
		cluster  types.ClusterManifest
		expected string
		hasError bool
	}{
		{
			name: "valid cluster with project ID",
			cluster: types.ClusterManifest{
				Metadata: types.ResourceMetadata{
					Name: "cluster0",
					Labels: map[string]string{
						"atlas.mongodb.com/project-id": "507f1f77bcf86cd799439011",
					},
				},
			},
			expected: "mongodb+srv://<username>:<password>@cluster0.mongodb.net/",
			hasError: false,
		},
		{
			name: "cluster without project ID (fallback to generic SRV)",
			cluster: types.ClusterManifest{
				Metadata: types.ResourceMetadata{
					Name:   "cluster1",
					Labels: map[string]string{},
				},
			},
			expected: "mongodb+srv://<username>:<password>@cluster1.mongodb.net/",
			hasError: false,
		},
		{
			name: "cluster with empty labels (fallback to generic SRV)",
			cluster: types.ClusterManifest{
				Metadata: types.ResourceMetadata{
					Name: "cluster2",
				},
			},
			expected: "mongodb+srv://<username>:<password>@cluster2.mongodb.net/",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := enumerator.getClusterConnectionString(tt.cluster)

			if tt.hasError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCredentialHelpers(t *testing.T) {
	// URIs without credentials
	uri := "mongodb+srv://cluster.mongodb.net/"
	require.False(t, hasCredentials(uri))
	newURI := injectCredentials(uri, "user", "pass")
	require.True(t, hasCredentials(newURI))
	require.Contains(t, newURI, "user:pass@")

	// URIs with credentials already
	uri2 := "mongodb://u:p@localhost:27017"
	require.True(t, hasCredentials(uri2))
	require.Equal(t, uri2, injectCredentials(uri2, "x", "y"))
}

// TestEnumerateClusterDatabases_EmptyInput tests behavior with empty cluster list
func TestEnumerateClusterDatabases_EmptyInput(t *testing.T) {
	enumerator := &DatabaseEnumerator{
		maxConcurrency: 1,
		timeout:        5 * time.Second,
		verbose:        false,
	}

	ctx := context.Background()
	clusters := []types.ClusterManifest{}

	databases, err := enumerator.EnumerateClusterDatabases(ctx, clusters)
	assert.NoError(t, err)
	assert.Empty(t, databases)
}

// TestDatabaseEnumerationError tests the error type
func TestDatabaseEnumerationError(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		expected string
	}{
		{
			name:     "single error",
			errors:   []error{assert.AnError},
			expected: "database enumeration failed: assert.AnError general error for testing",
		},
		{
			name:     "multiple errors",
			errors:   []error{assert.AnError, assert.AnError},
			expected: "database enumeration failed for 2 clusters: assert.AnError general error for testing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &DatabaseEnumerationError{
				ClusterErrors: tt.errors,
			}

			assert.Equal(t, tt.expected, err.Error())
			assert.Equal(t, tt.errors, err.Unwrap())
		})
	}
}

// TestDatabaseInfoStructure tests the database info structure
func TestDatabaseInfoStructure(t *testing.T) {
	dbInfo := DatabaseInfo{
		Name:        "testdb",
		ClusterName: "cluster0",
		SizeOnDisk:  1024000,
		Collections: []CollectionInfo{
			{
				Name:          "users",
				DocumentCount: 1000,
				StorageSize:   512000,
				IndexCount:    3,
				Indexes: []IndexInfo{
					{
						Name:   "_id_",
						Keys:   map[string]int{"_id": 1},
						Unique: true,
					},
				},
			},
			{
				Name:          "products",
				DocumentCount: 2000,
				StorageSize:   1024000,
				IndexCount:    5,
			},
		},
	}

	assert.Equal(t, "testdb", dbInfo.Name)
	assert.Equal(t, "cluster0", dbInfo.ClusterName)
	assert.Equal(t, int64(1024000), dbInfo.SizeOnDisk)
	assert.Len(t, dbInfo.Collections, 2)

	// Test first collection
	usersCollection := dbInfo.Collections[0]
	assert.Equal(t, "users", usersCollection.Name)
	assert.Equal(t, int64(1000), usersCollection.DocumentCount)
	assert.Equal(t, int64(512000), usersCollection.StorageSize)
	assert.Equal(t, 3, usersCollection.IndexCount)
	assert.Len(t, usersCollection.Indexes, 1)

	// Test index
	idIndex := usersCollection.Indexes[0]
	assert.Equal(t, "_id_", idIndex.Name)
	assert.True(t, idIndex.Unique)
	assert.False(t, idIndex.Sparse)
	assert.Nil(t, idIndex.TTL)

	// Test second collection
	productsCollection := dbInfo.Collections[1]
	assert.Equal(t, "products", productsCollection.Name)
	assert.Equal(t, int64(2000), productsCollection.DocumentCount)
	assert.Equal(t, int64(1024000), productsCollection.StorageSize)
	assert.Equal(t, 5, productsCollection.IndexCount)
	assert.Len(t, productsCollection.Indexes, 0)
}

// TestCollectionInfo tests the collection info structure
func TestCollectionInfo(t *testing.T) {
	collInfo := CollectionInfo{
		Name:          "orders",
		DocumentCount: 50000,
		StorageSize:   25000000,
		IndexCount:    7,
		Indexes: []IndexInfo{
			{
				Name:   "_id_",
				Keys:   map[string]int{"_id": 1},
				Unique: true,
			},
			{
				Name:   "customer_id_1",
				Keys:   map[string]int{"customer_id": 1},
				Unique: false,
				Sparse: true,
			},
			{
				Name:          "expires_at_1",
				Keys:          map[string]int{"expires_at": 1},
				TTL:           &[]int{3600}[0], // 1 hour TTL
				PartialFilter: map[string]interface{}{"status": "pending"},
			},
		},
		ShardKey:       map[string]int{"customer_id": 1},
		ValidationRule: map[string]interface{}{"$jsonSchema": map[string]interface{}{"required": []string{"customer_id"}}},
	}

	assert.Equal(t, "orders", collInfo.Name)
	assert.Equal(t, int64(50000), collInfo.DocumentCount)
	assert.Equal(t, int64(25000000), collInfo.StorageSize)
	assert.Equal(t, 7, collInfo.IndexCount)
	assert.Len(t, collInfo.Indexes, 3)

	// Test _id index
	idIndex := collInfo.Indexes[0]
	assert.Equal(t, "_id_", idIndex.Name)
	assert.True(t, idIndex.Unique)
	assert.False(t, idIndex.Sparse)
	assert.Nil(t, idIndex.TTL)
	assert.Nil(t, idIndex.PartialFilter)

	// Test customer_id index
	customerIndex := collInfo.Indexes[1]
	assert.Equal(t, "customer_id_1", customerIndex.Name)
	assert.False(t, customerIndex.Unique)
	assert.True(t, customerIndex.Sparse)
	assert.Nil(t, customerIndex.TTL)

	// Test TTL index
	ttlIndex := collInfo.Indexes[2]
	assert.Equal(t, "expires_at_1", ttlIndex.Name)
	assert.False(t, ttlIndex.Unique)
	assert.False(t, ttlIndex.Sparse)
	require.NotNil(t, ttlIndex.TTL)
	assert.Equal(t, 3600, *ttlIndex.TTL)
	assert.NotNil(t, ttlIndex.PartialFilter)

	// Test shard key and validation rule
	assert.NotNil(t, collInfo.ShardKey)
	assert.NotNil(t, collInfo.ValidationRule)
}

// TestIndexInfo tests the index info structure
func TestIndexInfo(t *testing.T) {
	tests := []struct {
		name  string
		index IndexInfo
	}{
		{
			name: "simple unique index",
			index: IndexInfo{
				Name:   "email_1",
				Keys:   map[string]int{"email": 1},
				Unique: true,
			},
		},
		{
			name: "compound index",
			index: IndexInfo{
				Name: "status_created_1",
				Keys: map[string]int{
					"status":     1,
					"created_at": 1,
				},
				Sparse: true,
			},
		},
		{
			name: "TTL index",
			index: IndexInfo{
				Name: "session_expires_1",
				Keys: map[string]int{"expires_at": 1},
				TTL:  &[]int{1800}[0], // 30 minutes
			},
		},
		{
			name: "partial index",
			index: IndexInfo{
				Name:          "active_users_email_1",
				Keys:          map[string]int{"email": 1},
				Unique:        true,
				PartialFilter: map[string]interface{}{"active": true},
			},
		},
		{
			name: "text index",
			index: IndexInfo{
				Name: "content_text",
				Keys: map[string]interface{}{
					"title":       "text",
					"description": "text",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := tt.index

			assert.NotEmpty(t, index.Name)
			assert.NotNil(t, index.Keys)

			// Test TTL specific logic
			if index.TTL != nil {
				assert.Greater(t, *index.TTL, 0)
			}

			// Test partial filter logic
			if index.PartialFilter != nil {
				assert.NotEmpty(t, index.PartialFilter)
			}
		})
	}
}

// TestClusterDatabaseResult tests the result structure
func TestClusterDatabaseResult(t *testing.T) {
	result := clusterDatabaseResult{
		ClusterName: "test-cluster",
		Databases: []DatabaseInfo{
			{Name: "db1", ClusterName: "test-cluster"},
			{Name: "db2", ClusterName: "test-cluster"},
		},
		Error: nil,
	}

	assert.Equal(t, "test-cluster", result.ClusterName)
	assert.Len(t, result.Databases, 2)
	assert.NoError(t, result.Error)

	// Test error case
	errorResult := clusterDatabaseResult{
		ClusterName: "error-cluster",
		Databases:   nil,
		Error:       assert.AnError,
	}

	assert.Equal(t, "error-cluster", errorResult.ClusterName)
	assert.Nil(t, errorResult.Databases)
	assert.Error(t, errorResult.Error)
}

// BenchmarkIsSystemDatabase benchmarks the system database check
func BenchmarkIsSystemDatabase(b *testing.B) {
	enumerator := &DatabaseEnumerator{}

	testCases := []string{
		"admin",
		"local",
		"config",
		"myapp",
		"production",
		"analytics",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dbName := testCases[i%len(testCases)]
		enumerator.isSystemDatabase(dbName)
	}
}

// BenchmarkIsSystemCollection benchmarks the system collection check
func BenchmarkIsSystemCollection(b *testing.B) {
	enumerator := &DatabaseEnumerator{}

	testCases := []string{
		"system.users",
		"system.indexes",
		"fs.files",
		"fs.chunks",
		"users",
		"products",
		"orders",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collName := testCases[i%len(testCases)]
		enumerator.isSystemCollection(collName)
	}
}

// TestDatabaseEnumerator_NilClient tests behavior with nil client
func TestDatabaseEnumerator_NilClient(t *testing.T) {
	opts := &DiscoverOptions{
		MaxConcurrency: 1,
		Timeout:        5 * time.Second,
		Verbose:        false,
	}

	enumerator := NewDatabaseEnumerator(nil, opts)
	require.NotNil(t, enumerator)
	assert.Nil(t, enumerator.atlasClient)

	// Test that the enumerator is created but would fail on actual operations
	assert.Equal(t, 1, enumerator.maxConcurrency)
	assert.Equal(t, 5*time.Second, enumerator.timeout)
	assert.False(t, enumerator.verbose)
}

// TestDatabaseEnumerationWorkflow tests the complete workflow with mock data
func TestDatabaseEnumerationWorkflow(t *testing.T) {
	// This test validates the data structures and workflow without actual MongoDB connections

	// Mock cluster data
	clusters := []types.ClusterManifest{
		{
			Metadata: types.ResourceMetadata{
				Name: "cluster0",
				Labels: map[string]string{
					"atlas.mongodb.com/project-id": "507f1f77bcf86cd799439011",
				},
			},
			Spec: types.ClusterSpec{
				ClusterType: "REPLICASET",
			},
		},
		{
			Metadata: types.ResourceMetadata{
				Name: "cluster1",
				Labels: map[string]string{
					"atlas.mongodb.com/project-id": "507f1f77bcf86cd799439011",
				},
			},
			Spec: types.ClusterSpec{
				ClusterType: "SHARDED",
			},
		},
	}

	// Test connection string generation
	enumerator := &DatabaseEnumerator{}
	for _, cluster := range clusters {
		connStr, err := enumerator.getClusterConnectionString(cluster)
		assert.NoError(t, err)
		assert.Contains(t, connStr, cluster.Metadata.Name)
		assert.Contains(t, connStr, "mongodb+srv://")
	}

	// Test system filtering
	testDatabases := []string{"admin", "local", "config", "myapp", "analytics"}
	userDatabases := make([]string, 0)
	for _, db := range testDatabases {
		if !enumerator.isSystemDatabase(db) {
			userDatabases = append(userDatabases, db)
		}
	}
	assert.Equal(t, []string{"myapp", "analytics"}, userDatabases)

	testCollections := []string{"system.users", "fs.files", "users", "products", "orders"}
	userCollections := make([]string, 0)
	for _, coll := range testCollections {
		if !enumerator.isSystemCollection(coll) {
			userCollections = append(userCollections, coll)
		}
	}
	assert.Equal(t, []string{"users", "products", "orders"}, userCollections)
}
