package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
)

// Interface abstracts Kubernetes client operations for both real and fake
// clusters, enabling testability across all consumers.
type Interface interface {
	// BatchV1ClientSet returns a Batch v1 client for the given namespace.
	BatchV1ClientSet(string) (batchv1client.BatchV1Interface, error)

	// ClientSet returns a full Kubernetes clientset for the given namespace.
	ClientSet(string) (kubernetes.Interface, error)

	// Connected verifies the cluster is reachable.
	Connected() error

	// CoreV1ClientSet returns a CoreV1 client for the given namespace.
	CoreV1ClientSet(string) (corev1client.CoreV1Interface, error)

	// DiscoveryClient returns a discovery client for the given namespace.
	DiscoveryClient(string) (discovery.DiscoveryInterface, error)

	// DynamicClient returns a dynamic client for the given namespace.
	DynamicClient(string) (dynamic.Interface, error)

	// GetDynamicClientForObjectRef returns a dynamic resource client for the
	// object reference.
	GetDynamicClientForObjectRef(
		*corev1.ObjectReference,
	) (dynamic.ResourceInterface, error)

	// RBACV1ClientSet returns an RBAC v1 client for the given namespace.
	RBACV1ClientSet(string) (rbacv1client.RbacV1Interface, error)

	// RESTClientGetter returns a REST client getter for the given namespace.
	RESTClientGetter(string) genericclioptions.RESTClientGetter
}
