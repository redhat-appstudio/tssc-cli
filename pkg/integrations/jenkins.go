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

// JenkinsIntegration represents the RHTAP Jenkins integration.
type JenkinsIntegration struct {
	logger *slog.Logger   // application logger
	cfg    *config.Config // installer configuration
	kube   *k8s.Kube      // kubernetes client

	force bool // overwrite the existing secret

	token    string // API token credentials
	url      string // Jenkins service URL
	username string // user to connect to the service
}

// PersistentFlags sets the persistent flags for the Jenkins integration.
func (j *JenkinsIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&j.force, "force", j.force,
		"Overwrite the existing secret")

	p.StringVar(&j.token, "token", j.token,
		"Jenkins API token")
	p.StringVar(&j.username, "username", j.username,
		"Jenkins user to connect to the service")
	p.StringVar(&j.url, "url", j.url,
		"Jenkins URL to the service")
}

// log logger with contextual information.
func (j *JenkinsIntegration) log() *slog.Logger {
	return j.logger.With(
		"force", j.force,
		"token-len", len(j.token),
		"url", j.url,
		"username", j.username,
	)
}

// Validate checks if the required configuration is set.
func (j *JenkinsIntegration) Validate() error {
	if j.token == "" {
		return fmt.Errorf("token is required")
	}
	if j.url == "" {
		return fmt.Errorf("url is required")
	} else {
		u, err := url.Parse(j.url)
		if err != nil {
			return fmt.Errorf("invalid url")
		}
		if !strings.HasPrefix(u.Scheme, "http") {
			return fmt.Errorf("invalid url scheme, expected one of 'http', 'https'")
		}
	}
	if j.username == "" {
		return fmt.Errorf("username is required")
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the Jenkins integration secret
// is created on the cluster.
func (j *JenkinsIntegration) EnsureNamespace(ctx context.Context) error {
	feature, err := j.cfg.GetFeature(config.RedHatDeveloperHub)
	if err != nil {
		return err
	}
	return k8s.EnsureOpenShiftProject(
		ctx,
		j.log(),
		j.kube,
		feature.GetNamespace(),
	)
}

// secretName returns the secret name for the integration. The name is "lazy"
// generated to make sure configuration is already loaded.
func (j *JenkinsIntegration) secretName() types.NamespacedName {
	feature, _ := j.cfg.GetFeature(config.RedHatDeveloperHub)
	return types.NamespacedName{
		Namespace: feature.GetNamespace(),
		Name:      "rhtap-jenkins-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (j *JenkinsIntegration) prepareSecret(ctx context.Context) error {
	j.log().Debug("Checking if integration secret exists")
	exists, err := k8s.SecretExists(ctx, j.kube, j.secretName())
	if err != nil {
		return err
	}
	if !exists {
		j.log().Debug("Integration secret does not exist")
		return nil
	}
	if !j.force {
		j.log().Debug("Integration secret already exists")
		return fmt.Errorf("%w: %s",
			ErrSecretAlreadyExists, j.secretName().String())
	}
	j.log().Debug("Integration secret already exists, recreating it")
	return k8s.DeleteSecret(ctx, j.kube, j.secretName())
}

// store creates the secret with the integration data.
func (j *JenkinsIntegration) store(
	ctx context.Context,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: j.secretName().Namespace,
			Name:      j.secretName().Name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"baseUrl":  []byte(j.url),
			"token":    []byte(j.token),
			"username": []byte(j.username),
		},
	}
	logger := j.log().With(
		"secret-namespace", secret.GetNamespace(),
		"secret-name", secret.GetName(),
	)

	logger.Debug("Creating integration secret")
	coreClient, err := j.kube.CoreV1ClientSet(j.secretName().Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(j.secretName().Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		logger.Info("Integration secret created successfully!")
	}
	return err
}

// Create creates the Jenkins integration Kubernetes secret.
func (j *JenkinsIntegration) Create(ctx context.Context) error {
	logger := j.log()
	logger.Info("Inspecting the cluster for an existing Jenkins integration secret")
	if err := j.prepareSecret(ctx); err != nil {
		return err
	}
	return j.store(ctx)
}

func NewJenkinsIntegration(
	logger *slog.Logger,
	cfg *config.Config,
	kube *k8s.Kube,
) *JenkinsIntegration {
	return &JenkinsIntegration{
		logger: logger,
		cfg:    cfg,
		kube:   kube,

		force:    false,
		token:    "",
		url:      "",
		username: "",
	}
}
