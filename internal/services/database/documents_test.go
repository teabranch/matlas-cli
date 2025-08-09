package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestNewDocumentService(t *testing.T) {
	dbService := NewService(zap.NewNop())

	tests := []struct {
		name      string
		dbService *Service
		logger    *zap.Logger
	}{
		{
			name:      "with logger",
			dbService: dbService,
			logger:    zap.NewNop(),
		},
		{
			name:      "nil logger",
			dbService: dbService,
			logger:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docService := NewDocumentService(tt.dbService, tt.logger)

			require.NotNil(t, docService)
			assert.Equal(t, tt.dbService, docService.dbService)
			assert.NotNil(t, docService.logger)
		})
	}
}

func TestDocumentService_InsertDocument_Validation(t *testing.T) {
	dbService := NewService(zap.NewNop())
	docService := NewDocumentService(dbService, zap.NewNop())
	ctx := context.Background()

	connInfo := &types.ConnectionInfo{
		ConnectionString: "mongodb://localhost:27017",
	}

	tests := []struct {
		name           string
		databaseName   string
		collectionName string
		document       map[string]interface{}
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "empty database name",
			databaseName:   "",
			collectionName: "test_collection",
			document:       map[string]interface{}{"field": "value"},
			wantErr:        true,
			errMsg:         "database name is required",
		},
		{
			name:           "empty collection name",
			databaseName:   "test_db",
			collectionName: "",
			document:       map[string]interface{}{"field": "value"},
			wantErr:        true,
			errMsg:         "collection name is required",
		},
		{
			name:           "nil document",
			databaseName:   "test_db",
			collectionName: "test_collection",
			document:       nil,
			wantErr:        true,
			errMsg:         "document is required",
		},
		// Note: Removed connection test to avoid network calls in unit tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := docService.InsertDocument(ctx, connInfo, tt.databaseName, tt.collectionName, tt.document)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, id)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, id)
			}
		})
	}
}

func TestDocumentService_FindDocuments_Validation(t *testing.T) {
	dbService := NewService(zap.NewNop())
	docService := NewDocumentService(dbService, zap.NewNop())
	ctx := context.Background()

	connInfo := &types.ConnectionInfo{
		ConnectionString: "mongodb://localhost:27017",
	}

	tests := []struct {
		name           string
		databaseName   string
		collectionName string
		filter         map[string]interface{}
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "empty database name",
			databaseName:   "",
			collectionName: "test_collection",
			filter:         map[string]interface{}{"field": "value"},
			wantErr:        true,
			errMsg:         "database name is required",
		},
		{
			name:           "empty collection name",
			databaseName:   "test_db",
			collectionName: "",
			filter:         map[string]interface{}{"field": "value"},
			wantErr:        true,
			errMsg:         "collection name is required",
		},
		// Note: Removed connection tests to avoid network calls in unit tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs, err := docService.FindDocuments(ctx, connInfo, tt.databaseName, tt.collectionName, tt.filter, 0)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, docs)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, docs)
			}
		})
	}
}

func TestDocumentService_FindDocumentByID_Validation(t *testing.T) {
	dbService := NewService(zap.NewNop())
	docService := NewDocumentService(dbService, zap.NewNop())
	ctx := context.Background()

	connInfo := &types.ConnectionInfo{
		ConnectionString: "mongodb://localhost:27017",
	}

	validObjectID := primitive.NewObjectID()

	tests := []struct {
		name           string
		databaseName   string
		collectionName string
		id             primitive.ObjectID
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "empty database name",
			databaseName:   "",
			collectionName: "test_collection",
			id:             validObjectID,
			wantErr:        true,
			errMsg:         "database name is required",
		},
		{
			name:           "empty collection name",
			databaseName:   "test_db",
			collectionName: "",
			id:             validObjectID,
			wantErr:        true,
			errMsg:         "collection name is required",
		},
		// Note: Removed connection tests to avoid network calls in unit tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := docService.FindDocumentByID(ctx, connInfo, tt.databaseName, tt.collectionName, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, doc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, doc)
			}
		})
	}
}

