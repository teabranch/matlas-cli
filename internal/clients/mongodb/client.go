// Package mongodb contains a light wrapper around the official MongoDB Go driver.
package mongodb

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/types"
)

// Client wraps the MongoDB driver with Atlas-specific functionality
type Client struct {
	client *mongo.Client
	logger *logging.Logger
	config *ClientConfig
}

// GetUnderlyingClient returns the underlying mongo.Client for advanced operations
func (c *Client) GetUnderlyingClient() *mongo.Client {
	return c.client
}

// ClientConfig holds configuration for the MongoDB client
type ClientConfig struct {
	ConnectionString    string
	ConnectTimeout      time.Duration
	ServerSelectTimeout time.Duration
	MaxPoolSize         uint64
	RetryWrites         bool
	ReadPreference      string
	TLSEnabled          bool
	TLSInsecure         bool
}

// DefaultClientConfig returns a default configuration suitable for Atlas
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ConnectTimeout:      30 * time.Second,
		ServerSelectTimeout: 30 * time.Second,
		MaxPoolSize:         100,
		RetryWrites:         true,
		ReadPreference:      "primary",
		TLSEnabled:          true,
		TLSInsecure:         false,
	}
}

// NewClient creates a new MongoDB client with Atlas-optimized settings
func NewClient(ctx context.Context, config *ClientConfig, logger *logging.Logger) (*Client, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	if logger == nil {
		logger = logging.Default()
	}

	clientOptions := options.Client().
		ApplyURI(config.ConnectionString).
		SetConnectTimeout(config.ConnectTimeout).
		SetServerSelectionTimeout(config.ServerSelectTimeout).
		SetMaxPoolSize(config.MaxPoolSize).
		SetRetryWrites(config.RetryWrites)

	// Configure read preference
	if config.ReadPreference != "" {
		mode, err := readpref.ModeFromString(config.ReadPreference)
		if err != nil {
			return nil, fmt.Errorf("invalid read preference %q: %w", config.ReadPreference, err)
		}
		rp, err := readpref.New(mode)
		if err != nil {
			return nil, fmt.Errorf("failed to create read preference: %w", err)
		}
		clientOptions.SetReadPreference(rp)
	}

	// Configure TLS settings for Atlas
	if config.TLSEnabled {
		// Note: Allowing TLSInsecure via configuration for local/test scenarios.
		// Avoid enabling in production environments.
		tlsConfig := &tls.Config{ //nolint:gosec // configurable for local/test use only
			InsecureSkipVerify: config.TLSInsecure, //nolint:gosec
		}
		clientOptions.SetTLSConfig(tlsConfig)
	}

	// Create the client
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	logger.Info("Successfully connected to MongoDB",
		"connection_string", maskConnectionString(config.ConnectionString))

	return &Client{
		client: client,
		logger: logger,
		config: config,
	}, nil
}

// Close disconnects the client from MongoDB
func (c *Client) Close(ctx context.Context) error {
	if c.client != nil {
		if err := c.client.Disconnect(ctx); err != nil {
			c.logger.Error("Failed to disconnect MongoDB client", "error", err.Error())
			return fmt.Errorf("failed to disconnect: %w", err)
		}
		c.logger.Info("Disconnected from MongoDB")
	}
	return nil
}

