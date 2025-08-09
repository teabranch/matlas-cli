package networkpeering

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetworkPeeringCmd_Metadata(t *testing.T) {
	cmd := NewNetworkPeeringCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "network-peering", cmd.Use)
	// Not hidden; only create is gated
	assert.False(t, cmd.Hidden)
}

func TestNetworkPeering_CreateValidatesFlags(t *testing.T) {
	cmd := newCreateCmd()
	cmd.SetArgs([]string{
		"--project-id", "",
		"--cloud-provider", "AWS",
		"--vpc-id", "vpc-123456",
		"--region", "US_EAST_1",
		"--cidr-block", "10.0.0.0/16",
	})
	err := cmd.Execute()
	require.Error(t, err)
}
