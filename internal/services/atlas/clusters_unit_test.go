package atlas

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/teabranch/matlas-cli/internal/clients/atlas"
)

func TestNewClustersService(t *testing.T) {
	client := &atlas.Client{} // Mock client

	service := NewClustersService(client)

	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
}

func TestClustersService_Structure(t *testing.T) {
	client := &atlas.Client{} // Mock client
	service := NewClustersService(client)

	// Test that service structure is correct
	assert.NotNil(t, service)
	assert.NotNil(t, service.client)
}
