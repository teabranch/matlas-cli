package types

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAPIVersion_Constants(t *testing.T) {
	tests := []struct {
		name     string
		version  APIVersion
		expected string
	}{
		{"V1Alpha1", APIVersionV1Alpha1, "matlas.mongodb.com/v1alpha1"},
		{"V1Beta1", APIVersionV1Beta1, "matlas.mongodb.com/v1beta1"},
		{"V1", APIVersionV1, "matlas.mongodb.com/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.version) != tt.expected {
				t.Errorf("APIVersion %s = %s, want %s", tt.name, tt.version, tt.expected)
			}
		})
	}
}

func TestResourceKind_Constants(t *testing.T) {
	tests := []struct {
		name     string
		kind     ResourceKind
		expected string
	}{
		{"Project", KindProject, "Project"},
		{"Cluster", KindCluster, "Cluster"},
		{"DatabaseUser", KindDatabaseUser, "DatabaseUser"},
		{"DatabaseDirectUser", KindDatabaseDirectUser, "DatabaseDirectUser"},
		{"DatabaseRole", KindDatabaseRole, "DatabaseRole"},
		{"NetworkAccess", KindNetworkAccess, "NetworkAccess"},
		{"ApplyDocument", KindApplyDocument, "ApplyDocument"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.kind) != tt.expected {
				t.Errorf("ResourceKind %s = %s, want %s", tt.name, tt.kind, tt.expected)
			}
		})
	}
}

func TestResourceStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   ResourceStatus
		expected string
	}{
		{"Pending", StatusPending, "Pending"},
		{"Creating", StatusCreating, "Creating"},
		{"Ready", StatusReady, "Ready"},
		{"Updating", StatusUpdating, "Updating"},
		{"Deleting", StatusDeleting, "Deleting"},
		{"Error", StatusError, "Error"},
		{"Unknown", StatusUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("ResourceStatus %s = %s, want %s", tt.name, tt.status, tt.expected)
			}
		})
	}
}

func TestApplyDocument_YAMLMarshaling(t *testing.T) {
	doc := ApplyDocument{
		APIVersion: APIVersionV1,
		Kind:       KindApplyDocument,
		Metadata: MetadataConfig{
			Name:      "multi-resource-config",
			Namespace: "production",
			Labels: map[string]string{
				"environment": "prod",
				"version":     "1.2.3",
			},
		},
		Resources: []ResourceManifest{
			{
				APIVersion: APIVersionV1,
				Kind:       KindCluster,
				Metadata: ResourceMetadata{
					Name: "production-cluster",
					Labels: map[string]string{
						"tier": "production",
					},
				},
				Spec: ClusterSpec{
					ProjectName:  "production-project",
					Provider:     "AWS",
					Region:       "us-west-2",
					InstanceSize: "M30",
				},
			},
		},
	}

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("YAML marshal failed: %v", err)
	}

	// Test YAML unmarshaling
	var unmarshaled ApplyDocument
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	if err != nil {
		t.Fatalf("YAML unmarshal failed: %v", err)
	}

	// Verify fields
	if unmarshaled.APIVersion != doc.APIVersion {
		t.Errorf("APIVersion = %s, want %s", unmarshaled.APIVersion, doc.APIVersion)
	}
	if unmarshaled.Kind != doc.Kind {
		t.Errorf("Kind = %s, want %s", unmarshaled.Kind, doc.Kind)
	}
	if len(unmarshaled.Resources) != 1 {
		t.Errorf("Resources length = %d, want 1", len(unmarshaled.Resources))
	}
	if unmarshaled.Resources[0].Kind != KindCluster {
		t.Errorf("Resources[0].Kind = %s, want %s", unmarshaled.Resources[0].Kind, KindCluster)
	}
}

