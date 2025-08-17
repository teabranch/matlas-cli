package infra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	admin "go.mongodb.org/atlas-sdk/v20250312005/admin"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/services/database"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/ui"
)

// ApplyOptions contains the options for the apply command
type ApplyOptions struct {
	Files            []string
	DryRun           bool
	DryRunMode       string
	OutputFormat     string
	AutoApprove      bool
	Timeout          time.Duration
	Verbose          bool
	NoColor          bool
	ProjectID        string
	StrictEnv        bool
	Watch            bool
	WatchInterval    time.Duration
	PreserveExisting bool
}

// NewInfraCmd creates the infra command for declarative configuration
func NewInfraCmd() *cobra.Command {
	opts := &ApplyOptions{}

	cmd := &cobra.Command{
		Use:   "infra",
		Short: "Manage infrastructure with declarative configuration",
		Long: `Apply declarative configuration files to manage Atlas resources.

This command reads configuration files and applies the desired state to Atlas resources.
It supports dry-run mode to preview changes before applying them.`,
		SilenceUsage: true,
		Example: `  # Apply configuration from a file
  matlas infra -f config.yaml

  # Dry run to see what changes would be made
  matlas infra -f config.yaml --dry-run

  # Apply multiple files with glob pattern
  matlas infra -f "configs/*.yaml"

  # Apply configuration from stdin
  cat config.yaml | matlas infra -f -

  # Only add new resources, preserve existing ones
  matlas infra -f config.yaml --preserve-existing

  # Apply with thorough validation
  matlas infra -f config.yaml --dry-run --dry-run-mode thorough

  # Apply with JSON output
  matlas infra -f config.yaml --dry-run --output json

  # Watch mode for continuous reconciliation
  matlas infra -f config.yaml --watch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support positional arguments as files if no --file flag provided
			if len(opts.Files) == 0 && len(args) > 0 {
				opts.Files = args
			}
			return runApply(cmd, opts)
		},
	}

	// File input flags
	cmd.Flags().StringSliceVarP(&opts.Files, "file", "f", []string{}, "Configuration files to apply (supports glob patterns and stdin with '-')")

	// Dry run flags
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be applied without making changes")
	cmd.Flags().StringVar(&opts.DryRunMode, "dry-run-mode", "quick", "Dry run validation mode: quick, thorough, detailed")

	// Output and behavior flags
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "table", "Output format: table, json, yaml, summary, detailed")
	cmd.Flags().BoolVar(&opts.AutoApprove, "auto-approve", false, "Skip interactive approval prompts")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 30*time.Minute, "Timeout for the apply operation")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")

	// Project context
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Atlas project ID (overrides config)")

	// Template processing flags
	cmd.Flags().BoolVar(&opts.StrictEnv, "strict-env", false, "Fail on undefined environment variables")

	// Safety flags
	cmd.Flags().BoolVar(&opts.PreserveExisting, "preserve-existing", false, "Only add new resources, never delete existing ones")

	// Watch mode flags
	cmd.Flags().BoolVar(&opts.Watch, "watch", false, "Enable watch mode for continuous reconciliation")
	cmd.Flags().DurationVar(&opts.WatchInterval, "watch-interval", 5*time.Minute, "Interval between reconciliation checks in watch mode")

	// Add subcommands
	cmd.AddCommand(NewValidateCmd())
	cmd.AddCommand(NewPlanCmd())
	cmd.AddCommand(NewDiffCmd())
	cmd.AddCommand(NewShowCmd())
	cmd.AddCommand(NewDestroyCmd())

	return cmd
}

func runApply(cmd *cobra.Command, opts *ApplyOptions) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	// Validate options
	if err := validateApplyOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Expand file patterns and handle stdin
	files, err := expandFilePatterns(opts.Files)
	if err != nil {
		return fmt.Errorf("failed to expand file patterns: %w", err)
	}

	// Load and parse configuration files first
	configs, err := loadConfigurations(files, opts)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	// For dry-run mode, we can proceed without services
	if opts.DryRun {
		return performDryRunOnly(ctx, configs, opts)
	}

	// For actual apply/watch, we need services and config
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	services, err := initializeServices(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Handle watch mode
	if opts.Watch {
		return runWatchMode(ctx, configs, services, cfg, opts)
	}

	// Perform single apply operation
	return performApply(ctx, configs, services, cfg, opts)
}

func loadConfigurations(files []string, opts *ApplyOptions) ([]*apply.LoadResult, error) {
	// Initialize configuration loader
	loaderOpts := &apply.LoaderOptions{
		StrictEnv:    opts.StrictEnv,
		Debug:        opts.Verbose,
		CacheEnabled: true,
		AllowStdin:   true,
		MaxFileSize:  10 * 1024 * 1024, // 10MB
	}

	loader := apply.NewConfigurationLoader(loaderOpts)

	var configs []*apply.LoadResult

	for _, file := range files {
		result, err := loader.LoadApplyConfig(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", file, err)
		}

		configs = append(configs, result)

		// Report any warnings
		if len(result.Warnings) > 0 && opts.Verbose {
			for _, warning := range result.Warnings {
				fmt.Fprintf(os.Stderr, "Warning in %s: %s\n", file, warning.Message)
			}
		}

		// Report any errors
		if len(result.Errors) > 0 {
			for _, errMsg := range result.Errors {
				fmt.Fprintf(os.Stderr, "Error in %s: %s\n", file, errMsg.Message)
			}
			return nil, fmt.Errorf("configuration errors found in %s", file)
		}
	}

	return configs, nil
}

type ServiceClients struct {
	ClustersService      *atlas.ClustersService
	UsersService         *atlas.DatabaseUsersService
	NetworkAccessService *atlas.NetworkAccessListsService
	ProjectsService      *atlas.ProjectsService
	DatabaseService      *database.Service
}

func initializeServices(cfg *config.Config) (*ServiceClients, error) {
	// Create Atlas client
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Atlas client: %w", err)
	}

	// Initialize Atlas services
	clustersService := atlas.NewClustersService(atlasClient)
	usersService := atlas.NewDatabaseUsersService(atlasClient)
	networkAccessService := atlas.NewNetworkAccessListsService(atlasClient)
	projectsService := atlas.NewProjectsService(atlasClient)

	// Initialize database service with standardized logger
	logger := logging.Default()
	databaseService := database.NewService(logger)

	return &ServiceClients{
		ClustersService:      clustersService,
		UsersService:         usersService,
		NetworkAccessService: networkAccessService,
		ProjectsService:      projectsService,
		DatabaseService:      databaseService,
	}, nil
}

func performApply(ctx context.Context, configs []*apply.LoadResult, services *ServiceClients, cfg *config.Config, opts *ApplyOptions) error {
	// Build desired state from configurations first
	desiredState, err := buildDesiredState(configs)
	if err != nil {
		return fmt.Errorf("failed to build desired state: %w", err)
	}

	projectNameOrID := getProjectID(configs, opts)

	// Extract organization ID from configuration for project resolution
	orgID := getOrganizationID(configs)

	// Resolve project name to project ID
	resolvedProjectID := projectNameOrID
	if projectNameOrID != "" {
		// For dry-run mode, we don't need to resolve the project ID
		if !opts.DryRun {
			resolvedID, err := resolveProjectID(ctx, projectNameOrID, services.ProjectsService, orgID)
			if err != nil {
				return fmt.Errorf("failed to resolve project ID for '%s': %w", projectNameOrID, err)
			}
			resolvedProjectID = resolvedID
		}
	}

	// For dry-run mode, create a simplified plan without Atlas discovery
	if opts.DryRun {
		return performDryRun(ctx, desiredState, resolvedProjectID, opts)
	}

	// For actual apply, we need Atlas client and discovery
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		return fmt.Errorf("failed to create Atlas client for discovery: %w", err)
	}

	// Initialize apply engine components
	discoveryService := apply.NewAtlasStateDiscovery(atlasClient)

	diffEngine := apply.NewDiffEngine()
	diffEngine.PreserveExisting = opts.PreserveExisting

	planOptimizer := apply.NewPlanOptimizer()

	// Create enhanced executor
	// Propagate preserve-existing intent into executor config for typed conflict handling
	enhancedCfg := apply.DefaultEnhancedExecutorConfig()
	enhancedCfg.BaseConfig.PreserveExisting = opts.PreserveExisting

	enhancedExecutor := apply.NewEnhancedExecutor(
		services.ClustersService,
		services.UsersService,
		services.NetworkAccessService,
		services.ProjectsService,
		services.DatabaseService,
		enhancedCfg,
	)

	// Discover current state
	if opts.Verbose {
		fmt.Printf("Discovering current state for project %s (resolved from '%s')...\n", resolvedProjectID, projectNameOrID)
	}

	currentState, err := discoveryService.DiscoverProject(ctx, resolvedProjectID)
	if err != nil {
		return fmt.Errorf("failed to discover current state: %w", err)
	}

	// Compute diff
	if opts.Verbose {
		fmt.Println("Computing differences...")
	}

	diff, err := diffEngine.ComputeProjectDiff(desiredState, currentState)
	if err != nil {
		return fmt.Errorf("failed to compute diff: %w", err)
	}

	// Create execution plan
	planBuilder := apply.NewPlanBuilder(resolvedProjectID)

	// Add operations from diff
	planBuilder.AddOperations(diff.Operations)

	plan, err := planBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to create execution plan: %w", err)
	}

	// Optimize plan
	optimizationResult, err := planOptimizer.OptimizePlan(plan)
	if err != nil {
		return fmt.Errorf("failed to optimize plan: %w", err)
	}

	optimizedPlan := optimizationResult.OptimizedPlan

	// Show plan summary and get approval
	if !opts.AutoApprove && optimizedPlan.Summary.RequiresApproval {
		if err := showPlanAndGetApproval(optimizedPlan, opts); err != nil {
			return err
		}
	}

	// Execute the plan
	if opts.Verbose {
		fmt.Println("Executing plan...")
	}

	result, err := enhancedExecutor.Execute(ctx, optimizedPlan)
	if err != nil {
		return fmt.Errorf("failed to execute plan: %w", err)
	}

	// Display results
	return displayExecutionResults(result, opts)
}

func runWatchMode(ctx context.Context, configs []*apply.LoadResult, services *ServiceClients, cfg *config.Config, opts *ApplyOptions) error {
	fmt.Printf("Starting watch mode with %d minute intervals...\n", int(opts.WatchInterval.Minutes()))

	ticker := time.NewTicker(opts.WatchInterval)
	defer ticker.Stop()

	// Perform initial apply
	if err := performApply(ctx, configs, services, cfg, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Initial apply failed: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Watch mode cancelled")
			return ctx.Err()
		case <-ticker.C:
			fmt.Printf("Performing reconciliation at %s...\n", time.Now().Format(time.RFC3339))
			if err := performApply(ctx, configs, services, cfg, opts); err != nil {
				fmt.Fprintf(os.Stderr, "Reconciliation failed: %v\n", err)
			}
		}
	}
}

func getProjectID(configs []*apply.LoadResult, opts *ApplyOptions) string {
	// Use explicit project ID if provided
	if opts.ProjectID != "" {
		return opts.ProjectID
	}

	// Try to extract project name from configurations
	// Note: This returns the project NAME, which needs to be resolved to an ID later
	for _, cfg := range configs {
		// Handle ApplyDocument format (converted from DiscoveredProject)
		if applyDoc, ok := cfg.Config.(*types.ApplyDocument); ok {
			// Extract project ID from labels (set during conversion)
			if projectID, exists := applyDoc.Metadata.Labels["matlas-mongodb-com-project-id"]; exists {
				return projectID
			}
			// Also check annotations as fallback
			if projectID, exists := applyDoc.Metadata.Annotations["matlas.mongodb.com/project-id"]; exists {
				return projectID
			}
			// Fallback: look for Project resource in the document
			for _, resource := range applyDoc.Resources {
				if resource.Kind == types.KindProject {
					if spec, ok := resource.Spec.(map[string]interface{}); ok {
						if name, ok := spec["name"].(string); ok && name != "" {
							return name
						}
					}
				}
			}
		}

		if applyConfig, ok := cfg.Config.(*types.ApplyConfig); ok {
			// Check if it's a converted Project manifest (has Kind == "Project")
			if applyConfig.Kind == "Project" && applyConfig.Spec.Name != "" {
				return applyConfig.Spec.Name
			}
			// Regular ApplyConfig
			if applyConfig.Spec.Name != "" {
				return applyConfig.Spec.Name
			}
		}
		// Handle Project-type configurations (in case they're loaded differently)
		if projectConfig, ok := cfg.Config.(*types.ProjectConfig); ok {
			if projectConfig.Name != "" {
				return projectConfig.Name
			}
		}
	}

	// TODO: Fall back to config file or environment variable
	return ""
}

// resolveProjectID resolves a project name to a project ID using the Atlas API
func resolveProjectID(ctx context.Context, projectNameOrID string, projectsService *atlas.ProjectsService, orgID string) (string, error) {
	// If it looks like a project ID (24-char hex string), use it directly
	if len(projectNameOrID) == 24 && isHexString(projectNameOrID) {
		return projectNameOrID, nil
	}

	// Otherwise, treat it as a project name and resolve to ID
	var projects []admin.Group
	var err error

	if orgID != "" {
		// If we have an organization ID, search within that org for better performance
		projects, err = projectsService.ListByOrg(ctx, orgID)
	} else {
		// Fall back to listing all projects
		projects, err = projectsService.List(ctx)
	}

	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	// Find project by name
	for _, project := range projects {
		if project.GetName() == projectNameOrID {
			return project.GetId(), nil
		}
	}

	return "", fmt.Errorf("project '%s' not found in organization", projectNameOrID)
}

// isHexString checks if a string is a valid hexadecimal string
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// getOrganizationID extracts the organization ID from configurations
func getOrganizationID(configs []*apply.LoadResult) string {
	for _, cfg := range configs {
		if applyConfig, ok := cfg.Config.(*types.ApplyConfig); ok {
			if applyConfig.Spec.OrganizationID != "" {
				return applyConfig.Spec.OrganizationID
			}
		}
		if projectConfig, ok := cfg.Config.(*types.ProjectConfig); ok {
			if projectConfig.OrganizationID != "" {
				return projectConfig.OrganizationID
			}
		}
	}
	return ""
}

func buildDesiredState(configs []*apply.LoadResult) (*apply.ProjectState, error) {
	state := &apply.ProjectState{
		Project:       nil,
		Clusters:      []types.ClusterManifest{},
		DatabaseUsers: []types.DatabaseUserManifest{},
		DatabaseRoles: []types.DatabaseRoleManifest{},
		NetworkAccess: []types.NetworkAccessManifest{},
	}

	for _, cfg := range configs {
		// Handle ApplyDocument format (converted from DiscoveredProject)
		if applyDoc, ok := cfg.Config.(*types.ApplyDocument); ok {
			err := mergeApplyDocumentToState(state, applyDoc)
			if err != nil {
				return nil, fmt.Errorf("failed to merge ApplyDocument: %w", err)
			}
			continue
		}

		// Handle original ApplyConfig format
		applyConfig, ok := cfg.Config.(*types.ApplyConfig)
		if !ok {
			continue
		}

		projectName := applyConfig.Spec.Name

		// Populate desired project manifest including tags/encryption hints if present
		if state.Project == nil {
			projManifest := &types.ProjectManifest{
				APIVersion: types.APIVersionV1,
				Kind:       types.KindProject,
				Metadata: types.ResourceMetadata{
					Name: projectName,
				},
				Spec: types.ProjectConfig{
					Name:           projectName,
					OrganizationID: applyConfig.Spec.OrganizationID,
					Tags:           applyConfig.Spec.Tags,
				},
			}
			state.Project = projManifest
		}

		// Merge clusters
		for _, cluster := range applyConfig.Spec.Clusters {
			spec := types.ClusterSpec{
				ProjectName:      projectName,
				Provider:         cluster.Provider,
				Region:           cluster.Region,
				InstanceSize:     cluster.InstanceSize,
				DiskSizeGB:       cluster.DiskSizeGB,
				BackupEnabled:    cluster.BackupEnabled,
				TierType:         cluster.TierType,
				MongoDBVersion:   cluster.MongoDBVersion,
				ClusterType:      cluster.ClusterType,
				ReplicationSpecs: cluster.ReplicationSpecs,
				AutoScaling:      cluster.AutoScaling,
				Encryption:       cluster.Encryption,
				BiConnector:      cluster.BiConnector,
			}
			manifest := types.ClusterManifest{
				APIVersion: types.APIVersionV1,
				Kind:       types.KindCluster,
				Metadata:   cluster.Metadata,
				Spec:       spec,
			}
			state.Clusters = append(state.Clusters, manifest)
		}

		// Merge database users
		for _, user := range applyConfig.Spec.DatabaseUsers {
			spec := types.DatabaseUserSpec{
				ProjectName:  projectName,
				Username:     user.Username,
				Password:     user.Password,
				Roles:        user.Roles,
				AuthDatabase: user.AuthDatabase,
				Scopes:       user.Scopes,
			}
			manifest := types.DatabaseUserManifest{
				APIVersion: types.APIVersionV1,
				Kind:       types.KindDatabaseUser,
				Metadata:   user.Metadata,
				Spec:       spec,
			}
			state.DatabaseUsers = append(state.DatabaseUsers, manifest)
		}

		// Note: ApplyConfig does not yet include custom roles. DatabaseRole manifests
		// are supplied via ApplyDocument resources. We only merge roles from ApplyDocument below.

		// Merge network access lists
		for _, access := range applyConfig.Spec.NetworkAccess {
			spec := types.NetworkAccessSpec{
				ProjectName:      projectName,
				IPAddress:        access.IPAddress,
				CIDR:             access.CIDR,
				AWSSecurityGroup: access.AWSSecurityGroup,
				Comment:          access.Comment,
				DeleteAfterDate:  access.DeleteAfterDate,
			}
			manifest := types.NetworkAccessManifest{
				APIVersion: types.APIVersionV1,
				Kind:       types.KindNetworkAccess,
				Metadata:   access.Metadata,
				Spec:       spec,
			}
			state.NetworkAccess = append(state.NetworkAccess, manifest)
		}
	}

	return state, nil
}

// mergeApplyDocumentToState merges an ApplyDocument into the project state
func mergeApplyDocumentToState(state *apply.ProjectState, applyDoc *types.ApplyDocument) error {
	for _, resource := range applyDoc.Resources {
		switch resource.Kind {
		case types.KindProject:
			// Handle project resource - extract project configuration
			if projectSpec, ok := resource.Spec.(map[string]interface{}); ok {
				// Convert spec to ProjectManifest if needed
				// For now, we'll skip project spec merging as it's not in the original structure
				_ = projectSpec
			}

		case types.KindCluster:
			// Convert resource to ClusterManifest
			// resource.Spec is already a ClusterSpec thanks to YAML unmarshaling
			clusterSpec, ok := resource.Spec.(types.ClusterSpec)
			if !ok {
				// Fallback to map-based conversion if needed
				clusterSpec = convertToClusterSpec(resource.Spec)
			}
			clusterManifest := types.ClusterManifest{
				APIVersion: resource.APIVersion,
				Kind:       resource.Kind,
				Metadata:   resource.Metadata,
				Spec:       clusterSpec,
			}
			state.Clusters = append(state.Clusters, clusterManifest)

		case types.KindDatabaseUser:
			// Convert resource to DatabaseUserManifest
			// resource.Spec is already a DatabaseUserSpec thanks to YAML unmarshaling
			userSpec, ok := resource.Spec.(types.DatabaseUserSpec)
			if !ok {
				// Fallback to map-based conversion if needed
				userSpec = convertToDatabaseUserSpec(resource.Spec)
			}
			userManifest := types.DatabaseUserManifest{
				APIVersion: resource.APIVersion,
				Kind:       resource.Kind,
				Metadata:   resource.Metadata,
				Spec:       userSpec,
			}
			state.DatabaseUsers = append(state.DatabaseUsers, userManifest)

		case types.KindDatabaseRole:
			// Convert resource to DatabaseRoleManifest
			roleSpec, ok := resource.Spec.(types.DatabaseRoleSpec)
			if !ok {
				// Fallback to map-based conversion if needed
				if specMap, okm := resource.Spec.(map[string]interface{}); okm {
					var tmp types.DatabaseRoleSpec
					// minimal conversion
					if v, ok := specMap["roleName"].(string); ok {
						tmp.RoleName = v
					}
					if v, ok := specMap["databaseName"].(string); ok {
						tmp.DatabaseName = v
					}
					// privileges and inheritedRoles conversion omitted for brevity here; validation already checks
					roleSpec = tmp
				}
			}
			roleManifest := types.DatabaseRoleManifest{
				APIVersion: resource.APIVersion,
				Kind:       resource.Kind,
				Metadata:   resource.Metadata,
				Spec:       roleSpec,
			}
			state.DatabaseRoles = append(state.DatabaseRoles, roleManifest)

		case types.KindNetworkAccess:
			// Convert resource to NetworkAccessManifest
			// resource.Spec is already a NetworkAccessSpec thanks to YAML unmarshaling
			networkSpec, ok := resource.Spec.(types.NetworkAccessSpec)
			if !ok {
				// Fallback to map-based conversion if needed
				networkSpec = convertToNetworkAccessSpec(resource.Spec)
			}
			networkManifest := types.NetworkAccessManifest{
				APIVersion: resource.APIVersion,
				Kind:       resource.Kind,
				Metadata:   resource.Metadata,
				Spec:       networkSpec,
			}
			state.NetworkAccess = append(state.NetworkAccess, networkManifest)
		}
	}
	return nil
}

// Helper functions to convert specs from generic interface{} to typed specs
func convertToClusterSpec(spec interface{}) types.ClusterSpec {
	// Complete conversion for all ClusterSpec fields
	if specMap, ok := spec.(map[string]interface{}); ok {
		clusterSpec := types.ClusterSpec{}
		if projectName, ok := specMap["projectName"].(string); ok {
			clusterSpec.ProjectName = projectName
		}
		if provider, ok := specMap["provider"].(string); ok {
			clusterSpec.Provider = provider
		}
		if region, ok := specMap["region"].(string); ok {
			clusterSpec.Region = region
		}
		if instanceSize, ok := specMap["instanceSize"].(string); ok {
			clusterSpec.InstanceSize = instanceSize
		}
		if diskSizeGB, ok := specMap["diskSizeGB"].(float64); ok {
			clusterSpec.DiskSizeGB = &diskSizeGB
		}
		if backupEnabled, ok := specMap["backupEnabled"].(bool); ok {
			clusterSpec.BackupEnabled = &backupEnabled
		}
		if tierType, ok := specMap["tierType"].(string); ok {
			clusterSpec.TierType = tierType
		}
		if mongodbVersion, ok := specMap["mongodbVersion"].(string); ok {
			clusterSpec.MongoDBVersion = mongodbVersion
		}
		if clusterType, ok := specMap["clusterType"].(string); ok {
			clusterSpec.ClusterType = clusterType
		}
		// TODO: Add conversion for complex fields like ReplicationSpecs, AutoScaling, etc.
		return clusterSpec
	}
	return types.ClusterSpec{}
}

func convertToDatabaseUserSpec(spec interface{}) types.DatabaseUserSpec {
	if specMap, ok := spec.(map[string]interface{}); ok {
		userSpec := types.DatabaseUserSpec{}
		if projectName, ok := specMap["projectName"].(string); ok {
			userSpec.ProjectName = projectName
		}
		if username, ok := specMap["username"].(string); ok {
			userSpec.Username = username
		}
		if password, ok := specMap["password"].(string); ok {
			userSpec.Password = password
		}
		if authDatabase, ok := specMap["authDatabase"].(string); ok {
			userSpec.AuthDatabase = authDatabase
		}
		// Convert roles array
		if rolesRaw, ok := specMap["roles"].([]interface{}); ok {
			for _, roleRaw := range rolesRaw {
				if roleMap, ok := roleRaw.(map[string]interface{}); ok {
					role := types.DatabaseRoleConfig{}
					if roleName, ok := roleMap["roleName"].(string); ok {
						role.RoleName = roleName
					}
					if databaseName, ok := roleMap["databaseName"].(string); ok {
						role.DatabaseName = databaseName
					}
					if collectionName, ok := roleMap["collectionName"].(string); ok {
						role.CollectionName = collectionName
					}
					userSpec.Roles = append(userSpec.Roles, role)
				}
			}
		}
		// Convert scopes array
		if scopesRaw, ok := specMap["scopes"].([]interface{}); ok {
			for _, scopeRaw := range scopesRaw {
				if scopeMap, ok := scopeRaw.(map[string]interface{}); ok {
					scope := types.UserScopeConfig{}
					if name, ok := scopeMap["name"].(string); ok {
						scope.Name = name
					}
					if scopeType, ok := scopeMap["type"].(string); ok {
						scope.Type = scopeType
					}
					userSpec.Scopes = append(userSpec.Scopes, scope)
				}
			}
		}
		return userSpec
	}
	return types.DatabaseUserSpec{}
}

func convertToNetworkAccessSpec(spec interface{}) types.NetworkAccessSpec {
	if specMap, ok := spec.(map[string]interface{}); ok {
		networkSpec := types.NetworkAccessSpec{}
		if ipAddress, ok := specMap["ipAddress"].(string); ok {
			networkSpec.IPAddress = ipAddress
		}
		if cidr, ok := specMap["cidr"].(string); ok {
			networkSpec.CIDR = cidr
		}
		if projectName, ok := specMap["projectName"].(string); ok {
			networkSpec.ProjectName = projectName
		}
		// Add more fields as needed
		return networkSpec
	}
	return types.NetworkAccessSpec{}
}

func showPlanAndGetApproval(plan *apply.Plan, opts *ApplyOptions) error {
	// Display plan summary
	fmt.Printf("\nPlan Summary:\n")
	fmt.Printf("  Operations: %d total\n", plan.Summary.TotalOperations)
	for opType, count := range plan.Summary.OperationsByType {
		fmt.Printf("    %s: %d\n", opType, count)
	}
	fmt.Printf("  Estimated Duration: %s\n", plan.Summary.EstimatedDuration)
	fmt.Printf("  Highest Risk Level: %s\n", plan.Summary.HighestRiskLevel)

	if plan.Summary.DestructiveOperations > 0 {
		fmt.Printf("  ⚠️  WARNING: %d destructive operations\n", plan.Summary.DestructiveOperations)
	}

	// Get user confirmation
	fmt.Printf("\nDo you want to apply these changes?\n")
	confirmPrompt := ui.NewConfirmationPrompt(false, false)
	confirmed, err := confirmPrompt.Confirm("Apply changes")
	if err != nil {
		return fmt.Errorf("failed to get user confirmation: %w", err)
	}

	if !confirmed {
		return fmt.Errorf("apply cancelled by user")
	}

	return nil
}

func displayExecutionResults(result *apply.ExecutionResult, opts *ApplyOptions) error {
	// Create structured output data
	outputData := output.TableData{
		Headers: []string{"Metric", "Value"},
		Rows: [][]string{
			{"Duration", result.Duration.String()},
			{"Completed Operations", fmt.Sprintf("%d", result.Summary.CompletedOperations)},
			{"Failed Operations", fmt.Sprintf("%d", result.Summary.FailedOperations)},
			{"Skipped Operations", fmt.Sprintf("%d", result.Summary.SkippedOperations)},
		},
	}

	// Display execution summary
	formatter := output.NewFormatter(config.OutputTable, os.Stdout)
	if err := formatter.Format(outputData); err != nil {
		return fmt.Errorf("failed to format execution results: %w", err)
	}

	// Display errors if any
	if len(result.Errors) > 0 {
		fmt.Println("\nErrors encountered:")
		errorData := output.TableData{
			Headers: []string{"Error"},
			Rows:    make([][]string, len(result.Errors)),
		}
		for i, err := range result.Errors {
			errorData.Rows[i] = []string{err.Message}
		}

		if err := formatter.Format(errorData); err != nil {
			return fmt.Errorf("failed to format error results: %w", err)
		}
		return fmt.Errorf("execution completed with errors")
	}

	return nil
}

func performDryRunOnly(ctx context.Context, configs []*apply.LoadResult, opts *ApplyOptions) error {
	// Build desired state from configurations
	desiredState, err := buildDesiredState(configs)
	if err != nil {
		return fmt.Errorf("failed to build desired state: %w", err)
	}

	projectID := getProjectID(configs, opts)
	return performDryRun(ctx, desiredState, projectID, opts)
}

func performDryRun(ctx context.Context, desiredState *apply.ProjectState, projectID string, opts *ApplyOptions) error {
	// For dry run, we create a plan based only on the desired state
	// without needing to discover current state from Atlas
	planBuilder := apply.NewPlanBuilder(projectID)

	// Convert desired state to operations (assuming everything is a create operation for dry run)
	operations := []apply.Operation{}

	// Add cluster operations
	for _, cluster := range desiredState.Clusters {
		operations = append(operations, apply.Operation{
			Type:         apply.OperationCreate,
			ResourceType: "cluster",
			ResourceName: cluster.Metadata.Name,
			Desired:      cluster,
			Impact: &apply.OperationImpact{
				IsDestructive:     false,
				RequiresDowntime:  false,
				EstimatedDuration: 15 * time.Minute,
				RiskLevel:         apply.RiskLevelMedium,
			},
		})
	}

	// Add database user operations
	for _, user := range desiredState.DatabaseUsers {
		operations = append(operations, apply.Operation{
			Type:         apply.OperationCreate,
			ResourceType: "databaseUser",
			ResourceName: user.Metadata.Name,
			Desired:      user,
			Impact: &apply.OperationImpact{
				IsDestructive:     false,
				RequiresDowntime:  false,
				EstimatedDuration: 2 * time.Minute,
				RiskLevel:         apply.RiskLevelLow,
			},
		})
	}

	// Add network access operations
	for _, network := range desiredState.NetworkAccess {
		operations = append(operations, apply.Operation{
			Type:         apply.OperationCreate,
			ResourceType: "networkAccess",
			ResourceName: network.Metadata.Name,
			Desired:      network,
			Impact: &apply.OperationImpact{
				IsDestructive:     false,
				RequiresDowntime:  false,
				EstimatedDuration: 1 * time.Minute,
				RiskLevel:         apply.RiskLevelLow,
			},
		})
	}

	// Add operations to plan builder
	planBuilder.AddOperations(operations)

	plan, err := planBuilder.Build()
	if err != nil {
		return fmt.Errorf("failed to create dry-run plan: %w", err)
	}

	// Run the dry-run executor
	return runDryRun(ctx, plan, opts)
}

func runDryRun(ctx context.Context, plan *apply.Plan, opts *ApplyOptions) error {
	// Parse dry-run mode
	var mode apply.DryRunMode
	switch strings.ToLower(opts.DryRunMode) {
	case "quick":
		mode = apply.DryRunModeQuick
	case "thorough":
		mode = apply.DryRunModeThorough
	case "detailed":
		mode = apply.DryRunModeDetailed
	default:
		return fmt.Errorf("invalid dry-run mode: %s (valid options: quick, thorough, detailed)", opts.DryRunMode)
	}

	// Create dry-run executor
	executor := apply.NewDryRunExecutor(mode)

	// Execute dry-run
	result, err := executor.Execute(ctx, plan)
	if err != nil {
		return fmt.Errorf("dry-run execution failed: %w", err)
	}

	// Parse output format
	var format apply.DryRunOutputFormat
	switch strings.ToLower(opts.OutputFormat) {
	case "table":
		format = apply.DryRunFormatTable
	case "json":
		format = apply.DryRunFormatJSON
	case "yaml":
		format = apply.DryRunFormatYAML
	case "summary":
		format = apply.DryRunFormatSummary
	case "detailed":
		format = apply.DryRunFormatDetailed
	default:
		return fmt.Errorf("invalid output format: %s (valid options: table, json, yaml, summary, detailed)", opts.OutputFormat)
	}

	// Format and display results
	formatter := apply.NewDryRunFormatter(format, !opts.NoColor, opts.Verbose)
	output, err := formatter.Format(result)
	if err != nil {
		return fmt.Errorf("failed to format dry-run results: %w", err)
	}

	fmt.Print(output)

	// Return error if any operations would fail
	if result.Summary.OperationsWouldFail > 0 || len(result.Errors) > 0 {
		return fmt.Errorf("dry-run completed with %d operations that would fail and %d errors",
			result.Summary.OperationsWouldFail, len(result.Errors))
	}

	return nil
}

func validateApplyOptions(opts *ApplyOptions) error {
	if len(opts.Files) == 0 {
		return fmt.Errorf("at least one configuration file must be specified with --file")
	}

	// Validate timeout only if explicitly set to negative (zero means use default)
	if opts.Timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}

	// Watch mode is incompatible with dry-run (check conflict first)
	if opts.Watch && opts.DryRun {
		return fmt.Errorf("watch mode cannot be used with dry-run")
	}

	// Validate watch interval (specific validation)
	if opts.Watch && opts.WatchInterval <= 0 {
		return fmt.Errorf("watch interval must be positive")
	}

	// Validate dry-run mode (format validation)
	validDryRunModes := []string{"quick", "thorough", "detailed"}
	if opts.DryRun && !contains(validDryRunModes, strings.ToLower(opts.DryRunMode)) {
		return fmt.Errorf("invalid dry-run mode: %s (valid options: %s)", opts.DryRunMode, strings.Join(validDryRunModes, ", "))
	}

	// Validate output format (format validation)
	validOutputFormats := []string{"table", "json", "yaml", "summary", "detailed"}
	if !contains(validOutputFormats, strings.ToLower(opts.OutputFormat)) {
		return fmt.Errorf("invalid output format: %s (valid options: %s)", opts.OutputFormat, strings.Join(validOutputFormats, ", "))
	}

	return nil
}

func expandFilePatterns(patterns []string) ([]string, error) {
	var files []string
	var stdinCount int

	for _, pattern := range patterns {
		// Handle stdin
		if pattern == "-" {
			stdinCount++
			if stdinCount > 1 {
				return nil, fmt.Errorf("stdin (-) can only be specified once")
			}
			files = append(files, "-")
			continue
		}

		// Expand glob pattern
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid file pattern '%s': %w", pattern, err)
		}

		if len(matches) == 0 {
			// If the pattern has no glob characters, treat it as a literal filename
			if !hasGlobChars(pattern) {
				// Check if file exists
				if _, err := os.Stat(pattern); err != nil {
					return nil, fmt.Errorf("file '%s' does not exist: %w", pattern, err)
				}
				files = append(files, pattern)
				continue
			}
			return nil, fmt.Errorf("no files found matching pattern: %s", pattern)
		}

		// Filter out directories and add valid files
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				return nil, fmt.Errorf("failed to stat file '%s': %w", match, err)
			}

			if !info.IsDir() {
				// Check for supported file extensions
				ext := strings.ToLower(filepath.Ext(match))
				if ext == ".yaml" || ext == ".yml" || ext == "" {
					files = append(files, match)
				}
			}
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no valid configuration files found")
	}

	return files, nil
}

// hasGlobChars checks if a string contains glob pattern characters
func hasGlobChars(s string) bool {
	return strings.ContainsAny(s, "*?[]")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// createMockPlan creates a mock execution plan for demonstration purposes
// TODO: Remove this once actual plan building is implemented
func createMockPlan(projectID string) *apply.Plan {
	if projectID == "" {
		projectID = "507f1f77bcf86cd799439011" // Mock project ID
	}

	// Create mock operations
	operations := []apply.PlannedOperation{
		{
			Operation: apply.Operation{
				Type:         apply.OperationCreate,
				ResourceType: types.KindCluster,
				ResourceName: "MyCluster",
				Desired: types.ClusterSpec{
					ProjectName:  "MyProject",
					Provider:     "AWS",
					Region:       "US_EAST_1",
					InstanceSize: "M10",
				},
				Impact: &apply.OperationImpact{
					IsDestructive:     false,
					RequiresDowntime:  false,
					EstimatedDuration: 10 * time.Minute,
					RiskLevel:         apply.RiskLevelMedium,
					Warnings:          []string{"This will create a new cluster"},
				},
			},
			ID:           "op-1",
			Dependencies: []string{},
			Priority:     100,
			Stage:        0,
			Status:       apply.OperationStatusPending,
		},
		{
			Operation: apply.Operation{
				Type:         apply.OperationCreate,
				ResourceType: types.KindDatabaseUser,
				ResourceName: "myuser",
				Desired: types.DatabaseUserSpec{
					ProjectName:  "MyProject",
					Username:     "myuser",
					Password:     "***hidden***",
					AuthDatabase: "admin",
					Roles: []types.DatabaseRoleConfig{
						{
							DatabaseName: "myapp",
							RoleName:     "readWrite",
						},
					},
				},
				Impact: &apply.OperationImpact{
					IsDestructive:     false,
					RequiresDowntime:  false,
					EstimatedDuration: 30 * time.Second,
					RiskLevel:         apply.RiskLevelLow,
				},
			},
			ID:           "op-2",
			Dependencies: []string{"op-1"}, // Depends on cluster creation
			Priority:     50,
			Stage:        1,
			Status:       apply.OperationStatusPending,
		},
		{
			Operation: apply.Operation{
				Type:         apply.OperationUpdate,
				ResourceType: types.KindNetworkAccess,
				ResourceName: "office-ip",
				Current: types.NetworkAccessSpec{
					ProjectName: "MyProject",
					IPAddress:   "203.0.113.1",
					Comment:     "Old office IP",
				},
				Desired: types.NetworkAccessSpec{
					ProjectName: "MyProject",
					IPAddress:   "203.0.113.100",
					Comment:     "New office IP",
				},
				Impact: &apply.OperationImpact{
					IsDestructive:     false,
					RequiresDowntime:  false,
					EstimatedDuration: 15 * time.Second,
					RiskLevel:         apply.RiskLevelLow,
				},
			},
			ID:           "op-3",
			Dependencies: []string{},
			Priority:     75,
			Stage:        0,
			Status:       apply.OperationStatusPending,
		},
	}

	// Create plan summary
	summary := apply.PlanSummary{
		TotalOperations:       len(operations),
		OperationsByType:      map[apply.OperationType]int{apply.OperationCreate: 2, apply.OperationUpdate: 1},
		OperationsByStage:     map[int]int{0: 2, 1: 1}, // Stage 0 has 2 ops, stage 1 has 1 op
		EstimatedDuration:     11*time.Minute + 45*time.Second,
		HighestRiskLevel:      apply.RiskLevelMedium,
		DestructiveOperations: 0,
		RequiresApproval:      true,
		ParallelizationFactor: 0.67, // 2 parallel ops out of 3 total
	}

	plan := &apply.Plan{
		ID:         fmt.Sprintf("plan-%d", time.Now().Unix()),
		ProjectID:  projectID,
		CreatedAt:  time.Now(),
		Operations: operations,
		Summary:    summary,
		Status:     apply.PlanStatusDraft,
		Config:     apply.PlanConfig{RequireApproval: true},
	}

	return plan
}
