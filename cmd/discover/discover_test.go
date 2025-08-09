package discover

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/types"
)

// TestNewDiscoverCmd tests the command creation and flag setup
func TestNewDiscoverCmd(t *testing.T) {
	cmd := NewDiscoverCmd()
	require.NotNil(t, cmd)

	// Test basic command properties
	assert.Equal(t, "discover", cmd.Use)
	assert.Contains(t, cmd.Short, "Discover Atlas project")
	assert.Contains(t, cmd.Long, "This command connects to Atlas")
	assert.NotEmpty(t, cmd.Example)

	// Test required flags
	projectIDFlag := cmd.Flags().Lookup("project-id")
	require.NotNil(t, projectIDFlag)
	assert.Equal(t, "", projectIDFlag.DefValue)

	// Test optional flags with defaults
	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag)
	assert.Equal(t, "yaml", outputFlag.DefValue)

	outputFileFlag := cmd.Flags().Lookup("output-file")
	require.NotNil(t, outputFileFlag)
	assert.Equal(t, "", outputFileFlag.DefValue)
	assert.Equal(t, "o", outputFileFlag.Shorthand)

	maskSecretsFlag := cmd.Flags().Lookup("mask-secrets")
	require.NotNil(t, maskSecretsFlag)
	assert.Equal(t, "false", maskSecretsFlag.DefValue)

	includeDatabasesFlag := cmd.Flags().Lookup("include-databases")
	require.NotNil(t, includeDatabasesFlag)
	assert.Equal(t, "false", includeDatabasesFlag.DefValue)

	parallelFlag := cmd.Flags().Lookup("parallel")
	require.NotNil(t, parallelFlag)
	assert.Equal(t, "true", parallelFlag.DefValue)

	maxConcurrencyFlag := cmd.Flags().Lookup("max-concurrency")
	require.NotNil(t, maxConcurrencyFlag)
	assert.Equal(t, "5", maxConcurrencyFlag.DefValue)

	timeoutFlag := cmd.Flags().Lookup("timeout")
	require.NotNil(t, timeoutFlag)
	assert.Equal(t, "10m0s", timeoutFlag.DefValue)

	// Test new convert-to-apply flag
	convertFlag := cmd.Flags().Lookup("convert-to-apply")
	require.NotNil(t, convertFlag)
	assert.Equal(t, "false", convertFlag.DefValue)

	// Test new db enumeration flags
	mongoURIF := cmd.Flags().Lookup("mongo-uri")
	require.NotNil(t, mongoURIF)
	assert.Equal(t, "", mongoURIF.DefValue)

	mongoUserF := cmd.Flags().Lookup("mongo-username")
	require.NotNil(t, mongoUserF)
	assert.Equal(t, "", mongoUserF.DefValue)

	mongoPassF := cmd.Flags().Lookup("mongo-password")
	require.NotNil(t, mongoPassF)
	assert.Equal(t, "", mongoPassF.DefValue)

	cacheStatsF := cmd.Flags().Lookup("cache-stats")
	require.NotNil(t, cacheStatsF)
	assert.Equal(t, "false", cacheStatsF.DefValue)

	// Temp user flags
	useTemp := cmd.Flags().Lookup("use-temp-user")
	require.NotNil(t, useTemp)
	assert.Equal(t, "false", useTemp.DefValue)

	tempDb := cmd.Flags().Lookup("temp-user-database")
	require.NotNil(t, tempDb)
	assert.Equal(t, "", tempDb.DefValue)
}

