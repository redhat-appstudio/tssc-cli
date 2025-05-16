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

// defaultPublicAzureHost is the default host for public Azure.
const defaultPublicAzureHost = "dev.azure.com"

// AzureIntegration represents the TSSC Azure integration.
type AzureIntegration struct {
	logger *slog.Logger // application logger
	kube   *k8s.Kube    // kubernetes client

	force bool // overwrite the existing secret

	host         string // Azure host
	token        string // API token credentials
	org          string // Azure organization name
	clientId     string // Azure client ID
	clientSecret string // Azure client secret
	tenantId     string // Azure tenant ID
}

// PersistentFlags sets the persistent flags for the Azure integration.
func (g *AzureIntegration) PersistentFlags(p *pflag.FlagSet) {
	p.BoolVar(&g.force, "force", g.force,
		"Overwrite the existing secret")

	p.StringVar(&g.host, "host", g.host,
		"Azure host, defaults to 'dev.azure.com'")
	p.StringVar(&g.token, "token", g.token,
		"Azure API token")
	p.StringVar(&g.org, "organization", g.org,
		"Azure organization name")
	p.StringVar(&g.clientId, "client-id", g.clientId,
		"Azure client ID")
	p.StringVar(&g.clientSecret, "client-secret", g.clientSecret,
		"Azure client secret")
	p.StringVar(&g.tenantId, "tenant-id", g.tenantId,
		"Azure tenant ID")
}

// log logger with contextual information.
func (g *AzureIntegration) log() *slog.Logger {
	return g.logger.With(
		"force", g.force,
		"host", g.host,
		"token-len", len(g.token),
		"organization", g.org,
		"clientId", g.clientId,
		"clientSecret-len", len(g.clientSecret),
		"tenantId-len", len(g.tenantId),
	)
}

// Validate checks if the required configuration is set.
func (g *AzureIntegration) Validate() error {
	if g.host == "" {
		g.host = defaultPublicAzureHost
	}
	// Personal access token is required in register existing component
	if g.token == "" {
		return fmt.Errorf("personal access token is required")
	}
	if g.clientId == "" && (g.clientSecret != "" || g.tenantId != "") {
		return fmt.Errorf("client-id is required when client-secret or tenant-id is specified")
	}
	if g.clientSecret == "" && g.tenantId != "" {
		return fmt.Errorf("client-secret is required when tenant-id is specified")
	}
	if g.clientSecret != "" && g.tenantId == "" {
		return fmt.Errorf("tenant-id is required when client-secret is specified")
	}
	if g.org == "" {
		return fmt.Errorf("organization is required")
	}

	return nil
}

// EnsureNamespace ensures the namespace needed for the Azure integration secret
// is created on the cluster.
func (g *AzureIntegration) EnsureNamespace(
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
func (g *AzureIntegration) secretName(cfg *config.Config) types.NamespacedName {
	return types.NamespacedName{
		Namespace: cfg.Installer.Namespace,
		Name:      "tssc-azure-integration",
	}
}

// prepareSecret checks if the secret already exists, and if so, it will delete
// the secret if the force flag is enabled.
func (g *AzureIntegration) prepareSecret(
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
func (g *AzureIntegration) store(
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
			"host":         []byte(g.host),
			"token":        []byte(g.token),
			"organization": []byte(g.org),
			"clientId":     []byte(g.clientId),
			"clientSecret": []byte(g.clientSecret),
			"tenantId":     []byte(g.tenantId),
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

// Create creates the Azure integration Kubernetes secret.
func (g *AzureIntegration) Create(
	ctx context.Context,
	cfg *config.Config,
) error {
	logger := g.log()
	logger.Info("Inspecting the cluster for an existing Azure integration secret")
	if err := g.prepareSecret(ctx, cfg); err != nil {
		return err
	}
	return g.store(ctx, cfg)
}

func NewAzureIntegration(logger *slog.Logger, kube *k8s.Kube) *AzureIntegration {
	return &AzureIntegration{
		logger: logger,
		kube:   kube,

		force:        false,
		host:         "",
		token:        "",
		org:          "",
		clientId:     "",
		clientSecret: "",
		tenantId:     "",
	}
}
