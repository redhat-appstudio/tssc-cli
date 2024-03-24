package k8s

import (
	"errors"
	"fmt"

	"github.com/otaviof/rhtap-installer-cli/pkg/flags"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

// Kube represents the Kubernetes client helper.
type Kube struct {
	flags *flags.Flags // global flags
}

// ErrClientNotConnected kubernetes clients is not able to access the API.
var ErrClientNotConnected = errors.New("kubernetes client not connected")

func (k *Kube) RESTClientGetter(namespace string) genericclioptions.RESTClientGetter {
	g := genericclioptions.NewConfigFlags(false)
	g.KubeConfig = &k.flags.KubeConfigPath
	g.Namespace = &namespace
	return g
}

// DiscoveryClient instantiates a discovery client for the given namespace.
func (k *Kube) DiscoveryClient(namespace string) (*discovery.DiscoveryClient, error) {
	restConfig, err := k.RESTClientGetter(namespace).ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(restConfig)
}

func (k *Kube) DynamicClient(namespace string) (*dynamic.DynamicClient, error) {
	restConfig, err := k.RESTClientGetter(namespace).ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(restConfig)
}

func (k *Kube) GetDynamicClientOnKind(
	apiVersion, kind, namespace string,
) (dynamic.NamespaceableResourceInterface, bool, error) {
	gvk := schema.FromAPIVersionAndKind(apiVersion, kind)
	dc, err := k.DiscoveryClient(namespace)
	if err != nil {
		return nil, false, err
	}

	resList, err := dc.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return nil, false, err
	}
	var apiResource metav1.APIResource
	for _, r := range resList.APIResources {
		if r.Kind == kind {
			apiResource = r
			apiResource.Group = gvk.Group
			apiResource.Version = gvk.Version
		}
	}

	gvr := gvk.GroupVersion().WithResource(apiResource.Name)
	dynamicClient, err := k.DynamicClient(namespace)
	if err != nil {
		return nil, false, err
	}

	return dynamicClient.Resource(gvr), apiResource.Namespaced, nil
}

// Connected reads the cluster's version, to assert if the client is working. For
// this purpose it assumes namespace "default".
func (k *Kube) Connected() error {
	dc, err := k.DiscoveryClient("default")
	if err != nil {
		return err
	}
	if _, err = dc.ServerVersion(); err != nil {
		return fmt.Errorf("%w: %s", ErrClientNotConnected, err.Error())
	}
	return nil
}

func NewKube(flags *flags.Flags) *Kube {
	return &Kube{flags: flags}
}