func TestDocumentService_UpdateDocument_Validation(t *testing.T) {
	dbService := NewService(zap.NewNop())
	docService := NewDocumentService(dbService, zap.NewNop())
	ctx := context.Background()

	connInfo := &types.ConnectionInfo{
		ConnectionString: "mongodb://localhost:27017",
	}

	validObjectID := primitive.NewObjectID()

	tests := []struct {
		name           string
		databaseName   string
		collectionName string
		id             primitive.ObjectID
		update         map[string]interface{}
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "empty database name",
			databaseName:   "",
			collectionName: "test_collection",
			id:             validObjectID,
			update:         map[string]interface{}{"field": "new_value"},
			wantErr:        true,
			errMsg:         "database name is required",
		},
		{
			name:           "empty collection name",
			databaseName:   "test_db",
			collectionName: "",
			id:             validObjectID,
			update:         map[string]interface{}{"field": "new_value"},
			wantErr:        true,
			errMsg:         "collection name is required",
		},
		{
			name:           "nil update",
			databaseName:   "test_db",
			collectionName: "test_collection",
			id:             validObjectID,
			update:         nil,
			wantErr:        true,
			errMsg:         "update is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := docService.UpdateDocument(ctx, connInfo, tt.databaseName, tt.collectionName, tt.id, tt.update)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDocumentService_DeleteDocument_Validation(t *testing.T) {
	dbService := NewService(zap.NewNop())
	docService := NewDocumentService(dbService, zap.NewNop())
	ctx := context.Background()

	connInfo := &types.ConnectionInfo{
		ConnectionString: "mongodb://localhost:27017",
	}

	validObjectID := primitive.NewObjectID()

	tests := []struct {
		name           string
		databaseName   string
		collectionName string
		id             primitive.ObjectID
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "empty database name",
			databaseName:   "",
			collectionName: "test_collection",
			id:             validObjectID,
			wantErr:        true,
			errMsg:         "database name is required",
		},
		{
			name:           "empty collection name",
			databaseName:   "test_db",
			collectionName: "",
			id:             validObjectID,
			wantErr:        true,
			errMsg:         "collection name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := docService.DeleteDocument(ctx, connInfo, tt.databaseName, tt.collectionName, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDocumentService_CountDocuments_Validation(t *testing.T) {
	dbService := NewService(zap.NewNop())
	docService := NewDocumentService(dbService, zap.NewNop())
	ctx := context.Background()

	connInfo := &types.ConnectionInfo{
		ConnectionString: "mongodb://localhost:27017",
	}

	tests := []struct {
		name           string
		databaseName   string
		collectionName string
		filter         map[string]interface{}
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "empty database name",
			databaseName:   "",
			collectionName: "test_collection",
			filter:         map[string]interface{}{"field": "value"},
			wantErr:        true,
			errMsg:         "database name is required",
		},
		{
			name:           "empty collection name",
			databaseName:   "test_db",
			collectionName: "",
			filter:         map[string]interface{}{"field": "value"},
			wantErr:        true,
			errMsg:         "collection name is required",
		},
		// Note: Removed connection test to avoid network calls in unit tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := docService.CountDocuments(ctx, connInfo, tt.databaseName, tt.collectionName, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Equal(t, int64(0), count)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, count, int64(0))
			}
		})
	}
}

func TestDocument_Structure(t *testing.T) {
	// Test Document struct
	objectID := primitive.NewObjectID()
	doc := Document{
		ID: objectID,
		Data: map[string]interface{}{
			"field1": "value1",
			"field2": 42,
			"field3": true,
		},
	}

	assert.Equal(t, objectID, doc.ID)
	assert.Equal(t, "value1", doc.Data["field1"])
	assert.Equal(t, 42, doc.Data["field2"])
	assert.Equal(t, true, doc.Data["field3"])
}

func TestDocument_EmptyDocument(t *testing.T) {
	// Test empty Document
	doc := Document{}

	assert.True(t, doc.ID.IsZero())
	assert.Nil(t, doc.Data)
}

func TestDocument_WithData(t *testing.T) {
	// Test Document with data but no ID
	doc := Document{
		Data: map[string]interface{}{
			"name":   "test",
			"count":  100,
			"active": true,
			"tags":   []string{"tag1", "tag2"},
			"nested": map[string]interface{}{"inner": "value"},
		},
	}

	assert.True(t, doc.ID.IsZero())
	assert.NotNil(t, doc.Data)
	assert.Equal(t, "test", doc.Data["name"])
	assert.Equal(t, 100, doc.Data["count"])
	assert.Equal(t, true, doc.Data["active"])

	tags, ok := doc.Data["tags"].([]string)
	assert.True(t, ok)
	assert.Equal(t, []string{"tag1", "tag2"}, tags)

	nested, ok := doc.Data["nested"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "value", nested["inner"])
}
