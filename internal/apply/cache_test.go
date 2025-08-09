package apply

import (
	"fmt"
	"testing"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

func TestNewInMemoryStateCache(t *testing.T) {
	cache := NewInMemoryStateCache(10, 5*time.Minute)

	if cache == nil {
		t.Fatal("NewInMemoryStateCache returned nil")
	}

	stats := cache.Stats()
	if stats.MaxEntries != 10 {
		t.Errorf("Expected maxEntries 10, got %d", stats.MaxEntries)
	}

	if stats.DefaultTTL != 5*time.Minute {
		t.Errorf("Expected TTL 5 minutes, got %v", stats.DefaultTTL)
	}

	if stats.HitCount != 0 {
		t.Error("hitCount should start at 0")
	}

	if stats.MissCount != 0 {
		t.Error("missCount should start at 0")
	}
}

func TestInMemoryStateCache_Set_Get(t *testing.T) {
	cache := NewInMemoryStateCache(5, 1*time.Hour)
	projectID := "test-project"

	// Test setting and getting state
	state := &ProjectState{
		Project: &types.ProjectManifest{
			Metadata: types.ResourceMetadata{Name: projectID},
		},
		Clusters:      []types.ClusterManifest{},
		DatabaseUsers: []types.DatabaseUserManifest{},
		NetworkAccess: []types.NetworkAccessManifest{},
		Fingerprint:   "test-fingerprint",
		DiscoveredAt:  time.Now(),
	}

	cache.Set(projectID, state, 0)

	// Test successful get
	retrieved, found := cache.Get(projectID)
	if !found {
		t.Fatal("Failed to retrieve cached state")
	}

	if retrieved.Project.Metadata.Name != projectID {
		t.Errorf("Expected project name %s, got %s", projectID, retrieved.Project.Metadata.Name)
	}

	if retrieved.Fingerprint != "test-fingerprint" {
		t.Errorf("Expected fingerprint test-fingerprint, got %s", retrieved.Fingerprint)
	}
}

func TestInMemoryStateCache_Get_Miss(t *testing.T) {
	cache := NewInMemoryStateCache(5, 1*time.Hour)

	// Test cache miss
	result, found := cache.Get("non-existent-project")
	if found {
		t.Error("Expected cache miss")
	}

	if result != nil {
		t.Error("Expected nil result for cache miss")
	}

	// Check miss count increased
	stats := cache.Stats()
	if stats.MissCount != 1 {
		t.Errorf("Expected miss count 1, got %d", stats.MissCount)
	}
}

func TestInMemoryStateCache_Get_Hit(t *testing.T) {
	cache := NewInMemoryStateCache(5, 1*time.Hour)
	projectID := "test-project"

	state := &ProjectState{
		Project: &types.ProjectManifest{
			Metadata: types.ResourceMetadata{Name: projectID},
		},
		Fingerprint:  "test",
		DiscoveredAt: time.Now(),
	}
	cache.Set(projectID, state, 0)

	// First get - should be a hit
	result1, found1 := cache.Get(projectID)
	if !found1 || result1 == nil {
		t.Fatal("Expected cache hit")
	}

	// Second get - should also be a hit
	result2, found2 := cache.Get(projectID)
	if !found2 || result2 == nil {
		t.Fatal("Expected cache hit")
	}

	// Check hit count
	stats := cache.Stats()
	if stats.HitCount != 2 {
		t.Errorf("Expected hit count 2, got %d", stats.HitCount)
	}

	if stats.MissCount != 0 {
		t.Errorf("Expected miss count 0, got %d", stats.MissCount)
	}
}

func TestInMemoryStateCache_TTL_Expiration(t *testing.T) {
	cache := NewInMemoryStateCache(5, 50*time.Millisecond)
	projectID := "test-project"

	state := &ProjectState{
		Project: &types.ProjectManifest{
			Metadata: types.ResourceMetadata{Name: projectID},
		},
		Fingerprint:  "test",
		DiscoveredAt: time.Now(),
	}
	cache.Set(projectID, state, 0)

	// Should be available immediately
	result1, found1 := cache.Get(projectID)
	if !found1 || result1 == nil {
		t.Error("Expected cache hit before expiration")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	result2, found2 := cache.Get(projectID)
	if found2 || result2 != nil {
		t.Error("Expected cache miss after expiration")
	}
}

func TestInMemoryStateCache_MaxSize_LRU_Eviction(t *testing.T) {
	cache := NewInMemoryStateCache(2, 1*time.Hour) // Max size 2

	// Add first item
	state1 := &ProjectState{
		Project:      &types.ProjectManifest{Metadata: types.ResourceMetadata{Name: "project-1"}},
		Fingerprint:  "test1",
		DiscoveredAt: time.Now(),
	}
	cache.Set("project-1", state1, 0)

	// Add second item
	state2 := &ProjectState{
		Project:      &types.ProjectManifest{Metadata: types.ResourceMetadata{Name: "project-2"}},
		Fingerprint:  "test2",
		DiscoveredAt: time.Now(),
	}
	cache.Set("project-2", state2, 0)

	// Both should be in cache
	if _, found := cache.Get("project-1"); !found {
		t.Error("project-1 should be in cache")
	}
	if _, found := cache.Get("project-2"); !found {
		t.Error("project-2 should be in cache")
	}

	// Add third item - should evict the least recently used (project-1)
	state3 := &ProjectState{
		Project:      &types.ProjectManifest{Metadata: types.ResourceMetadata{Name: "project-3"}},
		Fingerprint:  "test3",
		DiscoveredAt: time.Now(),
	}
	cache.Set("project-3", state3, 0)

	// project-1 should be evicted
	if _, found := cache.Get("project-1"); found {
		t.Error("project-1 should have been evicted")
	}

	// project-2 and project-3 should still be in cache
	if _, found := cache.Get("project-2"); !found {
		t.Error("project-2 should still be in cache")
	}
	if _, found := cache.Get("project-3"); !found {
		t.Error("project-3 should be in cache")
	}
}

func TestInMemoryStateCache_Delete(t *testing.T) {
	cache := NewInMemoryStateCache(5, 1*time.Hour)
	projectID := "test-project"

	// Add item to cache
	state := &ProjectState{
		Project:      &types.ProjectManifest{Metadata: types.ResourceMetadata{Name: projectID}},
		Fingerprint:  "test",
		DiscoveredAt: time.Now(),
	}
	cache.Set(projectID, state, 0)

	// Verify it's in cache
	if _, found := cache.Get(projectID); !found {
		t.Fatal("Item should be in cache before deletion")
	}

	// Delete the item
	cache.Delete(projectID)

	// Verify it's no longer in cache
	if _, found := cache.Get(projectID); found {
		t.Error("Item should not be in cache after deletion")
	}
}

func TestInMemoryStateCache_Clear(t *testing.T) {
	cache := NewInMemoryStateCache(5, 1*time.Hour)

	// Add multiple items
	for i := 0; i < 3; i++ {
		projectID := fmt.Sprintf("project-%d", i)
		state := &ProjectState{
			Project:      &types.ProjectManifest{Metadata: types.ResourceMetadata{Name: projectID}},
			Fingerprint:  fmt.Sprintf("test%d", i),
			DiscoveredAt: time.Now(),
		}
		cache.Set(projectID, state, 0)
	}

	// Verify items are in cache
	for i := 0; i < 3; i++ {
		projectID := fmt.Sprintf("project-%d", i)
		if _, found := cache.Get(projectID); !found {
			t.Errorf("project-%d should be in cache", i)
		}
	}

	// Clear cache
	cache.Clear()

	// Verify all items are gone
	for i := 0; i < 3; i++ {
		projectID := fmt.Sprintf("project-%d", i)
		if _, found := cache.Get(projectID); found {
			t.Errorf("project-%d should not be in cache after clear", i)
		}
	}
}

func TestInMemoryStateCache_Stats(t *testing.T) {
	cache := NewInMemoryStateCache(5, 1*time.Hour)
	projectID := "test-project"

	// Initial stats
	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected initial size 0, got %d", stats.Size)
	}
	if stats.HitCount != 0 {
		t.Errorf("Expected initial hit count 0, got %d", stats.HitCount)
	}
	if stats.MissCount != 0 {
		t.Errorf("Expected initial miss count 0, got %d", stats.MissCount)
	}

	// Add item and test
	state := &ProjectState{
		Project:      &types.ProjectManifest{Metadata: types.ResourceMetadata{Name: projectID}},
		Fingerprint:  "test",
		DiscoveredAt: time.Now(),
	}
	cache.Set(projectID, state, 0)

	// Test hit
	cache.Get(projectID)

	// Test miss
	cache.Get("non-existent")

	// Check updated stats
	stats = cache.Stats()
	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}
	if stats.HitCount != 1 {
		t.Errorf("Expected hit count 1, got %d", stats.HitCount)
	}
	if stats.MissCount != 1 {
		t.Errorf("Expected miss count 1, got %d", stats.MissCount)
	}
	if stats.HitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", stats.HitRate)
	}
}

