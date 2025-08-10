package atlas

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/subosito/gotenv"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

func TestDatabaseUsersService_CRUD(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping database users tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewDatabaseUsersService(client)

	// List existing users
	users, err := svc.List(ctx, pid)
	if err != nil {
		t.Fatalf("List err: %v", err)
	}
	t.Logf("found %d existing database users", len(users))

	// Create a test user
	testUsername := fmt.Sprintf("test-user-%d", time.Now().Unix())
	testUser := &admin.CloudDatabaseUser{
		DatabaseName: "admin",
		Username:     testUsername,
		Password:     admin.PtrString("TestPassword123!"),
		Roles: &[]admin.DatabaseUserRole{
			{
				RoleName:     "read",
				DatabaseName: "admin",
			},
		},
	}

	// Ensure cleanup happens
	defer func() {
		if testUser != nil && testUser.Username != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := svc.Delete(ctx, pid, "admin", testUser.Username)
			if err != nil {
				t.Logf("cleanup warning: failed to delete test user %s: %v", testUser.Username, err)
			} else {
				t.Logf("cleanup: deleted test user %s", testUser.Username)
			}
		}
	}()

	created, err := svc.Create(ctx, pid, testUser)
	if err != nil {
		t.Fatalf("Create err: %v", err)
	}
	if created == nil || created.GetUsername() != testUsername {
		t.Fatalf("created user mismatch: expected %s, got %v", testUsername, created)
	}
	t.Logf("created user: %s", created.GetUsername())

	// Get the created user
	retrieved, err := svc.Get(ctx, pid, "admin", testUsername)
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if retrieved == nil || retrieved.GetUsername() != testUsername {
		t.Fatalf("retrieved user mismatch: expected %s, got %v", testUsername, retrieved)
	}

	// Update the user (change password)
	updateUser := &admin.CloudDatabaseUser{
		DatabaseName: "admin",
		Username:     testUsername,
		Password:     admin.PtrString("NewPassword456!"),
		Roles: &[]admin.DatabaseUserRole{
			{
				RoleName:     "readWrite",
				DatabaseName: "admin",
			},
		},
	}

	updated, err := svc.Update(ctx, pid, "admin", testUsername, updateUser)
	if err != nil {
		t.Fatalf("Update err: %v", err)
	}
	if updated == nil || updated.GetUsername() != testUsername {
		t.Fatalf("updated user mismatch: expected %s, got %v", testUsername, updated)
	}
	t.Logf("updated user: %s", updated.GetUsername())

	// Delete will be handled by defer cleanup
}

// Error scenario tests for database users

func TestDatabaseUsersService_Create_DuplicateUser(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping database user error tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewDatabaseUsersService(client)

	// Create a test user first
	testUsername := fmt.Sprintf("duplicate-test-user-%d", time.Now().Unix())
	testUser := &admin.CloudDatabaseUser{
		DatabaseName: "admin",
		Username:     testUsername,
		Password:     admin.PtrString("TestPassword123!"),
		Roles: &[]admin.DatabaseUserRole{
			{
				RoleName:     "read",
				DatabaseName: "admin",
			},
		},
	}

	// Cleanup function
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = svc.Delete(ctx, pid, "admin", testUsername)
	}()

	// Create the user first time - should succeed
	_, err = svc.Create(ctx, pid, testUser)
	if err != nil {
		t.Fatalf("First create should succeed: %v", err)
	}

	// Try to create the same user again - should fail
	_, err = svc.Create(ctx, pid, testUser)
	if err == nil {
		t.Error("Expected error when creating duplicate user, got nil")
	}
	t.Logf("Duplicate user error (expected): %v", err)
}

func TestDatabaseUsersService_Get_NonExistentUser(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping database user error tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewDatabaseUsersService(client)

	// Try to get a non-existent user
	user, err := svc.Get(ctx, pid, "admin", "non-existent-user-12345")
	if err == nil {
		t.Error("Expected error for non-existent user, got nil")
	}
	if user != nil {
		t.Error("Expected nil user for non-existent user")
	}
	t.Logf("Non-existent user error (expected): %v", err)
}

func TestDatabaseUsersService_Delete_NonExistentUser(t *testing.T) {
	_ = gotenv.Load("../../.env", "../.env", ".env")

	pid := os.Getenv("PROJECT_ID")
	if pid == "" {
		t.Skip("PROJECT_ID env not set; skipping database user error tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := atlasclient.NewClient(atlasclient.Config{})
	if err != nil {
		t.Fatalf("client err: %v", err)
	}
	svc := NewDatabaseUsersService(client)

	// Try to delete a non-existent user
	err = svc.Delete(ctx, pid, "admin", "non-existent-user-12345")
	if err == nil {
		t.Error("Expected error for deleting non-existent user, got nil")
	}
	t.Logf("Delete non-existent user error (expected): %v", err)
}
