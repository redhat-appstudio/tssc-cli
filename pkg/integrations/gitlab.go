package integrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/tssc/pkg/config"
	"github.com/redhat-appstudio/tssc/pkg/k8s"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// defaultPublicGitLabHost is the default host for public GitLab.
const defaultPublicGitLabHost = "gitlab.com"

// GitLabIntegration represents the TSSC GitLab integration.
type GitLabIntegration struct {
	logger *slog.Logger // application logger
	kube   *k8s.Kube    // kubernetes client

	force bool // overwrite the existing secret

	host         string // GitLab host
	clientId     string // GitLab application client id
	clientSecret string // GitLab application client secret
	token        string // API token credentials
	group        string // GitLab group name
}

// PersistentFlags sets the persistent flags for the GitLab integration.
func (g *GitLabIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&g.force, "force", g.force,
		"Overwrite the existing secret")

	p.StringVar(&g.host, "host", g.host,
		"GitLab host, defaults to 'gitlab.com'")
	p.StringVar(&g.clientId, "app-id", g.clientId,
		"GitLab application client id")
	p.StringVar(&g.clientSecret, "app-secret", g.clientSecret,
		"GitLab application client secret")
	p.StringVar(&g.token, "token", g.token,
		"GitLab API token")
	p.StringVar(&g.group, "group", g.group,
		"GitLab group name")
}

// log logger with contextual information.
func (g *GitLabIntegration) log() *slog.Logger {
	return g.logger.With(
		"force", g.force,
		"host", g.host,
		"clientId", g.clientId,
		"clientSecret-len", len(g.clientSecret),
		"token-len", len(g.token),
		"group", g.group,
	)
}

// Validate checks if the required configuration is set.
func (g *GitLabIntegration) Validate() error {
	if g.host == "" {
		g.host = defaultPublicGitLabHost
	}
	if g.clientId != "" && g.clientSecret == "" {
		return fmt.Errorf("app-secret is required when id is specified")
	}
	if g.clientId == "" && g.clientSecret != "" {
		return fmt.Errorf("app-id is required when app-secret is specified")
	}
	if g.token == "" {
		return fmt.Errorf("token is required")
	}
	if g.group == "" {
		return fmt.Errorf("group is required")
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the GitLab integration secret
// is created on the cluster.
func (g *GitLabIntegration) EnsureNamespace(
	ctx context.Context,
	cfg *config.Config,
) error {
	return k8s.EnsureOpenShiftProject(
		ctx,
		g.log(),
		g.kube,
		cfg.Installer.Namespace,
	)
}

// secretName returns the secret name for the integration. The name is "lazy"
// generated to make sure configuration is already loaded.
func (g *GitLabIntegration) secretName(cfg *config.Config) types.NamespacedName {
	return types.NamespacedName{
		Namespace: cfg.Installer.Namespace,
		Name:      "tssc-gitlab-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (g *GitLabIntegration) prepareSecret(
	ctx context.Context,
	cfg *config.Config,
) error {
	g.log().Debug("Checking if integration secret exists")
	exists, err := k8s.SecretExists(ctx, g.kube, g.secretName(cfg))
	if err != nil {
		return err
	}
	if !exists {
		g.log().Debug("Integration secret does not exist")
		return nil
	}
	if !g.force {
		g.log().Debug("Integration secret already exists")
		return fmt.Errorf("%w: %s",
			ErrSecretAlreadyExists, g.secretName(cfg).String())
	}
	g.log().Debug("Integration secret already exists, recreating it")
	return k8s.DeleteSecret(ctx, g.kube, g.secretName(cfg))
}

// store creates the secret with the integration data.
func (g *GitLabIntegration) store(
	ctx context.Context,
	cfg *config.Config,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: g.secretName(cfg).Namespace,
			Name:      g.secretName(cfg).Name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"clientId":     []byte(g.clientId),
			"clientSecret": []byte(g.clientSecret),
			"host":         []byte(g.host),
			"token":        []byte(g.token),
			"group":        []byte(g.group),
		},
	}
	logger := g.log().With(
		"secret-namespace", secret.GetNamespace(),
		"secret-name", secret.GetName(),
	)

	logger.Debug("Creating integration secret")
	coreClient, err := g.kube.CoreV1ClientSet(g.secretName(cfg).Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(g.secretName(cfg).Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		logger.Info("Integration secret created successfully!")
	}
	return err
}

// Create creates the GitLab integration Kubernetes secret.
func (g *GitLabIntegration) Create(
	ctx context.Context,
	cfg *config.Config,
) error {
	logger := g.log()
	logger.Info("Inspecting the cluster for an existing GitLab integration secret")
	if err := g.prepareSecret(ctx, cfg); err != nil {
		return err
	}
	return g.store(ctx, cfg)
}

func NewGitLabIntegration(
	logger *slog.Logger,
	kube *k8s.Kube,
) *GitLabIntegration {
	return &GitLabIntegration{
		logger: logger,
		kube:   kube,

		force:        false,
		host:         "",
		clientId:     "",
		clientSecret: "",
		token:        "",
		group:        "",
	}
}
