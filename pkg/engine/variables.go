package engine

import (
	"context"

	"github.com/otaviof/rhtap-installer-cli/pkg/config"
	"github.com/otaviof/rhtap-installer-cli/pkg/k8s"

	operatorv1client "github.com/openshift/client-go/operator/clientset/versioned/typed/operator/v1"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Variables represents the variables available for "values-template" file.
type Variables struct {
	Installer chartutil.Values // .Installer
	OpenShift chartutil.Values // .OpenShift
}

// SetInstaller sets the installer configuration.
func (v *Variables) SetInstaller(cfg *config.Spec) error {
	var err error
	v.Installer, err = UnstructuredType(cfg)
	return err
}

// SetOpenShift sets the OpenShift context variables.
func (v *Variables) SetOpenShift(kube *k8s.Kube) error {
	objectRef := &corev1.ObjectReference{
		APIVersion: "operator.openshift.io/v1",
		Namespace:  "openshift-ingress-operator",
		Name:       "default",
	}

	restConfig, err := kube.RESTClientGetter(objectRef.Namespace).ToRESTConfig()
	if err != nil {
		return err
	}
	operatorClient, err := operatorv1client.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	ingressController, err := operatorClient.
		IngressControllers(objectRef.Namespace).
		Get(context.Background(), objectRef.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	v.OpenShift = chartutil.Values{
		"Ingress": chartutil.Values{
			"Domain": ingressController.Status.Domain,
		},
	}
	return nil
}

// Unstructured returns the variables as "chartutils.Values".
func (v *Variables) Unstructured() (chartutil.Values, error) {
	return UnstructuredType(v)
}

// NewVariables instantiates Variables empty.
func NewVariables() *Variables {
	return &Variables{
		Installer: chartutil.Values{},
		OpenShift: chartutil.Values{},
	}
}
