package infra

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/internal/apply"
	"github.com/teabranch/matlas-cli/internal/cli"
	"github.com/teabranch/matlas-cli/internal/config"
	"github.com/teabranch/matlas-cli/internal/output"
	"github.com/teabranch/matlas-cli/internal/types"
	"github.com/teabranch/matlas-cli/internal/validation"
)

// ShowOptions contains the options for the show command
type ShowOptions struct {
	ProjectID    string
	OutputFormat string
	Verbose      bool
	NoColor      bool
	Timeout      time.Duration
	ResourceType string
	ResourceName string
	ShowSecrets  bool
	ShowMetadata bool
}

// NewShowCmd creates the show subcommand
func NewShowCmd() *cobra.Command {
	opts := &ShowOptions{}

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display current state of Atlas resources",
		Long: `Display the current state of Atlas resources in a project.

This command discovers and displays the current configuration of Atlas resources
such as clusters, database users, and network access lists. It's useful for
understanding the current state before making changes.`,
		Example: `  # Show all resources in a project
  matlas infra show --project-id 507f1f77bcf86cd799439011

  # Show specific resource type
  matlas infra show --project-id 507f1f77bcf86cd799439011 --resource-type clusters

  # Show specific resource
  matlas infra show --project-id 507f1f77bcf86cd799439011 --resource-type clusters --resource-name MyCluster

  # Show with JSON output
  matlas infra show --project-id 507f1f77bcf86cd799439011 --output json

  # Show with sensitive information
  matlas infra show --project-id 507f1f77bcf86cd799439011 --show-secrets`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, opts)
		},
	}

	// Required flags
	cmd.Flags().StringVar(&opts.ProjectID, "project-id", "", "Atlas project ID (required)")

	// Output flags
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "table", "Output format: table, json, yaml, summary")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")

	// Filtering flags
	cmd.Flags().StringVar(&opts.ResourceType, "resource-type", "", "Filter by resource type: clusters, users, network-access")
	cmd.Flags().StringVar(&opts.ResourceName, "resource-name", "", "Filter by specific resource name")

	// Display options
	cmd.Flags().BoolVar(&opts.ShowSecrets, "show-secrets", false, "Show sensitive information (passwords, keys)")
	cmd.Flags().BoolVar(&opts.ShowMetadata, "show-metadata", false, "Show additional metadata (creation time, etc.)")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 5*time.Minute, "Timeout for state discovery")

	return cmd
}

func runShow(cmd *cobra.Command, opts *ShowOptions) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), opts.Timeout)
	defer cancel()

	// Validate options
	if err := validateShowOptions(opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Tighten project-id validation (ObjectID format)
	if err := validation.ValidateProjectID(opts.ProjectID); err != nil {
		return cli.FormatValidationError("project-id", opts.ProjectID, err.Error())
	}

	// Initialize services
	cfg, err := config.Load(cmd, "")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	services, err := initializeServices(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Discover current state
	state, err := discoverCurrentState(ctx, services, cfg, opts)
	if err != nil {
		return fmt.Errorf("failed to discover current state: %w", err)
	}

	// Filter state based on options
	filteredState := filterState(state, opts)

	// Display state
	return displayState(filteredState, opts)
}

func discoverCurrentState(ctx context.Context, services *ServiceClients, cfg *config.Config, opts *ShowOptions) (*apply.ProjectState, error) {
	if opts.Verbose {
		fmt.Printf("Discovering current state for project %s...\n", opts.ProjectID)
	}

	// Create Atlas client for discovery
	atlasClient, err := cfg.CreateAtlasClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Atlas client for discovery: %w", err)
	}

	// Initialize discovery service
	discoveryService := apply.NewAtlasStateDiscovery(atlasClient)

	// Discover project state
	state, err := discoveryService.DiscoverProject(ctx, opts.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to discover project state: %w", err)
	}

	return state, nil
}

