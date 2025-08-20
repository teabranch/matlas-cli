package apply

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/types"
)

// StateDiscovery defines the interface for discovering current state of resources
type StateDiscovery interface {
	// DiscoverProject fetches the complete state of a project and all its resources
	DiscoverProject(ctx context.Context, projectID string) (*ProjectState, error)

	// DiscoverClusters fetches all clusters in a project
	DiscoverClusters(ctx context.Context, projectID string) ([]types.ClusterManifest, error)

	// DiscoverDatabaseUsers fetches all database users in a project
	DiscoverDatabaseUsers(ctx context.Context, projectID string) ([]types.DatabaseUserManifest, error)

	// DiscoverNetworkAccess fetches all network access entries in a project
	DiscoverNetworkAccess(ctx context.Context, projectID string) ([]types.NetworkAccessManifest, error)

	// DiscoverSearchIndexes fetches all search indexes in a project
	DiscoverSearchIndexes(ctx context.Context, projectID string) ([]types.SearchIndexManifest, error)

	// DiscoverProjectSettings fetches project-level configuration
	DiscoverProjectSettings(ctx context.Context, projectID string) (*types.ProjectManifest, error)
	// DiscoverVPCEndpoints fetches all VPC endpoint services in a project
	DiscoverVPCEndpoints(ctx context.Context, projectID string) ([]types.VPCEndpointManifest, error)
}

// ProjectState represents the complete discovered state of an Atlas project
type ProjectState struct {
	Project       *types.ProjectManifest        `json:"project"`
	Clusters      []types.ClusterManifest       `json:"clusters"`
	DatabaseUsers []types.DatabaseUserManifest  `json:"databaseUsers"`
	DatabaseRoles []types.DatabaseRoleManifest  `json:"databaseRoles"`
	NetworkAccess []types.NetworkAccessManifest `json:"networkAccess"`
	SearchIndexes []types.SearchIndexManifest   `json:"searchIndexes"`
	VPCEndpoints  []types.VPCEndpointManifest   `json:"vpcEndpoints"`
	Fingerprint   string                        `json:"fingerprint"`
	DiscoveredAt  time.Time                     `json:"discoveredAt"`
}

// AtlasStateDiscovery implements StateDiscovery using Atlas services
type AtlasStateDiscovery struct {
	client           *atlasclient.Client
	projectsService  *atlas.ProjectsService
	clustersService  *atlas.ClustersService
	usersService     *atlas.DatabaseUsersService
	networkService   *atlas.NetworkAccessListsService
	searchService    *atlas.SearchService
	vpcService       *atlas.VPCEndpointsService
	rateLimiter      *RateLimiter
	maxConcurrentOps int
}

// NewAtlasStateDiscovery creates a new AtlasStateDiscovery instance
func NewAtlasStateDiscovery(client *atlasclient.Client) *AtlasStateDiscovery {
	return &AtlasStateDiscovery{
		client:           client,
		projectsService:  atlas.NewProjectsService(client),
		clustersService:  atlas.NewClustersService(client),
		usersService:     atlas.NewDatabaseUsersService(client),
		networkService:   atlas.NewNetworkAccessListsService(client),
		searchService:    atlas.NewSearchService(client),
		vpcService:       atlas.NewVPCEndpointsService(client),
		rateLimiter:      NewRateLimiter(10, time.Second), // 10 requests per second
		maxConcurrentOps: 5,                               // Maximum 5 concurrent API calls
	}
}

