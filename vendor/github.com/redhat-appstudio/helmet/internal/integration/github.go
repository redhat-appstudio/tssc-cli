package integration

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/githubapp"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/google/go-github/scrape"
	"github.com/google/go-github/v80/github"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// GitHub represents the GitHub App integration attributes. It collects, validates
// and issues the attributes to the GitHub App API.
type GitHub struct {
	logger *slog.Logger         // application logger
	kube   *k8s.Kube            // kubernetes client
	client *githubapp.GitHubApp // github API client

	description string // application description
	callbackURL string // github app callback URL
	homepageURL string // github app homepage URL
	webhookURL  string // github app webhook URL
	token       string // github personal access token

	name string // application name
}

var _ Interface = &GitHub{}

// GitHubAppName key to identify the GitHubApp name.
const GitHubAppName = "name"

// PersistentFlags adds the persistent flags to the informed Cobra command.
func (g *GitHub) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.StringVar(&g.description, "description", g.description,
		"GitHub App description")
	p.StringVar(&g.callbackURL, "callback-url", g.callbackURL,
		"GitHub App callback URL")
	p.StringVar(&g.homepageURL, "homepage-url", g.homepageURL,
		"GitHub App homepage URL")
	p.StringVar(&g.webhookURL, "webhook-url", g.webhookURL,
		"GitHub App webhook URL")
	p.StringVar(&g.token, "token", g.token,
		"GitHub personal access token")

	if err := c.MarkPersistentFlagRequired("token"); err != nil {
		panic(err)
	}

	// Including GitHub App API client flags.
	g.client.PersistentFlags(c)
}

// SetArgument sets the GitHub App name.
func (g *GitHub) SetArgument(k, v string) error {
	if k != GitHubAppName {
		return fmt.Errorf("invalid argument %q (%q)", k, v)
	}
	g.name = v
	return nil
}

// LoggerWith decorates the logger with the integration flags.
func (g *GitHub) LoggerWith(logger *slog.Logger) *slog.Logger {
	return logger.With(
		"app-name", g.name,
		"callback-url", g.callbackURL,
		"webhook-url", g.webhookURL,
		"homepage-url", g.homepageURL,
		"token-len", len(g.token),
	)
}

// log logger with integration attributes.
func (g *GitHub) log() *slog.Logger {
	return g.LoggerWith(g.logger)
}

// Validate validates the integration configuration.
func (g *GitHub) Validate() error {
	return g.client.Validate()
}

// Type returns the type of the integration.
func (g *GitHub) Type() corev1.SecretType {
	return corev1.SecretTypeOpaque
}

// setClusterURLs sets the cluster URLs for the integration. It uses the TSSC
// configuration to identify Developer Hub's namespace, and queries the cluster to
// obtain its ingress domain.
func (g *GitHub) setClusterURLs(
	ctx context.Context,
	cfg *config.Config,
) error {
	developerHub, err := cfg.GetProduct(config.DeveloperHub)
	if err != nil {
		return err
	}
	ingressDomain, err := k8s.GetOpenShiftIngressDomain(ctx, g.kube)
	if err != nil {
		return err
	}

	if g.callbackURL == "" {
		g.callbackURL = fmt.Sprintf(
			"https://backstage-developer-hub-%s.%s/api/auth/github/handler/frame",
			developerHub.GetNamespace(),
			ingressDomain,
		)
	}
	if g.webhookURL == "" {
		g.webhookURL = fmt.Sprintf(
			"https://pipelines-as-code-controller-%s.%s",
			"openshift-pipelines",
			ingressDomain,
		)
	}
	if g.homepageURL == "" {
		g.homepageURL = fmt.Sprintf(
			"https://backstage-developer-hub-%s.%s",
			developerHub.GetNamespace(),
			ingressDomain,
		)
	}
	return nil
}

