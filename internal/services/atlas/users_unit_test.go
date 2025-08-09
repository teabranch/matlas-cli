package atlas

import (
	"context"
	"testing"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// Unit tests for DatabaseUsersService validation (no API calls)
func TestNewDatabaseUsersService(t *testing.T) {
	client := &atlasclient.Client{}
	service := NewDatabaseUsersService(client)

	if service == nil {
		t.Fatal("NewDatabaseUsersService returned nil")
	}
	if service.client != client {
		t.Fatal("NewDatabaseUsersService did not set client correctly")
	}
}

func TestDatabaseUsersService_List_Validation(t *testing.T) {
	service := NewDatabaseUsersService(&atlasclient.Client{})
	ctx := context.Background()

	// Test empty projectID
	users, err := service.List(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty projectID")
	}
	if users != nil {
		t.Fatal("expected nil users for empty projectID")
	}
	if err.Error() != "projectID required" {
		t.Fatalf("expected 'projectID required', got: %s", err.Error())
	}
}

func TestDatabaseUsersService_ListWithPagination_Validation(t *testing.T) {
	service := NewDatabaseUsersService(&atlasclient.Client{})
	ctx := context.Background()

	// empty projectID
	users, err := service.ListWithPagination(ctx, "", 1, 10, false)
	if err == nil || users != nil {
		t.Fatalf("expected error for empty projectID, got users=%v err=%v", users, err)
	}

	// invalid page
	users, err = service.ListWithPagination(ctx, "proj123", 0, 10, false)
	if err == nil || users != nil {
		t.Fatalf("expected error for invalid page, got users=%v err=%v", users, err)
	}

	// invalid limit
	users, err = service.ListWithPagination(ctx, "proj123", 1, 0, false)
	if err == nil || users != nil {
		t.Fatalf("expected error for invalid limit, got users=%v err=%v", users, err)
	}

	// Note: when all=true, the function performs API calls.
	// We avoid calling it here because the test uses a nil Atlas client.
}

func TestDatabaseUsersService_Get_Validation(t *testing.T) {
	service := NewDatabaseUsersService(&atlasclient.Client{})
	ctx := context.Background()

	tests := []struct {
		name         string
		projectID    string
		databaseName string
		username     string
		wantErr      string
	}{
		{
			name:         "empty projectID",
			projectID:    "",
			databaseName: "admin",
			username:     "testuser",
			wantErr:      "projectID, databaseName, and username are required",
		},
		{
			name:         "empty databaseName",
			projectID:    "proj123",
			databaseName: "",
			username:     "testuser",
			wantErr:      "projectID, databaseName, and username are required",
		},
		{
			name:         "empty username",
			projectID:    "proj123",
			databaseName: "admin",
			username:     "",
			wantErr:      "projectID, databaseName, and username are required",
		},
		{
			name:         "all empty",
			projectID:    "",
			databaseName: "",
			username:     "",
			wantErr:      "projectID, databaseName, and username are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Get(ctx, tt.projectID, tt.databaseName, tt.username)
			if err == nil {
				t.Fatal("expected error for invalid parameters")
			}
			if user != nil {
				t.Fatal("expected nil user for invalid parameters")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected '%s', got: %s", tt.wantErr, err.Error())
			}
		})
	}
}

func TestDatabaseUsersService_Create_Validation(t *testing.T) {
	service := NewDatabaseUsersService(&atlasclient.Client{})
	ctx := context.Background()

	tests := []struct {
		name      string
		projectID string
		user      *admin.CloudDatabaseUser
		wantErr   string
	}{
		{
			name:      "empty projectID",
			projectID: "",
			user:      &admin.CloudDatabaseUser{},
			wantErr:   "projectID and user are required",
		},
		{
			name:      "nil user",
			projectID: "proj123",
			user:      nil,
			wantErr:   "projectID and user are required",
		},
		{
			name:      "both invalid",
			projectID: "",
			user:      nil,
			wantErr:   "projectID and user are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Create(ctx, tt.projectID, tt.user)
			if err == nil {
				t.Fatal("expected error for invalid parameters")
			}
			if user != nil {
				t.Fatal("expected nil user for invalid parameters")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected '%s', got: %s", tt.wantErr, err.Error())
			}
		})
	}
}

func TestDatabaseUsersService_Update_Validation(t *testing.T) {
	service := NewDatabaseUsersService(&atlasclient.Client{})
	ctx := context.Background()

	tests := []struct {
		name         string
		projectID    string
		databaseName string
		username     string
		user         *admin.CloudDatabaseUser
		wantErr      string
	}{
		{
			name:         "empty projectID",
			projectID:    "",
			databaseName: "admin",
			username:     "testuser",
			user:         &admin.CloudDatabaseUser{},
			wantErr:      "projectID, databaseName, username, and user are required",
		},
		{
			name:         "empty databaseName",
			projectID:    "proj123",
			databaseName: "",
			username:     "testuser",
			user:         &admin.CloudDatabaseUser{},
			wantErr:      "projectID, databaseName, username, and user are required",
		},
		{
			name:         "empty username",
			projectID:    "proj123",
			databaseName: "admin",
			username:     "",
			user:         &admin.CloudDatabaseUser{},
			wantErr:      "projectID, databaseName, username, and user are required",
		},
		{
			name:         "nil user",
			projectID:    "proj123",
			databaseName: "admin",
			username:     "testuser",
			user:         nil,
			wantErr:      "projectID, databaseName, username, and user are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.Update(ctx, tt.projectID, tt.databaseName, tt.username, tt.user)
			if err == nil {
				t.Fatal("expected error for invalid parameters")
			}
			if user != nil {
				t.Fatal("expected nil user for invalid parameters")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected '%s', got: %s", tt.wantErr, err.Error())
			}
		})
	}
}

func TestDatabaseUsersService_Delete_Validation(t *testing.T) {
	service := NewDatabaseUsersService(&atlasclient.Client{})
	ctx := context.Background()

	tests := []struct {
		name         string
		projectID    string
		databaseName string
		username     string
		wantErr      string
	}{
		{
			name:         "empty projectID",
			projectID:    "",
			databaseName: "admin",
			username:     "testuser",
			wantErr:      "projectID, databaseName, and username are required",
		},
		{
			name:         "empty databaseName",
			projectID:    "proj123",
			databaseName: "",
			username:     "testuser",
			wantErr:      "projectID, databaseName, and username are required",
		},
		{
			name:         "empty username",
			projectID:    "proj123",
			databaseName: "admin",
			username:     "",
			wantErr:      "projectID, databaseName, and username are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Delete(ctx, tt.projectID, tt.databaseName, tt.username)
			if err == nil {
				t.Fatal("expected error for invalid parameters")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("expected '%s', got: %s", tt.wantErr, err.Error())
			}
		})
	}
}

func TestDatabaseUsersService_StructBasics(t *testing.T) {
	// Test that we can create a user structure
	user := &admin.CloudDatabaseUser{}

	// Test basic field assignment that matches existing tests
	user.DatabaseName = "admin"
	user.Username = "testuser"

	if user.DatabaseName != "admin" {
		t.Fatal("expected database name to be 'admin'")
	}
	if user.Username != "testuser" {
		t.Fatal("expected username to be 'testuser'")
	}
}
