package k8s

import (
	"errors"
	"fmt"

	"github.com/otaviof/rhtap-installer-cli/pkg/flags"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
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