// generateAppManifest creates the application manifest for the GitHub-App.
func (g *GitHub) generateAppManifest() scrape.AppManifest {
	return scrape.AppManifest{
		Name: github.Ptr(g.name),
		URL:  github.Ptr(g.homepageURL),
		CallbackURLs: []string{
			g.callbackURL,
		},
		Description:    github.Ptr(g.description),
		HookAttributes: map[string]string{"url": g.webhookURL},
		Public:         github.Ptr(true),
		DefaultEvents: []string{
			"check_run",
			"check_suite",
			"commit_comment",
			"issue_comment",
			"pull_request",
			"push",
		},
		DefaultPermissions: &github.InstallationPermissions{
			// Permissions for Pipeline-as-Code.
			Checks:           github.Ptr("write"),
			Contents:         github.Ptr("write"),
			Issues:           github.Ptr("write"),
			Members:          github.Ptr("read"),
			Metadata:         github.Ptr("read"),
			OrganizationPlan: github.Ptr("read"),
			PullRequests:     github.Ptr("write"),
			// Permissions for Red Hat Developer Hub (RHDH).
			Administration: github.Ptr("write"),
			Workflows:      github.Ptr("write"),
		},
	}
}

// getCurrentGitHubUser executes a additional API call, with a new client, to
// obtain the username for the informed GitHub App hostname.
func (g *GitHub) getCurrentGitHubUser(
	ctx context.Context,
	hostname string,
) (string, error) {
	client := github.NewClient(nil).WithAuthToken(g.token)
	if hostname != "github.com" {
		baseURL := fmt.Sprintf("https://%s/api/v3/", hostname)
		uploadsURL := fmt.Sprintf("https://%s/api/uploads/", hostname)
		enterpriseClient, err := client.WithEnterpriseURLs(baseURL, uploadsURL)
		if err != nil {
			return "", err
		}
		client = enterpriseClient
	}

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", err
	}
	return user.GetLogin(), nil
}

// Data generates the GitHub App integration data after interacting with the
// service API to create the application, storing the results of this interaction.
func (g *GitHub) Data(
	ctx context.Context,
	cfg *config.Config,
) (map[string][]byte, error) {
	g.log().Info("Configuring GitHub App URLs for Developer Hub")
	err := g.setClusterURLs(ctx, cfg)
	if err != nil {
		return nil, err
	}

	g.log().Info("Generating the GitHub application manifest")
	manifest := g.generateAppManifest()

	g.log().Info("Creating the GitHub App using the service API")
	appConfig, err := g.client.Create(ctx, manifest)
	if err != nil {
		return nil, err
	}

	g.log().Info("Parsing the GitHub App endpoint URL")
	u, err := url.Parse(appConfig.GetHTMLURL())
	if err != nil {
		return nil, err
	}

	g.log().With("hostname", u.Hostname()).
		Info("Getting the current GitHub user from the application URL")
	username, err := g.getCurrentGitHubUser(ctx, u.Hostname())
	if err != nil {
		return nil, err
	}

	g.log().With("username", username).
		Debug("Generating the secret data for the GitHub App")
	return map[string][]byte{
		"clientId":      []byte(appConfig.GetClientID()),
		"clientSecret":  []byte(appConfig.GetClientSecret()),
		"createdAt":     []byte(appConfig.CreatedAt.String()),
		"externalURL":   []byte(appConfig.GetExternalURL()),
		"htmlURL":       []byte(appConfig.GetHTMLURL()),
		"host":          []byte(u.Hostname()),
		"id":            []byte(github.Stringify(appConfig.GetID())),
		"name":          []byte(appConfig.GetName()),
		"nodeId":        []byte(appConfig.GetNodeID()),
		"ownerLogin":    []byte(appConfig.Owner.GetLogin()),
		"ownerId":       []byte(github.Stringify(appConfig.Owner.GetID())),
		"pem":           []byte(appConfig.GetPEM()),
		"slug":          []byte(appConfig.GetSlug()),
		"updatedAt":     []byte(appConfig.UpdatedAt.String()),
		"webhookSecret": []byte(appConfig.GetWebhookSecret()),
		"token":         []byte(g.token),
		"username":      []byte(username),
	}, nil
}

// NewGitHub instances a new GitHub App integration.
func NewGitHub(logger *slog.Logger, kube *k8s.Kube) *GitHub {
	return &GitHub{
		logger: logger,
		kube:   kube,
		client: githubapp.NewGitHubApp(logger),
	}
}
