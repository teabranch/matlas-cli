package apply

import (
	"context"
	"sync"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

// StateCache provides caching for discovered project states
type StateCache interface {
	// Get retrieves a cached project state
	Get(projectID string) (*ProjectState, bool)

	// Set stores a project state in the cache
	Set(projectID string, state *ProjectState, ttl time.Duration)

	// Delete removes a project state from the cache
	Delete(projectID string)

	// InvalidateByResourceType removes all cached states that contain the specified resource type
	InvalidateByResourceType(resourceType types.ResourceKind)

	// InvalidateByResource removes cached states containing a specific resource
	InvalidateByResource(projectID string, resourceType types.ResourceKind, resourceName string)

	// Clear removes all cached states
	Clear()

	// Stats returns cache statistics
	Stats() CacheStats
}

// CacheEntry represents a cached project state with metadata
type CacheEntry struct {
	State      *ProjectState `json:"state"`
	CachedAt   time.Time     `json:"cachedAt"`
	ExpiresAt  time.Time     `json:"expiresAt"`
	AccessedAt time.Time     `json:"accessedAt"`
	HitCount   int64         `json:"hitCount"`
    // Monotonic sequence to break ties when AccessedAt times are equal across platforms
    LastAccessSeq uint64 `json:"lastAccessSeq"`
}

// CacheStats provides statistics about cache usage
type CacheStats struct {
	Size        int           `json:"size"`
	HitCount    int64         `json:"hitCount"`
	MissCount   int64         `json:"missCount"`
	HitRate     float64       `json:"hitRate"`
	EvictCount  int64         `json:"evictCount"`
	ExpireCount int64         `json:"expireCount"`
	LastCleanup time.Time     `json:"lastCleanup"`
	MaxEntries  int           `json:"maxEntries"`
	DefaultTTL  time.Duration `json:"defaultTTL"`
}

// InMemoryStateCache implements StateCache using in-memory storage
type InMemoryStateCache struct {
	mu            sync.RWMutex
	entries       map[string]*CacheEntry
	stats         CacheStats
	maxEntries    int
	defaultTTL    time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
    // Monotonic counter incremented on every Set/Get that touches entries
    accessCounter uint64
}

// NewInMemoryStateCache creates a new in-memory state cache
func NewInMemoryStateCache(maxEntries int, defaultTTL time.Duration) *InMemoryStateCache {
	cache := &InMemoryStateCache{
		entries:     make(map[string]*CacheEntry),
		maxEntries:  maxEntries,
		defaultTTL:  defaultTTL,
		stopCleanup: make(chan struct{}),
		stats: CacheStats{
			MaxEntries: maxEntries,
			DefaultTTL: defaultTTL,
		},
	}

	// Start background cleanup goroutine
	cache.startCleanup()

	return cache
}

// Get retrieves a cached project state
func (c *InMemoryStateCache) Get(projectID string) (*ProjectState, bool) {
    // Use write lock to safely mutate access metadata and stats
    c.mu.Lock()
    defer c.mu.Unlock()

	entry, exists := c.entries[projectID]
	if !exists {
        c.stats.MissCount++
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, projectID)
		c.stats.ExpireCount++
        c.stats.MissCount++
		return nil, false
	}

	// Update access statistics
	entry.AccessedAt = time.Now()
    c.accessCounter++
    entry.LastAccessSeq = c.accessCounter
	entry.HitCount++
	c.stats.HitCount++

	return entry.State, true
}

// Set stores a project state in the cache
func (c *InMemoryStateCache) Set(projectID string, state *ProjectState, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	now := time.Now()
	entry := &CacheEntry{
		State:      state,
		CachedAt:   now,
		ExpiresAt:  now.Add(ttl),
		AccessedAt: now,
        HitCount:   0,
	}
    // Assign initial access sequence so insertion order is tracked deterministically
    c.accessCounter++
    entry.LastAccessSeq = c.accessCounter

	// Check if we need to evict entries to make room
	if len(c.entries) >= c.maxEntries {
		c.evictLRU()
	}

	c.entries[projectID] = entry
}

// Delete removes a project state from the cache
func (c *InMemoryStateCache) Delete(projectID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, projectID)
}

// InvalidateByResourceType removes all cached states that contain the specified resource type
func (c *InMemoryStateCache) InvalidateByResourceType(resourceType types.ResourceKind) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toDelete []string

	for projectID, entry := range c.entries {
		if c.stateContainsResourceType(entry.State, resourceType) {
			toDelete = append(toDelete, projectID)
		}
	}

	for _, projectID := range toDelete {
		delete(c.entries, projectID)
		c.stats.EvictCount++
	}
}

// InvalidateByResource removes cached states containing a specific resource
func (c *InMemoryStateCache) InvalidateByResource(projectID string, resourceType types.ResourceKind, resourceName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[projectID]
	if !exists {
		return
	}

	if c.stateContainsResource(entry.State, resourceType, resourceName) {
		delete(c.entries, projectID)
		c.stats.EvictCount++
	}
}

// Clear removes all cached states
func (c *InMemoryStateCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	evicted := len(c.entries)
	c.entries = make(map[string]*CacheEntry)
	c.stats.EvictCount += int64(evicted)
}

// Stats returns cache statistics
func (c *InMemoryStateCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Size = len(c.entries)

	total := stats.HitCount + stats.MissCount
	if total > 0 {
		stats.HitRate = float64(stats.HitCount) / float64(total)
	}

	return stats
}

