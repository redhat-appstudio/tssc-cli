package integrations

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TASIntegration represents the TSSC TAS integration.
type TASIntegration struct {
	logger *slog.Logger // application logger
	kube   *k8s.Kube    // kubernetes client

	force bool // overwrite the existing secret

	rekorURL string // TAS rekor url
	tufURL   string // TAS tuf url
}

// PersistentFlags sets the persistent flags for the TAS integration.
func (t *TASIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&t.force, "force", t.force,
		"Overwrite the existing secret")

	p.StringVar(&t.rekorURL, "rekor-url", t.rekorURL,
		"TAS rekor url")
	p.StringVar(&t.tufURL, "tuf-url", t.tufURL,
		"TAS tuf url")
}

// log logger with contextual information.
func (t *TASIntegration) log() *slog.Logger {
	return t.logger.With(
		"force", t.force,
		"rekor-url", t.rekorURL,
		"tuf-url", len(t.tufURL),
	)
}

// Validate checks if the required configuration is set.
func (t *TASIntegration) Validate() error {
	if t.rekorURL == "" {
		return fmt.Errorf("rekor-url is required")
	}
	if !strings.Contains(t.rekorURL, "://") {
		return fmt.Errorf("invalid rekor url, the protocol should be specified")
	}
	if t.tufURL == "" {
		return fmt.Errorf("tuf url is required")
	}
	if !strings.Contains(t.tufURL, "://") {
		return fmt.Errorf("invalid tuf url, the protocol should be specified")
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the TAS integration secret
// is created on the cluster.
func (t *TASIntegration) EnsureNamespace(
	ctx context.Context,
	cfg *config.Config,
) error {
	return k8s.EnsureOpenShiftProject(
		ctx,
		t.log(),
		t.kube,
		cfg.Installer.Namespace,
	)
}

// secretName returns the secret name for the integration. The name is "lazy"
// generated to make sure configuration is already loaded.
func (t *TASIntegration) secretName(cfg *config.Config) types.NamespacedName {
	return types.NamespacedName{
		Namespace: cfg.Installer.Namespace,
		Name:      "tssc-tas-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (t *TASIntegration) prepareSecret(
	ctx context.Context,
	cfg *config.Config,
) error {
	t.log().Debug("Checking if integration secret exists")
	exists, err := k8s.SecretExists(ctx, t.kube, t.secretName(cfg))
	if err != nil {
		return err
	}
	if !exists {
		t.log().Debug("Integration secret does not exist")
		return nil
	}
	if !t.force {
		t.log().Debug("Integration secret already exists")
		return fmt.Errorf("%w: %s",
			ErrSecretAlreadyExists, t.secretName(cfg).String())
	}
	t.log().Debug("Integration secret already exists, recreating it")
	return k8s.DeleteSecret(ctx, t.kube, t.secretName(cfg))
}

// store creates the secret with the integration data.
func (t *TASIntegration) store(
	ctx context.Context,
	cfg *config.Config,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: t.secretName(cfg).Namespace,
			Name:      t.secretName(cfg).Name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"rekor_url": []byte(t.rekorURL),
			"tuf_url":   []byte(t.tufURL),
		},
	}
	logger := t.log().With(
		"secret-namespace", secret.GetNamespace(),
		"secret-name", secret.GetName(),
	)

	logger.Debug("Creating integration secret")
	coreClient, err := t.kube.CoreV1ClientSet(t.secretName(cfg).Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(t.secretName(cfg).Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		logger.Info("Integration secret created successfully!")
	}
	return err
}

// Create creates the TAS integration Kubernetes secret.
func (t *TASIntegration) Create(
	ctx context.Context,
	cfg *config.Config,
) error {
	logger := t.log()
	logger.Info("Inspecting the cluster for an existing TAS integration secret")
	if err := t.prepareSecret(ctx, cfg); err != nil {
		return err
	}
	return t.store(ctx, cfg)
}

func NewTASIntegration(
	logger *slog.Logger,
	kube *k8s.Kube,
) *TASIntegration {
	return &TASIntegration{
		logger: logger,
		kube:   kube,

		force:    false,
		rekorURL: "",
		tufURL:   "",
	}
}
