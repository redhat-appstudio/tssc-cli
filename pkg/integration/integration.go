package integration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
	"github.com/redhat-appstudio/tssc-cli/pkg/k8s"

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
func (i *Integration) log(nsm types.NamespacedName) *slog.Logger {
	return i.data.LoggerWith(i.logger.With(
		"secret-namespace", nsm.Namespace,
		"secret-name", nsm.Name,
		"secret-type", i.data.Type(),
	))
}

// Exists checks whether the integration secret exists in the cluster.
func (i *Integration) Exists(
	ctx context.Context,
	nsm types.NamespacedName,
) (bool, error) {
	return k8s.SecretExists(ctx, i.kube, nsm)
}

// prepare prepares the cluster to receive the integration secret, when the force
// flag is enabled a existing secret is deleted.
func (i *Integration) prepare(ctx context.Context, nsm types.NamespacedName) error {
	i.log(nsm).Debug("Checking whether the integration secret exists")
	exists, err := i.Exists(ctx, nsm)
	if err != nil {
		return err
	}
	if !exists {
		i.log(nsm).Debug("Integration secret does not exist")
		return nil
	}
	if !i.force {
		i.log(nsm).Debug("Integration secret already exists")
		return fmt.Errorf("%w: %s", ErrSecretAlreadyExists, nsm.String())
	}
	i.log(nsm).Debug("Integration secret already exists, recreating it")
	return i.Delete(ctx, nsm)
}

// Create creates the integration secret in the cluster. It uses the integration
// data provider to obtain the secret payload.
func (i *Integration) Create(
	ctx context.Context,
	cfg *config.Config,
	nsm types.NamespacedName,
) error {
	err := i.prepare(ctx, nsm)
	if err != nil {
		return err
	}

	// The integration provider prepares and returns the payload to create the
	// Kubernetes secret.
	i.log(nsm).Debug("Preparing the integration secret payload")
	payload, err := i.data.Data(ctx, cfg)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsm.Namespace,
			Name:      nsm.Name,
		},
		Type: i.data.Type(),
		Data: payload,
	}

	i.log(nsm).Debug("Creating the integration secret")
	coreClient, err := i.kube.CoreV1ClientSet(nsm.Namespace)
	if err != nil {
		return err
	}
	_, err = coreClient.Secrets(nsm.Namespace).
		Create(ctx, secret, metav1.CreateOptions{})
	if err == nil {
		i.log(nsm).Info("Integration secret is created successfully!")
	}
	return err
}

// Delete deletes the Kubernetes secret.
func (i *Integration) Delete(
	ctx context.Context,
	nsm types.NamespacedName,
) error {
	return k8s.DeleteSecret(ctx, i.kube, nsm)
}

// NewSecret instantiates a new secret manager, it uses the integration data
// provider to generate the Kubernetes Secret payload.
func NewSecret(logger *slog.Logger, kube *k8s.Kube, data Interface) *Integration {
	return &Integration{
		logger: logger,
		kube:   kube,
		data:   data,
	}
}
