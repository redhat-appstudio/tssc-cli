package engine

import (
	"testing"

	"github.com/redhat-appstudio/tssc-cli/pkg/chartfs"
	"github.com/redhat-appstudio/tssc-cli/pkg/config"

	o "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

// testYamlTmpl is a template to render a YAML payload based on the information
// available for the installer template engine.
const testYamlTmpl = `
---
root:
  namespace: {{ .Installer.Namespace }} 
  settings:
    key: value
  products:
{{- range $k, $v := .Installer.Products }}
  {{- $k | nindent 4 }}:
  {{- $v | toYaml | nindent 6 }}
{{- end }}
  catalogURL: {{ .Installer.Products.Developer_Hub.Properties.catalogURL }}
`

func TestEngine_Render(t *testing.T) {
	g := o.NewWithT(t)

	cfs, err := chartfs.NewChartFS("../../installer")
	g.Expect(err).To(o.Succeed())

	cfg, err := config.NewConfigFromFile(cfs, "config.yaml")
	g.Expect(err).To(o.Succeed())

	variables := NewVariables()
	err = variables.SetInstaller(&cfg.Installer)
	g.Expect(err).To(o.Succeed())

	t.Logf("Template: %s", testYamlTmpl)

	e := NewEngine(nil, testYamlTmpl)
	payload, err := e.Render(variables)
	g.Expect(err).To(o.Succeed())
	g.Expect(payload).NotTo(o.BeEmpty())

	t.Logf("Output: %s", payload)

	// Unmarshal the rendered payload to check the actual structure of the YAML
	// file created with the template engine.
	var outputMap map[string]interface{}
	err = yaml.Unmarshal([]byte(payload), &outputMap)
	g.Expect(err).To(o.Succeed())
	g.Expect(outputMap).NotTo(o.BeEmpty())

	g.Expect(outputMap).To(o.HaveKey("root"))
	g.Expect(outputMap["root"]).NotTo(o.BeNil())

	root := outputMap["root"].(map[string]interface{})
	g.Expect(root).To(o.HaveKey("namespace"))
	g.Expect(root["namespace"]).To(o.Equal(cfg.Installer.Namespace))

	g.Expect(root).To(o.HaveKey("settings"))
	g.Expect(root["settings"]).NotTo(o.BeNil())

	g.Expect(root).To(o.HaveKey("products"))
	g.Expect(root["products"]).NotTo(o.BeNil())

	g.Expect(root).To(o.HaveKey("catalogURL"))
	g.Expect(root["catalogURL"]).NotTo(o.BeNil())

	product, err := cfg.GetProduct(config.DeveloperHub)
	g.Expect(err).To(o.Succeed())
	g.Expect(root["catalogURL"]).To(o.Equal(product.Properties["catalogURL"]))
}