// DiscoverProject fetches the complete state of a project and all its resources
func (d *AtlasStateDiscovery) DiscoverProject(ctx context.Context, projectID string) (*ProjectState, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}

	// Create a semaphore to limit concurrent operations
	maxOps := d.maxConcurrentOps
	if maxOps <= 0 {
		maxOps = 5 // Default fallback
	}
	semaphore := make(chan struct{}, maxOps)

	// Use sync.WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	// Results channels
	type result struct {
		data interface{}
		err  error
	}

	projectCh := make(chan result, 1)
	clustersCh := make(chan result, 1)
	usersCh := make(chan result, 1)
	networkCh := make(chan result, 1)
	searchCh := make(chan result, 1)
	vpceCh := make(chan result, 1)

	// Discover project settings first
	wg.Add(1)
	go func() {
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() { <-semaphore }()

		project, err := d.DiscoverProjectSettings(ctx, projectID)
		projectCh <- result{data: project, err: err}
	}()

	// Discover database users
	wg.Add(1)
	go func() {
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() { <-semaphore }()

		users, err := d.DiscoverDatabaseUsers(ctx, projectID)
		usersCh <- result{data: users, err: err}
	}()

	// Discover network access
	wg.Add(1)
	go func() {
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() { <-semaphore }()

		network, err := d.DiscoverNetworkAccess(ctx, projectID)
		networkCh <- result{data: network, err: err}
	}()

	// Discover search indexes
	wg.Add(1)
	go func() {
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() { <-semaphore }()

		search, err := d.DiscoverSearchIndexes(ctx, projectID)
		searchCh <- result{data: search, err: err}
	}()

	// Discover VPC endpoints
	wg.Add(1)
	go func() {
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() { <-semaphore }()
		// Fetch VPC endpoint services
		servicesMap, err := d.vpcService.ListAllPrivateEndpointServices(ctx, projectID)
		var manifests []types.VPCEndpointManifest
		for provider, list := range servicesMap {
			for _, svc := range list {
				spec := types.VPCEndpointSpec{ProjectName: projectID, CloudProvider: provider, Region: svc.GetRegionName(), EndpointID: svc.GetId()}
				manifest := types.VPCEndpointManifest{APIVersion: types.APIVersionV1, Kind: types.KindVPCEndpoint, Metadata: types.ResourceMetadata{Name: svc.GetEndpointServiceName()}, Spec: spec}
				manifests = append(manifests, manifest)
			}
		}
		vpceCh <- result{data: manifests, err: err}
	}()

	// Wait for project discovery to complete first
	wg.Wait()
	close(projectCh)

	// Collect project result and check for errors
	var errors []error
	projectState := &ProjectState{
		DiscoveredAt: time.Now().UTC(),
	}

	// Get project settings first
	projectResult := <-projectCh
	var projectName string
	if projectResult.err != nil {
		errors = append(errors, fmt.Errorf("failed to discover project settings: %w", projectResult.err))
	} else if projectResult.data != nil {
		projectState.Project = projectResult.data.(*types.ProjectManifest)
		projectName = projectState.Project.Spec.Name
	}

	// Now discover clusters with the project name
	wg.Add(1)
	go func() {
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() { <-semaphore }()

		clusters, err := d.DiscoverClustersWithProjectName(ctx, projectID, projectName)
		clustersCh <- result{data: clusters, err: err}
	}()

	// Wait for remaining operations to complete
	wg.Wait()
	close(clustersCh)
	close(usersCh)
	close(networkCh)
	close(searchCh)
	close(vpceCh)

	// Clusters
	clustersResult := <-clustersCh
	if clustersResult.err != nil {
		errors = append(errors, fmt.Errorf("failed to discover clusters: %w", clustersResult.err))
	} else if clustersResult.data != nil {
		projectState.Clusters = clustersResult.data.([]types.ClusterManifest)
	}

	// Database users
	usersResult := <-usersCh
	if usersResult.err != nil {
		errors = append(errors, fmt.Errorf("failed to discover database users: %w", usersResult.err))
	} else if usersResult.data != nil {
		projectState.DatabaseUsers = usersResult.data.([]types.DatabaseUserManifest)
	}

	// Network access
	networkResult := <-networkCh
	if networkResult.err != nil {
		errors = append(errors, fmt.Errorf("failed to discover network access: %w", networkResult.err))
	} else if networkResult.data != nil {
		projectState.NetworkAccess = networkResult.data.([]types.NetworkAccessManifest)
	}

	// Search indexes
	searchResult := <-searchCh
	if searchResult.err != nil {
		errors = append(errors, fmt.Errorf("failed to discover search indexes: %w", searchResult.err))
	} else if searchResult.data != nil {
		projectState.SearchIndexes = searchResult.data.([]types.SearchIndexManifest)
	}

	// VPC endpoints
	vpceResult := <-vpceCh
	if vpceResult.err != nil {
		errors = append(errors, fmt.Errorf("failed to discover VPC endpoints: %w", vpceResult.err))
	} else if vpceResult.data != nil {
		projectState.VPCEndpoints = vpceResult.data.([]types.VPCEndpointManifest)
	}

	// Return aggregated errors if any
	if len(errors) > 0 {
		return projectState, &DiscoveryError{
			ProjectID: projectID,
			Errors:    errors,
		}
	}

	// Generate fingerprint for the complete state
	fingerprint, err := GenerateStateFingerprint(projectState)
	if err != nil {
		return projectState, fmt.Errorf("failed to generate state fingerprint: %w", err)
	}
	projectState.Fingerprint = fingerprint

	return projectState, nil
}

