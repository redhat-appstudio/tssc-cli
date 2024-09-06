package hooks

import (
	"bytes"
	"testing"

	"github.com/redhat-appstudio/rhtap-cli/pkg/chartfs"
	"github.com/redhat-appstudio/rhtap-cli/pkg/config"

	o "github.com/onsi/gomega"
)

func TestNewHooks(t *testing.T) {
	g := o.NewWithT(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	h := NewHooks(
		chartfs.NewChartFS("../../test"),
		&config.Dependency{
			Chart:     "charts/testing",
			Namespace: "rhtap",
		},
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
