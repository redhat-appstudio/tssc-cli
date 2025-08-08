package resolver

import (
	"testing"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/config"

	o "github.com/onsi/gomega"
)

func TestNewTopology(t *testing.T) {
	g := o.NewWithT(t)

	cfs, err := chartfs.NewChartFS("../../installer")
	g.Expect(err).To(o.Succeed())
	g.Expect(cfs).ToNot(o.BeNil())

	openShiftChart, err := cfs.GetChartFiles("charts/tssc-openshift")
	g.Expect(err).To(o.Succeed())
	openShiftDep := config.NewDependency(openShiftChart, "default")

	subscriptionsChart, err := cfs.GetChartFiles("charts/tssc-subscriptions")
	g.Expect(err).To(o.Succeed())
	subscriptionsDep := config.NewDependency(subscriptionsChart, "default")

	infrastructureChart, err := cfs.GetChartFiles("charts/tssc-infrastructure")
	g.Expect(err).To(o.Succeed())
	infrastructureDep := config.NewDependency(infrastructureChart, "default")

	backingServicesChart, err := cfs.GetChartFiles("charts/tssc-backing-services")
	g.Expect(err).To(o.Succeed())
	backingServicesDep := config.NewDependency(backingServicesChart, "default")

	topology := NewTopology()

	t.Run("Append", func(t *testing.T) {
		topology.Append(*backingServicesDep)
	})

	t.Run("PrependBefore", func(t *testing.T) {
		topology.PrependBefore(
			backingServicesChart.Name(),
			*openShiftDep,
			*infrastructureDep,
		)
	})

	t.Run("AppendAfter", func(t *testing.T) {
		topology.AppendAfter(openShiftChart.Name(), *subscriptionsDep)
	})

	t.Run("GetDependencies", func(t *testing.T) {
		deps := topology.GetDependencies()
		g.Expect(deps).ToNot(o.BeNil())
		names := []string{}
		for _, d := range deps {
			names = append(names, d.Chart.Name())
		}
		g.Expect(names).To(o.Equal([]string{
			"tssc-openshift",
			"tssc-subscriptions",
			"tssc-infrastructure",
			"tssc-backing-services",
		}))
	})
}
