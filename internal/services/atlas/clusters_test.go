package atlas

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/subosito/gotenv"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
)

func TestClustersService_List(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping clusters tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewClustersService(client)

	clusters, err := svc.List(ctx, pid)
	if err != nil {
		t.Fatalf("List err: %v", err)
	}
	t.Logf("fetched %d clusters", len(clusters))

	if len(clusters) == 0 {
		t.Skip("no clusters present; skipping Get test")
	}

	cname := clusters[0].GetName()
	cl, err := svc.Get(ctx, pid, cname)
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if cl == nil || cl.GetName() != cname {
		t.Fatalf("expected cluster %s, got %+v", cname, cl)
	}
}

// Error scenario tests

func TestClustersService_List_InvalidProjectID(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	// Skip if no Atlas credentials
	if os.Getenv("ATLAS_PUB_KEY") == "" || os.Getenv("ATLAS_API_KEY") == "" {
		t.Skip("Atlas credentials not set; skipping error scenario tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewClustersService(client)

	// Test with invalid project ID
	clusters, err := svc.List(ctx, "invalid-project-id-12345")
	if err == nil {
		t.Error("Expected error for invalid project ID, got nil")
	}
	if clusters != nil {
		t.Error("Expected nil clusters for invalid project ID")
	}
	t.Logf("Error (expected): %v", err)
}

func TestClustersService_List_EmptyProjectID(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	if os.Getenv("ATLAS_PUB_KEY") == "" || os.Getenv("ATLAS_API_KEY") == "" {
		t.Skip("Atlas credentials not set; skipping error scenario tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewClustersService(client)

	// Test with empty project ID
	clusters, err := svc.List(ctx, "")
	if err == nil {
		t.Error("Expected error for empty project ID, got nil")
	}
	if clusters != nil {
		t.Error("Expected nil clusters for empty project ID")
	}
}

func TestClustersService_Get_NonExistentCluster(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping error scenario tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewClustersService(client)

	// Test with non-existent cluster name
	cluster, err := svc.Get(ctx, pid, "non-existent-cluster-12345")
	if err == nil {
		t.Error("Expected error for non-existent cluster, got nil")
	}
	if cluster != nil {
		t.Error("Expected nil cluster for non-existent cluster name")
	}
	t.Logf("Error (expected): %v", err)
}

func TestClustersService_Get_InvalidClusterName(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping error scenario tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewClustersService(client)

	invalidNames := []string{
		"",                                 // empty name
		"invalid cluster name with spaces", // spaces not allowed
		"cluster-with-special-chars-!@#$%", // special characters
		"a",                                // too short
	}

	for _, name := range invalidNames {
		t.Run("InvalidName_"+name, func(t *testing.T) {
			cluster, err := svc.Get(ctx, pid, name)
			if err == nil {
				t.Errorf("Expected error for invalid cluster name '%s', got nil", name)
			}
			if cluster != nil {
				t.Errorf("Expected nil cluster for invalid cluster name '%s'", name)
			}
		})
	}
}

func TestClustersService_ContextCancellation(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping context cancellation tests")
	}

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewClustersService(client)

	// Create a context that's immediately cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	clusters, err := svc.List(ctx, pid)
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
	if clusters != nil {
		t.Error("Expected nil clusters for cancelled context")
	}
	t.Logf("Context cancellation error (expected): %v", err)
}

func TestClustersService_ShortTimeout(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping timeout tests")
	}

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewClustersService(client)

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	clusters, err := svc.List(ctx, pid)
	// Note: This might succeed if the call is very fast, but typically should timeout
	if err != nil {
		t.Logf("Timeout error (may be expected): %v", err)
	} else {
		t.Logf("Call succeeded despite short timeout (call was very fast), got %d clusters", len(clusters))
	}
}
