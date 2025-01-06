package chartfs

import (
	"testing"

	o "github.com/onsi/gomega"
)

func TestNewChartFS(t *testing.T) {
	g := o.NewWithT(t)

	c, err := NewChartFS("../../installer")
	g.Expect(err).To(o.Succeed())
	g.Expect(c).ToNot(o.BeNil())

	t.Run("ReadFile", func(t *testing.T) {
		valuesTmplBytes, err := c.ReadFile("charts/values.yaml.tpl")
		g.Expect(err).To(o.Succeed())
		g.Expect(valuesTmplBytes).ToNot(o.BeEmpty())
	})

	t.Run("GetChartForDep", func(t *testing.T) {
		chart, err := c.GetChartForDep("charts/rhtap-openshift")
		g.Expect(err).To(o.Succeed())
		g.Expect(chart).ToNot(o.BeNil())
		g.Expect(chart.Name()).To(o.Equal("rhtap-openshift"))
		g.Expect(chart.Files).ToNot(o.BeEmpty())
		g.Expect(chart.Templates).ToNot(o.BeEmpty())

		// Asserting the chart templates are present, it should contain at least a
		// few files, plus the presence of the "NOTES.txt" common file.
		names := []string{}
		for _, tmpl := range chart.Templates {
			names = append(names, tmpl.Name)
		}
		g.Expect(len(names)).To(o.BeNumerically(">", 1))
		g.Expect(names).To(o.ContainElement("templates/NOTES.txt"))
	})
}