func TestResourceManifest_YAMLMarshaling(t *testing.T) {
	manifest := ResourceManifest{
		APIVersion: APIVersionV1Beta1,
		Kind:       KindDatabaseUser,
		Metadata: ResourceMetadata{
			Name: "app-user",
			Labels: map[string]string{
				"application": "backend",
			},
			DeletionPolicy: DeletionPolicyRetain,
		},
		Spec: DatabaseUserSpec{
			ProjectName: "my-project",
			Username:    "backend-service",
			Roles: []DatabaseRoleConfig{
				{
					RoleName:     "readWrite",
					DatabaseName: "app-db",
				},
			},
		},
		Status: &ResourceStatusInfo{
			Phase:   StatusReady,
			Message: "User is ready for use",
			Conditions: []StatusCondition{
				{
					Type:               "Ready",
					Status:             "True",
					LastTransitionTime: "2024-01-15T10:30:00Z",
					Reason:             "UserCreated",
					Message:            "Database user created successfully",
				},
			},
		},
	}

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatalf("YAML marshal failed: %v", err)
	}

	// Test YAML unmarshaling
	var unmarshaled ResourceManifest
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	if err != nil {
		t.Fatalf("YAML unmarshal failed: %v", err)
	}

	// Verify basic fields
	if unmarshaled.APIVersion != manifest.APIVersion {
		t.Errorf("APIVersion = %s, want %s", unmarshaled.APIVersion, manifest.APIVersion)
	}
	if unmarshaled.Kind != manifest.Kind {
		t.Errorf("Kind = %s, want %s", unmarshaled.Kind, manifest.Kind)
	}

	// Verify status
	if unmarshaled.Status == nil {
		t.Fatal("Status is nil")
	}
	if unmarshaled.Status.Phase != StatusReady {
		t.Errorf("Status.Phase = %s, want %s", unmarshaled.Status.Phase, StatusReady)
	}
	if len(unmarshaled.Status.Conditions) != 1 {
		t.Errorf("Status.Conditions length = %d, want 1", len(unmarshaled.Status.Conditions))
	}
}

func TestClusterManifest_YAMLMarshaling(t *testing.T) {
	cluster := ClusterManifest{
		APIVersion: APIVersionV1,
		Kind:       KindCluster,
		Metadata: ResourceMetadata{
			Name: "production-cluster",
			Labels: map[string]string{
				"environment": "production",
				"team":        "platform",
			},
			Annotations: map[string]string{
				"managed-by": "matlas-cli",
			},
		},
		Spec: ClusterSpec{
			ProjectName:    "prod-project",
			Provider:       "AWS",
			Region:         "US_EAST_1",
			InstanceSize:   "M40",
			MongoDBVersion: "7.0",
			ClusterType:    "REPLICASET",
			AutoScaling: &AutoScalingConfig{
				DiskGB: &AutoScalingLimits{
					Enabled:   boolPtr(true),
					MinimumGB: intPtr(20),
					MaximumGB: intPtr(200),
				},
			},
		},
	}

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(cluster)
	if err != nil {
		t.Fatalf("YAML marshal failed: %v", err)
	}

	// Test YAML unmarshaling
	var unmarshaled ClusterManifest
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	if err != nil {
		t.Fatalf("YAML unmarshal failed: %v", err)
	}

	// Verify fields
	if unmarshaled.Spec.ProjectName != cluster.Spec.ProjectName {
		t.Errorf("Spec.ProjectName = %s, want %s", unmarshaled.Spec.ProjectName, cluster.Spec.ProjectName)
	}
	if unmarshaled.Spec.MongoDBVersion != cluster.Spec.MongoDBVersion {
		t.Errorf("Spec.MongoDBVersion = %s, want %s", unmarshaled.Spec.MongoDBVersion, cluster.Spec.MongoDBVersion)
	}
	if unmarshaled.Spec.AutoScaling == nil {
		t.Fatal("Spec.AutoScaling is nil")
	}
	if *unmarshaled.Spec.AutoScaling.DiskGB.MaximumGB != 200 {
		t.Errorf("Spec.AutoScaling.DiskGB.MaximumGB = %d, want 200", *unmarshaled.Spec.AutoScaling.DiskGB.MaximumGB)
	}
}

func TestDatabaseUserManifest_YAMLMarshaling(t *testing.T) {
	user := DatabaseUserManifest{
		APIVersion: APIVersionV1,
		Kind:       KindDatabaseUser,
		Metadata: ResourceMetadata{
			Name: "analytics-user",
		},
		Spec: DatabaseUserSpec{
			ProjectName:  "analytics-project",
			Username:     "analytics-service",
			AuthDatabase: "admin",
			Roles: []DatabaseRoleConfig{
				{
					RoleName:     "read",
					DatabaseName: "analytics",
				},
				{
					RoleName:       "readWrite",
					DatabaseName:   "logs",
					CollectionName: "app-logs",
				},
			},
			Scopes: []UserScopeConfig{
				{
					Name: "analytics-cluster",
					Type: "CLUSTER",
				},
			},
		},
	}

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(user)
	if err != nil {
		t.Fatalf("YAML marshal failed: %v", err)
	}

	// Test YAML unmarshaling
	var unmarshaled DatabaseUserManifest
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	if err != nil {
		t.Fatalf("YAML unmarshal failed: %v", err)
	}

	// Verify fields
	if unmarshaled.Spec.Username != user.Spec.Username {
		t.Errorf("Spec.Username = %s, want %s", unmarshaled.Spec.Username, user.Spec.Username)
	}
	if len(unmarshaled.Spec.Roles) != 2 {
		t.Errorf("Spec.Roles length = %d, want 2", len(unmarshaled.Spec.Roles))
	}
	if unmarshaled.Spec.Roles[1].CollectionName != "app-logs" {
		t.Errorf("Spec.Roles[1].CollectionName = %s, want app-logs", unmarshaled.Spec.Roles[1].CollectionName)
	}
}

