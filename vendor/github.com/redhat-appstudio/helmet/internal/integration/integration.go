package integration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Integration represents a generic Kubernetes Secret manager for integrations, it
// holds the common actions integrations will perform against secrets.
type Integration struct {
	logger *slog.Logger // application logger
	kube   *k8s.Kube    // kubernetes client
	name   string       // kubernetes secret name
	data   Interface    // provides secret data

	force bool // overwrite the existing secret
}

// ErrSecretAlreadyExists integration secret already exists.
var ErrSecretAlreadyExists = fmt.Errorf("secret already exists")

// PersistentFlags decorates the cobra instance with persistent flags.
func (i *Integration) PersistentFlags(cmd *cobra.Command) {
	p := cmd.PersistentFlags()

	p.BoolVar(&i.force, "force", i.force, "Overwrite the existing secret")

	// Decorating the command with integration data flags.
	i.data.PersistentFlags(cmd)
}

// SetArgument exposes the data provider method.
func (i *Integration) SetArgument(k, v string) error {
	return i.data.SetArgument(k, v)
}

// Validate validates the secret payload, using the data interface.
func (i *Integration) Validate() error {
	return i.data.Validate()
}

// log returns a logger decorated with secret and data attributes.
func (i *Integration) log() *slog.Logger {
	return i.data.LoggerWith(i.logger.With(
		"secret-name", i.name,
		"secret-type", i.data.Type(),
	))
}

// secretName generates the namespaced name for the integration secret.
func (i *Integration) secretName(cfg *config.Config) types.NamespacedName {
	return types.NamespacedName{
		Namespace: cfg.Namespace(),
		Name:      i.name,
	}
}

// Exists checks whether the integration secret exists in the cluster.
func (i *Integration) Exists(
	ctx context.Context,
	cfg *config.Config,
) (bool, error) {
	return k8s.SecretExists(ctx, i.kube, i.secretName(cfg))
}

// prepare prepares the cluster to receive the integration secret, when the force
// flag is enabled a existing secret is deleted.
func (i *Integration) prepare(ctx context.Context, cfg *config.Config) error {
	i.log().Debug("Checking whether the integration secret exists")
	exists, err := i.Exists(ctx, cfg)
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
	return i.Delete(ctx, cfg)
}

// Create creates the integration secret in the cluster. It uses the integration
// data provider to obtain the secret payload.
func (i *Integration) Create(ctx context.Context, cfg *config.Config) error {
	err := i.prepare(ctx, cfg)
	if err != nil {
		return err
	}

	// The integration provider prepares and returns the payload to create the
	// Kubernetes secret.
	i.log().Debug("Preparing the integration secret payload")
	payload, err := i.data.Data(ctx, cfg)
	if err != nil {
		return err
	}
	namespace := i.secretName(cfg).Namespace
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      i.name,
		},
		Type: i.data.Type(),
		Data: payload,
	}

	i.log().Debug("Creating the integration secret")
	coreClient, err := i.kube.CoreV1ClientSet(namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		i.log().Info("Integration secret is created successfully!")
	}
	return err
}

// Delete deletes the Kubernetes secret.
func (i *Integration) Delete(ctx context.Context, cfg *config.Config) error {
	return k8s.DeleteSecret(ctx, i.kube, i.secretName(cfg))
}

// NewSecret instantiates a new secret manager, it uses the integration data
// provider to generate the Kubernetes Secret payload.
func NewSecret(
	logger *slog.Logger,
	kube *k8s.Kube,
	name string,
	data Interface,
) *Integration {
	return &Integration{logger: logger, kube: kube, name: name, data: data}
}
