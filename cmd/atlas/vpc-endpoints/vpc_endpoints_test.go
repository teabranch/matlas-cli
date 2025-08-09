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
	assert.True(t, cmd.Hidden)
	assert.Contains(t, cmd.Short, "unsupported")
}

func TestVPCEndpoints_ListUnsupported(t *testing.T) {
	cmd := newListCmd()

	// No flags required for list; should return unsupported error immediately
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet supported")
}

func TestVPCEndpoints_GetUnsupported(t *testing.T) {
	cmd := newGetCmd()
	cmd.SetArgs([]string{"--project-id", "507f1f77bcf86cd799439011", "--endpoint-id", "5e2211c17a3e5a48f5497de3"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet supported")
}

func TestVPCEndpoints_CreateUnsupported(t *testing.T) {
	cmd := newCreateCmd()
	cmd.SetArgs([]string{"--project-id", "507f1f77bcf86cd799439011", "--cloud-provider", "AWS", "--region", "US_EAST_1"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet supported")
}

func TestVPCEndpoints_DeleteUnsupported(t *testing.T) {
	cmd := newDeleteCmd()
	cmd.SetArgs([]string{"--project-id", "507f1f77bcf86cd799439011", "--endpoint-id", "5e2211c17a3e5a48f5497de3", "--force"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet supported")
}
