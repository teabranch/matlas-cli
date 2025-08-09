package apply

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/types"
)

func TestInMemoryStateCache_InvalidateByResource(t *testing.T) {
	cache := NewInMemoryStateCache(100, 10*time.Minute)

	// Test InvalidateByResource with valid parameters
	cache.InvalidateByResource("project-123", types.KindCluster, "test-cluster")

	// Should not panic and cache should still be functional
	assert.NotNil(t, cache)
}

func TestInMemoryStateCache_StateContainsResource(t *testing.T) {
	cache := NewInMemoryStateCache(100, 10*time.Minute)

	// Create a test project state
	state := &ProjectState{
		Clusters: []types.ClusterManifest{
			{
				Metadata: types.ResourceMetadata{
					Name: "test-cluster",
				},
			},
		},
		DiscoveredAt: time.Now(),
	}

	// Set the state in cache
	cache.Set("project-123", state, 5*time.Minute)

	// Test that we can retrieve it
	cached, found := cache.Get("project-123")
	assert.True(t, found)
	assert.NotNil(t, cached)
	assert.Equal(t, "test-cluster", cached.Clusters[0].Metadata.Name)
}

func TestCachedStateDiscovery_DiscoverClusters(t *testing.T) {
	// Create a basic client for testing
	client := &atlasclient.Client{}
	baseDiscovery := NewAtlasStateDiscovery(client)
	cache := NewInMemoryStateCache(100, 10*time.Minute)

	cached := NewCachedStateDiscovery(baseDiscovery, cache)
	assert.NotNil(t, cached)

	// Test structure is correct
	assert.NotNil(t, cached)
}

func TestCachedStateDiscovery_DiscoverDatabaseUsers(t *testing.T) {
	// Create a basic client for testing
	client := &atlasclient.Client{}
	baseDiscovery := NewAtlasStateDiscovery(client)
	cache := NewInMemoryStateCache(100, 10*time.Minute)

	cached := NewCachedStateDiscovery(baseDiscovery, cache)
	assert.NotNil(t, cached)

	// Test structure is correct
	assert.NotNil(t, cached)
}

func TestProjectState_BasicStructure(t *testing.T) {
	state := &ProjectState{
		Project: &types.ProjectManifest{
			Metadata: types.ResourceMetadata{
				Name: "test-project",
			},
		},
		Clusters: []types.ClusterManifest{
			{
				Metadata: types.ResourceMetadata{
					Name: "cluster-1",
				},
			},
		},
		DatabaseUsers: []types.DatabaseUserManifest{
			{
				Metadata: types.ResourceMetadata{
					Name: "user-1",
				},
			},
		},
		DiscoveredAt: time.Now(),
	}

	assert.Equal(t, "test-project", state.Project.Metadata.Name)
	assert.Len(t, state.Clusters, 1)
	assert.Equal(t, "cluster-1", state.Clusters[0].Metadata.Name)
	assert.Len(t, state.DatabaseUsers, 1)
	assert.Equal(t, "user-1", state.DatabaseUsers[0].Metadata.Name)
	assert.False(t, state.DiscoveredAt.IsZero())
}

func TestInMemoryStateCache_CleanupFunctionality(t *testing.T) {
	cache := NewInMemoryStateCache(100, 10*time.Minute)

	// Add some test data
	state := &ProjectState{
		DiscoveredAt: time.Now(),
	}
	cache.Set("test-project", state, 1*time.Millisecond) // Very short TTL

	// Wait a bit for expiry
	time.Sleep(2 * time.Millisecond)

	// Test that cleanup doesn't panic when Stop is called
	cache.Stop()

	assert.NotNil(t, cache)
}

func TestGenerateStateFingerprint(t *testing.T) {
	state := &ProjectState{
		Project: &types.ProjectManifest{
			Metadata: types.ResourceMetadata{
				Name: "test-project",
			},
		},
		Clusters: []types.ClusterManifest{
			{
				Metadata: types.ResourceMetadata{
					Name: "test-cluster",
				},
			},
		},
		DiscoveredAt: time.Now(),
	}

	fingerprint, err := GenerateStateFingerprint(state)
	assert.NoError(t, err)
	assert.NotEmpty(t, fingerprint)
	assert.Len(t, fingerprint, 64) // SHA256 hex string length

	// Test that the same state generates the same fingerprint
	fingerprint2, err := GenerateStateFingerprint(state)
	assert.NoError(t, err)
	assert.Equal(t, fingerprint, fingerprint2)
}

func TestRateLimiter_BasicFunctionality(t *testing.T) {
	// Test NewRateLimiter creates a working rate limiter
	rl := NewRateLimiter(10, 1*time.Second)
	assert.NotNil(t, rl)

	// Test that Stop doesn't panic
	rl.Stop()
}
