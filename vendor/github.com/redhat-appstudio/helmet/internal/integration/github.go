package integration

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/redhat-appstudio/helmet/api/integrations"
	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/githubapp"
	"github.com/redhat-appstudio/helmet/internal/runcontext"

	"github.com/google/go-github/scrape"
	"github.com/google/go-github/v80/github"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
)

// URLProvider defines an interface for providing GitHub App URLs.
// Implementations can use the RunContext, config.Config and context to derive
// URLs from cluster configuration or other sources.
type URLProvider interface {
	// GetCallbackURL returns the GitHub App callback URL.
	GetCallbackURL(ctx context.Context, runCtx *runcontext.RunContext, cfg *config.Config) (string, error)
	// GetHomepageURL returns the GitHub App homepage URL.
	GetHomepageURL(ctx context.Context, runCtx *runcontext.RunContext, cfg *config.Config) (string, error)
	// GetWebhookURL returns the GitHub App webhook URL.
	GetWebhookURL(ctx context.Context, runCtx *runcontext.RunContext, cfg *config.Config) (string, error)
}

// GitHub represents the GitHub App integration attributes. It collects, validates
// and issues the attributes to the GitHub App API.
type GitHub struct {
	logger *slog.Logger         // application logger
	client *githubapp.GitHubApp // github API client

	urlProvider integrations.URLProvider // optional; when set, adapter is built in setClusterURLs
	description string                   // application description
	callbackURL string                   // github app callback URL
	homepageURL string                   // github app homepage URL
	webhookURL  string                   // github app webhook URL
	token       string                   // github personal access token

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

// SetURLProvider sets an optional URLProvider (api/integrations). When set,
// an adapter is built in setClusterURLs from runCtx and cfg so the provider
// receives only integrations.IntegrationContext. The provider is consulted for
// any URL not already set by flags.
func (g *GitHub) SetURLProvider(provider integrations.URLProvider) {
	g.urlProvider = provider
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

// setClusterURLs resolves GitHub App URLs from flags first, then from the
// optional URL provider (via adapter). It validates that required URLs (webhook, homepage) are set.
func (g *GitHub) setClusterURLs(
	ctx context.Context,
	runCtx *runcontext.RunContext,
	cfg *config.Config,
) error {
	if g.urlProvider != nil {
		provider := newURLProviderAdapter(g.urlProvider, runCtx, cfg)
		if g.callbackURL == "" {
			url, err := provider.GetCallbackURL(ctx, runCtx, cfg)
			if err != nil {
				return fmt.Errorf("get callback URL: %w", err)
			}
			g.callbackURL = url
		}
		if g.webhookURL == "" {
			url, err := provider.GetWebhookURL(ctx, runCtx, cfg)
			if err != nil {
				return fmt.Errorf("get webhook URL: %w", err)
			}
			g.webhookURL = url
		}
		if g.homepageURL == "" {
			url, err := provider.GetHomepageURL(ctx, runCtx, cfg)
			if err != nil {
				return fmt.Errorf("get homepage URL: %w", err)
			}
			g.homepageURL = url
		}
	}

	if g.webhookURL == "" || g.homepageURL == "" {
		return fmt.Errorf("GitHub App webhook and homepage URLs must be provided via flags or URLProvider")
	}

	return nil
}

// generateAppManifest creates the application manifest for the GitHub-App.
func (g *GitHub) generateAppManifest() scrape.AppManifest {
	var callbackURLs []string
	if g.callbackURL != "" {
		callbackURLs = []string{g.callbackURL}
	}
	return scrape.AppManifest{
		Name:           github.Ptr(g.name),
		URL:            github.Ptr(g.homepageURL),
		CallbackURLs:   callbackURLs,
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
	runCtx *runcontext.RunContext,
	cfg *config.Config,
) (map[string][]byte, error) {
	g.log().Info("Configuring GitHub App URLs")
	err := g.setClusterURLs(ctx, runCtx, cfg)
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
func NewGitHub(logger *slog.Logger) *GitHub {
	return &GitHub{
		logger: logger,
		client: githubapp.NewGitHubApp(logger),
	}
}
