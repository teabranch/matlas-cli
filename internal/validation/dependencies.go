// Package validation implements schema and cross-resource dependency validation.
package validation

import (
	"fmt"
	"strings"
	"time"

	"github.com/teabranch/matlas-cli/internal/types"
)

// DependencyValidator validates cross-resource dependencies
type DependencyValidator struct {
	// Configuration for validation behavior
	StrictMode         bool
	CheckExistingState bool
	AllowCircularDeps  bool
	MaxDepthCheck      int
}

// NewDependencyValidator creates a new dependency validator
func NewDependencyValidator(strictMode bool) *DependencyValidator {
	return &DependencyValidator{
		StrictMode:         strictMode,
		CheckExistingState: true,
		AllowCircularDeps:  false,
		MaxDepthCheck:      10,
	}
}

// DependencyIssue represents a dependency validation problem
type DependencyIssue struct {
	SourceResource  string   `json:"sourceResource"`
	TargetResource  string   `json:"targetResource"`
	DependencyType  string   `json:"dependencyType"`
	Severity        string   `json:"severity"`
	Message         string   `json:"message"`
	Suggestions     []string `json:"suggestions,omitempty"`
	ResolutionSteps []string `json:"resolutionSteps,omitempty"`
}

// Error implements the error interface
func (di DependencyIssue) Error() string {
	return fmt.Sprintf("%s -> %s (%s): %s", di.SourceResource, di.TargetResource, di.DependencyType, di.Message)
}

// DependencyGraph represents the dependency relationships
type DependencyGraph struct {
	Nodes map[string]*ResourceNode `json:"nodes"`
	Edges []*DependencyEdge        `json:"edges"`
}

