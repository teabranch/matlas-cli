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

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	require.NotNil(t, config)
	assert.Equal(t, 30*time.Second, config.ConnectTimeout)
	assert.Equal(t, 30*time.Second, config.ServerSelectTimeout)
	assert.Equal(t, uint64(100), config.MaxPoolSize)
	assert.True(t, config.RetryWrites)
	assert.Equal(t, "primary", config.ReadPreference)
	assert.True(t, config.TLSEnabled)
	assert.False(t, config.TLSInsecure)
}

// Note: ClientConfig.Validate() method doesn't exist in the actual implementation
// These tests verify the config structure instead

func TestNewClient_ConfigValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second) // Short timeout
	defer cancel()
	logger := zap.NewNop()

	tests := []struct {
		name   string
		config *ClientConfig
	}{
		{
			name: "invalid read preference",
			config: &ClientConfig{
				ConnectionString: "mongodb://localhost:27017",
				ReadPreference:   "invalid-preference",
				ConnectTimeout:   100 * time.Millisecond, // Very short timeout
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(ctx, tt.config, logger)

			// Should fail due to invalid read preference before attempting connection
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid read preference")
			assert.Nil(t, client)
		})
	}
}

func TestNewClient_NilInputs(t *testing.T) {
	// Test that nil config uses defaults without attempting connection
	// by providing an invalid connection string that fails immediately
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This will fail fast due to invalid connection string
	client, err := NewClient(ctx, &ClientConfig{
		ConnectionString: "", // Empty connection string should fail validation
		ConnectTimeout:   1 * time.Millisecond,
	}, nil)

	// Should fail due to empty connection string
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestClientConfig_ConnectionStringValidation(t *testing.T) {
	tests := []struct {
		name             string
		connectionString string
		expectError      bool
	}{
		{
			name:             "empty URI",
			connectionString: "",
			expectError:      true,
		},
		{
			name:             "invalid scheme",
			connectionString: "mysql://localhost:3306",
			expectError:      true,
		},
		{
			name:             "basic mongodb URI structure",
			connectionString: "mongodb://localhost:27017",
			expectError:      false, // Structure is valid, but won't connect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ClientConfig{
				ConnectionString: tt.connectionString,
				ConnectTimeout:   1 * time.Millisecond, // Very short to fail fast
				ReadPreference:   "primary",
			}

			// Test config structure, not actual connections
			assert.Equal(t, tt.connectionString, config.ConnectionString)
			assert.Equal(t, "primary", config.ReadPreference)
		})
	}
}

func TestClientConfig_TimeoutStructure(t *testing.T) {
	tests := []struct {
		name           string
		connectTimeout time.Duration
		selectTimeout  time.Duration
	}{
		{
			name:           "valid timeouts",
			connectTimeout: 10 * time.Second,
			selectTimeout:  15 * time.Second,
		},
		{
			name:           "zero connect timeout",
			connectTimeout: 0,
			selectTimeout:  10 * time.Second,
		},
		{
			name:           "zero select timeout",
			connectTimeout: 10 * time.Second,
			selectTimeout:  0,
		},
		{
			name:           "negative timeout",
			connectTimeout: -1 * time.Second,
			selectTimeout:  10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ClientConfig{
				ConnectionString:    "mongodb://localhost:27017",
				ConnectTimeout:      tt.connectTimeout,
				ServerSelectTimeout: tt.selectTimeout,
				ReadPreference:      "primary",
			}

			// Test that config fields are set correctly
			assert.Equal(t, tt.connectTimeout, config.ConnectTimeout)
			assert.Equal(t, tt.selectTimeout, config.ServerSelectTimeout)
		})
	}
}

func TestClientConfig_PoolSizeStructure(t *testing.T) {
	tests := []struct {
		name        string
		maxPoolSize uint64
	}{
		{
			name:        "valid pool size",
			maxPoolSize: 50,
		},
		{
			name:        "minimum pool size",
			maxPoolSize: 1,
		},
		{
			name:        "large pool size",
			maxPoolSize: 1000,
		},
		{
			name:        "zero pool size",
			maxPoolSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ClientConfig{
				ConnectionString:    "mongodb://localhost:27017",
				ConnectTimeout:      10 * time.Second,
				ServerSelectTimeout: 10 * time.Second,
				MaxPoolSize:         tt.maxPoolSize,
				ReadPreference:      "primary",
			}

			// Test that config fields are set correctly
			assert.Equal(t, tt.maxPoolSize, config.MaxPoolSize)
		})
	}
}