// ListDatabases returns a list of databases in the MongoDB instance
func (c *Client) ListDatabases(ctx context.Context) ([]types.DatabaseInfo, error) {
	result, err := c.client.ListDatabases(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	databases := make([]types.DatabaseInfo, 0, len(result.Databases))
	for _, db := range result.Databases {
		dbInfo := types.DatabaseInfo{
			Name:       db.Name,
			SizeOnDisk: db.SizeOnDisk,
			Empty:      db.Empty,
		}
		databases = append(databases, dbInfo)
	}

	c.logger.Debug("Listed databases", "count", len(databases))
	return databases, nil
}

// ListCollections returns a list of collections in the specified database
func (c *Client) ListCollections(ctx context.Context, dbName string) ([]types.CollectionInfo, error) {
	db := c.client.Database(dbName)

	cursor, err := db.ListCollections(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections for database %q: %w", dbName, err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var collections []types.CollectionInfo
	for cursor.Next(ctx) {
		var collInfo bson.M
		if err := cursor.Decode(&collInfo); err != nil {
			c.logger.Warn("Failed to decode collection info", "error", err.Error())
			continue
		}

		collection := types.CollectionInfo{
			Name: collInfo["name"].(string),
		}

		if collType, ok := collInfo["type"].(string); ok {
			collection.Type = collType
		}

		if options, ok := collInfo["options"].(bson.M); ok {
			collection.Options = make(map[string]interface{})
			for k, v := range options {
				collection.Options[k] = v
			}
		}

		collections = append(collections, collection)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing collections: %w", err)
	}

	c.logger.Debug("Listed collections",
		"database", dbName,
		"count", len(collections))

	return collections, nil
}

// CreateCollection creates a new collection in the specified database
func (c *Client) CreateCollection(ctx context.Context, dbName, collectionName string, opts *options.CreateCollectionOptions) error {
	db := c.client.Database(dbName)

	if err := db.CreateCollection(ctx, collectionName, opts); err != nil {
		return fmt.Errorf("failed to create collection %q in database %q: %w", collectionName, dbName, err)
	}

	c.logger.Info("Created collection",
		"database", dbName,
		"collection", collectionName)

	return nil
}

// DropCollection drops a collection from the specified database
func (c *Client) DropCollection(ctx context.Context, dbName, collectionName string) error {
	db := c.client.Database(dbName)
	collection := db.Collection(collectionName)

	if err := collection.Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop collection %q from database %q: %w", collectionName, dbName, err)
	}

	c.logger.Info("Dropped collection",
		"database", dbName,
		"collection", collectionName)

	return nil
}

// GetCollectionStats retrieves statistics for a specific collection
func (c *Client) GetCollectionStats(ctx context.Context, dbName, collectionName string) (*types.CollectionStats, error) {
	db := c.client.Database(dbName)

	var result bson.M
	err := db.RunCommand(ctx, bson.D{
		bson.E{Key: "collStats", Value: collectionName},
		bson.E{Key: "indexDetails", Value: true},
	}).Decode(&result)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats for collection %q in database %q: %w", collectionName, dbName, err)
	}

	stats := &types.CollectionStats{}

	if count, ok := result["count"].(int64); ok {
		stats.Count = count
	} else if count, ok := result["count"].(int32); ok {
		stats.Count = int64(count)
	}

	if size, ok := result["size"].(int64); ok {
		stats.Size = size
	} else if size, ok := result["size"].(int32); ok {
		stats.Size = int64(size)
	}

	if avgObjSize, ok := result["avgObjSize"].(int64); ok {
		stats.AvgObjSize = avgObjSize
	} else if avgObjSize, ok := result["avgObjSize"].(int32); ok {
		stats.AvgObjSize = int64(avgObjSize)
	}

	if indexSizes, ok := result["indexSizes"].(bson.M); ok {
		stats.IndexSizes = make(map[string]int64)
		for name, size := range indexSizes {
			if sizeInt64, ok := size.(int64); ok {
				stats.IndexSizes[name] = sizeInt64
			} else if sizeInt32, ok := size.(int32); ok {
				stats.IndexSizes[name] = int64(sizeInt32)
			}
		}
	}

	return stats, nil
}

// Ping tests the connection to MongoDB
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx, readpref.Primary())
}

// DropDatabase drops an entire database
func (c *Client) DropDatabase(ctx context.Context, dbName string) error {
	if err := c.client.Database(dbName).Drop(ctx); err != nil {
		return fmt.Errorf("failed to drop database %q: %w", dbName, err)
	}

	c.logger.Info("Dropped database", "database", dbName)
	return nil
}