func TestInMemoryStateCache_Stats_EdgeCases(t *testing.T) {
	cache := NewInMemoryStateCache(5, 1*time.Hour)

	// Test stats with no operations
	stats := cache.Stats()
	if stats.HitRate != 0.0 {
		t.Errorf("Expected hit rate 0.0 with no operations, got %f", stats.HitRate)
	}

	// Test stats with only misses
	cache.Get("non-existent-1")
	cache.Get("non-existent-2")

	stats = cache.Stats()
	if stats.HitRate != 0.0 {
		t.Errorf("Expected hit rate 0.0 with only misses, got %f", stats.HitRate)
	}
}

func TestInMemoryStateCache_Stop(t *testing.T) {
	cache := NewInMemoryStateCache(5, 100*time.Millisecond)

	// Add some items
	state := &ProjectState{
		Project:      &types.ProjectManifest{Metadata: types.ResourceMetadata{Name: "test"}},
		Fingerprint:  "test",
		DiscoveredAt: time.Now(),
	}
	cache.Set("test", state, 0)

	// Stop the cache
	cache.Stop()

	// Cache should still work for basic operations
	result, found := cache.Get("test")
	if !found || result == nil {
		t.Error("Cache should still work after stop")
	}

	// But cleanup goroutine should be stopped
	// (Hard to test directly, but we can verify Stop doesn't panic)
}

func TestInMemoryStateCache_ConcurrentAccess(t *testing.T) {
	cache := NewInMemoryStateCache(10, 1*time.Hour)

	// Test concurrent sets and gets
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			projectID := fmt.Sprintf("project-%d", i)
			state := &ProjectState{
				Project:      &types.ProjectManifest{Metadata: types.ResourceMetadata{Name: projectID}},
				Fingerprint:  fmt.Sprintf("test%d", i),
				DiscoveredAt: time.Now(),
			}
			cache.Set(projectID, state, 0)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			projectID := fmt.Sprintf("project-%d", i)
			cache.Get(projectID) // May hit or miss
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify cache is still functional
	state := &ProjectState{
		Project:      &types.ProjectManifest{Metadata: types.ResourceMetadata{Name: "final-test"}},
		Fingerprint:  "final",
		DiscoveredAt: time.Now(),
	}
	cache.Set("final-test", state, 0)

	result, found := cache.Get("final-test")
	if !found || result == nil {
		t.Error("Cache should be functional after concurrent access")
	}
}
