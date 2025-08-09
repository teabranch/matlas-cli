package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnsupportedFeatureError_WithDetails(t *testing.T) {
	err := UnsupportedFeatureError("VPC endpoints", "Requires PrivateEndpointServicesApi in SDK")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VPC endpoints is not yet supported in this build.")
	assert.Contains(t, err.Error(), "Requires PrivateEndpointServicesApi in SDK")
}

func TestUnsupportedFeatureError_WithoutDetails(t *testing.T) {
	err := UnsupportedFeatureError("network peering create")
	assert.Error(t, err)
	assert.Equal(t, "network peering create is not yet supported in this build.", err.Error())
}
