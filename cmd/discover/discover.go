package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/cli"
	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/config"
	atlasservice "github.com/teabranch/matlas-cli/internal/services/atlas"
	dbservice "github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/validation"
)

// DiscoverOptions contains the options for the discover command
type DiscoverOptions struct {
	ProjectID        string
	OutputFormat     string
	OutputFile       string
	IncludeTypes     []string
	ExcludeTypes     []string
	MaskSecrets      bool
	IncludeDatabases bool
	NoCache          bool
	Timeout          time.Duration
	Verbose          bool
	Parallel         bool
	MaxConcurrency   int
	// New fields for resource-specific discovery
	ResourceType string
	ResourceName string
	// Conversion options
	ConvertToApply bool
	// Database enumeration connection overrides
	MongoURI         string
	MongoUsername    string
	MongoPassword    string
	UseTempUser      bool
	TempUserDatabase string
	// Cache visibility
	ShowCacheStats bool
}

// DiscoveryResult represents the complete discovery output
type DiscoveryResult struct {
	APIVersion    string                        `yaml:"apiVersion" json:"apiVersion"`
	Kind          string                        `yaml:"kind" json:"kind"`
	Metadata      DiscoveryMetadata             `yaml:"metadata" json:"metadata"`
	Project       *types.ProjectManifest        `yaml:"project,omitempty" json:"project,omitempty"`
	Clusters      []types.ClusterManifest       `yaml:"clusters,omitempty" json:"clusters,omitempty"`
	DatabaseUsers []types.DatabaseUserManifest  `yaml:"databaseUsers,omitempty" json:"databaseUsers,omitempty"`
	NetworkAccess []types.NetworkAccessManifest `yaml:"networkAccess,omitempty" json:"networkAccess,omitempty"`
	Databases     []DatabaseInfo                `yaml:"databases,omitempty" json:"databases,omitempty"`
}

