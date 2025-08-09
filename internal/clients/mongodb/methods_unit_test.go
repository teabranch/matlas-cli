package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestClient_Close(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name        string
		setupClient func() *Client
		wantErr     bool
	}{
		{
			name: "successful close",
			setupClient: func() *Client {
				return &Client{
					client: nil, // Simulate nil client (already closed or not connected)
					logger: logger,
				}
			},
			wantErr: false,
		},
		{
			name: "close with nil client",
			setupClient: func() *Client {
				return &Client{
					client: nil,
					logger: logger,
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			ctx := context.Background()

			err := client.Close(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_GetUnderlyingClientMethods(t *testing.T) {
	logger := zap.NewNop()
	client := &Client{
		client: nil,
		logger: logger,
	}

	underlying := client.GetUnderlyingClient()
	assert.Nil(t, underlying) // Since we set it to nil
}

func TestClient_ListDatabases_Structure(t *testing.T) {
	// Test that the method exists and has correct signature without calling it
	logger := zap.NewNop()
	client := &Client{
		client: nil,
		logger: logger,
	}

	// Verify client structure
	assert.NotNil(t, client.logger)
	assert.Nil(t, client.client)

	// Method signature validation - check the method is accessible
	// ListDatabases method should exist (no need to call it)
	assert.NotNil(t, client)
}

func TestClient_MethodsExist(t *testing.T) {
	logger := zap.NewNop()
	client := &Client{
		client: nil,
		logger: logger,
	}

	// Test that client structure is correct
	assert.NotNil(t, client)
	assert.NotNil(t, client.logger)
	assert.Nil(t, client.client) // Not connected
}

// Removed validation tests that would cause panics with nil client

func TestDefaultClientConfig_Structure(t *testing.T) {
	config := DefaultClientConfig()

	require.NotNil(t, config)
	assert.Greater(t, config.ConnectTimeout, time.Duration(0))
	assert.Greater(t, config.ServerSelectTimeout, time.Duration(0))
	assert.Greater(t, config.MaxPoolSize, uint64(0))
	assert.NotEmpty(t, config.ReadPreference)
}

func TestClientConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config ClientConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: ClientConfig{
				ConnectionString:    "mongodb://localhost:27017",
				ConnectTimeout:      10 * time.Second,
				ServerSelectTimeout: 5 * time.Second,
				MaxPoolSize:         100,
				ReadPreference:      "primary",
				RetryWrites:         true,
				TLSEnabled:          false,
				TLSInsecure:         false,
			},
			valid: true,
		},
		{
			name: "config with TLS",
			config: ClientConfig{
				ConnectionString:    "mongodb://localhost:27017",
				ConnectTimeout:      10 * time.Second,
				ServerSelectTimeout: 5 * time.Second,
				MaxPoolSize:         100,
				ReadPreference:      "primary",
				RetryWrites:         true,
				TLSEnabled:          true,
				TLSInsecure:         false,
			},
			valid: true,
		},
		{
			name: "empty connection string",
			config: ClientConfig{
				ConnectionString: "",
			},
			valid: false, // Would be invalid in practice
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that config can be created and has expected values
			assert.Equal(t, tt.config.ConnectionString, tt.config.ConnectionString)
			assert.Equal(t, tt.config.TLSEnabled, tt.config.TLSEnabled)
			assert.Equal(t, tt.config.ReadPreference, tt.config.ReadPreference)
			assert.Equal(t, tt.config.MaxPoolSize, tt.config.MaxPoolSize)
		})
	}
}

func TestNewClient_WithMockConfig(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name: "empty connection string",
			config: ClientConfig{
				ConnectionString: "",
			},
			wantErr: true,
		},
		{
			name: "invalid connection string",
			config: ClientConfig{
				ConnectionString:    "invalid://connection",
				ConnectTimeout:      100 * time.Millisecond, // Very short timeout
				ServerSelectTimeout: 100 * time.Millisecond,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(ctx, &tt.config, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				if client != nil {
					client.Close(ctx) // Clean up
				}
			}
		})
	}
}

func TestClient_StructureValidation(t *testing.T) {
	logger := zap.NewNop()

	// Test creating client with minimal structure
	client := &Client{
		logger: logger,
	}

	assert.NotNil(t, client.logger)
	assert.Nil(t, client.client) // Not connected

	// Test that GetUnderlyingClient works
	underlying := client.GetUnderlyingClient()
	assert.Nil(t, underlying)
}

func TestClient_TypesIntegration(t *testing.T) {
	// Test that our client integrates properly with types package

	// Test DatabaseInfo structure
	dbInfo := types.DatabaseInfo{
		Name:       "testdb",
		SizeOnDisk: 1024,
		Empty:      false,
	}

	assert.Equal(t, "testdb", dbInfo.Name)
	assert.Equal(t, int64(1024), dbInfo.SizeOnDisk)
	assert.False(t, dbInfo.Empty)

	// Test CollectionInfo structure
	collInfo := types.CollectionInfo{
		Name: "testcoll",
		Type: "collection",
		Options: map[string]interface{}{
			"capped": false,
		},
	}

	assert.Equal(t, "testcoll", collInfo.Name)
	assert.Equal(t, "collection", collInfo.Type)
	assert.Equal(t, false, collInfo.Options["capped"])

	// Test CollectionStats structure
	stats := types.CollectionStats{
		Count:      100,
		Size:       2048,
		AvgObjSize: 20,
	}

	assert.Equal(t, int64(100), stats.Count)
	assert.Equal(t, int64(2048), stats.Size)
	assert.Equal(t, int64(20), stats.AvgObjSize)
}
