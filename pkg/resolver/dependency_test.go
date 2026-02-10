package resolver

import (
	"testing"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"

	o "github.com/onsi/gomega"
)

func TestNewDependency(t *testing.T) {
	g := o.NewWithT(t)

	cfs, err := chartfs.NewChartFS("../../installer")
	g.Expect(err).To(o.Succeed())

	chartFiles, err := cfs.GetChartFiles("charts/tssc-konflux")
	g.Expect(err).To(o.Succeed())

	d := NewDependency(chartFiles)

	t.Run("Chart", func(t *testing.T) {
		g.Expect(d.Chart()).NotTo(o.BeNil())
	})

	t.Run("Name", func(t *testing.T) {
		g.Expect(d.Name()).To(o.Equal("tssc-konflux"))
	})

	t.Run("Namespace", func(t *testing.T) {
		g.Expect(d.Namespace()).To(o.Equal(""))
	})

	t.Run("DependsOn", func(t *testing.T) {
		dependsOn := d.DependsOn()
		g.Expect(len(dependsOn)).To(o.BeNumerically(">", 1))
		g.Expect(dependsOn[0]).To(o.Equal("tssc-openshift"))
	})

	t.Run("ProductName", func(t *testing.T) {
		g.Expect(d.ProductName()).To(o.Equal("Konflux"))
	})

	t.Run("UseProductNamespace", func(t *testing.T) {
		g.Expect(d.UseProductNamespace()).To(o.BeEmpty())
	})
}