// DiscoveryMetadata contains metadata about the discovery operation
type DiscoveryMetadata struct {
	Name         string            `yaml:"name" json:"name"`
	ProjectID    string            `yaml:"projectId" json:"projectId"`
	DiscoveredAt string            `yaml:"discoveredAt" json:"discoveredAt"`
	Version      string            `yaml:"version" json:"version"`
	Fingerprint  string            `yaml:"fingerprint" json:"fingerprint"`
	Options      interface{}       `yaml:"options" json:"options"`
	Stats        DiscoveryStats    `yaml:"stats" json:"stats"`
	Labels       map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// DiscoveryStats contains statistics about the discovery operation
type DiscoveryStats struct {
	ClustersFound       int           `yaml:"clustersFound" json:"clustersFound"`
	DatabaseUsersFound  int           `yaml:"databaseUsersFound" json:"databaseUsersFound"`
	NetworkEntriesFound int           `yaml:"networkEntriesFound" json:"networkEntriesFound"`
	DatabasesFound      int           `yaml:"databasesFound" json:"databasesFound"`
	CollectionsFound    int           `yaml:"collectionsFound" json:"collectionsFound"`
	Duration            time.Duration `yaml:"duration" json:"duration"`
	CacheHit            bool          `yaml:"cacheHit" json:"cacheHit"`
}

// DatabaseInfo contains information about discovered databases
type DatabaseInfo struct {
	Name        string           `yaml:"name" json:"name"`
	ClusterName string           `yaml:"clusterName" json:"clusterName"`
	SizeOnDisk  int64            `yaml:"sizeOnDisk,omitempty" json:"sizeOnDisk,omitempty"`
	Collections []CollectionInfo `yaml:"collections,omitempty" json:"collections,omitempty"`
	Indexes     []IndexInfo      `yaml:"indexes,omitempty" json:"indexes,omitempty"`
}

// CollectionInfo contains information about discovered collections
type CollectionInfo struct {
	Name           string      `yaml:"name" json:"name"`
	DocumentCount  int64       `yaml:"documentCount,omitempty" json:"documentCount,omitempty"`
	StorageSize    int64       `yaml:"storageSize,omitempty" json:"storageSize,omitempty"`
	IndexCount     int         `yaml:"indexCount,omitempty" json:"indexCount,omitempty"`
	Indexes        []IndexInfo `yaml:"indexes,omitempty" json:"indexes,omitempty"`
	ShardKey       interface{} `yaml:"shardKey,omitempty" json:"shardKey,omitempty"`
	ValidationRule interface{} `yaml:"validationRule,omitempty" json:"validationRule,omitempty"`
}

// IndexInfo contains information about discovered indexes
type IndexInfo struct {
	Name          string      `yaml:"name" json:"name"`
	Keys          interface{} `yaml:"keys" json:"keys"`
	Unique        bool        `yaml:"unique,omitempty" json:"unique,omitempty"`
	Sparse        bool        `yaml:"sparse,omitempty" json:"sparse,omitempty"`
	TTL           *int        `yaml:"ttl,omitempty" json:"ttl,omitempty"`
	PartialFilter interface{} `yaml:"partialFilter,omitempty" json:"partialFilter,omitempty"`
}

// NewDiscoverCmd creates the discover command for project discovery
func NewDiscoverCmd() *cobra.Command {
	opts := &DiscoverOptions{}

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover Atlas project configuration",
		Long: `Discover and export the complete configuration of an Atlas project.

This command connects to Atlas and enumerates all resources in a project,
including clusters, database users, network access entries, and optionally
databases and collections. The output can be saved as YAML or JSON configuration
files that can be used with the apply command.

You can also discover specific individual resources by specifying --resource-type
and --resource-name flags. This outputs the resource in its individual manifest
format with the appropriate API version.

Use --convert-to-apply to automatically convert the discovered project into
ApplyDocument format that can be directly applied with 'matlas infra'.`,
		SilenceUsage: true,
		Example: `  # Discover all resources in a project
  matlas discover --project-id 507f1f77bcf86cd799439011

  # Discover and save to file
  matlas discover --project-id 507f1f77bcf86cd799439011 -o project.yaml

  # Discover with database enumeration
  matlas discover --project-id 507f1f77bcf86cd799439011 --include-databases

  # Discover with database enumeration using a temporary Atlas DB user (recommended)
  matlas discover --project-id 507f1f77bcf86cd799439011 --include-databases --use-temp-user

  # Discover specific resource types only
  matlas discover --project-id 507f1f77bcf86cd799439011 --include clusters,users

  # Discover and mask sensitive information
  matlas discover --project-id 507f1f77bcf86cd799439011 --mask-secrets

  # Discover with JSON output
  matlas discover --project-id 507f1f77bcf86cd799439011 --output json

  # Discover and pipe to apply validate
  matlas discover --project-id 507f1f77bcf86cd799439011 | matlas apply validate -f -

  # Convert discovered project to ApplyDocument format for direct application
  matlas discover --project-id 507f1f77bcf86cd799439011 --convert-to-apply
  
  # Convert and save as ApplyDocument
  matlas discover --project-id 507f1f77bcf86cd799439011 --convert-to-apply -o apply-ready.yaml
  
  # Convert and immediately apply (workflow)
  matlas discover --project-id 507f1f77bcf86cd799439011 --convert-to-apply | matlas infra -f -

  # Discover a specific cluster
  matlas discover --project-id 507f1f77bcf86cd799439011 --resource-type cluster --resource-name my-cluster

  # Discover a specific database user
  matlas discover --project-id 507f1f77bcf86cd799439011 --resource-type user --resource-name app-user

  # Discover a specific network access entry
  matlas discover --project-id 507f1f77bcf86cd799439011 --resource-type network --resource-name office-access`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiscover(cmd, opts)
		},
	}

	// Core options
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Atlas project ID to discover (required)")

	// Output options
	cmd.Flags().StringVar(&opts.OutputFormat, "output", "yaml", "Output format: yaml, json")
	cmd.Flags().StringVarP(&opts.OutputFile, "output-file", "o", "", "File to write output to (default: stdout)")
	cmd.Flags().BoolVar(&opts.ConvertToApply, "convert-to-apply", false, "Convert discovered project to ApplyDocument format for direct application")

	// Filtering options
	cmd.Flags().StringSliceVar(&opts.IncludeTypes, "include", []string{}, "Resource types to include: project,clusters,users,network,databases")
	cmd.Flags().StringSliceVar(&opts.ExcludeTypes, "exclude", []string{}, "Resource types to exclude: project,clusters,users,network,databases")

	// Resource-specific discovery options
	cmd.Flags().StringVar(&opts.ResourceType, "resource-type", "", "Discover a specific resource type (cluster, user, network)")
	cmd.Flags().StringVar(&opts.ResourceName, "resource-name", "", "Name of the specific resource to discover (requires --resource-type)")

	// Discovery options
	cmd.Flags().BoolVar(&opts.MaskSecrets, "mask-secrets", false, "Hide sensitive information like passwords")
	cmd.Flags().BoolVar(&opts.IncludeDatabases, "include-databases", false, "Include database and collection enumeration")
	cmd.Flags().BoolVar(&opts.NoCache, "no-cache", false, "Disable discovery caching")
	cmd.Flags().BoolVar(&opts.Parallel, "parallel", true, "Enable parallel discovery of resources")
	cmd.Flags().IntVar(&opts.MaxConcurrency, "max-concurrency", 5, "Maximum concurrent API calls")
	// Database enumeration connection controls
	cmd.Flags().StringVar(&opts.MongoURI, "mongo-uri", "", "MongoDB connection URI for database enumeration (overrides cluster connection strings). Can also be set via MONGODB_URI env var")
	cmd.Flags().StringVar(&opts.MongoUsername, "mongo-username", "", "Username for MongoDB when connection string lacks credentials. Can also be set via MONGODB_USERNAME env var")
	cmd.Flags().StringVar(&opts.MongoPassword, "mongo-password", "", "Password for MongoDB when connection string lacks credentials. Can also be set via MONGODB_PASSWORD env var")
	cmd.Flags().BoolVar(&opts.UseTempUser, "use-temp-user", false, "Create a short-lived Atlas DB user for database enumeration and inject credentials automatically")
	cmd.Flags().StringVar(&opts.TempUserDatabase, "temp-user-database", "", "Database name for temporary user role scope (default: admin; specify to grant least privilege)")
	// Cache metrics
	cmd.Flags().BoolVar(&opts.ShowCacheStats, "cache-stats", false, "Show discovery cache statistics on stderr")

	// Runtime options
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 10*time.Minute, "Timeout for discovery operation")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")

	return cmd
}

