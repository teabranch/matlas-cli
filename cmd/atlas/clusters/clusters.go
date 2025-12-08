package clusters

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"go.mongodb.org/atlas-sdk/v20250312010/admin"
	"gopkg.in/yaml.v3"

	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/services/atlas"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/ui"
	"github.com/teabranch/matlas-cli/internal/validation"
)

// CreateClusterOptions contains all options for creating a cluster
type CreateClusterOptions struct {
	ProjectID                    string
	ClusterName                  string
	Tier                         string
	Provider                     string
	Region                       string
	DiskSizeGB                   int
	DiskIOPS                     int
	EBSVolumeType                string
	BackupEnabled                bool
	PitEnabled                   bool
	MongoDBVersion               string
	ClusterType                  string
	NumShards                    int
	ReplicationFactor            int
	AutoScalingEnabled           bool
	AutoScalingDiskEnabled       bool
	AutoScalingComputeEnabled    bool
	MinInstanceSize              string
	MaxInstanceSize              string
	EncryptionAtRest             bool
	AWSKMSKeyID                  string
	AzureKeyVaultKeyID           string
	GCPKMSKeyID                  string
	AdditionalRegions            []string
	BIConnectorEnabled           bool
	BIConnectorReadPreference    string
	TerminationProtectionEnabled bool
	ConfigFile                   string
	Tags                         map[string]string
}

// APISpecYAML represents the API specification format for YAML files
type APISpecYAML struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   types.ResourceMetadata `yaml:"metadata"`
	Spec       types.ProjectConfig    `yaml:"spec"`
}

func NewClustersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "clusters",
		Short:        "Manage Atlas clusters",
		Long:         "List and manage MongoDB Atlas clusters",
		Aliases:      []string{"cluster"},
		SilenceUsage: true,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var projectID string
	var paginationFlags cli.PaginationFlags

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List clusters",
		Long: `List all clusters in a project.

This command retrieves and displays all MongoDB Atlas clusters in the specified project.
The output includes cluster name, tier, cloud provider, region, and current state.`,
		SilenceUsage: true,
		Example: `  # List clusters in a project
  matlas atlas clusters list --project-id 507f1f77bcf86cd799439011

  # List with pagination
  matlas atlas clusters list --project-id 507f1f77bcf86cd799439011 --page 2 --limit 10

  # List all clusters (no pagination)
  matlas atlas clusters list --project-id 507f1f77bcf86cd799439011 --all

  # Output as JSON
  matlas atlas clusters list --project-id 507f1f77bcf86cd799439011 --output json

  # Using alias
  matlas atlas clusters ls --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListClusters(cmd, projectID, &paginationFlags)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	cli.AddPaginationFlags(cmd, &paginationFlags)

	return cmd
}

func newGetCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "get <cluster-name>",
		Short: "Get cluster details",
		Long: `Get detailed information about a specific cluster.

This command retrieves comprehensive information about a single cluster, including
configuration, connection strings, backup settings, and current operational status.`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		Example: `  # Get cluster details
  matlas atlas clusters get MyCluster --project-id 507f1f77bcf86cd799439011

  # Output as YAML for configuration review
  matlas atlas clusters get MyCluster --project-id 507f1f77bcf86cd799439011 --output yaml

  # Get cluster with verbose logging
  matlas atlas clusters get MyCluster --project-id 507f1f77bcf86cd799439011 --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clusterName := args[0]
			return runGetCluster(cmd, projectID, clusterName)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")

	return cmd
}