func filterState(state *apply.ProjectState, opts *ShowOptions) *apply.ProjectState {
	if opts.ResourceType == "" && opts.ResourceName == "" {
		return state // No filtering needed
	}

	filtered := &apply.ProjectState{
		Clusters:      []types.ClusterManifest{},
		DatabaseUsers: []types.DatabaseUserManifest{},
		NetworkAccess: []types.NetworkAccessManifest{},
	}

	// Filter by resource type
	switch strings.ToLower(opts.ResourceType) {
	case "clusters", "cluster":
		if opts.ResourceName != "" {
			// Filter by specific cluster name
			for _, cluster := range state.Clusters {
				if cluster.Metadata.Name == opts.ResourceName {
					filtered.Clusters = append(filtered.Clusters, cluster)
				}
			}
		} else {
			// Include all clusters
			filtered.Clusters = state.Clusters
		}
	case "users", "user", "database-users":
		if opts.ResourceName != "" {
			// Filter by specific user name
			for _, user := range state.DatabaseUsers {
				if user.Spec.Username == opts.ResourceName {
					filtered.DatabaseUsers = append(filtered.DatabaseUsers, user)
				}
			}
		} else {
			// Include all users
			filtered.DatabaseUsers = state.DatabaseUsers
		}
	case "network-access", "network":
		if opts.ResourceName != "" {
			// Filter by specific IP or comment
			for _, access := range state.NetworkAccess {
				if access.Spec.IPAddress == opts.ResourceName || access.Spec.Comment == opts.ResourceName {
					filtered.NetworkAccess = append(filtered.NetworkAccess, access)
				}
			}
		} else {
			// Include all network access
			filtered.NetworkAccess = state.NetworkAccess
		}
	case "":
		// No resource type filter, but check for resource name across all types
		if opts.ResourceName != "" {
			for _, cluster := range state.Clusters {
				if cluster.Metadata.Name == opts.ResourceName {
					filtered.Clusters = append(filtered.Clusters, cluster)
				}
			}
			for _, user := range state.DatabaseUsers {
				if user.Spec.Username == opts.ResourceName {
					filtered.DatabaseUsers = append(filtered.DatabaseUsers, user)
				}
			}
			for _, access := range state.NetworkAccess {
				if access.Spec.IPAddress == opts.ResourceName || access.Spec.Comment == opts.ResourceName {
					filtered.NetworkAccess = append(filtered.NetworkAccess, access)
				}
			}
		}
	}

	return filtered
}

func displayState(state *apply.ProjectState, opts *ShowOptions) error {
	switch strings.ToLower(opts.OutputFormat) {
	case "json":
		return displayStateJSON(state, opts)
	case "yaml":
		return displayStateYAML(state, opts)
	case "summary":
		return displayStateSummary(state, opts)
	default: // table
		return displayStateTable(state, opts)
	}
}

func displayStateJSON(state *apply.ProjectState, opts *ShowOptions) error {
	// Mask secrets if not explicitly shown
	displayState := state
	if !opts.ShowSecrets {
		displayState = maskSecrets(state)
	}

	formatter := output.NewFormatter(config.OutputJSON, os.Stdout)
	return formatter.Format(displayState)
}

func displayStateYAML(state *apply.ProjectState, opts *ShowOptions) error {
	// Mask secrets if not explicitly shown
	displayState := state
	if !opts.ShowSecrets {
		displayState = maskSecrets(state)
	}

	formatter := output.NewFormatter(config.OutputYAML, os.Stdout)
	return formatter.Format(displayState)
}

func displayStateSummary(state *apply.ProjectState, opts *ShowOptions) error {
	// Use internal/output to standardize summary output
	rows := [][]string{
		{"Clusters", fmt.Sprintf("%d", len(state.Clusters))},
		{"Database Users", fmt.Sprintf("%d", len(state.DatabaseUsers))},
		{"Network Access Lists", fmt.Sprintf("%d", len(state.NetworkAccess))},
	}
	data := output.TableData{Headers: []string{"Resource", "Count"}, Rows: rows}
	formatter := output.NewFormatter(config.OutputTable, os.Stdout)
	if state.Project != nil {
		fmt.Fprintf(os.Stdout, "Project: %s\n\n", state.Project.Metadata.Name)
	} else {
		fmt.Fprintln(os.Stdout, "Project State")
		fmt.Fprintln(os.Stdout)
	}
	return formatter.Format(data)
}

