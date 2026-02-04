package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/redhat-appstudio/helmet/internal/annotations"
	"github.com/redhat-appstudio/helmet/internal/constants"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapManager the actor responsible for managing installer configuration in
// the cluster.
type ConfigMapManager struct {
	kube *k8s.Kube // kubernetes client
	name string    // configmap name
}

// Selector label selector for installer configuration.
const Selector = annotations.Config + "=true"

// Name returns the ConfigMap name.
func (m *ConfigMapManager) Name() string {
	return m.name
}

var (
	// ErrConfigMapNotFound when the configmap isn't created in the cluster.
	ErrConfigMapNotFound = errors.New("cluster configmap not found")
	// ErrMultipleConfigMapFound when the label selector find multiple resources.
	ErrMultipleConfigMapFound = errors.New("multiple cluster configmaps found")
	// ErrIncompleteConfigMap when the ConfigMap exists, but doesn't contain the
	// expected payload.
	ErrIncompleteConfigMap = errors.New("invalid configmap found in the cluster")
)

// GetConfigMap retrieves the ConfigMap from the cluster, checking if a single
// resource is present.
func (m *ConfigMapManager) GetConfigMap(
	ctx context.Context,
) (*corev1.ConfigMap, error) {
	coreClient, err := m.kube.CoreV1ClientSet("")
	if err != nil {
		return nil, err
	}

	// Listing all ConfigMaps matching the label selector.
	configMapList, err := coreClient.ConfigMaps("").List(ctx, metav1.ListOptions{
		LabelSelector: Selector,
	})
	if err != nil {
		return nil, err
	}

	// When no ConfigMaps matching criteria is found in the cluster.
	if len(configMapList.Items) == 0 {
		return nil, fmt.Errorf(
			"%w: using label selector %q",
			ErrConfigMapNotFound,
			Selector,
		)
	}
	// Also, important to error out when multiple ConfigMaps are present in the
	// cluster. Collecting and printing out the resources found by the label
	// selector.
	if len(configMapList.Items) > 1 {
		configMaps := []string{}
		for _, cm := range configMapList.Items {
			configMaps = append(
				configMaps,
				fmt.Sprintf("%s/%s", cm.GetNamespace(), cm.GetName()),
			)
		}
		return nil, fmt.Errorf(
			"%w: multiple configmaps found on namespace/name pairs: %v",
			ErrMultipleConfigMapFound,
			configMaps,
		)
	}
	return &configMapList.Items[0], nil
}

// GetConfig retrieves configuration from a cluster's ConfigMap.
func (m *ConfigMapManager) GetConfig(ctx context.Context) (*Config, error) {
	configMap, err := m.GetConfigMap(ctx)
	if err != nil {
		return nil, err
	}
	payload, ok := configMap.Data[constants.ConfigFilename]
	if !ok || len(payload) == 0 {
		return nil, fmt.Errorf(
			"%w: key %q not found in ConfigMap %s/%s",
			ErrIncompleteConfigMap,
			constants.ConfigFilename,
			configMap.GetNamespace(),
			configMap.GetName(),
		)
	}

	return NewConfigFromBytes([]byte(payload), configMap.GetNamespace())
}

// configMapForConfig generate a ConfigMap resource based on informed Config.
func (m *ConfigMapManager) configMapForConfig(
	cfg *Config,
) (*corev1.ConfigMap, error) {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.name,
			Namespace: cfg.Namespace(),
			Labels: map[string]string{
				annotations.Config: "true",
			},
		},
		Data: map[string]string{
			constants.ConfigFilename: cfg.String(),
		},
	}, nil
}

// Create Bootstrap a ConfigMap with the provided configuration.
func (m *ConfigMapManager) Create(ctx context.Context, cfg *Config) error {
	cm, err := m.configMapForConfig(cfg)
	if err != nil {
		return err
	}
	coreClient, err := m.kube.CoreV1ClientSet(cfg.Namespace())
	if err != nil {
		return nil
	}
	_, err = coreClient.
		ConfigMaps(cfg.Namespace()).
		Create(ctx, cm, metav1.CreateOptions{})
	return err
}

// Update updates a ConfigMap with informed configuration.
func (m *ConfigMapManager) Update(ctx context.Context, cfg *Config) error {
	cm, err := m.configMapForConfig(cfg)
	if err != nil {
		return err
	}
	coreClient, err := m.kube.CoreV1ClientSet(cfg.Namespace())
	if err != nil {
		return nil
	}
	_, err = coreClient.
		ConfigMaps(cfg.Namespace()).
		Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

// Delete find and delete the ConfigMap from the cluster.
func (m *ConfigMapManager) Delete(ctx context.Context) error {
	cm, err := m.GetConfigMap(ctx)
	if err != nil {
		return err
	}

	coreClient, err := m.kube.CoreV1ClientSet(cm.GetNamespace())
	if err != nil {
		return nil
	}

	return coreClient.ConfigMaps(cm.GetNamespace()).
		Delete(ctx, cm.GetName(), metav1.DeleteOptions{})
}

// NewConfigMapManager instantiates the ConfigMapManager.
// The appName parameter is used to generate the ConfigMap name as "{appName}-config".
func NewConfigMapManager(kube *k8s.Kube, appName string) *ConfigMapManager {
	return &ConfigMapManager{
		kube: kube,
		name: fmt.Sprintf("%s-config", appName),
	}
}
