package integrations

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ArtifactoryIntegration represents the TSSC Artifactory integration.
type ArtifactoryIntegration struct {
	logger *slog.Logger // application logger
	kube   *k8s.Kube    // kubernetes client

	force bool // overwrite the existing secret

	dockerconfigjson string // dockerconfig credentials
	token            string // API token credentials
	url              string // artifactory URL
}

// PersistentFlags sets the persistent flags for the Artifactory integration.
func (a *ArtifactoryIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&a.force, "force", a.force,
		"Overwrite the existing secret")

	p.StringVar(&a.dockerconfigjson, "dockerconfigjson", a.dockerconfigjson,
		"Artifactory dockerconfigjson, e.g. '{ \"auths\": { \"****\": { \"auth\": \"****\", \"email\": \"\" }}}'")
	p.StringVar(&a.token, "token", a.token,
		"Artifactory API token")
	p.StringVar(&a.url, "url", a.url,
		"Artifactory URL")
}

// log logger with contextual information.
func (a *ArtifactoryIntegration) log() *slog.Logger {
	return a.logger.With(
		"url", a.url,
		"force", a.force,
		"dockerconfigjson-len", len(a.dockerconfigjson),
		"token-len", len(a.token),
	)
}

// Validate checks if the required configuration is set.
func (a *ArtifactoryIntegration) Validate() error {
	if a.dockerconfigjson == "" {
		return fmt.Errorf("dockerconfigjson is required")
	}
	if a.token == "" {
		return fmt.Errorf("token is required")
	}
	if a.url == "" {
		return fmt.Errorf("url is required")
	} else {
		u, err := url.Parse(a.url)
		if err != nil {
			return fmt.Errorf("invalid url")
		}
		if !strings.HasPrefix(u.Scheme, "http") {
			return fmt.Errorf("invalid url scheme, expected one of 'http', 'https'")
		}
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the Artifactory integration secret
// is created on the cluster.
func (a *ArtifactoryIntegration) EnsureNamespace(
	ctx context.Context,
	cfg *config.Config,
) error {
	return k8s.EnsureOpenShiftProject(
		ctx,
		a.log(),
		a.kube,
		cfg.Installer.Namespace,
	)
}

// secretName returns the secret name for the integration. The name is "lazy"
// generated to make sure configuration is already loaded.
func (a *ArtifactoryIntegration) secretName(
	cfg *config.Config,
) types.NamespacedName {
	return types.NamespacedName{
		Namespace: cfg.Installer.Namespace,
		Name:      "tssc-artifactory-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (a *ArtifactoryIntegration) prepareSecret(
	ctx context.Context,
	cfg *config.Config,
) error {
	a.log().Debug("Checking if integration secret exists")
	exists, err := k8s.SecretExists(ctx, a.kube, a.secretName(cfg))
	if err != nil {
		return err
	}
	if !exists {
		a.log().Debug("Integration secret does not exist")
		return nil
	}
	if !a.force {
		a.log().Debug("Integration secret already exists")
		return fmt.Errorf("%w: %s",
			ErrSecretAlreadyExists, a.secretName(cfg).String())
	}
	a.log().Debug("Integration secret already exists, recreating it")
	return k8s.DeleteSecret(ctx, a.kube, a.secretName(cfg))
}

// store creates the secret with the integration data.
func (a *ArtifactoryIntegration) store(
	ctx context.Context,
	cfg *config.Config,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: a.secretName(cfg).Namespace,
			Name:      a.secretName(cfg).Name,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte(a.dockerconfigjson),
			"token":             []byte(a.token),
			"url":               []byte(a.url),
		},
	}
	logger := a.log().With(
		"secret-namespace", secret.GetNamespace(),
		"secret-name", secret.GetName(),
	)

	logger.Debug("Creating integration secret")
	coreClient, err := a.kube.CoreV1ClientSet(a.secretName(cfg).Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(a.secretName(cfg).Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		logger.Info("Integration secret created successfully!")
	}
	return err
}

// Create creates the Artifactory integration Kubernetes secret.
func (a *ArtifactoryIntegration) Create(
	ctx context.Context,
	cfg *config.Config,
) error {
	logger := a.log()
	logger.Info("Inspecting the cluster for an existing Artifactory integration secret")
	if err := a.prepareSecret(ctx, cfg); err != nil {
		return err
	}
	return a.store(ctx, cfg)
}

func NewArtifactoryIntegration(
	logger *slog.Logger,
	kube *k8s.Kube,
) *ArtifactoryIntegration {
	return &ArtifactoryIntegration{
		logger: logger,
		kube:   kube,

		force:            false,
		dockerconfigjson: "",
		token:            "",
		url:              "",
	}
}