// TestDiscoverOptions tests the options structure
func TestDiscoverOptions(t *testing.T) {
	opts := &DiscoverOptions{
		ProjectID:        "507f1f77bcf86cd799439011",
		OutputFormat:     "json",
		OutputFile:       "output.json",
		IncludeTypes:     []string{"clusters", "users"},
		ExcludeTypes:     []string{"network"},
		MaskSecrets:      true,
		IncludeDatabases: true,
		NoCache:          false,
		Timeout:          5 * time.Minute,
		Verbose:          true,
		Parallel:         true,
		MaxConcurrency:   10,
		ConvertToApply:   true,
	}

	assert.Equal(t, "507f1f77bcf86cd799439011", opts.ProjectID)
	assert.Equal(t, "json", opts.OutputFormat)
	assert.Equal(t, "output.json", opts.OutputFile)
	assert.Equal(t, []string{"clusters", "users"}, opts.IncludeTypes)
	assert.Equal(t, []string{"network"}, opts.ExcludeTypes)
	assert.True(t, opts.MaskSecrets)
	assert.True(t, opts.IncludeDatabases)
	assert.False(t, opts.NoCache)
	assert.Equal(t, 5*time.Minute, opts.Timeout)
	assert.True(t, opts.Verbose)
	assert.True(t, opts.Parallel)
	assert.Equal(t, 10, opts.MaxConcurrency)
	assert.True(t, opts.ConvertToApply)
}

// TestDiscoveryResult tests the discovery result structure
func TestDiscoveryResult(t *testing.T) {
	result := &DiscoveryResult{
		APIVersion: "matlas.mongodb.com/v1",
		Kind:       "DiscoveredProject",
		Metadata: DiscoveryMetadata{
			Name:         "discovery-test",
			ProjectID:    "507f1f77bcf86cd799439011",
			DiscoveredAt: time.Now().UTC().Format(time.RFC3339),
			Version:      "1.0.0",
			Fingerprint:  "abc123",
			Options:      map[string]interface{}{"verbose": true},
			Stats: DiscoveryStats{
				ClustersFound:       2,
				DatabaseUsersFound:  5,
				NetworkEntriesFound: 3,
				DatabasesFound:      10,
				CollectionsFound:    25,
				Duration:            30 * time.Second,
				CacheHit:            false,
			},
			Labels: map[string]string{
				"matlas.mongodb.com/discovered-by": "matlas-cli",
				"matlas.mongodb.com/project-id":    "507f1f77bcf86cd799439011",
			},
		},
	}

	assert.Equal(t, "matlas.mongodb.com/v1", result.APIVersion)
	assert.Equal(t, "DiscoveredProject", result.Kind)
	assert.Equal(t, "discovery-test", result.Metadata.Name)
	assert.Equal(t, "507f1f77bcf86cd799439011", result.Metadata.ProjectID)
	assert.Equal(t, 2, result.Metadata.Stats.ClustersFound)
	assert.Equal(t, 5, result.Metadata.Stats.DatabaseUsersFound)
	assert.Equal(t, 3, result.Metadata.Stats.NetworkEntriesFound)
	assert.Equal(t, 10, result.Metadata.Stats.DatabasesFound)
	assert.Equal(t, 25, result.Metadata.Stats.CollectionsFound)
	assert.Equal(t, 30*time.Second, result.Metadata.Stats.Duration)
	assert.False(t, result.Metadata.Stats.CacheHit)
}

