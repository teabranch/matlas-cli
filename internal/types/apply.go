// Package types contains shared domain types used by apply, services and CLI layers.
package types

import (
	"fmt"
)

// APIVersion represents the supported API versions for configuration files.
type APIVersion string

const (
	APIVersionV1Alpha1 APIVersion = "matlas.mongodb.com/v1alpha1"
	APIVersionV1Beta1  APIVersion = "matlas.mongodb.com/v1beta1"
	APIVersionV1       APIVersion = "matlas.mongodb.com/v1"
)

// ResourceKind represents the type of resource being configured.
type ResourceKind string

const (
	KindProject       ResourceKind = "Project"
	KindCluster       ResourceKind = "Cluster"
	KindDatabaseUser  ResourceKind = "DatabaseUser"
	KindDatabaseRole  ResourceKind = "DatabaseRole"
	KindNetworkAccess ResourceKind = "NetworkAccess"
	KindSearchIndex   ResourceKind = "SearchIndex"
	KindVPCEndpoint   ResourceKind = "VPCEndpoint"
	KindApplyDocument ResourceKind = "ApplyDocument"
)

// ResourceStatus represents the current status of a resource.
type ResourceStatus string

const (
	StatusPending  ResourceStatus = "Pending"
	StatusCreating ResourceStatus = "Creating"
	StatusReady    ResourceStatus = "Ready"
	StatusUpdating ResourceStatus = "Updating"
	StatusDeleting ResourceStatus = "Deleting"
	StatusError    ResourceStatus = "Error"
	StatusUnknown  ResourceStatus = "Unknown"
)

// ApplyDocument represents a document that can contain multiple resources
type ApplyDocument struct {
	APIVersion APIVersion         `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind       `yaml:"kind" json:"kind"`
	Metadata   MetadataConfig     `yaml:"metadata" json:"metadata"`
	Resources  []ResourceManifest `yaml:"resources" json:"resources"`
}

// ResourceManifest represents a single resource within an ApplyDocument
type ResourceManifest struct {
	APIVersion APIVersion          `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind        `yaml:"kind" json:"kind"`
	Metadata   ResourceMetadata    `yaml:"metadata" json:"metadata"`
	Spec       interface{}         `yaml:"spec" json:"spec"`
	Status     *ResourceStatusInfo `yaml:"status,omitempty" json:"status,omitempty"`
}

// ResourceStatusInfo contains detailed status information about a resource
type ResourceStatusInfo struct {
	Phase      ResourceStatus    `yaml:"phase" json:"phase"`
	Message    string            `yaml:"message,omitempty" json:"message,omitempty"`
	Reason     string            `yaml:"reason,omitempty" json:"reason,omitempty"`
	LastUpdate string            `yaml:"lastUpdate,omitempty" json:"lastUpdate,omitempty"`
	Conditions []StatusCondition `yaml:"conditions,omitempty" json:"conditions,omitempty"`
}

// StatusCondition represents a condition of a resource's status
type StatusCondition struct {
	Type               string `yaml:"type" json:"type"`
	Status             string `yaml:"status" json:"status"`
	LastTransitionTime string `yaml:"lastTransitionTime" json:"lastTransitionTime"`
	Reason             string `yaml:"reason,omitempty" json:"reason,omitempty"`
	Message            string `yaml:"message,omitempty" json:"message,omitempty"`
}

// ProjectManifest represents a project resource manifest
type ProjectManifest struct {
	APIVersion APIVersion          `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind        `yaml:"kind" json:"kind"`
	Metadata   ResourceMetadata    `yaml:"metadata" json:"metadata"`
	Spec       ProjectConfig       `yaml:"spec" json:"spec"`
	Status     *ResourceStatusInfo `yaml:"status,omitempty" json:"status,omitempty"`
}

// ClusterManifest represents a cluster resource manifest
type ClusterManifest struct {
	APIVersion APIVersion          `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind        `yaml:"kind" json:"kind"`
	Metadata   ResourceMetadata    `yaml:"metadata" json:"metadata"`
	Spec       ClusterSpec         `yaml:"spec" json:"spec"`
	Status     *ResourceStatusInfo `yaml:"status,omitempty" json:"status,omitempty"`
}

