package hooks

import (
	"testing"

	o "github.com/onsi/gomega"
	"github.com/otaviof/rhtap-installer-cli/pkg/config"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestNewHooks(t *testing.T) {
	g := o.NewWithT(t)

	h := NewHooks(config.Dependency{
		Chart:     "../../test/charts/testing",
		Namespace: "rhtap",
	})

	vals := chartutil.Values{"key": "value"}

	err := h.PreDeploy(vals)
	g.Expect(err).To(o.Succeed())
	err = h.PostDeploy(vals)
	g.Expect(err).To(o.Succeed())
}