// runDiscover executes the discovery operation
func runDiscover(cmd *cobra.Command, opts *DiscoverOptions) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	// Load configuration
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate project ID
	if err := validation.ValidateProjectID(opts.ProjectID); err != nil {
		return cli.FormatValidationError("project-id", opts.ProjectID, err.Error())
	}

	// Validate resource-specific discovery options
	if err := validateResourceDiscoveryOptions(opts); err != nil {
		return err
	}

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		return fmt.Errorf("failed to create Atlas client: %w", err)
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Starting discovery for project %s...\n", opts.ProjectID)
	}

	// Create discovery service
	baseDiscovery := apply.NewAtlasStateDiscovery(client)
	var discovery apply.StateDiscovery = baseDiscovery

	// Add caching if enabled
	var cache *apply.InMemoryStateCache
	if !opts.NoCache {
		cache = apply.NewInMemoryStateCache(100, 1*time.Hour) // max 100 entries, 1 hour TTL
		discovery = apply.NewCachedStateDiscovery(baseDiscovery, cache)
	}

	// Env fallbacks for DB enumeration
	if opts.MongoURI == "" {
		opts.MongoURI = os.Getenv("MONGODB_URI")
	}
	if opts.MongoUsername == "" {
		opts.MongoUsername = os.Getenv("MONGODB_USERNAME")
	}
	if opts.MongoPassword == "" {
		opts.MongoPassword = os.Getenv("MONGODB_PASSWORD")
	}

	startTime := time.Now()

	// Discover project state
	projectState, err := discovery.DiscoverProject(ctx, opts.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to discover project: %w", err)
	}

	discoveryDuration := time.Since(startTime)

	// Handle resource-specific discovery if requested
	if opts.ResourceType != "" {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Searching for %s '%s'...\n", opts.ResourceType, opts.ResourceName)
		}

		// Find the specific resource
		resource, err := findSpecificResource(projectState, opts.ResourceType, opts.ResourceName)
		if err != nil {
			return err
		}

		// Mask secrets if requested
		if opts.MaskSecrets {
			maskResourceSecrets(resource)
		}

		// Output the individual resource manifest
		if err := outputResourceManifest(resource, opts); err != nil {
			return fmt.Errorf("failed to output resource manifest: %w", err)
		}

		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Successfully discovered %s '%s'\n", opts.ResourceType, opts.ResourceName)
		}

		return nil
	}

	// Create discovery result
	result := &DiscoveryResult{
		APIVersion: "matlas.mongodb.com/v1",
		Kind:       "DiscoveredProject",
		Metadata: DiscoveryMetadata{
			Name:         fmt.Sprintf("discovery-%s", opts.ProjectID),
			ProjectID:    opts.ProjectID,
			DiscoveredAt: time.Now().UTC().Format(time.RFC3339),
			Version:      "1.0.0",
			Fingerprint:  projectState.Fingerprint,
			Options:      *opts,
			Stats: DiscoveryStats{
				ClustersFound:       len(projectState.Clusters),
				DatabaseUsersFound:  len(projectState.DatabaseUsers),
				NetworkEntriesFound: len(projectState.NetworkAccess),
				Duration:            discoveryDuration,
			},
			Labels: map[string]string{
				"matlas.mongodb.com/discovered-by": "matlas-cli",
				"matlas.mongodb.com/project-id":    opts.ProjectID,
			},
		},
	}

	// Apply filtering
	if err := applyFiltering(result, projectState, opts); err != nil {
		return fmt.Errorf("failed to apply filtering: %w", err)
	}

	// Discover databases if requested
	if opts.IncludeDatabases && shouldIncludeType("databases", opts) {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Discovering databases and collections (experimental)...\n")
		}

		// Optionally create a temporary user for enumeration and set credentials
		var cleanupTemp func()
		if opts.UseTempUser {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Creating temporary database user for enumeration...\n")
			}
			tempCleanup, err := createTempUserForEnumeration(ctx, client, opts.ProjectID, projectState.Clusters, opts)
			if err != nil {
				if opts.Verbose {
					fmt.Fprintf(os.Stderr, "Warning: Failed to create temporary user for enumeration: %v\n", err)
				}
			} else {
				cleanupTemp = tempCleanup
			}
		}

		databases, err := discoverDatabases(ctx, client, projectState.Clusters, opts)
		if err != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to discover databases: %v\n", err)
			}
		} else {
			result.Databases = databases
			result.Metadata.Stats.DatabasesFound = len(databases)

			// Count total collections
			totalCollections := 0
			for _, db := range databases {
				totalCollections += len(db.Collections)
			}
			result.Metadata.Stats.CollectionsFound = totalCollections
		}

		// Cleanup temporary user if created
		if cleanupTemp != nil {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Cleaning up temporary user...\n")
			}
			cleanupTemp()
		}
	}

	// Mask secrets if requested
	if opts.MaskSecrets {
		maskSecrets(result)
	}

	// Output result
	if err := outputResult(result, opts); err != nil {
		return fmt.Errorf("failed to output result: %w", err)
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "Discovery completed in %v\n", discoveryDuration)
		fmt.Fprintf(os.Stderr, "Found: %d clusters, %d users, %d network entries",
			result.Metadata.Stats.ClustersFound,
			result.Metadata.Stats.DatabaseUsersFound,
			result.Metadata.Stats.NetworkEntriesFound)
		if result.Metadata.Stats.DatabasesFound > 0 {
			fmt.Fprintf(os.Stderr, ", %d databases, %d collections",
				result.Metadata.Stats.DatabasesFound,
				result.Metadata.Stats.CollectionsFound)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Optionally display cache statistics
	if opts.ShowCacheStats && cache != nil {
		stats := cache.Stats()
		fmt.Fprintf(os.Stderr, "Cache stats: size=%d hits=%d misses=%d hitRate=%.2f evictions=%d expires=%d\n",
			stats.Size, stats.HitCount, stats.MissCount, stats.HitRate, stats.EvictCount, stats.ExpireCount)
	}

	return nil
}