func newCreateCmd() *cobra.Command {
	var projectID string
	var clusterName string
	var tier string
	var provider string
	var region string
	var diskSizeGB int
	var backupEnabled bool
	var pitEnabled bool

	// MongoDB version and cluster type
	var mongoDBVersion string
	var clusterType string

	// Advanced configuration
	var numShards int
	var replicationFactor int
	var autoScalingEnabled bool
	var autoScalingDiskEnabled bool
	var autoScalingComputeEnabled bool
	var minInstanceSize string
	var maxInstanceSize string

	// Security and encryption
	var encryptionAtRest bool
	var awsKMSKeyID string
	var azureKeyVaultKeyID string
	var gcpKMSKeyID string

	// Additional regions for multi-region deployment
	var additionalRegions []string

	// BI Connector
	var biConnectorEnabled bool
	var biConnectorReadPreference string

	// Termination protection
	var terminationProtectionEnabled bool

	// YAML configuration file
	var configFile string

	// Disk IOPS for provisioned IOPS
	var diskIOPS int
	var ebsVolumeType string

	// Tags for resource tagging
	var tags []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a cluster",
		Long: `Create a new MongoDB Atlas cluster with comprehensive configuration options.

This command creates a new MongoDB Atlas cluster with advanced configuration support
including autoscaling, encryption, multi-region deployment, and more.`,
		Example: `  # Basic cluster creation
  matlas atlas clusters create --name myCluster --project-id 507f1f77bcf86cd799439011

  # Production cluster with encryption and backup
  matlas atlas clusters create \
    --name production-cluster \
    --project-id 507f1f77bcf86cd799439011 \
    --tier M30 \
    --provider AWS \
    --region US_EAST_1 \
    --mongodb-version 8.0 \
    --backup \
    --encryption-at-rest \
    --termination-protection

  # Multi-region cluster with autoscaling
  matlas atlas clusters create \
    --name global-cluster \
    --project-id 507f1f77bcf86cd799439011 \
    --tier M40 \
    --provider AWS \
    --region US_EAST_1 \
    --additional-regions US_WEST_2,EU_WEST_1 \
    --autoscaling \
    --min-instance-size M30 \
    --max-instance-size M80

  # Sharded cluster
  matlas atlas clusters create \
    --name sharded-cluster \
    --project-id 507f1f77bcf86cd799439011 \
    --tier M30 \
    --cluster-type SHARDED \
    --num-shards 3 \
    --replication-factor 3

  # Create from YAML configuration (API specification format)
  matlas atlas clusters create --config cluster-spec.yaml

  # High-performance cluster with custom IOPS
  matlas atlas clusters create \
    --name performance-cluster \
    --project-id 507f1f77bcf86cd799439011 \
    --tier M60 \
    --disk-size 500 \
    --disk-iops 3000 \
    --ebs-volume-type gp3

  # Cluster with resource tags
  matlas atlas clusters create \
    --name tagged-cluster \
    --project-id 507f1f77bcf86cd799439011 \
    --tier M30 \
    --tag environment=production \
    --tag team=backend \
    --tag cost-center=engineering`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate PIT flag usage - PIT cannot be enabled during cluster creation
			if pitEnabled {
				return fmt.Errorf("Point-in-Time Recovery (--pit) cannot be enabled during cluster creation. Please create the cluster with --backup first, then use 'matlas atlas clusters update' to enable PIT")
			}

			// Parse tags from CLI
			parsedTags, err := parseTagsFromStrings(tags)
			if err != nil {
				return fmt.Errorf("invalid tag format: %w", err)
			}

			return runCreateClusterAdvanced(cmd, &CreateClusterOptions{
				ProjectID:                    projectID,
				ClusterName:                  clusterName,
				Tier:                         tier,
				Provider:                     provider,
				Region:                       region,
				DiskSizeGB:                   diskSizeGB,
				DiskIOPS:                     diskIOPS,
				EBSVolumeType:                ebsVolumeType,
				BackupEnabled:                backupEnabled,
				PitEnabled:                   false, // Never enable PIT during creation
				MongoDBVersion:               mongoDBVersion,
				ClusterType:                  clusterType,
				NumShards:                    numShards,
				ReplicationFactor:            replicationFactor,
				AutoScalingEnabled:           autoScalingEnabled,
				AutoScalingDiskEnabled:       autoScalingDiskEnabled,
				AutoScalingComputeEnabled:    autoScalingComputeEnabled,
				MinInstanceSize:              minInstanceSize,
				MaxInstanceSize:              maxInstanceSize,
				EncryptionAtRest:             encryptionAtRest,
				AWSKMSKeyID:                  awsKMSKeyID,
				AzureKeyVaultKeyID:           azureKeyVaultKeyID,
				GCPKMSKeyID:                  gcpKMSKeyID,
				AdditionalRegions:            additionalRegions,
				BIConnectorEnabled:           biConnectorEnabled,
				BIConnectorReadPreference:    biConnectorReadPreference,
				TerminationProtectionEnabled: terminationProtectionEnabled,
				ConfigFile:                   configFile,
				Tags:                         parsedTags,
			})
		},
	}

	// Basic flags
	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&clusterName, "name", "", "Cluster name (required)")
	cmd.Flags().StringVar(&tier, "tier", "M10", "Cluster tier (M0, M10, M20, M30, M40, M50, M60, M80, M140, M200, M300, R40, R50, etc.)")
	cmd.Flags().StringVar(&provider, "provider", "AWS", "Cloud provider (AWS, GCP, AZURE)")
	cmd.Flags().StringVar(&region, "region", "US_EAST_1", "Provider region")

	// Storage configuration
	cmd.Flags().IntVar(&diskSizeGB, "disk-size", 0, "Disk size in GB (0 for default, max varies by tier)")
	cmd.Flags().IntVar(&diskIOPS, "disk-iops", 0, "Provisioned IOPS for the cluster (AWS only, set only when >0)")
	cmd.Flags().StringVar(&ebsVolumeType, "ebs-volume-type", "", "EBS volume type (STANDARD, PROVISIONED, gp3) - AWS only; set only when specified")

	// MongoDB and cluster configuration
	cmd.Flags().StringVar(&mongoDBVersion, "mongodb-version", "7.0", "MongoDB version (7.0, 8.0, latest)")
	cmd.Flags().StringVar(&clusterType, "cluster-type", "REPLICASET", "Cluster type (REPLICASET, SHARDED)")
	cmd.Flags().IntVar(&numShards, "num-shards", 1, "Number of shards (for sharded clusters, 1-70)")
	cmd.Flags().IntVar(&replicationFactor, "replication-factor", 3, "Number of replica set members (3, 5)")

	// Backup
	cmd.Flags().BoolVar(&backupEnabled, "backup", true, "Enable continuous cloud backup")
	cmd.Flags().BoolVar(&pitEnabled, "pit", false, "Enable point-in-time recovery (only for updates - not supported during cluster creation)")

	// Autoscaling
	cmd.Flags().BoolVar(&autoScalingEnabled, "autoscaling", false, "Enable cluster tier autoscaling")
	cmd.Flags().BoolVar(&autoScalingDiskEnabled, "autoscaling-disk", true, "Enable disk autoscaling")
	cmd.Flags().BoolVar(&autoScalingComputeEnabled, "autoscaling-compute", false, "Enable compute autoscaling")
	cmd.Flags().StringVar(&minInstanceSize, "min-instance-size", "", "Minimum instance size for autoscaling")
	cmd.Flags().StringVar(&maxInstanceSize, "max-instance-size", "", "Maximum instance size for autoscaling")

	// Security and encryption
	cmd.Flags().BoolVar(&encryptionAtRest, "encryption-at-rest", false, "Enable encryption at rest")
	cmd.Flags().StringVar(&awsKMSKeyID, "aws-kms-key-id", "", "AWS KMS key ID for encryption (AWS only)")
	cmd.Flags().StringVar(&azureKeyVaultKeyID, "azure-key-vault-key-id", "", "Azure Key Vault key ID (Azure only)")
	cmd.Flags().StringVar(&gcpKMSKeyID, "gcp-kms-key-id", "", "GCP KMS key ID (GCP only)")

	// Multi-region
	cmd.Flags().StringSliceVar(&additionalRegions, "additional-regions", []string{}, "Additional regions for multi-region deployment (comma-separated)")

	// BI Connector
	cmd.Flags().BoolVar(&biConnectorEnabled, "bi-connector", false, "Enable BI Connector for Atlas")
	cmd.Flags().StringVar(&biConnectorReadPreference, "bi-connector-read-preference", "secondary", "BI Connector read preference (primary, secondary, analytics)")

	// Protection
	cmd.Flags().BoolVar(&terminationProtectionEnabled, "termination-protection", false, "Enable termination protection")

	// Configuration file
	cmd.Flags().StringVar(&configFile, "config", "", "Path to YAML configuration file (API specification format: apiVersion: matlas.mongodb.com/v1)")

	// Tags
	cmd.Flags().StringSliceVar(&tags, "tag", []string{}, "Resource tags as key=value pairs (can be specified multiple times)")

	// Note: We don't mark "name" as required here because it can come from YAML config
	// The validation will be done in the RunE function after loading YAML

	return cmd
}

