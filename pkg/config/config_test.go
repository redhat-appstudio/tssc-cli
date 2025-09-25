package config

import (
	"testing"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"

	o "github.com/onsi/gomega"
)

func TestNewConfigFromFile(t *testing.T) {
	g := o.NewWithT(t)

	cfs, err := chartfs.NewChartFS("../../installer")
	g.Expect(err).To(o.Succeed())

	cfg, err := NewConfigFromFile(cfs, "config.yaml")
	g.Expect(err).To(o.Succeed())
	g.Expect(cfg).NotTo(o.BeNil())
	g.Expect(cfg.Installer).NotTo(o.BeNil())

	t.Run("Validate", func(t *testing.T) {
		err := cfg.Validate()
		g.Expect(err).To(o.Succeed())
	})

	t.Run("GetEnabledProducts", func(t *testing.T) {
		products := cfg.GetEnabledProducts()
		g.Expect(products).NotTo(o.BeEmpty())
		g.Expect(len(products)).To(o.BeNumerically(">", 1))
	})

	t.Run("GetProduct", func(t *testing.T) {
		_, err := cfg.GetProduct("product1")
		g.Expect(err).NotTo(o.Succeed())

		product, err := cfg.GetProduct("Developer Hub")
		g.Expect(err).To(o.Succeed())
		g.Expect(product).NotTo(o.BeNil())
		g.Expect(product.GetNamespace()).NotTo(o.BeEmpty())
	})

	t.Run("MarshalYAML and UnmarshalYAML", func(t *testing.T) {
		payload, err := cfg.MarshalYAML()
		g.Expect(err).To(o.Succeed())
		g.Expect(string(payload)).To(o.ContainSubstring("tssc:"))

		err = cfg.UnmarshalYAML(payload)
		g.Expect(err).To(o.Succeed())
	})

	t.Run("DecodeNode", func(t *testing.T) {
		err := cfg.DecodeNode()
		g.Expect(err).To(o.Succeed())
		g.Expect(cfg.Installer).NotTo(o.BeNil())
	})

	t.Run("String", func(t *testing.T) {
		original, err := cfs.ReadFile("config.yaml")
		g.Expect(err).To(o.Succeed())

		configString := cfg.String()
		g.Expect(err).To(o.Succeed())
		g.Expect(string(configString)).To(o.ContainSubstring("tssc:"))

		// Asserting the original configuration looks like the marshaled one.
		g.Expect(string(original)).To(o.Equal(configString))
	})

	t.Run("SetNamespace", func(t *testing.T) {
		err := cfg.Set("tssc.namespace", "testnamespace")
		g.Expect(err).To(o.Succeed())
		configString := cfg.String()
		g.Expect(string(configString)).To(o.ContainSubstring("testnamespace"))
	})

	t.Run("SetSettings", func(t *testing.T) {
		data := map[string]interface{}{
			"crc": true,
			"ci": map[string]interface{}{
				"debug": true,
			},
		}
		err := cfg.Set("tssc.settings", data)
		g.Expect(err).To(o.Succeed())
		configString := cfg.String()
		g.Expect(string(configString)).To(o.ContainSubstring("crc: true"))
		g.Expect(string(configString)).To(o.ContainSubstring("debug: true"))
	})

	t.Run("ExtractKeyPathsAndValues", func(t *testing.T) {
		data := map[string]interface{}{
			"key1": "value1",
			"key2": map[string]interface{}{
				"key3": "value2",
			},
		}
		expectedPaths := []string{"prefix.key1", "prefix.key2.key3"}
		expectedValues := []interface{}{"value1", "value2"}
		paths, values := ExtractKeyPathsAndValues(data, "prefix")
		g.Expect(paths).To(o.HaveLen(len(expectedPaths)))
		for _, path := range expectedPaths {
			g.Expect(paths).To(o.ContainElement(path))
		}
		g.Expect(values).To(o.HaveLen(len(expectedValues)))
		for _, value := range expectedValues {
			g.Expect(values).To(o.ContainElement(value))
		}
	})
}
