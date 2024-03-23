package k8s

import (
	"testing"

	"github.com/otaviof/rhtap-installer-cli/pkg/flags"

	o "github.com/onsi/gomega"
)

func TestNewKubernetes(t *testing.T) {
	g := o.NewWithT(t)

	k := NewKube(flags.NewFlags())

	t.Run("Build", func(t *testing.T) {
		restConfig, err := k.RESTClientGetter("default").ToRESTConfig()
		g.Expect(err).To(o.Succeed())
		g.Expect(restConfig).NotTo(o.BeNil())
	})
}