func TestDatabaseDirectUserManifest_YAMLMarshaling(t *testing.T) {
	user := DatabaseDirectUserManifest{
		APIVersion: APIVersionV1,
		Kind:       KindDatabaseDirectUser,
		Metadata: ResourceMetadata{
			Name: "app-direct-user",
		},
		Spec: DatabaseDirectUserSpec{
			ConnectionConfig: ConnectionConfigSpec{
				Cluster:      "app-cluster",
				ProjectID:    "project123",
				UseTempUser:  true,
				TempUserRole: "userAdminAnyDatabase@admin",
			},
			Username: "app-database-user",
			Password: "SecurePassword123!",
			Database: "application",
			Roles: []DatabaseRoleConfig{
				{
					RoleName:     "readWrite",
					DatabaseName: "application",
				},
				{
					RoleName:       "read",
					DatabaseName:   "logs",
					CollectionName: "app-events",
				},
			},
		},
	}

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(user)
	if err != nil {
		t.Fatalf("YAML marshal failed: %v", err)
	}

	// Test YAML unmarshaling
	var unmarshaled DatabaseDirectUserManifest
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	if err != nil {
		t.Fatalf("YAML unmarshal failed: %v", err)
	}

	// Verify fields
	if unmarshaled.Spec.Username != user.Spec.Username {
		t.Errorf("Spec.Username = %s, want %s", unmarshaled.Spec.Username, user.Spec.Username)
	}
	if unmarshaled.Spec.Database != user.Spec.Database {
		t.Errorf("Spec.Database = %s, want %s", unmarshaled.Spec.Database, user.Spec.Database)
	}
	if unmarshaled.Spec.ConnectionConfig.Cluster != user.Spec.ConnectionConfig.Cluster {
		t.Errorf("Spec.ConnectionConfig.Cluster = %s, want %s", unmarshaled.Spec.ConnectionConfig.Cluster, user.Spec.ConnectionConfig.Cluster)
	}
	if unmarshaled.Spec.ConnectionConfig.ProjectID != user.Spec.ConnectionConfig.ProjectID {
		t.Errorf("Spec.ConnectionConfig.ProjectID = %s, want %s", unmarshaled.Spec.ConnectionConfig.ProjectID, user.Spec.ConnectionConfig.ProjectID)
	}
	if unmarshaled.Spec.ConnectionConfig.UseTempUser != user.Spec.ConnectionConfig.UseTempUser {
		t.Errorf("Spec.ConnectionConfig.UseTempUser = %v, want %v", unmarshaled.Spec.ConnectionConfig.UseTempUser, user.Spec.ConnectionConfig.UseTempUser)
	}
	if len(unmarshaled.Spec.Roles) != 2 {
		t.Errorf("Spec.Roles length = %d, want 2", len(unmarshaled.Spec.Roles))
	}
	if unmarshaled.Spec.Roles[1].CollectionName != "app-events" {
		t.Errorf("Spec.Roles[1].CollectionName = %s, want app-events", unmarshaled.Spec.Roles[1].CollectionName)
	}
}

func TestNetworkAccessManifest_YAMLMarshaling(t *testing.T) {
	networkAccess := NetworkAccessManifest{
		APIVersion: APIVersionV1,
		Kind:       KindNetworkAccess,
		Metadata: ResourceMetadata{
			Name: "office-network",
		},
		Spec: NetworkAccessSpec{
			ProjectName:     "office-project",
			CIDR:            "10.0.0.0/8",
			Comment:         "Office network access for development",
			DeleteAfterDate: "2024-12-31T23:59:59Z",
		},
	}

	// Test YAML marshaling
	yamlData, err := yaml.Marshal(networkAccess)
	if err != nil {
		t.Fatalf("YAML marshal failed: %v", err)
	}

	// Test YAML unmarshaling
	var unmarshaled NetworkAccessManifest
	err = yaml.Unmarshal(yamlData, &unmarshaled)
	if err != nil {
		t.Fatalf("YAML unmarshal failed: %v", err)
	}

	// Verify fields
	if unmarshaled.Spec.CIDR != networkAccess.Spec.CIDR {
		t.Errorf("Spec.CIDR = %s, want %s", unmarshaled.Spec.CIDR, networkAccess.Spec.CIDR)
	}
	if unmarshaled.Spec.DeleteAfterDate != networkAccess.Spec.DeleteAfterDate {
		t.Errorf("Spec.DeleteAfterDate = %s, want %s", unmarshaled.Spec.DeleteAfterDate, networkAccess.Spec.DeleteAfterDate)
	}
}

