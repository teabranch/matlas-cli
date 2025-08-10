package infra

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

// helper to create a temp YAML file
func writeTempYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil { //nolint:gosec // test file
		t.Fatalf("failed writing temp yaml: %v", err)
	}
	return path
}

// Test cross-file dependency validation: a user references a cluster that exists in a different file.
func TestValidateBatch_CrossFileDependencies(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// File 1: ApplyDocument with Project and Cluster (so cross-file aggregate sees the cluster)
	clusterYAML := `
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: doc1
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: Project
    metadata:
      name: my-proj
    spec:
      name: my-proj
      organizationId: 507f1f77bcf86cd799439011
  - apiVersion: matlas.mongodb.com/v1
    kind: Cluster
    metadata:
      name: c1
    spec:
      projectName: my-proj
      provider: AWS
      region: us-west-2
      instanceSize: M10
`
	file1 := writeTempYAML(t, tmpDir, "cluster.yaml", clusterYAML)

	// File 2: User referencing cluster c1 (should be valid when both files provided)
	userYAML := `
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: doc2
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: u1
    spec:
      projectName: my-proj
      username: alice
      password: secretsecret
      roles:
        - roleName: readWrite
          databaseName: admin
      scopes:
        - name: c1
          type: CLUSTER
`
	file2 := writeTempYAML(t, tmpDir, "user.yaml", userYAML)

	loader := apply.NewConfigurationLoader(apply.DefaultLoaderOptions())
	opts := &ValidateOptions{Verbose: false, BatchMode: true}
	vopts := apply.DefaultValidatorOptions()

	results, err := validateBatch(context.TODO(), []string{file1, file2}, loader, vopts, opts)
	if err != nil {
		t.Fatalf("validateBatch error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Expect no dependency errors because the cluster exists across files
	for _, r := range results {
		for _, dep := range r.DependencyResults {
			if dep.Severity == "error" {
				t.Fatalf("unexpected dependency error: %+v", dep)
			}
		}
	}
}

// Test cross-file dependency validation detects missing cluster when only user file provided
func TestValidateBatch_MissingCrossFileDependency(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	userOnlyYAML := `
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata:
  name: doc2
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata:
      name: u1
    spec:
      projectName: my-proj
      username: alice
      password: secretsecret
      roles:
        - roleName: readWrite
          databaseName: admin
      scopes:
        - name: missing-cluster
          type: CLUSTER
`
	file := writeTempYAML(t, tmpDir, "user-only.yaml", userOnlyYAML)

	loader := apply.NewConfigurationLoader(apply.DefaultLoaderOptions())
	opts := &ValidateOptions{Verbose: true, BatchMode: true}
	vopts := apply.DefaultValidatorOptions()
	vopts.StrictMode = true

	results, err := validateBatch(context.TODO(), []string{file}, loader, vopts, opts)
	if err != nil {
		t.Fatalf("validateBatch error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	// We expect at least one dependency error about missing cluster
	found := false
	for _, dep := range r.DependencyResults {
		if dep.Severity == "error" && dep.Field == "missing_resource" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing_resource dependency error, got: %+v", r.DependencyResults)
	}
}

// New unit test to validate project tags are carried into desired state during build
func TestBuildDesiredState_IncludesProjectTags(t *testing.T) {
	cfg := &types.ApplyConfig{
		APIVersion: "matlas.mongodb.com/v1",
		Kind:       "Project",
		Metadata:   types.MetadataConfig{Name: "proj"},
		Spec: types.ProjectConfig{
			Name:           "proj",
			OrganizationID: "507f1f77bcf86cd799439011",
			Tags:           map[string]string{"env": "staging"},
		},
	}

	lr := &apply.LoadResult{Config: cfg}
	state, err := buildDesiredState([]*apply.LoadResult{lr})
	if err != nil {
		t.Fatalf("buildDesiredState error: %v", err)
	}
	if state.Project == nil {
		t.Fatalf("expected state.Project to be populated")
	}
	if got := state.Project.Spec.Tags["env"]; got != "staging" {
		t.Fatalf("expected project tag env=staging, got %q", got)
	}
}
