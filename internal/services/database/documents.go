package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/teabranch/matlas-cli/internal/types"
)

// DocumentService provides CRUD operations for MongoDB documents
type DocumentService struct {
	dbService *Service
	logger    *zap.Logger
}

// NewDocumentService creates a new document service
func NewDocumentService(dbService *Service, logger *zap.Logger) *DocumentService {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &DocumentService{
		dbService: dbService,
		logger:    logger,
	}
}

// Document represents a MongoDB document
type Document struct {
	ID   primitive.ObjectID     `json:"_id,omitempty" bson:"_id,omitempty"`
	Data map[string]interface{} `json:"data,omitempty" bson:",inline"`
}

// InsertDocument inserts a single document into the specified collection
func (ds *DocumentService) InsertDocument(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string, document map[string]interface{}) (*primitive.ObjectID, error) {
	if databaseName == "" {
		return nil, fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}
	if document == nil {
		return nil, fmt.Errorf("document is required")
	}

	client, err := ds.dbService.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return nil, err
	}

	collection := client.GetUnderlyingClient().Database(databaseName).Collection(collectionName)

	result, err := collection.InsertOne(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	objectID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("unexpected ID type: %T", result.InsertedID)
	}

	ds.logger.Debug("Inserted document",
		zap.String("database", databaseName),
		zap.String("collection", collectionName),
		zap.String("id", objectID.Hex()))

	return &objectID, nil
}

// FindDocuments finds documents matching the given filter
func (ds *DocumentService) FindDocuments(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string, filter map[string]interface{}, limit int64) ([]Document, error) {
	if databaseName == "" {
		return nil, fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	client, err := ds.dbService.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return nil, err
	}

	collection := client.GetUnderlyingClient().Database(databaseName).Collection(collectionName)

	// Convert filter to BSON
	bsonFilter := bson.M{}
	if filter != nil {
		for k, v := range filter {
			bsonFilter[k] = v
		}
	}

	// Set up find options
	findOptions := options.Find()
	if limit > 0 {
		findOptions.SetLimit(limit)
	}

	cursor, err := collection.Find(ctx, bsonFilter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}
	defer cursor.Close(ctx)

	var documents []Document
	for cursor.Next(ctx) {
		var doc Document
		if err := cursor.Decode(&doc); err != nil {
			ds.logger.Warn("Failed to decode document", zap.Error(err))
			continue
		}
		documents = append(documents, doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	ds.logger.Debug("Found documents",
		zap.String("database", databaseName),
		zap.String("collection", collectionName),
		zap.Int("count", len(documents)))

	return documents, nil
}

// FindDocumentByID finds a single document by its ObjectID
func (ds *DocumentService) FindDocumentByID(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string, id primitive.ObjectID) (*Document, error) {
	if databaseName == "" {
		return nil, fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}

	client, err := ds.dbService.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return nil, err
	}

	collection := client.GetUnderlyingClient().Database(databaseName).Collection(collectionName)

	var doc Document
	err = collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	ds.logger.Debug("Found document by ID",
		zap.String("database", databaseName),
		zap.String("collection", collectionName),
		zap.String("id", id.Hex()))

	return &doc, nil
}

// UpdateDocument updates a single document by its ObjectID
func (ds *DocumentService) UpdateDocument(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string, id primitive.ObjectID, update map[string]interface{}) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}
	if update == nil {
		return fmt.Errorf("update is required")
	}

	client, err := ds.dbService.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return err
	}

	collection := client.GetUnderlyingClient().Database(databaseName).Collection(collectionName)

	// Prepare update document
	updateDoc := bson.M{"$set": update}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": id}, updateDoc)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("document with ID %s not found", id.Hex())
	}

	ds.logger.Debug("Updated document",
		zap.String("database", databaseName),
		zap.String("collection", collectionName),
		zap.String("id", id.Hex()),
		zap.Int64("modified_count", result.ModifiedCount))

	return nil
}

// DeleteDocument deletes a single document by its ObjectID
func (ds *DocumentService) DeleteDocument(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string, id primitive.ObjectID) error {
	if databaseName == "" {
		return fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}

	client, err := ds.dbService.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return err
	}

	collection := client.GetUnderlyingClient().Database(databaseName).Collection(collectionName)

	result, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("document with ID %s not found", id.Hex())
	}

	ds.logger.Debug("Deleted document",
		zap.String("database", databaseName),
		zap.String("collection", collectionName),
		zap.String("id", id.Hex()))

	return nil
}

// CountDocuments counts documents matching the given filter
func (ds *DocumentService) CountDocuments(ctx context.Context, connInfo *types.ConnectionInfo, databaseName, collectionName string, filter map[string]interface{}) (int64, error) {
	if databaseName == "" {
		return 0, fmt.Errorf("database name is required")
	}
	if collectionName == "" {
		return 0, fmt.Errorf("collection name is required")
	}

	client, err := ds.dbService.GetOrCreateClient(ctx, connInfo)
	if err != nil {
		return 0, err
	}

	collection := client.GetUnderlyingClient().Database(databaseName).Collection(collectionName)

	// Convert filter to BSON
	bsonFilter := bson.M{}
	if filter != nil {
		for k, v := range filter {
			bsonFilter[k] = v
		}
	}

	count, err := collection.CountDocuments(ctx, bsonFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	ds.logger.Debug("Counted documents",
		zap.String("database", databaseName),
		zap.String("collection", collectionName),
		zap.Int64("count", count))

	return count, nil
}
