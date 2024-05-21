package engine

import (
	"context"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"github.com/redhat-appstudio/rhtap-cli/pkg/k8s"

	"helm.sh/helm/v3/pkg/chartutil"
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
func (v *Variables) SetOpenShift(ctx context.Context, kube *k8s.Kube) error {
	ingressDomain, err := k8s.GetOpenShiftIngressDomain(ctx, kube)
	if err != nil {
		return err
	}
	v.OpenShift = chartutil.Values{
		"Ingress": chartutil.Values{
			"Domain": ingressDomain,
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
