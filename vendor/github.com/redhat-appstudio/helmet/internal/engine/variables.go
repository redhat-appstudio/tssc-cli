package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/redhat-appstudio/helmet/internal/config"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"helm.sh/helm/v3/pkg/chartutil"
)

// Variables represents the variables available for "values-template" file.
type Variables struct {
	Installer chartutil.Values // .Installer
	OpenShift chartutil.Values // .OpenShift
}

// SetInstaller sets the installer configuration.
func (v *Variables) SetInstaller(cfg *config.Config) error {
	v.Installer["Namespace"] = cfg.Namespace()
	settings, err := UnstructuredType(cfg.Installer.Settings)
	if err != nil {
		return err
	}
	v.Installer["Settings"] = settings.AsMap()
	products := map[string]interface{}{}
	for _, product := range cfg.Installer.Products {
		products[product.KeyName()] = product
	}
	v.Installer["Products"], err = UnstructuredType(products)
	return err
}

func getMinorVersion(version string) (string, error) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("version does not include a minor part")
	}
	minorVersion := strings.Join(parts[:2], ".")

	return minorVersion, nil
}

// SetOpenShift sets the OpenShift context variables.
func (v *Variables) SetOpenShift(ctx context.Context, kube *k8s.Kube) error {
	ingressDomain, err := k8s.GetOpenShiftIngressDomain(ctx, kube)
	if err != nil {
		return err
	}
	ingressRouterCA, err := k8s.GetOpenShiftIngressRouteCA(ctx, kube)
	if err != nil {
		return err
	}
	clusterVersion, err := k8s.GetOpenShiftVersion(ctx, kube)
	if err != nil {
		return err
	}
	minorVersion, err := getMinorVersion(clusterVersion)
	if err != nil {
		return err
	}
	v.OpenShift = chartutil.Values{
		"Ingress": chartutil.Values{
			"Domain":   ingressDomain,
			"RouterCA": ingressRouterCA,
		},
		"Version":      clusterVersion,
		"MinorVersion": minorVersion,
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
