package mongodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/types"
)

func TestClient_ListDatabases_ErrorPath(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	// Configure mock to return error
	client.SetError("listdb", true)

	ctx := context.Background()

	// Test that method handles error gracefully
	dbs, err := client.ListDatabases(ctx)

	assert.Error(t, err)
	assert.Nil(t, dbs)
}

func TestClient_ListCollections_ErrorPath(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	// Configure mock to return error
	client.SetError("listcoll", true)

	ctx := context.Background()

	// Test that method handles error gracefully
	collections, err := client.ListCollections(ctx, "testdb")

	assert.Error(t, err)
	assert.Nil(t, collections)
}

func TestClient_CreateCollection_ErrorPath(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	// Configure mock to return error
	client.SetError("createcoll", true)

	ctx := context.Background()

	// Test that method handles error gracefully
	err := client.CreateCollection(ctx, "testdb", "testcoll", nil)

	assert.Error(t, err)
}

func TestClient_DropCollection_ErrorPath(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	// Configure mock to return error
	client.SetError("dropcoll", true)

	ctx := context.Background()

	// Test that method handles error gracefully
	err := client.DropCollection(ctx, "testdb", "testcoll")

	assert.Error(t, err)
}

func TestClient_GetCollectionStats_ErrorPath(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	// Configure mock to return error
	client.SetError("stats", true)

	ctx := context.Background()

	// Test that method handles error gracefully
	stats, err := client.GetCollectionStats(ctx, "testdb", "testcoll")

	assert.Error(t, err)
	assert.Nil(t, stats)
}

func TestClient_Ping_ErrorPath(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	// Configure mock to return error
	client.SetError("connect", true)

	ctx := context.Background()

	// Test that method handles error gracefully
	err := client.Ping(ctx)

	assert.Error(t, err)
}

func TestMockClient_SuccessPath(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	// Test successful operations with proper types
	client.SetMockDatabases([]types.DatabaseInfo{
		{Name: "db1", SizeOnDisk: 1024},
		{Name: "db2", SizeOnDisk: 2048},
		{Name: "db3", SizeOnDisk: 4096},
	})

	client.SetMockCollections("db1", []types.CollectionInfo{
		{Name: "coll1", Type: "collection"},
		{Name: "coll2", Type: "collection"},
	})

	client.SetMockStats("db1", "coll1", &types.CollectionStats{
		Count: 100,
		Size:  1024,
	})

	ctx := context.Background()

	// Test successful database listing
	dbs, err := client.ListDatabases(ctx)
	assert.NoError(t, err)
	assert.Len(t, dbs, 3)
	assert.Equal(t, "db1", dbs[0].Name)
	assert.Equal(t, "db2", dbs[1].Name)
	assert.Equal(t, "db3", dbs[2].Name)
	assert.Equal(t, int64(1024), dbs[0].SizeOnDisk)

	// Test successful collection listing
	collections, err := client.ListCollections(ctx, "db1")
	assert.NoError(t, err)
	assert.Len(t, collections, 2)
	assert.Equal(t, "coll1", collections[0].Name)
	assert.Equal(t, "coll2", collections[1].Name)
	assert.Equal(t, "collection", collections[0].Type)

	// Test successful stats retrieval
	stats, err := client.GetCollectionStats(ctx, "db1", "coll1")
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(100), stats.Count)
	assert.Equal(t, int64(1024), stats.Size)

	// Test successful ping
	err = client.Ping(ctx)
	assert.NoError(t, err)

	// Test successful collection operations (should not error when no error flag set)
	err = client.CreateCollection(ctx, "testdb", "testcoll", nil)
	assert.NoError(t, err)

	err = client.DropCollection(ctx, "testdb", "testcoll")
	assert.NoError(t, err)
}

func TestMockClient_EmptyCollections(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	ctx := context.Background()

	// Test listing collections for non-existent database
	collections, err := client.ListCollections(ctx, "non-existent-db")
	assert.NoError(t, err)
	assert.Empty(t, collections)
}

func TestMockClient_DefaultStats(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	ctx := context.Background()

	// Test getting stats for collection without pre-set stats
	stats, err := client.GetCollectionStats(ctx, "testdb", "testcoll")
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.Count)
	assert.Equal(t, int64(0), stats.Size)
	assert.Equal(t, int64(0), stats.AvgObjSize)
	assert.NotNil(t, stats.IndexSizes)
}

func TestMockClient_GetUnderlyingClient(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	// Test that mock client returns nil for underlying client
	underlying := client.GetUnderlyingClient()
	assert.Nil(t, underlying)
}

func TestMockClient_Close(t *testing.T) {
	logger := logging.Default()
	client := NewMockClient(logger)

	ctx := context.Background()

	// Test that close doesn't error
	err := client.Close(ctx)
	assert.NoError(t, err)
}

func TestMockError_Error(t *testing.T) {
	err := &MockError{
		Operation: "testOperation",
		Message:   "test error message",
	}

	assert.Equal(t, "testOperation: test error message", err.Error())
}

func TestNewMockClient_WithNilLogger(t *testing.T) {
	// Test that NewMockClient handles nil logger gracefully
	client := NewMockClient(nil)

	assert.NotNil(t, client)
	assert.NotNil(t, client.logger) // Should be set to logging.Default()
}