func displayStateTable(state *apply.ProjectState, opts *ShowOptions) error {
	// Header
	if state.Project != nil {
		fmt.Fprintf(os.Stdout, "Atlas Project State: %s\n\n", state.Project.Metadata.Name)
	} else {
		fmt.Fprintln(os.Stdout, "Atlas Project State")
		fmt.Fprintln(os.Stdout)
	}

	formatter := output.NewFormatter(config.OutputTable, os.Stdout)

	// Clusters table
	if len(state.Clusters) > 0 {
		clusterRows := make([][]string, 0, len(state.Clusters))
		for _, c := range state.Clusters {
			status := "Active"
			clusterRows = append(clusterRows, []string{
				c.Metadata.Name,
				c.Spec.Provider,
				c.Spec.InstanceSize,
				c.Spec.Region,
				status,
			})
		}
		_ = formatter.Format(output.TableData{
			Headers: []string{"Name", "Provider", "Instance", "Region", "Status"},
			Rows:    clusterRows,
		})
		fmt.Fprintln(os.Stdout)
	}

	// Users table
	if len(state.DatabaseUsers) > 0 {
		userRows := make([][]string, 0, len(state.DatabaseUsers))
		for _, u := range state.DatabaseUsers {
			rolesList := make([]string, len(u.Spec.Roles))
			for i, role := range u.Spec.Roles {
				rolesList[i] = fmt.Sprintf("%s@%s", role.RoleName, role.DatabaseName)
			}
			roles := strings.Join(rolesList, ", ")
			if len(roles) > 35 {
				roles = roles[:32] + "..."
			}
			password := "***hidden***"
			if opts.ShowSecrets && u.Spec.Password != "" {
				password = u.Spec.Password
			}
			userRows = append(userRows, []string{u.Spec.Username, u.Spec.AuthDatabase, roles, password})
		}
		_ = formatter.Format(output.TableData{
			Headers: []string{"Username", "Auth DB", "Roles", "Password"},
			Rows:    userRows,
		})
		fmt.Fprintln(os.Stdout)
	}

	// Network access table
	if len(state.NetworkAccess) > 0 {
		hasCIDR := false
		for _, a := range state.NetworkAccess {
			if a.Spec.CIDR != "" && a.Spec.CIDR != a.Spec.IPAddress+"/32" {
				hasCIDR = true
				break
			}
		}
		var headers []string
		rows := make([][]string, 0, len(state.NetworkAccess))
		if hasCIDR {
			headers = []string{"IP Address", "CIDR Block", "Comment"}
			for _, a := range state.NetworkAccess {
				cidr := a.Spec.CIDR
				if cidr == "" {
					cidr = "N/A"
				}
				comment := a.Spec.Comment
				if comment == "" {
					comment = "No description"
				}
				rows = append(rows, []string{a.Spec.IPAddress, cidr, comment})
			}
		} else {
			headers = []string{"IP Address", "Comment"}
			for _, a := range state.NetworkAccess {
				comment := a.Spec.Comment
				if comment == "" {
					comment = "No description"
				}
				rows = append(rows, []string{a.Spec.IPAddress, comment})
			}
		}
		_ = formatter.Format(output.TableData{Headers: headers, Rows: rows})
		fmt.Fprintln(os.Stdout)
	}

	if opts.ShowMetadata && opts.Verbose {
		fmt.Fprintf(os.Stdout, "Metadata:\n  Discovery completed at: %s\n  Total resources: %d\n",
			time.Now().Format(time.RFC3339),
			len(state.Clusters)+len(state.DatabaseUsers)+len(state.NetworkAccess))
	}

	return nil
}

func maskSecrets(state *apply.ProjectState) *apply.ProjectState {
	// Create a deep copy with secrets masked
	masked := &apply.ProjectState{}
	// Copy clusters (no secrets to mask)
	if len(state.Clusters) > 0 {
		masked.Clusters = make([]types.ClusterManifest, len(state.Clusters))
		copy(masked.Clusters, state.Clusters)
	} else {
		masked.Clusters = []types.ClusterManifest{}
	}

	// Copy users with masked passwords
	for _, user := range state.DatabaseUsers {
		maskedUser := user
		maskedUser.Spec.Password = "***hidden***"
		masked.DatabaseUsers = append(masked.DatabaseUsers, maskedUser)
	}

	// Copy network access lists (no secrets to mask)
	if len(state.NetworkAccess) > 0 {
		masked.NetworkAccess = make([]types.NetworkAccessManifest, len(state.NetworkAccess))
		copy(masked.NetworkAccess, state.NetworkAccess)
	} else {
		masked.NetworkAccess = []types.NetworkAccessManifest{}
	}

	return masked
}

func validateShowOptions(opts *ShowOptions) error {
	if opts.ProjectID == "" {
		return fmt.Errorf("project ID is required")
	}

	// Validate output format
	validOutputFormats := []string{"table", "json", "yaml", "summary"}
	if !contains(validOutputFormats, strings.ToLower(opts.OutputFormat)) {
		return fmt.Errorf("invalid output format: %s (valid options: %s)", opts.OutputFormat, strings.Join(validOutputFormats, ", "))
	}

	// Validate resource type
	if opts.ResourceType != "" {
		validResourceTypes := []string{"clusters", "cluster", "users", "user", "database-users", "network-access", "network"}
		if !contains(validResourceTypes, strings.ToLower(opts.ResourceType)) {
			return fmt.Errorf("invalid resource type: %s (valid options: clusters, users, network-access)", opts.ResourceType)
		}
	}

	// Validate timeout
	if opts.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}
