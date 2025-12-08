package atlas

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/subosito/gotenv"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312010/admin"
)

func TestNetworkAccessListsService_CRUD(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping network access list tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewNetworkAccessListsService(client)

	// List existing entries
	entries, err := svc.List(ctx, pid)
	if err != nil {
		t.Fatalf("List err: %v", err)
	}
	t.Logf("found %d existing network access entries", len(entries))

	// Create a test IP entry (using a safe test IP)
	testIP := "203.0.113.1" // TEST-NET-3 RFC 5737 - safe for testing
	testEntry := admin.NetworkPermissionEntry{
		IpAddress: admin.PtrString(testIP),
		Comment:   admin.PtrString("Test entry created by unit tests"),
	}

	// Ensure cleanup happens
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := svc.Delete(ctx, pid, testIP)
		if err != nil {
			t.Logf("cleanup warning: failed to delete test IP %s: %v", testIP, err)
		} else {
			t.Logf("cleanup: deleted test IP %s", testIP)
		}
	}()

	created, err := svc.Create(ctx, pid, []admin.NetworkPermissionEntry{testEntry})
	if err != nil {
		t.Fatalf("Create err: %v", err)
	}
	if created == nil || created.Results == nil || len(*created.Results) == 0 {
		t.Fatalf("create returned empty results")
	}
	t.Logf("created network access entry for IP: %s", testIP)

	// Get the created entry
	retrieved, err := svc.Get(ctx, pid, testIP)
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if retrieved == nil || retrieved.IpAddress == nil || *retrieved.IpAddress != testIP {
		t.Fatalf("retrieved entry mismatch: expected %s, got %v", testIP, retrieved)
	}
	t.Logf("retrieved entry: %s", *retrieved.IpAddress)

	// Delete will be handled by defer cleanup
}

func TestNetworkAccessListsService_ValidationHelpers(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		fn      func(string) error
	}{
		{"valid IP", "192.168.1.1", false, ValidateIP},
		{"invalid IP", "999.999.999.999", true, ValidateIP},
		{"valid CIDR", "192.168.1.0/24", false, ValidateCIDR},
		{"invalid CIDR", "192.168.1.0/99", true, ValidateCIDR},
		{"IP as CIDR should fail", "192.168.1.1", true, ValidateCIDR},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Error scenario tests for network access

func TestNetworkAccessListsService_Create_DuplicateEntry(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping network access error tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewNetworkAccessListsService(client)

	// Use a unique test IP (TEST-NET-3 range is safe for testing)
	testIP := "203.0.113.100"
	testEntry := admin.NetworkPermissionEntry{
		IpAddress: admin.PtrString(testIP),
		Comment:   admin.PtrString("Duplicate entry test"),
	}

	// Cleanup function
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = svc.Delete(ctx, pid, testIP)
	}()

	// Create the entry first time - should succeed
	_, err = svc.Create(ctx, pid, []admin.NetworkPermissionEntry{testEntry})
	if err != nil {
		t.Fatalf("First create should succeed: %v", err)
	}

	// Try to create the same entry again - should fail
	_, err = svc.Create(ctx, pid, []admin.NetworkPermissionEntry{testEntry})
	if err == nil {
		t.Error("Expected error when creating duplicate network access entry, got nil")
	}
	t.Logf("Duplicate entry error (expected): %v", err)
}

