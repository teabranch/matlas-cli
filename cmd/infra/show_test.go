package infra

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

func TestValidateShowOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        *ShowOptions
		expectError bool
	}{
		{name: "missing project id", opts: &ShowOptions{ProjectID: "", OutputFormat: "table", Timeout: time.Minute}, expectError: true},
		{name: "invalid output", opts: &ShowOptions{ProjectID: "507f1f77bcf86cd799439011", OutputFormat: "csv", Timeout: time.Minute}, expectError: true},
		{name: "invalid resource type", opts: &ShowOptions{ProjectID: "507f1f77bcf86cd799439011", OutputFormat: "table", ResourceType: "invalid", Timeout: time.Minute}, expectError: true},
		{name: "negative timeout", opts: &ShowOptions{ProjectID: "507f1f77bcf86cd799439011", OutputFormat: "json", Timeout: -1 * time.Second}, expectError: true},
		{name: "valid options", opts: &ShowOptions{ProjectID: "507f1f77bcf86cd799439011", OutputFormat: "yaml", Timeout: time.Minute}, expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateShowOptions(tt.opts)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMaskSecrets(t *testing.T) {
	state := &apply.ProjectState{
		Clusters: []types.ClusterManifest{{
			Metadata: types.ResourceMetadata{Name: "c1"},
			Spec:     types.ClusterSpec{Provider: "AWS", InstanceSize: "M10", Region: "US_EAST_1"},
		}},
		DatabaseUsers: []types.DatabaseUserManifest{{
			Metadata: types.ResourceMetadata{Name: "u1"},
			Spec:     types.DatabaseUserSpec{Username: "u1", Password: "secret", AuthDatabase: "admin"},
		}},
		NetworkAccess: []types.NetworkAccessManifest{{
			Metadata: types.ResourceMetadata{Name: "n1"},
			Spec:     types.NetworkAccessSpec{IPAddress: "192.168.1.1", Comment: "office"},
		}},
	}

	masked := maskSecrets(state)
	// Clusters copied unchanged
	assert.Equal(t, state.Clusters, masked.Clusters)
	// Users password masked
	if assert.Len(t, masked.DatabaseUsers, 1) {
		assert.Equal(t, "***hidden***", masked.DatabaseUsers[0].Spec.Password)
		assert.Equal(t, "u1", masked.DatabaseUsers[0].Spec.Username)
	}
	// Network access copied unchanged
	assert.Equal(t, state.NetworkAccess, masked.NetworkAccess)
}
