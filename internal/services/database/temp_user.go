package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"go.mongodb.org/atlas-sdk/v20250312005/admin"
)

// DatabaseUsersServiceInterface defines the interface for database user operations
type DatabaseUsersServiceInterface interface {
	Create(ctx context.Context, projectID string, user *admin.CloudDatabaseUser) (*admin.CloudDatabaseUser, error)
	Delete(ctx context.Context, projectID, databaseName, username string) error
	List(ctx context.Context, projectID string) ([]admin.CloudDatabaseUser, error)
}

// TempUserManager manages temporary database users for command execution
type TempUserManager struct {
	usersService DatabaseUsersServiceInterface
	projectID    string
}

// TempUserConfig represents configuration for a temporary user
type TempUserConfig struct {
	Username     string
	Password     string
	Roles        []admin.DatabaseUserRole
	Scopes       []admin.UserScope
	TTL          time.Duration // Time-to-live for the user
	ClusterNames []string      // Clusters the user needs access to
	Purpose      string        // Description of why the user was created
}

// TempUserResult represents the result of temporary user creation
type TempUserResult struct {
	Username      string
	Password      string
	ExpiresAt     time.Time
	CleanupFunc   func(context.Context) error
	ConnectionURI string
}

// NewTempUserManager creates a new temporary user manager
func NewTempUserManager(usersService DatabaseUsersServiceInterface, projectID string) *TempUserManager {
	return &TempUserManager{
		usersService: usersService,
		projectID:    projectID,
	}
}

// CreateTempUser creates a temporary database user with the given configuration
func (m *TempUserManager) CreateTempUser(ctx context.Context, config TempUserConfig) (*TempUserResult, error) {
	username := config.Username
	if username == "" {
		username = generateTempUsername(config.Purpose)
	}

	// Respect a caller-supplied password if provided, otherwise generate one.
	password := config.Password
	if password == "" {
		password = generateSecurePassword()
	}

	// Create Atlas SDK user object
	atlasUser := &admin.CloudDatabaseUser{
		Username:     username,
		DatabaseName: "admin", // Atlas users must be created on admin database
		Password:     admin.PtrString(password),
	}

	// Set default TTL if not provided
	if config.TTL == 0 {
		config.TTL = 1 * time.Hour // Default 1 hour
	}

	// Set default roles if not provided (read-only access)
	if len(config.Roles) == 0 {
		config.Roles = []admin.DatabaseUserRole{
			{
				RoleName:     "readAnyDatabase",
				DatabaseName: "admin",
			},
		}
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(config.TTL)

	// Set roles on the user
	atlasUser.Roles = &config.Roles

	// Add scopes if provided
	if len(config.Scopes) > 0 {
		atlasUser.Scopes = &config.Scopes
	}

	// Set expiration time
	atlasUser.DeleteAfterDate = admin.PtrTime(expiresAt)

	// Create the user
	_, err := m.usersService.Create(ctx, m.projectID, atlasUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary user: %w", err)
	}

	// Create cleanup function
	cleanupFunc := func(cleanupCtx context.Context) error {
		return m.usersService.Delete(cleanupCtx, m.projectID, "admin", username)
	}

	// Generate connection URI (simplified - in practice, you'd want proper cluster resolution)
	connectionURI := fmt.Sprintf("mongodb+srv://%s:%s@<cluster-endpoint>/<database>",
		username, password)

	return &TempUserResult{
		Username:      username,
		Password:      password,
		ExpiresAt:     expiresAt,
		CleanupFunc:   cleanupFunc,
		ConnectionURI: connectionURI,
	}, nil
}

// CreateTempUserForDiscovery creates a temporary user specifically for database discovery
func (m *TempUserManager) CreateTempUserForDiscovery(ctx context.Context, clusterNames []string, databaseName string) (*TempUserResult, error) {
	// Create scopes for all specified clusters
	var scopes []admin.UserScope
	for _, clusterName := range clusterNames {
		scopes = append(scopes, admin.UserScope{
			Name: clusterName,
			Type: "CLUSTER",
		})
	}

	if databaseName == "" {
		databaseName = "admin" // Default to admin when none provided
	}

	// Determine the least-privileged role name according to requirements
	roleName := "read"           // default expectation
	if databaseName != "admin" { // specific DB supplied – allow readAnyDatabase
		roleName = "readAnyDatabase"
	}

	config := TempUserConfig{
		Purpose:      "database-discovery",
		TTL:          5 * time.Minute, // Reduced TTL since we cleanup immediately
		ClusterNames: clusterNames,
		Scopes:       scopes,
		Roles: []admin.DatabaseUserRole{
			{
				RoleName:     roleName,
				DatabaseName: databaseName,
			},
		},
	}

	return m.CreateTempUser(ctx, config)
}

// CreateTempUserForMaintenance creates a temporary user for maintenance operations
func (m *TempUserManager) CreateTempUserForMaintenance(ctx context.Context, clusterNames []string) (*TempUserResult, error) {
	var scopes []admin.UserScope
	for _, clusterName := range clusterNames {
		scopes = append(scopes, admin.UserScope{
			Name: clusterName,
			Type: "CLUSTER",
		})
	}

	config := TempUserConfig{
		Purpose:      "maintenance-operations",
		TTL:          2 * time.Hour, // Longer TTL for maintenance
		ClusterNames: clusterNames,
		Scopes:       scopes,
		Roles: []admin.DatabaseUserRole{
			{
				RoleName:     "readWriteAnyDatabase",
				DatabaseName: "admin",
			},
			{
				RoleName:     "dbAdminAnyDatabase",
				DatabaseName: "admin",
			},
		},
	}

	return m.CreateTempUser(ctx, config)
}

// CleanupExpiredUsers removes expired temporary users (for cleanup operations)
func (m *TempUserManager) CleanupExpiredUsers(ctx context.Context) error {
	// List all users in the project
	users, err := m.usersService.List(ctx, m.projectID)
	if err != nil {
		return fmt.Errorf("failed to list users for cleanup: %w", err)
	}

	var errors []error
	now := time.Now()

	for _, user := range users {
		// Check if user is temporary and expired
		if m.isTempUser(user) && m.isExpired(user, now) {
			err := m.usersService.Delete(ctx, m.projectID, user.GetDatabaseName(), user.GetUsername())
			if err != nil {
				errors = append(errors, fmt.Errorf("failed to cleanup user %s: %w", user.GetUsername(), err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// Helper functions

func generateTempUsername(purpose string) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)

	if purpose == "" {
		purpose = "temp"
	}

	return fmt.Sprintf("matlas-%s-%d-%s", purpose, timestamp, randomStr)
}

func generateSecurePassword() string {
	// Generate a secure random password using URL-safe characters
	// Avoiding special characters that could cause URL encoding issues
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, 32)

	for i := range password {
		randomBytes := make([]byte, 1)
		rand.Read(randomBytes)
		password[i] = charset[int(randomBytes[0])%len(charset)]
	}

	return string(password)
}

func (m *TempUserManager) isTempUser(user admin.CloudDatabaseUser) bool {
	if user.Labels == nil {
		return false
	}

	for _, label := range *user.Labels {
		if label.GetKey() == "temporary" && label.GetValue() == "true" {
			return true
		}
	}
	return false
}

func (m *TempUserManager) isExpired(user admin.CloudDatabaseUser, now time.Time) bool {
	if user.DeleteAfterDate == nil {
		return false
	}

	return user.DeleteAfterDate.Before(now)
}
