package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// MockDatabaseUsersService for testing
type MockDatabaseUsersService struct {
	CreateFunc func(ctx context.Context, projectID string, user *admin.CloudDatabaseUser) (*admin.CloudDatabaseUser, error)
	DeleteFunc func(ctx context.Context, projectID, databaseName, username string) error
	ListFunc   func(ctx context.Context, projectID string) ([]admin.CloudDatabaseUser, error)
}

func (m *MockDatabaseUsersService) Create(ctx context.Context, projectID string, user *admin.CloudDatabaseUser) (*admin.CloudDatabaseUser, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, projectID, user)
	}
	return user, nil
}

func (m *MockDatabaseUsersService) Delete(ctx context.Context, projectID, databaseName, username string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, projectID, databaseName, username)
	}
	return nil
}

func (m *MockDatabaseUsersService) List(ctx context.Context, projectID string) ([]admin.CloudDatabaseUser, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, projectID)
	}
	return []admin.CloudDatabaseUser{}, nil
}

func TestNewTempUserManager(t *testing.T) {
	mockService := &MockDatabaseUsersService{}
	projectID := "test-project-id"

	manager := NewTempUserManager(mockService, projectID)

	if manager == nil {
		t.Fatal("NewTempUserManager returned nil")
	}

	if manager.projectID != projectID {
		t.Errorf("Expected projectID '%s', got '%s'", projectID, manager.projectID)
	}
}

func TestTempUserManager_CreateTempUserForDiscovery(t *testing.T) {
	tests := []struct {
		name         string
		clusterNames []string
		createError  error
		expectError  bool
		description  string
	}{
		{
			name:         "Successful Creation",
			clusterNames: []string{"cluster1", "cluster2"},
			createError:  nil,
			expectError:  false,
			description:  "Should create temporary user successfully",
		},
		{
			name:         "API Error",
			clusterNames: []string{"cluster1"},
			createError:  errors.New("atlas api error"),
			expectError:  true,
			description:  "Should return error when Atlas API fails",
		},
		{
			name:         "Empty Cluster Names",
			clusterNames: []string{},
			createError:  nil,
			expectError:  false,
			description:  "Should handle empty cluster names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockDatabaseUsersService{
				CreateFunc: func(ctx context.Context, projectID string, user *admin.CloudDatabaseUser) (*admin.CloudDatabaseUser, error) {
					if tt.createError != nil {
						return nil, tt.createError
					}

					// Validate user configuration using current API structure
					if user.Username == "" {
						t.Error("Expected username to be set")
					}

					if user.Password == nil || *user.Password == "" {
						t.Error("Expected password to be set")
					}

					if user.DatabaseName != "admin" {
						t.Error("Expected database name to be 'admin'")
					}

					if user.Roles == nil || len(*user.Roles) == 0 {
						t.Error("Expected roles to be set")
					}

					// Check for proper permissions (discovery operations require admin-level access)
					hasProperRole := false
					for _, role := range *user.Roles {
						if role.RoleName == "readWriteAnyDatabase" || role.RoleName == "dbAdminAnyDatabase" {
							hasProperRole = true
							break
						}
					}
					if !hasProperRole {
						t.Error("Expected user to have proper discovery roles (readWriteAnyDatabase or dbAdminAnyDatabase)")
					}

					if len(tt.clusterNames) > 0 && (user.Scopes == nil || len(*user.Scopes) == 0) {
						t.Error("Expected scopes to be set when cluster names provided")
					}

					return user, nil
				},
			}

			manager := NewTempUserManager(mockService, "test-project-id")
			ctx := context.Background()

			result, err := manager.CreateTempUserForDiscovery(ctx, tt.clusterNames, "")

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			if result.Username == "" {
				t.Error("Username should not be empty")
			}

			if result.Password == "" {
				t.Error("Password should not be empty")
			}

			if result.ExpiresAt.IsZero() {
				t.Error("ExpiresAt should be set")
			}

			if result.CleanupFunc == nil {
				t.Error("CleanupFunc should not be nil")
			}
		})
	}
}

