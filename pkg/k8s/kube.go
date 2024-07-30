package k8s

import (
	"errors"
	"fmt"

	"github.com/redhat-appstudio/rhtap-cli/pkg/flags"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Kube represents the Kubernetes client helper.
type Kube struct {
	flags *flags.Flags // global flags
}

var _ Interface = &Kube{}

// ErrClientNotConnected kubernetes clients is not able to access the API.
var ErrClientNotConnected = errors.New("kubernetes client not connected")

// RESTClientGetter returns a REST client getter for the given namespace.
func (k *Kube) RESTClientGetter(namespace string) genericclioptions.RESTClientGetter {
	g := genericclioptions.NewConfigFlags(false)
	g.KubeConfig = &k.flags.KubeConfigPath
	g.Namespace = &namespace
	return g
}

// ClientSet returns a "corev1" Kubernetes Clientset.
func (k *Kube) ClientSet(namespace string) (kubernetes.Interface, error) {
	restConfig, err := k.RESTClientGetter(namespace).ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(restConfig)
}

// CoreV1ClientSet returns a "corev1" Kubernetes Clientset.
func (k *Kube) CoreV1ClientSet(
	namespace string,
) (corev1client.CoreV1Interface, error) {
	restConfig, err := k.RESTClientGetter(namespace).ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return corev1client.NewForConfig(restConfig)
}

// DiscoveryClient instantiates a discovery client for the given namespace.
func (k *Kube) DiscoveryClient(namespace string) (discovery.DiscoveryInterface, error) {
	restConfig, err := k.RESTClientGetter(namespace).ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(restConfig)
}

// DynamicClient instantiates a dynamic client for the given namespace.
func (k *Kube) DynamicClient(namespace string) (dynamic.Interface, error) {
	restConfig, err := k.RESTClientGetter(namespace).ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(restConfig)
}

// GetDynamicClientForObjectRef returns a dynamic client for the object reference.
func (k *Kube) GetDynamicClientForObjectRef(
	objectRef *corev1.ObjectReference,
) (dynamic.ResourceInterface, error) {
	dc, err := k.DiscoveryClient(objectRef.Namespace)
	if err != nil {
		return nil, err
	}
	gvk := objectRef.GroupVersionKind()
	resList, err := dc.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return nil, err
	}
	var apiResource metav1.APIResource
	for _, r := range resList.APIResources {
		if r.Kind == objectRef.Kind {
			apiResource = r
			apiResource.Group = gvk.Group
			apiResource.Version = gvk.Version
		}
	}

	gvr := gvk.GroupVersion().WithResource(apiResource.Name)
	dynamicClient, err := k.DynamicClient(objectRef.Namespace)
	if err != nil {
		return nil, err
	}
	if apiResource.Namespaced {
		return dynamicClient.Resource(gvr).Namespace(objectRef.Namespace), nil
	}
	return dynamicClient.Resource(gvr), nil
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

// NewKube instantiates the Kubernetes client helper.
func NewKube(flags *flags.Flags) *Kube {
	return &Kube{flags: flags}
}
