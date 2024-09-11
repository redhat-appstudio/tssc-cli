package config

import (
	"log/slog"
	"testing"

	o "github.com/onsi/gomega"
)

func TestNewConfigFromFile(t *testing.T) {
	g := o.NewWithT(t)

	cfg, err := NewConfigFromFile("../../installer/config.yaml")
	g.Expect(err).To(o.Succeed())
	g.Expect(cfg).NotTo(o.BeNil())
	g.Expect(cfg.Installer).NotTo(o.BeNil())

	t.Run("Validate", func(t *testing.T) {
		err := cfg.Validate()
		g.Expect(err).To(o.Succeed())
	})

	t.Run("GetEnabledDependencies", func(t *testing.T) {
		deps := cfg.GetEnabledDependencies(slog.Default())
		g.Expect(deps).NotTo(o.BeEmpty())
		g.Expect(len(deps)).To(o.BeNumerically(">=", 1))
	})

	t.Run("GetFeature", func(t *testing.T) {
		_, err := cfg.GetFeature("feature1")
		g.Expect(err).NotTo(o.Succeed())

		feature, err := cfg.GetFeature(RedHatDeveloperHub)
		g.Expect(err).To(o.Succeed())
		g.Expect(feature).NotTo(o.BeNil())
		g.Expect(feature.GetNamespace()).NotTo(o.BeEmpty())
	})
}
