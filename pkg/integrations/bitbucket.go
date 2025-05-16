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

// defaultPublicBitBucketHost is the default host for public BitBucket.
const defaultPublicBitBucketHost = "bitbucket.org"

// BitBucketIntegration represents the TSSC BitBucket integration.
type BitBucketIntegration struct {
	logger *slog.Logger // application logger
	kube   *k8s.Kube    // kubernetes client

	force bool // overwrite the existing secret

	appPassword string // BitBucket application password
	host        string // BitBucket host
	username    string // BitBucket username
}

// PersistentFlags sets the persistent flags for the BitBucket integration.
func (g *BitBucketIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&g.force, "force", g.force,
		"Overwrite the existing secret")

	p.StringVar(&g.appPassword, "app-password", g.appPassword,
		"BitBucket application password")
	p.StringVar(&g.host, "host", g.host,
		"BitBucket host, defaults to 'bitbucket.org'")
	p.StringVar(&g.username, "username", g.username,
		"BitBucket username")
}

// log logger with contextual information.
func (g *BitBucketIntegration) log() *slog.Logger {
	return g.logger.With(
		"force", g.force,
		"host", g.host,
		"app-password", len(g.appPassword),
		"username", g.username,
	)
}

// Validate checks if the required configuration is set.
func (g *BitBucketIntegration) Validate() error {
	if g.appPassword == "" {
		return fmt.Errorf("app-password is required")
	}
	if g.host == "" {
		g.host = defaultPublicBitBucketHost
	}
	if g.username == "" {
		return fmt.Errorf("username is required")
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the BitBucket integration
// secret is created on the cluster.
func (g *BitBucketIntegration) EnsureNamespace(
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
func (g *BitBucketIntegration) secretName(cfg *config.Config) types.NamespacedName {
	return types.NamespacedName{
		Namespace: cfg.Installer.Namespace,
		Name:      "tssc-bitbucket-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (g *BitBucketIntegration) prepareSecret(
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
func (g *BitBucketIntegration) store(
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
			"appPassword": []byte(g.appPassword),
			"host":        []byte(g.host),
			"username":    []byte(g.username),
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

// Create creates the BitBucket integration Kubernetes secret.
func (g *BitBucketIntegration) Create(
	ctx context.Context,
	cfg *config.Config,
) error {
	logger := g.log()
	logger.Info("Inspecting the cluster for an existing BitBucket integration secret")
	if err := g.prepareSecret(ctx, cfg); err != nil {
		return err
	}
	return g.store(ctx, cfg)
}

func NewBitBucketIntegration(
	logger *slog.Logger,
	kube *k8s.Kube,
) *BitBucketIntegration {
	return &BitBucketIntegration{
		logger: logger,
		kube:   kube,

		force:       false,
		appPassword: "",
		host:        "",
		username:    "",
	}
}
