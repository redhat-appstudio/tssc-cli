package resolver

import (
	"testing"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"

	o "github.com/onsi/gomega"
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
}