// CreateIndex creates an index on a collection
func (c *Client) CreateIndex(ctx context.Context, dbName, collectionName string, keys map[string]int, opts map[string]interface{}) (string, error) {
	db := c.client.Database(dbName)
	collection := db.Collection(collectionName)

	// Convert keys map to IndexModel
	indexKeys := bson.D{}
	for field, order := range keys {
		indexKeys = append(indexKeys, bson.E{Key: field, Value: order})
	}

	indexModel := mongo.IndexModel{
		Keys: indexKeys,
	}

	// Apply options if provided
	if opts != nil {
		indexOptions := options.Index()

		if unique, ok := opts["unique"].(bool); ok {
			indexOptions.SetUnique(unique)
		}
		if sparse, ok := opts["sparse"].(bool); ok {
			indexOptions.SetSparse(sparse)
		}
		if background, ok := opts["background"].(bool); ok {
			// Background option deprecated since MongoDB 4.2; kept for backward compat.
			indexOptions.SetBackground(background) //nolint:staticcheck
		}
		if name, ok := opts["name"].(string); ok {
			indexOptions.SetName(name)
		}

		indexModel.Options = indexOptions
	}

	indexName, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return "", fmt.Errorf("failed to create index on collection %q in database %q: %w", collectionName, dbName, err)
	}

	c.logger.Info("Created index",
		"database", dbName,
		"collection", collectionName,
		"index", indexName)

	return indexName, nil
}

// DropIndex drops an index from a collection
func (c *Client) DropIndex(ctx context.Context, dbName, collectionName, indexName string) error {
	db := c.client.Database(dbName)
	collection := db.Collection(collectionName)

	if _, err := collection.Indexes().DropOne(ctx, indexName); err != nil {
		return fmt.Errorf("failed to drop index %q from collection %q in database %q: %w", indexName, collectionName, dbName, err)
	}

	c.logger.Info("Dropped index",
		"database", dbName,
		"collection", collectionName,
		"index", indexName)

	return nil
}

// ListIndexes lists all indexes for a collection
func (c *Client) ListIndexes(ctx context.Context, dbName, collectionName string) ([]types.IndexInfo, error) {
	db := c.client.Database(dbName)
	collection := db.Collection(collectionName)

	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes for collection %q in database %q: %w", collectionName, dbName, err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var indexes []types.IndexInfo
	for cursor.Next(ctx) {
		var indexBson bson.M
		if err := cursor.Decode(&indexBson); err != nil {
			c.logger.Warn("Failed to decode index info", "error", err.Error())
			continue
		}

		index := types.IndexInfo{
			Options: make(map[string]interface{}),
		}

		if name, ok := indexBson["name"].(string); ok {
			index.Name = name
		}

		if keys, ok := indexBson["key"].(bson.M); ok {
			index.Keys = make(map[string]interface{})
			for k, v := range keys {
				index.Keys[k] = v
			}
		}

		if unique, ok := indexBson["unique"].(bool); ok {
			index.Unique = unique
		}

		if sparse, ok := indexBson["sparse"].(bool); ok {
			index.Sparse = sparse
		}

		if background, ok := indexBson["background"].(bool); ok {
			index.Background = background
		}

		if version, ok := indexBson["v"].(int32); ok {
			index.Version = int(version)
		}

		// Copy other options
		for k, v := range indexBson {
			if k != "name" && k != "key" && k != "unique" && k != "sparse" && k != "background" && k != "v" {
				index.Options[k] = v
			}
		}

		indexes = append(indexes, index)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing indexes: %w", err)
	}

	c.logger.Debug("Listed indexes",
		"database", dbName,
		"collection", collectionName,
		"count", len(indexes))

	return indexes, nil
}

// maskConnectionString masks sensitive information in connection strings for logging
func maskConnectionString(connectionString string) string {
	// Simple masking - in production, use more sophisticated parsing
	if len(connectionString) > 50 {
		return connectionString[:20] + "***MASKED***" + connectionString[len(connectionString)-10:]
	}
	return "***MASKED***"
}
