package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/types"
)

// MockClient is a mock implementation of the MongoDB client for testing
type MockClient struct {
	logger *logging.Logger

	// Mock data
	databases   []types.DatabaseInfo
	collections map[string][]types.CollectionInfo // keyed by database name
	stats       map[string]*types.CollectionStats // keyed by "db.collection"

	// Error simulation
	shouldErrorOnConnect    bool
	shouldErrorOnListDB     bool
	shouldErrorOnListColl   bool
	shouldErrorOnCreateColl bool
	shouldErrorOnDropColl   bool
	shouldErrorOnGetStats   bool
}

// NewMockClient creates a new mock MongoDB client
func NewMockClient(logger *logging.Logger) *MockClient {
	if logger == nil {
		logger = logging.Default()
	}

	return &MockClient{
		logger:      logger,
		databases:   []types.DatabaseInfo{},
		collections: make(map[string][]types.CollectionInfo),
		stats:       make(map[string]*types.CollectionStats),
	}
}

// SetMockDatabases sets the mock databases that will be returned by ListDatabases
func (mc *MockClient) SetMockDatabases(databases []types.DatabaseInfo) {
	mc.databases = databases
}

// SetMockCollections sets the mock collections for a specific database
func (mc *MockClient) SetMockCollections(dbName string, collections []types.CollectionInfo) {
	mc.collections[dbName] = collections
}

// SetMockStats sets the mock statistics for a specific collection
func (mc *MockClient) SetMockStats(dbName, collectionName string, stats *types.CollectionStats) {
	key := dbName + "." + collectionName
	mc.stats[key] = stats
}

// SetError configures the mock to return errors for specific operations
func (mc *MockClient) SetError(operation string, shouldError bool) {
	switch operation {
	case "connect":
		mc.shouldErrorOnConnect = shouldError
	case "listdb":
		mc.shouldErrorOnListDB = shouldError
	case "listcoll":
		mc.shouldErrorOnListColl = shouldError
	case "createcoll":
		mc.shouldErrorOnCreateColl = shouldError
	case "dropcoll":
		mc.shouldErrorOnDropColl = shouldError
	case "stats":
		mc.shouldErrorOnGetStats = shouldError
	}
}

// Close simulates closing the connection
func (mc *MockClient) Close(ctx context.Context) error {
	mc.logger.Info("Mock client closed")
	return nil
}

// ListDatabases returns the mock databases
func (mc *MockClient) ListDatabases(ctx context.Context) ([]types.DatabaseInfo, error) {
	if mc.shouldErrorOnListDB {
		return nil, &MockError{Operation: "listDatabases", Message: "mock error"}
	}

	mc.logger.Debug("Mock listed databases", "count", len(mc.databases))
	return mc.databases, nil
}

// ListCollections returns the mock collections for the specified database
func (mc *MockClient) ListCollections(ctx context.Context, dbName string) ([]types.CollectionInfo, error) {
	if mc.shouldErrorOnListColl {
		return nil, &MockError{Operation: "listCollections", Message: "mock error"}
	}

	collections, exists := mc.collections[dbName]
	if !exists {
		collections = []types.CollectionInfo{}
	}

	mc.logger.Debug("Mock listed collections",
		"database", dbName,
		"count", len(collections))

	return collections, nil
}

// CreateCollection simulates creating a collection
func (mc *MockClient) CreateCollection(ctx context.Context, dbName, collectionName string, opts *options.CreateCollectionOptions) error {
	if mc.shouldErrorOnCreateColl {
		return &MockError{Operation: "createCollection", Message: "mock error"}
	}

	// Add the collection to our mock data
	if mc.collections[dbName] == nil {
		mc.collections[dbName] = []types.CollectionInfo{}
	}

	newCollection := types.CollectionInfo{
		Name: collectionName,
		Type: "collection",
	}

	mc.collections[dbName] = append(mc.collections[dbName], newCollection)

	mc.logger.Info("Mock created collection",
		"database", dbName,
		"collection", collectionName)

	return nil
}

// DropCollection simulates dropping a collection
func (mc *MockClient) DropCollection(ctx context.Context, dbName, collectionName string) error {
	if mc.shouldErrorOnDropColl {
		return &MockError{Operation: "dropCollection", Message: "mock error"}
	}

	// Remove the collection from our mock data
	if collections, exists := mc.collections[dbName]; exists {
		for i, coll := range collections {
			if coll.Name == collectionName {
				mc.collections[dbName] = append(collections[:i], collections[i+1:]...)
				break
			}
		}
	}

	// Remove stats for the collection
	key := dbName + "." + collectionName
	delete(mc.stats, key)

	mc.logger.Info("Mock dropped collection",
		"database", dbName,
		"collection", collectionName)

	return nil
}

// GetCollectionStats returns mock statistics for the specified collection
func (mc *MockClient) GetCollectionStats(ctx context.Context, dbName, collectionName string) (*types.CollectionStats, error) {
	if mc.shouldErrorOnGetStats {
		return nil, &MockError{Operation: "getCollectionStats", Message: "mock error"}
	}

	key := dbName + "." + collectionName
	if stats, exists := mc.stats[key]; exists {
		return stats, nil
	}

	// Return default stats if none are set
	return &types.CollectionStats{
		Count:      0,
		Size:       0,
		AvgObjSize: 0,
		IndexSizes: map[string]int64{},
	}, nil
}

// Ping simulates testing the connection
func (mc *MockClient) Ping(ctx context.Context) error {
	if mc.shouldErrorOnConnect {
		return &MockError{Operation: "ping", Message: "mock connection error"}
	}
	return nil
}

// GetUnderlyingClient returns nil for the mock client as there's no real underlying client
func (mc *MockClient) GetUnderlyingClient() interface{} {
	return nil
}

// MockError represents an error returned by the mock client
type MockError struct {
	Operation string
	Message   string
}

func (e *MockError) Error() string {
	return e.Operation + ": " + e.Message
}

// NewMockClientWithData creates a mock client pre-populated with test data
func NewMockClientWithData(logger *logging.Logger) *MockClient {
	client := NewMockClient(logger)

	// Add some sample databases
	client.SetMockDatabases([]types.DatabaseInfo{
		{
			Name:       "testdb1",
			SizeOnDisk: 1024000,
			Empty:      false,
		},
		{
			Name:       "testdb2",
			SizeOnDisk: 512000,
			Empty:      false,
		},
		{
			Name:       "admin",
			SizeOnDisk: 0,
			Empty:      true,
		},
	})

	// Add some sample collections
	client.SetMockCollections("testdb1", []types.CollectionInfo{
		{
			Name: "users",
			Type: "collection",
		},
		{
			Name: "orders",
			Type: "collection",
		},
	})

	client.SetMockCollections("testdb2", []types.CollectionInfo{
		{
			Name: "products",
			Type: "collection",
		},
	})

	// Add some sample stats
	client.SetMockStats("testdb1", "users", &types.CollectionStats{
		Count:      1000,
		Size:       512000,
		AvgObjSize: 512,
		IndexSizes: map[string]int64{
			"_id_":        10240,
			"email_index": 5120,
		},
	})

	client.SetMockStats("testdb1", "orders", &types.CollectionStats{
		Count:      250,
		Size:       128000,
		AvgObjSize: 512,
		IndexSizes: map[string]int64{
			"_id_":          2560,
			"user_id_index": 1280,
			"status_index":  640,
		},
	})

	return client
}