// ClusterSpec represents the specification for a cluster resource
type ClusterSpec struct {
	ProjectName      string             `yaml:"projectName" json:"projectName"`
	Provider         string             `yaml:"provider" json:"provider"`
	Region           string             `yaml:"region" json:"region"`
	InstanceSize     string             `yaml:"instanceSize" json:"instanceSize"`
	DiskSizeGB       *float64           `yaml:"diskSizeGB,omitempty" json:"diskSizeGB,omitempty"`
	BackupEnabled    *bool              `yaml:"backupEnabled,omitempty" json:"backupEnabled,omitempty"`
	TierType         string             `yaml:"tierType,omitempty" json:"tierType,omitempty"`
	MongoDBVersion   string             `yaml:"mongodbVersion,omitempty" json:"mongodbVersion,omitempty"`
	ClusterType      string             `yaml:"clusterType,omitempty" json:"clusterType,omitempty"`
	ReplicationSpecs []ReplicationSpec  `yaml:"replicationSpecs,omitempty" json:"replicationSpecs,omitempty"`
	AutoScaling      *AutoScalingConfig `yaml:"autoScaling,omitempty" json:"autoScaling,omitempty"`
	Encryption       *EncryptionConfig  `yaml:"encryption,omitempty" json:"encryption,omitempty"`
	BiConnector      *BiConnectorConfig `yaml:"biConnector,omitempty" json:"biConnector,omitempty"`
	Tags             map[string]string  `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// SearchIndexManifest represents a search index resource manifest
type SearchIndexManifest struct {
	APIVersion APIVersion          `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind        `yaml:"kind" json:"kind"`
	Metadata   ResourceMetadata    `yaml:"metadata" json:"metadata"`
	Spec       SearchIndexSpec     `yaml:"spec" json:"spec"`
	Status     *ResourceStatusInfo `yaml:"status,omitempty" json:"status,omitempty"`
}

// SearchIndexSpec represents the specification for a search index resource
type SearchIndexSpec struct {
	ProjectName    string                 `yaml:"projectName" json:"projectName"`
	ClusterName    string                 `yaml:"clusterName" json:"clusterName"`
	DatabaseName   string                 `yaml:"databaseName" json:"databaseName"`
	CollectionName string                 `yaml:"collectionName" json:"collectionName"`
	IndexName      string                 `yaml:"indexName" json:"indexName"`
	IndexType      string                 `yaml:"indexType,omitempty" json:"indexType,omitempty"` // "search" or "vectorSearch"
	Definition     map[string]interface{} `yaml:"definition" json:"definition"`
	DependsOn      []string               `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
}

// VPCEndpointManifest represents a VPC endpoint resource manifest
type VPCEndpointManifest struct {
	APIVersion APIVersion          `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind        `yaml:"kind" json:"kind"`
	Metadata   ResourceMetadata    `yaml:"metadata" json:"metadata"`
	Spec       VPCEndpointSpec     `yaml:"spec" json:"spec"`
	Status     *ResourceStatusInfo `yaml:"status,omitempty" json:"status,omitempty"`
}

// VPCEndpointSpec represents the specification for a VPC endpoint resource
type VPCEndpointSpec struct {
	ProjectName   string   `yaml:"projectName" json:"projectName"`
	CloudProvider string   `yaml:"cloudProvider" json:"cloudProvider"` // AWS, AZURE, GCP
	Region        string   `yaml:"region" json:"region"`
	EndpointID    string   `yaml:"endpointId,omitempty" json:"endpointId,omitempty"`
	DependsOn     []string `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty"`
}

// DatabaseUserManifest represents a database user resource manifest
type DatabaseUserManifest struct {
	APIVersion APIVersion          `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind        `yaml:"kind" json:"kind"`
	Metadata   ResourceMetadata    `yaml:"metadata" json:"metadata"`
	Spec       DatabaseUserSpec    `yaml:"spec" json:"spec"`
	Status     *ResourceStatusInfo `yaml:"status,omitempty" json:"status,omitempty"`
}

// DatabaseUserSpec represents the specification for a database user resource
type DatabaseUserSpec struct {
	ProjectName  string               `yaml:"projectName" json:"projectName"`
	Username     string               `yaml:"username" json:"username"`
	Password     string               `yaml:"password,omitempty" json:"password,omitempty"`
	Roles        []DatabaseRoleConfig `yaml:"roles" json:"roles"`
	AuthDatabase string               `yaml:"authDatabase,omitempty" json:"authDatabase,omitempty"`
	Scopes       []UserScopeConfig    `yaml:"scopes,omitempty" json:"scopes,omitempty"`
}


// DatabaseRoleManifest represents a database role resource manifest
type DatabaseRoleManifest struct {
	APIVersion APIVersion          `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind        `yaml:"kind" json:"kind"`
	Metadata   ResourceMetadata    `yaml:"metadata" json:"metadata"`
	Spec       DatabaseRoleSpec    `yaml:"spec" json:"spec"`
	Status     *ResourceStatusInfo `yaml:"status,omitempty" json:"status,omitempty"`
}

// DatabaseRoleSpec represents the specification for a database role resource
type DatabaseRoleSpec struct {
	RoleName       string                          `yaml:"roleName" json:"roleName"`
	DatabaseName   string                          `yaml:"databaseName" json:"databaseName"`
	Privileges     []CustomRolePrivilegeConfig     `yaml:"privileges,omitempty" json:"privileges,omitempty"`
	InheritedRoles []CustomRoleInheritedRoleConfig `yaml:"inheritedRoles,omitempty" json:"inheritedRoles,omitempty"`
}

