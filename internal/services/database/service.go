package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/teabranch/matlas-cli/internal/clients/mongodb"
	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/types"
)

// Service provides database operations
type Service struct {
	clients map[string]*mongodb.Client // keyed by connection string
	logger  *logging.Logger
}

// NewService creates a new database service
func NewService(logger *logging.Logger) *Service {
	if logger == nil {
		logger = logging.Default()
	}

	return &Service{
		clients: make(map[string]*mongodb.Client),
		logger:  logger,
	}
}

// GetOrCreateClient returns an existing client or creates a new one for the given connection info
func (s *Service) GetOrCreateClient(ctx context.Context, connInfo *types.ConnectionInfo) (*mongodb.Client, error) {
	if connInfo == nil || connInfo.ConnectionString == "" {
		return nil, fmt.Errorf("connection info is required")
	}

	// Check if we already have a client for this connection string
	if client, exists := s.clients[connInfo.ConnectionString]; exists {
		// Test the connection to make sure it's still valid
		if err := client.Ping(ctx); err == nil {
			return client, nil
		}
		// Connection is stale, remove it
		_ = client.Close(ctx)
		delete(s.clients, connInfo.ConnectionString)
	}

	// Create a new client
	config := mongodb.DefaultClientConfig()
	config.ConnectionString = connInfo.ConnectionString

	// Apply any additional options from ConnectionInfo
	if connInfo.Options != nil {
		if timeout, ok := connInfo.Options["connectTimeout"]; ok {
			// In a real implementation, parse the timeout string
			_ = timeout
		}
		if maxPoolSize, ok := connInfo.Options["maxPoolSize"]; ok {
			// In a real implementation, parse the maxPoolSize string
			_ = maxPoolSize
		}
	}

	client, err := mongodb.NewClient(ctx, config, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Cache the client
	s.clients[connInfo.ConnectionString] = client

	s.logger.Info("Created new MongoDB client",
		"connection_string", connInfo.ConnectionString[:20]+"...")

	return client, nil
}

// ListDatabases lists all databases for the given connection
func (s *Service) ListDatabases(ctx context.Context, connInfo *types.ConnectionInfo) ([]types.DatabaseInfo, error) {
	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return nil, err
	}

	databases, err := client.ListDatabases(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	s.logger.Debug("Listed databases",
		"count", len(databases),
		"connection", connInfo.ConnectionString[:20]+"...")

	return databases, nil
}

// ListCollections lists all collections in the specified database
func (s *Service) ListCollections(ctx context.Context, connInfo *types.ConnectionInfo, databaseName string) ([]types.CollectionInfo, error) {
	if databaseName == "" {
		return nil, fmt.Errorf("database name is required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return nil, err
	}

	collections, err := client.ListCollections(ctx, databaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	// Enrich collections with statistics
	for i := range collections {
		stats, err := client.GetCollectionStats(ctx, databaseName, collections[i].Name)
		if err != nil {
			s.logger.Warn("Failed to get collection stats",
				"database", databaseName,
				"collection", collections[i].Name,
				"error", err.Error())
			continue
		}
		collections[i].Info = *stats
	}

	s.logger.Debug("Listed collections",
		"database", databaseName,
		"count", len(collections))

	return collections, nil
}

// CreateCollection creates a new collection in the specified database
func (s *Service) CreateCollection(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string, opts map[string]interface{}) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return err
	}

	// Convert options map to MongoDB options
	var createOpts *options.CreateCollectionOptions
	if opts != nil {
		createOpts = options.CreateCollection()

		// Handle common options
		if capped, ok := opts["capped"].(bool); ok && capped {
			createOpts.SetCapped(true)
			if size, ok := opts["size"].(int64); ok {
				createOpts.SetSizeInBytes(size)
			}
			if max, ok := opts["max"].(int64); ok {
				createOpts.SetMaxDocuments(max)
			}
		}

		// Add other options as needed
	}

	if err := client.CreateCollection(ctx, databaseName, collectionName, createOpts); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	s.logger.Info("Created collection",
		"database", databaseName,
		"collection", collectionName)

	return nil
}

// DropCollection drops a collection from the specified database
func (s *Service) DropCollection(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return err
	}

	if err := client.DropCollection(ctx, databaseName, collectionName); err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}

	s.logger.Info("Dropped collection",
		"database", databaseName,
		"collection", collectionName)

	return nil
}

