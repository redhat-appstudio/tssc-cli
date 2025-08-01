package integrations

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/redhat-appstudio/tssc/pkg/config"
	"github.com/redhat-appstudio/tssc/pkg/k8s"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// defaultPublicQuayURL is the default URL for public Quay.
const defaultPublicQuayURL = "https://quay.io"

// QuayIntegration represents the TSSC Quay integration.
type QuayIntegration struct {
	logger *slog.Logger // application logger
	kube   *k8s.Kube    // kubernetes client

	force bool // overwrite the existing secret

	dockerconfigjson         string // dockerconfig credentials
	dockerconfigjsonreadonly string // dockerconfigjsonreadonly credentials
	token                    string // API token credentials
	url                      string // quay URL
}

// PersistentFlags sets the persistent flags for the Quay integration.
func (q *QuayIntegration) PersistentFlags(c *cobra.Command) {
	p := c.PersistentFlags()

	p.BoolVar(&q.force, "force", q.force,
		"Overwrite the existing secret")

	p.StringVar(&q.dockerconfigjson, "dockerconfigjson", q.dockerconfigjson,
		"Quay dockerconfigjson, e.g. '{ \"auths\": { \"quay.io\": { \"auth\": \"****\", \"email\": \"\" }}}'")
	p.StringVar(&q.dockerconfigjsonreadonly, "dockerconfigjsonreadonly", q.dockerconfigjsonreadonly,
		"Quay dockerconfigjson for read only account, e.g. '{ \"auths\": { \"quay.io\": { \"auth\": \"****\", \"email\": \"\" }}}")
	p.StringVar(&q.token, "token", q.token,
		"Quay API token")
	p.StringVar(&q.url, "url", q.url,
		"Quay URL")

	for _, f := range []string{"dockerconfigjson", "token", "url"} {
		if err := c.MarkPersistentFlagRequired(f); err != nil {
			panic(err)
		}
	}
}

// log logger with contextual information.
func (q *QuayIntegration) log() *slog.Logger {
	return q.logger.With(
		"url", q.url,
		"force", q.force,
		"dockerconfigjson-len", len(q.dockerconfigjson),
		"dockerconfigjsonreadonly-len", len(q.dockerconfigjsonreadonly),
		"token-len", len(q.token),
	)
}

// Validate checks if the required configuration is set.
func (q *QuayIntegration) Validate() error {
	u, err := url.Parse(q.url)
	if err != nil {
		return fmt.Errorf("invalid url: %s", err)
	}
	if !strings.HasPrefix(u.Scheme, "http") {
		return fmt.Errorf("invalid url scheme, expected one of 'http', 'https'")
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the Quay integration secret
// is created on the cluster.
func (q *QuayIntegration) EnsureNamespace(
	ctx context.Context,
	cfg *config.Config,
) error {
	return k8s.EnsureOpenShiftProject(
		ctx,
		q.log(),
		q.kube,
		cfg.Installer.Namespace,
	)
}

// secretName returns the secret name for the integration. The name is "lazy"
// generated to make sure configuration is already loaded.
func (q *QuayIntegration) secretName(cfg *config.Config) types.NamespacedName {
	return types.NamespacedName{
		Namespace: cfg.Installer.Namespace,
		Name:      "tssc-quay-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (q *QuayIntegration) prepareSecret(
	ctx context.Context,
	cfg *config.Config,
) error {
	q.log().Debug("Checking if integration secret exists")
	exists, err := k8s.SecretExists(ctx, q.kube, q.secretName(cfg))
	if err != nil {
		return err
	}
	if !exists {
		q.log().Debug("Integration secret does not exist")
		return nil
	}
	if !q.force {
		q.log().Debug("Integration secret already exists")
		return fmt.Errorf("%w: %s",
			ErrSecretAlreadyExists, q.secretName(cfg).String())
	}
	q.log().Debug("Integration secret already exists, recreating it")
	return k8s.DeleteSecret(ctx, q.kube, q.secretName(cfg))
}

// store creates the secret with the integration data.
func (q *QuayIntegration) store(
	ctx context.Context,
	cfg *config.Config,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: q.secretName(cfg).Namespace,
			Name:      q.secretName(cfg).Name,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson":         []byte(q.dockerconfigjson),
			".dockerconfigjsonreadonly": []byte(q.dockerconfigjsonreadonly),
			"token":                     []byte(q.token),
			"url":                       []byte(q.url),
		},
	}
	logger := q.log().With(
		"secret-namespace", secret.GetNamespace(),
		"secret-name", secret.GetName(),
	)

	logger.Debug("Creating integration secret")
	coreClient, err := q.kube.CoreV1ClientSet(q.secretName(cfg).Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(q.secretName(cfg).Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		logger.Info("Integration secret created successfully!")
	}
	return err
}

// Create creates the Quay integration Kubernetes secret.
func (q *QuayIntegration) Create(
	ctx context.Context,
	cfg *config.Config,
) error {
	logger := q.log()
	logger.Info("Inspecting the cluster for an existing Quay integration secret")
	if err := q.prepareSecret(ctx, cfg); err != nil {
		return err
	}
	return q.store(ctx, cfg)
}

func NewQuayIntegration(
	logger *slog.Logger,
	kube *k8s.Kube,
) *QuayIntegration {
	return &QuayIntegration{
		logger: logger,
		kube:   kube,

		force:                    false,
		dockerconfigjson:         "",
		dockerconfigjsonreadonly: "",
		token:                    "",
		url:                      defaultPublicQuayURL,
	}
}