// TestShouldIncludeType tests the include type filtering logic
func TestShouldIncludeType(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		opts         *DiscoverOptions
		expected     bool
	}{
		{
			name:         "empty include list includes all",
			resourceType: "clusters",
			opts:         &DiscoverOptions{IncludeTypes: []string{}},
			expected:     true,
		},
		{
			name:         "nil include list includes all",
			resourceType: "users",
			opts:         &DiscoverOptions{IncludeTypes: nil},
			expected:     true,
		},
		{
			name:         "specific type included",
			resourceType: "clusters",
			opts:         &DiscoverOptions{IncludeTypes: []string{"clusters", "users"}},
			expected:     true,
		},
		{
			name:         "specific type not included",
			resourceType: "network",
			opts:         &DiscoverOptions{IncludeTypes: []string{"clusters", "users"}},
			expected:     false,
		},
		{
			name:         "case insensitive matching",
			resourceType: "CLUSTERS",
			opts:         &DiscoverOptions{IncludeTypes: []string{"clusters"}},
			expected:     true,
		},
		{
			name:         "case insensitive matching reverse",
			resourceType: "clusters",
			opts:         &DiscoverOptions{IncludeTypes: []string{"CLUSTERS"}},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIncludeType(tt.resourceType, tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestShouldExcludeType tests the exclude type filtering logic
func TestShouldExcludeType(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		opts         *DiscoverOptions
		expected     bool
	}{
		{
			name:         "empty exclude list excludes nothing",
			resourceType: "clusters",
			opts:         &DiscoverOptions{ExcludeTypes: []string{}},
			expected:     false,
		},
		{
			name:         "nil exclude list excludes nothing",
			resourceType: "users",
			opts:         &DiscoverOptions{ExcludeTypes: nil},
			expected:     false,
		},
		{
			name:         "specific type excluded",
			resourceType: "network",
			opts:         &DiscoverOptions{ExcludeTypes: []string{"network", "databases"}},
			expected:     true,
		},
		{
			name:         "specific type not excluded",
			resourceType: "clusters",
			opts:         &DiscoverOptions{ExcludeTypes: []string{"network", "databases"}},
			expected:     false,
		},
		{
			name:         "case insensitive matching",
			resourceType: "NETWORK",
			opts:         &DiscoverOptions{ExcludeTypes: []string{"network"}},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldExcludeType(tt.resourceType, tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestApplyFiltering tests the filtering logic
func TestApplyFiltering(t *testing.T) {
	// Create mock project state
	projectState := createMockProjectState()

	tests := []struct {
		name     string
		opts     *DiscoverOptions
		validate func(t *testing.T, result *DiscoveryResult)
	}{
		{
			name: "no filtering includes everything",
			opts: &DiscoverOptions{},
			validate: func(t *testing.T, result *DiscoveryResult) {
				assert.NotNil(t, result.Project)
				assert.Len(t, result.Clusters, 2)
				assert.Len(t, result.DatabaseUsers, 3)
				assert.Len(t, result.NetworkAccess, 1)
			},
		},
		{
			name: "include only clusters",
			opts: &DiscoverOptions{IncludeTypes: []string{"clusters"}},
			validate: func(t *testing.T, result *DiscoveryResult) {
				assert.Nil(t, result.Project)
				assert.Len(t, result.Clusters, 2)
				assert.Len(t, result.DatabaseUsers, 0)
				assert.Len(t, result.NetworkAccess, 0)
			},
		},
		{
			name: "include clusters and users",
			opts: &DiscoverOptions{IncludeTypes: []string{"clusters", "users"}},
			validate: func(t *testing.T, result *DiscoveryResult) {
				assert.Nil(t, result.Project)
				assert.Len(t, result.Clusters, 2)
				assert.Len(t, result.DatabaseUsers, 3)
				assert.Len(t, result.NetworkAccess, 0)
			},
		},
		{
			name: "exclude network",
			opts: &DiscoverOptions{ExcludeTypes: []string{"network"}},
			validate: func(t *testing.T, result *DiscoveryResult) {
				assert.NotNil(t, result.Project)
				assert.Len(t, result.Clusters, 2)
				assert.Len(t, result.DatabaseUsers, 3)
				assert.Len(t, result.NetworkAccess, 0)
			},
		},
		{
			name: "include project only",
			opts: &DiscoverOptions{IncludeTypes: []string{"project"}},
			validate: func(t *testing.T, result *DiscoveryResult) {
				assert.NotNil(t, result.Project)
				assert.Len(t, result.Clusters, 0)
				assert.Len(t, result.DatabaseUsers, 0)
				assert.Len(t, result.NetworkAccess, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &DiscoveryResult{}
			err := applyFiltering(result, projectState, tt.opts)
			require.NoError(t, err)
			tt.validate(t, result)
		})
	}
}

// TestMaskSecrets tests secret masking functionality
func TestMaskSecrets(t *testing.T) {
	result := &DiscoveryResult{
		DatabaseUsers: []types.DatabaseUserManifest{
			{
				Spec: types.DatabaseUserSpec{
					Username: "user1",
					Password: "secretpassword123",
				},
			},
			{
				Spec: types.DatabaseUserSpec{
					Username: "user2",
					Password: "anothersecret456",
				},
			},
		},
	}

	maskSecrets(result)

	for i, user := range result.DatabaseUsers {
		assert.Equal(t, "***MASKED***", user.Spec.Password, "Password should be masked for user %d", i)
	}
}

// TestOutputResult tests output formatting
func TestOutputResult(t *testing.T) {
	result := createMockDiscoveryResult()

	tests := []struct {
		name         string
		outputFormat string
		validate     func(t *testing.T, output string)
		expectError  bool
	}{
		{
			name:         "JSON output",
			outputFormat: "json",
			validate: func(t *testing.T, output string) {
				var jsonResult DiscoveryResult
				err := json.Unmarshal([]byte(output), &jsonResult)
				require.NoError(t, err)
				assert.Equal(t, "DiscoveredProject", jsonResult.Kind)
				assert.Equal(t, "test-project", jsonResult.Metadata.ProjectID)
			},
		},
		{
			name:         "YAML output",
			outputFormat: "yaml",
			validate: func(t *testing.T, output string) {
				var yamlResult DiscoveryResult
				err := yaml.Unmarshal([]byte(output), &yamlResult)
				require.NoError(t, err)
				assert.Equal(t, "DiscoveredProject", yamlResult.Kind)
				assert.Equal(t, "test-project", yamlResult.Metadata.ProjectID)
			},
		},
		{
			name:         "YML output (alias)",
			outputFormat: "yml",
			validate: func(t *testing.T, output string) {
				var yamlResult DiscoveryResult
				err := yaml.Unmarshal([]byte(output), &yamlResult)
				require.NoError(t, err)
				assert.Equal(t, "DiscoveredProject", yamlResult.Kind)
			},
		},
		{
			name:         "Invalid output format",
			outputFormat: "xml",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := &DiscoverOptions{OutputFormat: tt.outputFormat}

			// Create a temporary file to capture output
			tempFile, err := os.CreateTemp("", "test-output-*")
			require.NoError(t, err)
			defer os.Remove(tempFile.Name())
			defer tempFile.Close()

			opts.OutputFile = tempFile.Name()

			// Test the output
			err = outputResult(result, opts)
			if tt.expectError {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
			}

			// Read the output from the temporary file
			tempFile.Seek(0, 0)
			buf.ReadFrom(tempFile)

			if tt.validate != nil {
				tt.validate(t, buf.String())
			}
		})
	}
}

// TestDatabaseInfo tests database info structures
func TestDatabaseInfo(t *testing.T) {
	dbInfo := DatabaseInfo{
		Name:        "testdb",
		ClusterName: "Cluster0",
		SizeOnDisk:  1024000,
		Collections: []CollectionInfo{
			{
				Name:          "users",
				DocumentCount: 1000,
				StorageSize:   512000,
				IndexCount:    3,
				Indexes: []IndexInfo{
					{
						Name:   "_id_",
						Keys:   map[string]int{"_id": 1},
						Unique: true,
					},
					{
						Name: "email_1",
						Keys: map[string]int{"email": 1},
					},
				},
			},
		},
	}

	assert.Equal(t, "testdb", dbInfo.Name)
	assert.Equal(t, "Cluster0", dbInfo.ClusterName)
	assert.Equal(t, int64(1024000), dbInfo.SizeOnDisk)
	assert.Len(t, dbInfo.Collections, 1)

	collection := dbInfo.Collections[0]
	assert.Equal(t, "users", collection.Name)
	assert.Equal(t, int64(1000), collection.DocumentCount)
	assert.Equal(t, int64(512000), collection.StorageSize)
	assert.Equal(t, 3, collection.IndexCount)
	assert.Len(t, collection.Indexes, 2)

	idIndex := collection.Indexes[0]
	assert.Equal(t, "_id_", idIndex.Name)
	assert.True(t, idIndex.Unique)

	emailIndex := collection.Indexes[1]
	assert.Equal(t, "email_1", emailIndex.Name)
	assert.False(t, emailIndex.Unique)
}

// TestDiscoverDatabases tests database discovery functionality
func TestDiscoverDatabases(t *testing.T) {
	// Skip this test if we're in unit test mode without real cluster access
	if os.Getenv("ATLAS_PROJECT_ID") == "" {
		t.Skip("Skipping database discovery test - no real Atlas project configured")
	}

	clusters := []types.ClusterManifest{
		{
			Metadata: types.ResourceMetadata{
				Name: "MockCluster0",
				Labels: map[string]string{
					"atlas.mongodb.com/project-id": "507f1f77bcf86cd799439011",
				},
			},
			Spec: types.ClusterSpec{
				ClusterType: "REPLICASET",
			},
		},
	}

	opts := &DiscoverOptions{
		Verbose:        false, // Disable verbose to reduce test noise
		MaxConcurrency: 1,
		Timeout:        2 * time.Second, // Shorter timeout for tests
	}

	// Test with empty clusters should return empty result
	emptyDatabases, err := discoverDatabases(context.Background(), nil, []types.ClusterManifest{}, opts)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(emptyDatabases))

	// Test with mock clusters that have connection issues (expected to fail gracefully)
	// The database enumeration will fail because these are mock clusters,
	// but it should handle the error gracefully and return empty results
	databases, err := discoverDatabases(context.Background(), nil, clusters, opts)

	// Should not return an error because we handle connection failures gracefully
	// The discoverDatabases function should catch DatabaseEnumerationError and return empty results
	assert.NoError(t, err)
	// Should return empty slice when connections fail
	assert.Equal(t, 0, len(databases))
}

// Helper function to create mock project state for testing
func createMockProjectState() *apply.ProjectState {
	return &apply.ProjectState{
		Project: &types.ProjectManifest{
			Metadata: types.ResourceMetadata{Name: "test-project"},
			Spec: types.ProjectConfig{
				Name:           "test-project",
				OrganizationID: "org123",
			},
		},
		Clusters: []types.ClusterManifest{
			{
				Metadata: types.ResourceMetadata{Name: "cluster1"},
				Spec: types.ClusterSpec{
					ClusterType: "REPLICASET",
				},
			},
			{
				Metadata: types.ResourceMetadata{Name: "cluster2"},
				Spec: types.ClusterSpec{
					ClusterType: "SHARDED",
				},
			},
		},
		DatabaseUsers: []types.DatabaseUserManifest{
			{
				Metadata: types.ResourceMetadata{Name: "user1"},
				Spec: types.DatabaseUserSpec{
					Username: "user1",
					Password: "secret1",
				},
			},
			{
				Metadata: types.ResourceMetadata{Name: "user2"},
				Spec: types.DatabaseUserSpec{
					Username: "user2",
					Password: "secret2",
				},
			},
			{
				Metadata: types.ResourceMetadata{Name: "user3"},
				Spec: types.DatabaseUserSpec{
					Username: "user3",
					Password: "secret3",
				},
			},
		},
		NetworkAccess: []types.NetworkAccessManifest{
			{
				Metadata: types.ResourceMetadata{Name: "network1"},
				Spec: types.NetworkAccessSpec{
					IPAddress: "203.0.113.0/24",
					Comment:   "Test network",
				},
			},
		},
	}
}

// Helper function to create mock discovery result for testing
func createMockDiscoveryResult() *DiscoveryResult {
	return &DiscoveryResult{
		APIVersion: "matlas.mongodb.com/v1",
		Kind:       "DiscoveredProject",
		Metadata: DiscoveryMetadata{
			Name:         "discovery-test",
			ProjectID:    "test-project",
			DiscoveredAt: "2024-01-01T00:00:00Z",
			Version:      "1.0.0",
			Fingerprint:  "abc123",
			Options:      map[string]interface{}{"verbose": true},
			Stats: DiscoveryStats{
				ClustersFound:       1,
				DatabaseUsersFound:  2,
				NetworkEntriesFound: 1,
				Duration:            30 * time.Second,
			},
			Labels: map[string]string{
				"matlas.mongodb.com/discovered-by": "matlas-cli",
				"matlas.mongodb.com/project-id":    "test-project",
			},
		},
		Project: &types.ProjectManifest{
			Metadata: types.ResourceMetadata{Name: "test-project"},
		},
		Clusters: []types.ClusterManifest{
			{
				Metadata: types.ResourceMetadata{Name: "test-cluster"},
			},
		},
		DatabaseUsers: []types.DatabaseUserManifest{
			{
				Metadata: types.ResourceMetadata{Name: "test-user1"},
				Spec: types.DatabaseUserSpec{
					Username: "test-user1",
					Password: "secret123",
				},
			},
			{
				Metadata: types.ResourceMetadata{Name: "test-user2"},
				Spec: types.DatabaseUserSpec{
					Username: "test-user2",
					Password: "secret456",
				},
			},
		},
		NetworkAccess: []types.NetworkAccessManifest{
			{
				Metadata: types.ResourceMetadata{Name: "test-network"},
				Spec: types.NetworkAccessSpec{
					IPAddress: "203.0.113.0/24",
				},
			},
		},
	}
}

// TestCommandValidation tests command validation scenarios
func TestCommandValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "valid project-id",
			args:        []string{"--project-id", "507f1f77bcf86cd799439011"},
			expectError: false,
		},
		{
			name:        "valid output format",
			args:        []string{"--project-id", "507f1f77bcf86cd799439011", "--output", "json"},
			expectError: false,
		},
		{
			name:        "valid include types",
			args:        []string{"--project-id", "507f1f77bcf86cd799439011", "--include", "clusters,users"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewDiscoverCmd()
			cmd.SetArgs(tt.args)

			// Test flag parsing
			err := cmd.ParseFlags(tt.args)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDiscoveryStats tests statistics tracking
func TestDiscoveryStats(t *testing.T) {
	stats := DiscoveryStats{
		ClustersFound:       5,
		DatabaseUsersFound:  10,
		NetworkEntriesFound: 3,
		DatabasesFound:      20,
		CollectionsFound:    100,
		Duration:            2 * time.Minute,
		CacheHit:            true,
	}

	assert.Equal(t, 5, stats.ClustersFound)
	assert.Equal(t, 10, stats.DatabaseUsersFound)
	assert.Equal(t, 3, stats.NetworkEntriesFound)
	assert.Equal(t, 20, stats.DatabasesFound)
	assert.Equal(t, 100, stats.CollectionsFound)
	assert.Equal(t, 2*time.Minute, stats.Duration)
	assert.True(t, stats.CacheHit)
}

// BenchmarkShouldIncludeType benchmarks the include type checking
func BenchmarkShouldIncludeType(b *testing.B) {
	opts := &DiscoverOptions{
		IncludeTypes: []string{"clusters", "users", "network", "databases", "projects"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shouldIncludeType("clusters", opts)
	}
}

// BenchmarkApplyFiltering benchmarks the filtering logic
func BenchmarkApplyFiltering(b *testing.B) {
	projectState := createMockProjectState()
	opts := &DiscoverOptions{
		IncludeTypes: []string{"clusters", "users"},
		ExcludeTypes: []string{"network"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := &DiscoveryResult{}
		applyFiltering(result, projectState, opts)
	}
}

// TestDiscoveryMetadataValidation tests metadata validation
func TestDiscoveryMetadataValidation(t *testing.T) {
	metadata := DiscoveryMetadata{
		Name:         "discovery-test",
		ProjectID:    "507f1f77bcf86cd799439011",
		DiscoveredAt: time.Now().UTC().Format(time.RFC3339),
		Version:      "1.0.0",
		Fingerprint:  "abc123def456",
		Options:      map[string]interface{}{"test": true},
		Stats: DiscoveryStats{
			ClustersFound: 1,
			Duration:      30 * time.Second,
		},
		Labels: map[string]string{
			"matlas.mongodb.com/discovered-by": "matlas-cli",
		},
	}

	// Basic validation tests
	assert.NotEmpty(t, metadata.Name)
	assert.NotEmpty(t, metadata.ProjectID)
	assert.NotEmpty(t, metadata.DiscoveredAt)
	assert.NotEmpty(t, metadata.Version)
	assert.NotEmpty(t, metadata.Fingerprint)
	assert.NotNil(t, metadata.Options)
	assert.NotEmpty(t, metadata.Labels)

	// Test timestamp parsing
	_, err := time.Parse(time.RFC3339, metadata.DiscoveredAt)
	assert.NoError(t, err, "DiscoveredAt should be valid RFC3339 timestamp")

	// Test project ID format (should be 24 character hex string for MongoDB ObjectId)
	assert.Len(t, metadata.ProjectID, 24, "ProjectID should be 24 characters for MongoDB ObjectId")
}
