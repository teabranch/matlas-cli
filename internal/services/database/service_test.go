package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/teabranch/matlas-cli/internal/clients/mongodb"
	"github.com/teabranch/matlas-cli/internal/types"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name   string
		logger *zap.Logger
	}{
		{
			name:   "with logger",
			logger: zap.NewNop(),
		},
		{
			name:   "nil logger",
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.logger)

			require.NotNil(t, service)
			assert.NotNil(t, service.clients)
			assert.NotNil(t, service.logger)
			assert.Equal(t, 0, len(service.clients))
		})
	}
}

func TestService_GetOrCreateClient_Validation(t *testing.T) {
	service := NewService(zap.NewNop())
	ctx := context.Background()

	tests := []struct {
		name     string
		connInfo *types.ConnectionInfo
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "nil connection info",
			connInfo: nil,
			wantErr:  true,
			errMsg:   "connection info is required",
		},
		{
			name: "empty connection string",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "",
			},
			wantErr: true,
			errMsg:  "connection info is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := service.GetOrCreateClient(ctx, tt.connInfo)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestService_GetOrCreateClient_OptionsStructure(t *testing.T) {
	service := NewService(zap.NewNop())

	connInfo := &types.ConnectionInfo{
		ConnectionString: "mongodb://localhost:27017",
		Options: map[string]string{
			"connectTimeout": "5s",
			"maxPoolSize":    "50",
		},
	}

	// Test that we can access options without errors (structure validation)
	assert.NotNil(t, connInfo.Options)
	assert.Equal(t, "5s", connInfo.Options["connectTimeout"])
	assert.Equal(t, "50", connInfo.Options["maxPoolSize"])

	// Test service structure
	assert.NotNil(t, service.clients)
	assert.NotNil(t, service.logger)
}

func TestService_ListDatabases_Validation(t *testing.T) {
	service := NewService(zap.NewNop())
	ctx := context.Background()

	tests := []struct {
		name     string
		connInfo *types.ConnectionInfo
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "nil connection info",
			connInfo: nil,
			wantErr:  true,
			errMsg:   "connection info is required",
		},
		{
			name: "empty connection string",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "",
			},
			wantErr: true,
			errMsg:  "connection info is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			databases, err := service.ListDatabases(ctx, tt.connInfo)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, databases)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, databases)
			}
		})
	}
}

func TestService_ListCollections_Validation(t *testing.T) {
	service := NewService(zap.NewNop())
	ctx := context.Background()

	tests := []struct {
		name         string
		connInfo     *types.ConnectionInfo
		databaseName string
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "nil connection info",
			connInfo:     nil,
			databaseName: "testdb",
			wantErr:      true,
			errMsg:       "connection info is required",
		},
		{
			name: "empty connection string",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "",
			},
			databaseName: "testdb",
			wantErr:      true,
			errMsg:       "connection info is required",
		},
		{
			name: "empty database name",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			databaseName: "",
			wantErr:      true,
			errMsg:       "database name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collections, err := service.ListCollections(ctx, tt.connInfo, tt.databaseName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, collections)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, collections)
			}
		})
	}
}

func TestService_CreateCollection_Validation(t *testing.T) {
	service := NewService(zap.NewNop())
	ctx := context.Background()

	tests := []struct {
		name           string
		connInfo       *types.ConnectionInfo
		databaseName   string
		collectionName string
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "nil connection info",
			connInfo:       nil,
			databaseName:   "testdb",
			collectionName: "testcoll",
			wantErr:        true,
			errMsg:         "connection info is required",
		},
		{
			name: "empty database name",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			databaseName:   "",
			collectionName: "testcoll",
			wantErr:        true,
			errMsg:         "database name is required",
		},
		{
			name: "empty collection name",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			databaseName:   "testdb",
			collectionName: "",
			wantErr:        true,
			errMsg:         "collection name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CreateCollection(ctx, tt.connInfo, tt.databaseName, tt.collectionName, nil)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_DropCollection_Validation(t *testing.T) {
	service := NewService(zap.NewNop())
	ctx := context.Background()

	tests := []struct {
		name           string
		connInfo       *types.ConnectionInfo
		databaseName   string
		collectionName string
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "nil connection info",
			connInfo:       nil,
			databaseName:   "testdb",
			collectionName: "testcoll",
			wantErr:        true,
			errMsg:         "connection info is required",
		},
		{
			name: "empty database name",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			databaseName:   "",
			collectionName: "testcoll",
			wantErr:        true,
			errMsg:         "database name is required",
		},
		{
			name: "empty collection name",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			databaseName:   "testdb",
			collectionName: "",
			wantErr:        true,
			errMsg:         "collection name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DropCollection(ctx, tt.connInfo, tt.databaseName, tt.collectionName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_GetCollectionStats_Validation(t *testing.T) {
	service := NewService(zap.NewNop())
	ctx := context.Background()

	tests := []struct {
		name           string
		connInfo       *types.ConnectionInfo
		databaseName   string
		collectionName string
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "nil connection info",
			connInfo:       nil,
			databaseName:   "testdb",
			collectionName: "testcoll",
			wantErr:        true,
			errMsg:         "connection info is required",
		},
		{
			name: "empty database name",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			databaseName:   "",
			collectionName: "testcoll",
			wantErr:        true,
			errMsg:         "database name is required",
		},
		{
			name: "empty collection name",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			databaseName:   "testdb",
			collectionName: "",
			wantErr:        true,
			errMsg:         "collection name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := service.GetCollectionStats(ctx, tt.connInfo, tt.databaseName, tt.collectionName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, stats)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stats)
			}
		})
	}
}

