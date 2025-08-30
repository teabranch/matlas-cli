package atlas

import (
	"github.com/spf13/cobra"

	"github.com/teabranch/matlas-cli/cmd/atlas/alerts"
	"github.com/teabranch/matlas-cli/cmd/atlas/clusters"
	"github.com/teabranch/matlas-cli/cmd/atlas/network"
	networkcontainers "github.com/teabranch/matlas-cli/cmd/atlas/network-containers"
	networkpeering "github.com/teabranch/matlas-cli/cmd/atlas/network-peering"
	"github.com/teabranch/matlas-cli/cmd/atlas/projects"
	"github.com/teabranch/matlas-cli/cmd/atlas/search"
	"github.com/teabranch/matlas-cli/cmd/atlas/users"
	vpcendpoints "github.com/teabranch/matlas-cli/cmd/atlas/vpc-endpoints"
)

// NewAtlasCmd creates the atlas command with all its subcommands
func NewAtlasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "atlas",
		Short:        "Manage MongoDB Atlas resources",
		Long:         "Manage MongoDB Atlas projects, clusters, database users, and network access lists",
		SilenceUsage: true,
	}

	// Add subcommands
	cmd.AddCommand(projects.NewProjectsCmd())
	cmd.AddCommand(clusters.NewClustersCmd())
	cmd.AddCommand(users.NewUsersCmd())
	cmd.AddCommand(network.NewNetworkCmd())
	cmd.AddCommand(vpcendpoints.NewVPCEndpointsCmd())
	cmd.AddCommand(networkpeering.NewNetworkPeeringCmd())
	cmd.AddCommand(networkcontainers.NewNetworkContainersCmd())
	cmd.AddCommand(search.NewSearchCmd())
	cmd.AddCommand(alerts.NewAlertsCmd())
	cmd.AddCommand(alerts.NewAlertConfigurationsCmd())

	return cmd
}