// Stop stops the cache cleanup goroutine
func (c *InMemoryStateCache) Stop() {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}
	close(c.stopCleanup)
}

// startCleanup starts the background cleanup goroutine
func (c *InMemoryStateCache) startCleanup() {
	c.cleanupTicker = time.NewTicker(time.Minute * 5) // Clean up every 5 minutes

	go func() {
		for {
			select {
			case <-c.cleanupTicker.C:
				c.cleanup()
			case <-c.stopCleanup:
				return
			}
		}
	}()
}

// cleanup removes expired entries from the cache
func (c *InMemoryStateCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var expired []string

	for projectID, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			expired = append(expired, projectID)
		}
	}

	for _, projectID := range expired {
		delete(c.entries, projectID)
		c.stats.ExpireCount++
	}

	c.stats.LastCleanup = now
}

// evictLRU evicts the least recently used entry
func (c *InMemoryStateCache) evictLRU() {
    var lruProjectID string
    var lruSeq uint64
    var lruTime time.Time

	for projectID, entry := range c.entries {
        // Prefer sequence comparison to avoid platform-dependent time resolution ties
        if lruProjectID == "" {
            lruProjectID = projectID
            lruSeq = entry.LastAccessSeq
            lruTime = entry.AccessedAt
            continue
        }
        if entry.LastAccessSeq < lruSeq {
            lruProjectID = projectID
            lruSeq = entry.LastAccessSeq
            lruTime = entry.AccessedAt
            continue
        }
        // As a secondary check, fall back to time for older entries without seq
        if entry.LastAccessSeq == lruSeq && entry.AccessedAt.Before(lruTime) {
            lruProjectID = projectID
            lruTime = entry.AccessedAt
        }
	}

	if lruProjectID != "" {
		delete(c.entries, lruProjectID)
		c.stats.EvictCount++
	}
}

// stateContainsResourceType checks if a project state contains resources of the specified type
func (c *InMemoryStateCache) stateContainsResourceType(state *ProjectState, resourceType types.ResourceKind) bool {
	switch resourceType {
	case types.KindProject:
		return state.Project != nil
	case types.KindCluster:
		return len(state.Clusters) > 0
	case types.KindDatabaseUser:
		return len(state.DatabaseUsers) > 0
	case types.KindNetworkAccess:
		return len(state.NetworkAccess) > 0
	default:
		return false
	}
}

// stateContainsResource checks if a project state contains a specific resource
func (c *InMemoryStateCache) stateContainsResource(state *ProjectState, resourceType types.ResourceKind, resourceName string) bool {
	switch resourceType {
	case types.KindProject:
		return state.Project != nil && state.Project.Metadata.Name == resourceName
	case types.KindCluster:
		for _, cluster := range state.Clusters {
			if cluster.Metadata.Name == resourceName {
				return true
			}
		}
	case types.KindDatabaseUser:
		for _, user := range state.DatabaseUsers {
			if user.Metadata.Name == resourceName {
				return true
			}
		}
	case types.KindNetworkAccess:
		for _, entry := range state.NetworkAccess {
			if entry.Metadata.Name == resourceName {
				return true
			}
		}
	}
	return false
}

// CachedStateDiscovery wraps a StateDiscovery with caching capabilities
type CachedStateDiscovery struct {
	discovery StateDiscovery
	cache     StateCache
}

// NewCachedStateDiscovery creates a new cached state discovery service
func NewCachedStateDiscovery(discovery StateDiscovery, cache StateCache) *CachedStateDiscovery {
	return &CachedStateDiscovery{
		discovery: discovery,
		cache:     cache,
	}
}

// DiscoverProject implements StateDiscovery with caching
func (c *CachedStateDiscovery) DiscoverProject(ctx context.Context, projectID string) (*ProjectState, error) {
	// Try to get from cache first
	if cached, found := c.cache.Get(projectID); found {
		return cached, nil
	}

	// Not in cache, discover from source
	state, err := c.discovery.DiscoverProject(ctx, projectID)
	if err != nil {
		return state, err
	}

	// Cache the result
	if state != nil {
		c.cache.Set(projectID, state, 0) // Use default TTL
	}

	return state, nil
}

// DiscoverClusters implements StateDiscovery
func (c *CachedStateDiscovery) DiscoverClusters(ctx context.Context, projectID string) ([]types.ClusterManifest, error) {
	return c.discovery.DiscoverClusters(ctx, projectID)
}

// DiscoverDatabaseUsers implements StateDiscovery
func (c *CachedStateDiscovery) DiscoverDatabaseUsers(ctx context.Context, projectID string) ([]types.DatabaseUserManifest, error) {
	return c.discovery.DiscoverDatabaseUsers(ctx, projectID)
}

// DiscoverNetworkAccess implements StateDiscovery
func (c *CachedStateDiscovery) DiscoverNetworkAccess(ctx context.Context, projectID string) ([]types.NetworkAccessManifest, error) {
	return c.discovery.DiscoverNetworkAccess(ctx, projectID)
}

// DiscoverProjectSettings implements StateDiscovery
func (c *CachedStateDiscovery) DiscoverProjectSettings(ctx context.Context, projectID string) (*types.ProjectManifest, error) {
	return c.discovery.DiscoverProjectSettings(ctx, projectID)
}