func newUpdateCmd() *cobra.Command {
	var projectID string
	var tier string
	var diskSizeGB int
	var backupEnabled bool
	var pitEnabled bool
	var tagStrings []string
	var clearTags bool

	cmd := &cobra.Command{
		Use:   "update <cluster-name>",
		Short: "Update a cluster",
		Long:  "Update an existing MongoDB Atlas cluster configuration",
		Args:  cobra.ExactArgs(1),
		Example: `  # Update cluster tier
  matlas atlas clusters update myCluster --project-id 507f1f77bcf86cd799439011 --tier M20

  # Update disk size
  matlas atlas clusters update myCluster --project-id 507f1f77bcf86cd799439011 --disk-size 200

  # Enable backup
  matlas atlas clusters update myCluster --project-id 507f1f77bcf86cd799439011 --backup`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clusterName := args[0]
			// Parse tags
			tags, err := parseTagsFromStrings(tagStrings)
			if err != nil {
				return fmt.Errorf("invalid tag format: %w", err)
			}
			return runUpdateCluster(cmd, projectID, clusterName, tier, diskSizeGB, backupEnabled, cmd.Flags().Lookup("backup").Changed, pitEnabled, cmd.Flags().Lookup("pit").Changed, tags, clearTags)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().StringVar(&tier, "tier", "", "New cluster tier")
	cmd.Flags().IntVar(&diskSizeGB, "disk-size", 0, "New disk size in GB")
	cmd.Flags().BoolVar(&backupEnabled, "backup", false, "Enable/update backup settings")
	cmd.Flags().BoolVar(&pitEnabled, "pit", false, "Enable/update point-in-time recovery settings")
	cmd.Flags().StringSliceVar(&tagStrings, "tag", []string{}, "Resource tags as key=value pairs (repeatable)")
	cmd.Flags().BoolVar(&clearTags, "clear-tags", false, "Remove all tags from the cluster")

	return cmd
}

func runUpdateCluster(cmd *cobra.Command, projectID, clusterName, tier string, diskSizeGB int, backupEnabled bool, backupChanged bool, pitEnabled bool, pitChanged bool, tags map[string]string, clearTags bool) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	if err := validation.ValidateClusterName(clusterName); err != nil {
		return cli.FormatValidationError("cluster-name", clusterName, err.Error())
	}

	// Must have at least one thing to update
	if tier == "" && diskSizeGB == 0 && !backupChanged && !pitChanged && !clearTags && len(tags) == 0 {
		return fmt.Errorf("nothing to update: specify --tier, --disk-size, --backup, --pit, --tag, or --clear-tags")
	}

	// Validate PIT configuration
	if pitChanged && pitEnabled {
		// Check if backup will be enabled after this update
		if backupChanged && !backupEnabled {
			return fmt.Errorf("Point-in-Time Recovery requires backup to be enabled. Cannot enable PIT while disabling backup")
		}

		// If backup is not being changed in this update, we need to check current cluster state
		if !backupChanged {
			print_info := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
			print_info.Print("Checking current cluster backup status...")

			// Check current cluster backup status
			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
			defer cancel()

			client, err := cfg.CreateAtlasClient()
			if err != nil {
				return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
			}

			service := atlas.NewClustersService(client)
			existingCluster, err := service.Get(ctx, projectID, clusterName)
			if err != nil {
				return fmt.Errorf("failed to get cluster for validation: %w", err)
			}

			if existingCluster.BackupEnabled == nil || !*existingCluster.BackupEnabled {
				return fmt.Errorf("Point-in-Time Recovery requires backup to be enabled. Please enable backup first with --backup")
			}
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Updating cluster '%s'...", clusterName))

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewClustersService(client)

	// Get existing cluster
	existingCluster, err := service.Get(ctx, projectID, clusterName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to fetch existing cluster")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	// Create update object with only changed fields
	updateCluster := &admin.ClusterDescription20240805{}

	// Apply backup change
	if backupChanged {
		updateCluster.BackupEnabled = &backupEnabled
	}

	// Apply PIT change
	if pitChanged {
		updateCluster.PitEnabled = &pitEnabled
	}

	// Apply tag changes
	if clearTags {
		empty := []admin.ResourceTag{}
		updateCluster.Tags = &empty
	} else if len(tags) > 0 {
		updateCluster.Tags = convertTagsToAtlasFormat(tags)
	}

	// Apply tier/disk changes across existing replication specs
	if tier != "" || diskSizeGB > 0 {
		if existingCluster.ReplicationSpecs != nil {
			replicationSpecs := make([]admin.ReplicationSpec20240805, len(*existingCluster.ReplicationSpecs))
			for i, spec := range *existingCluster.ReplicationSpecs {
				replicationSpecs[i] = spec
				if spec.RegionConfigs != nil {
					regionConfigs := make([]admin.CloudRegionConfig20240805, len(*spec.RegionConfigs))
					for j, regionConfig := range *spec.RegionConfigs {
						regionConfigs[j] = regionConfig
						if regionConfig.ElectableSpecs != nil {
							electableSpecs := *regionConfig.ElectableSpecs
							if tier != "" {
								electableSpecs.InstanceSize = &tier
							}
							if diskSizeGB > 0 {
								diskSize := float64(diskSizeGB)
								electableSpecs.DiskSizeGB = &diskSize
							}
							regionConfigs[j].ElectableSpecs = &electableSpecs
						}
					}
					replicationSpecs[i].RegionConfigs = &regionConfigs
				}
			}
			updateCluster.ReplicationSpecs = &replicationSpecs
		}
	}

	// Apply the update
	updatedCluster, err := service.Update(ctx, projectID, clusterName, updateCluster)
	if err != nil {
		progress.StopSpinnerWithError("Failed to update cluster")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("")

	// Display updated cluster details
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(updatedCluster)
}

func newDeleteCmd() *cobra.Command {
	var projectID string
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete <cluster-name>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a cluster",
		Long:    "Delete a MongoDB Atlas cluster permanently",
		Args:    cobra.ExactArgs(1),
		Example: `  # Delete cluster with confirmation
  matlas atlas clusters delete myCluster --project-id 507f1f77bcf86cd799439011

  # Delete without confirmation prompt
  matlas atlas clusters delete myCluster --project-id 507f1f77bcf86cd799439011 --yes

  # Using alias
  matlas atlas clusters rm myCluster --project-id 507f1f77bcf86cd799439011`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clusterName := args[0]
			return runDeleteCluster(cmd, projectID, clusterName, yes)
		},
	}

	cmd.Flags().StringVar(&projectID, "project-id", "", "Project ID (can be set via ATLAS_PROJECT_ID env var)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")

	return cmd
}

func runListClusters(cmd *cobra.Command, projectID string, paginationFlags *cli.PaginationFlags) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate project ID
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	// Validate pagination
	paginationOpts, err := paginationFlags.Validate()
	if err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	progress.StartSpinner("Fetching clusters...")

	// Create Atlas client and service
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewClustersService(client)

	// Fetch clusters
	clusters, err := service.List(ctx, projectID)
	if err != nil {
		progress.StopSpinnerWithError("Failed to fetch clusters")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("Clusters retrieved successfully")

	// Apply pagination if needed
	if paginationOpts.ShouldPaginate() && !paginationFlags.All {
		skip := paginationOpts.CalculateSkip()
		end := skip + paginationOpts.Limit

		if skip >= len(clusters) {
			clusters = []admin.ClusterDescription20240805{}
		} else {
			if end > len(clusters) {
				end = len(clusters)
			}
			clusters = clusters[skip:end]
		}
	}

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)

	return output.FormatList(formatter, clusters,
		[]string{"NAME", "TIER", "PROVIDER", "REGION", "STATE"},
		func(item interface{}) []string {
			cluster := item.(admin.ClusterDescription20240805)
			name := getStringValue(cluster.Name)
			tier := extractClusterTier(cluster)
			provider := extractClusterProvider(cluster)
			region := extractClusterRegion(cluster)
			state := getStringValue(cluster.StateName)

			return []string{name, tier, provider, region, state}
		})
}

func runGetCluster(cmd *cobra.Command, projectID, clusterName string) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	if err := validation.ValidateClusterName(clusterName); err != nil {
		return cli.FormatValidationError("cluster-name", clusterName, err.Error())
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)

	progress.StartSpinner(fmt.Sprintf("Fetching cluster '%s'...", clusterName))

	// Create Atlas client and service
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewClustersService(client)

	// Fetch cluster
	cluster, err := service.Get(ctx, projectID, clusterName)
	if err != nil {
		progress.StopSpinnerWithError(fmt.Sprintf("Failed to fetch cluster '%s'", clusterName))
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Cluster '%s' retrieved successfully", clusterName))

	// Format and display output
	formatter := output.NewFormatter(cfg.Output, os.Stdout)
	return formatter.Format(cluster)
}

// runCreateClusterAdvanced handles the comprehensive cluster creation with full configuration support
func runCreateClusterAdvanced(cmd *cobra.Command, opts *CreateClusterOptions) error {
	// Load configuration file if provided
	if opts.ConfigFile != "" {
		if err := loadConfigFromYAML(opts); err != nil {
			return fmt.Errorf("failed to load configuration from YAML: %w", err)
		}
	}

	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	opts.ProjectID = cfg.ResolveProjectID(opts.ProjectID)

	// Validate inputs
	if err := validateClusterOptions(opts); err != nil {
		return err
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Creating cluster '%s'...", opts.ClusterName))

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		progress.StopSpinnerWithError("Failed to initialize Atlas client")
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewClustersService(client)

	// Build cluster configuration
	cluster, err := buildClusterConfiguration(opts)
	if err != nil {
		progress.StopSpinnerWithError("Failed to build cluster configuration")
		return fmt.Errorf("failed to build cluster configuration: %w", err)
	}

	// Create the cluster
	createdCluster, err := service.Create(ctx, opts.ProjectID, cluster)
	if err != nil {
		progress.StopSpinnerWithError("Failed to create cluster")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner("")

	// Display created cluster details with prettier formatting
	formatter := output.NewCreateResultFormatter(cfg.Output, os.Stdout)
	return formatter.FormatCreateResult(createdCluster, "cluster")
}

// loadConfigFromYAML loads cluster configuration from API specification YAML file
func loadConfigFromYAML(opts *CreateClusterOptions) error {
	if opts.ConfigFile == "" {
		return nil
	}

	// Resolve the file path
	configPath, err := filepath.Abs(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("invalid config file path: %w", err)
	}

	// Read the YAML file
	data, err := os.ReadFile(configPath) //nolint:gosec // reading user-provided path is expected for CLI tool
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse as API specification format
	var apiSpec APISpecYAML
	if err := yaml.Unmarshal(data, &apiSpec); err != nil {
		return fmt.Errorf("failed to parse API specification YAML: %w", err)
	}

	// Validate API version
	if apiSpec.APIVersion == "" {
		return fmt.Errorf("missing required field: apiVersion (expected: matlas.mongodb.com/v1)")
	}
	if apiSpec.APIVersion != "matlas.mongodb.com/v1" {
		return fmt.Errorf("unsupported API version: %s (expected: matlas.mongodb.com/v1)", apiSpec.APIVersion)
	}

	// Validate kind
	if apiSpec.Kind == "" {
		return fmt.Errorf("missing required field: kind (expected: Project)")
	}
	if apiSpec.Kind != "Project" {
		return fmt.Errorf("unsupported kind: %s (expected: Project)", apiSpec.Kind)
	}

	return loadFromAPISpec(opts, &apiSpec)
}

// loadFromAPISpec loads configuration from API specification format
func loadFromAPISpec(opts *CreateClusterOptions, apiSpec *APISpecYAML) error {

	// Extract project information
	if opts.ProjectID == "" {
		// Use project metadata name or organizationId for project identification
		// In API spec format, the actual project should already exist or be created separately
		opts.ProjectID = apiSpec.Spec.OrganizationID // This may need to be resolved differently
	}

	// Process clusters from the spec
	if len(apiSpec.Spec.Clusters) == 0 {
		return fmt.Errorf("no clusters defined in API specification")
	}

	// For cluster creation, we take the first cluster or find one that matches the provided name
	var clusterConfig *types.ClusterConfig
	if opts.ClusterName != "" {
		// Find cluster by name
		for i := range apiSpec.Spec.Clusters {
			if apiSpec.Spec.Clusters[i].Metadata.Name == opts.ClusterName {
				clusterConfig = &apiSpec.Spec.Clusters[i]
				break
			}
		}
		if clusterConfig == nil {
			return fmt.Errorf("cluster '%s' not found in API specification", opts.ClusterName)
		}
	} else {
		// Use the first cluster and set the name
		clusterConfig = &apiSpec.Spec.Clusters[0]
		opts.ClusterName = clusterConfig.Metadata.Name
	}

	// Map API spec fields to options (CLI flags take precedence)
	if opts.Tier == "M10" && clusterConfig.InstanceSize != "" {
		opts.Tier = clusterConfig.InstanceSize
	}
	if opts.Provider == "AWS" && clusterConfig.Provider != "" {
		opts.Provider = clusterConfig.Provider
	}
	if opts.Region == "US_EAST_1" && clusterConfig.Region != "" {
		opts.Region = strings.ToUpper(strings.ReplaceAll(clusterConfig.Region, "-", "_"))
	}
	if opts.MongoDBVersion == "7.0" && clusterConfig.MongoDBVersion != "" {
		opts.MongoDBVersion = clusterConfig.MongoDBVersion
	}
	if opts.ClusterType == "REPLICASET" && clusterConfig.ClusterType != "" {
		opts.ClusterType = clusterConfig.ClusterType
	}

	// Storage configuration
	if clusterConfig.DiskSizeGB != nil && opts.DiskSizeGB == 0 {
		opts.DiskSizeGB = int(*clusterConfig.DiskSizeGB)
	}

	// Backup configuration
	if clusterConfig.BackupEnabled != nil && opts.BackupEnabled {
		opts.BackupEnabled = *clusterConfig.BackupEnabled
	}

	// Auto-scaling configuration
	if clusterConfig.AutoScaling != nil {
		if clusterConfig.AutoScaling.DiskGB != nil && clusterConfig.AutoScaling.DiskGB.Enabled != nil {
			opts.AutoScalingDiskEnabled = *clusterConfig.AutoScaling.DiskGB.Enabled
		}
		if clusterConfig.AutoScaling.Compute != nil {
			if clusterConfig.AutoScaling.Compute.Enabled != nil {
				opts.AutoScalingComputeEnabled = *clusterConfig.AutoScaling.Compute.Enabled
			}
			if opts.MinInstanceSize == "" && clusterConfig.AutoScaling.Compute.MinInstanceSize != "" {
				opts.MinInstanceSize = clusterConfig.AutoScaling.Compute.MinInstanceSize
			}
			if opts.MaxInstanceSize == "" && clusterConfig.AutoScaling.Compute.MaxInstanceSize != "" {
				opts.MaxInstanceSize = clusterConfig.AutoScaling.Compute.MaxInstanceSize
			}
		}
	}

	// Encryption configuration
	if clusterConfig.Encryption != nil && !opts.EncryptionAtRest {
		if clusterConfig.Encryption.EncryptionAtRestProvider != "" && clusterConfig.Encryption.EncryptionAtRestProvider != "NONE" {
			opts.EncryptionAtRest = true

			// Set KMS key IDs based on provider
			if clusterConfig.Encryption.AWSKMSConfig != nil && opts.AWSKMSKeyID == "" {
				opts.AWSKMSKeyID = clusterConfig.Encryption.AWSKMSConfig.CustomerMasterKeyID
			}
			if clusterConfig.Encryption.AzureKeyVaultConfig != nil && opts.AzureKeyVaultKeyID == "" {
				opts.AzureKeyVaultKeyID = clusterConfig.Encryption.AzureKeyVaultConfig.KeyIdentifier
			}
			if clusterConfig.Encryption.GoogleCloudKMSConfig != nil && opts.GCPKMSKeyID == "" {
				opts.GCPKMSKeyID = clusterConfig.Encryption.GoogleCloudKMSConfig.KeyVersionResourceID
			}
		}
	}

	// BI Connector configuration
	if clusterConfig.BiConnector != nil && !opts.BIConnectorEnabled {
		if clusterConfig.BiConnector.Enabled != nil {
			opts.BIConnectorEnabled = *clusterConfig.BiConnector.Enabled
		}
		if opts.BIConnectorReadPreference == "secondary" && clusterConfig.BiConnector.ReadPreference != "" {
			opts.BIConnectorReadPreference = clusterConfig.BiConnector.ReadPreference
		}
	}

	// Tags configuration - merge YAML tags with any existing CLI tags
	if len(clusterConfig.Tags) > 0 {
		if opts.Tags == nil {
			opts.Tags = make(map[string]string)
		}
		// CLI tags take precedence over YAML tags
		for key, value := range clusterConfig.Tags {
			if _, exists := opts.Tags[key]; !exists {
				opts.Tags[key] = value
			}
		}
	}

	return nil
}

// validateClusterOptions validates the cluster creation options
func validateClusterOptions(opts *CreateClusterOptions) error {
	// Validate that required fields are provided
	if opts.ClusterName == "" {
		return cli.FormatValidationError("name", "", "cluster name is required")
	}
	if opts.ProjectID == "" {
		return cli.FormatValidationError("project-id", "", "project ID is required")
	}

	// Validate project ID
	if err := validation.ValidateProjectID(opts.ProjectID); err != nil {
		return cli.FormatValidationError("project-id", opts.ProjectID, err.Error())
	}

	// Validate cluster name
	if err := validation.ValidateClusterName(opts.ClusterName); err != nil {
		return cli.FormatValidationError("name", opts.ClusterName, err.Error())
	}

	// Validate tier
	validTiers := []string{"M0", "M10", "M20", "M30", "M40", "M50", "M60", "M80", "M140", "M200", "M300", "M400", "M700", "R40", "R50", "R60", "R80", "R200", "R300", "R400", "R700"}
	if !contains(validTiers, opts.Tier) {
		return cli.FormatValidationError("tier", opts.Tier, "must be one of: "+strings.Join(validTiers, ", "))
	}

	// Validate provider
	validProviders := []string{"AWS", "GCP", "AZURE"}
	if !contains(validProviders, opts.Provider) {
		return cli.FormatValidationError("provider", opts.Provider, "must be one of: "+strings.Join(validProviders, ", "))
	}

	// Validate MongoDB version
	validVersions := []string{"7.0", "8.0", "latest"}
	if !contains(validVersions, opts.MongoDBVersion) {
		return cli.FormatValidationError("mongodb-version", opts.MongoDBVersion, "must be one of: "+strings.Join(validVersions, ", "))
	}

	// Validate cluster type
	validClusterTypes := []string{"REPLICASET", "SHARDED"}
	if !contains(validClusterTypes, opts.ClusterType) {
		return cli.FormatValidationError("cluster-type", opts.ClusterType, "must be one of: "+strings.Join(validClusterTypes, ", "))
	}

	// Validate shards for sharded clusters
	if opts.ClusterType == "SHARDED" {
		if opts.NumShards < 1 || opts.NumShards > 70 {
			return cli.FormatValidationError("num-shards", strconv.Itoa(opts.NumShards), "must be between 1 and 70 for sharded clusters")
		}
		// Sharded clusters require M30 or higher
		if strings.HasPrefix(opts.Tier, "M") {
			tierNum, err := strconv.Atoi(opts.Tier[1:])
			if err == nil && tierNum < 30 {
				return cli.FormatValidationError("tier", opts.Tier, "sharded clusters require M30 or higher")
			}
		}
	}

	// Validate replication factor
	validReplicationFactors := []int{3, 5}
	if !containsInt(validReplicationFactors, opts.ReplicationFactor) {
		return cli.FormatValidationError("replication-factor", strconv.Itoa(opts.ReplicationFactor), "must be 3 or 5")
	}

	// Validate autoscaling configuration
	if opts.AutoScalingComputeEnabled {
		if opts.MinInstanceSize == "" || opts.MaxInstanceSize == "" {
			return fmt.Errorf("min-instance-size and max-instance-size are required when compute autoscaling is enabled")
		}
	}

	// Validate BI Connector read preference
	if opts.BIConnectorEnabled {
		validReadPrefs := []string{"primary", "secondary", "analytics"}
		if !contains(validReadPrefs, opts.BIConnectorReadPreference) {
			return cli.FormatValidationError("bi-connector-read-preference", opts.BIConnectorReadPreference, "must be one of: "+strings.Join(validReadPrefs, ", "))
		}
	}

	// Validate Atlas resource tags
	if err := validation.ValidateAtlasResourceTags(opts.Tags, "tags"); err != nil {
		return cli.FormatValidationError("tags", fmt.Sprintf("%v", opts.Tags), err.Error())
	}

	return nil
}

// buildClusterConfiguration builds the Atlas SDK cluster configuration from options
func buildClusterConfiguration(opts *CreateClusterOptions) (*admin.ClusterDescription20240805, error) {
	// Basic cluster configuration
	cluster := &admin.ClusterDescription20240805{
		Name:                &opts.ClusterName,
		ClusterType:         &opts.ClusterType,
		MongoDBMajorVersion: &opts.MongoDBVersion,
		BackupEnabled:       &opts.BackupEnabled,
		Tags:                convertTagsToAtlasFormat(opts.Tags),
	}

	// Set termination protection
	if opts.TerminationProtectionEnabled {
		cluster.TerminationProtectionEnabled = &opts.TerminationProtectionEnabled
	}

	// Build replication specs
	if err := buildReplicationSpecs(cluster, opts); err != nil {
		return nil, err
	}

	// Configure encryption
	if opts.EncryptionAtRest {
		cluster.EncryptionAtRestProvider = admin.PtrString("AWS") // Default to AWS for now
		if opts.AWSKMSKeyID != "" {
			// Note: KMS configuration requires more complex setup in real Atlas API
			// This is a simplified implementation
		}
	}

	// Configure BI Connector
	if opts.BIConnectorEnabled {
		cluster.BiConnector = &admin.BiConnector{
			Enabled:        &opts.BIConnectorEnabled,
			ReadPreference: &opts.BIConnectorReadPreference,
		}
	}

	// Configure autoscaling (simplified - real implementation would be more complex)
	if opts.AutoScalingDiskEnabled || opts.AutoScalingComputeEnabled {
		// Note: Autoscaling configuration in Atlas API is more complex
		// This is a placeholder for the structure
	}

	return cluster, nil
}

// buildReplicationSpecs builds the replication specifications for the cluster
func buildReplicationSpecs(cluster *admin.ClusterDescription20240805, opts *CreateClusterOptions) error {
	var replicationSpecs []admin.ReplicationSpec20240805

	// For sharded clusters, create multiple shards
	numShards := opts.NumShards
	if opts.ClusterType == "REPLICASET" {
		numShards = 1
	}

	for i := 0; i < numShards; i++ {
		// Build region configurations
		regionConfigs, err := buildRegionConfigs(opts)
		if err != nil {
			return err
		}

		replicationSpec := admin.ReplicationSpec20240805{
			RegionConfigs: &regionConfigs,
		}

		replicationSpecs = append(replicationSpecs, replicationSpec)
	}

	cluster.ReplicationSpecs = &replicationSpecs
	return nil
}

// buildRegionConfigs builds the region configurations for the cluster
func buildRegionConfigs(opts *CreateClusterOptions) ([]admin.CloudRegionConfig20240805, error) {
	var regionConfigs []admin.CloudRegionConfig20240805

	// Primary region configuration
	primaryConfig, err := buildSingleRegionConfig(opts.Provider, opts.Region, opts.Tier, opts.DiskSizeGB, opts.DiskIOPS, opts.EBSVolumeType, 7, opts.ReplicationFactor)
	if err != nil {
		return nil, err
	}
	regionConfigs = append(regionConfigs, primaryConfig)

	// Additional regions for multi-region deployment
	priority := 6
	for _, region := range opts.AdditionalRegions {
		additionalConfig, err := buildSingleRegionConfig(opts.Provider, region, opts.Tier, opts.DiskSizeGB, opts.DiskIOPS, opts.EBSVolumeType, priority, 2)
		if err != nil {
			return nil, err
		}
		regionConfigs = append(regionConfigs, additionalConfig)
		priority--
		if priority < 1 {
			priority = 1
		}
	}

	return regionConfigs, nil
}

// buildSingleRegionConfig builds a single region configuration
func buildSingleRegionConfig(provider, region, tier string, diskSizeGB, diskIOPS int, ebsVolumeType string, priority, nodeCount int) (admin.CloudRegionConfig20240805, error) {
	priorityPtr := priority

	// Build hardware specs (only set optional fields when explicitly provided)
	electableSpecs := admin.HardwareSpec20240805{
		InstanceSize: &tier,
		NodeCount:    &nodeCount,
	}
	if ebsVolumeType != "" {
		electableSpecs.EbsVolumeType = &ebsVolumeType
	}
	if diskIOPS > 0 {
		electableSpecs.DiskIOPS = &diskIOPS
	}

	// Set disk size if specified
	if diskSizeGB > 0 {
		diskSize := float64(diskSizeGB)
		electableSpecs.DiskSizeGB = &diskSize
	}

	regionConfig := admin.CloudRegionConfig20240805{
		ProviderName:   &provider,
		RegionName:     &region,
		Priority:       &priorityPtr,
		ElectableSpecs: &electableSpecs,
	}

	return regionConfig, nil
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// convertTagsToAtlasFormat converts map[string]string tags to []admin.ResourceTag format
func convertTagsToAtlasFormat(tags map[string]string) *[]admin.ResourceTag {
	if len(tags) == 0 {
		return nil
	}

	var atlasTags []admin.ResourceTag
	for key, value := range tags {
		atlasTags = append(atlasTags, admin.ResourceTag{
			Key:   key,
			Value: value,
		})
	}

	return &atlasTags
}

// parseTagsFromStrings parses key=value strings into a map[string]string
func parseTagsFromStrings(tagStrings []string) (map[string]string, error) {
	if len(tagStrings) == 0 {
		return nil, nil
	}

	tags := make(map[string]string)
	for _, tagStr := range tagStrings {
		parts := strings.SplitN(tagStr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("tag must be in format key=value, got: %s", tagStr)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("tag key cannot be empty in: %s", tagStr)
		}

		// Check for duplicate keys
		if _, exists := tags[key]; exists {
			return nil, fmt.Errorf("duplicate tag key: %s", key)
		}

		tags[key] = value
	}

	return tags, nil
}

func containsInt(slice []int, item int) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

func runDeleteCluster(cmd *cobra.Command, projectID, clusterName string, yes bool) error {
	// Get configuration first to resolve project ID if not provided
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve project ID from flag or config/env
	projectID = cfg.ResolveProjectID(projectID)

	// Validate inputs
	if err := validation.ValidateProjectID(projectID); err != nil {
		return cli.FormatValidationError("project-id", projectID, err.Error())
	}

	if err := validation.ValidateClusterName(clusterName); err != nil {
		return cli.FormatValidationError("cluster-name", clusterName, err.Error())
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Create Atlas client
	client, err := cfg.CreateAtlasClient()
	if err != nil {
		return cli.WrapWithSuggestion(err, "Check your API key and public key configuration")
	}

	service := atlas.NewClustersService(client)

	// Confirmation prompt (unless --yes flag is used)
	if !yes {
		confirmPrompt := ui.NewConfirmationPrompt(false, false)
		confirmed, err := confirmPrompt.Confirm(fmt.Sprintf("Are you sure you want to delete cluster '%s' from project '%s'? This action cannot be undone and will permanently delete all data.", clusterName, projectID))
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("Cluster deletion cancelled.")
			return nil
		}
	}

	// Create progress indicator
	progress := ui.NewProgressIndicator(cmd.Flag("verbose").Changed, false)
	progress.StartSpinner(fmt.Sprintf("Deleting cluster '%s'...", clusterName))

	// Delete the cluster
	err = service.Delete(ctx, projectID, clusterName)
	if err != nil {
		progress.StopSpinnerWithError("Failed to delete cluster")
		errorFormatter := cli.NewErrorFormatter(cmd.Flag("verbose").Changed)
		return fmt.Errorf("%s", errorFormatter.Format(err))
	}

	progress.StopSpinner(fmt.Sprintf("Cluster '%s' deletion initiated successfully", clusterName))
	return nil
}

// Helper function to safely extract string values from pointers
func getStringValue(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

// extractClusterTier extracts the tier (instance size) from cluster replication specs
func extractClusterTier(cluster admin.ClusterDescription20240805) string {
	if cluster.ReplicationSpecs == nil || len(*cluster.ReplicationSpecs) == 0 {
		return "N/A"
	}

	// Get first replication spec
	repSpec := (*cluster.ReplicationSpecs)[0]
	if repSpec.RegionConfigs == nil || len(*repSpec.RegionConfigs) == 0 {
		return "N/A"
	}

	// Get first region config
	regionConfig := (*repSpec.RegionConfigs)[0]

	// Try electable specs first (most common)
	if regionConfig.ElectableSpecs != nil {
		if instanceSize := regionConfig.ElectableSpecs.GetInstanceSize(); instanceSize != "" {
			return instanceSize
		}
	}

	// Try analytics specs
	if regionConfig.AnalyticsSpecs != nil {
		if instanceSize := regionConfig.AnalyticsSpecs.GetInstanceSize(); instanceSize != "" {
			return instanceSize
		}
	}

	// Try read-only specs
	if regionConfig.ReadOnlySpecs != nil {
		if instanceSize := regionConfig.ReadOnlySpecs.GetInstanceSize(); instanceSize != "" {
			return instanceSize
		}
	}

	return "N/A"
}

// extractClusterProvider extracts the cloud provider from cluster replication specs
func extractClusterProvider(cluster admin.ClusterDescription20240805) string {
	if cluster.ReplicationSpecs == nil || len(*cluster.ReplicationSpecs) == 0 {
		return "N/A"
	}

	// Get first replication spec
	repSpec := (*cluster.ReplicationSpecs)[0]
	if repSpec.RegionConfigs == nil || len(*repSpec.RegionConfigs) == 0 {
		return "N/A"
	}

	// Get first region config
	regionConfig := (*repSpec.RegionConfigs)[0]

	// Get provider name
	if provider := regionConfig.GetProviderName(); provider != "" {
		return provider
	}

	// Fall back to backing provider name for tenant clusters
	if backingProvider := regionConfig.GetBackingProviderName(); backingProvider != "" {
		return backingProvider
	}

	return "N/A"
}

// extractClusterRegion extracts the region from cluster replication specs
func extractClusterRegion(cluster admin.ClusterDescription20240805) string {
	if cluster.ReplicationSpecs == nil || len(*cluster.ReplicationSpecs) == 0 {
		return "N/A"
	}

	// Get first replication spec
	repSpec := (*cluster.ReplicationSpecs)[0]
	if repSpec.RegionConfigs == nil || len(*repSpec.RegionConfigs) == 0 {
		return "N/A"
	}

	// Get first region config
	regionConfig := (*repSpec.RegionConfigs)[0]

	// Get region name
	if region := regionConfig.GetRegionName(); region != "" {
		return region
	}

	return "N/A"
}
