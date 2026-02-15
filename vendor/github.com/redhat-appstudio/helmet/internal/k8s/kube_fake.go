package k8s

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/testing"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
)

type FakeKube struct {
	objects []runtime.Object
}

var _ Interface = &FakeKube{}

func (f *FakeKube) BatchV1ClientSet(
	namespace string,
) (batchv1client.BatchV1Interface, error) {
	cs, err := f.ClientSet(namespace)
	if err != nil {
		return nil, err
	}
	return cs.BatchV1(), nil
}

func (f *FakeKube) ClientSet(string) (kubernetes.Interface, error) {
	cs := fake.NewSimpleClientset(f.objects...)

	// Add reactor to automatically set namespace status to Active when created
	cs.PrependReactor(
		"create",
		"namespaces",
		func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			createAction := action.(testing.CreateAction)
			obj := createAction.GetObject()
			if ns, ok := obj.(*corev1.Namespace); ok {
				ns.Status.Phase = corev1.NamespaceActive
			}
			return false, obj, nil
		})

	return cs, nil
}

func (f *FakeKube) Connected() error {
	return nil
}

func (f *FakeKube) CoreV1ClientSet(
	namespace string,
) (corev1client.CoreV1Interface, error) {
	cs, err := f.ClientSet(namespace)
	if err != nil {
		return nil, err
	}
	return cs.CoreV1(), nil
}

func (f *FakeKube) DiscoveryClient(
	namespace string,
) (discovery.DiscoveryInterface, error) {
	cs, err := f.ClientSet(namespace)
	if err != nil {
		return nil, err
	}
	return cs.Discovery(), nil
}

func (f *FakeKube) DynamicClient(namespace string) (dynamic.Interface, error) {
	restConfig, err := f.RESTClientGetter(namespace).ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(restConfig)
}

func (f *FakeKube) GetDynamicClientForObjectRef(
	objectRef *corev1.ObjectReference,
) (dynamic.ResourceInterface, error) {
	dc, err := f.DiscoveryClient(objectRef.Namespace)
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
	dynamicClient, err := f.DynamicClient(objectRef.Namespace)
	if err != nil {
		return nil, err
	}
	if apiResource.Namespaced {
		return dynamicClient.Resource(gvr).Namespace(objectRef.Namespace), nil
	}
	return dynamicClient.Resource(gvr), nil
}

func (f *FakeKube) RBACV1ClientSet(
	namespace string,
) (rbacv1client.RbacV1Interface, error) {
	cs, err := f.ClientSet(namespace)
	if err != nil {
		return nil, err
	}
	return cs.RbacV1(), nil
}

func (f *FakeKube) RESTClientGetter(_ string) genericclioptions.RESTClientGetter {
	return cmdtesting.NewTestFactory()
}

func NewFakeKube(objects ...runtime.Object) *FakeKube {
	// Set Status.Phase to Active for any Namespace objects that don't have it set
	for i, obj := range objects {
		if ns, ok := obj.(*corev1.Namespace); ok {
			if ns.Status.Phase == "" {
				ns.Status.Phase = corev1.NamespaceActive
				objects[i] = ns
			}
		}
	}
	return &FakeKube{
		objects: objects,
	}
}