// GetCollectionStats retrieves statistics for a specific collection
func (s *Service) GetCollectionStats(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string) (*types.CollectionStats, error) {
	if databaseName == "" {
		return nil, fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return nil, err
	}

	stats, err := client.GetCollectionStats(ctx, databaseName, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	return stats, nil
}

// CreateDatabase creates a new database by creating a collection in it
// Note: MongoDB creates databases lazily when first collection is created
// DEPRECATED: Use CreateDatabaseWithCollection instead
func (s *Service) CreateDatabase(ctx context.Context, connInfo *types.ConnectionInfo, databaseName string) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return err
	}

	// MongoDB creates databases implicitly when first collection is created
	// We'll create a temporary collection and then drop it to ensure the database exists
	tempCollectionName := "__temp_collection_for_db_creation"

	if err := client.CreateCollection(ctx, databaseName, tempCollectionName, nil); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Drop the temporary collection
	if err := client.DropCollection(ctx, databaseName, tempCollectionName); err != nil {
		s.logger.Warn("Failed to drop temporary collection",
			"database", databaseName,
			"collection", tempCollectionName,
			"error", err.Error())
	}

	s.logger.Info("Created database",
		"database", databaseName)

	return nil
}

// CreateDatabaseWithCollection creates a new database with a specific collection
// This ensures the database is visible in Atlas UI and has a persistent collection
func (s *Service) CreateDatabaseWithCollection(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return err
	}

	// MongoDB creates databases implicitly when first collection is created
	// Create the specified collection to ensure the database exists and is visible
	if err := client.CreateCollection(ctx, databaseName, collectionName, nil); err != nil {
		return fmt.Errorf("failed to create database with collection: %w", err)
	}

	s.logger.Info("Created database with collection",
		"database", databaseName,
		"collection", collectionName)

	return nil
}

// DropDatabase drops an entire database
func (s *Service) DropDatabase(ctx context.Context, connInfo *types.ConnectionInfo, databaseName string) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return err
	}

	if err := client.DropDatabase(ctx, databaseName); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	s.logger.Info("Dropped database",
		"database", databaseName)

	return nil
}

// CreateIndex creates an index on a collection
func (s *Service) CreateIndex(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string, keys map[string]int, opts map[string]interface{}) (string, error) {
	if databaseName == "" {
		return "", fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return "", fmt.Errorf("collection name is required")
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("index keys are required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return "", err
	}

	indexName, err := client.CreateIndex(ctx, databaseName, collectionName, keys, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create index: %w", err)
	}

	s.logger.Info("Created index",
		"database", databaseName,
		"collection", collectionName,
		"index", indexName)

	return indexName, nil
}

// DropIndex drops an index from a collection
func (s *Service) DropIndex(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName, indexName string) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}
	if indexName == "" {
		return fmt.Errorf("index name is required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return err
	}

	if err := client.DropIndex(ctx, databaseName, collectionName, indexName); err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	s.logger.Info("Dropped index",
		"database", databaseName,
		"collection", collectionName,
		"index", indexName)

	return nil
}

// ListIndexes lists all indexes for a collection
func (s *Service) ListIndexes(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string) ([]types.IndexInfo, error) {
	if databaseName == "" {
		return nil, fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	client, err := s.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return nil, err
	}

	indexes, err := client.ListIndexes(ctx, databaseName, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}

	s.logger.Debug("Listed indexes",
		"database", databaseName,
		"collection", collectionName,
		"count", len(indexes))

	return indexes, nil
}

// Close closes all cached MongoDB clients
func (s *Service) Close(ctx context.Context) error {
	var errors []error

	for connString, client := range s.clients {
		if err := client.Close(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to close client for %s: %w", connString, err))
		}
	}

	s.clients = make(map[string]*mongodb.Client)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing clients: %v", errors)
	}

	s.logger.Info("Closed all MongoDB clients")
	return nil
}