func TestDependencyGraph_BasicOperations(t *testing.T) {
	dg := NewDependencyGraph()

	// Add resources
	project := &ResourceNode{
		Name: "my-project",
		Kind: KindProject,
	}
	cluster := &ResourceNode{
		Name:         "my-cluster",
		Kind:         KindCluster,
		Dependencies: []string{"my-project"},
	}
	user := &ResourceNode{
		Name:         "my-user",
		Kind:         KindDatabaseUser,
		Dependencies: []string{"my-cluster"},
	}

	dg.AddResource(project)
	dg.AddResource(cluster)
	dg.AddResource(user)

	// Test resource retrieval
	if len(dg.Resources) != 3 {
		t.Errorf("Resources count = %d, want 3", len(dg.Resources))
	}

	// Test dependency retrieval
	clusterDeps := dg.GetDependencies("", "my-cluster")
	if len(clusterDeps) != 1 || clusterDeps[0] != "my-project" {
		t.Errorf("my-cluster dependencies = %v, want [my-project]", clusterDeps)
	}

	userDeps := dg.GetDependencies("", "my-user")
	if len(userDeps) != 1 || userDeps[0] != "my-cluster" {
		t.Errorf("my-user dependencies = %v, want [my-cluster]", userDeps)
	}
}

func TestDependencyGraph_CycleDetection(t *testing.T) {
	dg := NewDependencyGraph()

	// Create a circular dependency: A -> B -> C -> A
	nodeA := &ResourceNode{
		Name:         "resource-a",
		Kind:         KindCluster,
		Dependencies: []string{"resource-c"},
	}
	nodeB := &ResourceNode{
		Name:         "resource-b",
		Kind:         KindDatabaseUser,
		Dependencies: []string{"resource-a"},
	}
	nodeC := &ResourceNode{
		Name:         "resource-c",
		Kind:         KindNetworkAccess,
		Dependencies: []string{"resource-b"},
	}

	dg.AddResource(nodeA)
	dg.AddResource(nodeB)
	dg.AddResource(nodeC)

	// Test cycle detection
	hasCycle, cycle := dg.HasCycles()
	if !hasCycle {
		t.Error("Expected cycle to be detected")
	}
	if len(cycle) == 0 {
		t.Error("Expected cycle information to be returned")
	}

	// Test topological sort fails with cycle
	_, err := dg.TopologicalSort()
	if err == nil {
		t.Error("Expected topological sort to fail with cycle")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("Expected 'circular dependency' in error, got: %v", err)
	}
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	dg := NewDependencyGraph()

	// Create a valid dependency chain: project -> cluster -> user
	project := &ResourceNode{
		Name: "project",
		Kind: KindProject,
	}
	cluster := &ResourceNode{
		Name:         "cluster",
		Kind:         KindCluster,
		Dependencies: []string{"project"},
	}
	user := &ResourceNode{
		Name:         "user",
		Kind:         KindDatabaseUser,
		Dependencies: []string{"cluster"},
	}
	network := &ResourceNode{
		Name:         "network",
		Kind:         KindNetworkAccess,
		Dependencies: []string{"project"}, // Also depends on project, but independent of cluster/user
	}

	dg.AddResource(project)
	dg.AddResource(cluster)
	dg.AddResource(user)
	dg.AddResource(network)

	// Test no cycles
	hasCycle, _ := dg.HasCycles()
	if hasCycle {
		t.Error("No cycle should be detected")
	}

	// Test topological sort
	sorted, err := dg.TopologicalSort()
	if err != nil {
		t.Fatalf("Topological sort failed: %v", err)
	}

	if len(sorted) != 4 {
		t.Errorf("Sorted length = %d, want 4", len(sorted))
	}

	// Verify dependencies are respected
	projectIndex := indexOf(sorted, "project")
	clusterIndex := indexOf(sorted, "cluster")
	userIndex := indexOf(sorted, "user")
	networkIndex := indexOf(sorted, "network")

	if projectIndex == -1 || clusterIndex == -1 || userIndex == -1 || networkIndex == -1 {
		t.Errorf("Missing resources in sorted order: %v", sorted)
	}

	// project should come before cluster and network
	if projectIndex > clusterIndex {
		t.Errorf("project should come before cluster in sorted order")
	}
	if projectIndex > networkIndex {
		t.Errorf("project should come before network in sorted order")
	}

	// cluster should come before user
	if clusterIndex > userIndex {
		t.Errorf("cluster should come before user in sorted order")
	}
}

