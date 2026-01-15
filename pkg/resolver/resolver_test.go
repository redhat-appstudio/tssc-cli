package resolver

import (
	"os"
	"testing"

	"github.com/redhat-appstudio/helmet/pkg/api"
	"github.com/redhat-appstudio/helmet/pkg/chartfs"
	"github.com/redhat-appstudio/helmet/pkg/config"

	o "github.com/onsi/gomega"
)

func TestNewResolver(t *testing.T) {
	g := o.NewWithT(t)

	cfs := chartfs.New(os.DirFS("../../installer"))

	installerNamespace := "test-namespace"
	cfg, err := config.NewConfigFromFile(cfs, "config.yaml", installerNamespace)
	g.Expect(err).To(o.Succeed())

	charts, err := cfs.GetAllCharts()
	g.Expect(err).To(o.Succeed())

	appCtx := api.NewAppContext("tssc")
	c, err := NewCollection(appCtx, charts)
	g.Expect(err).To(o.Succeed())

	t.Run("Resolve", func(t *testing.T) {
		topology := NewTopology()
		r := NewResolver(cfg, c, topology)

		err := r.Resolve()
		g.Expect(err).To(o.Succeed())

		// Extracting the Helm chart names and namespaces from the topology.
		dependencyNamespaceMap := map[string]string{}
		dependencySlice := []string{}
		for _, d := range topology.Dependencies() {
			dependencyNamespaceMap[d.Name()] = d.Namespace()
			dependencySlice = append(dependencySlice, d.Name())
		}
		// Showing the resolved dependencies.
		t.Logf("Resolved dependencies (%d)", len(dependencySlice))
		i := 1
		for name, ns := range dependencyNamespaceMap {
			t.Logf("(%2d) %s -> %s", i, name, ns)
			i++
		}
		g.Expect(len(dependencySlice)).To(o.Equal(13))

		// Validating the order of the resolved dependencies, as well as the
		// namespace of each dependency.
		g.Expect(dependencyNamespaceMap).To(o.Equal(map[string]string{
			"tssc-openshift":      installerNamespace,
			"tssc-subscriptions":  installerNamespace,
			"tssc-infrastructure": installerNamespace,
			"tssc-iam":            installerNamespace,
			"tssc-tpa":            "tssc-tpa",
			"tssc-tas":            "tssc-tas",
			"tssc-pipelines":      installerNamespace,
			"tssc-gitops":         "tssc-gitops",
			"tssc-app-namespaces": installerNamespace,
			"tssc-dh":             "tssc-dh",
			"tssc-acs":            "tssc-acs",
			"tssc-acs-test":       "tssc-acs",
			"tssc-integrations":   installerNamespace,
		}))
		g.Expect(dependencySlice).To(o.Equal([]string{
			"tssc-openshift",
			"tssc-subscriptions",
			"tssc-acs",
			"tssc-gitops",
			"tssc-infrastructure",
			"tssc-iam",
			"tssc-tas",
			"tssc-pipelines",
			"tssc-tpa",
			"tssc-app-namespaces",
			"tssc-dh",
			"tssc-integrations",
			"tssc-acs-test",
		}))
	})
}