// DiscoveryError represents aggregated errors from parallel discovery operations
type DiscoveryError struct {
	ProjectID string
	Errors    []error
}

func (e *DiscoveryError) Error() string {
	if len(e.Errors) == 1 {
		return fmt.Sprintf("discovery failed for project %s: %v", e.ProjectID, e.Errors[0])
	}
	return fmt.Sprintf("discovery failed for project %s with %d errors: %v", e.ProjectID, len(e.Errors), e.Errors[0])
}

func (e *DiscoveryError) Unwrap() []error {
	return e.Errors
}

// DiscoverProjectSettings fetches project-level configuration
func (d *AtlasStateDiscovery) DiscoverProjectSettings(ctx context.Context, projectID string) (*types.ProjectManifest, error) {
	if err := d.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	project, err := d.projectsService.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project: %w", err)
	}

	manifest := &types.ProjectManifest{
		APIVersion: types.APIVersionV1,
		Kind:       types.KindProject,
		Metadata: types.ResourceMetadata{
			Name: project.GetName(),
			Labels: map[string]string{
				"atlas.mongodb.com/project-id": project.GetId(),
				"atlas.mongodb.com/org-id":     project.GetOrgId(),
			},
		},
		Spec: types.ProjectConfig{
			Name:           project.GetName(),
			OrganizationID: project.GetOrgId(),
		},
		Status: &types.ResourceStatusInfo{
			Phase:      types.StatusReady,
			LastUpdate: time.Now().UTC().Format(time.RFC3339),
		},
	}

	// Map Atlas project tags into spec.tags when available
	if project.Tags != nil && len(*project.Tags) > 0 {
		tagsMap := make(map[string]string, len(*project.Tags))
		for _, rt := range *project.Tags {
			key := rt.GetKey()
			val := rt.GetValue()
			if key != "" {
				tagsMap[key] = val
			}
		}
		if len(tagsMap) > 0 {
			manifest.Spec.Tags = tagsMap
		}
	}

	return manifest, nil
}

// DiscoverClusters fetches all clusters in a project
func (d *AtlasStateDiscovery) DiscoverClusters(ctx context.Context, projectID string) ([]types.ClusterManifest, error) {
	return d.DiscoverClustersWithProjectName(ctx, projectID, "")
}