func TestDependencyGraph_WithNamespaces(t *testing.T) {
	dg := NewDependencyGraph()

	// Add resources with namespaces
	project := &ResourceNode{
		Name:      "project",
		Kind:      KindProject,
		Namespace: "production",
	}
	cluster := &ResourceNode{
		Name:         "cluster",
		Kind:         KindCluster,
		Namespace:    "production",
		Dependencies: []string{"production/project"},
	}

	dg.AddResource(project)
	dg.AddResource(cluster)

	// Test namespace handling
	deps := dg.GetDependencies("production", "cluster")
	if len(deps) != 1 || deps[0] != "production/project" {
		t.Errorf("cluster dependencies = %v, want [production/project]", deps)
	}

	// Test resources are stored with namespace keys
	if _, exists := dg.Resources["production/project"]; !exists {
		t.Error("project should be stored with namespace key")
	}
	if _, exists := dg.Resources["production/cluster"]; !exists {
		t.Error("cluster should be stored with namespace key")
	}
}

func TestValidateAPIVersion(t *testing.T) {
	tests := []struct {
		name    string
		version APIVersion
		wantErr bool
	}{
		{"Valid v1alpha1", APIVersionV1Alpha1, false},
		{"Valid v1beta1", APIVersionV1Beta1, false},
		{"Valid v1", APIVersionV1, false},
		{"Invalid version", APIVersion("invalid/v1"), true},
		{"Empty version", APIVersion(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAPIVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateResourceKind(t *testing.T) {
	tests := []struct {
		name    string
		kind    ResourceKind
		wantErr bool
	}{
		{"Valid Project", KindProject, false},
		{"Valid Cluster", KindCluster, false},
		{"Valid DatabaseUser", KindDatabaseUser, false},
		{"Valid DatabaseDirectUser", KindDatabaseDirectUser, false},
		{"Valid DatabaseRole", KindDatabaseRole, false},
		{"Valid NetworkAccess", KindNetworkAccess, false},
		{"Valid ApplyDocument", KindApplyDocument, false},
		{"Invalid kind", ResourceKind("InvalidKind"), true},
		{"Empty kind", ResourceKind(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceKind(tt.kind)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResourceKind() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResourceKey(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		resName   string
		expected  string
	}{
		{"No namespace", "", "resource", "resource"},
		{"With namespace", "prod", "resource", "prod/resource"},
		{"Empty resource name", "prod", "", "prod/"},
		{"Both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resourceKey(tt.namespace, tt.resName)
			if result != tt.expected {
				t.Errorf("resourceKey(%s, %s) = %s, want %s", tt.namespace, tt.resName, result, tt.expected)
			}
		})
	}
}

func TestResourceStatusInfo_JSONMarshaling(t *testing.T) {
	status := ResourceStatusInfo{
		Phase:      StatusReady,
		Message:    "Resource is healthy",
		Reason:     "AllChecksPass",
		LastUpdate: "2024-01-15T10:30:00Z",
		Conditions: []StatusCondition{
			{
				Type:               "Ready",
				Status:             "True",
				LastTransitionTime: "2024-01-15T10:25:00Z",
				Reason:             "ResourceCreated",
				Message:            "Resource created successfully",
			},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled ResourceStatusInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Verify fields
	if unmarshaled.Phase != status.Phase {
		t.Errorf("Phase = %s, want %s", unmarshaled.Phase, status.Phase)
	}
	if len(unmarshaled.Conditions) != 1 {
		t.Errorf("Conditions length = %d, want 1", len(unmarshaled.Conditions))
	}
	if unmarshaled.Conditions[0].Type != "Ready" {
		t.Errorf("Conditions[0].Type = %s, want Ready", unmarshaled.Conditions[0].Type)
	}
}

// Helper functions for tests

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func boolPtr(b bool) *bool {
	return &b
}
