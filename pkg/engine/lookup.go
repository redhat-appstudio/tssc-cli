package engine

import (
	"context"

	"github.com/otaviof/rhtap-installer-cli/pkg/k8s"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

// LookupFuncs represents the template functions that will need to lookup
// Kubernetes resources.
type LookupFuncs struct {
	kube *k8s.Kube
}

type LookupFn func(string, string, string, string) (map[string]interface{}, error)

func (l *LookupFuncs) lookup(
	apiVersion, kind, namespace, name string,
) (map[string]interface{}, error) {
	empty := map[string]interface{}{}

	dc, namespaced, err := l.kube.GetDynamicClientOnKind(
		apiVersion, kind, namespace)
	if err != nil {
		return empty, err
	}

	var client dynamic.ResourceInterface
	if namespaced {
		client = dc.Namespace(namespace)
	} else {
		client = dc
	}

	ctx := context.Background()
	if name != "" {
		obj, err := client.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return empty, nil
			}
			return empty, err
		}
		return obj.UnstructuredContent(), nil
	}

	objList, err := client.List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return empty, nil
		}
		return empty, err
	}
	return objList.UnstructuredContent(), nil
}

func (l *LookupFuncs) Lookup() LookupFn {
	return l.lookup
}

// NewLookupFuncs creates a new LookupFuncs instance.
func NewLookupFuncs(kube *k8s.Kube) *LookupFuncs {
	return &LookupFuncs{kube: kube}
}