// DiscoverClustersWithProjectName fetches all clusters in a project with the project name for manifest population
func (d *AtlasStateDiscovery) DiscoverClustersWithProjectName(ctx context.Context, projectID, projectName string) ([]types.ClusterManifest, error) {
	if err := d.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	clusters, err := d.clustersService.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch clusters: %w", err)
	}

	manifests := make([]types.ClusterManifest, 0, len(clusters))
	for _, cluster := range clusters {
		manifest := d.convertClusterToManifest(&cluster, projectName)
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// DiscoverDatabaseUsers fetches all database users in a project
func (d *AtlasStateDiscovery) DiscoverDatabaseUsers(ctx context.Context, projectID string) ([]types.DatabaseUserManifest, error) {
	if err := d.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Use server-side pagination to retrieve all users when supported
	users, err := d.usersService.ListWithPagination(ctx, projectID, 1, 500, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch database users: %w", err)
	}

	manifests := make([]types.DatabaseUserManifest, 0, len(users))
	for _, user := range users {
		manifest := d.convertUserToManifest(&user)
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// DiscoverNetworkAccess fetches all network access entries in a project
func (d *AtlasStateDiscovery) DiscoverNetworkAccess(ctx context.Context, projectID string) ([]types.NetworkAccessManifest, error) {
	if err := d.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	entries, err := d.networkService.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch network access entries: %w", err)
	}

	manifests := make([]types.NetworkAccessManifest, 0, len(entries))
	for _, entry := range entries {
		manifest := d.convertNetworkAccessToManifest(&entry)
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// DiscoverSearchIndexes fetches all search indexes in a project
func (d *AtlasStateDiscovery) DiscoverSearchIndexes(ctx context.Context, projectID string) ([]types.SearchIndexManifest, error) {
	if err := d.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	indexes, err := d.searchService.ListAllIndexes(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search indexes: %w", err)
	}

	manifests := make([]types.SearchIndexManifest, 0, len(indexes))
	for _, index := range indexes {
		manifest := d.convertSearchIndexToManifest(&index)
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

// DiscoverVPCEndpoints fetches all VPC endpoint services in a project
func (d *AtlasStateDiscovery) DiscoverVPCEndpoints(ctx context.Context, projectID string) ([]types.VPCEndpointManifest, error) {
	// Rate limit
	if err := d.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}
	// Use vpcService to list all endpoint services
	servicesMap, err := d.vpcService.ListAllPrivateEndpointServices(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list VPC endpoint services: %w", err)
	}
	var manifests []types.VPCEndpointManifest
	for provider, list := range servicesMap {
		for _, svc := range list {
			spec := types.VPCEndpointSpec{
				ProjectName:   projectID,
				CloudProvider: provider,
				Region:        svc.GetRegionName(),
				EndpointID:    svc.GetId(),
			}
			manifest := types.VPCEndpointManifest{
				APIVersion: types.APIVersionV1,
				Kind:       types.KindVPCEndpoint,
				Metadata:   types.ResourceMetadata{Name: svc.GetEndpointServiceName()},
				Spec:       spec,
			}
			manifests = append(manifests, manifest)
		}
	}
	return manifests, nil
}

// GenerateStateFingerprint generates a SHA256 hash of the project state for change detection
func GenerateStateFingerprint(state *ProjectState) (string, error) {
	// Create a copy of the state without the fingerprint and timestamp for consistent hashing
	hashableState := struct {
		Project       *types.ProjectManifest        `json:"project"`
		Clusters      []types.ClusterManifest       `json:"clusters"`
		DatabaseUsers []types.DatabaseUserManifest  `json:"databaseUsers"`
		NetworkAccess []types.NetworkAccessManifest `json:"networkAccess"`
		SearchIndexes []types.SearchIndexManifest   `json:"searchIndexes"`
	}{
		Project:       state.Project,
		Clusters:      state.Clusters,
		DatabaseUsers: state.DatabaseUsers,
		NetworkAccess: state.NetworkAccess,
		SearchIndexes: state.SearchIndexes,
	}

	data, err := json.Marshal(hashableState)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state for fingerprinting: %w", err)
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// RateLimiter provides simple rate limiting for API calls
type RateLimiter struct {
	ticker   *time.Ticker
	tokens   chan struct{}
	capacity int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, per time.Duration) *RateLimiter {
	rl := &RateLimiter{
		ticker:   time.NewTicker(per / time.Duration(rate)),
		tokens:   make(chan struct{}, rate),
		capacity: rate,
	}

	// Fill the token bucket initially
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}

	// Refill tokens at the specified rate
	go func() {
		for range rl.ticker.C {
			select {
			case rl.tokens <- struct{}{}:
			default:
				// Bucket is full, skip
			}
		}
	}()

	return rl
}

// Wait waits for a token to become available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
}
