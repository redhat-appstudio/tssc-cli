package config

import (
	"os"
	"testing"

	"github.com/redhat-appstudio/helmet/pkg/chartfs"

	o "github.com/onsi/gomega"
)

func TestNewConfigFromFile(t *testing.T) {
	g := o.NewWithT(t)

	cfs := chartfs.New(os.DirFS("../../installer"))

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
		// ACS is product 0
		err := cfg.Set("tssc.products.0.namespace", "acstest")
		g.Expect(err).To(o.Succeed())

		// TPA is product 4
		err = cfg.Set("tssc.products.4.enabled", false)
		g.Expect(err).To(o.Succeed())

		// DH is product 5
		dhData := map[string]any{
			"catalogURL":   "https://someIP.io",
			"authProvider": "gitlab",
		}
		err = cfg.Set("tssc.products.5.properties", dhData)
		g.Expect(err).To(o.Succeed())

		configString := cfg.String()
		g.Expect(string(configString)).To(o.ContainSubstring("namespace: acstest"))
		g.Expect(string(configString)).To(o.ContainSubstring("enabled: false"))
		g.Expect(string(configString)).To(o.ContainSubstring("catalogURL: https://someIP.io"))
		g.Expect(string(configString)).To(o.ContainSubstring("authProvider: gitlab"))
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
		product, err := cfg.GetProduct("Developer Hub")
		g.Expect(err).To(o.Succeed())
		g.Expect(product).NotTo(o.BeNil())

		// Modify it
		product.Enabled = false
		newNamespace := "new-dh-namespace"
		product.Namespace = &newNamespace
		product.Properties["catalogURL"] = "http://new.url/catalog.yaml"

		// Call SetProduct
		err = cfg.SetProduct("Developer Hub", *product)
		g.Expect(err).To(o.Succeed())

		// Assert changes
		configString := cfg.String()
		g.Expect(configString).To(o.ContainSubstring("enabled: false"))
		g.Expect(configString).To(o.ContainSubstring("namespace: new-dh-namespace"))
		g.Expect(configString).To(o.ContainSubstring(
			"catalogURL: http://new.url/catalog.yaml"))

		// Test non-existent product
		err = cfg.SetProduct("NonExistentProduct", Product{})
		g.Expect(err).NotTo(o.Succeed())
		g.Expect(err.Error()).To(o.ContainSubstring(
			"product \"NonExistentProduct\" not found"))
	})
}