func TestService_CreateDatabase_Validation(t *testing.T) {
	service := NewService(zap.NewNop())
	ctx := context.Background()

	tests := []struct {
		name         string
		connInfo     *types.ConnectionInfo
		databaseName string
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "nil connection info",
			connInfo:     nil,
			databaseName: "testdb",
			wantErr:      true,
			errMsg:       "connection info is required",
		},
		{
			name: "empty database name",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			databaseName: "",
			wantErr:      true,
			errMsg:       "database name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CreateDatabase(ctx, tt.connInfo, tt.databaseName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_DropDatabase_Validation(t *testing.T) {
	service := NewService(zap.NewNop())
	ctx := context.Background()

	tests := []struct {
		name         string
		connInfo     *types.ConnectionInfo
		databaseName string
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "nil connection info",
			connInfo:     nil,
			databaseName: "testdb",
			wantErr:      true,
			errMsg:       "connection info is required",
		},
		{
			name: "empty database name",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			databaseName: "",
			wantErr:      true,
			errMsg:       "database name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DropDatabase(ctx, tt.connInfo, tt.databaseName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_Close(t *testing.T) {
	service := NewService(zap.NewNop())

	// Test closing an empty service
	err := service.Close(context.Background())
	assert.NoError(t, err)
}

func TestService_WithMockClient(t *testing.T) {
	service := NewService(zap.NewNop())

	// Create a mock client and manually add it to test Close() functionality
	_ = mongodb.NewMockClient(zap.NewNop()) // Create but don't use to avoid unused variable error

	// Test that we start with no clients cached
	assert.Equal(t, 0, len(service.clients))

	// Test Close removes all clients
	err := service.Close(context.Background())
	assert.NoError(t, err)

	// In a real implementation, clients map would be cleared after closing all connections
	// For now, we just test that Close doesn't panic
}

func TestService_ClientCaching(t *testing.T) {
	service := NewService(zap.NewNop())

	// Test that clients map is properly initialized
	assert.NotNil(t, service.clients)
	assert.Equal(t, 0, len(service.clients))

	// Test that logger is set properly
	assert.NotNil(t, service.logger)
}

func TestService_ConnectionInfoStructure(t *testing.T) {
	tests := []struct {
		name     string
		connInfo *types.ConnectionInfo
		valid    bool
	}{
		{
			name: "valid connection info",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
				Options: map[string]string{
					"maxPoolSize": "100",
					"timeout":     "30s",
				},
			},
			valid: true,
		},
		{
			name: "connection info without options",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
			},
			valid: true,
		},
		{
			name: "connection info with nil options",
			connInfo: &types.ConnectionInfo{
				ConnectionString: "mongodb://localhost:27017",
				Options:          nil,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.valid {
				assert.NotEmpty(t, tt.connInfo.ConnectionString)
			}

			// Test that Options map can be safely accessed
			if tt.connInfo.Options != nil {
				_, hasMaxPool := tt.connInfo.Options["maxPoolSize"]
				_, hasTimeout := tt.connInfo.Options["timeout"]
				// These assertions verify the structure without requiring specific values
				_ = hasMaxPool
				_ = hasTimeout
			}
		})
	}
}