// ResourceNode represents a resource in the dependency graph
type ResourceNode struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Name         string                 `json:"name"`
	Dependencies []string               `json:"dependencies"`
	Dependents   []string               `json:"dependents"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// DependencyEdge represents a dependency relationship
type DependencyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Weight int    `json:"weight"` // For ordering operations
}

// ValidateProjectDependencies validates dependencies within a project configuration
func (dv *DependencyValidator) ValidateProjectDependencies(config *types.ProjectConfig) ([]DependencyIssue, error) {
	var issues []DependencyIssue

	// Build dependency graph
	graph, err := dv.buildProjectDependencyGraph(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Check for circular dependencies
	if !dv.AllowCircularDeps {
		circularIssues := dv.checkCircularDependencies(graph)
		issues = append(issues, circularIssues...)
	}

	// Validate cluster dependencies
	clusterIssues := dv.validateClusterDependencies(config, graph)
	issues = append(issues, clusterIssues...)

	// Validate database user dependencies
	userIssues := dv.validateDatabaseUserDependencies(config, graph)
	issues = append(issues, userIssues...)

	// Validate network access dependencies
	networkIssues := dv.validateNetworkAccessDependencies(config, graph)
	issues = append(issues, networkIssues...)

	// Validate resource naming conflicts
	namingIssues := dv.validateResourceNaming(config)
	issues = append(issues, namingIssues...)

	return issues, nil
}

// buildProjectDependencyGraph creates a dependency graph for the project
func (dv *DependencyValidator) buildProjectDependencyGraph(config *types.ProjectConfig) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Nodes: make(map[string]*ResourceNode),
		Edges: []*DependencyEdge{},
	}

	// Add project node
	projectID := fmt.Sprintf("project:%s", config.Name)
	graph.Nodes[projectID] = &ResourceNode{
		ID:           projectID,
		Type:         "project",
		Name:         config.Name,
		Dependencies: []string{},
		Dependents:   []string{},
		Metadata: map[string]interface{}{
			"organizationId": config.OrganizationID,
		},
	}

	// Add cluster nodes
	for _, cluster := range config.Clusters {
		clusterID := fmt.Sprintf("cluster:%s", cluster.Metadata.Name)
		graph.Nodes[clusterID] = &ResourceNode{
			ID:           clusterID,
			Type:         "cluster",
			Name:         cluster.Metadata.Name,
			Dependencies: []string{projectID},
			Dependents:   []string{},
			Metadata: map[string]interface{}{
				"instanceSize": cluster.InstanceSize,
				"provider":     cluster.Provider,
				"region":       cluster.Region,
			},
		}

		// Add edge from cluster to project
		graph.Edges = append(graph.Edges, &DependencyEdge{
			Source: clusterID,
			Target: projectID,
			Type:   "requires",
			Weight: 1,
		})
	}

	// Add database user nodes
	for _, user := range config.DatabaseUsers {
		userID := fmt.Sprintf("user:%s", user.Username)
		dependencies := []string{projectID}

		// Users depend on clusters if scoped to specific clusters
		for _, scope := range user.Scopes {
			if scope.Type == "CLUSTER" {
				clusterID := fmt.Sprintf("cluster:%s", scope.Name)
				dependencies = append(dependencies, clusterID)
			}
		}

		graph.Nodes[userID] = &ResourceNode{
			ID:           userID,
			Type:         "database_user",
			Name:         user.Username,
			Dependencies: dependencies,
			Dependents:   []string{},
			Metadata: map[string]interface{}{
				"roles": user.Roles,
			},
		}

		// Add edges
		for _, dep := range dependencies {
			graph.Edges = append(graph.Edges, &DependencyEdge{
				Source: userID,
				Target: dep,
				Type:   "requires",
				Weight: 2,
			})
		}
	}

	// Add network access nodes
	for i, netAccess := range config.NetworkAccess {
		netID := fmt.Sprintf("network:%d", i)
		graph.Nodes[netID] = &ResourceNode{
			ID:           netID,
			Type:         "network_access",
			Name:         fmt.Sprintf("network-rule-%d", i),
			Dependencies: []string{projectID},
			Dependents:   []string{},
			Metadata: map[string]interface{}{
				"ipAddress": netAccess.IPAddress,
				"cidr":      netAccess.CIDR,
			},
		}

		// Add edge from network access to project
		graph.Edges = append(graph.Edges, &DependencyEdge{
			Source: netID,
			Target: projectID,
			Type:   "requires",
			Weight: 1,
		})
	}

	return graph, nil
}

// checkCircularDependencies detects circular dependency chains
func (dv *DependencyValidator) checkCircularDependencies(graph *DependencyGraph) []DependencyIssue {
	var issues []DependencyIssue
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	for nodeID := range graph.Nodes {
		if !visited[nodeID] {
			if cycle := dv.detectCycle(nodeID, graph, visited, recursionStack, []string{}); len(cycle) > 0 {
				issues = append(issues, DependencyIssue{
					SourceResource: cycle[0],
					TargetResource: cycle[len(cycle)-1],
					DependencyType: "circular",
					Severity:       "error",
					Message:        fmt.Sprintf("Circular dependency detected: %s", strings.Join(cycle, " -> ")),
					Suggestions: []string{
						"Remove one of the dependencies to break the cycle",
						"Restructure resources to avoid circular references",
					},
					ResolutionSteps: []string{
						"1. Identify which dependency can be safely removed",
						"2. Update resource configuration to remove the dependency",
						"3. Re-validate the configuration",
					},
				})
			}
		}
	}

	return issues
}

// detectCycle performs DFS to detect cycles in the dependency graph
func (dv *DependencyValidator) detectCycle(nodeID string, graph *DependencyGraph, visited, recursionStack map[string]bool, path []string) []string {
	if recursionStack[nodeID] {
		// Found a cycle, return the path
		cycleStart := -1
		for i, p := range path {
			if p == nodeID {
				cycleStart = i
				break
			}
		}
		if cycleStart >= 0 {
			return append(path[cycleStart:], nodeID)
		}
		return path
	}

	if visited[nodeID] {
		return nil
	}

	visited[nodeID] = true
	recursionStack[nodeID] = true
	currentPath := append(path, nodeID)

	// Check dependencies
	if node := graph.Nodes[nodeID]; node != nil {
		for _, dep := range node.Dependencies {
			if cycle := dv.detectCycle(dep, graph, visited, recursionStack, currentPath); len(cycle) > 0 {
				return cycle
			}
		}
	}

	recursionStack[nodeID] = false
	return nil
}

// validateClusterDependencies validates cluster-specific dependencies
func (dv *DependencyValidator) validateClusterDependencies(config *types.ProjectConfig, graph *DependencyGraph) []DependencyIssue {
	var issues []DependencyIssue

	for _, cluster := range config.Clusters {
		// Validate provider-region compatibility
		if !dv.isProviderRegionCompatible(cluster.Provider, cluster.Region) {
			issues = append(issues, DependencyIssue{
				SourceResource: fmt.Sprintf("cluster:%s", cluster.Metadata.Name),
				TargetResource: "provider-region",
				DependencyType: "compatibility",
				Severity:       "error",
				Message:        fmt.Sprintf("Provider %s does not support region %s", cluster.Provider, cluster.Region),
				Suggestions: []string{
					fmt.Sprintf("Use a valid region for %s provider", cluster.Provider),
					"Check Atlas documentation for supported regions",
				},
				ResolutionSteps: []string{
					"1. Check Atlas supported regions for your provider",
					"2. Update the region in your cluster configuration",
					"3. Re-validate the configuration",
				},
			})
		}

		// Validate instance size for provider
		if !dv.isInstanceSizeAvailableInRegion(cluster.InstanceSize, cluster.Provider, cluster.Region) {
			issues = append(issues, DependencyIssue{
				SourceResource: fmt.Sprintf("cluster:%s", cluster.Metadata.Name),
				TargetResource: "instance-availability",
				DependencyType: "availability",
				Severity:       "warning",
				Message:        fmt.Sprintf("Instance size %s may not be available in %s region on %s", cluster.InstanceSize, cluster.Region, cluster.Provider),
				Suggestions: []string{
					"Verify instance size availability in the target region",
					"Consider using a different instance size or region",
				},
			})
		}
	}

	return issues
}

// validateDatabaseUserDependencies validates database user dependencies
func (dv *DependencyValidator) validateDatabaseUserDependencies(config *types.ProjectConfig, graph *DependencyGraph) []DependencyIssue {
	var issues []DependencyIssue

	clusterNames := make(map[string]bool)
	for _, cluster := range config.Clusters {
		clusterNames[cluster.Metadata.Name] = true
	}

	for _, user := range config.DatabaseUsers {
		// Check that scoped clusters exist
		for _, scope := range user.Scopes {
			if scope.Type == "CLUSTER" && !clusterNames[scope.Name] {
				issues = append(issues, DependencyIssue{
					SourceResource: fmt.Sprintf("user:%s", user.Username),
					TargetResource: fmt.Sprintf("cluster:%s", scope.Name),
					DependencyType: "missing_resource",
					Severity:       "error",
					Message:        fmt.Sprintf("Database user %s references non-existent cluster %s", user.Username, scope.Name),
					Suggestions: []string{
						fmt.Sprintf("Create cluster %s before creating the database user", scope.Name),
						"Remove the cluster scope from the database user",
						"Check the cluster name for typos",
					},
					ResolutionSteps: []string{
						"1. Verify the cluster name is correct",
						"2. Ensure the cluster is defined in the same configuration",
						"3. Check the dependency order in your apply plan",
					},
				})
			}
		}

		// Validate role-database combinations
		for _, role := range user.Roles {
			if role.DatabaseName != "" && dv.StrictMode {
				// In strict mode, warn about custom databases
				issues = append(issues, DependencyIssue{
					SourceResource: fmt.Sprintf("user:%s", user.Username),
					TargetResource: fmt.Sprintf("database:%s", role.DatabaseName),
					DependencyType: "database_reference",
					Severity:       "warning",
					Message:        fmt.Sprintf("User %s references database %s which may not exist", user.Username, role.DatabaseName),
					Suggestions: []string{
						"Ensure the database exists before applying",
						"Use 'admin' for built-in administrative roles",
					},
				})
			}
		}
	}

	return issues
}

// validateNetworkAccessDependencies validates network access dependencies
func (dv *DependencyValidator) validateNetworkAccessDependencies(config *types.ProjectConfig, graph *DependencyGraph) []DependencyIssue {
	var issues []DependencyIssue

	// Check for overlapping network rules
	for i, netAccess1 := range config.NetworkAccess {
		for j, netAccess2 := range config.NetworkAccess {
			if i >= j {
				continue
			}

			if dv.networkRulesOverlap(netAccess1, netAccess2) {
				issues = append(issues, DependencyIssue{
					SourceResource: fmt.Sprintf("network:%d", i),
					TargetResource: fmt.Sprintf("network:%d", j),
					DependencyType: "overlap",
					Severity:       "warning",
					Message:        "Network access rules have overlapping IP ranges",
					Suggestions: []string{
						"Consolidate overlapping network rules",
						"Use broader CIDR ranges instead of multiple specific IPs",
					},
				})
			}
		}
	}

	// Check for expiration dependencies
	now := time.Now()
	for i, netAccess := range config.NetworkAccess {
		if netAccess.DeleteAfterDate != "" {
			if expiry, err := time.Parse(time.RFC3339, netAccess.DeleteAfterDate); err == nil {
				if expiry.Before(now) {
					issues = append(issues, DependencyIssue{
						SourceResource: fmt.Sprintf("network:%d", i),
						TargetResource: "time",
						DependencyType: "temporal",
						Severity:       "error",
						Message:        "Network access rule has expiration date in the past",
						Suggestions: []string{
							"Update the expiration date to a future time",
							"Remove the expiration date if not needed",
						},
					})
				} else if expiry.Before(now.Add(24 * time.Hour)) {
					issues = append(issues, DependencyIssue{
						SourceResource: fmt.Sprintf("network:%d", i),
						TargetResource: "time",
						DependencyType: "temporal",
						Severity:       "warning",
						Message:        "Network access rule expires within 24 hours",
						Suggestions: []string{
							"Consider extending the expiration date",
							"Ensure this is intentional for short-term access",
						},
					})
				}
			}
		}
	}

	return issues
}

// validateResourceNaming checks for naming conflicts across resource types
func (dv *DependencyValidator) validateResourceNaming(config *types.ProjectConfig) []DependencyIssue {
	var issues []DependencyIssue
	names := make(map[string][]string)

	// Collect all resource names
	names["cluster"] = []string{}
	for _, cluster := range config.Clusters {
		names["cluster"] = append(names["cluster"], cluster.Metadata.Name)
	}

	names["user"] = []string{}
	for _, user := range config.DatabaseUsers {
		names["user"] = append(names["user"], user.Username)
	}

	// Check for duplicates within each type
	for resourceType, nameList := range names {
		seen := make(map[string]int)
		for i, name := range nameList {
			if firstIndex, exists := seen[name]; exists {
				issues = append(issues, DependencyIssue{
					SourceResource: fmt.Sprintf("%s:%s[%d]", resourceType, name, i),
					TargetResource: fmt.Sprintf("%s:%s[%d]", resourceType, name, firstIndex),
					DependencyType: "naming_conflict",
					Severity:       "error",
					Message:        fmt.Sprintf("Duplicate %s name: %s", resourceType, name),
					Suggestions: []string{
						fmt.Sprintf("Use unique names for each %s", resourceType),
						"Add a suffix or prefix to differentiate resources",
					},
					ResolutionSteps: []string{
						"1. Choose unique names for each resource",
						"2. Update the configuration file",
						"3. Re-validate the configuration",
					},
				})
			}
			seen[name] = i
		}
	}

	return issues
}

// Helper functions for business logic validation

func (dv *DependencyValidator) isProviderRegionCompatible(provider, region string) bool {
	// Simplified provider-region compatibility check
	// In production, this would query Atlas API for actual supported regions
	providerRegions := map[string][]string{
		"AWS":   {"US_EAST_1", "us-west-2", "eu-west-1", "ap-southeast-1"},
		"GCP":   {"us-central1", "europe-west1", "asia-southeast1"},
		"AZURE": {"eastus", "westeurope", "southeastasia"},
	}

	supportedRegions, exists := providerRegions[strings.ToUpper(provider)]
	if !exists {
		return false
	}

	for _, supportedRegion := range supportedRegions {
		if region == supportedRegion {
			return true
		}
	}
	return false
}

func (dv *DependencyValidator) isInstanceSizeAvailableInRegion(instanceSize, provider, region string) bool {
	// Simplified availability check
	// In production, this would check actual Atlas availability
	restrictedSizes := map[string]map[string][]string{
		"AWS": {
			"US_EAST_1":      {},               // All sizes available
			"ap-southeast-1": {"M700", "R700"}, // Some large sizes not available
		},
	}

	if restrictions, exists := restrictedSizes[strings.ToUpper(provider)][region]; exists {
		for _, restricted := range restrictions {
			if instanceSize == restricted {
				return false
			}
		}
	}
	return true
}

func (dv *DependencyValidator) networkRulesOverlap(rule1, rule2 types.NetworkAccessConfig) bool {
	// Simple overlap detection for IP addresses and CIDR ranges
	// In production, this would use proper IP range overlap detection

	if rule1.IPAddress != "" && rule2.IPAddress != "" {
		return rule1.IPAddress == rule2.IPAddress
	}

	// For CIDR ranges, this is a simplified check
	if rule1.CIDR != "" && rule2.CIDR != "" {
		return rule1.CIDR == rule2.CIDR
	}

	return false
}