func TestNetworkAccessListsService_Create_InvalidInputs(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping network access error tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewNetworkAccessListsService(client)

	type testCase struct {
		name  string
		entry admin.NetworkPermissionEntry
		key   string
	}

	cases := []testCase{
		{name: "invalid ip - range", entry: admin.NetworkPermissionEntry{IpAddress: admin.PtrString("999.999.999.999"), Comment: admin.PtrString("Invalid IP test")}, key: "999.999.999.999"},
		{name: "invalid ip - incomplete", entry: admin.NetworkPermissionEntry{IpAddress: admin.PtrString("192.168.1"), Comment: admin.PtrString("Invalid IP test")}, key: "192.168.1"},
		{name: "invalid ip - too many octets", entry: admin.NetworkPermissionEntry{IpAddress: admin.PtrString("192.168.1.1.1"), Comment: admin.PtrString("Invalid IP test")}, key: "192.168.1.1.1"},
		{name: "invalid ip - non numeric", entry: admin.NetworkPermissionEntry{IpAddress: admin.PtrString("not.an.ip.address"), Comment: admin.PtrString("Invalid IP test")}, key: "not.an.ip.address"},
		{name: "invalid ip - empty", entry: admin.NetworkPermissionEntry{IpAddress: admin.PtrString(""), Comment: admin.PtrString("Invalid IP test")}, key: ""},
		{name: "invalid cidr - prefix length", entry: admin.NetworkPermissionEntry{CidrBlock: admin.PtrString("192.168.1.0/99"), Comment: admin.PtrString("Invalid CIDR test")}, key: "192.168.1.0/99"},
		{name: "invalid cidr - negative prefix", entry: admin.NetworkPermissionEntry{CidrBlock: admin.PtrString("192.168.1.0/-1"), Comment: admin.PtrString("Invalid CIDR test")}, key: "192.168.1.0/-1"},
		{name: "invalid cidr - missing prefix", entry: admin.NetworkPermissionEntry{CidrBlock: admin.PtrString("192.168.1.0/"), Comment: admin.PtrString("Invalid CIDR test")}, key: "192.168.1.0/"},
		{name: "invalid cidr - non-numeric prefix", entry: admin.NetworkPermissionEntry{CidrBlock: admin.PtrString("192.168.1.0/abc"), Comment: admin.PtrString("Invalid CIDR test")}, key: "192.168.1.0/abc"},
		{name: "invalid cidr - invalid ip", entry: admin.NetworkPermissionEntry{CidrBlock: admin.PtrString("999.999.999.0/24"), Comment: admin.PtrString("Invalid CIDR test")}, key: "999.999.999.0/24"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(ctx, pid, []admin.NetworkPermissionEntry{tc.entry})
			if err == nil {
				t.Errorf("Expected error for invalid input '%s', got nil", tc.key)
				// Clean up if it somehow succeeded
				_ = svc.Delete(ctx, pid, tc.key)
			}
		})
	}
}

func TestNetworkAccessListsService_Get_NonExistentEntry(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping network access error tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewNetworkAccessListsService(client)

	// Try to get a non-existent entry
	entry, err := svc.Get(ctx, pid, "203.0.113.200")
	if err == nil {
		t.Error("Expected error for non-existent network access entry, got nil")
	}
	if entry != nil {
		t.Error("Expected nil entry for non-existent network access entry")
	}
	t.Logf("Non-existent entry error (expected): %v", err)
}

func TestNetworkAccessListsService_Delete_NonExistentEntry(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping network access error tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewNetworkAccessListsService(client)

	// Try to delete a non-existent entry
	err = svc.Delete(ctx, pid, "203.0.113.201")
	if err == nil {
		t.Error("Expected error for deleting non-existent network access entry, got nil")
	}
	t.Logf("Delete non-existent entry error (expected): %v", err)
}

func TestNetworkAccessListsService_InvalidProjectID(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	// Skip if no Atlas credentials
	if os.Getenv("ATLAS_PUB_KEY") == "" || os.Getenv("ATLAS_API_KEY") == "" {
		t.Skip("Atlas credentials not set; skipping error scenario tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewNetworkAccessListsService(client)

	// Test with invalid project ID
	entries, err := svc.List(ctx, "invalid-project-id-12345")
	if err == nil {
		t.Error("Expected error for invalid project ID, got nil")
	}
	if entries != nil {
		t.Error("Expected nil entries for invalid project ID")
	}
	t.Logf("Invalid project ID error (expected): %v", err)
}