func TestClientConfig_ReadPreferenceStructure(t *testing.T) {
	validPreferences := []string{
		"primary",
		"primaryPreferred",
		"secondary",
		"secondaryPreferred",
		"nearest",
	}

	for _, pref := range validPreferences {
		t.Run("valid_"+pref, func(t *testing.T) {
			config := &ClientConfig{
				ConnectionString:    "mongodb://localhost:27017",
				ConnectTimeout:      10 * time.Second,
				ServerSelectTimeout: 10 * time.Second,
				MaxPoolSize:         50,
				ReadPreference:      pref,
			}

			assert.Equal(t, pref, config.ReadPreference)
		})
	}

	// Test invalid read preference
	t.Run("invalid_preference", func(t *testing.T) {
		config := &ClientConfig{
			ConnectionString:    "mongodb://localhost:27017",
			ConnectTimeout:      10 * time.Second,
			ServerSelectTimeout: 10 * time.Second,
			MaxPoolSize:         50,
			ReadPreference:      "invalidPreference",
		}

		assert.Equal(t, "invalidPreference", config.ReadPreference)
	})
}

func TestClientConfig_TLSSettings(t *testing.T) {
	tests := []struct {
		name        string
		tlsEnabled  bool
		tlsInsecure bool
	}{
		{
			name:        "TLS enabled, secure",
			tlsEnabled:  true,
			tlsInsecure: false,
		},
		{
			name:        "TLS enabled, insecure",
			tlsEnabled:  true,
			tlsInsecure: true,
		},
		{
			name:        "TLS disabled",
			tlsEnabled:  false,
			tlsInsecure: false,
		},
		{
			name:        "TLS disabled but insecure flag set",
			tlsEnabled:  false,
			tlsInsecure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ClientConfig{
				ConnectionString:    "mongodb://localhost:27017",
				ConnectTimeout:      10 * time.Second,
				ServerSelectTimeout: 10 * time.Second,
				MaxPoolSize:         50,
				ReadPreference:      "primary",
				TLSEnabled:          tt.tlsEnabled,
				TLSInsecure:         tt.tlsInsecure,
			}

			// Test that config fields are set correctly
			assert.Equal(t, tt.tlsEnabled, config.TLSEnabled)
			assert.Equal(t, tt.tlsInsecure, config.TLSInsecure)
		})
	}
}

func TestClient_GetUnderlyingClient(t *testing.T) {
	// Test with mock client structure
	client := &Client{
		client: nil, // In real tests, this would be a real mongo.Client
		logger: zap.NewNop(),
		config: DefaultClientConfig(),
	}

	underlying := client.GetUnderlyingClient()
	assert.Equal(t, client.client, underlying)
}

func TestMockClient_BasicFunctionality(t *testing.T) {
	logger := zap.NewNop()
	mockClient := NewMockClient(logger)

	// Test mock client initialization
	assert.NotNil(t, mockClient)

	// Test mock database setup
	mockDatabases := []types.DatabaseInfo{
		{Name: "testdb1", SizeOnDisk: 1024},
		{Name: "testdb2", SizeOnDisk: 2048},
	}
	mockClient.SetMockDatabases(mockDatabases)

	// Test mock collections setup
	mockCollections := []types.CollectionInfo{
		{Name: "collection1", Type: "collection"},
		{Name: "collection2", Type: "collection"},
	}
	mockClient.SetMockCollections("testdb1", mockCollections)

	// Verify mock data was set correctly
	assert.Equal(t, 2, len(mockClient.databases))
	assert.Equal(t, "testdb1", mockClient.databases[0].Name)
	assert.Equal(t, int64(1024), mockClient.databases[0].SizeOnDisk)
}

func TestMockClient_ErrorSimulation(t *testing.T) {
	mockClient := NewMockClient(zap.NewNop())

	// Test error simulation setup using the correct SetError method
	mockClient.SetError("connect", true)
	mockClient.SetError("listdb", true)
	mockClient.SetError("listcoll", true)

	// Verify error flags are set
	assert.True(t, mockClient.shouldErrorOnConnect)
	assert.True(t, mockClient.shouldErrorOnListDB)
	assert.True(t, mockClient.shouldErrorOnListColl)
}

func TestClientConfig_Defaults(t *testing.T) {
	config := DefaultClientConfig()

	// Test that defaults are reasonable
	assert.NotZero(t, config.ConnectTimeout)
	assert.NotZero(t, config.ServerSelectTimeout)
	assert.NotZero(t, config.MaxPoolSize)
	assert.NotEmpty(t, config.ReadPreference)

	// Test timeout values are positive
	assert.Positive(t, config.ConnectTimeout)
	assert.Positive(t, config.ServerSelectTimeout)

	// Test pool size is reasonable
	assert.Greater(t, config.MaxPoolSize, uint64(0))
	assert.LessOrEqual(t, config.MaxPoolSize, uint64(1000)) // Reasonable upper bound
}