// applyFiltering applies include/exclude filtering to the discovery result
func applyFiltering(result *DiscoveryResult, projectState *apply.ProjectState, opts *DiscoverOptions) error {
	// If no includes specified, include everything by default
	if len(opts.IncludeTypes) == 0 {
		result.Project = projectState.Project
		result.Clusters = projectState.Clusters
		result.DatabaseUsers = projectState.DatabaseUsers
		result.NetworkAccess = projectState.NetworkAccess
	} else {
		// Only include specified types
		if shouldIncludeType("project", opts) {
			result.Project = projectState.Project
		}
		if shouldIncludeType("clusters", opts) {
			result.Clusters = projectState.Clusters
		}
		if shouldIncludeType("users", opts) {
			result.DatabaseUsers = projectState.DatabaseUsers
		}
		if shouldIncludeType("network", opts) {
			result.NetworkAccess = projectState.NetworkAccess
		}
	}

	// Apply exclusions
	if shouldExcludeType("project", opts) {
		result.Project = nil
	}
	if shouldExcludeType("clusters", opts) {
		result.Clusters = nil
	}
	if shouldExcludeType("users", opts) {
		result.DatabaseUsers = nil
	}
	if shouldExcludeType("network", opts) {
		result.NetworkAccess = nil
	}

	return nil
}

