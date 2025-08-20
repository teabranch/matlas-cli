package vpcendpoints

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVPCEndpointsCmd_Metadata(t *testing.T) {
	cmd := NewVPCEndpointsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "vpc-endpoints", cmd.Use)
	assert.False(t, cmd.Hidden) // VPC endpoints feature is now enabled and visible
	// Short description should indicate management of VPC endpoints
	assert.Contains(t, cmd.Short, "VPC endpoints")
	// Command should have aliases
	assert.Contains(t, cmd.Aliases, "vpc-endpoint")
	assert.Contains(t, cmd.Aliases, "vpc")
}

// Further tests for create/list/get/delete behavior should be added to validate new implementation.
