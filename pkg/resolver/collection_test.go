package resolver

import (
	"testing"

	o "github.com/onsi/gomega"
	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"helm.sh/helm/v3/pkg/chart"
)

func TestNewCollection(t *testing.T) {
	g := o.NewWithT(t)

	cfs, err := chartfs.NewChartFS("../../installer")
	g.Expect(err).To(o.Succeed())

	charts, err := cfs.GetAllCharts()
	g.Expect(err).To(o.Succeed())

	c, err := NewCollection(charts)
	g.Expect(err).To(o.Succeed())
	g.Expect(c).NotTo(o.BeNil())

	var tsscTPAChart *chart.Chart
	t.Run("Get", func(t *testing.T) {
		name := "tssc-tpa"
		var err error
		tsscTPAChart, err = c.Get(name)
		g.Expect(err).To(o.Succeed())
		g.Expect(tsscTPAChart).NotTo(o.BeNil())
		g.Expect(tsscTPAChart.Name()).To(o.Equal(name))
	})

	t.Run("DependsOn", func(t *testing.T) {
		dependsOn := c.DependsOn(tsscTPAChart)
		g.Expect(len(dependsOn)).To(o.BeNumerically(">=", 1))
		g.Expect(dependsOn).To(o.Equal([]string{
			"tssc-openshift",
			"tssc-subscriptions",
			"tssc-infrastructure",
			"tssc-backing-services",
			"tssc-tpa-realm",
		}))
	})

	t.Run("ProductName", func(t *testing.T) {
		productName := c.ProductName(tsscTPAChart)
		g.Expect(productName).To(o.Equal("Trusted Profile Analyzer"))
	})
}
