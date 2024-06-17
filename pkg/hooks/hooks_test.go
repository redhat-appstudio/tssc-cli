package hooks

import (
	"testing"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"helm.sh/helm/v3/pkg/chartutil"

	o "github.com/onsi/gomega"
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
