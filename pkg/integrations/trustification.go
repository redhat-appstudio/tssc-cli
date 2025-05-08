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

// TrustificationIntegration represents the RHTAP Trustification integration.
type TrustificationIntegration struct {
	logger *slog.Logger // application logger
	kube   *k8s.Kube    // kubernetes client

	force bool // overwrite the existing secret

	bombasticAPIURL           string // URL of the BOMbastic api host
	oidcIssuerURL             string // URL of the OIDC token issuer
	oidcClientId              string // OIDC client ID
	oidcClientSecret          string // OIDC client secret
	supportedCycloneDXVersion string // CycloneDX supported version.
}

// PersistentFlags sets the persistent flags for the Trustification integration.
func (i *TrustificationIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&i.force, "force", i.force,
		"Overwrite the existing secret")

	p.StringVar(&i.bombasticAPIURL, "bombastic-api-url", i.bombasticAPIURL,
		"URL of the BOMbastic api host (e.g. https://sbom.trustification.dev)")
	p.StringVar(&i.oidcIssuerURL, "oidc-issuer-url", i.oidcIssuerURL,
		"URL of the OIDC token issuer (e.g. https://sso.trustification.dev/realms/chicken)")
	p.StringVar(&i.oidcClientId, "oidc-client-id", i.oidcClientId,
		"OIDC client ID")
	p.StringVar(&i.oidcClientSecret, "oidc-client-secret", i.oidcClientSecret,
		"OIDC client secret")
	p.StringVar(&i.supportedCycloneDXVersion, "supported-cyclonedx-version", i.supportedCycloneDXVersion,
		"If the SBOM uses a higher CycloneDX version, Syft convert to the supported version before uploading.")
}

// log logger with contextual information.
func (i *TrustificationIntegration) log() *slog.Logger {
	return i.logger.With(
		"force", i.force,
		"bombasticAPIURL", i.bombasticAPIURL,
		"oidcIssuerURL", i.oidcIssuerURL,
		"oidcClientId", i.oidcClientId,
		"oidcClientSecret-len", len(i.oidcClientSecret),
		"supportedCyclonedxVersion", i.supportedCycloneDXVersion,
	)
}

// Validate checks if the required configuration is set.
func (i *TrustificationIntegration) Validate() error {
	if i.bombasticAPIURL == "" {
		return fmt.Errorf("bombastic-api-url is required")
	}
	if !strings.Contains(i.bombasticAPIURL, "://") {
		return fmt.Errorf("invalid bombastic-api-url, the protocol should be specified")
	}
	if i.oidcIssuerURL == "" {
		return fmt.Errorf("oidc-issuer-url is required")
	}
	if !strings.Contains(i.oidcIssuerURL, "://") {
		return fmt.Errorf("invalid oidc-issuer-url, the protocol should be specified")
	}
	if i.oidcClientId == "" {
		return fmt.Errorf("oidc-client-id is required")
	}
	if i.oidcClientSecret == "" {
		return fmt.Errorf("oidc-client-secret is required")
	}
	return nil
}

// EnsureNamespace ensures the namespace needed for the Trustification integration secret
// is created on the cluster.
func (i *TrustificationIntegration) EnsureNamespace(
	ctx context.Context,
	cfg *config.Config,
) error {
	return k8s.EnsureOpenShiftProject(
		ctx,
		i.log(),
		i.kube,
		cfg.Installer.Namespace,
	)
}

// secretName returns the secret name for the integration. The name is "lazy"
// generated to make sure configuration is already loaded.
func (i *TrustificationIntegration) secretName(
	cfg *config.Config,
) types.NamespacedName {
	return types.NamespacedName{
		Namespace: cfg.Installer.Namespace,
		Name:      "tssc-trustification-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (i *TrustificationIntegration) prepareSecret(
	ctx context.Context,
	cfg *config.Config,
) error {
	i.log().Debug("Checking if integration secret exists")
	exists, err := k8s.SecretExists(ctx, i.kube, i.secretName(cfg))
	if err != nil {
		return err
	}
	if !exists {
		i.log().Debug("Integration secret does not exist")
		return nil
	}
	if !i.force {
		i.log().Debug("Integration secret already exists")
		return fmt.Errorf("%w: %s",
			ErrSecretAlreadyExists, i.secretName(cfg).String())
	}
	i.log().Debug("Integration secret already exists, recreating it")
	return k8s.DeleteSecret(ctx, i.kube, i.secretName(cfg))
}

// store creates the secret with the integration data.
func (i *TrustificationIntegration) store(
	ctx context.Context,
	cfg *config.Config,
) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: i.secretName(cfg).Namespace,
			Name:      i.secretName(cfg).Name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"bombastic_api_url":           []byte(i.bombasticAPIURL),
			"oidc_client_id":              []byte(i.oidcClientId),
			"oidc_client_secret":          []byte(i.oidcClientSecret),
			"oidc_issuer_url":             []byte(i.oidcIssuerURL),
			"supported_cyclonedx_version": []byte(i.supportedCycloneDXVersion),
		},
	}
	logger := i.log().With(
		"secret-namespace", secret.GetNamespace(),
		"secret-name", secret.GetName(),
	)

	logger.Debug("Creating integration secret")
	coreClient, err := i.kube.CoreV1ClientSet(i.secretName(cfg).Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(i.secretName(cfg).Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		logger.Info("Integration secret created successfully!")
	}
	return err
}

// Create creates the Trustification integration Kubernetes secret.
func (i *TrustificationIntegration) Create(
	ctx context.Context,
	cfg *config.Config,
) error {
	logger := i.log()
	logger.Info("Inspecting the cluster for an existing Trustification integration secret")
	if err := i.prepareSecret(ctx, cfg); err != nil {
		return err
	}
	return i.store(ctx, cfg)
}

func NewTrustificationIntegration(
	logger *slog.Logger,
	kube *k8s.Kube,
) *TrustificationIntegration {
	return &TrustificationIntegration{
		logger: logger,
		kube:   kube,

		force:                     false,
		bombasticAPIURL:           "",
		oidcClientId:              "",
		oidcClientSecret:          "",
		oidcIssuerURL:             "",
		supportedCycloneDXVersion: "",
	}
}
