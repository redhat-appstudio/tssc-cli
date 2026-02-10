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

	cfg, err := NewConfigFromFile(cfs, "config.yaml", "test-namespace")
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

		product, err := cfg.GetProduct("Konflux")
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

	t.Run("SetProducts", func(t *testing.T) {
		// Trusted Artifact Signer is product 1 and has a namespace in config
		err := cfg.Set("tssc.products.1.namespace", "tas-test-namespace")
		g.Expect(err).To(o.Succeed())

		// Trusted Profile Analyzer is product 4
		err = cfg.Set("tssc.products.4.enabled", false)
		g.Expect(err).To(o.Succeed())

		configString := cfg.String()
		g.Expect(string(configString)).To(o.ContainSubstring("namespace: tas-test-namespace"))
		g.Expect(string(configString)).To(o.ContainSubstring("enabled: false"))
	})

	t.Run("FlattenMap", func(t *testing.T) {
		data := map[string]interface{}{
			"key1": "value1",
			"key2": map[string]interface{}{
				"key3": "value2",
			},
		}
		expectedKeyValues := map[string]interface{}{
			"prefix.key1":      "value1",
			"prefix.key2.key3": "value2",
		}
		keyPaths, err := FlattenMap(data, "prefix")
		g.Expect(err).To(o.Succeed())
		g.Expect(keyPaths).To(o.HaveLen(len(expectedKeyValues)))
		for key, value := range keyPaths {
			g.Expect(value).To(o.Equal(expectedKeyValues[key]))
		}
	})

	t.Run("SetProduct", func(t *testing.T) {
		// Get an existing product
		product, err := cfg.GetProduct("Konflux")
		g.Expect(err).To(o.Succeed())
		g.Expect(product).NotTo(o.BeNil())

		// Modify existing fields and an existing property (UpdateMappingValue only replaces existing keys)
		product.Enabled = false
		newNamespace := "new-konflux-namespace"
		product.Namespace = &newNamespace
		product.Properties["manageSubscription"] = false

		// Call SetProduct
		err = cfg.SetProduct("Konflux", *product)
		g.Expect(err).To(o.Succeed())

		// Assert changes
		configString := cfg.String()
		g.Expect(configString).To(o.ContainSubstring("enabled: false"))
		g.Expect(configString).To(o.ContainSubstring("namespace: new-konflux-namespace"))
		g.Expect(configString).To(o.ContainSubstring("manageSubscription: false"))

		// Test non-existent product
		err = cfg.SetProduct("NonExistentProduct", Product{})
		g.Expect(err).NotTo(o.Succeed())
		g.Expect(err.Error()).To(o.ContainSubstring(
			"product \"NonExistentProduct\" not found"))
	})

	t.Run("EnableDisableProduct", func(t *testing.T) {
		productName := "Cert-Manager"

		// Disable the product
		config, err := cfg.EnableDisableProduct(productName, false)
		g.Expect(err).To(o.Succeed())

		// Verify the product is disabled
		spec, err := config.GetProduct(productName)
		g.Expect(err).To(o.Succeed())
		g.Expect(spec).NotTo(o.BeNil())
		g.Expect(spec.Enabled).To(o.BeFalse())

		// Enabled the product
		config, err = cfg.EnableDisableProduct(productName, true)
		g.Expect(err).To(o.Succeed())

		// Verify the product is enabled
		spec, err = config.GetProduct(productName)
		g.Expect(err).To(o.Succeed())
		g.Expect(spec).NotTo(o.BeNil())
		g.Expect(spec.Enabled).To(o.BeTrue())
	})
}
