package atlas

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/teabranch/matlas-cli/internal/clients/atlas"
)

func TestNewOrganizationsService(t *testing.T) {
	client := &atlas.Client{} // Mock client

	service := NewOrganizationsService(client)

	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
}

func TestOrganizationsService_Structure(t *testing.T) {
	client := &atlas.Client{}

	service := NewOrganizationsService(client)

	// Test that service structure is correct
	assert.NotNil(t, service)
	assert.NotNil(t, service.client)
}
