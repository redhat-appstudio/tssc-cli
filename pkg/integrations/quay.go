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

// defaultPublicQuayURL is the default URL for public Quay.
const defaultPublicQuayURL = "https://quay.io"

// QuayIntegration represents the RHTAP Quay integration.
type QuayIntegration struct {
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	force bool // overwrite the existing secret

	dockerconfigjson string // dockerconfig credentials
	token            string // API token credentials
	url              string // quay URL
}

// PersistentFlags sets the persistent flags for the Quay integration.
func (q *QuayIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&q.force, "force", q.force,
		"Overwrite the existing secret")

	p.StringVar(&q.dockerconfigjson, "dockerconfigjson", q.dockerconfigjson,
		"Quay dockerconfigjson, e.g. '{ \"auths\": { \"quay.io\": { \"auth\": \"****\", \"email\": \"\" }}}'")
	p.StringVar(&q.token, "token", q.token,
		"Quay API token")
	p.StringVar(&q.url, "url", q.url,
		"Quay URL")
}

// log logger with contextual information.
func (q *QuayIntegration) log() *slog.Logger {
	return q.logger.With(
		"url", q.url,
		"force", q.force,
		"dockerconfigjson-len", len(q.dockerconfigjson),
		"token-len", len(q.token),
	)
}

// Validate checks if the required configuration is set.
func (q *QuayIntegration) Validate() error {
	if q.dockerconfigjson == "" {
		return fmt.Errorf("dockerconfigjson is required")
	}
	if q.token == "" {
		return fmt.Errorf("token is required")
	}
	if q.url == "" {
		q.url = defaultPublicQuayURL
	} else {
		u, err := url.Parse(q.url)
		if err != nil {
			return fmt.Errorf("invalid url")
		}
		if !strings.HasPrefix(u.Scheme, "http") {
			return fmt.Errorf("invalid url scheme, expected one of 'http', 'https'")
		}
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the Quay integration secret
// is created on the cluster.
func (q *QuayIntegration) EnsureNamespace(ctx context.Context) error {
	feature, err := q.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	return k8s.EnsureOpenShiftProject(
		ctx,
		q.log(),
		q.kube,
		feature.GetNamespace(),
	)
}

// secretName returns the secret name for the integration. The name is "lazy"
// generated to make sure configuration is already loaded.
func (q *QuayIntegration) secretName() types.NamespacedName {
	feature, _ := q.cfg.GetFeature(config.RedHatDeveloperHub)
	return types.NamespacedName{
		Namespace: feature.GetNamespace(),
		Name:      "rhtap-quay-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (q *QuayIntegration) prepareSecret(ctx context.Context) error {
	q.log().Debug("Checking if integration secret exists")
	exists, err := k8s.SecretExists(ctx, q.kube, q.secretName())
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
			ErrSecretAlreadyExists, q.secretName().String())
	}
	q.log().Debug("Integration secret already exists, recreating it")
	return k8s.DeleteSecret(ctx, q.kube, q.secretName())
}

// store creates the secret with the integration data.
func (q *QuayIntegration) store(
	ctx context.Context,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: q.secretName().Namespace,
			Name:      q.secretName().Name,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": []byte(q.dockerconfigjson),
			"token":             []byte(q.token),
			"url":               []byte(q.url),
		},
	}
	logger := q.log().With(
		"secret-namespace", secret.GetNamespace(),
		"secret-name", secret.GetName(),
	)

	logger.Debug("Creating integration secret")
	coreClient, err := q.kube.CoreV1ClientSet(q.secretName().Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(q.secretName().Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		logger.Info("Integration secret created successfully!")
	}
	return err
}

// Create creates the Quay integration Kubernetes secret.
func (q *QuayIntegration) Create(ctx context.Context) error {
	logger := q.log()
	logger.Info("Inspecting the cluster for an existing Quay integration secret")
	if err := q.prepareSecret(ctx); err != nil {
		return err
	}
	return q.store(ctx)
}

func NewQuayIntegration(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *QuayIntegration {
	return &QuayIntegration{
		logger: logger,
		cfg:    cfg,
		kube:   kube,

		force:            false,
		dockerconfigjson: "",
		token:            "",
		url:              "",
	}
}
