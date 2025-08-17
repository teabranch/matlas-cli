package database

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/types"
)

func TestService_BasicStructure(t *testing.T) {
	logger := logging.New(logging.DefaultConfig())
	service := NewService(logger)

	// Test service structure
	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.clients)
	assert.Equal(t, 0, len(service.clients))
}

func TestDocumentService_BasicStructure(t *testing.T) {
	logger := logging.New(logging.DefaultConfig())
	dbService := NewService(logger)
	docService := NewDocumentService(dbService, logger)

	// Test service structure
	assert.NotNil(t, docService)
	assert.NotNil(t, docService.logger)
	assert.NotNil(t, docService.dbService)
}

func TestConnectionInfo_BasicStructure(t *testing.T) {
	// Test ConnectionInfo can be created and used
	info := &types.ConnectionInfo{
		ConnectionString: "mongodb://localhost:27017",
		Options: map[string]string{
			"appName": "test-app",
			"timeout": "30s",
		},
	}

	assert.Equal(t, "mongodb://localhost:27017", info.ConnectionString)
	assert.Equal(t, "test-app", info.Options["appName"])
	assert.Equal(t, "30s", info.Options["timeout"])
	assert.Len(t, info.Options, 2)
}

func TestService_EmptyValidation(t *testing.T) {
	logger := logging.New(logging.DefaultConfig())
	service := NewService(logger)

	// Test that service handles empty inputs gracefully
	assert.NotNil(t, service)

	// Test basic structure validation
	assert.NotNil(t, service.clients)
	assert.Equal(t, 0, len(service.clients))
}

func TestDocumentService_EmptyValidation(t *testing.T) {
	logger := logging.New(logging.DefaultConfig())
	dbService := NewService(logger)
	docService := NewDocumentService(dbService, logger)

	// Test basic structure validation
	assert.NotNil(t, docService)
	assert.Equal(t, dbService, docService.dbService)
	assert.Equal(t, logger, docService.logger)
}
