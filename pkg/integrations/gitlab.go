package integrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// defaultPublicGitLabHost is the default host for public GitLab.
const defaultPublicGitLabHost = "gitlab.com"

// GitLabIntegration represents the RHTAP GitLab integration.
type GitLabIntegration struct {
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	force bool // overwrite the existing secret

	host  string // GitLab host
	token string // API token credentials
}

// PersistentFlags sets the persistent flags for the GitLab integration.
func (g *GitLabIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&g.force, "force", g.force,
		"Overwrite the existing secret")

	p.StringVar(&g.host, "host", g.host,
		"GitLab host, defaults to 'gitlab.com'")
	p.StringVar(&g.token, "token", g.token,
		"GitLab API token")
}

// log logger with contextual information.
func (g *GitLabIntegration) log() *slog.Logger {
	return g.logger.With(
		"force", g.force,
		"host", g.host,
		"token-len", len(g.token),
	)
}

// Validate checks if the required configuration is set.
func (g *GitLabIntegration) Validate() error {
	if g.host == "" {
		g.host = defaultPublicGitLabHost
	}
	if g.token == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the GitLab integration secret
// is created on the cluster.
func (g *GitLabIntegration) EnsureNamespace(ctx context.Context) error {
	feature, err := g.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	return k8s.EnsureOpenShiftProject(
		ctx,
		g.log(),
		g.kube,
		feature.GetNamespace(),
	)
}

// secretName returns the secret name for the integration. The name is "lazy"
// generated to make sure configuration is already loaded.
func (g *GitLabIntegration) secretName() types.NamespacedName {
	feature, _ := g.cfg.GetFeature(config.RedHatDeveloperHub)
	return types.NamespacedName{
		Namespace: feature.GetNamespace(),
		Name:      "rhtap-gitlab-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (g *GitLabIntegration) prepareSecret(ctx context.Context) error {
	g.log().Debug("Checking if integration secret exists")
	exists, err := k8s.SecretExists(ctx, g.kube, g.secretName())
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
			ErrSecretAlreadyExists, g.secretName().String())
	}
	g.log().Debug("Integration secret already exists, recreating it")
	return k8s.DeleteSecret(ctx, g.kube, g.secretName())
}

// store creates the secret with the integration data.
func (g *GitLabIntegration) store(
	ctx context.Context,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: g.secretName().Namespace,
			Name:      g.secretName().Name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"host":  []byte(g.host),
			"token": []byte(g.token),
		},
	}
	logger := g.log().With(
		"secret-namespace", secret.GetNamespace(),
		"secret-name", secret.GetName(),
	)

	logger.Debug("Creating integration secret")
	coreClient, err := g.kube.CoreV1ClientSet(g.secretName().Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(g.secretName().Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		logger.Info("Integration secret created successfully!")
	}
	return err
}

// Create creates the GitLab integration Kubernetes secret.
func (g *GitLabIntegration) Create(ctx context.Context) error {
	logger := g.log()
	logger.Info("Inspecting the cluster for an existing GitLab integration secret")
	if err := g.prepareSecret(ctx); err != nil {
		return err
	}
	return g.store(ctx)
}

func NewGitLabIntegration(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *GitLabIntegration {
	return &GitLabIntegration{
		logger: logger,
		cfg:    cfg,
		kube:   kube,

		force: false,
		host:  "",
		token: "",
	}
}
