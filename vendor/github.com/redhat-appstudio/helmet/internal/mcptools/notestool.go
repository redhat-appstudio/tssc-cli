package mcptools

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/deployer"
	"github.com/redhat-appstudio/helmet/internal/flags"
	"github.com/redhat-appstudio/helmet/internal/installer"
	"github.com/redhat-appstudio/helmet/internal/k8s"
	"github.com/redhat-appstudio/helmet/internal/resolver"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NotesTool a MCP tool to provide connection instructions for products. These
// products must be completely deployed before its "NOTES.txt" is generated.
type NotesTool struct {
	appName string                    // application name
	logger  *slog.Logger              // application logger
	flags   *flags.Flags              // global flags
	kube    *k8s.Kube                 // kubernetes client
	cm      *config.ConfigMapManager  // cluster configuration
	tb      *resolver.TopologyBuilder // topology builder
	job     *installer.Job            // cluster deployment job
}

var _ Interface = &NotesTool{}

const (
	// notesSuffix retrieves the connection instruction for a product suffix.
	notesSuffix = "_notes"
)

// notesHandler retrieves the Helm chart notes (NOTES.txt) for a specified Red Hat
// product. It ensures the product name is provided, checks if the cluster
// installation is in a "completed" phase, and then uses a Helm client to fetch
// and return the notes.
func (n *NotesTool) notesHandler(
	ctx context.Context,
	ctr mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Ensure the user has provided the product name.
	name := ctr.GetString(NameArg, "")
	if name == "" {
		return mcp.NewToolResultError(`
		You must inform the Red Hat product name`,
		), nil
	}

	// Check if the cluster is ready. If not, provide instructions on how to
	// proceed. The installer must be on "completed" status.
	phase, err := getInstallerPhase(ctx, n.cm, n.tb, n.job)
	currentStatus := fmt.Sprintf(`
# Current Status: %q

The cluster is not ready, use the tool %q to check the overall status and general
directions on how to proceed.`,
		phase, n.appName+statusSuffix,
	)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf(`%s

Inspecting the cluster returned the following error:

> %s`,
			currentStatus, err.Error(),
		)), nil
	}
	if phase != CompletedPhase {
		return mcp.NewToolResultText(currentStatus), nil
	}

	dep, err := n.tb.GetCollection().GetProductDependency(name)
	if err != nil {
		return mcp.NewToolResultErrorFromErr(
			fmt.Sprintf(`
Unable to find the dependency for the informed product name %q`,
				name,
			),
			err,
		), nil
	}

	hc, err := deployer.NewHelm(
		n.logger, n.flags, n.kube, dep.Namespace(), dep.Chart())
	if err != nil {
		return mcp.NewToolResultErrorFromErr(
			fmt.Sprintf(`
Error trying to instantiate a Helm client for the chart %q on namespace %q.`,
				dep.Chart().Name(),
				dep.Namespace(),
			),
			err,
		), nil
	}

	notes, err := hc.GetNotes()
	if err != nil {
		return mcp.NewToolResultErrorFromErr(
			fmt.Sprintf(`
Unable to get "NOTES.txt" for the chart %q on namespace %q.`,
				dep.Chart().Name(),
				dep.Namespace(),
			),
			err,
		), nil
	}

	return mcp.NewToolResultText(notes), nil
}

func (n *NotesTool) Init(s *server.MCPServer) {
	s.AddTools([]server.ServerTool{{
		Tool: mcp.NewTool(
			n.appName+notesSuffix,
			mcp.WithDescription(`
Retrieve the service notes, the initial coordinates to utilize services deployed
by this installer, from the informed product name.`,
			),
			mcp.WithString(
				NameArg,
				mcp.Description(`
The name of the Red Hat product to retrieve connection information.`,
				),
			),
		),
		Handler: n.notesHandler,
	}}...)
}

func NewNotesTool(
	appName string,
	logger *slog.Logger,
	f *flags.Flags,
	kube *k8s.Kube,
	cm *config.ConfigMapManager,
	tb *resolver.TopologyBuilder,
	job *installer.Job,
) *NotesTool {
	return &NotesTool{
		appName: appName,
		logger:  logger,
		flags:   f,
		kube:    kube,
		cm:      cm,
		tb:      tb,
		job:     job,
	}
}
