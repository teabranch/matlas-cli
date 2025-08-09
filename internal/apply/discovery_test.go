package apply

import (
	"context"
	"testing"
	"time"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/types"
)

func TestAtlasStateDiscovery_NewAtlasStateDiscovery(t *testing.T) {
	client := &atlasclient.Client{} // Mock client for testing
	discovery := NewAtlasStateDiscovery(client)

	if discovery == nil {
		t.Fatal("NewAtlasStateDiscovery returned nil")
	}

	if discovery.client != client {
		t.Error("Client not set correctly")
	}

	if discovery.rateLimiter == nil {
		t.Error("Rate limiter not initialized")
	}

	if discovery.maxConcurrentOps != 5 {
		t.Errorf("Expected maxConcurrentOps to be 5, got %d", discovery.maxConcurrentOps)
	}
}

func TestInMemoryStateCache_Basic(t *testing.T) {
	cache := NewInMemoryStateCache(10, time.Hour)
	defer cache.Stop()

	projectID := "test-project-123"
	state := &ProjectState{
		DiscoveredAt: time.Now(),
		Fingerprint:  "test-fingerprint",
	}

	// Test initial state
	_, found := cache.Get(projectID)
	if found {
		t.Error("Expected cache miss, but found entry")
	}

	// Test set and get
	cache.Set(projectID, state, time.Hour)

	retrieved, found := cache.Get(projectID)
	if !found {
		t.Error("Expected cache hit, but got miss")
	}

	if retrieved.Fingerprint != state.Fingerprint {
		t.Errorf("Expected fingerprint %s, got %s", state.Fingerprint, retrieved.Fingerprint)
	}

	// Test delete
	cache.Delete(projectID)

	_, found = cache.Get(projectID)
	if found {
		t.Error("Expected cache miss after delete, but found entry")
	}
}

func TestInMemoryStateCache_Expiration(t *testing.T) {
	cache := NewInMemoryStateCache(10, time.Millisecond*100)
	defer cache.Stop()

	projectID := "test-project-expire"
	state := &ProjectState{
		DiscoveredAt: time.Now(),
		Fingerprint:  "test-fingerprint-expire",
	}

	// Set with short TTL
	cache.Set(projectID, state, time.Millisecond*50)

	// Should be available immediately
	_, found := cache.Get(projectID)
	if !found {
		t.Error("Expected cache hit immediately after set")
	}

	// Wait for expiration
	time.Sleep(time.Millisecond * 100)

	// Should be expired now
	_, found = cache.Get(projectID)
	if found {
		t.Error("Expected cache miss after expiration")
	}
}

func TestInMemoryStateCache_Stats_Discovery(t *testing.T) {
	cache := NewInMemoryStateCache(10, time.Hour)
	defer cache.Stop()

	projectID := "test-project-stats"
	state := &ProjectState{
		DiscoveredAt: time.Now(),
		Fingerprint:  "test-fingerprint-stats",
	}

	// Initial stats
	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected initial size 0, got %d", stats.Size)
	}

	// After cache miss
	_, _ = cache.Get(projectID)
	stats = cache.Stats()
	if stats.MissCount != 1 {
		t.Errorf("Expected miss count 1, got %d", stats.MissCount)
	}

	// After set and hit
	cache.Set(projectID, state, time.Hour)
	_, _ = cache.Get(projectID)

	stats = cache.Stats()
	if stats.HitCount != 1 {
		t.Errorf("Expected hit count 1, got %d", stats.HitCount)
	}
	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}
}

func TestInMemoryStateCache_InvalidateByResourceType(t *testing.T) {
	cache := NewInMemoryStateCache(10, time.Hour)
	defer cache.Stop()

	projectID := "test-project-invalidate"
	state := &ProjectState{
		Clusters: []types.ClusterManifest{
			{
				Metadata: types.ResourceMetadata{Name: "test-cluster"},
			},
		},
		DiscoveredAt: time.Now(),
		Fingerprint:  "test-fingerprint-invalidate",
	}

	cache.Set(projectID, state, time.Hour)

	// Verify it's cached
	_, found := cache.Get(projectID)
	if !found {
		t.Error("Expected cache hit before invalidation")
	}

	// Invalidate by cluster type
	cache.InvalidateByResourceType(types.KindCluster)

	// Should be invalidated now
	_, found = cache.Get(projectID)
	if found {
		t.Error("Expected cache miss after invalidation")
	}
}

func TestCachedStateDiscovery_Integration(t *testing.T) {
	// Create a mock discovery service that returns a test state
	mockDiscovery := &MockStateDiscovery{
		projectState: &ProjectState{
			DiscoveredAt: time.Now(),
			Fingerprint:  "mock-fingerprint",
		},
	}

	cache := NewInMemoryStateCache(10, time.Hour)
	defer cache.Stop()

	cachedDiscovery := NewCachedStateDiscovery(mockDiscovery, cache)

	ctx := context.Background()
	projectID := "test-project-cached"

	// First call should hit the mock discovery
	state1, err := cachedDiscovery.DiscoverProject(ctx, projectID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mockDiscovery.callCount != 1 {
		t.Errorf("Expected 1 call to mock discovery, got %d", mockDiscovery.callCount)
	}

	// Second call should hit the cache
	state2, err := cachedDiscovery.DiscoverProject(ctx, projectID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if mockDiscovery.callCount != 1 {
		t.Errorf("Expected still 1 call to mock discovery, got %d", mockDiscovery.callCount)
	}

	if state1.Fingerprint != state2.Fingerprint {
		t.Error("Expected same state from cache")
	}
}

// MockStateDiscovery is a simple mock for testing
type MockStateDiscovery struct {
	projectState *ProjectState
	callCount    int
}

func (m *MockStateDiscovery) DiscoverProject(ctx context.Context, projectID string) (*ProjectState, error) {
	m.callCount++
	return m.projectState, nil
}

func (m *MockStateDiscovery) DiscoverClusters(ctx context.Context, projectID string) ([]types.ClusterManifest, error) {
	return nil, nil
}

func (m *MockStateDiscovery) DiscoverDatabaseUsers(ctx context.Context, projectID string) ([]types.DatabaseUserManifest, error) {
	return nil, nil
}

func (m *MockStateDiscovery) DiscoverNetworkAccess(ctx context.Context, projectID string) ([]types.NetworkAccessManifest, error) {
	return nil, nil
}

func (m *MockStateDiscovery) DiscoverProjectSettings(ctx context.Context, projectID string) (*types.ProjectManifest, error) {
	return nil, nil
}
