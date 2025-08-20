package resolver

import (
	"testing"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"

	o "github.com/onsi/gomega"
)

func TestNewTopology(t *testing.T) {
	g := o.NewWithT(t)

	cfs, err := chartfs.NewChartFS("../../installer")
	g.Expect(err).To(o.Succeed())
	g.Expect(cfs).ToNot(o.BeNil())

	ns := "default"
	openShiftChart, err := cfs.GetChartFiles("charts/tssc-openshift")
	g.Expect(err).To(o.Succeed())
	openShiftDep := NewDependencyWithNamespace(openShiftChart, ns)

	subscriptionsChart, err := cfs.GetChartFiles("charts/tssc-subscriptions")
	g.Expect(err).To(o.Succeed())
	subscriptionsDep := NewDependencyWithNamespace(subscriptionsChart, ns)

	infrastructureChart, err := cfs.GetChartFiles("charts/tssc-infrastructure")
	g.Expect(err).To(o.Succeed())
	infrastructureDep := NewDependencyWithNamespace(infrastructureChart, ns)

	iamChart, err := cfs.GetChartFiles("charts/tssc-iam")
	g.Expect(err).To(o.Succeed())
	iamDep := NewDependencyWithNamespace(iamChart, ns)

	topology := NewTopology()

	t.Run("Append", func(t *testing.T) {
		topology.Append(*iamDep)
	})

	t.Run("PrependBefore", func(t *testing.T) {
		topology.PrependBefore(
			iamDep.Name(),
			*openShiftDep,
			*infrastructureDep,
		)
	})

	t.Run("AppendAfter", func(t *testing.T) {
		topology.AppendAfter(openShiftChart.Name(), *subscriptionsDep)
	})

	t.Run("GetDependencies", func(t *testing.T) {
		deps := topology.Dependencies()
		g.Expect(deps).ToNot(o.BeNil())
		names := []string{}
		for _, d := range deps {
			names = append(names, d.Name())
		}
		g.Expect(names).To(o.Equal([]string{
			"tssc-openshift",
			"tssc-subscriptions",
			"tssc-infrastructure",
			"tssc-iam",
		}))
	})
}
