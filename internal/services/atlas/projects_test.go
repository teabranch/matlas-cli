package atlas

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/subosito/gotenv"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
)

func TestProjectsService_ListAndGet(t *testing.T) {
	// Load env vars from repository root or current dir.
	err := gotenv.Load("../../../.env", "../../.env", "../.env", ".env")
	if err != nil {
		t.Logf("gotenv.Load error: %v", err)
	}

	// Debug: Check if environment variables are loaded
	apiKey := os.Getenv("ATLAS_API_KEY")
	pubKey := os.Getenv("ATLAS_PUB_KEY")
	orgID := os.Getenv("ORG_ID")
	projectID := os.Getenv("PROJECT_ID")
	t.Logf("Environment variables loaded - API_KEY: %s, PUB_KEY: %s, ORG_ID: %s, PROJECT_ID: %s",
		maskKey(apiKey), maskKey(pubKey), orgID, projectID)

	if apiKey == "" || pubKey == "" {
		t.Skip("ATLAS_API_KEY or ATLAS_PUB_KEY not set, skipping Atlas API tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	svc := NewProjectsService(client)

	projects, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(projects) == 0 {
		t.Skip("no projects returned; credentials may lack permissions")
	}
	t.Logf("fetched %d projects", len(projects))

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping Get test")
	}
	p, err := svc.Get(ctx, pid)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if p == nil || p.GetId() != pid {
		t.Fatalf("expected project %s, got %+v", pid, p)
	}
}

// Unit tests for validation and structure (no API calls)
func TestNewProjectsService(t *testing.T) {
	client := &atlasclient.Client{}
	service := NewProjectsService(client)

	if service == nil {
		t.Fatal("NewProjectsService returned nil")
	}
	if service.client != client {
		t.Fatal("NewProjectsService did not set client correctly")
	}
}

func TestProjectsService_ListByOrg_Validation(t *testing.T) {
	service := NewProjectsService(&atlasclient.Client{})
	ctx := context.Background()

	// Test empty orgID
	projects, err := service.ListByOrg(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty orgID")
	}
	if projects != nil {
		t.Fatal("expected nil projects for empty orgID")
	}
	if err.Error() != "orgID is required" {
		t.Fatalf("expected 'orgID is required', got: %s", err.Error())
	}
}

func TestProjectsService_Get_Validation(t *testing.T) {
	service := NewProjectsService(&atlasclient.Client{})
	ctx := context.Background()

	// Test empty projectID
	project, err := service.Get(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty projectID")
	}
	if project != nil {
		t.Fatal("expected nil project for empty projectID")
	}
	if err.Error() != "projectID is required" {
		t.Fatalf("expected 'projectID is required', got: %s", err.Error())
	}
}

func TestProjectsService_Create_Validation(t *testing.T) {
	service := NewProjectsService(&atlasclient.Client{})
	ctx := context.Background()

	tests := []struct {
		name     string
		projName string
		orgID    string
		wantErr  string
	}{
		{
			name:     "empty name",
			projName: "",
			orgID:    "org123",
			wantErr:  "name and orgID are required",
		},
		{
			name:     "empty orgID",
			projName: "test-project",
			orgID:    "",
			wantErr:  "name and orgID are required",
		},
		{
			name:     "both empty",
			projName: "",
			orgID:    "",
			wantErr:  "name and orgID are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := service.Create(ctx, tt.projName, tt.orgID, nil)
			if err == nil {
				t.Fatal("expected error for invalid parameters")
			}
			if project != nil {
				t.Fatal("expected nil project for invalid parameters")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected '%s', got: %s", tt.wantErr, err.Error())
			}
		})
	}
}

func TestProjectsService_Delete_Validation(t *testing.T) {
	service := NewProjectsService(&atlasclient.Client{})
	ctx := context.Background()

	// Test empty projectID
	err := service.Delete(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty projectID")
	}
	if err.Error() != "projectID is required" {
		t.Fatalf("expected 'projectID is required', got: %s", err.Error())
	}
}