func TestTempUserManager_CreateTempUserForDiscovery_WithDatabaseName(t *testing.T) {
	mockService := &MockDatabaseUsersService{}
	projectID := "test-project-id"
	manager := NewTempUserManager(mockService, projectID)

	var createdUser *admin.CloudDatabaseUser
	mockService.CreateFunc = func(ctx context.Context, projectID string, user *admin.CloudDatabaseUser) (*admin.CloudDatabaseUser, error) {
		createdUser = user
		return user, nil
	}

	ctx := context.Background()
	clusterNames := []string{"test-cluster"}
	databaseName := "myapp"

	result, err := manager.CreateTempUserForDiscovery(ctx, clusterNames, databaseName)

	if err != nil {
		t.Fatalf("CreateTempUserForDiscovery failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	// Verify that the created user has roles defined
	// Note: Atlas users always use "admin" database for authentication
	if createdUser == nil || createdUser.Roles == nil || len(*createdUser.Roles) == 0 {
		t.Fatal("Expected user to have roles defined")
	}

	roles := *createdUser.Roles
	for _, role := range roles {
		// When a specific database name is provided, roles should be scoped to that database for better security
		if role.DatabaseName != databaseName {
			t.Errorf("Expected role to have database name '%s', got '%s'", databaseName, role.DatabaseName)
		}
	}

	// Verify specific roles are present for database-specific operations
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.RoleName
	}

	expectedRoles := []string{"readWrite", "dbAdmin"}
	for _, expectedRole := range expectedRoles {
		found := false
		for _, roleName := range roleNames {
			if roleName == expectedRole {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected role '%s' not found in roles: %v", expectedRole, roleNames)
		}
	}
}

func TestTempUserManager_CreateTempUserForDiscovery_DefaultDatabase(t *testing.T) {
	mockService := &MockDatabaseUsersService{}
	projectID := "test-project-id"
	manager := NewTempUserManager(mockService, projectID)

	var createdUser *admin.CloudDatabaseUser
	mockService.CreateFunc = func(ctx context.Context, projectID string, user *admin.CloudDatabaseUser) (*admin.CloudDatabaseUser, error) {
		createdUser = user
		return user, nil
	}

	ctx := context.Background()
	clusterNames := []string{"test-cluster"}

	result, err := manager.CreateTempUserForDiscovery(ctx, clusterNames, "")

	if err != nil {
		t.Fatalf("CreateTempUserForDiscovery failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	// Verify that the created user has the default "admin" database name in roles
	if createdUser == nil || createdUser.Roles == nil || len(*createdUser.Roles) == 0 {
		t.Fatal("Expected user to have roles defined")
	}

	roles := *createdUser.Roles
	for _, role := range roles {
		if role.DatabaseName != "admin" {
			t.Errorf("Expected role to have default database name 'admin', got '%s'", role.DatabaseName)
		}
	}
}

func TestTempUserManager_CreateTempUser(t *testing.T) {
	tests := []struct {
		name        string
		config      TempUserConfig
		createError error
		expectError bool
		description string
	}{
		{
			name: "Basic Configuration",
			config: TempUserConfig{
				Purpose: "test",
				TTL:     30 * time.Minute,
			},
			createError: nil,
			expectError: false,
			description: "Should create user with basic configuration",
		},
		{
			name: "Custom Username and Password",
			config: TempUserConfig{
				Username: "custom-user",
				Password: "custom-pass",
				Purpose:  "custom-test",
				TTL:      1 * time.Hour,
			},
			createError: nil,
			expectError: false,
			description: "Should use provided username and password",
		},
		{
			name: "Custom Roles",
			config: TempUserConfig{
				Purpose: "admin-test",
				TTL:     1 * time.Hour,
				Roles: []admin.DatabaseUserRole{
					{
						RoleName:     "readWrite",
						DatabaseName: "testdb",
					},
				},
			},
			createError: nil,
			expectError: false,
			description: "Should use custom roles",
		},
		{
			name: "API Failure",
			config: TempUserConfig{
				Purpose: "failure-test",
				TTL:     30 * time.Minute,
			},
			createError: errors.New("api failure"),
			expectError: true,
			description: "Should handle API failures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockDatabaseUsersService{
				CreateFunc: func(ctx context.Context, projectID string, user *admin.CloudDatabaseUser) (*admin.CloudDatabaseUser, error) {
					if tt.createError != nil {
						return nil, tt.createError
					}

					// Validate the configuration was applied correctly
					if tt.config.Username != "" && user.Username != tt.config.Username {
						t.Errorf("Expected username '%s', got '%s'", tt.config.Username, user.Username)
					}

					if tt.config.Password != "" && (user.Password == nil || *user.Password != tt.config.Password) {
						t.Errorf("Expected password '%s', got '%v'", tt.config.Password, user.Password)
					}

					if len(tt.config.Roles) > 0 {
						if user.Roles == nil {
							t.Error("Expected roles to be set")
						} else {
							actualRoles := *user.Roles
							if len(actualRoles) != len(tt.config.Roles) {
								t.Errorf("Expected %d roles, got %d", len(tt.config.Roles), len(actualRoles))
							}
						}
					}

					return user, nil
				},
			}

			manager := NewTempUserManager(mockService, "test-project-id")
			ctx := context.Background()

			result, err := manager.CreateTempUser(ctx, tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Result is nil")
			}

			// Validate result
			if result.Username == "" {
				t.Error("Username should not be empty")
			}

			if result.Password == "" {
				t.Error("Password should not be empty")
			}

			if result.CleanupFunc == nil {
				t.Error("CleanupFunc should not be nil")
			}
		})
	}
}

func TestTempUserManager_CleanupExpiredUsers(t *testing.T) {
	// Create test users
	now := time.Now()
	expiredUser := admin.CloudDatabaseUser{
		Username:        "expired-user",
		DatabaseName:    "admin",
		DeleteAfterDate: admin.PtrTime(now.Add(-1 * time.Hour)), // Expired
		Labels: &[]admin.ComponentLabel{
			{
				Key:   admin.PtrString("temporary"),
				Value: admin.PtrString("true"),
			},
		},
	}

	validUser := admin.CloudDatabaseUser{
		Username:        "valid-user",
		DatabaseName:    "admin",
		DeleteAfterDate: admin.PtrTime(now.Add(1 * time.Hour)), // Not expired
		Labels: &[]admin.ComponentLabel{
			{
				Key:   admin.PtrString("temporary"),
				Value: admin.PtrString("true"),
			},
		},
	}

	regularUser := admin.CloudDatabaseUser{
		Username:     "regular-user",
		DatabaseName: "admin",
		// No temporary label
	}

	tests := []struct {
		name        string
		users       []admin.CloudDatabaseUser
		deleteError error
		expectError bool
		description string
	}{
		{
			name:        "No Users",
			users:       []admin.CloudDatabaseUser{},
			deleteError: nil,
			expectError: false,
			description: "Should handle empty user list",
		},
		{
			name:        "Mixed Users",
			users:       []admin.CloudDatabaseUser{expiredUser, validUser, regularUser},
			deleteError: nil,
			expectError: false,
			description: "Should only clean up expired temporary users",
		},
		{
			name:        "Delete Error",
			users:       []admin.CloudDatabaseUser{expiredUser},
			deleteError: errors.New("delete failed"),
			expectError: true,
			description: "Should handle delete errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletedUsers := []string{}

			mockService := &MockDatabaseUsersService{
				ListFunc: func(ctx context.Context, projectID string) ([]admin.CloudDatabaseUser, error) {
					return tt.users, nil
				},
				DeleteFunc: func(ctx context.Context, projectID, databaseName, username string) error {
					if tt.deleteError != nil {
						return tt.deleteError
					}
					deletedUsers = append(deletedUsers, username)
					return nil
				},
			}

			manager := NewTempUserManager(mockService, "test-project-id")
			ctx := context.Background()

			err := manager.CleanupExpiredUsers(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify only expired temporary users were deleted
			expectedDeleted := 0
			for _, user := range tt.users {
				if manager.isTempUser(user) && manager.isExpired(user, now) {
					expectedDeleted++
				}
			}

			if len(deletedUsers) != expectedDeleted {
				t.Errorf("Expected %d users to be deleted, got %d", expectedDeleted, len(deletedUsers))
			}
		})
	}
}

func TestTempUserManager_isTempUser(t *testing.T) {
	manager := &TempUserManager{}

	tests := []struct {
		name     string
		user     admin.CloudDatabaseUser
		expected bool
	}{
		{
			name: "Temporary User",
			user: admin.CloudDatabaseUser{
				Labels: &[]admin.ComponentLabel{
					{
						Key:   admin.PtrString("temporary"),
						Value: admin.PtrString("true"),
					},
				},
			},
			expected: true,
		},
		{
			name: "Non-Temporary User",
			user: admin.CloudDatabaseUser{
				Labels: &[]admin.ComponentLabel{
					{
						Key:   admin.PtrString("other"),
						Value: admin.PtrString("value"),
					},
				},
			},
			expected: false,
		},
		{
			name: "User Without Labels",
			user: admin.CloudDatabaseUser{
				Labels: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.isTempUser(tt.user)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTempUserManager_isExpired(t *testing.T) {
	manager := &TempUserManager{}
	now := time.Now()

	tests := []struct {
		name     string
		user     admin.CloudDatabaseUser
		expected bool
	}{
		{
			name: "Expired User",
			user: admin.CloudDatabaseUser{
				DeleteAfterDate: admin.PtrTime(now.Add(-1 * time.Hour)),
			},
			expected: true,
		},
		{
			name: "Non-Expired User",
			user: admin.CloudDatabaseUser{
				DeleteAfterDate: admin.PtrTime(now.Add(1 * time.Hour)),
			},
			expected: false,
		},
		{
			name: "User Without Expiration",
			user: admin.CloudDatabaseUser{
				DeleteAfterDate: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.isExpired(tt.user, now)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
