package atlas

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/subosito/gotenv"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
)

func TestOrganizationsService_ListAndGet(t *testing.T) {
	// Load env vars for credentials.
	err := gotenv.Load("../../../.env", "../../.env", "../.env", ".env")
	if err != nil {
		t.Logf("gotenv.Load error: %v", err)
	}

	// Debug: Check if environment variables are loaded
	apiKey := os.Getenv("ATLAS_API_KEY")
	pubKey := os.Getenv("ATLAS_PUB_KEY")
	orgID := os.Getenv("ORG_ID")
	t.Logf("Environment variables loaded - API_KEY: %s, PUB_KEY: %s, ORG_ID: %s",
		maskKey(apiKey), maskKey(pubKey), orgID)

	if apiKey == "" || pubKey == "" {
		t.Skip("ATLAS_API_KEY or ATLAS_PUB_KEY not set, skipping Atlas API tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("new client err: %v", err)
	}
	orgSvc := NewOrganizationsService(client)

	orgs, err := orgSvc.List(ctx)
	if err != nil {
		t.Fatalf("List returned err: %v", err)
	}
	if len(orgs) == 0 {
		t.Skip("no orgs returned; creds might be limited")
	}
	t.Logf("fetched %d organizations", len(orgs))

	oid := os.Getenv("ORG_ID")
	if oid == "" {
		t.Skip("ORG_ID env not set; skipping Get test")
	}
	org, err := orgSvc.Get(ctx, oid)
	if err != nil {
		t.Fatalf("Get returned err: %v", err)
	}
	if org == nil || org.GetId() != oid {
		t.Fatalf("expected org %s, got %+v", oid, org)
	}
}

// maskKey masks sensitive keys for logging
func maskKey(key string) string {
	if len(key) == 0 {
		return "NOT_SET"
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}
