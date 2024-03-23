package config

import (
	"testing"

	o "github.com/onsi/gomega"
)

func TestNewConfigFromFile(t *testing.T) {
	g := o.NewWithT(t)

	cfg, err := NewConfigFromFile("../../config.yaml")
	g.Expect(err).To(o.Succeed())
	g.Expect(cfg).NotTo(o.BeNil())
	g.Expect(cfg.Installer).NotTo(o.BeNil())

	t.Run("Validate", func(t *testing.T) {
		err := cfg.Validate()
		g.Expect(err).To(o.Succeed())
	})
}
