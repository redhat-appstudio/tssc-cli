package resolver

import (
	"testing"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/config"

	o "github.com/onsi/gomega"
)

func TestNewResolver(t *testing.T) {
	g := o.NewWithT(t)

	cfs, err := chartfs.NewChartFS("../../installer")
	g.Expect(err).To(o.Succeed())

	cfg, err := config.NewConfigFromFile(cfs, "config.yaml")
	g.Expect(err).To(o.Succeed())

	charts, err := cfs.GetAllCharts()
	g.Expect(err).To(o.Succeed())

	c, err := NewCollection(charts)
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
		g.Expect(len(dependencySlice)).To(o.Equal(14))

		// Validating the order of the resolved dependencies, as well as the
		// namespace of each dependency.
		g.Expect(dependencyNamespaceMap).To(o.Equal(map[string]string{
			"tssc-openshift":        "tssc",
			"tssc-subscriptions":    "tssc",
			"tssc-infrastructure":   "tssc",
			"tssc-backing-services": "tssc-keycloak",
			"tssc-tpa-realm":        "tssc-tpa",
			"tssc-tpa":              "tssc-tpa",
			"tssc-tas":              "tssc-tas",
			"tssc-pipelines":        "tssc",
			"tssc-gitops":           "tssc-gitops",
			"tssc-app-namespaces":   "tssc",
			"tssc-dh":               "tssc-dh",
			"tssc-acs":              "tssc-acs",
			"tssc-acs-test":         "tssc-acs",
			"tssc-integrations":     "tssc",
		}))
		g.Expect(dependencySlice).To(o.Equal([]string{
			"tssc-openshift",
			"tssc-subscriptions",
			"tssc-infrastructure",
			"tssc-backing-services",
			"tssc-acs",
			"tssc-acs-test",
			"tssc-gitops",
			"tssc-tas",
			"tssc-pipelines",
			"tssc-tpa-realm",
			"tssc-tpa",
			"tssc-app-namespaces",
			"tssc-dh",
			"tssc-integrations",
		}))
	})
}
