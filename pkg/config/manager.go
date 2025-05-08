package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapManager the actor responsible for managing installer configuration in
// the cluster.
type ConfigMapManager struct {
	kube *k8s.Kube // kubernetes client
}

const (
	// Filename the default
	Filename = "config.yaml"
	// Label label selector to find the cluster's installer configuration.
	Label = "tssc.redhat-appstudio.github.com/config"
)

var (
	// ErrConfigMapNotFound when the configmap isn't created in the cluster.
	ErrConfigMapNotFound = errors.New("cluster configmap not found")
	// ErrMultipleConfigMapFound when the label selector find multiple resources.
	ErrMultipleConfigMapFound = errors.New("multiple cluster configmaps found")
	// ErrIncompleteConfigMap when the ConfigMap exists, but doesn't contain the
	// expected payload.
	ErrIncompleteConfigMap = errors.New("invalid configmap found in the cluster")
)

// selectorLabel returns the label selector.
func (m *ConfigMapManager) selectorLabel() string {
	return fmt.Sprintf("%s=true", Label)
}

// GetConfigMap retrieves the ConfigMap from the cluster, checking if a single
// resource is present.
func (m *ConfigMapManager) GetConfigMap(
	ctx context.Context,
) (*corev1.ConfigMap, error) {
	coreClient, err := m.kube.CoreV1ClientSet("")
	if err != nil {
		return nil, nil
	}

	// Listing all ConfigMaps matching the label selector.
	configMapList, err := coreClient.ConfigMaps("").List(ctx, metav1.ListOptions{
		LabelSelector: m.selectorLabel(),
	})
	if err != nil {
		return nil, err
	}

	// When no ConfigMaps matching criteria is found in the cluster.
	if len(configMapList.Items) == 0 {
		return nil, fmt.Errorf(
			"%w: using label selector %q",
			ErrConfigMapNotFound,
			m.selectorLabel(),
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
	payload, ok := configMap.Data[Filename]
	if !ok || len(payload) == 0 {
		return nil, fmt.Errorf(
			"%w: key %q not found in ConfigMap %s/%s",
			ErrIncompleteConfigMap,
			Filename,
			configMap.GetNamespace(),
			configMap.GetName(),
		)
	}

	return NewConfigFromBytes([]byte(payload))
}

// configMapForConfig generate a ConfigMap resource based on informed Config.
func (m *ConfigMapManager) configMapForConfig(
	cfg *Config,
) (*corev1.ConfigMap, error) {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tssc-config",
			Namespace: cfg.Installer.Namespace,
			Labels: map[string]string{
				Label: "true",
			},
		},
		Data: map[string]string{
			Filename: cfg.String(),
		},
	}, nil
}

// Create Bootstrap a ConfigMap with the provided configuration.
func (m *ConfigMapManager) Create(ctx context.Context, cfg *Config) error {
	cm, err := m.configMapForConfig(cfg)
	if err != nil {
		return err
	}
	coreClient, err := m.kube.CoreV1ClientSet(cfg.Installer.Namespace)
	if err != nil {
		return nil
	}
	_, err = coreClient.
		ConfigMaps(cfg.Installer.Namespace).
		Create(ctx, cm, metav1.CreateOptions{})
	return err
}

// Update updates a ConfigMap with informed configuration.
func (m *ConfigMapManager) Update(ctx context.Context, cfg *Config) error {
	cm, err := m.configMapForConfig(cfg)
	if err != nil {
		return err
	}
	coreClient, err := m.kube.CoreV1ClientSet(cfg.Installer.Namespace)
	if err != nil {
		return nil
	}
	_, err = coreClient.
		ConfigMaps(cfg.Installer.Namespace).
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
func NewConfigMapManager(kube *k8s.Kube) *ConfigMapManager {
	return &ConfigMapManager{
		kube: kube,
	}
}
