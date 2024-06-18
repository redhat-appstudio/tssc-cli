package hooks

import (
	"bytes"
	"testing"

	"github.com/redhat-appstudio/rhtap-cli/pkg/chartfs"
	"github.com/redhat-appstudio/rhtap-cli/pkg/config"

	"helm.sh/helm/v3/pkg/chartutil"

	o "github.com/onsi/gomega"
)

func TestNewHooks(t *testing.T) {
	g := o.NewWithT(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	h := NewHooks(
		chartfs.NewChartFS("../.."),
		&config.Dependency{
			Chart:     "test/charts/testing",
			Namespace: "rhtap",
		},
		&stdout,
		&stderr,
	)
	vals := chartutil.Values{"key": "value"}

	t.Run("PreDeploy", func(t *testing.T) {
		err := h.PreDeploy(vals)
		g.Expect(err).To(o.Succeed())

		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
		g.Expect(stdout.String()).To(o.ContainSubstring("script runs before"))

		stdout.Reset()
		stderr.Reset()
	})

	t.Run("PostDeploy", func(t *testing.T) {
		err := h.PostDeploy(vals)
		g.Expect(err).To(o.Succeed())

		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
		g.Expect(stdout.String()).To(o.ContainSubstring("script runs after"))

		stdout.Reset()
		stderr.Reset()
	})
}
