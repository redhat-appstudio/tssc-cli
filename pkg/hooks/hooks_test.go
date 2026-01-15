package hooks

import (
	"bytes"
	"os"
	"testing"

	"github.com/redhat-appstudio/helmet/pkg/api"
	"github.com/redhat-appstudio/helmet/pkg/chartfs"
	"github.com/redhat-appstudio/helmet/pkg/resolver"

	o "github.com/onsi/gomega"
)

func TestNewHooks(t *testing.T) {
	g := o.NewWithT(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cfs := chartfs.New(os.DirFS("../../test"))

	chart, err := cfs.GetChartFiles("charts/testing")
	g.Expect(err).To(o.Succeed())

	appCtx := api.NewAppContext("tssc")
	h := NewHooks(
		resolver.NewDependencyWithNamespace(chart, appCtx.Namespace),
		&stdout,
		&stderr,
	)

	vals := map[string]interface{}{
		"key": map[string]interface{}{
			"nested": "value",
		},
	}

	t.Run("PreDeploy", func(t *testing.T) {
		err := h.PreDeploy(vals)
		g.Expect(err).To(o.Succeed())

		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
		// Asserting the environment variable is printed out by the hook script,
		// the variable is passed by the informed values.
		g.Expect(stdout.String()).
			To(o.ContainSubstring("# INSTALLER__KEY__NESTED='value'"))

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
