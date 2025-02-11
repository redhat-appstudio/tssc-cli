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

// NexusIntegration represents the RHTAP Nexus integration.
type NexusIntegration struct {
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	force bool // overwrite the existing secret

	dockerconfigjson string // dockerconfig credentials
	url              string // nexus URL
}

// PersistentFlags sets the persistent flags for the Nexus integration.
func (n *NexusIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&n.force, "force", n.force,
		"Overwrite the existing secret")

	p.StringVar(&n.dockerconfigjson, "dockerconfigjson", n.dockerconfigjson,
		"Nexus dockerconfigjson, e.g. '{ \"auths\": { \"****\": { \"auth\": \"****\", \"email\": \"\" }}}'")
	p.StringVar(&n.url, "url", n.url,
		"Nexus URL")
}

// log logger with contextual information.
func (n *NexusIntegration) log() *slog.Logger {
	return n.logger.With(
		"url", n.url,
		"force", n.force,
		"dockerconfigjson-len", len(n.dockerconfigjson),
	)
}

// Validate checks if the required configuration is set.
func (n *NexusIntegration) Validate() error {
	if n.dockerconfigjson == "" {
		return fmt.Errorf("dockerconfigjson is required")
	}
	if n.url == "" {
		return fmt.Errorf("url is required")
	} else {
		u, err := url.Parse(n.url)
		if err != nil {
			return fmt.Errorf("invalid url")
		}
		if !strings.HasPrefix(u.Scheme, "http") {
			return fmt.Errorf("invalid url scheme, expected one of 'http', 'https'")
		}
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the Nexus integration secret
// is created on the cluster.
func (n *NexusIntegration) EnsureNamespace(ctx context.Context) error {
	return k8s.EnsureOpenShiftProject(
		ctx,
		n.log(),
		n.kube,
		n.cfg.Installer.Namespace,
	)
}

// secretName returns the secret name for the integration. The name is "lazy"
// generated to make sure configuration is already loaded.
func (n *NexusIntegration) secretName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: n.cfg.Installer.Namespace,
		Name:      "rhtap-nexus-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (n *NexusIntegration) prepareSecret(ctx context.Context) error {
	n.log().Debug("Checking if integration secret exists")
	exists, err := k8s.SecretExists(ctx, n.kube, n.secretName())
	if err != nil {
		return err
	}
	if !exists {
		n.log().Debug("Integration secret does not exist")
		return nil
	}
	if !n.force {
		n.log().Debug("Integration secret already exists")
		return fmt.Errorf("%w: %s",
			ErrSecretAlreadyExists, n.secretName().String())
	}
	n.log().Debug("Integration secret already exists, recreating it")
	return k8s.DeleteSecret(ctx, n.kube, n.secretName())
}

// store creates the secret with the integration data.
func (n *NexusIntegration) store(
	ctx context.Context,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: n.secretName().Namespace,
			Name:      n.secretName().Name,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte(n.dockerconfigjson),
			"url":               []byte(n.url),
		},
	}
	logger := n.log().With(
		"secret-namespace", secret.GetNamespace(),
		"secret-name", secret.GetName(),
	)

	logger.Debug("Creating integration secret")
	coreClient, err := n.kube.CoreV1ClientSet(n.secretName().Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(n.secretName().Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		logger.Info("Integration secret created successfully!")
	}
	return err
}

// Create creates the Nexus integration Kubernetes secret.
func (n *NexusIntegration) Create(ctx context.Context) error {
	logger := n.log()
	logger.Info("Inspecting the cluster for an existing Nexus integration secret")
	if err := n.prepareSecret(ctx); err != nil {
		return err
	}
	return n.store(ctx)
}

func NewNexusIntegration(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *NexusIntegration {
	return &NexusIntegration{
		logger: logger,
		cfg:    cfg,
		kube:   kube,

		force:            false,
		dockerconfigjson: "",
		url:              "",
	}
}