// shouldIncludeType checks if a resource type should be included
func shouldIncludeType(resourceType string, opts *DiscoverOptions) bool {
	if len(opts.IncludeTypes) == 0 {
		return true // Include all by default if no includes specified
	}

	for _, t := range opts.IncludeTypes {
		if strings.EqualFold(t, resourceType) {
			return true
		}
	}
	return false
}

// shouldExcludeType checks if a resource type should be excluded
func shouldExcludeType(resourceType string, opts *DiscoverOptions) bool {
	for _, t := range opts.ExcludeTypes {
		if strings.EqualFold(t, resourceType) {
			return true
		}
	}
	return false
}

// discoverDatabases discovers databases and collections from Atlas clusters
func discoverDatabases(ctx context.Context, client *atlasclient.Client, clusters []types.ClusterManifest, opts *DiscoverOptions) ([]DatabaseInfo, error) {
	if len(clusters) == 0 {
		return []DatabaseInfo{}, nil
	}

	// Create database enumerator
	enumerator := NewDatabaseEnumerator(client, opts)

	// Enumerate databases for all clusters
	databases, err := enumerator.EnumerateClusterDatabases(ctx, clusters)
	if err != nil {
		// Check if it's a DatabaseEnumerationError (partial failure)
		if dbErr, ok := err.(*DatabaseEnumerationError); ok {
			if opts.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: Database enumeration completed with %d errors\n", len(dbErr.ClusterErrors))
			}
			// Return partial results even with errors
			return databases, nil
		}
		// For other errors, return the error
		return databases, err
	}

	return databases, nil
}

// maskSecrets removes sensitive information from the discovery result
func maskSecrets(result *DiscoveryResult) {
	// Mask passwords in database users
	for i := range result.DatabaseUsers {
		result.DatabaseUsers[i].Spec.Password = "***MASKED***"
	}

	// Could also mask connection strings, API keys, etc.
	// Add more masking logic as needed
}

// maskResourceSecrets removes sensitive information from individual resource manifests
func maskResourceSecrets(resource interface{}) {
	switch r := resource.(type) {
	case *types.DatabaseUserManifest:
		r.Spec.Password = "***MASKED***"
		// Other resource types don't currently have sensitive fields to mask
		// but this can be extended as needed
	}
}

// outputResult outputs the discovery result in the specified format
func outputResult(result *DiscoveryResult, opts *DiscoverOptions) error {
	var output io.Writer = os.Stdout

	// Open output file if specified
	if opts.OutputFile != "" {
		file, err := os.Create(opts.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", err)
			}
		}()
		output = file
	}

	// Convert to ApplyDocument format if requested
	var outputData interface{} = result
	if opts.ConvertToApply {
		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Converting DiscoveredProject to ApplyDocument format...\n")
		}

		// Convert struct to map[string]interface{} that the converter expects
		resultBytes, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal DiscoveryResult for conversion: %w", err)
		}

		var resultMap map[string]interface{}
		if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
			return fmt.Errorf("failed to unmarshal DiscoveryResult to map for conversion: %w", err)
		}

		converter := apply.NewDiscoveredProjectConverter()
		applyDoc, err := converter.ConvertToApplyDocument(resultMap)
		if err != nil {
			return fmt.Errorf("failed to convert to ApplyDocument format: %w", err)
		}
		outputData = applyDoc

		if opts.Verbose {
			fmt.Fprintf(os.Stderr, "Conversion completed. Ready for 'matlas infra' apply.\n")
		}
	}

	// Format and write output
	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		return encoder.Encode(outputData)

	case "yaml", "yml":
		encoder := yaml.NewEncoder(output)
		encoder.SetIndent(2)
		return encoder.Encode(outputData)

	default:
		return fmt.Errorf("unsupported output format: %s (supported: yaml, json)", opts.OutputFormat)
	}
}

// outputResourceManifest outputs a single resource manifest in the specified format
func outputResourceManifest(manifest interface{}, opts *DiscoverOptions) error {
	var output io.Writer = os.Stdout

	// Open output file if specified
	if opts.OutputFile != "" {
		file, err := os.Create(opts.OutputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", err)
			}
		}()
		output = file
	}

	// Format and write output
	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		return encoder.Encode(manifest)

	case "yaml", "yml":
		encoder := yaml.NewEncoder(output)
		encoder.SetIndent(2)
		return encoder.Encode(manifest)

	default:
		return fmt.Errorf("unsupported output format: %s (supported: yaml, json)", opts.OutputFormat)
	}
}

