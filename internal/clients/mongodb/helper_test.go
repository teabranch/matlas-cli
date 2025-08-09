package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestClientConfig_AllFields(t *testing.T) {
	config := &ClientConfig{
		ConnectionString:    "mongodb://test:27017",
		ConnectTimeout:      15 * time.Second,
		ServerSelectTimeout: 10 * time.Second,
		MaxPoolSize:         50,
		RetryWrites:         false,
		ReadPreference:      "secondary",
		TLSEnabled:          true,
		TLSInsecure:         false,
	}

	// Test all fields are accessible and have expected values
	assert.Equal(t, "mongodb://test:27017", config.ConnectionString)
	assert.Equal(t, 15*time.Second, config.ConnectTimeout)
	assert.Equal(t, 10*time.Second, config.ServerSelectTimeout)
	assert.Equal(t, uint64(50), config.MaxPoolSize)
	assert.False(t, config.RetryWrites)
	assert.Equal(t, "secondary", config.ReadPreference)
	assert.True(t, config.TLSEnabled)
	assert.False(t, config.TLSInsecure)
}

func TestNewClient_FailFastValidation(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name: "very short timeout - should fail fast",
			config: ClientConfig{
				ConnectionString:    "mongodb://nonexistent:27017",
				ConnectTimeout:      1 * time.Millisecond,
				ServerSelectTimeout: 1 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "invalid protocol in connection string",
			config: ClientConfig{
				ConnectionString:    "http://localhost:27017", // Wrong protocol
				ConnectTimeout:      100 * time.Millisecond,
				ServerSelectTimeout: 100 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "malformed connection string",
			config: ClientConfig{
				ConnectionString:    "not-a-valid-connection-string",
				ConnectTimeout:      100 * time.Millisecond,
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
					client.Close(ctx)
				}
			}
		})
	}
}

func TestDefaultClientConfig_FieldValues(t *testing.T) {
	config := DefaultClientConfig()

	// Test specific default values
	assert.Equal(t, 30*time.Second, config.ConnectTimeout)
	assert.Equal(t, 30*time.Second, config.ServerSelectTimeout)
	assert.Equal(t, uint64(100), config.MaxPoolSize)
	assert.True(t, config.RetryWrites)
	assert.Equal(t, "primary", config.ReadPreference)
	assert.True(t, config.TLSEnabled)
}

func TestClient_ConfigAssignment(t *testing.T) {
	logger := zap.NewNop()

	// Test that we can create a client struct (even if connection fails)
	// This tests the basic constructor logic
	client := &Client{
		client: nil,
		logger: logger,
	}

	assert.NotNil(t, client.logger)
	assert.Nil(t, client.client)

	// Test GetUnderlyingClient with nil client
	underlying := client.GetUnderlyingClient()
	assert.Nil(t, underlying)
}

func TestClient_MethodSignatures(t *testing.T) {
	// Test that we can call Close safely with nil client
	logger := zap.NewNop()
	client := &Client{
		client: nil,
		logger: logger,
	}

	ctx := context.Background()

	// Close should succeed even with nil client (safe method)
	err := client.Close(ctx)
	assert.NoError(t, err)

	// Test GetUnderlyingClient (safe method)
	underlying := client.GetUnderlyingClient()
	assert.Nil(t, underlying)
}

func TestConfigFieldAssignments(t *testing.T) {
	// Test that config fields can be set and read
	config := &ClientConfig{}

	config.ConnectionString = "mongodb://user:pass@host:27017/db"
	config.ConnectTimeout = 5 * time.Second
	config.ServerSelectTimeout = 3 * time.Second
	config.MaxPoolSize = 75
	config.RetryWrites = true
	config.ReadPreference = "primaryPreferred"
	config.TLSEnabled = false
	config.TLSInsecure = true

	// Verify assignments
	assert.Equal(t, "mongodb://user:pass@host:27017/db", config.ConnectionString)
	assert.Equal(t, 5*time.Second, config.ConnectTimeout)
	assert.Equal(t, 3*time.Second, config.ServerSelectTimeout)
	assert.Equal(t, uint64(75), config.MaxPoolSize)
	assert.True(t, config.RetryWrites)
	assert.Equal(t, "primaryPreferred", config.ReadPreference)
	assert.False(t, config.TLSEnabled)
	assert.True(t, config.TLSInsecure)
}