// NetworkAccessManifest represents a network access resource manifest
type NetworkAccessManifest struct {
	APIVersion APIVersion          `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind        `yaml:"kind" json:"kind"`
	Metadata   ResourceMetadata    `yaml:"metadata" json:"metadata"`
	Spec       NetworkAccessSpec   `yaml:"spec" json:"spec"`
	Status     *ResourceStatusInfo `yaml:"status,omitempty" json:"status,omitempty"`
}

// NetworkAccessSpec represents the specification for a network access resource
type NetworkAccessSpec struct {
	ProjectName      string `yaml:"projectName" json:"projectName"`
	IPAddress        string `yaml:"ipAddress,omitempty" json:"ipAddress,omitempty"`
	CIDR             string `yaml:"cidr,omitempty" json:"cidr,omitempty"`
	AWSSecurityGroup string `yaml:"awsSecurityGroup,omitempty" json:"awsSecurityGroup,omitempty"`
	Comment          string `yaml:"comment,omitempty" json:"comment,omitempty"`
	DeleteAfterDate  string `yaml:"deleteAfterDate,omitempty" json:"deleteAfterDate,omitempty"`
}

// DependencyGraph represents the dependency relationships between resources
type DependencyGraph struct {
	Resources    map[string]*ResourceNode `json:"resources"`
	Dependencies map[string][]string      `json:"dependencies"`
}

// ResourceNode represents a node in the dependency graph
type ResourceNode struct {
	Name         string            `json:"name"`
	Kind         ResourceKind      `json:"kind"`
	Namespace    string            `json:"namespace,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Resources:    make(map[string]*ResourceNode),
		Dependencies: make(map[string][]string),
	}
}

// AddResource adds a resource to the dependency graph
func (dg *DependencyGraph) AddResource(resource *ResourceNode) {
	key := resourceKey(resource.Namespace, resource.Name)
	dg.Resources[key] = resource
	dg.Dependencies[key] = resource.Dependencies
}

// GetDependencies returns the dependencies for a given resource
func (dg *DependencyGraph) GetDependencies(namespace, name string) []string {
	key := resourceKey(namespace, name)
	return dg.Dependencies[key]
}

// HasCycles detects if there are circular dependencies in the graph
func (dg *DependencyGraph) HasCycles() (bool, []string) {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cycle []string

	for resource := range dg.Resources {
		if !visited[resource] {
			if dg.hasCycleUtil(resource, visited, recStack, &cycle) {
				return true, cycle
			}
		}
	}
	return false, nil
}

// TopologicalSort returns resources in dependency order
func (dg *DependencyGraph) TopologicalSort() ([]string, error) {
	if hasCycle, cycle := dg.HasCycles(); hasCycle {
		return nil, fmt.Errorf("circular dependency detected: %v", cycle)
	}

	// Calculate in-degree for each resource
	inDegree := make(map[string]int)
	for resource := range dg.Resources {
		inDegree[resource] = 0
	}

	// Count in-degree for each resource
	// If resource depends on dep, then resource has incoming edge from dep
	for resource := range dg.Resources {
		inDegree[resource] = len(dg.Dependencies[resource])
	}

	// Find all resources with no incoming edges
	var queue []string
	for resource := range dg.Resources {
		if inDegree[resource] == 0 {
			queue = append(queue, resource)
		}
	}

	var result []string

	// Process resources with no dependencies first
	for len(queue) > 0 {
		// Remove a resource with no incoming edges
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each resource that depends on current
		for resource := range dg.Resources {
			for _, dep := range dg.Dependencies[resource] {
				if dep == current {
					inDegree[resource]--
					if inDegree[resource] == 0 {
						queue = append(queue, resource)
					}
				}
			}
		}
	}

	return result, nil
}

// ValidateAPIVersion validates the API version
func ValidateAPIVersion(version APIVersion) error {
	switch version {
	case APIVersionV1Alpha1, APIVersionV1Beta1, APIVersionV1:
		return nil
	default:
		return fmt.Errorf("unsupported API version: %s", version)
	}
}

// ValidateResourceKind validates the resource kind
func ValidateResourceKind(kind ResourceKind) error {
	switch kind {
	case KindProject, KindCluster, KindDatabaseUser, KindDatabaseRole, KindNetworkAccess, KindApplyDocument, KindSearchIndex, KindVPCEndpoint:
		return nil
	default:
		return fmt.Errorf("unsupported resource kind: %s", kind)
	}
}

// Helper functions

func resourceKey(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

func (dg *DependencyGraph) hasCycleUtil(resource string, visited, recStack map[string]bool, cycle *[]string) bool {
	visited[resource] = true
	recStack[resource] = true
	*cycle = append(*cycle, resource)

	for _, dep := range dg.Dependencies[resource] {
		if !visited[dep] {
			if dg.hasCycleUtil(dep, visited, recStack, cycle) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[resource] = false
	if len(*cycle) > 0 && (*cycle)[len(*cycle)-1] == resource {
		*cycle = (*cycle)[:len(*cycle)-1]
	}
	return false
}

// topologicalSortUtil is no longer needed with Kahn's algorithm implementation