// findSpecificResource finds a specific resource by name and type from the discovered project state
func findSpecificResource(projectState *apply.ProjectState, resourceType, resourceName string) (interface{}, error) {
	switch strings.ToLower(resourceType) {
	case "cluster":
		for _, cluster := range projectState.Clusters {
			if cluster.Metadata.Name == resourceName {
				return cluster, nil
			}
		}
		return nil, fmt.Errorf("cluster '%s' not found in project", resourceName)

	case "user", "databaseuser":
		for i, user := range projectState.DatabaseUsers {
			if user.Metadata.Name == resourceName {
				return &projectState.DatabaseUsers[i], nil
			}
		}
		return nil, fmt.Errorf("database user '%s' not found in project", resourceName)

	case "network", "networkaccess":
		for _, network := range projectState.NetworkAccess {
			if network.Metadata.Name == resourceName {
				return network, nil
			}
		}
		return nil, fmt.Errorf("network access entry '%s' not found in project", resourceName)

	case "project":
		if projectState.Project != nil && projectState.Project.Metadata.Name == resourceName {
			return projectState.Project, nil
		}
		return nil, fmt.Errorf("project '%s' not found", resourceName)

	default:
		return nil, fmt.Errorf("unsupported resource type: %s (supported: cluster, user, network, project)", resourceType)
	}
}

// validateResourceDiscoveryOptions validates the resource-specific discovery options
func validateResourceDiscoveryOptions(opts *DiscoverOptions) error {
	// If resource-type is specified, resource-name must also be specified
	if opts.ResourceType != "" && opts.ResourceName == "" {
		return fmt.Errorf("--resource-name is required when --resource-type is specified")
	}

	// If resource-name is specified without resource-type, show helpful error
	if opts.ResourceName != "" && opts.ResourceType == "" {
		return fmt.Errorf("--resource-type is required when --resource-name is specified")
	}

	// Validate resource type if specified
	if opts.ResourceType != "" {
		validTypes := []string{"cluster", "user", "databaseuser", "network", "networkaccess", "project"}
		validType := false
		normalizedType := strings.ToLower(opts.ResourceType)

		for _, vt := range validTypes {
			if normalizedType == vt {
				validType = true
				break
			}
		}

		if !validType {
			return fmt.Errorf("invalid resource type '%s'. Supported types: cluster, user, network, project", opts.ResourceType)
		}
	}

	// Resource-specific discovery is incompatible with certain options
	if opts.ResourceType != "" {
		if len(opts.IncludeTypes) > 0 {
			return fmt.Errorf("--include cannot be used with resource-specific discovery (--resource-type)")
		}
		if len(opts.ExcludeTypes) > 0 {
			return fmt.Errorf("--exclude cannot be used with resource-specific discovery (--resource-type)")
		}
		if opts.IncludeDatabases {
			return fmt.Errorf("--include-databases cannot be used with resource-specific discovery (--resource-type)")
		}
	}

	return nil
}

// createTempUserForEnumeration creates a scoped temporary Atlas DB user across clusters and injects creds into opts
func createTempUserForEnumeration(ctx context.Context, client *atlasclient.Client, projectID string, clusters []types.ClusterManifest, opts *DiscoverOptions) (func(), error) {
	// No clusters, nothing to do
	if len(clusters) == 0 {
		return nil, nil
	}

	usersSvc := atlasservice.NewDatabaseUsersService(client)
	manager := dbservice.NewTempUserManager(usersSvc, projectID)

	// Collect cluster names
	clusterNames := make([]string, 0, len(clusters))
	for _, c := range clusters {
		clusterNames = append(clusterNames, c.Metadata.Name)
	}

	// Create temporary user
	temp, err := manager.CreateTempUserForDiscovery(ctx, clusterNames, opts.TempUserDatabase)
	if err != nil {
		return nil, err
	}

	// Inject credentials into options for enumeration path
	opts.MongoUsername = temp.Username
	opts.MongoPassword = temp.Password

	// Provide cleanup
	cleanup := func() {
		// Best effort cleanup with a short context
		cctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = temp.CleanupFunc(cctx)
	}

	// Allow propagation
	time.Sleep(6 * time.Second)

	return cleanup, nil
}
